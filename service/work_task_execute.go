package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

type workOperationCompletion struct {
	customerID       uint64
	assetID          uint64
	businessObjectID uint64
	operationID      uint64
	task             *crmmodel.Task
	formInput        *workFormInput
	resultValue      string
	todoID           uint64
	progressOnly     bool
}

type workFormOperationCompletion struct {
	customerID            uint64
	assetID               uint64
	businessObjectID      uint64
	createdAsset          bool
	createdBusinessObject bool
	operationID           uint64
	task                  *crmmodel.Task
	formInput             *workFormInput
	resultValue           string
	fromState             *crmmodel.CustomerStage
}

func executeWorkTask(ctx context.Context, staff *WorkStaffSession, payload map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	taskID := firstUint64(payload, "task_id", "taskId")
	if taskID == 0 {
		return nil, fmt.Errorf("任务不能为空")
	}
	customerID := firstUint64(payload, "customer_id", "customerId")
	assetID := firstUint64(payload, "asset_id", "assetId")
	businessObjectID := firstUint64(payload, "business_object_id", "businessObjectId")
	if businessObjectID == 0 {
		businessObjectID = firstUint64(mapFromAny(payload["values"]), "business_object_id", "businessObjectId")
	}
	if assetID > 0 && !workCustomerOwnsAsset(ctx, customerID, assetID) {
		return nil, fmt.Errorf("客户资产不存在")
	}
	if businessObjectID > 0 && !workCustomerOwnsBusinessObject(ctx, customerID, assetID, businessObjectID) {
		return nil, fmt.Errorf("业务对象不存在")
	}
	task := workAllowedTask(ctx, staff, taskID, customerID, assetID, firstUint64(payload, "todo_id", "todoId"))
	if task == nil {
		return nil, fmt.Errorf("当前人员无权执行该任务")
	}
	if !beginWorkTaskExecution(runtime, customerID, assetID, task.ID) {
		return map[string]any{"customer_id": customerID, "skipped": true}, nil
	}
	defer endWorkTaskExecution(runtime, customerID, assetID, task.ID)
	switch task.TaskType {
	case crmmodel.TaskTypeCreate:
		if customerID > 0 {
			return nil, fmt.Errorf("创建资料任务不能在已有客户上执行")
		}
		return executeCreateCustomerTask(ctx, staff, task, mapFromAny(payload["values"]), runtime)
	case crmmodel.TaskTypeForm:
		values := workActionValues(payload)
		values["business_object_id"] = businessObjectID
		return executeFormTask(ctx, staff, task, customerID, assetID, values, runtime)
	case crmmodel.TaskTypeAssign:
		if customerID == 0 {
			return nil, fmt.Errorf("客户不能为空")
		}
		values := workActionValues(payload)
		values["business_object_id"] = businessObjectID
		return executeAssignCustomerTask(ctx, staff, task, customerID, assetID, values, runtime)
	case crmmodel.TaskTypeCollaborate:
		if customerID == 0 {
			return nil, fmt.Errorf("客户不能为空")
		}
		values := workActionValues(payload)
		values["business_object_id"] = businessObjectID
		return executeCollaborateCustomerTask(ctx, staff, task, customerID, assetID, values, runtime)
	case crmmodel.TaskTypeDecision:
		if customerID == 0 {
			return nil, fmt.Errorf("客户不能为空")
		}
		return executeDecisionCustomerTask(ctx, staff, task, customerID, assetID, mapFromAny(payload["values"]), runtime)
	case crmmodel.TaskTypeBooking:
		if customerID == 0 {
			return nil, fmt.Errorf("客户不能为空")
		}
		return executeBookingCustomerTask(ctx, staff, task, customerID, assetID, mapFromAny(payload["values"]), runtime)
	default:
		return nil, fmt.Errorf("该任务动作暂未接入工作台")
	}
}

func executeFormTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if customerID == 0 {
		return nil, fmt.Errorf("客户不能为空")
	}
	return executeEditFormTask(ctx, staff, task, customerID, assetID, values, runtime)
}

