package setting

import (
	"context"
	"fmt"
	"time"

	"github.com/shemic/dever/server"
	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func (CrmHook) ProviderBeforeSavePublicResourceCate(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	if !partial && util.ToStringTrimmed(record["name"]) == "" {
		panicCrmField("form.name", "分类名称不能为空。")
	}
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBeforeSavePublicResource(_ *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "name", partial)
	if !partial {
		if util.ToUint64(record["resource_cate_id"]) == 0 {
			record["resource_cate_id"] = crmmodel.DefaultResourceCateID
		}
		if util.ToStringTrimmed(record["name"]) == "" {
			panicCrmField("form.name", "资源名称不能为空。")
		}
	}
	defaultCrmInt(record, "capacity", 0, partial)
	defaultCrmInt16(record, "status", crmmodel.StatusEnabled, partial)
	defaultCrmInt(record, "sort", 100, partial)
	return record
}

func (CrmHook) ProviderBeforeSavePublicResourceBooking(c *server.Context, params []any) any {
	record := cloneCrmRecord(params)
	if len(record) == 0 {
		return record
	}
	partial := isPartialCrmRecord(record)
	trimCrmStringField(record, "title", partial)
	trimCrmStringField(record, "remark", partial)
	trimCrmStringField(record, "booking_status", partial)
	if !partial {
		if util.ToUint64(record["resource_id"]) == 0 {
			panicCrmField("form.resource_id", "公共资源不能为空。")
		}
		if util.ToUint64(record["customer_id"]) == 0 {
			panicCrmField("form.customer_id", "客户不能为空。")
		}
		if util.ToStringTrimmed(record["title"]) == "" {
			panicCrmField("form.title", "用途不能为空。")
		}
	}
	normalizeBookingTimeField(record, "start_at", partial)
	normalizeBookingTimeField(record, "end_at", partial)
	if !partial || shouldNormalizeCrmField(record, "booking_status", partial) {
		if util.ToStringTrimmed(record["booking_status"]) == "" {
			record["booking_status"] = crmmodel.ResourceBookingStatusReserved
		}
	}
	if util.ToUint64(record["resource_id"]) > 0 && record["start_at"] != nil && record["end_at"] != nil {
		startAt, startOK := record["start_at"].(time.Time)
		endAt, endOK := record["end_at"].(time.Time)
		if startOK && endOK {
			if err := validateResourceBookingTime(c.Context(), util.ToUint64(record["id"]), util.ToUint64(record["resource_id"]), startAt, endAt); err != nil {
				panicCrmField("form.start_at", err.Error())
			}
		}
	}
	defaultCrmInt(record, "asset_id", 0, partial)
	defaultCrmInt(record, "task_id", 0, partial)
	defaultCrmInt(record, "operation_log_id", 0, partial)
	defaultCrmInt(record, "booker_staff_id", 0, partial)
	defaultCrmInt(record, "booker_department_id", 0, partial)
	defaultCrmInt(record, "approved_by_staff_id", 0, partial)
	return record
}

func (CrmHook) ProviderBuildPublicResourceBookingRows(_ *server.Context, params []any) any {
	rows := rowsFromProviderParams(params)
	for _, row := range rows {
		row["booking_status_name"] = resourceBookingStatusName(row["booking_status"])
	}
	return rows
}

func normalizeBookingTimeField(record map[string]any, field string, partial bool) {
	if partial {
		if _, exists := record[field]; !exists {
			return
		}
	}
	value := util.ToStringTrimmed(record[field])
	if value == "" {
		if !partial {
			panicCrmField("form."+field, "预定时间不能为空。")
		}
		return
	}
	parsed, err := parseWorkDateTime(value)
	if err != nil {
		panicCrmField("form."+field, "预定时间格式错误。")
	}
	record[field] = parsed
}

func validateResourceBookingTime(ctx context.Context, currentID uint64, resourceID uint64, startAt time.Time, endAt time.Time) error {
	if !endAt.After(startAt) {
		return fmt.Errorf("结束时间必须晚于开始时间")
	}
	for _, booking := range crmmodel.NewPublicResourceBookingModel().Select(ctx, map[string]any{"resource_id": resourceID}) {
		if booking == nil || booking.ID == currentID || resourceBookingCanceled(booking.BookingStatus) {
			continue
		}
		if startAt.Before(booking.EndAt) && endAt.After(booking.StartAt) {
			return fmt.Errorf("该资源在所选时间已被预定")
		}
	}
	return nil
}

func resourceBookingCanceled(status string) bool {
	return status == crmmodel.ResourceBookingStatusCanceled || status == crmmodel.ResourceBookingStatusRejected
}

func resourceBookingStatusName(value any) string {
	switch util.ToStringTrimmed(value) {
	case crmmodel.ResourceBookingStatusPending:
		return "待确认"
	case crmmodel.ResourceBookingStatusReserved:
		return "已预定"
	case crmmodel.ResourceBookingStatusCanceled:
		return "已取消"
	case crmmodel.ResourceBookingStatusRejected:
		return "已拒绝"
	case crmmodel.ResourceBookingStatusDone:
		return "已完成"
	default:
		return util.ToStringTrimmed(value)
	}
}

func parseWorkDateTime(value string) (time.Time, error) {
	value = util.ToStringTrimmed(value)
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid datetime")
}
