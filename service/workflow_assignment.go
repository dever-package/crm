package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

type workflowStaffLoad func(*crmmodel.Staff) int64

func resolveStageOwner(ctx context.Context, stage *crmmodel.Stage, requestedStaffID uint64) (*crmmodel.Staff, bool, error) {
	if stage == nil || !enabledDepartment(ctx, stage.OwnerDepartmentID) {
		return nil, false, fmt.Errorf("阶段负责部门不存在或已停用")
	}
	if requestedStaffID > 0 {
		staff := enabledStaffInDepartment(ctx, requestedStaffID, stage.OwnerDepartmentID)
		if staff == nil {
			return nil, false, fmt.Errorf("所选负责人不属于%s阶段的负责部门或已停用", stage.Name)
		}
		return staff, false, nil
	}
	mode := stage.AssignmentMode
	if mode == "" {
		mode = crmmodel.StageAssignmentAuto
	}
	switch mode {
	case crmmodel.StageAssignmentAuto:
		staff, err := selectStageOwner(ctx, stage.OwnerDepartmentID)
		return staff, staff != nil, err
	case crmmodel.StageAssignmentManual:
		return nil, false, fmt.Errorf("进入%s阶段前请选择负责人", stage.Name)
	default:
		return nil, false, fmt.Errorf("阶段分配方式无效")
	}
}

func resolveStageTransitionOwner(
	ctx context.Context,
	instance *crmmodel.WorkflowInstance,
	stage *crmmodel.Stage,
	requestedStaffID uint64,
) (*crmmodel.Staff, bool, error) {
	if requestedStaffID > 0 {
		return resolveStageOwner(ctx, stage, requestedStaffID)
	}
	if owner := inheritedStageOwner(ctx, instance, stage); owner != nil {
		return owner, false, nil
	}
	return resolveStageOwner(ctx, stage, 0)
}

func inheritedStageOwner(
	ctx context.Context,
	instance *crmmodel.WorkflowInstance,
	stage *crmmodel.Stage,
) *crmmodel.Staff {
	if instance == nil || stage == nil || instance.OwnerDepartmentID != stage.OwnerDepartmentID {
		return nil
	}
	return enabledStaffInDepartment(ctx, instance.OwnerStaffID, stage.OwnerDepartmentID)
}

func selectStageOwner(ctx context.Context, departmentID uint64) (*crmmodel.Staff, error) {
	return selectDepartmentAssignee(ctx, departmentID, func(staff *crmmodel.Staff) int64 {
		return crmmodel.NewWorkflowInstanceModel().Count(ctx, map[string]any{
			"owner_staff_id": staff.ID,
			"status":         crmmodel.ProgressStatusActive,
		})
	})
}

func selectTaskAssignee(ctx context.Context, departmentID uint64) (*crmmodel.Staff, error) {
	return selectDepartmentAssignee(ctx, departmentID, taskPendingLoad(ctx))
}

func previewTaskAssignee(ctx context.Context, instance *crmmodel.WorkflowInstance, task *crmmodel.Task) (*crmmodel.Staff, error) {
	if instance == nil || task == nil {
		return nil, fmt.Errorf("流程实例和任务不能为空")
	}
	switch task.AssigneeMode {
	case crmmodel.TaskAssigneeStage:
		return enabledStaffInDepartment(ctx, instance.OwnerStaffID, instance.OwnerDepartmentID), nil
	case crmmodel.TaskAssigneeAuto:
		return previewDepartmentAssignee(ctx, task.AssigneeDepartmentID, taskPendingLoad(ctx))
	case crmmodel.TaskAssigneePrevious:
		if staff := previousWorkflowDepartmentAssignee(ctx, instance, task.AssigneeDepartmentID); staff != nil {
			return staff, nil
		}
		return previewDepartmentAssignee(ctx, task.AssigneeDepartmentID, taskPendingLoad(ctx))
	case crmmodel.TaskAssigneeDepartmentLeader:
		return enabledDepartmentLeader(ctx, task.AssigneeDepartmentID), nil
	case crmmodel.TaskAssigneeManual:
		return nil, nil
	default:
		return nil, fmt.Errorf("任务负责方式无效")
	}
}

