"use client";

import {
  BaseEdge,
  getSmoothStepPath,
  type EdgeProps,
  type Edge,
} from "@xyflow/react";
import type { TraitEdgeKind } from "./traitTreeModel";

export type TraitParentEdgeData = {
  kind: TraitEdgeKind;

  numParentsNeeded?: number | null;

  readOnly: boolean;
};

type TraitParentEdgeType = Edge<TraitParentEdgeData, "traitParentEdge">;

function edgeLabel(data: TraitParentEdgeData | undefined): string | null {
  if (!data) return null;
  if (data.kind === "and") {
    return data.numParentsNeeded && data.numParentsNeeded > 0
      ? `${data.numParentsNeeded}+`
      : "ALL";
  }
  if (data.kind === "all") return "ALL";
  return null;
}

export function TraitParentEdge({
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style,
  markerEnd,
  data,
}: EdgeProps<TraitParentEdgeType>) {
  const [edgePath, labelX, labelY] = getSmoothStepPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
    borderRadius: 8,
  });

  const label = edgeLabel(data);

  return (
    <>
      <BaseEdge
        path={edgePath}
        style={{
          ...(data?.readOnly ? { strokeDasharray: "5 3" } : null),
          ...style,
        }}
        markerEnd={markerEnd}
      />
      {label && (
        <foreignObject
          x={labelX - 14}
          y={labelY - 8}
          width={28}
          height={16}
          className="pointer-events-none"
        >
          <div className="flex items-center justify-center w-full h-full">
            <span className="text-micro font-semibold text-accent bg-background/80 px-1 rounded">
              {label}
            </span>
          </div>
        </foreignObject>
      )}
    </>
  );
}
