import { useMemo } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiCountryStats,
  type GuiNode,
  type GuiPopularity,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import type { TextSource } from "./editModel";
import { matchNamePath } from "./gameNavigation";
import { keySource } from "./textSources";
import { patchTree } from "./topbarLayout";

export const POLITICS = "countrypoliticsview";

const FACTION_NAME_AT = { x: 40, y: 14 };

export interface PoliticsModel {
  win: GuiWindow;

  hidden: Set<string>;

  shown: Set<string>;
  textOverrides: Record<string, string>;

  sources: Record<string, TextSource>;
}

function idOf(node: GuiNode): string {
  return (node as GuiNode & { _id?: string })._id ?? "";
}

export function politicsExtraSprites(
  stats: GuiCountryStats | undefined,
): string[] {
  const icon = stats?.faction?.icon;
  return icon ? [icon] : [];
}

export function buildPoliticsModel({
  win,
  stats,
  extraSprites,
}: {
  win: GuiWindow;
  stats: GuiCountryStats | undefined;
  extraSprites: Readonly<Record<string, GuiSprite>>;
}): PoliticsModel {
  const root = win.node;
  const hidden = new Set<string>();
  const shown = new Set<string>();
  const textOverrides: Record<string, string> = {};
  const sources: Record<string, TextSource> = {};
  const patches: Record<string, Partial<GuiNode>> = {};
  const sprites: Record<string, GuiSprite> = { ...win.sprites };
  const ids = (namePath: string) =>
    matchNamePath(root, namePath).map(idOf).filter(Boolean);
  const write = (
    namePath: string,
    text: string,
    source?: TextSource | null,
  ) => {
    for (const id of ids(namePath)) {
      textOverrides[id] = text;
      if (source) sources[id] = source;
    }
  };

  write(
    "ideology",
    stats?.politics?.ideology ?? "",
    keySource(stats?.politics?.ruling_party, "allCountries"),
  );
  write(
    "elections",
    stats?.politics?.elections ?? "",
    keySource(stats?.politics?.elections_key, "everywhere"),
  );

  if (stats?.popularities?.length) {
    for (const id of ids("political_pie_chart/pol_view_bg")) hidden.add(id);
  }

  const faction = stats?.faction;
  if (faction) {
    for (const id of ids("faction/no_faction")) hidden.add(id);
    for (const id of ids("faction/in_faction")) shown.add(id);
    write(
      "in_faction/faction_name",
      faction.name,
      keySource(faction.name_key, "everywhere"),
    );
    const nameNode = matchNamePath(root, "in_faction/faction_name")[0];
    const line =
      (nameNode?.font && win.fonts[nameNode.font]?.bitmap?.line) || 18;
    for (const id of ids("in_faction/faction_name")) {
      patches[id] = {
        text: faction.name,
        position: {
          x: FACTION_NAME_AT.x,
          y: FACTION_NAME_AT.y - Math.round(line / 2),
        },
      };
    }

    const icon = faction.icon;
    if (icon && extraSprites[icon]?.resolved) {
      sprites[icon] = extraSprites[icon];
      for (const id of ids("in_faction/faction_icon"))
        patches[id] = { ...patches[id], sprite: icon };
    }
  }

  return {
    win: { ...win, node: patchTree(root, patches), sprites },
    hidden,
    shown,
    textOverrides,
    sources,
  };
}

const NO_SPRITES: Readonly<Record<string, GuiSprite>> = {};

export function usePolitics({
  win,
  projectId,
  stats,
}: {
  win: GuiWindow;
  projectId: number;
  stats: GuiCountryStats | undefined;
}): PoliticsModel | null {
  const active = win.name === POLITICS;
  const names = useMemo(
    () => (active ? politicsExtraSprites(stats) : []),
    [active, stats],
  );
  const extra = useSWR<Record<string, GuiSprite>>(
    names.length
      ? ["game-view-sprites", projectId, "politics", names.join(",")]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(names);
    },
    { revalidateOnFocus: false },
  );
  return useMemo(
    () =>
      active
        ? buildPoliticsModel({
            win,
            stats,
            extraSprites: extra.data ?? NO_SPRITES,
          })
        : null,
    [active, win, stats, extra.data],
  );
}

export const PIE_RADIUS = 27;

export interface PieSlice {
  party: string;
  d: string;
  fill: string;
}

export function pieSlicePaths(
  pie: readonly GuiPopularity[],
  cx: number,
  cy: number,
  r: number = PIE_RADIUS,
): PieSlice[] {
  const total = pie.reduce((sum, slice) => sum + Math.max(0, slice.value), 0);
  if (total <= 0) return [];
  const at = (angle: number) =>
    `${round(cx + r * Math.cos(angle))} ${round(cy - r * Math.sin(angle))}`;
  const out: PieSlice[] = [];
  let start = 0;
  for (const slice of pie) {
    if (slice.value <= 0) continue;
    const sweep = (slice.value / total) * 2 * Math.PI;
    const fill = `rgb(${slice.color[0]}, ${slice.color[1]}, ${slice.color[2]})`;
    if (sweep >= 2 * Math.PI - 1e-9) {
      out.push({
        party: slice.party,
        fill,
        d: `M ${round(cx - r)} ${round(cy)} a ${r} ${r} 0 1 0 ${2 * r} 0 a ${r} ${r} 0 1 0 ${-2 * r} 0 Z`,
      });
    } else {
      const end = start + sweep;

      out.push({
        party: slice.party,
        fill,
        d: `M ${round(cx)} ${round(cy)} L ${at(start)} A ${r} ${r} 0 ${sweep > Math.PI ? 1 : 0} 0 ${at(end)} Z`,
      });
    }
    start += sweep;
  }
  return out;
}

function round(value: number): number {
  return Math.round(value * 100) / 100;
}
