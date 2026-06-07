package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "my/package/crm/model"
)

type FlowDesignerService struct{}

func NewFlowDesignerService() FlowDesignerService {
	return FlowDesignerService{}
}

func ResolveFlowTemplateID(ctx context.Context, flowTemplateID uint64) uint64 {
	if flowTemplateID > 0 {
		return flowTemplateID
	}
	flow := crmmodel.NewFlowTemplateModel().Find(ctx, map[string]any{
		"resource_type": crmmodel.ResourceTypeAssetDisposal,
		"status":        crmmodel.StatusEnabled,
	})
	if flow != nil {
		return flow.ID
	}
	flow = crmmodel.NewFlowTemplateModel().Find(ctx, map[string]any{"id": crmmodel.DefaultFlowTemplateID})
	if flow != nil {
		return flow.ID
	}
	return 0
}

func EnsureFlowTemplateID(ctx context.Context, flowTemplateID uint64) uint64 {
	if resolved := ResolveFlowTemplateID(ctx, flowTemplateID); resolved > 0 {
		return resolved
	}
	now := time.Now()
	return uint64(crmmodel.NewFlowTemplateModel().Insert(ctx, map[string]any{
		"name":           "业务流程",
		"description":    "",
		"resource_type":  crmmodel.ResourceTypeAssetDisposal,
		"publish_status": crmmodel.PublishStatusDraft,
		"config_json":    "{}",
		"status":         crmmodel.StatusEnabled,
		"sort":           100,
		"created_at":     now,
		"updated_at":     now,
	}))
}

func (FlowDesignerService) Workspace(ctx context.Context, flowTemplateID uint64) (map[string]any, error) {
	flowTemplateID = EnsureFlowTemplateID(ctx, flowTemplateID)
	if flowTemplateID == 0 {
		return nil, fmt.Errorf("默认流程模板不存在")
	}
	flow := crmmodel.NewFlowTemplateModel().Find(ctx, map[string]any{"id": flowTemplateID})
	if flow == nil {
		return nil, fmt.Errorf("流程模板不存在")
	}

	return map[string]any{
		"flow_template_id": flowTemplateID,
		"flow":             crmmodel.NewFlowTemplateModel().FindMap(ctx, map[string]any{"id": flowTemplateID}),
		"stages":           crmmodel.NewFlowStageModel().SelectMap(ctx, map[string]any{"flow_template_id": flowTemplateID, "status": crmmodel.StatusEnabled}),
		"nodes":            crmmodel.NewFlowNodeModel().SelectMap(ctx, map[string]any{"flow_template_id": flowTemplateID, "status": crmmodel.StatusEnabled}),
		"edges":            crmmodel.NewFlowEdgeModel().SelectMap(ctx, map[string]any{"flow_template_id": flowTemplateID, "status": crmmodel.StatusEnabled}),
		"task_templates":   crmmodel.NewTaskTemplateModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}),
		"rule_scripts":     crmmodel.NewRuleScriptModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}),
		"departments":      crmmodel.NewDepartmentModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}),
		"staff":            crmmodel.NewStaffModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}),
	}, nil
}

