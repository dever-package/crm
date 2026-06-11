package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type RuleScript struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:脚本ID"`
	CateID         uint64    `dorm:"type:bigint;not null;default:0;comment:脚本分类"`
	Name           string    `dorm:"type:varchar(128);not null;comment:脚本名称"`
	Description    string    `dorm:"type:text;not null;default:'';comment:描述"`
	Usage          string    `dorm:"type:varchar(32);not null;default:'task_rule';comment:脚本用途"`
	Language       string    `dorm:"type:varchar(32);not null;default:'javascript';comment:语言"`
	Script         string    `dorm:"type:text;not null;default:'';comment:脚本"`
	Entry          string    `dorm:"type:varchar(64);not null;default:'evaluate';comment:入口函数"`
	TimeoutMS      int       `dorm:"type:int;not null;default:100;comment:超时时间毫秒"`
	SampleInput    string    `dorm:"type:text;not null;default:'{}';comment:样例输入"`
	ExpectedOutput string    `dorm:"type:text;not null;default:'{}';comment:期望输出"`
	Status         int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort           int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type RuleScriptIndex struct {
	CateStatus  struct{} `index:"cate_id,status,sort,id"`
	UsageStatus struct{} `index:"usage,status,sort,id"`
	NameStatus  struct{} `index:"name,status,id"`
	StatusSort  struct{} `index:"status,sort,id"`
}

var ruleScriptSeed = []map[string]any{}

func NewRuleScriptModel() *orm.Model[RuleScript] {
	return orm.LoadModel[RuleScript]("脚本规则", "crm_rule_script", orm.ModelConfig{
		Index:    RuleScriptIndex{},
		Seeds:    ruleScriptSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"usage":  scriptUsageOptions,
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			ruleScriptCateRelation,
		},
	})
}
