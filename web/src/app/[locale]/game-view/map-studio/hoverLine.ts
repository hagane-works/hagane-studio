import type { WorldMap } from "@/components/workbench/world-map/worldMapModel";
import { landOf, ownerOf, stateOwnerOf, type Overrides } from "./ownerPaint";
import { terrainLand, terrainOf, type TerrainOverrides } from "./terrainPaint";

export interface HoverWords {
  say: (
    key: "hovering" | "hoverPainted",
    values: Record<string, string>,
  ) => string;

  ownerLabel: (key: string) => string;
  terrainName: (terrain: string) => string;
}

export function describeHover(
  map: WorldMap | null,
  hovered: number,
  mode: string,
  overrides: Overrides,
  terrains: TerrainOverrides,
  words: HoverWords,
): string | null {
  if (!map || !hovered) return null;

  const province = map.provinces[hovered];
  if (!province) return null;
  const name = map.states.get(province.state)?.name ?? `#${province.state}`;
  if (mode === "terrain") {
    if (!terrainLand(map, [hovered]).length) return null;
    const now = terrainOf(map, terrains, hovered);
    return province.terrain !== now
      ? words.say("hoverPainted", {
          state: name,
          from: words.terrainName(province.terrain),
          to: words.terrainName(now),
        })
      : words.say("hovering", { state: name, owner: words.terrainName(now) });
  }
  if (!landOf(map, [hovered]).length) return null;
  const was = stateOwnerOf(map, hovered);
  const now = ownerOf(map, overrides, hovered);
  return was !== now
    ? words.say("hoverPainted", {
        state: name,
        from: words.ownerLabel(was),
        to: words.ownerLabel(now),
      })
    : words.say("hovering", { state: name, owner: words.ownerLabel(now) });
}
