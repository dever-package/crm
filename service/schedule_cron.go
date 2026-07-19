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
	ScheduleReminderCronProvider = "crm.ScheduleCronService.DispatchReminders"
	scheduleReminderCronName     = "CRM 日程到期提醒"
	scheduleReminderCronTimezone = "Asia/Shanghai"
	scheduleReminderCronSpec     = "0 * * * * *"
)

func init() {
	frontmodel.RegisterCronProvider(ScheduleReminderCronProvider, scheduleReminderCronName)
	frontcron.RegisterProvider(ScheduleReminderCronProvider, func(ctx context.Context, _ map[string]any) (any, error) {
		return DispatchScheduleReminders(ctx)
	})
	frontcron.RegisterBootstrap(EnsureScheduleReminderCron)
}

type ScheduleCronService struct{}

func (ScheduleCronService) ProviderDispatchReminders(c *server.Context, params []any) any {
	result, err := DispatchScheduleReminders(crmCronContext(c, params))
	if err != nil {
		panic(err)
	}
	return result
}

func EnsureScheduleReminderCron(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	model := frontmodel.NewCronModel()
	existing := model.FindMap(ctx, map[string]any{"use": ScheduleReminderCronProvider})
	now := time.Now()
	nextRunAt, err := cronexpr.Next(scheduleReminderCronSpec, scheduleReminderCronTimezone, now)
	if err != nil {
		return err
	}
	data := map[string]any{
		"name":            scheduleReminderCronName,
		"status":          frontmodel.CronStatusEnabled,
		"spec":            scheduleReminderCronSpec,
		"schedule_mode":   frontmodel.CronScheduleEveryMinutes,
		"schedule_config": `{"interval_minutes":1}`,
		"timezone":        scheduleReminderCronTimezone,
		"kind":            frontmodel.CronKindProvider,
		"use":             ScheduleReminderCronProvider,
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
		return fmt.Errorf("创建 CRM 日程提醒任务失败")
	}
	return nil
}
