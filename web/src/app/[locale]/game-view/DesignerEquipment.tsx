"use client";

import { useMemo } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiDesignerEquipmentGroup,
  type GuiNode,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "../gui-renderer/GuiCanvas";
import { gameTooltipProps } from "../gui-renderer/GameTooltip";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";
import { matchNamePath } from "./gameNavigation";

const ARCHETYPE_ENTRY = "division_equipment_archetype_entry";
const TYPE_ENTRY = "division_equipment_type_entry";

export const CHECKED = 2;
const NO_HIDDEN: Set<string> = new Set();

const TEXTS = {
  allowed: "REINFORCE_WITH_EQUIPMENT",
  counter: "ALLOWED_EQUIPMENT_OF_TYPE",
} as const;

export function equipmentRows(
  groups: readonly GuiDesignerEquipmentGroup[],
  heights: { archetype: number; type: number },
): {
  kind: "archetype" | "type";
  key: string;
  label: string;
  counter?: string;
  top: number;
}[] {
  const rows: {
    kind: "archetype" | "type";
    key: string;
    label: string;
    counter?: string;
    top: number;
  }[] = [];
  let top = 0;
  for (const group of groups) {
    rows.push({
      kind: "archetype",
      key: group.archetype,
      label: group.label,
      top,
    });
    top += heights.archetype;
    for (const type of group.types) {
      rows.push({
        kind: "type",
        key: type.type,
        label: type.label,
        counter: `${type.allowed}/${type.total}`,
        top,
      });
      top += heights.type;
    }
  }
  return rows;
}

function Stamp({
  entry,
  id,
  texts,
  frames,
  left,
  top,
}: {
  entry: GuiWindow;
  id: string;
  texts: Record<string, string>;
  frames: Record<string, number>;
  left: number;
  top: number;
}) {
  const size = entry.node.size as { width?: number; height?: number } | null;
  const w = Number(size?.width) || 460;
  const h = Number(size?.height) || 30;
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;

    node.position = { x: 0, y: 0 };
    for (const child of node.children ?? []) {
      if (child.name && frames[child.name] !== undefined)
        child.frame = frames[child.name];
    }
    assignIds(node);
    const overrides: Record<string, string> = {};
    for (const [name, text] of Object.entries(texts)) {
      for (const hit of matchNamePath(node, name) as (GuiNode & {
        _id?: string;
      })[]) {
        if (hit._id) overrides[hit._id] = text;
      }
    }
    return { node, overrides };
  }, [entry.node, texts, frames]);
  const edit = useEditCanvas();
  return (
    <div style={{ position: "absolute", left, top, width: w, height: h }}>
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${id}`,
          entry,
          built.node,
          entry.sprites,
          built.overrides,
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

const TYPE_FRAMES = { is_type_allowed_button: CHECKED };
const NO_FRAMES = {};

export function DesignerEquipment({
  projectId,
  rect,
  groups,
}: {
  projectId: number;

  rect: { x: number; y: number; w: number; h: number } | undefined;
  groups: readonly GuiDesignerEquipmentGroup[];
}) {
  const archetype = useTemplateWindow(projectId, ARCHETYPE_ENTRY);
  const type = useTemplateWindow(projectId, TYPE_ENTRY);
  const tips = useSWR<Record<string, string>>(
    ["game-view-loc", projectId, "designer-equipment"],
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(Object.values(TEXTS), { markup: true });
    },
    { revalidateOnFocus: false },
  );
  if (!rect || !archetype.data || !type.data) return null;
  const heightOf = (entry: GuiWindow, fallback: number) =>
    Number((entry.node.size as { height?: number } | null)?.height) || fallback;
  const indent =
    Number((type.data.node.position as { x?: number } | null)?.x) || 0;
  const rows = equipmentRows(groups, {
    archetype: heightOf(archetype.data, 30),
    type: heightOf(type.data, 33),
  });
  const tip = (
    text: string | undefined,
    key: string,
    box: { x: number; y: number; w: number; h: number },
  ) =>
    text ? (
      <div
        key={key}
        {...gameTooltipProps(text, { key })}
        style={{
          position: "absolute",
          left: box.x,
          top: box.y,
          width: box.w,
          height: box.h,
          pointerEvents: "auto",
        }}
      />
    ) : null;

  const child = (name: string) =>
    (type.data?.node.children ?? []).find((node) => node.name === name) as
      (GuiNode & { position?: { x?: number; y?: number } }) | undefined;
  const allowedAt = child("is_type_allowed_button")?.position;
  const counterAt = child("equipment_counter")?.position;
  return (
    <div
      style={{
        position: "absolute",
        left: rect.x,
        top: rect.y,
        width: rect.w,
        height: rect.h,
        overflow: "hidden",
      }}
    >
      {rows.map((row) =>
        row.kind === "archetype" ? (
          <Stamp
            key={`a:${row.key}`}
            entry={archetype.data as GuiWindow}
            id={row.key}
            texts={{ archetype_name: row.label }}
            frames={NO_FRAMES}
            left={0}
            top={row.top}
          />
        ) : (
          <div key={`t:${row.key}`}>
            <Stamp
              entry={type.data as GuiWindow}
              id={row.key}
              texts={{
                type_name: row.label,
                equipment_counter: row.counter ?? "",
              }}
              frames={TYPE_FRAMES}
              left={indent}
              top={row.top}
            />
            {tip(tips.data?.[TEXTS.allowed], `equipment-allowed:${row.key}`, {
              x: indent + (Number(allowedAt?.x) || 5),
              y: row.top + (Number(allowedAt?.y) || 4),
              w: 26,
              h: 26,
            })}
            {tip(tips.data?.[TEXTS.counter], `equipment-counter:${row.key}`, {
              x: indent + (Number(counterAt?.x) || 380),
              y: row.top + (Number(counterAt?.y) || 8),
              w: 50,
              h: 20,
            })}
          </div>
        ),
      )}
    </div>
  );
}
