package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Dashboard struct {
	ID           uint64    `dorm:"primaryKey;autoIncrement;comment:看板ID"`
	Name         string    `dorm:"type:varchar(128);not null;comment:看板名称"`
	ResourceType string    `dorm:"type:varchar(32);not null;default:'asset_disposal';comment:资源类型"`
	Description  string    `dorm:"type:text;not null;default:'';comment:描述"`
	ConfigJSON   string    `dorm:"type:text;not null;default:'{}';comment:配置JSON"`
	Status       int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort         int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DashboardIndex struct {
	ResourceName   struct{} `unique:"resource_type,name"`
	ResourceStatus struct{} `index:"resource_type,status,sort,id"`
}

func NewDashboardModel() *orm.Model[Dashboard] {
	return orm.LoadModel[Dashboard]("CRM看板", "crm_dashboard", orm.ModelConfig{
		Index:    DashboardIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"resource_type": resourceTypeOptions,
			"status":        statusOptions,
		},
	})
}
