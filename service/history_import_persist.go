package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	uploadservice "github.com/dever-package/front/service/upload"
	uploadrepo "github.com/dever-package/front/service/upload/repository"

	crmmodel "github.com/dever-package/crm/model"
)

func persistHistoryImportCase(
	ctx context.Context,
	catalog historyImportCatalog,
	plan historyImportCasePlan,
	options HistoryImportOptions,
) (historyImportTarget, HistoryImportCounts, []HistoryImportIssue, error) {
	target := plan.Target
	counts := HistoryImportCounts{}
	issues := make([]HistoryImportIssue, 0)
	var err error

	target.LeadID, err = persistHistoryLead(ctx, catalog, plan.Input, target.LeadID)
	if err != nil {
		return target, counts, issues, err
	}
	if target.LeadID > 0 {
		counts.Leads = 1
	}
	target.CustomerID, err = persistHistoryCustomer(ctx, catalog, plan.Input, target.CustomerID)
	if err != nil {
		return target, counts, issues, err
	}
	if target.CustomerID > 0 {
		counts.Customers = 1
	}
	if target.LeadID > 0 && target.CustomerID > 0 {
		linkHistoryLeadCustomer(ctx, target.LeadID, target.CustomerID, plan.Input)
	}

	if target.AssetIDs == nil {
		target.AssetIDs = map[string]uint64{}
	}
	for _, assetInput := range plan.Input.Assets {
		assetID, persistErr := persistHistoryAsset(ctx, target.CustomerID, assetInput, target.AssetIDs[assetInput.Key])
		if persistErr != nil {
			return target, counts, issues, persistErr
		}
		target.AssetIDs[assetInput.Key] = assetID
		if assetID > 0 {
			counts.Assets++
		}
	}
	primaryAssetID := historyPrimaryAssetID(plan.Input.Assets, target.AssetIDs)

	target.WorkflowInstanceID, err = persistHistoryWorkflow(
		ctx, catalog, plan.Input, target.CustomerID, primaryAssetID, target.WorkflowInstanceID, options,
	)
	if err != nil {
		return target, counts, issues, err
	}

	if target.CustomerID > 0 {
		for _, record := range plan.Input.CustomerRecords {
			_, changed, recordIssues, persistErr := persistHistoryDataRecord(ctx, catalog, workDataOwnership{
				CustomerID: target.CustomerID,
			}, record)
			issues = append(issues, recordIssues...)
			if persistErr != nil {
				return target, counts, issues, persistErr
			}
			if changed {
				counts.Records++
			}
		}
	}
	for _, assetInput := range plan.Input.Assets {
		assetID := target.AssetIDs[assetInput.Key]
		dataRecordIDs := map[string]uint64{}
		for _, record := range assetInput.Records {
			dataRecordID, changed, recordIssues, persistErr := persistHistoryDataRecord(ctx, catalog, workDataOwnership{
				CustomerID: target.CustomerID,
				AssetID:    assetID,
			}, record)
			issues = append(issues, recordIssues...)
			if persistErr != nil {
				return target, counts, issues, persistErr
			}
			if changed {
				counts.Records++
			}
			dataRecordIDs[historyCatalogKey(record.TemplateName)] = dataRecordID
		}
		for _, attachment := range assetInput.Attachments {
			dataRecordID := dataRecordIDs[historyCatalogKey(attachment.TemplateName)]
			attachmentIDs, attachmentIssues := persistHistoryAttachments(
				ctx, []HistoryImportAttachmentInput{attachment}, target.CustomerID, assetID, 0, dataRecordID, options.UploadRuleID,
			)
			target.AttachmentIDs = append(target.AttachmentIDs, attachmentIDs...)
			issues = append(issues, attachmentIssues...)
			counts.Attachments += len(attachmentIDs)
		}
	}

	for _, operation := range plan.Input.Operations {
		operationID, created := persistHistoryOperation(
			ctx, catalog, plan.Input.CaseID, operation, target.CustomerID, primaryAssetID, target.WorkflowInstanceID,
		)
		if operationID > 0 {
			target.OperationIDs = append(target.OperationIDs, operationID)
		}
		if created {
			counts.Operations++
		}
	}
	for _, meeting := range plan.Input.Meetings {
		meetingID, created, meetingIssues := persistHistoryMeeting(
			ctx, catalog, plan.Input.CaseID, meeting, target.CustomerID, primaryAssetID,
			target.WorkflowInstanceID, options.UploadRuleID,
		)
		issues = append(issues, meetingIssues...)
		if meetingID > 0 {
			target.MeetingIDs = append(target.MeetingIDs, meetingID)
		}
		if created {
			counts.Meetings++
		}
	}
	for _, group := range plan.Input.Groups {
		groupID, created, groupIssues := persistHistoryGroup(
			ctx, catalog, plan.Input.CaseID, group, target.CustomerID, primaryAssetID, target.WorkflowInstanceID,
		)
		issues = append(issues, groupIssues...)
		if groupID > 0 {
			target.GroupIDs = append(target.GroupIDs, groupID)
		}
		if created {
			counts.Groups++
		}
	}
	return target, counts, issues, nil
}

