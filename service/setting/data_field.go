package setting

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	pinyin "github.com/mozillazg/go-pinyin"
	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

var dataFieldOptionTypes = map[string]struct{}{
	"radio":        {},
	"checkbox":     {},
	"select":       {},
	"multi_select": {},
}

const (
	dataFieldOptionSourceCustom    = "custom"
	dataFieldOptionSourceOptionSet = "option_set"
)

var crmDataFieldPinyinArgs = func() pinyin.Args {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	return args
}()

func (CrmHook) ProviderBeforeSaveDataField(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	return normalizeCrmDataFieldRecord(c, record, partial, nil)
}

func normalizeCrmDataFieldRecord(c *server.Context, record map[string]any, partial bool, reservedKeys map[string]bool) map[string]any {
	existing := existingCrmDataField(c, util.ToUint64(record["id"]))
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "field_key", partial)
	trimCrmStringField(record, "field_type", partial)
	trimCrmStringField(record, "default_value", partial)
	if !partial {
		if _, hasTemplateID := record["data_template_id"]; hasTemplateID && util.ToUint64(record["data_template_id"]) == 0 {
			panicCrmField("form.data_template_id", "数据模板不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "字段名称不能为空。")
		}
	}
	if shouldNormalizeCrmField(record, "field_type", partial) && util.ToStringTrimmed(record["field_type"]) == "" {
		record["field_type"] = "text"
	}
	optionSource := normalizeCrmDataFieldOptionSource(record, existing, partial)
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	normalizeCrmDataFieldParent(ctx, record, existing, partial)
	normalizeCrmDataFieldOptionSet(ctx, record, existing, partial, optionSource)
	normalizeCrmDataFieldCapabilities(ctx, record, existing, partial)
	ensureCrmDataFieldKey(ctx, record, existing, partial, reservedKeys)
	validateCrmDataFieldKey(ctx, record, existing, partial)
	reserveCrmDataFieldKey(reservedKeys, effectiveCrmDataFieldKey(record, existing))
	normalizeCrmDataFieldChildren(c, record, existing, partial, reservedKeys)
	normalizeCrmDataFieldOptions(ctx, record, existing, partial)
	defaultCrmInt(record, "parent_field_id", 0, partial)
	defaultCrmInt(record, "option_set_id", 0, partial)
	defaultCrmInt(record, "finance_type_id", 0, partial)
	defaultCrmBool(record, "stat_enabled", false, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	delete(record, "option_source")
	delete(record, "parent_field_key")
	delete(record, "data_template_key_prefix")
	return record
}

func normalizeCrmDataFieldCapabilities(ctx context.Context, record map[string]any, existing *crmmodel.DataField, partial bool) {
	fieldType := effectiveCrmDataFieldType(record, existing)
	unsupported := fieldType == "group" || fieldType == "attachment"
	fieldTypeChanged := shouldNormalizeCrmField(record, "field_type", partial)
	if unsupported && (fieldTypeChanged || !partial) {
		record["finance_type_id"] = uint64(0)
		record["stat_enabled"] = false
		return
	}

	if shouldNormalizeCrmField(record, "finance_type_id", partial) {
		financeTypeID := util.ToUint64(record["finance_type_id"])
		if financeTypeID > 0 && crmmodel.NewFinanceTypeModel().Find(ctx, map[string]any{
			"id":     financeTypeID,
			"status": crmmodel.StatusEnabled,
		}) == nil {
			panicCrmField("form.finance_type_id", "财务类型不存在或已停用。")
		}
		record["finance_type_id"] = financeTypeID
	}
	if shouldNormalizeCrmField(record, "stat_enabled", partial) {
		record["stat_enabled"] = configBool(record["stat_enabled"])
	}
}

func (CrmHook) ProviderBuildDataFieldForm(c *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	normalizeDataFieldFormOptions(c, record)
	normalizeDataFieldFormChildren(c, record)
	return record
}

func normalizeDataFieldFormOptions(c *server.Context, record map[string]any) {
	optionSetID := util.ToUint64(record["option_set_id"])
	record["option_source"] = dataFieldOptionSourceCustom
	if optionSetID > 0 {
		record["option_source"] = dataFieldOptionSourceOptionSet
		record["options"] = []map[string]any{}
		return
	}
	fieldID := util.ToUint64(record["id"])
	if fieldID == 0 {
		if options := formFieldRows(record["options"]); options != nil {
			record["options"] = options
		} else if _, exists := record["options"]; !exists {
			record["options"] = []map[string]any{}
		}
		return
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
	record["options"] = dataFieldPrivateOptionRows(ctx, field)
}

func normalizeCrmDataFieldOptionSource(record map[string]any, existing *crmmodel.DataField, partial bool) string {
	if !shouldNormalizeCrmField(record, "option_source", partial) &&
		!shouldNormalizeCrmField(record, "option_set_id", partial) &&
		!shouldNormalizeCrmField(record, "field_type", partial) {
		if effectiveCrmDataFieldOptionSetID(record, existing) > 0 {
			return dataFieldOptionSourceOptionSet
		}
		return dataFieldOptionSourceCustom
	}
	source := util.ToStringTrimmed(record["option_source"])
	if source == "" {
		if effectiveCrmDataFieldOptionSetID(record, existing) > 0 {
			source = dataFieldOptionSourceOptionSet
		} else {
			source = dataFieldOptionSourceCustom
		}
	}
	if source != dataFieldOptionSourceOptionSet {
		source = dataFieldOptionSourceCustom
		record["option_set_id"] = uint64(0)
	}
	return source
}

func ensureCrmDataFieldKey(ctx context.Context, record map[string]any, existing *crmmodel.DataField, partial bool, reservedKeys map[string]bool) {
	if !shouldNormalizeCrmField(record, "field_key", partial) &&
		!shouldNormalizeCrmField(record, "name", partial) &&
		!shouldNormalizeCrmField(record, "field_type", partial) &&
		!shouldNormalizeCrmField(record, "parent_field_id", partial) &&
		!shouldNormalizeCrmField(record, "data_template_id", partial) &&
		!shouldNormalizeCrmField(record, "data_template_key_prefix", partial) {
		return
	}
	fieldKey := effectiveCrmDataFieldKey(record, existing)
	if fieldKey != "" {
		record["field_key"] = fieldKey
		return
	}
	if existing != nil && strings.TrimSpace(existing.FieldKey) != "" && !shouldNormalizeCrmField(record, "field_key", partial) {
		record["field_key"] = strings.TrimSpace(existing.FieldKey)
		return
	}
	base := generatedCrmDataFieldKeyBase(
		effectiveCrmDataFieldName(record, existing),
		effectiveCrmDataFieldType(record, existing),
	)
	parentPrefix := crmDataFieldParentKeyPrefix(ctx, record, existing)
	if parentPrefix == "" {
		parentPrefix = crmDataFieldTemplateKeyPrefix(ctx, record, existing)
	}
	record["field_key"] = uniqueCrmDataFieldKey(ctx, parentPrefix, base, util.ToUint64(record["id"]), reservedKeys)
}

func generatedCrmDataFieldKeyBase(name string, fieldType string) string {
	base := crmDataFieldKeyText(name)
	if base != "" {
		return base
	}
	if strings.TrimSpace(fieldType) == "group" {
		return "group"
	}
	return "field"
}

func crmDataFieldKeyText(value string) string {
	return strings.Join(crmDataFieldKeyTokens(value, false), "")
}

func crmDataFieldKeyInitials(value string) string {
	return strings.Join(crmDataFieldKeyTokens(value, true), "")
}

func crmDataFieldKeyTokens(value string, initials bool) []string {
	var ascii strings.Builder
	tokens := make([]string, 0, 4)
	flushASCII := func() {
		if ascii.Len() == 0 {
			return
		}
		tokens = append(tokens, ascii.String())
		ascii.Reset()
	}
	for _, char := range strings.TrimSpace(value) {
		switch {
		case char >= 'a' && char <= 'z':
			ascii.WriteRune(char)
		case char >= 'A' && char <= 'Z':
			ascii.WriteRune(unicode.ToLower(char))
		case char >= '0' && char <= '9':
			ascii.WriteRune(char)
		case unicode.Is(unicode.Han, char):
			flushASCII()
			token := crmDataFieldPinyinToken(char)
			if initials && token != "" {
				token = token[:1]
			}
			tokens = append(tokens, token)
		default:
			flushASCII()
		}
	}
	flushASCII()
	return tokens
}

func crmDataFieldPinyinToken(char rune) string {
	values := pinyin.LazyPinyin(string(char), crmDataFieldPinyinArgs)
	if len(values) == 0 {
		return fmt.Sprintf("u%x", char)
	}
	token := crmDataFieldASCIIKeyToken(values[0])
	if token == "" {
		return fmt.Sprintf("u%x", char)
	}
	return token
}

func crmDataFieldASCIIKeyToken(value string) string {
	value = strings.NewReplacer("ü", "v", "Ü", "v", "u:", "v", "U:", "v").Replace(value)
	var builder strings.Builder
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func crmDataFieldParentKeyPrefix(ctx context.Context, record map[string]any, existing *crmmodel.DataField) string {
	if parentKey := util.ToStringTrimmed(record["parent_field_key"]); parentKey != "" {
		return parentKey
	}
	parentID := effectiveCrmDataFieldParentID(record, existing)
	if parentID == 0 {
		return ""
	}
	parent := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": parentID})
	if parent == nil {
		return ""
	}
	parentKey := strings.TrimSpace(parent.FieldKey)
	if parentKey == "" {
		panicCrmField("form.parent_field_id", "父级分组必须先配置字段编码。")
	}
	return parentKey
}

func crmDataFieldTemplateKeyPrefix(ctx context.Context, record map[string]any, existing *crmmodel.DataField) string {
	if prefix := crmDataFieldKeyInitials(util.ToStringTrimmed(record["data_template_key_prefix"])); prefix != "" {
		return prefix
	}
	templateID := effectiveCrmDataFieldTemplateID(record, existing)
	if templateID == 0 {
		return ""
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID})
	if template == nil {
		return ""
	}
	return crmDataFieldKeyInitials(template.Name)
}

func uniqueCrmDataFieldKey(ctx context.Context, parentPrefix string, base string, currentID uint64, reservedKeys map[string]bool) string {
	model := crmmodel.NewDataFieldModel()
	for index := 0; index < 1000; index++ {
		suffix := ""
		if index > 0 {
			suffix = fmt.Sprintf("_%d", index+1)
		}
		leaf := fitCrmDataFieldKeyLeaf(base, parentPrefix, suffix)
		candidate := leaf
		if parentPrefix != "" {
			candidate = parentPrefix + "." + leaf
		}
		if reservedKeys != nil && reservedKeys[candidate] {
			continue
		}
		field := model.Find(ctx, map[string]any{"field_key": candidate})
		if field == nil || field.ID == currentID {
			return candidate
		}
	}
	panicCrmField("form.field_key", "字段编码自动生成失败，请手动填写。")
	return ""
}

func fitCrmDataFieldKeyLeaf(base string, parentPrefix string, suffix string) string {
	maxLen := 128 - len(suffix)
	if parentPrefix != "" {
		maxLen -= len(parentPrefix) + 1
	}
	if maxLen <= 0 {
		panicCrmField("form.parent_field_id", "父级字段编码过长，无法自动生成子字段编码。")
	}
	leaf := strings.TrimSpace(base)
	if len(leaf) > maxLen {
		leaf = strings.Trim(leaf[:maxLen], "_-.")
	}
	if leaf == "" {
		leaf = "field"
	}
	return leaf + suffix
}

func reserveCrmDataFieldKey(reservedKeys map[string]bool, fieldKey string) {
	if reservedKeys == nil || fieldKey == "" {
		return
	}
	if reservedKeys[fieldKey] {
		panicCrmField("form.field_key", "字段编码必须全局唯一。")
	}
	reservedKeys[fieldKey] = true
}

func normalizeCrmDataFieldChildren(c *server.Context, record map[string]any, existing *crmmodel.DataField, partial bool, reservedKeys map[string]bool) {
	_, hasChildren := record["children"]
	fieldTypeChanged := shouldNormalizeCrmField(record, "field_type", partial)
	fieldType := effectiveCrmDataFieldType(record, existing)
	fieldID := util.ToUint64(record["id"])
	if fieldType != "group" {
		if fieldID > 0 && (hasChildren || fieldTypeChanged) {
			ctx := context.Background()
			if c != nil {
				ctx = c.Context()
			}
			crmmodel.NewDataFieldModel().Delete(ctx, map[string]any{"parent_field_id": fieldID})
		}
		delete(record, "children")
		return
	}
	if !hasChildren {
		return
	}
	rows := formFieldRows(record["children"])
	if rows == nil {
		record["children"] = []map[string]any{}
		return
	}
	templateID := effectiveCrmDataFieldTemplateID(record, existing)
	parentKey := effectiveCrmDataFieldKey(record, existing)
	normalized := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if util.ToStringTrimmed(row["field_type"]) == "group" {
			panicCrmField("form.children", "分组字段下不能再添加分组。")
		}
		if fieldID > 0 {
			row["parent_field_id"] = fieldID
		}
		if templateID > 0 {
			row["data_template_id"] = templateID
		}
		if parentKey != "" {
			row["parent_field_key"] = parentKey
		}
		normalized = append(normalized, normalizeCrmDataFieldRecord(c, row, false, reservedKeys))
	}
	record["children"] = normalized
}

type crmDataFieldOptionInput struct {
	name  string
	value string
	sort  int
}

func isDataFieldOptionType(fieldType string) bool {
	_, ok := dataFieldOptionTypes[fieldType]
	return ok
}

func normalizeCrmDataFieldParent(ctx context.Context, record map[string]any, existing *crmmodel.DataField, partial bool) {
	if !shouldNormalizeCrmField(record, "parent_field_id", partial) &&
		!shouldNormalizeCrmField(record, "data_template_id", partial) &&
		!shouldNormalizeCrmField(record, "field_type", partial) {
		return
	}
	currentID := util.ToUint64(record["id"])
	parentID := effectiveCrmDataFieldParentID(record, existing)
	fieldType := effectiveCrmDataFieldType(record, existing)
	if fieldType == "group" && parentID > 0 {
		panicCrmField("form.parent_field_id", "分组字段不能属于其他分组。")
	}
	if parentID == 0 {
		if shouldNormalizeCrmField(record, "parent_field_id", partial) {
			record["parent_field_id"] = uint64(0)
		}
		return
	}
	if parentID == currentID && currentID > 0 {
		panicCrmField("form.parent_field_id", "字段不能选择自己作为父级。")
	}
	parent := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":         parentID,
		"field_type": "group",
		"status":     crmmodel.StatusEnabled,
	})
	if parent == nil {
		panicCrmField("form.parent_field_id", "父级分组不存在或已停用。")
	}
	templateID := effectiveCrmDataFieldTemplateID(record, existing)
	if templateID == 0 || parent.DataTemplateID != templateID {
		panicCrmField("form.parent_field_id", "父级分组必须属于同一个数据模板。")
	}
	record["parent_field_id"] = parentID
}

