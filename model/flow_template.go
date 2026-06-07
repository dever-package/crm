package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type FlowTemplate struct {
	ID               uint64    `dorm:"primaryKey;autoIncrement;comment:流程模板ID"`
	Name             string    `dorm:"type:varchar(128);not null;comment:流程名称"`
	Description      string    `dorm:"type:text;not null;default:'';comment:描述"`
	ResourceType     string    `dorm:"type:varchar(32);not null;default:'asset_disposal';comment:资源类型"`
	PublishStatus    string    `dorm:"type:varchar(32);not null;default:'draft';comment:发布状态"`
	CurrentReleaseID uint64    `dorm:"type:bigint;not null;default:0;comment:当前发布版本"`
	ReleaseVersion   int       `dorm:"type:int;not null;default:0;comment:发布版本号"`
	ConfigJSON       string    `dorm:"type:text;not null;default:'{}';comment:配置JSON"`
	Status           int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort             int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt        time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt        time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type FlowTemplateIndex struct {
	ResourceStatus struct{} `index:"resource_type,status,sort,id"`
	PublishStatus  struct{} `index:"publish_status,current_release_id"`
	StatusSort     struct{} `index:"status,sort,id"`
}

var flowTemplateSeed = []map[string]any{}

func NewFlowTemplateModel() *orm.Model[FlowTemplate] {
	return orm.LoadModel[FlowTemplate]("流程模板", "crm_flow_template", orm.ModelConfig{
		Index:    FlowTemplateIndex{},
		Seeds:    flowTemplateSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"resource_type":  resourceTypeOptions,
			"publish_status": publishStatusOptions,
			"status":         statusOptions,
		},
	})
}
