package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

func CompleteWorkflowStage(ctx context.Context, staff *WorkStaffSession, instanceID, nextOwnerStaffID uint64) (*crmmodel.WorkflowInstance, error) {
	var completed *crmmodel.WorkflowInstance
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		instance, err := activeWorkflowInstance(txCtx, instanceID)
		if err != nil {
			return err
		}
		if !canCompleteWorkflowStage(staff, instance) {
			return fmt.Errorf("只有当前负责人或流程调度员可以完成阶段")
		}
		completed, err = completeWorkflowStage(txCtx, staff, instance, nextOwnerStaffID)
		return err
	})
	return completed, err
}

func completeWorkflowStage(
	ctx context.Context,
	staff *WorkStaffSession,
	instance *crmmodel.WorkflowInstance,
	nextOwnerStaffID uint64,
) (*crmmodel.WorkflowInstance, error) {
	if instance == nil || instance.Status != crmmodel.ProgressStatusActive {
		return nil, fmt.Errorf("流程实例已结束")
	}
	if pendingRequiredTodoCount(ctx, instance) > 0 {
		return nil, fmt.Errorf("必做任务尚未全部完成")
	}

	workflow, stage, err := nextWorkflowStage(ctx, instance)
	if err != nil {
		return nil, err
	}
	cancelPendingOptionalTodos(ctx, instance)
	if workflow == nil || stage == nil {
		if err := completeWorkflowInstance(ctx, staff, instance); err != nil {
			return nil, err
		}
		return instance, nil
	}

	fromWorkflowID := instance.WorkflowID
	fromStageID := instance.StageID
	owner, err := enterWorkflowStage(ctx, instance, workflow, stage, nextOwnerStaffID)
	if err != nil {
		return nil, err
	}
	ownerName := "待派单"
	ownerStaffID := uint64(0)
	if owner != nil {
		ownerName = owner.Name
		ownerStaffID = owner.ID
	}
	if recordWorkStageChange(ctx, staff, instance, workStageChange{
		FromWorkflowID: fromWorkflowID,
		FromStageID:    fromStageID,
		ToWorkflowID:   workflow.ID,
		ToStageID:      stage.ID,
		ResultValue:    "entered",
		Title:          "进入阶段：" + stage.Name,
		Content:        "负责人：" + ownerName,
		Snapshot: map[string]any{
			"owner_department_id": stage.OwnerDepartmentID,
			"owner_staff_id":      ownerStaffID,
		},
	}) == 0 {
		return nil, fmt.Errorf("阶段流转记录创建失败")
	}
	if owner != nil {
		if err := createStageTodos(ctx, instance, stage); err != nil {
			return nil, err
		}
	}
	return instance, nil
}

func TerminateWorkflowInstance(ctx context.Context, staff *WorkStaffSession, instanceID uint64, reason string) (*crmmodel.WorkflowInstance, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("请填写终止原因")
	}
	var terminated *crmmodel.WorkflowInstance
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		instance, err := activeWorkflowInstance(txCtx, instanceID)
		if err != nil {
			return err
		}
		if staff == nil || staff.ID == 0 || instance.OwnerStaffID != staff.ID {
			return fmt.Errorf("只有当前负责人可以终止流程")
		}
		if err := terminateActiveWorkflowInstance(txCtx, staff, instance, reason); err != nil {
			return err
		}
		terminated = instance
		return nil
	})
	return terminated, err
}

func terminateActiveWorkflowInstance(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance, reason string) error {
	if instance == nil || instance.Status != crmmodel.ProgressStatusActive {
		return fmt.Errorf("流程实例已结束")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Errorf("请填写终止原因")
	}
	now := time.Now()
	crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"status":               crmmodel.WorkTodoStatusPending,
	}, map[string]any{
		"status":     crmmodel.WorkTodoStatusCanceled,
		"result":     "流程已终止：" + reason,
		"updated_at": now,
	})
	if crmmodel.NewWorkflowInstanceModel().Update(ctx, map[string]any{
		"id":     instance.ID,
		"status": crmmodel.ProgressStatusActive,
	}, map[string]any{
		"status":            crmmodel.ProgressStatusTerminated,
		"terminated_at":     now,
		"terminated_reason": reason,
		"updated_at":        now,
	}) == 0 {
		return fmt.Errorf("流程已变化，请刷新后重试")
	}
	instance.Status = crmmodel.ProgressStatusTerminated
	instance.TerminatedAt = &now
	instance.TerminatedReason = reason
	instance.UpdatedAt = now
	if err := LoseCustomerProductForInstance(ctx, instance); err != nil {
		return err
	}
	if recordWorkStageChange(ctx, staff, instance, workStageChange{
		FromWorkflowID: instance.WorkflowID,
		FromStageID:    instance.StageID,
		ResultValue:    "terminated",
		Title:          "流程已终止",
		Content:        reason,
		Snapshot:       map[string]any{"reason": reason},
	}) == 0 {
		return fmt.Errorf("流程终止记录创建失败")
	}
	return nil
}

