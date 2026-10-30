"use client";

import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import {
  ReactFlow,
  Background,
  MiniMap,
  applyNodeChanges,
  type OnNodesChange,
  type OnConnect,
  type OnConnectStart,
  type OnConnectEnd,
  type EdgeMouseHandler,
  type Node,
  type Edge,
  BackgroundVariant,
  MarkerType,
  type Connection,
  useReactFlow,
  ReactFlowProvider,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { useFocusTreeStore } from "@/lib/store/focusTreeStore";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import type { FocusNode } from "@/types";
import {
  focusNodesToRFNodes,
  buildFocusGraphNode,
  focusNodesToRFEdges,
  pixelToHoi4,
  hoi4ToPixel,
  resolveAbsolutePosition,
  focusVisualSubtree,
  parseEdgeId,
  bfsLayoutFocusNodes,
  repairFocusLayout,
  compactFocusLayoutHorizontally,
  GRID_CELL_WIDTH,
  GRID_CELL_HEIGHT,
  GRID_LINE_OFFSET_X,
  GRID_LINE_OFFSET_Y,
  NODE_WIDTH,
  NODE_HEIGHT_NOMINAL,
  type FocusGraphNodeData,
} from "@/lib/focusTreeGraph";
import { toast } from "sonner";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { exportTreePng } from "@/lib/exportTreePng";
import { useBaseModDepKeys } from "@/lib/api/basemod";
import { applyEdgeDisplayState } from "@/lib/focusEdgeDisplay";
import { focusNodeTypes, focusEdgeTypes } from "./index";
import { FocusGraphRuler } from "./FocusGraphRuler";
import { FocusGraphToolbar } from "./FocusGraphToolbar";
import { FocusInGamePreviewContext } from "./inGamePreviewContext";
import { FocusContextMenu } from "./FocusContextMenu";
import {
  sameSelectedNodeIds,
  selectedNodeIdsAfterChanges,
  splitControlledSelectionChanges,
} from "./controlledNodeSelection";

const DEFAULT_EDGE_OPTIONS = {
  markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16 },
} as const;

const PREREQ_CONNECTION_LINE_STYLE = {
  stroke: "var(--color-muted-foreground)",
  strokeWidth: 2,
} as const;

const MUTEX_CONNECTION_LINE_STYLE = {
  stroke: "var(--color-destructive)",
  strokeWidth: 2,
  strokeDasharray: "6 3",
} as const;

interface FocusTreeGraphProps {
  nodes: FocusNode[];
  selectedNodeId: number | null;
  selectedNodeIds: number[];
  selectedEdgeId: string | null;
  projectId: number;
  treeId: number;

  filenameBase?: string;
  availableIcons?: Set<string>;

  iconCacheBust?: { gfxName?: string; version: number };
  onSelectNodes: (nodeIds: number[]) => void;
  onSelectEdge: (edgeId: string | null) => void;
  onUpdatePosition: (nodeId: number, x: number, y: number) => void;

  onBulkUpdatePositions: (
    positions: Array<{ id: number; x: number; y: number }>,
  ) => void;
  onCreateNode: (data?: Partial<FocusNode>) => void;
  onDeleteNode: (nodeId: number) => void;
  onAddPrerequisite: (nodeId: number, prereqFocusId: string) => void;
  onRemovePrerequisite: (
    nodeId: number,
    prereqFocusId: string,
    groupIndex: number,
  ) => void;
  onAddMutualExclusion: (nodeId: number, otherFocusId: string) => void;
  onRemoveMutualExclusion: (nodeId: number, otherFocusId: string) => void;
  onCreateChildNode?: (parentNodeId: number) => void;

  onCreateMutexChildNode?: (
    parentNodeId: number,
    side: "left" | "right",
  ) => void;

  onCreateConnectedNode?: (
    parentNodeId: number,
    connectionType: "prereq" | "mutex",
    position: { x: number; y: number },
  ) => Promise<void>;
}