func taskPendingLoad(ctx context.Context) workflowStaffLoad {
	return func(staff *crmmodel.Staff) int64 {
		return crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
			"assignee_staff_id": staff.ID,
			"status":            crmmodel.WorkTodoStatusPending,
		})
	}
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
		return nil, nil
	}
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

func previousWorkflowDepartmentAssignee(
	ctx context.Context,
	instance *crmmodel.WorkflowInstance,
	departmentID uint64,
) *crmmodel.Staff {
	if instance == nil || departmentID == 0 {
		return nil
	}
	todos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"workflow_instance_id":   instance.ID,
		"assignee_department_id": departmentID,
	}, map[string]any{"order": "updated_at desc,id desc"})
	for _, todo := range todos {
		if todo == nil || todo.AssigneeStaffID == 0 {
			continue
		}
		if staff := enabledStaffInDepartment(ctx, todo.AssigneeStaffID, departmentID); staff != nil {
			return staff
		}
	}
	return nil
}

func resolveTaskAssignee(ctx context.Context, instance *crmmodel.WorkflowInstance, task *crmmodel.Task) (uint64, uint64, bool, error) {
	if instance == nil || task == nil {
		return 0, 0, false, fmt.Errorf("流程实例和任务不能为空")
	}
	switch task.AssigneeMode {
	case crmmodel.TaskAssigneeStage:
		staff := enabledStaffInDepartment(ctx, instance.OwnerStaffID, instance.OwnerDepartmentID)
		if staff == nil {
			return 0, 0, false, fmt.Errorf("当前阶段负责人不存在或已停用")
		}
		return staff.DepartmentID, staff.ID, false, nil
	case crmmodel.TaskAssigneeAuto:
		return resolveDepartmentTaskAssignee(ctx, task.AssigneeDepartmentID)
	case crmmodel.TaskAssigneePrevious:
		if staff := previousWorkflowDepartmentAssignee(ctx, instance, task.AssigneeDepartmentID); staff != nil {
			return staff.DepartmentID, staff.ID, false, nil
		}
		return resolveDepartmentTaskAssignee(ctx, task.AssigneeDepartmentID)
	case crmmodel.TaskAssigneeDepartmentLeader:
		staff := enabledDepartmentLeader(ctx, task.AssigneeDepartmentID)
		if staff == nil {
			return 0, 0, false, fmt.Errorf("任务负责部门尚未配置启用的部门负责人")
		}
		return staff.DepartmentID, staff.ID, false, nil
	case crmmodel.TaskAssigneeManual:
		if !enabledDepartment(ctx, task.AssigneeDepartmentID) {
			return 0, 0, false, fmt.Errorf("任务目标部门不存在或已停用")
		}
		return task.AssigneeDepartmentID, 0, false, nil
	default:
		return 0, 0, false, fmt.Errorf("任务负责方式无效")
	}
}

func enabledDepartmentLeader(ctx context.Context, departmentID uint64) *crmmodel.Staff {
	if departmentID == 0 {
		return nil
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{
		"id":     departmentID,
		"status": crmmodel.StatusEnabled,
	})
	if department == nil || department.LeaderStaffID == 0 {
		return nil
	}
	return enabledStaffInDepartment(ctx, department.LeaderStaffID, department.ID)
}

