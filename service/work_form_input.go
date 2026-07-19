package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

type workFormInput struct {
	leadFields           map[string]any
	customerFields       map[string]any
	assetFields          map[string]any
	customerTagIDs       []uint64
	customerTagsProvided bool
	leadDataRecords      map[uint64]map[string]any
	customerDataRecords  map[uint64]map[string]any
	assetDataRecords     map[uint64]map[string]any
}

type workFormInputOptions struct {
	allowEmptyCustomerContactFields bool
	allowMissingRequiredFields      bool
	skipMissingFields               bool
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
		leadFields:          map[string]any{},
		customerFields:      map[string]any{},
		assetFields:         map[string]any{},
		customerTagIDs:      []uint64{},
		leadDataRecords:     map[uint64]map[string]any{},
		customerDataRecords: map[uint64]map[string]any{},
		assetDataRecords:    map[uint64]map[string]any{},
	}
	inputOptions := workFormInputOptions{}
	if len(options) > 0 {
		inputOptions = options[0]
	}
	for _, sourceField := range fields {
		for _, field := range expandWorkInputFormFields(ctx, sourceField) {
			if field == nil {
				continue
			}
			value, submitted := values[workFieldInputKey(field)]
			if field.Required && emptyWorkFieldValue(value) {
				if !inputOptions.allowMissingRequiredFields && (!inputOptions.allowEmptyCustomerContactFields || !isWorkCustomerContactField(field)) {
					return nil, fmt.Errorf("%s不能为空", field.Name)
				}
			}
			if inputOptions.skipMissingFields && !submitted {
				continue
			}
			if field.Readonly {
				continue
			}
			if field.MainField != "" {
				if isWorkLeadTemplateField(field) {
					applyWorkLeadMainField(result.leadFields, field.MainField, value)
					continue
				}
				if isWorkAssetMainField(field) {
					applyWorkAssetMainField(result.assetFields, field.MainField, value)
					continue
				}
				if field.MainField == "tags" {
					selection, err := ResolveCustomerTagSelection(ctx, value)
					if err != nil {
						return nil, err
					}
					result.customerTagIDs = selection.TagIDs
					result.customerTagsProvided = submitted
					continue
				}
				applyWorkCustomerMainField(result.customerFields, field.MainField, value)
				continue
			}
			if field.DataTemplateID > 0 && field.DataFieldID > 0 {
				records := workFormRecordBucket(ctx, result, field)
				if records[field.DataTemplateID] == nil {
					records[field.DataTemplateID] = map[string]any{}
				}
				records[field.DataTemplateID][fmt.Sprintf("%d", field.DataFieldID)] = value
			}
		}
	}
	return result, nil
}

func expandWorkInputFormFields(ctx context.Context, field *crmmodel.FormField) []*crmmodel.FormField {
	if field == nil || field.DataFieldID == 0 {
		return []*crmmodel.FormField{field}
	}
	dataField := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":     field.DataFieldID,
		"status": crmmodel.StatusEnabled,
	})
	if dataField == nil || dataField.FieldType != "group" {
		return []*crmmodel.FormField{field}
	}
	children := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"data_template_id": dataField.DataTemplateID,
		"parent_field_id":  dataField.ID,
		"status":           crmmodel.StatusEnabled,
	})
	result := make([]*crmmodel.FormField, 0, len(children))
	for _, child := range children {
		if child == nil || child.FieldType == "group" {
			continue
		}
		childField := *field
		childField.DataTemplateID = child.DataTemplateID
		childField.DataFieldID = child.ID
		childField.MainField = ""
		childField.Name = child.Name
		result = append(result, &childField)
	}
	return result
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
		skipMissingFields:          true,
	})
}

