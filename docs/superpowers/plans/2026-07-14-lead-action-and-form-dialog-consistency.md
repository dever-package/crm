# Lead Action And Form Dialog Consistency Implementation Plan

> **For agentic workers:** Execute inline in the current session. The user explicitly forbids subagents, builds and automated tests.

**Goal:** 去除线索确认与编辑的重复入口，并统一工作台录入、编辑和流程任务表单的弹窗体验。

**Architecture:** 在线索列表中通过一个明确的表单任务判断函数控制编辑入口。任务表单继续使用现有配置驱动字段渲染器，只收敛公共标题、业务对象描述、紧凑/工作区布局和底部操作，不复制业务表单。

**Tech Stack:** React、TypeScript、Dever Page JSON

---

### Task 1: 消除线索动作冲突

**Files:**
- Modify: `front/src/nodes/show/work-lead.tsx`

- [x] 提取“是否存在待处理表单任务”的判断，避免把任务类型条件散落在 JSX 中。
- [x] 表单任务待处理时隐藏“编辑”，表单任务完成后恢复“编辑”，保留其他权限和状态判断。
- [x] 检查桌面表格和移动列表共用同一个 `LeadActions`，避免维护两套规则。

### Task 2: 统一任务弹窗标题与业务对象描述

**Files:**
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/page/work/lead.json`
- Modify: `front/page/work/work.json`

- [x] 任务弹窗标题改用实际任务名称，并生成线索、客户、资产的简短描述。
- [x] 打开线索任务时把线索作为只读上下文交给弹窗展示，提交时仍不产生客户 ID。
- [x] 两个工作台页面使用相同的任务弹窗描述路径、正文滚动和底部操作配置。

### Task 3: 收敛紧凑表单与长表单布局

**Files:**
- Modify: `front/src/nodes/show/work-task-form.tsx`
- Modify: `front/src/nodes/show/work-task-form-fields.tsx`

- [x] 紧凑任务表单移除重复的客户/资产指标块，使用与编辑线索一致的两列字段节奏。
- [x] 长表单保留分组导航和填写进度，但改为扁平上下文，不使用嵌套卡片。
- [x] 统一文本域、附件、多选等宽字段规则以及字段标签、错误提示和响应式单列布局。

### Task 4: 静态自检

**Files:**
- Review: `front/src/nodes/show/work-lead.tsx`
- Review: `front/src/nodes/show/work-auth.tsx`
- Review: `front/src/nodes/show/work-task-form.tsx`
- Review: `front/src/nodes/show/work-task-form-fields.tsx`
- Review: `front/page/work/lead.json`
- Review: `front/page/work/work.json`

- [x] 使用 `jq empty` 检查修改后的 Page JSON。
- [x] 使用 `git diff --check` 检查空白和冲突标记。
- [x] 检查差异范围和重复样式；按照项目要求不运行 build 或任何自动化测试。
