package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type StatEvent struct {
	ID                   uint64    `dorm:"primaryKey;autoIncrement;comment:统计事件ID"`
	EventType            string    `dorm:"type:varchar(32);not null;comment:事件类型"`
	EventKey             string    `dorm:"type:varchar(128);not null;comment:事件编码"`
	CustomerID           uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID              uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	WorkflowInstanceID   uint64    `dorm:"type:bigint;not null;default:0;comment:流程实例"`
	CustomerProductID    uint64    `dorm:"type:bigint;not null;default:0;comment:客户产品"`
	WorkflowID           uint64    `dorm:"type:bigint;not null;default:0;comment:流程"`
	StageID              uint64    `dorm:"type:bigint;not null;default:0;comment:阶段"`
	FromStageID          uint64    `dorm:"type:bigint;not null;default:0;comment:来源阶段"`
	ToStageID            uint64    `dorm:"type:bigint;not null;default:0;comment:目标阶段"`
	TaskID               uint64    `dorm:"type:bigint;not null;default:0;comment:任务"`
	TaskType             string    `dorm:"type:varchar(32);not null;default:'';comment:任务动作"`
	ResultValue          string    `dorm:"type:varchar(64);not null;default:'';comment:结果值"`
	OperationLogID       uint64    `dorm:"type:bigint;not null;default:0;comment:操作记录"`
	OperatorStaffID      uint64    `dorm:"type:bigint;not null;default:0;comment:操作人员"`
	OperatorDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:操作部门"`
	EventAt              time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:事件时间"`
	CreatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type StatEventIndex struct {
	EventOperation struct{} `unique:"event_key,operation_log_id"`
	EventTime      struct{} `index:"event_type,event_at,id"`
	CustomerTime   struct{} `index:"customer_id,event_at,id"`
	AssetTime      struct{} `index:"asset_id,event_at,id"`
	InstanceTime   struct{} `index:"workflow_instance_id,event_at,id"`
	ProductTime    struct{} `index:"customer_product_id,event_at,id"`
	StageTime      struct{} `index:"workflow_id,stage_id,event_at,id"`
	TransitionTime struct{} `index:"from_stage_id,to_stage_id,event_at,id"`
	TaskValue      struct{} `index:"task_id,result_value,event_at,id"`
	OperatorTime   struct{} `index:"operator_staff_id,event_at,id"`
}

func NewStatEventModel() *orm.Model[StatEvent] {
	return orm.LoadModel[StatEvent]("统计事件", "crm_stat_event", orm.ModelConfig{
		Index:    StatEventIndex{},
		Order:    "event_at desc,id desc",
		Database: "default",
		Options: map[string]any{
			"event_type": statEventTypeOptions,
			"task_type":  taskTypeOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			workflowInstanceRelation,
			customerProductRelation,
			workflowRelation,
			stageRelation,
			fromStageRelation,
			toStageRelation,
			taskRelation,
			operatorStaffRelation,
			operatorDepartmentRelation,
		},
	})
}
