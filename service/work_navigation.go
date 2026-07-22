package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	crmmodel "github.com/dever-package/crm/model"
)

const workGlobalSearchLimit = 12

func (WorkService) Navigation(ctx context.Context, staff *WorkStaffSession) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	workflows := crmmodel.NewWorkflowModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
	pendingCounts := currentWorkPersonalWorkload(ctx, staff).pendingTaskCountByWorkflow()
	rows := make([]map[string]any, 0, len(workflows))
	for _, workflow := range workflows {
		if workflow == nil || !canAccessWorkflow(ctx, staff, workflow) {
			continue
		}
		rows = append(rows, map[string]any{
			"id":            workflow.ID,
			"name":          workflow.Name,
			"subject_type":  workflow.SubjectType,
			"path":          workflowNavigationPath(workflow),
			"pending_count": pendingCounts[workflow.ID],
		})
	}
	return map[string]any{
		"list":     rows,
		"dispatch": workDispatchNavigation(ctx, staff),
	}, nil
}

func workflowNavigationPath(workflow *crmmodel.Workflow) string {
	if workflow == nil {
		return "/crm/stats"
	}
	basePath := "/crm/work"
	if workflow.SubjectType == crmmodel.WorkflowSubjectLead {
		basePath = "/crm/lead"
	}
	return fmt.Sprintf("%s?workflow_id=%d", basePath, workflow.ID)
}

func (WorkService) GlobalSearch(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	keyword := firstText(payload, "keyword")
	if keyword == "" {
		return map[string]any{"list": []map[string]any{}}, nil
	}
	rows := make([]map[string]any, 0, workGlobalSearchLimit)
	rows = appendWorkLeadSearchRows(ctx, staff, rows, keyword)
	rows = appendWorkCustomerSearchRows(ctx, staff, rows, keyword)
	if len(rows) > workGlobalSearchLimit {
		rows = rows[:workGlobalSearchLimit]
	}
	return map[string]any{"list": rows}, nil
}

func appendWorkLeadSearchRows(ctx context.Context, staff *WorkStaffSession, rows []map[string]any, keyword string) []map[string]any {
	for _, lead := range crmmodel.NewLeadModel().Select(ctx, map[string]any{}, map[string]any{"order": "id desc"}) {
		if len(rows) >= workGlobalSearchLimit || lead == nil || !matchesWorkLeadKeyword(lead, keyword) {
			continue
		}
		instance, workflow := searchableLeadWorkflowInstance(ctx, staff, lead.ID)
		if instance == nil || workflow == nil {
			continue
		}
		rows = append(rows, map[string]any{
			"type":         "lead",
			"type_name":    "线索",
			"id":           lead.ID,
			"title":        lead.Name,
			"subtitle":     joinWorkSearchText(lead.Phone, lead.Code, workflow.Name),
			"lead_id":      lead.ID,
			"workflow_id":  workflow.ID,
			"subject_type": workflow.SubjectType,
			"path":         workSearchNavigationPath("/crm/lead", workflow.ID, keyword),
		})
	}
	return rows
}

func searchableLeadWorkflowInstance(ctx context.Context, staff *WorkStaffSession, leadID uint64) (*crmmodel.WorkflowInstance, *crmmodel.Workflow) {
	instances := crmmodel.NewWorkflowInstanceModel().Select(ctx, map[string]any{"lead_id": leadID}, map[string]any{"order": "id desc"})
	for _, instance := range instances {
		if instance == nil || !canViewWorkflowInstance(ctx, staff, instance) {
			continue
		}
		workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
			"id":           instance.WorkflowID,
			"subject_type": crmmodel.WorkflowSubjectLead,
			"status":       crmmodel.StatusEnabled,
		})
		if workflow != nil {
			return instance, workflow
		}
	}
	return nil, nil
}

