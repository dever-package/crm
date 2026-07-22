package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

const (
	leadDispatchQueueBatchSize = 500
	leadDispatchRetryBatchSize = 50
)

type leadDispatchRetrySummary struct {
	Queued    int `json:"queued"`
	Assigned  int `json:"assigned"`
	Remaining int `json:"remaining"`
}

func reservedLeadDispatchInstanceIDs(ctx context.Context, sourceDepartmentID uint64) map[uint64]bool {
	result := map[uint64]bool{}
	if sourceDepartmentID == 0 {
		return result
	}
	for _, handoff := range crmmodel.NewLeadDispatchHandoffModel().Select(ctx, map[string]any{
		"source_department_id": sourceDepartmentID,
		"status": []string{
			crmmodel.LeadDispatchHandoffPending,
			crmmodel.LeadDispatchHandoffProcessing,
		},
	}) {
		if handoff != nil {
			result[handoff.WorkflowInstanceID] = true
		}
	}
	return result
}

func retryLeadDispatchWorkflow(ctx context.Context, workflowID uint64) (leadDispatchRetrySummary, error) {
	summary := leadDispatchRetrySummary{}
	if !leadDispatchRouteEnabled(ctx, workflowID) {
		return summary, nil
	}
	workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
		"id":           workflowID,
		"subject_type": crmmodel.WorkflowSubjectLead,
		"status":       crmmodel.StatusEnabled,
	})
	scope, err := resolveLeadDispatchScope(ctx, workflow)
	if err != nil {
		return summary, err
	}
	summary.Queued, err = queueExistingLeadDispatchHandoffs(ctx, scope)
	assigned, retryErr := retryPendingLeadDispatchHandoffs(ctx, workflowID, leadDispatchRetryBatchSize)
	summary.Assigned = assigned
	summary.Remaining = pendingLeadDispatchCount(ctx, workflowID)
	return summary, errors.Join(err, retryErr)
}

func queueExistingLeadDispatchHandoffs(ctx context.Context, scope *leadDispatchScope) (int, error) {
	if scope == nil || scope.Workflow == nil || scope.SourceStage == nil || scope.SourceDepartment == nil {
		return 0, fmt.Errorf("线索派单范围不完整")
	}
	instances := crmmodel.NewWorkflowInstanceModel().Select(ctx, map[string]any{
		"workflow_id":         scope.Workflow.ID,
		"stage_id":            scope.SourceStage.ID,
		"owner_department_id": scope.SourceDepartment.ID,
		"status":              crmmodel.ProgressStatusActive,
	}, map[string]any{"order": "id asc"})
	if len(instances) == 0 {
		return 0, nil
	}
	instanceIDs := make([]uint64, 0, len(instances))
	leadIDs := make([]uint64, 0, len(instances))
	for _, instance := range instances {
		if instance == nil || instance.LeadID == 0 {
			continue
		}
		instanceIDs = append(instanceIDs, instance.ID)
		leadIDs = append(leadIDs, instance.LeadID)
	}
	if len(instanceIDs) == 0 {
		return 0, nil
	}
	existingInstanceIDs := map[uint64]bool{}
	for _, handoff := range crmmodel.NewLeadDispatchHandoffModel().Select(ctx, map[string]any{
		"workflow_instance_id": instanceIDs,
	}) {
		if handoff != nil {
			existingInstanceIDs[handoff.WorkflowInstanceID] = true
		}
	}
	pendingLeadIDs := map[uint64]bool{}
	for _, lead := range crmmodel.NewLeadModel().Select(ctx, map[string]any{
		"id":     leadIDs,
		"status": crmmodel.LeadStatusPending,
	}) {
		if lead != nil {
			pendingLeadIDs[lead.ID] = true
		}
	}
	candidateIDs := make([]uint64, 0, leadDispatchQueueBatchSize)
	for _, instance := range instances {
		if instance == nil || existingInstanceIDs[instance.ID] || !pendingLeadIDs[instance.LeadID] {
			continue
		}
		candidateIDs = append(candidateIDs, instance.ID)
		if len(candidateIDs) >= leadDispatchQueueBatchSize {
			break
		}
	}

	queued := 0
	var queueErr error
	for _, instanceID := range candidateIDs {
		created, err := queueExistingLeadDispatchInstance(ctx, scope, instanceID)
		if errors.Is(err, errPendingDispatchChanged) {
			continue
		}
		if err != nil {
			if queueErr == nil {
				queueErr = err
			}
			continue
		}
		if created {
			queued++
		}
	}
	return queued, queueErr
}

func queueExistingLeadDispatchInstance(ctx context.Context, scope *leadDispatchScope, instanceID uint64) (bool, error) {
	queued := false
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		if !leadDispatchRouteEnabled(txCtx, scope.Workflow.ID) {
			return nil
		}
		instance := crmmodel.NewWorkflowInstanceModel().Find(txCtx, map[string]any{
			"id":                  instanceID,
			"workflow_id":         scope.Workflow.ID,
			"stage_id":            scope.SourceStage.ID,
			"owner_department_id": scope.SourceDepartment.ID,
			"status":              crmmodel.ProgressStatusActive,
		})
		if instance == nil || instance.LeadID == 0 {
			return nil
		}
		lead := crmmodel.NewLeadModel().Find(txCtx, map[string]any{
			"id":     instance.LeadID,
			"status": crmmodel.LeadStatusPending,
		})
		if lead == nil {
			return nil
		}
		if crmmodel.NewLeadDispatchHandoffModel().Find(txCtx, map[string]any{
			"workflow_instance_id": instance.ID,
		}) != nil {
			return nil
		}
		if err := deferExistingLeadWorkflowForDispatch(txCtx, lead, instance, scope); err != nil {
			return err
		}
		if _, err := createLeadDispatchHandoff(txCtx, lead, instance, scope); err != nil {
			return err
		}
		queued = true
		return nil
	})
	return queued, err
}

