# CRM Work Dashboard Density Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 CRM 工作台统计页改造成紧凑的数据看板，在首屏展示统计带、趋势和最近操作，同时保留现有接口与钻取行为。

**Architecture:** `stats.json` 只保留统计节点，刷新入口移动到 `ShowCrmWorkStats` 内部。`work-auth.tsx` 继续复用现有 summary 请求、指标映射、趋势 ECharts 和客户列表跳转；六个指标改为单一统计带，分布图改为原生排行列表，避免新增依赖和并行实现。

**Tech Stack:** Dever Page JSON、React 19、TypeScript、Tailwind utility classes、ECharts、Lucide React。

**Verification constraint:** 按项目要求，不运行 `npm run build`、`dever build` 或任何自动化测试。仅使用 `git diff --check`、当前 Dever 环境和 Playwright 浏览器检查。

---

### Task 1: 简化统计页结构并重组看板

**Files:**
- Modify: `front/page/work/stats.json`
- Modify: `front/src/nodes/show/work-auth.tsx:1810`

- [ ] **Step 1: 删除 Page JSON 的独立刷新占位**

将 `stats.json` 改为只有内容容器，避免工作台标题栏下方出现单独的刷新空行：

```json
{
  "page": {
    "name": "工作台",
    "icon": "layout-dashboard",
    "parent": "crm-work-customer-center",
    "type": 1,
    "sort": 0
  },
  "layout": {
    "type": "container",
    "className": "crm-body-page",
    "children": {
      "content": {
        "type": "container",
        "className": "min-w-0"
      }
    }
  },
  "nodes": {
    "content": [
      { "type": "show-crm-work-stats" }
    ]
  },
  "data": {},
  "state": {},
  "action": {}
}
```

- [ ] **Step 2: 将刷新与更新时间放入统计状态栏**

在 `ShowCrmWorkStats` 的返回结构顶部加入无卡片状态栏：

```tsx
<div className="flex flex-wrap items-center justify-between gap-3">
  <div>
    <h2 className="text-sm font-semibold text-foreground">今日概览</h2>
    <p className="mt-1 text-xs text-muted-foreground">
      当前账号的客户、资产、任务与操作汇总
    </p>
  </div>
  <div className="flex items-center gap-3">
    <span className="text-xs text-muted-foreground">
      更新于 {formatWorkDate(summary.generated_at)}
    </span>
    <ShowCrmWorkRefreshButton />
  </div>
</div>
```

- [ ] **Step 3: 重组统计内容顺序**

将 `ShowCrmWorkStats` 主体改为：状态栏、统计带、趋势/最近操作双栏、两个分布排行双栏。

```tsx
return (
  <div className="grid gap-3">
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 className="text-sm font-semibold text-foreground">今日概览</h2>
        <p className="mt-1 text-xs text-muted-foreground">
          当前账号的客户、资产、任务与操作汇总
        </p>
      </div>
      <div className="flex items-center gap-3">
        <span className="text-xs text-muted-foreground">
          更新于 {formatWorkDate(summary.generated_at)}
        </span>
        <ShowCrmWorkRefreshButton />
      </div>
    </div>
    <WorkStatsMetricGrid metrics={summary.metrics || []} />
    <div className="grid min-w-0 gap-3 xl:grid-cols-[minmax(0,1.65fr)_minmax(320px,0.75fr)]">
      <WorkStatsTrendCard points={summary.trend || []} />
      <WorkStatsRecentOperations
        operations={summary.recent_operations || []}
        loading={loading}
      />
    </div>
    <div className="grid min-w-0 gap-3 lg:grid-cols-2">
      <WorkStatsBreakdownCard
        title="阶段分布"
        description="客户或资产当前所在阶段"
        rows={summary.stage_breakdown || []}
        emptyText="暂无阶段数据"
        drilldownType="stage"
      />
      <WorkStatsBreakdownCard
        title="待办任务类型"
        description="当前待处理任务按动作类型汇总"
        rows={summary.task_breakdown || []}
        emptyText="当前没有待办任务"
        drilldownType="task"
      />
    </div>
  </div>
);
```

- [ ] **Step 4: 提交页面结构变更**

```bash
git add front/page/work/stats.json front/src/nodes/show/work-auth.tsx
git commit -m "refactor: compact crm dashboard structure"
```

### Task 2: 将六个指标改成连续统计带

**Files:**
- Modify: `front/src/nodes/show/work-auth.tsx:1910`

- [ ] **Step 1: 使用单一表面和 `gap-px` 分隔指标**

完整替换 `WorkStatsMetricGrid`：

```tsx
function WorkStatsMetricGrid({ metrics }: { metrics: WorkSummaryMetric[] }) {
  if (metrics.length === 0) {
    return (
      <div className="rounded-md border border-border/70 bg-background px-5 py-10">
        <WorkEmptyText>暂无统计指标</WorkEmptyText>
      </div>
    );
  }
  return (
    <div className="grid grid-cols-2 gap-px overflow-hidden rounded-md border border-border/70 bg-border/70 md:grid-cols-3 min-[1440px]:grid-cols-6">
      {metrics.map((metric) => (
        <WorkStatsMetricCard
          key={textValue(metric.key || metric.name)}
          metric={metric}
          onOpen={() => openWorkCustomerList(workStatsMetricDrilldown(metric))}
        />
      ))}
    </div>
  );
}
```

