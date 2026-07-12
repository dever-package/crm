package service

import (
	"context"

	crmmodel "github.com/dever-package/crm/model"
)

func workDataUsageFieldByType(ctx context.Context, dataFieldID uint64, usageType string) *crmmodel.DataUsageField {
	if dataFieldID == 0 || usageType == "" {
		return nil
	}
	for _, usageField := range crmmodel.NewDataUsageFieldModel().Select(ctx, map[string]any{
		"data_field_id": dataFieldID,
		"status":        crmmodel.StatusEnabled,
	}) {
		if usageField == nil || usageField.UsageID == 0 {
			continue
		}
		usage := crmmodel.NewDataUsageModel().Find(ctx, map[string]any{
			"id":         usageField.UsageID,
			"usage_type": usageType,
			"status":     crmmodel.StatusEnabled,
		})
		if usage != nil {
			return usageField
		}
	}
	return nil
}

func workDataUsageFieldsByType(ctx context.Context, usageType string) []*crmmodel.DataUsageField {
	if usageType == "" {
		return nil
	}
	result := make([]*crmmodel.DataUsageField, 0)
	usages := crmmodel.NewDataUsageModel().Select(ctx, map[string]any{
		"usage_type": usageType,
		"status":     crmmodel.StatusEnabled,
	})
	for _, usage := range usages {
		if usage == nil || usage.ID == 0 {
			continue
		}
		result = append(result, crmmodel.NewDataUsageFieldModel().Select(ctx, map[string]any{
			"usage_id": usage.ID,
			"status":   crmmodel.StatusEnabled,
		})...)
	}
	return result
}

func workDataUsageByID(ctx context.Context, usageID uint64) *crmmodel.DataUsage {
	if usageID == 0 {
		return nil
	}
	return crmmodel.NewDataUsageModel().Find(ctx, map[string]any{
		"id":     usageID,
		"status": crmmodel.StatusEnabled,
	})
}
