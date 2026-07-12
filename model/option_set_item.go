package model

import "github.com/shemic/dever/orm"

type OptionSetItem struct {
	ID          uint64 `dorm:"primaryKey;autoIncrement;comment:选项ID"`
	OptionSetID uint64 `dorm:"type:bigint;not null;comment:选项集"`
	Name        string `dorm:"type:varchar(128);not null;comment:选项名"`
	Value       string `dorm:"type:varchar(255);not null;comment:选项值"`
	Sort        int    `dorm:"type:int;not null;default:100;comment:排序"`
	Status      int16  `dorm:"type:smallint;not null;default:1;comment:状态"`
}

type OptionSetItemIndex struct {
	SetValue struct{} `unique:"option_set_id,value"`
	SetSort  struct{} `index:"option_set_id,status,sort,id"`
}

var optionSetItemSetRelation = orm.Relation{
	Field:      "option_set_id",
	Option:     "crm.NewOptionSetModel",
	OptionKeys: []string{"name"},
}

func NewOptionSetItemModel() *orm.Model[OptionSetItem] {
	return orm.LoadModel[OptionSetItem]("常用选项", "crm_option_set_item", orm.ModelConfig{
		Index:    OptionSetItemIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{optionSetItemSetRelation},
	})
}
