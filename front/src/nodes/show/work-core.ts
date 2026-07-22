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
  level_id?: string | number;
  level_name?: string;
  level_sort?: string | number;
  sort?: string | number;
};

export type WorkDataCompletenessTemplate = {
  template_id?: string | number;
  template_name?: string;
  name?: string;
  total?: string | number;
  filled?: string | number;
  percent?: string | number;
  missing?: string[];
  is_probe?: boolean;
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
  group_id?: string | number;
  group_key?: string;
  group_label?: string;
  placeholder?: string;
  required?: boolean;
  readonly?: boolean;
  visible_when_field_id?: string | number;
  visible_when_operator?: string;
  visible_when_value?: string;
  default_value?: string | number | boolean;
  options?: WorkFieldOption[];
  children?: WorkFormField[];
  meta?: Record<string, unknown>;
};

export type WorkForm = {
  id?: string | number;
  name?: string;
  fields?: WorkFormField[];
};

export type WorkTaskFieldRenderConfig = {
  type: string;
  placeholderPrefix: string;
  inputType?: "text" | "number" | "date" | "datetime-local";
  fullWidth?: boolean;
  options?: WorkCommonOption[];
  meta?: Record<string, unknown>;
};

export type WorkTask = {
  id?: string | number;
  task_id?: string | number;
  name?: string;
  task_name?: string;
  todo_id?: string | number;
  todo_status?: string;
  todo_required?: boolean;
  todo_sort?: string | number;
  status?: string;
  status_name?: string;
  assigned_at?: string;
  due_at?: string;
  result?: string;
  context_result?: string;
  context_result_label?: string;
  can_operate?: boolean;
  can_assign?: boolean;
  can_reassign?: boolean;
  unassigned?: boolean;
  required?: boolean;
  assignee_mode?:
    | "stage"
    | "auto"
    | "previous"
    | "manual"
    | "department_leader"
    | string;
  opinion_requirement?:
    | "optional"
    | "reject_required"
    | "required"
    | string;
  reject_submit_form?: boolean | string | number;
  workflow_instance_id?: string | number;
  customer_product_id?: string | number;
  workflow_id?: string | number;
  workflow_name?: string;
  stage_id?: string | number;
  stage_name?: string;
  lead_id?: string | number;
  customer_id?: string | number;
  asset_id?: string | number;
  assignee_department_id?: string | number;
  assignee_department_name?: string;
  assignee_staff_id?: string | number;
  assignee_staff_name?: string;
  task_type?: string;
  meeting_enabled?: boolean | string | number;
  communication_group_enabled?: boolean | string | number;
  product_options?: WorkProductOption[];
  selected_product_ids?: Array<string | number>;
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
  tags?: string;
  tag_ids?: Array<string | number>;
  source_id?: string | number;
  channel_id?: string | number;
  level_id?: string | number;
  status_name?: string;
  stage_name?: string;
  stage_code?: string;
  status_code?: string;
  current_stage_name?: string;
  current_status_name?: string;
  current_owner_staff_name?: string;
  current_owner_department_name?: string;
  stage_entered_at?: string;
  stage_days?: string | number;
  last_operated_at?: string;
  created_at?: string;
  create_time?: string;
  tasks?: WorkTask[];
  row_tasks?: WorkTask[];
  edit_tasks?: WorkTask[];
  assets?: WorkAsset[];
  operations?: WorkOperation[];
  data_values?: Record<string, unknown>;
  data_file_values?: Record<string, unknown>;
  data_value_labels?: Record<string, string>;
  data_completeness?: WorkDataCompletenessTemplate[];
  source_lead?: WorkSourceLead;
  [key: string]: unknown;
};

export type WorkSourceLead = {
  id?: string | number;
  code?: string;
  name?: string;
  phone?: string;
  wechat?: string;
  source_name?: string;
  channel_name?: string;
  external_id?: string;
  city?: string;
  initial_need?: string;
  created_at?: string;
  converted_at?: string;
  data_values?: Record<string, unknown>;
};