func (service FlowDesignerService) Save(ctx context.Context, payload map[string]any) (map[string]any, error) {
	flowTemplateID, _, err := saveFlowTemplate(ctx, payload)
	if err != nil {
		return nil, err
	}
	if flowTemplateID == 0 {
		return nil, fmt.Errorf("缺少流程模板ID")
	}

	stats := map[string]any{}
	stageRows := payloadRows(payload["stages"])
	stats["stages"] = saveRows(ctx, crmmodel.NewFlowStageModel(), stageRows, map[string]any{"flow_template_id": flowTemplateID}, flowStageFields, []string{"stage_key", "name"})
	stageIDs := flowStageIDMap(ctx, flowTemplateID, stageRows)

	nodeRows := payloadRows(payload["nodes"])
	for _, row := range nodeRows {
		if inputUint64(row["stage_id"]) > 0 {
			continue
		}
		stageKey := inputText(row["_stage_key"])
		if stageKey == "" {
			stageKey = inputText(row["stage_key"])
		}
		if stageID := stageIDs[stageKey]; stageID > 0 {
			row["stage_id"] = stageID
		}
	}
	stats["nodes"] = saveRows(ctx, crmmodel.NewFlowNodeModel(), nodeRows, map[string]any{"flow_template_id": flowTemplateID}, flowNodeFields, []string{"node_key", "name"})
	nodeIDs := flowNodeIDMap(ctx, flowTemplateID)

	edgeRows := payloadRows(payload["edges"])
	for _, row := range edgeRows {
		if fromKey := inputText(row["from_node_key"]); fromKey != "" {
			row["from_node_id"] = nodeIDs[fromKey]
		}
		if toKey := inputText(row["to_node_key"]); toKey != "" {
			row["to_node_id"] = nodeIDs[toKey]
		}
	}
	stats["edges"] = saveRows(ctx, crmmodel.NewFlowEdgeModel(), edgeRows, map[string]any{"flow_template_id": flowTemplateID}, flowEdgeFields, []string{"from_node_key", "to_node_key"})
	if len(payloadRows(payload["stages"])) > 0 || len(nodeRows) > 0 || len(edgeRows) > 0 {
		markFlowTemplateEditing(ctx, flowTemplateID)
	}

	workspace, err := service.Workspace(ctx, flowTemplateID)
	if err != nil {
		return nil, err
	}
	workspace["saved"] = true
	workspace["stats"] = stats
	return workspace, nil
}

func (service FlowDesignerService) Clone(ctx context.Context, flowTemplateID uint64) (map[string]any, error) {
	flow := crmmodel.NewFlowTemplateModel().Find(ctx, map[string]any{"id": flowTemplateID})
	if flow == nil {
		return nil, fmt.Errorf("流程模板不存在")
	}

	now := time.Now()
	newFlowID := uint64(crmmodel.NewFlowTemplateModel().Insert(ctx, map[string]any{
		"name":           fmt.Sprintf("%s 复制", flow.Name),
		"description":    flow.Description,
		"resource_type":  flow.ResourceType,
		"publish_status": crmmodel.PublishStatusDraft,
		"config_json":    flow.ConfigJSON,
		"status":         flow.Status,
		"sort":           flow.Sort + 1,
		"created_at":     now,
		"updated_at":     now,
	}))
	if newFlowID == 0 {
		return nil, fmt.Errorf("复制流程模板失败")
	}

	stageIDs := cloneStages(ctx, flowTemplateID, newFlowID, now)
	nodeIDs := cloneNodes(ctx, flowTemplateID, newFlowID, stageIDs, now)
	cloneEdges(ctx, flowTemplateID, newFlowID, nodeIDs, now)

	workspace, err := service.Workspace(ctx, newFlowID)
	if err != nil {
		return nil, err
	}
	workspace["cloned"] = true
	return map[string]any{
		"cloned":              true,
		"source_template_id":  flowTemplateID,
		"flow_template_id":    newFlowID,
		"stage_count":         len(stageIDs),
		"node_count":          len(nodeIDs),
		"data_template_reuse": true,
		"workspace":           workspace,
	}, nil
}

var flowStageFields = []string{"stage_key", "name", "description", "default_department_id", "position_json", "sort", "status"}

var taskTemplateFields = []string{
	"cate_id", "name", "description", "sort", "status",
}

var flowNodeFields = []string{
	"stage_id", "node_key", "name", "description", "node_type", "task_template_id", "script_id", "executor_mode",
	"default_department_id", "default_role_id", "default_staff_id", "enable_deadline", "deadline_minutes",
	"position_json", "config_json", "sort", "status",
}

var taskFieldFields = []string{
	"task_template_id", "data_template_cate_id", "data_template_id", "field_source", "main_field", "data_field_id",
	"name", "required", "sort", "status",
}

var taskResultFields = []string{"task_template_id", "name", "result_value", "is_success", "requires_comment", "sort", "status"}

var flowEdgeFields = []string{
	"from_node_id", "from_node_key", "to_node_id", "to_node_key", "match_result", "match_script_id",
	"target_resource_status", "target_department_id", "target_role_id", "condition_json", "sort", "status",
}

