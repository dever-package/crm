package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DepartmentDispatchSetting struct {
	ID           uint64    `dorm:"primaryKey;autoIncrement;comment:派单配置ID"`
	DepartmentID uint64    `dorm:"type:bigint;not null;comment:部门"`
	ActivePoolID uint64    `dorm:"type:bigint;not null;default:0;comment:当前派单池"`
	LastMemberID uint64    `dorm:"type:bigint;not null;default:0;comment:轮转游标成员"`
	Version      uint64    `dorm:"type:bigint;not null;default:1;comment:配置版本"`
	Status       int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DepartmentDispatchSettingIndex struct {
	Department struct{} `unique:"department_id"`
	PoolStatus struct{} `index:"active_pool_id,status,id"`
}

func NewDepartmentDispatchSettingModel() *orm.Model[DepartmentDispatchSetting] {
	return orm.LoadModel[DepartmentDispatchSetting]("部门派单配置", "crm_department_dispatch_setting", orm.ModelConfig{
		Index:    DepartmentDispatchSettingIndex{},
		Order:    "id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			departmentRelation,
			{
				Field:      "active_pool_id",
				Option:     "crm.NewDispatchPoolModel",
				OptionKeys: []string{"name", "pool_type", "status"},
			},
		},
	})
}
