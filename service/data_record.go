package service

import (
	"context"
	"fmt"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

type DataRecordService struct{}

func NewDataRecordService() DataRecordService {
	return DataRecordService{}
}

func (DataRecordService) Section(ctx context.Context, customerID uint64, assetID uint64, workflowInstanceID ...uint64) (map[string]any, error) {
	filter := map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	}
	instanceID := firstOptionalUint64(workflowInstanceID)
	if instanceID > 0 {
		filter["workflow_instance_id"] = instanceID
	}
	return map[string]any{
		"customer_id":          customerID,
		"asset_id":             assetID,
		"workflow_instance_id": instanceID,
		"sections":             crmmodel.NewDataRecordModel().SelectMap(ctx, filter),
	}, nil
}

func (DataRecordService) Save(ctx context.Context, payload map[string]any) (map[string]any, error) {
	templateID := firstUint64(payload, "data_template_id", "dataTemplateId")
	if templateID == 0 {
		return nil, fmt.Errorf("数据模板不能为空")
	}
	ownership, err := resolveDataRecordOwnership(ctx, workDataOwnership{
		CustomerID:         firstUint64(payload, "customer_id", "customerId"),
		AssetID:            firstUint64(payload, "asset_id", "assetId"),
		WorkflowInstanceID: firstUint64(payload, "workflow_instance_id", "workflowInstanceId"),
		CustomerProductID:  firstUint64(payload, "customer_product_id", "customerProductId"),
	}, templateID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	record := map[string]any{
		"customer_id":          ownership.CustomerID,
		"asset_id":             ownership.AssetID,
		"workflow_instance_id": ownership.WorkflowInstanceID,
		"customer_product_id":  ownership.CustomerProductID,
		"data_template_id":     templateID,
		"task_id":              firstUint64(payload, "task_id", "taskId"),
		"operation_log_id":     firstUint64(payload, "operation_log_id", "operationLogId"),
		"record_json":          firstText(payload, "record_json", "recordJSON"),
		"summary":              firstText(payload, "summary"),
		"status":               crmmodel.StatusEnabled,
		"sort":                 inputInt(payload["sort"]),
		"updated_at":           now,
	}
	if record["record_json"] == "" {
		record["record_json"] = "{}"
	}
	if record["sort"] == 0 {
		record["sort"] = 100
	}
	id := firstUint64(payload, "id")
	model := crmmodel.NewDataRecordModel()
	if id > 0 {
		if model.Update(ctx, map[string]any{"id": id}, record) == 0 {
			return nil, fmt.Errorf("数据记录不存在或已变化")
		}
	} else {
		record["created_at"] = now
		id = uint64(model.Insert(ctx, record))
		if id == 0 {
			return nil, fmt.Errorf("数据记录保存失败")
		}
	}
	return map[string]any{"id": id, "saved": true}, nil
}

func resolveDataRecordOwnership(ctx context.Context, ownership workDataOwnership, templateID uint64) (workDataOwnership, error) {
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
		"id":     templateID,
		"status": crmmodel.StatusEnabled,
	})
	if template == nil {
		return workDataOwnership{}, fmt.Errorf("数据模板不存在或已停用")
	}
	if ownership.CustomerID == 0 || crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": ownership.CustomerID}) == nil {
		return workDataOwnership{}, fmt.Errorf("客户不存在")
	}
	if template.CateID == crmmodel.CustomerAssetDataTemplateCateID {
		if ownership.AssetID == 0 || crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{
			"id":          ownership.AssetID,
			"customer_id": ownership.CustomerID,
		}) == nil {
			return workDataOwnership{}, fmt.Errorf("客户资产不存在")
		}
		return workDataOwnership{CustomerID: ownership.CustomerID, AssetID: ownership.AssetID}, nil
	}
	return workDataOwnership{CustomerID: ownership.CustomerID}, nil
}
