package setting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSaveCustomer(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}

	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "phone", partial)
	trimCrmStringField(record, "wechat", partial)
	trimCrmStringField(record, "id_card", partial)
	trimCrmStringField(record, "tags", partial)
	trimCrmStringField(record, "remark", partial)
	if shouldNormalizeCrmField(record, "source_id", partial) && util.ToUint64(record["source_id"]) == 0 {
		record["source_id"] = crmmodel.DefaultCustomerSourceID
	}
	if shouldNormalizeCrmField(record, "channel_id", partial) && util.ToUint64(record["channel_id"]) == 0 {
		record["channel_id"] = crmmodel.DefaultCustomerChannelID
	}
	if shouldNormalizeCrmField(record, "level_id", partial) && util.ToUint64(record["level_id"]) == 0 {
		record["level_id"] = crmmodel.DefaultCustomerLevelID
	}

	if !partial {
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "客户姓名不能为空。")
		}
		if util.ToStringTrimmed(record["phone"]) == "" {
			panicCrmField("form.phone", "手机号不能为空。")
		}
	}

	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	code := util.ToStringTrimmed(record["code"])
	customerID := util.ToUint64(record["id"])
	if code == "" && customerID > 0 {
		if current := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}); current != nil {
			code = strings.TrimSpace(current.Code)
		}
	}
	if code == "" {
		generatedCode, err := crmmodel.GenerateUniqueCustomerCode(ctx)
		if err != nil {
			panicCrmField("form.code", err.Error())
		}
		code = generatedCode
	}
	record["code"] = code
	delete(record, "creator_id")
	return record
}

func (CrmHook) ProviderBuildCustomerRows(c *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}

	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	prefix := customerCodePrefix(ctx)

	for _, row := range rows {
		code := strings.TrimSpace(util.ToString(row["code"]))
		if code != "" {
			row["code_display"] = prefix + code
		} else {
			row["code_display"] = ""
		}
		row["source_name"] = relationName(row, "source.name")
		row["channel_name"] = relationName(row, "channel.name")
		row["level_name"] = relationName(row, "level.name")
	}
	return rows
}

func (CrmHook) ProviderBeforeSaveCustomerSource(_ *server.Context, params []any) any {
	return normalizeNamedOptionRecord(params, "来源")
}

func (CrmHook) ProviderBeforeSaveCustomerChannel(_ *server.Context, params []any) any {
	return normalizeNamedOptionRecord(params, "渠道")
}

func (CrmHook) ProviderBeforeSaveCustomerLevel(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "等级名称不能为空。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	if partial {
		return record
	}
	if util.ToStringTrimmed(record["code"]) == "" {
		record["code"] = uniqueCustomerLevelCode()
	}
	return record
}

func normalizeNamedOptionRecord(params []any, label string) map[string]any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "code", partial)
	trimCrmStringField(record, "name", partial)
	if !partial {
		if util.ToStringTrimmed(record["code"]) == "" {
			panicCrmField("form.code", label+"标识不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", label+"名称不能为空。")
		}
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func rowsFromProviderParams(params []any) []map[string]any {
	if len(params) == 0 {
		return nil
	}
	payload, ok := params[0].(map[string]any)
	if !ok {
		return nil
	}
	switch rows := payload["rows"].(type) {
	case []map[string]any:
		return rows
	case []any:
		result := make([]map[string]any, 0, len(rows))
		for _, item := range rows {
			if row, ok := item.(map[string]any); ok {
				result = append(result, row)
			}
		}
		return result
	default:
		return nil
	}
}

func relationName(row map[string]any, key string) string {
	return strings.TrimSpace(util.ToString(row[key]))
}

func uniqueCustomerLevelCode() string {
	return fmt.Sprintf("level_%d", time.Now().UnixNano())
}
