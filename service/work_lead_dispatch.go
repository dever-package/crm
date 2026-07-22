package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

type leadDispatchScope struct {
	Workflow         *crmmodel.Workflow
	SourceStage      *crmmodel.Stage
	SourceDepartment *crmmodel.Department
	TargetWorkflow   *crmmodel.Workflow
	TargetStage      *crmmodel.Stage
	TargetDepartment *crmmodel.Department
}

func resolveLeadDispatchScope(ctx context.Context, workflow *crmmodel.Workflow) (*leadDispatchScope, error) {
	if workflow == nil || workflow.SubjectType != crmmodel.WorkflowSubjectLead || workflow.Status != crmmodel.StatusEnabled {
		return nil, fmt.Errorf("线索流程不存在或已停用")
	}
	sourceStage := firstEnabledStage(ctx, workflow.ID)
	if sourceStage == nil {
		return nil, fmt.Errorf("线索流程没有已启用阶段")
	}
	sourceDepartment := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{
		"id":     sourceStage.OwnerDepartmentID,
		"status": crmmodel.StatusEnabled,
	})
	if sourceDepartment == nil {
		return nil, fmt.Errorf("线索入口阶段负责部门不存在或已停用")
	}
	target, err := nextWorkflowAssignmentTarget(ctx, &crmmodel.WorkflowInstance{
		LeadID:     1,
		WorkflowID: workflow.ID,
		StageID:    sourceStage.ID,
	})
	if err != nil {
		return nil, err
	}
	if target == nil || target.Workflow == nil || target.Stage == nil {
		return nil, fmt.Errorf("线索流程没有可派单的下一阶段")
	}
	targetDepartment := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{
		"id":     target.Stage.OwnerDepartmentID,
		"status": crmmodel.StatusEnabled,
	})
	if targetDepartment == nil {
		return nil, fmt.Errorf("下一阶段负责部门不存在或已停用")
	}
	return &leadDispatchScope{
		Workflow:         workflow,
		SourceStage:      sourceStage,
		SourceDepartment: sourceDepartment,
		TargetWorkflow:   target.Workflow,
		TargetStage:      target.Stage,
		TargetDepartment: targetDepartment,
	}, nil
}

