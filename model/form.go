package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Form struct {
	ID          uint64    `dorm:"primaryKey;autoIncrement;comment:资料模板ID"`
	Name        string    `dorm:"type:varchar(128);not null;comment:资料模板名称"`
	Description string    `dorm:"type:text;not null;default:'';comment:资料模板描述"`
	Sort        int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status      int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type FormIndex struct {
	Name       struct{} `unique:"name"`
	StatusSort struct{} `index:"status,sort,id"`
}

var formFieldsThroughRelation = orm.Relation{
	Field:      "fields",
	Through:    "crm.NewFormFieldModel",
	OwnerField: "form_id",
	Order:      "sort asc,id asc",
}

func NewFormModel() *orm.Model[Form] {
	return orm.LoadModel[Form]("资料模板", "crm_form", orm.ModelConfig{
		Index:    FormIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{formFieldsThroughRelation},
	})
}
