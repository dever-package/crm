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
	TaskTypeApproval = "approval"
	TaskTypeRule     = "rule"

	// Legacy task types remain until the old runtime is removed later in this refactor.
	TaskTypeCreate      = "create"
	TaskTypeForm        = "form"
	TaskTypeAssign      = "assign"
	TaskTypeCollaborate = "collaborate"
	TaskTypeDecision    = "decision"
	TaskTypeBooking     = "booking"
	TaskTypeSystemRule  = "system_rule"
)

const (
	TaskAssigneeStage      = "stage"
	TaskAssigneeDepartment = "department"
	TaskAssigneeStaff      = "staff"
)

const (
	ProgressStatusActive    = "active"
	ProgressStatusCompleted = "completed"
)

const (
	TaskAssignModeStaff      = "staff"
	TaskAssignModeDepartment = "department"
)

const (
	TaskCompleteAssignTaskID = "complete_assign_task_id"
)

const (
	TaskCompletionSubmit = "submit"
	TaskCompletionManual = "manual"
)

const (
	WorkTodoStatusPending  = "pending"
	WorkTodoStatusDone     = "done"
	WorkTodoStatusCanceled = "canceled"
)

const (
	CollaborationCompleteAll    = "all"
	CollaborationCompleteAny    = "any"
	CollaborationCompleteManual = "manual"
)

const (
	ResourceBookingStatusPending  = "pending"
	ResourceBookingStatusReserved = "reserved"
	ResourceBookingStatusCanceled = "canceled"
	ResourceBookingStatusRejected = "rejected"
	ResourceBookingStatusDone     = "done"
)

const (
	TaskTriggerManual     = "manual"
	TaskTriggerAfterTask  = "after_task"
	TaskTriggerStageEnter = "on_stage_enter"
)

func TaskTypeSupportsAutoTrigger(taskType string) bool {
	switch taskType {
	case TaskTypeAssign, TaskTypeCollaborate, TaskTypeDecision:
		return true
	default:
		return false
	}
}

const (
	StageOwnerKeep            = "keep"
	StageOwnerAssign          = "assign"
	StageOwnerFixedDepartment = "fixed_department"
	StageOwnerFixedStaff      = "fixed_staff"
	StageOwnerCreator         = "creator"
)

const (
	MemberRelationCreator     = "creator"
	MemberRelationAssignee    = "assignee"
	MemberRelationFollower    = "follower"
	MemberRelationParticipant = "participant"
	MemberRelationViewer      = "viewer"
)

const (
	BusinessObjectParentCustomer       = "customer"
	BusinessObjectParentCustomerAsset  = "customer_asset"
	BusinessObjectParentBusinessObject = "business_object"
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

	ProductCategoryJudicial       = "judicial"
	ProductCategoryAssetOperation = "asset_operation"
	ProductCategoryDebtStructure  = "debt_structure"
	ProductCategoryStageService   = "stage_service"
	ProductCategoryRiskDisposal   = "risk_disposal"
	ProductCategoryConsulting     = "consulting"

	ProductSigningNonSealed = "non_sealed_asset_signing"
	ProductSigningSealed    = "sealed_asset_service_signing"
	ProductSigningManual    = "manual_review"
)

const (
	FinanceLedgerSourceForm    = "form"
	FinanceLedgerSourceReverse = "reverse"
)

const (
	TaskPointLedgerSourceTaskComplete = "task_complete"
)

const (
	StaffTypeLeader   = "leader"
	StaffTypeEmployee = "employee"
)

var statusOptions = []map[string]any{
	{"id": StatusEnabled, "value": "启用"},
	{"id": StatusDisabled, "value": "停用"},
}

var businessObjectStatusOptions = []map[string]any{
	{"id": "active", "value": "进行中"},
	{"id": "pending", "value": "待出租"},
	{"id": "rented", "value": "已出租"},
	{"id": "delivering", "value": "交付中"},
	{"id": "ended", "value": "已退租"},
	{"id": "abnormal", "value": "异常"},
	{"id": "closed", "value": "已关闭"},
}

