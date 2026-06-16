package setting

import (
	"context"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

func (CrmHook) ProviderBeforeSaveTask(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "task_type", partial)
	trimCrmStringField(record, "trigger_type", partial)
	trimCrmStringField(record, "assign_mode", partial)
	trimCrmStringField(record, "next_stage_code", partial)
	trimCrmStringField(record, "booking_need_confirm", partial)
	if !partial {
		if util.ToUint64(record["stage_id"]) == 0 {
			panicCrmField("form.stage_id", "所属阶段不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "任务名称不能为空。")
		}
	}
	ensureTaskNextStageExists(contextFromServer(c), record, partial)
	ensureUniqueTaskName(contextFromServer(c), record, partial)
	normalizeTaskVisibleWhen(contextFromServer(c), record, partial)
	defaultCrmInt(record, "stage_id", 0, partial)
	if shouldNormalizeCrmField(record, "task_type", partial) && util.ToStringTrimmed(record["task_type"]) == "" {
		record["task_type"] = crmmodel.TaskTypeForm
	}
	if shouldNormalizeCrmField(record, "trigger_type", partial) && util.ToStringTrimmed(record["trigger_type"]) == "" {
		record["trigger_type"] = crmmodel.TaskTriggerManual
	}
	normalizeTaskAutoTriggerSupport(contextFromServer(c), record, partial)
	normalizeTaskTriggerConfig(record, partial)
	normalizeTaskTypeConfig(contextFromServer(c), record, partial)
	ensureTaskFormExists(contextFromServer(c), record, partial)
	defaultCrmInt(record, "form_id", 0, partial)
	defaultCrmInt(record, "script_id", 0, partial)
	defaultCrmInt(record, "trigger_task_id", 0, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func ensureUniqueTaskName(ctx context.Context, record map[string]any, partial bool) {
	taskID := util.ToUint64(record["id"])
	stageID := util.ToUint64(record["stage_id"])
	name := util.ToStringTrimmed(record["name"])

	if partial && taskID > 0 && (stageID == 0 || name == "") {
		if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": taskID}); current != nil {
			if stageID == 0 {
				stageID = current.StageID
			}
			if name == "" {
				name = current.Name
			}
		}
	}

	if stageID == 0 || name == "" {
		return
	}

	filters := map[string]any{
		"stage_id": stageID,
		"name":     name,
	}
	if taskID > 0 {
		filters["id"] = map[string]any{"!=": taskID}
	}
	if crmmodel.NewTaskModel().Count(ctx, filters) > 0 {
		panicCrmField("form.name", "当前阶段下任务名称已存在。")
	}
}

func normalizeTaskTriggerConfig(record map[string]any, partial bool) {
	if !shouldNormalizeCrmField(record, "trigger_type", partial) && !shouldNormalizeCrmField(record, "trigger_task_id", partial) {
		return
	}
	triggerType := util.ToStringTrimmed(record["trigger_type"])
	if triggerType == "" {
		triggerType = crmmodel.TaskTriggerManual
		record["trigger_type"] = triggerType
	}
	if triggerType == crmmodel.TaskTriggerManual || triggerType == crmmodel.TaskTriggerStageEnter {
		record["trigger_task_id"] = uint64(0)
		return
	}
	if !partial && triggerType == crmmodel.TaskTriggerAfterTask && util.ToUint64(record["trigger_task_id"]) == 0 {
		panicCrmField("form.trigger_task_id", "任务后触发必须选择触发任务。")
	}
}

func normalizeTaskAutoTriggerSupport(ctx context.Context, record map[string]any, partial bool) {
	if !shouldNormalizeCrmField(record, "trigger_type", partial) && !shouldNormalizeCrmField(record, "task_type", partial) {
		return
	}
	if effectiveTaskTriggerType(ctx, record, partial) == crmmodel.TaskTriggerManual {
		return
	}
	taskType := effectiveTaskType(ctx, record, partial)
	if crmmodel.TaskTypeSupportsAutoTrigger(taskType) {
		return
	}
	record["trigger_type"] = crmmodel.TaskTriggerManual
	record["trigger_task_id"] = uint64(0)
}

func effectiveTaskType(ctx context.Context, record map[string]any, partial bool) string {
	taskType := util.ToStringTrimmed(record["task_type"])
	if taskType != "" || !partial {
		return taskType
	}
	if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
		return current.TaskType
	}
	return ""
}

func effectiveTaskTriggerType(ctx context.Context, record map[string]any, partial bool) string {
	triggerType := util.ToStringTrimmed(record["trigger_type"])
	if triggerType != "" || !partial {
		return triggerType
	}
	if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
		return current.TriggerType
	}
	return crmmodel.TaskTriggerManual
}

func effectiveTaskNextStageCode(ctx context.Context, record map[string]any, partial bool) string {
	if shouldNormalizeCrmField(record, "next_stage_code", partial) {
		return util.ToStringTrimmed(record["next_stage_code"])
	}
	return util.ToStringTrimmed(currentTaskConfigValue(ctx, record, "next_stage_code"))
}

func ensureTaskFormExists(ctx context.Context, record map[string]any, partial bool) {
	taskType := util.ToStringTrimmed(record["task_type"])
	formID := util.ToUint64(record["form_id"])
	if partial && (taskType == "" || formID == 0) {
		if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
			if taskType == "" {
				taskType = current.TaskType
			}
			if formID == 0 {
				formID = current.FormID
			}
		}
	}
	if taskType != crmmodel.TaskTypeCreate && taskType != crmmodel.TaskTypeForm {
		return
	}
	if formID == 0 {
		return
	}
	form := crmmodel.NewFormModel().Find(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled})
	if form == nil {
		panicCrmField("form.form_id", "资料模板不存在或已停用。")
	}
}
