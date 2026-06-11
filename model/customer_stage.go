package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type CustomerStage struct {
	ID                  uint64    `dorm:"primaryKey;autoIncrement;comment:客户阶段ID"`
	CustomerID          uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID             uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	CurrentStageCode    string    `dorm:"type:varchar(32);not null;default:'';comment:当前阶段"`
	CurrentDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:当前主责部门"`
	CurrentStaffID      uint64    `dorm:"type:bigint;not null;default:0;comment:当前主责人"`
	LastOperationLogID  uint64    `dorm:"type:bigint;not null;default:0;comment:最后操作记录"`
	LastTransitionLogID uint64    `dorm:"type:bigint;not null;default:0;comment:最后流转记录"`
	LastOperatedAt      time.Time `dorm:"comment:最后操作时间"`
	ContextJSON         string    `dorm:"type:text;not null;default:'{}';comment:上下文JSON"`
	CreatedAt           time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt           time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type CustomerStageIndex struct {
	CustomerAsset struct{} `unique:"customer_id,asset_id"`
	StageDept     struct{} `index:"current_stage_code,current_department_id,current_staff_id,id"`
	StaffStage    struct{} `index:"current_staff_id,current_stage_code,id"`
}

func NewCustomerStageModel() *orm.Model[CustomerStage] {
	return orm.LoadModel[CustomerStage]("客户阶段", "crm_customer_stage", orm.ModelConfig{
		Index:    CustomerStageIndex{},
		Order:    "id desc",
		Database: "default",
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			currentStageRelation,
			currentDepartmentRelation,
			currentStaffRelation,
		},
	})
}
