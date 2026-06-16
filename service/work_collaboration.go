package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	crmmodel "my/package/crm/model"
)

type workCollaborationTodoTarget struct {
	Name           string
	DepartmentID   uint64
	StaffID        uint64
	FormID         uint64
	CompletionMode string
	TaskPoints     float64
	Required       bool
	Sort           int
}

func createWorkCollaborationTodos(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, operationID uint64, targets []workCollaborationTodoTarget) []uint64 {
	now := time.Now()
	assignedAt := now
	if operation := crmmodel.NewOperationLogModel().Find(ctx, map[string]any{"id": operationID}); operation != nil && !operation.CreatedAt.IsZero() {
		assignedAt = operation.CreatedAt
	}
	ids := make([]uint64, 0, len(targets))
	model := crmmodel.NewWorkTodoModel()
	for _, target := range targets {
		record := map[string]any{
			"customer_id":                customerID,
			"asset_id":                   assetID,
			"source_task_id":             task.ID,
			"parent_operation_log_id":    operationID,
			"sub_task_name":              target.Name,
			"form_id":                    target.FormID,
			"completion_mode":            normalizeWorkTaskCompletionMode(target.CompletionMode),
			"task_points":                target.TaskPoints,
			"assignee_department_id":     target.DepartmentID,
			"assignee_staff_id":          target.StaffID,
			"required":                   target.Required,
			"sort":                       target.Sort,
			"status":                     crmmodel.WorkTodoStatusPending,
			"assigned_at":                assignedAt,
			"completed_operation_log_id": uint64(0),
			"created_by_staff_id":        staff.ID,
			"created_at":                 now,
			"updated_at":                 now,
		}
		id := uint64(model.Insert(ctx, record))
		ids = append(ids, id)
		upsertWorkTodoMember(ctx, customerID, assetID, target.DepartmentID, target.StaffID)
	}
	return ids
}

func workCollaborationTargets(ctx context.Context, task *crmmodel.Task, values map[string]any) ([]workCollaborationTodoTarget, error) {
	configTargets := configuredWorkCollaborationTargets(task)
	rawTargets := configTargets
	if len(rawTargets) > 0 {
		rawTargets = mergeSubmittedWorkCollaborationStaff(rawTargets, mapsFromAny(firstWorkValue(values, "collaboration_targets", "collaborationTargets")))
	} else {
		rawTargets = mapsFromAny(firstWorkValue(values, "collaboration_targets", "collaborationTargets"))
	}
	result := make([]workCollaborationTodoTarget, 0, len(rawTargets))
	seen := map[string]bool{}
	for _, raw := range rawTargets {
		if workCollaborationTargetIsBlank(raw) {
			continue
		}
		target, err := normalizeWorkCollaborationTarget(ctx, raw)
		if err != nil {
			return nil, err
		}
		if target.DepartmentID == 0 {
			continue
		}
		key := fmt.Sprintf("%s:%d:%d", target.Name, target.DepartmentID, target.StaffID)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, target)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Sort != result[j].Sort {
			return result[i].Sort < result[j].Sort
		}
		if result[i].DepartmentID != result[j].DepartmentID {
			return result[i].DepartmentID < result[j].DepartmentID
		}
		return result[i].StaffID < result[j].StaffID
	})
	return result, nil
}

func configuredWorkCollaborationTargets(task *crmmodel.Task) []map[string]any {
	if task == nil {
		return nil
	}
	config := mapFromAny(task.ConfigJSON)
	return mapsFromAny(config["collaboration_items"])
}

func mergeSubmittedWorkCollaborationStaff(configTargets []map[string]any, submittedTargets []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(configTargets))
	submittedByKey := submittedWorkCollaborationTargetsByKey(submittedTargets)
	for index, configTarget := range configTargets {
		target := copyMap(configTarget)
		if inputUint64(firstWorkValue(configTarget, "staff_id", "assignee_staff_id")) == 0 {
			submittedTarget := submittedWorkCollaborationTarget(configTarget, submittedByKey, submittedTargets, index)
			submittedStaffID := inputUint64(firstWorkValue(submittedTarget, "staff_id", "assignee_staff_id"))
			if submittedStaffID > 0 {
				target["staff_id"] = submittedStaffID
			}
		}
		result = append(result, target)
	}
	return result
}

