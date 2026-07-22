import { useMemo, useState } from "react";
import { Loader2, UsersRound } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";

import {
  errorMessage,
  formatWorkDate,
  textValue,
  workApi,
  workRefreshEvent,
} from "./work-core";
import type {
  DispatchAssignResult,
  DispatchStaff,
  PendingDispatchRow,
} from "./work-dispatch-types";

type WorkDispatchPendingLeadsProps = {
  rows: PendingDispatchRow[];
  assignees: DispatchStaff[];
  sourceStageName: string;
  targetStageName: string;
  onBeforeAssign: () => boolean;
  onAssigned: () => Promise<void>;
};

export function WorkDispatchPendingLeads({
  rows,
  assignees,
  sourceStageName,
  targetStageName,
  onBeforeAssign,
  onAssigned,
}: WorkDispatchPendingLeadsProps) {
  const [selectedIDs, setSelectedIDs] = useState<Set<string>>(new Set());
  const [batchStaffID, setBatchStaffID] = useState("");
  const [rowStaffIDs, setRowStaffIDs] = useState<Record<string, string>>({});
  const [submittingKey, setSubmittingKey] = useState("");

  const rowIDs = useMemo(
    () => rows.map((row) => textValue(row.handoff_id || row.id)).filter(Boolean),
    [rows],
  );
  const allSelected = rowIDs.length > 0 && rowIDs.every((id) => selectedIDs.has(id));

  const toggleAll = (checked: boolean) => {
    setSelectedIDs(checked ? new Set(rowIDs) : new Set());
  };

  const toggleRow = (id: string, checked: boolean) => {
    setSelectedIDs((current) => {
      const next = new Set(current);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  };

  const assign = async (handoffIDs: string[], staffID: string, key: string) => {
    if (!handoffIDs.length || !staffID || submittingKey) return;
    if (!onBeforeAssign()) return;
    setSubmittingKey(key);
    try {
      const result = await workApi<DispatchAssignResult>("/crm/work/assign_lead_dispatch", {
        method: "POST",
        body: JSON.stringify({
          handoff_ids: handoffIDs,
          assignee_staff_id: staffID,
        }),
      });
      toast.success(`已派单 ${Number(result.assigned_count) || handoffIDs.length} 条线索`);
      setSelectedIDs(new Set());
      setBatchStaffID("");
      setRowStaffIDs({});
      await onAssigned();
      window.dispatchEvent(new CustomEvent(workRefreshEvent));
    } catch (error) {
      toast.error(errorMessage(error, "线索派单失败"));
    } finally {
      setSubmittingKey("");
    }
  };

  return (
    <section className="crm-dispatch-pending">
      <div className="crm-dispatch-section-head">
        <div><strong>待派单</strong><span> · {rows.length} 条</span></div>
        <span>未找到可用接单人员的线索停留在这里</span>
      </div>

      {rows.length ? (
        <>
          <div className="crm-dispatch-pending-toolbar">
            <span>已选 {selectedIDs.size} 条</span>
            <select
              aria-label="批量选择接单人员"
              value={batchStaffID}
              onChange={(event) => setBatchStaffID(event.currentTarget.value)}
            >
              <option value="">选择接单人员</option>
              {assignees.map((person) => (
                <option key={textValue(person.id)} value={textValue(person.id)}>
                  {textValue(person.name)}
                </option>
              ))}
            </select>
            <Button
              type="button"
              size="sm"
              disabled={!selectedIDs.size || !batchStaffID || Boolean(submittingKey)}
              onClick={() => void assign([...selectedIDs], batchStaffID, "batch")}
            >
              {submittingKey === "batch" ? <Loader2 className="animate-spin" size={14} /> : <UsersRound size={14} />}
              批量派单
            </Button>
          </div>

          <div className="crm-dispatch-pending-list">
            <div className="crm-dispatch-pending-row is-head">
              <input
                type="checkbox"
                aria-label="全选待派单线索"
                checked={allSelected}
                onChange={(event) => toggleAll(event.currentTarget.checked)}
              />
              <span>线索</span>
              <span>流转</span>
              <span>等待时间</span>
              <span>接单人员</span>
              <span>操作</span>
            </div>
            {rows.map((row) => {
              const handoffID = textValue(row.handoff_id || row.id);
              const rowKey = `row:${handoffID}`;
              return (
                <div key={handoffID} className="crm-dispatch-pending-row">
                  <input
                    type="checkbox"
                    aria-label={`选择${textValue(row.lead_name) || "线索"}`}
                    checked={selectedIDs.has(handoffID)}
                    onChange={(event) => toggleRow(handoffID, event.currentTarget.checked)}
                  />
                  <div className="crm-dispatch-pending-lead">
                    <strong>{textValue(row.lead_name) || "未命名"}</strong>
                    <small>{[row.lead_code, row.phone].map(textValue).filter(Boolean).join(" · ")}</small>
                  </div>
                  <span>{sourceStageName} → {targetStageName}</span>
                  <span>{formatWorkDate(row.created_at)}</span>
                  <select
                    aria-label="选择接单人员"
                    value={rowStaffIDs[handoffID] || ""}
                    onChange={(event) => setRowStaffIDs((current) => ({
                      ...current,
                      [handoffID]: event.currentTarget.value,
                    }))}
                  >
                    <option value="">选择人员</option>
                    {assignees.map((person) => (
                      <option key={textValue(person.id)} value={textValue(person.id)}>
                        {textValue(person.name)}
                      </option>
                    ))}
                  </select>
                  <Button
                    type="button"
                    size="sm"
                    disabled={!rowStaffIDs[handoffID] || Boolean(submittingKey)}
                    onClick={() => void assign([handoffID], rowStaffIDs[handoffID], rowKey)}
                  >
                    {submittingKey === rowKey ? <Loader2 className="animate-spin" size={14} /> : "派单"}
                  </Button>
                </div>
              );
            })}
          </div>
        </>
      ) : (
        <div className="crm-dispatch-empty"><p>当前没有待派单线索</p></div>
      )}
    </section>
  );
}
