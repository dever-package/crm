package api

import (
	"github.com/shemic/dever/server"

	crmapi "github.com/dever-package/crm/api"
	crmservice "github.com/dever-package/crm/service"
)

type Data struct{}

var dataRecordService = crmservice.NewDataRecordService()

func (Data) GetSection(c *server.Context) error {
	customerID := crmapi.Uint64FromInput(c.Input("customer_id"))
	if customerID == 0 {
		customerID = crmapi.Uint64FromInput(c.Input("id"))
	}
	assetID := crmapi.Uint64FromInput(c.Input("asset_id"))
	workflowInstanceID := crmapi.Uint64FromInput(c.Input("workflow_instance_id"))
	data, err := dataRecordService.Section(c.Context(), customerID, assetID, workflowInstanceID)
	return crmapi.WriteJSON(c, data, err)
}

func (Data) PostSave(c *server.Context) error {
	body, err := crmapi.BindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := dataRecordService.Save(c.Context(), body)
	return crmapi.WriteJSON(c, data, err)
}
