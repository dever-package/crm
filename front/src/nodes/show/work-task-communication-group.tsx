import { AlertCircle } from "lucide-react";

import {
  setWorkStoreValue,
  textValue,
  workStoreValue,
  workTaskCommunicationGroupContextPath,
  workTaskCommunicationGroupDraftPath,
  workTaskCommunicationGroupErrorPath,
  workTaskFormDataPath,
  workTaskFormFieldsPath,
  workTaskPath,
  type WorkStoreLike,
  type WorkTask,
  type WorkTaskCommunicationGroupContext,
  type WorkTaskFormField,
} from "./work-core";
import {
  CommunicationGroupForm,
  communicationGroupDraft,
  communicationGroupDraftPayload,
  useCommunicationGroupPeopleOptions,
  validateCommunicationGroupDraft,
  type CommunicationGroupDraft,
} from "./work-communication-group-form";
import {
  emptyWorkTaskFields,
  emptyWorkTaskRecord,
  useWorkTaskStoreValue,
} from "./work-task-form-fields";

const communicationGroupStatusFieldKey = "service_group_status";
const communicationGroupCreatedValue = "created";

const emptyCommunicationGroupContext: WorkTaskCommunicationGroupContext = {
  groups: [],
  groupTypes: [],
  workflowInstanceID: "",
  canManage: false,
};

const emptyCommunicationGroupDraft = communicationGroupDraft(null, [], "");

export function WorkTaskCommunicationGroupSection({
  fields,
  store,
}: {
  fields: WorkTaskFormField[];
  store?: WorkStoreLike;
}) {
  const values = useWorkTaskStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    emptyWorkTaskRecord,
  );
  const context = useWorkTaskStoreValue<WorkTaskCommunicationGroupContext>(
    store,
    workTaskCommunicationGroupContextPath,
    emptyCommunicationGroupContext,
  );
  const draft = useWorkTaskStoreValue<CommunicationGroupDraft>(
    store,
    workTaskCommunicationGroupDraftPath,
    emptyCommunicationGroupDraft,
  );
  const validationError = useWorkTaskStoreValue<string>(
    store,
    workTaskCommunicationGroupErrorPath,
    "",
  );
  const task = useWorkTaskStoreValue<WorkTask | null>(
    store,
    workTaskPath,
    null,
  );
  const visible = workTaskRequiresCommunicationGroup(
    fields,
    values,
    taskCapabilityEnabled(task),
  );
  const { peopleOptions, peopleLoading, peopleError } =
    useCommunicationGroupPeopleOptions(visible);

  if (!visible) return null;
  const activeGroup = context.groups.find((group) => group.status === "active");
  const contextError = !context.workflowInstanceID
    ? "当前任务未关联有效案件流程"
    : !context.canManage
      ? "当前人员无权维护该案件的沟通群"
      : context.groupTypes.length === 0
        ? "后台未配置可用的沟通群类型"
        : "";
  const error = validationError || peopleError || contextError;

  return (
    <section
      id="work-task-communication-group-section"
      className="col-span-full my-2 rounded-md border border-border bg-muted/10 p-5"
    >
      <div className="mb-4 flex min-h-6 items-center justify-between gap-3">
        <h3 className="text-sm font-semibold">
          {activeGroup ? "沟通群信息" : "建群信息"}
        </h3>
        {activeGroup ? (
          <span className="text-xs text-muted-foreground">编辑现有群</span>
        ) : null}
      </div>
      {error ? (
        <div
          className="mb-4 flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/5 px-3 py-2.5 text-sm text-destructive"
          role="alert"
        >
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{error}</span>
        </div>
      ) : null}
      <CommunicationGroupForm
        draft={draft}
        groupTypes={context.groupTypes}
        peopleOptions={peopleOptions}
        peopleLoading={peopleLoading}
        existingGroup={activeGroup || null}
        disabled={Boolean(contextError)}
        onChange={(nextDraft) => {
          setWorkStoreValue(
            store,
            workTaskCommunicationGroupDraftPath,
            nextDraft,
          );
          setWorkStoreValue(store, workTaskCommunicationGroupErrorPath, "");
        }}
      />
    </section>
  );
}

