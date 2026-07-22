package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

var errPendingDispatchChanged = errors.New("待派单已变化")

type pendingDispatchSummary struct {
	StageAssigned int `json:"stage_assigned"`
	TaskAssigned  int `json:"task_assigned"`
	Remaining     int `json:"remaining"`
}

func RetryPendingDepartmentDispatch(ctx context.Context, departmentID uint64) (pendingDispatchSummary, error) {
	summary := pendingDispatchSummary{}
	var retryErr error
	if !enabledDepartment(ctx, departmentID) {
		return summary, fmt.Errorf("目标部门不存在或已停用")
	}
	instances := crmmodel.NewWorkflowInstanceModel().Select(ctx, map[string]any{
		"owner_department_id": departmentID,
		"owner_staff_id":      uint64(0),
		"status":              crmmodel.ProgressStatusActive,
	}, map[string]any{"order": "id asc"})
	reservedInstanceIDs := reservedLeadDispatchInstanceIDs(ctx, departmentID)
	for _, instance := range instances {
		if instance == nil || reservedInstanceIDs[instance.ID] {
			continue
		}
		assigned := false
		err := orm.Transaction(ctx, func(txCtx context.Context) error {
			var err error
			assigned, err = retryPendingWorkflowOwner(txCtx, instance.ID, departmentID)
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
		if assigned {
			summary.StageAssigned++
		}
	}

	todos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"assignee_department_id": departmentID,
		"assignee_staff_id":      uint64(0),
		"status":                 crmmodel.WorkTodoStatusPending,
	}, map[string]any{"order": "id asc"})
	for _, todo := range todos {
		if todo == nil {
			continue
		}
		assigned := false
		err := orm.Transaction(ctx, func(txCtx context.Context) error {
			var err error
			assigned, err = retryPendingWorkTodo(txCtx, todo.ID, departmentID)
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
		if assigned {
			summary.TaskAssigned++
		}
	}
	summary.Remaining = pendingDepartmentDispatchCount(ctx, departmentID)
	return summary, retryErr
}

func retryPendingWorkflowOwner(ctx context.Context, instanceID, departmentID uint64) (bool, error) {
	instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
		"id":                  instanceID,
		"owner_department_id": departmentID,
		"owner_staff_id":      uint64(0),
		"status":              crmmodel.ProgressStatusActive,
	})
	if instance == nil {
		return false, nil
	}
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{
		"id":                  instance.StageID,
		"owner_department_id": departmentID,
		"status":              crmmodel.StatusEnabled,
	})
	if stage == nil || stageAssignmentMode(stage) != crmmodel.StageAssignmentAuto {
		return false, nil
	}
	owner, err := selectStageOwner(ctx, departmentID)
	if err != nil || owner == nil {
		return false, err
	}
	now := time.Now()
	if crmmodel.NewWorkflowInstanceModel().Update(ctx, map[string]any{
		"id":                  instance.ID,
		"owner_department_id": departmentID,
		"owner_staff_id":      uint64(0),
		"status":              crmmodel.ProgressStatusActive,
	}, map[string]any{
		"owner_staff_id": owner.ID,
		"updated_at":     now,
	}) == 0 {
		return false, errPendingDispatchChanged
	}
	instance.OwnerStaffID = owner.ID
	instance.UpdatedAt = now
	if instance.LeadID > 0 {
		crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": instance.LeadID}, map[string]any{
			"owner_department_id": departmentID,
			"owner_staff_id":      owner.ID,
			"updated_at":          now,
		})
	}
	if err := recordAutomaticDispatch(ctx, departmentID, owner.ID, workflowDispatchReference{
		Source:             crmmodel.DispatchSourcePending,
		LeadID:             instance.LeadID,
		WorkflowInstanceID: instance.ID,
	}); err != nil {
		return false, err
	}
	if recordWorkStageChange(ctx, nil, instance, workStageChange{
		FromWorkflowID: instance.WorkflowID,
		FromStageID:    instance.StageID,
		ToWorkflowID:   instance.WorkflowID,
		ToStageID:      instance.StageID,
		ResultValue:    "assigned",
		Title:          "待派单已分配",
		Content:        "负责人：" + owner.Name,
		Snapshot: map[string]any{
			"owner_department_id": departmentID,
			"owner_staff_id":      owner.ID,
		},
	}) == 0 {
		return false, fmt.Errorf("待派单阶段记录创建失败")
	}
	if err := createStageTodos(ctx, instance, stage); err != nil {
		return false, err
	}
	return true, nil
}

