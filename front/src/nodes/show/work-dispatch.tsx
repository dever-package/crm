import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ArrowDown,
  ArrowRight,
  ArrowUp,
  CalendarClock,
  Check,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Search,
  Trash2,
  UserRoundPlus,
  UsersRound,
} from "lucide-react";
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

import {
  errorMessage,
  textValue,
  workApi,
  workRefreshEvent,
  type WorkNodeProps,
} from "./work-core";
import { WorkDispatchPendingLeads } from "./work-dispatch-pending-leads";
import { WorkDispatchScheduleDialog } from "./work-dispatch-schedule";
import { WorkDispatchStyles } from "./work-dispatch-styles";
import {
  cloneDispatchSchedule,
  fullWeekDispatchSchedule,
  type DispatchConfigPayload,
  type DispatchMemberDraft,
  type DispatchPool,
  type DispatchSchedule,
} from "./work-dispatch-types";

type DispatchTab = "direct" | "group";

export function ShowCrmWorkDispatch(_props: WorkNodeProps = {}) {
  const [config, setConfig] = useState<DispatchConfigPayload | null>(null);
  const [workflowID, setWorkflowID] = useState("");
  const [selectedPoolID, setSelectedPoolID] = useState("");
  const [activePoolID, setActivePoolID] = useState("");
  const [autoHandoffEnabled, setAutoHandoffEnabled] = useState(false);
  const [tab, setTab] = useState<DispatchTab>("direct");
  const [members, setMembers] = useState<DispatchMemberDraft[]>([]);
  const [memberSearch, setMemberSearch] = useState("");
  const [memberSearchOpen, setMemberSearchOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [scheduleStaffID, setScheduleStaffID] = useState("");
  const [groupDialog, setGroupDialog] = useState<"create" | "rename" | null>(null);
  const [groupName, setGroupName] = useState("");
  const addRef = useRef<HTMLDivElement>(null);

  const applyPayload = useCallback((payload: DispatchConfigPayload, preferredPoolID = "") => {
    setConfig(payload);
    setWorkflowID(textValue(payload.workflow_id));
    setAutoHandoffEnabled(Boolean(payload.auto_handoff_enabled));
    const pools = Array.isArray(payload.pools) ? payload.pools : [];
    const serverActivePoolID = textValue(payload.active_pool_id);
    setActivePoolID(serverActivePoolID);
    const preferred = pools.find((pool) => textValue(pool.id) === preferredPoolID);
    const selected = preferred ||
      pools.find((pool) => textValue(pool.id) === serverActivePoolID) ||
      pools.find((pool) => pool.pool_type === "direct") ||
      pools[0];
    const poolID = textValue(selected?.id);
    setSelectedPoolID(poolID);
    setTab(selected?.pool_type === "group" ? "group" : "direct");
    setMembers(memberDrafts(selected));
    setDirty(false);
  }, []);

  const loadConfig = useCallback(async (targetWorkflowID = workflowID, preferredPoolID = selectedPoolID) => {
    setLoading(true);
    try {
      const query = targetWorkflowID
        ? `?workflow_id=${encodeURIComponent(targetWorkflowID)}`
        : "";
      const payload = await workApi<DispatchConfigPayload>(`/crm/work/lead_dispatch_config${query}`);
      applyPayload(payload, preferredPoolID);
    } catch (error) {
      toast.error(errorMessage(error, "派单配置加载失败"));
    } finally {
      setLoading(false);
    }
  }, [applyPayload, workflowID, selectedPoolID]);

  useEffect(() => {
    void loadConfig("", "");
  }, []);

  useEffect(() => {
    const close = (event: MouseEvent) => {
      if (!addRef.current?.contains(event.target as Node)) setMemberSearchOpen(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  const pools = Array.isArray(config?.pools) ? config.pools : [];
  const staff = Array.isArray(config?.staff) ? config.staff : [];
  const selectedPool = pools.find((pool) => textValue(pool.id) === selectedPoolID);
  const directPool = pools.find((pool) => pool.pool_type === "direct");
  const groupPools = pools.filter((pool) => pool.pool_type === "group");
  const selectedStaffIDs = useMemo(() => new Set(members.map((member) => member.staffId)), [members]);
  const availableStaff = useMemo(() => {
    const keyword = memberSearch.trim().toLocaleLowerCase();
    return staff.filter((row) => {
      const id = textValue(row.id);
      if (!id || selectedStaffIDs.has(id)) return false;
      if (!keyword) return true;
      return `${textValue(row.name)} ${textValue(row.phone)}`.toLocaleLowerCase().includes(keyword);
    });
  }, [memberSearch, selectedStaffIDs, staff]);

  const confirmDiscardDraft = () =>
    !dirty || window.confirm("当前修改尚未保存，继续操作将丢失这些修改。确认继续？");

  const resetSharedDraft = () => {
    setActivePoolID(textValue(config?.active_pool_id));
    setAutoHandoffEnabled(Boolean(config?.auto_handoff_enabled));
  };

  const choosePool = (pool: DispatchPool) => {
    const poolID = textValue(pool.id);
    if (!poolID || poolID === selectedPoolID) return true;
    if (!confirmDiscardDraft()) return false;
    resetSharedDraft();
    setSelectedPoolID(poolID);
    setMembers(memberDrafts(pool));
    setDirty(false);
    return true;
  };

  const chooseTab = (nextTab: DispatchTab) => {
    if (nextTab === tab) return;
    const target = nextTab === "direct" ? directPool :
      groupPools.find((pool) => textValue(pool.id) === activePoolID) || groupPools[0];
    if (target && !choosePool(target)) return;
    if (!target) {
      if (!confirmDiscardDraft()) return;
      resetSharedDraft();
      setSelectedPoolID("");
      setMembers([]);
      setDirty(false);
    }
    setTab(nextTab);
  };

  const updateMember = (staffID: string, update: Partial<DispatchMemberDraft>) => {
    setMembers((current) => current.map((member) =>
      member.staffId === staffID ? { ...member, ...update } : member,
    ));
    setDirty(true);
  };

  const moveMember = (index: number, offset: -1 | 1) => {
    setMembers((current) => {
      const target = index + offset;
      if (target < 0 || target >= current.length) return current;
      const next = [...current];
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
    setDirty(true);
  };

  const save = async () => {
    if (!workflowID || !selectedPoolID || saving) return;
    setSaving(true);
    try {
      const payload = await workApi<DispatchConfigPayload>("/crm/work/save_lead_dispatch_config", {
        method: "POST",
        body: JSON.stringify({
          workflow_id: workflowID,
          auto_handoff_enabled: autoHandoffEnabled,
          pool_id: selectedPoolID,
          active_pool_id: activePoolID || selectedPoolID,
          members: members.map((member) => ({
            staff_id: member.staffId,
            enabled: member.enabled,
            daily_limit: member.dailyLimit,
            weekly_schedule: member.schedule,
          })),
        }),
      });
      applyPayload(payload, selectedPoolID);
      toast.success("派单配置已保存");
      if (payload.retry_warning) toast.warning(payload.retry_warning);
      window.dispatchEvent(new CustomEvent(workRefreshEvent));
    } catch (error) {
      toast.error(errorMessage(error, "派单配置保存失败"));
    } finally {
      setSaving(false);
    }
  };

  const submitGroup = async () => {
    if (!groupDialog || !groupName.trim()) return;
    if (!confirmDiscardDraft()) return;
    const path = groupDialog === "create"
      ? "/crm/work/create_lead_dispatch_group"
      : "/crm/work/rename_lead_dispatch_group";
    try {
      const payload = await workApi<DispatchConfigPayload>(path, {
        method: "POST",
        body: JSON.stringify({
          workflow_id: workflowID,
          pool_id: selectedPoolID,
          name: groupName.trim(),
        }),
      });
      const created = groupDialog === "create"
        ? (payload.pools || []).find((pool) => pool.name === groupName.trim() && pool.pool_type === "group")
        : undefined;
      applyPayload(payload, textValue(created?.id) || selectedPoolID);
      setGroupDialog(null);
      toast.success(groupDialog === "create" ? "工作组已创建" : "工作组已重命名");
    } catch (error) {
      toast.error(errorMessage(error, "工作组保存失败"));
    }
  };

  const deleteGroup = async () => {
    if (selectedPool?.pool_type !== "group") return;
    const discardNotice = dirty ? "当前未保存的修改也会丢失。" : "";
    if (!window.confirm(`${discardNotice}确认删除工作组“${textValue(selectedPool.name)}”？`)) return;
    try {
      const payload = await workApi<DispatchConfigPayload>("/crm/work/delete_lead_dispatch_group", {
        method: "POST",
        body: JSON.stringify({ workflow_id: workflowID, pool_id: selectedPoolID }),
      });
      applyPayload(payload, textValue(payload.active_pool_id));
      toast.success("工作组已删除");
    } catch (error) {
      toast.error(errorMessage(error, "工作组删除失败"));
    }
  };

  const scheduleMember = members.find((member) => member.staffId === scheduleStaffID);
  const scheduleStaff = staff.find((row) => textValue(row.id) === scheduleStaffID);

  return (
    <div className="crm-dispatch-page">
      <WorkDispatchStyles />
      <header className="crm-dispatch-header">
        <div>
          <h2>线索派单管理</h2>
          <p>自动派给下一阶段接单人员，人工派单不计每日上限</p>
        </div>
        <div className="crm-dispatch-header-actions">
          {(config?.routes?.length || 0) > 1 ? (
            <select
              className="crm-dispatch-department"
              value={workflowID}
              aria-label="选择线索流程"
              onChange={(event) => {
                if (!confirmDiscardDraft()) return;
                void loadConfig(event.currentTarget.value, "");
              }}
            >
              {(config?.routes || []).map((route) => (
                <option key={textValue(route.workflow_id)} value={textValue(route.workflow_id)}>
                  {textValue(route.workflow_name)} · {textValue(route.target_department_name)}
                </option>
              ))}
            </select>
          ) : <span>{textValue(config?.workflow_name)}</span>}
          <Button type="button" variant="outline" size="icon" title="刷新" aria-label="刷新" disabled={loading} onClick={() => { if (confirmDiscardDraft()) void loadConfig(); }}>
            {loading ? <Loader2 className="animate-spin" size={16} /> : <RefreshCw size={16} />}
          </Button>
          <Button type="button" disabled={saving || !selectedPoolID} onClick={() => void save()}>
            {saving ? <Loader2 className="animate-spin" size={16} /> : <Save size={16} />}
            保存
          </Button>
        </div>
      </header>

      <section className="crm-dispatch-route">
        <div className="crm-dispatch-route-point">
          <small>来源</small>
          <strong>{textValue(config?.source_department_name) || "-"}</strong>
          <span>{textValue(config?.source_stage_name) || "-"}</span>
        </div>
        <ArrowRight size={20} aria-hidden="true" />
        <div className="crm-dispatch-route-point">
          <small>接收</small>
          <strong>{textValue(config?.target_department_name) || "-"}</strong>
          <span>{textValue(config?.target_stage_name) || "-"}</span>
        </div>
        <div className="crm-dispatch-route-switch">
          <span>
            <strong>自动流转</strong>
            <small>新线索跳过确认任务</small>
          </span>
          <button
            type="button"
            className={`crm-dispatch-toggle ${autoHandoffEnabled ? "is-on" : ""}`}
            role="switch"
            aria-checked={autoHandoffEnabled}
            aria-label="自动流转"
            title={autoHandoffEnabled ? "已开启自动流转" : "已关闭自动流转"}
            onClick={() => {
              setAutoHandoffEnabled((current) => !current);
              setDirty(true);
            }}
          />
        </div>
      </section>

      <div className="crm-dispatch-tabs" role="tablist" aria-label="派单方式">
        <Button type="button" variant="ghost" className={tab === "direct" ? "is-active" : ""} onClick={() => chooseTab("direct")}>按员工分配</Button>
        <Button type="button" variant="ghost" className={tab === "group" ? "is-active" : ""} onClick={() => chooseTab("group")}>按工作组</Button>
      </div>

      {loading && !config ? (
        <div className="crm-dispatch-empty"><Loader2 className="animate-spin" size={22} /></div>
      ) : (
        <div className="crm-dispatch-workspace">
          <main className="crm-dispatch-main">
            {selectedPool ? (
              <>
                <div className="crm-dispatch-section-head">
                  <div>
                    <strong>{textValue(selectedPool.name)}</strong>
                    <span> · {members.length} 人</span>
                  </div>
                  {activePoolID === selectedPoolID ? (
                    <span className="crm-dispatch-current">当前生效</span>
                  ) : (
                    <Button type="button" size="sm" variant="outline" onClick={() => { setActivePoolID(selectedPoolID); setDirty(true); }}>
                      <Check size={14} />设为当前
                    </Button>
                  )}
                </div>

                <div ref={addRef} className="crm-dispatch-add">
                  <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={memberSearch}
                    className="pl-9"
                    placeholder={`搜索并添加${textValue(config?.target_department_name) || "接收部门"}接单人员`}
                    onFocus={() => setMemberSearchOpen(true)}
                    onChange={(event) => { setMemberSearch(event.currentTarget.value); setMemberSearchOpen(true); }}
                  />
                  {memberSearchOpen ? (
                    <div className="crm-dispatch-add-results">
                      {availableStaff.length ? availableStaff.map((row) => (
                        <button
                          key={textValue(row.id)}
                          type="button"
                          onClick={() => {
                            setMembers((current) => [...current, {
                              staffId: textValue(row.id),
                              enabled: true,
                              dailyLimit: 0,
                              schedule: fullWeekDispatchSchedule(),
                            }]);
                            setMemberSearch("");
                            setMemberSearchOpen(false);
                            setDirty(true);
                          }}
                        >
                          <span>{textValue(row.name)}</span>
                          <small>{textValue(row.phone) || "无手机号"}</small>
                        </button>
                      )) : <div className="p-3 text-center text-muted-foreground">没有可添加员工</div>}
                    </div>
                  ) : null}
                </div>

                {members.length ? (
                  <div className="crm-dispatch-members">
                    {members.map((member, index) => {
                      const person = staff.find((row) => textValue(row.id) === member.staffId);
                      return (
                        <div key={member.staffId} className="crm-dispatch-member">
                          <div className="crm-dispatch-order">
                            <Button type="button" variant="ghost" size="icon" title="上移" aria-label="上移" disabled={index === 0} onClick={() => moveMember(index, -1)}><ArrowUp size={14} /></Button>
                            <Button type="button" variant="ghost" size="icon" title="下移" aria-label="下移" disabled={index === members.length - 1} onClick={() => moveMember(index, 1)}><ArrowDown size={14} /></Button>
                          </div>
                          <div className="crm-dispatch-member-person">
                            <strong>{textValue(person?.name) || "未知员工"}</strong>
                            <span>今日自动 {Number(person?.today_auto_count) || 0} 单</span>
                          </div>
                          <button type="button" className={`crm-dispatch-toggle ${member.enabled ? "is-on" : ""}`} role="switch" aria-checked={member.enabled} title={member.enabled ? "已参与轮转" : "已暂停轮转"} onClick={() => updateMember(member.staffId, { enabled: !member.enabled })} />
                          <div className="crm-dispatch-limit">
                            <Input type="number" min={0} max={100000} value={member.dailyLimit} aria-label="每日自动派单上限" onChange={(event) => updateMember(member.staffId, { dailyLimit: Math.max(0, Number(event.currentTarget.value) || 0) })} />
                            <small>{member.dailyLimit ? "单/日" : "不限"}</small>
                          </div>
                          <div className="crm-dispatch-member-actions">
                            <Button type="button" variant="ghost" size="icon" title="工作时间" aria-label="工作时间" onClick={() => setScheduleStaffID(member.staffId)}><CalendarClock size={16} /></Button>
                            <Button type="button" variant="ghost" size="icon" title="移除" aria-label="移除" onClick={() => { setMembers((current) => current.filter((row) => row.staffId !== member.staffId)); setDirty(true); }}><Trash2 size={16} /></Button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <div className="crm-dispatch-empty"><div><UserRoundPlus size={24} /><p>请添加参与自动派单的员工</p></div></div>
                )}
              </>
            ) : (
              <div className="crm-dispatch-empty"><div><UsersRound size={24} /><p>{tab === "group" ? "先创建一个工作组" : "默认派单池未初始化"}</p></div></div>
            )}
          </main>

          <aside className="crm-dispatch-side">
            <div className="crm-dispatch-section-head">
              <strong>{tab === "group" ? "工作组" : "派单池"}</strong>
              {tab === "group" ? (
                <Button type="button" variant="ghost" size="icon" title="新增工作组" aria-label="新增工作组" onClick={() => { setGroupName(""); setGroupDialog("create"); }}><Plus size={16} /></Button>
              ) : null}
            </div>
            <div className="crm-dispatch-groups">
              {(tab === "direct" ? (directPool ? [directPool] : []) : groupPools).map((pool) => (
                <button key={textValue(pool.id)} type="button" className={`crm-dispatch-group ${textValue(pool.id) === selectedPoolID ? "is-selected" : ""}`} onClick={() => choosePool(pool)}>
                  <span>{textValue(pool.name)}</span>
                  <small>{pool.is_active ? "当前" : `${pool.member_list?.length || 0}人`}</small>
                </button>
              ))}
            </div>
            {tab === "group" && selectedPool?.pool_type === "group" ? (
              <div className="mt-4 flex gap-2 border-t pt-4">
                <Button type="button" variant="outline" size="sm" onClick={() => { setGroupName(textValue(selectedPool.name)); setGroupDialog("rename"); }}><Pencil size={14} />重命名</Button>
                <Button type="button" variant="outline" size="sm" onClick={() => void deleteGroup()}><Trash2 size={14} />删除</Button>
              </div>
            ) : null}
          </aside>
        </div>
      )}

      <WorkDispatchPendingLeads
        key={workflowID || "lead-dispatch-pending"}
        rows={config?.pending || []}
        assignees={config?.assignee_options || []}
        sourceStageName={textValue(config?.source_stage_name)}
        targetStageName={textValue(config?.target_stage_name)}
        onBeforeAssign={confirmDiscardDraft}
        onAssigned={() => loadConfig(workflowID, selectedPoolID)}
      />

      <WorkDispatchScheduleDialog
        open={Boolean(scheduleMember)}
        staffName={textValue(scheduleStaff?.name)}
        value={scheduleMember?.schedule || fullWeekDispatchSchedule()}
        onOpenChange={(open) => !open && setScheduleStaffID("")}
        onSave={(schedule: DispatchSchedule) => scheduleMember && updateMember(scheduleMember.staffId, { schedule })}
      />

      <Dialog open={Boolean(groupDialog)} onOpenChange={(open) => !open && setGroupDialog(null)}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>{groupDialog === "create" ? "新增工作组" : "重命名工作组"}</DialogTitle>
            <DialogDescription>工作组用于当前接收部门的线索派单。</DialogDescription>
          </DialogHeader>
          <Input value={groupName} maxLength={64} autoFocus placeholder="工作组名称" onChange={(event) => setGroupName(event.currentTarget.value)} onKeyDown={(event) => { if (event.key === "Enter") void submitGroup(); }} />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setGroupDialog(null)}>取消</Button>
            <Button type="button" disabled={!groupName.trim()} onClick={() => void submitGroup()}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function memberDrafts(pool?: DispatchPool): DispatchMemberDraft[] {
  return (pool?.member_list || []).map((member) => ({
    staffId: textValue(member.staff_id),
    enabled: Number(member.status) !== 2,
    dailyLimit: Math.max(0, Number(member.daily_limit) || 0),
    schedule: cloneDispatchSchedule(member.weekly_schedule),
  })).filter((member) => Boolean(member.staffId));
}
