import { useState } from "react";
import type { ReactNode } from "react";
import {
  ClipboardList,
  Eye,
  RefreshCw,
  ShieldCheck,
  UserRound,
} from "lucide-react";

import { Button } from "@/components/ui/button";

import {
  displayText,
  formatWorkDate,
  type WorkCustomerMode,
  type WorkCustomerScope,
  type WorkFlowDetail,
  type WorkItem,
  type WorkTask,
} from "./work-core";
import { WorkFlowModeTabs } from "./work-flow-mode-tabs";
import { WorkFlowOwnerDialog } from "./work-flow-owner-dialog";
import { WorkListState } from "./work-list-state";
import { WorkPagination } from "./work-pagination";

export type WorkCustomerListTaskView = {
  key: string;
  label: string;
  result?: string;
  kind: "action" | "rule";
  canOperate: boolean;
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
  processedTaskName: string;
  processedResult: string;
  processedContent: string;
  processedAt: string;
  flow: WorkFlowDetail | null;
  tasks: WorkCustomerListTaskView[];
};

type WorkCustomerListViewProps = {
  rows: WorkCustomerListRowView[];
  loading: boolean;
  mode: WorkCustomerMode;
  modeCounts: Record<WorkCustomerMode, number>;
  scope: WorkCustomerScope;
  canDispatch: boolean;
  page: number;
  pageSize: number;
  total: number;
  emptyTitle: string;
  emptyDescription: string;
  onModeChange: (mode: WorkCustomerMode) => void;
  onScopeChange: (scope: WorkCustomerScope) => void;
  onPageChange: (page: number) => void;
  onRefresh: () => void;
  onOpenDetail: (row: WorkCustomerListRowView) => void;
  onOpenTask: (
    row: WorkCustomerListRowView,
    task: WorkCustomerListTaskView,
  ) => void;
};

const scopeOptions: Array<{ value: WorkCustomerScope; label: string }> = [
  { value: "mine", label: "我的" },
  { value: "all", label: "全部" },
];

export function WorkCustomerListView({
  rows,
  loading,
  mode,
  modeCounts,
  scope,
  canDispatch,
  page,
  pageSize,
  total,
  emptyTitle,
  emptyDescription,
  onModeChange,
  onScopeChange,
  onPageChange,
  onRefresh,
  onOpenDetail,
  onOpenTask,
}: WorkCustomerListViewProps) {
  const [reassignFlow, setReassignFlow] = useState<WorkFlowDetail | null>(null);
  const initialLoading = loading && rows.length === 0;

  return (
    <div className="crm-work-customer-list space-y-4">
      <WorkCustomerListStyles />
      <div className="flex flex-wrap items-center gap-3">
        <WorkFlowModeTabs
          mode={mode}
          counts={modeCounts}
          onChange={onModeChange}
        />
        <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
          {canDispatch && mode !== "processed" ? (
            <WorkCustomerScopeToggle scope={scope} onChange={onScopeChange} />
          ) : null}
          <Button
            type="button"
            variant="ghost"
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

      <section className="overflow-hidden bg-background">
        <div className="md:hidden">
          <WorkCustomerMobileList
            rows={rows}
            mode={mode}
            loading={initialLoading}
            emptyTitle={emptyTitle}
            emptyDescription={emptyDescription}
            onOpenDetail={onOpenDetail}
            onOpenTask={onOpenTask}
            onOpenReassign={(row) => setReassignFlow(row.flow)}
          />
        </div>

        <div className="hidden overflow-x-auto md:block">
          <table className="crm-customer-list-table w-full table-fixed border-collapse">
            <colgroup>
              <col style={{ width: "22%" }} />
              <col style={{ width: "24%" }} />
              <col style={{ width: "16%" }} />
              <col style={{ width: "20%" }} />
              <col style={{ width: "18%" }} />
            </colgroup>
            <thead className="crm-customer-list-table-head text-left">
              <tr>
                <WorkCustomerTableHead>客户</WorkCustomerTableHead>
                <WorkCustomerTableHead>房产/资产</WorkCustomerTableHead>
                <WorkCustomerTableHead>流程阶段</WorkCustomerTableHead>
                <WorkCustomerTableHead>
                  {mode === "processed" ? "最近处理" : "当前待办"}
                </WorkCustomerTableHead>
                <WorkCustomerTableHead className="text-right">
                  操作
                </WorkCustomerTableHead>
              </tr>
            </thead>
            <tbody>
              {initialLoading ? (
                <WorkCustomerTableState
                  loading
                  title="正在加载"
                  description="正在同步客户数据"
                />
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
                    mode={mode}
                    onOpenDetail={onOpenDetail}
                    onOpenTask={onOpenTask}
                    onOpenReassign={(row) => setReassignFlow(row.flow)}
                  />
                ))
              )}
            </tbody>
          </table>
        </div>

        <WorkPagination
          loading={loading}
          hidden={initialLoading}
          page={page}
          pageSize={pageSize}
          total={total}
          onPageChange={onPageChange}
        />
      </section>
      <WorkFlowOwnerDialog
        flow={reassignFlow}
        open={Boolean(reassignFlow)}
        title="改派负责人"
        confirmLabel="确认改派"
        onOpenChange={(open) => {
          if (!open) setReassignFlow(null);
        }}
      />
    </div>
  );
}

