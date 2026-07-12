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
    --sidebar-width: 240px;
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
  }
`;

export function ShowCrmWorkSkin() {
  return <style>{crmWorkSkin}</style>;
}
