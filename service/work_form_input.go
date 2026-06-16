package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	crmmodel "my/package/crm/model"
)

type workFormInput struct {
	customerFields      map[string]any
	assetFields         map[string]any
	customerDataRecords map[uint64]map[string]any
	assetDataRecords    map[uint64]map[string]any
}

type workFormInputOptions struct {
	allowEmptyCustomerContactFields bool
	allowMissingRequiredFields      bool
	skipEmptyFields                 bool
}

func collectWorkFormInput(ctx context.Context, task *crmmodel.Task, values map[string]any) (*workFormInput, error) {
	if task == nil {
		return nil, fmt.Errorf("任务不能为空")
	}
	return collectWorkFormInputByFormID(ctx, task.FormID, values)
}

func collectWorkCreateFormInput(ctx context.Context, task *crmmodel.Task, values map[string]any) (*workFormInput, error) {
	if task == nil {
		return nil, fmt.Errorf("任务不能为空")
	}
	return collectWorkFormInputByFormID(ctx, task.FormID, values, workFormInputOptions{
		allowEmptyCustomerContactFields: true,
	})
}

func collectWorkFormInputByFormID(ctx context.Context, formID uint64, values map[string]any, options ...workFormInputOptions) (*workFormInput, error) {
	if formID == 0 {
		return nil, fmt.Errorf("任务未配置有效资料模板")
	}
	form := crmmodel.NewFormModel().Find(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled})
	if form == nil {
		return nil, fmt.Errorf("任务未配置有效资料模板")
	}
	fields := crmmodel.NewFormFieldModel().Select(ctx, map[string]any{"form_id": form.ID, "status": crmmodel.StatusEnabled})
	if len(fields) == 0 {
		return nil, fmt.Errorf("资料模板未配置字段")
	}
	result := &workFormInput{
		customerFields:      map[string]any{},
		assetFields:         map[string]any{},
		customerDataRecords: map[uint64]map[string]any{},
		assetDataRecords:    map[uint64]map[string]any{},
	}
	inputOptions := workFormInputOptions{}
	if len(options) > 0 {
		inputOptions = options[0]
	}
	for _, field := range fields {
		if field == nil {
			continue
		}
		value := values[workFieldInputKey(field)]
		if field.Required && emptyWorkFieldValue(value) {
			if !inputOptions.allowMissingRequiredFields && (!inputOptions.allowEmptyCustomerContactFields || !isWorkCustomerContactField(field)) {
				return nil, fmt.Errorf("%s不能为空", field.Name)
			}
		}
		if inputOptions.skipEmptyFields && emptyWorkFieldValue(value) {
			continue
		}
		if field.MainField != "" {
			if isWorkAssetMainField(field) {
				applyWorkAssetMainField(result.assetFields, field.MainField, value)
				continue
			}
			applyWorkCustomerMainField(result.customerFields, field.MainField, value)
			continue
		}
		if field.DataTemplateID > 0 && field.DataFieldID > 0 {
			records := result.customerDataRecords
			if isWorkAssetTemplateField(field) {
				records = result.assetDataRecords
			}
			if records[field.DataTemplateID] == nil {
				records[field.DataTemplateID] = map[string]any{}
			}
			records[field.DataTemplateID][fmt.Sprintf("%d", field.DataFieldID)] = value
		}
	}
	return result, nil
}

func collectWorkFormInputForForm(ctx context.Context, formID uint64, values map[string]any) (*workFormInput, error) {
	return collectWorkFormInputByFormID(ctx, formID, values)
}

func collectWorkProgressFormInput(ctx context.Context, task *crmmodel.Task, values map[string]any) (*workFormInput, error) {
	if task == nil {
		return nil, fmt.Errorf("任务不能为空")
	}
	return collectWorkFormInputByFormID(ctx, task.FormID, values, workFormInputOptions{
		allowMissingRequiredFields: true,
		skipEmptyFields:            true,
	})
}

func collectWorkProgressFormInputForForm(ctx context.Context, formID uint64, values map[string]any) (*workFormInput, error) {
	return collectWorkFormInputByFormID(ctx, formID, values, workFormInputOptions{
		allowMissingRequiredFields: true,
		skipEmptyFields:            true,
	})
}

func collectOptionalWorkFormInput(ctx context.Context, task *crmmodel.Task, values map[string]any) (*workFormInput, error) {
	if task == nil || task.FormID == 0 {
		return emptyWorkFormInput(), nil
	}
	return collectWorkFormInput(ctx, task, values)
}

func emptyWorkFormInput() *workFormInput {
	return &workFormInput{
		customerFields:      map[string]any{},
		assetFields:         map[string]any{},
		customerDataRecords: map[uint64]map[string]any{},
		assetDataRecords:    map[uint64]map[string]any{},
	}
}

