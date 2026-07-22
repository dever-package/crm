import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import FullCalendar from "@fullcalendar/react";
import dayGridPlugin from "@fullcalendar/daygrid";
import interactionPlugin from "@fullcalendar/interaction";
import timeGridPlugin from "@fullcalendar/timegrid";
import zhCnLocale from "@fullcalendar/core/locales/zh-cn";
import type {
  DateSelectArg,
  DatesSetArg,
  EventApi,
  EventClickArg,
  EventContentArg,
  EventInput,
} from "@fullcalendar/core";
import {
  CalendarDays,
  ChevronLeft,
  ChevronRight,
  Clock3,
} from "lucide-react";

import { Button } from "@/components/ui/button";

import { textValue } from "./work-core";
import type {
  WorkScheduleEvent,
  WorkScheduleRange,
  WorkScheduleView,
} from "./work-schedule-types";

type WorkScheduleCalendarProps = {
  events: WorkScheduleEvent[];
  loading?: boolean;
  onRangeChange: (range: WorkScheduleRange) => void;
  onCreate: (start: Date, end: Date) => void;
  onOpen: (event: WorkScheduleEvent) => void;
  onMove: (
    event: WorkScheduleEvent,
    start: Date,
    end: Date,
  ) => Promise<boolean>;
};

type WorkScheduleMoveArg = {
  event: EventApi;
  revert: () => void;
};

const scheduleCalendarPlugins = [
  dayGridPlugin,
  timeGridPlugin,
  interactionPlugin,
];
const scheduleCalendarLocales = [zhCnLocale];

export function WorkScheduleCalendar({
  events,
  loading = false,
  onRangeChange,
  onCreate,
  onOpen,
  onMove,
}: WorkScheduleCalendarProps) {
  const calendarRef = useRef<FullCalendar | null>(null);
  const reportedDateProfileRef = useRef("");
  const [view, setView] = useState<WorkScheduleView>("dayGridMonth");
  const [title, setTitle] = useState("");
  const [anchor, setAnchor] = useState(() => new Date());
  const calendarEvents = useMemo(() => events.map(scheduleEventInput), [events]);

  const datesChanged = useCallback((info: DatesSetArg) => {
    const profileKey = calendarDateProfileKey(info);
    if (reportedDateProfileRef.current === profileKey) return;
    reportedDateProfileRef.current = profileKey;
    setTitle(info.view.title);
    setAnchor(new Date(info.view.currentStart));
    setView(info.view.type as WorkScheduleView);
    onRangeChange({ start: info.start, end: info.end });
  }, [onRangeChange]);

  const changeView = (nextView: WorkScheduleView) => {
    calendarRef.current?.getApi().changeView(nextView);
    setView(nextView);
  };

  const move = useCallback(async (info: WorkScheduleMoveArg) => {
    const schedule = info.event.extendedProps.schedule as WorkScheduleEvent;
    const start = info.event.start;
    if (!start) {
      info.revert();
      return;
    }
    const end = info.event.end ?? new Date(start.getTime() + 30 * 60_000);
    if (!(await onMove(schedule, start, end))) info.revert();
  }, [onMove]);

  const selectRange = useCallback((info: DateSelectArg) => {
    if (!info.allDay) {
      onCreate(info.start, info.end);
      return;
    }
    const start = new Date(info.start);
    start.setHours(9, 0, 0, 0);
    onCreate(start, new Date(start.getTime() + 30 * 60_000));
  }, [onCreate]);

  const openEvent = useCallback((info: EventClickArg) => {
    onOpen(info.event.extendedProps.schedule as WorkScheduleEvent);
  }, [onOpen]);

  const moveEvent = useCallback((info: WorkScheduleMoveArg) => {
    void move(info);
  }, [move]);

  return (
    <div className="crm-schedule-calendar-layout">
      <MiniScheduleCalendar
        anchor={anchor}
        onSelect={(date) => calendarRef.current?.getApi().gotoDate(date)}
      />
      <section className="crm-schedule-calendar-main" aria-label="日程日历">
        <div className="crm-schedule-toolbar">
          <div className="crm-schedule-date-controls">
            <Button
              type="button"
              variant="outline"
              onClick={() => calendarRef.current?.getApi().today()}
            >
              今天
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="上一时间段"
              title="上一时间段"
              onClick={() => calendarRef.current?.getApi().prev()}
            >
              <ChevronLeft size={18} />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="下一时间段"
              title="下一时间段"
              onClick={() => calendarRef.current?.getApi().next()}
            >
              <ChevronRight size={18} />
            </Button>
            <h2>{title}</h2>
          </div>
          <div className="crm-schedule-view-switch" aria-label="日历视图">
            {([
              ["timeGridDay", "日"],
              ["timeGridWeek", "周"],
              ["dayGridMonth", "月"],
            ] as Array<[WorkScheduleView, string]>).map(([value, label]) => (
              <Button
                key={value}
                type="button"
                variant={view === value ? "default" : "ghost"}
                onClick={() => changeView(value)}
              >
                {label}
              </Button>
            ))}
          </div>
        </div>
        <div className="crm-schedule-calendar-stage" data-loading={loading || undefined}>
          <FullCalendar
            ref={calendarRef}
            plugins={scheduleCalendarPlugins}
            locales={scheduleCalendarLocales}
            locale="zh-cn"
            initialView={view}
            headerToolbar={false}
            height="100%"
            expandRows
            nowIndicator
            selectable
            selectMirror
            editable
            eventStartEditable
            eventDurationEditable
            allDaySlot={false}
            slotMinTime="05:00:00"
            slotMaxTime="23:00:00"
            slotDuration="00:30:00"
            slotLabelInterval="01:00:00"
            dayMaxEvents={4}
            events={calendarEvents}
            datesSet={datesChanged}
            select={selectRange}
            eventClick={openEvent}
            eventDrop={moveEvent}
            eventResize={moveEvent}
            eventContent={renderScheduleEvent}
          />
        </div>
      </section>
    </div>
  );
}

