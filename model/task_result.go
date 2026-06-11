package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type TaskResult struct {
	ID              uint64    `dorm:"primaryKey;autoIncrement;comment:任务结果ID"`
	TaskID          uint64    `dorm:"type:bigint;not null;comment:任务"`
	Name            string    `dorm:"type:varchar(64);not null;comment:结果名称"`
	ResultValue     string    `dorm:"type:varchar(64);not null;comment:结果值"`
	IsSuccess       bool      `dorm:"not null;default:false;comment:是否成功结果"`
	RequiresComment bool      `dorm:"not null;default:false;comment:是否要求备注"`
	Sort            int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status          int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt       time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt       time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type TaskResultIndex struct {
	TaskValue  struct{} `unique:"task_id,result_value"`
	TaskStatus struct{} `index:"task_id,status,sort,id"`
}

func NewTaskResultModel() *orm.Model[TaskResult] {
	return orm.LoadModel[TaskResult]("任务结果", "crm_task_result", orm.ModelConfig{
		Index:    TaskResultIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{taskRelation},
	})
}