func normalizeCrmDataFieldOptionSet(ctx context.Context, record map[string]any, existing *crmmodel.DataField, partial bool, optionSource string) {
	if !shouldNormalizeCrmField(record, "option_set_id", partial) &&
		!shouldNormalizeCrmField(record, "field_type", partial) &&
		!shouldNormalizeCrmField(record, "option_source", partial) {
		return
	}
	optionSetID := effectiveCrmDataFieldOptionSetID(record, existing)
	fieldType := effectiveCrmDataFieldType(record, existing)
	if !isDataFieldOptionType(fieldType) {
		record["option_set_id"] = uint64(0)
		return
	}
	if optionSource != dataFieldOptionSourceOptionSet {
		record["option_set_id"] = uint64(0)
		return
	}
	if optionSetID == 0 {
		panicCrmField("form.option_set_id", "请选择常用选项集。")
	}
	if crmmodel.NewOptionSetModel().Find(ctx, map[string]any{"id": optionSetID, "status": crmmodel.StatusEnabled}) == nil {
		panicCrmField("form.option_set_id", "常用选项集不存在或已停用。")
	}
	record["option_set_id"] = optionSetID
	if util.ToUint64(record["id"]) > 0 {
		crmmodel.NewDataFieldOptionModel().Delete(ctx, map[string]any{"data_field_id": util.ToUint64(record["id"])})
	}
	delete(record, "options")
}

