import {
  useCallback,
  useEffect,
  useMemo,
  useSyncExternalStore,
} from "react";
import type { ReactElement } from "react";
import { AlertCircle, Check } from "lucide-react";

import {
  displayText,
  inputClassName,
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
  type WorkAsset,
  type WorkCommonOption,
  type WorkCustomer,
  type WorkNodeProps,
  type WorkStoreLike,
  type WorkTask,
  type WorkTaskFormField,
  type WorkTaskFormGroup,
  type WorkTaskFormNode,
  type WorkTaskFormSection,
  type WorkTaskLayoutMode,
} from "./work-core";
import { ShowCrmWorkTaskUpload } from "./work-upload";

type TaskFieldRendererProps = {
  field: WorkTaskFormField;
  value: unknown;
  setValue: (value: unknown) => void;
  store?: WorkStoreLike;
  error?: string;
};

type TaskFieldRenderer = (props: TaskFieldRendererProps) => ReactElement;

const taskFieldRenderers: Record<string, TaskFieldRenderer> = {
  "form-input": renderTaskInput,
  "form-textarea": renderTaskTextarea,
  "form-select": renderTaskSelect,
  "form-switch": renderTaskBoolean,
  "show-crm-work-task-upload": renderTaskUpload,
};

const emptyWorkTaskRecord: Record<string, never> = {};
const emptyWorkTaskFields: WorkTaskFormField[] = [];

