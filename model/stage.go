package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Stage struct {
	ID                uint64    `dorm:"primaryKey;autoIncrement;comment:阶段ID"`
	WorkflowID        uint64    `dorm:"type:bigint;not null;default:0;comment:所属流程"`
	Name              string    `dorm:"type:varchar(128);not null;comment:阶段名称"`
	OwnerDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:默认负责部门"`
	Sort              int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status            int16     `dorm:"type:smallint;not null;default:2;comment:状态"`
	CreatedAt         time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt         time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type StageIndex struct {
	WorkflowStatus struct{} `index:"workflow_id,status,sort,id"`
	OwnerStatus    struct{} `index:"owner_department_id,status,sort,id"`
}

func NewStageModel() *orm.Model[Stage] {
	return orm.LoadModel[Stage]("阶段配置", "crm_stage", orm.ModelConfig{
		Index:    StageIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			workflowRelation,
			ownerDepartmentRelation,
		},
	})
}
