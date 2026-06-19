package setting

import (
	"context"
	"strings"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

var dataFieldOptionTypes = map[string]struct{}{
	"radio":        {},
	"checkbox":     {},
	"select":       {},
	"multi_select": {},
	"boolean":      {},
}

func (CrmHook) ProviderBeforeSaveDataField(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "stat_enabled", "status", "sort")
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
	if shouldNormalizeCrmField(record, "field_type", partial) && util.ToStringTrimmed(record["field_type"]) == "" {
		record["field_type"] = "text"
	}
	if shouldNormalizeCrmField(record, "stat_type", partial) && util.ToStringTrimmed(record["stat_type"]) == "" {
		record["stat_type"] = crmmodel.DataFieldStatTypeDimension
	}
	ensureCrmFinanceDataField(c, record, existing, partial)
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
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	normalizeCrmDataFieldOptions(ctx, record, existing, partial)
	defaultCrmBool(record, "stat_enabled", false, partial)
	defaultCrmInt(record, "stat_id", 0, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

type crmDataFieldOptionInput struct {
	name  string
	value string
	sort  int
}

func isDataFieldOptionType(fieldType string) bool {
	_, ok := dataFieldOptionTypes[fieldType]
	return ok
}

func normalizeCrmDataFieldOptions(ctx context.Context, record map[string]any, existing *crmmodel.DataField, partial bool) {
	_, hasOptions := record["options"]
	fieldTypeChanged := shouldNormalizeCrmField(record, "field_type", partial)
	fieldType := effectiveCrmDataFieldType(record, existing)
	if !hasOptions && !(fieldTypeChanged && !isDataFieldOptionType(fieldType)) {
		return
	}
	fieldID := util.ToUint64(record["id"])
	if !isDataFieldOptionType(fieldType) {
		if fieldID > 0 {
			crmmodel.NewDataFieldOptionModel().Delete(ctx, map[string]any{"data_field_id": fieldID})
		}
		delete(record, "options")
		return
	}
	if fieldID == 0 || !hasOptions {
		if hasOptions {
			record["options"] = normalizeCrmDataFieldOptionRecords(record["options"])
		}
		return
	}
	syncCrmDataFieldOptions(ctx, fieldID, record["options"])
	delete(record, "options")
}

func syncCrmDataFieldOptions(ctx context.Context, fieldID uint64, rawOptions any) {
	records := normalizeCrmDataFieldOptionRecords(rawOptions)
	model := crmmodel.NewDataFieldOptionModel()
	model.Delete(ctx, map[string]any{"data_field_id": fieldID})
	for _, record := range records {
		option := util.CloneMap(record)
		option["data_field_id"] = fieldID
		model.Insert(ctx, option)
	}
}

func normalizeCrmDataFieldOptionRecords(rawOptions any) []map[string]any {
	inputs := normalizeCrmDataFieldOptionInputs(rawOptions)
	records := make([]map[string]any, 0, len(inputs))
	for _, input := range inputs {
		records = append(records, map[string]any{
			"name":  input.name,
			"value": input.value,
			"sort":  input.sort,
		})
	}
	return records
}

func normalizeCrmDataFieldOptionInputs(rawOptions any) []crmDataFieldOptionInput {
	rows := formFieldRows(rawOptions)
	inputs := make([]crmDataFieldOptionInput, 0, len(rows))
	seenValues := map[string]bool{}
	for index, row := range rows {
		if blankCrmDataFieldOptionRow(row) {
			continue
		}
		name := util.ToStringTrimmed(row["name"])
		value := util.ToStringTrimmed(row["value"])
		if name == "" {
			panicCrmField("form.options", "选项名不能为空。")
		}
		if value == "" {
			panicCrmField("form.options", "选项值不能为空。")
		}
		if seenValues[value] {
			panicCrmField("form.options", "选项值不能重复。")
		}
		seenValues[value] = true
		inputs = append(inputs, crmDataFieldOptionInput{
			name:  name,
			value: value,
			sort:  util.ToIntDefault(row["sort"], (index+1)*10),
		})
	}
	return inputs
}

func blankCrmDataFieldOptionRow(row map[string]any) bool {
	return util.ToUint64(row["id"]) == 0 &&
		util.ToStringTrimmed(row["name"]) == "" &&
		util.ToStringTrimmed(row["value"]) == ""
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

func effectiveCrmDataFieldType(record map[string]any, existing *crmmodel.DataField) string {
	if _, ok := record["field_type"]; ok {
		return util.ToStringTrimmed(record["field_type"])
	}
	if existing == nil {
		return ""
	}
	return strings.TrimSpace(existing.FieldType)
}

func effectiveCrmDataFieldStatType(record map[string]any, existing *crmmodel.DataField) string {
	if _, ok := record["stat_type"]; ok {
		return util.ToStringTrimmed(record["stat_type"])
	}
	if existing == nil {
		return crmmodel.DataFieldStatTypeDimension
	}
	return strings.TrimSpace(existing.StatType)
}

func effectiveCrmDataFieldStatID(record map[string]any, existing *crmmodel.DataField) uint64 {
	if _, ok := record["stat_id"]; ok {
		return util.ToUint64(record["stat_id"])
	}
	if existing == nil {
		return 0
	}
	return existing.StatID
}

func ensureCrmFinanceDataField(c *server.Context, record map[string]any, existing *crmmodel.DataField, partial bool) {
	if !crmDataFieldFinanceConfigTouched(record, partial) {
		return
	}
	statEnabled := effectiveCrmDataFieldStatEnabled(record, existing)
	statType := effectiveCrmDataFieldStatType(record, existing)
	if !statEnabled || statType != crmmodel.DataFieldStatTypeFinance {
		if shouldNormalizeCrmField(record, "stat_type", partial) || effectiveCrmDataFieldStatID(record, existing) > 0 {
			record["stat_id"] = uint64(0)
		}
		return
	}
	fieldType := effectiveCrmDataFieldType(record, existing)
	if fieldType != "money" && fieldType != "number" {
		panicCrmField("form.field_type", "财务类型字段必须使用金额或数字字段。")
	}
	statID := effectiveCrmDataFieldStatID(record, existing)
	if statID == 0 {
		panicCrmField("form.stat_id", "财务类型字段必须选择财务类型。")
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	if crmmodel.NewFinanceTypeModel().Find(ctx, map[string]any{"id": statID, "status": crmmodel.StatusEnabled}) == nil {
		panicCrmField("form.stat_id", "财务类型不存在或已停用。")
	}
	record["stat_id"] = statID
	if util.ToStringTrimmed(record["stat_group"]) == "" && (existing == nil || strings.TrimSpace(existing.StatGroup) == "") {
		record["stat_group"] = "财务"
	}
}

func crmDataFieldFinanceConfigTouched(record map[string]any, partial bool) bool {
	if !partial {
		return true
	}
	for _, field := range []string{"field_type", "stat_enabled", "stat_type", "stat_id"} {
		if _, ok := record[field]; ok {
			return true
		}
	}
	return false
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
