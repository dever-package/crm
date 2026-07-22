package service

import (
	"context"
	"fmt"

	crmmodel "github.com/dever-package/crm/model"
)

// workBusinessOperationRows projects engine audit logs into the business
// timeline without deleting the underlying audit records.
func workBusinessOperationRows(ctx context.Context, rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if inputText(row["result_value"]) == "assigned" {
			continue
		}
		if isWorkflowStartedOperation(row) {
			leadID := firstUint64(mapFromAny(row["data_snapshot_json"]), "lead_id", "leadId")
			if leadID == 0 {
				continue
			}
			row["business_event"] = workBusinessEventLeadCreated
			row["title"] = "新增线索"
			row["content"] = "线索已录入"
			attachLeadCreatorToOperation(ctx, row, crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": leadID}))
		}
		if inputText(row["result_value"]) == workBusinessEventLeadConverted {
			row["business_event"] = workBusinessEventLeadConverted
		}
		if isCommunicationGroupOperation(row) {
			row["business_event"] = inputText(row["result_value"])
		}
		result = append(result, row)
	}
	return result
}

func isCommunicationGroupOperation(row map[string]any) bool {
	switch inputText(row["result_value"]) {
	case workBusinessEventCommunicationGroupCreated,
		workBusinessEventCommunicationGroupUpdated,
		workBusinessEventCommunicationGroupDissolved:
		return true
	default:
		return false
	}
}

func workCommunicationGroupOperationSummaryItems(snapshot map[string]any) []map[string]any {
	items := []map[string]any{}
	for _, field := range []struct {
		key   string
		label string
	}{
		{key: "group_name", label: "群名称"},
		{key: "group_type", label: "群类型"},
		{key: "established_at", label: "建群日期"},
		{key: "dissolved_at", label: "解散日期"},
		{key: "dissolve_reason", label: "解散原因"},
		{key: "staff_names", label: "关联人员"},
		{key: "summary", label: "智能总结"},
		{key: "remark", label: "备注"},
	} {
		if value := inputText(snapshot[field.key]); value != "" {
			items = appendWorkOperationSummaryValue(items, "communication_group", "沟通群", field.key, field.label, value)
		}
	}
	return items
}

func isWorkflowStartedOperation(row map[string]any) bool {
	if inputText(row["result_value"]) != "entered" {
		return false
	}
	snapshot := mapFromAny(row["data_snapshot_json"])
	return firstUint64(snapshot, "from_workflow_id", "fromWorkflowId") == 0 &&
		firstUint64(snapshot, "from_stage_id", "fromStageId") == 0 &&
		firstUint64(snapshot, "to_workflow_id", "toWorkflowId") > 0
}

func attachLeadCreatorToOperation(ctx context.Context, row map[string]any, lead *crmmodel.Lead) {
	if lead == nil || lead.CreatedByStaffID == 0 {
		return
	}
	row["operator_staff_id"] = lead.CreatedByStaffID
	if creator := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": lead.CreatedByStaffID}); creator != nil {
		row["operator_department_id"] = creator.DepartmentID
	}
}

