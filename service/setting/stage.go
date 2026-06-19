package setting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSaveStage(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "code", partial)
	trimCrmStringField(record, "name", partial)
	if !partial && util.ToStringTrimmed(record["code"]) == "" {
		panicCrmField("form.code", "状态码不能为空。")
	}
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "阶段名称不能为空。")
	}
	defaultCrmInt(record, "owner_department_id", 0, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBeforeSaveStageTransition(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "from_stage_code", partial)
	trimCrmStringField(record, "to_stage_code", partial)
	trimCrmStringField(record, "result_value", partial)
	trimCrmStringField(record, "owner_mode", partial)
	normalizeStageTransitionCondition(contextFromServer(c), record, partial)
	if shouldNormalizeCrmField(record, "condition_json", partial) {
		record["condition_json"] = normalizeConditionJSON(record["condition_json"])
	}
	if !partial && util.ToStringTrimmed(record["from_stage_code"]) == "" {
		panicCrmField("form.from_stage_code", "来源阶段不能为空。")
	}
	if !partial && util.ToStringTrimmed(record["to_stage_code"]) == "" {
		panicCrmField("form.to_stage_code", "目标阶段不能为空。")
	}
	if shouldNormalizeCrmField(record, "owner_mode", partial) && util.ToStringTrimmed(record["owner_mode"]) == "" {
		record["owner_mode"] = crmmodel.StageOwnerKeep
	}
	defaultCrmInt(record, "task_id", 0, partial)
	defaultCrmInt(record, "script_id", 0, partial)
	defaultCrmInt(record, "to_department_id", 0, partial)
	defaultCrmInt(record, "to_staff_id", 0, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func normalizeStageTransitionCondition(ctx context.Context, record map[string]any, partial bool) {
	if !shouldNormalizeCrmField(record, "condition_path", partial) {
		return
	}
	fieldID, value := conditionFieldAndValueFromPath(record["condition_path"])
	if fieldID == 0 {
		fieldID = conditionDataFieldID(record["condition_path"])
	}
	if fieldID == 0 {
		if hasStageTransitionConditionJSON(record["condition_json"]) {
			return
		}
		record["condition_json"] = "{}"
		return
	}
	field := requireConditionDataField(ctx, fieldID, "form.condition_path", "流转条件字段")
	if strings.TrimSpace(field.FieldKey) == "" {
		panicCrmField("form.condition_path", "流转条件字段必须配置字段编码。")
	}
	if value == "" {
		panicCrmField("form.condition_path", "流转条件值不能为空。")
	}
	requireConditionDataFieldOption(ctx, field, value, "form.condition_path", "流转条件值")
	record["condition_json"] = encodeStageTransitionCondition(ctx, field, value)
}

func hasStageTransitionConditionJSON(value any) bool {
	raw := util.ToStringTrimmed(value)
	return raw != "" && raw != "{}" && raw != "[]"
}

func encodeStageTransitionCondition(ctx context.Context, field *crmmodel.DataField, value string) string {
	if field == nil || strings.TrimSpace(field.FieldKey) == "" || strings.TrimSpace(value) == "" {
		return "{}"
	}
	condition := map[string]any{
		"field":         stageTransitionConditionFieldPath(ctx, field),
		"operator":      "equals",
		"value":         strings.TrimSpace(value),
		"data_field_id": field.ID,
	}
	encoded, err := json.Marshal(condition)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func stageTransitionConditionFieldPath(ctx context.Context, field *crmmodel.DataField) string {
	if field == nil {
		return ""
	}
	if stageTransitionFieldCateID(ctx, field) == crmmodel.CustomerAssetDataTemplateCateID {
		return "current.asset.fields." + strings.TrimSpace(field.FieldKey)
	}
	return "customer.fields." + strings.TrimSpace(field.FieldKey)
}

func normalizeConditionJSON(value any) string {
	raw := util.ToStringTrimmed(value)
	if raw == "" {
		return "{}"
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		panicCrmField("form.condition_json", fmt.Sprintf("条件JSON格式错误：%s", err.Error()))
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func (CrmHook) ProviderBuildStageTransitionForm(c *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	applyStageTransitionConditionForm(contextFromServer(c), record)
	return record
}

func applyStageTransitionConditionForm(ctx context.Context, record map[string]any) {
	fieldPath, value := stageTransitionConditionFromJSON(ctx, record["condition_json"])
	if fieldPath == "" || value == "" {
		record["condition_path"] = []any{}
		return
	}
	field := stageTransitionConditionField(ctx, fieldPath)
	if field == nil {
		record["condition_path"] = []any{}
		return
	}
	record["condition_path"] = conditionFieldPath(ctx, field.ID, value, true)
}

func stageTransitionConditionFromJSON(ctx context.Context, value any) (string, string) {
	raw := util.ToStringTrimmed(value)
	if raw == "" || raw == "{}" || raw == "[]" {
		return "", ""
	}
	condition := decodeTaskConfig(raw)
	if len(condition) == 0 {
		rows := taskConfigRows(raw)
		if len(rows) == 0 {
			return "", ""
		}
		condition = rows[0]
	}
	if rows := taskConfigRows(condition["conditions"]); len(rows) > 0 {
		condition = rows[0]
	}
	if rows := taskConfigRows(condition["any"]); len(rows) > 0 {
		condition = rows[0]
	}
	if rows := taskConfigRows(condition["all"]); len(rows) > 0 {
		condition = rows[0]
	}
	if fieldID := util.ToUint64(condition["data_field_id"]); fieldID > 0 {
		field := conditionDataField(ctx, fieldID)
		valueText := util.ToStringTrimmed(firstTaskConfigValue(condition, "value", "expected"))
		if field != nil && valueText != "" {
			return stageTransitionConditionFieldPath(ctx, field), valueText
		}
	}
	fieldPath := util.ToStringTrimmed(firstTaskConfigValue(condition, "field", "path"))
	valueText := util.ToStringTrimmed(firstTaskConfigValue(condition, "value", "expected"))
	if fieldPath == "" || valueText == "" {
		return "", ""
	}
	if stageTransitionConditionField(ctx, fieldPath) == nil {
		return "", ""
	}
	return fieldPath, valueText
}

func stageTransitionConditionField(ctx context.Context, path string) *crmmodel.DataField {
	prefixes := []string{"customer.fields.", "current.asset.fields."}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			fieldKey := strings.TrimSpace(strings.TrimPrefix(path, prefix))
			if fieldKey == "" {
				return nil
			}
			cateID := crmmodel.CustomerDataTemplateCateID
			if prefix == "current.asset.fields." {
				cateID = crmmodel.CustomerAssetDataTemplateCateID
			}
			for _, field := range crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
				"field_key":    fieldKey,
				"stat_enabled": true,
				"status":       crmmodel.StatusEnabled,
			}) {
				if stageTransitionFieldCateID(ctx, field) == cateID {
					return field
				}
			}
			return nil
		}
	}
	return nil
}

func stageTransitionFieldCateID(ctx context.Context, field *crmmodel.DataField) uint64 {
	if field == nil || field.DataTemplateID == 0 {
		return 0
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": field.DataTemplateID})
	if template == nil {
		return 0
	}
	return template.CateID
}

func (CrmHook) ProviderBuildStageRows(c *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}
	namesByStageID := taskNamesByStageID(contextFromServer(c), rows)
	for _, row := range rows {
		stageID := util.ToUint64(row["id"])
		row["task_names"] = strings.Join(namesByStageID[stageID], "、")
		row["task_count"] = len(namesByStageID[stageID])
	}
	return rows
}

func (CrmHook) ProviderBuildStageTransitionRows(_ *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}
	for _, row := range rows {
		row["owner_mode_name"] = stageOwnerModeName(row["owner_mode"])
		if util.ToStringTrimmed(row["result_value"]) == "" {
			row["result_value_display"] = "任意结果"
		} else {
			row["result_value_display"] = row["result_value"]
		}
	}
	return rows
}

func stageOwnerModeName(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.StageOwnerAssign:
		return "使用分配结果"
	case crmmodel.StageOwnerFixedDepartment:
		return "固定部门"
	case crmmodel.StageOwnerFixedStaff:
		return "固定人员"
	case crmmodel.StageOwnerCreator:
		return "创建人"
	default:
		return "保持当前"
	}
}

func taskNamesByStageID(ctx context.Context, rows []map[string]any) map[uint64][]string {
	stageIDs := make(map[uint64]bool)
	for _, row := range rows {
		stageID := util.ToUint64(row["id"])
		if stageID > 0 {
			stageIDs[stageID] = true
		}
	}
	if len(stageIDs) == 0 {
		return map[uint64][]string{}
	}
	result := make(map[uint64][]string, len(stageIDs))
	for _, task := range crmmodel.NewTaskModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled}) {
		if task == nil || task.StageID == 0 || !stageIDs[task.StageID] {
			continue
		}
		name := strings.TrimSpace(task.Name)
		if name != "" {
			result[task.StageID] = append(result[task.StageID], name)
		}
	}
	return result
}

func contextFromServer(c *server.Context) context.Context {
	if c == nil {
		return context.Background()
	}
	return c.Context()
}
