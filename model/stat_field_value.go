package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type StatFieldValue struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:统计字段值ID"`
	CustomerID     uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID        uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	DataTemplateID uint64    `dorm:"type:bigint;not null;comment:数据模板"`
	DataFieldID    uint64    `dorm:"type:bigint;not null;comment:数据字段"`
	FieldKey       string    `dorm:"type:varchar(128);not null;comment:字段编码"`
	FieldName      string    `dorm:"type:varchar(128);not null;comment:字段名称"`
	FieldType      string    `dorm:"type:varchar(32);not null;default:'text';comment:字段类型"`
	StatType       string    `dorm:"type:varchar(32);not null;default:'dimension';comment:条件值类型"`
	StatGroup      string    `dorm:"type:varchar(64);not null;default:'';comment:条件分组"`
	ValueText      string    `dorm:"type:text;not null;default:'';comment:文本值"`
	ValueNumber    float64   `dorm:"type:double precision;not null;default:0;comment:数值"`
	ValueDate      time.Time `dorm:"comment:时间值"`
	ValueBool      bool      `dorm:"not null;default:false;comment:布尔值"`
	ValueJSON      string    `dorm:"type:text;not null;default:'{}';comment:原始值JSON"`
	Source         string    `dorm:"type:varchar(32);not null;default:'form';comment:来源"`
	TaskID         uint64    `dorm:"type:bigint;not null;default:0;comment:来源任务"`
	OperationLogID uint64    `dorm:"type:bigint;not null;default:0;comment:操作记录"`
	Status         int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type StatFieldValueIndex struct {
	CustomerDataField struct{} `unique:"customer_id,asset_id,data_field_id"`
	CustomerField     struct{} `index:"customer_id,asset_id,field_key"`
	CustomerTime      struct{} `index:"customer_id,updated_at,id"`
	AssetTime         struct{} `index:"asset_id,updated_at,id"`
	FieldValue        struct{} `index:"field_key,value_text,status,id"`
	FieldNumber       struct{} `index:"field_key,value_number,status,id"`
	FieldDate         struct{} `index:"field_key,value_date,status,id"`
	StatGroup         struct{} `index:"stat_group,stat_type,status,id"`
	TaskTime          struct{} `index:"task_id,updated_at,id"`
	Operation         struct{} `index:"operation_log_id,id"`
}

func NewStatFieldValueModel() *orm.Model[StatFieldValue] {
	return orm.LoadModel[StatFieldValue]("统计字段值", "crm_stat_field_value", orm.ModelConfig{
		Index:    StatFieldValueIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"stat_type": dataFieldStatTypeOptions,
			"source": []map[string]any{
				{"id": StatValueSourceForm, "value": "表单"},
				{"id": StatValueSourceTransition, "value": "流转"},
				{"id": StatValueSourceTask, "value": "任务"},
			},
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			dataTemplateRelation,
			dataFieldRelation,
			taskRelation,
		},
	})
}
