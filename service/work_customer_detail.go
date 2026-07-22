package service

import (
	"context"
	"fmt"

	crmmodel "github.com/dever-package/crm/model"
)

type workCustomerDetailAccess struct {
	ViewStaff          *WorkStaffSession
	CustomerID         uint64
	AssetID            uint64
	WorkflowInstanceID uint64
	DetailInstance     *crmmodel.WorkflowInstance
}

type workCustomerDetailTarget struct {
	Access     *workCustomerDetailAccess
	Customer   map[string]any
	Asset      map[string]any
	SourceLead *crmmodel.Lead
}

func (WorkService) CustomerProfile(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	access, err := resolveWorkCustomerDetailAccess(ctx, staff, payload)
	if err != nil {
		return nil, err
	}
	target, err := loadWorkCustomerDetailTarget(ctx, access)
	if err != nil {
		return nil, err
	}
	return workCustomerProfileResult(ctx, staff, target), nil
}

func (WorkService) CustomerDetail(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	access, err := resolveWorkCustomerDetailAccess(ctx, staff, payload)
	if err != nil {
		return nil, err
	}
	target, err := loadWorkCustomerDetailTarget(ctx, access)
	if err != nil {
		return nil, err
	}
	operations, err := (WorkService{}).Operations(ctx, access.ViewStaff, workCustomerDetailOperationPayload(access))
	if err != nil {
		return nil, err
	}
	result := workCustomerProfileResult(ctx, staff, target)
	result["operations"] = operations["list"]
	result["todos"] = operations["todos"]
	return result, nil
}

func (WorkService) CustomerOperations(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	access, err := resolveWorkCustomerDetailAccess(ctx, staff, payload)
	if err != nil {
		return nil, err
	}
	rows, err := loadWorkOperationRows(ctx, access.ViewStaff, workCustomerDetailOperationPayload(access))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"list":  rows,
		"total": len(rows),
	}, nil
}

func (WorkService) CustomerAttachments(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	access, err := resolveWorkCustomerDetailAccess(ctx, staff, payload)
	if err != nil {
		return nil, err
	}
	collector := newWorkCustomerAttachmentCollector()
	workCustomerEntityAttachments(ctx, access, collector)
	workCustomerStoredAttachments(ctx, access, collector)
	workCustomerOperationAttachments(ctx, access, collector)
	return map[string]any{
		"list":  collector.Rows,
		"total": len(collector.Rows),
	}, nil
}

func resolveWorkCustomerDetailAccess(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (*workCustomerDetailAccess, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	viewStaff := staff
	if staff.CanDispatch {
		viewStaff = workStaffWithScope(staff, "all")
	}
	customerID := firstUint64(payload, "customer_id", "customerId")
	if customerID == 0 {
		return nil, fmt.Errorf("请选择客户")
	}
	if !canViewWorkCustomer(ctx, viewStaff, customerID) {
		return nil, fmt.Errorf("无权查看该客户")
	}
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	access := &workCustomerDetailAccess{
		ViewStaff:          viewStaff,
		CustomerID:         customerID,
		AssetID:            firstUint64(payload, "asset_id", "assetId"),
		WorkflowInstanceID: firstUint64(payload, "workflow_instance_id", "workflowInstanceId"),
	}
	if access.AssetID == 0 {
		if access.WorkflowInstanceID > 0 {
			return nil, fmt.Errorf("查看流程记录时请选择资产")
		}
		return access, nil
	}
	if !canViewWorkAsset(ctx, viewStaff, customerID, access.AssetID) {
		return nil, fmt.Errorf("无权查看该资产")
	}
	asset := crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{"id": access.AssetID})
	if asset == nil || asset.CustomerID != customerID {
		return nil, fmt.Errorf("资产不存在或不可见")
	}
	if access.WorkflowInstanceID == 0 {
		return access, nil
	}
	instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": access.WorkflowInstanceID})
	if instance == nil || instance.CustomerID != customerID || instance.AssetID != access.AssetID {
		return nil, fmt.Errorf("流程实例不存在或不属于该资产")
	}
	if !canViewWorkflowInstance(ctx, viewStaff, instance) {
		return nil, fmt.Errorf("无权查看该流程记录")
	}
	access.DetailInstance = instance
	return access, nil
}