func BusinessObjectStatusName(status string) string {
	return crmOptionName(businessObjectStatusOptions, status)
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

var productCategoryOptions = []map[string]any{
	{"id": ProductCategoryJudicial, "value": "司法推进类"},
	{"id": ProductCategoryAssetOperation, "value": "资产运营类"},
	{"id": ProductCategoryDebtStructure, "value": "债务结构类"},
	{"id": ProductCategoryStageService, "value": "阶段性服务类"},
	{"id": ProductCategoryRiskDisposal, "value": "风险处置类"},
	{"id": ProductCategoryConsulting, "value": "咨询/预审类"},
}

var productSigningTypeOptions = []map[string]any{
	{"id": ProductSigningNonSealed, "value": "非查封资产签约"},
	{"id": ProductSigningSealed, "value": "查封服务签约"},
	{"id": ProductSigningManual, "value": "人工复核"},
}

var financeLedgerSourceOptions = []map[string]any{
	{"id": FinanceLedgerSourceForm, "value": "表单"},
	{"id": FinanceLedgerSourceReverse, "value": "冲正"},
}

var taskPointLedgerSourceOptions = []map[string]any{
	{"id": TaskPointLedgerSourceTaskComplete, "value": "任务完成"},
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
}

var taskAssigneeModeOptions = []map[string]any{
	{"id": TaskAssigneeStage, "value": "跟随阶段负责部门"},
	{"id": TaskAssigneeDepartment, "value": "指定部门"},
	{"id": TaskAssigneeStaff, "value": "指定人员"},
}

var progressStatusOptions = []map[string]any{
	{"id": ProgressStatusActive, "value": "进行中"},
	{"id": ProgressStatusCompleted, "value": "已完成"},
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

var taskTriggerOptions = []map[string]any{
	{"id": TaskTriggerManual, "value": "手动触发"},
	{"id": TaskTriggerAfterTask, "value": "任务后触发"},
	{"id": TaskTriggerStageEnter, "value": "进入阶段触发"},
}

var stageOwnerModeOptions = []map[string]any{
	{"id": StageOwnerKeep, "value": "保持当前"},
	{"id": StageOwnerAssign, "value": "使用分配结果"},
	{"id": StageOwnerFixedDepartment, "value": "固定部门"},
	{"id": StageOwnerFixedStaff, "value": "固定人员"},
	{"id": StageOwnerCreator, "value": "创建人"},
}

var memberRelationOptions = []map[string]any{
	{"id": MemberRelationCreator, "value": "创建人"},
	{"id": MemberRelationAssignee, "value": "负责人"},
	{"id": MemberRelationFollower, "value": "跟进人"},
	{"id": MemberRelationParticipant, "value": "参与人"},
	{"id": MemberRelationViewer, "value": "查看人"},
}

var businessObjectParentTargetOptions = []map[string]any{
	{"id": BusinessObjectParentCustomer, "value": "客户"},
	{"id": BusinessObjectParentCustomerAsset, "value": "客户资产"},
	{"id": BusinessObjectParentBusinessObject, "value": "业务对象"},
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

var workflowRelation = orm.Relation{
	Field:      "workflow_id",
	Option:     "crm.NewWorkflowModel",
	OptionKeys: []string{"name"},
}

var businessObjectTypeRelation = orm.Relation{
	Field:      "business_object_type_id",
	Option:     "crm.NewBusinessObjectTypeModel",
	OptionKeys: []string{"name", "code", "parent_target"},
}

var businessObjectRelation = orm.Relation{
	Field:      "business_object_id",
	Option:     "crm.NewBusinessObjectModel",
	OptionKeys: []string{"object_no", "object_name", "object_status", "business_object_type_id"},
}

var parentBusinessObjectRelation = orm.Relation{
	Field:      "parent_object_id",
	Option:     "crm.NewBusinessObjectModel",
	OptionKeys: []string{"object_no", "object_name", "object_status", "business_object_type_id"},
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

var todoRelation = orm.Relation{
	Field:      "todo_id",
	Option:     "crm.NewWorkTodoModel",
	OptionKeys: []string{"status", "result", "completed_at"},
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

var triggerTaskRelation = orm.Relation{
	Field:      "trigger_task_id",
	Option:     "crm.NewTaskModel",
	OptionKeys: []string{"name", "task_type"},
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

var currentStageRelation = orm.Relation{
	Field:      "current_stage_code",
	Option:     "crm.NewStageModel",
	OptionKeys: []string{"code", "name"},
}

var stageCodeRelation = orm.Relation{
	Field:      "stage_code",
	Option:     "crm.NewStageModel",
	OptionKeys: []string{"code", "name"},
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
	OptionKeys: []string{"name", "target_table"},
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

var currentDepartmentRelation = orm.Relation{
	Field:      "current_department_id",
	Option:     "crm.NewDepartmentModel",
	OptionKeys: []string{"name", "code"},
}

var toDepartmentRelation = orm.Relation{
	Field:      "to_department_id",
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

var currentStaffRelation = orm.Relation{
	Field:      "current_staff_id",
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