func workLeadConversionSummaryItems(ctx context.Context, leadID, customerID, assetID uint64) []map[string]any {
	items := []map[string]any{}
	if lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": leadID}); lead != nil {
		items = appendWorkOperationSummaryValue(items, "lead", "线索信息", "lead:code", "线索编号", lead.Code)
		items = appendWorkOperationSummaryValue(items, "lead", "线索信息", "lead:name", "姓名", lead.Name)
		items = appendWorkOperationSummaryValue(items, "lead", "线索信息", "lead:phone", "手机号", lead.Phone)
		items = appendWorkOperationSummaryValue(items, "lead", "线索信息", "lead:wechat", "微信", lead.Wechat)
		items = appendWorkOperationSummaryValue(items, "lead", "线索信息", "lead:source", "来源", workCustomerSourceName(ctx, lead.SourceID))
		items = appendWorkOperationSummaryValue(items, "lead", "线索信息", "lead:channel", "渠道", workCustomerChannelName(ctx, lead.ChannelID))
		items = appendWorkOperationSummaryValue(items, "lead", "线索信息", "lead:external_id", "外部线索ID", lead.ExternalID)
		items = appendWorkOperationSummaryValue(items, "lead", "线索信息", "lead:city", "城市", lead.City)
		items = appendWorkOperationSummaryValue(items, "lead", "线索信息", "lead:initial_need", "初始诉求", lead.InitialNeed)
		items = appendWorkDetailSectionSummaryItems(items, "lead", "线索信息", workDataDetailSections(
			ctx,
			crmmodel.DataTemplateTargetLead,
			crmmodel.LeadDataTemplateCateID,
			workLeadDataValues(lead),
		))
	}

	customer := crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
	if len(customer) > 0 {
		items = appendWorkOperationSummaryValue(items, "customer", "客户信息", "customer:code", "客户编号", customerCodeDisplayForWork(ctx, inputText(customer["code"])))
		items = appendWorkOperationSummaryValue(items, "customer", "客户信息", "customer:name", "姓名", customer["name"])
		items = appendWorkOperationSummaryValue(items, "customer", "客户信息", "customer:phone", "手机号", customer["phone"])
		items = appendWorkOperationSummaryValue(items, "customer", "客户信息", "customer:wechat", "微信", customer["wechat"])
		items = appendWorkOperationSummaryValue(items, "customer", "客户信息", "customer:source", "来源", workCustomerSourceName(ctx, inputUint64(customer["source_id"])))
		items = appendWorkOperationSummaryValue(items, "customer", "客户信息", "customer:channel", "渠道", workCustomerChannelName(ctx, inputUint64(customer["channel_id"])))
		items = appendWorkOperationSummaryValue(items, "customer", "客户信息", "customer:remark", "备注", customer["remark"])
		items = appendWorkDetailSectionSummaryItems(items, "customer", "客户信息", workDataDetailSections(
			ctx,
			crmmodel.DataTemplateTargetCustomer,
			crmmodel.CustomerDataTemplateCateID,
			workCustomerFormValues(ctx, customerID, 0, customer),
		))
	}

	asset := crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{"id": assetID, "customer_id": customerID})
	if len(asset) > 0 {
		items = appendWorkOperationSummaryValue(items, "asset", "资产信息", "asset:asset_no", "资产编号", asset["asset_no"])
		items = appendWorkOperationSummaryValue(items, "asset", "资产信息", "asset:asset_name", "资产名称", asset["asset_name"])
		items = appendWorkOperationSummaryValue(items, "asset", "资产信息", "asset:status", "资产状态", workAssetStatusName(ctx, inputUint64(asset["asset_status_id"])))
		items = appendWorkOperationSummaryValue(items, "asset", "资产信息", "asset:remark", "备注", asset["remark"])
		items = appendWorkDetailSectionSummaryItems(items, "asset", "资产信息", workDataDetailSections(
			ctx,
			crmmodel.DataTemplateTargetCustomerAsset,
			crmmodel.CustomerAssetDataTemplateCateID,
			workAssetFormValues(ctx, customerID, assetID, asset),
		))
	}
	return items
}

func appendWorkOperationSummaryValue(
	items []map[string]any,
	groupID string,
	groupLabel string,
	key string,
	label string,
	value any,
) []map[string]any {
	displayValue := inputText(value)
	if displayValue == "" {
		return items
	}
	return append(items, map[string]any{
		"key":         key,
		"label":       label,
		"value":       displayValue,
		"group_id":    groupID,
		"group_label": groupLabel,
	})
}

func appendWorkDetailSectionSummaryItems(
	items []map[string]any,
	groupID string,
	groupLabel string,
	sections []map[string]any,
) []map[string]any {
	for _, section := range sections {
		for _, field := range mapListFromAny(section["fields"]) {
			if booleanFromAny(field["empty"]) || inputText(field["value"]) == "" {
				continue
			}
			item := map[string]any{
				"key":         fmt.Sprintf("%s:%v:%v", groupID, section["id"], field["key"]),
				"label":       field["label"],
				"value":       field["value"],
				"group_id":    groupID,
				"group_label": groupLabel,
			}
			for _, metaKey := range []string{"value_type", "files"} {
				if value, exists := field[metaKey]; exists {
					item[metaKey] = value
				}
			}
			items = append(items, item)
		}
	}
	return items
}
