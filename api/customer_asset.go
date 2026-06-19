package api

import (
	"github.com/shemic/dever/server"

	crmservice "github.com/dever-package/crm/service"
)

type CustomerAsset struct{}

var customerAssetService = crmservice.NewCustomerAssetService()

func (CustomerAsset) PostCreate(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := customerAssetService.Create(c.Context(), body)
	return crmJSON(c, data, err)
}

func (CustomerAsset) GetDetail(c *server.Context) error {
	assetID := uint64FromInput(c.Input("asset_id"))
	if assetID == 0 {
		assetID = uint64FromInput(c.Input("id"))
	}
	data, err := customerAssetService.Detail(c.Context(), assetID)
	return crmJSON(c, data, err)
}
