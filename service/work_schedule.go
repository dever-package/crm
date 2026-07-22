package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

type scheduleArrangeInput struct {
	CustomerID               uint64
	OwnerStaffID             uint64
	OperatorStaffID          uint64
	OperatorDepartmentID     uint64
	SourceWorkflowInstanceID uint64
	Title                    string
	Remark                   string
	StartAt                  time.Time
	EndAt                    time.Time
	ReminderMinutes          int
	Source                   string
	ParticipantIDs           []uint64
	ResourceIDs              []uint64
	ReplaceParticipants      bool
	ReplaceResources         bool
	SkipOperation            bool
	TaskID                   uint64
}

func (WorkService) ScheduleCalendar(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	startAt, endAt, reminderMinutes, err := scheduleTimesFromPayload(payload, nil)
	if err != nil {
		return nil, err
	}
	scheduleType := firstText(payload, "schedule_type", "scheduleType")
	if scheduleType == "" {
		scheduleType = crmmodel.ScheduleTypePersonal
	}
	var event *crmmodel.ScheduleEvent
	err = orm.Transaction(ctx, func(txCtx context.Context) error {
		var scheduleErr error
		switch scheduleType {
		case crmmodel.ScheduleTypePersonal:
			event, scheduleErr = arrangePersonalSchedule(txCtx, staff, payload, startAt, endAt, reminderMinutes)
			return scheduleErr
		case crmmodel.ScheduleTypeCustomerFollow:
			customerID := firstUint64(payload, "customer_id", "customerId")
			if !canScheduleCustomerFollow(txCtx, staff, customerID) {
				return fmt.Errorf("只能为当前有权限跟进的客户创建日程")
			}
			requestedEventID := firstUint64(payload, "schedule_event_id", "scheduleEventId", "id")
			existingEvent := findPendingCustomerFollowEvent(txCtx, customerID)
			if requestedEventID > 0 && (existingEvent == nil || existingEvent.ID != requestedEventID) {
				return fmt.Errorf("无权修改该客户跟进日程")
			}
			if existingEvent != nil && !canEditScheduleEvent(txCtx, staff, existingEvent) {
				return fmt.Errorf("无权修改该客户跟进日程")
			}
			instance := currentWorkTargetInstance(txCtx, staff, customerID, 0)
			fallbackOwnerStaffID := staff.ID
			workflowInstanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId")
			if instance != nil {
				fallbackOwnerStaffID = preferredUint64(instance.OwnerStaffID, fallbackOwnerStaffID)
				workflowInstanceID = preferredUint64(workflowInstanceID, instance.ID)
			}
			ownerStaffID, ownerErr := requestedScheduleOwnerID(
				txCtx,
				staff,
				firstUint64(payload, "owner_staff_id", "ownerStaffId"),
				fallbackOwnerStaffID,
				instance != nil,
			)
			if ownerErr != nil {
				return ownerErr
			}
			resourceIDs, resourcesProvided := scheduleIDsFromPayload(payload, "resource_ids", "resourceIds")
			participantIDs, participantsProvided := scheduleIDsFromPayload(payload, "participant_ids", "participantIds")
			participantIDs = uniqueScheduleStaffIDs(staff.ID, participantIDs)
			event, scheduleErr = arrangeCustomerFollow(txCtx, scheduleArrangeInput{
				CustomerID:               customerID,
				OwnerStaffID:             ownerStaffID,
				OperatorStaffID:          staff.ID,
				OperatorDepartmentID:     staff.DepartmentID,
				SourceWorkflowInstanceID: workflowInstanceID,
				Title:                    firstText(payload, "title"),
				Remark:                   firstText(payload, "remark"),
				StartAt:                  startAt,
				EndAt:                    endAt,
				ReminderMinutes:          reminderMinutes,
				Source:                   crmmodel.ScheduleSourceCalendar,
				ParticipantIDs:           participantIDs,
				ResourceIDs:              resourceIDs,
				ReplaceParticipants:      participantsProvided,
				ReplaceResources:         resourcesProvided,
			})
			return scheduleErr
		default:
			return fmt.Errorf("日程类型无效")
		}
	})
	if err != nil {
		return nil, err
	}
	return scheduleEventResult(ctx, event), nil
}

