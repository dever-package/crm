package setting

import (
	"context"
	"strings"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
	crmservice "github.com/dever-package/crm/service"
)

func (CrmHook) ProviderBuildDouyinConfigForm(c *server.Context, _ []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	config := crmservice.CurrentDouyinConfig(ctx)
	return map[string]any{
		"id":                    crmmodel.DefaultDouyinConfigID,
		"enabled":               config.Enabled,
		"client_key":            config.ClientKey,
		"client_secret":         "",
		"account_id":            config.AccountID,
		"root_life_account_id":  config.RootLifeAccountID,
		"last_sync_started_at":  config.LastSyncStartedAt,
		"last_sync_finished_at": config.LastSyncFinishedAt,
		"last_sync_cursor_at":   config.LastSyncCursorAt,
		"last_sync_status":      config.LastSyncStatus,
		"last_sync_message":     config.LastSyncMessage,
		"last_synced_count":     config.LastSyncedCount,
	}
}

func (CrmHook) ProviderBeforeSaveDouyinConfig(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	record["id"] = crmmodel.DefaultDouyinConfigID
	for _, field := range []string{"client_key", "client_secret", "account_id", "root_life_account_id"} {
		trimCrmStringField(record, field, false)
	}
	existing := crmmodel.NewDouyinConfigModel().Find(ctx, map[string]any{"id": crmmodel.DefaultDouyinConfigID})
	if strings.TrimSpace(util.ToStringTrimmed(record["client_secret"])) == "" && existing != nil {
		record["client_secret"] = existing.ClientSecret
	}
	enabled := util.ToBool(record["enabled"])
	record["enabled"] = enabled
	if enabled {
		if util.ToStringTrimmed(record["client_key"]) == "" {
			panicCrmField("form.client_key", "Client Key 不能为空。")
		}
		if util.ToStringTrimmed(record["client_secret"]) == "" {
			panicCrmField("form.client_secret", "Client Secret 不能为空。")
		}
		if util.ToStringTrimmed(record["account_id"]) == "" {
			panicCrmField("form.account_id", "来客账户 ID 不能为空。")
		}
	}
	for _, field := range []string{
		"last_sync_started_at",
		"last_sync_finished_at",
		"last_sync_cursor_at",
		"last_sync_status",
		"last_sync_message",
		"last_synced_count",
	} {
		delete(record, field)
	}
	record["updated_at"] = time.Now()
	return record
}

func (CrmHook) ProviderAfterSaveDouyinConfig(c *server.Context, _ []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	if err := crmservice.RefreshDouyinLeadCron(ctx); err != nil {
		panic(err)
	}
	return nil
}
