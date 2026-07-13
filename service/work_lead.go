package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

type workLeadDuplicate struct {
	LeadID     uint64
	CustomerID uint64
	Reason     string
}

func (WorkService) LeadPool(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	if !canManageWorkLeads(ctx, staff) {
		return map[string]any{
			"enabled":         false,
			"list":            []map[string]any{},
			"total":           0,
			"sources":         workLeadSourceOptions(ctx),
			"channels":        workLeadChannelOptions(ctx),
			"invalid_reasons": workLeadInvalidReasonOptions(ctx),
			"statuses":        workLeadStatusOptions(),
			"templates":       workLeadTemplateRows(ctx),
		}, nil
	}

	filter := map[string]any{}
	if !staff.CanDispatch {
		filter["owner_department_id"] = staff.DepartmentID
	}
	status := firstText(payload, "status")
	if status != "" && validWorkLeadStatus(status) {
		filter["status"] = status
	}
	leads := crmmodel.NewLeadModel().Select(ctx, filter, map[string]any{"order": "id desc"})
	keyword := firstText(payload, "keyword")
	rows := make([]map[string]any, 0, len(leads))
	for _, lead := range leads {
		if lead == nil || !matchesWorkLeadKeyword(lead, keyword) {
			continue
		}
		rows = append(rows, workLeadRow(ctx, lead))
	}

	page, pageSize, start, end := workLeadPageBounds(len(rows), payload)
	pageRows := []map[string]any{}
	if start < len(rows) {
		pageRows = rows[start:end]
	}
	return map[string]any{
		"enabled":         true,
		"list":            pageRows,
		"total":           len(rows),
		"page":            page,
		"page_size":       pageSize,
		"sources":         workLeadSourceOptions(ctx),
		"channels":        workLeadChannelOptions(ctx),
		"invalid_reasons": workLeadInvalidReasonOptions(ctx),
		"statuses":        workLeadStatusOptions(),
		"templates":       workLeadTemplateRows(ctx),
	}, nil
}

func (WorkService) RegisterLead(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if !canManageWorkLeads(ctx, staff) {
		return nil, fmt.Errorf("只有市场部门或流程调度员可以录入线索")
	}
	name := firstText(payload, "name")
	phone := normalizeWorkLeadPhone(firstText(payload, "phone", "mobile"))
	wechat := firstText(payload, "wechat")
	if name == "" {
		return nil, fmt.Errorf("请填写线索姓名")
	}
	if phone == "" && wechat == "" {
		return nil, fmt.Errorf("手机号和微信号至少填写一项")
	}
	sourceID := firstUint64(payload, "source_id", "sourceId")
	if sourceID == 0 {
		sourceID = crmmodel.DefaultCustomerSourceID
	}
	channelID := firstUint64(payload, "channel_id", "channelId")
	if channelID == 0 {
		channelID = crmmodel.DefaultCustomerChannelID
	}
	if crmmodel.NewCustomerSourceModel().Find(ctx, map[string]any{"id": sourceID, "status": crmmodel.StatusEnabled}) == nil {
		return nil, fmt.Errorf("线索来源不存在或已停用")
	}
	if crmmodel.NewCustomerChannelModel().Find(ctx, map[string]any{"id": channelID, "status": crmmodel.StatusEnabled}) == nil {
		return nil, fmt.Errorf("线索渠道不存在或已停用")
	}

	var created *crmmodel.Lead
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		code, err := crmmodel.GenerateUniqueLeadCode(txCtx)
		if err != nil {
			return err
		}
		externalID := firstText(payload, "external_id", "externalId")
		duplicate := findWorkLeadDuplicate(txCtx, 0, phone, wechat, sourceID, externalID)
		status := crmmodel.LeadStatusPending
		duplicateLeadID := uint64(0)
		duplicateCustomerID := uint64(0)
		duplicateReason := ""
		if duplicate != nil {
			status = crmmodel.LeadStatusDuplicate
			duplicateLeadID = duplicate.LeadID
			duplicateCustomerID = duplicate.CustomerID
			duplicateReason = duplicate.Reason
		}
		now := time.Now()
		dataValues := workLeadInputDataValues(txCtx, payload)
		leadID := uint64(crmmodel.NewLeadModel().Insert(txCtx, map[string]any{
			"code":                  code,
			"name":                  name,
			"phone":                 phone,
			"wechat":                wechat,
			"source_id":             sourceID,
			"channel_id":            channelID,
			"external_id":           externalID,
			"city":                  firstText(payload, "city"),
			"initial_need":          firstText(payload, "initial_need", "initialNeed", "need"),
			"status":                status,
			"duplicate_lead_id":     duplicateLeadID,
			"duplicate_customer_id": duplicateCustomerID,
			"duplicate_reason":      duplicateReason,
			"invalid_reason_id":     uint64(0),
			"invalid_note":          "",
			"customer_id":           uint64(0),
			"record_json":           jsonText(map[string]any{"data_values": dataValues}),
			"owner_department_id":   staff.DepartmentID,
			"owner_staff_id":        staff.ID,
			"created_by_staff_id":   staff.ID,
			"converted_by_staff_id": uint64(0),
			"created_at":            now,
			"updated_at":            now,
			"origin_task_id":        uint64(0),
			"origin_form_id":        uint64(0),
			"input_snapshot_json":   jsonText(payload),
		}))
		if leadID == 0 {
			return fmt.Errorf("线索录入失败")
		}
		created = crmmodel.NewLeadModel().Find(txCtx, map[string]any{"id": leadID})
		if created == nil {
			return fmt.Errorf("线索录入后无法读取")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"success": true,
		"lead":    workLeadRow(ctx, created),
	}, nil
}

