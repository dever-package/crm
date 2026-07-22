package service

import (
	"context"
	"fmt"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

type dispatchActiveLeadTarget struct {
	instance *crmmodel.WorkflowInstance
	lead     *crmmodel.Lead
}

func (WorkService) DispatchActiveLeads(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	department, _, err := dispatchDepartmentScope(ctx, staff, firstUint64(payload, "department_id", "departmentId"))
	if err != nil {
		return nil, err
	}
	ownerStaffID := firstUint64(payload, "owner_staff_id", "ownerStaffId")
	if ownerStaffID > 0 && crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"id":            ownerStaffID,
		"department_id": department.ID,
	}) == nil {
		return nil, fmt.Errorf("所选原负责人不属于当前部门")
	}
	filters := map[string]any{
		"owner_department_id": department.ID,
		"status":              crmmodel.ProgressStatusActive,
	}
	if ownerStaffID > 0 {
		filters["owner_staff_id"] = ownerStaffID
	}
	keyword := firstText(payload, "keyword")
	leadsByID := map[uint64]*crmmodel.Lead{}
	for _, lead := range crmmodel.NewLeadModel().Select(ctx, map[string]any{}) {
		if lead != nil && matchesWorkLeadKeyword(lead, keyword) {
			leadsByID[lead.ID] = lead
		}
	}
	targets := make([]dispatchActiveLeadTarget, 0)
	for _, instance := range crmmodel.NewWorkflowInstanceModel().Select(ctx, filters, map[string]any{"order": "updated_at desc,id desc"}) {
		if instance == nil || instance.LeadID == 0 {
			continue
		}
		lead := leadsByID[instance.LeadID]
		if lead == nil {
			continue
		}
		targets = append(targets, dispatchActiveLeadTarget{instance: instance, lead: lead})
	}
	page, pageSize, start, end := workLeadPageBounds(len(targets), payload)
	pageTargets := []dispatchActiveLeadTarget{}
	if start < len(targets) {
		pageTargets = targets[start:end]
	}
	rows := make([]map[string]any, 0, len(pageTargets))
	stageNames := map[uint64]string{}
	ownerNames := map[uint64]string{}
	for _, target := range pageTargets {
		rows = append(rows, dispatchActiveLeadRow(ctx, target, stageNames, ownerNames))
	}
	return map[string]any{
		"department_id": department.ID,
		"list":          rows,
		"owner_options": dispatchDepartmentOwnerOptions(ctx, department.ID),
		"total":         len(targets),
		"page":          page,
		"page_size":     pageSize,
	}, nil
}

func dispatchDepartmentOwnerOptions(ctx context.Context, departmentID uint64) []map[string]any {
	staffRows := crmmodel.NewStaffModel().Select(ctx, map[string]any{
		"department_id": departmentID,
	}, map[string]any{"order": "id asc"})
	options := make([]map[string]any, 0, len(staffRows))
	for _, staff := range staffRows {
		if staff == nil {
			continue
		}
		options = append(options, map[string]any{
			"id":     staff.ID,
			"name":   staff.Name,
			"status": staff.Status,
		})
	}
	return options
}

func dispatchActiveLeadRow(
	ctx context.Context,
	target dispatchActiveLeadTarget,
	stageNames map[uint64]string,
	ownerNames map[uint64]string,
) map[string]any {
	instance := target.instance
	lead := target.lead
	row := map[string]any{
		"workflow_instance_id": instance.ID,
		"lead_id":              lead.ID,
		"lead_name":            lead.Name,
		"lead_code":            lead.Code,
		"phone":                lead.Phone,
		"stage_id":             instance.StageID,
		"stage_name":           "",
		"owner_staff_id":       instance.OwnerStaffID,
		"owner_staff_name":     "",
		"started_at":           instance.StartedAt,
		"updated_at":           instance.UpdatedAt,
	}
	stageName, stageLoaded := stageNames[instance.StageID]
	if !stageLoaded {
		if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": instance.StageID}); stage != nil {
			stageName = stage.Name
		}
		stageNames[instance.StageID] = stageName
	}
	row["stage_name"] = stageName
	ownerName, ownerLoaded := ownerNames[instance.OwnerStaffID]
	if !ownerLoaded {
		if owner := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": instance.OwnerStaffID}); owner != nil {
			ownerName = owner.Name
		}
		ownerNames[instance.OwnerStaffID] = ownerName
	}
	row["owner_staff_name"] = ownerName
	return row
}

func (WorkService) BatchReassignDispatchLeads(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	department, _, err := dispatchDepartmentScope(ctx, staff, firstUint64(payload, "department_id", "departmentId"))
	if err != nil {
		return nil, err
	}
	instanceIDs := uint64ListFromAny(firstPresent(payload, "workflow_instance_ids", "workflowInstanceIds", "ids"))
	if len(instanceIDs) == 0 {
		return nil, fmt.Errorf("请至少选择一条在办线索")
	}
	if len(instanceIDs) > 500 {
		return nil, fmt.Errorf("单次最多改派500条在办线索")
	}
	ownerStaffID := firstUint64(payload, "owner_staff_id", "ownerStaffId", "staff_id", "staffId")
	if enabledStaffInDepartment(ctx, ownerStaffID, department.ID) == nil {
		return nil, fmt.Errorf("所选负责人不属于当前部门或已停用")
	}
	changedCount := 0
	err = orm.Transaction(ctx, func(txCtx context.Context) error {
		for _, instanceID := range instanceIDs {
			instance, activeErr := activeWorkflowInstance(txCtx, instanceID)
			if activeErr != nil || instance.LeadID == 0 || instance.OwnerDepartmentID != department.ID {
				return fmt.Errorf("在办线索已变化或不属于当前部门，请刷新后重试")
			}
			previousStaffID := instance.OwnerStaffID
			if _, changeErr := changeWorkflowInstanceOwner(txCtx, staff, instance, ownerStaffID); changeErr != nil {
				return changeErr
			}
			if previousStaffID != ownerStaffID {
				changedCount++
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"success":        true,
		"selected_count": len(instanceIDs),
		"changed_count":  changedCount,
	}, nil
}
