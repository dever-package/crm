package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const CustomerSourceDefault = "default"

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

var customerSourceSeed = []map[string]any{
	{"id": 1, "code": CustomerSourceDefault, "name": "默认来源", "status": StatusEnabled, "sort": 10},
	{"id": 2, "code": "douyin", "name": "抖音", "status": StatusEnabled, "sort": 20},
	{"id": 3, "code": "kuaishou", "name": "快手", "status": StatusEnabled, "sort": 30},
	{"id": 4, "code": "xiaohongshu", "name": "小红书", "status": StatusEnabled, "sort": 40},
	{"id": 5, "code": "organic", "name": "自然流", "status": StatusEnabled, "sort": 50},
	{"id": 6, "code": "self_develop", "name": "自拓", "status": StatusEnabled, "sort": 60},
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
