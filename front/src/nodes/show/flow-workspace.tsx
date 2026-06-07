import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties, MouseEvent as ReactMouseEvent, ReactNode } from "react";
import {
  Background,
  BaseEdge,
  Controls,
  EdgeLabelRenderer,
  Handle,
  Position,
  ReactFlow,
  ReactFlowProvider,
  applyNodeChanges,
  getBezierPath,
  useReactFlow,
} from "@xyflow/react";
import type {
  Connection,
  Edge as ReactFlowEdge,
  EdgeProps,
  FinalConnectionState,
  Node as ReactFlowNode,
  NodeChange,
  NodeProps,
  OnConnectStartParams,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
  Check,
  GitBranch,
  ListChecks,
  Loader2,
  Plus,
  Save,
  SquarePen,
  Trash2,
  Workflow,
  X,
} from "lucide-react";
import { toast } from "sonner";
import { request } from "@/lib/request";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { AssistantContextFormFillButton } from "@/components/assistant/form-actions";
import type { NodeItemProps } from "@/page/nodes";

type Row = {
  id?: number;
  _key?: string;
  [key: string]: any;
};

type Selection =
  | { kind: "stage"; key: string }
  | { kind: "stage_edge"; index: number }
  | { kind: "node"; key: string }
  | { kind: "edge"; index: number }
  | null;

type ViewMode = "stage" | "node";

type Workspace = {
  flow: Row;
  stages: Row[];
  nodes: Row[];
  edges: Row[];
  task_templates: Row[];
  rule_scripts: Row[];
  departments: Row[];
  staff: Row[];
};

type GraphNodeData = {
  kind: "stage" | "node";
  item: Row;
  subtitle: string;
  badges: string[];
  onEdit: (selection: Exclude<Selection, null>) => void;
  onDelete: (selection: Exclude<Selection, null>) => void;
} & Record<string, unknown>;

type GraphEdgeData = {
  kind: "stage" | "node";
  index: number;
  item: Row;
  preview?: boolean;
  highlighted?: boolean;
  onEdit: (selection: Exclude<Selection, null>) => void;
  onDelete: (selection: Exclude<Selection, null>) => void;
} & Record<string, unknown>;

type GraphEdgeInput = {
  kind: "stage" | "node";
  edge: Row;
  index: number;
};

type CrmGraphNode = ReactFlowNode<GraphNodeData, "crmGraphNode">;
type CrmGraphEdge = ReactFlowEdge<GraphEdgeData, "crmGraphEdge">;
type ProximityConnection = {
  source: string;
  target: string;
};

const emptyWorkspace: Workspace = {
  flow: {},
  stages: [],
  nodes: [],
  edges: [],
  task_templates: [],
  rule_scripts: [],
  departments: [],
  staff: [],
};

const nodeTypes = {
  crmGraphNode: CrmGraphNodeCard,
};

const edgeTypes = {
  crmGraphEdge: CrmGraphEdgeLine,
};

const CARD_WIDTH = 64;
const CARD_HEIGHT = 64;
const GRAPH_ACTION_BUTTON_STYLE: CSSProperties = {
  width: 24,
  height: 24,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  padding: 0,
  lineHeight: 0,
};
const GRAPH_ACTION_ICON_STYLE: CSSProperties = {
  display: "block",
  flex: "0 0 auto",
};
const NODE_TYPES = [
  { id: "task", value: "任务" },
  { id: "branch", value: "判断" },
];
const PUBLISH_STATUS_LABEL: Record<string, string> = {
  draft: "草稿",
  published: "已发布",
  editing: "编辑草稿",
};
const CONNECTED_STAGE_MAX_DISTANCE = 220;
const CONNECTED_STAGE_FALLBACK_DISTANCE = 170;
const PROXIMITY_CONNECT_DISTANCE = 150;

export function ShowCrmFlowWorkspace({ item }: NodeItemProps) {
  return (
    <ReactFlowProvider>
      <CrmFlowWorkspaceContent item={item} />
    </ReactFlowProvider>
  );
}

