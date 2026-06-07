package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type StaffRole struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:人员角色ID"`
	StaffID   uint64    `dorm:"type:bigint;not null;comment:人员"`
	RoleType  string    `dorm:"type:varchar(32);not null;comment:业务角色"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type StaffRoleIndex struct {
	StaffRole  struct{} `unique:"staff_id,role_type"`
	RoleStatus struct{} `index:"role_type,status,id"`
}

func NewStaffRoleModel() *orm.Model[StaffRole] {
	return orm.LoadModel[StaffRole]("CRM人员角色", "crm_staff_role", orm.ModelConfig{
		Index:    StaffRoleIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"role_type": businessRoleOptions,
			"status":    statusOptions,
		},
		Relations: []orm.Relation{staffRelation},
	})
}
