package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func recordWorkTaskOperation(
	ctx context.Context,
	staff *WorkStaffSession,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	resultValue string,
	content string,
	snapshot map[string]any,
	emitStat bool,
) uint64 {
	if todo == nil || task == nil {
		return 0
	}
	now := time.Now()
	staffID, departmentID := workOperatorIDs(staff)
	operationID := uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"customer_id":            todo.CustomerID,
		"asset_id":               todo.AssetID,
		"workflow_id":            todo.WorkflowID,
		"stage_id":               todo.StageID,
		"task_id":                task.ID,
		"task_type":              task.TaskType,
		"result_value":           resultValue,
		"title":                  task.Name,
		"content":                content,
		"data_snapshot_json":     jsonText(snapshot),
		"operator_staff_id":      staffID,
		"operator_department_id": departmentID,
		"created_at":             now,
	}))
	if emitStat && operationID > 0 {
		insertWorkStatEvent(ctx, map[string]any{
			"event_type":             crmmodel.StatEventTypeTask,
			"event_key":              fmt.Sprintf("task:%d:%s", task.ID, resultValue),
			"customer_id":            todo.CustomerID,
			"asset_id":               todo.AssetID,
			"workflow_id":            todo.WorkflowID,
			"stage_id":               todo.StageID,
			"from_stage_id":          uint64(0),
			"to_stage_id":            uint64(0),
			"task_id":                task.ID,
			"task_type":              task.TaskType,
			"result_value":           resultValue,
			"operation_log_id":       operationID,
			"operator_staff_id":      staffID,
			"operator_department_id": departmentID,
			"event_at":               now,
			"created_at":             now,
		})
	}
	return operationID
}

type workStageChange struct {
	FromWorkflowID uint64
	FromStageID    uint64
	ToWorkflowID   uint64
	ToStageID      uint64
	ResultValue    string
	Title          string
	Content        string
	Snapshot       map[string]any
}

func recordWorkStageChange(ctx context.Context, staff *WorkStaffSession, progress *crmmodel.CustomerStage, change workStageChange) uint64 {
	if progress == nil {
		return 0
	}
	now := time.Now()
	title := strings.TrimSpace(change.Title)
	if title == "" {
		title = "流程阶段变更"
	}
	workflowID := change.ToWorkflowID
	if workflowID == 0 {
		workflowID = change.FromWorkflowID
	}
	if workflowID == 0 {
		workflowID = progress.WorkflowID
	}
	stageID := change.ToStageID
	if stageID == 0 {
		stageID = change.FromStageID
	}
	if stageID == 0 {
		stageID = progress.StageID
	}
	snapshot := map[string]any{
		"from_workflow_id": change.FromWorkflowID,
		"from_stage_id":    change.FromStageID,
		"to_workflow_id":   change.ToWorkflowID,
		"to_stage_id":      change.ToStageID,
	}
	for key, value := range change.Snapshot {
		snapshot[key] = value
	}
	staffID, departmentID := workOperatorIDs(staff)
	operationID := uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"customer_id":            progress.CustomerID,
		"asset_id":               progress.AssetID,
		"workflow_id":            workflowID,
		"stage_id":               stageID,
		"task_id":                uint64(0),
		"task_type":              "",
		"result_value":           change.ResultValue,
		"title":                  title,
		"content":                change.Content,
		"data_snapshot_json":     jsonText(snapshot),
		"operator_staff_id":      staffID,
		"operator_department_id": departmentID,
		"created_at":             now,
	}))
	if operationID == 0 {
		return 0
	}
	insertWorkStatEvent(ctx, map[string]any{
		"event_type":             crmmodel.StatEventTypeTransition,
		"event_key":              fmt.Sprintf("transition:%d:%d:%d:%d", change.FromWorkflowID, change.FromStageID, change.ToWorkflowID, change.ToStageID),
		"customer_id":            progress.CustomerID,
		"asset_id":               progress.AssetID,
		"workflow_id":            workflowID,
		"stage_id":               stageID,
		"from_stage_id":          change.FromStageID,
		"to_stage_id":            change.ToStageID,
		"task_id":                uint64(0),
		"task_type":              "",
		"result_value":           change.ResultValue,
		"operation_log_id":       operationID,
		"operator_staff_id":      staffID,
		"operator_department_id": departmentID,
		"event_at":               now,
		"created_at":             now,
	})
	return operationID
}

