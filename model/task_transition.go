package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type TaskTransition struct {
	ID                   uint64    `dorm:"primaryKey;autoIncrement;comment:任务流转ID"`
	FlowTemplateID       uint64    `dorm:"type:bigint;not null;comment:流程模板"`
	FromTaskTemplateID   uint64    `dorm:"type:bigint;not null;comment:来源任务"`
	MatchResult          string    `dorm:"type:varchar(64);not null;default:'';comment:匹配结果"`
	MatchScriptID        uint64    `dorm:"type:bigint;not null;default:0;comment:匹配脚本"`
	ToStageID            uint64    `dorm:"type:bigint;not null;default:0;comment:目标阶段"`
	ToTaskTemplateID     uint64    `dorm:"type:bigint;not null;default:0;comment:目标任务"`
	TargetResourceStatus string    `dorm:"type:varchar(32);not null;default:'';comment:目标资源状态"`
	TargetDepartmentID   uint64    `dorm:"type:bigint;not null;default:0;comment:目标部门"`
	TargetRoleID         uint64    `dorm:"type:bigint;not null;default:0;comment:目标角色"`
	ConditionJSON        string    `dorm:"type:text;not null;default:'{}';comment:条件JSON"`
	Sort                 int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status               int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type TaskTransitionIndex struct {
	FlowStatus struct{} `index:"flow_template_id,status,sort,id"`
	FromResult struct{} `index:"from_task_template_id,match_result,status,id"`
	ToTask     struct{} `index:"to_task_template_id,status,id"`
}

func NewTaskTransitionModel() *orm.Model[TaskTransition] {
	return orm.LoadModel[TaskTransition]("任务流转", "crm_task_transition", orm.ModelConfig{
		Index:    TaskTransitionIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"target_resource_status": resourceStatusOptions,
			"status":                 statusOptions,
		},
		Relations: []orm.Relation{
			flowTemplateRelation,
			fromTaskTemplateRelation,
			matchScriptRelation,
			toStageRelation,
			toTaskTemplateRelation,
			targetDepartmentRelation,
		},
	})
}
