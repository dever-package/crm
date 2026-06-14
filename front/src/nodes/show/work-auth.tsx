import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ChangeEvent, FormEvent, ReactNode } from "react";
import {
  Check,
  Bot,
  Download,
  FileText,
  Inbox,
  Loader2,
  Plus,
  RefreshCw,
  Trash2,
  Upload,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  downloadUploadFile,
  uploadFileByRule,
  type UploadFileItem,
} from "@/lib/upload";
import {
  formatUploadSize,
  normalizeUploadItems,
  resolveResourcePreviewKind,
} from "@/lib/resource";
import { getStoreValueByPath, setStoreValueByPath } from "@/lib/store";

type StoreLike = Record<string, unknown>;

type WorkNodeProps = {
  item?: {
    id?: string;
    name?: string;
    value?: string;
    placeholder?: string;
    meta?: Record<string, unknown>;
  };
  store?: StoreLike;
  data?: Record<string, unknown>;
};

type RuntimeSite = {
  siteKey?: string;
  path?: string;
  base?: string;
  basePath?: string;
  siteHost?: string;
  host?: string;
  apiHost?: string;
  name?: string;
  subtitle?: string;
  logo?: string;
  site?: {
    name?: string;
    subtitle?: string;
    logo?: string;
  };
  runtime?: {
    siteKey?: string;
    basePath?: string;
    host?: string;
    siteHost?: string;
  };
};

type WorkFieldOption = {
  id?: string | number;
  name?: string;
  label?: string;
  value?: string | number;
};

type WorkFormField = {
  id?: string | number;
  name?: string;
  label?: string;
  field?: string;
  field_key?: string;
  field_name?: string;
  field_type?: string;
  main_field?: string;
  data_field_id?: string | number;
  data_template_id?: string | number;
  data_template_cate_id?: string | number;
  required?: boolean;
  default_value?: string | number;
  options?: WorkFieldOption[];
};

type WorkForm = {
  id?: string | number;
  name?: string;
  fields?: WorkFormField[];
};

type WorkTaskFieldRenderConfig = {
  type: string;
  placeholderPrefix: string;
  options?: WorkCommonOption[];
  meta?: Record<string, unknown>;
};

type WorkTask = {
  id?: string | number;
  name?: string;
  task_name?: string;
  todo_id?: string | number;
  todo_status?: string;
  todo_required?: boolean;
  todo_sort?: string | number;
  assigned_at?: string;
  assignee_department_id?: string | number;
  assignee_staff_id?: string | number;
  action_type?: string;
  task_action?: string;
  task_type?: string;
  trigger_type?: string;
  assign_mode?: string;
  assign_department_ids?: Array<string | number> | string;
  form_id?: string | number;
  form?: WorkForm | null;
};

type WorkCustomer = {
  id?: string | number;
  customer_id?: string | number;
  customer_no?: string;
  code_display?: string;
  code?: string;
  no?: string;
  name?: string;
  customer_name?: string;
  phone?: string;
  mobile?: string;
  wechat?: string;
  gender?: string;
  gender_name?: string;
  source_name?: string;
  source?: string;
  channel_name?: string;
  channel?: string;
  level_name?: string;
  customer_level?: string;
  source_id?: string | number;
  channel_id?: string | number;
  level_id?: string | number;
  status_name?: string;
  stage_name?: string;
  stage_code?: string;
  status_code?: string;
  current_stage_name?: string;
  current_status_name?: string;
  created_at?: string;
  create_time?: string;
  tasks?: WorkTask[];
  row_tasks?: WorkTask[];
  edit_tasks?: WorkTask[];
  assets?: WorkAsset[];
  operations?: WorkOperation[];
  data_values?: Record<string, unknown>;
  data_value_labels?: Record<string, string>;
  [key: string]: unknown;
};

type WorkAsset = {
  id?: string | number;
  asset_id?: string | number;
  customer_id?: string | number;
  asset_no?: string;
  asset_code?: string;
  code?: string;
  name?: string;
  asset_name?: string;
  asset_status_id?: string | number;
  asset_status_name?: string;
  status_name?: string;
  stage_name?: string;
  status_code?: string;
  stage_code?: string;
  current_stage_name?: string;
  current_status_name?: string;
  remark?: string;
  tasks?: WorkTask[];
  row_tasks?: WorkTask[];
  operations?: WorkOperation[];
  data_values?: Record<string, unknown>;
  data_value_labels?: Record<string, string>;
  [key: string]: unknown;
};

type WorkOperation = {
  id?: string | number;
  asset_id?: string | number;
  customer_id?: string | number;
  task_type?: string;
  result_value?: string;
  title?: string;
  summary?: string;
  operation_name?: string;
  task_name?: string;
  content?: string;
  remark?: string;
  operator_name?: string;
  operator_is_current?: boolean;
  "operator_staff.name"?: string;
  "operator_department.name"?: string;
  "task.name"?: string;
  created_at?: string;
  create_time?: string;
  summary_items?: WorkOperationSummaryItem[];
  [key: string]: unknown;
};

type WorkOperationSummaryItem = {
  key?: string;
  label?: string;
  value?: unknown;
  value_type?: string;
  files?: UploadFileItem[];
};

type WorkItem = {
  id: string;
  targetType: "customer" | "asset";
  customer: WorkCustomer;
  asset?: WorkAsset;
  tasks: WorkTask[];
};

type WorkCustomerMode = "all" | "pending" | "done";

type WorkSearchFilters = {
  customerNo: string;
  customerName: string;
  phone: string;
  wechat: string;
  assetNo: string;
  status: string;
};

type WorkDepartmentOption = {
  id?: string | number;
  name?: string;
  department_name?: string;
};

type WorkStaffOption = {
  id?: string | number;
  name?: string;
  real_name?: string;
  phone?: string;
  department_id?: string | number;
};

type WorkOptions = {
  departments: WorkDepartmentOption[];
  staffs: WorkStaffOption[];
};

type WorkCommonOption = {
  id: string;
  value: string;
  [key: string]: unknown;
};

type WorkTaskFormNode = {
  id: string;
  type: string;
  name?: string;
  placeholder?: string;
  value?: string;
  mode?: "form";
  option?: WorkCommonOption[] | string;
  validate?: Array<Record<string, unknown>>;
  meta?: Record<string, unknown>;
};

type WorkTaskUploadMeta = {
  ruleId: number;
  kind: string;
  maxCount: number;
  bizKey: string;
  bizName: string;
};

type WorkTaskUploadProgress = {
  fileName: string;
  percent: number;
  currentIndex: number;
  total: number;
};

type WorkTaskFormState = {
  nodes: WorkTaskFormNode[];
  values: Record<string, unknown>;
  fieldMap: Record<string, string>;
};

type WorkAIFillResponse = {
  values?: Record<string, unknown>;
  summary?: string;
  filled_count?: number;
};

type WorkPageStoreState = {
  schema?: {
    nodes?: Record<string, WorkTaskFormNode[]>;
    [key: string]: unknown;
  };
  errors?: Record<string, string>;
  validateForm?: () => boolean;
};

const workRefreshEvent = "crm-work-refresh";
const workTokenKey = "crm_work_token";
const workUserKey = "crm_work_user";
const legacyWorkTokenKey = "gjj_crm_work_token";
const legacyWorkUserKey = "gjj_crm_work_user";
const legacyFrontTokenKey = "front-token:work";
const legacyFrontUserKey = "front-user:work";
const defaultWorkSiteKey = "work";
const authCookieMaxAge = 3600 * 24 * 7;
const workTaskFormSectionID = "work-task-form-section";
const workTaskFormDataPath = "data.workTaskForm";
const workTaskFieldMapPath = "data.actionTarget.workTaskFieldMap";

const buttonBase =
  "inline-flex items-center justify-center gap-2 rounded-md text-sm font-medium shadow-sm transition disabled:cursor-not-allowed disabled:opacity-60";
const primaryButton = `${buttonBase} bg-primary text-primary-foreground hover:bg-primary/90`;
const outlineButton = `${buttonBase} border border-border bg-background hover:bg-muted`;
const inputClassName =
  "h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm outline-none transition placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20";

const workSearchFields: Array<{
  key: keyof WorkSearchFilters;
  placeholder: string;
  className: string;
}> = [
  {
    key: "customerNo",
    placeholder: "客户编号",
    className: "h-10 w-[160px] max-w-full",
  },
  {
    key: "customerName",
    placeholder: "姓名",
    className: "h-10 w-[140px] max-w-full",
  },
  {
    key: "phone",
    placeholder: "手机号",
    className: "h-10 w-[150px] max-w-full",
  },
  {
    key: "wechat",
    placeholder: "微信号",
    className: "h-10 w-[150px] max-w-full",
  },
  {
    key: "assetNo",
    placeholder: "资产编号",
    className: "h-10 w-[170px] max-w-full",
  },
  {
    key: "status",
    placeholder: "状态",
    className: "h-10 w-[140px] max-w-full",
  },
];

function emptyWorkSearchFilters(): WorkSearchFilters {
  return {
    customerNo: "",
    customerName: "",
    phone: "",
    wechat: "",
    assetNo: "",
    status: "",
  };
}

const workCustomerModeConfig: Record<
  WorkCustomerMode,
  {
    emptyTitle: string;
    emptyDescription: string;
  }
> = {
  all: {
    emptyTitle: "暂无客户工作",
    emptyDescription: "当前没有需要处理或已跟进的客户资产记录",
  },
  pending: {
    emptyTitle: "暂无待处理工作",
    emptyDescription: "当前没有需要处理的客户或资产任务",
  },
  done: {
    emptyTitle: "暂无已完成工作",
    emptyDescription: "完成客户或资产任务后，会在这里留存记录",
  },
};

const workTableHeadClass =
  "h-12 whitespace-nowrap px-4 text-left text-sm font-medium text-muted-foreground";
const workTableCellClass = "h-14 whitespace-nowrap px-4 align-middle text-sm";
const workUploadGridColumns = "minmax(0, 1fr) 6rem 7rem";
const workImageExtensions = new Set([
  "png",
  "jpg",
  "jpeg",
  "gif",
  "webp",
  "bmp",
  "svg",
]);

function textValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  return String(value).trim();
}

function errorMessage(error: unknown, fallback = "操作失败"): string {
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

function getRuntime(): RuntimeSite {
  const globalWindow = window as unknown as {
    appRuntime?: RuntimeSite;
    __DEVER_RUNTIME__?: RuntimeSite;
  };
  return globalWindow.appRuntime ?? globalWindow.__DEVER_RUNTIME__ ?? {};
}

function getRuntimeSite() {
  const runtime = getRuntime();
  const site = runtime.site ?? {};
  return {
    name: textValue(site.name) || textValue(runtime.name) || "DoublePlus平台",
    subtitle:
      textValue(site.subtitle) ||
      textValue(runtime.subtitle) ||
      "客户中心工作台",
    logo: textValue(site.logo) || textValue(runtime.logo),
  };
}

function normalizeStorageScope(value: string): string {
  return value.trim().replace(/[^a-zA-Z0-9._-]+/g, "_") || "default";
}

function getWorkSiteKey(): string {
  const runtime = getRuntime();
  return (
    textValue(runtime.siteKey) ||
    textValue(runtime.runtime?.siteKey) ||
    defaultWorkSiteKey
  );
}

function getCurrentHostKey(): string {
  const runtime = getRuntime();
  return (
    textValue(window.location.host) ||
    textValue(window.location.hostname) ||
    textValue(runtime.siteHost) ||
    textValue(runtime.runtime?.siteHost) ||
    textValue(runtime.host) ||
    textValue(runtime.runtime?.host) ||
    "default"
  );
}

function getFrontAuthScope(): string {
  return `${normalizeStorageScope(getWorkSiteKey())}_${normalizeStorageScope(
    getCurrentHostKey(),
  )}`;
}

function getFrontTokenKey(): string {
  return `front-token:${getFrontAuthScope()}`;
}

function getFrontUserKey(): string {
  return `front-user:${getFrontAuthScope()}`;
}

function getCookieValue(name: string): string {
  const parts = `; ${document.cookie}`.split(`; ${name}=`);
  if (parts.length !== 2) return "";
  return parts.pop()?.split(";").shift() ?? "";
}

function setCookieValue(name: string, value: string, maxAge: number) {
  document.cookie = `${name}=${value}; path=/; max-age=${maxAge}`;
}

function removeCookieValue(name: string) {
  setCookieValue(name, "", 0);
}

function readTokenCookie(name: string): string {
  const raw = getCookieValue(name);
  if (!raw) return "";
  try {
    const parsed = JSON.parse(raw) as unknown;
    return textValue(parsed);
  } catch {
    return textValue(raw);
  }
}

function getRuntimeBasePath(): string {
  const runtime = getRuntime();
  return (
    textValue(runtime.basePath) ||
    textValue(runtime.runtime?.basePath) ||
    textValue(runtime.base) ||
    "/work"
  );
}

function getWorkEntryPath(): string {
  const basePath = getRuntimeBasePath();
  const normalized = basePath.startsWith("/") ? basePath : `/${basePath}`;
  return normalized.replace(/\/login\/?$/, "") || "/work";
}

function saveWorkSession(token: string, user: unknown) {
  window.localStorage.setItem(workTokenKey, token);
  window.localStorage.setItem(workUserKey, JSON.stringify(user ?? {}));
  window.localStorage.setItem(legacyWorkTokenKey, token);
  window.localStorage.setItem(legacyWorkUserKey, JSON.stringify(user ?? {}));
  window.localStorage.setItem(legacyFrontTokenKey, token);
  window.localStorage.setItem(legacyFrontUserKey, JSON.stringify(user ?? {}));

  setCookieValue(getFrontTokenKey(), JSON.stringify(token), authCookieMaxAge);
  window.localStorage.setItem(getFrontUserKey(), JSON.stringify(user ?? {}));
}

function clearWorkSession() {
  window.localStorage.removeItem(workTokenKey);
  window.localStorage.removeItem(workUserKey);
  window.localStorage.removeItem(legacyWorkTokenKey);
  window.localStorage.removeItem(legacyWorkUserKey);
  window.localStorage.removeItem(legacyFrontTokenKey);
  window.localStorage.removeItem(legacyFrontUserKey);
  window.localStorage.removeItem(getFrontUserKey());

  removeCookieValue(getFrontTokenKey());
  removeCookieValue(legacyFrontTokenKey);
}

function getWorkToken(): string {
  return (
    readTokenCookie(getFrontTokenKey()) ||
    readTokenCookie(legacyFrontTokenKey) ||
    window.localStorage.getItem(workTokenKey) ||
    window.localStorage.getItem(legacyWorkTokenKey) ||
    window.localStorage.getItem(legacyFrontTokenKey) ||
    ""
  );
}

async function workApi<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers || {});
  const token = getWorkToken();

  if (token) headers.set("Authorization", `Bearer ${token}`);
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(path, {
    ...init,
    headers,
    credentials: "include",
  });
  const payload = await response.json().catch(() => ({}));

  if (!response.ok || payload?.code) {
    throw new Error(
      textValue(payload?.message) ||
        textValue(payload?.msg) ||
        `请求失败：${response.status}`,
    );
  }

  return (payload?.data ?? payload) as T;
}