func (WorkService) RescheduleCalendar(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	eventID := firstUint64(payload, "schedule_event_id", "scheduleEventId", "id")
	event := crmmodel.NewScheduleEventModel().Find(ctx, map[string]any{"id": eventID})
	if !canEditScheduleEvent(ctx, staff, event) {
		return nil, fmt.Errorf("无权修改该日程")
	}
	if event.Status != crmmodel.ScheduleStatusPending {
		return nil, fmt.Errorf("只能调整待进行日程")
	}
	startAt, endAt, reminderMinutes, err := scheduleTimesFromPayload(payload, event)
	if err != nil {
		return nil, err
	}
	var updated *crmmodel.ScheduleEvent
	err = orm.Transaction(ctx, func(txCtx context.Context) error {
		current := crmmodel.NewScheduleEventModel().Find(txCtx, map[string]any{"id": event.ID})
		if current == nil || current.Status != crmmodel.ScheduleStatusPending {
			return fmt.Errorf("日程不存在或状态已变化")
		}
		if current.ScheduleType == crmmodel.ScheduleTypeCustomerFollow {
			var arrangeErr error
			updated, arrangeErr = arrangeCustomerFollow(txCtx, scheduleArrangeInput{
				CustomerID:               current.CustomerID,
				OwnerStaffID:             current.OwnerStaffID,
				OperatorStaffID:          staff.ID,
				OperatorDepartmentID:     staff.DepartmentID,
				SourceWorkflowInstanceID: current.SourceWorkflowInstanceID,
				Title:                    current.Title,
				Remark:                   current.Remark,
				StartAt:                  startAt,
				EndAt:                    endAt,
				ReminderMinutes:          reminderMinutes,
				Source:                   crmmodel.ScheduleSourceCalendar,
			})
			return arrangeErr
		}
		return reschedulePersonalEvent(txCtx, current, staff, startAt, endAt, reminderMinutes)
	})
	if err != nil {
		return nil, err
	}
	if updated == nil {
		updated = crmmodel.NewScheduleEventModel().Find(ctx, map[string]any{"id": event.ID})
	}
	return scheduleEventResult(ctx, updated), nil
}

func (WorkService) CompleteCalendar(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	eventID := firstUint64(payload, "schedule_event_id", "scheduleEventId", "id")
	var result map[string]any
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		event := crmmodel.NewScheduleEventModel().Find(txCtx, map[string]any{"id": eventID})
		if !canEditScheduleEvent(txCtx, staff, event) {
			return fmt.Errorf("无权完成该日程")
		}
		if event.Status != crmmodel.ScheduleStatusPending {
			return fmt.Errorf("日程已处理")
		}
		now := time.Now()
		if event.ScheduleType == crmmodel.ScheduleTypeCustomerFollow {
			recordCustomerScheduleOperation(txCtx, event, staff.ID, staff.DepartmentID, "completed", event.StartAt, time.Time{}, firstText(payload, "remark"))
		}
		if crmmodel.NewScheduleEventModel().Update(txCtx, map[string]any{"id": event.ID, "status": crmmodel.ScheduleStatusPending}, map[string]any{
			"pending_customer_key": nil,
			"status":               crmmodel.ScheduleStatusCompleted,
			"completed_at":         now,
			"updated_at":           now,
		}) == 0 {
			return fmt.Errorf("日程状态已变化")
		}
		completeScheduleResources(txCtx, event.ID, now)
		nextStart, parseErr := parseScheduleTime(firstPresent(payload, "next_start_at", "nextStartAt"))
		if parseErr != nil {
			return parseErr
		}
		result = map[string]any{"completed": true, "id": event.ID}
		if nextStart.IsZero() {
			return nil
		}
		if event.ScheduleType != crmmodel.ScheduleTypeCustomerFollow {
			return fmt.Errorf("只有客户跟进日程可以完成并安排下一次")
		}
		participantIDs := scheduleParticipantIDs(txCtx, event.ID)
		nextEvent, arrangeErr := arrangeCustomerFollow(txCtx, scheduleArrangeInput{
			CustomerID:               event.CustomerID,
			OwnerStaffID:             event.OwnerStaffID,
			OperatorStaffID:          staff.ID,
			OperatorDepartmentID:     staff.DepartmentID,
			SourceWorkflowInstanceID: event.SourceWorkflowInstanceID,
			Title:                    event.Title,
			Remark:                   firstText(payload, "next_remark", "nextRemark"),
			StartAt:                  nextStart,
			EndAt:                    nextStart.Add(defaultScheduleDuration),
			ReminderMinutes:          event.ReminderMinutes,
			Source:                   crmmodel.ScheduleSourceCalendar,
			ParticipantIDs:           participantIDs,
			ReplaceParticipants:      true,
		})
		if arrangeErr != nil {
			return arrangeErr
		}
		result["next"] = scheduleEventResult(txCtx, nextEvent)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (WorkService) CancelCalendar(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	eventID := firstUint64(payload, "schedule_event_id", "scheduleEventId", "id")
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		event := crmmodel.NewScheduleEventModel().Find(txCtx, map[string]any{"id": eventID})
		if !canEditScheduleEvent(txCtx, staff, event) {
			return fmt.Errorf("无权取消该日程")
		}
		return cancelScheduleEvent(txCtx, event, staff.ID, staff.DepartmentID, firstText(payload, "remark"))
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"canceled": true, "id": eventID}, nil
}