func persistHistoryLead(
	ctx context.Context,
	catalog historyImportCatalog,
	input HistoryImportCaseInput,
	leadID uint64,
) (uint64, error) {
	if input.Lead == nil {
		return leadID, nil
	}
	source := catalog.SourcesByCode[historyCatalogKey(input.Lead.SourceCode)]
	channel := catalog.ChannelsByCode[historyCatalogKey(input.Lead.ChannelCode)]
	if source == nil || channel == nil {
		return 0, fmt.Errorf("案件%s的来源或渠道未配置", input.CaseID)
	}
	owner, _ := resolveHistoryStaff(catalog, input.Lead.Owner, false, input.CaseID, "")
	if leadID > 0 {
		lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": leadID})
		if lead == nil {
			return 0, fmt.Errorf("案件%s关联线索不存在", input.CaseID)
		}
		updates := map[string]any{}
		mergeHistoryString(updates, "name", lead.Name, input.Lead.Name)
		mergeHistoryString(updates, "phone", lead.Phone, input.Lead.Phone)
		mergeHistoryString(updates, "wechat", lead.Wechat, input.Lead.Wechat)
		mergeHistoryString(updates, "external_id", lead.ExternalID, input.Lead.ExternalID)
		mergeHistoryString(updates, "city", lead.City, input.Lead.City)
		mergeHistoryString(updates, "initial_need", lead.InitialNeed, input.Lead.InitialNeed)
		if len(updates) > 0 {
			updates["updated_at"] = time.Now()
			crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": lead.ID}, updates)
		}
		return lead.ID, nil
	}
	code, err := crmmodel.GenerateUniqueLeadCode(ctx)
	if err != nil {
		return 0, err
	}
	createdAt := historyInputTime(input.Lead.CreatedAt)
	ownerStaffID, ownerDepartmentID := historyStaffOwnership(owner)
	leadID = uint64(crmmodel.NewLeadModel().Insert(ctx, map[string]any{
		"code": code, "name": historyRequiredName(input.Lead.Name), "phone": strings.TrimSpace(input.Lead.Phone),
		"wechat": strings.TrimSpace(input.Lead.Wechat), "source_id": source.ID, "channel_id": channel.ID,
		"external_id": strings.TrimSpace(input.Lead.ExternalID), "city": strings.TrimSpace(input.Lead.City),
		"initial_need": strings.TrimSpace(input.Lead.InitialNeed), "status": crmmodel.LeadStatusPending,
		"record_json":         jsonText(map[string]any{"history_case_id": input.CaseID}),
		"owner_department_id": ownerDepartmentID, "owner_staff_id": ownerStaffID,
		"created_by_staff_id": ownerStaffID, "input_snapshot_json": jsonText(map[string]any{"history_case_id": input.CaseID}),
		"created_at": createdAt, "updated_at": createdAt,
	}))
	if leadID == 0 {
		return 0, fmt.Errorf("案件%s创建历史线索失败", input.CaseID)
	}
	return leadID, nil
}

