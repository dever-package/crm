package service

import (
	"context"
	"fmt"
	"strings"

	crmmodel "github.com/dever-package/crm/model"
)

const elevenDimensionProbeCount = 11
const elevenDimensionRuleScriptName = "十一维T节点自动判断"

func elevenDimensionProbeCode(index int) string {
	return fmt.Sprintf("P%02d", index)
}

func normalizeElevenDimensionProbeCode(value string) (string, bool) {
	parts := strings.Fields(strings.ToUpper(strings.TrimSpace(value)))
	if len(parts) == 0 || len(parts[0]) != 3 || parts[0][0] != 'P' {
		return "", false
	}
	digit1 := parts[0][1]
	digit2 := parts[0][2]
	if digit1 < '0' || digit1 > '9' || digit2 < '0' || digit2 > '9' {
		return "", false
	}
	index := int(digit1-'0')*10 + int(digit2-'0')
	if index < 1 || index > elevenDimensionProbeCount {
		return "", false
	}
	return elevenDimensionProbeCode(index), true
}

func elevenDimensionProbeFieldCode(field *crmmodel.DataField, parentNames map[uint64]string) (string, bool) {
	if field == nil || field.FieldType == "group" {
		return "", false
	}
	if code, ok := normalizeElevenDimensionProbeCode(field.FieldKey); ok {
		return code, true
	}
	return normalizeElevenDimensionProbeCode(parentNames[field.ParentFieldID])
}

func elevenDimensionProbeFields(fields []*crmmodel.DataField, parentNames map[uint64]string) map[uint64]string {
	fieldIDsByCode := map[string]uint64{}
	for _, field := range fields {
		code, ok := elevenDimensionProbeFieldCode(field, parentNames)
		if !ok || fieldIDsByCode[code] != 0 {
			continue
		}
		fieldIDsByCode[code] = field.ID
	}
	result := make(map[uint64]string, len(fieldIDsByCode))
	for code, fieldID := range fieldIDsByCode {
		result[fieldID] = code
	}
	return result
}

func isElevenDimensionProbeTemplate(probeFields map[uint64]string) bool {
	return len(probeFields) == elevenDimensionProbeCount
}

func isElevenDimensionRuleTask(ctx context.Context, task *crmmodel.Task) bool {
	if task == nil || task.ScriptID == 0 {
		return false
	}
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{
		"id":   task.ScriptID,
		"name": elevenDimensionRuleScriptName,
	})
	return script != nil
}

func elevenDimensionRuleInputSnapshot(input map[string]any) map[string]any {
	current := mapFromAny(input["current"])
	asset := mapFromAny(current["asset"])
	fields := mapFromAny(asset["fields"])
	snapshot := make(map[string]any, elevenDimensionProbeCount)
	for index := 1; index <= elevenDimensionProbeCount; index++ {
		code := elevenDimensionProbeCode(index)
		snapshot[code] = fields[code]
	}
	return snapshot
}

func isElevenDimensionRuleOutput(fields map[string]any) bool {
	for _, key := range []string{"eleven_completion", "npl_candidate_t", "routing_t", "rule_release_version"} {
		if _, exists := fields[key]; !exists {
			return false
		}
	}
	return true
}

func operationBackedElevenDimensionFields(operationID uint64, fields map[string]any) map[string]any {
	if operationID == 0 || !isElevenDimensionRuleOutput(fields) {
		return fields
	}
	result := make(map[string]any, len(fields)+1)
	for key, value := range fields {
		result[key] = value
	}
	result["judgment_snapshot_no"] = fmt.Sprintf("JUDGMENT-%d", operationID)
	return result
}
