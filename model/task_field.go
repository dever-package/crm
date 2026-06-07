package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type TaskField struct {
	ID                 uint64    `dorm:"primaryKey;autoIncrement;comment:收集数据ID"`
	TaskTemplateID     uint64    `dorm:"type:bigint;not null;comment:任务"`
	DataTemplateCateID uint64    `dorm:"type:bigint;not null;default:1;comment:数据模板分类"`
	DataTemplateID     uint64    `dorm:"type:bigint;not null;default:0;comment:数据模板"`
	FieldSource        string    `dorm:"type:varchar(96);not null;default:'';comment:字段来源"`
	CollectPath        string    `dorm:"type:text;not null;default:'[]';comment:收集数据路径"`
	MainField          string    `dorm:"type:varchar(64);not null;default:'';comment:主表字段"`
	DataFieldID        uint64    `dorm:"type:bigint;not null;default:0;comment:数据模板字段"`
	Name               string    `dorm:"type:varchar(128);not null;comment:收集数据"`
	Required           bool      `dorm:"not null;default:true;comment:是否必填"`
	Sort               int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status             int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt          time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt          time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type TaskFieldIndex struct {
	TaskStatus struct{} `index:"task_template_id,status,sort,id"`
	CateStatus struct{} `index:"data_template_cate_id,status,id"`
	Template   struct{} `index:"data_template_id,data_field_id"`
	Source     struct{} `index:"field_source"`
	MainField  struct{} `index:"main_field"`
}

func NewTaskFieldModel() *orm.Model[TaskField] {
	return orm.LoadModel[TaskField]("收集数据", "crm_task_field", orm.ModelConfig{
		Index:    TaskFieldIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			taskTemplateRelation,
			taskFieldDataTemplateCateRelation,
			taskFieldDataTemplateRelation,
			taskFieldDataFieldRelation,
		},
	})
}
