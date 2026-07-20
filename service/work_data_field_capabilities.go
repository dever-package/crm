package service

import (
	"context"
	"sort"

	crmmodel "github.com/dever-package/crm/model"
)

func workSubmittedDataFields(ctx context.Context, formInput *workFormInput) []*crmmodel.DataField {
	fieldIDs := workSubmittedDataFieldIDs(formInput)
	if len(fieldIDs) == 0 {
		return nil
	}
	return crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"id":     fieldIDs,
		"status": crmmodel.StatusEnabled,
	})
}

func workSubmittedDataFieldIDs(formInput *workFormInput) []uint64 {
	if formInput == nil {
		return nil
	}
	seen := map[uint64]bool{}
	for _, records := range []map[uint64]map[string]any{
		formInput.leadDataRecords,
		formInput.customerDataRecords,
		formInput.assetDataRecords,
	} {
		for _, record := range records {
			for fieldIDText := range record {
				fieldID := inputUint64(fieldIDText)
				if fieldID > 0 {
					seen[fieldID] = true
				}
			}
		}
	}
	fieldIDs := make([]uint64, 0, len(seen))
	for fieldID := range seen {
		fieldIDs = append(fieldIDs, fieldID)
	}
	sort.Slice(fieldIDs, func(i, j int) bool { return fieldIDs[i] < fieldIDs[j] })
	return fieldIDs
}

func workSubmittedDataFieldValue(
	formInput *workFormInput,
	ownership workDataOwnership,
	field *crmmodel.DataField,
) (any, workDataOwnership, bool) {
	if formInput == nil || field == nil {
		return nil, ownership, false
	}
	fieldID := inputText(field.ID)
	if value, submitted := workDataRecordFieldValue(formInput.leadDataRecords, field.DataTemplateID, fieldID); submitted {
		return value, workDataOwnership{
			LeadID:             ownership.LeadID,
			WorkflowInstanceID: ownership.WorkflowInstanceID,
		}, true
	}
	if value, submitted := workDataRecordFieldValue(formInput.customerDataRecords, field.DataTemplateID, fieldID); submitted {
		return value, workDataOwnership{
			CustomerID:         ownership.CustomerID,
			WorkflowInstanceID: ownership.WorkflowInstanceID,
			CustomerProductID:  ownership.CustomerProductID,
		}, true
	}
	if value, submitted := workDataRecordFieldValue(formInput.assetDataRecords, field.DataTemplateID, fieldID); submitted {
		return value, ownership, true
	}
	return nil, ownership, false
}

func workDataRecordFieldValue(records map[uint64]map[string]any, templateID uint64, fieldID string) (any, bool) {
	record := records[templateID]
	if record == nil {
		return nil, false
	}
	value, exists := record[fieldID]
	return value, exists
}
