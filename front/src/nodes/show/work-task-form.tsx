import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
} from "react";
import type { ReactElement } from "react";
import { Check } from "lucide-react";

import {
  displayText,
  inputClassName,
  setWorkStoreValue,
  textValue,
  workIsRecord,
  workStoreValue,
  workTaskFormDataPath,
  workTaskFormKey,
  workTaskValidationErrorsPath,
  type WorkCommonOption,
  type WorkNodeProps,
  type WorkStoreLike,
  type WorkTaskFormField,
  type WorkTaskFormGroup,
  type WorkTaskFormNode,
  type WorkTaskFormSection,
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

      @media (max-width: 767px) {
        .crm-work-task-field-grid,
        .crm-work-task-multi-options {
          grid-template-columns: minmax(0, 1fr);
        }
      }
    `}</style>
  );
}

export function ShowCrmWorkTaskGroupTabs({ item, store }: WorkNodeProps) {
  const rawTabs = item?.meta?.["tabs"];
  const tabs = useMemo(() => normalizeWorkTaskFormGroups(rawTabs), [rawTabs]);
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
              onClick={() => setActiveTabID(tab.id)}
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
    .map((group) => ({
      id:
        textValue(group["id"]) ||
        workTaskFormKey(textValue(group["label"])),
      label: textValue(group["label"]) || "分组",
      fields: normalizeWorkTaskFormFields(group["fields"]),
    }))
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
