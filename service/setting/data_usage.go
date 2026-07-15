package setting

import (
	"context"
	"strings"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSaveDataUsage(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "usage_type", partial)
	trimCrmStringField(record, "description", partial)
	normalizeEmbeddedDataUsageFields(c, record, partial)
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "用途名称不能为空。")
	}
	if shouldNormalizeCrmField(record, "usage_type", partial) {
		record["usage_type"] = normalizeDataUsageType(record["usage_type"])
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBeforeSaveDataUsageField(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	normalizeDataUsageFieldRecord(c, record, partial, true)
	return record
}

func (CrmHook) ProviderBuildDataUsageForm(c *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	normalizeDataUsageFieldRows(c, record)
	return record
}

func (CrmHook) ProviderBuildDataUsageFieldForm(c *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	normalizeDataUsageFieldFormRow(c, record)
	return record
}

func normalizeEmbeddedDataUsageFields(c *server.Context, record map[string]any, partial bool) {
	if partial {
		if _, exists := record["fields"]; !exists {
			return
		}
	}
	rows := formFieldRows(record["fields"])
	if rows == nil {
		return
	}
	normalized := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if util.ToUint64(row["data_field_id"]) == 0 && len(collectPathItems(row["field_path"])) == 0 {
			continue
		}
		if util.ToUint64(row["usage_id"]) == 0 {
			delete(row, "usage_id")
		}
		row["usage_type"] = normalizeDataUsageType(record["usage_type"])
		normalizeDataUsageFieldRecord(c, row, false, false)
		normalized = append(normalized, row)
	}
	record["fields"] = normalized
}

func normalizeDataUsageFieldRows(c *server.Context, record map[string]any) {
	rows := formFieldRows(record["fields"])
	if rows == nil {
		return
	}
	normalized := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if util.ToUint64(row["data_field_id"]) == 0 && dataUsageFieldIDFromPath(row["field_path"]) == 0 {
			continue
		}
		normalizeDataUsageFieldFormRow(c, row)
		normalized = append(normalized, row)
	}
	record["fields"] = normalized
}

func normalizeDataUsageFieldFormRow(c *server.Context, row map[string]any) {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	fieldID := util.ToUint64(row["data_field_id"])
	if fieldID == 0 {
		fieldID = dataUsageFieldIDFromPath(row["field_path"])
	}
	if fieldID == 0 {
		return
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
	if field == nil {
		return
	}
	row["field_path"] = dataUsageFieldPath(ctx, fieldID)
	row["data_template_id"] = field.DataTemplateID
	row["data_field_id"] = field.ID
	row["data_field"] = map[string]any{
		"id":         field.ID,
		"name":       field.Name,
		"field_key":  field.FieldKey,
		"field_type": field.FieldType,
	}
	if util.ToStringTrimmed(row["display_name"]) == "" {
		row["display_name"] = field.Name
	}
	if financeTypeID := util.ToUint64(row["finance_type_id"]); financeTypeID > 0 {
		if financeType := crmmodel.NewFinanceTypeModel().Find(ctx, map[string]any{
			"id":     financeTypeID,
			"status": crmmodel.StatusEnabled,
		}); financeType != nil {
			row["finance_type"] = financeType.Name
		}
	}
}

func normalizeDataUsageFieldRecord(c *server.Context, record map[string]any, partial bool, requireUsage bool) {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	if requireUsage && !partial && util.ToUint64(record["usage_id"]) == 0 {
		panicCrmField("form.usage_id", "系统用途不能为空。")
	}
	if shouldNormalizeCrmField(record, "usage_id", partial) {
		if usageID := util.ToUint64(record["usage_id"]); usageID > 0 {
			record["usage_id"] = usageID
		} else {
			delete(record, "usage_id")
		}
	}
	if shouldNormalizeCrmField(record, "finance_type_id", partial) {
		record["finance_type_id"] = util.ToUint64(record["finance_type_id"])
	}
	trimCrmStringField(record, "value_type", partial)
	trimCrmStringField(record, "aggregate_type", partial)
	trimCrmStringField(record, "display_name", partial)
	trimCrmStringField(record, "config_json", partial)
	if shouldNormalizeCrmField(record, "field_path", partial) {
		record["data_field_id"] = dataUsageFieldIDFromPath(record["field_path"])
	}
	fieldID := util.ToUint64(record["data_field_id"])
	if fieldID == 0 {
		if !partial {
			panicCrmField("form.field_path", "用途字段不能为空。")
		}
		return
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID, "status": crmmodel.StatusEnabled})
	if field == nil || field.FieldType == "group" {
		panicCrmField("form.field_path", "用途字段必须选择启用的普通字段。")
	}
	record["data_template_id"] = field.DataTemplateID
	record["data_field_id"] = field.ID
	if shouldNormalizeCrmField(record, "value_type", partial) {
		record["value_type"] = normalizeDataUsageValueType(record["value_type"], field.FieldType)
	}
	if util.ToStringTrimmed(record["display_name"]) == "" {
		record["display_name"] = field.Name
	}
	if util.ToStringTrimmed(record["config_json"]) == "" {
		record["config_json"] = "{}"
	}
	if dataUsageFieldUsageType(ctx, record) == crmmodel.DataUsageTypeFinance && util.ToUint64(record["finance_type_id"]) == 0 {
		panicCrmField("form.finance_type_id", "财务用途必须选择财务类型。")
	}
	delete(record, "data_field")
	delete(record, "finance_type")
	delete(record, "usage_type")
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
}