func (WorkService) ActOnLead(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if !canManageWorkLeads(ctx, staff) {
		return nil, fmt.Errorf("只有市场部门或流程调度员可以处理线索")
	}
	leadID := firstUint64(payload, "lead_id", "leadId", "id")
	if leadID == 0 {
		return nil, fmt.Errorf("请选择线索")
	}
	action := firstText(payload, "action")
	var result map[string]any
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		lead := crmmodel.NewLeadModel().Find(txCtx, map[string]any{"id": leadID})
		if lead == nil || !canAccessWorkLead(staff, lead) {
			return fmt.Errorf("线索不存在或无权操作")
		}
		var err error
		switch action {
		case "invalid":
			err = invalidateWorkLead(txCtx, lead, payload)
		case "duplicate":
			err = markWorkLeadDuplicate(txCtx, lead)
		case "reopen":
			err = reopenWorkLead(txCtx, lead)
		case "convert":
			result, err = convertWorkLead(txCtx, staff, lead, payload)
		default:
			err = fmt.Errorf("不支持的线索操作")
		}
		if err != nil {
			return err
		}
		if result == nil {
			refreshed := crmmodel.NewLeadModel().Find(txCtx, map[string]any{"id": lead.ID})
			result = map[string]any{"success": true, "lead": workLeadRow(txCtx, refreshed)}
		}
		return nil
	})
	return result, err
}

func invalidateWorkLead(ctx context.Context, lead *crmmodel.Lead, payload map[string]any) error {
	if lead.Status == crmmodel.LeadStatusConverted {
		return fmt.Errorf("已转化线索不能判为无效")
	}
	reasonID := firstUint64(payload, "invalid_reason_id", "invalidReasonId", "reason_id", "reasonId")
	if reasonID == 0 || crmmodel.NewLeadInvalidReasonModel().Find(ctx, map[string]any{
		"id": reasonID, "status": crmmodel.StatusEnabled,
	}) == nil {
		return fmt.Errorf("请选择有效的无效原因")
	}
	if crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": lead.ID}, map[string]any{
		"status":            crmmodel.LeadStatusInvalid,
		"invalid_reason_id": reasonID,
		"invalid_note":      firstText(payload, "note", "invalid_note", "invalidNote"),
		"updated_at":        time.Now(),
	}) == 0 {
		return fmt.Errorf("线索状态已变化，请刷新后重试")
	}
	return nil
}

func markWorkLeadDuplicate(ctx context.Context, lead *crmmodel.Lead) error {
	if lead.Status == crmmodel.LeadStatusConverted {
		return fmt.Errorf("已转化线索不能标记为重复")
	}
	duplicate := findWorkLeadDuplicate(ctx, lead.ID, lead.Phone, lead.Wechat, lead.SourceID, lead.ExternalID)
	if duplicate == nil {
		return fmt.Errorf("没有发现可确认的重复线索或客户")
	}
	if crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": lead.ID}, map[string]any{
		"status":                crmmodel.LeadStatusDuplicate,
		"duplicate_lead_id":     duplicate.LeadID,
		"duplicate_customer_id": duplicate.CustomerID,
		"duplicate_reason":      duplicate.Reason,
		"updated_at":            time.Now(),
	}) == 0 {
		return fmt.Errorf("线索状态已变化，请刷新后重试")
	}
	return nil
}

