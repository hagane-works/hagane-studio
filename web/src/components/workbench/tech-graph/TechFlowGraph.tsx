"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import {
  ReactFlow,
  Background,
  MiniMap,
  Panel,
  type OnNodesChange,
  type OnConnect,
  type NodeMouseHandler,
  type Node,
  type Edge,
  BackgroundVariant,
  MarkerType,
  type Connection,
  useReactFlow,
  useViewport,
  useStore,
  ReactFlowProvider,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { HelpCircle } from "lucide-react";

import type { Technology } from "@/types";
import { isTechSurfaceEditableLayout } from "@/lib/techSurface";
import {
  techsToRFNodes,
  techsToRFEdges,
  pixelToStoreFields,
  pixelToRowFields,
  getNodePosition,
  parseTechEdgeId,
  buildYearRowScale,
  formatCategoryLabel,
  bfsDoctrineLayout,
  bfsTreeLayout,
  buildTreePositionOverrides,
  pixelToCellFields,
  pixelToCell,
  cellToPixel,
  type GridGeometry,
  type OrphanGroup,
  assignGridsToTechs,
  buildFolderGroupPositions,
  classifyOrphans,
  techsUseStoredRow,
  buildTimelineTicks,
  buildRowBands,
  DOCTRINE_PANEL_WIDTH,
  DOCTRINE_PANEL_GAP,
  NODE_OFFSET_X,
  NODE_OFFSET_Y,
  COLUMN_WIDTH,
  ROW_HEIGHT_PER_YEAR,
  type TechGraphNodeData,
  type TechLayoutMode,
  type TechLayoutClass,
  type TechNodePosition,
  type CategoryColumn,
  type DoctrinePanel,
  type ReachabilityMap,
  type FolderGeometry,
  technologyAtSelectedPlacement,
  resolveTechnologyPlacementIndex,
} from "@/lib/techGraph";
import { applyTechEdgeDisplayState } from "@/lib/techEdgeDisplay";
import { useTechTreeGeometry } from "./useTechTreeGeometry";
import { useBaseModDepKeys } from "@/lib/api/basemod";
import { useTechnologyStore } from "@/lib/store/technologyStore";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { useTechSections } from "@/lib/visualSectionMapping";
import { techNodeTypes, techEdgeTypes } from "./index";
import { TechGraphToolbar } from "./TechGraphToolbar";

function StickyTimelineLabels({
  layout,
  side,
  ticks,
}: {
  layout: TechLayoutMode;
  side: "primary" | "secondary";

  ticks: Array<{ key: string; text: string; pixel: number }>;
}) {
  const { x, y, zoom } = useViewport();
  const isVertical = layout === "vertical";

  if (isVertical) {
    const sideKey: "left" | "right" = side === "primary" ? "left" : "right";
    return (
      <Panel
        position={sideKey === "left" ? "top-left" : "top-right"}
        className="!m-0 !p-0 pointer-events-none"
      >
        <div
          className="relative bg-card/85 border-border/60 backdrop-blur-sm"
          style={{
            width: 56,
            height: "calc(100vh - 220px)",
            borderRightWidth: sideKey === "left" ? 1 : 0,
            borderLeftWidth: sideKey === "right" ? 1 : 0,
          }}
        >
          {ticks.map((tick) => {
            const screenY = tick.pixel * zoom + y;
            if (screenY < -20 || screenY > 2400) return null;
            return (
              <div
                key={tick.key}
                className="absolute font-mono text-micro text-muted-foreground select-none tech-tabular"
                style={{
                  top: `${screenY - 8}px`,
                  [sideKey === "left" ? "right" : "left"]: 8,
                }}
              >
                {tick.text}
              </div>
            );
          })}
        </div>
      </Panel>
    );
  }

  const sideKey: "top" | "bottom" = side === "primary" ? "top" : "bottom";
  return (
    <Panel
      position={sideKey === "top" ? "top-left" : "bottom-left"}
      className="!m-0 !p-0 pointer-events-none !right-0"
    >
      <div
        className="relative bg-card/85 border-border/60 backdrop-blur-sm"
        style={{
          height: 22,
          width: "calc(100vw - 700px)",
          borderBottomWidth: sideKey === "top" ? 1 : 0,
          borderTopWidth: sideKey === "bottom" ? 1 : 0,
        }}
      >
        {ticks.map((tick) => {
          const screenX = tick.pixel * zoom + x;
          if (screenX < -50 || screenX > 4000) return null;
          return (
            <div
              key={tick.key}
              className="absolute font-mono text-micro text-muted-foreground select-none tech-tabular"
              style={{
                left: `${screenX - 18}px`,
                [sideKey === "top" ? "bottom" : "top"]: 3,
              }}
            >
              {tick.text}
            </div>
          );
        })}
      </div>
    </Panel>
  );
}

