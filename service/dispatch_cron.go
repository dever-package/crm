package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	frontmodel "github.com/dever-package/front/model"
	frontcron "github.com/dever-package/front/service/cron"
	"github.com/dever-package/front/service/cronexpr"
)

const (
	PendingDispatchCronProvider = "crm.DispatchCronService.RetryPending"
	pendingDispatchCronName     = "CRM 待派单重试"
	pendingDispatchCronTimezone = "Asia/Shanghai"
	pendingDispatchCronSpec     = "0 * * * * *"
)

func init() {
	frontmodel.RegisterCronProvider(PendingDispatchCronProvider, pendingDispatchCronName)
	frontcron.RegisterProvider(PendingDispatchCronProvider, func(ctx context.Context, _ map[string]any) (any, error) {
		return RetryPendingDispatches(ctx)
	})
	frontcron.RegisterBootstrap(EnsurePendingDispatchCron)
}

type DispatchCronService struct{}

func (DispatchCronService) ProviderRetryPending(c *server.Context, params []any) any {
	result, err := RetryPendingDispatches(crmCronContext(c, params))
	if err != nil {
		panic(err)
	}
	return result
}

func EnsurePendingDispatchCron(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ensureDepartmentDispatchDefaults(ctx); err != nil {
		return err
	}
	model := frontmodel.NewCronModel()
	existing := model.FindMap(ctx, map[string]any{"use": PendingDispatchCronProvider})
	now := time.Now()
	nextRunAt, err := cronexpr.Next(pendingDispatchCronSpec, pendingDispatchCronTimezone, now)
	if err != nil {
		return err
	}
	data := map[string]any{
		"name":            pendingDispatchCronName,
		"status":          frontmodel.CronStatusEnabled,
		"spec":            pendingDispatchCronSpec,
		"schedule_mode":   frontmodel.CronScheduleEveryMinutes,
		"schedule_config": `{"interval_minutes":1}`,
		"timezone":        pendingDispatchCronTimezone,
		"kind":            frontmodel.CronKindProvider,
		"use":             PendingDispatchCronProvider,
		"payload_json":    "{}",
		"timeout_seconds": 60,
		"next_run_at":     nextRunAt,
		"updated_at":      now,
	}
	if len(existing) > 0 {
		model.Update(ctx, map[string]any{"id": existing["id"]}, data)
		return nil
	}
	data["created_at"] = now
	if util.ToUint64(model.Insert(ctx, data)) == 0 {
		return fmt.Errorf("创建 CRM 待派单重试任务失败")
	}
	return nil
}
