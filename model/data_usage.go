package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DataUsage struct {
	ID          uint64    `dorm:"primaryKey;autoIncrement;comment:系统用途ID"`
	Name        string    `dorm:"type:varchar(128);not null;comment:用途名称"`
	UsageType   string    `dorm:"type:varchar(32);not null;default:'stat';comment:用途类型"`
	Description string    `dorm:"type:text;not null;default:'';comment:说明"`
	Sort        int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status      int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DataUsageIndex struct {
	TypeStatus struct{} `index:"usage_type,status,sort,id"`
	StatusSort struct{} `index:"status,sort,id"`
}

var dataUsageFieldRelation = orm.Relation{
	Field:      "fields",
	Through:    "crm.NewDataUsageFieldModel",
	OwnerField: "usage_id",
	Order:      "sort asc,id asc",
}

func NewDataUsageModel() *orm.Model[DataUsage] {
	return orm.LoadModel[DataUsage]("系统用途", "crm_data_usage", orm.ModelConfig{
		Index:    DataUsageIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"usage_type": dataUsageTypeOptions,
			"status":     statusOptions,
		},
		Relations: []orm.Relation{dataUsageFieldRelation},
	})
}
