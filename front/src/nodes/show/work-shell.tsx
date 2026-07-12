import { useEffect, useMemo, useState } from "react";
import {
  BriefcaseBusiness,
  LayoutDashboard,
  PanelLeft,
  UsersRound,
} from "lucide-react";
import {
  SiteLogo,
  getSiteConfig,
  useNavigate,
} from "@dever/front-plugin";

type WorkNavItem = {
  path: string;
  title: string;
  icon: typeof LayoutDashboard;
};

const ROUTE_CHANGE_EVENT = "crm-work-route-change";

const workNavItems: WorkNavItem[] = [
  { path: "/crm/stats", title: "工作台", icon: LayoutDashboard },
  { path: "/crm/lead", title: "线索池", icon: UsersRound },
  { path: "/crm/work", title: "客户列表", icon: BriefcaseBusiness },
];

export function ShowCrmWorkSidebar() {
  const site = getSiteConfig();
  const navigate = useNavigate();
  const pathname = useWorkPath();
  const activePage = useMemo(() => resolveWorkPage(pathname), [pathname]);
  const [collapsed, setCollapsed] = useState(false);

  const openPage = async (path: string) => {
    try {
      await navigate({ to: path });
      notifyRouteChange(readWorkPath() || path);
    } catch {
      notifyRouteChange(readWorkPath());
    }
  };

  return (
    <aside
      className={cx("crm-body-sidebar", collapsed && "is-collapsed")}
      aria-label="CRM 工作台导航"
    >
      <WorkShellStyles />
      <div className="crm-body-sidebar-head">
        <div className="crm-body-brand" aria-label={site.name || "CRM 工作台"}>
          <SiteLogo className="crm-body-brand-logo" />
          <span>{site.name || "CRM 工作台"}</span>
        </div>
        <button
          type="button"
          className="crm-body-collapse"
          aria-label={collapsed ? "展开侧栏" : "收起侧栏"}
          aria-expanded={!collapsed}
          title={collapsed ? "展开侧栏" : "收起侧栏"}
          onClick={() => setCollapsed((value) => !value)}
        >
          <PanelLeft size={18} strokeWidth={1.9} />
        </button>
      </div>

      <nav className="crm-body-nav" aria-label="客户中心菜单">
        {workNavItems.map((nav) => (
          <WorkNavButton
            key={nav.path}
            active={activePage.path === nav.path}
            item={nav}
            onClick={() => void openPage(nav.path)}
          />
        ))}
      </nav>
    </aside>
  );
}

export function ShowCrmWorkTitlebar() {
  const pathname = useWorkPath();
  const page = useMemo(() => resolveWorkPage(pathname), [pathname]);

  return (
    <header className="crm-body-topbar">
      <h1>{page.title}</h1>
    </header>
  );
}

function WorkNavButton({
  active,
  item,
  onClick,
}: {
  active: boolean;
  item: WorkNavItem;
  onClick: () => void;
}) {
  const Icon = item.icon;

  return (
    <button
      type="button"
      className={cx("crm-body-nav-item", active && "is-active")}
      aria-current={active ? "page" : undefined}
      aria-label={item.title}
      title={item.title}
      onClick={onClick}
    >
      <Icon size={20} strokeWidth={1.85} />
      <span>{item.title}</span>
    </button>
  );
}

function useWorkPath() {
  const [pathname, setPathname] = useState(readWorkPath);

  useEffect(() => {
    const syncLocation = () => setPathname(readWorkPath());
    const syncNavigation = (event: Event) => {
      const nextPath = (event as CustomEvent<string>).detail;
      setPathname(nextPath || readWorkPath());
    };

    window.addEventListener("popstate", syncLocation);
    window.addEventListener("hashchange", syncLocation);
    window.addEventListener(ROUTE_CHANGE_EVENT, syncNavigation);

    return () => {
      window.removeEventListener("popstate", syncLocation);
      window.removeEventListener("hashchange", syncLocation);
      window.removeEventListener(ROUTE_CHANGE_EVENT, syncNavigation);
    };
  }, []);

  return pathname;
}

function resolveWorkPage(pathname: string) {
  return (
    workNavItems.find(
      ({ path }) => pathname === path || pathname.startsWith(`${path}/`),
    ) || workNavItems[0]
  );
}