var dataTemplateFields = []string{"cate_id", "name", "status", "sort"}

var dataFieldFields = []string{"data_template_id", "name", "field_type", "default_value", "sort", "status"}

func saveFlowTemplate(ctx context.Context, payload map[string]any) (uint64, string, error) {
	flowData := mapFromAny(payload["flow"])
	flowTemplateID := firstUint64(payload, "flow_template_id", "id")
	if flowTemplateID == 0 {
		flowTemplateID = inputUint64(flowData["id"])
	}
	if flowTemplateID == 0 && firstText(flowData, "name") == "" && firstText(payload, "name") == "" {
		flowTemplateID = EnsureFlowTemplateID(ctx, flowTemplateID)
	}

	now := time.Now()
	resourceType := firstText(flowData, "resource_type")
	if resourceType == "" {
		resourceType = firstText(payload, "resource_type")
	}

	record := cleanRecord(flowData, []string{"name", "description", "resource_type", "config_json", "status", "sort"})

	if flowTemplateID == 0 {
		if resourceType == "" {
			resourceType = "asset_disposal"
		}
		record["resource_type"] = resourceType
		name := firstText(flowData, "name")
		if name == "" {
			name = firstText(payload, "name")
		}
		if name == "" {
			return 0, resourceType, fmt.Errorf("新建流程模板必须填写名称")
		}
		record["name"] = name
		record["publish_status"] = crmmodel.PublishStatusDraft
		record["created_at"] = now
		record["updated_at"] = now
		flowTemplateID = uint64(crmmodel.NewFlowTemplateModel().Insert(ctx, record))
		return flowTemplateID, resourceType, nil
	}

	flow := crmmodel.NewFlowTemplateModel().Find(ctx, map[string]any{"id": flowTemplateID})
	if flow == nil {
		return 0, resourceType, fmt.Errorf("流程模板不存在")
	}
	if resourceType == "" {
		resourceType = flow.ResourceType
	}
	if _, ok := record["resource_type"]; ok {
		resourceType = inputText(record["resource_type"])
	}
	if len(record) > 0 {
		if flow.PublishStatus == crmmodel.PublishStatusPublished {
			record["publish_status"] = crmmodel.PublishStatusEditing
		}
		record["updated_at"] = now
		crmmodel.NewFlowTemplateModel().Update(ctx, map[string]any{"id": flowTemplateID}, record)
	}
	return flowTemplateID, resourceType, nil
}

func saveRows[T any](ctx context.Context, model *orm.Model[T], rows []map[string]any, base map[string]any, allowed []string, required []string) []uint64 {
	now := time.Now()
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		id := inputUint64(row["id"])
		record := cleanRecord(row, allowed)
		for key, value := range base {
			record[key] = value
		}
		if id == 0 && !hasRequired(row, required) {
			continue
		}
		if id > 0 {
			if len(record) == 0 {
				continue
			}
			record["updated_at"] = now
			model.Update(ctx, map[string]any{"id": id}, record)
			ids = append(ids, id)
			continue
		}
		record["created_at"] = now
		record["updated_at"] = now
		if _, ok := record["status"]; !ok {
			record["status"] = crmmodel.StatusEnabled
		}
		ids = append(ids, uint64(model.Insert(ctx, record)))
	}
	return ids
}

func flowStageIDMap(ctx context.Context, flowTemplateID uint64, payloadRows []map[string]any) map[string]uint64 {
	result := map[string]uint64{}
	rows := crmmodel.NewFlowStageModel().Select(ctx, map[string]any{"flow_template_id": flowTemplateID, "status": crmmodel.StatusEnabled})
	for _, row := range rows {
		if row.StageKey != "" {
			result[row.StageKey] = row.ID
		}
		result[fmt.Sprintf("%d", row.ID)] = row.ID
		for _, payloadRow := range payloadRows {
			if inputText(payloadRow["stage_key"]) != row.StageKey {
				continue
			}
			if key := inputText(payloadRow["_key"]); key != "" {
				result[key] = row.ID
			}
			if key := inputText(payloadRow["id"]); key != "" {
				result[key] = row.ID
			}
		}
	}
	return result
}

