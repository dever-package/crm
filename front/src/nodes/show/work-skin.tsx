const crmWorkSkin = `
  .crm-work-app {
    --crm-work-bg: #f4f6f5;
    --crm-work-surface: #ffffff;
    --crm-work-text: #171a19;
    --crm-work-muted: #6b7370;
    --crm-work-faint: #9ca3a0;
    --crm-work-line: #e4e8e6;
    --crm-work-line-strong: #d2d9d6;
    --crm-work-active: #e4e8e6;
    --crm-work-primary: #1a4a35;
    --primary: #1a4a35;
    --primary-foreground: #ffffff;
    --ring: #6b8d7e;
    --sidebar-width: 240px;
    display: contents;
    min-height: 100svh;
    background: var(--crm-work-bg);
    color: var(--crm-work-text);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, "Noto Sans SC", "PingFang SC", sans-serif;
    font-size: 12.8px;
    letter-spacing: 0;
  }

  .crm-work-app *,
  .crm-work-app *::before,
  .crm-work-app *::after {
    box-sizing: border-box;
    letter-spacing: 0;
  }

  .crm-work-app [data-slot="sidebar-wrapper"] {
    background: var(--crm-work-bg);
  }

  .crm-work-app [data-slot="sidebar-container"] {
    padding: 11px 8px;
  }

  .crm-work-app [data-sidebar="sidebar"] {
    border: 0;
    border-radius: 0;
    background: var(--crm-work-bg);
    box-shadow: none;
  }

  .crm-work-app [data-sidebar="header"] {
    gap: 0;
    padding: 5px 11px 22px;
  }

  .crm-work-app [data-sidebar="header"] [data-sidebar="menu-button"] {
    height: 34px;
    gap: 8px;
    padding: 0;
    border-radius: 6px;
    color: var(--crm-work-text);
    font-size: 12.8px;
  }

  .crm-work-app [data-sidebar="header"] [data-sidebar="menu-button"] img {
    width: 20px;
    height: 20px;
  }

  .crm-work-app [data-sidebar="content"] {
    gap: 3px;
  }

  .crm-work-app [data-sidebar="group"] {
    padding: 0;
  }

  .crm-work-app [data-sidebar="menu"] {
    gap: 3px;
  }

  .crm-work-app [data-sidebar="content"] [data-sidebar="menu-button"] {
    min-height: 40px;
    gap: 11px;
    border-radius: 6px;
    padding: 0 13px;
    color: var(--crm-work-text);
    font-size: 12.8px;
    font-weight: 400;
  }

  .crm-work-app [data-sidebar="content"] [data-sidebar="menu-button"] svg {
    width: 16px;
    height: 16px;
    color: var(--crm-work-muted);
  }

  .crm-work-app [data-sidebar="content"] [data-sidebar="menu-button"]:hover,
  .crm-work-app [data-sidebar="content"] [data-sidebar="menu-button"].active,
  .crm-work-app [data-sidebar="content"] [data-sidebar="menu-button"][data-active="true"] {
    background: var(--crm-work-active);
    color: var(--crm-work-text);
  }

  .crm-work-app [data-sidebar="content"] [data-sidebar="menu-button"].active,
  .crm-work-app [data-sidebar="content"] [data-sidebar="menu-button"][data-active="true"] {
    font-weight: 500;
  }

  .crm-work-app [data-slot="sidebar-rail"]::after {
    background: transparent;
  }

  .crm-work-app [data-slot="sidebar-inset"] {
    margin: 11px 11px 11px 0;
    overflow: hidden;
    border-radius: 6px;
    background: var(--crm-work-surface);
    box-shadow: none;
  }

  .crm-work-app [data-slot="sidebar-inset"] > header {
    position: relative;
    top: auto;
    height: 38px;
    min-height: 38px;
    flex: 0 0 38px;
    border-color: var(--crm-work-line);
    background: var(--crm-work-surface);
    backdrop-filter: none;
  }

  .crm-work-app [data-slot="sidebar-inset"] > header > div {
    min-height: 38px;
    gap: 10px;
    padding: 0 18px;
  }

  .crm-work-app [data-slot="sidebar-inset"] > header button {
    border-radius: 6px;
    font-size: 12.8px;
  }

  .crm-work-app [data-sidebar="trigger"] {
    width: 26px;
    height: 26px;
    color: var(--crm-work-muted);
    box-shadow: none;
  }

  .crm-work-app [data-slot="sidebar-inset"] > header input {
    height: 28px;
    border-color: var(--crm-work-line);
    border-radius: 6px;
    background: #ffffff;
    font-size: 12px;
    box-shadow: none;
  }

  .crm-work-app .crm-work-page {
    width: 100%;
    min-width: 0;
    min-height: 100%;
    background: var(--crm-work-surface);
    color: var(--crm-work-text);
    padding: 24px 31px 40px;
    font-size: 12.8px;
    line-height: 1.45;
  }

  .crm-work-app .crm-work-page .text-xs {
    font-size: 10.5px;
    line-height: 1.4;
  }

  .crm-work-app .crm-work-page .text-sm {
    font-size: 12.8px;
    line-height: 1.45;
  }

  .crm-work-app .crm-work-page .text-base {
    font-size: 14px;
    line-height: 1.4;
  }

  .crm-work-app .crm-work-page .text-lg,
  .crm-work-app .crm-work-page .text-xl,
  .crm-work-app .crm-work-page .text-2xl {
    font-size: 14.5px;
    line-height: 1.35;
  }

  .crm-work-app .crm-work-page h1,
  .crm-work-app .crm-work-page h2,
  .crm-work-app .crm-work-page h3,
  .crm-work-app .crm-work-page h4,
  .crm-work-app .crm-work-page p {
    letter-spacing: 0;
  }

  .crm-work-app .crm-work-page h1 {
    color: var(--crm-work-text);
    font-size: 14.5px;
    font-weight: 500;
    line-height: 1.35;
  }

  .crm-work-app .crm-work-page .text-muted-foreground {
    color: var(--crm-work-muted);
  }

  .crm-work-app .crm-work-page .shadow-sm,
  .crm-work-app .crm-work-page .shadow-xs {
    box-shadow: none;
  }

  .crm-work-app .crm-work-page .rounded-lg {
    border-radius: 6px;
  }

  .crm-work-app .crm-work-page .rounded-md {
    border-radius: 6px;
  }

  .crm-work-app .crm-work-page section,
  .crm-work-app .crm-work-page form,
  .crm-work-app .crm-work-page table,
  .crm-work-app .crm-work-page [class*="border-border"] {
    border-color: var(--crm-work-line);
  }

  .crm-work-app .crm-work-page input,
  .crm-work-app .crm-work-page select,
  .crm-work-app .crm-work-page textarea {
    border-color: var(--crm-work-line-strong);
    border-radius: 6px;
    background: #ffffff;
    color: var(--crm-work-text);
    font: inherit;
    font-size: 12px;
    box-shadow: none;
  }

  .crm-work-app .crm-work-page input,
  .crm-work-app .crm-work-page select {
    height: 32px;
    min-height: 32px;
    padding-top: 0;
    padding-bottom: 0;
  }

  .crm-work-app .crm-work-page input:focus,
  .crm-work-app .crm-work-page select:focus,
  .crm-work-app .crm-work-page textarea:focus {
    border-color: #7f9f91;
    outline: none;
    box-shadow: 0 0 0 2px rgba(127, 159, 145, 0.14);
  }

  .crm-work-app .crm-work-page button[data-slot="button"] {
    height: 30px;
    min-height: 30px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    box-shadow: none;
  }

  .crm-work-app .crm-work-page button[data-slot="button"][data-size="icon"],
  .crm-work-app .crm-work-page button[data-slot="button"].size-9 {
    width: 30px;
    min-width: 30px;
    height: 30px;
    padding: 0;
  }

  .crm-work-app .crm-work-page button[data-slot="button"] svg {
    width: 14px;
    height: 14px;
  }

  .crm-work-app .crm-work-page table {
    color: var(--crm-work-text);
    font-size: 12.8px;
  }

  .crm-work-app .crm-work-page thead {
    background: #f7f8f7;
    color: var(--crm-work-muted);
  }

  .crm-work-app .crm-work-page th {
    height: 36px;
    padding: 8px 14px;
    color: var(--crm-work-muted);
    font-size: 11.5px !important;
    font-weight: 500;
    line-height: 1.25;
  }

  .crm-work-app .crm-work-page td {
    padding: 10px 14px;
    font-size: 12.8px;
    line-height: 1.4;
  }

  .crm-work-app .crm-work-page tbody tr {
    border-color: var(--crm-work-line);
  }

  .crm-work-app .crm-work-page tbody tr:hover {
    background: #fafbfa;
  }

  .crm-work-app .crm-work-leads,
  .crm-work-app .crm-work-customers,
  .crm-work-app .crm-work-stats {
    gap: 12px;
  }

  .crm-work-app .crm-work-leads > div:first-child,
  .crm-work-app .crm-work-customers > div:first-child {
    min-height: 32px;
    gap: 10px;
  }

  .crm-work-app .crm-work-leads > div:first-child h1,
  .crm-work-app .crm-work-customers > div:first-child h1 {
    margin: 0;
  }

  .crm-work-app .crm-work-leads > div:first-child p {
    margin-top: 2px;
    color: var(--crm-work-muted);
    font-size: 10.5px;
  }

  .crm-work-app .crm-work-leads > section {
    border-radius: 3px;
  }

  .crm-work-app .crm-work-leads > section > form {
    gap: 8px;
    padding: 10px 14px;
    background: #fafbfa;
  }

  .crm-work-app .crm-work-leads > section > form select {
    width: 150px;
  }

  .crm-work-app .crm-work-leads tbody .rounded-full,
  .crm-work-app .crm-work-customers tbody .rounded-full {
    min-height: 20px;
    align-items: center;
    border-radius: 4px;
    padding: 2px 7px;
    font-size: 10.5px;
    line-height: 1.25;
  }

  .crm-work-app .crm-work-customers > div:nth-child(2) {
    border: 1px solid var(--crm-work-line);
    border-radius: 3px;
    box-shadow: none;
  }

  .crm-work-app .crm-work-customers > div:nth-child(2) > form {
    gap: 8px;
    padding: 10px 14px;
    background: #fafbfa;
  }

  .crm-work-app .crm-work-customers table thead {
    background: #f7f8f7;
  }

  .crm-work-app .crm-work-customers table td,
  .crm-work-app .crm-work-customers table th {
    border-color: var(--crm-work-line);
  }

  .crm-work-app .crm-work-stats > div,
  .crm-work-app .crm-work-stats section,
  .crm-work-app .crm-work-stats button {
    border-color: var(--crm-work-line);
    box-shadow: none;
  }

  .crm-work-app .crm-work-stats > div:first-child {
    border-radius: 3px;
    padding: 12px 14px;
  }

  .crm-work-app .crm-work-stats .text-3xl {
    margin-top: 4px;
    font-size: 22px;
    font-weight: 600;
    line-height: 1.2;
  }

  .crm-work-app .crm-work-stats button.rounded-lg {
    border-radius: 3px;
    padding: 13px 14px;
  }

  .crm-work-app .crm-work-stats button.rounded-lg > div > span {
    width: 32px;
    height: 32px;
  }

  .crm-work-app .crm-work-stats button.rounded-lg > div > span svg {
    width: 16px;
    height: 16px;
  }

  .crm-work-app .crm-work-stats section.rounded-lg {
    border-radius: 3px;
    padding: 14px;
  }

  body:has(.crm-work-app) [data-slot="dialog-content"],
  body:has(.crm-work-app) [data-slot="sheet-content"] {
    --primary: #1a4a35;
    --primary-foreground: #ffffff;
    --ring: #6b8d7e;
    border-color: var(--crm-work-line, #e4e8e6);
    background: #ffffff;
    color: #171a19;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, "Noto Sans SC", "PingFang SC", sans-serif;
    font-size: 12.8px;
    letter-spacing: 0;
  }

  body:has(.crm-work-app) [data-slot="dialog-content"] {
    border-radius: 6px;
    box-shadow: 0 24px 70px rgba(16, 24, 21, 0.16);
  }

  body:has(.crm-work-app) [data-slot="sheet-content"] {
    border-color: #e4e8e6;
    box-shadow: -12px 0 36px rgba(16, 24, 21, 0.1);
  }

  body:has(.crm-work-app) [data-slot="dialog-content"] *,
  body:has(.crm-work-app) [data-slot="sheet-content"] * {
    letter-spacing: 0;
  }

  body:has(.crm-work-app) [data-slot="dialog-header"],
  body:has(.crm-work-app) [data-slot="sheet-header"] {
    gap: 3px;
    border-bottom: 1px solid #e4e8e6;
    padding: 13px 18px;
  }

  body:has(.crm-work-app) [data-slot="sheet-header"] {
    padding-right: 22px;
    padding-left: 22px;
  }

  body:has(.crm-work-app) [data-slot="dialog-title"],
  body:has(.crm-work-app) [data-slot="sheet-title"] {
    color: #171a19;
    font-size: 14.5px;
    font-weight: 600;
    line-height: 1.35;
  }

  body:has(.crm-work-app) [data-slot="dialog-description"],
  body:has(.crm-work-app) [data-slot="sheet-description"] {
    color: #6b7370;
    font-size: 11px;
    line-height: 1.4;
  }

  body:has(.crm-work-app) [data-slot="dialog-footer"] {
    flex-direction: row;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    border-top: 1px solid #e4e8e6;
    padding: 11px 18px;
  }

  body:has(.crm-work-app) [role="dialog"] .text-xs {
    font-size: 10.5px;
    line-height: 1.4;
  }

  body:has(.crm-work-app) [role="dialog"] .text-sm {
    font-size: 12.8px;
    line-height: 1.45;
  }

  body:has(.crm-work-app) [role="dialog"] .text-base {
    font-size: 14px;
    line-height: 1.4;
  }

  body:has(.crm-work-app) [role="dialog"] .text-lg,
  body:has(.crm-work-app) [role="dialog"] .text-xl,
  body:has(.crm-work-app) [role="dialog"] .text-2xl {
    font-size: 14.5px;
    line-height: 1.35;
  }

  body:has(.crm-work-app) [role="dialog"] .text-muted-foreground {
    color: #6b7370;
  }

  body:has(.crm-work-app) [role="dialog"] .shadow-sm,
  body:has(.crm-work-app) [role="dialog"] .shadow-xs {
    box-shadow: none;
  }

  body:has(.crm-work-app) [role="dialog"] .rounded-lg,
  body:has(.crm-work-app) [role="dialog"] .rounded-md {
    border-radius: 6px;
  }

  body:has(.crm-work-app) [role="dialog"] input,
  body:has(.crm-work-app) [role="dialog"] select,
  body:has(.crm-work-app) [role="dialog"] textarea {
    border-color: #d2d9d6;
    border-radius: 6px;
    background: #ffffff;
    color: #171a19;
    font: inherit;
    font-size: 12px;
    box-shadow: none;
  }

  body:has(.crm-work-app) [role="dialog"] input,
  body:has(.crm-work-app) [role="dialog"] select {
    height: 34px;
    min-height: 34px;
  }

  body:has(.crm-work-app) [role="dialog"] input:focus,
  body:has(.crm-work-app) [role="dialog"] select:focus,
  body:has(.crm-work-app) [role="dialog"] textarea:focus {
    border-color: #7f9f91;
    outline: none;
    box-shadow: 0 0 0 2px rgba(127, 159, 145, 0.14);
  }

  body:has(.crm-work-app) [role="dialog"] button[data-slot="button"] {
    height: 30px;
    min-height: 30px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    box-shadow: none;
  }

  body:has(.crm-work-app) [role="dialog"] button[data-slot="dialog-close"],
  body:has(.crm-work-app) [role="dialog"] button[data-slot="sheet-close"] {
    width: 28px;
    min-width: 28px;
    height: 28px;
    min-height: 28px;
    border-radius: 6px;
  }

  body:has(.crm-work-app) [role="dialog"] .crm-work-task-modal-body {
    padding: 12px 18px;
  }

  body:has(.crm-work-app) [role="dialog"] .crm-work-task-modal-body > div {
    margin-bottom: 10px;
  }

  body:has(.crm-work-app) [data-crm-work-detail="true"],
  body:has(.crm-work-app) [data-crm-work-record-detail="true"] {
    color: #171a19;
    font-size: 12.8px;
  }

  body:has(.crm-work-app) [data-crm-work-detail="true"] > div,
  body:has(.crm-work-app) [data-crm-work-record-detail="true"] {
    gap: 14px;
  }

  body:has(.crm-work-app) [data-crm-work-detail="true"] section,
  body:has(.crm-work-app) [data-crm-work-detail="true"] article,
  body:has(.crm-work-app) [data-crm-work-record-detail="true"] section,
  body:has(.crm-work-app) [data-crm-work-record-detail="true"] article {
    border-color: #e4e8e6;
    box-shadow: none;
  }

  body:has(.crm-work-app) [data-crm-work-detail="true"] [class*="bg-muted"],
  body:has(.crm-work-app) [data-crm-work-record-detail="true"] [class*="bg-muted"] {
    background-color: #fafbfa;
  }

  body:has(.crm-work-app) [data-crm-work-detail="true"] table,
  body:has(.crm-work-app) [data-crm-work-record-detail="true"] table {
    font-size: 12.8px;
  }

  body:has(.crm-work-app) [data-crm-work-detail="true"] th,
  body:has(.crm-work-app) [data-crm-work-record-detail="true"] th {
    color: #6b7370;
    font-size: 11.5px;
    font-weight: 500;
  }

  body:has(.crm-work-app) [data-crm-work-detail="true"] td,
  body:has(.crm-work-app) [data-crm-work-record-detail="true"] td {
    padding-top: 8px;
    padding-bottom: 8px;
  }

  body:has(.crm-work-app) [data-crm-work-detail="true"] button,
  body:has(.crm-work-app) [data-crm-work-record-detail="true"] button {
    font-size: 12px;
  }

  @media (max-width: 767px) {
    .crm-work-app [data-slot="sidebar-inset"] {
      margin: 0;
      border-radius: 0;
    }

    .crm-work-app [data-slot="sidebar-inset"] > header {
      height: 40px;
      min-height: 40px;
      flex-basis: 40px;
    }

    .crm-work-app [data-slot="sidebar-inset"] > header > div {
      min-height: 40px;
      padding: 0 10px;
    }

    .crm-work-app .crm-work-page {
      padding: 14px 12px 28px;
    }

    .crm-work-app .crm-work-leads > section > form select,
    .crm-work-app .crm-work-leads > section > form input,
    .crm-work-app .crm-work-customers > div:nth-child(2) > form label,
    .crm-work-app .crm-work-customers > div:nth-child(2) > form input {
      width: 100%;
      max-width: none;
    }

    body:has(.crm-work-app) [data-slot="dialog-content"] {
      width: calc(100vw - 20px);
      max-width: calc(100vw - 20px);
      max-height: calc(100svh - 20px);
    }

    body:has(.crm-work-app) [data-slot="sheet-content"] {
      width: min(100vw, 42rem);
      max-width: 100vw;
    }

    body:has(.crm-work-app) [data-slot="dialog-header"],
    body:has(.crm-work-app) [data-slot="sheet-header"] {
      padding: 12px 14px;
    }

    body:has(.crm-work-app) [data-slot="dialog-footer"] {
      padding: 10px 14px;
    }

    body:has(.crm-work-app) [role="dialog"] .crm-work-task-modal-body {
      padding: 10px 14px;
    }

    body:has(.crm-work-app) [data-crm-work-detail="true"] > div,
    body:has(.crm-work-app) [data-crm-work-record-detail="true"] {
      gap: 12px;
    }
  }
`;

export function ShowCrmWorkSkin() {
  return <style>{crmWorkSkin}</style>;
}
