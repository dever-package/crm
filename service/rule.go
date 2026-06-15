package service

import (
	"context"
	"fmt"

	crmmodel "my/package/crm/model"
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

type ScriptDryRunRequest struct {
	Script     string
	ScriptID   uint64
	TaskID     uint64
	CustomerID uint64
	AssetID    uint64
	StaffID    uint64
	Staff      *WorkStaffSession
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

func (RuleService) DryRun(ctx context.Context, req ScriptDryRunRequest) (map[string]any, error) {
	staff := resolveScriptDryRunStaff(ctx, req)
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请选择样例执行人")
	}
	task := crmmodel.NewTaskModel().Find(ctx, map[string]any{
		"id":     req.TaskID,
		"status": crmmodel.StatusEnabled,
	})
	if task == nil {
		return nil, fmt.Errorf("任务不存在或已停用")
	}
	if task.TaskType != crmmodel.TaskTypeDecision {
		return nil, fmt.Errorf("脚本干跑目前仅支持决策任务")
	}
	if req.CustomerID == 0 || crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": req.CustomerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	if req.AssetID > 0 && !workCustomerOwnsAsset(ctx, req.CustomerID, req.AssetID) {
		return nil, fmt.Errorf("资产不属于当前客户")
	}

	script, scriptID, err := scriptDryRunSource(ctx, req, task)
	if err != nil {
		return nil, err
	}
	state := currentWorkCustomerStage(ctx, req.CustomerID, req.AssetID)
	input := workDecisionInput(ctx, staff, task, req.CustomerID, req.AssetID, state)
	config := mapFromAny(task.ConfigJSON)
	response := scriptDryRunBaseResponse(task, scriptID, input, config)

	result, err := fronteval.Run(ctx, fronteval.Request{
		Language: fronteval.LanguageJavaScript,
		Script:   script,
		Entry:    fronteval.DefaultEntry,
		Input:    input,
		Config:   config,
	})
	if err != nil {
		response["error"] = err.Error()
		return response, nil
	}
	response["raw_result"] = result.Value
	response["duration_ms"] = result.DurationMS

	decision, err := normalizeWorkDecisionResult(result.Value, result.DurationMS)
	if err != nil {
		response["error"] = err.Error()
		return response, nil
	}
	response["matched"] = true
	response["decision_result"] = decision
	return response, nil
}

func resolveScriptDryRunStaff(ctx context.Context, req ScriptDryRunRequest) *WorkStaffSession {
	if req.Staff != nil && req.Staff.ID > 0 {
		return req.Staff
	}
	if req.StaffID == 0 {
		return nil
	}
	staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"id":     req.StaffID,
		"status": crmmodel.StatusEnabled,
	})
	if staff == nil {
		return nil
	}
	return &WorkStaffSession{
		ID:           staff.ID,
		Name:         staff.Name,
		Phone:        staff.Phone,
		DepartmentID: staff.DepartmentID,
	}
}

func scriptDryRunSource(ctx context.Context, req ScriptDryRunRequest, task *crmmodel.Task) (string, uint64, error) {
	if req.Script != "" {
		return req.Script, req.ScriptID, nil
	}
	scriptID := req.ScriptID
	if scriptID == 0 && task != nil {
		scriptID = task.ScriptID
	}
	if scriptID == 0 {
		return "", 0, fmt.Errorf("脚本内容不能为空")
	}
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{
		"id":     scriptID,
		"status": crmmodel.StatusEnabled,
	})
	if script == nil {
		return "", 0, fmt.Errorf("脚本规则不存在或已停用")
	}
	return script.Script, script.ID, nil
}

func scriptDryRunBaseResponse(task *crmmodel.Task, scriptID uint64, input map[string]any, config map[string]any) map[string]any {
	return map[string]any{
		"matched":     false,
		"script_id":   scriptID,
		"input":       input,
		"config":      config,
		"raw_result":  nil,
		"duration_ms": int64(0),
		"task": map[string]any{
			"id":           task.ID,
			"name":         task.Name,
			"task_type":    task.TaskType,
			"trigger_type": task.TriggerType,
		},
	}
}