func submittedWorkCollaborationTargetsByKey(targets []map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any, len(targets))
	for _, target := range targets {
		key := inputText(firstWorkValue(target, "key", "target_key", "targetKey"))
		if key != "" {
			result[key] = target
		}
	}
	return result
}

func submittedWorkCollaborationTarget(configTarget map[string]any, submittedByKey map[string]map[string]any, submittedTargets []map[string]any, index int) map[string]any {
	key := inputText(firstWorkValue(configTarget, "key", "target_key", "targetKey"))
	if key != "" {
		if submittedTarget, ok := submittedByKey[key]; ok {
			return submittedTarget
		}
	}
	if index < len(submittedTargets) {
		return submittedTargets[index]
	}
	return nil
}

func workCollaborationTargetIsBlank(raw map[string]any) bool {
	return inputUint64(firstWorkValue(raw, "department_id", "assignee_department_id")) == 0 &&
		inputUint64(firstWorkValue(raw, "staff_id", "assignee_staff_id")) == 0
}

func normalizeWorkCollaborationTarget(ctx context.Context, raw map[string]any) (workCollaborationTodoTarget, error) {
	target := workCollaborationTodoTarget{
		Name:           inputText(firstWorkValue(raw, "name", "task_name", "sub_task_name")),
		DepartmentID:   inputUint64(firstWorkValue(raw, "department_id", "assignee_department_id")),
		StaffID:        inputUint64(firstWorkValue(raw, "staff_id", "assignee_staff_id")),
		FormID:         inputUint64(firstWorkValue(raw, "form_id")),
		CompletionMode: normalizeWorkTaskCompletionMode(inputText(firstWorkValue(raw, "completion_mode", "completionMode"))),
		TaskPoints:     positiveWorkPoints(numericValue(firstWorkValue(raw, "task_points", "points"))),
		Required:       true,
		Sort:           int(inputUint64(firstWorkValue(raw, "sort"))),
	}
	if value, exists := raw["required"]; exists {
		target.Required = booleanFromAny(value)
	}
	if target.Name == "" {
		target.Name = "协作子任务"
	}
	if target.DepartmentID == 0 {
		return target, fmt.Errorf("协作子任务目标部门不能为空")
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": target.DepartmentID, "status": crmmodel.StatusEnabled})
	if department == nil {
		return target, fmt.Errorf("协作子任务目标部门不存在或已停用")
	}
	if target.StaffID == 0 {
		return target, fmt.Errorf("协作子任务处理人员不能为空")
	}
	staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": target.StaffID, "status": crmmodel.StatusEnabled})
	if staff == nil {
		return target, fmt.Errorf("协作子任务处理人员不存在或已停用")
	}
	if staff.DepartmentID != target.DepartmentID {
		return target, fmt.Errorf("协作子任务处理人员不属于目标部门")
	}
	if target.FormID > 0 && crmmodel.NewFormModel().Find(ctx, map[string]any{"id": target.FormID, "status": crmmodel.StatusEnabled}) == nil {
		return target, fmt.Errorf("协作子任务资料模板不存在或已停用")
	}
	return target, nil
}

func firstWorkValue(row map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, exists := row[key]; exists {
			return value
		}
	}
	return nil
}

