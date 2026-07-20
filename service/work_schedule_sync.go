package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

const (
	defaultScheduleDuration        = 30 * time.Minute
	defaultScheduleReminderMinutes = crmmodel.ScheduleReminder30Min
	customerFollowTimeLayout       = "2006-01-02 15:04:05"
	workCustomerFollowKey          = "follow_up:start_at"
)

func workCustomerFollowFormFields() []map[string]any {
	return []map[string]any{
		{
			"id":          workCustomerFollowKey,
			"field_key":   workCustomerFollowKey,
			"field_type":  "datetime",
			"name":        "下次跟进时间",
			"required":    false,
			"readonly":    false,
			"group_key":   "system_customer_follow",
			"group_label": "客户跟进",
			"sort":        10,
		},
	}
}

func syncWorkCustomerFollowFromTaskForm(
	ctx context.Context,
	staff *WorkStaffSession,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	values map[string]any,
	operationID uint64,
) error {
	if task == nil || !task.CustomerFollowEnabled {
		return nil
	}
	if staff == nil || staff.ID == 0 || todo == nil || todo.CustomerID == 0 {
		return fmt.Errorf("客户跟进缺少任务、客户或办理人员")
	}
	startAt, err := parseScheduleTime(values[workCustomerFollowKey])
	if err != nil {
		return fmt.Errorf("下次跟进时间无效：%w", err)
	}
	existing := findPendingCustomerFollowEvent(ctx, todo.CustomerID)
	if startAt.IsZero() {
		if existing == nil {
			return nil
		}
		return cancelScheduleEvent(ctx, existing, staff.ID, staff.DepartmentID, "任务已清空下次跟进时间")
	}
	ownerStaffID, _ := customerFollowOperator(ctx, workDataOwnership{
		CustomerID:         todo.CustomerID,
		AssetID:            todo.AssetID,
		WorkflowInstanceID: todo.WorkflowInstanceID,
		CustomerProductID:  todo.CustomerProductID,
	}, operationID)
	if ownerStaffID == 0 {
		ownerStaffID = staff.ID
	}
	endAt := startAt.Add(defaultScheduleDuration)
	reminderMinutes := defaultScheduleReminderMinutes
	if existing != nil {
		if duration := existing.EndAt.Sub(existing.StartAt); duration > 0 {
			endAt = startAt.Add(duration)
		}
		reminderMinutes = existing.ReminderMinutes
	}
	_, err = arrangeCustomerFollow(ctx, scheduleArrangeInput{
		CustomerID:               todo.CustomerID,
		OwnerStaffID:             ownerStaffID,
		OperatorStaffID:          staff.ID,
		OperatorDepartmentID:     staff.DepartmentID,
		SourceWorkflowInstanceID: todo.WorkflowInstanceID,
		Title:                    customerFollowDefaultTitle(ctx, todo.CustomerID),
		StartAt:                  startAt,
		EndAt:                    endAt,
		ReminderMinutes:          reminderMinutes,
		Source:                   crmmodel.ScheduleSourceWorkForm,
		TaskID:                   task.ID,
	})
	return err
}

func parseScheduleTime(value any) (time.Time, error) {
	if emptyWorkFieldValue(value) {
		return time.Time{}, nil
	}
	location := scheduleLocation()
	switch typed := value.(type) {
	case time.Time:
		return typed.In(location), nil
	case *time.Time:
		if typed == nil {
			return time.Time{}, nil
		}
		return typed.In(location), nil
	}
	text := inputText(value)
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		customerFollowTimeLayout,
		"2006-01-02 15:04",
	} {
		if parsed, err := time.ParseInLocation(layout, text, location); err == nil {
			return parsed.In(location), nil
		}
	}
	return time.Time{}, fmt.Errorf("日程时间格式错误")
}

func scheduleLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		return location
	}
	return time.FixedZone("Asia/Shanghai", 8*60*60)
}

func customerFollowTimeValue(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.In(scheduleLocation()).Format(customerFollowTimeLayout)
}

func validScheduleReminderMinutes(value int) bool {
	switch value {
	case crmmodel.ScheduleReminderOnTime,
		crmmodel.ScheduleReminder10Min,
		crmmodel.ScheduleReminder30Min,
		crmmodel.ScheduleReminder1Hour,
		crmmodel.ScheduleReminder1Day:
		return true
	default:
		return false
	}
}

func scheduleReminderAt(startAt time.Time, reminderMinutes int) time.Time {
	return startAt.Add(-time.Duration(reminderMinutes) * time.Minute)
}

func customerFollowOperator(ctx context.Context, ownership workDataOwnership, operationID uint64) (uint64, uint64) {
	if ownership.WorkflowInstanceID > 0 {
		if instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": ownership.WorkflowInstanceID}); instance != nil && instance.OwnerStaffID > 0 {
			return instance.OwnerStaffID, instance.OwnerDepartmentID
		}
	}
	if operationID > 0 {
		if operation := crmmodel.NewOperationLogModel().Find(ctx, map[string]any{"id": operationID}); operation != nil && operation.OperatorStaffID > 0 {
			return operation.OperatorStaffID, operation.OperatorDepartmentID
		}
	}
	if instance := currentWorkTargetInstance(ctx, nil, ownership.CustomerID, ownership.AssetID); instance != nil {
		return instance.OwnerStaffID, instance.OwnerDepartmentID
	}
	return 0, 0
}

func customerFollowDefaultTitle(ctx context.Context, customerID uint64) string {
	customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
	if customer == nil || strings.TrimSpace(customer.Name) == "" {
		return "客户跟进"
	}
	return "跟进 - " + strings.TrimSpace(customer.Name)
}