func appendWorkCustomerSearchRows(ctx context.Context, staff *WorkStaffSession, rows []map[string]any, keyword string) []map[string]any {
	for _, customer := range crmmodel.NewCustomerModel().Select(ctx, map[string]any{}, map[string]any{"order": "id desc"}) {
		if len(rows) >= workGlobalSearchLimit || customer == nil {
			continue
		}
		customerMatches := containsFold(customer.Code, keyword) || containsFold(customer.Name, keyword) ||
			containsFold(customer.Phone, keyword) || containsFold(customer.Wechat, keyword)
		assets := crmmodel.NewCustomerAssetModel().Select(ctx, map[string]any{"customer_id": customer.ID}, map[string]any{"order": "id desc"})
		if customerMatches {
			assetID, workflowID := searchableCustomerWorkflowTarget(ctx, staff, customer.ID, assets)
			if workflowID > 0 {
				rows = append(rows, workCustomerSearchRow(ctx, customer, assetID, workflowID, keyword))
				if len(rows) >= workGlobalSearchLimit {
					break
				}
			}
		}
		for _, asset := range assets {
			if len(rows) >= workGlobalSearchLimit || asset == nil ||
				(!containsFold(asset.AssetNo, keyword) && !containsFold(asset.AssetName, keyword)) {
				continue
			}
			workflowID := searchableAssetWorkflowID(ctx, staff, customer.ID, asset.ID)
			if workflowID == 0 {
				continue
			}
			rows = append(rows, workAssetSearchRow(ctx, customer, asset, workflowID, keyword))
		}
	}
	return rows
}

func searchableCustomerWorkflowTarget(ctx context.Context, staff *WorkStaffSession, customerID uint64, assets []*crmmodel.CustomerAsset) (uint64, uint64) {
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		if workflowID := searchableAssetWorkflowID(ctx, staff, customerID, asset.ID); workflowID > 0 {
			return asset.ID, workflowID
		}
	}
	return 0, 0
}

func searchableAssetWorkflowID(ctx context.Context, staff *WorkStaffSession, customerID, assetID uint64) uint64 {
	instances := crmmodel.NewWorkflowInstanceModel().Select(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	}, map[string]any{"order": "id desc"})
	for _, activeOnly := range []bool{true, false} {
		for _, instance := range instances {
			if instance == nil || activeOnly && instance.Status != crmmodel.ProgressStatusActive ||
				!canViewWorkflowInstanceInScope(ctx, staff, instance) {
				continue
			}
			workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
				"id":           instance.WorkflowID,
				"subject_type": crmmodel.WorkflowSubjectCustomerAsset,
				"status":       crmmodel.StatusEnabled,
			})
			if workflow != nil && canAccessWorkflow(ctx, staff, workflow) {
				return workflow.ID
			}
		}
	}
	return 0
}

func workCustomerSearchRow(ctx context.Context, customer *crmmodel.Customer, assetID, workflowID uint64, keyword string) map[string]any {
	return map[string]any{
		"type":         "customer",
		"type_name":    "客户",
		"id":           customer.ID,
		"title":        customer.Name,
		"subtitle":     joinWorkSearchText(customer.Phone, customerCodeDisplayForWork(ctx, customer.Code)),
		"customer_id":  customer.ID,
		"asset_id":     assetID,
		"workflow_id":  workflowID,
		"subject_type": crmmodel.WorkflowSubjectCustomerAsset,
		"path":         workSearchNavigationPath("/crm/work", workflowID, keyword),
	}
}

func workAssetSearchRow(ctx context.Context, customer *crmmodel.Customer, asset *crmmodel.CustomerAsset, workflowID uint64, keyword string) map[string]any {
	return map[string]any{
		"type":         "asset",
		"type_name":    "客户资产",
		"id":           asset.ID,
		"title":        asset.AssetName,
		"subtitle":     joinWorkSearchText(customer.Name, asset.AssetNo, customerCodeDisplayForWork(ctx, customer.Code)),
		"customer_id":  customer.ID,
		"asset_id":     asset.ID,
		"workflow_id":  workflowID,
		"subject_type": crmmodel.WorkflowSubjectCustomerAsset,
		"path":         workSearchNavigationPath("/crm/work", workflowID, keyword),
	}
}

func workSearchNavigationPath(basePath string, workflowID uint64, keyword string) string {
	query := url.Values{}
	if workflowID > 0 {
		query.Set("workflow_id", fmt.Sprintf("%d", workflowID))
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		query.Set("keyword", keyword)
	}
	if basePath == "/crm/work" {
		query.Set("mode", workCustomerModeAll)
		query.Set("scope", "mine")
	}
	if queryString := query.Encode(); queryString != "" {
		return basePath + "?" + queryString
	}
	return basePath
}

func joinWorkSearchText(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " · ")
}