func flowNodeIDMap(ctx context.Context, flowTemplateID uint64) map[string]uint64 {
	result := map[string]uint64{}
	rows := crmmodel.NewFlowNodeModel().Select(ctx, map[string]any{"flow_template_id": flowTemplateID, "status": crmmodel.StatusEnabled})
	for _, row := range rows {
		if row.NodeKey != "" {
			result[row.NodeKey] = row.ID
		}
	}
	return result
}

func markFlowTemplateEditing(ctx context.Context, flowTemplateID uint64) {
	flow := crmmodel.NewFlowTemplateModel().Find(ctx, map[string]any{"id": flowTemplateID})
	if flow == nil || flow.PublishStatus != crmmodel.PublishStatusPublished {
		return
	}
	crmmodel.NewFlowTemplateModel().Update(ctx, map[string]any{"id": flowTemplateID}, map[string]any{
		"publish_status": crmmodel.PublishStatusEditing,
		"updated_at":     time.Now(),
	})
}

func cleanRecord(row map[string]any, allowed []string) map[string]any {
	record := map[string]any{}
	for _, key := range allowed {
		value, ok := row[key]
		if !ok {
			continue
		}
		if normalized, ok := normalizeSaveValue(key, value); ok {
			record[key] = normalized
		}
	}
	return record
}

func normalizeSaveValue(key string, value any) (any, bool) {
	switch key {
	case "status":
		status := int16(inputInt(value))
		if status == 0 {
			status = crmmodel.StatusEnabled
		}
		return status, true
	case "sort", "deadline_minutes":
		return inputInt(value), true
	case "flow_template_id", "stage_id", "flow_node_id", "task_template_id", "script_id", "default_department_id", "default_role_id", "default_staff_id",
		"from_task_template_id", "match_script_id", "to_stage_id", "to_task_template_id", "from_node_id", "to_node_id",
		"target_department_id", "target_role_id", "data_template_cate_id", "data_template_id", "data_field_id", "cate_id":
		return inputUint64(value), true
	case "required", "enable_deadline", "is_success", "requires_comment":
		return boolValue(value), true
	case "position_json", "config_json", "options_json", "condition_json":
		return jsonText(value), true
	default:
		return inputText(value), true
	}
}

func boolValue(value any) bool {
	switch row := value.(type) {
	case bool:
		return row
	case string:
		return row == "1" || row == "true" || row == "yes" || row == "on"
	default:
		return inputInt(value) > 0
	}
}

func hasRequired(row map[string]any, required []string) bool {
	for _, key := range required {
		value := row[key]
		if inputText(value) != "" || inputUint64(value) > 0 {
			continue
		}
		return false
	}
	return true
}

func payloadRows(value any) []map[string]any {
	switch rows := value.(type) {
	case []map[string]any:
		return rows
	case []any:
		result := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			if mapped := mapFromAny(row); len(mapped) > 0 {
				result = append(result, mapped)
			}
		}
		return result
	case string:
		var decoded []map[string]any
		if err := json.Unmarshal([]byte(rows), &decoded); err == nil {
			return decoded
		}
		var loose []any
		if err := json.Unmarshal([]byte(rows), &loose); err == nil {
			return payloadRows(loose)
		}
	}
	return nil
}

func taskFieldMaps(ctx context.Context, tasks []*crmmodel.TaskTemplate) []map[string]any {
	rows := []map[string]any{}
	for _, task := range tasks {
		rows = append(rows, crmmodel.NewTaskFieldModel().SelectMap(ctx, map[string]any{"task_template_id": task.ID, "status": crmmodel.StatusEnabled})...)
	}
	return rows
}

func taskResultMaps(ctx context.Context, tasks []*crmmodel.TaskTemplate) []map[string]any {
	rows := []map[string]any{}
	for _, task := range tasks {
		rows = append(rows, crmmodel.NewTaskResultModel().SelectMap(ctx, map[string]any{"task_template_id": task.ID, "status": crmmodel.StatusEnabled})...)
	}
	return rows
}

