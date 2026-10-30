"use client";

import { memo } from "react";
import {
  BaseEdge,
  getSmoothStepPath,
  type EdgeProps,
  type Edge,
} from "@xyflow/react";
import { useFocusInGamePreview } from "./inGamePreviewContext";

type PrereqEdgeData = {
  groupIndex: number;
  isOrGroup: boolean;
  prereqFocusId: string;
  targetNodeId: number;
  dimmed?: boolean;
  highlighted?: boolean;
  isSelected?: boolean;
};

type PrereqEdge = Edge<PrereqEdgeData, "prereqEdge">;

function FocusPrereqEdgeComponent({
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style,
  markerEnd,
  data,
}: EdgeProps<PrereqEdge>) {
  const [edgePath, labelX, labelY] = getSmoothStepPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
    borderRadius: 8,
  });

  const isSelected = data?.isSelected;

  const inGame = useFocusInGamePreview();
  const edgeStyle = {
    ...style,
    stroke:
      inGame && !isSelected && !data?.highlighted
        ? "var(--color-ingame-gold)"
        : style?.stroke,
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
        ? "drop-shadow(0 0 4px rgba(59, 130, 246, 0.5))"
        : "none",
  };

  return (
    <>
      <BaseEdge
        path={edgePath}
        style={edgeStyle}
        markerEnd={markerEnd}
        interactionWidth={24}
      />
      {data?.isOrGroup && (
        <foreignObject
          x={labelX - 10}
          y={labelY - 8}
          width={20}
          height={16}
          className={`pointer-events-none ${data?.dimmed ? "opacity-20" : ""}`}
        >
          <div className="flex items-center justify-center w-full h-full">
            <span
              className={`text-micro font-semibold px-1 rounded ${inGame ? "text-ingame-gold bg-ingame-band/85" : "text-accent bg-background/80"}`}
            >
              OR
            </span>
          </div>
        </foreignObject>
      )}
    </>
  );
}

function arePrereqEdgePropsEqual(
  prev: EdgeProps<PrereqEdge>,
  next: EdgeProps<PrereqEdge>,
): boolean {
  return (
    prev.sourceX === next.sourceX &&
    prev.sourceY === next.sourceY &&
    prev.targetX === next.targetX &&
    prev.targetY === next.targetY &&
    prev.sourcePosition === next.sourcePosition &&
    prev.targetPosition === next.targetPosition &&
    prev.markerEnd === next.markerEnd &&
    prev.style?.stroke === next.style?.stroke &&
    prev.style?.strokeWidth === next.style?.strokeWidth &&
    prev.style?.opacity === next.style?.opacity &&
    prev.data?.isOrGroup === next.data?.isOrGroup &&
    prev.data?.groupIndex === next.data?.groupIndex &&
    prev.data?.dimmed === next.data?.dimmed &&
    prev.data?.highlighted === next.data?.highlighted &&
    prev.data?.isSelected === next.data?.isSelected
  );
}

export const FocusPrereqEdge = memo(
  FocusPrereqEdgeComponent,
  arePrereqEdgePropsEqual,
);
