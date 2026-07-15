package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Workflow struct {
	ID           uint64    `dorm:"primaryKey;autoIncrement;comment:流程ID"`
	Name         string    `dorm:"type:varchar(128);not null;comment:流程名称"`
	SubjectType  string    `dorm:"type:varchar(32);not null;default:'customer_asset';comment:流程对象"`
	DefaultEntry bool      `dorm:"not null;default:false;comment:默认入口流程"`
	Sort         int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status       int16     `dorm:"type:smallint;not null;default:2;comment:状态"`
	CreatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type WorkflowIndex struct {
	StatusSort   struct{} `index:"status,sort,id"`
	SubjectSort  struct{} `index:"subject_type,status,sort,id"`
	DefaultEntry struct{} `index:"subject_type,default_entry,status,id"`
}

func NewWorkflowModel() *orm.Model[Workflow] {
	return orm.LoadModel[Workflow]("流程配置", "crm_workflow", orm.ModelConfig{
		Index:    WorkflowIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"subject_type": workflowSubjectTypeOptions,
			"status":       statusOptions,
		},
	})
}
