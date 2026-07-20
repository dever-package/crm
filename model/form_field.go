package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type FormField struct {
	ID                  uint64    `dorm:"primaryKey;autoIncrement;comment:表单字段ID"`
	FormID              uint64    `dorm:"type:bigint;not null;comment:任务表单"`
	DataTemplateCateID  uint64    `dorm:"type:bigint;not null;default:0;comment:数据模板分类"`
	DataTemplateID      uint64    `dorm:"type:bigint;not null;default:0;comment:数据模板"`
	FieldSource         string    `dorm:"type:varchar(96);not null;default:'';comment:字段来源"`
	FieldPath           string    `dorm:"type:text;not null;default:'[]';comment:字段路径"`
	MainField           string    `dorm:"type:varchar(64);not null;default:'';comment:主表字段"`
	DataFieldID         uint64    `dorm:"type:bigint;not null;default:0;comment:数据模板字段"`
	Name                string    `dorm:"type:varchar(128);not null;comment:字段名称"`
	Required            bool      `dorm:"not null;default:true;comment:是否必填"`
	Readonly            bool      `dorm:"not null;default:false;comment:是否只读"`
	VisibleWhenFieldID  uint64    `dorm:"type:bigint;not null;default:0;comment:显示条件字段"`
	VisibleWhenOperator string    `dorm:"type:varchar(32);not null;default:'';comment:显示条件操作符"`
	VisibleWhenValue    string    `dorm:"type:text;not null;default:'';comment:显示条件值"`
	Sort                int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status              int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt           time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt           time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type FormFieldIndex struct {
	FormStatus struct{} `index:"form_id,status,sort,id"`
	CateStatus struct{} `index:"data_template_cate_id,status,id"`
	Template   struct{} `index:"data_template_id,data_field_id"`
	Source     struct{} `index:"field_source"`
	MainField  struct{} `index:"main_field"`
}

func NewFormFieldModel() *orm.Model[FormField] {
	return orm.LoadModel[FormField]("表单字段", "crm_form_field", orm.ModelConfig{
		Index:    FormFieldIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"visible_when_operator": formFieldVisibleOperatorOptions,
			"status":                statusOptions,
		},
		Relations: []orm.Relation{
			formRelation,
			formFieldDataTemplateCateRelation,
			formFieldDataTemplateRelation,
			formFieldDataFieldRelation,
			formFieldVisibleWhenFieldRelation,
		},
	})
}
