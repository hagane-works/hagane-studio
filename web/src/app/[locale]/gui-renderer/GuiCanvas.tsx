"use client";

import { useRef, useState } from "react";
import type {
  GuiNode,
  GuiSprite,
  GuiText,
  GuiFontMeta,
} from "@/lib/guiRenderApi";
import { guiRenderApi } from "@/lib/guiRenderApi";
import { textColors } from "@/lib/referenceTables";
import { classifyNode, AXIS_STYLE } from "./guiClassification";
import { clipRects, containerScroll } from "./containerScroll";
import { BitmapText } from "./BitmapText";
import { GameTooltipHost, useTooltipSource } from "./GameTooltip";
import { RichText } from "./RichText";
import type { TooltipHit } from "./tooltipHub";
import {
  fillRuntimeSlots,
  hasTooltip,
  nodeTooltip,
  normalizeTooltip,
  type GameTooltipContent,
  type GameTooltipInput,
} from "./tooltipText";

const RICH_MARKUP = /[§£]/;
const NO_SLOTS: Readonly<Record<string, number | string>> = {};

export const GUI_UNHANDLED_ACTIVATE_EVENT = "gui-canvas:unhandled-activate";

export interface GuiUnhandledActivateDetail {
  control: string;
  owner?: string;
}

type Sprites = Record<string, GuiSprite>;
type Texts = Record<string, GuiText>;
type Fonts = Record<string, GuiFontMeta>;
type Versions = Record<string, number>;

function outlineShadow(size: number): string {
  const d = size <= 16 ? 1 : 2;
  const c = "#0a0a0a";
  return [
    `-${d}px -${d}px 0 ${c}`,
    `${d}px -${d}px 0 ${c}`,
    `-${d}px ${d}px 0 ${c}`,
    `${d}px ${d}px 0 ${c}`,
    `0 1px 2px rgba(0,0,0,0.65)`,
  ].join(",");
}

export function assignIds(root: GuiNode): void {
  const walk = (n: GuiNode, id: string) => {
    (n as GuiNode & { _id: string })._id = id;
    if (n.background)
      (n.background as GuiNode & { _id: string })._id = id + "/bg";
    (n.children || []).forEach((c, i) => walk(c, `${id}/${i}`));
  };
  walk(root, "root");
}

export interface TreeRow {
  id: string;
  node: GuiNode;
  depth: number;
}

export function flattenTree(root: GuiNode): TreeRow[] {
  const rows: TreeRow[] = [];
  const walk = (n: GuiNode, depth: number) => {
    rows.push({ id: (n as GuiNode & { _id: string })._id, node: n, depth });
    (n.children || []).forEach((c) => walk(c, depth + 1));
  };
  walk(root, 0);
  return rows;
}

function nid(n: GuiNode): string {
  return (n as GuiNode & { _id?: string })._id || "";
}

function resolvePct(val: number | string | undefined, base: number): number {
  if (typeof val === "string" && val.trim().endsWith("%")) {
    const n = parseFloat(val);
    return Math.round((base * (isNaN(n) ? 100 : n)) / 100);
  }
  const n = Number(val);
  return isNaN(n) ? 0 : Math.round(n);
}

function resolveSize(val: number | string | undefined, base: number): number {
  const n = resolvePct(val, base);
  return n < 0 ? Math.max(0, base + n) : n;
}

function asNum(val: number | string | undefined): number {
  if (val == null) return 0;
  if (typeof val === "number") return val;
  if (val.trim().endsWith("%")) return NaN;
  const n = parseFloat(val);
  return isNaN(n) ? 0 : n;
}

type RawItem = { k?: unknown; v?: unknown };

export function sizeMin(
  node: GuiNode,
): { width?: number | string; height?: number | string } | null {
  const tokens = (
    node as GuiNode & { point_tokens?: { size?: { __block__?: RawItem[] } } }
  ).point_tokens;
  const items = tokens?.size?.__block__;
  if (!Array.isArray(items)) return null;
  const min = items.find(
    (item) => String(item.k ?? "").toLowerCase() === "min",
  );
  const inner = (min?.v as { __block__?: RawItem[] } | undefined)?.__block__;
  if (!Array.isArray(inner)) return null;
  const out: { width?: number | string; height?: number | string } = {};
  for (const item of inner) {
    const key = String(item.k ?? "").toLowerCase();
    const value =
      typeof item.v === "number" || typeof item.v === "string"
        ? item.v
        : undefined;
    if (key === "width" || key === "x") out.width = value;
    if (key === "height" || key === "y") out.height = value;
  }
  return out;
}

function pointW(size: GuiNode["size"]): number | string | undefined {
  return size?.width ?? size?.x;
}

function pointH(size: GuiNode["size"]): number | string | undefined {
  return size?.height ?? size?.y;
}

const SIZE_HINT_RE = /(?:^|_)(\d{2,4})x(\d{2,4})(?:_|$)/i;

function spriteFrameSize(
  meta: GuiSprite | undefined,
  ref?: string | null,
): { w: number; h: number } {
  if (ref) {
    const hit = ref.match(SIZE_HINT_RE);
    if (hit && (!meta?.width || !meta?.height)) {
      return { w: Number(hit[1]) || 0, h: Number(hit[2]) || 0 };
    }
  }
  if (!meta) return { w: 0, h: 0 };
  return {
    w: meta.width
      ? Math.max(1, Math.round(meta.width / Math.max(1, meta.frames || 1)))
      : 0,
    h: meta.height || 0,
  };
}

export function anchor(
  orientation: string | null,
  px: number,
  py: number,
  pw: number,
  ph: number,
): [number, number] {
  const o = (orientation || "UPPER_LEFT").toUpperCase();
  let ax = px;
  let ay = py;

  if (o.includes("RIGHT")) ax = px + pw;
  else if (o.includes("LEFT")) ax = px;
  else if (o.includes("CENTER")) ax = px + Math.floor(pw / 2);
  const vTop =
    o.includes("UPPER") || o.includes("TOP") || /(^|_)UP($|_)/.test(o);
  if (o.includes("LOWER") || o.includes("DOWN") || o.includes("BOTTOM"))
    ay = py + ph;
  else if (vTop) ay = py;
  else if (o.includes("CENTER")) ay = py + Math.floor(ph / 2);
  return [ax, ay];
}

function applyOrigo(
  origo: string | null,
  x: number,
  y: number,
  w: number,
  h: number,
): [number, number] {
  const o = (origo || "").toUpperCase();
  if (!o) return [x, y];
  let ox = x;
  let oy = y;

  if (o.includes("RIGHT")) ox -= w;
  else if (o.includes("LEFT")) ox -= 0;
  else if (o.includes("CENTER")) ox -= Math.floor(w / 2);
  const vTop =
    o.includes("UPPER") || o.includes("TOP") || /(^|_)UP($|_)/.test(o);
  if (o.includes("LOWER") || o.includes("DOWN") || o.includes("BOTTOM"))
    oy -= h;
  else if (vTop) oy -= 0;
  else if (o.includes("CENTER")) oy -= Math.floor(h / 2);
  return [ox, oy];
}