func dataUsageFieldUsageType(ctx context.Context, record map[string]any) string {
	if usageType := util.ToStringTrimmed(record["usage_type"]); usageType != "" {
		return normalizeDataUsageType(usageType)
	}
	usageID := util.ToUint64(record["usage_id"])
	if usageID == 0 {
		return ""
	}
	usage := crmmodel.NewDataUsageModel().Find(ctx, map[string]any{"id": usageID})
	if usage == nil {
		return ""
	}
	return usage.UsageType
}

func dataUsageFieldPath(ctx context.Context, fieldID uint64) []any {
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
		"cate:" + util.ToString(template.CateID),
		collectDataTemplateSource(template.ID),
	}
	for _, parentID := range dataUsageFieldAncestorIDs(ctx, field) {
		path = append(path, collectFieldSourceDataPrefix+util.ToString(parentID))
	}
	path = append(path, collectFieldSourceDataPrefix+util.ToString(field.ID))
	return path
}

func dataUsageFieldAncestorIDs(ctx context.Context, field *crmmodel.DataField) []uint64 {
	if field == nil || field.ParentFieldID == 0 {
		return nil
	}
	seen := map[uint64]bool{}
	ancestors := []uint64{}
	parentID := field.ParentFieldID
	for parentID > 0 && !seen[parentID] {
		seen[parentID] = true
		parent := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
			"id":               parentID,
			"data_template_id": field.DataTemplateID,
			"status":           crmmodel.StatusEnabled,
		})
		if parent == nil {
			break
		}
		ancestors = append([]uint64{parent.ID}, ancestors...)
		parentID = parent.ParentFieldID
	}
	return ancestors
}

func normalizeDataUsageType(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.DataUsageTypeFinance:
		return crmmodel.DataUsageTypeFinance
	case crmmodel.DataUsageTypeDisplay:
		return crmmodel.DataUsageTypeDisplay
	default:
		return crmmodel.DataUsageTypeStat
	}
}

func normalizeDataUsageValueType(value any, fieldType string) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.DataUsageValueTypeNumber,
		crmmodel.DataUsageValueTypeAmount,
		crmmodel.DataUsageValueTypeTime,
		crmmodel.DataUsageValueTypeStatus,
		crmmodel.DataUsageValueTypeDimension,
		crmmodel.DataUsageValueTypeText:
		return util.ToStringTrimmed(value)
	}
	switch fieldType {
	case "number":
		return crmmodel.DataUsageValueTypeNumber
	case "money":
		return crmmodel.DataUsageValueTypeAmount
	case "date", "datetime":
		return crmmodel.DataUsageValueTypeTime
	case "radio", "select", "boolean":
		return crmmodel.DataUsageValueTypeStatus
	default:
		return crmmodel.DataUsageValueTypeText
	}
}