func workflowInstanceByID(ctx context.Context, instanceID uint64) (*crmmodel.WorkflowInstance, error) {
	if instanceID == 0 {
		return nil, fmt.Errorf("流程实例不能为空")
	}
	instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": instanceID})
	if instance == nil {
		return nil, fmt.Errorf("流程实例不存在")
	}
	return instance, nil
}

func activeWorkflowInstance(ctx context.Context, instanceID uint64) (*crmmodel.WorkflowInstance, error) {
	instance, err := workflowInstanceByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if instance.Status != crmmodel.ProgressStatusActive {
		return nil, fmt.Errorf("流程实例已结束")
	}
	return instance, nil
}

func canCompleteWorkflowStage(staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) bool {
	if staff == nil || staff.ID == 0 || instance == nil {
		return false
	}
	return instance.OwnerStaffID == staff.ID || staff.CanDispatch
}

func pendingRequiredTodoCount(ctx context.Context, instance *crmmodel.WorkflowInstance) int64 {
	if instance == nil {
		return 0
	}
	return crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"stage_id":             instance.StageID,
		"required":             true,
		"status":               crmmodel.WorkTodoStatusPending,
	})
}

func cancelPendingOptionalTodos(ctx context.Context, instance *crmmodel.WorkflowInstance) {
	if instance == nil {
		return
	}
	crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"stage_id":             instance.StageID,
		"required":             false,
		"status":               crmmodel.WorkTodoStatusPending,
	}, map[string]any{
		"status":     crmmodel.WorkTodoStatusCanceled,
		"result":     "阶段已完成",
		"updated_at": time.Now(),
	})
}

func nextWorkflowStage(ctx context.Context, instance *crmmodel.WorkflowInstance) (*crmmodel.Workflow, *crmmodel.Stage, error) {
	if instance == nil {
		return nil, nil, fmt.Errorf("流程实例不存在")
	}
	workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
		"id":     instance.WorkflowID,
		"status": crmmodel.StatusEnabled,
	})
	if workflow == nil {
		return nil, nil, fmt.Errorf("当前流程不存在或已停用")
	}
	stage := nextEnabledStage(ctx, workflow.ID, instance.StageID)
	if stage == nil {
		return nil, nil, nil
	}
	return workflow, stage, nil
}

type workflowAssignmentTarget struct {
	Workflow    *crmmodel.Workflow
	Stage       *crmmodel.Stage
	CrossObject bool
}

func nextWorkflowAssignmentTarget(ctx context.Context, instance *crmmodel.WorkflowInstance) (*workflowAssignmentTarget, error) {
	workflow, stage, err := nextWorkflowStage(ctx, instance)
	if err != nil {
		return nil, err
	}
	if stage != nil || instance.LeadID == 0 {
		return &workflowAssignmentTarget{Workflow: workflow, Stage: stage}, nil
	}
	nextWorkflow, nextStage := defaultEntryWorkflowStage(ctx, crmmodel.WorkflowSubjectCustomerAsset)
	if nextWorkflow == nil || nextStage == nil {
		return nil, ErrNoAvailableWorkflow
	}
	return &workflowAssignmentTarget{
		Workflow:    nextWorkflow,
		Stage:       nextStage,
		CrossObject: true,
	}, nil
}

func completeWorkflowInstance(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) error {
	now := time.Now()
	if crmmodel.NewWorkflowInstanceModel().Update(ctx, map[string]any{
		"id":          instance.ID,
		"workflow_id": instance.WorkflowID,
		"stage_id":    instance.StageID,
		"status":      crmmodel.ProgressStatusActive,
	}, map[string]any{
		"status":       crmmodel.ProgressStatusCompleted,
		"completed_at": now,
		"updated_at":   now,
	}) == 0 {
		return fmt.Errorf("流程已变化，请刷新后重试")
	}
	instance.Status = crmmodel.ProgressStatusCompleted
	instance.CompletedAt = &now
	instance.UpdatedAt = now
	if recordWorkStageChange(ctx, staff, instance, workStageChange{
		FromWorkflowID: instance.WorkflowID,
		FromStageID:    instance.StageID,
		ResultValue:    "completed",
		Title:          "流程已完成",
	}) == 0 {
		return fmt.Errorf("流程完成记录创建失败")
	}
	if instance.LeadID > 0 {
		return nil
	}
	if instance.CustomerProductID == 0 {
		return StartConfirmedProductWorkflows(ctx, instance)
	}
	return CompleteCustomerProductForInstance(ctx, instance)
}
