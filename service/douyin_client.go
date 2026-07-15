package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	douyinAPIBase          = "https://open.douyin.com"
	douyinClientTokenPath  = "/oauth/client_token/"
	douyinClueQueryPath    = "/goodlife/v1/open_api/crm/clue/query/"
	douyinRequestTimeout   = 15 * time.Second
	douyinMaxRetryAttempts = 3
)

var douyinRetryableCodes = map[int]bool{
	2100001:  true,
	2100004:  true,
	2119002:  true,
	2119003:  true,
	2190002:  true,
	2190008:  true,
	45002002: true,
	5000001:  true,
}

type douyinCredentials struct {
	ClientKey         string
	ClientSecret      string
	AccountID         string
	RootLifeAccountID string
}

type douyinCluePage struct {
	Clues      []map[string]any
	PageNumber int
	PageSize   int
	PageTotal  int
	Total      int
}

type douyinRequestError struct {
	Message   string
	Code      int
	Retryable bool
}

func (e *douyinRequestError) Error() string {
	if e == nil {
		return "抖音接口请求失败"
	}
	return e.Message
}

var douyinTokenCache = struct {
	sync.Mutex
	Token     string
	ClientKey string
	ExpiresAt time.Time
}{}

func queryDouyinCluePage(
	ctx context.Context,
	credentials douyinCredentials,
	startTime string,
	endTime string,
	page int,
	pageSize int,
) (douyinCluePage, error) {
	var result douyinCluePage
	err := withDouyinRetry(ctx, func() error {
		token, err := getDouyinClientToken(ctx, credentials)
		if err != nil {
			return err
		}
		query := url.Values{}
		query.Set("account_id", credentials.AccountID)
		query.Set("start_time", startTime)
		query.Set("end_time", endTime)
		query.Set("page", strconv.Itoa(page))
		query.Set("page_size", strconv.Itoa(pageSize))

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			douyinAPIBase+douyinClueQueryPath+"?"+query.Encode(),
			nil,
		)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("access-token", token)
		if credentials.RootLifeAccountID != "" {
			req.Header.Set("Rpc-Transit-Life-Account", credentials.RootLifeAccountID)
		}

		payload, err := doDouyinRequest(req)
		if err != nil {
			if requestErr, ok := err.(*douyinRequestError); ok && (requestErr.Code == 2190002 || requestErr.Code == 2190008) {
				clearDouyinClientToken()
			}
			return err
		}
		data := douyinMap(payload["data"])
		pageData := douyinMap(data["page"])
		result = douyinCluePage{
			Clues:      douyinMapList(data["clue_data"]),
			PageNumber: douyinIntDefault(pageData["page_number"], page),
			PageSize:   douyinIntDefault(pageData["page_size"], pageSize),
			PageTotal:  douyinIntDefault(pageData["page_total"], 0),
			Total:      douyinIntDefault(pageData["total"], 0),
		}
		return nil
	})
	return result, err
}

func getDouyinClientToken(ctx context.Context, credentials douyinCredentials) (string, error) {
	douyinTokenCache.Lock()
	if douyinTokenCache.Token != "" &&
		douyinTokenCache.ClientKey == credentials.ClientKey &&
		time.Until(douyinTokenCache.ExpiresAt) > time.Minute {
		token := douyinTokenCache.Token
		douyinTokenCache.Unlock()
		return token, nil
	}
	douyinTokenCache.Unlock()

	body, err := json.Marshal(map[string]any{
		"grant_type":    "client_credential",
		"client_key":    credentials.ClientKey,
		"client_secret": credentials.ClientSecret,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, douyinAPIBase+douyinClientTokenPath, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	payload, err := doDouyinRequest(req)
	if err != nil {
		return "", err
	}
	data := douyinMap(payload["data"])
	token := strings.TrimSpace(fmt.Sprint(data["access_token"]))
	if token == "" || token == "<nil>" {
		return "", fmt.Errorf("抖音未返回 access_token")
	}
	expiresIn := douyinIntDefault(data["expires_in"], 7200)
	douyinTokenCache.Lock()
	douyinTokenCache.Token = token
	douyinTokenCache.ClientKey = credentials.ClientKey
	douyinTokenCache.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	douyinTokenCache.Unlock()
	return token, nil
}

func clearDouyinClientToken() {
	douyinTokenCache.Lock()
	douyinTokenCache.Token = ""
	douyinTokenCache.ClientKey = ""
	douyinTokenCache.ExpiresAt = time.Time{}
	douyinTokenCache.Unlock()
}

func doDouyinRequest(req *http.Request) (map[string]any, error) {
	client := &http.Client{Timeout: douyinRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &douyinRequestError{Message: err.Error(), Retryable: true}
	}
	defer resp.Body.Close()
	payload := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, &douyinRequestError{
			Message:   "抖音接口返回内容无法解析",
			Retryable: resp.StatusCode >= http.StatusInternalServerError,
		}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &douyinRequestError{
			Message:   fmt.Sprintf("抖音接口请求失败：%d", resp.StatusCode),
			Retryable: resp.StatusCode >= http.StatusInternalServerError,
		}
	}
	code := douyinResponseErrorCode(payload)
	if code == 0 {
		return payload, nil
	}
	return nil, &douyinRequestError{
		Message:   douyinResponseErrorMessage(payload),
		Code:      code,
		Retryable: douyinRetryableCodes[code],
	}
}

func withDouyinRetry(ctx context.Context, operation func() error) error {
	var lastErr error
	for attempt := 1; attempt <= douyinMaxRetryAttempts; attempt++ {
		if err := operation(); err != nil {
			lastErr = err
			requestErr, retryable := err.(*douyinRequestError)
			if !retryable || !requestErr.Retryable || attempt == douyinMaxRetryAttempts {
				return err
			}
			delay := time.Duration(1<<(attempt-1)) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		return nil
	}
	return lastErr
}

func douyinResponseErrorCode(payload map[string]any) int {
	if code := douyinIntDefault(douyinMap(payload["data"])["error_code"], 0); code != 0 {
		return code
	}
	return douyinIntDefault(douyinMap(payload["extra"])["error_code"], 0)
}

func douyinResponseErrorMessage(payload map[string]any) string {
	data := douyinMap(payload["data"])
	extra := douyinMap(payload["extra"])
	message := firstDouyinText(
		data["description"],
		extra["description"],
		extra["sub_description"],
		payload["message"],
	)
	if message == "" {
		message = "抖音接口请求失败"
	}
	if logID := firstDouyinText(extra["logid"]); logID != "" {
		message += "（logid: " + logID + "）"
	}
	return message
}

func douyinMap(value any) map[string]any {
	row, _ := value.(map[string]any)
	if row == nil {
		return map[string]any{}
	}
	return row
}

func douyinMapList(value any) []map[string]any {
	items, _ := value.([]any)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if row, ok := item.(map[string]any); ok && row != nil {
			result = append(result, row)
		}
	}
	return result
}

func douyinIntDefault(value any, fallback int) int {
	switch current := value.(type) {
	case float64:
		return int(current)
	case int:
		return current
	case int64:
		return int(current)
	case json.Number:
		if parsed, err := current.Int64(); err == nil {
			return int(parsed)
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(current)); err == nil {
			return parsed
		}
	}
	return fallback
}

func firstDouyinText(values ...any) string {
	for _, value := range values {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}