func resolveDepartmentTaskAssignee(ctx context.Context, departmentID uint64) (uint64, uint64, bool, error) {
	if !enabledDepartment(ctx, departmentID) {
		return 0, 0, false, fmt.Errorf("任务目标部门不存在或已停用")
	}
	staff, err := selectTaskAssignee(ctx, departmentID)
	if err != nil {
		return 0, 0, false, err
	}
	if staff == nil {
		return departmentID, 0, false, nil
	}
	return staff.DepartmentID, staff.ID, true, nil
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
		instance, err := activeWorkflowInstance(txCtx, todo.WorkflowInstanceID)
		if err != nil || instance.WorkflowID != todo.WorkflowID || instance.StageID != todo.StageID {
			return fmt.Errorf("待办不属于流程当前阶段")
		}
		task := crmmodel.NewTaskModel().Find(txCtx, map[string]any{"id": todo.TaskID})
		if task == nil {
			return fmt.Errorf("任务配置不存在")
		}
		if !canAssignPendingTodo(txCtx, staff, instance, todo, task) {
			return fmt.Errorf("只有当前负责人、目标部门负责人或流程调度员可以分配该任务")
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
		if recordWorkTodoAssignment(txCtx, staff, instance, todo, task, previousStaffID, target) == 0 {
			return fmt.Errorf("任务分配记录创建失败")
		}
		if err := recordManualDispatch(txCtx, target.DepartmentID, target.ID, workflowDispatchReference{
			Source:             crmmodel.DispatchSourceManual,
			LeadID:             todo.LeadID,
			WorkflowInstanceID: todo.WorkflowInstanceID,
			WorkTodoID:         todo.ID,
			PreviousStaffID:    previousStaffID,
			OperatorStaffID:    staff.ID,
		}); err != nil {
			return err
		}
		if err := syncWorkflowMeetingParticipants(txCtx, todo.WorkflowInstanceID); err != nil {
			return err
		}
		assigned = todo
		return nil
	})
	return assigned, err
}

func canAssignPendingTodo(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance, todo *crmmodel.WorkTodo, task *crmmodel.Task) bool {
	if staff == nil || staff.ID == 0 || instance == nil || todo == nil || task == nil || instance.Status != crmmodel.ProgressStatusActive {
		return false
	}
	if staff.CanDispatch {
		return true
	}
	if isDepartmentLeader(ctx, staff, todo.AssigneeDepartmentID) {
		return true
	}
	return instance.OwnerStaffID == staff.ID && task.AssigneeMode == crmmodel.TaskAssigneeManual
}

func ChangeWorkflowInstanceOwner(ctx context.Context, staff *WorkStaffSession, instanceID, ownerStaffID uint64) (*crmmodel.WorkflowInstance, error) {
	var changed *crmmodel.WorkflowInstance
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		instance, err := activeWorkflowInstance(txCtx, instanceID)
		if err != nil {
			return err
		}
		changed, err = changeWorkflowInstanceOwner(txCtx, staff, instance, ownerStaffID)
		return err
	})
	return changed, err
}

func changeWorkflowInstanceOwner(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance, ownerStaffID uint64) (*crmmodel.WorkflowInstance, error) {
	if !canChangeWorkflowOwner(ctx, staff, instance) {
		return nil, fmt.Errorf("只有当前负责部门负责人或流程调度员可以更换负责人")
	}
	target := enabledStaffInDepartment(ctx, ownerStaffID, instance.OwnerDepartmentID)
	if target == nil {
		return nil, fmt.Errorf("所选人员不属于当前阶段负责部门或已停用")
	}
	previousStaffID := instance.OwnerStaffID
	if previousStaffID == target.ID {
		if err := reassignWorkflowOwnerTodos(ctx, staff, instance, target, time.Now()); err != nil {
			return nil, err
		}
		if err := syncWorkflowMeetingParticipants(ctx, instance.ID); err != nil {
			return nil, err
		}
		return instance, nil
	}
	now := time.Now()
	if crmmodel.NewWorkflowInstanceModel().Update(ctx, map[string]any{
		"id":             instance.ID,
		"status":         crmmodel.ProgressStatusActive,
		"owner_staff_id": previousStaffID,
	}, map[string]any{
		"owner_staff_id": target.ID,
		"updated_at":     now,
	}) == 0 {
		return nil, fmt.Errorf("流程已变化，请刷新后重试")
	}
	if err := reassignWorkflowOwnerTodos(ctx, staff, instance, target, now); err != nil {
		return nil, err
	}
	instance.OwnerStaffID = target.ID
	instance.UpdatedAt = now
	if instance.LeadID > 0 {
		if crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": instance.LeadID}, map[string]any{
			"owner_department_id": instance.OwnerDepartmentID,
			"owner_staff_id":      target.ID,
			"updated_at":          now,
		}) == 0 {
			return nil, fmt.Errorf("线索负责人更新失败")
		}
	}
	if recordWorkManagementOperation(ctx, staff, instance, "owner_changed", "更换阶段负责人", target.Name, map[string]any{
		"from_staff_id": previousStaffID,
		"to_staff_id":   target.ID,
	}) == 0 {
		return nil, fmt.Errorf("负责人变更记录创建失败")
	}
	if err := recordManualDispatch(ctx, target.DepartmentID, target.ID, workflowDispatchReference{
		Source:             crmmodel.DispatchSourceManual,
		LeadID:             instance.LeadID,
		WorkflowInstanceID: instance.ID,
		PreviousStaffID:    previousStaffID,
		OperatorStaffID:    staff.ID,
	}); err != nil {
		return nil, err
	}
	if previousStaffID == 0 {
		stage := crmmodel.NewStageModel().Find(ctx, map[string]any{
			"id":     instance.StageID,
			"status": crmmodel.StatusEnabled,
		})
		if stage == nil {
			return nil, fmt.Errorf("当前阶段不存在或已停用")
		}
		if err := createStageTodos(ctx, instance, stage); err != nil {
			return nil, err
		}
	} else if err := syncWorkflowMeetingParticipants(ctx, instance.ID); err != nil {
		return nil, err
	}
	return instance, nil
}

