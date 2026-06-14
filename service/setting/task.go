package setting

import (
	"context"
	"encoding/json"
	"fmt"
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
		row["trigger_type_name"] = taskTriggerTypeName(row["trigger_type"])
	}
	return rows
}

func taskTypeName(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.TaskTypeCreate:
		return "创建资料"
	case crmmodel.TaskTypeForm:
		return "填写资料"
	case crmmodel.TaskTypeAssign:
		return "分配"
	case crmmodel.TaskTypeCollaborate:
		return "协作任务"
	case crmmodel.TaskTypeDecision:
		return "决策"
	case crmmodel.TaskTypeBooking:
		return "资源预定"
	default:
		return util.ToStringTrimmed(value)
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
	trimCrmStringField(record, "trigger_type", partial)
	trimCrmStringField(record, "assign_mode", partial)
	trimCrmStringField(record, "next_stage_code", partial)
	trimCrmStringField(record, "booking_need_confirm", partial)
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
	normalizeTaskVisibleWhen(contextFromServer(c), record, partial)
	defaultCrmInt(record, "stage_id", 0, partial)
	if shouldNormalizeCrmField(record, "task_type", partial) && util.ToStringTrimmed(record["task_type"]) == "" {
		record["task_type"] = crmmodel.TaskTypeForm
	}
	if shouldNormalizeCrmField(record, "trigger_type", partial) && util.ToStringTrimmed(record["trigger_type"]) == "" {
		record["trigger_type"] = crmmodel.TaskTriggerManual
	}
	normalizeTaskTriggerConfig(record, partial)
	normalizeTaskTypeConfig(contextFromServer(c), record, partial)
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

func normalizeTaskTypeConfig(ctx context.Context, record map[string]any, partial bool) {
	if !shouldNormalizeTaskConfig(record, partial) {
		return
	}
	taskType := effectiveTaskType(ctx, record, partial)
	switch taskType {
	case crmmodel.TaskTypeCreate:
		if !partial && util.ToUint64(record["form_id"]) == 0 {
			panicCrmField("form.form_id", "创建资料任务必须选择资料模板。")
		}
		record["script_id"] = uint64(0)
		mergeTaskConfigField(record, "next_stage_code", util.ToStringTrimmed(record["next_stage_code"]))
	case crmmodel.TaskTypeForm:
		if !partial && util.ToUint64(record["form_id"]) == 0 {
			panicCrmField("form.form_id", "填写资料任务必须选择资料模板。")
		}
		record["script_id"] = uint64(0)
		mergeTaskConfigField(record, "next_stage_code", util.ToStringTrimmed(record["next_stage_code"]))
	case crmmodel.TaskTypeAssign:
		record["script_id"] = uint64(0)
		assignMode := util.ToStringTrimmed(record["assign_mode"])
		if assignMode == "" {
			assignMode = crmmodel.TaskAssignModeStaff
		}
		if assignMode != crmmodel.TaskAssignModeDepartment {
			assignMode = crmmodel.TaskAssignModeStaff
		}
		departmentIDs := normalizeTaskAssignDepartmentIDs(record["assign_department_ids"])
		if len(departmentIDs) == 0 {
			panicCrmField("form.assign_department_ids", "可选部门不能为空。")
		}
		record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, map[string]any{
			"assign_mode":           assignMode,
			"assign_department_ids": departmentIDs,
			"next_stage_code":       util.ToStringTrimmed(record["next_stage_code"]),
		}))
		record["form_id"] = uint64(0)
	case crmmodel.TaskTypeCollaborate:
		record["script_id"] = uint64(0)
		record["form_id"] = uint64(0)
		items := normalizeTaskCollaborationItems(ctx, record["collaboration_items"], partial)
		if !partial && len(items) == 0 {
			panicCrmField("form.collaboration_items", "协作任务必须配置子任务。")
		}
		completeMode := normalizeTaskCollaborationCompleteMode(record["collaboration_complete_mode"])
		record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, map[string]any{
			"collaboration_items":         items,
			"collaboration_complete_mode": completeMode,
			"next_stage_code":             util.ToStringTrimmed(record["next_stage_code"]),
		}))
	case crmmodel.TaskTypeBooking:
		record["form_id"] = uint64(0)
		record["script_id"] = uint64(0)
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
		if decisionTaskIsManual(ctx, record, partial) {
			record["script_id"] = uint64(0)
		} else if effectiveTaskScriptID(ctx, record, partial) == 0 {
			panicCrmField("form.script_id", "自动触发的决策任务必须选择脚本规则。")
		}
		resultFieldID := normalizeDecisionResultField(ctx, record, partial)
		record["form_id"] = uint64(0)
		record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, map[string]any{
			"decision_result_field_id": resultFieldID,
			"next_stage_code":          util.ToStringTrimmed(record["next_stage_code"]),
		}))
	default:
		record["task_type"] = crmmodel.TaskTypeForm
		record["script_id"] = uint64(0)
		mergeTaskConfigField(record, "next_stage_code", util.ToStringTrimmed(record["next_stage_code"]))
	}
}

