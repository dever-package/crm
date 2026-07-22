package setting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSaveDepartment(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "code", partial)
	trimCrmStringField(record, "name", partial)
	if !partial {
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "部门名称不能为空。")
		}
	}
	if !partial || shouldNormalizeCrmField(record, "code", partial) {
		record["code"] = resolveDepartmentCode(c, record)
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	if c != nil {
		departmentID := util.ToUint64(record["id"])
		leaderStaffID := util.ToUint64(record["leader_staff_id"])
		if departmentID > 0 && leaderStaffID > 0 && crmmodel.NewStaffModel().Find(c.Context(), map[string]any{
			"id":            leaderStaffID,
			"department_id": departmentID,
			"status":        crmmodel.StatusEnabled,
		}) == nil {
			panicCrmField("form.leader_staff_id", "部门负责人必须是本部门启用人员。")
		}
	}
	return record
}

func (CrmHook) ProviderAfterSaveDepartment(c *server.Context, params []any) any {
	if c == nil || len(params) == 0 {
		return nil
	}
	payload, ok := params[0].(map[string]any)
	if !ok {
		return nil
	}
	departmentID := savedRecordID(payload)
	if departmentID == 0 {
		return nil
	}
	syncDepartmentLeaderStaffTypes(c.Context(), departmentID)
	return nil
}

func resolveDepartmentCode(c *server.Context, record map[string]any) string {
	code := util.ToStringTrimmed(record["code"])
	if code != "" {
		return code
	}
	departmentID := util.ToUint64(record["id"])
	if c != nil && departmentID > 0 {
		if current := crmmodel.NewDepartmentModel().Find(c.Context(), map[string]any{"id": departmentID}); current != nil && strings.TrimSpace(current.Code) != "" {
			return strings.TrimSpace(current.Code)
		}
	}
	return fmt.Sprintf("dept_%d", time.Now().UnixNano())
}

func (CrmHook) ProviderBeforeSaveStaff(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	// 飞书 OpenID 只允许登录流程自动绑定，后台人员表单不能写入或清空。
	delete(record, "feishu_open_id")
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "staff_type", partial)
	trimCrmStringField(record, "phone", partial)
	if shouldNormalizeCrmField(record, "department_id", partial) && util.ToUint64(record["department_id"]) == 0 {
		record["department_id"] = crmmodel.DefaultDepartmentID
	}
	if shouldNormalizeCrmField(record, "staff_type", partial) && util.ToStringTrimmed(record["staff_type"]) == "" {
		record["staff_type"] = crmmodel.StaffTypeEmployee
	}
	if !partial {
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "姓名不能为空。")
		}
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	validateStaffUniqueField(ctx, "phone", "form.phone", "该手机号已存在，请更换。", record)
	defaultCrmBool(record, "can_dispatch", false, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	return record
}

func validateStaffUniqueField(ctx context.Context, field string, formPath string, message string, record map[string]any) {
	value := util.ToStringTrimmed(record[field])
	if value == "" {
		return
	}
	filters := map[string]any{field: value}
	if staffID := util.ToUint64(record["id"]); staffID > 0 {
		filters["id"] = map[string]any{"!=": staffID}
	}
	if crmmodel.NewStaffModel().Count(ctx, filters) > 0 {
		panicCrmField(formPath, message)
	}
}

func (CrmHook) ProviderAfterSaveStaff(c *server.Context, params []any) any {
	if c == nil || len(params) == 0 {
		return nil
	}
	payload, ok := params[0].(map[string]any)
	if !ok {
		return nil
	}
	staffID := savedRecordID(payload)
	if staffID == 0 {
		return nil
	}
	staff := crmmodel.NewStaffModel().Find(c.Context(), map[string]any{"id": staffID})
	if staff == nil {
		return nil
	}
	for _, department := range crmmodel.NewDepartmentModel().Select(c.Context(), map[string]any{"leader_staff_id": staff.ID}) {
		if department == nil || department.ID == staff.DepartmentID && staff.Status == crmmodel.StatusEnabled {
			continue
		}
		crmmodel.NewDepartmentModel().Update(c.Context(), map[string]any{"id": department.ID}, map[string]any{"leader_staff_id": uint64(0)})
		syncDepartmentLeaderStaffTypes(c.Context(), department.ID)
	}
	record, _ := payload["payload"].(map[string]any)
	if staff.Status == crmmodel.StatusEnabled && util.ToStringTrimmed(record["staff_type"]) == crmmodel.StaffTypeLeader {
		crmmodel.NewDepartmentModel().Update(c.Context(), map[string]any{"id": staff.DepartmentID}, map[string]any{
			"leader_staff_id": staff.ID,
		})
	} else if util.ToStringTrimmed(record["staff_type"]) == crmmodel.StaffTypeEmployee {
		crmmodel.NewDepartmentModel().Update(c.Context(), map[string]any{
			"id":              staff.DepartmentID,
			"leader_staff_id": staff.ID,
		}, map[string]any{"leader_staff_id": uint64(0)})
	}
	syncDepartmentLeaderStaffTypes(c.Context(), staff.DepartmentID)
	return nil
}

func syncDepartmentLeaderStaffTypes(ctx context.Context, departmentID uint64) {
	if departmentID == 0 {
		return
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": departmentID})
	if department == nil {
		return
	}
	crmmodel.NewStaffModel().Update(ctx, map[string]any{
		"department_id": departmentID,
		"staff_type":    crmmodel.StaffTypeLeader,
		"id":            map[string]any{"!=": department.LeaderStaffID},
	}, map[string]any{"staff_type": crmmodel.StaffTypeEmployee})
	if department.LeaderStaffID > 0 {
		crmmodel.NewStaffModel().Update(ctx, map[string]any{
			"id":            department.LeaderStaffID,
			"department_id": departmentID,
			"status":        crmmodel.StatusEnabled,
		}, map[string]any{"staff_type": crmmodel.StaffTypeLeader})
	}
}

func savedRecordID(payload map[string]any) uint64 {
	if id := util.ToUint64(payload["id"]); id > 0 {
		return id
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		return 0
	}
	for _, key := range []string{"id", "main.id"} {
		if id := util.ToUint64(result[key]); id > 0 {
			return id
		}
	}
	return 0
}

func (CrmHook) ProviderBuildStaffRows(_ *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}
	for _, row := range rows {
		row["staff_type_name"] = staffTypeName(row["staff_type"])
		row["can_dispatch_name"] = "-"
		if configBool(row["can_dispatch"]) {
			row["can_dispatch_name"] = "可调度"
		}
	}
	return rows
}

func staffTypeName(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.StaffTypeLeader:
		return "负责人"
	default:
		return "员工"
	}
}