func arrangeCustomerFollow(ctx context.Context, input scheduleArrangeInput) (*crmmodel.ScheduleEvent, error) {
	if input.CustomerID == 0 || input.OwnerStaffID == 0 {
		return nil, fmt.Errorf("客户和负责人不能为空")
	}
	if err := validateSchedulePeriod(input.StartAt, input.EndAt); err != nil {
		return nil, err
	}
	if !validScheduleReminderMinutes(input.ReminderMinutes) {
		return nil, fmt.Errorf("提醒时间无效")
	}
	if input.Title == "" {
		input.Title = customerFollowDefaultTitle(ctx, input.CustomerID)
	}
	now := time.Now()
	model := crmmodel.NewScheduleEventModel()
	event := model.Find(ctx, map[string]any{
		"customer_id":   input.CustomerID,
		"schedule_type": crmmodel.ScheduleTypeCustomerFollow,
		"status":        crmmodel.ScheduleStatusPending,
	}, map[string]any{"order": "id desc"})
	previousStart := time.Time{}
	reminderChanged := true
	if event != nil {
		previousStart = event.StartAt
		reminderChanged = !event.StartAt.Equal(input.StartAt) || event.ReminderMinutes != input.ReminderMinutes
		updates := map[string]any{
			"pending_customer_key":        customerFollowPendingKey(input.CustomerID),
			"owner_staff_id":              input.OwnerStaffID,
			"source_workflow_instance_id": preferredUint64(input.SourceWorkflowInstanceID, event.SourceWorkflowInstanceID),
			"source_task_id":              preferredUint64(input.TaskID, event.SourceTaskID),
			"title":                       input.Title,
			"remark":                      input.Remark,
			"start_at":                    input.StartAt,
			"end_at":                      input.EndAt,
			"reminder_minutes":            input.ReminderMinutes,
			"remind_at":                   scheduleReminderAt(input.StartAt, input.ReminderMinutes),
			"source":                      input.Source,
			"updated_at":                  now,
		}
		if model.Update(ctx, map[string]any{"id": event.ID, "status": crmmodel.ScheduleStatusPending}, updates) == 0 {
			return nil, fmt.Errorf("客户跟进日程已变化，请重试")
		}
		applyScheduleEventUpdates(event, updates)
	} else {
		record := map[string]any{
			"schedule_type":               crmmodel.ScheduleTypeCustomerFollow,
			"customer_id":                 input.CustomerID,
			"pending_customer_key":        customerFollowPendingKey(input.CustomerID),
			"owner_staff_id":              input.OwnerStaffID,
			"created_by_staff_id":         preferredUint64(input.OperatorStaffID, input.OwnerStaffID),
			"source_workflow_instance_id": input.SourceWorkflowInstanceID,
			"source_task_id":              input.TaskID,
			"operation_log_id":            uint64(0),
			"title":                       input.Title,
			"remark":                      input.Remark,
			"start_at":                    input.StartAt,
			"end_at":                      input.EndAt,
			"reminder_minutes":            input.ReminderMinutes,
			"remind_at":                   scheduleReminderAt(input.StartAt, input.ReminderMinutes),
			"source":                      input.Source,
			"status":                      crmmodel.ScheduleStatusPending,
			"created_at":                  now,
			"updated_at":                  now,
		}
		eventID := uint64(model.Insert(ctx, record))
		if eventID == 0 {
			return nil, fmt.Errorf("客户跟进日程创建失败")
		}
		event = model.Find(ctx, map[string]any{"id": eventID})
		if event == nil {
			return nil, fmt.Errorf("客户跟进日程创建后读取失败")
		}
	}
	operationID := uint64(0)
	if !input.SkipOperation && (previousStart.IsZero() || !previousStart.Equal(input.StartAt)) {
		action := "arranged"
		if !previousStart.IsZero() {
			action = "rescheduled"
		}
		operationID = recordCustomerScheduleOperation(ctx, event, input.OperatorStaffID, input.OperatorDepartmentID, action, previousStart, input.StartAt, input.Remark)
		if event.OperationLogID == 0 && operationID > 0 {
			model.Update(ctx, map[string]any{"id": event.ID}, map[string]any{"operation_log_id": operationID, "updated_at": now})
			event.OperationLogID = operationID
		}
	}
	if err := syncScheduleParticipants(ctx, event.ID, input.OwnerStaffID, input.ParticipantIDs, input.ReplaceParticipants, reminderChanged); err != nil {
		return nil, err
	}
	if err := syncScheduleResources(ctx, event, input.ResourceIDs, input.ReplaceResources, input.OperatorDepartmentID); err != nil {
		return nil, err
	}
	return event, nil
}

