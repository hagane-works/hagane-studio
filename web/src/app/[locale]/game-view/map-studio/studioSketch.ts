export const SKETCH_W = 1408;
export const SKETCH_H = 512;

export const CELL = 4;

export type RGB = readonly [number, number, number];

export const TERRAINS = [
  { key: "ocean", legend: [40, 80, 170], look: [44, 82, 122], height: 0 },
  { key: "plains", legend: [160, 210, 100], look: [174, 190, 122], height: 1 },
  { key: "forest", legend: [30, 110, 40], look: [86, 124, 72], height: 1.2 },
  { key: "hills", legend: [200, 160, 90], look: [178, 158, 108], height: 2 },
  { key: "mountain", legend: [120, 100, 90], look: [140, 126, 114], height: 3 },
  { key: "desert", legend: [245, 225, 140], look: [222, 202, 146], height: 1 },
  { key: "marsh", legend: [100, 150, 140], look: [114, 140, 116], height: 0.6 },
  { key: "jungle", legend: [0, 170, 110], look: [60, 120, 80], height: 1.2 },
] as const satisfies readonly {
  key: string;
  legend: RGB;
  look: RGB;
  height: number;
}[];

export type TerrainKey = (typeof TERRAINS)[number]["key"];
export const SEA = 0;

export interface Sketch {
  w: number;
  h: number;

  terrain: Uint8Array;

  owner: Uint8Array;
}

export interface Rect {
  x0: number;
  y0: number;
  x1: number;
  y1: number;
}

export function blankSketch(w = SKETCH_W, h = SKETCH_H): Sketch {
  return { w, h, terrain: new Uint8Array(w * h), owner: new Uint8Array(w * h) };
}

export function cloneSketch(s: Sketch): Sketch {
  return { w: s.w, h: s.h, terrain: s.terrain.slice(), owner: s.owner.slice() };
}

export function unionRect(a: Rect | null, b: Rect | null): Rect | null {
  if (!a) return b;
  if (!b) return a;
  return {
    x0: Math.min(a.x0, b.x0),
    y0: Math.min(a.y0, b.y0),
    x1: Math.max(a.x1, b.x1),
    y1: Math.max(a.y1, b.y1),
  };
}

function hash(x: number, y: number, seed: number): number {
  let h =
    Math.imul(x | 0, 374761393) ^
    Math.imul(y | 0, 668265263) ^
    Math.imul(seed | 0, 2246822519);
  h = Math.imul(h ^ (h >>> 13), 1274126177);
  h ^= h >>> 16;
  return (h >>> 0) / 4294967296;
}

const smooth = (t: number) => t * t * (3 - 2 * t);

export function valueNoise(
  x: number,
  y: number,
  seed: number,
  period = 0,
): number {
  const xi = Math.floor(x);
  const yi = Math.floor(y);
  const fx = smooth(x - xi);
  const fy = smooth(y - yi);
  const x0 = period ? ((xi % period) + period) % period : xi;
  const x1 = period ? (x0 + 1) % period : xi + 1;
  const a = hash(x0, yi, seed);
  const b = hash(x1, yi, seed);
  const c = hash(x0, yi + 1, seed);
  const d = hash(x1, yi + 1, seed);
  return a + (b - a) * fx + (c - a) * fy + (a - b - c + d) * fx * fy;
}

function fbm(
  x: number,
  y: number,
  seed: number,
  period: number,
  octaves: number,
): number {
  let sum = 0;
  let amp = 0.5;
  let total = 0;
  let f = 1;
  for (let o = 0; o < octaves; o++) {
    sum += amp * valueNoise(x * f, y * f, seed + o * 101, period * f);
    total += amp;
    amp *= 0.5;
    f *= 2;
  }
  return sum / total;
}

