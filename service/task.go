package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	crmmodel "my/package/crm/model"
)

type TaskService struct{}

func NewTaskService() TaskService {
	return TaskService{}
}

func (TaskService) DetailFromDB(ctx context.Context, taskID uint64) (map[string]any, error) {
	task := crmmodel.NewResourceTaskModel().Find(ctx, map[string]any{"id": taskID})
	if task == nil {
		return nil, fmt.Errorf("任务不存在")
	}
	fields, results := taskDetailConfig(ctx, task)
	return map[string]any{
		"id":      taskID,
		"task":    crmmodel.NewResourceTaskModel().FindMap(ctx, map[string]any{"id": taskID}),
		"fields":  fields,
		"results": results,
	}, nil
}

func (TaskService) SubmitToDB(ctx context.Context, payload map[string]any) (map[string]any, error) {
	taskID := firstUint64(payload, "task_id", "taskId", "id")
	if taskID == 0 {
		return nil, fmt.Errorf("任务不能为空")
	}
	task := crmmodel.NewResourceTaskModel().Find(ctx, map[string]any{"id": taskID})
	if task == nil {
		return nil, fmt.Errorf("任务不存在")
	}
	resultValue := firstText(payload, "result_value", "resultValue", "result")
	if resultValue == "" {
		resultValue = "complete"
	}
	recordMap := mapFromAny(payload["record"])
	if len(recordMap) == 0 {
		recordMap = mapFromAny(payload["record_json"])
	}
	if err := runScriptTask(ctx, task, payload, recordMap); err != nil {
		return nil, err
	}
	recordJSON := jsonText(recordMap)
	operatorID := firstUint64(payload, "operator_id", "operatorId")

	recordID := uint64(crmmodel.NewTaskRecordModel().Insert(ctx, map[string]any{
		"resource_id":      task.ResourceID,
		"task_id":          task.ID,
		"task_template_id": task.TaskTemplateID,
		"stage_id":         task.StageID,
		"flow_node_id":     task.FlowNodeID,
		"operator_id":      operatorID,
		"record_json":      recordJSON,
		"result_value":     resultValue,
		"remark":           firstText(payload, "remark"),
		"created_at":       time.Now(),
	}))

	crmmodel.NewResourceTaskModel().Update(ctx, map[string]any{"id": task.ID}, map[string]any{
		"status":       crmmodel.TaskStatusCompleted,
		"result_value": resultValue,
		"result_text":  firstText(payload, "result_text", "resultText"),
		"output_json":  recordJSON,
		"finished_at":  time.Now(),
		"updated_at":   time.Now(),
	})

	nextTaskID := createNextTask(ctx, task, resultValue)
	syncTaskOutput(ctx, task, recordMap)
	if task.ResourceID > 0 {
		crmmodel.NewCustomerResourceModel().Update(ctx, map[string]any{"id": task.ResourceID}, map[string]any{
			"updated_at": time.Now(),
		})
	}

	crmmodel.NewTimelineModel().Insert(ctx, map[string]any{
		"resource_id": task.ResourceID,
		"task_id":     task.ID,
		"event_type":  "task_completed",
		"title":       "任务完成",
		"content":     task.TaskName,
		"operator_id": operatorID,
		"created_at":  time.Now(),
	})

	return map[string]any{
		"record_id":    recordID,
		"next_task_id": nextTaskID,
	}, nil
}

func runScriptTask(ctx context.Context, task *crmmodel.ResourceTask, payload map[string]any, record map[string]any) error {
	if task == nil {
		return nil
	}
	node := crmmodel.NewFlowNodeModel().Find(ctx, map[string]any{"id": task.FlowNodeID})
	template := crmmodel.NewTaskTemplateModel().Find(ctx, map[string]any{"id": task.TaskTemplateID})
	if !isScriptTask(task, node, template) {
		return nil
	}
	scriptID := scriptIDForTask(node, template, payload)
	if scriptID == 0 {
		return fmt.Errorf("脚本任务未绑定脚本规则")
	}
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{"id": scriptID, "status": crmmodel.StatusEnabled})
	if script == nil {
		return fmt.Errorf("脚本规则不存在")
	}
	result, err := NewRuleService().Validate(ctx, ScriptValidateRequest{
		Language:  script.Language,
		Script:    script.Script,
		Entry:     script.Entry,
		Input:     record,
		Config:    map[string]any{},
		TimeoutMS: script.TimeoutMS,
	})
	if err != nil {
		return err
	}
	output := mapFromAny(result.Result.Value)
	for key, value := range output {
		record[key] = value
	}
	return nil
}

