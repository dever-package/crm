package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
	frontmodel "github.com/dever-package/front/model"
	frontcron "github.com/dever-package/front/service/cron"
	"github.com/dever-package/front/service/cronexpr"
)

const (
	DouyinLeadCronProvider = "crm.DouyinCronService.SyncLeads"
	douyinLeadCronName     = "抖音来客线索同步"
	douyinLeadCronTimezone = "Asia/Shanghai"
	douyinLeadCronSpec     = "0 */5 * * * *"
	douyinLeadPageSize     = 100
	douyinLeadMaxPage      = 100
	douyinLeadSafeDelay    = 10 * time.Minute
	douyinLeadOverlap      = 5 * time.Minute
)

func init() {
	frontmodel.RegisterCronProvider(DouyinLeadCronProvider, douyinLeadCronName)
	frontcron.RegisterProvider(DouyinLeadCronProvider, func(ctx context.Context, payload map[string]any) (any, error) {
		return SyncDouyinLeads(ctx, payload)
	})
	frontcron.RegisterBootstrap(EnsureDouyinLeadCron)
}

type DouyinCronService struct{}

type douyinSyncTotals struct {
	PagesFetched int
	Pulled       int
	Created      int
	Updated      int
	Unchanged    int
	Skipped      int
}

func (DouyinCronService) ProviderSyncLeads(c *server.Context, params []any) any {
	result, err := SyncDouyinLeads(crmCronContext(c, params), crmCronPayload(params))
	if err != nil {
		panic(err)
	}
	return result
}

func CurrentDouyinConfig(ctx context.Context) crmmodel.DouyinConfig {
	config := crmmodel.NewDouyinConfigModel().Find(ctx, map[string]any{"id": crmmodel.DefaultDouyinConfigID})
	if config == nil {
		return crmmodel.DouyinConfig{ID: crmmodel.DefaultDouyinConfigID}
	}
	return *config
}

func EnsureDouyinLeadCron(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	config := CurrentDouyinConfig(ctx)
	cronModel := frontmodel.NewCronModel()
	existing := cronModel.FindMap(ctx, map[string]any{"use": DouyinLeadCronProvider})
	if len(existing) > 0 {
		return updateDouyinLeadCron(ctx, config.Enabled && douyinConfigReady(config))
	}
	now := time.Now()
	enabled := config.Enabled && douyinConfigReady(config)
	var nextRunAt any
	if enabled {
		next, err := cronexpr.Next(douyinLeadCronSpec, douyinLeadCronTimezone, now)
		if err != nil {
			return err
		}
		nextRunAt = next
	}
	cronID := util.ToUint64(cronModel.Insert(ctx, map[string]any{
		"name":            douyinLeadCronName,
		"status":          douyinCronStatus(enabled),
		"spec":            douyinLeadCronSpec,
		"schedule_mode":   frontmodel.CronScheduleEveryMinutes,
		"schedule_config": `{"interval_minutes":5}`,
		"timezone":        douyinLeadCronTimezone,
		"kind":            frontmodel.CronKindProvider,
		"use":             DouyinLeadCronProvider,
		"payload_json":    "{}",
		"timeout_seconds": 120,
		"next_run_at":     nextRunAt,
		"created_at":      now,
		"updated_at":      now,
	}))
	if cronID == 0 {
		return fmt.Errorf("创建抖音来客线索同步任务失败")
	}
	return nil
}

func RefreshDouyinLeadCron(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := EnsureDouyinLeadCron(ctx); err != nil {
		return err
	}
	config := CurrentDouyinConfig(ctx)
	return updateDouyinLeadCron(ctx, config.Enabled && douyinConfigReady(config))
}

func updateDouyinLeadCron(ctx context.Context, enabled bool) error {
	cronModel := frontmodel.NewCronModel()
	cron := cronModel.FindMap(ctx, map[string]any{"use": DouyinLeadCronProvider})
	if len(cron) == 0 {
		return nil
	}
	var nextRunAt any
	if enabled {
		next, err := cronexpr.Next(douyinLeadCronSpec, douyinLeadCronTimezone, time.Now())
		if err != nil {
			return err
		}
		nextRunAt = next
	}
	cronModel.Update(ctx, map[string]any{"id": cron["id"]}, map[string]any{
		"name":            douyinLeadCronName,
		"status":          douyinCronStatus(enabled),
		"spec":            douyinLeadCronSpec,
		"schedule_mode":   frontmodel.CronScheduleEveryMinutes,
		"schedule_config": `{"interval_minutes":5}`,
		"timezone":        douyinLeadCronTimezone,
		"kind":            frontmodel.CronKindProvider,
		"payload_json":    "{}",
		"timeout_seconds": 120,
		"next_run_at":     nextRunAt,
		"updated_at":      time.Now(),
	})
	return nil
}

