package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func SyncConfirmedCustomerProducts(ctx context.Context, instanceID uint64, productIDs []uint64) ([]*crmmodel.CustomerProduct, error) {
	entry, err := workflowInstanceByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if entry.CustomerProductID != 0 {
		return nil, fmt.Errorf("只有入口流程可以确认产品")
	}
	selectedIDs := uniqueProductIDs(productIDs)
	products := make(map[uint64]*crmmodel.Product, len(selectedIDs))
	for _, productID := range selectedIDs {
		product := crmmodel.NewProductModel().Find(ctx, map[string]any{
			"id":     productID,
			"status": crmmodel.StatusEnabled,
		})
		if product == nil {
			return nil, fmt.Errorf("所选产品不存在或已停用")
		}
		products[productID] = product
	}

	model := crmmodel.NewCustomerProductModel()
	existingRows := model.Select(ctx, map[string]any{"source_workflow_instance_id": entry.ID})
	existingByProduct := make(map[uint64]*crmmodel.CustomerProduct, len(existingRows))
	for _, current := range existingRows {
		if current != nil {
			existingByProduct[current.ProductID] = current
		}
	}
	selected := make(map[uint64]bool, len(selectedIDs))
	now := time.Now()
	for _, productID := range selectedIDs {
		selected[productID] = true
		current := existingByProduct[productID]
		if current == nil {
			id := uint64(model.Insert(ctx, map[string]any{
				"customer_id":                 entry.CustomerID,
				"asset_id":                    entry.AssetID,
				"product_id":                  productID,
				"source_workflow_instance_id": entry.ID,
				"status":                      crmmodel.CustomerProductStatusConfirmed,
				"created_at":                  now,
				"updated_at":                  now,
			}))
			if id == 0 {
				return nil, fmt.Errorf("客户产品创建失败")
			}
			continue
		}
		if current.Status == crmmodel.CustomerProductStatusLost || current.Status == crmmodel.CustomerProductStatusCandidate {
			model.Update(ctx, map[string]any{"id": current.ID}, map[string]any{
				"status":     crmmodel.CustomerProductStatusConfirmed,
				"updated_at": now,
			})
		}
	}
	for _, current := range existingRows {
		if current == nil || selected[current.ProductID] || current.Status == crmmodel.CustomerProductStatusLost {
			continue
		}
		if current.Status == crmmodel.CustomerProductStatusProcessing || current.Status == crmmodel.CustomerProductStatusCompleted {
			return nil, fmt.Errorf("产品已进入处理流程，不能取消")
		}
		model.Update(ctx, map[string]any{"id": current.ID}, map[string]any{
			"status":     crmmodel.CustomerProductStatusLost,
			"updated_at": now,
		})
	}

	result := make([]*crmmodel.CustomerProduct, 0, len(selectedIDs))
	for _, productID := range selectedIDs {
		current := model.Find(ctx, map[string]any{
			"source_workflow_instance_id": entry.ID,
			"product_id":                  productID,
		})
		if current == nil || products[productID] == nil {
			return nil, fmt.Errorf("客户产品同步失败")
		}
		result = append(result, current)
	}
	return result, nil
}

func StartConfirmedProductWorkflows(ctx context.Context, entry *crmmodel.WorkflowInstance) error {
	if entry == nil || entry.ID == 0 || entry.CustomerProductID != 0 {
		return fmt.Errorf("入口流程实例无效")
	}
	rows := crmmodel.NewCustomerProductModel().Select(ctx, map[string]any{
		"source_workflow_instance_id": entry.ID,
		"status":                      crmmodel.CustomerProductStatusConfirmed,
	}, map[string]any{"order": "id asc"})
	for _, customerProduct := range rows {
		if customerProduct == nil {
			continue
		}
		product := crmmodel.NewProductModel().Find(ctx, map[string]any{
			"id":     customerProduct.ProductID,
			"status": crmmodel.StatusEnabled,
		})
		if product == nil {
			return fmt.Errorf("已确认产品不存在或已停用")
		}
		if product.ServiceWorkflowID == 0 {
			if err := updateCustomerProductStatus(ctx, customerProduct, crmmodel.CustomerProductStatusCompleted); err != nil {
				return err
			}
			continue
		}
		instance, err := startWorkflowInstance(
			ctx,
			customerProduct.CustomerID,
			customerProduct.AssetID,
			customerProduct.ID,
			product.ServiceWorkflowID,
			0,
		)
		if err != nil {
			return fmt.Errorf("产品“%s”服务流程启动失败：%w", product.Name, err)
		}
		targetStatus := crmmodel.CustomerProductStatusProcessing
		if instance.Status == crmmodel.ProgressStatusCompleted {
			targetStatus = crmmodel.CustomerProductStatusCompleted
		} else if instance.Status == crmmodel.ProgressStatusTerminated {
			targetStatus = crmmodel.CustomerProductStatusLost
		}
		if err := updateCustomerProductStatus(ctx, customerProduct, targetStatus); err != nil {
			return err
		}
	}
	return nil
}

func CompleteCustomerProductForInstance(ctx context.Context, instance *crmmodel.WorkflowInstance) error {
	if instance == nil || instance.CustomerProductID == 0 {
		return nil
	}
	customerProduct := crmmodel.NewCustomerProductModel().Find(ctx, map[string]any{"id": instance.CustomerProductID})
	if customerProduct == nil {
		return fmt.Errorf("流程关联的客户产品不存在")
	}
	return updateCustomerProductStatus(ctx, customerProduct, crmmodel.CustomerProductStatusCompleted)
}

func LoseCustomerProductForInstance(ctx context.Context, instance *crmmodel.WorkflowInstance) error {
	if instance == nil || instance.CustomerProductID == 0 {
		return nil
	}
	customerProduct := crmmodel.NewCustomerProductModel().Find(ctx, map[string]any{"id": instance.CustomerProductID})
	if customerProduct == nil {
		return fmt.Errorf("流程关联的客户产品不存在")
	}
	return updateCustomerProductStatus(ctx, customerProduct, crmmodel.CustomerProductStatusLost)
}

func updateCustomerProductStatus(ctx context.Context, customerProduct *crmmodel.CustomerProduct, status string) error {
	if customerProduct == nil || customerProduct.Status == status {
		return nil
	}
	if crmmodel.NewCustomerProductModel().Update(ctx, map[string]any{
		"id":     customerProduct.ID,
		"status": customerProduct.Status,
	}, map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}) == 0 {
		return fmt.Errorf("客户产品状态已变化，请刷新后重试")
	}
	customerProduct.Status = status
	return nil
}

func uniqueProductIDs(productIDs []uint64) []uint64 {
	seen := make(map[uint64]bool, len(productIDs))
	result := make([]uint64, 0, len(productIDs))
	for _, productID := range productIDs {
		if productID == 0 || seen[productID] {
			continue
		}
		seen[productID] = true
		result = append(result, productID)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
