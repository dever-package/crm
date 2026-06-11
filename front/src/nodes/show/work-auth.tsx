import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import { Check, Loader2, Plus, RefreshCw, X } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { getStoreValueByPath, setStoreValueByPath } from "@/lib/store";

type StoreLike = Record<string, unknown>;

type WorkNodeProps = {
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

type WorkDecisionResult = {
  name?: string;
  label?: string;
  result_value?: string;
  value?: string;
  target_stage_id?: string | number;
};

type WorkTask = {
  id?: string | number;
  name?: string;
  task_name?: string;
  action_type?: string;
  task_action?: string;
  task_type?: string;
  trigger_type?: string;
  form_id?: string | number;
  form?: WorkForm | null;
  result_schema?: WorkDecisionResult[];
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
  title?: string;
  operation_name?: string;
  task_name?: string;
  content?: string;
  remark?: string;
  operator_name?: string;
  "operator_staff.name"?: string;
  created_at?: string;
  create_time?: string;
  [key: string]: unknown;
};

type WorkItem = {
  id: string;
  targetType: "customer" | "asset";
  customer: WorkCustomer;
  asset?: WorkAsset;
  tasks: WorkTask[];
};

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

const workRefreshEvent = "crm-work-refresh";
const workTokenKey = "crm_work_token";
const workUserKey = "crm_work_user";
const legacyWorkTokenKey = "gjj_crm_work_token";
const legacyWorkUserKey = "gjj_crm_work_user";
const legacyFrontTokenKey = "front-token:work";
const legacyFrontUserKey = "front-user:work";
const defaultWorkSiteKey = "work";
const authCookieMaxAge = 3600 * 24 * 7;

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
  { key: "customerNo", placeholder: "客户编号", className: "h-10 w-[160px] max-w-full" },
  { key: "customerName", placeholder: "姓名", className: "h-10 w-[140px] max-w-full" },
  { key: "phone", placeholder: "手机号", className: "h-10 w-[150px] max-w-full" },
  { key: "wechat", placeholder: "微信号", className: "h-10 w-[150px] max-w-full" },
  { key: "assetNo", placeholder: "资产编号", className: "h-10 w-[170px] max-w-full" },
  { key: "status", placeholder: "状态", className: "h-10 w-[140px] max-w-full" },
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

const workTableHeadClass =
  "h-12 whitespace-nowrap px-4 text-left text-sm font-medium text-muted-foreground";
const workTableCellClass =
  "h-14 whitespace-nowrap px-4 align-middle text-sm";

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
      textValue(site.subtitle) || textValue(runtime.subtitle) || "客户中心工作台",
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

function workStoreValue<T>(store: StoreLike | undefined, path: string, fallback: T): T {
  const value = store ? getStoreValueByPath(store, path) : undefined;
  return value === undefined || value === null ? fallback : (value as T);
}

function setWorkStoreValue(store: StoreLike | undefined, path: string, value: unknown) {
  if (!store) return;
  setStoreValueByPath(store, path, value);
}

function setWorkModalOpen(store: StoreLike | undefined, key: string, open: boolean) {
  const getState = (store as { getState?: () => { setPageState?: (key: string, value: boolean) => void } } | undefined)
    ?.getState;
  const state = typeof getState === "function" ? getState() : undefined;
  if (typeof state?.setPageState === "function") {
    state.setPageState(key, open);
    return;
  }
  setWorkStoreValue(store, `state.${key}`, open);
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
  return text.replace("T", " ").replace(/\.\d+Z?$/, "").slice(0, 16);
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
  return workAssetName(asset) || workAssetNo(asset) || `资产${textValue(asset?.id)}`;
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
  return action === "decision" || action === "决策";
}

function workTaskIsBooking(task: WorkTask): boolean {
  const action = workTaskAction(task);
  return action === "booking" || action === "resource" || action === "资源预定";
}

function workTaskIsForm(task: WorkTask): boolean {
  return !workTaskIsAssign(task) && !workTaskIsDecision(task) && !workTaskIsBooking(task);
}

function workTaskShouldRenderFields(task: WorkTask): boolean {
  const fields = task.form?.fields || [];
  if (fields.length === 0) return workTaskIsForm(task) || workTaskIsBooking(task);
  return workTaskIsForm(task) || workTaskIsBooking(task) || workTaskIsAssign(task);
}

function workTaskButtonLabel(task: WorkTask): string {
  const name = workTaskName(task);
  if (name && name !== "任务") return name;
  if (workTaskIsAssign(task)) return "派单";
  if (workTaskIsDecision(task)) return "决策";
  if (workTaskIsBooking(task)) return "资源预定";
  return "填写资料";
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
  return customers.flatMap((customer) => {
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

function renderStatus(
  target?: Pick<WorkCustomer, "stage_code" | "stage_name" | "status_code" | "status_name" | "current_status_name" | "current_stage_name"> | null,
) {
  const statusName = workStatusName(target);
  if (!statusName || statusName === "-") {
    return <span className="text-muted-foreground">-</span>;
  }
  return <span className="rounded-full bg-muted px-2 py-1 text-xs font-medium">{statusName}</span>;
}

function renderAssetStatus(asset: WorkAsset) {
  const statusName = textValue(asset.asset_status_name);
  if (!statusName) {
    return null;
  }
  return <span className="rounded-full border px-2 py-0.5 text-xs text-muted-foreground">{statusName}</span>;
}

function readableValueItems(
  values?: Record<string, unknown>,
  labels?: Record<string, string>,
): Array<[string, ReactNode]> {
  if (!values) return [];
  return Object.entries(values)
    .map(([key, value]) => {
      const label = textValue(labels?.[key]) || key;
      const text = displayText(value, "");
      return text ? ([label, text] as [string, ReactNode]) : null;
    })
    .filter(Boolean) as Array<[string, ReactNode]>;
}

function workReadableDataValueItems(target: WorkCustomer | WorkAsset): Array<[string, ReactNode]> {
  return readableValueItems(target.data_values, target.data_value_labels);
}

function openRowTask(
  customer: WorkCustomer,
  task: WorkTask,
  store?: StoreLike,
  asset?: WorkAsset,
) {
  setWorkStoreValue(store, "data.actionTarget.workTask", task);
  setWorkStoreValue(store, "data.actionTarget.workTaskCustomer", customer);
  setWorkStoreValue(store, "data.actionTarget.workTaskAsset", asset ?? null);
  setWorkStoreValue(store, "data.actionTarget.workTaskName", workTaskButtonLabel(task));
  setWorkModalOpen(store, "dialog.workTask", true);
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
  const [submitting, setSubmitting] = useState(false);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!phone || !password) {
      toast.error("请输入手机号和密码");
      return;
    }

    setSubmitting(true);
    try {
      const payload = await workApi<{ token?: string; user?: unknown }>("/crm/work/login", {
        method: "POST",
        body: JSON.stringify({ phone, password }),
      });
      const token = textValue(payload.token);
      if (!token) throw new Error("登录返回缺少 token");
      saveWorkSession(token, payload.user);
      window.location.href = getWorkEntryPath();
    } catch (error) {
      toast.error(errorMessage(error, "登录失败"));
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
            <img src={site.logo} alt={site.name} style={{ width: 56, height: 56, minWidth: 56 }} />
          ) : (
            <div
              className="flex items-center justify-center bg-primary text-lg font-semibold text-primary-foreground shadow-sm"
              style={{ width: 56, height: 56, minWidth: 56, borderRadius: 14 }}
            >
              {site.name.slice(0, 1) || "D"}
            </div>
          )}
          <div>
            <h1 className="text-2xl font-semibold leading-tight">{site.name}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{site.subtitle}</p>
          </div>
        </div>
        <form onSubmit={submit} className="w-full rounded-lg border bg-card p-6 shadow-sm">
          <div style={{ marginBottom: 28 }}>
            <h2 className="text-lg font-semibold leading-tight">人员登录</h2>
            <p className="text-sm text-muted-foreground" style={{ marginTop: 8 }}>
              请输入手机号和密码进入工作台
            </p>
          </div>
          <div className="flex flex-col" style={{ gap: 22 }}>
            <label className="block">
              <span className="block text-sm font-medium" style={{ marginBottom: 8 }}>
                手机号
              </span>
              <input
                className={inputClassName}
                value={phone}
                onChange={(event) => setPhone(event.target.value)}
                placeholder="请输入手机号"
                autoComplete="tel"
              />
            </label>
            <label className="block">
              <span className="block text-sm font-medium" style={{ marginBottom: 8 }}>
                密码
              </span>
              <input
                className={inputClassName}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="请输入密码"
                type="password"
                autoComplete="current-password"
              />
            </label>
          </div>
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
    <Button type="button" variant="outline" size="sm" onClick={notifyWorkDataChanged}>
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
      const payload = await workApi<{ tasks?: WorkTask[]; list?: WorkTask[] }>("/crm/work/tasks");
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
    setWorkStoreValue(store, "data.actionTarget.workTask", task);
    setWorkStoreValue(store, "data.actionTarget.workTaskCustomer", null);
    setWorkStoreValue(store, "data.actionTarget.workTaskAsset", null);
    setWorkStoreValue(store, "data.actionTarget.workTaskName", workTaskName(task));
    setWorkModalOpen(store, "dialog.workTask", true);
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
        <Button type="button" key={textValue(task.id)} size="sm" onClick={() => openTask(task)}>
          <Plus className="h-4 w-4" />
          {workTaskName(task)}
        </Button>
      ))}
    </div>
  );
}

