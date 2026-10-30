"use client";

import { useCallback, useMemo } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  BackgroundVariant,
  MarkerType,
  type Node,
  type Edge,
  type Connection,
  type OnConnect,
  type OnNodesChange,
  type NodeMouseHandler,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import type { CustomTrait } from "@/types";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";
import { TraitGraphNode, type TraitGraphNodeData } from "./TraitGraphNode";
import { TraitParentEdge } from "./TraitParentEdge";
import { TraitMutexEdge } from "./TraitMutexEdge";
import { TraitGraphToolbar } from "./TraitGraphToolbar";
import {
  buildTraitTreeNodes,
  buildTraitTreeEdges,
  traitsToRFNodes,
  traitEdgesToRFEdges,
  pixelToGrid,
  withAnyParents,
  readRawBlocks,
  parseTokenList,
  TRAIT_CELL_WIDTH,
  TRAIT_CELL_HEIGHT,
} from "./traitTreeModel";

const traitNodeTypes = { traitNode: TraitGraphNode };
const traitEdgeTypes = {
  traitParentEdge: TraitParentEdge,
  traitMutexEdge: TraitMutexEdge,
};

export interface TraitTreePatch {
  extra_fields?: Record<string, unknown>;
  raw_extras?: Record<string, unknown>;
}

interface TraitTreeGraphProps {
  traits: CustomTrait[];
  selectedTraitDbId: number | null;
  onSelectTrait: (dbId: number | null) => void;
  onPersist: (dbId: number, patch: TraitTreePatch) => Promise<void>;
  locale: ContentLocale;
}

function TraitTreeGraphInner({
  traits,
  selectedTraitDbId,
  onSelectTrait,
  onPersist,
  locale,
}: TraitTreeGraphProps) {
  const t = useTranslations("customTraitsPanel");

  const treeNodes = useMemo(
    () => buildTraitTreeNodes(traits, locale),
    [traits, locale],
  );
  const treeEdges = useMemo(() => buildTraitTreeEdges(treeNodes), [treeNodes]);
  const baseRFNodes = useMemo(
    () => traitsToRFNodes(treeNodes, selectedTraitDbId),
    [treeNodes, selectedTraitDbId],
  );
  const rfEdges: Edge[] = useMemo(
    () => traitEdgesToRFEdges(treeEdges, treeNodes),
    [treeEdges, treeNodes],
  );

  const handleNodesChange: OnNodesChange<Node<TraitGraphNodeData>> =
    useCallback(() => {}, []);

  const traitByDbId = useMemo(
    () => new Map(traits.map((tr) => [tr.id, tr])),
    [traits],
  );

  const handleNodeDragStop = useCallback(
    async (_e: React.MouseEvent, node: Node<TraitGraphNodeData>) => {
      const trait = traitByDbId.get(Number(node.id));
      if (!trait) return;
      const { column, row } = pixelToGrid(node.position.x, node.position.y);
      const extras = (trait.extra_fields ?? {}) as Record<string, unknown>;
      const prevRow = typeof extras.gui_row === "number" ? extras.gui_row : -1;
      const prevCol =
        typeof extras.gui_column === "number" ? extras.gui_column : -1;
      if (prevRow === row && prevCol === column) return;
      try {
        await onPersist(trait.id, {
          extra_fields: { ...extras, gui_row: row, gui_column: column },
        });
      } catch (err) {
        toast.error(
          t("canvasSaveFailed", {
            msg: err instanceof Error ? err.message : String(err),
          }),
        );
      }
    },
    [traitByDbId, onPersist, t],
  );

  const handleNodeClick: NodeMouseHandler<Node<TraitGraphNodeData>> =
    useCallback((_e, node) => onSelectTrait(Number(node.id)), [onSelectTrait]);

  const handleConnect: OnConnect = useCallback(
    async (connection: Connection) => {
      if (!connection.source || !connection.target) return;
      if (connection.source === connection.target) return;
      const parent = traitByDbId.get(Number(connection.source));
      const child = traitByDbId.get(Number(connection.target));
      if (!parent || !child) return;

      const raw = (child.raw_extras ?? {}) as Record<string, unknown>;
      const current = readRawBlocks(raw, "any_parent").flatMap(parseTokenList);
      const next = withAnyParents(raw, [...current, parent.trait_id]);
      if (next === raw) return;
      try {
        await onPersist(child.id, { raw_extras: next });
      } catch (err) {
        toast.error(
          t("canvasSaveFailed", {
            msg: err instanceof Error ? err.message : String(err),
          }),
        );
      }
    },
    [traitByDbId, onPersist, t],
  );

  const handleEdgesDelete = useCallback(
    async (deleted: Edge[]) => {
      for (const edge of deleted) {
        const child = traitByDbId.get(Number(edge.target));
        const parent = traitByDbId.get(Number(edge.source));
        if (!child || !parent) continue;
        const raw = (child.raw_extras ?? {}) as Record<string, unknown>;
        const current = readRawBlocks(raw, "any_parent").flatMap(
          parseTokenList,
        );
        const next = withAnyParents(
          raw,
          current.filter((tok) => tok !== parent.trait_id),
        );
        if (next === raw) continue;
        try {
          await onPersist(child.id, { raw_extras: next });
        } catch (err) {
          toast.error(
            t("canvasSaveFailed", {
              msg: err instanceof Error ? err.message : String(err),
            }),
          );
        }
      }
    },
    [traitByDbId, onPersist, t],
  );

  const isValidConnection = useCallback(
    (c: Connection | Edge) => !!c.source && !!c.target && c.source !== c.target,
    [],
  );

  if (treeNodes.length === 0) {
    return (
      <div className="flex h-full items-center justify-center px-8">
        <p className="text-xs text-muted-foreground text-center max-w-sm leading-relaxed">
          {t("canvasEmpty")}
        </p>
      </div>
    );
  }

  return (
    <ReactFlow
      nodes={baseRFNodes}
      edges={rfEdges}
      nodeTypes={traitNodeTypes}
      edgeTypes={traitEdgeTypes}
      onNodesChange={handleNodesChange}
      onNodeDragStop={handleNodeDragStop}
      onNodeClick={handleNodeClick}
      onPaneClick={() => onSelectTrait(null)}
      onConnect={handleConnect}
      onEdgesDelete={handleEdgesDelete}
      isValidConnection={isValidConnection}
      defaultEdgeOptions={{
        markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16 },
      }}

      nodesDraggable
      deleteKeyCode="Delete"
      fitView
      fitViewOptions={{ padding: 0.2 }}
      className="bg-background"
      proOptions={{ hideAttribution: true }}
    >
      <Background
        variant={BackgroundVariant.Lines}
        gap={[TRAIT_CELL_WIDTH, TRAIT_CELL_HEIGHT]}
        lineWidth={1}
        color="var(--color-border)"
        style={{ opacity: 0.4 }}
      />
      <TraitGraphToolbar />
    </ReactFlow>
  );
}

export function TraitTreeGraph(props: TraitTreeGraphProps) {
  return (
    <ReactFlowProvider>
      <TraitTreeGraphInner {...props} />
    </ReactFlowProvider>
  );
}
