package service

import (
	"context"
	"fmt"
	"strings"

	crmmodel "github.com/dever-package/crm/model"
)

func resolveHistoryExistingTargets(
	ctx context.Context,
	catalog historyImportCatalog,
	plan *historyImportCasePlan,
) {
	if plan == nil {
		return
	}
	input := plan.Input
	if plan.Target.LeadID > 0 {
		if lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": plan.Target.LeadID}); lead == nil {
			plan.Issues = append(plan.Issues, historyCaseError(input.CaseID, "lead_override_missing", "指定线索不存在"))
		} else if input.Lead != nil {
			plan.Issues = append(plan.Issues, historyMainFieldConflicts(input.CaseID, "线索", map[string][2]string{
				"姓名": {lead.Name, input.Lead.Name}, "手机号": {lead.Phone, input.Lead.Phone},
				"微信号": {lead.Wechat, input.Lead.Wechat}, "外部线索ID": {lead.ExternalID, input.Lead.ExternalID},
				"城市": {lead.City, input.Lead.City}, "初始诉求": {lead.InitialNeed, input.Lead.InitialNeed},
			})...)
		}
	} else if input.Lead != nil {
		source := catalog.SourcesByCode[historyCatalogKey(input.Lead.SourceCode)]
		if source != nil && strings.TrimSpace(input.Lead.ExternalID) != "" {
			matches := crmmodel.NewLeadModel().Select(ctx, map[string]any{
				"source_id":   source.ID,
				"external_id": strings.TrimSpace(input.Lead.ExternalID),
			})
			if len(matches) == 1 && matches[0] != nil {
				plan.Target.LeadID = matches[0].ID
			} else if len(matches) > 1 {
				plan.Issues = append(plan.Issues, historyCaseError(input.CaseID, "multiple_lead_targets", "来源和外部线索ID对应多个当前线索"))
			}
		}
		if plan.Target.LeadID == 0 {
			appendHistoryPhoneConflict(ctx, &plan.Issues, input.CaseID, input.Lead.Phone, "线索")
		}
	}

	if plan.Target.CustomerID > 0 {
		if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": plan.Target.CustomerID}); customer == nil {
			plan.Issues = append(plan.Issues, historyCaseError(input.CaseID, "customer_override_missing", "指定客户不存在"))
		} else if input.Customer != nil {
			plan.Issues = append(plan.Issues, historyMainFieldConflicts(input.CaseID, "客户", map[string][2]string{
				"姓名": {customer.Name, input.Customer.Name}, "手机号": {customer.Phone, input.Customer.Phone},
				"微信号": {customer.Wechat, input.Customer.Wechat}, "身份证号": {customer.IDCard, input.Customer.IDCard},
				"备注": {customer.Remark, input.Customer.Remark},
			})...)
		}
	} else {
		if plan.Target.LeadID > 0 {
			if lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": plan.Target.LeadID}); lead != nil && lead.CustomerID > 0 {
				plan.Target.CustomerID = lead.CustomerID
			}
		}
		if plan.Target.CustomerID == 0 && input.Customer != nil {
			appendHistoryCustomerPhoneConflict(ctx, &plan.Issues, input.CaseID, input.Customer.Phone)
		}
	}

	if plan.Target.CustomerID > 0 && len(plan.Target.AssetIDs) == 0 && len(input.Assets) == 1 {
		assets := crmmodel.NewCustomerAssetModel().Select(ctx, map[string]any{
			"customer_id": plan.Target.CustomerID,
		})
		if len(assets) == 1 && assets[0] != nil {
			plan.Target.AssetIDs[input.Assets[0].Key] = assets[0].ID
		}
	}
	for key, assetID := range plan.Target.AssetIDs {
		asset := crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{"id": assetID})
		if asset == nil {
			plan.Issues = append(plan.Issues, historyCaseError(input.CaseID, "asset_override_missing", fmt.Sprintf("指定资产不存在：%s", key)))
			continue
		}
		if plan.Target.CustomerID > 0 && asset.CustomerID != plan.Target.CustomerID {
			plan.Issues = append(plan.Issues, historyCaseError(input.CaseID, "asset_customer_mismatch", fmt.Sprintf("资产不属于指定客户：%s", key)))
		}
	}

	if plan.Target.WorkflowInstanceID > 0 {
		instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": plan.Target.WorkflowInstanceID})
		if instance == nil {
			plan.Issues = append(plan.Issues, historyCaseError(input.CaseID, "workflow_override_missing", "指定流程实例不存在"))
		} else {
			plan.Target.CustomerID = preferredUint64(plan.Target.CustomerID, instance.CustomerID)
			if instance.AssetID > 0 && len(plan.Target.AssetIDs) == 0 {
				plan.Target.AssetIDs["primary"] = instance.AssetID
			}
		}
	}
}

