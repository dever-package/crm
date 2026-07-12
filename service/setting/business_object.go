package setting

import (
	"context"
	"strings"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSaveBusinessObjectType(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "code", partial)
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "parent_target", partial)
	trimCrmStringField(record, "description", partial)
	if !partial {
		if util.ToStringTrimmed(record["code"]) == "" {
			panicCrmField("form.code", "类型编码不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "类型名称不能为空。")
		}
	}
	parentTarget := util.ToStringTrimmed(record["parent_target"])
	if parentTarget == "" {
		record["parent_target"] = crmmodel.BusinessObjectParentCustomerAsset
	} else if !validBusinessObjectParentTarget(parentTarget) {
		panicCrmField("form.parent_target", "归属对象不支持。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBeforeSaveBusinessObject(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "object_no", partial)
	trimCrmStringField(record, "object_name", partial)
	trimCrmStringField(record, "object_status", partial)
	trimCrmStringField(record, "record_json", partial)
	trimCrmStringField(record, "remark", partial)
	defaultCrmInt(record, "business_object_type_id", 0, partial)
	defaultCrmInt(record, "customer_id", 0, partial)
	defaultCrmInt(record, "asset_id", 0, partial)
	defaultCrmInt(record, "parent_object_id", 0, partial)
	defaultCrmInt(record, "owner_department_id", 0, partial)
	defaultCrmInt(record, "owner_staff_id", 0, partial)
	if !partial {
		if util.ToUint64(record["business_object_type_id"]) == 0 {
			panicCrmField("form.business_object_type_id", "业务对象类型不能为空。")
		}
		if util.ToUint64(record["customer_id"]) == 0 {
			panicCrmField("form.customer_id", "所属客户不能为空。")
		}
		if util.ToStringTrimmed(record["object_name"]) == "" {
			panicCrmField("form.object_name", "对象名称不能为空。")
		}
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	validateBusinessObjectReferences(ctx, record, partial)
	if util.ToStringTrimmed(record["record_json"]) == "" {
		record["record_json"] = "{}"
	}
	record["updated_at"] = time.Now()
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBuildBusinessObjectRows(_ *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}

	for _, row := range rows {
		row["object_status_name"] = crmmodel.BusinessObjectStatusName(util.ToString(row["object_status"]))
	}
	return rows
}

func validBusinessObjectParentTarget(parentTarget string) bool {
	switch strings.TrimSpace(parentTarget) {
	case crmmodel.BusinessObjectParentCustomer,
		crmmodel.BusinessObjectParentCustomerAsset,
		crmmodel.BusinessObjectParentBusinessObject:
		return true
	default:
		return false
	}
}

func validateBusinessObjectReferences(ctx context.Context, record map[string]any, partial bool) {
	if !shouldNormalizeCrmField(record, "business_object_type_id", partial) &&
		!shouldNormalizeCrmField(record, "customer_id", partial) &&
		!shouldNormalizeCrmField(record, "asset_id", partial) &&
		!shouldNormalizeCrmField(record, "parent_object_id", partial) &&
		!shouldNormalizeCrmField(record, "owner_department_id", partial) &&
		!shouldNormalizeCrmField(record, "owner_staff_id", partial) {
		return
	}
	typeID := util.ToUint64(record["business_object_type_id"])
	if typeID > 0 && crmmodel.NewBusinessObjectTypeModel().Find(ctx, map[string]any{"id": typeID, "status": crmmodel.StatusEnabled}) == nil {
		panicCrmField("form.business_object_type_id", "业务对象类型不存在或已停用。")
	}
	customerID := util.ToUint64(record["customer_id"])
	if customerID > 0 && crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		panicCrmField("form.customer_id", "所属客户不存在。")
	}
	assetID := util.ToUint64(record["asset_id"])
	if assetID > 0 {
		filter := map[string]any{"id": assetID}
		if customerID > 0 {
			filter["customer_id"] = customerID
		}
		if crmmodel.NewCustomerAssetModel().Find(ctx, filter) == nil {
			panicCrmField("form.asset_id", "所属资产不存在或不属于当前客户。")
		}
	}
	parentObjectID := util.ToUint64(record["parent_object_id"])
	if parentObjectID > 0 && crmmodel.NewBusinessObjectModel().Find(ctx, map[string]any{"id": parentObjectID}) == nil {
		panicCrmField("form.parent_object_id", "父级业务对象不存在。")
	}
	ownerDepartmentID := util.ToUint64(record["owner_department_id"])
	if ownerDepartmentID > 0 && crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": ownerDepartmentID, "status": crmmodel.StatusEnabled}) == nil {
		panicCrmField("form.owner_department_id", "负责部门不存在或已停用。")
	}
	ownerStaffID := util.ToUint64(record["owner_staff_id"])
	if ownerStaffID == 0 {
		return
	}
	ownerStaff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": ownerStaffID, "status": crmmodel.StatusEnabled})
	if ownerStaff == nil {
		panicCrmField("form.owner_staff_id", "负责人不存在或已停用。")
	}
	if ownerDepartmentID > 0 && ownerStaff.DepartmentID != ownerDepartmentID {
		panicCrmField("form.owner_staff_id", "负责人必须属于所选负责部门。")
	}
}
