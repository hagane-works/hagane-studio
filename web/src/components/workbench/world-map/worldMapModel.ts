export interface WorldMapPayload {
  provinces: [number, number, number, number, string, number, number, number][];

  states: [number, string, string][];

  regions: [number, string][];

  points?: [number, number, string][];
  continents: string[];

  colors: Record<string, [number, number, number]>;
}

export type ProvinceKind = "land" | "sea" | "lake";
const KINDS: ProvinceKind[] = ["land", "sea", "lake"];

export interface Province {
  id: number;
  rgb: number;
  kind: ProvinceKind;
  coastal: boolean;
  terrain: string;
  continent: number;
  state: number;
  region: number;
}

export interface WorldMap {
  provinces: Province[];
  indexOfColor: Map<number, number>;
  states: Map<number, { name: string; owner: string }>;
  regions: Map<number, string>;

  points: Map<number, { value: number; name: string }>;
  continents: string[];
  colors: Record<string, [number, number, number]>;

  provinceOwners?: ReadonlyMap<number, string>;

  provinceTerrains?: ReadonlyMap<number, string>;

  provinceFlags?: ReadonlySet<number>;
}

export function parseWorldMap(payload: WorldMapPayload): WorldMap {
  const provinces: Province[] = [
    {
      id: 0,
      rgb: -1,
      kind: "land",
      coastal: false,
      terrain: "",
      continent: 0,
      state: 0,
      region: 0,
    },
  ];
  const indexOfColor = new Map<number, number>();
  for (const [
    id,
    rgb,
    kind,
    coastal,
    terrain,
    continent,
    state,
    region,
  ] of payload.provinces) {
    indexOfColor.set(rgb, provinces.length);
    provinces.push({
      id,
      rgb,
      kind: KINDS[kind] ?? "land",
      coastal: coastal === 1,
      terrain,
      continent,
      state,
      region,
    });
  }
  return {
    provinces,
    indexOfColor,
    states: new Map(
      payload.states.map(([id, name, owner]) => [id, { name, owner }]),
    ),
    regions: new Map(payload.regions),
    points: new Map(
      (payload.points ?? []).map(([province, value, name]) => [
        province,
        { value, name },
      ]),
    ),
    continents: payload.continents,
    colors: payload.colors,
  };
}

export function pixelIndex(
  rgba: Uint8ClampedArray,
  indexOfColor: Map<number, number>,
): Uint16Array {
  const out = new Uint16Array(rgba.length / 4);
  let lastRGB = -1;
  let lastIndex = 0;
  for (let p = 0, i = 0; p < out.length; p++, i += 4) {
    const rgb = (rgba[i] << 16) | (rgba[i + 1] << 8) | rgba[i + 2];
    if (rgb !== lastRGB) {
      lastRGB = rgb;
      lastIndex = indexOfColor.get(rgb) ?? 0;
    }
    out[p] = lastIndex;
  }
  return out;
}

export type Layer = "political" | "state" | "region" | "names" | "terrain";

export function pack(r: number, g: number, b: number): number {
  return ((255 << 24) | (b << 16) | (g << 8) | r) >>> 0;
}

function unpack(color: number): [number, number, number] {
  return [color & 255, (color >>> 8) & 255, (color >>> 16) & 255];
}

export function mix(
  color: number,
  toward: [number, number, number],
  amount: number,
): number {
  const [r, g, b] = unpack(color);
  return pack(
    Math.round(r + (toward[0] - r) * amount),
    Math.round(g + (toward[1] - g) * amount),
    Math.round(b + (toward[2] - b) * amount),
  );
}

export const SEA = pack(46, 74, 104);

export const FLAGGED: [number, number, number] = [222, 60, 48];
export const LAKE = pack(84, 124, 156);
export const UNOWNED = pack(150, 146, 132);

export const ERASED = pack(232, 226, 210);
const NOTHING = pack(20, 20, 20);

export const TERRAIN_COLOURS: Readonly<
  Record<string, readonly [number, number, number]>
> = {
  plains: [174, 190, 122],
  forest: [74, 116, 62],
  hills: [184, 156, 100],
  mountain: [132, 116, 108],
  desert: [226, 206, 146],
  marsh: [104, 146, 128],
  jungle: [40, 108, 72],
  urban: [150, 136, 150],

  lakes: [84, 124, 156],

  ocean: [46, 74, 104],
};
const OTHER_TERRAIN = pack(160, 156, 140);