func validateCrmDataFieldKey(ctx context.Context, record map[string]any, existing *crmmodel.DataField, partial bool) {
	if !shouldNormalizeCrmField(record, "field_key", partial) &&
		!shouldNormalizeCrmField(record, "data_template_id", partial) &&
		!shouldNormalizeCrmField(record, "parent_field_id", partial) {
		return
	}
	fieldKey := effectiveCrmDataFieldKey(record, existing)
	if fieldKey == "" {
		if !partial {
			panicCrmField("form.field_key", "字段编码不能为空。")
		}
		return
	}
	if !validDataFieldKey(fieldKey) {
		panicCrmField("form.field_key", "字段编码只能包含字母、数字、下划线、点和短横线。")
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"field_key": fieldKey,
	})
	if field != nil && field.ID != util.ToUint64(record["id"]) {
		panicCrmField("form.field_key", "字段编码必须全局唯一。")
	}
	validateCrmDataFieldKeyPath(ctx, record, existing, fieldKey)
}

func validateCrmDataFieldKeyPath(ctx context.Context, record map[string]any, existing *crmmodel.DataField, fieldKey string) {
	parentID := effectiveCrmDataFieldParentID(record, existing)
	if parentID == 0 {
		return
	}
	parent := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": parentID})
	if parent == nil {
		return
	}
	parentKey := strings.TrimSpace(parent.FieldKey)
	if parentKey == "" {
		panicCrmField("form.parent_field_id", "父级分组必须先配置字段编码。")
	}
	if !strings.HasPrefix(fieldKey, parentKey+".") {
		panicCrmField("form.field_key", "子字段编码必须以父级分组编码加点号开头。")
	}
}

