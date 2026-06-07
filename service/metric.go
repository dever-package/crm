package service

import (
	"context"
	"strings"

	crmmodel "my/package/crm/model"
)

type MetricService struct{}

func NewMetricService() MetricService {
	return MetricService{}
}

func (MetricService) Summary(ctx context.Context) (map[string]any, error) {
	resources := crmmodel.NewCustomerResourceModel().SelectMap(ctx, map[string]any{})
	tasks := crmmodel.NewResourceTaskModel().SelectMap(ctx, map[string]any{})
	pending := 0
	completed := 0
	blocked := 0
	for _, task := range tasks {
		switch inputText(task["status"]) {
		case crmmodel.TaskStatusPending, crmmodel.TaskStatusProcessing, crmmodel.TaskStatusWaitingReview:
			pending++
		case crmmodel.TaskStatusCompleted:
			completed++
		case crmmodel.TaskStatusBlocked:
			blocked++
		}
	}
	return map[string]any{
		"resource_total": len(resources),
		"task_total":     len(tasks),
		"task_pending":   pending,
		"task_completed": completed,
		"task_blocked":   blocked,
		"task_overdue":   0,
	}, nil
}

func (MetricService) Funnel(ctx context.Context) (map[string]any, error) {
	nodes := crmmodel.NewFlowNodeModel().SelectMap(ctx, map[string]any{
		"flow_template_id": 1,
		"status":           crmmodel.StatusEnabled,
	})
	records := crmmodel.NewTaskRecordModel().SelectMap(ctx, map[string]any{})
	countByNode := map[uint64]int{}
	for _, record := range records {
		countByNode[inputUint64(record["flow_node_id"])]++
	}
	steps := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		nodeID := inputUint64(node["id"])
		steps = append(steps, map[string]any{
			"id":    nodeID,
			"name":  node["name"],
			"count": countByNode[nodeID],
		})
	}
	return map[string]any{
		"steps": steps,
	}, nil
}

func (MetricService) Widget(ctx context.Context, key string) (map[string]any, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return map[string]any{"key": "", "value": 0}, nil
	}
	rows := crmmodel.NewMetricRecordModel().SelectMap(ctx, map[string]any{"metric_key": key})
	sum := float64(0)
	groups := map[string]int{}
	for _, row := range rows {
		sum += numericValue(row["metric_value_number"])
		if text := inputText(row["metric_value_text"]); text != "" {
			groups[text]++
		}
	}
	return map[string]any{
		"key":    key,
		"count":  len(rows),
		"sum":    sum,
		"groups": groups,
	}, nil
}
