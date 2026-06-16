package setting

import (
	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

func (CrmHook) ProviderBeforeSaveFinanceType(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "code", partial)
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "direction", partial)
	if !partial {
		if util.ToStringTrimmed(record["code"]) == "" {
			panicCrmField("form.code", "类型编码不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "类型名称不能为空。")
		}
	}
	if shouldNormalizeCrmField(record, "code", partial) && !validDataFieldStatKey(util.ToStringTrimmed(record["code"])) {
		panicCrmField("form.code", "类型编码只能包含字母、数字、下划线、点和短横线。")
	}
	if shouldNormalizeCrmField(record, "direction", partial) && util.ToStringTrimmed(record["direction"]) != crmmodel.FinanceDirectionExpense {
		record["direction"] = crmmodel.FinanceDirectionIncome
	}
	defaultFinanceTypeListFields(record, partial)
	return record
}

func defaultFinanceTypeListFields(record map[string]any, partial bool) {
	if partial || util.ToUint64(record["id"]) == 0 {
		defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
		defaultCrmInt(record, "sort", 100, partial)
	}
}
