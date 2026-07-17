package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

type workflowCustomerTarget struct {
	instance *crmmodel.WorkflowInstance
	updated  time.Time
}

func workflowCustomerList(
	ctx context.Context,
	staff *WorkStaffSession,
	workflowID uint64,
	payload map[string]any,
) (map[string]any, error) {
	workflow := workflowForSubject(ctx, workflowID, crmmodel.WorkflowSubjectCustomerAsset)
	if workflow == nil || !canAccessWorkflow(ctx, staff, workflow) {
		return nil, fmt.Errorf("流程不存在或无权查看")
	}
	scope := normalizeWorkScope(staff, firstText(payload, "scope"))
	staff = workStaffWithScope(staff, scope)
	mode := normalizeWorkCustomerMode(firstText(payload, "mode"))
	quickFilter := firstText(payload, "quick_filter", "quickFilter")
	instances := crmmodel.NewWorkflowInstanceModel().Select(ctx, map[string]any{
		"workflow_id": workflow.ID,
	}, map[string]any{"order": "updated_at desc,id desc"})
	latestTargetsByAsset := map[string]workflowCustomerTarget{}
	modeCounts := map[string]map[string]bool{
		workCustomerModeAll:     {},
		workCustomerModePending: {},
		workCustomerModeDone:    {},
	}
	for _, instance := range instances {
		if instance == nil || instance.CustomerID == 0 || instance.AssetID == 0 ||
			!canViewWorkflowInstanceInScope(ctx, staff, instance) {
			continue
		}
		assetKey := fmt.Sprintf("%d:%d", instance.CustomerID, instance.AssetID)
		if _, exists := latestTargetsByAsset[assetKey]; exists {
			continue
		}
		latestTargetsByAsset[assetKey] = workflowCustomerTarget{instance: instance, updated: instance.UpdatedAt}
	}

	targets := make([]workflowCustomerTarget, 0, len(latestTargetsByAsset))
	for assetKey, target := range latestTargetsByAsset {
		modeCounts[workCustomerModeAll][assetKey] = true
		if target.instance.Status == crmmodel.ProgressStatusActive {
			modeCounts[workCustomerModePending][assetKey] = true
		} else {
			modeCounts[workCustomerModeDone][assetKey] = true
		}
		if !workflowInstanceMatchesMode(target.instance, mode) {
			continue
		}
		targets = append(targets, target)
	}
	if isWorkPersonalQuickFilter(quickFilter) {
		targets = workflowCustomerPersonalQuickFilterTargets(ctx, staff, instances, mode, quickFilter)
	}
	sort.SliceStable(targets, func(i, j int) bool {
		leftPending := targets[i].instance.Status == crmmodel.ProgressStatusActive
		rightPending := targets[j].instance.Status == crmmodel.ProgressStatusActive
		if leftPending != rightPending {
			return leftPending
		}
		if targets[i].updated.Equal(targets[j].updated) {
			return targets[i].instance.ID > targets[j].instance.ID
		}
		return targets[i].updated.After(targets[j].updated)
	})
	rows := workflowCustomerRows(ctx, staff, targets)
	if hasWorkCustomerStructuredFilter(payload) {
		rows = filterWorkCustomersByFields(rows, payload)
	}
	if keyword := firstText(payload, "keyword"); keyword != "" {
		rows = filterWorkCustomers(rows, keyword)
	}
	if hasWorkCustomerWorkFilter(payload) {
		rows = filterWorkCustomersByWorkFilters(rows, payload)
	}
	list, page, pageSize, total := paginateWorkCustomerRows(rows, payload)
	return map[string]any{
		"list":      workCustomerListRows(list),
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"mode_counts": map[string]int{
			workCustomerModeAll:     len(modeCounts[workCustomerModeAll]),
			workCustomerModePending: len(modeCounts[workCustomerModePending]),
			workCustomerModeDone:    len(modeCounts[workCustomerModeDone]),
		},
		"stage_options": workStageOptions(ctx, workflow.ID),
		"scope":         scope,
		"can_dispatch":  staff.CanDispatch,
		"workflow": map[string]any{
			"id":           workflow.ID,
			"name":         workflow.Name,
			"subject_type": workflow.SubjectType,
		},
	}, nil
}

func workflowCustomerPersonalQuickFilterTargets(
	ctx context.Context,
	staff *WorkStaffSession,
	instances []*crmmodel.WorkflowInstance,
	mode string,
	quickFilter string,
) []workflowCustomerTarget {
	targets := make([]workflowCustomerTarget, 0)
	seenAssets := map[string]bool{}
	for _, instance := range instances {
		if instance == nil || instance.CustomerID == 0 || instance.AssetID == 0 ||
			!workflowInstanceMatchesMode(instance, mode) ||
			!workflowInstanceMatchesPersonalQuickFilter(ctx, staff, instance, quickFilter) {
			continue
		}
		if !canViewWorkflowInstanceInScope(ctx, staff, instance) && quickFilter != "completedToday" {
			continue
		}
		assetKey := fmt.Sprintf("%d:%d", instance.CustomerID, instance.AssetID)
		if seenAssets[assetKey] {
			continue
		}
		seenAssets[assetKey] = true
		targets = append(targets, workflowCustomerTarget{instance: instance, updated: instance.UpdatedAt})
	}
	return targets
}

