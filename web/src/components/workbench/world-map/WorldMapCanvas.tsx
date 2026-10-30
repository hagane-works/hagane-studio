"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  changedRects,
  groupPlaces,
  layerTable,
  paintLayer,
  paintLayerRect,
  pixelIndex,
  provinceBoxes,
  provinceCentroids,
  provincesInCircle,
  type Centroids,
  type Layer,
  type PickScope,
  type PixelRect,
  type Place,
  type WorldMap,
} from "./worldMapModel";

interface View {
  scale: number;
  x: number;
  y: number;
}

interface Labels {
  map: WorldMap;
  centroids: Centroids;
  states: Map<number, Place>;
  seas: Map<number, Place>;

  cities: { index: number; value: number; name: string }[];

  indexOfId: Map<number, number>;
}

export interface MapUnit {
  province: number;
  count: number;
}

function drawUnits(
  ctx: CanvasRenderingContext2D,
  units: readonly MapUnit[],
  labels: Labels,
  view: View,
  width: number,
  height: number,
) {
  if (view.scale < 0.3) return;
  ctx.font = "700 11px system-ui, sans-serif";
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  for (const unit of units) {
    const index = labels.indexOfId.get(unit.province);
    if (!index || !labels.centroids.count[index]) continue;
    const x = view.x + labels.centroids.x[index] * view.scale;
    const y = view.y + labels.centroids.y[index] * view.scale;
    if (x < -30 || y < -30 || x > width + 30 || y > height + 30) continue;
    const text = String(unit.count);
    const w = Math.max(22, ctx.measureText(text).width + 12);
    const h = 16;
    ctx.fillStyle = "rgba(48, 56, 36, 0.92)";
    ctx.fillRect(x - w / 2, y - h / 2, w, h);
    ctx.strokeStyle = "rgba(214, 205, 160, 0.95)";
    ctx.lineWidth = 1;
    ctx.strokeRect(x - w / 2 + 0.5, y - h / 2 + 0.5, w - 1, h - 1);
    ctx.fillStyle = "rgb(250, 246, 236)";
    ctx.fillText(text, x, y + 0.5);
  }
}

const MIN_SCALE = 0.1;
const MAX_SCALE = 12;

export type MapRect = [number, number, number, number];

export interface LandLayer {
  sketch: (CanvasImageSource & { width: number; height: number }) | null;
  rect: MapRect | null;
  overlay: { image: CanvasImageSource; rect: MapRect } | null;
  onRect: (rect: MapRect) => void;
}

const MIN_FRAME = 8;

export function frameBetween(
  start: [number, number],
  end: [number, number],
  aspect: number | null,
  size: [number, number],
): MapRect {
  let w = Math.abs(end[0] - start[0]);
  let h = Math.abs(end[1] - start[1]);
  if (aspect) {
    w = Math.max(w, h * aspect);
    h = w / aspect;
  }
  w = Math.max(MIN_FRAME, Math.round(w));
  h = Math.max(MIN_FRAME, Math.round(h));
  let x = end[0] >= start[0] ? start[0] : start[0] - w;
  let y = end[1] >= start[1] ? start[1] : start[1] - h;
  w = Math.min(w, size[0]);
  h = Math.min(h, size[1]);
  x = Math.min(Math.max(0, Math.round(x)), size[0] - w);
  y = Math.min(Math.max(0, Math.round(y)), size[1] - h);
  return [x, y, w, h];
}

export function moveFrame(
  rect: MapRect,
  dx: number,
  dy: number,
  size: [number, number],
): MapRect {
  const [x, y, w, h] = rect;
  return [
    Math.min(Math.max(0, Math.round(x + dx)), size[0] - w),
    Math.min(Math.max(0, Math.round(y + dy)), size[1] - h),
    w,
    h,
  ];
}

export interface MapBrush {
  radius: number;

  onPaint: (
    provinces: number[],
    phase: "start" | "move" | "end",
    at: number,
    line?: { from: [number, number]; to: [number, number] },
  ) => void;
}

export interface MapMarks {
  base: ReadonlyMap<number, number>;
  over: ReadonlyMap<number, number>;
}

