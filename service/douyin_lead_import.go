package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

const (
	douyinLeadSourceCode  = "ad_feed"
	douyinLeadChannelCode = "douyin"
)

var douyinDataFieldKeys = []string{
	"douyin_account_name",
	"douyin_promotion_name",
	"douyin_product_name",
	"douyin_entered_at",
	"douyin_assigned_at",
	"douyin_owner_name",
	"douyin_stage",
	"douyin_follow_status",
	"douyin_tags",
	"douyin_follow_record",
	"douyin_traffic_type",
	"douyin_lead_cost",
}

type douyinLeadImportResult struct {
	Created   bool
	Updated   bool
	Unchanged bool
	Skipped   bool
}

type douyinMappedLead struct {
	ExternalID  string
	Name        string
	Phone       string
	Wechat      string
	City        string
	InitialNeed string
	DataFields  map[string]any
	Raw         map[string]any
}

func importDouyinLead(
	ctx context.Context,
	credentials douyinCredentials,
	workflow *crmmodel.Workflow,
	source *crmmodel.CustomerSource,
	channel *crmmodel.CustomerChannel,
	fieldIDs map[string]uint64,
	raw map[string]any,
) (douyinLeadImportResult, error) {
	mapped := mapDouyinLead(raw, credentials.ClientSecret)
	if mapped.ExternalID == "" || mapped.Name == "" || mapped.Phone == "" && mapped.Wechat == "" {
		return douyinLeadImportResult{Skipped: true}, nil
	}
	leadModel := crmmodel.NewLeadModel()
	existing := leadModel.Find(ctx, map[string]any{
		"source_id":   source.ID,
		"external_id": mapped.ExternalID,
	})
	dataValues := douyinLeadDataValues(fieldIDs, mapped.DataFields)
	if existing != nil {
		changed := updateExistingDouyinLead(ctx, existing, mapped, dataValues)
		return douyinLeadImportResult{Updated: changed, Unchanged: !changed}, nil
	}

	payload := map[string]any{
		"name":          mapped.Name,
		"phone":         mapped.Phone,
		"wechat":        mapped.Wechat,
		"source_id":     source.ID,
		"channel_id":    channel.ID,
		"external_id":   mapped.ExternalID,
		"city":          mapped.City,
		"initial_need":  mapped.InitialNeed,
		"data_values":   dataValues,
		"import_source": "douyin_laike",
		"douyin_raw":    mapped.Raw,
	}
	if _, err := createWorkLead(ctx, workflow, nil, 0, payload); err != nil {
		return douyinLeadImportResult{}, err
	}
	return douyinLeadImportResult{Created: true}, nil
}

func loadDouyinLeadDependencies(ctx context.Context) (
	*crmmodel.Workflow,
	*crmmodel.CustomerSource,
	*crmmodel.CustomerChannel,
	map[string]uint64,
	error,
) {
	workflow := workflowForSubject(ctx, 0, crmmodel.WorkflowSubjectLead)
	if workflow == nil {
		return nil, nil, nil, nil, fmt.Errorf("缺少启用的默认线索流程")
	}
	source := crmmodel.NewCustomerSourceModel().Find(ctx, map[string]any{
		"code":   douyinLeadSourceCode,
		"status": crmmodel.StatusEnabled,
	})
	if source == nil {
		return nil, nil, nil, nil, fmt.Errorf("缺少启用的“信息流投放”线索来源")
	}
	channel := crmmodel.NewCustomerChannelModel().Find(ctx, map[string]any{
		"code":   douyinLeadChannelCode,
		"status": crmmodel.StatusEnabled,
	})
	if channel == nil {
		return nil, nil, nil, nil, fmt.Errorf("缺少启用的“抖音”线索渠道")
	}
	fieldIDs := map[string]uint64{}
	fieldModel := crmmodel.NewDataFieldModel()
	for _, fieldKey := range douyinDataFieldKeys {
		field := fieldModel.Find(ctx, map[string]any{
			"field_key": fieldKey,
			"status":    crmmodel.StatusEnabled,
		})
		if field != nil {
			fieldIDs[fieldKey] = field.ID
		}
	}
	return workflow, source, channel, fieldIDs, nil
}

func douyinLeadDataValues(fieldIDs map[string]uint64, values map[string]any) map[string]any {
	result := map[string]any{}
	for fieldKey, value := range values {
		fieldID := fieldIDs[fieldKey]
		if fieldID == 0 || emptyWorkFieldValue(value) {
			continue
		}
		result[fmt.Sprintf("data:%d", fieldID)] = value
	}
	return result
}