export function terrainColour(terrain: string): number {
  const rgb = TERRAIN_COLOURS[terrain];
  return rgb ? pack(rgb[0], rgb[1], rgb[2]) : OTHER_TERRAIN;
}

export function idColor(
  id: number,
  saturation = 0.42,
  lightness = 0.62,
): number {
  const hue = ((id * 137.508) % 360) / 360;
  const q =
    lightness < 0.5
      ? lightness * (1 + saturation)
      : lightness + saturation - lightness * saturation;
  const p = 2 * lightness - q;
  const channel = (t: number) => {
    let x = t;
    if (x < 0) x += 1;
    if (x > 1) x -= 1;
    if (x < 1 / 6) return p + (q - p) * 6 * x;
    if (x < 1 / 2) return q;
    if (x < 2 / 3) return p + (q - p) * (2 / 3 - x) * 6;
    return p;
  };
  return pack(
    Math.round(channel(hue + 1 / 3) * 255),
    Math.round(channel(hue) * 255),
    Math.round(channel(hue - 1 / 3) * 255),
  );
}

export type PickScope = "province" | "state" | "region";

export interface Selection {
  province: number;

  scope?: PickScope;
}

const WHITE: [number, number, number] = [255, 255, 255];

const ownerGroups = new Map<string, number>();
function ownerGroup(owner: string): number {
  let id = ownerGroups.get(owner);
  if (id === undefined) {
    id = ownerGroups.size + 1;
    ownerGroups.set(owner, id);
  }
  return id;
}

export function layerTable(
  map: WorldMap,
  layer: Layer,
  selection: Selection,
): { colors: Uint32Array; groups: Int32Array } {
  const colors = new Uint32Array(map.provinces.length);
  const groups = new Int32Array(map.provinces.length);
  const selected = map.provinces[selection.province];
  for (let i = 1; i < map.provinces.length; i++) {
    const province = map.provinces[i];
    let color: number;
    if (province.kind !== "land") {
      color =
        province.kind === "lake"
          ? LAKE
          : layer === "region"
            ? mix(idColor(province.region, 0.35, 0.42), [46, 74, 104], 0.55)
            : SEA;
    } else if (layer === "political" || layer === "names") {
      const painted = map.provinceOwners?.get(i);
      const owner = painted ?? map.states.get(province.state)?.owner ?? "";
      const rgb = owner ? map.colors[owner] : undefined;
      color =
        painted === "" ? ERASED : rgb ? pack(rgb[0], rgb[1], rgb[2]) : UNOWNED;
      if (layer === "names") color = mix(color, [236, 230, 214], 0.3);
    } else if (layer === "terrain") {
      color = terrainColour(map.provinceTerrains?.get(i) ?? province.terrain);
    } else if (layer === "state") {
      color = province.state ? idColor(province.state) : UNOWNED;
    } else {
      color = province.region ? idColor(province.region, 0.38, 0.6) : UNOWNED;
    }
    if (selected && selection.province && !selection.scope) {
      if (i === selection.province) color = mix(color, WHITE, 0.6);
      else if (province.state && province.state === selected.state)
        color = mix(color, WHITE, 0.3);
    } else if (selected && selection.province) {
      const inside =
        i === selection.province ||
        (selection.scope === "state" &&
          selected.state > 0 &&
          province.state === selected.state) ||
        (selection.scope === "region" &&
          selected.region > 0 &&
          province.region === selected.region);
      if (inside) color = mix(color, WHITE, 0.5);
    }
    if (map.provinceFlags?.has(i)) color = mix(color, FLAGGED, 0.75);
    colors[i] = color;
    if (layer === "political" && selection.scope === "state") {
      groups[i] = province.kind === "land" ? province.state : -1;
    } else if (layer === "political" && selection.scope === "region") {
      groups[i] = province.region;
    } else if (layer === "political") {
      groups[i] = ownerGroup(
        province.kind === "land"
          ? (map.provinceOwners?.get(i) ??
              map.states.get(province.state)?.owner ??
              "")
          : "~sea",
      );
    } else if (layer === "terrain") {
      groups[i] = ownerGroup(
        province.kind === "land"
          ? `~terrain:${map.provinceTerrains?.get(i) ?? province.terrain}`
          : "~sea",
      );
    } else if (layer === "state" || layer === "names") {
      groups[i] = province.kind === "land" ? province.state : -1;
    } else {
      groups[i] = province.region;
    }
  }
  colors[0] = NOTHING;
  return { colors, groups };
}

