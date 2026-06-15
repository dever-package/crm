import { request } from "@dever/front-plugin";
import { getStoreValueByPath, setStoreValueByPath } from "@/lib/store";
import type { UploadFileItem } from "@/lib/upload";

export type WorkStoreLike = Record<string, unknown>;

export type WorkNodeProps = {
  item?: {
    id?: string;
    name?: string;
    value?: string;
    placeholder?: string;
    meta?: Record<string, unknown>;
  };
  store?: WorkStoreLike;
  data?: Record<string, unknown>;
  value?: unknown;
  setValue?: (value: unknown) => void;
  error?: string;
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

export type WorkFieldOption = {
  id?: string | number;
  name?: string;
  label?: string;
  value?: string | number;
};

export type WorkFormField = {
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

export type WorkForm = {
  id?: string | number;
  name?: string;
  fields?: WorkFormField[];
};

export type WorkTaskFieldRenderConfig = {
  type: string;
  placeholderPrefix: string;
  options?: WorkCommonOption[];
  meta?: Record<string, unknown>;
};

export type WorkTask = {
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
  collaboration_items?: Array<Record<string, unknown>> | string;
  collaboration_complete_mode?: string;
  form_id?: string | number;
  form?: WorkForm | null;
};

export type WorkCustomer = {
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

export type WorkAsset = {
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

export type WorkOperation = {
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

export type WorkOperationSummaryItem = {
  key?: string;
  label?: string;
  value?: unknown;
  value_type?: string;
  files?: UploadFileItem[];
};

export type WorkItem = {
  id: string;
  targetType: "customer" | "asset";
  customer: WorkCustomer;
  asset?: WorkAsset;
  tasks: WorkTask[];
};

export type WorkCustomerMode = "all" | "pending" | "done";

export type WorkSearchFilters = {
  customerNo: string;
  customerName: string;
  phone: string;
  wechat: string;
  assetNo: string;
  status: string;
};

export type WorkDepartmentOption = {
  id?: string | number;
  name?: string;
  department_name?: string;
};

export type WorkStaffOption = {
  id?: string | number;
  name?: string;
  real_name?: string;
  phone?: string;
  department_id?: string | number;
};

export type WorkOptions = {
  departments: WorkDepartmentOption[];
  staffs: WorkStaffOption[];
  forms: WorkCommonOption[];
};

export type WorkCommonOption = {
  id: string;
  value: string;
  [key: string]: unknown;
};

export type WorkTaskFormNode = {
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

export type WorkTaskUploadMeta = {
  ruleId: number;
  kind: string;
  maxCount: number;
  bizKey: string;
  bizName: string;
};

export type WorkTaskUploadProgress = {
  fileName: string;
  percent: number;
  currentIndex: number;
  total: number;
};

export type WorkTaskFormState = {
  nodes: WorkTaskFormNode[];
  values: Record<string, unknown>;
  fieldMap: Record<string, string>;
};

export type WorkAIFillResponse = {
  values?: Record<string, unknown>;
  summary?: string;
  filled_count?: number;
};

type WorkApiMethod = "get" | "post" | "put" | "delete" | "patch";

type WorkApiResponse<T> = {
  code?: number;
  status?: number;
  msg?: string;
  message?: string;
  data?: T;
};

export type WorkPageStoreState = {
  schema?: {
    nodes?: Record<string, WorkTaskFormNode[]>;
    [key: string]: unknown;
  };
  errors?: Record<string, string>;
  validateForm?: () => boolean;
};

export const workRefreshEvent = "crm-work-refresh";
export const workTaskFormSectionID = "work-task-form-section";
export const workTaskFormDataPath = "data.workTaskForm";
export const workTaskFieldMapPath = "data.actionTarget.workTaskFieldMap";

const buttonBase =
  "inline-flex items-center justify-center gap-2 rounded-md text-sm font-medium shadow-sm transition disabled:cursor-not-allowed disabled:opacity-60";
export const primaryButton = `${buttonBase} bg-primary text-primary-foreground hover:bg-primary/90`;
export const outlineButton = `${buttonBase} border border-border bg-background hover:bg-muted`;
export const inputClassName =
  "h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm outline-none transition placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20";

export const workSearchFields: Array<{
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

export function emptyWorkSearchFilters(): WorkSearchFilters {
  return {
    customerNo: "",
    customerName: "",
    phone: "",
    wechat: "",
    assetNo: "",
    status: "",
  };
}

export const workCustomerModeConfig: Record<
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

export const workTableHeadClass =
  "h-12 whitespace-nowrap px-4 text-left text-sm font-medium text-muted-foreground";
export const workTableCellClass =
  "min-h-14 whitespace-nowrap px-4 py-3 align-middle text-sm";
export const workTableStickyLeftHeadClass =
  "sticky left-0 z-30 border-r bg-muted/40";
export const workTableStickyLeftCellClass =
  "sticky left-0 z-20 border-r";
export const workTableStickyRightHeadClass =
  "sticky right-0 z-30 border-l bg-muted/40";
export const workTableStickyRightCellClass =
  "sticky right-0 z-20 border-l";
export const workUploadGridColumns = "minmax(0, 1fr) 6rem 7rem";
export const workImageExtensions = new Set([
  "png",
  "jpg",
  "jpeg",
  "gif",
  "webp",
  "bmp",
  "svg",
]);

const workTokenKey = "crm_work_token";
const workUserKey = "crm_work_user";
const legacyWorkTokenKey = "gjj_crm_work_token";
const legacyWorkUserKey = "gjj_crm_work_user";
const legacyFrontTokenKey = "front-token:work";
const legacyFrontUserKey = "front-user:work";
const defaultWorkSiteKey = "work";
const authCookieMaxAge = 3600 * 24 * 7;

export function textValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  return String(value).trim();
}

export function errorMessage(error: unknown, fallback = "操作失败"): string {
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

export function getRuntimeSite() {
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

function setCookieValue(name: string, value: string, maxAge: number) {
  document.cookie = `${name}=${value}; path=/; max-age=${maxAge}`;
}

function removeCookieValue(name: string) {
  setCookieValue(name, "", 0);
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

export function getWorkEntryPath(): string {
  const basePath = getRuntimeBasePath();
  const normalized = basePath.startsWith("/") ? basePath : `/${basePath}`;
  const rawSiteRoot = normalized.replace(/\/(?:login|sign-in|index)\/?$/, "");
  const siteRoot = rawSiteRoot && rawSiteRoot !== "/" ? rawSiteRoot : "/work";
  return `${siteRoot}/index`;
}

export function saveWorkSession(token: string, user: unknown) {
  window.localStorage.setItem(workTokenKey, token);
  window.localStorage.setItem(workUserKey, JSON.stringify(user ?? {}));
  window.localStorage.setItem(legacyWorkTokenKey, token);
  window.localStorage.setItem(legacyWorkUserKey, JSON.stringify(user ?? {}));
  window.localStorage.setItem(legacyFrontTokenKey, token);
  window.localStorage.setItem(legacyFrontUserKey, JSON.stringify(user ?? {}));

  setCookieValue(getFrontTokenKey(), JSON.stringify(token), authCookieMaxAge);
  window.localStorage.setItem(getFrontUserKey(), JSON.stringify(user ?? {}));
}

export function clearWorkSession() {
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

export async function workApi<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  ensureWorkAuthCookie();
  const method = normalizeWorkApiMethod(init.method);
  const payload = normalizeWorkApiPayload(init.body);
  const result = (await request(path, method, payload)) as
    | WorkApiResponse<T>
    | T;

  return unwrapWorkApiResult(result);
}

function ensureWorkAuthCookie() {
  const token = readCurrentWorkToken();
  if (!token) return;
  setCookieValue(getFrontTokenKey(), JSON.stringify(token), authCookieMaxAge);
}

function readCurrentWorkToken(): string {
  return (
    readTokenCookie(getFrontTokenKey()) ||
    readTokenCookie(legacyFrontTokenKey) ||
    textValue(window.localStorage.getItem(workTokenKey)) ||
    textValue(window.localStorage.getItem(legacyWorkTokenKey)) ||
    textValue(window.localStorage.getItem(legacyFrontTokenKey))
  );
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

function getCookieValue(name: string): string {
  const parts = `; ${document.cookie}`.split(`; ${name}=`);
  if (parts.length !== 2) return "";
  return parts.pop()?.split(";").shift() ?? "";
}

function normalizeWorkApiMethod(method: string | undefined): WorkApiMethod {
  const normalized = textValue(method).toLowerCase();
  if (
    normalized === "post" ||
    normalized === "put" ||
    normalized === "delete" ||
    normalized === "patch"
  ) {
    return normalized;
  }
  return "get";
}

function normalizeWorkApiPayload(
  body: BodyInit | null | undefined,
): Record<string, unknown> | FormData | undefined {
  if (!body) return undefined;
  if (typeof FormData !== "undefined" && body instanceof FormData) {
    return body;
  }
  if (typeof body === "string") {
    try {
      const parsed = JSON.parse(body) as unknown;
      return isWorkApiPayload(parsed) ? parsed : undefined;
    } catch {
      return undefined;
    }
  }
  return undefined;
}

function unwrapWorkApiResult<T>(result: WorkApiResponse<T> | T): T {
  if (!isWorkApiResponse(result)) {
    return result as T;
  }

  const code = typeof result.code === "number" ? result.code : 0;
  const status = typeof result.status === "number" ? result.status : 1;

  if (code !== 0 || (status > 0 && status !== 1)) {
    throw new Error(
      textValue(result.message) || textValue(result.msg) || "请求失败",
    );
  }

  return (result.data ?? result) as T;
}

function isWorkApiResponse<T>(
  result: WorkApiResponse<T> | T,
): result is WorkApiResponse<T> {
  return (
    !!result &&
    typeof result === "object" &&
    !Array.isArray(result) &&
    ("data" in result ||
      "code" in result ||
      "status" in result ||
      "msg" in result ||
      "message" in result)
  );
}

function isWorkApiPayload(
  value: unknown,
): value is Record<string, unknown> | FormData {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

export function workStoreValue<T>(
  store: WorkStoreLike | undefined,
  path: string,
  fallback: T,
): T {
  const value = store ? getStoreValueByPath(store, path) : undefined;
  return value === undefined || value === null ? fallback : (value as T);
}

export function setWorkStoreValue(
  store: WorkStoreLike | undefined,
  path: string,
  value: unknown,
) {
  if (!store) return;
  setStoreValueByPath(store, path, value);
}

export function setWorkModalOpen(
  store: WorkStoreLike | undefined,
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

export function currentWorkStoreState(
  store: WorkStoreLike | undefined,
): WorkPageStoreState | undefined {
  return (
    store as { getState?: () => WorkPageStoreState } | undefined
  )?.getState?.();
}

export function updateWorkStoreErrors(
  store: WorkStoreLike | undefined,
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

export function positiveTextID(value: unknown): string {
  const id = textValue(value);
  return id && id !== "0" ? id : "";
}

export function displayText(value: unknown, fallback = "-"): string {
  const text = textValue(value);
  return text || fallback;
}

export function formatWorkDate(value: unknown): string {
  const text = textValue(value);
  if (!text) return "-";
  return text
    .replace("T", " ")
    .replace(/\.\d+Z?$/, "")
    .slice(0, 16);
}
