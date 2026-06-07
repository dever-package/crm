package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type FlowStage struct {
	ID                  uint64    `dorm:"primaryKey;autoIncrement;comment:阶段ID"`
	FlowTemplateID      uint64    `dorm:"type:bigint;not null;comment:流程模板"`
	StageKey            string    `dorm:"type:varchar(64);not null;comment:阶段标识"`
	Name                string    `dorm:"type:varchar(128);not null;comment:阶段名称"`
	Description         string    `dorm:"type:text;not null;default:'';comment:描述"`
	DefaultDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:默认部门"`
	PositionJSON        string    `dorm:"type:text;not null;default:'{}';comment:画布位置JSON"`
	Sort                int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status              int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt           time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt           time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type FlowStageIndex struct {
	TemplateKey    struct{} `unique:"flow_template_id,stage_key"`
	TemplateStatus struct{} `index:"flow_template_id,status,sort,id"`
	Department     struct{} `index:"default_department_id,status,id"`
}

var flowStageSeed = []map[string]any{}

func NewFlowStageModel() *orm.Model[FlowStage] {
	return orm.LoadModel[FlowStage]("流程阶段", "crm_flow_stage", orm.ModelConfig{
		Index:    FlowStageIndex{},
		Seeds:    flowStageSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			flowTemplateRelation,
			defaultDepartmentRelation,
		},
	})
}
