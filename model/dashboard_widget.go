package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DashboardWidget struct {
	ID          uint64    `dorm:"primaryKey;autoIncrement;comment:看板组件ID"`
	DashboardID uint64    `dorm:"type:bigint;not null;comment:看板"`
	Name        string    `dorm:"type:varchar(128);not null;comment:组件名称"`
	WidgetType  string    `dorm:"type:varchar(32);not null;default:'card';comment:组件类型"`
	MetricKey   string    `dorm:"type:varchar(64);not null;default:'';comment:指标标识"`
	ConfigJSON  string    `dorm:"type:text;not null;default:'{}';comment:配置JSON"`
	Sort        int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status      int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DashboardWidgetIndex struct {
	DashboardStatus struct{} `index:"dashboard_id,status,sort,id"`
	MetricStatus    struct{} `index:"metric_key,status,id"`
}

func NewDashboardWidgetModel() *orm.Model[DashboardWidget] {
	return orm.LoadModel[DashboardWidget]("看板组件", "crm_dashboard_widget", orm.ModelConfig{
		Index:    DashboardWidgetIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{dashboardRelation},
	})
}