const BLACK: [number, number, number] = [0, 0, 0];

export function paintLayer(
  index: Uint16Array,
  width: number,
  table: { colors: Uint32Array; groups: Int32Array },
  out: Uint32Array,
): void {
  paintLayerRect(index, width, table, out, [0, 0, width, index.length / width]);
}

export type PixelRect = [number, number, number, number];

export function paintLayerRect(
  index: Uint16Array,
  width: number,
  table: { colors: Uint32Array; groups: Int32Array },
  out: Uint32Array,
  rect: PixelRect,
): void {
  const { colors, groups } = table;
  const strong = new Uint32Array(colors.length);
  const faint = new Uint32Array(colors.length);
  for (let i = 0; i < colors.length; i++) {
    strong[i] = mix(colors[i], BLACK, 0.55);
    faint[i] = mix(colors[i], BLACK, 0.14);
  }
  const height = index.length / width;
  const [x0, y0, x1, y1] = [
    Math.max(0, rect[0]),
    Math.max(0, rect[1]),
    Math.min(width, rect[2]),
    Math.min(height, rect[3]),
  ];
  for (let y = y0; y < y1; y++) {
    const row = y * width;
    for (let x = x0; x < x1; x++) {
      const p = row + x;
      const here = index[p];
      const right = x + 1 < width ? index[p + 1] : here;
      const down = y + 1 < height ? index[p + width] : here;
      if (right === here && down === here) {
        out[p] = colors[here];
      } else if (
        (right !== here && groups[right] !== groups[here]) ||
        (down !== here && groups[down] !== groups[here])
      ) {
        out[p] = strong[here];
      } else {
        out[p] = faint[here];
      }
    }
  }
}

export function provinceBoxes(
  index: Uint16Array,
  width: number,
  provinces: number,
): Int32Array {
  const boxes = new Int32Array(provinces * 4);
  for (let i = 0; i < provinces; i++) {
    boxes[i * 4] = width;
    boxes[i * 4 + 1] = index.length;
  }
  for (let p = 0; p < index.length; p++) {
    const i = index[p] * 4;
    const x = p % width;
    const y = (p - x) / width;
    if (x < boxes[i]) boxes[i] = x;
    if (y < boxes[i + 1]) boxes[i + 1] = y;
    if (x + 1 > boxes[i + 2]) boxes[i + 2] = x + 1;
    if (y + 1 > boxes[i + 3]) boxes[i + 3] = y + 1;
  }
  return boxes;
}

export function changedRects(
  before: { colors: Uint32Array; groups: Int32Array },
  after: { colors: Uint32Array; groups: Int32Array },
  boxes: Int32Array,
  limit = 3000,
): PixelRect[] | null {
  if (before.colors.length !== after.colors.length) return null;
  const rects: PixelRect[] = [];
  for (let i = 1; i < after.colors.length; i++) {
    if (
      before.colors[i] === after.colors[i] &&
      before.groups[i] === after.groups[i]
    )
      continue;
    if (rects.length >= limit) return null;
    if (boxes[i * 4 + 2] === 0) continue;
    rects.push([
      boxes[i * 4] - 1,
      boxes[i * 4 + 1] - 1,
      boxes[i * 4 + 2] + 1,
      boxes[i * 4 + 3] + 1,
    ]);
  }
  return rects;
}

export function provincesInCircle(
  index: Uint16Array,
  width: number,
  cx: number,
  cy: number,
  radius: number,
): number[] {
  const height = index.length / width;
  const step = Math.max(1, Math.floor(radius / 8));
  const found = new Set<number>();
  const r = Math.max(0.5, radius);
  for (let dy = -Math.floor(r); dy <= r; dy += step) {
    const y = Math.floor(cy + dy);
    if (y < 0 || y >= height) continue;
    const half = Math.sqrt(Math.max(0, r * r - dy * dy));
    for (let dx = -Math.floor(half); dx <= half; dx += step) {
      const x = ((Math.floor(cx + dx) % width) + width) % width;
      const i = index[y * width + x];
      if (i) found.add(i);
    }
  }
  const centre =
    index[
      Math.min(height - 1, Math.max(0, Math.floor(cy))) * width +
        (((Math.floor(cx) % width) + width) % width)
    ];
  if (centre) found.add(centre);
  return [...found];
}

