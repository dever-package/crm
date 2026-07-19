package service

import (
	"context"
	"fmt"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func syncScheduleParticipants(ctx context.Context, eventID uint64, ownerStaffID uint64, participantIDs []uint64, replace bool, resetReminder bool) error {
	if eventID == 0 || ownerStaffID == 0 {
		return fmt.Errorf("日程和负责人不能为空")
	}
	targetIDs := uniqueScheduleStaffIDs(ownerStaffID, participantIDs)
	for _, staffID := range targetIDs {
		if crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID, "status": crmmodel.StatusEnabled}) == nil {
			return fmt.Errorf("日程参与人不存在或已停用")
		}
	}
	model := crmmodel.NewScheduleParticipantModel()
	existing := model.Select(ctx, map[string]any{"schedule_event_id": eventID})
	existingByStaff := make(map[uint64]*crmmodel.ScheduleParticipant, len(existing))
	for _, row := range existing {
		if row != nil {
			existingByStaff[row.StaffID] = row
		}
	}
	now := time.Now()
	if resetReminder {
		model.Update(
			ctx,
			map[string]any{"schedule_event_id": eventID},
			scheduleParticipantReminderReset(now),
		)
	}
	if replace {
		for staffID, row := range existingByStaff {
			if !uint64SetContains(targetIDs, staffID) {
				model.Delete(ctx, map[string]any{"id": row.ID})
			}
		}
	}
	for _, staffID := range targetIDs {
		role := crmmodel.MemberRelationParticipant
		if staffID == ownerStaffID {
			role = crmmodel.MemberRelationAssignee
		}
		if row := existingByStaff[staffID]; row != nil {
			model.Update(ctx, map[string]any{"id": row.ID}, map[string]any{
				"role":       role,
				"updated_at": now,
			})
			continue
		}
		model.Insert(ctx, map[string]any{
			"schedule_event_id": eventID,
			"staff_id":          staffID,
			"role":              role,
			"feishu_attempts":   0,
			"feishu_last_error": "",
			"created_at":        now,
			"updated_at":        now,
		})
	}
	return nil
}

func resetScheduleParticipantReminders(ctx context.Context, eventID uint64) {
	crmmodel.NewScheduleParticipantModel().Update(
		ctx,
		map[string]any{"schedule_event_id": eventID},
		scheduleParticipantReminderReset(time.Now()),
	)
}

func scheduleParticipantReminderReset(updatedAt time.Time) map[string]any {
	return map[string]any{
		"workbench_read_at": nil,
		"feishu_sent_at":    nil,
		"feishu_claimed_at": nil,
		"feishu_attempts":   0,
		"feishu_last_error": "",
		"updated_at":        updatedAt,
	}
}

func scheduleParticipantIDs(ctx context.Context, eventID uint64) []uint64 {
	rows := crmmodel.NewScheduleParticipantModel().Select(ctx, map[string]any{"schedule_event_id": eventID})
	result := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.StaffID > 0 {
			result = append(result, row.StaffID)
		}
	}
	return result
}

func uniqueScheduleStaffIDs(ownerStaffID uint64, participantIDs []uint64) []uint64 {
	result := make([]uint64, 0, len(participantIDs)+1)
	seen := map[uint64]bool{}
	appendID := func(staffID uint64) {
		if staffID == 0 || seen[staffID] {
			return
		}
		seen[staffID] = true
		result = append(result, staffID)
	}
	appendID(ownerStaffID)
	for _, staffID := range participantIDs {
		appendID(staffID)
	}
	return result
}
