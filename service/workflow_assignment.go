package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

type workflowStaffLoad func(*crmmodel.Staff) int64

func resolveStageOwner(ctx context.Context, stage *crmmodel.Stage, requestedStaffID uint64) (*crmmodel.Staff, error) {
	if stage == nil || !enabledDepartment(ctx, stage.OwnerDepartmentID) {
		return nil, fmt.Errorf("阶段负责部门不存在或已停用")
	}
	mode := stage.AssignmentMode
	if mode == "" {
		mode = crmmodel.StageAssignmentAuto
	}
	switch mode {
	case crmmodel.StageAssignmentAuto:
		return selectStageOwner(ctx, stage.OwnerDepartmentID)
	case crmmodel.StageAssignmentManual:
		if requestedStaffID == 0 {
			return nil, fmt.Errorf("进入%s阶段前请选择负责人", stage.Name)
		}
		staff := enabledStaffInDepartment(ctx, requestedStaffID, stage.OwnerDepartmentID)
		if staff == nil {
			return nil, fmt.Errorf("所选负责人不属于%s阶段的负责部门或已停用", stage.Name)
		}
		return staff, nil
	default:
		return nil, fmt.Errorf("阶段分配方式无效")
	}
}

func selectStageOwner(ctx context.Context, departmentID uint64) (*crmmodel.Staff, error) {
	return selectLeastLoadedStaff(ctx, departmentID, func(staff *crmmodel.Staff) int64 {
		return crmmodel.NewCustomerStageModel().Count(ctx, map[string]any{
			"owner_staff_id": staff.ID,
			"status":         crmmodel.ProgressStatusActive,
		})
	})
}

func selectTaskAssignee(ctx context.Context, departmentID uint64) (*crmmodel.Staff, error) {
	return selectLeastLoadedStaff(ctx, departmentID, func(staff *crmmodel.Staff) int64 {
		return crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
			"assignee_staff_id": staff.ID,
			"status":            crmmodel.WorkTodoStatusPending,
		})
	})
}

func selectLeastLoadedStaff(ctx context.Context, departmentID uint64, load workflowStaffLoad) (*crmmodel.Staff, error) {
	staffRows := enabledDepartmentStaff(ctx, departmentID)
	var selected *crmmodel.Staff
	var selectedLoad int64
	for _, staff := range staffRows {
		if staff == nil {
			continue
		}
		currentLoad := load(staff)
		if selected == nil || currentLoad < selectedLoad || currentLoad == selectedLoad && assignedBefore(staff, selected) {
			selected = staff
			selectedLoad = currentLoad
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("目标部门没有启用人员，无法自动分配")
	}
	now := time.Now()
	crmmodel.NewStaffModel().Update(ctx, map[string]any{"id": selected.ID}, map[string]any{"last_assigned_at": now})
	selected.LastAssignedAt = &now
	return selected, nil
}

func assignedBefore(current, selected *crmmodel.Staff) bool {
	if current == nil || selected == nil {
		return current != nil
	}
	if current.LastAssignedAt == nil {
		return selected.LastAssignedAt != nil || current.ID < selected.ID
	}
	if selected.LastAssignedAt == nil {
		return false
	}
	if current.LastAssignedAt.Equal(*selected.LastAssignedAt) {
		return current.ID < selected.ID
	}
	return current.LastAssignedAt.Before(*selected.LastAssignedAt)
}

func enabledDepartmentStaff(ctx context.Context, departmentID uint64) []*crmmodel.Staff {
	if !enabledDepartment(ctx, departmentID) {
		return nil
	}
	return crmmodel.NewStaffModel().Select(ctx, map[string]any{
		"department_id": departmentID,
		"status":        crmmodel.StatusEnabled,
	}, map[string]any{"order": "id asc"})
}

func enabledStaffInDepartment(ctx context.Context, staffID, departmentID uint64) *crmmodel.Staff {
	if staffID == 0 || departmentID == 0 {
		return nil
	}
	return crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"id":            staffID,
		"department_id": departmentID,
		"status":        crmmodel.StatusEnabled,
	})
}