func persistHistoryCustomer(
	ctx context.Context,
	catalog historyImportCatalog,
	input HistoryImportCaseInput,
	customerID uint64,
) (uint64, error) {
	if input.Customer == nil {
		return customerID, nil
	}
	if customerID > 0 {
		customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
		if customer == nil {
			return 0, fmt.Errorf("案件%s关联客户不存在", input.CaseID)
		}
		updates := map[string]any{}
		mergeHistoryString(updates, "name", customer.Name, input.Customer.Name)
		mergeHistoryString(updates, "phone", customer.Phone, input.Customer.Phone)
		mergeHistoryString(updates, "wechat", customer.Wechat, input.Customer.Wechat)
		mergeHistoryString(updates, "id_card", customer.IDCard, input.Customer.IDCard)
		mergeHistoryString(updates, "remark", customer.Remark, input.Customer.Remark)
		if len(updates) > 0 {
			updates["updated_at"] = time.Now()
			crmmodel.NewCustomerModel().Update(ctx, map[string]any{"id": customer.ID}, updates)
		}
		return customer.ID, nil
	}
	source := catalog.SourcesByCode[historyCatalogKey(input.Customer.SourceCode)]
	channel := catalog.ChannelsByCode[historyCatalogKey(input.Customer.ChannelCode)]
	if source == nil || channel == nil {
		return 0, fmt.Errorf("案件%s的客户来源或渠道未配置", input.CaseID)
	}
	code, err := crmmodel.GenerateUniqueCustomerCode(ctx)
	if err != nil {
		return 0, err
	}
	createdAt := historyInputTime(input.Customer.CreatedAt)
	customerID = uint64(crmmodel.NewCustomerModel().Insert(ctx, map[string]any{
		"code": code, "name": historyRequiredName(input.Customer.Name), "phone": strings.TrimSpace(input.Customer.Phone),
		"wechat": strings.TrimSpace(input.Customer.Wechat), "id_card": strings.TrimSpace(input.Customer.IDCard),
		"gender": "unknown", "source_id": source.ID, "channel_id": channel.ID,
		"level_id": uint64(0), "tags": "", "remark": strings.TrimSpace(input.Customer.Remark),
		"created_by_staff_id": uint64(0), "created_at": createdAt, "updated_at": createdAt,
	}))
	if customerID == 0 {
		return 0, fmt.Errorf("案件%s创建历史客户失败", input.CaseID)
	}
	return customerID, nil
}

func linkHistoryLeadCustomer(ctx context.Context, leadID uint64, customerID uint64, input HistoryImportCaseInput) {
	lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": leadID})
	if lead == nil || lead.CustomerID > 0 {
		return
	}
	convertedAt := time.Now()
	if input.Workflow.EndedAt != nil {
		convertedAt = *input.Workflow.EndedAt
	}
	crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": leadID}, map[string]any{
		"status": crmmodel.LeadStatusConverted, "customer_id": customerID,
		"converted_at": convertedAt, "updated_at": time.Now(),
	})
}

func persistHistoryAsset(
	ctx context.Context,
	customerID uint64,
	input HistoryImportAssetInput,
	assetID uint64,
) (uint64, error) {
	if customerID == 0 {
		return 0, fmt.Errorf("历史资产缺少客户归属")
	}
	if assetID > 0 {
		asset := crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{"id": assetID, "customer_id": customerID})
		if asset == nil {
			return 0, fmt.Errorf("历史资产目标不存在或不属于客户")
		}
		updates := map[string]any{}
		mergeHistoryString(updates, "asset_name", asset.AssetName, input.Name)
		mergeHistoryString(updates, "remark", asset.Remark, input.Remark)
		if len(updates) > 0 {
			updates["updated_at"] = time.Now()
			crmmodel.NewCustomerAssetModel().Update(ctx, map[string]any{"id": asset.ID}, updates)
		}
		return asset.ID, nil
	}
	customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
	if customer == nil {
		return 0, fmt.Errorf("历史资产所属客户不存在")
	}
	assets := crmmodel.NewCustomerAssetModel().Select(ctx, map[string]any{"customer_id": customerID})
	seq := uint64(len(assets) + 1)
	assetNo := historyUniqueAssetNo(ctx, customer.Code, seq)
	now := time.Now()
	assetID = uint64(crmmodel.NewCustomerAssetModel().Insert(ctx, map[string]any{
		"asset_no": assetNo, "asset_name": historyRequiredAssetName(input.Name, customer.Name),
		"asset_seq": seq, "customer_id": customerID, "asset_status_id": crmmodel.DefaultAssetStatusID,
		"remark": strings.TrimSpace(input.Remark), "created_at": now, "updated_at": now,
	}))
	if assetID == 0 {
		return 0, fmt.Errorf("创建历史客户资产失败")
	}
	return assetID, nil
}

