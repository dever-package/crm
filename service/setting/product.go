package setting

import (
	"context"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSaveProduct(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "code", partial)
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "category", partial)
	trimCrmStringField(record, "default_signing_business_type", partial)
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
	if shouldNormalizeCrmField(record, "category", partial) && !productCategoryAllowed(util.ToStringTrimmed(record["category"])) {
		record["category"] = crmmodel.ProductCategoryConsulting
	}
	if shouldNormalizeCrmField(record, "default_signing_business_type", partial) && !productSigningTypeAllowed(util.ToStringTrimmed(record["default_signing_business_type"])) {
		record["default_signing_business_type"] = crmmodel.ProductSigningManual
	}
	defaultCrmBool(record, "need_pm_review", true, partial)
	defaultCrmBool(record, "need_lawyer_review", false, partial)
	defaultCrmBool(record, "need_ala_review", false, partial)
	defaultCrmBool(record, "need_finance_review", false, partial)
	defaultCrmBool(record, "need_contract_review", true, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderAfterSaveProduct(c *server.Context, _ []any) any {
	syncProductOptionSet(contextFromServer(c))
	return nil
}

func productCategoryAllowed(category string) bool {
	switch category {
	case crmmodel.ProductCategoryJudicial,
		crmmodel.ProductCategoryAssetOperation,
		crmmodel.ProductCategoryDebtStructure,
		crmmodel.ProductCategoryStageService,
		crmmodel.ProductCategoryRiskDisposal,
		crmmodel.ProductCategoryConsulting:
		return true
	default:
		return false
	}
}

func productSigningTypeAllowed(signingType string) bool {
	switch signingType {
	case crmmodel.ProductSigningNonSealed,
		crmmodel.ProductSigningSealed,
		crmmodel.ProductSigningManual:
		return true
	default:
		return false
	}
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
