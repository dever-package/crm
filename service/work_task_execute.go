package service

import (
	"context"
	"fmt"

	crmmodel "github.com/dever-package/crm/model"
)

func executeWorkTask(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	todo, task, err := pendingTodoTaskForStaff(ctx, staff, payload)
	if err != nil {
		return nil, err
	}
	values := workActionValues(payload)
	switch task.TaskType {
	case crmmodel.TaskTypeTodo:
		return completeSimpleTodo(ctx, staff, todo, task, values)
	case crmmodel.TaskTypeForm:
		return saveOrCompleteFormTodo(ctx, staff, todo, task, values)
	case crmmodel.TaskTypeApproval:
		return completeApprovalTodo(ctx, staff, todo, task, values)
	case crmmodel.TaskTypeRule:
		return nil, fmt.Errorf("自动核验任务不能手工完成")
	default:
		return nil, fmt.Errorf("不支持的任务类型")
	}
}
