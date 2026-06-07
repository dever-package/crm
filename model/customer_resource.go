package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type CustomerResource struct {
	ID          uint64    `dorm:"primaryKey;autoIncrement;comment:资产ID"`
	ResourceNo  string    `dorm:"type:varchar(96);not null;comment:资产编号"`
	AssetName   string    `dorm:"type:varchar(128);not null;default:'';comment:资产名称"`
	AssetSeq    uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产序号"`
	CustomerID  uint64    `dorm:"type:bigint;not null;comment:所属客户"`
	AssetCateID uint64    `dorm:"type:bigint;not null;default:1;comment:资产分类"`
	AssetStatus string    `dorm:"type:varchar(32);not null;default:'default';comment:资产状态"`
	Remark      string    `dorm:"type:text;not null;default:'';comment:备注"`
	CreatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type CustomerResourceIndex struct {
	ResourceNo  struct{} `unique:"resource_no"`
	CustomerSeq struct{} `index:"customer_id,asset_seq"`
	Customer    struct{} `index:"customer_id,id"`
	AssetName   struct{} `index:"asset_name,id"`
	Cate        struct{} `index:"asset_cate_id,id"`
	AssetStatus struct{} `index:"asset_status,id"`
}

func NewCustomerResourceModel() *orm.Model[CustomerResource] {
	return orm.LoadModel[CustomerResource]("客户资产", "crm_customer_resource", orm.ModelConfig{
		Index:    CustomerResourceIndex{},
		Order:    "id desc",
		Database: "default",
		Relations: []orm.Relation{
			customerRelation,
			assetCateRelation,
		},
	})
}
