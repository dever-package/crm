package service

import (
	"context"
	"fmt"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func syncScheduleResources(ctx context.Context, event *crmmodel.ScheduleEvent, resourceIDs []uint64, replace bool, departmentID uint64) error {
	if event == nil || event.ID == 0 {
		return fmt.Errorf("日程不存在")
	}
	model := crmmodel.NewPublicResourceBookingModel()
	existing := model.Select(ctx, map[string]any{"schedule_event_id": event.ID})
	existingByResource := make(map[uint64]*crmmodel.PublicResourceBooking, len(existing))
	activeIDs := make([]uint64, 0, len(existing))
	for _, booking := range existing {
		if booking == nil {
			continue
		}
		existingByResource[booking.ResourceID] = booking
		if !ResourceBookingInactive(booking.BookingStatus) {
			activeIDs = append(activeIDs, booking.ResourceID)
		}
	}
	targetIDs := activeIDs
	if replace {
		targetIDs = uniqueUint64Values(resourceIDs)
	}
	for _, resourceID := range targetIDs {
		resource := crmmodel.NewPublicResourceModel().Find(ctx, map[string]any{
			"id":     resourceID,
			"status": crmmodel.StatusEnabled,
		})
		if resource == nil {
			return fmt.Errorf("公共资源不存在或已停用")
		}
		currentID := uint64(0)
		if booking := existingByResource[resourceID]; booking != nil {
			currentID = booking.ID
		}
		if err := ValidateResourceBookingTime(ctx, currentID, resourceID, event.StartAt, event.EndAt); err != nil {
			return err
		}
	}
	now := time.Now()
	if replace {
		for resourceID, booking := range existingByResource {
			if !uint64SetContains(targetIDs, resourceID) && !ResourceBookingInactive(booking.BookingStatus) {
				model.Update(ctx, map[string]any{"id": booking.ID}, map[string]any{
					"booking_status": crmmodel.ResourceBookingStatusCanceled,
					"updated_at":     now,
				})
			}
		}
	}
	for _, resourceID := range targetIDs {
		data := map[string]any{
			"schedule_event_id":    event.ID,
			"customer_id":          event.CustomerID,
			"asset_id":             event.AssetID,
			"task_id":              event.SourceTaskID,
			"operation_log_id":     event.OperationLogID,
			"stage_code":           "",
			"booking_status":       crmmodel.ResourceBookingStatusReserved,
			"title":                event.Title,
			"remark":               event.Remark,
			"start_at":             event.StartAt,
			"end_at":               event.EndAt,
			"booker_staff_id":      event.OwnerStaffID,
			"booker_department_id": departmentID,
			"approved_by_staff_id": uint64(0),
			"updated_at":           now,
		}
		if booking := existingByResource[resourceID]; booking != nil {
			model.Update(ctx, map[string]any{"id": booking.ID}, data)
			continue
		}
		data["resource_id"] = resourceID
		data["created_at"] = now
		if model.Insert(ctx, data) == 0 {
			return fmt.Errorf("公共资源预约失败")
		}
	}
	return nil
}

func ValidateResourceBookingTime(ctx context.Context, currentID uint64, resourceID uint64, startAt time.Time, endAt time.Time) error {
	if resourceID == 0 {
		return fmt.Errorf("公共资源不能为空")
	}
	if !endAt.After(startAt) {
		return fmt.Errorf("结束时间必须晚于开始时间")
	}
	for _, booking := range crmmodel.NewPublicResourceBookingModel().Select(ctx, map[string]any{"resource_id": resourceID}) {
		if booking == nil || booking.ID == currentID || ResourceBookingInactive(booking.BookingStatus) {
			continue
		}
		if startAt.Before(booking.EndAt) && endAt.After(booking.StartAt) {
			return fmt.Errorf("会议室该时间段预约已满，请协调其他会议室或时间。")
		}
	}
	return nil
}

func ResourceBookingInactive(status string) bool {
	return status == crmmodel.ResourceBookingStatusCanceled ||
		status == crmmodel.ResourceBookingStatusRejected ||
		status == crmmodel.ResourceBookingStatusDone
}

func cancelScheduleResources(ctx context.Context, eventID uint64, changedAt time.Time) {
	crmmodel.NewPublicResourceBookingModel().Update(ctx, map[string]any{
		"schedule_event_id": eventID,
	}, map[string]any{
		"booking_status": crmmodel.ResourceBookingStatusCanceled,
		"updated_at":     changedAt,
	})
}

func completeScheduleResources(ctx context.Context, eventID uint64, changedAt time.Time) {
	crmmodel.NewPublicResourceBookingModel().Update(ctx, map[string]any{
		"schedule_event_id": eventID,
	}, map[string]any{
		"booking_status": crmmodel.ResourceBookingStatusDone,
		"updated_at":     changedAt,
	})
}

func scheduleResourceIDs(ctx context.Context, eventID uint64) []uint64 {
	rows := crmmodel.NewPublicResourceBookingModel().Select(ctx, map[string]any{"schedule_event_id": eventID})
	result := make([]uint64, 0, len(rows))
	for _, booking := range rows {
		if booking != nil && !ResourceBookingInactive(booking.BookingStatus) {
			result = append(result, booking.ResourceID)
		}
	}
	return uniqueUint64Values(result)
}
