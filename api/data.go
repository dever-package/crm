package api

import (
	"github.com/shemic/dever/server"

	crmservice "github.com/dever-package/crm/service"
)

type Data struct{}

var dataRecordService = crmservice.NewDataRecordService()

func (Data) GetSection(c *server.Context) error {
	customerID := uint64FromInput(c.Input("customer_id"))
	if customerID == 0 {
		customerID = uint64FromInput(c.Input("id"))
	}
	assetID := uint64FromInput(c.Input("asset_id"))
	data, err := dataRecordService.Section(c.Context(), customerID, assetID)
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
