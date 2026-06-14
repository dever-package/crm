package setting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

type OptionService struct{}

const taskVisibleValueSourcePrefix = "value:"

var ensureBaseDataTemplateCatesOnce sync.Once

func (OptionService) ProviderLoadStageOptions(c *server.Context, _ []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	ensureDefaultStage(ctx)
	rows := crmmodel.NewStageModel().SelectMap(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}, stageSelectOptions())
	return loadStageOptions(rows)
}

func (OptionService) ProviderLoadDataTemplateCates(c *server.Context, _ []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	ensureBaseDataTemplateCates(ctx)
	rows := crmmodel.NewDataTemplateCateModel().SelectMap(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}, dataTemplateCateSelectOptions())
	return loadCrmCateOptions(rows)
}

func (OptionService) ProviderLoadFormFieldCascaderOptions(c *server.Context, params []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	ensureBaseDataTemplateCates(ctx)
	parentID := optionString(c, params, "parent_id", "parentId", "rootValue")
	switch {
	case parentID == "" || parentID == "0":
		return formFieldCateOptions(ctx)
	case strings.HasPrefix(parentID, "cate:"):
		return formFieldTemplateLevelOptions(ctx, util.ToUint64(strings.TrimPrefix(parentID, "cate:")))
	case isCollectMainTemplateSource(parentID):
		return formFieldMainFieldOptions(collectMainTemplateCateID(parentID))
	case strings.HasPrefix(parentID, collectTemplateSourceDataPrefix):
		cateID, templateID := parseCollectTemplateSource(ctx, parentID)
		if templateID == 0 {
			return []map[string]any{}
		}
		template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
			"id":      templateID,
			"cate_id": cateID,
			"status":  crmmodel.StatusEnabled,
		})
		if template == nil {
			return []map[string]any{}
		}
		return formFieldDataFieldOptions(ctx, cateID, templateID, template.Name)
	default:
		return []map[string]any{}
	}
}

func (OptionService) ProviderLoadTaskVisibleConditionOptions(c *server.Context, params []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	ensureBaseDataTemplateCates(ctx)
	parentID := optionString(c, params, "parent_id", "parentId", "rootValue")
	switch {
	case parentID == "" || parentID == "0":
		return formFieldCateOptions(ctx)
	case strings.HasPrefix(parentID, "cate:"):
		return taskVisibleTemplateOptions(ctx, util.ToUint64(strings.TrimPrefix(parentID, "cate:")))
	case strings.HasPrefix(parentID, collectTemplateSourceDataPrefix):
		_, templateID := parseCollectTemplateSource(ctx, parentID)
		return taskVisibleFieldOptions(ctx, templateID)
	case strings.HasPrefix(parentID, collectFieldSourceDataPrefix):
		fieldID := util.ToUint64(strings.TrimPrefix(parentID, collectFieldSourceDataPrefix))
		return taskVisibleValueOptions(ctx, fieldID)
	default:
		return []map[string]any{}
	}
}

func (OptionService) ProviderLoadDecisionResultFieldOptions(c *server.Context, params []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	ensureBaseDataTemplateCates(ctx)
	parentID := optionString(c, params, "parent_id", "parentId", "rootValue")
	switch {
	case parentID == "" || parentID == "0":
		return formFieldCateOptions(ctx)
	case strings.HasPrefix(parentID, "cate:"):
		return taskVisibleTemplateOptions(ctx, util.ToUint64(strings.TrimPrefix(parentID, "cate:")))
	case strings.HasPrefix(parentID, collectTemplateSourceDataPrefix):
		_, templateID := parseCollectTemplateSource(ctx, parentID)
		return decisionResultFieldOptions(ctx, templateID)
	default:
		return []map[string]any{}
	}
}