function OrphanSectionHeader({ groups }: { groups: OrphanGroup[] }) {
  const t = useTranslations("techFlowGraph");
  if (groups.length === 0) return null;
  return (
    <Panel position="bottom-left" className="!m-2 !mb-12">
      <div className="pointer-events-none flex flex-col gap-1">
        {groups.map((g) => (
          <div
            key={g.kind}
            className="rounded-md border border-border bg-card/90 backdrop-blur-sm px-2 py-1
                       text-micro text-muted-foreground select-none whitespace-nowrap shadow-sm"
          >
            {t(g.kind === "no_folder" ? "orphanNoFolder" : "orphanUnmatched", {
              count: g.techs.length,
            })}
          </div>
        ))}
      </div>
    </Panel>
  );
}

function GuiYearLabels({
  geometry,
  offsetY = 0,
}: {
  geometry: FolderGeometry;

  offsetY?: number;
}) {
  const { x, y, zoom } = useViewport();
  if (geometry.yearLabels.length === 0) return null;
  return (
    <Panel
      position="top-left"
      className="!m-0 !p-0 pointer-events-none !inset-0"
    >
      <div className="relative w-full h-full">
        {geometry.yearLabels.map((label, i) => {
          const screenX = (label.x + label.borderX) * zoom + x;
          const screenY = (label.y + label.borderY + offsetY) * zoom + y;
          if (screenX < -120 || screenY < -60) return null;
          const boxW = label.maxWidth * zoom;
          const shift =
            label.align === "center"
              ? boxW / 2
              : label.align === "right"
                ? boxW
                : 0;
          return (
            <div
              key={`${i}:${label.name}`}
              className="absolute font-mono text-micro font-semibold
                         text-muted-foreground/90 select-none tech-tabular whitespace-nowrap"
              style={{
                left: `${screenX + shift}px`,
                top: `${screenY}px`,
                transform:
                  label.align === "center"
                    ? "translateX(-50%)"
                    : label.align === "right"
                      ? "translateX(-100%)"
                      : undefined,
              }}
              title={label.name}
            >
              {label.text}
            </div>
          );
        })}
      </div>
    </Panel>
  );
}

function StickyCategoryHeaders({
  layout,
  columns,
}: {
  layout: TechLayoutMode;
  columns: CategoryColumn[];
}) {
  const { x, y, zoom } = useViewport();

  const locale = useContentLocaleStore((s) => s.locale);
  if (columns.length === 0) return null;
  const isVertical = layout === "vertical";

  return (
    <Panel
      position="top-left"
      className={`!m-0 !p-0 pointer-events-none ${
        isVertical ? "!left-0 !right-0 !w-full !translate-x-0" : ""
      }`}
    >
      <div className={`relative ${isVertical ? "w-full h-7" : "w-32 h-full"}`}>
        {columns.map((col) => {
          const centerCanvas =
            col.startCol * COLUMN_WIDTH +
            (col.width / 2) * COLUMN_WIDTH +
            NODE_OFFSET_X;

          const labelText =
            col.names?.[locale] ||
            col.names?.en ||
            formatCategoryLabel(col.category);

          if (isVertical) {
            const screenX = centerCanvas * zoom + x;
            if (screenX < -100 || screenX > 4000) return null;
            return (
              <div
                key={col.category}
                className="absolute top-0 -translate-x-1/2 px-2 py-0.5 rounded
                           bg-card/85 border border-border/60 backdrop-blur-sm
                           text-xs font-semibold text-foreground/85 select-none
                           shadow-sm whitespace-nowrap"
                style={{ left: `${screenX}px` }}
                title={col.names?.en || col.category}
              >
                {labelText}
              </div>
            );
          }

          const screenY = centerCanvas * zoom + y;
          if (screenY < -50 || screenY > 2400) return null;
          return (
            <div
              key={col.category}
              className="absolute left-2 -translate-y-1/2 px-2 py-0.5 rounded
                         bg-card/85 border border-border/60 backdrop-blur-sm
                         text-xs font-semibold text-foreground/85 select-none
                         shadow-sm whitespace-nowrap max-w-[120px] truncate"
              style={{ top: `${screenY}px` }}
              title={col.names?.en || col.category}
            >
              {labelText}
            </div>
          );
        })}
      </div>
    </Panel>
  );
}

