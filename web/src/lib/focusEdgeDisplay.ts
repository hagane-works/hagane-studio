import type { Edge } from "@xyflow/react";

export const FOCUS_EDGE_NEUTRAL = "var(--color-muted-foreground)";
export const FOCUS_EDGE_MUTEX = "var(--color-destructive)";
export const FOCUS_EDGE_SELECTED_SOURCE = "var(--color-foreground)";

export const FOCUS_GROUP_PALETTE = [
  "var(--color-focus-edge-group-1)",
  "var(--color-focus-edge-group-2)",
  "var(--color-focus-edge-group-3)",
  "var(--color-focus-edge-group-4)",
];

export interface EdgeDisplayState {
  selectedNodeId: number | null;
  selectedEdgeId: string | null;
  hoveredEdgeId: string | null;

  hoveredNodeId?: number | null;
}

export function applyEdgeDisplayState(
  edges: Edge[],
  {
    selectedNodeId,
    selectedEdgeId,
    hoveredEdgeId,
    hoveredNodeId,
  }: EdgeDisplayState,
): Edge[] {
  const selStr = selectedNodeId != null ? String(selectedNodeId) : null;

  const activeNodeStr = hoveredNodeId != null ? String(hoveredNodeId) : selStr;
  const edgeIsSelected = selectedEdgeId != null;

  return edges.map((edge) => {
    const isMutex = edge.type === "mutexEdge";
    const touchesSelected =
      activeNodeStr != null &&
      (edge.source === activeNodeStr || edge.target === activeNodeStr);
    const isHovered = edge.id === hoveredEdgeId;
    const isEdgeSelected = edge.id === selectedEdgeId;

    let stroke = isMutex ? FOCUS_EDGE_MUTEX : FOCUS_EDGE_NEUTRAL;
    let opacity = 1;
    let strokeWidth = 2;

    if (edgeIsSelected) {
      if (isEdgeSelected) {
        stroke = "var(--color-accent)";
        opacity = 1;
        strokeWidth = 4;
      } else {
        opacity = 0.2;
      }
    } else if (activeNodeStr != null) {
      if (!touchesSelected) {
        opacity = 0.2;
      } else if (!isMutex) {
        if (edge.target === activeNodeStr) {
          const gi = (edge.data?.groupIndex as number | undefined) ?? 0;
          stroke = FOCUS_GROUP_PALETTE[gi % FOCUS_GROUP_PALETTE.length];
        } else {
          stroke = FOCUS_EDGE_SELECTED_SOURCE;
        }
      }
    }

    if (isHovered) {
      opacity = 1;
      strokeWidth = 3.5;
    }

    return {
      ...edge,
      style: {
        ...edge.style,
        stroke,
        strokeWidth,
        opacity,
        ...(isMutex && !isEdgeSelected ? { strokeDasharray: "6 3" } : {}),
        ...(isEdgeSelected
          ? { filter: "drop-shadow(0 0 6px rgba(192, 132, 16, 0.8))" }
          : {}),
      },
      data: {
        ...edge.data,
        dimmed: opacity < 1,
        highlighted: isHovered || isEdgeSelected,
        isSelected: isEdgeSelected,
      },
    };
  });
}
