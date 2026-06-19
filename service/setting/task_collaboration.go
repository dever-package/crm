package setting

import (
	"context"
	"strings"

	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func taskCollaborationItemsForForm(items []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row := util.CloneMap(item)
		row["staff_id"] = optionalTaskConfigID(row["staff_id"])
		row["form_id"] = optionalTaskConfigID(row["form_id"])
		row["task_points"] = normalizeTaskPoints(row["task_points"])
		result = append(result, row)
	}
	return result
}

func normalizeTaskCollaborationCompleteMode(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.CollaborationCompleteAny:
		return crmmodel.CollaborationCompleteAny
	case crmmodel.CollaborationCompleteManual:
		return crmmodel.CollaborationCompleteManual
	default:
		return crmmodel.CollaborationCompleteAll
	}
}

func effectiveTaskCollaborationItems(ctx context.Context, record map[string]any, partial bool) []map[string]any {
	if shouldNormalizeCrmField(record, "collaboration_items", partial) {
		return normalizeTaskCollaborationItems(ctx, record["collaboration_items"], partial)
	}
	return normalizeTaskCollaborationItems(ctx, currentTaskConfigValue(ctx, record, "collaboration_items"), true)
}

func effectiveTaskCollaborationCompleteMode(ctx context.Context, record map[string]any, partial bool) string {
	if shouldNormalizeCrmField(record, "collaboration_complete_mode", partial) {
		return normalizeTaskCollaborationCompleteMode(record["collaboration_complete_mode"])
	}
	return normalizeTaskCollaborationCompleteMode(currentTaskConfigValue(ctx, record, "collaboration_complete_mode"))
}

func ensureAutoTaskCollaborationStaff(items []map[string]any, triggerType string) {
	if triggerType == "" || triggerType == crmmodel.TaskTriggerManual {
		return
	}
	for _, item := range items {
		if util.ToUint64(item["staff_id"]) == 0 {
			panicCrmField("form.collaboration_items", "自动触发的协作任务必须为每个子任务指定处理人员。")
		}
	}
}

func normalizeTaskCollaborationItems(ctx context.Context, value any, partial bool) []map[string]any {
	rows := taskConfigRows(value)
	result := make([]map[string]any, 0, len(rows))
	seen := map[string]bool{}
	for index, row := range rows {
		name := util.ToStringTrimmed(firstTaskConfigValue(row, "name", "task_name", "sub_task_name"))
		departmentID := util.ToUint64(firstTaskConfigValue(row, "department_id", "assignee_department_id"))
		staffID := util.ToUint64(firstTaskConfigValue(row, "staff_id", "assignee_staff_id"))
		formID := util.ToUint64(firstTaskConfigValue(row, "form_id"))
		taskPoints := normalizeTaskPoints(firstTaskConfigValue(row, "task_points", "points"))
		if name == "" && departmentID == 0 && staffID == 0 && formID == 0 && taskPoints == 0 {
			continue
		}
		if name == "" {
			panicCrmField("form.collaboration_items", "协作子任务名称不能为空。")
		}
		if departmentID == 0 {
			panicCrmField("form.collaboration_items", "协作子任务目标部门不能为空。")
		}
		if ctx != nil && crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": departmentID, "status": crmmodel.StatusEnabled}) == nil {
			panicCrmField("form.collaboration_items", "协作子任务目标部门不存在或已停用。")
		}
		if staffID > 0 && ctx != nil {
			staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID, "status": crmmodel.StatusEnabled})
			if staff == nil {
				panicCrmField("form.collaboration_items", "协作子任务处理人员不存在或已停用。")
			}
			if staff != nil && staff.DepartmentID != departmentID {
				panicCrmField("form.collaboration_items", "协作子任务处理人员不属于目标部门。")
			}
		}
		if formID > 0 && ctx != nil && crmmodel.NewFormModel().Find(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled}) == nil {
			panicCrmField("form.collaboration_items", "协作子任务资料模板不存在或已停用。")
		}
		sortValue := taskConfigSort(row["sort"], index)
		itemKey := taskCollaborationItemKey(name, departmentID, formID, sortValue)
		seenKey := itemKey + ":" + util.ToString(staffID)
		if seen[seenKey] {
			continue
		}
		seen[seenKey] = true
		result = append(result, map[string]any{
			"key":             itemKey,
			"name":            name,
			"department_id":   departmentID,
			"staff_id":        staffID,
			"form_id":         formID,
			"completion_mode": normalizeTaskCompletionMode(row["completion_mode"]),
			"task_points":     taskPoints,
			"required":        taskConfigBoolDefault(row["required"], true),
			"sort":            sortValue,
		})
	}
	return result
}

func taskCollaborationItemKey(name string, departmentID uint64, formID uint64, sort int) string {
	return strings.Join([]string{
		util.ToString(sort),
		util.ToString(departmentID),
		util.ToString(formID),
		name,
	}, ":")
}
