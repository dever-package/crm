import { useEffect, useState } from "react";
import { Check, Search } from "lucide-react";

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
  WorkScheduleDepartmentOption,
  WorkScheduleStaffOption,
} from "./work-schedule-types";

export type WorkScheduleParticipantDialogProps = {
  open: boolean;
  selectedStaffIDs: string[];
  currentDepartmentID: string;
  staff: WorkScheduleStaffOption[];
  departments: WorkScheduleDepartmentOption[];
  onAdd: (staffIDs: string[]) => void;
  onOpenChange: (open: boolean) => void;
};

export function WorkScheduleParticipantDialog({
  open,
  selectedStaffIDs,
  currentDepartmentID,
  staff,
  departments,
  onAdd,
  onOpenChange,
}: WorkScheduleParticipantDialogProps) {
  const defaultDepartmentID = scheduleDefaultDepartmentID(
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
          <DialogTitle>添加参与人</DialogTitle>
          <DialogDescription className="sr-only">
            选择要加入当前日程的人员
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
                              toggleParticipantID(current, staffID),
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

function scheduleDefaultDepartmentID(
  departments: WorkScheduleDepartmentOption[],
  currentDepartmentID: string,
): string {
  return departments.some(
    (department) => textValue(department.id) === currentDepartmentID,
  )
    ? currentDepartmentID
    : textValue(departments[0]?.id);
}

function toggleParticipantID(values: string[], id: string): string[] {
  if (!id) return values;
  return values.includes(id)
    ? values.filter((value) => value !== id)
    : [...values, id];
}