func historyUniqueAssetNo(ctx context.Context, customerCode string, seq uint64) string {
	base := strings.TrimSpace(customerCode)
	if base == "" {
		base = fmt.Sprintf("HISTORY-%d", time.Now().UnixNano())
	}
	for offset := uint64(0); offset < 200; offset++ {
		candidate := fmt.Sprintf("%s-%d", base, seq+offset)
		if crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{"asset_no": candidate}) == nil {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

func persistHistoryWorkflow(
	ctx context.Context,
	catalog historyImportCatalog,
	input HistoryImportCaseInput,
	customerID uint64,
	assetID uint64,
	instanceID uint64,
	options HistoryImportOptions,
) (uint64, error) {
	if customerID == 0 || assetID == 0 {
		return instanceID, nil
	}
	if instanceID > 0 {
		if crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": instanceID}) == nil {
			return 0, fmt.Errorf("案件%s关联流程实例不存在", input.CaseID)
		}
		return instanceID, nil
	}
	workflow, stage := resolveHistoryWorkflowStage(catalog, input.Workflow.StageName)
	if workflow == nil || stage == nil {
		return 0, fmt.Errorf("案件%s无法解析签约流程阶段：%s", input.CaseID, input.Workflow.StageName)
	}
	if existing := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
		"customer_id": customerID, "asset_id": assetID, "workflow_id": workflow.ID,
	}, map[string]any{"order": "id desc"}); existing != nil {
		return existing.ID, nil
	}
	owner, _ := resolveHistoryStaff(catalog, input.Workflow.Owner, options.RestoreActiveWorkflows, input.CaseID, "")
	ownerID, departmentID := historyStaffOwnership(owner)
	if departmentID == 0 {
		departmentID = stage.OwnerDepartmentID
	}
	status := input.Workflow.Status
	if status != crmmodel.ProgressStatusCompleted && status != crmmodel.ProgressStatusTerminated {
		status = crmmodel.ProgressStatusActive
	}
	startedAt := historyInputTime(input.Workflow.StartedAt)
	data := map[string]any{
		"lead_id": uint64(0), "customer_id": customerID, "asset_id": assetID,
		"customer_product_id": uint64(0), "workflow_id": workflow.ID, "stage_id": stage.ID,
		"owner_department_id": departmentID, "owner_staff_id": ownerID, "status": status,
		"started_at": startedAt, "terminated_reason": strings.TrimSpace(input.Workflow.Reason),
		"updated_at": historyInputTime(input.Workflow.EndedAt),
	}
	if status == crmmodel.ProgressStatusCompleted {
		data["completed_at"] = historyInputTime(input.Workflow.EndedAt)
	}
	if status == crmmodel.ProgressStatusTerminated {
		data["terminated_at"] = historyInputTime(input.Workflow.EndedAt)
	}
	instanceID = uint64(crmmodel.NewWorkflowInstanceModel().Insert(ctx, data))
	if instanceID == 0 {
		return 0, fmt.Errorf("案件%s创建历史流程失败", input.CaseID)
	}
	if status == crmmodel.ProgressStatusActive && options.RestoreActiveWorkflows {
		instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": instanceID})
		if instance == nil || owner == nil || owner.Status != crmmodel.StatusEnabled {
			return 0, fmt.Errorf("案件%s无法恢复活动流程负责人", input.CaseID)
		}
		if err := createStageTodos(ctx, instance, stage); err != nil {
			return 0, err
		}
	}
	return instanceID, nil
}

