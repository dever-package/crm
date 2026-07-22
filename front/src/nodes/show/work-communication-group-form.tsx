import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import { Loader2 } from "lucide-react";

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
  errorMessage,
  formatWorkDate,
  textValue,
  workApi,
  type WorkCommunicationGroup,
  type WorkCommunicationGroupType,
} from "./work-core";
import { WorkPeoplePicker } from "./work-people-picker";
import type { WorkPeopleOptions } from "./work-people-types";

export type CommunicationGroupDraft = {
  id: string;
  workflowInstanceID: string;
  groupTypeID: string;
  name: string;
  externalGroupID: string;
  establishedAt: string;
  summary: string;
  remark: string;
  staffIDs: string[];
};

export function CommunicationGroupForm({
  draft,
  groupTypes,
  peopleOptions,
  peopleLoading,
  existingGroup,
  disabled = false,
  onChange,
}: {
  draft: CommunicationGroupDraft;
  groupTypes: WorkCommunicationGroupType[];
  peopleOptions: WorkPeopleOptions | null;
  peopleLoading: boolean;
  existingGroup?: WorkCommunicationGroup | null;
  disabled?: boolean;
  onChange: (draft: CommunicationGroupDraft) => void;
}) {
  const updateDraft = <K extends keyof CommunicationGroupDraft>(
    key: K,
    value: CommunicationGroupDraft[K],
  ) => onChange({ ...draft, [key]: value });
  const people = peopleOptions || {};

  return (
    <div className="grid gap-x-5 gap-y-5 sm:grid-cols-2">
      <CommunicationGroupField label="群名称" required>
        <Input
          value={draft.name}
          placeholder="请输入群名称"
          disabled={disabled}
          onChange={(event) => updateDraft("name", event.currentTarget.value)}
        />
      </CommunicationGroupField>
      <CommunicationGroupField label="群类型" required>
        <Select
          value={draft.groupTypeID}
          disabled={disabled}
          onValueChange={(value) => updateDraft("groupTypeID", value)}
        >
          <SelectTrigger>
            <SelectValue placeholder="请选择群类型" />
          </SelectTrigger>
          <SelectContent>
            {groupTypes.map((groupType) => (
              <SelectItem
                key={textValue(groupType.id)}
                value={textValue(groupType.id)}
                disabled={
                  textValue(groupType.status) === "2" &&
                  textValue(groupType.id) !== draft.groupTypeID
                }
              >
                {textValue(groupType.name)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </CommunicationGroupField>
      <CommunicationGroupField label="建群日期" required>
        <Input
          type="date"
          value={draft.establishedAt}
          disabled={disabled}
          onChange={(event) =>
            updateDraft("establishedAt", event.currentTarget.value)
          }
        />
      </CommunicationGroupField>
      <CommunicationGroupField label="外部群ID">
        <Input
          value={draft.externalGroupID}
          placeholder="可选"
          disabled={disabled}
          onChange={(event) =>
            updateDraft("externalGroupID", event.currentTarget.value)
          }
        />
      </CommunicationGroupField>
      <CommunicationGroupField label="智能总结" className="sm:col-span-2">
        <Textarea
          rows={3}
          value={draft.summary}
          placeholder="记录群聊智能总结"
          disabled={disabled}
          onChange={(event) => updateDraft("summary", event.currentTarget.value)}
        />
      </CommunicationGroupField>
      <CommunicationGroupField label="备注" className="sm:col-span-2">
        <Textarea
          rows={3}
          value={draft.remark}
          placeholder="记录补充说明"
          disabled={disabled}
          onChange={(event) => updateDraft("remark", event.currentTarget.value)}
        />
      </CommunicationGroupField>
      <div className="sm:col-span-2">
        {peopleLoading ? (
          <div className="flex min-h-24 items-center justify-center gap-2 rounded-md border text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            加载人员
          </div>
        ) : (
          <WorkPeoplePicker
            value={draft.staffIDs}
            currentDepartmentID={textValue(people.current_department_id)}
            staff={people.staff || []}
            departments={people.departments || []}
            existingPeople={existingGroup?.staff || []}
            label="关联人员"
            emptyLabel="暂未关联人员"
            disabled={disabled}
            onChange={(value) => updateDraft("staffIDs", value)}
          />
        )}
      </div>
    </div>
  );
}

export function CommunicationGroupField({
  label,
  required = false,
  className = "",
  children,
}: {
  label: string;
  required?: boolean;
  className?: string;
  children: ReactNode;
}) {
  return (
    <label className={`grid gap-1.5 ${className}`}>
      <span className="text-sm font-medium">
        {label}
        {required ? <span className="ml-1 text-destructive">*</span> : null}
      </span>
      {children}
    </label>
  );
}

export function communicationGroupDraft(
  group: WorkCommunicationGroup | null,
  groupTypes: WorkCommunicationGroupType[],
  workflowInstanceID: string,
): CommunicationGroupDraft {
  return {
    id: textValue(group?.id),
    workflowInstanceID:
      textValue(group?.workflow_instance_id) || workflowInstanceID,
    groupTypeID:
      textValue(group?.group_type_id) ||
      textValue(
        groupTypes.find((groupType) => textValue(groupType.status) !== "2")?.id,
      ),
    name: textValue(group?.name),
    externalGroupID: textValue(group?.external_group_id),
    establishedAt: communicationGroupDateInput(group?.established_at),
    summary: textValue(group?.summary),
    remark: textValue(group?.remark),
    staffIDs: (group?.staff || [])
      .map((person) => textValue(person.staff_id))
      .filter(Boolean),
  };
}

export function communicationGroupDraftPayload(
  draft: CommunicationGroupDraft,
): Record<string, unknown> {
  return {
    communication_group_id: draft.id || undefined,
    workflow_instance_id: draft.workflowInstanceID || undefined,
    group_type_id: draft.groupTypeID,
    name: draft.name.trim(),
    external_group_id: draft.externalGroupID.trim(),
    established_at: draft.establishedAt,
    summary: draft.summary.trim(),
    remark: draft.remark.trim(),
    staff_ids: draft.staffIDs,
  };
}

export function validateCommunicationGroupDraft(
  draft: CommunicationGroupDraft,
): string {
  if (!draft.workflowInstanceID) return "当前任务未关联有效案件流程";
  if (!draft.name.trim()) return "请填写群名称";
  if (!draft.groupTypeID) return "请选择群类型";
  if (!draft.establishedAt) return "请选择建群日期";
  return "";
}

export function useCommunicationGroupPeopleOptions(enabled: boolean) {
  const [peopleOptions, setPeopleOptions] =
    useState<WorkPeopleOptions | null>(null);
  const [peopleLoading, setPeopleLoading] = useState(false);
  const [peopleError, setPeopleError] = useState("");

  useEffect(() => {
    if (!enabled) return;
    let active = true;
    setPeopleLoading(true);
    setPeopleError("");
    void workApi<WorkPeopleOptions>("/crm/work/people_options")
      .then((options) => {
        if (active) setPeopleOptions(options);
      })
      .catch((error) => {
        if (!active) return;
        setPeopleOptions(null);
        setPeopleError(errorMessage(error, "人员列表加载失败"));
      })
      .finally(() => {
        if (active) setPeopleLoading(false);
      });
    return () => {
      active = false;
    };
  }, [enabled]);

  return { peopleOptions, peopleLoading, peopleError };
}

export function communicationGroupDate(value: unknown): string {
  const formatted = formatWorkDate(value);
  return formatted === "-" ? "-" : formatted.slice(0, 10);
}

function communicationGroupDateInput(value: unknown): string {
  const date = communicationGroupDate(value);
  return date === "-" ? communicationGroupToday() : date;
}

export function communicationGroupToday(): string {
  const parts: Record<string, string> = {};
  const formatter = new Intl.DateTimeFormat("zh-CN", {
    timeZone: "Asia/Shanghai",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
  formatter.formatToParts(new Date()).forEach((part) => {
    if (part.type !== "literal") parts[part.type] = part.value;
  });
  return `${parts.year}-${parts.month}-${parts.day}`;
}