function markOf(marks: MapMarks | null, p: number): number {
  if (!marks) return 0;
  return marks.over.get(p) ?? marks.base.get(p) ?? 0;
}

function stampMarks(
  marks: MapMarks,
  width: number,
  out: Uint32Array,
  rect?: PixelRect,
) {
  const inside = (p: number) => {
    if (!rect) return true;
    const x = p % width;
    const y = (p - x) / width;
    return x >= rect[0] && y >= rect[1] && x < rect[2] && y < rect[3];
  };
  for (const [p, colour] of marks.base)
    if (!marks.over.has(p) && inside(p)) out[p] = colour;
  for (const [p, colour] of marks.over)
    if (colour && inside(p)) out[p] = colour;
}

export interface MapFocus {
  province: number;
  scale: number;

  at: number;
}

function drawLabels(
  ctx: CanvasRenderingContext2D,
  labels: Labels,
  view: View,
  width: number,
  height: number,
) {
  ctx.textBaseline = "middle";
  ctx.lineJoin = "round";
  const onScreen = (x: number, y: number) =>
    x > -80 && y > -20 && x < width + 80 && y < height + 20;

  const taken: [number, number, number, number][] = [];
  const room = (left: number, top: number, w: number, h: number) => {
    for (const [l, t, r, b] of taken)
      if (left < r && left + w > l && top < b && top + h > t) return false;
    taken.push([left, top, left + w, top + h]);
    return true;
  };
  const text = (value: string, x: number, y: number, fill: string) => {
    ctx.lineWidth = 3;
    ctx.strokeStyle = "rgba(16, 20, 26, 0.8)";
    ctx.strokeText(value, x, y);
    ctx.fillStyle = fill;
    ctx.fillText(value, x, y);
  };

  ctx.textAlign = "center";
  for (const [id, place] of labels.seas) {
    const name = labels.map.regions.get(id);
    const x = view.x + place.x * view.scale;
    const y = view.y + place.y * view.scale;
    if (!name || !onScreen(x, y)) continue;
    const extent = Math.sqrt(place.size) * view.scale;
    const size = Math.min(16, Math.max(10, extent / 7));
    ctx.font = `italic 400 ${size}px system-ui, sans-serif`;
    const w = ctx.measureText(name).width;
    if (w > extent * 1.4 || !room(x - w / 2, y - size / 2, w, size)) continue;
    text(name, x, y, "rgb(186, 208, 230)");
  }

  for (const [id, place] of labels.states) {
    const name = labels.map.states.get(id)?.name;
    const x = view.x + place.x * view.scale;
    const y = view.y + place.y * view.scale;
    if (!name || !onScreen(x, y)) continue;
    const extent = Math.sqrt(place.size) * view.scale;
    const size = Math.min(18, Math.max(10, extent / 5));
    ctx.font = `500 ${size}px system-ui, sans-serif`;
    const w = ctx.measureText(name).width;
    if (w > extent * 1.5 || !room(x - w / 2, y - size / 2, w, size)) continue;
    text(name, x, y, "rgb(250, 246, 236)");
  }

  ctx.textAlign = "left";
  ctx.font = "400 11px system-ui, sans-serif";
  for (const city of labels.cities) {
    if (view.scale < (city.value >= 15 ? 0.9 : 2)) continue;
    const x = view.x + labels.centroids.x[city.index] * view.scale;
    const y = view.y + labels.centroids.y[city.index] * view.scale;
    if (!onScreen(x, y)) continue;
    const w = city.name ? ctx.measureText(city.name).width + 8 : 6;
    if (!room(x - 3, y, w, 12)) continue;
    ctx.fillStyle = "rgb(250, 246, 236)";
    ctx.strokeStyle = "rgba(16, 20, 26, 0.9)";
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    ctx.arc(x, y + 6, 2.5, 0, Math.PI * 2);
    ctx.fill();
    ctx.stroke();
    if (city.name) text(city.name, x + 5, y + 6, "rgb(255, 226, 160)");
  }
}

