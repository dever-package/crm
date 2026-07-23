package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

const (
	historyLeadTemplateName     = "历史飞书线索资料"
	historyCustomerTemplateName = "历史飞书客户资料"
	historyAssetTemplateName    = "历史飞书资产资料"
)

type historyImportTemplateCatalog struct {
	Template *crmmodel.DataTemplate
	Fields   map[string]*crmmodel.DataField
	Options  map[uint64]map[string]string
}

type historyImportCatalog struct {
	Templates        map[string]historyImportTemplateCatalog
	StaffByOpenID    map[string][]*crmmodel.Staff
	StaffByName      map[string][]*crmmodel.Staff
	SourcesByCode    map[string]*crmmodel.CustomerSource
	ChannelsByCode   map[string]*crmmodel.CustomerChannel
	ResourcesByName  map[string][]*crmmodel.PublicResource
	WorkflowsByName  map[string]*crmmodel.Workflow
	StagesByWorkflow map[uint64]map[string]*crmmodel.Stage
	GroupType        *crmmodel.CommunicationGroupType
}

func loadHistoryImportCatalog(ctx context.Context) (historyImportCatalog, error) {
	catalog := historyImportCatalog{
		Templates:        map[string]historyImportTemplateCatalog{},
		StaffByOpenID:    map[string][]*crmmodel.Staff{},
		StaffByName:      map[string][]*crmmodel.Staff{},
		SourcesByCode:    map[string]*crmmodel.CustomerSource{},
		ChannelsByCode:   map[string]*crmmodel.CustomerChannel{},
		ResourcesByName:  map[string][]*crmmodel.PublicResource{},
		WorkflowsByName:  map[string]*crmmodel.Workflow{},
		StagesByWorkflow: map[uint64]map[string]*crmmodel.Stage{},
	}

	for _, template := range crmmodel.NewDataTemplateModel().Select(ctx, map[string]any{}) {
		if template == nil {
			continue
		}
		entry := historyImportTemplateCatalog{
			Template: template,
			Fields:   map[string]*crmmodel.DataField{},
			Options:  map[uint64]map[string]string{},
		}
		for _, field := range crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
			"data_template_id": template.ID,
		}) {
			if field == nil || field.Status != crmmodel.StatusEnabled {
				continue
			}
			entry.Fields[historyCatalogKey(field.Name)] = field
			entry.Fields[historyCatalogKey(field.FieldKey)] = field
			if field.FieldType != "select" && field.FieldType != "multi_select" &&
				field.FieldType != "radio" && field.FieldType != "checkbox" {
				continue
			}
			options := map[string]string{}
			if field.OptionSetID > 0 {
				for _, option := range crmmodel.NewOptionSetItemModel().Select(ctx, map[string]any{
					"option_set_id": field.OptionSetID,
					"status":        crmmodel.StatusEnabled,
				}) {
					if option == nil {
						continue
					}
					options[historyCatalogKey(option.Name)] = option.Value
					options[historyCatalogKey(option.Value)] = option.Value
				}
			}
			for _, option := range crmmodel.NewDataFieldOptionModel().Select(ctx, map[string]any{
				"data_field_id": field.ID,
			}) {
				if option == nil {
					continue
				}
				options[historyCatalogKey(option.Name)] = option.Value
				options[historyCatalogKey(option.Value)] = option.Value
			}
			entry.Options[field.ID] = options
		}
		catalog.Templates[historyCatalogKey(template.Name)] = entry
	}

	for _, staff := range crmmodel.NewStaffModel().Select(ctx, map[string]any{}) {
		if staff == nil {
			continue
		}
		if key := historyCatalogKey(staff.FeishuOpenID); key != "" {
			catalog.StaffByOpenID[key] = append(catalog.StaffByOpenID[key], staff)
		}
		if key := historyCatalogKey(staff.Name); key != "" {
			catalog.StaffByName[key] = append(catalog.StaffByName[key], staff)
		}
	}
	for _, source := range crmmodel.NewCustomerSourceModel().Select(ctx, map[string]any{}) {
		if source != nil {
			catalog.SourcesByCode[historyCatalogKey(source.Code)] = source
		}
	}
	for _, channel := range crmmodel.NewCustomerChannelModel().Select(ctx, map[string]any{}) {
		if channel != nil {
			catalog.ChannelsByCode[historyCatalogKey(channel.Code)] = channel
		}
	}
	for _, resource := range crmmodel.NewPublicResourceModel().Select(ctx, map[string]any{}) {
		if resource == nil {
			continue
		}
		key := historyResourceKey(resource.Name)
		catalog.ResourcesByName[key] = append(catalog.ResourcesByName[key], resource)
	}
	for _, workflow := range crmmodel.NewWorkflowModel().Select(ctx, map[string]any{}) {
		if workflow == nil {
			continue
		}
		catalog.WorkflowsByName[historyCatalogKey(workflow.Name)] = workflow
		stageMap := map[string]*crmmodel.Stage{}
		for _, stage := range crmmodel.NewStageModel().Select(ctx, map[string]any{
			"workflow_id": workflow.ID,
		}) {
			if stage != nil {
				stageMap[historyCatalogKey(stage.Name)] = stage
			}
		}
		catalog.StagesByWorkflow[workflow.ID] = stageMap
	}
	catalog.GroupType = crmmodel.NewCommunicationGroupTypeModel().Find(ctx, map[string]any{
		"code": crmmodel.CommunicationGroupTypeEnterpriseWechat,
	})
	return catalog, nil
}