export function seededRandom(seed: number): () => number {
  let a = seed | 0;
  return () => {
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export interface RandomOptions {
  seed: number;

  land: number;

  continents: number;

  mountains: number;
}

export const RANDOM_DEFAULTS: Omit<RandomOptions, "seed"> = {
  land: 0.3,
  continents: 3,
  mountains: 0.5,
};

const clamp = (v: number, lo: number, hi: number) =>
  Math.min(hi, Math.max(lo, v));

export function randomWorld(
  options: RandomOptions,
  w = SKETCH_W,
  h = SKETCH_H,
): Sketch {
  const seed = options.seed | 0;
  const land = clamp(options.land, 0.1, 0.6);
  const count = clamp(Math.round(options.continents), 1, 6);
  const mountains = clamp(options.mountains, 0, 1);
  const rng = seededRandom(seed);
  const blobs = Array.from({ length: count }, (_, i) => ({
    x: ((i + 0.5 + (rng() - 0.5) * 0.6) / count) * w,
    y: h * (0.3 + rng() * 0.4),
    rx: (w / count) * (0.28 + rng() * 0.2),
    ry: h * (0.16 + rng() * 0.16),
  }));

  const period = 8;
  const perPixel = period / w;
  const elevation = new Float32Array(w * h);
  for (let y = 0; y < h; y++) {
    const pole = Math.min(y, h - 1 - y) / h;
    for (let x = 0; x < w; x++) {
      let continent = 0;
      for (const b of blobs) {
        let dx = Math.abs(x - b.x);
        dx = Math.min(dx, w - dx);
        const dy = y - b.y;
        continent = Math.max(
          continent,
          1 - Math.sqrt((dx / b.rx) ** 2 + (dy / b.ry) ** 2),
        );
      }
      let v =
        continent * 0.85 +
        (fbm(x * perPixel, y * perPixel, seed, period, 5) - 0.5) * 2.2;
      if (pole < 0.07) v -= (0.07 - pole) * 14;
      elevation[y * w + x] = v;
    }
  }

  const sample: number[] = [];
  for (let i = 0; i < elevation.length; i += 7) sample.push(elevation[i]);
  sample.sort((a, b) => a - b);
  const level =
    sample[Math.min(sample.length - 1, Math.floor(sample.length * (1 - land)))];
  let top = level;
  for (const v of elevation) top = Math.max(top, v);
  const span = Math.max(1e-6, top - level);

  const ridged = (x: number, y: number) => {
    let sum = 0;
    let amp = 0.6;
    let f = 2;
    for (let o = 0; o < 4; o++) {
      const v =
        1 -
        Math.abs(
          2 * valueNoise(x * f, y * f, seed + 7 + o * 31, period * f) - 1,
        );
      sum += v * v * amp;
      amp *= 0.5;
      f *= 2;
    }
    return sum;
  };
  const s = blankSketch(w, h);
  const rugged = new Float32Array(w * h).fill(-1);
  const landRugged: number[] = [];
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const i = y * w + x;
      if (elevation[i] < level) continue;
      const height = (elevation[i] - level) / span;
      rugged[i] = ridged(x * perPixel, y * perPixel) + height * 0.35;
      if (i % 5 === 0) landRugged.push(rugged[i]);
    }
  }

  landRugged.sort((a, b) => a - b);
  const share = (q: number) =>
    landRugged[
      Math.min(landRugged.length - 1, Math.floor(landRugged.length * (1 - q)))
    ] ?? Infinity;
  const mountainAt = share(0.02 + 0.1 * mountains);
  const hillsAt = share(0.08 + 0.2 * mountains);
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const i = y * w + x;
      if (rugged[i] < 0) continue;
      const nx = x * perPixel;
      const ny = y * perPixel;

      const latitude =
        Math.abs(y / h - 0.5) * 2 +
        (fbm(nx * 2, ny * 2, seed + 19, period * 2, 3) - 0.5) * 0.45;
      const wet = fbm(nx * 2, ny * 2, seed + 13, period * 2, 3);
      const height = (elevation[i] - level) / span;
      let cls: number;
      if (rugged[i] >= mountainAt) cls = 4;
      else if (rugged[i] >= hillsAt) cls = 3;
      else if (latitude < 0.2 && wet > 0.52) cls = 7;
      else if (latitude > 0.24 && latitude < 0.62 && wet < 0.44) cls = 5;
      else if (height < 0.07 && wet > 0.55) cls = 6;
      else if (wet > 0.53) cls = 2;
      else cls = 1;
      s.terrain[i] = cls;
    }
  }
  return s;
}

