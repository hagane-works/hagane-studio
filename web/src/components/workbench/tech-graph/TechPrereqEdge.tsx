"use client";

import {
  BaseEdge,
  getSmoothStepPath,
  type EdgeProps,
  type Edge,
} from "@xyflow/react";

type TechPrereqEdgeData = {
  prereqId: string;
  targetDbId: number;
  isMainChain?: boolean;
  dimmed?: boolean;
  highlighted?: boolean;
};

type TechPrereqEdge = Edge<TechPrereqEdgeData, "techPrereqEdge">;

export function TechPrereqEdge({
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style,
  markerEnd,
}: EdgeProps<TechPrereqEdge>) {
  const [edgePath] = getSmoothStepPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
    borderRadius: 8,
  });

  return <BaseEdge path={edgePath} style={style} markerEnd={markerEnd} />;
}
