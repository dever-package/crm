package setting

import (
	"context"

	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func normalizeTaskAssignDepartmentIDs(value any) []uint64 {
	items := compactStringList(value)
	result := make([]uint64, 0, len(items))
	seen := map[uint64]bool{}
	for _, item := range items {
		id := util.ToUint64(item)
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}

func effectiveTaskAssignMode(ctx context.Context, record map[string]any, partial bool) string {
	if shouldNormalizeCrmField(record, "assign_mode", partial) {
		return normalizeTaskAssignMode(record["assign_mode"])
	}
	return normalizeTaskAssignMode(currentTaskConfigValue(ctx, record, "assign_mode"))
}

func normalizeTaskAssignMode(value any) string {
	if util.ToStringTrimmed(value) == crmmodel.TaskAssignModeDepartment {
		return crmmodel.TaskAssignModeDepartment
	}
	return crmmodel.TaskAssignModeStaff
}

func effectiveTaskAssignDepartmentIDs(ctx context.Context, record map[string]any, partial bool) []uint64 {
	if shouldNormalizeCrmField(record, "assign_department_ids", partial) {
		return normalizeTaskAssignDepartmentIDs(record["assign_department_ids"])
	}
	return normalizeTaskAssignDepartmentIDs(currentTaskConfigValue(ctx, record, "assign_department_ids"))
}

func normalizeTaskAutoAssignTarget(ctx context.Context, record map[string]any, assignMode string, allowedDepartmentIDs []uint64, partial bool) (uint64, uint64) {
	if effectiveTaskTriggerType(ctx, record, partial) == crmmodel.TaskTriggerManual {
		return 0, 0
	}
	departmentID := util.ToUint64(record["auto_assign_department_id"])
	staffID := util.ToUint64(record["auto_assign_staff_id"])
	if assignMode == crmmodel.TaskAssignModeDepartment {
		staffID = 0
	}
	if partial && departmentID == 0 && !shouldNormalizeCrmField(record, "auto_assign_department_id", partial) {
		if current := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
			config := decodeTaskConfig(current.ConfigJSON)
			departmentID = util.ToUint64(config["auto_assign_department_id"])
			if staffID == 0 && !shouldNormalizeCrmField(record, "auto_assign_staff_id", partial) {
				staffID = util.ToUint64(config["auto_assign_staff_id"])
			}
		}
	}
	if departmentID == 0 {
		panicCrmField("form.auto_assign_department_id", "自动触发的分配任务必须选择自动分配部门。")
	}
	if !containsUint64(allowedDepartmentIDs, departmentID) {
		panicCrmField("form.auto_assign_department_id", "自动分配部门必须在可选部门范围内。")
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": departmentID, "status": crmmodel.StatusEnabled})
	if department == nil {
		panicCrmField("form.auto_assign_department_id", "自动分配部门不存在或已停用。")
	}
	if staffID > 0 {
		staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID, "status": crmmodel.StatusEnabled})
		if staff == nil {
			panicCrmField("form.auto_assign_staff_id", "自动分配人员不存在或已停用。")
		}
		if staff.DepartmentID != departmentID {
			panicCrmField("form.auto_assign_staff_id", "自动分配人员不属于自动分配部门。")
		}
		return departmentID, staffID
	}
	if taskDepartmentLeaderStaffID(ctx, department) == 0 {
		panicCrmField("form.auto_assign_staff_id", "未选择自动分配人员时，自动分配部门必须配置负责人。")
	}
	return departmentID, 0
}

func taskDepartmentLeaderStaffID(ctx context.Context, department *crmmodel.Department) uint64 {
	if department == nil || department.ID == 0 {
		return 0
	}
	if department.LeaderStaffID > 0 {
		if staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
			"id":            department.LeaderStaffID,
			"department_id": department.ID,
			"status":        crmmodel.StatusEnabled,
		}); staff != nil {
			return staff.ID
		}
	}
	if staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"department_id": department.ID,
		"staff_type":    crmmodel.StaffTypeLeader,
		"status":        crmmodel.StatusEnabled,
	}); staff != nil {
		return staff.ID
	}
	return 0
}

func containsUint64(values []uint64, target uint64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
