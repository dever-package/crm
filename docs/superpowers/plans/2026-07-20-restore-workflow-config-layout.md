# Workflow Configuration Layout Restoration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 恢复流程配置的顶部流程 Tab、左侧阶段、右侧任务布局，简化任务编辑高级配置，并清理误建的空交付流程。

**Architecture:** 复用已提交的 `nav-tab + show-category-list + show-table` 页面结构，不新增前端组件。任务高级配置仍使用现有 Task 字段和条件显示，只调整字段顺序、名称及显示条件；空流程只在确认无阶段、任务和实例后从当前数据库删除。

**Tech Stack:** Dever page JSON、Go Model 元数据、PostgreSQL。

**Verification Constraint:** 项目明确禁止运行 build 和任何 test。本计划只执行 JSON 解析、Dever audit、差异检查、只读数据库核验和服务监听检查；UI 由用户手工验证。未经用户要求不创建 Git commit。

---

### Task 1: 恢复流程配置三层布局

**Files:**
- Modify: `front/page/admin/workflow/list.json`

- [x] **Step 1: 恢复顶部流程操作区和流程 Tab**

将当前左侧 `workflow-list` 恢复为页头“新增/编辑/删除流程”按钮和 `workflow-tabs-row` 的 `nav-tab`，选中值继续写入 `data.search.workflow_id`。

- [x] **Step 2: 恢复左阶段右任务布局**

`content-shell` 左列只保留 `stage-column`，右列继续保留任务搜索、任务表格和新增任务入口；流程编辑和删除继续使用当前选中的 `data.search.workflow_id`。

- [x] **Step 3: 清理只服务于错误双列表布局的状态**

从 `data.actionTarget` 删除 `workflowId` 和 `deleteWorkflowId`，阶段、任务弹窗状态和目标 ID 保持不变。

### Task 2: 简化任务高级配置

**Files:**
- Modify: `front/page/admin/task/update.json`

- [x] **Step 1: 基础字段保持在前**

顺序固定为任务名称、任务类型、按类型出现的表单或规则、必做任务、负责方式、负责部门、办理期限。

- [x] **Step 2: 高级配置移动到底部**

将 `state.show_advanced` 开关放在办理期限之后；默认值继续为 `false`，原高级字段值仍保留在 `data.form` 和提交白名单中。

- [x] **Step 3: 使用业务化名称并收紧显示范围**

将名称调整为“任务创建方式”“任务适用条件”“审核驳回后回到”“完成后触发任务”“加入案件会议”“会议开始时间来源”“会议时长来源”“会议室来源”。驳回目标仅审核任务显示；会议来源仅填写资料任务显示；“加入案件会议”保留给所有任务类型，用于把普通事项、审核等任务的负责人加入同一会议。其余高级项只在高级配置开启时显示。

### Task 3: 清理误建的空交付流程

**Data:**
- Delete: 当前数据库 `gjj_crm_workflow` 中名称为“交付流程”、状态为停用且无阶段/实例引用的记录。

- [x] **Step 1: 删除前再次核验引用**

查询目标流程的阶段数、通过阶段关联的任务数和流程实例数；三者必须均为 0，否则停止删除。

- [x] **Step 2: 事务内精确删除**

只按已核验的流程 ID 删除该行，不修改“签约流程”和“运营流程”中的“交付部接单”任务。

- [x] **Step 3: 删除后核验**

确认“交付流程”记录为 0，同时签约流程、运营流程和两条“交付部接单”任务仍存在。

### Task 4: 静态验证与交付

**Files:**
- Verify: `front/page/admin/workflow/list.json`
- Verify: `front/page/admin/task/update.json`

- [x] **Step 1: 解析 JSON**

运行 `jq empty front/page/admin/workflow/list.json front/page/admin/task/update.json`，期望退出码为 0。

- [x] **Step 2: 执行 Dever 静态审计**

运行 `bash /root/.agents/skills/shemic-dever/scripts/audit.sh front/page/admin/workflow/list.json front/page/admin/task/update.json`，期望输出 `dever skill audit 通过`。

- [x] **Step 3: 检查差异和服务**

运行目标文件的 `git diff --check`，确认无空白错误；确认 `/data/project/demo/gjj/tmp/dever-run/app` 仍监听 `8082`。不运行 build 或 test。
