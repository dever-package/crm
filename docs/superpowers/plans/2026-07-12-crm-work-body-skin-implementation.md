# CRM Work Body Skin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 CRM `work` 站点完整统一为 `bot/body` 的紧凑视觉风格，同时保持现有导航、业务组件和数据流程不变。

**Architecture:** 在 CRM front plugin 增加单一皮肤节点，由 work 主框架挂载一次；所有覆盖都限定在 `.crm-work-app` 根作用域。现有页面与 React 节点只补语义 class，不复制组件，也不修改 `package/front` 全局 runtime。

**Tech Stack:** Dever page JSON、React、TypeScript、Tailwind class、作用域 CSS、Playwright 浏览器检查。

---

### Task 1: 记录现状并建立 CRM work 皮肤入口

**Files:**
- Create: `front/src/nodes/show/work-skin.tsx`
- Modify: `front/src/plugin.ts`
- Modify: `front/page/work/main.json`

- [ ] **Step 1: 记录当前桌面与手机视图**

使用现有 CRM 浏览器账号打开 `work/crm/stats`、`work/crm/lead`、`work/crm/work`，保存 1440x900 与 390x844 截图，并记录控制台错误和失败请求。该步骤只做浏览器诊断，不运行测试命令。

- [ ] **Step 2: 创建单一皮肤节点**

`work-skin.tsx` 只导出样式节点，不读取数据、不调用 API：

```tsx
const crmWorkSkin = `
  .crm-work-app {
    --crm-work-bg: #f4f6f5;
    --crm-work-surface: #ffffff;
    --crm-work-text: #171a19;
    --crm-work-muted: #6b7370;
    --crm-work-faint: #9ca3a0;
    --crm-work-line: #e4e8e6;
    --crm-work-line-strong: #d2d9d6;
    --crm-work-active: #e4e8e6;
    --crm-work-primary: #1a4a35;
    min-height: 100svh;
    background: var(--crm-work-bg);
    color: var(--crm-work-text);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, "Noto Sans SC", "PingFang SC", sans-serif;
    font-size: 12.8px;
    letter-spacing: 0;
  }
`;

export function ShowCrmWorkSkin() {
  return <style>{crmWorkSkin}</style>;
}
```

- [ ] **Step 3: 注册并挂载皮肤节点**

在 `plugin.ts` 通过独立 lazy loader 注册 `show-crm-work-skin`。在 `main.json` 根布局增加 `className: "crm-work-app"`，并在 `__root__` 节点中挂载皮肤节点一次。

- [ ] **Step 4: 提交皮肤入口**

```bash
git add front/src/nodes/show/work-skin.tsx front/src/plugin.ts front/page/work/main.json
git commit -m "style: add crm work body skin"
```

### Task 2: 统一主框架、页面和数据表密度