func reopenWorkLead(ctx context.Context, lead *crmmodel.Lead) error {
	if lead.Status != crmmodel.LeadStatusInvalid && lead.Status != crmmodel.LeadStatusDuplicate {
		return fmt.Errorf("只有无效或重复线索可以恢复")
	}
	duplicate := findWorkLeadDuplicate(ctx, lead.ID, lead.Phone, lead.Wechat, lead.SourceID, lead.ExternalID)
	status := crmmodel.LeadStatusPending
	duplicateLeadID := uint64(0)
	duplicateCustomerID := uint64(0)
	duplicateReason := ""
	if duplicate != nil {
		status = crmmodel.LeadStatusDuplicate
		duplicateLeadID = duplicate.LeadID
		duplicateCustomerID = duplicate.CustomerID
		duplicateReason = duplicate.Reason
	}
	if crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": lead.ID}, map[string]any{
		"status":                status,
		"duplicate_lead_id":     duplicateLeadID,
		"duplicate_customer_id": duplicateCustomerID,
		"duplicate_reason":      duplicateReason,
		"invalid_reason_id":     uint64(0),
		"invalid_note":          "",
		"updated_at":            time.Now(),
	}) == 0 {
		return fmt.Errorf("线索状态已变化，请刷新后重试")
	}
	return nil
}

func convertWorkLead(ctx context.Context, staff *WorkStaffSession, lead *crmmodel.Lead, payload map[string]any) (map[string]any, error) {
	if lead.Status == crmmodel.LeadStatusConverted && lead.CustomerID > 0 {
		asset := crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{"customer_id": lead.CustomerID})
		assetID := uint64(0)
		if asset != nil {
			assetID = asset.ID
		}
		return map[string]any{
			"success":     true,
			"converted":   true,
			"customer_id": lead.CustomerID,
			"asset_id":    assetID,
			"lead":        workLeadRow(ctx, lead),
		}, nil
	}
	if lead.Status != crmmodel.LeadStatusPending {
		return nil, fmt.Errorf("只有待处理线索可以转为客户")
	}
	if duplicate := findWorkLeadDuplicate(ctx, lead.ID, lead.Phone, lead.Wechat, lead.SourceID, lead.ExternalID); duplicate != nil {
		crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": lead.ID}, map[string]any{
			"status":                crmmodel.LeadStatusDuplicate,
			"duplicate_lead_id":     duplicate.LeadID,
			"duplicate_customer_id": duplicate.CustomerID,
			"duplicate_reason":      duplicate.Reason,
			"updated_at":            time.Now(),
		})
		return map[string]any{
			"success":   true,
			"converted": false,
			"duplicate": true,
			"message":   duplicate.Reason,
			"lead":      workLeadRow(ctx, crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": lead.ID})),
		}, nil
	}

	customerCode, err := crmmodel.GenerateUniqueCustomerCode(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	customerRecord := defaultWorkCustomerRecord(staff)
	customerRecord["code"] = customerCode
	customerRecord["name"] = lead.Name
	customerRecord["phone"] = lead.Phone
	customerRecord["wechat"] = lead.Wechat
	customerRecord["source_id"] = lead.SourceID
	customerRecord["channel_id"] = lead.ChannelID
	customerRecord["remark"] = lead.InitialNeed
	customerID := uint64(crmmodel.NewCustomerModel().Insert(ctx, customerRecord))
	if customerID == 0 {
		return nil, fmt.Errorf("线索转客户失败")
	}

	assetInput := emptyWorkFormInput()
	assetInput.assetFields["asset_name"] = lead.Name + "-待补充资产"
	assetInput.assetFields["remark"] = "线索转化自动建档，待后续补充资产资料"
	assetID, err := createWorkCustomerAsset(ctx, customerID, assetInput)
	if err != nil || assetID == 0 {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("线索转化后创建资产失败")
	}
	ownerStaffID := firstUint64(payload, "owner_staff_id", "ownerStaffId")
	if err := startAssetWorkflow(ctx, customerID, assetID, ownerStaffID); err != nil {
		return nil, err
	}

	if crmmodel.NewLeadModel().Update(ctx, map[string]any{
		"id": lead.ID, "status": crmmodel.LeadStatusPending,
	}, map[string]any{
		"status":                crmmodel.LeadStatusConverted,
		"customer_id":           customerID,
		"converted_by_staff_id": staff.ID,
		"converted_at":          now,
		"duplicate_lead_id":     uint64(0),
		"duplicate_customer_id": uint64(0),
		"duplicate_reason":      "",
		"invalid_reason_id":     uint64(0),
		"invalid_note":          "",
		"updated_at":            now,
	}) == 0 {
		return nil, fmt.Errorf("线索状态已变化，请刷新后重试")
	}
	if progress := currentWorkEntryInstance(ctx, customerID, assetID); progress != nil {
		recordWorkManagementOperation(ctx, staff, progress, "lead_converted", "线索已转为客户", lead.Code, map[string]any{
			"lead_id":     lead.ID,
			"customer_id": customerID,
			"asset_id":    assetID,
		})
	}
	refreshed := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": lead.ID})
	return map[string]any{
		"success":     true,
		"converted":   true,
		"customer_id": customerID,
		"asset_id":    assetID,
		"lead":        workLeadRow(ctx, refreshed),
	}, nil
}

