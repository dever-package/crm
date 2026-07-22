package service

import (
	"context"
	"fmt"
	"sort"

	crmmodel "github.com/dever-package/crm/model"
)

// LeadDetail exposes downstream customer progress through a lead the current
// staff can already view. It deliberately does not grant general customer access.
func (WorkService) LeadDetail(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	leadID := firstUint64(payload, "lead_id", "leadId")
	lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": leadID})
	if lead == nil {
		return nil, fmt.Errorf("线索不存在")
	}
	workflow := workflowForSubject(ctx, firstUint64(payload, "workflow_id", "workflowId"), crmmodel.WorkflowSubjectLead)
	instance := (*crmmodel.WorkflowInstance)(nil)
	if workflow != nil {
		instance = workflowInstanceForLead(ctx, lead.ID, workflow.ID)
	}
	if instance == nil || !canViewWorkflowInstance(ctx, staff, instance) {
		return nil, fmt.Errorf("无权查看该线索")
	}

	leadRow := workLeadRow(ctx, lead, workflow.ID)
	attachWorkLeadDispatchAssignees(ctx, []map[string]any{leadRow})
	leadRow["flow"] = workLeadFlowDetail(ctx, staff, instance)
	detailSections := workDataDetailSections(
		ctx,
		crmmodel.DataTemplateTargetLead,
		crmmodel.LeadDataTemplateCateID,
		workLeadDataValues(lead),
	)
	result := map[string]any{
		"lead":            leadRow,
		"customer":        nil,
		"operations":      workLeadDetailOperationRows(ctx, staff, instance.ID, lead.CustomerID),
		"detail_sections": detailSections,
	}
	if lead.CustomerID == 0 {
		return result, nil
	}

	customer := workLeadDetailCustomer(ctx, lead.CustomerID, leadRow)
	if len(customer) == 0 {
		return result, nil
	}
	result["customer"] = customer
	result["detail_sections"] = append(detailSections, workDataDetailSections(
		ctx,
		crmmodel.DataTemplateTargetCustomer,
		crmmodel.CustomerDataTemplateCateID,
		mapFromAny(customer["data_values"]),
	)...)
	return result, nil
}

func workLeadDetailCustomer(ctx context.Context, customerID uint64, sourceLead map[string]any) map[string]any {
	customer := crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
	if len(customer) == 0 {
		return map[string]any{}
	}
	attachWorkEntityDataValues(
		ctx,
		customer,
		workCustomerFormValues(ctx, customerID, 0, customer),
		crmmodel.CustomerDataTemplateCateID,
	)
	attachWorkCustomerTagIDs(ctx, customer, customerID)
	enrichWorkCustomerRow(ctx, customer)
	customer["source_lead"] = sourceLead
	attachWorkLeadCustomerOwner(ctx, customer, currentWorkEntryInstance(ctx, customerID, 0))
	return customer
}

func attachWorkLeadCustomerOwner(ctx context.Context, customer map[string]any, instance *crmmodel.WorkflowInstance) {
	if len(customer) == 0 || instance == nil {
		return
	}
	attachWorkStageFields(ctx, customer, instance)
	customer["current_stage_name"] = workStageName(ctx, instance.StageID)
	customer["current_owner_staff_id"] = instance.OwnerStaffID
	customer["current_owner_department_id"] = instance.OwnerDepartmentID
	if owner := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": instance.OwnerStaffID}); owner != nil {
		customer["current_owner_staff_name"] = owner.Name
	}
	if department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": instance.OwnerDepartmentID}); department != nil {
		customer["current_owner_department_name"] = department.Name
	}
}

func workLeadDetailOperationRows(
	ctx context.Context,
	staff *WorkStaffSession,
	leadInstanceID uint64,
	customerID uint64,
) []map[string]any {
	rows := crmmodel.NewOperationLogModel().SelectMap(ctx, map[string]any{
		"workflow_instance_id": leadInstanceID,
	})
	if customerID > 0 {
		rows = append(rows, crmmodel.NewOperationLogModel().SelectMap(ctx, map[string]any{
			"customer_id": customerID,
		})...)
	}
	rows = uniqueWorkLeadOperationRows(rows)
	rows = workBusinessOperationRows(ctx, rows)
	enrichWorkOperationRows(ctx, staff, rows)
	return rows
}

func uniqueWorkLeadOperationRows(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	seen := map[uint64]bool{}
	for _, row := range rows {
		id := inputUint64(row["id"])
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, row)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return inputUint64(result[i]["id"]) > inputUint64(result[j]["id"])
	})
	return result
}
