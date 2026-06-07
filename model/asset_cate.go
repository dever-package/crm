package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const DefaultAssetCateID uint64 = 1

type AssetCate struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:资产分类ID"`
	Name      string    `dorm:"type:varchar(128);not null;comment:名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type AssetCateIndex struct {
	Name       struct{} `unique:"name"`
	StatusSort struct{} `index:"status,sort,id"`
}

var assetCateSeed = []map[string]any{
	{"id": DefaultAssetCateID, "name": "默认资产分类", "status": StatusEnabled, "sort": 10},
}

func NewAssetCateModel() *orm.Model[AssetCate] {
	return orm.LoadModel[AssetCate]("资产分类", "crm_asset_cate", orm.ModelConfig{
		Index:    AssetCateIndex{},
		Seeds:    assetCateSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