func canManageWorkLeads(ctx context.Context, staff *WorkStaffSession) bool {
	if staff == nil || staff.ID == 0 {
		return false
	}
	if staff.CanDispatch {
		return true
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{
		"id": staff.DepartmentID, "status": crmmodel.StatusEnabled,
	})
	return department != nil && strings.EqualFold(strings.TrimSpace(department.Code), "MKT")
}

func canAccessWorkLead(staff *WorkStaffSession, lead *crmmodel.Lead) bool {
	if staff == nil || lead == nil {
		return false
	}
	return staff.CanDispatch || lead.OwnerDepartmentID == staff.DepartmentID
}

func findWorkLeadDuplicate(ctx context.Context, leadID uint64, phone, wechat string, sourceID uint64, externalID string) *workLeadDuplicate {
	customerModel := crmmodel.NewCustomerModel()
	if phone != "" {
		if customer := customerModel.Find(ctx, map[string]any{"phone": phone}); customer != nil {
			return &workLeadDuplicate{CustomerID: customer.ID, Reason: "手机号已存在于客户库：" + customer.Code}
		}
	}
	if wechat != "" {
		if customer := customerModel.Find(ctx, map[string]any{"wechat": wechat}); customer != nil {
			return &workLeadDuplicate{CustomerID: customer.ID, Reason: "微信号已存在于客户库：" + customer.Code}
		}
	}

	leadModel := crmmodel.NewLeadModel()
	for _, candidate := range leadModel.Select(ctx, map[string]any{}, map[string]any{"order": "id asc"}) {
		if candidate == nil || candidate.ID == leadID || candidate.Status == crmmodel.LeadStatusInvalid {
			continue
		}
		reason := ""
		switch {
		case phone != "" && candidate.Phone == phone:
			reason = "手机号已存在于线索池：" + candidate.Code
		case wechat != "" && candidate.Wechat == wechat:
			reason = "微信号已存在于线索池：" + candidate.Code
		case externalID != "" && candidate.SourceID == sourceID && candidate.ExternalID == externalID:
			reason = "同一来源的外部线索ID已存在：" + candidate.Code
		}
		if reason == "" {
			continue
		}
		duplicateLeadID := candidate.ID
		if candidate.DuplicateLeadID > 0 {
			duplicateLeadID = candidate.DuplicateLeadID
		}
		return &workLeadDuplicate{LeadID: duplicateLeadID, CustomerID: candidate.CustomerID, Reason: reason}
	}
	return nil
}

func normalizeWorkLeadPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	return phone
}

func matchesWorkLeadKeyword(lead *crmmodel.Lead, keyword string) bool {
	if keyword == "" {
		return true
	}
	return containsFold(lead.Code, keyword) ||
		containsFold(lead.Name, keyword) ||
		containsFold(lead.Phone, keyword) ||
		containsFold(lead.Wechat, keyword) ||
		containsFold(lead.City, keyword) ||
		containsFold(lead.InitialNeed, keyword)
}

