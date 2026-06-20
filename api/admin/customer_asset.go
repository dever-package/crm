package api

import (
	"github.com/shemic/dever/server"

	crmapi "github.com/dever-package/crm/api"
	crmservice "github.com/dever-package/crm/service"
)

type CustomerAsset struct{}

var customerAssetService = crmservice.NewCustomerAssetService()

func (CustomerAsset) PostCreate(c *server.Context) error {
	body, err := crmapi.BindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := customerAssetService.Create(c.Context(), body)
	return crmapi.WriteJSON(c, data, err)
}

func (CustomerAsset) GetDetail(c *server.Context) error {
	assetID := crmapi.Uint64FromInput(c.Input("asset_id"))
	if assetID == 0 {
		assetID = crmapi.Uint64FromInput(c.Input("id"))
	}
	data, err := customerAssetService.Detail(c.Context(), assetID)
	return crmapi.WriteJSON(c, data, err)
}
