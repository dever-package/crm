package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type CustomerChannel struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:渠道ID"`
	Code      string    `dorm:"type:varchar(32);not null;comment:渠道标识"`
	Name      string    `dorm:"type:varchar(64);not null;comment:渠道名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type CustomerChannelIndex struct {
	Code       struct{} `unique:"code"`
	StatusSort struct{} `index:"status,sort,id"`
}

const DefaultCustomerChannelID uint64 = 1

var customerChannelSeed = []map[string]any{
	{
		"id":     DefaultCustomerChannelID,
		"code":   "douyin",
		"name":   "抖音",
		"status": StatusEnabled,
		"sort":   100,
	},
}

func NewCustomerChannelModel() *orm.Model[CustomerChannel] {
	return orm.LoadModel[CustomerChannel]("客户渠道", "crm_customer_channel", orm.ModelConfig{
		Index:    CustomerChannelIndex{},
		Seeds:    customerChannelSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
