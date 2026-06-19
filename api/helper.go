package api

import (
	"strings"

	"github.com/shemic/dever/server"

	frontstream "github.com/dever-package/front/service/stream"
)

func bindBody(c *server.Context) (map[string]any, error) {
	body := map[string]any{}
	if err := c.BindJSON(&body); err != nil {
		return nil, err
	}
	return body, nil
}

func crmJSON(c *server.Context, data any, err error) error {
	if err != nil {
		return c.JSONPayload(200, map[string]any{
			"status": 2,
			"data":   map[string]any{},
			"msg":    err.Error(),
		})
	}
	return c.JSONPayload(200, map[string]any{
		"status": 1,
		"data":   data,
		"msg":    "",
	})
}

func uint64FromInput(value any) uint64 {
	return uint64(frontstream.InputInt64(value, 0))
}

func uint64FromBody(body map[string]any, keys ...string) uint64 {
	for _, key := range keys {
		if value := uint64FromInput(body[key]); value > 0 {
			return value
		}
	}
	return 0
}

func textFromBody(body map[string]any, keys ...string) string {
	for _, key := range keys {
		if text := strings.TrimSpace(frontstream.InputText(body[key])); text != "" {
			return text
		}
	}
	return ""
}

func boolFromBody(body map[string]any, keys ...string) bool {
	for _, key := range keys {
		switch strings.ToLower(strings.TrimSpace(frontstream.InputText(body[key]))) {
		case "1", "true", "yes", "y", "on":
			return true
		}
	}
	return false
}