func collectWorkProgressFormInputForForm(ctx context.Context, formID uint64, values map[string]any) (*workFormInput, error) {
	return collectWorkFormInputByFormID(ctx, formID, values, workFormInputOptions{
		allowMissingRequiredFields: true,
		skipMissingFields:          true,
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
		leadFields:          map[string]any{},
		customerFields:      map[string]any{},
		assetFields:         map[string]any{},
		customerTagIDs:      []uint64{},
		leadDataRecords:     map[uint64]map[string]any{},
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
	for key, value := range source.leadFields {
		target.leadFields[key] = value
	}
	for key, value := range source.customerFields {
		target.customerFields[key] = value
	}
	for key, value := range source.assetFields {
		target.assetFields[key] = value
	}
	if source.customerTagsProvided {
		target.customerTagIDs = append([]uint64(nil), source.customerTagIDs...)
		target.customerTagsProvided = true
	}
	mergeWorkFormRecordMap(target.leadDataRecords, source.leadDataRecords)
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

func saveWorkFormInput(ctx context.Context, leadID uint64, customerID uint64, assetID uint64, formInput *workFormInput) error {
	if formInput == nil {
		return nil
	}
	if err := saveWorkLeadFormInput(ctx, leadID, formInput); err != nil {
		return err
	}
	if len(formInput.customerFields) > 0 {
		if customerID == 0 {
			return fmt.Errorf("客户不能为空")
		}
		formInput.customerFields["updated_at"] = time.Now()
		crmmodel.NewCustomerModel().Update(ctx, map[string]any{"id": customerID}, formInput.customerFields)
	}
	if formInput.customerTagsProvided {
		if customerID == 0 {
			return fmt.Errorf("客户不能为空")
		}
		needsSync, err := CustomerTagsNeedSync(ctx, customerID, formInput.customerTagIDs)
		if err != nil {
			return err
		}
		if needsSync {
			if _, err := SyncCustomerTags(ctx, customerID, formInput.customerTagIDs); err != nil {
				return err
			}
		}
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

func saveWorkLeadFormInput(ctx context.Context, leadID uint64, formInput *workFormInput) error {
	if formInput == nil || len(formInput.leadFields) == 0 && len(formInput.leadDataRecords) == 0 {
		return nil
	}
	if leadID == 0 {
		return fmt.Errorf("线索不能为空")
	}
	lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": leadID})
	if lead == nil {
		return fmt.Errorf("线索不存在")
	}
	updates := copyMap(formInput.leadFields)
	phone := lead.Phone
	wechat := lead.Wechat
	sourceID := lead.SourceID
	externalID := lead.ExternalID
	if value, exists := updates["phone"]; exists {
		phone = normalizeWorkLeadPhone(inputText(value))
		updates["phone"] = phone
	}
	if value, exists := updates["wechat"]; exists {
		wechat = inputText(value)
		updates["wechat"] = wechat
	}
	if phone == "" && wechat == "" {
		return fmt.Errorf("手机号和微信号至少填写一项")
	}
	if value, exists := updates["source_id"]; exists {
		sourceID = inputUint64(value)
		if sourceID == 0 || crmmodel.NewCustomerSourceModel().Find(ctx, map[string]any{
			"id": sourceID, "status": crmmodel.StatusEnabled,
		}) == nil {
			return fmt.Errorf("线索来源不存在或已停用")
		}
		updates["source_id"] = sourceID
	}
	if value, exists := updates["channel_id"]; exists {
		channelID := inputUint64(value)
		if channelID == 0 || crmmodel.NewCustomerChannelModel().Find(ctx, map[string]any{
			"id": channelID, "status": crmmodel.StatusEnabled,
		}) == nil {
			return fmt.Errorf("线索渠道不存在或已停用")
		}
		updates["channel_id"] = channelID
	}
	if value, exists := updates["external_id"]; exists {
		externalID = inputText(value)
		updates["external_id"] = externalID
	}
	if duplicate := findWorkLeadDuplicate(ctx, lead.ID, phone, wechat, sourceID, externalID); duplicate != nil {
		return fmt.Errorf("线索信息与现有记录重复：%s", duplicate.Reason)
	}
	if len(formInput.leadDataRecords) > 0 {
		record := mapFromAny(lead.RecordJSON)
		values := workLeadDataValues(lead)
		for _, fields := range formInput.leadDataRecords {
			for fieldID, value := range fields {
				values["data:"+fieldID] = value
			}
		}
		record["data_values"] = values
		updates["record_json"] = jsonText(record)
	}
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now()
	if crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": lead.ID}, updates) == 0 {
		return fmt.Errorf("线索信息保存失败")
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

func saveWorkFormDataRecords(ctx context.Context, ownership workDataOwnership, taskID uint64, operationID uint64, formInput *workFormInput) error {
	if formInput == nil {
		return nil
	}
	for templateID, record := range formInput.customerDataRecords {
		recordID := saveWorkDataRecord(ctx, workDataOwnership{CustomerID: ownership.CustomerID}, templateID, taskID, operationID, record)
		if recordID == 0 {
			return fmt.Errorf("客户资料保存失败")
		}
		if err := syncCustomerFollowFromForm(ctx, ownership, recordID, templateID, taskID, operationID, record); err != nil {
			return err
		}
	}
	if ownership.AssetID > 0 {
		for templateID, record := range formInput.assetDataRecords {
			saveWorkDataRecord(ctx, workDataOwnership{
				CustomerID: ownership.CustomerID,
				AssetID:    ownership.AssetID,
			}, templateID, taskID, operationID, record)
		}
	}
	return nil
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
	case "tags":
		return
	case "name", "phone", "wechat", "id_card", "gender", "remark":
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

func applyWorkLeadMainField(record map[string]any, field string, value any) {
	switch field {
	case "name", "phone", "wechat", "external_id", "city", "initial_need":
		record[field] = inputText(value)
	case "source_id", "channel_id":
		if id := inputUint64(value); id > 0 {
			record[field] = id
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

func isWorkLeadTemplateField(field *crmmodel.FormField) bool {
	return field != nil && field.DataTemplateCateID == crmmodel.LeadDataTemplateCateID
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

func workFormRecordBucket(_ context.Context, formInput *workFormInput, field *crmmodel.FormField) map[uint64]map[string]any {
	if field == nil {
		return formInput.customerDataRecords
	}
	switch field.DataTemplateCateID {
	case crmmodel.LeadDataTemplateCateID:
		return formInput.leadDataRecords
	case crmmodel.CustomerAssetDataTemplateCateID:
		return formInput.assetDataRecords
	default:
		return formInput.customerDataRecords
	}
}