func (OptionService) ProviderLoadFormFieldTemplateOptions(c *server.Context, params []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	ensureBaseDataTemplateCates(ctx)
	cateID := optionUint64(c, params, "data_template_cate_id", "dataTemplateCateId", "cate_id", "cateId")
	if cateID == 0 {
		cateID = optionUint64(c, params, "form_data_template_cate_id", "formDataTemplateCateId")
	}
	if cateID == 0 {
		cateID = optionUint64(c, params, "parent_id", "parentId", "rootValue")
	}
	if selected := firstSelectedOptionValue(c, params); selected != "" {
		if option := formFieldSelectedTemplateOption(ctx, cateID, selected); option != nil {
			return []map[string]any{option}
		}
	}
	if cateID == 0 {
		return []map[string]any{}
	}
	cates := crmmodel.NewDataTemplateCateModel().SelectMap(ctx, map[string]any{
		"id":     cateID,
		"status": crmmodel.StatusEnabled,
	}, dataTemplateCateSelectOptions())
	options := make([]map[string]any, 0, len(cates))
	for _, cate := range cates {
		cateID := util.ToUint64(cate["id"])
		cateName := util.ToStringTrimmed(cate["name"])
		options = append(options, map[string]any{
			"id":                    formFieldMainTemplateOptionID(cateID),
			"value":                 "主表",
			"name":                  "主表",
			"cate_name":             cateName,
			"template_name":         "主表",
			"is_main":               true,
			"data_template_cate_id": cateID,
		})
		templates := crmmodel.NewDataTemplateModel().SelectMap(ctx, map[string]any{
			"cate_id": cateID,
			"status":  crmmodel.StatusEnabled,
		}, map[string]any{
			"field": "main.id, main.name, main.sort",
			"order": "main.sort asc, main.id asc",
		})
		for _, template := range templates {
			templateID := util.ToUint64(template["id"])
			templateName := util.ToStringTrimmed(template["name"])
			options = append(options, map[string]any{
				"id":                    formFieldDataTemplateOptionID(templateID),
				"value":                 templateName,
				"name":                  templateName,
				"cate_name":             cateName,
				"template_name":         templateName,
				"is_main":               false,
				"data_template_id":      templateID,
				"data_template_cate_id": cateID,
			})
		}
	}
	return options
}

func formFieldCateOptions(ctx context.Context) []map[string]any {
	allowedCateIDs := map[uint64]bool{
		crmmodel.CustomerDataTemplateCateID:      true,
		crmmodel.CustomerAssetDataTemplateCateID: true,
	}
	cates := crmmodel.NewDataTemplateCateModel().SelectMap(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}, dataTemplateCateSelectOptions())
	options := make([]map[string]any, 0, len(cates))
	for _, cate := range cates {
		cateID := util.ToUint64(cate["id"])
		if !allowedCateIDs[cateID] {
			continue
		}
		options = append(options, map[string]any{
			"id":           fmt.Sprintf("cate:%d", cateID),
			"value":        util.ToStringTrimmed(cate["name"]),
			"name":         util.ToStringTrimmed(cate["name"]),
			"leaf":         false,
			"target_table": util.ToStringTrimmed(cate["target_table"]),
			"sort":         util.ToIntDefault(cate["sort"], 0),
		})
	}
	return options
}

func formFieldTemplateLevelOptions(ctx context.Context, cateID uint64) []map[string]any {
	if cateID == 0 {
		return []map[string]any{}
	}
	options := []map[string]any{{
		"id":                    collectMainTemplateSource(cateID),
		"value":                 "主表",
		"name":                  "主表",
		"is_main":               true,
		"leaf":                  false,
		"data_template_cate_id": cateID,
	}}
	templates := crmmodel.NewDataTemplateModel().SelectMap(ctx, map[string]any{
		"cate_id": cateID,
		"status":  crmmodel.StatusEnabled,
	}, map[string]any{
		"field": "main.id, main.name, main.sort",
		"order": "main.sort asc, main.id asc",
	})
	for _, template := range templates {
		templateID := util.ToUint64(template["id"])
		options = append(options, map[string]any{
			"id":                    collectDataTemplateSource(templateID),
			"value":                 util.ToStringTrimmed(template["name"]),
			"name":                  util.ToStringTrimmed(template["name"]),
			"is_main":               false,
			"leaf":                  false,
			"data_template_id":      templateID,
			"data_template_cate_id": cateID,
		})
	}
	return options
}

func taskVisibleTemplateOptions(ctx context.Context, cateID uint64) []map[string]any {
	if cateID == 0 {
		return []map[string]any{}
	}
	templates := crmmodel.NewDataTemplateModel().SelectMap(ctx, map[string]any{
		"cate_id": cateID,
		"status":  crmmodel.StatusEnabled,
	}, map[string]any{
		"field": "main.id, main.name, main.sort",
		"order": "main.sort asc, main.id asc",
	})
	options := make([]map[string]any, 0, len(templates))
	for _, template := range templates {
		templateID := util.ToUint64(template["id"])
		options = append(options, map[string]any{
			"id":                    collectDataTemplateSource(templateID),
			"value":                 util.ToStringTrimmed(template["name"]),
			"name":                  util.ToStringTrimmed(template["name"]),
			"leaf":                  false,
			"data_template_id":      templateID,
			"data_template_cate_id": cateID,
			"sort":                  util.ToIntDefault(template["sort"], 0),
		})
	}
	return options
}

