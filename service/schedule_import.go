package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

type CustomerFollowScheduleSyncResult struct {
	Applied          bool
	TemplateID       uint64
	FieldID          uint64
	ScannedCustomers int
	CreateCount      int
	UpdateCount      int
	SkippedEmpty     int
	Unassigned       []uint64
}

func SyncExistingCustomerFollowSchedules(ctx context.Context, apply bool) (CustomerFollowScheduleSyncResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	templateID, fieldID, err := legacyCustomerFollowField(ctx)
	if err != nil {
		return CustomerFollowScheduleSyncResult{}, err
	}
	result := CustomerFollowScheduleSyncResult{
		Applied:    apply,
		TemplateID: templateID,
		FieldID:    fieldID,
	}
	run := func(runCtx context.Context) error {
		seenCustomers := map[uint64]bool{}
		records := crmmodel.NewDataRecordModel().Select(runCtx, map[string]any{
			"data_template_id":     templateID,
			"asset_id":             uint64(0),
			"workflow_instance_id": uint64(0),
			"customer_product_id":  uint64(0),
			"status":               crmmodel.StatusEnabled,
		}, map[string]any{"order": "id desc"})
		for _, record := range records {
			if record == nil || record.CustomerID == 0 || seenCustomers[record.CustomerID] {
				continue
			}
			seenCustomers[record.CustomerID] = true
			result.ScannedCustomers++
			value, exists := mapFromAny(record.RecordJSON)[fmt.Sprintf("%d", fieldID)]
			if !exists || emptyWorkFieldValue(value) {
				result.SkippedEmpty++
				continue
			}
			startAt, parseErr := parseScheduleTime(value)
			if parseErr != nil {
				return fmt.Errorf("客户 %d 的跟进时间无效：%w", record.CustomerID, parseErr)
			}
			existing := findPendingCustomerFollowEvent(runCtx, record.CustomerID)
			instance := crmmodel.NewWorkflowInstanceModel().Find(runCtx, map[string]any{
				"customer_id": record.CustomerID,
				"status":      crmmodel.ProgressStatusActive,
			}, map[string]any{"order": "updated_at desc,id desc"})
			ownerStaffID := uint64(0)
			ownerDepartmentID := uint64(0)
			workflowInstanceID := uint64(0)
			if instance != nil {
				ownerStaffID = instance.OwnerStaffID
				ownerDepartmentID = instance.OwnerDepartmentID
				workflowInstanceID = instance.ID
			}
			if ownerStaffID == 0 && existing != nil {
				ownerStaffID = existing.OwnerStaffID
				if owner := crmmodel.NewStaffModel().Find(runCtx, map[string]any{"id": ownerStaffID}); owner != nil {
					ownerDepartmentID = owner.DepartmentID
				}
			}
			if ownerStaffID == 0 {
				result.Unassigned = append(result.Unassigned, record.CustomerID)
				continue
			}
			if existing == nil {
				result.CreateCount++
			} else {
				result.UpdateCount++
			}
			if !apply {
				continue
			}
			endAt := startAt.Add(defaultScheduleDuration)
			reminderMinutes := defaultScheduleReminderMinutes
			if existing != nil {
				duration := existing.EndAt.Sub(existing.StartAt)
				if duration > 0 {
					endAt = startAt.Add(duration)
				}
				reminderMinutes = existing.ReminderMinutes
			}
			if _, arrangeErr := arrangeCustomerFollow(runCtx, scheduleArrangeInput{
				CustomerID:               record.CustomerID,
				OwnerStaffID:             ownerStaffID,
				OperatorStaffID:          ownerStaffID,
				OperatorDepartmentID:     ownerDepartmentID,
				SourceWorkflowInstanceID: workflowInstanceID,
				Title:                    customerFollowDefaultTitle(runCtx, record.CustomerID),
				StartAt:                  startAt,
				EndAt:                    endAt,
				ReminderMinutes:          reminderMinutes,
				Source:                   crmmodel.ScheduleSourceWorkForm,
				SkipOperation:            true,
			}); arrangeErr != nil {
				return arrangeErr
			}
		}
		sort.Slice(result.Unassigned, func(i, j int) bool { return result.Unassigned[i] < result.Unassigned[j] })
		return nil
	}
	if !apply {
		return result, run(ctx)
	}
	err = orm.Transaction(ctx, run)
	return result, err
}

func legacyCustomerFollowField(ctx context.Context) (uint64, uint64, error) {
	fields := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"field_key":  "next_follow_at",
		"field_type": "datetime",
		"status":     crmmodel.StatusEnabled,
	})
	for _, field := range fields {
		if field == nil {
			continue
		}
		template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
			"id":      field.DataTemplateID,
			"cate_id": crmmodel.CustomerDataTemplateCateID,
			"status":  crmmodel.StatusEnabled,
		})
		if template != nil {
			return template.ID, field.ID, nil
		}
	}
	return 0, 0, fmt.Errorf("未找到旧版客户下次跟进时间字段")
}
