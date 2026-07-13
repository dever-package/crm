package service

import (
	"context"
	"log"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func syncWorkFinanceLedgers(ctx context.Context, staff *WorkStaffSession, completion workOperationCompletion) {
	defer recoverWorkSideEffect("finance_ledger", completion)
	if staff == nil || staff.ID == 0 || completion.task == nil || completion.formInput == nil || completion.operationID == 0 {
		return
	}
	changedAt := workOperationCreatedAt(ctx, completion.operationID)
	syncWorkFinanceLedgerRecords(ctx, staff, completion, completion.formInput.customerDataRecords, 0, changedAt)
	syncWorkFinanceLedgerRecords(ctx, staff, completion, completion.formInput.assetDataRecords, completion.ownership.AssetID, changedAt)
	syncWorkFinanceLedgerRecords(ctx, staff, completion, completion.formInput.businessDataRecords, completion.ownership.AssetID, changedAt)
}

func syncWorkFinanceLedgerRecords(ctx context.Context, staff *WorkStaffSession, completion workOperationCompletion, records map[uint64]map[string]any, assetID uint64, changedAt time.Time) {
	if completion.ownership.CustomerID == 0 || len(records) == 0 {
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
			field, usageField := workFinanceDataField(ctx, templateID, inputUint64(fieldIDText))
			if field == nil || usageField == nil {
				continue
			}
			existing := model.Find(ctx, map[string]any{
				"workflow_instance_id": completion.ownership.WorkflowInstanceID,
				"operation_log_id":     completion.operationID,
				"data_field_id":        field.ID,
				"source":               crmmodel.FinanceLedgerSourceForm,
			})
			if existing != nil {
				continue
			}
			financeType := workFinanceType(ctx, usageField.FinanceTypeID)
			if financeType == nil {
				continue
			}
			data := workFinanceLedgerRecord(completion, staff, assetID, field, financeType, value, changedAt)
			model.Insert(ctx, data)
		}
	}
}

func workFinanceDataField(ctx context.Context, templateID uint64, fieldID uint64) (*crmmodel.DataField, *crmmodel.DataUsageField) {
	if templateID == 0 || fieldID == 0 {
		return nil, nil
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":               fieldID,
		"data_template_id": templateID,
		"status":           crmmodel.StatusEnabled,
	})
	if field == nil || field.FieldType == "group" {
		return nil, nil
	}
	usageField := workDataUsageFieldByType(ctx, field.ID, crmmodel.DataUsageTypeFinance)
	if usageField == nil || usageField.FinanceTypeID == 0 {
		return nil, nil
	}
	return field, usageField
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
		"customer_id":          completion.ownership.CustomerID,
		"asset_id":             assetID,
		"workflow_instance_id": completion.ownership.WorkflowInstanceID,
		"customer_product_id":  completion.ownership.CustomerProductID,
		"task_id":              completion.task.ID,
		"operation_log_id":     completion.operationID,
		"data_field_id":        field.ID,
		"finance_type_id":      financeType.ID,
		"finance_type_code":    financeType.Code,
		"finance_type_name":    financeType.Name,
		"direction":            financeType.Direction,
		"amount":               numericValue(value),
		"raw_value":            inputText(value),
		"staff_id":             staff.ID,
		"department_id":        staff.DepartmentID,
		"source":               crmmodel.FinanceLedgerSourceForm,
		"created_at":           changedAt,
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
			completion.ownership.CustomerID,
			completion.ownership.AssetID,
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