**Files:**
- Modify: `front/src/nodes/show/work-skin.tsx`
- Modify: `front/page/work/stats.json`
- Modify: `front/page/work/lead.json`
- Modify: `front/page/work/work.json`
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/src/nodes/show/work-lead.tsx`

- [ ] **Step 1: 为页面增加稳定语义 class**

三个页面根布局统一使用 `crm-work-page`，并分别增加 `crm-work-stats-page`、`crm-work-lead-page`、`crm-work-customers-page`。React 根节点增加 `crm-work-stats`、`crm-work-leads`、`crm-work-customers`，不改变状态和事件处理。

- [ ] **Step 2: 按 body 参数覆盖主框架**

在 `.crm-work-app` 下使用稳定 `data-slot`/`data-sidebar` 选择器：

```css
.crm-work-app [data-slot="sidebar-wrapper"] { background: var(--crm-work-bg); }
.crm-work-app [data-slot="sidebar-container"] { width: 240px; padding: 11px 8px 11px 8px; }
.crm-work-app [data-slot="sidebar-gap"] { width: 240px; }
.crm-work-app [data-sidebar="sidebar"] { border: 0; border-radius: 0; box-shadow: none; background: var(--crm-work-bg); }
.crm-work-app [data-slot="sidebar-inset"] { margin: 11px 11px 11px 0; border-radius: 6px; box-shadow: none; background: #fff; overflow: hidden; }
.crm-work-app [data-slot="sidebar-inset"] > header { height: 38px; min-height: 38px; border-color: var(--crm-work-line); background: #fff; }
.crm-work-app [data-slot="sidebar-inset"] > header > div { min-height: 38px; padding: 0 18px; }
```

侧栏菜单高度固定为 40px、图标 16px、文字 12.8px，激活态使用 `#e4e8e6`，圆角 6px；折叠态继续使用 front 原生行为。

- [ ] **Step 3: 统一页面与表格**

`.crm-work-page` 使用白色内容背景和 `24px 31px 40px` 桌面内边距。页面标题为 14.5px/500，说明文字为 11px；表格基础字号 12.8px，表头 11.5px/500，单元格纵向内边距 10px；筛选控件高度 32px，按钮高度 30px，边框与悬停色使用 body 令牌。

- [ ] **Step 4: 统一状态和统计容器**

状态标签保持语义颜色，但改为 10.5px、20px 高度和 4px 圆角。统计卡片去除大阴影和 8px 大圆角，统一为 3px/6px 圆角、细边框和紧凑间距；数字字号从 30px 降为 22px，避免工作台营销化。

- [ ] **Step 5: 提交主内容样式**

```bash
git add front/page/work front/src/nodes/show/work-auth.tsx front/src/nodes/show/work-lead.tsx front/src/nodes/show/work-skin.tsx
git commit -m "style: align crm work pages with body"
```

### Task 3: 统一详情、弹窗与移动端

**Files:**
- Modify: `front/src/nodes/show/work-skin.tsx`
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/src/nodes/show/work-lead.tsx`

- [ ] **Step 1: 统一反馈组件**

在 CRM 根作用域内调整 `[role="dialog"]`、`[data-slot="dialog-content"]`、抽屉和表单控件：标题 14.5px，正文 12.8px，说明 11px，输入框 34px，textarea 保持可调整高度，圆角 6px，阴影使用单层低透明度阴影。保留现有宽度和任务表单滚动逻辑。

- [ ] **Step 2: 统一客户详情与流程记录**

复用已有 `data-crm-work-detail`、`data-crm-work-record-detail` 和流程组件结构，将嵌套卡片降为无阴影分区；Tab、操作记录、资料表格和流程任务使用相同字号、边框和间距，不改变按钮权限或任务状态。

- [ ] **Step 3: 校准移动端**

在 `max-width: 767px` 下：页面内边距 14px；标题和操作区允许换行；筛选输入占满一行；移动卡片维持 6px 圆角；弹窗宽度为 `calc(100vw - 20px)`；任何按钮、标签、文字不得重叠或横向溢出。

- [ ] **Step 4: 浏览器检查并修正**

分别检查工作台、线索池、客户列表、客户详情和任务弹窗的桌面/手机截图。检查 body 字体栈、12.8px 基准、240px 侧栏、38px 顶栏、表格密度、遮挡、控制台错误和失败请求；只修正样式断点，不改业务逻辑。

- [ ] **Step 5: 提交反馈界面与响应式样式**

```bash
git add front/src/nodes/show/work-skin.tsx front/src/nodes/show/work-auth.tsx front/src/nodes/show/work-lead.tsx
git commit -m "style: refine crm work feedback surfaces"
```

### Task 4: 静态检查与最终浏览器验收

**Files:**
- Verify: all files changed by Tasks 1-3

- [ ] **Step 1: 检查源码格式和 JSON**

```bash
git diff --check
jq empty front/page/work/main.json front/page/work/stats.json front/page/work/lead.json front/page/work/work.json
```

预期：命令退出码为 0，无输出错误。

- [ ] **Step 2: 执行 Dever 静态 audit**

```bash
bash /root/.agents/skills/shemic-dever/scripts/audit.sh --changed
```

预期：输出 `dever skill audit 通过`。

- [ ] **Step 3: 最终浏览器验收**

保存线索池、客户列表、工作台和客户详情的桌面/手机最终截图；确认各页面能加载、交互入口存在、控制台错误为 0、失败请求为 0。按用户要求不运行 `npm run build`、`go test` 或任何测试命令。

- [ ] **Step 4: 确认提交与工作区状态**

```bash
git status --short --branch
git log -4 --oneline
```

预期：工作区无未提交的本次改动，最近提交对应皮肤入口、页面样式和响应式调整。
