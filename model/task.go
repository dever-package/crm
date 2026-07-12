package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Task struct {
	ID                   uint64    `dorm:"primaryKey;autoIncrement;comment:任务ID"`
	StageID              uint64    `dorm:"type:bigint;not null;default:0;comment:所属阶段"`
	Name                 string    `dorm:"type:varchar(128);not null;comment:任务名称"`
	TaskType             string    `dorm:"type:varchar(32);not null;default:'todo';comment:任务类型"`
	Required             bool      `dorm:"not null;default:true;comment:是否必做"`
	AssigneeMode         string    `dorm:"type:varchar(32);not null;default:'stage';comment:负责方式"`
	AssigneeDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:负责部门"`
	AssigneeStaffID      uint64    `dorm:"type:bigint;not null;default:0;comment:负责人"`
	FormID               uint64    `dorm:"type:bigint;not null;default:0;comment:资料表单"`
	ScriptID             uint64    `dorm:"type:bigint;not null;default:0;comment:核验规则"`
	DueDays              int       `dorm:"type:int;not null;default:0;comment:办理期限天数"`
	Sort                 int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status               int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type TaskIndex struct {
	StageStatus    struct{} `index:"stage_id,status,sort,id"`
	TypeStatus     struct{} `index:"task_type,status,sort,id"`
	AssigneeStatus struct{} `index:"assignee_department_id,assignee_staff_id,status,id"`
}

func NewTaskModel() *orm.Model[Task] {
	return orm.LoadModel[Task]("任务配置", "crm_task", orm.ModelConfig{
		Index:    TaskIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"task_type":     taskTypeOptions,
			"assignee_mode": taskAssigneeModeOptions,
			"status":        statusOptions,
		},
		Relations: []orm.Relation{
			stageRelation,
			assigneeDepartmentRelation,
			assigneeStaffRelation,
			formRelation,
			ruleScriptRelation,
		},
	})
}