func mergeWorkFormInput(target *workFormInput, source *workFormInput) *workFormInput {
	if target == nil {
		target = emptyWorkFormInput()
	}
	if source == nil {
		return target
	}
	for key, value := range source.customerFields {
		target.customerFields[key] = value
	}
	for key, value := range source.assetFields {
		target.assetFields[key] = value
	}
	mergeWorkFormRecordMap(target.customerDataRecords, source.customerDataRecords)
	mergeWorkFormRecordMap(target.assetDataRecords, source.assetDataRecords)
	return target
}

func mergeWorkFormRecordMap(target map[uint64]map[string]any, source map[uint64]map[string]any) {
	for templateID, record := range source {
		if target[templateID] == nil {
			target[templateID] = map[string]any{}
		}
		for key, value := range record {
			target[templateID][key] = value
		}
	}
}

func workFormTaskResultValue(task *crmmodel.Task, formInput *workFormInput) string {
	config := mapFromAny(task.ConfigJSON)
	fieldID := inputUint64(config["result_field_id"])
	if fieldID == 0 {
		return workResultSuccess
	}
	actual := workFormInputDataFieldValue(formInput, fieldID)
	if emptyWorkFieldValue(actual) {
		return "empty"
	}
	if workFormResultAllowed(actual, config["success_values"]) {
		return workResultSuccess
	}
	return inputText(actual)
}

func workFormInputDataFieldValue(formInput *workFormInput, fieldID uint64) any {
	if formInput == nil || fieldID == 0 {
		return nil
	}
	fieldKey := fmt.Sprintf("%d", fieldID)
	for _, records := range []map[uint64]map[string]any{formInput.customerDataRecords, formInput.assetDataRecords} {
		for _, record := range records {
			if value, exists := record[fieldKey]; exists {
				return value
			}
		}
	}
	return nil
}

func workFormResultAllowed(actual any, expected any) bool {
	allowed := stringListFromAny(expected)
	if len(allowed) == 0 {
		return inputText(actual) == workResultSuccess
	}
	for _, value := range allowed {
		if valuesEqual(actual, value) {
			return true
		}
	}
	return false
}

func formInputHasAssetValues(formInput *workFormInput) bool {
	return formInput != nil && (len(formInput.assetFields) > 0 || len(formInput.assetDataRecords) > 0)
}

func ensureWorkFormAsset(ctx context.Context, customerID uint64, assetID uint64, formInput *workFormInput) (uint64, bool, error) {
	if assetID > 0 || !formInputHasAssetValues(formInput) {
		return assetID, false, nil
	}
	createdAssetID, err := createWorkCustomerAsset(ctx, customerID, formInput)
	if err != nil {
		return 0, false, err
	}
	return createdAssetID, true, nil
}

func ensureCreatedWorkAssetStage(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, operationID uint64, taskID uint64, createdAsset bool, fallback *crmmodel.CustomerStage) *crmmodel.CustomerStage {
	if !createdAsset {
		return fallback
	}
	insertWorkCustomerStage(ctx, staff, customerID, assetID, operationID, taskID)
	if state := currentWorkCustomerStage(ctx, customerID, assetID); state != nil {
		return state
	}
	return fallback
}

func workEnteredStageCode(createdStage bool, state *crmmodel.CustomerStage, transitionStageCode string) string {
	if transitionStageCode != "" {
		return transitionStageCode
	}
	if createdStage && state != nil {
		return state.CurrentStageCode
	}
	return ""
}

func saveWorkFormInput(ctx context.Context, customerID uint64, assetID uint64, formInput *workFormInput) error {
	if formInput == nil {
		return nil
	}
	if len(formInput.customerFields) > 0 {
		formInput.customerFields["updated_at"] = time.Now()
		crmmodel.NewCustomerModel().Update(ctx, map[string]any{"id": customerID}, formInput.customerFields)
	}
	if len(formInput.assetFields) > 0 {
		if assetID == 0 {
			return fmt.Errorf("客户资产不能为空")
		}
		formInput.assetFields["updated_at"] = time.Now()
		crmmodel.NewCustomerAssetModel().Update(ctx, map[string]any{"id": assetID, "customer_id": customerID}, formInput.assetFields)
	}
	return nil
}

func createWorkCustomerAsset(ctx context.Context, customerID uint64, formInput *workFormInput) (uint64, error) {
	assetRecord := defaultWorkAssetRecord(customerID)
	assetSeq, assetNo, err := nextWorkCustomerAssetIdentity(ctx, customerID)
	if err != nil {
		return 0, err
	}
	assetRecord["asset_seq"] = assetSeq
	assetRecord["asset_no"] = assetNo
	if formInput != nil {
		for key, value := range formInput.assetFields {
			assetRecord[key] = value
		}
		formInput.assetFields = map[string]any{}
	}
	return uint64(crmmodel.NewCustomerAssetModel().Insert(ctx, assetRecord)), nil
}

