package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

const (
	minimumMeetingDurationMinutes = 15
	maximumMeetingDurationMinutes = 8 * 60
	workMeetingStartFieldKey      = "meeting:start_at"
	workMeetingDurationFieldKey   = "meeting:duration_minutes"
	workMeetingResourceFieldKey   = "meeting:resource_id"
	workMeetingArrivalFieldKey    = "meeting:customer_arrived"
	workMeetingGroupKey           = "system_meeting"
	workMeetingGroupLabel         = "会议预约"
)

func workMeetingFormFields(ctx context.Context) []map[string]any {
	return []map[string]any{
		{
			"id":          workMeetingStartFieldKey,
			"field_key":   workMeetingStartFieldKey,
			"field_type":  "datetime",
			"name":        "预约时间",
			"required":    true,
			"readonly":    false,
			"group_key":   workMeetingGroupKey,
			"group_label": workMeetingGroupLabel,
			"sort":        10,
		},
		{
			"id":            workMeetingDurationFieldKey,
			"field_key":     workMeetingDurationFieldKey,
			"field_type":    "number",
			"name":          "会议时长（分钟）",
			"required":      true,
			"readonly":      false,
			"default_value": 60,
			"group_key":     workMeetingGroupKey,
			"group_label":   workMeetingGroupLabel,
			"sort":          20,
		},
		{
			"id":            workMeetingResourceFieldKey,
			"field_key":     workMeetingResourceFieldKey,
			"field_type":    "public_resource",
			"name":          "会议室",
			"required":      true,
			"readonly":      false,
			"group_key":     workMeetingGroupKey,
			"group_label":   workMeetingGroupLabel,
			"sort":          30,
			"options":       workPublicResourceOptions(ctx),
			"option_source": "public_resource",
		},
	}
}

func workMeetingTaskFormFields(ctx context.Context, task map[string]any, configuredFields []map[string]any) []map[string]any {
	fields := workMeetingFormFields(ctx)
	event := findWorkMeetingEvent(ctx, inputUint64(task["workflow_instance_id"]), inputUint64(task["id"]))
	applyWorkMeetingFormDefaults(ctx, fields, event)
	for _, field := range configuredFields {
		if inputText(field["field_type"]) == "group" {
			fields = append(fields, field)
			continue
		}
		groupedField := copyMap(field)
		delete(groupedField, "group_id")
		groupedField["group_key"] = workMeetingGroupKey
		groupedField["group_label"] = workMeetingGroupLabel
		fields = append(fields, groupedField)
	}
	fields = append(fields, workMeetingArrivalFormField(event))
	return fields
}

func workMeetingArrivalFormField(event *crmmodel.ScheduleEvent) map[string]any {
	return map[string]any{
		"id":            workMeetingArrivalFieldKey,
		"field_key":     workMeetingArrivalFieldKey,
		"field_type":    "boolean",
		"name":          "客户已到访",
		"required":      false,
		"readonly":      false,
		"default_value": event != nil && event.CustomerArrivedAt != nil,
		"group_key":     workMeetingGroupKey,
		"group_label":   workMeetingGroupLabel,
		"sort":          1000,
	}
}

func applyWorkMeetingFormDefaults(ctx context.Context, fields []map[string]any, event *crmmodel.ScheduleEvent) {
	values := workMeetingEventValues(ctx, event)
	for _, field := range fields {
		key := inputText(field["field_key"])
		if value, exists := values[key]; exists {
			field["default_value"] = value
		}
	}
}

func workMeetingSourceKey(workflowInstanceID uint64, taskID uint64) string {
	if workflowInstanceID == 0 || taskID == 0 {
		return ""
	}
	return fmt.Sprintf("workflow:%d:task:%d", workflowInstanceID, taskID)
}

func findWorkMeetingEvent(ctx context.Context, workflowInstanceID uint64, taskID uint64) *crmmodel.ScheduleEvent {
	meetingKey := workMeetingSourceKey(workflowInstanceID, taskID)
	if meetingKey == "" {
		return nil
	}
	return crmmodel.NewScheduleEventModel().Find(ctx, map[string]any{"meeting_source_key": meetingKey})
}

