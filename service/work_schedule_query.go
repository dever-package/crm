package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func (WorkService) Schedules(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	startAt, endAt, err := scheduleQueryRange(payload)
	if err != nil {
		return nil, err
	}
	eventIDs := visibleScheduleEventIDs(ctx, staff)
	rows := make([]*crmmodel.ScheduleEvent, 0, len(eventIDs))
	if len(eventIDs) > 0 {
		rows = crmmodel.NewScheduleEventModel().Select(ctx, map[string]any{"id": eventIDs})
	}
	status := firstText(payload, "status")
	events := make([]map[string]any, 0, len(rows))
	for _, event := range rows {
		if event == nil || status != "" && event.Status != status {
			continue
		}
		if !event.StartAt.Before(endAt) || !event.EndAt.After(startAt) {
			continue
		}
		row := scheduleEventResult(ctx, event)
		row["can_edit"] = canEditScheduleEvent(ctx, staff, event)
		events = append(events, row)
	}
	sort.SliceStable(events, func(i, j int) bool {
		return workTimeValue(events[i]["start_at"]).Before(workTimeValue(events[j]["start_at"]))
	})
	return map[string]any{
		"list":        events,
		"total":       len(events),
		"range_start": startAt,
		"range_end":   endAt,
	}, nil
}

func (WorkService) ScheduleOptions(ctx context.Context, staff *WorkStaffSession) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	return map[string]any{
		"customers": scheduleCustomerOptions(ctx, staff),
		"staff":     scheduleStaffOptions(ctx),
		"resources": scheduleResourceOptions(ctx),
		"reminders": crmmodel.ScheduleReminderOptions(),
		"config":    customerFollowConfiguration(ctx),
	}, nil
}

func (WorkService) ScheduleReminders(ctx context.Context, staff *WorkStaffSession) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	now := time.Now()
	rows := crmmodel.NewScheduleParticipantModel().Select(ctx, map[string]any{"staff_id": staff.ID})
	reminders := make([]map[string]any, 0)
	for _, participant := range rows {
		if participant == nil || participant.WorkbenchReadAt != nil {
			continue
		}
		event := crmmodel.NewScheduleEventModel().Find(ctx, map[string]any{
			"id":     participant.ScheduleEventID,
			"status": crmmodel.ScheduleStatusPending,
		})
		if event == nil || event.RemindAt.After(now) {
			continue
		}
		row := scheduleEventResult(ctx, event)
		row["can_edit"] = canEditScheduleEvent(ctx, staff, event)
		reminders = append(reminders, row)
	}
	sort.SliceStable(reminders, func(i, j int) bool {
		return workTimeValue(reminders[i]["remind_at"]).Before(workTimeValue(reminders[j]["remind_at"]))
	})
	return map[string]any{"list": reminders, "total": len(reminders)}, nil
}

func (WorkService) ReadScheduleReminder(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	eventID := firstUint64(payload, "schedule_event_id", "scheduleEventId", "id")
	participant := crmmodel.NewScheduleParticipantModel().Find(ctx, map[string]any{
		"schedule_event_id": eventID,
		"staff_id":          staff.ID,
	})
	if participant == nil {
		return nil, fmt.Errorf("日程提醒不存在")
	}
	readAt := time.Now()
	crmmodel.NewScheduleParticipantModel().Update(ctx, map[string]any{"id": participant.ID}, map[string]any{
		"workbench_read_at": readAt,
		"updated_at":        readAt,
	})
	return map[string]any{"read": true, "id": eventID}, nil
}

func scheduleQueryRange(payload map[string]any) (time.Time, time.Time, error) {
	startAt, err := parseScheduleTime(firstPresent(payload, "start_at", "startAt"))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endAt, err := parseScheduleTime(firstPresent(payload, "end_at", "endAt"))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	now := time.Now().In(scheduleLocation())
	if startAt.IsZero() {
		startAt = workBeginningOfDay(now).AddDate(0, 0, -7)
	}
	if endAt.IsZero() {
		endAt = workBeginningOfDay(now).AddDate(0, 0, 35)
	}
	if !endAt.After(startAt) {
		return time.Time{}, time.Time{}, fmt.Errorf("日历结束时间必须晚于开始时间")
	}
	if endAt.Sub(startAt) > 370*24*time.Hour {
		return time.Time{}, time.Time{}, fmt.Errorf("单次最多查询一年日程")
	}
	return startAt, endAt, nil
}