func isWorkPersonalQuickFilter(value string) bool {
	return value == "personalPending" || value == "overdue" || value == "completedToday"
}

func workflowInstanceMatchesPersonalQuickFilter(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance, quickFilter string) bool {
	if !isWorkPersonalQuickFilter(quickFilter) {
		return true
	}
	if staff == nil || staff.ID == 0 || instance == nil {
		return false
	}
	status := crmmodel.WorkTodoStatusPending
	if quickFilter == "completedToday" {
		status = crmmodel.WorkTodoStatusDone
	}
	now := time.Now()
	today := workBeginningOfDay(now)
	tomorrow := today.AddDate(0, 0, 1)
	for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"assignee_staff_id":    staff.ID,
		"status":               status,
	}) {
		if todo == nil {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
		if task == nil || task.TaskType == crmmodel.TaskTypeRule {
			continue
		}
		if quickFilter == "personalPending" {
			return true
		}
		if quickFilter == "overdue" && todo.DueAt != nil && todo.DueAt.Before(now) {
			return true
		}
		if quickFilter == "completedToday" && todo.CompletedAt != nil {
			if !todo.CompletedAt.Before(today) && todo.CompletedAt.Before(tomorrow) {
				return true
			}
		}
	}
	return false
}

func workflowInstanceMatchesSummaryFilters(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance, stageFilter string, taskFilter string) bool {
	if staff == nil || staff.ID == 0 || instance == nil {
		return false
	}
	if stageFilter != "" && fmt.Sprintf("%d", instance.StageID) != stageFilter {
		return false
	}
	if taskFilter == "" {
		return true
	}
	for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"assignee_staff_id":    staff.ID,
		"status":               crmmodel.WorkTodoStatusPending,
	}) {
		if todo == nil {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
		if task != nil && task.TaskType != crmmodel.TaskTypeRule && task.TaskType == taskFilter {
			return true
		}
	}
	return false
}

func workflowInstanceMatchesMode(instance *crmmodel.WorkflowInstance, mode string) bool {
	if instance == nil {
		return false
	}
	switch normalizeWorkCustomerMode(mode) {
	case workCustomerModeAll:
		return true
	case workCustomerModeDone:
		return instance.Status != crmmodel.ProgressStatusActive
	default:
		return instance.Status == crmmodel.ProgressStatusActive
	}
}

func workflowCustomerRows(ctx context.Context, staff *WorkStaffSession, targets []workflowCustomerTarget) []map[string]any {
	builder := newWorkCustomerListRowBuilder(ctx, staff)
	customerRows := map[uint64]map[string]any{}
	customerOrder := make([]uint64, 0)
	for _, target := range targets {
		instance := target.instance
		if instance == nil {
			continue
		}
		customer := customerRows[instance.CustomerID]
		if customer == nil {
			customer = builder.customerBaseRow(instance.CustomerID)
			if len(customer) == 0 {
				continue
			}
			customer["assets"] = []map[string]any{}
			customerRows[instance.CustomerID] = customer
			customerOrder = append(customerOrder, instance.CustomerID)
		}
		asset := crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{
			"id":          instance.AssetID,
			"customer_id": instance.CustomerID,
		}, map[string]any{"field": "id,customer_id,asset_no,asset_name,asset_status_id,remark,created_at"})
		if len(asset) == 0 {
			continue
		}
		asset["asset_status_name"] = builder.assetStatusName(inputUint64(asset["asset_status_id"]))
		asset["customer_products"] = workCustomerProductRows(ctx, staff, instance.CustomerID, instance.AssetID)
		builder.attachStageFields(asset, instance)
		asset["flow"] = workFlowDetail(ctx, staff, instance.ID)
		asset["row_tasks"] = workflowInstanceTodoRows(ctx, staff, instance)
		customer["assets"] = append(mapListFromAny(customer["assets"]), asset)
	}
	rows := make([]map[string]any, 0, len(customerOrder))
	for _, customerID := range customerOrder {
		if row := customerRows[customerID]; len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func workflowInstanceTodoRows(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) []map[string]any {
	if instance == nil {
		return nil
	}
	todos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"status":               crmmodel.WorkTodoStatusPending,
	})
	rows := make([]map[string]any, 0, len(todos))
	for _, todo := range todos {
		if todo == nil || instance.OwnerStaffID != staff.ID && !canOperateWorkTodo(staff, todo) && !(staff.CanDispatch && staff.ViewAll) {
			continue
		}
		if row := workTodoTaskMap(ctx, staff, todo, false); len(row) > 0 {
			rows = append(rows, row)
		}
	}
	sortWorkTodoTaskMaps(rows)
	return rows
}
