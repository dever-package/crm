package service

import (
	"context"

	crmmodel "my/package/crm/model"
)

func CurrentBasicConfig(ctx context.Context) crmmodel.BasicConfig {
	if ctx == nil {
		ctx = context.Background()
	}
	config := crmmodel.NewBasicConfigModel().Find(ctx, map[string]any{
		"id": crmmodel.DefaultBasicConfigID,
	})
	if config != nil {
		return *config
	}
	return crmmodel.DefaultBasicConfig()
}
