package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func (WorkService) LeadDispatchConfig(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	scope, scopes, err := manageableLeadDispatchScope(ctx, staff, firstUint64(payload, "workflow_id", "workflowId"))
	if err != nil {
		return nil, err
	}
	if _, _, err := ensureDepartmentDispatchSetting(ctx, scope.TargetDepartment.ID); err != nil {
		return nil, err
	}
	return leadDispatchConfigPayload(ctx, staff, scope, scopes), nil
}

func (WorkService) SaveLeadDispatchConfig(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	scope, scopes, err := manageableLeadDispatchScope(ctx, staff, firstUint64(payload, "workflow_id", "workflowId"))
	if err != nil {
		return nil, err
	}
	enabled := booleanFromAny(firstPresent(payload, "auto_handoff_enabled", "autoHandoffEnabled", "enabled"))
	err = saveDepartmentDispatchConfiguration(ctx, scope.TargetDepartment.ID, payload, func(txCtx context.Context) error {
		return saveLeadDispatchRoute(txCtx, scope.Workflow.ID, enabled)
	})
	if err != nil {
		return nil, err
	}
	leadRetry, leadRetryErr := retryLeadDispatchWorkflow(ctx, scope.Workflow.ID)
	_, departmentRetryErr := RetryPendingDepartmentDispatch(ctx, scope.TargetDepartment.ID)
	result := leadDispatchConfigPayload(ctx, staff, scope, scopes)
	result["lead_retry"] = leadRetry
	retryWarnings := make([]string, 0, 2)
	if leadRetryErr != nil {
		retryWarnings = append(retryWarnings, leadRetryErr.Error())
	}
	if departmentRetryErr != nil {
		retryWarnings = append(retryWarnings, departmentRetryErr.Error())
	}
	if len(retryWarnings) > 0 {
		result["retry_warning"] = strings.Join(retryWarnings, "；")
	}
	return result, nil
}

func (WorkService) CreateLeadDispatchGroup(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	scope, scopes, err := manageableLeadDispatchScope(ctx, staff, firstUint64(payload, "workflow_id", "workflowId"))
	if err != nil {
		return nil, err
	}
	name, err := dispatchGroupName(payload)
	if err != nil {
		return nil, err
	}
	if err := createDepartmentDispatchGroup(ctx, scope.TargetDepartment.ID, name); err != nil {
		return nil, err
	}
	return leadDispatchConfigPayload(ctx, staff, scope, scopes), nil
}

func (WorkService) RenameLeadDispatchGroup(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	scope, scopes, err := manageableLeadDispatchScope(ctx, staff, firstUint64(payload, "workflow_id", "workflowId"))
	if err != nil {
		return nil, err
	}
	name, err := dispatchGroupName(payload)
	if err != nil {
		return nil, err
	}
	poolID := firstUint64(payload, "pool_id", "poolId")
	if poolID == 0 {
		return nil, fmt.Errorf("请选择工作组")
	}
	if err := renameDepartmentDispatchGroup(ctx, scope.TargetDepartment.ID, poolID, name); err != nil {
		return nil, err
	}
	return leadDispatchConfigPayload(ctx, staff, scope, scopes), nil
}

func (WorkService) DeleteLeadDispatchGroup(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	scope, scopes, err := manageableLeadDispatchScope(ctx, staff, firstUint64(payload, "workflow_id", "workflowId"))
	if err != nil {
		return nil, err
	}
	poolID := firstUint64(payload, "pool_id", "poolId")
	if poolID == 0 {
		return nil, fmt.Errorf("请选择工作组")
	}
	if err := deleteDepartmentDispatchGroup(ctx, scope.TargetDepartment.ID, poolID); err != nil {
		return nil, err
	}
	return leadDispatchConfigPayload(ctx, staff, scope, scopes), nil
}

func (WorkService) AssignLeadDispatch(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	handoffIDs := uint64ListFromAny(firstPresent(payload, "handoff_ids", "handoffIds", "ids"))
	if handoffID := firstUint64(payload, "handoff_id", "handoffId", "id"); handoffID > 0 {
		handoffIDs = append(handoffIDs, handoffID)
	}
	assigned, err := AssignLeadDispatchHandoffs(
		ctx,
		staff,
		handoffIDs,
		firstUint64(payload, "assignee_staff_id", "assigneeStaffId", "staff_id", "staffId"),
	)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"success":        true,
		"selected_count": len(handoffIDs),
		"assigned_count": assigned,
	}, nil
}

func saveLeadDispatchRoute(ctx context.Context, workflowID uint64, enabled bool) error {
	if workflowID == 0 {
		return fmt.Errorf("线索流程不能为空")
	}
	status := crmmodel.StatusDisabled
	if enabled {
		status = crmmodel.StatusEnabled
	}
	model := crmmodel.NewLeadDispatchRouteModel()
	now := time.Now()
	route := model.Find(ctx, map[string]any{"workflow_id": workflowID})
	if route == nil {
		if model.Insert(ctx, map[string]any{
			"workflow_id": workflowID,
			"status":      status,
			"created_at":  now,
			"updated_at":  now,
		}) == 0 {
			return fmt.Errorf("线索自动派单配置创建失败")
		}
		return nil
	}
	if model.Update(ctx, map[string]any{
		"id":          route.ID,
		"workflow_id": workflowID,
	}, map[string]any{
		"status":     status,
		"updated_at": now,
	}) == 0 {
		return fmt.Errorf("线索自动派单配置已变化，请刷新后重试")
	}
	return nil
}