export const ANCHOR_CELLS: string[][] = [
  ["UPPER_LEFT", "CENTER_UP", "UPPER_RIGHT"],
  ["CENTER_LEFT", "CENTER", "CENTER_RIGHT"],
  ["LOWER_LEFT", "CENTER_DOWN", "LOWER_RIGHT"],
];

function backgroundBox(
  meta: GuiSprite | undefined,
  ref: string,
  fillW: number,
  fillH: number,
): { w: number; h: number } {
  if (!meta || meta.cornered || meta.procedural) return { w: fillW, h: fillH };
  const natural = meta.resolved
    ? { w: Math.round(meta.width / Math.max(1, meta.frames)), h: meta.height }
    : spriteFrameSize(meta, ref);
  if (!natural.w || !natural.h) return { w: fillW, h: fillH };
  return { w: natural.w, h: natural.h };
}

function isActivatable(node: GuiNode): boolean {
  const t = node.type.toLowerCase();
  return t.endsWith("buttontype") || t === "checkboxtype";
}

function seeThroughBackdrop(node: GuiNode): boolean {
  return (
    node.type.toLowerCase() === "background" &&
    /opacity|transparent|shade|veil/i.test(node.sprite ?? "")
  );
}

export interface PickedHit {
  id: string;
  node: GuiNode;

  w?: number;
  h?: number;

  size?: { width: number; height: number };
}

export function shownSize(
  hit: {
    node: GuiNode;
    x: number;
    y: number;
    ow: number;
    oh: number;
    px: number;
    py: number;
    pw: number;
    ph: number;
  },
  W: number,
  H: number,
): { width: number; height: number } {
  const backdrop = hit.node.type.toLowerCase() === "background";
  const [x, y, w, h] = backdrop
    ? [hit.px, hit.py, hit.pw, hit.ph]
    : [hit.x, hit.y, hit.ow, hit.oh];
  const width = Math.min(x + w, W) - Math.max(x, 0);
  const height = Math.min(y + h, H) - Math.max(y, 0);
  return { width: width > 0 ? width : w, height: height > 0 ? height : h };
}

export function pickInStack<T extends PickedHit>(
  full: readonly T[],
): T | undefined {
  let stack = full;
  while (stack.length > 1 && seeThroughBackdrop(stack[0].node))
    stack = stack.slice(1);
  const top = stack[0];
  if (!top || isActivatable(top.node)) return top;
  const area = (entry: PickedHit) => (entry.w ?? 0) * (entry.h ?? 0);
  for (let at = 0; at < stack.length; at += 1) {
    const entry = stack[at];
    if (isActivatable(entry.node)) {
      if (!(entry.node.sprite || entry.node.background?.sprite)) return top;
      const badge = stack
        .slice(0, at)
        .find(
          (above) =>
            above.node.type.toLowerCase() === "icontype" &&
            area(above) < area(entry) * 0.8,
        );
      return badge ?? entry;
    }
    const t = entry.node.type.toLowerCase();

    if (t !== "icontype" && !t.endsWith("textboxtype")) return top;
  }
  return top;
}

type ButtonState = "over" | "down";

const RAMP_EASE = "cubic-bezier(0.333, 0.75, 0.667, 1.083)";
const LOOK = {
  rampOver: {
    filter: "contrast(0.9032) brightness(1.24)",
    transition: `filter 250ms ${RAMP_EASE}`,
  },
  rampDown: {
    filter: "brightness(0.7857) contrast(1.12)",
    transition: `filter 62.5ms ${RAMP_EASE}`,
  },
  fullOver: { filter: "contrast(0.8846) brightness(1.3)" },
  fullDown: { filter: "brightness(0.7391) contrast(1.15)" },
  plusTenth: { filter: "contrast(0.8333) brightness(1.2)" },
  minusTenth: { filter: "brightness(0.8333) contrast(1.2)" },
  none: {},
} satisfies Record<string, React.CSSProperties>;

function effectLook(
  effect: string | null | undefined,
  state: ButtonState,
): React.CSSProperties | null {
  const e = (effect || "").toLowerCase();
  if (!e.includes("buttonstate")) return null;
  if (e.includes("onlydisable") || e.includes("rendertarget")) return LOOK.none;
  if (e.includes("nodown"))
    return state === "over" ? LOOK.plusTenth : LOOK.none;
  if (e.includes("linear"))
    return state === "over" ? LOOK.plusTenth : LOOK.minusTenth;
  if (e.includes("blendframes") || e.includes("fade_frames_to_black")) {
    return state === "over" ? LOOK.fullOver : LOOK.fullDown;
  }
  return state === "over" ? LOOK.rampOver : LOOK.rampDown;
}

export function buttonLook(
  node: GuiNode,
  meta: GuiSprite | undefined,
  state: ButtonState | null,
): { frame: number | null | undefined; style?: React.CSSProperties } {
  if (!state || !meta || !isActivatable(node)) return { frame: node.frame };
  const look = effectLook(meta.effect, state);
  if (look) return { frame: node.frame, style: look };
  const frames = meta.frames || 1;

  if (
    frames > 1 &&
    node.type.toLowerCase().endsWith("buttontype") &&
    (node.frame ?? 1) === 1
  ) {
    return { frame: state === "down" ? Math.min(3, frames) : 2 };
  }
  return { frame: node.frame };
}

export function anchorCell(value: string | null | undefined): [number, number] {
  const o = (value || "").toUpperCase();
  let col = 0;
  if (o.includes("RIGHT")) col = 2;
  else if (o.includes("LEFT")) col = 0;
  else if (o.includes("CENTER")) col = 1;
  let row = 0;
  const vTop =
    o.includes("UPPER") || o.includes("TOP") || /(^|_)UP($|_)/.test(o);
  if (o.includes("LOWER") || o.includes("DOWN") || o.includes("BOTTOM"))
    row = 2;
  else if (vTop) row = 0;
  else if (o.includes("CENTER")) row = 1;
  return [col, row];
}

export function cellAnchor(col: number, row: number): string {
  return ANCHOR_CELLS[row]?.[col] ?? "UPPER_LEFT";
}

function fontStyleFor(font: string | null): {
  fontFamily?: string;
  fontSize: number;
} {
  const n = (font || "").toLowerCase();
  const m = (font || "").match(/(\d{2})/);
  const size = m ? Math.max(8, Math.min(parseInt(m[1], 10), 64)) : 14;
  let fontFamily: string | undefined;
  if (n.startsWith("aldrich")) fontFamily = "ModAldrich";
  else if (n.startsWith("bomb")) fontFamily = "ModBomb";
  else if (n.includes("times") || n.includes("typewriter"))
    fontFamily = "ModTimes";
  return { fontFamily, fontSize: size };
}

