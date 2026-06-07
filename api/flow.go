package api

import (
	"github.com/shemic/dever/server"

	crmservice "my/package/crm/service"
)

type Flow struct{}

var flowDesigner = crmservice.NewFlowDesignerService()
var flowRelease = crmservice.NewFlowReleaseService()

func (Flow) GetWorkspace(c *server.Context) error {
	flowTemplateID := uint64FromInput(c.Input("flow_template_id"))
	if flowTemplateID == 0 {
		flowTemplateID = uint64FromInput(c.Input("id"))
	}
	data, err := flowDesigner.Workspace(c.Context(), flowTemplateID)
	return crmJSON(c, data, err)
}

func (Flow) PostSave(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := flowDesigner.Save(c.Context(), body)
	return crmJSON(c, data, err)
}

func (Flow) PostPublish(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	flowTemplateID := crmservice.ResolveFlowTemplateID(c.Context(), uint64FromBody(body, "flow_template_id", "flowTemplateId", "id"))
	data, err := flowRelease.Publish(c.Context(), flowTemplateID)
	return crmJSON(c, data, err)
}

func (Flow) PostClone(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := flowDesigner.Clone(c.Context(), uint64FromBody(body, "flow_template_id", "flowTemplateId", "id"))
	return crmJSON(c, data, err)
}
