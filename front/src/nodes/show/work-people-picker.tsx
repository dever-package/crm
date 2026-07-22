import { useEffect, useMemo, useState } from "react";
import { Check, Plus, Search, Trash2, UserRound } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import { textValue } from "./work-core";
import type {
  WorkDepartmentOption,
  WorkPersonSnapshot,
  WorkStaffOption,
} from "./work-people-types";

export type WorkPeoplePickerProps = {
  value: string[];
  currentDepartmentID: string;
  staff: WorkStaffOption[];
  departments: WorkDepartmentOption[];
  existingPeople?: WorkPersonSnapshot[];
  lockedIDs?: string[];
  badgeLabels?: Record<string, string[]>;
  label?: string;
  addLabel?: string;
  emptyLabel?: string;
  disabled?: boolean;
  onChange: (value: string[]) => void;
};

type WorkPersonView = {
  name: string;
  phone: string;
  departmentName: string;
};

export function WorkPeoplePicker({
  value,
  currentDepartmentID,
  staff,
  departments,
  existingPeople = [],
  lockedIDs = [],
  badgeLabels = {},
  label = "关联人员",
  addLabel = "添加人员",
  emptyLabel = "暂未关联人员",
  disabled = false,
  onChange,
}: WorkPeoplePickerProps) {
  const [open, setOpen] = useState(false);
  const departmentNames = useMemo(
    () => workDepartmentNames(departments),
    [departments],
  );
  const people = useMemo(
    () => workPeopleByID(staff, existingPeople, departmentNames),
    [departmentNames, existingPeople, staff],
  );
  const normalizedLockedIDs = uniqueWorkPersonIDs(lockedIDs);
  const lockedSet = new Set(normalizedLockedIDs);
  const selectedIDs = uniqueWorkPersonIDs([...normalizedLockedIDs, ...value]);

  return (
    <div className="min-w-0">
      <div className="mb-1.5 flex min-h-9 items-center justify-between gap-3">
        <div className="flex min-w-0 items-baseline gap-2">
          <span className="text-sm font-medium">{label}</span>
          <span className="text-xs text-muted-foreground">
            {selectedIDs.length} 人
          </span>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={() => setOpen(true)}
        >
          <Plus className="h-4 w-4" />
          {addLabel}
        </Button>
      </div>

      <div className="max-h-64 divide-y overflow-y-auto rounded-md border bg-background">
        {selectedIDs.length ? (
          selectedIDs.map((staffID) => {
            const person = people.get(staffID) || unknownWorkPerson();
            const description = workPersonDescription(person);
            return (
              <div
                key={staffID}
                className="flex min-h-12 items-center gap-3 px-3 py-2"
              >
                <span className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
                  <UserRound className="h-4 w-4" />
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex min-w-0 flex-wrap items-center gap-1.5">
                    <span className="truncate text-sm font-medium">
                      {person.name}
                    </span>
                    {(badgeLabels[staffID] || []).map((badge) => (
                      <WorkPersonTag key={badge}>{badge}</WorkPersonTag>
                    ))}
                  </div>
                  {description ? (
                    <p className="truncate text-xs text-muted-foreground">
                      {description}
                    </p>
                  ) : null}
                </div>
                {!lockedSet.has(staffID) ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 shrink-0 text-muted-foreground hover:text-destructive"
                    title={`移除${person.name}`}
                    aria-label={`移除${person.name}`}
                    disabled={disabled}
                    onClick={() =>
                      onChange(value.filter((valueID) => valueID !== staffID))
                    }
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                ) : null}
              </div>
            );
          })
        ) : (
          <div className="px-3 py-8 text-center text-sm text-muted-foreground">
            {emptyLabel}
          </div>
        )}
      </div>

      <WorkPeopleDialog
        open={open}
        title={addLabel}
        selectedStaffIDs={selectedIDs}
        currentDepartmentID={currentDepartmentID}
        staff={staff}
        departments={departments}
        onAdd={(staffIDs) =>
          onChange(uniqueWorkPersonIDs([...value, ...staffIDs]))
        }
        onOpenChange={setOpen}
      />
    </div>
  );
}

type WorkPeopleDialogProps = {
  open: boolean;
  title: string;
  selectedStaffIDs: string[];
  currentDepartmentID: string;
  staff: WorkStaffOption[];
  departments: WorkDepartmentOption[];
  onAdd: (staffIDs: string[]) => void;
  onOpenChange: (open: boolean) => void;
};

