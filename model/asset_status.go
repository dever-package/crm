package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type AssetStatus struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:资产状态ID"`
	Code      string    `dorm:"type:varchar(32);not null;comment:状态标识"`
	Name      string    `dorm:"type:varchar(64);not null;comment:状态名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:启用状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type AssetStatusIndex struct {
	Code       struct{} `unique:"code"`
	StatusSort struct{} `index:"status,sort,id"`
}

const DefaultAssetStatusID uint64 = 1

var assetStatusSeed = []map[string]any{
	{
		"id":     DefaultAssetStatusID,
		"code":   "default",
		"name":   "默认状态",
		"status": StatusEnabled,
		"sort":   100,
	},
}

func NewAssetStatusModel() *orm.Model[AssetStatus] {
	return orm.LoadModel[AssetStatus]("资产状态", "crm_asset_status", orm.ModelConfig{
		Index:    AssetStatusIndex{},
		Seeds:    assetStatusSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
