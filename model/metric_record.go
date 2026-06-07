package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type MetricRecord struct {
	ID                uint64    `dorm:"primaryKey;autoIncrement;comment:统计明细ID"`
	ResourceID        uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	TaskID            uint64    `dorm:"type:bigint;not null;default:0;comment:任务"`
	DataTemplateID    uint64    `dorm:"type:bigint;not null;default:0;comment:数据模板"`
	FlowTemplateID    uint64    `dorm:"type:bigint;not null;default:0;comment:流程模板"`
	FlowReleaseID     uint64    `dorm:"type:bigint;not null;default:0;comment:流程版本"`
	StageID           uint64    `dorm:"type:bigint;not null;default:0;comment:阶段"`
	TaskTemplateID    uint64    `dorm:"type:bigint;not null;default:0;comment:任务模板"`
	DepartmentID      uint64    `dorm:"type:bigint;not null;default:0;comment:部门"`
	StaffID           uint64    `dorm:"type:bigint;not null;default:0;comment:人员"`
	MetricKey         string    `dorm:"type:varchar(64);not null;comment:指标标识"`
	MetricValueText   string    `dorm:"type:varchar(255);not null;default:'';comment:文本值"`
	MetricValueNumber float64   `dorm:"type:decimal(18,4);not null;default:0;comment:数值"`
	MetricTime        time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:指标时间"`
	CreatedAt         time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type MetricRecordIndex struct {
	KeyTime      struct{} `index:"metric_key,metric_time,id"`
	ResourceKey  struct{} `index:"resource_id,metric_key,id"`
	TaskKey      struct{} `index:"task_id,metric_key,id"`
	Department   struct{} `index:"department_id,metric_key,metric_time"`
	Staff        struct{} `index:"staff_id,metric_key,metric_time"`
	FlowStageKey struct{} `index:"flow_template_id,stage_id,metric_key"`
}

func NewMetricRecordModel() *orm.Model[MetricRecord] {
	return orm.LoadModel[MetricRecord]("统计明细", "crm_metric_record", orm.ModelConfig{
		Index:    MetricRecordIndex{},
		Order:    "id desc",
		Database: "default",
		Relations: []orm.Relation{
			resourceRelation,
			taskRelation,
			dataTemplateRelation,
			flowTemplateRelation,
			flowReleaseRelation,
			flowStageRelation,
			taskTemplateRelation,
			departmentRelation,
			staffRelation,
		},
	})
}
