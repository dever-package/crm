package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type StageTransitionLog struct {
	ID              uint64    `dorm:"primaryKey;autoIncrement;comment:阶段流转日志ID"`
	CustomerID      uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID         uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	FromStageCode   string    `dorm:"type:varchar(32);not null;default:'';comment:来源阶段"`
	ToStageCode     string    `dorm:"type:varchar(32);not null;default:'';comment:目标阶段"`
	TaskID          uint64    `dorm:"type:bigint;not null;default:0;comment:任务"`
	ResultValue     string    `dorm:"type:varchar(64);not null;default:'';comment:匹配结果"`
	OperationLogID  uint64    `dorm:"type:bigint;not null;default:0;comment:操作记录"`
	OperatorStaffID uint64    `dorm:"type:bigint;not null;default:0;comment:操作人员"`
	CreatedAt       time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type StageTransitionLogIndex struct {
	CustomerTime struct{} `index:"customer_id,created_at,id"`
	AssetTime    struct{} `index:"asset_id,created_at,id"`
	FromTo       struct{} `index:"from_stage_code,to_stage_code,created_at,id"`
	ResultTime   struct{} `index:"result_value,created_at,id"`
	Operation    struct{} `index:"operation_log_id,id"`
}

func NewStageTransitionLogModel() *orm.Model[StageTransitionLog] {
	return orm.LoadModel[StageTransitionLog]("阶段流转日志", "crm_stage_transition_log", orm.ModelConfig{
		Index:    StageTransitionLogIndex{},
		Order:    "id desc",
		Database: "default",
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			fromStageRelation,
			toStageRelation,
			taskRelation,
			operatorStaffRelation,
		},
	})
}
