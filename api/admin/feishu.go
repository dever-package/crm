package api

import (
	"github.com/shemic/dever/server"

	crmapi "github.com/dever-package/crm/api"
	crmservice "github.com/dever-package/crm/service"
)

type Feishu struct{}

var feishuService = crmservice.NewFeishuService()

func (Feishu) PostTest(c *server.Context) error {
	body, err := crmapi.BindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := feishuService.SendTestMessage(
		c.Context(),
		crmapi.Uint64FromBody(body, "staff_id", "staffId"),
	)
	return crmapi.WriteJSON(c, data, err)
}
