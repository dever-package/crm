package service

import (
	"context"
	"sort"

	crmmodel "github.com/dever-package/crm/model"
)

// workPersonalWorkload is the single source of truth for personal counters.
// Management visibility is intentionally handled elsewhere.
type workPersonalWorkload struct {
	instances              []*crmmodel.WorkflowInstance
	pendingTodosByInstance map[uint64][]*crmmodel.WorkTodo
}

func currentWorkPersonalWorkload(ctx context.Context, staff *WorkStaffSession) workPersonalWorkload {
	workload := workPersonalWorkload{
		instances:              []*crmmodel.WorkflowInstance{},
		pendingTodosByInstance: map[uint64][]*crmmodel.WorkTodo{},
	}
	if staff == nil || staff.ID == 0 {
		return workload
	}

	instancesByID := map[uint64]*crmmodel.WorkflowInstance{}
	appendInstance := func(instance *crmmodel.WorkflowInstance) {
		if instance == nil || instance.ID == 0 || instance.Status != crmmodel.ProgressStatusActive || instancesByID[instance.ID] != nil {
			return
		}
		instancesByID[instance.ID] = instance
		workload.instances = append(workload.instances, instance)
	}

	for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"assignee_staff_id": staff.ID,
		"status":            crmmodel.WorkTodoStatusPending,
	}) {
		if todo == nil || todo.WorkflowInstanceID == 0 {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
		if task == nil || task.TaskType == crmmodel.TaskTypeRule {
			continue
		}
		instance := instancesByID[todo.WorkflowInstanceID]
		if instance == nil {
			instance = crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
				"id":     todo.WorkflowInstanceID,
				"status": crmmodel.ProgressStatusActive,
			})
		}
		if instance == nil || instance.WorkflowID != todo.WorkflowID {
			continue
		}
		appendInstance(instance)
		workload.pendingTodosByInstance[instance.ID] = append(workload.pendingTodosByInstance[instance.ID], todo)
	}

	sort.SliceStable(workload.instances, func(i, j int) bool {
		if workload.instances[i].UpdatedAt.Equal(workload.instances[j].UpdatedAt) {
			return workload.instances[i].ID > workload.instances[j].ID
		}
		return workload.instances[i].UpdatedAt.After(workload.instances[j].UpdatedAt)
	})
	return workload
}

func (workload workPersonalWorkload) pendingTaskCountByWorkflow() map[uint64]int {
	counts := map[uint64]int{}
	for _, instance := range workload.instances {
		if instance == nil || instance.WorkflowID == 0 {
			continue
		}
		counts[instance.WorkflowID] += len(workload.pendingTodosByInstance[instance.ID])
	}
	return counts
}

func (workload workPersonalWorkload) pendingTaskRows(ctx context.Context, staff *WorkStaffSession, instanceID uint64) []map[string]any {
	todos := workload.pendingTodosByInstance[instanceID]
	rows := make([]map[string]any, 0, len(todos))
	for _, todo := range todos {
		if row := workTodoTaskMap(ctx, staff, todo, false); len(row) > 0 {
			rows = append(rows, row)
		}
	}
	sortWorkTodoTaskMaps(rows)
	return rows
}
