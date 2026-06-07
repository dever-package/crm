package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DashboardFunnelStep struct {
	ID         uint64    `dorm:"primaryKey;autoIncrement;comment:漏斗步骤ID"`
	FunnelID   uint64    `dorm:"type:bigint;not null;comment:漏斗"`
	Name       string    `dorm:"type:varchar(128);not null;comment:步骤名称"`
	MetricKey  string    `dorm:"type:varchar(64);not null;comment:指标标识"`
	MatchValue string    `dorm:"type:varchar(128);not null;default:'';comment:匹配值"`
	Sort       int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status     int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt  time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt  time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DashboardFunnelStepIndex struct {
	FunnelStatus struct{} `index:"funnel_id,status,sort,id"`
	MetricStatus struct{} `index:"metric_key,status,id"`
}

func NewDashboardFunnelStepModel() *orm.Model[DashboardFunnelStep] {
	return orm.LoadModel[DashboardFunnelStep]("漏斗步骤", "crm_dashboard_funnel_step", orm.ModelConfig{
		Index:    DashboardFunnelStepIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{dashboardFunnelRelation},
	})
}
