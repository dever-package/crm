package api

import (
	"github.com/shemic/dever/server"

	crmservice "my/package/crm/service"
)

type Data struct{}

var dataRecordService = crmservice.NewDataRecordService()

func (Data) GetSection(c *server.Context) error {
	resourceID := uint64FromInput(c.Input("resource_id"))
	if resourceID == 0 {
		resourceID = uint64FromInput(c.Input("id"))
	}
	data, err := dataRecordService.Section(c.Context(), resourceID)
	return crmJSON(c, data, err)
}

func (Data) PostSave(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := dataRecordService.Save(c.Context(), body)
	return crmJSON(c, data, err)
}
