import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ChangeEvent, FormEvent, ReactNode, RefObject } from "react";
import { createPortal } from "react-dom";
import {
  Check,
  Bot,
  Download,
  FileText,
  Inbox,
  LogIn,
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

import {
  clearWorkSession,
  currentWorkStoreState,
  displayText,
  emptyWorkSearchFilters,
  errorMessage,
  formatWorkDate,
  getRuntimeSite,
  getWorkEntryPath,
  inputClassName,
  outlineButton,
  positiveTextID,
  primaryButton,
  saveWorkSession,
  setWorkModalOpen,
  setWorkStoreValue,
  textValue,
  updateWorkStoreErrors,
  workApi,
  workCustomerModeConfig,
  workImageExtensions,
  workRefreshEvent,
  workSearchFields,
  workStoreValue,
  workTableCellClass,
  workTableHeadClass,
  workTableStickyLeftCellClass,
  workTableStickyLeftHeadClass,
  workTableStickyRightCellClass,
  workTableStickyRightHeadClass,
  workTaskFieldMapPath,
  workTaskFormDataPath,
  workTaskFormSectionID,
  workUploadGridColumns,
  type WorkAIFillResponse,
  type WorkAsset,
  type WorkCommonOption,
  type WorkCustomer,
  type WorkCustomerMode,
  type WorkDepartmentOption,
  type WorkFieldOption,
  type WorkFormField,
  type WorkItem,
  type WorkNodeProps,
  type WorkOperation,
  type WorkOperationSummaryItem,
  type WorkOptions,
  type WorkPageStoreState,
  type WorkSearchFilters,
  type WorkStaffOption,
  type WorkStoreLike,
  type WorkTask,
  type WorkTaskFieldRenderConfig,
  type WorkTaskFormNode,
  type WorkTaskFormState,
  type WorkTaskUploadMeta,
  type WorkTaskUploadProgress,
} from "./work-core";
import {
  buildFeishuOAuthURL,
  getFeishuAuthCode,
  isFeishuClient,
  loadFeishuSDK,
} from "./feishu-login";

type StoreLike = WorkStoreLike;

type WorkCollaborationTarget = {
  key?: string;
  name: string;
  department_id: string;
  staff_id: string;
  form_id?: string;
  form?: {
    id?: string | number;
    name?: string;
    fields?: WorkFormField[];
  } | null;
  fields?: WorkFormField[];
  required?: boolean;
  sort?: number;
  staff_locked?: boolean;
};

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
  if (!workTaskNeedsAssigneeOptions(task)) {
    return { departments: [], staffs: [], forms: [] };
  }

  try {
    const payload = await workApi<Partial<WorkOptions>>("/crm/work/options");
    return {
      departments: Array.isArray(payload.departments)
        ? payload.departments
        : [],
      staffs: Array.isArray(payload.staffs) ? payload.staffs : [],
      forms: Array.isArray(payload.forms) ? payload.forms : [],
    };
  } catch (error) {
    toast.error(errorMessage(error, "选项加载失败"));
    return { departments: [], staffs: [], forms: [] };
  }
}

function workTaskNeedsAssigneeOptions(task: WorkTask): boolean {
  return workTaskIsAssign(task) || workTaskCanSelectCollaborationTargets(task);
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
  options: WorkOptions = { departments: [], staffs: [], forms: [] },
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
    addWorkTaskAssignTargetNodes(nodes, values, fieldMap, task, options);
  }

  if (workTaskCanSelectCollaborationTargets(task)) {
    addWorkTaskCollaborationTargetNode(
      nodes,
      values,
      fieldMap,
      task,
      options,
      customer,
      asset,
    );
  }

  return { nodes, values, fieldMap };
}

function addWorkTaskAssignTargetNodes(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  task: WorkTask,
  options: WorkOptions,
) {
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
      meta: workStaffSelectMeta(departmentFormKey),
    });
  }
}

function addWorkTaskCollaborationTargetNode(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  task: WorkTask,
  options: WorkOptions,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
) {
  const formKey = uniqueWorkTaskFormKey("collaboration_targets", fieldMap);
  values[formKey] = initialWorkCollaborationTargets(task);
  fieldMap[formKey] = "collaboration_targets";
  nodes.push({
    id: `work-task-field-${formKey}`,
    type: "show-crm-work-collaboration-targets",
    name: "协作对象",
    value: `workTaskForm.${formKey}`,
    mode: "form",
    meta: {
      formLayout: "horizontal",
      departments: workDepartmentOptions(options.departments),
      staffs: workStaffOptions(options.staffs),
      forms: workFormOptions(options.forms),
      defaultName: workTaskName(task),
    },
  });
  addWorkTaskCollaborationFormNodes(
    nodes,
    values,
    fieldMap,
    task,
    customer,
    asset,
  );
}