func normalizeCrmDataFieldOptions(ctx context.Context, record map[string]any, existing *crmmodel.DataField, partial bool) {
	_, hasOptions := record["options"]
	fieldTypeChanged := shouldNormalizeCrmField(record, "field_type", partial)
	fieldType := effectiveCrmDataFieldType(record, existing)
	optionSetID := effectiveCrmDataFieldOptionSetID(record, existing)
	if optionSetID > 0 {
		if util.ToUint64(record["id"]) > 0 {
			crmmodel.NewDataFieldOptionModel().Delete(ctx, map[string]any{"data_field_id": util.ToUint64(record["id"])})
		}
		delete(record, "options")
		return
	}
	if !hasOptions && !(fieldTypeChanged && !isDataFieldOptionType(fieldType)) {
		return
	}
	fieldID := util.ToUint64(record["id"])
	if !isDataFieldOptionType(fieldType) {
		if fieldID > 0 {
			crmmodel.NewDataFieldOptionModel().Delete(ctx, map[string]any{"data_field_id": fieldID})
		}
		delete(record, "options")
		return
	}
	if fieldID == 0 || !hasOptions {
		if hasOptions {
			record["options"] = normalizeCrmDataFieldOptionRecords(record["options"])
		}
		return
	}
	syncCrmDataFieldOptions(ctx, fieldID, record["options"])
	delete(record, "options")
}

