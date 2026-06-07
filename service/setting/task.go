package setting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "my/package/crm/model"
)

const (
	collectTemplateSourceMainPrefix = "main_table:"
	collectTemplateSourceDataPrefix = "template:"
	collectFieldSourceMainPrefix    = "main:"
	collectFieldSourceDataPrefix    = "data:"
)

var obsoleteTaskTemplateFields = []string{
	"task_key",
	"task_type",
	"script_id",
	"executor_mode",
	"default_department_id",
	"default_role_id",
	"default_staff_id",
	"enable_deadline",
	"deadline_minutes",
	"config_json",
}

var obsoleteTaskFieldFields = []string{
	"field_key",
	"field_type",
	"options_json",
	"default_value",
	"placeholder",
	"help_text",
	"sync_data_template_id",
	"sync_data_field_id",
	"is_blocking",
	"is_metric",
	"metric_key",
	"metric_type",
	"aggregate",
	"collect_labels",
	"collect_cate_name",
	"collect_template_name",
	"collect_field_name",
}

var collectMainFields = map[uint64][]collectFieldOption{
	crmmodel.CustomerDataTemplateCateID: {
		{Key: "name", Name: "姓名"},
		{Key: "phone", Name: "手机号"},
		{Key: "wechat", Name: "微信"},
	},
	crmmodel.CustomerAssetDataTemplateCateID: {
		{Key: "asset_name", Name: "资产名称"},
		{Key: "asset_status", Name: "资产状态"},
	},
}

type collectFieldOption struct {
	Key  string
	Name string
}

func (CrmHook) ProviderBeforeSaveTaskTemplate(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	for _, field := range obsoleteTaskTemplateFields {
		delete(record, field)
	}
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "description", partial)
	normalizeEmbeddedTaskFields(c, record, partial)
	if !partial {
		if util.ToUint64(record["cate_id"]) == 0 {
			panicCrmField("form.cate_id", "任务分类不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "任务名不能为空。")
		}
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBeforeSaveTaskField(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	normalizeTaskFieldRecord(c, record, partial, true)
	return record
}

func (CrmHook) ProviderBuildTaskTemplateForm(c *server.Context, params []any) any {
	record := taskFormRecord(params)
	if len(record) == 0 {
		return record
	}
	normalizeTaskFormRows(c, record)
	return record
}

func (CrmHook) ProviderBuildTaskFieldForm(c *server.Context, params []any) any {
	record := taskFormRecord(params)
	if len(record) == 0 {
		return record
	}
	normalizeTaskFieldFormRow(c, record)
	return record
}

func (CrmHook) ProviderBuildTaskFieldRows(c *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}
	for _, row := range rows {
		normalizeTaskFieldFormRow(c, row)
	}
	return rows
}

func taskFormRecord(params []any) map[string]any {
	if len(params) == 0 {
		return nil
	}
	payload, ok := params[0].(map[string]any)
	if !ok {
		return nil
	}
	if record, ok := payload["record"].(map[string]any); ok {
		return util.CloneMap(record)
	}
	return util.CloneMap(payload)
}

func normalizeTaskFormRows(c *server.Context, record map[string]any) {
	rows := taskFieldRows(record["fields"])
	if rows == nil {
		return
	}
	for _, row := range rows {
		normalizeTaskFieldFormRow(c, row)
	}
	record["fields"] = rows
}

func normalizeTaskFieldFormRow(c *server.Context, row map[string]any) {
	path := collectPathItems(row["collect_path"])
	if len(path) == 0 {
		path = collectPathItems(buildTaskCollectPath(row))
	}
	row["collect_path"] = stringSliceToAny(path)
	attachTaskCollectDisplay(c, row)
	attachTaskCollectLabels(row)
	normalizeTaskFieldFormTemplateValue(row)
}

func normalizeEmbeddedTaskFields(c *server.Context, record map[string]any, partial bool) {
	if partial {
		if _, exists := record["fields"]; !exists {
			return
		}
	}
	rows := taskFieldRows(record["fields"])
	if rows == nil {
		return
	}
	normalized := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if !hasTaskCollectSelection(row) {
			continue
		}
		normalizeTaskFieldRecord(c, row, false, false)
		normalized = append(normalized, row)
	}
	record["fields"] = normalized
}

