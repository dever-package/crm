import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import { createPortal } from "react-dom";
import {
  AlertTriangle,
  ArrowRight,
  Check,
  ClipboardList,
  Download,
  Inbox,
  LogIn,
  Loader2,
  Plus,
  RefreshCw,
  Sparkles,
  TrendingUp,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { downloadUploadFile, type UploadFileItem } from "@/lib/upload";
import { normalizeUploadItems } from "@/lib/resource";

import {
  currentWorkStoreState,
  displayText,
  errorMessage,
  formatWorkDate,
  getRuntimeSite,
  getWorkEntryPath,
  normalizeWorkDetailSections,
  positiveTextID,
  readWorkListSearch,
  saveWorkSession,
  setWorkModalOpen,
  setWorkStoreValue,
  textValue,
  updateWorkStoreErrors,
  workApi,
  workCustomerModeConfig,
  workListSearchEvent,
  workRefreshEvent,
  workStoreValue,
  workTaskCommunicationGroupContextPath,
  workTaskCommunicationGroupDraftPath,
  workTaskCommunicationGroupErrorPath,
  workTaskFieldMapPath,
  workTaskFormFieldRequired,
  workTaskFormFieldVisible,
  workTaskFormFieldVisibilityRules,
  workTaskActiveGroupPath,
  workTaskFormDataPath,
  workTaskFormFieldsPath,
  workTaskFormKey,
  workTaskFormSectionID,
  workTaskLayoutPath,
  workTaskUploadFilesPath,
  workTaskUploadPendingPath,
  workTaskValidationErrorsPath,
  type WorkAIFillResponse,
  type WorkAsset,
  type WorkCommonOption,
  type WorkCommunicationGroup,
  type WorkCommunicationGroupType,
  type WorkCustomer,
  type WorkCustomerMode,
  type WorkCustomerScope,
  type WorkDetailSection,
  type WorkFieldOption,
  type WorkFormField,
  type WorkFlowDetail,
  type WorkItem,
  type WorkNodeProps,
  type WorkOperation,
  type WorkOperationSummaryItem,
  type WorkPageStoreState,
  type WorkStoreLike,
  type WorkSummary,
  type WorkSummaryBreakdown,
  type WorkSummaryMetric,
  type WorkSummaryTrendPoint,
  type WorkTask,
  type WorkTaskCommunicationGroupContext,
  type WorkTaskFieldRenderConfig,
  type WorkTaskFormField,
  type WorkTaskFormFieldVisibilityRule,
  type WorkTaskFormGroup,
  type WorkTaskFormNode,
  type WorkTaskFormState,
} from "./work-core";
import { communicationGroupDraft } from "./work-communication-group-form";
import {
  WorkCustomerListView,
  type WorkCustomerListRowView,
  type WorkCustomerListTaskView,
} from "./work-customer-list";
import {
  emptyWorkFlowModeCounts as emptyWorkCustomerModeCounts,
  normalizeWorkFlowModeCounts as normalizeWorkCustomerModeCounts,
  type WorkFlowModeCounts as WorkCustomerModeCounts,
} from "./work-flow-mode-tabs";
import {
  WorkCustomerFlowTimeline,
  type WorkCustomerFlowCurrentState,
  type WorkCustomerFlowEntryView,
  type WorkCustomerFlowTimelineVariant,
} from "./work-customer-detail";
import {
  WorkCustomerDetailWorkspace,
  normalizeWorkCustomerDetailAttachments,
  type WorkCustomerDetailAttachment,
  type WorkCustomerDetailWorkspaceSummary,
} from "./work-customer-detail-workspace";
import { WorkFlowOwnerDialog } from "./work-flow-owner-dialog";
import {
  useWorkFeedbackModalFooterTargets,
  useWorkFeedbackModalHeaderTarget,
} from "./work-feedback-modal";
import {
  focusFirstWorkTaskFormError,
  workTaskFormValueEmpty,
  workTaskLayoutMode,
  workTaskNodeFormFields,
} from "./work-task-form";
import {
  applyWorkTaskCommunicationGroupError,
  clearWorkTaskCommunicationGroupError,
  collectWorkTaskCommunicationGroup,
  validateWorkTaskCommunicationGroup,
} from "./work-task-communication-group";
import {
  emptyWorkTaskRecord,
  useWorkTaskStoreValue,
} from "./work-task-form-fields";
import {
  buildFeishuOAuthURL,
  getFeishuAuthCode,
  isFeishuClient,
  loadFeishuSDK,
} from "./feishu-login";
import { WorkTaskUploadPreviewDialog } from "./work-upload";
import {
  CrmEChart,
  crmChartAxisColor,
  crmChartSplitLineColor,
  crmChartTextColor,
  type EChartsOption,
} from "./crm-echarts";

export { ShowCrmWorkTaskUpload } from "./work-upload";
export { ShowCrmWorkTaskGroupTabs } from "./work-task-form";

type StoreLike = WorkStoreLike;

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

function workTaskIsProduct(task: WorkTask): boolean {
  return workTaskAction(task) === "product";
}

function workTaskAllowsProgress(task: WorkTask): boolean {
  return workTaskIsForm(task);
}

function workTaskNeedsCompleteAction(task: WorkTask): boolean {
  return !workTaskIsRule(task);
}

function workTaskFlagEnabled(value: unknown): boolean {
  return value === true || value === 1 || value === "1" || value === "true";
}

function workTaskHasMeeting(task: WorkTask): boolean {
  return workTaskFlagEnabled(task.meeting_enabled);
}

function workTaskRequiresArrivalConfirmation(task: WorkTask): boolean {
  return workTaskHasMeeting(task);
}

function workTaskRejectSkipsConfiguredForm(
  task: WorkTask,
  decision: string,
): boolean {
  return (
    workTaskIsApproval(task) &&
    (decision === "rejected" || decision === "reject") &&
    !workTaskFlagEnabled(task.reject_submit_form)
  );
}

function workTaskShouldRenderFields(task: WorkTask): boolean {
  return (
    (workTaskIsForm(task) || workTaskIsApproval(task)) &&
    (task.form?.fields || []).length > 0
  );
}

function workTaskButtonLabel(task: WorkTask): string {
  const name = workTaskName(task);
  if (name && name !== "任务") return name;
  if (workTaskIsForm(task)) return "填写资料";
  if (workTaskIsApproval(task)) return "审核";
  if (workTaskIsRule(task)) return "自动核验";
  if (workTaskIsProduct(task)) return "确认产品";
  return "办理事项";
}

function workTaskSubmitSuccessMessage(
  task: WorkTask,
  mode: "complete" | "progress",
): string {
  if (workTaskRequiresArrivalConfirmation(task)) {
    return mode === "progress" ? "预约已保存" : "客户到访已确认";
  }
  if (mode === "progress") return "进度已保存";
  if (workTaskIsApproval(task)) return "审核结果已提交";
  if (workTaskIsProduct(task)) return "产品已确认";
  return workTaskIsForm(task) ? "资料已提交" : "任务已完成";
}

function workTaskKey(task: WorkTask): string {
  const todoID = positiveTextID(task.todo_id);
  const taskID = positiveTextID(task.id);
  return todoID
    ? `${taskID || "task"}:todo:${todoID}`
    : taskID || workTaskName(task);
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

function workCustomerQuery(
  keyword: string,
  mode: WorkCustomerMode,
  page: number,
  pageSize: number,
  workFilters: {
    workflowID: string;
    quickFilter: WorkQuickFilter;
    stageFilter: string;
    taskFilter: string;
    scope: WorkCustomerScope;
  },
): string {
  const params = new URLSearchParams();
  const searchKeyword = textValue(keyword);
  if (searchKeyword) params.set("keyword", searchKeyword);
  params.set("mode", mode);
  params.set("page", String(page));
  params.set("page_size", String(pageSize));
  params.set("scope", workFilters.scope);
  if (workFilters.workflowID) {
    params.set("workflow_id", workFilters.workflowID);
  }
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
  if (
    urlMode === "all" ||
    urlMode === "done" ||
    urlMode === "pending" ||
    urlMode === "processed"
  ) {
    return urlMode;
  }
  const configured = textValue(item?.meta?.mode || item?.meta?.customerMode);
  if (configured === "all") return "all";
  if (configured === "done") return "done";
  if (configured === "pending") return "pending";
  if (configured === "processed") return "processed";
  const pathname = textValue(window.location.pathname);
  return pathname.endsWith("/work/done") || pathname.includes("/work/done/")
    ? "done"
    : "pending";
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
      detail = await refreshWorkTaskDetailTarget(
        store,
        customerID,
        assetID,
        positiveTextID(task.workflow_instance_id || asset?.flow?.id),
      );
      taskCustomer = detail?.customer || customer;
      taskAsset = assetID ? detail?.asset || asset : undefined;
      fullTask =
        findWorkDetailTask(fullTask, taskCustomer, taskAsset) || fullTask;
      fullTask = withWorkTaskRuleContext(fullTask, detail?.operations || []);
    } catch (error) {
      toast.error(errorMessage(error, "客户资料加载失败"));
    }
  }
  setWorkStoreValue(store, "data.actionTarget.workTask", fullTask);
  setWorkStoreValue(store, "data.actionTarget.workTaskFlow", null);
  setWorkStoreValue(store, "data.actionTarget.workTaskCustomer", taskCustomer);
  setWorkStoreValue(
    store,
    "data.actionTarget.workTaskAsset",
    taskAsset ?? null,
  );
  setWorkStoreValue(
    store,
    "data.actionTarget.workTaskName",
    workTaskDialogTitle(fullTask),
  );
  setWorkStoreValue(
    store,
    "data.actionTarget.workTaskDescription",
    workTaskDialogDescription(taskCustomer, taskAsset),
  );
  await openPreparedWorkTaskModal(
    store,
    fullTask,
    taskCustomer,
    taskAsset,
    detail,
  );
}

export async function openWorkLeadTask(
  task: WorkTask,
  leadValues: WorkCustomer,
  store?: StoreLike,
  flow?: WorkFlowDetail | null,
) {
  setWorkStoreValue(store, "data.actionTarget.workTask", task);
  setWorkStoreValue(store, "data.actionTarget.workTaskFlow", flow ?? null);
  setWorkStoreValue(store, "data.actionTarget.workTaskCustomer", leadValues);
  setWorkStoreValue(store, "data.actionTarget.workTaskAsset", null);
  setWorkStoreValue(
    store,
    "data.actionTarget.workTaskName",
    workTaskDialogTitle(task),
  );
  setWorkStoreValue(
    store,
    "data.actionTarget.workTaskDescription",
    workTaskDialogDescription(leadValues),
  );
  await openPreparedWorkTaskModal(store, task, leadValues);
}

async function openPreparedWorkTaskModal(
  store: StoreLike | undefined,
  task: WorkTask,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
  detail?: WorkDetailTargetResponse | null,
) {
  await prepareWorkTaskForm(store, task, customer, asset, detail);
  setWorkModalOpen(store, "dialog.workTask", true);
}

function workTaskDialogTitle(task: WorkTask): string {
  return workTaskName(task) || "处理任务";
}