- [ ] **Step 2: 压缩单项高度和视觉层级**

完整替换 `WorkStatsMetricCard`，保留整项点击与现有图标映射：

```tsx
function WorkStatsMetricCard({ metric, onOpen }: {
  metric: WorkSummaryMetric;
  onOpen: () => void;
}) {
  const Icon = workStatsMetricIcon(metric.key);
  return (
    <button
      type="button"
      className="group min-h-[94px] bg-background px-4 py-3 text-left transition-colors hover:bg-muted/20 focus-visible:z-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
      onClick={onOpen}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-xs font-medium text-muted-foreground">
          {displayText(metric.name)}
        </span>
        <Icon className="h-4 w-4 shrink-0 text-muted-foreground/80" />
      </div>
      <div className="mt-1.5 text-2xl font-semibold leading-8 text-foreground">
        {displayText(metric.value, "0")}
      </div>
      <p className="mt-1 truncate text-[11px] leading-4 text-muted-foreground">
        {displayText(metric.description)}
      </p>
    </button>
  );
}
```

- [ ] **Step 3: 提交统计带变更**

```bash
git add front/src/nodes/show/work-auth.tsx
git commit -m "style: add compact crm metric strip"
```

### Task 3: 压缩趋势与最近操作面板

**Files:**
- Modify: `front/src/nodes/show/work-auth.tsx:2023`

- [ ] **Step 1: 统一面板外观**

提取三个统计面板共同使用的样式和标题结构：

```tsx
const workStatsPanelClass =
  "rounded-md border border-border/70 bg-background p-4";

function WorkStatsPanelHeader({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div>
      <h3 className="text-sm font-semibold leading-5 text-foreground">
        {title}
      </h3>
      <p className="mt-0.5 text-xs leading-5 text-muted-foreground">
        {description}
      </p>
    </div>
  );
}
```

趋势面板改为固定桌面高度并复用标题：

```tsx
function WorkStatsTrendCard({ points }: { points: WorkSummaryTrendPoint[] }) {
  return (
    <section className={`${workStatsPanelClass} flex min-h-0 flex-col xl:h-[304px]`}>
      <WorkStatsPanelHeader
        title="近 14 天趋势"
        description="任务完成、阶段流转和操作记录"
      />
      <div className="mt-3 min-h-0 flex-1">
        {points.length === 0 ? (
          <WorkEmptyText>暂无趋势数据</WorkEmptyText>
        ) : (
          <WorkStatsTrendChart points={points} />
        )}
      </div>
    </section>
  );
}
```

最近操作面板使用相同高度，列表在面板内部滚动：

```tsx
function WorkStatsRecentOperations({ operations, loading }: {
  operations: WorkOperation[];
  loading: boolean;
}) {
  return (
    <section className={`${workStatsPanelClass} flex min-h-0 flex-col xl:h-[304px]`}>
      <WorkStatsPanelHeader
        title="最近操作"
        description="当前账号最近提交的任务和流转"
      />
      <div className="mt-3 min-h-0 flex-1 overflow-y-auto pr-1">
        {loading && operations.length === 0 ? (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            正在加载最近操作
          </div>
        ) : operations.length === 0 ? (
          <WorkEmptyText>暂无最近操作</WorkEmptyText>
        ) : (
          <div>
            {operations.map((operation, index) => (
              <WorkStatsOperationRow
                key={workOperationTimelineKey(operation, index)}
                operation={operation}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
```

- [ ] **Step 2: 缩小趋势图并消除移动横向滚动**

完整替换 `WorkStatsTrendChart`：

```tsx
function WorkStatsTrendChart({ points }: { points: WorkSummaryTrendPoint[] }) {
  const option = useMemo(() => buildWorkStatsTrendOption(points), [points]);
  return (
    <CrmEChart
      option={option}
      height={220}
      minWidth={0}
      ariaLabel="近 14 天工作趋势"
    />
  );
}
```

同时将趋势 option 的 `grid.top` 调整为 `34`、图例字号调整为 `11`、折线宽度调整为 `2`、节点大小调整为 `5`，保留三条系列和 tooltip。

- [ ] **Step 3: 将最近操作从卡片列表改为分隔列表**

完整替换 `WorkStatsOperationRow` 的外层样式：

```tsx
<article className="border-b border-border/60 py-2.5 last:border-b-0">
  <div className="flex min-w-0 items-start justify-between gap-3">
    <div className="min-w-0">
      <div className="flex min-w-0 flex-wrap items-center gap-1.5">
        <span className="min-w-0 truncate text-xs font-medium text-foreground">
          {workOperationTitle(operation)}
        </span>
        <span className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${tone.badge}`}>
          {workOperationBadgeText(operation)}
        </span>
      </div>
      {description ? (
        <p className="mt-1 line-clamp-2 text-[11px] leading-4 text-muted-foreground">
          {description}
        </p>
      ) : null}
    </div>
    <span className="shrink-0 whitespace-nowrap text-[10px] leading-4 text-muted-foreground">
      {formatWorkDate(operation.created_at || operation.create_time)}
    </span>
  </div>
