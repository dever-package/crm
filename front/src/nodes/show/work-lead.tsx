import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import {
  Ban,
  ClipboardList,
  Eye,
  Loader2,
  Plus,
  RefreshCw,
} from "lucide-react";
import { toast } from "sonner";

import { AssistantContextFormFillButton } from "@/components/assistant/form-actions";
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
import { Textarea } from "@/components/ui/textarea";
import type {
  AssistantFieldContext,
  AssistantPageContext,
} from "@/lib/assistant/context";

import {
  displayText,
  errorMessage,
  formatWorkDate,
  readWorkListSearch,
  setWorkModalOpen,
  setWorkStoreValue,
  textValue,
  workApi,
  workListSearchEvent,
  workRefreshEvent,
  type WorkCustomer,
  type WorkDetailField,
  type WorkDetailSection,
  type WorkFlowDetail,
  type WorkNodeProps,
  type WorkOperation,
  type WorkTask,
} from "./work-core";
import {
  openWorkLeadTask,
  WorkCustomerOperationTimeline,
} from "./work-auth";
import { WorkListState } from "./work-list-state";
import {
  WorkCustomerDetailStyles,
  WorkDetailSectionsData,
  WorkDetailTabs,
  workDetailValueEmpty,
  type WorkDetailTab,
} from "./work-customer-detail";
import { useWorkFeedbackModalHeaderTarget } from "./work-feedback-modal";
import { WorkPagination } from "./work-pagination";
import { useWorkTaskStoreValue } from "./work-task-form-fields";
import {
  initialWorkLeadTemplateValues,
  workLeadTemplateFieldKey,
  WorkLeadTemplateFields,
  type WorkLeadTemplate,
  type WorkLeadTemplateField,
} from "./work-lead-template-fields";

type WorkLeadOption = {
  id?: string | number;
  name?: string;
};

type WorkLead = {
  id?: string | number;
  code?: string;
  name?: string;
  phone?: string;
  wechat?: string;
  source_id?: string | number;
  source_name?: string;
  channel_id?: string | number;
  channel_name?: string;
  external_id?: string;
  city?: string;
  initial_need?: string;
  status?: string;
  status_name?: string;
  duplicate_reason?: string;
  invalid_reason_name?: string;
  invalid_note?: string;
  customer_id?: string | number;
  customer_code?: string;
  customer_code_display?: string;
  customer_name?: string;
  workflow_id?: string | number;
  workflow_instance_id?: string | number;
  workflow_name?: string;
  workflow_status?: string;
  stage_name?: string;
  owner_staff_name?: string;
  created_at?: string;
  data_values?: Record<string, unknown>;
  flow?: WorkFlowDetail;
};

type WorkLeadPoolResponse = {
  enabled?: boolean;
  can_create?: boolean;
  pending_count?: string | number;
  list?: WorkLead[];
  total?: string | number;
  page?: string | number;
  page_size?: string | number;
  sources?: WorkLeadOption[];
  channels?: WorkLeadOption[];
  invalid_reasons?: WorkLeadOption[];
  statuses?: WorkLeadOption[];
  templates?: WorkLeadTemplate[];
};

const workLeadPageSize = 30;

type LeadDraft = {
  name: string;
  phone: string;
  wechat: string;
  sourceID: string;
  channelID: string;
  externalID: string;
  city: string;
  initialNeed: string;
  dataValues: Record<string, unknown>;
};

type LeadDraftTextKey = Exclude<keyof LeadDraft, "dataValues">;

const workLeadAssistantCoreFields: ReadonlyArray<{
  key: LeadDraftTextKey;
  name: string;
  type: string;
  placeholder?: string;
  required?: boolean;
}> = [
  {
    key: "name",
    name: "姓名",
    type: "form-input",
    placeholder: "请输入姓名",
    required: true,
  },
  {
    key: "phone",
    name: "手机号",
    type: "form-input",
    placeholder: "请输入手机号",
  },
  {
    key: "wechat",
    name: "微信号",
    type: "form-input",
    placeholder: "请输入微信号",
  },
  {
    key: "city",
    name: "城市",
    type: "form-input",
    placeholder: "请输入城市",
  },
  { key: "sourceID", name: "来源", type: "form-select" },
  { key: "channelID", name: "渠道", type: "form-select" },
  {
    key: "externalID",
    name: "外部线索ID",
    type: "form-input",
    placeholder: "请输入外部线索ID",
  },
  {
    key: "initialNeed",
    name: "初始诉求",
    type: "form-textarea",
    placeholder: "请输入初始诉求",
  },
];

const emptyLeadDraft: LeadDraft = {
  name: "",
  phone: "",
  wechat: "",
  sourceID: "",
  channelID: "",
  externalID: "",
  city: "",
  initialNeed: "",
  dataValues: {},
};

type WorkLeadEditorTarget = {
  lead?: WorkLead | null;
  sources?: WorkLeadOption[];
  channels?: WorkLeadOption[];
  templates?: WorkLeadTemplate[];
  workflowID?: string;
};

const emptyWorkLeadEditorTarget: WorkLeadEditorTarget = {};
const workLeadEditorTargetPath = "data.actionTarget.workLeadEditor";
const workLeadEditorModalKey = "dialog.workLeadEditor";

type WorkLeadDetailTarget = {
  lead?: WorkLead | null;
  options?: WorkLeadPoolResponse;
};

type WorkLeadOperationsResponse = {
  list?: WorkOperation[];
};

const emptyWorkLeadDetailTarget: WorkLeadDetailTarget = {};
const workLeadDetailTargetPath = "data.actionTarget.workLeadDetail";
const workLeadDetailDrawerKey = "drawer.workLeadDetail";

