package service

import (
	"context"
	"fmt"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

func (WorkService) FlowAssignees(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	if todoID := firstUint64(payload, "todo_id", "todoId"); todoID > 0 {
		return workTodoAssignees(ctx, staff, todoID)
	}
	instanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId")
	instance, err := activeWorkflowInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	target := firstText(payload, "target")
	if target == "next_stage" {
		if !canCompleteWorkflowStage(staff, instance) {
			return nil, fmt.Errorf("无权选择下一阶段负责人")
		}
		nextTarget, nextErr := nextWorkflowAssignmentTarget(ctx, instance)
		if nextErr != nil {
			return nil, nextErr
		}
		if nextTarget.Stage == nil {
			return map[string]any{"list": []map[string]any{}, "terminal": true}, nil
		}
		return workDepartmentAssignees(ctx, nextTarget.Stage.OwnerDepartmentID, nextTarget.Stage.AssignmentMode), nil
	}
	if !canChangeWorkflowOwner(staff, instance) {
		return nil, fmt.Errorf("只有当前负责部门负责人或流程调度员可以选择阶段负责人")
	}
	return workDepartmentAssignees(ctx, instance.OwnerDepartmentID, "manual"), nil
}

func (WorkService) AssignFlowTask(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	todoID := firstUint64(payload, "todo_id", "todoId")
	assigneeStaffID := firstUint64(payload, "assignee_staff_id", "assigneeStaffId", "staff_id", "staffId")
	if todoID == 0 || assigneeStaffID == 0 {
		return nil, fmt.Errorf("请选择待办和负责人")
	}
	todo, err := AssignPendingWorkTodo(ctx, staff, todoID, assigneeStaffID)
	if err != nil {
		return nil, err
	}
	return workFlowActionResult(ctx, staff, todo.WorkflowInstanceID), nil
}

func (WorkService) ChangeFlowOwner(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	instanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId")
	ownerStaffID := firstUint64(payload, "owner_staff_id", "ownerStaffId", "staff_id", "staffId")
	if instanceID == 0 || ownerStaffID == 0 {
		return nil, fmt.Errorf("请选择流程和负责人")
	}
	instance, err := ChangeWorkflowInstanceOwner(ctx, staff, instanceID, ownerStaffID)
	if err != nil {
		return nil, err
	}
	return workFlowActionResult(ctx, staff, instance.ID), nil
}

func (WorkService) CompleteFlowStage(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	instanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId")
	nextOwnerStaffID := firstUint64(payload, "next_owner_staff_id", "nextOwnerStaffId", "owner_staff_id", "ownerStaffId")
	instance, err := CompleteWorkflowStage(ctx, staff, instanceID, nextOwnerStaffID)
	if err != nil {
		return nil, err
	}
	return workFlowActionResult(ctx, staff, instance.ID), nil
}

func (WorkService) TerminateFlow(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	instanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId")
	reason := firstText(payload, "reason", "remark")
	instance, err := TerminateWorkflowInstance(ctx, staff, instanceID, reason)
	if err != nil {
		return nil, err
	}
	return workFlowActionResult(ctx, staff, instance.ID), nil
}

func (WorkService) RestartFlow(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	instanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId")
	var restarted *crmmodel.WorkflowInstance
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		previous, err := workflowInstanceByID(txCtx, instanceID)
		if err != nil {
			return err
		}
		if previous.Status != crmmodel.ProgressStatusTerminated {
			return fmt.Errorf("只有已终止的流程可以重新发起")
		}
		if previous.LeadID > 0 || previous.CustomerID == 0 || previous.AssetID == 0 || previous.CustomerProductID > 0 {
			return fmt.Errorf("该流程不支持重新发起")
		}
		if staff == nil || staff.ID == 0 || previous.OwnerStaffID != staff.ID && !staff.CanDispatch {
			return fmt.Errorf("只有原负责人或流程调度员可以重新发起流程")
		}
		workflow := crmmodel.NewWorkflowModel().Find(txCtx, map[string]any{
			"id":           previous.WorkflowID,
			"subject_type": crmmodel.WorkflowSubjectCustomerAsset,
			"status":       crmmodel.StatusEnabled,
		})
		if workflow == nil {
			return fmt.Errorf("原流程不存在或已停用")
		}
		if crmmodel.NewWorkflowInstanceModel().Find(txCtx, map[string]any{
			"customer_id":         previous.CustomerID,
			"asset_id":            previous.AssetID,
			"customer_product_id": uint64(0),
			"workflow_id":         workflow.ID,
			"status":              crmmodel.ProgressStatusActive,
		}) != nil {
			return fmt.Errorf("该客户资产已在此流程中")
		}
		restarted, err = restartWorkflowInstance(
			txCtx,
			assetWorkflowSubject(previous.CustomerID, previous.AssetID, 0),
			workflow.ID,
		)
		if err != nil {
			return err
		}
		if restarted == nil || restarted.ID == previous.ID {
			return fmt.Errorf("流程重新发起失败")
		}
		if recordWorkManagementOperation(txCtx, staff, restarted, "workflow_restarted", "重新发起流程", previous.TerminatedReason, map[string]any{
			"source_workflow_instance_id": previous.ID,
		}) == 0 {
			return fmt.Errorf("流程重新发起记录创建失败")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return workFlowActionResult(ctx, staff, restarted.ID), nil
}

func workTodoAssignees(ctx context.Context, staff *WorkStaffSession, todoID uint64) (map[string]any, error) {
	todo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
		"id":     todoID,
		"status": crmmodel.WorkTodoStatusPending,
	})
	if todo == nil {
		return nil, fmt.Errorf("待办不存在或已完成")
	}
	instance, err := activeWorkflowInstance(ctx, todo.WorkflowInstanceID)
	if err != nil || instance.WorkflowID != todo.WorkflowID || instance.StageID != todo.StageID {
		return nil, fmt.Errorf("待办不属于流程当前阶段")
	}
	task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
	if task == nil || !canAssignPendingTodo(staff, instance, task) {
		return nil, fmt.Errorf("无权分配该任务")
	}
	return workDepartmentAssignees(ctx, todo.AssigneeDepartmentID, task.AssigneeMode), nil
}

