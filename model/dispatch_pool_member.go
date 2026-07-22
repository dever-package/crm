package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DispatchPoolMember struct {
	ID                 uint64    `dorm:"primaryKey;autoIncrement;comment:派单池成员ID"`
	PoolID             uint64    `dorm:"type:bigint;not null;comment:派单池"`
	DepartmentID       uint64    `dorm:"type:bigint;not null;comment:部门"`
	StaffID            uint64    `dorm:"type:bigint;not null;comment:人员"`
	WeeklyScheduleJSON string    `dorm:"type:text;not null;default:'{}';comment:每周工作时间JSON"`
	DailyLimit         int       `dorm:"type:int;not null;default:0;comment:每日自动派单上限"`
	Status             int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort               int       `dorm:"type:int;not null;default:100;comment:轮转顺序"`
	CreatedAt          time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt          time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DispatchPoolMemberIndex struct {
	PoolStaff       struct{} `unique:"pool_id,staff_id"`
	PoolStatusSort  struct{} `index:"pool_id,status,sort,id"`
	DepartmentStaff struct{} `index:"department_id,staff_id,status,id"`
}

func NewDispatchPoolMemberModel() *orm.Model[DispatchPoolMember] {
	return orm.LoadModel[DispatchPoolMember]("派单池成员", "crm_dispatch_pool_member", orm.ModelConfig{
		Index:    DispatchPoolMemberIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			{
				Field:      "pool_id",
				Option:     "crm.NewDispatchPoolModel",
				OptionKeys: []string{"name", "pool_type", "department_id", "status"},
			},
			departmentRelation,
			staffRelation,
		},
	})
}