export function ShowCrmWorkLeadPool({ store }: WorkNodeProps = {}) {
  const routeQuery = new URLSearchParams(window.location.search);
  const workflowID = routeQuery.get("workflow_id") || "";
  const routeKeyword = routeQuery.get("keyword") || "";
  const routeQuickFilter = routeQuery.get("quick_filter") || "";
  const routeStageFilter = routeQuery.get("stage_filter") || "";
  const routeTaskFilter = routeQuery.get("task_filter") || "";
  const [leads, setLeads] = useState<WorkLead[]>([]);
  const [options, setOptions] = useState<WorkLeadPoolResponse>({});
  const [activeKeyword, setActiveKeyword] = useState(routeKeyword);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [invalidLead, setInvalidLead] = useState<WorkLead | null>(null);
  const [invalidating, setInvalidating] = useState(false);

  const loadLeads = useCallback(async () => {
    setLoading(true);
    try {
      const query = new URLSearchParams();
      if (workflowID) query.set("workflow_id", workflowID);
      if (activeKeyword) query.set("keyword", activeKeyword);
      query.set("page", String(page));
      query.set("page_size", String(workLeadPageSize));
      if (routeQuickFilter) query.set("quick_filter", routeQuickFilter);
      if (routeStageFilter) query.set("stage_filter", routeStageFilter);
      if (routeTaskFilter) query.set("task_filter", routeTaskFilter);
      const payload = await workApi<WorkLeadPoolResponse>(
        `/crm/work/leads${query.size ? `?${query.toString()}` : ""}`,
      );
      const nextLeads = Array.isArray(payload.list) ? payload.list : [];
      const total = Math.max(0, Number(payload.total) || 0);
      const responsePageSize = Math.max(
        1,
        Number(payload.page_size) || workLeadPageSize,
      );
      const totalPages = Math.max(1, Math.ceil(total / responsePageSize));
      const responsePage = Math.max(1, Number(payload.page) || page);
      setLeads(nextLeads);
      setOptions(payload);
      if (responsePage > totalPages) setPage(totalPages);
    } catch (error) {
      toast.error(errorMessage(error, "线索池加载失败"));
    } finally {
      setLoading(false);
    }
  }, [
    activeKeyword,
    page,
    routeQuickFilter,
    routeStageFilter,
    routeTaskFilter,
    workflowID,
  ]);

  useEffect(() => {
    void loadLeads();
  }, [loadLeads]);

  useEffect(() => {
    const search = (event: Event) => {
      const detail = readWorkListSearch(event);
      if (detail.workflowID && detail.workflowID !== workflowID) return;
      setLeads([]);
      setPage(1);
      setActiveKeyword(detail.keyword);
    };
    window.addEventListener(workListSearchEvent, search);
    return () => window.removeEventListener(workListSearchEvent, search);
  }, [workflowID]);

  useEffect(() => {
    const refresh = () => {
      void loadLeads();
    };
    window.addEventListener(workRefreshEvent, refresh);
    return () => window.removeEventListener(workRefreshEvent, refresh);
  }, [loadLeads]);

  const openLeadTask = (lead: WorkLead, task: WorkTask) => {
    void openWorkLeadTask(
      task,
      workLeadTaskEntity(lead),
      store,
      lead.flow,
    );
  };

  const goToPage = (nextPage: number) => {
    if (loading || nextPage === page) return;
    setLeads([]);
    setPage(nextPage);
  };

  const invalidateLead = useCallback(
    async (reasonID: string, note: string) => {
      const leadID = textValue(invalidLead?.id);
      if (!leadID || invalidating) return;
      setInvalidating(true);
      try {
        await workApi("/crm/work/lead_action", {
          method: "POST",
          body: JSON.stringify({
            lead_id: leadID,
            workflow_id: workflowID,
            action: "invalid",
            invalid_reason_id: reasonID,
            note,
          }),
        });
        toast.success("线索已判为无效");
        setInvalidLead(null);
        window.dispatchEvent(new CustomEvent(workRefreshEvent));
      } catch (error) {
        toast.error(errorMessage(error, "线索判无效失败"));
      } finally {
        setInvalidating(false);
      }
    },
    [invalidLead?.id, invalidating, workflowID],
  );

  if (!loading && options.enabled === false) {
    return (
      <section className="bg-background px-5 py-12 text-center text-muted-foreground">
        当前账号没有线索池权限
      </section>
    );
  }

  return (
    <div className="crm-work-lead-pool space-y-4">
      <WorkLeadPoolStyles />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-muted-foreground">
          待办 {Number(options.pending_count) || 0} 项，共{" "}
          {Math.max(0, Number(options.total) || 0)} 条线索
        </p>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            title="刷新"
            onClick={() => void loadLeads()}
            disabled={loading}
          >
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
          {options.can_create ? (
            <Button
              type="button"
              onClick={() =>
                openWorkLeadEditor(store, options, workflowID, null)
              }
            >
              <Plus className="h-4 w-4" />
              录入线索
            </Button>
          ) : null}
        </div>
      </div>

      <section className="overflow-hidden bg-background">
        <div className="hidden overflow-x-auto md:block">
          <table className="crm-work-lead-table w-full min-w-[1060px] table-fixed border-collapse">
            <colgroup>
              <col style={{ width: "15%" }} />
              <col style={{ width: "13%" }} />
              <col style={{ width: "9%" }} />
              <col style={{ width: "20%" }} />
              <col style={{ width: "17%" }} />
              <col style={{ width: "11%" }} />
              <col style={{ width: "15%" }} />
            </colgroup>
            <thead className="crm-work-lead-table-head text-left">
              <tr>
                <th className="px-3 py-2.5 font-medium">线索</th>
                <th className="px-3 py-2.5 font-medium">联系方式</th>
                <th className="px-3 py-2.5 font-medium">来源</th>
                <th className="px-3 py-2.5 font-medium">诉求</th>
                <th className="px-3 py-2.5 font-medium">状态</th>
                <th className="px-3 py-2.5 font-medium">录入时间</th>
                <th className="px-3 py-2.5 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              {leads.map((lead) => (
                <WorkLeadTableRow
                  key={textValue(lead.id)}
                  lead={lead}
                  onDetail={(currentLead) =>
                    openWorkLeadDetail(
                      store,
                      options,
                      currentLead,
                    )
                  }
                  onTask={openLeadTask}
                  onInvalid={setInvalidLead}
                />
              ))}
            </tbody>
          </table>
        </div>

        <div className="md:hidden">
          {leads.map((lead) => (
            <WorkLeadMobileRow
              key={textValue(lead.id)}
              lead={lead}
              onDetail={(currentLead) =>
                openWorkLeadDetail(store, options, currentLead)
              }
              onTask={openLeadTask}
              onInvalid={setInvalidLead}
            />
          ))}
        </div>

        {!loading && leads.length === 0 ? (
          <WorkListState
            title="暂无线索"
            description="当前没有可查看的线索记录"
          />
        ) : null}
        {loading && leads.length === 0 ? (
          <WorkListState
            loading
            title="正在加载线索"
            description="正在同步最新的线索数据"
          />
        ) : null}
        <WorkPagination
          loading={loading}
          hidden={loading && leads.length === 0}
          page={Number(options.page) || page}
          pageSize={Number(options.page_size) || workLeadPageSize}
          total={Math.max(0, Number(options.total) || 0)}
          onPageChange={goToPage}
        />
      </section>

      <InvalidateLeadDialog
        lead={invalidLead}
        reasons={options.invalid_reasons || []}
        submitting={invalidating}
        onClose={() => setInvalidLead(null)}
        onSubmit={(reasonID, note) => void invalidateLead(reasonID, note)}
      />
    </div>
  );
}

