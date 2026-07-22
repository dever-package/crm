import { useEffect, useMemo, useState } from "react";
import { CalendarClock } from "lucide-react";

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
  cloneDispatchSchedule,
  type DispatchSchedule,
} from "./work-dispatch-types";

const dispatchWeekdays = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"];

export function WorkDispatchScheduleDialog({
  open,
  staffName,
  value,
  onOpenChange,
  onSave,
}: {
  open: boolean;
  staffName: string;
  value: DispatchSchedule;
  onOpenChange: (open: boolean) => void;
  onSave: (schedule: DispatchSchedule) => void;
}) {
  const [hours, setHours] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (open) setHours(scheduleToHours(value));
  }, [open, value]);

  const selectedCount = useMemo(() => hours.size, [hours]);

  const applyPreset = (preset: "all" | "weekday" | "empty") => {
    const next = new Set<string>();
    if (preset === "all") {
      for (let day = 1; day <= 7; day++) {
        for (let hour = 0; hour < 24; hour++) next.add(hourKey(day, hour));
      }
    }
    if (preset === "weekday") {
      for (let day = 1; day <= 5; day++) {
        for (let hour = 9; hour < 18; hour++) next.add(hourKey(day, hour));
      }
    }
    setHours(next);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="crm-dispatch-schedule-dialog max-w-4xl">
        <DialogHeader>
          <DialogTitle>工作时间</DialogTitle>
          <DialogDescription>
            {staffName || "当前员工"} · 蓝色时段参与自动派单
          </DialogDescription>
        </DialogHeader>

        <div className="crm-dispatch-schedule-toolbar">
          <div>
            <Button type="button" variant="outline" size="sm" onClick={() => applyPreset("all")}>
              全天
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={() => applyPreset("weekday")}>
              工作日
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={() => applyPreset("empty")}>
              清空
            </Button>
          </div>
          <span><CalendarClock size={15} />已选 {selectedCount} 小时/周</span>
        </div>

        <div className="crm-dispatch-schedule-scroll">
          <div className="crm-dispatch-schedule-grid" role="grid" aria-label="每周派单工作时间">
            <span className="crm-dispatch-schedule-corner" />
            {Array.from({ length: 24 }, (_, hour) => (
              <span key={`hour-${hour}`} className="crm-dispatch-schedule-hour">
                {hour}
              </span>
            ))}
            {dispatchWeekdays.map((weekday, weekdayIndex) => {
              const day = weekdayIndex + 1;
              return [
                <strong key={`label-${day}`} className="crm-dispatch-schedule-day">{weekday}</strong>,
                ...Array.from({ length: 24 }, (_, hour) => {
                  const key = hourKey(day, hour);
                  const selected = hours.has(key);
                  return (
                    <button
                      key={key}
                      type="button"
                      className={selected ? "is-selected" : ""}
                      aria-label={`${weekday} ${String(hour).padStart(2, "0")}:00-${String(hour + 1).padStart(2, "0")}:00`}
                      aria-pressed={selected}
                      title={`${weekday} ${String(hour).padStart(2, "0")}:00`}
                      onClick={() => {
                        setHours((current) => {
                          const next = new Set(current);
                          if (next.has(key)) next.delete(key);
                          else next.add(key);
                          return next;
                        });
                      }}
                    />
                  );
                }),
              ];
            })}
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            type="button"
            onClick={() => {
              onSave(hoursToSchedule(hours));
              onOpenChange(false);
            }}
          >
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function hourKey(day: number, hour: number) {
  return `${day}:${hour}`;
}

function scheduleToHours(value: DispatchSchedule) {
  const schedule = cloneDispatchSchedule(value);
  const result = new Set<string>();
  for (let day = 1; day <= 7; day++) {
    for (const [start, end] of schedule[String(day)] || []) {
      const firstHour = Math.max(0, Math.floor(start / 60));
      const lastHour = Math.min(24, Math.ceil(end / 60));
      for (let hour = firstHour; hour < lastHour; hour++) {
        result.add(hourKey(day, hour));
      }
    }
  }
  return result;
}

function hoursToSchedule(hours: Set<string>): DispatchSchedule {
  const result: DispatchSchedule = {};
  for (let day = 1; day <= 7; day++) {
    const selected = Array.from({ length: 24 }, (_, hour) => hour).filter((hour) =>
      hours.has(hourKey(day, hour)),
    );
    const periods: Array<[number, number]> = [];
    let start: number | null = null;
    for (let hour = 0; hour <= 24; hour++) {
      const active = selected.includes(hour);
      if (active && start === null) start = hour;
      if (!active && start !== null) {
        periods.push([start * 60, hour * 60]);
        start = null;
      }
    }
    result[String(day)] = periods;
  }
  return result;
}