function CrmFlowWorkspaceContent({ item }: NodeItemProps) {
  const meta = item.meta ?? {};
  const initialFlowTemplateID = useMemo(() => resolveFlowTemplateID(), []);
  const workspaceApi = String(meta.workspaceApi || "/crm/flow/workspace");
  const saveApi = String(meta.saveApi || "/crm/flow/save");
  const publishApi = String(meta.publishApi || "/crm/flow/publish");

  const [flowTemplateID, setFlowTemplateID] = useState(initialFlowTemplateID);
  const [workspace, setWorkspace] = useState<Workspace>(emptyWorkspace);
  const [view, setView] = useState<ViewMode>("stage");
  const [selectedStageKey, setSelectedStageKey] = useState("");
  const [selection, setSelection] = useState<Selection>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [dragStageKey, setDragStageKey] = useState("");
  const [proximityConnection, setProximityConnection] = useState<ProximityConnection | null>(null);
  const draggingNodeRef = useRef(false);
  const connectSourceRef = useRef<string | null>(null);
  const syncedViewRef = useRef<ViewMode>("stage");
  const { fitView, screenToFlowPosition } = useReactFlow<CrmGraphNode, CrmGraphEdge>();

  const sortedStages = useMemo(
    () => activeRows(workspace.stages).sort(compareSort),
    [workspace.stages],
  );
  const sortedNodes = useMemo(
    () => activeRows(workspace.nodes).sort(compareSort),
    [workspace.nodes],
  );
  const activeStage = useMemo(
    () => sortedStages.find((stage) => rowKey(stage) === selectedStageKey),
    [selectedStageKey, sortedStages],
  );
  const activeStageNodes = useMemo(
    () =>
      activeStage
        ? sortedNodes.filter((node) => nodeBelongsToStage(node, activeStage))
        : [],
    [activeStage, sortedNodes],
  );
  const activeNodeKeys = useMemo(
    () => new Set(activeStageNodes.map(nodeStableKey)),
    [activeStageNodes],
  );
  const activeEdges = useMemo(
    () =>
      workspace.edges
        .map((edge, index) => ({ edge, index }))
        .filter(({ edge }) => Number(edge.status || 1) === 1)
        .sort((left, right) => compareSort(left.edge, right.edge)),
    [workspace.edges],
  );
  const activeStageEdges = useMemo(
    () =>
      activeEdges.filter(
        ({ edge }) =>
          activeNodeKeys.has(inputText(edge.from_node_key)) &&
          activeNodeKeys.has(inputText(edge.to_node_key)),
      ),
    [activeEdges, activeNodeKeys],
  );
  const activeStageEdgeRows = useMemo(
    () => activeStageEdges.map(({ edge }) => edge),
    [activeStageEdges],
  );
  const stageEdges = useMemo(
    () => normalizeStageEdges(parseJSON(workspace.flow?.config_json).stage_edges),
    [workspace.flow?.config_json],
  );
  const stageNodeCounts = useMemo(() => {
    const counts = new Map<string, number>();
    sortedStages.forEach((stage) => {
      counts.set(
        rowKey(stage),
        sortedNodes.filter((node) => nodeBelongsToStage(node, stage)).length,
      );
    });
    return counts;
  }, [sortedNodes, sortedStages]);

  const loadWorkspace = useCallback(async () => {
    setLoading(true);
    try {
      const params = flowTemplateID ? { flow_template_id: flowTemplateID } : {};
      const result = await request(workspaceApi, "get", params);
      if (!isSuccess(result)) {
        throw new Error(errorMessage(result) || "加载流程配置失败");
      }
      const next = normalizeWorkspace(result.data);
      const nextFlowTemplateID = Number(
        result.data?.flow_template_id || next.flow?.id || flowTemplateID || 0,
      );
      if (nextFlowTemplateID) {
        setFlowTemplateID(nextFlowTemplateID);
      }
      setWorkspace(next);
      setView("stage");
      setSelectedStageKey("");
      setSelection(null);
      setEditorOpen(false);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载流程配置失败");
    } finally {
      setLoading(false);
    }
  }, [flowTemplateID, workspaceApi]);

  useEffect(() => {
    void loadWorkspace();
  }, [loadWorkspace]);

  const patchStage = (row: Row, patch: Row) => {
    setWorkspace((current) => ({
      ...current,
      stages: current.stages.map((item) =>
        sameRow(item, row) ? { ...item, ...patch } : item,
      ),
    }));
  };

  const patchNode = (row: Row, patch: Row) => {
    setWorkspace((current) => ({
      ...current,
      nodes: current.nodes.map((item) =>
        sameRow(item, row) ? { ...item, ...patch } : item,
      ),
    }));
  };

  const patchEdge = (index: number, patch: Row) => {
    setWorkspace((current) => ({
      ...current,
      edges: current.edges.map((item, currentIndex) =>
        currentIndex === index ? { ...item, ...patch } : item,
      ),
    }));
  };

  const setStageEdges = (nextEdges: Row[]) => {
    setWorkspace((current) => ({
      ...current,
      flow: {
        ...current.flow,
        config_json: jsonFlowConfig(current.flow?.config_json, {
          stage_edges: nextEdges,
        }),
      },
    }));
  };

  const createStageRow = useCallback(
    (position?: { x: number; y: number }) => ({
      _key: tempKey("stage"),
      flow_template_id: flowTemplateID || "",
      stage_key: nextCode("stage", workspace.stages),
      name: nextIndexedName("新阶段", workspace.stages, (stage) => inputText(stage.name)),
      description: "",
      default_department_id: "",
      position_json: jsonPosition(position || nextGraphPosition(workspace.stages)),
      status: 1,
      sort: nextSort(workspace.stages),
    }),
    [flowTemplateID, workspace.stages],
  );

  const createNodeRow = useCallback(
    (stage: Row, position?: { x: number; y: number }) => {
      const stageNodes = activeRows(workspace.nodes).filter((node) => nodeBelongsToStage(node, stage));
      return {
        _key: tempKey("node"),
        flow_template_id: flowTemplateID || "",
        stage_id: stage?.id || "",
        _stage_key: rowKey(stage),
        node_key: nextCode("node", workspace.nodes),
        name: nextIndexedName("新节点", stageNodes, (node) => inputText(node.name)),
        description: "",
        node_type: "task",
        task_template_id: "",
        script_id: "",
        executor_mode: "staff",
        default_department_id: "",
        default_role_id: "",
        default_staff_id: "",
        enable_deadline: false,
        deadline_minutes: 0,
        position_json: jsonPosition(position || nextGraphPosition(stageNodes)),
        config_json: "{}",
        status: 1,
        sort: nextSort(workspace.nodes),
      };
    },
    [flowTemplateID, workspace.nodes],
  );

  const addStageEdgeByRows = useCallback(
    (fromStage: Row, toStage: Row, currentStageEdges = stageEdges) => {
      const fromStageKey = stageStableKey(fromStage);
      const toStageKey = stageStableKey(toStage);
      if (!fromStageKey || !toStageKey || fromStageKey === toStageKey) {
        return null;
      }
      if (
        currentStageEdges.some(
          (edge) =>
            inputText(edge.from_stage_key) === fromStageKey &&
            inputText(edge.to_stage_key) === toStageKey,
        )
      ) {
        return null;
      }
      return {
        _key: tempKey("stage_edge"),
        from_stage_key: fromStageKey,
        to_stage_key: toStageKey,
        status: 1,
        sort: nextSort(currentStageEdges),
      };
    },
    [stageEdges],
  );

  const addStage = () => {
    const row = createStageRow();
    setWorkspace((current) => ({
      ...current,
      stages: [...current.stages, row],
    }));
    setSelectedStageKey(rowKey(row));
    setView("stage");
    setSelection({ kind: "stage", key: rowKey(row) });
  };

  const addConnectedStage = useCallback(
    (fromStageKey: string, position: { x: number; y: number }) => {
      const fromStage = workspace.stages.find((stage) => rowKey(stage) === fromStageKey);
      if (!fromStage) {
        return;
      }
      const row = createStageRow(position);
      const nextEdge = addStageEdgeByRows(fromStage, row);
      setWorkspace((current) => ({
        ...current,
        flow: {
          ...current.flow,
          config_json: jsonFlowConfig(current.flow?.config_json, {
            stage_edges: nextEdge ? [...stageEdges, nextEdge] : stageEdges,
          }),
        },
        stages: [...current.stages, row],
      }));
      setSelectedStageKey(rowKey(row));
      setView("stage");
      setSelection({ kind: "stage", key: rowKey(row) });
    },
    [addStageEdgeByRows, createStageRow, stageEdges, workspace.stages],
  );

  const addNode = () => {
    const stage = activeStage;
    if (!stage) {
      toast.info("请先选择阶段");
      return;
    }
    const row = createNodeRow(stage);
    setWorkspace((current) => ({
      ...current,
      nodes: [...current.nodes, row],
    }));
    openEditor({ kind: "node", key: rowKey(row) });
  };

  const addConnectedNode = useCallback(
    (fromNodeKey: string, position: { x: number; y: number }) => {
      const stage = activeStage;
      if (!stage) {
        return;
      }
      const fromNode = activeStageNodes.find((node) => rowKey(node) === fromNodeKey);
      if (!fromNode) {
        return;
      }
      const row = createNodeRow(stage, position);
      const nextEdge = createNodeEdgeRow(fromNode, row, workspace.edges, flowTemplateID);
      setWorkspace((current) => ({
        ...current,
        nodes: [...current.nodes, row],
        edges: nextEdge ? [...current.edges, nextEdge] : current.edges,
      }));
      setSelection({ kind: "node", key: rowKey(row) });
    },
    [activeStage, activeStageNodes, createNodeRow, flowTemplateID, workspace.edges],
  );

  const connectStages = useCallback(
    (fromGraphID: string, toGraphID: string, notifyDuplicate = true) => {
      const source = graphNodeInfo(fromGraphID);
      const target = graphNodeInfo(toGraphID);
      if (
        source.kind !== "stage" ||
        target.kind !== "stage" ||
        !source.key ||
        !target.key ||
        source.key === target.key
      ) {
        return false;
      }
      const fromStage = workspace.stages.find((stage) => rowKey(stage) === source.key);
      const toStage = workspace.stages.find((stage) => rowKey(stage) === target.key);
      if (!fromStage || !toStage) {
        return false;
      }
      const nextEdge = addStageEdgeByRows(fromStage, toStage);
      if (!nextEdge) {
        if (notifyDuplicate) {
          toast.info("连线已存在");
        }
        return false;
      }
      setStageEdges([...stageEdges, nextEdge]);
      return true;
    },
    [addStageEdgeByRows, stageEdges, workspace.stages],
  );

  const connectNodes = useCallback(
    (fromGraphID: string, toGraphID: string, notifyDuplicate = true) => {
      const source = graphNodeInfo(fromGraphID);
      const target = graphNodeInfo(toGraphID);
      if (
        source.kind !== "node" ||
        target.kind !== "node" ||
        !source.key ||
        !target.key ||
        source.key === target.key
      ) {
        return false;
      }
      const fromNode = activeStageNodes.find((node) => rowKey(node) === source.key);
      const toNode = activeStageNodes.find((node) => rowKey(node) === target.key);
      if (!fromNode || !toNode) {
        return false;
      }
      const nextEdge = createNodeEdgeRow(fromNode, toNode, workspace.edges, flowTemplateID);
      if (!nextEdge) {
        if (notifyDuplicate) {
          toast.info("连线已存在");
        }
        return false;
      }
      setWorkspace((current) => ({
        ...current,
        edges: [...current.edges, nextEdge],
      }));
      return true;
    },
    [activeStageNodes, flowTemplateID, workspace.edges],
  );

  const addEdge = (connection: Connection) => {
    if (view === "stage") {
      connectStages(connection.source || "", connection.target || "");
      return;
    }
    connectNodes(connection.source || "", connection.target || "");
  };

  const removeSelection = (target: Exclude<Selection, null>) => {
    if (target.kind === "stage") {
      const row = workspace.stages.find((item) => rowKey(item) === target.key);
      if (!row) return;
      const removedNodeKeys = new Set(
        workspace.nodes
          .filter((node) => nodeBelongsToStage(node, row))
          .map(nodeStableKey)
          .filter(Boolean),
      );
      setWorkspace((current) => ({
        ...current,
        flow: {
          ...current.flow,
          config_json: jsonFlowConfig(current.flow?.config_json, {
            stage_edges: normalizeStageEdges(parseJSON(current.flow?.config_json).stage_edges).filter(
              (edge) => !stageEdgeTouchesStage(edge, row),
            ),
          }),
        },
        stages: removeRow(current.stages, row),
        nodes: current.nodes
          .map((node) =>
            nodeBelongsToStage(node, row) && node.id ? { ...node, status: 2 } : node,
          )
          .filter((node) => !nodeBelongsToStage(node, row) || node.id),
        edges: current.edges
          .map((edge) =>
            edgeTouchesAnyNode(edge, removedNodeKeys) && edge.id ? { ...edge, status: 2 } : edge,
          )
          .filter((edge) => !edgeTouchesAnyNode(edge, removedNodeKeys) || edge.id),
      }));
      if (selectedStageKey === target.key) {
        setSelectedStageKey("");
        setView("stage");
      }
    }
    if (target.kind === "stage_edge") {
      setStageEdges(stageEdges.filter((_, index) => index !== target.index));
    }
    if (target.kind === "node") {
      const row = workspace.nodes.find((item) => rowKey(item) === target.key);
      if (!row) return;
      const removedNodeKey = nodeStableKey(row);
      setWorkspace((current) => ({
        ...current,
        nodes: removeRow(current.nodes, row),
        edges: current.edges
          .map((edge) => (edgeConnectedToNode(edge, removedNodeKey) ? { ...edge, status: 2 } : edge))
          .filter((edge) => !edgeConnectedToNode(edge, removedNodeKey) || edge.id),
      }));
    }
    if (target.kind === "edge") {
      setWorkspace((current) => ({
        ...current,
        edges: current.edges
          .map((edge, index) =>
            index === target.index && edge.id ? { ...edge, status: 2 } : edge,
          )
          .filter((edge, index) => index !== target.index || edge.id),
      }));
    }
    setSelection(null);
    setEditorOpen(false);
  };

  const moveGraphNode = (nodeID: string, position: { x: number; y: number }) => {
    const info = graphNodeInfo(nodeID);
    if (info.kind === "stage") {
      const row = workspace.stages.find((item) => rowKey(item) === info.key);
      if (row) patchStage(row, { position_json: jsonPosition(position) });
    }
    if (info.kind === "node") {
      const row = workspace.nodes.find((item) => rowKey(item) === info.key);
      if (row) patchNode(row, { position_json: jsonPosition(position) });
    }
  };

  const reorderStage = (fromKey: string, toKey: string) => {
    if (!fromKey || fromKey === toKey) {
      return;
    }
    setWorkspace((current) => ({
      ...current,
      stages: reorderRows(current.stages, fromKey, toKey),
    }));
  };

  const saveWorkspace = async (silent = false) => {
    if (saving) {
      return 0;
    }
    setSaving(true);
    try {
      const payloadWorkspace = normalizeWorkspaceGraphKeys(workspace);
      const result = await request(saveApi, "post", {
        flow_template_id: flowTemplateID || "",
        flow: payloadWorkspace.flow,
        stages: payloadWorkspace.stages,
        nodes: payloadWorkspace.nodes.map(normalizeNodeForSave),
        edges: payloadWorkspace.edges,
      });
      if (!isSuccess(result)) {
        throw new Error(errorMessage(result) || "保存流程配置失败");
      }
      const next = normalizeWorkspace(result.data);
      const nextFlowTemplateID = Number(
        result.data?.flow_template_id || next.flow?.id || flowTemplateID || 0,
      );
      if (nextFlowTemplateID) {
        setFlowTemplateID(nextFlowTemplateID);
      }
      setWorkspace(next);
      if (selectedStageKey) {
        const savedStage = findSavedStage(next.stages, payloadWorkspace.stages, selectedStageKey);
        setSelectedStageKey(savedStage ? rowKey(savedStage) : selectedStageKey);
      }
      if (!silent) {
        toast.success("流程配置已保存");
      }
      return nextFlowTemplateID;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "保存流程配置失败");
      return 0;
    } finally {
      setSaving(false);
    }
  };

  const publishWorkspace = async () => {
    if (publishing) {
      return;
    }
    setPublishing(true);
    try {
      const savedFlowTemplateID = await saveWorkspace(true);
      if (!savedFlowTemplateID) {
        return;
      }
      const result = await request(publishApi, "post", {
        flow_template_id: savedFlowTemplateID,
      });
      if (!isSuccess(result)) {
        throw new Error(errorMessage(result) || "发布流程失败");
      }
      toast.success(`流程已发布${result.data?.version ? `，版本 ${result.data.version}` : ""}`);
      await loadWorkspace();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "发布流程失败");
    } finally {
      setPublishing(false);
    }
  };

  const openEditor = (target: Exclude<Selection, null>) => {
    setSelection(target);
    setEditorOpen(true);
  };

  const openStage = (stage: Row) => {
    setSelectedStageKey(rowKey(stage));
    setSelection(null);
    setView("node");
  };

  const graph = buildGraph({
    view,
    stages: view === "stage" ? sortedStages : [],
    nodes: view === "node" ? activeStageNodes : [],
    edges:
      view === "stage"
        ? stageEdges.map((edge, index) => ({ kind: "stage" as const, edge, index }))
        : activeStageEdges.map(({ edge, index }) => ({ kind: "node" as const, edge, index })),
    stageNodeCounts,
    selection,
    proximityConnection,
    selectedStageStableKey:
      selection?.kind === "stage"
        ? stageStableKey(sortedStages.find((stage) => rowKey(stage) === selection.key) || {})
        : "",
    selectedNodeStableKey:
      selection?.kind === "node"
        ? nodeStableKey(activeStageNodes.find((node) => rowKey(node) === selection.key) || {})
        : "",
    onEdit: openEditor,
    onDelete: removeSelection,
  });
  const graphSignature = [
    `view:${view}:${selectedStageKey}`,
    ...graph.nodes.map(
      (node) =>
        `${node.id}:${node.data.item.name}:${node.data.subtitle}:${node.selected ? 1 : 0}`,
    ),
    ...graph.edges.map((edge) => `${edge.id}:${edge.source}:${edge.target}:${edge.selected ? 1 : 0}:${edge.animated ? 1 : 0}:${edge.data?.kind}:${edge.data?.highlighted ? 1 : 0}:${edge.data?.item.match_result || ""}`),
    proximityConnection ? `proximity:${proximityConnection.source}:${proximityConnection.target}` : "",
  ].join("|");
  const fitViewSignature = [
    `view:${view}:${view === "node" ? selectedStageKey : ""}`,
    ...graph.nodes.map((node) => node.id),
  ].join("|");
  const [graphNodes, setGraphNodes] = useState<CrmGraphNode[]>(graph.nodes);
  const [graphEdges, setGraphEdges] = useState<CrmGraphEdge[]>(graph.edges);

  useEffect(() => {
    if (draggingNodeRef.current) {
      return;
    }
    window.requestAnimationFrame(() => {
      void fitView({ padding: 0.24, maxZoom: 1.15, duration: 160 });
    });
  }, [fitView, fitViewSignature]);

  useEffect(() => {
    if (draggingNodeRef.current) {
      setGraphEdges(graph.edges);
      return;
    }
    if (syncedViewRef.current !== view) {
      syncedViewRef.current = view;
      setGraphNodes(graph.nodes);
      setGraphEdges(graph.edges);
      return;
    }
    setGraphNodes((current) => mergeGraphNodes(current, graph.nodes));
    setGraphEdges(graph.edges);
  }, [graphSignature, view]);

  const handleNodesChange = useCallback(
    (changes: NodeChange<CrmGraphNode>[]) => {
      setGraphNodes((current) => applyNodeChanges(changes, current));
    },
    [],
  );

  const handleNodeDragStart = useCallback(() => {
    draggingNodeRef.current = true;
    setProximityConnection(null);
  }, []);

  const handleNodeDrag = useCallback(
    (_event: ReactMouseEvent, node: CrmGraphNode) => {
      const nextConnection = resolveProximityConnection(
        node,
        graphNodes,
        view === "stage" ? stageEdges : activeStageEdgeRows,
      );
      setProximityConnection((current) =>
        sameProximityConnection(current, nextConnection) ? current : nextConnection,
      );
    },
    [activeStageEdgeRows, graphNodes, stageEdges, view],
  );

  const handleNodeDragStop = useCallback(
    (_event: ReactMouseEvent, node: CrmGraphNode) => {
      draggingNodeRef.current = false;
      const nextConnection = resolveProximityConnection(
        node,
        graphNodes,
        view === "stage" ? stageEdges : activeStageEdgeRows,
      );
      setProximityConnection(null);
      const position = canvasPointFromPosition(node.position);
      setGraphNodes((current) =>
        mergeGraphNodes(current, graph.nodes).map((item) =>
          item.id === node.id ? { ...item, position } : item,
        ),
      );
      moveGraphNode(node.id, position);
      if (nextConnection) {
        if (view === "stage") {
          connectStages(nextConnection.source, nextConnection.target, false);
        } else {
          connectNodes(nextConnection.source, nextConnection.target, false);
        }
      }
    },
    [activeStageEdgeRows, connectNodes, connectStages, graph.nodes, graphNodes, stageEdges, view],
  );

  const handleConnectStart = useCallback(
    (_event: MouseEvent | TouchEvent, params: OnConnectStartParams) => {
      connectSourceRef.current = params.nodeId || null;
    },
    [],
  );

  const handleConnectEnd = useCallback(
    (event: MouseEvent | TouchEvent, connectionState: FinalConnectionState) => {
      const fromNodeID = connectSourceRef.current;
      connectSourceRef.current = null;
      if (!fromNodeID || connectionState.toNode) {
        return;
      }
      const sourceInfo = graphNodeInfo(fromNodeID);
      if (sourceInfo.kind !== view || !sourceInfo.key) {
        return;
      }
      const clientPoint = clientPointFromConnectEvent(event);
      if (!clientPoint) {
        return;
      }
      const flowPoint = screenToFlowPosition(clientPoint);
      const sourceNode = graphNodes.find((node) => node.id === fromNodeID);
      if (!isMeaningfulConnectDrag(sourceNode?.position, flowPoint)) {
        return;
      }
      const position = connectedNodePosition(sourceNode?.position, flowPoint);
      if (view === "stage") {
        addConnectedStage(sourceInfo.key, position);
      } else {
        addConnectedNode(sourceInfo.key, position);
      }
    },
    [addConnectedNode, addConnectedStage, graphNodes, screenToFlowPosition, view],
  );

  const selectedStage = selection?.kind === "stage"
    ? workspace.stages.find((row) => rowKey(row) === selection.key)
    : undefined;
  const selectedNode = selection?.kind === "node"
    ? workspace.nodes.find((row) => rowKey(row) === selection.key)
    : undefined;
  const selectedEdge = selection?.kind === "edge"
    ? workspace.edges[selection.index]
    : undefined;
  const publishStatus = inputText(workspace.flow?.publish_status) || "draft";

  return (
    <div
      className="grid min-h-0 overflow-hidden rounded-md border bg-background"
      style={{
        gridTemplateColumns: "16rem minmax(0, 1fr)",
        height: "min(76vh, 48rem)",
        minHeight: "34rem",
      }}
    >
      <aside className="flex min-h-0 min-w-0 flex-col border-r bg-muted/20">
        <div className="border-b p-4">
          <div className="text-xs text-muted-foreground">当前流程</div>
          <div className="mt-1 truncate text-base font-semibold">
            {workspace.flow?.name || "业务流程"}
          </div>
          <div className="mt-2 flex flex-wrap gap-2">
            <span className="inline-flex rounded bg-background px-2 py-0.5 text-xs text-muted-foreground">
              {PUBLISH_STATUS_LABEL[publishStatus] || publishStatus}
            </span>
            {workspace.flow?.release_version ? (
              <span className="inline-flex rounded bg-background px-2 py-0.5 text-xs text-muted-foreground">
                版本 {workspace.flow.release_version}
              </span>
            ) : null}
          </div>
        </div>
        <div className="border-b p-2">
          <button
            type="button"
            className={classes(
              "flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm",
              view === "stage"
                ? "bg-primary text-primary-foreground"
                : "hover:bg-muted",
            )}
            onClick={() => {
              setView("stage");
              setSelectedStageKey("");
              setSelection(null);
            }}
          >
            <Workflow className="size-4 shrink-0" />
            <span className="min-w-0 flex-1 truncate">阶段视图</span>
          </button>
        </div>
        <div className="flex items-center justify-between px-3 py-2">
          <span className="text-sm font-medium">阶段列表</span>
          <Button
            type="button"
            size="icon"
            variant="ghost"
            disabled={loading || saving || publishing}
            onClick={addStage}
          >
            <Plus className="size-4" />
          </Button>
        </div>
        <div
          className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto px-2 pb-3 pr-1"
          style={{ scrollbarGutter: "stable" }}
        >
          {sortedStages.length ? (
            sortedStages.map((stage) => {
              const key = rowKey(stage);
              const active = view === "node" && selectedStageKey === key;
              const count = stageNodeCounts.get(key) || 0;
              return (
                <div
                  key={key}
                  draggable={!loading && !saving && !publishing}
                  aria-grabbed={dragStageKey === key}
                  className={classes(
                    "mb-1 flex w-full select-none items-center gap-1 rounded-md",
                    active ? "bg-primary text-primary-foreground" : "hover:bg-muted",
                    dragStageKey === key && "opacity-60",
                  )}
                  onDragStart={(event) => {
                    if (loading || saving || publishing) {
                      event.preventDefault();
                      return;
                    }
                    setDragStageKey(key);
                    event.dataTransfer.effectAllowed = "move";
                    event.dataTransfer.setData("text/plain", key);
                  }}
                  onDragOver={(event) => {
                    if (dragStageKey && dragStageKey !== key) {
                      event.preventDefault();
                      event.dataTransfer.dropEffect = "move";
                    }
                  }}
                  onDrop={(event) => {
                    event.preventDefault();
                    reorderStage(dragStageKey, key);
                    setDragStageKey("");
                  }}
                  onDragEnd={() => setDragStageKey("")}
                >
                  <button
                    type="button"
                    className="flex min-w-0 flex-1 items-center gap-2 px-3 py-2 text-left text-sm"
                    onClick={() => openStage(stage)}
                  >
                    <Workflow className="size-4 shrink-0" />
                    <span className="min-w-0 flex-1 truncate">
                      {stage.name || "未命名阶段"}
                    </span>
                    <span className="shrink-0 text-xs opacity-70">{count}</span>
                  </button>
                  <button
                    type="button"
                    className={classes(
                      "mr-2 inline-flex size-6 items-center justify-center rounded hover:bg-background/70",
                      active && "hover:bg-primary-foreground/15",
                    )}
                    onClick={(event) => {
                      event.preventDefault();
                      event.stopPropagation();
                      openEditor({ kind: "stage", key });
                    }}
                  >
                    <SquarePen className="size-3.5" />
                  </button>
                </div>
              );
            })
          ) : (
            <div className="rounded-md border border-dashed bg-background/60 px-3 py-6 text-center text-sm text-muted-foreground">
              暂无阶段
            </div>
          )}
        </div>
      </aside>

      <section
        className="grid min-h-0 min-w-0"
        style={{ gridTemplateRows: "auto minmax(0, 1fr)" }}
      >
        <div className="flex flex-wrap items-center gap-2 border-b px-4 py-3">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={loading || saving || publishing}
            onClick={view === "stage" ? addStage : addNode}
          >
            <Plus className="size-4" />
            {view === "stage" ? "新增阶段" : "新增节点"}
          </Button>
          <Button type="button" variant="outline" size="sm" disabled={loading || saving || publishing} onClick={() => void saveWorkspace()}>
            {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
            保存
          </Button>
          <Button type="button" variant="outline" size="sm" disabled={loading || saving || publishing} onClick={publishWorkspace}>
            {publishing ? <Loader2 className="size-4 animate-spin" /> : <Check className="size-4" />}
            发布
          </Button>
          <div className="ml-auto min-w-0 text-sm text-muted-foreground">
            {view === "node" && activeStage ? (
              <span className="block truncate">
                {activeStage.name || "未命名阶段"} · {activeStageNodes.length} 个节点
              </span>
            ) : (
              <span>阶段视图 · {sortedStages.length} 个阶段</span>
            )}
          </div>
          {loading ? <span className="text-sm text-muted-foreground">加载中...</span> : null}
        </div>

        <div className="min-h-0 bg-muted/10">
            {loading ? (
              <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
                <Loader2 className="mr-2 size-4 animate-spin" />
                加载中
              </div>
            ) : (
              <div className="relative h-full min-h-0 min-w-0 overflow-hidden bg-white">
                <GraphStyle />
                <ReactFlow<CrmGraphNode, CrmGraphEdge>
                  className="team-workflow-react-flow"
                  nodes={graphNodes}
                  edges={graphEdges}
                  nodeTypes={nodeTypes}
                  edgeTypes={edgeTypes}
                  fitView
                  fitViewOptions={{ padding: 0.24, maxZoom: 1.15 }}
                  minZoom={0.35}
                  maxZoom={1.8}
                  nodeDragThreshold={4}
                  nodesDraggable
                  nodesConnectable
                  nodesFocusable
                  edgesFocusable
                  elementsSelectable
                  connectOnClick={false}
                  connectionRadius={48}
                  defaultEdgeOptions={{ type: "crmGraphEdge" }}
                  deleteKeyCode={null}
                  proOptions={{ hideAttribution: true }}
                  onConnect={addEdge}
                  onConnectStart={handleConnectStart}
                  onConnectEnd={handleConnectEnd}
                  onNodeClick={(_, node) => {
                    const info = graphNodeInfo(node.id);
                    setSelection({ kind: info.kind, key: info.key } as Exclude<Selection, null>);
                  }}
                  onNodeDoubleClick={(_, node) => {
                    const info = graphNodeInfo(node.id);
                    openEditor({ kind: info.kind, key: info.key } as Exclude<Selection, null>);
                  }}
                  onEdgeClick={(_, edge) => {
                    setSelection(edgeSelectionFromData(edge.data));
                  }}
                  onEdgeDoubleClick={(_, edge) => {
                    const selection = edgeSelectionFromData(edge.data);
                    if (selection.kind === "edge") {
                      openEditor(selection);
                    }
                  }}
                  onNodesChange={handleNodesChange}
                  onNodeDragStart={handleNodeDragStart}
                  onNodeDrag={handleNodeDrag}
                  onNodeDragStop={handleNodeDragStop}
                  onPaneClick={() => setSelection(null)}
                >
                  <Background gap={24} size={1} />
                  <Controls showInteractive={false} position="top-right" />
                </ReactFlow>
                {!graphNodes.length ? (
                  <div className="pointer-events-none absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
                    {view === "stage" ? "暂无阶段，点击新增阶段开始配置" : "当前阶段暂无节点，点击新增节点添加"}
                  </div>
                ) : null}
              </div>
            )}
        </div>
      </section>

      <EditorDialog
        open={editorOpen}
        onOpenChange={setEditorOpen}
        selection={selection}
        stage={selectedStage}
        node={selectedNode}
        edge={selectedEdge}
        taskTemplates={workspace.task_templates}
        ruleScripts={workspace.rule_scripts}
        departments={workspace.departments}
        staff={workspace.staff}
        onPatchStage={(patch) => selectedStage && patchStage(selectedStage, patch)}
        onPatchNode={(patch) => selectedNode && patchNode(selectedNode, patch)}
        onPatchEdge={(patch) => selection?.kind === "edge" && patchEdge(selection.index, patch)}
      />
    </div>
  );
}

