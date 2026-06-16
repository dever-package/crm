package setting

import (
	"context"
	"sync"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

var ensureDefaultStageOnce sync.Once

func (CrmHook) ProviderBuildTaskRows(c *server.Context, params []any) any {
	ensureDefaultStage(contextFromServer(c))
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}
	for _, row := range rows {
		row["task_type_name"] = taskTypeName(row["task_type"])
		row["trigger_type_name"] = taskTriggerTypeName(row["trigger_type"])
		row["task_points"] = normalizeTaskPoints(decodeTaskConfig(row["config_json"])["task_points"])
	}
	return rows
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
	resourceCateID := util.ToUint64(config["resource_cate_id"])
	if resourceCateID == 0 {
		resourceCateID = crmmodel.DefaultResourceCateID
	}
	record["booking_resource_cate_id"] = resourceCateID
	record["booking_need_confirm"] = util.ToBool(config["need_confirm"])
	applyDecisionResultFieldForm(ctx, record, config)
	applyTaskVisibleWhenForm(ctx, record, config)
}