func updateExistingDouyinLead(
	ctx context.Context,
	lead *crmmodel.Lead,
	mapped douyinMappedLead,
	dataValues map[string]any,
) bool {
	if lead == nil {
		return false
	}
	updates := map[string]any{}
	record := mapFromAny(lead.RecordJSON)
	existingValues := workLeadDataValues(lead)
	dataChanged := false
	for key, value := range dataValues {
		if reflect.DeepEqual(existingValues[key], value) {
			continue
		}
		existingValues[key] = value
		dataChanged = true
	}
	if dataChanged {
		record["data_values"] = existingValues
		updates["record_json"] = jsonText(record)
	}
	if lead.City == "" && mapped.City != "" {
		updates["city"] = mapped.City
	}
	if lead.InitialNeed == "" && mapped.InitialNeed != "" {
		updates["initial_need"] = mapped.InitialNeed
	}
	snapshot := jsonText(map[string]any{
		"import_source": "douyin_laike",
		"douyin_raw":    mapped.Raw,
	})
	if strings.TrimSpace(lead.InputSnapshotJSON) != snapshot {
		updates["input_snapshot_json"] = snapshot
	}
	if len(updates) == 0 {
		return false
	}
	updates["updated_at"] = time.Now()
	return crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": lead.ID}, updates) > 0
}

func mapDouyinLead(raw map[string]any, clientSecret string) douyinMappedLead {
	encryptedPhone := douyinPayloadText(raw, "telephone", "mobile", "tel")
	phone := normalizeWorkLeadPhone(decryptDouyinLegacyText(encryptedPhone, clientSecret))
	encryptedWechat := douyinPayloadText(raw, "weixin", "wx", "wechat")
	wechat := decryptDouyinLegacyText(encryptedWechat, clientSecret)
	name := douyinPayloadText(raw, "name")
	if name == "" {
		name = douyinFallbackLeadName(phone, wechat)
	}
	accountName := douyinPayloadText(
		raw,
		"account_name",
		"advertiser_name",
		"follow_life_account_name",
		"intention_life_account_name",
	)
	enteredAt := douyinPayloadText(raw, "create_time_detail", "create_time", "clue_create_time")
	assignedAt := douyinPayloadText(
		raw,
		"assign_time",
		"assigned_time",
		"allocation_time",
		"allocated_time",
		"distribute_time",
		"modify_time",
	)
	stage, followStatus := douyinClueStageAndStatus(raw)
	trafficType := douyinPayloadText(raw, "traffic_type", "flow_type")
	if trafficType == "1" {
		trafficType = "自然流量"
	} else if trafficType != "" && trafficType != "自然流量" && trafficType != "营销流量" {
		trafficType = "营销流量"
	}
	productName := douyinPayloadText(raw, "product_name")
	return douyinMappedLead{
		ExternalID:  douyinPayloadText(raw, "clue_id"),
		Name:        name,
		Phone:       phone,
		Wechat:      strings.TrimSpace(wechat),
		City:        douyinPayloadText(raw, "city_name", "auto_city_name"),
		InitialNeed: douyinInitialNeed(raw),
		Raw:         raw,
		DataFields: map[string]any{
			"douyin_account_name":   accountName,
			"douyin_promotion_name": douyinPayloadText(raw, "promotion_name"),
			"douyin_product_name":   productName,
			"douyin_entered_at":     normalizeDouyinDateTime(enteredAt),
			"douyin_assigned_at":    normalizeDouyinDateTime(assignedAt),
			"douyin_owner_name":     douyinPayloadText(raw, "clue_owner_name", "owner_name", "follow_user_name"),
			"douyin_stage":          stage,
			"douyin_follow_status":  followStatus,
			"douyin_tags":           douyinPayloadText(raw, "tags", "system_tags"),
			"douyin_follow_record":  douyinPayloadText(raw, "remark_dict", "remark", "follow_record", "followRecord"),
			"douyin_traffic_type":   trafficType,
			"douyin_lead_cost": douyinPayloadText(
				raw,
				"lead_cost",
				"single_customer_cost",
				"customer_cost",
				"mkt_cost",
				"cost",
			),
		},
	}
}

func douyinInitialNeed(raw map[string]any) string {
	parts := make([]string, 0, 4)
	seen := map[string]bool{}
	for _, item := range []struct {
		Label string
		Value string
	}{
		{Label: "产品", Value: douyinPayloadText(raw, "product_name")},
		{Label: "意向", Value: douyinPayloadText(raw, "clue_intention")},
		{Label: "业务", Value: douyinPayloadText(raw, "business")},
		{Label: "备注", Value: douyinPayloadText(raw, "remark")},
	} {
		if item.Value == "" || seen[item.Value] {
			continue
		}
		seen[item.Value] = true
		parts = append(parts, item.Label+"："+item.Value)
	}
	return strings.Join(parts, "；")
}

