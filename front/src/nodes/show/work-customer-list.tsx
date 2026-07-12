import { useState } from "react";
import type { FormEvent, ReactNode } from "react";
import {
  ClipboardList,
  Ellipsis,
  Eye,
  Inbox,
  RefreshCw,
  Search,
  ShieldCheck,
  SlidersHorizontal,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

import {
  displayText,
  type WorkCustomerMode,
  type WorkCustomerScope,
  type WorkItem,
  type WorkSearchFilters,
  type WorkStageOption,
  type WorkTask,
} from "./work-core";

export type WorkCustomerListTaskView = {
  key: string;
  label: string;
  result?: string;
  kind: "action" | "rule";
  task: WorkTask;
};

export type WorkCustomerListRowView = {
  id: string;
  item: WorkItem;
  customerName: string;
  customerNo: string;
  phone: string;
  wechat: string;
  assetName: string;
  assetNo: string;
  assetStatus: string;
  stageName: string;
  hasStage: boolean;
  ownerName: string;
  stageDays: number;
  lastOperatedAt: string;
  tasks: WorkCustomerListTaskView[];
};

type WorkCustomerListViewProps = {
  rows: WorkCustomerListRowView[];
  loading: boolean;
  mode: WorkCustomerMode;
  modeCounts: Record<WorkCustomerMode, number>;
  scope: WorkCustomerScope;
  canDispatch: boolean;
  filters: WorkSearchFilters;
  stageFilter: string;
  stageOptions: WorkStageOption[];
  page: number;
  pageSize: number;
  total: number;
  emptyTitle: string;
  emptyDescription: string;
  onFiltersChange: (filters: WorkSearchFilters) => void;
  onSearch: () => void;
  onReset: () => void;
  onModeChange: (mode: WorkCustomerMode) => void;
  onScopeChange: (scope: WorkCustomerScope) => void;
  onStageChange: (stage: string) => void;
  onPageChange: (page: number) => void;
  onRefresh: () => void;
  onOpenDetail: (row: WorkCustomerListRowView) => void;
  onOpenTask: (
    row: WorkCustomerListRowView,
    task: WorkCustomerListTaskView,
  ) => void;
};

const modeOptions: Array<{ value: WorkCustomerMode; label: string }> = [
  { value: "all", label: "全部" },
  { value: "pending", label: "待处理" },
  { value: "done", label: "已结束" },
];

const scopeOptions: Array<{ value: WorkCustomerScope; label: string }> = [
  { value: "mine", label: "我的" },
  { value: "all", label: "全部" },
];

const exactFilterFields: Array<{
  key: keyof Pick<
    WorkSearchFilters,
    "customerNo" | "customerName" | "phone" | "wechat" | "assetNo"
  >;
  placeholder: string;
}> = [
  { key: "customerNo", placeholder: "客户编号" },
  { key: "customerName", placeholder: "姓名" },
  { key: "phone", placeholder: "手机号" },
  { key: "wechat", placeholder: "微信号" },
  { key: "assetNo", placeholder: "资产编号" },
];

export function WorkCustomerListView({
  rows,
  loading,
  mode,
  modeCounts,
  scope,
  canDispatch,
  filters,
  stageFilter,
  stageOptions,
  page,
  pageSize,
  total,
  emptyTitle,
  emptyDescription,
  onFiltersChange,
  onSearch,
  onReset,
  onModeChange,
  onScopeChange,
  onStageChange,
  onPageChange,
  onRefresh,
  onOpenDetail,
  onOpenTask,
}: WorkCustomerListViewProps) {
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const initialLoading = loading && rows.length === 0;

  const submitSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onSearch();
  };

  return (
    <div className="grid gap-3">
      <WorkCustomerListStyles />
      <div className="flex flex-wrap items-center gap-3">
        <WorkCustomerModeTabs
          mode={mode}
          counts={modeCounts}
          onChange={onModeChange}
        />
        <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
          {canDispatch ? (
            <WorkCustomerScopeToggle scope={scope} onChange={onScopeChange} />
          ) : null}
          <Button
            type="button"
            variant="outline"
            size="icon"
            aria-label="刷新"
            title="刷新"
            onClick={onRefresh}
            disabled={loading}
          >
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
        </div>
      </div>

      <div className="overflow-hidden rounded-md border border-border/70 bg-background">
        <form
          onSubmit={submitSearch}
          className="border-b border-border/70 bg-muted/10 px-4 py-3"
        >
          <div className="crm-customer-list-search-grid grid gap-2.5">
            <label className="min-w-0">
              <span className="sr-only">综合搜索</span>
              <Input
                className="w-full"
                value={filters.keyword}
                onChange={(event) =>
                  onFiltersChange({
                    ...filters,
                    keyword: event.currentTarget.value,
                  })
                }
                placeholder="搜索姓名、手机、微信、客户或资产编号"
              />
            </label>
            <label>
              <span className="sr-only">当前阶段</span>
              <select
                className="h-9 w-full rounded-md border border-input bg-background px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/20"
                value={stageFilter}
                onChange={(event) => onStageChange(event.currentTarget.value)}
              >
                <option value="">全部阶段</option>
                {stageOptions.map((option) => (
                  <option key={option.id} value={option.id}>
                    {option.value}
                  </option>
                ))}
              </select>
            </label>
            <Button type="submit" size="sm" disabled={loading}>
              <Search className="h-4 w-4" />
              搜索
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              aria-expanded={advancedOpen}
              onClick={() => setAdvancedOpen((open) => !open)}
            >
              <SlidersHorizontal className="h-4 w-4" />
              更多筛选
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onReset}
              disabled={loading}
            >
              <RefreshCw className="h-4 w-4" />
              重置
            </Button>
          </div>

          {advancedOpen ? (
            <div className="crm-customer-list-advanced-grid mt-3 grid gap-2.5 border-t border-border/60 pt-3">
              {exactFilterFields.map((field) => (
                <label key={field.key}>
                  <span className="sr-only">{field.placeholder}</span>
                  <Input
                    className="w-full"
                    value={filters[field.key]}
                    placeholder={field.placeholder}
                    onChange={(event) =>
                      onFiltersChange({
                        ...filters,
                        [field.key]: event.currentTarget.value,
                      })
                    }
                  />
                </label>
              ))}
            </div>
          ) : null}
        </form>

        <div className="p-3 md:hidden">
          <WorkCustomerMobileList
            rows={rows}
            loading={initialLoading}
            emptyTitle={emptyTitle}
            emptyDescription={emptyDescription}
            onOpenDetail={onOpenDetail}
            onOpenTask={onOpenTask}
          />
        </div>

        <div className="hidden overflow-x-auto md:block">
          <table className="crm-customer-list-table w-full table-fixed border-collapse text-sm">
            <colgroup>
              <col style={{ width: 288 }} />
              <col style={{ width: 320 }} />
              <col style={{ width: 240 }} />
              <col style={{ width: 288 }} />
              <col style={{ width: 160 }} />
            </colgroup>
            <thead className="bg-muted/20">
              <tr className="border-b border-border/70">
                <WorkCustomerTableHead>客户</WorkCustomerTableHead>
                <WorkCustomerTableHead>房产/资产</WorkCustomerTableHead>
                <WorkCustomerTableHead>流程阶段</WorkCustomerTableHead>
                <WorkCustomerTableHead>当前待办</WorkCustomerTableHead>
                <WorkCustomerTableHead className="text-center">
                  操作
                </WorkCustomerTableHead>
              </tr>
            </thead>
            <tbody>
              {initialLoading ? (
                <WorkCustomerTableState title="正在加载" description="正在同步客户数据" />
              ) : rows.length === 0 ? (
                <WorkCustomerTableState
                  title={emptyTitle}
                  description={emptyDescription}
                />
              ) : (
                rows.map((row) => (
                  <WorkCustomerTableRow
                    key={row.id}
                    row={row}
                    onOpenDetail={onOpenDetail}
                    onOpenTask={onOpenTask}
                  />
                ))
              )}
            </tbody>
          </table>
        </div>

        <WorkCustomerPagination
          loading={loading}
          hidden={initialLoading}
          page={page}
          pageSize={pageSize}
          total={total}
          onPageChange={onPageChange}
        />
      </div>
    </div>
  );
}