function TechGridOverlay({
  isVertical,
  ghosts,
  yearBands,
  showGrid,
}: {
  isVertical: boolean;

  ghosts: Array<{
    id: string;
    x: number;
    y: number;
    width: number;
    height: number;
    topLeft?: boolean;
  }>;

  yearBands: Array<{
    key: string;
    start: number;
    size: number;
    shade: boolean;
  }> | null;

  showGrid: boolean;
}) {
  const { x, y, zoom } = useViewport();
  const vw = useStore((s) => s.width);
  const vh = useStore((s) => s.height);
  if (!vw || !vh) return null;

  const left = -x / zoom;
  const top = -y / zoom;
  const right = (vw - x) / zoom;
  const bottom = (vh - y) / zoom;

  const colSize = COLUMN_WIDTH;
  const rowSize = ROW_HEIGHT_PER_YEAR;
  const colAnchor = NODE_OFFSET_X;
  const rowAnchor = NODE_OFFSET_Y;

  const idxRange = (lo: number, hi: number, anchor: number, size: number) => ({
    first: Math.floor((lo - anchor) / size) - 1,
    last: Math.ceil((hi - anchor) / size) + 1,
  });
  const boundary = (n: number, anchor: number, size: number) =>
    anchor + (n - 0.5) * size;

  const colAxis = isVertical
    ? idxRange(left, right, colAnchor, colSize)
    : idxRange(top, bottom, colAnchor, colSize);
  const rowAxis = isVertical
    ? idxRange(top, bottom, rowAnchor, rowSize)
    : idxRange(left, right, rowAnchor, rowSize);

  const bandRect = (key: string, start: number, size: number) =>
    isVertical ? (
      <rect key={key} x={left} y={start} width={right - left} height={size} />
    ) : (
      <rect key={key} x={start} y={top} width={size} height={bottom - top} />
    );

  const bands: React.ReactElement[] = [];
  if (yearBands) {
    for (const b of yearBands) {
      if (!b.shade) continue;
      bands.push(bandRect(b.key, b.start, b.size));
    }
  } else {
    for (let m = rowAxis.first; m <= rowAxis.last; m += 1) {
      if (((m % 2) + 2) % 2 !== 0) continue;
      bands.push(bandRect(`b${m}`, boundary(m, rowAnchor, rowSize), rowSize));
    }
  }

  const lines: React.ReactElement[] = [];
  for (let n = colAxis.first; n <= colAxis.last; n += 1) {
    const b = boundary(n, colAnchor, colSize);
    lines.push(
      isVertical ? (
        <line key={`c${n}`} x1={b} y1={top} x2={b} y2={bottom} />
      ) : (
        <line key={`c${n}`} x1={left} y1={b} x2={right} y2={b} />
      ),
    );
  }
  for (let m = rowAxis.first; m <= rowAxis.last; m += 1) {
    const b = boundary(m, rowAnchor, rowSize);
    lines.push(
      isVertical ? (
        <line key={`r${m}`} x1={left} y1={b} x2={right} y2={b} />
      ) : (
        <line key={`r${m}`} x1={b} y1={top} x2={b} y2={bottom} />
      ),
    );
  }

  return (
    <svg
      className="absolute inset-0 h-full w-full"
      style={{ pointerEvents: "none", zIndex: 0 }}
    >
      <g transform={`translate(${x},${y}) scale(${zoom})`}>
        <g fill="var(--color-muted)" opacity={0.35}>
          {bands}
        </g>
        {showGrid && (
          <g
            stroke="var(--color-border)"
            strokeWidth={1 / zoom}
            opacity={0.5}
            shapeRendering="crispEdges"
          >
            {lines}
          </g>
        )}

        {ghosts.map((g) => (
          <rect
            key={g.id}
            x={g.topLeft ? g.x : g.x - g.width / 2}
            y={g.topLeft ? g.y : g.y - g.height / 2}
            width={g.width}
            height={g.height}
            rx={6 / zoom}
            fill="var(--color-accent)"
            fillOpacity={0.12}
            stroke="var(--color-accent)"
            strokeWidth={2 / zoom}
            strokeDasharray={`${6 / zoom},${4 / zoom}`}
          />
        ))}
      </g>
    </svg>
  );
}

