package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

func SyncCandidateCustomerProducts(ctx context.Context, instanceID uint64, productCodes []string) error {
	entry, err := workflowInstanceByID(ctx, instanceID)
	if err != nil {
		return err
	}
	if entry.CustomerProductID != 0 {
		return fmt.Errorf("只有入口流程可以生成候选产品")
	}

	productsByCode := map[string]*crmmodel.Product{}
	for _, product := range crmmodel.NewProductModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled}) {
		if product == nil {
			continue
		}
		code := normalizeProductCode(product.Code)
		if code != "" {
			productsByCode[code] = product
		}
	}
	selectedProducts := make([]*crmmodel.Product, 0, len(productCodes))
	selectedIDs := map[uint64]bool{}
	for _, code := range uniqueProductCodes(productCodes) {
		product := productsByCode[code]
		if product == nil {
			return fmt.Errorf("候选产品编码不存在或已停用：%s", code)
		}
		selectedProducts = append(selectedProducts, product)
		selectedIDs[product.ID] = true
	}

	existingRows := entryCustomerProducts(ctx, entry.ID)
	existingByProduct := customerProductsByProductID(existingRows)
	for _, product := range selectedProducts {
		current := existingByProduct[product.ID]
		if current == nil {
			if err := createEntryCustomerProduct(ctx, entry, product.ID, crmmodel.CustomerProductStatusCandidate); err != nil {
				return err
			}
			continue
		}
		if current.Status == crmmodel.CustomerProductStatusLost {
			if err := updateCustomerProductStatus(ctx, current, crmmodel.CustomerProductStatusCandidate); err != nil {
				return err
			}
		}
	}
	for _, current := range existingRows {
		if current == nil || current.Status != crmmodel.CustomerProductStatusCandidate || selectedIDs[current.ProductID] {
			continue
		}
		if err := updateCustomerProductStatus(ctx, current, crmmodel.CustomerProductStatusLost); err != nil {
			return err
		}
	}
	return nil
}

func SyncConfirmedCustomerProducts(ctx context.Context, instanceID uint64, productIDs []uint64) ([]*crmmodel.CustomerProduct, error) {
	entry, err := workflowInstanceByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if entry.CustomerProductID != 0 {
		return nil, fmt.Errorf("只有入口流程可以确认产品")
	}
	selectedIDs := uniqueProductIDs(productIDs)
	for _, productID := range selectedIDs {
		product := crmmodel.NewProductModel().Find(ctx, map[string]any{
			"id":     productID,
			"status": crmmodel.StatusEnabled,
		})
		if product == nil {
			return nil, fmt.Errorf("所选产品不存在或已停用")
		}
	}

	model := crmmodel.NewCustomerProductModel()
	existingRows := entryCustomerProducts(ctx, entry.ID)
	existingByProduct := customerProductsByProductID(existingRows)
	selected := make(map[uint64]bool, len(selectedIDs))
	for _, productID := range selectedIDs {
		selected[productID] = true
		current := existingByProduct[productID]
		if current == nil {
			if err := createEntryCustomerProduct(ctx, entry, productID, crmmodel.CustomerProductStatusConfirmed); err != nil {
				return nil, err
			}
			continue
		}
		if current.Status == crmmodel.CustomerProductStatusLost || current.Status == crmmodel.CustomerProductStatusCandidate {
			if err := updateCustomerProductStatus(ctx, current, crmmodel.CustomerProductStatusConfirmed); err != nil {
				return nil, err
			}
		}
	}
	for _, current := range existingRows {
		if current == nil || selected[current.ProductID] || current.Status == crmmodel.CustomerProductStatusLost {
			continue
		}
		if current.Status == crmmodel.CustomerProductStatusProcessing || current.Status == crmmodel.CustomerProductStatusCompleted {
			return nil, fmt.Errorf("产品已进入处理流程，不能取消")
		}
		if err := updateCustomerProductStatus(ctx, current, crmmodel.CustomerProductStatusLost); err != nil {
			return nil, err
		}
	}

	result := make([]*crmmodel.CustomerProduct, 0, len(selectedIDs))
	for _, productID := range selectedIDs {
		current := model.Find(ctx, map[string]any{
			"source_workflow_instance_id": entry.ID,
			"product_id":                  productID,
		})
		if current == nil {
			return nil, fmt.Errorf("客户产品同步失败")
		}
		result = append(result, current)
	}
	return result, nil
}

func entryCustomerProducts(ctx context.Context, workflowInstanceID uint64) []*crmmodel.CustomerProduct {
	return crmmodel.NewCustomerProductModel().Select(ctx, map[string]any{
		"source_workflow_instance_id": workflowInstanceID,
	})
}

func customerProductsByProductID(rows []*crmmodel.CustomerProduct) map[uint64]*crmmodel.CustomerProduct {
	result := make(map[uint64]*crmmodel.CustomerProduct, len(rows))
	for _, customerProduct := range rows {
		if customerProduct != nil {
			result[customerProduct.ProductID] = customerProduct
		}
	}
	return result
}

func createEntryCustomerProduct(ctx context.Context, entry *crmmodel.WorkflowInstance, productID uint64, status string) error {
	if entry == nil || entry.ID == 0 || productID == 0 {
		return fmt.Errorf("客户产品归属无效")
	}
	now := time.Now()
	if crmmodel.NewCustomerProductModel().Insert(ctx, map[string]any{
		"customer_id":                 entry.CustomerID,
		"asset_id":                    entry.AssetID,
		"product_id":                  productID,
		"source_workflow_instance_id": entry.ID,
		"status":                      status,
		"created_at":                  now,
		"updated_at":                  now,
	}) == 0 {
		return fmt.Errorf("客户产品创建失败")
	}
	return nil
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
			assetWorkflowSubject(customerProduct.CustomerID, customerProduct.AssetID, customerProduct.ID),
			product.ServiceWorkflowID,
			0,
		)
		if err != nil {
			return fmt.Errorf("产品“%s”签约后流程启动失败：%w", product.Name, err)
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

func uniqueProductCodes(productCodes []string) []string {
	seen := make(map[string]bool, len(productCodes))
	result := make([]string, 0, len(productCodes))
	for _, productCode := range productCodes {
		code := normalizeProductCode(productCode)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		result = append(result, code)
	}
	sort.Strings(result)
	return result
}

func normalizeProductCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
