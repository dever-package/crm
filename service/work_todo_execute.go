package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

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
	if instanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId"); instanceID > 0 && instanceID != todo.WorkflowInstanceID {
		return nil, nil, fmt.Errorf("待办不属于当前流程")
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
	instance, err := activeWorkflowInstance(ctx, todo.WorkflowInstanceID)
	if err != nil || instance.WorkflowID != todo.WorkflowID || instance.StageID != todo.StageID {
		return nil, nil, fmt.Errorf("待办不属于流程当前阶段")
	}
	return todo, task, nil
}

func completeProductTodo(ctx context.Context, staff *WorkStaffSession, todo *crmmodel.WorkTodo, task *crmmodel.Task, values map[string]any) (map[string]any, error) {
	productIDs := uint64ListFromAny(values["product_ids"])
	if len(productIDs) == 0 {
		productIDs = uint64ListFromAny(values["productIds"])
	}
	if task.Required && len(productIDs) == 0 {
		return nil, fmt.Errorf("请至少选择一个产品")
	}
	customerProducts, err := SyncConfirmedCustomerProducts(ctx, todo.WorkflowInstanceID, productIDs)
	if err != nil {
		return nil, err
	}
	selectedIDs := make([]uint64, 0, len(customerProducts))
	for _, customerProduct := range customerProducts {
		if customerProduct != nil {
			selectedIDs = append(selectedIDs, customerProduct.ProductID)
		}
	}
	resultText := "已确认产品"
	if len(selectedIDs) == 0 {
		resultText = "未选择产品"
	}
	snapshot := copyMap(values)
	snapshot["product_ids"] = selectedIDs
	operationID := recordWorkTaskOperation(ctx, staff, todo, task, "confirmed", resultText, snapshot, true)
	if operationID == 0 {
		return nil, fmt.Errorf("产品确认记录创建失败")
	}
	if err := completeWorkTodo(ctx, todo, resultText); err != nil {
		return nil, err
	}
	result := workTodoExecutionResult(todo, operationID, "confirmed", false)
	result["product_ids"] = selectedIDs
	return result, nil
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
	arrivalDecision := workMeetingArrivalDecision(values)
	if progressOnly && arrivalDecision != "" {
		return nil, fmt.Errorf("到访结果请使用确认按钮提交")
	}
	keepPending := progressOnly || task.MeetingEnabled && arrivalDecision == crmmodel.MeetingArrivalNoShow
	var formInput *workFormInput
	var err error
	if keepPending {
		formInput, err = collectWorkProgressFormInput(ctx, task, values)
	} else {
		formInput, err = collectWorkFormInput(ctx, task, values)
	}
	if err != nil {
		return nil, err
	}
	operationSnapshot, hasFormChanges := buildWorkFormOperationSnapshot(ctx, todo, task, formInput, values)
	if err := saveWorkFormInput(ctx, todo.LeadID, todo.CustomerID, todo.AssetID, formInput); err != nil {
		return nil, err
	}
	if err := syncWorkCommunicationGroupFromFormTask(ctx, staff, todo, task, formInput, values); err != nil {
		return nil, err
	}
	resultValue := "submitted"
	resultText := firstText(values, "result", "remark")
	if progressOnly {
		resultValue = workResultProgress
		if resultText == "" {
			if task.MeetingEnabled {
				resultText = "已保存预约"
			} else {
				resultText = "已保存进度"
			}
		}
	} else if task.MeetingEnabled {
		if arrivalDecision == crmmodel.MeetingArrivalNoShow {
			resultValue = crmmodel.MeetingArrivalNoShow
			if resultText == "" {
				resultText = "已记录未到访：" + inputText(values[workMeetingNoShowReasonKey])
			}
		} else if resultText == "" {
			resultText = "已确认客户到访"
		}
	}
	operationContent := resultText
	if !hasFormChanges && operationContent == "" {
		operationContent = "本次未修改资料"
	}
	operationID := recordWorkTaskOperation(ctx, staff, todo, task, resultValue, operationContent, operationSnapshot, !keepPending)
	if operationID == 0 {
		return nil, fmt.Errorf("任务操作记录创建失败")
	}
	ownership := workDataOwnership{
		LeadID:             todo.LeadID,
		CustomerID:         todo.CustomerID,
		AssetID:            todo.AssetID,
		WorkflowInstanceID: todo.WorkflowInstanceID,
		CustomerProductID:  todo.CustomerProductID,
	}
	if err := saveWorkFormDataRecords(ctx, ownership, task.ID, operationID, formInput); err != nil {
		return nil, err
	}
	if task.MeetingEnabled {
		if err := syncWorkMeetingFromTaskForm(ctx, staff, todo, task, values, operationID); err != nil {
			return nil, err
		}
	}
	if !progressOnly {
		if _, err := confirmWorkMeetingArrival(ctx, staff, todo, task, values); err != nil {
			return nil, err
		}
		if !keepPending {
			if err := syncWorkCustomerFollowFromTaskForm(ctx, staff, todo, task, values, operationID); err != nil {
				return nil, err
			}
			if err := syncWorkDataFieldFinanceLedgers(ctx, staff, ownership, task.ID, operationID, formInput); err != nil {
				return nil, err
			}
		}
	}
	if keepPending {
		crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
			"id":     todo.ID,
			"status": crmmodel.WorkTodoStatusPending,
		}, map[string]any{
			"result":     resultText,
			"updated_at": time.Now(),
		})
		if err := rerunPendingRuleTodos(ctx, todo.WorkflowInstanceID, todo.StageID); err != nil {
			return nil, err
		}
		result := workTodoExecutionResult(todo, operationID, resultValue, true)
		result["kept_pending"] = true
		result["workflow_instance_id"] = todo.WorkflowInstanceID
		result["customer_product_id"] = todo.CustomerProductID
		return result, nil
	}
	if resultText == "" {
		resultText = "资料已提交"
	}
	if err := completeWorkTodo(ctx, todo, resultText); err != nil {
		return nil, err
	}
	if err := rerunPendingRuleTodos(ctx, todo.WorkflowInstanceID, todo.StageID); err != nil {
		return nil, err
	}
	result := workTodoExecutionResult(todo, operationID, resultValue, false)
	result["workflow_instance_id"] = todo.WorkflowInstanceID
	result["customer_product_id"] = todo.CustomerProductID
	return result, nil
}

