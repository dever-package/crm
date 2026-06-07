package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type FlowRelease struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:发布ID"`
	FlowTemplateID uint64    `dorm:"type:bigint;not null;comment:流程模板"`
	Version        int       `dorm:"type:int;not null;comment:版本号"`
	SnapshotJSON   string    `dorm:"type:text;not null;default:'{}';comment:发布快照JSON"`
	Status         int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type FlowReleaseIndex struct {
	TemplateVersion struct{} `unique:"flow_template_id,version"`
	TemplateStatus  struct{} `index:"flow_template_id,status,id"`
}

func NewFlowReleaseModel() *orm.Model[FlowRelease] {
	return orm.LoadModel[FlowRelease]("流程发布", "crm_flow_release", orm.ModelConfig{
		Index:    FlowReleaseIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{flowTemplateRelation},
	})
}
