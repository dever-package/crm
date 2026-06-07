package service

import (
	"context"
	"fmt"
	"time"

	crmmodel "my/package/crm/model"
)

type ResourceService struct{}

func NewResourceService() ResourceService {
	return ResourceService{}
}

func (ResourceService) CreateWithTask(ctx context.Context, payload map[string]any) (map[string]any, error) {
	customerID := firstUint64(payload, "customer_id", "customerId")
	if customerID == 0 {
		return nil, fmt.Errorf("客户不能为空")
	}
	assetName := firstText(payload, "asset_name", "assetName")
	if assetName == "" {
		return nil, fmt.Errorf("资产名称不能为空")
	}

	resourceNo := firstText(payload, "resource_no", "resourceNo")
	if resourceNo == "" {
		resourceNo = defaultResourceNo()
	}
	assetCateID := firstUint64(payload, "asset_cate_id", "assetCateId")
	if assetCateID == 0 {
		assetCateID = crmmodel.DefaultAssetCateID
	}
	assetStatus := firstText(payload, "asset_status", "assetStatus")
	if assetStatus == "" {
		assetStatus = crmmodel.AssetStatusDefault
	}
	record := map[string]any{
		"resource_no":   resourceNo,
		"asset_name":    assetName,
		"customer_id":   customerID,
		"asset_cate_id": assetCateID,
		"asset_status":  assetStatus,
		"remark":        firstText(payload, "remark"),
		"created_at":    time.Now(),
		"updated_at":    time.Now(),
	}

	resourceID := uint64(crmmodel.NewCustomerResourceModel().Insert(ctx, record))
	if resourceID == 0 {
		return nil, fmt.Errorf("客户资产创建失败")
	}

	flowTemplateID := ResolveFlowTemplateID(ctx, firstUint64(payload, "flow_template_id", "flowTemplateId"))
	flowReleaseID := uint64(0)
	if flowTemplateID > 0 {
		flow := crmmodel.NewFlowTemplateModel().Find(ctx, map[string]any{"id": flowTemplateID})
		if flow != nil {
			flowReleaseID = flow.CurrentReleaseID
		}
	}
	taskID := createFirstTask(ctx, resourceID, flowTemplateID, flowReleaseID)

	return map[string]any{
		"id":      resourceID,
		"task_id": taskID,
	}, nil
}

func (ResourceService) Detail(ctx context.Context, resourceID uint64) (map[string]any, error) {
	return map[string]any{
		"id":            resourceID,
		"resource":      crmmodel.NewCustomerResourceModel().FindMap(ctx, map[string]any{"id": resourceID}),
		"tasks":         crmmodel.NewResourceTaskModel().SelectMap(ctx, map[string]any{"resource_id": resourceID}),
		"records":       crmmodel.NewTaskRecordModel().SelectMap(ctx, map[string]any{"resource_id": resourceID}),
		"data_sections": crmmodel.NewDataRecordModel().SelectMap(ctx, map[string]any{"resource_id": resourceID}),
		"timeline":      crmmodel.NewTimelineModel().SelectMap(ctx, map[string]any{"resource_id": resourceID}),
	}, nil
}

func createFirstTask(ctx context.Context, resourceID uint64, flowTemplateID uint64, flowReleaseID uint64) uint64 {
	if node, template, ok := firstTaskFromReleaseSnapshot(ctx, flowReleaseID); ok {
		return createTaskFromNode(ctx, resourceID, flowTemplateID, flowReleaseID, node, template)
	}
	nodes := crmmodel.NewFlowNodeModel().Select(ctx, map[string]any{
		"flow_template_id": flowTemplateID,
		"status":           crmmodel.StatusEnabled,
	})
	if len(nodes) == 0 {
		return 0
	}
	return createTaskFromNode(ctx, resourceID, flowTemplateID, flowReleaseID, nodes[0], nil)
}

func createTaskFromNode(ctx context.Context, resourceID uint64, flowTemplateID uint64, flowReleaseID uint64, node *crmmodel.FlowNode, template *crmmodel.TaskTemplate) uint64 {
	return createTaskFromNodeWithAssignment(ctx, resourceID, flowTemplateID, flowReleaseID, node, template, 0, 0)
}

func createTaskFromNodeWithAssignment(ctx context.Context, resourceID uint64, flowTemplateID uint64, flowReleaseID uint64, node *crmmodel.FlowNode, template *crmmodel.TaskTemplate, departmentID uint64, roleID uint64) uint64 {
	if node == nil {
		return 0
	}
	if template == nil && node.TaskTemplateID > 0 {
		template = crmmodel.NewTaskTemplateModel().Find(ctx, map[string]any{"id": node.TaskTemplateID})
	}
	executorMode := firstNonEmpty(node.ExecutorMode, "department")
	departmentID = firstNonZero(departmentID, node.DefaultDepartmentID)
	roleID = firstNonZero(roleID, node.DefaultRoleID)
	staffID := node.DefaultStaffID
	enableDeadline := node.EnableDeadline
	deadlineMinutes := node.DeadlineMinutes
	taskKey := firstNonEmpty(node.NodeKey, fmt.Sprintf("task_%d", node.ID))
	taskID := uint64(crmmodel.NewResourceTaskModel().Insert(ctx, map[string]any{
		"resource_id":            resourceID,
		"flow_template_id":       flowTemplateID,
		"flow_release_id":        flowReleaseID,
		"stage_id":               node.StageID,
		"flow_node_id":           node.ID,
		"flow_node_key":          node.NodeKey,
		"node_type":              firstNonEmpty(node.NodeType, crmmodel.NodeTypeInput),
		"task_template_id":       node.TaskTemplateID,
		"task_key":               taskKey,
		"task_name":              firstNonEmpty(node.Name, templateName(template), "未命名任务"),
		"status":                 crmmodel.TaskStatusPending,
		"executor_mode":          executorMode,
		"assignee_department_id": departmentID,
		"assignee_role_id":       roleID,
		"assignee_staff_id":      staffID,
		"enable_deadline":        enableDeadline,
		"deadline_at":            deadlineAt(time.Now(), enableDeadline, deadlineMinutes),
		"input_snapshot_json":    taskInstanceSnapshotJSON(ctx, flowReleaseID, node, template),
		"created_at":             time.Now(),
		"updated_at":             time.Now(),
	}))
	if taskID > 0 {
		crmmodel.NewTimelineModel().Insert(ctx, map[string]any{
			"resource_id": resourceID,
			"task_id":     taskID,
			"event_type":  "task_created",
			"title":       "生成任务",
			"content":     firstNonEmpty(node.Name, templateName(template), "未命名任务"),
			"created_at":  time.Now(),
		})
	}
	return taskID
}

func firstNonZero(values ...uint64) uint64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func deadlineAt(now time.Time, enabled bool, minutes int) time.Time {
	if !enabled || minutes <= 0 {
		return time.Time{}
	}
	return now.Add(time.Duration(minutes) * time.Minute)
}

func templateName(template *crmmodel.TaskTemplate) string {
	if template == nil {
		return ""
	}
	return template.Name
}