export function WorkTaskFormStyles() {
  return (
    <style>{`
      .crm-work-task-field-grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        column-gap: 20px;
        row-gap: 16px;
      }

      .crm-work-task-field[data-work-full-width="true"] {
        grid-column: 1 / -1;
      }

      .crm-work-task-multi-options {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }

      .crm-work-task-context-grid {
        display: grid;
        grid-template-columns: repeat(4, minmax(0, 1fr));
      }

      .crm-work-task-workspace-grid {
        display: grid;
        grid-template-columns: 176px minmax(0, 1fr);
        gap: 20px;
      }

      [role="dialog"]:has([data-crm-work-task-layout="workspace"]) {
        width: min(1120px, calc(100vw - 32px)) !important;
        max-width: min(1120px, calc(100vw - 32px)) !important;
        max-height: calc(100vh - 32px);
      }

      .crm-work-task-context {
        position: sticky;
        top: -8px;
        z-index: 8;
      }

      .crm-work-task-scroll-area {
        overscroll-behavior: contain;
      }

      @media (max-width: 767px) {
        .crm-work-task-field-grid,
        .crm-work-task-multi-options {
          grid-template-columns: minmax(0, 1fr);
        }

        .crm-work-task-context-grid {
          grid-template-columns: repeat(2, minmax(0, 1fr));
        }

        .crm-work-task-workspace-grid {
          grid-template-columns: minmax(0, 1fr);
          gap: 14px;
        }

        .crm-work-task-section-nav {
          display: flex;
          overflow-x: auto;
          padding-bottom: 4px;
        }

        .crm-work-task-section-nav button {
          min-width: 148px;
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
  const customer = useWorkTaskStoreValue<WorkCustomer | null>(
    store,
    "data.actionTarget.workTaskCustomer",
    null,
  );
  const asset = useWorkTaskStoreValue<WorkAsset | null>(
    store,
    "data.actionTarget.workTaskAsset",
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
  const filled = fields.filter(
    (field) => !workTaskFormValueEmpty(values[field.formKey]),
  ).length;
  const errorFields = fields.filter(
    (field) => errors[`workTaskForm.${field.formKey}`],
  );

  if (!task) return null;

  return (
    <section
      data-crm-work-task-layout={layout}
      className="crm-work-task-context -mx-1 border-b border-border/70 bg-background px-1 pb-4 pt-1"
    >
      <WorkTaskFormStyles />
      <div className="flex min-w-0 flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="break-words text-base font-semibold text-foreground">
            {textValue(customer?.name || customer?.customer_name) || "未命名客户"}
          </h3>
          <p className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
            <span>{workTaskCustomerNo(customer)}</span>
            {workTaskCustomerPhone(customer) ? (
              <span>{workTaskCustomerPhone(customer)}</span>
            ) : null}
            {asset ? <span>{workTaskAssetNo(asset)}</span> : null}
          </p>
        </div>
        <span className="rounded bg-muted px-2 py-1 text-xs font-medium text-foreground">
          已填写 {filled} / {fields.length}
        </span>
      </div>

      <div className="crm-work-task-context-grid mt-3 gap-px overflow-hidden rounded-md border border-border/60 bg-border/60">
        <WorkTaskContextMetric
          label="任务"
          value={textValue(task.task_name || task.name) || "待处理任务"}
        />
        <WorkTaskContextMetric
          label="阶段"
          value={textValue(task.stage_name) || "未进入阶段"}
        />
        <WorkTaskContextMetric
          label="资产"
          value={asset ? workTaskAssetName(asset) : "未录入资产"}
        />
        <WorkTaskContextMetric
          label="负责人"
          value={
            textValue(
              task.assignee_staff_name || task.assignee_department_name,
            ) || "暂未分配"
          }
        />
      </div>

      {errorFields.length > 0 ? (
        <div
          className="mt-3 rounded-md border border-destructive/40 bg-destructive/5 px-3 py-2.5"
          role="alert"
        >
          <div className="flex items-center gap-2 text-sm font-medium text-destructive">
            <AlertCircle className="h-4 w-4" />
            <span>请补充 {errorFields.length} 个必填项</span>
          </div>
          <div className="mt-2 flex flex-wrap gap-2">
            {errorFields.map((field) => (
              <button
                key={field.formKey}
                type="button"
                className="rounded border border-destructive/30 bg-background px-2 py-1 text-xs text-destructive hover:bg-destructive/10"
                onClick={() => focusWorkTaskFormField(store, field)}
              >
                {field.label}
              </button>
            ))}
          </div>
        </div>
      ) : null}
    </section>
  );
}

function WorkTaskContextMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 bg-background px-3 py-2.5">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 truncate text-sm font-medium text-foreground">
        {displayText(value)}
      </div>
    </div>
  );
}

function workTaskCustomerNo(customer?: WorkCustomer | null): string {
  return (
    textValue(
      customer?.code_display ||
        customer?.customer_no ||
        customer?.code ||
        customer?.no,
    ) || "未生成客户编号"
  );
}

function workTaskCustomerPhone(customer?: WorkCustomer | null): string {
  return textValue(customer?.phone || customer?.mobile);
}

function workTaskAssetNo(asset?: WorkAsset | null): string {
  return (
    textValue(asset?.asset_no || asset?.asset_code || asset?.code) ||
    "未生成资产编号"
  );
}

function workTaskAssetName(asset?: WorkAsset | null): string {
  return textValue(asset?.asset_name || asset?.name) || "未命名资产";
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
  const requestedTabID = useWorkTaskStoreValue<string>(
    store,
    workTaskActiveGroupPath,
    "",
  );
  const layout = useWorkTaskStoreValue<WorkTaskLayoutMode>(
    store,
    workTaskLayoutPath,
    "compact",
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

  useEffect(() => {
    if (activeTabID && requestedTabID !== activeTabID) {
      setWorkStoreValue(store, workTaskActiveGroupPath, activeTabID);
    }
  }, [activeTabID, requestedTabID, store]);

  if (tabs.length === 0) return null;
  const activeTab = tabs.find((tab) => tab.id === activeTabID) || tabs[0];

  if (layout === "workspace") {
    return (
      <section className="border-t border-border/70 pt-5 first:border-t-0 first:pt-0">
        <div className="crm-work-task-workspace-grid">
          <nav className="crm-work-task-section-nav grid content-start gap-1">
            {tabs.map((tab) => (
              <WorkTaskGroupButton
                key={tab.id}
                tab={tab}
                active={tab.id === activeTab.id}
                values={values}
                errors={errors}
                onClick={() =>
                  setWorkStoreValue(store, workTaskActiveGroupPath, tab.id)
                }
              />
            ))}
          </nav>
          <div className="min-w-0">
            <div className="mb-4 border-b border-border/70 pb-3">
              <h3 className="text-sm font-semibold text-foreground">
                {activeTab.label}
              </h3>
              <p className="mt-1 text-xs text-muted-foreground">
                已填写 {workTaskGroupFilledCount(activeTab, values)} /{" "}
                {activeTab.fields.length}
              </p>
            </div>
            <WorkTaskFieldGrid fields={activeTab.fields} store={store} />
          </div>
        </div>
      </section>
    );
  }

  return (
    <section className="grid gap-4 border-t border-border/70 pt-5 first:border-t-0 first:pt-0">
      <div className="flex flex-wrap gap-2 border-b border-border/70 pb-3">
        {tabs.map((tab) => {
          const active = tab.id === activeTab.id;
          return (
            <button
              key={tab.id}
              type="button"
              className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                active
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground hover:text-foreground"
              }`}
              onClick={() =>
                setWorkStoreValue(store, workTaskActiveGroupPath, tab.id)
              }
            >
              {tab.label}
            </button>
          );
        })}
      </div>
      <WorkTaskFieldGrid fields={activeTab.fields} store={store} />
    </section>
  );
}