func arrangePersonalSchedule(ctx context.Context, staff *WorkStaffSession, payload map[string]any, startAt time.Time, endAt time.Time, reminderMinutes int) (*crmmodel.ScheduleEvent, error) {
	if err := validateSchedulePeriod(startAt, endAt); err != nil {
		return nil, err
	}
	ownerStaffID, err := requestedScheduleOwner(ctx, staff, payload)
	if err != nil {
		return nil, err
	}
	title := firstText(payload, "title")
	if title == "" {
		title = "个人日程"
	}
	model := crmmodel.NewScheduleEventModel()
	eventID := firstUint64(payload, "schedule_event_id", "scheduleEventId", "id")
	now := time.Now()
	resourceIDs, resourcesProvided := scheduleIDsFromPayload(payload, "resource_ids", "resourceIds")
	participantIDs, participantsProvided := scheduleIDsFromPayload(payload, "participant_ids", "participantIds")
	participantIDs = uniqueScheduleStaffIDs(staff.ID, participantIDs)
	var event *crmmodel.ScheduleEvent
	reminderChanged := true
	if eventID > 0 {
		event = model.Find(ctx, map[string]any{"id": eventID, "schedule_type": crmmodel.ScheduleTypePersonal})
		if !canEditScheduleEvent(ctx, staff, event) {
			return nil, fmt.Errorf("无权修改该日程")
		}
		reminderChanged = !event.StartAt.Equal(startAt) || event.ReminderMinutes != reminderMinutes
		updates := map[string]any{
			"owner_staff_id":   ownerStaffID,
			"title":            title,
			"remark":           firstText(payload, "remark"),
			"start_at":         startAt,
			"end_at":           endAt,
			"reminder_minutes": reminderMinutes,
			"remind_at":        scheduleReminderAt(startAt, reminderMinutes),
			"updated_at":       now,
		}
		if model.Update(ctx, map[string]any{"id": event.ID, "status": crmmodel.ScheduleStatusPending}, updates) == 0 {
			return nil, fmt.Errorf("个人日程不存在或状态已变化")
		}
		applyScheduleEventUpdates(event, updates)
	} else {
		eventID = uint64(model.Insert(ctx, map[string]any{
			"schedule_type":       crmmodel.ScheduleTypePersonal,
			"customer_id":         uint64(0),
			"owner_staff_id":      ownerStaffID,
			"created_by_staff_id": staff.ID,
			"title":               title,
			"remark":              firstText(payload, "remark"),
			"start_at":            startAt,
			"end_at":              endAt,
			"reminder_minutes":    reminderMinutes,
			"remind_at":           scheduleReminderAt(startAt, reminderMinutes),
			"source":              crmmodel.ScheduleSourceCalendar,
			"status":              crmmodel.ScheduleStatusPending,
			"created_at":          now,
			"updated_at":          now,
		}))
		event = model.Find(ctx, map[string]any{"id": eventID})
	}
	if event == nil {
		return nil, fmt.Errorf("个人日程保存失败")
	}
	if err := syncScheduleParticipants(ctx, event.ID, ownerStaffID, participantIDs, participantsProvided, reminderChanged); err != nil {
		return nil, err
	}
	if err := syncScheduleResources(ctx, event, resourceIDs, resourcesProvided, staff.DepartmentID); err != nil {
		return nil, err
	}
	return event, nil
}

