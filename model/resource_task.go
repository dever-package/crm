package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type ResourceTask struct {
	ID                   uint64    `dorm:"primaryKey;autoIncrement;comment:任务ID"`
	ResourceID           uint64    `dorm:"type:bigint;not null;comment:客户资产"`
	FlowTemplateID       uint64    `dorm:"type:bigint;not null;default:0;comment:流程模板"`
	FlowReleaseID        uint64    `dorm:"type:bigint;not null;default:0;comment:流程发布版本"`
	StageID              uint64    `dorm:"type:bigint;not null;default:0;comment:阶段"`
	FlowNodeID           uint64    `dorm:"type:bigint;not null;default:0;comment:流程节点"`
	FlowNodeKey          string    `dorm:"type:varchar(64);not null;default:'';comment:流程节点标识"`
	NodeType             string    `dorm:"type:varchar(32);not null;default:'task';comment:节点类型"`
	TaskTemplateID       uint64    `dorm:"type:bigint;not null;default:0;comment:任务模板"`
	TaskKey              string    `dorm:"type:varchar(64);not null;default:'';comment:任务标识"`
	TaskName             string    `dorm:"type:varchar(128);not null;comment:任务名称"`
	Status               string    `dorm:"type:varchar(32);not null;default:'pending';comment:任务状态"`
	ExecutorMode         string    `dorm:"type:varchar(32);not null;default:'department';comment:执行人模式"`
	AssigneeDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:处理部门"`
	AssigneeRoleID       uint64    `dorm:"type:bigint;not null;default:0;comment:处理角色"`
	AssigneeStaffID      uint64    `dorm:"type:bigint;not null;default:0;comment:处理人"`
	StartedAt            time.Time `dorm:"comment:开始时间"`
	DeadlineAt           time.Time `dorm:"comment:截止时间"`
	FinishedAt           time.Time `dorm:"comment:完成时间"`
	EnableDeadline       bool      `dorm:"not null;default:false;comment:是否启用截止"`
	ResultValue          string    `dorm:"type:varchar(64);not null;default:'';comment:处理结果"`
	ResultText           string    `dorm:"type:text;not null;default:'';comment:结果说明"`
	InputSnapshotJSON    string    `dorm:"type:text;not null;default:'{}';comment:输入快照JSON"`
	OutputJSON           string    `dorm:"type:text;not null;default:'{}';comment:输出JSON"`
	CreatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type ResourceTaskIndex struct {
	ResourceStatus struct{} `index:"resource_id,status,id"`
	AssigneeStatus struct{} `index:"assignee_staff_id,status,deadline_at,id"`
	DeptStatus     struct{} `index:"assignee_department_id,status,deadline_at,id"`
	StageStatus    struct{} `index:"stage_id,status,id"`
	NodeStatus     struct{} `index:"flow_node_id,status,id"`
	TemplateStatus struct{} `index:"task_template_id,status,id"`
}

func NewResourceTaskModel() *orm.Model[ResourceTask] {
	return orm.LoadModel[ResourceTask]("资源任务", "crm_resource_task", orm.ModelConfig{
		Index:    ResourceTaskIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status":        taskStatusOptions,
			"executor_mode": executorModeOptions,
		},
		Relations: []orm.Relation{
			resourceRelation,
			flowTemplateRelation,
			flowReleaseRelation,
			flowStageRelation,
			flowNodeRelation,
			taskTemplateRelation,
			assigneeDepartmentRelation,
			assigneeStaffRelation,
		},
	})
}