func taskFieldRows(value any) []map[string]any {
	switch rows := value.(type) {
	case []map[string]any:
		return rows
	case []any:
		result := make([]map[string]any, 0, len(rows))
		for _, item := range rows {
			if row, ok := item.(map[string]any); ok {
				result = append(result, row)
			}
		}
		return result
	case string:
		var decoded []map[string]any
		if err := json.Unmarshal([]byte(rows), &decoded); err == nil {
			return decoded
		}
		return nil
	default:
		return nil
	}
}

func hasTaskCollectSelection(record map[string]any) bool {
	if collectFieldSourceFromRecord(record) != "" {
		return true
	}
	for _, item := range collectPathItems(record["collect_path"]) {
		if strings.HasPrefix(item, collectFieldSourceMainPrefix) || strings.HasPrefix(item, collectFieldSourceDataPrefix) {
			return true
		}
	}
	return false
}

func normalizeTaskFieldRecord(c *server.Context, record map[string]any, partial bool, requireTask bool) {
	for _, field := range obsoleteTaskFieldFields {
		delete(record, field)
	}
	normalizeTaskCollectPath(record, partial)
	trimCrmStringField(record, "field_source", partial)
	trimCrmStringField(record, "collect_path", partial)
	trimCrmStringField(record, "main_field", partial)
	trimCrmStringField(record, "name", partial)

	if requireTask && !partial && util.ToUint64(record["task_template_id"]) == 0 {
		panicCrmField("form.task_template_id", "任务不能为空。")
	}
	if shouldNormalizeCrmField(record, "data_template_cate_id", partial) && util.ToUint64(record["data_template_cate_id"]) == 0 {
		record["data_template_cate_id"] = crmmodel.CustomerDataTemplateCateID
	}
	if shouldNormalizeCrmField(record, "field_source", partial) || shouldNormalizeCrmField(record, "collect_path", partial) {
		normalizeTaskFieldSource(c, record)
	}
	if shouldNormalizeCrmField(record, "collect_path", partial) {
		record["collect_path"] = buildTaskCollectPath(record)
	}
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.field_source", "收集数据不能为空。")
	}
	if shouldNormalizeCrmField(record, "required", partial) {
		if _, exists := record["required"]; !exists || record["required"] == nil || record["required"] == "" {
			record["required"] = true
		}
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
}

func normalizeTaskCollectPath(record map[string]any, partial bool) {
	_, hasCollectPath := record["collect_path"]
	if partial {
		if !hasCollectPath {
			return
		}
	}
	path := collectPathItems(record["collect_path"])
	source := util.ToStringTrimmed(record["field_source"])
	if hasCollectPath && source == "" {
		record["main_field"] = ""
		record["data_field_id"] = uint64(0)
		record["data_template_id"] = uint64(0)
		record["name"] = ""
	}
	if len(path) == 0 {
		if hasCollectPath {
			record["collect_path"] = "[]"
		} else if shouldNormalizeCrmField(record, "collect_path", partial) {
			record["collect_path"] = buildTaskCollectPath(record)
		}
		return
	}
	record["collect_path"] = encodeCollectPath(path)
	cateID := uint64(0)
	templateID := uint64(0)
	for _, item := range path {
		switch {
		case strings.HasPrefix(item, "cate:"):
			cateID = util.ToUint64(strings.TrimPrefix(item, "cate:"))
		case isCollectMainTemplateSource(item):
			cateID = collectMainTemplateCateID(item)
			templateID = 0
		case strings.HasPrefix(item, collectTemplateSourceDataPrefix):
			templateID = util.ToUint64(strings.TrimPrefix(item, collectTemplateSourceDataPrefix))
			record["data_template_id"] = templateID
		case strings.HasPrefix(item, collectFieldSourceMainPrefix), strings.HasPrefix(item, collectFieldSourceDataPrefix):
			record["field_source"] = item
		}
	}
	if cateID > 0 {
		record["data_template_cate_id"] = cateID
	}
	if templateID == 0 {
		if len(path) >= 2 && isCollectMainTemplateSource(path[1]) {
			record["data_template_id"] = uint64(0)
		}
	}
	if hasCollectPath && util.ToStringTrimmed(record["field_source"]) == "" {
		record["field_source"] = source
	}
}

