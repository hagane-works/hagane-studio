"use client";

import {
  BaseEdge,
  getStraightPath,
  type EdgeProps,
  type Edge,
} from "@xyflow/react";

type TechXorEdgeData = {
  sourceTechId: string;
  targetTechId: string;
  dimmed?: boolean;
};

type XorEdge = Edge<TechXorEdgeData, "techXorEdge">;

export function TechXorEdge({
  sourceX,
  sourceY,
  targetX,
  targetY,
  style,
  data,
}: EdgeProps<XorEdge>) {
  const [edgePath, labelX, labelY] = getStraightPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
  });

  return (
    <>
      <BaseEdge path={edgePath} style={style} />
      <foreignObject
        x={labelX - 8}
        y={labelY - 8}
        width={16}
        height={16}
        className={`pointer-events-none ${data?.dimmed ? "opacity-20" : ""}`}
      >
        <div className="flex items-center justify-center w-full h-full">
          <span className="text-micro font-semibold text-destructive">✕</span>
        </div>
      </foreignObject>
    </>
  );
}
