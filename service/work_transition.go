package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	crmmodel "my/package/crm/model"
	fronteval "my/package/front/service/eval"
)

func runWorkAutoTriggers(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, sourceTask *crmmodel.Task, resultValue string, enteredStageCode string, runtime *workExecutionRuntime) {
	if staff == nil || sourceTask == nil || customerID == 0 || runtime == nil || runtime.depth >= maxWorkAutoTriggerDepth {
		return
	}
	for _, task := range workAfterTaskTriggers(ctx, sourceTask.ID) {
		executeAutoWorkTask(ctx, staff, customerID, assetID, task, runtime)
	}
	if enteredStageCode == "" {
		return
	}
	for _, task := range workStageEnterTriggers(ctx, enteredStageCode) {
		executeAutoWorkTask(ctx, staff, customerID, assetID, task, runtime)
	}
}

func executeAutoWorkTask(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, task *crmmodel.Task, runtime *workExecutionRuntime) {
	if task == nil || !crmmodel.TaskTypeSupportsAutoTrigger(task.TaskType) {
		return
	}
	if workAutoTaskAlreadySucceeded(ctx, customerID, assetID, task.ID) {
		return
	}
	if !beginWorkTaskExecution(runtime, customerID, assetID, task.ID) {
		return
	}
	defer endWorkTaskExecution(runtime, customerID, assetID, task.ID)
	if _, err := executeAutoWorkTaskByType(ctx, staff, customerID, assetID, task, runtime); err != nil {
		insertWorkAutoTaskFailureLog(ctx, staff, task, customerID, assetID, err)
	}
}

func workAutoTaskAlreadySucceeded(ctx context.Context, customerID uint64, assetID uint64, taskID uint64) bool {
	if customerID == 0 || taskID == 0 {
		return false
	}
	operation := crmmodel.NewOperationLogModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"task_id":     taskID,
	})
	return operation != nil && operation.ResultValue != workResultAutoFailed
}

func executeAutoWorkTaskByType(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, task *crmmodel.Task, runtime *workExecutionRuntime) (map[string]any, error) {
	switch task.TaskType {
	case crmmodel.TaskTypeDecision:
		if task.ScriptID == 0 {
			return nil, fmt.Errorf("自动决策任务必须配置脚本规则")
		}
		return executeDecisionCustomerTask(ctx, staff, task, customerID, assetID, map[string]any{}, runtime)
	case crmmodel.TaskTypeAssign:
		values, err := workAutoAssignValues(ctx, task)
		if err != nil {
			return nil, err
		}
		return executeAssignCustomerTask(ctx, staff, task, customerID, assetID, values, runtime)
	case crmmodel.TaskTypeCollaborate:
		return executeCollaborateCustomerTask(ctx, staff, task, customerID, assetID, map[string]any{}, runtime)
	default:
		return nil, fmt.Errorf("任务动作不支持自动触发")
	}
}

func workAutoAssignValues(ctx context.Context, task *crmmodel.Task) (map[string]any, error) {
	if task == nil {
		return nil, fmt.Errorf("任务不存在")
	}
	config := mapFromAny(task.ConfigJSON)
	departmentID := inputUint64(config["auto_assign_department_id"])
	staffID := inputUint64(config["auto_assign_staff_id"])
	if normalizeWorkAssignMode(inputText(config["assign_mode"])) == crmmodel.TaskAssignModeDepartment {
		staffID = 0
	}
	if departmentID == 0 {
		return nil, fmt.Errorf("自动分配任务必须配置自动分配部门")
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": departmentID, "status": crmmodel.StatusEnabled})
	if department == nil {
		return nil, fmt.Errorf("自动分配部门不存在或已停用")
	}
	allowedDepartmentIDs := uint64ListFromAny(config["assign_department_ids"])
	if len(allowedDepartmentIDs) > 0 && !uint64SetContains(allowedDepartmentIDs, departmentID) {
		return nil, fmt.Errorf("自动分配部门不在当前任务可选范围内")
	}
	if staffID > 0 {
		staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID, "status": crmmodel.StatusEnabled})
		if staff == nil {
			return nil, fmt.Errorf("自动分配人员不存在或已停用")
		}
		if staff.DepartmentID != departmentID {
			return nil, fmt.Errorf("自动分配人员不属于自动分配部门")
		}
	}
	return map[string]any{
		"department_id": departmentID,
		"staff_id":      staffID,
	}, nil
}

func workAfterTaskTriggers(ctx context.Context, taskID uint64) []*crmmodel.Task {
	if taskID == 0 {
		return nil
	}
	return crmmodel.NewTaskModel().Select(ctx, map[string]any{
		"trigger_type":    crmmodel.TaskTriggerAfterTask,
		"trigger_task_id": taskID,
		"status":          crmmodel.StatusEnabled,
	})
}