func buildTaskCollectPath(record map[string]any) string {
	cateID := util.ToUint64(record["data_template_cate_id"])
	source := collectFieldSourceFromRecord(record)
	if cateID == 0 || source == "" {
		return "[]"
	}
	selector := collectMainTemplateSource(cateID)
	if templateID := util.ToUint64(record["data_template_id"]); templateID > 0 {
		selector = collectDataTemplateSource(templateID)
	}
	return encodeCollectPath([]string{
		fmt.Sprintf("cate:%d", cateID),
		selector,
		source,
	})
}

func encodeCollectPath(path []string) string {
	encoded, err := json.Marshal(compactCollectPathItems(path))
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func collectPathItems(value any) []string {
	switch current := value.(type) {
	case []string:
		return compactCollectPathItems(current)
	case []any:
		items := make([]string, 0, len(current))
		for _, item := range current {
			items = append(items, util.ToStringTrimmed(item))
		}
		return compactCollectPathItems(items)
	case string:
		var decoded []string
		if err := json.Unmarshal([]byte(current), &decoded); err == nil {
			return compactCollectPathItems(decoded)
		}
		var decodedAny []any
		if err := json.Unmarshal([]byte(current), &decodedAny); err == nil {
			return collectPathItems(decodedAny)
		}
		return compactCollectPathItems(strings.Split(current, ","))
	default:
		return nil
	}
}

func stringSliceToAny(items []string) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

func compactCollectPathItems(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item = normalizeCollectPathItem(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func normalizeCollectPathItem(item string) string {
	item = strings.TrimSpace(item)
	if strings.HasPrefix(item, collectFieldSourceMainPrefix) {
		raw := strings.TrimPrefix(item, collectFieldSourceMainPrefix)
		if strings.Contains(raw, ":") {
			return item
		}
		if cateID := util.ToUint64(raw); cateID > 0 {
			return collectMainTemplateSource(cateID)
		}
	}
	return item
}

func attachTaskCollectDisplay(c *server.Context, row map[string]any) {
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}

	cateID := util.ToUint64(row["data_template_cate_id"])
	if cateID == 0 {
		for _, item := range collectPathItems(row["collect_path"]) {
			if strings.HasPrefix(item, "cate:") {
				cateID = util.ToUint64(strings.TrimPrefix(item, "cate:"))
				break
			}
		}
	}
	if cateID > 0 {
		if cate := crmmodel.NewDataTemplateCateModel().Find(ctx, map[string]any{"id": cateID}); cate != nil {
			row["data_template_cate"] = map[string]any{
				"id":           cate.ID,
				"name":         cate.Name,
				"target_table": cate.TargetTable,
			}
		}
	}

	templateID := util.ToUint64(row["data_template_id"])
	if templateID > 0 {
		if template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID}); template != nil {
			row["data_template"] = map[string]any{
				"id":      template.ID,
				"name":    template.Name,
				"cate_id": template.CateID,
			}
		}
	} else {
		row["data_template"] = map[string]any{
			"id":      uint64(0),
			"name":    "主表",
			"cate_id": cateID,
		}
	}

	if fieldID := util.ToUint64(row["data_field_id"]); fieldID > 0 {
		if field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID}); field != nil {
			row["data_field"] = map[string]any{
				"id":         field.ID,
				"name":       field.Name,
				"field_type": field.FieldType,
			}
			if util.ToStringTrimmed(row["name"]) == "" {
				row["name"] = field.Name
			}
		}
		return
	}

	if option, ok := collectMainFieldByKey(cateID, util.ToStringTrimmed(row["main_field"])); ok {
		if util.ToStringTrimmed(row["name"]) == "" {
			row["name"] = option.Name
		}
	}
}