function addWorkTaskCollaborationFormNodes(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  task: WorkTask,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
) {
  for (const target of initialWorkCollaborationTargets(task)) {
    const fields = workCollaborationTargetFields(target);
    const targetKey = workCollaborationTargetKey(target);
    if (!targetKey || fields.length === 0) continue;

    const title = target.name || "协作子任务";
    for (const field of fields) {
      const rawKey = workFieldKey(field);
      if (!rawKey) continue;
      addWorkTaskFieldNode(nodes, values, fieldMap, field, customer, asset, {
        labelPrefix: `${title} / `,
        rawKey: `collaboration_form:${targetKey}:${rawKey}`,
        required: false,
      });
    }
  }
}

function workStaffSelectMeta(departmentFormKey: string): Record<string, unknown> {
  return {
    hiddenWhen: [{ path: `workTaskForm.${departmentFormKey}`, operator: "empty" }],
    optionFilter: [
      {
        field: "department_id",
        path: `workTaskForm.${departmentFormKey}`,
        operator: "equals",
      },
    ],
  };
}

function addWorkTaskFieldNode(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  field: WorkFormField,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
  config: {
    labelPrefix?: string;
    rawKey?: string;
    required?: boolean;
  } = {},
) {
  const rawKey = config.rawKey || workFieldKey(field);
  if (!rawKey) return;

  const formKey = workTaskFormKey(rawKey);
  const label =
    `${config.labelPrefix || ""}${textValue(field.label) || textValue(field.name) || rawKey}`;
  const options = Array.isArray(field.options)
    ? field.options.map(workFieldOption)
    : [];
  const renderConfig = workTaskFieldRenderConfig(field, options);

  addWorkTaskTextNode(nodes, values, fieldMap, {
    formKey,
    rawKey,
    label,
    placeholder: `${renderConfig.placeholderPrefix}${label}`,
    required:
      typeof config.required === "boolean"
        ? config.required
        : Boolean(field.required),
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

function workFormOptions(forms: WorkCommonOption[]): WorkCommonOption[] {
  return forms
    .map((form) => {
      const id = textValue(form.id);
      const value =
        textValue(form.name) ||
        textValue(form["label"]) ||
        textValue(form.value);
      return id && value ? { ...form, id, value } : null;
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

function workTaskCanSelectCollaborationTargets(task: WorkTask): boolean {
  return workTaskIsCollaborate(task) && !workTaskIsTodo(task);
}

function initialWorkCollaborationTargets(
  task: WorkTask,
): WorkCollaborationTarget[] {
  return normalizeWorkCollaborationTargets(task.collaboration_items);
}

function normalizeWorkCollaborationTargets(
  value: unknown,
): WorkCollaborationTarget[] {
  return normalizeWorkCollaborationTargetRows(value)
    .filter((target) => !workCollaborationTargetIsEmpty(target));
}

function normalizeWorkCollaborationTargetRows(
  value: unknown,
): WorkCollaborationTarget[] {
  return workRecordArray(value).map((row) => workCollaborationTargetFromRecord(row));
}

function workRecordArray(value: unknown): Record<string, unknown>[] {
  if (Array.isArray(value)) {
    return value.filter(workIsRecord);
  }
  const raw = textValue(value);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    return Array.isArray(parsed) ? parsed.filter(workIsRecord) : [];
  } catch {
    return [];
  }
}

function workIsRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
}

function workCollaborationTargetFromRecord(
  row: Record<string, unknown>,
): WorkCollaborationTarget {
  const explicitStaffLocked = row["staff_locked"] ?? row["staffLocked"];
  const form = workCollaborationTargetForm(row);
  const fields =
    workFormFields(row["fields"]) ||
    workFormFields(form?.fields) ||
    [];
  return {
    key:
      textValue(row["key"]) ||
      textValue(row["target_key"]) ||
      textValue(row["targetKey"]),
    name:
      textValue(row["name"]) ||
      textValue(row["task_name"]) ||
      textValue(row["sub_task_name"]),
    department_id:
      positiveTextID(row["department_id"]) ||
      positiveTextID(row["assignee_department_id"]),
    staff_id:
      positiveTextID(row["staff_id"]) ||
      positiveTextID(row["assignee_staff_id"]),
    form_id: positiveTextID(row["form_id"]),
    form,
    fields,
    staff_locked:
      typeof explicitStaffLocked === "boolean"
        ? explicitStaffLocked
        : Boolean(
            positiveTextID(row["staff_id"]) ||
              positiveTextID(row["assignee_staff_id"]),
          ),
    required:
      typeof row["required"] === "boolean"
        ? row["required"]
        : row["required"] !== false,
    sort: Number(row["sort"]) > 0 ? Number(row["sort"]) : 0,
  };
}

function workCollaborationTargetForm(
  row: Record<string, unknown>,
): WorkCollaborationTarget["form"] {
  const form = row["form"];
  return workIsRecord(form)
    ? {
        id: positiveTextID(form["id"]) || textValue(form["id"]),
        name: textValue(form["name"]),
        fields: workFormFields(form["fields"]) || [],
      }
    : null;
}

function workCollaborationTargetFields(
  target: WorkCollaborationTarget,
): WorkFormField[] {
  return (
    workFormFields(target.fields) ||
    workFormFields(target.form?.fields) ||
    []
  );
}

function workFormFields(value: unknown): WorkFormField[] | null {
  if (!Array.isArray(value)) return null;
  return value.filter(workIsRecord) as WorkFormField[];
}

function workCollaborationTargetKey(target: WorkCollaborationTarget): string {
  const key = textValue(target.key);
  if (key) return key;
  if (
    !target.name &&
    !target.department_id &&
    !target.form_id &&
    !target.sort
  ) {
    return "";
  }
  return [
    target.sort || 0,
    target.department_id || 0,
    target.form_id || 0,
    target.name,
  ].join(":");
}

function workCollaborationTargetIsEmpty(
  target: WorkCollaborationTarget,
): boolean {
  return (
    !textValue(target.department_id) &&
    !textValue(target.staff_id) &&
    !textValue(target.form_id)
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

type WorkLoginStatus = "idle" | "loading" | "success" | "error";

type WorkFeishuConfig = {
  enabled?: boolean;
  app_id?: string;
  appId?: string;
};

function normalizeWorkLoginCode(): string {
  if (typeof window === "undefined") return "";
  const params = new URLSearchParams(window.location.search);
  return textValue(params.get("code"));
}

function normalizeWorkRedirectState(): string {
  const fallback = getWorkEntryPath();
  if (typeof window === "undefined") return fallback;
  const params = new URLSearchParams(window.location.search);
  const target = textValue(params.get("state") || params.get("redirect"));
  return isWorkRedirectTarget(target, fallback) ? target : fallback;
}

function isWorkRedirectTarget(target: string, fallback: string): boolean {
  if (!target || !target.startsWith("/") || target.startsWith("//")) {
    return false;
  }

  const scope = workRedirectScopePath(fallback);
  if (target === `${scope}/sign-in` || target.startsWith(`${scope}/sign-in?`)) {
    return false;
  }
  if (target === `${scope}/login` || target.startsWith(`${scope}/login?`)) {
    return false;
  }
  return target === scope || target.startsWith(`${scope}/`);
}

function workRedirectScopePath(path: string): string {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  const siteKey = normalized.split("/").filter(Boolean)[0];
  return siteKey ? `/${siteKey}` : "/work";
}

function workFeishuAppID(config: WorkFeishuConfig): string {
  return textValue(config.app_id) || textValue(config.appId);
}

export function ShowCrmWorkLogin() {
  const site = getRuntimeSite();
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [loginError, setLoginError] = useState("");
  const [feishuStatus, setFeishuStatus] = useState<WorkLoginStatus>("idle");
  const [feishuMessage, setFeishuMessage] = useState("飞书内可免输密码登录，浏览器中可跳转飞书授权。");
  const [submitting, setSubmitting] = useState(false);
  const feishuAutoLoginRef = useRef(false);

  const finishWorkLogin = (payload: { token?: string; user?: unknown }) => {
    const token = textValue(payload.token);
    if (!token) throw new Error("登录返回缺少 token");
    saveWorkSession(token, payload.user);
    window.location.href = normalizeWorkRedirectState();
  };

  const fetchFeishuConfig = async () => {
    const config = await workApi<WorkFeishuConfig>("/crm/work/feishu_config");
    if (!config.enabled || !workFeishuAppID(config)) {
      throw new Error("服务端未配置飞书登录");
    }
    return config;
  };

  const loginByFeishuCode = async (code: string) => {
    const payload = await workApi<{ token?: string; user?: unknown }>(
      "/crm/work/feishu_login",
      {
        method: "POST",
        body: JSON.stringify({ code }),
      },
    );
    finishWorkLogin(payload);
  };

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
      finishWorkLogin(payload);
    } catch (error) {
      const message = errorMessage(error, "登录失败");
      setLoginError(message);
      toast.error(message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleBrowserFeishuLogin = async () => {
    setLoginError("");
    setFeishuStatus("loading");
    setFeishuMessage("正在发起飞书登录...");
    try {
      const config = await fetchFeishuConfig();
      const redirectURL = new URL(window.location.pathname, window.location.origin);
      window.location.href = buildFeishuOAuthURL(
        workFeishuAppID(config),
        redirectURL.toString(),
        normalizeWorkRedirectState(),
      );
    } catch (error) {
      const message = errorMessage(error, "飞书登录发起失败");
      setFeishuStatus("error");
      setFeishuMessage(message);
      toast.error(message);
    }
  };

  useEffect(() => {
    if (isFeishuClient()) {
      void loadFeishuSDK().catch(() => undefined);
    }
  }, []);

  useEffect(() => {
    const code = normalizeWorkLoginCode();
    if (!code || feishuAutoLoginRef.current) return;

    feishuAutoLoginRef.current = true;
    setFeishuStatus("loading");
    setFeishuMessage("正在校验飞书身份...");
    void loginByFeishuCode(code).catch((error) => {
      const message = errorMessage(error, "飞书登录失败");
      setFeishuStatus("error");
      setFeishuMessage(message);
      toast.error(message);
    });
  }, []);

  useEffect(() => {
    if (!isFeishuClient() || feishuAutoLoginRef.current) return;

    feishuAutoLoginRef.current = true;
    setFeishuStatus("loading");
    setFeishuMessage("正在获取飞书身份...");
    void (async () => {
      const config = await fetchFeishuConfig();
      const code = await getFeishuAuthCode(workFeishuAppID(config));
      await loginByFeishuCode(code);
    })().catch((error) => {
      const message = errorMessage(error, "飞书免登录失败");
      setFeishuStatus("error");
      setFeishuMessage(message);
    });
  }, []);

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
          <div className="relative" style={{ marginTop: 22 }}>
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t" />
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-card px-2 text-muted-foreground">
                或使用飞书
              </span>
            </div>
          </div>
          <button
            type="button"
            disabled={feishuStatus === "loading"}
            onClick={handleBrowserFeishuLogin}
            className={`${outlineButton} h-10 w-full px-4`}
            style={{ marginTop: 20 }}
          >
            {feishuStatus === "loading" ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <LogIn className="h-4 w-4" />
            )}
            飞书登录
          </button>
          <p
            className={`text-xs ${
              feishuStatus === "error"
                ? "text-destructive"
                : "text-muted-foreground"
            }`}
            style={{ marginTop: 10 }}
          >
            {feishuMessage}
          </p>
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
            <table className="w-full min-w-[1280px] table-fixed border-collapse text-sm">
              <thead className="bg-muted/40">
                <tr className="border-b">
                  <WorkTableHead
                    className={`${workCustomerNoTableColumnClass} ${workTableStickyLeftHeadClass}`}
                  >
                    客户编号
                  </WorkTableHead>
                  <WorkTableHead className={workCustomerNameTableColumnClass}>
                    姓名
                  </WorkTableHead>
                  <WorkTableHead className={workPhoneTableColumnClass}>
                    手机号
                  </WorkTableHead>
                  <WorkTableHead className={workWechatTableColumnClass}>
                    微信号
                  </WorkTableHead>
                  <WorkTableHead className={workAssetNoTableColumnClass}>
                    资产编号
                  </WorkTableHead>
                  <WorkTableHead className={workAssetNameTableColumnClass}>
                    资产名称
                  </WorkTableHead>
                  <WorkTableHead>状态</WorkTableHead>
                  <WorkTableHead
                    className={`${workActionTableColumnClass} ${workTableStickyRightHeadClass} text-center`}
                  >
                    操作
                  </WorkTableHead>
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

const workCustomerNoTableColumnClass =
  "w-[17rem] min-w-[17rem] max-w-[17rem]";
const workCustomerNameTableColumnClass =
  "w-[12rem] min-w-[12rem] max-w-[12rem]";
const workPhoneTableColumnClass = "w-[9rem] min-w-[9rem] max-w-[9rem]";
const workWechatTableColumnClass = "w-[10rem] min-w-[10rem] max-w-[10rem]";
const workAssetNoTableColumnClass = "w-[10rem] min-w-[10rem] max-w-[10rem]";
const workAssetNameTableColumnClass =
  "w-[14rem] min-w-[14rem] max-w-[14rem]";
const workActionTableColumnClass =
  "w-[13rem] min-w-[13rem] max-w-[13rem]";

function WorkTableWrappedText({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <span className={`block whitespace-normal break-all leading-5 ${className}`}>
      {children}
    </span>
  );
}

function WorkItemTableRow({
  item,
  store,
}: {
  item: WorkItem;
  store?: StoreLike;
}) {
  const { customer, asset } = item;
  const customerNo = workItemCustomerNo(item);
  const customerName = workCustomerTitle(customer);
  const customerWechat = displayText(customer.wechat);
  const assetNo = workItemAssetNo(item);
  const assetName = asset ? assetTitle(asset) : "";
  const openDetail = () => openWorkDetail(customer, store, asset);

  return (
    <tr className="border-b bg-background odd:bg-background even:bg-muted/20 last:border-b-0">
      <WorkTableCell
        className={`${workCustomerNoTableColumnClass} ${workTableStickyLeftCellClass} bg-inherit text-muted-foreground`}
      >
        <WorkTableWrappedText>{customerNo}</WorkTableWrappedText>
      </WorkTableCell>
      <WorkTableCell className={workCustomerNameTableColumnClass}>
        <button
          type="button"
          className="block w-full whitespace-normal break-all text-left font-medium leading-5"
          onClick={openDetail}
        >
          {customerName}
        </button>
      </WorkTableCell>
      <WorkTableCell>{workCustomerPhone(customer)}</WorkTableCell>
      <WorkTableCell className={workWechatTableColumnClass}>
        <WorkTableWrappedText>{customerWechat}</WorkTableWrappedText>
      </WorkTableCell>
      <WorkTableCell
        className={`${workAssetNoTableColumnClass} text-muted-foreground`}
      >
        <WorkTableWrappedText>{assetNo}</WorkTableWrappedText>
      </WorkTableCell>
      <WorkTableCell className={`${workAssetNameTableColumnClass} font-medium`}>
        {asset ? (
          <WorkTableWrappedText>{assetName}</WorkTableWrappedText>
        ) : (
          <span className="text-muted-foreground">未录入资产</span>
        )}
      </WorkTableCell>
      <WorkTableCell>{renderWorkItemStatus(item)}</WorkTableCell>
      <WorkTableCell
        className={`${workActionTableColumnClass} ${workTableStickyRightCellClass} bg-inherit text-center`}
      >
        <WorkItemActions item={item} store={store} />
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
                <WorkItemActions item={item} store={store} />
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
  store,
}: {
  item: WorkItem;
  store?: StoreLike;
}) {
  const { customer, asset, tasks } = item;
  const openDetail = () => openWorkDetail(customer, store, asset);

  return (
    <div className="flex flex-wrap justify-center gap-2">
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

function useWorkOperations(customerID: string, assetID = "") {
  const [operations, setOperations] = useState<WorkOperation[]>([]);
  const [loading, setLoading] = useState(false);

  const reload = useCallback(async () => {
    if (!customerID) {
      setOperations([]);
      return;
    }
    setLoading(true);
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
      setLoading(false);
    }
  }, [assetID, customerID]);

  return { operations, loading, reload };
}

function WorkCustomerDetailContent({
  customer,
  store,
}: {
  customer: WorkCustomer;
  store?: StoreLike;
}) {
  const [operationScope, setOperationScope] =
    useState<WorkOperationScope>("all");
  const customerID = workCustomerID(customer);
  const customerTasks = customer ? workCustomerRowTasks(customer) : [];
  const [activeTab, setActiveTab] = useState<WorkDetailTab>("base");
  const {
    operations,
    loading: loadingOperations,
    reload: loadOperations,
  } = useWorkOperations(customerID);

  const refreshDetail = useCallback(async () => {
    await refreshWorkDetailTarget(store, customerID);
  }, [customerID, store]);

  useEffect(() => {
    setActiveTab("base");
    setOperationScope("all");
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
                scope={operationScope}
                onScopeChange={setOperationScope}
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
  const [operationScope, setOperationScope] =
    useState<WorkOperationScope>("all");
  const customerID = workCustomerID(customer);
  const assetID = textValue(asset?.id);
  const assetTasks = asset ? workAssetRowTasks(asset) : [];
  const [activeTab, setActiveTab] = useState<WorkDetailTab>("base");
  const {
    operations,
    loading: loadingOperations,
    reload: loadOperations,
  } = useWorkOperations(customerID, assetID);

  const refreshDetail = useCallback(async () => {
    await refreshWorkDetailTarget(store, customerID, assetID);
  }, [assetID, customerID, store]);

  useEffect(() => {
    setActiveTab("base");
    setOperationScope("all");
  }, [assetID]);

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
                scope={operationScope}
                onScopeChange={setOperationScope}
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

type WorkOperationScope = "all" | "mine";

const workOperationScopeOptions: Array<{
  value: WorkOperationScope;
  label: string;
}> = [
  { value: "all", label: "全部记录" },
  { value: "mine", label: "我的记录" },
];

function WorkOperationCards({
  operations,
  loading,
  scope,
  onScopeChange,
  store,
}: {
  operations: WorkOperation[];
  loading: boolean;
  scope: WorkOperationScope;
  onScopeChange: (scope: WorkOperationScope) => void;
  store?: StoreLike;
}) {
  const filteredOperations =
    scope === "mine"
      ? operations.filter((operation) => operation.operator_is_current)
      : operations;

  if (loading) {
    return (
      <div className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        正在加载资料记录
      </div>
    );
  }

  return (
    <div className="grid gap-4">
      <WorkOperationScopeTabs scope={scope} onScopeChange={onScopeChange} />

      {filteredOperations.length === 0 ? (
        <WorkEmptyText>
          {scope === "mine" ? "暂无我的操作记录" : "暂无已收集资料"}
        </WorkEmptyText>
      ) : (
        <div className="grid gap-3">
          {filteredOperations.map((operation, index) => (
            <WorkOperationCard
              key={`${textValue(operation.id) || index}`}
              operation={operation}
              store={store}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function WorkOperationScopeTabs({
  scope,
  onScopeChange,
}: {
  scope: WorkOperationScope;
  onScopeChange: (scope: WorkOperationScope) => void;
}) {
  return (
    <div className="inline-flex w-fit rounded-md border border-border/70 bg-muted/20 p-1">
      {workOperationScopeOptions.map((option) => (
        <button
          type="button"
          key={option.value}
          className={`rounded px-3 py-1.5 text-sm font-medium transition-colors ${
            scope === option.value
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
          onClick={() => onScopeChange(option.value)}
        >
          {option.label}
        </button>
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
  const headerActionTarget = useWorkTaskModalHeaderActionTarget(contentRef);

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
      if (!applyWorkTaskSubmitError(store, message)) {
        toast.error(message || "保存失败");
      }
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
  const aiFillButton = canAIFill ? (
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
  ) : null;

  return (
    <div ref={contentRef} className="contents">
      {aiFillButton && headerActionTarget
        ? createPortal(aiFillButton, headerActionTarget)
        : null}
      {submitting ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          正在保存
        </div>
      ) : null}
    </div>
  );
}

function useWorkTaskModalHeaderActionTarget(
  contentRef: RefObject<HTMLElement | null>,
) {
  const [target, setTarget] = useState<HTMLElement | null>(null);

  useEffect(() => {
    const content = contentRef.current;
    if (!content) {
      setTarget(null);
      return undefined;
    }

    const body = content.closest(".crm-work-task-modal-body");
    const form = content.closest("form");
    const dialog = form?.closest('[role="dialog"]');
    const header = form?.previousElementSibling;
    const headerActions =
      header instanceof HTMLElement
        ? header.querySelector<HTMLElement>(".flex.shrink-0.items-start.gap-2")
        : null;

    if (!body || !dialog || !headerActions) {
      setTarget(null);
      return undefined;
    }

    const mount = document.createElement("div");
    mount.setAttribute("data-crm-work-task-ai-fill-action", "true");
    mount.className = "contents";
    headerActions.insertBefore(mount, headerActions.lastElementChild);
    setTarget(mount);

    return () => {
      mount.remove();
      setTarget(null);
    };
  }, [contentRef]);

  return target;
}

export function ShowCrmWorkCollaborationTargets({
  item,
  store,
  value,
  setValue,
  error,
}: WorkNodeProps) {
  const meta = item?.meta || {};
  const departments = normalizeWorkCommonOptions(meta["departments"]);
  const staffs = normalizeWorkCommonOptions(meta["staffs"]);
  const forms = normalizeWorkCommonOptions(meta["forms"]);
  const defaultName = textValue(meta["defaultName"]) || "协作子任务";
  const targets = workCollaborationTargetRowsForEdit(value);
  const errorMessage =
    error ||
    (item?.value
      ? workStoreValue<Record<string, string>>(store, "errors", {})[
          item.value
        ]
      : "");

  const updateTargets = useCallback(
    (nextTargets: WorkCollaborationTarget[]) => {
      setValue?.(nextTargets);
    },
    [setValue],
  );

  const updateTarget = useCallback(
    (
      index: number,
      patch: Partial<WorkCollaborationTarget>,
    ) => {
      updateTargets(
        targets.map((target, targetIndex) =>
          targetIndex === index
            ? {
                ...target,
                ...patch,
              }
            : target,
        ),
      );
    },
    [targets, updateTargets],
  );

  return (
    <div className="w-full space-y-3">
      <div className="overflow-hidden rounded-lg border border-border/70 bg-background">
        <div className="hidden border-b bg-muted/30 text-xs font-medium text-muted-foreground md:grid md:grid-cols-12">
          <div className="col-span-3 px-3 py-2">子任务</div>
          <div className="col-span-3 px-3 py-2">目标部门</div>
          <div className="col-span-3 px-3 py-2">处理人员</div>
          <div className="col-span-3 px-3 py-2">处理资料模板</div>
        </div>
        {targets.length === 0 ? (
          <div className="px-3 py-4 text-sm text-muted-foreground">
            请先在后台配置协作对象
          </div>
        ) : null}
        {targets.map((target, index) => {
          const staffOptions = collaborationStaffOptions(
            staffs,
            target.department_id,
          );
          const departmentName = workOptionValueByID(
            departments,
            target.department_id,
          );
          const selectedStaffName = workOptionValueByID(staffs, target.staff_id);
          const formName = workOptionValueByID(forms, target.form_id);
          const staffLocked = Boolean(target.staff_locked && target.staff_id);
          return (
            <div
              key={index}
              className="grid gap-2 border-b px-3 py-3 last:border-b-0 md:grid-cols-12 md:items-center md:py-2"
            >
              <div className="space-y-1 md:col-span-3 md:space-y-0">
                <div className="text-xs font-medium text-muted-foreground md:hidden">
                  子任务
                </div>
                <div className="rounded-md border border-transparent py-2 text-sm text-foreground break-words">
                  {target.name || defaultName}
                </div>
              </div>
              <div className="space-y-1 md:col-span-3 md:space-y-0">
                <div className="text-xs font-medium text-muted-foreground md:hidden">
                  目标部门
                </div>
                <div className="rounded-md border border-transparent py-2 text-sm text-foreground break-words">
                  {departmentName || target.department_id || "-"}
                </div>
              </div>
              <div className="space-y-1 md:col-span-3 md:space-y-0">
                <div className="text-xs font-medium text-muted-foreground md:hidden">
                  处理人员
                </div>
                {staffLocked ? (
                  <div className="rounded-md border border-transparent py-2 text-sm text-foreground break-words">
                    {selectedStaffName || target.staff_id || "-"}
                  </div>
                ) : (
                  <select
                    className={inputClassName}
                    value={target.staff_id}
                    disabled={!target.department_id}
                    onChange={(event) =>
                      updateTarget(index, { staff_id: event.target.value })
                    }
                  >
                    <option value="">
                      {target.department_id
                        ? "请选择处理人员"
                        : "请先配置目标部门"}
                    </option>
                    {staffOptions.map((staff) => (
                      <option key={staff.id} value={staff.id}>
                        {staff.value}
                      </option>
                    ))}
                  </select>
                )}
              </div>
              <div className="space-y-1 md:col-span-3 md:space-y-0">
                <div className="text-xs font-medium text-muted-foreground md:hidden">
                  处理资料模板
                </div>
                <div className="rounded-md border border-transparent py-2 text-sm text-foreground break-words">
                  {formName || target.form_id || "-"}
                </div>
              </div>
            </div>
          );
        })}
      </div>
      <p className="text-xs text-muted-foreground">
        协作对象由后台任务配置决定；后台未指定处理人员时，需要选择目标部门内的处理人员。
      </p>
      {errorMessage ? (
        <p className="text-xs text-destructive">{errorMessage}</p>
      ) : null}
    </div>
  );
}

function workCollaborationTargetRowsForEdit(
  value: unknown,
): WorkCollaborationTarget[] {
  return normalizeWorkCollaborationTargetRows(value).filter(
    (target) => !workCollaborationTargetIsEmpty(target),
  );
}

function normalizeWorkCommonOptions(value: unknown): WorkCommonOption[] {
  return Array.isArray(value)
    ? value
        .filter(workIsRecord)
        .map((option) => ({
          ...option,
          id: textValue(option["id"]),
          value:
            textValue(option["value"]) ||
            textValue(option["name"]) ||
            textValue(option["label"]) ||
            textValue(option["real_name"]),
        }))
        .filter((option) => option.id && option.value)
    : [];
}

function collaborationStaffOptions(
  staffs: WorkCommonOption[],
  departmentID: string,
): WorkCommonOption[] {
  const selectedDepartmentID = textValue(departmentID);
  if (!selectedDepartmentID) return [];
  return staffs.filter(
    (staff) => textValue(staff.department_id) === selectedDepartmentID,
  );
}

function workOptionValueByID(
  options: WorkCommonOption[],
  id: unknown,
): string {
  const selectedID = textValue(id);
  if (!selectedID) return "";
  return textValue(
    options.find((option) => textValue(option.id) === selectedID)?.value,
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
      ? workStoreValue<unknown>(store, relationPath, undefined)
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
        setWorkStoreValue(store, relationPath, nextFiles);
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
  if (typeof validateForm === "function" && !validateForm()) return false;
  return validateCurrentWorkTaskDomainRules(store);
}

function validateCurrentWorkTaskDomainRules(
  store: StoreLike | undefined,
): boolean {
  const collaborationError = currentWorkTaskCollaborationTargetsError(store);
  if (!collaborationError) return true;

  const errorKey = currentWorkTaskFormErrorKey(store, "collaboration_targets");
  if (errorKey) {
    setCurrentWorkTaskFormErrors(store, {
      [errorKey]: collaborationError,
    });
  } else {
    toast.error(collaborationError);
  }
  return false;
}

function currentWorkTaskCollaborationTargetsError(
  store: StoreLike | undefined,
): string {
  const fieldMap = workStoreValue<Record<string, string>>(
    store,
    workTaskFieldMapPath,
    {},
  );
  const formKey = Object.entries(fieldMap).find(
    ([, rawKey]) => workTaskRawMainField(rawKey) === "collaboration_targets",
  )?.[0];
  if (!formKey) return "";

  const formValues = workStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    {},
  );
  const targets = normalizeWorkCollaborationTargets(formValues[formKey]);
  if (targets.length === 0) return "请先在后台配置协作对象";
  if (targets.some((target) => !target.department_id)) {
    return "协作子任务目标部门不能为空";
  }
  if (targets.some((target) => !target.staff_id)) {
    return "请选择协作子任务处理人员";
  }
  return "";
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
      values[rawKey] =
        workTaskRawMainField(rawKey) === "collaboration_targets"
          ? submitWorkCollaborationTargets(formValues[formKey])
          : formValues[formKey];
      return values;
    },
    {},
  );
}

function submitWorkCollaborationTargets(value: unknown): Record<string, unknown>[] {
  return normalizeWorkCollaborationTargets(value).map((target) => {
    const result: Record<string, unknown> = {
      name: target.name,
      department_id: target.department_id,
      staff_id: target.staff_id,
      required: target.required,
      sort: target.sort,
    };
    if (target.key) result.key = target.key;
    if (target.form_id) result.form_id = target.form_id;
    if (target.staff_locked) result.staff_locked = target.staff_locked;
    return result;
  });
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
  if (!workTaskShouldRenderFields(task)) return false;
  return (task.form?.fields || []).some((field) => !workTaskFieldIsUpload(field));
}

function clearCurrentWorkTaskFormErrors(store: StoreLike | undefined) {
  setCurrentWorkTaskFormErrors(store, {});
}

function applyWorkTaskSubmitError(
  store: StoreLike | undefined,
  message: string,
): boolean {
  const errorField = workTaskSubmitErrorField(message);
  if (!errorField) return false;

  const errorKey = currentWorkTaskFormErrorKey(store, errorField);
  if (!errorKey) return false;

  setCurrentWorkTaskFormErrors(store, {
    [errorKey]: message,
  });
  return true;
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
  if (
    message.includes("协作子任务") ||
    message.includes("协作对象") ||
    message.includes("目标部门") ||
    message.includes("处理人员")
  ) {
    return "collaboration_targets";
  }
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