export function localPoint(
  canvas: HTMLCanvasElement,
  clientX: number,
  clientY: number,
): [number, number] {
  const rect = canvas.getBoundingClientRect();
  const kx = canvas.clientWidth ? rect.width / canvas.clientWidth : 1;
  const ky = canvas.clientHeight ? rect.height / canvas.clientHeight : 1;
  return [(clientX - rect.left) / (kx || 1), (clientY - rect.top) / (ky || 1)];
}

export function WorldMapCanvas({
  map,
  imageUrl,
  layer,
  selected,
  focus,
  onPick,
  onHover,
  loadingText,
  land,
  quiet = false,
  pickScope,
  brush,
  onIndex,
  marks,
  units,
}: {
  map: WorldMap;

  imageUrl: string;
  layer: Layer;

  selected: number;

  pickScope?: PickScope;
  focus?: MapFocus | null;
  onPick: (province: number) => void;
  onHover: (province: number) => void;
  loadingText: string;

  land?: LandLayer | null;

  quiet?: boolean;

  brush?: MapBrush | null;

  onIndex?: (index: Uint16Array, width: number) => void;

  marks?: MapMarks | null;

  units?: readonly MapUnit[] | null;
}) {
  const boxRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const pictureRef = useRef<{
    canvas: HTMLCanvasElement;
    data: ImageData;
    index: Uint16Array;
    boxes: Int32Array;
  } | null>(null);

  const paintedRef = useRef<{
    picture: object;
    layer: Layer;
    table: ReturnType<typeof layerTable>;
    marks: MapMarks | null;
  } | null>(null);
  const brushRef = useRef<MapBrush | null>(brush ?? null);
  const unitsRef = useRef<readonly MapUnit[] | null>(units ?? null);
  const onIndexRef = useRef(onIndex);
  const centroidsRef = useRef<{ picture: object; centroids: Centroids } | null>(
    null,
  );

  const pointerRef = useRef<[number, number] | null>(null);
  const labelsRef = useRef<Labels | null>(null);
  const layerRef = useRef<Layer>(layer);

  const mapRef = useRef(map);
  const viewRef = useRef<View | null>(null);
  const landRef = useRef<LandLayer | null>(land ?? null);

  const frameRef = useRef<MapRect | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | string>("loading");

  const [pictureCount, setPictureCount] = useState(0);

  const [shownUrl, setShownUrl] = useState("");

  const draw = useCallback(() => {
    const canvas = canvasRef.current;
    const picture = pictureRef.current;
    const view = viewRef.current;
    if (!canvas || !picture || !view) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    const dpr = window.devicePixelRatio || 1;
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.fillStyle = "rgb(16, 22, 30)";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

    ctx.imageSmoothingEnabled = view.scale < 1.5;
    ctx.drawImage(
      picture.canvas,
      view.x,
      view.y,
      picture.data.width * view.scale,
      picture.data.height * view.scale,
    );
    if (layerRef.current === "names" && labelsRef.current)
      drawLabels(
        ctx,
        labelsRef.current,
        view,
        canvas.width / dpr,
        canvas.height / dpr,
      );
    if (unitsRef.current?.length && labelsRef.current)
      drawUnits(
        ctx,
        unitsRef.current,
        labelsRef.current,
        view,
        canvas.width / dpr,
        canvas.height / dpr,
      );
    const land = landRef.current;
    const frame = frameRef.current ?? land?.rect ?? null;
    const at = (rect: MapRect) =>
      [
        view.x + rect[0] * view.scale,
        view.y + rect[1] * view.scale,
        rect[2] * view.scale,
        rect[3] * view.scale,
      ] as const;
    if (land?.overlay && !frameRef.current) {
      ctx.imageSmoothingEnabled = false;
      ctx.drawImage(land.overlay.image, ...at(land.overlay.rect));
    } else if (land?.sketch && frame) {
      ctx.globalAlpha = 0.6;
      ctx.imageSmoothingEnabled = false;
      ctx.drawImage(land.sketch, ...at(frame));
      ctx.globalAlpha = 1;
    }
    if (land && frame) {
      ctx.setLineDash([6, 4]);
      ctx.lineWidth = 1.5;
      ctx.strokeStyle = "rgb(255, 196, 92)";
      ctx.strokeRect(...at(frame));
      ctx.setLineDash([]);
    }
    const ring = brushRef.current;
    const pointer = pointerRef.current;
    if (ring && pointer) {
      const r = Math.max(2, ring.radius * view.scale);
      ctx.lineWidth = 1.5;
      ctx.strokeStyle = "rgba(16, 20, 26, 0.85)";
      ctx.beginPath();
      ctx.arc(pointer[0], pointer[1], r + 1, 0, Math.PI * 2);
      ctx.stroke();
      ctx.strokeStyle = "rgba(250, 246, 236, 0.95)";
      ctx.beginPath();
      ctx.arc(pointer[0], pointer[1], r, 0, Math.PI * 2);
      ctx.stroke();
    }
  }, []);

  const fit = useCallback(() => {
    const box = boxRef.current;
    const picture = pictureRef.current;
    if (!box || !picture) return;
    const scale = Math.min(
      box.clientWidth / picture.data.width,
      box.clientHeight / picture.data.height,
    );
    viewRef.current = {
      scale,
      x: (box.clientWidth - picture.data.width * scale) / 2,
      y: (box.clientHeight - picture.data.height * scale) / 2,
    };
  }, []);

  useEffect(() => {
    mapRef.current = map;
  }, [map]);

  useEffect(() => {
    landRef.current = land ?? null;
    draw();
  }, [land, draw]);

  useEffect(() => {
    brushRef.current = brush ?? null;
    draw();
  }, [brush, draw]);

  useEffect(() => {
    onIndexRef.current = onIndex;
  }, [onIndex]);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const response = await fetch(imageUrl);
        if (!response.ok) {
          const out = (await response.json().catch(() => ({}))) as {
            detail?: string;
          };
          throw new Error(out.detail || `HTTP ${response.status}`);
        }
        const bitmap = await createImageBitmap(await response.blob(), {
          colorSpaceConversion: "none",
          premultiplyAlpha: "none",
        });
        const canvas = document.createElement("canvas");
        canvas.width = bitmap.width;
        canvas.height = bitmap.height;
        const ctx = canvas.getContext("2d", { willReadFrequently: true });
        if (!ctx) throw new Error("canvas unavailable");
        ctx.drawImage(bitmap, 0, 0);
        const data = ctx.getImageData(0, 0, bitmap.width, bitmap.height);
        const index = pixelIndex(data.data, mapRef.current.indexOfColor);
        const boxes = provinceBoxes(
          index,
          data.width,
          mapRef.current.provinces.length,
        );
        if (!alive) return;

        const before = pictureRef.current;
        if (
          !before ||
          before.data.width !== data.width ||
          before.data.height !== data.height
        )
          viewRef.current = null;
        pictureRef.current = { canvas, data, index, boxes };
        onIndexRef.current?.(index, data.width);
        setStatus("ready");
        setPictureCount((n) => n + 1);
        setShownUrl(imageUrl);
      } catch (err) {
        if (alive) setStatus(err instanceof Error ? err.message : String(err));
      }
    })();
    return () => {
      alive = false;
    };
  }, [imageUrl]);

  useEffect(() => {
    const picture = pictureRef.current;
    if (status !== "ready" || !picture) return;

    const known = centroidsRef.current;
    const centroids =
      known &&
      known.picture === picture &&
      known.centroids.count.length === map.provinces.length
        ? known.centroids
        : provinceCentroids(
            picture.index,
            picture.data.width,
            map.provinces.length,
          );
    centroidsRef.current = { picture, centroids };
    const indexOfId = new Map<number, number>();
    for (let i = 1; i < map.provinces.length; i++)
      indexOfId.set(map.provinces[i].id, i);
    const cities: Labels["cities"] = [];
    for (const [id, point] of map.points) {
      const index = indexOfId.get(id);
      if (index && centroids.count[index])
        cities.push({ index, value: point.value, name: point.name });
    }
    cities.sort((a, b) => b.value - a.value);
    labelsRef.current = {
      map,
      centroids,
      states: groupPlaces(
        map,
        centroids,
        (p) => (p.kind === "land" ? p.state : 0),
        picture.data.width,
      ),
      seas: groupPlaces(
        map,
        centroids,
        (p) => (p.kind === "sea" ? p.region : 0),
        picture.data.width,
      ),
      cities,
      indexOfId,
    };
    draw();
  }, [status, pictureCount, map, draw]);

  useEffect(() => {
    unitsRef.current = units ?? null;
    draw();
  }, [units, draw]);

  useEffect(() => {
    const current = pictureRef.current;
    layerRef.current = layer;
    if (status !== "ready" || !current) return;
    const table = layerTable(map, layer, {
      province: selected,
      scope: pickScope,
    });
    const out = new Uint32Array(current.data.data.buffer);
    const painted = paintedRef.current;
    const shownMarks = marks ?? null;
    const width = current.data.width;

    const rects =
      painted &&
      painted.picture === current &&
      painted.layer === layer &&
      painted.marks?.base === shownMarks?.base
        ? changedRects(painted.table, table, current.boxes)
        : null;
    if (rects) {
      let union: PixelRect | null = null;
      const grow = (rect: PixelRect) => {
        union = union
          ? [
              Math.min(union[0], rect[0]),
              Math.min(union[1], rect[1]),
              Math.max(union[2], rect[2]),
              Math.max(union[3], rect[3]),
            ]
          : rect;
      };
      for (const rect of rects) {
        paintLayerRect(current.index, width, table, out, rect);
        grow(rect);
      }

      const before = painted?.marks ?? null;
      if (before !== shownMarks) {
        let box: PixelRect | null = null;
        const check = (p: number) => {
          if (markOf(before, p) === markOf(shownMarks, p)) return;
          const x = p % width;
          const y = (p - x) / width;
          box = box
            ? [
                Math.min(box[0], x),
                Math.min(box[1], y),
                Math.max(box[2], x + 1),
                Math.max(box[3], y + 1),
              ]
            : [x, y, x + 1, y + 1];
        };
        for (const p of before?.over.keys() ?? []) check(p);
        for (const p of shownMarks?.over.keys() ?? []) check(p);
        if (box) {
          paintLayerRect(current.index, width, table, out, box);
          grow(box);
        }
      }
      const area = union as PixelRect | null;
      if (area) {
        if (shownMarks) stampMarks(shownMarks, width, out, area);
        const [x0, y0] = [Math.max(0, area[0]), Math.max(0, area[1])];
        const [x1, y1] = [
          Math.min(width, area[2]),
          Math.min(current.data.height, area[3]),
        ];
        current.canvas
          .getContext("2d")
          ?.putImageData(current.data, 0, 0, x0, y0, x1 - x0, y1 - y0);
      }
    } else {
      paintLayer(current.index, width, table, out);
      if (shownMarks) stampMarks(shownMarks, width, out);
      current.canvas.getContext("2d")?.putImageData(current.data, 0, 0);
    }
    paintedRef.current = { picture: current, layer, table, marks: shownMarks };
    if (!viewRef.current) fit();
    draw();
  }, [status, pictureCount, map, layer, selected, pickScope, draw, fit, marks]);

  useEffect(() => {
    const box = boxRef.current;
    const labels = labelsRef.current;
    if (
      !focus ||
      status !== "ready" ||
      !box ||
      !labels ||
      !labels.centroids.count[focus.province]
    )
      return;
    const scale = Math.max(viewRef.current?.scale ?? 0, focus.scale);
    viewRef.current = {
      scale,
      x: box.clientWidth / 2 - labels.centroids.x[focus.province] * scale,
      y: box.clientHeight / 2 - labels.centroids.y[focus.province] * scale,
    };
    draw();
  }, [focus, status, pictureCount, draw]);

  useEffect(() => {
    const box = boxRef.current;
    const canvas = canvasRef.current;
    if (!box || !canvas) return;
    const resize = () => {
      const dpr = window.devicePixelRatio || 1;
      canvas.width = Math.max(1, Math.round(box.clientWidth * dpr));
      canvas.height = Math.max(1, Math.round(box.clientHeight * dpr));
      draw();
    };
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(box);
    const onWheel = (e: WheelEvent) => {
      const view = viewRef.current;
      if (!view) return;
      e.preventDefault();
      const [mx, my] = localPoint(canvas, e.clientX, e.clientY);
      const scale = Math.min(
        MAX_SCALE,
        Math.max(MIN_SCALE, view.scale * (e.deltaY < 0 ? 1.2 : 1 / 1.2)),
      );
      viewRef.current = {
        scale,
        x: mx - ((mx - view.x) * scale) / view.scale,
        y: my - ((my - view.y) * scale) / view.scale,
      };
      draw();
    };
    canvas.addEventListener("wheel", onWheel, { passive: false });
    return () => {
      observer.disconnect();
      canvas.removeEventListener("wheel", onWheel);
    };
  }, [draw]);

  const provinceAt = (clientX: number, clientY: number): number => {
    const canvas = canvasRef.current;
    const picture = pictureRef.current;
    const view = viewRef.current;
    if (!canvas || !picture || !view) return 0;
    const [x, y] = localPoint(canvas, clientX, clientY);
    const px = Math.floor((x - view.x) / view.scale);
    const py = Math.floor((y - view.y) / view.scale);
    if (
      px < 0 ||
      py < 0 ||
      px >= picture.data.width ||
      py >= picture.data.height
    )
      return 0;
    const province = picture.index[py * picture.data.width + px];

    return province < mapRef.current.provinces.length ? province : 0;
  };

  const mapPoint = (
    clientX: number,
    clientY: number,
  ): [number, number] | null => {
    const canvas = canvasRef.current;
    const view = viewRef.current;
    if (!canvas || !view) return null;
    const [x, y] = localPoint(canvas, clientX, clientY);
    return [(x - view.x) / view.scale, (y - view.y) / view.scale];
  };

  const drag = useRef<{
    x: number;
    y: number;
    viewX: number;
    viewY: number;
    moved: boolean;
  } | null>(null);

  const painting = useRef<[number, number] | null>(null);

  const brushed = (
    from: [number, number],
    to: [number, number],
    radius: number,
  ): number[] => {
    const picture = pictureRef.current;
    if (!picture) return [];
    const step = Math.max(1, radius / 2);
    const n = Math.max(
      1,
      Math.ceil(Math.hypot(to[0] - from[0], to[1] - from[1]) / step),
    );
    const found = new Set<number>();
    for (let k = 0; k <= n; k++) {
      const t = k / n;
      for (const i of provincesInCircle(
        picture.index,
        picture.data.width,
        from[0] + (to[0] - from[0]) * t,
        from[1] + (to[1] - from[1]) * t,
        radius,
      ))
        found.add(i);
    }
    return [...found];
  };

  const framing = useRef<{
    start: [number, number];
    from: MapRect | null;
  } | null>(null);
  const hovered = useRef(0);

  return (
    <div
      ref={boxRef}
      className={`relative h-full w-full overflow-hidden ${quiet ? "" : "bg-background"}`}
    >
      <canvas
        ref={canvasRef}
        className={`absolute inset-0 h-full w-full touch-none ${quiet ? "" : "cursor-crosshair"}`}
        onContextMenu={(e) => {
          if (landRef.current || brushRef.current) e.preventDefault();
        }}
        onPointerDown={(e) => {
          const view = viewRef.current;
          if (!view) return;
          const placing = landRef.current;
          if (placing && e.button === 0) {
            const point = mapPoint(e.clientX, e.clientY);
            if (!point) return;
            (e.currentTarget as HTMLCanvasElement).setPointerCapture(
              e.pointerId,
            );
            const rect = placing.rect;
            const inside =
              rect &&
              point[0] >= rect[0] &&
              point[1] >= rect[1] &&
              point[0] < rect[0] + rect[2] &&
              point[1] < rect[1] + rect[3];
            framing.current = { start: point, from: inside ? rect : null };
            return;
          }
          const painter = brushRef.current;
          if (painter && e.button === 0) {
            const point = mapPoint(e.clientX, e.clientY);
            if (!point) return;
            (e.currentTarget as HTMLCanvasElement).setPointerCapture(
              e.pointerId,
            );
            painting.current = point;
            painter.onPaint(
              brushed(point, point, painter.radius),
              "start",
              provinceAt(e.clientX, e.clientY),
              { from: point, to: point },
            );
            return;
          }

          if (e.button !== (placing || painter ? 2 : 0)) return;
          (e.currentTarget as HTMLCanvasElement).setPointerCapture(e.pointerId);
          const [x, y] = localPoint(e.currentTarget, e.clientX, e.clientY);
          drag.current = { x, y, viewX: view.x, viewY: view.y, moved: false };
        }}
        onPointerMove={(e) => {
          const framed = framing.current;
          const current = pictureRef.current;
          if (framed && current) {
            const point = mapPoint(e.clientX, e.clientY);
            if (!point) return;
            const size: [number, number] = [
              current.data.width,
              current.data.height,
            ];
            const sketch = landRef.current?.sketch;
            frameRef.current = framed.from
              ? moveFrame(
                  framed.from,
                  point[0] - framed.start[0],
                  point[1] - framed.start[1],
                  size,
                )
              : frameBetween(
                  framed.start,
                  point,
                  sketch ? sketch.width / sketch.height : null,
                  size,
                );
            draw();
            return;
          }
          const painter = brushRef.current;
          if (painter) {
            pointerRef.current = localPoint(
              e.currentTarget,
              e.clientX,
              e.clientY,
            );
            const from = painting.current;
            const point = from ? mapPoint(e.clientX, e.clientY) : null;
            if (from && point) {
              painting.current = point;
              painter.onPaint(
                brushed(from, point, painter.radius),
                "move",
                provinceAt(e.clientX, e.clientY),
                { from, to: point },
              );
            }
            if (!drag.current) draw();
          }
          const pressed = drag.current;
          const view = viewRef.current;
          if (pressed && view) {
            const [x, y] = localPoint(e.currentTarget, e.clientX, e.clientY);
            const dx = x - pressed.x;
            const dy = y - pressed.y;
            if (pressed.moved || Math.abs(dx) + Math.abs(dy) > 4) {
              pressed.moved = true;
              viewRef.current = {
                ...view,
                x: pressed.viewX + dx,
                y: pressed.viewY + dy,
              };
              draw();
            }
            return;
          }
          const province = provinceAt(e.clientX, e.clientY);
          if (province !== hovered.current) {
            hovered.current = province;
            onHover(province);
          }
        }}
        onPointerUp={(e) => {
          if (painting.current) {
            painting.current = null;
            brushRef.current?.onPaint([], "end", 0);
            return;
          }
          if (framing.current) {
            framing.current = null;
            const frame = frameRef.current;
            frameRef.current = null;
            if (frame) landRef.current?.onRect(frame);
            draw();
            return;
          }
          const pressed = drag.current;
          drag.current = null;
          if (pressed && !pressed.moved && !landRef.current)
            onPick(provinceAt(e.clientX, e.clientY));
        }}
        onPointerLeave={() => {
          if (pointerRef.current) {
            pointerRef.current = null;
            draw();
          }
          if (hovered.current) {
            hovered.current = 0;
            onHover(0);
          }
        }}
        onDoubleClick={() => {
          fit();
          draw();
        }}
      />
      {!quiet && status === "ready" && shownUrl !== imageUrl && (
        <div className="pointer-events-none absolute left-1/2 top-3 -translate-x-1/2 animate-pulse-ring rounded-md border border-accent bg-card px-3 py-1.5 text-xs text-muted-foreground">
          {loadingText}
        </div>
      )}
      {!quiet && status !== "ready" && (
        <div className="absolute inset-0 flex items-center justify-center p-6">
          {status === "loading" ? (
            <div className="animate-pulse-ring rounded-md border border-accent bg-card px-4 py-3 text-sm text-muted-foreground">
              {loadingText}
            </div>
          ) : (
            <div className="max-w-md rounded-md border border-destructive/40 bg-card px-4 py-3 text-sm text-destructive">
              {status}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
