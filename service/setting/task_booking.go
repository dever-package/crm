package setting

import (
	"context"

	"github.com/shemic/dever/util"

	crmmodel "github.com/dever-package/crm/model"
)

func effectiveTaskBookingResourceCateID(ctx context.Context, record map[string]any, partial bool) uint64 {
	if shouldNormalizeCrmField(record, "booking_resource_cate_id", partial) {
		return normalizeTaskBookingResourceCateID(record["booking_resource_cate_id"])
	}
	return normalizeTaskBookingResourceCateID(currentTaskConfigValue(ctx, record, "resource_cate_id"))
}

func normalizeTaskBookingResourceCateID(value any) uint64 {
	id := util.ToUint64(value)
	if id == 0 {
		return crmmodel.DefaultResourceCateID
	}
	return id
}

func effectiveTaskBookingNeedConfirm(ctx context.Context, record map[string]any, partial bool) bool {
	if shouldNormalizeCrmField(record, "booking_need_confirm", partial) {
		return util.ToBool(record["booking_need_confirm"])
	}
	return util.ToBool(currentTaskConfigValue(ctx, record, "need_confirm"))
}