func workDepartmentAssignees(ctx context.Context, departmentID uint64, assignmentMode string) map[string]any {
	departmentName := ""
	if department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": departmentID}); department != nil {
		departmentName = department.Name
	}
	return map[string]any{
		"list":            workflowStaffCandidates(ctx, departmentID),
		"department_id":   departmentID,
		"department_name": departmentName,
		"assignment_mode": assignmentMode,
	}
}

func workFlowActionResult(ctx context.Context, staff *WorkStaffSession, instanceID uint64) map[string]any {
	return map[string]any{
		"success": true,
		"flow":    workFlowDetail(ctx, staff, instanceID),
	}
}

func workFlowDetail(ctx context.Context, staff *WorkStaffSession, instanceID uint64) map[string]any {
	result := map[string]any{
		"workflow_instance_id": instanceID,
		"tasks":                []map[string]any{},
		"status":               "not_started",
	}
	instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": instanceID})
	if instance == nil {
		return result
	}
	workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": instance.WorkflowID})
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": instance.StageID})
	owner := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": instance.OwnerStaffID})
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": instance.OwnerDepartmentID})
	workflowName := ""
	if workflow != nil {
		workflowName = workflow.Name
	}
	stageName := ""
	if stage != nil {
		stageName = stage.Name
	}
	ownerName := ""
	if owner != nil {
		ownerName = owner.Name
	}
	departmentName := ""
	if department != nil {
		departmentName = department.Name
	}
	pendingRequired := pendingRequiredTodoCount(ctx, instance)
	isActive := instance.Status == crmmodel.ProgressStatusActive
	isCurrentOwner := staff != nil && staff.ID > 0 && instance.OwnerStaffID == staff.ID
	canComplete := isActive && canCompleteWorkflowStage(staff, instance)

	result["id"] = instance.ID
	result["lead_id"] = instance.LeadID
	result["customer_id"] = instance.CustomerID
	result["asset_id"] = instance.AssetID
	result["customer_product_id"] = instance.CustomerProductID
	result["workflow_id"] = instance.WorkflowID
	result["workflow_name"] = workflowName
	result["stage_id"] = instance.StageID
	result["stage_name"] = stageName
	result["stage_assignment_mode"] = stageAssignmentMode(stage)
	result["owner_department_id"] = instance.OwnerDepartmentID
	result["owner_department_name"] = departmentName
	result["owner_staff_id"] = instance.OwnerStaffID
	result["owner_staff_name"] = ownerName
	result["status"] = instance.Status
	result["started_at"] = instance.StartedAt
	result["completed_at"] = instance.CompletedAt
	result["terminated_at"] = instance.TerminatedAt
	result["terminated_reason"] = instance.TerminatedReason
	result["pending_required_count"] = pendingRequired
	result["is_current_owner"] = isCurrentOwner
	result["can_dispatch"] = staff != nil && staff.CanDispatch
	result["can_complete_stage"] = canComplete
	result["ready_to_complete"] = canComplete && pendingRequired == 0
	result["can_terminate"] = isActive && isCurrentOwner
	result["can_change_owner"] = canChangeWorkflowOwner(staff, instance)
	result["can_restart"] = instance.Status == crmmodel.ProgressStatusTerminated && instance.LeadID == 0 &&
		instance.CustomerID > 0 && instance.AssetID > 0 && instance.CustomerProductID == 0 &&
		staff != nil && (isCurrentOwner || staff.CanDispatch) && workflow != nil &&
		workflow.Status == crmmodel.StatusEnabled && workflow.SubjectType == crmmodel.WorkflowSubjectCustomerAsset
	result["tasks"] = workCurrentStageTodoRows(ctx, staff, instance)
	attachCustomerProductFlowFields(ctx, result, instance.CustomerProductID)

	if isActive {
		nextTarget, err := nextWorkflowAssignmentTarget(ctx, instance)
		if err != nil {
			result["configuration_error"] = err.Error()
			result["ready_to_complete"] = false
		} else if nextTarget.Stage == nil {
			result["next_terminal"] = true
		} else {
			nextStage := nextTarget.Stage
			nextAssignmentMode := stageAssignmentMode(nextStage)
			canInheritOwner := inheritedStageOwner(ctx, instance, nextStage) != nil
			if nextTarget.Workflow != nil {
				result["next_workflow_id"] = nextTarget.Workflow.ID
				result["next_workflow_name"] = nextTarget.Workflow.Name
			}
			result["next_stage_id"] = nextStage.ID
			result["next_stage_name"] = nextStage.Name
			result["next_department_id"] = nextStage.OwnerDepartmentID
			result["next_assignment_mode"] = nextAssignmentMode
			result["next_owner_required"] = nextTarget.CrossObject ||
				(!canInheritOwner && nextAssignmentMode == crmmodel.StageAssignmentManual)
		}
	}
	return result
}