function workTaskDialogDescription(
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
): string {
  const customerName = textValue(customer?.name || customer?.customer_name);
  const customerNo = textValue(
    customer?.code_display ||
      customer?.customer_no ||
      customer?.code ||
      customer?.no,
  );
  const phone = textValue(customer?.phone || customer?.mobile);
  const assetName = textValue(asset?.asset_name || asset?.name);
  const assetNo = textValue(
    asset?.asset_no || asset?.asset_code || asset?.code,
  );
  return [customerName, customerNo, phone, assetName, assetNo]
    .filter(Boolean)
    .join(" · ");
}

function findWorkDetailTask(
  task: WorkTask,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
): WorkTask | null {
  const tasks = asset
    ? [...(asset.flow?.tasks || []), ...workAssetRowTasks(asset)]
    : workCustomerRowTasks(customer || null);
  const detailTask = tasks.find((candidate) => sameWorkTask(candidate, task));
  return detailTask ? { ...task, ...detailTask } : null;
}

function withWorkTaskRuleContext(
  task: WorkTask,
  operations: WorkOperation[],
): WorkTask {
  if (!workTaskIsApproval(task) || operations.length === 0) return task;
  const workflowInstanceID = positiveTextID(task.workflow_instance_id);
  const stageID = positiveTextID(task.stage_id);
  const operation = operations.find((candidate) => {
    const taskType = textValue(
      candidate.task_type || candidate["task.task_type"],
    );
    if (taskType !== "rule") return false;
    if (
      workflowInstanceID &&
      positiveTextID(candidate.workflow_instance_id) !== workflowInstanceID
    ) {
      return false;
    }
    if (stageID && positiveTextID(candidate.stage_id) !== stageID) return false;
    return Boolean(textValue(candidate.content || candidate.summary));
  });
  if (!operation) return task;
  const diagnosisApproval = workTaskName(task).includes("诊断");
  return {
    ...task,
    context_result: textValue(operation.content || operation.summary),
    context_result_label: diagnosisApproval
      ? "自动诊断结果"
      : displayText(operation.title || operation.task_name, "自动核验结果"),
  };
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
  detail?: WorkDetailTargetResponse | null,
) {
  const formState = buildWorkTaskFormState(task, customer, asset);
  const communicationGroups = Array.isArray(detail?.communication_groups)
    ? detail.communication_groups
    : [];
  const communicationGroupTypes = Array.isArray(
    detail?.communication_group_types,
  )
    ? detail.communication_group_types
    : [];
  const communicationGroupWorkflowInstanceID =
    textValue(detail?.communication_group_workflow_instance_id) ||
    positiveTextID(task.workflow_instance_id);
  const communicationGroupContext: WorkTaskCommunicationGroupContext = {
    groups: communicationGroups,
    groupTypes: communicationGroupTypes,
    workflowInstanceID: communicationGroupWorkflowInstanceID,
    canManage: Boolean(detail?.can_manage_communication_groups),
  };
  const activeCommunicationGroup =
    communicationGroups.find((group) => group.status === "active") || null;
  setWorkStoreValue(store, workTaskFormDataPath, formState.values);
  setWorkStoreValue(store, workTaskFieldMapPath, formState.fieldMap);
  setWorkStoreValue(store, workTaskFormFieldsPath, formState.fields);
  setWorkStoreValue(store, workTaskLayoutPath, formState.layout);
  setWorkStoreValue(store, workTaskActiveGroupPath, "");
  setWorkStoreValue(store, workTaskUploadFilesPath, {});
  setWorkStoreValue(store, workTaskUploadPendingPath, {});
  setWorkStoreValue(
    store,
    workTaskCommunicationGroupContextPath,
    communicationGroupContext,
  );
  setWorkStoreValue(
    store,
    workTaskCommunicationGroupDraftPath,
    communicationGroupDraft(
      activeCommunicationGroup,
      communicationGroupTypes,
      communicationGroupWorkflowInstanceID,
    ),
  );
  setWorkStoreValue(store, workTaskCommunicationGroupErrorPath, "");
  setCurrentWorkTaskFormErrors(store, {});
  replaceWorkTaskFormNodes(store, formState.nodes);
}

function replaceWorkTaskFormNodes(
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
  const tabs: WorkTaskFormGroup[] = [];

  if (workTaskShouldRenderFields(task)) {
    const { groupFields, sectionFields } = workTaskOrganizeFormFields(
      task.form?.fields || [],
    );
    tabs.push(
      ...buildWorkTaskConfiguredGroupTabs(
        groupFields,
        values,
        fieldMap,
        customer,
        asset,
      ),
      ...buildWorkTaskDataTabs(
        sectionFields,
        values,
        fieldMap,
        customer,
        asset,
      ),
    );
  }
  const taskFields = buildWorkTaskActionFields(values, fieldMap, task).map(
    (field) => ({ ...field, groupId: "task" }),
  );
  if (taskFields.length > 0) {
    tabs.push({ id: "task", label: "任务处理", fields: taskFields });
  }
  const mergedTabs = mergeWorkTaskTabs(tabs);
  addWorkTaskTabsNode(nodes, mergedTabs);
  addWorkTaskDateNodes(nodes, mergedTabs, fieldMap);

  const fields = nodes.flatMap(workTaskNodeFormFields);
  const layout = workTaskLayoutMode(nodes);
  nodes.unshift({
    id: "work-task-context",
    type: "show-crm-work-task-context",
    meta: { layout },
  });

  nodes.push({
    id: "work-task-submit-controller",
    type: "show-crm-work-task-form",
  });

  return {
    nodes,
    fields,
    layout,
    values,
    fieldMap,
  };
}

function buildWorkTaskActionFields(
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  task: WorkTask,
): WorkTaskFormField[] {
  if (workTaskIsProduct(task)) {
    const options = (task.product_options || [])
      .map((product) => {
        const id = positiveTextID(product.id);
        const details = [
          textValue(product.category_name),
          textValue(product.service_workflow_name),
        ].filter(Boolean);
        return {
          id,
          value: `${textValue(product.name) || textValue(product.code) || id}${
            details.length > 0 ? ` · ${details.join(" / ")}` : ""
          }`,
        };
      })
      .filter((product) => Boolean(product.id));
    return buildWorkTaskConfiguredFields(
      values,
      fieldMap,
      [
        {
          formKey: "product_ids",
          rawKey: "product_ids",
          label: "适用产品",
          placeholder: "搜索并选择产品",
          required: Boolean(task.todo_required ?? task.required),
          type: "form-select",
          fullWidth: true,
          options,
          meta: { multiple: true, searchable: true },
          initialValue: task.selected_product_ids || [],
        },
      ],
    );
  }
  if (workTaskIsTodo(task)) {
    return buildWorkTaskConfiguredFields(
      values,
      fieldMap,
      [
        {
          formKey: "result",
          rawKey: "result",
          label: "办理结果",
          placeholder: "请输入本次办理结果",
          required: true,
          type: "form-textarea",
          fullWidth: true,
        },
      ],
    );
  }
  if (!workTaskIsApproval(task)) return [];
  const opinionRequirement =
    textValue(task.opinion_requirement) || "reject_required";
  return buildWorkTaskConfiguredFields(
    values,
    fieldMap,
    [
      {
        formKey: "approval_result",
        rawKey: "approval_result",
        label: "审核结果",
        placeholder: "请选择审核结果",
        required: true,
        type: "form-select",
        options: [
          { id: "approved", value: "通过" },
          { id: "rejected", value: "驳回" },
        ],
      },
      {
        formKey: "opinion",
        rawKey: "opinion",
        label: "审核意见",
        placeholder: "请输入审核意见",
        required: opinionRequirement === "required",
        type: "form-textarea",
        fullWidth: true,
        meta:
          opinionRequirement === "reject_required"
            ? {
                requiredWhenRawKey: "approval_result",
                requiredWhenValue: "rejected",
              }
            : undefined,
      },
    ],
  );
}

type WorkTaskConfiguredField = Omit<WorkTaskFormField, "formKey"> & {
  formKey: string;
  rawKey: string;
  initialValue?: unknown;
};

function buildWorkTaskConfiguredFields(
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  configuredFields: WorkTaskConfiguredField[],
): WorkTaskFormField[] {
  return configuredFields.map((field) => {
    const formKey = uniqueWorkTaskFormKey(field.formKey, fieldMap);
    values[formKey] = formatWorkTaskInitialValue({
      type: field.type,
      initialValue: field.initialValue,
    });
    fieldMap[formKey] = field.rawKey;
    return {
      formKey,
      label: field.label,
      placeholder: field.placeholder,
      required: field.required,
      readonly: field.readonly,
      type: field.type,
      inputType: field.inputType,
      fullWidth: field.fullWidth,
      options: field.options,
      meta: field.meta,
    };
  });
}

function workFormFieldIsGroup(field: WorkFormField): boolean {
  return textValue(field.field_type) === "group";
}

function workTaskOrganizeFormFields(fields: WorkFormField[]): {
  groupFields: WorkFormField[];
  sectionFields: WorkFormField[];
} {
  const groups = new Map<string, WorkFormField>();
  const sectionFields: WorkFormField[] = [];

  for (const field of fields) {
    if (workFormFieldIsGroup(field)) {
      mergeWorkTaskFormGroup(groups, workTaskFormGroupKey(field), field);
      continue;
    }
    const groupKey = workTaskFormGroupKey(field);
    if (!groupKey) {
      sectionFields.push(field);
      continue;
    }
    mergeWorkTaskFormGroup(groups, groupKey, {
      id: field.group_id,
      name: textValue(field.group_label) || textValue(field.group_key),
      field_key: textValue(field.group_key) || `group:${groupKey}`,
      field_type: "group",
      children: [field],
    });
  }

  return {
    groupFields: Array.from(groups.values()),
    sectionFields,
  };
}

function workTaskFormGroupKey(field: WorkFormField): string {
  if (workFormFieldIsGroup(field)) {
    return (
      positiveTextID(field.data_field_id || field.id) ||
      textValue(field.field_key)
    );
  }
  return positiveTextID(field.group_id) || textValue(field.group_key);
}

function mergeWorkTaskFormGroup(
  groups: Map<string, WorkFormField>,
  key: string,
  incoming: WorkFormField,
) {
  if (!key) return;
  const current = groups.get(key);
  if (!current) {
    groups.set(key, {
      ...incoming,
      children: uniqueWorkTaskGroupChildren(incoming.children || []),
    });
    return;
  }
  groups.set(key, {
    ...current,
    ...incoming,
    children: uniqueWorkTaskGroupChildren([
      ...(current.children || []),
      ...(incoming.children || []),
    ]),
  });
}

