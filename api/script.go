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