const COLOR_CODES: Record<string, string> = {
  R: "#c83c3c",
  G: "#5ac85a",
  B: "#5a8cdc",
  Y: "#dcc85a",
  W: "#ebebeb",
  O: "#dc963c",
  T: "#5ac8c8",
  P: "#c878dc",
};

function textColor(kind: string, code: string | null): string {
  if (kind === "runtime") return "#96aa96";
  if (kind === "key") return "#c8a05a";
  return COLOR_CODES[code || ""] || "#e1e1e1";
}

export function textAlignOf(
  format: string | null | undefined,
  fallback: "left" | "center" | "right" = "left",
): "left" | "center" | "right" {
  const word = String(format || fallback).toLowerCase();
  if (word === "centre" || word === "center") return "center";
  if (word === "right") return "right";
  return "left";
}

export function isPlayerHidden(node: GuiNode): boolean {
  return rawTruthy(node.raw?.gameview_hidden);
}

function rawTruthy(value: unknown): boolean {
  if (typeof value === "boolean") return value;
  if (typeof value === "number") return value !== 0;
  if (typeof value === "string") return /^(yes|true|1)$/i.test(value.trim());
  return false;
}

function runtimeHidden(node: GuiNode): boolean {
  const raw = node.raw || {};

  if (node.type === "expandedwindow") return true;
  return rawTruthy(raw.hide) || rawTruthy(raw.hidden);
}

function spriteStyle(
  ref: string,
  meta: GuiSprite,
  w: number,
  h: number,
  version: number | undefined,
  frame: number | null | undefined = 1,
): React.CSSProperties {
  const url =
    meta.vanilla_url && !version
      ? meta.vanilla_url
      : guiRenderApi.spriteUrl(ref, version ?? meta.version);
  const nat = { w: meta.width || w, h: meta.height || h };

  if (meta.cornered && meta.bordersize) {
    const { x: bx, y: by } = meta.bordersize;

    const borderY = h > 0 ? Math.min(by, Math.floor(h / 2)) : by;
    const borderX = w > 0 ? Math.min(bx, Math.floor(w / 2)) : bx;
    return {
      width: w,
      height: h,
      borderStyle: "solid",
      borderWidth: `${borderY}px ${borderX}px`,
      borderImageSource: `url("${url}")`,
      borderImageSlice: `${by} ${bx} fill`,
      borderImageRepeat: "round",
    };
  }
  if (meta.frames > 1) {
    const frames = Math.max(1, meta.frames || 1);
    const fw = Math.floor(nat.w / frames);
    const idx = Math.max(0, Math.min(frames - 1, (frame || 1) - 1));
    const drawW = w || fw;
    const drawH = h || nat.h;
    return {
      width: drawW,
      height: drawH,
      backgroundImage: `url("${url}")`,
      backgroundPosition: `-${idx * drawW}px 0`,
      backgroundRepeat: "no-repeat",
      backgroundSize: `${drawW * frames}px ${drawH}px`,
    };
  }
  if ((w >= nat.w * 2 || h >= nat.h * 2) && nat.w <= 256 && nat.h <= 256) {
    return {
      width: w,
      height: h,
      backgroundImage: `url("${url}")`,
      backgroundRepeat: "repeat",
    };
  }
  return {
    width: w,
    height: h,
    backgroundImage: `url("${url}")`,
    backgroundSize: `${w}px ${h}px`,
    backgroundRepeat: "no-repeat",
  };
}

export interface LayoutOverride {
  x?: number;
  y?: number;
  width?: number;
  height?: number;

  orientation?: string;

  origo?: string;
}

export interface DragState {
  mode: "move" | "resize";
  startX: number;
  startY: number;
  baseX: number;
  baseY: number;

  axes?: "x" | "y" | "xy";

  started?: boolean;
}

export type ResizeAxes = "xy" | "x" | "none";

export function dragLayoutResult(
  drag: DragState,
  clientX: number,
  clientY: number,
  zoom: number,
): LayoutOverride | null {
  const z = zoom || 1;
  const axes = drag.mode === "resize" ? (drag.axes ?? "xy") : "xy";
  const dx = axes === "y" ? 0 : Math.round((clientX - drag.startX) / z);
  const dy = axes === "x" ? 0 : Math.round((clientY - drag.startY) / z);
  if (!dx && !dy) return null;
  if (drag.mode === "move") return { x: drag.baseX + dx, y: drag.baseY + dy };
  if (axes === "x") return { width: Math.max(1, drag.baseX + dx) };
  if (axes === "y") return { height: Math.max(1, drag.baseY + dy) };
  return {
    width: Math.max(1, drag.baseX + dx),
    height: Math.max(1, drag.baseY + dy),
  };
}

interface HitRect {
  id: string;
  x: number;
  y: number;
  w: number;
  h: number;

  ow: number;
  oh: number;

  sx: number;
  sy: number;

  px: number;
  py: number;
  pw: number;
  ph: number;

  orientation: string;
  origo: string;
  node: GuiNode;
}

export interface TooltipTarget {
  id: string;
  x: number;
  y: number;
  w: number;
  h: number;
  node: GuiNode;
}

export interface MeasuredRect {
  x: number;
  y: number;
  w: number;
  h: number;
  hidden: boolean;
}

interface Ctx {
  sprites: Sprites;
  texts: Texts;
  fonts: Fonts;
  versions: Versions;
  textOverrides: Record<string, string>;
  layoutOverrides: Record<string, LayoutOverride>;
  showUnresolvedKeys: boolean;
  showMissingSprites: boolean;
  hidden: Set<string>;
  isolateId: string | null;
  out: React.ReactNode[];
  hits: HitRect[];

  tips: TooltipTarget[];
  keySeq: { n: number };

  rects: Record<string, MeasuredRect> | null;

  hoverId: string | null;
  pressId: string | null;

  disabled: ReadonlySet<string> | null;

  nativeButtonSprites?: boolean;

  containerHits?: boolean;

  showPlayerHidden?: boolean;
}

function pushSprite(
  ctx: Ctx,
  key: string,
  ref: string,
  meta: GuiSprite | undefined,
  x: number,
  y: number,
  w: number,
  h: number,
  frame?: number | null,
  look?: React.CSSProperties,

  background = false,
): void {
  if (meta && meta.resolved) {
    ctx.out.push(
      <div
        key={key}
        data-window-bg={background ? "" : undefined}
        style={{
          position: "absolute",
          left: x,
          top: y,
          pointerEvents: "none",
          ...spriteStyle(
            ref,
            meta,
            w || meta.width,
            h || meta.height,
            ctx.versions[ref],
            frame,
          ),
          ...look,
        }}
      />,
    );
  } else if (meta?.procedural) {
    const isPie = meta.block_type === "piecharttype";
    ctx.out.push(
      <div
        key={key}
        style={{
          position: "absolute",
          left: x,
          top: y,
          width: w || meta.width || 30,
          height: h || meta.height || 30,
          border: "2px solid rgba(107,220,220,0.75)",
          borderRadius: isPie ? "50%" : 0,
          background: "rgba(44,92,92,0.28)",
          boxShadow: "inset 0 0 0 1px rgba(0,0,0,0.75)",
          pointerEvents: "none",
        }}
      />,
    );
  } else if (ctx.showMissingSprites) {
    ctx.out.push(
      <div
        key={key}
        style={{
          position: "absolute",
          left: x,
          top: y,
          width: w || 40,
          height: h || 20,
          border: "1px solid rgba(255,80,80,0.5)",
          color: "rgba(255,120,120,0.85)",
          fontSize: 9,
          overflow: "hidden",
          whiteSpace: "nowrap",
          pointerEvents: "none",
        }}
      >
        {ref.replace("GFX_", "")}
      </div>,
    );
  }
}

