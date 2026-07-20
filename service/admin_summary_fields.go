package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

type adminSummaryFieldStatistic struct {
	Field        *crmmodel.DataField
	TemplateName string
	Count        int
	Sum          float64
	Min          float64
	Max          float64
	HasNumber    bool
	DateMin      time.Time
	DateMax      time.Time
	Values       map[string]int
}

func adminSummaryFieldStatistics(ctx context.Context, query AdminSummaryQuery, rangeValue adminSummaryRange) []map[string]any {
	fields, allowedTasks, configuredTaskIDs := adminSummaryConfiguredStatFields(ctx, query.WorkflowID)
	if len(fields) == 0 {
		return []map[string]any{}
	}

	stats := make(map[uint64]*adminSummaryFieldStatistic, len(fields))
	templateNames := map[uint64]string{}
	for fieldID, field := range fields {
		templateName, loaded := templateNames[field.DataTemplateID]
		if !loaded {
			templateName = adminSummaryDataTemplateName(ctx, field.DataTemplateID)
			templateNames[field.DataTemplateID] = templateName
		}
		stats[fieldID] = &adminSummaryFieldStatistic{
			Field:        field,
			TemplateName: templateName,
			Values:       map[string]int{},
		}
	}
	for _, snapshot := range crmmodel.NewStatFieldValueModel().Select(ctx, map[string]any{
		"task_id": configuredTaskIDs,
		"status":  crmmodel.StatusEnabled,
	}) {
		if snapshot == nil || !allowedTasks[snapshot.TaskID] || !adminSummaryInRange(snapshot.UpdatedAt, rangeValue) {
			continue
		}
		stat := stats[snapshot.DataFieldID]
		if stat == nil || adminSummaryStatValueEmpty(snapshot.ValueJSON) {
			continue
		}
		adminSummaryAccumulateFieldStatistic(stat, snapshot)
	}

	ordered := make([]*adminSummaryFieldStatistic, 0, len(stats))
	for _, stat := range stats {
		ordered = append(ordered, stat)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Field.Sort != ordered[j].Field.Sort {
			return ordered[i].Field.Sort < ordered[j].Field.Sort
		}
		return ordered[i].Field.ID < ordered[j].Field.ID
	})

	result := make([]map[string]any, 0, len(ordered))
	for _, stat := range ordered {
		result = append(result, adminSummaryFieldStatisticRow(stat))
	}
	return result
}

func adminSummaryConfiguredStatFields(ctx context.Context, workflowID uint64) (map[uint64]*crmmodel.DataField, map[uint64]bool, []uint64) {
	stageIDs := map[uint64]bool{}
	stageFilters := map[string]any{}
	if workflowID > 0 {
		stageFilters["workflow_id"] = workflowID
	}
	stageFilters["status"] = crmmodel.StatusEnabled
	for _, stage := range crmmodel.NewStageModel().Select(ctx, stageFilters) {
		if stage != nil {
			stageIDs[stage.ID] = true
		}
	}
	if len(stageIDs) == 0 {
		return map[uint64]*crmmodel.DataField{}, map[uint64]bool{}, []uint64{}
	}
	stageIDList := make([]uint64, 0, len(stageIDs))
	for stageID := range stageIDs {
		stageIDList = append(stageIDList, stageID)
	}
	allowedTasks := map[uint64]bool{}
	formIDs := map[uint64]bool{}
	for _, task := range crmmodel.NewTaskModel().Select(ctx, map[string]any{
		"stage_id": stageIDList,
		"status":   crmmodel.StatusEnabled,
	}) {
		if task != nil {
			allowedTasks[task.ID] = true
			if task.FormID > 0 {
				formIDs[task.FormID] = true
			}
		}
	}
	if len(allowedTasks) == 0 {
		return map[uint64]*crmmodel.DataField{}, map[uint64]bool{}, []uint64{}
	}
	taskIDs := make([]uint64, 0, len(allowedTasks))
	for taskID := range allowedTasks {
		taskIDs = append(taskIDs, taskID)
	}

	fieldIDs := adminSummaryFormDataFieldIDs(ctx, formIDs)
	fields := map[uint64]*crmmodel.DataField{}
	if len(fieldIDs) > 0 {
		for _, field := range crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
			"id":           fieldIDs,
			"stat_enabled": true,
			"status":       crmmodel.StatusEnabled,
		}) {
			if workStatDataFieldSupported(field) {
				fields[field.ID] = field
			}
		}
	}
	return fields, allowedTasks, taskIDs
}