func taskVisibleFieldOptions(ctx context.Context, templateID uint64) []map[string]any {
	if templateID == 0 {
		return []map[string]any{}
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
		"id":     templateID,
		"status": crmmodel.StatusEnabled,
	})
	if template == nil {
		return []map[string]any{}
	}
	rows := crmmodel.NewDataFieldModel().SelectMap(ctx, map[string]any{
		"data_template_id": templateID,
		"stat_enabled":     true,
		"status":           crmmodel.StatusEnabled,
	}, map[string]any{
		"field": "main.id, main.name, main.field_type, main.sort",
		"order": "main.sort asc, main.id asc",
	})
	options := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		fieldID := util.ToUint64(row["id"])
		if !taskVisibleFieldHasOptions(util.ToStringTrimmed(row["field_type"])) ||
			crmmodel.NewDataFieldOptionModel().Count(ctx, map[string]any{"data_field_id": fieldID}) == 0 {
			continue
		}
		options = append(options, map[string]any{
			"id":                    fmt.Sprintf("%s%d", collectFieldSourceDataPrefix, fieldID),
			"value":                 util.ToStringTrimmed(row["name"]),
			"name":                  util.ToStringTrimmed(row["name"]),
			"leaf":                  false,
			"source":                "data_field",
			"data_template_id":      templateID,
			"data_field_id":         fieldID,
			"field_type":            util.ToStringTrimmed(row["field_type"]),
			"data_template_cate_id": template.CateID,
			"sort":                  util.ToIntDefault(row["sort"], 0),
		})
	}
	return options
}

func decisionResultFieldOptions(ctx context.Context, templateID uint64) []map[string]any {
	if templateID == 0 {
		return []map[string]any{}
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
		"id":     templateID,
		"status": crmmodel.StatusEnabled,
	})
	if template == nil {
		return []map[string]any{}
	}
	rows := crmmodel.NewDataFieldModel().SelectMap(ctx, map[string]any{
		"data_template_id": templateID,
		"stat_enabled":     true,
		"status":           crmmodel.StatusEnabled,
	}, map[string]any{
		"field": "main.id, main.name, main.field_key, main.field_type, main.sort",
		"order": "main.sort asc, main.id asc",
	})
	options := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		fieldID := util.ToUint64(row["id"])
		fieldKey := util.ToStringTrimmed(row["field_key"])
		if fieldKey == "" || !taskVisibleFieldHasOptions(util.ToStringTrimmed(row["field_type"])) ||
			crmmodel.NewDataFieldOptionModel().Count(ctx, map[string]any{"data_field_id": fieldID}) == 0 {
			continue
		}
		options = append(options, map[string]any{
			"id":                    fmt.Sprintf("%s%d", collectFieldSourceDataPrefix, fieldID),
			"value":                 util.ToStringTrimmed(row["name"]),
			"name":                  util.ToStringTrimmed(row["name"]),
			"leaf":                  true,
			"source":                "data_field",
			"data_template_id":      templateID,
			"data_field_id":         fieldID,
			"field_key":             fieldKey,
			"field_type":            util.ToStringTrimmed(row["field_type"]),
			"data_template_cate_id": template.CateID,
			"sort":                  util.ToIntDefault(row["sort"], 0),
		})
	}
	return options
}

func taskVisibleValueOptions(ctx context.Context, fieldID uint64) []map[string]any {
	if fieldID == 0 {
		return []map[string]any{}
	}
	rows := crmmodel.NewDataFieldOptionModel().SelectMap(ctx, map[string]any{
		"data_field_id": fieldID,
	}, map[string]any{
		"field": "main.name, main.value, main.sort",
		"order": "main.sort asc, main.id asc",
	})
	options := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		value := util.ToStringTrimmed(row["value"])
		if value == "" {
			continue
		}
		name := util.ToStringTrimmed(row["name"])
		if name == "" {
			name = value
		}
		options = append(options, map[string]any{
			"id":            taskVisibleValueSource(fieldID, value),
			"value":         name,
			"name":          name,
			"leaf":          true,
			"data_field_id": fieldID,
			"option_value":  value,
			"sort":          util.ToIntDefault(row["sort"], 0),
		})
	}
	return options
}

