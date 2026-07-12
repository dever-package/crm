package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

type workOperationCompletion struct {
	customerID       uint64
	assetID          uint64
	businessObjectID uint64
	operationID      uint64
	task             *crmmodel.Task
	formInput        *workFormInput
	resultValue      string
	todoID           uint64
}

func pendingTodoTaskForStaff(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (*crmmodel.WorkTodo, *crmmodel.Task, error) {
	if staff == nil || staff.ID == 0 {
		return nil, nil, fmt.Errorf("请先登录")
	}
	todoID := firstUint64(payload, "todo_id", "todoId")
	if todoID == 0 {
		return nil, nil, fmt.Errorf("待办不能为空")
	}
	todo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
		"id":     todoID,
		"status": crmmodel.WorkTodoStatusPending,
	})
	if todo == nil {
		return nil, nil, fmt.Errorf("待办不存在或已完成")
	}
	if customerID := firstUint64(payload, "customer_id", "customerId"); customerID > 0 && customerID != todo.CustomerID {
		return nil, nil, fmt.Errorf("待办不属于当前客户")
	}
	if assetID := firstUint64(payload, "asset_id", "assetId"); assetID > 0 && assetID != todo.AssetID {
		return nil, nil, fmt.Errorf("待办不属于当前资产")
	}
	if taskID := firstUint64(payload, "task_id", "taskId"); taskID > 0 && taskID != todo.TaskID {
		return nil, nil, fmt.Errorf("待办与任务不匹配")
	}
	if !canOperateWorkTodo(staff, todo) {
		return nil, nil, fmt.Errorf("当前人员无权执行该待办")
	}
	task := crmmodel.NewTaskModel().Find(ctx, map[string]any{
		"id":       todo.TaskID,
		"stage_id": todo.StageID,
	})
	if task == nil {
		return nil, nil, fmt.Errorf("任务配置不存在")
	}
	progress := currentWorkCustomerStage(ctx, todo.CustomerID, todo.AssetID)
	if progress == nil || progress.Status != crmmodel.ProgressStatusActive ||
		progress.WorkflowID != todo.WorkflowID || progress.StageID != todo.StageID {
		return nil, nil, fmt.Errorf("待办不属于资产当前阶段")
	}
	return todo, task, nil
}

func completeSimpleTodo(ctx context.Context, staff *WorkStaffSession, todo *crmmodel.WorkTodo, task *crmmodel.Task, values map[string]any) (map[string]any, error) {
	result := firstText(values, "result", "remark", "content")
	if result == "" {
		return nil, fmt.Errorf("请填写办理结果")
	}
	operationID := recordWorkTaskOperation(ctx, staff, todo, task, "completed", result, values, true)
	if operationID == 0 {
		return nil, fmt.Errorf("任务操作记录创建失败")
	}
	if err := completeWorkTodo(ctx, todo, result); err != nil {
		return nil, err
	}
	return workTodoExecutionResult(todo, operationID, "completed", false), nil
}

func saveOrCompleteFormTodo(ctx context.Context, staff *WorkStaffSession, todo *crmmodel.WorkTodo, task *crmmodel.Task, values map[string]any) (map[string]any, error) {
	if task.FormID == 0 {
		return nil, fmt.Errorf("资料任务未配置表单")
	}
	progressOnly := workSubmitIsProgress(values)
	var formInput *workFormInput
	var err error
	if progressOnly {
		formInput, err = collectWorkProgressFormInput(ctx, task, values)
	} else {
		formInput, err = collectWorkFormInput(ctx, task, values)
	}
	if err != nil {
		return nil, err
	}
	if err := saveWorkFormInput(ctx, todo.CustomerID, todo.AssetID, formInput); err != nil {
		return nil, err
	}
	businessObjectID := firstUint64(values, "business_object_id", "businessObjectId")
	if businessObjectID > 0 && !workCustomerOwnsBusinessObject(ctx, todo.CustomerID, todo.AssetID, businessObjectID) {
		return nil, fmt.Errorf("业务对象不存在")
	}
	businessObjectID, _, err = ensureWorkFormBusinessObject(ctx, staff, todo.CustomerID, todo.AssetID, businessObjectID, formInput)
	if err != nil {
		return nil, err
	}
	resultValue := "submitted"
	resultText := firstText(values, "result", "remark")
	if progressOnly {
		resultValue = workResultProgress
		if resultText == "" {
			resultText = "已保存进度"
		}
	}
	operationID := recordWorkTaskOperation(ctx, staff, todo, task, resultValue, resultText, values, !progressOnly)
	if operationID == 0 {
		return nil, fmt.Errorf("任务操作记录创建失败")
	}
	saveWorkFormObjectDataRecords(ctx, todo.CustomerID, todo.AssetID, businessObjectID, task.ID, operationID, formInput)
	syncWorkFinanceLedgers(ctx, staff, workOperationCompletion{
		customerID:       todo.CustomerID,
		assetID:          todo.AssetID,
		businessObjectID: businessObjectID,
		operationID:      operationID,
		task:             task,
		formInput:        formInput,
		resultValue:      resultValue,
		todoID:           todo.ID,
	})
	if progressOnly {
		crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
			"id":     todo.ID,
			"status": crmmodel.WorkTodoStatusPending,
		}, map[string]any{
			"result":     resultText,
			"updated_at": time.Now(),
		})
		if err := rerunPendingRuleTodos(ctx, todo.AssetID, todo.StageID); err != nil {
			return nil, err
		}
		result := workTodoExecutionResult(todo, operationID, resultValue, true)
		result["business_object_id"] = businessObjectID
		return result, nil
	}
	if resultText == "" {
		resultText = "资料已提交"
	}
	if err := completeWorkTodo(ctx, todo, resultText); err != nil {
		return nil, err
	}
	if err := rerunPendingRuleTodos(ctx, todo.AssetID, todo.StageID); err != nil {
		return nil, err
	}
	result := workTodoExecutionResult(todo, operationID, resultValue, false)
	result["business_object_id"] = businessObjectID
	return result, nil
}

