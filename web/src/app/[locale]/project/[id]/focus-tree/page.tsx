"use client";
import { focusEditRevisions } from "@/components/workbench/save-diagnostics/editRevisions";
import { OpenInGameLink } from "@/components/workbench/OpenInGameLink";
import { gameViewHref } from "@/lib/gameViewLinks";
import { focusPreviewForSelection } from "@/components/workbench/save-diagnostics/focusPreviewSnapshot";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { useShallow } from "zustand/react/shallow";
import dynamic from "@/platform/dynamic";
import { useParams, useSearchParams } from "@/platform/navigation";
import {
  Check,
  ChevronDown,
  ChevronRight,
  ClipboardList,
  FolderTree,
  Network,
  Plus,
  Search,
  Settings2,
  Trash2,
  X,
} from "lucide-react";
import { useFocusTreeStore } from "@/lib/store/focusTreeStore";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useImportButton } from "@/components/workbench/ImportButton";
import {
  absOccupiedCells,
  findFreeGridCell,
  findFreeMutexCellLeft,
  findFreeMutexCellRight,
  resolveAbsolutePosition,
} from "@/lib/focusTreeGraph";
import { FocusTreeSidebar } from "@/components/workbench/focus-graph/FocusTreeSidebar";
import { useEditorWiring } from "@/shared/editors/useEditorWiring";
import {
  buildInitialShowPositionUpdate,
  focusTreesApi,
  importApi,
  readableError,
} from "@/lib/api";
import { isGameFolderMissing } from "@/lib/api/gameIndex";
import { useFocusIconInventory } from "@/lib/hooks/useIconInventory";
import { useDeadFlagLint } from "@/shared/useDeadFlagLint";
import { DeadFlagBanner } from "@/components/workbench/DeadFlagBanner";
import {
  useProjectStore,
  useActiveCountry,
  useEffectiveCountryFilter,
} from "@/lib/store/projectStore";
import { matchesCountryScope, partitionByScope } from "@/lib/utils/scope";
import {
  parseRunCheckEntityDeepLink,
  resolveDeepLinkScopeReveal,
} from "@/lib/runCheckDeepLink";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { ScopeSectionHeader } from "@/components/workbench/ScopeBadge";
import { InfoTooltip, TooltipProvider } from "@/components/ui/tooltip";
import { TreeValidationButton } from "@/components/workbench/TreeValidationDialog";
import {
  previewMarkersForLintTarget,
  type LintReport,
} from "@/components/workbench/treeValidationMarkers";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { FocusIconLibraryDialog } from "@/components/workbench/focus-graph/FocusIconLibraryDialog";
import { ReadableScriptedName } from "@/components/workbench/focus-graph/scriptedName";
import { useBatchGfxApply } from "@/lib/hooks/useBatchGfxApply";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { toast } from "sonner";
import type {
  FocusNode,
  FocusNodeCreateInput,
  FocusTree,
  InitialShowPosition,
} from "@/types";
import { detectFocusPrefix, suggestNewFocusId } from "@/lib/focusIdSuggest";
import {
  getInitialPositionEditorState,
  type InitialPositionMode,
} from "./initialPositionUi";
import {
  buildDraftPreviewEntities,
  findPersistedSelectedEntity,
} from "./focusNodeDraftView";

function GraphLoadingFallback() {
  const t = useTranslations("focusTree");
  return (
    <div className="flex-1 flex items-center justify-center text-muted-foreground">
      {t("loadingGraph")}
    </div>
  );
}

const FocusTreeGraph = dynamic(
  () =>
    import("@/components/workbench/focus-graph/FocusTreeGraph").then(
      (m) => m.FocusTreeGraph,
    ),
  { ssr: false, loading: () => <GraphLoadingFallback /> },
);

const EMPTY_FOCUS_NODES: FocusNode[] = [];

function TreeRow({
  tree,
  selected,
  onSelect,
  onDelete,
  navProps,
}: {
  tree: FocusTree;
  selected: boolean;
  onSelect: () => void;
  onDelete: (e: React.MouseEvent) => void;

  navProps?: React.HTMLAttributes<HTMLDivElement>;
}) {
  const t = useTranslations("focusTree");

  const isPool = tree.tree_kind === "shared_pool";
  const refCount = tree.shared_focus_refs?.length ?? 0;
  return (
    <div
      {...navProps}
      className={`group flex items-center justify-between px-3 py-2 rounded-md cursor-pointer text-sm ui-interactive focus-visible:ring-1 focus-visible:ring-ring ${
        selected ? "bg-accent/10 text-accent font-medium" : "hover:bg-muted"
      }`}
      onClick={onSelect}
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1">
          <p className="font-medium truncate font-mono text-xs flex-1">
            {tree.tree_id}
          </p>

          {!isPool && tree.country_tag && (
            <span
              className="shrink-0 text-micro font-mono font-semibold px-1 py-0.5 rounded bg-muted text-muted-foreground leading-none"
              title={t("primaryAiCountryTitle", { tag: tree.country_tag })}
            >
              {tree.country_tag}
            </span>
          )}
        </div>
        <p className="text-xs text-muted-foreground">
          {t("nodeCount", { count: tree.nodes.length })}
          {refCount > 0 && <span> · {t("refCount", { count: refCount })}</span>}
          {isPool && tree.source_file && (
            <span className="block truncate opacity-70">
              {tree.source_file}
            </span>
          )}
          {tree.file_policy && (
            <span
              className="block truncate opacity-80"
              title={t("filePolicyHint", {
                layer: tree.file_policy.target_layer,
                reason:
                  tree.file_policy.emit_mode === "mandatory_shadow_same_path"
                    ? t("filePolicyMandatory")
                    : t("filePolicyAddition"),
              })}
            >
              {t("filePolicyTarget", { path: tree.file_policy.target_path })}
            </span>
          )}
        </p>
      </div>
      <button
        className="text-xs text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 shrink-0 ml-2"
        onClick={onDelete}
        aria-label={t("deleteTree")}
        title={t("deleteTree")}
      >
        x
      </button>
    </div>
  );
}

function OwnTreesSectionHeader({ count }: { count: number }) {
  const t = useTranslations("focusTree");
  return (
    <div className="flex items-center gap-1.5 px-1 mt-2 mb-2 first:mt-0">
      <span className="text-xs font-semibold text-foreground">
        {t("sectionOwnLabel")}
      </span>
      <span className="text-xs text-muted-foreground">{count}</span>
      <TooltipProvider>
        <InfoTooltip content={t("sectionOwnHint")} />
      </TooltipProvider>
    </div>
  );
}