</article>
```

- [ ] **Step 4: 提交分析区变更**

```bash
git add front/src/nodes/show/work-auth.tsx
git commit -m "style: compact crm dashboard analysis panels"
```

### Task 4: 用排行列表替换分布柱状图

**Files:**
- Modify: `front/src/nodes/show/work-auth.tsx:2125-2275`

- [ ] **Step 1: 新增可复用排行列表**

以一个参数化组件同时服务阶段和任务分布：

```tsx
function WorkStatsBreakdownList({ rows, type }: {
  rows: WorkSummaryBreakdown[];
  type: "stage" | "task";
}) {
  return (
    <div className="mt-3 grid gap-1">
      {rows.map((row) => {
        const value = textValue(row.key || row.name);
        const percent = workStatsPercent(row.percent);
        const params = type === "stage"
          ? { mode: "all", stage_filter: value }
          : { mode: "pending", task_filter: value };
        return (
          <button
            type="button"
            key={`${type}:${value}`}
            className="group rounded-md px-2 py-2 text-left transition-colors hover:bg-muted/25"
            onClick={() => openWorkCustomerList(params)}
          >
            <span className="flex items-center justify-between gap-3">
              <span className="min-w-0 truncate text-xs font-medium text-foreground">
                {displayText(row.name)}
              </span>
              <span className="shrink-0 text-xs font-semibold text-foreground">
                {workStatsNumber(row.count)}
                <small className="ml-2 font-normal text-muted-foreground">{percent}%</small>
              </span>
            </span>
            <span className="mt-1.5 block h-1 overflow-hidden rounded-full bg-muted">
              <span
                className={`block h-full rounded-full ${type === "stage" ? "bg-blue-600" : "bg-emerald-600"}`}
                style={{ width: `${percent}%` }}
              />
            </span>
          </button>
        );
      })}
    </div>
  );
}
```

- [ ] **Step 2: 在分布面板中使用排行列表**

`WorkStatsBreakdownCard` 复用相同面板表面、标题和空状态：

```tsx
function WorkStatsBreakdownCard({
  title,
  description,
  rows,
  emptyText,
  drilldownType,
}: {
  title: string;
  description: string;
  rows: WorkSummaryBreakdown[];
  emptyText: string;
  drilldownType: "stage" | "task";
}) {
  return (
    <section className={workStatsPanelClass}>
      <WorkStatsPanelHeader title={title} description={description} />
      {rows.length === 0 ? (
        <div className="mt-3">
          <WorkEmptyText>{emptyText}</WorkEmptyText>
        </div>
      ) : (
        <WorkStatsBreakdownList rows={rows} type={drilldownType} />
      )}
    </section>
  );
}
```

- [ ] **Step 3: 删除重复实现**

删除不再使用的 `WorkStatsBreakdownDrilldowns`、`WorkStatsBreakdownChart` 和 `buildWorkStatsBreakdownOption`。保留 `workStatsNumber`、`workStatsPercent`，因为排行列表继续使用。

- [ ] **Step 4: 静态检查并提交**

```bash
git diff --check
rg -n "WorkStatsBreakdown(Chart|Drilldowns)|buildWorkStatsBreakdownOption" front/src/nodes/show/work-auth.tsx
git add front/src/nodes/show/work-auth.tsx
git commit -m "refactor: replace crm breakdown charts with rankings"
```

预期：`git diff --check` 无输出；`rg` 无匹配并返回退出码 1。

### Task 5: 浏览器验收

**Files:**
- Modify when required: `front/src/nodes/show/work-auth.tsx`
- Modify when required: `front/page/work/stats.json`

- [ ] **Step 1: 确认现有 Dever 环境**

检查 8082 和 18082 端口；环境未运行时执行项目既有 `dever run`。不运行 build 或测试命令。

- [ ] **Step 2: 检查 1920x1080 首屏**

登录 `/work` 并打开 `/work/crm/stats`，确认：

- 顶部无独立刷新空行。
- 六个指标在一条统计带中。
- 趋势和最近操作同处首屏双栏。
- 两个分布区域显示排行和进度条，不显示柱状图。

- [ ] **Step 3: 检查钻取与刷新**

点击一个指标、一个阶段排行和一个任务排行，确认进入 `/work/crm/work` 且查询参数正确。返回工作台后点击刷新，确认更新时间更新。

- [ ] **Step 4: 检查 390x844 移动视口**

确认指标为两列，分析与排行面板为单列，页面没有横向滚动，内容不被底部导航遮挡。

- [ ] **Step 5: 检查浏览器异常并提交必要修正**

确认控制台无新增错误、失败请求为空。若需要修正，只调整本任务涉及的布局类和图表尺寸：

```bash
git diff --check
git add front/src/nodes/show/work-auth.tsx front/page/work/stats.json
git commit -m "fix: refine compact crm dashboard layout"
```

无需修正时不创建空提交。