func historyMainFieldConflicts(caseID string, targetName string, values map[string][2]string) []HistoryImportIssue {
	issues := make([]HistoryImportIssue, 0)
	for fieldName, pair := range values {
		current, historical := strings.TrimSpace(pair[0]), strings.TrimSpace(pair[1])
		if current == "" || historical == "" || current == historical {
			continue
		}
		issues = append(issues, HistoryImportIssue{
			Severity: HistoryImportSeverityWarning, Code: "current_value_kept", CaseID: caseID,
			Field: fieldName, Message: fmt.Sprintf("%s%s已有非空值，预览将保留当前值", targetName, fieldName),
		})
	}
	return issues
}

func validateHistoryCaseMappings(
	ctx context.Context,
	catalog historyImportCatalog,
	plan historyImportCasePlan,
	options HistoryImportOptions,
) []HistoryImportIssue {
	issues := make([]HistoryImportIssue, 0)
	input := plan.Input
	if input.Lead != nil {
		if strings.TrimSpace(input.Lead.SourceCode) == "" {
			issues = append(issues, historyCaseError(input.CaseID, "source_missing", "线索来源映射为空"))
		}
		if strings.TrimSpace(input.Lead.ChannelCode) == "" {
			issues = append(issues, historyCaseError(input.CaseID, "channel_missing", "线索渠道映射为空"))
		}
	}

	for _, operation := range input.Operations {
		_, issue := resolveHistoryStaff(catalog, operation.Operator, false, input.CaseID, operation.SourceKey)
		if issue != nil {
			issues = append(issues, *issue)
		}
	}
	for _, meeting := range input.Meetings {
		_, issue := resolveHistoryStaff(catalog, meeting.Owner, false, input.CaseID, meeting.SourceKey)
		if issue != nil {
			issues = append(issues, *issue)
		}
		if strings.TrimSpace(meeting.ResourceName) != "" {
			matches := catalog.ResourcesByName[historyResourceKey(meeting.ResourceName)]
			if len(matches) != 1 {
				issues = append(issues, HistoryImportIssue{
					Severity: HistoryImportSeverityWarning, Code: "meeting_resource_unresolved",
					CaseID: input.CaseID, SourceKey: meeting.SourceKey,
					Message: fmt.Sprintf("会议室无法唯一匹配：%s", meeting.ResourceName),
				})
			}
		}
	}
	for _, group := range input.Groups {
		for _, member := range group.Staff {
			_, issue := resolveHistoryStaff(catalog, member.Person, false, input.CaseID, group.SourceKey)
			if issue != nil {
				issues = append(issues, *issue)
			}
		}
	}

	if len(input.Assets) > 0 {
		workflow, stage := resolveHistoryWorkflowStage(catalog, input.Workflow.StageName)
		if workflow == nil {
			issues = append(issues, historyCaseError(input.CaseID, "workflow_missing", "未找到已启用的签约流程"))
		} else if stage == nil {
			issues = append(issues, historyCaseError(input.CaseID, "stage_missing", fmt.Sprintf("未找到流程阶段：%s", input.Workflow.StageName)))
		}
		if input.Workflow.Status == crmmodel.ProgressStatusActive && options.RestoreActiveWorkflows {
			owner, issue := resolveHistoryStaff(catalog, input.Workflow.Owner, true, input.CaseID, "")
			if issue != nil {
				issue.Severity = HistoryImportSeverityError
				issue.Code = "active_owner_unresolved"
				issues = append(issues, *issue)
			} else if owner == nil {
				issues = append(issues, historyCaseError(
					input.CaseID,
					"active_owner_missing",
					"活动案件缺少当前流程负责人，不能恢复待办",
				))
			}
		}
	}

	for _, record := range input.CustomerRecords {
		issues = append(issues, validateHistoryDataRecord(catalog, input.CaseID, record)...)
		issues = append(issues, previewHistoryDataConflicts(ctx, catalog, input.CaseID, workDataOwnership{
			CustomerID: plan.Target.CustomerID,
		}, record)...)
	}
	for _, asset := range input.Assets {
		for _, record := range asset.Records {
			issues = append(issues, validateHistoryDataRecord(catalog, input.CaseID, record)...)
			issues = append(issues, previewHistoryDataConflicts(ctx, catalog, input.CaseID, workDataOwnership{
				CustomerID: plan.Target.CustomerID,
				AssetID:    plan.Target.AssetIDs[asset.Key],
			}, record)...)
		}
	}
	_ = ctx
	return issues
}

