package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type CustomerSource struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:来源ID"`
	Code      string    `dorm:"type:varchar(32);not null;comment:来源标识"`
	Name      string    `dorm:"type:varchar(64);not null;comment:来源名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type CustomerSourceIndex struct {
	Code       struct{} `unique:"code"`
	StatusSort struct{} `index:"status,sort,id"`
}

const DefaultCustomerSourceID uint64 = 1

var customerSourceSeed = []map[string]any{
	{
		"id":     DefaultCustomerSourceID,
		"code":   "ad_feed",
		"name":   "信息流投放",
		"status": StatusEnabled,
		"sort":   100,
	},
}

func NewCustomerSourceModel() *orm.Model[CustomerSource] {
	return orm.LoadModel[CustomerSource]("客户来源", "crm_customer_source", orm.ModelConfig{
		Index:    CustomerSourceIndex{},
		Seeds:    customerSourceSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