function calendarDateProfileKey(info: DatesSetArg): string {
  return [
    info.view.type,
    info.view.title,
    info.view.currentStart.getTime(),
    info.start.getTime(),
    info.end.getTime(),
  ].join(":");
}

function renderScheduleEvent(info: EventContentArg) {
  const schedule = info.event.extendedProps.schedule as WorkScheduleEvent;
  return (
    <div className="crm-schedule-event-content">
      <strong>{info.event.title}</strong>
      {schedule.customer_name ? <span>{schedule.customer_name}</span> : null}
    </div>
  );
}

function scheduleEventInput(schedule: WorkScheduleEvent): EventInput {
  const type = schedule.schedule_type || "personal";
  const status = schedule.status || "pending";
  return {
    id: textValue(schedule.id),
    title: textValue(schedule.title) || "未命名日程",
    start: schedule.start_at,
    end: schedule.end_at,
    editable: status === "pending" && schedule.can_edit !== false,
    classNames: [
      `crm-schedule-event-${type}`,
      `crm-schedule-event-${status}`,
    ],
    extendedProps: { schedule },
  };
}

function MiniScheduleCalendar({
  anchor,
  onSelect,
}: {
  anchor: Date;
  onSelect: (date: Date) => void;
}) {
  const [visibleMonth, setVisibleMonth] = useState(
    () => new Date(anchor.getFullYear(), anchor.getMonth(), 1),
  );
  useEffect(() => {
    setVisibleMonth(new Date(anchor.getFullYear(), anchor.getMonth(), 1));
  }, [anchor]);
  const days = useMemo(() => miniCalendarDays(visibleMonth), [visibleMonth]);
  const todayKey = scheduleDateKey(new Date());
  const anchorKey = scheduleDateKey(anchor);

  return (
    <aside className="crm-schedule-mini" aria-label="小月历">
      <div className="crm-schedule-mini-header">
        <strong>
          {visibleMonth.getFullYear()}年{visibleMonth.getMonth() + 1}月
        </strong>
        <div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="上个月"
            title="上个月"
            onClick={() =>
              setVisibleMonth(
                new Date(visibleMonth.getFullYear(), visibleMonth.getMonth() - 1, 1),
              )
            }
          >
            <ChevronLeft size={15} />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="下个月"
            title="下个月"
            onClick={() =>
              setVisibleMonth(
                new Date(visibleMonth.getFullYear(), visibleMonth.getMonth() + 1, 1),
              )
            }
          >
            <ChevronRight size={15} />
          </Button>
        </div>
      </div>
      <div className="crm-schedule-mini-weekdays" aria-hidden="true">
        {"日一二三四五六".split("").map((day) => (
          <span key={day}>{day}</span>
        ))}
      </div>
      <div className="crm-schedule-mini-days">
        {days.map((date) => {
          const key = scheduleDateKey(date);
          const outside = date.getMonth() !== visibleMonth.getMonth();
          return (
            <button
              key={key}
              type="button"
              className={[
                outside ? "is-outside" : "",
                key === todayKey ? "is-today" : "",
                key === anchorKey ? "is-selected" : "",
              ]
                .filter(Boolean)
                .join(" ")}
              aria-label={`${date.getMonth() + 1}月${date.getDate()}日`}
              onClick={() => onSelect(date)}
            >
              {date.getDate()}
            </button>
          );
        })}
      </div>
      <div className="crm-schedule-legend">
        <span><CalendarDays size={13} />客户跟进</span>
        <span><Clock3 size={13} />个人日程</span>
      </div>
    </aside>
  );
}

function miniCalendarDays(month: Date): Date[] {
  const first = new Date(month.getFullYear(), month.getMonth(), 1);
  const start = new Date(first);
  start.setDate(first.getDate() - first.getDay());
  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(start);
    date.setDate(start.getDate() + index);
    return date;
  });
}

function scheduleDateKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}
