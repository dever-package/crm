package setting

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

var ensureDefaultStageOnce sync.Once

func (CrmHook) ProviderBuildTaskRows(c *server.Context, params []any) any {
	ctx := contextFromServer(c)
	ensureDefaultStage(ctx)
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}
	names := newTaskDisplayNameResolver(ctx)
	for _, row := range rows {
		config := decodeTaskConfig(row["config_json"])
		row["task_type_name"] = taskTypeName(row["task_type"])
		row["trigger_type_name"] = taskTriggerTypeName(row["trigger_type"])
		row["task_points"] = normalizeTaskPoints(config["task_points"])
		row["config_summary"] = taskConfigSummary(row, config, names)
		row["next_stage_name"] = taskNextStageName(config, names)
	}
	return rows
}

type taskDisplayNameResolver struct {
	ctx           context.Context
	forms         map[uint64]string
	departments   map[uint64]string
	staff         map[uint64]string
	stages        map[string]string
	resourceCates map[uint64]string
	scripts       map[uint64]string
	dataFields    map[uint64]string
}

func newTaskDisplayNameResolver(ctx context.Context) *taskDisplayNameResolver {
	return &taskDisplayNameResolver{
		ctx:           ctx,
		forms:         map[uint64]string{},
		departments:   map[uint64]string{},
		staff:         map[uint64]string{},
		stages:        map[string]string{},
		resourceCates: map[uint64]string{},
		scripts:       map[uint64]string{},
		dataFields:    map[uint64]string{},
	}
}

func taskConfigSummary(row map[string]any, config map[string]any, names *taskDisplayNameResolver) string {
	switch util.ToStringTrimmed(row["task_type"]) {
	case crmmodel.TaskTypeCreate:
		return taskFormSummary(util.ToUint64(row["form_id"]), "", names)
	case crmmodel.TaskTypeForm:
		return taskFormSummary(util.ToUint64(row["form_id"]), taskCompletionModeName(config["completion_mode"]), names)
	case crmmodel.TaskTypeAssign:
		return taskAssignSummary(row, config, names)
	case crmmodel.TaskTypeCollaborate:
		return taskCollaborationSummary(config)
	case crmmodel.TaskTypeDecision:
		return taskDecisionSummary(row, config, names)
	case crmmodel.TaskTypeBooking:
		return taskBookingSummary(config, names)
	default:
		return "未配置"
	}
}

func taskFormSummary(formID uint64, extra string, names *taskDisplayNameResolver) string {
	formName := names.formName(formID)
	if formName == "" {
		formName = "未配置"
	}
	parts := []string{"资料模板：" + formName}
	if extra != "" {
		parts = append(parts, extra)
	}
	return strings.Join(parts, "；")
}

func taskAssignSummary(row map[string]any, config map[string]any, names *taskDisplayNameResolver) string {
	parts := []string{taskAssignModeName(config["assign_mode"])}
	if departmentNames := names.departmentNames(normalizeTaskAssignDepartmentIDs(config["assign_department_ids"])); departmentNames != "" {
		parts = append(parts, "可选部门："+departmentNames)
	} else {
		parts = append(parts, "未配置可选部门")
	}
	if taskIsAutoTriggered(row) {
		if autoTarget := taskAutoAssignTargetName(config, names); autoTarget != "" {
			parts = append(parts, "自动分配："+autoTarget)
		} else {
			parts = append(parts, "未配置自动分配目标")
		}
	}
	return strings.Join(parts, "；")
}

func taskCollaborationSummary(config map[string]any) string {
	items := taskConfigRows(config["collaboration_items"])
	if len(items) == 0 {
		return "未配置子任务"
	}
	requiredCount := 0
	for _, item := range items {
		if taskConfigBoolDefault(item["required"], true) {
			requiredCount++
		}
	}
	return fmt.Sprintf("%d个子任务（%d个必做）；%s", len(items), requiredCount, taskCollaborationCompleteModeName(config["collaboration_complete_mode"]))
}

func taskDecisionSummary(row map[string]any, config map[string]any, names *taskDisplayNameResolver) string {
	parts := []string{}
	if fieldName := names.dataFieldName(util.ToUint64(config["decision_result_field_id"])); fieldName != "" {
		parts = append(parts, "结果写入："+fieldName)
	} else {
		parts = append(parts, "未配置结果字段")
	}
	if taskIsAutoTriggered(row) {
		if scriptName := names.scriptName(util.ToUint64(row["script_id"])); scriptName != "" {
			parts = append(parts, "自动脚本："+scriptName)
		} else {
			parts = append(parts, "未配置自动脚本")
		}
	}
	return strings.Join(parts, "；")
}

