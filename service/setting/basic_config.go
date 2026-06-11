package setting

import (
	"context"
	"strings"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

func (CrmHook) ProviderBeforeSaveBasicConfig(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}

	record["id"] = crmmodel.DefaultBasicConfigID
	prefix := strings.TrimSpace(util.ToStringTrimmed(record["customer_code_prefix"]))
	record["customer_code_prefix"] = prefix
	return record
}

func currentBasicConfig(ctx context.Context) crmmodel.BasicConfig {
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

func customerCodePrefix(ctx context.Context) string {
	return strings.TrimSpace(currentBasicConfig(ctx).CustomerCodePrefix)
}
