package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

type FeishuService struct{}

type feishuMessageResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func NewFeishuService() FeishuService {
	return FeishuService{}
}

func (FeishuService) SendTestMessage(ctx context.Context, staffID uint64) (map[string]any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if staffID == 0 {
		return nil, fmt.Errorf("请选择测试人员")
	}
	staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"id":     staffID,
		"status": crmmodel.StatusEnabled,
	})
	if staff == nil {
		return nil, fmt.Errorf("测试人员不存在或已停用")
	}
	openID := strings.TrimSpace(staff.FeishuOpenID)
	if openID == "" {
		return nil, fmt.Errorf("该人员尚未绑定飞书，请先使用飞书登录工作台")
	}
	token, err := fetchWorkFeishuAppAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	sentAt := time.Now()
	message := strings.Join([]string{
		"CRM 飞书提醒测试",
		"接收人：" + staff.Name,
		"时间：" + sentAt.In(scheduleLocation()).Format("2006-01-02 15:04:05"),
		"飞书提醒配置正常。",
	}, "\n")
	deliveryKey := fmt.Sprintf("crm-test-%x", sentAt.UnixNano())
	if err := sendFeishuTextMessage(ctx, token, openID, deliveryKey, message); err != nil {
		return nil, fmt.Errorf("飞书测试消息发送失败：%w", err)
	}
	return map[string]any{
		"staff_id":   staff.ID,
		"staff_name": staff.Name,
		"sent_at":    sentAt.In(scheduleLocation()).Format(time.RFC3339),
	}, nil
}

func sendFeishuTextMessage(ctx context.Context, token string, openID string, deliveryKey string, text string) error {
	openID = strings.TrimSpace(openID)
	if openID == "" {
		return fmt.Errorf("飞书 OpenID 不能为空")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("飞书消息内容不能为空")
	}
	content, err := json.Marshal(map[string]any{"text": text})
	if err != nil {
		return err
	}
	var response feishuMessageResponse
	path := "/im/v1/messages?receive_id_type=" + url.QueryEscape("open_id")
	if err := postFeishuJSON(ctx, path, map[string]any{
		"receive_id": openID,
		"msg_type":   "text",
		"content":    string(content),
		"uuid":       deliveryKey,
	}, token, &response); err != nil {
		return err
	}
	if response.Code != 0 {
		return fmt.Errorf("飞书接口返回错误：%s", fallbackFeishuMessage(response.Msg))
	}
	return nil
}