func taskVisibleValueSource(fieldID uint64, value string) string {
	return fmt.Sprintf("%s%d:%s", taskVisibleValueSourcePrefix, fieldID, strings.TrimSpace(value))
}

func parseTaskVisibleValueSource(source string) (uint64, string, bool) {
	source = strings.TrimSpace(source)
	if !strings.HasPrefix(source, taskVisibleValueSourcePrefix) {
		return 0, "", false
	}
	raw := strings.TrimPrefix(source, taskVisibleValueSourcePrefix)
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return 0, "", false
	}
	fieldID := util.ToUint64(parts[0])
	value := strings.TrimSpace(parts[1])
	return fieldID, value, fieldID > 0 && value != ""
}

func (OptionService) ProviderLoadFormFieldOptions(c *server.Context, params []any) any {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	templateSource := optionString(c, params, "data_template_id", "dataTemplateId", "template_id", "templateId")
	if templateSource == "" {
		templateSource = optionString(c, params, "form_data_template_id", "formDataTemplateId")
	}
	if templateSource == "" {
		templateSource = optionString(c, params, "parent_id", "parentId", "rootValue")
	}
	if templateSource == "" {
		return []map[string]any{}
	}
	cateID, templateID := parseCollectTemplateSource(ctx, templateSource)
	if cateID == 0 {
		cateID = optionUint64(c, params, "cate_id", "cateId", "data_template_cate_id", "dataTemplateCateId")
	}
	if cateID == 0 {
		cateID = optionUint64(c, params, "form_data_template_cate_id", "formDataTemplateCateId")
	}
	if cateID == 0 && templateID > 0 {
		if template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID}); template != nil {
			cateID = template.CateID
		}
	}
	if selected := firstSelectedOptionValue(c, params); selected != "" {
		if option := formFieldSelectedFieldOption(ctx, cateID, templateID, selected); option != nil {
			return []map[string]any{option}
		}
	}
	if cateID == 0 {
		return []map[string]any{}
	}
	if templateID == 0 {
		return formFieldMainFieldOptions(cateID)
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
		"id":      templateID,
		"cate_id": cateID,
		"status":  crmmodel.StatusEnabled,
	})
	if template == nil {
		return []map[string]any{}
	}
	return formFieldDataFieldOptions(ctx, cateID, templateID, template.Name)
}

func firstSelectedOptionValue(c *server.Context, params []any) string {
	raw := optionString(c, params, "selected")
	if raw == "" {
		return ""
	}
	var decoded []any
	if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
		for _, item := range decoded {
			if value := util.ToStringTrimmed(item); value != "" {
				return value
			}
		}
		return ""
	}
	return raw
}

func formFieldSelectedTemplateOption(ctx context.Context, cateID uint64, selected string) map[string]any {
	selected = strings.TrimSpace(selected)
	if isCollectMainTemplateSource(selected) {
		selectedCateID := collectMainTemplateCateID(selected)
		if selectedCateID > 0 {
			cateID = selectedCateID
		}
		return formFieldMainTemplateOption(ctx, cateID)
	}
	if strings.HasPrefix(selected, collectTemplateSourceDataPrefix) {
		templateID := util.ToUint64(strings.TrimPrefix(selected, collectTemplateSourceDataPrefix))
		return formFieldTemplateOption(ctx, templateID)
	}
	if templateID := util.ToUint64(selected); templateID > 0 {
		return formFieldTemplateOption(ctx, templateID)
	}
	return nil
}

func formFieldMainTemplateOption(ctx context.Context, cateID uint64) map[string]any {
	if cateID == 0 {
		return nil
	}
	cateName := ""
	if cate := crmmodel.NewDataTemplateCateModel().Find(ctx, map[string]any{"id": cateID}); cate != nil {
		cateName = cate.Name
	}
	return map[string]any{
		"id":                    formFieldMainTemplateOptionID(cateID),
		"value":                 "主表",
		"name":                  "主表",
		"cate_name":             cateName,
		"template_name":         "主表",
		"is_main":               true,
		"data_template_cate_id": cateID,
	}
}

func formFieldTemplateOption(ctx context.Context, templateID uint64) map[string]any {
	if templateID == 0 {
		return nil
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID})
	if template == nil {
		return nil
	}
	templateName := template.Name
	cateName := ""
	if cate := crmmodel.NewDataTemplateCateModel().Find(ctx, map[string]any{"id": template.CateID}); cate != nil {
		cateName = cate.Name
	}
	return map[string]any{
		"id":                    formFieldDataTemplateOptionID(template.ID),
		"value":                 templateName,
		"name":                  templateName,
		"cate_name":             cateName,
		"template_name":         templateName,
		"is_main":               false,
		"data_template_id":      template.ID,
		"data_template_cate_id": template.CateID,
	}
}

