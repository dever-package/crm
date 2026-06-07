package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type AuditLog struct {
	ID         uint64    `dorm:"primaryKey;autoIncrement;comment:日志ID"`
	ResourceID uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	TaskID     uint64    `dorm:"type:bigint;not null;default:0;comment:任务"`
	OperatorID uint64    `dorm:"type:bigint;not null;default:0;comment:操作人"`
	Action     string    `dorm:"type:varchar(64);not null;comment:动作"`
	Module     string    `dorm:"type:varchar(64);not null;comment:模块"`
	Content    string    `dorm:"type:text;not null;default:'';comment:内容"`
	CreatedAt  time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type AuditLogIndex struct {
	ResourceTime struct{} `index:"resource_id,created_at,id"`
	TaskTime     struct{} `index:"task_id,created_at,id"`
	OperatorTime struct{} `index:"operator_id,created_at,id"`
	ActionTime   struct{} `index:"action,created_at,id"`
}

func NewAuditLogModel() *orm.Model[AuditLog] {
	return orm.LoadModel[AuditLog]("CRM操作日志", "crm_audit_log", orm.ModelConfig{
		Index:    AuditLogIndex{},
		Order:    "id desc",
		Database: "default",
		Relations: []orm.Relation{
			resourceRelation,
			taskRelation,
			operatorRelation,
		},
	})
}
