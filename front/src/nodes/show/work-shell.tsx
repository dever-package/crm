import {
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import {
  BriefcaseBusiness,
  CalendarDays,
  LayoutDashboard,
  LoaderCircle,
  LogOut,
  PanelLeft,
  Search,
  UsersRound,
  X,
} from "lucide-react";
import { SiteLogo, getSiteConfig, useNavigate } from "@dever/front-plugin";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

import {
  clearWorkSession,
  getWorkEntryPath,
  readWorkSessionUser,
  textValue,
  workApi,
  workListSearchEvent,
  workRefreshEvent,
} from "./work-core";

type WorkNavItem = {
  to: string;
  title: string;
  icon: typeof LayoutDashboard;
  workflowId?: string;
  subjectType?: string;
  pendingCount?: number;
};

type WorkNavigationRow = {
  id?: string | number;
  name?: string;
  subject_type?: string;
  path?: string;
  pending_count?: string | number;
};

type WorkSearchResult = {
  type?: string;
  type_name?: string;
  id?: string | number;
  title?: string;
  subtitle?: string;
  path?: string;
  workflow_id?: string | number;
};

type HistoryStateMethod = (
  data: unknown,
  unused: string,
  url?: string | URL | null,
) => void;

const HISTORY_CHANGE_EVENT = "crm-work-history-change";
let historyObserverInstalled = false;

const workbenchNavItem: WorkNavItem = {
  to: "/crm/stats",
  title: "工作台",
  icon: LayoutDashboard,
};
const scheduleNavItem: WorkNavItem = {
  to: "/crm/schedule",
  title: "日程",
  icon: CalendarDays,
};
let workNavigationSnapshot: WorkNavItem[] = [workbenchNavItem];
let workNavigationPromise: Promise<void> | null = null;
const workNavigationListeners = new Set<() => void>();

export function ShowCrmWorkSidebar() {
  const site = getSiteConfig();
  const navigate = useNavigate();
  const location = useWorkLocation();
  const workNavItems = useWorkNavigation();
  const activePage = useMemo(
    () => resolveWorkPage(location, workNavItems),
    [location, workNavItems],
  );
  const [collapsed, setCollapsed] = useState(false);

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
        <Button
          type="button"
          variant="ghost"
          className="crm-body-collapse"
          aria-label={collapsed ? "展开侧栏" : "收起侧栏"}
          aria-expanded={!collapsed}
          title={collapsed ? "展开侧栏" : "收起侧栏"}
          onClick={() => setCollapsed((value) => !value)}
        >
          <PanelLeft size={18} strokeWidth={1.9} />
        </Button>
      </div>

      <nav className="crm-body-nav" aria-label="客户中心菜单">
        {workNavItems.map((nav) => (
          <WorkNavButton
            key={nav.to}
            active={activePage.to === nav.to}
            item={nav}
            onClick={() => void navigate({ to: nav.to })}
          />
        ))}
      </nav>
    </aside>
  );
}

export function ShowCrmWorkTitlebar() {
  const location = useWorkLocation();
  const workNavItems = useWorkNavigation();
  const page = useMemo(
    () => resolveWorkPage(location, workNavItems),
    [location, workNavItems],
  );
  const user = useMemo(readWorkSessionUser, []);
  const userName = textValue(user.name) || "当前账号";
  const userPhone = textValue(user.phone);

  return (
    <header className="crm-body-topbar">
      <h1>{page.title}</h1>
      <WorkGlobalSearch />
      <WorkAccountMenu userName={userName} userPhone={userPhone} />
    </header>
  );
}

