import { useEffect, useId, useState } from "react";
import { Loader2, UserRound } from "lucide-react";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import {
  displayText,
  errorMessage,
  textValue,
  workApi,
  workRefreshEvent,
  type WorkFlowAssignee,
  type WorkFlowAssigneeResponse,
  type WorkFlowDetail,
} from "./work-core";

export type WorkFlowOwnerDialogProps = {
  flow?: WorkFlowDetail | null;
  open: boolean;
  title?: string;
  confirmLabel?: string;
  target?: "current_owner" | "next_stage";
  defaultToCurrentOwner?: boolean;
  onConfirmSelection?: (
    staffID: string,
  ) => boolean | void | Promise<boolean | void>;
  onOpenChange: (open: boolean) => void;
};

export function WorkFlowOwnerDialog({
  flow,
  open,
  title = "更换负责人",
  confirmLabel = "确认分配",
  target = "current_owner",
  defaultToCurrentOwner = true,
  onConfirmSelection,
  onOpenChange,
}: WorkFlowOwnerDialogProps) {
  const ownerSelectID = useId();
  const [assignees, setAssignees] = useState<WorkFlowAssignee[]>([]);
  const [departmentName, setDepartmentName] = useState("");
  const [selectedStaffID, setSelectedStaffID] = useState("");
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const workflowInstanceID = textValue(flow?.workflow_instance_id || flow?.id);
  const ownerStaffID = textValue(flow?.owner_staff_id);

  useEffect(() => {
    setAssignees([]);
    setDepartmentName("");
    setSelectedStaffID("");
    if (!open || !workflowInstanceID) return;

    let active = true;
    setLoading(true);
    void workApi<WorkFlowAssigneeResponse>(
      `/crm/work/flow_assignees?${new URLSearchParams({
        workflow_instance_id: workflowInstanceID,
        target,
      }).toString()}`,
    )
      .then((payload) => {
        if (!active) return;
        const list = Array.isArray(payload.list) ? payload.list : [];
        setAssignees(list);
        setDepartmentName(textValue(payload.department_name));
        if (
          defaultToCurrentOwner &&
          list.some((assignee) => textValue(assignee.id) === ownerStaffID)
        ) {
          setSelectedStaffID(ownerStaffID);
        }
      })
      .catch((error) => {
        if (active) toast.error(errorMessage(error, "负责人加载失败"));
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [defaultToCurrentOwner, open, ownerStaffID, target, workflowInstanceID]);

  const submit = async () => {
    if (!workflowInstanceID || !selectedStaffID || submitting) return;
    setSubmitting(true);
    try {
      if (onConfirmSelection) {
        const confirmed = await onConfirmSelection(selectedStaffID);
        if (confirmed !== false) onOpenChange(false);
        return;
      }
      await workApi("/crm/work/change_flow_owner", {
        method: "POST",
        body: JSON.stringify({
          workflow_instance_id: workflowInstanceID,
          owner_staff_id: selectedStaffID,
        }),
      });
      toast.success("负责人已更新");
      onOpenChange(false);
      window.dispatchEvent(new CustomEvent(workRefreshEvent));
    } catch (error) {
      toast.error(errorMessage(error, "负责人更新失败"));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => !submitting && onOpenChange(nextOpen)}
    >
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {target === "next_stage"
              ? `${displayText(flow?.next_workflow_name || flow?.workflow_name)} · ${displayText(flow?.next_stage_name)}`
              : `${displayText(flow?.workflow_name)} · ${displayText(flow?.stage_name)}`}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <label className="block text-sm font-medium" htmlFor={ownerSelectID}>
            负责人
          </label>
          <Select
            value={selectedStaffID}
            onValueChange={setSelectedStaffID}
            disabled={loading || submitting || assignees.length === 0}
          >
            <SelectTrigger id={ownerSelectID} className="w-full">
              <SelectValue
                placeholder={
                  loading
                    ? "正在加载..."
                    : assignees.length === 0
                      ? "当前部门暂无可用人员"
                      : "请选择负责人"
                }
              />
            </SelectTrigger>
            <SelectContent position="popper">
              {assignees
                .filter((assignee) => textValue(assignee.id))
                .map((assignee) => (
                  <SelectItem
                    key={textValue(assignee.id)}
                    value={textValue(assignee.id)}
                  >
                    {workFlowAssigneeLabel(assignee)}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
          <p className="text-xs text-muted-foreground">
            {target === "next_stage" ? "下一阶段部门：" : "当前部门："}
            {displayText(
              departmentName ||
                (target === "next_stage"
                  ? undefined
                  : flow?.owner_department_name),
              "未设置",
            )}
            {target === "current_owner" && flow?.owner_staff_name
              ? ` · 当前负责人：${displayText(flow.owner_staff_name)}`
              : ""}
          </p>
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={submitting}
            onClick={() => onOpenChange(false)}
          >
            取消
          </Button>
          <Button
            type="button"
            disabled={loading || submitting || !selectedStaffID}
            onClick={() => void submit()}
          >
            {submitting ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <UserRound className="h-4 w-4" />
            )}
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function workFlowAssigneeLabel(assignee: WorkFlowAssignee): string {
  const activeFlows =
    Number(assignee.active_flow_count ?? assignee.active_asset_count) || 0;
  const pendingTasks = Number(assignee.pending_task_count) || 0;
  return `${displayText(assignee.name)}（在办流程 ${activeFlows} · 待办任务 ${pendingTasks}）`;
}
