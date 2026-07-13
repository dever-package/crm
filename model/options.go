package model

import (
	"fmt"
	"strings"

	"github.com/shemic/dever/orm"
)

const (
	StatusEnabled  int16 = 1
	StatusDisabled int16 = 2
)

const (
	TaskTypeTodo     = "todo"
	TaskTypeForm     = "form"
	TaskTypeApproval = "approval"
	TaskTypeRule     = "rule"
	TaskTypeProduct  = "product"
)

const (
	TaskAssigneeStage  = "stage"
	TaskAssigneeAuto   = "auto"
	TaskAssigneeManual = "manual"
)

const (
	StageAssignmentAuto   = "auto"
	StageAssignmentManual = "manual"
)

const (
	ProgressStatusActive     = "active"
	ProgressStatusCompleted  = "completed"
	ProgressStatusTerminated = "terminated"
)

const (
	LeadStatusPending   = "pending"
	LeadStatusInvalid   = "invalid"
	LeadStatusDuplicate = "duplicate"
	LeadStatusConverted = "converted"
)

const (
	WorkTodoStatusPending  = "pending"
	WorkTodoStatusDone     = "done"
	WorkTodoStatusCanceled = "canceled"
)

const (
	ResourceBookingStatusPending  = "pending"
	ResourceBookingStatusReserved = "reserved"
	ResourceBookingStatusCanceled = "canceled"
	ResourceBookingStatusRejected = "rejected"
	ResourceBookingStatusDone     = "done"
)

const (
	MemberRelationCreator     = "creator"
	MemberRelationAssignee    = "assignee"
	MemberRelationFollower    = "follower"
	MemberRelationParticipant = "participant"
	MemberRelationViewer      = "viewer"
)

const (
	StatEventTypeTask       = "task"
	StatEventTypeTransition = "transition"
)

const (
	StatValueSourceForm       = "form"
	StatValueSourceTransition = "transition"
	StatValueSourceTask       = "task"
)

const (
	DataUsageTypeStat    = "stat"
	DataUsageTypeFinance = "finance"
	DataUsageTypeDisplay = "display"
	DataUsageTypeReport  = "report"
)

const (
	DataUsageValueTypeText      = "text"
	DataUsageValueTypeNumber    = "number"
	DataUsageValueTypeAmount    = "amount"
	DataUsageValueTypeTime      = "time"
	DataUsageValueTypeStatus    = "status"
	DataUsageValueTypeDimension = "dimension"
)

const (
	DataUsageAggregateCount = "count"
	DataUsageAggregateSum   = "sum"
	DataUsageAggregateAvg   = "avg"
	DataUsageAggregateGroup = "group"
)

const (
	DataFieldStatTypeDimension = "dimension"
	DataFieldStatTypeMetric    = "metric"
	DataFieldStatTypeAmount    = "amount"
	DataFieldStatTypeFinance   = "finance"
	DataFieldStatTypeTime      = "time"
	DataFieldStatTypeStatus    = "status"
	DataFieldStatTypeText      = "text"
)

const (
	FinanceDirectionIncome  = "income"
	FinanceDirectionExpense = "expense"
)

const (
	ProductOptionSetName = "S产品"
)

const (
	FinanceLedgerSourceForm    = "form"
	FinanceLedgerSourceReverse = "reverse"
)

const (
	StaffTypeLeader   = "leader"
	StaffTypeEmployee = "employee"
)

var statusOptions = []map[string]any{
	{"id": StatusEnabled, "value": "启用"},
	{"id": StatusDisabled, "value": "停用"},
}

var leadStatusOptions = []map[string]any{
	{"id": LeadStatusPending, "value": "待处理"},
	{"id": LeadStatusInvalid, "value": "无效"},
	{"id": LeadStatusDuplicate, "value": "重复"},
	{"id": LeadStatusConverted, "value": "已转化"},
}

func LeadStatusName(status string) string {
	return crmOptionName(leadStatusOptions, status)
}