function workStoreValue<T>(
  store: StoreLike | undefined,
  path: string,
  fallback: T,
): T {
  const value = store ? getStoreValueByPath(store, path) : undefined;
  return value === undefined || value === null ? fallback : (value as T);
}

function setWorkStoreValue(
  store: StoreLike | undefined,
  path: string,
  value: unknown,
) {
  if (!store) return;
  setStoreValueByPath(store, path, value);
}

function setWorkModalOpen(
  store: StoreLike | undefined,
  key: string,
  open: boolean,
) {
  const getState = (
    store as
      | {
          getState?: () => {
            setPageState?: (key: string, value: boolean) => void;
          };
        }
      | undefined
  )?.getState;
  const state = typeof getState === "function" ? getState() : undefined;
  if (typeof state?.setPageState === "function") {
    state.setPageState(key, open);
    return;
  }
  setWorkStoreValue(store, `state.${key}`, open);
}

function currentWorkStoreState(
  store: StoreLike | undefined,
): WorkPageStoreState | undefined {
  return (
    store as { getState?: () => WorkPageStoreState } | undefined
  )?.getState?.();
}

function updateWorkStoreErrors(
  store: StoreLike | undefined,
  nextErrors: (errors: Record<string, string>) => Record<string, string>,
) {
  const storeApi = store as
    | {
        getState?: () => WorkPageStoreState;
        setState?: (
          updater: (state: WorkPageStoreState) => Partial<WorkPageStoreState>,
        ) => void;
      }
    | undefined;

  if (typeof storeApi?.setState === "function") {
    storeApi.setState((state) => ({
      errors: nextErrors(state.errors || {}),
    }));
    return;
  }

  const state = storeApi?.getState?.();
  if (state) {
    state.errors = nextErrors(state.errors || {});
  }
}

function positiveTextID(value: unknown): string {
  const id = textValue(value);
  return id && id !== "0" ? id : "";
}

function displayText(value: unknown, fallback = "-"): string {
  const text = textValue(value);
  return text || fallback;
}

function formatWorkDate(value: unknown): string {
  const text = textValue(value);
  if (!text) return "-";
  return text
    .replace("T", " ")
    .replace(/\.\d+Z?$/, "")
    .slice(0, 16);
}

function workCustomerID(customer?: WorkCustomer | null): string {
  return positiveTextID(customer?.id) || positiveTextID(customer?.customer_id);
}

function workAssetID(asset?: WorkAsset | null): string {
  return positiveTextID(asset?.id) || positiveTextID(asset?.asset_id);
}

function workCustomerNo(customer?: WorkCustomer | null): string {
  return (
    textValue(customer?.code_display) ||
    textValue(customer?.customer_no) ||
    textValue(customer?.code) ||
    textValue(customer?.no) ||
    "-"
  );
}

function workCustomerTitle(customer?: WorkCustomer | null): string {
  return workCustomerName(customer) || workCustomerNo(customer);
}

function workCustomerName(customer?: WorkCustomer | null): string {
  return textValue(customer?.name) || textValue(customer?.customer_name) || "-";
}

function workCustomerPhone(customer?: WorkCustomer | null): string {
  return textValue(customer?.phone) || textValue(customer?.mobile) || "-";
}

function workAssetNo(asset?: WorkAsset | null): string {
  return (
    textValue(asset?.asset_no) ||
    textValue(asset?.asset_code) ||
    textValue(asset?.code) ||
    ""
  );
}

function workAssetName(asset?: WorkAsset | null): string {
  return textValue(asset?.asset_name) || textValue(asset?.name) || "-";
}

function assetTitle(asset?: WorkAsset | null): string {
  return (
    workAssetName(asset) || workAssetNo(asset) || `资产${textValue(asset?.id)}`
  );
}

function workStatusName(target?: WorkCustomer | WorkAsset | null): string {
  return (
    textValue(target?.status_name) ||
    textValue(target?.current_status_name) ||
    textValue(target?.stage_name) ||
    textValue(target?.current_stage_name) ||
    textValue(target?.status_code) ||
    textValue(target?.stage_code) ||
    "-"
  );
}

function workTaskName(task: WorkTask): string {
  return textValue(task.task_name) || textValue(task.name) || "任务";
}

function workTaskAction(task: WorkTask): string {
  return (
    textValue(task.action_type) ||
    textValue(task.task_action) ||
    textValue(task.task_type) ||
    "form"
  );
}

function workTaskIsAssign(task: WorkTask): boolean {
  const action = workTaskAction(task);
  return action === "assign" || action === "dispatch" || action === "分配";
}

function workTaskIsDecision(task: WorkTask): boolean {
  const action = workTaskAction(task);
  return action === "decision" || action === "决策" || action === "自动决策";
}

function workTaskIsBooking(task: WorkTask): boolean {
  const action = workTaskAction(task);
  return action === "booking" || action === "resource" || action === "资源预定";
}

function workTaskIsCollaborate(task: WorkTask): boolean {
  const action = workTaskAction(task);
  return action === "collaborate" || action === "collaboration" || action === "协作任务";
}

function workTaskIsTodo(task: WorkTask): boolean {
  return Boolean(positiveTextID(task.todo_id));
}

function workTaskIsCreate(task: WorkTask): boolean {
  const action = workTaskAction(task);
  return action === "create" || action === "创建资料";
}

function workTaskIsForm(task: WorkTask): boolean {
  return (
    !workTaskIsCreate(task) &&
    !workTaskIsAssign(task) &&
    !workTaskIsDecision(task) &&
    !workTaskIsBooking(task) &&
    !workTaskIsCollaborate(task)
  );
}

function workTaskShouldRenderFields(task: WorkTask): boolean {
  const fields = task.form?.fields || [];
  if (fields.length === 0) {
    return (
      workTaskIsCreate(task) ||
      workTaskIsForm(task) ||
      workTaskIsBooking(task) ||
      workTaskIsCollaborate(task)
    );
  }
  return (
    workTaskIsCreate(task) ||
    workTaskIsForm(task) ||
    workTaskIsDecision(task) ||
    workTaskIsBooking(task) ||
    workTaskIsAssign(task) ||
    workTaskIsCollaborate(task)
  );
}

function workTaskButtonLabel(task: WorkTask): string {
  const name = workTaskName(task);
  if (name && name !== "任务") return name;
  if (workTaskIsCreate(task)) return "创建资料";
  if (workTaskIsAssign(task)) return "派单";
  if (workTaskIsDecision(task)) return "决策";
  if (workTaskIsBooking(task)) return "资源预定";
  if (workTaskIsCollaborate(task)) return workTaskIsTodo(task) ? "完成协作" : "协作任务";
  return "填写资料";
}

function workTaskKey(task: WorkTask): string {
  const todoID = positiveTextID(task.todo_id);
  const taskID = positiveTextID(task.id);
  return todoID ? `${taskID || "task"}:todo:${todoID}` : taskID || workTaskName(task);
}

function workCustomerRowTasks(customer: WorkCustomer): WorkTask[] {
  if (Array.isArray(customer.row_tasks)) return customer.row_tasks;
  if (Array.isArray(customer.edit_tasks)) return customer.edit_tasks;
  return Array.isArray(customer.tasks) ? customer.tasks : [];
}

function workAssetRowTasks(asset: WorkAsset): WorkTask[] {
  if (Array.isArray(asset.row_tasks)) return asset.row_tasks;
  return Array.isArray(asset.tasks) ? asset.tasks : [];
}

function buildWorkItems(customers: WorkCustomer[]): WorkItem[] {
  const items = customers.flatMap((customer) => {
    const customerID = workCustomerID(customer);
    const assets = Array.isArray(customer.assets) ? customer.assets : [];
    if (assets.length === 0) {
      return [
        {
          id: `customer:${customerID || workCustomerNo(customer)}`,
          targetType: "customer",
          customer,
          tasks: workCustomerRowTasks(customer),
        },
      ];
    }

    return assets.map((asset, index) => ({
      id: `asset:${customerID || workCustomerNo(customer)}:${workAssetID(asset) || workAssetNo(asset) || index}`,
      targetType: "asset",
      customer,
      asset,
      tasks: workAssetRowTasks(asset),
    }));
  });
  return sortWorkItems(items);
}

function sortWorkItems(items: WorkItem[]): WorkItem[] {
  return [...items].sort((left, right) => {
    const leftPending = workItemHasPendingTasks(left);
    const rightPending = workItemHasPendingTasks(right);
    if (leftPending !== rightPending) return leftPending ? -1 : 1;
    return 0;
  });
}

function workItemHasPendingTasks(item: WorkItem): boolean {
  return item.tasks.length > 0;
}

function workItemAssetNo(item: WorkItem): string {
  return item.asset ? workAssetNo(item.asset) || "-" : "-";
}

function workItemCustomerNo(item: WorkItem): string {
  return workCustomerNo(item.customer);
}

function renderWorkItemStatus(item: WorkItem) {
  return renderStatus(item.asset || item.customer);
}

function workSearchQuery(filters: WorkSearchFilters): string {
  const params = new URLSearchParams();
  const entries: Array<[string, string]> = [
    ["customer_no", filters.customerNo],
    ["customer_name", filters.customerName],
    ["phone", filters.phone],
    ["wechat", filters.wechat],
    ["asset_no", filters.assetNo],
    ["status", filters.status],
  ];
  entries.forEach(([key, value]) => {
    const text = textValue(value);
    if (text) params.set(key, text);
  });
  const query = params.toString();
  return query ? `?${query}` : "";
}

function workCustomerQuery(
  filters: WorkSearchFilters,
  mode: WorkCustomerMode,
): string {
  const params = new URLSearchParams(
    workSearchQuery(filters).replace(/^\?/, ""),
  );
  params.set("mode", mode);
  const query = params.toString();
  return query ? `?${query}` : "";
}

function workCustomerModeFromNode(
  item?: WorkNodeProps["item"],
): WorkCustomerMode {
  const configured = textValue(item?.meta?.mode || item?.meta?.customerMode);
  if (configured === "all") return "all";
  if (configured === "done") return "done";
  if (configured === "pending") return "pending";
  const pathname = textValue(window.location.pathname);
  return pathname.endsWith("/work/done") || pathname.includes("/work/done/")
    ? "done"
    : "all";
}

function renderStatus(
  target?: Pick<
    WorkCustomer,
    | "stage_code"
    | "stage_name"
    | "status_code"
    | "status_name"
    | "current_status_name"
    | "current_stage_name"
  > | null,
) {
  const statusName = workStatusName(target);
  if (!statusName || statusName === "-") {
    return <span className="text-muted-foreground">-</span>;
  }
  return (
    <span className="rounded-full bg-muted px-2 py-1 text-xs font-medium">
      {statusName}
    </span>
  );
}

function renderAssetStatus(asset: WorkAsset) {
  const statusName = textValue(asset.asset_status_name);
  if (!statusName) {
    return null;
  }
  return (
    <span className="rounded-full border px-2 py-0.5 text-xs text-muted-foreground">
      {statusName}
    </span>
  );
}