function GraphStyle() {
  return (
    <style>{`
      .team-workflow-react-flow .react-flow__node {
        background: transparent;
        border: 0;
        box-shadow: none;
        opacity: 1;
        overflow: visible;
      }
      .team-workflow-react-flow .react-flow__node.dragging,
      .team-workflow-react-flow .react-flow__node.selected {
        z-index: 1000 !important;
        opacity: 1 !important;
      }
      .team-workflow-react-flow .react-flow__node.dragging .team-graph-node-circle {
        box-shadow: 0 14px 34px rgb(15 23 42 / 0.18);
      }
      .team-workflow-react-flow .react-flow__node:focus,
      .team-workflow-react-flow .react-flow__node:focus-visible {
        outline: none;
      }
      .team-workflow-react-flow .react-flow__edge-path {
        stroke-linecap: round;
        stroke-linejoin: round;
        transition: stroke 0.25s ease, stroke-width 0.25s ease, opacity 0.25s ease, stroke-dasharray 0.25s ease;
      }
      .team-workflow-react-flow .react-flow__edge.animated .react-flow__edge-path {
        animation-duration: 0.9s;
      }
      .team-graph-node .react-flow__handle {
        width: 9px;
        height: 9px;
        border: 1.5px solid rgb(15 23 42 / 0.34);
        background: hsl(var(--background));
        opacity: 0.45;
        transition: opacity 150ms ease, border-color 150ms ease, box-shadow 150ms ease, transform 150ms ease;
      }
      .team-graph-node:hover .react-flow__handle,
      .team-graph-node[data-selected="true"] .react-flow__handle {
        opacity: 0.9;
        border-color: rgb(99 102 241 / 0.64);
        box-shadow: 0 0 0 3px rgb(99 102 241 / 0.12);
      }
      .team-graph-node {
        position: relative;
        display: flex;
        width: ${CARD_WIDTH}px;
        height: ${CARD_HEIGHT}px;
        align-items: center;
        justify-content: center;
        user-select: none;
      }
      .team-graph-node-circle {
        position: relative;
        display: flex;
        width: ${CARD_WIDTH}px;
        height: ${CARD_HEIGHT}px;
        align-items: center;
        justify-content: center;
        border-radius: 9999px;
        border: 2px solid hsl(var(--border));
        background: hsl(var(--background));
        color: hsl(var(--foreground));
        box-shadow: 0 4px 12px rgb(15 23 42 / 0.12);
        transition: border-color 180ms ease, box-shadow 180ms ease, transform 180ms ease;
      }
      .team-graph-node:hover .team-graph-node-circle {
        box-shadow: 0 8px 20px rgb(15 23 42 / 0.15);
      }
      .team-graph-node-label {
        position: absolute;
        top: ${CARD_HEIGHT + 8}px;
        left: 50%;
        width: 150px;
        transform: translateX(-50%);
        pointer-events: auto;
        user-select: none;
        text-align: center;
      }
      .team-graph-node-title {
        display: block;
        pointer-events: none;
        max-width: 100%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: hsl(var(--foreground));
        font-size: 11px;
        font-weight: 700;
        line-height: 1.1;
      }
      .team-graph-node-subtitle {
        display: block;
        pointer-events: none;
        margin-top: 2px;
        max-width: 100%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: hsl(var(--muted-foreground));
        font-size: 9px;
        line-height: 1;
        opacity: 0.7;
      }
      .team-graph-actions {
        position: relative;
        z-index: 30;
        display: flex;
        height: 24px;
        align-items: center;
        justify-content: center;
        gap: 6px;
        margin-top: 6px;
        opacity: 0;
        transform: translateY(-3px);
        pointer-events: none;
        transition: opacity 150ms ease, transform 150ms ease;
      }
      .team-graph-node:hover .team-graph-actions,
      .team-graph-node[data-selected="true"] .team-graph-actions {
        opacity: 1;
        transform: translateY(0);
        pointer-events: auto;
      }
      .team-graph-action-button {
        pointer-events: auto;
        border: 1px solid hsl(var(--border));
        border-radius: 9999px;
        background: hsl(var(--background));
        color: hsl(var(--muted-foreground));
        box-shadow: 0 3px 10px rgb(15 23 42 / 0.10);
        transition: border-color 150ms ease, color 150ms ease, background 150ms ease, transform 150ms ease;
      }
      .team-graph-action-button:hover {
        border-color: rgb(99 102 241 / 0.55);
        color: hsl(var(--foreground));
        transform: translateY(-1px);
      }
      .team-graph-action-button-danger:hover {
        border-color: hsl(var(--destructive) / 0.45);
        color: hsl(var(--destructive));
      }
      .team-workflow-react-flow .react-flow__controls {
        border-color: hsl(var(--border));
        box-shadow: 0 8px 24px rgb(15 23 42 / 0.08);
      }
      .team-workflow-react-flow .react-flow__controls-button {
        border-color: hsl(var(--border));
        background: hsl(var(--background));
        color: hsl(var(--foreground));
      }
    `}</style>
  );
}

