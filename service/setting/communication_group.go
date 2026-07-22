package setting

import (
	crmmodel "github.com/dever-package/crm/model"
	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"
)

func (CrmHook) ProviderBeforeSaveCommunicationGroupType(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "code", partial)
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "description", partial)
	if !partial {
		if util.ToStringTrimmed(record["code"]) == "" {
			panicCrmField("form.code", "类型编码不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "类型名称不能为空。")
		}
	}
	if shouldNormalizeCrmField(record, "code", partial) && !validDataFieldKey(util.ToStringTrimmed(record["code"])) {
		panicCrmField("form.code", "类型编码只能包含字母、数字、下划线、点和短横线。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}
