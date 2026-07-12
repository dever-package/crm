package setting

import (
	"context"

	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func normalizeTaskTypeConfig(ctx context.Context, record map[string]any, partial bool) {
	if !shouldNormalizeTaskConfig(record, partial) {
		return
	}

	base := taskConfigBase{
		NextStageCode: effectiveTaskNextStageCode(ctx, record, partial),
		TaskPoints:    taskPointsConfigValue(ctx, record, partial),
	}
	switch effectiveTaskType(ctx, record, partial) {
	case crmmodel.TaskTypeCreate:
		normalizeCreateTaskConfig(record, partial, base)
	case crmmodel.TaskTypeForm:
		normalizeFormTaskConfig(ctx, record, partial, base)
	case crmmodel.TaskTypeAssign:
		normalizeAssignTaskConfig(ctx, record, partial, base)
	case crmmodel.TaskTypeCollaborate:
		normalizeCollaborateTaskConfig(ctx, record, partial, base)
	case crmmodel.TaskTypeBooking:
		normalizeBookingTaskConfig(ctx, record, partial, base)
	case crmmodel.TaskTypeDecision:
		normalizeDecisionTaskConfig(ctx, record, partial, base)
	default:
		normalizeFallbackTaskConfig(record, base)
	}
}

type taskConfigBase struct {
	NextStageCode string
	TaskPoints    any
}

func (base taskConfigBase) updates() map[string]any {
	return map[string]any{
		"next_stage_code": base.NextStageCode,
		"task_points":     base.TaskPoints,
	}
}

func normalizeCreateTaskConfig(record map[string]any, partial bool, base taskConfigBase) {
	if !partial && util.ToUint64(record["form_id"]) == 0 {
		panicCrmField("form.form_id", "创建资料任务必须选择资料模板。")
	}
	record["script_id"] = uint64(0)
	record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, base.updates()))
}

func normalizeFormTaskConfig(ctx context.Context, record map[string]any, partial bool, base taskConfigBase) {
	if !partial && util.ToUint64(record["form_id"]) == 0 {
		panicCrmField("form.form_id", "填写资料任务必须选择资料模板。")
	}
	updates := base.updates()
	updates["completion_mode"] = effectiveTaskCompletionMode(ctx, record, partial)
	if assignTaskID := effectiveTaskCompleteAssignTaskID(ctx, record, partial); assignTaskID > 0 {
		updates[crmmodel.TaskCompleteAssignTaskID] = assignTaskID
	} else {
		updates[crmmodel.TaskCompleteAssignTaskID] = nil
	}
	record["script_id"] = uint64(0)
	record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, updates))
}

func normalizeAssignTaskConfig(ctx context.Context, record map[string]any, partial bool, base taskConfigBase) {
	record["script_id"] = uint64(0)
	assignMode := effectiveTaskAssignMode(ctx, record, partial)
	departmentIDs := effectiveTaskAssignDepartmentIDs(ctx, record, partial)
	if len(departmentIDs) == 0 {
		panicCrmField("form.assign_department_ids", "可选部门不能为空。")
	}

	autoDepartmentID, autoStaffID := normalizeTaskAutoAssignTarget(ctx, record, assignMode, departmentIDs, partial)
	updates := base.updates()
	updates["assign_mode"] = assignMode
	updates["assign_department_ids"] = departmentIDs
	updates["auto_assign_department_id"] = autoDepartmentID
	updates["auto_assign_staff_id"] = autoStaffID
	if effectiveTaskHiddenFromWorkList(ctx, record, partial) {
		updates["hide_from_work_list"] = true
	} else {
		updates["hide_from_work_list"] = nil
	}
	if effectiveTaskTriggerType(ctx, record, partial) == crmmodel.TaskTriggerManual {
		updates["auto_assign_department_id"] = nil
		updates["auto_assign_staff_id"] = nil
	}

	record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, updates))
	record["form_id"] = uint64(0)
}

func normalizeCollaborateTaskConfig(ctx context.Context, record map[string]any, partial bool, base taskConfigBase) {
	record["script_id"] = uint64(0)
	record["form_id"] = uint64(0)
	items := effectiveTaskCollaborationItems(ctx, record, partial)
	if !partial && len(items) == 0 {
		panicCrmField("form.collaboration_items", "协作任务必须配置子任务。")
	}
	ensureAutoTaskCollaborationStaff(items, effectiveTaskTriggerType(ctx, record, partial))

	updates := base.updates()
	updates["collaboration_items"] = items
	updates["collaboration_complete_mode"] = effectiveTaskCollaborationCompleteMode(ctx, record, partial)
	record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, updates))
}

func normalizeBookingTaskConfig(ctx context.Context, record map[string]any, partial bool, base taskConfigBase) {
	updates := base.updates()
	updates["resource_cate_id"] = effectiveTaskBookingResourceCateID(ctx, record, partial)
	updates["need_confirm"] = effectiveTaskBookingNeedConfirm(ctx, record, partial)
	record["form_id"] = uint64(0)
	record["script_id"] = uint64(0)
	record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, updates))
}

func normalizeDecisionTaskConfig(ctx context.Context, record map[string]any, partial bool, base taskConfigBase) {
	if decisionTaskIsManual(ctx, record, partial) {
		record["script_id"] = uint64(0)
	} else if effectiveTaskScriptID(ctx, record, partial) == 0 {
		panicCrmField("form.script_id", "自动触发的决策任务必须选择脚本规则。")
	}

	updates := base.updates()
	updates["decision_result_field_id"] = normalizeDecisionResultField(ctx, record, partial)
	record["form_id"] = uint64(0)
	record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, updates))
}

func normalizeFallbackTaskConfig(record map[string]any, base taskConfigBase) {
	record["task_type"] = crmmodel.TaskTypeForm
	record["script_id"] = uint64(0)
	record["config_json"] = encodeTaskConfig(mergedTaskConfig(record, base.updates()))
}

func decisionTaskIsManual(ctx context.Context, record map[string]any, partial bool) bool {
	return effectiveTaskTriggerType(ctx, record, partial) == crmmodel.TaskTriggerManual
}

func effectiveTaskScriptID(ctx context.Context, record map[string]any, partial bool) uint64 {
	if scriptID := util.ToUint64(record["script_id"]); scriptID > 0 || !partial {
		return scriptID
	}
	if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
		return current.ScriptID
	}
	return 0
}