func persistHistoryDataRecord(
	ctx context.Context,
	catalog historyImportCatalog,
	ownership workDataOwnership,
	input HistoryImportDataRecordInput,
) (uint64, bool, []HistoryImportIssue, error) {
	entry, exists := catalog.Templates[historyCatalogKey(input.TemplateName)]
	if !exists || entry.Template == nil {
		return 0, false, nil, fmt.Errorf("目标数据模板不存在：%s", input.TemplateName)
	}
	converted := map[string]any{}
	issues := make([]HistoryImportIssue, 0)
	for name, value := range input.Fields {
		if historyValueEmpty(value) {
			continue
		}
		field := resolveHistoryDataField(entry, input.TemplateName, name)
		if field == nil {
			return 0, false, issues, fmt.Errorf("目标字段不存在：%s/%s", input.TemplateName, name)
		}
		convertedValue, handled, ok := historySpecialFieldValue(catalog, input.TemplateName, name, value)
		if !handled {
			convertedValue, ok = historyTargetFieldValue(entry, field, value)
		}
		if !ok {
			issues = append(issues, HistoryImportIssue{
				Severity: HistoryImportSeverityWarning, Code: "option_unresolved", Field: name,
				Message: fmt.Sprintf("目标选项无法解析：%s/%s", input.TemplateName, name),
			})
			continue
		}
		converted[field.FieldKey] = convertedValue
	}
	model := crmmodel.NewDataRecordModel()
	existing := model.Find(ctx, workDataRecordOwnershipFilter(ownership, entry.Template.ID))
	now := time.Now()
	if existing == nil {
		if len(converted) == 0 {
			return 0, false, issues, nil
		}
		id := uint64(model.Insert(ctx, map[string]any{
			"customer_id": ownership.CustomerID, "asset_id": ownership.AssetID,
			"workflow_instance_id": ownership.WorkflowInstanceID, "customer_product_id": ownership.CustomerProductID,
			"data_template_id": entry.Template.ID, "task_id": uint64(0), "operation_log_id": uint64(0),
			"record_json": jsonText(converted), "summary": "", "status": crmmodel.StatusEnabled,
			"sort": 100, "created_at": now, "updated_at": now,
		}))
		if id == 0 {
			return 0, false, issues, fmt.Errorf("写入数据记录失败：%s", input.TemplateName)
		}
		return id, true, issues, nil
	}
	current := mapFromAny(existing.RecordJSON)
	changed := false
	for key, value := range converted {
		if historyValueEmpty(current[key]) {
			current[key] = value
			changed = true
			continue
		}
		if !historyValuesEqual(current[key], value) {
			issues = append(issues, HistoryImportIssue{
				Severity: HistoryImportSeverityWarning, Code: "current_value_kept", Field: key,
				Message: fmt.Sprintf("当前CRM已有非空值，保留当前值：%s/%s", input.TemplateName, key),
			})
		}
	}
	if changed {
		model.Update(ctx, map[string]any{"id": existing.ID}, map[string]any{
			"record_json": jsonText(current), "updated_at": now,
		})
	}
	return existing.ID, changed, issues, nil
}

func historySpecialFieldValue(
	catalog historyImportCatalog,
	templateName string,
	fieldName string,
	value any,
) (any, bool, bool) {
	if historyCatalogKey(templateName) != historyCatalogKey("邀约到访") ||
		historyCatalogKey(fieldName) != historyCatalogKey("预约会议室") {
		return nil, false, false
	}
	matches := catalog.ResourcesByName[historyResourceKey(fmt.Sprint(value))]
	if len(matches) != 1 || matches[0] == nil {
		return nil, true, false
	}
	return matches[0].ID, true, true
}

func historyTargetFieldValue(
	entry historyImportTemplateCatalog,
	field *crmmodel.DataField,
	value any,
) (any, bool) {
	options := entry.Options[field.ID]
	if len(options) == 0 {
		return value, true
	}
	values := historyStringValues(value)
	resolved := make([]string, 0, len(values))
	for _, item := range values {
		optionValue := options[historyCatalogKey(item)]
		if optionValue == "" {
			return nil, false
		}
		resolved = append(resolved, optionValue)
	}
	if field.FieldType == "checkbox" || field.FieldType == "multi_select" {
		return resolved, true
	}
	if len(resolved) == 0 {
		return nil, true
	}
	return resolved[0], true
}

func persistHistoryOperation(
	ctx context.Context,
	catalog historyImportCatalog,
	caseID string,
	input HistoryImportOperationInput,
	customerID uint64,
	assetID uint64,
	workflowInstanceID uint64,
) (uint64, bool) {
	if customerID == 0 {
		return 0, false
	}
	if existing := crmmodel.NewOperationLogModel().Find(ctx, map[string]any{"source_key": input.SourceKey}); existing != nil {
		return existing.ID, false
	}
	staff, _ := resolveHistoryStaff(catalog, input.Operator, false, caseID, input.SourceKey)
	staffID, departmentID := historyStaffOwnership(staff)
	instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": workflowInstanceID})
	workflowID, stageID := uint64(0), uint64(0)
	if instance != nil {
		workflowID, stageID = instance.WorkflowID, instance.StageID
	}
	id := uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"source_key": input.SourceKey, "customer_id": customerID, "asset_id": assetID,
		"workflow_instance_id": workflowInstanceID, "customer_product_id": uint64(0),
		"workflow_id": workflowID, "stage_id": stageID, "task_id": uint64(0),
		"task_type": "", "result_value": strings.TrimSpace(input.Result),
		"title": strings.TrimSpace(input.Title), "content": strings.TrimSpace(input.Content),
		"data_snapshot_json": jsonText(input.Snapshot), "operator_staff_id": staffID,
		"operator_department_id": departmentID, "created_at": historyInputTime(input.CreatedAt),
	}))
	return id, id > 0
}

