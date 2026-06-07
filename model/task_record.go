package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type TaskRecord struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:任务记录ID"`
	ResourceID     uint64    `dorm:"type:bigint;not null;comment:客户资产"`
	TaskID         uint64    `dorm:"type:bigint;not null;comment:任务"`
	TaskTemplateID uint64    `dorm:"type:bigint;not null;default:0;comment:任务模板"`
	StageID        uint64    `dorm:"type:bigint;not null;default:0;comment:阶段"`
	FlowNodeID     uint64    `dorm:"type:bigint;not null;default:0;comment:流程节点"`
	OperatorID     uint64    `dorm:"type:bigint;not null;default:0;comment:操作人"`
	RecordJSON     string    `dorm:"type:text;not null;default:'{}';comment:记录JSON"`
	ResultValue    string    `dorm:"type:varchar(64);not null;default:'';comment:结果"`
	Remark         string    `dorm:"type:text;not null;default:'';comment:备注"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type TaskRecordIndex struct {
	ResourceTime struct{} `index:"resource_id,created_at,id"`
	TaskTime     struct{} `index:"task_id,created_at,id"`
	OperatorTime struct{} `index:"operator_id,created_at,id"`
	ResultTime   struct{} `index:"result_value,created_at,id"`
}

func NewTaskRecordModel() *orm.Model[TaskRecord] {
	return orm.LoadModel[TaskRecord]("任务记录", "crm_task_record", orm.ModelConfig{
		Index:    TaskRecordIndex{},
		Order:    "id desc",
		Database: "default",
		Relations: []orm.Relation{
			resourceRelation,
			taskRelation,
			taskTemplateRelation,
			flowStageRelation,
			flowNodeRelation,
			operatorRelation,
		},
	})
}