func attachCustomerProductFlowFields(ctx context.Context, target map[string]any, customerProductID uint64) {
	if target == nil || customerProductID == 0 {
		return
	}
	customerProduct := crmmodel.NewCustomerProductModel().Find(ctx, map[string]any{"id": customerProductID})
	if customerProduct == nil {
		return
	}
	target["customer_product_status"] = customerProduct.Status
	if product := crmmodel.NewProductModel().Find(ctx, map[string]any{"id": customerProduct.ProductID}); product != nil {
		target["product_id"] = product.ID
		target["product_name"] = product.Name
	}
}

func workCurrentStageTodoRows(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance, withForm ...bool) []map[string]any {
	if instance == nil {
		return []map[string]any{}
	}
	todos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"stage_id":             instance.StageID,
	})
	rows := make([]map[string]any, 0, len(todos))
	for _, todo := range todos {
		row := workTodoTaskMap(ctx, staff, todo, len(withForm) > 0 && withForm[0])
		if len(row) == 0 {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
		canAssign := todo.Status == crmmodel.WorkTodoStatusPending && canAssignPendingTodo(staff, instance, task)
		row["can_assign"] = canAssign
		row["can_reassign"] = canAssign && todo.AssigneeStaffID > 0
		rows = append(rows, row)
	}
	sortWorkTodoTaskMaps(rows)
	return rows
}

func stageAssignmentMode(stage *crmmodel.Stage) string {
	if stage == nil || stage.AssignmentMode == "" {
		return crmmodel.StageAssignmentAuto
	}
	return stage.AssignmentMode
}
