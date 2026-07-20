package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

// TaskStatField is retained for historical configuration audit only.
// New configuration and runtime reads use DataField.StatEnabled.
type TaskStatField struct {
	ID          uint64    `dorm:"primaryKey;autoIncrement;comment:任务统计字段关联ID"`
	TaskID      uint64    `dorm:"type:bigint;not null;comment:任务"`
	DataFieldID uint64    `dorm:"type:bigint;not null;comment:数据字段"`
	CreatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type TaskStatFieldIndex struct {
	TaskField struct{} `unique:"task_id,data_field_id"`
	FieldTask struct{} `index:"data_field_id,task_id,id"`
}

func NewTaskStatFieldModel() *orm.Model[TaskStatField] {
	return orm.LoadModel[TaskStatField]("任务统计字段历史", "crm_task_stat_field", orm.ModelConfig{
		Index:    TaskStatFieldIndex{},
		Order:    "id asc",
		Database: "default",
		Relations: []orm.Relation{
			taskRelation,
			dataFieldRelation,
		},
	})
}