func isScriptTask(task *crmmodel.ResourceTask, node *crmmodel.FlowNode, _ *crmmodel.TaskTemplate) bool {
	if task != nil && task.NodeType == crmmodel.NodeTypeScriptEval {
		return true
	}
	if node != nil && node.NodeType == crmmodel.NodeTypeScriptEval {
		return true
	}
	return false
}

func scriptIDForTask(node *crmmodel.FlowNode, _ *crmmodel.TaskTemplate, payload map[string]any) uint64 {
	if scriptID := firstUint64(payload, "script_id", "scriptId"); scriptID > 0 {
		return scriptID
	}
	if node != nil {
		if node.ScriptID > 0 {
			return node.ScriptID
		}
		config := mapFromAny(node.ConfigJSON)
		if scriptID := firstUint64(config, "script_id", "scriptId"); scriptID > 0 {
			return scriptID
		}
	}
	return 0
}

func syncTaskOutput(ctx context.Context, task *crmmodel.ResourceTask, record map[string]any) {
	if task == nil || len(record) == 0 {
		return
	}
	fields := crmmodel.NewTaskFieldModel().Select(ctx, map[string]any{
		"task_template_id": task.TaskTemplateID,
		"status":           crmmodel.StatusEnabled,
	})
	now := time.Now()
	grouped := map[uint64]map[string]any{}
	for _, field := range fields {
		key := taskFieldRecordKey(field)
		value, ok := record[key]
		if !ok && field.Name != "" {
			value, ok = record[field.Name]
		}
		if !ok {
			continue
		}
		if field.MainField != "" {
			syncMainTaskField(ctx, task, field, value, now)
			continue
		}
		if field.DataTemplateID > 0 && field.DataFieldID > 0 {
			dataField := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": field.DataFieldID})
			if dataField != nil {
				if grouped[field.DataTemplateID] == nil {
					grouped[field.DataTemplateID] = map[string]any{}
				}
				grouped[field.DataTemplateID][dataField.Name] = value
			}
		}
	}
	for templateID, values := range grouped {
		saveSyncedDataRecord(ctx, task, templateID, values, now)
	}
}

func saveSyncedDataRecord(ctx context.Context, task *crmmodel.ResourceTask, templateID uint64, values map[string]any, now time.Time) {
	if task == nil || templateID == 0 || len(values) == 0 {
		return
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID})
	if template == nil {
		return
	}
	recordModel := crmmodel.NewDataRecordModel()
	existing := recordModel.Find(ctx, map[string]any{
		"resource_id":      task.ResourceID,
		"data_template_id": templateID,
		"status":           crmmodel.StatusEnabled,
	})
	if existing != nil {
		merged := mapFromAny(existing.RecordJSON)
		for key, value := range values {
			merged[key] = value
		}
		recordModel.Update(ctx, map[string]any{"id": existing.ID}, map[string]any{
			"task_id":     task.ID,
			"record_json": jsonText(merged),
			"summary":     dataRecordSummary(merged),
			"updated_at":  now,
		})
		return
	}
	recordModel.Insert(ctx, map[string]any{
		"resource_id":      task.ResourceID,
		"data_template_id": templateID,
		"task_id":          task.ID,
		"record_json":      jsonText(values),
		"summary":          dataRecordSummary(values),
		"status":           crmmodel.StatusEnabled,
		"sort":             100,
		"created_at":       now,
		"updated_at":       now,
	})
}

var customerCollectColumns = map[string]string{
	"name":   "name",
	"phone":  "phone",
	"wechat": "wechat",
}

var assetCollectColumns = map[string]string{
	"asset_name":   "asset_name",
	"asset_status": "asset_status",
}

func taskFieldRecordKey(field *crmmodel.TaskField) string {
	if field == nil {
		return ""
	}
	if field.FieldSource != "" {
		return field.FieldSource
	}
	if field.MainField != "" {
		return field.MainField
	}
	if field.DataFieldID > 0 {
		return fmt.Sprintf("data:%d", field.DataFieldID)
	}
	return field.Name
}

