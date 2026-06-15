package api

import (
	"github.com/shemic/dever/server"

	crmservice "my/package/crm/service"
)

type Script struct{}

var ruleService = crmservice.NewRuleService()

func (Script) PostValidate(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := ruleService.Validate(c.Context(), crmservice.ScriptValidateRequest{
		Script:          textFromBody(body, "script"),
		Input:           body["input"],
		Config:          body["config"],
		Expected:        body["expected"],
		CompareExpected: boolFromBody(body, "compare_expected", "compareExpected"),
	})
	return crmJSON(c, data, err)
}

func (Script) PostDryRun(c *server.Context) error {
	body, err := bindBody(c)
	if err != nil {
		return c.Error(err)
	}
	data, err := ruleService.DryRun(c.Context(), crmservice.ScriptDryRunRequest{
		Script:     textFromBody(body, "script"),
		ScriptID:   uint64FromBody(body, "script_id", "scriptId"),
		TaskID:     uint64FromBody(body, "task_id", "taskId"),
		CustomerID: uint64FromBody(body, "customer_id", "customerId"),
		AssetID:    uint64FromBody(body, "asset_id", "assetId"),
		StaffID:    uint64FromBody(body, "staff_id", "staffId"),
		Staff:      crmservice.CurrentWorkStaff(c.Context()),
	})
	return crmJSON(c, data, err)
}