func completeWorkTodo(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, todoID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	todo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
		"id":     todoID,
		"status": crmmodel.WorkTodoStatusPending,
	})
	if todo == nil {
		return nil, fmt.Errorf("协作待办不存在或已完成")
	}
	if todo.CustomerID != customerID || todo.AssetID != assetID {
		return nil, fmt.Errorf("协作待办不属于当前客户资产")
	}
	if !canOperateWorkTodo(staff, todo) {
		return nil, fmt.Errorf("当前人员无权完成该协作待办")
	}
	progressSubmit := workSubmitIsProgress(values) && workCollaborationTodoAllowsProgress(todo)
	formInput := emptyWorkFormInput()
	if todo.FormID > 0 {
		var todoFormInput *workFormInput
		var err error
		if progressSubmit {
			todoFormInput, err = collectWorkProgressFormInputForForm(ctx, todo.FormID, values)
		} else {
			todoFormInput, err = collectWorkFormInputForForm(ctx, todo.FormID, values)
		}
		if err != nil {
			return nil, err
		}
		formInput = mergeWorkFormInput(formInput, todoFormInput)
	}
	if err := saveWorkFormInput(ctx, customerID, assetID, formInput); err != nil {
		return nil, err
	}
	logValues := copyMap(values)
	logValues["todo_id"] = todo.ID
	logValues["todo_name"] = todo.SubTaskName
	resultValue := workResultSuccess
	if progressSubmit {
		resultValue = workResultProgress
	}
	operationID := insertWorkOperationLogRecord(ctx, staff, task, customerID, assetID, currentWorkCustomerStage(ctx, customerID, assetID), logValues, resultValue, todo.SubTaskName)
	saveWorkFormDataRecords(ctx, customerID, assetID, task.ID, operationID, formInput)
	afterWorkOperationCompleted(ctx, staff, workOperationCompletion{
		customerID:   customerID,
		assetID:      assetID,
		operationID:  operationID,
		task:         task,
		formInput:    formInput,
		resultValue:  resultValue,
		todoID:       todo.ID,
		progressOnly: progressSubmit,
	})
	if progressSubmit {
		updateWorkCustomerStageOperation(ctx, customerID, assetID, operationID)
		return map[string]any{
			"customer_id":  customerID,
			"asset_id":     assetID,
			"todo_id":      todo.ID,
			"result_value": resultValue,
			"saved":        true,
			"progress":     true,
		}, nil
	}
	now := time.Now()
	crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{"id": todo.ID}, map[string]any{
		"status":                     crmmodel.WorkTodoStatusDone,
		"completed_at":               now,
		"completed_operation_log_id": operationID,
		"updated_at":                 now,
	})
	if workCollaborationShouldFlow(ctx, task, todo) {
		fromState := currentWorkCustomerStage(ctx, customerID, assetID)
		transitionStageCode := applyWorkStageTransitionWithOwner(ctx, staff, customerID, assetID, fromState, task, operationID, workResultSuccess, 0, 0)
		runWorkAutoTriggers(ctx, staff, customerID, assetID, task, workResultSuccess, transitionStageCode, runtime)
	}
	return map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"todo_id":     todo.ID,
		"saved":       true,
	}, nil
}

func canOperateWorkTodo(staff *WorkStaffSession, todo *crmmodel.WorkTodo) bool {
	if staff == nil || todo == nil {
		return false
	}
	if todo.AssigneeStaffID > 0 {
		return todo.AssigneeStaffID == staff.ID
	}
	return todo.AssigneeDepartmentID > 0 && todo.AssigneeDepartmentID == staff.DepartmentID
}

func workCollaborationShouldFlow(ctx context.Context, task *crmmodel.Task, todo *crmmodel.WorkTodo) bool {
	if task == nil || todo == nil || todo.ParentOperationLogID == 0 || workCollaborationAlreadyFlowed(ctx, task, todo.ParentOperationLogID) {
		return false
	}
	mode := normalizeWorkCollaborationCompleteMode(inputText(mapFromAny(task.ConfigJSON)["collaboration_complete_mode"]))
	switch mode {
	case crmmodel.CollaborationCompleteManual:
		return false
	case crmmodel.CollaborationCompleteAny:
		return workTodoCount(ctx, todo.ParentOperationLogID, map[string]any{"status": crmmodel.WorkTodoStatusDone}) == 1
	default:
		requiredTotal := workTodoCount(ctx, todo.ParentOperationLogID, map[string]any{"required": true})
		if requiredTotal == 0 {
			return workTodoCount(ctx, todo.ParentOperationLogID, map[string]any{"status": crmmodel.WorkTodoStatusDone}) == 1
		}
		if !todo.Required {
			return false
		}
		return workTodoCount(ctx, todo.ParentOperationLogID, map[string]any{
			"required": true,
			"status":   crmmodel.WorkTodoStatusPending,
		}) == 0
	}
}

func workTodoCount(ctx context.Context, parentOperationID uint64, filter map[string]any) int64 {
	query := map[string]any{"parent_operation_log_id": parentOperationID}
	for key, value := range filter {
		query[key] = value
	}
	return crmmodel.NewWorkTodoModel().Count(ctx, query)
}

