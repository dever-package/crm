package setting

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

var ensureDefaultStageOnce sync.Once

func (CrmHook) ProviderBuildTaskRows(c *server.Context, params []any) any {
	ensureDefaultStage(contextFromServer(c))
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}
	for _, row := range rows {
		row["task_type_name"] = taskTypeName(row["task_type"])
		row["form_mode_name"] = taskFormModeName(row["task_type"], row["form_mode"])
		row["trigger_type_name"] = taskTriggerTypeName(row["trigger_type"])
	}
	return rows
}

func taskTypeName(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.TaskTypeForm:
		return "填写资料"
	case crmmodel.TaskTypeAssign:
		return "分配"
	case crmmodel.TaskTypeDecision:
		return "决策"
	case crmmodel.TaskTypeBooking:
		return "资源预定"
	default:
		return util.ToStringTrimmed(value)
	}
}

func taskFormModeName(taskType any, value any) string {
	if util.ToStringTrimmed(taskType) != crmmodel.TaskTypeForm {
		return "-"
	}
	switch util.ToStringTrimmed(value) {
	case crmmodel.TaskFormModeCreate:
		return "新增"
	default:
		return "编辑"
	}
}

func taskTriggerTypeName(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.TaskTriggerAfterTask:
		return "任务后触发"
	case crmmodel.TaskTriggerStageEnter:
		return "进入阶段触发"
	default:
		return "手动触发"
	}
}

func ensureDefaultStage(ctx context.Context) {
	ensureDefaultStageOnce.Do(func() {
		stageModel := crmmodel.NewStageModel()
		if stageModel.Find(ctx, map[string]any{"id": crmmodel.DefaultStageID}) != nil {
			return
		}
		if stageModel.Find(ctx, map[string]any{"code": crmmodel.DefaultStageCode}) != nil {
			return
		}
		stageModel.Insert(ctx, crmmodel.DefaultStageRecord())
	})
}

