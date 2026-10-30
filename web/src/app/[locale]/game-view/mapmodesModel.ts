import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import type { Layer } from "@/components/workbench/world-map/worldMapModel";
import { matchNamePath } from "./gameNavigation";

export const MAPMODES_WINDOW = "MapmodesInterface_Ingame";

export type MapModeId =
  | "land"
  | "naval"
  | "air"
  | "operatives"
  | "supply"
  | "terrain"
  | "resistance"
  | "compliance"
  | "resource"
  | "infrastructure"
  | "factions";

export const MAP_MODE_SLOTS: Record<
  string,
  { mode: MapModeId; frame: number }
> = {
  mapmode_button_1: { mode: "land", frame: 1 },
  mapmode_button_2: { mode: "naval", frame: 3 },
  mapmode_button_3: { mode: "air", frame: 2 },
  mapmode_button_4: { mode: "operatives", frame: 4 },
  mapmode_button_5: { mode: "supply", frame: 6 },
  mapmode_button_6: { mode: "terrain", frame: 16 },
  mapmode_button_7: { mode: "resistance", frame: 7 },
  mapmode_button_8: { mode: "compliance", frame: 8 },
  mapmode_button_9: { mode: "resource", frame: 9 },
  mapmode_button_10: { mode: "infrastructure", frame: 18 },
  mapmode_button_11: { mode: "factions", frame: 11 },
};

export const MAP_MODE_LAYER: Partial<Record<MapModeId, Layer>> = {
  land: "political",
  naval: "region",
  air: "region",
};

export function mapModeSlotOf(name: string | null | undefined): string | null {
  const slot = (name ?? "").replace(/_s$/, "");
  return MAP_MODE_SLOTS[slot] ? slot : null;
}

const HIDDEN_PARTS = [
  "3_button_mapmode",
  "social_view_interface_window",
  "tutorial_highlight_default_mapmode",
  "logistics_tutorial_glow",
  "logistics_tutorial_glow_1",
  "logistics_tutorial_glow_2",
];

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function mapmodesWindow(
  win: GuiWindow,
  selected: MapModeId,
): { win: GuiWindow; hidden: string[]; shown: string[] } {
  const node = structuredClone(win.node) as GuiNode;
  const hidden: string[] = [];
  const walk = (at: GuiNode) => {
    for (const child of at.children ?? []) {
      const name = child.name ?? "";
      const slot = mapModeSlotOf(name);
      if (slot && (name === slot || name === `${slot}_s`)) {
        child.frame = MAP_MODE_SLOTS[slot].frame;
        const lit = MAP_MODE_SLOTS[slot].mode === selected;
        if (name === slot ? lit : !lit) hidden.push(idOf(child));
      } else if (
        name.endsWith("_s_bg") &&
        MAP_MODE_SLOTS[name.slice(0, -"_s_bg".length)]
      ) {
        if (MAP_MODE_SLOTS[name.slice(0, -"_s_bg".length)].mode !== selected)
          hidden.push(idOf(child));
      }
      walk(child);
    }
  };
  walk(node);
  for (const part of HIDDEN_PARTS)
    hidden.push(...matchNamePath(node, part).map(idOf));
  const shown = matchNamePath(node, "4_button_mapmode")
    .map(idOf)
    .filter(Boolean);
  return { win: { ...win, node }, hidden: hidden.filter(Boolean), shown };
}
