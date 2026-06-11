package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type CustomerLevel struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:等级ID"`
	Code      string    `dorm:"type:varchar(32);not null;comment:等级标识"`
	Name      string    `dorm:"type:varchar(64);not null;comment:等级名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type CustomerLevelIndex struct {
	Code       struct{} `unique:"code"`
	StatusSort struct{} `index:"status,sort,id"`
}

const DefaultCustomerLevelID uint64 = 1

var customerLevelSeed = []map[string]any{
	{
		"id":     DefaultCustomerLevelID,
		"code":   "default",
		"name":   "默认等级",
		"status": StatusEnabled,
		"sort":   100,
	},
}

func NewCustomerLevelModel() *orm.Model[CustomerLevel] {
	return orm.LoadModel[CustomerLevel]("客户等级", "crm_customer_level", orm.ModelConfig{
		Index:    CustomerLevelIndex{},
		Seeds:    customerLevelSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
