package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const (
	CustomerProductStatusCandidate  = "candidate"
	CustomerProductStatusConfirmed  = "confirmed"
	CustomerProductStatusProcessing = "processing"
	CustomerProductStatusCompleted  = "completed"
	CustomerProductStatusLost       = "lost"
)

var customerProductStatusOptions = []map[string]any{
	{"id": CustomerProductStatusCandidate, "value": "候选"},
	{"id": CustomerProductStatusConfirmed, "value": "已确认"},
	{"id": CustomerProductStatusProcessing, "value": "处理中"},
	{"id": CustomerProductStatusCompleted, "value": "已完成"},
	{"id": CustomerProductStatusLost, "value": "已流失"},
}

type CustomerProduct struct {
	ID                       uint64    `dorm:"primaryKey;autoIncrement;comment:客户产品ID"`
	CustomerID               uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID                  uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	ProductID                uint64    `dorm:"type:bigint;not null;comment:产品"`
	SourceWorkflowInstanceID uint64    `dorm:"type:bigint;not null;default:0;comment:来源流程实例"`
	Status                   string    `dorm:"type:varchar(32);not null;default:'confirmed';comment:状态"`
	CreatedAt                time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt                time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type CustomerProductIndex struct {
	SourceProduct  struct{} `unique:"source_workflow_instance_id,product_id"`
	CustomerStatus struct{} `index:"customer_id,asset_id,status,id"`
	ProductStatus  struct{} `index:"product_id,status,id"`
}

func NewCustomerProductModel() *orm.Model[CustomerProduct] {
	return orm.LoadModel[CustomerProduct]("客户产品", "crm_customer_product", orm.ModelConfig{
		Index:    CustomerProductIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status": customerProductStatusOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			productRelation,
			sourceWorkflowInstanceRelation,
		},
	})
}
