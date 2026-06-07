package setting

import (
	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

var obsoleteDataFieldFields = []string{
	"field_key",
	"options_json",
	"required",
	"is_metric",
	"metric_key",
	"metric_type",
	"aggregate",
}

var dataFieldOptionTypes = map[string]struct{}{
	"radio":        {},
	"checkbox":     {},
	"select":       {},
	"multi_select": {},
}

func (CrmHook) ProviderBeforeSaveDataField(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	for _, field := range obsoleteDataFieldFields {
		delete(record, field)
	}
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "field_type", partial)
	trimCrmStringField(record, "default_value", partial)
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
	if shouldNormalizeCrmField(record, "options", partial) && !isDataFieldOptionType(util.ToStringTrimmed(record["field_type"])) {
		delete(record, "options")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func isDataFieldOptionType(fieldType string) bool {
	_, ok := dataFieldOptionTypes[fieldType]
	return ok
}
