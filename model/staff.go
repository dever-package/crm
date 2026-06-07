package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Staff struct {
	ID           uint64    `dorm:"primaryKey;autoIncrement;comment:人员ID"`
	Name         string    `dorm:"type:varchar(64);not null;comment:姓名"`
	AccountID    uint64    `dorm:"type:bigint;not null;comment:系统账号"`
	DepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:所属部门"`
	Phone        string    `dorm:"type:varchar(32);not null;default:'';comment:联系电话"`
	Status       int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type StaffIndex struct {
	Account          struct{} `unique:"account_id"`
	NameStatus       struct{} `index:"name,status,id"`
	DepartmentStatus struct{} `index:"department_id,status,id"`
}

func NewStaffModel() *orm.Model[Staff] {
	return orm.LoadModel[Staff]("CRM人员", "crm_staff", orm.ModelConfig{
		Index:    StaffIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			accountRelation,
			departmentRelation,
		},
	})
}
