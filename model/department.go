package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Department struct {
	ID            uint64    `dorm:"primaryKey;autoIncrement;comment:部门ID"`
	Code          string    `dorm:"type:varchar(32);not null;comment:部门标识"`
	Name          string    `dorm:"type:varchar(64);not null;comment:部门名称"`
	LeaderStaffID uint64    `dorm:"type:bigint;not null;default:0;comment:部门负责人"`
	Status        int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort          int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt     time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type DepartmentIndex struct {
	Code        struct{} `unique:"code"`
	Name        struct{} `unique:"name"`
	LeaderStaff struct{} `index:"leader_staff_id,status,id"`
	StatusSort  struct{} `index:"status,sort,id"`
}

var departmentSeed = []map[string]any{
	{"id": 1, "code": "default", "name": "默认部门", "leader_staff_id": 0, "status": StatusEnabled, "sort": 10},
}

func NewDepartmentModel() *orm.Model[Department] {
	return orm.LoadModel[Department]("CRM部门", "crm_department", orm.ModelConfig{
		Index:    DepartmentIndex{},
		Seeds:    departmentSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{leaderStaffRelation},
	})
}
