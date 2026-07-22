package setting

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
	"github.com/shemic/dever/orm"
	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"
)

type fieldExportObject struct {
	Name   string
	Code   string
	Schema any
	Config orm.ModelConfig
}

type fieldExportTaskContext struct {
	Workflow *crmmodel.Workflow
	Stage    *crmmodel.Stage
	Task     *crmmodel.Task
}

type fieldExportCatalog struct {
	templates      []*crmmodel.DataTemplate
	fields         []*crmmodel.DataField
	privateOptions []*crmmodel.DataFieldOption
	optionSets     []*crmmodel.OptionSet
	optionItems    []*crmmodel.OptionSetItem
	financeTypes   []*crmmodel.FinanceType
	workflows      []*crmmodel.Workflow
	stages         []*crmmodel.Stage
	tasks          []*crmmodel.Task
	forms          []*crmmodel.Form
	formFields     []*crmmodel.FormField

	templateByID       map[uint64]*crmmodel.DataTemplate
	fieldByID          map[uint64]*crmmodel.DataField
	optionSetByID      map[uint64]*crmmodel.OptionSet
	financeTypeByID    map[uint64]*crmmodel.FinanceType
	privateByFieldID   map[uint64][]*crmmodel.DataFieldOption
	optionItemsBySetID map[uint64][]*crmmodel.OptionSetItem
	workflowByID       map[uint64]*crmmodel.Workflow
	stageByID          map[uint64]*crmmodel.Stage
	formByID           map[uint64]*crmmodel.Form
	formFieldsByFormID map[uint64][]*crmmodel.FormField
	taskContextsByForm map[uint64][]fieldExportTaskContext
}

func (CrmHook) ProviderBuildFieldExport(c *server.Context, _ []any) any {
	ctx := contextFromServer(c)
	objects := newFieldExportObjects()
	catalog := loadFieldExportCatalog(ctx)
	fieldRows, optionRows := catalog.buildFieldRows(objects)

	return map[string]any{
		"fileName": fmt.Sprintf("CRM字段字典-%s.xlsx", time.Now().Format("20060102-150405")),
		"sheets": []map[string]any{
			buildFieldExportSheet("字段总表", fieldExportFieldColumns, fieldRows),
			buildFieldExportSheet("选项明细", fieldExportOptionColumns, optionRows),
			buildFieldExportSheet("流程阶段任务", fieldExportWorkflowColumns, catalog.buildWorkflowRows()),
			buildFieldExportSheet("任务字段引用", fieldExportReferenceColumns, catalog.buildReferenceRows()),
		},
	}
}

func newFieldExportObjects() []fieldExportObject {
	return []fieldExportObject{
		{
			Name:   "线索",
			Code:   crmmodel.DataTemplateTargetLead,
			Schema: crmmodel.Lead{},
			Config: crmmodel.NewLeadModel().Config(),
		},
		{
			Name:   "客户",
			Code:   crmmodel.DataTemplateTargetCustomer,
			Schema: crmmodel.Customer{},
			Config: crmmodel.NewCustomerModel().Config(),
		},
		{
			Name:   "客户资产",
			Code:   crmmodel.DataTemplateTargetCustomerAsset,
			Schema: crmmodel.CustomerAsset{},
			Config: crmmodel.NewCustomerAssetModel().Config(),
		},
	}
}