func loadWorkCustomerDetailTarget(ctx context.Context, access *workCustomerDetailAccess) (*workCustomerDetailTarget, error) {
	if access == nil {
		return nil, fmt.Errorf("客户详情参数无效")
	}
	customer := workCustomerRow(ctx, access.ViewStaff, access.CustomerID)
	if len(customer) == 0 {
		return nil, fmt.Errorf("客户不存在")
	}
	sourceLead := crmmodel.NewLeadModel().Find(ctx, map[string]any{
		"customer_id": access.CustomerID,
		"status":      crmmodel.LeadStatusConverted,
	}, map[string]any{"order": "converted_at desc,id desc"})
	if sourceLead != nil {
		customer["source_lead"] = workLeadRow(ctx, sourceLead)
	}
	var asset map[string]any
	if access.AssetID > 0 {
		asset = workCustomerRowAsset(customer, access.AssetID)
		if len(asset) == 0 {
			return nil, fmt.Errorf("资产不存在或不可见")
		}
		if access.DetailInstance != nil {
			attachWorkStageFields(ctx, asset, access.DetailInstance)
		}
	}
	return &workCustomerDetailTarget{
		Access:     access,
		Customer:   customer,
		Asset:      asset,
		SourceLead: sourceLead,
	}, nil
}

func workCustomerProfileResult(ctx context.Context, staff *WorkStaffSession, target *workCustomerDetailTarget) map[string]any {
	access := target.Access
	detailSections := make([]map[string]any, 0)
	if target.SourceLead != nil {
		detailSections = append(detailSections, workDataDetailSections(
			ctx,
			crmmodel.DataTemplateTargetLead,
			crmmodel.LeadDataTemplateCateID,
			workLeadDataValues(target.SourceLead),
		)...)
	}
	detailSections = append(detailSections, workDataDetailSections(
		ctx,
		crmmodel.DataTemplateTargetCustomer,
		crmmodel.CustomerDataTemplateCateID,
		mapFromAny(target.Customer["data_values"]),
	)...)
	if len(target.Asset) > 0 {
		detailSections = append(detailSections, workDataDetailSections(
			ctx,
			crmmodel.DataTemplateTargetCustomerAsset,
			crmmodel.CustomerAssetDataTemplateCateID,
			mapFromAny(target.Asset["data_values"]),
		)...)
	}
	result := map[string]any{
		"customer":                  target.Customer,
		"asset":                     target.Asset,
		"detail_sections":           detailSections,
		"customer_products":         []map[string]any{},
		"workflow_instances":        []map[string]any{},
		"communication_groups":      workCommunicationGroupRows(ctx, staff, access.CustomerID, access.AssetID, access.WorkflowInstanceID),
		"communication_group_types": workCommunicationGroupTypes(ctx),
		"communication_group_workflow_instance_id": uint64(0),
		"can_manage_communication_groups":          false,
	}
	if access.AssetID == 0 {
		return result
	}
	customerProducts := mapListFromAny(target.Asset["customer_products"])
	result["customer_products"] = customerProducts
	workflowInstances := make([]map[string]any, 0, len(customerProducts)+1)
	if access.DetailInstance != nil {
		detailFlow := workFlowDetail(ctx, staff, access.DetailInstance.ID)
		if access.DetailInstance.CustomerProductID > 0 {
			detailFlow["flow_role"] = "product"
		} else {
			detailFlow["flow_role"] = "entry"
		}
		result["flow"] = detailFlow
	}
	groupInstance := access.DetailInstance
	if instance := currentWorkEntryInstance(ctx, access.CustomerID, access.AssetID); instance != nil {
		if groupInstance == nil {
			groupInstance = instance
		}
		entryFlow := workFlowDetail(ctx, staff, instance.ID)
		entryFlow["flow_role"] = "entry"
		if access.DetailInstance == nil {
			result["flow"] = entryFlow
		}
		workflowInstances = append(workflowInstances, entryFlow)
	} else if access.DetailInstance == nil {
		result["flow"] = map[string]any{
			"customer_id": access.CustomerID,
			"asset_id":    access.AssetID,
			"status":      "not_started",
			"tasks":       []map[string]any{},
		}
	}
	for _, customerProduct := range customerProducts {
		flow := mapFromAny(customerProduct["flow"])
		if len(flow) == 0 {
			continue
		}
		flow["flow_role"] = "product"
		flow["product_id"] = customerProduct["product_id"]
		flow["product_name"] = customerProduct["product_name"]
		flow["customer_product_id"] = customerProduct["customer_product_id"]
		workflowInstances = append(workflowInstances, flow)
	}
	result["workflow_instances"] = workflowInstances
	if groupInstance != nil {
		result["communication_group_workflow_instance_id"] = groupInstance.ID
		result["can_manage_communication_groups"] = canManageCommunicationGroup(ctx, staff, groupInstance)
	}
	return result
}

func workCustomerDetailOperationPayload(access *workCustomerDetailAccess) map[string]any {
	payload := map[string]any{"customer_id": access.CustomerID}
	if access.AssetID > 0 {
		payload["asset_id"] = access.AssetID
	}
	return payload
}

type workCustomerAttachmentCollector struct {
	Rows []map[string]any
	seen map[string]bool
}

func newWorkCustomerAttachmentCollector() *workCustomerAttachmentCollector {
	return &workCustomerAttachmentCollector{
		Rows: make([]map[string]any, 0),
		seen: map[string]bool{},
	}
}