func completeApprovalTodo(ctx context.Context, staff *WorkStaffSession, todo *crmmodel.WorkTodo, task *crmmodel.Task, values map[string]any) (map[string]any, error) {
	decision := strings.ToLower(firstText(values, "approval_result", "approvalResult", "decision", "result"))
	opinion := firstText(values, "opinion", "reason", "remark", "content")
	switch decision {
	case "approved", "approve", "pass", "passed":
	case "rejected", "reject":
	default:
		return nil, fmt.Errorf("请选择通过或驳回")
	}
	if approvalOpinionRequired(task, decision) && opinion == "" {
		return nil, fmt.Errorf("请填写审核意见")
	}

	var executionResult map[string]any
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		decisionValue := "approved"
		if decision == "rejected" || decision == "reject" {
			decisionValue = "rejected"
		}
		formInput, operationSnapshot, err := prepareApprovalFormSubmission(txCtx, todo, task, values, decisionValue, opinion)
		if err != nil {
			return err
		}
		if decisionValue == "rejected" {
			executionResult, err = rejectApprovalTodo(txCtx, staff, todo, task, operationSnapshot, formInput, opinion)
		} else {
			executionResult, err = approveApprovalTodo(txCtx, staff, todo, task, operationSnapshot, formInput, opinion)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return executionResult, nil
}

func prepareApprovalFormSubmission(
	ctx context.Context,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	values map[string]any,
	decision string,
	opinion string,
) (*workFormInput, map[string]any, error) {
	operationSnapshot := copyMap(values)
	operationSnapshot["approval_result"] = decision
	operationSnapshot["opinion"] = opinion
	if task.FormID == 0 || decision == "rejected" && !task.RejectSubmitForm {
		return nil, operationSnapshot, nil
	}
	formInput, err := collectWorkFormInput(ctx, task, values)
	if err != nil {
		return nil, nil, err
	}
	operationSnapshot, _ = buildWorkFormOperationSnapshot(ctx, todo, task, formInput, values)
	operationSnapshot["approval_result"] = decision
	operationSnapshot["opinion"] = opinion
	if err := saveWorkFormInput(ctx, todo.LeadID, todo.CustomerID, todo.AssetID, formInput); err != nil {
		return nil, nil, err
	}
	return formInput, operationSnapshot, nil
}

func rejectApprovalTodo(
	ctx context.Context,
	staff *WorkStaffSession,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	operationSnapshot map[string]any,
	formInput *workFormInput,
	opinion string,
) (map[string]any, error) {
	rejectAction := configuredTaskRejectAction(task)
	operationID := recordWorkTaskOperation(ctx, staff, todo, task, "rejected", opinion, operationSnapshot, rejectAction != crmmodel.TaskRejectStay)
	if operationID == 0 {
		return nil, fmt.Errorf("审核记录创建失败")
	}
	if err := saveApprovalFormData(ctx, staff, todo, task, operationID, formInput); err != nil {
		return nil, err
	}
	switch rejectAction {
	case crmmodel.TaskRejectStay:
		crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
			"id":     todo.ID,
			"status": crmmodel.WorkTodoStatusPending,
		}, map[string]any{
			"result":     opinion,
			"updated_at": time.Now(),
		})
		return workTodoExecutionResult(todo, operationID, "rejected", false), nil
	case crmmodel.TaskRejectTerminate:
		resultText := strings.TrimSpace(opinion)
		if resultText == "" {
			resultText = "审核不通过"
		}
		if err := completeWorkTodo(ctx, todo, resultText); err != nil {
			return nil, err
		}
		instance, err := activeWorkflowInstance(ctx, todo.WorkflowInstanceID)
		if err != nil {
			return nil, err
		}
		if err := terminateActiveWorkflowInstance(ctx, staff, instance, task.Name+"不通过："+resultText); err != nil {
			return nil, err
		}
		result := workTodoExecutionResult(todo, operationID, "rejected", false)
		result["workflow_status"] = crmmodel.ProgressStatusTerminated
		result["workflow_terminated"] = true
		return result, nil
	case crmmodel.TaskRejectRoute:
		if task.RejectTargetTaskID == 0 {
			return nil, fmt.Errorf("审核任务未配置驳回目标")
		}
		cancelPendingRoutedWorkflowTask(ctx, todo, task.CompleteTargetTaskID, "前置审核驳回，等待复核通过后重新创建")
	default:
		return nil, fmt.Errorf("审核任务驳回处理方式无效")
	}
	if err := completeWorkTodo(ctx, todo, "审核驳回："+opinion); err != nil {
		return nil, err
	}
	routedTodo, activated, err := activateRoutedWorkflowTask(ctx, todo, task.RejectTargetTaskID, true)
	if err != nil {
		return nil, err
	}
	result := workTodoExecutionResult(todo, operationID, "rejected", false)
	attachRoutedTaskResult(result, routedTodo, activated)
	return result, nil
}

