# Meeting Booking Form Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将预约会议室任务的预约时间、会议时长、会议室和到访次数集中到单一会议预约页签。

**Architecture:** 复用后端现有表单字段分组元数据，在会议任务组装表单时集中重写普通字段的展示分组，不改变字段保存键。通过幂等 SQL 停用重复时间字段的表单关联，保留数据字段和历史记录。

**Tech Stack:** Go、Dever ORM、PostgreSQL、CRM front plugin 现有动态表单协议。

---

### Task 1: 合并会议任务字段分组

**Files:**
- Modify: `service/work_schedule_meeting.go`
- Modify: `service/work.go`

- [x] **Step 1: 提取会议分组常量，并将系统开始时间名称改为“预约时间”**

```go
const (
	workMeetingGroupKey   = "system_meeting"
	workMeetingGroupLabel = "会议预约"
)
```

三个系统字段统一复用常量，第一个字段的 `name` 改为 `预约时间`。

- [x] **Step 2: 新增 `workMeetingTaskFormFields`，复用 `copyMap` 将普通表单字段归入会议分组**

```go
func workMeetingTaskFormFields(ctx context.Context, configuredFields []map[string]any) []map[string]any {
	fields := workMeetingFormFields(ctx)
	for _, field := range configuredFields {
		if inputText(field["field_type"]) == "group" {
			fields = append(fields, field)
			continue
		}
		groupedField := copyMap(field)
		delete(groupedField, "group_id")
		groupedField["group_key"] = workMeetingGroupKey
		groupedField["group_label"] = workMeetingGroupLabel
		fields = append(fields, groupedField)
	}
	return fields
}
```

- [x] **Step 3: 在 `attachWorkTaskForm` 中让会议任务使用统一字段组装入口**

```go
if booleanFromAny(task["meeting_enabled"]) {
	fields = workMeetingTaskFormFields(ctx, fields)
}
```

### Task 2: 停用重复时间字段

**Files:**
- Create: `migrations/postgres/025_meeting_booking_form_consolidation.sql`

- [x] **Step 1: 按会议任务和稳定字段编码停用“预计到访时间”的表单关联**

```sql
UPDATE gjj_crm_form_field AS form_field
SET status = 2,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_task AS task,
     gjj_crm_data_field AS field
WHERE task.meeting_enabled = TRUE
  AND task.form_id = form_field.form_id
  AND field.id = form_field.data_field_id
  AND field.field_key = 'yydf.yujidaofangshijian'
  AND form_field.status = 1;
```

- [x] **Step 2: 保持数据字段、数据记录和历史操作记录不变**

迁移只更新 `gjj_crm_form_field.status`，不更新或删除 `gjj_crm_data_field`、`gjj_crm_data_record`、`gjj_crm_operation_log`。

### Task 3: 静态验证与数据对账

**Files:**
- Modify: `.trellis/tasks/07-19-crm-collaboration-workflow-enhancements/implement.md`

- [x] **Step 1: 执行 `gofmt -d` 和 `git diff --check`，预期无输出**

- [x] **Step 2: 执行迁移并查询表单 21，预期仅“到访次数”保持启用**

- [x] **Step 3: 确认现有 Dever watcher 进程存活；按项目要求不运行 build 或测试**

## Self-Review

- 设计覆盖单页签、四字段顺序、历史数据保留和原保存逻辑复用。
- 未新增前端专用分支、CRUD Service、API 或数据模板行为映射。
- 文件和函数名与现有代码一致，无占位实现。
