import { withoutEditRevisions } from "@/components/workbench/save-diagnostics/editRevisions";
import {
  captureFocusPreview,
  type FocusPreviewSnapshot,
} from "@/components/workbench/save-diagnostics/focusPreviewSnapshot";
import { create } from "zustand";
import { subscribeEditRevisionWrites } from "@/components/workbench/save-diagnostics/revisionRequest";
import type { FocusTree, FocusNode, FocusNodeCreateInput } from "@/types";
import { focusTreesApi } from "@/lib/api";
import { applyFocusTreeDiagnostics } from "@/components/workbench/save-diagnostics/focusTreeDiagnostics";
import {
  createNodeWriteTracker,
  type NodeWriteTicket,
} from "@/components/workbench/save-diagnostics/nodeWriteTracker";

let _previewTimer: ReturnType<typeof setTimeout> | null = null;
let _previewController: AbortController | null = null;
let _previewRequestVersion = 0;
let _treeListController: AbortController | null = null;
let _treeListRequestVersion = 0;

let _dragDebounceTimers: Map<string, ReturnType<typeof setTimeout>> | null =
  null;

let _dragVersions: Map<string, number> | null = null;

const _nodeUpdateVersions = new Map<string, number>();
const _nodeWrites = createNodeWriteTracker<FocusNode>();
const _writeFields = new Map<
  string,
  { generation: number; fields: Set<keyof FocusNode> }
>();
function beginNodeWrite(key: string, before: FocusNode, fields: string[]) {
  const ticket = _nodeWrites.begin(key, before);
  let group = _writeFields.get(key);
  if (!group || group.generation !== ticket.generation) {
    group = { generation: ticket.generation, fields: new Set() };
    _writeFields.set(key, group);
  }
  for (const field of fields) group.fields.add(field as keyof FocusNode);
  return ticket;
}
function rollbackNode(
  current: FocusNode,
  ticket: NodeWriteTicket,
  canonical: FocusNode,
) {
  const restored = { ...current };
  const group = _writeFields.get(ticket.key);
  if (group?.generation === ticket.generation) {
    for (const field of group.fields)
      Object.assign(restored, { [field]: canonical[field] });
  }
  return restored;
}
const _dragTickets = new Map<string, NodeWriteTicket>();
const _projectWriteEpoch = new Map<number, number>();
const _pendingWrites = new Map<number, number>();
const _overlappingWrites = new Set<number>();

function beginWrite(projectId: number) {
  const pending = _pendingWrites.get(projectId) ?? 0;
  if (pending > 0) _overlappingWrites.add(projectId);
  _pendingWrites.set(projectId, pending + 1);
  const epoch = (_projectWriteEpoch.get(projectId) ?? 0) + 1;
  _projectWriteEpoch.set(projectId, epoch);
  return epoch;
}

function finishWrite(projectId: number) {
  const pending = Math.max(0, (_pendingWrites.get(projectId) ?? 1) - 1);
  _pendingWrites.set(projectId, pending);
  const epoch = (_projectWriteEpoch.get(projectId) ?? 0) + 1;
  _projectWriteEpoch.set(projectId, epoch);
  if (pending || !_overlappingWrites.delete(projectId)) return;

  if (typeof focusTreesApi.list !== "function") return;
  void Promise.resolve(focusTreesApi.list(projectId))
    .then((trees) => {
      if (
        _projectWriteEpoch.get(projectId) !== epoch ||
        (_pendingWrites.get(projectId) ?? 0) > 0
      )
        return;
      const byId = new Map(trees.map((tree) => [tree.id, tree]));
      useFocusTreeStore.setState((state) => ({
        trees: state.trees.map((tree) => {
          const assessed = byId.get(tree.id);
          if (!assessed || assessed.project_id !== projectId) return tree;
          const nodes = new Map(assessed.nodes.map((node) => [node.id, node]));
          return {
            ...tree,
            diagnostics: assessed.diagnostics ?? tree.diagnostics,
            nodes: tree.nodes.map((node) => ({
              ...node,
              diagnostics: nodes.get(node.id)?.diagnostics ?? node.diagnostics,
            })),
          };
        }),
      }));
    })
    .catch(() => {});
}

type FocusHistoryEntry =
  | {
      kind: "update_node";
      treeId: number;
      nodeId: number;
      before: Partial<FocusNode>;
      after: Partial<FocusNode>;
    }
  | {
      kind: "create_node";
      treeId: number;
      nodeFocusId: string;
      nodeSnapshot: FocusNode;
    }
  | {
      kind: "delete_node";
      treeId: number;
      nodeSnapshot: FocusNode;
    }
  | {
      kind: "update_node_positions";
      treeId: number;
      entries: Array<{
        nodeId: number;
        before: { x: number; y: number };
        after: { x: number; y: number };
      }>;
    };

const UNDO_STACK_CAP = 60;

interface FocusTreeStore {
  trees: FocusTree[];
  selectedTreeId: number | null;
  selectedNodeId: number | null;

  selectedNodeIds: number[];

  pendingViewportFocusId: number | null;
  selectedEdgeId: string | null;
  scriptPreview: string;
  scriptPreviewSnapshot: FocusPreviewSnapshot | null;
  previewEnabled: boolean;
  loading: boolean;
  error: string | null;
  errorProjectId: number | null;

  historyUndo: FocusHistoryEntry[];
  historyRedo: FocusHistoryEntry[];

  isReplaying: boolean;

  importNonce: number;

  notifyImported: () => void;
  applyUploadedNodeIcon: (treeId: number, nodeId: number, icon: string) => void;

  fetchTrees: (projectId: number) => Promise<void>;
  selectTree: (id: number | null) => void;
  selectNode: (id: number | null) => void;
  clearPendingViewportFocus: () => void;
  selectEdge: (id: string | null) => void;
  setSelectedNodeIds: (ids: number[]) => void;
  createTree: (projectId: number, countryTag?: string) => Promise<FocusTree>;
  updateTree: (
    projectId: number,
    treeId: number,
    data: Partial<FocusTree>,
  ) => Promise<void>;
  deleteTree: (projectId: number, treeId: number) => Promise<void>;

  createNode: (
    projectId: number,
    treeId: number,
    data?: FocusNodeCreateInput,
  ) => Promise<FocusNode | null>;
  updateNode: (
    projectId: number,
    treeId: number,
    nodeId: number,
    data: Partial<FocusNode>,
  ) => Promise<void>;
  deleteNode: (
    projectId: number,
    treeId: number,
    nodeId: number,
  ) => Promise<void>;
  fetchPreview: (
    projectId: number,
    treeId: number,
    nodeId?: number,
  ) => Promise<void>;
  setPreviewEnabled: (enabled: boolean) => void;

