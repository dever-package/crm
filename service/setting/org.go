package setting

import (
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
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

func (CrmHook) ProviderBeforeSaveStaff(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
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
		if util.ToStringTrimmed(record["phone"]) == "" {
			panicCrmField("form.phone", "手机号不能为空。")
		}
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	return record
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