func reassignWorkflowOwnerTodos(
	ctx context.Context,
	operator *WorkStaffSession,
	instance *crmmodel.WorkflowInstance,
	owner *crmmodel.Staff,
	changedAt time.Time,
) error {
	if instance == nil || owner == nil {
		return fmt.Errorf("流程负责人不能为空")
	}
	todos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"stage_id":             instance.StageID,
		"status":               crmmodel.WorkTodoStatusPending,
	})
	for _, todo := range todos {
		if todo == nil {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
		if !todoFollowsWorkflowOwner(instance, todo, task) || todo.AssigneeStaffID == owner.ID {
			continue
		}
		previousStaffID := todo.AssigneeStaffID
		if crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
			"id":                todo.ID,
			"status":            crmmodel.WorkTodoStatusPending,
			"assignee_staff_id": previousStaffID,
		}, map[string]any{
			"assignee_department_id": owner.DepartmentID,
			"assignee_staff_id":      owner.ID,
			"updated_at":             changedAt,
		}) == 0 {
			return fmt.Errorf("阶段负责人待办已变化，请刷新后重试")
		}
		todo.AssigneeDepartmentID = owner.DepartmentID
		todo.AssigneeStaffID = owner.ID
		todo.UpdatedAt = changedAt
		if recordWorkTodoAssignment(ctx, operator, instance, todo, task, previousStaffID, owner) == 0 {
			return fmt.Errorf("任务改派记录创建失败")
		}
	}
	return nil
}

func todoFollowsWorkflowOwner(instance *crmmodel.WorkflowInstance, todo *crmmodel.WorkTodo, task *crmmodel.Task) bool {
	if instance == nil || todo == nil || task == nil {
		return false
	}
	if task.AssigneeMode == crmmodel.TaskAssigneeStage {
		return true
	}
	// A lead handoff also transfers pending work owned by its current department.
	return instance.LeadID > 0 && todo.AssigneeDepartmentID == instance.OwnerDepartmentID
}

func workflowStaffCandidates(ctx context.Context, departmentID uint64) []map[string]any {
	rows := enabledDepartmentStaff(ctx, departmentID)
	result := make([]map[string]any, 0, len(rows))
	for _, staff := range rows {
		if staff == nil {
			continue
		}
		activeFlowCount := crmmodel.NewWorkflowInstanceModel().Count(ctx, map[string]any{
			"owner_staff_id": staff.ID,
			"status":         crmmodel.ProgressStatusActive,
		})
		result = append(result, map[string]any{
			"id":                 staff.ID,
			"name":               staff.Name,
			"department_id":      staff.DepartmentID,
			"active_flow_count":  activeFlowCount,
			"active_asset_count": activeFlowCount,
			"pending_task_count": crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{"assignee_staff_id": staff.ID, "status": crmmodel.WorkTodoStatusPending}),
			"last_assigned_at":   staff.LastAssignedAt,
		})
	}
	return result
}