export type WorkAsset = {
  id?: string | number;
  asset_id?: string | number;
  customer_id?: string | number;
  workflow_id?: string | number;
  workflow_instance_id?: string | number;
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
  stage_entered_at?: string;
  stage_days?: string | number;
  last_operated_at?: string;
  remark?: string;
  tasks?: WorkTask[];
  row_tasks?: WorkTask[];
  operations?: WorkOperation[];
  data_values?: Record<string, unknown>;
  data_file_values?: Record<string, unknown>;
  data_value_labels?: Record<string, string>;
  data_completeness?: WorkDataCompletenessTemplate[];
  customer_products?: WorkCustomerProduct[];
  flow?: WorkFlowDetail;
  [key: string]: unknown;
};

export type WorkProductOption = {
  id?: string | number;
  name?: string;
  code?: string;
  category_name?: string;
  service_workflow_name?: string;
};

export type WorkCustomerProduct = {
  id?: string | number;
  customer_product_id?: string | number;
  product_id?: string | number;
  product_name?: string;
  product_code?: string;
  status?: string;
  status_name?: string;
  workflow_instance_id?: string | number;
  workflow_id?: string | number;
  workflow_name?: string;
  stage_id?: string | number;
  stage_name?: string;
  owner_staff_name?: string;
  flow?: WorkFlowDetail;
  created_at?: string;
  updated_at?: string;
  data_values?: Record<string, unknown>;
  data_value_labels?: Record<string, string>;
  data_completeness?: WorkDataCompletenessTemplate[];
  [key: string]: unknown;
};

export type WorkCommunicationGroupType = {
  id?: string | number;
  code?: string;
  name?: string;
  status?: string | number;
};

export type WorkCommunicationGroupStaff = {
  staff_id?: string | number;
  staff_name?: string;
  phone?: string;
  department_id?: string | number;
  department_name?: string;
  role?: string;
  role_name?: string;
  staff_status?: string | number;
};

export type WorkCommunicationGroup = {
  id?: string | number;
  communication_group_id?: string | number;
  customer_id?: string | number;
  asset_id?: string | number;
  workflow_instance_id?: string | number;
  group_type_id?: string | number;
  group_type_name?: string;
  group_type_code?: string;
  name?: string;
  external_group_id?: string;
  status?: "active" | "dissolved" | string;
  status_name?: string;
  established_at?: string;
  dissolved_at?: string;
  dissolve_reason?: string;
  summary?: string;
  remark?: string;
  can_edit?: boolean;
  staff?: WorkCommunicationGroupStaff[];
  created_at?: string;
  updated_at?: string;
};

export type WorkTaskCommunicationGroupContext = {
  groups: WorkCommunicationGroup[];
  groupTypes: WorkCommunicationGroupType[];
  workflowInstanceID: string;
  canManage: boolean;
};