func requireExistingWorkBusinessObject(ctx context.Context, task *crmmodel.Task, formInput *workFormInput, businessObjectID uint64) error {
	if task == nil || !formInputHasBusinessObjectValues(formInput) {
		return nil
	}
	if normalizeWorkBusinessObjectMode(inputText(mapFromAny(task.ConfigJSON)["business_object_mode"])) != "select" {
		return nil
	}
	if businessObjectID > 0 {
		return nil
	}
	typeID, err := workBusinessObjectTypeIDForFormInput(ctx, formInput)
	if err != nil {
		return err
	}
	typeName := workBusinessObjectTypeName(ctx, typeID)
	if typeName == "" {
		typeName = "业务对象"
	}
	return fmt.Errorf("请选择%s", typeName)
}

func completeWorkFormOperation(ctx context.Context, staff *WorkStaffSession, runtime *workExecutionRuntime, completion workFormOperationCompletion) (map[string]any, error) {
	saveWorkFormObjectDataRecords(ctx, completion.customerID, completion.assetID, completion.businessObjectID, completion.task.ID, completion.operationID, completion.formInput)
	afterWorkOperationCompleted(ctx, staff, workOperationCompletion{
		customerID:       completion.customerID,
		assetID:          completion.assetID,
		businessObjectID: completion.businessObjectID,
		operationID:      completion.operationID,
		task:             completion.task,
		formInput:        completion.formInput,
		resultValue:      completion.resultValue,
	})
	fromState := ensureCreatedWorkAssetStage(ctx, staff, completion.customerID, completion.assetID, completion.operationID, completion.task.ID, completion.createdAsset, completion.fromState)
	transitionStageCode := applyWorkStageTransition(ctx, staff, completion.customerID, completion.assetID, fromState, completion.task, completion.operationID, completion.resultValue)
	runWorkAutoTriggers(ctx, staff, completion.customerID, completion.assetID, completion.task, completion.resultValue, workEnteredStageCode(completion.createdAsset, fromState, transitionStageCode), runtime)
	return map[string]any{
		"customer_id":        completion.customerID,
		"asset_id":           completion.assetID,
		"business_object_id": completion.businessObjectID,
		"result_value":       completion.resultValue,
		"saved":              true,
	}, nil
}

func afterWorkOperationCompleted(ctx context.Context, staff *WorkStaffSession, completion workOperationCompletion) {
	if completion.operationID == 0 || completion.task == nil {
		return
	}
	if !completion.progressOnly {
		syncWorkTaskPointLedger(ctx, staff, completion)
	}
	syncWorkFinanceLedgers(ctx, staff, completion)
}

func executeCreateCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	formInput, err := collectWorkCreateFormInput(ctx, task, values)
	if err != nil {
		return nil, err
	}
	customerRecord := defaultWorkCustomerRecord(staff)
	for key, value := range formInput.customerFields {
		customerRecord[key] = value
	}
	if err := validateWorkCustomerContact(customerRecord); err != nil {
		return nil, err
	}
	if duplicateField := duplicatedWorkCustomerField(ctx, customerRecord); duplicateField != "" {
		return nil, fmt.Errorf("%s", duplicateWorkCustomerFieldMessage(duplicateField))
	}
	if inputText(customerRecord["code"]) == "" {
		code, err := crmmodel.GenerateUniqueCustomerCode(ctx)
		if err != nil {
			return nil, err
		}
		customerRecord["code"] = code
	}
	customerID := uint64(crmmodel.NewCustomerModel().Insert(ctx, customerRecord))
	operationID := insertWorkOperationLog(ctx, staff, task, customerID, 0, values)
	for templateID, record := range formInput.customerDataRecords {
		saveWorkDataRecord(ctx, customerID, 0, templateID, task.ID, operationID, record)
	}
	afterWorkOperationCompleted(ctx, staff, workOperationCompletion{
		customerID:  customerID,
		operationID: operationID,
		task:        task,
		formInput:   formInput,
		resultValue: workResultSuccess,
	})
	insertWorkCustomerMember(ctx, staff, customerID)
	insertWorkCustomerStage(ctx, staff, customerID, 0, operationID, task.ID)
	fromState := currentWorkCustomerStage(ctx, customerID, 0)
	transitionStageCode := applyWorkStageTransition(ctx, staff, customerID, 0, fromState, task, operationID, workResultSuccess)
	runWorkAutoTriggers(ctx, staff, customerID, 0, task, workResultSuccess, workEnteredStageCode(true, fromState, transitionStageCode), runtime)
	return map[string]any{
		"customer_id": customerID,
		"saved":       true,
	}, nil
}

func executeEditFormTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	fromState := currentWorkCustomerStage(ctx, customerID, assetID)
	progressSubmit := workSubmitIsProgress(values) && workTaskAllowsProgress(task)
	var formInput *workFormInput
	var err error
	if progressSubmit {
		formInput, err = collectWorkProgressFormInput(ctx, task, values)
	} else {
		formInput, err = collectWorkFormInput(ctx, task, values)
	}
	if err != nil {
		return nil, err
	}
	businessObjectID := firstUint64(values, "business_object_id", "businessObjectId")
	var createdAsset bool
	assetID, createdAsset, err = ensureWorkFormAsset(ctx, customerID, assetID, formInput)
	if err != nil {
		return nil, err
	}
	if err := requireExistingWorkBusinessObject(ctx, task, formInput, businessObjectID); err != nil {
		return nil, err
	}
	var createdBusinessObject bool
	businessObjectID, createdBusinessObject, err = ensureWorkFormBusinessObject(ctx, staff, customerID, assetID, businessObjectID, formInput)
	if err != nil {
		return nil, err
	}
	if err := saveWorkFormInput(ctx, customerID, assetID, formInput); err != nil {
		return nil, err
	}
	if progressSubmit {
		operationID := insertWorkOperationLogWithResult(ctx, staff, task, customerID, assetID, fromState, values, workResultProgress)
		saveWorkFormObjectDataRecords(ctx, customerID, assetID, businessObjectID, task.ID, operationID, formInput)
		afterWorkOperationCompleted(ctx, staff, workOperationCompletion{
			customerID:       customerID,
			assetID:          assetID,
			businessObjectID: businessObjectID,
			operationID:      operationID,
			task:             task,
			formInput:        formInput,
			resultValue:      workResultProgress,
			progressOnly:     true,
		})
		fromState = ensureCreatedWorkAssetStage(ctx, staff, customerID, assetID, operationID, task.ID, createdAsset, fromState)
		updateWorkCustomerStageOperation(ctx, customerID, assetID, operationID)
		return map[string]any{
			"customer_id":        customerID,
			"asset_id":           assetID,
			"business_object_id": businessObjectID,
			"result_value":       workResultProgress,
			"saved":              true,
			"progress":           true,
		}, nil
	}
	resultValue := workFormTaskResultValue(task, formInput)
	operationID := insertWorkOperationLogWithResult(ctx, staff, task, customerID, assetID, fromState, values, resultValue)
	return completeWorkFormOperation(ctx, staff, runtime, workFormOperationCompletion{
		customerID:            customerID,
		assetID:               assetID,
		businessObjectID:      businessObjectID,
		createdAsset:          createdAsset,
		createdBusinessObject: createdBusinessObject,
		operationID:           operationID,
		task:                  task,
		formInput:             formInput,
		resultValue:           resultValue,
		fromState:             fromState,
	})
}

func executeAssignCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	formInput, err := collectOptionalWorkFormInput(ctx, task, values)
	if err != nil {
		return nil, err
	}
	config := mapFromAny(task.ConfigJSON)
	assignMode := normalizeWorkAssignMode(inputText(config["assign_mode"]))
	targetDepartmentID, targetStaffID, err := resolveWorkAssignTarget(ctx, assignMode, uint64ListFromAny(config["assign_department_ids"]), values)
	if err != nil {
		return nil, err
	}
	fromState := currentWorkCustomerStage(ctx, customerID, assetID)
	businessObjectID := firstUint64(values, "business_object_id", "businessObjectId")
	var createdAsset bool
	assetID, createdAsset, err = ensureWorkFormAsset(ctx, customerID, assetID, formInput)
	if err != nil {
		return nil, err
	}
	if err := requireExistingWorkBusinessObject(ctx, task, formInput, businessObjectID); err != nil {
		return nil, err
	}
	businessObjectID, _, err = ensureWorkFormBusinessObject(ctx, staff, customerID, assetID, businessObjectID, formInput)
	if err != nil {
		return nil, err
	}
	if err := saveWorkFormInput(ctx, customerID, assetID, formInput); err != nil {
		return nil, err
	}
	logValues := copyMap(values)
	logValues["department_id"] = targetDepartmentID
	logValues["staff_id"] = targetStaffID
	operationID := insertWorkOperationLog(ctx, staff, task, customerID, assetID, logValues)
	saveWorkFormObjectDataRecords(ctx, customerID, assetID, businessObjectID, task.ID, operationID, formInput)
	afterWorkOperationCompleted(ctx, staff, workOperationCompletion{
		customerID:       customerID,
		assetID:          assetID,
		businessObjectID: businessObjectID,
		operationID:      operationID,
		task:             task,
		formInput:        formInput,
		resultValue:      workResultSuccess,
	})
	fromState = ensureCreatedWorkAssetStage(ctx, staff, customerID, assetID, operationID, task.ID, createdAsset, fromState)
	updateWorkCustomerOwner(ctx, customerID, assetID, targetDepartmentID, targetStaffID, operationID)
	upsertWorkAssigneeMember(ctx, customerID, assetID, targetDepartmentID, targetStaffID)
	transitionStageCode := applyWorkStageTransitionWithOwner(ctx, staff, customerID, assetID, fromState, task, operationID, workResultSuccess, targetDepartmentID, targetStaffID)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, workResultSuccess, workEnteredStageCode(createdAsset, fromState, transitionStageCode), runtime)
	return map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"saved":       true,
	}, nil
}

func executeCollaborateCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	todoID := firstUint64(values, "todo_id", "todoId")
	if todoID > 0 {
		return completeWorkTodo(ctx, staff, task, customerID, assetID, todoID, values, runtime)
	}
	if workCollaborationConfirmRequested(values) {
		return confirmWorkCollaborationTask(ctx, staff, task, customerID, assetID, values, runtime)
	}

	targets, err := workCollaborationTargets(ctx, task, values)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("请配置协作子任务")
	}
	formInput, err := collectOptionalWorkFormInput(ctx, task, values)
	if err != nil {
		return nil, err
	}
	businessObjectID := firstUint64(values, "business_object_id", "businessObjectId")
	assetID, createdAsset, err := ensureWorkFormAsset(ctx, customerID, assetID, formInput)
	if err != nil {
		return nil, err
	}
	if err := requireExistingWorkBusinessObject(ctx, task, formInput, businessObjectID); err != nil {
		return nil, err
	}
	businessObjectID, _, err = ensureWorkFormBusinessObject(ctx, staff, customerID, assetID, businessObjectID, formInput)
	if err != nil {
		return nil, err
	}
	if err := saveWorkFormInput(ctx, customerID, assetID, formInput); err != nil {
		return nil, err
	}
	logValues := copyMap(values)
	logValues["todo_count"] = len(targets)
	operationID := insertWorkOperationLog(ctx, staff, task, customerID, assetID, logValues)
	saveWorkFormObjectDataRecords(ctx, customerID, assetID, businessObjectID, task.ID, operationID, formInput)
	afterWorkOperationCompleted(ctx, staff, workOperationCompletion{
		customerID:       customerID,
		assetID:          assetID,
		businessObjectID: businessObjectID,
		operationID:      operationID,
		task:             task,
		formInput:        formInput,
		resultValue:      workResultSuccess,
	})
	fromState := ensureCreatedWorkAssetStage(ctx, staff, customerID, assetID, operationID, task.ID, createdAsset, nil)
	todos := createWorkCollaborationTodos(ctx, staff, task, customerID, assetID, operationID, targets)
	updateWorkCustomerStageOperation(ctx, customerID, assetID, operationID)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, workResultSuccess, workEnteredStageCode(createdAsset, fromState, ""), runtime)
	return map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"todo_count":  len(todos),
		"saved":       true,
	}, nil
}

func executeDecisionCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	fromState := currentWorkCustomerStage(ctx, customerID, assetID)
	result, err := resolveWorkDecisionResult(ctx, staff, task, customerID, assetID, fromState, values)
	if err != nil {
		return nil, err
	}
	resultTarget, err := resolveWorkDecisionResultTarget(ctx, customerID, assetID, task, result.Value)
	if err != nil {
		return nil, err
	}
	logValues := map[string]any{"decision_result": result}
	if strings.TrimSpace(result.Reason) != "" {
		logValues["decision_reason"] = result.Reason
	}
	operationID := insertWorkOperationLogWithResult(ctx, staff, task, customerID, assetID, fromState, logValues, result.Value)
	if err := writeWorkDecisionResult(ctx, customerID, task, operationID, resultTarget, result.Value); err != nil {
		return nil, err
	}
	afterWorkOperationCompleted(ctx, staff, workOperationCompletion{
		customerID:  customerID,
		assetID:     assetID,
		operationID: operationID,
		task:        task,
		resultValue: result.Value,
	})
	transitionStageCode := applyWorkStageTransition(ctx, staff, customerID, assetID, fromState, task, operationID, result.Value)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, result.Value, transitionStageCode, runtime)
	return map[string]any{
		"customer_id":  customerID,
		"result_value": result.Value,
		"reason":       result.Reason,
		"saved":        true,
	}, nil
}

func executeBookingCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	fromState := currentWorkCustomerStage(ctx, customerID, assetID)
	bookingInput, err := collectWorkBookingInput(ctx, task, values)
	if err != nil {
		return nil, err
	}
	resource := crmmodel.NewPublicResourceModel().Find(ctx, map[string]any{
		"id":     bookingInput.resourceID,
		"status": crmmodel.StatusEnabled,
	})
	if resource == nil {
		return nil, fmt.Errorf("公共资源不存在或已停用")
	}
	if resource.ResourceCateID != bookingInput.resourceCateID {
		return nil, fmt.Errorf("该资源不属于当前任务配置的资源分类")
	}
	if err := ensureWorkBookingTimeAvailable(ctx, 0, bookingInput.resourceID, bookingInput.startAt, bookingInput.endAt); err != nil {
		return nil, err
	}
	bookingStatus := crmmodel.ResourceBookingStatusReserved
	if bookingInput.needConfirm || resource.NeedConfirm {
		bookingStatus = crmmodel.ResourceBookingStatusPending
	}
	resultValue := workResultSuccess
	if bookingStatus == crmmodel.ResourceBookingStatusPending {
		resultValue = crmmodel.ResourceBookingStatusPending
	}
	operationID := insertWorkOperationLogWithResult(ctx, staff, task, customerID, assetID, fromState, values, resultValue)
	bookingID := insertWorkResourceBooking(ctx, staff, task, customerID, assetID, operationID, fromState, bookingInput, bookingStatus)
	afterWorkOperationCompleted(ctx, staff, workOperationCompletion{
		customerID:  customerID,
		assetID:     assetID,
		operationID: operationID,
		task:        task,
		resultValue: resultValue,
	})
	transitionStageCode := applyWorkStageTransition(ctx, staff, customerID, assetID, fromState, task, operationID, resultValue)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, resultValue, transitionStageCode, runtime)
	return map[string]any{
		"customer_id":    customerID,
		"booking_id":     bookingID,
		"booking_status": bookingStatus,
		"result_value":   resultValue,
		"saved":          true,
	}, nil
}

type workDecisionResultTarget struct {
	field      *crmmodel.DataField
	writeAsset uint64
}

func resolveWorkDecisionResultTarget(ctx context.Context, customerID uint64, assetID uint64, task *crmmodel.Task, resultValue string) (workDecisionResultTarget, error) {
	fieldID := inputUint64(mapFromAny(task.ConfigJSON)["decision_result_field_id"])
	if fieldID == 0 {
		return workDecisionResultTarget{}, fmt.Errorf("决策任务未配置结果写入字段")
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":     fieldID,
		"status": crmmodel.StatusEnabled,
	})
	if field == nil {
		return workDecisionResultTarget{}, fmt.Errorf("结果写入字段不存在或已停用")
	}
	if strings.TrimSpace(field.FieldKey) == "" {
		return workDecisionResultTarget{}, fmt.Errorf("结果写入字段必须配置字段编码")
	}
	if !workDecisionResultFieldHasOptions(field.FieldType) {
		return workDecisionResultTarget{}, fmt.Errorf("结果写入字段必须是单选或下拉字段")
	}
	if !workDecisionFieldOptionExists(ctx, field.ID, resultValue) {
		return workDecisionResultTarget{}, fmt.Errorf("自动决策结果 %s 不属于结果写入字段的可选项", resultValue)
	}
	writeAssetID := assetID
	templateCateID := workDataFieldTemplateCateID(ctx, field)
	if templateCateID == crmmodel.CustomerDataTemplateCateID {
		writeAssetID = 0
	} else if templateCateID == crmmodel.CustomerAssetDataTemplateCateID && writeAssetID == 0 {
		return workDecisionResultTarget{}, fmt.Errorf("资产级结果写入字段需要当前客户资产")
	}
	return workDecisionResultTarget{field: field, writeAsset: writeAssetID}, nil
}