func completeApprovalTodo(ctx context.Context, staff *WorkStaffSession, todo *crmmodel.WorkTodo, task *crmmodel.Task, values map[string]any) (map[string]any, error) {
	decision := strings.ToLower(firstText(values, "approval_result", "approvalResult", "decision", "result"))
	opinion := firstText(values, "opinion", "reason", "remark", "content")
	switch decision {
	case "approved", "approve", "pass", "passed":
		operationID := recordWorkTaskOperation(ctx, staff, todo, task, "approved", opinion, values, true)
		if operationID == 0 {
			return nil, fmt.Errorf("审核记录创建失败")
		}
		result := opinion
		if result == "" {
			result = "审核通过"
		}
		if err := completeWorkTodo(ctx, todo, result); err != nil {
			return nil, err
		}
		return workTodoExecutionResult(todo, operationID, "approved", false), nil
	case "rejected", "reject":
		if opinion == "" {
			return nil, fmt.Errorf("请填写驳回原因")
		}
		operationID := recordWorkTaskOperation(ctx, staff, todo, task, "rejected", opinion, values, false)
		if operationID == 0 {
			return nil, fmt.Errorf("审核记录创建失败")
		}
		crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
			"id":     todo.ID,
			"status": crmmodel.WorkTodoStatusPending,
		}, map[string]any{
			"result":     opinion,
			"updated_at": time.Now(),
		})
		return workTodoExecutionResult(todo, operationID, "rejected", false), nil
	default:
		return nil, fmt.Errorf("请选择通过或驳回")
	}
}

func completeWorkTodo(ctx context.Context, todo *crmmodel.WorkTodo, result string) error {
	now := time.Now()
	if crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
		"id":     todo.ID,
		"status": crmmodel.WorkTodoStatusPending,
	}, map[string]any{
		"status":       crmmodel.WorkTodoStatusDone,
		"result":       result,
		"completed_at": now,
		"updated_at":   now,
	}) == 0 {
		return fmt.Errorf("待办已被处理，请刷新后重试")
	}
	return nil
}

func rerunPendingRuleTodos(ctx context.Context, assetID, stageID uint64) error {
	rows := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"asset_id": assetID,
		"stage_id": stageID,
		"status":   crmmodel.WorkTodoStatusPending,
	})
	for _, todo := range rows {
		if todo == nil {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
		if task == nil || task.TaskType != crmmodel.TaskTypeRule {
			continue
		}
		if !workRuleTodoReady(ctx, todo, task) {
			continue
		}
		if err := executePendingRuleTodo(ctx, todo, task); err != nil {
			return err
		}
	}
	return nil
}

func workRuleTodoReady(ctx context.Context, todo *crmmodel.WorkTodo, task *crmmodel.Task) bool {
	if todo == nil || task == nil || task.TaskType != crmmodel.TaskTypeRule {
		return false
	}
	pendingTodos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"asset_id": todo.AssetID,
		"stage_id": todo.StageID,
		"required": true,
		"status":   crmmodel.WorkTodoStatusPending,
	})
	for _, pendingTodo := range pendingTodos {
		if pendingTodo == nil || pendingTodo.ID == todo.ID {
			continue
		}
		pendingTask := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": pendingTodo.TaskID})
		if pendingTask != nil && taskComesBefore(pendingTask, task) {
			return false
		}
	}
	return true
}