func attachTaskCollectLabels(row map[string]any) {
	cateName := nestedTaskCollectName(row, "data_template_cate")
	templateName := nestedTaskCollectName(row, "data_template")
	fieldName := util.ToStringTrimmed(row["name"])
	row["collect_cate_name"] = cateName
	row["collect_template_name"] = templateName
	row["collect_field_name"] = fieldName
	row["collect_labels"] = []any{cateName, templateName, fieldName}
}

func normalizeTaskFieldFormTemplateValue(row map[string]any) {
	if raw := util.ToStringTrimmed(row["data_template_id"]); isCollectMainTemplateSource(raw) || strings.HasPrefix(raw, collectTemplateSourceDataPrefix) {
		return
	}
	cateID := util.ToUint64(row["data_template_cate_id"])
	if cateID == 0 {
		return
	}
	if templateID := util.ToUint64(row["data_template_id"]); templateID > 0 {
		row["data_template_id"] = collectDataTemplateSource(templateID)
		return
	}
	row["data_template_id"] = collectMainTemplateSource(cateID)
}

func nestedTaskCollectName(row map[string]any, key string) string {
	value, ok := row[key].(map[string]any)
	if !ok {
		return ""
	}
	return util.ToStringTrimmed(value["name"])
}

func normalizeTaskFieldSource(c *server.Context, record map[string]any) {
	source := util.ToStringTrimmed(record["field_source"])
	cateID := util.ToUint64(record["data_template_cate_id"])
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	cateID, templateID := normalizeCollectTemplateValue(ctx, record, cateID)
	if cateID == 0 {
		if templateID > 0 {
			if template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID}); template != nil {
				cateID = template.CateID
				record["data_template_cate_id"] = template.CateID
			}
		}
	}
	if cateID == 0 {
		cateID = crmmodel.CustomerDataTemplateCateID
		record["data_template_cate_id"] = cateID
	}
	if strings.HasPrefix(source, collectFieldSourceMainPrefix) {
		sourceCateID, mainField := parseCollectMainFieldSource(source, cateID)
		if sourceCateID > 0 {
			cateID = sourceCateID
			record["data_template_cate_id"] = cateID
		}
		option, ok := collectMainFieldByKey(cateID, mainField)
		if !ok {
			panicCrmField("form.field_source", "主表字段不存在。")
		}
		record["main_field"] = option.Key
		record["data_template_id"] = uint64(0)
		record["data_field_id"] = uint64(0)
		record["collect_path"] = buildTaskCollectPath(record)
		if util.ToStringTrimmed(record["name"]) == "" {
			record["name"] = option.Name
		}
		return
	}
	if strings.HasPrefix(source, collectFieldSourceDataPrefix) {
		dataFieldID := util.ToUint64(strings.TrimPrefix(source, collectFieldSourceDataPrefix))
		if dataFieldID == 0 {
			panicCrmField("form.field_source", "数据模板字段不存在。")
		}
		dataField := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": dataFieldID})
		if dataField == nil {
			panicCrmField("form.field_source", "数据模板字段不存在。")
		}
		template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": dataField.DataTemplateID})
		if template == nil {
			panicCrmField("form.data_template_id", "数据模板不存在。")
		}
		record["data_template_cate_id"] = template.CateID
		record["data_template_id"] = template.ID
		record["data_field_id"] = dataField.ID
		record["main_field"] = ""
		record["collect_path"] = buildTaskCollectPath(record)
		if util.ToStringTrimmed(record["name"]) == "" {
			record["name"] = dataField.Name
		}
		return
	}
	if source == "" {
		panicCrmField("form.field_source", "收集数据不能为空。")
	}
	panicCrmField("form.field_source", "收集数据来源无效。")
}

