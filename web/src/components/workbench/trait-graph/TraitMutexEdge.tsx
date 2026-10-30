"use client";

import {
  BaseEdge,
  getSmoothStepPath,
  type EdgeProps,
  type Edge,
} from "@xyflow/react";

type TraitMutexEdgeType = Edge<Record<string, never>, "traitMutexEdge">;

export function TraitMutexEdge({
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style,
}: EdgeProps<TraitMutexEdgeType>) {
  const [edgePath] = getSmoothStepPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
    borderRadius: 8,
  });

  return (
    <BaseEdge
      path={edgePath}
      style={{
        stroke: "var(--color-destructive)",
        strokeWidth: 1.5,
        strokeDasharray: "6 3",
        ...style,
      }}
    />
  );
}