function WorkTaskGroupButton({
  tab,
  active,
  values,
  errors,
  onClick,
}: {
  tab: WorkTaskFormGroup;
  active: boolean;
  values: Record<string, unknown>;
  errors: Record<string, string>;
  onClick: () => void;
}) {
  const errorCount = tab.fields.filter(
    (field) => errors[`workTaskForm.${field.formKey}`],
  ).length;
  return (
    <button
      type="button"
      className={`rounded-md px-3 py-2.5 text-left transition-colors ${
        active
          ? "bg-muted text-foreground"
          : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
      }`}
      onClick={onClick}
    >
      <span className="block truncate text-sm font-medium">{tab.label}</span>
      <span
        className={`mt-1 block text-xs ${
          errorCount > 0 ? "text-destructive" : "opacity-75"
        }`}
      >
        {errorCount > 0
          ? `${errorCount} 项待补充`
          : `${workTaskGroupFilledCount(tab, values)} / ${tab.fields.length}`}
      </span>
    </button>
  );
}

export function ShowCrmWorkTaskFieldSection({ item, store }: WorkNodeProps) {
  const section = useMemo(
    () => normalizeWorkTaskFormSection(item?.meta),
    [item?.meta],
  );

  if (!section || section.fields.length === 0) return null;

  return (
    <section className="grid gap-4 border-t border-border/70 pt-5 first:border-t-0 first:pt-0">
      <h3 className="text-sm font-semibold text-foreground">
        {section.label}
      </h3>
      <WorkTaskFieldGrid fields={section.fields} store={store} />
    </section>
  );
}

function WorkTaskFieldGrid({
  fields,
  store,
}: {
  fields: WorkTaskFormField[];
  store?: WorkStoreLike;
}) {
  return (
    <div className="crm-work-task-field-grid">
      {fields.map((field) => (
        <WorkTaskField key={field.formKey} field={field} store={store} />
      ))}
    </div>
  );
}

