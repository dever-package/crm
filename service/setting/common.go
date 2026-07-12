package setting

import (
	"context"
	"strings"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	frontaction "github.com/dever-package/front/service/action"
)

type CrmHook struct{}

func cloneCrmRecord(params []any) map[string]any {
	if len(params) == 0 || params[0] == nil {
		return map[string]any{}
	}
	if row, ok := params[0].(map[string]any); ok {
		return util.CloneMap(row)
	}
	return map[string]any{}
}

func isPartialCrmRecord(record map[string]any) bool {
	switch value := record["_partial"].(type) {
	case bool:
		return value
	case string:
		normalized := strings.ToLower(strings.TrimSpace(value))
		return normalized == "1" || normalized == "true" || normalized == "yes"
	case int:
		return value != 0
	case int64:
		return value != 0
	case float64:
		return value != 0
	default:
		return false
	}
}

func isPartialOrInlineCrmRecord(record map[string]any, inlineFields ...string) bool {
	if isPartialCrmRecord(record) {
		return true
	}
	if util.ToUint64(record["id"]) == 0 {
		return false
	}
	allowed := map[string]bool{
		"id":         true,
		"_partial":   true,
		"updated_at": true,
	}
	for _, field := range inlineFields {
		allowed[field] = true
	}
	for field := range record {
		if !allowed[field] {
			return false
		}
	}
	return true
}

func trimCrmStringField(record map[string]any, field string, partial bool) {
	if partial {
		if _, exists := record[field]; !exists {
			return
		}
	}
	record[field] = util.ToStringTrimmed(record[field])
}

func panicCrmField(field string, message string) {
	panic(frontaction.NewFieldError(field, message))
}

func contextFromServer(c *server.Context) context.Context {
	if c == nil {
		return context.Background()
	}
	return c.Context()
}
