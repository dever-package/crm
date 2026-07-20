package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DataField struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:数据字段ID"`
	DataTemplateID uint64    `dorm:"type:bigint;not null;comment:数据模板"`
	ParentFieldID  uint64    `dorm:"type:bigint;not null;default:0;comment:父级字段"`
	OptionSetID    uint64    `dorm:"type:bigint;not null;default:0;comment:常用选项集"`
	Name           string    `dorm:"type:varchar(128);not null;comment:字段名称"`
	FieldKey       string    `dorm:"type:varchar(128);not null;default:'';comment:字段编码"`
	FieldType      string    `dorm:"type:varchar(32);not null;default:'text';comment:字段类型"`
	DefaultValue   string    `dorm:"type:text;not null;default:'';comment:默认值"`
	FinanceTypeID  uint64    `dorm:"type:bigint;not null;default:0;comment:财务类型"`
	StatEnabled    bool      `dorm:"not null;default:false;comment:是否参与统计"`
	Sort           int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status         int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DataFieldIndex struct {
	FieldKey       struct{} `unique:"field_key"`
	ParentStatus   struct{} `index:"parent_field_id,status,sort,id"`
	TemplateStatus struct{} `index:"data_template_id,status,sort,id"`
	TypeStatus     struct{} `index:"field_type,status,id"`
	OptionSet      struct{} `index:"option_set_id,status,id"`
	FinanceStatus  struct{} `index:"finance_type_id,status,id"`
	StatStatus     struct{} `index:"stat_enabled,status,sort,id"`
}

var dataFieldOptionRelation = orm.Relation{
	Field:      "options",
	Through:    "crm.NewDataFieldOptionModel",
	OwnerField: "data_field_id",
	Order:      "sort asc,id asc",
}

var dataFieldParentRelation = orm.Relation{
	Field:      "parent_field_id",
	Option:     "crm.NewDataFieldModel",
	OptionKeys: []string{"name", "field_key", "field_type"},
}

var dataFieldOptionSetRelation = orm.Relation{
	Field:      "option_set_id",
	Option:     "crm.NewOptionSetModel",
	OptionKeys: []string{"name"},
}

var dataFieldChildRelation = orm.Relation{
	Field:      "children",
	Through:    "crm.NewDataFieldModel",
	OwnerField: "parent_field_id",
	Order:      "sort asc,id asc",
}

func NewDataFieldModel() *orm.Model[DataField] {
	return orm.LoadModel[DataField]("业务数据字段", "crm_data_field", orm.ModelConfig{
		Index:    DataFieldIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"field_type": fieldTypeOptions,
			"status":     statusOptions,
		},
		Relations: []orm.Relation{
			dataTemplateRelation,
			dataFieldParentRelation,
			dataFieldOptionSetRelation,
			financeTypeRelation,
			dataFieldOptionRelation,
			dataFieldChildRelation,
		},
	})
}