func deferExistingLeadWorkflowForDispatch(
	ctx context.Context,
	lead *crmmodel.Lead,
	instance *crmmodel.WorkflowInstance,
	scope *leadDispatchScope,
) error {
	if lead == nil || instance == nil || scope == nil || scope.SourceDepartment == nil {
		return fmt.Errorf("存量线索派单上下文不完整")
	}
	now := time.Now()
	previousOwnerStaffID := instance.OwnerStaffID
	if crmmodel.NewWorkflowInstanceModel().Update(ctx, map[string]any{
		"id":                  instance.ID,
		"workflow_id":         instance.WorkflowID,
		"stage_id":            instance.StageID,
		"owner_department_id": scope.SourceDepartment.ID,
		"owner_staff_id":      previousOwnerStaffID,
		"status":              crmmodel.ProgressStatusActive,
	}, map[string]any{
		"owner_staff_id": uint64(0),
		"updated_at":     now,
	}) == 0 {
		return errPendingDispatchChanged
	}

	todos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"stage_id":             instance.StageID,
		"status":               crmmodel.WorkTodoStatusPending,
	}, map[string]any{"order": "id asc"})
	canceledTodoIDs := workTodoIDs(todos)
	if len(canceledTodoIDs) > 0 {
		updated := crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
			"id":     canceledTodoIDs,
			"status": crmmodel.WorkTodoStatusPending,
		}, map[string]any{
			"status":     crmmodel.WorkTodoStatusCanceled,
			"result":     "自动流转已开启，转入待派单",
			"updated_at": now,
		})
		if updated != int64(len(canceledTodoIDs)) {
			return errPendingDispatchChanged
		}
	}
	if crmmodel.NewLeadModel().Update(ctx, map[string]any{
		"id":     lead.ID,
		"status": crmmodel.LeadStatusPending,
	}, map[string]any{
		"owner_department_id": scope.SourceDepartment.ID,
		"owner_staff_id":      uint64(0),
		"updated_at":          now,
	}) == 0 {
		return errPendingDispatchChanged
	}
	instance.OwnerStaffID = 0
	instance.UpdatedAt = now
	lead.OwnerDepartmentID = scope.SourceDepartment.ID
	lead.OwnerStaffID = 0
	lead.UpdatedAt = now
	if recordWorkStageChange(ctx, nil, instance, workStageChange{
		FromWorkflowID: instance.WorkflowID,
		FromStageID:    instance.StageID,
		ToWorkflowID:   instance.WorkflowID,
		ToStageID:      instance.StageID,
		ResultValue:    "pending_dispatch",
		Title:          "进入待派单",
		Content:        "自动流转已开启，等待接单",
		Snapshot: map[string]any{
			"previous_owner_staff_id": previousOwnerStaffID,
			"owner_department_id":     scope.SourceDepartment.ID,
			"owner_staff_id":          uint64(0),
			"canceled_todo_ids":       canceledTodoIDs,
		},
	}) == 0 {
		return fmt.Errorf("存量线索待派单记录创建失败")
	}
	return nil
}

func retryPendingLeadDispatchHandoffs(ctx context.Context, workflowID uint64, limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	handoffs := crmmodel.NewLeadDispatchHandoffModel().Select(ctx, map[string]any{
		"source_workflow_id": workflowID,
		"status":             crmmodel.LeadDispatchHandoffPending,
	}, map[string]any{"order": "created_at asc,id asc"})
	assigned := 0
	processed := 0
	var retryErr error
	for _, handoff := range handoffs {
		if handoff == nil || processed >= limit {
			break
		}
		processed++
		wasAssigned := false
		noAssignee := false
		err := orm.Transaction(ctx, func(txCtx context.Context) error {
			var err error
			wasAssigned, noAssignee, err = attemptAutomaticLeadDispatch(txCtx, handoff.ID)
			return err
		})
		if errors.Is(err, errPendingDispatchChanged) {
			continue
		}
		if err != nil {
			if retryErr == nil {
				retryErr = err
			}
			continue
		}
		if noAssignee {
			break
		}
		if wasAssigned {
			assigned++
		}
	}
	return assigned, retryErr
}

func attemptAutomaticLeadDispatch(ctx context.Context, handoffID uint64) (bool, bool, error) {
	handoff := crmmodel.NewLeadDispatchHandoffModel().Find(ctx, map[string]any{
		"id":     handoffID,
		"status": crmmodel.LeadDispatchHandoffPending,
	})
	if handoff == nil {
		return false, false, errPendingDispatchChanged
	}
	if !leadDispatchRouteEnabled(ctx, handoff.SourceWorkflowID) {
		return false, true, nil
	}
	assignee, err := selectConfiguredDepartmentAssignee(ctx, handoff.TargetDepartmentID)
	if err != nil {
		return false, false, err
	}
	if assignee == nil {
		return false, true, nil
	}
	if _, err := executeLeadDispatchHandoff(ctx, nil, handoff.ID, assignee.ID, crmmodel.DispatchTypeAuto); err != nil {
		return false, false, err
	}
	return true, false, nil
}