function WorkTaskField({
  field,
  store,
}: {
  field: WorkTaskFormField;
  store?: WorkStoreLike;
}) {
  const formValues = useWorkTaskStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    emptyWorkTaskRecord,
  );
  const errors = useWorkTaskStoreValue<Record<string, string>>(
    store,
    "errors",
    emptyWorkTaskRecord,
  );
  const taskErrors = useWorkTaskStoreValue<Record<string, string>>(
    store,
    workTaskValidationErrorsPath,
    emptyWorkTaskRecord,
  );
  const value = formValues[field.formKey];
  const errorKey = `workTaskForm.${field.formKey}`;
  const error = taskErrors[errorKey] || errors[errorKey];
  const setValue = useCallback(
    (nextValue: unknown) => {
      if (field.readonly) return;
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
    [field.formKey, field.readonly, store],
  );
  const renderer =
    taskFieldRenderers[field.type] || taskFieldRenderers["form-input"];

  return (
    <div
      className={`crm-work-task-field min-w-0 ${
        field.readonly ? "opacity-70" : ""
      }`}
      data-work-form-key={field.formKey}
      data-work-full-width={field.fullWidth ? "true" : "false"}
      tabIndex={-1}
    >
      <div className="mb-1.5 flex min-h-5 items-center gap-1 text-sm font-medium text-foreground">
        <span>{field.label}</span>
        {field.required ? <span className="text-destructive">*</span> : null}
        {field.readonly ? (
          <span className="ml-auto text-xs font-normal text-muted-foreground">
            只读
          </span>
        ) : null}
      </div>
      {renderer({ field, value, setValue, store, error })}
      {error ? (
        <p className="mt-1.5 text-xs text-destructive">{error}</p>
      ) : null}
    </div>
  );
}

function useWorkTaskStoreValue<T>(
  store: WorkStoreLike | undefined,
  path: string,
  fallback: T,
): T {
  const subscribe = useCallback(
    (notify: () => void) => {
      const storeApi = store as
        | { subscribe?: (listener: () => void) => (() => void) | void }
        | undefined;
      const unsubscribe = storeApi?.subscribe?.(notify);
      return typeof unsubscribe === "function" ? unsubscribe : () => undefined;
    },
    [store],
  );
  const getSnapshot = useCallback(
    () => workStoreValue(store, path, fallback),
    [fallback, path, store],
  );
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}

function renderTaskInput({
  field,
  value,
  setValue,
  error,
}: TaskFieldRendererProps) {
  return (
    <input
      type={field.inputType || "text"}
      step={field.inputType === "number" ? "any" : undefined}
      className={workTaskControlClassName(error)}
      placeholder={field.placeholder}
      value={workTaskTextValue(value)}
      disabled={field.readonly}
      aria-invalid={Boolean(error)}
      onChange={(event) => setValue(event.currentTarget.value)}
    />
  );
}

function renderTaskTextarea({
  field,
  value,
  setValue,
  error,
}: TaskFieldRendererProps) {
  return (
    <textarea
      className={`${workTaskControlClassName(error)} min-h-24 resize-y py-2`}
      rows={Number(field.meta?.["rows"] || 4)}
      placeholder={field.placeholder}
      value={workTaskTextValue(value)}
      disabled={field.readonly}
      aria-invalid={Boolean(error)}
      onChange={(event) => setValue(event.currentTarget.value)}
    />
  );
}

function renderTaskSelect(props: TaskFieldRendererProps) {
  return props.field.meta?.["multiple"]
    ? renderTaskMultiSelect(props)
    : renderTaskSingleSelect(props);
}

function renderTaskSingleSelect({
  field,
  value,
  setValue,
  error,
}: TaskFieldRendererProps) {
  return (
    <select
      className={workTaskControlClassName(error)}
      value={textValue(value)}
      disabled={field.readonly}
      aria-invalid={Boolean(error)}
      onChange={(event) => setValue(event.currentTarget.value)}
    >
      <option value="">{field.placeholder}</option>
      {(field.options || []).map((option) => (
        <option key={option.id} value={option.id}>
          {displayText(option.value || option.id)}
        </option>
      ))}
    </select>
  );
}