func workMeetingEventValues(ctx context.Context, event *crmmodel.ScheduleEvent) map[string]any {
	if event == nil {
		return map[string]any{}
	}
	values := map[string]any{
		workMeetingStartFieldKey:    customerFollowTimeValue(event.StartAt),
		workMeetingDurationFieldKey: int(math.Round(event.EndAt.Sub(event.StartAt).Minutes())),
		workMeetingArrivalFieldKey:  event.CustomerArrivedAt != nil,
	}
	if resourceIDs := scheduleResourceIDs(ctx, event.ID); len(resourceIDs) > 0 {
		values[workMeetingResourceFieldKey] = resourceIDs[0]
	}
	return values
}

func normalizeWorkMeetingAuditValue(key string, value any) any {
	switch key {
	case workMeetingStartFieldKey:
		if startAt, err := parseScheduleTime(value); err == nil && !startAt.IsZero() {
			return customerFollowTimeValue(startAt)
		}
	case workMeetingDurationFieldKey:
		return int(math.Round(numericValue(value)))
	case workMeetingResourceFieldKey:
		return inputUint64(value)
	}
	return value
}

func syncWorkMeetingFromTaskForm(
	ctx context.Context,
	staff *WorkStaffSession,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	values map[string]any,
	operationID uint64,
) error {
	if task == nil || !task.MeetingEnabled {
		return nil
	}
	if staff == nil || staff.ID == 0 || todo == nil {
		return fmt.Errorf("会议预约缺少办理人员或流程待办")
	}
	startAt, err := parseScheduleTime(values[workMeetingStartFieldKey])
	if err != nil {
		return fmt.Errorf("预约时间无效：%w", err)
	}
	if startAt.IsZero() {
		return fmt.Errorf("预约时间不能为空")
	}
	durationMinutes := int(math.Round(numericValue(values[workMeetingDurationFieldKey])))
	if durationMinutes < minimumMeetingDurationMinutes || durationMinutes > maximumMeetingDurationMinutes {
		return fmt.Errorf("会议时长必须在 %d 到 %d 分钟之间", minimumMeetingDurationMinutes, maximumMeetingDurationMinutes)
	}
	resourceID := inputUint64(values[workMeetingResourceFieldKey])
	if resourceID == 0 {
		return fmt.Errorf("请选择会议室")
	}

	meetingKey := workMeetingSourceKey(todo.WorkflowInstanceID, task.ID)
	now := time.Now()
	model := crmmodel.NewScheduleEventModel()
	event := model.Find(ctx, map[string]any{"meeting_source_key": meetingKey})
	title := meetingScheduleTitle(ctx, todo.CustomerID, todo.AssetID)
	endAt := startAt.Add(time.Duration(durationMinutes) * time.Minute)
	reminderChanged := true
	if event == nil {
		eventID := uint64(model.Insert(ctx, map[string]any{
			"schedule_type":               crmmodel.ScheduleTypeMeeting,
			"customer_id":                 todo.CustomerID,
			"asset_id":                    todo.AssetID,
			"owner_staff_id":              staff.ID,
			"created_by_staff_id":         staff.ID,
			"source_workflow_instance_id": todo.WorkflowInstanceID,
			"source_task_id":              task.ID,
			"meeting_source_key":          meetingKey,
			"operation_log_id":            operationID,
			"title":                       title,
			"remark":                      task.Name,
			"start_at":                    startAt,
			"end_at":                      endAt,
			"reminder_minutes":            defaultScheduleReminderMinutes,
			"remind_at":                   scheduleReminderAt(startAt, defaultScheduleReminderMinutes),
			"source":                      crmmodel.ScheduleSourceWorkForm,
			"status":                      crmmodel.ScheduleStatusPending,
			"created_at":                  now,
			"updated_at":                  now,
		}))
		if eventID == 0 {
			return fmt.Errorf("会议日程创建失败")
		}
		event = model.Find(ctx, map[string]any{"id": eventID})
	} else {
		if event.Status != crmmodel.ScheduleStatusPending {
			return fmt.Errorf("会议日程已处理，不能重复预约")
		}
		reminderChanged = !event.StartAt.Equal(startAt) || !event.EndAt.Equal(endAt)
		updates := map[string]any{
			"customer_id":                 todo.CustomerID,
			"asset_id":                    todo.AssetID,
			"owner_staff_id":              staff.ID,
			"source_workflow_instance_id": todo.WorkflowInstanceID,
			"source_task_id":              task.ID,
			"operation_log_id":            operationID,
			"title":                       title,
			"remark":                      task.Name,
			"start_at":                    startAt,
			"end_at":                      endAt,
			"reminder_minutes":            defaultScheduleReminderMinutes,
			"remind_at":                   scheduleReminderAt(startAt, defaultScheduleReminderMinutes),
			"source":                      crmmodel.ScheduleSourceWorkForm,
			"updated_at":                  now,
		}
		if model.Update(ctx, map[string]any{"id": event.ID, "status": crmmodel.ScheduleStatusPending}, updates) == 0 {
			return fmt.Errorf("会议日程已变化，请重试")
		}
		applyScheduleEventUpdates(event, updates)
		event.AssetID = todo.AssetID
		event.SourceTaskID = task.ID
		event.OperationLogID = operationID
	}
	if event == nil {
		return fmt.Errorf("会议日程创建后无法读取")
	}
	participantIDs := workflowMeetingParticipantIDs(ctx, todo.WorkflowInstanceID, staff.ID)
	if err := syncScheduleParticipants(ctx, event.ID, staff.ID, participantIDs, true, reminderChanged); err != nil {
		return err
	}
	return syncScheduleResources(ctx, event, []uint64{resourceID}, true, staff.DepartmentID)
}

