package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type CustomerStage struct {
	ID                uint64    `dorm:"primaryKey;autoIncrement;comment:资产进度ID"`
	CustomerID        uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID           uint64    `dorm:"type:bigint;not null;comment:客户资产"`
	WorkflowID        uint64    `dorm:"type:bigint;not null;comment:当前流程"`
	StageID           uint64    `dorm:"type:bigint;not null;comment:当前阶段"`
	OwnerDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:负责部门"`
	OwnerStaffID      uint64    `dorm:"type:bigint;not null;default:0;comment:负责人"`
	Status            string    `dorm:"type:varchar(32);not null;default:'active';comment:进度状态"`
	StartedAt         time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:开始时间"`
	UpdatedAt         time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type CustomerStageIndex struct {
	CustomerAsset struct{} `unique:"customer_id,asset_id"`
	WorkflowStage struct{} `index:"workflow_id,stage_id,status,id"`
	OwnerStatus   struct{} `index:"owner_department_id,owner_staff_id,status,id"`
}

func NewCustomerStageModel() *orm.Model[CustomerStage] {
	return orm.LoadModel[CustomerStage]("资产流程进度", "crm_asset_progress", orm.ModelConfig{
		Index:    CustomerStageIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status": progressStatusOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			workflowRelation,
			stageRelation,
			ownerDepartmentRelation,
			ownerStaffRelation,
		},
	})
}
