package setting

import (
	"context"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSaveOptionSet(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "name", partial)
	normalizeOptionSetItems(c, record, partial)
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "选项集名称不能为空。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBeforeSaveOptionSetItem(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "value", partial)
	if !partial {
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "选项名不能为空。")
		}
		if util.ToStringTrimmed(record["value"]) == "" {
			panicCrmField("form.value", "选项值不能为空。")
		}
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBuildOptionSetForm(_ *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	rows := formFieldRows(record["items"])
	if rows == nil {
		return record
	}
	for _, row := range rows {
		defaultCrmInt16(row, "status", crmmodel.StatusEnabled, false)
		defaultCrmInt(row, "sort", 100, false)
	}
	record["items"] = rows
	return record
}

func normalizeOptionSetItems(c *server.Context, record map[string]any, partial bool) {
	_, hasItems := record["items"]
	if partial && !hasItems {
		return
	}
	if !hasItems {
		return
	}
	items := normalizeCrmDataFieldOptionRecords(record["items"])
	optionSetID := util.ToUint64(record["id"])
	if optionSetID == 0 {
		record["items"] = items
		return
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	model := crmmodel.NewOptionSetItemModel()
	model.Delete(ctx, map[string]any{"option_set_id": optionSetID})
	for _, item := range items {
		row := util.CloneMap(item)
		row["option_set_id"] = optionSetID
		row["status"] = crmmodel.StatusEnabled
		model.Insert(ctx, row)
	}
	delete(record, "items")
}
