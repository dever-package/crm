import { defineFrontPlugin, lazyNode } from "@dever/front-plugin";

const loadWorkAuth = () => import("./nodes/show/work-auth");
const loadWorkLead = () => import("./nodes/show/work-lead");
const loadWorkShell = () => import("./nodes/show/work-shell");
const loadWorkSkin = () => import("./nodes/show/work-skin");
const loadAdminStats = () => import("./nodes/show/admin-stats");

export default defineFrontPlugin({
  name: "crm",
  nodes: {
    "show-crm-work-login": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkLogin,
      })),
    ),
    "show-crm-work-tasks": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkTasks,
      })),
    ),
    "show-crm-work-refresh-button": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkRefreshButton,
      })),
    ),
    "show-crm-work-header-actions": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkHeaderActions,
      })),
    ),
    "show-crm-work-task-form": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkTaskForm,
      })),
    ),
    "show-crm-work-task-group-tabs": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkTaskGroupTabs,
      })),
    ),
    "show-crm-work-task-field-section": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkTaskFieldSection,
      })),
    ),
    "show-crm-work-task-upload": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkTaskUpload,
      })),
    ),
    "show-crm-work-customer-table": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkCustomerTable,
      })),
    ),
    "show-crm-work-lead-pool": lazyNode(() =>
      loadWorkLead().then((mod) => ({
        default: mod.ShowCrmWorkLeadPool,
      })),
    ),
    "show-crm-work-sidebar": lazyNode(() =>
      loadWorkShell().then((mod) => ({
        default: mod.ShowCrmWorkSidebar,
      })),
    ),
    "show-crm-work-titlebar": lazyNode(() =>
      loadWorkShell().then((mod) => ({
        default: mod.ShowCrmWorkTitlebar,
      })),
    ),
    "show-crm-work-skin": lazyNode(() =>
      loadWorkSkin().then((mod) => ({
        default: mod.ShowCrmWorkSkin,
      })),
    ),
    "show-crm-work-stats": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkStats,
      })),
    ),
    "show-crm-admin-stats": lazyNode(() =>
      loadAdminStats().then((mod) => ({
        default: mod.ShowCrmAdminStats,
      })),
    ),
    "show-crm-work-detail": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkDetail,
      })),
    ),
    "show-crm-work-record-detail": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkRecordDetail,
      })),
    ),
  },
});
