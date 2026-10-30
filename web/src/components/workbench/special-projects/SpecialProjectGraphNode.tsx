"use client";

import { memo } from "react";
import { Handle, Position, type NodeProps, type Node } from "@xyflow/react";

export interface SpecialProjectNodeData {
  dbId: number;
  token: string;
  label: string;
  specialization: string;
  complexity: string;
  [key: string]: unknown;
}

type SpecialProjectNodeType = Node<
  SpecialProjectNodeData,
  "specialProjectNode"
>;

function SpecialProjectGraphNodeComponent({
  data,
  selected,
}: NodeProps<SpecialProjectNodeType>) {
  return (
    <div
      className={`
        relative rounded-lg border-2 shadow-lg
        w-[130px] text-center overflow-hidden
        bg-gradient-to-b from-card to-card/80 text-card-foreground
        transition-[border-color,box-shadow] duration-150
        ${
          selected
            ? "border-primary shadow-primary/30 ring-1 ring-primary/20"
            : "border-border/80 hover:border-muted-foreground/60"
        }
      `}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!w-4 !h-1.5 !rounded-sm !bg-muted-foreground/50 !border-0 !min-w-0 !min-h-0"
      />

      <div className="px-2 pt-2 pb-1">
        <p
          className="text-xs font-semibold leading-tight truncate"
          title={data.token}
        >
          {data.label}
        </p>
      </div>

      <div className="px-2 pb-1.5 space-y-0.5">
        {data.specialization && (
          <span
            className="inline-block text-micro px-1.5 py-0.5 rounded bg-accent/15 text-accent font-mono truncate max-w-full"
            title={data.specialization}
          >
            {data.specialization.replace("specialization_", "")}
          </span>
        )}
        {data.complexity && (
          <div>
            <span className="inline-block text-micro px-1.5 py-0.5 rounded-full bg-primary/15 text-primary font-semibold">
              {data.complexity.replace("sp_complexity.", "")}
            </span>
          </div>
        )}
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-4 !h-1.5 !rounded-sm !bg-muted-foreground/50 !border-0 !min-w-0 !min-h-0"
      />
    </div>
  );
}

export const SpecialProjectGraphNode = memo(SpecialProjectGraphNodeComponent);