export function sampleWorld(w = SKETCH_W, h = SKETCH_H): Sketch {
  const s = blankSketch(w, h);
  const kx = w / 352;
  const ky = h / 128;
  const rect = (
    x: number,
    y: number,
    rw: number,
    rh: number,
    cls: number,
    inside: (cx: number, cy: number) => boolean,
  ) => {
    for (
      let cy = Math.floor(y * ky);
      cy < Math.min(h, Math.ceil((y + rh) * ky));
      cy++
    ) {
      for (
        let cx = Math.floor(x * kx);
        cx < Math.min(w, Math.ceil((x + rw) * kx));
        cx++
      ) {
        if (inside(cx, cy)) s.terrain[cy * w + cx] = cls;
      }
    }
  };
  const ellipse =
    (ex: number, ey: number, rx: number, ry: number) =>
    (cx: number, cy: number) =>
      ((cx + 0.5) / kx - ex) ** 2 / rx ** 2 +
        ((cy + 0.5) / ky - ey) ** 2 / ry ** 2 <=
      1;
  const west = ellipse(90, 60, 70, 40);
  rect(20, 20, 140, 80, 1, west);
  rect(20, 20, 60, 35, 2, west);
  rect(110, 50, 50, 50, 3, west);
  const east = ellipse(260, 64, 72, 44);
  rect(188, 20, 144, 88, 1, east);
  rect(245, 20, 14, 90, 4, east);
  rect(270, 70, 60, 40, 2, east);
  const island = ellipse(176, 100, 12, 9);
  rect(164, 91, 24, 18, 1, island);
  return s;
}

export type Tool =
  "paint" | "terrain" | "fill" | "country" | "countryFill" | "erase";

export interface Brush {
  tool: Tool;

  terrain: number;

  country: number;

  radius: number;

  natural: boolean;
}

function apply(s: Sketch, i: number, brush: Brush): boolean {
  const t = s.terrain[i];
  const o = s.owner[i];
  switch (brush.tool) {
    case "paint":
    case "fill":
      s.terrain[i] = brush.terrain;
      if (brush.terrain === SEA) s.owner[i] = 0;
      break;
    case "terrain":
      if (t !== SEA && brush.terrain !== SEA) s.terrain[i] = brush.terrain;
      break;
    case "country":
    case "countryFill":
      if (t !== SEA) s.owner[i] = brush.country;
      break;
    case "erase":
      s.owner[i] = 0;
      break;
  }
  return s.terrain[i] !== t || s.owner[i] !== o;
}

export function stamp(
  s: Sketch,
  cx: number,
  cy: number,
  brush: Brush,
): Rect | null {
  const r = Math.max(0.5, brush.radius);
  const reach = Math.ceil(r * 1.25) + 1;
  const x0 = Math.max(0, Math.floor(cx - reach));
  const x1 = Math.min(s.w, Math.ceil(cx + reach) + 1);
  const y0 = Math.max(0, Math.floor(cy - reach));
  const y1 = Math.min(s.h, Math.ceil(cy + reach) + 1);
  const ragged = brush.natural && r >= 2;
  const grain = Math.max(2, r * 0.6);
  let changed: Rect | null = null;
  for (let y = y0; y < y1; y++) {
    for (let x = x0; x < x1; x++) {
      const d = Math.hypot(x + 0.5 - cx, y + 0.5 - cy);

      const limit = ragged
        ? r * (0.72 + 0.56 * valueNoise(x / grain, y / grain, 913))
        : r;
      if (d > limit) continue;
      if (apply(s, y * s.w + x, brush))
        changed = unionRect(changed, { x0: x, y0: y, x1: x + 1, y1: y + 1 });
    }
  }
  return changed;
}