func visibleScheduleEventIDs(ctx context.Context, staff *WorkStaffSession) []uint64 {
	if staff == nil || staff.ID == 0 {
		return []uint64{}
	}
	seen := map[uint64]bool{}
	result := make([]uint64, 0)
	appendID := func(eventID uint64) {
		if eventID == 0 || seen[eventID] {
			return
		}
		seen[eventID] = true
		result = append(result, eventID)
	}
	for _, participant := range crmmodel.NewScheduleParticipantModel().Select(ctx, map[string]any{"staff_id": staff.ID}) {
		if participant != nil {
			appendID(participant.ScheduleEventID)
		}
	}
	for _, event := range crmmodel.NewScheduleEventModel().Select(ctx, map[string]any{"owner_staff_id": staff.ID}) {
		if event != nil {
			appendID(event.ID)
		}
	}
	if staff.CanDispatch {
		for _, event := range crmmodel.NewScheduleEventModel().Select(ctx, map[string]any{}) {
			if event != nil {
				appendID(event.ID)
			}
		}
	}
	return result
}

func scheduleCustomerOptions(ctx context.Context, staff *WorkStaffSession) []map[string]any {
	seen := map[uint64]bool{}
	result := make([]map[string]any, 0)
	for _, customerID := range scheduleCustomerOptionIDs(ctx, staff) {
		if customerID == 0 || seen[customerID] || !canScheduleCustomerFollow(ctx, staff, customerID) {
			continue
		}
		customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
		if customer == nil {
			continue
		}
		seen[customerID] = true
		row := map[string]any{
			"id":    customer.ID,
			"name":  customer.Name,
			"phone": customer.Phone,
		}
		if event := findPendingCustomerFollowEvent(ctx, customer.ID); event != nil {
			row["next_follow_at"] = event.StartAt
			row["schedule_event_id"] = event.ID
		}
		if instance := currentWorkTargetInstance(ctx, staff, customer.ID, 0); instance != nil {
			row["owner_staff_id"] = instance.OwnerStaffID
			if owner := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": instance.OwnerStaffID}); owner != nil {
				row["owner_staff_name"] = owner.Name
			}
		}
		result = append(result, row)
	}
	sort.SliceStable(result, func(i, j int) bool { return inputText(result[i]["name"]) < inputText(result[j]["name"]) })
	return result
}

func scheduleCustomerOptionIDs(ctx context.Context, staff *WorkStaffSession) []uint64 {
	if staff == nil || staff.ID == 0 {
		return []uint64{}
	}
	seen := map[uint64]bool{}
	result := make([]uint64, 0)
	appendCustomer := func(customerID uint64) {
		if customerID == 0 || seen[customerID] {
			return
		}
		seen[customerID] = true
		result = append(result, customerID)
	}
	if staff.CanDispatch {
		for _, instance := range crmmodel.NewWorkflowInstanceModel().Select(ctx, map[string]any{
			"status": crmmodel.ProgressStatusActive,
		}) {
			if instance != nil {
				appendCustomer(instance.CustomerID)
			}
		}
		for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
			"status": crmmodel.WorkTodoStatusPending,
		}) {
			if todo != nil {
				appendCustomer(todo.CustomerID)
			}
		}
		return result
	}
	for _, target := range workPendingTargets(ctx, staff) {
		appendCustomer(target.customerID)
	}
	return result
}

func scheduleStaffOptions(ctx context.Context) []map[string]any {
	staffRows := crmmodel.NewStaffModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	result := make([]map[string]any, 0, len(staffRows))
	for _, staff := range staffRows {
		if staff == nil {
			continue
		}
		result = append(result, map[string]any{
			"id":            staff.ID,
			"name":          staff.Name,
			"department_id": staff.DepartmentID,
		})
	}
	return result
}

func scheduleResourceOptions(ctx context.Context) []map[string]any {
	resources := crmmodel.NewPublicResourceModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	result := make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		result = append(result, map[string]any{
			"id":       resource.ID,
			"name":     resource.Name,
			"location": resource.Location,
			"capacity": resource.Capacity,
		})
	}
	return result
}

func customerFollowConfiguration(ctx context.Context) map[string]any {
	binding, err := resolveCustomerFollowFieldBinding(ctx)
	if err != nil {
		return map[string]any{"ready": false, "message": err.Error()}
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": binding.FieldID})
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": binding.TemplateID})
	result := map[string]any{
		"ready":               true,
		"data_usage_field_id": binding.UsageFieldID,
		"data_template_id":    binding.TemplateID,
		"data_field_id":       binding.FieldID,
	}
	if field != nil {
		result["field_name"] = field.Name
	}
	if template != nil {
		result["template_name"] = template.Name
	}
	return result
}
