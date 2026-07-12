package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	agentmodel "github.com/dever-package/bot/model/agent"
	agentservice "github.com/dever-package/bot/service/agent"
	crmmodel "github.com/dever-package/crm/model"
)

func (WorkService) AIFill(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	taskID := firstUint64(payload, "task_id", "taskId")
	if taskID == 0 {
		return nil, fmt.Errorf("任务不能为空")
	}
	customerID := firstUint64(payload, "customer_id", "customerId")
	assetID := firstUint64(payload, "asset_id", "assetId")
	if assetID > 0 && !workCustomerOwnsAsset(ctx, customerID, assetID) {
		return nil, fmt.Errorf("客户资产不存在")
	}
	task := workAllowedTask(ctx, staff, taskID, customerID, assetID, firstUint64(payload, "todo_id", "todoId"))
	if task == nil {
		return nil, fmt.Errorf("当前人员无权执行该任务")
	}
	formID, err := workAIFillFormID(ctx, staff, task, customerID, assetID, payload)
	if err != nil {
		return nil, err
	}
	fields := workAIFieldsForForm(ctx, formID)
	if len(fields) == 0 {
		return nil, fmt.Errorf("当前任务没有可 AI 填写的字段")
	}
	result, err := agentservice.NewService().RunInternal(ctx, agentservice.InternalRunRequest{
		AgentID: agentmodel.FrontAssistantAgentID,
		Input:   workAIFillInput(ctx, staff, task, formID, fields, customerID, assetID, payload),
		Options: map[string]any{
			"max_steps": 1,
		},
	})
	if err != nil {
		return nil, err
	}
	action := extractWorkAIFillFrontActionFromOutput(result.Output, result.Summary)
	values := sanitizeWorkAIFillValues(fields, extractWorkAIFillValues(result.Output, result.Summary, action))
	return map[string]any{
		"values":       values,
		"summary":      inputText(firstPresent(action, "summary", "text")),
		"request_id":   result.RequestID,
		"run_id":       result.RunID,
		"filled_count": len(values),
	}, nil
}

type workAIFillField struct {
	Key       string
	Name      string
	FieldType string
	Required  bool
	Options   []map[string]any
}

func workAIFillFormID(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, payload map[string]any) (uint64, error) {
	if task == nil {
		return 0, nil
	}
	todoID := firstUint64(payload, "todo_id", "todoId")
	if todoID == 0 {
		return task.FormID, nil
	}
	todo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
		"id":     todoID,
		"status": crmmodel.WorkTodoStatusPending,
	})
	if todo == nil {
		return 0, fmt.Errorf("协作待办不存在或已完成")
	}
	if todo.CustomerID != customerID || todo.AssetID != assetID {
		return 0, fmt.Errorf("协作待办不属于当前客户资产")
	}
	if !canOperateWorkTodo(staff, todo) {
		return 0, fmt.Errorf("当前人员无权完成该协作待办")
	}
	return todo.FormID, nil
}

func workAIFieldsForForm(ctx context.Context, formID uint64) []workAIFillField {
	if formID == 0 {
		return nil
	}
	form := crmmodel.NewFormModel().Find(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled})
	if form == nil {
		return nil
	}
	rows := crmmodel.NewFormFieldModel().Select(ctx, map[string]any{"form_id": form.ID, "status": crmmodel.StatusEnabled})
	fields := make([]workAIFillField, 0, len(rows))
	for _, sourceField := range rows {
		for _, field := range expandWorkInputFormFields(ctx, sourceField) {
			if field == nil || field.Readonly {
				continue
			}
			row := map[string]any{
				"main_field":    field.MainField,
				"data_field_id": field.DataFieldID,
			}
			attachWorkFormFieldOptions(ctx, row)
			fieldType := inputText(row["field_type"])
			if workAIFillFieldIsUpload(fieldType) {
				continue
			}
			key := workFieldInputKey(field)
			if key == "" {
				continue
			}
			fields = append(fields, workAIFillField{
				Key:       key,
				Name:      field.Name,
				FieldType: fieldType,
				Required:  field.Required,
				Options:   mapsFromAny(row["options"]),
			})
		}
	}
	return fields
}

