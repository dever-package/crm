package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Timeline struct {
	ID         uint64    `dorm:"primaryKey;autoIncrement;comment:时间线ID"`
	ResourceID uint64    `dorm:"type:bigint;not null;comment:客户资产"`
	TaskID     uint64    `dorm:"type:bigint;not null;default:0;comment:任务"`
	EventType  string    `dorm:"type:varchar(64);not null;comment:事件类型"`
	Title      string    `dorm:"type:varchar(128);not null;comment:标题"`
	Content    string    `dorm:"type:text;not null;default:'';comment:内容"`
	OperatorID uint64    `dorm:"type:bigint;not null;default:0;comment:操作人"`
	CreatedAt  time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type TimelineIndex struct {
	ResourceTime struct{} `index:"resource_id,created_at,id"`
	TaskTime     struct{} `index:"task_id,created_at,id"`
	EventTime    struct{} `index:"event_type,created_at,id"`
}

func NewTimelineModel() *orm.Model[Timeline] {
	return orm.LoadModel[Timeline]("CRM时间线", "crm_timeline", orm.ModelConfig{
		Index:    TimelineIndex{},
		Order:    "id desc",
		Database: "default",
		Relations: []orm.Relation{
			resourceRelation,
			taskRelation,
			operatorRelation,
		},
	})
}