func crmOptionName(options []map[string]any, id string) string {
	target := strings.TrimSpace(id)
	if target == "" {
		return ""
	}
	for _, option := range options {
		if strings.TrimSpace(fmt.Sprint(option["id"])) == target {
			return strings.TrimSpace(fmt.Sprint(option["value"]))
		}
	}
	return target
}

var staffTypeOptions = []map[string]any{
	{"id": StaffTypeLeader, "value": "负责人"},
	{"id": StaffTypeEmployee, "value": "员工"},
}

var customerGenderOptions = []map[string]any{
	{"id": "male", "value": "男"},
	{"id": "female", "value": "女"},
	{"id": "unknown", "value": "未知"},
}

var dataFieldStatTypeOptions = []map[string]any{
	{"id": DataFieldStatTypeDimension, "value": "分类"},
	{"id": DataFieldStatTypeMetric, "value": "数值"},
	{"id": DataFieldStatTypeAmount, "value": "金额"},
	{"id": DataFieldStatTypeFinance, "value": "财务"},
	{"id": DataFieldStatTypeTime, "value": "时间"},
	{"id": DataFieldStatTypeStatus, "value": "状态"},
	{"id": DataFieldStatTypeText, "value": "文本"},
}

var dataUsageTypeOptions = []map[string]any{
	{"id": DataUsageTypeStat, "value": "统计"},
	{"id": DataUsageTypeFinance, "value": "财务"},
	{"id": DataUsageTypeDisplay, "value": "展示"},
	{"id": DataUsageTypeReport, "value": "报表"},
}

var dataUsageValueTypeOptions = []map[string]any{
	{"id": DataUsageValueTypeText, "value": "文本"},
	{"id": DataUsageValueTypeNumber, "value": "数字"},
	{"id": DataUsageValueTypeAmount, "value": "金额"},
	{"id": DataUsageValueTypeTime, "value": "时间"},
	{"id": DataUsageValueTypeStatus, "value": "状态"},
	{"id": DataUsageValueTypeDimension, "value": "维度"},
}

var dataUsageAggregateTypeOptions = []map[string]any{
	{"id": "", "value": "不聚合"},
	{"id": DataUsageAggregateCount, "value": "计数"},
	{"id": DataUsageAggregateSum, "value": "求和"},
	{"id": DataUsageAggregateAvg, "value": "平均"},
	{"id": DataUsageAggregateGroup, "value": "分组"},
}

var financeDirectionOptions = []map[string]any{
	{"id": FinanceDirectionIncome, "value": "收入"},
	{"id": FinanceDirectionExpense, "value": "支出"},
}

var financeLedgerSourceOptions = []map[string]any{
	{"id": FinanceLedgerSourceForm, "value": "表单"},
	{"id": FinanceLedgerSourceReverse, "value": "冲正"},
}

var statEventTypeOptions = []map[string]any{
	{"id": StatEventTypeTask, "value": "任务"},
	{"id": StatEventTypeTransition, "value": "流转"},
}

var taskTypeOptions = []map[string]any{
	{"id": TaskTypeTodo, "value": "普通事项"},
	{"id": TaskTypeForm, "value": "填写资料"},
	{"id": TaskTypeApproval, "value": "审核"},
	{"id": TaskTypeRule, "value": "自动核验"},
	{"id": TaskTypeProduct, "value": "确认产品"},
}

var taskAssigneeModeOptions = []map[string]any{
	{"id": TaskAssigneeStage, "value": "跟随阶段负责人"},
	{"id": TaskAssigneeAuto, "value": "自动分配到部门"},
	{"id": TaskAssigneeManual, "value": "由当前负责人手动分配"},
}

var stageAssignmentModeOptions = []map[string]any{
	{"id": StageAssignmentAuto, "value": "自动分配"},
	{"id": StageAssignmentManual, "value": "手动分配"},
}

var progressStatusOptions = []map[string]any{
	{"id": ProgressStatusActive, "value": "进行中"},
	{"id": ProgressStatusCompleted, "value": "已完成"},
	{"id": ProgressStatusTerminated, "value": "已终止"},
}

