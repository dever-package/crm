package setting

import (
	"context"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

var simpleTaskTypes = map[string]bool{
	crmmodel.TaskTypeTodo:     true,
	crmmodel.TaskTypeForm:     true,
	crmmodel.TaskTypeApproval: true,
	crmmodel.TaskTypeRule:     true,
}

var simpleTaskAssigneeModes = map[string]bool{
	crmmodel.TaskAssigneeStage:      true,
	crmmodel.TaskAssigneeDepartment: true,
	crmmodel.TaskAssigneeStaff:      true,
}

func (CrmHook) ProviderBeforeSaveWorkflow(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "name", partial)
	validateConfigName(record, partial, "流程名称不能为空。")
	defaultCrmInt(record, "sort", 100, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusDisabled, partial)
	if recordEnablesConfig(record, partial) {
		validateWorkflowCanEnable(contextFromServer(c), util.ToUint64(record["id"]))
	}
	return record
}

func (CrmHook) ProviderBeforeSaveStage(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "name", partial)
	validateConfigName(record, partial, "阶段名称不能为空。")

	ctx := contextFromServer(c)
	workflowID := effectiveStageWorkflowID(ctx, record, partial)
	if workflowID == 0 || crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": workflowID}) == nil {
		panicCrmField("form.workflow_id", "所属流程不存在。")
	}
	if shouldNormalizeCrmField(record, "workflow_id", partial) {
		record["workflow_id"] = workflowID
	}
	defaultCrmInt(record, "owner_department_id", 0, partial)
	defaultCrmInt(record, "sort", 100, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusDisabled, partial)
	if recordEnablesConfig(record, partial) {
		validateStageCanEnable(ctx, util.ToUint64(record["id"]))
	}
	return record
}

func (CrmHook) ProviderBeforeSaveTask(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "task_type", partial)
	trimCrmStringField(record, "assignee_mode", partial)
	validateConfigName(record, partial, "任务名称不能为空。")

	ctx := contextFromServer(c)
	effective := effectiveTaskConfig(ctx, record, partial)
	stageID := util.ToUint64(effective["stage_id"])
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": stageID})
	if stage == nil || stage.WorkflowID == 0 {
		panicCrmField("form.stage_id", "所属阶段不存在。")
	}
	if shouldNormalizeCrmField(record, "stage_id", partial) {
		record["stage_id"] = stageID
	}

	taskType := util.ToStringTrimmed(effective["task_type"])
	if !simpleTaskTypes[taskType] {
		panicCrmField("form.task_type", "任务类型无效。")
	}
	taskEnabled := int16(util.ToIntDefault(effective["status"], int(crmmodel.StatusEnabled))) == crmmodel.StatusEnabled
	normalizeSimpleTaskTarget(ctx, record, effective, partial, taskType, taskEnabled)
	normalizeSimpleTaskAssignee(ctx, record, effective, partial, taskEnabled)

	defaultCrmBool(record, "required", true, partial)
	defaultCrmInt(record, "due_days", 0, partial)
	if shouldNormalizeCrmField(record, "due_days", partial) && util.ToIntDefault(record["due_days"], 0) < 0 {
		panicCrmField("form.due_days", "办理期限不能小于 0 天。")
	}
	defaultCrmInt(record, "sort", 100, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	return record
}

func validateConfigName(record map[string]any, partial bool, message string) {
	if shouldNormalizeCrmField(record, "name", partial) && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", message)
	}
}

func recordEnablesConfig(record map[string]any, partial bool) bool {
	return shouldNormalizeCrmField(record, "status", partial) &&
		int16(util.ToIntDefault(record["status"], 0)) == crmmodel.StatusEnabled
}

func validateWorkflowCanEnable(ctx context.Context, workflowID uint64) {
	if workflowID == 0 || crmmodel.NewStageModel().Count(ctx, map[string]any{
		"workflow_id": workflowID,
		"status":      crmmodel.StatusEnabled,
	}) == 0 {
		panicCrmField("form.status", "流程至少包含一个已启用阶段后才能启用。")
	}
}

