package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type RuleScript struct {
	ID          uint64    `dorm:"primaryKey;autoIncrement;comment:脚本ID"`
	CateID      uint64    `dorm:"type:bigint;not null;default:0;comment:脚本分类"`
	Name        string    `dorm:"type:varchar(128);not null;comment:脚本名称"`
	Description string    `dorm:"type:text;not null;default:'';comment:描述"`
	Script      string    `dorm:"type:text;not null;default:'';comment:脚本"`
	Status      int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort        int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt   time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type RuleScriptIndex struct {
	CateStatus struct{} `index:"cate_id,status,sort,id"`
	NameStatus struct{} `index:"name,status,id"`
	StatusSort struct{} `index:"status,sort,id"`
}

var ruleScriptSeed = []map[string]any{}

func NewRuleScriptModel() *orm.Model[RuleScript] {
	return orm.LoadModel[RuleScript]("脚本规则", "crm_rule_script", orm.ModelConfig{
		Index:    RuleScriptIndex{},
		Seeds:    ruleScriptSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			ruleScriptCateRelation,
		},
	})
}