func workLeadPageBounds(total int, payload map[string]any) (int, int, int, int) {
	page := inputInt(payload["page"])
	if page <= 0 {
		page = 1
	}
	pageSize := inputInt(firstPresent(payload, "page_size", "pageSize", "limit"))
	if pageSize <= 0 {
		pageSize = 30
	}
	if pageSize > 100 {
		pageSize = 100
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return page, pageSize, start, end
}

func validWorkLeadStatus(status string) bool {
	switch status {
	case crmmodel.LeadStatusPending, crmmodel.LeadStatusInvalid, crmmodel.LeadStatusDuplicate, crmmodel.LeadStatusConverted:
		return true
	default:
		return false
	}
}

func workLeadRow(ctx context.Context, lead *crmmodel.Lead) map[string]any {
	if lead == nil {
		return map[string]any{}
	}
	row := map[string]any{
		"id":                    lead.ID,
		"code":                  lead.Code,
		"name":                  lead.Name,
		"phone":                 lead.Phone,
		"wechat":                lead.Wechat,
		"source_id":             lead.SourceID,
		"channel_id":            lead.ChannelID,
		"external_id":           lead.ExternalID,
		"city":                  lead.City,
		"initial_need":          lead.InitialNeed,
		"status":                lead.Status,
		"status_name":           crmmodel.LeadStatusName(lead.Status),
		"duplicate_lead_id":     lead.DuplicateLeadID,
		"duplicate_customer_id": lead.DuplicateCustomerID,
		"duplicate_reason":      workLeadDuplicateReasonForDisplay(ctx, lead),
		"invalid_reason_id":     lead.InvalidReasonID,
		"invalid_note":          lead.InvalidNote,
		"customer_id":           lead.CustomerID,
		"owner_department_id":   lead.OwnerDepartmentID,
		"owner_staff_id":        lead.OwnerStaffID,
		"converted_by_staff_id": lead.ConvertedByStaffID,
		"converted_at":          lead.ConvertedAt,
		"created_at":            lead.CreatedAt,
		"updated_at":            lead.UpdatedAt,
		"data_values":           workLeadDataValues(lead),
	}
	if source := crmmodel.NewCustomerSourceModel().Find(ctx, map[string]any{"id": lead.SourceID}); source != nil {
		row["source_name"] = source.Name
	}
	if channel := crmmodel.NewCustomerChannelModel().Find(ctx, map[string]any{"id": lead.ChannelID}); channel != nil {
		row["channel_name"] = channel.Name
	}
	if reason := crmmodel.NewLeadInvalidReasonModel().Find(ctx, map[string]any{"id": lead.InvalidReasonID}); reason != nil {
		row["invalid_reason_name"] = reason.Name
	}
	if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": lead.CustomerID}); customer != nil {
		row["customer_code"] = customer.Code
		row["customer_code_display"] = customerCodeDisplayForWork(ctx, customer.Code)
		row["customer_name"] = customer.Name
	}
	if duplicateLead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": lead.DuplicateLeadID}); duplicateLead != nil {
		row["duplicate_lead_code"] = duplicateLead.Code
		row["duplicate_lead_name"] = duplicateLead.Name
	}
	if owner := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": lead.OwnerStaffID}); owner != nil {
		row["owner_staff_name"] = owner.Name
	}
	return row
}

func workLeadDuplicateReasonForDisplay(ctx context.Context, lead *crmmodel.Lead) string {
	if lead == nil {
		return ""
	}
	reason := strings.TrimSpace(lead.DuplicateReason)
	if lead.DuplicateCustomerID == 0 || reason == "" {
		return reason
	}
	customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": lead.DuplicateCustomerID})
	if customer == nil {
		return reason
	}
	code := customerCodeDisplayForWork(ctx, customer.Code)
	if code == "" {
		return reason
	}
	switch {
	case strings.HasPrefix(reason, "手机号已存在于客户库："):
		return "手机号已存在于客户库：" + code
	case strings.HasPrefix(reason, "微信号已存在于客户库："):
		return "微信号已存在于客户库：" + code
	default:
		return reason
	}
}