func formFieldSelectedFieldOption(ctx context.Context, cateID uint64, templateID uint64, selected string) map[string]any {
	selected = strings.TrimSpace(selected)
	if strings.HasPrefix(selected, collectFieldSourceMainPrefix) {
		if templateID != 0 {
			return nil
		}
		sourceCateID, mainField := parseCollectMainFieldSource(selected, cateID)
		if sourceCateID > 0 {
			cateID = sourceCateID
		}
		option, ok := collectMainFieldByKey(cateID, mainField)
		if !ok {
			return nil
		}
		return formFieldMainFieldOption(cateID, 0, option)
	}
	if strings.HasPrefix(selected, collectFieldSourceDataPrefix) {
		if templateID == 0 {
			return nil
		}
		fieldID := util.ToUint64(strings.TrimPrefix(selected, collectFieldSourceDataPrefix))
		option := formFieldDataFieldOption(ctx, fieldID)
		if option == nil {
			return nil
		}
		if util.ToUint64(option["data_template_id"]) != templateID || util.ToUint64(option["data_template_cate_id"]) != cateID {
			return nil
		}
		return option
	}
	return nil
}

func formFieldMainTemplateOptionID(cateID uint64) string {
	return collectMainTemplateSource(cateID)
}

func formFieldDataTemplateOptionID(templateID uint64) string {
	return collectDataTemplateSource(templateID)
}

func parseCollectTemplateSource(ctx context.Context, source string) (uint64, uint64) {
	source = strings.TrimSpace(source)
	if isCollectMainTemplateSource(source) {
		return collectMainTemplateCateID(source), 0
	}
	if strings.HasPrefix(source, collectTemplateSourceDataPrefix) {
		templateID := util.ToUint64(strings.TrimPrefix(source, collectTemplateSourceDataPrefix))
		if templateID == 0 {
			return 0, 0
		}
		template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID})
		if template == nil {
			return 0, templateID
		}
		return template.CateID, templateID
	}
	templateID := util.ToUint64(source)
	if templateID == 0 {
		return 0, 0
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID})
	if template == nil {
		return 0, templateID
	}
	return template.CateID, templateID
}

func formFieldDataFieldOptions(ctx context.Context, cateID uint64, templateID uint64, templateName string) []map[string]any {
	rows := crmmodel.NewDataFieldModel().SelectMap(ctx, map[string]any{
		"data_template_id": templateID,
		"status":           crmmodel.StatusEnabled,
	}, map[string]any{
		"field": "main.id, main.name, main.field_type, main.sort",
		"order": "main.sort asc, main.id asc",
	})
	options := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		options = append(options, map[string]any{
			"id":                    fmt.Sprintf("%s%d", collectFieldSourceDataPrefix, util.ToUint64(row["id"])),
			"value":                 util.ToStringTrimmed(row["name"]),
			"name":                  util.ToStringTrimmed(row["name"]),
			"leaf":                  true,
			"source":                "data_field",
			"data_template_id":      templateID,
			"data_field_id":         util.ToUint64(row["id"]),
			"field_type":            util.ToStringTrimmed(row["field_type"]),
			"data_template_cate_id": cateID,
			"sort":                  util.ToIntDefault(row["sort"], 0),
		})
	}
	return options
}

func formFieldMainFieldOptions(cateID uint64) []map[string]any {
	fields := collectMainFields[cateID]
	options := make([]map[string]any, 0, len(fields))
	for index, field := range fields {
		options = append(options, formFieldMainFieldOption(cateID, index, field))
	}
	return options
}

func formFieldMainFieldOption(cateID uint64, index int, field collectFieldOption) map[string]any {
	return map[string]any{
		"id":                    collectMainFieldSource(cateID, field.Key),
		"value":                 field.Name,
		"name":                  field.Name,
		"leaf":                  true,
		"source":                "main_field",
		"main_field":            field.Key,
		"data_template_cate_id": cateID,
		"sort":                  (index + 1) * 10,
	}
}