  updateNodePosition: (
    projectId: number,
    treeId: number,
    nodeId: number,
    x: number,
    y: number,
  ) => Promise<void>;

  bulkUpdateNodePositions: (
    projectId: number,
    treeId: number,
    positions: Array<{ id: number; x: number; y: number }>,
  ) => Promise<void>;
  addPrerequisite: (
    projectId: number,
    treeId: number,
    nodeId: number,
    prereqFocusId: string,
  ) => Promise<void>;
  removePrerequisite: (
    projectId: number,
    treeId: number,
    nodeId: number,
    prereqFocusId: string,
    groupIndex: number,
  ) => Promise<void>;
  addMutualExclusion: (
    projectId: number,
    treeId: number,
    nodeId: number,
    otherFocusId: string,
  ) => Promise<void>;
  removeMutualExclusion: (
    projectId: number,
    treeId: number,
    nodeId: number,
    otherFocusId: string,
  ) => Promise<void>;

  undo: (projectId: number) => Promise<void>;
  redo: (projectId: number) => Promise<void>;
  duplicateNode: (
    projectId: number,
    treeId: number,
    nodeId: number,
  ) => Promise<void>;
  selectAllNodes: (treeId: number) => void;
  canUndo: () => boolean;
  canRedo: () => boolean;
}

function pushHistory(
  stack: FocusHistoryEntry[],
  entry: FocusHistoryEntry,
): FocusHistoryEntry[] {
  const next = [...stack, entry];
  if (next.length > UNDO_STACK_CAP) next.shift();
  return next;
}

function findNode(
  trees: FocusTree[],
  treeId: number,
  nodeId: number,
): FocusNode | undefined {
  const tree = trees.find((t) => t.id === treeId);
  return tree?.nodes.find((n) => n.id === nodeId);
}

function findNodeByFocusId(
  trees: FocusTree[],
  treeId: number,
  focusId: string,
): FocusNode | undefined {
  const tree = trees.find((t) => t.id === treeId);
  return tree?.nodes.find((n) => n.focus_id === focusId);
}

function activePreviewNodeId(
  state: FocusTreeStore,
  treeId: number,
): number | undefined {
  if (state.selectedTreeId !== treeId) return undefined;
  const nodeId = state.selectedNodeId;
  return nodeId != null && nodeId > 0 ? nodeId : undefined;
}

