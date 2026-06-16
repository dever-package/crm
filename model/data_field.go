package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DataField struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:数据字段ID"`
	DataTemplateID uint64    `dorm:"type:bigint;not null;comment:数据模板"`
	Name           string    `dorm:"type:varchar(128);not null;comment:字段名称"`
	FieldKey       string    `dorm:"type:varchar(128);not null;default:'';comment:字段编码"`
	FieldType      string    `dorm:"type:varchar(32);not null;default:'text';comment:字段类型"`
	DefaultValue   string    `dorm:"type:text;not null;default:'';comment:默认值"`
	StatEnabled    bool      `dorm:"not null;default:false;comment:条件字段"`
	StatType       string    `dorm:"type:varchar(32);not null;default:'dimension';comment:条件值类型"`
	StatID         uint64    `dorm:"type:bigint;not null;default:0;comment:条件类型关联ID"`
	StatGroup      string    `dorm:"type:varchar(64);not null;default:'';comment:条件分组"`
	Sort           int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status         int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DataFieldIndex struct {
	TemplateName   struct{} `unique:"data_template_id,name"`
	TemplateKey    struct{} `index:"data_template_id,field_key"`
	StatKey        struct{} `index:"stat_enabled,field_key,status,id"`
	StatGroup      struct{} `index:"stat_group,stat_type,status,id"`
	StatRef        struct{} `index:"stat_type,stat_id,status,id"`
	TemplateStatus struct{} `index:"data_template_id,status,sort,id"`
	TypeStatus     struct{} `index:"field_type,status,id"`
}

var dataFieldOptionRelation = orm.Relation{
	Field:      "options",
	Through:    "crm.NewDataFieldOptionModel",
	OwnerField: "data_field_id",
	Order:      "sort asc,id asc",
}

var dataFieldStatRelation = orm.Relation{
	Field:      "stat_id",
	Option:     "crm.NewFinanceTypeModel",
	OptionKeys: []string{"name", "code", "direction"},
}

func NewDataFieldModel() *orm.Model[DataField] {
	return orm.LoadModel[DataField]("业务数据字段", "crm_data_field", orm.ModelConfig{
		Index:    DataFieldIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"field_type": fieldTypeOptions,
			"stat_type":  dataFieldStatTypeOptions,
			"status":     statusOptions,
		},
		Relations: []orm.Relation{
			dataTemplateRelation,
			dataFieldOptionRelation,
			dataFieldStatRelation,
		},
	})
}