function inIsolateScope(id: string, isolateId: string | null): boolean {
  if (!isolateId) return true;
  return (
    id === isolateId ||
    id.startsWith(isolateId + "/") ||
    isolateId.startsWith(id + "/")
  );
}

function isRuntimeVisibleForIsolation(
  id: string,
  isolateId: string | null,
): boolean {
  if (!isolateId) return false;
  return (
    id === isolateId ||
    id.startsWith(isolateId + "/") ||
    isolateId.startsWith(id + "/")
  );
}

function emit(
  node: GuiNode,
  ctx: Ctx,
  px: number,
  py: number,
  pw: number,
  ph: number,
  parentHidden: boolean,
): void {
  if (!isPlayerHidden(node))
    return emitNode(node, ctx, px, py, pw, ph, parentHidden);
  if (!ctx.showPlayerHidden) return emitNode(node, ctx, px, py, pw, ph, true);
  const start = ctx.out.length;
  emitNode(node, ctx, px, py, pw, ph, parentHidden);
  if (ctx.out.length > start) {
    const own = ctx.out.splice(start);
    ctx.out.push(
      <div
        key={`h${ctx.keySeq.n++}`}
        data-player-hidden=""
        style={{ position: "absolute", left: 0, top: 0, opacity: 0.3 }}
      >
        {own}
      </div>,
    );
  }
}