function DoctrinePanelHeaders({ panels }: { panels: DoctrinePanel[] }) {
  const { x, zoom } = useViewport();
  if (panels.length === 0) return null;

  return (
    <Panel
      position="top-left"
      className="!m-0 !p-0 pointer-events-none !left-0 !right-0 !w-full !translate-x-0"
    >
      <div className="relative w-full h-9">
        {panels.map((p) => {
          const panelStartX =
            p.panelIndex * (DOCTRINE_PANEL_WIDTH + DOCTRINE_PANEL_GAP);
          const panelCenterX = panelStartX + DOCTRINE_PANEL_WIDTH / 2;
          const screenX = panelCenterX * zoom + x;
          if (screenX < -200 || screenX > 4000) return null;
          return (
            <div
              key={p.category}
              className="absolute top-1 -translate-x-1/2 px-3 py-1 rounded-md
                         bg-card/90 border border-border/70 backdrop-blur-sm
                         text-xs font-semibold text-foreground select-none
                         shadow-md whitespace-nowrap"
              style={{ left: `${screenX}px` }}
            >
              {formatCategoryLabel(p.category)}
            </div>
          );
        })}
      </div>
    </Panel>
  );
}

function DoctrinePanelBackgrounds({
  panels,
  canvasHeight,
}: {
  panels: DoctrinePanel[];
  canvasHeight: number;
}) {
  if (panels.length === 0) return null;
  return (
    <svg
      style={{
        position: "absolute",
        top: 0,
        left: 0,
        width:
          panels.length * (DOCTRINE_PANEL_WIDTH + DOCTRINE_PANEL_GAP) + 100,
        height: canvasHeight,
        pointerEvents: "none",
        zIndex: 0,
      }}
    >
      {panels.map((p) => {
        const x = p.panelIndex * (DOCTRINE_PANEL_WIDTH + DOCTRINE_PANEL_GAP);
        return (
          <rect
            key={p.category}
            x={x}
            y={20}
            width={DOCTRINE_PANEL_WIDTH}
            height={canvasHeight - 40}
            fill="currentColor"
            fillOpacity={0.03}
            stroke="currentColor"
            strokeOpacity={0.15}
            strokeWidth={1}
            strokeDasharray="6,4"
            rx={8}
            ry={8}
            className="text-muted-foreground"
          />
        );
      })}
    </svg>
  );
}

interface TechFlowGraphProps {
  techs: Technology[];
  selectedId: number | null;
  projectId: number;
  iconCacheBust?: number;

  availableTechIcons?: Set<string> | null;
  layout?: TechLayoutMode;

  layoutClass?: TechLayoutClass;

  categoryColumns?: CategoryColumn[];

  folder?: string;

  reachable?: ReachabilityMap | null;

  activeCountryTag?: string | null;

  selectedPlacementIndex?: number;

  layoutMode?: boolean;

  surfaceManaged?: boolean;
  onSelectTech: (techId: number | null) => void;

  onUpdatePosition?: (
    techId: number,
    positionX: number,
    cell: { position_y: number } | null,
    fallbackStartYear: number,

    needsRowBackfill?: boolean,

    placementIndex?: number,
  ) => void;
  onCreateTech?: (data?: {
    start_year?: number;
    position_x?: number;
    position_y?: number;
  }) => void;
  onDeleteTech?: (techId: number) => void;
  onAddPrerequisite?: (techDbId: number, prereqTechId: string) => void;
  onRemovePrerequisite?: (techDbId: number, prereqTechId: string) => void;
}

