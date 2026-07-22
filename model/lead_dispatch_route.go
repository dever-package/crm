package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type LeadDispatchRoute struct {
	ID         uint64    `dorm:"primaryKey;autoIncrement;comment:线索派单规则ID"`
	WorkflowID uint64    `dorm:"type:bigint;not null;comment:线索流程"`
	Status     int16     `dorm:"type:smallint;not null;default:2;comment:自动派单状态"`
	CreatedAt  time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt  time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type LeadDispatchRouteIndex struct {
	Workflow struct{} `unique:"workflow_id"`
	Status   struct{} `index:"status,workflow_id,id"`
}

func NewLeadDispatchRouteModel() *orm.Model[LeadDispatchRoute] {
	return orm.LoadModel[LeadDispatchRoute]("线索派单规则", "crm_lead_dispatch_route", orm.ModelConfig{
		Index:    LeadDispatchRouteIndex{},
		Order:    "id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{workflowRelation},
	})
}