function uniqueWorkTaskGroupChildren(fields: WorkFormField[]): WorkFormField[] {
  const seen = new Set<string>();
  return fields.filter((field) => {
    const key =
      positiveTextID(field.data_field_id || field.id) || workFieldKey(field);
    if (!key || seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function buildWorkTaskDataTabs(
  fields: WorkFormField[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
): WorkTaskFormGroup[] {
  return workTaskFieldSections(fields).flatMap((section) => {
    const controls = section.fields
      .map((field) =>
        workTaskGroupField(field, values, fieldMap, customer, asset),
      )
      .filter((field): field is WorkTaskFormField => Boolean(field))
      .map((field) => ({ ...field, groupId: section.id }));
    return controls.length > 0
      ? [{ id: section.id, label: section.label, fields: controls }]
      : [];
  });
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
  return ["lead", "customer", "asset", "task"]
    .map((id) => sections.get(id))
    .filter((section): section is FieldSectionDraft =>
      Boolean(section && section.fields.length > 0),
    );
}

function workTaskFieldSection(field: WorkFormField): {
  id: string;
  label: string;
} {
  if (workFormFieldBelongsToLead(field)) {
    return { id: "lead", label: "线索信息" };
  }
  if (workFormFieldBelongsToAsset(field)) {
    return { id: "asset", label: "资产信息" };
  }
  if (workFormFieldBelongsToCustomer(field)) {
    return { id: "customer", label: "客户信息" };
  }
  return { id: "task", label: "任务处理" };
}

function workFormFieldBelongsToLead(field: WorkFormField): boolean {
  return positiveTextID(field.data_template_cate_id) === "4";
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

function buildWorkTaskConfiguredGroupTabs(
  groupFields: WorkFormField[],
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
): WorkTaskFormGroup[] {
  return groupFields
    .map((group) => workTaskGroupTab(group, values, fieldMap, customer, asset))
    .filter((tab): tab is WorkTaskFormGroup => Boolean(tab?.fields.length));
}

function addWorkTaskTabsNode(
  nodes: WorkTaskFormNode[],
  tabs: WorkTaskFormGroup[],
) {
  if (tabs.length === 0) return;

  nodes.push({
    id: "work-task-group-tabs",
    type: "show-crm-work-task-group-tabs",
    meta: {
      tabs,
    },
  });
}

function addWorkTaskDateNodes(
  nodes: WorkTaskFormNode[],
  tabs: WorkTaskFormGroup[],
  fieldMap: Record<string, string>,
) {
  const fields = tabs.flatMap((tab) => tab.fields);
  for (const tab of tabs) {
    for (const field of tab.fields) {
      if (field.type !== "form-date") continue;
      const visibility = workTaskDateVisibility(field, fieldMap, fields);
      nodes.push({
        id: `work-task-date-${field.formKey}`,
        type: "form-date",
        className: "crm-work-task-date-field",
        name: field.label,
        placeholder: field.placeholder,
        value: `workTaskForm.${field.formKey}`,
        mode: "form",
        validate: field.required
          ? [{ type: "required", message: `${field.label}不能为空` }]
          : undefined,
        meta: {
          ...field.meta,
          formLayout: "vertical",
          inputType:
            field.inputType === "datetime-local"
              ? "datetime-local"
              : "date",
          readonly: field.readonly || undefined,
          showWhen:
            visibility.conditions.length > 0
              ? visibility.conditions
              : undefined,
          showCondition:
            visibility.conditions.length > 0 ? visibility.mode : undefined,
          hiddenWhen: [
            {
              path: workTaskActiveGroupPath,
              operator: "notEquals",
              value: tab.id,
            },
          ],
        },
      });
    }
  }
}

function workTaskDateVisibility(
  field: WorkTaskFormField,
  fieldMap: Record<string, string>,
  fields: WorkTaskFormField[],
): {
  conditions: Array<Record<string, unknown>>;
  mode: "all" | "any";
} {
  const rules = workTaskFormFieldVisibilityRules(field, fieldMap, fields);
  if (rules.length === 0) return { conditions: [], mode: "all" };
  const groups = rules.map((rule) =>
    workTaskDateVisibilityForRule(rule, fieldMap),
  );
  if (groups.length === 1) return groups[0];
  if (groups.every((group) => group.mode === "all")) {
    return {
      conditions: groups.flatMap((group) => group.conditions),
      mode: "all",
    };
  }
  return groups[groups.length - 1];
}

function workTaskDateVisibilityForRule(
  rule: WorkTaskFormFieldVisibilityRule,
  fieldMap: Record<string, string>,
): {
  conditions: Array<Record<string, unknown>>;
  mode: "all" | "any";
} {
  const driverFormKey = Object.entries(fieldMap).find(
    ([, rawKey]) => rawKey === rule.driverRawKey,
  )?.[0];
  if (!driverFormKey) return { conditions: [], mode: "all" };
  const path = `${workTaskFormDataPath}.${driverFormKey}`;
  if (rule.operator === "empty" || rule.operator === "not_empty") {
    return {
      conditions: [
        {
          path,
          operator: rule.operator === "empty" ? "empty" : "notEmpty",
        },
      ],
      mode: "all",
    };
  }
  const values =
    rule.operator === "equals" || rule.operator === "not_equals"
      ? rule.expectedValues.slice(0, 1)
      : rule.expectedValues;
  return {
    conditions: values.map((value) => ({
      path,
      operator:
        rule.operator === "not_equals" || rule.operator === "not_in"
          ? "notEquals"
          : "equals",
      value,
    })),
    mode: rule.operator === "in" && values.length > 1 ? "any" : "all",
  };
}

function mergeWorkTaskTabs(tabs: WorkTaskFormGroup[]): WorkTaskFormGroup[] {
  const merged = new Map<string, WorkTaskFormGroup>();
  for (const tab of tabs) {
    const current = merged.get(tab.id);
    if (!current) {
      merged.set(tab.id, { ...tab, fields: [...tab.fields] });
      continue;
    }
    const fieldKeys = new Set(current.fields.map((field) => field.formKey));
    const additionalFields = tab.fields.filter(
      (field) => !fieldKeys.has(field.formKey),
    );
    merged.set(tab.id, {
      ...current,
      fields: [...current.fields, ...additionalFields],
    });
  }
  return Array.from(merged.values());
}

function workTaskGroupTab(
  group: WorkFormField,
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  customer?: WorkCustomer | null,
  asset?: WorkAsset,
): WorkTaskFormGroup | null {
  const label =
    textValue(group.label) ||
    textValue(group.name) ||
    textValue(group.field_key);
  const id = workTaskFormKey(workFieldKey(group) || label || "group");
  const children = Array.isArray(group.children) ? group.children : [];
  const fields = children
    .filter((field) => !workFormFieldIsGroup(field))
    .map((field) =>
      workTaskGroupField(field, values, fieldMap, customer, asset),
    )
    .filter((field): field is WorkTaskFormField => Boolean(field))
    .map((field) => ({ ...field, groupId: id }));
  if (fields.length === 0) return null;
  return {
    id,
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
): WorkTaskFormField | null {
  const rawKey = workFieldKey(field);
  if (!rawKey) return null;
  const options = Array.isArray(field.options)
    ? field.options.map(workFieldOption)
    : [];
  const renderConfig = workTaskFieldRenderConfig(field, options);
  const formKey = uniqueWorkTaskFormKey(workTaskFormKey(rawKey), fieldMap);
  const label = textValue(field.label) || textValue(field.name) || rawKey;
  const initialValue = workFieldInitialValue(
    field,
    customer,
    asset,
    renderConfig,
  );
  values[formKey] = formatWorkTaskInitialValue({
    type: renderConfig.type,
    initialValue,
  });
  fieldMap[formKey] = rawKey;
  const meta = {
    ...renderConfig.meta,
    ...(field.meta || {}),
    dataFieldKey: textValue(field.field_key) || undefined,
  };
  return {
    formKey,
    label,
    placeholder:
      textValue(field.placeholder) || `${renderConfig.placeholderPrefix}${label}`,
    required: Boolean(field.required),
    readonly: Boolean(field.readonly),
    visibleWhenFieldId:
      positiveTextID(field.visible_when_field_id) || undefined,
    visibleWhenRawKey:
      textValue(field.meta?.["visibleWhenRawKey"]) || undefined,
    visibleWhenOperator:
      textValue(field.visible_when_operator) || undefined,
    visibleWhenValue: textValue(field.visible_when_value) || undefined,
    type: renderConfig.type,
    inputType: renderConfig.inputType,
    fullWidth: renderConfig.fullWidth,
    options: renderConfig.options,
    meta:
      renderConfig.type === "show-crm-work-task-upload"
        ? {
            ...meta,
            initialFiles: workEntityFileValue(field, customer, asset),
          }
        : meta,
  };
}

function workTaskFieldRenderConfig(
  field: WorkFormField,
  options: WorkCommonOption[],
): WorkTaskFieldRenderConfig {
  const fieldType = textValue(field.field_type);
  if (fieldType === "customer_tags") {
    return {
      type: "show-crm-work-customer-tags",
      placeholderPrefix: "请选择",
      fullWidth: true,
      options,
      meta: { multiple: true },
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

  if (fieldType === "public_resource") {
    return {
      type: "form-select",
      placeholderPrefix: "请选择",
      options,
    };
  }

  if (options.length > 0) {
    const multiple =
      fieldType === "multi_select" ||
      fieldType === "multiple_select" ||
      fieldType === "checkbox";
    return {
      type: "form-select",
      placeholderPrefix: "请选择",
      fullWidth: multiple,
      options,
      meta: multiple ? { multiple: true } : undefined,
    };
  }

  if (workTaskFieldIsUpload(field)) {
    return {
      type: "show-crm-work-task-upload",
      placeholderPrefix: "请上传",
      fullWidth: true,
      meta: workTaskUploadFieldMeta(field),
    };
  }

  if (fieldType === "textarea") {
    return {
      type: "form-textarea",
      placeholderPrefix: "请输入",
      fullWidth: true,
    };
  }

  if (
    fieldType === "number" ||
    fieldType === "decimal" ||
    fieldType === "money"
  ) {
    return {
      type: "form-input",
      inputType: "number",
      placeholderPrefix: "请输入",
    };
  }

  if (fieldType === "date") {
    return {
      type: "form-date",
      inputType: "date",
      placeholderPrefix: "请选择",
      meta: { inputType: "date" },
    };
  }

  if (fieldType === "datetime") {
    return {
      type: "form-date",
      inputType: "datetime-local",
      placeholderPrefix: "请选择",
      meta: { inputType: "datetime-local" },
    };
  }

  return {
    type: "form-input",
    inputType: "text",
    placeholderPrefix: "请输入",
  };
}

function workTaskFieldIsUpload(field: WorkFormField): boolean {
  const fieldType = textValue(field.field_type);
  return (
    fieldType === "attachment" ||
    fieldType === "file" ||
    fieldType === "image" ||
    fieldType === "audio" ||
    fieldType === "video"
  );
}

function workTaskUploadFieldMeta(
  field: WorkFormField,
): Record<string, unknown> {
  const fieldType = textValue(field.field_type);
  const uploadConfig =
    workTaskUploadConfig[fieldType as keyof typeof workTaskUploadConfig] ||
    workTaskUploadConfig.attachment;
  return {
    ruleId: uploadConfig.ruleId,
    kind: uploadConfig.kind,
    uploadType: "list",
    maxCount: 10,
    bizKey: "crm.work",
    bizName: "CRM工作台",
  };
}

const workTaskUploadConfig = {
  attachment: { ruleId: 6, kind: "file" },
  file: { ruleId: 6, kind: "file" },
  image: { ruleId: 1, kind: "image" },
  audio: { ruleId: 3, kind: "audio" },
  video: { ruleId: 2, kind: "video" },
} as const;

function formatWorkTaskInitialValue(config: {
  type?: string;
  initialValue?: unknown;
}): unknown {
  if (config.type === "show-crm-work-task-upload") {
    return formatUploadInitialValue(config.initialValue);
  }
  if (Array.isArray(config.initialValue)) {
    return config.initialValue.map(textValue).filter(Boolean);
  }
  if (typeof config.initialValue === "boolean") {
    return config.initialValue;
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

function openWorkDetail(
  customer: WorkCustomer,
  store?: StoreLike,
  asset?: WorkAsset,
) {
  setWorkStoreValue(store, workDetailProfilePrefetchPath, null);
  setWorkDetailTarget(store, customer, asset);
  setWorkModalOpen(store, "dialog.workDetail", true);
}

const workDetailProfilePrefetchPath =
  "data.actionTarget.workDetailProfilePrefetch";

export async function openWorkCustomerDetailDrawer(
  store: WorkStoreLike | undefined,
  customerID: string,
  assetID = "",
  workflowInstanceID = "",
): Promise<boolean> {
  const detail = await refreshWorkDetailProfileTarget(
    store,
    customerID,
    assetID,
    workflowInstanceID,
  );
  if (!detail?.customer) return false;
  setWorkStoreValue(store, workDetailProfilePrefetchPath, detail);
  setWorkModalOpen(store, "dialog.workDetail", true);
  return true;
}

type WorkDetailTargetResponse = {
  customer?: WorkCustomer;
  asset?: WorkAsset | null;
  operations?: WorkOperation[];
  list?: WorkOperation[];
  flow?: WorkFlowDetail | null;
  detail_sections?: WorkDetailSection[];
  communication_groups?: WorkCommunicationGroup[];
  communication_group_types?: WorkCommunicationGroupType[];
  communication_group_workflow_instance_id?: string | number;
  can_manage_communication_groups?: boolean;
};

type WorkDetailOperationsResponse = {
  list?: WorkOperation[];
  operations?: WorkOperation[];
};

type WorkDetailAttachmentsResponse = {
  list?: unknown[];
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

async function refreshWorkDetailProfileTarget(
  store: StoreLike | undefined,
  customerID: string,
  assetID = "",
  workflowInstanceID = "",
): Promise<WorkDetailTargetResponse | null> {
  if (!customerID) return null;
  const query = workDetailQuery(customerID, assetID, workflowInstanceID);
  const payload = await workApi<WorkDetailTargetResponse>(
    `/crm/work/customer_profile?${query.toString()}`,
  );
  if (payload.customer) {
    setWorkDetailTarget(store, payload.customer, payload.asset ?? null);
  }
  return payload;
}

async function refreshWorkTaskDetailTarget(
  store: StoreLike | undefined,
  customerID: string,
  assetID = "",
  workflowInstanceID = "",
): Promise<WorkDetailTargetResponse | null> {
  if (!customerID) return null;
  const query = workDetailQuery(customerID, assetID, workflowInstanceID);
  const payload = await workApi<WorkDetailTargetResponse>(
    `/crm/work/customer_detail?${query.toString()}`,
  );
  if (payload.customer) {
    setWorkDetailTarget(store, payload.customer, payload.asset ?? null);
  }
  return payload;
}

async function fetchWorkDetailOperations(
  customerID: string,
  assetID = "",
): Promise<WorkOperation[]> {
  const query = workDetailQuery(customerID, assetID);
  const payload = await workApi<WorkDetailOperationsResponse>(
    `/crm/work/customer_operations?${query.toString()}`,
  );
  return Array.isArray(payload.list)
    ? payload.list
    : Array.isArray(payload.operations)
      ? payload.operations
      : [];
}

async function fetchWorkDetailAttachments(
  customerID: string,
  assetID = "",
  workflowInstanceID = "",
): Promise<WorkCustomerDetailAttachment[]> {
  const query = workDetailQuery(customerID, assetID, workflowInstanceID);
  const payload = await workApi<WorkDetailAttachmentsResponse>(
    `/crm/work/customer_attachments?${query.toString()}`,
  );
  return normalizeWorkCustomerDetailAttachments(payload.list);
}

function workDetailQuery(
  customerID: string,
  assetID = "",
  workflowInstanceID = "",
): URLSearchParams {
  const query = new URLSearchParams({ customer_id: customerID });
  if (assetID) query.set("asset_id", assetID);
  if (workflowInstanceID) {
    query.set("workflow_instance_id", workflowInstanceID);
  }
  return query;
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

const workRecordChangePresentation: Record<
  string,
  { label: string; className: string }
> = {
  added: {
    label: "新增",
    className: "bg-emerald-50 text-emerald-700",
  },
  updated: {
    label: "修改",
    className: "bg-amber-50 text-amber-700",
  },
  cleared: {
    label: "清空",
    className: "bg-rose-50 text-rose-700",
  },
};

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
  | "archived"
  | "personalPending"
  | "overdue"
  | "completedToday";

function workQuickFilterFromURL(): WorkQuickFilter {
  const value = workURLFilterValue("quick_filter", "quickFilter");
  if (
    value === "hasTasks" ||
    value === "missingAsset" ||
    value === "archived" ||
    value === "personalPending" ||
    value === "overdue" ||
    value === "completedToday"
  ) {
    return value;
  }
  return "all";
}

type WorkCustomerPageState = {
  page: number;
  pageSize: number;
  total: number;
};

const workCustomerPageSize = 10;

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
  const [feishuMessage, setFeishuMessage] = useState(
    "飞书内可免输密码登录，浏览器中可跳转飞书授权。",
  );
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
      const redirectURL = new URL(
        window.location.pathname,
        window.location.origin,
      );
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
              <Input
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
              <Input
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
          <Button
            type="submit"
            disabled={submitting}
            className="h-10 w-full px-4"
            style={{ marginTop: 28 }}
          >
            {submitting ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Check className="h-4 w-4" />
            )}
            登录
          </Button>
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
          <Button
            type="button"
            variant="outline"
            disabled={feishuStatus === "loading"}
            onClick={handleBrowserFeishuLogin}
            className="h-10 w-full px-4"
            style={{ marginTop: 20 }}
          >
            {feishuStatus === "loading" ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <LogIn className="h-4 w-4" />
            )}
            飞书登录
          </Button>
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
    <div className="crm-stats-dashboard grid gap-3">
      <WorkStatsDashboardStyles />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold text-foreground">我的工作概览</h2>
          <p className="mt-1 text-xs text-muted-foreground">
            只统计当前账号需要处理和今天已经完成的任务
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

      <div className="crm-stats-dashboard-analysis grid min-w-0 gap-3">
        <WorkStatsTrendCard points={summary.trend || []} />
        <WorkStatsRecentOperations
          operations={summary.recent_operations || []}
          loading={loading}
        />
      </div>

      <div className="crm-stats-dashboard-breakdowns grid min-w-0 gap-3">
        <WorkStatsBreakdownCard
          title="阶段分布"
          description="我有待办的流程当前所在阶段"
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

function WorkStatsDashboardStyles() {
  return (
    <style>{`
      .crm-stats-dashboard-metrics {
        grid-template-columns: repeat(4, minmax(0, 1fr));
      }

      .crm-stats-dashboard-analysis {
        grid-template-columns: minmax(0, 1.65fr) minmax(320px, 0.75fr);
      }

      .crm-stats-dashboard-breakdowns {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }

      .crm-stats-dashboard-metric {
        min-height: 94px;
      }

      .crm-stats-dashboard-analysis-panel {
        height: 312px;
      }

      @media (max-width: 1439px) {
        .crm-stats-dashboard-metrics {
          grid-template-columns: repeat(3, minmax(0, 1fr));
        }

        .crm-stats-dashboard-analysis {
          grid-template-columns: minmax(0, 1fr);
        }
      }

      @media (max-width: 767px) {
        .crm-stats-dashboard-metrics {
          grid-template-columns: repeat(2, minmax(0, 1fr));
        }

        .crm-stats-dashboard-breakdowns {
          grid-template-columns: minmax(0, 1fr);
        }
      }
    `}</style>
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
    <div className="crm-stats-dashboard-metrics grid gap-px overflow-hidden rounded-md border border-border/70 bg-border/70">
      {metrics.map((metric) => (
        <WorkStatsMetricCard
          key={textValue(metric.key || metric.name)}
          metric={metric}
          onOpen={
            metric.drilldown_path
              ? () => openWorkSummaryDrilldown(metric.drilldown_path)
              : undefined
          }
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
  onOpen?: () => void;
}) {
  const Icon = workStatsMetricIcon(metric.key);
  const content = (
    <>
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
    </>
  );
  if (!onOpen) {
    return (
      <div className="crm-stats-dashboard-metric w-full bg-background px-4 py-3 text-left">
        {content}
      </div>
    );
  }
  return (
    <Button
      type="button"
      variant="ghost"
      className="crm-stats-dashboard-metric h-auto w-full flex-col items-stretch gap-0 rounded-none bg-background px-4 py-3 text-left hover:bg-muted/20 focus-visible:z-10 focus-visible:ring-inset"
      onClick={onOpen}
    >
      {content}
    </Button>
  );
}

function openWorkSummaryDrilldown(path?: string) {
  const target = textValue(path);
  if (!target) return;
  const base = getWorkEntryPath().replace(/\/$/, "");
  window.location.assign(
    `${base}${target.startsWith("/") ? target : `/${target}`}`,
  );
}

function workStatsMetricIcon(key?: string) {
  switch (textValue(key)) {
    case "pending_targets":
    case "pending_tasks":
      return ClipboardList;
    case "completed_today":
      return TrendingUp;
    case "overdue_tasks":
      return AlertTriangle;
    default:
      return ClipboardList;
  }
}

type WorkStatsTrendSeriesKey =
  | "task_count"
  | "transition_count";

const workStatsTrendSeries: Array<{
  key: WorkStatsTrendSeriesKey;
  label: string;
  color: string;
}> = [
  { key: "task_count", label: "任务完成", color: "#111827" },
  { key: "transition_count", label: "阶段流转", color: "#2563eb" },
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
      className={`${workStatsPanelClass} crm-stats-dashboard-analysis-panel flex min-h-0 flex-col`}
    >
      <WorkStatsPanelHeader
        title="近 14 天趋势"
        description="任务完成和阶段流转分别统计，不重复相加"
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

function WorkStatsBreakdownList({
  rows,
  type,
}: {
  rows: WorkSummaryBreakdown[];
  type: "stage" | "task";
}) {
  return (
    <div className="mt-3 grid gap-1">
      {rows.map((row, index) => {
        const value = textValue(row.key || row.name);
        const percent = workStatsPercent(row.percent);
        const drilldownPath = textValue(row.drilldown_path);
        const content = (
          <>
            <span className="flex items-center justify-between gap-3">
              <span className="min-w-0 truncate text-xs font-medium text-foreground">
                {displayText(row.name)}
              </span>
              <span className="shrink-0 text-xs font-semibold text-foreground">
                {workStatsNumber(row.count)}
                <small className="ml-2 font-normal text-muted-foreground">
                  {percent}%
                </small>
              </span>
            </span>
            <span
              className="mt-1.5 block h-1 overflow-hidden rounded-full bg-muted"
              aria-hidden="true"
            >
              <span
                className={`block h-full rounded-full ${
                  type === "stage" ? "bg-blue-600" : "bg-emerald-600"
                }`}
                style={{ width: `${percent}%` }}
              />
            </span>
          </>
        );
        if (!drilldownPath) {
          return (
            <div
              key={`${type}:${value || index}`}
              className="w-full rounded-md px-2 py-2 text-left"
            >
              {content}
            </div>
          );
        }
        return (
          <Button
            type="button"
            key={`${type}:${value || index}`}
            variant="ghost"
            className="group h-auto w-full flex-col items-stretch gap-0 rounded-md px-2 py-2 text-left font-normal hover:bg-muted/25"
            onClick={() => openWorkSummaryDrilldown(drilldownPath)}
          >
            {content}
          </Button>
        );
      })}
    </div>
  );
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
      className={`${workStatsPanelClass} crm-stats-dashboard-analysis-panel flex min-h-0 flex-col`}
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

function workCustomerListTaskView(task: WorkTask): WorkCustomerListTaskView {
  return {
    key: workTaskKey(task),
    label: workTaskButtonLabel(task),
    result: textValue(task.result),
    kind: workTaskIsRule(task) ? "rule" : "action",
    canOperate: task.can_operate !== false,
    task,
  };
}

function workCustomerListRowView(item: WorkItem): WorkCustomerListRowView {
  const { customer, asset } = item;
  const target = asset || customer;
  const flow = asset?.flow || null;
  const tasks = item.tasks.map(workCustomerListTaskView);
  const firstAction = tasks.find((task) => task.kind === "action")?.task;
  const ownerName = textValue(
    flow?.owner_staff_name ||
      target.owner_staff_name ||
      target["state.owner_staff_name"] ||
      firstAction?.assignee_staff_name,
  );
  const statusName = workStatusName(target);
  const hasStage = Boolean(statusName && statusName !== "-");

  return {
    id: item.id,
    item,
    customerName: workCustomerTitle(customer),
    customerNo: workItemCustomerNo(item),
    phone: workCustomerPhone(customer),
    wechat: displayText(customer.wechat),
    assetName: asset ? assetTitle(asset) : "未录入资产",
    assetNo: asset ? workItemAssetNo(item) : "后续任务补充",
    assetStatus: asset ? displayText(asset.asset_status_name) : "-",
    stageName: hasStage ? statusName : "未进入流程",
    hasStage,
    ownerName: displayText(ownerName),
    stageDays: workPositiveNumber(target.stage_days),
    lastOperatedAt: textValue(target.last_operated_at),
    processedTaskName: textValue(target.processed_task_name),
    processedResult: textValue(target.processed_result),
    processedContent: textValue(target.processed_content),
    processedAt: textValue(target.processed_at),
    flow,
    tasks,
  };
}

export function ShowCrmWorkCustomerTable({ item, store }: WorkNodeProps) {
  const workflowID = workURLFilterValue("workflow_id", "workflowId");
  const stageFilter = workURLFilterValue("stage_filter", "stage");
  const taskFilter = workURLFilterValue("task_filter", "task");
  const [customers, setCustomers] = useState<WorkCustomer[]>([]);
  const [keyword, setKeyword] = useState(() => workURLFilterValue("keyword"));
  const [mode, setMode] = useState<WorkCustomerMode>(() =>
    workCustomerModeFromNode(item),
  );
  const [quickFilter, setQuickFilter] = useState<WorkQuickFilter>(workQuickFilterFromURL);
  const [modeCounts, setModeCounts] = useState<WorkCustomerModeCounts>(
    emptyWorkCustomerModeCounts,
  );
  const [scope, setScope] = useState<WorkCustomerScope>(() =>
    workURLFilterValue("scope") === "all" ? "all" : "mine",
  );
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

  const loadCustomers = useCallback(
    async (page = 1) => {
      const query = workCustomerQuery(
        keyword,
        mode,
        page,
        workCustomerPageSize,
        {
          workflowID,
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
    },
    [
      keyword,
      mode,
      quickFilter,
      scope,
      stageFilter,
      taskFilter,
      workflowID,
    ],
  );

  const openSearchTarget = useCallback(
    async (customerID: string, assetID: string) => {
      if (!customerID) return;
      try {
        await openWorkCustomerDetailDrawer(store, customerID, assetID);
      } catch (error) {
        toast.error(errorMessage(error, "客户详情加载失败"));
      }
    },
    [store],
  );

  useEffect(() => {
    setCustomers([]);
    loadCustomers(1);
  }, [loadCustomers]);

  useEffect(() => {
    const params = currentWorkURLParams();
    const customerID = textValue(
      params.get("customer_id") || params.get("customerId"),
    );
    const assetID = textValue(params.get("asset_id") || params.get("assetId"));
    if (customerID) void openSearchTarget(customerID, assetID);
  }, [openSearchTarget]);

  useEffect(() => {
    const search = (event: Event) => {
      const detail = readWorkListSearch(event);
      if (detail.workflowID && detail.workflowID !== workflowID) return;
      setCustomers([]);
      setKeyword(detail.keyword);
      if (
        detail.mode === "all" ||
        detail.mode === "pending" ||
        detail.mode === "processed" ||
        detail.mode === "done"
      ) {
        setMode(detail.mode);
      }
      if (detail.scope === "mine" || detail.scope === "all") {
        setScope(detail.scope);
      }
    };
    window.addEventListener(workListSearchEvent, search);
    return () => window.removeEventListener(workListSearchEvent, search);
  }, [workflowID]);

  useEffect(() => {
    const handler = () => {
      setCustomers([]);
      loadCustomers(1);
    };
    window.addEventListener(workRefreshEvent, handler);
    return () => window.removeEventListener(workRefreshEvent, handler);
  }, [loadCustomers]);

  const workItems = useMemo(() => buildWorkItems(customers), [customers]);
  const rows = useMemo(
    () => workItems.map(workCustomerListRowView),
    [workItems],
  );
  const goToPage = (nextPage: number) => {
    if (loading || nextPage === pageState.page) return;
    setCustomers([]);
    loadCustomers(nextPage);
  };

  return (
    <WorkCustomerListView
      rows={rows}
      loading={loading}
      mode={mode}
      modeCounts={modeCounts}
      scope={scope}
      canDispatch={canDispatch}
      page={pageState.page}
      pageSize={pageState.pageSize}
      total={pageState.total}
      emptyTitle={modeConfig.emptyTitle}
      emptyDescription={modeConfig.emptyDescription}
      onModeChange={(nextMode) => {
        setMode(nextMode);
        setQuickFilter("all");
      }}
      onScopeChange={(nextScope) => {
        setCustomers([]);
        setScope(nextScope);
      }}
      onPageChange={goToPage}
      onRefresh={notifyWorkDataChanged}
      onOpenDetail={(row) =>
        openWorkDetail(row.item.customer, store, row.item.asset)
      }
      onOpenTask={(row, task) =>
        void openRowTask(row.item.customer, task.task, store, row.item.asset)
      }
    />
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

  const initialProfile = workStoreValue<WorkDetailTargetResponse | null>(
    store,
    workDetailProfilePrefetchPath,
    null,
  );

  return (
    <WorkDetailContent
      customer={customer}
      asset={asset}
      store={store}
      initialProfile={initialProfile}
    />
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
  const summaryGroups = useMemo(
    () => workRecordSummaryGroups(summaryItems),
    [summaryItems],
  );
  const content = record.content || record.remark || "";
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
              <Button
                type="button"
                key={group.id}
                variant="ghost"
                aria-pressed={active}
                className={`h-auto rounded-none border-b-2 px-1.5 py-2 text-sm font-medium ${
                  active
                    ? "border-primary text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                }`}
                onClick={() => onActiveGroupChange(group.id)}
              >
                {group.label}
              </Button>
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
              <td className="w-[140px] min-w-[140px] bg-muted/15 px-4 py-2.5 text-muted-foreground">
                <div className="flex min-w-0 flex-wrap items-center gap-1.5">
                  <span className="break-words">
                    {displayText(item.label, "-")}
                  </span>
                  <WorkRecordChangeBadge changeType={item.change_type} />
                </div>
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

function WorkRecordChangeBadge({ changeType }: { changeType: unknown }) {
  const presentation = workRecordChangePresentation[textValue(changeType)];
  if (!presentation) return null;
  return (
    <span
      className={`shrink-0 rounded px-1.5 py-0.5 text-[11px] font-medium leading-4 ${presentation.className}`}
    >
      {presentation.label}
    </span>
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
  const changeType = textValue(item.change_type);
  const previousValue = textValue(item.previous_value);
  const currentValue = (
    <WorkRecordCurrentSummaryValue
      item={item}
      onPreviewFile={onPreviewFile}
    />
  );
  if (changeType !== "added" && previousValue) {
    return (
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span className="break-words text-muted-foreground line-through decoration-muted-foreground/50">
          {previousValue}
        </span>
        <ArrowRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground/70" />
        <div className="min-w-0 flex-1">{currentValue}</div>
      </div>
    );
  }
  return currentValue;
}

function WorkRecordCurrentSummaryValue({
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
            <Button
              type="button"
              variant="ghost"
              className="h-auto min-w-0 justify-start truncate px-0 py-0 text-left text-sm font-medium text-foreground underline-offset-4 hover:bg-transparent hover:text-primary hover:underline"
              title={file.name}
              onClick={() => onPreviewFile(file)}
            >
              {file.name || "附件"}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0 text-muted-foreground"
              aria-label="下载附件"
              onClick={() => void downloadUploadFile(file)}
            >
              <Download className="h-4 w-4" />
            </Button>
          </div>
        ))}
      </div>
    );
  }

  return <>{displayText(item.value, "-")}</>;
}

function workDetailProfileForTarget(
  profile: WorkDetailTargetResponse | null | undefined,
  customerID: string,
  assetID: string,
): WorkDetailTargetResponse | null {
  if (!profile?.customer || workCustomerID(profile.customer) !== customerID) {
    return null;
  }
  const profileAssetID = textValue(profile.asset?.id);
  if (profileAssetID !== assetID) return null;
  return profile;
}

function useWorkDetailData(
  initialCustomer: WorkCustomer,
  initialAsset: WorkAsset | null | undefined,
  store?: StoreLike,
  initialProfile?: WorkDetailTargetResponse | null,
) {
  const customerID = workCustomerID(initialCustomer);
  const assetID = textValue(initialAsset?.id);
  const workflowInstanceID = positiveTextID(
    initialAsset?.flow?.workflow_instance_id ||
      initialAsset?.flow?.id ||
      initialAsset?.workflow_instance_id,
  );
  const initialCustomerRef = useRef(initialCustomer);
  const initialAssetRef = useRef<WorkAsset | null>(initialAsset ?? null);
  const profileSeed = workDetailProfileForTarget(
    initialProfile,
    customerID,
    assetID,
  );
  const profileSeedRef = useRef<WorkDetailTargetResponse | null>(profileSeed);
  const profileSeedConsumedRef = useRef(false);
  const profileRequestRef = useRef(0);
  const operationsRequestRef = useRef(0);
  const attachmentsRequestRef = useRef(0);
  const [customer, setCustomer] = useState(
    profileSeed?.customer ?? initialCustomer,
  );
  const [asset, setAsset] = useState<WorkAsset | null>(
    profileSeed?.asset ?? initialAsset ?? null,
  );
  const [flow, setFlow] = useState<WorkFlowDetail | null>(
    profileSeed?.flow ?? profileSeed?.asset?.flow ?? initialAsset?.flow ?? null,
  );
  const [operations, setOperations] = useState<WorkOperation[]>([]);
  const [attachments, setAttachments] = useState<
    WorkCustomerDetailAttachment[]
  >([]);
  const [detailSections, setDetailSections] = useState<WorkDetailSection[]>(
    normalizeWorkDetailSections(profileSeed?.detail_sections),
  );
  const [communicationGroups, setCommunicationGroups] = useState<
    WorkCommunicationGroup[]
  >(
    Array.isArray(profileSeed?.communication_groups)
      ? profileSeed.communication_groups
      : [],
  );
  const [communicationGroupTypes, setCommunicationGroupTypes] = useState<
    WorkCommunicationGroupType[]
  >(
    Array.isArray(profileSeed?.communication_group_types)
      ? profileSeed.communication_group_types
      : [],
  );
  const [
    communicationGroupWorkflowInstanceID,
    setCommunicationGroupWorkflowInstanceID,
  ] = useState(
    textValue(profileSeed?.communication_group_workflow_instance_id),
  );
  const [canManageCommunicationGroups, setCanManageCommunicationGroups] =
    useState(Boolean(profileSeed?.can_manage_communication_groups));
  const [profileLoading, setProfileLoading] = useState(!profileSeed);
  const [operationsLoading, setOperationsLoading] = useState(true);
  const [attachmentsLoading, setAttachmentsLoading] = useState(true);
  const [profileError, setProfileError] = useState("");
  const [operationsError, setOperationsError] = useState("");
  const [attachmentsError, setAttachmentsError] = useState("");

  const applyProfile = useCallback(
    (data: WorkDetailTargetResponse) => {
      if (data.customer) setCustomer(data.customer);
      setAsset(data.asset ?? null);
      setFlow(data.flow ?? data.asset?.flow ?? null);
      setDetailSections(normalizeWorkDetailSections(data.detail_sections));
      setCommunicationGroups(
        Array.isArray(data.communication_groups)
          ? data.communication_groups
          : [],
      );
      setCommunicationGroupTypes(
        Array.isArray(data.communication_group_types)
          ? data.communication_group_types
          : [],
      );
      setCommunicationGroupWorkflowInstanceID(
        textValue(data.communication_group_workflow_instance_id),
      );
      setCanManageCommunicationGroups(
        Boolean(data.can_manage_communication_groups),
      );
    },
    [],
  );

  const reloadProfile = useCallback(async () => {
    if (!customerID) return;
    const requestID = profileRequestRef.current + 1;
    profileRequestRef.current = requestID;
    setProfileLoading(true);
    setProfileError("");
    try {
      const data = await refreshWorkDetailProfileTarget(
        store,
        customerID,
        assetID,
        workflowInstanceID,
      );
      if (data && profileRequestRef.current === requestID) {
        applyProfile(data);
      }
    } catch (error) {
      if (profileRequestRef.current === requestID) {
        setProfileError(errorMessage(error, "详细信息加载失败"));
      }
    } finally {
      if (profileRequestRef.current === requestID) {
        setProfileLoading(false);
      }
    }
  }, [applyProfile, assetID, customerID, store, workflowInstanceID]);

  const reloadOperations = useCallback(async () => {
    if (!customerID) return;
    const requestID = operationsRequestRef.current + 1;
    operationsRequestRef.current = requestID;
    setOperationsLoading(true);
    setOperationsError("");
    try {
      const rows = await fetchWorkDetailOperations(customerID, assetID);
      if (operationsRequestRef.current === requestID) {
        setOperations(rows);
      }
    } catch (error) {
      if (operationsRequestRef.current === requestID) {
        setOperationsError(errorMessage(error, "时间轴加载失败"));
      }
    } finally {
      if (operationsRequestRef.current === requestID) {
        setOperationsLoading(false);
      }
    }
  }, [assetID, customerID]);

  const reloadAttachments = useCallback(async () => {
    if (!customerID) return;
    const requestID = attachmentsRequestRef.current + 1;
    attachmentsRequestRef.current = requestID;
    setAttachmentsLoading(true);
    setAttachmentsError("");
    try {
      const rows = await fetchWorkDetailAttachments(
        customerID,
        assetID,
        workflowInstanceID,
      );
      if (attachmentsRequestRef.current === requestID) {
        setAttachments(rows);
      }
    } catch (error) {
      if (attachmentsRequestRef.current === requestID) {
        setAttachmentsError(errorMessage(error, "附件加载失败"));
      }
    } finally {
      if (attachmentsRequestRef.current === requestID) {
        setAttachmentsLoading(false);
      }
    }
  }, [assetID, customerID, workflowInstanceID]);

  const reloadAll = useCallback(
    () =>
      Promise.allSettled([
        reloadProfile(),
        reloadOperations(),
        reloadAttachments(),
      ]),
    [reloadAttachments, reloadOperations, reloadProfile],
  );

  useEffect(() => {
    initialCustomerRef.current = initialCustomer;
    initialAssetRef.current = initialAsset ?? null;
  }, [initialAsset, initialCustomer]);

  useEffect(() => {
    profileRequestRef.current += 1;
    operationsRequestRef.current += 1;
    attachmentsRequestRef.current += 1;
    profileSeedRef.current = workDetailProfileForTarget(
      initialProfile,
      customerID,
      assetID,
    );
    profileSeedConsumedRef.current = false;
    const seed = profileSeedRef.current;
    setCustomer(seed?.customer ?? initialCustomerRef.current);
    setAsset(seed?.asset ?? initialAssetRef.current);
    setFlow(
      seed?.flow ??
        seed?.asset?.flow ??
        initialAssetRef.current?.flow ??
        null,
    );
    setOperations([]);
    setAttachments([]);
    setDetailSections(normalizeWorkDetailSections(seed?.detail_sections));
    setCommunicationGroups(
      Array.isArray(seed?.communication_groups)
        ? seed.communication_groups
        : [],
    );
    setCommunicationGroupTypes(
      Array.isArray(seed?.communication_group_types)
        ? seed.communication_group_types
        : [],
    );
    setCommunicationGroupWorkflowInstanceID(
      textValue(seed?.communication_group_workflow_instance_id),
    );
    setCanManageCommunicationGroups(
      Boolean(seed?.can_manage_communication_groups),
    );
    setProfileError("");
    setOperationsError("");
    setAttachmentsError("");
    setProfileLoading(!seed);
    setOperationsLoading(true);
    setAttachmentsLoading(true);
  }, [assetID, customerID, initialProfile]);

  useEffect(() => {
    if (profileSeedRef.current && !profileSeedConsumedRef.current) {
      profileSeedConsumedRef.current = true;
      void Promise.allSettled([reloadOperations(), reloadAttachments()]);
      return;
    }
    void reloadAll();
  }, [reloadAll, reloadAttachments, reloadOperations]);

  useEffect(() => {
    if (!customerID) {
      return undefined;
    }
    const reload = () => {
      void reloadAll();
    };
    window.addEventListener(workRefreshEvent, reload);
    return () => {
      window.removeEventListener(workRefreshEvent, reload);
    };
  }, [customerID, reloadAll]);

  return {
    customer,
    asset,
    flow,
    operations,
    attachments,
    detailSections,
    communicationGroups,
    communicationGroupTypes,
    communicationGroupWorkflowInstanceID,
    canManageCommunicationGroups,
    profileLoading,
    operationsLoading,
    attachmentsLoading,
    profileError,
    operationsError,
    attachmentsError,
    reloadProfile,
    reloadOperations,
    reloadAttachments,
  };
}

function WorkDetailContent({
  customer,
  asset,
  store,
  initialProfile,
}: {
  customer: WorkCustomer;
  asset?: WorkAsset | null;
  store?: StoreLike;
  initialProfile?: WorkDetailTargetResponse | null;
}) {
  const [operationScope, setOperationScope] =
    useState<WorkOperationScope>("all");
  const customerID = workCustomerID(customer);
  const assetID = workAssetID(asset);
  const {
    customer: detailCustomer,
    asset: detailAsset,
    flow,
    operations,
    attachments,
    detailSections,
    communicationGroups,
    communicationGroupTypes,
    communicationGroupWorkflowInstanceID,
    canManageCommunicationGroups,
    profileLoading,
    operationsLoading,
    attachmentsLoading,
    profileError,
    operationsError,
    attachmentsError,
    reloadProfile,
    reloadOperations,
    reloadAttachments,
  } = useWorkDetailData(customer, asset ?? null, store, initialProfile);
  const activeAsset = detailAsset || asset || undefined;
  const summary = useMemo(
    () => workDetailWorkspaceSummary(detailCustomer, activeAsset, flow),
    [activeAsset, detailCustomer, flow],
  );

  useEffect(() => {
    setOperationScope("all");
  }, [assetID, customerID]);

  return (
    <WorkCustomerDetailWorkspace
      key={`${customerID}:${assetID}`}
      customer={detailCustomer}
      asset={activeAsset}
      summary={summary}
      sections={detailSections}
      attachments={attachments}
      profileLoading={profileLoading}
      profileError={profileError}
      attachmentsLoading={attachmentsLoading}
      attachmentsError={attachmentsError}
      timelineError={operationsError}
      timelineHasData={operations.length > 0}
      communicationGroups={communicationGroups}
      communicationGroupTypes={communicationGroupTypes}
      communicationGroupWorkflowInstanceID={
        communicationGroupWorkflowInstanceID
      }
      canManageCommunicationGroups={canManageCommunicationGroups}
      onReloadProfile={() => void reloadProfile()}
      onReloadAttachments={() => void reloadAttachments()}
      onReloadTimeline={() => void reloadOperations()}
      timeline={
        <WorkCustomerOperationTimeline
          operations={operations}
          loading={operationsLoading}
          scope={operationScope}
          onScopeChange={setOperationScope}
          store={store}
          variant="rail"
          currentState={
            flow
              ? {
                  ownerName: summary.ownerName,
                  statusName:
                    summary.flowStatus === "active"
                      ? "处理中"
                      : summary.flowStatusName,
                }
              : undefined
          }
        />
      }
    />
  );
}

function workDetailWorkspaceSummary(
  customer: WorkCustomer,
  asset: WorkAsset | undefined,
  flow: WorkFlowDetail | null,
): WorkCustomerDetailWorkspaceSummary {
  const target = asset || customer;
  const customerNo = workCustomerNo(customer);
  const customerPhone = workCustomerPhone(customer);
  const assetNo = workAssetNo(asset);
  return {
    title: workCustomerTitle(customer),
    subtitle: asset ? assetTitle(asset) : "客户资料",
    identifiers: [
      customerNo === "-" ? "" : `客户 ${customerNo}`,
      customerPhone === "-" ? "" : customerPhone,
      assetNo ? `资产 ${assetNo}` : "",
    ].filter(Boolean),
    statusName: workStatusName(target),
    workflowName: displayText(flow?.workflow_name),
    stageName: displayText(flow?.stage_name, workStatusName(target)),
    ownerName: displayText(
      flow?.owner_staff_name || customer.current_owner_staff_name,
    ),
    updatedAt: formatWorkDate(
      target.last_operated_at || target.updated_at || customer.created_at,
    ),
    flowStatus: textValue(flow?.status),
    flowStatusName: workFlowStatusName(flow?.status),
    stageDays: textValue(target.stage_days),
  };
}

function workFlowStatusName(status: unknown): string {
  switch (textValue(status)) {
    case "active":
      return "进行中";
    case "completed":
      return "已完成";
    case "terminated":
      return "已终止";
    case "not_started":
      return "未开始";
    default:
      return "未开始";
  }
}

type WorkOperationScope = "all" | "mine";

export function WorkCustomerOperationTimeline({
  operations,
  loading,
  scope,
  onScopeChange,
  store,
  variant,
  currentState,
  loadingText,
  emptyText,
}: {
  operations: WorkOperation[];
  loading: boolean;
  scope: WorkOperationScope;
  onScopeChange: (scope: WorkOperationScope) => void;
  store?: StoreLike;
  variant?: WorkCustomerFlowTimelineVariant;
  currentState?: WorkCustomerFlowCurrentState;
  loadingText?: string;
  emptyText?: string;
}) {
  const filteredOperations =
    scope === "mine"
      ? operations.filter((operation) => operation.operator_is_current)
      : operations;
  const sortedOperations = [...filteredOperations].sort(
    (left, right) =>
      workOperationTimeValue(right) - workOperationTimeValue(left),
  );
  const entries = sortedOperations.map(workCustomerFlowEntryView);
  return (
    <WorkCustomerFlowTimeline
      entries={entries}
      loading={loading}
      scope={scope}
      onScopeChange={onScopeChange}
      onOpen={(entry) => openWorkRecordDetail(entry.operation, store)}
      variant={variant}
      currentState={currentState}
      loadingText={loadingText}
      emptyText={emptyText}
    />
  );
}

function workCustomerFlowEntryView(
  operation: WorkOperation,
  index: number,
): WorkCustomerFlowEntryView {
  const tone = workOperationTone(operation);
  return {
    id: workOperationTimelineKey(operation, index),
    title: workOperationTitle(operation),
    description: workOperationDescription(operation),
    badge: workOperationBadgeText(operation),
    badgeClassName: tone.badge,
    dotClassName: tone.dot,
    stageName: workOperationStageLabel(operation),
    operatorName: displayText(
      operation.operator_name || operation["operator_staff.name"],
      "系统",
    ),
    time: formatWorkDate(operation.created_at || operation.create_time),
    operation,
  };
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

function workOperationBusinessEvent(operation: WorkOperation): string {
  return textValue(operation.business_event || operation.result_value);
}

function workOperationBadgeText(operation: WorkOperation): string {
  const businessEvent = workOperationBusinessEvent(operation);
  if (businessEvent === "lead_created") return "线索";
  if (businessEvent === "lead_converted") return "转化";
  if (businessEvent.startsWith("communication_group_")) return "沟通群";
  const resultValue = textValue(operation.result_value);
  if (resultValue === "progress") return "进度";
  const taskType = textValue(
    operation.task_type || operation["task.task_type"],
  );
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
  const businessEvent = workOperationBusinessEvent(operation);
  if (businessEvent === "lead_created") return "已新增";
  if (businessEvent === "lead_converted") return "已转化";
  if (businessEvent === "communication_group_created") return "已建群";
  if (businessEvent === "communication_group_updated") return "已更新";
  if (businessEvent === "communication_group_dissolved") return "已解散";
  const resultName = textValue(
    operation.result_value_name || operation.result_value_display,
  );
  if (resultName) return resultName;

  switch (textValue(operation.result_value)) {
    case "progress":
      return "保存进度";
    case "completed":
      return "已完成";
    case "terminated":
      return "已终止";
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
  if (resultValue === "communication_group_dissolved") {
    return {
      badge: "bg-muted text-muted-foreground",
      border: "border-border/60",
      dot: "bg-muted-foreground/70",
    };
  }
  if (
    resultValue === "communication_group_created" ||
    resultValue === "communication_group_updated"
  ) {
    return {
      badge: "bg-emerald-50 text-emerald-700",
      border: "border-emerald-200/80",
      dot: "bg-emerald-500",
    };
  }
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
  const taskType = textValue(
    operation.task_type || operation["task.task_type"],
  );
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
  const taskFlow = workStoreValue<WorkFlowDetail | null>(
    store,
    "data.actionTarget.workTaskFlow",
    null,
  );
  const [submitting, setSubmitting] = useState(false);
  const [completeConfirmOpen, setCompleteConfirmOpen] = useState(false);
  const [ownerDialogOpen, setOwnerDialogOpen] = useState(false);
  const [aiFilling, setAiFilling] = useState(false);
  const [aiPromptOpen, setAiPromptOpen] = useState(false);
  const [aiInstruction, setAiInstruction] = useState("");
  const uploadPending = useWorkTaskStoreValue<Record<string, boolean>>(
    store,
    workTaskUploadPendingPath,
    emptyWorkTaskRecord,
  );
  const taskFormValues = useWorkTaskStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    emptyWorkTaskRecord,
  );
  const taskFieldMap = useWorkTaskStoreValue<Record<string, string>>(
    store,
    workTaskFieldMapPath,
    emptyWorkTaskRecord,
  );
  const hasUploadingFiles = Object.values(uploadPending).some(Boolean);
  const contentRef = useRef<HTMLDivElement | null>(null);
  const canSaveProgress = task ? workTaskAllowsProgress(task) : false;
  const canCompleteDirectly = task ? workTaskNeedsCompleteAction(task) : false;
  const requiresArrivalConfirmation = task
    ? workTaskRequiresArrivalConfirmation(task)
    : false;
  const meetingDecision = workTaskRawFieldText(
    taskFormValues,
    taskFieldMap,
    workMeetingArrivalRawKey,
  );
  const meetingNoShow = meetingDecision === "no_show";
  const meetingActionLabel = meetingNoShow ? "确认未到访" : "确认到访";
  const canAIFill = task ? workTaskCanAIFill(task) : false;
  const aiHeaderTarget = useWorkFeedbackModalHeaderTarget(
    contentRef,
    canAIFill,
  );
  const footerTargets = useWorkFeedbackModalFooterTargets(
    contentRef,
    canCompleteDirectly,
    canCompleteDirectly,
  );

  const close = useCallback(() => {
    setWorkModalOpen(store, "dialog.workTask", false);
  }, [store]);

  const validateSubmit = useCallback(
    (mode: "complete" | "progress") => {
      if (workTaskHasPendingUploads(store)) {
        toast.info("附件正在上传，请稍候");
        return false;
      }
      clearCurrentWorkTaskFormErrors(store);
      clearWorkTaskCommunicationGroupError(store);
      const noShow =
        mode === "complete" &&
        currentWorkTaskMeetingDecision(store) === "no_show";
      const skipRejectedForm =
        mode === "complete" &&
        Boolean(
          task &&
            workTaskRejectSkipsConfiguredForm(
              task,
              currentWorkTaskApprovalDecision(store),
            ),
        );
      const requiredRawKeys = skipRejectedForm
        ? workApprovalActionRawKeys
        : task && workTaskHasMeeting(task)
          ? mode === "progress"
            ? workMeetingReservationRawKeys
            : noShow
              ? workMeetingNoShowRawKeys
              : undefined
          : undefined;
      if (
        !validateCurrentWorkTaskForm(store, {
          allowMissingRequired: mode === "progress" || noShow || skipRejectedForm,
          requiredRawKeys,
        })
      ) {
        focusFirstWorkTaskFormError(store);
        return false;
      }
      if (!validateWorkTaskCommunicationGroup(store)) return false;
      return true;
    },
    [store, task],
  );

  const submit = useCallback(
    async (
      mode: "complete" | "progress" = "complete",
      nextOwnerStaffID = "",
      validated = false,
    ) => {
      if (!task || submitting) return false;
      if (!validated && !validateSubmit(mode)) return false;
      setSubmitting(true);
      try {
        const communicationGroup = collectWorkTaskCommunicationGroup(store);
        const execution = await workApi<{ kept_pending?: boolean }>(
          "/crm/work/execute",
          {
            method: "POST",
            body: JSON.stringify({
              task_id: task.id,
              todo_id: positiveTextID(task.todo_id) || undefined,
              workflow_instance_id:
                positiveTextID(task.workflow_instance_id) || undefined,
              customer_id: workCustomerID(customer),
              asset_id: workAssetID(asset),
              submit_mode: mode,
              next_owner_staff_id: nextOwnerStaffID || undefined,
              values: {
                ...collectWorkTaskSubmitValues(store),
                ...(communicationGroup
                  ? { communication_group: communicationGroup }
                  : {}),
                submit_mode: mode,
              },
            }),
          },
        );
        toast.success(
          execution.kept_pending
            ? "已记录未到访，可重新预约"
            : nextOwnerStaffID
              ? "任务已完成并流转"
              : workTaskSubmitSuccessMessage(task, mode),
        );
        notifyWorkRefresh();
        setOwnerDialogOpen(false);
        close();
      } catch (error) {
        const message = errorMessage(error);
        const handled =
          applyWorkTaskCommunicationGroupError(store, message) ||
          applyWorkTaskSubmitError(store, message);
        if (nextOwnerStaffID || !handled) {
          toast.error(message || "保存失败");
        }
        return false;
      } finally {
        setSubmitting(false);
      }
      return true;
    },
    [asset, close, customer, store, submitting, task, validateSubmit],
  );

  const requestComplete = useCallback(() => {
    if (!task || submitting || !validateSubmit("complete")) return;
    const noShow = currentWorkTaskMeetingDecision(store) === "no_show";
    if (!noShow && workTaskNeedsNextStageOwner(task, taskFlow)) {
      setAiPromptOpen(false);
      setOwnerDialogOpen(true);
      return;
    }
    setAiPromptOpen(false);
    setCompleteConfirmOpen(true);
  }, [store, submitting, task, taskFlow, validateSubmit]);

  const confirmComplete = useCallback(async () => {
    const completed = await submit("complete", "", true);
    if (completed) setCompleteConfirmOpen(false);
  }, [submit]);

  const aiFill = useCallback(
    async (instruction = "") => {
      if (!task || aiFilling || submitting || workTaskHasPendingUploads(store))
        return;

      setAiFilling(true);
      try {
        const payload = await workApi<WorkAIFillResponse>("/crm/work/ai_fill", {
          method: "POST",
          body: JSON.stringify({
            task_id: task.id,
            todo_id: positiveTextID(task.todo_id) || undefined,
            workflow_instance_id:
              positiveTextID(task.workflow_instance_id) || undefined,
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
    },
    [aiFilling, asset, customer, store, submitting, task],
  );

  useEffect(() => {
    const form = contentRef.current?.closest("form");
    if (!form) return undefined;

    const handleSubmit = (event: Event) => {
      event.preventDefault();
      event.stopPropagation();
      requestComplete();
    };

    form.addEventListener("submit", handleSubmit);
    return () => {
      form.removeEventListener("submit", handleSubmit);
    };
  }, [requestComplete]);

  if (!task) return null;
  const aiFillControl = canAIFill ? (
    <div className="relative -mt-2 size-8 shrink-0 self-start">
      <Button
        type="button"
        variant="ghost"
        size="icon"
        title="AI 填写"
        className="size-8"
        onClick={() => setAiPromptOpen((open) => !open)}
        disabled={aiFilling || submitting || hasUploadingFiles}
      >
        {aiFilling ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <Sparkles className="size-4" />
        )}
        <span className="sr-only">AI 填写</span>
      </Button>
      {aiPromptOpen ? (
        <div className="absolute right-0 top-10 z-50 w-80 rounded-md border bg-background p-3 shadow-lg">
          <Textarea
            className="min-h-20 resize-none"
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
              disabled={aiFilling || hasUploadingFiles}
            >
              取消
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={() => void aiFill(aiInstruction)}
              disabled={aiFilling || submitting || hasUploadingFiles}
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
        onClick={() => void submit("progress")}
        disabled={
          submitting ||
          aiFilling ||
          hasUploadingFiles ||
          (requiresArrivalConfirmation && Boolean(meetingDecision))
        }
      >
        {requiresArrivalConfirmation ? "保存预约" : "保存进度"}
      </Button>
      <Button
        type="button"
        onClick={requestComplete}
        disabled={submitting || aiFilling || hasUploadingFiles}
      >
        {requiresArrivalConfirmation ? meetingActionLabel : "确认完成"}
      </Button>
    </>
  ) : canCompleteDirectly ? (
    <Button
      type="button"
      onClick={requestComplete}
      disabled={submitting || aiFilling || hasUploadingFiles}
    >
      {workTaskIsApproval(task) ? "提交审核" : "完成任务"}
    </Button>
  ) : null;

  return (
    <div ref={contentRef} className="contents">
      {aiHeaderTarget && aiFillControl
        ? createPortal(aiFillControl, aiHeaderTarget)
        : null}
      {footerTargets?.actions
        ? createPortal(manualActionButtons, footerTargets.actions)
        : null}
      {canCompleteDirectly && !footerTargets ? (
        <div className="mt-4 flex items-center justify-between gap-3 border-t pt-4">
          <div />
          <div className="flex items-center gap-2">{manualActionButtons}</div>
        </div>
      ) : null}
      {submitting ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          正在保存
        </div>
      ) : null}
      <WorkFlowOwnerDialog
        flow={taskFlow}
        open={ownerDialogOpen}
        title="分配下一阶段负责人"
        confirmLabel={requiresArrivalConfirmation ? meetingActionLabel : "确认完成"}
        target="next_stage"
        defaultToCurrentOwner={false}
        onConfirmSelection={(staffID) => submit("complete", staffID, true)}
        onOpenChange={setOwnerDialogOpen}
      />
      <ConfirmDialog
        open={completeConfirmOpen}
        onOpenChange={(open) => !submitting && setCompleteConfirmOpen(open)}
        title={
          requiresArrivalConfirmation
            ? meetingNoShow
              ? "确认客户未到访"
              : "确认客户到访"
            : "确认完成"
        }
        desc={
          requiresArrivalConfirmation
            ? meetingNoShow
              ? "确认客户本次未到访吗？记录后任务保留，可重新预约。"
              : "确认客户已经到访吗？确认后流程将进入下一环节。"
            : "确认完成当前任务吗？确认后将提交当前填写内容。"
        }
        confirmText={requiresArrivalConfirmation ? meetingActionLabel : "确认完成"}
        isLoading={submitting}
        handleConfirm={() => void confirmComplete()}
        className="sm:max-w-md"
      />
    </div>
  );
}

function workTaskNeedsNextStageOwner(
  task: WorkTask,
  flow: WorkFlowDetail | null,
): boolean {
  if (!flow?.next_owner_required || flow.next_terminal) return false;
  const pendingRequired = Number(flow.pending_required_count) || 0;
  const taskRequired = Boolean(task.todo_required ?? task.required);
  return pendingRequired === 0 || (taskRequired && pendingRequired === 1);
}

function workTaskHasPendingUploads(store: StoreLike | undefined): boolean {
  return Object.values(
    workStoreValue<Record<string, boolean>>(
      store,
      workTaskUploadPendingPath,
      {},
    ),
  ).some(Boolean);
}

const workMeetingReservationRawKeys = new Set([
  "meeting:start_at",
  "meeting:duration_minutes",
  "meeting:resource_id",
]);
const workMeetingArrivalRawKey = "meeting:arrival_status";
const workMeetingNoShowReasonRawKey = "meeting:no_show_reason";
const workMeetingNoShowRawKeys = new Set([
  ...workMeetingReservationRawKeys,
  workMeetingArrivalRawKey,
  workMeetingNoShowReasonRawKey,
]);
const workApprovalActionRawKeys = new Set(["approval_result", "opinion"]);

function validateCurrentWorkTaskForm(
  store: StoreLike | undefined,
  options: {
    allowMissingRequired?: boolean;
    requiredRawKeys?: Set<string>;
  } = {},
): boolean {
  if (options.allowMissingRequired) {
    return options.requiredRawKeys
      ? validateCurrentWorkTaskDomainRules(store, {
          requiredRawKeys: options.requiredRawKeys,
        })
      : true;
  }
  const validateForm = currentWorkStoreState(store)?.validateForm;
  const hostFormValid =
    typeof validateForm !== "function" ? true : validateForm();
  const taskFormValid = validateCurrentWorkTaskDomainRules(store);
  return hostFormValid && taskFormValid;
}

function validateCurrentWorkTaskDomainRules(
  store: StoreLike | undefined,
  options: {
    requiredRawKeys?: Set<string>;
  } = {},
): boolean {
  const errors = currentWorkTaskRequiredErrors(
    store,
    options.requiredRawKeys,
  );
  if (Object.keys(errors).length > 0) {
    setCurrentWorkTaskFormErrors(store, errors);
    return false;
  }
  return true;
}

function currentWorkTaskRequiredErrors(
  store: StoreLike | undefined,
  requiredRawKeys?: Set<string>,
): Record<string, string> {
  const fields = workStoreValue<WorkTaskFormField[]>(
    store,
    workTaskFormFieldsPath,
    [],
  );
  const values = workStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    {},
  );
  const fieldMap = workStoreValue<Record<string, string>>(
    store,
    workTaskFieldMapPath,
    {},
  );
  const errors: Record<string, string> = {};
  for (const field of fields) {
    if (!workTaskFormFieldRequired(field, values, fieldMap)) continue;
    if (
      requiredRawKeys &&
      !requiredRawKeys.has(fieldMap[field.formKey] || "")
    ) {
      continue;
    }
    if (!workTaskFormFieldVisible(field, values, fieldMap, fields)) continue;
    if (!workTaskFormValueEmpty(values[field.formKey])) continue;
    errors[`workTaskForm.${field.formKey}`] = `${field.label}不能为空。`;
  }
  return errors;
}

function currentWorkTaskMeetingDecision(store: StoreLike | undefined): string {
  const values = workStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    {},
  );
  const fieldMap = workStoreValue<Record<string, string>>(
    store,
    workTaskFieldMapPath,
    {},
  );
  return workTaskRawFieldText(values, fieldMap, workMeetingArrivalRawKey);
}

function currentWorkTaskApprovalDecision(store: StoreLike | undefined): string {
  const values = workStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    {},
  );
  const fieldMap = workStoreValue<Record<string, string>>(
    store,
    workTaskFieldMapPath,
    {},
  );
  return workTaskRawFieldText(values, fieldMap, "approval_result");
}

function workTaskRawFieldText(
  values: Record<string, unknown>,
  fieldMap: Record<string, string>,
  rawKey: string,
): string {
  const formKey = Object.entries(fieldMap).find(
    ([, currentRawKey]) => currentRawKey === rawKey,
  )?.[0];
  return formKey ? textValue(values[formKey]) : "";
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
  const fields = workStoreValue<WorkTaskFormField[]>(
    store,
    workTaskFormFieldsPath,
    [],
  );
  const fieldsByFormKey = new Map(
    fields.map((field) => [field.formKey, field]),
  );
  return Object.entries(fieldMap).reduce<Record<string, unknown>>(
    (values, [formKey, rawKey]) => {
      const field = fieldsByFormKey.get(formKey);
      if (
        field &&
        !workTaskFormFieldVisible(field, formValues, fieldMap, fields)
      ) {
        return values;
      }
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
  if (!workTaskIsForm(task) || !workTaskShouldRenderFields(task)) return false;
  return (task.form?.fields || []).some(
    (field) => !workTaskFieldIsUpload(field),
  );
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
  setWorkStoreValue(store, workTaskValidationErrorsPath, formErrors);
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
  if (message.includes("产品") || message.includes("product"))
    return "product_ids";
  if (message.includes("客户标签") || message.includes("标签")) return "tags";
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
  renderConfig?: WorkTaskFieldRenderConfig,
): unknown {
  const rawValue = workEntityFieldValue(field, customer, asset);
  if (renderConfig?.type === "show-crm-work-task-upload") return rawValue;

  const options = Array.isArray(field.options) ? field.options : [];
  if (renderConfig?.meta?.["multiple"]) {
    const values = Array.isArray(rawValue)
      ? rawValue
      : textValue(rawValue)
          .split(",")
          .map((value) => value.trim())
          .filter(Boolean);
    const resolveOptionID =
      renderConfig.type === "show-crm-work-customer-tags"
        ? workCustomerTagOptionID
        : workFieldOptionID;
    return values.map((value) => resolveOptionID(options, value));
  }

  const value = formatFormValue(rawValue);
  if (!value || options.length === 0) return value;

  return workFieldOptionID(options, value);
}

function workCustomerTagOptionID(
  options: WorkFieldOption[],
  rawValue: unknown,
): string {
  const value = textValue(rawValue);
  if (!value || options.length === 0) return value;

  const exactOption = options.find((option) => workOptionID(option) === value);
  if (exactOption) return workOptionID(exactOption);

  const matchingOptions = options.filter(
    (option) =>
      workOptionValue(option) === value || workOptionLabel(option) === value,
  );
  return matchingOptions.length === 1
    ? workOptionID(matchingOptions[0])
    : value;
}

function workFieldOptionID(
  options: WorkFieldOption[],
  rawValue: unknown,
): string {
  const value = textValue(rawValue);
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
  tags: ["tag_ids", "tags"],
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
    const keys =
      mainField === "tags"
        ? workMainFieldAliases.tags
        : [mainField, ...(workMainFieldAliases[mainField] || [])];
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

function workEntityFileValue(
  field: WorkFormField,
  customer?: WorkCustomer | null,
  asset?: WorkAsset | null,
): unknown {
  const key = workFieldKey(field);
  if (!key) return [];
  if (asset?.data_file_values?.[key] !== undefined) {
    return asset.data_file_values[key];
  }
  if (customer?.data_file_values?.[key] !== undefined) {
    return customer.data_file_values[key];
  }
  return [];
}