function WorkLeadTableRow({
  lead,
  onDetail,
  onTask,
  onInvalid,
}: WorkLeadRowProps) {
  return (
    <tr className="crm-work-lead-row align-top">
      <td className="px-3 py-3">
        <LeadIdentity lead={lead} />
      </td>
      <td className="px-3 py-3">
        <LeadContact lead={lead} />
      </td>
      <td className="px-3 py-3">
        <div>{displayText(lead.source_name)}</div>
        <div className="mt-1 text-xs text-muted-foreground">
          {displayText(lead.channel_name)}
        </div>
      </td>
      <td className="max-w-[260px] px-3 py-3">
        <div className="line-clamp-2 break-words">
          {displayText(lead.initial_need)}
        </div>
        <div className="mt-1 text-xs text-muted-foreground">
          {displayText(lead.city)}
        </div>
      </td>
      <td className="px-3 py-3">
        <LeadStatus lead={lead} />
      </td>
      <td className="whitespace-nowrap px-3 py-3 text-muted-foreground">
        {formatWorkDate(lead.created_at)}
      </td>
      <td className="px-3 py-3">
        <LeadActions
          lead={lead}
          onDetail={onDetail}
          onTask={onTask}
          onInvalid={onInvalid}
        />
      </td>
    </tr>
  );
}

function WorkLeadMobileRow({
  lead,
  onDetail,
  onTask,
  onInvalid,
}: WorkLeadRowProps) {
  return (
    <article className="crm-work-lead-mobile-row space-y-3 px-3 py-4">
      <div className="flex min-w-0 items-start justify-between gap-3">
        <LeadIdentity lead={lead} />
        <LeadStatus lead={lead} />
      </div>
      <LeadContact lead={lead} />
      <div className="text-muted-foreground">
        {displayText(lead.source_name)} · {displayText(lead.channel_name)} ·{" "}
        {displayText(lead.city)}
      </div>
      <p className="break-words">{displayText(lead.initial_need)}</p>
      <LeadActions
        lead={lead}
        onDetail={onDetail}
        onTask={onTask}
        onInvalid={onInvalid}
      />
    </article>
  );
}

type WorkLeadRowProps = {
  lead: WorkLead;
  onDetail: (lead: WorkLead) => void;
  onTask: (lead: WorkLead, task: WorkTask) => void;
  onInvalid: (lead: WorkLead) => void;
};

function LeadIdentity({ lead }: { lead: WorkLead }) {
  return (
    <div className="min-w-0">
      <div className="break-words font-medium">{displayText(lead.name)}</div>
      <div className="mt-1 text-xs text-muted-foreground">
        {displayText(lead.code)}
      </div>
    </div>
  );
}

function LeadContact({ lead }: { lead: WorkLead }) {
  return (
    <div>
      <div>{displayText(lead.phone)}</div>
      <div className="mt-1 text-xs text-muted-foreground">
        微信 {displayText(lead.wechat)}
      </div>
    </div>
  );
}

function workLeadTaskEntity(lead: WorkLead): WorkCustomer {
  return {
    code: lead.code,
    name: lead.name,
    phone: lead.phone,
    wechat: lead.wechat,
    source_id: lead.source_id,
    channel_id: lead.channel_id,
    external_id: lead.external_id,
    city: lead.city,
    initial_need: lead.initial_need,
    data_values: lead.data_values,
  };
}

