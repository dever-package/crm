package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type LeadInvalidReason struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:无效原因ID"`
	Name      string    `dorm:"type:varchar(64);not null;default:'';comment:原因"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type LeadInvalidReasonIndex struct {
	StatusSort struct{} `index:"status,sort,id"`
}

var leadInvalidReasonSeeds = []map[string]any{
	{"id": uint64(1), "name": "空号或错号", "status": StatusEnabled, "sort": 10},
	{"id": uint64(2), "name": "测试数据", "status": StatusEnabled, "sort": 20},
	{"id": uint64(3), "name": "非目标客户", "status": StatusEnabled, "sort": 30},
	{"id": uint64(4), "name": "区域不符", "status": StatusEnabled, "sort": 40},
}

func NewLeadInvalidReasonModel() *orm.Model[LeadInvalidReason] {
	return orm.LoadModel[LeadInvalidReason]("线索无效原因", "crm_lead_invalid_reason", orm.ModelConfig{
		Index:    LeadInvalidReasonIndex{},
		Seeds:    leadInvalidReasonSeeds,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
