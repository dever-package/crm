package api

import (
	"github.com/shemic/dever/server"

	crmservice "my/package/crm/service"
)

type Dashboard struct{}

var metricService = crmservice.NewMetricService()

func (Dashboard) GetSummary(c *server.Context) error {
	data, err := metricService.Summary(c.Context())
	return crmJSON(c, data, err)
}

func (Dashboard) GetFunnel(c *server.Context) error {
	data, err := metricService.Funnel(c.Context())
	return crmJSON(c, data, err)
}

func (Dashboard) GetWidget(c *server.Context) error {
	data, err := metricService.Widget(c.Context(), c.Input("key"))
	return crmJSON(c, data, err)
}
