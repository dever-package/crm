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
)

type customerFollowFieldBinding struct {
	UsageID      uint64
	UsageFieldID uint64
	TemplateID   uint64
	FieldID      uint64
}

func resolveCustomerFollowFieldBinding(ctx context.Context) (*customerFollowFieldBinding, error) {
	bindings := workDataUsageFieldsByType(ctx, crmmodel.DataUsageTypeCustomerFollowAt)
	if len(bindings) == 0 {
		return nil, fmt.Errorf("请先在系统用途中绑定客户下次跟进时间字段")
	}
	if len(bindings) != 1 || bindings[0] == nil {
		return nil, fmt.Errorf("客户下次跟进时间只能绑定一个启用字段")
	}
	usageField := bindings[0]
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":     usageField.DataFieldID,
		"status": crmmodel.StatusEnabled,
	})
	if field == nil || field.FieldType != "datetime" {
		return nil, fmt.Errorf("客户下次跟进时间必须绑定启用的日期时间字段")
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
		"id":     field.DataTemplateID,
		"status": crmmodel.StatusEnabled,
	})
	if template == nil || template.CateID != crmmodel.CustomerDataTemplateCateID {
		return nil, fmt.Errorf("客户下次跟进时间必须绑定客户信息模板字段")
	}
	return &customerFollowFieldBinding{
		UsageID:      usageField.UsageID,
		UsageFieldID: usageField.ID,
		TemplateID:   template.ID,
		FieldID:      field.ID,
	}, nil
}

func customerFollowBindingForTemplate(ctx context.Context, templateID uint64) (*customerFollowFieldBinding, error) {
	bindings := workDataUsageFieldsByType(ctx, crmmodel.DataUsageTypeCustomerFollowAt)
	if len(bindings) == 0 {
		return nil, nil
	}
	binding, err := resolveCustomerFollowFieldBinding(ctx)
	if err != nil {
		return nil, err
	}
	if binding.TemplateID != templateID {
		return nil, nil
	}
	return binding, nil
}

func customerFollowSubmittedValue(record map[string]any, fieldID uint64) (any, bool) {
	if fieldID == 0 || len(record) == 0 {
		return nil, false
	}
	value, exists := record[fmt.Sprintf("%d", fieldID)]
	return value, exists
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

func customerFollowDataRecord(ctx context.Context, binding *customerFollowFieldBinding, customerID uint64, recordID uint64) *crmmodel.DataRecord {
	if binding == nil || customerID == 0 {
		return nil
	}
	model := crmmodel.NewDataRecordModel()
	if recordID > 0 {
		if record := model.Find(ctx, map[string]any{
			"id":               recordID,
			"customer_id":      customerID,
			"data_template_id": binding.TemplateID,
			"status":           crmmodel.StatusEnabled,
		}); record != nil {
			return record
		}
	}
	return model.Find(ctx, workDataRecordOwnershipFilter(workDataOwnership{CustomerID: customerID}, binding.TemplateID))
}

func writeCustomerFollowTime(
	ctx context.Context,
	binding *customerFollowFieldBinding,
	customerID uint64,
	recordID uint64,
	startAt time.Time,
	taskID uint64,
	operationID uint64,
) (uint64, error) {
	if binding == nil || customerID == 0 {
		return 0, fmt.Errorf("客户跟进字段或客户不能为空")
	}
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return 0, fmt.Errorf("客户不存在")
	}
	now := time.Now()
	fieldKey := fmt.Sprintf("%d", binding.FieldID)
	value := customerFollowTimeValue(startAt)
	model := crmmodel.NewDataRecordModel()
	record := customerFollowDataRecord(ctx, binding, customerID, recordID)
	data := map[string]any{
		"customer_id":          customerID,
		"asset_id":             uint64(0),
		"workflow_instance_id": uint64(0),
		"customer_product_id":  uint64(0),
		"data_template_id":     binding.TemplateID,
		"task_id":              taskID,
		"operation_log_id":     operationID,
		"summary":              "",
		"status":               crmmodel.StatusEnabled,
		"sort":                 100,
		"updated_at":           now,
	}
	values := map[string]any{}
	if record != nil {
		values = mapFromAny(record.RecordJSON)
	}
	values[fieldKey] = value
	data["record_json"] = jsonText(values)
	if record != nil {
		if model.Update(ctx, map[string]any{"id": record.ID}, data) == 0 {
			return 0, fmt.Errorf("客户跟进时间更新失败")
		}
		recordID = record.ID
	} else {
		data["created_at"] = now
		recordID = uint64(model.Insert(ctx, data))
		if recordID == 0 {
			return 0, fmt.Errorf("客户跟进资料创建失败")
		}
	}
	syncWorkStatFieldValues(
		ctx,
		workDataOwnership{CustomerID: customerID},
		binding.TemplateID,
		taskID,
		operationID,
		map[string]any{fieldKey: value},
		now,
	)
	return recordID, nil
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