function emitNode(
  node: GuiNode,
  ctx: Ctx,
  px: number,
  py: number,
  pw: number,
  ph: number,
  parentHidden: boolean,
): void {
  const id = nid(node);
  const ref = node.sprite;
  const meta = ref ? ctx.sprites[ref] : undefined;
  const scale = node.scale || 1.0;

  const outStart = ctx.out.length;

  const lo = ctx.layoutOverrides[id];
  const effPosX = lo && lo.x != null ? lo.x : node.position?.x;
  const effPosY = lo && lo.y != null ? lo.y : node.position?.y;
  const effW = lo && lo.width != null ? lo.width : pointW(node.size);
  const effH = lo && lo.height != null ? lo.height : pointH(node.size);
  const hasSize =
    node.size != null ||
    (lo != null && (lo.width != null || lo.height != null));

  let ow = 0;
  let oh = 0;
  if (hasSize) {
    ow = resolveSize(effW, pw);
    oh = resolveSize(effH, ph);
    const floor = sizeMin(node);
    if (floor?.width != null) ow = Math.max(ow, resolveSize(floor.width, pw));
    if (floor?.height != null) oh = Math.max(oh, resolveSize(floor.height, ph));
  } else if (meta && meta.resolved) {
    ow = Math.round((meta.width / Math.max(1, meta.frames)) * scale);
    oh = Math.round(meta.height * scale);
  } else if (meta) {
    const hint = spriteFrameSize(meta, ref);
    ow = Math.round(hint.w * scale);
    oh = Math.round(hint.h * scale);
  }
  if (
    ctx.nativeButtonSprites &&
    hasSize &&
    meta?.resolved &&
    !meta.cornered &&
    !meta.procedural &&
    node.type.toLowerCase().endsWith("buttontype")
  ) {
    ow = Math.round((meta.width / Math.max(1, meta.frames)) * scale);
    oh = Math.round(meta.height * scale);
  }

  const effOrientation = lo?.orientation ?? node.orientation;
  const effOrigo = lo?.origo ?? node.origo;
  const [ax, ay] = anchor(effOrientation, px, py, pw, ph);
  let x = ax + resolvePct(effPosX, pw);
  let y = ay + resolvePct(effPosY, ph);
  if (effOrigo) {
    [x, y] = applyOrigo(effOrigo, x, y, ow, oh);
  } else if (node.centerposition) {
    x -= Math.floor(ow / 2);
    y -= Math.floor(oh / 2);
  }

  const hidden =
    parentHidden ||
    ctx.hidden.has(id) ||
    (runtimeHidden(node) && !isRuntimeVisibleForIsolation(id, ctx.isolateId));
  const show = !hidden && inIsolateScope(id, ctx.isolateId);

  const isContainer = node.type.endsWith("windowtype");
  const groupOnly = isContainer && !hasSize;
  const cw = hasSize && ow ? ow : groupOnly ? 0 : pw;
  const ch = hasSize && oh ? oh : groupOnly ? 0 : ph;
  const cpx = hasSize || groupOnly ? x : px;
  const cpy = hasSize || groupOnly ? y : py;

  if (show && node.background?.sprite) {
    const bg = node.background;
    const bref = bg.sprite;
    if (!bref) return;
    const bmeta = ctx.sprites[bref];
    const fill = bg.size
      ? {
          w: resolveSize(pointW(bg.size), cw),
          h: resolveSize(pointH(bg.size), ch),
        }
      : backgroundBox(bmeta, bref, cw, ch);
    let bw = fill.w;
    let bh = fill.h;
    if (!bw || !bh) {
      const hint = spriteFrameSize(bmeta, bref);
      bw ||= hint.w;
      bh ||= hint.h;
    }
    const [bax, bay] = anchor(bg.orientation, cpx, cpy, cw, ch);
    let bx = bax + resolvePct(bg.position?.x, cw);
    let by = bay + resolvePct(bg.position?.y, ch);
    if (bg.origo) {
      [bx, by] = applyOrigo(bg.origo, bx, by, bw, bh);
    } else if (bg.centerposition) {
      bx -= Math.floor(bw / 2);
      by -= Math.floor(bh / 2);
    }
    pushSprite(
      ctx,
      `bg${ctx.keySeq.n++}`,
      bref,
      bmeta,
      bx,
      by,
      bw,
      bh,
      bg.frame,
      undefined,
      true,
    );

    if (!(ctx.containerHits && isContainer && hasSize))
      ctx.hits.push({
        id: nid(bg),
        x: bx,
        y: by,
        w: bw,
        h: bh,
        ow: bw,
        oh: bh,
        sx: NaN,
        sy: NaN,
        px: cpx,
        py: cpy,
        pw: cw,
        ph: ch,
        orientation: (bg.orientation || "UPPER_LEFT").toUpperCase(),
        origo: (bg.origo || "").toUpperCase(),
        node: bg,
      });

    if (bg.raw?.pdx_tooltip || bg.raw?.pdx_tooltip_delayed) {
      ctx.tips.push({ id: nid(bg), x: bx, y: by, w: bw, h: bh, node: bg });
    }
  }

  if (
    ctx.containerHits &&
    show &&
    isContainer &&
    hasSize &&
    !node.name?.endsWith("_content")
  ) {
    ctx.hits.push({
      id,
      x,
      y,
      w: ow,
      h: oh,
      ow,
      oh,
      sx: asNum(effPosX),
      sy: asNum(effPosY),
      px,
      py,
      pw,
      ph,
      orientation: (effOrientation || "UPPER_LEFT").toUpperCase(),
      origo: (effOrigo || "").toUpperCase(),
      node,
    });
  }

  const k = ctx.keySeq.n++;
  if (show && ref) {
    const off = !!ctx.disabled?.has(id);
    const state: ButtonState | null = off
      ? null
      : id === ctx.pressId
        ? "down"
        : id === ctx.hoverId
          ? "over"
          : null;
    const look = buttonLook(node, meta, state);
    pushSprite(ctx, `s${k}`, ref, meta, x, y, ow, oh, look.frame, look.style);
  }

  let textW = 0;
  let textH = 0;

  const overridden = Object.prototype.hasOwnProperty.call(
    ctx.textOverrides,
    id,
  );
  if (show && (node.text || overridden)) {
    const info = (node.text && ctx.texts[node.text]) || {
      display: node.text ?? "",
      kind: "literal" as const,
    };
    const hasOverride = overridden;
    const displayText = hasOverride ? ctx.textOverrides[id] : info.display;
    const isButton = node.type.endsWith("buttontype");
    const boxW =
      node.maxwidth || ow || (node.text_source === "buttontext" ? 120 : 0);
    textW = boxW;
    const fstyle = fontStyleFor(node.font);
    const fmeta = ctx.fonts[node.font || ""];
    textH = isButton && oh ? oh : fstyle.fontSize + 4;
    const fmt = textAlignOf(node.format, isButton ? "center" : "left");

    const rawSizeH = pointH(node.size);
    const sizeH = typeof rawSizeH === "number" ? rawSizeH : 0;
    const capH = isButton && oh ? oh : node.maxheight || sizeH || undefined;

    const clipLine = isButton || (info.kind === "key" && !hasOverride);
    const renderText =
      hasOverride || ctx.showUnresolvedKeys || info.kind !== "key";
    const bitmap = fmeta?.bitmap;
    const vAlign = String(node.raw?.vertical_alignment ?? "").toLowerCase();

    const markupSource = hasOverride
      ? RICH_MARKUP.test(displayText)
        ? displayText
        : null
      : (info.markup ?? null);
    const markup = markupSource
      ? fillRuntimeSlots(markupSource, NO_SLOTS)
      : null;

    const pad = isButton
      ? 0
      : Number((node.raw?.bordersize as { x?: unknown } | undefined)?.x) || 0;
    const padW = boxW ? Math.max(0, boxW - 2 * pad) : 0;

    const fixed = /^(yes|true)$/i.test(String(node.raw?.fixedsize ?? ""));

    if (!isButton && vAlign === "center" && capH) textH = Math.max(textH, capH);
    if (renderText && bitmap && markup !== null) {
      ctx.out.push(
        <RichText
          key={`t${k}`}
          font={bitmap}
          markup={markup}
          left={x + pad}
          top={y}
          width={padW}
          height={isButton && oh ? oh : capH}
          align={fmt}
          vcenter={(isButton && !!oh) || vAlign === "center"}
          maxLines={
            clipLine
              ? 1
              : fixed && capH
                ? Math.max(1, Math.floor(capH / bitmap.line))
                : 64
          }
        />,
      );
    } else if (renderText && bitmap && (hasOverride || info.kind !== "key")) {
      const lineCap =
        fixed && capH ? Math.max(1, Math.floor(capH / bitmap.line)) : 64;
      ctx.out.push(
        <BitmapText
          key={`t${k}`}
          font={bitmap}

          color={node.text_color ? textColors()[node.text_color] : undefined}
          text={displayText}
          left={x + pad}
          top={y}
          width={padW}
          height={isButton && oh ? oh : capH}
          align={fmt}
          vcenter={(isButton && !!oh) || vAlign === "center"}
          maxLines={clipLine ? 1 : lineCap}
          ellipsis={fixed}
        />,
      );
    } else if (renderText) {
      ctx.out.push(
        <div
          key={`t${k}`}
          style={{
            position: "absolute",
            left: x,
            top: y,
            width: boxW || undefined,
            height: isButton && oh ? oh : undefined,
            maxHeight: capH,
            display: isButton ? "flex" : "block",
            alignItems: isButton ? "center" : undefined,
            justifyContent:
              fmt === "right"
                ? "flex-end"
                : fmt === "center"
                  ? "center"
                  : "flex-start",
            textAlign: fmt,
            color: textColor(info.kind, node.text_color),
            ...fstyle,
            lineHeight: fmeta?.line_ratio || 1.1,
            textShadow: fmeta?.outline
              ? outlineShadow(fstyle.fontSize)
              : undefined,
            whiteSpace: clipLine ? "nowrap" : undefined,
            textOverflow: clipLine ? "ellipsis" : undefined,
            pointerEvents: "none",
            overflow: "hidden",
          }}
        >
          {displayText}
        </div>,
      );
    }
  }

  if (show && !node.alwaystransparent) {
    const tw = ow || textW;
    const th = oh || textH;
    if (tw > 0 && th > 0) ctx.tips.push({ id, x, y, w: tw, h: th, node });
  }

  if (show && (ref || node.text)) {
    ctx.hits.push({
      id,
      x,
      y,
      w: Math.max(ow, textW, 12),
      h: Math.max(oh, textH, 12),
      ow: ow || Math.max(textW, 12),
      oh: oh || Math.max(textH, 12),
      sx: asNum(effPosX),
      sy: asNum(effPosY),
      px,
      py,
      pw,
      ph,
      orientation: (effOrientation || "UPPER_LEFT").toUpperCase(),
      origo: (effOrigo || "").toUpperCase(),
      node,
    });
  }

  if (ctx.rects) {
    const rect: MeasuredRect = { x, y, w: ow, h: oh, hidden };
    ctx.rects[id] = rect;
    const nameKey = node.name ? `name:${node.name.toLowerCase()}` : "";
    if (nameKey && !ctx.rects[nameKey]) ctx.rects[nameKey] = rect;
  }

  const alpha = Number(node.raw?.alpha);
  if (
    node.raw?.alpha != null &&
    Number.isFinite(alpha) &&
    alpha < 1 &&
    ctx.out.length > outStart
  ) {
    const own = ctx.out.splice(outStart);
    ctx.out.push(
      <div key={`a${ctx.keySeq.n++}`} style={{ opacity: Math.max(0, alpha) }}>
        {own}
      </div>,
    );
  }

  if (ctx.disabled?.has(id) && ctx.out.length > outStart) {
    const own = ctx.out.splice(outStart);
    ctx.out.push(
      <div
        key={`d${ctx.keySeq.n++}`}
        style={{
          position: "absolute",
          left: 0,
          top: 0,
          filter: "grayscale(1)",
        }}
      >
        {own}
      </div>,
    );
  }

  const scroll = containerScroll(node);
  if (!scroll) {
    for (const child of node.children || [])
      emit(child, ctx, cpx, cpy, cw, ch, hidden);
    return;
  }

  const drawnAt = ctx.out.length;
  const hitsAt = ctx.hits.length;
  const tipsAt = ctx.tips.length;
  for (const child of node.children || [])
    emit(child, ctx, cpx - scroll.x, cpy - scroll.y, cw, ch, hidden);
  const box = { x: cpx, y: cpy, w: cw, h: ch };
  if (ctx.out.length > drawnAt) {
    const inner = ctx.out.splice(drawnAt);
    ctx.out.push(
      <div
        key={`s${ctx.keySeq.n++}`}
        style={{
          position: "absolute",
          left: box.x,
          top: box.y,
          width: box.w,
          height: box.h,
          overflow: "hidden",
          pointerEvents: "none",
        }}
      >
        <div style={{ position: "absolute", left: -box.x, top: -box.y }}>
          {inner}
        </div>
      </div>,
    );
  }
  ctx.hits.splice(
    hitsAt,
    ctx.hits.length - hitsAt,
    ...clipRects(ctx.hits.slice(hitsAt), box),
  );
  ctx.tips.splice(
    tipsAt,
    ctx.tips.length - tipsAt,
    ...clipRects(ctx.tips.slice(tipsAt), box),
  );
}