func loadFieldExportCatalog(ctx context.Context) fieldExportCatalog {
	catalog := fieldExportCatalog{
		templates:          crmmodel.NewDataTemplateModel().Select(ctx, map[string]any{}),
		fields:             crmmodel.NewDataFieldModel().Select(ctx, map[string]any{}),
		privateOptions:     crmmodel.NewDataFieldOptionModel().Select(ctx, map[string]any{}),
		optionSets:         crmmodel.NewOptionSetModel().Select(ctx, map[string]any{}),
		optionItems:        crmmodel.NewOptionSetItemModel().Select(ctx, map[string]any{}),
		financeTypes:       crmmodel.NewFinanceTypeModel().Select(ctx, map[string]any{}),
		workflows:          crmmodel.NewWorkflowModel().Select(ctx, map[string]any{}),
		stages:             crmmodel.NewStageModel().Select(ctx, map[string]any{}),
		tasks:              crmmodel.NewTaskModel().Select(ctx, map[string]any{}),
		forms:              crmmodel.NewFormModel().Select(ctx, map[string]any{}),
		formFields:         crmmodel.NewFormFieldModel().Select(ctx, map[string]any{}),
		templateByID:       map[uint64]*crmmodel.DataTemplate{},
		fieldByID:          map[uint64]*crmmodel.DataField{},
		optionSetByID:      map[uint64]*crmmodel.OptionSet{},
		financeTypeByID:    map[uint64]*crmmodel.FinanceType{},
		privateByFieldID:   map[uint64][]*crmmodel.DataFieldOption{},
		optionItemsBySetID: map[uint64][]*crmmodel.OptionSetItem{},
		workflowByID:       map[uint64]*crmmodel.Workflow{},
		stageByID:          map[uint64]*crmmodel.Stage{},
		formByID:           map[uint64]*crmmodel.Form{},
		formFieldsByFormID: map[uint64][]*crmmodel.FormField{},
		taskContextsByForm: map[uint64][]fieldExportTaskContext{},
	}

	for _, template := range catalog.templates {
		if template != nil {
			catalog.templateByID[template.ID] = template
		}
	}
	for _, field := range catalog.fields {
		if field != nil {
			catalog.fieldByID[field.ID] = field
		}
	}
	for _, optionSet := range catalog.optionSets {
		if optionSet != nil {
			catalog.optionSetByID[optionSet.ID] = optionSet
		}
	}
	for _, financeType := range catalog.financeTypes {
		if financeType != nil {
			catalog.financeTypeByID[financeType.ID] = financeType
		}
	}
	for _, option := range catalog.privateOptions {
		if option != nil {
			catalog.privateByFieldID[option.DataFieldID] = append(catalog.privateByFieldID[option.DataFieldID], option)
		}
	}
	for _, item := range catalog.optionItems {
		if item != nil {
			catalog.optionItemsBySetID[item.OptionSetID] = append(catalog.optionItemsBySetID[item.OptionSetID], item)
		}
	}
	for _, workflow := range catalog.workflows {
		if workflow != nil {
			catalog.workflowByID[workflow.ID] = workflow
		}
	}
	for _, stage := range catalog.stages {
		if stage != nil {
			catalog.stageByID[stage.ID] = stage
		}
	}
	for _, form := range catalog.forms {
		if form != nil {
			catalog.formByID[form.ID] = form
		}
	}
	for _, field := range catalog.formFields {
		if field != nil {
			catalog.formFieldsByFormID[field.FormID] = append(catalog.formFieldsByFormID[field.FormID], field)
		}
	}
	for _, task := range catalog.tasks {
		if task == nil || task.FormID == 0 {
			continue
		}
		stage := catalog.stageByID[task.StageID]
		var workflow *crmmodel.Workflow
		if stage != nil {
			workflow = catalog.workflowByID[stage.WorkflowID]
		}
		catalog.taskContextsByForm[task.FormID] = append(catalog.taskContextsByForm[task.FormID], fieldExportTaskContext{
			Workflow: workflow,
			Stage:    stage,
			Task:     task,
		})
	}

	return catalog
}

