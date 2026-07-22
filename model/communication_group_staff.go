package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const (
	CommunicationGroupStaffParticipant = "participant"
	CommunicationGroupStaffNPLOwner    = "npl_owner"
	CommunicationGroupStaffPMOwner     = "pm_owner"
	CommunicationGroupStaffALAOwner    = "ala_owner"
)

var communicationGroupStaffRoleOptions = []map[string]any{
	{"id": CommunicationGroupStaffParticipant, "value": "关联人员"},
	{"id": CommunicationGroupStaffNPLOwner, "value": "NPL负责人"},
	{"id": CommunicationGroupStaffPMOwner, "value": "PM负责人"},
	{"id": CommunicationGroupStaffALAOwner, "value": "ALA负责人"},
}

func CommunicationGroupStaffRoleName(role string) string {
	return crmOptionName(communicationGroupStaffRoleOptions, role)
}

type CommunicationGroupStaff struct {
	ID                   uint64    `dorm:"primaryKey;autoIncrement;comment:群关联人员ID"`
	CommunicationGroupID uint64    `dorm:"type:bigint;not null;comment:沟通群"`
	StaffID              uint64    `dorm:"type:bigint;not null;comment:关联人员"`
	Role                 string    `dorm:"type:varchar(32);not null;default:'participant';comment:关联角色"`
	CreatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type CommunicationGroupStaffIndex struct {
	GroupStaff struct{} `unique:"communication_group_id,staff_id"`
	StaffGroup struct{} `index:"staff_id,communication_group_id,id"`
}

func NewCommunicationGroupStaffModel() *orm.Model[CommunicationGroupStaff] {
	return orm.LoadModel[CommunicationGroupStaff]("群关联人员", "crm_communication_group_staff", orm.ModelConfig{
		Index:    CommunicationGroupStaffIndex{},
		Order:    "id asc",
		Database: "default",
		Options: map[string]any{
			"role": communicationGroupStaffRoleOptions,
		},
		Relations: []orm.Relation{
			communicationGroupRelation,
			staffRelation,
		},
	})
}
