package model

import "github.com/shemic/dever/orm"

const (
	StatusEnabled  int16 = 1
	StatusDisabled int16 = 2
)

const (
	TaskTypeForm       = "form"
	TaskTypeAssign     = "assign"
	TaskTypeDecision   = "decision"
	TaskTypeBooking    = "booking"
	TaskTypeSystemRule = "system_rule"
)

const (
	TaskFormModeCreate = "create"
	TaskFormModeEdit   = "edit"
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

const (
	ScriptUsageTaskRule   = "task_rule"
	ScriptUsageTransition = "transition"
	ScriptUsageFieldCalc  = "field_calc"
	ScriptUsageValidation = "validation"
)

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
	StaffTypeLeader   = "leader"
	StaffTypeEmployee = "employee"
)

var statusOptions = []map[string]any{
	{"id": StatusEnabled, "value": "启用"},
	{"id": StatusDisabled, "value": "停用"},
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

var scriptUsageOptions = []map[string]any{
	{"id": ScriptUsageTaskRule, "value": "任务规则"},
	{"id": ScriptUsageTransition, "value": "流转判断"},
	{"id": ScriptUsageFieldCalc, "value": "字段计算"},
	{"id": ScriptUsageValidation, "value": "数据校验"},
}

var taskTypeOptions = []map[string]any{
	{"id": TaskTypeForm, "value": "填写资料"},
	{"id": TaskTypeAssign, "value": "分配"},
	{"id": TaskTypeDecision, "value": "决策"},
	{"id": TaskTypeBooking, "value": "资源预定"},
}

var taskFormModeOptions = []map[string]any{
	{"id": TaskFormModeCreate, "value": "新增"},
	{"id": TaskFormModeEdit, "value": "编辑"},
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

var stageRelation = orm.Relation{
	Field:      "stage_id",
	Option:     "crm.NewStageModel",
	OptionKeys: []string{"code", "name"},
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
	Field:      "from_stage_code",
	Option:     "crm.NewStageModel",
	OptionKeys: []string{"code", "name"},
}

var toStageRelation = orm.Relation{
	Field:      "to_stage_code",
	Option:     "crm.NewStageModel",
	OptionKeys: []string{"code", "name"},
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
	OptionKeys: []string{"name", "usage"},
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
	OptionKeys: []string{"name", "usage"},
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