function WorkCustomerListStyles() {
  return (
    <style>{`
      .crm-customer-list-search-grid {
        grid-template-columns: minmax(280px, 1fr) 180px auto auto auto;
      }

      .crm-customer-list-advanced-grid {
        grid-template-columns: repeat(5, minmax(0, 1fr));
      }

      .crm-customer-list-table {
        min-width: 1296px;
      }

      .crm-customer-list-row {
        height: 84px;
      }

      .crm-customer-task-menu {
        width: 224px;
      }

      .crm-customer-task-menu-trigger::-webkit-details-marker {
        display: none;
      }

      .crm-customer-two-line {
        display: -webkit-box;
        overflow: hidden;
        -webkit-box-orient: vertical;
        -webkit-line-clamp: 2;
      }

      @media (max-width: 1180px) {
        .crm-customer-list-search-grid {
          grid-template-columns: minmax(240px, 1fr) 170px auto auto;
        }

        .crm-customer-list-advanced-grid {
          grid-template-columns: repeat(3, minmax(0, 1fr));
        }
      }

      @media (max-width: 767px) {
        .crm-customer-list-search-grid {
          grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
        }

        .crm-customer-list-search-grid > :first-child {
          grid-column: 1 / -1;
        }

        .crm-customer-list-search-grid > :last-child {
          grid-column: auto;
        }

        .crm-customer-list-advanced-grid {
          grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
        }
      }
    `}</style>
  );
}