export const useFocusTreeStore = create<FocusTreeStore>((set, get) => ({
  trees: [],
  selectedTreeId: null,
  selectedNodeId: null,
  selectedNodeIds: [],
  pendingViewportFocusId: null,
  selectedEdgeId: null,
  scriptPreview: "",
  scriptPreviewSnapshot: null,
  previewEnabled: false,
  loading: false,
  error: null,
  errorProjectId: null,
  historyUndo: [],
  historyRedo: [],
  isReplaying: false,
  importNonce: 0,
  notifyImported: () => set((s) => ({ importNonce: s.importNonce + 1 })),
  applyUploadedNodeIcon: (treeId, nodeId, icon) =>
    set((s) => ({
      trees: s.trees.map((tree) =>
        tree.id === treeId
          ? {
              ...tree,
              nodes: tree.nodes.map((node) =>
                node.id === nodeId ? { ...node, icon } : node,
              ),
            }
          : tree,
      ),
    })),

  fetchTrees: async (projectId) => {
    _treeListController?.abort();
    const controller = new AbortController();
    _treeListController = controller;
    const requestVersion = ++_treeListRequestVersion;
    const writeEpoch = _projectWriteEpoch.get(projectId) ?? 0;
    set({ loading: true, error: null });
    try {
      const trees = await focusTreesApi.list(projectId, controller.signal);
      if (requestVersion !== _treeListRequestVersion) return;
      if (
        writeEpoch !== (_projectWriteEpoch.get(projectId) ?? 0) ||
        (_pendingWrites.get(projectId) ?? 0) > 0
      )
        return;
      const currentSelected = get().selectedTreeId;
      const autoSelect =
        trees.length > 0 && !currentSelected ? trees[0].id : currentSelected;
      const selected = get().selectedNodeId;
      set({
        trees,
        selectedTreeId: autoSelect,
        selectedNodeId: trees
          .find((tree) => tree.id === autoSelect)
          ?.nodes.some((node) => node.id === selected)
          ? selected
          : null,
      });
    } catch (err) {
      if (
        controller.signal.aborted ||
        requestVersion !== _treeListRequestVersion
      )
        return;
      set({
        error: (err as Error).message || "Failed to load focus trees",
        errorProjectId: projectId,
      });
    } finally {
      if (requestVersion === _treeListRequestVersion) set({ loading: false });
      if (_treeListController === controller) _treeListController = null;
    }
  },

  setPreviewEnabled: (enabled) => {
    if (!enabled) {
      if (_previewTimer) clearTimeout(_previewTimer);
      _previewTimer = null;
      _previewController?.abort();
      _previewController = null;
      _previewRequestVersion += 1;
    }
    set({ previewEnabled: enabled });
  },

  selectTree: (id) => {
    set({
      selectedTreeId: id,
      selectedNodeId: null,
      selectedNodeIds: [],
      selectedEdgeId: null,
      scriptPreview: "",
      scriptPreviewSnapshot: null,
      historyUndo: [],
      historyRedo: [],
    });
  },

  selectNode: (id) =>
    set({
      selectedNodeId: id,
      selectedNodeIds: id !== null ? [id] : [],
      selectedEdgeId: null,
    }),
  clearPendingViewportFocus: () => set({ pendingViewportFocusId: null }),

  selectEdge: (id) =>
    set(
      id === null
        ? { selectedEdgeId: null }
        : { selectedEdgeId: id, selectedNodeId: null, selectedNodeIds: [] },
    ),

  setSelectedNodeIds: (ids) =>
    set((s) => ({
      selectedNodeIds: ids,
      selectedNodeId: ids.length === 1 ? ids[0] : null,
      selectedEdgeId: ids.length > 0 ? null : s.selectedEdgeId,
    })),

  createTree: async (projectId, countryTag?) => {
    let tree;
    try {
      tree = await focusTreesApi.create(
        projectId,
        countryTag ? { country_tag: countryTag } : undefined,
      );
    } catch (err) {
      const msg = (err as Error).message || "";
      if (msg.includes("already exists") || msg.includes("race-collision")) {
        return null as unknown as FocusTree;
      }
      throw err;
    }
    set((s) => ({
      trees: [...s.trees, tree],
      selectedTreeId: tree.id,
      selectedNodeId: null,
    }));
    return tree;
  },

  updateTree: async (projectId, treeId, data) => {
    const updated = await focusTreesApi.update(
      projectId,
      treeId,
      data,
      "edit_revision" in data
        ? data.edit_revision
        : get().trees.find((tree) => tree.id === treeId)?.edit_revision,
    );
    set((s) => ({
      trees: s.trees.map((t) => (t.id === treeId ? updated : t)),
    }));
    if (get().selectedTreeId === treeId) {
      get().fetchPreview(projectId, treeId, activePreviewNodeId(get(), treeId));
    }
  },

  deleteTree: async (projectId, treeId) => {
    await focusTreesApi.delete(
      projectId,
      treeId,
      get().trees.find((tree) => tree.id === treeId)?.edit_revision,
    );
    set((s) => ({
      trees: s.trees.filter((t) => t.id !== treeId),
      selectedTreeId: s.selectedTreeId === treeId ? null : s.selectedTreeId,
      selectedNodeId: s.selectedTreeId === treeId ? null : s.selectedNodeId,
      scriptPreview: s.selectedTreeId === treeId ? "" : s.scriptPreview,
      scriptPreviewSnapshot:
        s.selectedTreeId === treeId ? null : s.scriptPreviewSnapshot,
    }));
  },

  createNode: async (projectId, treeId, data) => {
    const tree = get().trees.find((t) => t.id === treeId);
    if (
      data?.focus_id &&
      tree?.nodes.find((n) => n.focus_id === data.focus_id)
    ) {
      return null;
    }

    const tempId = -Date.now();
    const tempNode: FocusNode = {
      id: tempId,
      focus_tree_id: treeId,
      focus_id: data?.focus_id || "new_focus",
      name: data?.name_en || "",
      name_en: data?.name_en || "",
      name_zh: data?.name_zh || "",
      x: data?.x ?? 0,
      y: data?.y ?? 0,
      icon: data?.icon || "",
      cost: data?.cost ?? 0,
      available: data?.available || "",
      completion_reward: data?.completion_reward || "",
      bypass: data?.bypass || "",
      cancel: data?.cancel || "",
      select_effect: "",
      ai_will_do: data?.ai_will_do || "",
      search_filters: [],
      cancel_if_invalid: true,
      continue_if_invalid: false,
      available_if_capitulated: false,
      will_lead_to_war_with: [],
      prerequisites: data?.prerequisites || [],
      mutually_exclusive: data?.mutually_exclusive || [],
      allow_branch_ast: data?.allow_branch_ast || {},
      load_focus_tree_rules: [],
      relative_position_id: data?.relative_position_id || "",
      raw_extras: data?.raw_extras || {},
      desc_en: "",
      desc_zh: "",
      display_name_locs: {},
      description_locs: {},
      effect_tooltip_locs: {},
    };

    set((s) => {
      const next: Partial<FocusTreeStore> = {
        trees: s.trees.map((t) =>
          t.id === treeId ? { ...t, nodes: [...t.nodes, tempNode] } : t,
        ),
        selectedNodeId: tempId,
        selectedNodeIds: [tempId],
        selectedEdgeId: null,

        pendingViewportFocusId: tempId,
      };

      if (!s.isReplaying) {
        next.historyUndo = pushHistory(s.historyUndo, {
          kind: "create_node",
          treeId,
          nodeFocusId: tempNode.focus_id,
          nodeSnapshot: tempNode,
        });
        next.historyRedo = [];
      }
      return next as FocusTreeStore;
    });

    try {
      const createdNode = await focusTreesApi.createNode(
        projectId,
        treeId,
        data,
        tree?.edit_revision,
      );

      set((s) => {
        const next: Partial<FocusTreeStore> = {
          trees: s.trees.map((t) =>
            t.id === treeId
              ? {
                  ...t,
                  nodes: t.nodes.map((n) =>
                    n.id === tempId ? createdNode : n,
                  ),
                }
              : t,
          ),
          selectedNodeId: createdNode.id,
          selectedNodeIds: [createdNode.id],
          selectedEdgeId: null,

          pendingViewportFocusId: createdNode.id,
        };

        if (!s.isReplaying && s.historyUndo.length > 0) {
          const lastEntry = s.historyUndo[s.historyUndo.length - 1];
          if (
            lastEntry.kind === "create_node" &&
            lastEntry.nodeFocusId === createdNode.focus_id
          ) {
            next.historyUndo = [
              ...s.historyUndo.slice(0, -1),
              {
                kind: "create_node",
                treeId,
                nodeFocusId: createdNode.focus_id,
                nodeSnapshot: createdNode,
              },
            ];
          }
        }
        return next as FocusTreeStore;
      });

      setTimeout(() => {
        get().fetchPreview(
          projectId,
          treeId,
          activePreviewNodeId(get(), treeId),
        );
      }, 800);
      return createdNode;
    } catch (err) {
      const msg = (err as Error).message || "";

      const status = (err as { status?: number }).status;
      if (
        status === 409 ||
        msg.includes("already exists") ||
        msg.includes("race-collision")
      ) {
        set((s) => {
          const next: Partial<FocusTreeStore> = {
            trees: s.trees.map((t) =>
              t.id === treeId
                ? { ...t, nodes: t.nodes.filter((n) => n.id !== tempId) }
                : t,
            ),
            selectedNodeId:
              s.selectedNodeId === tempId ? null : s.selectedNodeId,
            selectedNodeIds: s.selectedNodeIds.filter((id) => id !== tempId),
          };

          if (!s.isReplaying && s.historyUndo.length > 0) {
            const lastEntry = s.historyUndo[s.historyUndo.length - 1];
            if (
              lastEntry.kind === "create_node" &&
              lastEntry.nodeSnapshot.id === tempId
            ) {
              next.historyUndo = s.historyUndo.slice(0, -1);
            }
          }
          return next as FocusTreeStore;
        });
        return null;
      }

      set((s) => {
        const next: Partial<FocusTreeStore> = {
          trees: s.trees.map((t) =>
            t.id === treeId
              ? { ...t, nodes: t.nodes.filter((n) => n.id !== tempId) }
              : t,
          ),
          selectedNodeId: s.selectedNodeId === tempId ? null : s.selectedNodeId,
          selectedNodeIds: s.selectedNodeIds.filter((id) => id !== tempId),
        };

        if (!s.isReplaying && s.historyUndo.length > 0) {
          const lastEntry = s.historyUndo[s.historyUndo.length - 1];
          if (
            lastEntry.kind === "create_node" &&
            lastEntry.nodeSnapshot.id === tempId
          ) {
            next.historyUndo = s.historyUndo.slice(0, -1);
          }
        }
        return next as FocusTreeStore;
      });
      set({
        error: err instanceof Error ? err.message : String(err),
        errorProjectId: projectId,
      });
      console.error("[createNode] persist failed:", err);
      throw err;
    }
  },

  updateNode: async (projectId, treeId, nodeId, data) => {
    const writeEpoch = beginWrite(projectId);
    const updateKey = `${projectId}:${treeId}:${nodeId}`;
    const updateVersion = (_nodeUpdateVersions.get(updateKey) ?? 0) + 1;
    _nodeUpdateVersions.set(updateKey, updateVersion);

    const beforeNode = findNode(get().trees, treeId, nodeId);
    const continuation = !("tree_edit_revision" in data);
    const inputRevision =
      "tree_edit_revision" in data
        ? data.tree_edit_revision
        : beforeNode?.tree_edit_revision;
    data = withoutEditRevisions(data);
    const ticket = beforeNode
      ? beginNodeWrite(updateKey, beforeNode, Object.keys(data))
      : null;
    set({ error: null });
    const before: Partial<FocusNode> = {};
    if (beforeNode) {
      for (const key of Object.keys(data)) {
        (before as any)[key] = (beforeNode as any)[key];
      }
    }

    const historyEntry: FocusHistoryEntry | null = beforeNode
      ? { kind: "update_node", treeId, nodeId, before, after: data }
      : null;

    set((s) => {
      const next: Partial<FocusTreeStore> = {
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) =>
                  n.id === nodeId ? { ...n, ...data } : n,
                ),
              }
            : t,
        ),
      };
      if (!s.isReplaying && historyEntry) {
        next.historyUndo = pushHistory(s.historyUndo, historyEntry);
        next.historyRedo = [];
      }
      return next as FocusTreeStore;
    });

    try {
      const updated = await focusTreesApi.updateNode(
        projectId,
        treeId,
        nodeId,
        data,
        inputRevision,
        continuation,
      );
      const settled = ticket
        ? _nodeWrites.settle(ticket, { ok: true, node: updated })
        : null;
      if (settled && !settled.apply) return;
      const canonical = settled?.apply ? settled.canonical : updated;

      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? applyFocusTreeDiagnostics(
                {
                  ...t,
                  nodes: t.nodes.map((n) =>
                    n.id === nodeId ? { ...n, ...canonical } : n,
                  ),
                },
                _projectWriteEpoch.get(projectId) === writeEpoch
                  ? canonical
                  : { ...canonical, tree_diagnostics: undefined },
              )
            : t,
        ),
      }));

      if (get().selectedTreeId === treeId) {
        get().fetchPreview(
          projectId,
          treeId,
          activePreviewNodeId(get(), treeId),
        );
      }
    } catch (err) {
      const settled = ticket ? _nodeWrites.settle(ticket, { ok: false }) : null;

      set((s) => ({
        historyUndo: s.historyUndo.filter((entry) => entry !== historyEntry),
      }));

      if (settled?.apply) {
        set((s) => ({
          trees: s.trees.map((t) =>
            t.id === treeId
              ? {
                  ...t,
                  nodes: t.nodes.map((n) =>
                    n.id === nodeId && ticket
                      ? rollbackNode(n, ticket, settled.canonical)
                      : n,
                  ),
                }
              : t,
          ),
        }));
      }
      if (_nodeUpdateVersions.get(updateKey) !== updateVersion) throw err;
      console.error("[updateNode] persist failed:", err);
      set({
        error: err instanceof Error ? err.message : String(err),
        errorProjectId: projectId,
      });
      throw err;
    } finally {
      finishWrite(projectId);
    }
  },

  deleteNode: async (projectId, treeId, nodeId) => {
    const snapshot = findNode(get().trees, treeId, nodeId);

    set((s) => {
      const next: Partial<FocusTreeStore> = {
        trees: s.trees.map((t) =>
          t.id === treeId
            ? { ...t, nodes: t.nodes.filter((n) => n.id !== nodeId) }
            : t,
        ),
        selectedNodeId: s.selectedNodeId === nodeId ? null : s.selectedNodeId,
        selectedNodeIds: s.selectedNodeIds.filter((id) => id !== nodeId),
      };
      if (!s.isReplaying && snapshot) {
        next.historyUndo = pushHistory(s.historyUndo, {
          kind: "delete_node",
          treeId,
          nodeSnapshot: snapshot,
        });
        next.historyRedo = [];
      }
      return next as FocusTreeStore;
    });

    try {
      await focusTreesApi.deleteNode(
        projectId,
        treeId,
        nodeId,
        get().trees.find((tree) => tree.id === treeId)?.edit_revision,
      );

      get().fetchPreview(projectId, treeId, activePreviewNodeId(get(), treeId));
    } catch (err) {
      set((s) => {
        const next: Partial<FocusTreeStore> = {
          trees: s.trees.map((t) =>
            t.id === treeId
              ? {
                  ...t,
                  nodes: snapshot ? [...t.nodes, snapshot] : t.nodes,
                }
              : t,
          ),
        };
        if (snapshot) {
          next.selectedNodeIds = [...s.selectedNodeIds, snapshot.id];
          next.selectedNodeId =
            s.selectedNodeIds.length === 0 ? snapshot.id : null;
        }
        return next as FocusTreeStore;
      });
      set({
        error: err instanceof Error ? err.message : String(err),
        errorProjectId: projectId,
      });
      console.error("[deleteNode] persist failed:", err);
    }
  },

  fetchPreview: (projectId, treeId, nodeId?) => {
    if (!get().previewEnabled) return Promise.resolve();

    if (nodeId !== undefined && nodeId <= 0) {
      if (_previewTimer) clearTimeout(_previewTimer);
      _previewTimer = null;
      _previewController?.abort();
      _previewController = null;
      _previewRequestVersion += 1;
      set({
        scriptPreview: "# Save this focus before previewing it",
        scriptPreviewSnapshot: null,
      });
      return Promise.resolve();
    }

    if (_previewTimer) clearTimeout(_previewTimer);

    _previewController?.abort();
    _previewController = null;
    const requestVersion = ++_previewRequestVersion;
    _previewTimer = setTimeout(async () => {
      _previewTimer = null;
      const controller = new AbortController();
      _previewController = controller;
      try {
        const res = await focusTreesApi.preview(
          projectId,
          treeId,
          nodeId,
          controller.signal,
        );
        if (requestVersion === _previewRequestVersion) {
          set({
            scriptPreview: res.script,
            scriptPreviewSnapshot: captureFocusPreview(
              res,
              projectId,
              treeId,
              nodeId,
            ),
          });
        }
      } catch {
        if (
          controller.signal.aborted ||
          requestVersion !== _previewRequestVersion
        )
          return;
        set({
          scriptPreview: "# Preview unavailable",
          scriptPreviewSnapshot: null,
        });
      } finally {
        if (_previewController === controller) _previewController = null;
      }
    }, 250);
    return Promise.resolve();
  },

  bulkUpdateNodePositions: async (projectId, treeId, positions) => {
    if (positions.length === 0) return;
    const tree = get().trees.find((t) => t.id === treeId);
    if (!tree) return;
    const writeEpoch = beginWrite(projectId);
    set({ error: null });
    const tickets = new Map(
      positions.flatMap((p) => {
        const row = tree.nodes.find((node) => node.id === p.id);
        return row
          ? [
              [
                p.id,
                beginNodeWrite(`${projectId}:${treeId}:${p.id}`, row, [
                  "x",
                  "y",
                ]),
              ] as const,
            ]
          : [];
      }),
    );

    const wanted = new Map(positions.map((p) => [p.id, p]));
    const entries: Array<{
      nodeId: number;
      before: { x: number; y: number };
      after: { x: number; y: number };
    }> = [];
    for (const n of tree.nodes) {
      const p = wanted.get(n.id);
      if (!p) continue;
      if (n.x === p.x && n.y === p.y) continue;
      entries.push({
        nodeId: n.id,
        before: { x: n.x, y: n.y },
        after: { x: p.x, y: p.y },
      });
    }

    const versions = new Map(
      positions.map((p) => {
        const key = `${projectId}:${treeId}:${p.id}`;
        const version = (_nodeUpdateVersions.get(key) ?? 0) + 1;
        _nodeUpdateVersions.set(key, version);
        return [p.id, version];
      }),
    );
    const owns = (id: number) =>
      versions.has(id) &&
      _nodeUpdateVersions.get(`${projectId}:${treeId}:${id}`) ===
        versions.get(id);

    set((s) => ({
      trees: s.trees.map((t) =>
        t.id === treeId
          ? {
              ...t,
              nodes: t.nodes.map((n) => {
                const p = wanted.get(n.id);
                return p ? { ...n, x: p.x, y: p.y } : n;
              }),
            }
          : t,
      ),
    }));

    try {
      const updated = await focusTreesApi.bulkUpdateNodePositions(
        projectId,
        treeId,
        positions,
        tree?.edit_revision,
        true,
      );
      const byId = new Map(updated.map((n) => [n.id, n]));
      const settledRows = new Map<number, FocusNode>();
      for (const [id, ticket] of tickets) {
        const row = byId.get(id);
        const settled = _nodeWrites.settle(
          ticket,
          row ? { ok: true, node: row } : { ok: false },
        );
        if (settled.apply) settledRows.set(id, settled.canonical);
      }
      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) =>
                  settledRows.has(n.id)
                    ? { ...n, ...settledRows.get(n.id) }
                    : n,
                ),
              }
            : t,
        ),
      }));
      if (_projectWriteEpoch.get(projectId) === writeEpoch && updated[0]) {
        set((s) => ({
          trees: s.trees.map((t) =>
            t.id === treeId ? applyFocusTreeDiagnostics(t, updated[0]) : t,
          ),
        }));
      }
      if (entries.length > 0) {
        set((s) =>
          s.isReplaying
            ? {}
            : {
                historyUndo: pushHistory(s.historyUndo, {
                  kind: "update_node_positions",
                  treeId,
                  entries: entries.filter((entry) => owns(entry.nodeId)),
                }),
                historyRedo: [],
              },
        );
      }
      get().fetchPreview(projectId, treeId, activePreviewNodeId(get(), treeId));
    } catch (err) {
      const beforeById = new Map<number, FocusNode>();
      for (const [id, ticket] of tickets) {
        const settled = _nodeWrites.settle(ticket, { ok: false });
        if (settled.apply) beforeById.set(id, settled.canonical);
      }
      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) => {
                  const prior = beforeById.get(n.id);
                  const ticket = tickets.get(n.id);
                  return prior && ticket ? rollbackNode(n, ticket, prior) : n;
                }),
              }
            : t,
        ),
      }));
      set({
        error: err instanceof Error ? err.message : String(err),
        errorProjectId: projectId,
      });
      console.error("[bulkUpdateNodePositions] persist failed:", err);
    } finally {
      finishWrite(projectId);
    }
  },

  updateNodePosition: (projectId, treeId, nodeId, x, y) => {
    const beforeNode = findNode(get().trees, treeId, nodeId);
    if (!beforeNode) return Promise.resolve();
    const writeEpoch = beginWrite(projectId);
    set({ error: null });
    const timerKey = `${projectId}:${treeId}:${nodeId}`;
    const ticket = beginNodeWrite(timerKey, beforeNode, ["x", "y"]);
    const previousTicket = _dragTickets.get(timerKey);
    if (previousTicket && _dragDebounceTimers?.has(timerKey)) {
      _nodeWrites.settle(previousTicket, { ok: false });
      finishWrite(projectId);
    }
    _dragTickets.set(timerKey, ticket);
    const beforeX = beforeNode?.x ?? 0;
    const beforeY = beforeNode?.y ?? 0;
    const positionChanged =
      beforeNode != null && (beforeX !== x || beforeY !== y);

    set((s) => ({
      trees: s.trees.map((t) =>
        t.id === treeId
          ? {
              ...t,
              nodes: t.nodes.map((n) => (n.id === nodeId ? { ...n, x, y } : n)),
            }
          : t,
      ),
    }));

    if (!_dragDebounceTimers) _dragDebounceTimers = new Map();
    if (!_dragVersions) _dragVersions = new Map();
    if (_dragDebounceTimers.has(timerKey)) {
      clearTimeout(_dragDebounceTimers.get(timerKey));
    }

    const currentVersion = (_dragVersions.get(timerKey) ?? 0) + 1;
    _dragVersions.set(timerKey, currentVersion);
    const updateVersion = (_nodeUpdateVersions.get(timerKey) ?? 0) + 1;
    _nodeUpdateVersions.set(timerKey, updateVersion);

    const timer = setTimeout(async () => {
      _dragDebounceTimers?.delete(timerKey);
      if (_nodeUpdateVersions.get(timerKey) !== updateVersion) {
        const settled = _nodeWrites.settle(ticket, { ok: false });
        if (settled.apply)
          set((s) => ({
            trees: s.trees.map((t) =>
              t.id === treeId
                ? {
                    ...t,
                    nodes: t.nodes.map((n) =>
                      n.id === nodeId
                        ? rollbackNode(n, ticket, settled.canonical)
                        : n,
                    ),
                  }
                : t,
            ),
          }));
        finishWrite(projectId);
        return;
      }

      try {
        const updated = await focusTreesApi.updateNode(
          projectId,
          treeId,
          nodeId,
          { x, y },
          beforeNode.tree_edit_revision,
          true,
        );
        const settled = _nodeWrites.settle(ticket, { ok: true, node: updated });

        if (!settled.apply) {
          return;
        }

        set((s) => {
          const next: Partial<FocusTreeStore> = {
            trees: s.trees.map((t) =>
              t.id === treeId
                ? applyFocusTreeDiagnostics(
                    {
                      ...t,
                      nodes: t.nodes.map((n) =>
                        n.id === nodeId ? { ...n, ...settled.canonical } : n,
                      ),
                    },
                    _projectWriteEpoch.get(projectId) === writeEpoch
                      ? settled.canonical
                      : { ...settled.canonical, tree_diagnostics: undefined },
                  )
                : t,
            ),
          };
          if (
            !s.isReplaying &&
            positionChanged &&
            _nodeUpdateVersions.get(timerKey) === updateVersion
          ) {
            next.historyUndo = pushHistory(s.historyUndo, {
              kind: "update_node",
              treeId,
              nodeId,
              before: { x: beforeX, y: beforeY },
              after: { x, y },
            });
            next.historyRedo = [];
          }
          return next as FocusTreeStore;
        });
      } catch (err) {
        const settled = _nodeWrites.settle(ticket, { ok: false });

        if (settled.apply)
          set((s) => ({
            trees: s.trees.map((t) =>
              t.id === treeId
                ? {
                    ...t,
                    nodes: t.nodes.map((n) =>
                      n.id === nodeId
                        ? rollbackNode(n, ticket, settled.canonical)
                        : n,
                    ),
                  }
                : t,
            ),
          }));
        if (_nodeUpdateVersions.get(timerKey) !== updateVersion) return;
        console.error("[focus drag] persist failed:", err);
        set({
          error: err instanceof Error ? err.message : String(err),
          errorProjectId: projectId,
        });
      } finally {
        if (_dragTickets.get(timerKey) === ticket)
          _dragTickets.delete(timerKey);
        finishWrite(projectId);
      }
    }, 250);

    _dragDebounceTimers.set(timerKey, timer);
    return Promise.resolve();
  },

  addPrerequisite: async (projectId, treeId, nodeId, prereqFocusId) => {
    const node = findNode(get().trees, treeId, nodeId);
    if (!node) return;

    if (prereqFocusId === node.focus_id) return;

    const current = node.prerequisites || [];

    const updated = [...current, [prereqFocusId]];

    set((s) => ({
      trees: s.trees.map((t) =>
        t.id === treeId
          ? {
              ...t,
              nodes: t.nodes.map((n) =>
                n.id === nodeId ? { ...n, prerequisites: updated } : n,
              ),
            }
          : t,
      ),
    }));

    try {
      const result = await focusTreesApi.updateNode(
        projectId,
        treeId,
        nodeId,
        {
          prerequisites: updated,
        },
        node.tree_edit_revision,
      );

      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) => (n.id === nodeId ? result : n)),
              }
            : t,
        ),
      }));
      get().fetchPreview(projectId, treeId, activePreviewNodeId(get(), treeId));
    } catch (err) {
      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) =>
                  n.id === nodeId ? { ...n, prerequisites: current } : n,
                ),
              }
            : t,
        ),
      }));
      set({
        error: err instanceof Error ? err.message : String(err),
        errorProjectId: projectId,
      });
      console.error("[addPrerequisite] persist failed:", err);
    }
  },

  removePrerequisite: async (
    projectId,
    treeId,
    nodeId,
    prereqFocusId,
    groupIndex,
  ) => {
    const node = findNode(get().trees, treeId, nodeId);
    if (!node) return;

    const current = (node.prerequisites || []).map((g) => [...g]);
    if (groupIndex < 0 || groupIndex >= current.length) return;

    current[groupIndex] = current[groupIndex].filter(
      (f) => f !== prereqFocusId,
    );

    const updated = current.filter((g) => g.length > 0);

    set((s) => ({
      trees: s.trees.map((t) =>
        t.id === treeId
          ? {
              ...t,
              nodes: t.nodes.map((n) =>
                n.id === nodeId ? { ...n, prerequisites: updated } : n,
              ),
            }
          : t,
      ),
    }));

    try {
      const result = await focusTreesApi.updateNode(
        projectId,
        treeId,
        nodeId,
        {
          prerequisites: updated,
        },
        node.tree_edit_revision,
      );

      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) => (n.id === nodeId ? result : n)),
              }
            : t,
        ),
      }));
      get().fetchPreview(projectId, treeId, activePreviewNodeId(get(), treeId));
    } catch (err) {
      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) =>
                  n.id === nodeId
                    ? {
                        ...n,
                        prerequisites: current.filter((g) => g.length > 0),
                      }
                    : n,
                ),
              }
            : t,
        ),
      }));
      set({
        error: err instanceof Error ? err.message : String(err),
        errorProjectId: projectId,
      });
      console.error("[removePrerequisite] persist failed:", err);
    }
  },

  addMutualExclusion: async (projectId, treeId, nodeId, otherFocusId) => {
    const node = findNode(get().trees, treeId, nodeId);
    const otherNode = findNodeByFocusId(get().trees, treeId, otherFocusId);
    if (!node || !otherNode) return;

    if (otherFocusId === node.focus_id) return;

    const myMutex = node.mutually_exclusive || [];
    const otherMutex = otherNode.mutually_exclusive || [];

    if (myMutex.includes(otherFocusId)) return;

    const newMyMutex = [...myMutex, otherFocusId];
    const newOtherMutex = [...otherMutex, node.focus_id];

    set((s) => ({
      trees: s.trees.map((t) =>
        t.id === treeId
          ? {
              ...t,
              nodes: t.nodes.map((n) => {
                if (n.id === nodeId)
                  return { ...n, mutually_exclusive: newMyMutex };
                if (n.id === otherNode.id)
                  return { ...n, mutually_exclusive: newOtherMutex };
                return n;
              }),
            }
          : t,
      ),
    }));

    try {
      const updated = await focusTreesApi.setMutualExclusion(
        projectId,
        treeId,
        nodeId,
        otherNode.id,
        true,
        node.tree_edit_revision,
      );
      const updatedNode = updated.find((row) => row.id === nodeId)!;
      const updatedOther = updated.find((row) => row.id === otherNode.id)!;

      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) => {
                  if (n.id === nodeId) return updatedNode;
                  if (n.id === otherNode.id) return updatedOther;
                  return n;
                }),
              }
            : t,
        ),
      }));

      get().fetchPreview(projectId, treeId, activePreviewNodeId(get(), treeId));
    } catch (err) {
      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) => {
                  if (n.id === nodeId)
                    return { ...n, mutually_exclusive: myMutex };
                  if (n.id === otherNode.id)
                    return { ...n, mutually_exclusive: otherMutex };
                  return n;
                }),
              }
            : t,
        ),
      }));
      set({
        error: err instanceof Error ? err.message : String(err),
        errorProjectId: projectId,
      });
      console.error("[addMutualExclusion] persist failed:", err);
    }
  },

  removeMutualExclusion: async (projectId, treeId, nodeId, otherFocusId) => {
    const node = findNode(get().trees, treeId, nodeId);
    const otherNode = findNodeByFocusId(get().trees, treeId, otherFocusId);
    if (!node || !otherNode) return;

    const myMutex = node.mutually_exclusive || [];
    const otherMutex = otherNode.mutually_exclusive || [];

    const newMyMutex = myMutex.filter((f) => f !== otherFocusId);
    const newOtherMutex = otherMutex.filter((f) => f !== node.focus_id);

    set((s) => ({
      trees: s.trees.map((t) =>
        t.id === treeId
          ? {
              ...t,
              nodes: t.nodes.map((n) => {
                if (n.id === nodeId)
                  return { ...n, mutually_exclusive: newMyMutex };
                if (n.id === otherNode.id)
                  return { ...n, mutually_exclusive: newOtherMutex };
                return n;
              }),
            }
          : t,
      ),
    }));

    try {
      const updated = await focusTreesApi.setMutualExclusion(
        projectId,
        treeId,
        nodeId,
        otherNode.id,
        false,
        node.tree_edit_revision,
      );
      const updatedNode = updated.find((row) => row.id === nodeId)!;
      const updatedOther = updated.find((row) => row.id === otherNode.id)!;

      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) => {
                  if (n.id === nodeId) return updatedNode;
                  if (n.id === otherNode.id) return updatedOther;
                  return n;
                }),
              }
            : t,
        ),
      }));

      get().fetchPreview(projectId, treeId, activePreviewNodeId(get(), treeId));
    } catch (err) {
      set((s) => ({
        trees: s.trees.map((t) =>
          t.id === treeId
            ? {
                ...t,
                nodes: t.nodes.map((n) => {
                  if (n.id === nodeId)
                    return { ...n, mutually_exclusive: myMutex };
                  if (n.id === otherNode.id)
                    return { ...n, mutually_exclusive: otherMutex };
                  return n;
                }),
              }
            : t,
        ),
      }));
      set({
        error: err instanceof Error ? err.message : String(err),
        errorProjectId: projectId,
      });
      console.error("[removeMutualExclusion] persist failed:", err);
    }
  },

  canUndo: () => get().historyUndo.length > 0,
  canRedo: () => get().historyRedo.length > 0,

  undo: async (projectId) => {
    const state = get();
    if (state.historyUndo.length === 0) return;
    const entry = state.historyUndo[state.historyUndo.length - 1];
    const revision = state.trees.find(
      (tree) => tree.id === entry.treeId,
    )?.edit_revision;
    set({
      historyUndo: state.historyUndo.slice(0, -1),
      isReplaying: true,
    });
    try {
      if (entry.kind === "update_node_positions") {
        const positions = entry.entries.map((e) => ({
          nodeId: e.nodeId,
          ...e.before,
        }));
        await focusTreesApi.bulkUpdateNodePositions(
          projectId,
          entry.treeId,
          positions.map((p) => ({ id: p.nodeId, x: p.x, y: p.y })),
          revision,
        );
        const byId = new Map(positions.map((p) => [p.nodeId, p]));
        set((s) => ({
          trees: s.trees.map((t) =>
            t.id === entry.treeId
              ? {
                  ...t,
                  nodes: t.nodes.map((n) => {
                    const p = byId.get(n.id);
                    return p ? { ...n, x: p.x, y: p.y } : n;
                  }),
                }
              : t,
          ),

          historyRedo: pushHistory(s.historyRedo, entry),
        }));
      } else if (entry.kind === "update_node") {
        await focusTreesApi.updateNode(
          projectId,
          entry.treeId,
          entry.nodeId,
          entry.before,
          revision,
        );

        set((s) => ({
          trees: s.trees.map((t) =>
            t.id === entry.treeId
              ? {
                  ...t,
                  nodes: t.nodes.map((n) =>
                    n.id === entry.nodeId ? { ...n, ...entry.before } : n,
                  ),
                }
              : t,
          ),
          historyRedo: pushHistory(s.historyRedo, entry),
        }));
      } else if (entry.kind === "create_node") {
        const current = findNodeByFocusId(
          get().trees,
          entry.treeId,
          entry.nodeFocusId,
        );
        if (current) {
          await focusTreesApi.deleteNode(
            projectId,
            entry.treeId,
            current.id,
            revision,
          );
          set((s) => ({
            trees: s.trees.map((t) =>
              t.id === entry.treeId
                ? { ...t, nodes: t.nodes.filter((n) => n.id !== current.id) }
                : t,
            ),
            selectedNodeId:
              s.selectedNodeId === current.id ? null : s.selectedNodeId,
            selectedNodeIds: s.selectedNodeIds.filter(
              (id) => id !== current.id,
            ),
            historyRedo: pushHistory(s.historyRedo, entry),
          }));
        }
      } else if (entry.kind === "delete_node") {
        const snap = entry.nodeSnapshot;

        const recreated = await focusTreesApi.createNode(
          projectId,
          entry.treeId,
          {
            focus_id: snap.focus_id,
            name_en: snap.name_en,
            name_zh: snap.name_zh,
            icon: snap.icon,
            x: snap.x,
            y: snap.y,
            cost: snap.cost,
            available: snap.available,
            completion_reward: snap.completion_reward,
            bypass: snap.bypass,
            cancel: snap.cancel,
            ai_will_do: snap.ai_will_do,
            prerequisites: snap.prerequisites,
            mutually_exclusive: snap.mutually_exclusive,
            allow_branch_ast: snap.allow_branch_ast,
            relative_position_id: snap.relative_position_id,
            raw_extras: snap.raw_extras,
          },
          revision,
        );
        set((s) => ({
          trees: s.trees.map((t) =>
            t.id === entry.treeId
              ? { ...t, nodes: [...t.nodes, recreated] }
              : t,
          ),

          historyRedo: pushHistory(s.historyRedo, {
            kind: "delete_node",
            treeId: entry.treeId,
            nodeSnapshot: recreated,
          }),
        }));
      }
      get().fetchPreview(projectId, entry.treeId);
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        errorProjectId: projectId,
      });
      console.error("[focus undo] failed:", err);

      set((s) => ({ historyUndo: pushHistory(s.historyUndo, entry) }));
    } finally {
      set({ isReplaying: false });
    }
  },

  redo: async (projectId) => {
    const state = get();
    if (state.historyRedo.length === 0) return;
    const entry = state.historyRedo[state.historyRedo.length - 1];
    const revision = state.trees.find(
      (tree) => tree.id === entry.treeId,
    )?.edit_revision;
    set({
      historyRedo: state.historyRedo.slice(0, -1),
      isReplaying: true,
    });
    try {
      if (entry.kind === "update_node_positions") {
        const positions = entry.entries.map((e) => ({
          nodeId: e.nodeId,
          ...e.after,
        }));
        await focusTreesApi.bulkUpdateNodePositions(
          projectId,
          entry.treeId,
          positions.map((p) => ({ id: p.nodeId, x: p.x, y: p.y })),
          revision,
        );
        const byId = new Map(positions.map((p) => [p.nodeId, p]));
        set((s) => ({
          trees: s.trees.map((t) =>
            t.id === entry.treeId
              ? {
                  ...t,
                  nodes: t.nodes.map((n) => {
                    const p = byId.get(n.id);
                    return p ? { ...n, x: p.x, y: p.y } : n;
                  }),
                }
              : t,
          ),
          historyUndo: pushHistory(s.historyUndo, entry),
        }));
      } else if (entry.kind === "update_node") {
        await focusTreesApi.updateNode(
          projectId,
          entry.treeId,
          entry.nodeId,
          entry.after,
          revision,
        );
        set((s) => ({
          trees: s.trees.map((t) =>
            t.id === entry.treeId
              ? {
                  ...t,
                  nodes: t.nodes.map((n) =>
                    n.id === entry.nodeId ? { ...n, ...entry.after } : n,
                  ),
                }
              : t,
          ),
          historyUndo: pushHistory(s.historyUndo, entry),
        }));
      } else if (entry.kind === "create_node") {
        const snap = entry.nodeSnapshot;
        const recreated = await focusTreesApi.createNode(
          projectId,
          entry.treeId,
          {
            focus_id: snap.focus_id,
            name_en: snap.name_en,
            name_zh: snap.name_zh,
            icon: snap.icon,
            x: snap.x,
            y: snap.y,
            cost: snap.cost,
            available: snap.available,
            completion_reward: snap.completion_reward,
            bypass: snap.bypass,
            cancel: snap.cancel,
            ai_will_do: snap.ai_will_do,
            prerequisites: snap.prerequisites,
            mutually_exclusive: snap.mutually_exclusive,
            allow_branch_ast: snap.allow_branch_ast,
            relative_position_id: snap.relative_position_id,
            raw_extras: snap.raw_extras,
          },
          revision,
        );
        set((s) => ({
          trees: s.trees.map((t) =>
            t.id === entry.treeId
              ? { ...t, nodes: [...t.nodes, recreated] }
              : t,
          ),
          historyUndo: pushHistory(s.historyUndo, {
            kind: "create_node",
            treeId: entry.treeId,
            nodeFocusId: recreated.focus_id,
            nodeSnapshot: recreated,
          }),
        }));
      } else if (entry.kind === "delete_node") {
        const current = findNodeByFocusId(
          get().trees,
          entry.treeId,
          entry.nodeSnapshot.focus_id,
        );
        if (current) {
          await focusTreesApi.deleteNode(
            projectId,
            entry.treeId,
            current.id,
            revision,
          );
          set((s) => ({
            trees: s.trees.map((t) =>
              t.id === entry.treeId
                ? { ...t, nodes: t.nodes.filter((n) => n.id !== current.id) }
                : t,
            ),
            selectedNodeId:
              s.selectedNodeId === current.id ? null : s.selectedNodeId,
            selectedNodeIds: s.selectedNodeIds.filter(
              (id) => id !== current.id,
            ),
            historyUndo: pushHistory(s.historyUndo, {
              kind: "delete_node",
              treeId: entry.treeId,
              nodeSnapshot: current,
            }),
          }));
        }
      }
      get().fetchPreview(projectId, entry.treeId);
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        errorProjectId: projectId,
      });
      console.error("[focus redo] failed:", err);
      set((s) => ({ historyRedo: pushHistory(s.historyRedo, entry) }));
    } finally {
      set({ isReplaying: false });
    }
  },

  duplicateNode: async (projectId, treeId, nodeId) => {
    const original = findNode(get().trees, treeId, nodeId);
    if (!original) return;

    const tree = get().trees.find((t) => t.id === treeId);
    const existingIds = new Set((tree?.nodes ?? []).map((n) => n.focus_id));
    let candidate = `${original.focus_id}_copy`;
    let n = 2;
    while (existingIds.has(candidate)) {
      candidate = `${original.focus_id}_copy_${n}`;
      n += 1;
    }

    await get().createNode(projectId, treeId, {
      focus_id: candidate,
      name_en: original.name_en ? `${original.name_en} (copy)` : "",
      name_zh: original.name_zh ? `${original.name_zh}(副本)` : "",
      icon: original.icon,
      x: original.x + 40,
      y: original.y + 40,
      cost: original.cost,
      available: original.available,
      completion_reward: original.completion_reward,
      bypass: original.bypass,
      cancel: original.cancel,
      ai_will_do: original.ai_will_do,

      prerequisites: [],
      mutually_exclusive: [],
      allow_branch_ast: original.allow_branch_ast,
      raw_extras: original.raw_extras,
    });
  },

  selectAllNodes: (treeId) => {
    const tree = get().trees.find((t) => t.id === treeId);
    if (!tree) return;
    const ids = tree.nodes.map((n) => n.id);
    set((s) => ({
      selectedNodeIds: ids,
      selectedNodeId: tree.nodes.length === 1 ? tree.nodes[0].id : null,
      selectedEdgeId: ids.length > 0 ? null : s.selectedEdgeId,
    }));
  },
}));

subscribeEditRevisionWrites((projectId, revisions) => {
  useFocusTreeStore.setState((state) => ({
    trees: state.trees.map((tree) => {
      const revision = revisions["focus_tree:" + tree.id];
      if (tree.project_id !== projectId || !revision) return tree;
      return {
        ...tree,
        edit_revision: revision,
        nodes: tree.nodes.map((node) => ({
          ...node,
          tree_edit_revision: revision,
        })),
      };
    }),
  }));
});
