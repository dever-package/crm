package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type PublicResourceCate struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:分类ID"`
	Name      string    `dorm:"type:varchar(64);not null;comment:分类名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type PublicResourceCateIndex struct {
	StatusSort struct{} `index:"status,sort,id"`
}

const DefaultResourceCateID uint64 = 1

var publicResourceCateSeed = []map[string]any{
	{
		"id":     DefaultResourceCateID,
		"name":   "会议室",
		"status": StatusEnabled,
		"sort":   100,
	},
}

func NewPublicResourceCateModel() *orm.Model[PublicResourceCate] {
	return orm.LoadModel[PublicResourceCate]("资源分类", "crm_public_resource_cate", orm.ModelConfig{
		Index:    PublicResourceCateIndex{},
		Seeds:    publicResourceCateSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
