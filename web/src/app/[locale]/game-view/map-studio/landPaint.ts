import type { WorldMap } from "@/components/workbench/world-map/worldMapModel";
import type { Neighbours } from "./ownerPaint";
import { TERRAINS } from "./studioSketch";

export const LAND_MAX_PIXELS = 2048 * 1024;

export const LAND_PROVINCE_PIXELS = 400;

export type LandEdits = ReadonlyMap<number, number>;

export function strokePixels(
  width: number,
  height: number,
  from: [number, number],
  to: [number, number],
  radius: number,
): number[] {
  const out = new Set<number>();
  const r = Math.max(0.5, radius);
  const steps = Math.max(
    1,
    Math.ceil(
      Math.hypot(to[0] - from[0], to[1] - from[1]) / Math.max(1, r / 2),
    ),
  );
  for (let k = 0; k <= steps; k++) {
    const cx = from[0] + ((to[0] - from[0]) * k) / steps;
    const cy = from[1] + ((to[1] - from[1]) * k) / steps;
    const x0 = Math.max(0, Math.floor(cx - r));
    const x1 = Math.min(width - 1, Math.floor(cx + r));
    const y0 = Math.max(0, Math.floor(cy - r));
    const y1 = Math.min(height - 1, Math.floor(cy + r));
    for (let y = y0; y <= y1; y++) {
      for (let x = x0; x <= x1; x++) {
        const dx = x + 0.5 - cx;
        const dy = y + 0.5 - cy;
        if (dx * dx + dy * dy <= r * r) out.add(y * width + x);
      }
    }
  }
  return [...out];
}

export function landArea(edits: LandEdits): number {
  let n = 0;
  for (const v of edits.values()) if (v > 0) n++;
  return n;
}

export function landSketch(
  edits: LandEdits,
  width: number,
  height: number,
): {
  rect: [number, number, number, number];
  rgba: Uint8ClampedArray<ArrayBuffer>;
} | null {
  let x0 = Infinity;
  let y0 = Infinity;
  let x1 = -Infinity;
  let y1 = -Infinity;
  for (const [p, v] of edits) {
    if (v <= 0) continue;
    const x = p % width;
    const y = (p - x) / width;
    x0 = Math.min(x0, x);
    y0 = Math.min(y0, y);
    x1 = Math.max(x1, x);
    y1 = Math.max(y1, y);
  }
  if (x0 === Infinity) return null;
  x0 = Math.max(0, x0 - 2);
  y0 = Math.max(0, y0 - 2);
  x1 = Math.min(width - 1, Math.max(x1 + 2, x0 + 3));
  y1 = Math.min(height - 1, Math.max(y1 + 2, y0 + 3));
  x0 = Math.max(0, Math.min(x0, x1 - 3));
  y0 = Math.max(0, Math.min(y0, y1 - 3));
  const w = x1 - x0 + 1;
  const h = y1 - y0 + 1;
  const rgba = new Uint8ClampedArray(w * h * 4);
  const sea = TERRAINS[0].legend;
  for (let i = 0; i < w * h; i++)
    rgba.set([sea[0], sea[1], sea[2], 255], i * 4);
  for (const [p, v] of edits) {
    if (v <= 0 || v >= TERRAINS.length) continue;
    const x = (p % width) - x0;
    const y = Math.floor(p / width) - y0;
    const c = TERRAINS[v].legend;
    rgba.set([c[0], c[1], c[2], 255], (y * w + x) * 4);
  }
  return { rect: [x0, y0, w, h], rgba };
}

export function unreachedSeas(
  map: WorldMap,
  neighbours: Neighbours,
  marked: readonly number[],
): number[] {
  const chosen = new Set(marked);
  const seen = new Set<number>();
  const out: number[] = [];
  for (const first of marked) {
    if (seen.has(first)) continue;
    seen.add(first);
    const group = [first];
    let reaches = false;
    for (let k = 0; k < group.length; k++) {
      for (const next of neighbours.get(group[k]) ?? []) {
        if (chosen.has(next)) {
          if (!seen.has(next)) {
            seen.add(next);
            group.push(next);
          }
        } else if (map.provinces[next]?.kind === "sea") {
          reaches = true;
        }
      }
    }
    if (!reaches)
      out.push(Math.min(...group.map((i) => map.provinces[i]?.id ?? Infinity)));
  }
  return out.sort((a, b) => a - b);
}