export function measureWindow(
  node: GuiNode,
  sprites: Sprites,
  baseW: number,
  baseH: number,
  hidden?: Set<string>,
  opts?: { nativeButtonSprites?: boolean },
): Record<string, MeasuredRect> {
  const size = node.size || {};
  const width = resolveSize(pointW(size), baseW) || 800;
  const height = resolveSize(pointH(size), baseH) || 600;
  assignIds(node);
  const rects: Record<string, MeasuredRect> = {};
  const ctx: Ctx = {
    sprites,
    texts: {},
    fonts: {},
    versions: {},
    textOverrides: {},
    layoutOverrides: {},
    showUnresolvedKeys: false,
    showMissingSprites: false,
    hidden: hidden || new Set<string>(),
    isolateId: null,
    out: [],
    hits: [],
    tips: [],
    keySeq: { n: 0 },
    rects,
    hoverId: null,
    pressId: null,
    disabled: null,
    nativeButtonSprites: opts?.nativeButtonSprites,
  };

  for (const child of node.children || [])
    emit(child, ctx, 0, 0, width, height, false);
  return rects;
}

export function tooltipAt(
  tips: TooltipTarget[],
  resolve: (node: GuiNode) => GameTooltipContent | null,
  x: number,
  y: number,
): TooltipHit | null {
  for (let i = tips.length - 1; i >= 0; i--) {
    const tip = tips[i];
    if (x < tip.x || y < tip.y || x >= tip.x + tip.w || y >= tip.y + tip.h)
      continue;
    const content = resolve(tip.node);
    if (content && hasTooltip(content)) {
      return {
        key: tip.id,
        content,
        rect: { x: tip.x, y: tip.y, w: tip.w, h: tip.h },
      };
    }
  }
  return null;
}

export function resolveNodeTooltip(
  node: GuiNode,
  texts: Texts,
  tooltipFor?: (node: GuiNode) => GameTooltipInput,
): GameTooltipContent | null {
  const own = tooltipFor?.(node);
  if (own !== undefined) return normalizeTooltip(own);
  return nodeTooltip(node, texts);
}

export function tooltipTargets(
  node: GuiNode,
  sprites: Sprites,
  texts: Texts,
  fonts: Fonts,
  baseW: number,
  baseH: number,
  hidden?: Set<string>,
): TooltipTarget[] {
  const size = node.size || {};
  const width = resolveSize(pointW(size), baseW) || 800;
  const height = resolveSize(pointH(size), baseH) || 600;
  assignIds(node);
  const ctx: Ctx = {
    sprites,
    texts,
    fonts,
    versions: {},
    textOverrides: {},
    layoutOverrides: {},
    showUnresolvedKeys: false,
    showMissingSprites: false,
    hidden: hidden || new Set<string>(),
    isolateId: null,
    out: [],
    hits: [],
    tips: [],
    keySeq: { n: 0 },
    rects: null,
    hoverId: null,
    pressId: null,
    disabled: null,
  };
  for (const child of node.children || [])
    emit(child, ctx, 0, 0, width, height, false);
  return ctx.tips;
}