function FocusTreeGraphInner({
  nodes: focusNodes,
  selectedNodeId,
  selectedNodeIds,
  selectedEdgeId,
  projectId,
  treeId,
  filenameBase,
  availableIcons,
  iconCacheBust,
  onSelectNodes,
  onSelectEdge,
  onUpdatePosition,
  onBulkUpdatePositions,
  onCreateNode,
  onDeleteNode,
  onAddPrerequisite,
  onRemovePrerequisite,
  onAddMutualExclusion,
  onRemoveMutualExclusion,
  onCreateChildNode,
  onCreateMutexChildNode,
  onCreateConnectedNode,
}: FocusTreeGraphProps) {
  const t = useTranslations("focusTreeGraph");
  const confirmDialog = useConfirm();

  const locale = useContentLocaleStore((s) => s.locale);

  const baseModDepKeys = useBaseModDepKeys();
  const rfInstance = useReactFlow();
  const { screenToFlowPosition } = rfInstance;
  const [exporting, setExporting] = useState(false);

  const [inGamePreview, setInGamePreview] = useState(false);

  const pendingFitRef = useRef(false);

  const isDraggingRef = useRef(false);

  const dragSubtreeRef = useRef<{
    descIds: Set<string>;
    startPx: Map<string, { x: number; y: number }>;
    rootStartPx: { x: number; y: number };
  } | null>(null);

  const [hoveredEdgeId, setHoveredEdgeId] = useState<string | null>(null);
  const [hoveredNodeId, setHoveredNodeId] = useState<number | null>(null);

  const [contextMenu, setContextMenu] = useState<
    | { kind: "edge"; id: string; x: number; y: number }
    | { kind: "node"; id: string; x: number; y: number }
    | { kind: "selection"; ids: number[]; x: number; y: number }
    | null
  >(null);
  const [connectHint, setConnectHint] = useState<{
    type: "prereq" | "mutex";
    x: number;
    y: number;
  } | null>(null);

  const [draggedConnection, setDraggedConnection] = useState<{
    fromNodeId: number;
    handleType: "prereq" | "mutex";
  } | null>(null);

  const historyUndoLen = useFocusTreeStore((s) => s.historyUndo.length);
  const historyRedoLen = useFocusTreeStore((s) => s.historyRedo.length);
  const undo = useFocusTreeStore((s) => s.undo);
  const redo = useFocusTreeStore((s) => s.redo);
  const duplicateNode = useFocusTreeStore((s) => s.duplicateNode);
  const selectAllNodes = useFocusTreeStore((s) => s.selectAllNodes);
  const pendingViewportFocusId = useFocusTreeStore(
    (s) => s.pendingViewportFocusId,
  );
  const clearPendingViewportFocus = useFocusTreeStore(
    (s) => s.clearPendingViewportFocus,
  );
  const canUndo = historyUndoLen > 0;
  const canRedo = historyRedoLen > 0;

  const handleUndo = useCallback(() => {
    if (canUndo) undo(projectId);
  }, [canUndo, undo, projectId]);

  const handleRedo = useCallback(() => {
    if (canRedo) redo(projectId);
  }, [canRedo, redo, projectId]);

  const handleDuplicate = useCallback(() => {
    if (selectedNodeId != null)
      duplicateNode(projectId, treeId, selectedNodeId);
  }, [selectedNodeId, duplicateNode, projectId, treeId]);

  const handleSelectAll = useCallback(() => {
    selectAllNodes(treeId);
  }, [selectAllNodes, treeId]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.ctrlKey || e.metaKey)) return;
      const target = e.target as HTMLElement | null;
      if (target) {
        const tag = target.tagName;
        if (tag === "INPUT" || tag === "TEXTAREA" || target.isContentEditable)
          return;
      }
      const k = e.key.toLowerCase();
      if (k === "z" && !e.shiftKey) {
        e.preventDefault();
        handleUndo();
      } else if (k === "y" || (k === "z" && e.shiftKey)) {
        e.preventDefault();
        handleRedo();
      } else if (k === "d") {
        e.preventDefault();
      } else if (k === "a") {
        e.preventDefault();
        handleSelectAll();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [handleUndo, handleRedo, handleSelectAll]);

  const nodeCacheRef = useRef<Map<FocusNode, Node<FocusGraphNodeData>>>(
    new Map(),
  );
  const prevGlobalDepsRef = useRef<unknown[] | null>(null);
  const prevFocusNodesRef = useRef<FocusNode[] | null>(null);
  const prevIconCacheBustRef = useRef<typeof iconCacheBust>(undefined);
  const selectedNodeIdsRef = useRef(selectedNodeIds);
  useEffect(() => {
    selectedNodeIdsRef.current = selectedNodeIds;
  }, [selectedNodeIds]);
  const selectedNodeIdSet = useMemo(
    () => new Set(selectedNodeIds),
    [selectedNodeIds],
  );

  const baseRFNodes = useMemo(() => {
    const globalDeps = [
      availableIcons,
      projectId,
      locale,
      baseModDepKeys,
      onCreateChildNode,
      onCreateMutexChildNode,
    ];
    const prev = prevGlobalDepsRef.current;
    const globalDepsChanged =
      !prev ||
      prev.length !== globalDeps.length ||
      globalDeps.some((v, i) => !Object.is(v, prev[i]));
    prevGlobalDepsRef.current = globalDeps;
    const focusStructureChanged = prevFocusNodesRef.current !== focusNodes;
    prevFocusNodesRef.current = focusNodes;
    const previousIconCacheBust = prevIconCacheBustRef.current;
    const iconCacheBustChanged = previousIconCacheBust !== iconCacheBust;
    prevIconCacheBustRef.current = iconCacheBust;
    const globalIconBust =
      iconCacheBustChanged &&
      ((iconCacheBust != null && !iconCacheBust.gfxName) ||
        (previousIconCacheBust != null && !previousIconCacheBust.gfxName));

    const hasRelativeNode = focusNodes.some((n) => !!n.relative_position_id);

    if (
      globalDepsChanged ||
      globalIconBust ||
      (hasRelativeNode && focusStructureChanged)
    ) {
      const fresh = focusNodesToRFNodes(
        focusNodes,
        selectedNodeId,
        availableIcons,
        projectId,
        iconCacheBust,
        locale,
        baseModDepKeys,
        onCreateChildNode,
        onCreateMutexChildNode,
      ).map((node) => {
        const selected = selectedNodeIdSet.has(Number(node.id));
        return node.selected === selected ? node : { ...node, selected };
      });
      const nextCache = new Map<FocusNode, Node<FocusGraphNodeData>>();
      focusNodes.forEach((fn, i) => nextCache.set(fn, fresh[i]));
      nodeCacheRef.current = nextCache;
      return fresh;
    }

    const prevCache = nodeCacheRef.current;
    const nextCache = new Map<FocusNode, Node<FocusGraphNodeData>>();
    const result = focusNodes.map((fn) => {
      const cached = prevCache.get(fn);
      const shouldBeSelected = selectedNodeIdSet.has(fn.id);
      const scopedIconChanged =
        iconCacheBustChanged &&
        (fn.icon === iconCacheBust?.gfxName ||
          fn.icon === previousIconCacheBust?.gfxName);
      if (
        cached &&
        cached.selected === shouldBeSelected &&
        !scopedIconChanged
      ) {
        nextCache.set(fn, cached);
        return cached;
      }
      const baseNode = buildFocusGraphNode(
        fn,
        focusNodes,
        selectedNodeId,
        availableIcons,
        projectId,
        iconCacheBust,
        locale,
        baseModDepKeys,
        onCreateChildNode,
        onCreateMutexChildNode,
      );
      const rebuilt =
        baseNode.selected === shouldBeSelected
          ? baseNode
          : { ...baseNode, selected: shouldBeSelected };
      nextCache.set(fn, rebuilt);
      return rebuilt;
    });
    nodeCacheRef.current = nextCache;
    return result;
  }, [
    focusNodes,
    selectedNodeId,
    selectedNodeIdSet,
    availableIcons,
    projectId,
    iconCacheBust,
    locale,
    baseModDepKeys,
    onCreateChildNode,
    onCreateMutexChildNode,
  ]);
  const rfEdges = useMemo(() => focusNodesToRFEdges(focusNodes), [focusNodes]);

  const displayEdges = useMemo(
    () =>
      applyEdgeDisplayState(rfEdges, {
        selectedNodeId,
        selectedEdgeId,
        hoveredEdgeId,
        hoveredNodeId,
      }),
    [rfEdges, selectedNodeId, selectedEdgeId, hoveredEdgeId, hoveredNodeId],
  );

  const [nodes, setNodes] = useState<Node<FocusGraphNodeData>[]>(baseRFNodes);
  useEffect(() => {
    if (isDraggingRef.current) {
      return;
    }
    setNodes(baseRFNodes);
  }, [baseRFNodes]);

  const handleNodesChange: OnNodesChange<Node<FocusGraphNodeData>> =
    useCallback(
      (changes) => {
        const { selectionChanges, renderChanges } =
          splitControlledSelectionChanges(changes);
        if (selectionChanges.length > 0) {
          const current = selectedNodeIdsRef.current;
          const next = selectedNodeIdsAfterChanges(current, selectionChanges);
          if (!sameSelectedNodeIds(current, next)) {
            selectedNodeIdsRef.current = next;
            onSelectNodes(next);
          }
        }
        if (renderChanges.length > 0) {
          setNodes((nds) => applyNodeChanges(renderChanges, nds));
        }
      },
      [onSelectNodes],
    );

  const pixelToStored = useCallback(
    (
      focusNode: FocusNode,
      pixel: { x: number; y: number },
      refBaseline: FocusNode[],
    ) => {
      const abs = pixelToHoi4(pixel.x, pixel.y);
      if (focusNode.relative_position_id) {
        const refNode = refBaseline.find(
          (n) => n.focus_id === focusNode.relative_position_id,
        );
        if (refNode) {
          const refAbs = resolveAbsolutePosition(refNode, refBaseline);
          return { x: abs.x - refAbs.x, y: abs.y - refAbs.y };
        }
      }
      return { x: abs.x, y: abs.y };
    },
    [],
  );

  const persistDraggedNodes = useCallback(
    (dragged: Node<FocusGraphNodeData>[]) => {
      const updates: Array<{ id: number; x: number; y: number }> = [];
      for (const dn of dragged) {
        const fn = focusNodes.find((n) => n.id === Number(dn.id));
        if (!fn) continue;
        const s = pixelToStored(fn, dn.position, focusNodes);
        updates.push({ id: fn.id, x: s.x, y: s.y });
      }
      if (updates.length > 0) onBulkUpdatePositions(updates);
    },
    [focusNodes, onBulkUpdatePositions, pixelToStored],
  );

  const handleNodeDragStart = useCallback(
    (
      _event: React.MouseEvent,
      node: Node<FocusGraphNodeData>,
      dragged: Node<FocusGraphNodeData>[],
    ) => {
      isDraggingRef.current = true;
      if (dragged.length > 1) {
        dragSubtreeRef.current = null;
        return;
      }
      const rootDbId = Number(node.id);
      const descIds = new Set(
        focusVisualSubtree(focusNodes, rootDbId).map(String),
      );
      const startPx = new Map<string, { x: number; y: number }>();
      for (const n of focusNodes) {
        if (!descIds.has(String(n.id))) continue;
        const abs = resolveAbsolutePosition(n, focusNodes);
        startPx.set(String(n.id), hoi4ToPixel(abs.x, abs.y));
      }
      dragSubtreeRef.current = {
        descIds,
        startPx,
        rootStartPx: { ...node.position },
      };
    },
    [focusNodes],
  );

  const handleNodeDrag = useCallback(
    (_event: React.MouseEvent, node: Node<FocusGraphNodeData>) => {
      const ctx = dragSubtreeRef.current;
      if (!ctx || ctx.descIds.size === 0) return;
      const dx = node.position.x - ctx.rootStartPx.x;
      const dy = node.position.y - ctx.rootStartPx.y;
      setNodes((nds) =>
        nds.map((n) => {
          const s = ctx.descIds.has(n.id) ? ctx.startPx.get(n.id) : undefined;
          return s ? { ...n, position: { x: s.x + dx, y: s.y + dy } } : n;
        }),
      );
    },
    [],
  );

  const handleNodeDragStop = useCallback(
    (
      _event: React.MouseEvent,
      node: Node<FocusGraphNodeData>,
      dragged: Node<FocusGraphNodeData>[],
    ) => {
      isDraggingRef.current = false;
      const ctx = dragSubtreeRef.current;
      dragSubtreeRef.current = null;

      if (dragged.length > 1) {
        persistDraggedNodes(dragged);
        return;
      }

      const dbId = Number(node.id);
      const focusNode = focusNodes.find((n) => n.id === dbId);
      if (!focusNode) return;

      const rootStored = pixelToStored(focusNode, node.position, focusNodes);
      if (!ctx || ctx.descIds.size === 0) {
        onUpdatePosition(dbId, rootStored.x, rootStored.y);
        return;
      }

      const dx = node.position.x - ctx.rootStartPx.x;
      const dy = node.position.y - ctx.rootStartPx.y;
      const updates: Array<{ id: number; x: number; y: number }> = [
        { id: dbId, ...rootStored },
      ];
      for (const n of focusNodes) {
        const s = ctx.descIds.has(String(n.id))
          ? ctx.startPx.get(String(n.id))
          : undefined;
        if (!s) continue;
        const stored = pixelToStored(
          n,
          { x: s.x + dx, y: s.y + dy },
          focusNodes,
        );
        updates.push({ id: n.id, x: stored.x, y: stored.y });
      }
      onBulkUpdatePositions(updates);
    },
    [
      focusNodes,
      onUpdatePosition,
      onBulkUpdatePositions,
      pixelToStored,
      persistDraggedNodes,
    ],
  );

  const handleSelectionDragStart = useCallback(() => {
    isDraggingRef.current = true;
    dragSubtreeRef.current = null;
  }, []);
  const handleSelectionDragStop = useCallback(
    (_event: React.MouseEvent, dragged: Node<FocusGraphNodeData>[]) => {
      isDraggingRef.current = false;
      persistDraggedNodes(dragged);
    },
    [persistDraggedNodes],
  );

  const handleEdgeClick: EdgeMouseHandler = useCallback(
    (_event, edge) => {
      onSelectEdge(edge.id);
    },
    [onSelectEdge],
  );

  const handleAutoLayout = useCallback(() => {
    const repair = repairFocusLayout(focusNodes);
    if (!repair.needsFullLayout) {
      const conflictPairs = repair.unaligned.reduce(
        (n, u) => n + u.focusIds.length,
        0,
      );
      if (repair.changes.length === 0) {
        if (conflictPairs > 0) {
          toast.warning(t("layoutMutexConflict", { count: conflictPairs }));
        } else {
          toast.success(t("layoutAlreadyTidy"));
        }
        return;
      }
      onBulkUpdatePositions(
        [...repair.positions].map(([id, pos]) => ({ id, x: pos.x, y: pos.y })),
      );
      if (conflictPairs > 0) {
        toast.warning(
          t("layoutTidied", { count: repair.changes.length }) +
            " · " +
            t("layoutMutexConflict", { count: conflictPairs }),
        );
      } else {
        toast.success(t("layoutTidied", { count: repair.changes.length }));
      }
      pendingFitRef.current = true;
      return;
    }
    const newPositions = bfsLayoutFocusNodes(focusNodes);

    onBulkUpdatePositions(
      [...newPositions].map(([id, pos]) => ({ id, x: pos.x, y: pos.y })),
    );

    pendingFitRef.current = true;
  }, [focusNodes, onBulkUpdatePositions, t]);

  const handleCompactLayout = useCallback(async () => {
    const res = compactFocusLayoutHorizontally(focusNodes);
    if (res.changes === 0) {
      toast.success(t("compactNone"));
      return;
    }

    const percent = Math.round((res.changes / focusNodes.length) * 100);
    const ok = await confirmDialog({
      body: t("compactConfirm", {
        total: focusNodes.length,
        count: res.changes,
        percent,
      }),
    });
    if (!ok) return;
    onBulkUpdatePositions(
      [...res.positions].map(([id, pos]) => ({ id, x: pos.x, y: pos.y })),
    );
    toast.success(t("compactDone", { count: res.changes }));
    pendingFitRef.current = true;
  }, [focusNodes, onBulkUpdatePositions, t, confirmDialog]);

  useEffect(() => {
    if (!pendingFitRef.current) return;
    pendingFitRef.current = false;
    rfInstance.fitView({ padding: 0.2, duration: 300 });
  }, [nodes, rfInstance]);

  useEffect(() => {
    if (pendingViewportFocusId == null) return;
    const target = nodes.find((n) => n.id === String(pendingViewportFocusId));
    if (!target) return;
    const w = target.measured?.width ?? NODE_WIDTH;
    const h = target.measured?.height ?? NODE_HEIGHT_NOMINAL;
    rfInstance.setCenter(target.position.x + w / 2, target.position.y + h / 2, {
      zoom: rfInstance.getZoom(),
      duration: 300,
    });
    clearPendingViewportFocus();
  }, [pendingViewportFocusId, nodes, rfInstance, clearPendingViewportFocus]);

  const handleExportPng = useCallback(async () => {
    if (focusNodes.length === 0) return;
    setExporting(true);
    try {
      await exportTreePng(rfInstance, { filenameBase });
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      toast.error(`PNG export failed: ${msg}`);
    } finally {
      setExporting(false);
    }
  }, [focusNodes.length, rfInstance, filenameBase]);

  const handlePaneClick = useCallback(() => {
    if (selectedNodeIdsRef.current.length > 0) {
      selectedNodeIdsRef.current = [];
      onSelectNodes([]);
    }
    onSelectEdge(null);
    setContextMenu(null);
  }, [onSelectNodes, onSelectEdge]);

  const handleConnect: OnConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target) return;

      const sourceId = Number(connection.source);
      const targetId = Number(connection.target);

      if (sourceId === targetId) return;

      const isMutex =
        connection.sourceHandle === "mutex-left" ||
        connection.targetHandle === "mutex-right";

      if (isMutex) {
        const sourceNode = focusNodes.find((n) => n.id === sourceId);
        if (sourceNode) {
          onAddMutualExclusion(targetId, sourceNode.focus_id);
        }
      } else {
        const sourceNode = focusNodes.find((n) => n.id === sourceId);
        if (sourceNode) {
          onAddPrerequisite(targetId, sourceNode.focus_id);
        }
      }
    },
    [focusNodes, onAddPrerequisite, onAddMutualExclusion],
  );

  const isValidConnection = useCallback(
    (conn: Connection | Edge) => conn.source !== conn.target,
    [],
  );

  const removeEdgeById = useCallback(
    (edgeId: string) => {
      const parsed = parseEdgeId(edgeId);
      if (!parsed) return;
      if (parsed.type === "prereq" && parsed.groupIndex !== undefined) {
        const sourceNode = focusNodes.find((n) => n.id === parsed.sourceDbId);
        if (sourceNode) {
          onRemovePrerequisite(
            parsed.targetDbId,
            sourceNode.focus_id,
            parsed.groupIndex,
          );
        }
      } else if (parsed.type === "mutex") {
        const targetNode = focusNodes.find((n) => n.id === parsed.targetDbId);
        if (targetNode) {
          onRemoveMutualExclusion(parsed.sourceDbId, targetNode.focus_id);
        }
      }
    },
    [focusNodes, onRemovePrerequisite, onRemoveMutualExclusion],
  );

  const handleEdgesDelete = useCallback(
    (edges: Edge[]) => {
      for (const edge of edges) removeEdgeById(edge.id);
    },
    [removeEdgeById],
  );

  const handleEdgeMouseEnter = useCallback(
    (_e: React.MouseEvent, edge: Edge) => setHoveredEdgeId(edge.id),
    [],
  );
  const handleEdgeMouseLeave = useCallback(() => setHoveredEdgeId(null), []);

  const handleNodeMouseEnter = useCallback(
    (_e: React.MouseEvent, node: Node) => setHoveredNodeId(Number(node.id)),
    [],
  );
  const handleNodeMouseLeave = useCallback(() => setHoveredNodeId(null), []);

  const handleEdgeContextMenu = useCallback(
    (e: React.MouseEvent, edge: Edge) => {
      e.preventDefault();
      setContextMenu({ kind: "edge", id: edge.id, x: e.clientX, y: e.clientY });
    },
    [],
  );

  const handleNodeContextMenu = useCallback(
    (e: React.MouseEvent, node: Node) => {
      e.preventDefault();

      const selected = nodes.filter((n) => n.selected);
      if (selected.length > 1 && selected.some((n) => n.id === node.id)) {
        setContextMenu({
          kind: "selection",
          ids: selected.map((n) => Number(n.id)),
          x: e.clientX,
          y: e.clientY,
        });
      } else {
        setContextMenu({
          kind: "node",
          id: node.id,
          x: e.clientX,
          y: e.clientY,
        });
      }
    },
    [nodes],
  );

  const handleSelectionContextMenu = useCallback(
    (e: React.MouseEvent, selNodes: Node[]) => {
      e.preventDefault();
      setContextMenu({
        kind: "selection",
        ids: selNodes.map((n) => Number(n.id)),
        x: e.clientX,
        y: e.clientY,
      });
    },
    [],
  );

  const handleCreateConnectedNode = useCallback(
    async (
      parentNodeId: number,
      connectionType: "prereq" | "mutex",
      position: { x: number; y: number },
    ) => {
      if (onCreateConnectedNode) {
        try {
          await onCreateConnectedNode(parentNodeId, connectionType, position);
        } catch (err) {
          const msg = err instanceof Error ? err.message : String(err);
          toast.error(`Failed to create connected node: ${msg}`);
        }
      }
    },
    [onCreateConnectedNode],
  );

  const handleConnectStart: OnConnectStart = useCallback((event, params) => {
    const type =
      params.handleId === "mutex-left" || params.handleId === "mutex-right"
        ? "mutex"
        : "prereq";
    const pt = "touches" in event ? event.touches[0] : (event as MouseEvent);
    setConnectHint({ type, x: pt.clientX, y: pt.clientY });

    setDraggedConnection({
      fromNodeId: Number(params.nodeId),
      handleType: type,
    });
  }, []);

  const handleConnectEnd: OnConnectEnd = useCallback(
    async (event, connectionState) => {
      setConnectHint(null);

      if (connectionState.toNode === null && draggedConnection) {
        const { fromNodeId, handleType } = draggedConnection;

        const mouseEvent = event as MouseEvent;
        const flowPos = screenToFlowPosition({
          x: mouseEvent.clientX,
          y: mouseEvent.clientY,
        });
        const hoi4Pos = pixelToHoi4(flowPos.x, flowPos.y);

        await handleCreateConnectedNode(fromNodeId, handleType, hoi4Pos);
      }

      setDraggedConnection(null);
    },
    [draggedConnection, screenToFlowPosition, handleCreateConnectedNode],
  );

  useEffect(() => {
    if (!connectHint) return;
    const onMove = (e: MouseEvent) => {
      setConnectHint((h) => (h ? { ...h, x: e.clientX, y: e.clientY } : h));
    };
    window.addEventListener("mousemove", onMove);
    return () => window.removeEventListener("mousemove", onMove);
  }, [connectHint]);

  const handlePaneDoubleClick = useCallback(
    (event: React.MouseEvent) => {
      const target = event.target as HTMLElement;

      if (!target.closest(".react-flow__pane")) return;
      if (target.closest(".react-flow__node")) return;
      const flowPos = screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });
      const hoi4Pos = pixelToHoi4(flowPos.x, flowPos.y);
      onCreateNode({ x: hoi4Pos.x, y: hoi4Pos.y });
    },
    [screenToFlowPosition, onCreateNode],
  );

  const handleNodesDelete = useCallback(
    (deletedNodes: Node[]) => {
      for (const node of deletedNodes) {
        onDeleteNode(Number(node.id));
      }
    },
    [onDeleteNode],
  );

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Delete" && e.key !== "Backspace") return;
      const target = e.target as HTMLElement | null;
      if (target) {
        const tag = target.tagName;
        if (tag === "INPUT" || tag === "TEXTAREA" || target.isContentEditable)
          return;
      }
      if (!selectedEdgeId) return;
      e.preventDefault();
      removeEdgeById(selectedEdgeId);
      onSelectEdge(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [selectedEdgeId, removeEdgeById, onSelectEdge]);

  return (
    <>
      <FocusInGamePreviewContext.Provider value={inGamePreview}>
        <ReactFlow
          nodes={nodes}
          edges={displayEdges}
          nodeTypes={focusNodeTypes}
          edgeTypes={focusEdgeTypes}
          onNodesChange={handleNodesChange}
          onNodeDragStart={handleNodeDragStart}
          onNodeDrag={handleNodeDrag}
          onNodeDragStop={handleNodeDragStop}
          onSelectionDragStart={handleSelectionDragStart}
          onSelectionDragStop={handleSelectionDragStop}
          onEdgeClick={handleEdgeClick}
          onPaneClick={handlePaneClick}
          onConnect={handleConnect}
          isValidConnection={isValidConnection}
          onConnectStart={handleConnectStart}
          onConnectEnd={handleConnectEnd}
          onEdgeMouseEnter={handleEdgeMouseEnter}
          onEdgeMouseLeave={handleEdgeMouseLeave}
          onNodeMouseEnter={handleNodeMouseEnter}
          onNodeMouseLeave={handleNodeMouseLeave}
          onEdgeContextMenu={handleEdgeContextMenu}
          onNodeContextMenu={handleNodeContextMenu}
          onSelectionContextMenu={handleSelectionContextMenu}
          onEdgesDelete={handleEdgesDelete}
          onNodesDelete={handleNodesDelete}
          onDoubleClick={handlePaneDoubleClick}
          connectionLineStyle={
            connectHint?.type === "mutex"
              ? MUTEX_CONNECTION_LINE_STYLE
              : PREREQ_CONNECTION_LINE_STYLE
          }
          defaultEdgeOptions={DEFAULT_EDGE_OPTIONS}
          fitView
          fitViewOptions={{ padding: 0.2 }}
          deleteKeyCode={["Delete", "Backspace"]}
          connectionRadius={28}
          multiSelectionKeyCode="Shift"
          className={inGamePreview ? "bg-ingame-panel-deep" : "bg-background"}
          proOptions={{ hideAttribution: true }}
        >
          <Background
            variant={BackgroundVariant.Lines}
            gap={[GRID_CELL_WIDTH, GRID_CELL_HEIGHT]}
            offset={[GRID_LINE_OFFSET_X, GRID_LINE_OFFSET_Y]}
            lineWidth={1}
            color={
              inGamePreview ? "var(--color-ingame-edge)" : "var(--color-border)"
            }
            style={{ opacity: inGamePreview ? 0.25 : 0.5 }}
          />
          <FocusGraphRuler />

          <MiniMap
            nodeColor="var(--color-primary)"
            maskColor="color-mix(in oklab, var(--color-background) 72%, transparent)"
            maskStrokeColor="var(--color-accent)"
            maskStrokeWidth={2}
            pannable
            zoomable
            style={{ right: 12, bottom: 12 }}
            className="rounded-lg! bg-card! border-border! shadow-md! overflow-hidden"
          />

          <FocusGraphToolbar
            canUndo={canUndo}
            canRedo={canRedo}
            onUndo={handleUndo}
            onRedo={handleRedo}
            hasSelection={selectedNodeId != null}
            onDuplicate={handleDuplicate}
            hasNodes={focusNodes.length > 0}
            onAutoLayout={handleAutoLayout}
            onCompactLayout={handleCompactLayout}
            onExportPng={handleExportPng}
            exporting={exporting}
            inGamePreview={inGamePreview}
            onToggleInGamePreview={() => setInGamePreview((value) => !value)}
          />
        </ReactFlow>
      </FocusInGamePreviewContext.Provider>
      {contextMenu && (
        <FocusContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          items={
            contextMenu.kind === "edge"
              ? [
                  {
                    label: t("edgeMenuDelete"),
                    destructive: true,
                    onSelect: () => removeEdgeById(contextMenu.id),
                  },
                ]
              : contextMenu.kind === "selection"
                ? [
                    {
                      label: t("nodeMenuDeleteN", {
                        count: contextMenu.ids.length,
                      }),
                      destructive: true,
                      onSelect: () =>
                        contextMenu.ids.forEach((id) => onDeleteNode(id)),
                    },
                  ]
                : [
                    {
                      label: t("nodeMenuDelete"),
                      destructive: true,
                      onSelect: () => onDeleteNode(Number(contextMenu.id)),
                    },
                  ]
          }
          onClose={() => setContextMenu(null)}
        />
      )}
      {connectHint && (
        <div
          className="ui-glass pointer-events-none fixed z-50 rounded-full px-2 py-1 text-xs"
          style={{ left: connectHint.x + 14, top: connectHint.y + 14 }}
        >
          {connectHint.type === "mutex"
            ? t("connectHintMutex")
            : t("connectHintPrereq")}
        </div>
      )}
    </>
  );
}

export const FocusTreeGraph = memo(function FocusTreeGraph(
  props: FocusTreeGraphProps,
) {
  return (
    <ReactFlowProvider>
      <FocusTreeGraphInner {...props} />
    </ReactFlowProvider>
  );
});
