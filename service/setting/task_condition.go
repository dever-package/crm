package setting

import (
	"context"
	"fmt"
	"strings"

	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func normalizeTaskVisibleWhen(ctx context.Context, record map[string]any, partial bool) {
	if !shouldNormalizeCrmField(record, "visible_condition_path", partial) &&
		!shouldNormalizeCrmField(record, "visible_field_path", partial) &&
		!shouldNormalizeCrmField(record, "visible_value", partial) {
		return
	}
	fieldID, value := conditionFieldAndValueFromPath(record["visible_condition_path"])
	if fieldID == 0 {
		fieldID = conditionDataFieldID(record["visible_field_path"])
		value = util.ToStringTrimmed(record["visible_value"])
	}
	if fieldID == 0 {
		record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, map[string]any{
			"visible_when": nil,
		}))
		return
	}
	field := requireConditionDataField(ctx, fieldID, "form.visible_condition_path", "显示条件字段")
	if field == nil {
		panicCrmField("form.visible_condition_path", "显示条件字段不存在或已停用。")
	}
	if value == "" {
		panicCrmField("form.visible_condition_path", "显示条件值不能为空。")
	}
	requireConditionDataFieldOption(ctx, field, value, "form.visible_condition_path", "显示条件值")
	record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, map[string]any{
		"visible_when": map[string]any{
			"data_field_id": field.ID,
			"op":            "eq",
			"value":         value,
		},
	}))
}

func conditionFieldAndValueFromPath(value any) (uint64, string) {
	for _, item := range collectPathItems(value) {
		fieldID, optionValue, ok := parseTaskVisibleValueSource(item)
		if ok {
			return fieldID, optionValue
		}
	}
	return 0, ""
}

func conditionDataFieldID(value any) uint64 {
	fieldID := uint64(0)
	for _, item := range collectPathItems(value) {
		if strings.HasPrefix(item, collectFieldSourceDataPrefix) {
			fieldID = util.ToUint64(strings.TrimPrefix(item, collectFieldSourceDataPrefix))
		}
	}
	if fieldID > 0 {
		return fieldID
	}
	return util.ToUint64(value)
}

func conditionFieldHasOptions(fieldType string) bool {
	switch strings.TrimSpace(fieldType) {
	case "radio", "select":
		return true
	default:
		return false
	}
}

func requireConditionDataField(ctx context.Context, fieldID uint64, formPath string, label string) *crmmodel.DataField {
	if fieldID == 0 {
		return nil
	}
	field := conditionDataField(ctx, fieldID)
	if field == nil {
		panicCrmField(formPath, fmt.Sprintf("%s不存在或已停用。", label))
	}
	return field
}

func conditionDataField(ctx context.Context, fieldID uint64) *crmmodel.DataField {
	if fieldID == 0 {
		return nil
	}
	return crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":     fieldID,
		"status": crmmodel.StatusEnabled,
	})
}

func requireConditionDataFieldOption(ctx context.Context, field *crmmodel.DataField, value string, formPath string, label string) {
	if field == nil || !conditionFieldHasOptions(field.FieldType) || strings.TrimSpace(value) == "" {
		return
	}
	if !dataFieldOptionExists(ctx, field, value) {
		panicCrmField(formPath, fmt.Sprintf("%s不属于该字段的可选项。", label))
	}
}

func conditionFieldPath(ctx context.Context, fieldID uint64, value string, includeValue bool) []any {
	if fieldID == 0 {
		return []any{}
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
	if field == nil {
		return []any{}
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": field.DataTemplateID})
	if template == nil {
		return []any{}
	}
	path := []any{
		fmt.Sprintf("cate:%d", template.CateID),
		collectDataTemplateSource(template.ID),
		fmt.Sprintf("%s%d", collectFieldSourceDataPrefix, field.ID),
	}
	if includeValue {
		path = append(path, taskVisibleValueSource(field.ID, value))
	}
	return path
}

func normalizeDecisionResultField(ctx context.Context, record map[string]any, partial bool) uint64 {
	fieldID := conditionDataFieldID(record["decision_result_field_path"])
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
	field := requireConditionDataField(ctx, fieldID, "form.decision_result_field_path", "结果写入字段")
	if strings.TrimSpace(field.FieldKey) == "" {
		panicCrmField("form.decision_result_field_path", "结果写入字段必须配置字段编码。")
	}
	if !conditionFieldHasOptions(field.FieldType) {
		panicCrmField("form.decision_result_field_path", "结果写入字段必须是单选或下拉字段。")
	}
	if dataFieldOptionCount(ctx, field) == 0 {
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
	record["decision_result_field_path"] = conditionFieldPath(ctx, fieldID, "", false)
}

func applyTaskVisibleWhenForm(ctx context.Context, record map[string]any, config map[string]any) {
	visibleWhen := taskConfigObject(config["visible_when"])
	fieldID := util.ToUint64(visibleWhen["data_field_id"])
	if fieldID == 0 {
		record["visible_condition_path"] = []any{}
		return
	}
	value := util.ToStringTrimmed(visibleWhen["value"])
	record["visible_condition_path"] = conditionFieldPath(ctx, fieldID, value, true)
}