func configuredTaskRejectAction(task *crmmodel.Task) string {
	if task == nil {
		return crmmodel.TaskRejectStay
	}
	switch task.RejectAction {
	case crmmodel.TaskRejectStay, crmmodel.TaskRejectRoute, crmmodel.TaskRejectTerminate:
		return task.RejectAction
	default:
		if task.RejectTargetTaskID > 0 {
			return crmmodel.TaskRejectRoute
		}
		return crmmodel.TaskRejectStay
	}
}

func approvalOpinionRequired(task *crmmodel.Task, decision string) bool {
	requirement := crmmodel.TaskOpinionRejectRequired
	if task != nil && strings.TrimSpace(task.OpinionRequirement) != "" {
		requirement = task.OpinionRequirement
	}
	switch requirement {
	case crmmodel.TaskOpinionRequired:
		return true
	case crmmodel.TaskOpinionOptional:
		return false
	default:
		return decision == "rejected" || decision == "reject"
	}
}

func approveApprovalTodo(
	ctx context.Context,
	staff *WorkStaffSession,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	operationSnapshot map[string]any,
	formInput *workFormInput,
	opinion string,
) (map[string]any, error) {
	operationID := recordWorkTaskOperation(ctx, staff, todo, task, "approved", opinion, operationSnapshot, true)
	if operationID == 0 {
		return nil, fmt.Errorf("审核记录创建失败")
	}
	if err := saveApprovalFormData(ctx, staff, todo, task, operationID, formInput); err != nil {
		return nil, err
	}
	resultText := opinion
	if resultText == "" {
		resultText = "审核通过"
	}
	if err := completeWorkTodo(ctx, todo, resultText); err != nil {
		return nil, err
	}
	return workTodoExecutionResult(todo, operationID, "approved", false), nil
}