function renderTaskMultiSelect({
  field,
  value,
  setValue,
  error,
}: TaskFieldRendererProps) {
  const selected = new Set(workTaskSelectedValues(value));
  return (
    <div
      className={`crm-work-task-multi-options gap-2 rounded-md border bg-background p-2 ${
        error ? "border-destructive" : "border-input"
      }`}
      aria-invalid={Boolean(error)}
    >
      {(field.options || []).map((option) => {
        const checked = selected.has(option.id);
        return (
          <label
            key={option.id}
            className={`flex min-w-0 items-center gap-2 rounded px-2 py-2 text-sm ${
              field.readonly
                ? "cursor-not-allowed"
                : "cursor-pointer hover:bg-muted/60"
            }`}
          >
            <input
              type="checkbox"
              className="sr-only"
              checked={checked}
              disabled={field.readonly}
              onChange={() => {
                const next = new Set(selected);
                if (checked) next.delete(option.id);
                else next.add(option.id);
                setValue(Array.from(next));
              }}
            />
            <span
              className={`inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border ${
                checked
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-input bg-background"
              }`}
            >
              {checked ? <Check className="h-3 w-3" /> : null}
            </span>
            <span className="min-w-0 break-words">
              {displayText(option.value || option.id)}
            </span>
          </label>
        );
      })}
    </div>
  );
}

function renderTaskBoolean({
  field,
  value,
  setValue,
  error,
}: TaskFieldRendererProps) {
  const selected = workTaskBooleanValue(value);
  return (
    <div
      className={`inline-flex rounded-md border bg-muted/20 p-1 ${
        error ? "border-destructive" : "border-input"
      }`}
      aria-invalid={Boolean(error)}
    >
      {[
        { value: true, label: "是" },
        { value: false, label: "否" },
      ].map((option) => {
        const active = selected === option.value;
        return (
          <button
            key={option.label}
            type="button"
            className={`min-w-16 rounded px-3 py-1.5 text-sm font-medium transition-colors ${
              active
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
            disabled={field.readonly}
            onClick={() => setValue(option.value)}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}

function renderTaskUpload({
  field,
  value,
  setValue,
  store,
}: TaskFieldRendererProps) {
  return (
    <ShowCrmWorkTaskUpload
      item={{
        id: `work-task-upload-${field.formKey}`,
        name: field.label,
        value: `workTaskForm.${field.formKey}`,
        placeholder: field.placeholder,
        meta: { ...field.meta, readonly: field.readonly },
      }}
      store={store}
      value={value}
      setValue={setValue}
    />
  );
}

function workTaskControlClassName(error?: string): string {
  return `${inputClassName} ${error ? "border-destructive focus:border-destructive" : ""}`;
}

function workTaskTextValue(value: unknown): string {
  if (value === null || value === undefined || typeof value === "object") {
    return "";
  }
  return String(value);
}

function workTaskSelectedValues(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(textValue).filter(Boolean);
  const single = textValue(value);
  if (!single) return [];
  return single
    .split(",")
    .map((part) => part.trim())
    .filter(Boolean);
}

function workTaskBooleanValue(value: unknown): boolean | null {
  if (value === true || value === 1 || value === "1" || value === "true") {
    return true;
  }
  if (value === false || value === 0 || value === "0" || value === "false") {
    return false;
  }
  return null;
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

export function normalizeWorkTaskFormSection(
  value: unknown,
): WorkTaskFormSection | null {
  if (!workIsRecord(value)) return null;
  const label =
    textValue(value["title"]) ||
    textValue(value["label"]) ||
    textValue(value["name"]);
  const fields = normalizeWorkTaskFormFields(value["fields"]);
  if (!label || fields.length === 0) return null;
  return {
    id: textValue(value["id"]) || workTaskFormKey(label),
    label,
    fields,
  };
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
  if (node?.type === "show-crm-work-task-field-section") {
    return normalizeWorkTaskFormFields(node.meta?.["fields"]);
  }
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
  return fieldCount > 6 || groupCount > 1 ? "workspace" : "compact";
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