func effectiveTaskType(ctx context.Context, record map[string]any, partial bool) string {
	taskType := util.ToStringTrimmed(record["task_type"])
	if taskType != "" || !partial {
		return taskType
	}
	if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
		return current.TaskType
	}
	return ""
}

func decisionTaskIsManual(ctx context.Context, record map[string]any, partial bool) bool {
	return effectiveTaskTriggerType(ctx, record, partial) == crmmodel.TaskTriggerManual
}

func effectiveTaskScriptID(ctx context.Context, record map[string]any, partial bool) uint64 {
	if scriptID := util.ToUint64(record["script_id"]); scriptID > 0 || !partial {
		return scriptID
	}
	if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
		return current.ScriptID
	}
	return 0
}

func effectiveTaskTriggerType(ctx context.Context, record map[string]any, partial bool) string {
	triggerType := util.ToStringTrimmed(record["trigger_type"])
	if triggerType != "" || !partial {
		return triggerType
	}
	if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
		return current.TriggerType
	}
	return crmmodel.TaskTriggerManual
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
	if taskType != crmmodel.TaskTypeCreate && taskType != crmmodel.TaskTypeForm {
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
		"form_id",
		"assign_mode",
		"assign_department_ids",
		"collaboration_items",
		"collaboration_complete_mode",
		"next_stage_code",
		"trigger_type",
		"booking_resource_cate_id",
		"booking_need_confirm",
		"script_id",
		"decision_result_field_path",
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
		if value == nil {
			delete(config, key)
			continue
		}
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

func normalizeTaskVisibleWhen(ctx context.Context, record map[string]any, partial bool) {
	if !shouldNormalizeCrmField(record, "visible_condition_path", partial) &&
		!shouldNormalizeCrmField(record, "visible_field_path", partial) &&
		!shouldNormalizeCrmField(record, "visible_value", partial) {
		return
	}
	fieldID, value := taskVisibleConditionFromPath(record["visible_condition_path"])
	if fieldID == 0 {
		fieldID = visibleDataFieldID(record["visible_field_path"])
		value = util.ToStringTrimmed(record["visible_value"])
	}
	if fieldID == 0 {
		record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, map[string]any{
			"visible_when": nil,
		}))
		return
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":           fieldID,
		"stat_enabled": true,
		"status":       crmmodel.StatusEnabled,
	})
	if field == nil {
		panicCrmField("form.visible_condition_path", "显示条件字段不存在、未开启条件字段或已停用。")
	}
	if value == "" {
		panicCrmField("form.visible_condition_path", "显示条件值不能为空。")
	}
	if taskVisibleFieldHasOptions(field.FieldType) && crmmodel.NewDataFieldOptionModel().Find(ctx, map[string]any{
		"data_field_id": field.ID,
		"value":         value,
	}) == nil {
		panicCrmField("form.visible_condition_path", "显示条件值不属于该字段的可选项。")
	}
	record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, map[string]any{
		"visible_when": map[string]any{
			"data_field_id": field.ID,
			"op":            "eq",
			"value":         value,
		},
	}))
}

func taskVisibleConditionFromPath(value any) (uint64, string) {
	for _, item := range collectPathItems(value) {
		fieldID, optionValue, ok := parseTaskVisibleValueSource(item)
		if ok {
			return fieldID, optionValue
		}
	}
	return 0, ""
}

func visibleDataFieldID(value any) uint64 {
	for _, item := range collectPathItems(value) {
		if strings.HasPrefix(item, collectFieldSourceDataPrefix) {
			return util.ToUint64(strings.TrimPrefix(item, collectFieldSourceDataPrefix))
		}
	}
	return util.ToUint64(value)
}

func taskVisibleFieldHasOptions(fieldType string) bool {
	switch strings.TrimSpace(fieldType) {
	case "radio", "select":
		return true
	default:
		return false
	}
}

