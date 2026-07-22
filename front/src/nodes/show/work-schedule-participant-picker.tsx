import { useMemo, useState } from "react";
import { Plus, Trash2, UserRound } from "lucide-react";

import { Button } from "@/components/ui/button";

import { textValue } from "./work-core";
import { WorkScheduleParticipantDialog } from "./work-schedule-participant-dialog";
import type {
  WorkScheduleDepartmentOption,
  WorkScheduleParticipant,
  WorkScheduleStaffOption,
} from "./work-schedule-types";

export type WorkScheduleParticipantPickerProps = {
  value: string[];
  currentStaffID: string;
  ownerStaffID: string;
  currentDepartmentID: string;
  staff: WorkScheduleStaffOption[];
  departments: WorkScheduleDepartmentOption[];
  existingParticipants: WorkScheduleParticipant[];
  disabled?: boolean;
  onChange: (value: string[]) => void;
};

type ScheduleParticipantPerson = {
  name: string;
  phone: string;
  departmentName: string;
};

export function WorkScheduleParticipantPicker({
  value,
  currentStaffID,
  ownerStaffID,
  currentDepartmentID,
  staff,
  departments,
  existingParticipants,
  disabled = false,
  onChange,
}: WorkScheduleParticipantPickerProps) {
  const [open, setOpen] = useState(false);

  const departmentNames = useMemo(() => {
    const result = new Map<string, string>();
    departments.forEach((department) => {
      const id = textValue(department.id);
      if (id) result.set(id, textValue(department.name));
    });
    return result;
  }, [departments]);
  const people = useMemo(
    () => scheduleParticipantPeople(staff, existingParticipants, departmentNames),
    [departmentNames, existingParticipants, staff],
  );
  const lockedIDs = uniqueScheduleParticipantIDs([
    currentStaffID,
    ownerStaffID,
  ]);
  const lockedSet = new Set(lockedIDs);
  const selectedIDs = uniqueScheduleParticipantIDs([...lockedIDs, ...value]);

  return (
    <div className="min-w-0">
      <div className="mb-1.5 flex min-h-9 items-center justify-between gap-3">
        <div className="flex min-w-0 items-baseline gap-2">
          <span className="text-sm font-medium">参与人</span>
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
          添加人员
        </Button>
      </div>

      <div className="max-h-64 divide-y overflow-y-auto rounded-md border bg-background">
        {selectedIDs.map((staffID) => {
          const person = people.get(staffID) || unknownParticipant();
          const isCurrent = staffID === currentStaffID;
          const isOwner = staffID === ownerStaffID;
          const locked = lockedSet.has(staffID);
          const description = participantDescription(person);
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
                  {isCurrent ? <ParticipantTag>自己</ParticipantTag> : null}
                  {isOwner ? <ParticipantTag>负责人</ParticipantTag> : null}
                </div>
                {description ? (
                  <p className="truncate text-xs text-muted-foreground">
                    {description}
                  </p>
                ) : null}
              </div>
              {!locked ? (
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
        })}
      </div>

      <WorkScheduleParticipantDialog
        open={open}
        selectedStaffIDs={selectedIDs}
        currentDepartmentID={currentDepartmentID}
        staff={staff}
        departments={departments}
        onAdd={(staffIDs) =>
          onChange(uniqueScheduleParticipantIDs([...value, ...staffIDs]))
        }
        onOpenChange={setOpen}
      />
    </div>
  );
}

export function uniqueScheduleParticipantIDs(values: unknown[]): string[] {
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

function scheduleParticipantPeople(
  staff: WorkScheduleStaffOption[],
  existingParticipants: WorkScheduleParticipant[],
  departmentNames: Map<string, string>,
): Map<string, ScheduleParticipantPerson> {
  const result = new Map<string, ScheduleParticipantPerson>();
  existingParticipants.forEach((participant) => {
    const id = textValue(participant.staff_id);
    if (!id) return;
    const departmentID = textValue(participant.department_id);
    result.set(id, {
      name: textValue(participant.staff_name) || "未知人员",
      phone: "",
      departmentName:
        textValue(participant.department_name) ||
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
      phone: textValue(person.phone),
      departmentName:
        departmentNames.get(departmentID) || current?.departmentName || "",
    });
  });
  return result;
}

function unknownParticipant(): ScheduleParticipantPerson {
  return {
    name: "未知人员",
    phone: "",
    departmentName: "",
  };
}

function participantDescription(person: ScheduleParticipantPerson): string {
  return [person.departmentName, person.phone].filter(Boolean).join(" · ");
}

function ParticipantTag({ children }: { children: string }) {
  return (
    <span className="inline-flex h-5 items-center rounded border bg-muted px-1.5 text-[11px] text-muted-foreground">
      {children}
    </span>
  );
}
