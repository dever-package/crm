import type { ReactElement } from "react";
import { Check } from "lucide-react";

import { Button } from "@/components/ui/button";

import {
  textValue,
  workStoreValue,
  type WorkNodeProps,
} from "./work-core";

export type CustomerTagOption = {
  id: string;
  name: string;
  levelID: string;
  levelName: string;
  levelSort: number;
  sort: number;
};

type CustomerTagSelectorProps = {
  options: CustomerTagOption[];
  value: string[];
  disabled?: boolean;
  error?: string;
  onChange: (tagIDs: string[]) => void;
};

type CustomerTagGroup = {
  id: string;
  name: string;
  sort: number;
  options: CustomerTagOption[];
};

const customerTagLevelDots = [
  "bg-emerald-500",
  "bg-cyan-500",
  "bg-pink-500",
  "bg-violet-500",
];

export function CustomerTagSelector({
  options,
  value,
  disabled,
  error,
  onChange,
}: CustomerTagSelectorProps): ReactElement {
  const groups = groupCustomerTagOptions(options);
  const optionsByID = new Map(options.map((option) => [option.id, option]));
  const selectedOption = value
    .map(textValue)
    .map((tagID) => optionsByID.get(tagID))
    .find((option): option is CustomerTagOption => Boolean(option));
  const selected = new Set(selectedOption ? [selectedOption.id] : []);
  const activeGroup = groups.find((group) =>
    group.options.some((option) => selected.has(option.id)),
  );

  const selectTag = (option: CustomerTagOption) => {
    if (disabled) return;
    onChange([option.id]);
  };

  return (
    <div
      className={`space-y-3 rounded-md border bg-background p-3 ${
        error ? "border-destructive" : "border-input"
      }`}
      aria-invalid={Boolean(error)}
    >
      {groups.length > 0 ? (
        groups.map((group, groupIndex) => {
          const groupSelected = activeGroup?.id === group.id;
          return (
            <section
              key={group.id}
              className={groupIndex > 0 ? "border-t border-border/60 pt-3" : ""}
            >
              <div className="mb-2 flex items-center gap-2 text-sm font-medium text-foreground">
                <span
                  className={`h-2 w-2 shrink-0 rounded-full ${
                    customerTagLevelDots[groupIndex % customerTagLevelDots.length]
                  }`}
                />
                <span>{group.name}</span>
                {groupSelected ? (
                  <span className="ml-auto text-xs font-normal text-muted-foreground">
                    已选择
                  </span>
                ) : null}
              </div>
              <div
                className="flex flex-wrap gap-2"
                role="radiogroup"
                aria-label={group.name}
              >
                {group.options.map((option) => {
                  const checked = selected.has(option.id);
                  return (
                    <Button
                      key={option.id}
                      type="button"
                      variant={checked ? "default" : "outline"}
                      size="sm"
                      role="radio"
                      aria-checked={checked}
                      disabled={disabled}
                      className={`h-9 justify-start gap-2 px-3 font-normal ${
                        checked
                          ? "border-primary bg-primary text-primary-foreground shadow-sm hover:bg-primary/90 hover:text-primary-foreground"
                          : "text-muted-foreground hover:text-foreground"
                      }`}
                      onClick={() => selectTag(option)}
                    >
                      <span
                        className={`inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-full border ${
                          checked
                            ? "border-primary-foreground bg-primary-foreground/15 text-primary-foreground"
                            : "border-input bg-background"
                        }`}
                      >
                        {checked ? <Check className="h-3 w-3" /> : null}
                      </span>
                      <span>{option.name}</span>
                    </Button>
                  );
                })}
              </div>
            </section>
          );
        })
      ) : (
        <p className="py-2 text-sm text-muted-foreground">暂无可用客户标签</p>
      )}
      <div className="border-t border-border/60 pt-2 text-sm">
        <span className="text-muted-foreground">自动判定：</span>
        <span className="font-medium text-foreground">
          {activeGroup?.name || "待选择标签"}
        </span>
      </div>
    </div>
  );
}

export function ShowCrmCustomerTagSelector({
  item,
  store,
  value,
  setValue,
  error,
}: WorkNodeProps): ReactElement {
  const optionsPath = textValue(item?.meta?.optionsPath);
  const options = normalizeCustomerTagOptions(
    workStoreValue(store, optionsPath, []),
  );
  return (
    <CustomerTagSelector
      options={options}
      value={normalizeCustomerTagIDs(value)}
      disabled={Boolean(item?.meta?.readonly)}
      error={error}
      onChange={(tagIDs) => setValue?.(tagIDs)}
    />
  );
}

export function normalizeCustomerTagOptions(raw: unknown): CustomerTagOption[] {
  if (!Array.isArray(raw)) return [];
  return raw
    .map((source) => {
      if (!source || typeof source !== "object" || Array.isArray(source)) {
        return null;
      }
      const row = source as Record<string, unknown>;
      const id = textValue(row.id || row.value);
      const name = textValue(row.name || row.label);
      const levelID = textValue(row.level_id || row.levelID);
      const levelName = textValue(
        row.level_name || row.levelName || row["level.name"],
      );
      if (!id || !name || !levelID || !levelName) return null;
      return {
        id,
        name,
        levelID,
        levelName,
        levelSort: Number(row.level_sort || row.levelSort) || 100,
        sort: Number(row.sort) || 100,
      } satisfies CustomerTagOption;
    })
    .filter((option): option is CustomerTagOption => Boolean(option));
}

export function normalizeCustomerTagIDs(raw: unknown): string[] {
  if (Array.isArray(raw)) return raw.map(textValue).filter(Boolean);
  return textValue(raw)
    .split(",")
    .map((tagID) => tagID.trim())
    .filter(Boolean);
}

function groupCustomerTagOptions(
  options: CustomerTagOption[],
): CustomerTagGroup[] {
  const groups = new Map<string, CustomerTagGroup>();
  for (const option of options) {
    const current = groups.get(option.levelID);
    if (current) {
      current.options.push(option);
      continue;
    }
    groups.set(option.levelID, {
      id: option.levelID,
      name: option.levelName,
      sort: option.levelSort,
      options: [option],
    });
  }
  return Array.from(groups.values())
    .map((group) => ({
      ...group,
      options: [...group.options].sort(
        (left, right) => left.sort - right.sort || left.name.localeCompare(right.name),
      ),
    }))
    .sort((left, right) => left.sort - right.sort || left.name.localeCompare(right.name));
}