func (OptionService) ProviderLoadDataUsageFieldOptions(c *server.Context, params []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	ensureBaseDataTemplateCates(ctx)
	parentID := optionString(c, params, "parent_id", "parentId", "rootValue")
	switch {
	case parentID == "" || parentID == "0":
		return formFieldCateOptions(ctx)
	case strings.HasPrefix(parentID, "cate:"):
		return dataUsageTemplateOptions(ctx, util.ToUint64(strings.TrimPrefix(parentID, "cate:")))
	case strings.HasPrefix(parentID, collectTemplateSourceDataPrefix):
		_, templateID := parseCollectTemplateSource(ctx, parentID)
		return dataUsageDataFieldOptions(ctx, templateID)
	case strings.HasPrefix(parentID, collectFieldSourceDataPrefix):
		fieldID := util.ToUint64(strings.TrimPrefix(parentID, collectFieldSourceDataPrefix))
		return dataUsageChildDataFieldOptions(ctx, fieldID)
	default:
		return []map[string]any{}
	}
}

func dataUsageFieldIDFromPath(value any) uint64 {
	items := collectPathItems(value)
	for index := len(items) - 1; index >= 0; index-- {
		item := strings.TrimSpace(items[index])
		if strings.HasPrefix(item, collectFieldSourceDataPrefix) {
			return util.ToUint64(strings.TrimPrefix(item, collectFieldSourceDataPrefix))
		}
	}
	return 0
}

func (OptionService) ProviderLoadFinanceTypeCascaderOptions(c *server.Context, params []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	parentID := optionString(c, params, "parent_id", "parentId", "rootValue")
	if parentID != "" && parentID != "0" {
		return []map[string]any{}
	}
	rows := crmmodel.NewFinanceTypeModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	})
	options := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		options = append(options, map[string]any{
			"id":        row.ID,
			"value":     row.Name,
			"name":      row.Name,
			"leaf":      true,
			"code":      row.Code,
			"direction": row.Direction,
			"sort":      row.Sort,
		})
	}
	return options
}

func dataUsageDataFieldOptions(ctx context.Context, templateID uint64) []map[string]any {
	if templateID == 0 {
		return []map[string]any{}
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
		"id":     templateID,
		"status": crmmodel.StatusEnabled,
	})
	if template == nil {
		return []map[string]any{}
	}
	return dataUsageDataFieldOptionsByParent(ctx, template, 0)
}

func dataUsageChildDataFieldOptions(ctx context.Context, parentFieldID uint64) []map[string]any {
	if parentFieldID == 0 {
		return []map[string]any{}
	}
	parent := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":     parentFieldID,
		"status": crmmodel.StatusEnabled,
	})
	if parent == nil || parent.FieldType != "group" {
		return []map[string]any{}
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
		"id":     parent.DataTemplateID,
		"status": crmmodel.StatusEnabled,
	})
	if template == nil {
		return []map[string]any{}
	}
	return dataUsageDataFieldOptionsByParent(ctx, template, parent.ID)
}

func dataUsageDataFieldOptionsByParent(ctx context.Context, template *crmmodel.DataTemplate, parentFieldID uint64) []map[string]any {
	if template == nil {
		return []map[string]any{}
	}
	fields := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"data_template_id": template.ID,
		"parent_field_id":  parentFieldID,
		"status":           crmmodel.StatusEnabled,
	})
	options := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		options = append(options, dataUsageDataFieldOption(template, field))
	}
	return options
}

func dataUsageDataFieldOption(template *crmmodel.DataTemplate, field *crmmodel.DataField) map[string]any {
	isGroup := field.FieldType == "group"
	return map[string]any{
		"id":                    collectFieldSourceDataPrefix + util.ToString(field.ID),
		"value":                 field.Name,
		"name":                  field.Name,
		"leaf":                  !isGroup,
		"source":                "data_field",
		"data_template_id":      template.ID,
		"data_field_id":         field.ID,
		"field_key":             field.FieldKey,
		"field_type":            field.FieldType,
		"parent_field_id":       field.ParentFieldID,
		"data_template_cate_id": template.CateID,
		"sort":                  field.Sort,
	}
}
