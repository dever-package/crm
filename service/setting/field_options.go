package setting

import (
	"context"

	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func dataFieldOptionRows(ctx context.Context, field *crmmodel.DataField) []map[string]any {
	if field == nil || field.ID == 0 {
		return []map[string]any{}
	}
	if field.OptionSetID > 0 {
		return optionSetItemRows(ctx, field.OptionSetID)
	}
	return dataFieldPrivateOptionRows(ctx, field)
}

func dataFieldPrivateOptionRows(ctx context.Context, field *crmmodel.DataField) []map[string]any {
	if field == nil || field.ID == 0 {
		return []map[string]any{}
	}
	return crmmodel.NewDataFieldOptionModel().SelectMap(ctx, map[string]any{
		"data_field_id": field.ID,
	}, map[string]any{
		"order": "main.sort asc, main.id asc",
	})
}

func dataFieldOptionCount(ctx context.Context, field *crmmodel.DataField) int64 {
	if field == nil || field.ID == 0 {
		return 0
	}
	if field.OptionSetID > 0 {
		return crmmodel.NewOptionSetItemModel().Count(ctx, map[string]any{
			"option_set_id": field.OptionSetID,
			"status":        crmmodel.StatusEnabled,
		})
	}
	return crmmodel.NewDataFieldOptionModel().Count(ctx, map[string]any{"data_field_id": field.ID})
}

func dataFieldOptionExists(ctx context.Context, field *crmmodel.DataField, value string) bool {
	if field == nil || field.ID == 0 || value == "" {
		return false
	}
	if field.OptionSetID > 0 {
		return crmmodel.NewOptionSetItemModel().Find(ctx, map[string]any{
			"option_set_id": field.OptionSetID,
			"value":         value,
			"status":        crmmodel.StatusEnabled,
		}) != nil
	}
	return crmmodel.NewDataFieldOptionModel().Find(ctx, map[string]any{
		"data_field_id": field.ID,
		"value":         value,
	}) != nil
}

func optionSetItemRows(ctx context.Context, optionSetID uint64) []map[string]any {
	rows := crmmodel.NewOptionSetItemModel().SelectMap(ctx, map[string]any{
		"option_set_id": optionSetID,
		"status":        crmmodel.StatusEnabled,
	}, map[string]any{
		"field": "main.id, main.name, main.value, main.sort",
		"order": "main.sort asc, main.id asc",
	})
	for _, row := range rows {
		row["data_field_id"] = uint64(0)
		row["option_set_id"] = optionSetID
		row["sort"] = util.ToIntDefault(row["sort"], 100)
	}
	return rows
}
