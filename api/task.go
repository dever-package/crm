package api

import (
	"github.com/shemic/dever/server"

	crmservice "my/package/crm/service"
)

type Task struct{}

var taskService = crmservice.NewTaskService()

func (Task) GetDetail(c *server.Context) error {
	taskID := uint64FromInput(c.Input("task_id"))
	if taskID == 0 {
		taskID = uint64FromInput(c.Input("id"))
	}
	data, err := taskService.DetailFromDB(c.Context(), taskID)
	return crmJSON(c, data, err)
}

func (Task) PostSubmit(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := taskService.SubmitToDB(c.Context(), body)
	return crmJSON(c, data, err)
}

func (Task) PostAssign(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := taskService.Assign(c.Context(), body)
	return crmJSON(c, data, err)
}