func (catalog fieldExportCatalog) buildFieldRows(objects []fieldExportObject) ([]map[string]any, []map[string]any) {
	fieldRows := make([]map[string]any, 0, len(catalog.fields)+64)
	optionRows := make([]map[string]any, 0, len(catalog.privateOptions)+len(catalog.optionItems)+16)
	for _, object := range objects {
		mainFields, mainOptions := buildMainFieldExportRows(object)
		fieldRows = append(fieldRows, mainFields...)
		optionRows = append(optionRows, mainOptions...)
	}

	fieldTypeOptions := crmmodel.NewDataFieldModel().Config().Options["field_type"]
	for _, field := range catalog.fields {
		if field == nil {
			continue
		}
		template := catalog.templateByID[field.DataTemplateID]
		cateID := uint64(0)
		if template != nil {
			cateID = template.CateID
		}
		objectName, objectCode := fieldExportObjectIdentity(cateID)
		parentName := ""
		if parent := catalog.fieldByID[field.ParentFieldID]; parent != nil {
			parentName = parent.Name
		}
		fieldCode := fieldExportDataFieldCode(field)
		financeType := catalog.financeTypeByID[field.FinanceTypeID]
		financeTypeCode := ""
		financeTypeName := ""
		if financeType != nil {
			financeTypeCode = financeType.Code
			financeTypeName = financeType.Name
		} else if field.FinanceTypeID > 0 {
			financeTypeName = fmt.Sprintf("ID:%d", field.FinanceTypeID)
		}

		fieldRows = append(fieldRows, map[string]any{
			"object_name":       objectName,
			"object_code":       objectCode,
			"source_type":       "数据模板",
			"template_id":       field.DataTemplateID,
			"template_name":     fieldExportTemplateName(template, field.DataTemplateID),
			"template_status":   fieldExportTemplateStatus(template),
			"parent_field":      parentName,
			"field_path":        catalog.dataFieldPath(field),
			"field_id":          field.ID,
			"field_name":        field.Name,
			"field_code":        fieldCode,
			"field_type_code":   field.FieldType,
			"field_type_name":   fieldExportOptionLabel(fieldTypeOptions, field.FieldType),
			"finance_type_code": financeTypeCode,
			"finance_type_name": financeTypeName,
			"stat_enabled":      fieldExportBool(field.StatEnabled),
			"storage":           fieldExportDynamicStorage(cateID, field.ID),
			"default_value":     field.DefaultValue,
			"field_status":      fieldExportStatus(field.Status),
			"sort":              field.Sort,
		})

		for _, option := range catalog.privateByFieldID[field.ID] {
			if option == nil {
				continue
			}
			optionRows = append(optionRows, buildFieldOptionExportRow(
				objectName, "数据模板", fieldExportTemplateName(template, field.DataTemplateID),
				field.Name, fieldCode, "字段私有选项", "", option.Name, option.Value, "启用", option.Sort,
			))
		}
		if field.OptionSetID == 0 {
			continue
		}
		optionSet := catalog.optionSetByID[field.OptionSetID]
		for _, item := range catalog.optionItemsBySetID[field.OptionSetID] {
			if item == nil {
				continue
			}
			optionRows = append(optionRows, buildFieldOptionExportRow(
				objectName, "数据模板", fieldExportTemplateName(template, field.DataTemplateID),
				field.Name, fieldCode, "常用选项集", fieldExportOptionSetName(optionSet, field.OptionSetID),
				item.Name, item.Value, fieldExportStatus(item.Status), item.Sort,
			))
		}
	}

	return fieldRows, optionRows
}

func buildMainFieldExportRows(object fieldExportObject) ([]map[string]any, []map[string]any) {
	schemaType := reflect.TypeOf(object.Schema)
	if schemaType.Kind() == reflect.Pointer {
		schemaType = schemaType.Elem()
	}
	if schemaType.Kind() != reflect.Struct {
		return nil, nil
	}

	relationFields := map[string]bool{}
	for _, relation := range object.Config.Relations {
		relationFields[strings.TrimSpace(relation.Field)] = true
	}
	rows := make([]map[string]any, 0, schemaType.NumField())
	optionRows := make([]map[string]any, 0)
	for index := 0; index < schemaType.NumField(); index++ {
		structField := schemaType.Field(index)
		dormTag := structField.Tag.Get("dorm")
		if dormTag == "-" {
			continue
		}
		column := util.ToSnake(structField.Name)
		fieldCode := object.Code + "." + column
		storageType := fieldExportStorageType(structField, dormTag)
		typeName := storageType
		switch {
		case relationFields[column]:
			typeName = "关联"
		case object.Config.Options[column] != nil:
			typeName = "选项"
		}
		fieldName := fieldExportTagValue(dormTag, "comment")
		if fieldName == "" {
			fieldName = column
		}

		rows = append(rows, map[string]any{
			"object_name":     object.Name,
			"object_code":     object.Code,
			"source_type":     "主表",
			"template_name":   "主表",
			"field_path":      column,
			"field_name":      fieldName,
			"field_code":      fieldCode,
			"field_type_code": storageType,
			"field_type_name": typeName,
			"storage":         object.Config.Table + "." + column,
			"default_value":   fieldExportTagValue(dormTag, "default"),
			"field_status":    "固定字段",
			"sort":            index + 1,
		})

		for optionIndex, option := range fieldExportOptionItems(object.Config.Options[column]) {
			optionRows = append(optionRows, buildFieldOptionExportRow(
				object.Name, "主表", "主表", fieldName, fieldCode, "Model枚举", "",
				strings.TrimSpace(fmt.Sprint(option["value"])), strings.TrimSpace(fmt.Sprint(option["id"])),
				"启用", optionIndex+1,
			))
		}
	}
	return rows, optionRows
}