func workStageEnterTriggers(ctx context.Context, stageCode string) []*crmmodel.Task {
	if stageCode == "" {
		return nil
	}
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{
		"code":   stageCode,
		"status": crmmodel.StatusEnabled,
	})
	if stage == nil {
		return nil
	}
	return crmmodel.NewTaskModel().Select(ctx, map[string]any{
		"stage_id":     stage.ID,
		"trigger_type": crmmodel.TaskTriggerStageEnter,
		"status":       crmmodel.StatusEnabled,
	})
}

type workDecisionResult struct {
	Value      string `json:"value"`
	Reason     string `json:"reason,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

func resolveWorkDecisionResult(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage, values map[string]any) (workDecisionResult, error) {
	if task != nil && task.TriggerType == crmmodel.TaskTriggerManual {
		return resolveManualWorkDecisionResult(values)
	}
	if task == nil || task.ScriptID == 0 {
		return workDecisionResult{}, fmt.Errorf("自动决策任务必须配置脚本规则")
	}
	return executeWorkDecisionScript(ctx, staff, task, customerID, assetID, state)
}

func resolveManualWorkDecisionResult(values map[string]any) (workDecisionResult, error) {
	resultValue := firstText(values, "decision_result", "result_value", "value")
	if resultValue == "" {
		return workDecisionResult{}, fmt.Errorf("请选择决策结果")
	}
	return workDecisionResult{
		Value:  resultValue,
		Reason: firstText(values, "decision_reason", "reason"),
	}, nil
}

func executeWorkDecisionScript(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage) (workDecisionResult, error) {
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{"id": task.ScriptID, "status": crmmodel.StatusEnabled})
	if script == nil {
		return workDecisionResult{}, fmt.Errorf("自动决策脚本不存在或已停用")
	}
	result, err := fronteval.Run(ctx, fronteval.Request{
		Language: fronteval.LanguageJavaScript,
		Script:   script.Script,
		Entry:    fronteval.DefaultEntry,
		Input:    workDecisionInput(ctx, staff, task, customerID, assetID, state),
		Config:   mapFromAny(task.ConfigJSON),
	})
	if err != nil {
		return workDecisionResult{}, err
	}
	return normalizeWorkDecisionResult(result.Value, result.DurationMS)
}

func normalizeWorkDecisionResult(value any, durationMS int64) (workDecisionResult, error) {
	payload := mapFromAny(value)
	resultValue := inputText(payload["value"])
	if resultValue == "" {
		return workDecisionResult{}, fmt.Errorf("自动决策脚本必须返回 { value: \"T几\" }")
	}
	return workDecisionResult{
		Value:      resultValue,
		Reason:     inputText(payload["reason"]),
		DurationMS: durationMS,
	}, nil
}

func workDecisionInput(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage) map[string]any {
	customer := crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
	customer["fields"] = workCustomerFieldValues(ctx, customerID)
	return map[string]any{
		"staff": map[string]any{
			"id":            staff.ID,
			"name":          staff.Name,
			"phone":         staff.Phone,
			"department_id": staff.DepartmentID,
		},
		"task": map[string]any{
			"id":        task.ID,
			"name":      task.Name,
			"task_type": task.TaskType,
		},
		"customer": customer,
		"assets":   workDecisionAssets(ctx, customerID),
		"current": map[string]any{
			"stage_code": workCurrentStageCode(state),
			"asset_id":   assetID,
			"asset":      workDecisionCurrentAsset(ctx, customerID, assetID),
		},
	}
}

func workDecisionCurrentAsset(ctx context.Context, customerID uint64, assetID uint64) map[string]any {
	if customerID == 0 || assetID == 0 {
		return map[string]any{}
	}
	asset := crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{
		"id":          assetID,
		"customer_id": customerID,
	})
	if len(asset) == 0 {
		return map[string]any{}
	}
	asset["fields"] = workAssetFieldValues(ctx, customerID, assetID)
	return asset
}

func workCurrentStageCode(state *crmmodel.CustomerStage) string {
	if state == nil {
		return ""
	}
	return state.CurrentStageCode
}

func workStateSnapshot(state *crmmodel.CustomerStage) map[string]any {
	if state == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":                    state.ID,
		"customer_id":           state.CustomerID,
		"asset_id":              state.AssetID,
		"current_stage_code":    state.CurrentStageCode,
		"current_department_id": state.CurrentDepartmentID,
		"current_staff_id":      state.CurrentStaffID,
		"last_operation_log_id": state.LastOperationLogID,
	}
}

func insertWorkOperationLog(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any) uint64 {
	return insertWorkOperationLogWithResult(ctx, staff, task, customerID, assetID, currentWorkCustomerStage(ctx, customerID, assetID), values, workResultSuccess)
}

func insertWorkOperationLogWithTitle(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, title string) uint64 {
	if title == "" || task == nil {
		return insertWorkOperationLog(ctx, staff, task, customerID, assetID, values)
	}
	return insertWorkOperationLogRecord(ctx, staff, task, customerID, assetID, currentWorkCustomerStage(ctx, customerID, assetID), values, workResultSuccess, title)
}

func insertWorkOperationLogWithResult(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage, values map[string]any, resultValue string) uint64 {
	title := ""
	if task != nil {
		title = task.Name
	}
	return insertWorkOperationLogRecord(ctx, staff, task, customerID, assetID, state, values, resultValue, title)
}

func insertWorkOperationLogRecord(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage, values map[string]any, resultValue string, title string) uint64 {
	now := time.Now()
	statusCode := ""
	if state != nil {
		statusCode = state.CurrentStageCode
	}
	if resultValue == "" {
		resultValue = workResultSuccess
	}
	if task == nil {
		return 0
	}
	if title == "" {
		title = task.Name
	}
	operationID := uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"customer_id":            customerID,
		"asset_id":               assetID,
		"task_id":                task.ID,
		"task_type":              task.TaskType,
		"stage_code":             statusCode,
		"result_value":           resultValue,
		"title":                  title,
		"content":                "",
		"data_snapshot_json":     jsonText(values),
		"operator_staff_id":      staff.ID,
		"operator_department_id": staff.DepartmentID,
		"created_at":             now,
	}))
	syncWorkTaskStatEvent(ctx, staff, task, customerID, assetID, statusCode, operationID, resultValue, now)
	return operationID
}

func insertWorkAutoTaskFailureLog(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, cause error) uint64 {
	if staff == nil || task == nil || cause == nil {
		return 0
	}
	state := currentWorkCustomerStage(ctx, customerID, assetID)
	stageCode := ""
	if state != nil {
		stageCode = state.CurrentStageCode
	}
	now := time.Now()
	message := cause.Error()
	return uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"customer_id":  customerID,
		"asset_id":     assetID,
		"task_id":      task.ID,
		"task_type":    task.TaskType,
		"stage_code":   stageCode,
		"result_value": workResultAutoFailed,
		"title":        fmt.Sprintf("自动任务失败：%s", task.Name),
		"content":      message,
		"data_snapshot_json": jsonText(map[string]any{
			"error":        message,
			"task_type":    task.TaskType,
			"trigger_type": task.TriggerType,
			"script_id":    task.ScriptID,
		}),
		"operator_staff_id":      staff.ID,
		"operator_department_id": staff.DepartmentID,
		"created_at":             now,
	}))
}

func saveWorkDataRecord(ctx context.Context, customerID uint64, assetID uint64, templateID uint64, taskID uint64, operationID uint64, record map[string]any) {
	now := time.Now()
	recordJSON := jsonText(record)
	data := map[string]any{
		"customer_id":      customerID,
		"asset_id":         assetID,
		"data_template_id": templateID,
		"task_id":          taskID,
		"operation_log_id": operationID,
		"record_json":      recordJSON,
		"summary":          "",
		"status":           crmmodel.StatusEnabled,
		"sort":             100,
		"updated_at":       now,
	}
	model := crmmodel.NewDataRecordModel()
	existing := model.Find(ctx, map[string]any{
		"customer_id":      customerID,
		"asset_id":         assetID,
		"data_template_id": templateID,
		"status":           crmmodel.StatusEnabled,
	})
	if existing != nil {
		merged := mapFromAny(existing.RecordJSON)
		for key, value := range record {
			merged[key] = value
		}
		data["record_json"] = jsonText(merged)
		model.Update(ctx, map[string]any{"id": existing.ID}, data)
		syncWorkStatFieldValues(ctx, customerID, assetID, templateID, taskID, operationID, record, now)
		return
	}
	data["created_at"] = now
	model.Insert(ctx, data)
	syncWorkStatFieldValues(ctx, customerID, assetID, templateID, taskID, operationID, record, now)
}

func syncWorkStatFieldValues(ctx context.Context, customerID uint64, assetID uint64, templateID uint64, taskID uint64, operationID uint64, record map[string]any, changedAt time.Time) {
	defer func() {
		_ = recover()
	}()
	if customerID == 0 || templateID == 0 || len(record) == 0 {
		return
	}
	model := crmmodel.NewStatFieldValueModel()
	for fieldIDText, value := range record {
		fieldID := inputUint64(fieldIDText)
		if fieldID == 0 {
			continue
		}
		field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
			"id":               fieldID,
			"data_template_id": templateID,
			"stat_enabled":     true,
			"status":           crmmodel.StatusEnabled,
		})
		if field == nil || strings.TrimSpace(field.FieldKey) == "" {
			continue
		}
		data := workStatFieldValueRecord(customerID, assetID, templateID, taskID, operationID, field, value, changedAt)
		existing := model.Find(ctx, map[string]any{
			"customer_id":   customerID,
			"asset_id":      assetID,
			"data_field_id": field.ID,
		})
		if existing != nil {
			model.Update(ctx, map[string]any{"id": existing.ID}, data)
			continue
		}
		data["created_at"] = changedAt
		model.Insert(ctx, data)
	}
}

func workStatFieldValueRecord(customerID uint64, assetID uint64, templateID uint64, taskID uint64, operationID uint64, field *crmmodel.DataField, value any, changedAt time.Time) map[string]any {
	valueText := inputText(value)
	if emptyWorkFieldValue(value) {
		valueText = ""
	}
	return map[string]any{
		"customer_id":      customerID,
		"asset_id":         assetID,
		"data_template_id": templateID,
		"data_field_id":    field.ID,
		"field_key":        field.FieldKey,
		"field_name":       field.Name,
		"field_type":       field.FieldType,
		"stat_type":        normalizeWorkStatType(field.StatType),
		"stat_group":       field.StatGroup,
		"value_text":       valueText,
		"value_number":     workStatNumberValue(field, value),
		"value_date":       workStatDateValue(field, value),
		"value_bool":       booleanFromAny(value),
		"value_json":       workStatJSONValue(value),
		"source":           crmmodel.StatValueSourceForm,
		"task_id":          taskID,
		"operation_log_id": operationID,
		"status":           crmmodel.StatusEnabled,
		"updated_at":       changedAt,
	}
}

func normalizeWorkStatType(statType string) string {
	switch strings.TrimSpace(statType) {
	case crmmodel.DataFieldStatTypeMetric,
		crmmodel.DataFieldStatTypeAmount,
		crmmodel.DataFieldStatTypeFinance,
		crmmodel.DataFieldStatTypeTime,
		crmmodel.DataFieldStatTypeStatus,
		crmmodel.DataFieldStatTypeText:
		return strings.TrimSpace(statType)
	default:
		return crmmodel.DataFieldStatTypeDimension
	}
}

func workStatNumberValue(field *crmmodel.DataField, value any) float64 {
	if field == nil {
		return 0
	}
	switch field.StatType {
	case crmmodel.DataFieldStatTypeMetric, crmmodel.DataFieldStatTypeAmount, crmmodel.DataFieldStatTypeFinance:
		return numericValue(value)
	}
	switch field.FieldType {
	case "number", "money":
		return numericValue(value)
	default:
		return 0
	}
}

func workStatDateValue(field *crmmodel.DataField, value any) time.Time {
	if field == nil {
		return time.Time{}
	}
	if field.StatType != crmmodel.DataFieldStatTypeTime && field.FieldType != "date" && field.FieldType != "datetime" {
		return time.Time{}
	}
	text := inputText(value)
	if text == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func workStatJSONValue(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(raw)
}

func syncWorkTaskStatEvent(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, stageCode string, operationID uint64, resultValue string, eventAt time.Time) {
	defer func() {
		_ = recover()
	}()
	if staff == nil || task == nil || customerID == 0 || operationID == 0 {
		return
	}
	eventKey := fmt.Sprintf("task:%d:%s", task.ID, resultValue)
	insertWorkStatEvent(ctx, map[string]any{
		"event_type":             crmmodel.StatEventTypeTask,
		"event_key":              eventKey,
		"customer_id":            customerID,
		"asset_id":               assetID,
		"stage_code":             stageCode,
		"from_stage_code":        "",
		"to_stage_code":          "",
		"task_id":                task.ID,
		"task_type":              task.TaskType,
		"result_value":           resultValue,
		"operation_log_id":       operationID,
		"transition_log_id":      0,
		"operator_staff_id":      staff.ID,
		"operator_department_id": staff.DepartmentID,
		"event_at":               eventAt,
		"created_at":             eventAt,
	})
}

func syncWorkTransitionStatEvent(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, fromStageCode string, toStageCode string, operationID uint64, transitionLogID uint64, resultValue string, eventAt time.Time) {
	defer func() {
		_ = recover()
	}()
	if staff == nil || task == nil || customerID == 0 || operationID == 0 || transitionLogID == 0 {
		return
	}
	eventKey := fmt.Sprintf("transition:%s:%s:%d:%s", fromStageCode, toStageCode, task.ID, resultValue)
	insertWorkStatEvent(ctx, map[string]any{
		"event_type":             crmmodel.StatEventTypeTransition,
		"event_key":              eventKey,
		"customer_id":            customerID,
		"asset_id":               assetID,
		"stage_code":             toStageCode,
		"from_stage_code":        fromStageCode,
		"to_stage_code":          toStageCode,
		"task_id":                task.ID,
		"task_type":              task.TaskType,
		"result_value":           resultValue,
		"operation_log_id":       operationID,
		"transition_log_id":      transitionLogID,
		"operator_staff_id":      staff.ID,
		"operator_department_id": staff.DepartmentID,
		"event_at":               eventAt,
		"created_at":             eventAt,
	})
}

func insertWorkStatEvent(ctx context.Context, record map[string]any) {
	model := crmmodel.NewStatEventModel()
	if existing := model.Find(ctx, map[string]any{
		"event_key":         record["event_key"],
		"operation_log_id":  record["operation_log_id"],
		"transition_log_id": record["transition_log_id"],
	}); existing != nil {
		return
	}
	model.Insert(ctx, record)
}

func insertWorkCustomerMember(ctx context.Context, staff *WorkStaffSession, customerID uint64) {
	crmmodel.NewCustomerMemberModel().Insert(ctx, map[string]any{
		"customer_id":   customerID,
		"asset_id":      0,
		"department_id": staff.DepartmentID,
		"staff_id":      staff.ID,
		"relation_type": crmmodel.MemberRelationCreator,
		"can_view":      true,
		"status":        crmmodel.StatusEnabled,
		"created_at":    time.Now(),
	})
}

func upsertWorkAssigneeMember(ctx context.Context, customerID uint64, assetID uint64, departmentID uint64, staffID uint64) {
	if customerID == 0 || (departmentID == 0 && staffID == 0) {
		return
	}
	model := crmmodel.NewCustomerMemberModel()
	existing := model.Find(ctx, map[string]any{
		"customer_id":   customerID,
		"asset_id":      assetID,
		"relation_type": crmmodel.MemberRelationAssignee,
		"status":        crmmodel.StatusEnabled,
	})
	record := map[string]any{
		"department_id": departmentID,
		"staff_id":      staffID,
		"can_view":      true,
	}
	if existing != nil {
		model.Update(ctx, map[string]any{"id": existing.ID}, record)
		return
	}
	record["customer_id"] = customerID
	record["asset_id"] = assetID
	record["relation_type"] = crmmodel.MemberRelationAssignee
	record["status"] = crmmodel.StatusEnabled
	record["created_at"] = time.Now()
	model.Insert(ctx, record)
}

func insertWorkCustomerStage(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, operationID uint64, taskID uint64) {
	stage := workInitialStageForTask(ctx, staff, taskID)
	stageCode := ""
	if stage != nil {
		stageCode = stage.Code
	}
	now := time.Now()
	model := crmmodel.NewCustomerStageModel()
	if existing := model.Find(ctx, map[string]any{"customer_id": customerID, "asset_id": assetID}); existing != nil {
		model.Update(ctx, map[string]any{"id": existing.ID}, map[string]any{
			"last_operation_log_id": operationID,
			"last_operated_at":      now,
			"updated_at":            now,
		})
		return
	}
	record := map[string]any{
		"customer_id":            customerID,
		"asset_id":               assetID,
		"current_stage_code":     stageCode,
		"current_department_id":  staff.DepartmentID,
		"current_staff_id":       0,
		"last_operation_log_id":  operationID,
		"last_transition_log_id": 0,
		"last_operated_at":       now,
		"context_json":           "{}",
		"created_at":             now,
		"updated_at":             now,
	}
	model.Insert(ctx, record)
}

func workInitialStageForTask(ctx context.Context, staff *WorkStaffSession, taskID uint64) *crmmodel.Stage {
	if taskID > 0 {
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": taskID, "status": crmmodel.StatusEnabled})
		if task != nil && task.StageID > 0 {
			if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": task.StageID, "status": crmmodel.StatusEnabled}); stage != nil {
				return stage
			}
		}
	}
	if staff == nil || staff.DepartmentID == 0 {
		return nil
	}
	return crmmodel.NewStageModel().Find(ctx, map[string]any{
		"owner_department_id": staff.DepartmentID,
		"status":              crmmodel.StatusEnabled,
	})
}

func currentWorkCustomerStage(ctx context.Context, customerID uint64, assetID uint64) *crmmodel.CustomerStage {
	return crmmodel.NewCustomerStageModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	})
}

func ensureCurrentWorkCustomerStage(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) *crmmodel.CustomerStage {
	state := crmmodel.NewCustomerStageModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	})
	if state != nil || customerID == 0 {
		return state
	}
	return ensureWorkCustomerStage(ctx, staff, customerID, assetID)
}

func ensureWorkCustomerStage(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) *crmmodel.CustomerStage {
	customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
	if customer == nil {
		return nil
	}
	stage := firstEnabledWorkStage(ctx)
	if stage == nil {
		return nil
	}
	departmentID := initialWorkCustomerDepartmentID(ctx, staff, stage, customer)
	now := time.Now()
	record := map[string]any{
		"customer_id":            customerID,
		"asset_id":               assetID,
		"current_stage_code":     stage.Code,
		"current_department_id":  departmentID,
		"current_staff_id":       uint64(0),
		"last_operation_log_id":  uint64(0),
		"last_transition_log_id": uint64(0),
		"last_operated_at":       now,
		"context_json":           "{}",
		"created_at":             now,
		"updated_at":             now,
	}
	model := crmmodel.NewCustomerStageModel()
	model.Insert(ctx, record)
	return model.Find(ctx, map[string]any{"customer_id": customerID, "asset_id": assetID})
}

func firstEnabledWorkStage(ctx context.Context) *crmmodel.Stage {
	return crmmodel.NewStageModel().Find(
		ctx,
		map[string]any{"status": crmmodel.StatusEnabled},
		map[string]any{"order": "sort asc, id asc"},
	)
}

func initialWorkCustomerDepartmentID(ctx context.Context, staff *WorkStaffSession, stage *crmmodel.Stage, customer *crmmodel.Customer) uint64 {
	if stage != nil && stage.OwnerDepartmentID > 0 {
		return stage.OwnerDepartmentID
	}
	if staff != nil && staff.DepartmentID > 0 {
		return staff.DepartmentID
	}
	if customer != nil && customer.CreatedByStaffID > 0 {
		if creator := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": customer.CreatedByStaffID, "status": crmmodel.StatusEnabled}); creator != nil {
			return creator.DepartmentID
		}
	}
	return 0
}

func updateWorkCustomerOwner(ctx context.Context, customerID uint64, assetID uint64, departmentID uint64, staffID uint64, operationID uint64) {
	now := time.Now()
	crmmodel.NewCustomerStageModel().Update(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	}, map[string]any{
		"current_department_id": departmentID,
		"current_staff_id":      staffID,
		"last_operation_log_id": operationID,
		"last_operated_at":      now,
		"updated_at":            now,
	})
}

func updateWorkCustomerStageOperation(ctx context.Context, customerID uint64, assetID uint64, operationID uint64) {
	now := time.Now()
	crmmodel.NewCustomerStageModel().Update(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	}, map[string]any{
		"last_operation_log_id": operationID,
		"last_operated_at":      now,
		"updated_at":            now,
	})
}

func applyWorkStageTransition(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, fromState *crmmodel.CustomerStage, task *crmmodel.Task, operationID uint64, resultValue string) string {
	return applyWorkStageTransitionWithOwner(ctx, staff, customerID, assetID, fromState, task, operationID, resultValue, 0, 0)
}

func applyWorkStageTransitionWithOwner(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, fromState *crmmodel.CustomerStage, task *crmmodel.Task, operationID uint64, resultValue string, assignedDepartmentID uint64, assignedStaffID uint64) string {
	if fromState == nil || task == nil {
		updateWorkCustomerStageOperation(ctx, customerID, assetID, operationID)
		return ""
	}
	transition := findWorkStageTransition(ctx, staff, customerID, fromState, task, resultValue)
	if transition == nil {
		updateWorkCustomerStageOperation(ctx, customerID, assetID, operationID)
		return ""
	}
	departmentID, staffID := transitionOwner(ctx, staff, fromState, transition, assignedDepartmentID, assignedStaffID)
	now := time.Now()
	transitionLogID := uint64(crmmodel.NewStageTransitionLogModel().Insert(ctx, map[string]any{
		"customer_id":       customerID,
		"asset_id":          assetID,
		"from_stage_code":   fromState.CurrentStageCode,
		"to_stage_code":     transition.ToStageCode,
		"task_id":           task.ID,
		"result_value":      resultValue,
		"operation_log_id":  operationID,
		"operator_staff_id": staff.ID,
		"created_at":        now,
	}))
	crmmodel.NewCustomerStageModel().Update(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	}, map[string]any{
		"current_stage_code":     transition.ToStageCode,
		"current_department_id":  departmentID,
		"current_staff_id":       staffID,
		"last_operation_log_id":  operationID,
		"last_transition_log_id": transitionLogID,
		"last_operated_at":       now,
		"updated_at":             now,
	})
	if departmentID > 0 || staffID > 0 {
		upsertWorkAssigneeMember(ctx, customerID, assetID, departmentID, staffID)
	}
	syncWorkTransitionStatEvent(ctx, staff, task, customerID, assetID, fromState.CurrentStageCode, transition.ToStageCode, operationID, transitionLogID, resultValue, now)
	if transition.ToStageCode == fromState.CurrentStageCode {
		return ""
	}
	return transition.ToStageCode
}

func findWorkStageTransition(ctx context.Context, staff *WorkStaffSession, customerID uint64, fromState *crmmodel.CustomerStage, task *crmmodel.Task, resultValue string) *crmmodel.StageTransition {
	if fromState == nil || task == nil {
		return nil
	}
	transitions := crmmodel.NewStageTransitionModel().Select(ctx, map[string]any{
		"from_stage_code": fromState.CurrentStageCode,
		"task_id":         task.ID,
		"status":          crmmodel.StatusEnabled,
	})
	var fallback *crmmodel.StageTransition
	for _, transition := range transitions {
		if transition == nil {
			continue
		}
		if transition.ResultValue == resultValue && workTransitionMatches(ctx, staff, customerID, fromState, task, transition, resultValue) {
			return transition
		}
		if transition.ResultValue == "" && fallback == nil && workTransitionMatches(ctx, staff, customerID, fromState, task, transition, resultValue) {
			fallback = transition
		}
	}
	if fallback != nil {
		return fallback
	}
	if len(transitions) > 0 || workTaskResultRequiresExplicitTransition(task, resultValue) {
		return nil
	}
	return defaultWorkTaskTransition(ctx, fromState, task)
}

func workTaskResultRequiresExplicitTransition(task *crmmodel.Task, resultValue string) bool {
	if task == nil {
		return false
	}
	if task.TaskType == crmmodel.TaskTypeDecision {
		return true
	}
	config := mapFromAny(task.ConfigJSON)
	return inputUint64(config["result_field_id"]) > 0 && resultValue != workResultSuccess
}

func defaultWorkTaskTransition(ctx context.Context, fromState *crmmodel.CustomerStage, task *crmmodel.Task) *crmmodel.StageTransition {
	if task.TaskType == crmmodel.TaskTypeDecision {
		return nil
	}
	nextStageCode := inputText(mapFromAny(task.ConfigJSON)["next_stage_code"])
	if nextStageCode == "" || nextStageCode == fromState.CurrentStageCode {
		return nil
	}
	if crmmodel.NewStageModel().Find(ctx, map[string]any{"code": nextStageCode, "status": crmmodel.StatusEnabled}) == nil {
		return nil
	}
	ownerMode := crmmodel.StageOwnerKeep
	if task.TaskType == crmmodel.TaskTypeAssign {
		ownerMode = crmmodel.StageOwnerAssign
	}
	return &crmmodel.StageTransition{
		FromStageCode: fromState.CurrentStageCode,
		TaskID:        task.ID,
		ToStageCode:   nextStageCode,
		OwnerMode:     ownerMode,
		Status:        crmmodel.StatusEnabled,
	}
}

func workTransitionMatches(ctx context.Context, staff *WorkStaffSession, customerID uint64, state *crmmodel.CustomerStage, task *crmmodel.Task, transition *crmmodel.StageTransition, resultValue string) bool {
	input := workTransitionInput(ctx, staff, task, transition, customerID, state, resultValue)
	if !workTransitionConditionMatches(transition.ConditionJSON, input) {
		return false
	}
	return workTransitionScriptMatches(ctx, transition, input)
}

func workTransitionInput(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, transition *crmmodel.StageTransition, customerID uint64, state *crmmodel.CustomerStage, resultValue string) map[string]any {
	assetID := uint64(0)
	if state != nil {
		assetID = state.AssetID
	}
	input := workDecisionInput(ctx, staff, task, customerID, assetID, state)
	input["transition"] = map[string]any{
		"id":              transition.ID,
		"from_stage_code": transition.FromStageCode,
		"to_stage_code":   transition.ToStageCode,
		"result_value":    transition.ResultValue,
	}
	input["result_value"] = resultValue
	return input
}

func workTransitionConditionMatches(raw string, input map[string]any) bool {
	mode, rows := workTransitionConditionRows(raw)
	if len(rows) == 0 {
		return true
	}
	matched := 0
	for _, row := range rows {
		if workTransitionConditionRowMatches(row, input) {
			matched++
			if mode == workTransitionModeAny {
				return true
			}
		} else if mode == workTransitionModeAll {
			return false
		}
	}
	return matched == len(rows)
}

func workTransitionConditionRows(raw string) (string, []map[string]any) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" || raw == "[]" {
		return workTransitionModeAll, nil
	}
	var object map[string]any
	if err := json.Unmarshal([]byte(raw), &object); err == nil {
		mode := inputText(object["mode"])
		if mode == "" {
			mode = workTransitionModeAll
		}
		if rows := mapsFromAny(object[workTransitionModeAny]); len(rows) > 0 {
			return workTransitionModeAny, rows
		}
		if rows := mapsFromAny(object[workTransitionModeAll]); len(rows) > 0 {
			return workTransitionModeAll, rows
		}
		if rows := mapsFromAny(object["conditions"]); len(rows) > 0 {
			return mode, rows
		}
		if inputText(object["field"]) != "" || inputText(object["path"]) != "" {
			return mode, []map[string]any{object}
		}
		return mode, nil
	}
	return workTransitionModeAll, mapsFromJSON(raw)
}

func workTransitionConditionRowMatches(row map[string]any, input map[string]any) bool {
	path := firstText(row, "field", "path")
	if path == "" {
		return true
	}
	actual := valueByPath(input, path)
	expected := firstPresent(row, "value", "values", "expected")
	operator := firstText(row, "operator", "op")
	if operator == "" {
		operator = "equals"
	}
	return workConditionValueMatches(actual, expected, operator)
}

func workConditionValueMatches(actual any, expected any, operator string) bool {
	switch operator {
	case "equals", "eq", "=", "==":
		return valuesEqual(actual, expected)
	case "notEquals", "ne", "!=", "<>":
		return !valuesEqual(actual, expected)
	case "in":
		return valueInList(actual, expected)
	case "notIn":
		return !valueInList(actual, expected)
	case "empty":
		return emptyWorkFieldValue(actual)
	case "notEmpty":
		return !emptyWorkFieldValue(actual)
	case "contains":
		return strings.Contains(inputText(actual), inputText(expected))
	default:
		return valuesEqual(actual, expected)
	}
}

func workTransitionScriptMatches(ctx context.Context, transition *crmmodel.StageTransition, input map[string]any) bool {
	if transition.ScriptID == 0 {
		return true
	}
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{"id": transition.ScriptID, "status": crmmodel.StatusEnabled})
	if script == nil {
		return false
	}
	result, err := fronteval.Run(ctx, fronteval.Request{
		Language: fronteval.LanguageJavaScript,
		Script:   script.Script,
		Entry:    fronteval.DefaultEntry,
		Input:    input,
		Config:   map[string]any{},
	})
	if err != nil {
		return false
	}
	return workTransitionScriptResultPassed(result.Value)
}

func workTransitionScriptResultPassed(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case map[string]any:
		for _, key := range []string{"matched", "pass", "ok", "success"} {
			if booleanFromAny(typed[key]) {
				return true
			}
		}
		return inputText(typed["result"]) == workTransitionScriptPass
	default:
		return booleanFromAny(value) || inputText(value) == workTransitionScriptPass
	}
}

func transitionOwner(ctx context.Context, staff *WorkStaffSession, fromState *crmmodel.CustomerStage, transition *crmmodel.StageTransition, assignedDepartmentID uint64, assignedStaffID uint64) (uint64, uint64) {
	switch transition.OwnerMode {
	case crmmodel.StageOwnerFixedDepartment:
		return transition.ToDepartmentID, 0
	case crmmodel.StageOwnerFixedStaff:
		if targetStaff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": transition.ToStaffID, "status": crmmodel.StatusEnabled}); targetStaff != nil {
			departmentID := transition.ToDepartmentID
			if departmentID == 0 {
				departmentID = targetStaff.DepartmentID
			}
			return departmentID, targetStaff.ID
		}
	case crmmodel.StageOwnerCreator:
		if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": fromState.CustomerID}); customer != nil {
			if creator := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": customer.CreatedByStaffID, "status": crmmodel.StatusEnabled}); creator != nil {
				return creator.DepartmentID, creator.ID
			}
		}
	case crmmodel.StageOwnerAssign:
		if assignedStaffID > 0 || assignedDepartmentID > 0 {
			return assignedDepartmentID, assignedStaffID
		}
		if transition.ToStaffID > 0 || transition.ToDepartmentID > 0 {
			return transition.ToDepartmentID, transition.ToStaffID
		}
		return fromState.CurrentDepartmentID, fromState.CurrentStaffID
	case crmmodel.StageOwnerKeep:
		return fromState.CurrentDepartmentID, fromState.CurrentStaffID
	}
	if transition.ToStaffID > 0 || transition.ToDepartmentID > 0 {
		return transition.ToDepartmentID, transition.ToStaffID
	}
	if staff != nil {
		return staff.DepartmentID, staff.ID
	}
	return fromState.CurrentDepartmentID, fromState.CurrentStaffID
}