func confirmWorkMeetingArrival(
	ctx context.Context,
	staff *WorkStaffSession,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	values map[string]any,
) error {
	if task == nil || !task.MeetingEnabled {
		return nil
	}
	if !booleanFromAny(values[workMeetingArrivalFieldKey]) {
		return fmt.Errorf("请确认客户已到访")
	}
	if staff == nil || staff.ID == 0 || todo == nil {
		return fmt.Errorf("客户到访确认缺少办理人员或流程待办")
	}
	event := findWorkMeetingEvent(ctx, todo.WorkflowInstanceID, task.ID)
	if event == nil || event.Status != crmmodel.ScheduleStatusPending {
		return fmt.Errorf("请先保存有效的会议预约")
	}
	now := time.Now()
	if now.Before(event.StartAt) {
		return fmt.Errorf("预约时间尚未到，暂不能确认客户到访")
	}
	if event.CustomerArrivedAt != nil {
		return nil
	}
	if crmmodel.NewScheduleEventModel().Update(ctx, map[string]any{
		"id":                  event.ID,
		"status":              crmmodel.ScheduleStatusPending,
		"customer_arrived_at": nil,
	}, map[string]any{
		"customer_arrived_at":          now,
		"customer_arrived_by_staff_id": staff.ID,
		"updated_at":                   now,
	}) == 0 {
		return fmt.Errorf("客户到访状态已变化，请刷新后重试")
	}
	return nil
}

func meetingScheduleTitle(ctx context.Context, customerID uint64, assetID uint64) string {
	name := ""
	if asset := crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{"id": assetID, "customer_id": customerID}); asset != nil {
		name = asset.AssetName
	}
	if name == "" {
		if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}); customer != nil {
			name = customer.Name
		}
	}
	if name == "" {
		return "案件会议"
	}
	return "案件会议 - " + name
}