export function provinceNeighbours(
  index: Uint16Array,
  width: number,
): Map<number, Set<number>> {
  const out = new Map<number, Set<number>>();
  const link = (a: number, b: number) => {
    if (!a || !b || a === b) return;
    let set = out.get(a);
    if (!set) out.set(a, (set = new Set()));
    set.add(b);
    let back = out.get(b);
    if (!back) out.set(b, (back = new Set()));
    back.add(a);
  };
  const height = index.length / width;
  for (let y = 0; y < height; y++) {
    const row = y * width;
    for (let x = 0; x < width; x++) {
      const here = index[row + x];
      link(here, index[row + ((x + 1) % width)]);
      if (y + 1 < height) link(here, index[row + width + x]);
    }
  }
  return out;
}

export interface Centroids {
  x: Float32Array;
  y: Float32Array;
  count: Uint32Array;
}

export function provinceCentroids(
  index: Uint16Array,
  width: number,
  provinces: number,
): Centroids {
  const sumX = new Float64Array(provinces);
  const sumY = new Float64Array(provinces);
  const count = new Uint32Array(provinces);
  for (let p = 0; p < index.length; p++) {
    const i = index[p];
    sumX[i] += p % width;
    sumY[i] += (p / width) | 0;
    count[i]++;
  }
  const x = new Float32Array(provinces);
  const y = new Float32Array(provinces);
  for (let i = 0; i < provinces; i++) {
    if (count[i]) {
      x[i] = sumX[i] / count[i];
      y[i] = sumY[i] / count[i];
    }
  }
  return { x, y, count };
}

export interface Place {
  x: number;
  y: number;
  size: number;

  province: number;
}

export function groupPlaces(
  map: WorldMap,
  centroids: Centroids,
  groupOf: (province: Province) => number,
  width: number,
): Map<number, Place> {
  const spans = new Map<number, { min: number; max: number }>();
  for (let i = 1; i < map.provinces.length; i++) {
    const group = groupOf(map.provinces[i]);
    if (!group || !centroids.count[i]) continue;
    const span = spans.get(group) ?? { min: Infinity, max: -Infinity };
    span.min = Math.min(span.min, centroids.x[i]);
    span.max = Math.max(span.max, centroids.x[i]);
    spans.set(group, span);
  }
  const sums = new Map<
    number,
    { x: number; y: number; size: number; province: number }
  >();
  for (let i = 1; i < map.provinces.length; i++) {
    const group = groupOf(map.provinces[i]);
    const n = centroids.count[i];
    if (!group || !n) continue;
    const span = spans.get(group)!;
    const across = span.max - span.min > width / 2;
    const x =
      across && centroids.x[i] < width / 2
        ? centroids.x[i] + width
        : centroids.x[i];
    const sum = sums.get(group) ?? { x: 0, y: 0, size: 0, province: i };
    sum.x += x * n;
    sum.y += centroids.y[i] * n;
    sum.size += n;
    sums.set(group, sum);
  }
  const out = new Map<number, Place>();
  for (const [group, sum] of sums)
    out.set(group, {
      x: (sum.x / sum.size) % width,
      y: sum.y / sum.size,
      size: sum.size,
      province: sum.province,
    });
  return out;
}

export interface NameHit {
  kind: "state" | "city" | "region";

  id: number;
  name: string;

  province: number;
}

export function searchNames(
  map: WorldMap,
  query: string,
  limit = 20,
): NameHit[] {
  const q = query.trim().toLowerCase();
  if (!q) return [];
  const provinceOf = new Map<number, number>();
  const stateProvince = new Map<number, number>();
  const regionProvince = new Map<number, number>();
  for (let i = 1; i < map.provinces.length; i++) {
    const province = map.provinces[i];
    provinceOf.set(province.id, i);
    if (province.state && !stateProvince.has(province.state))
      stateProvince.set(province.state, i);
    if (province.region && !regionProvince.has(province.region))
      regionProvince.set(province.region, i);
  }
  const out: NameHit[] = [];
  const add = (hit: NameHit) => {
    if (out.length < limit && hit.name.toLowerCase().includes(q)) out.push(hit);
  };
  for (const [id, state] of map.states)
    add({
      kind: "state",
      id,
      name: state.name,
      province: stateProvince.get(id) ?? 0,
    });
  for (const [id, point] of map.points)
    if (point.name)
      add({
        kind: "city",
        id,
        name: point.name,
        province: provinceOf.get(id) ?? 0,
      });
  for (const [id, name] of map.regions)
    add({ kind: "region", id, name, province: regionProvince.get(id) ?? 0 });
  return out.filter((hit) => hit.province > 0);
}