func reschedulePersonalEvent(ctx context.Context, event *crmmodel.ScheduleEvent, staff *WorkStaffSession, startAt time.Time, endAt time.Time, reminderMinutes int) error {
	if event == nil {
		return fmt.Errorf("日程不存在")
	}
	reminderChanged := !event.StartAt.Equal(startAt) || event.ReminderMinutes != reminderMinutes
	if crmmodel.NewScheduleEventModel().Update(ctx, map[string]any{"id": event.ID, "status": crmmodel.ScheduleStatusPending}, map[string]any{
		"start_at":         startAt,
		"end_at":           endAt,
		"reminder_minutes": reminderMinutes,
		"remind_at":        scheduleReminderAt(startAt, reminderMinutes),
		"updated_at":       time.Now(),
	}) == 0 {
		return fmt.Errorf("个人日程不存在或状态已变化")
	}
	if reminderChanged {
		resetScheduleParticipantReminders(ctx, event.ID)
	}
	event.StartAt = startAt
	event.EndAt = endAt
	return syncScheduleResources(ctx, event, nil, false, staff.DepartmentID)
}

func cancelScheduleEvent(ctx context.Context, event *crmmodel.ScheduleEvent, operatorStaffID uint64, operatorDepartmentID uint64, remark string) error {
	if event == nil || event.Status != crmmodel.ScheduleStatusPending {
		return fmt.Errorf("日程不存在或已处理")
	}
	now := time.Now()
	if event.ScheduleType == crmmodel.ScheduleTypeCustomerFollow {
		recordCustomerScheduleOperation(ctx, event, operatorStaffID, operatorDepartmentID, "canceled", event.StartAt, time.Time{}, remark)
	}
	if crmmodel.NewScheduleEventModel().Update(ctx, map[string]any{"id": event.ID, "status": crmmodel.ScheduleStatusPending}, map[string]any{
		"pending_customer_key": nil,
		"status":               crmmodel.ScheduleStatusCanceled,
		"canceled_at":          now,
		"updated_at":           now,
	}) == 0 {
		return fmt.Errorf("日程状态已变化")
	}
	cancelScheduleResources(ctx, event.ID, now)
	return nil
}

func findPendingCustomerFollowEvent(ctx context.Context, customerID uint64) *crmmodel.ScheduleEvent {
	if customerID == 0 {
		return nil
	}
	return crmmodel.NewScheduleEventModel().Find(ctx, map[string]any{
		"customer_id":   customerID,
		"schedule_type": crmmodel.ScheduleTypeCustomerFollow,
		"status":        crmmodel.ScheduleStatusPending,
	}, map[string]any{"order": "id desc"})
}

func customerFollowPendingKey(customerID uint64) string {
	return fmt.Sprintf("customer:%d", customerID)
}

func scheduleTimesFromPayload(payload map[string]any, current *crmmodel.ScheduleEvent) (time.Time, time.Time, int, error) {
	startAt, err := parseScheduleTime(firstPresent(payload, "start_at", "startAt"))
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	if startAt.IsZero() && current != nil {
		startAt = current.StartAt
	}
	if startAt.IsZero() {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("开始时间不能为空")
	}
	endAt, err := parseScheduleTime(firstPresent(payload, "end_at", "endAt"))
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	if endAt.IsZero() {
		if current != nil && current.EndAt.After(current.StartAt) {
			endAt = startAt.Add(current.EndAt.Sub(current.StartAt))
		} else {
			endAt = startAt.Add(defaultScheduleDuration)
		}
	}
	reminderMinutes := defaultScheduleReminderMinutes
	if current != nil {
		reminderMinutes = current.ReminderMinutes
	}
	if value, exists := firstExisting(payload, "reminder_minutes", "reminderMinutes"); exists {
		reminderMinutes = inputInt(value)
	}
	if !validScheduleReminderMinutes(reminderMinutes) {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("提醒时间无效")
	}
	if err := validateSchedulePeriod(startAt, endAt); err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	return startAt, endAt, reminderMinutes, nil
}