func taskBookingSummary(config map[string]any, names *taskDisplayNameResolver) string {
	resourceCateName := names.resourceCateName(util.ToUint64(config["resource_cate_id"]))
	if resourceCateName == "" {
		resourceCateName = "未配置"
	}
	return strings.Join([]string{
		"资源分类：" + resourceCateName,
		taskBookingConfirmName(config["need_confirm"]),
	}, "；")
}

func taskNextStageName(config map[string]any, names *taskDisplayNameResolver) string {
	stageCode := util.ToStringTrimmed(config["next_stage_code"])
	if stageCode == "" {
		return "不自动流转"
	}
	return names.stageName(stageCode)
}

func taskIsAutoTriggered(row map[string]any) bool {
	triggerType := util.ToStringTrimmed(row["trigger_type"])
	return triggerType != "" && triggerType != crmmodel.TaskTriggerManual
}

func taskTypeName(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.TaskTypeCreate:
		return "创建资料"
	case crmmodel.TaskTypeForm:
		return "填写资料"
	case crmmodel.TaskTypeAssign:
		return "分配"
	case crmmodel.TaskTypeCollaborate:
		return "协作任务"
	case crmmodel.TaskTypeDecision:
		return "决策"
	case crmmodel.TaskTypeBooking:
		return "资源预定"
	default:
		return util.ToStringTrimmed(value)
	}
}

func operationResultName(value any) string {
	switch util.ToStringTrimmed(value) {
	case "":
		return ""
	case "success":
		return "成功"
	case "progress":
		return "保存进度"
	case "auto_failed":
		return "自动执行失败"
	case crmmodel.ResourceBookingStatusPending:
		return "待确认"
	case crmmodel.ResourceBookingStatusReserved:
		return "已预定"
	case crmmodel.ResourceBookingStatusCanceled:
		return "已取消"
	case crmmodel.ResourceBookingStatusRejected:
		return "已拒绝"
	case crmmodel.ResourceBookingStatusDone:
		return "已完成"
	default:
		return util.ToStringTrimmed(value)
	}
}

func taskTriggerTypeName(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.TaskTriggerAfterTask:
		return "任务后触发"
	case crmmodel.TaskTriggerStageEnter:
		return "进入阶段触发"
	default:
		return "手动触发"
	}
}

func taskAssignModeName(value any) string {
	if util.ToStringTrimmed(value) == crmmodel.TaskAssignModeDepartment {
		return "按部门派单"
	}
	return "按人员派单"
}

func taskCompletionModeName(value any) string {
	if normalizeTaskCompletionMode(value) == crmmodel.TaskCompletionManual {
		return "手动完成"
	}
	return "提交即完成"
}

func taskCollaborationCompleteModeName(value any) string {
	switch normalizeTaskCollaborationCompleteMode(value) {
	case crmmodel.CollaborationCompleteAny:
		return "任一子任务完成后流转"
	case crmmodel.CollaborationCompleteManual:
		return "主任务确认后流转"
	default:
		return "全部必做完成后流转"
	}
}

func taskBookingConfirmName(value any) string {
	if util.ToBool(value) {
		return "需要确认"
	}
	return "无需确认"
}

func taskAutoAssignTargetName(config map[string]any, names *taskDisplayNameResolver) string {
	departmentName := names.departmentName(util.ToUint64(config["auto_assign_department_id"]))
	if departmentName == "" {
		return ""
	}
	staffID := util.ToUint64(config["auto_assign_staff_id"])
	if staffID == 0 {
		return departmentName + " / 部门负责人"
	}
	staffName := names.staffName(staffID)
	if staffName == "" {
		return departmentName
	}
	return departmentName + " / " + staffName
}

func (r *taskDisplayNameResolver) formName(id uint64) string {
	return r.cachedUint64Name(r.forms, id, func() string {
		if form := crmmodel.NewFormModel().Find(r.ctx, map[string]any{"id": id}); form != nil {
			return strings.TrimSpace(form.Name)
		}
		return ""
	})
}

func (r *taskDisplayNameResolver) departmentName(id uint64) string {
	return r.cachedUint64Name(r.departments, id, func() string {
		if department := crmmodel.NewDepartmentModel().Find(r.ctx, map[string]any{"id": id}); department != nil {
			return strings.TrimSpace(department.Name)
		}
		return ""
	})
}

func (r *taskDisplayNameResolver) departmentNames(ids []uint64) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name := r.departmentName(id); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, "、")
}

func (r *taskDisplayNameResolver) staffName(id uint64) string {
	return r.cachedUint64Name(r.staff, id, func() string {
		if staff := crmmodel.NewStaffModel().Find(r.ctx, map[string]any{"id": id}); staff != nil {
			return strings.TrimSpace(staff.Name)
		}
		return ""
	})
}

