export function WorkScheduleStyles() {
  return (
    <style>{`
      .crm-schedule-page-shell {
        height: 100%;
        min-height: 0;
        padding: 0 !important;
        overflow: hidden;
      }

      .crm-schedule-page-shell > *,
      .crm-schedule-page-shell > * > * {
        min-height: 0;
      }

      .crm-schedule-workspace {
        --schedule-line: #e2e7e5;
        --schedule-muted: #68716d;
        --schedule-ink: #171b19;
        --schedule-customer: #2563eb;
        --schedule-customer-soft: #eaf1ff;
        --schedule-personal: #16835f;
        --schedule-personal-soft: #e7f6f0;
        --schedule-meeting: #9f1239;
        --schedule-meeting-soft: #fff1f2;
        position: relative;
        display: flex;
        width: 100%;
        height: calc(100vh - 60px);
        min-height: 620px;
        flex-direction: column;
        overflow: hidden;
        background: #ffffff;
        color: var(--schedule-ink);
      }

      .crm-schedule-page-header {
        display: flex;
        min-height: 64px;
        flex: 0 0 64px;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        border-bottom: 1px solid var(--schedule-line);
        padding: 10px 20px;
      }

      .crm-schedule-page-header > div:first-child {
        display: flex;
        min-width: 0;
        align-items: baseline;
        gap: 10px;
      }

      .crm-schedule-page-header strong {
        font-size: 15px;
        font-weight: 650;
      }

      .crm-schedule-page-header span {
        color: var(--schedule-muted);
        font-size: 12px;
      }

      .crm-schedule-page-actions {
        display: flex;
        align-items: center;
        gap: 8px;
      }

      .crm-schedule-page-actions button {
        border-radius: 6px;
      }

      .crm-schedule-reminder-control {
        position: relative;
      }

      .crm-schedule-reminder-control > button > span {
        display: inline-flex;
        min-width: 18px;
        height: 18px;
        align-items: center;
        justify-content: center;
        border-radius: 9px;
        background: #dc2626;
        color: #ffffff;
        padding: 0 5px;
        font-size: 10px;
      }

      .crm-schedule-reminder-panel {
        position: absolute;
        z-index: 50;
        top: calc(100% + 8px);
        right: 0;
        width: min(340px, calc(100vw - 28px));
        overflow: hidden;
        border: 1px solid var(--schedule-line);
        border-radius: 6px;
        background: #ffffff;
        box-shadow: 0 16px 42px rgba(15, 23, 20, 0.14);
      }

      .crm-schedule-reminder-title {
        display: flex;
        align-items: center;
        justify-content: space-between;
        border-bottom: 1px solid var(--schedule-line);
        padding: 12px 14px;
      }

      .crm-schedule-reminder-panel > p {
        margin: 0;
        color: var(--schedule-muted);
        padding: 26px 14px;
        text-align: center;
      }

      .crm-schedule-reminder-list {
        max-height: 360px;
        overflow: auto;
        padding: 5px;
      }

      .crm-schedule-reminder-list button {
        display: grid;
        width: 100%;
        cursor: pointer;
        grid-template-columns: minmax(0, 1fr) auto;
        gap: 3px 10px;
        border: 0;
        border-radius: 5px;
        background: transparent;
        padding: 10px;
        text-align: left;
      }

      .crm-schedule-reminder-list button:hover {
        background: #f3f6f5;
      }

      .crm-schedule-reminder-list strong {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .crm-schedule-reminder-list span {
        color: #b42318;
        font-variant-numeric: tabular-nums;
      }

      .crm-schedule-reminder-list small {
        grid-column: 1 / -1;
        color: var(--schedule-muted);
      }

      .crm-schedule-calendar-layout {
        display: grid;
        min-width: 0;
        min-height: 0;
        flex: 1;
        grid-template-columns: 250px minmax(0, 1fr);
        overflow: hidden;
      }

      .crm-schedule-mini {
        min-height: 0;
        overflow-y: auto;
        border-right: 1px solid var(--schedule-line);
        background: #f7f9f8;
        padding: 18px 14px;
      }

      .crm-schedule-mini-header {
        display: flex;
        height: 34px;
        align-items: center;
        justify-content: space-between;
        padding: 0 2px 5px;
      }

      .crm-schedule-mini-header strong {
        font-size: 13px;
        font-weight: 650;
      }

      .crm-schedule-mini-header > div {
        display: flex;
      }

      .crm-schedule-mini-header button {
        width: 28px;
        height: 28px;
      }

      .crm-schedule-mini-weekdays,
      .crm-schedule-mini-days {
        display: grid;
        grid-template-columns: repeat(7, minmax(0, 1fr));
      }

      .crm-schedule-mini-weekdays span {
        display: flex;
        height: 30px;
        align-items: center;
        justify-content: center;
        color: #7a8580;
        font-size: 11px;
      }

      .crm-schedule-mini-days button {
        display: flex;
        aspect-ratio: 1;
        min-width: 0;
        cursor: pointer;
        align-items: center;
        justify-content: center;
        border: 0;
        border-radius: 5px;
        background: transparent;
        color: var(--schedule-ink);
        font-size: 11px;
      }

      .crm-schedule-mini-days button:hover {
        background: #e9eeec;
      }

      .crm-schedule-mini-days button.is-outside {
        color: #b5bcb9;
      }

      .crm-schedule-mini-days button.is-today {
        box-shadow: inset 0 0 0 1px var(--schedule-customer);
        color: var(--schedule-customer);
      }

      .crm-schedule-mini-days button.is-selected {
        background: #dfe6e3;
        color: #111513;
        font-weight: 650;
      }

      .crm-schedule-legend {
        display: grid;
        gap: 10px;
        border-top: 1px solid var(--schedule-line);
        margin-top: 18px;
        padding: 16px 4px 0;
      }

      .crm-schedule-legend span {
        display: flex;
        align-items: center;
        gap: 7px;
        color: var(--schedule-muted);
      }

      .crm-schedule-legend span:first-child svg {
        color: var(--schedule-customer);
      }

      .crm-schedule-legend span:last-child svg {
        color: var(--schedule-personal);
      }

      .crm-schedule-calendar-main {
        display: flex;
        min-width: 0;
        min-height: 0;
        flex-direction: column;
        overflow: hidden;
      }

      .crm-schedule-toolbar {
        display: flex;
        min-height: 64px;
        flex: 0 0 64px;
        align-items: center;
        justify-content: space-between;
        gap: 14px;
        border-bottom: 1px solid var(--schedule-line);
        padding: 9px 18px;
      }

      .crm-schedule-date-controls {
        display: flex;
        min-width: 0;
        align-items: center;
        gap: 4px;
      }

      .crm-schedule-date-controls h2 {
        min-width: 0;
        overflow: hidden;
        margin: 0 0 0 8px;
        font-size: 16px;
        font-weight: 650;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .crm-schedule-date-controls button,
      .crm-schedule-view-switch button {
        height: 34px;
        border-radius: 5px;
      }

      .crm-schedule-date-controls button[aria-label] {
        width: 32px;
        padding: 0;
      }

      .crm-schedule-view-switch {
        display: grid;
        grid-template-columns: repeat(3, 50px);
        border: 1px solid var(--schedule-line);
        border-radius: 6px;
        padding: 2px;
      }

      .crm-schedule-view-switch button {
        width: 50px;
        border: 0;
      }

      .crm-schedule-calendar-stage {
        position: relative;
        min-width: 0;
        min-height: 0;
        flex: 1;
        overflow: hidden;
      }

      .crm-schedule-calendar-stage[data-loading]::after {
        position: absolute;
        z-index: 10;
        inset: 0;
        content: "";
        background: rgba(255, 255, 255, 0.46);
        pointer-events: none;
      }

      .crm-schedule-calendar-stage .fc {
        height: 100%;
        color: var(--schedule-ink);
        font-family: inherit;
        font-size: 12px;
      }

      .crm-schedule-calendar-stage .fc-theme-standard td,
      .crm-schedule-calendar-stage .fc-theme-standard th,
      .crm-schedule-calendar-stage .fc-theme-standard .fc-scrollgrid {
        border-color: var(--schedule-line);
      }

      .crm-schedule-calendar-stage .fc-col-header-cell {
        background: #fbfcfc;
      }

      .crm-schedule-calendar-stage .fc-col-header-cell-cushion {
        color: #3e4743;
        padding: 10px 4px;
        font-weight: 600;
      }

      .crm-schedule-calendar-stage .fc-timegrid-slot-label-cushion,
      .crm-schedule-calendar-stage .fc-daygrid-day-number {
        color: #74807a;
        font-variant-numeric: tabular-nums;
      }

      .crm-schedule-calendar-stage .fc-timegrid-now-indicator-line {
        border-color: #dc2626;
      }

      .crm-schedule-calendar-stage .fc-timegrid-now-indicator-arrow {
        border-left-color: #dc2626;
        border-right-color: #dc2626;
      }

      .crm-schedule-calendar-stage .fc-event {
        overflow: hidden;
        border: 0;
        border-left: 3px solid currentColor;
        border-radius: 4px;
        box-shadow: none;
        padding: 2px 4px;
      }

      .crm-schedule-calendar-stage .crm-schedule-event-customer_follow {
        background: var(--schedule-customer-soft);
        color: #174ea6;
      }

      .crm-schedule-calendar-stage .crm-schedule-event-personal {
        background: var(--schedule-personal-soft);
        color: #116149;
      }

      .crm-schedule-calendar-stage .crm-schedule-event-meeting {
        background: var(--schedule-meeting-soft);
        color: var(--schedule-meeting);
      }

      .crm-schedule-calendar-stage .crm-schedule-event-completed,
      .crm-schedule-calendar-stage .crm-schedule-event-canceled {
        opacity: 0.48;
      }

      .crm-schedule-event-content {
        display: grid;
        min-width: 0;
        gap: 1px;
      }

      .crm-schedule-event-content strong,
      .crm-schedule-event-content span {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .crm-schedule-event-content strong {
        font-size: 11px;
        font-weight: 650;
      }

      .crm-schedule-event-content span {
        font-size: 10px;
        opacity: 0.78;
      }

      @media (max-width: 900px) {
        .crm-schedule-calendar-layout {
          grid-template-columns: 210px minmax(0, 1fr);
        }

        .crm-schedule-mini {
          padding-inline: 10px;
        }

      }

      @media (max-width: 640px) {
        .crm-schedule-workspace {
          height: calc(100vh - 98px);
          min-height: 520px;
        }

        .crm-schedule-page-header {
          min-height: 58px;
          flex: 0 0 auto;
          padding: 8px 10px;
        }

        .crm-schedule-page-header > div:first-child {
          display: none;
        }

        .crm-schedule-page-actions {
          width: 100%;
          justify-content: flex-end;
        }

        .crm-schedule-calendar-layout {
          display: block;
        }

        .crm-schedule-mini {
          display: none;
        }

        .crm-schedule-calendar-main {
          height: 100%;
        }

        .crm-schedule-toolbar {
          min-height: 56px;
          flex: 0 0 auto;
          gap: 6px;
          padding: 7px 9px;
        }

        .crm-schedule-date-controls {
          flex: 1;
        }

        .crm-schedule-date-controls h2 {
          max-width: 118px;
          margin-left: 2px;
          font-size: 13px;
        }

        .crm-schedule-date-controls > button:first-child {
          display: none;
        }

        .crm-schedule-view-switch {
          grid-template-columns: repeat(3, 36px);
        }

        .crm-schedule-view-switch button {
          width: 36px;
          padding: 0;
        }

        .crm-schedule-calendar-stage .fc-timegrid-axis {
          width: 42px;
        }

        .crm-schedule-calendar-stage .fc-timegrid-slot-label-cushion {
          font-size: 10px;
        }

      }
    `}</style>
  );
}
