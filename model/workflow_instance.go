package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type WorkflowInstance struct {
	ID                uint64     `dorm:"primaryKey;autoIncrement;comment:流程实例ID"`
	LeadID            uint64     `dorm:"type:bigint;not null;default:0;comment:线索"`
	CustomerID        uint64     `dorm:"type:bigint;not null;default:0;comment:客户"`
	AssetID           uint64     `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	CustomerProductID uint64     `dorm:"type:bigint;not null;default:0;comment:客户产品"`
	WorkflowID        uint64     `dorm:"type:bigint;not null;comment:流程"`
	StageID           uint64     `dorm:"type:bigint;not null;comment:当前阶段"`
	OwnerDepartmentID uint64     `dorm:"type:bigint;not null;default:0;comment:负责部门"`
	OwnerStaffID      uint64     `dorm:"type:bigint;not null;default:0;comment:负责人"`
	Status            string     `dorm:"type:varchar(32);not null;default:'active';comment:状态"`
	StartedAt         time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:阶段开始时间"`
	CompletedAt       *time.Time `dorm:"null;comment:流程完成时间"`
	TerminatedAt      *time.Time `dorm:"null;comment:流程终止时间"`
	TerminatedReason  string     `dorm:"type:text;not null;default:'';comment:终止原因"`
	UpdatedAt         time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type WorkflowInstanceIndex struct {
	LeadFlow      struct{} `index:"lead_id,workflow_id,status,id"`
	CustomerAsset struct{} `index:"customer_id,asset_id,status,id"`
	ProductFlow   struct{} `index:"customer_product_id,workflow_id,status,id"`
	WorkflowStage struct{} `index:"workflow_id,stage_id,status,id"`
	OwnerStatus   struct{} `index:"owner_department_id,owner_staff_id,status,id"`
}

func NewWorkflowInstanceModel() *orm.Model[WorkflowInstance] {
	return orm.LoadModel[WorkflowInstance]("流程实例", "crm_workflow_instance", orm.ModelConfig{
		Index:    WorkflowInstanceIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status": progressStatusOptions,
		},
		Relations: []orm.Relation{
			leadRelation,
			customerRelation,
			assetRelation,
			customerProductRelation,
			workflowRelation,
			stageRelation,
			ownerDepartmentRelation,
			ownerStaffRelation,
		},
	})
}
