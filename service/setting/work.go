package setting

import (
	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmservice "my/package/crm/service"
)

func (CrmHook) ProviderLoadWorkCustomers(c *server.Context, _ []any) any {
	staff := crmservice.CurrentWorkStaff(c.Context())
	if staff == nil || staff.ID == 0 {
		return workTablePayload(nil, 1, 20)
	}

	data, err := crmservice.NewWorkService().Customers(c.Context(), staff, map[string]any{
		"keyword": c.Input("keyword"),
	})
	if err != nil {
		return workTablePayload(nil, 1, 20)
	}

	rows, _ := data["list"].([]map[string]any)
	return workTablePayload(
		rows,
		util.ToIntDefault(c.Input("page"), 1),
		util.ToIntDefault(c.Input("pageSize"), 20),
	)
}

func (CrmHook) ProviderLoadWorkOperations(c *server.Context, _ []any) any {
	staff := crmservice.CurrentWorkStaff(c.Context())
	if staff == nil || staff.ID == 0 {
		return workTablePayload(nil, 1, 20)
	}

	data, err := crmservice.NewWorkService().Operations(c.Context(), staff, map[string]any{
		"keyword": c.Input("keyword"),
	})
	if err != nil {
		return workTablePayload(nil, 1, 20)
	}

	rows, _ := data["list"].([]map[string]any)
	return workTablePayload(
		rows,
		util.ToIntDefault(c.Input("page"), 1),
		util.ToIntDefault(c.Input("pageSize"), 20),
	)
}

func (CrmHook) ProviderLoadWorkBookings(c *server.Context, _ []any) any {
	staff := crmservice.CurrentWorkStaff(c.Context())
	if staff == nil || staff.ID == 0 {
		return workTablePayload(nil, 1, 20)
	}

	data, err := crmservice.NewWorkService().Bookings(c.Context(), staff, map[string]any{
		"keyword": c.Input("keyword"),
	})
	if err != nil {
		return workTablePayload(nil, 1, 20)
	}

	rows, _ := data["list"].([]map[string]any)
	return workTablePayload(
		rows,
		util.ToIntDefault(c.Input("page"), 1),
		util.ToIntDefault(c.Input("pageSize"), 20),
	)
}

func workTablePayload(rows []map[string]any, page int, pageSize int) map[string]any {
	if rows == nil {
		rows = []map[string]any{}
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	total := len(rows)
	start := (page - 1) * pageSize
	switch {
	case start >= total:
		rows = []map[string]any{}
	case start > 0 || total > pageSize:
		end := start + pageSize
		if end > total {
			end = total
		}
		rows = rows[start:end]
	}
	return map[string]any{
		"list":     rows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	}
}
