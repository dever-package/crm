package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type CustomerTagRelation struct {
	ID         uint64    `dorm:"primaryKey;autoIncrement;comment:关联ID"`
	CustomerID uint64    `dorm:"type:bigint;not null;comment:客户"`
	TagID      uint64    `dorm:"type:bigint;not null;comment:客户标签"`
	CreatedAt  time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type CustomerTagRelationIndex struct {
	CustomerTag struct{} `unique:"customer_id,tag_id"`
	TagCustomer struct{} `index:"tag_id,customer_id,id"`
}

func NewCustomerTagRelationModel() *orm.Model[CustomerTagRelation] {
	return orm.LoadModel[CustomerTagRelation]("客户标签关系", "crm_customer_tag_relation", orm.ModelConfig{
		Index:    CustomerTagRelationIndex{},
		Order:    "id asc",
		Database: "default",
		Relations: []orm.Relation{
			customerRelation,
			customerTagRelation,
		},
	})
}
