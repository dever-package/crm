# CRM Work Direct Body Shell Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 `bot/body` 同款自定义外壳替换 CRM 前台通用外壳，同时保留现有真实路由、权限和业务页面。

**Architecture:** 新建一个聚焦于展示的 `work-shell.tsx`，导出侧栏和标题栏两个 Dever 节点，并共享同一份路由配置与路径订阅逻辑。`work/main.json` 使用普通容器组合侧栏、标题栏和 `app-outlet`；线索、客户、统计组件继续负责原有数据与交互，不进入外壳组件。

**Tech Stack:** Dever Page JSON、React 19、TypeScript、`@dever/front-plugin`、Lucide React、原生 CSS。

**Verification constraint:** 按项目要求，不运行 `npm run build` 或任何自动化测试。仅使用差异检查、运行中的 Dever 环境和浏览器桌面/移动视口验收。

---

### Task 1: 新建 Bot Body 同构工作台外壳

**Files:**
- Create: `front/src/nodes/show/work-shell.tsx`
- Modify: `front/src/plugin.ts`

- [ ] **Step 1: 定义唯一的 CRM 工作台导航配置**

在 `work-shell.tsx` 中定义共享配置，避免侧栏、标题栏分别维护路径和名称：

```tsx
type WorkNavItem = {
  path: string;
  title: string;
  icon: typeof LayoutDashboard;
};

const workNavItems: WorkNavItem[] = [
  { path: "/crm/stats", title: "工作台", icon: LayoutDashboard },
  { path: "/crm/lead", title: "线索池", icon: UsersRound },
  { path: "/crm/work", title: "客户列表", icon: BriefcaseBusiness },
];

function resolveWorkPage(pathname: string) {
  return (
    workNavItems.find(
      ({ path }) => pathname === path || pathname.startsWith(`${path}/`),
    ) || workNavItems[0]
  );
}
```

- [ ] **Step 2: 实现路径同步 hook**

使用初始 `window.location.pathname`、`popstate` 和组件内部派发的 `crm-work-route-change` 事件同步活动菜单。点击菜单时调用 Dever 的 `useNavigate()`，不直接操作 `history`：

```tsx
const ROUTE_CHANGE_EVENT = "crm-work-route-change";

function useWorkPath() {
  const [path, setPath] = useState(() => window.location.pathname);
  useEffect(() => {
    const syncLocation = () => setPath(window.location.pathname);
    const syncNavigation = (event: Event) => {
      const nextPath = (event as CustomEvent<string>).detail;
      setPath(nextPath || window.location.pathname);
    };
    window.addEventListener("popstate", syncLocation);
    window.addEventListener(ROUTE_CHANGE_EVENT, syncNavigation);
    return () => {
      window.removeEventListener("popstate", syncLocation);
      window.removeEventListener(ROUTE_CHANGE_EVENT, syncNavigation);
    };
  }, []);
  return path;
}
```

- [ ] **Step 3: 实现侧栏与标题栏节点**

`ShowCrmWorkSidebar` 只负责品牌、折叠状态和真实路由跳转；`ShowCrmWorkTitlebar` 根据同一配置显示标题：

```tsx
export function ShowCrmWorkSidebar() {
  const site = getSiteConfig();
  const navigate = useNavigate();
  const pathname = useWorkPath();
  const [collapsed, setCollapsed] = useState(false);

  const openPage = (path: string) => {
    navigate({ to: path });
    window.dispatchEvent(new CustomEvent(ROUTE_CHANGE_EVENT, { detail: path }));
  };

  return (
    <aside className={cx("crm-body-sidebar", collapsed && "is-collapsed")}>
      <WorkShellStyles />
      <div className="crm-body-sidebar-head">
        <div className="crm-body-brand">
          <SiteLogo className="crm-body-brand-logo" />
          <span>{site.name || "CRM 工作台"}</span>
        </div>
        <button
          type="button"
          className="crm-body-collapse"
          aria-label={collapsed ? "展开侧栏" : "收起侧栏"}
          title={collapsed ? "展开侧栏" : "收起侧栏"}
          onClick={() => setCollapsed((value) => !value)}
        >
          <PanelLeft size={18} strokeWidth={1.9} />
        </button>
      </div>
      <nav className="crm-body-nav" aria-label="CRM 工作台导航">
        {workNavItems.map(({ path, title, icon: Icon }) => (
          <button
            key={path}
            type="button"
            className={cx(
              "crm-body-nav-item",
              resolveWorkPage(pathname).path === path && "is-active",
            )}
            aria-current={resolveWorkPage(pathname).path === path ? "page" : undefined}
            title={title}
            onClick={() => openPage(path)}
          >
            <Icon size={20} strokeWidth={1.85} />
            <span>{title}</span>
          </button>
        ))}
      </nav>
    </aside>
  );
}

export function ShowCrmWorkTitlebar() {
  const pathname = useWorkPath();
  return <header className="crm-body-topbar"><h1>{resolveWorkPage(pathname).title}</h1></header>;
}
```

- [ ] **Step 4: 直接改编 Bot Body CSS**

将 `home-shell.tsx` 的结构参数改为 `crm-body-*` 作用域：固定全屏外壳、240px 侧栏、11px 主区间距、6px 内容框、38px 标题栏、`12.8px` 系统字体。移动端在 `640px` 以下把侧栏变为 58px 底部导航；不复制积分、社区、新建等无关视觉。

- [ ] **Step 5: 注册新节点**

在 `front/src/plugin.ts` 使用一个懒加载入口注册：

