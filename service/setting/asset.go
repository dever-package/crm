package setting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

const maxAssetNoAttempts = 200

var obsoleteCustomerResourceFields = []string{
	"title",
	"resource_type",
	"source",
	"channel",
	"city",
	"flow_template_id",
	"flow_release_id",
	"current_stage_id",
	"current_task_id",
	"owner_department_id",
	"owner_staff_id",
	"status",
	"risk_level",
	"summary_json",
	"closed_at",
}

func (CrmHook) ProviderBeforeSaveCustomerResource(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}

	partial := isPartialCrmRecord(record)
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}

	cleanCustomerResourceRecord(record)
	trimCrmStringField(record, "asset_name", partial)
	trimCrmStringField(record, "asset_status", partial)
	trimCrmStringField(record, "remark", partial)

	resourceID := util.ToUint64(record["id"])
	if resourceID > 0 {
		preserveCustomerResourceIdentity(ctx, record, resourceID)
		return record
	}

	if partial {
		return record
	}

	customerID := util.ToUint64(record["customer_id"])
	if customerID == 0 {
		panicCrmField("form.customer_id", "所属客户不能为空。")
	}
	if util.ToStringTrimmed(record["asset_name"]) == "" {
		panicCrmField("form.asset_name", "资产名称不能为空。")
	}
	if util.ToUint64(record["asset_cate_id"]) == 0 {
		record["asset_cate_id"] = crmmodel.DefaultAssetCateID
	}
	if util.ToStringTrimmed(record["asset_status"]) == "" {
		record["asset_status"] = crmmodel.AssetStatusDefault
	}

	customerCode := ensureCustomerCode(ctx, customerID)
	assetSeq, assetNo := nextCustomerAssetNo(ctx, customerID, customerCode)
	record["asset_seq"] = assetSeq
	record["resource_no"] = assetNo
	return record
}

func (CrmHook) ProviderBuildCustomerResourceRows(c *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}

	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	statusNames := assetStatusNames(ctx)
	for _, row := range rows {
		status := strings.TrimSpace(util.ToString(row["asset_status"]))
		row["asset_status_name"] = statusNames[status]
		if row["asset_status_name"] == "" {
			row["asset_status_name"] = status
		}
	}
	return rows
}

func cleanCustomerResourceRecord(record map[string]any) {
	for _, field := range obsoleteCustomerResourceFields {
		delete(record, field)
	}
}

func (CrmHook) ProviderBeforeSaveAssetCate(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "分类名称不能为空。")
	}
	if !partial || record["status"] != nil {
		if util.ToIntDefault(record["status"], 0) == 0 {
			record["status"] = crmmodel.StatusEnabled
		}
	}
	if !partial || record["sort"] != nil {
		if util.ToIntDefault(record["sort"], 0) == 0 {
			record["sort"] = 100
		}
	}
	return record
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
		record["code"] = uniqueAssetStatusCode(ctx)
	}
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "状态名称不能为空。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func uniqueAssetStatusCode(ctx context.Context) string {
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

func preserveCustomerResourceIdentity(ctx context.Context, record map[string]any, resourceID uint64) {
	current := crmmodel.NewCustomerResourceModel().Find(ctx, map[string]any{"id": resourceID})
	if current == nil {
		return
	}
	record["customer_id"] = current.CustomerID
	record["asset_seq"] = current.AssetSeq
	record["resource_no"] = current.ResourceNo
	if util.ToUint64(record["asset_cate_id"]) == 0 {
		record["asset_cate_id"] = crmmodel.DefaultAssetCateID
	}
	if util.ToStringTrimmed(record["asset_status"]) == "" {
		record["asset_status"] = crmmodel.AssetStatusDefault
	}
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

	code = generateUniqueCustomerCode(ctx)
	crmmodel.NewCustomerModel().Update(ctx, map[string]any{"id": customerID}, map[string]any{
		"code":       code,
		"updated_at": time.Now(),
	})
	return code
}

func nextCustomerAssetNo(ctx context.Context, customerID uint64, customerCode string) (uint64, string) {
	model := crmmodel.NewCustomerResourceModel()
	assets := model.Select(ctx, map[string]any{"customer_id": customerID})
	seq := uint64(len(assets) + 1)
	prefix := customerCodePrefix(ctx)

	for i := 0; i < maxAssetNoAttempts; i++ {
		assetNo := fmt.Sprintf("%s%s-%d", prefix, customerCode, seq)
		if model.Find(ctx, map[string]any{"resource_no": assetNo}) == nil {
			return seq, assetNo
		}
		seq++
	}
	panicCrmField("form.resource_no", "资产编号生成失败，请重试。")
	return 0, ""
}

func assetStatusNames(ctx context.Context) map[string]string {
	rows := crmmodel.NewAssetStatusModel().Select(ctx, map[string]any{})
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		result[row.Code] = row.Name
	}
	return result
}
