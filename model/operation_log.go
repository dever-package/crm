package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type OperationLog struct {
	ID                   uint64    `dorm:"primaryKey;autoIncrement;comment:操作记录ID"`
	CustomerID           uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID              uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	WorkflowInstanceID   uint64    `dorm:"type:bigint;not null;default:0;comment:流程实例"`
	CustomerProductID    uint64    `dorm:"type:bigint;not null;default:0;comment:客户产品"`
	WorkflowID           uint64    `dorm:"type:bigint;not null;default:0;comment:流程"`
	StageID              uint64    `dorm:"type:bigint;not null;default:0;comment:阶段"`
	TaskID               uint64    `dorm:"type:bigint;not null;default:0;comment:任务"`
	TaskType             string    `dorm:"type:varchar(32);not null;default:'';comment:任务动作"`
	ResultValue          string    `dorm:"type:varchar(64);not null;default:'';comment:操作结果"`
	Title                string    `dorm:"type:varchar(128);not null;default:'';comment:标题"`
	Content              string    `dorm:"type:text;not null;default:'';comment:内容"`
	DataSnapshotJSON     string    `dorm:"type:text;not null;default:'{}';comment:数据快照JSON"`
	OperatorStaffID      uint64    `dorm:"type:bigint;not null;default:0;comment:操作人员"`
	OperatorDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:操作部门"`
	CreatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type OperationLogIndex struct {
	CustomerTime struct{} `index:"customer_id,created_at,id"`
	AssetTime    struct{} `index:"asset_id,created_at,id"`
	InstanceTime struct{} `index:"workflow_instance_id,created_at,id"`
	ProductTime  struct{} `index:"customer_product_id,created_at,id"`
	TaskTime     struct{} `index:"task_id,created_at,id"`
	OperatorTime struct{} `index:"operator_staff_id,created_at,id"`
	StageTime    struct{} `index:"workflow_id,stage_id,created_at,id"`
}

func NewOperationLogModel() *orm.Model[OperationLog] {
	return orm.LoadModel[OperationLog]("操作记录", "crm_operation_log", orm.ModelConfig{
		Index:    OperationLogIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"task_type": taskTypeOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			workflowInstanceRelation,
			customerProductRelation,
			workflowRelation,
			stageRelation,
			taskRelation,
			operatorStaffRelation,
			operatorDepartmentRelation,
		},
	})
}
