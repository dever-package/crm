package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Form struct {
	ID                  uint64    `dorm:"primaryKey;autoIncrement;comment:任务表单ID"`
	Name                string    `dorm:"type:varchar(128);not null;comment:任务表单名称"`
	Description         string    `dorm:"type:text;not null;default:'';comment:任务表单描述"`
	CalculationScriptID uint64    `dorm:"type:bigint;not null;default:0;comment:计算规则"`
	Sort                int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status              int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt           time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt           time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type FormIndex struct {
	StatusSort struct{} `index:"status,sort,id"`
}

var formFieldsThroughRelation = orm.Relation{
	Field:      "fields",
	Through:    "crm.NewFormFieldModel",
	OwnerField: "form_id",
	Order:      "sort asc,id asc",
}

var formCalculationScriptRelation = orm.Relation{
	Field:      "calculation_script_id",
	Option:     "crm.NewRuleScriptModel",
	OptionKeys: []string{"name"},
}

func NewFormModel() *orm.Model[Form] {
	return orm.LoadModel[Form]("任务表单", "crm_form", orm.ModelConfig{
		Index:    FormIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			formCalculationScriptRelation,
			formFieldsThroughRelation,
		},
	})
}