func formFieldDataFieldOption(ctx context.Context, fieldID uint64) map[string]any {
	if fieldID == 0 {
		return nil
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
	if field == nil {
		return nil
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": field.DataTemplateID})
	if template == nil {
		return nil
	}
	return map[string]any{
		"id":                    fmt.Sprintf("%s%d", collectFieldSourceDataPrefix, field.ID),
		"value":                 field.Name,
		"name":                  field.Name,
		"leaf":                  true,
		"source":                "data_field",
		"data_template_id":      field.DataTemplateID,
		"data_field_id":         field.ID,
		"field_type":            field.FieldType,
		"data_template_cate_id": template.CateID,
		"sort":                  field.Sort,
	}
}

func optionUint64(c *server.Context, params []any, keys ...string) uint64 {
	for _, param := range params {
		row, ok := param.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range keys {
			if value := util.ToUint64(row[key]); value > 0 {
				return value
			}
		}
	}
	if c == nil {
		return 0
	}
	for _, key := range keys {
		if value := util.ToUint64(strings.TrimSpace(c.Query(key))); value > 0 {
			return value
		}
	}
	return 0
}

func optionString(c *server.Context, params []any, keys ...string) string {
	for _, param := range params {
		row, ok := param.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range keys {
			if value := util.ToStringTrimmed(row[key]); value != "" {
				return value
			}
		}
	}
	if c == nil {
		return ""
	}
	for _, key := range keys {
		if value := strings.TrimSpace(c.Query(key)); value != "" {
			return value
		}
	}
	return ""
}

func assetCateSelectOptions() map[string]any {
	return map[string]any{
		"field": "main.id, main.name, main.status, main.sort",
		"order": "main.sort asc, main.id asc",
	}
}

func dataTemplateCateSelectOptions() map[string]any {
	return map[string]any{
		"field": "main.id, main.name, main.target_table, main.status, main.sort",
		"order": "main.sort asc, main.id asc",
	}
}

func stageSelectOptions() map[string]any {
	return map[string]any{
		"field": "main.id, main.name, main.code, main.owner_department_id, main.status, main.sort",
		"order": "main.sort asc, main.id asc",
	}
}

func loadCrmCateOptions(rows []map[string]any) []map[string]any {
	options := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		options = append(options, map[string]any{
			"id":           util.ToUint64(row["id"]),
			"value":        util.ToStringTrimmed(row["name"]),
			"name":         util.ToStringTrimmed(row["name"]),
			"target_table": util.ToStringTrimmed(row["target_table"]),
			"status":       util.ToIntDefault(row["status"], 0),
			"sort":         util.ToIntDefault(row["sort"], 0),
		})
	}
	return options
}

func loadStageOptions(rows []map[string]any) []map[string]any {
	options := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		name := util.ToStringTrimmed(row["name"])
		code := util.ToStringTrimmed(row["code"])
		options = append(options, map[string]any{
			"id":                  util.ToUint64(row["id"]),
			"value":               name,
			"name":                name,
			"code":                code,
			"display_name":        stageDisplayName(code, name),
			"owner_department_id": util.ToUint64(row["owner_department_id"]),
			"status":              util.ToIntDefault(row["status"], 0),
			"sort":                util.ToIntDefault(row["sort"], 0),
		})
	}
	return options
}

func stageDisplayName(code string, name string) string {
	switch {
	case code == "":
		return name
	case name == "":
		return code
	default:
		return name + " " + code
	}
}

func ensureBaseDataTemplateCates(ctx context.Context) {
	ensureBaseDataTemplateCatesOnce.Do(func() {
		ensureBaseDataTemplateCate(ctx, crmmodel.CustomerDataTemplateCateID, "客户信息", crmmodel.DataTemplateTargetCustomer, 10)
		ensureBaseDataTemplateCate(ctx, crmmodel.CustomerAssetDataTemplateCateID, "客户资产", crmmodel.DataTemplateTargetCustomerAsset, 20)
	})
}

func ensureBaseDataTemplateCate(ctx context.Context, id uint64, name string, targetTable string, sort int) {
	model := crmmodel.NewDataTemplateCateModel()
	if model.Find(ctx, map[string]any{"id": id}) == nil {
		model.Insert(ctx, map[string]any{
			"id":           id,
			"name":         name,
			"target_table": targetTable,
			"status":       crmmodel.StatusEnabled,
			"sort":         sort,
		})
		return
	}
	model.Update(ctx, map[string]any{"id": id}, map[string]any{
		"name":         name,
		"target_table": targetTable,
		"sort":         sort,
	})
}