function CrmGraphNodeCard({ data, selected }: NodeProps<CrmGraphNode>) {
  const selection = { kind: data.kind, key: rowKey(data.item) } as Exclude<Selection, null>;
  const subtitle = data.badges.filter(Boolean).join(" · ") || data.subtitle;
  const isStage = data.kind === "stage";

  return (
    <div
      data-selected={selected ? "true" : undefined}
      className={classes("team-graph-node", isStage && "team-graph-node-stage")}
      style={{ cursor: "move" }}
    >
      <Handle type="target" position={Position.Left} />
      <Handle type="source" position={Position.Right} />
      <div className="team-graph-node-circle" style={nodeCircleStyle(selected, isStage)}>
        {isStage ? <Workflow size={20} color="#6366f1" /> : <ListChecks size={20} color="#0f766e" />}
      </div>
      <div className="team-graph-node-label">
        <span className="team-graph-node-title">{data.item.name || "未命名"}</span>
        <span className="team-graph-node-subtitle">{subtitle}</span>
        <div className="team-graph-actions">
          <button
            type="button"
            className="nodrag nopan team-graph-action-button"
            style={GRAPH_ACTION_BUTTON_STYLE}
            title="编辑"
            onClick={(event) => {
              event.stopPropagation();
              data.onEdit(selection);
            }}
            onMouseDown={(event) => event.stopPropagation()}
          >
            <SquarePen size={13} style={GRAPH_ACTION_ICON_STYLE} />
          </button>
          <button
            type="button"
            className="nodrag nopan team-graph-action-button team-graph-action-button-danger"
            style={{ ...GRAPH_ACTION_BUTTON_STYLE, color: "hsl(var(--destructive))" }}
            title="删除"
            onClick={(event) => {
              event.stopPropagation();
              data.onDelete(selection);
            }}
            onMouseDown={(event) => event.stopPropagation()}
          >
            <Trash2 size={13} style={GRAPH_ACTION_ICON_STYLE} />
          </button>
        </div>
      </div>
    </div>
  );
}

