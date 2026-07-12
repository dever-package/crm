import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent, ReactNode, RefObject } from "react";
import { createPortal } from "react-dom";
import {
  Check,
  ClipboardList,
  Download,
  Inbox,
  LogIn,
  Loader2,
  Plus,
  RefreshCw,
  Search,
  TrendingUp,
  UserRound,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { downloadUploadFile, type UploadFileItem } from "@/lib/upload";
import { normalizeUploadItems } from "@/lib/resource";

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
  type WorkAIFillResponse,
  type WorkAsset,
  type WorkBusinessObject,
  type WorkCommonOption,
  type WorkCustomer,
  type WorkCustomerMode,
  type WorkCustomerScope,
  type WorkDataCompletenessTemplate,
  type WorkDisplayField,
  type WorkFieldOption,
  type WorkFormField,
  type WorkFlowDetail,
  type WorkItem,
  type WorkNodeProps,
  type WorkOperation,
  type WorkOperationSummaryItem,
  type WorkPageStoreState,
  type WorkSearchFilters,
  type WorkStoreLike,
  type WorkSummary,
  type WorkSummaryBreakdown,
  type WorkSummaryMetric,
  type WorkSummaryTrendPoint,
  type WorkTask,
  type WorkTaskFieldRenderConfig,
  type WorkTaskFormNode,
  type WorkTaskFormState,
  type WorkTodo,
} from "./work-core";
import { WorkFlowActions } from "./work-flow-actions";
import {
  buildFeishuOAuthURL,
  getFeishuAuthCode,
  isFeishuClient,
  loadFeishuSDK,
} from "./feishu-login";
import {
  ShowCrmWorkTaskUpload,
  WorkTaskUploadPreviewDialog,
} from "./work-upload";
import {
  CrmEChart,
  crmChartAxisColor,
  crmChartSplitLineColor,
  crmChartTextColor,
  type EChartsOption,
} from "./crm-echarts";

export { ShowCrmWorkTaskUpload } from "./work-upload";

type StoreLike = WorkStoreLike;

type WorkTaskGroupField = {
  formKey: string;
  label: string;
  placeholder: string;
  required: boolean;
  type: string;
  options?: WorkCommonOption[];
  meta?: Record<string, unknown>;
};

type WorkTaskGroupTab = {
  id: string;
  label: string;
  fields: WorkTaskGroupField[];
};

