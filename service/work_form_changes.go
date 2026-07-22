package service

import (
	"context"
	"sort"
	"strings"

	crmmodel "github.com/dever-package/crm/model"
)

const workFormChangeSnapshotType = "form_changes"

func buildWorkFormOperationSnapshot(
	ctx context.Context,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	formInput *workFormInput,
	values map[string]any,
) (map[string]any, bool) {
	changes := workFormInputChanges(ctx, todo, formInput)
	appendWorkTaskSystemChanges(&changes, ctx, todo, task, values)
	return map[string]any{
		"snapshot_type": workFormChangeSnapshotType,
		"changes":       changes,
	}, len(changes) > 0
}

func appendWorkTaskSystemChanges(
	changes *[]map[string]any,
	ctx context.Context,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	values map[string]any,
) {
	if task == nil {
		return
	}
	if task.CustomerFollowEnabled {
		before := any(nil)
		if todo != nil {
			if event := findPendingCustomerFollowEvent(ctx, todo.CustomerID); event != nil {
				before = event.StartAt.In(scheduleLocation()).Format(customerFollowTimeLayout)
			}
		}
		appendWorkFormChange(changes, workCustomerFollowKey, before, values[workCustomerFollowKey])
	}
	if task.MeetingEnabled {
		beforeValues := workMeetingEventValues(ctx, findWorkMeetingEvent(ctx, todo.WorkflowInstanceID, task.ID))
		for _, key := range []string{
			workMeetingStartFieldKey,
			workMeetingDurationFieldKey,
			workMeetingResourceFieldKey,
		} {
			if value, exists := values[key]; exists && !emptyWorkFieldValue(value) {
				appendWorkFormChange(changes, key, beforeValues[key], normalizeWorkMeetingAuditValue(key, value))
			}
		}
		if decision := workMeetingArrivalDecision(values); decision != "" {
			appendWorkFormChange(changes, workMeetingArrivalFieldKey, beforeValues[workMeetingArrivalFieldKey], decision)
			if decision == crmmodel.MeetingArrivalNoShow {
				appendWorkFormChange(changes, workMeetingNoShowReasonKey, beforeValues[workMeetingNoShowReasonKey], inputText(values[workMeetingNoShowReasonKey]))
			}
		}
	}
}

func workFormInputChanges(ctx context.Context, todo *crmmodel.WorkTodo, formInput *workFormInput) []map[string]any {
	if todo == nil || formInput == nil {
		return []map[string]any{}
	}
	changes := make([]map[string]any, 0)
	appendWorkMainFieldChanges(&changes, workLeadMainValues(ctx, todo.LeadID), formInput.leadFields)
	appendWorkMainFieldChanges(&changes, workCustomerMainValues(ctx, todo.CustomerID), formInput.customerFields)
	appendWorkMainFieldChanges(&changes, workAssetMainValues(ctx, todo.CustomerID, todo.AssetID), formInput.assetFields)
	appendWorkLeadDataChanges(&changes, ctx, todo.LeadID, formInput.leadDataRecords)
	appendWorkDataRecordChanges(&changes, ctx, workDataOwnership{CustomerID: todo.CustomerID}, formInput.customerDataRecords)
	appendWorkDataRecordChanges(&changes, ctx, workDataOwnership{
		CustomerID: todo.CustomerID,
		AssetID:    todo.AssetID,
	}, formInput.assetDataRecords)
	return changes
}

func workLeadMainValues(ctx context.Context, leadID uint64) map[string]any {
	if leadID == 0 {
		return map[string]any{}
	}
	return crmmodel.NewLeadModel().FindMap(ctx, map[string]any{"id": leadID})
}

func workCustomerMainValues(ctx context.Context, customerID uint64) map[string]any {
	if customerID == 0 {
		return map[string]any{}
	}
	return crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
}

func workAssetMainValues(ctx context.Context, customerID uint64, assetID uint64) map[string]any {
	if customerID == 0 || assetID == 0 {
		return map[string]any{}
	}
	return crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{
		"id":          assetID,
		"customer_id": customerID,
	})
}

func appendWorkMainFieldChanges(changes *[]map[string]any, current map[string]any, submitted map[string]any) {
	for _, field := range sortedMapKeys(submitted) {
		appendWorkFormChange(changes, "main:"+field, current[field], submitted[field])
	}
}

func appendWorkLeadDataChanges(
	changes *[]map[string]any,
	ctx context.Context,
	leadID uint64,
	records map[uint64]map[string]any,
) {
	if len(records) == 0 {
		return
	}
	lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": leadID})
	current := workLeadDataValues(lead)
	for _, templateID := range sortedWorkFormTemplateIDs(records) {
		for _, fieldID := range sortedMapKeys(records[templateID]) {
			key := "data:" + fieldID
			appendWorkFormChange(changes, key, current[key], records[templateID][fieldID])
		}
	}
}

func appendWorkDataRecordChanges(
	changes *[]map[string]any,
	ctx context.Context,
	ownership workDataOwnership,
	records map[uint64]map[string]any,
) {
	for _, templateID := range sortedWorkFormTemplateIDs(records) {
		current := map[string]any{}
		if record := crmmodel.NewDataRecordModel().Find(ctx, workDataRecordOwnershipFilter(ownership, templateID)); record != nil {
			current = mapFromAny(record.RecordJSON)
		}
		for _, fieldID := range sortedMapKeys(records[templateID]) {
			appendWorkFormChange(changes, "data:"+fieldID, current[fieldID], records[templateID][fieldID])
		}
	}
}

func appendWorkFormChange(changes *[]map[string]any, key string, before any, after any) {
	if strings.TrimSpace(key) == "" || workFormAuditValuesEqual(before, after) {
		return
	}
	*changes = append(*changes, map[string]any{
		"key":    key,
		"before": before,
		"after":  after,
	})
}

func sortedWorkFormTemplateIDs(records map[uint64]map[string]any) []uint64 {
	ids := make([]uint64, 0, len(records))
	for templateID := range records {
		ids = append(ids, templateID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func workFormAuditValuesEqual(before any, after any) bool {
	if emptyWorkFieldValue(before) || emptyWorkFieldValue(after) {
		return emptyWorkFieldValue(before) && emptyWorkFieldValue(after)
	}
	beforeList, beforeIsList := workFormAuditList(before)
	afterList, afterIsList := workFormAuditList(after)
	if beforeIsList || afterIsList {
		return beforeIsList && afterIsList && workFormAuditListsEqual(beforeList, afterList)
	}
	return inputText(before) == inputText(after)
}

func workFormAuditList(value any) ([]string, bool) {
	switch typed := value.(type) {
	case []any, []string:
		values := append([]string(nil), stringListFromAny(typed)...)
		sort.Strings(values)
		return values, true
	case string:
		text := strings.TrimSpace(typed)
		if strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]") {
			values := append([]string(nil), stringListFromAny(text)...)
			sort.Strings(values)
			return values, true
		}
	}
	return nil, false
}

func workFormAuditListsEqual(before []string, after []string) bool {
	if len(before) != len(after) {
		return false
	}
	for index := range before {
		if before[index] != after[index] {
			return false
		}
	}
	return true
}
