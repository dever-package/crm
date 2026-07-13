import { useCallback, useEffect, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import {
  Ban,
  CheckCircle2,
  Loader2,
  Plus,
  RefreshCw,
  RotateCcw,
  Search,
  UserRoundPlus,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

import {
  displayText,
  errorMessage,
  formatWorkDate,
  inputClassName,
  textValue,
  workApi,
  workRefreshEvent,
} from "./work-core";
import {
  initialWorkLeadTemplateValues,
  WorkLeadTemplateFields,
  type WorkLeadTemplate,
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
  customer_name?: string;
  created_at?: string;
};

type WorkLeadPoolResponse = {
  enabled?: boolean;
  list?: WorkLead[];
  total?: string | number;
  sources?: WorkLeadOption[];
  channels?: WorkLeadOption[];
  invalid_reasons?: WorkLeadOption[];
  statuses?: WorkLeadOption[];
  templates?: WorkLeadTemplate[];
};

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

export function ShowCrmWorkLeadPool() {
  const [leads, setLeads] = useState<WorkLead[]>([]);
  const [options, setOptions] = useState<WorkLeadPoolResponse>({});
  const [keyword, setKeyword] = useState("");
  const [activeKeyword, setActiveKeyword] = useState("");
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [invalidLead, setInvalidLead] = useState<WorkLead | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const loadLeads = useCallback(async () => {
    setLoading(true);
    try {
      const query = new URLSearchParams();
      if (activeKeyword) query.set("keyword", activeKeyword);
      if (status) query.set("status", status);
      const payload = await workApi<WorkLeadPoolResponse>(
        `/crm/work/leads${query.size ? `?${query.toString()}` : ""}`,
      );
      setLeads(Array.isArray(payload.list) ? payload.list : []);
      setOptions(payload);
    } catch (error) {
      toast.error(errorMessage(error, "线索池加载失败"));
    } finally {
      setLoading(false);
    }
  }, [activeKeyword, status]);

  useEffect(() => {
    void loadLeads();
  }, [loadLeads]);

  useEffect(() => {
    const refresh = () => void loadLeads();
    window.addEventListener(workRefreshEvent, refresh);
    return () => window.removeEventListener(workRefreshEvent, refresh);
  }, [loadLeads]);

  const actOnLead = async (lead: WorkLead, action: string, extra: Record<string, unknown> = {}) => {
    const leadID = textValue(lead.id);
    if (!leadID) return;
    setSubmitting(true);
    try {
      const result = await workApi<{
        converted?: boolean;
        duplicate?: boolean;
        message?: string;
      }>("/crm/work/lead_action", {
        method: "POST",
        body: JSON.stringify({ lead_id: leadID, action, ...extra }),
      });
      if (result.duplicate) {
        toast.error(result.message || "检测到重复客户或线索");
      } else {
        toast.success(leadActionSuccessText(action));
      }
      setInvalidLead(null);
      await loadLeads();
      if (result.converted) {
        window.dispatchEvent(new CustomEvent(workRefreshEvent));
      }
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  if (!loading && options.enabled === false) {
    return (
      <section className="rounded-md border border-border/70 bg-background px-5 py-12 text-center text-sm text-muted-foreground">
        当前账号没有线索池权限
      </section>
    );
  }

  return (
    <div className="space-y-4">
      <WorkLeadPoolStyles />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">待处理 {leadCountByStatus(leads, "pending")} 条，共 {Number(options.total) || leads.length} 条</p>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" size="icon" title="刷新" onClick={() => void loadLeads()} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
          <Button type="button" onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            录入线索
          </Button>
        </div>
      </div>

      <section className="overflow-hidden rounded-md border border-border/70 bg-background">
        <form
          className="crm-work-lead-search-grid grid gap-2 border-b border-border/70 bg-muted/10 px-4 py-3"
          onSubmit={(event) => {
            event.preventDefault();
            setActiveKeyword(keyword.trim());
          }}
        >
          <Input
            className="w-full"
            placeholder="姓名、手机、微信或线索编号"
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
          />
          <select className={inputClassName} value={status} onChange={(event) => setStatus(event.target.value)}>
            <option value="">全部状态</option>
            {(options.statuses || []).map((option) => (
              <option key={textValue(option.id)} value={textValue(option.id)}>{displayText(option.name)}</option>
            ))}
          </select>
          <Button type="submit" size="sm">
            <Search className="h-4 w-4" />
            搜索
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              setKeyword("");
              setActiveKeyword("");
              setStatus("");
            }}
          >
            重置
          </Button>
        </form>

        <div className="hidden overflow-x-auto md:block">
          <table className="w-full min-w-[1060px] text-sm">
            <thead className="bg-muted/20 text-left text-muted-foreground">
              <tr>
                <th className="px-4 py-3 font-medium">线索</th>
                <th className="px-4 py-3 font-medium">联系方式</th>
                <th className="px-4 py-3 font-medium">来源</th>
                <th className="px-4 py-3 font-medium">诉求</th>
                <th className="px-4 py-3 font-medium">状态</th>
                <th className="px-4 py-3 font-medium">录入时间</th>
                <th className="px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60">
              {leads.map((lead) => <WorkLeadTableRow key={textValue(lead.id)} lead={lead} submitting={submitting} onAction={actOnLead} onInvalid={setInvalidLead} />)}
            </tbody>
          </table>
        </div>

        <div className="divide-y divide-border/60 md:hidden">
          {leads.map((lead) => <WorkLeadMobileRow key={textValue(lead.id)} lead={lead} submitting={submitting} onAction={actOnLead} onInvalid={setInvalidLead} />)}
        </div>

        {!loading && leads.length === 0 ? (
          <div className="px-4 py-14 text-center text-sm text-muted-foreground">暂无线索</div>
        ) : null}
        {loading && leads.length === 0 ? (
          <div className="flex items-center justify-center px-4 py-14 text-muted-foreground"><Loader2 className="h-5 w-5 animate-spin" /></div>
        ) : null}
      </section>

      <CreateLeadDialog
        open={createOpen}
        sources={options.sources || []}
        channels={options.channels || []}
        templates={options.templates || []}
        submitting={submitting}
        onOpenChange={setCreateOpen}
        onCreated={async () => {
          setCreateOpen(false);
          await loadLeads();
        }}
        setSubmitting={setSubmitting}
      />
      <InvalidateLeadDialog
        lead={invalidLead}
        reasons={options.invalid_reasons || []}
        submitting={submitting}
        onClose={() => setInvalidLead(null)}
        onSubmit={(reasonID, note) => void actOnLead(invalidLead || {}, "invalid", { invalid_reason_id: reasonID, note })}
      />
    </div>
  );
}

function WorkLeadTableRow({ lead, submitting, onAction, onInvalid }: WorkLeadRowProps) {
  return (
    <tr className="align-top hover:bg-muted/10">
      <td className="px-4 py-3"><LeadIdentity lead={lead} /></td>
      <td className="px-4 py-3"><LeadContact lead={lead} /></td>
      <td className="px-4 py-3"><div>{displayText(lead.source_name)}</div><div className="mt-1 text-xs text-muted-foreground">{displayText(lead.channel_name)}</div></td>
      <td className="max-w-[260px] px-4 py-3"><div className="line-clamp-2 break-words">{displayText(lead.initial_need)}</div><div className="mt-1 text-xs text-muted-foreground">{displayText(lead.city)}</div></td>
      <td className="px-4 py-3"><LeadStatus lead={lead} /></td>
      <td className="whitespace-nowrap px-4 py-3 text-muted-foreground">{formatWorkDate(lead.created_at)}</td>
      <td className="px-4 py-3"><LeadActions lead={lead} submitting={submitting} onAction={onAction} onInvalid={onInvalid} /></td>
    </tr>
  );
}

function WorkLeadMobileRow({ lead, submitting, onAction, onInvalid }: WorkLeadRowProps) {
  return (
    <article className="space-y-3 px-4 py-4">
      <div className="flex min-w-0 items-start justify-between gap-3"><LeadIdentity lead={lead} /><LeadStatus lead={lead} /></div>
      <LeadContact lead={lead} />
      <div className="text-sm text-muted-foreground">{displayText(lead.source_name)} · {displayText(lead.channel_name)} · {displayText(lead.city)}</div>
      <p className="break-words text-sm">{displayText(lead.initial_need)}</p>
      <LeadActions lead={lead} submitting={submitting} onAction={onAction} onInvalid={onInvalid} />
    </article>
  );
}

type WorkLeadRowProps = {
  lead: WorkLead;
  submitting: boolean;
  onAction: (lead: WorkLead, action: string, extra?: Record<string, unknown>) => Promise<void>;
  onInvalid: (lead: WorkLead) => void;
};

function LeadIdentity({ lead }: { lead: WorkLead }) {
  return <div className="min-w-0"><div className="break-words font-medium">{displayText(lead.name)}</div><div className="mt-1 text-xs text-muted-foreground">{displayText(lead.code)}</div></div>;
}

function LeadContact({ lead }: { lead: WorkLead }) {
  return <div><div>{displayText(lead.phone)}</div><div className="mt-1 text-xs text-muted-foreground">微信 {displayText(lead.wechat)}</div></div>;
}

function LeadStatus({ lead }: { lead: WorkLead }) {
  const status = textValue(lead.status);
  return (
    <div className="max-w-[240px]">
      <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${leadStatusClass(status)}`}>{displayText(lead.status_name)}</span>
      {lead.duplicate_reason ? <p className="mt-1 break-words text-xs text-amber-700">{lead.duplicate_reason}</p> : null}
      {lead.invalid_reason_name ? <p className="mt-1 break-words text-xs text-muted-foreground">{lead.invalid_reason_name}{lead.invalid_note ? `：${lead.invalid_note}` : ""}</p> : null}
      {status === "converted" ? <p className="mt-1 break-words text-xs text-muted-foreground">{displayText(lead.customer_code || lead.customer_name)}</p> : null}
    </div>
  );
}

function LeadActions({ lead, submitting, onAction, onInvalid }: WorkLeadRowProps) {
  const status = textValue(lead.status);
  return (
    <div className="flex flex-wrap justify-end gap-2">
      {status === "pending" ? (
        <>
          <Button type="button" size="sm" disabled={submitting} onClick={() => {
            if (window.confirm(`将“${displayText(lead.name)}”转为客户并创建资产？`)) void onAction(lead, "convert");
          }}><UserRoundPlus className="h-4 w-4" />转客户</Button>
          <Button type="button" variant="outline" size="sm" disabled={submitting} onClick={() => onInvalid(lead)}><Ban className="h-4 w-4" />判无效</Button>
        </>
      ) : null}
      {status === "duplicate" || status === "invalid" ? (
        <Button type="button" variant="outline" size="sm" disabled={submitting} onClick={() => void onAction(lead, "reopen")}><RotateCcw className="h-4 w-4" />恢复</Button>
      ) : null}
      {status === "converted" ? <span className="inline-flex items-center gap-1 text-sm text-emerald-700"><CheckCircle2 className="h-4 w-4" />已转化</span> : null}
    </div>
  );
}

function CreateLeadDialog({ open, sources, channels, templates, submitting, onOpenChange, onCreated, setSubmitting }: {
  open: boolean;
  sources: WorkLeadOption[];
  channels: WorkLeadOption[];
  templates: WorkLeadTemplate[];
  submitting: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => Promise<void>;
  setSubmitting: (value: boolean) => void;
}) {
  const [draft, setDraft] = useState<LeadDraft>(emptyLeadDraft);

  useEffect(() => {
    if (!open) return;
    setDraft({
      ...emptyLeadDraft,
      sourceID: textValue(sources[0]?.id),
      channelID: textValue(channels[0]?.id),
      dataValues: initialWorkLeadTemplateValues(templates),
    });
  }, [channels, open, sources, templates]);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    try {
      const payload = await workApi<{ lead?: WorkLead }>("/crm/work/create_lead", {
        method: "POST",
        body: JSON.stringify({
          name: draft.name,
          phone: draft.phone,
          wechat: draft.wechat,
          source_id: draft.sourceID,
          channel_id: draft.channelID,
          external_id: draft.externalID,
          city: draft.city,
          initial_need: draft.initialNeed,
          data_values: draft.dataValues,
        }),
      });
      toast.success(textValue(payload.lead?.status) === "duplicate" ? "线索已录入，并标记为重复" : "线索已录入");
      await onCreated();
    } catch (error) {
      toast.error(errorMessage(error, "线索录入失败"));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !submitting && onOpenChange(nextOpen)}>
      <DialogContent className="max-w-2xl">
        <DialogHeader><DialogTitle>录入线索</DialogTitle><DialogDescription>录入基本联系方式和初始诉求，系统会自动检查重复。</DialogDescription></DialogHeader>
        <form className="grid gap-4 sm:grid-cols-2" onSubmit={(event) => void submit(event)}>
          <LeadField label="姓名" required><Input placeholder="请输入姓名" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} /></LeadField>
          <LeadField label="手机号"><Input placeholder="请输入手机号" value={draft.phone} onChange={(event) => setDraft({ ...draft, phone: event.target.value })} /></LeadField>
          <LeadField label="微信号"><Input placeholder="请输入微信号" value={draft.wechat} onChange={(event) => setDraft({ ...draft, wechat: event.target.value })} /></LeadField>
          <LeadField label="城市"><Input placeholder="请输入城市" value={draft.city} onChange={(event) => setDraft({ ...draft, city: event.target.value })} /></LeadField>
          <LeadField label="来源"><select className={inputClassName} value={draft.sourceID} onChange={(event) => setDraft({ ...draft, sourceID: event.target.value })}>{sources.map((option) => <option key={textValue(option.id)} value={textValue(option.id)}>{displayText(option.name)}</option>)}</select></LeadField>
          <LeadField label="渠道"><select className={inputClassName} value={draft.channelID} onChange={(event) => setDraft({ ...draft, channelID: event.target.value })}>{channels.map((option) => <option key={textValue(option.id)} value={textValue(option.id)}>{displayText(option.name)}</option>)}</select></LeadField>
          <LeadField label="外部线索ID"><Input placeholder="请输入外部线索ID" value={draft.externalID} onChange={(event) => setDraft({ ...draft, externalID: event.target.value })} /></LeadField>
          <LeadField label="初始诉求" className="crm-work-lead-form-wide"><textarea className="min-h-24 w-full resize-y rounded-md border border-input bg-background px-3 py-2 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/20" placeholder="请输入初始诉求" value={draft.initialNeed} onChange={(event) => setDraft({ ...draft, initialNeed: event.target.value })} /></LeadField>
          <WorkLeadTemplateFields
            templates={templates}
            values={draft.dataValues}
            onChange={(dataValues) => setDraft({ ...draft, dataValues })}
          />
          <div className="crm-work-lead-form-actions"><Button type="button" variant="outline" disabled={submitting} onClick={() => onOpenChange(false)}>取消</Button><Button type="submit" disabled={submitting || !draft.name.trim() || (!draft.phone.trim() && !draft.wechat.trim())}>{submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}确认录入</Button></div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function InvalidateLeadDialog({ lead, reasons, submitting, onClose, onSubmit }: {
  lead: WorkLead | null;
  reasons: WorkLeadOption[];
  submitting: boolean;
  onClose: () => void;
  onSubmit: (reasonID: string, note: string) => void;
}) {
  const [reasonID, setReasonID] = useState("");
  const [note, setNote] = useState("");
  useEffect(() => { if (lead) { setReasonID(textValue(reasons[0]?.id)); setNote(""); } }, [lead, reasons]);
  return (
    <Dialog open={Boolean(lead)} onOpenChange={(open) => !open && !submitting && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader><DialogTitle>判为无效线索</DialogTitle><DialogDescription>{displayText(lead?.name)} · {displayText(lead?.phone)}</DialogDescription></DialogHeader>
        <div className="space-y-4">
          <LeadField label="无效原因" required><select className={inputClassName} value={reasonID} onChange={(event) => setReasonID(event.target.value)}>{reasons.map((option) => <option key={textValue(option.id)} value={textValue(option.id)}>{displayText(option.name)}</option>)}</select></LeadField>
          <LeadField label="补充说明"><textarea className="min-h-20 w-full resize-y rounded-md border border-input bg-background px-3 py-2 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/20" value={note} onChange={(event) => setNote(event.target.value)} /></LeadField>
          <div className="flex justify-end gap-2"><Button type="button" variant="outline" disabled={submitting} onClick={onClose}>取消</Button><Button type="button" variant="destructive" disabled={submitting || !reasonID} onClick={() => onSubmit(reasonID, note)}>{submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Ban className="h-4 w-4" />}确认无效</Button></div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function WorkLeadPoolStyles() {
  return (
    <style>{`
      .crm-work-lead-search-grid {
        grid-template-columns: minmax(260px, 300px) 180px auto auto;
        align-items: center;
        justify-content: start;
      }

      .crm-work-lead-form-wide,
      .crm-work-lead-form-actions {
        grid-column: 1 / -1;
      }

      .crm-work-lead-form-actions {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
      }

      @media (max-width: 639px) {
        .crm-work-lead-search-grid {
          grid-template-columns: minmax(0, 1fr);
          justify-content: stretch;
        }
      }
    `}</style>
  );
}

function LeadField({ label, required = false, className = "", children }: { label: string; required?: boolean; className?: string; children: ReactNode }) {
  return <label className={className}><span className="mb-1.5 block text-sm font-medium">{label}{required ? <span className="ml-1 text-destructive">*</span> : null}</span>{children}</label>;
}

function leadStatusClass(status: string): string {
  if (status === "converted") return "bg-emerald-50 text-emerald-700";
  if (status === "duplicate") return "bg-amber-50 text-amber-700";
  if (status === "invalid") return "bg-muted text-muted-foreground";
  return "bg-blue-50 text-blue-700";
}

function leadCountByStatus(leads: WorkLead[], status: string): number {
  return leads.filter((lead) => textValue(lead.status) === status).length;
}

function leadActionSuccessText(action: string): string {
  if (action === "convert") return "线索已转为客户";
  if (action === "invalid") return "线索已判为无效";
  if (action === "reopen") return "线索已恢复";
  return "线索已更新";
}
