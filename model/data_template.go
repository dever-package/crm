package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DataTemplate struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:数据模板ID"`
	CateID    uint64    `dorm:"type:bigint;not null;default:1;comment:数据模板分类"`
	Name      string    `dorm:"type:varchar(128);not null;comment:模板名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DataTemplateIndex struct {
	CateStatus struct{} `index:"cate_id,status,sort,id"`
}

var dataTemplateFieldRelation = orm.Relation{
	Field:      "fields",
	Through:    "crm.NewDataFieldModel",
	OwnerField: "data_template_id",
	Order:      "sort asc,id asc",
}

func NewDataTemplateModel() *orm.Model[DataTemplate] {
	return orm.LoadModel[DataTemplate]("业务数据模板", "crm_data_template", orm.ModelConfig{
		Index:    DataTemplateIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			dataTemplateCateRelation,
			dataTemplateFieldRelation,
		},
	})
}
