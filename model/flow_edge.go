package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type FlowEdge struct {
	ID                   uint64    `dorm:"primaryKey;autoIncrement;comment:流程连线ID"`
	FlowTemplateID       uint64    `dorm:"type:bigint;not null;comment:流程模板"`
	FromNodeID           uint64    `dorm:"type:bigint;not null;comment:来源节点"`
	FromNodeKey          string    `dorm:"type:varchar(64);not null;default:'';comment:来源节点标识"`
	ToNodeID             uint64    `dorm:"type:bigint;not null;default:0;comment:目标节点"`
	ToNodeKey            string    `dorm:"type:varchar(64);not null;default:'';comment:目标节点标识"`
	MatchResult          string    `dorm:"type:varchar(64);not null;default:'';comment:匹配结果"`
	MatchScriptID        uint64    `dorm:"type:bigint;not null;default:0;comment:匹配脚本"`
	TargetResourceStatus string    `dorm:"type:varchar(32);not null;default:'';comment:目标资源状态"`
	TargetDepartmentID   uint64    `dorm:"type:bigint;not null;default:0;comment:目标部门"`
	TargetRoleID         uint64    `dorm:"type:bigint;not null;default:0;comment:目标角色"`
	ConditionJSON        string    `dorm:"type:text;not null;default:'{}';comment:条件JSON"`
	Sort                 int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status               int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt            time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type FlowEdgeIndex struct {
	FlowStatus struct{} `index:"flow_template_id,status,sort,id"`
	FromResult struct{} `index:"from_node_id,match_result,status,id"`
	FromKey    struct{} `index:"flow_template_id,from_node_key,match_result,status,id"`
	ToNode     struct{} `index:"to_node_id,status,id"`
}

var flowEdgeSeed = []map[string]any{}

func NewFlowEdgeModel() *orm.Model[FlowEdge] {
	return orm.LoadModel[FlowEdge]("流程连线", "crm_flow_edge", orm.ModelConfig{
		Index:    FlowEdgeIndex{},
		Seeds:    flowEdgeSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"target_resource_status": resourceStatusOptions,
			"status":                 statusOptions,
		},
		Relations: []orm.Relation{
			flowTemplateRelation,
			fromFlowNodeRelation,
			toFlowNodeRelation,
			matchScriptRelation,
			targetDepartmentRelation,
		},
	})
}
