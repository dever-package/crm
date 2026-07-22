package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type LeadDispatchHandoff struct {
	ID                 uint64     `dorm:"primaryKey;autoIncrement;comment:线索待派单ID"`
	LeadID             uint64     `dorm:"type:bigint;not null;comment:线索"`
	WorkflowInstanceID uint64     `dorm:"type:bigint;not null;comment:来源流程实例"`
	SourceWorkflowID   uint64     `dorm:"type:bigint;not null;comment:来源流程"`
	SourceStageID      uint64     `dorm:"type:bigint;not null;comment:来源阶段"`
	SourceDepartmentID uint64     `dorm:"type:bigint;not null;comment:来源部门"`
	TargetWorkflowID   uint64     `dorm:"type:bigint;not null;comment:目标流程"`
	TargetStageID      uint64     `dorm:"type:bigint;not null;comment:目标阶段"`
	TargetDepartmentID uint64     `dorm:"type:bigint;not null;comment:接收部门"`
	AssigneeStaffID    uint64     `dorm:"type:bigint;not null;default:0;comment:接单人员"`
	DispatchType       string     `dorm:"type:varchar(32);not null;default:'';comment:派单方式"`
	OperatorStaffID    uint64     `dorm:"type:bigint;not null;default:0;comment:操作人员"`
	Status             string     `dorm:"type:varchar(32);not null;default:'pending';comment:派单状态"`
	CompletedAt        *time.Time `dorm:"null;comment:完成时间"`
	CreatedAt          time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt          time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type LeadDispatchHandoffIndex struct {
	Instance         struct{} `unique:"workflow_instance_id"`
	LeadStatus       struct{} `index:"lead_id,status,id"`
	SourceStatus     struct{} `index:"source_department_id,source_workflow_id,status,created_at,id"`
	TargetStatus     struct{} `index:"target_department_id,status,created_at,id"`
	AssigneeComplete struct{} `index:"assignee_staff_id,completed_at,id"`
}

func NewLeadDispatchHandoffModel() *orm.Model[LeadDispatchHandoff] {
	return orm.LoadModel[LeadDispatchHandoff]("线索待派单", "crm_lead_dispatch_handoff", orm.ModelConfig{
		Index:    LeadDispatchHandoffIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"dispatch_type": dispatchTypeOptions,
			"status":        leadDispatchHandoffStatusOptions,
		},
		Relations: []orm.Relation{
			leadRelation,
			workflowInstanceRelation,
			{
				Field:      "source_workflow_id",
				Option:     "crm.NewWorkflowModel",
				OptionKeys: []string{"name", "subject_type"},
			},
			{
				Field:      "source_stage_id",
				Option:     "crm.NewStageModel",
				OptionKeys: []string{"name", "workflow_id"},
			},
			{
				Field:      "source_department_id",
				Option:     "crm.NewDepartmentModel",
				OptionKeys: []string{"name", "code"},
			},
			{
				Field:      "target_workflow_id",
				Option:     "crm.NewWorkflowModel",
				OptionKeys: []string{"name", "subject_type"},
			},
			{
				Field:      "target_stage_id",
				Option:     "crm.NewStageModel",
				OptionKeys: []string{"name", "workflow_id"},
			},
			{
				Field:      "target_department_id",
				Option:     "crm.NewDepartmentModel",
				OptionKeys: []string{"name", "code"},
			},
			{
				Field:      "assignee_staff_id",
				Option:     "crm.NewStaffModel",
				OptionKeys: []string{"name", "department_id", "status"},
			},
			{
				Field:      "operator_staff_id",
				Option:     "crm.NewStaffModel",
				OptionKeys: []string{"name", "department_id", "status"},
			},
		},
	})
}