func workflowMeetingParticipantIDs(ctx context.Context, workflowInstanceID uint64, ownerStaffID uint64) []uint64 {
	participantIDs := []uint64{ownerStaffID}
	assignedTaskIDs := map[uint64]bool{}
	for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"workflow_instance_id": workflowInstanceID,
	}) {
		if todo == nil || todo.AssigneeStaffID == 0 {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{
			"id":     todo.TaskID,
			"status": crmmodel.StatusEnabled,
		})
		if task != nil && task.IncludeInMeeting {
			participantIDs = append(participantIDs, todo.AssigneeStaffID)
			assignedTaskIDs[task.ID] = true
		}
	}
	instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": workflowInstanceID})
	participantIDs = append(participantIDs, configuredWorkflowMeetingParticipantIDs(ctx, instance, assignedTaskIDs)...)
	return uniqueUint64Values(participantIDs)
}

func configuredWorkflowMeetingParticipantIDs(
	ctx context.Context,
	instance *crmmodel.WorkflowInstance,
	assignedTaskIDs map[uint64]bool,
) []uint64 {
	if instance == nil || instance.WorkflowID == 0 {
		return nil
	}
	participantIDs := make([]uint64, 0)
	for _, stage := range crmmodel.NewStageModel().Select(ctx, map[string]any{
		"workflow_id": instance.WorkflowID,
		"status":      crmmodel.StatusEnabled,
	}) {
		if stage == nil {
			continue
		}
		for _, task := range crmmodel.NewTaskModel().Select(ctx, map[string]any{
			"stage_id": stage.ID,
			"status":   crmmodel.StatusEnabled,
		}) {
			if task == nil || !task.IncludeInMeeting || assignedTaskIDs[task.ID] {
				continue
			}
			_, staffID, err := resolveTaskAssignee(ctx, instance, task)
			if err == nil && staffID > 0 {
				participantIDs = append(participantIDs, staffID)
			}
		}
	}
	return participantIDs
}

func syncWorkflowMeetingParticipants(ctx context.Context, workflowInstanceID uint64) error {
	if workflowInstanceID == 0 {
		return nil
	}
	events := crmmodel.NewScheduleEventModel().Select(ctx, map[string]any{
		"source_workflow_instance_id": workflowInstanceID,
		"schedule_type":               crmmodel.ScheduleTypeMeeting,
		"status":                      crmmodel.ScheduleStatusPending,
	})
	for _, event := range events {
		if event == nil || event.OwnerStaffID == 0 {
			continue
		}
		participantIDs := workflowMeetingParticipantIDs(ctx, workflowInstanceID, event.OwnerStaffID)
		if err := syncScheduleParticipants(ctx, event.ID, event.OwnerStaffID, participantIDs, true, false); err != nil {
			return err
		}
	}
	return nil
}

func (WorkService) CheckInSchedule(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	eventID := firstUint64(payload, "schedule_event_id", "scheduleEventId", "id")
	var checkedInAt time.Time
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		event := crmmodel.NewScheduleEventModel().Find(txCtx, map[string]any{
			"id":            eventID,
			"schedule_type": crmmodel.ScheduleTypeMeeting,
			"status":        crmmodel.ScheduleStatusPending,
		})
		if event == nil {
			return fmt.Errorf("会议不存在或已结束")
		}
		participant := crmmodel.NewScheduleParticipantModel().Find(txCtx, map[string]any{
			"schedule_event_id": event.ID,
			"staff_id":          staff.ID,
		})
		if participant == nil {
			return fmt.Errorf("当前人员不在会议参与人中")
		}
		if participant.CheckedInAt != nil {
			checkedInAt = *participant.CheckedInAt
			return nil
		}
		checkedInAt = time.Now()
		if checkedInAt.Before(event.StartAt) {
			return fmt.Errorf("会议尚未开始")
		}
		if crmmodel.NewScheduleParticipantModel().Update(txCtx, map[string]any{
			"id":            participant.ID,
			"checked_in_at": nil,
		}, map[string]any{
			"checked_in_at":     checkedInAt,
			"workbench_read_at": checkedInAt,
			"updated_at":        checkedInAt,
		}) == 0 {
			return fmt.Errorf("签到状态已变化，请刷新后重试")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"checked_in": true, "id": eventID, "checked_in_at": checkedInAt}, nil
}
