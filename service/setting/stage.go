package setting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

func (CrmHook) ProviderBeforeSaveStage(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "code", partial)
	trimCrmStringField(record, "name", partial)
	if !partial && util.ToStringTrimmed(record["code"]) == "" {
		panicCrmField("form.code", "状态码不能为空。")
	}
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "阶段名称不能为空。")
	}
	defaultCrmInt(record, "owner_department_id", 0, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBeforeSaveStageTransition(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "from_stage_code", partial)
	trimCrmStringField(record, "to_stage_code", partial)
	trimCrmStringField(record, "result_value", partial)
	trimCrmStringField(record, "owner_mode", partial)
	if shouldNormalizeCrmField(record, "condition_json", partial) {
		record["condition_json"] = normalizeConditionJSON(record["condition_json"])
	}
	if !partial && util.ToStringTrimmed(record["from_stage_code"]) == "" {
		panicCrmField("form.from_stage_code", "来源阶段不能为空。")
	}
	if !partial && util.ToStringTrimmed(record["to_stage_code"]) == "" {
		panicCrmField("form.to_stage_code", "目标阶段不能为空。")
	}
	if shouldNormalizeCrmField(record, "owner_mode", partial) && util.ToStringTrimmed(record["owner_mode"]) == "" {
		record["owner_mode"] = crmmodel.StageOwnerKeep
	}
	defaultCrmInt(record, "task_id", 0, partial)
	defaultCrmInt(record, "script_id", 0, partial)
	defaultCrmInt(record, "to_department_id", 0, partial)
	defaultCrmInt(record, "to_staff_id", 0, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func normalizeConditionJSON(value any) string {
	raw := util.ToStringTrimmed(value)
	if raw == "" {
		return "{}"
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		panicCrmField("form.condition_json", fmt.Sprintf("条件JSON格式错误：%s", err.Error()))
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func (CrmHook) ProviderBuildStageRows(c *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}
	namesByStageID := taskNamesByStageID(contextFromServer(c), rows)
	for _, row := range rows {
		stageID := util.ToUint64(row["id"])
		row["task_names"] = strings.Join(namesByStageID[stageID], "、")
		row["task_count"] = len(namesByStageID[stageID])
	}
	return rows
}

func (CrmHook) ProviderBuildStageTransitionRows(_ *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}
	for _, row := range rows {
		row["owner_mode_name"] = stageOwnerModeName(row["owner_mode"])
		if util.ToStringTrimmed(row["result_value"]) == "" {
			row["result_value_display"] = "任意结果"
		} else {
			row["result_value_display"] = row["result_value"]
		}
	}
	return rows
}

func stageOwnerModeName(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.StageOwnerAssign:
		return "使用分配结果"
	case crmmodel.StageOwnerFixedDepartment:
		return "固定部门"
	case crmmodel.StageOwnerFixedStaff:
		return "固定人员"
	case crmmodel.StageOwnerCreator:
		return "创建人"
	default:
		return "保持当前"
	}
}

func taskNamesByStageID(ctx context.Context, rows []map[string]any) map[uint64][]string {
	stageIDs := make(map[uint64]bool)
	for _, row := range rows {
		stageID := util.ToUint64(row["id"])
		if stageID > 0 {
			stageIDs[stageID] = true
		}
	}
	if len(stageIDs) == 0 {
		return map[uint64][]string{}
	}
	result := make(map[uint64][]string, len(stageIDs))
	for _, task := range crmmodel.NewTaskModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled}) {
		if task == nil || task.StageID == 0 || !stageIDs[task.StageID] {
			continue
		}
		name := strings.TrimSpace(task.Name)
		if name != "" {
			result[task.StageID] = append(result[task.StageID], name)
		}
	}
	return result
}

func contextFromServer(c *server.Context) context.Context {
	if c == nil {
		return context.Background()
	}
	return c.Context()
}
