import { useEffect, useMemo, useRef } from "react";
import { AlertCircle } from "lucide-react";

import { Button } from "@/components/ui/button";

import {
  setWorkStoreValue,
  textValue,
  workIsRecord,
  workStoreValue,
  workTaskActiveGroupPath,
  workTaskFormDataPath,
  workTaskFormFieldsPath,
  workTaskFormKey,
  workTaskLayoutPath,
  workTaskValidationErrorsPath,
  type WorkCommonOption,
  type WorkNodeProps,
  type WorkStoreLike,
  type WorkTask,
  type WorkTaskFormField,
  type WorkTaskFormGroup,
  type WorkTaskFormNode,
  type WorkTaskLayoutMode,
} from "./work-core";
import {
  emptyWorkTaskFields,
  emptyWorkTaskRecord,
  WorkTaskFieldGrid,
  useWorkTaskStoreValue,
} from "./work-task-form-fields";

export function WorkTaskFormStyles() {
  return (
    <style>{`
      .crm-work-task-field-grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 16px;
      }

      .crm-work-task-field[data-work-full-width="true"] {
        grid-column: 1 / -1;
      }

      .crm-work-task-multi-options {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }

      .crm-work-task-tab-list {
        scrollbar-width: thin;
      }

      @media (min-width: 768px) {
        .crm-work-task-group-tabs[data-work-tab-layout="sidebar"] {
          grid-template-columns: 13.5rem minmax(0, 1fr);
          align-items: start;
          gap: 24px;
        }

        .crm-work-task-group-tabs[data-work-tab-layout="sidebar"]
          .crm-work-task-tab-list {
          position: sticky;
          top: 0;
          max-height: min(58vh, 32rem);
          flex-direction: column;
          gap: 4px;
          overflow-x: hidden;
          overflow-y: auto;
          border-right-width: 1px;
          border-bottom: 0;
          padding: 4px 12px 4px 0;
          scrollbar-gutter: stable;
        }

        .crm-work-task-group-tabs[data-work-tab-layout="sidebar"]
          .crm-work-task-tab-button {
          width: 100%;
          justify-content: space-between;
          border-bottom-width: 0;
          border-left-width: 2px;
          border-radius: 6px;
          padding: 10px 12px;
          text-align: left;
        }

        .crm-work-task-group-tabs[data-work-tab-layout="sidebar"]
          .crm-work-task-tab-label {
          min-width: 0;
          flex: 1 1 auto;
          overflow: hidden;
          text-overflow: ellipsis;
        }
      }

      .crm-work-task-modal-body:has(> [data-crm-work-task-layout]) {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 16px;
      }

      .crm-work-task-date-field {
        margin-bottom: 0;
      }

      .crm-work-task-date-field > button {
        min-height: 2.5rem;
      }

      .crm-work-task-group-tabs,
      .crm-work-task-context-result,
      .crm-work-task-validation-summary,
      .crm-work-task-date-field[data-work-full-width="true"] {
        grid-column: 1 / -1;
      }

      [role="dialog"]:has([data-crm-work-task-layout="workspace"]) {
        width: min(1040px, calc(100vw - 32px)) !important;
        max-width: min(1040px, calc(100vw - 32px)) !important;
        max-height: calc(100vh - 32px);
      }

      [role="dialog"]:has([data-crm-work-task-layout="compact"]) {
        width: min(42rem, calc(100vw - 32px)) !important;
        max-width: min(42rem, calc(100vw - 32px)) !important;
      }

      .crm-work-task-scroll-area {
        overscroll-behavior: contain;
      }

      @media (max-width: 767px) {
        .crm-work-task-modal-body:has(> [data-crm-work-task-layout]),
        .crm-work-task-field-grid,
        .crm-work-task-multi-options {
          grid-template-columns: minmax(0, 1fr);
        }
      }
    `}</style>
  );
}

export function ShowCrmWorkTaskContext({ store }: WorkNodeProps) {
  const task = useWorkTaskStoreValue<WorkTask | null>(
    store,
    "data.actionTarget.workTask",
    null,
  );
  const layout = useWorkTaskStoreValue<WorkTaskLayoutMode>(
    store,
    workTaskLayoutPath,
    "compact",
  );
  const fields = useWorkTaskStoreValue<WorkTaskFormField[]>(
    store,
    workTaskFormFieldsPath,
    emptyWorkTaskFields,
  );
  const errors = useWorkTaskStoreValue<Record<string, string>>(
    store,
    workTaskValidationErrorsPath,
    emptyWorkTaskRecord,
  );
  const errorFields = fields.filter(
    (field) => errors[`workTaskForm.${field.formKey}`],
  );
  const contextResult = textValue(task?.context_result);
  const contextResultLabel =
    textValue(task?.context_result_label) || "自动核验结果";
  return (
    <>
      <WorkTaskFormStyles />
      <span data-crm-work-task-layout={layout} className="hidden" aria-hidden />
      {contextResult ? (
        <section className="crm-work-task-context-result rounded-md border border-border bg-muted/25 px-4 py-3">
          <div className="text-xs text-muted-foreground">
            {contextResultLabel}
          </div>
          <p className="mt-1 whitespace-pre-wrap text-sm font-medium leading-6 text-foreground">
            {contextResult}
          </p>
        </section>
      ) : null}
      <WorkTaskValidationSummary store={store} errorFields={errorFields} />
    </>
  );
}

