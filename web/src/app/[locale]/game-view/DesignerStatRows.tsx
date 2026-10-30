"use client";

import { useMemo } from "react";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";

export const STAT_ENTRY = "designer_stat_entry";

const NO_HIDDEN: Set<string> = new Set();

function StatRow({
  entry,
  label,
  value,
  left,
  top,
}: {
  entry: GuiWindow;
  label: string;
  value: string;
  left: number;
  top: number;
}) {
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;
    assignIds(node);
    const texts: Record<string, string> = {};
    for (const child of node.children ?? []) {
      const id = (child as GuiNode & { _id?: string })._id;
      if (!id) continue;
      if (child.name === "label") texts[id] = label;
      if (child.name === "count") texts[id] = value;
    }
    return { node, texts };
  }, [entry.node, label, value]);
  const edit = useEditCanvas();
  const size = entry.node.size as { width?: number; height?: number } | null;
  const w = Number(size?.width) || 155;
  const h = Number(size?.height) || 20;
  return (
    <div style={{ position: "absolute", left, top, width: w, height: h }}>
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${label}`,
          entry,
          built.node,
          entry.sprites,
          built.texts,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={w}
        baseH={h}
      />
    </div>
  );
}

export function DesignerStatRows({
  projectId,
  rows,
  left,
  top,
  pitch = 20,
}: {
  projectId: number;
  rows: readonly { key: string; label: string; value: string }[];
  left: number;
  top: number;
  pitch?: number;
}) {
  const entry = useTemplateWindow(projectId, STAT_ENTRY);
  if (!entry.data) return null;
  return (
    <>
      {rows.map((row, at) => (
        <StatRow
          key={row.key}
          entry={entry.data as GuiWindow}
          label={row.label}
          value={row.value}
          left={left}
          top={top + at * pitch}
        />
      ))}
    </>
  );
}