func manageableLeadDispatchScopes(ctx context.Context, staff *WorkStaffSession) []*leadDispatchScope {
	if staff == nil || staff.ID == 0 {
		return nil
	}
	workflows := crmmodel.NewWorkflowModel().Select(ctx, map[string]any{
		"subject_type": crmmodel.WorkflowSubjectLead,
		"status":       crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
	result := make([]*leadDispatchScope, 0, len(workflows))
	for _, workflow := range workflows {
		scope, err := resolveLeadDispatchScope(ctx, workflow)
		if err != nil || !canManageLeadDispatchScope(ctx, staff, scope) {
			continue
		}
		result = append(result, scope)
	}
	return result
}

func manageableLeadDispatchScope(
	ctx context.Context,
	staff *WorkStaffSession,
	workflowID uint64,
) (*leadDispatchScope, []*leadDispatchScope, error) {
	scopes := manageableLeadDispatchScopes(ctx, staff)
	if len(scopes) == 0 {
		return nil, nil, fmt.Errorf("当前账号没有线索派单管理权限")
	}
	if workflowID == 0 {
		return scopes[0], scopes, nil
	}
	for _, scope := range scopes {
		if scope != nil && scope.Workflow.ID == workflowID {
			return scope, scopes, nil
		}
	}
	return nil, nil, fmt.Errorf("无权管理该线索流程的派单")
}

func canManageLeadDispatchScope(ctx context.Context, staff *WorkStaffSession, scope *leadDispatchScope) bool {
	if staff == nil || staff.ID == 0 || scope == nil || scope.SourceDepartment == nil {
		return false
	}
	return staff.CanDispatch || isDepartmentLeader(ctx, staff, scope.SourceDepartment.ID)
}

func leadDispatchRouteEnabled(ctx context.Context, workflowID uint64) bool {
	return workflowID > 0 && crmmodel.NewLeadDispatchRouteModel().Find(ctx, map[string]any{
		"workflow_id": workflowID,
		"status":      crmmodel.StatusEnabled,
	}) != nil
}

func startNewLeadDispatch(
	ctx context.Context,
	workflow *crmmodel.Workflow,
	lead *crmmodel.Lead,
) (*crmmodel.WorkflowInstance, bool, error) {
	if workflow == nil || lead == nil || !leadDispatchRouteEnabled(ctx, workflow.ID) {
		return nil, false, nil
	}
	scope, err := resolveLeadDispatchScope(ctx, workflow)
	if err != nil {
		return nil, true, err
	}
	instance, err := startDeferredWorkflowInstance(ctx, leadWorkflowSubject(lead.ID), workflow.ID)
	if err != nil {
		return nil, true, err
	}
	handoff, err := createLeadDispatchHandoff(ctx, lead, instance, scope)
	if err != nil {
		return nil, true, err
	}
	if _, _, err := attemptAutomaticLeadDispatch(ctx, handoff.ID); err != nil {
		return nil, true, err
	}
	return instance, true, nil
}

func createLeadDispatchHandoff(
	ctx context.Context,
	lead *crmmodel.Lead,
	instance *crmmodel.WorkflowInstance,
	scope *leadDispatchScope,
) (*crmmodel.LeadDispatchHandoff, error) {
	if lead == nil || instance == nil || scope == nil || scope.TargetStage == nil {
		return nil, fmt.Errorf("线索派单上下文不完整")
	}
	model := crmmodel.NewLeadDispatchHandoffModel()
	if existing := model.Find(ctx, map[string]any{"workflow_instance_id": instance.ID}); existing != nil {
		return existing, nil
	}
	now := time.Now()
	id := uint64(model.Insert(ctx, map[string]any{
		"lead_id":              lead.ID,
		"workflow_instance_id": instance.ID,
		"source_workflow_id":   instance.WorkflowID,
		"source_stage_id":      instance.StageID,
		"source_department_id": instance.OwnerDepartmentID,
		"target_workflow_id":   scope.TargetWorkflow.ID,
		"target_stage_id":      scope.TargetStage.ID,
		"target_department_id": scope.TargetDepartment.ID,
		"assignee_staff_id":    uint64(0),
		"dispatch_type":        "",
		"operator_staff_id":    uint64(0),
		"status":               crmmodel.LeadDispatchHandoffPending,
		"created_at":           now,
		"updated_at":           now,
	}))
	if id == 0 {
		return nil, fmt.Errorf("线索待派单记录创建失败")
	}
	handoff := model.Find(ctx, map[string]any{"id": id})
	if handoff == nil {
		return nil, fmt.Errorf("线索待派单记录创建后无法读取")
	}
	return handoff, nil
}

func AssignLeadDispatchHandoffs(
	ctx context.Context,
	staff *WorkStaffSession,
	handoffIDs []uint64,
	assigneeStaffID uint64,
) (int, error) {
	if staff == nil || staff.ID == 0 {
		return 0, fmt.Errorf("请先登录")
	}
	if len(handoffIDs) == 0 {
		return 0, fmt.Errorf("请至少选择一条待派单线索")
	}
	if len(handoffIDs) > 500 {
		return 0, fmt.Errorf("单次最多派单500条线索")
	}
	uniqueIDs := make([]uint64, 0, len(handoffIDs))
	seen := make(map[uint64]bool, len(handoffIDs))
	for _, handoffID := range handoffIDs {
		if handoffID == 0 || seen[handoffID] {
			continue
		}
		seen[handoffID] = true
		uniqueIDs = append(uniqueIDs, handoffID)
	}
	if len(uniqueIDs) == 0 || assigneeStaffID == 0 {
		return 0, fmt.Errorf("请选择待派单线索和接单人员")
	}
	assigned := 0
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		for _, handoffID := range uniqueIDs {
			handoff := crmmodel.NewLeadDispatchHandoffModel().Find(txCtx, map[string]any{
				"id":     handoffID,
				"status": crmmodel.LeadDispatchHandoffPending,
			})
			if handoff == nil {
				return fmt.Errorf("待派单线索已变化，请刷新后重试")
			}
			if !canManageLeadDispatchHandoff(txCtx, staff, handoff) {
				return fmt.Errorf("只有来源部门负责人或流程调度员可以处理待派单")
			}
			if _, err := executeLeadDispatchHandoff(txCtx, staff, handoff.ID, assigneeStaffID, crmmodel.DispatchTypeManual); err != nil {
				return err
			}
			assigned++
		}
		return nil
	})
	return assigned, err
}

func canManageLeadDispatchHandoff(ctx context.Context, staff *WorkStaffSession, handoff *crmmodel.LeadDispatchHandoff) bool {
	if staff == nil || staff.ID == 0 || handoff == nil {
		return false
	}
	return staff.CanDispatch || isDepartmentLeader(ctx, staff, handoff.SourceDepartmentID)
}

