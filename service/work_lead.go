package service

import (
	"context"
	"fmt"
	"sort"
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
	workflow := workflowForSubject(ctx, firstUint64(payload, "workflow_id", "workflowId"), crmmodel.WorkflowSubjectLead)
	if workflow == nil || !canAccessWorkflow(ctx, staff, workflow) {
		return map[string]any{
			"enabled":         false,
			"can_create":      false,
			"pending_count":   0,
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
	status := firstText(payload, "status")
	if status != "" && validWorkLeadStatus(status) {
		filter["status"] = status
	}
	leads := crmmodel.NewLeadModel().Select(ctx, filter, map[string]any{"order": "id desc"})
	sort.SliceStable(leads, func(i, j int) bool {
		leftPending := leads[i] != nil && leads[i].Status == crmmodel.LeadStatusPending
		rightPending := leads[j] != nil && leads[j].Status == crmmodel.LeadStatusPending
		return leftPending && !rightPending
	})
	keyword := firstText(payload, "keyword")
	quickFilter := firstText(payload, "quick_filter", "quickFilter")
	stageFilter := firstText(payload, "stage_filter", "stage")
	taskFilter := firstText(payload, "task_filter", "task")
	rows := make([]map[string]any, 0, len(leads))
	for _, lead := range leads {
		if lead == nil || !matchesWorkLeadKeyword(lead, keyword) {
			continue
		}
		instance := workflowInstanceForLead(ctx, lead.ID, workflow.ID)
		if instance == nil || !workflowInstanceMatchesPersonalQuickFilter(ctx, staff, instance, quickFilter) {
			continue
		}
		if !canViewWorkflowInstance(ctx, staff, instance) && quickFilter != "completedToday" {
			continue
		}
		if !workflowInstanceMatchesSummaryFilters(ctx, staff, instance, stageFilter, taskFilter) {
			continue
		}
		row := workLeadRow(ctx, lead, workflow.ID)
		row["flow"] = workLeadFlowDetail(ctx, staff, instance)
		rows = append(rows, row)
	}

	page, pageSize, start, end := workLeadPageBounds(len(rows), payload)
	pageRows := []map[string]any{}
	if start < len(rows) {
		pageRows = rows[start:end]
	}
	pendingCounts := currentWorkPersonalWorkload(ctx, staff).pendingTaskCountByWorkflow()
	return map[string]any{
		"enabled":         true,
		"can_create":      canCreateLeadInWorkflow(ctx, staff, workflow),
		"pending_count":   pendingCounts[workflow.ID],
		"list":            pageRows,
		"total":           len(rows),
		"page":            page,
		"page_size":       pageSize,
		"sources":         workLeadSourceOptions(ctx),
		"channels":        workLeadChannelOptions(ctx),
		"invalid_reasons": workLeadInvalidReasonOptions(ctx),
		"statuses":        workLeadStatusOptions(),
		"templates":       workLeadTemplateRows(ctx),
		"workflow": map[string]any{
			"id":           workflow.ID,
			"name":         workflow.Name,
			"subject_type": workflow.SubjectType,
		},
	}, nil
}

func (WorkService) RegisterLead(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	workflow := workflowForSubject(ctx, firstUint64(payload, "workflow_id", "workflowId"), crmmodel.WorkflowSubjectLead)
	if workflow == nil || !canCreateLeadInWorkflow(ctx, staff, workflow) {
		return nil, fmt.Errorf("只有线索流程首阶段负责部门或流程调度员可以录入线索")
	}
	requestedOwnerID := firstUint64(payload, "owner_staff_id", "ownerStaffId")
	if requestedOwnerID == 0 {
		requestedOwnerID = staff.ID
	}
	created, err := createWorkLead(ctx, workflow, staff, requestedOwnerID, payload)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"success": true,
		"lead":    workLeadRow(ctx, created, workflow.ID),
	}, nil
}

func createWorkLead(
	ctx context.Context,
	workflow *crmmodel.Workflow,
	creator *WorkStaffSession,
	requestedOwnerID uint64,
	payload map[string]any,
) (*crmmodel.Lead, error) {
	if workflow == nil || workflow.SubjectType != crmmodel.WorkflowSubjectLead {
		return nil, fmt.Errorf("线索流程不存在或已停用")
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
	creatorID := uint64(0)
	creatorDepartmentID := uint64(0)
	if creator != nil {
		creatorID = creator.ID
		creatorDepartmentID = creator.DepartmentID
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
			"owner_department_id":   creatorDepartmentID,
			"owner_staff_id":        creatorID,
			"created_by_staff_id":   creatorID,
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
		if _, err := startWorkflowInstance(txCtx, leadWorkflowSubject(leadID), workflow.ID, requestedOwnerID); err != nil {
			return err
		}
		instance := activeWorkflowInstanceForLead(txCtx, leadID, workflow.ID)
		if instance == nil {
			return fmt.Errorf("线索流程启动失败")
		}
		crmmodel.NewLeadModel().Update(txCtx, map[string]any{"id": leadID}, map[string]any{
			"owner_department_id": instance.OwnerDepartmentID,
			"owner_staff_id":      instance.OwnerStaffID,
			"updated_at":          now,
		})
		created.OwnerDepartmentID = instance.OwnerDepartmentID
		created.OwnerStaffID = instance.OwnerStaffID
		if status == crmmodel.LeadStatusDuplicate {
			if err := terminateActiveWorkflowInstance(txCtx, creator, instance, duplicateReason); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (WorkService) ActOnLead(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	workflow := workflowForSubject(ctx, firstUint64(payload, "workflow_id", "workflowId"), crmmodel.WorkflowSubjectLead)
	if workflow == nil || !canAccessWorkflow(ctx, staff, workflow) {
		return nil, fmt.Errorf("无权处理该线索流程")
	}
	leadID := firstUint64(payload, "lead_id", "leadId", "id")
	if leadID == 0 {
		return nil, fmt.Errorf("请选择线索")
	}
	action := firstText(payload, "action")
	var result map[string]any
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		lead := crmmodel.NewLeadModel().Find(txCtx, map[string]any{"id": leadID})
		instance := workflowInstanceForLead(txCtx, leadID, workflow.ID)
		if lead == nil || instance == nil || !canViewWorkflowInstance(txCtx, staff, instance) {
			return fmt.Errorf("线索不存在或无权操作")
		}
		if !canManageLeadWorkflow(staff, instance) {
			return fmt.Errorf("只有当前负责人或流程调度员可以处理线索")
		}
		var err error
		switch action {
		case "update":
			err = updateWorkLead(txCtx, staff, lead, instance, payload)
		case "invalid":
			err = invalidateWorkLead(txCtx, lead, payload)
			if err == nil {
				refreshed := crmmodel.NewLeadModel().Find(txCtx, map[string]any{"id": lead.ID})
				err = terminateLeadWorkflow(txCtx, staff, instance, leadTerminationReason(txCtx, refreshed))
			}
		case "duplicate":
			err = markWorkLeadDuplicate(txCtx, lead)
			if err == nil {
				refreshed := crmmodel.NewLeadModel().Find(txCtx, map[string]any{"id": lead.ID})
				err = terminateLeadWorkflow(txCtx, staff, instance, leadTerminationReason(txCtx, refreshed))
			}
		case "reopen":
			err = reopenWorkLead(txCtx, lead)
			if err == nil {
				refreshed := crmmodel.NewLeadModel().Find(txCtx, map[string]any{"id": lead.ID})
				if refreshed != nil && refreshed.Status == crmmodel.LeadStatusPending {
					err = reopenLeadWorkflow(txCtx, refreshed, workflow, payload)
				}
			}
		case "convert":
			ownerStaffID := firstUint64(payload, "next_owner_staff_id", "nextOwnerStaffId", "owner_staff_id", "ownerStaffId")
			result, err = convertWorkLead(txCtx, staff, lead, instance, workflow.ID, ownerStaffID)
		default:
			err = fmt.Errorf("不支持的线索操作")
		}
		if err != nil {
			return err
		}
		if result == nil {
			refreshed := crmmodel.NewLeadModel().Find(txCtx, map[string]any{"id": lead.ID})
			result = map[string]any{"success": true, "lead": workLeadRow(txCtx, refreshed, workflow.ID)}
		}
		return nil
	})
	return result, err
}

func updateWorkLead(
	ctx context.Context,
	staff *WorkStaffSession,
	lead *crmmodel.Lead,
	instance *crmmodel.WorkflowInstance,
	payload map[string]any,
) error {
	if lead == nil || instance == nil {
		return fmt.Errorf("线索不存在")
	}
	if lead.CustomerID > 0 {
		return fmt.Errorf("已转化线索不能编辑")
	}
	if lead.Status != crmmodel.LeadStatusPending {
		return fmt.Errorf("只有待处理线索可以编辑")
	}
	if staff == nil || staff.ID == 0 || instance.OwnerStaffID != staff.ID && !staff.CanDispatch {
		return fmt.Errorf("只有当前负责人或流程调度员可以编辑线索")
	}
	name := firstText(payload, "name")
	if name == "" {
		return fmt.Errorf("请填写线索姓名")
	}
	formInput := emptyWorkFormInput()
	formInput.leadFields = map[string]any{
		"name":         name,
		"phone":        firstText(payload, "phone", "mobile"),
		"wechat":       firstText(payload, "wechat"),
		"source_id":    firstUint64(payload, "source_id", "sourceId"),
		"channel_id":   firstUint64(payload, "channel_id", "channelId"),
		"external_id":  firstText(payload, "external_id", "externalId"),
		"city":         firstText(payload, "city"),
		"initial_need": firstText(payload, "initial_need", "initialNeed", "need"),
	}
	formInput.leadDataRecords = workLeadEditDataRecords(ctx, payload)
	return saveWorkLeadFormInput(ctx, lead.ID, formInput)
}

func workLeadEditDataRecords(ctx context.Context, payload map[string]any) map[uint64]map[string]any {
	input := mapFromAny(firstPresent(payload, "data_values", "dataValues"))
	if len(input) == 0 {
		return map[uint64]map[string]any{}
	}
	records := map[uint64]map[string]any{}
	for _, template := range workLeadTemplateRows(ctx) {
		templateID := inputUint64(template["id"])
		for _, field := range mapListFromAny(template["fields"]) {
			fieldID := inputUint64(field["id"])
			key := fmt.Sprintf("data:%d", fieldID)
			value, exists := input[key]
			if templateID == 0 || fieldID == 0 || !exists {
				continue
			}
			if records[templateID] == nil {
				records[templateID] = map[string]any{}
			}
			if emptyWorkFieldValue(value) {
				records[templateID][fmt.Sprintf("%d", fieldID)] = ""
				continue
			}
			if normalized, ok := normalizeWorkLeadFieldValue(field, value); ok {
				records[templateID][fmt.Sprintf("%d", fieldID)] = normalized
			}
		}
	}
	return records
}

func invalidateWorkLead(ctx context.Context, lead *crmmodel.Lead, payload map[string]any) error {
	if lead.Status == crmmodel.LeadStatusConverted || lead.CustomerID > 0 {
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
	if lead.Status == crmmodel.LeadStatusConverted || lead.CustomerID > 0 {
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
	if lead.CustomerID > 0 {
		return fmt.Errorf("已转化线索不能恢复")
	}
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

func convertWorkLead(
	ctx context.Context,
	staff *WorkStaffSession,
	lead *crmmodel.Lead,
	leadInstance *crmmodel.WorkflowInstance,
	leadWorkflowID uint64,
	ownerStaffID uint64,
) (map[string]any, error) {
	if lead.CustomerID > 0 {
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
			"lead":        workLeadRow(ctx, lead, leadWorkflowID),
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
		if err := terminateLeadWorkflow(ctx, staff, leadInstance, duplicate.Reason); err != nil {
			return nil, err
		}
		return map[string]any{
			"success":   true,
			"converted": false,
			"duplicate": true,
			"message":   duplicate.Reason,
			"lead":      workLeadRow(ctx, crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": lead.ID}), leadWorkflowID),
		}, nil
	}
	if ownerStaffID == 0 {
		return nil, fmt.Errorf("请选择签约流程首阶段负责人")
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
	if err := completeLeadWorkflow(ctx, staff, leadInstance); err != nil {
		return nil, err
	}
	refreshed := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": lead.ID})
	if progress := currentWorkEntryInstance(ctx, customerID, assetID); progress != nil {
		conversionSnapshot := map[string]any{
			"lead_id":     lead.ID,
			"customer_id": customerID,
			"asset_id":    assetID,
		}
		if summaryItems := workLeadConversionSummaryItems(ctx, lead.ID, customerID, assetID); len(summaryItems) > 0 {
			conversionSnapshot["summary_items"] = summaryItems
		}
		recordWorkManagementOperation(ctx, staff, progress, workBusinessEventLeadConverted, "线索已转为客户", lead.Code, conversionSnapshot)
	}
	return map[string]any{
		"success":     true,
		"converted":   true,
		"customer_id": customerID,
		"asset_id":    assetID,
		"lead":        workLeadRow(ctx, refreshed, leadWorkflowID),
	}, nil
}

func terminateLeadWorkflow(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance, reason string) error {
	if instance == nil || instance.Status != crmmodel.ProgressStatusActive {
		return nil
	}
	if staff == nil || staff.ID == 0 || instance.OwnerStaffID != staff.ID && !staff.CanDispatch {
		return fmt.Errorf("只有当前负责人或流程调度员可以处理线索")
	}
	return terminateActiveWorkflowInstance(ctx, staff, instance, reason)
}

func completeLeadWorkflow(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) error {
	if instance == nil || instance.Status != crmmodel.ProgressStatusActive || instance.LeadID == 0 {
		return fmt.Errorf("线索流程已结束")
	}
	if staff == nil || staff.ID == 0 || instance.OwnerStaffID != staff.ID && !staff.CanDispatch {
		return fmt.Errorf("只有当前负责人或流程调度员可以转化线索")
	}
	if nextEnabledStage(ctx, instance.WorkflowID, instance.StageID) != nil {
		return fmt.Errorf("线索尚未进入流程最后阶段，不能转为客户")
	}
	todos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"stage_id":             instance.StageID,
		"status":               crmmodel.WorkTodoStatusPending,
	}, map[string]any{"order": "id asc"})
	now := time.Now()
	for _, todo := range todos {
		if todo == nil {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
		if task == nil {
			return fmt.Errorf("线索任务配置不存在")
		}
		if todo.Required && (task.TaskType != crmmodel.TaskTypeTodo || todo.AssigneeStaffID != staff.ID && !staff.CanDispatch) {
			return fmt.Errorf("必做任务“%s”尚未完成", task.Name)
		}
		if !todo.Required {
			continue
		}
		if crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
			"id":     todo.ID,
			"status": crmmodel.WorkTodoStatusPending,
		}, map[string]any{
			"status":       crmmodel.WorkTodoStatusDone,
			"result":       "线索已确认并转化",
			"completed_at": now,
			"updated_at":   now,
		}) == 0 {
			return fmt.Errorf("线索任务已变化，请刷新后重试")
		}
		todo.Status = crmmodel.WorkTodoStatusDone
		if recordWorkTaskOperation(ctx, staff, todo, task, "completed", "线索已确认并转化", map[string]any{
			"lead_id": instance.LeadID,
		}, false) == 0 {
			return fmt.Errorf("线索任务记录创建失败")
		}
	}
	cancelPendingOptionalTodos(ctx, instance)
	return completeWorkflowInstance(ctx, staff, instance)
}

func reopenLeadWorkflow(ctx context.Context, lead *crmmodel.Lead, workflow *crmmodel.Workflow, payload map[string]any) error {
	if lead == nil || workflow == nil {
		return fmt.Errorf("线索或流程不存在")
	}
	ownerStaffID := firstUint64(payload, "owner_staff_id", "ownerStaffId")
	instance, err := startWorkflowInstance(ctx, leadWorkflowSubject(lead.ID), workflow.ID, ownerStaffID)
	if err != nil {
		return err
	}
	crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": lead.ID}, map[string]any{
		"owner_department_id": instance.OwnerDepartmentID,
		"owner_staff_id":      instance.OwnerStaffID,
		"updated_at":          time.Now(),
	})
	return nil
}

func leadTerminationReason(ctx context.Context, lead *crmmodel.Lead) string {
	if lead == nil {
		return "线索已终止"
	}
	switch lead.Status {
	case crmmodel.LeadStatusInvalid:
		reason := "无效线索"
		if invalidReason := crmmodel.NewLeadInvalidReasonModel().Find(ctx, map[string]any{"id": lead.InvalidReasonID}); invalidReason != nil {
			reason += "：" + invalidReason.Name
		}
		if note := strings.TrimSpace(lead.InvalidNote); note != "" {
			reason += "（" + note + "）"
		}
		return reason
	case crmmodel.LeadStatusDuplicate:
		if reason := strings.TrimSpace(lead.DuplicateReason); reason != "" {
			return reason
		}
		return "重复线索"
	default:
		return "线索已终止"
	}
}

func findWorkLeadDuplicate(ctx context.Context, leadID uint64, phone, wechat string, sourceID uint64, externalID string) *workLeadDuplicate {
	customerModel := crmmodel.NewCustomerModel()
	if phone != "" {
		if customer := customerModel.Find(ctx, map[string]any{"phone": phone}); customer != nil {
			return &workLeadDuplicate{
				CustomerID: customer.ID,
				Reason:     "手机号已存在于客户库：" + customerCodeDisplayForWork(ctx, customer.Code),
			}
		}
	}
	if wechat != "" {
		if customer := customerModel.Find(ctx, map[string]any{"wechat": wechat}); customer != nil {
			return &workLeadDuplicate{
				CustomerID: customer.ID,
				Reason:     "微信号已存在于客户库：" + customerCodeDisplayForWork(ctx, customer.Code),
			}
		}
	}

	leadModel := crmmodel.NewLeadModel()
	for _, candidate := range leadModel.Select(ctx, map[string]any{}, map[string]any{"order": "id asc"}) {
		if candidate == nil || candidate.ID == leadID ||
			candidate.Status == crmmodel.LeadStatusInvalid ||
			candidate.Status == crmmodel.LeadStatusDuplicate {
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

func workLeadRow(ctx context.Context, lead *crmmodel.Lead, workflowIDs ...uint64) map[string]any {
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
	if len(workflowIDs) > 0 && workflowIDs[0] > 0 {
		attachWorkLeadWorkflow(ctx, row, lead.ID, workflowIDs[0])
	}
	return row
}

func attachWorkLeadWorkflow(ctx context.Context, row map[string]any, leadID, workflowID uint64) {
	instance := workflowInstanceForLead(ctx, leadID, workflowID)
	if instance == nil {
		return
	}
	row["workflow_instance_id"] = instance.ID
	row["workflow_id"] = instance.WorkflowID
	row["stage_id"] = instance.StageID
	row["workflow_status"] = instance.Status
	row["workflow_owner_department_id"] = instance.OwnerDepartmentID
	row["workflow_owner_staff_id"] = instance.OwnerStaffID
	row["pending_task_count"] = crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"status":               crmmodel.WorkTodoStatusPending,
	})
	if workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": instance.WorkflowID}); workflow != nil {
		row["workflow_name"] = workflow.Name
	}
	if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": instance.StageID}); stage != nil {
		row["stage_name"] = stage.Name
	}
	if owner := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": instance.OwnerStaffID}); owner != nil {
		row["owner_staff_id"] = owner.ID
		row["owner_staff_name"] = owner.Name
	}
	if department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": instance.OwnerDepartmentID}); department != nil {
		row["owner_department_name"] = department.Name
	}
}

func workLeadFlowDetail(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) map[string]any {
	if instance == nil {
		return map[string]any{}
	}
	flow := workFlowDetail(ctx, staff, instance.ID)
	flow["lead_id"] = instance.LeadID
	flow["flow_role"] = "lead"
	flow["tasks"] = workCurrentStageTodoRows(ctx, staff, instance, true)
	return flow
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