type WorkTaskFieldSection = {
  id: string;
  label: string;
  fields: WorkTaskGroupField[];
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

function workBusinessObjectID(object?: WorkBusinessObject | null): string {
  return positiveTextID(object?.id) || positiveTextID(object?.business_object_id);
}

function workBusinessObjectTitle(object?: WorkBusinessObject | null): string {
  return (
    textValue(object?.object_name) ||
    textValue(object?.object_no) ||
    textValue(object?.business_object_type_name) ||
    "业务对象"
  );
}

function workBusinessObjectOptions(
  asset: WorkAsset | undefined,
  task: WorkTask,
): WorkCommonOption[] {
  const typeID = positiveTextID(task.business_object_type_id);
  const objects = Array.isArray(asset?.business_objects)
    ? asset.business_objects
    : [];
  return objects
    .filter(
      (object) =>
        !typeID || positiveTextID(object.business_object_type_id) === typeID,
    )
    .map((object) => ({
      id: workBusinessObjectID(object),
      value: [workBusinessObjectTitle(object), textValue(object.object_no)]
        .filter(Boolean)
        .join(" / "),
    }))
    .filter((option) => Boolean(option.id));
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
  return textValue(task.task_type) || "todo";
}

function workTaskIsTodo(task: WorkTask): boolean {
  return workTaskAction(task) === "todo";
}

function workTaskIsForm(task: WorkTask): boolean {
  return workTaskAction(task) === "form";
}

function workTaskIsApproval(task: WorkTask): boolean {
  return workTaskAction(task) === "approval";
}

function workTaskIsRule(task: WorkTask): boolean {
  return workTaskAction(task) === "rule";
}

function workTaskAllowsProgress(task: WorkTask): boolean {
  return workTaskIsForm(task);
}

function workTaskNeedsCompleteAction(task: WorkTask): boolean {
  return !workTaskIsRule(task);
}

function workTaskShouldRenderFields(task: WorkTask): boolean {
  return workTaskIsForm(task) && (task.form?.fields || []).length > 0;
}

function workTaskButtonLabel(task: WorkTask): string {
  const name = workTaskName(task);
  if (name && name !== "任务") return name;
  if (workTaskIsForm(task)) return "填写资料";
  if (workTaskIsApproval(task)) return "审核";
  if (workTaskIsRule(task)) return "自动核验";
  return "办理事项";
}

function confirmWorkTaskSubmit(
  task: WorkTask,
  mode: "complete" | "progress",
): boolean {
  if (mode === "progress") return true;
  const message = workTaskIsForm(task)
    ? "确认提交资料并完成当前任务吗？"
    : "";
  if (!message) return true;
  if (typeof globalThis.confirm !== "function") return true;
  return globalThis.confirm(message);
}

function workTaskSubmitSuccessMessage(
  task: WorkTask,
  mode: "complete" | "progress",
): string {
  if (mode === "progress") return "进度已保存";
  if (workTaskIsApproval(task)) return "审核结果已提交";
  return workTaskIsForm(task) ? "资料已提交" : "任务已完成";
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
  page: number,
  pageSize: number,
  workFilters: {
    quickFilter: WorkQuickFilter;
    stageFilter: string;
    taskFilter: string;
    scope: WorkCustomerScope;
  },
): string {
  const params = new URLSearchParams(
    workSearchQuery(filters).replace(/^\?/, ""),
  );
  params.set("mode", mode);
  params.set("page", String(page));
  params.set("page_size", String(pageSize));
  params.set("scope", workFilters.scope);
  if (workFilters.quickFilter && workFilters.quickFilter !== "all") {
    params.set("quick_filter", workFilters.quickFilter);
  }
  if (workFilters.stageFilter) {
    params.set("stage_filter", workFilters.stageFilter);
  }
  if (workFilters.taskFilter) {
    params.set("task_filter", workFilters.taskFilter);
  }
  const query = params.toString();
  return query ? `?${query}` : "";
}

function currentWorkURLParams(): URLSearchParams {
  if (typeof window === "undefined") return new URLSearchParams();
  return new URLSearchParams(window.location.search);
}

function workSearchFiltersFromURL(): WorkSearchFilters {
  const params = currentWorkURLParams();
  return {
    customerNo: textValue(params.get("customer_no") || params.get("customerNo")),
    customerName: textValue(params.get("customer_name") || params.get("customerName")),
    phone: textValue(params.get("phone")),
    wechat: textValue(params.get("wechat")),
    assetNo: textValue(params.get("asset_no") || params.get("assetNo")),
    status: textValue(params.get("status")),
  };
}

function workURLFilterValue(...keys: string[]): string {
  const params = currentWorkURLParams();
  for (const key of keys) {
    const value = textValue(params.get(key));
    if (value) return value;
  }
  return "";
}

function workCustomerModeFromNode(
  item?: WorkNodeProps["item"],
): WorkCustomerMode {
  const urlMode = textValue(currentWorkURLParams().get("mode"));
  if (urlMode === "all" || urlMode === "done" || urlMode === "pending") {
    return urlMode;
  }
  const configured = textValue(item?.meta?.mode || item?.meta?.customerMode);
  if (configured === "all") return "all";
  if (configured === "done") return "done";
  if (configured === "pending") return "pending";
  const pathname = textValue(window.location.pathname);
  return pathname.endsWith("/work/done") || pathname.includes("/work/done/")
    ? "done"
    : "pending";
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
  let taskCustomer = customer;
  let taskAsset = asset;
  let fullTask = task;
  const customerID =
    workCustomerID(customer) || positiveTextID(task.customer_id);
  const assetID = workAssetID(asset) || positiveTextID(task.asset_id);
  let detail: WorkDetailTargetResponse | null = null;
  if (customerID) {
    try {
      detail = await refreshWorkDetailTarget(store, customerID, assetID);
      taskCustomer = detail?.customer || customer;
      taskAsset = assetID ? detail?.asset || asset : undefined;
      fullTask =
        findWorkDetailTask(fullTask, taskCustomer, taskAsset) || fullTask;
    } catch (error) {
      toast.error(errorMessage(error, "客户资料加载失败"));
    }
  }
  setWorkStoreValue(store, "data.actionTarget.workTask", fullTask);
  setWorkStoreValue(store, "data.actionTarget.workTaskCustomer", taskCustomer);
  setWorkStoreValue(store, "data.actionTarget.workTaskAsset", taskAsset ?? null);
  setWorkStoreValue(
    store,
    "data.actionTarget.workTaskName",
    workTaskButtonLabel(fullTask),
  );
  await prepareWorkTaskForm(store, fullTask, taskCustomer, taskAsset);
  setWorkModalOpen(store, "dialog.workTask", true);
}

function findWorkDetailTask(
  task: WorkTask,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
): WorkTask | null {
  const tasks = asset
    ? workAssetRowTasks(asset)
    : workCustomerRowTasks(customer || null);
  return tasks.find((candidate) => sameWorkTask(candidate, task)) || null;
}

function sameWorkTask(left: WorkTask, right: WorkTask): boolean {
  const leftTodoID = positiveTextID(left.todo_id);
  const rightTodoID = positiveTextID(right.todo_id);
  if (leftTodoID || rightTodoID) {
    return leftTodoID !== "" && leftTodoID === rightTodoID;
  }
  const leftTaskID = positiveTextID(left.id);
  const rightTaskID = positiveTextID(right.id);
  return leftTaskID !== "" && leftTaskID === rightTaskID;
}

async function prepareWorkTaskForm(
  store: StoreLike | undefined,
  task: WorkTask,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
) {
  const formState = buildWorkTaskFormState(task, customer, asset);
  setWorkStoreValue(store, workTaskFormDataPath, formState.values);
  setWorkStoreValue(store, workTaskFieldMapPath, formState.fieldMap);
  replaceWorkTaskFormSection(store, formState.nodes);
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
): WorkTaskFormState {
  const nodes: WorkTaskFormNode[] = [];
  const values: Record<string, unknown> = {};
  const fieldMap: Record<string, string> = {};
  addWorkTaskBusinessObjectNode(nodes, values, fieldMap, task, asset);

  if (workTaskShouldRenderFields(task)) {
    const groupFields: WorkFormField[] = [];
    const sectionFields: WorkFormField[] = [];
    for (const field of task.form?.fields || []) {
      if (workFormFieldIsGroup(field)) {
        groupFields.push(field);
        continue;
      }
      sectionFields.push(field);
    }
    addWorkTaskFieldSectionNodes(
      nodes,
      values,
      fieldMap,
      sectionFields,
      customer,
      asset,
    );
    addWorkTaskGroupTabsNode(
      nodes,
      values,
      fieldMap,
      groupFields,
      customer,
      asset,
    );
  }
  addWorkTaskActionFields(nodes, values, fieldMap, task);

  nodes.push({
    id: "work-task-submit-controller",
    type: "show-crm-work-task-form",
  });

  return { nodes, values, fieldMap };
}

function addWorkTaskBusinessObjectNode(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  task: WorkTask,
  asset?: WorkAsset,
) {
  if (!positiveTextID(task.business_object_type_id)) return;
  const existing = workBusinessObjectOptions(asset, task);
  if (existing.length === 0) return;
  const typeName = textValue(task.business_object_type_name) || "运营记录";
  const formKey = uniqueWorkTaskFormKey("business_object_id", fieldMap);
  values[formKey] = existing[0].id;
  fieldMap[formKey] = "business_object_id";
  nodes.push({
    id: "work-task-business-object-section",
    type: "show-crm-work-task-field-section",
    meta: {
      title: "关联记录",
      fields: [
        {
          formKey,
          label: typeName,
          placeholder: `请选择${typeName}`,
          required: true,
          type: "form-select",
          options: [...existing, { id: "0", value: `新建${typeName}` }],
        },
      ],
    },
  });
}

function addWorkTaskActionFields(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  task: WorkTask,
) {
  if (workTaskIsTodo(task)) {
    addWorkTaskTextNode(nodes, values, fieldMap, {
      formKey: "result",
      rawKey: "result",
      label: "办理结果",
      placeholder: "请输入本次办理结果",
      required: true,
      type: "form-textarea",
    });
    return;
  }
  if (!workTaskIsApproval(task)) return;
  addWorkTaskSelectNode(nodes, values, fieldMap, {
    formKey: "approval_result",
    rawKey: "approval_result",
    label: "审核结果",
    placeholder: "请选择审核结果",
    required: true,
    options: [
      { id: "approved", value: "通过" },
      { id: "rejected", value: "驳回" },
    ],
  });
  addWorkTaskTextNode(nodes, values, fieldMap, {
    formKey: "opinion",
    rawKey: "opinion",
    label: "审核意见",
    placeholder: "请输入审核意见",
    required: true,
    type: "form-textarea",
  });
}

function workFormFieldIsGroup(field: WorkFormField): boolean {
  return textValue(field.field_type) === "group";
}

function addWorkTaskFieldSectionNodes(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  fields: WorkFormField[],
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
) {
  const sections = workTaskFieldSections(fields);
  if (sections.length <= 1) {
    for (const field of fields) {
      addWorkTaskFieldNode(nodes, values, fieldMap, field, customer, asset);
    }
    return;
  }

  for (const section of sections) {
    const controls = section.fields
      .map((field) =>
        workTaskGroupField(field, values, fieldMap, customer, asset),
      )
      .filter((field): field is WorkTaskGroupField => Boolean(field));
    if (controls.length === 0) continue;
    nodes.push({
      id: `work-task-field-section-${section.id}`,
      type: "show-crm-work-task-field-section",
      meta: {
        title: section.label,
        fields: controls,
      },
    });
  }
}

function workTaskFieldSections(fields: WorkFormField[]): Array<{
  id: string;
  label: string;
  fields: WorkFormField[];
}> {
  type FieldSectionDraft = {
    id: string;
    label: string;
    fields: WorkFormField[];
  };
  const sections = new Map<string, FieldSectionDraft>();
  for (const field of fields) {
    const section = workTaskFieldSection(field);
    if (!sections.has(section.id)) {
      sections.set(section.id, { ...section, fields: [] });
    }
    sections.get(section.id)?.fields.push(field);
  }
  return ["customer", "asset", "other"]
    .map((id) => sections.get(id))
    .filter((section): section is FieldSectionDraft =>
      Boolean(section && section.fields.length > 0),
    );
}

function workTaskFieldSection(field: WorkFormField): {
  id: string;
  label: string;
} {
  if (workFormFieldBelongsToAsset(field)) {
    return { id: "asset", label: "资产信息" };
  }
  if (workFormFieldBelongsToCustomer(field)) {
    return { id: "customer", label: "客户信息" };
  }
  return { id: "other", label: "补充信息" };
}

function workFormFieldBelongsToCustomer(field: WorkFormField): boolean {
  return positiveTextID(field.data_template_cate_id) === "1";
}

function workFormFieldBelongsToAsset(field: WorkFormField): boolean {
  if (positiveTextID(field.data_template_cate_id) === "2") return true;
  switch (textValue(field.main_field)) {
    case "asset_name":
    case "asset_status_id":
      return true;
    default:
      return false;
  }
}

function addWorkTaskGroupTabsNode(
  nodes: WorkTaskFormNode[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  groupFields: WorkFormField[],
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
) {
  const tabs = groupFields
    .map((group) =>
      workTaskGroupTab(group, values, fieldMap, customer, asset),
    )
    .filter((tab): tab is WorkTaskGroupTab => Boolean(tab?.fields.length));

  if (tabs.length === 0) return;

  nodes.push({
    id: "work-task-group-tabs",
    type: "show-crm-work-task-group-tabs",
    meta: {
      tabs,
    },
  });
}

function workTaskGroupTab(
  group: WorkFormField,
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
): WorkTaskGroupTab | null {
  const children = Array.isArray(group.children) ? group.children : [];
  const fields = children
    .filter((field) => !workFormFieldIsGroup(field))
    .map((field) =>
      workTaskGroupField(field, values, fieldMap, customer, asset),
    )
    .filter((field): field is WorkTaskGroupField => Boolean(field));
  if (fields.length === 0) return null;
  const label =
    textValue(group.label) ||
    textValue(group.name) ||
    textValue(group.field_key);
  return {
    id: workTaskFormKey(workFieldKey(group) || label || "group"),
    label: label || "分组",
    fields,
  };
}

function workTaskGroupField(
  field: WorkFormField,
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
): WorkTaskGroupField | null {
  const rawKey = workFieldKey(field);
  if (!rawKey) return null;
  const options = Array.isArray(field.options)
    ? field.options.map(workFieldOption)
    : [];
  const renderConfig = workTaskFieldRenderConfig(field, options);
  const formKey = uniqueWorkTaskFormKey(workTaskFormKey(rawKey), fieldMap);
  const label = textValue(field.label) || textValue(field.name) || rawKey;
  values[formKey] = formatWorkTaskInitialValue({
    type: renderConfig.type,
    initialValue: workFieldInitialValue(
      field,
      customer,
      asset,
      renderConfig.type,
    ),
  });
  fieldMap[formKey] = rawKey;
  return {
    formKey,
    label,
    placeholder: `${renderConfig.placeholderPrefix}${label}`,
    required: Boolean(field.required),
    type: renderConfig.type,
    options: renderConfig.options,
    meta: renderConfig.meta,
  };
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

  if (fieldType === "boolean") {
    return {
      type: "form-switch",
      placeholderPrefix: "",
      meta: {
        trueValue: true,
        falseValue: false,
      },
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

function workIsRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
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

type WorkDetailTargetResponse = {
  customer?: WorkCustomer;
  asset?: WorkAsset | null;
  operations?: WorkOperation[];
  list?: WorkOperation[];
  todos?: WorkTodo[];
  flow?: WorkFlowDetail | null;
};

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
): Promise<WorkDetailTargetResponse | null> {
  if (!customerID) return null;
  const query = new URLSearchParams({ customer_id: customerID });
  if (assetID) query.set("asset_id", assetID);
  const payload = await workApi<WorkDetailTargetResponse>(
    `/crm/work/customer_detail?${query.toString()}`,
  );
  if (payload.customer) {
    setWorkDetailTarget(store, payload.customer, payload.asset ?? null);
  }
  return payload;
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
  const statusName = workStatusName(asset || customer);
  const statusText = statusName === "-" ? "" : statusName;
  if (asset) {
    return [workCustomerTitle(customer), statusText]
      .map(textValue)
      .filter(Boolean)
      .join(" / ");
  }
  return statusText;
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

type WorkRecordSummaryGroup = {
  id: string;
  label: string;
  items: WorkOperationSummaryItem[];
};

function workRecordSummaryGroups(
  items: WorkOperationSummaryItem[],
): WorkRecordSummaryGroup[] {
  const groups = new Map<string, WorkRecordSummaryGroup>();
  for (const item of items) {
    if (workRecordSummaryItemEmpty(item)) continue;
    const group = workRecordSummaryItemGroup(item);
    if (!groups.has(group.id)) {
      groups.set(group.id, { ...group, items: [] });
    }
    groups.get(group.id)?.items.push(item);
  }
  return Array.from(groups.values()).filter((group) => group.items.length > 0);
}

function workRecordSummaryItemGroup(item: WorkOperationSummaryItem): {
  id: string;
  label: string;
} {
  const groupLabel = textValue(
    item.group_label || item.groupLabel || item.group_name || item.groupName,
  );
  const groupID = textValue(item.group_id || item.groupId) || groupLabel;
  if (groupID || groupLabel) {
    return {
      id: workTaskFormKey(groupID || groupLabel),
      label: groupLabel || groupID,
    };
  }
  return { id: "default", label: "提交明细" };
}

function workRecordSummaryItemEmpty(item: WorkOperationSummaryItem): boolean {
  if (textValue(item.value_type) === "files") {
    return normalizeUploadItems(item.files).length === 0;
  }
  const value = item.value;
  if (value === null || value === undefined || value === "") return true;
  if (Array.isArray(value)) return value.length === 0;
  const text = textValue(value).trim();
  return text === "" || text === "[]" || text === "{}";
}

function workOperationTitle(
  operation: WorkOperation,
  fallback = "操作记录",
): string {
  return displayText(
    operation.title ||
      operation.operation_name ||
      operation.task_name ||
      operation["task.name"],
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

type WorkTaskListState = {
  tasks: WorkTask[];
  loading: boolean;
};

type WorkQuickFilter =
  | "all"
  | "hasTasks"
  | "missingAsset"
  | "archived";

type WorkCustomerPageState = {
  page: number;
  pageSize: number;
  total: number;
};

type WorkCustomerModeCounts = Record<WorkCustomerMode, number>;

const workCustomerPageSize = 10;

const workTopFilterOptions: Array<{
  key: string;
  label: string;
  mode: WorkCustomerMode;
  quickFilter: WorkQuickFilter;
}> = [
  { key: "pending", label: "待处理", mode: "pending", quickFilter: "all" },
  { key: "done", label: "已结束", mode: "done", quickFilter: "all" },
  { key: "all", label: "全部", mode: "all", quickFilter: "all" },
];

function workTopFilterKey(mode: WorkCustomerMode): string {
  return mode;
}

function emptyWorkCustomerModeCounts(): WorkCustomerModeCounts {
  return {
    pending: 0,
    done: 0,
    all: 0,
  };
}

function normalizeWorkCustomerModeCounts(
  counts: Partial<Record<WorkCustomerMode, unknown>> | undefined,
  activeMode: WorkCustomerMode,
  activeTotal: unknown,
): WorkCustomerModeCounts {
  const normalized = emptyWorkCustomerModeCounts();
  for (const option of workTopFilterOptions) {
    normalized[option.mode] = workPositiveNumber(counts?.[option.mode]);
  }
  const activeCount = Number(activeTotal);
  const hasActiveCount = Object.prototype.hasOwnProperty.call(
    counts || {},
    activeMode,
  );
  if (!hasActiveCount && Number.isFinite(activeCount) && activeCount >= 0) {
    normalized[activeMode] = Math.floor(activeCount);
  }
  return normalized;
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

export function ShowCrmWorkHeaderActions({ store }: WorkNodeProps = {}) {
  const taskList = useWorkTaskList();
  return (
    <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
      <ShowCrmWorkRefreshButton />
      <WorkGlobalTaskButtons
        tasks={taskList.tasks}
        loading={taskList.loading}
        store={store}
        align="end"
      />
    </div>
  );
}

function useWorkTaskList(): WorkTaskListState {
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

  return { tasks, loading };
}

export function ShowCrmWorkTasks({ store }: WorkNodeProps = {}) {
  const { tasks, loading } = useWorkTaskList();
  return (
    <WorkGlobalTaskButtons tasks={tasks} loading={loading} store={store} />
  );
}

function WorkGlobalTaskButtons({
  tasks,
  loading,
  store,
  align = "start",
}: {
  tasks: WorkTask[];
  loading: boolean;
  store?: StoreLike;
  align?: "start" | "end";
}) {
  if (loading) {
    return (
      <Button type="button" variant="outline" size="sm" disabled>
        <Loader2 className="h-4 w-4 animate-spin" />
        加载任务
      </Button>
    );
  }

  if (tasks.length === 0) {
    return null;
  }
  const actionableTasks = tasks.filter((task) => !workTaskIsRule(task));
  if (actionableTasks.length === 0) return null;

  const openTask = (task: WorkTask) => {
    void openRowTask(null, task, store);
  };

  return (
    <div
      className={`flex flex-wrap items-center gap-2 ${
        align === "end" ? "justify-end" : ""
      }`}
    >
      {actionableTasks.map((task) => (
        <Button
          type="button"
          key={workTaskKey(task)}
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

export function ShowCrmWorkStats() {
  const [summary, setSummary] = useState<WorkSummary | null>(null);
  const [loading, setLoading] = useState(false);
  const loadVersionRef = useRef(0);

  const loadSummary = useCallback(async () => {
    const version = loadVersionRef.current + 1;
    loadVersionRef.current = version;
    setLoading(true);
    try {
      const payload = await workApi<WorkSummary>("/crm/work/summary");
      if (loadVersionRef.current !== version) return;
      setSummary(payload || {});
    } catch (error) {
      if (loadVersionRef.current !== version) return;
      toast.error(errorMessage(error, "工作台加载失败"));
      setSummary(null);
    } finally {
      if (loadVersionRef.current === version) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    loadSummary();
  }, [loadSummary]);

  useEffect(() => {
    window.addEventListener(workRefreshEvent, loadSummary);
    return () => window.removeEventListener(workRefreshEvent, loadSummary);
  }, [loadSummary]);

  if (loading && !summary) {
    return (
      <div className="rounded-lg border border-border/70 bg-background px-6 py-20 shadow-sm">
        <WorkStatusState
          icon="loading"
          title="正在加载统计"
          description="请稍候，正在汇总当前工作数据"
        />
      </div>
    );
  }

  if (!summary) {
    return (
      <div className="rounded-lg border border-border/70 bg-background px-6 py-20 shadow-sm">
        <WorkStatusState
          icon="empty"
          title="暂无统计数据"
          description="刷新后仍无数据时，请先确认当前账号是否有可查看客户"
        />
      </div>
    );
  }

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
}

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

function WorkStatsMetricCard({
  metric,
  onOpen,
}: {
  metric: WorkSummaryMetric;
  onOpen: () => void;
}) {
  const Icon = workStatsMetricIcon(metric.key);
  return (
    <button
      type="button"
      className="min-h-[94px] bg-background px-4 py-3 text-left transition-colors hover:bg-muted/20 focus-visible:z-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
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

function workStatsMetricDrilldown(metric: WorkSummaryMetric): Record<string, string> {
  switch (textValue(metric.key)) {
    case "pending_targets":
    case "pending_tasks":
      return { mode: "pending" };
    case "missing_assets":
      return { mode: "all" };
    case "recent_operations":
      return { mode: "done" };
    default:
      return { mode: "all" };
  }
}

function openWorkCustomerList(filters: Record<string, string>) {
  const base = `${getWorkEntryPath().replace(/\/$/, "")}/work`;
  const params = new URLSearchParams();
  Object.entries(filters).forEach(([key, value]) => {
    const text = textValue(value);
    if (text) params.set(key, text);
  });
  const query = params.toString();
  window.location.assign(query ? `${base}?${query}` : base);
}

function workStatsMetricIcon(key?: string) {
  switch (textValue(key)) {
    case "customers":
      return UserRound;
    case "assets":
      return Inbox;
    case "pending_targets":
    case "pending_tasks":
      return ClipboardList;
    case "missing_assets":
      return Inbox;
    case "recent_operations":
      return TrendingUp;
    default:
      return ClipboardList;
  }
}

type WorkStatsTrendSeriesKey =
  | "task_count"
  | "transition_count"
  | "operation_count";

const workStatsTrendSeries: Array<{
  key: WorkStatsTrendSeriesKey;
  label: string;
  color: string;
}> = [
  { key: "task_count", label: "任务完成", color: "#111827" },
  { key: "transition_count", label: "阶段流转", color: "#2563eb" },
  { key: "operation_count", label: "操作记录", color: "#059669" },
];

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

function WorkStatsTrendCard({ points }: { points: WorkSummaryTrendPoint[] }) {
  return (
    <section
      className={`${workStatsPanelClass} flex min-h-0 flex-col xl:h-[312px]`}
    >
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

function workStatsTrendPointValue(
  point: WorkSummaryTrendPoint,
  key: WorkStatsTrendSeriesKey,
): number {
  const value = Number(point[key]);
  return Number.isFinite(value) && value > 0 ? value : 0;
}

function buildWorkStatsTrendOption(
  points: WorkSummaryTrendPoint[],
): EChartsOption {
  return {
    animationDuration: 280,
    color: workStatsTrendSeries.map((series) => series.color),
    grid: {
      left: 8,
      right: 18,
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
      textStyle: { color: crmChartTextColor, fontSize: 11 },
    },
    tooltip: {
      trigger: "axis",
      confine: true,
      borderColor: crmChartAxisColor,
      backgroundColor: "#ffffff",
      textStyle: { color: "#0f172a" },
      valueFormatter: (value) => `${workStatsNumber(value)} 次`,
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
    series: workStatsTrendSeries.map((series) => ({
      name: series.label,
      type: "line",
      smooth: true,
      symbol: "circle",
      symbolSize: 5,
      lineStyle: { width: 2, color: series.color },
      itemStyle: { color: series.color },
      emphasis: { focus: "series" },
      data: points.map((point) => workStatsTrendPointValue(point, series.key)),
    })),
  };
}

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
    <section className="rounded-lg border border-border/70 bg-background p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold leading-6">{title}</h3>
          <p className="text-sm leading-6 text-muted-foreground">
            {description}
          </p>
        </div>
        <TrendingUp className="h-5 w-5 shrink-0 text-muted-foreground/70" />
      </div>
      <div className="mt-5">
        {rows.length === 0 ? (
          <WorkEmptyText>{emptyText}</WorkEmptyText>
        ) : (
          <>
            <WorkStatsBreakdownChart rows={rows} />
            <WorkStatsBreakdownDrilldowns rows={rows} type={drilldownType} />
          </>
        )}
      </div>
    </section>
  );
}

function WorkStatsBreakdownDrilldowns({
  rows,
  type,
}: {
  rows: WorkSummaryBreakdown[];
  type: "stage" | "task";
}) {
  return (
    <div className="mt-4 flex flex-wrap gap-2">
      {rows.slice(0, 8).map((row) => {
        const value = textValue(row.key || row.name);
        const params =
          type === "stage"
            ? { mode: "all", stage_filter: value }
            : { mode: "pending", task_filter: value };
        return (
          <button
            type="button"
            key={`${type}:${value}`}
            className="rounded-full border border-border/70 bg-background px-3 py-1 text-xs font-medium text-muted-foreground transition hover:border-primary/40 hover:text-foreground"
            onClick={() => openWorkCustomerList(params)}
          >
            {displayText(row.name)} · {workStatsNumber(row.count)}
          </button>
        );
      })}
    </div>
  );
}

function WorkStatsBreakdownChart({
  rows,
}: {
  rows: WorkSummaryBreakdown[];
}) {
  const option = useMemo(() => buildWorkStatsBreakdownOption(rows), [rows]);
  return (
    <CrmEChart
      option={option}
      height={Math.max(220, rows.length * 42 + 72)}
      minWidth={520}
      ariaLabel="统计分布"
    />
  );
}

function buildWorkStatsBreakdownOption(
  rows: WorkSummaryBreakdown[],
): EChartsOption {
  return {
    animationDuration: 240,
    grid: {
      left: 8,
      right: 52,
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
        const index = Number((params as { dataIndex?: number }).dataIndex) || 0;
        const row = rows[index];
        return [
          displayText(row?.name),
          `数量：${workStatsNumber(row?.count)} 个`,
          `占比：${workStatsPercent(row?.percent)}%`,
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
        data: rows.map((row) => workStatsNumber(row.count)),
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

function workPositiveNumber(value: unknown): number {
  const number = Number(value);
  return Number.isFinite(number) && number > 0 ? number : 0;
}

function workStatsNumber(value: unknown): number {
  return workPositiveNumber(value);
}

function workStatsPercent(value: unknown): number {
  const number = Number(value);
  if (!Number.isFinite(number) || number < 0) return 0;
  if (number > 100) return 100;
  return Math.round(number);
}

function WorkStatsRecentOperations({
  operations,
  loading,
}: {
  operations: WorkOperation[];
  loading: boolean;
}) {
  return (
    <section
      className={`${workStatsPanelClass} flex min-h-0 flex-col xl:h-[312px]`}
    >
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

function WorkStatsOperationRow({ operation }: { operation: WorkOperation }) {
  const tone = workOperationTone(operation);
  const description = workOperationDescription(operation);
  return (
    <article className="border-b border-border/60 py-2.5 last:border-b-0">
      <div className="flex min-w-0 items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-1.5">
            <span className="min-w-0 truncate text-xs font-medium text-foreground">
              {workOperationTitle(operation)}
            </span>
            <span
              className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${tone.badge}`}
            >
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
  );
}

export function ShowCrmWorkCustomerTable({ item, store }: WorkNodeProps) {
  const [customers, setCustomers] = useState<WorkCustomer[]>([]);
  const taskList = useWorkTaskList();
  const [filters, setFilters] = useState<WorkSearchFilters>(
    workSearchFiltersFromURL,
  );
  const [activeFilters, setActiveFilters] = useState<WorkSearchFilters>(
    workSearchFiltersFromURL,
  );
  const [mode, setMode] = useState<WorkCustomerMode>(() =>
    workCustomerModeFromNode(item),
  );
  const [quickFilter, setQuickFilter] = useState<WorkQuickFilter>("all");
  const [stageFilter, setStageFilter] = useState(() =>
    workURLFilterValue("stage_filter", "stage"),
  );
  const [taskFilter, setTaskFilter] = useState(() =>
    workURLFilterValue("task_filter", "task"),
  );
  const [modeCounts, setModeCounts] = useState<WorkCustomerModeCounts>(
    emptyWorkCustomerModeCounts,
  );
  const [scope, setScope] = useState<WorkCustomerScope>("mine");
  const [canDispatch, setCanDispatch] = useState(false);
  const [pageState, setPageState] = useState<WorkCustomerPageState>({
    page: 1,
    pageSize: workCustomerPageSize,
    total: 0,
  });
  const [loading, setLoading] = useState(false);
  const loadVersionRef = useRef(0);
  const loadingQueryRef = useRef("");
  const modeConfig = workCustomerModeConfig[mode];

  const loadCustomers = useCallback(async (page = 1) => {
    const query = workCustomerQuery(
      activeFilters,
      mode,
      page,
      workCustomerPageSize,
      {
        quickFilter,
        stageFilter,
        taskFilter,
        scope,
      },
    );
    if (loadingQueryRef.current === query) return;
    const version = loadVersionRef.current + 1;
    loadVersionRef.current = version;
    loadingQueryRef.current = query;
    setLoading(true);
    try {
      const payload = await workApi<{
        list?: WorkCustomer[];
        customers?: WorkCustomer[];
        data?: WorkCustomer[];
        total?: string | number;
        page?: string | number;
        page_size?: string | number;
        mode_counts?: Partial<Record<WorkCustomerMode, string | number>>;
        can_dispatch?: boolean;
        scope?: WorkCustomerScope;
      }>(`/crm/work/customers${query}`);
      if (loadVersionRef.current !== version) return;
      const list = payload.list || payload.customers || payload.data || [];
      const nextCustomers = Array.isArray(list) ? list : [];
      const nextTotal = Number(payload.total) || nextCustomers.length;
      setCustomers(nextCustomers);
      setCanDispatch(Boolean(payload.can_dispatch));
      setPageState({
        page: Number(payload.page) || page,
        pageSize: Number(payload.page_size) || workCustomerPageSize,
        total: nextTotal,
      });
      setModeCounts(
        normalizeWorkCustomerModeCounts(payload.mode_counts, mode, nextTotal),
      );
    } catch (error) {
      if (loadVersionRef.current !== version) return;
      toast.error(errorMessage(error, "客户列表加载失败"));
    } finally {
      if (loadingQueryRef.current === query) {
        loadingQueryRef.current = "";
      }
      if (loadVersionRef.current === version) {
        setLoading(false);
      }
    }
  }, [activeFilters, mode, quickFilter, scope, stageFilter, taskFilter]);

  useEffect(() => {
    setCustomers([]);
    loadCustomers(1);
  }, [loadCustomers]);

  useEffect(() => {
    const handler = () => {
      setCustomers([]);
      loadCustomers(1);
    };
    window.addEventListener(workRefreshEvent, handler);
    return () => window.removeEventListener(workRefreshEvent, handler);
  }, [loadCustomers]);

  const workItems = useMemo(() => buildWorkItems(customers), [customers]);
  const initialLoading = loading && customers.length === 0;
  const goToPage = (nextPage: number) => {
    if (loading || nextPage === pageState.page) return;
    setCustomers([]);
    loadCustomers(nextPage);
  };

  const submitSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setActiveFilters(filters);
  };

  const resetSearch = () => {
    const emptyFilters = emptyWorkSearchFilters();
    setFilters(emptyFilters);
    setActiveFilters(emptyFilters);
    setQuickFilter("all");
    setStageFilter("");
    setTaskFilter("");
    setMode(workCustomerModeFromNode(item));
  };

  return (
    <div className="space-y-4">
      <WorkCustomerListHeader
        mode={mode}
        modeCounts={modeCounts}
        taskList={taskList}
        scope={scope}
        canDispatch={canDispatch}
        store={store}
        onScopeChange={(nextScope) => {
          setCustomers([]);
          setScope(nextScope);
        }}
        onModeChange={(nextMode, nextQuickFilter) => {
          setMode(nextMode);
          setQuickFilter(nextQuickFilter);
        }}
      />
      <div className="overflow-hidden rounded-lg bg-background shadow-sm">
        <form
          onSubmit={submitSearch}
          className="flex flex-wrap items-center gap-2.5 border-b border-border/70 bg-muted/10 px-5 py-4"
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
            <Search className="h-4 w-4" />
            搜索
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={resetSearch}
            disabled={loading}
          >
            <RefreshCw className="h-4 w-4" />
            重置
          </Button>
        </form>
        <div className="p-4 md:hidden">
          <WorkItemCardList
            items={workItems}
            loading={initialLoading}
            emptyTitle={modeConfig.emptyTitle}
            emptyDescription={modeConfig.emptyDescription}
            store={store}
          />
        </div>

        <div className="hidden overflow-hidden bg-background md:block">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[1080px] table-fixed border-collapse text-sm">
              <thead className="bg-[#f8fafc]">
                <tr className="border-b border-border/70">
                  <WorkTableHead
                    className={`${workCustomerTableColumnClass} ${workTableStickyLeftHeadClass}`}
                  >
                    客户
                  </WorkTableHead>
                  <WorkTableHead className={workContactTableColumnClass}>
                    联系方式
                  </WorkTableHead>
                  <WorkTableHead className={workAssetTableColumnClass}>
                    房产/资产
                  </WorkTableHead>
                  <WorkTableHead className={workStageTableColumnClass}>
                    当前阶段
                  </WorkTableHead>
                  <WorkTableHead
                    className={`${workActionTableColumnClass} ${workTableStickyRightHeadClass} text-center`}
                  >
                    操作
                  </WorkTableHead>
                </tr>
              </thead>
              <tbody>
                {initialLoading ? (
                  <tr>
                    <td colSpan={5} className="px-6 py-24">
                      <WorkStatusState
                        icon="loading"
                        title="正在加载"
                        description="请稍候，正在同步最新数据"
                      />
                    </td>
                  </tr>
                ) : workItems.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-6 py-16">
                      <WorkStatusState
                        icon="empty"
                        title={modeConfig.emptyTitle}
                        description={modeConfig.emptyDescription}
                      />
                    </td>
                  </tr>
                ) : (
                  workItems.map((item) => (
                    <WorkItemTableRow key={item.id} item={item} store={store} />
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
        <WorkCustomerPagination
          loading={loading}
          hidden={initialLoading}
          pageState={pageState}
          onPageChange={goToPage}
        />
      </div>
    </div>
  );
}

function WorkCustomerListHeader({
  mode,
  modeCounts,
  taskList,
  scope,
  canDispatch,
  store,
  onModeChange,
  onScopeChange,
}: {
  mode: WorkCustomerMode;
  modeCounts: WorkCustomerModeCounts;
  taskList: WorkTaskListState;
  scope: WorkCustomerScope;
  canDispatch: boolean;
  store?: StoreLike;
  onModeChange: (mode: WorkCustomerMode, quickFilter: WorkQuickFilter) => void;
  onScopeChange: (scope: WorkCustomerScope) => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <WorkCustomerModeTabs
        mode={mode}
        modeCounts={modeCounts}
        onChange={onModeChange}
      />
      <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
        {canDispatch ? (
          <WorkCustomerScopeToggle scope={scope} onChange={onScopeChange} />
        ) : null}
        <ShowCrmWorkRefreshButton />
        <WorkGlobalTaskButtons
          tasks={taskList.tasks}
          loading={taskList.loading}
          store={store}
          align="end"
        />
      </div>
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
    <div className="inline-flex rounded-md border border-border/60 bg-muted/30 p-1 shadow-sm">
      {([
        ["mine", "我的"],
        ["all", "全部"],
      ] as const).map(([value, label]) => (
        <button
          type="button"
          key={value}
          className={`rounded px-2.5 py-1 text-xs font-medium transition-colors ${
            scope === value
              ? "bg-background text-foreground shadow-sm ring-1 ring-border/60"
              : "text-muted-foreground hover:text-foreground"
          }`}
          onClick={() => onChange(value)}
        >
          {label}
        </button>
      ))}
    </div>
  );
}

function WorkCustomerModeTabs({
  mode,
  onChange,
  modeCounts,
}: {
  mode: WorkCustomerMode;
  onChange: (mode: WorkCustomerMode, quickFilter: WorkQuickFilter) => void;
  modeCounts: WorkCustomerModeCounts;
}) {
  const activeKey = workTopFilterKey(mode);
  return (
    <div className="inline-flex rounded-md border border-border/60 bg-muted/30 p-1 shadow-sm">
      {workTopFilterOptions.map((option) => (
        <button
          type="button"
          key={option.key}
          className={`rounded px-3 py-1.5 text-sm font-medium transition-colors ${
            activeKey === option.key
              ? "bg-background text-foreground shadow-sm ring-1 ring-border/60"
              : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
          }`}
          onClick={() => onChange(option.mode, option.quickFilter)}
        >
          {option.label}({modeCounts[option.mode] || 0})
        </button>
      ))}
    </div>
  );
}

function WorkCustomerPagination({
  loading,
  hidden,
  pageState,
  onPageChange,
}: {
  loading: boolean;
  hidden: boolean;
  pageState: WorkCustomerPageState;
  onPageChange: (page: number) => void;
}) {
  if (hidden || pageState.total <= 0) return null;
  const pageSize = pageState.pageSize || workCustomerPageSize;
  const totalPages = Math.max(1, Math.ceil(pageState.total / pageSize));
  const currentPage = Math.min(
    totalPages,
    Math.max(1, Number(pageState.page) || 1),
  );

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border/70 px-5 py-3 text-xs text-muted-foreground">
      <span>
        第 {currentPage} / {totalPages} 页，每页 {pageSize} 条，共{" "}
        {pageState.total} 条
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

const workCustomerTableColumnClass =
  "w-[20rem] min-w-[20rem] max-w-[20rem]";
const workContactTableColumnClass =
  "w-[15rem] min-w-[15rem] max-w-[15rem]";
const workAssetTableColumnClass =
  "w-[20rem] min-w-[20rem] max-w-[20rem]";
const workStageTableColumnClass =
  "w-[11rem] min-w-[11rem] max-w-[11rem]";
const workActionTableColumnClass =
  "w-[18rem] min-w-[18rem] max-w-[18rem]";

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

function WorkTableMetaLine({
  label,
  value,
}: {
  label: string;
  value: unknown;
}) {
  return (
    <div className="flex min-w-0 gap-2 whitespace-normal leading-5">
      <span className="shrink-0 text-xs text-muted-foreground">{label}</span>
      <span className="min-w-0 break-all text-foreground">
        {displayText(value)}
      </span>
    </div>
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
  const assetName = asset ? assetTitle(asset) : "";
  const openDetail = () => openWorkDetail(customer, store, asset);

  return (
    <tr className="border-b border-border/60 bg-background transition-colors hover:bg-muted/25 last:border-b-0">
      <WorkTableCell
        className={`${workCustomerTableColumnClass} ${workTableStickyLeftCellClass} bg-inherit`}
      >
        <button
          type="button"
          className="block w-full whitespace-normal break-all text-left hover:text-primary"
          onClick={openDetail}
        >
          <span className="block font-semibold leading-5 text-foreground">
            {customerName}
          </span>
          <span className="mt-1 block text-xs leading-5 text-muted-foreground">
            {customerNo}
          </span>
        </button>
      </WorkTableCell>
      <WorkTableCell className={workContactTableColumnClass}>
        <WorkTableMetaLine label="手机" value={workCustomerPhone(customer)} />
        <WorkTableMetaLine label="微信" value={customerWechat} />
      </WorkTableCell>
      <WorkTableCell className={`${workAssetTableColumnClass} font-medium`}>
        {asset ? (
          <>
            <WorkTableWrappedText>{assetName}</WorkTableWrappedText>
            <WorkTableWrappedText className="mt-1 text-xs text-muted-foreground">
              {workItemAssetNo(item)}
            </WorkTableWrappedText>
          </>
        ) : (
          <div className="whitespace-normal rounded-md bg-muted/25 px-3 py-2 text-muted-foreground">
            未录入房产，后续任务补充
          </div>
        )}
      </WorkTableCell>
      <WorkTableCell className={workStageTableColumnClass}>
        <WorkItemStageCell item={item} />
      </WorkTableCell>
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
  store,
}: {
  items: WorkItem[];
  loading: boolean;
  emptyTitle: string;
  emptyDescription: string;
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
          <article key={item.id} className="rounded-md border bg-background shadow-sm">
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
                    {asset ? assetTitle(asset) : "未录入房产"}
                  </span>
                  <WorkItemStageCell item={item} compact />
                </div>
                <div className="mt-1 truncate text-xs text-muted-foreground">
                  {asset
                    ? `资产编号：${workItemAssetNo(item)}`
                    : "后续任务补充房产资料"}
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

function WorkItemStageCell({
  item,
  compact = false,
}: {
  item: WorkItem;
  compact?: boolean;
}) {
  const target = item.asset || item.customer;
  const stageDays = workPositiveNumber(target.stage_days);
  const lastOperatedAt = textValue(target.last_operated_at);
  return (
    <div className={compact ? "text-right" : "grid gap-1.5"}>
      {renderWorkItemStatus(item)}
      <div
        className={`text-xs leading-5 text-muted-foreground ${
          compact ? "mt-1" : ""
        }`}
      >
        {stageDays > 0 ? `停留 ${stageDays} 天` : "今日进入"}
      </div>
      {lastOperatedAt ? (
        <div className="text-xs leading-5 text-muted-foreground">
          最近 {formatWorkDate(lastOperatedAt)}
        </div>
      ) : null}
    </div>
  );
}

function WorkStatusFrame({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-md border bg-background px-4 py-20">{children}</div>
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
    <div className="mx-auto flex max-w-[15rem] flex-col items-stretch gap-2">
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="justify-start border-transparent bg-transparent shadow-none hover:bg-muted/60"
        onClick={openDetail}
      >
        <UserRound className="h-4 w-4" />
        详情
      </Button>
      {tasks.map((task, index) =>
        workTaskIsRule(task) ? (
          <div
            key={workTaskKey(task)}
            className="flex min-w-0 items-start gap-2 rounded-md bg-amber-50 px-3 py-2 text-left text-amber-900"
            title={textValue(task.result) || "等待资料满足核验条件"}
          >
            <RefreshCw className="mt-0.5 h-4 w-4 shrink-0" />
            <span className="min-w-0">
              <span className="block truncate text-sm font-medium">
                {workTaskButtonLabel(task)}
              </span>
              <span className="block truncate text-xs opacity-80">
                {textValue(task.result) || "等待资料满足核验条件"}
              </span>
            </span>
          </div>
        ) : (
          <Button
            type="button"
            key={workTaskKey(task)}
            variant={index === 0 ? "default" : "outline"}
            size="sm"
            className={
              index === 0
                ? "min-w-0 justify-start shadow-none"
                : "min-w-0 justify-start border-transparent bg-muted/35 shadow-none hover:bg-muted/60"
            }
            onClick={() => openRowTask(customer, task, store, asset)}
          >
            <ClipboardList className="h-4 w-4" />
            <span className="max-w-[9rem] truncate">
              {workTaskButtonLabel(task)}
            </span>
          </Button>
        ),
      )}
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
  const summaryGroups = useMemo(
    () => workRecordSummaryGroups(summaryItems),
    [summaryItems],
  );
  const content = summaryItems.length > 0 ? record.content || record.remark : "";
  const [previewFile, setPreviewFile] = useState<UploadFileItem | null>(null);
  const [activeGroupID, setActiveGroupID] = useState(
    summaryGroups[0]?.id || "",
  );

  useEffect(() => {
    if (summaryGroups.length === 0) {
      setActiveGroupID("");
      return;
    }
    if (!summaryGroups.some((group) => group.id === activeGroupID)) {
      setActiveGroupID(summaryGroups[0].id);
    }
  }, [activeGroupID, summaryGroups]);

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
      <WorkRecordDetailHeader record={record} />
      {summaryGroups.length > 0 ? (
        <WorkRecordSummaryGroups
          groups={summaryGroups}
          activeGroupID={activeGroupID}
          onActiveGroupChange={setActiveGroupID}
          onPreviewFile={setPreviewFile}
        />
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

function WorkRecordSummaryGroups({
  groups,
  activeGroupID,
  onActiveGroupChange,
  onPreviewFile,
}: {
  groups: WorkRecordSummaryGroup[];
  activeGroupID: string;
  onActiveGroupChange: (groupID: string) => void;
  onPreviewFile: (file: UploadFileItem) => void;
}) {
  const activeGroup =
    groups.find((group) => group.id === activeGroupID) || groups[0];
  if (!activeGroup) return null;

  return (
    <div className="grid gap-3">
      {groups.length > 1 ? (
        <div className="flex flex-wrap gap-2 border-b border-border/60">
          {groups.map((group) => {
            const active = group.id === activeGroup.id;
            return (
              <button
                type="button"
                key={group.id}
                className={`border-b-2 px-1.5 py-2 text-sm font-medium transition-colors ${
                  active
                    ? "border-primary text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                }`}
                onClick={() => onActiveGroupChange(group.id)}
              >
                {group.label}
              </button>
            );
          })}
        </div>
      ) : null}
      <WorkRecordSummaryTable
        items={activeGroup.items}
        onPreviewFile={onPreviewFile}
      />
    </div>
  );
}

function WorkRecordSummaryTable({
  items,
  onPreviewFile,
}: {
  items: WorkOperationSummaryItem[];
  onPreviewFile: (file: UploadFileItem) => void;
}) {
  if (items.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-muted-foreground/60">
        暂无提交明细
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border/50">
      <table className="w-full text-sm">
        <tbody>
          {items.map((item, index) => (
            <tr
              key={textValue(item.key || item.label || index)}
              className="border-b border-border/30 last:border-b-0"
            >
              <td className="w-[100px] min-w-[100px] bg-muted/15 px-4 py-2.5 text-muted-foreground">
                {displayText(item.label, "-")}
              </td>
              <td className="px-4 py-2.5 font-medium text-foreground/85">
                <WorkRecordSummaryValue
                  item={item}
                  onPreviewFile={onPreviewFile}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function WorkRecordDetailHeader({ record }: { record: WorkOperation }) {
  const tone = workOperationTone(record);
  const stage = workOperationStageLabel(record);
  const operator = textValue(
    record.operator_name || record["operator_staff.name"],
  );
  const result = workOperationDescription(record);
  const resultName = workOperationResultName(record);
  return (
    <div className={`rounded-lg border bg-muted/10 px-4 py-3 ${tone.border}`}>
      <div className="flex min-w-0 flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <span
              className={`rounded-full px-2 py-0.5 text-[11px] font-medium leading-5 ${tone.badge}`}
            >
              {workOperationBadgeText(record)}
            </span>
            <h3 className="break-words text-sm font-semibold leading-6 text-foreground">
              {workOperationTitle(record)}
            </h3>
          </div>
          {result ? (
            <p className="mt-1 text-sm leading-6 text-muted-foreground">
              {result}
            </p>
          ) : null}
        </div>
        <div className="shrink-0 whitespace-nowrap text-xs leading-6 text-muted-foreground">
          {workRecordTime(record)}
        </div>
      </div>
      <div className="mt-3 grid gap-2 sm:grid-cols-3">
        <WorkRecordMetaPill label="阶段" value={stage || "-"} />
        <WorkRecordMetaPill label="操作人" value={operator || "-"} />
        <WorkRecordMetaPill label="结果" value={resultName} />
      </div>
    </div>
  );
}

function WorkRecordMetaPill({
  label,
  value,
}: {
  label: string;
  value: unknown;
}) {
  return (
    <div className="rounded-md border border-border/40 bg-background px-3 py-2">
      <div className="text-xs leading-5 text-muted-foreground">{label}</div>
      <div className="mt-0.5 min-w-0 break-words text-sm font-medium text-foreground">
        {displayText(value)}
      </div>
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

function useWorkDetailData(
  initialCustomer: WorkCustomer,
  initialAsset: WorkAsset | null | undefined,
  store?: StoreLike,
) {
  const customerID = workCustomerID(initialCustomer);
  const assetID = textValue(initialAsset?.id);
  const initialCustomerRef = useRef(initialCustomer);
  const initialAssetRef = useRef<WorkAsset | null>(initialAsset ?? null);
  const [customer, setCustomer] = useState(initialCustomer);
  const [asset, setAsset] = useState<WorkAsset | null>(initialAsset ?? null);
  const [operations, setOperations] = useState<WorkOperation[]>([]);
  const [todos, setTodos] = useState<WorkTodo[]>([]);
  const [flow, setFlow] = useState<WorkFlowDetail | null>(null);
  const [loading, setLoading] = useState(false);

  const reload = useCallback(async () => {
    if (!customerID) {
      setCustomer(initialCustomerRef.current);
      setAsset(initialAssetRef.current);
      setOperations([]);
      setTodos([]);
      setFlow(null);
      return;
    }
    setLoading(true);
    try {
      const data = await refreshWorkDetailTarget(store, customerID, assetID);
      if (data?.customer) setCustomer(data.customer);
      if (assetID) setAsset(data?.asset ?? null);
      const list = Array.isArray(data?.operations)
        ? data.operations
        : Array.isArray(data?.list)
          ? data.list
          : [];
      setOperations(list);
      setTodos(Array.isArray(data?.todos) ? data.todos : []);
      setFlow(data?.flow ?? null);
    } catch (error) {
      toast.error(errorMessage(error, "详情加载失败"));
      setOperations([]);
      setTodos([]);
      setFlow(null);
    } finally {
      setLoading(false);
    }
  }, [assetID, customerID, store]);

  useEffect(() => {
    initialCustomerRef.current = initialCustomer;
    initialAssetRef.current = initialAsset ?? null;
  }, [initialAsset, initialCustomer]);

  useEffect(() => {
    setCustomer(initialCustomerRef.current);
    setAsset(initialAssetRef.current);
    setOperations([]);
    setTodos([]);
    setFlow(null);
  }, [assetID, customerID]);

  useEffect(() => {
    reload();
  }, [reload]);

  useEffect(() => {
    if (!customerID) {
      return undefined;
    }
    window.addEventListener(workRefreshEvent, reload);
    return () => {
      window.removeEventListener(workRefreshEvent, reload);
    };
  }, [customerID, reload]);

  return { customer, asset, operations, todos, flow, loading, reload };
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
  const [activeTab, setActiveTab] = useState<WorkDetailTab>("overview");
  const {
    customer: detailCustomer,
    operations,
    todos,
    loading: loadingDetail,
  } = useWorkDetailData(customer, null, store);

  useEffect(() => {
    setActiveTab("overview");
    setOperationScope("all");
  }, [customerID]);

  return (
    <div className="grid gap-6">
      <WorkDetailTabs activeTab={activeTab} onChange={setActiveTab} />

      <div>
        {activeTab === "overview" ? (
          <WorkCustomerOverview
            customer={detailCustomer}
            operations={operations}
            todos={todos}
            loadingOperations={loadingDetail}
            store={store}
          />
        ) : activeTab === "flow" ? (
          <WorkDetailOverview
            operations={operations}
            loadingOperations={loadingDetail}
            operationScope={operationScope}
            onOperationScopeChange={setOperationScope}
            store={store}
          />
        ) : (
          <WorkCustomerMainInfo customer={detailCustomer} />
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
  const assetID = textValue(asset?.id);
  const [activeTab, setActiveTab] = useState<WorkDetailTab>("overview");
  const {
    customer: detailCustomer,
    asset: detailAsset,
    operations,
    todos,
    flow,
    loading: loadingDetail,
  } = useWorkDetailData(customer, asset, store);

  useEffect(() => {
    setActiveTab("overview");
    setOperationScope("all");
  }, [assetID]);

  return (
    <div className="grid gap-6">
      <WorkDetailTabs activeTab={activeTab} onChange={setActiveTab} />

      <div>
        {activeTab === "overview" ? (
          <WorkCustomerOverview
            customer={detailCustomer}
            asset={detailAsset || undefined}
            operations={operations}
            todos={todos}
            loadingOperations={loadingDetail}
            store={store}
          />
        ) : activeTab === "flow" ? (
          <div className="grid gap-5">
            <WorkFlowActions flow={flow} loading={loadingDetail} />
            <WorkDetailOverview
              operations={operations}
              loadingOperations={loadingDetail}
              operationScope={operationScope}
              onOperationScopeChange={setOperationScope}
              store={store}
            />
          </div>
        ) : (
          <WorkCustomerMainInfo
            customer={detailCustomer}
            asset={detailAsset || undefined}
          />
        )}
      </div>
    </div>
  );
}

type WorkDetailTab = "overview" | "info" | "flow";

function WorkDetailOverview({
  operations,
  loadingOperations,
  operationScope,
  onOperationScopeChange,
  store,
}: {
  operations: WorkOperation[];
  loadingOperations: boolean;
  operationScope: WorkOperationScope;
  onOperationScopeChange: (scope: WorkOperationScope) => void;
  store?: StoreLike;
}) {
  return (
    <div className="grid gap-5">
      <WorkOperationCards
        operations={operations}
        loading={loadingOperations}
        scope={operationScope}
        onScopeChange={onOperationScopeChange}
        store={store}
      />
    </div>
  );
}

function WorkCustomerOverview({
  customer,
  asset,
  operations,
  todos,
  loadingOperations,
  store,
}: {
  customer: WorkCustomer;
  asset?: WorkAsset;
  operations: WorkOperation[];
  todos: WorkTodo[];
  loadingOperations: boolean;
  store?: StoreLike;
}) {
  const primaryAsset = asset || workOverviewPrimaryAsset(customer);
  const target = primaryAsset || customer;
  const taskNames = workOverviewTaskNames(target);
  const currentStage = displayText(
    target.current_stage_name || target.stage_name,
    "-",
  );
  const pendingTodoCount = todos.filter(
    (todo) => textValue(todo.status) === "pending",
  ).length;
  const businessObject = workOverviewPrimaryBusinessObject(primaryAsset);
  const completenessItems = workOverviewCompletenessItems(
    customer,
    primaryAsset,
    businessObject,
  );
  const completenessPercent = workOverviewAverageCompleteness(completenessItems);
  const latestOperations = workOverviewLatestOperations(operations);

  return (
    <div className="grid gap-5">
      <section className="rounded-lg bg-muted/20 px-5 py-4">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0">
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <h3 className="break-words text-lg font-semibold leading-7">
                {workCustomerTitle(customer)}
              </h3>
              {renderStatus(target)}
            </div>
            <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-sm leading-6 text-muted-foreground">
              <span>{workCustomerNo(customer)}</span>
              <span>{workCustomerPhone(customer)}</span>
              {primaryAsset ? <span>{workAssetNo(primaryAsset)}</span> : null}
            </div>
          </div>
          <div className="text-right text-sm leading-6 text-muted-foreground">
            <div>当前阶段</div>
            <div className="font-medium text-foreground">{currentStage}</div>
          </div>
        </div>
        <div className="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <WorkOverviewMetric
            label="当前任务"
            value={workOverviewTaskSummary(taskNames)}
          />
          <WorkOverviewMetric
            label="待办任务"
            value={`${pendingTodoCount} 个待处理`}
          />
          <WorkOverviewMetric label="资料完整度" value={completenessPercent} />
          <WorkOverviewMetric
            label="最近操作"
            value={
              latestOperations[0]
                ? formatWorkDate(
                    latestOperations[0].created_at ||
                      latestOperations[0].create_time,
                  )
                : "-"
            }
          />
        </div>
      </section>

      <div className="grid gap-5 lg:grid-cols-[minmax(0,1.1fr)_minmax(20rem,0.9fr)]">
        <div className="grid gap-5">
          <WorkOverviewPanel title="客户与资产">
            <WorkOverviewRows
              rows={[
                ["联系人", workCustomerName(customer)],
                ["手机", workCustomerPhone(customer)],
                ["微信", displayText(customer.wechat)],
                [
                  "来源",
                  displayText(customer.source_name || customer.source),
                ],
                [
                  "渠道",
                  displayText(customer.channel_name || customer.channel),
                ],
                [
                  "等级",
                  displayText(customer.level_name || customer.customer_level),
                ],
                ["资产", primaryAsset ? assetTitle(primaryAsset) : "未录入资产"],
                [
                  "资产状态",
                  primaryAsset
                    ? displayText(
                        primaryAsset.asset_status_name ||
                          primaryAsset.status_name,
                      )
                    : "-",
                ],
                [
                  "资产地址",
                  workOverviewFieldValue(
                    primaryAsset?.display_fields,
                    ["资产地址", "房产地址", "坐落地址", "地址"],
                  ),
                ],
              ]}
            />
          </WorkOverviewPanel>

          <WorkOverviewPanel title="租赁与收款">
            {businessObject ? (
              <WorkOverviewRows
                rows={[
                  ["租赁记录", workBusinessObjectTitle(businessObject)],
                  ["记录编号", displayText(businessObject.object_no)],
                  ["状态", displayText(businessObject.object_status)],
                  [
                    "租户",
                    workOverviewFieldValue(
                      businessObject.display_fields,
                      [
                        "租户姓名",
                        "租客姓名",
                        "承租人姓名",
                        "承租人",
                        "租户",
                        "租客",
                      ],
                    ),
                  ],
                  [
                    "月租金",
                    workOverviewFieldValue(
                      businessObject.display_fields,
                      ["月租金", "租金", "出租月租", "租赁月租"],
                    ),
                  ],
                  [
                    "收款状态",
                    workOverviewFieldValue(
                      businessObject.display_fields,
                      [
                        "租户付款状态",
                        "付款状态",
                        "收款状态",
                        "租金收款状态",
                      ],
                    ),
                  ],
                  [
                    "费用/收款",
                    workOverviewFinanceSummary(businessObject.display_fields),
                  ],
                ]}
              />
            ) : (
              <WorkEmptyText>暂无租赁记录。</WorkEmptyText>
            )}
          </WorkOverviewPanel>
        </div>

        <div className="grid gap-5 content-start">
          <WorkOverviewPanel title="资料完整度">
            <WorkOverviewCompletenessList items={completenessItems} />
          </WorkOverviewPanel>
          <WorkOverviewPanel title="最近记录">
            <WorkOverviewRecentOperations
              operations={latestOperations}
              loading={loadingOperations}
              store={store}
            />
          </WorkOverviewPanel>
        </div>
      </div>
    </div>
  );
}

function WorkOverviewMetric({
  label,
  value,
}: {
  label: string;
  value: unknown;
}) {
  return (
    <div className="rounded-md bg-background px-4 py-3">
      <div className="text-xs leading-5 text-muted-foreground">{label}</div>
      <div className="mt-1 min-w-0 break-words text-sm font-semibold leading-6">
        {displayText(value)}
      </div>
    </div>
  );
}

function WorkOverviewPanel({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="grid gap-3">
      <h3 className="text-[15px] font-semibold leading-6">{title}</h3>
      <div className="rounded-lg bg-muted/15 px-4 py-3">{children}</div>
    </section>
  );
}

function WorkOverviewRows({ rows }: { rows: Array<[string, unknown]> }) {
  const visibleRows = rows.filter(
    ([, value]) =>
      !workDetailDisplayValueEmpty(value) && displayText(value) !== "-",
  );
  if (visibleRows.length === 0) {
    return <WorkEmptyText>暂无可展示信息。</WorkEmptyText>;
  }
  return (
    <div className="grid gap-2 text-sm">
      {visibleRows.map(([label, value]) => (
        <WorkDetailMetaRow
          key={label}
          label={label}
          value={value}
        />
      ))}
    </div>
  );
}

function WorkOverviewCompletenessList({
  items,
}: {
  items: WorkDataCompletenessTemplate[];
}) {
  if (items.length === 0) {
    return <WorkEmptyText>暂无完整度数据。</WorkEmptyText>;
  }
  return (
    <div className="grid gap-3">
      {items.slice(0, 5).map((item, index) => {
        const percent = workOverviewCompletenessPercent(item);
        return (
          <div
            key={`${
              textValue(item.template_id || item.name) || "template"
            }-${index}`}
            className="grid gap-1.5"
          >
            <div className="flex items-center justify-between gap-3 text-sm">
              <span className="truncate font-medium">
                {displayText(item.template_name || item.name, "资料")}
              </span>
              <span className="shrink-0 text-xs text-muted-foreground">
                {percent}
              </span>
            </div>
            <div className="h-1.5 overflow-hidden rounded-full bg-background">
              <div
                className="h-full rounded-full bg-foreground"
                style={{ width: percent }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

function WorkOverviewRecentOperations({
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
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        正在加载
      </div>
    );
  }
  if (operations.length === 0) {
    return <WorkEmptyText>暂无流程记录。</WorkEmptyText>;
  }
  return (
    <div className="grid gap-2">
      {operations.slice(0, 5).map((operation, index) => (
        <button
          type="button"
          key={workOperationTimelineKey(operation, index)}
          className="min-w-0 rounded-md bg-background px-3 py-2 text-left transition-colors hover:bg-muted/50"
          onClick={() => openWorkRecordDetail(operation, store)}
        >
          <div className="truncate text-sm font-medium">
            {workOperationTitle(operation)}
          </div>
          <div className="mt-0.5 flex flex-wrap gap-x-2 gap-y-1 text-xs leading-5 text-muted-foreground">
            <span>
              {formatWorkDate(operation.created_at || operation.create_time)}
            </span>
            <span>
              {displayText(
                operation.operator_name || operation["operator_staff.name"],
              )}
            </span>
          </div>
        </button>
      ))}
    </div>
  );
}

function workOverviewTaskNames(target?: WorkCustomer | WorkAsset): string[] {
  const tasks = Array.isArray(target?.row_tasks)
    ? target?.row_tasks
    : Array.isArray(target?.tasks)
      ? target?.tasks
      : [];
  return tasks.map(workTaskButtonLabel).filter(Boolean);
}

function workOverviewTaskSummary(taskNames: string[]): string {
  if (taskNames.length === 0) return "暂无待办";
  if (taskNames.length === 1) return taskNames[0] || "1 个待办";
  return `${taskNames.length} 个待办`;
}

function workOverviewCompletenessItems(
  ...targets: Array<WorkCustomer | WorkAsset | WorkBusinessObject | null | undefined>
): WorkDataCompletenessTemplate[] {
  return targets.flatMap((target) =>
    Array.isArray(target?.data_completeness) ? target.data_completeness : [],
  );
}

function workOverviewAverageCompleteness(
  items: WorkDataCompletenessTemplate[],
): string {
  if (items.length === 0) return "-";
  const totals = items.reduce(
    (summary, item) => {
      const total = Number(item.total);
      const filled = Number(item.filled);
      if (
        Number.isFinite(total) &&
        total > 0 &&
        Number.isFinite(filled)
      ) {
        summary.total += total;
        summary.filled += Math.max(0, Math.min(filled, total));
      }
      return summary;
    },
    { filled: 0, total: 0 },
  );
  if (totals.total > 0) {
    return `${Math.round((totals.filled / totals.total) * 100)}%`;
  }
  const total = items.reduce(
    (sum, item) => sum + workOverviewPercentNumber(item),
    0,
  );
  return `${Math.round(total / items.length)}%`;
}

function workOverviewCompletenessPercent(
  item: WorkDataCompletenessTemplate,
): string {
  return `${workOverviewPercentNumber(item)}%`;
}

function workOverviewPercentNumber(item: WorkDataCompletenessTemplate): number {
  const configured = Number(item.percent);
  if (Number.isFinite(configured)) {
    return Math.max(0, Math.min(100, Math.round(configured)));
  }
  const total = Number(item.total);
  const filled = Number(item.filled);
  if (Number.isFinite(total) && total > 0 && Number.isFinite(filled)) {
    return Math.max(0, Math.min(100, Math.round((filled / total) * 100)));
  }
  return 0;
}

function workOverviewLatestOperations(
  operations: WorkOperation[],
): WorkOperation[] {
  return [...operations].sort(
    (left, right) =>
      workOperationTimeValue(right) - workOperationTimeValue(left),
  );
}

function workOverviewPrimaryAsset(
  customer?: WorkCustomer,
): WorkAsset | undefined {
  return workOverviewCustomerAssets(customer)[0];
}

function workOverviewCustomerAssets(customer?: WorkCustomer): WorkAsset[] {
  return Array.isArray(customer?.assets) ? customer.assets : [];
}

function workOverviewPrimaryBusinessObject(
  asset?: WorkAsset,
): WorkBusinessObject | null {
  const objects = Array.isArray(asset?.business_objects)
    ? asset.business_objects
    : [];
  return objects[0] || null;
}

function workOverviewFieldValue(
  fields: WorkDisplayField[] | undefined,
  labels: string | string[],
): string {
  const candidates = (Array.isArray(labels) ? labels : [labels])
    .map(workOverviewComparableLabel)
    .filter(Boolean);
  const rows = (fields || []).filter(
    (item) =>
      textValue(item.label) && !workDisplayFieldEmpty(item),
  );
  const exactField = rows.find((item) =>
    candidates.includes(workOverviewComparableLabel(item.label)),
  );
  if (exactField) return workDetailDisplayValue(exactField);

  const field = rows.find((item) => {
    const itemLabel = workOverviewComparableLabel(item.label);
    return candidates.some((candidate) => {
      if (candidate.length <= 2) return itemLabel === candidate;
      return itemLabel.includes(candidate);
    });
  });
  return field ? workDetailDisplayValue(field) : "-";
}

function workOverviewComparableLabel(label: unknown): string {
  return textValue(label).replace(/\s+/g, "");
}

function workOverviewFinanceSummary(
  fields: WorkDisplayField[] | undefined,
): string {
  const financeLabels = [
    "服务费",
    "律师费",
    "保证金",
    "成本",
    "费用",
    "投流",
    "运营",
    "租金",
    "收款",
    "付款",
  ];
  const values = (fields || [])
    .filter((field) => {
      if (workDisplayFieldEmpty(field)) return false;
      const fieldLabel = workOverviewComparableLabel(field.label);
      return financeLabels.some((label) =>
        fieldLabel.includes(workOverviewComparableLabel(label)),
      );
    })
    .map(
      (field) => `${displayText(field.label)}：${workDetailDisplayValue(field)}`,
    )
    .slice(0, 3);
  return values.length > 0 ? values.join("；") : "-";
}

function WorkDetailPanelSection({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="grid gap-3">
      <h3 className="text-[15px] font-semibold leading-6">{title}</h3>
      {children}
    </section>
  );
}

function WorkContactPanelCard({ customer }: { customer: WorkCustomer }) {
  const name = workCustomerName(customer);
  return (
    <div className="rounded-lg border border-border/60 bg-background p-4 shadow-sm">
      <div className="flex min-w-0 flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate text-base font-semibold leading-6">
            {name}
          </div>
          <div className="mt-0.5 text-xs leading-5 text-muted-foreground">
            {workCustomerNo(customer)}
          </div>
        </div>
        <div className="shrink-0">
          {renderStatus(customer)}
        </div>
      </div>
      <div className="mt-4 grid gap-2 text-sm">
        <WorkDetailMetaRow label="手机" value={workCustomerPhone(customer)} />
        <WorkDetailMetaRow
          label="微信"
          value={displayText(customer.wechat)}
        />
        <WorkDetailMetaRow
          label="来源"
          value={displayText(customer.source_name || customer.source)}
        />
        <WorkDetailMetaRow
          label="渠道"
          value={displayText(customer.channel_name || customer.channel)}
        />
        <WorkDetailMetaRow
          label="等级"
          value={displayText(customer.level_name || customer.customer_level)}
        />
        <WorkDisplayFieldRows
          fields={customer.display_fields}
          excludeLabels={["手机", "手机号", "微信", "来源", "渠道", "等级"]}
        />
      </div>
    </div>
  );
}

function WorkAssetPanelCard({ asset }: { asset?: WorkAsset }) {
  if (!asset) {
    return (
      <div className="rounded-lg border border-dashed border-border/70 bg-muted/10 p-4 text-sm leading-6 text-muted-foreground">
        资产资料尚未录入，后续阶段任务会补充到该客户记录。
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-border/60 bg-background p-4 shadow-sm">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="break-words text-base font-semibold leading-6">
            {assetTitle(asset)}
          </div>
          <div className="mt-0.5 break-all text-xs leading-5 text-muted-foreground">
            {workAssetNo(asset)}
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {renderStatus(asset)}
          {renderAssetStatus(asset)}
        </div>
      </div>
      <div className="mt-4 grid gap-2 text-sm">
        <WorkDetailMetaRow
          label="资产状态"
          value={displayText(asset.asset_status_name)}
        />
        <WorkDetailMetaRow label="备注" value={displayText(asset.remark)} />
        <WorkDisplayFieldRows
          fields={asset.display_fields}
          excludeLabels={["资产状态", "备注"]}
        />
      </div>
    </div>
  );
}

function WorkBusinessObjectPanelList({ assets }: { assets: WorkAsset[] }) {
  const objects = assets.flatMap((asset) =>
    Array.isArray(asset.business_objects) ? asset.business_objects : [],
  );
  if (objects.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border/70 bg-muted/10 p-4 text-sm leading-6 text-muted-foreground">
        暂无租赁记录。完成租赁记录创建任务后，会在这里看到租户、租期、收款和成本资料。
      </div>
    );
  }
  return (
    <div className="grid gap-3">
      {objects.map((object, index) => (
        <WorkBusinessObjectPanelCard
          key={workBusinessObjectID(object) || `business-object-${index}`}
          object={object}
        />
      ))}
    </div>
  );
}

function WorkBusinessObjectPanelCard({
  object,
}: {
  object: WorkBusinessObject;
}) {
  const fields = Array.isArray(object.display_fields)
    ? object.display_fields.filter(
        (field) =>
          textValue(field.label) &&
          !workDisplayFieldEmpty(field),
      )
    : [];
  return (
    <div className="rounded-lg border border-border/60 bg-background p-4 shadow-sm">
      <div className="flex min-w-0 flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="break-words text-base font-semibold leading-6">
            {workBusinessObjectTitle(object)}
          </div>
          <div className="mt-0.5 break-all text-xs leading-5 text-muted-foreground">
            {[textValue(object.business_object_type_name), textValue(object.object_no)]
              .filter(Boolean)
              .join(" / ") || "-"}
          </div>
        </div>
        {object.object_status ? (
          <span className="rounded-full bg-muted px-2.5 py-1 text-xs font-medium text-muted-foreground">
            {displayText(object.object_status)}
          </span>
        ) : null}
      </div>
      {fields.length > 0 ? (
        <div className="mt-4 grid gap-2 text-sm">
          {fields.map((field) => (
            <WorkDetailMetaRow
              key={textValue(field.key) || textValue(field.label)}
              label={displayText(field.label)}
              value={workDetailDisplayValue(field)}
            />
          ))}
        </div>
      ) : (
        <div className="mt-3 text-sm leading-6 text-muted-foreground">
          已创建记录，暂无可展示资料。
        </div>
      )}
    </div>
  );
}

function WorkDisplayFieldRows({
  fields,
  excludeLabels = [],
}: {
  fields?: WorkDisplayField[];
  excludeLabels?: string[];
}) {
  const excluded = new Set(excludeLabels.map(workOverviewComparableLabel));
  const rows = Array.isArray(fields)
    ? fields.filter(
        (field) =>
          textValue(field.label) &&
          !excluded.has(workOverviewComparableLabel(field.label)) &&
          !workDisplayFieldEmpty(field),
      )
    : [];
  if (rows.length === 0) return null;
  return (
    <>
      {rows.map((field) => (
        <WorkDetailMetaRow
          key={textValue(field.key) || textValue(field.label)}
          label={displayText(field.label)}
          value={workDetailDisplayValue(field)}
        />
      ))}
    </>
  );
}

function workDetailDisplayValue(field: WorkDisplayField): string {
  if (textValue(field.value_type) === "files") {
    const files = normalizeUploadItems(field.files);
    if (files.length > 0) {
      return `${files.length} 个附件`;
    }
  }
  return displayText(field.value);
}

function workDisplayFieldEmpty(field: WorkDisplayField): boolean {
  if (textValue(field.value_type) === "files") {
    return normalizeUploadItems(field.files).length === 0;
  }
  return workDetailDisplayValueEmpty(field.value);
}

function workDetailDisplayValueEmpty(value: unknown): boolean {
  if (value === null || value === undefined || value === "") return true;
  if (Array.isArray(value)) return value.length === 0;
  return false;
}

function WorkDetailMetaRow({
  label,
  value,
}: {
  label: string;
  value: unknown;
}) {
  return (
    <div className="flex min-w-0 items-start gap-3">
      <span className="w-16 shrink-0 text-muted-foreground">{label}</span>
      <span className="min-w-0 flex-1 break-words text-foreground">
        {displayText(value)}
      </span>
    </div>
  );
}

function WorkDetailTabs({
  activeTab,
  onChange,
}: {
  activeTab: WorkDetailTab;
  onChange: (tab: WorkDetailTab) => void;
}) {
  const tabs: Array<{ key: WorkDetailTab; label: string }> = [
    { key: "overview", label: "总览" },
    { key: "info", label: "资料" },
    { key: "flow", label: "流程" },
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
  const sortedOperations = [...filteredOperations].sort(
    (left, right) =>
      workOperationTimeValue(right) - workOperationTimeValue(left),
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        正在加载流程记录
      </div>
    );
  }

  return (
    <div className="grid gap-4">
      <WorkOperationScopeTabs scope={scope} onScopeChange={onScopeChange} />

      {filteredOperations.length === 0 ? (
        <WorkEmptyText>
          {scope === "mine" ? "暂无我的流程记录" : "暂无流程记录"}
        </WorkEmptyText>
      ) : (
        <div className="relative grid gap-3 md:gap-0">
          <span className="pointer-events-none absolute bottom-0 left-[5px] top-0 w-px bg-border/60 md:left-1/2 md:-translate-x-px" />
          {sortedOperations.map((operation, index) => (
            <WorkOperationCard
              key={workOperationTimelineKey(operation, index)}
              operation={operation}
              side={index % 2 === 0 ? "left" : "right"}
              onOpen={() => openWorkRecordDetail(operation, store)}
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
  side,
  onOpen,
}: {
  operation: WorkOperation;
  side: "left" | "right";
  onOpen: () => void;
}) {
  const content = workOperationDescription(operation);
  const tone = workOperationTone(operation);
  const stageLabel = workOperationStageLabel(operation);
  const card = (
    <button
      type="button"
      className="block w-full rounded-lg bg-muted/20 px-4 py-3 text-left transition-colors hover:bg-muted/35"
      onClick={onOpen}
    >
      <div className="flex min-w-0 items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <span className="min-w-0 break-words text-sm font-semibold leading-6 text-foreground">
              {workOperationTitle(operation)}
            </span>
            <span
              className={`rounded-full px-2 py-0.5 text-[11px] font-medium leading-5 ${tone.badge}`}
            >
              {workOperationBadgeText(operation)}
            </span>
          </div>
          <div className="mt-0.5 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 text-xs leading-5 text-muted-foreground/70">
            {stageLabel ? <span>{stageLabel}</span> : null}
            <span>
              操作人：
              {displayText(
                operation.operator_name || operation["operator_staff.name"],
              )}
            </span>
          </div>
        </div>
        <div className="shrink-0 whitespace-nowrap text-xs leading-6 text-muted-foreground/70">
          {formatWorkDate(operation.created_at || operation.create_time)}
        </div>
      </div>
      {content ? (
        <p className="mt-2 text-sm leading-6 text-muted-foreground/80">
          {content}
        </p>
      ) : null}
    </button>
  );

  return (
    <article className="relative pl-7 md:grid md:grid-cols-[minmax(0,1fr)_2.5rem_minmax(0,1fr)] md:items-start md:pl-0">
      <span
        className={`absolute left-0 top-4 z-10 h-3 w-3 rounded-full border-2 border-background md:left-1/2 md:-translate-x-1/2 ${tone.dot}`}
      />
      <div
        className={
          side === "left"
            ? "md:col-start-1 md:pr-5"
            : "md:col-start-3 md:pl-5"
        }
      >
        {card}
      </div>
    </article>
  );
}

function workOperationTimelineKey(
  operation: WorkOperation,
  index: number,
): string {
  return (
    textValue(operation.id) ||
    `${textValue(operation.created_at || operation.create_time)}-${index}`
  );
}

function workOperationTimeValue(operation: WorkOperation): number {
  const raw = textValue(operation.created_at || operation.create_time);
  if (!raw) return 0;
  const value = Date.parse(raw);
  return Number.isFinite(value) ? value : 0;
}

function workOperationStageLabel(operation: WorkOperation): string {
  return textValue(operation.stage_name) || textValue(operation.stage_code);
}

function workOperationBadgeText(operation: WorkOperation): string {
  const resultValue = textValue(operation.result_value);
  if (resultValue === "progress") return "进度";
  const taskType = textValue(operation.task_type || operation["task.task_type"]);
  switch (taskType) {
    case "todo":
      return "事项";
    case "form":
      return "资料";
    case "approval":
      return "审核";
    case "rule":
      return "核验";
    default:
      return taskType ? "任务" : "流程";
  }
}

function workOperationResultName(operation: WorkOperation): string {
  const resultName = textValue(
    operation.result_value_name || operation.result_value_display,
  );
  if (resultName) return resultName;

  switch (textValue(operation.result_value)) {
    case "progress":
      return "保存进度";
    case "completed":
      return "已完成";
    case "submitted":
      return "已提交";
    case "approved":
      return "审核通过";
    case "passed":
      return "核验通过";
    case "failed":
      return "核验未通过";
    case "canceled":
      return "已取消";
    case "rejected":
      return "审核驳回";
    case "entered":
      return "进入阶段";
    default:
      return displayText(operation.result_value, "记录");
  }
}

function workOperationTone(operation: WorkOperation): {
  badge: string;
  border: string;
  dot: string;
} {
  const resultValue = textValue(operation.result_value);
  if (resultValue === "rejected" || resultValue === "failed") {
    return {
      badge: "bg-red-50 text-red-700",
      border: "border-red-200/80",
      dot: "bg-red-500",
    };
  }
  if (resultValue === "approved" || resultValue === "passed") {
    return {
      badge: "bg-emerald-50 text-emerald-700",
      border: "border-emerald-200/80",
      dot: "bg-emerald-500",
    };
  }
  if (resultValue === "progress") {
    return {
      badge: "bg-amber-50 text-amber-700",
      border: "border-amber-200/80",
      dot: "bg-amber-500",
    };
  }
  const taskType = textValue(operation.task_type || operation["task.task_type"]);
  switch (taskType) {
    case "todo":
      return {
        badge: "bg-muted text-foreground",
        border: "border-border/60",
        dot: "bg-foreground/60",
      };
    case "form":
      return {
        badge: "bg-sky-50 text-sky-700",
        border: "border-sky-200/80",
        dot: "bg-sky-500",
      };
    case "approval":
      return {
        badge: "bg-amber-50 text-amber-700",
        border: "border-amber-200/80",
        dot: "bg-amber-500",
      };
    case "rule":
      return {
        badge: "bg-teal-50 text-teal-700",
        border: "border-teal-200/80",
        dot: "bg-teal-500",
      };
    default:
      return {
        badge: "bg-muted text-muted-foreground",
        border: "border-border/50",
        dot: "bg-muted-foreground/40",
      };
  }
}

function WorkCustomerMainInfo({
  customer,
  asset,
}: {
  customer: WorkCustomer;
  asset?: WorkAsset;
}) {
  const assets = asset ? [asset] : workOverviewCustomerAssets(customer);
  return (
    <div className="grid gap-5">
      <WorkDetailPanelSection title="联系人">
        <WorkContactPanelCard customer={customer} />
      </WorkDetailPanelSection>
      <WorkDetailPanelSection title="资产信息">
        {assets.length > 0 ? (
          <div className="grid gap-3">
            {assets.map((asset, index) => (
              <WorkAssetPanelCard
                key={workAssetID(asset) || workAssetNo(asset) || `asset-${index}`}
                asset={asset}
              />
            ))}
          </div>
        ) : (
          <WorkAssetPanelCard />
        )}
      </WorkDetailPanelSection>
      {assets.length > 0 ? (
        <WorkDetailPanelSection title="租赁记录">
          <WorkBusinessObjectPanelList assets={assets} />
        </WorkDetailPanelSection>
      ) : null}
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
  const [aiPromptOpen, setAiPromptOpen] = useState(false);
  const [aiInstruction, setAiInstruction] = useState("");
  const contentRef = useRef<HTMLDivElement | null>(null);
  const canSaveProgress = task ? workTaskAllowsProgress(task) : false;
  const canCompleteDirectly = task ? workTaskNeedsCompleteAction(task) : false;
  const canAIFill = task ? workTaskCanAIFill(task) : false;
  const footerTargets = useWorkTaskModalFooterTargets(
    contentRef,
    canAIFill || canCompleteDirectly,
    canCompleteDirectly,
  );

  const close = useCallback(() => {
    setWorkModalOpen(store, "dialog.workTask", false);
  }, [store]);

  const submit = useCallback(
    async (mode: "complete" | "progress" = "complete") => {
      if (!task || submitting) return false;
      clearCurrentWorkTaskFormErrors(store);
      if (
        !validateCurrentWorkTaskForm(store, {
          allowMissingRequired: mode === "progress",
        })
      ) {
        return false;
      }
      if (!confirmWorkTaskSubmit(task, mode)) return false;

      setSubmitting(true);
      try {
        await workApi("/crm/work/execute", {
          method: "POST",
          body: JSON.stringify({
            task_id: task.id,
            todo_id: positiveTextID(task.todo_id) || undefined,
            customer_id: workCustomerID(customer),
            asset_id: workAssetID(asset),
            submit_mode: mode,
            values: {
              ...collectWorkTaskSubmitValues(store),
              submit_mode: mode,
            },
          }),
        });
        toast.success(workTaskSubmitSuccessMessage(task, mode));
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
    },
    [asset, close, customer, store, submitting, task],
  );

  const aiFill = useCallback(async (instruction = "") => {
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
          instruction: textValue(instruction) || undefined,
          values: collectWorkTaskSubmitValues(store),
        }),
      });
      const count = applyWorkTaskAIFillValues(store, payload.values || {});
      if (count === 0) {
        toast.info("AI 没有返回可填写的字段");
        return;
      }
      toast.success(`AI 已填写 ${count} 个字段`);
      setAiPromptOpen(false);
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
  const aiFillControl = canAIFill ? (
    <div className="relative">
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => setAiPromptOpen((open) => !open)}
        disabled={aiFilling || submitting}
      >
        {aiFilling ? "AI填写中" : "AI填写"}
      </Button>
      {aiPromptOpen ? (
        <div className="absolute bottom-11 right-0 z-50 w-80 rounded-md border bg-background p-3 shadow-lg">
          <textarea
            className="min-h-20 w-full resize-none rounded-md border border-input bg-background px-3 py-2 text-sm outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
            value={aiInstruction}
            onChange={(event) => setAiInstruction(event.target.value)}
            placeholder="输入要让 AI 帮你提取或补全的信息..."
          />
          <div className="mt-2 flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setAiPromptOpen(false)}
              disabled={aiFilling}
            >
              取消
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={() => void aiFill(aiInstruction)}
              disabled={aiFilling || submitting}
            >
              {aiFilling ? "填写中" : "填写"}
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  ) : null;
  const manualActionButtons = canSaveProgress ? (
    <>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => void submit("progress")}
        disabled={submitting || aiFilling}
      >
        保存进度
      </Button>
      <Button
        type="button"
        size="sm"
        onClick={() => void submit("complete")}
        disabled={submitting || aiFilling}
      >
        确认完成
      </Button>
    </>
  ) : canCompleteDirectly ? (
    <Button
      type="button"
      size="sm"
      onClick={() => void submit("complete")}
      disabled={submitting || aiFilling}
    >
      {workTaskIsApproval(task) ? "提交审核" : "完成任务"}
    </Button>
  ) : null;

  return (
    <div ref={contentRef} className="contents">
      {footerTargets?.actions
        ? createPortal(
            <>
              {aiFillControl}
              {manualActionButtons}
            </>,
            footerTargets.actions,
          )
        : null}
      {(canAIFill || canCompleteDirectly) && !footerTargets ? (
        <div className="mt-4 flex items-center justify-between gap-3 border-t pt-4">
          <div />
          <div className="flex items-center gap-2">
            {aiFillControl}
            {manualActionButtons}
          </div>
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

export function ShowCrmWorkTaskGroupTabs({ item, store }: WorkNodeProps) {
  const rawTabs = item?.meta?.["tabs"];
  const tabs = useMemo(() => normalizeWorkTaskGroupTabs(rawTabs), [rawTabs]);
  const [activeTabID, setActiveTabID] = useState(tabs[0]?.id || "");

  useEffect(() => {
    if (tabs.length === 0) return;
    if (!tabs.some((tab) => tab.id === activeTabID)) {
      setActiveTabID(tabs[0].id);
    }
  }, [activeTabID, tabs]);

  if (tabs.length === 0) return null;

  const activeTab = tabs.find((tab) => tab.id === activeTabID) || tabs[0];

  return (
    <div className="space-y-4 rounded-md border border-border/70 bg-background p-3">
      <div className="flex flex-wrap gap-2 border-b border-border/70 pb-3">
        {tabs.map((tab) => {
          const active = tab.id === activeTab.id;
          return (
            <button
              key={tab.id}
              type="button"
              className={`rounded-md px-3 py-1.5 text-sm transition ${
                active
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground hover:bg-muted/80"
              }`}
              onClick={() => setActiveTabID(tab.id)}
            >
              {tab.label}
            </button>
          );
        })}
      </div>
      <div className="space-y-3">
        {activeTab.fields.map((field) => (
          <WorkTaskGroupFieldControl
            key={field.formKey}
            field={field}
            store={store}
          />
        ))}
      </div>
    </div>
  );
}

export function ShowCrmWorkTaskFieldSection({ item, store }: WorkNodeProps) {
  const section = useMemo(
    () => normalizeWorkTaskFieldSection(item?.meta),
    [item?.meta],
  );

  if (!section || section.fields.length === 0) return null;

  return (
    <section className="space-y-4 border-t border-border/70 pt-5 first:border-t-0 first:pt-0">
      <div>
        <h3 className="text-base font-semibold text-foreground">
          {section.label}
        </h3>
      </div>
      <div className="space-y-3">
        {section.fields.map((field) => (
          <WorkTaskGroupFieldControl
            key={field.formKey}
            field={field}
            store={store}
          />
        ))}
      </div>
    </section>
  );
}

function WorkTaskGroupFieldControl({
  field,
  store,
}: {
  field: WorkTaskGroupField;
  store?: StoreLike;
}) {
  const value = workStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    {},
  )[field.formKey];
  const error = workStoreValue<Record<string, string>>(store, "errors", {})[
    `workTaskForm.${field.formKey}`
  ];
  const setValue = useCallback(
    (nextValue: unknown) => {
      const current = workStoreValue<Record<string, unknown>>(
        store,
        workTaskFormDataPath,
        {},
      );
      setWorkStoreValue(store, workTaskFormDataPath, {
        ...current,
        [field.formKey]: nextValue,
      });
    },
    [field.formKey, store],
  );

  return (
    <label className="grid gap-1.5 text-sm md:grid-cols-[9rem_minmax(0,1fr)] md:items-start">
      <span className="pt-2 font-medium text-foreground">
        {field.label}
        {field.required ? <span className="text-destructive"> *</span> : null}
      </span>
      <span className="space-y-1">
        <WorkTaskGroupInput
          field={field}
          value={value}
          setValue={setValue}
          store={store}
          error={error}
        />
        {error ? (
          <span className="block text-xs text-destructive">{error}</span>
        ) : null}
      </span>
    </label>
  );
}

function WorkTaskGroupInput({
  field,
  value,
  setValue,
  store,
  error,
}: {
  field: WorkTaskGroupField;
  value: unknown;
  setValue: (value: unknown) => void;
  store?: StoreLike;
  error?: string;
}) {
  if (field.type === "show-crm-work-task-upload") {
    return (
      <ShowCrmWorkTaskUpload
        item={{
          id: `group-upload-${field.formKey}`,
          name: field.label,
          value: `workTaskForm.${field.formKey}`,
          placeholder: field.placeholder,
          meta: field.meta,
        }}
        store={store}
        value={value}
        setValue={setValue}
        error={error}
      />
    );
  }

  if (field.type === "form-select") {
    const multiple = Boolean(field.meta?.["multiple"]);
    const selectedValues = Array.isArray(value)
      ? value.map(textValue)
      : textValue(value)
        ? [textValue(value)]
        : [];
    return (
      <select
        className={inputClassName}
        value={multiple ? selectedValues : selectedValues[0] || ""}
        multiple={multiple}
        onChange={(event) => {
          if (multiple) {
            setValue(
              Array.from(event.currentTarget.selectedOptions).map(
                (option) => option.value,
              ),
            );
            return;
          }
          setValue(event.currentTarget.value);
        }}
      >
        {!multiple ? <option value="">{field.placeholder}</option> : null}
        {(field.options || []).map((option) => (
          <option key={textValue(option.id)} value={textValue(option.id)}>
            {displayText(option.value || option.name || option.id)}
          </option>
        ))}
      </select>
    );
  }

  if (field.type === "form-textarea") {
    return (
      <textarea
        className={inputClassName}
        rows={4}
        placeholder={field.placeholder}
        value={formatFormValue(value)}
        onChange={(event) => setValue(event.currentTarget.value)}
      />
    );
  }

  return (
    <Input
      placeholder={field.placeholder}
      value={formatFormValue(value)}
      onChange={(event) => setValue(event.currentTarget.value)}
    />
  );
}

function normalizeWorkTaskGroupTabs(value: unknown): WorkTaskGroupTab[] {
  if (!Array.isArray(value)) return [];
  return value
    .filter(workIsRecord)
    .map((tab) => ({
      id: textValue(tab["id"]) || workTaskFormKey(textValue(tab["label"])),
      label: displayText(tab["label"]),
      fields: normalizeWorkTaskGroupFields(tab["fields"]),
    }))
    .filter((tab) => tab.id && tab.label && tab.fields.length > 0);
}

function normalizeWorkTaskFieldSection(
  value: unknown,
): WorkTaskFieldSection | null {
  if (!workIsRecord(value)) return null;
  const label = displayText(value["title"] || value["label"] || value["name"]);
  const fields = normalizeWorkTaskGroupFields(value["fields"]);
  if (!label || fields.length === 0) return null;
  return {
    id: textValue(value["id"]) || workTaskFormKey(label),
    label,
    fields,
  };
}

function normalizeWorkTaskGroupFields(value: unknown): WorkTaskGroupField[] {
  if (!Array.isArray(value)) return [];
  return value
    .filter(workIsRecord)
    .map((field) => ({
      formKey: textValue(field["formKey"]),
      label: displayText(field["label"]),
      placeholder: textValue(field["placeholder"]),
      required: Boolean(field["required"]),
      type: textValue(field["type"]) || "form-input",
      options: normalizeWorkCommonOptions(field["options"]),
      meta: workIsRecord(field["meta"]) ? field["meta"] : undefined,
    }))
    .filter((field) => field.formKey && field.label);
}

type WorkTaskModalFooterTargets = {
  left: HTMLElement;
  actions: HTMLElement;
};

function useWorkTaskModalFooterTargets(
  contentRef: RefObject<HTMLElement | null>,
  enabled: boolean,
  replaceSubmit: boolean,
) {
  const [targets, setTargets] = useState<WorkTaskModalFooterTargets | null>(
    null,
  );

  useEffect(() => {
    if (!enabled) {
      setTargets(null);
      return undefined;
    }

    const content = contentRef.current;
    if (!content) {
      setTargets(null);
      return undefined;
    }

    const form = content.closest("form");
    const footer = findWorkTaskModalFooter(form);
    const submitButton =
      footer?.querySelector<HTMLButtonElement>('button[type="submit"]') ||
      null;
    if (!footer) {
      setTargets(null);
      return undefined;
    }

    const left = document.createElement("div");
    left.setAttribute("data-crm-work-task-footer-left", "true");
    left.className = "mr-auto flex items-center gap-2";

    const actions = document.createElement("div");
    actions.setAttribute("data-crm-work-task-footer-actions", "true");
    actions.className = "flex items-center gap-2";

    const previousSubmitDisplay = submitButton?.style.display || "";
    footer.insertBefore(left, footer.firstChild);
    if (submitButton) {
      footer.insertBefore(actions, submitButton);
      if (replaceSubmit) {
        submitButton.style.display = "none";
      }
    } else {
      footer.appendChild(actions);
    }
    setTargets({ left, actions });

    return () => {
      left.remove();
      actions.remove();
      if (submitButton) {
        submitButton.style.display = previousSubmitDisplay;
      }
      setTargets(null);
    };
  }, [contentRef, enabled, replaceSubmit]);

  return targets;
}

function findWorkTaskModalFooter(form: Element | null): HTMLElement | null {
  if (!form) return null;
  const children = Array.from(form.children).filter(
    (child): child is HTMLElement => child instanceof HTMLElement,
  );
  for (const child of [...children].reverse()) {
    if (child.querySelector('button[type="submit"]')) {
      return child;
    }
  }
  const submitButton = form.querySelector<HTMLButtonElement>(
    'button[type="submit"]',
  );
  return submitButton?.parentElement || null;
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

function validateCurrentWorkTaskForm(
  store: StoreLike | undefined,
  options: { allowMissingRequired?: boolean } = {},
): boolean {
  if (options.allowMissingRequired) return true;
  const validateForm = currentWorkStoreState(store)?.validateForm;
  if (typeof validateForm === "function" && !validateForm()) {
    return false;
  }
  return validateCurrentWorkTaskDomainRules(store);
}

function validateCurrentWorkTaskDomainRules(
  store: StoreLike | undefined,
): boolean {
  const groupErrors = currentWorkTaskGroupFieldErrors(store);
  if (Object.keys(groupErrors).length > 0) {
    setCurrentWorkTaskFormErrors(store, groupErrors);
    return false;
  }
  return true;
}

function currentWorkTaskGroupFieldErrors(
  store: StoreLike | undefined,
): Record<string, string> {
  const nodes = currentWorkStoreState(store)?.schema?.nodes?.[
    workTaskFormSectionID
  ];
  const values = workStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    {},
  );
  const errors: Record<string, string> = {};
  if (!Array.isArray(nodes)) return errors;

  for (const node of nodes) {
    for (const field of workTaskNodeRequiredFields(node)) {
      if (!field.required) continue;
      if (!workTaskClientValueEmpty(values[field.formKey])) continue;
      errors[`workTaskForm.${field.formKey}`] = `${field.label}不能为空。`;
    }
  }
  return errors;
}

function workTaskNodeRequiredFields(
  node: WorkTaskFormNode | undefined,
): WorkTaskGroupField[] {
  if (node?.type === "show-crm-work-task-field-section") {
    return normalizeWorkTaskGroupFields(node.meta?.["fields"]);
  }
  if (node?.type === "show-crm-work-task-group-tabs") {
    return normalizeWorkTaskGroupTabs(node.meta?.["tabs"]).flatMap(
      (tab) => tab.fields,
    );
  }
  return [];
}

function workTaskClientValueEmpty(value: unknown): boolean {
  if (value === null || value === undefined || value === "") return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "object") return Object.keys(value).length === 0;
  return textValue(value) === "";
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
  if (message.includes("审核意见")) return "opinion";
  if (message.includes("审核结果")) return "approval_result";
  if (message.includes("办理结果")) return "result";
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