export function ShowCrmWorkCustomerTable({ store }: WorkNodeProps) {
  const [customers, setCustomers] = useState<WorkCustomer[]>([]);
  const [filters, setFilters] = useState<WorkSearchFilters>(emptyWorkSearchFilters);
  const [activeFilters, setActiveFilters] = useState<WorkSearchFilters>(emptyWorkSearchFilters);
  const [loading, setLoading] = useState(false);
  const [selectedCustomerID, setSelectedCustomerID] = useState("");
  const [selectedAssetTarget, setSelectedAssetTarget] = useState<{
    customer: WorkCustomer;
    asset: WorkAsset;
  } | null>(null);

  const loadCustomers = useCallback(async () => {
    setLoading(true);
    try {
      const payload = await workApi<{
        list?: WorkCustomer[];
        customers?: WorkCustomer[];
        data?: WorkCustomer[];
      }>(`/crm/work/customers${workSearchQuery(activeFilters)}`);
      const list = payload.list || payload.customers || payload.data || [];
      setCustomers(Array.isArray(list) ? list : []);
    } catch (error) {
      toast.error(errorMessage(error, "客户列表加载失败"));
    } finally {
      setLoading(false);
    }
  }, [activeFilters]);

  useEffect(() => {
    loadCustomers();
  }, [loadCustomers]);

  useEffect(() => {
    const handler = () => loadCustomers();
    window.addEventListener(workRefreshEvent, handler);
    return () => window.removeEventListener(workRefreshEvent, handler);
  }, [loadCustomers]);

  const workItems = useMemo(() => buildWorkItems(customers), [customers]);
  const selectedCustomer = customers.find((customer) => workCustomerID(customer) === selectedCustomerID);

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
        <form onSubmit={submitSearch} className="flex flex-wrap items-center gap-2.5">
          {workSearchFields.map((field) => (
            <label key={field.key} className="shrink-0">
              <span className="sr-only">{field.placeholder}</span>
              <Input
                className={field.className}
                value={filters[field.key]}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, [field.key]: event.target.value }))
                }
                placeholder={field.placeholder}
              />
            </label>
          ))}
          <Button type="submit" size="sm" disabled={loading}>
            搜索
          </Button>
          <Button type="button" variant="outline" size="sm" onClick={resetSearch} disabled={loading}>
            重置
          </Button>
        </form>

        <div className="md:hidden">
          <WorkItemCardList
            items={workItems}
            loading={loading}
            store={store}
            onOpenCustomerDetail={(customer) => setSelectedCustomerID(workCustomerID(customer))}
            onOpenAssetDetail={(customer, asset) => setSelectedAssetTarget({ customer, asset })}
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
                  <WorkTableHead className="text-right">操作</WorkTableHead>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr>
                    <td colSpan={8} className="h-32 text-center text-muted-foreground">
                      <span className="inline-flex items-center gap-2">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        正在加载
                      </span>
                    </td>
                  </tr>
                ) : workItems.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="h-32 text-center text-muted-foreground">
                      暂无待处理工作
                    </td>
                  </tr>
                ) : (
                  workItems.map((item) => (
                    <WorkItemTableRow
                      key={item.id}
                      item={item}
                      store={store}
                      onOpenCustomerDetail={(customer) => setSelectedCustomerID(workCustomerID(customer))}
                      onOpenAssetDetail={(customer, asset) => setSelectedAssetTarget({ customer, asset })}
                    />
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <WorkCustomerDetailDrawer
        customer={selectedCustomer}
        store={store}
        onClose={() => setSelectedCustomerID("")}
      />
      <WorkAssetDetailDrawer
        customer={selectedAssetTarget?.customer}
        asset={selectedAssetTarget?.asset}
        store={store}
        onClose={() => setSelectedAssetTarget(null)}
      />
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
  store,
  onOpenCustomerDetail,
  onOpenAssetDetail,
}: {
  item: WorkItem;
  store?: StoreLike;
  onOpenCustomerDetail: (customer: WorkCustomer) => void;
  onOpenAssetDetail: (customer: WorkCustomer, asset: WorkAsset) => void;
}) {
  const { customer, asset } = item;
  const openDetail = () => {
    if (asset) {
      onOpenAssetDetail(customer, asset);
      return;
    }
    onOpenCustomerDetail(customer);
  };

  return (
    <tr className="border-b last:border-b-0">
      <WorkTableCell className="text-muted-foreground">{workItemCustomerNo(item)}</WorkTableCell>
      <WorkTableCell>
        <button type="button" className="max-w-[180px] truncate text-left font-medium" onClick={openDetail}>
          {workCustomerTitle(customer)}
        </button>
      </WorkTableCell>
      <WorkTableCell>{workCustomerPhone(customer)}</WorkTableCell>
      <WorkTableCell>{displayText(customer.wechat)}</WorkTableCell>
      <WorkTableCell className="text-muted-foreground">{workItemAssetNo(item)}</WorkTableCell>
      <WorkTableCell className="font-medium">
        {asset ? assetTitle(asset) : <span className="text-muted-foreground">未录入资产</span>}
      </WorkTableCell>
      <WorkTableCell>{renderWorkItemStatus(item)}</WorkTableCell>
      <WorkTableCell className="text-right">
        <WorkItemActions
          item={item}
          store={store}
          onOpenCustomerDetail={onOpenCustomerDetail}
          onOpenAssetDetail={onOpenAssetDetail}
        />
      </WorkTableCell>
    </tr>
  );
}

function WorkItemCardList({
  items,
  loading,
  store,
  onOpenCustomerDetail,
  onOpenAssetDetail,
}: {
  items: WorkItem[];
  loading: boolean;
  store?: StoreLike;
  onOpenCustomerDetail: (customer: WorkCustomer) => void;
  onOpenAssetDetail: (customer: WorkCustomer, asset: WorkAsset) => void;
}) {
  if (loading) {
    return (
      <div className="rounded-md border bg-background px-4 py-10 text-center text-sm text-muted-foreground">
        <span className="inline-flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          正在加载
        </span>
      </div>
    );
  }
  if (items.length === 0) {
    return (
      <div className="rounded-md border bg-background px-4 py-10 text-center text-sm text-muted-foreground">
        暂无待处理工作
      </div>
    );
  }
  return (
    <div className="grid gap-3">
      {items.map((item) => {
        const { customer, asset } = item;
        const openDetail = () => {
          if (asset) {
            onOpenAssetDetail(customer, asset);
            return;
          }
          onOpenCustomerDetail(customer);
        };
        return (
          <article key={item.id} className="rounded-md border bg-background">
            <div className="px-4 py-3">
              <div className="flex min-w-0 items-center justify-between gap-3">
                <button type="button" className="min-w-0 text-left" onClick={openDetail}>
                  <div className="truncate font-medium">{workCustomerTitle(customer)}</div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    {[workItemCustomerNo(item), textValue(customer.phone), textValue(customer.wechat)]
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
                  {asset ? `资产编号：${workItemAssetNo(item)}` : "请先补充客户资料或新增资产"}
                </div>
              </div>

              <div className="mt-3">
                <WorkItemActions
                  item={item}
                  store={store}
                  onOpenCustomerDetail={onOpenCustomerDetail}
                  onOpenAssetDetail={onOpenAssetDetail}
                />
              </div>
            </div>
          </article>
        );
      })}
    </div>
  );
}

function WorkItemActions({
  item,
  store,
  onOpenCustomerDetail,
  onOpenAssetDetail,
}: {
  item: WorkItem;
  store?: StoreLike;
  onOpenCustomerDetail: (customer: WorkCustomer) => void;
  onOpenAssetDetail: (customer: WorkCustomer, asset: WorkAsset) => void;
}) {
  const { customer, asset, tasks } = item;
  const openDetail = () => {
    if (asset) {
      onOpenAssetDetail(customer, asset);
      return;
    }
    onOpenCustomerDetail(customer);
  };

  return (
    <div className="flex flex-wrap gap-2 md:justify-end">
      <Button type="button" variant="outline" size="sm" onClick={openDetail}>
        详情
      </Button>
      {tasks.map((task) => (
        <Button
          type="button"
          key={textValue(task.id)}
          variant="outline"
          size="sm"
          onClick={() => openRowTask(customer, task, store, asset)}
        >
          {workTaskButtonLabel(task)}
        </Button>
      ))}
      {tasks.length === 0 ? <span className="self-center text-sm text-muted-foreground">暂无可执行任务</span> : null}
    </div>
  );
}

function WorkCustomerDetailDrawer({
  customer,
  store,
  onClose,
}: {
  customer?: WorkCustomer;
  store?: StoreLike;
  onClose: () => void;
}) {
  const [operations, setOperations] = useState<WorkOperation[]>([]);
  const [loadingOperations, setLoadingOperations] = useState(false);
  const customerID = workCustomerID(customer);
  const assets = customer ? (Array.isArray(customer.assets) ? customer.assets : []) : [];
  const customerTasks = customer ? workCustomerRowTasks(customer) : [];
  const customerExtraFields = customer ? workReadableDataValueItems(customer) : [];

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
    return () => {
      window.removeEventListener(workRefreshEvent, loadOperations);
    };
  }, [customerID, loadOperations]);

  if (!customer) {
    return null;
  }

  return (
    <WorkResponsiveDrawer ariaLabel="关闭客户详情" maxWidth="max-w-[920px]" onClose={onClose}>
      <div className="flex items-start justify-between gap-4 border-b px-6 py-5">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="truncate text-xl font-semibold">{workCustomerTitle(customer)}</h2>
            {renderStatus(customer)}
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            {workCustomerNo(customer) || "暂无客户编号"}
          </p>
        </div>
        <Button type="button" variant="ghost" size="sm" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto px-6 py-5">
        <div className="grid gap-5">
          <WorkDetailSection title="客户信息">
            <WorkDetailGrid
              items={[
                ["姓名", workCustomerName(customer)],
                ["手机号", workCustomerPhone(customer)],
                ["微信", displayText(customer.wechat)],
                ["性别", displayText(customer.gender_name || customer.gender)],
                ["来源", displayText(customer.source_name)],
                ["渠道", displayText(customer.channel_name)],
                ["客户等级", displayText(customer.level_name)],
                ["创建时间", formatWorkDate(customer.created_at)],
              ]}
            />
          </WorkDetailSection>

          <WorkDetailSection title="客户阶段">
            <div className="flex flex-wrap items-center gap-2">
              {renderStatus(customer)}
              <span className="text-sm text-muted-foreground">
                {textValue(customer.stage_name) && textValue(customer.stage_name) !== textValue(customer.stage_code)
                  ? textValue(customer.stage_name)
                  : "按当前状态展示可执行任务"}
              </span>
            </div>
          </WorkDetailSection>

          <WorkDetailSection title="当前应做">
            <WorkTaskButtons
              tasks={customerTasks}
              emptyText="当前客户暂无可执行任务"
              onOpen={(task) => openRowTask(customer, task, store)}
            />
          </WorkDetailSection>

          {customerExtraFields.length > 0 ? (
            <WorkDetailSection title="补充资料">
              <WorkDetailGrid items={customerExtraFields} />
            </WorkDetailSection>
          ) : null}

          <WorkDetailSection title="客户资产">
            {assets.length > 0 ? (
              <div className="grid gap-3">
                {assets.map((asset) => (
                  <WorkAssetDetailCard
                    key={textValue(asset.id) || workAssetNo(asset)}
                    customer={customer}
                    asset={asset}
                    store={store}
                  />
                ))}
              </div>
            ) : (
              <WorkEmptyText>暂无客户资产</WorkEmptyText>
            )}
          </WorkDetailSection>

          <WorkDetailSection title="操作记录">
            <WorkOperationTimeline operations={operations} loading={loadingOperations} />
          </WorkDetailSection>
        </div>
      </div>
    </WorkResponsiveDrawer>
  );
}

