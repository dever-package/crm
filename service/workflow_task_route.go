package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func taskUsesStageActivation(task *crmmodel.Task) bool {
	if task == nil {
		return false
	}
	return strings.TrimSpace(task.ActivationMode) == "" || task.ActivationMode == crmmodel.TaskActivationStage
}

func workflowTaskApplicable(ctx context.Context, instance *crmmodel.WorkflowInstance, task *crmmodel.Task) (bool, error) {
	if instance == nil || task == nil {
		return false, fmt.Errorf("流程实例和任务不能为空")
	}
	if task.ConditionScriptID == 0 {
		return true, nil
	}
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{
		"id":     task.ConditionScriptID,
		"status": crmmodel.StatusEnabled,
	})
	if script == nil {
		return false, fmt.Errorf("任务“%s”的适用条件不存在或已停用", task.Name)
	}
	virtualTodo := &crmmodel.WorkTodo{
		LeadID:             instance.LeadID,
		CustomerID:         instance.CustomerID,
		AssetID:            instance.AssetID,
		WorkflowInstanceID: instance.ID,
		CustomerProductID:  instance.CustomerProductID,
		WorkflowID:         instance.WorkflowID,
		StageID:            instance.StageID,
		TaskID:             task.ID,
	}
	result, err := evaluateTaskRuleScript(ctx, script.Script, workRuleInput(ctx, virtualTodo, task))
	if err != nil {
		return false, fmt.Errorf("任务“%s”适用条件判断失败：%w", task.Name, err)
	}
	return result.Passed, nil
}

func activateRoutedWorkflowTask(
	ctx context.Context,
	sourceTodo *crmmodel.WorkTodo,
	targetTaskID uint64,
	requireApplicable bool,
) (*crmmodel.WorkTodo, bool, error) {
	if sourceTodo == nil || targetTaskID == 0 {
		return nil, false, fmt.Errorf("来源待办和目标任务不能为空")
	}
	instance, err := activeWorkflowInstance(ctx, sourceTodo.WorkflowInstanceID)
	if err != nil {
		return nil, false, err
	}
	if instance.StageID != sourceTodo.StageID {
		return nil, false, fmt.Errorf("来源待办已不在流程当前阶段")
	}
	target := crmmodel.NewTaskModel().Find(ctx, map[string]any{
		"id":     targetTaskID,
		"status": crmmodel.StatusEnabled,
	})
	if target == nil || target.StageID != sourceTodo.StageID {
		return nil, false, fmt.Errorf("目标任务不存在、已停用或不属于当前阶段")
	}
	applicable, err := workflowTaskApplicable(ctx, instance, target)
	if err != nil {
		return nil, false, err
	}
	if !applicable {
		if requireApplicable {
			return nil, false, fmt.Errorf("目标任务“%s”当前不适用", target.Name)
		}
		return nil, false, nil
	}
	return activateWorkflowTaskTodo(ctx, instance, target)
}

func cancelPendingRoutedWorkflowTask(ctx context.Context, sourceTodo *crmmodel.WorkTodo, targetTaskID uint64, reason string) {
	if sourceTodo == nil || targetTaskID == 0 {
		return
	}
	crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
		"workflow_instance_id": sourceTodo.WorkflowInstanceID,
		"stage_id":             sourceTodo.StageID,
		"task_id":              targetTaskID,
		"status":               crmmodel.WorkTodoStatusPending,
	}, map[string]any{
		"status":     crmmodel.WorkTodoStatusCanceled,
		"result":     strings.TrimSpace(reason),
		"updated_at": time.Now(),
	})
}

func activateWorkflowTaskTodo(ctx context.Context, instance *crmmodel.WorkflowInstance, task *crmmodel.Task) (*crmmodel.WorkTodo, bool, error) {
	if instance == nil || task == nil {
		return nil, false, fmt.Errorf("流程实例和任务不能为空")
	}
	model := crmmodel.NewWorkTodoModel()
	existing := model.Find(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"stage_id":             task.StageID,
		"task_id":              task.ID,
	})
	if existing != nil && existing.Status == crmmodel.WorkTodoStatusPending {
		return existing, false, nil
	}

	now := time.Now()
	departmentID, staffID, automatic, err := resolveTaskAssignee(ctx, instance, task)
	if err != nil {
		return nil, false, err
	}
	var dueAt *time.Time
	if task.DueDays > 0 {
		deadline := now.AddDate(0, 0, task.DueDays)
		dueAt = &deadline
	}
	data := map[string]any{
		"lead_id":                instance.LeadID,
		"customer_id":            instance.CustomerID,
		"asset_id":               instance.AssetID,
		"workflow_instance_id":   instance.ID,
		"customer_product_id":    instance.CustomerProductID,
		"workflow_id":            instance.WorkflowID,
		"stage_id":               task.StageID,
		"task_id":                task.ID,
		"assignee_department_id": departmentID,
		"assignee_staff_id":      staffID,
		"required":               task.Required,
		"status":                 crmmodel.WorkTodoStatusPending,
		"due_at":                 dueAt,
		"result":                 "",
		"completed_at":           nil,
		"updated_at":             now,
	}
	if existing == nil {
		data["created_at"] = now
		todoID := uint64(model.Insert(ctx, data))
		if todoID == 0 {
			return nil, false, fmt.Errorf("目标待办创建失败")
		}
		existing = model.Find(ctx, map[string]any{"id": todoID})
	} else {
		if model.Update(ctx, map[string]any{
			"id":     existing.ID,
			"status": existing.Status,
		}, data) == 0 {
			return nil, false, fmt.Errorf("目标待办已变化，请刷新后重试")
		}
		existing = model.Find(ctx, map[string]any{"id": existing.ID})
	}
	if existing == nil {
		return nil, false, fmt.Errorf("目标待办激活后无法读取")
	}
	if automatic {
		if err := recordAutomaticDispatch(ctx, departmentID, staffID, workflowDispatchReference{
			Source:             crmmodel.DispatchSourceTask,
			LeadID:             instance.LeadID,
			WorkflowInstanceID: instance.ID,
			WorkTodoID:         existing.ID,
		}); err != nil {
			return nil, false, err
		}
	}
	if existing.AssigneeStaffID > 0 {
		assignee := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": existing.AssigneeStaffID})
		if recordWorkTodoAssignment(ctx, nil, instance, existing, task, 0, assignee) == 0 {
			return nil, false, fmt.Errorf("目标任务分配记录创建失败")
		}
	}
	if err := syncWorkflowMeetingParticipants(ctx, instance.ID); err != nil {
		return nil, false, err
	}
	return existing, true, nil
}

func attachRoutedTaskResult(result map[string]any, routedTodo *crmmodel.WorkTodo, activated bool) {
	if result == nil || routedTodo == nil {
		return
	}
	result["routed_task_id"] = routedTodo.TaskID
	result["routed_todo_id"] = routedTodo.ID
	result["routed_task_activated"] = activated
}
