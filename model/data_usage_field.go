package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DataUsageField struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:用途字段ID"`
	UsageID        uint64    `dorm:"type:bigint;not null;comment:系统用途"`
	DataTemplateID uint64    `dorm:"type:bigint;not null;comment:数据模板"`
	DataFieldID    uint64    `dorm:"type:bigint;not null;comment:数据字段"`
	ValueType      string    `dorm:"type:varchar(32);not null;default:'text';comment:值类型"`
	AggregateType  string    `dorm:"type:varchar(32);not null;default:'';comment:聚合方式"`
	FinanceTypeID  uint64    `dorm:"type:bigint;not null;default:0;comment:财务类型"`
	DisplayName    string    `dorm:"type:varchar(128);not null;default:'';comment:显示名称"`
	ConfigJSON     string    `dorm:"type:text;not null;default:'{}';comment:配置JSON"`
	Sort           int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status         int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DataUsageFieldIndex struct {
	UsageField struct{} `unique:"usage_id,data_field_id"`
	UsageSort  struct{} `index:"usage_id,status,sort,id"`
	FieldUsage struct{} `index:"data_field_id,usage_id,status,id"`
	Finance    struct{} `index:"finance_type_id,status,id"`
}

var dataUsageRelation = orm.Relation{
	Field:      "usage_id",
	Option:     "crm.NewDataUsageModel",
	OptionKeys: []string{"name", "usage_type"},
}

func NewDataUsageFieldModel() *orm.Model[DataUsageField] {
	return orm.LoadModel[DataUsageField]("系统用途字段", "crm_data_usage_field", orm.ModelConfig{
		Index:    DataUsageFieldIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"value_type":     dataUsageValueTypeOptions,
			"aggregate_type": dataUsageAggregateTypeOptions,
			"status":         statusOptions,
		},
		Relations: []orm.Relation{
			dataUsageRelation,
			dataTemplateRelation,
			dataFieldRelation,
			financeTypeRelation,
		},
	})
}
