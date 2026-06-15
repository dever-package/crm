package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Staff struct {
	ID           uint64    `dorm:"primaryKey;autoIncrement;comment:人员ID"`
	Name         string    `dorm:"type:varchar(64);not null;comment:姓名"`
	DepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:部门"`
	StaffType    string    `dorm:"type:varchar(32);not null;default:'employee';comment:人员类型"`
	Phone        string    `dorm:"type:varchar(32);not null;default:'';comment:手机号"`
	FeishuOpenID string    `dorm:"type:varchar(128);not null;default:'';comment:飞书OpenID"`
	Password     string    `dorm:"type:varchar(128);not null;default:'';comment:密码"`
	Status       int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type StaffIndex struct {
	Phone            struct{} `index:"phone,id"`
	FeishuOpenID     struct{} `index:"feishu_open_id,id"`
	NameStatus       struct{} `index:"name,status,id"`
	TypeStatus       struct{} `index:"staff_type,status,id"`
	DepartmentStatus struct{} `index:"department_id,status,id"`
}

const DefaultStaffID uint64 = 1

var staffSeed = []map[string]any{
	{
		"id":            DefaultStaffID,
		"name":          "默认人员",
		"department_id": DefaultDepartmentID,
		"staff_type":    StaffTypeLeader,
		"phone":         "13800000000",
		"password":      "123456",
		"status":        StatusEnabled,
	},
}

func NewStaffModel() *orm.Model[Staff] {
	return orm.LoadModel[Staff]("CRM人员", "crm_staff", orm.ModelConfig{
		Index:    StaffIndex{},
		Seeds:    staffSeed,
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"staff_type": staffTypeOptions,
			"status":     statusOptions,
		},
		Fields: map[string]orm.FieldConfig{
			"password": {Type: orm.FieldTypePassword},
		},
		Relations: []orm.Relation{departmentRelation},
	})
}
