package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type ScheduleParticipant struct {
	ID              uint64     `dorm:"primaryKey;autoIncrement;comment:日程参与人ID"`
	ScheduleEventID uint64     `dorm:"type:bigint;not null;comment:日程"`
	StaffID         uint64     `dorm:"type:bigint;not null;comment:参与人员"`
	Role            string     `dorm:"type:varchar(32);not null;default:'participant';comment:参与角色"`
	CheckedInAt     *time.Time `dorm:"null;comment:签到时间"`
	WorkbenchReadAt *time.Time `dorm:"null;comment:工作台已读时间"`
	FeishuSentAt    *time.Time `dorm:"null;comment:飞书发送时间"`
	FeishuClaimedAt *time.Time `dorm:"null;comment:飞书发送占用时间"`
	FeishuAttempts  int        `dorm:"type:int;not null;default:0;comment:飞书发送次数"`
	FeishuLastError string     `dorm:"type:text;not null;default:'';comment:飞书最后错误"`
	CreatedAt       time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt       time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type ScheduleParticipantIndex struct {
	EventStaff  struct{} `unique:"schedule_event_id,staff_id"`
	StaffEvent  struct{} `index:"staff_id,schedule_event_id,id"`
	FeishuState struct{} `index:"feishu_sent_at,feishu_attempts,id"`
}

func NewScheduleParticipantModel() *orm.Model[ScheduleParticipant] {
	return orm.LoadModel[ScheduleParticipant]("日程参与人", "crm_schedule_participant", orm.ModelConfig{
		Index:    ScheduleParticipantIndex{},
		Order:    "id asc",
		Database: "default",
		Options: map[string]any{
			"role": memberRelationOptions,
		},
		Relations: []orm.Relation{
			scheduleEventRelation,
			staffRelation,
		},
	})
}
