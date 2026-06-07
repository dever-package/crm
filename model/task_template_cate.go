package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const DefaultTaskTemplateCateID uint64 = 1

type TaskTemplateCate struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:任务分类ID"`
	Name      string    `dorm:"type:varchar(128);not null;comment:名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type TaskTemplateCateIndex struct {
	Name       struct{} `unique:"name"`
	StatusSort struct{} `index:"status,sort,id"`
}

var taskTemplateCateSeed = []map[string]any{
	{"id": DefaultTaskTemplateCateID, "name": "默认分类", "status": StatusEnabled, "sort": 10},
}

func NewTaskTemplateCateModel() *orm.Model[TaskTemplateCate] {
	return orm.LoadModel[TaskTemplateCate]("任务分类", "crm_task_template_cate", orm.ModelConfig{
		Index:    TaskTemplateCateIndex{},
		Seeds:    taskTemplateCateSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
