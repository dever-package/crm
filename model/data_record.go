package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DataRecord struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:数据记录ID"`
	ResourceID     uint64    `dorm:"type:bigint;not null;comment:客户资产"`
	DataTemplateID uint64    `dorm:"type:bigint;not null;comment:数据模板"`
	TaskID         uint64    `dorm:"type:bigint;not null;default:0;comment:来源任务"`
	RecordJSON     string    `dorm:"type:text;not null;default:'{}';comment:记录JSON"`
	Summary        string    `dorm:"type:varchar(255);not null;default:'';comment:摘要"`
	Status         int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort           int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DataRecordIndex struct {
	ResourceTemplate struct{} `index:"resource_id,data_template_id,status,id"`
	TaskTemplate     struct{} `index:"task_id,data_template_id,id"`
	TemplateStatus   struct{} `index:"data_template_id,status,id"`
}

func NewDataRecordModel() *orm.Model[DataRecord] {
	return orm.LoadModel[DataRecord]("沉淀数据", "crm_data_record", orm.ModelConfig{
		Index:    DataRecordIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			resourceRelation,
			dataTemplateRelation,
			taskRelation,
		},
	})
}
