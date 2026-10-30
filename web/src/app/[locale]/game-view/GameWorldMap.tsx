"use client";

import { useMemo, useRef, useState } from "react";
import useSWR from "swr";
import {
  WorldMapCanvas,
  type MapFocus,
  type MapUnit,
} from "@/components/workbench/world-map/WorldMapCanvas";
import { loadWorldMap } from "@/components/workbench/world-map/worldMapApi";
import {
  parseWorldMap,
  type Layer,
  type PickScope,
  type WorldMap,
} from "@/components/workbench/world-map/worldMapModel";

export function mapClickTarget(
  map: WorldMap,
  index: number,
  tag: string | null,
): { state: number } | { country: string } | null {
  const province = map.provinces[index];
  if (!index || !province || province.kind !== "land" || !province.state)
    return null;
  const owner = map.states.get(province.state)?.owner ?? "";
  if (!owner) return null;
  if (tag && owner.toUpperCase() === tag.toUpperCase())
    return { state: province.state };
  return { country: owner.toUpperCase() };
}

export function homeProvince(map: WorldMap, tag: string | null): number {
  if (!tag) return 0;
  let best = 0;
  let bestValue = -1;
  map.provinces.forEach((province, index) => {
    if (!index || province.kind !== "land" || !province.state) return;
    if (
      (map.states.get(province.state)?.owner ?? "").toUpperCase() !==
      tag.toUpperCase()
    )
      return;
    const value = map.points.get(province.id)?.value ?? 0;
    if (value > bestValue) {
      best = index;
      bestValue = value;
    }
  });
  return best;
}

export function unitCounters(
  divisions: readonly { location?: number }[] | undefined,
): MapUnit[] {
  const counts = new Map<number, number>();
  for (const division of divisions ?? []) {
    const province = division.location ?? 0;
    if (province > 0) counts.set(province, (counts.get(province) ?? 0) + 1);
  }
  return [...counts].map(([province, count]) => ({ province, count }));
}

export function overGameWindow(
  x: number,
  y: number,
  root: ParentNode = document,
): boolean {
  for (const el of root.querySelectorAll("[data-window-bg]")) {
    const r = el.getBoundingClientRect();
    if (
      r.width > 0 &&
      r.height > 0 &&
      x >= r.left &&
      x <= r.right &&
      y >= r.top &&
      y <= r.bottom
    )
      return true;
  }
  return false;
}

export function useGameWorldMap(
  projectId: number,
  date: string | null,
): WorldMap | null {
  const data = useSWR(
    date ? ["world-map", projectId, date] : null,
    () => loadWorldMap(projectId, date ?? ""),
    { revalidateOnFocus: false },
  );
  return useMemo(
    () => (data.data ? parseWorldMap(data.data) : null),
    [data.data],
  );
}

export function GameWorldMap({
  projectId,
  date,
  tag,
  width,
  height,
  onState,
  onCountry,
  onProvince,
  picked,
  pickScope,
  layer = "political",
  version = 0,
  onPictureShown,
  units,
}: {
  projectId: number;
  date: string;
  tag: string | null;
  width: number;
  height: number;
  onState: (stateId: number) => void;
  onCountry: (tag: string) => void;

  onProvince?: (index: number) => void;

  picked?: number;

  pickScope?: PickScope;

  layer?: Layer;

  version?: number;

  onPictureShown?: (version: number) => void;

  units?: readonly MapUnit[] | null;
}) {
  const map = useGameWorldMap(projectId, date);
  const [clicked, setSelected] = useState(0);
  const selected = onProvince ? (picked ?? 0) : clicked;

  const home = useMemo(() => (map ? homeProvince(map, tag) : 0), [map, tag]);
  const focus = useMemo<MapFocus | null>(
    () => (home ? { province: home, scale: 1, at: 0 } : null),
    [home],
  );

  const pressedAt = useRef<{ x: number; y: number } | null>(null);
  if (!map) return null;
  return (
    <div
      data-game-world-map
      style={{ position: "absolute", left: 0, top: 0, width, height }}
      onPointerDownCapture={(e) => {
        pressedAt.current = { x: e.clientX, y: e.clientY };
      }}
    >
      <WorldMapCanvas
        map={map}
        imageUrl={`/api/go/projects/${projectId}/gui-render/world-map/provinces.png?v=${version}`}
        onIndex={onPictureShown ? () => onPictureShown(version) : undefined}
        layer={layer}
        selected={selected}
        focus={focus}
        onPick={(index) => {
          const at = pressedAt.current;
          if (at && overGameWindow(at.x, at.y)) return;
          if (onProvince) {
            if (index) onProvince(index);
            return;
          }
          const target = mapClickTarget(map, index, tag);
          setSelected(target ? index : 0);
          if (!target) return;
          if ("state" in target) onState(target.state);
          else onCountry(target.country);
        }}
        onHover={() => {}}
        loadingText=""
        quiet
        pickScope={onProvince ? pickScope : undefined}
        units={units}
      />
    </div>
  );
}
