package service

import (
	"context"
	"sort"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

type workProcessedOperation struct {
	CustomerID uint64
	AssetID    uint64
	TaskName   string
	Result     string
	Content    string
	CreatedAt  time.Time
}

var ignoredProcessedResults = map[string]bool{
	workResultProgress: true,
	"assigned":         true,
	"reassigned":       true,
	"unassigned":       true,
}

func processedWorkCustomerListTargets(ctx context.Context, staff *WorkStaffSession) []workCustomerListTarget {
	operations := processedWorkOperations(ctx, staff, 0)
	targets := make([]workCustomerListTarget, 0)
	indexes := map[uint64]int{}
	for _, operation := range operations {
		if operation.CustomerID == 0 {
			continue
		}
		index, exists := indexes[operation.CustomerID]
		if !exists {
			index = len(targets)
			indexes[operation.CustomerID] = index
			targets = append(targets, workCustomerListTarget{
				customerID: operation.CustomerID,
				processed:  map[uint64]workProcessedOperation{},
			})
		}
		target := &targets[index]
		if target.latestProcessed == nil {
			latest := operation
			target.latestProcessed = &latest
		}
		if _, exists := target.processed[operation.AssetID]; exists {
			continue
		}
		target.processed[operation.AssetID] = operation
		if operation.AssetID > 0 {
			target.assetIDs = append(target.assetIDs, operation.AssetID)
		}
	}
	return targets
}

func processedWorkCustomers(ctx context.Context, staff *WorkStaffSession) []map[string]any {
	targets := processedWorkCustomerListTargets(ctx, staff)
	return newWorkCustomerListRowBuilder(ctx, staff).rows(workCustomerModeProcessed, targets)
}

func processedWorkOperations(ctx context.Context, staff *WorkStaffSession, workflowID uint64) []workProcessedOperation {
	if staff == nil || staff.ID == 0 {
		return []workProcessedOperation{}
	}
	filters := map[string]any{"operator_staff_id": staff.ID}
	if workflowID > 0 {
		filters["workflow_id"] = workflowID
	}
	rows := crmmodel.NewOperationLogModel().Select(ctx, filters, map[string]any{"order": "created_at desc,id desc"})
	result := make([]workProcessedOperation, 0, len(rows))
	for _, operation := range rows {
		if !isHandledWorkOperation(operation) {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": operation.TaskID})
		if task == nil || task.TaskType == crmmodel.TaskTypeRule {
			continue
		}
		result = append(result, workProcessedOperation{
			CustomerID: operation.CustomerID,
			AssetID:    operation.AssetID,
			TaskName:   task.Name,
			Result:     operation.ResultValue,
			Content:    operation.Content,
			CreatedAt:  operation.CreatedAt,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

func isHandledWorkOperation(operation *crmmodel.OperationLog) bool {
	if operation == nil || operation.CustomerID == 0 || operation.TaskID == 0 || operation.OperatorStaffID == 0 {
		return false
	}
	return !ignoredProcessedResults[strings.ToLower(strings.TrimSpace(operation.ResultValue))]
}

func (builder *workCustomerListRowBuilder) processedCustomerRow(target workCustomerListTarget) map[string]any {
	customer := builder.customerBaseRow(target.customerID)
	if len(customer) == 0 {
		return map[string]any{}
	}
	assets := make([]map[string]any, 0, len(target.assetIDs))
	for _, assetID := range uniqueUint64Values(target.assetIDs) {
		if !canViewWorkAsset(builder.ctx, builder.staff, target.customerID, assetID) {
			continue
		}
		asset := builder.doneAssetRow(target.customerID, assetID)
		if len(asset) == 0 {
			continue
		}
		delete(asset, "row_tasks")
		if operation, exists := target.processed[assetID]; exists {
			attachProcessedOperation(asset, operation)
		}
		assets = append(assets, asset)
	}
	customer["assets"] = assets
	if target.latestProcessed != nil {
		attachProcessedOperation(customer, *target.latestProcessed)
	}
	return workCustomerListRow(customer)
}

func attachProcessedOperation(target map[string]any, operation workProcessedOperation) {
	if target == nil {
		return
	}
	target["processed_task_name"] = operation.TaskName
	target["processed_result"] = operation.Result
	target["processed_content"] = operation.Content
	target["processed_at"] = operation.CreatedAt
}