func validateSchedulePeriod(startAt time.Time, endAt time.Time) error {
	if startAt.IsZero() {
		return fmt.Errorf("开始时间不能为空")
	}
	if !endAt.After(startAt) {
		return fmt.Errorf("结束时间必须晚于开始时间")
	}
	return nil
}

func requestedScheduleOwner(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (uint64, error) {
	return requestedScheduleOwnerID(
		ctx,
		staff,
		firstUint64(payload, "owner_staff_id", "ownerStaffId"),
		staff.ID,
		false,
	)
}

func requestedScheduleOwnerID(ctx context.Context, staff *WorkStaffSession, ownerStaffID uint64, fallbackStaffID uint64, allowFallbackOwner bool) (uint64, error) {
	explicitOwner := ownerStaffID > 0
	ownerStaffID = preferredUint64(ownerStaffID, fallbackStaffID)
	if ownerStaffID != staff.ID && !staff.CanDispatch && (explicitOwner || !allowFallbackOwner) {
		return 0, fmt.Errorf("无权指定其他日程负责人")
	}
	if crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": ownerStaffID, "status": crmmodel.StatusEnabled}) == nil {
		return 0, fmt.Errorf("日程负责人不存在或已停用")
	}
	return ownerStaffID, nil
}

func canScheduleCustomerFollow(ctx context.Context, staff *WorkStaffSession, customerID uint64) bool {
	if staff == nil || staff.ID == 0 || customerID == 0 {
		return false
	}
	if staff.CanDispatch {
		return customerHasPendingWork(ctx, customerID)
	}
	if !canViewWorkCustomer(ctx, staff, customerID) {
		return false
	}
	if crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
		"customer_id":    customerID,
		"owner_staff_id": staff.ID,
		"status":         crmmodel.ProgressStatusActive,
	}) != nil {
		return true
	}
	return crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
		"customer_id":       customerID,
		"assignee_staff_id": staff.ID,
		"status":            crmmodel.WorkTodoStatusPending,
	}) != nil
}

func customerHasPendingWork(ctx context.Context, customerID uint64) bool {
	if customerID == 0 {
		return false
	}
	if crmmodel.NewWorkflowInstanceModel().Count(ctx, map[string]any{
		"customer_id": customerID,
		"status":      crmmodel.ProgressStatusActive,
	}) > 0 {
		return true
	}
	return crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
		"customer_id": customerID,
		"status":      crmmodel.WorkTodoStatusPending,
	}) > 0
}

func canEditScheduleEvent(ctx context.Context, staff *WorkStaffSession, event *crmmodel.ScheduleEvent) bool {
	if staff == nil || staff.ID == 0 || event == nil {
		return false
	}
	if event.ScheduleType == crmmodel.ScheduleTypeMeeting {
		return false
	}
	if staff.CanDispatch || event.OwnerStaffID == staff.ID || event.CreatedByStaffID == staff.ID {
		return true
	}
	return event.ScheduleType == crmmodel.ScheduleTypeCustomerFollow &&
		canScheduleCustomerFollow(ctx, staff, event.CustomerID)
}

func scheduleIDsFromPayload(payload map[string]any, keys ...string) ([]uint64, bool) {
	value, exists := firstExisting(payload, keys...)
	if !exists {
		return nil, false
	}
	return uint64ListFromAny(value), true
}

func firstExisting(row map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, exists := row[key]; exists {
			return value, true
		}
	}
	return nil, false
}

func preferredUint64(value uint64, fallback uint64) uint64 {
	if value > 0 {
		return value
	}
	return fallback
}