func workLeadTemplateRows(ctx context.Context) []map[string]any {
	templates := crmmodel.NewDataTemplateModel().Select(ctx, map[string]any{
		"cate_id": crmmodel.LeadDataTemplateCateID,
		"status":  crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
	rows := make([]map[string]any, 0, len(templates))
	for _, template := range templates {
		if template == nil {
			continue
		}
		fields := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
			"data_template_id": template.ID,
			"status":           crmmodel.StatusEnabled,
		}, map[string]any{"order": "sort asc,id asc"})
		fieldRows := make([]map[string]any, 0, len(fields))
		parentNames := workDataCompletenessParentNames(ctx, fields)
		for _, field := range fields {
			if field == nil || field.FieldType == "group" || workIsAttachmentFieldType(field.FieldType) {
				continue
			}
			fieldRows = append(fieldRows, map[string]any{
				"id":            field.ID,
				"name":          field.Name,
				"field_key":     field.FieldKey,
				"field_type":    field.FieldType,
				"default_value": field.DefaultValue,
				"group_name":    parentNames[field.ParentFieldID],
				"sort":          field.Sort,
				"options":       workDataFieldOptionsForField(ctx, field),
			})
		}
		if len(fieldRows) == 0 {
			continue
		}
		rows = append(rows, map[string]any{
			"id":     template.ID,
			"name":   template.Name,
			"sort":   template.Sort,
			"fields": fieldRows,
		})
	}
	return rows
}

func workLeadInputDataValues(ctx context.Context, payload map[string]any) map[string]any {
	input := mapFromAny(firstPresent(payload, "data_values", "dataValues"))
	if len(input) == 0 {
		return map[string]any{}
	}
	result := map[string]any{}
	for _, template := range workLeadTemplateRows(ctx) {
		for _, fieldRow := range mapListFromAny(template["fields"]) {
			fieldID := inputUint64(fieldRow["id"])
			if fieldID == 0 {
				continue
			}
			key := fmt.Sprintf("data:%d", fieldID)
			value, exists := input[key]
			if !exists || emptyWorkFieldValue(value) {
				continue
			}
			if normalized, ok := normalizeWorkLeadFieldValue(fieldRow, value); ok {
				result[key] = normalized
			}
		}
	}
	return result
}

func normalizeWorkLeadFieldValue(field map[string]any, value any) (any, bool) {
	fieldType := inputText(field["field_type"])
	switch fieldType {
	case "checkbox", "multi_select":
		values := stringListFromAny(value)
		return values, workLeadOptionValuesAllowed(field, values)
	case "select", "radio":
		text := inputText(value)
		return text, text != "" && workLeadOptionValuesAllowed(field, []string{text})
	case "boolean":
		switch value.(type) {
		case bool:
			return value, true
		default:
			text := strings.ToLower(inputText(value))
			if text == "true" || text == "1" {
				return true, true
			}
			if text == "false" || text == "0" {
				return false, true
			}
			return nil, false
		}
	default:
		text := inputText(value)
		return text, text != ""
	}
}

func workLeadOptionValuesAllowed(field map[string]any, values []string) bool {
	if len(values) == 0 {
		return false
	}
	allowed := map[string]bool{}
	for _, option := range mapListFromAny(field["options"]) {
		value := inputText(option["value"])
		if value != "" {
			allowed[value] = true
		}
	}
	if len(allowed) == 0 {
		return false
	}
	for _, value := range values {
		if !allowed[value] {
			return false
		}
	}
	return true
}

func workLeadDataValues(lead *crmmodel.Lead) map[string]any {
	if lead == nil {
		return map[string]any{}
	}
	record := mapFromAny(lead.RecordJSON)
	if values := mapFromAny(record["data_values"]); len(values) > 0 {
		return values
	}
	values := map[string]any{}
	for key, value := range record {
		if strings.HasPrefix(key, "data:") {
			values[key] = value
		}
	}
	return values
}

func workLeadSourceOptions(ctx context.Context) []map[string]any {
	return workLeadNamedOptions(crmmodel.NewCustomerSourceModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}))
}

func workLeadChannelOptions(ctx context.Context) []map[string]any {
	return workLeadNamedOptions(crmmodel.NewCustomerChannelModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}))
}

func workLeadInvalidReasonOptions(ctx context.Context) []map[string]any {
	return workLeadNamedOptions(crmmodel.NewLeadInvalidReasonModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}))
}

func workLeadNamedOptions(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{"id": row["id"], "name": row["name"]})
	}
	return result
}

func workLeadStatusOptions() []map[string]any {
	statuses := []string{
		crmmodel.LeadStatusPending,
		crmmodel.LeadStatusDuplicate,
		crmmodel.LeadStatusInvalid,
		crmmodel.LeadStatusConverted,
	}
	result := make([]map[string]any, 0, len(statuses))
	for _, status := range statuses {
		result = append(result, map[string]any{"id": status, "name": crmmodel.LeadStatusName(status)})
	}
	return result
}
