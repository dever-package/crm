package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	crmmodel "my/package/crm/model"
)

type FlowReleaseService struct{}

func NewFlowReleaseService() FlowReleaseService {
	return FlowReleaseService{}
}

func (FlowReleaseService) Publish(ctx context.Context, flowTemplateID uint64) (map[string]any, error) {
	flow := crmmodel.NewFlowTemplateModel().Find(ctx, map[string]any{"id": flowTemplateID})
	if flow == nil {
		return nil, fmt.Errorf("流程模板不存在")
	}
	stages := crmmodel.NewFlowStageModel().Select(ctx, map[string]any{"flow_template_id": flowTemplateID, "status": crmmodel.StatusEnabled})
	nodes := crmmodel.NewFlowNodeModel().Select(ctx, map[string]any{"flow_template_id": flowTemplateID, "status": crmmodel.StatusEnabled})
	edges := crmmodel.NewFlowEdgeModel().Select(ctx, map[string]any{"flow_template_id": flowTemplateID, "status": crmmodel.StatusEnabled})
	taskTemplates := taskTemplatesByNodes(ctx, nodes)
	if len(stages) == 0 || len(nodes) == 0 {
		return nil, fmt.Errorf("发布前至少需要一个阶段和一个节点")
	}

	version := flow.ReleaseVersion + 1
	snapshot, err := json.Marshal(map[string]any{
		"flow":           flow,
		"stages":         stages,
		"nodes":          nodes,
		"edges":          edges,
		"task_templates": taskTemplates,
		"fields":         fieldsByTasks(ctx, taskTemplates),
		"results":        resultsByTasks(ctx, taskTemplates),
		"data_templates": referencedDataTemplates(ctx, taskTemplates),
		"data_fields":    referencedDataFields(ctx, taskTemplates),
		"rule_scripts":   referencedRuleScripts(ctx, nodes, edges),
	})
	if err != nil {
		return nil, fmt.Errorf("生成发布快照失败: %w", err)
	}

	releaseID := uint64(crmmodel.NewFlowReleaseModel().Insert(ctx, map[string]any{
		"flow_template_id": flowTemplateID,
		"version":          version,
		"snapshot_json":    string(snapshot),
		"status":           crmmodel.StatusEnabled,
		"created_at":       time.Now(),
	}))
	if releaseID == 0 {
		return nil, fmt.Errorf("流程发布失败")
	}
	crmmodel.NewFlowTemplateModel().Update(ctx, map[string]any{"id": flowTemplateID}, map[string]any{
		"publish_status":     crmmodel.PublishStatusPublished,
		"current_release_id": releaseID,
		"release_version":    version,
		"updated_at":         time.Now(),
	})

	return map[string]any{
		"published":    true,
		"release_id":   releaseID,
		"version":      version,
		"snapshot_len": len(snapshot),
	}, nil
}

func taskTemplatesByNodes(ctx context.Context, nodes []*crmmodel.FlowNode) []*crmmodel.TaskTemplate {
	templates := make([]*crmmodel.TaskTemplate, 0, len(nodes))
	seen := map[uint64]bool{}
	for _, node := range nodes {
		if node.TaskTemplateID == 0 || seen[node.TaskTemplateID] {
			continue
		}
		template := crmmodel.NewTaskTemplateModel().Find(ctx, map[string]any{"id": node.TaskTemplateID, "status": crmmodel.StatusEnabled})
		if template == nil {
			continue
		}
		seen[template.ID] = true
		templates = append(templates, template)
	}
	return templates
}

func fieldsByTasks(ctx context.Context, tasks []*crmmodel.TaskTemplate) map[uint64][]*crmmodel.TaskField {
	fields := map[uint64][]*crmmodel.TaskField{}
	for _, task := range tasks {
		fields[task.ID] = crmmodel.NewTaskFieldModel().Select(ctx, map[string]any{"task_template_id": task.ID, "status": crmmodel.StatusEnabled})
	}
	return fields
}

func resultsByTasks(ctx context.Context, tasks []*crmmodel.TaskTemplate) map[uint64][]*crmmodel.TaskResult {
	results := map[uint64][]*crmmodel.TaskResult{}
	for _, task := range tasks {
		results[task.ID] = crmmodel.NewTaskResultModel().Select(ctx, map[string]any{"task_template_id": task.ID, "status": crmmodel.StatusEnabled})
	}
	return results
}

func referencedDataTemplates(ctx context.Context, tasks []*crmmodel.TaskTemplate) map[uint64]*crmmodel.DataTemplate {
	templates := map[uint64]*crmmodel.DataTemplate{}
	for _, fields := range fieldsByTasks(ctx, tasks) {
		for _, field := range fields {
			if field.DataTemplateID == 0 {
				continue
			}
			if _, exists := templates[field.DataTemplateID]; exists {
				continue
			}
			template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": field.DataTemplateID})
			if template != nil {
				templates[template.ID] = template
			}
		}
	}
	return templates
}

func referencedDataFields(ctx context.Context, tasks []*crmmodel.TaskTemplate) map[uint64]*crmmodel.DataField {
	fields := map[uint64]*crmmodel.DataField{}
	for _, taskFields := range fieldsByTasks(ctx, tasks) {
		for _, taskField := range taskFields {
			if taskField.DataFieldID == 0 {
				continue
			}
			if _, exists := fields[taskField.DataFieldID]; exists {
				continue
			}
			dataField := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": taskField.DataFieldID})
			if dataField != nil {
				fields[dataField.ID] = dataField
			}
		}
	}
	return fields
}

func referencedRuleScripts(ctx context.Context, nodes []*crmmodel.FlowNode, edges []*crmmodel.FlowEdge) map[uint64]*crmmodel.RuleScript {
	scripts := map[uint64]*crmmodel.RuleScript{}
	addScript := func(scriptID uint64) {
		if scriptID == 0 {
			return
		}
		if _, exists := scripts[scriptID]; exists {
			return
		}
		script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{"id": scriptID})
		if script != nil {
			scripts[script.ID] = script
		}
	}
	for _, node := range nodes {
		addScript(node.ScriptID)
	}
	for _, edge := range edges {
		addScript(edge.MatchScriptID)
	}
	return scripts
}
