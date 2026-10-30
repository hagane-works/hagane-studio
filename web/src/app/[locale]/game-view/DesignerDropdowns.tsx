"use client";

import { useMemo, useRef, useState, type CSSProperties } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "../gui-renderer/GuiCanvas";
import { gameTooltipProps, hideGameTooltip } from "../gui-renderer/GameTooltip";
import { useTemplateWindow } from "./CountrySelectLayer";
import { matchNamePath } from "./gameNavigation";
import { useEditCanvas } from "./editContext";
import { expandButtonOf, frameStyle } from "./spriteFrame";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export interface DropdownItem {
  key: string;
  label: string;

  icon?: string;
  selected?: boolean;
  tooltip?: string;
}

const ENTRY_BG = "GFX_mini_tooltip";

export const SLIDER = {
  bg: "GFX_scrollbar_vertical_bg",
  track: "GFX_scroll_track",
  up: "GFX_scroll_up",
  down: "GFX_scroll_down",
  knob: "GFX_scroll_drager",
} as const;
export const SLIDER_W = 18;

export const SLIDER_INSET = 5;

const NO_HIDDEN: Set<string> = new Set();

export function expandedGeometry(box: GuiNode | undefined) {
  if (!box) return undefined;
  const open = (box.children ?? []).find(
    (child) => child.type === "expandedwindow",
  );
  if (!open) return undefined;
  const raw = (open.raw ?? {}) as Record<
    string,
    Record<string, number> | undefined
  >;
  const at = raw.show_position ?? { x: 0, y: 0 };
  const margin = (open.raw?.margin ?? {}) as Record<string, number>;
  const grid = (open.children ?? []).find(
    (child) => child.type === "gridboxtype",
  );
  const slot = ((grid?.raw ?? {}) as Record<string, Record<string, number>>)
    .slotsize;
  if (!slot) return undefined;
  return {
    node: open,
    w: Number(open.size?.width ?? 0),
    h: Number(open.size?.height ?? 0),
    at: { x: Number(at.x ?? 0), y: Number(at.y ?? 0) },
    grid: {
      x: Number(grid?.position?.x ?? 0),
      y: Number(grid?.position?.y ?? 0),
    },
    slot: { w: Number(slot.width ?? 0), h: Number(slot.height ?? 0) },
    margin: {
      top: Number(margin.top ?? 0),
      bottom: Number(margin.bottom ?? 0),
      right: Number(margin.right ?? 0),
    },
  };
}

function box(
  name: string,
  sprite: string,
  width: number,
  height: number,
  x = 0,
  y = 0,
): GuiNode {
  return {
    type: "containerwindowtype",
    name,
    position: { x, y },
    size: { width, height },
    orientation: null,
    origo: null,
    scale: null,
    sprite: null,
    frame: null,
    text: null,
    text_source: null,
    font: null,
    format: null,
    maxwidth: null,
    maxheight: null,
    text_color: null,
    centerposition: false,
    alwaystransparent: true,
    background: { ...icon(`${name}_bg`, sprite, 0, 0), type: "icontype" },
    children: [],
  };
}

function icon(name: string, sprite: string, x: number, y: number): GuiNode {
  return {
    type: "icontype",
    name,
    position: { x, y },
    size: null,
    orientation: null,
    origo: null,
    scale: null,
    sprite,
    frame: null,
    text: null,
    text_source: null,
    font: null,
    format: null,
    maxwidth: null,
    maxheight: null,
    text_color: null,
    centerposition: false,
    alwaystransparent: true,
    background: null,
    children: [],
  };
}