func (r *taskDisplayNameResolver) resourceCateName(id uint64) string {
	return r.cachedUint64Name(r.resourceCates, id, func() string {
		if cate := crmmodel.NewPublicResourceCateModel().Find(r.ctx, map[string]any{"id": id}); cate != nil {
			return strings.TrimSpace(cate.Name)
		}
		return ""
	})
}

func (r *taskDisplayNameResolver) scriptName(id uint64) string {
	return r.cachedUint64Name(r.scripts, id, func() string {
		if script := crmmodel.NewRuleScriptModel().Find(r.ctx, map[string]any{"id": id}); script != nil {
			return strings.TrimSpace(script.Name)
		}
		return ""
	})
}

func (r *taskDisplayNameResolver) dataFieldName(id uint64) string {
	return r.cachedUint64Name(r.dataFields, id, func() string {
		if field := crmmodel.NewDataFieldModel().Find(r.ctx, map[string]any{"id": id}); field != nil {
			return strings.TrimSpace(field.Name)
		}
		return ""
	})
}

func (r *taskDisplayNameResolver) stageName(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	if name, ok := r.stages[code]; ok {
		return name
	}
	name := code
	if stage := crmmodel.NewStageModel().Find(r.ctx, map[string]any{"code": code}); stage != nil {
		if stageName := strings.TrimSpace(stage.Name); stageName != "" {
			name = stageName
		}
	}
	r.stages[code] = name
	return name
}

func (r *taskDisplayNameResolver) cachedUint64Name(cache map[uint64]string, id uint64, load func() string) string {
	if id == 0 {
		return ""
	}
	if name, ok := cache[id]; ok {
		return name
	}
	name := strings.TrimSpace(load())
	if name == "" {
		name = fmt.Sprintf("#%d", id)
	}
	cache[id] = name
	return name
}

func ensureDefaultStage(ctx context.Context) {
	ensureDefaultStageOnce.Do(func() {
		stageModel := crmmodel.NewStageModel()
		if stageModel.Find(ctx, map[string]any{"id": crmmodel.DefaultStageID}) != nil {
			return
		}
		if stageModel.Find(ctx, map[string]any{"code": crmmodel.DefaultStageCode}) != nil {
			return
		}
		stageModel.Insert(ctx, crmmodel.DefaultStageRecord())
	})
}

// Keep this provider while generated service registry still references it.
func (CrmHook) ProviderAfterSaveTask(_ *server.Context, _ []any) any {
	return nil
}

func (CrmHook) ProviderBuildTaskForm(c *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	applyTaskConfigForm(contextFromServer(c), record)
	return record
}

func applyTaskConfigForm(ctx context.Context, record map[string]any) {
	if util.ToStringTrimmed(record["task_type"]) == "" {
		record["task_type"] = crmmodel.TaskTypeForm
	}
	config := decodeTaskConfig(record["config_json"])
	record["next_stage_code"] = util.ToStringTrimmed(config["next_stage_code"])
	record["task_points"] = normalizeTaskPoints(config["task_points"])
	assignMode := util.ToStringTrimmed(config["assign_mode"])
	if assignMode == "" {
		assignMode = crmmodel.TaskAssignModeStaff
	}
	if assignMode != crmmodel.TaskAssignModeDepartment {
		assignMode = crmmodel.TaskAssignModeStaff
	}
	record["assign_mode"] = assignMode
	record["assign_department_ids"] = normalizeTaskAssignDepartmentIDs(config["assign_department_ids"])
	record["auto_assign_department_id"] = optionalTaskConfigID(config["auto_assign_department_id"])
	record["auto_assign_staff_id"] = optionalTaskConfigID(config["auto_assign_staff_id"])
	record["collaboration_items"] = taskCollaborationItemsForForm(normalizeTaskCollaborationItems(nil, config["collaboration_items"], true))
	record["collaboration_complete_mode"] = normalizeTaskCollaborationCompleteMode(config["collaboration_complete_mode"])
	record["completion_mode"] = normalizeTaskCompletionMode(config["completion_mode"])
	record["complete_assign_task_id"] = optionalTaskConfigID(config[crmmodel.TaskCompleteAssignTaskID])
	record["hide_from_work_list"] = util.ToBool(config["hide_from_work_list"])
	resourceCateID := util.ToUint64(config["resource_cate_id"])
	if resourceCateID == 0 {
		resourceCateID = crmmodel.DefaultResourceCateID
	}
	record["booking_resource_cate_id"] = resourceCateID
	record["booking_need_confirm"] = util.ToBool(config["need_confirm"])
	applyDecisionResultFieldForm(ctx, record, config)
	applyTaskVisibleWhenForm(ctx, record, config)
}