func nextWorkCustomerAssetIdentity(ctx context.Context, customerID uint64) (uint64, string, error) {
	customerCode, err := ensureWorkCustomerCode(ctx, customerID)
	if err != nil {
		return 0, "", err
	}
	model := crmmodel.NewCustomerAssetModel()
	assets := model.Select(ctx, map[string]any{"customer_id": customerID})
	seq := uint64(len(assets) + 1)
	prefix := customerCodePrefixForWork(ctx)
	for i := 0; i < 200; i++ {
		assetNo := fmt.Sprintf("%s%s-%d", prefix, customerCode, seq)
		if model.Find(ctx, map[string]any{"asset_no": assetNo}) == nil {
			return seq, assetNo, nil
		}
		seq++
	}
	return 0, "", fmt.Errorf("资产编号生成失败，请重试")
}

func ensureWorkCustomerCode(ctx context.Context, customerID uint64) (string, error) {
	customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
	if customer == nil {
		return "", fmt.Errorf("客户不存在")
	}
	code := strings.TrimSpace(customer.Code)
	if code != "" {
		return code, nil
	}
	code, err := crmmodel.GenerateUniqueCustomerCode(ctx)
	if err != nil {
		return "", err
	}
	crmmodel.NewCustomerModel().Update(ctx, map[string]any{"id": customerID}, map[string]any{
		"code":       code,
		"updated_at": time.Now(),
	})
	return code, nil
}

func saveWorkFormDataRecords(ctx context.Context, customerID uint64, assetID uint64, taskID uint64, operationID uint64, formInput *workFormInput) {
	if formInput == nil {
		return
	}
	for templateID, record := range formInput.customerDataRecords {
		saveWorkDataRecord(ctx, customerID, 0, templateID, taskID, operationID, record)
	}
	if assetID == 0 {
		return
	}
	for templateID, record := range formInput.assetDataRecords {
		saveWorkDataRecord(ctx, customerID, assetID, templateID, taskID, operationID, record)
	}
}

func defaultWorkCustomerRecord(staff *WorkStaffSession) map[string]any {
	now := time.Now()
	return map[string]any{
		"code":                "",
		"name":                "",
		"phone":               "",
		"wechat":              "",
		"id_card":             "",
		"gender":              "unknown",
		"source_id":           crmmodel.DefaultCustomerSourceID,
		"channel_id":          crmmodel.DefaultCustomerChannelID,
		"level_id":            crmmodel.DefaultCustomerLevelID,
		"tags":                "",
		"remark":              "",
		"created_by_staff_id": staff.ID,
		"created_at":          now,
		"updated_at":          now,
	}
}

func defaultWorkAssetRecord(customerID uint64) map[string]any {
	now := time.Now()
	return map[string]any{
		"asset_no":        defaultAssetNo(),
		"asset_name":      "",
		"customer_id":     customerID,
		"asset_status_id": crmmodel.DefaultAssetStatusID,
		"remark":          "",
		"created_at":      now,
		"updated_at":      now,
	}
}

func duplicatedWorkCustomerField(ctx context.Context, record map[string]any) string {
	model := crmmodel.NewCustomerModel()
	for _, field := range []string{"phone", "wechat", "id_card"} {
		value := inputText(record[field])
		if value == "" {
			continue
		}
		if customer := model.Find(ctx, map[string]any{field: value}); customer != nil {
			return field
		}
	}
	return ""
}

func validateWorkCustomerContact(record map[string]any) error {
	if inputText(record["phone"]) != "" || inputText(record["wechat"]) != "" {
		return nil
	}
	return fmt.Errorf("手机号和微信号至少填写一个")
}

func duplicateWorkCustomerFieldMessage(field string) string {
	switch field {
	case "phone":
		return "手机号已存在。"
	case "wechat":
		return "微信已存在。"
	case "id_card":
		return "身份证号已存在。"
	default:
		return "线索已存在。"
	}
}

func applyWorkCustomerMainField(record map[string]any, field string, value any) {
	switch field {
	case "name", "phone", "wechat", "id_card", "gender", "tags", "remark":
		record[field] = inputText(value)
	case "source_id", "channel_id", "level_id":
		if id := inputUint64(value); id > 0 {
			record[field] = id
		}
	default:
		if field != "" {
			record[field] = value
		}
	}
}

func applyWorkAssetMainField(record map[string]any, field string, value any) {
	switch field {
	case "asset_name", "remark":
		record[field] = inputText(value)
	case "asset_status_id":
		if id := inputUint64(value); id > 0 {
			record[field] = id
		}
	default:
		if field != "" {
			record[field] = value
		}
	}
}

func isWorkAssetMainField(field *crmmodel.FormField) bool {
	if field == nil {
		return false
	}
	if isWorkAssetTemplateField(field) {
		return true
	}
	switch field.MainField {
	case "asset_name", "asset_status_id":
		return true
	default:
		return false
	}
}

func isWorkAssetTemplateField(field *crmmodel.FormField) bool {
	return field != nil && field.DataTemplateCateID == crmmodel.CustomerAssetDataTemplateCateID
}

func isWorkCustomerContactField(field *crmmodel.FormField) bool {
	if field == nil || isWorkAssetTemplateField(field) {
		return false
	}
	switch field.MainField {
	case "phone", "wechat":
		return true
	default:
		return false
	}
}
