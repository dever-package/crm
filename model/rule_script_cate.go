package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type RuleScriptCate struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:脚本分类ID"`
	Name      string    `dorm:"type:varchar(128);not null;comment:名称"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type RuleScriptCateIndex struct {
	StatusSort struct{} `index:"status,sort,id"`
}

var ruleScriptCateSeed = []map[string]any{}

func NewRuleScriptCateModel() *orm.Model[RuleScriptCate] {
	return orm.LoadModel[RuleScriptCate]("脚本分类", "crm_rule_script_cate", orm.ModelConfig{
		Index:    RuleScriptCateIndex{},
		Seeds:    ruleScriptCateSeed,
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
	})
}
