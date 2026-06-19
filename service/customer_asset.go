package service

import (
	"context"
	"fmt"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

type CustomerAssetService struct{}

func NewCustomerAssetService() CustomerAssetService {
	return CustomerAssetService{}
}

func (CustomerAssetService) Create(ctx context.Context, payload map[string]any) (map[string]any, error) {
	customerID := firstUint64(payload, "customer_id", "customerId")
	if customerID == 0 {
		return nil, fmt.Errorf("客户不能为空")
	}
	assetName := firstText(payload, "asset_name", "assetName")
	if assetName == "" {
		return nil, fmt.Errorf("资产名称不能为空")
	}
	assetNo := firstText(payload, "asset_no", "assetNo")
	if assetNo == "" {
		assetNo = defaultAssetNo()
	}
	assetStatusID := firstUint64(payload, "asset_status_id", "assetStatusId")
	if assetStatusID == 0 {
		assetStatusID = crmmodel.DefaultAssetStatusID
	}
	now := time.Now()
	assetID := uint64(crmmodel.NewCustomerAssetModel().Insert(ctx, map[string]any{
		"asset_no":        assetNo,
		"asset_name":      assetName,
		"customer_id":     customerID,
		"asset_status_id": assetStatusID,
		"remark":          firstText(payload, "remark"),
		"created_at":      now,
		"updated_at":      now,
	}))
	if assetID == 0 {
		return nil, fmt.Errorf("客户资产创建失败")
	}
	return map[string]any{"id": assetID}, nil
}

func (CustomerAssetService) Detail(ctx context.Context, assetID uint64) (map[string]any, error) {
	if assetID == 0 {
		return nil, fmt.Errorf("客户资产不能为空")
	}
	asset := crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{"id": assetID})
	return map[string]any{
		"id":            assetID,
		"asset":         asset,
		"data_sections": crmmodel.NewDataRecordModel().SelectMap(ctx, map[string]any{"asset_id": assetID}),
		"operations":    crmmodel.NewOperationLogModel().SelectMap(ctx, map[string]any{"asset_id": assetID}),
	}, nil
}
