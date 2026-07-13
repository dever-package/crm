# Task Form And Lead Layout Implementation Plan

> **For agentic workers:** Execute inline in the current session. The user explicitly forbids subagents, builds and automated tests.

**Goal:** 明确任务表单与线索数据模板的边界，并简化线索录入弹窗的动态字段布局。

**Architecture:** 保留现有线索数据模板读取与 `data_values` 保存链路，只调整 React 展示结构。后台继续复用现有 Form/FormField CRUD，仅修改业务名称；旧表单通过独立、可重复执行的 PostgreSQL 迁移安全清理。

**Tech Stack:** Go、Dever Page JSON、React/TypeScript、PostgreSQL

---

### Task 1: 统一任务表单命名

**Files:**
- Modify: `model/form.go`
- Modify: `model/form_field.go`
- Modify: `front/page/admin/form/list.json`
- Modify: `front/page/admin/form/update.json`
- Modify: `front/page/admin/form_field/list.json`
- Modify: `front/page/admin/form_field/update.json`
- Modify: `front/page/admin/task/update.json`

- [x] 将“资料模板”改为“任务表单”，“模板字段”改为“表单字段”，路由、模型构造器和数据库表名保持不变。
- [x] 说明任务表单只组合客户信息、客户资产和业务数据字段，不暗示其负责线索录入。
- [x] 使用 `jq empty` 和 `gofmt -d` 完成静态检查，不运行 build 或自动化测试。

### Task 2: 拉平线索动态字段

**Files:**
- Modify: `front/src/nodes/show/work-lead-template-fields.tsx`

- [x] 使用 `templates.flatMap((template) => template.fields || [])` 汇总字段，保留现有字段类型、默认值和变更处理逻辑。
- [x] 直接返回字段控件列表，使它们成为外层两列网格的直接子项；删除标题、分隔线和嵌套网格。
- [x] 检查 TypeScript diff 和组件引用，不运行前端 build 或测试。

### Task 3: 清理废弃任务表单

**Files:**
- Create: `migrations/postgres/013_task_form_cleanup.sql`

- [x] 仅删除状态为停用且不再被任何任务引用的旧签约表单；先删除其表单字段，再删除表单。
- [x] 在现有 CRM PostgreSQL 容器执行迁移，并查询确认当前签约、运营任务表单仍存在。
- [x] 运行最终静态核验，确认开发服务仍可访问；按项目要求不运行 build 或自动化测试。