func workAIFillFieldIsUpload(fieldType string) bool {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "attachment", "file", "image":
		return true
	default:
		return false
	}
}

func workAIFillInput(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, formID uint64, fields []workAIFillField, customerID uint64, assetID uint64, payload map[string]any) map[string]any {
	context := workAIFillContext(ctx, staff, task, formID, fields, customerID, assetID, payload)
	return map[string]any{
		"text": "请根据 CRM 工作台上下文补全当前任务表单。只输出 ```front-action fenced JSON，type 必须是 patch_form，values 只包含 fields 中存在的 key；不要保存、提交或执行任何业务动作。",
		"task": map[string]any{
			"type":                 "fill_current_form",
			"allowed_action_types": []string{"patch_form"},
			"empty_only":           true,
			"instruction":          "只补全空字段；已有值不覆盖；没有把握的字段不要填写；下拉字段必须返回选项 id；数字和金额只返回数字；日期返回 YYYY-MM-DD；时间返回 YYYY-MM-DD HH:mm:ss；布尔返回 true 或 false。",
		},
		"page_context": context,
	}
}

func workAIFillContext(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, formID uint64, fields []workAIFillField, customerID uint64, assetID uint64, payload map[string]any) map[string]any {
	customer := map[string]any{}
	if customerID > 0 {
		customer = crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
		customer["fields"] = workCustomerFieldValues(ctx, customerID)
	}
	instruction := strings.TrimSpace(inputText(firstPresent(payload, "instruction", "prompt", "text")))
	return map[string]any{
		"surface": "crm_workbench_task_form",
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
			"form_id":   formID,
		},
		"fields":         workAIFillFieldPayloads(fields),
		"current_values": mapFromAny(payload["values"]),
		"instruction":    instruction,
		"customer":       customer,
		"assets":         workDecisionAssets(ctx, customerID),
		"current": map[string]any{
			"customer_id": customerID,
			"asset_id":    assetID,
			"stage_code":  workCurrentStageCode(currentWorkCustomerStage(ctx, customerID, assetID)),
		},
		"recent_operations": workAIFillRecentOperations(ctx, customerID, assetID, 10),
	}
}

func workAIFillFieldPayloads(fields []workAIFillField) []map[string]any {
	rows := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		rows = append(rows, map[string]any{
			"key":        field.Key,
			"name":       field.Name,
			"field_type": field.FieldType,
			"required":   field.Required,
			"options":    workAIFillOptionPayloads(field.Options),
		})
	}
	return rows
}

func workAIFillOptionPayloads(options []map[string]any) []map[string]any {
	rows := make([]map[string]any, 0, len(options))
	for _, option := range options {
		id := inputText(firstPresent(option, "id", "value"))
		name := firstText(option, "name", "label", "value")
		if id == "" || name == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"id":    id,
			"name":  name,
			"value": inputText(firstPresent(option, "value", "id")),
		})
	}
	return rows
}

func workAIFillRecentOperations(ctx context.Context, customerID uint64, assetID uint64, limit int) []map[string]any {
	if customerID == 0 || limit <= 0 {
		return nil
	}
	filter := map[string]any{"customer_id": customerID}
	if assetID > 0 {
		filter["asset_id"] = assetID
	}
	rows := crmmodel.NewOperationLogModel().SelectMap(ctx, filter)
	if len(rows) > limit {
		rows = rows[:limit]
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{
			"title":      inputText(row["title"]),
			"task_type":  inputText(row["task_type"]),
			"stage_code": inputText(row["stage_code"]),
			"result":     inputText(row["result_value"]),
			"data":       mapFromAny(row["data_snapshot_json"]),
			"created_at": inputText(row["created_at"]),
		})
	}
	return result
}