function WorkAssetDetailDrawer({
  customer,
  asset,
  store,
  onClose,
}: {
  customer?: WorkCustomer;
  asset?: WorkAsset;
  store?: StoreLike;
  onClose: () => void;
}) {
  const [operations, setOperations] = useState<WorkOperation[]>([]);
  const [loadingOperations, setLoadingOperations] = useState(false);
  const customerID = workCustomerID(customer);
  const assetID = textValue(asset?.id);
  const assetTasks = asset ? workAssetRowTasks(asset) : [];
  const assetExtraFields = asset ? workReadableDataValueItems(asset) : [];

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
      setOperations(assetID ? list.filter((operation) => textValue(operation.asset_id) === assetID) : list);
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
    return () => {
      window.removeEventListener(workRefreshEvent, loadOperations);
    };
  }, [customerID, loadOperations]);

  if (!customer || !asset) {
    return null;
  }

  return (
    <WorkResponsiveDrawer ariaLabel="关闭资产详情" maxWidth="max-w-[760px]" onClose={onClose}>
      <div className="flex items-start justify-between gap-4 border-b px-6 py-5">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="truncate text-xl font-semibold">{assetTitle(asset)}</h2>
            {renderStatus(asset)}
            {renderAssetStatus(asset)}
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            {workAssetNo(asset) || "暂无资产编号"}
          </p>
        </div>
        <Button type="button" variant="ghost" size="sm" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto px-6 py-5">
        <div className="grid gap-5">
          <WorkDetailSection title="所属客户">
            <WorkDetailGrid
              items={[
                ["客户编号", workCustomerNo(customer)],
                ["姓名", workCustomerName(customer)],
                ["手机号", workCustomerPhone(customer)],
                ["微信", displayText(customer.wechat)],
              ]}
            />
          </WorkDetailSection>

          <WorkDetailSection title="资产信息">
            <WorkDetailGrid
              items={[
                ["资产名称", assetTitle(asset)],
                ["资产编号", workAssetNo(asset) || "-"],
                ["资产状态", textValue(asset.asset_status_name) || "-"],
                ["当前状态", workStatusName(asset)],
                ["备注", displayText(asset.remark)],
              ]}
            />
          </WorkDetailSection>

          <WorkDetailSection title="当前应做">
            <WorkTaskButtons
              tasks={assetTasks}
              emptyText="当前资产暂无可执行任务"
              onOpen={(task) => openRowTask(customer, task, store, asset)}
            />
          </WorkDetailSection>

          {assetExtraFields.length > 0 ? (
            <WorkDetailSection title="补充资料">
              <WorkDetailGrid items={assetExtraFields} />
            </WorkDetailSection>
          ) : null}

          <WorkDetailSection title="操作记录">
            <WorkOperationTimeline operations={operations} loading={loadingOperations} />
          </WorkDetailSection>
        </div>
      </div>
    </WorkResponsiveDrawer>
  );
}

