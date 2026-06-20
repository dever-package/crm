package api

import (
	"github.com/shemic/dever/server"

	crmapi "github.com/dever-package/crm/api"
	crmservice "github.com/dever-package/crm/service"
)

type Script struct{}

var ruleService = crmservice.NewRuleService()

func (Script) PostValidate(c *server.Context) error {
	body, err := crmapi.BindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := ruleService.Validate(c.Context(), crmservice.ScriptValidateRequest{
		Script:          crmapi.TextFromBody(body, "script"),
		Input:           body["input"],
		Config:          body["config"],
		Expected:        body["expected"],
		CompareExpected: crmapi.BoolFromBody(body, "compare_expected", "compareExpected"),
	})
	return crmapi.WriteJSON(c, data, err)
}

func (Script) PostDryRun(c *server.Context) error {
	body, err := crmapi.BindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := ruleService.DryRun(c.Context(), crmservice.ScriptDryRunRequest{
		Script:     crmapi.TextFromBody(body, "script"),
		ScriptID:   crmapi.Uint64FromBody(body, "script_id", "scriptId"),
		TaskID:     crmapi.Uint64FromBody(body, "task_id", "taskId"),
		CustomerID: crmapi.Uint64FromBody(body, "customer_id", "customerId"),
		AssetID:    crmapi.Uint64FromBody(body, "asset_id", "assetId"),
		StaffID:    crmapi.Uint64FromBody(body, "staff_id", "staffId"),
		Staff:      crmservice.CurrentWorkStaff(c.Context()),
	})
	return crmapi.WriteJSON(c, data, err)
}
