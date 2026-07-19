package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

var workLeadExportColumns = []struct {
	Name string
	Key  string
}{
	{Name: "线索编号", Key: "code"},
	{Name: "姓名", Key: "name"},
	{Name: "手机号", Key: "phone"},
	{Name: "微信号", Key: "wechat"},
	{Name: "来源", Key: "source_name"},
	{Name: "渠道", Key: "channel_name"},
	{Name: "外部线索ID", Key: "external_id"},
	{Name: "城市", Key: "city"},
	{Name: "初始诉求", Key: "initial_need"},
	{Name: "状态", Key: "status_name"},
	{Name: "MKT负责人", Key: "owner_staff_name"},
	{Name: "录入时间", Key: "created_at"},
}

func (WorkService) LeadExport(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	workflow := workflowForSubject(ctx, firstUint64(payload, "workflow_id", "workflowId"), crmmodel.WorkflowSubjectLead)
	if workflow == nil || !canAccessWorkflow(ctx, staff, workflow) {
		return nil, fmt.Errorf("当前账号没有线索导出权限")
	}
	targets, err := workLeadVisibleTargets(ctx, staff, workflow, payload)
	if err != nil {
		return nil, err
	}
	rows := workLeadRows(ctx, staff, targets, false)
	content, err := buildWorkLeadCSV(rows)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"filename": fmt.Sprintf("线索导出-%s.csv", time.Now().Format("20060102-150405")),
		"content":  content,
		"total":    len(rows),
	}, nil
}

func buildWorkLeadCSV(rows []map[string]any) (string, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buffer)
	header := make([]string, 0, len(workLeadExportColumns))
	for _, column := range workLeadExportColumns {
		header = append(header, column.Name)
	}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("生成线索导出表头失败：%w", err)
	}
	for _, row := range rows {
		values := make([]string, 0, len(workLeadExportColumns))
		for _, column := range workLeadExportColumns {
			values = append(values, workLeadExportValue(row[column.Key]))
		}
		if err := writer.Write(values); err != nil {
			return "", fmt.Errorf("生成线索导出内容失败：%w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("生成线索导出文件失败：%w", err)
	}
	return buffer.String(), nil
}

func workLeadExportValue(value any) string {
	text := ""
	switch current := value.(type) {
	case time.Time:
		text = current.Format("2006-01-02 15:04:05")
	case *time.Time:
		if current != nil {
			text = current.Format("2006-01-02 15:04:05")
		}
	default:
		text = inputText(value)
	}
	return safeWorkLeadCSVValue(text)
}

func safeWorkLeadCSVValue(value string) string {
	if value == "" {
		return ""
	}
	for _, prefix := range []string{"=", "+", "-", "@", "\t", "\r"} {
		if strings.HasPrefix(value, prefix) {
			return "'" + value
		}
	}
	return value
}
