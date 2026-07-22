import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Loader2, Search, UsersRound, X } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

import {
  errorMessage,
  formatWorkDate,
  textValue,
  workApi,
  workRefreshEvent,
} from "./work-core";
import type {
  DispatchActiveLead,
  DispatchActiveLeadPayload,
  DispatchBatchReassignResult,
  DispatchStaff,
} from "./work-dispatch-types";
import { WorkPagination } from "./work-pagination";

const dispatchActiveLeadPageSize = 50;

type WorkDispatchActiveLeadsProps = {
  departmentID: string;
  staff: DispatchStaff[];
};

export function WorkDispatchActiveLeads({
  departmentID,
  staff,
}: WorkDispatchActiveLeadsProps) {
  const [payload, setPayload] = useState<DispatchActiveLeadPayload>({});
  const [draftKeyword, setDraftKeyword] = useState("");
  const [keyword, setKeyword] = useState("");
  const [ownerStaffID, setOwnerStaffID] = useState("");
  const [targetStaffID, setTargetStaffID] = useState("");
  const [selectedIDs, setSelectedIDs] = useState<Set<string>>(new Set());
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const loadVersionRef = useRef(0);

  const loadLeads = useCallback(async () => {
    if (!departmentID) {
      setPayload({});
      setLoading(false);
      return;
    }
    const version = loadVersionRef.current + 1;
    loadVersionRef.current = version;
    setLoading(true);
    const query = new URLSearchParams({
      department_id: departmentID,
      page: String(page),
      page_size: String(dispatchActiveLeadPageSize),
    });
    if (keyword) query.set("keyword", keyword);
    if (ownerStaffID) query.set("owner_staff_id", ownerStaffID);
    try {
      const result = await workApi<DispatchActiveLeadPayload>(
        `/crm/work/dispatch_active_leads?${query.toString()}`,
      );
      if (loadVersionRef.current !== version) return;
      const total = Math.max(0, Number(result.total) || 0);
      const pageSize = Math.max(
        1,
        Number(result.page_size) || dispatchActiveLeadPageSize,
      );
      const totalPages = Math.max(1, Math.ceil(total / pageSize));
      if (page > totalPages) {
        setPage(totalPages);
        return;
      }
      setPayload(result);
    } catch (error) {
      if (loadVersionRef.current === version) {
        toast.error(errorMessage(error, "在办线索加载失败"));
      }
    } finally {
      if (loadVersionRef.current === version) setLoading(false);
    }
  }, [departmentID, keyword, ownerStaffID, page]);

  useEffect(() => {
    void loadLeads();
  }, [loadLeads]);

  const rows = Array.isArray(payload.list) ? payload.list : [];
  const ownerOptions = Array.isArray(payload.owner_options)
    ? payload.owner_options
    : staff;
  const pageIDs = useMemo(
    () => rows.map((row) => textValue(row.workflow_instance_id)).filter(Boolean),
    [rows],
  );
  const allPageSelected =
    pageIDs.length > 0 && pageIDs.every((id) => selectedIDs.has(id));
  const targetName = textValue(
    staff.find((person) => textValue(person.id) === targetStaffID)?.name,
  );

  const applySearch = () => {
    setKeyword(draftKeyword.trim());
    setPage(1);
    setSelectedIDs(new Set());
  };

  const togglePage = (checked: boolean) => {
    setSelectedIDs((current) => {
      const next = new Set(current);
      pageIDs.forEach((id) => {
        if (checked) next.add(id);
        else next.delete(id);
      });
      return next;
    });
  };

  const toggleLead = (instanceID: string, checked: boolean) => {
    setSelectedIDs((current) => {
      const next = new Set(current);
      if (checked) next.add(instanceID);
      else next.delete(instanceID);
      return next;
    });
  };

  const batchReassign = async () => {
    if (!departmentID || !targetStaffID || selectedIDs.size === 0 || submitting) {
      return;
    }
    if (!window.confirm(`确认将已选 ${selectedIDs.size} 条线索改派给“${targetName}”？`)) {
      return;
    }
    setSubmitting(true);
    try {
      const result = await workApi<DispatchBatchReassignResult>(
        "/crm/work/batch_reassign_dispatch_leads",
        {
          method: "POST",
          body: JSON.stringify({
            department_id: departmentID,
            workflow_instance_ids: Array.from(selectedIDs),
            owner_staff_id: targetStaffID,
          }),
        },
      );
      const selectedCount = Math.max(
        0,
        Number(result.selected_count) || selectedIDs.size,
      );
      const changedCount = Math.max(0, Number(result.changed_count) || 0);
      toast.success(
        changedCount === selectedCount
          ? `已改派 ${changedCount} 条线索`
          : `已处理 ${selectedCount} 条线索，其中改派 ${changedCount} 条`,
      );
      setSelectedIDs(new Set());
      await loadLeads();
      window.dispatchEvent(new CustomEvent(workRefreshEvent));
    } catch (error) {
      toast.error(errorMessage(error, "批量改派失败"));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="crm-dispatch-active">
      <div className="crm-dispatch-section-head">
        <div>
          <strong>在办线索</strong>
          <span> · {Math.max(0, Number(payload.total) || 0)} 条</span>
        </div>
        <span>人工改派不计每日上限</span>
      </div>

      <div className="crm-dispatch-active-toolbar">
        <div className="crm-dispatch-active-filters">
          <div className="relative min-w-0 flex-1">
            <Search
              size={15}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              className="pl-9"
              value={draftKeyword}
              placeholder="搜索姓名、编号、手机号或微信号"
              onChange={(event) => setDraftKeyword(event.currentTarget.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") applySearch();
              }}
            />
          </div>
          <select
            value={ownerStaffID}
            aria-label="筛选原负责人"
            onChange={(event) => {
              setOwnerStaffID(event.currentTarget.value);
              setPage(1);
              setSelectedIDs(new Set());
            }}
          >
            <option value="">全部负责人</option>
            {ownerOptions.map((person) => (
              <option key={textValue(person.id)} value={textValue(person.id)}>
                {textValue(person.name)}
                {Number(person.status) === 2 ? "（已停用）" : ""}
              </option>
            ))}
          </select>
          <Button type="button" variant="outline" onClick={applySearch} disabled={loading}>
            <Search size={15} />
            搜索
          </Button>
        </div>

        <div className="crm-dispatch-active-batch">
          <span>已选 {selectedIDs.size} 条</span>
          {selectedIDs.size > 0 ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setSelectedIDs(new Set())}
            >
              <X size={14} />
              清空
            </Button>
          ) : null}
          <select
            value={targetStaffID}
            aria-label="选择新负责人"
            onChange={(event) => setTargetStaffID(event.currentTarget.value)}
          >
            <option value="">选择新负责人</option>
            {staff.map((person) => (
              <option key={textValue(person.id)} value={textValue(person.id)}>
                {textValue(person.name)}
              </option>
            ))}
          </select>
          <Button
            type="button"
            disabled={selectedIDs.size === 0 || !targetStaffID || submitting}
            onClick={() => void batchReassign()}
          >
            {submitting ? <Loader2 className="animate-spin" size={15} /> : <UsersRound size={15} />}
            批量改派
          </Button>
        </div>
      </div>

      <div className="crm-dispatch-active-list">
        <div className="crm-dispatch-active-row is-head">
          <input
            type="checkbox"
            aria-label="全选当前页"
            checked={allPageSelected}
            onChange={(event) => togglePage(event.currentTarget.checked)}
          />
          <span>线索</span>
          <span>阶段</span>
          <span>当前负责人</span>
          <span>进入时间</span>
        </div>
        {loading && rows.length === 0 ? (
          <div className="crm-dispatch-empty">
            <Loader2 className="animate-spin" size={22} />
          </div>
        ) : rows.length ? (
          rows.map((row) => {
            const instanceID = textValue(row.workflow_instance_id);
            return (
              <DispatchActiveLeadRow
                key={instanceID}
                row={row}
                checked={selectedIDs.has(instanceID)}
                onCheckedChange={(checked) => toggleLead(instanceID, checked)}
              />
            );
          })
        ) : (
          <div className="crm-dispatch-empty">
            <p>当前没有符合条件的在办线索</p>
          </div>
        )}
      </div>

      <WorkPagination
        loading={loading}
        page={Math.max(1, Number(payload.page) || page)}
        pageSize={Math.max(
          1,
          Number(payload.page_size) || dispatchActiveLeadPageSize,
        )}
        total={Math.max(0, Number(payload.total) || 0)}
        onPageChange={setPage}
      />
    </section>
  );
}

function DispatchActiveLeadRow({
  row,
  checked,
  onCheckedChange,
}: {
  row: DispatchActiveLead;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <div className="crm-dispatch-active-row">
      <input
        type="checkbox"
        aria-label={`选择${textValue(row.lead_name) || "线索"}`}
        checked={checked}
        onChange={(event) => onCheckedChange(event.currentTarget.checked)}
      />
      <div className="crm-dispatch-active-lead">
        <strong>{textValue(row.lead_name) || "未命名"}</strong>
        <small>
          {[row.lead_code, row.phone].map(textValue).filter(Boolean).join(" · ")}
        </small>
      </div>
      <span>{textValue(row.stage_name) || "-"}</span>
      <span>{textValue(row.owner_staff_name) || "未分配"}</span>
      <span>{formatWorkDate(row.started_at)}</span>
    </div>
  );
}
