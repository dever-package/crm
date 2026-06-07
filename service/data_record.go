package service

import (
	"context"
	"fmt"
	"time"

	crmmodel "my/package/crm/model"
)

type DataRecordService struct{}

func NewDataRecordService() DataRecordService {
	return DataRecordService{}
}

func (DataRecordService) Section(ctx context.Context, resourceID uint64) (map[string]any, error) {
	return map[string]any{
		"resource_id": resourceID,
		"sections":    crmmodel.NewDataRecordModel().SelectMap(ctx, map[string]any{"resource_id": resourceID}),
	}, nil
}

func (DataRecordService) Save(ctx context.Context, payload map[string]any) (map[string]any, error) {
	resourceID := firstUint64(payload, "resource_id", "resourceId")
	templateID := firstUint64(payload, "data_template_id", "dataTemplateId")
	if resourceID == 0 || templateID == 0 {
		return nil, fmt.Errorf("客户资产和数据模板不能为空")
	}
	record := map[string]any{
		"resource_id":      resourceID,
		"data_template_id": templateID,
		"task_id":          firstUint64(payload, "task_id", "taskId"),
		"record_json":      firstText(payload, "record_json", "recordJSON"),
		"summary":          firstText(payload, "summary"),
		"status":           crmmodel.StatusEnabled,
		"sort":             inputInt(payload["sort"]),
		"updated_at":       time.Now(),
	}
	if record["record_json"] == "" {
		record["record_json"] = "{}"
	}
	if record["sort"] == 0 {
		record["sort"] = 100
	}
	id := firstUint64(payload, "id")
	if id > 0 {
		crmmodel.NewDataRecordModel().Update(ctx, map[string]any{"id": id}, record)
	} else {
		record["created_at"] = time.Now()
		id = uint64(crmmodel.NewDataRecordModel().Insert(ctx, record))
	}
	return map[string]any{
		"id":    id,
		"saved": true,
	}, nil
}
