package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type BusinessObject struct {
	ID                   uint64    `dorm:"primaryKey;autoIncrement;comment:业务对象ID"`
	BusinessObjectTypeID uint64    `dorm:"type:bigint;not null;comment:业务对象类型"`
	ObjectNo             string    `dorm:"type:varchar(96);not null;default:'';comment:对象编号"`
	ObjectName           string    `dorm:"type:varchar(128);not null;default:'';comment:对象名称"`
	CustomerID           uint64    `dorm:"type:bigint;not null;comment:所属客户"`
	AssetID              uint64    `dorm:"type:bigint;not null;default:0;comment:所属资产"`
	ParentObjectID       uint64    `dorm:"type:bigint;not null;default:0;comment:父级业务对象"`
	ObjectStatus         string    `dorm:"type:varchar(64);not null;default:'';comment:对象状态"`
	OwnerDepartmentID    uint64    `dorm:"type:bigint;not null;default:0;comment:负责部门"`
	OwnerStaffID         uint64    `dorm:"type:bigint;not null;default:0;comment:负责人"`
	RecordJSON           string    `dorm:"type:text;not null;default:'{}';comment:记录JSON"`
	Remark               string    `dorm:"type:text;not null;default:'';comment:备注"`
	Status               int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort                 int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type BusinessObjectIndex struct {
	ObjectNo     struct{} `index:"object_no"`
	TypeCustomer struct{} `index:"business_object_type_id,customer_id,status,id"`
	TypeAsset    struct{} `index:"business_object_type_id,asset_id,status,id"`
	ParentObject struct{} `index:"parent_object_id,status,id"`
	OwnerTime    struct{} `index:"owner_department_id,owner_staff_id,updated_at,id"`
	StatusSort   struct{} `index:"status,sort,id"`
}

func NewBusinessObjectModel() *orm.Model[BusinessObject] {
	return orm.LoadModel[BusinessObject]("租赁记录", "crm_business_object", orm.ModelConfig{
		Index:    BusinessObjectIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"object_status": businessObjectStatusOptions,
			"status":        statusOptions,
		},
		Relations: []orm.Relation{
			businessObjectTypeRelation,
			customerRelation,
			assetRelation,
			parentBusinessObjectRelation,
			ownerDepartmentRelation,
			ownerStaffRelation,
		},
	})
}
