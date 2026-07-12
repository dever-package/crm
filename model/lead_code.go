package model

import (
	"context"
	"fmt"
	"time"
)

const maxLeadCodeAttempts = 30

func GenerateUniqueLeadCode(ctx context.Context) (string, error) {
	model := NewLeadModel()
	datePrefix := time.Now().Format("20060102")
	for i := 0; i < maxLeadCodeAttempts; i++ {
		code := "L" + datePrefix + randomSixDigits()
		if model.Find(ctx, map[string]any{"code": code}) == nil {
			return code, nil
		}
	}
	return "", fmt.Errorf("线索编号生成失败，请重试")
}