function WorkCustomerModeTabs({
  mode,
  counts,
  onChange,
}: {
  mode: WorkCustomerMode;
  counts: Record<WorkCustomerMode, number>;
  onChange: (mode: WorkCustomerMode) => void;
}) {
  return (
    <div className="inline-flex rounded-md border border-border/60 bg-muted/25 p-1">
      {modeOptions.map((option) => (
        <button
          type="button"
          key={option.value}
          className={`rounded px-3 py-1.5 text-sm font-medium transition-colors ${
            mode === option.value
              ? "bg-background text-foreground shadow-sm ring-1 ring-border/50"
              : "text-muted-foreground hover:text-foreground"
          }`}
          onClick={() => onChange(option.value)}
        >
          {option.label}
          <span className="ml-1 text-xs text-muted-foreground">
            {counts[option.value] || 0}
          </span>
        </button>
      ))}
    </div>
  );
}

function WorkCustomerScopeToggle({
  scope,
  onChange,
}: {
  scope: WorkCustomerScope;
  onChange: (scope: WorkCustomerScope) => void;
}) {
  return (
    <div className="inline-flex rounded-md border border-border/60 bg-muted/25 p-1">
      {scopeOptions.map((option) => (
          <button
            type="button"
            key={option.value}
            className={`rounded px-2.5 py-1 text-xs font-medium transition-colors ${
              scope === option.value
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => onChange(option.value)}
          >
            {option.label}
          </button>
        ))}
    </div>
  );
}

function WorkCustomerTableHead({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <th className={`h-11 px-4 text-left text-xs font-medium text-muted-foreground ${className}`}>
      {children}
    </th>
  );
}

function WorkCustomerTableState({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <tr>
      <td colSpan={5} className="px-6 py-16">
        <WorkCustomerEmpty title={title} description={description} />
      </td>
    </tr>
  );
}

function WorkCustomerTableRow({
  row,
  onOpenDetail,
  onOpenTask,
}: {
  row: WorkCustomerListRowView;
  onOpenDetail: (row: WorkCustomerListRowView) => void;
  onOpenTask: (
    row: WorkCustomerListRowView,
    task: WorkCustomerListTaskView,
  ) => void;
}) {
  const actionableTasks = row.tasks.filter((task) => task.kind === "action");
  const primaryTask = actionableTasks[0];
  const ruleTask = row.tasks.find((task) => task.kind === "rule");
  const extraTasks = actionableTasks.slice(1);

  return (
    <tr
      className="crm-customer-list-row border-b border-border/60 bg-background transition-colors hover:bg-muted/20 last:border-b-0"
      onClick={() => onOpenDetail(row)}
    >
      <td className="px-4 py-3 align-middle">
        <button
          type="button"
          className="block w-full min-w-0 text-left"
          onClick={(event) => {
            event.stopPropagation();
            onOpenDetail(row);
          }}
        >
          <span className="block truncate font-semibold text-foreground">
            {row.customerName}
          </span>
          <span className="mt-1 block truncate text-xs text-muted-foreground">
            {row.customerNo}
          </span>
          <span className="mt-1 block truncate text-xs text-muted-foreground">
            {row.phone}
            {row.wechat !== "-" ? ` / ${row.wechat}` : ""}
          </span>
        </button>
      </td>
      <td className="px-4 py-3 align-middle">
        <div className="crm-customer-two-line font-medium text-foreground">
          {row.assetName}
        </div>
        <div className="mt-1 truncate text-xs text-muted-foreground">
          {row.assetNo}
        </div>
        {row.assetStatus !== "-" ? (
          <div className="mt-1 truncate text-xs text-muted-foreground">
            {row.assetStatus}
          </div>
        ) : null}
      </td>
      <td className="px-4 py-3 align-middle">
        <span className="inline-flex rounded bg-muted px-2 py-1 text-xs font-medium text-foreground">
          {row.stageName}
        </span>
        {row.hasStage ? (
          <>
            <div className="mt-1.5 truncate text-xs text-muted-foreground">
              {row.ownerName !== "-" ? row.ownerName : "暂未分配负责人"}
            </div>
            <div className="mt-1 text-xs text-muted-foreground">
              {row.stageDays > 0 ? `停留 ${row.stageDays} 天` : "今日进入"}
            </div>
          </>
        ) : (
          <div className="mt-1.5 text-xs text-muted-foreground">
            尚未进入流程
          </div>
        )}
      </td>
      <td className="px-4 py-3 align-middle">
        <WorkCustomerTaskSummary
          primaryTask={primaryTask}
          ruleTask={ruleTask}
          total={row.tasks.length}
        />
      </td>
      <td className="px-4 py-3 align-middle">
        <div
          className="flex items-center justify-center gap-1.5"
          onClick={(event) => event.stopPropagation()}
        >
          {primaryTask ? (
            <Button type="button" size="sm" onClick={() => onOpenTask(row, primaryTask)}>
              <ClipboardList className="h-4 w-4" />
              处理
            </Button>
          ) : (
            <Button type="button" variant="outline" size="sm" onClick={() => onOpenDetail(row)}>
              <Eye className="h-4 w-4" />
              查看
            </Button>
          )}
          <WorkCustomerTaskMenu
            row={row}
            tasks={extraTasks}
            onOpenTask={onOpenTask}
          />
        </div>
      </td>
    </tr>
  );
}

function WorkCustomerTaskSummary({
  primaryTask,
  ruleTask,
  total,
}: {
  primaryTask?: WorkCustomerListTaskView;
  ruleTask?: WorkCustomerListTaskView;
  total: number;
}) {
  if (primaryTask) {
    return (
      <div className="min-w-0">
        <div className="crm-customer-two-line font-medium text-foreground">
          {primaryTask.label}
        </div>
        <div className="mt-1 text-xs text-muted-foreground">
          {total > 1 ? `共 ${total} 项待办` : "等待处理"}
        </div>
      </div>
    );
  }
  if (ruleTask) {
    return (
      <div className="flex min-w-0 items-start gap-2 text-amber-800">
        <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0" />
        <div className="min-w-0">
          <div className="truncate text-sm font-medium">{ruleTask.label}</div>
          <div className="mt-0.5 crm-customer-two-line text-xs opacity-80">
            {displayText(ruleTask.result, "等待核验条件")}
          </div>
        </div>
      </div>
    );
  }
  return <span className="text-sm text-muted-foreground">暂无待办</span>;
}

function WorkCustomerTaskMenu({
  row,
  tasks,
  onOpenTask,
}: {
  row: WorkCustomerListRowView;
  tasks: WorkCustomerListTaskView[];
  onOpenTask: (
    row: WorkCustomerListRowView,
    task: WorkCustomerListTaskView,
  ) => void;
}) {
  if (tasks.length === 0) return null;
  return (
    <details className="relative">
      <summary
        className="crm-customer-task-menu-trigger inline-flex h-8 w-8 cursor-pointer list-none items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        aria-label={`还有 ${tasks.length} 项任务`}
        title={`还有 ${tasks.length} 项任务`}
      >
        <Ellipsis className="h-4 w-4" />
      </summary>
      <div className="crm-customer-task-menu absolute right-0 top-10 z-50 overflow-hidden rounded-md border border-border bg-background p-1 shadow-lg">
        {tasks.map((task) => (
          <button
            type="button"
            key={task.key}
            className="block w-full rounded px-3 py-2 text-left text-sm hover:bg-muted"
            onClick={() => onOpenTask(row, task)}
          >
            <span className="block truncate font-medium">{task.label}</span>
            {task.result ? (
              <span className="mt-0.5 block truncate text-xs text-muted-foreground">
                {task.result}
              </span>
            ) : null}
          </button>
        ))}
      </div>
    </details>
  );
}

function WorkCustomerMobileList({
  rows,
  loading,
  emptyTitle,
  emptyDescription,
  onOpenDetail,
  onOpenTask,
}: {
  rows: WorkCustomerListRowView[];
  loading: boolean;
  emptyTitle: string;
  emptyDescription: string;
  onOpenDetail: (row: WorkCustomerListRowView) => void;
  onOpenTask: (
    row: WorkCustomerListRowView,
    task: WorkCustomerListTaskView,
  ) => void;
}) {
  if (loading) {
    return <WorkCustomerEmpty title="正在加载" description="正在同步客户数据" />;
  }
  if (rows.length === 0) {
    return <WorkCustomerEmpty title={emptyTitle} description={emptyDescription} />;
  }
  return (
    <div className="grid gap-3">
      {rows.map((row) => {
        const actionableTasks = row.tasks.filter((task) => task.kind === "action");
        const primaryTask = actionableTasks[0];
        return (
          <article key={row.id} className="rounded-md border border-border/70 bg-background p-4">
            <button type="button" className="block w-full text-left" onClick={() => onOpenDetail(row)}>
              <div className="flex min-w-0 items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="truncate font-semibold">{row.customerName}</div>
                  <div className="mt-1 truncate text-xs text-muted-foreground">
                    {row.customerNo} / {row.phone}
                  </div>
                </div>
                <span className="shrink-0 rounded bg-muted px-2 py-1 text-xs font-medium">
                  {row.stageName}
                </span>
              </div>
              <div className="mt-3 border-t border-border/60 pt-3">
                <div className="truncate text-sm font-medium">{row.assetName}</div>
                <div className="mt-1 truncate text-xs text-muted-foreground">
                  {row.assetNo}
                </div>
              </div>
              <div className="mt-3 text-sm text-muted-foreground">
                {primaryTask ? primaryTask.label : "暂无待办"}
              </div>
            </button>
            <div className="mt-3 flex items-center gap-2 border-t border-border/60 pt-3">
              {primaryTask ? (
                <Button type="button" size="sm" className="flex-1" onClick={() => onOpenTask(row, primaryTask)}>
                  <ClipboardList className="h-4 w-4" />
                  处理
                </Button>
              ) : (
                <Button type="button" variant="outline" size="sm" className="flex-1" onClick={() => onOpenDetail(row)}>
                  <Eye className="h-4 w-4" />
                  查看
                </Button>
              )}
              <WorkCustomerTaskMenu
                row={row}
                tasks={actionableTasks.slice(1)}
                onOpenTask={onOpenTask}
              />
            </div>
          </article>
        );
      })}
    </div>
  );
}

function WorkCustomerEmpty({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="flex min-h-40 flex-col items-center justify-center px-5 py-10 text-center">
      <Inbox className="h-5 w-5 text-muted-foreground" />
      <div className="mt-2 text-sm font-medium text-foreground">{title}</div>
      <div className="mt-1 text-xs leading-5 text-muted-foreground">
        {description}
      </div>
    </div>
  );
}

function WorkCustomerPagination({
  loading,
  hidden,
  page,
  pageSize,
  total,
  onPageChange,
}: {
  loading: boolean;
  hidden: boolean;
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
}) {
  if (hidden || total <= 0) return null;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const currentPage = Math.min(totalPages, Math.max(1, page));
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border/70 px-4 py-3 text-xs text-muted-foreground">
      <span>
        第 {currentPage} / {totalPages} 页，共 {total} 条
      </span>
      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={loading || currentPage <= 1}
          onClick={() => onPageChange(currentPage - 1)}
        >
          上一页
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={loading || currentPage >= totalPages}
          onClick={() => onPageChange(currentPage + 1)}
        >
          下一页
        </Button>
      </div>
    </div>
  );
}