func writeWorkDecisionResult(ctx context.Context, customerID uint64, task *crmmodel.Task, operationID uint64, target workDecisionResultTarget, resultValue string) error {
	if target.field == nil {
		return fmt.Errorf("决策任务未配置结果写入字段")
	}
	saveWorkDataRecord(ctx, customerID, target.writeAsset, target.field.DataTemplateID, task.ID, operationID, map[string]any{
		fmt.Sprintf("%d", target.field.ID): resultValue,
	})
	return nil
}

func workDecisionResultFieldHasOptions(fieldType string) bool {
	switch strings.TrimSpace(fieldType) {
	case "radio", "select":
		return true
	default:
		return false
	}
}

func workDecisionFieldOptionExists(ctx context.Context, fieldID uint64, value string) bool {
	if fieldID == 0 || strings.TrimSpace(value) == "" {
		return false
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID, "status": crmmodel.StatusEnabled})
	if field == nil {
		return false
	}
	if field.OptionSetID > 0 {
		return crmmodel.NewOptionSetItemModel().Find(ctx, map[string]any{
			"option_set_id": field.OptionSetID,
			"value":         strings.TrimSpace(value),
			"status":        crmmodel.StatusEnabled,
		}) != nil
	}
	return crmmodel.NewDataFieldOptionModel().Find(ctx, map[string]any{
		"data_field_id": fieldID,
		"value":         strings.TrimSpace(value),
	}) != nil
}

func workDataFieldTemplateCateID(ctx context.Context, field *crmmodel.DataField) uint64 {
	if field == nil || field.DataTemplateID == 0 {
		return 0
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": field.DataTemplateID})
	if template == nil {
		return 0
	}
	return template.CateID
}

type workBookingInput struct {
	resourceCateID uint64
	resourceID     uint64
	startAt        time.Time
	endAt          time.Time
	title          string
	remark         string
	needConfirm    bool
}

func collectWorkBookingInput(ctx context.Context, task *crmmodel.Task, values map[string]any) (*workBookingInput, error) {
	config := mapFromAny(task.ConfigJSON)
	resourceCateID := inputUint64(config["resource_cate_id"])
	if resourceCateID == 0 {
		resourceCateID = crmmodel.DefaultResourceCateID
	}
	resourceID := firstUint64(values, "resource_id", "resourceId", "booking:resource_id", "main:resource_id")
	if resourceID == 0 {
		return nil, fmt.Errorf("请选择资源")
	}
	startAt, err := parseWorkBookingDateTime(firstText(values, "start_at", "startAt", "booking:start_at", "main:start_at"))
	if err != nil {
		return nil, fmt.Errorf("开始时间格式错误")
	}
	endAt, err := parseWorkBookingDateTime(firstText(values, "end_at", "endAt", "booking:end_at", "main:end_at"))
	if err != nil {
		return nil, fmt.Errorf("结束时间格式错误")
	}
	if !endAt.After(startAt) {
		return nil, fmt.Errorf("结束时间必须晚于开始时间")
	}
	title := firstText(values, "title", "booking:title", "main:title")
	if title == "" {
		return nil, fmt.Errorf("用途不能为空")
	}
	if crmmodel.NewPublicResourceCateModel().Find(ctx, map[string]any{"id": resourceCateID, "status": crmmodel.StatusEnabled}) == nil {
		return nil, fmt.Errorf("资源分类不存在或已停用")
	}
	return &workBookingInput{
		resourceCateID: resourceCateID,
		resourceID:     resourceID,
		startAt:        startAt,
		endAt:          endAt,
		title:          title,
		remark:         firstText(values, "remark", "booking:remark", "main:remark"),
		needConfirm:    booleanFromAny(config["need_confirm"]),
	}, nil
}

func ensureWorkBookingTimeAvailable(ctx context.Context, currentID uint64, resourceID uint64, startAt time.Time, endAt time.Time) error {
	for _, booking := range crmmodel.NewPublicResourceBookingModel().Select(ctx, map[string]any{"resource_id": resourceID}) {
		if booking == nil || booking.ID == currentID || workBookingInactiveStatus(booking.BookingStatus) {
			continue
		}
		if startAt.Before(booking.EndAt) && endAt.After(booking.StartAt) {
			return fmt.Errorf("该资源在所选时间已被预定")
		}
	}
	return nil
}

func workBookingInactiveStatus(status string) bool {
	return status == crmmodel.ResourceBookingStatusCanceled || status == crmmodel.ResourceBookingStatusRejected
}

func parseWorkBookingDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid datetime")
}

