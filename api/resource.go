package api

import (
	"github.com/shemic/dever/server"

	crmservice "my/package/crm/service"
)

type Resource struct{}

var resourceService = crmservice.NewResourceService()

func (Resource) PostCreate(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := resourceService.CreateWithTask(c.Context(), body)
	return crmJSON(c, data, err)
}

func (Resource) GetDetail(c *server.Context) error {
	resourceID := uint64FromInput(c.Input("resource_id"))
	if resourceID == 0 {
		resourceID = uint64FromInput(c.Input("id"))
	}
	data, err := resourceService.Detail(c.Context(), resourceID)
	return crmJSON(c, data, err)
}