func resolveTaskAssignee(ctx context.Context, progress *crmmodel.CustomerStage, task *crmmodel.Task) (uint64, uint64, error) {
	if progress == nil || task == nil {
		return 0, 0, fmt.Errorf("资产进度和任务不能为空")
	}
	switch task.AssigneeMode {
	case crmmodel.TaskAssigneeStage:
		staff := enabledStaffInDepartment(ctx, progress.OwnerStaffID, progress.OwnerDepartmentID)
		if staff == nil {
			return 0, 0, fmt.Errorf("当前阶段负责人不存在或已停用")
		}
		return staff.DepartmentID, staff.ID, nil
	case crmmodel.TaskAssigneeAuto:
		if !enabledDepartment(ctx, task.AssigneeDepartmentID) {
			return 0, 0, fmt.Errorf("任务目标部门不存在或已停用")
		}
		staff, err := selectTaskAssignee(ctx, task.AssigneeDepartmentID)
		if err != nil {
			return 0, 0, err
		}
		return staff.DepartmentID, staff.ID, nil
	case crmmodel.TaskAssigneeManual:
		if !enabledDepartment(ctx, task.AssigneeDepartmentID) {
			return 0, 0, fmt.Errorf("任务目标部门不存在或已停用")
		}
		return task.AssigneeDepartmentID, 0, nil
	default:
		return 0, 0, fmt.Errorf("任务负责方式无效")
	}
}

func enabledDepartment(ctx context.Context, departmentID uint64) bool {
	return departmentID > 0 && crmmodel.NewDepartmentModel().Find(ctx, map[string]any{
		"id":     departmentID,
		"status": crmmodel.StatusEnabled,
	}) != nil
}

func AssignPendingWorkTodo(ctx context.Context, staff *WorkStaffSession, todoID, assigneeStaffID uint64) (*crmmodel.WorkTodo, error) {
	var assigned *crmmodel.WorkTodo
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		if staff == nil || staff.ID == 0 {
			return fmt.Errorf("请先登录")
		}
		todo := crmmodel.NewWorkTodoModel().Find(txCtx, map[string]any{
			"id":     todoID,
			"status": crmmodel.WorkTodoStatusPending,
		})
		if todo == nil {
			return fmt.Errorf("待办不存在或已完成")
		}
		progress, err := activeAssetProgress(txCtx, todo.AssetID)
		if err != nil || progress.WorkflowID != todo.WorkflowID || progress.StageID != todo.StageID {
			return fmt.Errorf("待办不属于资产当前阶段")
		}
		task := crmmodel.NewTaskModel().Find(txCtx, map[string]any{"id": todo.TaskID})
		if task == nil {
			return fmt.Errorf("任务配置不存在")
		}
		isCurrentOwner := progress.OwnerStaffID == staff.ID
		if !staff.CanDispatch && (!isCurrentOwner || task.AssigneeMode != crmmodel.TaskAssigneeManual) {
			return fmt.Errorf("只有当前负责人可以分配手动任务，其他改派由流程调度员处理")
		}
		target := enabledStaffInDepartment(txCtx, assigneeStaffID, todo.AssigneeDepartmentID)
		if target == nil {
			return fmt.Errorf("所选人员不属于任务目标部门或已停用")
		}
		previousStaffID := todo.AssigneeStaffID
		if previousStaffID == target.ID {
			assigned = todo
			return nil
		}
		now := time.Now()
		if crmmodel.NewWorkTodoModel().Update(txCtx, map[string]any{
			"id":                todo.ID,
			"status":            crmmodel.WorkTodoStatusPending,
			"assignee_staff_id": previousStaffID,
		}, map[string]any{
			"assignee_staff_id": target.ID,
			"updated_at":        now,
		}) == 0 {
			return fmt.Errorf("待办已变化，请刷新后重试")
		}
		todo.AssigneeStaffID = target.ID
		todo.UpdatedAt = now
		if recordWorkTodoAssignment(txCtx, staff, progress, todo, task, previousStaffID, target) == 0 {
			return fmt.Errorf("任务分配记录创建失败")
		}
		assigned = todo
		return nil
	})
	return assigned, err
}

