import type {
  GuiCountryStats,
  GuiNode,
  GuiSprite,
  GuiWindow,
} from "@/lib/guiRenderApi";
import type { GameTooltipInput } from "../gui-renderer/tooltipText";
import { SPEEDS, TOPBAR_SPRITE_SWAPS } from "./gameNavigation";
import {
  SHORT_MONTHS,
  decisionsTakeable,
  fuelReadout,
  threatFrame,
  topbarBars,
  topbarValues,
  type FuelReadout,
} from "./topbarStats";
import { topbarTooltips, type Loc } from "./topbarTooltips";
import {
  findWithParent,
  hiddenExcept,
  nodesNamed,
  patchTree,
} from "./topbarLayout";

export interface TopbarUi {
  paused: boolean;

  speed: number;

  litButton: string | null;

  music: number;
}

export const IDLE_TOPBAR: TopbarUi = {
  paused: true,
  speed: 1,
  litButton: null,
  music: 0,
};

export const LAYER_DRAWN = [
  "industrial_ratio_bar",
  "fuel_ratio_bar",
  "supply_ratio_bar",
  "convoys_ratio_bar",
  "fuel_value",
  "achievements_button",
  "dismissed_alerts_button",
] as const;

export const PAUSED_BOX = "pause_button_date_paused";

export interface TopbarModel {
  win: GuiWindow;

  hidden: Set<string>;
  textOverrides: Record<string, string>;
  tooltipFor: (node: GuiNode) => GameTooltipInput;

  clock: {
    hidden: Set<string>;
    pausedBoxId: string;
    sprite: string | null;
  } | null;
  bars: Record<string, number>;
  fuel: FuelReadout | null;
}

export interface TopbarInput {
  win: GuiWindow;
  stats?: GuiCountryStats;
  date?: string | null;
  ui: TopbarUi;
  loc: Loc;

  extraSprites: Readonly<Record<string, GuiSprite>>;
}

function idOf(node: GuiNode): string {
  return (node as GuiNode & { _id?: string })._id ?? "";
}

export function topbarExtraSprites(
  stats: GuiCountryStats | undefined,
): string[] {
  const names = new Set<string>(Object.values(TOPBAR_SPRITE_SWAPS));
  if (stats?.faction?.icon) names.add(stats.faction.icon);
  return [...names].sort();
}

function slide(node: GuiNode, t: number): { x: number; y: number } | null {
  const show = node.raw?.show_position as
    { x?: unknown; y?: unknown } | undefined;
  if (!show) return null;
  const from = {
    x: Number(node.position?.x) || 0,
    y: Number(node.position?.y) || 0,
  };
  const to = { x: Number(show.x) || 0, y: Number(show.y) || 0 };
  return {
    x: Math.round(from.x + (to.x - from.x) * t),
    y: Math.round(from.y + (to.y - from.y) * t),
  };
}

export function buildTopbarModel({
  win,
  stats,
  date,
  ui,
  loc,
  extraSprites,
}: TopbarInput): TopbarModel {
  const root = win.node;
  const patches: Record<string, Partial<GuiNode>> = {};
  const hidden = new Set<string>();
  const byName = (name: string) => nodesNamed(root, name);
  const patch = (name: string, fields: Partial<GuiNode>) => {
    for (const node of byName(name))
      patches[idOf(node)] = { ...patches[idOf(node)], ...fields };
  };
  const hide = (name: string) => {
    for (const node of byName(name)) hidden.add(idOf(node));
  };

  for (const name of LAYER_DRAWN) hide(name);

  const sprites: Record<string, GuiSprite> = { ...win.sprites };
  for (const [button, sprite] of Object.entries(TOPBAR_SPRITE_SWAPS)) {
    if (!extraSprites[sprite]?.resolved) continue;
    sprites[sprite] = extraSprites[sprite];
    patch(button, { sprite });
  }

  const emblem = stats?.faction?.icon;
  if (emblem && extraSprites[emblem]?.resolved) {
    sprites[emblem] = extraSprites[emblem];
    patch("faction_icon", { sprite: emblem });
  } else {
    hide("faction_icon");
  }
  if (!stats?.faction) hide("initiative_text");

  if (!(decisionsTakeable(stats) > 0)) {
    hide("decisionview_amount_to_take_bg");
    hide("decisionview_amount_to_take_items");
  }
  if (!((stats?.nukes ?? 0) > 0)) hide("nukes");

  const fuel = stats ? fuelReadout(stats, loc) : null;
  if (fuel) patch("fuel_icon", { frame: fuel.frame });

  patch("threat_button", {
    type: "icontype",
    ...(stats?.world_tension !== undefined
      ? { frame: threatFrame(stats.world_tension) }
      : {}),
  });

  for (let pip = 1; pip <= SPEEDS; pip += 1)
    patch(`button_speedstep${pip}`, { frame: pip <= ui.speed ? 2 : 1 });
  if (ui.litButton) patch(ui.litButton, { frame: 2 });

  hide(ui.paused ? "pause_button_date" : PAUSED_BOX);

  for (const drawer of byName("toggle_music_commands_win")) {
    const at = slide(drawer, ui.music);
    if (at) patches[idOf(drawer)] = { ...patches[idOf(drawer)], position: at };
  }

  const node = patchTree(root, patches);

  let clock: TopbarModel["clock"] = null;
  const box = findWithParent(node, PAUSED_BOX);
  if (box?.parent && ui.paused) {
    const siblings = box.parent.children ?? [];
    const after = siblings.slice(siblings.indexOf(box.node) + 1);
    const keep = new Set(after.map(idOf).filter(Boolean));
    if (keep.size) {
      for (const id of keep) hidden.add(id);
      clock = {
        hidden: hiddenExcept(node, keep),
        pausedBoxId: idOf(box.node),
        sprite: box.node.sprite,
      };
    }
  }

  const values = stats
    ? topbarValues(
        stats,
        date,
        SHORT_MONTHS.map((m) => loc[m] || m),
      )
    : {};
  const textOverrides: Record<string, string> = {};
  for (const [name, value] of Object.entries(values)) {
    for (const target of nodesNamed(node, name))
      textOverrides[idOf(target)] = value;
  }

  const tips = topbarTooltips(node, stats, loc);
  const tooltipFor = (target: GuiNode): GameTooltipInput => {
    const name = target.name ?? "";
    return tips.has(name) ? (tips.get(name) ?? null) : undefined;
  };

  return {
    win: { ...win, node, sprites },
    hidden,
    textOverrides,
    tooltipFor,
    clock,
    bars: stats ? topbarBars(stats) : {},
    fuel,
  };
}