func saveApprovalFormData(
	ctx context.Context,
	staff *WorkStaffSession,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	operationID uint64,
	formInput *workFormInput,
) error {
	if formInput == nil {
		return nil
	}
	ownership := workDataOwnership{
		LeadID:             todo.LeadID,
		CustomerID:         todo.CustomerID,
		AssetID:            todo.AssetID,
		WorkflowInstanceID: todo.WorkflowInstanceID,
		CustomerProductID:  todo.CustomerProductID,
	}
	if err := saveWorkFormDataRecords(ctx, ownership, task.ID, operationID, formInput); err != nil {
		return err
	}
	return syncWorkDataFieldFinanceLedgers(ctx, staff, ownership, task.ID, operationID, formInput)
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

func rerunPendingRuleTodos(ctx context.Context, workflowInstanceID, stageID uint64) error {
	rows := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"workflow_instance_id": workflowInstanceID,
		"stage_id":             stageID,
		"status":               crmmodel.WorkTodoStatusPending,
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
		"workflow_instance_id": todo.WorkflowInstanceID,
		"stage_id":             todo.StageID,
		"required":             true,
		"status":               crmmodel.WorkTodoStatusPending,
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
	ruleInput := workRuleInput(ctx, todo, task)
	isElevenDimensionRule := isElevenDimensionRuleTask(ctx, task)
	result, err := NewRuleService().EvaluateTask(ctx, task, ruleInput)
	if err != nil {
		result = TaskRuleResult{Passed: false, Reason: err.Error()}
	}
	inputFields := elevenDimensionRuleInputSnapshot(ruleInput)
	if !result.Passed {
		reason := taskRuleResultText(result)
		if reason == "" {
			reason = "自动核验未通过"
		}
		operationSnapshot := map[string]any{"raw_result": result.RawResult}
		if isElevenDimensionRule {
			operationSnapshot["input_fields"] = inputFields
		}
		recordWorkTaskOperation(ctx, nil, todo, task, "failed", reason, operationSnapshot, false)
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
	operationSnapshot := map[string]any{
		"raw_result":    result.RawResult,
		"duration_ms":   result.DurationMS,
		"fields":        result.OutputFields,
		"product_codes": result.ProductCodes,
	}
	if isElevenDimensionRule {
		operationSnapshot["input_fields"] = inputFields
	}
	operationID := recordWorkTaskOperation(ctx, nil, todo, task, "passed", resultText, operationSnapshot, true)
	if operationID == 0 {
		return fmt.Errorf("自动核验记录创建失败")
	}
	if err := applyTaskRuleOutputs(ctx, todo, task, operationID, result); err != nil {
		return err
	}
	return completeWorkTodo(ctx, todo, resultText)
}

func applyTaskRuleOutputs(ctx context.Context, todo *crmmodel.WorkTodo, task *crmmodel.Task, operationID uint64, result TaskRuleResult) error {
	if todo == nil || task == nil {
		return fmt.Errorf("自动核验任务不存在")
	}
	formInput := emptyWorkFormInput()
	for fieldKey, value := range operationBackedElevenDimensionFields(operationID, result.OutputFields) {
		fieldKey = strings.TrimSpace(fieldKey)
		if fieldKey == "" {
			continue
		}
		field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
			"field_key": fieldKey,
			"status":    crmmodel.StatusEnabled,
		})
		if field == nil || field.FieldType == "group" {
			return fmt.Errorf("规则输出字段不存在或不可写：%s", fieldKey)
		}
		template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
			"id":     field.DataTemplateID,
			"status": crmmodel.StatusEnabled,
		})
		if template == nil {
			return fmt.Errorf("规则输出字段没有可写的数据模板：%s", fieldKey)
		}
		if todo.LeadID > 0 && template.CateID != crmmodel.LeadDataTemplateCateID {
			return fmt.Errorf("线索流程规则只能输出线索信息字段：%s", fieldKey)
		}
		if todo.LeadID == 0 && template.CateID == crmmodel.LeadDataTemplateCateID {
			return fmt.Errorf("客户资产流程规则不能输出线索信息字段：%s", fieldKey)
		}
		formField := &crmmodel.FormField{
			DataTemplateCateID: template.CateID,
			DataTemplateID:     template.ID,
			DataFieldID:        field.ID,
		}
		records := workFormRecordBucket(ctx, formInput, formField)
		if records[template.ID] == nil {
			records[template.ID] = map[string]any{}
		}
		records[template.ID][fmt.Sprintf("%d", field.ID)] = value
	}
	if err := saveWorkFormInput(ctx, todo.LeadID, todo.CustomerID, todo.AssetID, formInput); err != nil {
		return err
	}
	if err := saveWorkFormDataRecords(ctx, workDataOwnership{
		LeadID:             todo.LeadID,
		CustomerID:         todo.CustomerID,
		AssetID:            todo.AssetID,
		WorkflowInstanceID: todo.WorkflowInstanceID,
		CustomerProductID:  todo.CustomerProductID,
	}, task.ID, operationID, formInput); err != nil {
		return err
	}
	if err := syncWorkDataFieldFinanceLedgers(ctx, nil, workDataOwnership{
		LeadID:             todo.LeadID,
		CustomerID:         todo.CustomerID,
		AssetID:            todo.AssetID,
		WorkflowInstanceID: todo.WorkflowInstanceID,
		CustomerProductID:  todo.CustomerProductID,
	}, task.ID, operationID, formInput); err != nil {
		return err
	}
	if result.ProductCodes != nil && todo.CustomerID > 0 && todo.AssetID > 0 {
		if err := SyncCandidateCustomerProducts(ctx, todo.WorkflowInstanceID, result.ProductCodes); err != nil {
			return err
		}
	}
	return nil
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
	lead := map[string]any{}
	leadModel := crmmodel.NewLeadModel()
	if source := leadModel.Find(ctx, map[string]any{"id": todo.LeadID}); source != nil {
		lead = mapFromAny(leadModel.FindMap(ctx, map[string]any{"id": source.ID}))
		lead["fields"] = workLeadRuleFieldValues(ctx, source)
		lead["data_values"] = workLeadDataValues(source)
	}
	customer := map[string]any{}
	asset := map[string]any{}
	if todo.CustomerID > 0 {
		customer = mapFromAny(crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": todo.CustomerID}))
		customer["fields"] = workCustomerFieldValues(ctx, todo.CustomerID)
	}
	if todo.CustomerID > 0 && todo.AssetID > 0 {
		asset = mapFromAny(crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{
			"id":          todo.AssetID,
			"customer_id": todo.CustomerID,
		}))
		asset["fields"] = workAssetFieldValues(ctx, todo.CustomerID, todo.AssetID)
		asset["finance"] = workAssetFinanceRuleValues(ctx, todo.AssetID)
	}
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
		"lead":     lead,
		"customer": customer,
		"assets":   workRuleAssets(ctx, todo.CustomerID),
		"current": map[string]any{
			"workflow_instance_id": todo.WorkflowInstanceID,
			"customer_product_id":  todo.CustomerProductID,
			"workflow_id":          todo.WorkflowID,
			"workflow_name":        workflowName,
			"stage_id":             todo.StageID,
			"stage_name":           stageName,
			"asset_id":             todo.AssetID,
			"asset":                asset,
		},
	}
}

