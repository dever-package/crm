package service

import (
	"context"
	"fmt"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

const workFinanceDataFieldPrefix = "field:"

func syncWorkDataFieldFinanceLedgers(
	ctx context.Context,
	staff *WorkStaffSession,
	ownership workDataOwnership,
	taskID uint64,
	operationID uint64,
	formInput *workFormInput,
) error {
	if operationID == 0 || formInput == nil {
		return nil
	}
	fields := workSubmittedDataFields(ctx, formInput)
	financeTypes := map[uint64]*crmmodel.FinanceType{}
	changedAt := workOperationCreatedAt(ctx, operationID)
	staffID, departmentID := workFinanceOperator(ctx, staff, operationID)
	model := crmmodel.NewFinanceLedgerModel()

	for _, field := range fields {
		if field == nil || field.FinanceTypeID == 0 || field.FieldType == "group" || field.FieldType == "attachment" {
			continue
		}
		value, valueOwnership, submitted := workSubmittedDataFieldValue(formInput, ownership, field)
		if !submitted || emptyWorkFieldValue(value) {
			continue
		}
		if valueOwnership.CustomerID == 0 {
			return fmt.Errorf("%s财务字段缺少客户", field.Name)
		}
		financeType, loaded := financeTypes[field.FinanceTypeID]
		if !loaded {
			financeType = crmmodel.NewFinanceTypeModel().Find(ctx, map[string]any{
				"id":     field.FinanceTypeID,
				"status": crmmodel.StatusEnabled,
			})
			financeTypes[field.FinanceTypeID] = financeType
		}
		if financeType == nil {
			return fmt.Errorf("%s配置的财务类型不存在或已停用", field.Name)
		}
		amount := numericValue(value)
		if amount <= 0 {
			return fmt.Errorf("%s必须大于 0", field.Name)
		}
		sourceKey := workFinanceDataFieldKey(field.ID)
		if model.Find(ctx, map[string]any{
			"workflow_instance_id": valueOwnership.WorkflowInstanceID,
			"operation_log_id":     operationID,
			"finance_source_key":   sourceKey,
			"source":               crmmodel.FinanceLedgerSourceForm,
		}) != nil {
			continue
		}
		ledgerID := uint64(model.Insert(ctx, map[string]any{
			"customer_id":          valueOwnership.CustomerID,
			"asset_id":             valueOwnership.AssetID,
			"workflow_instance_id": valueOwnership.WorkflowInstanceID,
			"customer_product_id":  valueOwnership.CustomerProductID,
			"task_id":              taskID,
			"operation_log_id":     operationID,
			"data_field_id":        field.ID,
			"finance_source_key":   sourceKey,
			"finance_type_id":      financeType.ID,
			"finance_type_code":    financeType.Code,
			"finance_type_name":    financeType.Name,
			"direction":            financeType.Direction,
			"amount":               amount,
			"raw_value":            inputText(value),
			"staff_id":             staffID,
			"department_id":        departmentID,
			"source":               crmmodel.FinanceLedgerSourceForm,
			"created_at":           changedAt,
		}))
		if ledgerID == 0 {
			return fmt.Errorf("%s财务流水保存失败", field.Name)
		}
	}
	return nil
}

func workFinanceOperator(ctx context.Context, staff *WorkStaffSession, operationID uint64) (uint64, uint64) {
	if staff != nil && staff.ID > 0 {
		return staff.ID, staff.DepartmentID
	}
	if operation := crmmodel.NewOperationLogModel().Find(ctx, map[string]any{"id": operationID}); operation != nil {
		return operation.OperatorStaffID, operation.OperatorDepartmentID
	}
	return 0, 0
}

func workFinanceDataFieldKey(dataFieldID uint64) string {
	return fmt.Sprintf("%s%d", workFinanceDataFieldPrefix, dataFieldID)
}

func workOperationCreatedAt(ctx context.Context, operationID uint64) time.Time {
	if operationID > 0 {
		if operation := crmmodel.NewOperationLogModel().Find(ctx, map[string]any{"id": operationID}); operation != nil && !operation.CreatedAt.IsZero() {
			return operation.CreatedAt
		}
	}
	return time.Now()
}