function WorkResponsiveDrawer({
  ariaLabel,
  maxWidth,
  onClose,
  children,
}: {
  ariaLabel: string;
  maxWidth: string;
  onClose: () => void;
  children: ReactNode;
}) {
  return (
    <div className="fixed inset-0 z-40 flex items-end justify-center bg-black/45 md:items-stretch md:justify-end">
      <button
        type="button"
        aria-label={ariaLabel}
        className="absolute inset-0 cursor-default"
        onClick={onClose}
      />
      <aside
        className={`relative flex h-[92vh] w-full flex-col rounded-t-lg border bg-background shadow-2xl md:h-full md:rounded-none md:border-l ${maxWidth}`}
      >
        {children}
      </aside>
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
    <section className="rounded-md border bg-background">
      <div className="border-b px-4 py-3">
        <h3 className="text-sm font-semibold">{title}</h3>
      </div>
      <div className="px-4 py-4">{children}</div>
    </section>
  );
}

function WorkDetailGrid({ items }: { items: Array<[string, ReactNode]> }) {
  return (
    <dl className="grid gap-x-6 gap-y-3 md:grid-cols-2">
      {items.map(([label, value]) => (
        <div key={label} className="grid grid-cols-[5.5rem_minmax(0,1fr)] gap-3 text-sm">
          <dt className="text-muted-foreground">{label}</dt>
          <dd className="min-w-0 break-words font-medium">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function WorkAssetDetailCard({
  customer,
  asset,
  store,
}: {
  customer: WorkCustomer;
  asset: WorkAsset;
  store?: StoreLike;
}) {
  const tasks = workAssetRowTasks(asset);
  return (
    <div className="rounded-md border bg-muted/10 px-4 py-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="truncate font-medium">{assetTitle(asset)}</span>
            {renderStatus(asset)}
            {renderAssetStatus(asset)}
          </div>
          <div className="mt-1 text-sm text-muted-foreground">
            {workAssetNo(asset) || "暂无资产编号"}
          </div>
        </div>
        <WorkTaskButtons
          tasks={tasks}
          emptyText="暂无可执行任务"
          onOpen={(task) => openRowTask(customer, task, store, asset)}
        />
      </div>
    </div>
  );
}

function WorkTaskButtons({
  tasks,
  emptyText,
  onOpen,
}: {
  tasks: WorkTask[];
  emptyText: string;
  onOpen: (task: WorkTask) => void;
}) {
  if (tasks.length === 0) {
    return <WorkEmptyText>{emptyText}</WorkEmptyText>;
  }
  return (
    <div className="flex flex-wrap gap-2">
      {tasks.map((task) => (
        <Button
          type="button"
          key={textValue(task.id)}
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

function WorkOperationTimeline({
  operations,
  loading,
}: {
  operations: WorkOperation[];
  loading: boolean;
}) {
  if (loading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        正在加载操作记录
      </div>
    );
  }

  if (operations.length === 0) {
    return <WorkEmptyText>暂无操作记录</WorkEmptyText>;
  }

  return (
    <div className="space-y-3">
      {operations.map((operation, index) => (
        <div
          key={`${textValue(operation.id) || index}`}
          className="border-l-2 border-border pl-3"
        >
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
            <span className="font-medium">
              {displayText(operation.title || operation.operation_name || operation.task_name, "操作记录")}
            </span>
            <span className="text-xs text-muted-foreground">
              {formatWorkDate(operation.created_at || operation.create_time)}
            </span>
          </div>
          <div className="mt-1 text-sm text-muted-foreground">
            {displayText(operation.content || operation.remark)}
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            操作人：{displayText(operation.operator_name || operation["operator_staff.name"])}
          </div>
        </div>
      ))}
    </div>
  );
}

export function ShowCrmWorkTaskForm({ store }: WorkNodeProps) {
  const task = workStoreValue<WorkTask | null>(store, "data.actionTarget.workTask", null);
  const customer = workStoreValue<WorkCustomer | null>(
    store,
    "data.actionTarget.workTaskCustomer",
    null,
  );
  const asset = workStoreValue<WorkAsset | null>(store, "data.actionTarget.workTaskAsset", null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [options, setOptions] = useState<WorkOptions>({ departments: [], staffs: [] });
  const [submitting, setSubmitting] = useState(false);
  const contentRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!task) return;
    const initialValues: Record<string, string> = {};
    (task.form?.fields || []).forEach((field) => {
      const key = workFieldKey(field);
      if (key) initialValues[key] = formatFormValue(workEntityFieldValue(field, customer, asset));
    });
    setValues(initialValues);
  }, [task, customer, asset]);

  useEffect(() => {
    if (!task || !workTaskIsAssign(task)) return;
    workApi<Partial<WorkOptions>>("/crm/work/options")
      .then((payload) => {
        setOptions({
          departments: Array.isArray(payload.departments) ? payload.departments : [],
          staffs: Array.isArray(payload.staffs) ? payload.staffs : [],
        });
      })
      .catch((error) => toast.error(errorMessage(error, "选项加载失败")));
  }, [task]);

  const updateValue = (key: string, value: string) => {
    setValues((current) => ({ ...current, [key]: value }));
  };

  const close = useCallback(() => {
    setWorkModalOpen(store, "dialog.workTask", false);
  }, [store]);

  const submit = useCallback(async () => {
    if (!task) return false;
    if (submitting) return false;
    setSubmitting(true);
    try {
      await workApi("/crm/work/execute", {
        method: "POST",
        body: JSON.stringify({
          task_id: task.id,
          customer_id: workCustomerID(customer),
          asset_id: workAssetID(asset),
          values,
        }),
      });
      toast.success("保存成功");
      notifyWorkRefresh();
      close();
    } catch (error) {
      toast.error(errorMessage(error));
      return false;
    } finally {
      setSubmitting(false);
    }
    return true;
  }, [asset, close, customer, submitting, task, values]);

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

  return (
    <div ref={contentRef} className="space-y-5">
      {workTaskIsDecision(task) ? (
        <WorkDecisionFields task={task} values={values} onChange={updateValue} />
      ) : null}

      {workTaskShouldRenderFields(task) ? (
        <WorkDynamicFields
          fields={task.form?.fields || []}
          values={values}
          onChange={updateValue}
        />
      ) : null}

      {workTaskIsAssign(task) ? (
        <WorkAssignFields values={values} options={options} onChange={updateValue} />
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

function WorkAssignFields({
  values,
  options,
  onChange,
}: {
  values: Record<string, string>;
  options: WorkOptions;
  onChange: (key: string, value: string) => void;
}) {
  const departmentID = values.department_id || "";
  const staffs = departmentID
    ? options.staffs.filter((staff) => textValue(staff.department_id) === departmentID)
    : [];

  return (
    <div className="grid gap-4">
      <FieldLabel label="部门" required>
        <Select
          value={departmentID}
          onValueChange={(value) => {
            onChange("department_id", value);
            onChange("staff_id", "");
          }}
        >
          <SelectTrigger>
            <SelectValue placeholder="请选择部门" />
          </SelectTrigger>
          <SelectContent>
            {options.departments.map((department) => (
              <SelectItem key={textValue(department.id)} value={textValue(department.id)}>
                {displayText(department.department_name || department.name)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FieldLabel>
      <FieldLabel label="人员" required>
        <Select
          value={values.staff_id || ""}
          onValueChange={(value) => onChange("staff_id", value)}
          disabled={!departmentID}
        >
          <SelectTrigger>
            <SelectValue placeholder={departmentID ? "请选择人员" : "请先选择部门"} />
          </SelectTrigger>
          <SelectContent>
            {staffs.map((staff) => (
              <SelectItem key={textValue(staff.id)} value={textValue(staff.id)}>
                {displayText(staff.real_name || staff.name)}
                {staff.phone ? `（${staff.phone}）` : ""}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FieldLabel>
    </div>
  );
}

function WorkDecisionFields({
  task,
  values,
  onChange,
}: {
  task: WorkTask;
  values: Record<string, string>;
  onChange: (key: string, value: string) => void;
}) {
  const results = Array.isArray(task.result_schema) ? task.result_schema : [];

  return (
    <div className="grid gap-4">
      <FieldLabel label="决策结果" required>
        <Select
          value={values.result_value || ""}
          onValueChange={(value) => onChange("result_value", value)}
        >
          <SelectTrigger>
            <SelectValue placeholder="请选择决策结果" />
          </SelectTrigger>
          <SelectContent>
            {results.map((result) => (
              <SelectItem
                key={workDecisionValue(result)}
                value={workDecisionValue(result)}
              >
                {displayText(result.label || result.name || result.value || result.result_value)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FieldLabel>
      <FieldLabel label="备注">
        <Textarea
          value={values.remark || ""}
          onChange={(event) => onChange("remark", event.target.value)}
          placeholder="请输入备注"
        />
      </FieldLabel>
    </div>
  );
}

function WorkDynamicFields({
  fields,
  values,
  onChange,
}: {
  fields: WorkFormField[];
  values: Record<string, string>;
  onChange: (key: string, value: string) => void;
}) {
  if (fields.length === 0) {
    return <div className="text-sm text-slate-500">该任务未配置资料字段</div>;
  }

  return (
    <div className="grid gap-4">
      {fields.map((field) => {
        const key = workFieldKey(field);
        if (!key) return null;
        const label = textValue(field.label) || textValue(field.name) || key;
        const options = Array.isArray(field.options) ? field.options : [];
        const isTextarea = textValue(field.field_type) === "textarea";

        return (
          <FieldLabel key={key} label={label} required={Boolean(field.required)}>
            {options.length > 0 ? (
              <Select value={values[key] || ""} onValueChange={(value) => onChange(key, value)}>
                <SelectTrigger>
                  <SelectValue placeholder={`请选择${label}`} />
                </SelectTrigger>
                <SelectContent>
                  {options.map((option) => (
                    <SelectItem
                      key={workOptionValue(option)}
                      value={workOptionValue(option)}
                    >
                      {workOptionLabel(option)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : isTextarea ? (
              <Textarea
                value={values[key] || ""}
                onChange={(event) => onChange(key, event.target.value)}
                placeholder={`请输入${label}`}
              />
            ) : (
              <Input
                value={values[key] || ""}
                onChange={(event) => onChange(key, event.target.value)}
                placeholder={`请输入${label}`}
              />
            )}
          </FieldLabel>
        );
      })}
    </div>
  );
}

function FieldLabel({
  label,
  required,
  children,
}: {
  label: string;
  required?: boolean;
  children: ReactNode;
}) {
  return (
    <label className="grid gap-2 md:grid-cols-[120px_1fr] md:items-start">
      <span className="pt-2 text-sm font-medium text-slate-950">
        {label}
        {required ? <span className="ml-1 text-red-500">*</span> : null}
      </span>
      <span>{children}</span>
    </label>
  );
}

function workDecisionValue(result: WorkDecisionResult): string {
  return (
    textValue(result.result_value) ||
    textValue(result.value) ||
    textValue(result.name) ||
    textValue(result.label)
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

function workEntityFieldValue(
  field: WorkFormField,
  customer?: WorkCustomer | null,
  asset?: WorkAsset | null,
): unknown {
  const mainField = textValue(field.main_field);
  if (mainField) {
    if (asset && asset[mainField] !== undefined) return asset[mainField];
    if (customer && customer[mainField] !== undefined) return customer[mainField];
  }
  const key = workFieldKey(field);
  if (key) {
    if (asset && asset[key] !== undefined) return asset[key];
    if (customer && customer[key] !== undefined) return customer[key];
  }
  if (field.default_value !== undefined) return field.default_value;
  return "";
}
