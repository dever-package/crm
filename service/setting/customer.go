package setting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/orm"
	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
	crmservice "github.com/dever-package/crm/service"
)

func (CrmHook) ProviderBeforeSaveCustomer(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}

	partial := isPartialCrmRecord(record)
	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	trimCrmStringField(record, "name", partial)
	trimCrmStringField(record, "phone", partial)
	trimCrmStringField(record, "wechat", partial)
	trimCrmStringField(record, "id_card", partial)
	trimCrmStringField(record, "remark", partial)
	_, customerTagsSubmitted := record["tag_ids"]
	if rawTagIDs, submitted := record["tag_ids"]; submitted {
		needsSync, err := crmservice.CustomerTagsNeedSync(ctx, util.ToUint64(record["id"]), rawTagIDs)
		if err != nil {
			panicCrmField("form.tag_ids", err.Error())
		}
		if !needsSync {
			delete(record, "tag_ids")
		}
		delete(record, "tags")
		delete(record, "level_id")
	} else {
		trimCrmStringField(record, "tags", partial)
	}
	delete(record, "tag_options")
	if shouldNormalizeCrmField(record, "source_id", partial) && util.ToUint64(record["source_id"]) == 0 {
		record["source_id"] = crmmodel.DefaultCustomerSourceID
	}
	if shouldNormalizeCrmField(record, "channel_id", partial) && util.ToUint64(record["channel_id"]) == 0 {
		record["channel_id"] = crmmodel.DefaultCustomerChannelID
	}
	if !customerTagsSubmitted && shouldNormalizeCrmField(record, "level_id", partial) && util.ToUint64(record["level_id"]) == 0 {
		record["level_id"] = crmmodel.DefaultCustomerLevelID
	}

	if !partial {
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "客户姓名不能为空。")
		}
		if util.ToStringTrimmed(record["phone"]) == "" {
			panicCrmField("form.phone", "手机号不能为空。")
		}
	}

	code := util.ToStringTrimmed(record["code"])
	customerID := util.ToUint64(record["id"])
	if code == "" && customerID > 0 {
		if current := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}); current != nil {
			code = strings.TrimSpace(current.Code)
		}
	}
	if code == "" {
		generatedCode, err := crmmodel.GenerateUniqueCustomerCode(ctx)
		if err != nil {
			panicCrmField("form.code", err.Error())
		}
		code = generatedCode
	}
	record["code"] = code
	delete(record, "creator_id")
	return record
}

func (CrmHook) ProviderBuildCustomerForm(c *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	ctx := contextFromServer(c)
	record["tag_options"] = crmservice.CustomerTagOptions(ctx)
	record["tag_ids"] = crmservice.CustomerTagIDs(ctx, util.ToUint64(record["id"]))
	return record
}

func (CrmHook) ProviderAfterSaveCustomer(c *server.Context, params []any) any {
	if c == nil || len(params) == 0 {
		return nil
	}
	payload, ok := params[0].(map[string]any)
	if !ok {
		return nil
	}
	source, ok := payload["payload"].(map[string]any)
	if !ok {
		return nil
	}
	rawTagIDs, submitted := source["tag_ids"]
	if !submitted {
		return nil
	}
	if _, err := crmservice.SyncCustomerTags(c.Context(), savedRecordID(payload), rawTagIDs); err != nil {
		panicCrmField("form.tag_ids", err.Error())
	}
	return nil
}

func (CrmHook) ProviderBuildCustomerRows(c *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	if len(rows) == 0 {
		return rows
	}

	ctx := context.Background()
	if c != nil {
		ctx = c.Context()
	}
	prefix := customerCodePrefix(ctx)

	for _, row := range rows {
		code := strings.TrimSpace(util.ToString(row["code"]))
		if code != "" {
			row["code_display"] = prefix + code
		} else {
			row["code_display"] = ""
		}
		row["source_name"] = relationName(row, "source.name")
		row["channel_name"] = relationName(row, "channel.name")
		row["level_name"] = relationName(row, "level.name")
	}
	return rows
}

func (CrmHook) ProviderBeforeSaveCustomerSource(_ *server.Context, params []any) any {
	return normalizeNamedOptionRecord(params, "来源")
}

func (CrmHook) ProviderBeforeSaveCustomerChannel(_ *server.Context, params []any) any {
	return normalizeNamedOptionRecord(params, "渠道")
}

func (CrmHook) ProviderBeforeSaveCustomerLevel(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	if rawTags, submitted := record["tags"]; submitted {
		tags := normalizeCustomerLevelTagRows(rawTags)
		validateCustomerLevelTagRows(contextFromServer(c), util.ToUint64(record["id"]), tags)
		record["tags"] = tags
	}
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "等级名称不能为空。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	if partial {
		return record
	}
	code := util.ToStringTrimmed(record["code"])
	if code == "" && util.ToUint64(record["id"]) > 0 {
		if current := crmmodel.NewCustomerLevelModel().Find(contextFromServer(c), map[string]any{"id": util.ToUint64(record["id"])}); current != nil {
			code = strings.TrimSpace(current.Code)
		}
	}
	if code == "" {
		record["code"] = uniqueCustomerLevelCode()
	} else {
		record["code"] = code
	}
	return record
}