func syncMainTaskField(ctx context.Context, task *crmmodel.ResourceTask, field *crmmodel.TaskField, value any, now time.Time) {
	if task == nil || field == nil || task.ResourceID == 0 || field.MainField == "" {
		return
	}
	switch field.DataTemplateCateID {
	case crmmodel.CustomerDataTemplateCateID:
		column, ok := customerCollectColumns[field.MainField]
		if !ok {
			return
		}
		resource := crmmodel.NewCustomerResourceModel().Find(ctx, map[string]any{"id": task.ResourceID})
		if resource == nil || resource.CustomerID == 0 {
			return
		}
		crmmodel.NewCustomerModel().Update(ctx, map[string]any{"id": resource.CustomerID}, map[string]any{
			column:       value,
			"updated_at": now,
		})
	case crmmodel.CustomerAssetDataTemplateCateID:
		column, ok := assetCollectColumns[field.MainField]
		if !ok {
			return
		}
		crmmodel.NewCustomerResourceModel().Update(ctx, map[string]any{"id": task.ResourceID}, map[string]any{
			column:       value,
			"updated_at": now,
		})
	}
}

func dataRecordSummary(values map[string]any) string {
	for _, key := range []string{"summary", "title", "name", "decision", "pm_status", "payment_amount"} {
		if text := inputText(values[key]); text != "" {
			return text
		}
	}
	return ""
}

func (TaskService) Assign(ctx context.Context, payload map[string]any) (map[string]any, error) {
	taskID := firstUint64(payload, "task_id", "taskId", "id")
	if taskID == 0 {
		return nil, fmt.Errorf("任务不能为空")
	}
	task := crmmodel.NewResourceTaskModel().Find(ctx, map[string]any{"id": taskID})
	if task == nil {
		return nil, fmt.Errorf("任务不存在")
	}
	if task.Status == crmmodel.TaskStatusCompleted || task.Status == crmmodel.TaskStatusCancelled {
		return nil, fmt.Errorf("已结束任务不能分配")
	}

	departmentID := firstUint64(payload, "assignee_department_id", "assigneeDepartmentId", "department_id", "departmentId")
	roleID := firstUint64(payload, "assignee_role_id", "assigneeRoleId", "role_id", "roleId")
	staffID := firstUint64(payload, "assignee_staff_id", "assigneeStaffId", "staff_id", "staffId")
	if departmentID == 0 && staffID > 0 {
		staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID})
		if staff != nil {
			departmentID = staff.DepartmentID
		}
	}
	if departmentID == 0 && staffID == 0 {
		return nil, fmt.Errorf("处理部门或处理人不能为空")
	}

	update := map[string]any{
		"assignee_department_id": departmentID,
		"assignee_role_id":       roleID,
		"assignee_staff_id":      staffID,
		"updated_at":             time.Now(),
	}
	if staffID > 0 && task.Status == crmmodel.TaskStatusPending {
		update["status"] = crmmodel.TaskStatusProcessing
		update["started_at"] = time.Now()
	}
	if staffID == 0 {
		update["status"] = crmmodel.TaskStatusPending
	}
	crmmodel.NewResourceTaskModel().Update(ctx, map[string]any{"id": task.ID}, update)

	crmmodel.NewTimelineModel().Insert(ctx, map[string]any{
		"resource_id": task.ResourceID,
		"task_id":     task.ID,
		"event_type":  "task_assigned",
		"title":       "任务分配",
		"content":     task.TaskName,
		"operator_id": firstUint64(payload, "operator_id", "operatorId"),
		"created_at":  time.Now(),
	})

	return map[string]any{
		"assigned": true,
		"task_id":  task.ID,
	}, nil
}

func createNextTask(ctx context.Context, task *crmmodel.ResourceTask, resultValue string) uint64 {
	if task == nil {
		return 0
	}
	if nextNode, nextTemplate, nextEdge, ok := nextTaskFromReleaseSnapshot(ctx, task.FlowReleaseID, task.FlowNodeID, task.FlowNodeKey, resultValue); ok {
		if nextNode == nil {
			return 0
		}
		return createTaskFromNodeWithAssignment(ctx, task.ResourceID, task.FlowTemplateID, task.FlowReleaseID, nextNode, nextTemplate, edgeDepartmentID(nextEdge), edgeRoleID(nextEdge))
	}
	edge := nextEdgeFromDB(ctx, task, resultValue)
	if edge == nil || (edge.ToNodeID == 0 && edge.ToNodeKey == "") {
		return 0
	}
	nextNode := crmmodel.NewFlowNodeModel().Find(ctx, map[string]any{"id": edge.ToNodeID})
	if nextNode == nil && edge.ToNodeKey != "" {
		nextNode = crmmodel.NewFlowNodeModel().Find(ctx, map[string]any{
			"flow_template_id": task.FlowTemplateID,
			"node_key":         edge.ToNodeKey,
			"status":           crmmodel.StatusEnabled,
		})
	}
	return createTaskFromNodeWithAssignment(ctx, task.ResourceID, task.FlowTemplateID, task.FlowReleaseID, nextNode, nil, edge.TargetDepartmentID, edge.TargetRoleID)
}

