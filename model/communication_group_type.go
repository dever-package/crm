package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const CommunicationGroupTypeEnterpriseWechat = "enterprise_wechat"

type CommunicationGroupType struct {
	ID          uint64    `dorm:"primaryKey;autoIncrement;comment:沟通群类型ID"`
	Code        string    `dorm:"type:varchar(64);not null;comment:类型编码"`
	Name        string    `dorm:"type:varchar(128);not null;comment:类型名称"`
	Description string    `dorm:"type:text;not null;default:'';comment:说明"`
	Status      int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort        int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type CommunicationGroupTypeIndex struct {
	Code       struct{} `unique:"code"`
	StatusSort struct{} `index:"status,sort,id"`
}

var communicationGroupTypeSeed = []map[string]any{
	{
		"code":        CommunicationGroupTypeEnterpriseWechat,
		"name":        "企业微信",
		"description": "企业微信客户沟通群。",
		"status":      StatusEnabled,
		"sort":        10,
	},
}

func NewCommunicationGroupTypeModel() *orm.Model[CommunicationGroupType] {
	return orm.LoadModel[CommunicationGroupType]("沟通群类型", "crm_communication_group_type", orm.ModelConfig{
		Index:    CommunicationGroupTypeIndex{},
		Seeds:    communicationGroupTypeSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