async function openRowTask(
  customer: WorkCustomer | null,
  task: WorkTask,
  store?: StoreLike,
  asset?: WorkAsset,
) {
  setWorkStoreValue(store, "data.actionTarget.workTask", task);
  setWorkStoreValue(store, "data.actionTarget.workTaskCustomer", customer);
  setWorkStoreValue(store, "data.actionTarget.workTaskAsset", asset ?? null);
  setWorkStoreValue(
    store,
    "data.actionTarget.workTaskName",
    workTaskButtonLabel(task),
  );
  await prepareWorkTaskForm(store, task, customer, asset);
  setWorkModalOpen(store, "dialog.workTask", true);
}

async function prepareWorkTaskForm(
  store: StoreLike | undefined,
  task: WorkTask,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
) {
  const options = await loadWorkTaskOptions(task);
  const formState = buildWorkTaskFormState(task, customer, asset, options);
  setWorkStoreValue(store, workTaskFormDataPath, formState.values);
  setWorkStoreValue(store, workTaskFieldMapPath, formState.fieldMap);
  replaceWorkTaskFormSection(store, formState.nodes);
}

async function loadWorkTaskOptions(task: WorkTask): Promise<WorkOptions> {
  if (!workTaskIsAssign(task)) {
    return { departments: [], staffs: [] };
  }

  try {
    const payload = await workApi<Partial<WorkOptions>>("/crm/work/options");
    return {
      departments: Array.isArray(payload.departments)
        ? payload.departments
        : [],
      staffs: Array.isArray(payload.staffs) ? payload.staffs : [],
    };
  } catch (error) {
    toast.error(errorMessage(error, "选项加载失败"));
    return { departments: [], staffs: [] };
  }
}

function replaceWorkTaskFormSection(
  store: StoreLike | undefined,
  nodes: WorkTaskFormNode[],
) {
  const storeApi = store as
    | {
        getState?: () => WorkPageStoreState;
        setState?: (
          updater: (state: WorkPageStoreState) => Partial<WorkPageStoreState>,
        ) => void;
      }
    | undefined;

  if (typeof storeApi?.setState === "function") {
    storeApi.setState((state) => ({
      schema: {
        ...(state.schema || {}),
        nodes: {
          ...(state.schema?.nodes || {}),
          [workTaskFormSectionID]: nodes,
        },
      },
      errors: {},
    }));
    return;
  }

  const state = storeApi?.getState?.();
  if (!state?.schema) return;
  state.schema.nodes = {
    ...(state.schema.nodes || {}),
    [workTaskFormSectionID]: nodes,
  };
}

function buildWorkTaskFormState(
  task: WorkTask,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
  options: WorkOptions = { departments: [], staffs: [] },
): WorkTaskFormState {
  const nodes: WorkTaskFormNode[] = [
    {
      id: "work-task-submit-controller",
      type: "show-crm-work-task-form",
    },
  ];
  const values: Record<string, unknown> = {};
  const fieldMap: Record<string, string> = {};

  if (workTaskShouldRenderFields(task)) {
    for (const field of task.form?.fields || []) {
      addWorkTaskFieldNode(nodes, values, fieldMap, field, customer, asset);
    }
  }

  if (workTaskIsAssign(task)) {
    const assignMode = workTaskAssignMode(task);
    const departments = workAllowedDepartments(task, options.departments);
    const departmentFormKey = addWorkTaskSelectNode(nodes, values, fieldMap, {
      formKey: "assign_department_id",
      rawKey: "department_id",
      label: "部门",
      placeholder: "请选择部门",
      required: true,
      options: workDepartmentOptions(departments),
    });
    if (assignMode === "staff") {
      addWorkTaskSelectNode(nodes, values, fieldMap, {
        formKey: "staff_id",
        rawKey: "staff_id",
        label: "人员",
        placeholder: "请选择人员，不选则自动派给部门负责人",
        required: false,
        options: workStaffOptions(options.staffs),
        meta: {
          hiddenWhen: [
            { path: `workTaskForm.${departmentFormKey}`, operator: "empty" },
          ],
          optionFilter: [
            {
              field: "department_id",
              path: `workTaskForm.${departmentFormKey}`,
              operator: "equals",
            },
          ],
        },
      });
    }
  }

  return { nodes, values, fieldMap };
}

function addWorkTaskFieldNode(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  field: WorkFormField,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
) {
  const rawKey = workFieldKey(field);
  if (!rawKey) return;

  const formKey = workTaskFormKey(rawKey);
  const label = textValue(field.label) || textValue(field.name) || rawKey;
  const options = Array.isArray(field.options)
    ? field.options.map(workFieldOption)
    : [];
  const renderConfig = workTaskFieldRenderConfig(field, options);

  addWorkTaskTextNode(nodes, values, fieldMap, {
    formKey,
    rawKey,
    label,
    placeholder: `${renderConfig.placeholderPrefix}${label}`,
    required: Boolean(field.required),
    type: renderConfig.type,
    options: renderConfig.options,
    initialValue: workFieldInitialValue(
      field,
      customer,
      asset,
      renderConfig.type,
    ),
    meta: renderConfig.meta,
  });
}

function workTaskFieldRenderConfig(
  field: WorkFormField,
  options: WorkCommonOption[],
): WorkTaskFieldRenderConfig {
  const fieldType = textValue(field.field_type);
  if (options.length > 0) {
    return {
      type: "form-select",
      placeholderPrefix: "请选择",
      options,
      meta:
        fieldType === "multi_select" || fieldType === "checkbox"
          ? { multiple: true }
          : undefined,
    };
  }

  if (workTaskFieldIsUpload(field)) {
    return {
      type: "show-crm-work-task-upload",
      placeholderPrefix: "请上传",
      meta: workTaskUploadFieldMeta(field),
    };
  }

  if (fieldType === "textarea") {
    return {
      type: "form-textarea",
      placeholderPrefix: "请输入",
    };
  }

  return {
    type: "form-input",
    placeholderPrefix: "请输入",
  };
}

function workTaskFieldIsUpload(field: WorkFormField): boolean {
  const fieldType = textValue(field.field_type);
  return (
    fieldType === "attachment" || fieldType === "file" || fieldType === "image"
  );
}

function workTaskUploadFieldMeta(
  _field: WorkFormField,
): Record<string, unknown> {
  return {
    ruleId: 6,
    kind: "file",
    uploadType: "list",
    maxCount: 10,
    bizKey: "crm.work",
    bizName: "CRM工作台",
  };
}

function addWorkTaskSelectNode(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  config: {
    formKey: string;
    rawKey: string;
    label: string;
    placeholder: string;
    required?: boolean;
    options: WorkCommonOption[];
    initialValue?: unknown;
    meta?: Record<string, unknown>;
  },
): string {
  return addWorkTaskTextNode(nodes, values, fieldMap, {
    ...config,
    type: "form-select",
  });
}

function addWorkTaskTextNode(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  config: {
    formKey: string;
    rawKey: string;
    label: string;
    placeholder: string;
    required?: boolean;
    type?: string;
    options?: WorkCommonOption[];
    initialValue?: unknown;
    meta?: Record<string, unknown>;
  },
): string {
  const formKey = uniqueWorkTaskFormKey(config.formKey, fieldMap);
  values[formKey] = formatWorkTaskInitialValue(config);
  fieldMap[formKey] = config.rawKey;

  nodes.push({
    id: `work-task-field-${formKey}`,
    type: config.type || "form-input",
    name: config.label,
    placeholder: config.placeholder,
    value: `workTaskForm.${formKey}`,
    mode: "form",
    ...(config.options ? { option: config.options } : {}),
    ...(config.required
      ? {
          validate: [
            {
              type: "required",
              message: `${config.label}不能为空。`,
            },
          ],
        }
      : {}),
    meta: {
      formLayout: "horizontal",
      ...(config.type === "form-textarea" ? { rows: 4 } : {}),
      ...(config.meta || {}),
    },
  });
  return formKey;
}

function formatWorkTaskInitialValue(config: {
  type?: string;
  initialValue?: unknown;
}): unknown {
  if (config.type === "show-crm-work-task-upload") {
    return formatUploadInitialValue(config.initialValue);
  }
  return formatFormValue(config.initialValue);
}

function formatUploadInitialValue(value: unknown): unknown {
  if (value === null || value === undefined || value === "") return [];
  if (Array.isArray(value)) return value;
  if (typeof value === "object") return [value];
  const numericValue = Number(value);
  return Number.isFinite(numericValue) && numericValue > 0 ? numericValue : [];
}

function workTaskFormKey(key: string): string {
  const normalized = key
    .trim()
    .replace(/[^a-zA-Z0-9_]+/g, "_")
    .replace(/^_+|_+$/g, "");
  return normalized || "field";
}

function uniqueWorkTaskFormKey(
  key: string,
  fieldMap: Record<string, string>,
): string {
  const base = workTaskFormKey(key);
  let candidate = base;
  let index = 2;
  while (Object.prototype.hasOwnProperty.call(fieldMap, candidate)) {
    candidate = `${base}_${index}`;
    index += 1;
  }
  return candidate;
}

function workFieldOption(option: WorkFieldOption): WorkCommonOption {
  const id = workOptionID(option);
  return {
    ...option,
    id,
    value: workOptionLabel(option),
  };
}

function workDepartmentOptions(
  departments: WorkDepartmentOption[],
): WorkCommonOption[] {
  return departments
    .map((department) => {
      const id = textValue(department.id);
      return id
        ? {
            ...department,
            id,
            value: displayText(department.department_name || department.name),
          }
        : null;
    })
    .filter(Boolean) as WorkCommonOption[];
}

function workStaffOptions(staffs: WorkStaffOption[]): WorkCommonOption[] {
  return staffs
    .map((staff) => {
      const id = textValue(staff.id);
      return id
        ? {
            ...staff,
            id,
            department_id: textValue(staff.department_id),
            value: `${displayText(staff.real_name || staff.name)}${staff.phone ? `（${staff.phone}）` : ""}`,
          }
        : null;
    })
    .filter(Boolean) as WorkCommonOption[];
}

function workTaskAssignMode(task: WorkTask): "staff" | "department" {
  return textValue(task.assign_mode) === "department" ? "department" : "staff";
}

function workTaskAllowedDepartmentIDSet(task: WorkTask): Set<string> {
  const raw = task.assign_department_ids;
  const values = Array.isArray(raw)
    ? raw
    : (() => {
        const text = textValue(raw);
        if (!text) return [];
        try {
          const parsed = JSON.parse(text) as unknown;
          return Array.isArray(parsed) ? parsed : [text];
        } catch {
          return text.split(",");
        }
      })();
  return new Set(values.map((value) => textValue(value)).filter(Boolean));
}

function workAllowedDepartments(
  task: WorkTask,
  departments: WorkDepartmentOption[],
): WorkDepartmentOption[] {
  const allowed = workTaskAllowedDepartmentIDSet(task);
  if (allowed.size === 0) return departments;
  return departments.filter((department) =>
    allowed.has(textValue(department.id)),
  );
}

function openWorkDetail(
  customer: WorkCustomer,
  store?: StoreLike,
  asset?: WorkAsset,
) {
  setWorkDetailTarget(store, customer, asset);
  setWorkModalOpen(store, "dialog.workDetail", false);
  setWorkModalOpen(store, "drawer.workDetail", true);
}

function setWorkDetailTarget(
  store: StoreLike | undefined,
  customer: WorkCustomer,
  asset?: WorkAsset | null,
) {
  const detailAsset = asset ?? undefined;
  setWorkStoreValue(store, "data.actionTarget.workDetailCustomer", customer);
  setWorkStoreValue(store, "data.actionTarget.workDetailAsset", asset ?? null);
  setWorkStoreValue(
    store,
    "data.actionTarget.workDetailName",
    workDetailTitle(customer, detailAsset),
  );
  setWorkStoreValue(
    store,
    "data.actionTarget.workDetailDescription",
    workDetailDescription(customer, detailAsset),
  );
}

async function refreshWorkDetailTarget(
  store: StoreLike | undefined,
  customerID: string,
  assetID = "",
) {
  if (!customerID) return;
  try {
    const payload = await workApi<{
      list?: WorkCustomer[];
      customers?: WorkCustomer[];
      data?: WorkCustomer[];
    }>("/crm/work/customers?mode=all");
    const customers = payload.list || payload.customers || payload.data || [];
    if (!Array.isArray(customers)) return;
    const customer = customers.find((row) => workCustomerID(row) === customerID);
    if (!customer) return;
    const asset =
      assetID && Array.isArray(customer.assets)
        ? customer.assets.find((row) => workAssetID(row) === assetID)
        : undefined;
    setWorkDetailTarget(store, customer, asset || null);
  } catch (error) {
    toast.error(errorMessage(error, "详情刷新失败"));
  }
}

function workDetailTitle(customer: WorkCustomer, asset?: WorkAsset): string {
  if (asset) {
    return workAssetNo(asset) || assetTitle(asset);
  }
  return workCustomerNo(customer) || workCustomerTitle(customer);
}