type flowTaskSnapshot struct {
	Nodes         []*crmmodel.FlowNode              `json:"nodes"`
	Edges         []*crmmodel.FlowEdge              `json:"edges"`
	TaskTemplates []*crmmodel.TaskTemplate          `json:"task_templates"`
	Tasks         []*crmmodel.TaskTemplate          `json:"tasks"`
	Fields        map[uint64][]*crmmodel.TaskField  `json:"fields"`
	Results       map[uint64][]*crmmodel.TaskResult `json:"results"`
	Transitions   []*crmmodel.TaskTransition        `json:"transitions"`
}

func firstTaskFromReleaseSnapshot(ctx context.Context, releaseID uint64) (*crmmodel.FlowNode, *crmmodel.TaskTemplate, bool) {
	snapshot, ok := loadFlowTaskSnapshot(ctx, releaseID)
	if !ok {
		return nil, nil, false
	}
	if len(snapshot.Nodes) == 0 {
		return nil, nil, false
	}
	node := snapshot.Nodes[0]
	return node, templateByID(snapshot, node.TaskTemplateID), true
}

func nextTaskFromReleaseSnapshot(ctx context.Context, releaseID uint64, fromNodeID uint64, fromNodeKey string, resultValue string) (*crmmodel.FlowNode, *crmmodel.TaskTemplate, *crmmodel.FlowEdge, bool) {
	snapshot, ok := loadFlowTaskSnapshot(ctx, releaseID)
	if !ok {
		return nil, nil, nil, false
	}
	nextNodeID := uint64(0)
	nextNodeKey := ""
	var matchedEdge *crmmodel.FlowEdge
	for _, edge := range snapshot.Edges {
		if edgeMatches(edge.FromNodeID, edge.FromNodeKey, fromNodeID, fromNodeKey, resultValue, edge.MatchResult) {
			nextNodeID = edge.ToNodeID
			nextNodeKey = edge.ToNodeKey
			matchedEdge = edge
			break
		}
	}
	if nextNodeID == 0 && nextNodeKey == "" {
		for _, edge := range snapshot.Edges {
			if edgeMatches(edge.FromNodeID, edge.FromNodeKey, fromNodeID, fromNodeKey, "", edge.MatchResult) {
				nextNodeID = edge.ToNodeID
				nextNodeKey = edge.ToNodeKey
				matchedEdge = edge
				break
			}
		}
	}
	if nextNodeID == 0 && nextNodeKey == "" {
		return nil, nil, matchedEdge, true
	}
	for _, node := range snapshot.Nodes {
		if (nextNodeID > 0 && node.ID == nextNodeID) || (nextNodeKey != "" && node.NodeKey == nextNodeKey) {
			return node, templateByID(snapshot, node.TaskTemplateID), matchedEdge, true
		}
	}
	return nil, nil, matchedEdge, true
}

func edgeDepartmentID(edge *crmmodel.FlowEdge) uint64 {
	if edge == nil {
		return 0
	}
	return edge.TargetDepartmentID
}

func edgeRoleID(edge *crmmodel.FlowEdge) uint64 {
	if edge == nil {
		return 0
	}
	return edge.TargetRoleID
}