func previewHistoryDataConflicts(
	ctx context.Context,
	catalog historyImportCatalog,
	caseID string,
	ownership workDataOwnership,
	input HistoryImportDataRecordInput,
) []HistoryImportIssue {
	if ownership.CustomerID == 0 {
		return nil
	}
	entry, exists := catalog.Templates[historyCatalogKey(input.TemplateName)]
	if !exists || entry.Template == nil {
		return nil
	}
	existing := crmmodel.NewDataRecordModel().Find(ctx, workDataRecordOwnershipFilter(ownership, entry.Template.ID))
	if existing == nil {
		return nil
	}
	current := mapFromAny(existing.RecordJSON)
	issues := make([]HistoryImportIssue, 0)
	for name, value := range input.Fields {
		if historyValueEmpty(value) {
			continue
		}
		field := resolveHistoryDataField(entry, input.TemplateName, name)
		if field == nil || historyValueEmpty(current[field.FieldKey]) {
			continue
		}
		converted, handled, ok := historySpecialFieldValue(catalog, input.TemplateName, name, value)
		if !handled {
			converted, ok = historyTargetFieldValue(entry, field, value)
		}
		if ok && !historyValuesEqual(current[field.FieldKey], converted) {
			issues = append(issues, HistoryImportIssue{
				Severity: HistoryImportSeverityWarning, Code: "current_value_kept",
				CaseID: caseID, Field: field.FieldKey,
				Message: fmt.Sprintf("当前CRM已有非空值，预览将保留当前值：%s/%s", input.TemplateName, name),
			})
		}
	}
	return issues
}

func validateHistoryDataRecord(
	catalog historyImportCatalog,
	caseID string,
	record HistoryImportDataRecordInput,
) []HistoryImportIssue {
	entry, exists := catalog.Templates[historyCatalogKey(record.TemplateName)]
	if !exists {
		if strings.HasPrefix(strings.TrimSpace(record.TemplateName), "历史飞书") {
			return nil
		}
		return []HistoryImportIssue{historyCaseError(caseID, "template_missing", "目标数据模板不存在："+record.TemplateName)}
	}
	issues := make([]HistoryImportIssue, 0)
	for name, value := range record.Fields {
		if historyValueEmpty(value) {
			continue
		}
		field := resolveHistoryDataField(entry, record.TemplateName, name)
		if field == nil {
			if strings.HasPrefix(strings.TrimSpace(record.TemplateName), "历史飞书") {
				continue
			}
			issues = append(issues, historyCaseError(caseID, "field_missing", fmt.Sprintf("目标字段不存在：%s/%s", record.TemplateName, name)))
			continue
		}
		if _, handled, ok := historySpecialFieldValue(catalog, record.TemplateName, name, value); handled {
			if !ok {
				issues = append(issues, historyCaseError(caseID, "resource_unresolved", fmt.Sprintf("会议室无法唯一匹配：%v", value)))
			}
			continue
		}
		if options := entry.Options[field.ID]; len(options) > 0 {
			for _, optionValue := range historyStringValues(value) {
				if options[historyCatalogKey(optionValue)] == "" {
					issues = append(issues, historyCaseError(caseID, "option_unresolved", fmt.Sprintf("目标选项不存在：%s/%s=%s", record.TemplateName, name, optionValue)))
				}
			}
		}
	}
	return issues
}

func resolveHistoryDataField(
	entry historyImportTemplateCatalog,
	templateName string,
	fieldName string,
) *crmmodel.DataField {
	if field := entry.Fields[historyCatalogKey(fieldName)]; field != nil {
		return field
	}
	aliasKey := historyCatalogKey(templateName) + "/" + historyCatalogKey(fieldName)
	for _, alias := range historyImportFieldAliases[aliasKey] {
		if field := entry.Fields[historyCatalogKey(alias)]; field != nil {
			return field
		}
	}
	return nil
}

var historyImportFieldAliases = map[string][]string{
	historyCatalogKey("邀约到访") + "/" + historyCatalogKey("预约时间"):     {"邀约时间", "会议开始时间"},
	historyCatalogKey("邀约到访") + "/" + historyCatalogKey("会议时长"):     {"会议时长（分钟）", "会议时长字段（分钟）"},
	historyCatalogKey("邀约到访") + "/" + historyCatalogKey("预约会议室"):    {"会议室", "会议室字段"},
	historyCatalogKey("邀约到访") + "/" + historyCatalogKey("实际到访时间"):   {"到访时间"},
	historyCatalogKey("邀约到访") + "/" + historyCatalogKey("未到访或签约原因"): {"未到访原因"},
	historyCatalogKey("资产基础信息") + "/" + historyCatalogKey("资产地址"):   {"房屋地址", "地址"},
	historyCatalogKey("资产基础信息") + "/" + historyCatalogKey("面积/户型"):  {"面积户型"},
	historyCatalogKey("合同信息") + "/" + historyCatalogKey("服务费"):      {"服务费用"},
	historyCatalogKey("合同信息") + "/" + historyCatalogKey("评估租金"):     {"ALA最终评估租金", "最终评估租金"},
}