export function workTaskRequiresCommunicationGroup(
  fields: WorkTaskFormField[],
  values: Record<string, unknown>,
  capabilityEnabled = false,
): boolean {
  if (capabilityEnabled) return true;
  const statusField = fields.find(
    (field) =>
      textValue(field.meta?.["dataFieldKey"]) ===
      communicationGroupStatusFieldKey,
  );
  if (!statusField) return false;
  const createdOption = statusField.options?.find(
    (option) =>
      textValue(option.id) === communicationGroupCreatedValue ||
      textValue(option.value) === "已建群" ||
      textValue(option["name"]) === "已建群",
  );
  return Boolean(
    createdOption &&
      textValue(values[statusField.formKey]) === textValue(createdOption.id),
  );
}

export function validateWorkTaskCommunicationGroup(
  store?: WorkStoreLike,
): boolean {
  if (!currentWorkTaskRequiresCommunicationGroup(store)) {
    setWorkStoreValue(store, workTaskCommunicationGroupErrorPath, "");
    return true;
  }
  const context = workStoreValue<WorkTaskCommunicationGroupContext>(
    store,
    workTaskCommunicationGroupContextPath,
    emptyCommunicationGroupContext,
  );
  const draft = workStoreValue<CommunicationGroupDraft>(
    store,
    workTaskCommunicationGroupDraftPath,
    emptyCommunicationGroupDraft,
  );
  const error = !context.canManage
    ? "当前人员无权维护该案件的沟通群"
    : context.groupTypes.length === 0
      ? "后台未配置可用的沟通群类型"
      : validateCommunicationGroupDraft(draft);
  setWorkStoreValue(store, workTaskCommunicationGroupErrorPath, error);
  if (error) focusWorkTaskCommunicationGroup();
  return !error;
}

export function collectWorkTaskCommunicationGroup(
  store?: WorkStoreLike,
): Record<string, unknown> | undefined {
  if (!currentWorkTaskRequiresCommunicationGroup(store)) return undefined;
  return communicationGroupDraftPayload(
    workStoreValue<CommunicationGroupDraft>(
      store,
      workTaskCommunicationGroupDraftPath,
      emptyCommunicationGroupDraft,
    ),
  );
}

export function applyWorkTaskCommunicationGroupError(
  store: WorkStoreLike | undefined,
  message: string,
): boolean {
  if (!message.includes("沟通群") && !message.includes("建群")) return false;
  setWorkStoreValue(store, workTaskCommunicationGroupErrorPath, message);
  focusWorkTaskCommunicationGroup();
  return true;
}

export function clearWorkTaskCommunicationGroupError(store?: WorkStoreLike) {
  setWorkStoreValue(store, workTaskCommunicationGroupErrorPath, "");
}

function currentWorkTaskRequiresCommunicationGroup(
  store?: WorkStoreLike,
): boolean {
  const task = workStoreValue<WorkTask | null>(store, workTaskPath, null);
  return workTaskRequiresCommunicationGroup(
    workStoreValue<WorkTaskFormField[]>(
      store,
      workTaskFormFieldsPath,
      emptyWorkTaskFields,
    ),
    workStoreValue<Record<string, unknown>>(
      store,
      workTaskFormDataPath,
      emptyWorkTaskRecord,
    ),
    taskCapabilityEnabled(task),
  );
}

function taskCapabilityEnabled(task?: WorkTask | null): boolean {
  const value = task?.communication_group_enabled;
  return value === true || value === 1 || value === "1" || value === "true";
}

function focusWorkTaskCommunicationGroup() {
  if (typeof document === "undefined") return;
  window.requestAnimationFrame(() => {
    const section = document.getElementById(
      "work-task-communication-group-section",
    );
    section?.scrollIntoView({ behavior: "smooth", block: "center" });
    section?.querySelector<HTMLElement>("input, button")?.focus({
      preventScroll: true,
    });
  });
}