func validateStageCanEnable(ctx context.Context, stageID uint64) {
	if stageID == 0 || crmmodel.NewTaskModel().Count(ctx, map[string]any{
		"stage_id": stageID,
		"required": true,
		"status":   crmmodel.StatusEnabled,
	}) == 0 {
		panicCrmField("form.status", "阶段至少包含一个已启用的必做任务后才能启用。")
	}
}

func effectiveStageWorkflowID(ctx context.Context, record map[string]any, partial bool) uint64 {
	if workflowID := util.ToUint64(record["workflow_id"]); workflowID > 0 || !partial {
		return workflowID
	}
	if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); stage != nil {
		return stage.WorkflowID
	}
	return 0
}

func effectiveTaskConfig(ctx context.Context, record map[string]any, partial bool) map[string]any {
	effective := map[string]any{
		"task_type":     crmmodel.TaskTypeTodo,
		"assignee_mode": crmmodel.TaskAssigneeStage,
		"status":        crmmodel.StatusEnabled,
	}
	if partial {
		if task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); task != nil {
			effective = map[string]any{
				"stage_id":               task.StageID,
				"task_type":              task.TaskType,
				"assignee_mode":          task.AssigneeMode,
				"assignee_department_id": task.AssigneeDepartmentID,
				"assignee_staff_id":      task.AssigneeStaffID,
				"form_id":                task.FormID,
				"script_id":              task.ScriptID,
				"status":                 task.Status,
			}
		}
	}
	for key, value := range record {
		effective[key] = value
	}
	if util.ToStringTrimmed(effective["task_type"]) == "" {
		effective["task_type"] = crmmodel.TaskTypeTodo
		if shouldNormalizeCrmField(record, "task_type", partial) {
			record["task_type"] = crmmodel.TaskTypeTodo
		}
	}
	if util.ToStringTrimmed(effective["assignee_mode"]) == "" {
		effective["assignee_mode"] = crmmodel.TaskAssigneeStage
		if shouldNormalizeCrmField(record, "assignee_mode", partial) {
			record["assignee_mode"] = crmmodel.TaskAssigneeStage
		}
	}
	return effective
}

func normalizeSimpleTaskTarget(ctx context.Context, record map[string]any, effective map[string]any, partial bool, taskType string, validateTarget bool) {
	formID := util.ToUint64(effective["form_id"])
	scriptID := util.ToUint64(effective["script_id"])
	shouldWrite := !partial || shouldNormalizeCrmField(record, "task_type", partial)
	switch taskType {
	case crmmodel.TaskTypeForm:
		if validateTarget && (formID == 0 || crmmodel.NewFormModel().Find(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled}) == nil) {
			panicCrmField("form.form_id", "填写资料任务必须选择已启用的资料表单。")
		}
		if shouldWrite || shouldNormalizeCrmField(record, "script_id", partial) {
			record["script_id"] = uint64(0)
		}
	case crmmodel.TaskTypeRule:
		if validateTarget && (scriptID == 0 || crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{"id": scriptID, "status": crmmodel.StatusEnabled}) == nil) {
			panicCrmField("form.script_id", "自动核验任务必须选择已启用的规则。")
		}
		if shouldWrite || shouldNormalizeCrmField(record, "form_id", partial) {
			record["form_id"] = uint64(0)
		}
	default:
		if shouldWrite || shouldNormalizeCrmField(record, "form_id", partial) {
			record["form_id"] = uint64(0)
		}
		if shouldWrite || shouldNormalizeCrmField(record, "script_id", partial) {
			record["script_id"] = uint64(0)
		}
	}
}

