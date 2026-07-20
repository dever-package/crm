package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Task struct {
	ID                     uint64 `dorm:"primaryKey;autoIncrement;comment:任务ID"`
	StageID                uint64 `dorm:"type:bigint;not null;default:0;comment:所属阶段"`
	Name                   string `dorm:"type:varchar(128);not null;comment:任务名称"`
	TaskType               string `dorm:"type:varchar(32);not null;default:'todo';comment:任务类型"`
	Required               bool   `dorm:"not null;default:true;comment:是否必做"`
	AssigneeMode           string `dorm:"type:varchar(32);not null;default:'stage';comment:负责方式"`
	AssigneeDepartmentID   uint64 `dorm:"type:bigint;not null;default:0;comment:负责部门"`
	FormID                 uint64 `dorm:"type:bigint;not null;default:0;comment:资料表单"`
	ScriptID               uint64 `dorm:"type:bigint;not null;default:0;comment:核验规则"`
	ActivationMode         string `dorm:"type:varchar(32);not null;default:'stage';comment:激活方式"`
	ConditionScriptID      uint64 `dorm:"type:bigint;not null;default:0;comment:适用条件规则"`
	RejectTargetTaskID     uint64 `dorm:"type:bigint;not null;default:0;comment:驳回目标任务"`
	CompleteTargetTaskID   uint64 `dorm:"type:bigint;not null;default:0;comment:完成目标任务"`
	MeetingEnabled         bool   `dorm:"not null;default:false;comment:是否需要预约会议"`
	MeetingArrivalRequired bool   `dorm:"not null;default:false;comment:是否需要客户到访确认"`
	CustomerFollowEnabled  bool   `dorm:"not null;default:false;comment:是否填写下次跟进时间"`
	// 旧字段映射仅用于历史迁移兼容，运行时不再读取。
	MeetingStartFieldID    uint64    `dorm:"type:bigint;not null;default:0;comment:会议开始字段"`
	MeetingDurationFieldID uint64    `dorm:"type:bigint;not null;default:0;comment:会议时长字段"`
	MeetingResourceFieldID uint64    `dorm:"type:bigint;not null;default:0;comment:会议室字段"`
	IncludeInMeeting       bool      `dorm:"not null;default:false;comment:负责人加入案件会议"`
	DueDays                int       `dorm:"type:int;not null;default:0;comment:办理期限天数"`
	Sort                   int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status                 int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt              time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt              time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type TaskIndex struct {
	StageStatus    struct{} `index:"stage_id,status,sort,id"`
	TypeStatus     struct{} `index:"task_type,status,sort,id"`
	AssigneeStatus struct{} `index:"assignee_department_id,status,id"`
}

func NewTaskModel() *orm.Model[Task] {
	return orm.LoadModel[Task]("任务配置", "crm_task", orm.ModelConfig{
		Index:    TaskIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"task_type":       taskTypeOptions,
			"assignee_mode":   taskAssigneeModeOptions,
			"activation_mode": taskActivationModeOptions,
			"status":          statusOptions,
		},
		Relations: []orm.Relation{
			stageRelation,
			assigneeDepartmentRelation,
			formRelation,
			ruleScriptRelation,
		},
	})
}
