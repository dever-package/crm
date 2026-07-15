import { Check } from "lucide-react";

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

import { displayText, textValue } from "./work-core";

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
  const fields = templates
    .filter((template) => workLeadInputTemplateVisible(template, values))
    .flatMap((template) => template.fields || []);
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

function workLeadInputTemplateVisible(
  template: WorkLeadTemplate,
  values: Record<string, unknown>,
): boolean {
  const fields = template.fields || [];
  const douyinOnly =
    fields.length > 0 &&
    fields.every((field) => textValue(field.field_key).startsWith("douyin_"));
  if (!douyinOnly) return true;

  return fields.some((field) => {
    const value = values[workLeadTemplateFieldKey(field)];
    if (Array.isArray(value)) return value.length > 0;
    return value !== undefined && value !== null && value !== "";
  });
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
    <div className={fullWidth ? "sm:col-span-2" : ""}>
      <span className="mb-1.5 block text-sm font-medium">{label}</span>
      {field.group_name ? (
        <span className="mb-1 block text-xs text-muted-foreground">
          {field.group_name}
        </span>
      ) : null}
      {fieldType === "textarea" ? (
        <Textarea
          className="min-h-24 resize-y"
          value={workLeadTextValue(value)}
          onChange={(event) => onChange(event.currentTarget.value)}
        />
      ) : fieldType === "select" || fieldType === "radio" ? (
        <Select value={textValue(value)} onValueChange={onChange}>
          <SelectTrigger className="w-full">
            <SelectValue placeholder={`请选择${label}`} />
          </SelectTrigger>
          <SelectContent position="popper">
            {options
              .filter((option) => workLeadOptionValue(option))
              .map((option) => (
                <SelectItem
                  key={workLeadOptionValue(option)}
                  value={workLeadOptionValue(option)}
                >
                  {displayText(option.name || option.value)}
                </SelectItem>
              ))}
          </SelectContent>
        </Select>
      ) : fieldType === "checkbox" || fieldType === "multi_select" ? (
        <WorkLeadMultiOptions options={options} value={value} onChange={onChange} />
      ) : fieldType === "boolean" ? (
        <Button
          type="button"
          variant={value ? "default" : "outline"}
          role="switch"
          aria-checked={Boolean(value)}
          className="h-9"
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
        </Button>
      ) : (
        <Input
          type={workLeadInputType(fieldType)}
          step={fieldType === "number" || fieldType === "money" ? "any" : undefined}
          value={workLeadTextValue(value)}
          onChange={(event) => onChange(event.currentTarget.value)}
        />
      )}
    </div>
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
          <Button
            key={optionValue}
            type="button"
            variant="ghost"
            aria-pressed={checked}
            className="h-auto w-full min-w-0 justify-start gap-2 px-2 py-2 text-left text-sm font-normal"
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
          </Button>
        );
      })}
    </div>
  );
}

export function workLeadTemplateFieldKey(
  field: WorkLeadTemplateField,
): string {
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
