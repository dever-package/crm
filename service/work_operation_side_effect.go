package service

import (
	"context"
	"log"
	"time"

	crmmodel "my/package/crm/model"
)

func syncWorkTaskPointLedger(ctx context.Context, staff *WorkStaffSession, completion workOperationCompletion) {
	defer recoverWorkSideEffect("task_point_ledger", completion)
	if staff == nil || staff.ID == 0 || completion.task == nil || completion.operationID == 0 {
		return
	}
	points := workCompletionTaskPoints(ctx, completion)
	if points <= 0 {
		return
	}
	model := crmmodel.NewTaskPointLedgerModel()
	source := crmmodel.TaskPointLedgerSourceTaskComplete
	if existing := model.Find(ctx, map[string]any{
		"operation_log_id": completion.operationID,
		"todo_id":          completion.todoID,
		"source":           source,
	}); existing != nil {
		return
	}
	data := map[string]any{
		"customer_id":      completion.customerID,
		"asset_id":         completion.assetID,
		"task_id":          completion.task.ID,
		"operation_log_id": completion.operationID,
		"todo_id":          completion.todoID,
		"points":           points,
		"staff_id":         staff.ID,
		"department_id":    staff.DepartmentID,
		"result_value":     completion.resultValue,
		"source":           source,
		"created_at":       workOperationCreatedAt(ctx, completion.operationID),
	}
	model.Insert(ctx, data)
}

func workCompletionTaskPoints(ctx context.Context, completion workOperationCompletion) float64 {
	if completion.todoID > 0 {
		if todo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{"id": completion.todoID}); todo != nil {
			return positiveWorkPoints(todo.TaskPoints)
		}
		return 0
	}
	return workTaskPoints(completion.task)
}

func workTaskPoints(task *crmmodel.Task) float64 {
	if task == nil {
		return 0
	}
	return positiveWorkPoints(numericValue(mapFromAny(task.ConfigJSON)["task_points"]))
}

func positiveWorkPoints(points float64) float64 {
	if points > 0 {
		return points
	}
	return 0
}

func syncWorkFinanceLedgers(ctx context.Context, staff *WorkStaffSession, completion workOperationCompletion) {
	defer recoverWorkSideEffect("finance_ledger", completion)
	if staff == nil || staff.ID == 0 || completion.task == nil || completion.formInput == nil || completion.operationID == 0 {
		return
	}
	changedAt := workOperationCreatedAt(ctx, completion.operationID)
	syncWorkFinanceLedgerRecords(ctx, staff, completion, completion.formInput.customerDataRecords, 0, changedAt)
	syncWorkFinanceLedgerRecords(ctx, staff, completion, completion.formInput.assetDataRecords, completion.assetID, changedAt)
}

func syncWorkFinanceLedgerRecords(ctx context.Context, staff *WorkStaffSession, completion workOperationCompletion, records map[uint64]map[string]any, assetID uint64, changedAt time.Time) {
	if completion.customerID == 0 || len(records) == 0 {
		return
	}
	model := crmmodel.NewFinanceLedgerModel()
	for templateID, record := range records {
		if templateID == 0 || len(record) == 0 {
			continue
		}
		for fieldIDText, value := range record {
			if emptyWorkFieldValue(value) {
				continue
			}
			field := workFinanceDataField(ctx, templateID, inputUint64(fieldIDText))
			if field == nil {
				continue
			}
			existing := model.Find(ctx, map[string]any{
				"operation_log_id": completion.operationID,
				"data_field_id":    field.ID,
				"source":           crmmodel.FinanceLedgerSourceForm,
			})
			if existing != nil {
				continue
			}
			financeType := workFinanceType(ctx, field.StatID)
			if financeType == nil {
				continue
			}
			data := workFinanceLedgerRecord(completion, staff, assetID, field, financeType, value, changedAt)
			model.Insert(ctx, data)
		}
	}
}

func workFinanceDataField(ctx context.Context, templateID uint64, fieldID uint64) *crmmodel.DataField {
	if templateID == 0 || fieldID == 0 {
		return nil
	}
	return crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":               fieldID,
		"data_template_id": templateID,
		"stat_enabled":     true,
		"stat_type":        crmmodel.DataFieldStatTypeFinance,
		"status":           crmmodel.StatusEnabled,
	})
}

func workFinanceType(ctx context.Context, financeTypeID uint64) *crmmodel.FinanceType {
	if financeTypeID == 0 {
		return nil
	}
	return crmmodel.NewFinanceTypeModel().Find(ctx, map[string]any{
		"id":     financeTypeID,
		"status": crmmodel.StatusEnabled,
	})
}

func workFinanceLedgerRecord(completion workOperationCompletion, staff *WorkStaffSession, assetID uint64, field *crmmodel.DataField, financeType *crmmodel.FinanceType, value any, changedAt time.Time) map[string]any {
	return map[string]any{
		"customer_id":       completion.customerID,
		"asset_id":          assetID,
		"task_id":           completion.task.ID,
		"operation_log_id":  completion.operationID,
		"data_field_id":     field.ID,
		"finance_type_id":   financeType.ID,
		"finance_type_code": financeType.Code,
		"finance_type_name": financeType.Name,
		"direction":         financeType.Direction,
		"amount":            numericValue(value),
		"raw_value":         inputText(value),
		"staff_id":          staff.ID,
		"department_id":     staff.DepartmentID,
		"source":            crmmodel.FinanceLedgerSourceForm,
		"created_at":        changedAt,
	}
}

func workOperationCreatedAt(ctx context.Context, operationID uint64) time.Time {
	if operationID > 0 {
		if operation := crmmodel.NewOperationLogModel().Find(ctx, map[string]any{"id": operationID}); operation != nil && !operation.CreatedAt.IsZero() {
			return operation.CreatedAt
		}
	}
	return time.Now()
}

func recoverWorkSideEffect(name string, completion workOperationCompletion) {
	if recovered := recover(); recovered != nil {
		log.Printf(
			"crm work side effect %s failed: customer_id=%d asset_id=%d task_id=%d operation_log_id=%d todo_id=%d error=%v",
			name,
			completion.customerID,
			completion.assetID,
			workCompletionTaskID(completion),
			completion.operationID,
			completion.todoID,
			recovered,
		)
	}
}

func workCompletionTaskID(completion workOperationCompletion) uint64 {
	if completion.task == nil {
		return 0
	}
	return completion.task.ID
}
