package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type ScheduleEvent struct {
	ID                       uint64     `dorm:"primaryKey;autoIncrement;comment:日程ID"`
	ScheduleType             string     `dorm:"type:varchar(32);not null;default:'personal';comment:日程类型"`
	CustomerID               uint64     `dorm:"type:bigint;not null;default:0;comment:客户"`
	PendingCustomerKey       *string    `dorm:"type:varchar(64);null;comment:待跟进客户唯一键"`
	OwnerStaffID             uint64     `dorm:"type:bigint;not null;comment:负责人"`
	CreatedByStaffID         uint64     `dorm:"type:bigint;not null;comment:创建人"`
	SourceWorkflowInstanceID uint64     `dorm:"type:bigint;not null;default:0;comment:来源流程实例"`
	DataUsageFieldID         uint64     `dorm:"type:bigint;not null;default:0;comment:用途字段绑定"`
	DataRecordID             uint64     `dorm:"type:bigint;not null;default:0;comment:客户资料记录"`
	DataFieldID              uint64     `dorm:"type:bigint;not null;default:0;comment:客户资料字段"`
	OperationLogID           uint64     `dorm:"type:bigint;not null;default:0;comment:首次安排记录"`
	Title                    string     `dorm:"type:varchar(128);not null;comment:标题"`
	Remark                   string     `dorm:"type:text;not null;default:'';comment:备注"`
	StartAt                  time.Time  `dorm:"not null;comment:开始时间"`
	EndAt                    time.Time  `dorm:"not null;comment:结束时间"`
	ReminderMinutes          int        `dorm:"type:int;not null;default:0;comment:提前提醒分钟"`
	RemindAt                 time.Time  `dorm:"not null;comment:提醒时间"`
	Source                   string     `dorm:"type:varchar(32);not null;default:'calendar';comment:创建来源"`
	Status                   string     `dorm:"type:varchar(32);not null;default:'pending';comment:状态"`
	CompletedAt              *time.Time `dorm:"null;comment:完成时间"`
	CanceledAt               *time.Time `dorm:"null;comment:取消时间"`
	CreatedAt                time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt                time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type ScheduleEventIndex struct {
	PendingCustomer struct{} `unique:"pending_customer_key"`
	OwnerStatus     struct{} `index:"owner_staff_id,status,start_at,id"`
	CustomerStatus  struct{} `index:"customer_id,schedule_type,status,start_at,id"`
	ReminderStatus  struct{} `index:"remind_at,status,id"`
	SourceWorkflow  struct{} `index:"source_workflow_instance_id,id"`
	DataRecordField struct{} `index:"data_record_id,data_field_id,status,id"`
}

func NewScheduleEventModel() *orm.Model[ScheduleEvent] {
	return orm.LoadModel[ScheduleEvent]("工作日程", "crm_schedule_event", orm.ModelConfig{
		Index:    ScheduleEventIndex{},
		Order:    "start_at asc,id asc",
		Database: "default",
		Options: map[string]any{
			"schedule_type":    scheduleTypeOptions,
			"reminder_minutes": scheduleReminderOptions,
			"source":           scheduleSourceOptions,
			"status":           scheduleStatusOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			ownerStaffRelation,
			createdByStaffRelation,
			sourceWorkflowInstanceRelation,
			dataUsageFieldIDRelation,
			dataRecordRelation,
			dataFieldRelation,
			operationLogRelation,
		},
	})
}