func workCollaborationAlreadyFlowed(ctx context.Context, task *crmmodel.Task, parentOperationID uint64) bool {
	if task == nil || parentOperationID == 0 {
		return false
	}
	if crmmodel.NewStageTransitionLogModel().Find(ctx, map[string]any{
		"task_id":          task.ID,
		"operation_log_id": parentOperationID,
	}) != nil {
		return true
	}
	for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"parent_operation_log_id": parentOperationID,
	}) {
		if todo == nil || todo.CompletedOperationLogID == 0 {
			continue
		}
		if crmmodel.NewStageTransitionLogModel().Find(ctx, map[string]any{
			"task_id":          task.ID,
			"operation_log_id": todo.CompletedOperationLogID,
		}) != nil {
			return true
		}
	}
	return false
}

func normalizeWorkCollaborationCompleteMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case crmmodel.CollaborationCompleteAny:
		return crmmodel.CollaborationCompleteAny
	case crmmodel.CollaborationCompleteManual:
		return crmmodel.CollaborationCompleteManual
	default:
		return crmmodel.CollaborationCompleteAll
	}
}

func normalizeWorkTaskCompletionMode(mode string) string {
	if strings.TrimSpace(mode) == crmmodel.TaskCompletionManual {
		return crmmodel.TaskCompletionManual
	}
	return crmmodel.TaskCompletionSubmit
}

func workTaskAllowsProgress(task *crmmodel.Task) bool {
	if task == nil || task.TaskType != crmmodel.TaskTypeForm {
		return false
	}
	return normalizeWorkTaskCompletionMode(inputText(mapFromAny(task.ConfigJSON)["completion_mode"])) == crmmodel.TaskCompletionManual
}

func workSubmitMode(values map[string]any) string {
	switch firstText(values, "submit_mode", "submitMode") {
	case workSubmitModeProgress:
		return workSubmitModeProgress
	default:
		return workSubmitModeComplete
	}
}

func workSubmitIsProgress(values map[string]any) bool {
	return workSubmitMode(values) == workSubmitModeProgress
}

func workCollaborationTodoAllowsProgress(todo *crmmodel.WorkTodo) bool {
	if todo == nil {
		return false
	}
	if todo.FormID == 0 {
		return false
	}
	return normalizeWorkTaskCompletionMode(todoCompletionMode(todo)) == crmmodel.TaskCompletionManual
}

func todoCompletionMode(todo *crmmodel.WorkTodo) string {
	if todo == nil {
		return crmmodel.TaskCompletionSubmit
	}
	return todo.CompletionMode
}

func upsertWorkTodoMember(ctx context.Context, customerID uint64, assetID uint64, departmentID uint64, staffID uint64) {
	if customerID == 0 || (departmentID == 0 && staffID == 0) {
		return
	}
	model := crmmodel.NewCustomerMemberModel()
	filter := map[string]any{
		"customer_id":   customerID,
		"asset_id":      assetID,
		"department_id": departmentID,
		"staff_id":      staffID,
		"relation_type": crmmodel.MemberRelationParticipant,
		"status":        crmmodel.StatusEnabled,
	}
	if existing := model.Find(ctx, filter); existing != nil {
		model.Update(ctx, map[string]any{"id": existing.ID}, map[string]any{"can_view": true})
		return
	}
	record := copyMap(filter)
	record["can_view"] = true
	record["created_at"] = time.Now()
	model.Insert(ctx, record)
}

func normalizeWorkAssignMode(assignMode string) string {
	if strings.TrimSpace(assignMode) == crmmodel.TaskAssignModeDepartment {
		return crmmodel.TaskAssignModeDepartment
	}
	return crmmodel.TaskAssignModeStaff
}

func workDepartmentLeaderStaffID(ctx context.Context, department *crmmodel.Department) uint64 {
	if department == nil || department.ID == 0 {
		return 0
	}
	if department.LeaderStaffID > 0 {
		if staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
			"id":            department.LeaderStaffID,
			"department_id": department.ID,
			"status":        crmmodel.StatusEnabled,
		}); staff != nil {
			return staff.ID
		}
	}
	if staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"department_id": department.ID,
		"staff_type":    crmmodel.StaffTypeLeader,
		"status":        crmmodel.StatusEnabled,
	}); staff != nil {
		return staff.ID
	}
	return 0
}
