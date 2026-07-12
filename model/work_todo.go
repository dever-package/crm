package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type WorkTodo struct {
	ID                   uint64     `dorm:"primaryKey;autoIncrement;comment:任务待办ID"`
	CustomerID           uint64     `dorm:"type:bigint;not null;comment:客户"`
	AssetID              uint64     `dorm:"type:bigint;not null;comment:客户资产"`
	WorkflowID           uint64     `dorm:"type:bigint;not null;comment:流程"`
	StageID              uint64     `dorm:"type:bigint;not null;comment:阶段"`
	TaskID               uint64     `dorm:"type:bigint;not null;comment:任务"`
	AssigneeDepartmentID uint64     `dorm:"type:bigint;not null;default:0;comment:负责部门"`
	AssigneeStaffID      uint64     `dorm:"type:bigint;not null;default:0;comment:负责人"`
	Required             bool       `dorm:"not null;default:true;comment:是否必做"`
	Status               string     `dorm:"type:varchar(32);not null;default:'pending';comment:状态"`
	DueAt                *time.Time `dorm:"null;comment:截止时间"`
	Result               string     `dorm:"type:text;not null;default:'';comment:处理结果"`
	CompletedAt          *time.Time `dorm:"null;comment:完成时间"`
	CreatedAt            time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt            time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type WorkTodoIndex struct {
	AssigneeStatus struct{} `index:"assignee_department_id,assignee_staff_id,status,due_at,id"`
	CustomerStatus struct{} `index:"customer_id,asset_id,status,id"`
	StageTask      struct{} `index:"asset_id,stage_id,task_id,status,id"`
	WorkflowStatus struct{} `index:"workflow_id,stage_id,status,id"`
}

func NewWorkTodoModel() *orm.Model[WorkTodo] {
	return orm.LoadModel[WorkTodo]("任务待办", "crm_task_todo", orm.ModelConfig{
		Index:    WorkTodoIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status": workTodoStatusOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			workflowRelation,
			stageRelation,
			taskRelation,
			assigneeDepartmentRelation,
			assigneeStaffRelation,
		},
	})
}