func leadDispatchConfigPayload(
	ctx context.Context,
	staff *WorkStaffSession,
	scope *leadDispatchScope,
	scopes []*leadDispatchScope,
) map[string]any {
	result := departmentDispatchConfigPayload(
		ctx,
		staff,
		scope.TargetDepartment,
		[]*crmmodel.Department{scope.TargetDepartment},
	)
	routeRows := make([]map[string]any, 0, len(scopes))
	for _, current := range scopes {
		if current == nil {
			continue
		}
		routeRows = append(routeRows, leadDispatchScopeRow(current))
	}
	result["workflow_id"] = scope.Workflow.ID
	result["workflow_name"] = scope.Workflow.Name
	result["source_stage_id"] = scope.SourceStage.ID
	result["source_stage_name"] = scope.SourceStage.Name
	result["source_department_id"] = scope.SourceDepartment.ID
	result["source_department_name"] = scope.SourceDepartment.Name
	result["target_workflow_id"] = scope.TargetWorkflow.ID
	result["target_workflow_name"] = scope.TargetWorkflow.Name
	result["target_stage_id"] = scope.TargetStage.ID
	result["target_stage_name"] = scope.TargetStage.Name
	result["target_department_id"] = scope.TargetDepartment.ID
	result["target_department_name"] = scope.TargetDepartment.Name
	result["routes"] = routeRows
	result["auto_handoff_enabled"] = leadDispatchRouteEnabled(ctx, scope.Workflow.ID)
	result["pending"] = pendingLeadDispatchRows(ctx, scope.Workflow.ID)
	result["pending_count"] = pendingLeadDispatchCount(ctx, scope.Workflow.ID)
	result["assignee_options"] = configuredDispatchPoolStaffRows(ctx, scope.TargetDepartment.ID)
	delete(result, "departments")
	return result
}

func leadDispatchScopeRow(scope *leadDispatchScope) map[string]any {
	return map[string]any{
		"workflow_id":            scope.Workflow.ID,
		"workflow_name":          scope.Workflow.Name,
		"source_stage_id":        scope.SourceStage.ID,
		"source_stage_name":      scope.SourceStage.Name,
		"source_department_id":   scope.SourceDepartment.ID,
		"source_department_name": scope.SourceDepartment.Name,
		"target_workflow_id":     scope.TargetWorkflow.ID,
		"target_workflow_name":   scope.TargetWorkflow.Name,
		"target_stage_id":        scope.TargetStage.ID,
		"target_stage_name":      scope.TargetStage.Name,
		"target_department_id":   scope.TargetDepartment.ID,
		"target_department_name": scope.TargetDepartment.Name,
	}
}

func pendingLeadDispatchCount(ctx context.Context, workflowID uint64) int {
	return int(crmmodel.NewLeadDispatchHandoffModel().Count(ctx, map[string]any{
		"source_workflow_id": workflowID,
		"status":             crmmodel.LeadDispatchHandoffPending,
	}))
}

func pendingLeadDispatchRows(ctx context.Context, workflowID uint64) []map[string]any {
	handoffs := crmmodel.NewLeadDispatchHandoffModel().Select(ctx, map[string]any{
		"source_workflow_id": workflowID,
		"status":             crmmodel.LeadDispatchHandoffPending,
	}, map[string]any{"order": "created_at asc,id asc"})
	rows := make([]map[string]any, 0, len(handoffs))
	for _, handoff := range handoffs {
		if handoff == nil {
			continue
		}
		lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{
			"id":     handoff.LeadID,
			"status": crmmodel.LeadStatusPending,
		})
		instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
			"id":          handoff.WorkflowInstanceID,
			"workflow_id": handoff.SourceWorkflowID,
			"stage_id":    handoff.SourceStageID,
			"status":      crmmodel.ProgressStatusActive,
		})
		if lead == nil || instance == nil {
			continue
		}
		rows = append(rows, map[string]any{
			"kind":                 "lead_handoff",
			"id":                   handoff.ID,
			"handoff_id":           handoff.ID,
			"lead_id":              lead.ID,
			"lead_name":            lead.Name,
			"lead_code":            lead.Code,
			"phone":                lead.Phone,
			"source_stage_id":      handoff.SourceStageID,
			"target_stage_id":      handoff.TargetStageID,
			"target_department_id": handoff.TargetDepartmentID,
			"created_at":           handoff.CreatedAt,
		})
	}
	return rows
}

func configuredDispatchPoolStaffRows(ctx context.Context, departmentID uint64) []map[string]any {
	setting := crmmodel.NewDepartmentDispatchSettingModel().Find(ctx, map[string]any{
		"department_id": departmentID,
		"status":        crmmodel.StatusEnabled,
	})
	if setting == nil || enabledDispatchPool(ctx, departmentID, setting.ActivePoolID) == nil {
		return []map[string]any{}
	}
	members := crmmodel.NewDispatchPoolMemberModel().Select(ctx, map[string]any{
		"pool_id":       setting.ActivePoolID,
		"department_id": departmentID,
		"status":        crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
	rows := make([]map[string]any, 0, len(members))
	for _, member := range members {
		if member == nil {
			continue
		}
		person := enabledStaffInDepartment(ctx, member.StaffID, departmentID)
		if person == nil {
			continue
		}
		rows = append(rows, map[string]any{
			"id":   person.ID,
			"name": person.Name,
		})
	}
	return rows
}

func dispatchGroupName(payload map[string]any) (string, error) {
	name := strings.TrimSpace(firstText(payload, "name", "group_name", "groupName"))
	if name == "" {
		return "", fmt.Errorf("请填写工作组名称")
	}
	if len([]rune(name)) > 64 {
		return "", fmt.Errorf("工作组名称不能超过64个字符")
	}
	return name, nil
}