func SyncDouyinLeads(ctx context.Context, _ map[string]any) (map[string]any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	config := CurrentDouyinConfig(ctx)
	if !config.Enabled {
		return map[string]any{"skipped": true, "message": "抖音线索同步未启用"}, nil
	}
	if !douyinConfigReady(config) {
		return nil, fmt.Errorf("抖音线索同步配置不完整")
	}
	credentials := douyinCredentials{
		ClientKey:         strings.TrimSpace(config.ClientKey),
		ClientSecret:      strings.TrimSpace(config.ClientSecret),
		AccountID:         strings.TrimSpace(config.AccountID),
		RootLifeAccountID: strings.TrimSpace(config.RootLifeAccountID),
	}
	workflow, source, channel, fieldIDs, err := loadDouyinLeadDependencies(ctx)
	if err != nil {
		return nil, err
	}
	startTime, endTime := douyinSyncWindow(config, time.Now())
	startedAt := time.Now()
	updateDouyinSyncState(ctx, map[string]any{
		"last_sync_started_at": startedAt,
		"last_sync_status":     frontmodel.CronRunStatusRunning,
		"last_sync_message": fmt.Sprintf(
			"正在同步 %s 至 %s",
			formatDouyinQueryTime(startTime),
			formatDouyinQueryTime(endTime),
		),
		"last_synced_count": 0,
	})

	totals := douyinSyncTotals{}
	if !startTime.Before(endTime) {
		message := "同步窗口暂无新数据"
		finishDouyinSync(ctx, endTime, totals, message)
		return douyinSyncResult(startTime, endTime, totals, message), nil
	}
	for page := 1; page <= douyinLeadMaxPage; page++ {
		result, queryErr := queryDouyinCluePage(
			ctx,
			credentials,
			formatDouyinQueryTime(startTime),
			formatDouyinQueryTime(endTime),
			page,
			douyinLeadPageSize,
		)
		if queryErr != nil {
			failDouyinSync(ctx, queryErr)
			return nil, queryErr
		}
		totals.PagesFetched++
		totals.Pulled += len(result.Clues)
		for _, raw := range result.Clues {
			imported, importErr := importDouyinLead(ctx, credentials, workflow, source, channel, fieldIDs, raw)
			if importErr != nil {
				failDouyinSync(ctx, importErr)
				return nil, importErr
			}
			switch {
			case imported.Created:
				totals.Created++
			case imported.Updated:
				totals.Updated++
			case imported.Unchanged:
				totals.Unchanged++
			case imported.Skipped:
				totals.Skipped++
			}
		}
		if page == douyinLeadMaxPage && len(result.Clues) == douyinLeadPageSize && (result.PageTotal == 0 || result.PageTotal > page) {
			err := fmt.Errorf("本次抖音线索窗口超过 10000 条分页上限，请缩短同步时间窗口后重试")
			failDouyinSync(ctx, err)
			return nil, err
		}
		if len(result.Clues) < douyinLeadPageSize || result.PageTotal > 0 && page >= result.PageTotal {
			break
		}
	}
	message := fmt.Sprintf(
		"拉取 %d 条，新增 %d 条，更新 %d 条，未变化 %d 条，跳过 %d 条",
		totals.Pulled,
		totals.Created,
		totals.Updated,
		totals.Unchanged,
		totals.Skipped,
	)
	finishDouyinSync(ctx, endTime, totals, message)
	return douyinSyncResult(startTime, endTime, totals, message), nil
}

func douyinSyncWindow(config crmmodel.DouyinConfig, now time.Time) (time.Time, time.Time) {
	endTime := now.Add(-douyinLeadSafeDelay)
	startTime := endTime
	if config.LastSyncCursorAt != nil && !config.LastSyncCursorAt.IsZero() {
		startTime = config.LastSyncCursorAt.Add(-douyinLeadOverlap)
	}
	if startTime.After(endTime) {
		startTime = endTime
	}
	return startTime, endTime
}

func finishDouyinSync(ctx context.Context, cursor time.Time, totals douyinSyncTotals, message string) {
	updateDouyinSyncState(ctx, map[string]any{
		"last_sync_finished_at": time.Now(),
		"last_sync_cursor_at":   cursor,
		"last_sync_status":      frontmodel.CronRunStatusSuccess,
		"last_sync_message":     message,
		"last_synced_count":     totals.Created + totals.Updated,
	})
}

func failDouyinSync(ctx context.Context, syncErr error) {
	message := "抖音线索同步失败"
	if syncErr != nil && strings.TrimSpace(syncErr.Error()) != "" {
		message = strings.TrimSpace(syncErr.Error())
	}
	updateDouyinSyncState(ctx, map[string]any{
		"last_sync_finished_at": time.Now(),
		"last_sync_status":      frontmodel.CronRunStatusFailed,
		"last_sync_message":     message,
		"last_synced_count":     0,
	})
}

func updateDouyinSyncState(ctx context.Context, updates map[string]any) {
	updates["updated_at"] = time.Now()
	crmmodel.NewDouyinConfigModel().Update(ctx, map[string]any{"id": crmmodel.DefaultDouyinConfigID}, updates)
}

func douyinSyncResult(startTime, endTime time.Time, totals douyinSyncTotals, message string) map[string]any {
	return map[string]any{
		"success":         true,
		"start_time":      startTime,
		"end_time":        endTime,
		"pages_fetched":   totals.PagesFetched,
		"pulled_count":    totals.Pulled,
		"created_count":   totals.Created,
		"updated_count":   totals.Updated,
		"unchanged_count": totals.Unchanged,
		"skipped_count":   totals.Skipped,
		"message":         message,
	}
}

func formatDouyinQueryTime(value time.Time) string {
	return value.In(douyinLocation()).Format("2006-01-02 15:04:05")
}

func douyinConfigReady(config crmmodel.DouyinConfig) bool {
	return strings.TrimSpace(config.ClientKey) != "" &&
		strings.TrimSpace(config.ClientSecret) != "" &&
		strings.TrimSpace(config.AccountID) != ""
}

func douyinCronStatus(enabled bool) int {
	if enabled {
		return frontmodel.CronStatusEnabled
	}
	return frontmodel.CronStatusDisabled
}

func crmCronContext(c *server.Context, params []any) context.Context {
	if c != nil {
		return c.Context()
	}
	for _, item := range params {
		if ctx, ok := item.(context.Context); ok && ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

func crmCronPayload(params []any) map[string]any {
	for _, item := range params {
		if row, ok := item.(map[string]any); ok && row != nil {
			return copyMap(row)
		}
	}
	return map[string]any{}
}