func dataFieldMaps(ctx context.Context, templates []*crmmodel.DataTemplate) []map[string]any {
	rows := []map[string]any{}
	for _, template := range templates {
		rows = append(rows, crmmodel.NewDataFieldModel().SelectMap(ctx, map[string]any{"data_template_id": template.ID, "status": crmmodel.StatusEnabled})...)
	}
	return rows
}

func cloneStages(ctx context.Context, sourceFlowID uint64, targetFlowID uint64, now time.Time) map[uint64]uint64 {
	ids := map[uint64]uint64{}
	stages := crmmodel.NewFlowStageModel().Select(ctx, map[string]any{"flow_template_id": sourceFlowID, "status": crmmodel.StatusEnabled})
	for _, stage := range stages {
		id := uint64(crmmodel.NewFlowStageModel().Insert(ctx, map[string]any{
			"flow_template_id":      targetFlowID,
			"stage_key":             stage.StageKey,
			"name":                  stage.Name,
			"description":           stage.Description,
			"default_department_id": stage.DefaultDepartmentID,
			"position_json":         stage.PositionJSON,
			"sort":                  stage.Sort,
			"status":                stage.Status,
			"created_at":            now,
			"updated_at":            now,
		}))
		ids[stage.ID] = id
	}
	return ids
}

func cloneNodes(ctx context.Context, sourceFlowID uint64, targetFlowID uint64, stageIDs map[uint64]uint64, now time.Time) map[uint64]uint64 {
	ids := map[uint64]uint64{}
	nodes := crmmodel.NewFlowNodeModel().Select(ctx, map[string]any{"flow_template_id": sourceFlowID, "status": crmmodel.StatusEnabled})
	for _, node := range nodes {
		id := uint64(crmmodel.NewFlowNodeModel().Insert(ctx, map[string]any{
			"flow_template_id":      targetFlowID,
			"stage_id":              mappedID(stageIDs, node.StageID),
			"node_key":              node.NodeKey,
			"name":                  node.Name,
			"description":           node.Description,
			"node_type":             node.NodeType,
			"task_template_id":      node.TaskTemplateID,
			"script_id":             node.ScriptID,
			"executor_mode":         node.ExecutorMode,
			"default_department_id": node.DefaultDepartmentID,
			"default_role_id":       node.DefaultRoleID,
			"default_staff_id":      node.DefaultStaffID,
			"enable_deadline":       node.EnableDeadline,
			"deadline_minutes":      node.DeadlineMinutes,
			"position_json":         node.PositionJSON,
			"config_json":           node.ConfigJSON,
			"sort":                  node.Sort,
			"status":                node.Status,
			"created_at":            now,
			"updated_at":            now,
		}))
		ids[node.ID] = id
	}
	return ids
}

func cloneEdges(ctx context.Context, sourceFlowID uint64, targetFlowID uint64, nodeIDs map[uint64]uint64, now time.Time) {
	edges := crmmodel.NewFlowEdgeModel().Select(ctx, map[string]any{"flow_template_id": sourceFlowID, "status": crmmodel.StatusEnabled})
	for _, edge := range edges {
		crmmodel.NewFlowEdgeModel().Insert(ctx, map[string]any{
			"flow_template_id":       targetFlowID,
			"from_node_id":           mappedID(nodeIDs, edge.FromNodeID),
			"from_node_key":          edge.FromNodeKey,
			"to_node_id":             mappedID(nodeIDs, edge.ToNodeID),
			"to_node_key":            edge.ToNodeKey,
			"match_result":           edge.MatchResult,
			"match_script_id":        edge.MatchScriptID,
			"target_resource_status": edge.TargetResourceStatus,
			"target_department_id":   edge.TargetDepartmentID,
			"target_role_id":         edge.TargetRoleID,
			"condition_json":         edge.ConditionJSON,
			"sort":                   edge.Sort,
			"status":                 edge.Status,
			"created_at":             now,
			"updated_at":             now,
		})
	}
}

func mappedID(ids map[uint64]uint64, oldID uint64) uint64 {
	if oldID == 0 {
		return 0
	}
	return ids[oldID]
}