func executeLeadDispatchHandoff(
	ctx context.Context,
	operator *WorkStaffSession,
	handoffID uint64,
	assigneeStaffID uint64,
	dispatchType string,
) (*crmmodel.LeadDispatchHandoff, error) {
	model := crmmodel.NewLeadDispatchHandoffModel()
	handoff := model.Find(ctx, map[string]any{
		"id":     handoffID,
		"status": crmmodel.LeadDispatchHandoffPending,
	})
	if handoff == nil {
		return nil, fmt.Errorf("待派单线索已变化，请刷新后重试")
	}
	if dispatchType != crmmodel.DispatchTypeAuto && dispatchType != crmmodel.DispatchTypeManual {
		return nil, fmt.Errorf("派单方式无效")
	}
	instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
		"id":          handoff.WorkflowInstanceID,
		"lead_id":     handoff.LeadID,
		"workflow_id": handoff.SourceWorkflowID,
		"stage_id":    handoff.SourceStageID,
		"status":      crmmodel.ProgressStatusActive,
	})
	lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{
		"id":     handoff.LeadID,
		"status": crmmodel.LeadStatusPending,
	})
	if instance == nil || lead == nil {
		return nil, fmt.Errorf("待派单线索或流程已变化，请刷新后重试")
	}
	target, err := nextWorkflowAssignmentTarget(ctx, instance)
	if err != nil {
		return nil, err
	}
	if target == nil || target.Workflow == nil || target.Stage == nil ||
		target.Stage.OwnerDepartmentID == 0 {
		return nil, fmt.Errorf("流程没有可用的下一阶段")
	}
	assignee := configuredDispatchPoolStaff(ctx, target.Stage.OwnerDepartmentID, assigneeStaffID)
	if assignee == nil {
		return nil, fmt.Errorf("所选人员不是当前派单配置中的启用接单人员")
	}
	now := time.Now()
	if model.Update(ctx, map[string]any{
		"id":     handoff.ID,
		"status": crmmodel.LeadDispatchHandoffPending,
	}, map[string]any{
		"target_workflow_id":   target.Workflow.ID,
		"target_stage_id":      target.Stage.ID,
		"target_department_id": target.Stage.OwnerDepartmentID,
		"status":               crmmodel.LeadDispatchHandoffProcessing,
		"updated_at":           now,
	}) == 0 {
		return nil, fmt.Errorf("待派单线索已变化，请刷新后重试")
	}

	targetInstanceID := instance.ID
	if target.CrossObject {
		conversion, err := convertWorkLeadForDispatch(
			ctx,
			operator,
			lead,
			instance,
			target.Workflow,
			assignee.ID,
		)
		if err != nil {
			return nil, err
		}
		customerID := inputUint64(conversion["customer_id"])
		assetID := inputUint64(conversion["asset_id"])
		targetInstance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
			"customer_id":         customerID,
			"asset_id":            assetID,
			"customer_product_id": uint64(0),
			"workflow_id":         target.Workflow.ID,
		}, map[string]any{"order": "id desc"})
		if targetInstance == nil {
			return nil, fmt.Errorf("接收流程启动失败")
		}
		targetInstanceID = targetInstance.ID
	} else {
		if _, err := completeWorkflowStage(ctx, operator, instance, assignee.ID); err != nil {
			return nil, err
		}
	}

	reference := workflowDispatchReference{
		Source:             crmmodel.DispatchSourceLead,
		LeadID:             handoff.LeadID,
		WorkflowInstanceID: targetInstanceID,
	}
	if operator != nil {
		reference.OperatorStaffID = operator.ID
	}
	if dispatchType == crmmodel.DispatchTypeAuto {
		err = recordAutomaticDispatch(ctx, target.Stage.OwnerDepartmentID, assignee.ID, reference)
	} else {
		err = recordManualDispatch(ctx, target.Stage.OwnerDepartmentID, assignee.ID, reference)
	}
	if err != nil {
		return nil, err
	}
	completedAt := time.Now()
	if model.Update(ctx, map[string]any{
		"id":     handoff.ID,
		"status": crmmodel.LeadDispatchHandoffProcessing,
	}, map[string]any{
		"assignee_staff_id": assignee.ID,
		"dispatch_type":     dispatchType,
		"operator_staff_id": reference.OperatorStaffID,
		"status":            crmmodel.LeadDispatchHandoffCompleted,
		"completed_at":      completedAt,
		"updated_at":        completedAt,
	}) == 0 {
		return nil, fmt.Errorf("待派单完成状态保存失败")
	}
	return model.Find(ctx, map[string]any{"id": handoff.ID}), nil
}

func configuredDispatchPoolStaff(ctx context.Context, departmentID, staffID uint64) *crmmodel.Staff {
	if departmentID == 0 || staffID == 0 {
		return nil
	}
	setting := crmmodel.NewDepartmentDispatchSettingModel().Find(ctx, map[string]any{
		"department_id": departmentID,
		"status":        crmmodel.StatusEnabled,
	})
	if setting == nil || setting.ActivePoolID == 0 {
		return nil
	}
	member := crmmodel.NewDispatchPoolMemberModel().Find(ctx, map[string]any{
		"pool_id":       setting.ActivePoolID,
		"department_id": departmentID,
		"staff_id":      staffID,
		"status":        crmmodel.StatusEnabled,
	})
	if member == nil || enabledDispatchPool(ctx, departmentID, setting.ActivePoolID) == nil {
		return nil
	}
	return enabledStaffInDepartment(ctx, staffID, departmentID)
}
