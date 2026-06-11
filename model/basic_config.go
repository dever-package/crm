package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const (
	DefaultBasicConfigID      uint64 = 1
	DefaultCustomerCodePrefix        = ""
)

type BasicConfig struct {
	ID                 uint64    `dorm:"primaryKey;autoIncrement;comment:配置ID"`
	CustomerCodePrefix string    `dorm:"type:varchar(32);not null;default:'';comment:客户编号前缀"`
	CreatedAt          time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt          time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

var basicConfigSeed = []map[string]any{
	{
		"id":                   DefaultBasicConfigID,
		"customer_code_prefix": DefaultCustomerCodePrefix,
	},
}

func NewBasicConfigModel() *orm.Model[BasicConfig] {
	return orm.LoadModel[BasicConfig]("基本配置", "crm_basic_config", orm.ModelConfig{
		Seeds:    basicConfigSeed,
		Database: "default",
	})
}

func DefaultBasicConfig() BasicConfig {
	return BasicConfig{
		ID:                 DefaultBasicConfigID,
		CustomerCodePrefix: DefaultCustomerCodePrefix,
	}
}
