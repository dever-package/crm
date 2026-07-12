package setting

import (
	"context"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSaveDataTemplateCate(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "target_table", partial)
	defaultCrmInt(record, "business_object_type_id", 0, partial)

	id := util.ToUint64(record["id"])
	switch id {
	case crmmodel.CustomerDataTemplateCateID:
		record["name"] = "客户信息"
		record["target_table"] = crmmodel.DataTemplateTargetCustomer
		record["business_object_type_id"] = 0
	case crmmodel.CustomerAssetDataTemplateCateID:
		record["name"] = "客户资产"
		record["target_table"] = crmmodel.DataTemplateTargetCustomerAsset
		record["business_object_type_id"] = 0
	default:
		targetTable := util.ToStringTrimmed(record["target_table"])
		if targetTable == "" {
			record["target_table"] = crmmodel.DataTemplateTargetBusinessObject
			targetTable = crmmodel.DataTemplateTargetBusinessObject
		}
		if !validDataTemplateTargetTable(targetTable) {
			panicCrmField("form.target_table", "扩展主表类型无效。")
		}
		if targetTable != crmmodel.DataTemplateTargetBusinessObject {
			record["business_object_type_id"] = 0
		} else if util.ToUint64(record["business_object_type_id"]) == 0 {
			panicCrmField("form.business_object_type_id", "业务对象类型不能为空。")
		}
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func validDataTemplateTargetTable(targetTable string) bool {
	switch targetTable {
	case crmmodel.DataTemplateTargetCustomer,
		crmmodel.DataTemplateTargetCustomerAsset,
		crmmodel.DataTemplateTargetBusinessObject:
		return true
	default:
		return false
	}
}

func (CrmHook) ProviderBeforeSaveDataTemplate(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	delete(record, "description")
	normalizeEmbeddedDataTemplateFields(c, record, partial)
	if !partial {
		if util.ToUint64(record["cate_id"]) == 0 {
			panicCrmField("form.cate_id", "模板分类不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "模板名称不能为空。")
		}
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func normalizeEmbeddedDataTemplateFields(c *server.Context, record map[string]any, partial bool) {
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
	reservedKeys := map[string]bool{}
	templateKeyPrefix := crmDataFieldKeyInitials(util.ToStringTrimmed(record["name"]))
	for _, row := range rows {
		if util.ToUint64(row["data_template_id"]) == 0 {
			delete(row, "data_template_id")
		}
		if templateKeyPrefix != "" {
			row["data_template_key_prefix"] = templateKeyPrefix
		}
		normalized = append(normalized, normalizeCrmDataFieldRecord(c, row, false, reservedKeys))
	}
	record["fields"] = normalized
}

func (CrmHook) ProviderBuildDataTemplateForm(c *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	attachDataTemplateFieldOptions(c, record)
	return record
}

func attachDataTemplateFieldOptions(c *server.Context, record map[string]any) {
	rows := formFieldRows(record["fields"])
	if rows == nil {
		return
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	normalized := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if util.ToUint64(row["parent_field_id"]) > 0 {
			continue
		}
		fieldID := util.ToUint64(row["id"])
		if fieldID == 0 {
			normalizeDataFieldFormOptions(c, row)
			normalizeDataFieldFormChildren(c, row)
			if options := formFieldRows(row["options"]); options != nil {
				row["options"] = options
			} else if _, exists := row["options"]; !exists {
				row["options"] = []map[string]any{}
			}
			normalized = append(normalized, row)
			continue
		}
		field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
		row["option_source"] = dataFieldOptionSourceCustom
		if field != nil && field.OptionSetID > 0 {
			row["option_set_id"] = field.OptionSetID
			row["option_source"] = dataFieldOptionSourceOptionSet
			row["options"] = []map[string]any{}
		} else {
			row["options"] = dataFieldPrivateOptionRows(ctx, field)
		}
		if field != nil && field.FieldType == "group" {
			row["children"] = dataFieldChildFormRows(ctx, field)
		}
		normalized = append(normalized, row)
	}
	record["fields"] = normalized
}

func normalizeDataFieldFormChildren(c *server.Context, record map[string]any) {
	if util.ToStringTrimmed(record["field_type"]) != "group" {
		delete(record, "children")
		return
	}
	rows := formFieldRows(record["children"])
	fieldID := util.ToUint64(record["id"])
	if rows == nil || len(rows) == 0 {
		if fieldID == 0 {
			record["children"] = []map[string]any{}
			return
		}
		ctx := context.Background()
		if c != nil {
			ctx = c.Context()
		}
		field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
		record["children"] = dataFieldChildFormRows(ctx, field)
		return
	}
	for _, row := range rows {
		normalizeDataFieldFormOptions(c, row)
	}
	record["children"] = rows
}

func dataFieldChildFormRows(ctx context.Context, group *crmmodel.DataField) []map[string]any {
	if group == nil || group.ID == 0 {
		return []map[string]any{}
	}
	children := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"data_template_id": group.DataTemplateID,
		"parent_field_id":  group.ID,
		"status":           crmmodel.StatusEnabled,
	})
	rows := make([]map[string]any, 0, len(children))
	for _, child := range children {
		if child == nil || child.FieldType == "group" {
			continue
		}
		row := map[string]any{
			"id":               child.ID,
			"data_template_id": child.DataTemplateID,
			"parent_field_id":  child.ParentFieldID,
			"option_set_id":    child.OptionSetID,
			"name":             child.Name,
			"field_key":        child.FieldKey,
			"field_type":       child.FieldType,
			"default_value":    child.DefaultValue,
			"sort":             child.Sort,
			"status":           child.Status,
		}
		normalizeDataFieldFormOptions(nil, row)
		rows = append(rows, row)
	}
	return rows
}

func shouldNormalizeCrmField(record map[string]any, field string, partial bool) bool {
	if !partial {
		return true
	}
	_, exists := record[field]
	return exists
}

func shouldDefaultCrmField(record map[string]any, field string, partial bool) bool {
	if !shouldNormalizeCrmField(record, field, partial) {
		return false
	}
	if _, exists := record[field]; exists {
		return true
	}
	return util.ToUint64(record["id"]) == 0
}

func defaultCrmInt16(record map[string]any, field string, fallback int16, partial bool) {
	if !shouldDefaultCrmField(record, field, partial) {
		return
	}
	if util.ToIntDefault(record[field], 0) == 0 {
		record[field] = fallback
	}
}

func defaultCrmInt(record map[string]any, field string, fallback int, partial bool) {
	if !shouldDefaultCrmField(record, field, partial) {
		return
	}
	if util.ToIntDefault(record[field], 0) == 0 {
		record[field] = fallback
	}
}

func defaultCrmBool(record map[string]any, field string, fallback bool, partial bool) {
	if !shouldDefaultCrmField(record, field, partial) {
		return
	}
	if _, exists := record[field]; !exists {
		record[field] = fallback
	}
}
