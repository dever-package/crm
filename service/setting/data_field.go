package setting

import (
	"context"
	"strings"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

var dataFieldOptionTypes = map[string]struct{}{
	"radio":        {},
	"checkbox":     {},
	"select":       {},
	"multi_select": {},
}

func (CrmHook) ProviderBeforeSaveDataField(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	existing := existingCrmDataField(c, util.ToUint64(record["id"]))
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "field_key", partial)
	trimCrmStringField(record, "field_type", partial)
	trimCrmStringField(record, "default_value", partial)
	trimCrmStringField(record, "stat_type", partial)
	trimCrmStringField(record, "stat_group", partial)
	if !partial {
		if _, hasTemplateID := record["data_template_id"]; hasTemplateID && util.ToUint64(record["data_template_id"]) == 0 {
			panicCrmField("form.data_template_id", "数据模板不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "字段名称不能为空。")
		}
	}
	statEnabled := effectiveCrmDataFieldStatEnabled(record, existing)
	if statEnabled {
		fieldKey := effectiveCrmDataFieldKey(record, existing)
		if fieldKey == "" {
			panicCrmField("form.field_key", "条件字段必须填写字段编码。")
		}
		if !validDataFieldStatKey(fieldKey) {
			panicCrmField("form.field_key", "字段编码只能包含字母、数字、下划线、点和短横线。")
		}
		if duplicatedCrmDataFieldKey(c, fieldKey, util.ToUint64(record["id"])) {
			panicCrmField("form.field_key", "该字段编码已被其他条件字段使用。")
		}
	}
	if shouldNormalizeCrmField(record, "field_type", partial) && util.ToStringTrimmed(record["field_type"]) == "" {
		record["field_type"] = "text"
	}
	if shouldNormalizeCrmField(record, "stat_type", partial) && util.ToStringTrimmed(record["stat_type"]) == "" {
		record["stat_type"] = crmmodel.DataFieldStatTypeDimension
	}
	if shouldNormalizeCrmField(record, "options", partial) && !isDataFieldOptionType(util.ToStringTrimmed(record["field_type"])) {
		delete(record, "options")
	}
	defaultCrmBool(record, "stat_enabled", false, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func isDataFieldOptionType(fieldType string) bool {
	_, ok := dataFieldOptionTypes[fieldType]
	return ok
}

func existingCrmDataField(c *server.Context, id uint64) *crmmodel.DataField {
	if id == 0 {
		return nil
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	return crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": id})
}

func effectiveCrmDataFieldStatEnabled(record map[string]any, existing *crmmodel.DataField) bool {
	if _, ok := record["stat_enabled"]; ok {
		return util.ToBool(record["stat_enabled"])
	}
	return existing != nil && existing.StatEnabled
}

func effectiveCrmDataFieldKey(record map[string]any, existing *crmmodel.DataField) string {
	if _, ok := record["field_key"]; ok {
		return util.ToStringTrimmed(record["field_key"])
	}
	if existing == nil {
		return ""
	}
	return strings.TrimSpace(existing.FieldKey)
}

func duplicatedCrmDataFieldKey(c *server.Context, fieldKey string, currentID uint64) bool {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"field_key":    fieldKey,
		"stat_enabled": true,
		"status":       crmmodel.StatusEnabled,
	})
	return field != nil && field.ID != currentID
}

func validDataFieldStatKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for _, char := range key {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		if char == '_' || char == '.' || char == '-' {
			continue
		}
		return false
	}
	return true
}