func applyScheduleEventUpdates(event *crmmodel.ScheduleEvent, updates map[string]any) {
	if event == nil {
		return
	}
	event.OwnerStaffID = inputUint64(updates["owner_staff_id"])
	event.SourceWorkflowInstanceID = inputUint64(updates["source_workflow_instance_id"])
	event.SourceTaskID = inputUint64(updates["source_task_id"])
	event.Title = inputText(updates["title"])
	event.Remark = inputText(updates["remark"])
	event.StartAt = workTimeValue(updates["start_at"])
	event.EndAt = workTimeValue(updates["end_at"])
	event.ReminderMinutes = inputInt(updates["reminder_minutes"])
	event.RemindAt = workTimeValue(updates["remind_at"])
	event.Source = inputText(updates["source"])
}

func scheduleEventResult(ctx context.Context, event *crmmodel.ScheduleEvent, sessions ...*WorkStaffSession) map[string]any {
	if event == nil {
		return map[string]any{}
	}
	result := map[string]any{
		"id":                            event.ID,
		"schedule_type":                 event.ScheduleType,
		"customer_id":                   event.CustomerID,
		"asset_id":                      event.AssetID,
		"owner_staff_id":                event.OwnerStaffID,
		"source_workflow_instance_id":   event.SourceWorkflowInstanceID,
		"source_task_id":                event.SourceTaskID,
		"title":                         event.Title,
		"remark":                        event.Remark,
		"start_at":                      event.StartAt,
		"end_at":                        event.EndAt,
		"reminder_minutes":              event.ReminderMinutes,
		"remind_at":                     event.RemindAt,
		"source":                        event.Source,
		"status":                        event.Status,
		"meeting_attempt":               event.MeetingAttempt,
		"arrival_status":                event.ArrivalStatus,
		"arrival_confirmed_at":          event.ArrivalConfirmedAt,
		"arrival_confirmed_by_staff_id": event.ArrivalConfirmedByStaffID,
		"no_show_reason":                event.NoShowReason,
		"customer_arrived":              event.ArrivalStatus == crmmodel.MeetingArrivalArrived || event.CustomerArrivedAt != nil,
		"customer_arrived_at":           event.CustomerArrivedAt,
		"customer_arrived_by_staff_id":  event.CustomerArrivedByStaffID,
		"duration_minutes":              int(event.EndAt.Sub(event.StartAt).Minutes()),
		"participant_ids":               scheduleParticipantIDs(ctx, event.ID),
		"participants":                  scheduleParticipantResults(ctx, event.ID),
		"resource_ids":                  scheduleRecordedResourceIDs(ctx, event.ID),
	}
	if event.ScheduleType == crmmodel.ScheduleTypeMeeting {
		result["arrival_video_files"] = scheduleArrivalVideoFiles(ctx, event.ID)
	}
	if event.CustomerArrivedByStaffID > 0 {
		if arrivedBy := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": event.CustomerArrivedByStaffID}); arrivedBy != nil {
			result["customer_arrived_by_staff_name"] = arrivedBy.Name
		}
	}
	if event.ArrivalConfirmedByStaffID > 0 {
		if confirmedBy := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": event.ArrivalConfirmedByStaffID}); confirmedBy != nil {
			result["arrival_confirmed_by_staff_name"] = confirmedBy.Name
		}
	}
	staff := firstWorkScheduleSession(sessions)
	if staff == nil {
		staff = CurrentWorkStaff(ctx)
	}
	if staff != nil && staff.ID > 0 {
		result["can_manage_arrival_video"] = canManageMeetingEvidence(ctx, staff, event)
		participant := crmmodel.NewScheduleParticipantModel().Find(ctx, map[string]any{
			"schedule_event_id": event.ID,
			"staff_id":          staff.ID,
		})
		if participant != nil {
			result["is_participant"] = true
			result["checked_in_at"] = participant.CheckedInAt
			result["can_check_in"] = event.ScheduleType == crmmodel.ScheduleTypeMeeting &&
				event.Status == crmmodel.ScheduleStatusPending &&
				participant.CheckedInAt == nil && !time.Now().Before(event.StartAt)
		}
	}
	if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": event.CustomerID}); customer != nil {
		result["customer_name"] = customer.Name
		result["customer_phone"] = customer.Phone
	}
	return result
}

func firstWorkScheduleSession(sessions []*WorkStaffSession) *WorkStaffSession {
	for _, staff := range sessions {
		if staff != nil && staff.ID > 0 {
			return staff
		}
	}
	return nil
}
