import { useCallback, useState, useSyncExternalStore } from "react";
import type { ReactElement } from "react";
import { Check, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

import {
  displayText,
  setWorkStoreValue,
  textValue,
  workStoreValue,
  workTaskFormDataPath,
  workTaskValidationErrorsPath,
  type WorkStoreLike,
  type WorkTaskFormField,
} from "./work-core";
import {
  CustomerTagSelector,
  normalizeCustomerTagIDs,
  normalizeCustomerTagOptions,
} from "./customer-tag-selector";
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
  "show-crm-work-customer-tags": renderTaskCustomerTags,
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
    <Input
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
    <Textarea
      className={`${workTaskControlClassName(error)} min-h-24 resize-y`}
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

function renderTaskCustomerTags({
  field,
  value,
  setValue,
  error,
}: TaskFieldRendererProps) {
  return (
    <CustomerTagSelector
      options={normalizeCustomerTagOptions(field.options)}
      value={normalizeCustomerTagIDs(value)}
      disabled={field.readonly}
      error={error}
      onChange={setValue}
    />
  );
}

function renderTaskSingleSelect({
  field,
  value,
  setValue,
  error,
}: TaskFieldRendererProps) {
  return (
    <Select
      value={textValue(value)}
      disabled={field.readonly}
      onValueChange={setValue}
    >
      <SelectTrigger
        className={workTaskControlClassName(error)}
        aria-invalid={Boolean(error)}
      >
        <SelectValue placeholder={field.placeholder || "请选择"} />
      </SelectTrigger>
      <SelectContent position="popper">
        {(field.options || []).map((option) => (
          <SelectItem key={option.id} value={option.id}>
            {displayText(option.value || option.id)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function renderTaskMultiSelect({
  field,
  value,
  setValue,
  error,
}: TaskFieldRendererProps) {
  return (
    <TaskMultiSelect
      field={field}
      value={value}
      setValue={setValue}
      error={error}
    />
  );
}

function TaskMultiSelect({
  field,
  value,
  setValue,
  error,
}: TaskFieldRendererProps) {
  const [keyword, setKeyword] = useState("");
  const selected = new Set(workTaskSelectedValues(value));
  const normalizedKeyword = keyword.trim().toLowerCase();
  const options = (field.options || []).filter((option) => {
    if (!normalizedKeyword) return true;
    return textValue(option.value || option.id)
      .toLowerCase()
      .includes(normalizedKeyword);
  });
  return (
    <div
      className={`rounded-md border bg-background p-2 ${
        error ? "border-destructive" : "border-input"
      }`}
      aria-invalid={Boolean(error)}
    >
      {field.meta?.["searchable"] ? (
        <div className="relative mb-2">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            className="h-9 pl-8"
            placeholder={field.placeholder}
            value={keyword}
            disabled={field.readonly}
            onChange={(event) => setKeyword(event.currentTarget.value)}
          />
        </div>
      ) : null}
      <div className="crm-work-task-multi-options gap-2">
        {options.map((option) => {
          const checked = selected.has(option.id);
          const optionLabel = displayText(option.value || option.id);
          return (
            <Button
              key={option.id}
              type="button"
              variant="ghost"
              aria-pressed={checked}
              title={optionLabel}
              disabled={field.readonly}
              className={`h-auto w-full min-w-0 justify-start gap-2 px-2 py-2 text-sm font-normal ${
                field.readonly
                  ? "cursor-not-allowed"
                  : "cursor-pointer hover:bg-muted/60"
              }`}
              onClick={() => {
                const next = new Set(selected);
                if (checked) next.delete(option.id);
                else next.add(option.id);
                setValue(Array.from(next));
              }}
            >
              <span
                className={`inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border ${
                  checked
                    ? "border-primary bg-primary text-primary-foreground"
                    : "border-input bg-background"
                }`}
              >
                {checked ? <Check className="h-3 w-3" /> : null}
              </span>
              <span className="min-w-0 flex-1 truncate text-left">
                {optionLabel}
              </span>
            </Button>
          );
        })}
      </div>
      {options.length === 0 ? (
        <div className="px-2 py-5 text-center text-sm text-muted-foreground">
          没有匹配的产品
        </div>
      ) : null}
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
          <Button
            key={option.label}
            type="button"
            variant="ghost"
            aria-pressed={active}
            className={`h-auto min-w-16 px-3 py-1.5 text-sm font-medium ${
              active
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
            disabled={field.readonly}
            onClick={() => setValue(option.value)}
          >
            {option.label}
          </Button>
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
  return `w-full ${
    error
      ? "border-destructive focus-visible:ring-destructive/20"
      : ""
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
