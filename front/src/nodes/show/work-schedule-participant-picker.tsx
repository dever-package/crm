import { useMemo } from "react";

import { textValue } from "./work-core";
import {
  uniqueWorkPersonIDs,
  WorkPeoplePicker,
} from "./work-people-picker";
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
  const lockedIDs = uniqueWorkPersonIDs([currentStaffID, ownerStaffID]);
  const badgeLabels = useMemo(() => {
    const result: Record<string, string[]> = {};
    if (currentStaffID) result[currentStaffID] = ["自己"];
    if (ownerStaffID) {
      result[ownerStaffID] = uniqueLabels([
        ...(result[ownerStaffID] || []),
        "负责人",
      ]);
    }
    return result;
  }, [currentStaffID, ownerStaffID]);

  return (
    <WorkPeoplePicker
      value={value}
      currentDepartmentID={currentDepartmentID}
      staff={staff}
      departments={departments}
      existingPeople={existingParticipants}
      lockedIDs={lockedIDs}
      badgeLabels={badgeLabels}
      label="参与人"
      emptyLabel="暂无参与人"
      disabled={disabled}
      onChange={onChange}
    />
  );
}

export function uniqueScheduleParticipantIDs(values: unknown[]): string[] {
  return uniqueWorkPersonIDs(values);
}

function uniqueLabels(labels: string[]): string[] {
  return Array.from(new Set(labels.map(textValue).filter(Boolean)));
}
