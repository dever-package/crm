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

func defaultResourceNo() string {
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