export function strokeTo(
  s: Sketch,
  from: [number, number],
  to: [number, number],
  brush: Brush,
): Rect | null {
  const step = Math.max(0.5, brush.radius * 0.35);
  const n = Math.max(
    1,
    Math.ceil(Math.hypot(to[0] - from[0], to[1] - from[1]) / step),
  );
  let changed: Rect | null = null;
  for (let k = 1; k <= n; k++) {
    const t = k / n;
    changed = unionRect(
      changed,
      stamp(
        s,
        from[0] + (to[0] - from[0]) * t,
        from[1] + (to[1] - from[1]) * t,
        brush,
      ),
    );
  }
  return changed;
}

export function fill(
  s: Sketch,
  x: number,
  y: number,
  brush: Brush,
): Rect | null {
  if (x < 0 || y < 0 || x >= s.w || y >= s.h) return null;
  const start = y * s.w + x;
  let same: (i: number) => boolean;
  if (brush.tool === "countryFill") {
    if (s.terrain[start] === SEA) return null;
    const was = s.owner[start];
    if (was === brush.country) return null;
    same = (i) => s.terrain[i] !== SEA && s.owner[i] === was;
  } else {
    const was = s.terrain[start];
    if (was === brush.terrain) return null;
    same = (i) => s.terrain[i] === was;
  }
  const seen = new Uint8Array(s.w * s.h);
  const stack = new Int32Array(s.w * s.h);
  let top = 0;
  stack[top++] = start;
  seen[start] = 1;
  let changed: Rect | null = null;
  let minX = x;
  let maxX = x;
  let minY = y;
  let maxY = y;
  while (top > 0) {
    const i = stack[--top];
    apply(s, i, brush);
    const cx = i % s.w;
    const cy = (i - cx) / s.w;
    if (cx < minX) minX = cx;
    if (cx > maxX) maxX = cx;
    if (cy < minY) minY = cy;
    if (cy > maxY) maxY = cy;
    const next = [
      cx > 0 ? i - 1 : -1,
      cx < s.w - 1 ? i + 1 : -1,
      cy > 0 ? i - s.w : -1,
      cy < s.h - 1 ? i + s.w : -1,
    ];
    for (const j of next) {
      if (j >= 0 && !seen[j] && same(j)) {
        seen[j] = 1;
        stack[top++] = j;
      }
    }
  }
  changed = { x0: minX, y0: minY, x1: maxX + 1, y1: maxY + 1 };
  return changed;
}

export interface Patch {
  rect: Rect;
  before: { terrain: Uint8Array; owner: Uint8Array };
  after: { terrain: Uint8Array; owner: Uint8Array };
}

function cut(s: Sketch, r: Rect): { terrain: Uint8Array; owner: Uint8Array } {
  const w = r.x1 - r.x0;
  const terrain = new Uint8Array(w * (r.y1 - r.y0));
  const owner = new Uint8Array(terrain.length);
  for (let y = r.y0; y < r.y1; y++) {
    terrain.set(
      s.terrain.subarray(y * s.w + r.x0, y * s.w + r.x1),
      (y - r.y0) * w,
    );
    owner.set(s.owner.subarray(y * s.w + r.x0, y * s.w + r.x1), (y - r.y0) * w);
  }
  return { terrain, owner };
}

export function patchOf(
  before: Sketch,
  after: Sketch,
  rect: Rect | null,
): Patch | null {
  if (!rect) return null;
  const a = cut(before, rect);
  const b = cut(after, rect);
  let differs = false;
  for (let i = 0; i < a.terrain.length && !differs; i++)
    differs = a.terrain[i] !== b.terrain[i] || a.owner[i] !== b.owner[i];
  return differs ? { rect, before: a, after: b } : null;
}

