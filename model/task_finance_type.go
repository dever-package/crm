package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

// TaskFinanceType is retained for historical configuration audit only.
// New configuration and runtime reads use DataField.FinanceTypeID.
type TaskFinanceType struct {
	ID            uint64    `dorm:"primaryKey;autoIncrement;comment:任务财务类型关联ID"`
	TaskID        uint64    `dorm:"type:bigint;not null;comment:任务"`
	FinanceTypeID uint64    `dorm:"type:bigint;not null;comment:财务类型"`
	CreatedAt     time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type TaskFinanceTypeIndex struct {
	TaskFinance struct{} `unique:"task_id,finance_type_id"`
	FinanceTask struct{} `index:"finance_type_id,task_id,id"`
}

func NewTaskFinanceTypeModel() *orm.Model[TaskFinanceType] {
	return orm.LoadModel[TaskFinanceType]("任务财务类型历史", "crm_task_finance_type", orm.ModelConfig{
		Index:    TaskFinanceTypeIndex{},
		Order:    "id asc",
		Database: "default",
		Relations: []orm.Relation{
			taskRelation,
			financeTypeRelation,
		},
	})
}
