package setting

import (
	"context"
	"strings"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBuildLeadRows(c *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	for _, row := range rows {
		row["status_name"] = crmmodel.LeadStatusName(strings.TrimSpace(util.ToString(row["status"])))
		row["source_name"] = relationName(row, "source.name")
		row["channel_name"] = relationName(row, "channel.name")
		row["invalid_reason_name"] = relationName(row, "invalid_reason.name")
		row["owner_staff_name"] = relationName(row, "owner_staff.name")
		row["owner_department_name"] = relationName(row, "owner_department.name")
		row["customer_name"] = relationName(row, "customer.name")
		if row["customer_name"] == "" {
			customerID := util.ToUint64(row["customer_id"])
			if customerID > 0 {
				if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}); customer != nil {
					row["customer_name"] = customer.Name
				}
			}
		}
	}
	return rows
}