function CrmGraphEdgeLine(props: EdgeProps<CrmGraphEdge>) {
  const isStageEdge = props.data?.kind === "stage";
  const preview = Boolean(props.data?.preview);
  const highlighted = Boolean(props.selected || props.data?.highlighted || props.animated);
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX: props.sourceX,
    sourceY: props.sourceY,
    sourcePosition: props.sourcePosition,
    targetX: props.targetX,
    targetY: props.targetY,
    targetPosition: props.targetPosition,
  });
  const selection = edgeSelectionFromData(props.data);
  return (
    <>
      <BaseEdge
        path={edgePath}
        markerEnd={props.markerEnd}
        interactionWidth={32}
        style={{
          ...props.style,
          strokeWidth: highlighted || preview ? 2.4 : 1.6,
          stroke: highlighted || preview ? "#6366f1" : isStageEdge ? "#d4d4d8" : "#94a3b8",
          strokeDasharray: highlighted || !isStageEdge ? "8 7" : "7 9",
          opacity: highlighted || preview ? 1 : isStageEdge ? 0.72 : 0.9,
        }}
      />
      {!preview ? (
        <EdgeLabelRenderer>
          <div
            className={classes(
              "nodrag nopan flex items-center gap-1 rounded-full border bg-background px-1.5 py-1 text-[10px] text-muted-foreground shadow-sm",
              !props.selected && isStageEdge && "opacity-0",
            )}
            style={{
              position: "absolute",
              transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
              pointerEvents: "all",
            }}
          >
            {!isStageEdge ? (
              <button
                type="button"
                className="inline-flex size-5 items-center justify-center rounded-full hover:bg-muted"
                title="编辑连线"
                onClick={(event) => {
                  event.stopPropagation();
                  if (selection.kind === "edge") {
                    props.data?.onEdit(selection);
                  }
                }}
              >
                <GitBranch className="size-3" />
              </button>
            ) : null}
            {!isStageEdge ? <span>{props.data?.item?.match_result || "流转"}</span> : null}
            <button
              type="button"
              className="inline-flex size-5 items-center justify-center rounded-full text-destructive hover:bg-muted"
              title="删除连线"
              onClick={(event) => {
                event.stopPropagation();
                props.data?.onDelete(selection);
              }}
            >
              <X className="size-3" />
            </button>
          </div>
        </EdgeLabelRenderer>
      ) : null}
    </>
  );
}

function nodeCircleStyle(selected?: boolean, isStage = false): CSSProperties {
  if (!selected) return {};
  return {
    borderColor: isStage ? "#6366f1" : "#0f766e",
    boxShadow: isStage
      ? "0 0 15px rgb(99 102 241 / 0.35), 0 0 0 4px rgb(99 102 241 / 0.12)"
      : "0 0 15px rgb(15 118 110 / 0.28), 0 0 0 4px rgb(15 118 110 / 0.10)",
  };
}

function EditorDialog({
  open,
  onOpenChange,
  selection,
  stage,
  node,
  edge,
  taskTemplates,
  ruleScripts,
  departments,
  staff,
  onPatchStage,
  onPatchNode,
  onPatchEdge,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selection: Selection;
  stage?: Row;
  node?: Row;
  edge?: Row;
  taskTemplates: Row[];
  ruleScripts: Row[];
  departments: Row[];
  staff: Row[];
  onPatchStage: (patch: Row) => void;
  onPatchNode: (patch: Row) => void;
  onPatchEdge: (patch: Row) => void;
}) {
  const title = selection?.kind === "node" ? "编辑节点" : selection?.kind === "edge" ? "编辑连线" : "编辑阶段";
  const stageAssistantContext =
    selection?.kind === "stage" && stage ? buildStageAssistantContext(stage) : null;
  const headerAction =
    selection?.kind === "stage" && stage && stageAssistantContext ? (
      <AssistantContextFormFillButton
        context={stageAssistantContext}
        className="mt-[-0.125rem]"
        variant="outline"
        size="sm"
        onApplyValues={(values) => applyStageAssistantValues(values, onPatchStage)}
      />
    ) : null;
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        showCloseButton={false}
        className={classes(
          "flex flex-col gap-0 overflow-visible p-0",
          selection?.kind === "stage" ? "sm:max-w-2xl" : "sm:max-w-3xl",
        )}
        style={{ maxHeight: "min(82vh, 48rem)" }}
      >
        <DialogHeader className="shrink-0 px-6 py-4 text-start">
          <div className="flex items-start justify-between gap-4">
            <DialogTitle className="min-w-0 pt-1">{title}</DialogTitle>
            <div className="flex shrink-0 items-start gap-2">
              {headerAction}
              <DialogClose asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="-mr-3 -mt-2 size-8 shrink-0 self-start"
                >
                  <span className="sr-only">关闭</span>
                  <X className="size-4" />
                </Button>
              </DialogClose>
            </div>
          </div>
        </DialogHeader>
        <div
          className={classes(
            "min-h-0 overflow-y-auto px-6 pt-2",
            selection?.kind === "stage" ? "pb-12" : "pb-6",
          )}
        >
          {selection?.kind === "stage" && stage ? (
            <div className="space-y-1">
              <Field label="名称">
                <Input value={stage.name || ""} onChange={(event) => onPatchStage({ name: event.target.value })} />
              </Field>
            </div>
          ) : null}
          {selection?.kind === "node" && node ? (
            <div className="space-y-4">
              <Field label="节点名称">
                <Input value={node.name || ""} onChange={(event) => onPatchNode({ name: event.target.value })} />
              </Field>
              <Field label="节点类型">
                <OptionRadioGroup
                  value={normalizedNodeType(node.node_type)}
                  options={NODE_TYPES}
                  onValueChange={(value) =>
                    onPatchNode({
                      node_type: value,
                      task_template_id: value === "task" ? node.task_template_id || "" : "",
                      name:
                        value === "task"
                          ? node.name || taskNodeName(taskTemplateName(taskTemplates, node.task_template_id))
                          : node.name || "判断",
                      script_id: "",
                    })
                  }
                />
              </Field>
              {normalizedNodeType(node.node_type) === "task" ? (
                <Field label="任务模板">
                  <NativeSelect
                    value={stringValue(node.task_template_id)}
                    options={withEmptyOption(taskTemplates.map((task) => ({ id: task.id, value: task.name || `任务 ${task.id}` })), "不选择")}
                    onChange={(value) =>
                      onPatchNode({
                        task_template_id: value,
                        name: taskNodeName(taskTemplateName(taskTemplates, value)),
                      })
                    }
                  />
                </Field>
              ) : null}
              <DepartmentStaffPicker
                departments={departments}
                staff={staff}
                departmentID={stringValue(node.default_department_id)}
                staffID={stringValue(node.default_staff_id)}
                onChange={(patch) =>
                  onPatchNode({
                    ...patch,
                    executor_mode: "staff",
                  })
                }
              />
            </div>
          ) : null}
          {selection?.kind === "edge" && edge ? (
            <div className="space-y-4">
              <Field label="匹配结果">
                <Input value={edge.match_result || ""} onChange={(event) => onPatchEdge({ match_result: event.target.value })} />
              </Field>
              <TwoColumns>
                <Field label="匹配脚本">
                  <NativeSelect
                    value={stringValue(edge.match_script_id)}
                    options={withEmptyOption(ruleScripts.map((script) => ({ id: script.id, value: script.name || `脚本 ${script.id}` })), "不选择")}
                    onChange={(value) => onPatchEdge({ match_script_id: value })}
                  />
                </Field>
                <Field label="目标资源状态">
                  <Input value={edge.target_resource_status || ""} onChange={(event) => onPatchEdge({ target_resource_status: event.target.value })} />
                </Field>
              </TwoColumns>
            </div>
          ) : null}
          {!stage && !node && !edge ? <div className="text-sm text-muted-foreground">未找到配置项</div> : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="mb-4 space-y-2 text-sm">
      <div className="font-medium">{label}</div>
      {children}
    </div>
  );
}

