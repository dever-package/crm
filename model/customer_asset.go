package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type CustomerAsset struct {
	ID            uint64    `dorm:"primaryKey;autoIncrement;comment:资产ID"`
	AssetNo       string    `dorm:"type:varchar(96);not null;comment:资产编号"`
	AssetName     string    `dorm:"type:varchar(128);not null;default:'';comment:资产名称"`
	AssetSeq      uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产序号"`
	CustomerID    uint64    `dorm:"type:bigint;not null;comment:所属客户"`
	AssetStatusID uint64    `dorm:"type:bigint;not null;default:1;comment:资产状态"`
	Remark        string    `dorm:"type:text;not null;default:'';comment:备注"`
	CreatedAt     time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt     time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type CustomerAssetIndex struct {
	AssetNo     struct{} `unique:"asset_no"`
	CustomerSeq struct{} `index:"customer_id,asset_seq"`
	Customer    struct{} `index:"customer_id,id"`
	AssetStatus struct{} `index:"asset_status_id,id"`
}

func NewCustomerAssetModel() *orm.Model[CustomerAsset] {
	return orm.LoadModel[CustomerAsset]("客户资产", "crm_customer_asset", orm.ModelConfig{
		Index:    CustomerAssetIndex{},
		Order:    "id desc",
		Database: "default",
		Relations: []orm.Relation{
			customerRelation,
			assetStatusRelation,
		},
	})
}