func recordWorkManagementOperation(
	ctx context.Context,
	staff *WorkStaffSession,
	progress *crmmodel.CustomerStage,
	resultValue string,
	title string,
	content string,
	snapshot map[string]any,
) uint64 {
	if progress == nil {
		return 0
	}
	staffID, departmentID := workOperatorIDs(staff)
	return uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"customer_id":            progress.CustomerID,
		"asset_id":               progress.AssetID,
		"workflow_id":            progress.WorkflowID,
		"stage_id":               progress.StageID,
		"task_id":                uint64(0),
		"task_type":              "",
		"result_value":           resultValue,
		"title":                  title,
		"content":                content,
		"data_snapshot_json":     jsonText(snapshot),
		"operator_staff_id":      staffID,
		"operator_department_id": departmentID,
		"created_at":             time.Now(),
	}))
}

func recordWorkTodoAssignment(
	ctx context.Context,
	staff *WorkStaffSession,
	progress *crmmodel.CustomerStage,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	fromStaffID uint64,
	toStaff *crmmodel.Staff,
) uint64 {
	if progress == nil || todo == nil || task == nil || toStaff == nil {
		return 0
	}
	resultValue := "assigned"
	title := "任务已分配"
	if fromStaffID > 0 {
		resultValue = "reassigned"
		title = "任务已改派"
	}
	staffID, departmentID := workOperatorIDs(staff)
	return uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"customer_id":  progress.CustomerID,
		"asset_id":     progress.AssetID,
		"workflow_id":  progress.WorkflowID,
		"stage_id":     progress.StageID,
		"task_id":      task.ID,
		"task_type":    task.TaskType,
		"result_value": resultValue,
		"title":        title + "：" + task.Name,
		"content":      toStaff.Name,
		"data_snapshot_json": jsonText(map[string]any{
			"todo_id":       todo.ID,
			"from_staff_id": fromStaffID,
			"to_staff_id":   toStaff.ID,
		}),
		"operator_staff_id":      staffID,
		"operator_department_id": departmentID,
		"created_at":             time.Now(),
	}))
}

func workOperatorIDs(staff *WorkStaffSession) (uint64, uint64) {
	if staff == nil {
		return 0, 0
	}
	return staff.ID, staff.DepartmentID
}

func insertWorkStatEvent(ctx context.Context, record map[string]any) {
	model := crmmodel.NewStatEventModel()
	if model.Find(ctx, map[string]any{
		"event_key":        record["event_key"],
		"operation_log_id": record["operation_log_id"],
	}) != nil {
		return
	}
	model.Insert(ctx, record)
}

func currentWorkCustomerStage(ctx context.Context, customerID uint64, assetID uint64) *crmmodel.CustomerStage {
	if customerID == 0 || assetID == 0 {
		return nil
	}
	return crmmodel.NewCustomerStageModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	})
}

func ensureCurrentWorkCustomerStage(ctx context.Context, _ *WorkStaffSession, customerID uint64, assetID uint64) *crmmodel.CustomerStage {
	return currentWorkCustomerStage(ctx, customerID, assetID)
}

func saveWorkDataRecord(ctx context.Context, customerID uint64, assetID uint64, templateID uint64, taskID uint64, operationID uint64, record map[string]any) {
	saveWorkObjectDataRecord(ctx, customerID, assetID, 0, templateID, taskID, operationID, record)
}

func saveWorkObjectDataRecord(ctx context.Context, customerID uint64, assetID uint64, businessObjectID uint64, templateID uint64, taskID uint64, operationID uint64, record map[string]any) {
	now := time.Now()
	data := map[string]any{
		"customer_id":        customerID,
		"asset_id":           assetID,
		"business_object_id": businessObjectID,
		"data_template_id":   templateID,
		"task_id":            taskID,
		"operation_log_id":   operationID,
		"record_json":        jsonText(record),
		"summary":            "",
		"status":             crmmodel.StatusEnabled,
		"sort":               100,
		"updated_at":         now,
	}
	model := crmmodel.NewDataRecordModel()
	existing := model.Find(ctx, map[string]any{
		"customer_id":        customerID,
		"asset_id":           assetID,
		"business_object_id": businessObjectID,
		"data_template_id":   templateID,
		"status":             crmmodel.StatusEnabled,
	})
	if existing != nil {
		merged := mapFromAny(existing.RecordJSON)
		for key, value := range record {
			merged[key] = value
		}
		data["record_json"] = jsonText(merged)
		model.Update(ctx, map[string]any{"id": existing.ID}, data)
		syncWorkStatFieldValues(ctx, customerID, assetID, businessObjectID, templateID, taskID, operationID, record, now)
		return
	}
	data["created_at"] = now
	model.Insert(ctx, data)
	syncWorkStatFieldValues(ctx, customerID, assetID, businessObjectID, templateID, taskID, operationID, record, now)
}

