package setting

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

const maxCustomerCodeAttempts = 30

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
	trimCrmStringField(record, "source", partial)
	trimCrmStringField(record, "level", partial)
	trimCrmStringField(record, "tags", partial)
	trimCrmStringField(record, "remark", partial)

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
		code = generateUniqueCustomerCode(ctx)
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
	sourceNames := customerSourceNames(ctx)
	levelNames := customerLevelNames(ctx)
	prefix := customerCodePrefix(ctx)

	for _, row := range rows {
		code := strings.TrimSpace(util.ToString(row["code"]))
		if code != "" {
			row["code_display"] = prefix + code
		} else {
			row["code_display"] = ""
		}
		row["source_name"] = sourceNames[strings.TrimSpace(util.ToString(row["source"]))]
		if row["source_name"] == "" {
			row["source_name"] = row["source"]
		}
		row["level_name"] = levelNames[strings.TrimSpace(util.ToString(row["level"]))]
		if row["level_name"] == "" {
			row["level_name"] = row["level"]
		}
	}
	return rows
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

func customerSourceNames(ctx context.Context) map[string]string {
	rows := crmmodel.NewCustomerSourceModel().Select(ctx, map[string]any{})
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		result[row.Code] = row.Name
	}
	return result
}

func customerLevelNames(ctx context.Context) map[string]string {
	rows := crmmodel.NewCustomerLevelModel().Select(ctx, map[string]any{})
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		result[row.Code] = row.Name
	}
	return result
}

func generateUniqueCustomerCode(ctx context.Context) string {
	model := crmmodel.NewCustomerModel()
	datePrefix := time.Now().Format("20060102")
	for i := 0; i < maxCustomerCodeAttempts; i++ {
		code := datePrefix + randomSixDigits()
		if model.Find(ctx, map[string]any{"code": code}) == nil {
			return code
		}
	}
	panicCrmField("form.code", "客户编号生成失败，请重试。")
	return ""
}

func randomSixDigits() string {
	value, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		now := time.Now().UnixNano() % 1000000
		return fmt.Sprintf("%06d", now)
	}
	return fmt.Sprintf("%06d", value.Int64())
}
