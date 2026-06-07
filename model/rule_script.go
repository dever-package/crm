package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type RuleScript struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:脚本ID"`
	CateID         uint64    `dorm:"type:bigint;not null;default:1;comment:脚本分类"`
	Name           string    `dorm:"type:varchar(128);not null;comment:脚本名称"`
	Description    string    `dorm:"type:text;not null;default:'';comment:描述"`
	Usage          string    `dorm:"type:varchar(32);not null;default:'task_eval';comment:脚本用途"`
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

const defaultAdmissionScript = `function evaluate(input, config) {
  var missingRequired = Number(input.missing_required || input.missingRequired || 0);
  var riskLevel = String(input.risk_level || input.riskLevel || "medium");
  var blocking = input.blocking === true || input.blocking === "true" || input.blocking === 1;
  if (blocking) {
    return { decision: "reject", block_reason: "命中阻断项" };
  }
  if (missingRequired > 0) {
    return { decision: "need_more_data", block_reason: "关键资料未齐" };
  }
  if (riskLevel === "high" || riskLevel === "critical") {
    return { decision: "need_pm_review", risk_level: riskLevel };
  }
  return { decision: "pass", risk_level: riskLevel };
}`

var ruleScriptSeed = []map[string]any{
	{
		"id":              1,
		"cate_id":         DefaultRuleScriptCateID,
		"name":            "默认脚本判定",
		"description":     "默认样例脚本，只作为初始配置，可在后台替换。",
		"usage":           ScriptUsageTaskEval,
		"language":        "javascript",
		"script":          defaultAdmissionScript,
		"entry":           "evaluate",
		"timeout_ms":      100,
		"sample_input":    `{"risk_level":"medium","missing_required":0,"blocking":false}`,
		"expected_output": `{"decision":"pass","risk_level":"medium"}`,
		"status":          1,
		"sort":            10,
	},
}

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
