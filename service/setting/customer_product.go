package setting

import (
	"context"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSaveCustomerProduct(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	id := util.ToUint64(record["id"])
	if id == 0 {
		panicCrmField("form.id", "客户产品不存在。")
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	current := crmmodel.NewCustomerProductModel().Find(ctx, map[string]any{"id": id})
	if current == nil {
		panicCrmField("form.id", "客户产品不存在。")
	}
	targetStatus := util.ToStringTrimmed(record["status"])
	if targetStatus == "" || targetStatus == current.Status {
		record["status"] = current.Status
		return record
	}
	if current.Status == crmmodel.CustomerProductStatusProcessing || current.Status == crmmodel.CustomerProductStatusCompleted {
		panicCrmField("form.status", "处理中或已完成的产品不能直接修改状态。")
	}
	if targetStatus != crmmodel.CustomerProductStatusConfirmed && targetStatus != crmmodel.CustomerProductStatusLost {
		panicCrmField("form.status", "后台只能设置为已确认或已流失。")
	}
	if crmmodel.NewWorkflowInstanceModel().Count(ctx, map[string]any{
		"customer_product_id": current.ID,
		"status":              crmmodel.ProgressStatusActive,
	}) > 0 {
		panicCrmField("form.status", "该产品存在进行中的流程，不能直接修改状态。")
	}
	record = map[string]any{
		"id":         current.ID,
		"status":     targetStatus,
		"updated_at": time.Now(),
	}
	return record
}

func (CrmHook) ProviderBuildCustomerProductRows(c *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	for _, row := range rows {
		customerProductID := util.ToUint64(row["id"])
		row["status_name"] = crmmodel.CustomerProductStatusName(util.ToStringTrimmed(row["status"]))
		instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
			"customer_product_id": customerProductID,
		}, map[string]any{"order": "id desc"})
		if instance == nil {
			continue
		}
		row["workflow_instance_id"] = instance.ID
		row["workflow_status"] = instance.Status
		if workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": instance.WorkflowID}); workflow != nil {
			row["workflow_name"] = workflow.Name
		}
		if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": instance.StageID}); stage != nil {
			row["stage_name"] = stage.Name
		}
		if owner := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": instance.OwnerStaffID}); owner != nil {
			row["owner_staff_name"] = owner.Name
		}
	}
	return rows
}
