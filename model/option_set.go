package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type OptionSet struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:选项集ID"`
	Name      string    `dorm:"type:varchar(128);not null;comment:选项集名称"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type OptionSetIndex struct {
	StatusSort struct{} `index:"status,sort,id"`
}

var optionSetItemRelation = orm.Relation{
	Field:      "items",
	Through:    "crm.NewOptionSetItemModel",
	OwnerField: "option_set_id",
	Order:      "sort asc,id asc",
}

func NewOptionSetModel() *orm.Model[OptionSet] {
	return orm.LoadModel[OptionSet]("常用选项集", "crm_option_set", orm.ModelConfig{
		Index:    OptionSetIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{optionSetItemRelation},
	})
}