func (CrmHook) ProviderBeforeSaveTask(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "task_type", partial)
	trimCrmStringField(record, "form_mode", partial)
	trimCrmStringField(record, "trigger_type", partial)
	trimCrmStringField(record, "assign_mode", partial)
	trimCrmStringField(record, "next_stage_code", partial)
	trimCrmStringField(record, "booking_need_confirm", partial)
	if shouldNormalizeCrmField(record, "result_schema_json", partial) {
		record["result_schema_json"] = encodeTaskResultSchema(record["result_schema_json"])
	}
	if !partial {
		if util.ToUint64(record["stage_id"]) == 0 {
			panicCrmField("form.stage_id", "所属阶段不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "任务名称不能为空。")
		}
	}
	ensureTaskNextStageExists(contextFromServer(c), record, partial)
	ensureUniqueTaskName(contextFromServer(c), record, partial)
	defaultCrmInt(record, "stage_id", 0, partial)
	if shouldNormalizeCrmField(record, "task_type", partial) && util.ToStringTrimmed(record["task_type"]) == "" {
		record["task_type"] = crmmodel.TaskTypeForm
	}
	if shouldNormalizeCrmField(record, "form_mode", partial) && util.ToStringTrimmed(record["form_mode"]) == "" {
		record["form_mode"] = crmmodel.TaskFormModeEdit
	}
	if shouldNormalizeCrmField(record, "trigger_type", partial) && util.ToStringTrimmed(record["trigger_type"]) == "" {
		record["trigger_type"] = crmmodel.TaskTriggerManual
	}
	normalizeTaskTriggerConfig(record, partial)
	normalizeTaskTypeConfig(record, partial)
	ensureTaskFormExists(contextFromServer(c), record, partial)
	defaultCrmInt(record, "form_id", 0, partial)
	defaultCrmInt(record, "script_id", 0, partial)
	defaultCrmInt(record, "trigger_task_id", 0, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func ensureUniqueTaskName(ctx context.Context, record map[string]any, partial bool) {
	taskID := util.ToUint64(record["id"])
	stageID := util.ToUint64(record["stage_id"])
	name := util.ToStringTrimmed(record["name"])

	if partial && taskID > 0 && (stageID == 0 || name == "") {
		if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": taskID}); current != nil {
			if stageID == 0 {
				stageID = current.StageID
			}
			if name == "" {
				name = current.Name
			}
		}
	}

	if stageID == 0 || name == "" {
		return
	}

	filters := map[string]any{
		"stage_id": stageID,
		"name":     name,
	}
	if taskID > 0 {
		filters["id"] = map[string]any{"!=": taskID}
	}
	if crmmodel.NewTaskModel().Count(ctx, filters) > 0 {
		panicCrmField("form.name", "当前阶段下任务名称已存在。")
	}
}

func normalizeTaskTriggerConfig(record map[string]any, partial bool) {
	if !shouldNormalizeCrmField(record, "trigger_type", partial) && !shouldNormalizeCrmField(record, "trigger_task_id", partial) {
		return
	}
	triggerType := util.ToStringTrimmed(record["trigger_type"])
	if triggerType == "" {
		triggerType = crmmodel.TaskTriggerManual
		record["trigger_type"] = triggerType
	}
	if triggerType == crmmodel.TaskTriggerManual || triggerType == crmmodel.TaskTriggerStageEnter {
		record["trigger_task_id"] = uint64(0)
		return
	}
	if !partial && triggerType == crmmodel.TaskTriggerAfterTask && util.ToUint64(record["trigger_task_id"]) == 0 {
		panicCrmField("form.trigger_task_id", "任务后触发必须选择触发任务。")
	}
}

func normalizeTaskTypeConfig(record map[string]any, partial bool) {
	if !shouldNormalizeTaskConfig(record, partial) {
		return
	}
	taskType := util.ToStringTrimmed(record["task_type"])
	switch taskType {
	case crmmodel.TaskTypeForm:
		if !partial && util.ToUint64(record["form_id"]) == 0 {
			panicCrmField("form.form_id", "填写资料任务必须选择资料模板。")
		}
		record["script_id"] = uint64(0)
		record["result_schema_json"] = "[]"
		mergeTaskConfigField(record, "next_stage_code", util.ToStringTrimmed(record["next_stage_code"]))
	case crmmodel.TaskTypeAssign:
		record["script_id"] = uint64(0)
		record["result_schema_json"] = "[]"
		assignMode := util.ToStringTrimmed(record["assign_mode"])
		if assignMode == "" {
			assignMode = "department_staff"
		}
		record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, map[string]any{
			"assign_mode":     assignMode,
			"next_stage_code": util.ToStringTrimmed(record["next_stage_code"]),
		}))
	case crmmodel.TaskTypeBooking:
		record["form_id"] = uint64(0)
		record["form_mode"] = crmmodel.TaskFormModeEdit
		record["script_id"] = uint64(0)
		record["result_schema_json"] = "[]"
		resourceCateID := util.ToUint64(record["booking_resource_cate_id"])
		if resourceCateID == 0 {
			resourceCateID = crmmodel.DefaultResourceCateID
		}
		record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, map[string]any{
			"resource_cate_id": resourceCateID,
			"need_confirm":     util.ToBool(record["booking_need_confirm"]),
			"next_stage_code":  util.ToStringTrimmed(record["next_stage_code"]),
		}))
	case crmmodel.TaskTypeDecision:
		hasScript := util.ToUint64(record["script_id"]) > 0
		if !partial && !hasScript && len(taskResultSchemaRows(record["result_schema_json"])) == 0 {
			panicCrmField("form.result_schema_json", "决策任务必须配置任务结果。")
		}
		record["form_id"] = uint64(0)
		record["form_mode"] = crmmodel.TaskFormModeEdit
		mergeTaskConfigField(record, "next_stage_code", util.ToStringTrimmed(record["next_stage_code"]))
	default:
		record["task_type"] = crmmodel.TaskTypeForm
		record["script_id"] = uint64(0)
		record["result_schema_json"] = "[]"
		mergeTaskConfigField(record, "next_stage_code", util.ToStringTrimmed(record["next_stage_code"]))
	}
}

func ensureTaskFormExists(ctx context.Context, record map[string]any, partial bool) {
	taskType := util.ToStringTrimmed(record["task_type"])
	formID := util.ToUint64(record["form_id"])
	if partial && (taskType == "" || formID == 0) {
		if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
			if taskType == "" {
				taskType = current.TaskType
			}
			if formID == 0 {
				formID = current.FormID
			}
		}
	}
	if taskType != crmmodel.TaskTypeForm && taskType != crmmodel.TaskTypeAssign {
		return
	}
	if formID == 0 {
		return
	}
	form := crmmodel.NewFormModel().Find(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled})
	if form == nil {
		panicCrmField("form.form_id", "资料模板不存在或已停用。")
	}
}

func shouldNormalizeTaskConfig(record map[string]any, partial bool) bool {
	for _, field := range []string{
		"task_type",
		"form_mode",
		"form_id",
		"assign_mode",
		"next_stage_code",
		"booking_resource_cate_id",
		"booking_need_confirm",
		"script_id",
		"result_schema_json",
	} {
		if shouldNormalizeCrmField(record, field, partial) {
			return true
		}
	}
	return false
}

