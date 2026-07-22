package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

const (
	scheduleFeishuMaxAttempts  = 3
	scheduleFeishuClaimTimeout = 5 * time.Minute
)

func DispatchScheduleReminders(ctx context.Context) (map[string]any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now()
	participants := crmmodel.NewScheduleParticipantModel().Select(ctx, map[string]any{})
	due := make([]*crmmodel.ScheduleParticipant, 0)
	for _, participant := range participants {
		if participant == nil || participant.FeishuSentAt != nil || participant.FeishuAttempts >= scheduleFeishuMaxAttempts {
			continue
		}
		if participant.FeishuClaimedAt != nil && participant.FeishuClaimedAt.After(now.Add(-scheduleFeishuClaimTimeout)) {
			continue
		}
		event := crmmodel.NewScheduleEventModel().Find(ctx, map[string]any{
			"id":     participant.ScheduleEventID,
			"status": crmmodel.ScheduleStatusPending,
		})
		if event == nil || event.RemindAt.After(now) {
			continue
		}
		due = append(due, participant)
	}
	if len(due) == 0 {
		return map[string]any{"due": 0, "sent": 0, "failed": 0, "skipped": 0}, nil
	}
	config := currentWorkFeishuConfig(ctx)
	if strings.TrimSpace(config.FeishuAppID) == "" || strings.TrimSpace(config.FeishuAppSecret) == "" {
		return map[string]any{"due": len(due), "sent": 0, "failed": 0, "skipped": len(due), "message": "飞书应用未配置"}, nil
	}
	token := ""
	sent := 0
	failed := 0
	skipped := 0
	for _, participant := range due {
		event := crmmodel.NewScheduleEventModel().Find(ctx, map[string]any{
			"id":     participant.ScheduleEventID,
			"status": crmmodel.ScheduleStatusPending,
		})
		if event == nil {
			skipped++
			continue
		}
		staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
			"id":     participant.StaffID,
			"status": crmmodel.StatusEnabled,
		})
		if staff == nil || strings.TrimSpace(staff.FeishuOpenID) == "" {
			skipped++
			if participant.FeishuLastError != "人员未配置飞书 OpenID" {
				crmmodel.NewScheduleParticipantModel().Update(ctx, map[string]any{"id": participant.ID}, map[string]any{
					"feishu_claimed_at": nil,
					"feishu_last_error": "人员未配置飞书 OpenID",
					"updated_at":        time.Now(),
				})
			}
			continue
		}
		if token == "" {
			var err error
			token, err = fetchWorkFeishuAppAccessToken(ctx)
			if err != nil {
				return nil, err
			}
		}
		attempt := participant.FeishuAttempts + 1
		if crmmodel.NewScheduleParticipantModel().Update(ctx, scheduleFeishuClaimFilter(participant), map[string]any{
			"feishu_claimed_at": now,
			"feishu_attempts":   attempt,
			"updated_at":        now,
		}) == 0 {
			skipped++
			continue
		}
		deliveryKey := fmt.Sprintf("crm-schedule-%d-%d-%d", event.ID, event.RemindAt.Unix(), participant.StaffID)
		if sendErr := sendScheduleFeishuMessage(ctx, token, staff.FeishuOpenID, deliveryKey, event); sendErr != nil {
			failed++
			crmmodel.NewScheduleParticipantModel().Update(ctx, map[string]any{"id": participant.ID}, map[string]any{
				"feishu_claimed_at": nil,
				"feishu_last_error": sendErr.Error(),
				"updated_at":        time.Now(),
			})
			continue
		}
		sentAt := time.Now()
		crmmodel.NewScheduleParticipantModel().Update(ctx, map[string]any{"id": participant.ID}, map[string]any{
			"feishu_sent_at":    sentAt,
			"feishu_claimed_at": nil,
			"feishu_last_error": "",
			"updated_at":        sentAt,
		})
		sent++
	}
	return map[string]any{
		"due":     len(due),
		"sent":    sent,
		"failed":  failed,
		"skipped": skipped,
	}, nil
}

func scheduleFeishuClaimFilter(participant *crmmodel.ScheduleParticipant) map[string]any {
	filter := map[string]any{
		"id":              participant.ID,
		"feishu_sent_at":  nil,
		"feishu_attempts": participant.FeishuAttempts,
	}
	// ORM 需要无类型 nil 才会生成 IS NULL；带类型的 nil 指针会生成永不匹配的 = NULL。
	if participant.FeishuClaimedAt == nil {
		filter["feishu_claimed_at"] = nil
	} else {
		filter["feishu_claimed_at"] = *participant.FeishuClaimedAt
	}
	return filter
}

func sendScheduleFeishuMessage(ctx context.Context, token string, openID string, deliveryKey string, event *crmmodel.ScheduleEvent) error {
	if event == nil {
		return fmt.Errorf("日程不存在")
	}
	if err := sendFeishuTextMessage(ctx, token, openID, deliveryKey, scheduleFeishuMessageText(ctx, event)); err != nil {
		return fmt.Errorf("飞书日程提醒发送失败：%w", err)
	}
	return nil
}

func scheduleFeishuMessageText(ctx context.Context, event *crmmodel.ScheduleEvent) string {
	lines := []string{
		"日程提醒：" + event.Title,
		"时间：" + event.StartAt.In(scheduleLocation()).Format("2006-01-02 15:04"),
	}
	if event.CustomerID > 0 {
		if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": event.CustomerID}); customer != nil {
			lines = append(lines, "客户："+customer.Name)
		}
	}
	if strings.TrimSpace(event.Remark) != "" {
		lines = append(lines, "备注："+strings.TrimSpace(event.Remark))
	}
	lines = append(lines, "请打开 CRM 工作台日程查看。")
	return strings.Join(lines, "\n")
}