function OptionRadioGroup({
  options,
  value,
  onValueChange,
}: {
  options: Array<{ id: string; value: string }>;
  value: string;
  onValueChange: (value: string) => void;
}) {
  return (
    <RadioGroup
      value={value}
      onValueChange={onValueChange}
      className="grid gap-2 sm:grid-cols-3"
    >
      {options.map((option) => (
        <label
          key={option.id}
          className={classes(
            "flex cursor-pointer items-center gap-2 rounded-md border px-3 py-2 text-sm transition-colors",
            value === option.id && "border-primary bg-primary/5 text-primary",
          )}
        >
          <RadioGroupItem value={option.id} />
          <span>{option.value}</span>
        </label>
      ))}
    </RadioGroup>
  );
}

function DepartmentStaffPicker({
  departments,
  staff,
  departmentID,
  staffID,
  onChange,
}: {
  departments: Row[];
  staff: Row[];
  departmentID: string;
  staffID: string;
  onChange: (patch: Row) => void;
}) {
  const selectedDepartmentID = departmentID || departmentIDFromStaff(staff, staffID);
  const availableStaff = selectedDepartmentID
    ? staff.filter((item) => stringValue(item.department_id) === selectedDepartmentID)
    : [];
  const selectedStaffValid = availableStaff.some((item) => stringValue(item.id) === staffID);

  return (
    <Field label="人员">
      <TwoColumns>
        <NativeSelect
          value={selectedDepartmentID}
          options={withEmptyOption(
            departments.map((department) => ({
              id: department.id,
              value: department.name || `部门 ${department.id}`,
            })),
            "先选择部门",
          )}
          onChange={(value) =>
            onChange({
              default_department_id: value,
              default_staff_id: "",
            })
          }
        />
        <NativeSelect
          value={selectedStaffValid ? staffID : ""}
          options={withEmptyOption(
            availableStaff.map((item) => ({
              id: item.id,
              value: item.name || `人员 ${item.id}`,
            })),
            selectedDepartmentID ? "选择人员" : "请先选择部门",
          )}
          onChange={(value) => {
            const selectedStaff = staff.find((item) => stringValue(item.id) === value);
            onChange({
              default_department_id: selectedStaff
                ? stringValue(selectedStaff.department_id)
                : selectedDepartmentID,
              default_staff_id: value,
            });
          }}
        />
      </TwoColumns>
    </Field>
  );
}

function TwoColumns({ children }: { children: ReactNode }) {
  return <div className="grid gap-4 sm:grid-cols-2">{children}</div>;
}

function buildStageAssistantContext(stage: Row) {
  return {
    scope: "modal",
    route: "crm/flow/stage",
    page: {
      name: "编辑阶段",
      title: stage.name || stage.stage_key || rowKey(stage),
    },
    form: {
      fields: [
        {
          path: "form.name",
          name: "名称",
          type: "form-input",
        },
      ],
      values: {
        ...(stage.name ? { "form.name": stage.name } : {}),
      },
    },
  };
}

function applyStageAssistantValues(values: Record<string, unknown>, onPatchStage: (patch: Row) => void) {
  const patch: Row = {};
  const name = readAssistantTextValue(values, "form.name");
  if (name !== undefined) {
    patch.name = name;
  }
  if (Object.keys(patch).length) {
    onPatchStage(patch);
  }
}

function readAssistantTextValue(values: Record<string, unknown>, path: string) {
  const shortPath = path.replace(/^form\./, "");
  const value = values[path] ?? values[shortPath];
  if (value === undefined || value === null) {
    return undefined;
  }
  return typeof value === "string" ? value : JSON.stringify(value);
}

function NativeSelect({
  value,
  options,
  onChange,
}: {
  value: string;
  options: Array<{ id: any; value: string }>;
  onChange: (value: string) => void;
}) {
  return (
    <select
      className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
      value={value}
      onChange={(event) => onChange(event.target.value)}
    >
      {options.map((option) => (
        <option key={String(option.id)} value={String(option.id)}>
          {option.value}
        </option>
      ))}
    </select>
  );
}

function buildGraph({
  view,
  stages,
  nodes,
  edges,
  stageNodeCounts,
  selection,
  proximityConnection,
  selectedStageStableKey,
  selectedNodeStableKey,
  onEdit,
  onDelete,
}: {
  view: ViewMode;
  stages: Row[];
  nodes: Row[];
  edges: GraphEdgeInput[];
  stageNodeCounts: Map<string, number>;
  selection: Selection;
  proximityConnection: ProximityConnection | null;
  selectedStageStableKey: string;
  selectedNodeStableKey: string;
  onEdit: (selection: Exclude<Selection, null>) => void;
  onDelete: (selection: Exclude<Selection, null>) => void;
}) {
  const graphNodes: CrmGraphNode[] = [
    ...stages.map((stage, index) => ({
      id: graphNodeID("stage", rowKey(stage)),
      type: "crmGraphNode" as const,
      position: readPosition(stage, index),
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
      selected: selection?.kind === "stage" && selection.key === rowKey(stage),
      connectable: view === "stage",
      style: { width: CARD_WIDTH, height: CARD_HEIGHT },
      data: {
        kind: "stage" as const,
        item: stage,
        subtitle: `${stageNodeCounts.get(rowKey(stage)) || 0} 个节点`,
        badges: [],
        onEdit,
        onDelete,
      },
    })),
    ...nodes.map((node, index) => ({
      id: graphNodeID("node", rowKey(node)),
      type: "crmGraphNode" as const,
      position: readPosition(node, index + stages.length),
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
      selected: selection?.kind === "node" && selection.key === rowKey(node),
      connectable: view === "node",
      style: { width: CARD_WIDTH, height: CARD_HEIGHT },
      data: {
        kind: "node" as const,
        item: node,
        subtitle: nodeTypeLabel(node.node_type),
        badges: [],
        onEdit,
        onDelete,
      },
    })),
  ];

  const nodeKeyToGraphID = new Map(nodes.map((node) => [nodeStableKey(node), graphNodeID("node", rowKey(node))]));
  const stageKeyToGraphID = new Map(stages.map((stage) => [stageStableKey(stage), graphNodeID("stage", rowKey(stage))]));
  const graphEdges: CrmGraphEdge[] = edges
    .map(({ kind, edge, index }) => {
      const source =
        kind === "stage"
          ? stageKeyToGraphID.get(inputText(edge.from_stage_key))
          : nodeKeyToGraphID.get(inputText(edge.from_node_key));
      const target =
        kind === "stage"
          ? stageKeyToGraphID.get(inputText(edge.to_stage_key))
          : nodeKeyToGraphID.get(inputText(edge.to_node_key));
      if (!source || !target) {
        return null;
      }
      return {
        id: kind === "stage" ? `stage-edge:${edge._key || index}` : edge.id ? `edge:${edge.id}` : `edge:${edge._key || index}`,
        type: "crmGraphEdge" as const,
        source,
        target,
        selected:
          kind === "stage"
            ? selection?.kind === "stage_edge" && selection.index === index
            : selection?.kind === "edge" && selection.index === index,
        animated: graphEdgeHighlighted(
          kind,
          edge,
          index,
          selection,
          selectedStageStableKey,
          selectedNodeStableKey,
        ),
        style: graphEdgeStyle(
          kind,
          edge,
          index,
          selection,
          selectedStageStableKey,
          selectedNodeStableKey,
        ),
        data: {
          kind,
          index,
          item: edge,
          highlighted: graphEdgeHighlighted(
            kind,
            edge,
            index,
            selection,
            selectedStageStableKey,
            selectedNodeStableKey,
          ),
          onEdit,
          onDelete,
        },
      };
	    })
	    .filter(Boolean) as CrmGraphEdge[];

  if (proximityConnection) {
    const kind = view;
    graphEdges.push({
      id: proximityConnectionID(proximityConnection),
      type: "crmGraphEdge" as const,
      source: proximityConnection.source,
      target: proximityConnection.target,
      selected: false,
      animated: false,
      selectable: false,
      reconnectable: false,
      zIndex: 0,
      style: {
        stroke: "#6366f1",
        strokeWidth: 1.8,
        strokeDasharray: "5 7",
        strokeLinecap: "round",
        strokeLinejoin: "round",
        opacity: 0.8,
      },
      data: {
        kind,
        index: -1,
        item:
          kind === "stage"
            ? {
                _key: proximityConnectionID(proximityConnection),
                from_stage_key: proximityConnection.source,
                to_stage_key: proximityConnection.target,
              }
            : {
                _key: proximityConnectionID(proximityConnection),
                from_node_key: proximityConnection.source,
                to_node_key: proximityConnection.target,
              },
        preview: true,
        onEdit,
        onDelete,
      },
    });
  }

  return { nodes: graphNodes, edges: graphEdges };
}

function mergeGraphNodes(currentNodes: CrmGraphNode[], nextNodes: CrmGraphNode[]) {
  if (!currentNodes.length) {
    return nextNodes;
  }
  const currentByID = new Map(currentNodes.map((node) => [node.id, node]));
  return nextNodes.map((node) => {
    const current = currentByID.get(node.id);
    return current ? { ...current, ...node, position: current.position } : node;
  });
}

function normalizeWorkspace(data: any): Workspace {
  return {
    flow: data?.flow ?? {},
    stages: rows(data?.stages),
    nodes: rows(data?.nodes),
    edges: rows(data?.edges),
    task_templates: rows(data?.task_templates),
    rule_scripts: rows(data?.rule_scripts),
    departments: rows(data?.departments),
    staff: rows(data?.staff),
  };
}

function normalizeNodeForSave(node: Row) {
  const nodeType = normalizedNodeType(node.node_type);
  return {
    ...node,
    node_type: nodeType,
    task_template_id: nodeType === "task" ? node.task_template_id || "" : "",
    script_id: "",
  };
}

