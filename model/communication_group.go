package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const (
	CommunicationGroupStatusActive    = "active"
	CommunicationGroupStatusDissolved = "dissolved"
)

var communicationGroupStatusOptions = []map[string]any{
	{"id": CommunicationGroupStatusActive, "value": "使用中"},
	{"id": CommunicationGroupStatusDissolved, "value": "已解散"},
}

func CommunicationGroupStatusName(status string) string {
	return crmOptionName(communicationGroupStatusOptions, status)
}

type CommunicationGroup struct {
	ID                 uint64     `dorm:"primaryKey;autoIncrement;comment:沟通群ID"`
	CustomerID         uint64     `dorm:"type:bigint;not null;comment:客户"`
	AssetID            uint64     `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	WorkflowInstanceID uint64     `dorm:"type:bigint;not null;comment:案件流程"`
	GroupTypeID        uint64     `dorm:"type:bigint;not null;comment:群类型"`
	Name               string     `dorm:"type:varchar(160);not null;comment:群名称"`
	ExternalGroupID    string     `dorm:"type:varchar(160);not null;default:'';comment:外部群ID"`
	Status             string     `dorm:"type:varchar(32);not null;default:'active';comment:群状态"`
	EstablishedAt      time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:建群日期"`
	DissolvedAt        *time.Time `dorm:"null;comment:解散日期"`
	DissolveReason     string     `dorm:"type:text;not null;default:'';comment:解散原因"`
	Summary            string     `dorm:"type:text;not null;default:'';comment:智能总结"`
	Remark             string     `dorm:"type:text;not null;default:'';comment:备注"`
	SourceKey          *string    `dorm:"type:varchar(192);null;comment:导入来源唯一键"`
	CreatedByStaffID   uint64     `dorm:"type:bigint;not null;default:0;comment:创建人员"`
	CreatedAt          time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt          time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type CommunicationGroupIndex struct {
	SourceKey      struct{} `unique:"source_key"`
	InstanceStatus struct{} `index:"workflow_instance_id,status,established_at,id"`
	CustomerStatus struct{} `index:"customer_id,status,established_at,id"`
	AssetStatus    struct{} `index:"asset_id,status,established_at,id"`
	TypeStatus     struct{} `index:"group_type_id,status,established_at,id"`
	ExternalGroup  struct{} `index:"group_type_id,external_group_id,id"`
}

func NewCommunicationGroupModel() *orm.Model[CommunicationGroup] {
	return orm.LoadModel[CommunicationGroup]("沟通群", "crm_communication_group", orm.ModelConfig{
		Index:    CommunicationGroupIndex{},
		Order:    "status asc,established_at desc,id desc",
		Database: "default",
		Options: map[string]any{
			"status": communicationGroupStatusOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			workflowInstanceRelation,
			communicationGroupTypeRelation,
			createdByStaffRelation,
		},
	})
}
