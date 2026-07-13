package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Product struct {
	ID                uint64    `dorm:"primaryKey;autoIncrement;comment:产品ID"`
	Code              string    `dorm:"type:varchar(64);not null;comment:产品编码"`
	Name              string    `dorm:"type:varchar(128);not null;comment:产品名称"`
	CategoryID        uint64    `dorm:"type:bigint;not null;default:0;comment:产品分类"`
	ServiceWorkflowID uint64    `dorm:"type:bigint;not null;default:0;comment:服务流程"`
	Description       string    `dorm:"type:text;not null;default:'';comment:说明"`
	Status            int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort              int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt         time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt         time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type ProductIndex struct {
	Code           struct{} `unique:"code"`
	StatusSort     struct{} `index:"status,sort,id"`
	CategoryStatus struct{} `index:"category_id,status,sort,id"`
	WorkflowStatus struct{} `index:"service_workflow_id,status,id"`
}

func NewProductModel() *orm.Model[Product] {
	return orm.LoadModel[Product]("产品配置", "crm_product", orm.ModelConfig{
		Index:    ProductIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			productCategoryRelation,
			serviceWorkflowRelation,
		},
	})
}