func extractWorkAIFillValues(output map[string]any, summary string, action map[string]any) map[string]any {
	for _, source := range []any{
		output["values"],
		mapFromAny(output["content"])["values"],
		action["values"],
	} {
		if values := mapFromAny(source); len(values) > 0 {
			return values
		}
	}
	return map[string]any{}
}

func extractWorkAIFillFrontActionFromOutput(output map[string]any, summary string) map[string]any {
	for _, text := range []string{inputText(output["text"]), summary} {
		body, ok := extractWorkJSONFence(text, "front-action")
		if !ok {
			continue
		}
		action := mapFromAny(body)
		if strings.TrimSpace(inputText(action["type"])) == "patch_form" {
			return action
		}
	}
	return map[string]any{}
}

func extractWorkJSONFence(text string, lang string) (string, bool) {
	search := "```" + lang
	start := strings.Index(text, search)
	if start < 0 {
		return "", false
	}
	bodyStart := start + len(search)
	for bodyStart < len(text) && (text[bodyStart] == ' ' || text[bodyStart] == '\t' || text[bodyStart] == '\r' || text[bodyStart] == '\n') {
		bodyStart++
	}
	end := strings.Index(text[bodyStart:], "```")
	if end < 0 {
		return strings.TrimSpace(text[bodyStart:]), true
	}
	return strings.TrimSpace(text[bodyStart : bodyStart+end]), true
}

func sanitizeWorkAIFillValues(fields []workAIFillField, raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	allowed := map[string]workAIFillField{}
	for _, field := range fields {
		allowed[field.Key] = field
	}
	result := map[string]any{}
	for key, value := range raw {
		field, ok := allowed[strings.TrimSpace(key)]
		if !ok || emptyWorkFieldValue(value) {
			continue
		}
		if normalized, ok := sanitizeWorkAIFillValue(field, value); ok {
			result[field.Key] = normalized
		}
	}
	return result
}

func sanitizeWorkAIFillValue(field workAIFillField, value any) (any, bool) {
	if len(field.Options) == 0 {
		if workAIFillFieldRequiresOptions(field.FieldType) {
			return nil, false
		}
		return sanitizeWorkAIFillScalarValue(field, value)
	}
	optionIDs := make([]string, 0)
	seen := map[string]bool{}
	for _, optionValue := range workAIFillOptionInputValues(value) {
		optionID, ok := matchWorkAIFillOption(field.Options, optionValue)
		if !ok || seen[optionID] {
			continue
		}
		optionIDs = append(optionIDs, optionID)
		seen[optionID] = true
	}
	if len(optionIDs) == 0 {
		return nil, false
	}
	if workAIFillFieldIsMultiple(field.FieldType) {
		return optionIDs, true
	}
	return optionIDs[0], true
}

func workAIFillFieldRequiresOptions(fieldType string) bool {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "radio", "checkbox", "select", "multi_select", "multiple_select":
		return true
	default:
		return false
	}
}

func sanitizeWorkAIFillScalarValue(field workAIFillField, value any) (any, bool) {
	switch strings.ToLower(strings.TrimSpace(field.FieldType)) {
	case "number", "money":
		return sanitizeWorkAIFillNumber(value)
	case "date":
		return sanitizeWorkAIFillDate(value)
	case "datetime":
		return sanitizeWorkAIFillDateTime(value)
	case "boolean":
		return sanitizeWorkAIFillBoolean(value)
	default:
		text := inputText(value)
		if text == "" {
			return nil, false
		}
		return text, true
	}
}

func sanitizeWorkAIFillNumber(value any) (any, bool) {
	switch typed := value.(type) {
	case int:
		return strconv.FormatInt(int64(typed), 10), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case uint64:
		return strconv.FormatUint(typed, 10), true
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32), true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case json.Number:
		return sanitizeWorkAIFillNumberText(typed.String())
	default:
		return sanitizeWorkAIFillNumberText(inputText(value))
	}
}