func (CrmHook) ProviderAfterSaveTask(_ *server.Context, _ []any) any {
	return nil
}

func (CrmHook) ProviderBuildTaskForm(c *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	applyTaskConfigForm(contextFromServer(c), record)
	return record
}

func applyTaskConfigForm(ctx context.Context, record map[string]any) {
	if util.ToStringTrimmed(record["task_type"]) == "" {
		record["task_type"] = crmmodel.TaskTypeForm
	}
	config := decodeTaskConfig(record["config_json"])
	record["next_stage_code"] = util.ToStringTrimmed(config["next_stage_code"])
	assignMode := util.ToStringTrimmed(config["assign_mode"])
	if assignMode == "" {
		assignMode = crmmodel.TaskAssignModeStaff
	}
	if assignMode != crmmodel.TaskAssignModeDepartment {
		assignMode = crmmodel.TaskAssignModeStaff
	}
	record["assign_mode"] = assignMode
	record["assign_department_ids"] = normalizeTaskAssignDepartmentIDs(config["assign_department_ids"])
	record["collaboration_items"] = normalizeTaskCollaborationItems(nil, config["collaboration_items"], true)
	record["collaboration_complete_mode"] = normalizeTaskCollaborationCompleteMode(config["collaboration_complete_mode"])
	resourceCateID := util.ToUint64(config["resource_cate_id"])
	if resourceCateID == 0 {
		resourceCateID = crmmodel.DefaultResourceCateID
	}
	record["booking_resource_cate_id"] = resourceCateID
	record["booking_need_confirm"] = util.ToBool(config["need_confirm"])
	applyDecisionResultFieldForm(ctx, record, config)
	applyTaskVisibleWhenForm(ctx, record, config)
}

func normalizeDecisionResultField(ctx context.Context, record map[string]any, partial bool) uint64 {
	fieldID := visibleDataFieldID(record["decision_result_field_path"])
	if partial && fieldID == 0 && !shouldNormalizeCrmField(record, "decision_result_field_path", partial) {
		config := decodeTaskConfig(record["config_json"])
		if len(config) == 0 {
			if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
				config = decodeTaskConfig(current.ConfigJSON)
			}
		}
		fieldID = util.ToUint64(config["decision_result_field_id"])
	}
	if fieldID == 0 {
		panicCrmField("form.decision_result_field_path", "决策任务必须选择结果写入字段。")
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":           fieldID,
		"stat_enabled": true,
		"status":       crmmodel.StatusEnabled,
	})
	if field == nil {
		panicCrmField("form.decision_result_field_path", "结果写入字段不存在、未开启条件字段或已停用。")
	}
	if strings.TrimSpace(field.FieldKey) == "" {
		panicCrmField("form.decision_result_field_path", "结果写入字段必须配置字段编码。")
	}
	if !taskVisibleFieldHasOptions(field.FieldType) {
		panicCrmField("form.decision_result_field_path", "结果写入字段必须是单选或下拉字段。")
	}
	if crmmodel.NewDataFieldOptionModel().Count(ctx, map[string]any{"data_field_id": field.ID}) == 0 {
		panicCrmField("form.decision_result_field_path", "结果写入字段必须配置可选项。")
	}
	return field.ID
}

