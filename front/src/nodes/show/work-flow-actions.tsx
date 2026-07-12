import { useState } from "react";
import { ArrowRight, Ban, Check, GitBranch, Loader2, UserRound, X } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";

import {
  displayText,
  errorMessage,
  formatWorkDate,
  inputClassName,
  textValue,
  workApi,
  workRefreshEvent,
  type WorkFlowAssignee,
  type WorkFlowDetail,
  type WorkTask,
} from "./work-core";

type WorkFlowPickerKind = "task" | "owner" | "complete";

type WorkFlowPicker = {
  kind: WorkFlowPickerKind;
  todoID?: string;
  title: string;
};

type WorkFlowAssigneeResponse = {
  list?: WorkFlowAssignee[];
  department_name?: string;
  assignment_mode?: string;
};

export function WorkFlowActions({
  flow,
  loading = false,
}: {
  flow?: WorkFlowDetail | null;
  loading?: boolean;
}) {
  const [picker, setPicker] = useState<WorkFlowPicker | null>(null);
  const [assignees, setAssignees] = useState<WorkFlowAssignee[]>([]);
  const [selectedStaffID, setSelectedStaffID] = useState("");
  const [loadingAssignees, setLoadingAssignees] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [terminating, setTerminating] = useState(false);
  const [terminateReason, setTerminateReason] = useState("");

  if (loading && !flow) {
    return (
      <section className="flex min-h-36 items-center justify-center rounded-md border border-border/70 bg-background">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </section>
    );
  }

  if (!flow || textValue(flow.status) === "not_started") {
    return (
      <section className="rounded-md border border-border/70 bg-background px-4 py-10 text-center text-sm text-muted-foreground">
        暂无流程
      </section>
    );
  }

  const assetID = textValue(flow.asset_id);
  const tasks = Array.isArray(flow.tasks) ? flow.tasks : [];
  const pendingRequiredCount = Number(flow.pending_required_count) || 0;
  const status = textValue(flow.status);
  const active = status === "active";

  const closePicker = () => {
    setPicker(null);
    setAssignees([]);
    setSelectedStaffID("");
  };

  const loadAssignees = async (
    nextPicker: WorkFlowPicker,
    query: URLSearchParams,
  ) => {
    setTerminating(false);
    setPicker(nextPicker);
    setAssignees([]);
    setSelectedStaffID("");
    setLoadingAssignees(true);
    try {
      const payload = await workApi<WorkFlowAssigneeResponse>(
        `/crm/work/flow_assignees?${query.toString()}`,
      );
      setAssignees(Array.isArray(payload.list) ? payload.list : []);
    } catch (error) {
      closePicker();
      toast.error(errorMessage(error, "负责人加载失败"));
    } finally {
      setLoadingAssignees(false);
    }
  };

  const openTaskPicker = (task: WorkTask) => {
    const todoID = textValue(task.todo_id);
    if (!todoID) return;
    void loadAssignees(
      {
        kind: "task",
        todoID,
        title: task.can_reassign ? "改派任务" : "分配任务",
      },
      new URLSearchParams({ todo_id: todoID }),
    );
  };

  const openOwnerPicker = () => {
    if (!assetID) return;
    void loadAssignees(
      { kind: "owner", title: "更换阶段负责人" },
      new URLSearchParams({ asset_id: assetID, target: "current_owner" }),
    );
  };

  const openCompletePicker = () => {
    if (!assetID) return;
    void loadAssignees(
      { kind: "complete", title: "选择下一阶段负责人" },
      new URLSearchParams({ asset_id: assetID, target: "next_stage" }),
    );
  };

  const finishAction = (message: string) => {
    closePicker();
    setTerminating(false);
    setTerminateReason("");
    toast.success(message);
    window.dispatchEvent(new CustomEvent(workRefreshEvent));
  };

  const submitPicker = async () => {
    if (!picker || !selectedStaffID || !assetID) return;
    setSubmitting(true);
    try {
      if (picker.kind === "task") {
        await workApi("/crm/work/assign_flow_task", {
          method: "POST",
          body: JSON.stringify({
            todo_id: picker.todoID,
            assignee_staff_id: selectedStaffID,
          }),
        });
        finishAction("任务已分配");
      } else if (picker.kind === "owner") {
        await workApi("/crm/work/change_flow_owner", {
          method: "POST",
          body: JSON.stringify({
            asset_id: assetID,
            owner_staff_id: selectedStaffID,
          }),
        });
        finishAction("负责人已更新");
      } else {
        await workApi("/crm/work/complete_flow_stage", {
          method: "POST",
          body: JSON.stringify({
            asset_id: assetID,
            next_owner_staff_id: selectedStaffID,
          }),
        });
        finishAction("阶段已完成");
      }
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const completeStage = async () => {
    if (!assetID || !flow.ready_to_complete) return;
    if (flow.next_owner_required) {
      openCompletePicker();
      return;
    }
    const target = flow.next_terminal
      ? "完成当前阶段并结束流程？"
      : `完成当前阶段并进入“${displayText(flow.next_stage_name)}”？`;
    if (!window.confirm(target)) return;
    setSubmitting(true);
    try {
      await workApi("/crm/work/complete_flow_stage", {
        method: "POST",
        body: JSON.stringify({ asset_id: assetID }),
      });
      finishAction("阶段已完成");
    } catch (error) {
      toast.error(errorMessage(error, "阶段完成失败"));
    } finally {
      setSubmitting(false);
    }
  };

  const terminateFlow = async () => {
    const reason = terminateReason.trim();
    if (!assetID || !reason) return;
    setSubmitting(true);
    try {
      await workApi("/crm/work/terminate_flow", {
        method: "POST",
        body: JSON.stringify({ asset_id: assetID, reason }),
      });
      finishAction("流程已终止");
    } catch (error) {
      toast.error(errorMessage(error, "流程终止失败"));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="overflow-hidden rounded-md border border-border/70 bg-background">
      <div className="flex flex-wrap items-start justify-between gap-4 bg-muted/20 px-4 py-3.5">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <GitBranch className="h-4 w-4 shrink-0 text-muted-foreground" />
            <span className="font-semibold">{displayText(flow.workflow_name)}</span>
            <ArrowRight className="h-4 w-4 shrink-0 text-muted-foreground" />
            <span className="font-semibold">{displayText(flow.stage_name)}</span>
            <WorkFlowStatus status={status} />
          </div>
          <div className="mt-1.5 flex flex-wrap gap-x-5 gap-y-1 text-xs text-muted-foreground">
            <span>负责部门：{displayText(flow.owner_department_name)}</span>
            <span>负责人：{displayText(flow.owner_staff_name)}</span>
            <span>进入时间：{formatWorkDate(flow.started_at)}</span>
          </div>
        </div>
        {flow.can_change_owner ? (
          <Button type="button" variant="outline" size="sm" onClick={openOwnerPicker}>
            <UserRound className="h-4 w-4" />
            更换负责人
          </Button>
        ) : null}
      </div>

      {flow.configuration_error ? (
        <div className="border-t border-destructive/20 bg-destructive/5 px-4 py-2.5 text-sm text-destructive">
          {flow.configuration_error}
        </div>
      ) : null}

      <div className="divide-y divide-border/60 border-t border-border/60">
        {tasks.length > 0 ? (
          tasks.map((task) => (
            <WorkFlowTaskRow key={textValue(task.todo_id || task.id)} task={task} onAssign={openTaskPicker} />
          ))
        ) : (
          <div className="px-4 py-8 text-center text-sm text-muted-foreground">本阶段暂无任务</div>
        )}
      </div>

      {picker ? (
        <div className="border-t border-border/70 bg-muted/15 px-4 py-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="mr-1 text-sm font-medium">{picker.title}</span>
            <select
              className={`${inputClassName} h-9 min-w-[220px] max-w-full flex-1 sm:max-w-sm`}
              value={selectedStaffID}
              onChange={(event) => setSelectedStaffID(event.target.value)}
              disabled={loadingAssignees || submitting}
            >
              <option value="">请选择负责人</option>
              {assignees.map((assignee) => (
                <option key={textValue(assignee.id)} value={textValue(assignee.id)}>
                  {workFlowAssigneeLabel(assignee)}
                </option>
              ))}
            </select>
            <Button type="button" size="sm" disabled={!selectedStaffID || submitting} onClick={() => void submitPicker()}>
              {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
              确认
            </Button>
            <Button type="button" variant="ghost" size="icon" onClick={closePicker} disabled={submitting} title="取消">
              <X className="h-4 w-4" />
            </Button>
          </div>
          {!loadingAssignees && assignees.length === 0 ? (
            <div className="mt-2 text-xs text-muted-foreground">该部门暂无可用人员</div>
          ) : null}
        </div>
      ) : null}

      {terminating ? (
        <div className="border-t border-destructive/20 bg-destructive/5 px-4 py-3">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
            <label className="min-w-0 flex-1">
              <span className="mb-1 block text-xs font-medium text-destructive">终止原因</span>
              <textarea
                className="min-h-20 w-full resize-y rounded-md border border-input bg-background px-3 py-2 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/20"
                value={terminateReason}
                onChange={(event) => setTerminateReason(event.target.value)}
                disabled={submitting}
              />
            </label>
            <div className="flex shrink-0 gap-2">
              <Button type="button" variant="destructive" size="sm" disabled={!terminateReason.trim() || submitting} onClick={() => void terminateFlow()}>
                {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Ban className="h-4 w-4" />}
                确认终止
              </Button>
              <Button type="button" variant="outline" size="sm" onClick={() => setTerminating(false)} disabled={submitting}>
                取消
              </Button>
            </div>
          </div>
        </div>
      ) : null}

      {active ? (
        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border/70 px-4 py-3">
          <span className="text-xs text-muted-foreground">
            {pendingRequiredCount > 0 ? `还有 ${pendingRequiredCount} 项必做任务` : "必做任务已完成"}
          </span>
          <div className="flex flex-wrap items-center gap-2">
            {flow.can_terminate ? (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="text-destructive hover:text-destructive"
                onClick={() => {
                  closePicker();
                  setTerminating(true);
                }}
                disabled={submitting}
              >
                <Ban className="h-4 w-4" />
                终止
              </Button>
            ) : null}
            {flow.can_complete_stage ? (
              <Button type="button" size="sm" onClick={() => void completeStage()} disabled={!flow.ready_to_complete || submitting}>
                {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
                完成阶段
              </Button>
            ) : null}
          </div>
        </div>
      ) : null}
    </section>
  );
}

function WorkFlowTaskRow({ task, onAssign }: { task: WorkTask; onAssign: (task: WorkTask) => void }) {
  const status = textValue(task.todo_status || task.status);
  const required = Boolean(task.todo_required ?? task.required);
  const assignee = textValue(task.assignee_staff_name) || "待分配";
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-3 px-4 py-3">
      <div className="min-w-[180px] flex-1">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="font-medium">{displayText(task.task_name || task.name)}</span>
          {required ? <span className="text-xs text-destructive">必做</span> : null}
          <span className="text-xs text-muted-foreground">{workTaskTypeName(task.task_type)}</span>
        </div>
        <div className="mt-1 text-xs text-muted-foreground">
          {displayText(task.assignee_department_name)} / {assignee}
        </div>
        {textValue(task.result) ? (
          <div className="mt-1 text-xs text-amber-700">
            {textValue(task.result)}
          </div>
        ) : null}
      </div>
      <span className={`text-xs font-medium ${status === "done" ? "text-emerald-700" : status === "canceled" ? "text-muted-foreground" : "text-foreground"}`}>
        {workFlowTaskStatusName(status)}
      </span>
      {task.can_assign || task.can_reassign ? (
        <Button type="button" variant="outline" size="sm" onClick={() => onAssign(task)}>
          <UserRound className="h-4 w-4" />
          {task.can_reassign ? "改派" : "分配"}
        </Button>
      ) : null}
    </div>
  );
}

function WorkFlowStatus({ status }: { status: string }) {
  const labels: Record<string, string> = {
    active: "进行中",
    completed: "已完成",
    terminated: "已终止",
  };
  return <span className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">{labels[status] || status}</span>;
}

function workFlowTaskStatusName(status: string): string {
  if (status === "done") return "已完成";
  if (status === "canceled") return "已取消";
  return "待处理";
}

function workTaskTypeName(type: unknown): string {
  const labels: Record<string, string> = {
    todo: "事项",
    form: "资料",
    approval: "审核",
    rule: "自动核验",
  };
  return labels[textValue(type)] || textValue(type);
}

function workFlowAssigneeLabel(assignee: WorkFlowAssignee): string {
  const activeAssets = Number(assignee.active_asset_count) || 0;
  const pendingTasks = Number(assignee.pending_task_count) || 0;
  return `${displayText(assignee.name)}（资产 ${activeAssets} / 待办 ${pendingTasks}）`;
}
