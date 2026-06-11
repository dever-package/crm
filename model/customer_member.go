package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type CustomerMember struct {
	ID           uint64    `dorm:"primaryKey;autoIncrement;comment:客户成员ID"`
	CustomerID   uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID      uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	DepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:部门"`
	StaffID      uint64    `dorm:"type:bigint;not null;default:0;comment:人员"`
	RelationType string    `dorm:"type:varchar(32);not null;default:'viewer';comment:关系类型"`
	CanView      bool      `dorm:"not null;default:true;comment:是否可见"`
	Status       int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type CustomerMemberIndex struct {
	StaffCustomer struct{} `index:"staff_id,customer_id,status,id"`
	DeptCustomer  struct{} `index:"department_id,customer_id,status,id"`
	CustomerStaff struct{} `index:"customer_id,staff_id,status,id"`
	AssetStaff    struct{} `index:"asset_id,staff_id,status,id"`
}

func NewCustomerMemberModel() *orm.Model[CustomerMember] {
	return orm.LoadModel[CustomerMember]("客户成员", "crm_customer_member", orm.ModelConfig{
		Index:    CustomerMemberIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"relation_type": memberRelationOptions,
			"status":        statusOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			departmentRelation,
			staffRelation,
		},
	})
}