func applyDecisionResultFieldForm(ctx context.Context, record map[string]any, config map[string]any) {
	fieldID := util.ToUint64(config["decision_result_field_id"])
	if fieldID == 0 {
		record["decision_result_field_path"] = []any{}
		return
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
	if field == nil {
		record["decision_result_field_path"] = []any{}
		return
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": field.DataTemplateID})
	if template == nil {
		record["decision_result_field_path"] = []any{}
		return
	}
	record["decision_result_field_path"] = []any{
		fmt.Sprintf("cate:%d", template.CateID),
		collectDataTemplateSource(template.ID),
		fmt.Sprintf("%s%d", collectFieldSourceDataPrefix, field.ID),
	}
}

func applyTaskVisibleWhenForm(ctx context.Context, record map[string]any, config map[string]any) {
	visibleWhen := taskConfigObject(config["visible_when"])
	fieldID := util.ToUint64(visibleWhen["data_field_id"])
	if fieldID == 0 {
		record["visible_condition_path"] = []any{}
		return
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
	if field == nil {
		record["visible_condition_path"] = []any{}
		return
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": field.DataTemplateID})
	if template == nil {
		record["visible_condition_path"] = []any{}
		return
	}
	value := util.ToStringTrimmed(visibleWhen["value"])
	record["visible_condition_path"] = []any{
		fmt.Sprintf("cate:%d", template.CateID),
		collectDataTemplateSource(template.ID),
		fmt.Sprintf("%s%d", collectFieldSourceDataPrefix, field.ID),
		taskVisibleValueSource(field.ID, value),
	}
}

func taskConfigObject(value any) map[string]any {
	if row, ok := value.(map[string]any); ok {
		return row
	}
	return decodeTaskConfig(value)
}

func normalizeTaskAssignDepartmentIDs(value any) []uint64 {
	items := compactStringList(value)
	result := make([]uint64, 0, len(items))
	seen := map[uint64]bool{}
	for _, item := range items {
		id := util.ToUint64(item)
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}

func normalizeTaskCollaborationCompleteMode(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.CollaborationCompleteAny:
		return crmmodel.CollaborationCompleteAny
	case crmmodel.CollaborationCompleteManual:
		return crmmodel.CollaborationCompleteManual
	default:
		return crmmodel.CollaborationCompleteAll
	}
}

func normalizeTaskCollaborationItems(ctx context.Context, value any, partial bool) []map[string]any {
	rows := taskConfigRows(value)
	result := make([]map[string]any, 0, len(rows))
	seen := map[string]bool{}
	for index, row := range rows {
		name := util.ToStringTrimmed(firstTaskConfigValue(row, "name", "task_name", "sub_task_name"))
		departmentID := util.ToUint64(firstTaskConfigValue(row, "department_id", "assignee_department_id"))
		staffID := util.ToUint64(firstTaskConfigValue(row, "staff_id", "assignee_staff_id"))
		formID := util.ToUint64(firstTaskConfigValue(row, "form_id"))
		if name == "" && departmentID == 0 && staffID == 0 && formID == 0 {
			continue
		}
		if name == "" {
			panicCrmField("form.collaboration_items", "协作子任务名称不能为空。")
		}
		if departmentID == 0 {
			panicCrmField("form.collaboration_items", "协作子任务目标部门不能为空。")
		}
		if ctx != nil && crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": departmentID, "status": crmmodel.StatusEnabled}) == nil {
			panicCrmField("form.collaboration_items", "协作子任务目标部门不存在或已停用。")
		}
		if staffID > 0 && ctx != nil {
			staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID, "status": crmmodel.StatusEnabled})
			if staff == nil {
				panicCrmField("form.collaboration_items", "协作子任务处理人员不存在或已停用。")
			}
			if staff != nil && staff.DepartmentID != departmentID {
				panicCrmField("form.collaboration_items", "协作子任务处理人员不属于目标部门。")
			}
		}
		if formID > 0 && ctx != nil && crmmodel.NewFormModel().Find(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled}) == nil {
			panicCrmField("form.collaboration_items", "协作子任务资料模板不存在或已停用。")
		}
		key := name + ":" + util.ToString(departmentID) + ":" + util.ToString(staffID)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, map[string]any{
			"name":          name,
			"department_id": departmentID,
			"staff_id":      staffID,
			"form_id":       formID,
			"required":      taskConfigBoolDefault(row["required"], true),
			"sort":          taskConfigSort(row["sort"], index),
		})
	}
	return result
}

func firstTaskConfigValue(row map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			return value
		}
	}
	return nil
}

func taskConfigBoolDefault(value any, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return util.ToBool(value)
}

func taskConfigSort(value any, index int) int {
	sort := util.ToIntDefault(value, 0)
	if sort == 0 {
		return (index + 1) * 10
	}
	return sort
}

func decodeTaskConfig(value any) map[string]any {
	if row, ok := value.(map[string]any); ok {
		return util.CloneMap(row)
	}
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

func taskConfigRows(value any) []map[string]any {
	switch rows := value.(type) {
	case []map[string]any:
		return rows
	case []any:
		result := make([]map[string]any, 0, len(rows))
		for _, item := range rows {
			if row, ok := item.(map[string]any); ok {
				result = append(result, row)
			}
		}
		return result
	case string:
		raw := strings.TrimSpace(rows)
		if raw == "" {
			return nil
		}
		var mappedRows []map[string]any
		if err := json.Unmarshal([]byte(raw), &mappedRows); err == nil {
			return mappedRows
		}
		var anyRows []any
		if err := json.Unmarshal([]byte(raw), &anyRows); err == nil {
			return taskConfigRows(anyRows)
		}
		return nil
	default:
		return nil
	}
}
