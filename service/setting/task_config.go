package setting

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func shouldNormalizeTaskConfig(record map[string]any, partial bool) bool {
	for _, field := range []string{
		"task_type",
		"form_id",
		"assign_mode",
		"assign_department_ids",
		"auto_assign_department_id",
		"auto_assign_staff_id",
		"collaboration_items",
		"collaboration_complete_mode",
		"next_stage_code",
		"trigger_type",
		"booking_resource_cate_id",
		"booking_need_confirm",
		"script_id",
		"decision_result_field_path",
		"task_points",
	} {
		if shouldNormalizeCrmField(record, field, partial) {
			return true
		}
	}
	return false
}

func ensureTaskNextStageExists(ctx context.Context, record map[string]any, partial bool) {
	if !shouldNormalizeCrmField(record, "next_stage_code", partial) {
		return
	}
	nextStageCode := util.ToStringTrimmed(record["next_stage_code"])
	if nextStageCode == "" {
		return
	}
	if crmmodel.NewStageModel().Find(ctx, map[string]any{"code": nextStageCode, "status": crmmodel.StatusEnabled}) == nil {
		panicCrmField("form.next_stage_code", "完成后阶段不存在或已停用。")
	}
}

func mergedTaskConfig(record map[string]any, updates map[string]any) map[string]any {
	config := decodeTaskConfig(record["config_json"])
	for key, value := range updates {
		if value == nil {
			delete(config, key)
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			delete(config, key)
			continue
		}
		config[key] = value
	}
	return config
}

func currentTaskConfigValue(ctx context.Context, record map[string]any, key string) any {
	if value := recordTaskConfigValue(record, key); value != nil {
		return value
	}
	if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
		return decodeTaskConfig(current.ConfigJSON)[key]
	}
	return nil
}

func recordTaskConfigValue(record map[string]any, key string) any {
	config := decodeTaskConfig(record["config_json"])
	return config[key]
}

func taskPointsConfigValue(ctx context.Context, record map[string]any, partial bool) any {
	points := effectiveTaskPoints(ctx, record, partial)
	if points <= 0 {
		return nil
	}
	return points
}

func effectiveTaskPoints(ctx context.Context, record map[string]any, partial bool) float64 {
	if shouldNormalizeCrmField(record, "task_points", partial) {
		return normalizeTaskPoints(record["task_points"])
	}
	return normalizeTaskPoints(currentTaskConfigValue(ctx, record, "task_points"))
}

func normalizeTaskPoints(value any) float64 {
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return typed
		}
	case float32:
		if typed > 0 {
			return float64(typed)
		}
	case int:
		if typed > 0 {
			return float64(typed)
		}
	case int64:
		if typed > 0 {
			return float64(typed)
		}
	case uint64:
		if typed > 0 {
			return float64(typed)
		}
	case json.Number:
		number, _ := typed.Float64()
		if number > 0 {
			return number
		}
	default:
		number, _ := strconv.ParseFloat(strings.TrimSpace(util.ToStringTrimmed(value)), 64)
		if number > 0 {
			return number
		}
	}
	return 0
}

func effectiveTaskCompletionMode(ctx context.Context, record map[string]any, partial bool) string {
	if shouldNormalizeCrmField(record, "completion_mode", partial) {
		return normalizeTaskCompletionMode(record["completion_mode"])
	}
	return normalizeTaskCompletionMode(currentTaskConfigValue(ctx, record, "completion_mode"))
}

func encodeTaskConfig(config map[string]any) string {
	encoded, err := json.Marshal(config)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func taskConfigObject(value any) map[string]any {
	if row, ok := value.(map[string]any); ok {
		return row
	}
	return decodeTaskConfig(value)
}

func optionalTaskConfigID(value any) any {
	id := util.ToUint64(value)
	if id == 0 {
		return ""
	}
	return id
}

func normalizeTaskCompletionMode(value any) string {
	if util.ToStringTrimmed(value) == crmmodel.TaskCompletionManual {
		return crmmodel.TaskCompletionManual
	}
	return crmmodel.TaskCompletionSubmit
}

func firstTaskConfigValue(row map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			return value
		}
	}
	return nil
}

func taskConfigBoolDefault(value any, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return util.ToBool(value)
}

func taskConfigSort(value any, index int) int {
	sort := util.ToIntDefault(value, 0)
	if sort == 0 {
		return (index + 1) * 10
	}
	return sort
}

func decodeTaskConfig(value any) map[string]any {
	if row, ok := value.(map[string]any); ok {
		return util.CloneMap(row)
	}
	raw := util.ToStringTrimmed(value)
	if raw == "" {
		return map[string]any{}
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return map[string]any{}
	}
	return decoded
}

func taskConfigRows(value any) []map[string]any {
	switch rows := value.(type) {
	case []map[string]any:
		return rows
	case []any:
		result := make([]map[string]any, 0, len(rows))
		for _, item := range rows {
			if row, ok := item.(map[string]any); ok {
				result = append(result, row)
			}
		}
		return result
	case string:
		raw := strings.TrimSpace(rows)
		if raw == "" {
			return nil
		}
		var mappedRows []map[string]any
		if err := json.Unmarshal([]byte(raw), &mappedRows); err == nil {
			return mappedRows
		}
		var anyRows []any
		if err := json.Unmarshal([]byte(raw), &anyRows); err == nil {
			return taskConfigRows(anyRows)
		}
		return nil
	default:
		return nil
	}
}
