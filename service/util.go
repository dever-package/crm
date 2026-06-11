package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	frontstream "my/package/front/service/stream"
)

func inputUint64(value any) uint64 {
	return uint64(frontstream.InputInt64(value, 0))
}

func inputInt(value any) int {
	return int(frontstream.InputInt64(value, 0))
}

func inputText(value any) string {
	return strings.TrimSpace(frontstream.InputText(value))
}

func firstText(body map[string]any, keys ...string) string {
	for _, key := range keys {
		if text := inputText(body[key]); text != "" {
			return text
		}
	}
	return ""
}

func firstUint64(body map[string]any, keys ...string) uint64 {
	for _, key := range keys {
		if value := inputUint64(body[key]); value > 0 {
			return value
		}
	}
	return 0
}

func firstOptionalUint64(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	return values[0]
}

func defaultAssetNo() string {
	return fmt.Sprintf("DP%s", time.Now().Format("20060102150405"))
}

func mapFromAny(value any) map[string]any {
	switch row := value.(type) {
	case map[string]any:
		return row
	case string:
		var result map[string]any
		if err := json.Unmarshal([]byte(row), &result); err == nil && result != nil {
			return result
		}
	}
	return map[string]any{}
}

func copyMap(row map[string]any) map[string]any {
	result := make(map[string]any, len(row))
	for key, value := range row {
		result[key] = value
	}
	return result
}

func mapListFromAny(value any) []map[string]any {
	switch rows := value.(type) {
	case []map[string]any:
		return rows
	case []any:
		result := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			if mapped := mapFromAny(row); len(mapped) > 0 {
				result = append(result, mapped)
			}
		}
		return result
	case string:
		var decoded []map[string]any
		if err := json.Unmarshal([]byte(rows), &decoded); err == nil {
			return decoded
		}
		var generic []any
		if err := json.Unmarshal([]byte(rows), &generic); err == nil {
			return mapListFromAny(generic)
		}
	}
	return []map[string]any{}
}

func stringListFromJSON(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := inputText(item); text != "" {
				result = append(result, text)
			}
		}
		return result
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		var decoded []string
		if err := json.Unmarshal([]byte(typed), &decoded); err == nil {
			return decoded
		}
		var generic []any
		if err := json.Unmarshal([]byte(typed), &generic); err == nil {
			return stringListFromJSON(generic)
		}
	}
	return nil
}

func containsFold(text string, keyword string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	return keyword == "" || strings.Contains(text, keyword)
}

func jsonText(value any) string {
	if value == nil {
		return "{}"
	}
	if text, ok := value.(string); ok {
		if strings.TrimSpace(text) == "" {
			return "{}"
		}
		return text
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func numericValue(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case uint64:
		return float64(v)
	case json.Number:
		number, _ := v.Float64()
		return number
	case string:
		number, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return number
	default:
		return 0
	}
}