func ChangeAssetStageOwner(ctx context.Context, staff *WorkStaffSession, assetID, ownerStaffID uint64) (*crmmodel.CustomerStage, error) {
	var changed *crmmodel.CustomerStage
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		if staff == nil || staff.ID == 0 || !staff.CanDispatch {
			return fmt.Errorf("只有流程调度员可以更换阶段负责人")
		}
		progress, err := activeAssetProgress(txCtx, assetID)
		if err != nil {
			return err
		}
		target := enabledStaffInDepartment(txCtx, ownerStaffID, progress.OwnerDepartmentID)
		if target == nil {
			return fmt.Errorf("所选人员不属于当前阶段负责部门或已停用")
		}
		previousStaffID := progress.OwnerStaffID
		if previousStaffID == target.ID {
			changed = progress
			return nil
		}
		now := time.Now()
		if crmmodel.NewCustomerStageModel().Update(txCtx, map[string]any{
			"id":             progress.ID,
			"status":         crmmodel.ProgressStatusActive,
			"owner_staff_id": previousStaffID,
		}, map[string]any{
			"owner_staff_id": target.ID,
			"updated_at":     now,
		}) == 0 {
			return fmt.Errorf("流程已变化，请刷新后重试")
		}
		if err := reassignStageOwnerTodos(txCtx, progress, target.ID, now); err != nil {
			return err
		}
		progress.OwnerStaffID = target.ID
		progress.UpdatedAt = now
		if recordWorkManagementOperation(txCtx, staff, progress, "owner_changed", "更换阶段负责人", target.Name, map[string]any{
			"from_staff_id": previousStaffID,
			"to_staff_id":   target.ID,
		}) == 0 {
			return fmt.Errorf("负责人变更记录创建失败")
		}
		changed = progress
		return nil
	})
	return changed, err
}

func reassignStageOwnerTodos(ctx context.Context, progress *crmmodel.CustomerStage, ownerStaffID uint64, changedAt time.Time) error {
	todos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"asset_id": progress.AssetID,
		"stage_id": progress.StageID,
		"status":   crmmodel.WorkTodoStatusPending,
	})
	for _, todo := range todos {
		if todo == nil {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
		if task == nil || task.AssigneeMode != crmmodel.TaskAssigneeStage {
			continue
		}
		if crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
			"id":     todo.ID,
			"status": crmmodel.WorkTodoStatusPending,
		}, map[string]any{
			"assignee_department_id": progress.OwnerDepartmentID,
			"assignee_staff_id":      ownerStaffID,
			"updated_at":             changedAt,
		}) == 0 {
			return fmt.Errorf("阶段负责人待办已变化，请刷新后重试")
		}
	}
	return nil
}

func workflowStaffCandidates(ctx context.Context, departmentID uint64) []map[string]any {
	rows := enabledDepartmentStaff(ctx, departmentID)
	result := make([]map[string]any, 0, len(rows))
	for _, staff := range rows {
		if staff == nil {
			continue
		}
		result = append(result, map[string]any{
			"id":                 staff.ID,
			"name":               staff.Name,
			"department_id":      staff.DepartmentID,
			"active_asset_count": crmmodel.NewCustomerStageModel().Count(ctx, map[string]any{"owner_staff_id": staff.ID, "status": crmmodel.ProgressStatusActive}),
			"pending_task_count": crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{"assignee_staff_id": staff.ID, "status": crmmodel.WorkTodoStatusPending}),
			"last_assigned_at":   staff.LastAssignedAt,
		})
	}
	return result
}