function WorkTaskValidationSummary({
  store,
  errorFields,
}: {
  store?: WorkStoreLike;
  errorFields: WorkTaskFormField[];
}) {
  if (errorFields.length === 0) return null;
  return (
    <div
      className="crm-work-task-validation-summary mt-3 rounded-md border border-destructive/40 bg-destructive/5 px-3 py-2.5"
      role="alert"
    >
      <div className="flex items-center gap-2 text-sm font-medium text-destructive">
        <AlertCircle className="h-4 w-4" />
        <span>请补充 {errorFields.length} 个必填项</span>
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        {errorFields.map((field) => (
          <Button
            key={field.formKey}
            type="button"
            variant="outline"
            size="sm"
            className="h-auto border-destructive/30 px-2 py-1 text-xs text-destructive hover:bg-destructive/10 hover:text-destructive"
            onClick={() => focusWorkTaskFormField(store, field)}
          >
            {field.label}
          </Button>
        ))}
      </div>
    </div>
  );
}

function workTaskGroupFilledCount(
  group: WorkTaskFormGroup,
  values: Record<string, unknown>,
): number {
  return group.fields.filter(
    (field) => !workTaskFormValueEmpty(values[field.formKey]),
  ).length;
}

export function workTaskFormValueEmpty(value: unknown): boolean {
  if (value === null || value === undefined || value === "") return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "object") return Object.keys(value).length === 0;
  return textValue(value) === "";
}

export function focusFirstWorkTaskFormError(
  store: WorkStoreLike | undefined,
) {
  const fields = workStoreValue<WorkTaskFormField[]>(
    store,
    workTaskFormFieldsPath,
    emptyWorkTaskFields,
  );
  const errors = workStoreValue<Record<string, string>>(
    store,
    workTaskValidationErrorsPath,
    emptyWorkTaskRecord,
  );
  const first = fields.find(
    (field) => errors[`workTaskForm.${field.formKey}`],
  );
  if (first) focusWorkTaskFormField(store, first);
}

function focusWorkTaskFormField(
  store: WorkStoreLike | undefined,
  field: WorkTaskFormField,
) {
  if (field.groupId) {
    setWorkStoreValue(store, workTaskActiveGroupPath, field.groupId);
  }
  if (typeof document === "undefined") return;
  window.requestAnimationFrame(() => {
    window.requestAnimationFrame(() => {
      const root = Array.from(
        document.querySelectorAll<HTMLElement>("[data-work-form-key]"),
      ).find((element) => element.dataset.workFormKey === field.formKey);
      if (!root) return;
      root.scrollIntoView({ behavior: "smooth", block: "center" });
      const control = root.querySelector<HTMLElement>(
        "input:not([disabled]), textarea:not([disabled]), select:not([disabled]), button:not([disabled])",
      );
      (control || root).focus({ preventScroll: true });
    });
  });
}

export function ShowCrmWorkTaskGroupTabs({ item, store }: WorkNodeProps) {
  const rawTabs = item?.meta?.["tabs"];
  const tabs = useMemo(() => normalizeWorkTaskFormGroups(rawTabs), [rawTabs]);
  const tabListRef = useRef<HTMLDivElement>(null);
  const requestedTabID = useWorkTaskStoreValue<string>(
    store,
    workTaskActiveGroupPath,
    "",
  );
  const values = useWorkTaskStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    emptyWorkTaskRecord,
  );
  const errors = useWorkTaskStoreValue<Record<string, string>>(
    store,
    workTaskValidationErrorsPath,
    emptyWorkTaskRecord,
  );
  const activeTabID = tabs.some((tab) => tab.id === requestedTabID)
    ? requestedTabID
    : tabs[0]?.id || "";
  const useSidebarNavigation = tabs.length > 4;

  useEffect(() => {
    if (activeTabID && requestedTabID !== activeTabID) {
      setWorkStoreValue(store, workTaskActiveGroupPath, activeTabID);
    }
  }, [activeTabID, requestedTabID, store]);

  useEffect(() => {
    const activeTabElement = tabListRef.current?.querySelector<HTMLElement>(
      '[role="tab"][aria-selected="true"]',
    );
    activeTabElement?.scrollIntoView({ block: "nearest", inline: "nearest" });
  }, [activeTabID]);

  if (tabs.length === 0) return null;
  const activeTab = tabs.find((tab) => tab.id === activeTabID) || tabs[0];
  const customFields = activeTab.fields.filter(
    (field) => field.type !== "form-date",
  );

  return (
    <section
      className="crm-work-task-group-tabs grid min-w-0 gap-4"
      data-work-tab-layout={
        useSidebarNavigation ? "sidebar" : "horizontal"
      }
    >
      {tabs.length > 1 ? (
        <div
          ref={tabListRef}
          role="tablist"
          aria-label="任务表单分类"
          className="crm-work-task-tab-list flex min-w-0 overflow-x-auto border-b border-border/70"
        >
          {tabs.map((tab) => (
            <WorkTaskTabButton
              key={tab.id}
              tab={tab}
              active={tab.id === activeTab.id}
              sidebar={useSidebarNavigation}
              values={values}
              errors={errors}
              onClick={() =>
                setWorkStoreValue(store, workTaskActiveGroupPath, tab.id)
              }
            />
          ))}
        </div>
      ) : null}
      <div className="min-w-0">
        {customFields.length > 0 ? (
          <WorkTaskFieldGrid fields={customFields} store={store} />
        ) : null}
      </div>
    </section>
  );
}

