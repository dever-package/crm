package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Customer struct {
	ID               uint64    `dorm:"primaryKey;autoIncrement;comment:客户ID"`
	Code             string    `dorm:"type:varchar(32);not null;default:'';comment:客户编号"`
	Name             string    `dorm:"type:varchar(64);not null;comment:姓名"`
	Phone            string    `dorm:"type:varchar(32);not null;comment:手机号"`
	Wechat           string    `dorm:"type:varchar(64);not null;default:'';comment:微信"`
	IDCard           string    `dorm:"type:varchar(32);not null;default:'';comment:身份证号"`
	Gender           string    `dorm:"type:varchar(16);not null;default:'unknown';comment:性别"`
	SourceID         uint64    `dorm:"type:bigint;not null;default:1;comment:来源"`
	ChannelID        uint64    `dorm:"type:bigint;not null;default:1;comment:渠道"`
	LevelID          uint64    `dorm:"type:bigint;not null;default:1;comment:客户等级"`
	Tags             string    `dorm:"type:varchar(255);not null;default:'';comment:标签"`
	Remark           string    `dorm:"type:text;not null;default:'';comment:备注"`
	CreatedByStaffID uint64    `dorm:"type:bigint;not null;default:0;comment:创建人员"`
	CreatedAt        time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt        time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type CustomerIndex struct {
	Code        struct{} `unique:"code"`
	Phone       struct{} `index:"phone,id"`
	SourceLevel struct{} `index:"source_id,channel_id,level_id,id"`
	Creator     struct{} `index:"created_by_staff_id,id"`
}

func NewCustomerModel() *orm.Model[Customer] {
	return orm.LoadModel[Customer]("客户信息", "crm_customer", orm.ModelConfig{
		Index:    CustomerIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"gender": customerGenderOptions,
		},
		Relations: []orm.Relation{
			customerSourceRelation,
			customerChannelRelation,
			customerLevelRelation,
			createdByStaffRelation,
		},
	})
}