function WorkCustomerListStyles() {
  return (
    <style>{`
      .crm-customer-list-table {
        min-width: 1120px;
      }

      .crm-work-customer-list {
        color: var(--crm-body-text, #171a19);
        font-size: 12.8px;
        line-height: 1.45;
      }

      .crm-work-customer-list button,
      .crm-work-customer-list input,
      .crm-work-customer-list select,
      .crm-work-customer-list textarea {
        font-size: 12.8px;
      }

      .crm-customer-list-table-head {
        background: var(--crm-body-bg, #f4f6f5);
      }

      .crm-customer-list-table-head {
        color: var(--crm-body-muted, #6b7370);
      }

      .crm-customer-list-table-head tr,
      .crm-customer-list-row,
      .crm-customer-list-mobile-row {
        border-bottom: 1px solid var(--crm-body-line, #e4e8e6);
      }

      .crm-customer-list-row,
      .crm-customer-list-mobile-row {
        transition: background-color 120ms ease;
      }

      .crm-customer-list-row:hover,
      .crm-customer-list-mobile-row:hover {
        background: var(--crm-body-bg, #f4f6f5);
      }

      .crm-customer-list-row:last-child,
      .crm-customer-list-mobile-row:last-child {
        border-bottom: 0;
      }

      .crm-customer-two-line {
        display: -webkit-box;
        overflow: hidden;
        -webkit-box-orient: vertical;
        -webkit-line-clamp: 2;
      }
    `}</style>
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
    <div className="inline-flex items-center gap-1 rounded-md bg-muted/40 p-1">
      {scopeOptions.map((option) => (
        <Button
          type="button"
          key={option.value}
          variant="ghost"
          aria-pressed={scope === option.value}
          className={`h-auto rounded px-2.5 py-1 text-xs font-medium ${
            scope === option.value
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
          onClick={() => onChange(option.value)}
        >
          {option.label}
        </Button>
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
    <th className={`px-3 py-2.5 text-left font-medium ${className}`}>
      {children}
    </th>
  );
}

function WorkCustomerTableState({
  title,
  description,
  loading = false,
}: {
  title: string;
  description: string;
  loading?: boolean;
}) {
  return (
    <tr>
      <td colSpan={5} className="p-0">
        <WorkListState
          loading={loading}
          title={title}
          description={description}
        />
      </td>
    </tr>
  );
}

function WorkCustomerTableRow({
  row,
  mode,
  onOpenDetail,
  onOpenTask,
  onOpenReassign,
}: {
  row: WorkCustomerListRowView;
  mode: WorkCustomerMode;
  onOpenDetail: (row: WorkCustomerListRowView) => void;
  onOpenTask: (
    row: WorkCustomerListRowView,
    task: WorkCustomerListTaskView,
  ) => void;
  onOpenReassign: (row: WorkCustomerListRowView) => void;
}) {
  const { summaryTask, primaryTask, ruleTask, extraTasks } =
    workCustomerTaskGroups(row);

  return (
    <tr
      className="crm-customer-list-row align-top"
      onClick={() => onOpenDetail(row)}
    >
      <td className="px-3 py-3">
        <Button
          type="button"
          variant="ghost"
          className="h-auto w-full min-w-0 flex-col items-stretch gap-0 px-0 py-0 text-left hover:bg-transparent"
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
        </Button>
      </td>
      <td className="px-3 py-3">
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
      <td className="px-3 py-3">
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
      <td className="px-3 py-3">
        <WorkCustomerTaskSummary
          row={row}
          mode={mode}
          primaryTask={summaryTask}
          ruleTask={ruleTask}
          total={row.tasks.length}
        />
      </td>
      <td className="px-3 py-3">
        <WorkCustomerRowActions
          row={row}
          primaryTask={primaryTask}
          extraTasks={extraTasks}
          onOpenDetail={onOpenDetail}
          onOpenTask={onOpenTask}
          onOpenReassign={onOpenReassign}
          allowReassign={mode !== "processed"}
        />
      </td>
    </tr>
  );
}

function WorkCustomerTaskSummary({
  row,
  mode,
  primaryTask,
  ruleTask,
  total,
}: {
  row: WorkCustomerListRowView;
  mode: WorkCustomerMode;
  primaryTask?: WorkCustomerListTaskView;
  ruleTask?: WorkCustomerListTaskView;
  total: number;
}) {
  if (mode === "processed" && row.processedTaskName) {
    return (
      <div className="min-w-0">
        <div className="crm-customer-two-line font-medium text-foreground">
          {row.processedTaskName}
        </div>
        <div className="mt-1 crm-customer-two-line text-xs text-muted-foreground">
          {displayText(row.processedContent || row.processedResult, "已完成")}
          <span className="mx-1.5">·</span>
          {formatWorkDate(row.processedAt)}
        </div>
      </div>
    );
  }
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
          <div className="truncate font-medium">{ruleTask.label}</div>
          <div className="mt-0.5 crm-customer-two-line text-xs opacity-80">
            {displayText(ruleTask.result, "等待核验条件")}
          </div>
        </div>
      </div>
    );
  }
  return <span className="text-muted-foreground">暂无待办</span>;
}

function workCustomerTaskGroups(row: WorkCustomerListRowView) {
  const actionableTasks = row.tasks.filter((task) => task.kind === "action");
  const operableTasks = actionableTasks.filter((task) => task.canOperate);
  return {
    summaryTask: actionableTasks[0],
    primaryTask: operableTasks[0],
    extraTasks: operableTasks.slice(1),
    ruleTask: row.tasks.find((task) => task.kind === "rule"),
  };
}

function WorkCustomerRowActions({
  row,
  primaryTask,
  extraTasks,
  mobile = false,
  allowReassign = true,
  onOpenDetail,
  onOpenTask,
  onOpenReassign,
}: {
  row: WorkCustomerListRowView;
  primaryTask?: WorkCustomerListTaskView;
  extraTasks: WorkCustomerListTaskView[];
  mobile?: boolean;
  allowReassign?: boolean;
  onOpenDetail: (row: WorkCustomerListRowView) => void;
  onOpenTask: (
    row: WorkCustomerListRowView,
    task: WorkCustomerListTaskView,
  ) => void;
  onOpenReassign: (row: WorkCustomerListRowView) => void;
}) {
  const canReassign = allowReassign && Boolean(row.flow?.can_change_owner);
  const taskButtons = primaryTask ? [primaryTask, ...extraTasks] : extraTasks;
  const taskClassName = mobile
    ? "min-h-11 min-w-[8.5rem] flex-1 px-3"
    : "min-w-0 max-w-[9rem]";
  const mobileDetailClassName = taskButtons.length > 0
    ? "min-h-11 shrink-0 px-3"
    : canReassign
      ? "min-h-11 min-w-0 flex-1 px-3"
      : "min-h-11 w-full px-3";
  const detailClassName = mobile ? mobileDetailClassName : "shrink-0";

  return (
    <div
      className={`crm-customer-row-actions flex flex-wrap items-center justify-end gap-1.5 ${mobile ? "w-full" : ""}`}
      onClick={(event) => event.stopPropagation()}
    >
      {taskButtons.map((task) => (
        <Button
          type="button"
          key={task.key}
          variant="outline"
          size="sm"
          className={taskClassName}
          title={task.label}
          onClick={() => onOpenTask(row, task)}
        >
          <ClipboardList className="h-4 w-4 shrink-0" />
          <span className="min-w-0 truncate">{task.label}</span>
        </Button>
      ))}
      {canReassign ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={detailClassName}
          onClick={() => onOpenReassign(row)}
        >
          <UserRound className="h-4 w-4" />
          改派
        </Button>
      ) : null}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className={detailClassName}
        onClick={() => onOpenDetail(row)}
      >
        <Eye className="h-4 w-4" />
        详情
      </Button>
    </div>
  );
}

function WorkCustomerMobileList({
  rows,
  mode,
  loading,
  emptyTitle,
  emptyDescription,
  onOpenDetail,
  onOpenTask,
  onOpenReassign,
}: {
  rows: WorkCustomerListRowView[];
  mode: WorkCustomerMode;
  loading: boolean;
  emptyTitle: string;
  emptyDescription: string;
  onOpenDetail: (row: WorkCustomerListRowView) => void;
  onOpenTask: (
    row: WorkCustomerListRowView,
    task: WorkCustomerListTaskView,
  ) => void;
  onOpenReassign: (row: WorkCustomerListRowView) => void;
}) {
  if (loading) {
    return (
      <WorkListState
        loading
        title="正在加载"
        description="正在同步客户数据"
      />
    );
  }
  if (rows.length === 0) {
    return <WorkListState title={emptyTitle} description={emptyDescription} />;
  }
  return (
    <div>
      {rows.map((row) => {
        const { summaryTask, primaryTask, ruleTask, extraTasks } =
          workCustomerTaskGroups(row);
        return (
          <article
            key={row.id}
            className="crm-customer-list-mobile-row space-y-3 px-3 py-4"
          >
            <button
              type="button"
              className="block w-full min-w-0 text-left"
              onClick={() => onOpenDetail(row)}
            >
              <div className="flex min-w-0 items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="break-words font-medium">{row.customerName}</div>
                  <div className="mt-1 break-words text-xs text-muted-foreground">
                    {row.customerNo}
                  </div>
                </div>
                <span className="shrink-0 rounded bg-muted px-2 py-1 text-xs font-medium text-foreground">
                  {row.stageName}
                </span>
              </div>
              <div className="mt-3 text-foreground">
                {row.phone}
                {row.wechat !== "-" ? ` / ${row.wechat}` : ""}
              </div>
              <div className="mt-2 text-muted-foreground">
                <span className="break-words text-foreground">{row.assetName}</span>
                <span className="mx-1.5">·</span>
                {row.assetNo}
              </div>
              {row.hasStage ? (
                <div className="mt-2 text-muted-foreground">
                  {row.ownerName !== "-" ? row.ownerName : "暂未分配负责人"}
                  <span className="mx-1.5">·</span>
                  {row.stageDays > 0 ? `停留 ${row.stageDays} 天` : "今日进入"}
                </div>
              ) : null}
              <div className="mt-2">
                <span className="mr-2 text-xs text-muted-foreground">
                  {mode === "processed" ? "最近处理" : "当前待办"}
                </span>
                <span className="text-foreground">
                  {mode === "processed"
                    ? row.processedTaskName || "暂无处理记录"
                    : primaryTask?.label ||
                    summaryTask?.label ||
                    ruleTask?.label ||
                    "暂无待办"}
                </span>
                {mode === "processed" ? (
                  <div className="mt-1 text-xs text-muted-foreground">
                    {displayText(row.processedContent || row.processedResult, "已完成")}
                    <span className="mx-1.5">·</span>
                    {formatWorkDate(row.processedAt)}
                  </div>
                ) : null}
              </div>
            </button>
            <WorkCustomerRowActions
              row={row}
              primaryTask={primaryTask}
              extraTasks={extraTasks}
              mobile
              onOpenDetail={onOpenDetail}
              onOpenTask={onOpenTask}
              onOpenReassign={onOpenReassign}
              allowReassign={mode !== "processed"}
            />
          </article>
        );
      })}
    </div>
  );
}
