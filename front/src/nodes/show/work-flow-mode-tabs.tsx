import { Button } from "@/components/ui/button";

import type { WorkCustomerMode } from "./work-core";

export type WorkFlowModeCounts = Record<WorkCustomerMode, number>;

const workFlowModeOptions: ReadonlyArray<{
  value: WorkCustomerMode;
  label: string;
}> = [
  { value: "all", label: "全部" },
  { value: "pending", label: "待处理" },
  { value: "processed", label: "已处理" },
  { value: "done", label: "已结束" },
];

export function emptyWorkFlowModeCounts(): WorkFlowModeCounts {
  return {
    all: 0,
    pending: 0,
    processed: 0,
    done: 0,
  };
}

export function normalizeWorkFlowModeCounts(
  counts: Partial<Record<WorkCustomerMode, unknown>> | undefined,
  activeMode: WorkCustomerMode,
  activeTotal: unknown,
): WorkFlowModeCounts {
  const normalized = emptyWorkFlowModeCounts();
  workFlowModeOptions.forEach((option) => {
    normalized[option.value] = workFlowModeCount(counts?.[option.value]);
  });
  if (
    !Object.prototype.hasOwnProperty.call(counts || {}, activeMode) &&
    Number.isFinite(Number(activeTotal)) &&
    Number(activeTotal) >= 0
  ) {
    normalized[activeMode] = Math.floor(Number(activeTotal));
  }
  return normalized;
}

export function normalizeWorkFlowMode(
  value: unknown,
  fallback: WorkCustomerMode = "pending",
): WorkCustomerMode {
  return workFlowModeOptions.some((option) => option.value === value)
    ? (value as WorkCustomerMode)
    : fallback;
}

export function WorkFlowModeTabs({
  mode,
  counts,
  onChange,
}: {
  mode: WorkCustomerMode;
  counts: WorkFlowModeCounts;
  onChange: (mode: WorkCustomerMode) => void;
}) {
  return (
    <div className="inline-flex max-w-full items-center gap-1 overflow-x-auto rounded-md bg-muted/40 p-1">
      {workFlowModeOptions.map((option) => (
        <Button
          type="button"
          key={option.value}
          variant="ghost"
          aria-pressed={mode === option.value}
          className={`h-auto shrink-0 rounded px-3 py-1.5 text-sm font-medium ${
            mode === option.value
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
          onClick={() => onChange(option.value)}
        >
          {option.label}
          <span className="ml-1 text-xs text-muted-foreground">
            {counts[option.value] || 0}
          </span>
        </Button>
      ))}
    </div>
  );
}

function workFlowModeCount(value: unknown): number {
  const count = Number(value);
  return Number.isFinite(count) && count > 0 ? Math.floor(count) : 0;
}
