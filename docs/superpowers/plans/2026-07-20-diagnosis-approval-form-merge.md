# 诊断确认任务合并 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将“确认正式T”表单合并到“确认诊断结果”审批任务，在一次提交中完成正式 T 保存与审批，同时保留驳回和历史审计记录。

**Architecture:** 复用现有任务 `form_id` 配置，让审批任务可选挂载资料表单。前端按配置渲染表单；后端仅在审批通过时，在同一数据库事务中校验并保存表单、写审批记录、完成待办。幂等迁移只停用旧任务并处理与待审批任务重复的当前待办，不修改已完成待办和操作日志。

**Tech Stack:** Go、Dever ORM、React/TypeScript、Dever page JSON、PostgreSQL migration

**Project Verification Constraint:** 按项目要求不执行 `npm run build`、`go test` 或任何测试命令；仅执行格式化、JSON/SQL 静态检查、Dever audit 和差异检查。

---

### Task 1: 允许审批任务配置资料表单

**Files:**
- Modify: `service/setting/workflow.go`
- Modify: `front/page/admin/task/update.json`

- [ ] **Step 1: 扩展任务配置校验**

让 `approval` 接受可选 `form_id`，配置后校验表单已启用并继续执行流程主体字段边界校验；其他非表单任务仍清空 `form_id`。

```go
case crmmodel.TaskTypeApproval:
    if validateTarget && formID > 0 && crmmodel.NewFormModel().Find(...) == nil {
        panicCrmField("form.form_id", "审核任务选择的资料表单不存在或未启用。")
    }
```

- [ ] **Step 2: 后台编辑页同时对 form 和 approval 展示任务表单**

使用当前 page JSON 的 `hiddenCondition: "all"`，仅在任务类型既不是 `form` 也不是 `approval` 时隐藏。

```json
"hiddenCondition": "all",
"hiddenWhen": [
  { "path": "form.task_type", "operator": "notEquals", "value": "form" },
  { "path": "form.task_type", "operator": "notEquals", "value": "approval" }
]
```

### Task 2: 工作台按配置展示审批表单

**Files:**
- Modify: `service/work_query.go`
- Modify: `front/src/nodes/show/work-auth.tsx`

- [ ] **Step 1: API 为带 form_id 的审批任务附加表单定义**

```go
taskType := inputText(task["task_type"])
if withForm && (taskType == crmmodel.TaskTypeForm || taskType == crmmodel.TaskTypeApproval) {
    attachWorkTaskForm(ctx, task)
}
```

- [ ] **Step 2: 前端渲染审批任务的配置字段**

```ts
return (workTaskIsForm(task) || workTaskIsApproval(task)) &&
  (task.form?.fields || []).length > 0;
```

- [ ] **Step 3: 驳回时只校验审批动作字段**

当 `approval_result === "rejected"` 时，跳过配置表单必填项，仅保留“审核结果”和“审核意见”校验；通过时仍校验全部配置字段。

### Task 3: 审批通过时事务保存配置表单

**Files:**
- Modify: `service/work_todo_execute.go`

- [ ] **Step 1: 用 Dever ORM transaction 包裹审批写入**

```go
err := orm.Transaction(ctx, func(txCtx context.Context) error {
    // 表单、审批记录和待办状态使用同一个 txCtx。
    return nil
})
```

- [ ] **Step 2: 仅在通过时收集并保存可选表单**

复用 `collectOptionalWorkFormInput`、`saveWorkFormInput`、`buildWorkFormOperationSnapshot` 和 `saveWorkFormDataRecords`。表单校验或保存失败时整体回滚；驳回时不写正式 T。

- [ ] **Step 3: 保持现有审批路由语义**

无驳回目标时保持当前待办为 pending；有驳回目标时完成当前待办并激活目标任务。无表单的既有审批任务行为不变。

### Task 4: 幂等迁移当前诊断任务配置

**Files:**
- Create: `migrations/postgres/022_diagnosis_approval_form_merge.sql`

- [ ] **Step 1: 将正式 T 表单挂到确认诊断结果**

按启用流程、诊断阶段和当前任务配置查找 ID；迁移脚本内可以使用名称定位，运行时代码不得使用业务名称。

- [ ] **Step 2: 停用确认正式T任务**

保留任务行、已完成待办和操作日志，只将旧任务设为停用，防止新阶段继续生成第二个待办。

- [ ] **Step 3: 合并重复的当前待办**

仅当同一流程实例同时存在 pending 的旧表单待办和 pending 的诊断审批待办时，将旧表单待办标记为 canceled；其他历史或特殊当前状态保持不变。

### Task 5: 文档与静态核对

**Files:**
- Modify: `.trellis/tasks/07-19-crm-collaboration-workflow-enhancements/prd.md`
- Modify: `.trellis/tasks/07-19-crm-collaboration-workflow-enhancements/design.md`
- Modify: `.trellis/tasks/07-19-crm-collaboration-workflow-enhancements/implement.md`

- [ ] **Step 1: 记录需求、设计和实施项**

明确审批表单是配置能力，正式 T 仅在通过时落库，历史记录不重写。

- [ ] **Step 2: 执行允许的静态检查**

```bash
gofmt -w service/work_todo_execute.go service/work_query.go service/setting/workflow.go
jq empty front/page/admin/task/update.json
bash skills/skills-dever/scripts/audit.sh <changed paths>
git diff --check
```

- [ ] **Step 3: 应用迁移并只读核对数据库状态**

确认审批任务已挂表单、旧任务已停用、已完成记录数量不变；不执行自动测试，由用户在浏览器完成通过和驳回验收。
