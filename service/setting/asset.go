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

const maxAssetNoAttempts = 200

func (CrmHook) ProviderBeforeSaveCustomerAsset(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}

	partial := isPartialCrmRecord(record)
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}

	trimCrmStringField(record, "asset_name", partial)
	trimCrmStringField(record, "remark", partial)
	if shouldNormalizeCrmField(record, "asset_status_id", partial) && util.ToUint64(record["asset_status_id"]) == 0 {
		record["asset_status_id"] = crmmodel.DefaultAssetStatusID
	}

	assetID := util.ToUint64(record["id"])
	if assetID > 0 {
		preserveCustomerAssetIdentity(ctx, record, assetID)
		if !partial {
			validateCustomerAssetRecord(record)
		}
		return record
	}

	if partial {
		return record
	}

	validateCustomerAssetRecord(record)
	customerID := util.ToUint64(record["customer_id"])
	customerCode := ensureCustomerCode(ctx, customerID)
	assetSeq, assetNo := nextCustomerAssetNo(ctx, customerID, customerCode)
	record["asset_seq"] = assetSeq
	record["asset_no"] = assetNo
	return record
}

func validateCustomerAssetRecord(record map[string]any) {
	customerID := util.ToUint64(record["customer_id"])
	if customerID == 0 {
		panicCrmField("form.customer_id", "所属客户不能为空。")
	}
	if util.ToStringTrimmed(record["asset_name"]) == "" {
		panicCrmField("form.asset_name", "资产名称不能为空。")
	}
	if util.ToUint64(record["asset_status_id"]) == 0 {
		panicCrmField("form.asset_status_id", "资产状态不能为空。")
	}
}

func (CrmHook) ProviderBuildCustomerAssetRows(_ *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}

	for _, row := range rows {
		row["asset_status_name"] = relationName(row, "asset_status.name")
	}
	return rows
}

func (CrmHook) ProviderBeforeSaveAssetStatus(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	id := util.ToUint64(record["id"])
	if id > 0 {
		if current := crmmodel.NewAssetStatusModel().Find(ctx, map[string]any{"id": id}); current != nil {
			record["code"] = current.Code
		}
	} else if !partial {
		record["code"] = uniqueAssetStageCode(ctx)
	}
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "状态名称不能为空。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func uniqueAssetStageCode(ctx context.Context) string {
	model := crmmodel.NewAssetStatusModel()
	for i := 1; i <= 1000; i++ {
		code := fmt.Sprintf("status_%d", time.Now().UnixNano()+int64(i))
		if model.Find(ctx, map[string]any{"code": code}) == nil {
			return code
		}
	}
	panicCrmField("form.code", "资产状态标识生成失败，请重试。")
	return ""
}

func preserveCustomerAssetIdentity(ctx context.Context, record map[string]any, assetID uint64) {
	current := crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{"id": assetID})
	if current == nil {
		return
	}
	record["customer_id"] = current.CustomerID
	record["asset_seq"] = current.AssetSeq
	record["asset_no"] = current.AssetNo
}

func ensureCustomerCode(ctx context.Context, customerID uint64) string {
	customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
	if customer == nil {
		panicCrmField("form.customer_id", "所属客户不存在。")
	}
	code := strings.TrimSpace(customer.Code)
	if code != "" {
		return code
	}

	generatedCode, err := crmmodel.GenerateUniqueCustomerCode(ctx)
	if err != nil {
		panicCrmField("form.code", err.Error())
	}
	code = generatedCode
	crmmodel.NewCustomerModel().Update(ctx, map[string]any{"id": customerID}, map[string]any{
		"code":       code,
		"updated_at": time.Now(),
	})
	return code
}

func nextCustomerAssetNo(ctx context.Context, customerID uint64, customerCode string) (uint64, string) {
	model := crmmodel.NewCustomerAssetModel()
	assets := model.Select(ctx, map[string]any{"customer_id": customerID})
	seq := uint64(len(assets) + 1)
	prefix := customerCodePrefix(ctx)

	for i := 0; i < maxAssetNoAttempts; i++ {
		assetNo := fmt.Sprintf("%s%s-%d", prefix, customerCode, seq)
		if model.Find(ctx, map[string]any{"asset_no": assetNo}) == nil {
			return seq, assetNo
		}
		seq++
	}
	panicCrmField("form.asset_no", "资产编号生成失败，请重试。")
	return 0, ""
}
