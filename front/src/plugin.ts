import { defineFrontPlugin, lazyNode } from "@dever/front-plugin";

const loadWorkAuth = () => import("./nodes/show/work-auth");
const loadWorkTaskForm = () => import("./nodes/show/work-task-form");
const loadWorkLead = () => import("./nodes/show/work-lead");
const loadWorkShell = () => import("./nodes/show/work-shell");
const loadWorkSchedule = () => import("./nodes/show/work-schedule");
const loadAdminStats = () => import("./nodes/show/admin-stats");
const loadCustomerTagSelector = () =>
  import("./nodes/show/customer-tag-selector");

export default defineFrontPlugin({
  name: "crm",
  nodes: {
    "form-crm-customer-tags": lazyNode(() =>
      loadCustomerTagSelector().then((mod) => ({
        default: mod.ShowCrmCustomerTagSelector,
      })),
    ),
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
    "show-crm-work-task-context": lazyNode(() =>
      loadWorkTaskForm().then((mod) => ({
        default: mod.ShowCrmWorkTaskContext,
      })),
    ),
    "show-crm-work-task-group-tabs": lazyNode(() =>
      loadWorkTaskForm().then((mod) => ({
        default: mod.ShowCrmWorkTaskGroupTabs,
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
    "show-crm-work-lead-toolbar": lazyNode(() =>
      loadWorkLead().then((mod) => ({
        default: mod.ShowCrmWorkLeadToolbar,
      })),
    ),
    "show-crm-work-lead-filter-actions": lazyNode(() =>
      loadWorkLead().then((mod) => ({
        default: mod.ShowCrmWorkLeadFilterActions,
      })),
    ),
    "show-crm-work-lead-editor-form": lazyNode(() =>
      loadWorkLead().then((mod) => ({
        default: mod.ShowCrmWorkLeadEditorForm,
      })),
    ),
    "show-crm-work-lead-detail": lazyNode(() =>
      loadWorkLead().then((mod) => ({
        default: mod.ShowCrmWorkLeadDetail,
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
    "show-crm-work-schedule": lazyNode(() =>
      loadWorkSchedule().then((mod) => ({
        default: mod.ShowCrmWorkSchedule,
      })),
    ),
    "show-crm-work-schedule-form": lazyNode(() =>
      loadWorkSchedule().then((mod) => ({
        default: mod.ShowCrmWorkScheduleForm,
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
