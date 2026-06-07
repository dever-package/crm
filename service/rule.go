package service

import (
	"context"
	"time"

	fronteval "my/package/front/service/eval"
)

type RuleService struct{}

type ScriptValidateRequest struct {
	Language        string
	Script          string
	Entry           string
	Input           any
	Config          any
	Expected        any
	CompareExpected bool
	TimeoutMS       int
}

func NewRuleService() RuleService {
	return RuleService{}
}

func (RuleService) Validate(ctx context.Context, req ScriptValidateRequest) (fronteval.ValidateResult, error) {
	timeout := fronteval.DefaultTimeout
	if req.TimeoutMS > 0 {
		timeout = time.Duration(req.TimeoutMS) * time.Millisecond
	}
	return fronteval.Validate(ctx, fronteval.ValidateRequest{
		Request: fronteval.Request{
			Language: req.Language,
			Script:   req.Script,
			Input:    req.Input,
			Config:   req.Config,
			Entry:    req.Entry,
			Timeout:  timeout,
		},
		Expected:        req.Expected,
		CompareExpected: req.CompareExpected,
	})
}
