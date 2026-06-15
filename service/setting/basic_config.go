package setting

import (
	"context"
	"strings"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
	crmservice "my/package/crm/service"
)

func (CrmHook) ProviderBeforeSaveBasicConfig(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}

	record["id"] = crmmodel.DefaultBasicConfigID
	trimBasicConfigField(record, "customer_code_prefix")
	trimBasicConfigField(record, "feishu_app_id")
	trimBasicConfigField(record, "feishu_app_secret")
	return record
}

func trimBasicConfigField(record map[string]any, field string) {
	if _, exists := record[field]; !exists {
		return
	}
	record[field] = strings.TrimSpace(util.ToStringTrimmed(record[field]))
}

func currentBasicConfig(ctx context.Context) crmmodel.BasicConfig {
	return crmservice.CurrentBasicConfig(ctx)
}

func customerCodePrefix(ctx context.Context) string {
	return strings.TrimSpace(currentBasicConfig(ctx).CustomerCodePrefix)
}