function workDetailDescription(
  customer: WorkCustomer,
  asset?: WorkAsset,
): string {
  return [
    workCustomerPhone(customer),
    workCustomerName(customer),
    textValue(customer.wechat),
    workStatusName(asset || customer),
  ]
    .filter(Boolean)
    .join(" / ");
}

function openWorkRecords(
  customer: WorkCustomer,
  store?: StoreLike,
  asset?: WorkAsset,
) {
  setWorkStoreValue(store, "data.actionTarget.workRecordCustomer", customer);
  setWorkStoreValue(store, "data.actionTarget.workRecordAsset", asset ?? null);
  setWorkStoreValue(store, "data.actionTarget.workRecordName", "我的记录");
  setWorkStoreValue(
    store,
    "data.actionTarget.workRecordDescription",
    workRecordDescription(customer, asset),
  );
  setWorkModalOpen(store, "drawer.workRecords", true);
}

function openWorkRecordDetail(record: WorkOperation, store?: StoreLike) {
  setWorkStoreValue(store, "data.actionTarget.workRecordDetail", record);
  setWorkStoreValue(
    store,
    "data.actionTarget.workRecordDetailName",
    workRecordTitle(record),
  );
  setWorkStoreValue(
    store,
    "data.actionTarget.workRecordDetailDescription",
    workRecordDetailDescription(record),
  );
  setWorkModalOpen(store, "dialog.workRecordDetail", true);
}

function workRecordDescription(
  customer: WorkCustomer,
  asset?: WorkAsset,
): string {
  const values = asset
    ? [assetTitle(asset), workCustomerTitle(customer)]
    : [workCustomerTitle(customer), workCustomerPhone(customer)];
  return values.map(textValue).filter(Boolean).join(" / ");
}

function workRecordTitle(record: WorkOperation): string {
  return workOperationTitle(record, "任务记录");
}

function workRecordSubtitle(record: WorkOperation): string {
  return displayText(
    record.summary || record.content || record.remark,
    "任务记录",
  );
}

function workRecordTime(record: WorkOperation): string {
  return formatWorkDate(record.created_at || record.create_time);
}

function workRecordDetailDescription(record: WorkOperation): string {
  return [workRecordSubtitle(record), workRecordTime(record)]
    .map(textValue)
    .filter(Boolean)
    .join(" / ");
}

function workRecordSummaryItems(
  record: WorkOperation,
): WorkOperationSummaryItem[] {
  return Array.isArray(record.summary_items) ? record.summary_items : [];
}

function workOperationTitle(
  operation: WorkOperation,
  fallback = "操作记录",
): string {
  return displayText(
    operation.task_name ||
      operation.operation_name ||
      operation["task.name"] ||
      operation.title,
    fallback,
  );
}

function workOperationDescription(operation: WorkOperation): string {
  return (
    textValue(operation.content) ||
    textValue(operation.remark) ||
    textValue(operation.summary)
  );
}

function workPendingTaskSummary(tasks: WorkTask[]): string {
  return tasks.length > 0 ? `${tasks.length} 个待处理任务` : "暂无待处理任务";
}

function notifyWorkRefresh() {
  window.dispatchEvent(new CustomEvent(workRefreshEvent));
}

function notifyWorkDataChanged() {
  notifyWorkRefresh();
}

export function ShowCrmWorkLogin() {
  const site = getRuntimeSite();
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [loginError, setLoginError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setLoginError("");
    if (!phone || !password) {
      setLoginError("请输入手机号和密码");
      toast.error("请输入手机号和密码");
      return;
    }

    setSubmitting(true);
    try {
      const payload = await workApi<{ token?: string; user?: unknown }>(
        "/crm/work/login",
        {
          method: "POST",
          body: JSON.stringify({ phone, password }),
        },
      );
      const token = textValue(payload.token);
      if (!token) throw new Error("登录返回缺少 token");
      saveWorkSession(token, payload.user);
      window.location.href = getWorkEntryPath();
    } catch (error) {
      const message = errorMessage(error, "登录失败");
      setLoginError(message);
      toast.error(message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className="container grid h-svh max-w-none items-center justify-center bg-background text-foreground">
      <div
        className="mx-auto flex flex-col justify-center space-y-2 py-8 sm:p-8"
        style={{ width: "min(520px, calc(100vw - 40px))" }}
      >
        <div className="mb-6 flex items-center justify-center gap-4">
          {site.logo ? (
            <img
              src={site.logo}
              alt={site.name}
              style={{ width: 56, height: 56, minWidth: 56 }}
            />
          ) : (
            <div
              className="flex items-center justify-center bg-primary text-lg font-semibold text-primary-foreground shadow-sm"
              style={{ width: 56, height: 56, minWidth: 56, borderRadius: 14 }}
            >
              {site.name.slice(0, 1) || "D"}
            </div>
          )}
          <div>
            <h1 className="text-2xl font-semibold leading-tight">
              {site.name}
            </h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {site.subtitle}
            </p>
          </div>
        </div>
        <form
          onSubmit={submit}
          className="w-full rounded-lg border bg-card p-6 shadow-sm"
        >
          <div style={{ marginBottom: 28 }}>
            <h2 className="text-lg font-semibold leading-tight">人员登录</h2>
            <p
              className="text-sm text-muted-foreground"
              style={{ marginTop: 8 }}
            >
              请输入手机号和密码进入工作台
            </p>
          </div>
          <div className="flex flex-col" style={{ gap: 22 }}>
            <label className="block">
              <span
                className="block text-sm font-medium"
                style={{ marginBottom: 8 }}
              >
                手机号
              </span>
              <input
                className={inputClassName}
                value={phone}
                onChange={(event) => {
                  setPhone(event.target.value);
                  if (loginError) setLoginError("");
                }}
                placeholder="请输入手机号"
                autoComplete="tel"
              />
            </label>
            <label className="block">
              <span
                className="block text-sm font-medium"
                style={{ marginBottom: 8 }}
              >
                密码
              </span>
              <input
                className={inputClassName}
                value={password}
                onChange={(event) => {
                  setPassword(event.target.value);
                  if (loginError) setLoginError("");
                }}
                placeholder="请输入密码"
                type="password"
                autoComplete="current-password"
              />
            </label>
          </div>
          {loginError ? (
            <div
              role="alert"
              className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
              style={{ marginTop: 20 }}
            >
              {loginError}
            </div>
          ) : null}
          <button
            type="submit"
            disabled={submitting}
            className={`${primaryButton} h-10 w-full px-4`}
            style={{ marginTop: 28 }}
          >
            {submitting ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Check className="h-4 w-4" />
            )}
            登录
          </button>
        </form>
      </div>
    </main>
  );
}

export function ShowCrmWorkRefreshButton() {
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={notifyWorkDataChanged}
    >
      <RefreshCw className="h-4 w-4" />
      刷新
    </Button>
  );
}