func ensureTaskNextStageExists(ctx context.Context, record map[string]any, partial bool) {
	if !shouldNormalizeCrmField(record, "next_stage_code", partial) {
		return
	}
	nextStageCode := util.ToStringTrimmed(record["next_stage_code"])
	if nextStageCode == "" {
		return
	}
	if crmmodel.NewStageModel().Find(ctx, map[string]any{"code": nextStageCode, "status": crmmodel.StatusEnabled}) == nil {
		panicCrmField("form.next_stage_code", "完成后阶段不存在或已停用。")
	}
}

func mergedTaskConfig(record map[string]any, updates map[string]any) map[string]any {
	config := decodeTaskConfig(record["config_json"])
	for key, value := range updates {
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			delete(config, key)
			continue
		}
		config[key] = value
	}
	return config
}

func mergeTaskConfigField(record map[string]any, key string, value any) {
	record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, map[string]any{key: value}))
}

func encodeTaskConfig(config map[string]any) string {
	encoded, err := json.Marshal(config)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func (CrmHook) ProviderAfterSaveTask(_ *server.Context, _ []any) any {
	return nil
}

func (CrmHook) ProviderBuildTaskForm(_ *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	applyTaskConfigForm(record)
	applyTaskResultSchemaForm(record)
	return record
}

func applyTaskConfigForm(record map[string]any) {
	if util.ToStringTrimmed(record["task_type"]) == "" {
		record["task_type"] = crmmodel.TaskTypeForm
	}
	if util.ToStringTrimmed(record["form_mode"]) == "" {
		record["form_mode"] = crmmodel.TaskFormModeEdit
	}
	config := decodeTaskConfig(record["config_json"])
	record["next_stage_code"] = util.ToStringTrimmed(config["next_stage_code"])
	assignMode := util.ToStringTrimmed(config["assign_mode"])
	if assignMode == "" {
		assignMode = "department_staff"
	}
	record["assign_mode"] = assignMode
	resourceCateID := util.ToUint64(config["resource_cate_id"])
	if resourceCateID == 0 {
		resourceCateID = crmmodel.DefaultResourceCateID
	}
	record["booking_resource_cate_id"] = resourceCateID
	record["booking_need_confirm"] = util.ToBool(config["need_confirm"])
}

func applyTaskResultSchemaForm(record map[string]any) {
	record["result_schema_json"] = taskResultSchemaRowsForForm(record["result_schema_json"])
}

func decodeTaskConfig(value any) map[string]any {
	raw := util.ToStringTrimmed(value)
	if raw == "" {
		return map[string]any{}
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return map[string]any{}
	}
	return decoded
}

func encodeTaskResultSchema(value any) string {
	rows := taskResultSchemaRows(value)
	if len(rows) == 0 {
		return "[]"
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func taskResultSchemaRowsForForm(value any) []any {
	rows := taskResultSchemaRows(value)
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, row)
	}
	return result
}

func taskResultSchemaRows(value any) []map[string]any {
	switch rows := value.(type) {
	case []map[string]any:
		return normalizeTaskResultSchemaRows(rows)
	case []any:
		result := make([]map[string]any, 0, len(rows))
		for _, item := range rows {
			if row, ok := item.(map[string]any); ok {
				result = append(result, row)
			}
		}
		return normalizeTaskResultSchemaRows(result)
	case string:
		raw := strings.TrimSpace(rows)
		if raw == "" {
			return nil
		}
		var mappedRows []map[string]any
		if err := json.Unmarshal([]byte(raw), &mappedRows); err == nil {
			return normalizeTaskResultSchemaRows(mappedRows)
		}
		var anyRows []any
		if err := json.Unmarshal([]byte(raw), &anyRows); err == nil {
			return taskResultSchemaRows(anyRows)
		}
		return nil
	default:
		return nil
	}
}

func normalizeTaskResultSchemaRows(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	seen := map[string]bool{}
	for index, row := range rows {
		name := util.ToStringTrimmed(row["name"])
		resultValue := util.ToStringTrimmed(row["result_value"])
		if name == "" && resultValue == "" {
			continue
		}
		if name == "" {
			name = resultValue
		}
		if resultValue == "" {
			resultValue = name
		}
		if seen[resultValue] {
			continue
		}
		seen[resultValue] = true
		sortValue := util.ToIntDefault(row["sort"], 0)
		if sortValue == 0 {
			sortValue = (index + 1) * 10
		}
		result = append(result, map[string]any{
			"name":             name,
			"result_value":     resultValue,
			"is_success":       util.ToBool(row["is_success"]),
			"requires_comment": util.ToBool(row["requires_comment"]),
			"sort":             sortValue,
		})
	}
	return result
}
