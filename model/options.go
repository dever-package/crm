package model

import "github.com/shemic/dever/orm"

const (
	StatusEnabled  int16 = 1
	StatusDisabled int16 = 2
)

const (
	PublishStatusDraft     = "draft"
	PublishStatusPublished = "published"
	PublishStatusEditing   = "editing"
)

const (
	DefaultFlowTemplateID     uint64 = 1
	ResourceTypeAssetDisposal        = "asset_disposal"
)

const (
	ResourceStatusNew        = "new"
	ResourceStatusProcessing = "processing"
	ResourceStatusWaiting    = "waiting"
	ResourceStatusBlocked    = "blocked"
	ResourceStatusCompleted  = "completed"
	ResourceStatusClosed     = "closed"
	ResourceStatusInvalid    = "invalid"
)

const (
	TaskStatusPending       = "pending"
	TaskStatusProcessing    = "processing"
	TaskStatusWaitingReview = "waiting_review"
	TaskStatusCompleted     = "completed"
	TaskStatusRejected      = "rejected"
	TaskStatusCancelled     = "cancelled"
	TaskStatusBlocked       = "blocked"
)

const (
	TaskTypeManual     = "manual"
	TaskTypeInput      = "manual_input"
	TaskTypeConfirm    = "manual_confirm"
	TaskTypeWork       = "task"
	TaskTypeScriptEval = "script_eval"
	TaskTypeArchive    = "archive"
)

const (
	NodeTypeInput      = "manual_input"
	NodeTypeConfirm    = "manual_confirm"
	NodeTypeTask       = "task"
	NodeTypeScriptEval = "script_eval"
	NodeTypeBranch     = "branch"
	NodeTypeArchive    = "archive"
)

const (
	ScriptUsageTaskEval   = "task_eval"
	ScriptUsageTransition = "transition"
	ScriptUsageFieldCalc  = "field_calc"
	ScriptUsageValidation = "validation"
	ScriptUsageMetric     = "metric"
)

var statusOptions = []map[string]any{
	{"id": StatusEnabled, "value": "启用"},
	{"id": StatusDisabled, "value": "停用"},
}

var customerGenderOptions = []map[string]any{
	{"id": "male", "value": "男"},
	{"id": "female", "value": "女"},
	{"id": "unknown", "value": "未知"},
}

var resourceTypeOptions = []map[string]any{
	{"id": "asset_disposal", "value": "资产处置"},
	{"id": "debt_restructuring", "value": "债务重组"},
	{"id": "service", "value": "服务事项"},
	{"id": "cooperation", "value": "合作事项"},
	{"id": "other", "value": "其他"},
}

var resourceStatusOptions = []map[string]any{
	{"id": ResourceStatusNew, "value": "新建"},
	{"id": ResourceStatusProcessing, "value": "处理中"},
	{"id": ResourceStatusWaiting, "value": "等待中"},
	{"id": ResourceStatusBlocked, "value": "已卡死"},
	{"id": ResourceStatusCompleted, "value": "已完成"},
	{"id": ResourceStatusClosed, "value": "已关闭"},
	{"id": ResourceStatusInvalid, "value": "无效"},
}

var riskLevelOptions = []map[string]any{
	{"id": "low", "value": "低"},
	{"id": "medium", "value": "中"},
	{"id": "high", "value": "高"},
	{"id": "critical", "value": "严重"},
}

var publishStatusOptions = []map[string]any{
	{"id": PublishStatusDraft, "value": "草稿"},
	{"id": PublishStatusPublished, "value": "已发布"},
	{"id": PublishStatusEditing, "value": "编辑草稿"},
}

var taskTypeOptions = []map[string]any{
	{"id": TaskTypeManual, "value": "人工任务"},
	{"id": TaskTypeInput, "value": "人工录入"},
	{"id": TaskTypeConfirm, "value": "人工确认"},
	{"id": TaskTypeWork, "value": "任务"},
	{"id": TaskTypeScriptEval, "value": "脚本判定"},
	{"id": TaskTypeArchive, "value": "归档"},
}

var flowNodeTypeOptions = []map[string]any{
	{"id": NodeTypeInput, "value": "人工录入"},
	{"id": NodeTypeConfirm, "value": "人工确认"},
	{"id": NodeTypeTask, "value": "任务"},
	{"id": NodeTypeScriptEval, "value": "脚本判定"},
	{"id": NodeTypeBranch, "value": "条件分支"},
	{"id": NodeTypeArchive, "value": "归档"},
}

var scriptUsageOptions = []map[string]any{
	{"id": ScriptUsageTaskEval, "value": "任务判定"},
	{"id": ScriptUsageTransition, "value": "流转判断"},
	{"id": ScriptUsageFieldCalc, "value": "字段计算"},
	{"id": ScriptUsageValidation, "value": "数据校验"},
	{"id": ScriptUsageMetric, "value": "指标计算"},
}

