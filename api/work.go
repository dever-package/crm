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

func (Work) PostChangePassword(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.ChangePassword(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) GetNavigation(c *server.Context) error {
	data, err := workService.Navigation(c.Context(), crmservice.CurrentWorkStaff(c.Context()))
	return crmJSON(c, data, err)
}

func (Work) GetGlobalSearch(c *server.Context) error {
	data, err := workService.GlobalSearch(c.Context(), crmservice.CurrentWorkStaff(c.Context()), map[string]any{
		"keyword": c.Input("keyword"),
	})
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
		"workflow_id":   c.Input("workflow_id"),
		"workflowId":    c.Input("workflowId"),
		"page":          c.Input("page"),
		"page_size":     c.Input("page_size"),
		"pageSize":      c.Input("pageSize"),
		"limit":         c.Input("limit"),
	})
	return crmJSON(c, data, err)
}

func (Work) GetLeads(c *server.Context) error {
	data, err := workService.LeadPool(c.Context(), crmservice.CurrentWorkStaff(c.Context()), workLeadQueryPayload(c))
	return crmJSON(c, data, err)
}

func (Work) GetLeadExport(c *server.Context) error {
	data, err := workService.LeadExport(c.Context(), crmservice.CurrentWorkStaff(c.Context()), workLeadQueryPayload(c))
	return crmJSON(c, data, err)
}

func (Work) GetLeadDetail(c *server.Context) error {
	data, err := workService.LeadDetail(c.Context(), crmservice.CurrentWorkStaff(c.Context()), map[string]any{
		"lead_id":     c.Input("lead_id"),
		"leadId":      c.Input("leadId"),
		"workflow_id": c.Input("workflow_id"),
		"workflowId":  c.Input("workflowId"),
	})
	return crmJSON(c, data, err)
}

func workLeadQueryPayload(c *server.Context) map[string]any {
	return map[string]any{
		"keyword":        c.Input("keyword"),
		"status":         c.Input("status"),
		"source_id":      c.Input("source_id"),
		"sourceId":       c.Input("sourceId"),
		"channel_id":     c.Input("channel_id"),
		"channelId":      c.Input("channelId"),
		"owner_staff_id": c.Input("owner_staff_id"),
		"ownerStaffId":   c.Input("ownerStaffId"),
		"created_from":   c.Input("created_from"),
		"createdFrom":    c.Input("createdFrom"),
		"created_to":     c.Input("created_to"),
		"createdTo":      c.Input("createdTo"),
		"scope":          c.Input("scope"),
		"quick_filter":   c.Input("quick_filter"),
		"quickFilter":    c.Input("quickFilter"),
		"stage_filter":   c.Input("stage_filter"),
		"stage":          c.Input("stage"),
		"task_filter":    c.Input("task_filter"),
		"task":           c.Input("task"),
		"workflow_id":    c.Input("workflow_id"),
		"workflowId":     c.Input("workflowId"),
		"page":           c.Input("page"),
		"page_size":      c.Input("page_size"),
		"pageSize":       c.Input("pageSize"),
	}
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
		"customer_id":          c.Input("customer_id"),
		"customerId":           c.Input("customerId"),
		"asset_id":             c.Input("asset_id"),
		"assetId":              c.Input("assetId"),
		"workflow_instance_id": c.Input("workflow_instance_id"),
		"workflowInstanceId":   c.Input("workflowInstanceId"),
	})
	return crmJSON(c, data, err)
}

func (Work) GetSummary(c *server.Context) error {
	data, err := workService.Summary(c.Context(), crmservice.CurrentWorkStaff(c.Context()))
	return crmJSON(c, data, err)
}

func (Work) GetSchedules(c *server.Context) error {
	data, err := workService.Schedules(c.Context(), crmservice.CurrentWorkStaff(c.Context()), map[string]any{
		"start_at": c.Input("start_at"),
		"startAt":  c.Input("startAt"),
		"end_at":   c.Input("end_at"),
		"endAt":    c.Input("endAt"),
		"status":   c.Input("status"),
	})
	return crmJSON(c, data, err)
}

func (Work) GetScheduleOptions(c *server.Context) error {
	data, err := workService.ScheduleOptions(c.Context(), crmservice.CurrentWorkStaff(c.Context()))
	return crmJSON(c, data, err)
}

func (Work) GetScheduleReminders(c *server.Context) error {
	data, err := workService.ScheduleReminders(c.Context(), crmservice.CurrentWorkStaff(c.Context()))
	return crmJSON(c, data, err)
}

func (Work) PostSchedule(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.ScheduleCalendar(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostReschedule(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.RescheduleCalendar(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostCompleteSchedule(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.CompleteCalendar(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostCancelSchedule(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.CancelCalendar(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostReadScheduleReminder(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.ReadScheduleReminder(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) PostCheckInSchedule(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.CheckInSchedule(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
	return crmJSON(c, data, err)
}

func (Work) GetOperations(c *server.Context) error {
	data, err := workService.Operations(c.Context(), crmservice.CurrentWorkStaff(c.Context()), map[string]any{
		"customer_id":          c.Input("customer_id"),
		"asset_id":             c.Input("asset_id"),
		"workflow_instance_id": c.Input("workflow_instance_id"),
		"workflowInstanceId":   c.Input("workflowInstanceId"),
		"mine":                 c.Input("mine"),
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

func (Work) PostRestartFlow(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := workService.RestartFlow(c.Context(), crmservice.CurrentWorkStaff(c.Context()), body)
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
