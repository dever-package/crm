package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const CustomerDataTemplateCateID uint64 = 1
const DefaultDataTemplateCateID = CustomerDataTemplateCateID
const CustomerAssetDataTemplateCateID uint64 = 2

const (
	DataTemplateTargetCustomer      = "customer"
	DataTemplateTargetCustomerAsset = "customer_asset"
)

type DataTemplateCate struct {
	ID          uint64    `dorm:"primaryKey;autoIncrement;comment:数据模板分类ID"`
	Name        string    `dorm:"type:varchar(128);not null;comment:名称"`
	TargetTable string    `dorm:"type:varchar(32);not null;default:'customer';comment:主表"`
	Status      int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort        int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type DataTemplateCateIndex struct {
	Target     struct{} `index:"target_table,id"`
	StatusSort struct{} `index:"status,sort,id"`
}

var dataTemplateCateSeed = []map[string]any{
	{"id": CustomerDataTemplateCateID, "name": "客户信息", "target_table": DataTemplateTargetCustomer, "status": StatusEnabled, "sort": 10},
	{"id": CustomerAssetDataTemplateCateID, "name": "客户资产", "target_table": DataTemplateTargetCustomerAsset, "status": StatusEnabled, "sort": 20},
}

var dataTemplateTargetOptions = []map[string]any{
	{"id": DataTemplateTargetCustomer, "value": "客户信息"},
	{"id": DataTemplateTargetCustomerAsset, "value": "客户资产"},
}

func NewDataTemplateCateModel() *orm.Model[DataTemplateCate] {
	return orm.LoadModel[DataTemplateCate]("数据模板分类", "crm_data_template_cate", orm.ModelConfig{
		Index:    DataTemplateCateIndex{},
		Seeds:    dataTemplateCateSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"target_table": dataTemplateTargetOptions,
			"status":       statusOptions,
		},
	})
}
