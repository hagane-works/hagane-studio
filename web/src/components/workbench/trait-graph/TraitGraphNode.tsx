"use client";

import { memo } from "react";
import { useTranslations } from "use-intl";
import { Handle, Position, type NodeProps, type Node } from "@xyflow/react";

export interface TraitGraphNodeData {
  traitId: string;
  label: string;
  dbId: number;
  guiRow: number;
  column: number;

  columnIsAuto: boolean;
  traitType: string;

  externalParents: string[];

  hasAdvancedParents: boolean;
  [key: string]: unknown;
}

type TraitGraphNodeType = Node<TraitGraphNodeData, "traitNode">;

function TraitGraphNodeComponent({
  data,
  selected,
}: NodeProps<TraitGraphNodeType>) {
  const t = useTranslations("customTraitsPanel");

  return (
    <div
      className={`
        relative rounded-lg border-2 shadow-lg
        w-[120px] text-center overflow-hidden
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
          title={data.label}
        >
          {data.label}
        </p>
        <p
          className="text-micro font-mono text-muted-foreground truncate"
          title={data.traitId}
        >
          {data.traitId}
        </p>
      </div>

      <div className="flex items-center justify-center gap-1 pb-2">
        <span
          className="inline-block text-micro px-1.5 py-0.5 rounded-full bg-primary/15 text-primary font-semibold tech-tabular"
          title={t("nodeRowColTitle")}
        >
          r{data.guiRow}·c{data.column}
        </span>
        {data.columnIsAuto && (
          <span
            className="inline-block text-micro px-1 py-0.5 rounded bg-muted text-muted-foreground font-semibold"
            title={t("nodeAutoColumnTitle")}
          >
            {t("nodeAutoColumnBadge")}
          </span>
        )}
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-4 !h-1.5 !rounded-sm !bg-muted-foreground/50 !border-0 !min-w-0 !min-h-0"
      />

      {data.externalParents.length > 0 && (
        <div
          className="absolute top-0.5 left-0.5 text-micro leading-none px-1 py-0.5 rounded bg-accent/15 text-accent font-semibold shadow-sm"
          title={t("nodeExternalParentsTitle", {
            list: data.externalParents.join(", "),
          })}
        >
          {t("nodeExternalParentsBadge", {
            count: data.externalParents.length,
          })}
        </div>
      )}

      {data.hasAdvancedParents && (
        <div
          className="absolute top-0.5 right-0.5 text-micro leading-none px-1 py-0.5 rounded bg-muted text-muted-foreground font-semibold shadow-sm"
          title={t("nodeAdvancedParentsTitle")}
        >
          {t("nodeAdvancedParentsBadge")}
        </div>
      )}
    </div>
  );
}

export const TraitGraphNode = memo(TraitGraphNodeComponent);