func (catalog fieldExportCatalog) buildWorkflowRows() []map[string]any {
	rows := make([]map[string]any, 0, len(catalog.tasks)+len(catalog.stages))
	stageHasTask := map[uint64]bool{}
	workflowHasStage := map[uint64]bool{}
	for _, task := range catalog.tasks {
		if task == nil {
			continue
		}
		stage := catalog.stageByID[task.StageID]
		var workflow *crmmodel.Workflow
		if stage != nil {
			stageHasTask[stage.ID] = true
			workflowHasStage[stage.WorkflowID] = true
			workflow = catalog.workflowByID[stage.WorkflowID]
		}
		rows = append(rows, catalog.buildWorkflowRow(workflow, stage, task))
	}
	for _, stage := range catalog.stages {
		if stage == nil || stageHasTask[stage.ID] {
			continue
		}
		workflowHasStage[stage.WorkflowID] = true
		rows = append(rows, catalog.buildWorkflowRow(catalog.workflowByID[stage.WorkflowID], stage, nil))
	}
	for _, workflow := range catalog.workflows {
		if workflow == nil || workflowHasStage[workflow.ID] {
			continue
		}
		rows = append(rows, catalog.buildWorkflowRow(workflow, nil, nil))
	}
	return rows
}

func (catalog fieldExportCatalog) buildWorkflowRow(
	workflow *crmmodel.Workflow,
	stage *crmmodel.Stage,
	task *crmmodel.Task,
) map[string]any {
	row := map[string]any{}
	if workflow != nil {
		objectName, objectCode := fieldExportWorkflowObject(workflow.SubjectType)
		row["object_name"] = objectName
		row["object_code"] = objectCode
		row["workflow_id"] = workflow.ID
		row["workflow_name"] = workflow.Name
		row["default_entry"] = fieldExportBool(workflow.DefaultEntry)
		row["workflow_status"] = fieldExportStatus(workflow.Status)
		row["workflow_sort"] = workflow.Sort
	}
	if stage != nil {
		stageOptions := crmmodel.NewStageModel().Config().Options
		row["stage_id"] = stage.ID
		row["stage_name"] = stage.Name
		row["stage_assignment_mode"] = fieldExportOptionLabel(stageOptions["assignment_mode"], stage.AssignmentMode)
		row["stage_owner_department_id"] = stage.OwnerDepartmentID
		row["stage_status"] = fieldExportStatus(stage.Status)
		row["stage_sort"] = stage.Sort
	}
	if task != nil {
		taskOptions := crmmodel.NewTaskModel().Config().Options
		form := catalog.formByID[task.FormID]
		row["task_id"] = task.ID
		row["task_name"] = task.Name
		row["task_type"] = fieldExportOptionLabel(taskOptions["task_type"], task.TaskType)
		row["required"] = fieldExportBool(task.Required)
		row["assignee_mode"] = fieldExportOptionLabel(taskOptions["assignee_mode"], task.AssigneeMode)
		row["assignee_department_id"] = task.AssigneeDepartmentID
		row["form_id"] = task.FormID
		row["form_name"] = fieldExportFormName(form, task.FormID)
		row["script_id"] = task.ScriptID
		row["activation_mode"] = fieldExportOptionLabel(taskOptions["activation_mode"], task.ActivationMode)
		row["reject_action"] = fieldExportOptionLabel(taskOptions["reject_action"], task.RejectAction)
		row["reject_target_task_id"] = task.RejectTargetTaskID
		row["complete_target_task_id"] = task.CompleteTargetTaskID
		row["opinion_requirement"] = fieldExportOptionLabel(taskOptions["opinion_requirement"], task.OpinionRequirement)
		row["reject_submit_form"] = fieldExportBool(task.RejectSubmitForm)
		row["communication_group_enabled"] = fieldExportBool(task.CommunicationGroupEnabled)
		row["due_days"] = task.DueDays
		row["task_status"] = fieldExportStatus(task.Status)
		row["task_sort"] = task.Sort
	}
	return row
}

func (catalog fieldExportCatalog) buildReferenceRows() []map[string]any {
	rows := make([]map[string]any, 0, len(catalog.formFields))
	seenFields := map[uint64]bool{}
	for _, form := range catalog.forms {
		if form == nil {
			continue
		}
		for _, field := range catalog.formFieldsByFormID[form.ID] {
			if field == nil {
				continue
			}
			seenFields[field.ID] = true
			contexts := catalog.taskContextsByForm[form.ID]
			if len(contexts) == 0 {
				rows = append(rows, catalog.buildReferenceRow(form, field, fieldExportTaskContext{}))
				continue
			}
			for _, taskContext := range contexts {
				rows = append(rows, catalog.buildReferenceRow(form, field, taskContext))
			}
		}
	}
	for _, field := range catalog.formFields {
		if field == nil || seenFields[field.ID] {
			continue
		}
		rows = append(rows, catalog.buildReferenceRow(nil, field, fieldExportTaskContext{}))
	}
	return rows
}

