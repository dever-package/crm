package setting

import (
	"fmt"
	"reflect"
	"strings"

	crmmodel "github.com/dever-package/crm/model"
)

func buildFieldExportSheet(name string, columns []fieldExportColumn, rows []map[string]any) map[string]any {
	head := make([]map[string]any, 0, len(columns))
	for _, column := range columns {
		head = append(head, map[string]any{
			"key":   column.Key,
			"title": column.Title,
			"width": column.Width,
		})
	}
	return map[string]any{
		"name":       name,
		"startCell":  "A1",
		"freeze":     "A2",
		"autoFilter": true,
		"head":       head,
		"body":       rows,
	}
}

func buildFieldOptionExportRow(
	objectName string,
	sourceType string,
	templateName string,
	fieldName string,
	fieldCode string,
	optionSource string,
	optionSetName string,
	optionName string,
	optionValue string,
	optionStatus string,
	sort int,
) map[string]any {
	return map[string]any{
		"object_name":     objectName,
		"source_type":     sourceType,
		"template_name":   templateName,
		"field_name":      fieldName,
		"field_code":      fieldCode,
		"option_source":   optionSource,
		"option_set_name": optionSetName,
		"option_name":     optionName,
		"option_value":    optionValue,
		"option_status":   optionStatus,
		"sort":            sort,
	}
}

func fieldExportObjectIdentity(cateID uint64) (string, string) {
	switch cateID {
	case crmmodel.LeadDataTemplateCateID:
		return "线索", crmmodel.DataTemplateTargetLead
	case crmmodel.CustomerDataTemplateCateID:
		return "客户", crmmodel.DataTemplateTargetCustomer
	case crmmodel.CustomerAssetDataTemplateCateID:
		return "客户资产", crmmodel.DataTemplateTargetCustomerAsset
	default:
		return "", ""
	}
}

func fieldExportDataFieldCode(field *crmmodel.DataField) string {
	if field == nil {
		return ""
	}
	if fieldCode := strings.TrimSpace(field.FieldKey); fieldCode != "" {
		return fieldCode
	}
	return fmt.Sprintf("data_field:%d", field.ID)
}

func fieldExportDynamicStorage(cateID uint64, fieldID uint64) string {
	if cateID == crmmodel.LeadDataTemplateCateID {
		return fmt.Sprintf(
			`%s.record_json.data_values["data:%d"]`,
			crmmodel.NewLeadModel().Config().Table,
			fieldID,
		)
	}
	return fmt.Sprintf(
		`%s.record_json["%d"]`,
		crmmodel.NewDataRecordModel().Config().Table,
		fieldID,
	)
}

func fieldExportWorkflowObject(subjectType string) (string, string) {
	switch strings.TrimSpace(subjectType) {
	case crmmodel.WorkflowSubjectLead:
		return "线索", crmmodel.DataTemplateTargetLead
	case crmmodel.WorkflowSubjectCustomerAsset:
		return "客户资产", crmmodel.DataTemplateTargetCustomerAsset
	default:
		return subjectType, subjectType
	}
}

func fieldExportTemplateName(template *crmmodel.DataTemplate, templateID uint64) string {
	if template != nil {
		return template.Name
	}
	if templateID == 0 {
		return ""
	}
	return fmt.Sprintf("模板#%d", templateID)
}

func fieldExportTemplateStatus(template *crmmodel.DataTemplate) string {
	if template == nil {
		return ""
	}
	return fieldExportStatus(template.Status)
}

func fieldExportOptionSetName(optionSet *crmmodel.OptionSet, optionSetID uint64) string {
	if optionSet != nil {
		return optionSet.Name
	}
	return fmt.Sprintf("选项集#%d", optionSetID)
}

func fieldExportFormName(form *crmmodel.Form, formID uint64) string {
	if form != nil {
		return form.Name
	}
	if formID == 0 {
		return ""
	}
	return fmt.Sprintf("表单#%d", formID)
}

func fieldExportFormFieldPath(field *crmmodel.FormField, fallback string) string {
	path := collectPathItems(field.FieldPath)
	if len(path) > 0 {
		return strings.Join(path, " > ")
	}
	return fallback
}

func fieldExportStorageType(field reflect.StructField, dormTag string) string {
	if fieldType := fieldExportTagValue(dormTag, "type"); fieldType != "" {
		return fieldType
	}
	current := field.Type
	if current.Kind() == reflect.Pointer {
		current = current.Elem()
	}
	if current.PkgPath() == "time" && current.Name() == "Time" {
		return "timestamp"
	}
	return current.String()
}

func fieldExportTagValue(tag string, key string) string {
	for _, part := range strings.Split(tag, ";") {
		name, value, found := strings.Cut(strings.TrimSpace(part), ":")
		if found && strings.TrimSpace(name) == key {
			return strings.Trim(strings.TrimSpace(value), "'")
		}
	}
	return ""
}

func fieldExportOptionItems(raw any) []map[string]any {
	switch options := raw.(type) {
	case []map[string]any:
		return options
	case []any:
		result := make([]map[string]any, 0, len(options))
		for _, option := range options {
			if row, ok := option.(map[string]any); ok {
				result = append(result, row)
			}
		}
		return result
	default:
		return nil
	}
}

func fieldExportOptionLabel(raw any, value any) string {
	target := strings.TrimSpace(fmt.Sprint(value))
	for _, option := range fieldExportOptionItems(raw) {
		if strings.TrimSpace(fmt.Sprint(option["id"])) == target {
			return strings.TrimSpace(fmt.Sprint(option["value"]))
		}
	}
	return target
}

func fieldExportStatus(status int16) string {
	switch status {
	case crmmodel.StatusEnabled:
		return "启用"
	case crmmodel.StatusDisabled:
		return "停用"
	default:
		return fmt.Sprint(status)
	}
}

func fieldExportBool(value bool) string {
	if value {
		return "是"
	}
	return "否"
}
