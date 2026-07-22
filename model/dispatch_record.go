package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DispatchRecord struct {
	ID                 uint64    `dorm:"primaryKey;autoIncrement;comment:派单记录ID"`
	DispatchType       string    `dorm:"type:varchar(32);not null;comment:派单方式"`
	Source             string    `dorm:"type:varchar(32);not null;default:'';comment:派单来源"`
	DepartmentID       uint64    `dorm:"type:bigint;not null;comment:目标部门"`
	StaffID            uint64    `dorm:"type:bigint;not null;comment:目标人员"`
	PreviousStaffID    uint64    `dorm:"type:bigint;not null;default:0;comment:原负责人"`
	WorkflowInstanceID uint64    `dorm:"type:bigint;not null;default:0;comment:流程实例"`
	WorkTodoID         uint64    `dorm:"type:bigint;not null;default:0;comment:任务待办"`
	LeadID             uint64    `dorm:"type:bigint;not null;default:0;comment:线索"`
	OperatorStaffID    uint64    `dorm:"type:bigint;not null;default:0;comment:操作人"`
	CreatedAt          time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type DispatchRecordIndex struct {
	DepartmentTime struct{} `index:"department_id,created_at,id"`
	StaffAutoTime  struct{} `index:"staff_id,dispatch_type,created_at,id"`
	InstanceTime   struct{} `index:"workflow_instance_id,created_at,id"`
	TodoTime       struct{} `index:"work_todo_id,created_at,id"`
	LeadTime       struct{} `index:"lead_id,created_at,id"`
}

func NewDispatchRecordModel() *orm.Model[DispatchRecord] {
	return orm.LoadModel[DispatchRecord]("派单记录", "crm_dispatch_record", orm.ModelConfig{
		Index:    DispatchRecordIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"dispatch_type": dispatchTypeOptions,
		},
		Relations: []orm.Relation{
			departmentRelation,
			staffRelation,
			workflowInstanceRelation,
			leadRelation,
		},
	})
}