func syncWorkStatFieldValues(ctx context.Context, customerID uint64, assetID uint64, businessObjectID uint64, templateID uint64, taskID uint64, operationID uint64, record map[string]any, changedAt time.Time) {
	defer func() { _ = recover() }()
	if customerID == 0 || templateID == 0 || len(record) == 0 {
		return
	}
	model := crmmodel.NewStatFieldValueModel()
	for fieldIDText, value := range record {
		fieldID := inputUint64(fieldIDText)
		if fieldID == 0 {
			continue
		}
		field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
			"id":               fieldID,
			"data_template_id": templateID,
			"status":           crmmodel.StatusEnabled,
		})
		if field == nil || field.FieldType == "group" || strings.TrimSpace(field.FieldKey) == "" {
			continue
		}
		usageField := workDataUsageFieldByType(ctx, field.ID, crmmodel.DataUsageTypeStat)
		if usageField == nil {
			continue
		}
		data := workStatFieldValueRecord(
			customerID,
			assetID,
			businessObjectID,
			templateID,
			taskID,
			operationID,
			field,
			usageField,
			workDataUsageByID(ctx, usageField.UsageID),
			value,
			changedAt,
		)
		existing := model.Find(ctx, map[string]any{
			"customer_id":        customerID,
			"asset_id":           assetID,
			"business_object_id": businessObjectID,
			"data_field_id":      field.ID,
		})
		if existing != nil {
			model.Update(ctx, map[string]any{"id": existing.ID}, data)
			continue
		}
		data["created_at"] = changedAt
		model.Insert(ctx, data)
	}
}

func workStatFieldValueRecord(
	customerID uint64,
	assetID uint64,
	businessObjectID uint64,
	templateID uint64,
	taskID uint64,
	operationID uint64,
	field *crmmodel.DataField,
	usageField *crmmodel.DataUsageField,
	usage *crmmodel.DataUsage,
	value any,
	changedAt time.Time,
) map[string]any {
	valueText := inputText(value)
	if emptyWorkFieldValue(value) {
		valueText = ""
	}
	valueType := normalizeWorkStatType(usageField.ValueType)
	displayName := field.Name
	if strings.TrimSpace(usageField.DisplayName) != "" {
		displayName = strings.TrimSpace(usageField.DisplayName)
	}
	statGroup := ""
	if usage != nil {
		statGroup = usage.Name
	}
	return map[string]any{
		"customer_id":        customerID,
		"asset_id":           assetID,
		"business_object_id": businessObjectID,
		"data_template_id":   templateID,
		"data_field_id":      field.ID,
		"field_key":          field.FieldKey,
		"field_name":         displayName,
		"field_type":         field.FieldType,
		"stat_type":          valueType,
		"stat_group":         statGroup,
		"value_text":         valueText,
		"value_number":       workStatNumberValue(field, valueType, value),
		"value_date":         workStatDateValue(field, valueType, value),
		"value_bool":         booleanFromAny(value),
		"value_json":         workStatJSONValue(value),
		"source":             crmmodel.StatValueSourceForm,
		"task_id":            taskID,
		"operation_log_id":   operationID,
		"status":             crmmodel.StatusEnabled,
		"updated_at":         changedAt,
	}
}

func normalizeWorkStatType(statType string) string {
	switch strings.TrimSpace(statType) {
	case crmmodel.DataFieldStatTypeMetric,
		crmmodel.DataUsageValueTypeNumber,
		crmmodel.DataUsageValueTypeAmount,
		crmmodel.DataFieldStatTypeFinance,
		crmmodel.DataUsageValueTypeTime,
		crmmodel.DataUsageValueTypeStatus,
		crmmodel.DataUsageValueTypeText:
		return strings.TrimSpace(statType)
	default:
		return crmmodel.DataFieldStatTypeDimension
	}
}

func workStatNumberValue(field *crmmodel.DataField, valueType string, value any) float64 {
	if field == nil {
		return 0
	}
	switch normalizeWorkStatType(valueType) {
	case crmmodel.DataFieldStatTypeMetric, crmmodel.DataUsageValueTypeNumber, crmmodel.DataUsageValueTypeAmount, crmmodel.DataFieldStatTypeFinance:
		return numericValue(value)
	}
	if field.FieldType == "number" || field.FieldType == "money" {
		return numericValue(value)
	}
	return 0
}

func workStatDateValue(field *crmmodel.DataField, valueType string, value any) time.Time {
	if field == nil || normalizeWorkStatType(valueType) != crmmodel.DataFieldStatTypeTime && field.FieldType != "date" && field.FieldType != "datetime" {
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

func workStatJSONValue(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(raw)
}
