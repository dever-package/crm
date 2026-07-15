package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const DefaultDouyinConfigID uint64 = 1

type DouyinConfig struct {
	ID                 uint64     `dorm:"primaryKey;autoIncrement;comment:配置ID"`
	Enabled            bool       `dorm:"type:boolean;not null;default:false;comment:启用同步"`
	ClientKey          string     `dorm:"type:varchar(128);not null;default:'';comment:Client Key"`
	ClientSecret       string     `dorm:"type:varchar(255);not null;default:'';comment:Client Secret"`
	AccountID          string     `dorm:"type:varchar(128);not null;default:'';comment:来客账户ID"`
	RootLifeAccountID  string     `dorm:"type:varchar(128);not null;default:'';comment:根账户ID"`
	LastSyncStartedAt  *time.Time `dorm:"null;comment:最近同步开始时间"`
	LastSyncFinishedAt *time.Time `dorm:"null;comment:最近同步结束时间"`
	LastSyncCursorAt   *time.Time `dorm:"null;comment:增量同步游标"`
	LastSyncStatus     string     `dorm:"type:varchar(24);not null;default:'';comment:最近同步状态"`
	LastSyncMessage    string     `dorm:"type:text;not null;default:'';comment:最近同步结果"`
	LastSyncedCount    int        `dorm:"type:int;not null;default:0;comment:最近入库数量"`
	CreatedAt          time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt          time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

var douyinConfigSeed = []map[string]any{
	{
		"id":      DefaultDouyinConfigID,
		"enabled": false,
	},
}

func NewDouyinConfigModel() *orm.Model[DouyinConfig] {
	return orm.LoadModel[DouyinConfig]("抖音线索同步配置", "crm_douyin_config", orm.ModelConfig{
		Seeds:    douyinConfigSeed,
		Database: "default",
	})
}
