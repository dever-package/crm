import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Bell, CalendarPlus2, Loader2, RefreshCw } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";

import {
  errorMessage,
  textValue,
  workApi,
  workRefreshEvent,
  type WorkNodeProps,
} from "./work-core";
import { WorkScheduleCalendar } from "./work-schedule-calendar";
import {
  openWorkScheduleDialog,
  ShowCrmWorkScheduleForm,
  workScheduleChangedEvent,
} from "./work-schedule-dialog";
import { WorkScheduleStyles } from "./work-schedule-styles";
import type {
  WorkScheduleEvent,
  WorkScheduleListResponse,
  WorkScheduleOptions,
  WorkScheduleRange,
  WorkScheduleReminderResponse,
} from "./work-schedule-types";

export { ShowCrmWorkScheduleForm };

export function ShowCrmWorkSchedule({ store }: WorkNodeProps = {}) {
  const reminderRef = useRef<HTMLDivElement>(null);
  const [range, setRange] = useState<WorkScheduleRange | null>(null);
  const [events, setEvents] = useState<WorkScheduleEvent[]>([]);
  const [options, setOptions] = useState<WorkScheduleOptions>({});
  const [reminders, setReminders] = useState<WorkScheduleEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [reminderOpen, setReminderOpen] = useState(false);

  const loadOptions = useCallback(async () => {
    try {
      setOptions(await workApi<WorkScheduleOptions>("/crm/work/schedule_options"));
    } catch (error) {
      toast.error(errorMessage(error, "日程选项加载失败"));
    }
  }, []);

  const loadEvents = useCallback(async (targetRange: WorkScheduleRange) => {
    setLoading(true);
    try {
      const query = new URLSearchParams({
        start_at: targetRange.start.toISOString(),
        end_at: targetRange.end.toISOString(),
      });
      const payload = await workApi<WorkScheduleListResponse>(
        `/crm/work/schedules?${query.toString()}`,
      );
      setEvents(Array.isArray(payload.list) ? payload.list : []);
    } catch (error) {
      toast.error(errorMessage(error, "日程加载失败"));
    } finally {
      setLoading(false);
    }
  }, []);

  const loadReminders = useCallback(async () => {
    try {
      const payload = await workApi<WorkScheduleReminderResponse>(
        "/crm/work/schedule_reminders",
      );
      setReminders(Array.isArray(payload.list) ? payload.list : []);
    } catch {
      setReminders([]);
    }
  }, []);

  const refresh = useCallback(async () => {
    await Promise.all([
      range ? loadEvents(range) : Promise.resolve(),
      loadOptions(),
      loadReminders(),
    ]);
    window.dispatchEvent(new CustomEvent(workRefreshEvent));
  }, [loadEvents, loadOptions, loadReminders, range]);

  useEffect(() => {
    void loadOptions();
    void loadReminders();
  }, [loadOptions, loadReminders]);

  useEffect(() => {
    if (range) void loadEvents(range);
  }, [loadEvents, range]);

  useEffect(() => {
    const timer = window.setInterval(() => void loadReminders(), 60_000);
    return () => window.clearInterval(timer);
  }, [loadReminders]);

  useEffect(() => {
    const reload = () => void refresh();
    window.addEventListener(workScheduleChangedEvent, reload);
    return () => window.removeEventListener(workScheduleChangedEvent, reload);
  }, [refresh]);

  useEffect(() => {
    const close = (event: MouseEvent) => {
      if (!reminderRef.current?.contains(event.target as Node)) {
        setReminderOpen(false);
      }
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  const openCreate = useCallback(
    (start?: Date, end?: Date) => {
      openWorkScheduleDialog(store, {
        event: null,
        initialStart: start?.toISOString(),
        initialEnd: end?.toISOString(),
        options,
      });
    },
    [options, store],
  );

  const openEvent = useCallback(
    (event: WorkScheduleEvent) => {
      openWorkScheduleDialog(store, { event, options });
    },
    [options, store],
  );

  const move = useCallback(
    async (event: WorkScheduleEvent, start: Date, end: Date) => {
      try {
        await workApi("/crm/work/reschedule", {
          method: "POST",
          body: JSON.stringify({
            schedule_event_id: textValue(event.id),
            start_at: start.toISOString(),
            end_at: end.toISOString(),
          }),
        });
        toast.success("日程时间已调整");
        await refresh();
        return true;
      } catch (error) {
        toast.error(errorMessage(error, "日程改期失败"));
        return false;
      }
    },
    [refresh],
  );

  const openReminder = async (event: WorkScheduleEvent) => {
    setReminderOpen(false);
    if (event.action_type === "check_in") {
      try {
        await workApi("/crm/work/check_in_schedule", {
          method: "POST",
          body: JSON.stringify({ schedule_event_id: textValue(event.id) }),
        });
        toast.success("会议签到成功");
        await refresh();
      } catch (error) {
        toast.error(errorMessage(error, "会议签到失败"));
      }
      return;
    }
    openEvent(event);
    try {
      await workApi("/crm/work/read_schedule_reminder", {
        method: "POST",
        body: JSON.stringify({ schedule_event_id: textValue(event.id) }),
      });
      await loadReminders();
      window.dispatchEvent(new CustomEvent(workRefreshEvent));
    } catch {
      // Opening the event remains useful even if the read marker fails.
    }
  };

  const pendingCount = useMemo(
    () => events.filter((event) => event.status === "pending").length,
    [events],
  );

  return (
    <div className="crm-schedule-workspace">
      <WorkScheduleStyles />
      <header className="crm-schedule-page-header">
        <div>
          <strong>我的日程</strong>
          <span>{pendingCount} 项待进行</span>
        </div>
        <div className="crm-schedule-page-actions">
          <Button
            type="button"
            variant="outline"
            size="icon"
            aria-label="刷新日程"
            title="刷新日程"
            disabled={loading}
            onClick={() => void refresh()}
          >
            {loading ? <Loader2 className="animate-spin" size={16} /> : <RefreshCw size={16} />}
          </Button>
          <div ref={reminderRef} className="crm-schedule-reminder-control">
            <Button
              type="button"
              variant="outline"
              aria-label={`到期提醒 ${reminders.length} 项`}
              aria-expanded={reminderOpen}
              onClick={() => setReminderOpen((value) => !value)}
            >
              <Bell size={16} />
              提醒
              {reminders.length ? <span>{reminders.length}</span> : null}
            </Button>
            {reminderOpen ? (
              <div className="crm-schedule-reminder-panel">
                <div className="crm-schedule-reminder-title">
                  <strong>待办提醒</strong>
                  <span>{reminders.length}项</span>
                </div>
                {reminders.length ? (
                  <div className="crm-schedule-reminder-list">
                    {reminders.map((event) => (
                      <button
                        key={textValue(event.id)}
                        type="button"
                        onClick={() => void openReminder(event)}
                      >
                        <strong>{textValue(event.title) || "未命名日程"}</strong>
                        <span>{scheduleTimeLabel(event.start_at)}</span>
                        <small>
                          {event.action_type === "check_in"
                            ? "点击签到"
                            : event.customer_name || "查看日程"}
                        </small>
                      </button>
                    ))}
                  </div>
                ) : (
                  <p>当前没有待办提醒</p>
                )}
              </div>
            ) : null}
          </div>
          <Button type="button" onClick={() => openCreate()}>
            <CalendarPlus2 size={16} />创建日程
          </Button>
        </div>
      </header>

      <WorkScheduleCalendar
        events={events}
        loading={loading}
        onRangeChange={setRange}
        onCreate={openCreate}
        onOpen={openEvent}
        onMove={move}
      />

    </div>
  );
}

function scheduleTimeLabel(value: unknown): string {
  const date = new Date(textValue(value));
  if (Number.isNaN(date.getTime())) return textValue(value);
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}
