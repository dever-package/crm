package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type FlowNode struct {
	ID                  uint64    `dorm:"primaryKey;autoIncrement;comment:流程节点ID"`
	FlowTemplateID      uint64    `dorm:"type:bigint;not null;comment:流程模板"`
	StageID             uint64    `dorm:"type:bigint;not null;comment:所属阶段"`
	NodeKey             string    `dorm:"type:varchar(64);not null;comment:节点标识"`
	Name                string    `dorm:"type:varchar(128);not null;comment:节点名称"`
	Description         string    `dorm:"type:text;not null;default:'';comment:描述"`
	NodeType            string    `dorm:"type:varchar(32);not null;default:'task';comment:节点类型"`
	TaskTemplateID      uint64    `dorm:"type:bigint;not null;default:0;comment:任务模板"`
	ScriptID            uint64    `dorm:"type:bigint;not null;default:0;comment:脚本规则"`
	ExecutorMode        string    `dorm:"type:varchar(32);not null;default:'department';comment:执行人模式"`
	DefaultDepartmentID uint64    `dorm:"type:bigint;not null;default:0;comment:默认部门"`
	DefaultRoleID       uint64    `dorm:"type:bigint;not null;default:0;comment:默认角色"`
	DefaultStaffID      uint64    `dorm:"type:bigint;not null;default:0;comment:默认人员"`
	EnableDeadline      bool      `dorm:"not null;default:false;comment:是否启用截止时间"`
	DeadlineMinutes     int       `dorm:"type:int;not null;default:0;comment:截止分钟数"`
	PositionJSON        string    `dorm:"type:text;not null;default:'{}';comment:画布位置JSON"`
	ConfigJSON          string    `dorm:"type:text;not null;default:'{}';comment:配置JSON"`
	Sort                int       `dorm:"type:int;not null;default:100;comment:排序"`
	Status              int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	CreatedAt           time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt           time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type FlowNodeIndex struct {
	TemplateKey    struct{} `unique:"flow_template_id,node_key"`
	TemplateStatus struct{} `index:"flow_template_id,status,sort,id"`
	StageStatus    struct{} `index:"stage_id,status,sort,id"`
	TypeStatus     struct{} `index:"node_type,status,id"`
	TemplateTask   struct{} `index:"task_template_id,status,id"`
	DefaultDept    struct{} `index:"default_department_id,status,id"`
	DefaultStaff   struct{} `index:"default_staff_id,status,id"`
}

var flowNodeSeed = []map[string]any{}

func NewFlowNodeModel() *orm.Model[FlowNode] {
	return orm.LoadModel[FlowNode]("流程节点", "crm_flow_node", orm.ModelConfig{
		Index:    FlowNodeIndex{},
		Seeds:    flowNodeSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"node_type":     flowNodeTypeOptions,
			"executor_mode": executorModeOptions,
			"status":        statusOptions,
		},
		Relations: []orm.Relation{
			flowTemplateRelation,
			flowStageRelation,
			taskTemplateRelation,
			ruleScriptRelation,
			defaultDepartmentRelation,
			defaultStaffRelation,
		},
	})
}
