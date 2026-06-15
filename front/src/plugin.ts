import { defineFrontPlugin, lazyNode } from "@dever/front-plugin";

const loadWorkAuth = () => import("./nodes/show/work-auth");

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
    "show-crm-work-task-form": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkTaskForm,
      })),
    ),
    "show-crm-work-collaboration-targets": lazyNode(() =>
      loadWorkAuth().then((mod) => ({
        default: mod.ShowCrmWorkCollaborationTargets,
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
