package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type TaskTemplate struct {
	ID          uint64    `dorm:"primaryKey;autoIncrement;comment:任务ID"`
	CateID      uint64    `dorm:"type:bigint;not null;default:1;comment:任务分类"`
	Name        string    `dorm:"type:varchar(128);not null;comment:任务名"`
	Description string    `dorm:"type:text;not null;default:'';comment:任务描述"`
	Sort        int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status      int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type TaskTemplateIndex struct {
	CateName   struct{} `unique:"cate_id,name"`
	CateStatus struct{} `index:"cate_id,status,sort,id"`
}

var taskTemplateFieldRelation = orm.Relation{
	Field:      "fields",
	Through:    "crm.NewTaskFieldModel",
	OwnerField: "task_template_id",
	Order:      "sort asc,id asc",
}

func NewTaskTemplateModel() *orm.Model[TaskTemplate] {
	return orm.LoadModel[TaskTemplate]("任务管理", "crm_task_template", orm.ModelConfig{
		Index:    TaskTemplateIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			taskTemplateCateRelation,
			taskTemplateFieldRelation,
		},
	})
}
