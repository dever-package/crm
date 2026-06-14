package service

import (
	"context"

	fronteval "my/package/front/service/eval"
)

type RuleService struct{}

type ScriptValidateRequest struct {
	Script          string
	Input           any
	Config          any
	Expected        any
	CompareExpected bool
}

func NewRuleService() RuleService {
	return RuleService{}
}

func (RuleService) Validate(ctx context.Context, req ScriptValidateRequest) (fronteval.ValidateResult, error) {
	return fronteval.Validate(ctx, fronteval.ValidateRequest{
		Request: fronteval.Request{
			Language: fronteval.LanguageJavaScript,
			Script:   req.Script,
			Input:    req.Input,
			Config:   req.Config,
			Entry:    fronteval.DefaultEntry,
		},
		Expected:        req.Expected,
		CompareExpected: req.CompareExpected,
	})
}