export function applyPatch(
  s: Sketch,
  patch: Patch,
  side: "before" | "after",
): Rect {
  const { rect } = patch;
  const cells = patch[side];
  const w = rect.x1 - rect.x0;
  for (let y = rect.y0; y < rect.y1; y++) {
    s.terrain.set(
      cells.terrain.subarray((y - rect.y0) * w, (y - rect.y0 + 1) * w),
      y * s.w + rect.x0,
    );
    s.owner.set(
      cells.owner.subarray((y - rect.y0) * w, (y - rect.y0 + 1) * w),
      y * s.w + rect.x0,
    );
  }
  return rect;
}

export function countryShares(
  s: Sketch,
  countries: number,
): { land: number; owned: number[]; unowned: number } {
  const owned = new Array<number>(countries).fill(0);
  let land = 0;
  let unowned = 0;
  for (let i = 0; i < s.terrain.length; i++) {
    if (s.terrain[i] === SEA) continue;
    land++;
    const o = s.owner[i];
    if (o > 0 && o <= countries) owned[o - 1]++;
    else unowned++;
  }
  return { land, owned, unowned };
}

export function removeCountry(s: Sketch, slot: number): void {
  for (let i = 0; i < s.owner.length; i++) {
    const o = s.owner[i];
    if (o === slot) s.owner[i] = 0;
    else if (o > slot) s.owner[i] = o - 1;
  }
}

export function legendPixels(
  s: Sketch,
  legend: Partial<Record<string, RGB>> = {},
): Uint8ClampedArray<ArrayBuffer> {
  const colours = TERRAINS.map((t) => legend[t.key] ?? t.legend);
  const out = new Uint8ClampedArray(s.w * s.h * 4);
  for (let i = 0; i < s.terrain.length; i++) {
    const c = colours[s.terrain[i]] ?? colours[0];
    out[i * 4] = c[0];
    out[i * 4 + 1] = c[1];
    out[i * 4 + 2] = c[2];
    out[i * 4 + 3] = 255;
  }
  return out;
}

export function ownerPixels(s: Sketch): Uint8ClampedArray<ArrayBuffer> {
  const out = new Uint8ClampedArray(s.w * s.h * 4);
  for (let i = 0; i < s.owner.length; i++) {
    out[i * 4] = s.terrain[i] === SEA ? 0 : s.owner[i];
    out[i * 4 + 3] = 255;
  }
  return out;
}

export interface Look {
  countries: readonly RGB[];

  showCountries: boolean;
}

const shade = (v: number, by: number) =>
  v + by < 0 ? 0 : v + by > 255 ? 255 : v + by;

export function paintSketch(
  s: Sketch,
  out: Uint8ClampedArray,
  rect: Rect,
  look: Look,
): void {
  const { w, h, terrain, owner } = s;
  for (let y = Math.max(0, rect.y0); y < Math.min(h, rect.y1); y++) {
    for (let x = Math.max(0, rect.x0); x < Math.min(w, rect.x1); x++) {
      const i = y * w + x;
      const t = terrain[i];
      const kind = TERRAINS[t] ?? TERRAINS[0];
      let r: number = kind.look[0];
      let g: number = kind.look[1];
      let b: number = kind.look[2];
      const up = y > 0 ? i - w : i;
      const down = y < h - 1 ? i + w : i;
      const left = x > 0 ? i - 1 : i;
      const right = x < w - 1 ? i + 1 : i;
      if (t === SEA) {
        if (
          terrain[up] !== SEA ||
          terrain[down] !== SEA ||
          terrain[left] !== SEA ||
          terrain[right] !== SEA
        ) {
          r = shade(r, 26);
          g = shade(g, 30);
          b = shade(b, 28);
        }
      } else {
        const ul = y > 0 && x > 0 ? i - w - 1 : i;
        const lift =
          (kind.height - (TERRAINS[terrain[ul]] ?? TERRAINS[0]).height) * 14;
        const grain = (hash(x, y, 5) - 0.5) * 10;
        r = shade(r, lift + grain);
        g = shade(g, lift + grain);
        b = shade(b, lift + grain);
        if (
          terrain[up] === SEA ||
          terrain[down] === SEA ||
          terrain[left] === SEA ||
          terrain[right] === SEA
        ) {
          r = shade(r, -34);
          g = shade(g, -34);
          b = shade(b, -30);
        } else if (look.showCountries) {
          const o = owner[i];
          const c = o > 0 ? look.countries[o - 1] : undefined;
          if (c) {
            r = r * 0.45 + c[0] * 0.55;
            g = g * 0.45 + c[1] * 0.55;
            b = b * 0.45 + c[2] * 0.55;
          } else if (((x + y) & 7) === 0) {
            r = shade(r, 48);
            g = shade(g, 48);
            b = shade(b, 48);
          }
          const border = (j: number) => terrain[j] !== SEA && owner[j] !== o;
          if (border(right) || border(down)) {
            r *= 0.35;
            g *= 0.35;
            b *= 0.35;
          }
        }
      }
      out[i * 4] = r;
      out[i * 4 + 1] = g;
      out[i * 4 + 2] = b;
      out[i * 4 + 3] = 255;
    }
  }
}

