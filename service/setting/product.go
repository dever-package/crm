package setting

import (
	"context"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSaveProduct(c *server.Context, params []any) any {
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
			panicCrmField("form.code", "产品编码不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "产品名称不能为空。")
		}
	}
	if shouldNormalizeCrmField(record, "code", partial) && !validDataFieldKey(util.ToStringTrimmed(record["code"])) {
		panicCrmField("form.code", "产品编码只能包含字母、数字、下划线、点和短横线。")
	}
	ctx := contextFromServer(c)
	effective := effectiveProductConfig(ctx, record, partial)
	categoryID := util.ToUint64(effective["category_id"])
	if categoryID == 0 || crmmodel.NewProductCategoryModel().Find(ctx, map[string]any{
		"id":     categoryID,
		"status": crmmodel.StatusEnabled,
	}) == nil {
		panicCrmField("form.category_id", "请选择已启用的产品分类。")
	}
	if shouldNormalizeCrmField(record, "category_id", partial) {
		record["category_id"] = categoryID
	}
	serviceWorkflowID := util.ToUint64(effective["service_workflow_id"])
	if serviceWorkflowID > 0 {
		workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
			"id":     serviceWorkflowID,
			"status": crmmodel.StatusEnabled,
		})
		if workflow == nil || workflow.DefaultEntry {
			panicCrmField("form.service_workflow_id", "服务流程必须选择已启用的非入口流程。")
		}
	}
	if shouldNormalizeCrmField(record, "service_workflow_id", partial) {
		record["service_workflow_id"] = serviceWorkflowID
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func effectiveProductConfig(ctx context.Context, record map[string]any, partial bool) map[string]any {
	effective := map[string]any{
		"category_id":         uint64(0),
		"service_workflow_id": uint64(0),
	}
	if partial {
		if product := crmmodel.NewProductModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); product != nil {
			effective["category_id"] = product.CategoryID
			effective["service_workflow_id"] = product.ServiceWorkflowID
		}
	}
	for key, value := range record {
		effective[key] = value
	}
	return effective
}

func (CrmHook) ProviderBeforeSaveProductCategory(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "name", partial)
	if shouldNormalizeCrmField(record, "name", partial) && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "产品分类名称不能为空。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBeforeDeleteProductCategory(c *server.Context, params []any) any {
	id := configDeleteID(params)
	if id == 0 {
		panicCrmField("form.id", "产品分类不存在。")
	}
	if crmmodel.NewProductModel().Count(contextFromServer(c), map[string]any{"category_id": id}) > 0 {
		panicCrmField("form.id", "产品分类正在使用中，不能删除；可以先停用。")
	}
	return id
}

func (CrmHook) ProviderAfterSaveProduct(c *server.Context, _ []any) any {
	syncProductOptionSet(contextFromServer(c))
	return nil
}

func syncProductOptionSet(ctx context.Context) {
	optionSetModel := crmmodel.NewOptionSetModel()
	optionSet := optionSetModel.Find(ctx, map[string]any{"name": crmmodel.ProductOptionSetName})
	optionSetID := uint64(0)
	if optionSet != nil {
		optionSetID = optionSet.ID
	} else {
		optionSetID = uint64(optionSetModel.Insert(ctx, map[string]any{
			"name":   crmmodel.ProductOptionSetName,
			"status": crmmodel.StatusEnabled,
			"sort":   130,
		}))
	}
	if optionSetID == 0 {
		return
	}
	itemModel := crmmodel.NewOptionSetItemModel()
	itemModel.Delete(ctx, map[string]any{"option_set_id": optionSetID})
	for _, product := range crmmodel.NewProductModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled}) {
		if product == nil {
			continue
		}
		itemModel.Insert(ctx, map[string]any{
			"option_set_id": optionSetID,
			"name":          product.Name,
			"value":         product.Code,
			"sort":          product.Sort,
			"status":        crmmodel.StatusEnabled,
		})
	}
}