export type WorkOperation = {
  id?: string | number;
  asset_id?: string | number;
  customer_id?: string | number;
  workflow_id?: string | number;
  workflow_instance_id?: string | number;
  customer_product_id?: string | number;
  stage_id?: string | number;
  task_type?: string;
  business_event?: string;
  result_value?: string;
  stage_code?: string;
  stage_name?: string;
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

export type WorkTodo = {
  id?: string | number;
  todo_id?: string | number;
  task_id?: string | number;
  workflow_id?: string | number;
  workflow_instance_id?: string | number;
  customer_product_id?: string | number;
  workflow_name?: string;
  stage_id?: string | number;
  stage_name?: string;
  customer_id?: string | number;
  asset_id?: string | number;
  task_name?: string;
  task_type?: string;
  form_id?: string | number;
  assignee_department_id?: string | number;
  assignee_department_name?: string;
  assignee_staff_id?: string | number;
  assignee_staff_name?: string;
  todo_required?: boolean;
  todo_sort?: string | number;
  status?: string;
  status_name?: string;
  can_operate?: boolean;
  assigned_at?: string;
  due_at?: string;
  result?: string;
  completed_at?: string;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown;
};

export type WorkFlowAssignee = {
  id?: string | number;
  name?: string;
  department_id?: string | number;
  active_flow_count?: string | number;
  active_asset_count?: string | number;
  pending_task_count?: string | number;
  last_assigned_at?: string;
};

export type WorkFlowAssigneeResponse = {
  list?: WorkFlowAssignee[];
  department_name?: string;
  assignment_mode?: string;
  terminal?: boolean;
};

export type WorkFlowDetail = {
  id?: string | number;
  workflow_instance_id?: string | number;
  lead_id?: string | number;
  customer_id?: string | number;
  asset_id?: string | number;
  customer_product_id?: string | number;
  product_id?: string | number;
  product_name?: string;
  flow_role?: "entry" | "product" | string;
  workflow_id?: string | number;
  workflow_name?: string;
  stage_id?: string | number;
  stage_name?: string;
  stage_assignment_mode?: "auto" | "manual" | string;
  owner_department_id?: string | number;
  owner_department_name?: string;
  owner_staff_id?: string | number;
  owner_staff_name?: string;
  status?: "not_started" | "active" | "completed" | "terminated" | string;
  started_at?: string;
  completed_at?: string;
  terminated_at?: string;
  terminated_reason?: string;
  pending_required_count?: string | number;
  is_current_owner?: boolean;
  can_dispatch?: boolean;
  can_complete_stage?: boolean;
  ready_to_complete?: boolean;
  can_terminate?: boolean;
  can_change_owner?: boolean;
  can_restart?: boolean;
  tasks?: WorkTask[];
  next_terminal?: boolean;
  next_workflow_id?: string | number;
  next_workflow_name?: string;
  next_stage_id?: string | number;
  next_stage_name?: string;
  next_department_id?: string | number;
  next_assignment_mode?: "auto" | "manual" | string;
  next_owner_required?: boolean;
  configuration_error?: string;
};

export type WorkOperationSummaryItem = {
  key?: string;
  label?: string;
  value?: unknown;
  previous_value?: unknown;
  change_type?: "added" | "updated" | "cleared" | string;
  value_type?: string;
  files?: UploadFileItem[];
  group_id?: string | number;
  group_label?: string;
  group_name?: string;
  groupId?: string | number;
  groupLabel?: string;
  groupName?: string;
};

export type WorkSummaryMetric = {
  key?: string;
  name?: string;
  value?: string | number;
  description?: string;
  drilldown_path?: string;
};

export type WorkSummaryBreakdown = {
  key?: string;
  name?: string;
  count?: string | number;
  percent?: string | number;
  drilldown_path?: string;
};

export type WorkSummaryTrendPoint = {
  date?: string;
  label?: string;
  task_count?: string | number;
  transition_count?: string | number;
};

export type WorkSummary = {
  metrics?: WorkSummaryMetric[];
  trend?: WorkSummaryTrendPoint[];
  stage_breakdown?: WorkSummaryBreakdown[];
  task_breakdown?: WorkSummaryBreakdown[];
  recent_operations?: WorkOperation[];
  generated_at?: string;
};

export type WorkItem = {
  id: string;
  targetType: "customer" | "asset";
  customer: WorkCustomer;
  asset?: WorkAsset;
  tasks: WorkTask[];
};

export type WorkCustomerMode = "all" | "pending" | "processed" | "done";
export type WorkCustomerScope = "mine" | "all";

export type WorkDetailField = {
  key: string;
  label: string;
  value?: unknown;
  valueType?: string;
  empty?: boolean;
  group?: string;
  files?: unknown[];
};

export type WorkDetailSection = {
  id: string;
  name: string;
  targetType: "lead" | "customer" | "asset" | "workflow";
  templateId?: string | number;
  workflowInstanceId?: string | number;
  customerProductId?: string | number;
  productName?: string;
  filled: number;
  total: number;
  percent: number;
  fields: WorkDetailField[];
};

export type WorkTaskLayoutMode = "compact" | "workspace";

export type WorkTaskFormField = {
  formKey: string;
  groupId?: string;
  label: string;
  placeholder: string;
  required: boolean;
  readonly?: boolean;
  visibleWhenFieldId?: string;
  visibleWhenRawKey?: string;
  visibleWhenOperator?: string;
  visibleWhenValue?: string;
  type: string;
  inputType?: "text" | "number" | "date" | "datetime-local";
  fullWidth?: boolean;
  options?: WorkCommonOption[];
  meta?: Record<string, unknown>;
};

export type WorkTaskFormFieldVisibilityRule = {
  driverRawKey: string;
  operator: string;
  expectedValues: string[];
};

export function workTaskFormFieldVisibilityRule(
  field: WorkTaskFormField,
): WorkTaskFormFieldVisibilityRule | null {
  const configuredRawKey = textValue(field.visibleWhenRawKey);
  const driverFieldId = positiveTextID(field.visibleWhenFieldId);
  if (!configuredRawKey && !driverFieldId) return null;
  return {
    driverRawKey: configuredRawKey || `data:${driverFieldId}`,
    operator: textValue(field.visibleWhenOperator) || "equals",
    expectedValues: workTaskConditionExpectedValues(field.visibleWhenValue),
  };
}

export function workTaskFormFieldVisible(
  field: WorkTaskFormField,
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  fields: WorkTaskFormField[] = [],
): boolean {
  return workTaskFormFieldVisibleWithAncestors(
    field,
    values,
    fieldMap,
    fields,
    new Set<string>(),
  );
}

function workTaskFormFieldVisibleWithAncestors(
  field: WorkTaskFormField,
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  fields: WorkTaskFormField[],
  visiting: Set<string>,
): boolean {
  const rule = workTaskFormFieldVisibilityRule(field);
  if (!rule) return true;
  if (visiting.has(field.formKey)) return false;
  visiting.add(field.formKey);
  const driverFormKey = Object.entries(fieldMap).find(
    ([, rawKey]) => rawKey === rule.driverRawKey,
  )?.[0];
  const driverField = fields.find(
    (candidate) => candidate.formKey === driverFormKey,
  );
  if (
    driverField &&
    !workTaskFormFieldVisibleWithAncestors(
      driverField,
      values,
      fieldMap,
      fields,
      visiting,
    )
  ) {
    visiting.delete(field.formKey);
    return false;
  }
  const actual = driverFormKey ? values[driverFormKey] : undefined;
  visiting.delete(field.formKey);
  if (rule.operator === "empty") return workTaskConditionValueEmpty(actual);
  if (rule.operator === "not_empty") {
    return !workTaskConditionValueEmpty(actual);
  }
  const matched = workTaskConditionValues(actual).some((value) =>
    rule.expectedValues.includes(value),
  );
  if (rule.operator === "not_equals" || rule.operator === "not_in") {
    return !matched;
  }
  return rule.operator === "equals" || rule.operator === "in"
    ? matched
    : false;
}

export function workTaskFormFieldVisibilityRules(
  field: WorkTaskFormField,
  fieldMap: Record<string, string>,
  fields: WorkTaskFormField[],
): WorkTaskFormFieldVisibilityRule[] {
  return workTaskFormFieldVisibilityRulesWithAncestors(
    field,
    fieldMap,
    fields,
    new Set<string>(),
  );
}

function workTaskFormFieldVisibilityRulesWithAncestors(
  field: WorkTaskFormField,
  fieldMap: Record<string, string>,
  fields: WorkTaskFormField[],
  visiting: Set<string>,
): WorkTaskFormFieldVisibilityRule[] {
  const rule = workTaskFormFieldVisibilityRule(field);
  if (!rule || visiting.has(field.formKey)) return [];
  visiting.add(field.formKey);
  const driverFormKey = Object.entries(fieldMap).find(
    ([, rawKey]) => rawKey === rule.driverRawKey,
  )?.[0];
  const driverField = fields.find(
    (candidate) => candidate.formKey === driverFormKey,
  );
  const ancestors = driverField
    ? workTaskFormFieldVisibilityRulesWithAncestors(
        driverField,
        fieldMap,
        fields,
        visiting,
      )
    : [];
  visiting.delete(field.formKey);
  return [...ancestors, rule];
}

export function workTaskFormFieldRequired(
  field: WorkTaskFormField,
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
): boolean {
  if (field.required) return true;
  const driverRawKey = textValue(field.meta?.["requiredWhenRawKey"]);
  const expectedValue = textValue(field.meta?.["requiredWhenValue"]);
  if (!driverRawKey || !expectedValue) return false;
  const driverFormKey = Object.entries(fieldMap).find(
    ([, rawKey]) => rawKey === driverRawKey,
  )?.[0];
  return driverFormKey
    ? textValue(values[driverFormKey]) === expectedValue
    : false;
}

function workTaskConditionExpectedValues(value: unknown): string[] {
  const text = textValue(value).trim();
  if (!text) return [];
  if (text.startsWith("[")) {
    try {
      const decoded: unknown = JSON.parse(text);
      if (Array.isArray(decoded)) return workTaskConditionValues(decoded);
    } catch {
      // Fall through to comma-separated values.
    }
  }
  return text
    .split(/[\r\n,]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function workTaskConditionValues(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map((item) => textValue(item).trim()).filter(Boolean);
  }
  const normalized = textValue(value).trim();
  return normalized ? [normalized] : [];
}

function workTaskConditionValueEmpty(value: unknown): boolean {
  if (value === null || value === undefined || value === "") return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "object") return Object.keys(value).length === 0;
  return textValue(value).trim() === "";
}

export type WorkTaskFormGroup = {
  id: string;
  label: string;
  fields: WorkTaskFormField[];
};

export type WorkCommonOption = {
  id: string;
  value: string;
  [key: string]: unknown;
};

export type WorkTaskFormNode = {
  id: string;
  type: string;
  className?: string;
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
  fields: WorkTaskFormField[];
  layout: WorkTaskLayoutMode;
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
export const workListSearchEvent = "crm-work-list-search";
export const workTaskFormSectionID = "work-task-form-section";
export const workTaskFormDataPath = "data.workTaskForm";
export const workTaskPath = "data.actionTarget.workTask";
export const workTaskFieldMapPath = "data.actionTarget.workTaskFieldMap";
export const workTaskFormFieldsPath = "data.actionTarget.workTaskFormFields";
export const workTaskValidationErrorsPath =
  "data.actionTarget.workTaskValidationErrors";
export const workTaskLayoutPath = "data.actionTarget.workTaskLayout";
export const workTaskActiveGroupPath = "data.actionTarget.workTaskActiveGroup";
export const workTaskUploadPendingPath =
  "data.actionTarget.workTaskUploadPending";
export const workTaskUploadFilesPath =
  "data.actionTarget.workTaskUploadFiles";
export const workTaskCommunicationGroupContextPath =
  "data.actionTarget.workTaskCommunicationGroupContext";
export const workTaskCommunicationGroupDraftPath =
  "data.actionTarget.workTaskCommunicationGroupDraft";
export const workTaskCommunicationGroupErrorPath =
  "data.actionTarget.workTaskCommunicationGroupError";
let workApiFreshSeq = 0;

const buttonBase =
  "inline-flex items-center justify-center gap-2 rounded-md text-sm font-medium shadow-sm transition disabled:cursor-not-allowed disabled:opacity-60";
export const primaryButton = `${buttonBase} bg-primary text-primary-foreground hover:bg-primary/90`;
export const outlineButton = `${buttonBase} border border-border bg-background hover:bg-muted`;
export const inputClassName =
  "h-10 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm outline-none transition placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20";

export function readWorkListSearch(event: Event): {
  keyword: string;
  workflowID: string;
  mode: string;
  scope: string;
} {
  const detail =
    (event as CustomEvent<Record<string, unknown>>).detail || {};
  return {
    keyword: textValue(detail.keyword),
    workflowID: textValue(detail.workflow_id),
    mode: textValue(detail.mode),
    scope: textValue(detail.scope),
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
  processed: {
    emptyTitle: "暂无已处理案件",
    emptyDescription: "当前没有由你实际办理过的客户资产记录",
  },
  done: {
    emptyTitle: "暂无已结束业务",
    emptyDescription: "当前没有已完成或已终止的流程",
  },
};

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

export function workTaskFormKey(key: string): string {
  const normalized = key
    .trim()
    .replace(/[^a-zA-Z0-9_]+/g, "_")
    .replace(/^_+|_+$/g, "");
  return normalized || "field";
}

export function workIsRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
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
  return siteRoot;
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

export function readWorkSessionUser(): Record<string, unknown> {
  if (typeof window === "undefined") return {};
  const keys = [
    getFrontUserKey(),
    workUserKey,
    legacyWorkUserKey,
    legacyFrontUserKey,
  ];
  for (const key of keys) {
    const raw = window.localStorage.getItem(key);
    if (!raw) continue;
    try {
      const user = JSON.parse(raw) as unknown;
      if (workIsRecord(user)) return user;
    } catch {
      continue;
    }
  }
  return {};
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
  const requestPath = freshWorkApiPath(path, method);
  const result = (await request(requestPath, method, payload)) as
    | WorkApiResponse<T>
    | T;

  return unwrapWorkApiResult(result);
}

function freshWorkApiPath(path: string, method: WorkApiMethod): string {
  if (method !== "get") {
    return path;
  }
  workApiFreshSeq += 1;
  const joiner = path.includes("?") ? "&" : "?";
  return `${path}${joiner}_r=${Date.now()}_${workApiFreshSeq}`;
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

function normalizeWorkDetailFields(value: unknown): WorkDetailField[] {
  if (!Array.isArray(value)) return [];
  return value.filter(workIsRecord).map((field) => ({
    key: textValue(field.key),
    label: displayText(field.label),
    value: field.value,
    valueType: textValue(field.value_type) || "text",
    empty: Boolean(field.empty),
    group: textValue(field.group),
    files: Array.isArray(field.files) ? field.files : [],
  }));
}

export function normalizeWorkDetailSections(
  value: unknown,
): WorkDetailSection[] {
  if (!Array.isArray(value)) return [];
  return value.filter(workIsRecord).map((section) => {
    const rawTargetType = textValue(section.target_type);
    const targetType: WorkDetailSection["targetType"] =
      rawTargetType === "lead" ||
      rawTargetType === "asset" ||
      rawTargetType === "workflow"
        ? rawTargetType
        : "customer";
    return {
      id: textValue(section.id),
      name: displayText(section.name),
      targetType,
      templateId: section.template_id as string | number | undefined,
      workflowInstanceId: section.workflow_instance_id as
        | string
        | number
        | undefined,
      customerProductId: section.customer_product_id as
        | string
        | number
        | undefined,
      productName: textValue(section.product_name),
      filled: Number(section.filled) || 0,
      total: Number(section.total) || 0,
      percent: Number(section.percent) || 0,
      fields: normalizeWorkDetailFields(section.fields),
    };
  });
}

const workDateTimeFormatter = new Intl.DateTimeFormat("zh-CN", {
  timeZone: "Asia/Shanghai",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  hourCycle: "h23",
});

export function formatWorkDate(value: unknown): string {
  const text = textValue(value);
  if (!text) return "-";
  const date = new Date(text);
  if (Number.isNaN(date.getTime())) {
    return text.replace("T", " ").replace(/\.\d+Z?$/, "").slice(0, 16);
  }
  const parts: Record<string, string> = {};
  for (const part of workDateTimeFormatter.formatToParts(date)) {
    if (part.type !== "literal") parts[part.type] = part.value;
  }
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}`;
}
