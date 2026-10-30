"use client";

import { useMemo } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiCountryStats,
  type GuiNode,
  type GuiText,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { matchNamePath } from "./gameNavigation";
import { useEditCanvas } from "./editContext";
import { FieldHit } from "./FieldHit";

export const TECHNOLOGY = "countrytechnologyview";
const ROW = "research_slot_entry";

const ROW_W = 500;
const ROW_H = 100;

export const BASE_RESEARCH_SLOTS = 2;

const EMPTY_SLOT_HIDDEN = [
  "empty_research_slot_glow",
  "technology",
  "designer",
  "equipment",
  "carrier_fighter",
  "year",
  "eta",
  "research_progressbar",
] as const;

function idOf(node: GuiNode): string {
  return (node as GuiNode & { _id?: string })._id ?? "";
}

export const UNUSED_SLOT = "UNUSED_SLOT";

export function emptySlotRow(entry: GuiWindow): {
  node: GuiNode;
  hidden: Set<string>;
} {
  const node = structuredClone(entry.node) as GuiNode;
  assignIds(node);
  for (const box of matchNamePath(node, "title")) box.text = UNUSED_SLOT;
  const hidden = new Set<string>();
  for (const name of EMPTY_SLOT_HIDDEN) {
    for (const target of matchNamePath(node, name)) hidden.add(idOf(target));
  }
  return { node, hidden };
}

export function researchSlotCount(stats: GuiCountryStats | undefined): number {
  const slots = stats?.research_slots;
  return typeof slots === "number" && slots > 0 ? slots : BASE_RESEARCH_SLOTS;
}

export function ResearchLayer({
  win,
  projectId,
  stats,
  away = false,
  onOpenTree,
}: {
  win: GuiWindow;
  projectId: number;
  stats?: GuiCountryStats;

  away?: boolean;

  onOpenTree?: () => void;
}) {
  const edit = useEditCanvas();
  const active = win.name === TECHNOLOGY && !away;
  const entry = useTemplateWindow(projectId, active ? ROW : null);
  const loc = useSWR<Record<string, string>>(
    active ? ["game-view-loc", projectId, "UNUSED_SLOT"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(["UNUSED_SLOT"]);
    },
    { revalidateOnFocus: false },
  );
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const row = useMemo(
    () => (entry.data ? emptySlotRow(entry.data) : null),
    [entry.data],
  );

  const slotWin = useMemo<GuiWindow | null>(() => {
    if (!entry.data) return null;
    const title: GuiText = {
      display: loc.data?.UNUSED_SLOT ?? "",
      kind: "loc",
    };
    return {
      ...entry.data,
      texts: { ...entry.data.texts, [UNUSED_SLOT]: title },
    };
  }, [entry.data, loc.data]);

  const grid = rects["name:research_slots_grid"];
  if (!active || !row || !slotWin || !grid) return null;
  const count = researchSlotCount(stats);
  return (
    <div
      style={{
        position: "absolute",
        left: grid.x,
        top: grid.y,
        pointerEvents: "none",
      }}
    >
      {Array.from({ length: count }, (_, at) => (
        <div
          key={at}
          style={{ position: "absolute", left: 0, top: at * ROW_H }}
        >
          <GuiCanvas
            node={row.node}
            {...edit(`${ROW}:${at}`, slotWin, row.node, slotWin.sprites)}
            texts={slotWin.texts}
            fonts={slotWin.fonts}
            hidden={row.hidden}
            isolateId={null}
            transparent
            baseW={ROW_W}
            baseH={ROW_H}
          />
          {onOpenTree && (
            <FieldHit
              x={0}
              y={0}
              w={ROW_W}
              h={ROW_H}
              label={`${loc.data?.[UNUSED_SLOT] || UNUSED_SLOT} ${at + 1}`}
              onClick={onOpenTree}
              appearance="inspect"
            />
          )}
        </div>
      ))}
    </div>
  );
}