func adminSummaryFormDataFieldIDs(ctx context.Context, formIDs map[uint64]bool) []uint64 {
	if len(formIDs) == 0 {
		return nil
	}
	formIDList := make([]uint64, 0, len(formIDs))
	for formID := range formIDs {
		formIDList = append(formIDList, formID)
	}
	referencedFieldIDs := map[uint64]bool{}
	for _, formField := range crmmodel.NewFormFieldModel().Select(ctx, map[string]any{
		"form_id": formIDList,
		"status":  crmmodel.StatusEnabled,
	}) {
		if formField != nil && formField.DataFieldID > 0 {
			referencedFieldIDs[formField.DataFieldID] = true
		}
	}
	if len(referencedFieldIDs) == 0 {
		return nil
	}
	referencedIDList := make([]uint64, 0, len(referencedFieldIDs))
	for fieldID := range referencedFieldIDs {
		referencedIDList = append(referencedIDList, fieldID)
	}

	fieldIDs := map[uint64]bool{}
	groupIDs := make([]uint64, 0)
	for _, field := range crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"id":     referencedIDList,
		"status": crmmodel.StatusEnabled,
	}) {
		if field == nil {
			continue
		}
		if field.FieldType == "group" {
			groupIDs = append(groupIDs, field.ID)
			continue
		}
		fieldIDs[field.ID] = true
	}
	if len(groupIDs) > 0 {
		for _, child := range crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
			"parent_field_id": groupIDs,
			"status":          crmmodel.StatusEnabled,
		}) {
			if child != nil && child.FieldType != "group" {
				fieldIDs[child.ID] = true
			}
		}
	}
	result := make([]uint64, 0, len(fieldIDs))
	for fieldID := range fieldIDs {
		result = append(result, fieldID)
	}
	return result
}

func adminSummaryDataTemplateName(ctx context.Context, templateID uint64) string {
	if template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID}); template != nil {
		return template.Name
	}
	return ""
}

func adminSummaryStatValueEmpty(valueJSON string) bool {
	text := strings.TrimSpace(valueJSON)
	if text == "" {
		return true
	}
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return false
	}
	if mapped, ok := value.(map[string]any); ok {
		return len(mapped) == 0
	}
	return emptyWorkFieldValue(value)
}

func adminSummaryAccumulateFieldStatistic(stat *adminSummaryFieldStatistic, snapshot *crmmodel.StatFieldValue) {
	switch snapshot.StatType {
	case crmmodel.DataFieldStatTypeMetric, crmmodel.DataFieldStatTypeAmount, crmmodel.DataFieldStatTypeFinance:
		stat.Count++
		value := snapshot.ValueNumber
		stat.Sum += value
		if !stat.HasNumber || value < stat.Min {
			stat.Min = value
		}
		if !stat.HasNumber || value > stat.Max {
			stat.Max = value
		}
		stat.HasNumber = true
	case crmmodel.DataFieldStatTypeTime:
		if snapshot.ValueDate.IsZero() {
			return
		}
		stat.Count++
		if stat.DateMin.IsZero() || snapshot.ValueDate.Before(stat.DateMin) {
			stat.DateMin = snapshot.ValueDate
		}
		if stat.DateMax.IsZero() || snapshot.ValueDate.After(stat.DateMax) {
			stat.DateMax = snapshot.ValueDate
		}
	default:
		values := []string{strings.TrimSpace(snapshot.ValueText)}
		if snapshot.FieldType == "checkbox" || snapshot.FieldType == "multi_select" {
			values = strings.Split(snapshot.ValueText, "、")
		}
		validValues := make([]string, 0, len(values))
		for _, value := range values {
			if value = strings.TrimSpace(value); value != "" {
				validValues = append(validValues, value)
			}
		}
		if len(validValues) == 0 {
			return
		}
		stat.Count++
		for _, value := range validValues {
			stat.Values[value]++
		}
	}
}

func adminSummaryFieldStatisticRow(stat *adminSummaryFieldStatistic) map[string]any {
	row := map[string]any{
		"id":            stat.Field.ID,
		"key":           stat.Field.FieldKey,
		"name":          stat.Field.Name,
		"template_name": stat.TemplateName,
		"field_type":    stat.Field.FieldType,
		"stat_type":     workDataFieldStatType(stat.Field.FieldType),
		"count":         stat.Count,
		"values":        adminSummaryFieldValueRows(stat.Values, stat.Count),
	}
	if stat.HasNumber {
		row["sum"] = stat.Sum
		row["average"] = stat.Sum / float64(stat.Count)
		row["min"] = stat.Min
		row["max"] = stat.Max
	}
	if !stat.DateMin.IsZero() {
		row["date_min"] = stat.DateMin
		row["date_max"] = stat.DateMax
	}
	return row
}

func adminSummaryFieldValueRows(values map[string]int, total int) []map[string]any {
	type valueCount struct {
		Value string
		Count int
	}
	rows := make([]valueCount, 0, len(values))
	for value, count := range values {
		rows = append(rows, valueCount{Value: value, Count: count})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		return rows[i].Value < rows[j].Value
	})
	if len(rows) > 5 {
		rows = rows[:5]
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{
			"value":   row.Value,
			"count":   row.Count,
			"percent": adminSummaryPercent(row.Count, total),
		})
	}
	return result
}