func syncCrmDataFieldOptions(ctx context.Context, fieldID uint64, rawOptions any) {
	records := normalizeCrmDataFieldOptionRecords(rawOptions)
	model := crmmodel.NewDataFieldOptionModel()
	model.Delete(ctx, map[string]any{"data_field_id": fieldID})
	for _, record := range records {
		option := util.CloneMap(record)
		option["data_field_id"] = fieldID
		model.Insert(ctx, option)
	}
}

func normalizeCrmDataFieldOptionRecords(rawOptions any) []map[string]any {
	inputs := normalizeCrmDataFieldOptionInputs(rawOptions)
	records := make([]map[string]any, 0, len(inputs))
	for _, input := range inputs {
		records = append(records, map[string]any{
			"name":  input.name,
			"value": input.value,
			"sort":  input.sort,
		})
	}
	return records
}

func normalizeCrmDataFieldOptionInputs(rawOptions any) []crmDataFieldOptionInput {
	rows := formFieldRows(rawOptions)
	inputs := make([]crmDataFieldOptionInput, 0, len(rows))
	seenValues := map[string]bool{}
	for index, row := range rows {
		if blankCrmDataFieldOptionRow(row) {
			continue
		}
		name := util.ToStringTrimmed(row["name"])
		value := util.ToStringTrimmed(row["value"])
		if name == "" {
			panicCrmField("form.options", "选项名不能为空。")
		}
		if value == "" {
			panicCrmField("form.options", "选项值不能为空。")
		}
		if seenValues[value] {
			panicCrmField("form.options", "选项值不能重复。")
		}
		seenValues[value] = true
		inputs = append(inputs, crmDataFieldOptionInput{
			name:  name,
			value: value,
			sort:  util.ToIntDefault(row["sort"], (index+1)*10),
		})
	}
	return inputs
}

