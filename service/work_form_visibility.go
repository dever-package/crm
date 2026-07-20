package service

import (
	"encoding/json"
	"fmt"
	"strings"

	crmmodel "github.com/dever-package/crm/model"
)

func workFormFieldVisible(field *crmmodel.FormField, values map[string]any) bool {
	if field == nil || field.VisibleWhenFieldID == 0 {
		return true
	}
	driverValue := workFormConditionValue(values, field.VisibleWhenFieldID)
	switch field.VisibleWhenOperator {
	case crmmodel.FormFieldVisibleEmpty:
		return emptyWorkFieldValue(driverValue)
	case crmmodel.FormFieldVisibleNotEmpty:
		return !emptyWorkFieldValue(driverValue)
	}
	matched := workFormConditionMatches(driverValue, workFormConditionExpectedValues(field.VisibleWhenValue))
	switch field.VisibleWhenOperator {
	case crmmodel.FormFieldVisibleNotEquals, crmmodel.FormFieldVisibleNotIn:
		return !matched
	case crmmodel.FormFieldVisibleEquals, crmmodel.FormFieldVisibleIn, "":
		return matched
	default:
		return false
	}
}

func workFormConditionValue(values map[string]any, dataFieldID uint64) any {
	if dataFieldID == 0 || len(values) == 0 {
		return nil
	}
	for _, key := range []string{
		fmt.Sprintf("data:%d", dataFieldID),
		fmt.Sprintf("%d", dataFieldID),
	} {
		if value, exists := values[key]; exists {
			return value
		}
	}
	return nil
}

func workFormConditionExpectedValues(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	var decoded []any
	if strings.HasPrefix(value, "[") && json.Unmarshal([]byte(value), &decoded) == nil {
		return normalizedWorkFormConditionValues(decoded)
	}
	parts := strings.FieldsFunc(value, func(char rune) bool {
		return char == ',' || char == '\n' || char == '\r'
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if normalized := strings.TrimSpace(part); normalized != "" {
			result = append(result, normalized)
		}
	}
	return result
}

func workFormConditionMatches(actual any, expected []string) bool {
	if len(expected) == 0 {
		return false
	}
	actualValues := normalizedWorkFormConditionValues(actual)
	for _, current := range actualValues {
		for _, target := range expected {
			if current == target {
				return true
			}
		}
	}
	return false
}

func normalizedWorkFormConditionValues(value any) []string {
	switch typed := value.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if normalized := strings.TrimSpace(fmt.Sprint(item)); normalized != "" {
				result = append(result, normalized)
			}
		}
		return result
	case []string:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if normalized := strings.TrimSpace(item); normalized != "" {
				result = append(result, normalized)
			}
		}
		return result
	default:
		normalized := strings.TrimSpace(fmt.Sprint(value))
		if normalized == "" || normalized == "<nil>" {
			return []string{}
		}
		return []string{normalized}
	}
}
