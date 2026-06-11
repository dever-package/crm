package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Stage struct {
	ID                uint64    `dorm:"primaryKey;autoIncrement;comment:阶段ID"`
	Code              string    `dorm:"type:varchar(32);not null;comment:状态码"`
	Name              string    `dorm:"type:varchar(128);not null;comment:阶段名称"`
	OwnerDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:主责部门"`
	EntryCondition    string    `dorm:"type:text;not null;default:'';comment:进入条件"`
	ExitCondition     string    `dorm:"type:text;not null;default:'';comment:退出条件"`
	Sort              int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status            int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt         time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt         time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type StageIndex struct {
	Code       struct{} `unique:"code"`
	Owner      struct{} `index:"owner_department_id,status,sort,id"`
	StatusSort struct{} `index:"status,sort,id"`
}

const (
	DefaultStageID   uint64 = 1
	DefaultStageCode        = "default"
	DefaultStageName        = "默认阶段"
)

var stageSeed = []map[string]any{
	DefaultStageRecord(),
}

func DefaultStageRecord() map[string]any {
	return map[string]any{
		"id":                  DefaultStageID,
		"code":                DefaultStageCode,
		"name":                DefaultStageName,
		"owner_department_id": DefaultDepartmentID,
		"entry_condition":     "",
		"exit_condition":      "",
		"status":              StatusEnabled,
		"sort":                100,
	}
}

func NewStageModel() *orm.Model[Stage] {
	return orm.LoadModel[Stage]("阶段配置", "crm_stage", orm.ModelConfig{
		Index:    StageIndex{},
		Seeds:    stageSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{ownerDepartmentRelation},
	})
}