func workAssetFinanceRuleValues(ctx context.Context, assetID uint64) map[string]any {
	result := map[string]any{}
	if assetID == 0 {
		return result
	}
	for _, ledger := range crmmodel.NewFinanceLedgerModel().Select(ctx, map[string]any{"asset_id": assetID}) {
		if ledger == nil {
			continue
		}
		code := strings.TrimSpace(ledger.FinanceTypeCode)
		if code == "" {
			code = fmt.Sprintf("finance_type_%d", ledger.FinanceTypeID)
		}
		current := mapFromAny(result[code])
		current["amount"] = numericValue(current["amount"]) + ledger.Amount
		current["count"] = inputInt(current["count"]) + 1
		current["direction"] = ledger.Direction
		result[code] = current
	}
	return result
}

func workLeadRuleFieldValues(ctx context.Context, lead *crmmodel.Lead) map[string]any {
	values := map[string]any{}
	if lead == nil {
		return values
	}
	for _, template := range workLeadTemplateRows(ctx) {
		fields := workDataTemplateFieldsByID(ctx, inputUint64(template["id"]))
		for rawKey, value := range workLeadDataValues(lead) {
			field := fields[inputUint64(strings.TrimPrefix(rawKey, "data:"))]
			if field != nil && strings.TrimSpace(field.FieldKey) != "" {
				values[field.FieldKey] = value
			}
		}
	}
	return values
}

func workTodoExecutionResult(todo *crmmodel.WorkTodo, operationID uint64, resultValue string, progress bool) map[string]any {
	return map[string]any{
		"todo_id":              todo.ID,
		"task_id":              todo.TaskID,
		"lead_id":              todo.LeadID,
		"customer_id":          todo.CustomerID,
		"asset_id":             todo.AssetID,
		"workflow_instance_id": todo.WorkflowInstanceID,
		"customer_product_id":  todo.CustomerProductID,
		"workflow_id":          todo.WorkflowID,
		"stage_id":             todo.StageID,
		"operation_log_id":     operationID,
		"result_value":         resultValue,
		"progress":             progress,
		"saved":                true,
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

func canOperateCurrentState(staff *WorkStaffSession, state *crmmodel.WorkflowInstance) bool {
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
