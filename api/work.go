package api

import (
	"github.com/shemic/dever/server"

	crmservice "github.com/dever-package/crm/service"
)

type Work struct{}

var workService = crmservice.NewWorkService()

func (Work) PostLogin(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.Login(c.Context(), body)
	return crmJSON(c, data, err)
}

func (Work) GetFeishuConfig(c *server.Context) error {
	data, err := workService.FeishuConfig(c.Context())
	return crmJSON(c, data, err)
}

func (Work) PostFeishuLogin(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.FeishuLogin(c.Context(), body)
	return crmJSON(c, data, err)
}

func (Work) GetMe(c *server.Context) error {
	data, err := workService.Me(c.Context(), crmservice.CurrentWorkStaff(c.Context()))
	return crmJSON(c, data, err)
}

func (Work) GetCustomers(c *server.Context) error {
	data, err := workService.Customers(c.Context(), crmservice.CurrentWorkStaff(c.Context()), map[string]any{
		"keyword":       c.Input("keyword"),
		"customer_no":   c.Input("customer_no"),
		"customer_name": c.Input("customer_name"),
		"phone":         c.Input("phone"),
		"wechat":        c.Input("wechat"),
		"asset_no":      c.Input("asset_no"),
		"status":        c.Input("status"),
		"mode":          c.Input("mode"),
		"quick_filter":  c.Input("quick_filter"),
		"quickFilter":   c.Input("quickFilter"),
		"stage_filter":  c.Input("stage_filter"),
		"stage":         c.Input("stage"),
		"task_filter":   c.Input("task_filter"),
		"task":          c.Input("task"),
		"scope":         c.Input("scope"),
		"page":          c.Input("page"),
		"page_size":     c.Input("page_size"),
		"pageSize":      c.Input("pageSize"),
		"limit":         c.Input("limit"),
	})
	return crmJSON(c, data, err)
}

func (Work) GetLeads(c *server.Context) error {
	data, err := workService.LeadPool(c.Context(), crmservice.CurrentWorkStaff(c.Context()), map[string]any{
		"keyword":   c.Input("keyword"),
		"status":    c.Input("status"),
		"page":      c.Input("page"),
		"page_size": c.Input("page_size"),
		"pageSize":  c.Input("pageSize"),
	})
	return crmJSON(c, data, err)
}

func (Work) PostCreateLead(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.RegisterLead(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostLeadAction(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.ActOnLead(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) GetCustomerDetail(c *server.Context) error {
	data, err := workService.CustomerDetail(c.Context(), crmservice.CurrentWorkStaff(c.Context()), map[string]any{
		"customer_id": c.Input("customer_id"),
		"customerId":  c.Input("customerId"),
		"asset_id":    c.Input("asset_id"),
		"assetId":     c.Input("assetId"),
	})
	return crmJSON(c, data, err)
}

func (Work) GetSummary(c *server.Context) error {
	data, err := workService.Summary(c.Context(), crmservice.CurrentWorkStaff(c.Context()))
	return crmJSON(c, data, err)
}

func (Work) GetOperations(c *server.Context) error {
	data, err := workService.Operations(c.Context(), crmservice.CurrentWorkStaff(c.Context()), map[string]any{
		"customer_id": c.Input("customer_id"),
		"asset_id":    c.Input("asset_id"),
		"mine":        c.Input("mine"),
	})
	return crmJSON(c, data, err)
}

func (Work) GetTasks(c *server.Context) error {
	customerID := uint64FromInput(c.Input("customer_id"))
	assetID := uint64FromInput(c.Input("asset_id"))
	var data map[string]any
	var err error
	if assetID > 0 {
		data, err = workService.Tasks(c.Context(), crmservice.CurrentWorkStaff(c.Context()), customerID, assetID)
	} else {
		data, err = workService.Tasks(c.Context(), crmservice.CurrentWorkStaff(c.Context()), customerID)
	}
	return crmJSON(c, data, err)
}

func (Work) GetFlowAssignees(c *server.Context) error {
	data, err := workService.FlowAssignees(c.Context(), crmservice.CurrentWorkStaff(c.Context()), map[string]any{
		"todo_id":              c.Input("todo_id"),
		"todoId":               c.Input("todoId"),
		"workflow_instance_id": c.Input("workflow_instance_id"),
		"workflowInstanceId":   c.Input("workflowInstanceId"),
		"target":               c.Input("target"),
	})
	return crmJSON(c, data, err)
}

func (Work) PostAssignFlowTask(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.AssignFlowTask(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostChangeFlowOwner(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.ChangeFlowOwner(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostCompleteFlowStage(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.CompleteFlowStage(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostTerminateFlow(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.TerminateFlow(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostExecute(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.Execute(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostAiFill(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.AIFill(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}