```tsx
const loadWorkShell = () => import("./nodes/show/work-shell");

"show-crm-work-sidebar": lazyNode(() =>
  loadWorkShell().then((mod) => ({ default: mod.ShowCrmWorkSidebar })),
),
"show-crm-work-titlebar": lazyNode(() =>
  loadWorkShell().then((mod) => ({ default: mod.ShowCrmWorkTitlebar })),
),
```

- [ ] **Step 6: 提交外壳组件**

```bash
git add front/src/nodes/show/work-shell.tsx front/src/plugin.ts
git commit -m "feat: add crm body work shell"
```

### Task 2: 用新外壳组合真实路由出口

**Files:**
- Modify: `front/page/work/main.json`
- Modify: `front/page/work/stats.json`
- Modify: `front/page/work/lead.json`
- Modify: `front/page/work/work.json`
- Delete: `front/src/nodes/show/work-skin.tsx`
- Modify: `front/src/plugin.ts`

- [ ] **Step 1: 重组主页面 JSON**

将 `work/main.json` 改为普通容器组合，不再挂载通用侧栏、通用顶栏、助手和旧皮肤：

```json
{
  "layout": {
    "type": "container",
    "className": "crm-body-app",
    "children": {
      "sidebar": { "type": "container", "className": "crm-body-sidebar-slot" },
      "main": {
        "type": "container",
        "className": "crm-body-main",
        "children": {
          "frame": {
            "type": "container",
            "className": "crm-body-frame",
            "children": {
              "titlebar": { "type": "container", "className": "crm-body-titlebar-slot" },
              "content": { "type": "container", "className": "crm-body-content" }
            }
          }
        }
      }
    }
  },
  "nodes": {
    "sidebar": [{ "type": "show-crm-work-sidebar" }],
    "titlebar": [{ "type": "show-crm-work-titlebar" }],
    "content": [{ "type": "app-outlet" }]
  }
}
```

- [ ] **Step 2: 给三个业务页面使用新的内容根类**

把上一轮 `crm-work-page crm-work-*-page` 替换为单一的 `crm-body-page`。页面标题由外壳显示，工作台页面移除内部重复的 `show-title`，仅保留刷新动作和统计内容。

- [ ] **Step 3: 删除旧皮肤实现与注册**

删除 `work-skin.tsx`，并从 `plugin.ts` 删除 `loadWorkSkin` 与 `show-crm-work-skin`。确认仓库中不存在 `show-crm-work-skin` 和 `crm-work-app` 引用。

- [ ] **Step 4: 提交页面组合变更**

```bash
git add front/page/work/main.json front/page/work/stats.json front/page/work/lead.json front/page/work/work.json front/src/plugin.ts
git add -u front/src/nodes/show/work-skin.tsx
git commit -m "refactor: replace crm generic work shell"
```

### Task 3: 清理上一轮样式专用标记

**Files:**
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/src/nodes/show/work-lead.tsx`

- [ ] **Step 1: 恢复业务组件的中性根节点**

删除只为旧 `work-skin` 选择器加入的 `crm-work-stats`、`crm-work-customers`、`crm-work-leads` 类。保留原有 Tailwind 布局类和所有业务逻辑。

- [ ] **Step 2: 删除旧皮肤专用详情标记**

移除 `data-crm-work-detail`、`crm-work-detail` 和 `data-crm-work-task-form`。详情组件恢复直接返回 `WorkAssetDetailContent` 或 `WorkCustomerDetailContent`，空数据恢复直接返回 `WorkEmptyText`。

- [ ] **Step 3: 检查变更范围并提交**

运行 `git diff --check`，再用 `rg` 确认旧皮肤标记已清空；不执行构建或测试：

```bash
git diff --check
rg -n "show-crm-work-skin|crm-work-app|data-crm-work-(detail|task-form)" front || true
git add front/src/nodes/show/work-auth.tsx front/src/nodes/show/work-lead.tsx
git commit -m "refactor: remove rejected crm work skin markers"
```

### Task 4: 浏览器验收与针对性修正

**Files:**
- Modify when required: `front/src/nodes/show/work-shell.tsx`
- Modify when required: `front/page/work/*.json`

- [ ] **Step 1: 确认 CTF/Dever 环境正在运行**

只检查现有进程和端口；环境未运行时执行项目既有 `dever run`，不启动构建或测试命令。

- [ ] **Step 2: 桌面视口逐页检查**

在 1920×1080 下依次访问 `/crm/stats`、`/crm/lead`、`/crm/work`，检查：真实 URL、活动菜单、标题、240px 侧栏、38px 标题栏、白色内容框、无旧通用顶栏和全局搜索。

- [ ] **Step 3: 检查折叠和浏览器历史**

点击侧栏折叠按钮，确认主内容自动扩展；依次点击三个菜单并使用浏览器前进、后退，确认标题与活动菜单同步。

- [ ] **Step 4: 移动视口检查**

在 390×844 下确认底部导航显示三个入口，品牌头与折叠按钮隐藏，内容没有被底部导航遮挡，文字和操作按钮不重叠。

- [ ] **Step 5: 检查关键业务动作**

打开线索池录入弹窗、执行搜索、打开客户详情，确认外壳替换未破坏现有弹窗、抽屉和数据请求。只做可逆的浏览器操作，不新增无意义业务数据。

- [ ] **Step 6: 进行必要的视觉修正并提交**

修正仅限新外壳的尺寸、溢出和响应式问题。完成后运行 `git diff --check` 并提交：

```bash
git add front/src/nodes/show/work-shell.tsx front/page/work
git commit -m "fix: refine crm body shell layout"
```

若浏览器验收无需修正，则不创建空提交。
