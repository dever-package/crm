package setting

import (
	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

func (CrmHook) ProviderBeforeSaveDataTemplateCate(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "target_table", partial)

	id := util.ToUint64(record["id"])
	switch id {
	case crmmodel.CustomerDataTemplateCateID:
		record["name"] = "客户信息"
		record["target_table"] = crmmodel.DataTemplateTargetCustomer
	case crmmodel.CustomerAssetDataTemplateCateID:
		record["name"] = "客户资产"
		record["target_table"] = crmmodel.DataTemplateTargetCustomerAsset
	default:
		panicCrmField("form.name", "数据模板分类只允许使用客户信息和客户资产。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBeforeSaveDataTemplate(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	delete(record, "description")
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

func shouldNormalizeCrmField(record map[string]any, field string, partial bool) bool {
	if !partial {
		return true
	}
	_, exists := record[field]
	return exists
}

func defaultCrmInt16(record map[string]any, field string, fallback int16, partial bool) {
	if !shouldNormalizeCrmField(record, field, partial) {
		return
	}
	if util.ToIntDefault(record[field], 0) == 0 {
		record[field] = fallback
	}
}

func defaultCrmInt(record map[string]any, field string, fallback int, partial bool) {
	if !shouldNormalizeCrmField(record, field, partial) {
		return
	}
	if util.ToIntDefault(record[field], 0) == 0 {
		record[field] = fallback
	}
}

func defaultCrmBool(record map[string]any, field string, fallback bool, partial bool) {
	if !shouldNormalizeCrmField(record, field, partial) {
		return
	}
	if _, exists := record[field]; !exists {
		record[field] = fallback
	}
}
