package service

import (
	"context"
	"sort"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func queryWorkTodoTasks(ctx context.Context, staff *WorkStaffSession, customerID, assetID uint64, withForm bool) []map[string]any {
	filters := map[string]any{"status": crmmodel.WorkTodoStatusPending}
	if customerID > 0 {
		filters["customer_id"] = customerID
	}
	if assetID > 0 {
		filters["asset_id"] = assetID
	} else if customerID > 0 {
		filters["asset_id"] = uint64(0)
	}
	rows := crmmodel.NewWorkTodoModel().Select(ctx, filters)
	result := make([]map[string]any, 0, len(rows))
	for _, todo := range rows {
		if todo == nil || !canOperateWorkTodo(staff, todo) {
			continue
		}
		row := workTodoTaskMap(ctx, staff, todo, withForm)
		if len(row) > 0 {
			result = append(result, row)
		}
	}
	sortWorkTodoTaskMaps(result)
	return result
}

func queryWorkTodoRows(ctx context.Context, staff *WorkStaffSession, customerID, assetID uint64, pendingOnly bool) []map[string]any {
	filters := map[string]any{}
	if customerID > 0 {
		filters["customer_id"] = customerID
	}
	if assetID > 0 {
		filters["asset_id"] = assetID
	}
	if pendingOnly {
		filters["status"] = crmmodel.WorkTodoStatusPending
	}
	rows := crmmodel.NewWorkTodoModel().Select(ctx, filters)
	result := make([]map[string]any, 0, len(rows))
	for _, todo := range rows {
		if todo == nil {
			continue
		}
		if pendingOnly && !canOperateWorkTodo(staff, todo) {
			continue
		}
		row := workTodoTaskMap(ctx, staff, todo, false)
		if len(row) > 0 {
			result = append(result, row)
		}
	}
	sortWorkTodoTaskMaps(result)
	return result
}

func workTodoTaskMap(ctx context.Context, staff *WorkStaffSession, todo *crmmodel.WorkTodo, withForm bool) map[string]any {
	if todo == nil || todo.TaskID == 0 {
		return map[string]any{}
	}
	task := crmmodel.NewTaskModel().FindMap(ctx, map[string]any{"id": todo.TaskID})
	if len(task) == 0 {
		return map[string]any{}
	}
	workflowName := ""
	if workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": todo.WorkflowID}); workflow != nil {
		workflowName = workflow.Name
	}
	stageName := ""
	if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": todo.StageID}); stage != nil {
		stageName = stage.Name
	}
	departmentName := ""
	if department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": todo.AssigneeDepartmentID}); department != nil {
		departmentName = department.Name
	}
	staffName := ""
	if assignee := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": todo.AssigneeStaffID}); assignee != nil {
		staffName = assignee.Name
	}
	task["id"] = todo.TaskID
	task["task_id"] = todo.TaskID
	task["task_name"] = inputText(task["name"])
	task["todo_id"] = todo.ID
	task["lead_id"] = todo.LeadID
	task["workflow_instance_id"] = todo.WorkflowInstanceID
	task["customer_product_id"] = todo.CustomerProductID
	task["workflow_id"] = todo.WorkflowID
	task["workflow_name"] = workflowName
	task["stage_id"] = todo.StageID
	task["stage_name"] = stageName
	task["customer_id"] = todo.CustomerID
	task["asset_id"] = todo.AssetID
	task["required"] = todo.Required
	task["todo_required"] = todo.Required
	task["status"] = todo.Status
	task["todo_status"] = todo.Status
	task["status_name"] = workTodoStatusName(todo.Status)
	task["due_at"] = todo.DueAt
	task["result"] = todo.Result
	task["assigned_at"] = todo.CreatedAt
	task["todo_sort"] = inputInt(task["sort"])
	task["assignee_department_id"] = todo.AssigneeDepartmentID
	task["assignee_department_name"] = departmentName
	task["assignee_staff_id"] = todo.AssigneeStaffID
	task["assignee_staff_name"] = staffName
	task["can_operate"] = todo.Status == crmmodel.WorkTodoStatusPending && canOperateWorkTodo(staff, todo)
	task["unassigned"] = todo.AssigneeStaffID == 0
	if withForm && inputText(task["task_type"]) == crmmodel.TaskTypeForm {
		attachWorkTaskForm(ctx, task)
	}
	if inputText(task["task_type"]) == crmmodel.TaskTypeProduct {
		task["product_options"] = workEnabledProductOptions(ctx)
		task["selected_product_ids"] = workSelectedProductIDs(ctx, todo.WorkflowInstanceID)
	}
	return task
}

func workEnabledProductOptions(ctx context.Context) []map[string]any {
	products := crmmodel.NewProductModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled}, map[string]any{"order": "sort asc,id asc"})
	result := make([]map[string]any, 0, len(products))
	for _, product := range products {
		if product == nil {
			continue
		}
		row := map[string]any{
			"id":                  product.ID,
			"name":                product.Name,
			"code":                product.Code,
			"category_id":         product.CategoryID,
			"service_workflow_id": product.ServiceWorkflowID,
		}
		if category := crmmodel.NewProductCategoryModel().Find(ctx, map[string]any{"id": product.CategoryID}); category != nil {
			row["category_name"] = category.Name
		}
		if workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": product.ServiceWorkflowID}); workflow != nil {
			row["service_workflow_name"] = workflow.Name
		}
		result = append(result, row)
	}
	return result
}

func workSelectedProductIDs(ctx context.Context, workflowInstanceID uint64) []uint64 {
	rows := crmmodel.NewCustomerProductModel().Select(ctx, map[string]any{"source_workflow_instance_id": workflowInstanceID})
	result := make([]uint64, 0, len(rows))
	for _, customerProduct := range rows {
		if customerProduct != nil && customerProduct.Status != crmmodel.CustomerProductStatusLost {
			result = append(result, customerProduct.ProductID)
		}
	}
	return result
}

func sortWorkTodoTaskMaps(rows []map[string]any) {
	now := time.Now()
	sort.SliceStable(rows, func(i, j int) bool {
		leftDue := workTimeValue(rows[i]["due_at"])
		rightDue := workTimeValue(rows[j]["due_at"])
		leftOverdue := !leftDue.IsZero() && leftDue.Before(now)
		rightOverdue := !rightDue.IsZero() && rightDue.Before(now)
		if leftOverdue != rightOverdue {
			return leftOverdue
		}
		if leftDue.IsZero() != rightDue.IsZero() {
			return !leftDue.IsZero()
		}
		if !leftDue.Equal(rightDue) {
			return leftDue.Before(rightDue)
		}
		leftSort := inputInt(rows[i]["todo_sort"])
		rightSort := inputInt(rows[j]["todo_sort"])
		if leftSort != rightSort {
			return leftSort < rightSort
		}
		return inputUint64(rows[i]["todo_id"]) < inputUint64(rows[j]["todo_id"])
	})
}

func enrichWorkBookingRow(_ context.Context, row map[string]any) {
	if row == nil {
		return
	}
	switch inputText(row["booking_status"]) {
	case crmmodel.ResourceBookingStatusPending:
		row["booking_status_name"] = "待确认"
	case crmmodel.ResourceBookingStatusReserved:
		row["booking_status_name"] = "已预定"
	case crmmodel.ResourceBookingStatusCanceled:
		row["booking_status_name"] = "已取消"
	case crmmodel.ResourceBookingStatusRejected:
		row["booking_status_name"] = "已拒绝"
	case crmmodel.ResourceBookingStatusDone:
		row["booking_status_name"] = "已完成"
	}
}
