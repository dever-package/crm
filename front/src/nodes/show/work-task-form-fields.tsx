import { useCallback, useSyncExternalStore } from "react";
import type { ReactElement } from "react";
import { Check } from "lucide-react";

import {
  displayText,
  inputClassName,
  setWorkStoreValue,
  textValue,
  workStoreValue,
  workTaskFormDataPath,
  workTaskValidationErrorsPath,
  type WorkStoreLike,
  type WorkTaskFormField,
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

export const emptyWorkTaskRecord: Record<string, never> = {};
export const emptyWorkTaskFields: WorkTaskFormField[] = [];

export function WorkTaskFieldGrid({
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
        emptyWorkTaskRecord,
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

export function useWorkTaskStoreValue<T>(
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
  return `${inputClassName} ${
    error ? "border-destructive focus:border-destructive" : ""
  }`;
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