export function GuiCanvas({
  node,
  sprites,
  texts,
  fonts,
  versions,
  hidden,
  isolateId,
  selectedId,
  selectionOwner,
  onSelect,
  editMode = true,
  transparent = false,
  textOverrides = {},
  layoutOverrides = {},
  showUnresolvedKeys = false,
  showMissingSprites = false,
  zoom = 1,
  baseW = 1920,
  baseH = 1080,
  onLayoutPreview,
  onLayoutCommit,
  onLayoutCancel,
  onActivate,
  playInEdit,
  playInEditHint,
  tooltipFor,
  disabled,
  resizeAxes,
  nativeButtonSprites = false,
  containerHits = false,
  onEditNode,
  dragThreshold = 0,
}: {
  node: GuiNode;
  sprites: Sprites;
  texts: Texts;
  fonts: Fonts;
  versions: Versions;
  hidden: Set<string>;
  isolateId: string | null;
  selectedId: string | null;

  selectionOwner?: string;

  onSelect: (id: string, node: GuiNode, stack?: PickedHit[]) => void;
  editMode?: boolean;
  transparent?: boolean;
  textOverrides?: Record<string, string>;
  layoutOverrides?: Record<string, LayoutOverride>;
  showUnresolvedKeys?: boolean;
  showMissingSprites?: boolean;
  zoom?: number;
  baseW?: number;
  baseH?: number;
  onLayoutPreview?: (id: string, partial: LayoutOverride) => void;
  onLayoutCommit?: (id: string, partial: LayoutOverride) => void;
  onLayoutCancel?: (id: string) => void;

  onActivate?: (node: GuiNode) => boolean | void;

  playInEdit?: (node: GuiNode) => boolean;

  playInEditHint?: string;

  tooltipFor?: (node: GuiNode) => GameTooltipInput;

  disabled?: ReadonlySet<string>;

  resizeAxes?: (node: GuiNode, sprite: GuiSprite | undefined) => ResizeAxes;

  nativeButtonSprites?: boolean;

  containerHits?: boolean;

  onEditNode?: (id: string, node: GuiNode) => void;

  dragThreshold?: number;
}) {
  const size = node.size || {};
  const W = resolveSize(pointW(size), baseW) || 800;
  const H = resolveSize(pointH(size), baseH) || 600;

  const [hoverId, setHoverId] = useState<string | null>(null);
  const [pressId, setPressId] = useState<string | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  const ctx: Ctx = {
    sprites,
    texts,
    fonts,
    versions,
    textOverrides,
    layoutOverrides,
    showUnresolvedKeys,
    showMissingSprites,
    hidden,
    isolateId,
    out: [],
    hits: [],
    tips: [],
    keySeq: { n: 0 },
    rects: null,
    hoverId: editMode ? null : hoverId,
    pressId: editMode ? null : pressId,
    disabled: disabled ?? null,
    nativeButtonSprites,
    containerHits,
    showPlayerHidden: editMode,
  };

  const dragRef = useRef<({ id: string } & DragState) | null>(null);

  const hitsUnder = (
    clientX: number,
    clientY: number,
    fallback: HitRect,
  ): HitRect[] => {
    const root = rootRef.current;
    const out: HitRect[] = [];
    const under =
      typeof document.elementsFromPoint === "function"
        ? document.elementsFromPoint(clientX, clientY)
        : [];
    for (const el of under) {
      if (
        !(el instanceof HTMLElement) ||
        el.dataset.hit === undefined ||
        !root?.contains(el)
      )
        continue;
      const hit = ctx.hits[Number(el.dataset.hit)];
      if (hit && !out.includes(hit)) out.push(hit);
    }
    return out.length ? out : [fallback];
  };

  const beginDrag = (
    e: React.PointerEvent,
    clicked: HitRect,
    mode: "move" | "resize",
    axes: "x" | "y" | "xy" = "xy",
  ) => {
    e.stopPropagation();
    let h = clicked;
    if (mode === "move") {
      const stack = hitsUnder(e.clientX, e.clientY, clicked);
      h = pickInStack(stack) ?? clicked;
      onSelect(
        h.id,
        h.node,
        stack.map((hit) => ({
          id: hit.id,
          node: hit.node,
          size: shownSize(hit, W, H),
        })),
      );
    }
    const baseX = mode === "move" ? h.sx : h.ow;
    const baseY = mode === "move" ? h.sy : h.oh;

    if (isNaN(baseX) || isNaN(baseY)) return;
    (e.currentTarget as Element).setPointerCapture?.(e.pointerId);
    dragRef.current = {
      id: h.id,
      mode,
      startX: e.clientX,
      startY: e.clientY,
      baseX,
      baseY,
      axes,
    };
  };

  const moveDrag = (e: React.PointerEvent) => {
    const d = dragRef.current;
    if (!d) return;
    if (!d.started && dragThreshold > 0) {
      if (
        Math.hypot(e.clientX - d.startX, e.clientY - d.startY) < dragThreshold
      )
        return;
      d.started = true;
    }
    const r = dragLayoutResult(d, e.clientX, e.clientY, zoom);
    if (r) onLayoutPreview?.(d.id, r);
  };

  const endDrag = (e: React.PointerEvent) => {
    const d = dragRef.current;
    if (!d) return;
    const r =
      d.started || dragThreshold <= 0
        ? dragLayoutResult(d, e.clientX, e.clientY, zoom)
        : null;
    dragRef.current = null;
    if (r) onLayoutCommit?.(d.id, r);
    else onLayoutCancel?.(d.id);
  };

  const bg = node.background;
  let bgEl: React.ReactNode = null;
  if (
    bg &&
    bg.sprite &&
    !hidden.has(nid(bg)) &&
    inIsolateScope(nid(bg), isolateId)
  ) {
    const bmeta = sprites[bg.sprite];
    if (bmeta && bmeta.resolved) {
      bgEl = (
        <div
          data-window-bg=""
          style={{
            position: "absolute",
            left: 0,
            top: 0,
            pointerEvents: "none",
            ...(() => {
              const box = backgroundBox(bmeta, bg.sprite, W, H);
              return spriteStyle(
                bg.sprite,
                bmeta,
                box.w,
                box.h,
                versions[bg.sprite],
                bg.frame,
              );
            })(),
          }}
        />
      );

      const box = backgroundBox(bmeta, bg.sprite, W, H);
      ctx.hits.push({
        id: nid(bg),
        x: 0,
        y: 0,
        w: box.w || W,
        h: box.h || H,
        ow: box.w || W,
        oh: box.h || H,
        sx: NaN,
        sy: NaN,
        px: 0,
        py: 0,
        pw: W,
        ph: H,
        orientation: "UPPER_LEFT",
        origo: "",
        node: bg,
      });
    }
  }

  for (const child of node.children || []) emit(child, ctx, 0, 0, W, H, false);

  const tips = ctx.tips;
  useTooltipSource(
    !editMode,
    () => rootRef.current,
    (x, y) =>
      tooltipAt(tips, (n) => resolveNodeTooltip(n, texts, tooltipFor), x, y),
  );

  return (
    <div
      ref={rootRef}
      style={{
        position: "relative",
        width: W,
        height: H,
        background: transparent ? "transparent" : "#1c1c20",
        flexShrink: 0,
      }}
    >
      {bgEl}
      {ctx.out}

      {!editMode &&
        ctx.hits
          .filter((h) => isActivatable(h.node) && !disabled?.has(h.id))
          .map((h, i) => (
            <div
              key={`p${i}`}
              role="button"
              aria-label={h.node.name || h.node.type}
              data-node-name={h.node.name || undefined}
              onClick={(e) => {
                if (onActivate) {
                  e.stopPropagation();
                  if (onActivate(h.node) !== false) return;
                } else {
                  const semanticParent = e.currentTarget.parentElement?.closest(
                    "button, [role='button']",
                  );
                  if (semanticParent) return;
                  e.stopPropagation();
                }
                window.dispatchEvent(
                  new CustomEvent<GuiUnhandledActivateDetail>(
                    GUI_UNHANDLED_ACTIVATE_EVENT,
                    {
                      detail: {
                        control: h.node.name || h.node.type,
                        owner: selectionOwner,
                      },
                    },
                  ),
                );
              }}
              onPointerEnter={() => setHoverId(h.id)}
              onPointerLeave={() => {
                setHoverId((at) => (at === h.id ? null : at));
                setPressId((at) => (at === h.id ? null : at));
              }}
              onPointerDown={(e) => {
                if (e.button === 0) setPressId(h.id);
              }}
              onPointerUp={() => setPressId(null)}
              style={{
                position: "absolute",
                left: h.x,
                top: h.y,
                width: h.w,
                height: h.h,
                pointerEvents: "auto",
              }}
            />
          ))}

      {!editMode && <GameTooltipHost fallback />}

      {editMode &&
        ctx.hits.map((h, i) => {
          const sel = h.id === selectedId;
          const axis = AXIS_STYLE[classifyNode(h.node, texts)];
          const draggable = !isNaN(h.sx) && !isNaN(h.sy);
          return (
            <div
              key={`h${i}`}
              data-hit={i}
              onPointerDown={(e) => beginDrag(e, h, "move")}
              onPointerMove={moveDrag}
              onPointerUp={endDrag}

              onClick={(e) => e.stopPropagation()}
              onDoubleClick={
                onEditNode
                  ? (e) => {
                      e.stopPropagation();
                      onEditNode(h.id, h.node);
                    }
                  : undefined
              }
              title={`${h.node.type} ${h.node.name || ""}`}
              style={{
                position: "absolute",
                left: h.x,
                top: h.y,
                width: h.w,
                height: h.h,
                cursor: draggable ? "move" : "pointer",
                touchAction: "none",

                pointerEvents: "auto",
                outline: sel
                  ? `2px solid ${axis.outline}`
                  : `1px dashed ${axis.outline}`,
                outlineOffset: sel ? 1 : undefined,
                opacity: sel ? 1 : 0.55,
                background: sel ? axis.fill : "transparent",
              }}
              className="transition-opacity hover:opacity-100"
            />
          );
        })}

      {editMode &&
        onActivate &&
        playInEdit &&
        ctx.hits.map((h, i) =>
          isActivatable(h.node) &&
          !disabled?.has(h.id) &&
          playInEdit(h.node) ? (
            <div
              key={`e${i}`}
              title={`${h.node.type} ${h.node.name || ""}${playInEditHint ? ` · ${playInEditHint}` : ""}`}
              onClick={(e) => {
                e.stopPropagation();
                onActivate(h.node);
              }}
              onContextMenu={(e) => {
                e.preventDefault();
                e.stopPropagation();
                const press = e.currentTarget;
                press.style.pointerEvents = "none";
                const under = document.elementFromPoint(e.clientX, e.clientY);
                press.style.pointerEvents = "auto";
                const at =
                  under instanceof HTMLElement &&
                  rootRef.current?.contains(under)
                    ? Number(under.dataset.hit)
                    : NaN;
                const picked = Number.isInteger(at) ? (ctx.hits[at] ?? h) : h;
                onSelect(picked.id, picked.node);
              }}
              style={{
                position: "absolute",
                left: h.x,
                top: h.y,
                width: h.w,
                height: h.h,
                cursor: "pointer",
                pointerEvents: "auto",
              }}
            />
          ) : null,
        )}

      {editMode &&
        ctx.hits
          .filter((h) => h.id === selectedId)
          .slice(0, 1)
          .map((h) => {
            const [aax, aay] = anchor(h.orientation, h.px, h.py, h.pw, h.ph);
            const ex = h.x;
            const ey = h.y;
            const dx = ex - aax;
            const dy = ey - aay;
            const len = Math.max(1, Math.hypot(dx, dy));
            const angle = (Math.atan2(dy, dx) * 180) / Math.PI;
            const [acol, arow] = anchorCell(h.orientation);
            const line = "rgba(56,189,248,0.7)";
            const dotOff = "rgba(120,120,128,0.5)";
            const dotOn = "rgba(56,189,248,0.95)";
            return (
              <div
                key="anchor-viz"
                style={{
                  position: "absolute",
                  left: 0,
                  top: 0,
                  pointerEvents: "none",
                }}
              >
                <div
                  style={{
                    position: "absolute",
                    left: aax,
                    top: aay,
                    width: len,
                    height: 0,
                    borderTop: `1px dashed ${line}`,
                    transformOrigin: "0 0",
                    transform: `rotate(${angle}deg)`,
                  }}
                />

                <div
                  style={{
                    position: "absolute",
                    left: aax - 3,
                    top: aay - 3,
                    width: 6,
                    height: 6,
                    borderRadius: "50%",
                    background: dotOn,
                    boxShadow: "0 0 0 1px rgba(0,0,0,0.6)",
                  }}
                />

                {[0, 1, 2].map((row) =>
                  [0, 1, 2].map((col) => {
                    const cx = h.px + (h.pw * col) / 2;
                    const cy = h.py + (h.ph * row) / 2;
                    const on = col === acol && row === arow;
                    return (
                      <div
                        key={`ad${row}-${col}`}
                        style={{
                          position: "absolute",
                          left: cx - (on ? 3 : 2),
                          top: cy - (on ? 3 : 2),
                          width: on ? 6 : 4,
                          height: on ? 6 : 4,
                          borderRadius: "50%",
                          background: on ? dotOn : dotOff,
                          boxShadow: on
                            ? "0 0 0 1px rgba(0,0,0,0.6)"
                            : undefined,
                        }}
                      />
                    );
                  }),
                )}
              </div>
            );
          })}

      {editMode &&
        ctx.hits
          .filter((h) => h.id === selectedId && !isNaN(h.sx) && !isNaN(h.sy))
          .flatMap((h) => {
            const axes: ResizeAxes = resizeAxes
              ? resizeAxes(
                  h.node,
                  h.node.sprite ? sprites[h.node.sprite] : undefined,
                )
              : "xy";
            if (axes === "none") return [];
            const handle = (
              key: string,
              along: "x" | "y" | "xy",
              box: React.CSSProperties,
              cursor: string,
            ) => (
              <div
                key={key}
                data-resize-handle={along}
                onPointerDown={(e) => beginDrag(e, h, "resize", along)}
                onPointerMove={moveDrag}
                onPointerUp={endDrag}
                title="resize"
                style={{
                  position: "absolute",
                  ...box,
                  cursor,
                  touchAction: "none",
                  pointerEvents: "auto",
                  background: "#38bdf8",
                  border: "1px solid #0a0a0a",
                  borderRadius: 2,
                  boxShadow: "0 0 0 1px rgba(255,255,255,0.4)",
                }}
              />
            );
            const out: React.ReactNode[] = [];
            if (resizeAxes) {
              const barH = Math.min(18, Math.max(8, h.h - 14));
              out.push(
                handle(
                  "resize-x",
                  "x",
                  {
                    left: h.x + h.w - 3,
                    top: h.y + h.h / 2 - barH / 2,
                    width: 6,
                    height: barH,
                  },
                  "ew-resize",
                ),
              );
              if (axes === "xy") {
                const barW = Math.min(18, Math.max(8, h.w - 14));
                out.push(
                  handle(
                    "resize-y",
                    "y",
                    {
                      left: h.x + h.w / 2 - barW / 2,
                      top: h.y + h.h - 3,
                      width: barW,
                      height: 6,
                    },
                    "ns-resize",
                  ),
                );
              }
            }
            if (axes === "xy") {
              out.push(
                handle(
                  "resize-handle",
                  "xy",
                  {
                    left: h.x + h.w - 7,
                    top: h.y + h.h - 7,
                    width: 12,
                    height: 12,
                  },
                  "nwse-resize",
                ),
              );
            }
            return out;
          })}
    </div>
  );
}
