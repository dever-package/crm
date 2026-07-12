package api

import (
	"github.com/shemic/dever/server"

	crmapi "github.com/dever-package/crm/api"
	crmservice "github.com/dever-package/crm/service"
)

type Dashboard struct{}

var adminSummaryService = crmservice.NewAdminSummaryService()

func (Dashboard) GetSummary(c *server.Context) error {
	data, err := adminSummaryService.Summary(c.Context())
	return crmapi.WriteJSON(c, data, err)
}