export function ShowCrmWorkTasks({ store }: WorkNodeProps = {}) {
  const [tasks, setTasks] = useState<WorkTask[]>([]);
  const [loading, setLoading] = useState(false);

  const loadTasks = useCallback(async () => {
    setLoading(true);
    try {
      const payload = await workApi<{ tasks?: WorkTask[]; list?: WorkTask[] }>(
        "/crm/work/tasks",
      );
      const list = payload.tasks || payload.list || [];
      setTasks(Array.isArray(list) ? list : []);
    } catch (error) {
      toast.error(errorMessage(error, "任务加载失败"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTasks();
  }, [loadTasks]);

  useEffect(() => {
    window.addEventListener(workRefreshEvent, loadTasks);
    return () => window.removeEventListener(workRefreshEvent, loadTasks);
  }, [loadTasks]);

  const openTask = (task: WorkTask) => {
    void openRowTask(null, task, store);
  };

  if (loading) {
    return (
      <button type="button" className={`${outlineButton} h-9 px-3`} disabled>
        <Loader2 className="h-4 w-4 animate-spin" />
        加载任务
      </button>
    );
  }

  if (tasks.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      {tasks.map((task) => (
        <Button
          type="button"
          key={textValue(task.id)}
          size="sm"
          onClick={() => openTask(task)}
        >
          <Plus className="h-4 w-4" />
          {workTaskName(task)}
        </Button>
      ))}
    </div>
  );
}

export function ShowCrmWorkCustomerTable({ item, store }: WorkNodeProps) {
  const [customers, setCustomers] = useState<WorkCustomer[]>([]);
  const [filters, setFilters] = useState<WorkSearchFilters>(
    emptyWorkSearchFilters,
  );
  const [activeFilters, setActiveFilters] = useState<WorkSearchFilters>(
    emptyWorkSearchFilters,
  );
  const [loading, setLoading] = useState(false);
  const mode = workCustomerModeFromNode(item);
  const modeConfig = workCustomerModeConfig[mode];

  const loadCustomers = useCallback(async () => {
    setLoading(true);
    try {
      const payload = await workApi<{
        list?: WorkCustomer[];
        customers?: WorkCustomer[];
        data?: WorkCustomer[];
      }>(`/crm/work/customers${workCustomerQuery(activeFilters, mode)}`);
      const list = payload.list || payload.customers || payload.data || [];
      setCustomers(Array.isArray(list) ? list : []);
    } catch (error) {
      toast.error(errorMessage(error, "客户列表加载失败"));
    } finally {
      setLoading(false);
    }
  }, [activeFilters, mode]);

  useEffect(() => {
    loadCustomers();
  }, [loadCustomers]);

  useEffect(() => {
    const handler = () => loadCustomers();
    window.addEventListener(workRefreshEvent, handler);
    return () => window.removeEventListener(workRefreshEvent, handler);
  }, [loadCustomers]);

  const workItems = useMemo(() => buildWorkItems(customers), [customers]);

  const submitSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setActiveFilters(filters);
  };

  const resetSearch = () => {
    const emptyFilters = emptyWorkSearchFilters();
    setFilters(emptyFilters);
    setActiveFilters(emptyFilters);
  };

  return (
    <>
      <div className="flex flex-col gap-5">
        <form
          onSubmit={submitSearch}
          className="flex flex-wrap items-center gap-2.5"
        >
          {workSearchFields.map((field) => (
            <label key={field.key} className="shrink-0">
              <span className="sr-only">{field.placeholder}</span>
              <Input
                className={field.className}
                value={filters[field.key]}
                onChange={(event) =>
                  setFilters((current) => ({
                    ...current,
                    [field.key]: event.target.value,
                  }))
                }
                placeholder={field.placeholder}
              />
            </label>
          ))}
          <Button type="submit" size="sm" disabled={loading}>
            搜索
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={resetSearch}
            disabled={loading}
          >
            重置
          </Button>
        </form>

        <div className="md:hidden">
          <WorkItemCardList
            items={workItems}
            loading={loading}
            emptyTitle={modeConfig.emptyTitle}
            emptyDescription={modeConfig.emptyDescription}
            mode={mode}
            store={store}
          />
        </div>

        <div className="hidden overflow-hidden rounded-md border bg-background md:block">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[1120px] border-collapse text-sm">
              <thead className="bg-muted/40">
                <tr className="border-b">
                  <WorkTableHead>客户编号</WorkTableHead>
                  <WorkTableHead>姓名</WorkTableHead>
                  <WorkTableHead>手机号</WorkTableHead>
                  <WorkTableHead>微信号</WorkTableHead>
                  <WorkTableHead>资产编号</WorkTableHead>
                  <WorkTableHead>资产名称</WorkTableHead>
                  <WorkTableHead>状态</WorkTableHead>
                  <WorkTableHead className="text-center">操作</WorkTableHead>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr>
                    <td colSpan={8} className="px-6 py-16">
                      <WorkStatusState
                        icon="loading"
                        title="正在加载"
                        description="请稍候，正在同步最新数据"
                      />
                    </td>
                  </tr>
                ) : workItems.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="px-6 py-16">
                      <WorkStatusState
                        icon="empty"
                        title={modeConfig.emptyTitle}
                        description={modeConfig.emptyDescription}
                      />
                    </td>
                  </tr>
                ) : (
                  workItems.map((item) => (
                    <WorkItemTableRow
                      key={item.id}
                      item={item}
                      mode={mode}
                      store={store}
                    />
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </>
  );
}

function WorkTableHead({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return <th className={`${workTableHeadClass} ${className}`}>{children}</th>;
}

function WorkTableCell({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return <td className={`${workTableCellClass} ${className}`}>{children}</td>;
}

function WorkItemTableRow({
  item,
  mode,
  store,
}: {
  item: WorkItem;
  mode: WorkCustomerMode;
  store?: StoreLike;
}) {
  const { customer, asset } = item;
  const openDetail = () => openWorkDetail(customer, store, asset);

  return (
    <tr className="border-b last:border-b-0">
      <WorkTableCell className="text-muted-foreground">
        {workItemCustomerNo(item)}
      </WorkTableCell>
      <WorkTableCell>
        <button
          type="button"
          className="max-w-[180px] truncate text-left font-medium"
          onClick={openDetail}
        >
          {workCustomerTitle(customer)}
        </button>
      </WorkTableCell>
      <WorkTableCell>{workCustomerPhone(customer)}</WorkTableCell>
      <WorkTableCell>{displayText(customer.wechat)}</WorkTableCell>
      <WorkTableCell className="text-muted-foreground">
        {workItemAssetNo(item)}
      </WorkTableCell>
      <WorkTableCell className="font-medium">
        {asset ? (
          assetTitle(asset)
        ) : (
          <span className="text-muted-foreground">未录入资产</span>
        )}
      </WorkTableCell>
      <WorkTableCell>{renderWorkItemStatus(item)}</WorkTableCell>
      <WorkTableCell className="text-center">
        <WorkItemActions item={item} mode={mode} store={store} />
      </WorkTableCell>
    </tr>
  );
}

function WorkItemCardList({
  items,
  loading,
  emptyTitle,
  emptyDescription,
  mode,
  store,
}: {
  items: WorkItem[];
  loading: boolean;
  emptyTitle: string;
  emptyDescription: string;
  mode: WorkCustomerMode;
  store?: StoreLike;
}) {
  if (loading) {
    return (
      <WorkStatusFrame>
        <WorkStatusState
          compact
          icon="loading"
          title="正在加载"
          description="请稍候，正在同步最新数据"
        />
      </WorkStatusFrame>
    );
  }
  if (items.length === 0) {
    return (
      <WorkStatusFrame>
        <WorkStatusState
          compact
          icon="empty"
          title={emptyTitle}
          description={emptyDescription}
        />
      </WorkStatusFrame>
    );
  }
  return (
    <div className="grid gap-3">
      {items.map((item) => {
        const { customer, asset } = item;
        const openDetail = () => openWorkDetail(customer, store, asset);
        return (
          <article key={item.id} className="rounded-md border bg-background">
            <div className="px-4 py-3">
              <div className="flex min-w-0 items-center justify-between gap-3">
                <button
                  type="button"
                  className="min-w-0 text-left"
                  onClick={openDetail}
                >
                  <div className="truncate font-medium">
                    {workCustomerTitle(customer)}
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    {[
                      workItemCustomerNo(item),
                      textValue(customer.phone),
                      textValue(customer.wechat),
                    ]
                      .filter(Boolean)
                      .join(" / ") || "-"}
                  </div>
                </button>
              </div>

              <div className="mt-3 rounded-md bg-muted/25 px-3 py-2">
                <div className="flex min-w-0 items-center justify-between gap-3">
                  <span className="truncate font-medium">
                    {asset ? assetTitle(asset) : "未录入资产"}
                  </span>
                  {renderWorkItemStatus(item)}
                </div>
                <div className="mt-1 truncate text-xs text-muted-foreground">
                  {asset
                    ? `资产编号：${workItemAssetNo(item)}`
                    : "请先补充客户资料或新增资产"}
                </div>
              </div>

              <div className="mt-3">
                <WorkItemActions item={item} mode={mode} store={store} />
              </div>
            </div>
          </article>
        );
      })}
    </div>
  );
}

function WorkStatusFrame({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-md border bg-background px-4 py-14">{children}</div>
  );
}

function WorkStatusState({
  compact = false,
  icon,
  title,
  description,
}: {
  compact?: boolean;
  icon: "loading" | "empty";
  title: string;
  description: string;
}) {
  const isLoading = icon === "loading";
  const Icon = isLoading ? Loader2 : Inbox;
  const iconClassName = `${compact ? "h-5 w-5" : "h-6 w-6"} text-muted-foreground/70 ${
    isLoading ? "animate-spin" : ""
  }`;

  return (
    <div className="mx-auto flex max-w-sm flex-col items-center justify-center text-center">
      <Icon className={iconClassName} strokeWidth={1.8} />
      <div className={compact ? "mt-2" : "mt-3"}>
        <div className="text-sm font-medium text-foreground/90">{title}</div>
        <div className="mt-1 text-xs leading-5 text-muted-foreground">
          {description}
        </div>
      </div>
    </div>
  );
}

function WorkItemActions({
  item,
  mode,
  store,
}: {
  item: WorkItem;
  mode: WorkCustomerMode;
  store?: StoreLike;
}) {
  const { customer, asset, tasks } = item;
  const openDetail = () => openWorkDetail(customer, store, asset);
  const openRecords = () => openWorkRecords(customer, store, asset);

  return (
    <div className="flex flex-wrap justify-center gap-2">
      {mode !== "pending" ? (
        <Button type="button" variant="outline" size="sm" onClick={openRecords}>
          我的记录
        </Button>
      ) : null}
      <Button type="button" variant="outline" size="sm" onClick={openDetail}>
        详情
      </Button>
      {tasks.map((task) => (
        <Button
          type="button"
          key={workTaskKey(task)}
          variant="outline"
          size="sm"
          onClick={() => openRowTask(customer, task, store, asset)}
        >
          {workTaskButtonLabel(task)}
        </Button>
      ))}
    </div>
  );
}

export function ShowCrmWorkDetail({ store }: WorkNodeProps) {
  const customer = workStoreValue<WorkCustomer | null>(
    store,
    "data.actionTarget.workDetailCustomer",
    null,
  );
  const asset = workStoreValue<WorkAsset | null>(
    store,
    "data.actionTarget.workDetailAsset",
    null,
  );

  if (!customer) {
    return <WorkEmptyText>暂无详情</WorkEmptyText>;
  }

  if (asset) {
    return (
      <WorkAssetDetailContent customer={customer} asset={asset} store={store} />
    );
  }

  return <WorkCustomerDetailContent customer={customer} store={store} />;
}

export function ShowCrmWorkRecords({ store }: WorkNodeProps) {
  const customer = workStoreValue<WorkCustomer | null>(
    store,
    "data.actionTarget.workRecordCustomer",
    null,
  );
  const asset = workStoreValue<WorkAsset | null>(
    store,
    "data.actionTarget.workRecordAsset",
    null,
  );
  const [records, setRecords] = useState<WorkOperation[]>([]);
  const [loading, setLoading] = useState(false);
  const customerID = workCustomerID(customer);
  const assetID = textValue(asset?.id);

  const loadRecords = useCallback(async () => {
    if (!customerID) {
      setRecords([]);
      return;
    }
    setLoading(true);
    try {
      const query = new URLSearchParams({
        customer_id: customerID,
        mine: "1",
      });
      if (assetID) {
        query.set("asset_id", assetID);
      }
      const data = await workApi<{ list?: WorkOperation[] }>(
        `/crm/work/operations?${query.toString()}`,
      );
      setRecords(Array.isArray(data.list) ? data.list : []);
    } catch (error) {
      toast.error(errorMessage(error, "我的记录加载失败"));
      setRecords([]);
    } finally {
      setLoading(false);
    }
  }, [assetID, customerID]);

  useEffect(() => {
    loadRecords();
  }, [loadRecords]);

  if (!customer) {
    return (
      <div className="py-8 text-sm text-muted-foreground">暂无我的记录</div>
    );
  }

  return (
    <WorkMyRecordTimeline records={records} loading={loading} store={store} />
  );
}

function WorkMyRecordTimeline({
  records,
  loading,
  store,
}: {
  records: WorkOperation[];
  loading: boolean;
  store?: StoreLike;
}) {
  if (loading) {
    return (
      <div className="flex items-center justify-center gap-2 py-20 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        正在加载我的记录
      </div>
    );
  }

  if (records.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-20 text-sm text-muted-foreground">
        <FileText
          className="h-8 w-8 text-muted-foreground/40"
          strokeWidth={1.5}
        />
        <span>暂无我的操作记录</span>
      </div>
    );
  }

  return (
    <div className="relative px-1">
      {/* Timeline vertical line */}
      <div className="absolute left-[19px] top-0 bottom-0 w-[2px] bg-border/50" />

      <div>
        {records.map((record, index) => (
          <WorkMyRecordItem
            key={`${textValue(record.id) || index}`}
            record={record}
            index={index}
            store={store}
          />
        ))}
      </div>
    </div>
  );
}

const recordDotColors = [
  { border: "border-blue-500", bg: "bg-blue-500" },
  { border: "border-emerald-500", bg: "bg-emerald-500" },
  { border: "border-amber-500", bg: "bg-amber-500" },
  { border: "border-violet-500", bg: "bg-violet-500" },
  { border: "border-rose-500", bg: "bg-rose-500" },
  { border: "border-cyan-500", bg: "bg-cyan-500" },
];

function WorkMyRecordItem({
  record,
  index,
  store,
}: {
  record: WorkOperation;
  index: number;
  store?: StoreLike;
}) {
  const color = recordDotColors[index % recordDotColors.length];
  const content = record.content || record.summary || record.remark;

  return (
    <div className="relative" style={{ marginBottom: 10 }}>
      {/* Timeline dot */}
      <div
        className={`absolute left-[11px] z-10 flex size-[18px] items-center justify-center rounded-full border-[2.5px] bg-background shadow-sm ${color.border}`}
      >
        <div className={`size-[7px] rounded-full ${color.bg}`} />
      </div>

      {/* Content */}
      <button
        type="button"
        className="ml-14 block w-full rounded-lg border border-border/40 bg-card px-5 py-4 text-left shadow-sm transition-all duration-200 hover:border-border/80 hover:shadow-md active:shadow-sm"
        onClick={() => openWorkRecordDetail(record, store)}
      >
        {/* Header: title + time */}
        <div className="flex items-start justify-between gap-4">
          <span className="min-w-0 text-sm font-semibold leading-6 text-foreground/90">
            {workRecordTitle(record)}
          </span>
          <span className="shrink-0 whitespace-nowrap text-xs leading-6 text-muted-foreground/60">
            {workRecordTime(record)}
          </span>
        </div>

        {/* Content preview */}
        {content ? (
          <p className="mt-2 text-sm leading-6 text-muted-foreground/70">
            {content}
          </p>
        ) : null}
      </button>
    </div>
  );
}

export function ShowCrmWorkRecordDetail({ store }: WorkNodeProps) {
  const record = workStoreValue<WorkOperation | null>(
    store,
    "data.actionTarget.workRecordDetail",
    null,
  );
  if (!record) {
    return <WorkEmptyText>暂无记录详情</WorkEmptyText>;
  }
  return <WorkRecordDetailContent record={record} />;
}

function WorkRecordDetailContent({ record }: { record: WorkOperation }) {
  const summaryItems = workRecordSummaryItems(record);
  const content = record.content || record.remark;
  const [previewFile, setPreviewFile] = useState<UploadFileItem | null>(null);

  return (
    <div data-crm-work-record-detail="true" className="grid gap-5">
      <style>
        {`
          [role="dialog"]:has([data-crm-work-record-detail="true"]) {
            width: min(820px, calc(100vw - 32px)) !important;
            max-width: min(820px, calc(100vw - 32px)) !important;
          }
        `}
      </style>
      {/* Summary items table or content block */}
      {summaryItems.length > 0 ? (
        <div className="overflow-hidden rounded-lg border border-border/50">
          <table className="w-full text-sm">
            <tbody>
              {summaryItems.map((item, i) => (
                <tr
                  key={textValue(item.key || item.label || i)}
                  className="border-b border-border/30 last:border-b-0"
                >
                  <td className="w-[100px] min-w-[100px] bg-muted/15 px-4 py-2.5 text-muted-foreground">
                    {displayText(item.label, "-")}
                  </td>
                  <td className="px-4 py-2.5 font-medium text-foreground/85">
                    <WorkRecordSummaryValue
                      item={item}
                      onPreviewFile={setPreviewFile}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : content ? (
        <div className="rounded-lg border border-border/40 bg-muted/10 px-4 py-3.5 text-sm leading-6 text-muted-foreground/80">
          {content}
        </div>
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground/60">
          暂无提交明细
        </div>
      )}
      <WorkTaskUploadPreviewDialog
        file={previewFile}
        onOpenChange={(open) => {
          if (!open) setPreviewFile(null);
        }}
      />
    </div>
  );
}

function WorkRecordSummaryValue({
  item,
  onPreviewFile,
}: {
  item: WorkOperationSummaryItem;
  onPreviewFile: (file: UploadFileItem) => void;
}) {
  const files = normalizeUploadItems(item.files);
  if (textValue(item.value_type) === "files" && files.length > 0) {
    return (
      <div className="space-y-2">
        {files.map((file) => (
          <div
            key={String(file.id || file.name)}
            className="flex min-w-0 items-center justify-between gap-3"
          >
            <button
              type="button"
              className="min-w-0 truncate text-left text-sm font-medium text-foreground underline-offset-4 hover:text-primary hover:underline"
              title={file.name}
              onClick={() => onPreviewFile(file)}
            >
              {file.name || "附件"}
            </button>
            <button
              type="button"
              className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              aria-label="下载附件"
              onClick={() => void downloadUploadFile(file)}
            >
              <Download className="h-4 w-4" />
            </button>
          </div>
        ))}
      </div>
    );
  }

  return <>{displayText(item.value, "-")}</>;
}

function WorkCustomerDetailContent({
  customer,
  store,
}: {
  customer: WorkCustomer;
  store?: StoreLike;
}) {
  const [operations, setOperations] = useState<WorkOperation[]>([]);
  const [loadingOperations, setLoadingOperations] = useState(false);
  const customerID = workCustomerID(customer);
  const customerTasks = customer ? workCustomerRowTasks(customer) : [];
  const [activeTab, setActiveTab] = useState<WorkDetailTab>("base");

  const refreshDetail = useCallback(async () => {
    await refreshWorkDetailTarget(store, customerID);
  }, [customerID, store]);

  useEffect(() => {
    setActiveTab("base");
  }, [customerID]);

  const loadOperations = useCallback(async () => {
    if (!customerID) {
      setOperations([]);
      return;
    }
    setLoadingOperations(true);
    try {
      const data = await workApi<{ list?: WorkOperation[] }>(
        `/crm/work/operations?customer_id=${encodeURIComponent(customerID)}`,
      );
      setOperations(Array.isArray(data.list) ? data.list : []);
    } catch (error) {
      toast.error(errorMessage(error, "操作记录加载失败"));
      setOperations([]);
    } finally {
      setLoadingOperations(false);
    }
  }, [customerID]);

  useEffect(() => {
    loadOperations();
  }, [loadOperations]);

  useEffect(() => {
    if (!customerID) {
      return undefined;
    }
    window.addEventListener(workRefreshEvent, loadOperations);
    window.addEventListener(workRefreshEvent, refreshDetail);
    return () => {
      window.removeEventListener(workRefreshEvent, loadOperations);
      window.removeEventListener(workRefreshEvent, refreshDetail);
    };
  }, [customerID, loadOperations, refreshDetail]);

  return (
    <div className="grid gap-6">
      <WorkDetailTabs activeTab={activeTab} onChange={setActiveTab} />

      <div>
        {activeTab === "base" ? (
          <div className="grid gap-8">
            <WorkCurrentTaskSection
              tasks={customerTasks}
              onOpen={(task) => openRowTask(customer, task, store)}
            />

            <WorkDetailSection title="已收集资料">
              <WorkOperationCards
                operations={operations}
                loading={loadingOperations}
                store={store}
              />
            </WorkDetailSection>
          </div>
        ) : (
          <WorkCustomerMainInfo customer={customer} />
        )}
      </div>
    </div>
  );
}

function WorkAssetDetailContent({
  customer,
  asset,
  store,
}: {
  customer: WorkCustomer;
  asset: WorkAsset;
  store?: StoreLike;
}) {
  const [operations, setOperations] = useState<WorkOperation[]>([]);
  const [loadingOperations, setLoadingOperations] = useState(false);
  const customerID = workCustomerID(customer);
  const assetID = textValue(asset?.id);
  const assetTasks = asset ? workAssetRowTasks(asset) : [];
  const [activeTab, setActiveTab] = useState<WorkDetailTab>("base");

  const refreshDetail = useCallback(async () => {
    await refreshWorkDetailTarget(store, customerID, assetID);
  }, [assetID, customerID, store]);

  useEffect(() => {
    setActiveTab("base");
  }, [assetID]);

  const loadOperations = useCallback(async () => {
    if (!customerID) {
      setOperations([]);
      return;
    }
    setLoadingOperations(true);
    try {
      const data = await workApi<{ list?: WorkOperation[] }>(
        `/crm/work/operations?customer_id=${encodeURIComponent(customerID)}`,
      );
      const list = Array.isArray(data.list) ? data.list : [];
      setOperations(
        assetID
          ? list.filter(
              (operation) => textValue(operation.asset_id) === assetID,
            )
          : list,
      );
    } catch (error) {
      toast.error(errorMessage(error, "操作记录加载失败"));
      setOperations([]);
    } finally {
      setLoadingOperations(false);
    }
  }, [assetID, customerID]);

  useEffect(() => {
    loadOperations();
  }, [loadOperations]);

  useEffect(() => {
    if (!customerID) {
      return undefined;
    }
    window.addEventListener(workRefreshEvent, loadOperations);
    window.addEventListener(workRefreshEvent, refreshDetail);
    return () => {
      window.removeEventListener(workRefreshEvent, loadOperations);
      window.removeEventListener(workRefreshEvent, refreshDetail);
    };
  }, [customerID, loadOperations, refreshDetail]);

  return (
    <div className="grid gap-6">
      <WorkDetailTabs activeTab={activeTab} onChange={setActiveTab} />

      <div>
        {activeTab === "base" ? (
          <div className="grid gap-8">
            <WorkCurrentTaskSection
              tasks={assetTasks}
              onOpen={(task) => openRowTask(customer, task, store, asset)}
            />

            <WorkDetailSection title="已收集资料">
              <WorkOperationCards
                operations={operations}
                loading={loadingOperations}
                store={store}
              />
            </WorkDetailSection>
          </div>
        ) : (
          <WorkCustomerMainInfo customer={customer} asset={asset} />
        )}
      </div>
    </div>
  );
}

type WorkDetailTab = "base" | "operations";

function WorkDetailTabs({
  activeTab,
  onChange,
}: {
  activeTab: WorkDetailTab;
  onChange: (tab: WorkDetailTab) => void;
}) {
  const tabs: Array<{ key: WorkDetailTab; label: string }> = [
    { key: "base", label: "记录" },
    { key: "operations", label: "客户信息" },
  ];

  return (
    <div className="border-b border-border/70">
      <div className="flex gap-1">
        {tabs.map((tab) => (
          <button
            type="button"
            key={tab.key}
            className={`border-b-2 px-1.5 py-3 text-sm font-medium transition-colors ${
              activeTab === tab.key
                ? "border-primary text-foreground"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => onChange(tab.key)}
          >
            {tab.label}
          </button>
        ))}
      </div>
    </div>
  );
}

function WorkDetailSection({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="grid gap-4 border-t border-border/60 pt-6 first:border-t-0 first:pt-0">
      <h3 className="text-[15px] font-semibold leading-6">{title}</h3>
      <div>{children}</div>
    </section>
  );
}

function compactWorkDetailItems(
  items: Array<[string, unknown]>,
): Array<[string, ReactNode]> {
  return items
    .map(([label, value]) => {
      const text = textValue(value);
      return text ? ([label, text] as [string, ReactNode]) : null;
    })
    .filter(Boolean) as Array<[string, ReactNode]>;
}

function WorkDetailGrid({ items }: { items: Array<[string, ReactNode]> }) {
  if (items.length === 0) {
    return <WorkEmptyText>暂无已收集信息</WorkEmptyText>;
  }

  return (
    <dl className="grid gap-x-8 gap-y-5 sm:grid-cols-2 xl:grid-cols-3">
      {items.map(([label, value]) => (
        <div key={label} className="grid min-w-0 gap-1.5 text-sm">
          <dt className="text-[13px] leading-5 text-muted-foreground">
            {label}
          </dt>
          <dd className="min-w-0 break-words text-[15px] font-medium leading-6 text-foreground">
            {value}
          </dd>
        </div>
      ))}
    </dl>
  );
}

function WorkCurrentTaskSection({
  tasks,
  onOpen,
}: {
  tasks: WorkTask[];
  onOpen: (task: WorkTask) => void;
}) {
  return (
    <WorkDetailSection title="当前应做">
      <div className="flex flex-wrap items-center justify-between gap-3">
        {tasks.length > 0 ? (
          <>
            <div className="text-sm leading-6 text-muted-foreground">
              {workPendingTaskSummary(tasks)}
            </div>
            <WorkTaskButtons tasks={tasks} onOpen={onOpen} />
          </>
        ) : (
          <WorkEmptyText>暂无待处理任务</WorkEmptyText>
        )}
      </div>
    </WorkDetailSection>
  );
}

function WorkOperationCards({
  operations,
  loading,
  store,
}: {
  operations: WorkOperation[];
  loading: boolean;
  store?: StoreLike;
}) {
  if (loading) {
    return (
      <div className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        正在加载资料记录
      </div>
    );
  }

  if (operations.length === 0) {
    return <WorkEmptyText>暂无已收集资料</WorkEmptyText>;
  }

  return (
    <div className="grid gap-3">
      {operations.map((operation, index) => (
        <WorkOperationCard
          key={`${textValue(operation.id) || index}`}
          operation={operation}
          store={store}
        />
      ))}
    </div>
  );
}

function WorkOperationCard({
  operation,
  store,
}: {
  operation: WorkOperation;
  store?: StoreLike;
}) {
  const content = workOperationDescription(operation);
  return (
    <button
      type="button"
      className="block w-full rounded-lg border border-border/40 bg-card px-5 py-4 text-left shadow-sm transition-all duration-200 hover:border-border/80 hover:shadow-md active:shadow-sm"
      onClick={() => openWorkRecordDetail(operation, store)}
    >
      <div className="flex items-start justify-between gap-4">
        <span className="min-w-0 text-sm font-semibold leading-6 text-foreground/90">
          {workOperationTitle(operation)}
        </span>
        <span className="shrink-0 whitespace-nowrap text-xs leading-6 text-muted-foreground/60">
          {formatWorkDate(operation.created_at || operation.create_time)}
        </span>
      </div>
      {content ? (
        <p className="mt-2 text-sm leading-6 text-muted-foreground/70">
          {content}
        </p>
      ) : null}
      <div className="mt-3 text-xs leading-5 text-muted-foreground/70">
        操作人：
        {displayText(
          operation.operator_name || operation["operator_staff.name"],
        )}
      </div>
    </button>
  );
}

function WorkCustomerMainInfo({
  customer,
  asset,
}: {
  customer: WorkCustomer;
  asset?: WorkAsset;
}) {
  return (
    <div className="grid gap-8">
      <WorkDetailSection title="客户信息">
        <WorkDetailGrid items={workCustomerMainDetailItems(customer)} />
      </WorkDetailSection>

      <WorkDetailSection title="资产信息">
        {asset ? (
          <WorkDetailGrid items={workAssetMainDetailItems(asset)} />
        ) : (
          <WorkEmptyText>暂无资产信息</WorkEmptyText>
        )}
      </WorkDetailSection>
    </div>
  );
}

function workCustomerMainDetailItems(
  customer: WorkCustomer,
): Array<[string, ReactNode]> {
  return compactWorkDetailItems([
    ["客户编号", workCustomerNo(customer)],
    ["姓名", workCustomerName(customer)],
    ["手机号", workCustomerPhone(customer)],
    ["微信", displayText(customer.wechat)],
    ["性别", displayText(customer.gender_name || customer.gender)],
    ["来源", displayText(customer.source_name || customer.source)],
    ["渠道", displayText(customer.channel_name || customer.channel)],
    ["客户等级", displayText(customer.level_name || customer.customer_level)],
    ["当前状态", workStatusName(customer)],
    ["创建时间", formatWorkDate(customer.created_at || customer.create_time)],
  ]);
}

function workAssetMainDetailItems(
  asset: WorkAsset,
): Array<[string, ReactNode]> {
  return compactWorkDetailItems([
    ["资产名称", assetTitle(asset)],
    ["资产编号", workAssetNo(asset)],
    ["资产状态", textValue(asset.asset_status_name)],
    ["当前状态", workStatusName(asset)],
    ["备注", textValue(asset.remark)],
  ]);
}

function WorkTaskButtons({
  tasks,
  onOpen,
}: {
  tasks: WorkTask[];
  onOpen: (task: WorkTask) => void;
}) {
  if (tasks.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-wrap gap-2">
      {tasks.map((task) => (
        <Button
          type="button"
          key={workTaskKey(task)}
          variant="outline"
          size="sm"
          onClick={() => onOpen(task)}
        >
          {workTaskButtonLabel(task)}
        </Button>
      ))}
    </div>
  );
}

function WorkEmptyText({ children }: { children: ReactNode }) {
  return <div className="text-sm text-muted-foreground">{children}</div>;
}

export function ShowCrmWorkTaskForm({ store }: WorkNodeProps) {
  const task = workStoreValue<WorkTask | null>(
    store,
    "data.actionTarget.workTask",
    null,
  );
  const customer = workStoreValue<WorkCustomer | null>(
    store,
    "data.actionTarget.workTaskCustomer",
    null,
  );
  const asset = workStoreValue<WorkAsset | null>(
    store,
    "data.actionTarget.workTaskAsset",
    null,
  );
  const [submitting, setSubmitting] = useState(false);
  const [aiFilling, setAiFilling] = useState(false);
  const contentRef = useRef<HTMLDivElement | null>(null);

  const close = useCallback(() => {
    setWorkModalOpen(store, "dialog.workTask", false);
  }, [store]);

  const submit = useCallback(async () => {
    if (!task) return false;
    if (submitting) return false;
    clearCurrentWorkTaskFormErrors(store);
    if (!validateCurrentWorkTaskForm(store)) return false;

    setSubmitting(true);
    try {
      await workApi("/crm/work/execute", {
        method: "POST",
        body: JSON.stringify({
          task_id: task.id,
          todo_id: positiveTextID(task.todo_id) || undefined,
          customer_id: workCustomerID(customer),
          asset_id: workAssetID(asset),
          values: collectWorkTaskSubmitValues(store),
        }),
      });
      toast.success("保存成功");
      notifyWorkRefresh();
      close();
    } catch (error) {
      const message = errorMessage(error);
      applyWorkTaskSubmitError(store, message);
      toast.error(message);
      return false;
    } finally {
      setSubmitting(false);
    }
    return true;
  }, [asset, close, customer, store, submitting, task]);

  const aiFill = useCallback(async () => {
    if (!task || aiFilling || submitting) return;

    setAiFilling(true);
    try {
      const payload = await workApi<WorkAIFillResponse>("/crm/work/ai_fill", {
        method: "POST",
        body: JSON.stringify({
          task_id: task.id,
          todo_id: positiveTextID(task.todo_id) || undefined,
          customer_id: workCustomerID(customer),
          asset_id: workAssetID(asset),
          values: collectWorkTaskSubmitValues(store),
        }),
      });
      const count = applyWorkTaskAIFillValues(store, payload.values || {});
      if (count === 0) {
        toast.info("AI 没有返回可填写的字段");
        return;
      }
      toast.success(`AI 已填写 ${count} 个字段`);
    } catch (error) {
      toast.error(errorMessage(error, "AI 填写失败"));
    } finally {
      setAiFilling(false);
    }
  }, [aiFilling, asset, customer, store, submitting, task]);

  useEffect(() => {
    const form = contentRef.current?.closest("form");
    if (!form) return undefined;

    const handleSubmit = (event: Event) => {
      event.preventDefault();
      event.stopPropagation();
      void submit();
    };

    form.addEventListener("submit", handleSubmit);
    return () => {
      form.removeEventListener("submit", handleSubmit);
    };
  }, [submit]);

  if (!task) return null;
  const canAIFill = workTaskCanAIFill(task);

  return (
    <div ref={contentRef} className="contents">
      {canAIFill ? (
        <div className="mb-4 flex justify-end">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void aiFill()}
            disabled={aiFilling || submitting}
          >
            {aiFilling ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Bot className="h-4 w-4" />
            )}
            {aiFilling ? "AI填写中" : "AI填写"}
          </Button>
        </div>
      ) : null}
      {submitting ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          正在保存
        </div>
      ) : null}
    </div>
  );
}

export function ShowCrmWorkTaskUpload({
  item,
  value,
  setValue,
  store,
}: WorkNodeProps & {
  value?: unknown;
  setValue?: (value: unknown) => void;
}) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadMessage, setUploadMessage] = useState("");
  const [uploadProgress, setUploadProgress] =
    useState<WorkTaskUploadProgress | null>(null);
  const [localFiles, setLocalFiles] = useState<UploadFileItem[]>([]);
  const [previewFile, setPreviewFile] = useState<UploadFileItem | null>(null);
  const relationPath = inferWorkRelationPath(item?.value);
  const relationValue =
    store && relationPath
      ? getStoreValueByPath(store, relationPath)
      : undefined;
  const meta = resolveWorkTaskUploadMeta(item?.meta);
  const files = normalizeWorkTaskUploadItems(relationValue, value, localFiles);
  const remainingCount = Math.max(meta.maxCount - files.length, 0);
  const disabled = uploading || remainingCount <= 0;

  const syncFiles = useCallback(
    (nextFiles: UploadFileItem[]) => {
      setLocalFiles(nextFiles);
      setValue?.(nextFiles.map((file) => file.id));
      if (store && relationPath) {
        setStoreValueByPath(store, relationPath, nextFiles);
      }
    },
    [relationPath, setValue, store],
  );

  const handleChooseFiles = useCallback(
    async (event: ChangeEvent<HTMLInputElement>) => {
      const selected = Array.from(event.target.files || []);
      event.target.value = "";
      if (selected.length === 0 || uploading) return;

      const nextSelected = selected.slice(
        0,
        Math.max(meta.maxCount - files.length, 0),
      );
      if (nextSelected.length === 0) {
        setUploadMessage(`最多只能上传 ${meta.maxCount} 个文件。`);
        return;
      }

      setUploading(true);
      setUploadMessage("");
      setUploadProgress({
        fileName: nextSelected[0]?.name || "",
        percent: 0,
        currentIndex: 1,
        total: nextSelected.length,
      });
      try {
        let nextFiles = [...files];
        for (let index = 0; index < nextSelected.length; index += 1) {
          const file = nextSelected[index];
          if (!file) continue;
          const currentIndex = index + 1;
          setUploadProgress({
            fileName: file.name,
            percent: resolveWorkUploadOverallProgress(
              index,
              0,
              nextSelected.length,
            ),
            currentIndex,
            total: nextSelected.length,
          });
          const uploaded = await uploadFileByRule(meta.ruleId, file, {
            kind: meta.kind,
            bizKey: meta.bizKey,
            bizName: meta.bizName,
            onProgress: (loaded, total) => {
              setUploadProgress({
                fileName: file.name,
                percent: resolveWorkUploadOverallProgress(
                  index,
                  resolveWorkUploadFileProgress(loaded, total),
                  nextSelected.length,
                ),
                currentIndex,
                total: nextSelected.length,
              });
            },
          });
          const uploadedFile = normalizeUploadItems(uploaded)[0] || uploaded;
          nextFiles = [...nextFiles, uploadedFile];
        }
        syncFiles(nextFiles);
      } catch (uploadError) {
        setUploadMessage(errorMessage(uploadError) || "上传失败");
      } finally {
        setUploading(false);
        setUploadProgress(null);
      }
    },
    [files, meta, syncFiles, uploading],
  );

  const removeFile = useCallback(
    (targetID: UploadFileItem["id"]) => {
      syncFiles(files.filter((file) => String(file.id) !== String(targetID)));
    },
    [files, syncFiles],
  );

  return (
    <div className="w-full space-y-3">
      <input
        ref={inputRef}
        type="file"
        className="hidden"
        multiple
        onChange={handleChooseFiles}
      />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Button
          type="button"
          variant="outline"
          disabled={disabled}
          onClick={() => inputRef.current?.click()}
        >
          {uploading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Upload className="h-4 w-4" />
          )}
          {uploading ? "上传中..." : "上传文件"}
        </Button>
        <span className="text-xs text-muted-foreground">
          已选择 {files.length} 个文件
        </span>
      </div>
      {uploading && uploadProgress ? (
        <div className="rounded-lg border border-border/70 bg-muted/20 px-3 py-2">
          <div className="flex items-center justify-between gap-3 text-xs">
            <span
              className="min-w-0 truncate text-muted-foreground"
              title={uploadProgress.fileName}
            >
              正在上传 {uploadProgress.fileName}
              {uploadProgress.total > 1
                ? `（${uploadProgress.currentIndex}/${uploadProgress.total}）`
                : ""}
            </span>
            <span className="shrink-0 font-medium text-foreground">
              {uploadProgress.percent}%
            </span>
          </div>
          <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-primary transition-all duration-200"
              style={{ width: `${uploadProgress.percent}%` }}
            />
          </div>
        </div>
      ) : null}
      <div className="overflow-hidden rounded-xl border border-border/70 bg-background text-sm shadow-xs">
        <div
          className="grid border-b bg-muted/30"
          style={{ gridTemplateColumns: workUploadGridColumns }}
        >
          <div className="flex h-12 min-w-0 items-center px-4 font-medium text-muted-foreground">
            文件名
          </div>
          <div className="flex h-12 items-center whitespace-nowrap px-4 font-medium text-muted-foreground">
            大小
          </div>
          <div className="flex h-12 items-center whitespace-nowrap px-4 font-medium text-muted-foreground">
            操作
          </div>
        </div>
        {files.length === 0 ? (
          <div className="py-6 text-center text-sm text-muted-foreground">
            暂无附件
          </div>
        ) : (
          files.map((file) => (
            <div
              key={String(file.id)}
              className="grid border-b last:border-b-0"
              style={{ gridTemplateColumns: workUploadGridColumns }}
            >
              <div className="flex min-w-0 items-center overflow-hidden px-4 py-3">
                <button
                  type="button"
                  className="block w-full min-w-0 overflow-hidden truncate whitespace-nowrap text-left text-sm font-medium text-foreground underline-offset-4 hover:text-primary hover:underline"
                  title={file.name}
                  onClick={() => setPreviewFile(file)}
                >
                  {file.name}
                </button>
              </div>
              <div className="flex items-center whitespace-nowrap px-4 py-3 text-sm">
                {formatUploadSize(Number(file.size || 0))}
              </div>
              <div className="flex items-center px-4 py-3">
                <div
                  className="flex items-center gap-1"
                  style={{ flexWrap: "nowrap" }}
                >
                  <button
                    type="button"
                    className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-foreground transition-colors hover:bg-muted"
                    aria-label="下载附件"
                    onClick={() => void downloadUploadFile(file)}
                  >
                    <Download className="h-4 w-4" />
                  </button>
                  <button
                    type="button"
                    className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-destructive transition-colors hover:bg-muted hover:text-destructive disabled:cursor-not-allowed disabled:opacity-50"
                    aria-label="删除附件"
                    disabled={uploading}
                    onClick={() => removeFile(file.id)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
      {uploadMessage ? (
        <p className="text-xs text-destructive">{uploadMessage}</p>
      ) : null}
      <WorkTaskUploadPreviewDialog
        file={previewFile}
        onOpenChange={(open) => {
          if (!open) setPreviewFile(null);
        }}
      />
    </div>
  );
}

function resolveWorkTaskUploadMeta(
  meta?: Record<string, unknown>,
): WorkTaskUploadMeta {
  return {
    ruleId: Number(meta?.ruleId || 6),
    kind: textValue(meta?.kind) || "file",
    maxCount: Number(meta?.maxCount || 10),
    bizKey: textValue(meta?.bizKey) || "crm.work",
    bizName: textValue(meta?.bizName) || "CRM工作台",
  };
}

function normalizeWorkTaskUploadItems(
  relationValue: unknown,
  value: unknown,
  localFiles: UploadFileItem[],
): UploadFileItem[] {
  if (localFiles.length > 0) return localFiles;

  const relationItems = normalizeUploadItems(relationValue);
  if (relationItems.length > 0) return relationItems;

  const valueItems = normalizeUploadItems(value);
  if (valueItems.length > 0) return valueItems;

  if (Array.isArray(value)) {
    return value
      .filter((current) => current && typeof current === "object")
      .map((current) => normalizeUploadItems(current)[0])
      .filter((file): file is UploadFileItem => Boolean(file));
  }

  return [];
}

function resolveWorkUploadFileProgress(loaded: number, total: number): number {
  const totalValue = Number(total || 0);
  if (!Number.isFinite(totalValue) || totalValue <= 0) {
    return Number(loaded || 0) > 0 ? 100 : 0;
  }
  return clampWorkUploadPercent((Number(loaded || 0) / totalValue) * 100);
}

function resolveWorkUploadOverallProgress(
  completedFileCount: number,
  currentFilePercent: number,
  totalFileCount: number,
): number {
  const total = Math.max(Number(totalFileCount || 0), 1);
  const completed = Math.max(
    0,
    Math.min(Number(completedFileCount || 0), total),
  );
  const current = clampWorkUploadPercent(currentFilePercent) / 100;
  return clampWorkUploadPercent(((completed + current) / total) * 100);
}

function clampWorkUploadPercent(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, Math.round(value)));
}

function WorkTaskUploadPreviewDialog({
  file,
  onOpenChange,
}: {
  file: UploadFileItem | null;
  onOpenChange: (open: boolean) => void;
}) {
  const [imageFailed, setImageFailed] = useState(false);
  const previewKind = resolveWorkTaskUploadPreviewKind(file);
  const previewUrl = workTaskUploadPreviewUrl(file);
  const canPreviewImage = previewKind === "image" && previewUrl && !imageFailed;

  useEffect(() => {
    setImageFailed(false);
  }, [file?.id, previewUrl]);

  return (
    <Dialog open={Boolean(file)} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[88vh] max-h-[88vh] max-w-5xl flex-col gap-0 overflow-hidden p-0">
        <DialogHeader className="border-b px-6 py-4">
          <DialogTitle>{file?.name || "资源详情"}</DialogTitle>
          <DialogDescription>
            可查看当前选中资源，支持图片预览与附件下载。
          </DialogDescription>
        </DialogHeader>
        <div className="flex min-h-0 flex-1 flex-col">
          <div className="flex min-h-0 flex-1 items-center justify-center overflow-hidden bg-muted/30 px-6 py-6">
            {canPreviewImage ? (
              <img
                src={previewUrl}
                alt={file?.name || "附件预览"}
                className="max-h-full max-w-full rounded-xl object-contain shadow-sm"
                onError={() => setImageFailed(true)}
              />
            ) : (
              <div className="flex w-full max-w-2xl flex-col items-center gap-4 rounded-xl border bg-background px-6 py-8 text-center shadow-sm">
                <FileText className="h-10 w-10 text-muted-foreground" />
                <div className="max-w-full space-y-1">
                  <div className="truncate text-sm font-medium">
                    {file?.name || "未选择资源"}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    当前文件暂不支持直接预览，可以下载后查看。
                  </div>
                </div>
              </div>
            )}
          </div>
          <div className="flex flex-col gap-4 border-t bg-background px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0 flex-1 space-y-1">
              <div className="truncate text-sm font-medium">
                {file?.name || "未选择资源"}
              </div>
              <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                <span>{formatUploadSize(Number(file?.size || 0))}</span>
                {file?.ext ? (
                  <span>
                    {textValue(file.ext).replace(/^\./, "").toUpperCase()}
                  </span>
                ) : null}
              </div>
            </div>
            {file ? (
              <Button
                type="button"
                onClick={() => void downloadUploadFile(file)}
              >
                <Download className="h-4 w-4" />
                下载
              </Button>
            ) : null}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function workTaskUploadPreviewUrl(file?: UploadFileItem | null): string {
  const directUrl = textValue(
    file?.thumbnail || file?.url || file?.open_url || file?.download,
  );
  if (directUrl) return directUrl;

  const fileID = positiveTextID(file?.id);
  if (fileID) {
    return `/front/upload/open?id=${encodeURIComponent(fileID)}`;
  }
  return "";
}

function resolveWorkTaskUploadPreviewKind(
  file?: UploadFileItem | null,
): string {
  const resourceKind = resolveResourcePreviewKind(file);
  if (resourceKind) return resourceKind;
  if (!file) return "";

  const kind = textValue(file.kind).toLowerCase();
  if (kind === "image") return "image";

  const mime = textValue(file.mime).toLowerCase();
  if (mime.startsWith("image/")) return "image";

  const ext = workUploadFileExtension(file);
  return workImageExtensions.has(ext) ? "image" : "";
}

function workUploadFileExtension(file?: UploadFileItem | null): string {
  const explicitExt = normalizeWorkUploadExtension(file?.ext);
  if (explicitExt) return explicitExt;

  const name = textValue(file?.name).split(/[?#]/)[0];
  const dotIndex = name.lastIndexOf(".");
  if (dotIndex < 0) return "";
  return normalizeWorkUploadExtension(name.slice(dotIndex + 1));
}

function normalizeWorkUploadExtension(value: unknown): string {
  return textValue(value).replace(/^\./, "").toLowerCase();
}

function inferWorkRelationPath(valuePath?: string): string {
  const path = textValue(valuePath);
  if (!path) return "";
  if (path.endsWith("_ids")) return `${path.slice(0, -4)}s`;
  if (path.endsWith("_id")) return path.slice(0, -3);
  return "";
}

function validateCurrentWorkTaskForm(store: StoreLike | undefined): boolean {
  const validateForm = currentWorkStoreState(store)?.validateForm;
  return typeof validateForm === "function" ? validateForm() : true;
}

function collectWorkTaskSubmitValues(
  store: StoreLike | undefined,
): Record<string, unknown> {
  const formValues = workStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    {},
  );
  const fieldMap = workStoreValue<Record<string, string>>(
    store,
    workTaskFieldMapPath,
    {},
  );
  return Object.entries(fieldMap).reduce<Record<string, unknown>>(
    (values, [formKey, rawKey]) => {
      values[rawKey] = formValues[formKey];
      return values;
    },
    {},
  );
}

function applyWorkTaskAIFillValues(
  store: StoreLike | undefined,
  values: Record<string, unknown>,
): number {
  if (!store || Object.keys(values).length === 0) return 0;

  const fieldMap = workStoreValue<Record<string, string>>(
    store,
    workTaskFieldMapPath,
    {},
  );
  const formValues = {
    ...workStoreValue<Record<string, unknown>>(store, workTaskFormDataPath, {}),
  };
  let count = 0;

  for (const [formKey, rawKey] of Object.entries(fieldMap)) {
    if (!Object.prototype.hasOwnProperty.call(values, rawKey)) {
      continue;
    }
    if (!workAIFillShouldApply(formValues[formKey])) {
      continue;
    }
    formValues[formKey] = values[rawKey];
    count += 1;
  }

  if (count > 0) {
    setWorkStoreValue(store, workTaskFormDataPath, formValues);
  }
  return count;
}

function workAIFillShouldApply(value: unknown): boolean {
  if (value === null || value === undefined) return true;
  if (typeof value === "string") return value.trim() === "";
  if (Array.isArray(value)) return value.length === 0;
  return false;
}

function workTaskCanAIFill(task: WorkTask): boolean {
  if (!workBotComponentAvailable()) return false;
  if (!workTaskShouldRenderFields(task)) return false;
  return (task.form?.fields || []).some((field) => !workTaskFieldIsUpload(field));
}

function workBotComponentAvailable(): boolean {
  if (workAssistantFormFillComponentAvailable()) return true;
  const nodes = workFrontPluginNodes();
  return Boolean(
    nodes["show-agent"] ||
      nodes["show-team-workspace"] ||
      nodes["show-stream-request"],
  );
}

function workAssistantFormFillComponentAvailable(): boolean {
  const getCompatModule = (window as unknown as {
    DeverFront?: {
      sdk?: {
        getCompatModule?: (path: string) => Record<string, unknown>;
      };
    };
  }).DeverFront?.sdk?.getCompatModule;
  if (typeof getCompatModule !== "function") return false;

  try {
    return Boolean(
      getCompatModule("@/components/assistant/form-actions")
        ?.AssistantContextFormFillButton,
    );
  } catch {
    return false;
  }
}

function workFrontPluginNodes(): Record<string, unknown> {
  const front = (window as unknown as {
    DeverFront?: {
      plugins?: unknown;
      nodes?: Record<string, unknown>;
      sdk?: {
        plugins?: unknown;
        nodes?: Record<string, unknown>;
      };
    };
  }).DeverFront;
  return {
    ...workPluginNodesFromAny(front?.nodes),
    ...workPluginNodesFromAny(front?.sdk?.nodes),
    ...workPluginNodesFromAny(front?.plugins),
    ...workPluginNodesFromAny(front?.sdk?.plugins),
    ...workPluginNodesFromRuntime(),
  };
}

function workPluginNodesFromAny(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object") return {};
  if (Array.isArray(value)) {
    return value.reduce<Record<string, unknown>>(
      (nodes, item) => {
        const nodeName =
          typeof item === "string" || typeof item === "number"
            ? textValue(item)
            : "";
        return nodeName
          ? { ...nodes, [nodeName]: true }
          : { ...nodes, ...workPluginNodesFromAny(item) };
      },
      {},
    );
  }

  const mapped = value as Record<string, unknown>;
  const directNodes = mapped.nodes;
  if (directNodes) {
    return Array.isArray(directNodes)
      ? workPluginNodesFromAny(directNodes)
      : typeof directNodes === "object"
        ? (directNodes as Record<string, unknown>)
        : {};
  }

  const nested = Object.values(mapped).reduce<Record<string, unknown>>(
    (nodes, item) => ({ ...nodes, ...workPluginNodesFromAny(item) }),
    {},
  );
  return Object.keys(nested).length > 0 ? nested : mapped;
}

function workPluginNodesFromRuntime(): Record<string, unknown> {
  const runtime = (window as unknown as {
    appRuntime?: {
      runtime?: {
        plugins?: Array<{ name?: string } | string>;
      };
    };
  }).appRuntime;
  const plugins = runtime?.runtime?.plugins;
  if (!Array.isArray(plugins)) return {};
  const hasBot = plugins.some((plugin) =>
    typeof plugin === "string"
      ? plugin === "bot"
      : textValue(plugin?.name) === "bot",
  );
  return hasBot ? { "show-agent": true } : {};
}

function clearCurrentWorkTaskFormErrors(store: StoreLike | undefined) {
  setCurrentWorkTaskFormErrors(store, {});
}

function applyWorkTaskSubmitError(
  store: StoreLike | undefined,
  message: string,
) {
  const errorField = workTaskSubmitErrorField(message);
  if (!errorField) return;

  const errorKey = currentWorkTaskFormErrorKey(store, errorField);
  if (!errorKey) return;

  setCurrentWorkTaskFormErrors(store, {
    [errorKey]: message,
  });
}

function setCurrentWorkTaskFormErrors(
  store: StoreLike | undefined,
  formErrors: Record<string, string>,
) {
  updateWorkStoreErrors(store, (errors) => ({
    ...withoutCurrentWorkTaskFormErrors(errors),
    ...formErrors,
  }));
}

function withoutCurrentWorkTaskFormErrors(
  errors: Record<string, string>,
): Record<string, string> {
  return Object.entries(errors).reduce<Record<string, string>>(
    (result, [key, message]) => {
      if (!key.startsWith("workTaskForm.")) {
        result[key] = message;
      }
      return result;
    },
    {},
  );
}

function currentWorkTaskFormErrorKey(
  store: StoreLike | undefined,
  field: string,
): string {
  const fieldMap = workStoreValue<Record<string, string>>(
    store,
    workTaskFieldMapPath,
    {},
  );
  const matched = Object.entries(fieldMap).find(
    ([, rawKey]) => workTaskRawMainField(rawKey) === field,
  );
  return matched ? `workTaskForm.${matched[0]}` : "";
}

function workTaskRawMainField(rawKey: string): string {
  const normalized = textValue(rawKey);
  return normalized.startsWith("main:")
    ? normalized.slice("main:".length)
    : normalized;
}

function workTaskSubmitErrorField(message: string): string {
  if (message.includes("手机号") || message.includes("phone")) return "phone";
  if (message.includes("微信") || message.includes("wechat")) return "wechat";
  if (message.includes("身份证") || message.includes("id_card"))
    return "id_card";
  return "";
}

function workOptionID(option: WorkFieldOption): string {
  return (
    textValue(option.id) ||
    textValue(option.value) ||
    textValue(option.name) ||
    textValue(option.label)
  );
}

function workOptionValue(option: WorkFieldOption): string {
  return (
    textValue(option.value) ||
    textValue(option.id) ||
    textValue(option.name) ||
    textValue(option.label)
  );
}

function workOptionLabel(option: WorkFieldOption): string {
  return displayText(option.label || option.name || option.value || option.id);
}

function workFieldKey(field: WorkFormField): string {
  const mainField = textValue(field.main_field);
  if (mainField) return `main:${mainField}`;
  const dataFieldID = positiveTextID(field.data_field_id);
  if (dataFieldID) return `data:${dataFieldID}`;
  return (
    textValue(field.field_key) ||
    textValue(field.field_name) ||
    textValue(field.field) ||
    textValue(field.name) ||
    textValue(field.id)
  );
}

function formatFormValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "object") return "";
  return String(value);
}

function workFieldInitialValue(
  field: WorkFormField,
  customer?: WorkCustomer | null,
  asset?: WorkAsset | null,
  renderType?: string,
): unknown {
  const rawValue = workEntityFieldValue(field, customer, asset);
  if (renderType === "show-crm-work-task-upload") return rawValue;

  const value = formatFormValue(rawValue);
  const options = Array.isArray(field.options) ? field.options : [];
  if (!value || options.length === 0) return value;

  const exactOption = options.find((option) => workOptionID(option) === value);
  if (exactOption) return workOptionID(exactOption);

  const valueOption = options.find(
    (option) => workOptionValue(option) === value,
  );
  if (valueOption) return workOptionID(valueOption);

  const labelOption = options.find(
    (option) => workOptionLabel(option) === value,
  );
  return labelOption ? workOptionID(labelOption) : value;
}

const workMainFieldAliases: Record<string, string[]> = {
  source_id: ["source_id", "source", "customer_source_id", "source_name"],
  channel_id: ["channel_id", "channel", "customer_channel_id", "channel_name"],
  level_id: ["level_id", "customer_level_id", "customer_level", "level_name"],
  asset_status_id: ["asset_status_id", "asset_status", "asset_status_name"],
};

function workEntityValueByKeys(
  target: Record<string, unknown> | null | undefined,
  keys: string[],
): unknown {
  if (!target) return undefined;
  for (const key of keys) {
    const value = target[key];
    if (value !== undefined && value !== null && value !== "") return value;
  }
  return undefined;
}

function workEntityFieldValue(
  field: WorkFormField,
  customer?: WorkCustomer | null,
  asset?: WorkAsset | null,
): unknown {
  const mainField = textValue(field.main_field);
  if (mainField) {
    const keys = [mainField, ...(workMainFieldAliases[mainField] || [])];
    const assetValue = workEntityValueByKeys(asset, keys);
    if (assetValue !== undefined) return assetValue;
    const customerValue = workEntityValueByKeys(customer, keys);
    if (customerValue !== undefined) return customerValue;
  }
  const key = workFieldKey(field);
  if (key) {
    if (asset && asset[key] !== undefined) return asset[key];
    if (asset?.data_values && asset.data_values[key] !== undefined) {
      return asset.data_values[key];
    }
    if (customer && customer[key] !== undefined) return customer[key];
    if (customer?.data_values && customer.data_values[key] !== undefined) {
      return customer.data_values[key];
    }
  }
  if (field.default_value !== undefined) return field.default_value;
  return "";
}
