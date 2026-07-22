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
	var result map[string]any
	switch task.TaskType {
	case crmmodel.TaskTypeTodo:
		result, err = completeSimpleTodo(ctx, staff, todo, task, values)
	case crmmodel.TaskTypeForm:
		result, err = saveOrCompleteFormTodo(ctx, staff, todo, task, values)
	case crmmodel.TaskTypeApproval:
		result, err = completeApprovalTodo(ctx, staff, todo, task, values)
	case crmmodel.TaskTypeProduct:
		result, err = completeProductTodo(ctx, staff, todo, task, values)
	case crmmodel.TaskTypeRule:
		return nil, fmt.Errorf("自动核验任务不能手工完成")
	default:
		return nil, fmt.Errorf("不支持的任务类型")
	}
	if err != nil || workSubmitIsProgress(values) || booleanFromAny(result["kept_pending"]) || inputText(result["result_value"]) == "rejected" || task.CompleteTargetTaskID == 0 {
		return result, err
	}
	routedTodo, activated, err := activateRoutedWorkflowTask(ctx, todo, task.CompleteTargetTaskID, false)
	if err != nil {
		return nil, err
	}
	attachRoutedTaskResult(result, routedTodo, activated)
	return result, nil
}