func persistHistoryMeeting(
	ctx context.Context,
	catalog historyImportCatalog,
	caseID string,
	input HistoryImportMeetingInput,
	customerID uint64,
	assetID uint64,
	workflowInstanceID uint64,
	uploadRuleID uint64,
) (uint64, bool, []HistoryImportIssue) {
	issues := make([]HistoryImportIssue, 0)
	if existing := crmmodel.NewScheduleEventModel().Find(ctx, map[string]any{"meeting_source_key": input.SourceKey}); existing != nil {
		return existing.ID, false, issues
	}
	owner, issue := resolveHistoryStaff(catalog, input.Owner, false, caseID, input.SourceKey)
	if issue != nil {
		issues = append(issues, *issue)
	}
	ownerID, _ := historyStaffOwnership(owner)
	status := input.Status
	if status != crmmodel.ScheduleStatusCompleted && status != crmmodel.ScheduleStatusCanceled {
		status = crmmodel.ScheduleStatusPending
	}
	arrivalStatus := input.ArrivalStatus
	if arrivalStatus != crmmodel.MeetingArrivalArrived && arrivalStatus != crmmodel.MeetingArrivalNoShow {
		arrivalStatus = crmmodel.MeetingArrivalPending
	}
	startAt, endAt := input.StartAt, input.EndAt
	if endAt.IsZero() || !endAt.After(startAt) {
		endAt = startAt.Add(time.Hour)
	}
	now := time.Now()
	data := map[string]any{
		"schedule_type": crmmodel.ScheduleTypeMeeting, "customer_id": customerID, "asset_id": assetID,
		"owner_staff_id": ownerID, "created_by_staff_id": ownerID,
		"source_workflow_instance_id": workflowInstanceID, "source_task_id": uint64(0),
		"meeting_source_key": input.SourceKey, "meeting_attempt": historyPositiveInt(input.Attempt, 1),
		"operation_log_id": uint64(0), "title": strings.TrimSpace(input.Title), "remark": strings.TrimSpace(input.Remark),
		"start_at": startAt, "end_at": endAt, "reminder_minutes": 0, "remind_at": startAt,
		"source": crmmodel.ScheduleSourceCalendar, "status": status, "arrival_status": arrivalStatus,
		"no_show_reason": strings.TrimSpace(input.NoShowReason), "created_at": startAt, "updated_at": now,
	}
	if input.ArrivalAt != nil {
		data["customer_arrived_at"] = input.ArrivalAt
		data["arrival_confirmed_at"] = input.ArrivalAt
	}
	if status == crmmodel.ScheduleStatusCompleted {
		data["completed_at"] = endAt
	}
	eventID := uint64(crmmodel.NewScheduleEventModel().Insert(ctx, data))
	if eventID == 0 {
		issues = append(issues, historyCaseError(caseID, "meeting_write_failed", "历史日程写入失败"))
		return 0, false, issues
	}
	people := append([]HistoryImportPersonInput{input.Owner}, input.Participants...)
	seenStaff := map[uint64]bool{}
	for _, person := range people {
		staff, staffIssue := resolveHistoryStaff(catalog, person, false, caseID, input.SourceKey)
		if staffIssue != nil {
			issues = append(issues, *staffIssue)
		}
		if staff == nil || seenStaff[staff.ID] {
			continue
		}
		seenStaff[staff.ID] = true
		crmmodel.NewScheduleParticipantModel().Insert(ctx, map[string]any{
			"schedule_event_id": eventID, "staff_id": staff.ID, "role": crmmodel.MemberRelationParticipant,
			"created_at": now, "updated_at": now,
		})
	}
	if matches := catalog.ResourcesByName[historyResourceKey(input.ResourceName)]; len(matches) == 1 && matches[0] != nil {
		resource := matches[0]
		bookingStatus := crmmodel.ResourceBookingStatusReserved
		if status == crmmodel.ScheduleStatusCompleted {
			bookingStatus = crmmodel.ResourceBookingStatusDone
		} else if status == crmmodel.ScheduleStatusCanceled {
			bookingStatus = crmmodel.ResourceBookingStatusCanceled
		}
		crmmodel.NewPublicResourceBookingModel().Insert(ctx, map[string]any{
			"resource_id": resource.ID, "schedule_event_id": eventID, "customer_id": customerID, "asset_id": assetID,
			"task_id": uint64(0), "operation_log_id": uint64(0), "stage_code": "history",
			"booking_status": bookingStatus, "title": strings.TrimSpace(input.Title), "remark": strings.TrimSpace(input.Remark),
			"start_at": startAt, "end_at": endAt, "booker_staff_id": ownerID, "booker_department_id": historyDepartmentID(owner),
			"created_at": startAt, "updated_at": now,
		})
	} else if strings.TrimSpace(input.ResourceName) != "" {
		issues = append(issues, HistoryImportIssue{
			Severity: HistoryImportSeverityWarning, Code: "meeting_resource_unresolved", CaseID: caseID,
			SourceKey: input.SourceKey, Message: "未写入会议室预定：" + input.ResourceName,
		})
	}
	attachmentIDs, attachmentIssues := persistHistoryAttachments(
		ctx, input.Attachments, customerID, assetID, eventID, 0, uploadRuleID,
	)
	_ = attachmentIDs
	issues = append(issues, attachmentIssues...)
	return eventID, true, issues
}