func (catalog fieldExportCatalog) buildReferenceRow(
	form *crmmodel.Form,
	field *crmmodel.FormField,
	taskContext fieldExportTaskContext,
) map[string]any {
	row := map[string]any{}
	if taskContext.Workflow != nil {
		row["workflow_id"] = taskContext.Workflow.ID
		row["workflow_name"] = taskContext.Workflow.Name
	}
	if taskContext.Stage != nil {
		row["stage_id"] = taskContext.Stage.ID
		row["stage_name"] = taskContext.Stage.Name
	}
	if taskContext.Task != nil {
		row["task_id"] = taskContext.Task.ID
		row["task_name"] = taskContext.Task.Name
	}
	if form != nil {
		row["form_id"] = form.ID
		row["form_name"] = form.Name
		row["form_status"] = fieldExportStatus(form.Status)
	} else {
		row["form_id"] = field.FormID
	}

	resolved := catalog.resolveFormField(field)
	for key, value := range resolved {
		row[key] = value
	}
	row["form_field_id"] = field.ID
	row["form_field_name"] = field.Name
	row["required"] = fieldExportBool(field.Required)
	row["readonly"] = fieldExportBool(field.Readonly)
	row["field_status"] = fieldExportStatus(field.Status)
	row["sort"] = field.Sort
	return row
}

func (catalog fieldExportCatalog) resolveFormField(field *crmmodel.FormField) map[string]any {
	templateID := field.DataTemplateID
	dataFieldID := field.DataFieldID
	if dataFieldID == 0 && strings.HasPrefix(field.FieldSource, collectFieldSourceDataPrefix) {
		dataFieldID = util.ToUint64(strings.TrimPrefix(field.FieldSource, collectFieldSourceDataPrefix))
	}
	if dataField := catalog.fieldByID[dataFieldID]; dataField != nil {
		if templateID == 0 {
			templateID = dataField.DataTemplateID
		}
		template := catalog.templateByID[templateID]
		cateID := field.DataTemplateCateID
		if cateID == 0 && template != nil {
			cateID = template.CateID
		}
		objectName, _ := fieldExportObjectIdentity(cateID)
		financeType := catalog.financeTypeByID[dataField.FinanceTypeID]
		financeTypeCode := ""
		financeTypeName := ""
		if financeType != nil {
			financeTypeCode = financeType.Code
			financeTypeName = financeType.Name
		} else if dataField.FinanceTypeID > 0 {
			financeTypeName = fmt.Sprintf("ID:%d", dataField.FinanceTypeID)
		}
		return map[string]any{
			"object_name":       objectName,
			"source_type":       "数据模板",
			"template_id":       templateID,
			"template_name":     fieldExportTemplateName(template, templateID),
			"field_id":          dataField.ID,
			"field_code":        fieldExportDataFieldCode(dataField),
			"field_path":        fieldExportFormFieldPath(field, catalog.dataFieldPath(dataField)),
			"finance_type_code": financeTypeCode,
			"finance_type_name": financeTypeName,
			"stat_enabled":      fieldExportBool(dataField.StatEnabled),
		}
	}

	cateID := field.DataTemplateCateID
	mainField := strings.TrimSpace(field.MainField)
	if strings.HasPrefix(field.FieldSource, collectFieldSourceMainPrefix) {
		parsedCateID, parsedField := parseCollectMainFieldSource(field.FieldSource, cateID)
		if parsedCateID > 0 {
			cateID = parsedCateID
		}
		if parsedField != "" {
			mainField = parsedField
		}
	}
	objectName, objectCode := fieldExportObjectIdentity(cateID)
	fieldCode := mainField
	if objectCode != "" && mainField != "" {
		fieldCode = objectCode + "." + mainField
	}
	return map[string]any{
		"object_name": objectName,
		"source_type": "主表",
		"field_code":  fieldCode,
		"field_path":  fieldExportFormFieldPath(field, mainField),
	}
}

func (catalog fieldExportCatalog) dataFieldPath(field *crmmodel.DataField) string {
	if field == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	seen := map[uint64]bool{}
	current := field
	for current != nil && !seen[current.ID] {
		seen[current.ID] = true
		name := strings.TrimSpace(current.Name)
		if name == "" {
			name = strings.TrimSpace(current.FieldKey)
		}
		if name != "" {
			parts = append([]string{name}, parts...)
		}
		current = catalog.fieldByID[current.ParentFieldID]
	}
	return strings.Join(parts, " > ")
}