func nextEdgeFromDB(ctx context.Context, task *crmmodel.ResourceTask, resultValue string) *crmmodel.FlowEdge {
	if task == nil {
		return nil
	}
	filters := []map[string]any{
		{
			"flow_template_id": task.FlowTemplateID,
			"from_node_id":     task.FlowNodeID,
			"match_result":     resultValue,
			"status":           crmmodel.StatusEnabled,
		},
		{
			"flow_template_id": task.FlowTemplateID,
			"from_node_id":     task.FlowNodeID,
			"match_result":     "",
			"status":           crmmodel.StatusEnabled,
		},
	}
	if task.FlowNodeKey != "" {
		filters = append(filters,
			map[string]any{
				"flow_template_id": task.FlowTemplateID,
				"from_node_key":    task.FlowNodeKey,
				"match_result":     resultValue,
				"status":           crmmodel.StatusEnabled,
			},
			map[string]any{
				"flow_template_id": task.FlowTemplateID,
				"from_node_key":    task.FlowNodeKey,
				"match_result":     "",
				"status":           crmmodel.StatusEnabled,
			},
		)
	}
	for _, filter := range filters {
		edges := crmmodel.NewFlowEdgeModel().Select(ctx, filter)
		if len(edges) > 0 {
			return edges[0]
		}
	}
	return nil
}

func edgeMatches(edgeNodeID uint64, edgeNodeKey string, nodeID uint64, nodeKey string, resultValue string, edgeResult string) bool {
	if edgeResult != resultValue {
		return false
	}
	if edgeNodeID > 0 && nodeID > 0 && edgeNodeID == nodeID {
		return true
	}
	return edgeNodeKey != "" && nodeKey != "" && edgeNodeKey == nodeKey
}

func templateByID(snapshot flowTaskSnapshot, templateID uint64) *crmmodel.TaskTemplate {
	if templateID == 0 {
		return nil
	}
	for _, template := range snapshot.TaskTemplates {
		if template.ID == templateID {
			return template
		}
	}
	return nil
}

func taskInstanceSnapshotJSON(ctx context.Context, releaseID uint64, node *crmmodel.FlowNode, template *crmmodel.TaskTemplate) string {
	if node == nil && template == nil {
		return "{}"
	}
	snapshot := map[string]any{
		"node":     node,
		"template": template,
	}
	if releaseSnapshot, ok := loadFlowTaskSnapshot(ctx, releaseID); ok {
		templateID := uint64(0)
		if template != nil {
			templateID = template.ID
		}
		snapshot["fields"] = releaseSnapshot.Fields[templateID]
		snapshot["results"] = releaseSnapshot.Results[templateID]
		return jsonText(snapshot)
	}
	if template == nil {
		return jsonText(snapshot)
	}
	snapshot["fields"] = crmmodel.NewTaskFieldModel().Select(ctx, map[string]any{"task_template_id": template.ID, "status": crmmodel.StatusEnabled})
	snapshot["results"] = crmmodel.NewTaskResultModel().Select(ctx, map[string]any{"task_template_id": template.ID, "status": crmmodel.StatusEnabled})
	return jsonText(snapshot)
}

func taskDetailConfig(ctx context.Context, task *crmmodel.ResourceTask) ([]map[string]any, []map[string]any) {
	if task == nil {
		return nil, nil
	}
	snapshot := mapFromAny(task.InputSnapshotJSON)
	if len(snapshot) > 0 {
		fields := sliceMapsFromAny(snapshot["fields"])
		results := sliceMapsFromAny(snapshot["results"])
		if len(fields) > 0 || len(results) > 0 {
			return fields, results
		}
	}
	return crmmodel.NewTaskFieldModel().SelectMap(ctx, map[string]any{"task_template_id": task.TaskTemplateID, "status": crmmodel.StatusEnabled}),
		crmmodel.NewTaskResultModel().SelectMap(ctx, map[string]any{"task_template_id": task.TaskTemplateID, "status": crmmodel.StatusEnabled})
}

func loadFlowTaskSnapshot(ctx context.Context, releaseID uint64) (flowTaskSnapshot, bool) {
	if releaseID == 0 {
		return flowTaskSnapshot{}, false
	}
	release := crmmodel.NewFlowReleaseModel().Find(ctx, map[string]any{"id": releaseID})
	if release == nil || strings.TrimSpace(release.SnapshotJSON) == "" {
		return flowTaskSnapshot{}, false
	}
	var snapshot flowTaskSnapshot
	if err := json.Unmarshal([]byte(release.SnapshotJSON), &snapshot); err != nil {
		return flowTaskSnapshot{}, false
	}
	if len(snapshot.TaskTemplates) == 0 {
		snapshot.TaskTemplates = snapshot.Tasks
	}
	if len(snapshot.Nodes) == 0 || len(snapshot.TaskTemplates) == 0 {
		return flowTaskSnapshot{}, false
	}
	return snapshot, true
}

func sliceMapsFromAny(value any) []map[string]any {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil
	}
	return rows
}