func persistHistoryGroup(
	ctx context.Context,
	catalog historyImportCatalog,
	caseID string,
	input HistoryImportCommunicationGroupInput,
	customerID uint64,
	assetID uint64,
	workflowInstanceID uint64,
) (uint64, bool, []HistoryImportIssue) {
	issues := make([]HistoryImportIssue, 0)
	if existing := crmmodel.NewCommunicationGroupModel().Find(ctx, map[string]any{"source_key": input.SourceKey}); existing != nil {
		return existing.ID, false, issues
	}
	if catalog.GroupType == nil || customerID == 0 || workflowInstanceID == 0 {
		issues = append(issues, historyCaseError(caseID, "group_target_missing", "企业微信群缺少群类型、客户或流程归属"))
		return 0, false, issues
	}
	status := input.Status
	if status != crmmodel.CommunicationGroupStatusDissolved {
		status = crmmodel.CommunicationGroupStatusActive
	}
	if status == crmmodel.CommunicationGroupStatusActive {
		if active := crmmodel.NewCommunicationGroupModel().Find(ctx, map[string]any{
			"workflow_instance_id": workflowInstanceID, "status": crmmodel.CommunicationGroupStatusActive,
		}); active != nil {
			issues = append(issues, historyCaseError(caseID, "active_group_conflict", "当前案件已有使用中的沟通群"))
			return 0, false, issues
		}
	}
	now := time.Now()
	groupID := uint64(crmmodel.NewCommunicationGroupModel().Insert(ctx, map[string]any{
		"customer_id": customerID, "asset_id": assetID, "workflow_instance_id": workflowInstanceID,
		"group_type_id": catalog.GroupType.ID, "name": historyRequiredName(input.Name),
		"external_group_id": strings.TrimSpace(input.ExternalID), "status": status,
		"established_at": input.EstablishedAt, "dissolved_at": input.DissolvedAt,
		"dissolve_reason": strings.TrimSpace(input.DissolveReason), "summary": strings.TrimSpace(input.Summary),
		"remark": strings.TrimSpace(input.Remark), "source_key": input.SourceKey,
		"created_by_staff_id": uint64(0), "created_at": input.EstablishedAt, "updated_at": now,
	}))
	if groupID == 0 {
		issues = append(issues, historyCaseError(caseID, "group_write_failed", "企业微信群写入失败"))
		return 0, false, issues
	}
	for _, relation := range input.Staff {
		staff, staffIssue := resolveHistoryStaff(catalog, relation.Person, false, caseID, input.SourceKey)
		if staffIssue != nil {
			issues = append(issues, *staffIssue)
		}
		if staff == nil {
			continue
		}
		role := relation.Role
		switch role {
		case crmmodel.CommunicationGroupStaffNPLOwner, crmmodel.CommunicationGroupStaffPMOwner, crmmodel.CommunicationGroupStaffALAOwner:
		default:
			role = crmmodel.CommunicationGroupStaffParticipant
		}
		if crmmodel.NewCommunicationGroupStaffModel().Find(ctx, map[string]any{
			"communication_group_id": groupID, "staff_id": staff.ID,
		}) == nil {
			crmmodel.NewCommunicationGroupStaffModel().Insert(ctx, map[string]any{
				"communication_group_id": groupID, "staff_id": staff.ID, "role": role,
				"created_at": now, "updated_at": now,
			})
		}
	}
	return groupID, true, issues
}