function Row({
  entry,
  item,
  sprites,
  left,
  top,
  onPick,
}: {
  entry: GuiWindow;
  item: DropdownItem;
  sprites: Record<string, GuiSprite>;
  left: number;
  top: number;
  onPick: () => void;
}) {
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;
    for (const child of node.children ?? []) {
      if (child.name === "division") {
        child.sprite = item.icon ?? null;
        child.frame = null;
      }
    }
    if (!item.icon)
      node.children = (node.children ?? []).filter(
        (child) => child.name !== "division",
      );
    assignIds(node);
    const texts: Record<string, string> = {};
    for (const hit of matchNamePath(node, "name") as (GuiNode & {
      _id?: string;
    })[]) {
      if (hit._id)
        texts[hit._id] = item.selected ? `§Y${item.label}§!` : item.label;
    }
    return { node, texts };
  }, [entry.node, item.icon, item.label, item.selected]);
  const allSprites = useMemo(
    () => ({ ...entry.sprites, ...sprites }),
    [entry.sprites, sprites],
  );
  const edit = useEditCanvas();
  const size = entry.node.size as { width?: number; height?: number } | null;
  const w = Number(size?.width) || 198;
  const h = Number(size?.height) || 50;
  return (
    <div style={{ position: "absolute", left, top, width: w, height: h }}>
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${item.key}`,
          entry,
          built.node,
          allSprites,
          built.texts,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={w}
        baseH={h}
      />
      <button
        type="button"
        aria-label={item.label}
        onClick={() => {
          hideGameTooltip(`designer-dropdown:${item.key}`);
          onPick();
        }}
        {...(item.tooltip
          ? gameTooltipProps(item.tooltip, {
              key: `designer-dropdown:${item.key}`,
            })
          : {})}
        style={{
          position: "absolute",
          inset: 0,
          background: "transparent",
          border: "none",
          padding: 0,
          cursor: "inherit",
          pointerEvents: "auto",
        }}
      />
    </div>
  );
}

export function Slider({
  height,
  scroll,
  sprites,
  onStep,
}: {
  height: number;
  scroll: { top: number; max: number };
  sprites: Record<string, GuiSprite>;
  onStep: (dir: -1 | 1) => void;
}) {
  const up = sprites[SLIDER.up];
  const down = sprites[SLIDER.down];
  const knob = sprites[SLIDER.knob];
  const frame = useMemo(() => {
    const node = box("slider", SLIDER.bg, SLIDER_W, height);
    node.children = [
      box(
        "track",
        SLIDER.track,
        Math.max(1, sprites[SLIDER.track]?.width ?? 12),
        height,
        3,
        0,
      ),
    ];
    assignIds(node);
    return node;
  }, [height, sprites]);
  const edit = useEditCanvas();
  const upH = up ? up.height : 16;
  const downH = down ? down.height : 16;
  const knobH = knob ? knob.height : 18;

  const run = Math.max(0, height - upH - downH - knobH - 6);
  const knobTop =
    3 +
    upH +
    (scroll.max > 0 ? Math.round((scroll.top / scroll.max) * run) : 0);
  const arrow = (dir: -1 | 1): CSSProperties => ({
    position: "absolute",
    left: dir < 0 ? 2 : 1,
    top: dir < 0 ? 3 : height - downH - 3,
    width: 16,
    height: dir < 0 ? upH : downH,
    background: "transparent",
    border: "none",
    padding: 0,
    cursor: "inherit",
    pointerEvents: "auto",
  });
  return (
    <>
      <GuiCanvas
        node={frame}
        {...edit(
          "designer-dropdown:slider",
          {
            name: "slider",
            file: "",
            node: frame,
            sprites,
            texts: {},
            fonts: {},
          } as unknown as GuiWindow,
          frame,
          sprites,
        )}
        texts={{}}
        fonts={{}}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={SLIDER_W}
        baseH={height}
      />
      {up && <div style={frameStyle(up, 2, 3, { name: SLIDER.up })} />}
      {down && (
        <div
          style={frameStyle(down, 1, height - downH - 3, { name: SLIDER.down })}
        />
      )}
      {knob && (
        <div style={frameStyle(knob, -2, knobTop - 2, { name: SLIDER.knob })} />
      )}
      <button
        type="button"
        aria-label="scroll up"
        onClick={() => onStep(-1)}
        style={arrow(-1)}
      />
      <button
        type="button"
        aria-label="scroll down"
        onClick={() => onStep(1)}
        style={arrow(1)}
      />
    </>
  );
}

export function DesignerDropdown({
  projectId,
  name,
  box: dropBox,
  at,
  windowSprites,
  entryName,
  items,
  open,
  onToggle,
  onPick,
}: {
  projectId: number;

  name: string;

  box: GuiNode | undefined;

  at: { x: number; y: number } | undefined;
  windowSprites: Record<string, GuiSprite>;

  entryName: string;
  items: readonly DropdownItem[];
  open: boolean;
  onToggle: () => void;
  onPick: (key: string) => void;
}) {
  const arrow = useMemo(() => expandButtonOf(dropBox), [dropBox]);
  const list = useMemo(() => expandedGeometry(dropBox), [dropBox]);
  const entry = useTemplateWindow(projectId, open ? entryName : null);
  const frameSprite = list?.node.background?.sprite ?? null;
  const spriteNames = useMemo(() => {
    const names = new Set<string>([ENTRY_BG, ...Object.values(SLIDER)]);
    if (arrow) names.add(arrow.sprite);
    if (frameSprite) names.add(frameSprite);
    for (const item of items) if (item.icon) names.add(item.icon);
    return [...names].sort();
  }, [arrow, frameSprite, items]);
  const sprites = useSWR<Record<string, GuiSprite>>(
    [
      "game-view-sprites",
      projectId,
      "designer-dropdown",
      spriteNames.join(","),
    ],
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(spriteNames);
    },
    { revalidateOnFocus: false },
  );
  const all = useMemo(
    () => ({ ...windowSprites, ...(sprites.data ?? {}) }),
    [windowSprites, sprites.data],
  );
  const frame = useMemo(() => {
    if (!list || !frameSprite) return null;
    const node = box(`${name}_list`, frameSprite, list.w, list.h);
    assignIds(node);
    return node;
  }, [list, frameSprite, name]);
  const edit = useEditCanvas();
  const scroller = useRef<HTMLDivElement>(null);
  const [scroll, setScroll] = useState({ top: 0, max: 0 });

  if (!dropBox || !at) return null;
  const arrowSprite = arrow ? all[arrow.sprite] : undefined;
  const arrowW = arrowSprite
    ? Math.round(arrowSprite.width / Math.max(1, arrowSprite.frames || 1))
    : 0;
  const boxW = Number(dropBox.size?.width ?? 220);
  const boxH = Number(dropBox.size?.height ?? 30);
  const viewH = list ? list.h - list.margin.top - list.margin.bottom : 0;
  const rowsH = list ? items.length * list.slot.h : 0;
  const scrolls = rowsH > viewH;
  const measure = () => {
    const el = scroller.current;
    if (el)
      setScroll({
        top: el.scrollTop,
        max: Math.max(0, el.scrollHeight - el.clientHeight),
      });
  };

  return (
    <>
      {arrow && arrowSprite && !arrowSprite.missing_file && (
        <div
          style={frameStyle(arrowSprite, at.x + arrow.x, at.y + arrow.y, {
            scale: arrow.scale,
            name: arrow.sprite,
          })}
        />
      )}
      <button
        type="button"
        aria-label={name}
        aria-expanded={open}
        onClick={onToggle}
        style={{
          position: "absolute",
          left: at.x,
          top: at.y,

          width: Math.max(boxW, arrow ? arrow.x + arrowW : 0),
          height: boxH,
          background: "transparent",
          border: "none",
          padding: 0,
          cursor: "inherit",
          pointerEvents: "auto",
          zIndex: 6,
        }}
      />
      {open && list && entry.data && (
        <div
          style={{
            position: "absolute",
            left: at.x + list.at.x,
            top: at.y + list.at.y,
            width: list.w,
            height: list.h,
            pointerEvents: "auto",
            zIndex: 8,
          }}
        >
          {frame && (
            <GuiCanvas
              node={frame}
              {...edit(
                `${name}:list`,
                {
                  name: `${name}_list`,
                  file: "",
                  node: frame,
                  sprites: all,
                  texts: {},
                  fonts: {},
                } as unknown as GuiWindow,
                frame,
                all,
              )}
              texts={{}}
              fonts={{}}
              hidden={NO_HIDDEN}
              isolateId={null}
              transparent
              baseW={list.w}
              baseH={list.h}
            />
          )}
          <div
            ref={scroller}
            onScroll={measure}
            onPointerEnter={measure}
            style={{
              position: "absolute",
              left: list.grid.x,
              top: list.margin.top + list.grid.y,
              width:
                list.w - list.grid.x - (scrolls ? SLIDER_W + SLIDER_INSET : 0),
              height: viewH,
              overflowY: "auto",
              overflowX: "hidden",
              ...GAME_SCROLLBAR,
            }}
          >
            <div style={{ position: "relative", height: rowsH }}>
              {items.map((item, row) => (
                <Row
                  key={item.key}
                  entry={entry.data as GuiWindow}
                  item={item}
                  sprites={all}
                  left={0}
                  top={row * list.slot.h}
                  onPick={() => onPick(item.key)}
                />
              ))}
            </div>
          </div>
          {scrolls && (
            <div
              style={{
                position: "absolute",
                left: list.w - SLIDER_INSET - SLIDER_W,
                top: list.margin.top,
                width: SLIDER_W,
                height: viewH,
              }}
            >
              <Slider
                height={viewH}
                scroll={scroll}
                sprites={all}
                onStep={(dir) => {
                  const el = scroller.current;
                  if (!el) return;
                  el.scrollTop += dir * list.slot.h;
                  measure();
                }}
              />
            </div>
          )}
        </div>
      )}
    </>
  );
}
