import { useCallback, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import {
  Activity,
  AlertTriangle,
  ClipboardList,
  GitBranch,
  Home,
  ListChecks,
  Loader2,
  RefreshCw,
  ShieldCheck,
  TrendingUp,
  Users,
} from "lucide-react";
import { request } from "@dever/front-plugin";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";

import { displayText, errorMessage, formatWorkDate, textValue } from "./work-core";
import {
  CrmEChart,
  crmChartAxisColor,
  crmChartSplitLineColor,
  crmChartTextColor,
  type EChartsOption,
} from "./crm-echarts";

type AdminMetric = {
  key?: string;
  name?: string;
  value?: string | number;
  description?: string;
};

type AdminTrendPoint = {
  date?: string;
  label?: string;
  customer_count?: string | number;
  asset_count?: string | number;
  task_count?: string | number;
  transition_count?: string | number;
  operation_count?: string | number;
  income_amount?: string | number;
  expense_amount?: string | number;
  net_amount?: string | number;
  ledger_count?: string | number;
  [key: string]: string | number | undefined;
};

type AdminBreakdownRow = {
  key?: string;
  name?: string;
  count?: string | number;
  percent?: string | number;
  previous_count?: string | number;
  drop_count?: string | number;
  drop_percent?: string | number;
};

type AdminBacklogRow = AdminBreakdownRow & {
  task_count?: string | number;
  pending_todo_count?: string | number;
  avg_days?: string | number;
  max_days?: string | number;
  stale_3d?: string | number;
  stale_7d?: string | number;
  stale_15d?: string | number;
};

type AdminStaffRow = {
  id?: string | number;
  name?: string;
  task_count?: string | number;
  transition_count?: string | number;
  operation_count?: string | number;
  todo_done_count?: string | number;
  points?: string | number;
  last_active_at?: string;
  total?: string | number;
};

type AdminProbeDimensionRow = {
  key?: string;
  name?: string;
  total?: string | number;
  filled?: string | number;
  missing_count?: string | number;
  percent?: string | number;
};

type AdminProbeSummary = {
  asset_count?: string | number;
  started_asset_count?: string | number;
  complete_asset_count?: string | number;
  field_total?: string | number;
  field_filled?: string | number;
  percent?: string | number;
  dimensions?: AdminProbeDimensionRow[];
  missing_dimensions?: AdminProbeDimensionRow[];
};

type AdminFinanceTypeRow = {
  key?: string;
  name?: string;
  direction?: string;
  count?: string | number;
  amount?: string | number;
  percent?: string | number;
};

type AdminFinanceSummary = {
  metrics?: AdminMetric[];
  trend?: AdminTrendPoint[];
  type_breakdown?: AdminFinanceTypeRow[];
};

type AdminSummary = {
  metrics?: AdminMetric[];
  growth_trend?: AdminTrendPoint[];
  execution_trend?: AdminTrendPoint[];
  funnel?: AdminBreakdownRow[];
  pipeline_funnel?: AdminBreakdownRow[];
  node_backlog?: AdminBacklogRow[];
  task_breakdown?: AdminBreakdownRow[];
  finance_summary?: AdminFinanceSummary;
  staff_ranking?: AdminStaffRow[];
  staff_output?: AdminStaffRow[];
  probe_summary?: AdminProbeSummary;
  generated_at?: string;
};

type AdminApiResponse<T> = {
  code?: number;
  status?: number;
  data?: T;
  msg?: string;
  message?: string;
};

type ChartSeriesKey = string;

type ChartSeries = {
  key: ChartSeriesKey;
  label: string;
  color: string;
};

const growthSeries: ChartSeries[] = [
  { key: "customer_count", label: "新增客户", color: "#2563eb" },
  { key: "asset_count", label: "新增资产", color: "#059669" },
];

const executionSeries: ChartSeries[] = [
  { key: "task_count", label: "任务完成", color: "#111827" },
  { key: "transition_count", label: "阶段流转", color: "#d97706" },
  { key: "operation_count", label: "操作记录", color: "#dc2626" },
];

const financeSeries: ChartSeries[] = [
  { key: "income_amount", label: "收入", color: "#059669" },
  { key: "expense_amount", label: "支出", color: "#dc2626" },
  { key: "net_amount", label: "净额", color: "#2563eb" },
];

type AdminStatsMode = "all" | "business" | "finance" | "performance";

type AdminStatsNodeProps = {
  item?: {
    meta?: Record<string, unknown>;
  };
};

const adminStatsModeTitles: Record<
  AdminStatsMode,
  { title: string; description: string }
> = {
  all: {
    title: "CRM 数据看板",
    description: "分为业务数据、财务统计和绩效统计，统一查看 CRM 运行情况。",
  },
  business: {
    title: "业务数据",
    description: "查看客户、资产、阶段流转、任务执行、节点积压和十一维资料情况。",
  },
  finance: {
    title: "财务统计",
    description: "基于财务用途字段自动生成的流水，统计收入、支出、净额和财务类型。",
  },
  performance: {
    title: "绩效统计",
    description: "基于任务完成、阶段流转、协作待办和任务积分统计人员产出。",
  },
};

let adminApiFreshSeq = 0;

export function ShowCrmAdminStats({ item }: AdminStatsNodeProps = {}) {
  const mode = adminStatsModeFromNode(item);
  const intro = adminStatsModeTitles[mode];
  const [summary, setSummary] = useState<AdminSummary | null>(null);
  const [loading, setLoading] = useState(false);

  const loadSummary = useCallback(async () => {
    setLoading(true);
    try {
      const data = await crmAdminApi<AdminSummary>("/crm/admin/dashboard/summary");
      setSummary(data || {});
    } catch (error) {
      toast.error(errorMessage(error, "数据看板加载失败"));
      setSummary(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadSummary();
  }, [loadSummary]);

  if (loading && !summary) {
    return (
      <div className="rounded-md border bg-background px-6 py-20 shadow-sm">
        <AdminStatsState
          icon="loading"
          title="正在汇总数据"
          description="正在读取客户、资产、阶段任务和操作记录。"
        />
      </div>
    );
  }

  if (!summary) {
    return (
      <div className="rounded-md border bg-background px-6 py-20 shadow-sm">
        <AdminStatsState
          icon="empty"
          title="暂无统计数据"
          description="刷新后仍为空时，请先确认后台 API 权限和 CRM 数据。"
        />
      </div>
    );
  }

  return (
    <div className="grid gap-4">
      <section className="rounded-md border bg-background p-5 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold leading-7">{intro.title}</h2>
            <p className="text-sm leading-6 text-muted-foreground">
              {intro.description}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-xs text-muted-foreground">
              更新时间：{formatWorkDate(summary.generated_at)}
            </span>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={loading}
              onClick={loadSummary}
            >
              {loading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
              刷新
            </Button>
          </div>
        </div>
      </section>

      {adminStatsModeIncludes(mode, "business") ? (
        <>
          {mode === "all" ? (
            <AdminSectionTitle
              title="业务数据"
              description="查看客户、资产、阶段流转、任务执行、节点积压和十一维资料情况。"
            />
          ) : null}
          <AdminMetricGrid metrics={summary.metrics || []} />

          <div className="grid gap-4 2xl:grid-cols-2">
            <AdminCurveChart
              title="增长曲线"
              description="近 14 天新增客户与新增资产。"
              points={summary.growth_trend || []}
              series={growthSeries}
            />
            <AdminLineChart
              title="执行折线"
              description="近 14 天任务完成、阶段流转和操作记录。"
              points={summary.execution_trend || []}
              series={executionSeries}
            />
          </div>

          <div className="grid gap-4 2xl:grid-cols-[1.1fr_0.9fr]">
            <AdminFunnelChart rows={summary.pipeline_funnel || summary.funnel || []} />
            <AdminBreakdownCard
              title="任务类型分布"
              description="按当前阶段估算的可执行任务动作。"
              rows={summary.task_breakdown || []}
              emptyText="暂无任务分布"
            />
          </div>

          <div className="grid gap-4 2xl:grid-cols-[1.05fr_0.95fr]">
            <AdminNodeBacklog rows={summary.node_backlog || []} />
            <AdminProbeSummaryCard summary={summary.probe_summary} />
          </div>
        </>
      ) : null}

      {adminStatsModeIncludes(mode, "finance") ? (
        <>
          {mode === "all" ? (
            <AdminSectionTitle
              title="财务统计"
              description="基于财务用途字段自动生成的流水，统计收入、支出、净额和财务类型。"
            />
          ) : null}
          <AdminFinanceDashboard summary={summary.finance_summary} />
        </>
      ) : null}

      {adminStatsModeIncludes(mode, "performance") ? (
        <>
          {mode === "all" ? (
            <AdminSectionTitle
              title="绩效统计"
              description="基于任务完成、阶段流转、协作待办和任务积分统计人员产出。"
            />
          ) : null}
          <AdminStaffRanking rows={summary.staff_output || summary.staff_ranking || []} />
        </>
      ) : null}
    </div>
  );
}

function adminStatsModeFromNode(item?: AdminStatsNodeProps["item"]): AdminStatsMode {
  const rawMode = textValue(item?.meta?.mode);
  if (rawMode === "business" || rawMode === "finance" || rawMode === "performance") {
    return rawMode;
  }
  return "all";
}

function adminStatsModeIncludes(mode: AdminStatsMode, section: Exclude<AdminStatsMode, "all">) {
  return mode === "all" || mode === section;
}

function AdminSectionTitle({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="mt-2">
      <h2 className="text-lg font-semibold leading-7">{title}</h2>
      <p className="text-sm leading-6 text-muted-foreground">{description}</p>
    </div>
  );
}

async function crmAdminApi<T>(path: string): Promise<T> {
  const result = (await request(freshAdminApiPath(path), "get", {})) as
    | AdminApiResponse<T>
    | T;
  return unwrapAdminApiResult<T>(result);
}

function freshAdminApiPath(path: string): string {
  adminApiFreshSeq += 1;
  const joiner = path.includes("?") ? "&" : "?";
  return `${path}${joiner}_r=${Date.now()}_${adminApiFreshSeq}`;
}

function unwrapAdminApiResult<T>(result: AdminApiResponse<T> | T): T {
  if (isAdminApiResponse(result)) {
    const code = typeof result.code === "number" ? result.code : 0;
    const status = typeof result.status === "number" ? result.status : 1;
    if (code !== 0 || (status > 0 && status !== 1)) {
      throw new Error(result.msg || result.message || "请求失败");
    }
    return (result.data ?? result) as T;
  }
  return result as T;
}

function isAdminApiResponse<T>(value: AdminApiResponse<T> | T): value is AdminApiResponse<T> {
  return Boolean(
    value &&
      typeof value === "object" &&
      ("status" in value || "code" in value) &&
      "data" in value,
  );
}

function AdminMetricGrid({ metrics }: { metrics: AdminMetric[] }) {
  if (metrics.length === 0) {
    return (
      <div className="rounded-md border bg-background px-6 py-12 shadow-sm">
        <AdminEmptyText>暂无指标数据</AdminEmptyText>
      </div>
    );
  }
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {metrics.map((metric) => (
        <AdminMetricCard key={textValue(metric.key || metric.name)} metric={metric} />
      ))}
    </div>
  );
}

function AdminMetricCard({ metric }: { metric: AdminMetric }) {
  const Icon = adminMetricIcon(metric.key);
  return (
    <article className="rounded-md border bg-background p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-sm leading-5 text-muted-foreground">
            {displayText(metric.name)}
          </div>
          <div className="mt-2 text-3xl font-semibold leading-9">
            {formatNumber(metric.value)}
          </div>
        </div>
        <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md border bg-muted/25 text-muted-foreground">
          <Icon className="h-5 w-5" />
        </span>
      </div>
      <p className="mt-3 text-xs leading-5 text-muted-foreground">
        {displayText(metric.description)}
      </p>
    </article>
  );
}

function adminMetricIcon(key?: string) {
  switch (textValue(key)) {
    case "customers":
      return Users;
    case "assets":
    case "missing_assets":
      return Home;
    case "stage_targets":
    case "transitions_14d":
      return GitBranch;
    case "pending_todos":
    case "tasks_14d":
      return ClipboardList;
    case "operations_14d":
      return Activity;
    default:
      return TrendingUp;
  }
}

function AdminFinanceDashboard({ summary }: { summary?: AdminFinanceSummary }) {
  const metrics = summary?.metrics || [];
  const trend = summary?.trend || [];
  const breakdown = summary?.type_breakdown || [];
  return (
    <div className="grid gap-4">
      <AdminMetricGrid metrics={metrics} />
      <div className="grid gap-4 2xl:grid-cols-[1.1fr_0.9fr]">
        <AdminLineChart
          title="财务趋势"
          description="近 14 天收入、支出和净额变化。"
          points={trend}
          series={financeSeries}
          valueSuffix="元"
        />
        <AdminFinanceTypeBreakdown rows={breakdown} />
      </div>
    </div>
  );
}

function AdminFinanceTypeBreakdown({ rows }: { rows: AdminFinanceTypeRow[] }) {
  return (
    <section className="rounded-md border bg-background p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold leading-6">财务类型分布</h3>
          <p className="text-sm leading-6 text-muted-foreground">
            按财务类型统计流水金额和记录数。
          </p>
        </div>
        <TrendingUp className="h-5 w-5 shrink-0 text-muted-foreground/70" />
      </div>
      <div className="mt-5 overflow-hidden rounded-md border">
        {rows.length === 0 ? (
          <AdminEmptyText>暂无财务流水数据</AdminEmptyText>
        ) : (
          <table className="w-full min-w-[560px] text-sm">
            <thead className="bg-muted/50 text-left text-muted-foreground">
              <tr>
                <th className="px-4 py-3 font-medium">财务类型</th>
                <th className="px-4 py-3 font-medium">方向</th>
                <th className="px-4 py-3 font-medium">金额</th>
                <th className="px-4 py-3 font-medium">流水数</th>
                <th className="px-4 py-3 font-medium">占比</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {rows.map((row) => (
                <tr key={textValue(row.key || row.name)}>
                  <td className="px-4 py-3 font-medium">{displayText(row.name)}</td>
                  <td className="px-4 py-3">{financeDirectionName(row.direction)}</td>
                  <td className="px-4 py-3">{formatNumber(row.amount)}</td>
                  <td className="px-4 py-3">{formatNumber(row.count)}</td>
                  <td className="px-4 py-3">{formatPercent(row.percent)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </section>
  );
}

function financeDirectionName(direction?: string): string {
  return textValue(direction) === "expense" ? "支出" : "收入";
}

function AdminCurveChart({
  title,
  description,
  points,
  series,
  valueSuffix = "个",
}: {
  title: string;
  description: string;
  points: AdminTrendPoint[];
  series: ChartSeries[];
  valueSuffix?: string;
}) {
  return (
    <AdminChartCard title={title} description={description}>
      {points.length === 0 ? (
        <AdminEmptyText>暂无曲线数据</AdminEmptyText>
      ) : (
        <AdminTrendEChart
          points={points}
          series={series}
          smooth
          valueSuffix={valueSuffix}
        />
      )}
    </AdminChartCard>
  );
}

function AdminLineChart({
  title,
  description,
  points,
  series,
  valueSuffix = "个",
}: {
  title: string;
  description: string;
  points: AdminTrendPoint[];
  series: ChartSeries[];
  valueSuffix?: string;
}) {
  return (
    <AdminChartCard title={title} description={description}>
      {points.length === 0 ? (
        <AdminEmptyText>暂无折线数据</AdminEmptyText>
      ) : (
        <AdminTrendEChart
          points={points}
          series={series}
          valueSuffix={valueSuffix}
        />
      )}
    </AdminChartCard>
  );
}

function AdminChartCard({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <section className="rounded-md border bg-background p-5 shadow-sm">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold leading-6">{title}</h3>
          <p className="text-sm leading-6 text-muted-foreground">{description}</p>
        </div>
      </div>
      <div className="mt-5">{children}</div>
    </section>
  );
}

function AdminTrendEChart({
  points,
  series,
  smooth = false,
  valueSuffix = "个",
}: {
  points: AdminTrendPoint[];
  series: ChartSeries[];
  smooth?: boolean;
  valueSuffix?: string;
}) {
  const option = useMemo(
    () => buildAdminTrendOption(points, series, smooth, valueSuffix),
    [points, series, smooth, valueSuffix],
  );
  return (
    <CrmEChart
      option={option}
      height={310}
      minWidth={720}
      ariaLabel="CRM 统计趋势"
    />
  );
}

function buildAdminTrendOption(
  points: AdminTrendPoint[],
  series: ChartSeries[],
  smooth: boolean,
  valueSuffix: string,
): EChartsOption {
  return {
    animationDuration: 280,
    color: series.map((item) => item.color),
    grid: {
      left: 8,
      right: 18,
      top: 42,
      bottom: 8,
      containLabel: true,
    },
    legend: {
      top: 0,
      right: 0,
      icon: "circle",
      itemWidth: 8,
      itemHeight: 8,
      textStyle: { color: crmChartTextColor, fontSize: 12 },
    },
    tooltip: {
      trigger: "axis",
      confine: true,
      borderColor: crmChartAxisColor,
      backgroundColor: "#ffffff",
      textStyle: { color: "#0f172a" },
      valueFormatter: (value) =>
        valueSuffix ? `${formatNumber(value)} ${valueSuffix}` : formatNumber(value),
    },
    xAxis: {
      type: "category",
      boundaryGap: false,
      data: points.map((point) => displayText(point.label || point.date, "")),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: crmChartAxisColor } },
      axisLabel: {
        color: crmChartTextColor,
        hideOverlap: true,
      },
    },
    yAxis: {
      type: "value",
      minInterval: 1,
      axisLabel: { color: crmChartTextColor },
      splitLine: { lineStyle: { color: crmChartSplitLineColor } },
    },
    series: series.map((item) => ({
      name: item.label,
      type: "line",
      smooth,
      symbol: "circle",
      symbolSize: 7,
      lineStyle: { width: smooth ? 3 : 2.6, color: item.color },
      itemStyle: { color: item.color },
      emphasis: { focus: "series" },
      data: points.map((point) => pointNumber(point, item.key)),
    })),
  };
}

function AdminFunnelChart({ rows }: { rows: AdminBreakdownRow[] }) {
  const option = useMemo(() => buildAdminFunnelOption(rows), [rows]);
  return (
    <section className="rounded-md border bg-background p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold leading-6">阶段漏斗</h3>
          <p className="text-sm leading-6 text-muted-foreground">
            当前客户或资产在各阶段的分布。
          </p>
        </div>
        <GitBranch className="h-5 w-5 shrink-0 text-muted-foreground/70" />
      </div>
      <div className="mt-5">
        {rows.length === 0 ? (
          <AdminEmptyText>暂无阶段漏斗数据</AdminEmptyText>
        ) : (
          <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
            <CrmEChart
              option={option}
              height={300}
              minWidth={420}
              ariaLabel="CRM 阶段漏斗"
            />
            <AdminFunnelSteps rows={rows} />
          </div>
        )}
      </div>
    </section>
  );
}

function AdminFunnelSteps({ rows }: { rows: AdminBreakdownRow[] }) {
  return (
    <div className="grid content-start gap-2">
      {rows.map((row, index) => (
        <div
          key={textValue(row.key || row.name) || String(index)}
          className="rounded-md border bg-muted/10 px-3 py-2"
        >
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0">
              <div className="truncate text-sm font-medium">
                {index + 1}. {displayText(row.name)}
              </div>
              <div className="mt-1 text-xs text-muted-foreground">
                占比 {formatPercent(row.percent)}
                {numberValue(row.drop_count) > 0
                  ? ` / 掉点 ${formatNumber(row.drop_count)}`
                  : ""}
              </div>
            </div>
            <div className="shrink-0 text-right">
              <div className="text-base font-semibold">{formatNumber(row.count)}</div>
              <div className="text-xs text-muted-foreground">对象</div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function buildAdminFunnelOption(rows: AdminBreakdownRow[]): EChartsOption {
  return {
    animationDuration: 280,
    color: ["#111827", "#2563eb", "#059669", "#d97706", "#dc2626", "#7c3aed"],
    tooltip: {
      trigger: "item",
      confine: true,
      borderColor: crmChartAxisColor,
      backgroundColor: "#ffffff",
      textStyle: { color: "#0f172a" },
      formatter: (params) => {
        const row = rows[Number((params as { dataIndex?: number }).dataIndex) || 0];
        return [
          displayText(row?.name),
          `数量：${formatNumber(row?.count)} 个`,
          `占比：${formatPercent(row?.percent)}`,
        ].join("<br/>");
      },
    },
    series: [
      {
        name: "阶段漏斗",
        type: "funnel",
        left: 16,
        right: 16,
        top: 8,
        bottom: 8,
        minSize: "28%",
        maxSize: "100%",
        sort: "none",
        gap: 3,
        label: {
          color: "#ffffff",
          formatter: "{b}  {c}",
          overflow: "truncate",
        },
        labelLine: { show: false },
        itemStyle: {
          borderColor: "#ffffff",
          borderWidth: 1,
        },
        data: rows.map((row) => ({
          name: displayText(row.name),
          value: numberValue(row.count),
        })),
      },
    ],
  };
}

function AdminBreakdownCard({
  title,
  description,
  rows,
  emptyText,
}: {
  title: string;
  description: string;
  rows: AdminBreakdownRow[];
  emptyText: string;
}) {
  return (
    <section className="rounded-md border bg-background p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold leading-6">{title}</h3>
          <p className="text-sm leading-6 text-muted-foreground">{description}</p>
        </div>
        <ClipboardList className="h-5 w-5 shrink-0 text-muted-foreground/70" />
      </div>
      <div className="mt-5">
        {rows.length === 0 ? (
          <AdminEmptyText>{emptyText}</AdminEmptyText>
        ) : (
          <AdminBreakdownChart rows={rows} />
        )}
      </div>
    </section>
  );
}

function AdminBreakdownChart({ rows }: { rows: AdminBreakdownRow[] }) {
  const option = useMemo(() => buildAdminBreakdownOption(rows), [rows]);
  return (
    <CrmEChart
      option={option}
      height={Math.max(220, rows.length * 42 + 72)}
      minWidth={520}
      ariaLabel="CRM 任务类型分布"
    />
  );
}

function buildAdminBreakdownOption(rows: AdminBreakdownRow[]): EChartsOption {
  return {
    animationDuration: 240,
    grid: {
      left: 8,
      right: 54,
      top: 8,
      bottom: 8,
      containLabel: true,
    },
    tooltip: {
      trigger: "item",
      confine: true,
      borderColor: crmChartAxisColor,
      backgroundColor: "#ffffff",
      textStyle: { color: "#0f172a" },
      formatter: (params) => {
        const row = rows[Number((params as { dataIndex?: number }).dataIndex) || 0];
        return [
          displayText(row?.name),
          `数量：${formatNumber(row?.count)} 个`,
          `占比：${formatPercent(row?.percent)}`,
        ].join("<br/>");
      },
    },
    xAxis: {
      type: "value",
      minInterval: 1,
      axisLabel: { color: crmChartTextColor },
      splitLine: { lineStyle: { color: crmChartSplitLineColor } },
    },
    yAxis: {
      type: "category",
      inverse: true,
      data: rows.map((row) => displayText(row.name)),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: crmChartAxisColor } },
      axisLabel: {
        color: crmChartTextColor,
        width: 110,
        overflow: "truncate",
      },
    },
    series: [
      {
        name: "数量",
        type: "bar",
        barWidth: 14,
        data: rows.map((row) => numberValue(row.count)),
        label: {
          show: true,
          position: "right",
          formatter: "{c} 个",
          color: crmChartTextColor,
        },
        itemStyle: {
          color: "#2563eb",
          borderRadius: [0, 6, 6, 0],
        },
      },
    ],
  };
}

function AdminNodeBacklog({ rows }: { rows: AdminBacklogRow[] }) {
  const option = useMemo(() => buildAdminBacklogOption(rows), [rows]);
  return (
    <section className="rounded-md border bg-background p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold leading-6">节点积压</h3>
          <p className="text-sm leading-6 text-muted-foreground">
            按当前阶段统计停留天数、协作待办和超期对象。
          </p>
        </div>
        <AlertTriangle className="h-5 w-5 shrink-0 text-muted-foreground/70" />
      </div>
      <div className="mt-5">
        {rows.length === 0 ? (
          <AdminEmptyText>暂无节点积压数据</AdminEmptyText>
        ) : (
          <div className="grid gap-4">
            <CrmEChart
              option={option}
              height={Math.max(260, rows.length * 42 + 90)}
              minWidth={560}
              ariaLabel="CRM 节点积压"
            />
            <div className="overflow-hidden rounded-md border">
              <table className="w-full min-w-[680px] text-sm">
                <thead className="bg-muted/50 text-left text-muted-foreground">
                  <tr>
                    <th className="px-4 py-3 font-medium">节点</th>
                    <th className="px-4 py-3 font-medium">积压</th>
                    <th className="px-4 py-3 font-medium">平均/最长</th>
                    <th className="px-4 py-3 font-medium">7天+</th>
                    <th className="px-4 py-3 font-medium">15天+</th>
                    <th className="px-4 py-3 font-medium">待办</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {rows.map((row) => (
                    <tr key={textValue(row.key || row.name)}>
                      <td className="px-4 py-3 font-medium">{displayText(row.name)}</td>
                      <td className="px-4 py-3">{formatNumber(row.count)}</td>
                      <td className="px-4 py-3">
                        {formatNumber(row.avg_days)} / {formatNumber(row.max_days)} 天
                      </td>
                      <td className="px-4 py-3">{formatNumber(row.stale_7d)}</td>
                      <td className="px-4 py-3">{formatNumber(row.stale_15d)}</td>
                      <td className="px-4 py-3">{formatNumber(row.pending_todo_count)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </section>
  );
}

function buildAdminBacklogOption(rows: AdminBacklogRow[]): EChartsOption {
  return {
    animationDuration: 240,
    color: ["#2563eb", "#d97706", "#dc2626"],
    grid: {
      left: 8,
      right: 36,
      top: 34,
      bottom: 8,
      containLabel: true,
    },
    legend: {
      top: 0,
      right: 0,
      icon: "circle",
      itemWidth: 8,
      itemHeight: 8,
      textStyle: { color: crmChartTextColor, fontSize: 12 },
    },
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "shadow" },
      confine: true,
      borderColor: crmChartAxisColor,
      backgroundColor: "#ffffff",
      textStyle: { color: "#0f172a" },
    },
    xAxis: {
      type: "value",
      minInterval: 1,
      axisLabel: { color: crmChartTextColor },
      splitLine: { lineStyle: { color: crmChartSplitLineColor } },
    },
    yAxis: {
      type: "category",
      inverse: true,
      data: rows.map((row) => displayText(row.name)),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: crmChartAxisColor } },
      axisLabel: {
        color: crmChartTextColor,
        width: 120,
        overflow: "truncate",
      },
    },
    series: [
      {
        name: "积压对象",
        type: "bar",
        barWidth: 12,
        data: rows.map((row) => numberValue(row.count)),
        itemStyle: { borderRadius: [0, 6, 6, 0] },
      },
      {
        name: "7天以上",
        type: "bar",
        barWidth: 12,
        data: rows.map((row) => numberValue(row.stale_7d)),
        itemStyle: { borderRadius: [0, 6, 6, 0] },
      },
      {
        name: "15天以上",
        type: "bar",
        barWidth: 12,
        data: rows.map((row) => numberValue(row.stale_15d)),
        itemStyle: { borderRadius: [0, 6, 6, 0] },
      },
    ],
  };
}

function AdminProbeSummaryCard({ summary }: { summary?: AdminProbeSummary }) {
  const dimensions = summary?.dimensions || [];
  const missing = summary?.missing_dimensions || [];
  return (
    <section className="rounded-md border bg-background p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold leading-6">十一维收集</h3>
          <p className="text-sm leading-6 text-muted-foreground">
            统计 P01-P12 资料填写完整度。
          </p>
        </div>
        <ShieldCheck className="h-5 w-5 shrink-0 text-muted-foreground/70" />
      </div>
      {!summary || numberValue(summary.field_total) === 0 ? (
        <div className="mt-5">
          <AdminEmptyText>暂无十一维模板或收集数据</AdminEmptyText>
        </div>
      ) : (
        <div className="mt-5 grid gap-4">
          <div className="grid gap-3 sm:grid-cols-3">
            <AdminMiniStat label="涉及资产" value={summary.started_asset_count} />
            <AdminMiniStat label="字段完整度" value={`${formatPercent(summary.percent)}`} />
            <AdminMiniStat label="完整资产" value={summary.complete_asset_count} />
          </div>
          <div className="grid gap-2">
            {dimensions.slice(0, 12).map((row) => (
              <AdminProbeDimensionProgress key={textValue(row.key || row.name)} row={row} />
            ))}
          </div>
          <div className="grid gap-3">
            <AdminProbeList title="缺失最多" rows={missing} valueKey="missing_count" />
          </div>
        </div>
      )}
    </section>
  );
}

function AdminMiniStat({ label, value }: { label: string; value: unknown }) {
  const text =
    typeof value === "string" && value.includes("%") ? value : formatNumber(value);
  return (
    <div className="rounded-md border bg-muted/10 px-3 py-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-xl font-semibold">{text}</div>
    </div>
  );
}

function AdminProbeDimensionProgress({ row }: { row: AdminProbeDimensionRow }) {
  const percent = numberValue(row.percent);
  return (
    <div className="grid gap-1.5">
      <div className="flex items-center justify-between gap-3 text-sm">
        <span className="min-w-0 truncate font-medium">{displayText(row.name)}</span>
        <span className="shrink-0 text-xs text-muted-foreground">
          {formatNumber(row.filled)} / {formatNumber(row.total)}
        </span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full bg-primary"
          style={{ width: `${Math.max(0, Math.min(100, percent))}%` }}
        />
      </div>
    </div>
  );
}

function AdminProbeList({
  title,
  rows,
  valueKey,
}: {
  title: string;
  rows: Array<AdminProbeDimensionRow | AdminBreakdownRow>;
  valueKey: "missing_count" | "count";
}) {
  return (
    <div className="rounded-md border bg-muted/10 p-3">
      <div className="mb-2 text-sm font-medium">{title}</div>
      {rows.length === 0 ? (
        <div className="text-xs leading-6 text-muted-foreground">暂无数据</div>
      ) : (
        <div className="grid gap-2">
          {rows.slice(0, 6).map((row) => (
            <div
              key={textValue(row.key || row.name)}
              className="flex items-center justify-between gap-3 text-sm"
            >
              <span className="min-w-0 truncate text-muted-foreground">
                {displayText(row.name)}
              </span>
              <span className="shrink-0 font-medium">
                {formatNumber((row as Record<string, unknown>)[valueKey])}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function AdminStaffRanking({ rows }: { rows: AdminStaffRow[] }) {
  return (
    <section className="rounded-md border bg-background p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold leading-6">人员产出</h3>
          <p className="text-sm leading-6 text-muted-foreground">
            近 14 天任务、流转、协作完成和积分排行。
          </p>
        </div>
        <ListChecks className="h-5 w-5 shrink-0 text-muted-foreground/70" />
      </div>
      <div className="mt-5 overflow-hidden rounded-md border">
        {rows.length === 0 ? (
          <AdminEmptyText>暂无人员产出数据</AdminEmptyText>
        ) : (
          <table className="w-full min-w-[920px] text-sm">
            <thead className="bg-muted/50 text-left text-muted-foreground">
              <tr>
                <th className="px-4 py-3 font-medium">人员</th>
                <th className="px-4 py-3 font-medium">任务完成</th>
                <th className="px-4 py-3 font-medium">阶段流转</th>
                <th className="px-4 py-3 font-medium">操作记录</th>
                <th className="px-4 py-3 font-medium">协作完成</th>
                <th className="px-4 py-3 font-medium">积分</th>
                <th className="px-4 py-3 font-medium">合计</th>
                <th className="px-4 py-3 font-medium">最近产出</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {rows.map((row) => (
                <tr key={textValue(row.id || row.name)}>
                  <td className="px-4 py-3 font-medium">{displayText(row.name)}</td>
                  <td className="px-4 py-3">{formatNumber(row.task_count)}</td>
                  <td className="px-4 py-3">{formatNumber(row.transition_count)}</td>
                  <td className="px-4 py-3">{formatNumber(row.operation_count)}</td>
                  <td className="px-4 py-3">{formatNumber(row.todo_done_count)}</td>
                  <td className="px-4 py-3">{formatNumber(row.points)}</td>
                  <td className="px-4 py-3 font-semibold">{formatNumber(row.total)}</td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {formatWorkDate(row.last_active_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </section>
  );
}

function AdminStatsState({
  icon,
  title,
  description,
}: {
  icon: "loading" | "empty";
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col items-center justify-center text-center">
      {icon === "loading" ? (
        <Loader2 className="mb-3 h-8 w-8 animate-spin text-muted-foreground" />
      ) : (
        <TrendingUp className="mb-3 h-8 w-8 text-muted-foreground" />
      )}
      <div className="text-base font-semibold">{title}</div>
      <div className="mt-1 text-sm text-muted-foreground">{description}</div>
    </div>
  );
}

function AdminEmptyText({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-md border border-dashed bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">
      {children}
    </div>
  );
}

function pointNumber(point: AdminTrendPoint, key: ChartSeriesKey): number {
  return numberValue(point[key]);
}

function numberValue(value: unknown): number {
  const number = Number(value);
  return Number.isFinite(number) ? number : 0;
}

function formatNumber(value: unknown): string {
  const number = numberValue(value);
  return new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 0 }).format(number);
}

function formatPercent(value: unknown): string {
  return `${Math.max(0, Math.min(100, Math.round(numberValue(value))))}%`;
}