func retryPendingWorkTodo(ctx context.Context, todoID, departmentID uint64) (bool, error) {
	todo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
		"id":                     todoID,
		"assignee_department_id": departmentID,
		"assignee_staff_id":      uint64(0),
		"status":                 crmmodel.WorkTodoStatusPending,
	})
	if todo == nil {
		return false, nil
	}
	instance, err := activeWorkflowInstance(ctx, todo.WorkflowInstanceID)
	if err != nil || instance.StageID != todo.StageID || instance.WorkflowID != todo.WorkflowID {
		return false, nil
	}
	task := crmmodel.NewTaskModel().Find(ctx, map[string]any{
		"id":            todo.TaskID,
		"assignee_mode": crmmodel.TaskAssigneeAuto,
		"status":        crmmodel.StatusEnabled,
	})
	if task == nil || task.AssigneeDepartmentID != departmentID {
		return false, nil
	}
	assignee, err := selectTaskAssignee(ctx, departmentID)
	if err != nil || assignee == nil {
		return false, err
	}
	now := time.Now()
	if crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
		"id":                     todo.ID,
		"assignee_department_id": departmentID,
		"assignee_staff_id":      uint64(0),
		"status":                 crmmodel.WorkTodoStatusPending,
	}, map[string]any{
		"assignee_staff_id": assignee.ID,
		"updated_at":        now,
	}) == 0 {
		return false, errPendingDispatchChanged
	}
	todo.AssigneeStaffID = assignee.ID
	todo.UpdatedAt = now
	if err := recordAutomaticDispatch(ctx, departmentID, assignee.ID, workflowDispatchReference{
		Source:             crmmodel.DispatchSourcePending,
		LeadID:             todo.LeadID,
		WorkflowInstanceID: todo.WorkflowInstanceID,
		WorkTodoID:         todo.ID,
	}); err != nil {
		return false, err
	}
	if recordWorkTodoAssignment(ctx, nil, instance, todo, task, 0, assignee) == 0 {
		return false, fmt.Errorf("待派单任务记录创建失败")
	}
	if err := syncWorkflowMeetingParticipants(ctx, instance.ID); err != nil {
		return false, err
	}
	return true, nil
}

func RetryPendingDispatches(ctx context.Context) (map[string]any, error) {
	result := map[string]any{
		"lead_workflows": 0,
		"lead_queued":    0,
		"lead_assigned":  0,
		"lead_remaining": 0,
		"departments":    0,
		"stage_assigned": 0,
		"task_assigned":  0,
		"remaining":      0,
	}
	errorsFound := make([]string, 0)
	for _, route := range crmmodel.NewLeadDispatchRouteModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}, map[string]any{"order": "workflow_id asc,id asc"}) {
		if route == nil {
			continue
		}
		summary, err := retryLeadDispatchWorkflow(ctx, route.WorkflowID)
		workflowName := fmt.Sprintf("线索流程#%d", route.WorkflowID)
		if workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": route.WorkflowID}); workflow != nil {
			workflowName = workflow.Name
		}
		if err != nil {
			errorsFound = append(errorsFound, fmt.Sprintf("%s：%s", workflowName, err.Error()))
		}
		result["lead_workflows"] = result["lead_workflows"].(int) + 1
		result["lead_queued"] = result["lead_queued"].(int) + summary.Queued
		result["lead_assigned"] = result["lead_assigned"].(int) + summary.Assigned
		result["lead_remaining"] = result["lead_remaining"].(int) + summary.Remaining
	}
	for _, department := range crmmodel.NewDepartmentModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"}) {
		if department == nil {
			continue
		}
		summary, err := RetryPendingDepartmentDispatch(ctx, department.ID)
		if err != nil {
			errorsFound = append(errorsFound, fmt.Sprintf("%s：%s", department.Name, err.Error()))
		}
		result["departments"] = result["departments"].(int) + 1
		result["stage_assigned"] = result["stage_assigned"].(int) + summary.StageAssigned
		result["task_assigned"] = result["task_assigned"].(int) + summary.TaskAssigned
		result["remaining"] = result["remaining"].(int) + summary.Remaining
	}
	result["errors"] = errorsFound
	return result, nil
}

func pendingDepartmentDispatchCount(ctx context.Context, departmentID uint64) int {
	reservedInstanceIDs := reservedLeadDispatchInstanceIDs(ctx, departmentID)
	stageCount := 0
	for _, instance := range crmmodel.NewWorkflowInstanceModel().Select(ctx, map[string]any{
		"owner_department_id": departmentID,
		"owner_staff_id":      uint64(0),
		"status":              crmmodel.ProgressStatusActive,
	}) {
		if instance != nil && !reservedInstanceIDs[instance.ID] {
			stageCount++
		}
	}
	taskCount := crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
		"assignee_department_id": departmentID,
		"assignee_staff_id":      uint64(0),
		"status":                 crmmodel.WorkTodoStatusPending,
	})
	return stageCount + int(taskCount)
}
