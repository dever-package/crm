import { useState } from "react";
import {
  MessageCircleMore,
  Pencil,
  Plus,
  Unlink,
  UsersRound,
} from "lucide-react";

import { Button } from "@/components/ui/button";

import {
  textValue,
  type WorkCommunicationGroup,
  type WorkCommunicationGroupType,
} from "./work-core";
import {
  communicationGroupDate,
  CommunicationGroupDissolveDialog,
  CommunicationGroupEditor,
} from "./work-communication-group-dialogs";

export type WorkCommunicationGroupsProps = {
  groups: WorkCommunicationGroup[];
  groupTypes: WorkCommunicationGroupType[];
  workflowInstanceID: string;
  canCreate: boolean;
};

export function WorkCommunicationGroups({
  groups,
  groupTypes,
  workflowInstanceID,
  canCreate,
}: WorkCommunicationGroupsProps) {
  const [editingGroup, setEditingGroup] =
    useState<WorkCommunicationGroup | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [dissolvingGroup, setDissolvingGroup] =
    useState<WorkCommunicationGroup | null>(null);
  const hasActiveGroup = groups.some((group) => group.status === "active");
  const canAdd = canCreate && Boolean(workflowInstanceID) && !hasActiveGroup;

  const openCreate = () => {
    setEditingGroup(null);
    setEditorOpen(true);
  };
  const openEdit = (group: WorkCommunicationGroup) => {
    setEditingGroup(group);
    setEditorOpen(true);
  };

  return (
    <div className="grid gap-4">
      <div className="flex min-h-9 flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold">沟通群</h3>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {groups.length} 个群，保留当前群与历史解散记录
          </p>
        </div>
        {canAdd ? (
          <Button type="button" size="sm" onClick={openCreate}>
            <Plus className="h-4 w-4" />
            新增沟通群
          </Button>
        ) : null}
      </div>

      {groups.length ? (
        <div className="grid gap-3">
          {groups.map((group) => (
            <CommunicationGroupRow
              key={textValue(group.id)}
              group={group}
              onEdit={() => openEdit(group)}
              onDissolve={() => setDissolvingGroup(group)}
            />
          ))}
        </div>
      ) : (
        <div className="flex min-h-44 flex-col items-center justify-center text-center text-muted-foreground">
          <MessageCircleMore className="h-6 w-6" />
          <span className="mt-2 text-sm">暂无沟通群</span>
        </div>
      )}

      <CommunicationGroupEditor
        open={editorOpen}
        group={editingGroup}
        groupTypes={groupTypes}
        workflowInstanceID={workflowInstanceID}
        onOpenChange={setEditorOpen}
      />
      <CommunicationGroupDissolveDialog
        group={dissolvingGroup}
        onOpenChange={(open) => {
          if (!open) setDissolvingGroup(null);
        }}
      />
    </div>
  );
}

function CommunicationGroupRow({
  group,
  onEdit,
  onDissolve,
}: {
  group: WorkCommunicationGroup;
  onEdit: () => void;
  onDissolve: () => void;
}) {
  const people = Array.isArray(group.staff) ? group.staff : [];
  const active = group.status === "active";
  return (
    <article className="rounded-md border bg-background px-4 py-3.5">
      <div className="flex min-w-0 flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <h4 className="break-words text-sm font-semibold">
              {textValue(group.name) || "未命名沟通群"}
            </h4>
            <span className="rounded border bg-muted/50 px-2 py-0.5 text-[11px] text-muted-foreground">
              {textValue(group.group_type_name) || "沟通群"}
            </span>
            <span
              className={`rounded px-2 py-0.5 text-[11px] font-medium ${
                active
                  ? "bg-emerald-50 text-emerald-700"
                  : "bg-muted text-muted-foreground"
              }`}
            >
              {textValue(group.status_name) || (active ? "使用中" : "已解散")}
            </span>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            建群 {communicationGroupDate(group.established_at)}
            {group.dissolved_at
              ? ` · 解散 ${communicationGroupDate(group.dissolved_at)}`
              : ""}
          </p>
        </div>
        {group.can_edit ? (
          <div className="flex shrink-0 items-center gap-1">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              title="编辑沟通群"
              aria-label="编辑沟通群"
              onClick={onEdit}
            >
              <Pencil className="h-4 w-4" />
            </Button>
            {active ? (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-8 w-8 text-muted-foreground hover:text-destructive"
                title="解散沟通群"
                aria-label="解散沟通群"
                onClick={onDissolve}
              >
                <Unlink className="h-4 w-4" />
              </Button>
            ) : null}
          </div>
        ) : null}
      </div>

      <div className="mt-3 grid gap-3 border-t border-border/60 pt-3 sm:grid-cols-2">
        <CommunicationGroupText label="外部群ID" value={group.external_group_id} />
        <div>
          <div className="text-xs text-muted-foreground">关联人员</div>
          {people.length ? (
            <div className="mt-1.5 flex flex-wrap gap-1.5">
              {people.map((person) => (
                <span
                  key={textValue(person.staff_id)}
                  className="inline-flex items-center gap-1 rounded border bg-muted/30 px-2 py-1 text-xs"
                >
                  <UsersRound className="h-3.5 w-3.5 text-muted-foreground" />
                  {textValue(person.staff_name) || "未知人员"}
                  {person.department_name ? (
                    <span className="text-muted-foreground">
                      · {textValue(person.department_name)}
                    </span>
                  ) : null}
                  {person.role && person.role !== "participant" ? (
                    <span className="text-muted-foreground">
                      · {textValue(person.role_name) || textValue(person.role)}
                    </span>
                  ) : null}
                </span>
              ))}
            </div>
          ) : (
            <div className="mt-1.5 text-sm text-muted-foreground">未关联</div>
          )}
        </div>
        <CommunicationGroupText label="智能总结" value={group.summary} />
        <CommunicationGroupText label="备注" value={group.remark} />
        {!active ? (
          <CommunicationGroupText
            label="解散原因"
            value={group.dissolve_reason}
          />
        ) : null}
      </div>
    </article>
  );
}

function CommunicationGroupText({
  label,
  value,
}: {
  label: string;
  value: unknown;
}) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1.5 break-words text-sm">
        {textValue(value) || (
          <span className="text-muted-foreground">未填写</span>
        )}
      </div>
    </div>
  );
}