func persistHistoryAttachments(
	ctx context.Context,
	inputs []HistoryImportAttachmentInput,
	customerID uint64,
	assetID uint64,
	scheduleEventID uint64,
	dataRecordID uint64,
	uploadRuleID uint64,
) ([]uint64, []HistoryImportIssue) {
	ids := make([]uint64, 0, len(inputs))
	issues := make([]HistoryImportIssue, 0)
	for _, input := range inputs {
		if existing := crmmodel.NewAttachmentModel().Find(ctx, map[string]any{"source_key": input.SourceKey}); existing != nil {
			ids = append(ids, existing.ID)
			continue
		}
		if uploadRuleID == 0 || strings.TrimSpace(input.LocalPath) == "" {
			issues = append(issues, HistoryImportIssue{
				Severity: HistoryImportSeverityWarning, Code: "attachment_not_downloaded", SourceKey: input.SourceKey,
				Message: "附件未下载或未指定上传规则：" + input.FileName,
			})
			continue
		}
		file, err := uploadservice.ImportFile(ctx, uploadservice.ImportFileInput{
			RuleID: uploadRuleID, Kind: "file", Name: input.FileName, Mime: input.MIME,
			LocalPath: input.LocalPath, BizKey: "crm.history", BizName: "CRM历史资料",
		})
		if err != nil {
			issues = append(issues, HistoryImportIssue{
				Severity: HistoryImportSeverityWarning, Code: "attachment_upload_failed", SourceKey: input.SourceKey,
				Message: fmt.Sprintf("附件保存失败：%s：%v", input.FileName, err),
			})
			continue
		}
		payload := uploadrepo.BuildUploadFilePayload(file)
		fileURL := strings.TrimSpace(fmt.Sprint(payload["url"]))
		attachmentID := uint64(crmmodel.NewAttachmentModel().Insert(ctx, map[string]any{
			"source_key": input.SourceKey, "customer_id": customerID, "asset_id": assetID,
			"task_id": uint64(0), "operation_log_id": uint64(0), "data_record_id": dataRecordID,
			"field_id": uint64(0), "schedule_event_id": scheduleEventID, "upload_file_id": file.ID,
			"usage": strings.TrimSpace(input.FieldName), "file_name": input.FileName,
			"file_url": fileURL, "file_type": historyAttachmentType(input.MIME),
			"uploader_id": uint64(0), "created_at": time.Now(),
		}))
		if attachmentID == 0 {
			issues = append(issues, HistoryImportIssue{
				Severity: HistoryImportSeverityWarning, Code: "attachment_record_failed", SourceKey: input.SourceKey,
				Message: "附件业务记录写入失败：" + input.FileName,
			})
			continue
		}
		ids = append(ids, attachmentID)
	}
	return ids, issues
}

func historyPrimaryAssetID(inputs []HistoryImportAssetInput, values map[string]uint64) uint64 {
	if id := values["primary"]; id > 0 {
		return id
	}
	for _, input := range inputs {
		if id := values[input.Key]; id > 0 {
			return id
		}
	}
	return 0
}

func historyInputTime(value *time.Time) time.Time {
	if value != nil && !value.IsZero() {
		return *value
	}
	return time.Now()
}

func historyRequiredName(value string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return "未命名"
}

func historyRequiredAssetName(value string, customerName string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return historyRequiredName(customerName) + "-历史资产"
}

func historyStaffOwnership(staff *crmmodel.Staff) (uint64, uint64) {
	if staff == nil {
		return 0, 0
	}
	return staff.ID, staff.DepartmentID
}

func historyDepartmentID(staff *crmmodel.Staff) uint64 {
	_, departmentID := historyStaffOwnership(staff)
	return departmentID
}

func mergeHistoryString(updates map[string]any, key string, current string, historical string) {
	if strings.TrimSpace(current) == "" && strings.TrimSpace(historical) != "" {
		updates[key] = strings.TrimSpace(historical)
	}
}

func historyValueEmpty(value any) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []string:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	}
	return false
}

func historyStringValues(value any) []string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			result = append(result, historyStringValues(item)...)
		}
		return result
	default:
		return []string{strings.TrimSpace(fmt.Sprint(value))}
	}
}

func historyValuesEqual(left any, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func historyPositiveInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func historyAttachmentType(mime string) string {
	mime = strings.ToLower(strings.TrimSpace(mime))
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "audio/"):
		return "audio"
	default:
		return "file"
	}
}