function readWorkPath() {
  return typeof window === "undefined"
    ? workNavItems[0].path
    : window.location.pathname;
}

function notifyRouteChange(path: string) {
  window.dispatchEvent(
    new CustomEvent<string>(ROUTE_CHANGE_EVENT, { detail: path }),
  );
}

function WorkShellStyles() {
  return (
    <style>{`
      .crm-body-app {
        --crm-body-bg: #f4f6f5;
        --crm-body-sidebar-bg: #f4f6f5;
        --crm-body-surface: #ffffff;
        --crm-body-text: #171a19;
        --crm-body-muted: #6b7370;
        --crm-body-line: #e4e8e6;
        --crm-body-line-strong: #d2d9d6;
        --crm-body-active: #e4e8e6;
        position: fixed;
        inset: 0;
        z-index: 1;
        display: flex;
        width: 100vw;
        min-width: 100vw;
        height: 100vh;
        min-height: 100vh;
        overflow: hidden;
        background: var(--crm-body-bg);
        color: var(--crm-body-text);
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, "Noto Sans SC", "PingFang SC", sans-serif;
        font-size: 12.8px;
        letter-spacing: 0;
      }

      .crm-body-app *,
      .crm-body-app *::before,
      .crm-body-app *::after {
        box-sizing: border-box;
        letter-spacing: 0;
      }

      .crm-body-sidebar-slot,
      .crm-body-titlebar-slot {
        display: contents;
      }

      .crm-body-sidebar {
        display: flex;
        width: 240px;
        height: 100vh;
        flex: 0 0 240px;
        flex-direction: column;
        justify-content: flex-start;
        background: var(--crm-body-sidebar-bg);
        padding: 16px 8px 22px;
        transition: width 180ms ease, flex-basis 180ms ease, padding 180ms ease;
      }

      .crm-body-sidebar-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0 6px 22px 11px;
        transition: padding 180ms ease;
      }

      .crm-body-brand {
        display: inline-flex;
        min-width: 0;
        align-items: center;
        gap: 6px;
        color: var(--crm-body-text);
        font-size: 18px;
        font-weight: 700;
        line-height: 1;
      }

      .crm-body-brand span,
      .crm-body-nav-item span {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        transition: opacity 120ms ease, transform 120ms ease;
      }

      .crm-body-brand-logo {
        display: block;
        width: 20px;
        height: 20px;
        flex: 0 0 20px;
        object-fit: contain;
      }

      .crm-body-collapse,
      .crm-body-nav-item {
        appearance: none;
        border: 0;
        font: inherit;
        letter-spacing: 0;
      }

      .crm-body-collapse {
        display: inline-flex;
        width: 26px;
        height: 26px;
        flex: 0 0 26px;
        cursor: pointer;
        align-items: center;
        justify-content: center;
        border-radius: 6px;
        background: transparent;
        color: var(--crm-body-muted);
        transition: background-color 120ms ease, color 120ms ease, transform 180ms ease;
      }

      .crm-body-collapse svg {
        width: 14px;
        height: 14px;
      }

      .crm-body-collapse:hover {
        background: var(--crm-body-active);
        color: var(--crm-body-text);
      }

      .crm-body-nav {
        display: flex;
        flex-direction: column;
        gap: 3px;
      }

      .crm-body-nav-item {
        position: relative;
        display: flex;
        width: 100%;
        min-height: 40px;
        cursor: pointer;
        align-items: center;
        gap: 11px;
        border-radius: 6px;
        background: transparent;
        color: var(--crm-body-text);
        padding: 0 13px;
        text-align: left;
        transition: background-color 120ms ease, color 120ms ease;
      }

      .crm-body-nav-item svg {
        width: 16px;
        height: 16px;
        flex: 0 0 16px;
        color: var(--crm-body-muted);
      }

      .crm-body-nav-item span {
        min-width: 0;
        flex: 1;
        font-size: 12.8px;
        font-weight: 400;
        line-height: 1.2;
      }

      .crm-body-nav-item:hover,
      .crm-body-nav-item.is-active {
        background: var(--crm-body-active);
      }

      .crm-body-nav-item.is-active span {
        font-weight: 500;
      }

      .crm-body-sidebar.is-collapsed {
        width: 64px;
        flex-basis: 64px;
        padding: 16px 7px 18px;
      }

      .crm-body-sidebar.is-collapsed .crm-body-sidebar-head {
        justify-content: center;
        padding: 0 0 22px;
      }

      .crm-body-sidebar.is-collapsed .crm-body-brand {
        display: none;
      }

      .crm-body-sidebar.is-collapsed .crm-body-collapse {
        background: var(--crm-body-active);
        color: var(--crm-body-text);
        transform: rotate(180deg);
      }

      .crm-body-sidebar.is-collapsed .crm-body-nav {
        align-items: center;
      }

      .crm-body-sidebar.is-collapsed .crm-body-nav-item {
        width: 40px;
        min-height: 40px;
        justify-content: center;
        gap: 0;
        padding: 0;
      }

      .crm-body-sidebar.is-collapsed .crm-body-nav-item span {
        display: none;
      }

      .crm-body-main {
        min-width: 0;
        height: 100vh;
        flex: 1;
        overflow: hidden;
        background: var(--crm-body-bg);
        padding: 11px 11px 11px 0;
      }

      .crm-body-frame {
        display: flex;
        width: 100%;
        height: 100%;
        min-width: 0;
        flex-direction: column;
        overflow: hidden;
        border-radius: 6px;
        background: var(--crm-body-surface);
      }

      .crm-body-topbar {
        display: flex;
        width: 100%;
        height: 38px;
        flex: 0 0 38px;
        align-items: center;
        border-bottom: 1px solid var(--crm-body-line);
        background: var(--crm-body-surface);
        padding: 0 31px;
      }

      .crm-body-topbar h1 {
        margin: 0;
        color: var(--crm-body-text);
        font-size: 14.5px;
        font-weight: 500;
        line-height: 1;
      }

      .crm-body-content {
        min-width: 0;
        min-height: 0;
        flex: 1;
        overflow: auto;
        background: var(--crm-body-surface);
      }

      .crm-body-content > * {
        min-width: 0;
      }

      .crm-body-page {
        width: 100%;
        min-width: 0;
        min-height: 100%;
        background: var(--crm-body-surface);
        color: var(--crm-body-text);
        padding: 24px 31px 40px;
        font-size: 12.8px;
        line-height: 1.45;
      }

      @media (max-width: 900px) {
        .crm-body-sidebar {
          width: 190px;
          flex-basis: 190px;
        }

        .crm-body-topbar {
          padding: 0 19px;
        }

        .crm-body-page {
          padding: 22px 19px 32px;
        }
      }

      @media (max-width: 640px) {
        .crm-body-app {
          flex-direction: column;
        }

        .crm-body-sidebar,
        .crm-body-sidebar.is-collapsed {
          order: 2;
          width: 100%;
          height: 58px;
          flex: 0 0 58px;
          flex-direction: row;
          align-items: center;
          justify-content: center;
          overflow-x: auto;
          border-top: 1px solid var(--crm-body-line);
          padding: 6px 8px;
        }

        .crm-body-sidebar-head {
          display: none;
        }

        .crm-body-nav,
        .crm-body-sidebar.is-collapsed .crm-body-nav {
          width: max-content;
          min-width: 100%;
          flex-direction: row;
          justify-content: space-around;
          gap: 4px;
        }

        .crm-body-nav-item,
        .crm-body-sidebar.is-collapsed .crm-body-nav-item {
          width: 72px;
          min-height: 44px;
          flex: 0 0 72px;
          flex-direction: column;
          justify-content: center;
          gap: 4px;
          padding: 0;
        }

        .crm-body-nav-item span,
        .crm-body-sidebar.is-collapsed .crm-body-nav-item span {
          display: block;
          width: 100%;
          flex: none;
          font-size: 10px;
          font-weight: 500;
          text-align: center;
        }

        .crm-body-main {
          order: 1;
          width: 100%;
          height: auto;
          min-height: 0;
          flex: 1;
          padding: 0;
        }

        .crm-body-frame {
          border-radius: 0;
        }

        .crm-body-topbar {
          height: 40px;
          flex-basis: 40px;
          padding: 0 14px;
        }

        .crm-body-page {
          padding: 16px 14px 24px;
        }
      }
    `}</style>
  );
}

function cx(...classes: Array<string | false | null | undefined>) {
  return classes.filter(Boolean).join(" ");
}
