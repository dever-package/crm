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
	return record
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
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "staff_type", partial)
	trimCrmStringField(record, "phone", partial)
	trimCrmStringField(record, "feishu_open_id", partial)
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
	validateStaffUniqueField(ctx, "feishu_open_id", "form.feishu_open_id", "该飞书 OpenID 已绑定其他人员。", record)
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
	record, ok := payload["payload"].(map[string]any)
	if !ok || util.ToStringTrimmed(record["staff_type"]) != crmmodel.StaffTypeLeader {
		return nil
	}
	staffID := savedRecordID(payload)
	departmentID := util.ToUint64(record["department_id"])
	if staffID == 0 || departmentID == 0 {
		return nil
	}
	rows := crmmodel.NewStaffModel().Select(c.Context(), map[string]any{
		"department_id": departmentID,
		"staff_type":    crmmodel.StaffTypeLeader,
	})
	for _, row := range rows {
		if row == nil || row.ID == staffID {
			continue
		}
		crmmodel.NewStaffModel().Update(c.Context(), map[string]any{"id": row.ID}, map[string]any{"staff_type": crmmodel.StaffTypeEmployee})
	}
	return nil
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
