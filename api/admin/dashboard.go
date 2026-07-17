package api

import (
	"strconv"
	"strings"

	"github.com/shemic/dever/server"

	crmapi "github.com/dever-package/crm/api"
	crmservice "github.com/dever-package/crm/service"
)

type Dashboard struct{}

var adminSummaryService = crmservice.NewAdminSummaryService()

func (Dashboard) GetSummary(c *server.Context) error {
	data, err := adminSummaryService.Summary(c.Context(), crmservice.AdminSummaryQuery{
		Mode:         c.Query("mode"),
		WorkflowID:   adminSummaryUint64(c.Query("workflow_id")),
		DepartmentID: adminSummaryUint64(c.Query("department_id")),
		StaffID:      adminSummaryUint64(c.Query("staff_id")),
		DateFrom:     c.Query("date_from"),
		DateTo:       c.Query("date_to"),
	})
	return crmapi.WriteJSON(c, data, err)
}

func adminSummaryUint64(value string) uint64 {
	result, _ := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	return result
}