export default function FocusTreePage() {
  const t = useTranslations("focusTree");
  const tModCheck = useTranslations("modCheck");
  const tCommon = useTranslations("common");

  const locale = useContentLocaleStore((s) => s.locale);
  const params = useParams();
  const searchParams = useSearchParams();
  const projectId = Number(params.id);
  const runCheckTarget = useMemo(
    () => parseRunCheckEntityDeepLink(searchParams, "tree", "focus"),
    [searchParams],
  );
  const runCheckTargetKey = runCheckTarget
    ? `${runCheckTarget.containerId}:${runCheckTarget.entityId}`
    : null;
  const handledRunCheckTargetRef = useRef<string | null>(null);

  const currentProject = useProjectStore((s) => s.currentProject);
  const activeCountry = useActiveCountry();

  const countryTag = activeCountry || currentProject?.country_tag || "";

  const filterCountry = useEffectiveCountryFilter();

  const { report: deadFlagReport } = useDeadFlagLint(projectId || null);

  const confirmDialog = useConfirm();

  const treeListNav = useListboxNav();
  const nodeListNav = useListboxNav();

  const [nodeQuery, setNodeQuery] = useState("");

  const {
    trees,
    selectedTreeId,
    selectedNodeId,
    selectedNodeIds,
    selectedEdgeId,
    loading,
    error,
    fetchTrees,
    selectTree,
    selectNode,
    setSelectedNodeIds,
    selectEdge,
    createTree,
    updateTree,
    deleteTree,
    createNode: createNodeRaw,
    updateNode,
    deleteNode,
    fetchPreview,
    updateNodePosition,
    bulkUpdateNodePositions,
    addPrerequisite,
    removePrerequisite,
    addMutualExclusion,
    removeMutualExclusion,
    applyUploadedNodeIcon,
    setPreviewEnabled,
  } = useFocusTreeStore(
    useShallow((s) => ({
      trees: s.trees,
      selectedTreeId: s.selectedTreeId,
      selectedNodeId: s.selectedNodeId,
      selectedNodeIds: s.selectedNodeIds,
      selectedEdgeId: s.selectedEdgeId,
      loading: s.loading,
      error: s.error,
      fetchTrees: s.fetchTrees,
      selectTree: s.selectTree,
      selectNode: s.selectNode,
      setSelectedNodeIds: s.setSelectedNodeIds,
      selectEdge: s.selectEdge,
      createTree: s.createTree,
      updateTree: s.updateTree,
      deleteTree: s.deleteTree,
      createNode: s.createNode,
      updateNode: s.updateNode,
      deleteNode: s.deleteNode,
      fetchPreview: s.fetchPreview,
      updateNodePosition: s.updateNodePosition,
      bulkUpdateNodePositions: s.bulkUpdateNodePositions,
      addPrerequisite: s.addPrerequisite,
      removePrerequisite: s.removePrerequisite,
      addMutualExclusion: s.addMutualExclusion,
      removeMutualExclusion: s.removeMutualExclusion,
      applyUploadedNodeIcon: s.applyUploadedNodeIcon,
      setPreviewEnabled: s.setPreviewEnabled,
    })),
  );

  const createNode = useCallback(
    async (pid: number, treeId: number, data?: FocusNodeCreateInput) => {
      const created = await createNodeRaw(pid, treeId, data);
      if (created === null) {
        toast.error(t("focusIdConflict", { focusId: data?.focus_id ?? "" }));
      }
      return created;
    },
    [createNodeRaw, t],
  );

  const cachedScriptPreview = useFocusTreeStore((s) => s.scriptPreview);
  const scriptPreviewSnapshot = useFocusTreeStore(
    (s) => s.scriptPreviewSnapshot,
  );
  const {
    scriptPreview,
    editRevision: previewEditRevision,
    previewReady,
  } = focusPreviewForSelection(
    cachedScriptPreview,
    scriptPreviewSnapshot,
    projectId,
    selectedTreeId,
    selectedNodeId,
  );
  const errorProjectId = useFocusTreeStore((s) => s.errorProjectId);
  useEffect(() => {
    if (error && errorProjectId === projectId && trees.length > 0)
      toast.error(error);
  }, [error, errorProjectId, projectId, trees.length]);
  const importNonce = useFocusTreeStore((s) => s.importNonce);
  const previewEnabled = useFocusTreeStore((s) => s.previewEnabled);

  useEffect(() => () => setPreviewEnabled(false), [setPreviewEnabled]);

  const focusImport = useImportButton({
    label: t("import"),
    onImport: (file) =>
      importApi.focusTree(
        projectId,
        file,
        countryTag,
        undefined,
        focusEditRevisions(trees),
      ),
    onComplete: () => fetchTrees(projectId),
  });

  const { icons: availableIcons, refresh: refreshAvailableIcons } =
    useFocusIconInventory(projectId || null);
  const [iconCacheBust, setIconCacheBust] = useState<
    { gfxName?: string; version: number } | undefined
  >();
  useEffect(() => {
    if (importNonce > 0) setIconCacheBust({ version: Date.now() });
  }, [refreshAvailableIcons, importNonce]);

  useSWR(projectId ? ["focus/trees", projectId] : null, () =>
    fetchTrees(projectId),
  );

  useEffect(() => {
    if (!runCheckTarget || !runCheckTargetKey) {
      handledRunCheckTargetRef.current = null;
      return;
    }
    if (handledRunCheckTargetRef.current === runCheckTargetKey) return;

    const targetTree = trees.find(
      (tree) => tree.id === runCheckTarget.containerId,
    );
    const targetNode = targetTree?.nodes.find(
      (node) => node.id === runCheckTarget.entityId,
    );
    if (!targetTree || !targetNode) return;

    const projectState = useProjectStore.getState();
    const targetProject = projectState.currentProject;
    const projectCountryTags = targetProject?.country_tags?.length
      ? targetProject.country_tags
      : targetProject?.country_tag
        ? [targetProject.country_tag]
        : [];
    const effectiveCountry =
      projectState.viewMode === "all"
        ? null
        : (projectState.activeCountryTag ?? targetProject?.country_tag ?? null);
    const reveal = resolveDeepLinkScopeReveal(
      targetTree,
      effectiveCountry,
      projectCountryTags,
    );
    if (reveal.kind === "country") {
      projectState.setActiveCountryTag(reveal.countryTag);
    } else if (reveal.kind === "all") {
      projectState.setViewMode("all");
    }

    if (selectedTreeId !== targetTree.id) selectTree(targetTree.id);
    if (selectedNodeId !== targetNode.id) selectNode(targetNode.id);
    handledRunCheckTargetRef.current = runCheckTargetKey;
  }, [
    runCheckTarget,
    runCheckTargetKey,
    selectNode,
    selectTree,
    selectedNodeId,
    selectedTreeId,
    trees,
  ]);

  useEffect(() => {
    if (!selectedTreeId) return;
    const sel = trees.find((t) => t.id === selectedTreeId);
    if (!sel) return;
    if (sel.tree_kind === "shared_pool") return;
    if (filterCountry && !matchesCountryScope(sel, filterCountry)) {
      selectTree(null);
    }
  }, [filterCountry, selectedTreeId, trees, selectTree]);

  useSWR(
    previewEnabled && projectId && selectedTreeId
      ? ["focus/preview", projectId, selectedTreeId, selectedNodeId ?? null]
      : null,
    () =>
      selectedNodeId
        ? fetchPreview(projectId, selectedTreeId!, selectedNodeId)
        : fetchPreview(projectId, selectedTreeId!),
  );

  const currentTree = trees.find((t) => t.id === selectedTreeId);
  const nodes = currentTree?.nodes ?? EMPTY_FOCUS_NODES;
  const selectedNode = findPersistedSelectedEntity(nodes, selectedNodeId);
  const [focusNodeDraft, setFocusNodeDraft] = useState<FocusNode | null>(null);
  const [treeLintReport, setTreeLintReport] = useState<LintReport | null>(null);
  const handleSelectNodes = useCallback(
    (nodeIds: number[]) => {
      setSelectedNodeIds(nodeIds);
      if (nodeIds.length > 0) selectEdge(null);
    },
    [setSelectedNodeIds, selectEdge],
  );

  const [initialRevision, setInitialRevision] = useState<string | undefined>();
  const [initialSettingsOpen, setInitialSettingsOpen] = useState(false);
  const [initialSettingsTreeId, setInitialSettingsTreeId] = useState<
    number | null
  >(null);
  const [initialPositionMode, setInitialPositionMode] =
    useState<InitialPositionMode>("none");
  const [initialFocusId, setInitialFocusId] = useState("");
  const [initialX, setInitialX] = useState(0);
  const [initialY, setInitialY] = useState(0);
  const [initialOffsets, setInitialOffsets] = useState<
    { x: number; y: number; trigger: string }[]
  >([]);
  const [initialSaveState, setInitialSaveState] = useState<
    "idle" | "saving" | "saved" | "error"
  >("idle");

  const activeInitialSettingsTreeId = currentTree?.id ?? null;
  const persistedInitialPosition = getInitialPositionEditorState(currentTree);
  if (initialSettingsTreeId !== activeInitialSettingsTreeId) {
    setInitialSettingsTreeId(activeInitialSettingsTreeId);
    setInitialRevision(currentTree?.edit_revision);
    setInitialPositionMode(persistedInitialPosition.mode);
    setInitialFocusId(persistedInitialPosition.focusId);
    setInitialX(persistedInitialPosition.x);
    setInitialY(persistedInitialPosition.y);
    setInitialOffsets(persistedInitialPosition.offsets);
    setInitialSaveState("idle");
    setInitialSettingsOpen(false);
  }

  const unresolvedInitialPosition = persistedInitialPosition.unresolved;
  const initialPositionCanSave =
    initialPositionMode === "none" ||
    (initialPositionMode === "focus" && initialFocusId.trim().length > 0) ||
    (initialPositionMode === "coordinates" &&
      initialOffsets.every((offset) => offset.trigger.trim().length > 0));

  const handleSaveInitialPosition = async () => {
    if (!currentTree || !initialPositionCanSave) return;
    let payload: InitialShowPosition | null = null;
    if (initialPositionMode === "focus") {
      payload = { kind: "focus", focus_id: initialFocusId.trim() };
    } else if (initialPositionMode === "coordinates") {
      payload = {
        kind: "coordinates",
        x: Math.trunc(initialX),
        y: Math.trunc(initialY),
        offsets: initialOffsets.map((offset) => ({
          x: Math.trunc(offset.x),
          y: Math.trunc(offset.y),
          trigger: offset.trigger.trim(),
        })),
      };
    }
    setInitialSaveState("saving");
    try {
      await updateTree(projectId, currentTree.id, {
        ...buildInitialShowPositionUpdate(payload),
        edit_revision: initialRevision,
      });
      setInitialSaveState("saved");
    } catch {
      setInitialSaveState("error");
    }
  };

  const visualNodes = useMemo(
    () => buildDraftPreviewEntities(nodes, selectedNodeId, focusNodeDraft),
    [nodes, focusNodeDraft, selectedNodeId],
  );

  const primaryTrees = trees.filter(
    (t) =>
      t.tree_kind !== "shared_pool" && matchesCountryScope(t, filterCountry),
  );
  const sharedPoolTrees = trees.filter((t) => t.tree_kind === "shared_pool");

  const sortedNodes = [...visualNodes].sort((a, b) =>
    a.y !== b.y ? a.y - b.y : a.x - b.x,
  );

  const nodeQueryTrimmed = nodeQuery.trim().toLowerCase();

  const coordQuery = nodeQueryTrimmed.replace(/[()\s]/g, "");
  const visibleNodes = nodeQueryTrimmed
    ? sortedNodes.filter((n) => {
        const name = resolveDisplayName(
          locale,
          n.display_name_locs,
          n.name_en,
          n.name_zh,
          n.name || n.focus_id,
        );
        return (
          name.toLowerCase().includes(nodeQueryTrimmed) ||
          (n.focus_id || "").toLowerCase().includes(nodeQueryTrimmed) ||
          (coordQuery !== "" && `${n.x},${n.y}`.includes(coordQuery))
        );
      })
    : sortedNodes;

  const createTreeFailed = (err: unknown) =>
    toast.error(
      isGameFolderMissing(err)
        ? tCommon("gameFolderMissing")
        : readableError(err),
    );

  const handleCreateTree = async () => {
    try {
      await createTree(projectId, countryTag || undefined);
    } catch (err) {
      createTreeFailed(err);
    }
  };

  const handleHeroFirstStep = () => {
    void handleCreateTree();
  };

  const handleDeleteTree = async (treeId: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (await confirmDialog({ body: t("confirmDeleteTree"), danger: true })) {
      await deleteTree(projectId, treeId);
    }
  };

  const detectedPrefix = detectFocusPrefix(nodes);

  const projectFocusNodes = useMemo(
    () => trees.flatMap((t) => t.nodes),
    [trees],
  );

  const withSuggestedFocusId = useCallback(
    (merged: FocusNodeCreateInput): FocusNodeCreateInput => {
      if (merged.focus_id || !detectedPrefix) return merged;
      const suggestion = suggestNewFocusId(detectedPrefix, projectFocusNodes);
      return suggestion
        ? { ...merged, focus_id: suggestion, on_conflict: "suffix" }
        : merged;
    },
    [detectedPrefix, projectFocusNodes],
  );

  const batchGfx = useBatchGfxApply({
    applyOne: async (nodeId, gfxName) => {
      if (!selectedTreeId) throw new Error("no tree selected");
      await updateNode(projectId, selectedTreeId, nodeId, { icon: gfxName });
    },
    onComplete: () => {
      if (selectedTreeId) fetchPreview(projectId, selectedTreeId);
    },
  });

  const handleCreateNode = useCallback(
    async (data?: Partial<FocusNode>) => {
      if (!selectedTreeId) return;

      const merged: FocusNodeCreateInput = withSuggestedFocusId(
        data ? { ...data } : {},
      );

      if (merged.x === undefined && merged.y === undefined) {
        const occupied = absOccupiedCells(nodes);
        let i = 0;

        const limit = nodes.length + 1;
        while (i < limit * 5) {
          const x = i % 5;
          const y = Math.floor(i / 5);
          if (!occupied.has(`${x},${y}`)) {
            merged.x = x;
            merged.y = y;
            break;
          }
          i += 1;
        }
      }
      await createNode(
        projectId,
        selectedTreeId,
        Object.keys(merged).length ? merged : undefined,
      );
    },
    [selectedTreeId, projectId, nodes, withSuggestedFocusId, createNode],
  );

  const handleSaveNode = async (data: Partial<FocusNode>) => {
    if (!selectedTreeId || !selectedNodeId)
      throw new Error(t("noTreeSelectedDesc"));
    await updateNode(projectId, selectedTreeId, selectedNodeId, data);
  };

  const parsePreviewCall = useCallback(
    (script: string) =>
      focusTreesApi.parseNodePreview(
        projectId,
        selectedTreeId!,
        selectedNodeId!,
        script,
      ),
    [projectId, selectedTreeId, selectedNodeId],
  );
  const treeLintMarkers = useMemo(
    () =>
      selectedNode
        ? previewMarkersForLintTarget(
            treeLintReport,
            {
              tab: "focus",
              entityId: selectedNode.focus_id,
              field: "completion_reward",
            },
            {
              fieldSource: selectedNode.completion_reward ?? "",
              scriptPreview,
            },
          )
        : [],
    [selectedNode, scriptPreview, treeLintReport],
  );
  const editorWiring = useEditorWiring({
    editRevision: previewEditRevision,
    previewReady,
    workbench: "focus_node",
    entityId: selectedNodeId,
    scriptPreview,
    parsePreview: parsePreviewCall,
    applyEntity: (id, entityDict, editRevision) => {
      if (!selectedTreeId)
        return Promise.reject(new Error(t("noTreeSelectedDesc")));
      return updateNode(projectId, selectedTreeId, id, {
        ...entityDict,
        tree_edit_revision: editRevision,
      } as Partial<FocusNode>);
    },
    onAfterApply: async (id) => {
      if (selectedTreeId) await fetchPreview(projectId, selectedTreeId, id);
    },
    entityNoun: "focus node",
    projectId,
    externalMarkers: treeLintMarkers,
  });

  const handleDeleteNode = useCallback(
    async (nodeId: number) => {
      if (!selectedTreeId) return;
      await deleteNode(projectId, selectedTreeId, nodeId);
    },
    [selectedTreeId, projectId, deleteNode],
  );

  const handleDeleteNodeFromList = useCallback(
    async (nodeId: number, e: React.MouseEvent) => {
      e.stopPropagation();
      if (!selectedTreeId) return;
      if (await confirmDialog({ body: t("confirmDeleteNode"), danger: true })) {
        await deleteNode(projectId, selectedTreeId, nodeId);
      }
    },
    [selectedTreeId, projectId, deleteNode, confirmDialog, t],
  );

  const handleUpdatePosition = useCallback(
    async (nodeId: number, x: number, y: number) => {
      if (!selectedTreeId) return;
      await updateNodePosition(projectId, selectedTreeId, nodeId, x, y);
    },
    [selectedTreeId, projectId, updateNodePosition],
  );

  const handleBulkUpdatePositions = useCallback(
    (positions: Array<{ id: number; x: number; y: number }>) => {
      if (!selectedTreeId) return;
      void bulkUpdateNodePositions(projectId, selectedTreeId, positions);
    },
    [selectedTreeId, projectId, bulkUpdateNodePositions],
  );

  const handleAddPrerequisite = useCallback(
    async (nodeId: number, prereqFocusId: string) => {
      if (!selectedTreeId) return;
      await addPrerequisite(projectId, selectedTreeId, nodeId, prereqFocusId);
    },
    [selectedTreeId, projectId, addPrerequisite],
  );

  const handleRemovePrerequisite = useCallback(
    async (nodeId: number, prereqFocusId: string, groupIndex: number) => {
      if (!selectedTreeId) return;
      await removePrerequisite(
        projectId,
        selectedTreeId,
        nodeId,
        prereqFocusId,
        groupIndex,
      );
    },
    [selectedTreeId, projectId, removePrerequisite],
  );

  const handleAddMutualExclusion = useCallback(
    async (nodeId: number, otherFocusId: string) => {
      if (!selectedTreeId) return;
      await addMutualExclusion(projectId, selectedTreeId, nodeId, otherFocusId);
    },
    [selectedTreeId, projectId, addMutualExclusion],
  );

  const handleRemoveMutualExclusion = useCallback(
    async (nodeId: number, otherFocusId: string) => {
      if (!selectedTreeId) return;
      await removeMutualExclusion(
        projectId,
        selectedTreeId,
        nodeId,
        otherFocusId,
      );
    },
    [selectedTreeId, projectId, removeMutualExclusion],
  );

  const handleCreateChildNode = useCallback(
    async (parentNodeId: number) => {
      if (!selectedTreeId) return;
      const parent = nodes.find((n) => n.id === parentNodeId);
      if (!parent) return;

      const parentAbs = resolveAbsolutePosition(parent, nodes);
      const occupied = absOccupiedCells(nodes);
      const pos = findFreeGridCell(
        occupied,
        parentAbs.x,
        parentAbs.y + 1,
        nodes.length + 1,
      );

      const merged: FocusNodeCreateInput = withSuggestedFocusId({
        x: pos.x,
        y: pos.y,
      });

      merged.prerequisites = [[parent.focus_id]];

      await createNode(projectId, selectedTreeId, merged);
    },
    [selectedTreeId, projectId, nodes, withSuggestedFocusId, createNode],
  );

  const handleCreateMutexChildNode = useCallback(
    async (parentNodeId: number, side: "left" | "right") => {
      if (!selectedTreeId) return;
      const parent = nodes.find((n) => n.id === parentNodeId);
      if (!parent) return;

      const parentAbs = resolveAbsolutePosition(parent, nodes);
      const occupied = absOccupiedCells(nodes);

      const pos =
        side === "left"
          ? findFreeMutexCellLeft(
              occupied,
              parentAbs.x,
              parentAbs.y,
              nodes.length + 1,
            )
          : findFreeMutexCellRight(
              occupied,
              parentAbs.x,
              parentAbs.y,
              nodes.length + 1,
            );

      const merged: FocusNodeCreateInput = withSuggestedFocusId({
        x: pos.x,
        y: pos.y,
      });

      merged.mutually_exclusive = [parent.focus_id];

      const child = await createNode(projectId, selectedTreeId, merged);
      if (!child) return;

      await updateNode(projectId, selectedTreeId, parent.id, {
        mutually_exclusive: [
          ...(parent.mutually_exclusive || []),
          child.focus_id,
        ],
      });
    },
    [
      selectedTreeId,
      projectId,
      nodes,
      withSuggestedFocusId,
      createNode,
      updateNode,
    ],
  );

  const handleCreateConnectedNode = useCallback(
    async (
      parentNodeId: number,
      connectionType: "prereq" | "mutex",
      position: { x: number; y: number },
    ) => {
      if (!selectedTreeId) return;
      const parent = nodes.find((n) => n.id === parentNodeId);
      if (!parent) return;

      const occupied = absOccupiedCells(nodes);
      const pos = findFreeGridCell(
        occupied,
        position.x,
        position.y,
        nodes.length + 1,
      );

      const merged: FocusNodeCreateInput = withSuggestedFocusId({
        x: pos.x,
        y: pos.y,
      });

      if (connectionType === "prereq") {
        merged.prerequisites = [[parent.focus_id]];
      } else {
        merged.mutually_exclusive = [parent.focus_id];
      }

      const child = await createNode(projectId, selectedTreeId, merged);
      if (!child) return;

      if (connectionType === "mutex") {
        await updateNode(projectId, selectedTreeId, parent.id, {
          mutually_exclusive: [
            ...(parent.mutually_exclusive || []),
            child.focus_id,
          ],
        });
      }
    },
    [
      selectedTreeId,
      projectId,
      nodes,
      withSuggestedFocusId,
      createNode,
      updateNode,
    ],
  );

  const drawioInputRef = useRef<HTMLInputElement | null>(null);
  const [drawioImporting, setDrawioImporting] = useState(false);
  const handleDrawioFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    setDrawioImporting(true);
    try {
      const result = await importApi.drawio(projectId, file, countryTag);
      await fetchTrees(projectId);
      selectTree(result.tree_db_id);
      const warningMsg =
        result.warnings.length > 0
          ? t("drawioWarningSuffix", { count: result.warnings.length })
          : "";
      if (result.warnings.length > 0)
        console.warn("[drawio import] warnings:", result.warnings);
      toast.success(
        t("drawioImported", {
          focuses: result.focuses_created,
          edges: result.prereq_edges_created,
          tree: result.tree_id,
          warnings: warningMsg,
        }),
      );
    } catch (err) {
      toast.error(
        t("drawioImportFailed", {
          error: err instanceof Error ? err.message : t("unknownError"),
        }),
      );
    } finally {
      setDrawioImporting(false);
    }
  };

  return (
    <div className="flex h-full min-w-0 overflow-hidden">
      <div className="w-68 ui-ide-panel overflow-y-auto shrink-0">
        <div className="p-3">
          {deadFlagReport && deadFlagReport.dead_flags.length > 0 && (
            <div className="mb-3">
              <DeadFlagBanner report={deadFlagReport} />
            </div>
          )}

          <div className="mb-3 space-y-2">
            <div className="flex items-baseline gap-2 min-w-0">
              <h3 className="text-micro font-semibold uppercase tracking-widest text-muted-foreground">
                {t("focusTreesHeading")}
              </h3>
              <span className="bg-muted text-muted-foreground rounded-full px-2 text-xs">
                {trees.length}
              </span>
              <span className="flex-1" />
              <OpenInGameLink
                href={gameViewHref(projectId, {
                  panel: "nationalfocusview",
                  tree: currentTree?.tree_id,
                  tag: currentTree?.country_tag || countryTag,
                })}
              />
            </div>

            <div className="flex items-center gap-1">
              {focusImport.trigger}

              <input
                ref={drawioInputRef}
                type="file"
                accept=".drawio,.xml"
                style={{ display: "none" }}
                onChange={handleDrawioFile}
              />
              <Button
                size="sm"
                variant="outline"
                onClick={() => drawioInputRef.current?.click()}
                disabled={drawioImporting}
                title={t("drawioTitle")}
              >
                {drawioImporting ? t("drawioImporting") : t("drawio")}
              </Button>
              <Button
                size="sm"
                className="text-xs font-semibold"
                onClick={handleCreateTree}
              >
                {t("addTree")}
              </Button>
            </div>
            {focusImport.resultPanel && (
              <div className="mt-2">{focusImport.resultPanel}</div>
            )}
          </div>

          {loading && trees.length === 0 ? (
            <div className="space-y-1.5 mb-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <div
                  key={`skel-${i}`}
                  className="flex items-center justify-between px-3 py-2 rounded-md"
                >
                  <div className="flex items-center gap-2 flex-1 min-w-0">
                    <div className="h-3 w-8 rounded bg-muted/50 animate-pulse" />
                    <div className="h-3 w-40 rounded bg-muted/60 animate-pulse" />
                  </div>
                  <div className="h-3 w-10 rounded bg-muted/40 animate-pulse" />
                </div>
              ))}
            </div>
          ) : trees.length === 0 ? (
            <p className="text-xs text-muted-foreground text-center py-6">
              {t("noTrees")}
            </p>
          ) : (
            <div
              className="mb-3 space-y-3 outline-none"
              {...treeListNav.containerProps}
              aria-label={t("focusTreesHeading")}
            >
              {primaryTrees.length > 0 &&
                (() => {
                  const { own, shared } = partitionByScope(
                    primaryTrees,
                    filterCountry,
                  );
                  const renderTreeRow = (
                    tree: (typeof primaryTrees)[number],
                  ) => (
                    <TreeRow
                      key={tree.id}
                      tree={tree}
                      selected={selectedTreeId === tree.id}
                      navProps={treeListNav.optionProps(
                        selectedTreeId === tree.id,
                      )}
                      onSelect={() => {
                        selectTree(tree.id);
                        fetchPreview(projectId, tree.id);
                      }}
                      onDelete={(e) => handleDeleteTree(tree.id, e)}
                    />
                  );
                  if (!filterCountry) {
                    return (
                      <div className="space-y-1">
                        <ScopeSectionHeader
                          variant="all"
                          count={primaryTrees.length}
                        />
                        {primaryTrees.map(renderTreeRow)}
                      </div>
                    );
                  }
                  return (
                    <div className="space-y-1">
                      {own.length > 0 && (
                        <>
                          <OwnTreesSectionHeader count={own.length} />
                          {own.map(renderTreeRow)}
                        </>
                      )}
                      {shared.length > 0 && (
                        <>
                          <ScopeSectionHeader
                            variant="shared"
                            count={shared.length}
                          />
                          {shared.map(renderTreeRow)}
                        </>
                      )}
                    </div>
                  );
                })()}

              {primaryTrees.length === 0 && sharedPoolTrees.length > 0 && (
                <p className="text-xs text-muted-foreground px-1">
                  {t("noPrimaryTrees", {
                    country: countryTag || t("thisCountry"),
                  })}
                </p>
              )}

              {sharedPoolTrees.length > 0 && (
                <div className="space-y-1 border-t border-border/40 pt-2">
                  <div className="px-1 text-micro font-semibold uppercase tracking-widest text-muted-foreground flex items-center gap-1">
                    <span>{t("sharedPools")}</span>
                    <span className="text-muted-foreground/40">
                      ({sharedPoolTrees.length})
                    </span>
                  </div>
                  {sharedPoolTrees.map((tree) => (
                    <TreeRow
                      key={tree.id}
                      tree={tree}
                      selected={selectedTreeId === tree.id}
                      navProps={treeListNav.optionProps(
                        selectedTreeId === tree.id,
                      )}
                      onSelect={() => {
                        selectTree(tree.id);
                        fetchPreview(projectId, tree.id);
                      }}
                      onDelete={(e) => handleDeleteTree(tree.id, e)}
                    />
                  ))}
                </div>
              )}
            </div>
          )}

          {currentTree ? (
            <div className="mb-3 border-t border-border/40 pt-3">
              <button
                type="button"
                onClick={() => setInitialSettingsOpen((open) => !open)}
                aria-expanded={initialSettingsOpen}
                className="ui-interactive ui-focus-ring flex w-full items-center gap-1.5 rounded-md text-left text-xs text-foreground"
              >
                {initialSettingsOpen ? (
                  <ChevronDown
                    className="h-3.5 w-3.5 text-muted-foreground"
                    aria-hidden
                  />
                ) : (
                  <ChevronRight
                    className="h-3.5 w-3.5 text-muted-foreground"
                    aria-hidden
                  />
                )}
                <Settings2
                  className="h-3.5 w-3.5 text-muted-foreground"
                  aria-hidden
                />
                <span className="font-medium">{t("initialPositionTitle")}</span>
              </button>

              {initialSettingsOpen ? (
                <div className="mt-2 space-y-3 rounded-md border border-border bg-card p-3">
                  <p className="text-xs text-muted-foreground">
                    {t("initialPositionHint")}
                  </p>

                  {unresolvedInitialPosition ? (
                    <div className="rounded-md border border-warning/40 bg-warning/5 px-2 py-1.5 text-xs text-warning">
                      <p className="font-medium">
                        {tModCheck(
                          unresolvedInitialPosition.kind === "legacy_ambiguous"
                            ? "msg_focus_initial_position_legacy_ambiguous"
                            : "msg_focus_initial_position_invalid_shape",
                        )}
                      </p>
                      <code className="mt-1 block max-h-24 overflow-auto rounded-md border border-border bg-card px-2 py-1 font-mono text-foreground">
                        {unresolvedInitialPosition.raw || "—"}
                      </code>
                      <p className="mt-0.5 text-muted-foreground">
                        {tModCheck("hint_focus_initial_position_choose_kind")}
                      </p>
                    </div>
                  ) : null}

                  <div className="space-y-1">
                    <label
                      className="text-xs font-medium"
                      htmlFor="initial-position-mode"
                    >
                      {t("initialPositionMode")}
                    </label>
                    <select
                      id="initial-position-mode"
                      value={initialPositionMode}
                      onChange={(event) => {
                        setInitialPositionMode(
                          event.target.value as typeof initialPositionMode,
                        );
                        setInitialSaveState("idle");
                      }}
                      className="ui-focus-ring h-9 w-full rounded-md border border-input bg-card px-2 text-sm"
                    >
                      {initialPositionMode === "unresolved" ? (
                        <option value="unresolved" disabled>
                          {tModCheck("hint_focus_initial_position_choose_kind")}
                        </option>
                      ) : null}
                      <option value="none">{t("initialPositionNone")}</option>
                      <option value="focus">{t("initialPositionFocus")}</option>
                      <option value="coordinates">
                        {t("initialPositionCoordinates")}
                      </option>
                    </select>
                  </div>

                  {initialPositionMode === "focus" ? (
                    <div className="space-y-1">
                      <label
                        className="text-xs font-medium"
                        htmlFor="initial-position-focus"
                      >
                        {t("initialPositionFocusId")}
                      </label>
                      <Input
                        id="initial-position-focus"
                        list="focus-initial-position-options"
                        value={initialFocusId}
                        onChange={(event) => {
                          setInitialFocusId(event.target.value);
                          setInitialSaveState("idle");
                        }}
                        placeholder="BEL_perpetual_neutrality"
                        className="font-mono text-xs"
                      />
                      <datalist id="focus-initial-position-options">
                        {nodes.map((node) => (
                          <option key={node.id} value={node.focus_id} />
                        ))}
                      </datalist>
                    </div>
                  ) : null}

                  {initialPositionMode === "coordinates" ? (
                    <div className="space-y-3">
                      <div className="grid grid-cols-2 gap-2">
                        <div className="space-y-1">
                          <label
                            className="text-xs font-medium"
                            htmlFor="initial-position-x"
                          >
                            X
                          </label>
                          <Input
                            id="initial-position-x"
                            type="number"
                            value={initialX}
                            onChange={(event) => {
                              setInitialX(Number(event.target.value) || 0);
                              setInitialSaveState("idle");
                            }}
                            className="tech-tabular"
                          />
                        </div>
                        <div className="space-y-1">
                          <label
                            className="text-xs font-medium"
                            htmlFor="initial-position-y"
                          >
                            Y
                          </label>
                          <Input
                            id="initial-position-y"
                            type="number"
                            value={initialY}
                            onChange={(event) => {
                              setInitialY(Number(event.target.value) || 0);
                              setInitialSaveState("idle");
                            }}
                            className="tech-tabular"
                          />
                        </div>
                      </div>

                      <div className="space-y-2">
                        <div className="flex items-center justify-between gap-2">
                          <p className="text-xs font-medium">
                            {t("initialPositionOffsets")}
                          </p>
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={() => {
                              setInitialOffsets((offsets) => [
                                ...offsets,
                                { x: 0, y: 0, trigger: "" },
                              ]);
                              setInitialSaveState("idle");
                            }}
                          >
                            <Plus className="h-3.5 w-3.5" aria-hidden />
                            {t("initialPositionAddOffset")}
                          </Button>
                        </div>
                        {initialOffsets.length === 0 ? (
                          <p className="text-xs text-muted-foreground">
                            {t("initialPositionNoOffsets")}
                          </p>
                        ) : null}
                        {initialOffsets.map((offset, index) => (
                          <div
                            key={index}
                            className="space-y-2 rounded-md border border-border bg-muted/30 p-2"
                          >
                            <div className="flex items-center justify-between gap-2">
                              <span className="text-xs font-medium">
                                {t("initialPositionOffsetNumber", {
                                  number: index + 1,
                                })}
                              </span>
                              <Button
                                type="button"
                                size="icon"
                                variant="ghost"
                                onClick={() => {
                                  setInitialOffsets((offsets) =>
                                    offsets.filter((_, i) => i !== index),
                                  );
                                  setInitialSaveState("idle");
                                }}
                                aria-label={t("initialPositionRemoveOffset", {
                                  number: index + 1,
                                })}
                                title={t("initialPositionRemoveOffset", {
                                  number: index + 1,
                                })}
                              >
                                <Trash2 className="h-3.5 w-3.5" aria-hidden />
                              </Button>
                            </div>
                            <div className="grid grid-cols-2 gap-2">
                              <Input
                                type="number"
                                aria-label={t("initialPositionOffsetX", {
                                  number: index + 1,
                                })}
                                value={offset.x}
                                onChange={(event) => {
                                  const value = Number(event.target.value) || 0;
                                  setInitialOffsets((offsets) =>
                                    offsets.map((item, i) =>
                                      i === index
                                        ? { ...item, x: value }
                                        : item,
                                    ),
                                  );
                                  setInitialSaveState("idle");
                                }}
                                className="tech-tabular"
                              />
                              <Input
                                type="number"
                                aria-label={t("initialPositionOffsetY", {
                                  number: index + 1,
                                })}
                                value={offset.y}
                                onChange={(event) => {
                                  const value = Number(event.target.value) || 0;
                                  setInitialOffsets((offsets) =>
                                    offsets.map((item, i) =>
                                      i === index
                                        ? { ...item, y: value }
                                        : item,
                                    ),
                                  );
                                  setInitialSaveState("idle");
                                }}
                                className="tech-tabular"
                              />
                            </div>
                            <textarea
                              value={offset.trigger}
                              onChange={(event) => {
                                const value = event.target.value;
                                setInitialOffsets((offsets) =>
                                  offsets.map((item, i) =>
                                    i === index
                                      ? { ...item, trigger: value }
                                      : item,
                                  ),
                                );
                                setInitialSaveState("idle");
                              }}
                              placeholder='has_dlc = "Arms Against Tyranny"'
                              aria-label={t("initialPositionOffsetTrigger", {
                                number: index + 1,
                              })}
                              className="ui-focus-ring min-h-16 w-full rounded-md border border-input bg-card px-2 py-1 text-xs font-mono"
                            />
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : null}

                  {initialSaveState !== "idle" ? (
                    <p
                      className={`rounded-md border px-2 py-1.5 text-xs ${
                        initialSaveState === "saved"
                          ? "border-success/40 bg-success/5 text-success"
                          : initialSaveState === "error"
                            ? "border-destructive/40 bg-destructive/5 text-destructive"
                            : "border-border bg-muted text-muted-foreground"
                      }`}
                      aria-live="polite"
                    >
                      {initialSaveState === "saving"
                        ? t("initialPositionSaving")
                        : initialSaveState === "saved"
                          ? t("initialPositionSaved")
                          : t("initialPositionSaveError")}
                    </p>
                  ) : null}

                  <Button
                    type="button"
                    size="sm"
                    onClick={() => void handleSaveInitialPosition()}
                    disabled={
                      !initialPositionCanSave || initialSaveState === "saving"
                    }
                  >
                    {t("initialPositionSave")}
                  </Button>
                  {!initialPositionCanSave &&
                  initialPositionMode !== "unresolved" ? (
                    <p className="text-xs text-destructive">
                      {t("initialPositionIncomplete")}
                    </p>
                  ) : null}
                </div>
              ) : null}
            </div>
          ) : null}

          {currentTree && (
            <div className="border-t border-border/40 pt-3">
              <div className="flex items-center justify-between mb-3 gap-2">
                <div className="flex items-baseline gap-2 min-w-0">
                  <h3 className="text-micro font-semibold uppercase tracking-widest text-muted-foreground">
                    {t("focusNodes")}
                  </h3>
                  <span className="bg-muted text-muted-foreground rounded-full px-2 text-xs tech-tabular">
                    {nodeQueryTrimmed
                      ? `${visibleNodes.length}/${sortedNodes.length}`
                      : sortedNodes.length}
                  </span>
                </div>
                <div className="flex items-center gap-1 shrink-0">
                  <Button
                    size="sm"
                    className="text-xs font-semibold"
                    onClick={() => handleCreateNode()}
                  >
                    {t("addNode")}
                  </Button>
                </div>
              </div>
              {sortedNodes.length > 0 && (
                <div className="relative mb-2">
                  <Search
                    className="pointer-events-none absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground"
                    aria-hidden
                  />
                  <Input
                    value={nodeQuery}
                    onChange={(e) => setNodeQuery(e.target.value)}
                    placeholder={t("searchNodesPlaceholder")}
                    aria-label={t("searchNodesPlaceholder")}
                    className="h-8 pl-7 pr-7 text-xs"
                  />
                  {nodeQuery && (
                    <button
                      type="button"
                      onClick={() => setNodeQuery("")}
                      title={t("clearSearch")}
                      aria-label={t("clearSearch")}
                      className="ui-interactive absolute right-1.5 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted-foreground hover:text-foreground"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
              )}
              {sortedNodes.length > 0 && (
                <p className="mb-2 text-micro text-muted-foreground">
                  {t("multiSelectHint")}
                </p>
              )}
              {sortedNodes.length === 0 ? (
                <p className="text-xs text-muted-foreground text-center py-6">
                  {t("noNodesYet")}
                </p>
              ) : (
                <div
                  className="space-y-0.5 outline-none"
                  {...nodeListNav.containerProps}
                  aria-label={t("focusNodes")}
                >
                  {batchGfx.selectedIds.size > 0 && (
                    <div className="mb-2 rounded border border-primary/40 bg-primary/5 px-2 py-1.5 flex items-center gap-2 text-xs">
                      <span className="font-medium">
                        {t("selectedCount", {
                          count: batchGfx.selectedIds.size,
                        })}
                      </span>
                      <button
                        type="button"
                        onClick={batchGfx.openPicker}
                        disabled={batchGfx.progress !== null}
                        className="ml-auto px-2 py-0.5 rounded bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                      >
                        {t("batchApplyGfx")}
                      </button>
                      <button
                        type="button"
                        onClick={batchGfx.clear}
                        className="text-muted-foreground hover:text-foreground ui-interactive"
                      >
                        {t("cancel")}
                      </button>
                    </div>
                  )}
                  {batchGfx.progress && (
                    <div className="mb-2 rounded border border-success/40 bg-success/5 px-2 py-1.5 text-xs">
                      {t("applyingGfx", {
                        gfxName: batchGfx.progress.gfxName,
                        done: batchGfx.progress.done,
                        total: batchGfx.progress.total,
                      })}
                      {batchGfx.progress.failed > 0 && (
                        <span className="text-destructive">
                          {t("applyingFailed", {
                            failed: batchGfx.progress.failed,
                          })}
                        </span>
                      )}
                    </div>
                  )}
                  {visibleNodes.length === 0 && (
                    <p className="py-6 text-center text-xs text-muted-foreground">
                      {t("noMatchingNodes", { query: nodeQuery.trim() })}
                    </p>
                  )}
                  {visibleNodes.map((node) => {
                    const isBatchSelected = batchGfx.selectedIds.has(node.id);
                    const isSelected = selectedNodeId === node.id;

                    const displayName = resolveDisplayName(
                      locale,
                      node.display_name_locs,
                      node.name_en,
                      node.name_zh,
                      node.name || node.focus_id,
                    );
                    return (
                      <div
                        key={node.id}
                        {...nodeListNav.optionProps(isSelected)}
                        className={`group relative flex items-center gap-2 rounded-md py-1.5 pl-2.5 pr-2 cursor-pointer ui-interactive focus-visible:ring-1 focus-visible:ring-ring ${
                          isSelected
                            ? "bg-accent/10"
                            : isBatchSelected
                              ? "bg-success/20"
                              : "hover:bg-muted/60"
                        }`}
                        onClick={(e) => {
                          if (e.shiftKey || e.ctrlKey || e.metaKey) {
                            batchGfx.toggle(node.id, {
                              range: e.shiftKey,

                              sortedIds: visibleNodes.map((n) => n.id),
                            });
                          } else {
                            selectNode(node.id);
                          }
                        }}
                      >
                        {isSelected && (
                          <span className="absolute inset-y-1.5 left-0 w-0.5 rounded-full bg-accent" />
                        )}

                        <div className="min-w-0 flex-1">
                          <p
                            className={`truncate text-xs ${
                              isSelected
                                ? "font-semibold text-accent"
                                : "font-medium text-foreground"
                            }`}
                            title={displayName}
                          >
                            <ReadableScriptedName
                              name={displayName}
                              locale={locale}
                            />
                          </p>
                          <p className="truncate text-micro tech-tabular text-muted-foreground">
                            ({node.x}, {node.y})
                          </p>
                        </div>

                        {isBatchSelected && (
                          <Check className="h-3.5 w-3.5 shrink-0 text-success" />
                        )}

                        <span className="shrink-0 rounded bg-muted px-1.5 text-micro font-medium tech-tabular text-muted-foreground group-hover:hidden">
                          {node.cost}w
                        </span>
                        <button
                          title={t("deleteFocusTitle")}
                          className="hidden shrink-0 rounded p-0.5 text-muted-foreground ui-interactive hover:bg-destructive/10 hover:text-destructive group-hover:block"
                          onClick={(e) => handleDeleteNodeFromList(node.id, e)}
                        >
                          <X className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      <div className="flex-1 flex min-w-0">
        <div className="flex-1 min-w-0 relative flex flex-col">
          {(primaryTrees.length > 0 || sharedPoolTrees.length > 0) && (
            <div className="border-b border-border bg-muted/30 px-2 py-1 flex items-center gap-1 overflow-x-auto shrink-0">
              <span className="text-micro uppercase tracking-wide text-muted-foreground font-mono shrink-0 mr-1">
                {t("multiTree")}
              </span>
              {primaryTrees.map((tree) => {
                const active = selectedTreeId === tree.id;
                return (
                  <button
                    key={tree.id}
                    type="button"
                    onClick={() => {
                      selectTree(tree.id);
                      fetchPreview(projectId, tree.id);
                    }}
                    className={
                      "shrink-0 px-2 py-0.5 text-xs rounded border transition-colors flex items-center gap-1 " +
                      (active
                        ? "bg-primary/10 border-primary text-primary"
                        : "border-border bg-card hover:bg-muted")
                    }
                    title={t("treeTabTitle", {
                      tree: tree.tree_id,
                      count: tree.nodes.length,
                    })}
                  >
                    <Network className="h-3.5 w-3.5 shrink-0" aria-hidden />
                    <span className="font-mono truncate max-w-[140px]">
                      {tree.tree_id}
                    </span>
                    <span className="text-muted-foreground">
                      ({tree.nodes.length})
                    </span>
                  </button>
                );
              })}
              {sharedPoolTrees.length > 0 && (
                <span className="mx-1 text-muted-foreground/40 shrink-0">
                  |
                </span>
              )}
              {sharedPoolTrees.map((tree) => {
                const active = selectedTreeId === tree.id;
                return (
                  <button
                    key={tree.id}
                    type="button"
                    onClick={() => {
                      selectTree(tree.id);
                      fetchPreview(projectId, tree.id);
                    }}
                    className={
                      "shrink-0 px-2 py-0.5 text-xs rounded border transition-colors flex items-center gap-1 " +
                      (active
                        ? "bg-accent/15 border-accent/40 text-foreground"
                        : "border-border bg-card hover:bg-muted")
                    }
                    title={t("sharedPoolTabTitle", {
                      tree: tree.tree_id,
                      nodes: tree.nodes.length,
                      refs: tree.shared_focus_refs?.length ?? 0,
                    })}
                  >
                    <ClipboardList
                      className="h-3.5 w-3.5 shrink-0"
                      aria-hidden
                    />
                    <span className="font-mono truncate max-w-[140px]">
                      {tree.tree_id}
                    </span>
                    <span className="text-muted-foreground">
                      ({tree.nodes.length})
                    </span>
                  </button>
                );
              })}
              <button
                type="button"
                onClick={async () => {
                  try {
                    const tree = await focusTreesApi.create(projectId, {
                      tree_kind: "shared_pool",
                    });
                    await fetchTrees(projectId);
                    selectTree(tree.id);
                  } catch (err) {
                    createTreeFailed(err);
                  }
                }}
                className="shrink-0 px-2 py-0.5 text-xs rounded border border-dashed border-accent/50 text-accent hover:bg-accent/10 ui-interactive"
                title={t("newSharedPoolTitle")}
              >
                {t("addSharedPool")}
              </button>
              <div className="ml-auto flex items-center gap-2">
                {detectedPrefix && currentTree && (
                  <span
                    className="text-xs px-2 py-0.5 rounded border border-success/40 bg-success/5 text-success font-mono whitespace-nowrap"
                    title={t("autoPrefixTitle", {
                      count: nodes.length,
                      prefix: detectedPrefix,
                    })}
                  >
                    {t.rich("autoPrefixChip", {
                      prefix: detectedPrefix,
                      b: (chunks) => <strong>{chunks}</strong>,
                    })}
                  </span>
                )}
                <TreeValidationButton
                  projectId={projectId}
                  kind="focus"
                  onReport={setTreeLintReport}
                />
              </div>
            </div>
          )}
          <div className="flex-1 min-h-0 relative">
            {currentTree ? (
              <FocusTreeGraph
                nodes={visualNodes}
                selectedNodeId={selectedNodeId}
                selectedNodeIds={selectedNodeIds}
                selectedEdgeId={selectedEdgeId}
                projectId={projectId}
                availableIcons={availableIcons}
                iconCacheBust={iconCacheBust}
                treeId={currentTree.id}
                filenameBase={currentTree.tree_id}
                onSelectNodes={handleSelectNodes}
                onSelectEdge={selectEdge}
                onUpdatePosition={handleUpdatePosition}
                onBulkUpdatePositions={handleBulkUpdatePositions}
                onCreateNode={handleCreateNode}
                onDeleteNode={handleDeleteNode}
                onAddPrerequisite={handleAddPrerequisite}
                onRemovePrerequisite={handleRemovePrerequisite}
                onAddMutualExclusion={handleAddMutualExclusion}
                onRemoveMutualExclusion={handleRemoveMutualExclusion}
                onCreateChildNode={handleCreateChildNode}
                onCreateMutexChildNode={handleCreateMutexChildNode}
                onCreateConnectedNode={handleCreateConnectedNode}
              />
            ) : (
              <div className="flex items-center justify-center h-full p-6">
                {error && errorProjectId === projectId && trees.length === 0 ? (
                  <WorkbenchLoadError onRetry={() => fetchTrees(projectId)} />
                ) : !loading && trees.length === 0 ? (
                  <EditorEmptyState
                    variant="hero"
                    className="w-full max-w-md"
                    icon={
                      <FolderTree className="h-10 w-10 text-muted-foreground/60" />
                    }
                    title={t("emptyHeroTitle")}
                    description={t("emptyHeroDesc")}
                    actions={
                      <>
                        <Button size="sm" onClick={handleHeroFirstStep}>
                          {t("emptyHeroFirstStep")}
                        </Button>
                      </>
                    }
                  />
                ) : (
                  <EditorEmptyState
                    className="max-w-xs"
                    icon={
                      <FolderTree className="h-6 w-6 text-muted-foreground/60" />
                    }
                    title={
                      trees.length > 0
                        ? t("noTreeSelectedTitle")
                        : t("noTreeYetTitle")
                    }
                    description={
                      trees.length > 0
                        ? t("noTreeSelectedDesc")
                        : t("noTreeYetDesc")
                    }
                  />
                )}
              </div>
            )}
          </div>
        </div>

        <FocusTreeSidebar
          selectedNode={selectedNode}
          allNodes={nodes}
          projectId={projectId}
          countryTag={countryTag}
          onSaveNode={handleSaveNode}
          onDraftChange={setFocusNodeDraft}
          editorWiring={editorWiring}
          selectedTreeId={selectedTreeId ?? undefined}
          onNodeForked={async (forked) => {
            await fetchTrees(projectId);
            selectNode(forked.id);
          }}
          onIconUploaded={(result) => {
            const uploadedIcon =
              typeof result.icon === "string" ? result.icon : null;
            if (uploadedIcon && selectedTreeId && selectedNodeId) {
              applyUploadedNodeIcon(
                selectedTreeId,
                selectedNodeId,
                uploadedIcon,
              );
            }
            refreshAvailableIcons();
            if (uploadedIcon) {
              setIconCacheBust({ gfxName: uploadedIcon, version: Date.now() });
            }
            if (selectedTreeId)
              fetchPreview(
                projectId,
                selectedTreeId,
                selectedNodeId ?? undefined,
              );
          }}
          onPreviewActiveChange={(active) => {
            setPreviewEnabled(active);
            if (active && selectedTreeId) {
              fetchPreview(
                projectId,
                selectedTreeId,
                selectedNodeId ?? undefined,
              );
            }
          }}
        />

        <FocusIconLibraryDialog
          open={batchGfx.pickerOpen}
          onOpenChange={(o) =>
            o ? batchGfx.openPicker() : batchGfx.closePicker()
          }
          projectId={projectId}
          onSelect={batchGfx.applyGfx}
        />
      </div>
    </div>
  );
}