func resolveHistoryStaff(
	catalog historyImportCatalog,
	person HistoryImportPersonInput,
	requireEnabled bool,
	caseID string,
	sourceKey string,
) (*crmmodel.Staff, *HistoryImportIssue) {
	if strings.TrimSpace(person.OpenID) == "" && strings.TrimSpace(person.Name) == "" {
		return nil, nil
	}
	matches := []*crmmodel.Staff{}
	if key := historyCatalogKey(person.OpenID); key != "" {
		matches = catalog.StaffByOpenID[key]
	}
	if len(matches) == 0 {
		matches = catalog.StaffByName[historyCatalogKey(person.Name)]
	}
	if len(matches) == 1 && matches[0] != nil {
		if !requireEnabled || matches[0].Status == crmmodel.StatusEnabled {
			return matches[0], nil
		}
		return nil, &HistoryImportIssue{
			Severity: HistoryImportSeverityWarning, Code: "staff_disabled", CaseID: caseID,
			SourceKey: sourceKey, Message: "历史人员已停用，不能作为活动负责人：" + matches[0].Name,
		}
	}
	name := strings.TrimSpace(person.Name)
	if name == "" {
		name = strings.TrimSpace(person.OpenID)
	}
	code := "staff_unresolved"
	message := "历史人员无法匹配：" + name
	if len(matches) > 1 {
		code = "staff_ambiguous"
		message = "历史人员存在重名：" + name
	}
	return nil, &HistoryImportIssue{
		Severity: HistoryImportSeverityWarning, Code: code, CaseID: caseID,
		SourceKey: sourceKey, Message: message,
	}
}

func resolveHistoryWorkflowStage(
	catalog historyImportCatalog,
	stageName string,
) (*crmmodel.Workflow, *crmmodel.Stage) {
	workflow := catalog.WorkflowsByName[historyCatalogKey("签约流程")]
	if workflow == nil || workflow.Status != crmmodel.StatusEnabled {
		for _, candidate := range catalog.WorkflowsByName {
			if candidate != nil && candidate.SubjectType == crmmodel.WorkflowSubjectCustomerAsset && candidate.Status == crmmodel.StatusEnabled {
				workflow = candidate
				break
			}
		}
	}
	if workflow == nil {
		return nil, nil
	}
	stages := catalog.StagesByWorkflow[workflow.ID]
	stage := stages[historyCatalogKey(stageName)]
	if stage == nil || stage.Status != crmmodel.StatusEnabled {
		return workflow, nil
	}
	return workflow, stage
}

func appendHistoryPhoneConflict(
	ctx context.Context,
	issues *[]HistoryImportIssue,
	caseID string,
	phone string,
	targetName string,
) {
	phone = strings.TrimSpace(phone)
	if phone == "" || issues == nil {
		return
	}
	if count := crmmodel.NewLeadModel().Count(ctx, map[string]any{"phone": phone}); count > 0 {
		*issues = append(*issues, historyCaseError(caseID, "phone_requires_override", fmt.Sprintf("手机号已存在于%d条当前%s，需显式指定目标", count, targetName)))
	}
	if count := crmmodel.NewCustomerModel().Count(ctx, map[string]any{"phone": phone}); count > 0 {
		*issues = append(*issues, historyCaseError(caseID, "phone_requires_override", fmt.Sprintf("手机号已存在于%d个当前客户，需显式指定目标", count)))
	}
}

func appendHistoryCustomerPhoneConflict(
	ctx context.Context,
	issues *[]HistoryImportIssue,
	caseID string,
	phone string,
) {
	phone = strings.TrimSpace(phone)
	if phone == "" || issues == nil {
		return
	}
	if count := crmmodel.NewCustomerModel().Count(ctx, map[string]any{"phone": phone}); count > 0 {
		*issues = append(*issues, historyCaseError(caseID, "customer_phone_requires_override", fmt.Sprintf("手机号已存在于%d个当前客户，需显式指定目标", count)))
	}
}

func historyCaseError(caseID string, code string, message string) HistoryImportIssue {
	return HistoryImportIssue{
		Severity: HistoryImportSeverityError,
		Code:     code,
		CaseID:   caseID,
		Message:  message,
	}
}