function TechFlowGraphInner({
  techs,
  selectedId,
  projectId,
  iconCacheBust,
  availableTechIcons = null,
  layout = "vertical",
  layoutClass = "tree",
  categoryColumns = [],
  folder = "",
  reachable = null,
  activeCountryTag = null,
  selectedPlacementIndex = 0,
  layoutMode = false,
  surfaceManaged = false,
  onSelectTech,
  onUpdatePosition,
  onCreateTech,
  onDeleteTech,
  onAddPrerequisite,
  onRemovePrerequisite,
}: TechFlowGraphProps) {
  const editableLayout = isTechSurfaceEditableLayout(
    layoutMode,
    surfaceManaged,
  );
  const t = useTranslations("techFlowGraph");
  const reactFlow = useReactFlow();
  const { screenToFlowPosition } = reactFlow;
  const isDoctrine = layoutClass === "doctrine";

  const graphTechs = useMemo(
    () =>
      techs.map((tech) =>
        tech.id === selectedId
          ? technologyAtSelectedPlacement(
              tech,
              resolveTechnologyPlacementIndex(tech, selectedPlacementIndex),
            )
          : technologyAtSelectedPlacement(tech, 0),
      ),
    [techs, selectedId, selectedPlacementIndex],
  );

  const locale = useContentLocaleStore((s) => s.locale);

  const baseModDepKeys = useBaseModDepKeys();

  const yearScale = useMemo(() => buildYearRowScale(graphTechs), [graphTechs]);

  const canvasFolders = useMemo(() => {
    if (isDoctrine) return [];
    const set = new Set<string>();
    if (folder) set.add(folder);
    for (const t of graphTechs) {
      const f = String(t.folder ?? "");
      if (f) set.add(f);
    }
    return [...set];
  }, [isDoctrine, folder, graphTechs]);

  const { geometries } = useTechTreeGeometry(projectId, canvasFolders);

  const gridOf = useMemo(() => {
    if (geometries.size === 0) return null;
    const out = new Map<number, GridGeometry>();
    for (const [f, geo] of geometries) {
      const inFolder = graphTechs.filter((t) => String(t.folder ?? "") === f);
      if (inFolder.length === 0) continue;
      for (const [id, g] of assignGridsToTechs(inFolder, geo, {
        assignFallback: false,
      })) {
        out.set(id, g);
      }
    }
    return out;
  }, [graphTechs, geometries]);

  const geometryActive = !isDoctrine && geometries.size > 0 && gridOf != null;

  const historyUndoLen = useTechnologyStore((s) => s.historyUndo.length);
  const historyRedoLen = useTechnologyStore((s) => s.historyRedo.length);
  const undo = useTechnologyStore((s) => s.undo);
  const redo = useTechnologyStore((s) => s.redo);
  const canUndo = historyUndoLen > 0;
  const canRedo = historyRedoLen > 0;

  const handleUndo = useCallback(() => {
    if (canUndo) undo(projectId);
  }, [canUndo, undo, projectId]);

  const handleRedo = useCallback(() => {
    if (canRedo) redo(projectId);
  }, [canRedo, redo, projectId]);

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
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [handleUndo, handleRedo]);

  const [dragPositions, setDragPositions] = useState<
    Map<string, { x: number; y: number }>
  >(() => new Map());

  const doctrineLayout = useMemo(() => {
    if (!isDoctrine) return null;
    return bfsDoctrineLayout(graphTechs);
  }, [graphTechs, isDoctrine]);

  const techSections = useTechSections();
  const treeLayout = useMemo(() => {
    if (isDoctrine) return null;
    return bfsTreeLayout(graphTechs, folder, techSections);
  }, [graphTechs, isDoctrine, folder, techSections]);

  const geoLayout = useMemo(() => {
    if (!geometryActive) return null;
    return buildFolderGroupPositions(
      graphTechs,
      geometries,
      yearScale,
      folder ?? "",
    );
  }, [geometryActive, geometries, graphTechs, yearScale, folder]);
  const geoPositions = geoLayout?.positions ?? null;

  const offsetOf = geoLayout?.offsets ?? null;

  const orphanTechs = useMemo(() => {
    if (!geoPositions) return null;
    const orphans = graphTechs.filter((t) => !geoPositions.has(t.id));
    return orphans.length > 0 ? orphans : null;
  }, [geoPositions, graphTechs]);

  const orphanLayout = useMemo(() => {
    if (!orphanTechs) return null;
    return bfsTreeLayout(orphanTechs, folder, techSections);
  }, [orphanTechs, folder, techSections]);

  const orphanGroups = useMemo(
    () => (orphanTechs ? classifyOrphans(orphanTechs, geometries) : []),
    [orphanTechs, geometries],
  );

  const positionOverrides = useMemo<
    Map<number, TechNodePosition> | undefined
  >(() => {
    if (isDoctrine && doctrineLayout) {
      return doctrineLayout.nodePositions;
    }

    if (geoPositions) {
      if (!orphanTechs) {
        return geoPositions;
      }

      const orphanOverrides = buildTreePositionOverrides(
        orphanTechs,
        orphanLayout!,
        layout,
        yearScale,
      );
      return new Map([...geoPositions, ...orphanOverrides]);
    }
    if (treeLayout) {
      return buildTreePositionOverrides(
        graphTechs,
        treeLayout,
        layout,
        yearScale,
      );
    }
    return undefined;
  }, [
    isDoctrine,
    doctrineLayout,
    geoPositions,
    orphanTechs,
    orphanLayout,
    treeLayout,
    graphTechs,
    layout,
    yearScale,
  ]);

  const rfNodesBase = useMemo(
    () =>
      techsToRFNodes(
        graphTechs,
        selectedId,
        projectId,
        iconCacheBust,
        layout,
        positionOverrides,
        reachable,
        activeCountryTag,
        locale,
        baseModDepKeys,
        availableTechIcons,
        editableLayout,
      ),
    [
      graphTechs,
      selectedId,
      projectId,
      iconCacheBust,
      layout,
      positionOverrides,
      reachable,
      activeCountryTag,
      locale,
      baseModDepKeys,
      availableTechIcons,
      editableLayout,
    ],
  );
  const rfEdges = useMemo(() => techsToRFEdges(graphTechs), [graphTechs]);

  const [hoveredEdgeId, setHoveredEdgeId] = useState<string | null>(null);
  const displayEdges = useMemo(
    () =>
      applyTechEdgeDisplayState(rfEdges, {
        selectedTechDbId: selectedId,
        hoveredEdgeId,
      }),
    [rfEdges, selectedId, hoveredEdgeId],
  );
  const handleEdgeMouseEnter = useCallback(
    (_e: React.MouseEvent, edge: Edge) => setHoveredEdgeId(edge.id),
    [],
  );
  const handleEdgeMouseLeave = useCallback(() => setHoveredEdgeId(null), []);

  const measuredRef = useRef<Map<string, { width: number; height: number }>>(
    new Map(),
  );

  const withGeometryOrigin = useCallback(
    (node: Node<TechGraphNodeData>) =>
      geoPositions?.has(Number(node.id))
        ? { ...node, origin: [0, 0] as [number, number] }
        : node,
    [geoPositions],
  );

  const withMeasured = useCallback(
    (node: Node<TechGraphNodeData>) => {
      const m = measuredRef.current.get(node.id);
      return withGeometryOrigin(m ? { ...node, measured: m } : node);
    },
    [withGeometryOrigin],
  );

  const rfNodes = useMemo(() => {
    if (dragPositions.size === 0) return rfNodesBase.map(withMeasured);

    return rfNodesBase.map((node) => {
      const dragPos = dragPositions.get(node.id);
      if (!dragPos) return withMeasured(node);
      const m = measuredRef.current.get(node.id);
      return withGeometryOrigin(
        m
          ? { ...node, position: dragPos, measured: m }
          : { ...node, position: dragPos },
      );
    });
  }, [rfNodesBase, dragPositions, withMeasured, withGeometryOrigin]);

  const timelineTicks = useMemo(
    () => buildTimelineTicks(graphTechs, yearScale),
    [graphTechs, yearScale],
  );

  const yearBands = useMemo(
    () => (techsUseStoredRow(graphTechs) ? buildRowBands(graphTechs) : null),
    [graphTechs],
  );

  const dragGhosts = useMemo(() => {
    if (dragPositions.size === 0) return [];
    const out: Array<{
      id: string;
      x: number;
      y: number;
      width: number;
      height: number;
      topLeft?: boolean;
    }> = [];
    for (const [id, pos] of dragPositions) {
      const size = measuredRef.current.get(id);
      if (!size) continue;
      const grid = gridOf?.get(Number(id)) ?? null;

      const snapped =
        geometryActive && grid
          ? (() => {
              const off = offsetOf?.get(Number(id)) ?? 0;
              const c = cellToPixel(
                pixelToCell(pos.x, pos.y - off, grid),
                grid,
              );
              return { x: c.x, y: c.y + off };
            })()
          : getNodePosition(
              pixelToRowFields(pos.x, pos.y, layout) as unknown as Technology,
              layout,
              undefined,
              true,
            );
      out.push({
        id,
        x: snapped.x,
        y: snapped.y,
        width: size.width,
        height: size.height,

        topLeft: geometryActive && grid != null,
      });
    }
    return out;
  }, [dragPositions, gridOf, geometryActive, layout, offsetOf]);

  const handleNodeDragStart = useCallback(() => {
    setDragPositions((prev) => (prev.size === 0 ? prev : new Map()));
  }, []);

  const handleNodesChange: OnNodesChange<Node<TechGraphNodeData>> = useCallback(
    (changes) => {
      for (const change of changes) {
        if (change.type === "dimensions" && change.dimensions) {
          measuredRef.current.set(change.id, change.dimensions);
        }
        if (!editableLayout) continue;
        if (change.type === "position" && change.position && change.dragging) {
          setDragPositions((prev) => {
            const next = new Map(prev);
            next.set(change.id, change.position!);
            return next;
          });
        }
      }
    },
    [editableLayout],
  );

  const handleNodeDragStop = useCallback(
    (_event: React.MouseEvent, node: Node<TechGraphNodeData>) => {
      const dbId = Number(node.id);
      const grid = gridOf?.get(dbId) ?? null;

      setDragPositions((prev) => (prev.size === 0 ? prev : new Map()));

      if (geometryActive && grid) {
        const { position_x, position_y } = pixelToCellFields(
          node.position.x,
          node.position.y - (offsetOf?.get(dbId) ?? 0),
          grid,
        );
        const current = graphTechs.find((t) => t.id === dbId);

        onUpdatePosition?.(
          dbId,
          position_x,
          { position_y },
          current?.start_year ?? 0,
          !techsUseStoredRow(graphTechs),
          dbId === selectedId ? selectedPlacementIndex : 0,
        );
        return;
      }

      const { position_x, position_y } = pixelToRowFields(
        node.position.x,
        node.position.y,
        layout,
      );

      const current = graphTechs.find((t) => t.id === dbId);
      onUpdatePosition?.(
        dbId,
        position_x,
        { position_y },
        current?.start_year ?? 0,
        !techsUseStoredRow(graphTechs),
        dbId === selectedId ? selectedPlacementIndex : 0,
      );
    },
    [
      layout,
      onUpdatePosition,
      gridOf,
      geometryActive,
      graphTechs,
      offsetOf,
      selectedId,
      selectedPlacementIndex,
    ],
  );

  const fitSignature = `${folder}|${layout}|${rfNodesBase.length}`;
  const lastFitRef = useRef<string>("");
  useEffect(() => {
    if (rfNodesBase.length === 0) return;
    if (lastFitRef.current === fitSignature) return;
    lastFitRef.current = fitSignature;

    const raf = requestAnimationFrame(() => {
      reactFlow.fitView({ padding: 0.2 });
    });
    return () => cancelAnimationFrame(raf);
  }, [fitSignature, rfNodesBase.length, reactFlow]);

  const handleNodeClick: NodeMouseHandler<Node<TechGraphNodeData>> =
    useCallback(
      (_event, node) => {
        onSelectTech(Number(node.id));
      },
      [onSelectTech],
    );

  const handlePaneClick = useCallback(() => {
    onSelectTech(null);
  }, [onSelectTech]);

  const handleConnect: OnConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target) return;
      const sourceId = Number(connection.source);
      const targetId = Number(connection.target);
      const sourceTech = graphTechs.find((t) => t.id === sourceId);
      if (!sourceTech) return;
      onAddPrerequisite?.(targetId, sourceTech.tech_id);
    },
    [graphTechs, onAddPrerequisite],
  );

  const handleEdgesDelete = useCallback(
    (edges: Edge[]) => {
      for (const edge of edges) {
        const parsed = parseTechEdgeId(edge.id);
        if (!parsed) continue;
        const sourceTech = graphTechs.find((t) => t.id === parsed.sourceDbId);
        if (sourceTech) {
          onRemovePrerequisite?.(parsed.targetDbId, sourceTech.tech_id);
        }
      }
    },
    [graphTechs, onRemovePrerequisite],
  );

  const handlePaneDoubleClick = useCallback(
    (event: React.MouseEvent) => {
      const target = event.target as HTMLElement;

      if (
        !target.closest(".react-flow__pane") ||
        target.closest(".react-flow__node")
      )
        return;
      const flowPos = screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      const { start_year, position_x } = pixelToStoreFields(
        flowPos.x,
        flowPos.y,
        layout,
        yearScale,
      );
      if (techsUseStoredRow(graphTechs)) {
        const cell = pixelToRowFields(flowPos.x, flowPos.y, layout);
        onCreateTech?.({
          start_year,
          position_x: cell.position_x,
          position_y: cell.position_y,
        });
        return;
      }
      onCreateTech?.({ start_year, position_x });
    },
    [screenToFlowPosition, onCreateTech, layout, yearScale, graphTechs],
  );

  const handleNodesDelete = useCallback(
    (deletedNodes: Node[]) => {
      for (const node of deletedNodes) {
        onDeleteTech?.(Number(node.id));
      }
    },
    [onDeleteTech],
  );

  const isVertical = layout === "vertical";
  const maxCatSlots = useMemo(() => {
    if (graphTechs.length === 0) return 6;
    return Math.max(6, ...graphTechs.map((t) => t.position_x + 1)) + 1;
  }, [graphTechs]);
  const yearAxisLength = yearScale.axisLength + 120;
  const catAxisLength = maxCatSlots * COLUMN_WIDTH + 200;
  const canvasHeight = isVertical ? yearAxisLength : catAxisLength;

  return (
    <ReactFlow
      nodes={rfNodes}
      edges={displayEdges}
      nodeTypes={techNodeTypes}
      edgeTypes={techEdgeTypes}
      onNodesChange={handleNodesChange}
      onNodeDragStart={editableLayout ? handleNodeDragStart : undefined}
      onNodeDragStop={editableLayout ? handleNodeDragStop : undefined}
      onNodeClick={handleNodeClick}
      onPaneClick={handlePaneClick}
      onConnect={editableLayout ? handleConnect : undefined}
      onEdgesDelete={editableLayout ? handleEdgesDelete : undefined}
      onNodesDelete={editableLayout ? handleNodesDelete : undefined}
      onEdgeMouseEnter={handleEdgeMouseEnter}
      onEdgeMouseLeave={handleEdgeMouseLeave}
      onDoubleClick={editableLayout ? handlePaneDoubleClick : undefined}
      nodesDraggable={editableLayout}
      nodesConnectable={editableLayout}
      selectionOnDrag={editableLayout}
      defaultEdgeOptions={{
        markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16 },
      }}
      fitView
      fitViewOptions={{ padding: 0.2 }}
      deleteKeyCode={editableLayout ? "Delete" : []}
      multiSelectionKeyCode="Shift"
      className="bg-background tech-scrim"
      proOptions={{ hideAttribution: true }}

      nodeOrigin={[0.5, 0.5]}
    >
      {editableLayout && (
        <Background
          variant={BackgroundVariant.Dots}
          gap={[COLUMN_WIDTH, ROW_HEIGHT_PER_YEAR]}
          size={1}
          color="var(--color-border)"
          style={{ opacity: 0.35 }}
        />
      )}

      {isDoctrine ? (
        <>
          <DoctrinePanelBackgrounds
            panels={doctrineLayout?.panels ?? []}
            canvasHeight={canvasHeight}
          />
          <DoctrinePanelHeaders panels={doctrineLayout?.panels ?? []} />
        </>
      ) : geometryActive && geoLayout ? (
        <>
          {(geoLayout?.groups ?? []).map((g) => (
            <GuiYearLabels
              key={`y:${g.folder}`}
              geometry={g.geometry}
              offsetY={g.offsetY}
            />
          ))}
          <OrphanSectionHeader groups={orphanGroups} />
        </>
      ) : (
        <>
          <TechGridOverlay
            isVertical={isVertical}
            ghosts={dragGhosts}
            yearBands={yearBands}
            showGrid={editableLayout}
          />
          <StickyTimelineLabels
            layout={layout}
            side="primary"
            ticks={timelineTicks}
          />
          <StickyTimelineLabels
            layout={layout}
            side="secondary"
            ticks={timelineTicks}
          />
          <StickyCategoryHeaders layout={layout} columns={categoryColumns} />
        </>
      )}

      <MiniMap
        nodeColor="var(--color-primary)"
        maskColor="rgba(0, 0, 0, 0.6)"
        pannable
        zoomable
        style={{ right: 12, bottom: 12 }}
        className="!bg-card !border-border !shadow-md"
      />

      <TechGraphToolbar
        canUndo={canUndo}
        canRedo={canRedo}
        onUndo={handleUndo}
        onRedo={handleRedo}
      />

      <Panel position="bottom-left" className="!m-2 !mb-3">
        <div className="group relative pointer-events-auto">
          <button
            type="button"
            aria-label={t("footerHintAria")}
            className="flex items-center justify-center h-7 w-7 rounded-md bg-card/85 border border-border backdrop-blur-sm text-muted-foreground shadow-sm hover:text-foreground transition-colors"
          >
            <HelpCircle className="h-4 w-4" />
          </button>
          <div className="absolute bottom-0 left-9 hidden group-hover:block whitespace-nowrap px-3 py-1.5 rounded-md bg-card/95 border border-border backdrop-blur-sm text-micro text-muted-foreground font-medium shadow-md">
            {t(editableLayout ? "footerHintLayout" : "footerHintBrowse")}
          </div>
        </div>
      </Panel>
    </ReactFlow>
  );
}

export function TechFlowGraph(props: TechFlowGraphProps) {
  return (
    <ReactFlowProvider>
      <TechFlowGraphInner {...props} />
    </ReactFlowProvider>
  );
}