function WorkTaskTabButton({
  tab,
  active,
  sidebar,
  values,
  errors,
  onClick,
}: {
  tab: WorkTaskFormGroup;
  active: boolean;
  sidebar: boolean;
  values: Record<string, unknown>;
  errors: Record<string, string>;
  onClick: () => void;
}) {
  const errorCount = tab.fields.filter(
    (field) => errors[`workTaskForm.${field.formKey}`],
  ).length;
  return (
    <Button
      type="button"
      variant="ghost"
      role="tab"
      aria-selected={active}
      className={`crm-work-task-tab-button h-auto shrink-0 rounded-none border-b-2 px-3 py-3 text-sm font-medium ${
        active
          ? `border-primary text-foreground ${
              sidebar ? "md:bg-muted/60 md:hover:bg-muted/60" : ""
            }`
          : "border-transparent text-muted-foreground hover:bg-muted/30 hover:text-foreground"
      }`}
      onClick={onClick}
    >
      <span
        className="crm-work-task-tab-label whitespace-nowrap"
        title={tab.label}
      >
        {tab.label}
      </span>
      <span
        className={`ml-1.5 shrink-0 whitespace-nowrap text-xs font-normal ${
          errorCount > 0 ? "text-destructive" : "text-muted-foreground"
        }`}
      >
        {errorCount > 0
          ? `${errorCount}项待补充`
          : `${workTaskGroupFilledCount(tab, values)} / ${tab.fields.length}`}
      </span>
    </Button>
  );
}

export function normalizeWorkTaskFormGroups(value: unknown): WorkTaskFormGroup[] {
  if (!Array.isArray(value)) return [];
  return value
    .filter(workIsRecord)
    .map((group) => {
      const id =
        textValue(group["id"]) ||
        workTaskFormKey(textValue(group["label"]));
      return {
        id,
        label: textValue(group["label"]) || "分组",
        fields: normalizeWorkTaskFormFields(group["fields"]).map((field) => ({
          ...field,
          groupId: field.groupId || id,
        })),
      };
    })
    .filter((group) => group.id && group.fields.length > 0);
}

export function normalizeWorkTaskFormFields(
  value: unknown,
): WorkTaskFormField[] {
  if (!Array.isArray(value)) return [];
  return value
    .filter(workIsRecord)
    .map((field) => ({
      formKey: textValue(field["formKey"]),
      groupId: textValue(field["groupId"]) || undefined,
      label: textValue(field["label"]),
      placeholder: textValue(field["placeholder"]),
      required: Boolean(field["required"]),
      readonly: Boolean(field["readonly"]),
      type: textValue(field["type"]) || "form-input",
      inputType: normalizeWorkTaskInputType(field["inputType"]),
      fullWidth: Boolean(field["fullWidth"]),
      options: normalizeWorkTaskOptions(field["options"]),
      meta: workIsRecord(field["meta"]) ? field["meta"] : undefined,
    }))
    .filter((field) => field.formKey && field.label);
}

export function workTaskNodeFormFields(
  node: WorkTaskFormNode | undefined,
): WorkTaskFormField[] {
  if (node?.type === "show-crm-work-task-group-tabs") {
    return normalizeWorkTaskFormGroups(node.meta?.["tabs"]).flatMap(
      (group) => group.fields,
    );
  }
  return [];
}

export function workTaskLayoutMode(
  nodes: WorkTaskFormNode[],
): WorkTaskLayoutMode {
  const fieldCount = nodes.flatMap(workTaskNodeFormFields).length;
  const groupCount = nodes
    .filter((node) => node.type === "show-crm-work-task-group-tabs")
    .flatMap((node) => normalizeWorkTaskFormGroups(node.meta?.["tabs"]))
    .length;
  return fieldCount > 16 || groupCount > 2 ? "workspace" : "compact";
}

function normalizeWorkTaskInputType(
  value: unknown,
): WorkTaskFormField["inputType"] {
  const inputType = textValue(value);
  if (
    inputType === "number" ||
    inputType === "date" ||
    inputType === "datetime-local"
  ) {
    return inputType;
  }
  return "text";
}

function normalizeWorkTaskOptions(value: unknown): WorkCommonOption[] {
  if (!Array.isArray(value)) return [];
  return value
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
    .filter((option) => option.id && option.value);
}