func sanitizeWorkAIFillNumberText(text string) (any, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false
	}
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, "，", "")
	text = strings.TrimSpace(strings.Trim(text, "￥$¥ "))
	if text == "" {
		return nil, false
	}
	if _, err := strconv.ParseFloat(text, 64); err != nil {
		return nil, false
	}
	return text, true
}

func sanitizeWorkAIFillDate(value any) (any, bool) {
	parsed, ok := parseWorkAIFillTime(value)
	if !ok {
		return nil, false
	}
	return parsed.Format("2006-01-02"), true
}

func sanitizeWorkAIFillDateTime(value any) (any, bool) {
	parsed, ok := parseWorkAIFillTime(value)
	if !ok {
		return nil, false
	}
	return parsed.Format("2006-01-02 15:04:05"), true
}

func parseWorkAIFillTime(value any) (time.Time, bool) {
	if typed, ok := value.(time.Time); ok && !typed.IsZero() {
		return typed, true
	}
	text := normalizeWorkAIFillTimeText(inputText(value))
	if text == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006-1-2 15:04",
		"2006-1-2 15:04:05",
		"2006-1-2",
	} {
		if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func normalizeWorkAIFillTimeText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "/", "-")
	text = strings.ReplaceAll(text, "年", "-")
	text = strings.ReplaceAll(text, "月", "-")
	text = strings.ReplaceAll(text, "日", "")
	return strings.TrimSpace(text)
}

func sanitizeWorkAIFillBoolean(value any) (any, bool) {
	switch typed := value.(type) {
	case bool:
		return strconv.FormatBool(typed), true
	case int:
		return strconv.FormatBool(typed != 0), true
	case int64:
		return strconv.FormatBool(typed != 0), true
	case uint64:
		return strconv.FormatBool(typed != 0), true
	case float64:
		return strconv.FormatBool(typed != 0), true
	}
	switch strings.ToLower(inputText(value)) {
	case "true", "1", "yes", "y", "on", "是", "对", "通过", "成功":
		return "true", true
	case "false", "0", "no", "n", "off", "否", "不", "未通过", "失败":
		return "false", true
	default:
		return nil, false
	}
}

func workAIFillFieldIsMultiple(fieldType string) bool {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "checkbox", "multi_select", "multiple_select":
		return true
	default:
		return false
	}
}

func workAIFillOptionInputValues(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []string:
		values := make([]any, 0, len(typed))
		for _, item := range typed {
			values = append(values, item)
		}
		return values
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		var arrayValues []any
		if strings.HasPrefix(text, "[") && json.Unmarshal([]byte(text), &arrayValues) == nil {
			return arrayValues
		}
		if strings.Contains(text, ",") || strings.Contains(text, "，") {
			parts := strings.FieldsFunc(text, func(r rune) bool {
				return r == ',' || r == '，'
			})
			values := make([]any, 0, len(parts))
			for _, part := range parts {
				if part = strings.TrimSpace(part); part != "" {
					values = append(values, part)
				}
			}
			return values
		}
		return []any{text}
	default:
		if emptyWorkFieldValue(value) {
			return nil
		}
		return []any{value}
	}
}

func matchWorkAIFillOption(options []map[string]any, value any) (string, bool) {
	text := inputText(value)
	if text == "" {
		return "", false
	}
	for _, option := range options {
		candidates := []string{
			inputText(option["id"]),
			inputText(option["value"]),
			inputText(option["name"]),
			inputText(option["label"]),
		}
		for _, candidate := range candidates {
			if candidate != "" && strings.EqualFold(candidate, text) {
				id := inputText(option["id"])
				if id == "" {
					id = inputText(option["value"])
				}
				return id, id != ""
			}
		}
	}
	return "", false
}