func blankCrmDataFieldOptionRow(row map[string]any) bool {
	return util.ToUint64(row["id"]) == 0 &&
		util.ToStringTrimmed(row["name"]) == "" &&
		util.ToStringTrimmed(row["value"]) == ""
}

func existingCrmDataField(c *server.Context, id uint64) *crmmodel.DataField {
	if id == 0 {
		return nil
	}
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	return crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": id})
}

func effectiveCrmDataFieldKey(record map[string]any, existing *crmmodel.DataField) string {
	if _, ok := record["field_key"]; ok {
		return util.ToStringTrimmed(record["field_key"])
	}
	if existing == nil {
		return ""
	}
	return strings.TrimSpace(existing.FieldKey)
}

func effectiveCrmDataFieldName(record map[string]any, existing *crmmodel.DataField) string {
	if _, ok := record["name"]; ok {
		return util.ToStringTrimmed(record["name"])
	}
	if existing == nil {
		return ""
	}
	return strings.TrimSpace(existing.Name)
}

func effectiveCrmDataFieldTemplateID(record map[string]any, existing *crmmodel.DataField) uint64 {
	if _, ok := record["data_template_id"]; ok {
		return util.ToUint64(record["data_template_id"])
	}
	if existing == nil {
		return 0
	}
	return existing.DataTemplateID
}

func effectiveCrmDataFieldParentID(record map[string]any, existing *crmmodel.DataField) uint64 {
	if _, ok := record["parent_field_id"]; ok {
		return util.ToUint64(record["parent_field_id"])
	}
	if existing == nil {
		return 0
	}
	return existing.ParentFieldID
}

func effectiveCrmDataFieldOptionSetID(record map[string]any, existing *crmmodel.DataField) uint64 {
	if _, ok := record["option_set_id"]; ok {
		return util.ToUint64(record["option_set_id"])
	}
	if existing == nil {
		return 0
	}
	return existing.OptionSetID
}

func effectiveCrmDataFieldType(record map[string]any, existing *crmmodel.DataField) string {
	if _, ok := record["field_type"]; ok {
		return util.ToStringTrimmed(record["field_type"])
	}
	if existing == nil {
		return ""
	}
	return strings.TrimSpace(existing.FieldType)
}

func validDataFieldKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for _, char := range key {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		if char == '_' || char == '.' || char == '-' {
			continue
		}
		return false
	}
	return true
}
