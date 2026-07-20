import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import {
  CalendarCheck2,
  Check,
  Clock3,
  Loader2,
  Trash2,
  UserRound,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
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
  setWorkModalOpen,
  setWorkStoreValue,
  textValue,
  workApi,
  type WorkNodeProps,
} from "./work-core";
import { openWorkCustomerDetailDrawer } from "./work-auth";
import { useWorkFeedbackModalFooterTargets } from "./work-feedback-modal";
import { WorkFormField } from "./work-form-field";
import { useWorkTaskStoreValue } from "./work-task-form-fields";
import type {
  WorkScheduleCustomerOption,
  WorkScheduleDraft,
  WorkScheduleEvent,
  WorkScheduleOptions,
  WorkScheduleType,
} from "./work-schedule-types";

export type WorkScheduleDialogTarget = {
  event?: WorkScheduleEvent | null;
  initialStart?: string;
  initialEnd?: string;
  options?: WorkScheduleOptions;
};

const emptyWorkScheduleDialogTarget: WorkScheduleDialogTarget = {};
const workScheduleDialogTargetPath = "data.actionTarget.workSchedule";
const workScheduleDialogKey = "dialog.workSchedule";
export const workScheduleChangedEvent = "crm-work-schedule-changed";

export function openWorkScheduleDialog(
  store: WorkNodeProps["store"],
  target: WorkScheduleDialogTarget,
) {
  const event = target.event || null;
  const existing = Boolean(textValue(event?.id));
  setWorkStoreValue(store, workScheduleDialogTargetPath, target);
  setWorkStoreValue(
    store,
    "data.actionTarget.workScheduleTitle",
    existing ? "日程详情" : "创建日程",
  );
  setWorkStoreValue(
    store,
    "data.actionTarget.workScheduleDescription",
    event?.schedule_type === "customer_follow" && event.customer_name
      ? `${event.customer_name} · ${event.customer_phone || "未填写电话"}`
      : "安排时间、参与人员和可用资源",
  );
  setWorkModalOpen(store, workScheduleDialogKey, true);
}

