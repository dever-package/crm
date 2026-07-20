package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func syncWorkDataFieldStatValues(
	ctx context.Context,
	ownership workDataOwnership,
	taskID uint64,
	operationID uint64,
	formInput *workFormInput,
) error {
	if taskID == 0 || operationID == 0 || formInput == nil {
		return nil
	}
	fields := workSubmittedDataFields(ctx, formInput)
	if len(fields) == 0 {
		return nil
	}

	changedAt := workOperationCreatedAt(ctx, operationID)
	for _, field := range fields {
		if field == nil || !field.StatEnabled || !workStatDataFieldSupported(field) {
			continue
		}
		value, valueOwnership, submitted := workSubmittedDataFieldValue(formInput, ownership, field)
		if !submitted {
			continue
		}
		if err := saveWorkDataFieldStatValue(ctx, valueOwnership, taskID, operationID, field, value, changedAt); err != nil {
			return err
		}
	}
	return nil
}

func workStatDataFieldSupported(field *crmmodel.DataField) bool {
	return field != nil && field.ID > 0 && field.FieldType != "group" && field.FieldType != "attachment"
}

func saveWorkDataFieldStatValue(
	ctx context.Context,
	ownership workDataOwnership,
	taskID uint64,
	operationID uint64,
	field *crmmodel.DataField,
	value any,
	changedAt time.Time,
) error {
	if field == nil || field.ID == 0 {
		return fmt.Errorf("统计字段不存在")
	}
	record := workDataFieldStatValueRecord(ctx, ownership, taskID, operationID, field, value, changedAt)
	filters := map[string]any{
		"lead_id":              ownership.LeadID,
		"customer_id":          ownership.CustomerID,
		"asset_id":             ownership.AssetID,
		"workflow_instance_id": ownership.WorkflowInstanceID,
		"data_field_id":        field.ID,
	}
	model := crmmodel.NewStatFieldValueModel()
	if existing := model.Find(ctx, filters); existing != nil {
		model.Update(ctx, map[string]any{"id": existing.ID}, record)
		return nil
	}
	record["created_at"] = changedAt
	if uint64(model.Insert(ctx, record)) == 0 {
		return fmt.Errorf("%s统计数据保存失败", field.Name)
	}
	return nil
}

func workDataFieldStatValueRecord(
	ctx context.Context,
	ownership workDataOwnership,
	taskID uint64,
	operationID uint64,
	field *crmmodel.DataField,
	value any,
	changedAt time.Time,
) map[string]any {
	displayValue := workDataFieldStatDisplayValue(ctx, field, value)
	return map[string]any{
		"lead_id":              ownership.LeadID,
		"customer_id":          ownership.CustomerID,
		"asset_id":             ownership.AssetID,
		"workflow_instance_id": ownership.WorkflowInstanceID,
		"customer_product_id":  ownership.CustomerProductID,
		"data_template_id":     field.DataTemplateID,
		"data_field_id":        field.ID,
		"field_key":            field.FieldKey,
		"field_name":           field.Name,
		"field_type":           field.FieldType,
		"stat_type":            workDataFieldStatType(field.FieldType),
		"stat_group":           "",
		"value_text":           displayValue,
		"value_number":         workDataFieldStatNumber(field.FieldType, value),
		"value_date":           workDataFieldStatDate(field.FieldType, value),
		"value_bool":           booleanFromAny(value),
		"value_json":           workDataFieldStatJSON(value),
		"source":               crmmodel.StatValueSourceForm,
		"task_id":              taskID,
		"operation_log_id":     operationID,
		"status":               crmmodel.StatusEnabled,
		"updated_at":           changedAt,
	}
}

func workDataFieldStatDisplayValue(ctx context.Context, field *crmmodel.DataField, value any) string {
	if field.FieldType == "boolean" {
		if booleanFromAny(value) {
			return "是"
		}
		return "否"
	}
	if field.FieldType == "checkbox" || field.FieldType == "multi_select" {
		if displayValue := workDataFieldOptionDisplayValue(ctx, field, value); displayValue != "" {
			return displayValue
		}
		return strings.Join(stringListFromAny(value), "、")
	}
	displayValue, _ := workDataFieldDisplayValue(ctx, field, value)
	return displayValue
}

func workDataFieldStatType(fieldType string) string {
	switch strings.TrimSpace(fieldType) {
	case "number":
		return crmmodel.DataFieldStatTypeMetric
	case "money":
		return crmmodel.DataFieldStatTypeAmount
	case "date", "datetime":
		return crmmodel.DataFieldStatTypeTime
	case "boolean":
		return crmmodel.DataFieldStatTypeStatus
	case "radio", "checkbox", "select", "multi_select":
		return crmmodel.DataFieldStatTypeDimension
	default:
		return crmmodel.DataFieldStatTypeText
	}
}

func workDataFieldStatNumber(fieldType string, value any) float64 {
	if fieldType != "number" && fieldType != "money" {
		return 0
	}
	return numericValue(value)
}

func workDataFieldStatDate(fieldType string, value any) time.Time {
	if fieldType != "date" && fieldType != "datetime" {
		return time.Time{}
	}
	text := inputText(value)
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func workDataFieldStatJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(raw)
}