function normalizeWorkspaceGraphKeys(workspace: Workspace): Workspace {
  const stageUsedKeys = graphKeySet("stage", workspace.stages);
  const nodeUsedKeys = graphKeySet("node", workspace.nodes);
  const stageKeyMap = new Map<string, string>();
  const nextStages = workspace.stages.map((stage) => {
    if (stage.id || !shouldRegenerateGraphKey(stage.stage_key, "stage")) {
      return stage;
    }
    const nextStageKey = nextUniqueGraphKey("stage", stageUsedKeys);
    stageKeyMap.set(stageStableKey(stage), nextStageKey);
    stageKeyMap.set(rowKey(stage), nextStageKey);
    return { ...stage, stage_key: nextStageKey };
  });

  const nodeKeyMap = new Map<string, string>();
  const nextNodes = workspace.nodes.map((node) => {
    let nextNode = { ...node };
    const mappedStageKey = stageKeyMap.get(inputText(node._stage_key));
    if (mappedStageKey) {
      nextNode._stage_key = mappedStageKey;
    }
    if (!node.id && shouldRegenerateGraphKey(node.node_key, "node")) {
      const nextNodeKey = nextUniqueGraphKey("node", nodeUsedKeys);
      nodeKeyMap.set(nodeStableKey(node), nextNodeKey);
      nodeKeyMap.set(rowKey(node), nextNodeKey);
      nextNode.node_key = nextNodeKey;
    }
    return nextNode;
  });

  const nextEdges = nodeKeyMap.size
    ? workspace.edges.map((edge) => ({
        ...edge,
        from_node_key: nodeKeyMap.get(inputText(edge.from_node_key)) || edge.from_node_key,
        to_node_key: nodeKeyMap.get(inputText(edge.to_node_key)) || edge.to_node_key,
      }))
    : workspace.edges;

  if (!stageKeyMap.size) {
    return {
      ...workspace,
      stages: nextStages,
      nodes: nextNodes,
      edges: nextEdges,
    };
  }

  const flowConfig = parseJSON(workspace.flow?.config_json);
  const nextStageEdges = normalizeStageEdges(flowConfig.stage_edges).map((edge) => ({
    ...edge,
    from_stage_key: stageKeyMap.get(inputText(edge.from_stage_key)) || edge.from_stage_key,
    to_stage_key: stageKeyMap.get(inputText(edge.to_stage_key)) || edge.to_stage_key,
  }));

  return {
    ...workspace,
    flow: {
      ...workspace.flow,
      config_json: jsonFlowConfig(workspace.flow?.config_json, {
        stage_edges: nextStageEdges,
      }),
    },
    stages: nextStages,
    nodes: nextNodes,
    edges: nextEdges,
  };
}

function rows(value: any): Row[] {
  return Array.isArray(value) ? value.map((row) => ({ ...row })) : [];
}

function activeRows(rows: Row[]) {
  return rows.filter((row) => Number(row.status || 1) === 1);
}

function removeRow(rows: Row[], row: Row) {
  return rows
    .map((item) => (sameRow(item, row) && item.id ? { ...item, status: 2 } : item))
    .filter((item) => !sameRow(item, row) || item.id);
}

function reorderRows(rows: Row[], fromKey: string, toKey: string) {
  const orderedKeys = activeRows(rows).sort(compareSort).map(rowKey);
  const fromIndex = orderedKeys.indexOf(fromKey);
  const toIndex = orderedKeys.indexOf(toKey);
  if (fromIndex < 0 || toIndex < 0) {
    return rows;
  }
  const nextKeys = [...orderedKeys];
  const [movedKey] = nextKeys.splice(fromIndex, 1);
  nextKeys.splice(toIndex, 0, movedKey);
  const sortByKey = new Map(nextKeys.map((key, index) => [key, (index + 1) * 10]));
  return rows.map((row) => {
    const nextSort = sortByKey.get(rowKey(row));
    return nextSort ? { ...row, sort: nextSort } : row;
  });
}

function graphNodeID(kind: "stage" | "node", key: string) {
  return `${kind}:${key}`;
}

function graphNodeInfo(id: string) {
  const [kind, ...rest] = id.split(":");
  return { kind: kind as "stage" | "node", key: rest.join(":") };
}

function rowKey(row: Row) {
  return row.id ? String(row.id) : String(row._key || "");
}

function sameRow(left: Row, right: Row) {
  return rowKey(left) === rowKey(right);
}

function compareSort(left: Row, right: Row) {
  const leftSort = Number(left.sort || 0);
  const rightSort = Number(right.sort || 0);
  if (leftSort !== rightSort) {
    return leftSort - rightSort;
  }
  return Number(left.id || 0) - Number(right.id || 0);
}

function readPosition(row: Row, index: number) {
  const parsed = parseJSON(row.position_json);
  if (Number.isFinite(Number(parsed.x)) && Number.isFinite(Number(parsed.y))) {
    return { x: Number(parsed.x), y: Number(parsed.y) };
  }
  return defaultGraphPosition(index);
}

function defaultGraphPosition(index: number) {
  return {
    x: 90 + (index % 4) * 180,
    y: 90 + Math.floor(index / 4) * 140,
  };
}

function readStoredPosition(row: Row) {
  const parsed = parseJSON(row.position_json);
  if (Number.isFinite(Number(parsed.x)) && Number.isFinite(Number(parsed.y))) {
    return { x: Number(parsed.x), y: Number(parsed.y) };
  }
  return null;
}

function nextGraphPosition(rows: Row[]) {
  const activeSortedRows = activeRows(rows).sort(compareSort);
  const lastPosition = activeSortedRows.length
    ? readStoredPosition(activeSortedRows[activeSortedRows.length - 1])
    : null;
  if (!lastPosition) {
    return defaultGraphPosition(activeSortedRows.length);
  }
  return {
    x: lastPosition.x + 160,
    y: lastPosition.y,
  };
}

function jsonPosition(position: { x: number; y: number }) {
  return JSON.stringify({
    x: Math.round(Number(position.x || 0)),
    y: Math.round(Number(position.y || 0)),
  });
}

function jsonFlowConfig(value: any, patch: Row) {
  return JSON.stringify({
    ...parseJSON(value),
    ...patch,
  });
}

function parseJSON(value: any) {
  if (!value || typeof value !== "string") {
    return {};
  }
  try {
    return JSON.parse(value);
  } catch {
    return {};
  }
}

function nextSort(rows: Row[]) {
  return rows.reduce((max, row) => Math.max(max, Number(row.sort || 0)), 0) + 10;
}

function nextCode(prefix: string, rows: Row[]) {
  return nextUniqueGraphKey(prefix, graphKeySet(prefix, rows));
}

function nextUniqueGraphKey(prefix: string, used: Set<string>) {
  for (let index = 0; index < 100; index += 1) {
    const code = tempKey(prefix);
    if (!used.has(code)) {
      used.add(code);
      return code;
    }
  }
  const code = `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1000000)}`;
  used.add(code);
  return code;
}

function graphKeySet(prefix: string, rows: Row[]) {
  return new Set(
    rows
      .flatMap((row) => [inputText(row[`${prefix}_key`]), inputText(row._key)])
      .filter(Boolean),
  );
}

function shouldRegenerateGraphKey(value: any, prefix: string) {
  return new RegExp(`^${escapeRegExp(prefix)}_\\d+$`).test(inputText(value));
}

