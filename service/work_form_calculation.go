package service

import (
	"context"
	"fmt"
	"strings"

	crmmodel "github.com/dever-package/crm/model"
)

type workFormCalculation struct {
	Result      TaskRuleResult
	RawFields   map[string]any
	FieldValues map[string]any
}

func (WorkService) CalculateForm(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	todo, task, err := pendingTodoTaskForStaff(ctx, staff, payload)
	if err != nil {
		return nil, err
	}
	calculation, err := evaluateWorkFormCalculation(ctx, todo, task, workActionValues(payload))
	if err != nil {
		return nil, err
	}
	return workFormCalculationResponse(calculation), nil
}

func applyWorkFormCalculation(
	ctx context.Context,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	values map[string]any,
	requirePassed bool,
) (*workFormCalculation, error) {
	form := workCalculationForm(ctx, task)
	if form == nil || form.CalculationScriptID == 0 {
		return nil, nil
	}
	calculation, err := evaluateWorkFormCalculationWithForm(ctx, todo, task, form, values)
	if err != nil {
		return nil, err
	}
	for key, value := range calculation.RawFields {
		values[key] = value
	}
	if requirePassed && !calculation.Result.Passed {
		reason := strings.TrimSpace(calculation.Result.Reason)
		if reason == "" {
			reason = "请补充测算所需信息"
		}
		return calculation, fmt.Errorf("表单计算未完成：%s", reason)
	}
	return calculation, nil
}

func evaluateWorkFormCalculation(
	ctx context.Context,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	values map[string]any,
) (*workFormCalculation, error) {
	form := workCalculationForm(ctx, task)
	if form == nil || form.CalculationScriptID == 0 {
		return nil, fmt.Errorf("任务表单未配置计算规则")
	}
	return evaluateWorkFormCalculationWithForm(ctx, todo, task, form, values)
}

func evaluateWorkFormCalculationWithForm(
	ctx context.Context,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	form *crmmodel.Form,
	values map[string]any,
) (*workFormCalculation, error) {
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{
		"id":     form.CalculationScriptID,
		"status": crmmodel.StatusEnabled,
	})
	if script == nil {
		return nil, fmt.Errorf("表单计算规则不存在或已停用")
	}
	fields, inputValues := workFormCalculationFields(ctx, form.ID, values)
	input := workRuleInput(ctx, todo, task)
	input["form"] = inputValues
	input["calculation"] = map[string]any{
		"form_id":   form.ID,
		"form_name": form.Name,
		"script_id": script.ID,
	}
	result, err := evaluateTaskRuleScript(ctx, script.Script, input)
	if err != nil {
		return nil, fmt.Errorf("表单计算失败：%w", err)
	}
	rawFields, fieldValues, err := mapWorkFormCalculationOutputs(fields, result.OutputFields)
	if err != nil {
		return nil, err
	}
	return &workFormCalculation{
		Result:      result,
		RawFields:   rawFields,
		FieldValues: fieldValues,
	}, nil
}

func workCalculationForm(ctx context.Context, task *crmmodel.Task) *crmmodel.Form {
	if task == nil || task.FormID == 0 {
		return nil
	}
	if task.TaskType != crmmodel.TaskTypeForm && task.TaskType != crmmodel.TaskTypeApproval {
		return nil
	}
	return crmmodel.NewFormModel().Find(ctx, map[string]any{
		"id":     task.FormID,
		"status": crmmodel.StatusEnabled,
	})
}

func workFormCalculationFields(
	ctx context.Context,
	formID uint64,
	values map[string]any,
) (map[string]*crmmodel.FormField, map[string]any) {
	fields := map[string]*crmmodel.FormField{}
	inputValues := map[string]any{}
	for _, sourceField := range crmmodel.NewFormFieldModel().Select(ctx, map[string]any{
		"form_id": formID,
		"status":  crmmodel.StatusEnabled,
	}) {
		for _, binding := range expandWorkInputFormFieldBindings(ctx, sourceField) {
			formField := binding.FormField
			if formField == nil {
				continue
			}
			if formField.DataFieldID == 0 {
				if mainField := strings.TrimSpace(formField.MainField); mainField != "" {
					inputValues[mainField] = values[workFieldInputKey(formField)]
				}
				continue
			}
			dataField := binding.DataField
			if dataField == nil || strings.TrimSpace(dataField.FieldKey) == "" || dataField.FieldType == "group" {
				continue
			}
			fields[dataField.FieldKey] = formField
			inputValues[dataField.FieldKey] = values[workFieldInputKey(formField)]
		}
	}
	return fields, inputValues
}

func mapWorkFormCalculationOutputs(
	fields map[string]*crmmodel.FormField,
	outputs map[string]any,
) (map[string]any, map[string]any, error) {
	rawFields := map[string]any{}
	fieldValues := map[string]any{}
	for fieldKey, value := range outputs {
		fieldKey = strings.TrimSpace(fieldKey)
		if fieldKey == "" {
			continue
		}
		field := fields[fieldKey]
		if field == nil {
			return nil, nil, fmt.Errorf("计算规则输出字段不属于当前任务表单：%s", fieldKey)
		}
		rawFields[workFieldInputKey(field)] = value
		fieldValues[fieldKey] = value
	}
	return rawFields, fieldValues, nil
}

func workFormCalculationResponse(calculation *workFormCalculation) map[string]any {
	if calculation == nil {
		return map[string]any{}
	}
	return map[string]any{
		"passed":       calculation.Result.Passed,
		"value":        calculation.Result.Value,
		"reason":       calculation.Result.Reason,
		"fields":       calculation.RawFields,
		"field_values": calculation.FieldValues,
		"result":       calculation.Result.RawResult,
		"duration_ms":  calculation.Result.DurationMS,
	}
}
