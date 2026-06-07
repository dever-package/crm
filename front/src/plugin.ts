import { defineFrontPlugin, lazyNode } from "@dever/front-plugin";

export default defineFrontPlugin({
  name: "crm",
  nodes: {
    "show-crm-flow-workspace": lazyNode(() =>
      import("./nodes/show/flow-workspace").then((mod) => ({
        default: mod.ShowCrmFlowWorkspace,
      })),
    ),
  },
});