var executorModeOptions = []map[string]any{
	{"id": "manual", "value": "手动选择"},
	{"id": "department", "value": "按部门"},
	{"id": "staff", "value": "指定人员"},
	{"id": "resource_owner", "value": "资源负责人"},
}

var taskStatusOptions = []map[string]any{
	{"id": TaskStatusPending, "value": "待处理"},
	{"id": TaskStatusProcessing, "value": "处理中"},
	{"id": TaskStatusWaitingReview, "value": "待复核"},
	{"id": TaskStatusCompleted, "value": "已完成"},
	{"id": TaskStatusRejected, "value": "已驳回"},
	{"id": TaskStatusCancelled, "value": "已取消"},
	{"id": TaskStatusBlocked, "value": "已卡死"},
}

var fieldTypeOptions = []map[string]any{
	{"id": "text", "value": "单行文本"},
	{"id": "textarea", "value": "多行文本"},
	{"id": "number", "value": "数字"},
	{"id": "money", "value": "金额"},
	{"id": "date", "value": "日期"},
	{"id": "datetime", "value": "时间"},
	{"id": "radio", "value": "单选"},
	{"id": "checkbox", "value": "多选"},
	{"id": "select", "value": "下拉"},
	{"id": "multi_select", "value": "多选下拉"},
	{"id": "boolean", "value": "是/否"},
	{"id": "attachment", "value": "附件"},
}

var recordModeOptions = []map[string]any{
	{"id": "single", "value": "单条"},
	{"id": "multiple", "value": "多条"},
}

var metricTypeOptions = []map[string]any{
	{"id": "count", "value": "计数"},
	{"id": "sum", "value": "求和"},
	{"id": "avg", "value": "平均"},
	{"id": "group", "value": "分组"},
}

var departmentTypeOptions = []map[string]any{
	{"id": "mkt", "value": "MKT"},
	{"id": "npl", "value": "NPL"},
	{"id": "pm", "value": "PM"},
	{"id": "ala", "value": "ALA"},
	{"id": "finance", "value": "财务"},
	{"id": "management", "value": "管理层"},
	{"id": "other", "value": "其他"},
}

var businessRoleOptions = []map[string]any{
	{"id": "mkt", "value": "MKT"},
	{"id": "npl", "value": "NPL"},
	{"id": "pm", "value": "PM"},
	{"id": "ala", "value": "ALA"},
	{"id": "finance", "value": "财务"},
	{"id": "supervisor", "value": "主管"},
	{"id": "manager", "value": "管理层"},
}

var customerRelation = orm.Relation{
	Field:      "customer_id",
	Option:     "crm.NewCustomerModel",
	OptionKeys: []string{"name", "phone"},
}

var resourceRelation = orm.Relation{
	Field:      "resource_id",
	Option:     "crm.NewCustomerResourceModel",
	OptionKeys: []string{"resource_no", "asset_name", "asset_status"},
}

var assetCateRelation = orm.Relation{
	Field:      "asset_cate_id",
	Option:     "crm.NewAssetCateModel",
	OptionKeys: []string{"name"},
}

var flowTemplateRelation = orm.Relation{
	Field:      "flow_template_id",
	Option:     "crm.NewFlowTemplateModel",
	OptionKeys: []string{"name", "publish_status"},
}

var flowReleaseRelation = orm.Relation{
	Field:      "flow_release_id",
	Option:     "crm.NewFlowReleaseModel",
	OptionKeys: []string{"version", "status"},
}

var flowStageRelation = orm.Relation{
	Field:      "stage_id",
	Option:     "crm.NewFlowStageModel",
	OptionKeys: []string{"stage_key", "name"},
}

var taskTemplateRelation = orm.Relation{
	Field:      "task_template_id",
	Option:     "crm.NewTaskTemplateModel",
	OptionKeys: []string{"name"},
}

var taskTemplateCateRelation = orm.Relation{
	Field:      "cate_id",
	Option:     "crm.NewTaskTemplateCateModel",
	OptionKeys: []string{"name"},
}

var flowNodeRelation = orm.Relation{
	Field:      "flow_node_id",
	Option:     "crm.NewFlowNodeModel",
	OptionKeys: []string{"node_key", "name"},
}

var taskRelation = orm.Relation{
	Field:      "task_id",
	Option:     "crm.NewResourceTaskModel",
	OptionKeys: []string{"task_name", "status"},
}

var dataTemplateRelation = orm.Relation{
	Field:      "data_template_id",
	Option:     "crm.NewDataTemplateModel",
	OptionKeys: []string{"name"},
}

var dataTemplateCateRelation = orm.Relation{
	Field:      "cate_id",
	Option:     "crm.NewDataTemplateCateModel",
	OptionKeys: []string{"name"},
}

