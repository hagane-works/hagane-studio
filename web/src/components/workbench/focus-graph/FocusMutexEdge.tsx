"use client";

import { memo } from "react";
import {
  BaseEdge,
  getStraightPath,
  type EdgeProps,
  type Edge,
} from "@xyflow/react";

type MutexEdgeData = {
  sourceFocusId: string;
  targetFocusId: string;
  dimmed?: boolean;
  highlighted?: boolean;
  isSelected?: boolean;
};

type MutexEdge = Edge<MutexEdgeData, "mutexEdge">;

function FocusMutexEdgeComponent({
  sourceX,
  sourceY,
  targetX,
  targetY,
  style,
  data,
}: EdgeProps<MutexEdge>) {
  const [edgePath, labelX, labelY] = getStraightPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
  });

  const isSelected = data?.isSelected;
  const edgeStyle = {
    ...style,
    strokeWidth: isSelected
      ? 4
      : data?.highlighted
        ? 3
        : style?.strokeWidth || 2,
    opacity: isSelected
      ? 1
      : data?.dimmed
        ? 0.3
        : data?.highlighted
          ? 1
          : style?.opacity || 0.7,
    filter: isSelected
      ? "drop-shadow(0 0 6px rgba(192, 132, 16, 0.8))"
      : data?.highlighted
        ? "drop-shadow(0 0 4px rgba(220, 38, 38, 0.5))"
        : "none",
  };

  return (
    <>
      <BaseEdge path={edgePath} style={edgeStyle} interactionWidth={24} />
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

function areMutexEdgePropsEqual(
  prev: EdgeProps<MutexEdge>,
  next: EdgeProps<MutexEdge>,
): boolean {
  return (
    prev.sourceX === next.sourceX &&
    prev.sourceY === next.sourceY &&
    prev.targetX === next.targetX &&
    prev.targetY === next.targetY &&
    prev.style?.stroke === next.style?.stroke &&
    prev.style?.strokeWidth === next.style?.strokeWidth &&
    prev.style?.opacity === next.style?.opacity &&
    prev.data?.dimmed === next.data?.dimmed &&
    prev.data?.highlighted === next.data?.highlighted &&
    prev.data?.isSelected === next.data?.isSelected
  );
}

export const FocusMutexEdge = memo(
  FocusMutexEdgeComponent,
  areMutexEdgePropsEqual,
);
