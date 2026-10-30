import type { Edge } from "@xyflow/react";

export const TECH_EDGE_EMPHASIS_WIDTH = 3.5;

export const TECH_EDGE_DIM_OPACITY = 0.2;

export interface TechEdgeDisplayState {
  selectedTechDbId: number | null;
  hoveredEdgeId: string | null;
}

export function applyTechEdgeDisplayState(
  edges: Edge[],
  { selectedTechDbId, hoveredEdgeId }: TechEdgeDisplayState,
): Edge[] {
  const sel = selectedTechDbId != null ? String(selectedTechDbId) : null;

  return edges.map((edge) => {
    const touchesSelected =
      sel != null && (edge.source === sel || edge.target === sel);
    const isHovered = edge.id === hoveredEdgeId;
    const baseWidth =
      typeof edge.style?.strokeWidth === "number" ? edge.style.strokeWidth : 2;

    let opacity = 1;
    let strokeWidth = baseWidth;
    let animated = false;

    if (sel != null) {
      if (!touchesSelected) {
        opacity = TECH_EDGE_DIM_OPACITY;
      } else {
        strokeWidth = Math.max(baseWidth, TECH_EDGE_EMPHASIS_WIDTH);
        animated = true;
      }
    }

    if (isHovered) {
      opacity = 1;
      strokeWidth = Math.max(strokeWidth, TECH_EDGE_EMPHASIS_WIDTH);
      animated = true;
    }

    return {
      ...edge,
      animated,
      style: {
        ...edge.style,
        strokeWidth,
        opacity,
      },
      data: {
        ...edge.data,
        dimmed: opacity < 1,
        highlighted: touchesSelected || isHovered,
      },
    };
  });
}