function LeadStatus({ lead }: { lead: WorkLead }) {
  const status = workLeadEffectiveStatus(lead);
  return (
    <div className="max-w-[240px]">
      <span
        className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${leadStatusClass(status)}`}
      >
        {workLeadEffectiveStatusName(lead)}
      </span>
      {lead.stage_name ? (
        <p className="mt-1 break-words text-xs text-muted-foreground">
          {lead.stage_name}
          {lead.owner_staff_name ? ` · ${lead.owner_staff_name}` : ""}
        </p>
      ) : null}
      {lead.duplicate_reason ? (
        <p className="mt-1 break-words text-xs text-amber-700">
          {lead.duplicate_reason}
        </p>
      ) : null}
      {lead.invalid_reason_name ? (
        <p className="mt-1 break-words text-xs text-muted-foreground">
          {lead.invalid_reason_name}
          {lead.invalid_note ? `：${lead.invalid_note}` : ""}
        </p>
      ) : null}
      {status === "converted" ? (
        <p className="mt-1 break-words text-xs text-muted-foreground">
          {displayText(
            lead.customer_code_display ||
              lead.customer_code ||
              lead.customer_name,
          )}
        </p>
      ) : null}
    </div>
  );
}

function LeadActions({
  lead,
  onDetail,
  onTask,
  onInvalid,
}: WorkLeadRowProps) {
  const status = workLeadEffectiveStatus(lead);
  const flow = lead.flow;
  const tasks = workLeadOperableTasks(flow);
  return (
    <div className="flex flex-wrap justify-end gap-2">
      {status === "pending" ? (
        <>
          {tasks.map((task) => (
            <Button
              key={textValue(task.todo_id || task.id)}
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onTask(lead, task)}
            >
              <ClipboardList className="h-4 w-4" />
              {displayText(task.task_name || task.name, "处理任务")}
            </Button>
          ))}
          {tasks.length > 0 ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="text-destructive hover:text-destructive"
              onClick={() => onInvalid(lead)}
            >
              <Ban className="h-4 w-4" />
              判无效
            </Button>
          ) : null}
        </>
      ) : null}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => onDetail(lead)}
      >
        <Eye className="h-4 w-4" />
        详情
      </Button>
    </div>
  );
}

function InvalidateLeadDialog({
  lead,
  reasons,
  submitting,
  onClose,
  onSubmit,
}: {
  lead: WorkLead | null;
  reasons: WorkLeadOption[];
  submitting: boolean;
  onClose: () => void;
  onSubmit: (reasonID: string, note: string) => void;
}) {
  const [reasonID, setReasonID] = useState("");
  const [note, setNote] = useState("");

  useEffect(() => {
    if (!lead) return;
    setReasonID(textValue(reasons[0]?.id));
    setNote("");
  }, [lead, reasons]);

  return (
    <Dialog
      open={Boolean(lead)}
      onOpenChange={(open) => !open && !submitting && onClose()}
    >
      <DialogContent className="crm-work-lead-invalid-dialog">
        <DialogHeader>
          <DialogTitle>判为无效线索</DialogTitle>
          <DialogDescription>
            {displayText(lead?.name)} · {displayText(lead?.phone)}
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-5 py-1">
          <LeadField label="无效原因" required>
            <Select value={reasonID} onValueChange={setReasonID}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="请选择无效原因" />
              </SelectTrigger>
              <SelectContent>
                {reasons.map((reason) => (
                  <SelectItem
                    key={textValue(reason.id)}
                    value={textValue(reason.id)}
                  >
                    {displayText(reason.name)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </LeadField>
          <LeadField label="补充说明">
            <Textarea
              className="min-h-24 resize-y"
              placeholder="请输入补充说明"
              value={note}
              onChange={(event) => setNote(event.target.value)}
            />
          </LeadField>
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={submitting}
            onClick={onClose}
          >
            取消
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={submitting || !reasonID}
            onClick={() => onSubmit(reasonID, note)}
          >
            {submitting ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Ban className="h-4 w-4" />
            )}
            确认无效
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function workLeadOperableTasks(flow?: WorkFlowDetail): WorkTask[] {
  return (flow?.tasks || []).filter(
    (task) => task.can_operate && textValue(task.task_type) !== "rule",
  );
}

function workLeadHasCustomer(lead?: WorkLead | null): boolean {
  const customerID = Number(lead?.customer_id);
  return (
    (Number.isFinite(customerID) && customerID > 0) ||
    Boolean(
      textValue(
        lead?.customer_code_display ||
          lead?.customer_code ||
          lead?.customer_name,
      ),
    )
  );
}

function workLeadEffectiveStatus(lead?: WorkLead | null): string {
  return workLeadHasCustomer(lead) ? "converted" : textValue(lead?.status);
}

function workLeadEffectiveStatusName(lead?: WorkLead | null): string {
  return workLeadHasCustomer(lead)
    ? "已转化"
    : displayText(lead?.status_name);
}

function openWorkLeadDetail(
  store: WorkNodeProps["store"],
  options: WorkLeadPoolResponse,
  lead: WorkLead,
) {
  const target: WorkLeadDetailTarget = { lead, options };
  setWorkStoreValue(store, workLeadDetailTargetPath, target);
  setWorkStoreValue(
    store,
    "data.actionTarget.workLeadDetailTitle",
    `${displayText(lead.name)} · ${workLeadEffectiveStatusName(lead)}`,
  );
  setWorkStoreValue(
    store,
    "data.actionTarget.workLeadDetailDescription",
    `${displayText(lead.code)} · ${displayText(lead.phone)}`,
  );
  setWorkModalOpen(store, workLeadDetailDrawerKey, true);
}

export function ShowCrmWorkLeadDetail({ store }: WorkNodeProps) {
  const target = useWorkTaskStoreValue<WorkLeadDetailTarget>(
    store,
    workLeadDetailTargetPath,
    emptyWorkLeadDetailTarget,
  );
  const lead = target.lead || null;
  const options = target.options || {};
  const flow = lead?.flow;
  const workflowInstanceID = textValue(
    flow?.workflow_instance_id || flow?.id || lead?.workflow_instance_id,
  );
  const [activeTab, setActiveTab] = useState<WorkDetailTab>("records");
  const [operations, setOperations] = useState<WorkOperation[]>([]);
  const [operationsLoading, setOperationsLoading] = useState(false);
  const [operationScope, setOperationScope] = useState<"all" | "mine">(
    "all",
  );
  const sections = useMemo(
    () => (lead ? workLeadDetailSections(lead, options.templates || []) : []),
    [lead, options.templates],
  );
  useEffect(() => {
    setActiveTab("records");
    setOperationScope("all");
  }, [lead?.id]);

  useEffect(() => {
    let cancelled = false;
    setOperations([]);
    if (!workflowInstanceID) {
      setOperationsLoading(false);
      return;
    }
    setOperationsLoading(true);
    const query = new URLSearchParams({
      workflow_instance_id: workflowInstanceID,
    });
    void workApi<WorkLeadOperationsResponse>(
      `/crm/work/operations?${query.toString()}`,
    )
      .then((payload) => {
        if (!cancelled) setOperations(payload.list || []);
      })
      .catch((error) => {
        if (!cancelled) toast.error(errorMessage(error));
      })
      .finally(() => {
        if (!cancelled) setOperationsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [workflowInstanceID]);

  useEffect(() => {
    const closeOnRefresh = () => {
      setWorkModalOpen(store, workLeadDetailDrawerKey, false);
    };
    window.addEventListener(workRefreshEvent, closeOnRefresh);
    return () => window.removeEventListener(workRefreshEvent, closeOnRefresh);
  }, [store]);

  if (!lead) {
    return <div className="py-10 text-center text-sm text-muted-foreground">暂无线索详情</div>;
  }

  return (
    <div className="grid gap-5">
      <WorkCustomerDetailStyles />
      <WorkDetailTabs activeTab={activeTab} onChange={setActiveTab} />

      <div>
        {activeTab === "records" ? (
          <WorkCustomerOperationTimeline
            operations={operations}
            loading={operationsLoading}
            scope={operationScope}
            onScopeChange={setOperationScope}
            store={store}
            loadingText="正在加载记录"
            emptyText="暂无记录"
          />
        ) : (
          <WorkDetailSectionsData sections={sections} />
        )}
      </div>
    </div>
  );
}

function workLeadDetailSections(
  lead: WorkLead,
  templates: WorkLeadTemplate[],
): WorkDetailSection[] {
  const baseFields: WorkDetailField[] = [
    workLeadDetailField("name", "姓名", lead.name),
    workLeadDetailField("phone", "手机号", lead.phone),
    workLeadDetailField("wechat", "微信号", lead.wechat),
    workLeadDetailField("city", "城市", lead.city),
    workLeadDetailField("source", "来源", lead.source_name),
    workLeadDetailField("channel", "渠道", lead.channel_name),
    workLeadDetailField("external", "外部线索ID", lead.external_id),
    workLeadDetailField("created", "录入时间", formatWorkDate(lead.created_at)),
    workLeadDetailField("need", "初始诉求", lead.initial_need),
  ];
  if (lead.invalid_reason_name) {
    baseFields.push(
      workLeadDetailField("invalid", "无效原因", lead.invalid_reason_name),
    );
  }
  if (lead.invalid_note) {
    baseFields.push(
      workLeadDetailField("invalid-note", "无效说明", lead.invalid_note),
    );
  }
  if (lead.duplicate_reason) {
    baseFields.push(
      workLeadDetailField("duplicate", "重复说明", lead.duplicate_reason),
    );
  }
  const customerCode =
    lead.customer_code_display || lead.customer_code || lead.customer_name;
  if (customerCode) {
    baseFields.push(
      workLeadDetailField("customer", "客户编号", customerCode),
    );
  }
  const sections = [
    workLeadDetailSection("lead:base", "线索基础信息", baseFields),
  ];
  const values = lead.data_values || {};

  templates.forEach((template, templateIndex) => {
    const fields = (template.fields || []).flatMap((field) => {
      const key = workLeadTemplateFieldKey(field);
      if (!key) return [];
      const value = values[key];
      const fileValue = workLeadDetailFieldIsFile(field) ? value : undefined;
      return [
        workLeadDetailField(
          key,
          displayText(field.name, "扩展字段"),
          workLeadDetailTemplateValue(field, value),
          field.group_name,
          fileValue,
        ),
      ];
    });
    if (fields.length === 0) return;
    sections.push(
      workLeadDetailSection(
        `lead:template:${textValue(template.id) || templateIndex}`,
        displayText(template.name, "线索扩展信息"),
        fields,
        template.id,
      ),
    );
  });

  return sections;
}

function workLeadDetailSection(
  id: string,
  name: string,
  fields: WorkDetailField[],
  templateId?: string | number,
): WorkDetailSection {
  const filled = fields.filter((field) => !field.empty).length;
  const total = fields.length;
  return {
    id,
    name,
    targetType: "lead",
    templateId,
    filled,
    total,
    percent: total ? Math.round((filled / total) * 100) : 0,
    fields,
  };
}

function workLeadDetailField(
  key: string,
  label: string,
  value: unknown,
  group?: string,
  files?: unknown,
): WorkDetailField {
  const fileItems = Array.isArray(files) ? files : files ? [files] : [];
  const isFile = fileItems.length > 0;
  const empty = isFile
    ? fileItems.length === 0
    : workDetailValueEmpty(value);
  return {
    key,
    label,
    value,
    group,
    valueType: isFile ? "files" : "text",
    files: fileItems,
    empty,
  };
}

function workLeadDetailFieldIsFile(field: WorkLeadTemplateField): boolean {
  return ["file", "files", "upload"].includes(textValue(field.field_type));
}

function workLeadDetailTemplateValue(
  field: WorkLeadTemplateField,
  value: unknown,
): unknown {
  if (typeof value === "boolean") return value ? "是" : "否";
  const selected = Array.isArray(value)
    ? value.map(textValue)
    : [textValue(value)];
  if (selected.length === 0 || selected.every((item) => !item)) return value;
  return selected
    .map((item) => {
      const option = (field.options || []).find(
        (current) => textValue(current.value || current.id) === item,
      );
      return displayText(option?.name || option?.value, item);
    })
    .join("、");
}

function openWorkLeadEditor(
  store: WorkNodeProps["store"],
  options: WorkLeadPoolResponse,
  workflowID: string,
  lead: WorkLead | null,
) {
  const editing = Boolean(lead?.id);
  const target: WorkLeadEditorTarget = {
    lead,
    sources: options.sources || [],
    channels: options.channels || [],
    templates: options.templates || [],
    workflowID,
  };
  setWorkStoreValue(store, workLeadEditorTargetPath, target);
  setWorkStoreValue(
    store,
    "data.actionTarget.workLeadEditorTitle",
    editing ? "编辑线索" : "录入线索",
  );
  setWorkStoreValue(
    store,
    "data.actionTarget.workLeadEditorDescription",
    editing
      ? "修改线索资料，保存时会重新检查联系方式是否重复。"
      : "录入基本联系方式和初始诉求，系统会自动检查重复。",
  );
  setWorkModalOpen(store, workLeadEditorModalKey, true);
}

export function ShowCrmWorkLeadEditorForm({ store }: WorkNodeProps) {
  const target = useWorkTaskStoreValue<WorkLeadEditorTarget>(
    store,
    workLeadEditorTargetPath,
    emptyWorkLeadEditorTarget,
  );
  const lead = target.lead || null;
  const sources = target.sources || [];
  const channels = target.channels || [];
  const templates = target.templates || [];
  const editing = Boolean(lead?.id);
  const [draft, setDraft] = useState<LeadDraft>(() =>
    buildWorkLeadDraft(target),
  );
  const [submitting, setSubmitting] = useState(false);
  const contentRef = useRef<HTMLDivElement | null>(null);
  const aiHeaderTarget = useWorkFeedbackModalHeaderTarget(contentRef);
  const assistantContext = useMemo(
    () =>
      buildWorkLeadAssistantContext({
        editing,
        draft,
        sources,
        channels,
        templates,
      }),
    [channels, draft, editing, sources, templates],
  );

  const applyAssistantValues = useCallback(
    (values: Record<string, unknown>) => {
      setDraft((current) =>
        applyWorkLeadAssistantValues(current, templates, values),
      );
    },
    [templates],
  );

  useEffect(() => {
    setDraft(buildWorkLeadDraft(target));
  }, [target]);

  const submit = useCallback(async () => {
    if (
      submitting ||
      !draft.name.trim() ||
      (!draft.phone.trim() && !draft.wechat.trim())
    ) {
      return;
    }
    setSubmitting(true);
    try {
      const payload = await workApi<{ lead?: WorkLead }>(
        editing ? "/crm/work/lead_action" : "/crm/work/create_lead",
        {
          method: "POST",
          body: JSON.stringify({
            lead_id: editing ? lead?.id : undefined,
            action: editing ? "update" : undefined,
            name: draft.name,
            phone: draft.phone,
            wechat: draft.wechat,
            source_id: draft.sourceID,
            channel_id: draft.channelID,
            external_id: draft.externalID,
            city: draft.city,
            initial_need: draft.initialNeed,
            data_values: draft.dataValues,
            workflow_id: target.workflowID,
          }),
        },
      );
      toast.success(
        editing
          ? "线索已更新"
          : textValue(payload.lead?.status) === "duplicate"
            ? "线索已录入，并标记为重复"
            : "线索已录入",
      );
      setWorkModalOpen(store, workLeadEditorModalKey, false);
      window.dispatchEvent(new CustomEvent(workRefreshEvent));
    } catch (error) {
      toast.error(
        errorMessage(error, editing ? "线索更新失败" : "线索录入失败"),
      );
    } finally {
      setSubmitting(false);
    }
  }, [draft, editing, lead?.id, store, submitting, target.workflowID]);

  useEffect(() => {
    const form = contentRef.current?.closest("form");
    if (!form) return undefined;
    const handleSubmit = (event: Event) => {
      event.preventDefault();
      event.stopPropagation();
      void submit();
    };
    form.addEventListener("submit", handleSubmit);
    return () => form.removeEventListener("submit", handleSubmit);
  }, [submit]);

  useEffect(() => {
    const form = contentRef.current?.closest("form");
    const submitButton = form?.querySelector<HTMLButtonElement>(
      'button[type="submit"]',
    );
    if (!submitButton) return undefined;
    const previousDisabled = submitButton.disabled;
    submitButton.disabled =
      submitting ||
      !draft.name.trim() ||
      (!draft.phone.trim() && !draft.wechat.trim());
    return () => {
      submitButton.disabled = previousDisabled;
    };
  }, [draft.name, draft.phone, draft.wechat, submitting]);

  return (
    <>
      {aiHeaderTarget
        ? createPortal(
            <AssistantContextFormFillButton
              context={assistantContext}
              variant="ghost"
              size="icon"
              className="crm-work-ai-icon -mt-2 size-8 shrink-0 self-start p-0 text-muted-foreground"
              disabled={submitting}
              overwrite
              onApplyValues={applyAssistantValues}
            />,
            aiHeaderTarget,
          )
        : null}
      <div ref={contentRef} className="grid gap-4 sm:grid-cols-2">
        <LeadField label="姓名" required>
          <Input
            placeholder="请输入姓名"
            value={draft.name}
            onChange={(event) =>
              setDraft({ ...draft, name: event.target.value })
            }
          />
        </LeadField>
        <LeadField label="手机号">
          <Input
            placeholder="请输入手机号"
            value={draft.phone}
            onChange={(event) =>
              setDraft({ ...draft, phone: event.target.value })
            }
          />
        </LeadField>
        <LeadField label="微信号">
          <Input
            placeholder="请输入微信号"
            value={draft.wechat}
            onChange={(event) =>
              setDraft({ ...draft, wechat: event.target.value })
            }
          />
        </LeadField>
        <LeadField label="城市">
          <Input
            placeholder="请输入城市"
            value={draft.city}
            onChange={(event) =>
              setDraft({ ...draft, city: event.target.value })
            }
          />
        </LeadField>
        <LeadField label="来源">
          <Select
            value={draft.sourceID}
            onValueChange={(sourceID) => setDraft({ ...draft, sourceID })}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="请选择来源" />
            </SelectTrigger>
            <SelectContent position="popper">
              {sources
                .filter((option) => textValue(option.id))
                .map((option) => (
                  <SelectItem
                    key={textValue(option.id)}
                    value={textValue(option.id)}
                  >
                    {displayText(option.name)}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        </LeadField>
        <LeadField label="渠道">
          <Select
            value={draft.channelID}
            onValueChange={(channelID) => setDraft({ ...draft, channelID })}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="请选择渠道" />
            </SelectTrigger>
            <SelectContent position="popper">
              {channels
                .filter((option) => textValue(option.id))
                .map((option) => (
                  <SelectItem
                    key={textValue(option.id)}
                    value={textValue(option.id)}
                  >
                    {displayText(option.name)}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        </LeadField>
        <LeadField label="外部线索ID">
          <Input
            placeholder="请输入外部线索ID"
            value={draft.externalID}
            onChange={(event) =>
              setDraft({ ...draft, externalID: event.target.value })
            }
          />
        </LeadField>
        <LeadField label="初始诉求" className="crm-work-lead-form-wide">
          <Textarea
            className="min-h-24 resize-y"
            placeholder="请输入初始诉求"
            value={draft.initialNeed}
            onChange={(event) =>
              setDraft({ ...draft, initialNeed: event.target.value })
            }
          />
        </LeadField>
        <WorkLeadTemplateFields
          templates={templates}
          values={draft.dataValues}
          onChange={(dataValues) => setDraft({ ...draft, dataValues })}
        />
      </div>
    </>
  );
}

function buildWorkLeadAssistantContext({
  editing,
  draft,
  sources,
  channels,
  templates,
}: {
  editing: boolean;
  draft: LeadDraft;
  sources: WorkLeadOption[];
  channels: WorkLeadOption[];
  templates: WorkLeadTemplate[];
}): AssistantPageContext {
  const fields: AssistantFieldContext[] = workLeadAssistantCoreFields.map(
    (field) => ({
      path: `form.${field.key}`,
      name: field.name,
      type: field.type,
      required: field.required,
      placeholder: field.placeholder,
      options: workLeadAssistantOptions(field.key, sources, channels),
    }),
  );
  const values: Record<string, unknown> = {};
  workLeadAssistantCoreFields.forEach((field) => {
    setWorkLeadAssistantCurrentValue(
      values,
      `form.${field.key}`,
      draft[field.key],
    );
  });

  templates.forEach((template) => {
    (template.fields || []).forEach((field) => {
      const key = workLeadTemplateFieldKey(field);
      if (!key) return;
      const path = `form.dataValues.${key}`;
      fields.push({
        path,
        name: field.group_name
          ? `${field.group_name} - ${displayText(field.name, "扩展字段")}`
          : displayText(field.name, "扩展字段"),
        type: workLeadAssistantTemplateFieldType(field),
        options: workLeadAssistantTemplateOptions(field),
      });
      setWorkLeadAssistantCurrentValue(values, path, draft.dataValues[key]);
    });
  });

  return {
    scope: "modal",
    route: window.location.pathname,
    page: {
      name: editing ? "编辑线索" : "录入线索",
      title: editing ? "编辑线索" : "录入线索",
    },
    form: { fields, values },
  };
}

function applyWorkLeadAssistantValues(
  current: LeadDraft,
  templates: WorkLeadTemplate[],
  values: Record<string, unknown>,
): LeadDraft {
  const next: LeadDraft = {
    ...current,
    dataValues: { ...current.dataValues },
  };
  const coreFields = new Map<string, LeadDraftTextKey>(
    workLeadAssistantCoreFields.map((field) => [
      `form.${field.key}`,
      field.key,
    ] as [string, LeadDraftTextKey]),
  );
  const templateFields = new Map<string, WorkLeadTemplateField>();
  templates.forEach((template) => {
    (template.fields || []).forEach((field) => {
      const key = workLeadTemplateFieldKey(field);
      if (key) templateFields.set(key, field);
    });
  });

  Object.entries(values).forEach(([rawPath, value]) => {
    const path = normalizeWorkLeadAssistantPath(rawPath);
    const coreKey = coreFields.get(path);
    if (coreKey) {
      next[coreKey] = textValue(value);
      return;
    }
    const dataPrefix = "form.dataValues.";
    if (!path.startsWith(dataPrefix)) return;
    const fieldKey = path.slice(dataPrefix.length);
    const field = templateFields.get(fieldKey);
    if (!field) return;
    next.dataValues[fieldKey] = normalizeWorkLeadAssistantTemplateValue(
      field,
      value,
    );
  });

  return next;
}

function workLeadAssistantOptions(
  key: LeadDraftTextKey,
  sources: WorkLeadOption[],
  channels: WorkLeadOption[],
) {
  const options =
    key === "sourceID" ? sources : key === "channelID" ? channels : [];
  if (options.length === 0) return undefined;
  return options.map((option) => ({
    id: textValue(option.id),
    value: displayText(option.name),
  }));
}

function workLeadAssistantTemplateOptions(field: WorkLeadTemplateField) {
  const options = field.options || [];
  if (options.length === 0) return undefined;
  return options.map((option) => ({
    id: textValue(option.value || option.id),
    value: displayText(option.name || option.value),
  }));
}

function workLeadAssistantTemplateFieldType(
  field: WorkLeadTemplateField,
): string {
  const fieldType = textValue(field.field_type);
  if (fieldType === "textarea") return "form-textarea";
  if (fieldType === "select" || fieldType === "radio") return "form-select";
  if (fieldType === "checkbox" || fieldType === "multi_select") {
    return "form-checkbox";
  }
  if (fieldType === "boolean") return "form-switch";
  if (fieldType === "number" || fieldType === "money") return "form-number";
  if (fieldType === "date" || fieldType === "datetime") return "form-date";
  return "form-input";
}

function normalizeWorkLeadAssistantTemplateValue(
  field: WorkLeadTemplateField,
  value: unknown,
): unknown {
  const fieldType = textValue(field.field_type);
  if (fieldType === "boolean") {
    if (typeof value === "boolean") return value;
    return ["1", "true", "yes", "是"].includes(textValue(value).toLowerCase());
  }
  if (fieldType === "checkbox" || fieldType === "multi_select") {
    if (Array.isArray(value)) return value.map(textValue).filter(Boolean);
    return textValue(value)
      .split(",")
      .map((part) => part.trim())
      .filter(Boolean);
  }
  return value;
}

function setWorkLeadAssistantCurrentValue(
  values: Record<string, unknown>,
  path: string,
  value: unknown,
) {
  if (value === undefined || value === null || value === "") return;
  if (Array.isArray(value) && value.length === 0) return;
  values[path] = value;
}

function normalizeWorkLeadAssistantPath(path: string): string {
  return String(path || "")
    .trim()
    .replace(/^data\.form\./, "form.")
    .replace(/^data\./, "");
}

function buildWorkLeadDraft(target: WorkLeadEditorTarget): LeadDraft {
  const lead = target.lead;
  const sources = target.sources || [];
  const channels = target.channels || [];
  const templates = target.templates || [];
  return {
    ...emptyLeadDraft,
    name: textValue(lead?.name),
    phone: textValue(lead?.phone),
    wechat: textValue(lead?.wechat),
    sourceID: textValue(lead?.source_id || sources[0]?.id),
    channelID: textValue(lead?.channel_id || channels[0]?.id),
    externalID: textValue(lead?.external_id),
    city: textValue(lead?.city),
    initialNeed: textValue(lead?.initial_need),
    dataValues: {
      ...initialWorkLeadTemplateValues(templates),
      ...(lead?.data_values || {}),
    },
  };
}

function WorkLeadPoolStyles() {
  return (
    <style>{`
      .crm-work-lead-pool {
        color: var(--crm-body-text, #171a19);
        font-size: 12.8px;
        line-height: 1.45;
      }

      .crm-work-lead-pool button,
      .crm-work-lead-pool input,
      .crm-work-lead-pool select,
      .crm-work-lead-pool textarea {
        font-size: 12.8px;
      }

      .crm-work-lead-table-head {
        background: var(--crm-body-bg, #f4f6f5);
      }

      .crm-work-lead-table-head {
        color: var(--crm-body-muted, #6b7370);
      }

      .crm-work-lead-table-head tr,
      .crm-work-lead-row,
      .crm-work-lead-mobile-row {
        border-bottom: 1px solid var(--crm-body-line, #e4e8e6);
      }

      .crm-work-lead-row,
      .crm-work-lead-mobile-row {
        transition: background-color 120ms ease;
      }

      .crm-work-lead-row:hover,
      .crm-work-lead-mobile-row:hover {
        background: var(--crm-body-bg, #f4f6f5);
      }

      .crm-work-lead-row:last-child,
      .crm-work-lead-mobile-row:last-child {
        border-bottom: 0;
      }

      .crm-work-lead-form-wide {
        grid-column: 1 / -1;
      }

      .crm-work-lead-invalid-dialog {
        width: min(35rem, calc(100vw - 2rem)) !important;
        max-width: 35rem !important;
      }

      .crm-work-ai-icon {
        width: 2rem !important;
        height: 2rem !important;
        padding: 0 !important;
        gap: 0 !important;
        overflow: hidden;
        font-size: 0 !important;
      }

      .crm-work-ai-icon svg {
        width: 1rem;
        height: 1rem;
        flex: 0 0 auto;
      }

    `}</style>
  );
}

function LeadField({
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
    <div className={`block min-w-0 ${className}`.trim()}>
      <span className="mb-1.5 block text-sm font-medium">
        {label}
        {required ? <span className="ml-1 text-destructive">*</span> : null}
      </span>
      {children}
    </div>
  );
}

function leadStatusClass(status: string): string {
  if (status === "converted") return "bg-emerald-50 text-emerald-700";
  if (status === "duplicate") return "bg-amber-50 text-amber-700";
  if (status === "invalid") return "bg-muted text-muted-foreground";
  return "bg-blue-50 text-blue-700";
}