export function ShowCrmWorkScheduleForm({ store }: WorkNodeProps = {}) {
  const target = useWorkTaskStoreValue<WorkScheduleDialogTarget>(
    store,
    workScheduleDialogTargetPath,
    emptyWorkScheduleDialogTarget,
  );
  const event = target.event || null;
  const options = target.options || {};
  const [draft, setDraft] = useState<WorkScheduleDraft>(() =>
    scheduleDraft(target),
  );
  const [nextStartAt, setNextStartAt] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const contentRef = useRef<HTMLDivElement | null>(null);
  const existing = Boolean(textValue(event?.id));
  const pending = !event?.status || event.status === "pending";
  const editable =
    !existing ||
    (pending && event?.can_edit !== false && event?.schedule_type !== "meeting");
  const footerTargets = useWorkFeedbackModalFooterTargets(
    contentRef,
    existing,
    existing && !editable,
  );

  useEffect(() => {
    setDraft(scheduleDraft(target));
    setNextStartAt("");
  }, [target]);

  const customerOptions = useMemo(
    () => withCurrentScheduleCustomer(options.customers || [], event),
    [event, options.customers],
  );

  const selectedCustomer = useMemo(
    () =>
      customerOptions.find(
        (customer) => textValue(customer.id) === draft.customerID,
      ),
    [customerOptions, draft.customerID],
  );

  const close = useCallback(() => {
    setWorkModalOpen(store, workScheduleDialogKey, false);
  }, [store]);

  const notifyChanged = useCallback(() => {
    window.dispatchEvent(new CustomEvent(workScheduleChangedEvent));
  }, []);

  const changeType = (scheduleType: WorkScheduleType) => {
    if (existing) return;
    setDraft((current) => ({
      ...current,
      scheduleType,
      customerID: scheduleType === "personal" ? "" : current.customerID,
      title:
        scheduleType === "personal" && current.title.startsWith("跟进 - ")
          ? ""
          : current.title,
    }));
  };

  const changeCustomer = (customerID: string) => {
    const customer = customerOptions.find(
      (row) => textValue(row.id) === customerID,
    );
    setDraft((current) => ({
      ...current,
      customerID,
      title:
        !current.title || current.title.startsWith("跟进 - ")
          ? `跟进 - ${textValue(customer?.name) || "客户"}`
          : current.title,
    }));
  };

  const changeStartAt = (startAt: string) => {
    setDraft((current) => {
      const previousStart = new Date(current.startAt);
      const previousEnd = new Date(current.endAt);
      const nextStart = new Date(startAt);
      const duration =
        !Number.isNaN(previousStart.getTime()) &&
        !Number.isNaN(previousEnd.getTime()) &&
        previousEnd > previousStart
          ? previousEnd.getTime() - previousStart.getTime()
          : 30 * 60_000;
      return {
        ...current,
        startAt,
        endAt: Number.isNaN(nextStart.getTime())
          ? current.endAt
          : formatDateTimeInput(new Date(nextStart.getTime() + duration)),
      };
    });
  };

  const changeDuration = (minutes: number) => {
    setDraft((current) => {
      const startAt = new Date(current.startAt);
      if (Number.isNaN(startAt.getTime())) return current;
      return {
        ...current,
        endAt: formatDateTimeInput(
          new Date(startAt.getTime() + minutes * 60_000),
        ),
      };
    });
  };

  const save = useCallback(async () => {
    if (submitting || !editable) return false;
    if (!draft.title.trim()) {
      toast.error("请输入日程标题");
      return false;
    }
    if (draft.scheduleType === "customer_follow" && !draft.customerID) {
      toast.error("请选择待跟进客户");
      return false;
    }
    const startAt = new Date(draft.startAt);
    const endAt = new Date(draft.endAt);
    if (
      Number.isNaN(startAt.getTime()) ||
      Number.isNaN(endAt.getTime()) ||
      endAt <= startAt
    ) {
      toast.error("结束时间必须晚于开始时间");
      return false;
    }

    setSubmitting(true);
    try {
      await workApi("/crm/work/schedule", {
        method: "POST",
        body: JSON.stringify({
          schedule_event_id: textValue(event?.id),
          schedule_type: draft.scheduleType,
          customer_id: draft.customerID,
          title: draft.title.trim(),
          remark: draft.remark.trim(),
          start_at: startAt.toISOString(),
          end_at: endAt.toISOString(),
          reminder_minutes: Number(draft.reminderMinutes),
          participant_ids: draft.participantIDs,
          resource_ids: draft.resourceIDs,
        }),
      });
      toast.success("日程已保存");
      close();
      notifyChanged();
      return true;
    } catch (error) {
      toast.error(errorMessage(error, "日程保存失败"));
      return false;
    } finally {
      setSubmitting(false);
    }
  }, [close, draft, editable, event?.id, notifyChanged, submitting]);

  const complete = async (withNext = false) => {
    if (!event || submitting) return;
    const nextDate = withNext && nextStartAt ? new Date(nextStartAt) : null;
    if (withNext && (!nextDate || Number.isNaN(nextDate.getTime()))) {
      toast.error("下次跟进时间无效");
      return;
    }
    setSubmitting(true);
    try {
      await workApi("/crm/work/complete_schedule", {
        method: "POST",
        body: JSON.stringify({
          schedule_event_id: textValue(event.id),
          next_start_at: nextDate?.toISOString() || "",
        }),
      });
      toast.success(nextDate ? "已完成并安排下一次跟进" : "日程已完成");
      close();
      notifyChanged();
    } catch (error) {
      toast.error(errorMessage(error, "完成日程失败"));
    } finally {
      setSubmitting(false);
    }
  };

  const cancel = async () => {
    if (!event || submitting) return;
    setSubmitting(true);
    try {
      await workApi("/crm/work/cancel_schedule", {
        method: "POST",
        body: JSON.stringify({ schedule_event_id: textValue(event.id) }),
      });
      toast.success("日程已取消");
      close();
      notifyChanged();
    } catch (error) {
      toast.error(errorMessage(error, "取消日程失败"));
    } finally {
      setSubmitting(false);
    }
  };

  const checkIn = async () => {
    if (!event || submitting || !event.can_check_in) return;
    setSubmitting(true);
    try {
      await workApi("/crm/work/check_in_schedule", {
        method: "POST",
        body: JSON.stringify({ schedule_event_id: textValue(event.id) }),
      });
      toast.success("会议签到成功");
      close();
      notifyChanged();
    } catch (error) {
      toast.error(errorMessage(error, "会议签到失败"));
    } finally {
      setSubmitting(false);
    }
  };

  const openCustomer = async () => {
    const customerID = textValue(event?.customer_id);
    if (!customerID) return;
    try {
      const opened = await openWorkCustomerDetailDrawer(
        store,
        customerID,
        textValue(event?.asset_id),
        textValue(event?.source_workflow_instance_id),
      );
      if (!opened) {
        toast.error("未找到客户详情");
        return;
      }
      close();
    } catch (error) {
      toast.error(errorMessage(error, "客户详情加载失败"));
    }
  };

  useEffect(() => {
    const form = contentRef.current?.closest("form");
    if (!form) return undefined;
    const handleSubmit = (submitEvent: Event) => {
      submitEvent.preventDefault();
      submitEvent.stopPropagation();
      void save();
    };
    form.addEventListener("submit", handleSubmit);
    return () => form.removeEventListener("submit", handleSubmit);
  }, [save]);

  useEffect(() => {
    const form = contentRef.current?.closest("form");
    const submitButton = form?.querySelector<HTMLButtonElement>(
      'button[type="submit"]',
    );
    if (!submitButton) return undefined;
    const previousDisabled = submitButton.disabled;
    submitButton.disabled =
      submitting ||
      !editable ||
      !draft.title.trim() ||
      (draft.scheduleType === "customer_follow" && !draft.customerID);
    return () => {
      submitButton.disabled = previousDisabled;
    };
  }, [draft.customerID, draft.scheduleType, draft.title, editable, submitting]);

  const footerLeft = (
    <>
      {event?.customer_id ? (
        <Button
          type="button"
          variant="ghost"
          onClick={() => void openCustomer()}
        >
          <UserRound className="h-4 w-4" />
          客户详情
        </Button>
      ) : null}
      {editable ? (
        <Button
          type="button"
          variant="ghost"
          className="text-destructive hover:text-destructive"
          disabled={submitting}
          onClick={() => void cancel()}
        >
          <Trash2 className="h-4 w-4" />
          取消日程
        </Button>
      ) : null}
    </>
  );

  const footerActions = editable || event?.can_check_in || event?.checked_in_at ? (
    <>
      {event?.can_check_in ? (
        <Button
          type="button"
          disabled={submitting}
          onClick={() => void checkIn()}
        >
          <Check className="h-4 w-4" />
          签到
        </Button>
      ) : event?.checked_in_at ? (
        <Button type="button" variant="outline" disabled>
          <Check className="h-4 w-4" />
          已签到
        </Button>
      ) : null}
      {editable ? (
        <Button
          type="button"
          variant="outline"
          disabled={submitting}
          onClick={() => void complete(false)}
        >
          完成
        </Button>
      ) : null}
      {editable && event?.schedule_type === "customer_follow" ? (
        <Button
          type="button"
          variant="outline"
          disabled={!nextStartAt || submitting}
          onClick={() => void complete(true)}
        >
          完成并安排下一次
        </Button>
      ) : null}
    </>
  ) : null;

  return (
    <div ref={contentRef} className="grid gap-4 sm:grid-cols-2">
      {footerTargets?.left ? createPortal(footerLeft, footerTargets.left) : null}
      {footerTargets?.actions
        ? createPortal(footerActions, footerTargets.actions)
        : null}

      <fieldset className="contents" disabled={!editable || submitting}>
        <WorkFormField label="日程归属" className="sm:col-span-2">
          <div
            className={`grid max-w-lg gap-2 ${
              event?.schedule_type === "meeting" ? "grid-cols-3" : "grid-cols-2"
            }`}
          >
            <Button
              type="button"
              className="justify-start"
              variant={
                draft.scheduleType === "customer_follow" ? "default" : "outline"
              }
              disabled={existing}
              onClick={() => changeType("customer_follow")}
            >
              <UserRound className="h-4 w-4" />
              客户的
            </Button>
            <Button
              type="button"
              className="justify-start"
              variant={draft.scheduleType === "personal" ? "default" : "outline"}
              disabled={existing}
              onClick={() => changeType("personal")}
            >
              <CalendarCheck2 className="h-4 w-4" />
              自己的
            </Button>
            {event?.schedule_type === "meeting" ? (
              <Button
                type="button"
                className="justify-start"
                variant="default"
                disabled
              >
                <Clock3 className="h-4 w-4" />
                案件会议
              </Button>
            ) : null}
          </div>
        </WorkFormField>

        {draft.scheduleType === "customer_follow" ? (
          <WorkFormField
            label="待跟进客户"
            required
            className="sm:col-span-2"
            hint={
              selectedCustomer?.next_follow_at
                ? `当前跟进时间：${formatScheduleDisplayDate(selectedCustomer.next_follow_at)}`
                : undefined
            }
          >
            <Select
              value={draft.customerID}
              disabled={existing}
              onValueChange={changeCustomer}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="请选择当前待跟进客户" />
              </SelectTrigger>
              <SelectContent position="popper">
                {customerOptions.map((customer) => (
                  <SelectItem
                    key={textValue(customer.id)}
                    value={textValue(customer.id)}
                  >
                    {scheduleCustomerLabel(customer)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </WorkFormField>
        ) : null}

        <WorkFormField label="标题" required className="sm:col-span-2">
          <Input
            value={draft.title}
            placeholder={
              draft.scheduleType === "customer_follow" ? "客户跟进" : "日程标题"
            }
            onChange={(input) =>
              setDraft((current) => ({ ...current, title: input.target.value }))
            }
          />
        </WorkFormField>

        <WorkFormField label="开始时间" required>
          <Input
            type="datetime-local"
            value={draft.startAt}
            onChange={(input) => changeStartAt(input.target.value)}
          />
        </WorkFormField>

        <WorkFormField label="使用时长" className="sm:col-span-2">
          <div className="flex flex-wrap gap-2">
            {[30, 60, 90, 120].map((minutes) => (
              <Button
                key={minutes}
                type="button"
                variant={scheduleDraftDurationMinutes(draft) === minutes ? "default" : "outline"}
                onClick={() => changeDuration(minutes)}
              >
                {minutes < 60 ? `${minutes}分钟` : `${minutes / 60}小时`}
              </Button>
            ))}
          </div>
        </WorkFormField>
        <WorkFormField label="结束时间" required>
          <Input
            type="datetime-local"
            value={draft.endAt}
            onChange={(input) =>
              setDraft((current) => ({ ...current, endAt: input.target.value }))
            }
          />
        </WorkFormField>

        <WorkFormField label="提醒" className="sm:col-span-2">
          <Select
            value={draft.reminderMinutes}
            onValueChange={(reminderMinutes) =>
              setDraft((current) => ({ ...current, reminderMinutes }))
            }
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="请选择提醒时间" />
            </SelectTrigger>
            <SelectContent position="popper">
              {(options.reminders || []).map((reminder) => (
                <SelectItem
                  key={textValue(reminder.id)}
                  value={textValue(reminder.id)}
                >
                  {textValue(reminder.value)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </WorkFormField>

        <ScheduleMultiOptions
          label="参与人"
          values={draft.participantIDs}
          options={(options.staff || []).map((staff) => ({
            id: textValue(staff.id),
            label: textValue(staff.name),
          }))}
          onChange={(participantIDs) =>
            setDraft((current) => ({ ...current, participantIDs }))
          }
        />

        <ScheduleMultiOptions
          label="公共资源"
          values={draft.resourceIDs}
          options={(options.resources || []).map((resource) => ({
            id: textValue(resource.id),
            label: [textValue(resource.name), textValue(resource.location)]
              .filter(Boolean)
              .join(" · "),
          }))}
          emptyText="暂无可用资源"
          onChange={(resourceIDs) =>
            setDraft((current) => ({ ...current, resourceIDs }))
          }
        />

        <WorkFormField label="备注" className="sm:col-span-2">
          <Textarea
            value={draft.remark}
            placeholder="记录准备事项或跟进说明"
            className="min-h-24 resize-y"
            onChange={(input) =>
              setDraft((current) => ({ ...current, remark: input.target.value }))
            }
          />
        </WorkFormField>

        {existing && pending && event?.schedule_type === "customer_follow" ? (
          <WorkFormField
            label="完成后下次跟进"
            className="border-t pt-4 sm:col-span-2"
          >
            <Input
              type="datetime-local"
              value={nextStartAt}
              onChange={(input) => setNextStartAt(input.target.value)}
            />
          </WorkFormField>
        ) : null}
      </fieldset>

      {submitting ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground sm:col-span-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          正在保存日程
        </div>
      ) : null}
    </div>
  );
}

function ScheduleMultiOptions({
  label,
  values,
  options,
  emptyText = "暂无可选人员",
  onChange,
}: {
  label: string;
  values: string[];
  options: Array<{ id: string; label: string }>;
  emptyText?: string;
  onChange: (values: string[]) => void;
}) {
  const selected = new Set(values);
  return (
    <WorkFormField label={label} className="col-span-full">
      {options.length ? (
        <div className="grid gap-2 sm:grid-cols-2">
          {options.map((option) => {
            const checked = selected.has(option.id);
            return (
              <Button
                key={option.id}
                type="button"
                variant="outline"
                aria-pressed={checked}
                className={`h-auto min-h-10 justify-start gap-2 px-3 py-2 font-normal ${
                  checked
                    ? "border-primary bg-primary/10 text-primary shadow-sm hover:bg-primary/15 hover:text-primary"
                    : "text-muted-foreground hover:text-foreground"
                }`}
                onClick={() => onChange(toggleScheduleID(values, option.id))}
              >
                <span
                  className={`inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-sm border ${
                    checked
                      ? "border-primary bg-primary text-primary-foreground"
                      : "border-input bg-background"
                  }`}
                >
                  {checked ? <Check className="h-3 w-3" /> : null}
                </span>
                <span className="min-w-0 whitespace-normal break-words text-left leading-5">
                  {option.label}
                </span>
              </Button>
            );
          })}
        </div>
      ) : (
        <p className="text-xs text-muted-foreground">{emptyText}</p>
      )}
    </WorkFormField>
  );
}

function scheduleDraft(target: WorkScheduleDialogTarget): WorkScheduleDraft {
  const event = target.event || null;
  const initialStart = parseScheduleDate(target.initialStart);
  const initialEnd = parseScheduleDate(target.initialEnd);
  const start = event?.start_at
    ? new Date(event.start_at)
    : initialStart || nextHalfHour();
  const end = event?.end_at
    ? new Date(event.end_at)
    : initialEnd || new Date(start.getTime() + 30 * 60_000);
  return {
    scheduleType: event?.schedule_type || "customer_follow",
    customerID: textValue(event?.customer_id),
    title: textValue(event?.title),
    remark: textValue(event?.remark),
    startAt: formatDateTimeInput(start),
    endAt: formatDateTimeInput(end),
    reminderMinutes: textValue(event?.reminder_minutes) || "30",
    participantIDs: (event?.participant_ids || []).map(textValue).filter(Boolean),
    resourceIDs: (event?.resource_ids || []).map(textValue).filter(Boolean),
  };
}

function scheduleDraftDurationMinutes(draft: WorkScheduleDraft): number {
  const startAt = new Date(draft.startAt);
  const endAt = new Date(draft.endAt);
  if (
    Number.isNaN(startAt.getTime()) ||
    Number.isNaN(endAt.getTime()) ||
    endAt <= startAt
  ) {
    return 0;
  }
  return Math.round((endAt.getTime() - startAt.getTime()) / 60_000);
}

function parseScheduleDate(value: unknown): Date | null {
  const raw = textValue(value);
  if (!raw) return null;
  const date = new Date(raw);
  return Number.isNaN(date.getTime()) ? null : date;
}

function nextHalfHour(): Date {
  const date = new Date();
  date.setSeconds(0, 0);
  const minutes = date.getMinutes();
  date.setMinutes(minutes < 30 ? 30 : 60);
  return date;
}

export function formatDateTimeInput(value: Date): string {
  if (Number.isNaN(value.getTime())) return "";
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  const hour = String(value.getHours()).padStart(2, "0");
  const minute = String(value.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hour}:${minute}`;
}

function formatScheduleDisplayDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}

function scheduleCustomerLabel(customer: {
  name?: string;
  phone?: string;
  owner_staff_name?: string;
}): string {
  return [customer.name, customer.phone, customer.owner_staff_name]
    .map(textValue)
    .filter(Boolean)
    .join(" · ");
}

function withCurrentScheduleCustomer(
  customers: WorkScheduleCustomerOption[],
  event: WorkScheduleEvent | null,
): WorkScheduleCustomerOption[] {
  const customerID = textValue(event?.customer_id);
  if (
    !customerID ||
    customers.some((customer) => textValue(customer.id) === customerID)
  ) {
    return customers;
  }
  return [
    {
      id: customerID,
      name: textValue(event?.customer_name) || `客户 ${customerID}`,
      phone: textValue(event?.customer_phone),
    },
    ...customers,
  ];
}

function toggleScheduleID(values: string[], target: string): string[] {
  return values.includes(target)
    ? values.filter((value) => value !== target)
    : [...values, target];
}