func normalizeCollectTemplateValue(ctx context.Context, record map[string]any, cateID uint64) (uint64, uint64) {
	raw := util.ToStringTrimmed(record["data_template_id"])
	if isCollectMainTemplateSource(raw) {
		cateID = collectMainTemplateCateID(raw)
		record["data_template_cate_id"] = cateID
		record["data_template_id"] = uint64(0)
		return cateID, 0
	}
	if strings.HasPrefix(raw, collectTemplateSourceDataPrefix) {
		templateID := util.ToUint64(strings.TrimPrefix(raw, collectTemplateSourceDataPrefix))
		record["data_template_id"] = templateID
		if templateID == 0 {
			return cateID, 0
		}
		if template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID}); template != nil {
			record["data_template_cate_id"] = template.CateID
			return template.CateID, templateID
		}
		return cateID, templateID
	}
	templateID := util.ToUint64(record["data_template_id"])
	return cateID, templateID
}

func collectMainTemplateSource(cateID uint64) string {
	return fmt.Sprintf("%s%d", collectTemplateSourceMainPrefix, cateID)
}

func collectDataTemplateSource(templateID uint64) string {
	return fmt.Sprintf("%s%d", collectTemplateSourceDataPrefix, templateID)
}

func isCollectMainTemplateSource(source string) bool {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, collectTemplateSourceMainPrefix) {
		return true
	}
	if !strings.HasPrefix(source, collectFieldSourceMainPrefix) {
		return false
	}
	raw := strings.TrimPrefix(source, collectFieldSourceMainPrefix)
	return !strings.Contains(raw, ":") && util.ToUint64(raw) > 0
}

func collectMainTemplateCateID(source string) uint64 {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, collectTemplateSourceMainPrefix) {
		return util.ToUint64(strings.TrimPrefix(source, collectTemplateSourceMainPrefix))
	}
	if strings.HasPrefix(source, collectFieldSourceMainPrefix) {
		return util.ToUint64(strings.TrimPrefix(source, collectFieldSourceMainPrefix))
	}
	return 0
}

func collectMainFieldSource(cateID uint64, fieldKey string) string {
	fieldKey = strings.TrimSpace(fieldKey)
	if cateID == 0 {
		return collectFieldSourceMainPrefix + fieldKey
	}
	return fmt.Sprintf("%s%d:%s", collectFieldSourceMainPrefix, cateID, fieldKey)
}

func parseCollectMainFieldSource(source string, fallbackCateID uint64) (uint64, string) {
	raw := strings.TrimPrefix(strings.TrimSpace(source), collectFieldSourceMainPrefix)
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 2 {
		if cateID := util.ToUint64(parts[0]); cateID > 0 {
			return cateID, strings.TrimSpace(parts[1])
		}
	}
	return fallbackCateID, strings.TrimSpace(raw)
}

func collectMainFieldByKey(cateID uint64, key string) (collectFieldOption, bool) {
	for _, option := range collectMainFields[cateID] {
		if option.Key == key {
			return option, true
		}
	}
	return collectFieldOption{}, false
}

func collectFieldSourceFromRecord(record map[string]any) string {
	source := util.ToStringTrimmed(record["field_source"])
	if source != "" {
		return source
	}
	if mainField := util.ToStringTrimmed(record["main_field"]); mainField != "" {
		return collectMainFieldSource(util.ToUint64(record["data_template_cate_id"]), mainField)
	}
	if dataFieldID := util.ToUint64(record["data_field_id"]); dataFieldID > 0 {
		return fmt.Sprintf("%s%d", collectFieldSourceDataPrefix, dataFieldID)
	}
	return ""
}
