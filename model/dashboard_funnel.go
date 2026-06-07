package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DashboardFunnel struct {
	ID           uint64    `dorm:"primaryKey;autoIncrement;comment:漏斗ID"`
	DashboardID  uint64    `dorm:"type:bigint;not null;comment:看板"`
	Name         string    `dorm:"type:varchar(128);not null;comment:漏斗名称"`
	ResourceType string    `dorm:"type:varchar(32);not null;default:'asset_disposal';comment:资源类型"`
	ConfigJSON   string    `dorm:"type:text;not null;default:'{}';comment:配置JSON"`
	Sort         int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status       int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DashboardFunnelIndex struct {
	DashboardStatus struct{} `index:"dashboard_id,status,sort,id"`
	ResourceStatus  struct{} `index:"resource_type,status,id"`
}

func NewDashboardFunnelModel() *orm.Model[DashboardFunnel] {
	return orm.LoadModel[DashboardFunnel]("看板漏斗", "crm_dashboard_funnel", orm.ModelConfig{
		Index:    DashboardFunnelIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"resource_type": resourceTypeOptions,
			"status":        statusOptions,
		},
		Relations: []orm.Relation{dashboardRelation},
	})
}