func enrichWorkBookingRow(ctx context.Context, row map[string]any) {
	resourceID := inputUint64(row["resource_id"])
	if resourceID > 0 {
		resource := crmmodel.NewPublicResourceModel().Find(ctx, map[string]any{"id": resourceID})
		if resource != nil {
			row["resource.name"] = resource.Name
			row["resource.location"] = resource.Location
		}
	}
	customerID := inputUint64(row["customer_id"])
	if customerID > 0 {
		customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
		if customer != nil {
			row["customer.name"] = customer.Name
			row["customer.phone"] = customer.Phone
		}
	}
	row["booking_status_name"] = workBookingStatusName(inputText(row["booking_status"]))
}

func workBookingStatusName(status string) string {
	switch status {
	case crmmodel.ResourceBookingStatusPending:
		return "待确认"
	case crmmodel.ResourceBookingStatusReserved:
		return "已预定"
	case crmmodel.ResourceBookingStatusCanceled:
		return "已取消"
	case crmmodel.ResourceBookingStatusRejected:
		return "已拒绝"
	case crmmodel.ResourceBookingStatusDone:
		return "已完成"
	default:
		return status
	}
}

func insertWorkResourceBooking(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, operationID uint64, state *crmmodel.CustomerStage, input *workBookingInput, bookingStatus string) uint64 {
	now := time.Now()
	statusCode := ""
	if state != nil {
		statusCode = state.CurrentStageCode
	}
	return uint64(crmmodel.NewPublicResourceBookingModel().Insert(ctx, map[string]any{
		"resource_id":          input.resourceID,
		"customer_id":          customerID,
		"asset_id":             assetID,
		"task_id":              task.ID,
		"operation_log_id":     operationID,
		"stage_code":           statusCode,
		"booking_status":       bookingStatus,
		"title":                input.title,
		"remark":               input.remark,
		"start_at":             input.startAt,
		"end_at":               input.endAt,
		"booker_staff_id":      staff.ID,
		"booker_department_id": staff.DepartmentID,
		"created_at":           now,
		"updated_at":           now,
	}))
}

func resolveWorkAssignTarget(ctx context.Context, assignMode string, allowedDepartmentIDs []uint64, values map[string]any) (uint64, uint64, error) {
	departmentID := firstUint64(values, "department_id", "departmentId")
	staffID := firstUint64(values, "staff_id", "staffId")
	if departmentID == 0 {
		return 0, 0, fmt.Errorf("请选择部门")
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": departmentID, "status": crmmodel.StatusEnabled})
	if department == nil {
		return 0, 0, fmt.Errorf("部门不存在或已停用")
	}
	if len(allowedDepartmentIDs) > 0 && !uint64SetContains(allowedDepartmentIDs, departmentID) {
		return 0, 0, fmt.Errorf("该部门不在当前任务可选范围内")
	}
	switch assignMode {
	case crmmodel.TaskAssignModeDepartment:
		leaderStaffID := workDepartmentLeaderStaffID(ctx, department)
		if leaderStaffID == 0 {
			return 0, 0, fmt.Errorf("该部门未配置负责人，无法自动派单")
		}
		return departmentID, leaderStaffID, nil
	default:
		if staffID == 0 {
			leaderStaffID := workDepartmentLeaderStaffID(ctx, department)
			if leaderStaffID == 0 {
				return 0, 0, fmt.Errorf("该部门未配置负责人，无法自动派单")
			}
			return departmentID, leaderStaffID, nil
		}
		targetStaff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID, "status": crmmodel.StatusEnabled})
		if targetStaff == nil {
			return 0, 0, fmt.Errorf("人员不存在或已停用")
		}
		if targetStaff.DepartmentID != departmentID {
			return 0, 0, fmt.Errorf("人员不属于所选部门")
		}
		return departmentID, staffID, nil
	}
}
