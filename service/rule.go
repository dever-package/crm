package service

import (
	"context"
	"fmt"
	"strings"

	crmmodel "github.com/dever-package/crm/model"
	fronteval "github.com/dever-package/front/service/eval"
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

type TaskRuleResult struct {
	Passed       bool
	Value        string
	Reason       string
	OutputFields map[string]any
	ProductCodes []string
	RawResult    any
	DurationMS   int64
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

func (RuleService) EvaluateTask(ctx context.Context, task *crmmodel.Task, input map[string]any) (TaskRuleResult, error) {
	if task == nil || task.TaskType != crmmodel.TaskTypeRule {
		return TaskRuleResult{}, fmt.Errorf("任务不是自动核验类型")
	}
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{
		"id":     task.ScriptID,
		"status": crmmodel.StatusEnabled,
	})
	if script == nil {
		return TaskRuleResult{}, fmt.Errorf("核验规则不存在或已停用")
	}
	return evaluateTaskRuleScript(ctx, script.Script, input)
}

func evaluateTaskRuleScript(ctx context.Context, script string, input map[string]any) (TaskRuleResult, error) {
	if strings.TrimSpace(script) == "" {
		return TaskRuleResult{}, fmt.Errorf("规则脚本不能为空")
	}
	result, err := fronteval.Run(ctx, fronteval.Request{
		Language: fronteval.LanguageJavaScript,
		Script:   script,
		Entry:    fronteval.DefaultEntry,
		Input:    input,
		Config:   map[string]any{},
	})
	if err != nil {
		return TaskRuleResult{}, err
	}
	normalized := normalizeTaskRuleResult(result.Value)
	normalized.RawResult = result.Value
	normalized.DurationMS = result.DurationMS
	return normalized, nil
}

func normalizeTaskRuleResult(value any) TaskRuleResult {
	if passed, ok := value.(bool); ok {
		return TaskRuleResult{Passed: passed}
	}
	payload := mapFromAny(value)
	if len(payload) == 0 {
		return TaskRuleResult{Reason: "规则必须返回 true/false、{ passed, reason } 或 { value, reason }"}
	}
	outputFields := mapFromAny(firstPresent(payload, "fields", "output_fields", "outputFields"))
	productCodes := taskRuleProductCodes(payload)
	if _, exists := payload["passed"]; exists {
		return TaskRuleResult{
			Passed:       booleanFromAny(payload["passed"]),
			Value:        firstText(payload, "value", "result"),
			Reason:       firstText(payload, "reason", "message"),
			OutputFields: outputFields,
			ProductCodes: productCodes,
		}
	}
	resultValue := firstText(payload, "value", "result")
	if resultValue != "" {
		return TaskRuleResult{
			Passed:       true,
			Value:        resultValue,
			Reason:       firstText(payload, "reason", "message"),
			OutputFields: outputFields,
			ProductCodes: productCodes,
		}
	}
	return TaskRuleResult{
		Reason: "规则必须返回 true/false、{ passed, reason } 或 { value, reason }",
	}
}

func taskRuleProductCodes(payload map[string]any) []string {
	raw, exists := payload["product_codes"]
	if !exists {
		raw, exists = payload["productCodes"]
	}
	if !exists {
		return nil
	}
	values := stringListFromAny(raw)
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func (RuleService) DryRun(ctx context.Context, req ScriptDryRunRequest) (map[string]any, error) {
	staff := resolveScriptDryRunStaff(ctx, req)
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请选择样例执行人")
	}
	task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": req.TaskID})
	if task == nil || task.TaskType != crmmodel.TaskTypeRule {
		return nil, fmt.Errorf("请选择自动核验任务")
	}
	if req.CustomerID == 0 || crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": req.CustomerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	if req.AssetID == 0 || !workCustomerOwnsAsset(ctx, req.CustomerID, req.AssetID) {
		return nil, fmt.Errorf("资产不属于当前客户")
	}
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": task.StageID})
	if stage == nil {
		return nil, fmt.Errorf("任务阶段不存在")
	}
	script, scriptID, err := scriptDryRunSource(ctx, req, task)
	if err != nil {
		return nil, err
	}
	todo := &crmmodel.WorkTodo{
		CustomerID: req.CustomerID,
		AssetID:    req.AssetID,
		WorkflowID: stage.WorkflowID,
		StageID:    stage.ID,
		TaskID:     task.ID,
	}
	input := workRuleInput(ctx, todo, task)
	input["staff"] = map[string]any{
		"id":            staff.ID,
		"name":          staff.Name,
		"phone":         staff.Phone,
		"department_id": staff.DepartmentID,
	}
	response := map[string]any{
		"matched":     false,
		"script_id":   scriptID,
		"input":       input,
		"raw_result":  nil,
		"duration_ms": int64(0),
		"task": map[string]any{
			"id":        task.ID,
			"name":      task.Name,
			"task_type": task.TaskType,
		},
	}
	result, err := evaluateTaskRuleScript(ctx, script, input)
	if err != nil {
		response["error"] = err.Error()
		return response, nil
	}
	response["matched"] = result.Passed
	response["passed"] = result.Passed
	response["value"] = result.Value
	response["reason"] = result.Reason
	response["fields"] = result.OutputFields
	if result.ProductCodes != nil {
		response["product_codes"] = result.ProductCodes
	}
	response["raw_result"] = result.RawResult
	response["duration_ms"] = result.DurationMS
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
		CanDispatch:  staff.CanDispatch,
	}
}

func scriptDryRunSource(ctx context.Context, req ScriptDryRunRequest, task *crmmodel.Task) (string, uint64, error) {
	if strings.TrimSpace(req.Script) != "" {
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