func douyinClueStageAndStatus(raw map[string]any) (string, string) {
	effective := douyinPayloadText(raw, "effective_state", "clue_state")
	stage := map[string]string{
		"0":   "新线索",
		"1":   "有意向",
		"2":   "成交",
		"3":   "无效",
		"6":   "已加微信",
		"7":   "待再次沟通",
		"204": "到店",
	}[effective]
	if stage == "" {
		stage = douyinPayloadText(raw, "effective_state_name", "clue_state_name", "clue_stage")
	}
	follow := douyinPayloadText(raw, "follow_state_name", "follow_state")
	if label := map[string]string{
		"0": "待联系", "1": "未接通", "2": "已接通", "3": "有效沟通", "4": "深度沟通",
		"NOT_CALLED": "待联系", "NOT_ANSWERED": "未接通", "SHORT_ANSWERED": "已接通",
		"ANSWERED": "有效沟通", "DEEP_ANSWERED": "深度沟通",
	}[strings.ToUpper(follow)]; label != "" {
		follow = label
	}
	allocation := douyinPayloadText(raw, "allocation_status")
	if label := map[string]string{
		"0": "待分配", "1": "已分配", "2": "已分配",
		"NOT_ASSIGN": "待分配", "ASSIGNED": "已分配",
	}[strings.ToUpper(allocation)]; label != "" {
		allocation = label
	}
	return stage, joinDouyinText(follow, allocation)
}

func douyinPayloadText(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if text := douyinValueText(payload[key]); text != "" {
			return text
		}
	}
	return ""
}

func douyinValueText(value any) string {
	switch current := value.(type) {
	case nil:
		return ""
	case string:
		text := strings.TrimSpace(current)
		if text == "" {
			return ""
		}
		if strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
			var decoded any
			if json.Unmarshal([]byte(text), &decoded) == nil {
				if normalized := douyinValueText(decoded); normalized != "" {
					return normalized
				}
			}
		}
		return text
	case []any:
		parts := make([]string, 0, len(current))
		for _, item := range current {
			if text := douyinValueText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "；")
	case map[string]any:
		encoded, _ := json.Marshal(current)
		return strings.TrimSpace(string(encoded))
	default:
		return strings.TrimSpace(fmt.Sprint(current))
	}
}

func joinDouyinText(values ...string) string {
	parts := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		parts = append(parts, value)
	}
	return strings.Join(parts, "；")
}

func normalizeDouyinDateTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if timestamp, err := strconv.ParseInt(value, 10, 64); err == nil {
		if timestamp < 100000000000 {
			timestamp *= 1000
		}
		return time.UnixMilli(timestamp).In(douyinLocation()).Format("2006-01-02T15:04")
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
	} {
		var parsed time.Time
		var err error
		if layout == time.RFC3339 {
			parsed, err = time.Parse(layout, value)
		} else {
			parsed, err = time.ParseInLocation(layout, value, douyinLocation())
		}
		if err == nil {
			return parsed.In(douyinLocation()).Format("2006-01-02T15:04")
		}
	}
	return value
}

func douyinFallbackLeadName(phone, wechat string) string {
	identifier := phone
	if identifier == "" {
		identifier = wechat
	}
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return ""
	}
	runes := []rune(identifier)
	if len(runes) > 4 {
		runes = runes[len(runes)-4:]
	}
	return "抖音线索" + string(runes)
}

func decryptDouyinLegacyText(value string, clientSecret string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "Enc.") {
		return value
	}
	ciphertext, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return value
	}
	secret := normalizeDouyinSecret(clientSecret)
	if len(secret) != 32 {
		return value
	}
	block, err := aes.NewCipher([]byte(secret))
	if err != nil {
		return value
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, []byte(secret[16:])).CryptBlocks(plaintext, ciphertext)
	plaintext, ok := unpadDouyinPKCS7(plaintext)
	if !ok {
		return value
	}
	text := strings.TrimSpace(string(plaintext))
	if text == "" {
		return value
	}
	return text
}

func normalizeDouyinSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if len(secret) < 32 {
		right := (32 - len(secret)) / 2
		left := 32 - len(secret) - right
		return strings.Repeat("#", left) + secret + strings.Repeat("#", right)
	}
	if len(secret) > 32 {
		right := (len(secret) - 32) / 2
		left := len(secret) - 32 - right
		return secret[left : left+32]
	}
	return secret
}

func unpadDouyinPKCS7(value []byte) ([]byte, bool) {
	if len(value) == 0 {
		return nil, false
	}
	padding := int(value[len(value)-1])
	if padding <= 0 || padding > aes.BlockSize || padding > len(value) {
		return nil, false
	}
	for _, item := range value[len(value)-padding:] {
		if int(item) != padding {
			return nil, false
		}
	}
	return value[:len(value)-padding], true
}

func douyinLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}