function WorkAccountMenu({
  userName,
  userPhone,
}: {
  userName: string;
  userPhone: string;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!open) return;
    const closeOnOutsideClick = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", closeOnOutsideClick);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("mousedown", closeOnOutsideClick);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [open]);

  return (
    <div ref={containerRef} className="crm-body-account">
      <Button
        type="button"
        variant="ghost"
        className="crm-body-account-trigger"
        aria-label={`${userName}，打开账号菜单`}
        aria-haspopup="menu"
        aria-expanded={open}
        title="账号菜单"
        onClick={() => setOpen((value) => !value)}
      >
        <span aria-hidden="true">{workAccountInitial(userName)}</span>
      </Button>
      {open ? (
        <div className="crm-body-account-menu" role="menu">
          <div className="crm-body-account-profile">
            <strong>{userName}</strong>
            {userPhone ? <span>{userPhone}</span> : null}
          </div>
          <Button
            type="button"
            variant="ghost"
            role="menuitem"
            onClick={handleWorkLogout}
          >
            <LogOut size={15} aria-hidden="true" />
            退出登录
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function workAccountInitial(name: string) {
  return Array.from(name.trim())[0]?.toLocaleUpperCase() || "账";
}

function handleWorkLogout() {
  clearWorkSession();
  const entry = getWorkEntryPath().replace(/\/+$/, "");
  window.location.replace(`${entry}/login`);
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
    <Button
      type="button"
      variant="ghost"
      className={cx("crm-body-nav-item", active && "is-active")}
      aria-current={active ? "page" : undefined}
      aria-label={item.title}
      title={item.title}
      onClick={onClick}
    >
      <Icon size={20} strokeWidth={1.85} />
      <span>{item.title}</span>
      {item.pendingCount ? (
        <small className="crm-body-nav-count">{item.pendingCount}</small>
      ) : null}
    </Button>
  );
}

function useWorkNavigation() {
  const rows = useSyncExternalStore(
    subscribeWorkNavigation,
    readWorkNavigation,
    readWorkNavigation,
  );
  useEffect(() => {
    void loadWorkNavigation();
    const refresh = () => void loadWorkNavigation();
    window.addEventListener(workRefreshEvent, refresh);
    return () => window.removeEventListener(workRefreshEvent, refresh);
  }, []);
  return rows;
}

function subscribeWorkNavigation(listener: () => void) {
  workNavigationListeners.add(listener);
  return () => workNavigationListeners.delete(listener);
}

function readWorkNavigation() {
  return workNavigationSnapshot;
}

async function loadWorkNavigation() {
  if (workNavigationPromise) return workNavigationPromise;
  workNavigationPromise = Promise.all([
    workApi<{ list?: WorkNavigationRow[] }>("/crm/work/navigation"),
    workApi<{ total?: string | number }>("/crm/work/schedule_reminders").catch(
      () => ({ total: 0 }),
    ),
  ])
    .then(([payload, reminderPayload]) => {
      const workflows = Array.isArray(payload.list) ? payload.list : [];
      workNavigationSnapshot = [
        workbenchNavItem,
        ...workflows.flatMap((row): WorkNavItem[] => {
          const workflowId = textValue(row.id);
          const title = textValue(row.name);
          const to = textValue(row.path);
          const subjectType = textValue(row.subject_type);
          if (!workflowId || !title || !to) return [];
          return [
            {
              to,
              title,
              workflowId,
              subjectType,
              pendingCount: Number(row.pending_count) || 0,
              icon: subjectType === "lead" ? UsersRound : BriefcaseBusiness,
            },
          ];
        }),
        {
          ...scheduleNavItem,
          pendingCount: Number(reminderPayload.total) || 0,
        },
      ];
      workNavigationListeners.forEach((listener) => listener());
    })
    .catch(() => undefined)
    .finally(() => {
      workNavigationPromise = null;
    });
  return workNavigationPromise;
}

function useWorkLocation() {
  return useSyncExternalStore(
    subscribeWorkPath,
    readWorkLocation,
    readWorkLocation,
  );
}

function resolveWorkPage(location: string, items: WorkNavItem[]) {
  const current = workLocationParts(location);
  if (workPathMatches(current.pathname, workbenchNavItem.to)) {
    return workbenchNavItem;
  }
  const workflowId = current.searchParams.get("workflow_id") || "";
  return (
    items.find(
      (item) =>
        item.workflowId === workflowId &&
        workPathMatches(current.pathname, workLocationParts(item.to).pathname),
    ) ||
    items.find((item) =>
      workPathMatches(current.pathname, workLocationParts(item.to).pathname),
    ) ||
    workbenchNavItem
  );
}

function readWorkLocation() {
  return typeof window === "undefined"
    ? workbenchNavItem.to
    : `${window.location.pathname}${window.location.search}`;
}

function workLocationParts(location: string) {
  return new URL(location || workbenchNavItem.to, "http://crm.local");
}

function workPathMatches(pathname: string, route: string) {
  return pathname.endsWith(route) || pathname.includes(`${route}/`);
}

function notifyWorkListSearch(
  keyword: string,
  workflowID: string,
  mode = "",
  scope = "",
) {
  window.dispatchEvent(
    new CustomEvent(workListSearchEvent, {
      detail: {
        keyword,
        workflow_id: workflowID,
        mode,
        scope,
      },
    }),
  );
}

function WorkGlobalSearch() {
  const navigate = useNavigate();
  const location = useWorkLocation();
  const containerRef = useRef<HTMLFormElement>(null);
  const focusedRef = useRef(false);
  const [keyword, setKeyword] = useState(() =>
    workLocationParts(readWorkLocation()).searchParams.get("keyword") || "",
  );
  const [rows, setRows] = useState<WorkSearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const close = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) {
        focusedRef.current = false;
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  useEffect(() => {
    const current = workLocationParts(location);
    const routeKeyword = current.searchParams.get("keyword") || "";
    setKeyword(routeKeyword);
    setRows([]);
    setOpen(false);
    if (
      workPathMatches(current.pathname, "/crm/lead") ||
      workPathMatches(current.pathname, "/crm/work")
    ) {
      window.setTimeout(
        () =>
          notifyWorkListSearch(
            routeKeyword,
            current.searchParams.get("workflow_id") || "",
            current.searchParams.get("mode") || "",
            current.searchParams.get("scope") || "",
          ),
        0,
      );
    }
  }, [location]);

  useEffect(() => {
    const query = keyword.trim();
    if (!query) {
      setRows([]);
      setLoading(false);
      return;
    }
    let active = true;
    const timer = window.setTimeout(() => {
      setLoading(true);
      void workApi<{ list?: WorkSearchResult[] }>(
        `/crm/work/global_search?keyword=${encodeURIComponent(query)}`,
      )
        .then((payload) => {
          if (!active) return;
          setRows(Array.isArray(payload.list) ? payload.list : []);
          setOpen(focusedRef.current);
        })
        .catch(() => {
          if (active) setRows([]);
        })
        .finally(() => {
          if (active) setLoading(false);
        });
    }, 220);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [keyword]);

  const navigateToSearchList = (to: string) => {
    const target = workLocationParts(to);
    const nextKeyword = target.searchParams.get("keyword") || keyword.trim();
    const workflowID = target.searchParams.get("workflow_id") || "";
    focusedRef.current = false;
    setOpen(false);
    void navigate({ to });
    window.setTimeout(() => {
      notifyWorkListSearch(
        nextKeyword,
        workflowID,
        target.searchParams.get("mode") || "",
        target.searchParams.get("scope") || "",
      );
    }, 80);
  };

  const submitCurrentListSearch = () => {
    const current = workLocationParts(location);
    const isListPage =
      workPathMatches(current.pathname, "/crm/lead") ||
      workPathMatches(current.pathname, "/crm/work");
    if (!isListPage) {
      const firstResult = rows[0];
      const to = textValue(firstResult?.path);
      if (to) navigateToSearchList(to);
      return;
    }
    const nextKeyword = keyword.trim();
    if (nextKeyword) {
      current.searchParams.set("keyword", nextKeyword);
    } else {
      current.searchParams.delete("keyword");
    }
    const to = `${current.pathname}${current.search ? current.search : ""}`;
    focusedRef.current = false;
    setOpen(false);
    void navigate({ to });
    notifyWorkListSearch(
      nextKeyword,
      current.searchParams.get("workflow_id") || "",
      current.searchParams.get("mode") || "",
      current.searchParams.get("scope") || "",
    );
  };

  const clearSearch = () => {
    const current = workLocationParts(location);
    const workflowID = current.searchParams.get("workflow_id") || "";
    focusedRef.current = false;
    setKeyword("");
    setRows([]);
    setOpen(false);
    if (current.searchParams.has("keyword")) {
      current.searchParams.delete("keyword");
      void navigate({
        to: `${current.pathname}${current.search ? current.search : ""}`,
      });
    }
    notifyWorkListSearch(
      "",
      workflowID,
      current.searchParams.get("mode") || "",
      current.searchParams.get("scope") || "",
    );
  };

  return (
    <form
      ref={containerRef}
      className="crm-body-global-search"
      onSubmit={(event) => {
        event.preventDefault();
        submitCurrentListSearch();
      }}
    >
      <Search size={15} aria-hidden="true" />
      <Input
        className="h-auto border-0 bg-transparent p-0 shadow-none focus-visible:ring-0"
        value={keyword}
        placeholder="搜索线索、客户或资产"
        aria-label="搜索线索、客户或资产"
        onFocus={() => {
          focusedRef.current = true;
          if (keyword.trim()) setOpen(true);
        }}
        onChange={(event) => setKeyword(event.target.value)}
      />
      {loading ? (
        <LoaderCircle
          className="crm-body-search-loading"
          size={14}
          aria-hidden="true"
        />
      ) : keyword ? (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label="清空搜索"
          title="清空搜索"
          onClick={clearSearch}
        >
          <X size={14} />
        </Button>
      ) : null}
      {open && keyword.trim() ? (
        <div className="crm-body-search-results" role="listbox">
          {rows.length ? (
            rows.map((row) => (
              <Button
                key={`${row.type}-${row.id}`}
                type="button"
                variant="ghost"
                role="option"
                onClick={() => {
                  const to = textValue(row.path);
                  if (!to) return;
                  navigateToSearchList(to);
                }}
              >
                <span>
                  <strong>{textValue(row.title) || "未命名"}</strong>
                  <small>{textValue(row.subtitle)}</small>
                </span>
                <em>{textValue(row.type_name)}</em>
              </Button>
            ))
          ) : loading ? null : (
            <p>未找到匹配记录</p>
          )}
        </div>
      ) : null}
    </form>
  );
}

function subscribeWorkPath(listener: () => void) {
  if (typeof window === "undefined") {
    return () => undefined;
  }

  installHistoryObserver();
  window.addEventListener(HISTORY_CHANGE_EVENT, listener);
  window.addEventListener("popstate", listener);
  window.addEventListener("hashchange", listener);

  return () => {
    window.removeEventListener(HISTORY_CHANGE_EVENT, listener);
    window.removeEventListener("popstate", listener);
    window.removeEventListener("hashchange", listener);
  };
}

function installHistoryObserver() {
  if (historyObserverInstalled || typeof window === "undefined") {
    return;
  }

  const history = window.history as History &
    Record<"pushState" | "replaceState", HistoryStateMethod>;

  for (const method of ["pushState", "replaceState"] as const) {
    const original = history[method].bind(window.history);
    history[method] = (...args: Parameters<HistoryStateMethod>) => {
      original(...args);
      window.dispatchEvent(new Event(HISTORY_CHANGE_EVENT));
    };
  }

  historyObserverInstalled = true;
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

      .crm-body-nav-count {
        display: inline-flex;
        min-width: 18px;
        height: 18px;
        flex: 0 0 auto;
        align-items: center;
        justify-content: center;
        border-radius: 9px;
        background: var(--crm-body-surface);
        color: var(--crm-body-muted);
        padding: 0 5px;
        font-size: 10px;
        font-style: normal;
        line-height: 1;
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

      .crm-body-sidebar.is-collapsed .crm-body-nav-count {
        position: absolute;
        top: 2px;
        right: 1px;
        min-width: 14px;
        height: 14px;
        border-radius: 7px;
        padding: 0 3px;
        font-size: 8px;
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
        justify-content: space-between;
        gap: 12px;
        border-bottom: 1px solid var(--crm-body-line);
        background: var(--crm-body-surface);
        padding: 0 31px;
      }

      .crm-body-topbar h1 {
	        width: 190px;
	        min-width: 0;
	        flex: 0 1 190px;
        overflow: hidden;
        margin: 0;
        color: var(--crm-body-text);
        font-size: 14.5px;
        font-weight: 500;
        line-height: 1;
        text-overflow: ellipsis;
	        white-space: nowrap;
	      }

	      .crm-body-global-search {
	        position: relative;
	        display: flex;
	        width: min(430px, 46vw);
	        height: 28px;
	        min-width: 180px;
	        align-items: center;
	        gap: 7px;
	        border-radius: 5px;
	        background: var(--crm-body-bg);
	        color: var(--crm-body-muted);
	        padding: 0 9px;
	      }

	      .crm-body-global-search > svg {
	        flex: 0 0 auto;
	      }

	      .crm-body-global-search input {
	        width: 100%;
	        min-width: 0;
	        border: 0;
	        outline: 0;
	        background: transparent;
	        color: var(--crm-body-text);
	        font: inherit;
	      }

	      .crm-body-global-search input::placeholder {
	        color: #89918e;
	      }

	      .crm-body-global-search > button {
	        display: inline-flex;
	        width: 20px;
	        height: 20px;
	        flex: 0 0 20px;
	        cursor: pointer;
	        align-items: center;
	        justify-content: center;
	        border: 0;
	        border-radius: 4px;
	        background: transparent;
	        color: var(--crm-body-muted);
	        padding: 0;
	      }

	      .crm-body-search-loading {
	        animation: crm-body-spin 800ms linear infinite;
	      }

	      .crm-body-search-results {
	        position: absolute;
	        z-index: 60;
	        top: 34px;
	        left: 0;
	        width: 100%;
	        max-height: 360px;
	        overflow-y: auto;
	        border: 1px solid var(--crm-body-line);
	        border-radius: 6px;
	        background: var(--crm-body-surface);
	        padding: 5px;
	        box-shadow: 0 12px 30px rgb(23 26 25 / 12%);
	      }

	      .crm-body-search-results > button {
	        display: flex;
	        width: 100%;
	        cursor: pointer;
	        align-items: center;
	        justify-content: space-between;
	        gap: 12px;
	        border: 0;
	        border-radius: 5px;
	        background: transparent;
	        color: var(--crm-body-text);
	        padding: 8px 9px;
	        text-align: left;
	      }

	      .crm-body-search-results > button:hover {
	        background: var(--crm-body-bg);
	      }

	      .crm-body-search-results > button > span {
	        display: grid;
	        min-width: 0;
	        gap: 2px;
	      }

	      .crm-body-search-results strong,
	      .crm-body-search-results small {
	        overflow: hidden;
	        text-overflow: ellipsis;
	        white-space: nowrap;
	      }

	      .crm-body-search-results strong {
	        font-size: 12.8px;
	        font-weight: 500;
	      }

	      .crm-body-search-results small,
	      .crm-body-search-results em,
	      .crm-body-search-results p {
	        color: var(--crm-body-muted);
	        font-size: 11px;
	        font-style: normal;
	      }

	      .crm-body-search-results em {
	        flex: 0 0 auto;
	      }

	      .crm-body-search-results p {
	        margin: 0;
	        padding: 12px 9px;
	        text-align: center;
	      }

      @keyframes crm-body-spin {
	        to { transform: rotate(360deg); }
	      }

      .crm-body-account {
        position: relative;
        display: inline-flex;
        flex: 0 0 auto;
      }

      .crm-body-account-trigger {
        display: inline-flex;
        width: 28px;
        height: 28px;
        flex: 0 0 28px;
        cursor: pointer;
        align-items: center;
        justify-content: center;
        border: 0;
        border-radius: 50%;
        background: var(--crm-body-active);
        color: var(--crm-body-text);
        font: inherit;
        font-size: 12px;
        font-weight: 600;
        line-height: 1;
        transition: background-color 120ms ease, box-shadow 120ms ease;
      }

      .crm-body-account-trigger:hover {
        background: var(--crm-body-line-strong);
      }

      .crm-body-account-trigger:focus-visible {
        outline: none;
        box-shadow: 0 0 0 2px var(--crm-body-surface), 0 0 0 4px var(--crm-body-text);
      }

      .crm-body-account-menu {
        position: absolute;
        z-index: 80;
        top: 34px;
        right: 0;
        width: 208px;
        overflow: hidden;
        border: 1px solid var(--crm-body-line);
        border-radius: 6px;
        background: var(--crm-body-surface);
        box-shadow: 0 12px 30px rgb(23 26 25 / 12%);
      }

      .crm-body-account-profile {
        display: grid;
        gap: 3px;
        padding: 11px 12px;
      }

      .crm-body-account-profile strong,
      .crm-body-account-profile span {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .crm-body-account-profile strong {
        color: var(--crm-body-text);
        font-size: 12.8px;
        font-weight: 500;
      }

      .crm-body-account-profile span {
        color: var(--crm-body-muted);
        font-size: 11px;
      }

      .crm-body-account-menu > button {
        display: flex;
        width: 100%;
        cursor: pointer;
        align-items: center;
        gap: 8px;
        border: 0;
        border-top: 1px solid var(--crm-body-line);
        background: transparent;
        color: #b42318;
        padding: 9px 12px;
        font: inherit;
        text-align: left;
      }

      .crm-body-account-menu > button:hover,
      .crm-body-account-menu > button:focus-visible {
        outline: none;
        background: #fff3f1;
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

	        .crm-body-topbar h1 {
	          width: 150px;
	          flex-basis: 150px;
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

	        .crm-body-nav-count {
	          position: absolute;
	          top: 1px;
	          right: 8px;
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

	        .crm-body-topbar h1 {
	          width: auto;
	          max-width: 96px;
	          flex: 0 1 96px;
	        }

	        .crm-body-global-search {
	          width: auto;
	          min-width: 0;
	          flex: 1;
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
