package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type BusinessObjectType struct {
	ID           uint64    `dorm:"primaryKey;autoIncrement;comment:业务对象类型ID"`
	Code         string    `dorm:"type:varchar(64);not null;comment:类型编码"`
	Name         string    `dorm:"type:varchar(128);not null;comment:类型名称"`
	ParentTarget string    `dorm:"type:varchar(32);not null;default:'customer_asset';comment:归属对象"`
	Description  string    `dorm:"type:text;not null;default:'';comment:说明"`
	Status       int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort         int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type BusinessObjectTypeIndex struct {
	Code       struct{} `unique:"code"`
	StatusSort struct{} `index:"status,sort,id"`
	Parent     struct{} `index:"parent_target,status,id"`
}

func NewBusinessObjectTypeModel() *orm.Model[BusinessObjectType] {
	return orm.LoadModel[BusinessObjectType]("业务记录类型", "crm_business_object_type", orm.ModelConfig{
		Index:    BusinessObjectTypeIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"parent_target": businessObjectParentTargetOptions,
			"status":        statusOptions,
		},
	})
}