func (CrmHook) ProviderBuildCustomerLevelForm(c *server.Context, params []any) any {
	record := formConfigRecord(params)
	if len(record) == 0 {
		return record
	}
	rows := crmmodel.NewCustomerTagModel().SelectMap(contextFromServer(c), map[string]any{
		"level_id": util.ToUint64(record["id"]),
	})
	for _, row := range rows {
		defaultCrmInt16(row, "status", crmmodel.StatusEnabled, false)
		defaultCrmInt(row, "sort", 100, false)
	}
	record["tags"] = rows
	return record
}

func (CrmHook) ProviderBeforeSaveCustomerTag(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialOrInlineCrmRecord(record, "status", "sort")
	trimCrmStringField(record, "name", partial)
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "标签名称不能为空。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderAfterSaveCustomerLevel(c *server.Context, params []any) any {
	if c == nil || len(params) == 0 {
		return nil
	}
	payload, ok := params[0].(map[string]any)
	if !ok {
		return nil
	}
	source, ok := payload["payload"].(map[string]any)
	if !ok {
		return nil
	}
	rawTags, submitted := source["tags"]
	if !submitted {
		return nil
	}
	levelID := savedRecordID(payload)
	tags := normalizeCustomerLevelTagRows(rawTags)
	validateCustomerLevelTagRows(c.Context(), levelID, tags)
	if err := orm.Transaction(c.Context(), func(txCtx context.Context) error {
		model := crmmodel.NewCustomerTagModel()
		existingTags := model.Select(txCtx, map[string]any{"level_id": levelID})
		retained := map[uint64]bool{}
		now := time.Now()
		for _, tag := range tags {
			row := util.CloneMap(tag)
			tagID := util.ToUint64(row["id"])
			delete(row, "id")
			row["level_id"] = levelID
			row["updated_at"] = now
			if tagID > 0 {
				model.Update(txCtx, map[string]any{"id": tagID, "level_id": levelID}, row)
				retained[tagID] = true
				continue
			}
			if existing := model.Find(txCtx, map[string]any{"level_id": levelID, "name": row["name"]}); existing != nil {
				model.Update(txCtx, map[string]any{"id": existing.ID}, row)
				retained[existing.ID] = true
				continue
			}
			row["created_at"] = now
			model.Insert(txCtx, row)
		}
		for _, existing := range existingTags {
			if existing != nil && !retained[existing.ID] && existing.Status != crmmodel.StatusDisabled {
				model.Update(txCtx, map[string]any{"id": existing.ID}, map[string]any{
					"status":     crmmodel.StatusDisabled,
					"updated_at": now,
				})
			}
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return nil
}

func normalizeCustomerLevelTagRows(raw any) []map[string]any {
	rows := formFieldRows(raw)
	result := make([]map[string]any, 0, len(rows))
	seen := map[string]bool{}
	for _, source := range rows {
		name := util.ToStringTrimmed(source["name"])
		if name == "" {
			panicCrmField("form.tags", "标签名称不能为空。")
		}
		if seen[name] {
			panicCrmField("form.tags", "同一客户等级下的标签名称不能重复。")
		}
		seen[name] = true
		row := map[string]any{
			"id":     util.ToUint64(source["id"]),
			"name":   name,
			"status": util.ToIntDefault(source["status"], 0),
			"sort":   util.ToIntDefault(source["sort"], 0),
		}
		defaultCrmInt16(row, "status", crmmodel.StatusEnabled, false)
		defaultCrmInt(row, "sort", 100, false)
		result = append(result, row)
	}
	return result
}

func validateCustomerLevelTagRows(ctx context.Context, levelID uint64, rows []map[string]any) {
	for _, row := range rows {
		tagID := util.ToUint64(row["id"])
		if tagID == 0 {
			continue
		}
		if levelID == 0 || crmmodel.NewCustomerTagModel().Find(ctx, map[string]any{
			"id":       tagID,
			"level_id": levelID,
		}) == nil {
			panicCrmField("form.tags", "客户标签不属于当前客户等级。")
		}
	}
}

func normalizeNamedOptionRecord(params []any, label string) map[string]any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "code", partial)
	trimCrmStringField(record, "name", partial)
	if !partial {
		if util.ToStringTrimmed(record["code"]) == "" {
			panicCrmField("form.code", label+"标识不能为空。")
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", label+"名称不能为空。")
		}
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func rowsFromProviderParams(params []any) []map[string]any {
	if len(params) == 0 {
		return nil
	}
	payload, ok := params[0].(map[string]any)
	if !ok {
		return nil
	}
	switch rows := payload["rows"].(type) {
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
	default:
		return nil
	}
}

func relationName(row map[string]any, key string) string {
	return strings.TrimSpace(util.ToString(row[key]))
}

func uniqueCustomerLevelCode() string {
	return fmt.Sprintf("level_%d", time.Now().UnixNano())
}