function WorkPeopleDialog({
  open,
  title,
  selectedStaffIDs,
  currentDepartmentID,
  staff,
  departments,
  onAdd,
  onOpenChange,
}: WorkPeopleDialogProps) {
  const defaultDepartmentID = workDefaultDepartmentID(
    departments,
    currentDepartmentID,
  );
  const [departmentID, setDepartmentID] = useState("");
  const [keyword, setKeyword] = useState("");
  const [pendingStaffIDs, setPendingStaffIDs] = useState<string[]>([]);

  useEffect(() => {
    setDepartmentID(open ? defaultDepartmentID : "");
    setKeyword("");
    setPendingStaffIDs([]);
  }, [defaultDepartmentID, open]);

  const selectedSet = new Set(selectedStaffIDs);
  const pendingSet = new Set(pendingStaffIDs);
  const departmentStaff = staff.filter((person) => {
    const staffID = textValue(person.id);
    return (
      staffID &&
      textValue(person.department_id) === departmentID &&
      !selectedSet.has(staffID)
    );
  });
  const normalizedKeyword = keyword.trim().toLocaleLowerCase();
  const visibleStaff = departmentStaff.filter((person) => {
    if (!normalizedKeyword) return true;
    return [person.name, person.phone].some((item) =>
      textValue(item).toLocaleLowerCase().includes(normalizedKeyword),
    );
  });

  const confirmSelection = () => {
    if (!pendingStaffIDs.length) return;
    onAdd(pendingStaffIDs);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription className="sr-only">
            先选择部门，再选择需要关联的人员
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4">
          <div className="grid gap-1.5">
            <span className="text-sm font-medium">部门</span>
            <Select
              value={departmentID}
              onValueChange={(nextDepartmentID) => {
                setDepartmentID(nextDepartmentID);
                setKeyword("");
                setPendingStaffIDs([]);
              }}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="请选择部门" />
              </SelectTrigger>
              <SelectContent position="popper">
                {departments
                  .filter((department) => textValue(department.id))
                  .map((department) => (
                    <SelectItem
                      key={textValue(department.id)}
                      value={textValue(department.id)}
                    >
                      {textValue(department.name)}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-1.5">
            <span className="text-sm font-medium">人员</span>
            <div className="rounded-md border bg-background p-2">
              <div className="relative mb-2">
                <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  type="search"
                  className="h-9 pl-8"
                  placeholder="搜索姓名或手机号"
                  value={keyword}
                  onChange={(event) => setKeyword(event.currentTarget.value)}
                />
              </div>
              <div className="h-72 max-h-[45vh] overflow-y-auto">
                {visibleStaff.length ? (
                  <div className="grid gap-1">
                    {visibleStaff.map((person) => {
                      const staffID = textValue(person.id);
                      const checked = pendingSet.has(staffID);
                      return (
                        <Button
                          key={staffID}
                          type="button"
                          variant="ghost"
                          aria-pressed={checked}
                          className="h-auto min-h-10 w-full justify-start gap-2 px-2 py-2 font-normal"
                          onClick={() =>
                            setPendingStaffIDs((current) =>
                              toggleWorkPersonID(current, staffID),
                            )
                          }
                        >
                          <span
                            className={`inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-sm border ${
                              checked
                                ? "border-primary bg-primary text-primary-foreground"
                                : "border-input bg-background"
                            }`}
                          >
                            {checked ? <Check className="h-3 w-3" /> : null}
                          </span>
                          <span className="min-w-0 flex-1 text-left">
                            <span className="block truncate text-sm">
                              {textValue(person.name)}
                            </span>
                            {textValue(person.phone) ? (
                              <span className="block truncate text-xs text-muted-foreground">
                                {textValue(person.phone)}
                              </span>
                            ) : null}
                          </span>
                        </Button>
                      );
                    })}
                  </div>
                ) : (
                  <div className="flex h-full items-center justify-center px-4 text-center text-sm text-muted-foreground">
                    {departmentStaff.length
                      ? "没有匹配人员"
                      : "该部门暂无可添加人员"}
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            取消
          </Button>
          <Button
            type="button"
            disabled={!pendingStaffIDs.length}
            onClick={confirmSelection}
          >
            添加{pendingStaffIDs.length ? `（${pendingStaffIDs.length}）` : ""}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function uniqueWorkPersonIDs(values: unknown[]): string[] {
  const result: string[] = [];
  const seen = new Set<string>();
  values.forEach((value) => {
    const id = textValue(value);
    if (!id || seen.has(id)) return;
    seen.add(id);
    result.push(id);
  });
  return result;
}

function workDepartmentNames(
  departments: WorkDepartmentOption[],
): Map<string, string> {
  const result = new Map<string, string>();
  departments.forEach((department) => {
    const id = textValue(department.id);
    if (id) result.set(id, textValue(department.name));
  });
  return result;
}

function workPeopleByID(
  staff: WorkStaffOption[],
  existingPeople: WorkPersonSnapshot[],
  departmentNames: Map<string, string>,
): Map<string, WorkPersonView> {
  const result = new Map<string, WorkPersonView>();
  existingPeople.forEach((person) => {
    const id = textValue(person.staff_id);
    if (!id) return;
    const departmentID = textValue(person.department_id);
    result.set(id, {
      name: textValue(person.staff_name) || "未知人员",
      phone: textValue(person.phone),
      departmentName:
        textValue(person.department_name) ||
        departmentNames.get(departmentID) ||
        "",
    });
  });
  staff.forEach((person) => {
    const id = textValue(person.id);
    if (!id) return;
    const current = result.get(id);
    const departmentID = textValue(person.department_id);
    result.set(id, {
      name: textValue(person.name) || current?.name || "未知人员",
      phone: textValue(person.phone) || current?.phone || "",
      departmentName:
        departmentNames.get(departmentID) || current?.departmentName || "",
    });
  });
  return result;
}

function workDefaultDepartmentID(
  departments: WorkDepartmentOption[],
  currentDepartmentID: string,
): string {
  return departments.some(
    (department) => textValue(department.id) === currentDepartmentID,
  )
    ? currentDepartmentID
    : textValue(departments[0]?.id);
}

function toggleWorkPersonID(values: string[], id: string): string[] {
  if (!id) return values;
  return values.includes(id)
    ? values.filter((value) => value !== id)
    : [...values, id];
}

function unknownWorkPerson(): WorkPersonView {
  return { name: "未知人员", phone: "", departmentName: "" };
}

function workPersonDescription(person: WorkPersonView): string {
  return [person.departmentName, person.phone].filter(Boolean).join(" · ");
}

function WorkPersonTag({ children }: { children: string }) {
  return (
    <span className="inline-flex h-5 items-center rounded border bg-muted px-1.5 text-[11px] text-muted-foreground">
      {children}
    </span>
  );
}
