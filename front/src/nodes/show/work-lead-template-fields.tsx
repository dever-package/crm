import { Check } from "lucide-react";

import { Input } from "@/components/ui/input";

import { displayText, inputClassName, textValue } from "./work-core";

export type WorkLeadTemplateOption = {
  id?: string | number;
  name?: string;
  value?: string;
};

export type WorkLeadTemplateField = {
  id?: string | number;
  name?: string;
  field_key?: string;
  field_type?: string;
  default_value?: unknown;
  group_name?: string;
  options?: WorkLeadTemplateOption[];
};

export type WorkLeadTemplate = {
  id?: string | number;
  name?: string;
  fields?: WorkLeadTemplateField[];
};

export function initialWorkLeadTemplateValues(
  templates: WorkLeadTemplate[],
): Record<string, unknown> {
  const values: Record<string, unknown> = {};
  templates.forEach((template) => {
    (template.fields || []).forEach((field) => {
      const key = workLeadTemplateFieldKey(field);
      if (!key) {
        return;
      }
      if (field.field_type === "boolean") {
        values[key] = workLeadTemplateDefaultValue(field);
        return;
      }
      if (field.default_value === undefined || field.default_value === "") return;
      values[key] = workLeadTemplateDefaultValue(field);
    });
  });
  return values;
}

export function WorkLeadTemplateFields({
  templates,
  values,
  onChange,
}: {
  templates: WorkLeadTemplate[];
  values: Record<string, unknown>;
  onChange: (values: Record<string, unknown>) => void;
}) {
  const fields = templates.flatMap((template) => template.fields || []);
  if (fields.length === 0) return null;
  const setFieldValue = (field: WorkLeadTemplateField, value: unknown) => {
    const key = workLeadTemplateFieldKey(field);
    if (!key) return;
    onChange({ ...values, [key]: value });
  };
  return (
    <>
      {fields.map((field) => (
        <WorkLeadTemplateFieldControl
          key={workLeadTemplateFieldKey(field)}
          field={field}
          value={values[workLeadTemplateFieldKey(field)]}
          onChange={(value) => setFieldValue(field, value)}
        />
      ))}
    </>
  );
}

function WorkLeadTemplateFieldControl({
  field,
  value,
  onChange,
}: {
  field: WorkLeadTemplateField;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const fieldType = textValue(field.field_type);
  const label = displayText(field.name, "扩展字段");
  const options = field.options || [];
  const fullWidth =
    fieldType === "textarea" ||
    fieldType === "checkbox" ||
    fieldType === "multi_select";
  return (
    <label className={fullWidth ? "sm:col-span-2" : ""}>
      <span className="mb-1.5 block text-sm font-medium">{label}</span>
      {field.group_name ? (
        <span className="mb-1 block text-xs text-muted-foreground">
          {field.group_name}
        </span>
      ) : null}
      {fieldType === "textarea" ? (
        <textarea
          className="min-h-24 w-full resize-y rounded-md border border-input bg-background px-3 py-2 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/20"
          value={workLeadTextValue(value)}
          onChange={(event) => onChange(event.currentTarget.value)}
        />
      ) : fieldType === "select" || fieldType === "radio" ? (
        <select
          className={inputClassName}
          value={textValue(value)}
          onChange={(event) => onChange(event.currentTarget.value)}
        >
          <option value="">请选择{label}</option>
          {options.map((option) => (
            <option
              key={workLeadOptionValue(option)}
              value={workLeadOptionValue(option)}
            >
              {displayText(option.name || option.value)}
            </option>
          ))}
        </select>
      ) : fieldType === "checkbox" || fieldType === "multi_select" ? (
        <WorkLeadMultiOptions options={options} value={value} onChange={onChange} />
      ) : fieldType === "boolean" ? (
        <button
          type="button"
          role="switch"
          aria-checked={Boolean(value)}
          className={`inline-flex h-9 items-center gap-2 rounded-md border px-3 text-sm ${
            value
              ? "border-foreground bg-foreground text-background"
              : "border-input bg-background"
          }`}
          onClick={() => onChange(!Boolean(value))}
        >
          <span
            className={`inline-flex h-4 w-4 items-center justify-center rounded border ${
              value ? "border-background/50" : "border-input"
            }`}
          >
            {value ? <Check className="h-3 w-3" /> : null}
          </span>
          {value ? "是" : "否"}
        </button>
      ) : (
        <Input
          type={workLeadInputType(fieldType)}
          step={fieldType === "number" || fieldType === "money" ? "any" : undefined}
          value={workLeadTextValue(value)}
          onChange={(event) => onChange(event.currentTarget.value)}
        />
      )}
    </label>
  );
}

function WorkLeadMultiOptions({
  options,
  value,
  onChange,
}: {
  options: WorkLeadTemplateOption[];
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const selected = new Set(workLeadMultiValue(value));
  return (
    <div className="grid gap-1 rounded-md border border-input bg-background p-2 sm:grid-cols-2">
      {options.map((option) => {
        const optionValue = workLeadOptionValue(option);
        const checked = selected.has(optionValue);
        return (
          <button
            key={optionValue}
            type="button"
            className="flex min-w-0 items-center gap-2 rounded px-2 py-2 text-left text-sm hover:bg-muted/60"
            onClick={() => {
              const next = new Set(selected);
              if (checked) next.delete(optionValue);
              else next.add(optionValue);
              onChange(Array.from(next));
            }}
          >
            <span
              className={`inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border ${
                checked
                  ? "border-foreground bg-foreground text-background"
                  : "border-input"
              }`}
            >
              {checked ? <Check className="h-3 w-3" /> : null}
            </span>
            <span className="truncate">
              {displayText(option.name || option.value)}
            </span>
          </button>
        );
      })}
    </div>
  );
}

function workLeadTemplateFieldKey(field: WorkLeadTemplateField): string {
  const id = textValue(field.id);
  return id ? `data:${id}` : "";
}

function workLeadTemplateDefaultValue(field: WorkLeadTemplateField): unknown {
  const value = field.default_value;
  if (field.field_type === "boolean") {
    return value === true || value === "true" || value === "1" || value === 1;
  }
  if (field.field_type === "checkbox" || field.field_type === "multi_select") {
    return workLeadMultiValue(value);
  }
  return value;
}

function workLeadOptionValue(option: WorkLeadTemplateOption): string {
  return textValue(option.value || option.id);
}

function workLeadMultiValue(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(textValue).filter(Boolean);
  const text = textValue(value);
  if (!text) return [];
  return text.split(",").map((part) => part.trim()).filter(Boolean);
}

function workLeadInputType(
  fieldType: string,
): "text" | "number" | "date" | "datetime-local" {
  if (fieldType === "number" || fieldType === "money") return "number";
  if (fieldType === "date") return "date";
  if (fieldType === "datetime") return "datetime-local";
  return "text";
}

function workLeadTextValue(value: unknown): string {
  if (value === null || value === undefined || typeof value === "object") {
    return "";
  }
  return String(value);
}
