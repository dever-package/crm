package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type CustomerTag struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:标签ID"`
	LevelID   uint64    `dorm:"type:bigint;not null;comment:客户等级"`
	Name      string    `dorm:"type:varchar(128);not null;comment:标签名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type CustomerTagIndex struct {
	LevelName struct{} `unique:"level_id,name"`
	LevelSort struct{} `index:"level_id,status,sort,id"`
}

func NewCustomerTagModel() *orm.Model[CustomerTag] {
	return orm.LoadModel[CustomerTag]("客户标签", "crm_customer_tag", orm.ModelConfig{
		Index:    CustomerTagIndex{},
		Order:    "level_id asc,sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{customerLevelRelation},
	})
}
