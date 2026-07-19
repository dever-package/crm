package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

type CustomerTagSelection struct {
	LevelID   uint64
	LevelName string
	TagIDs    []uint64
	TagNames  []string
}

func CustomerTagOptions(ctx context.Context) []map[string]any {
	levels := enabledCustomerTagLevels(ctx)
	rows := crmmodel.NewCustomerTagModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	})
	result := make([]map[string]any, 0, len(rows))
	for _, tag := range rows {
		if tag == nil {
			continue
		}
		level := levels[tag.LevelID]
		if level == nil {
			continue
		}
		result = append(result, map[string]any{
			"id":         tag.ID,
			"name":       tag.Name,
			"value":      fmt.Sprintf("%d", tag.ID),
			"level_id":   level.ID,
			"level_name": level.Name,
			"level_sort": level.Sort,
			"sort":       tag.Sort,
		})
	}
	return result
}

func CustomerTagIDs(ctx context.Context, customerID uint64) []uint64 {
	if customerID == 0 {
		return []uint64{}
	}
	relations := crmmodel.NewCustomerTagRelationModel().Select(ctx, map[string]any{
		"customer_id": customerID,
	})
	selected := make(map[uint64]bool, len(relations))
	for _, relation := range relations {
		if relation != nil && relation.TagID > 0 {
			selected[relation.TagID] = true
		}
	}
	result := make([]uint64, 0, len(selected))
	levels := enabledCustomerTagLevels(ctx)
	for _, tag := range crmmodel.NewCustomerTagModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}) {
		if tag != nil && selected[tag.ID] && levels[tag.LevelID] != nil {
			result = append(result, tag.ID)
			break
		}
	}
	return result
}

func CustomerTagsNeedSync(ctx context.Context, customerID uint64, raw any) (bool, error) {
	selection, err := ResolveCustomerTagSelection(ctx, raw)
	if err != nil {
		return false, err
	}
	currentIDs := CustomerTagIDs(ctx, customerID)
	currentRelationCount := crmmodel.NewCustomerTagRelationModel().Count(ctx, map[string]any{
		"customer_id": customerID,
	})
	if currentRelationCount > int64(len(currentIDs)) {
		return true, nil
	}
	if len(selection.TagIDs) == 0 && len(currentIDs) == 0 {
		return false, nil
	}
	if !sameCustomerTagIDs(selection.TagIDs, currentIDs) {
		return true, nil
	}
	customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
	return customer != nil && (customer.LevelID != selection.LevelID || customer.Tags != strings.Join(selection.TagNames, ",")), nil
}

func ResolveCustomerTagSelection(ctx context.Context, raw any) (*CustomerTagSelection, error) {
	requested, valid := customerTagIDsFromInput(raw)
	if !valid {
		return nil, fmt.Errorf("客户标签不存在或已停用")
	}
	selection := &CustomerTagSelection{
		TagIDs:   []uint64{},
		TagNames: []string{},
	}
	if len(requested) == 0 {
		return selection, nil
	}
	if len(requested) > 1 {
		return nil, fmt.Errorf("客户标签只能选择一个")
	}

	requestedLookup := make(map[uint64]bool, len(requested))
	for _, tagID := range requested {
		requestedLookup[tagID] = true
	}
	levels := enabledCustomerTagLevels(ctx)
	for _, tag := range crmmodel.NewCustomerTagModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}) {
		if tag == nil || !requestedLookup[tag.ID] {
			continue
		}
		level := levels[tag.LevelID]
		if level == nil {
			return nil, fmt.Errorf("客户标签所属等级不存在或已停用")
		}
		selection.LevelID = level.ID
		selection.LevelName = level.Name
		selection.TagIDs = append(selection.TagIDs, tag.ID)
		selection.TagNames = append(selection.TagNames, tag.Name)
		delete(requestedLookup, tag.ID)
	}
	if len(requestedLookup) > 0 {
		return nil, fmt.Errorf("客户标签不存在或已停用")
	}
	return selection, nil
}

func SyncCustomerTags(ctx context.Context, customerID uint64, raw any) (*CustomerTagSelection, error) {
	var selection *CustomerTagSelection
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		if customerID == 0 || crmmodel.NewCustomerModel().Find(txCtx, map[string]any{"id": customerID}) == nil {
			return fmt.Errorf("客户不存在")
		}
		resolved, err := ResolveCustomerTagSelection(txCtx, raw)
		if err != nil {
			return err
		}
		selection = resolved

		relationModel := crmmodel.NewCustomerTagRelationModel()
		relationModel.Delete(txCtx, map[string]any{"customer_id": customerID})
		now := time.Now()
		for _, tagID := range selection.TagIDs {
			relationModel.Insert(txCtx, map[string]any{
				"customer_id": customerID,
				"tag_id":      tagID,
				"created_at":  now,
			})
		}

		updates := map[string]any{
			"tags":       strings.Join(selection.TagNames, ","),
			"updated_at": now,
		}
		if selection.LevelID > 0 {
			updates["level_id"] = selection.LevelID
		}
		crmmodel.NewCustomerModel().Update(txCtx, map[string]any{"id": customerID}, updates)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return selection, nil
}

func customerTagIDsFromInput(raw any) ([]uint64, bool) {
	switch values := raw.(type) {
	case nil:
		return []uint64{}, true
	case []uint64:
		for _, id := range values {
			if id == 0 {
				return nil, false
			}
		}
		return uniqueUint64Values(values), true
	case []string:
		return customerTagIDsFromValues(values)
	case []any:
		return customerTagIDsFromValues(values)
	case string:
		text := strings.TrimSpace(values)
		if text == "" {
			return []uint64{}, true
		}
		var decoded []any
		if err := json.Unmarshal([]byte(text), &decoded); err == nil {
			return customerTagIDsFromValues(decoded)
		}
		if id := inputUint64(text); id > 0 {
			return []uint64{id}, true
		}
		return nil, false
	default:
		if id := inputUint64(raw); id > 0 {
			return []uint64{id}, true
		}
		return nil, false
	}
}

func customerTagIDsFromValues[T any](values []T) ([]uint64, bool) {
	requested := make([]uint64, 0, len(values))
	for _, value := range values {
		id := inputUint64(value)
		if id == 0 {
			return nil, false
		}
		requested = append(requested, id)
	}
	return uniqueUint64Values(requested), true
}

func sameCustomerTagIDs(left []uint64, right []uint64) bool {
	if len(left) != len(right) {
		return false
	}
	rightSet := make(map[uint64]bool, len(right))
	for _, id := range right {
		rightSet[id] = true
	}
	for _, id := range left {
		if !rightSet[id] {
			return false
		}
	}
	return true
}

func enabledCustomerTagLevels(ctx context.Context) map[uint64]*crmmodel.CustomerLevel {
	result := map[uint64]*crmmodel.CustomerLevel{}
	for _, level := range crmmodel.NewCustomerLevelModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}) {
		if level != nil {
			result[level.ID] = level
		}
	}
	return result
}