func taskComesBefore(current *crmmodel.Task, target *crmmodel.Task) bool {
	if current == nil || target == nil {
		return false
	}
	return current.Sort < target.Sort || current.Sort == target.Sort && current.ID < target.ID
}

func executePendingRuleTodo(ctx context.Context, todo *crmmodel.WorkTodo, task *crmmodel.Task) error {
	if todo == nil || task == nil || task.TaskType != crmmodel.TaskTypeRule {
		return nil
	}
	result, err := NewRuleService().EvaluateTask(ctx, task, workRuleInput(ctx, todo, task))
	if err != nil {
		result = TaskRuleResult{Passed: false, Reason: err.Error()}
	}
	if !result.Passed {
		reason := taskRuleResultText(result)
		if reason == "" {
			reason = "自动核验未通过"
		}
		recordWorkTaskOperation(ctx, nil, todo, task, "failed", reason, map[string]any{"raw_result": result.RawResult}, false)
		crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
			"id":     todo.ID,
			"status": crmmodel.WorkTodoStatusPending,
		}, map[string]any{
			"result":     reason,
			"updated_at": time.Now(),
		})
		return nil
	}
	resultText := taskRuleResultText(result)
	if resultText == "" {
		resultText = "自动核验通过"
	}
	operationID := recordWorkTaskOperation(ctx, nil, todo, task, "passed", resultText, map[string]any{
		"raw_result":  result.RawResult,
		"duration_ms": result.DurationMS,
	}, true)
	if operationID == 0 {
		return fmt.Errorf("自动核验记录创建失败")
	}
	return completeWorkTodo(ctx, todo, resultText)
}

func taskRuleResultText(result TaskRuleResult) string {
	value := strings.TrimSpace(result.Value)
	reason := strings.TrimSpace(result.Reason)
	switch {
	case value != "" && reason != "":
		return value + "：" + reason
	case value != "":
		return value
	default:
		return reason
	}
}

func workRuleInput(ctx context.Context, todo *crmmodel.WorkTodo, task *crmmodel.Task) map[string]any {
	customer := crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": todo.CustomerID})
	customer["fields"] = workCustomerFieldValues(ctx, todo.CustomerID)
	asset := crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{
		"id":          todo.AssetID,
		"customer_id": todo.CustomerID,
	})
	asset["fields"] = workAssetFieldValues(ctx, todo.CustomerID, todo.AssetID)
	workflowName := ""
	if workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": todo.WorkflowID}); workflow != nil {
		workflowName = workflow.Name
	}
	stageName := ""
	if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": todo.StageID}); stage != nil {
		stageName = stage.Name
	}
	return map[string]any{
		"task": map[string]any{
			"id":        task.ID,
			"name":      task.Name,
			"task_type": task.TaskType,
		},
		"customer": customer,
		"assets":   workRuleAssets(ctx, todo.CustomerID),
		"current": map[string]any{
			"workflow_id":   todo.WorkflowID,
			"workflow_name": workflowName,
			"stage_id":      todo.StageID,
			"stage_name":    stageName,
			"asset_id":      todo.AssetID,
			"asset":         asset,
		},
	}
}

func workTodoExecutionResult(todo *crmmodel.WorkTodo, operationID uint64, resultValue string, progress bool) map[string]any {
	return map[string]any{
		"todo_id":          todo.ID,
		"task_id":          todo.TaskID,
		"customer_id":      todo.CustomerID,
		"asset_id":         todo.AssetID,
		"workflow_id":      todo.WorkflowID,
		"stage_id":         todo.StageID,
		"operation_log_id": operationID,
		"result_value":     resultValue,
		"progress":         progress,
		"saved":            true,
	}
}

func canOperateWorkTodo(staff *WorkStaffSession, todo *crmmodel.WorkTodo) bool {
	if staff == nil || todo == nil {
		return false
	}
	if todo.AssigneeStaffID > 0 {
		return todo.AssigneeStaffID == staff.ID
	}
	return false
}

func canOperateCurrentState(staff *WorkStaffSession, state *crmmodel.CustomerStage) bool {
	if staff == nil || state == nil {
		return false
	}
	return state.OwnerStaffID > 0 && state.OwnerStaffID == staff.ID
}

func workSubmitMode(values map[string]any) string {
	if firstText(values, "submit_mode", "submitMode", "mode") == workSubmitModeProgress {
		return workSubmitModeProgress
	}
	return workSubmitModeComplete
}

func workSubmitIsProgress(values map[string]any) bool {
	return workSubmitMode(values) == workSubmitModeProgress
}