func ensureHistoryImportConfiguration(
	ctx context.Context,
	catalog *historyImportCatalog,
	batch HistoryImportBatchInput,
) error {
	if catalog == nil {
		return fmt.Errorf("历史导入目录不能为空")
	}
	now := time.Now()
	for _, input := range batch.Cases {
		if input.Lead != nil {
			ensureHistoryImportSource(ctx, catalog, input.Lead.SourceCode, input.Lead.SourceName, now)
			ensureHistoryImportChannel(ctx, catalog, input.Lead.ChannelCode, input.Lead.ChannelName, now)
		}
		if input.Customer != nil {
			ensureHistoryImportSource(ctx, catalog, input.Customer.SourceCode, input.Customer.SourceName, now)
			ensureHistoryImportChannel(ctx, catalog, input.Customer.ChannelCode, input.Customer.ChannelName, now)
		}
	}

	templateFields := collectHistoryImportTemplateFields(batch)
	for templateName, fields := range templateFields {
		cateID := historyTemplateCateID(templateName)
		template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"name": templateName})
		if template == nil {
			templateID := uint64(crmmodel.NewDataTemplateModel().Insert(ctx, map[string]any{
				"cate_id":      cateID,
				"name":         templateName,
				"display_mode": crmmodel.DataTemplateDisplayFilled,
				"status":       crmmodel.StatusEnabled,
				"sort":         900,
				"created_at":   now,
				"updated_at":   now,
			}))
			if templateID == 0 {
				return fmt.Errorf("创建数据模板失败：%s", templateName)
			}
			template = crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID})
		}
		if template == nil {
			return fmt.Errorf("读取数据模板失败：%s", templateName)
		}
		if template.DisplayMode != crmmodel.DataTemplateDisplayFilled && strings.HasPrefix(templateName, "历史飞书") {
			crmmodel.NewDataTemplateModel().Update(ctx, map[string]any{"id": template.ID}, map[string]any{
				"display_mode": crmmodel.DataTemplateDisplayFilled,
				"updated_at":   now,
			})
		}
		names := make([]string, 0, len(fields))
		for name := range fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for index, name := range names {
			if crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
				"data_template_id": template.ID,
				"name":             name,
			}) != nil {
				continue
			}
			fieldID := uint64(crmmodel.NewDataFieldModel().Insert(ctx, map[string]any{
				"data_template_id": template.ID,
				"parent_field_id":  uint64(0),
				"option_set_id":    uint64(0),
				"name":             name,
				"field_key":        historyFieldKey(templateName, name),
				"field_type":       historyFieldType(fields[name]),
				"default_value":    "",
				"finance_type_id":  uint64(0),
				"stat_enabled":     false,
				"sort":             100 + index,
				"status":           crmmodel.StatusEnabled,
				"created_at":       now,
				"updated_at":       now,
			}))
			if fieldID == 0 {
				return fmt.Errorf("创建历史字段失败：%s/%s", templateName, name)
			}
		}
	}

	if catalog.GroupType == nil {
		groupTypeID := uint64(crmmodel.NewCommunicationGroupTypeModel().Insert(ctx, map[string]any{
			"code":        crmmodel.CommunicationGroupTypeEnterpriseWechat,
			"name":        "企业微信",
			"description": "企业微信客户沟通群。",
			"status":      crmmodel.StatusEnabled,
			"sort":        10,
			"created_at":  now,
			"updated_at":  now,
		}))
		if groupTypeID == 0 {
			return fmt.Errorf("创建企业微信群类型失败")
		}
	}
	reloaded, err := loadHistoryImportCatalog(ctx)
	if err != nil {
		return err
	}
	for _, input := range batch.Cases {
		if input.Lead != nil && (reloaded.SourcesByCode[historyCatalogKey(input.Lead.SourceCode)] == nil ||
			reloaded.ChannelsByCode[historyCatalogKey(input.Lead.ChannelCode)] == nil) {
			return fmt.Errorf("案件%s的线索来源或渠道创建失败", input.CaseID)
		}
		if input.Customer != nil && (reloaded.SourcesByCode[historyCatalogKey(input.Customer.SourceCode)] == nil ||
			reloaded.ChannelsByCode[historyCatalogKey(input.Customer.ChannelCode)] == nil) {
			return fmt.Errorf("案件%s的客户来源或渠道创建失败", input.CaseID)
		}
	}
	*catalog = reloaded
	return nil
}