func normalizeSimpleTaskAssignee(ctx context.Context, record map[string]any, effective map[string]any, partial bool, validateTarget bool) {
	mode := util.ToStringTrimmed(effective["assignee_mode"])
	if !simpleTaskAssigneeModes[mode] {
		panicCrmField("form.assignee_mode", "负责方式无效。")
	}
	shouldWrite := !partial || shouldNormalizeCrmField(record, "assignee_mode", partial)
	switch mode {
	case crmmodel.TaskAssigneeStage:
		if shouldWrite {
			record["assignee_department_id"] = uint64(0)
			record["assignee_staff_id"] = uint64(0)
		}
	case crmmodel.TaskAssigneeDepartment:
		departmentID := util.ToUint64(effective["assignee_department_id"])
		if validateTarget && (departmentID == 0 || crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": departmentID, "status": crmmodel.StatusEnabled}) == nil) {
			panicCrmField("form.assignee_department_id", "指定部门任务必须选择已启用的部门。")
		}
		if shouldWrite || shouldNormalizeCrmField(record, "assignee_staff_id", partial) {
			record["assignee_staff_id"] = uint64(0)
		}
	case crmmodel.TaskAssigneeStaff:
		staffID := util.ToUint64(effective["assignee_staff_id"])
		staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID, "status": crmmodel.StatusEnabled})
		if validateTarget && staff == nil {
			panicCrmField("form.assignee_staff_id", "指定人员任务必须选择已启用的人员。")
		}
		if staff == nil {
			return
		}
		if shouldWrite || shouldNormalizeCrmField(record, "assignee_staff_id", partial) {
			record["assignee_department_id"] = staff.DepartmentID
		}
	}
}

func (CrmHook) ProviderBeforeDeleteWorkflow(c *server.Context, params []any) any {
	id := configDeleteID(params)
	if id == 0 {
		panicCrmField("form.id", "流程不存在。")
	}
	ctx := contextFromServer(c)
	if crmmodel.NewCustomerStageModel().Count(ctx, map[string]any{"workflow_id": id, "status": crmmodel.ProgressStatusActive}) > 0 ||
		crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{"workflow_id": id, "status": crmmodel.WorkTodoStatusPending}) > 0 {
		panicCrmField("form.id", "流程正在使用中，不能删除；可以先停用。")
	}
	if crmmodel.NewStageModel().Count(ctx, map[string]any{"workflow_id": id}) > 0 {
		panicCrmField("form.id", "请先删除流程下的阶段。")
	}
	return id
}

func (CrmHook) ProviderBeforeDeleteStage(c *server.Context, params []any) any {
	id := configDeleteID(params)
	if id == 0 {
		panicCrmField("form.id", "阶段不存在。")
	}
	ctx := contextFromServer(c)
	if crmmodel.NewCustomerStageModel().Count(ctx, map[string]any{"stage_id": id, "status": crmmodel.ProgressStatusActive}) > 0 ||
		crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{"stage_id": id, "status": crmmodel.WorkTodoStatusPending}) > 0 {
		panicCrmField("form.id", "阶段正在使用中，不能删除；可以先停用。")
	}
	if crmmodel.NewTaskModel().Count(ctx, map[string]any{"stage_id": id}) > 0 {
		panicCrmField("form.id", "请先删除阶段下的任务。")
	}
	return id
}

func (CrmHook) ProviderBeforeDeleteTask(c *server.Context, params []any) any {
	id := configDeleteID(params)
	if id == 0 {
		panicCrmField("form.id", "任务不存在。")
	}
	if crmmodel.NewWorkTodoModel().Count(contextFromServer(c), map[string]any{
		"task_id": id,
		"status":  crmmodel.WorkTodoStatusPending,
	}) > 0 {
		panicCrmField("form.id", "任务存在未完成待办，不能删除；可以先停用。")
	}
	return id
}

func configDeleteID(params []any) uint64 {
	for _, param := range params {
		if row, ok := param.(map[string]any); ok {
			if id := util.ToUint64(row["id"]); id > 0 {
				return id
			}
			continue
		}
		if id := util.ToUint64(param); id > 0 {
			return id
		}
	}
	return 0
}
