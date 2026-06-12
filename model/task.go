package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Task struct {
	ID               uint64    `dorm:"primaryKey;autoIncrement;comment:任务ID"`
	StageID          uint64    `dorm:"type:bigint;not null;default:0;comment:所属阶段"`
	Name             string    `dorm:"type:varchar(128);not null;comment:任务名称"`
	TaskType         string    `dorm:"type:varchar(32);not null;default:'form';comment:任务动作"`
	FormID           uint64    `dorm:"type:bigint;not null;default:0;comment:资料模板"`
	TriggerType      string    `dorm:"type:varchar(32);not null;default:'manual';comment:触发方式"`
	TriggerTaskID    uint64    `dorm:"type:bigint;not null;default:0;comment:触发任务"`
	ScriptID         uint64    `dorm:"type:bigint;not null;default:0;comment:脚本规则"`
	ResultSchemaJSON string    `dorm:"type:text;not null;default:'[]';comment:结果配置JSON"`
	ConfigJSON       string    `dorm:"type:text;not null;default:'{}';comment:任务配置JSON"`
	Sort             int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status           int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt        time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt        time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type TaskIndex struct {
	StageName     struct{} `unique:"stage_id,name"`
	StageStatus   struct{} `index:"stage_id,status,sort,id"`
	TypeStatus    struct{} `index:"task_type,status,sort,id"`
	TriggerStatus struct{} `index:"trigger_type,trigger_task_id,status,id"`
}

func NewTaskModel() *orm.Model[Task] {
	return orm.LoadModel[Task]("任务配置", "crm_task", orm.ModelConfig{
		Index:    TaskIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"task_type":    taskTypeOptions,
			"trigger_type": taskTriggerOptions,
			"status":       statusOptions,
		},
		Relations: []orm.Relation{
			stageRelation,
			formRelation,
			triggerTaskRelation,
			ruleScriptRelation,
		},
	})
}
