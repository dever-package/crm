package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type ProductCategory struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:产品分类ID"`
	Name      string    `dorm:"type:varchar(128);not null;comment:分类名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type ProductCategoryIndex struct {
	StatusSort struct{} `index:"status,sort,id"`
}

func NewProductCategoryModel() *orm.Model[ProductCategory] {
	return orm.LoadModel[ProductCategory]("产品分类", "crm_product_category", orm.ModelConfig{
		Index:    ProductCategoryIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
