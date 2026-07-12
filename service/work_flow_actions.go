package service

import (
	"context"
	"fmt"

	crmmodel "github.com/dever-package/crm/model"
)

func (WorkService) FlowAssignees(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	if todoID := firstUint64(payload, "todo_id", "todoId"); todoID > 0 {
		return workTodoAssignees(ctx, staff, todoID)
	}
	assetID := firstUint64(payload, "asset_id", "assetId")
	progress, err := activeAssetProgress(ctx, assetID)
	if err != nil {
		return nil, err
	}
	target := firstText(payload, "target")
	if target == "next_stage" {
		if !canCompleteAssetStage(staff, progress) {
			return nil, fmt.Errorf("无权选择下一阶段负责人")
		}
		_, stage, nextErr := nextWorkflowStage(ctx, progress)
		if nextErr != nil {
			return nil, nextErr
		}
		if stage == nil {
			return map[string]any{"list": []map[string]any{}, "terminal": true}, nil
		}
		return workDepartmentAssignees(ctx, stage.OwnerDepartmentID, stage.AssignmentMode), nil
	}
	if !staff.CanDispatch {
		return nil, fmt.Errorf("只有流程调度员可以选择阶段负责人")
	}
	return workDepartmentAssignees(ctx, progress.OwnerDepartmentID, "manual"), nil
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
	return workFlowActionResult(ctx, staff, todo.CustomerID, todo.AssetID), nil
}

func (WorkService) ChangeFlowOwner(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	assetID := firstUint64(payload, "asset_id", "assetId")
	ownerStaffID := firstUint64(payload, "owner_staff_id", "ownerStaffId", "staff_id", "staffId")
	if assetID == 0 || ownerStaffID == 0 {
		return nil, fmt.Errorf("请选择资产和负责人")
	}
	progress, err := ChangeAssetStageOwner(ctx, staff, assetID, ownerStaffID)
	if err != nil {
		return nil, err
	}
	return workFlowActionResult(ctx, staff, progress.CustomerID, progress.AssetID), nil
}

func (WorkService) CompleteFlowStage(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	assetID := firstUint64(payload, "asset_id", "assetId")
	nextOwnerStaffID := firstUint64(payload, "next_owner_staff_id", "nextOwnerStaffId", "owner_staff_id", "ownerStaffId")
	progress, err := CompleteAssetStage(ctx, staff, assetID, nextOwnerStaffID)
	if err != nil {
		return nil, err
	}
	return workFlowActionResult(ctx, staff, progress.CustomerID, progress.AssetID), nil
}

func (WorkService) TerminateFlow(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	assetID := firstUint64(payload, "asset_id", "assetId")
	reason := firstText(payload, "reason", "remark")
	progress, err := TerminateAssetWorkflow(ctx, staff, assetID, reason)
	if err != nil {
		return nil, err
	}
	return workFlowActionResult(ctx, staff, progress.CustomerID, progress.AssetID), nil
}

func workTodoAssignees(ctx context.Context, staff *WorkStaffSession, todoID uint64) (map[string]any, error) {
	todo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
		"id":     todoID,
		"status": crmmodel.WorkTodoStatusPending,
	})
	if todo == nil {
		return nil, fmt.Errorf("待办不存在或已完成")
	}
	progress, err := activeAssetProgress(ctx, todo.AssetID)
	if err != nil || progress.WorkflowID != todo.WorkflowID || progress.StageID != todo.StageID {
		return nil, fmt.Errorf("待办不属于资产当前阶段")
	}
	task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
	if task == nil || !canAssignPendingTodo(staff, progress, task) {
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

func workFlowActionResult(ctx context.Context, staff *WorkStaffSession, customerID, assetID uint64) map[string]any {
	return map[string]any{
		"success": true,
		"flow":    workFlowDetail(ctx, staff, customerID, assetID),
	}
}

func workFlowDetail(ctx context.Context, staff *WorkStaffSession, customerID, assetID uint64) map[string]any {
	result := map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"tasks":       []map[string]any{},
		"status":      "not_started",
	}
	progress := currentWorkCustomerStage(ctx, customerID, assetID)
	if progress == nil {
		return result
	}
	workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": progress.WorkflowID})
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": progress.StageID})
	owner := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": progress.OwnerStaffID})
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": progress.OwnerDepartmentID})
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
	pendingRequired := pendingRequiredTodoCount(ctx, progress)
	isActive := progress.Status == crmmodel.ProgressStatusActive
	isCurrentOwner := staff != nil && staff.ID > 0 && progress.OwnerStaffID == staff.ID
	canComplete := isActive && canCompleteAssetStage(staff, progress)

	result["id"] = progress.ID
	result["workflow_id"] = progress.WorkflowID
	result["workflow_name"] = workflowName
	result["stage_id"] = progress.StageID
	result["stage_name"] = stageName
	result["stage_assignment_mode"] = stageAssignmentMode(stage)
	result["owner_department_id"] = progress.OwnerDepartmentID
	result["owner_department_name"] = departmentName
	result["owner_staff_id"] = progress.OwnerStaffID
	result["owner_staff_name"] = ownerName
	result["status"] = progress.Status
	result["started_at"] = progress.StartedAt
	result["completed_at"] = progress.CompletedAt
	result["terminated_at"] = progress.TerminatedAt
	result["terminated_reason"] = progress.TerminatedReason
	result["pending_required_count"] = pendingRequired
	result["is_current_owner"] = isCurrentOwner
	result["can_dispatch"] = staff != nil && staff.CanDispatch
	result["can_complete_stage"] = canComplete
	result["ready_to_complete"] = canComplete && pendingRequired == 0
	result["can_terminate"] = isActive && isCurrentOwner
	result["can_change_owner"] = isActive && staff != nil && staff.CanDispatch
	result["tasks"] = workCurrentStageTodoRows(ctx, staff, progress)

	if isActive {
		nextWorkflow, nextStage, err := nextWorkflowStage(ctx, progress)
		if err != nil {
			result["configuration_error"] = err.Error()
			result["ready_to_complete"] = false
		} else if nextStage == nil {
			result["next_terminal"] = true
		} else {
			result["next_workflow_id"] = nextWorkflow.ID
			result["next_workflow_name"] = nextWorkflow.Name
			result["next_stage_id"] = nextStage.ID
			result["next_stage_name"] = nextStage.Name
			result["next_department_id"] = nextStage.OwnerDepartmentID
			result["next_assignment_mode"] = stageAssignmentMode(nextStage)
			result["next_owner_required"] = stageAssignmentMode(nextStage) == crmmodel.StageAssignmentManual
		}
	}
	return result
}

func workCurrentStageTodoRows(ctx context.Context, staff *WorkStaffSession, progress *crmmodel.CustomerStage) []map[string]any {
	if progress == nil {
		return []map[string]any{}
	}
	todos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"asset_id": progress.AssetID,
		"stage_id": progress.StageID,
	})
	rows := make([]map[string]any, 0, len(todos))
	for _, todo := range todos {
		row := workTodoTaskMap(ctx, staff, todo, false)
		if len(row) == 0 {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
		canAssign := todo.Status == crmmodel.WorkTodoStatusPending && canAssignPendingTodo(staff, progress, task)
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