function nextIndexedName<T>(prefix: string, rows: T[], getName: (row: T) => string) {
  const usedIndexes = new Set<number>();
  const pattern = new RegExp(`^${escapeRegExp(prefix)}(\\d+)$`);
  rows.forEach((row) => {
    const match = inputText(getName(row)).match(pattern);
    if (match) {
      usedIndexes.add(Number(match[1]));
    }
  });
  let index = 1;
  while (usedIndexes.has(index)) {
    index += 1;
  }
  return `${prefix}${index}`;
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function tempKey(prefix: string) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 10000)}`;
}

function classes(...values: Array<string | false | null | undefined>) {
  return values.filter(Boolean).join(" ");
}

function resolveFlowTemplateID() {
  if (typeof window === "undefined") return 0;
  const params = new URLSearchParams(window.location.search);
  return Number(params.get("flow_template_id") || params.get("id") || 0);
}

function isSuccess(result: any) {
  if (result?.code != null) {
    return Number(result.code) === 0;
  }
  return Number(result?.status || 0) === 1;
}

function errorMessage(result: any) {
  return result?.message || result?.msg || "";
}

function inputText(value: any) {
  return String(value ?? "").trim();
}

function inputNumber(value: any) {
  const numberValue = Number(value || 0);
  return Number.isFinite(numberValue) ? numberValue : 0;
}

function stringValue(value: any) {
  const text = String(value ?? "");
  return text === "0" ? "" : text;
}

function withEmptyOption(options: Array<{ id: any; value: string }>, label: string) {
  return [{ id: "", value: label }, ...options];
}

function nodeTypeLabel(value: any) {
  const option = NODE_TYPES.find((item) => item.id === normalizedNodeType(value));
  return option?.value || "节点";
}

function taskTemplateName(taskTemplates: Row[], taskTemplateID: any) {
  const task = taskTemplates.find((item) => String(item.id || "") === String(taskTemplateID || ""));
  return inputText(task?.name);
}

function taskNodeName(taskName: string) {
  return taskName ? `任务：${taskName}` : "任务";
}

function normalizedNodeType(value: any) {
  const nodeType = inputText(value);
  return NODE_TYPES.some((item) => item.id === nodeType) ? nodeType : "task";
}

function departmentIDFromStaff(staff: Row[], staffID: string) {
  if (!staffID) {
    return "";
  }
  const selectedStaff = staff.find((item) => stringValue(item.id) === staffID);
  return selectedStaff ? stringValue(selectedStaff.department_id) : "";
}

function stageStableKey(stage: Row) {
  return inputText(stage.stage_key) || rowKey(stage);
}

function nodeStableKey(node: Row) {
  return inputText(node.node_key) || rowKey(node);
}

function nodeBelongsToStage(node: Row, stage: Row) {
  if (node.stage_id && stage.id) {
    return String(node.stage_id) === String(stage.id);
  }
  return inputText(node._stage_key) === rowKey(stage);
}

function findSavedStage(savedStages: Row[], previousStages: Row[], selectedKey: string) {
  const selectedStage = previousStages.find((stage) => rowKey(stage) === selectedKey);
  if (!selectedStage) {
    return savedStages.find((stage) => rowKey(stage) === selectedKey);
  }
  const selectedStageKey = inputText(selectedStage.stage_key);
  return (
    savedStages.find((stage) => rowKey(stage) === selectedKey) ||
    savedStages.find((stage) => selectedStageKey && inputText(stage.stage_key) === selectedStageKey) ||
    null
  );
}

function edgeConnectedToNode(edge: Row, nodeKey: string) {
  return inputText(edge.from_node_key) === nodeKey || inputText(edge.to_node_key) === nodeKey;
}

function edgeTouchesAnyNode(edge: Row, nodeKeys: Set<string>) {
  return nodeKeys.has(inputText(edge.from_node_key)) || nodeKeys.has(inputText(edge.to_node_key));
}

function normalizeStageEdges(value: any): Row[] {
  const edges = Array.isArray(value) ? value : [];
  return edges
    .map((edge, index) => ({
      _key: inputText(edge?._key) || `stage_edge_${index}`,
      from_stage_key: inputText(edge?.from_stage_key),
      to_stage_key: inputText(edge?.to_stage_key),
      status: Number(edge?.status || 1),
      sort: Number(edge?.sort || (index + 1) * 10),
    }))
    .filter((edge) => edge.status === 1 && edge.from_stage_key && edge.to_stage_key && edge.from_stage_key !== edge.to_stage_key)
    .sort(compareSort);
}

function stageEdgeTouchesStage(edge: Row, stage: Row) {
  const key = stageStableKey(stage);
  return inputText(edge.from_stage_key) === key || inputText(edge.to_stage_key) === key;
}

function createNodeEdgeRow(
  fromNode: Row,
  toNode: Row,
  currentEdges: Row[],
  flowTemplateID: number | string,
) {
  const fromNodeKey = nodeStableKey(fromNode);
  const toNodeKey = nodeStableKey(toNode);
  if (!fromNodeKey || !toNodeKey || fromNodeKey === toNodeKey) {
    return null;
  }
  if (
    currentEdges.some(
      (edge) =>
        inputText(edge.from_node_key) === fromNodeKey &&
        inputText(edge.to_node_key) === toNodeKey &&
        Number(edge.status || 1) === 1,
    )
  ) {
    return null;
  }
  return {
    _key: tempKey("edge"),
    flow_template_id: flowTemplateID || fromNode.flow_template_id || toNode.flow_template_id || "",
    from_node_id: fromNode.id || "",
    from_node_key: fromNodeKey,
    to_node_id: toNode.id || "",
    to_node_key: toNodeKey,
    match_result: "",
    match_script_id: "",
    target_resource_status: "",
    target_department_id: "",
    target_role_id: "",
    condition_json: "{}",
    status: 1,
    sort: nextSort(currentEdges),
  };
}

function resolveProximityConnection(
  draggedNode: CrmGraphNode,
  graphNodes: CrmGraphNode[],
  graphEdges: Row[],
): ProximityConnection | null {
  const draggedInfo = graphNodeInfo(draggedNode.id);
  if (
    (draggedInfo.kind !== "stage" && draggedInfo.kind !== "node") ||
    !isGraphNodeIsolated(draggedNode, graphEdges)
  ) {
    return null;
  }

  const draggedCenter = nodeCenter(draggedNode.position);
  let closestNode: CrmGraphNode | null = null;
  let closestDistance = Number.MAX_VALUE;
  graphNodes.forEach((node) => {
    if (node.id === draggedNode.id || graphNodeInfo(node.id).kind !== draggedInfo.kind) {
      return;
    }
    const center = nodeCenter(node.position);
    const distance = Math.hypot(center.x - draggedCenter.x, center.y - draggedCenter.y);
    if (distance < closestDistance && distance < PROXIMITY_CONNECT_DISTANCE) {
      closestNode = node;
      closestDistance = distance;
    }
  });

  if (!closestNode) {
    return null;
  }

  const closestIsSource = closestNode.position.x < draggedNode.position.x;
  const connection = {
    source: closestIsSource ? closestNode.id : draggedNode.id,
    target: closestIsSource ? draggedNode.id : closestNode.id,
  };
  return graphEdgeExists(graphEdges, connection, graphNodes) ? null : connection;
}

function graphItemStableKey(node: CrmGraphNode) {
  return node.data.kind === "stage" ? stageStableKey(node.data.item) : nodeStableKey(node.data.item);
}

function isGraphNodeIsolated(node: CrmGraphNode, graphEdges: Row[]) {
  const key = graphItemStableKey(node);
  return !graphEdges.some((edge) => edgeTouchesGraphItem(edge, node.data.kind, key));
}

function graphEdgeExists(
  graphEdges: Row[],
  connection: ProximityConnection,
  graphNodes: CrmGraphNode[],
) {
  const nodesByID = new Map(graphNodes.map((node) => [node.id, node]));
  const source = nodesByID.get(connection.source);
  const target = nodesByID.get(connection.target);
  if (!source || !target || source.data.kind !== target.data.kind) {
    return false;
  }
  const sourceKey = graphItemStableKey(source);
  const targetKey = graphItemStableKey(target);
  if (source.data.kind === "stage") {
    return graphEdges.some(
      (edge) =>
        inputText(edge.from_stage_key) === sourceKey &&
        inputText(edge.to_stage_key) === targetKey,
    );
  }
  return graphEdges.some(
    (edge) =>
      inputText(edge.from_node_key) === sourceKey &&
      inputText(edge.to_node_key) === targetKey,
  );
}

function edgeTouchesGraphItem(edge: Row, kind: "stage" | "node", key: string) {
  if (kind === "stage") {
    return inputText(edge.from_stage_key) === key || inputText(edge.to_stage_key) === key;
  }
  return inputText(edge.from_node_key) === key || inputText(edge.to_node_key) === key;
}

function sameProximityConnection(
  left: ProximityConnection | null,
  right: ProximityConnection | null,
) {
  return left?.source === right?.source && left?.target === right?.target;
}

function proximityConnectionID(connection: ProximityConnection) {
  return `proximity:${connection.source}:${connection.target}`;
}

function edgeSelectionFromData(data?: GraphEdgeData): Exclude<Selection, null> {
  return data?.kind === "stage"
    ? { kind: "stage_edge", index: data?.index ?? 0 }
    : { kind: "edge", index: data?.index ?? 0 };
}

function graphEdgeHighlighted(
  kind: "stage" | "node",
  edge: Row,
  index: number,
  selection: Selection,
  selectedStageStableKey = "",
  selectedNodeStableKey = "",
) {
  if (kind === "stage") {
    if (selection?.kind === "stage_edge") {
      return selection.index === index;
    }
    if (selection?.kind === "stage") {
      return (
        inputText(edge.from_stage_key) === selectedStageStableKey ||
        inputText(edge.to_stage_key) === selectedStageStableKey
      );
    }
    return false;
  }
  if (selection?.kind === "edge") {
    return selection.index === index;
  }
  if (selection?.kind === "node") {
    return (
      inputText(edge.from_node_key) === selectedNodeStableKey ||
      inputText(edge.to_node_key) === selectedNodeStableKey
    );
  }
  return false;
}

function graphEdgeStyle(
  kind: "stage" | "node",
  edge: Row,
  index: number,
  selection: Selection,
  selectedStageStableKey = "",
  selectedNodeStableKey = "",
): CSSProperties {
  const highlighted = graphEdgeHighlighted(
    kind,
    edge,
    index,
    selection,
    selectedStageStableKey,
    selectedNodeStableKey,
  );
  return {
    stroke: highlighted ? "#6366f1" : kind === "stage" ? "#d4d4d8" : "#94a3b8",
    strokeWidth: highlighted ? 2.4 : 1.6,
    strokeDasharray: highlighted ? "8 7" : kind === "stage" ? "7 9" : "8 7",
    strokeLinecap: "round",
    strokeLinejoin: "round",
    filter: highlighted ? "drop-shadow(0 0 5px rgb(99 102 241 / 0.42))" : undefined,
  };
}

function canvasPointFromPosition(position: Partial<{ x: number; y: number }>, yOffset = 0) {
  const x = Number(position.x);
  const y = Number(position.y) + yOffset;
  return {
    x: Number.isFinite(x) ? x : 0,
    y: Number.isFinite(y) ? y : 0,
  };
}

function connectedNodePosition(
  sourcePosition: { x: number; y: number } | undefined,
  dropPoint: { x: number; y: number },
) {
  const dropPosition = canvasPointFromPosition(dropPoint, -CARD_HEIGHT / 2);
  if (!sourcePosition) {
    return dropPosition;
  }
  const sourceCenter = nodeCenter(sourcePosition);
  const dropCenter = nodeCenter(dropPosition);
  const dx = dropCenter.x - sourceCenter.x;
  const dy = dropCenter.y - sourceCenter.y;
  const distance = Math.hypot(dx, dy);
  if (distance > 0 && distance <= CONNECTED_STAGE_MAX_DISTANCE) {
    return dropPosition;
  }
  const directionX = distance > 0 ? dx / distance : 1;
  const directionY = distance > 0 ? dy / distance : 0;
  return {
    x: sourceCenter.x + directionX * CONNECTED_STAGE_FALLBACK_DISTANCE - CARD_WIDTH / 2,
    y: sourceCenter.y + directionY * CONNECTED_STAGE_FALLBACK_DISTANCE - CARD_HEIGHT / 2,
  };
}

function nodeCenter(position: { x: number; y: number }) {
  return {
    x: position.x + CARD_WIDTH / 2,
    y: position.y + CARD_HEIGHT / 2,
  };
}

function clientPointFromConnectEvent(event: MouseEvent | TouchEvent) {
  if ("clientX" in event) {
    return { x: event.clientX, y: event.clientY };
  }
  const touch = event.changedTouches[0] ?? event.touches[0];
  return touch ? { x: touch.clientX, y: touch.clientY } : null;
}

function isMeaningfulConnectDrag(
  source: { x: number; y: number } | undefined,
  point: { x: number; y: number },
) {
  if (!source) {
    return false;
  }
  const start = {
    x: source.x + CARD_WIDTH,
    y: source.y + CARD_HEIGHT / 2,
  };
  return Math.hypot(point.x - start.x, point.y - start.y) > 48;
}
