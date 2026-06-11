package model

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

const maxCustomerCodeAttempts = 30

func GenerateUniqueCustomerCode(ctx context.Context) (string, error) {
	model := NewCustomerModel()
	datePrefix := time.Now().Format("20060102")
	for i := 0; i < maxCustomerCodeAttempts; i++ {
		code := datePrefix + randomSixDigits()
		if model.Find(ctx, map[string]any{"code": code}) == nil {
			return code, nil
		}
	}
	return "", fmt.Errorf("客户编号生成失败，请重试")
}

func randomSixDigits() string {
	value, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		now := time.Now().UnixNano() % 1000000
		return fmt.Sprintf("%06d", now)
	}
	return fmt.Sprintf("%06d", value.Int64())
}