var workTodoStatusOptions = []map[string]any{
	{"id": WorkTodoStatusPending, "value": "待处理"},
	{"id": WorkTodoStatusDone, "value": "已完成"},
	{"id": WorkTodoStatusCanceled, "value": "已取消"},
}

var resourceBookingStatusOptions = []map[string]any{
	{"id": ResourceBookingStatusPending, "value": "待确认"},
	{"id": ResourceBookingStatusReserved, "value": "已预定"},
	{"id": ResourceBookingStatusCanceled, "value": "已取消"},
	{"id": ResourceBookingStatusRejected, "value": "已拒绝"},
	{"id": ResourceBookingStatusDone, "value": "已完成"},
}

var memberRelationOptions = []map[string]any{
	{"id": MemberRelationCreator, "value": "创建人"},
	{"id": MemberRelationAssignee, "value": "负责人"},
	{"id": MemberRelationFollower, "value": "跟进人"},
	{"id": MemberRelationParticipant, "value": "参与人"},
	{"id": MemberRelationViewer, "value": "查看人"},
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
	{"id": "boolean", "value": "开关"},
	{"id": "attachment", "value": "附件"},
	{"id": "group", "value": "分组"},
}

var customerRelation = orm.Relation{
	Field:      "customer_id",
	Option:     "crm.NewCustomerModel",
	OptionKeys: []string{"name", "phone"},
}

var assetRelation = orm.Relation{
	Field:      "asset_id",
	Option:     "crm.NewCustomerAssetModel",
	OptionKeys: []string{"asset_no", "asset_name", "asset_status_id"},
}

var productRelation = orm.Relation{
	Field:      "product_id",
	Option:     "crm.NewProductModel",
	OptionKeys: []string{"name", "code", "category_id", "service_workflow_id"},
}

var customerProductRelation = orm.Relation{
	Field:      "customer_product_id",
	Option:     "crm.NewCustomerProductModel",
	OptionKeys: []string{"customer_id", "asset_id", "product_id", "status"},
}

var workflowInstanceRelation = orm.Relation{
	Field:      "workflow_instance_id",
	Option:     "crm.NewWorkflowInstanceModel",
	OptionKeys: []string{"customer_id", "asset_id", "customer_product_id", "workflow_id", "stage_id", "status"},
}

var sourceWorkflowInstanceRelation = orm.Relation{
	Field:      "source_workflow_instance_id",
	Option:     "crm.NewWorkflowInstanceModel",
	OptionKeys: []string{"customer_id", "asset_id", "workflow_id", "status"},
}

var workflowRelation = orm.Relation{
	Field:      "workflow_id",
	Option:     "crm.NewWorkflowModel",
	OptionKeys: []string{"name"},
}

var serviceWorkflowRelation = orm.Relation{
	Field:      "service_workflow_id",
	Option:     "crm.NewWorkflowModel",
	OptionKeys: []string{"name"},
}

var productCategoryRelation = orm.Relation{
	Field:      "category_id",
	Option:     "crm.NewProductCategoryModel",
	OptionKeys: []string{"name"},
}

var assetStatusRelation = orm.Relation{
	Field:      "asset_status_id",
	Option:     "crm.NewAssetStatusModel",
	OptionKeys: []string{"name", "code"},
}

var customerSourceRelation = orm.Relation{
	Field:      "source_id",
	Option:     "crm.NewCustomerSourceModel",
	OptionKeys: []string{"name", "code"},
}

var customerChannelRelation = orm.Relation{
	Field:      "channel_id",
	Option:     "crm.NewCustomerChannelModel",
	OptionKeys: []string{"name", "code"},
}

var customerLevelRelation = orm.Relation{
	Field:      "level_id",
	Option:     "crm.NewCustomerLevelModel",
	OptionKeys: []string{"name", "code"},
}

var taskRelation = orm.Relation{
	Field:      "task_id",
	Option:     "crm.NewTaskModel",
	OptionKeys: []string{"name", "task_type"},
}