func ensureHistoryImportSource(
	ctx context.Context,
	catalog *historyImportCatalog,
	code string,
	name string,
	now time.Time,
) {
	code = strings.TrimSpace(code)
	key := historyCatalogKey(code)
	if code == "" || catalog.SourcesByCode[key] != nil {
		return
	}
	model := crmmodel.NewCustomerSourceModel()
	if existing := model.Find(ctx, map[string]any{"code": code}); existing != nil {
		catalog.SourcesByCode[key] = existing
		return
	}
	id := uint64(model.Insert(ctx, map[string]any{
		"code": code, "name": fallbackHistoryName(name, code), "status": crmmodel.StatusEnabled,
		"sort": 900, "created_at": now,
	}))
	if id > 0 {
		catalog.SourcesByCode[key] = model.Find(ctx, map[string]any{"id": id})
	}
}

func ensureHistoryImportChannel(
	ctx context.Context,
	catalog *historyImportCatalog,
	code string,
	name string,
	now time.Time,
) {
	code = strings.TrimSpace(code)
	key := historyCatalogKey(code)
	if code == "" || catalog.ChannelsByCode[key] != nil {
		return
	}
	model := crmmodel.NewCustomerChannelModel()
	if existing := model.Find(ctx, map[string]any{"code": code}); existing != nil {
		catalog.ChannelsByCode[key] = existing
		return
	}
	id := uint64(model.Insert(ctx, map[string]any{
		"code": code, "name": fallbackHistoryName(name, code), "status": crmmodel.StatusEnabled,
		"sort": 900, "created_at": now,
	}))
	if id > 0 {
		catalog.ChannelsByCode[key] = model.Find(ctx, map[string]any{"id": id})
	}
}

func collectHistoryImportTemplateFields(batch HistoryImportBatchInput) map[string]map[string]any {
	result := map[string]map[string]any{}
	collect := func(records []HistoryImportDataRecordInput) {
		for _, record := range records {
			templateName := strings.TrimSpace(record.TemplateName)
			if templateName == "" || !strings.HasPrefix(templateName, "历史飞书") {
				continue
			}
			if result[templateName] == nil {
				result[templateName] = map[string]any{}
			}
			for name, value := range record.Fields {
				if strings.TrimSpace(name) != "" && !historyValueEmpty(value) {
					result[templateName][name] = value
				}
			}
		}
	}
	for _, input := range batch.Cases {
		collect(input.CustomerRecords)
		for _, asset := range input.Assets {
			collect(asset.Records)
		}
	}
	return result
}

func historyTemplateCateID(templateName string) uint64 {
	switch strings.TrimSpace(templateName) {
	case historyLeadTemplateName:
		// 沉淀数据目前没有 lead_id；转化后的线索历史按客户资料展示，
		// 未转化线索仍可从线索快照和历史导入审计追溯。
		return crmmodel.CustomerDataTemplateCateID
	case historyAssetTemplateName:
		return crmmodel.CustomerAssetDataTemplateCateID
	default:
		return crmmodel.CustomerDataTemplateCateID
	}
}

func historyFieldKey(templateName string, fieldName string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(templateName) + "\x00" + strings.TrimSpace(fieldName)))
	return "feishu_history_" + hex.EncodeToString(sum[:10])
}

func historyFieldType(value any) string {
	switch value.(type) {
	case bool:
		return "boolean"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return "number"
	case time.Time, *time.Time:
		return "datetime"
	default:
		return "textarea"
	}
}

func historyCatalogKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func historyResourceKey(value string) string {
	replacer := strings.NewReplacer("一号", "1号", "二号", "2号", "三号", "3号", " ", "")
	return historyCatalogKey(replacer.Replace(value))
}

func fallbackHistoryName(name string, code string) string {
	if name = strings.TrimSpace(name); name != "" {
		return name
	}
	return strings.TrimSpace(code)
}