export function encodeRuns(cells: Uint8Array): string {
  const bytes: number[] = [];
  let i = 0;
  while (i < cells.length) {
    const v = cells[i];
    let n = 1;
    while (i + n < cells.length && cells[i + n] === v && n < 0xffff) n++;
    bytes.push(v, n & 0xff, n >> 8);
    i += n;
  }
  let text = "";
  for (let k = 0; k < bytes.length; k += 0x8000)
    text += String.fromCharCode(...bytes.slice(k, k + 0x8000));
  return btoa(text);
}

export function decodeRuns(text: string, length: number): Uint8Array | null {
  let raw: string;
  try {
    raw = atob(text);
  } catch {
    return null;
  }
  if (raw.length % 3 !== 0) return null;
  const out = new Uint8Array(length);
  let at = 0;
  for (let k = 0; k < raw.length; k += 3) {
    const n = raw.charCodeAt(k + 1) | (raw.charCodeAt(k + 2) << 8);
    if (n === 0 || at + n > length) return null;
    out.fill(raw.charCodeAt(k), at, at + n);
    at += n;
  }
  return at === length ? out : null;
}

export interface Draft {
  sketch: Sketch;
  countries: string[];
  size: "fine" | "medium" | "coarse";
  seed: number;
}

export function draftKey(projectId: number): string {
  return `hoi4.mapStudio.draft.${projectId}`;
}

export function encodeDraft(d: Draft): string {
  return JSON.stringify({
    v: 1,
    w: d.sketch.w,
    h: d.sketch.h,
    terrain: encodeRuns(d.sketch.terrain),
    owner: encodeRuns(d.sketch.owner),
    countries: d.countries,
    size: d.size,
    seed: d.seed,
  });
}

export function decodeDraft(text: string | null): Draft | null {
  if (!text) return null;
  try {
    const raw = JSON.parse(text) as Record<string, unknown>;
    if (raw.v !== 1 || raw.w !== SKETCH_W || raw.h !== SKETCH_H) return null;
    const terrain = decodeRuns(String(raw.terrain), SKETCH_W * SKETCH_H);
    const owner = decodeRuns(String(raw.owner), SKETCH_W * SKETCH_H);
    if (!terrain || !owner) return null;
    const countries = Array.isArray(raw.countries)
      ? raw.countries
          .filter((tag): tag is string => typeof tag === "string")
          .slice(0, 8)
      : [];
    for (let i = 0; i < owner.length; i++)
      if (owner[i] > countries.length || terrain[i] === SEA) owner[i] = 0;
    const size =
      raw.size === "fine" || raw.size === "coarse" ? raw.size : "medium";
    const seed =
      typeof raw.seed === "number" && Number.isFinite(raw.seed) ? raw.seed : 1;
    return {
      sketch: { w: SKETCH_W, h: SKETCH_H, terrain, owner },
      countries,
      size,
      seed,
    };
  } catch {
    return null;
  }
}