var operationLogRelation = orm.Relation{
	Field:      "operation_log_id",
	Option:     "crm.NewOperationLogModel",
	OptionKeys: []string{"title", "result_value", "created_at"},
}

var financeTypeRelation = orm.Relation{
	Field:      "finance_type_id",
	Option:     "crm.NewFinanceTypeModel",
	OptionKeys: []string{"name", "code", "direction"},
}

var stageRelation = orm.Relation{
	Field:      "stage_id",
	Option:     "crm.NewStageModel",
	OptionKeys: []string{"name", "workflow_id"},
}

var formRelation = orm.Relation{
	Field:      "form_id",
	Option:     "crm.NewFormModel",
	OptionKeys: []string{"name"},
}

var formFieldRelation = orm.Relation{
	Field:      "form_field_id",
	Option:     "crm.NewFormFieldModel",
	OptionKeys: []string{"name"},
}

var fromStageRelation = orm.Relation{
	Field:      "from_stage_id",
	Option:     "crm.NewStageModel",
	OptionKeys: []string{"name", "workflow_id"},
}

var toStageRelation = orm.Relation{
	Field:      "to_stage_id",
	Option:     "crm.NewStageModel",
	OptionKeys: []string{"name", "workflow_id"},
}

var dataTemplateRelation = orm.Relation{
	Field:      "data_template_id",
	Option:     "crm.NewDataTemplateModel",
	OptionKeys: []string{"name"},
}

var dataFieldRelation = orm.Relation{
	Field:      "data_field_id",
	Option:     "crm.NewDataFieldModel",
	OptionKeys: []string{"name", "field_key", "field_type"},
}

var dataTemplateCateRelation = orm.Relation{
	Field:      "cate_id",
	Option:     "crm.NewDataTemplateCateModel",
	OptionKeys: []string{"name"},
}

var formFieldDataTemplateCateRelation = orm.Relation{
	Field:      "data_template_cate_id",
	Option:     "crm.NewDataTemplateCateModel",
	OptionKeys: []string{"name"},
}

var formFieldDataTemplateRelation = orm.Relation{
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
	OptionKeys: []string{"name"},
}

var resourceCateRelation = orm.Relation{
	Field:      "resource_cate_id",
	Option:     "crm.NewPublicResourceCateModel",
	OptionKeys: []string{"name"},
}

var resourceRelation = orm.Relation{
	Field:      "resource_id",
	Option:     "crm.NewPublicResourceModel",
	OptionKeys: []string{"name", "location"},
}

var matchScriptRelation = orm.Relation{
	Field:      "match_script_id",
	Option:     "crm.NewRuleScriptModel",
	OptionKeys: []string{"name"},
}

var formFieldDataFieldRelation = orm.Relation{
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

var assigneeDepartmentRelation = orm.Relation{
	Field:      "assignee_department_id",
	Option:     "crm.NewDepartmentModel",
	OptionKeys: []string{"name", "code"},
}

var operatorDepartmentRelation = orm.Relation{
	Field:      "operator_department_id",
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

var assigneeStaffRelation = orm.Relation{
	Field:      "assignee_staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone", "department_id"},
}

var bookerStaffRelation = orm.Relation{
	Field:      "booker_staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}

var bookerDepartmentRelation = orm.Relation{
	Field:      "booker_department_id",
	Option:     "crm.NewDepartmentModel",
	OptionKeys: []string{"name", "code"},
}

var approvedByStaffRelation = orm.Relation{
	Field:      "approved_by_staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}

var toStaffRelation = orm.Relation{
	Field:      "to_staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}

var operatorStaffRelation = orm.Relation{
	Field:      "operator_staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}

var createdByStaffRelation = orm.Relation{
	Field:      "created_by_staff_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}

var uploaderRelation = orm.Relation{
	Field:      "uploader_id",
	Option:     "crm.NewStaffModel",
	OptionKeys: []string{"name", "phone"},
}
