import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

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
import { Textarea } from "@/components/ui/textarea";

import {
  errorMessage,
  workApi,
  workRefreshEvent,
  type WorkCommunicationGroup,
  type WorkCommunicationGroupType,
} from "./work-core";
import {
  CommunicationGroupField,
  CommunicationGroupForm,
  communicationGroupDraft,
  communicationGroupDraftPayload,
  communicationGroupToday,
  useCommunicationGroupPeopleOptions,
  validateCommunicationGroupDraft,
  type CommunicationGroupDraft,
} from "./work-communication-group-form";

export { communicationGroupDate } from "./work-communication-group-form";

export function CommunicationGroupEditor({
  open,
  group,
  groupTypes,
  workflowInstanceID,
  onOpenChange,
}: {
  open: boolean;
  group: WorkCommunicationGroup | null;
  groupTypes: WorkCommunicationGroupType[];
  workflowInstanceID: string;
  onOpenChange: (open: boolean) => void;
}) {
  const [draft, setDraft] = useState<CommunicationGroupDraft>(() =>
    communicationGroupDraft(group, groupTypes, workflowInstanceID),
  );
  const [saving, setSaving] = useState(false);
  const { peopleOptions, peopleLoading, peopleError } =
    useCommunicationGroupPeopleOptions(open);

  useEffect(() => {
    if (!open) return;
    setDraft(communicationGroupDraft(group, groupTypes, workflowInstanceID));
  }, [group, groupTypes, open, workflowInstanceID]);

  useEffect(() => {
    if (open && peopleError) toast.error(peopleError);
  }, [open, peopleError]);

  const submit = async () => {
    const validationError = validateCommunicationGroupDraft(draft);
    if (validationError) return toast.error(validationError);
    setSaving(true);
    try {
      await workApi("/crm/work/save_communication_group", {
        method: "POST",
        body: JSON.stringify(communicationGroupDraftPayload(draft)),
      });
      toast.success(draft.id ? "沟通群已更新" : "沟通群已创建");
      onOpenChange(false);
      window.dispatchEvent(new CustomEvent(workRefreshEvent));
    } catch (error) {
      toast.error(errorMessage(error, "沟通群保存失败"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[88vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{draft.id ? "编辑沟通群" : "新增沟通群"}</DialogTitle>
          <DialogDescription className="sr-only">
            维护沟通群资料和关联人员
          </DialogDescription>
        </DialogHeader>

        <CommunicationGroupForm
          draft={draft}
          groupTypes={groupTypes}
          peopleOptions={peopleOptions}
          peopleLoading={peopleLoading}
          existingGroup={group}
          disabled={saving}
          onChange={setDraft}
        />

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={saving}
            onClick={() => onOpenChange(false)}
          >
            取消
          </Button>
          <Button type="button" disabled={saving} onClick={() => void submit()}>
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function CommunicationGroupDissolveDialog({
  group,
  onOpenChange,
}: {
  group: WorkCommunicationGroup | null;
  onOpenChange: (open: boolean) => void;
}) {
  const [dissolvedAt, setDissolvedAt] = useState(communicationGroupToday());
  const [reason, setReason] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!group) return;
    setDissolvedAt(communicationGroupToday());
    setReason("");
  }, [group]);

  const submit = async () => {
    if (!group) return;
    setSaving(true);
    try {
      await workApi("/crm/work/dissolve_communication_group", {
        method: "POST",
        body: JSON.stringify({
          communication_group_id: group.id,
          dissolved_at: dissolvedAt,
          reason,
        }),
      });
      toast.success("沟通群已解散");
      onOpenChange(false);
      window.dispatchEvent(new CustomEvent(workRefreshEvent));
    } catch (error) {
      toast.error(errorMessage(error, "沟通群解散失败"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={Boolean(group)} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>解散沟通群</DialogTitle>
          <DialogDescription>
            {group?.name || "当前沟通群"}将转为历史记录，重新建群时会创建新记录。
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4">
          <CommunicationGroupField label="解散日期" required>
            <Input
              type="date"
              value={dissolvedAt}
              onChange={(event) => setDissolvedAt(event.currentTarget.value)}
            />
          </CommunicationGroupField>
          <CommunicationGroupField label="解散原因">
            <Textarea
              rows={4}
              value={reason}
              placeholder="可选"
              onChange={(event) => setReason(event.currentTarget.value)}
            />
          </CommunicationGroupField>
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={saving}
            onClick={() => onOpenChange(false)}
          >
            取消
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={saving || !dissolvedAt}
            onClick={() => void submit()}
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            确认解散
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