var taskFieldDataTemplateCateRelation = orm.Relation{
	Field:      "data_template_cate_id",
	Option:     "crm.NewDataTemplateCateModel",
	OptionKeys: []string{"name", "target_table"},
}

var taskFieldDataTemplateRelation = orm.Relation{
	Field:      "data_template_id",
	Option:     "crm.NewDataTemplateModel",
	OptionKeys: []string{"name", "cate_id"},
}

var ruleScriptCateRelation = orm.Relation{
	Field:      "cate_id",
	Option:     "crm.NewRuleScriptCateModel",
	OptionKeys: []string{"name"},
}

var ruleScriptRelation = orm.Relation{
	Field:      "script_id",
	Option:     "crm.NewRuleScriptModel",
	OptionKeys: []string{"name", "usage"},
}

var matchScriptRelation = orm.Relation{
	Field:      "match_script_id",
	Option:     "crm.NewRuleScriptModel",
	OptionKeys: []string{"name", "usage"},
}

var taskFieldDataFieldRelation = orm.Relation{
	Field:      "data_field_id",
	Option:     "crm.NewDataFieldModel",
	OptionKeys: []string{"name", "field_type"},
}

var dataRecordRelation = orm.Relation{
	Field:      "data_record_id",
	Option:     "crm.NewDataRecordModel",
	OptionKeys: []string{"summary", "status"},
}

var departmentRelation = orm.Relation{
	Field:      "department_id",
	Option:     "crm.NewDepartmentModel",
	OptionKeys: []string{"name", "code"},
}

var ownerDepartmentRelation = orm.Relation{
	Field:      "owner_department_id",
	Option:     "crm.NewDepartmentModel",
	OptionKeys: []string{"name", "code"},
}

var defaultDepartmentRelation = orm.Relation{
	Field:      "default_department_id",
	Option:     "crm.NewDepartmentModel",
	OptionKeys: []string{"name", "code"},
}

var targetDepartmentRelation = orm.Relation{
	Field:      "target_department_id",
	Option:     "crm.NewDepartmentModel",
	OptionKeys: []string{"name", "code"},
}

var assigneeDepartmentRelation = orm.Relation{
	Field:      "assignee_department_id",
	Option:     "crm.NewDepartmentModel",
	OptionKeys: []string{"name", "code"},
}

var leaderStaffRelation = orm.Relation{
	Field:      "leader_staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}

var staffRelation = orm.Relation{
	Field:      "staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}

var ownerStaffRelation = orm.Relation{
	Field:      "owner_staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}

var defaultStaffRelation = orm.Relation{
	Field:      "default_staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}

var assigneeStaffRelation = orm.Relation{
	Field:      "assignee_staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}

var accountRelation = orm.Relation{
	Field:      "account_id",
	Option:     "front.NewAccountModel",
	OptionKeys: []string{"name", "account"},
}

var operatorRelation = orm.Relation{
	Field:      "operator_id",
	Option:     "front.NewAccountModel",
	OptionKeys: []string{"name", "account"},
}

var creatorRelation = orm.Relation{
	Field:      "creator_id",
	Option:     "front.NewAccountModel",
	OptionKeys: []string{"name", "account"},
}

var uploaderRelation = orm.Relation{
	Field:      "uploader_id",
	Option:     "front.NewAccountModel",
	OptionKeys: []string{"name", "account"},
}

var fromTaskTemplateRelation = orm.Relation{
	Field:      "from_task_template_id",
	Option:     "crm.NewTaskTemplateModel",
	OptionKeys: []string{"name"},
}

var toTaskTemplateRelation = orm.Relation{
	Field:      "to_task_template_id",
	Option:     "crm.NewTaskTemplateModel",
	OptionKeys: []string{"name"},
}

var fromFlowNodeRelation = orm.Relation{
	Field:      "from_node_id",
	Option:     "crm.NewFlowNodeModel",
	OptionKeys: []string{"node_key", "name"},
}

var toFlowNodeRelation = orm.Relation{
	Field:      "to_node_id",
	Option:     "crm.NewFlowNodeModel",
	OptionKeys: []string{"node_key", "name"},
}

var toStageRelation = orm.Relation{
	Field:      "to_stage_id",
	Option:     "crm.NewFlowStageModel",
	OptionKeys: []string{"stage_key", "name"},
}

var dashboardRelation = orm.Relation{
	Field:      "dashboard_id",
	Option:     "crm.NewDashboardModel",
	OptionKeys: []string{"name", "resource_type"},
}

var dashboardFunnelRelation = orm.Relation{
	Field:      "funnel_id",
	Option:     "crm.NewDashboardFunnelModel",
	OptionKeys: []string{"name", "resource_type"},
}