func (collector *workCustomerAttachmentCollector) Append(files any, source string, fieldLabel string) {
	for _, file := range mapListFromAny(files) {
		key := firstText(file, "id", "url", "open_url", "download", "name")
		if key == "" {
			key = fmt.Sprintf("%s:%s:%d", source, fieldLabel, len(collector.Rows))
		}
		if collector.seen[key] {
			continue
		}
		collector.seen[key] = true
		collector.Rows = append(collector.Rows, map[string]any{
			"key":         key,
			"file":        file,
			"source":      source,
			"field_label": fieldLabel,
		})
	}
}

func workCustomerEntityAttachments(ctx context.Context, access *workCustomerDetailAccess, collector *workCustomerAttachmentCollector) {
	if access == nil || collector == nil {
		return
	}
	if sourceLead := crmmodel.NewLeadModel().Find(ctx, map[string]any{
		"customer_id": access.CustomerID,
		"status":      crmmodel.LeadStatusConverted,
	}, map[string]any{"order": "converted_at desc,id desc"}); sourceLead != nil {
		collectWorkDetailSectionAttachments(collector, workDataDetailSections(
			ctx,
			crmmodel.DataTemplateTargetLead,
			crmmodel.LeadDataTemplateCateID,
			workLeadDataValues(sourceLead),
		))
	}
	if customer := crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": access.CustomerID}); len(customer) > 0 {
		collectWorkDetailSectionAttachments(collector, workDataDetailSections(
			ctx,
			crmmodel.DataTemplateTargetCustomer,
			crmmodel.CustomerDataTemplateCateID,
			workCustomerFormValues(ctx, access.CustomerID, 0, customer),
		))
	}
	if access.AssetID == 0 {
		return
	}
	if asset := crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{
		"id":          access.AssetID,
		"customer_id": access.CustomerID,
	}); len(asset) > 0 {
		collectWorkDetailSectionAttachments(collector, workDataDetailSections(
			ctx,
			crmmodel.DataTemplateTargetCustomerAsset,
			crmmodel.CustomerAssetDataTemplateCateID,
			workAssetFormValues(ctx, access.CustomerID, access.AssetID, asset),
		))
	}
}

func collectWorkDetailSectionAttachments(collector *workCustomerAttachmentCollector, sections []map[string]any) {
	for _, section := range sections {
		source := firstText(section, "name")
		for _, field := range mapListFromAny(section["fields"]) {
			collector.Append(field["files"], source, firstText(field, "label"))
		}
	}
}

func workCustomerStoredAttachments(ctx context.Context, access *workCustomerDetailAccess, collector *workCustomerAttachmentCollector) {
	if access == nil || collector == nil {
		return
	}
	rows := crmmodel.NewAttachmentModel().Select(ctx, map[string]any{
		"customer_id": access.CustomerID,
		"asset_id":    workCustomerDetailAssetIDs(access),
	}, map[string]any{"order": "created_at desc,id desc"})
	for _, row := range rows {
		if row == nil || row.UploadFileID == 0 {
			continue
		}
		source, fieldLabel := workStoredAttachmentPresentation(row)
		collector.Append(
			workUploadFilePayloads(ctx, []uint64{row.UploadFileID}),
			source,
			fieldLabel,
		)
	}
}

func workStoredAttachmentPresentation(row *crmmodel.Attachment) (string, string) {
	if row == nil {
		return "业务附件", "附件"
	}
	if row.Usage == crmmodel.AttachmentUsageArrivalVideo {
		return "预约及到访", "到访视频"
	}
	if row.ScheduleEventID > 0 {
		return "日程记录", "附件"
	}
	if row.TaskID > 0 || row.OperationLogID > 0 {
		return "流程任务", "附件"
	}
	return "业务附件", "附件"
}

func workCustomerOperationAttachments(ctx context.Context, access *workCustomerDetailAccess, collector *workCustomerAttachmentCollector) {
	if access == nil || collector == nil {
		return
	}
	filter := map[string]any{
		"customer_id": access.CustomerID,
		"asset_id":    workCustomerDetailAssetIDs(access),
	}
	rows := crmmodel.NewOperationLogModel().SelectMap(ctx, filter)
	rows = workBusinessOperationRows(ctx, rows)
	for _, row := range rows {
		source := firstText(row, "title")
		if source == "" {
			source = "流程记录"
		}
		for _, item := range workOperationSummaryItems(ctx, row) {
			collector.Append(item["files"], source, firstText(item, "label"))
		}
	}
}

func workCustomerDetailAssetIDs(access *workCustomerDetailAccess) []uint64 {
	assetIDs := []uint64{0}
	if access != nil && access.AssetID > 0 {
		assetIDs = append(assetIDs, access.AssetID)
	}
	return assetIDs
}
