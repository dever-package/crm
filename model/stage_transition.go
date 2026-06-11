package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type StageTransition struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:阶段流转ID"`
	FromStageCode  string    `dorm:"type:varchar(32);not null;comment:来源阶段"`
	TaskID         uint64    `dorm:"type:bigint;not null;default:0;comment:任务"`
	ResultValue    string    `dorm:"type:varchar(64);not null;default:'';comment:匹配结果"`
	ScriptID       uint64    `dorm:"type:bigint;not null;default:0;comment:匹配脚本"`
	ToStageCode    string    `dorm:"type:varchar(32);not null;comment:目标阶段"`
	OwnerMode      string    `dorm:"type:varchar(32);not null;default:'keep';comment:主责模式"`
	ToDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:目标部门"`
	ToStaffID      uint64    `dorm:"type:bigint;not null;default:0;comment:目标人员"`
	ConditionJSON  string    `dorm:"type:text;not null;default:'{}';comment:条件JSON"`
	Sort           int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status         int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type StageTransitionIndex struct {
	FromResult struct{} `index:"from_stage_code,task_id,result_value,status,id"`
	ToStatus   struct{} `index:"to_stage_code,status,sort,id"`
	StatusSort struct{} `index:"status,sort,id"`
}

func NewStageTransitionModel() *orm.Model[StageTransition] {
	return orm.LoadModel[StageTransition]("阶段流转", "crm_stage_transition", orm.ModelConfig{
		Index:    StageTransitionIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"owner_mode": stageOwnerModeOptions,
			"status":     statusOptions,
		},
		Relations: []orm.Relation{
			fromStageRelation,
			taskRelation,
			ruleScriptRelation,
			toStageRelation,
			toDepartmentRelation,
			toStaffRelation,
		},
	})
}
