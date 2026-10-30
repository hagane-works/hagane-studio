"use client";

import {
  useEffect,
  useMemo,
  useRef,
  type CSSProperties,
  type ReactNode,
} from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { guiArray, readGuiResponse } from "./guiResponse";
import {
  guiRenderApi,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
  withGameLanguage,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { insetByMargin } from "../gui-renderer/windowMargins";
import {
  withContainerScroll,
  type ScrollOffset,
} from "../gui-renderer/containerScroll";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";
import { namedPrinted } from "./textSources";
import { FieldHit } from "./FieldHit";
import { matchNamePath } from "./gameNavigation";
import { spriteUrlOf } from "./spriteFrame";

export const TECH_TREE = "countrytechtreeview";

const SLOT = 70;

const PIPE_AT = 27;
const PIPE = 16;
const NO_HIDDEN = new Set<string>();
const NO_SCROLL: ScrollOffset = { x: 0, y: 0 };

const WHEEL_STEP = 40;

const SCREEN_H = 1080;

export type TechState = "researched" | "available" | "unavailable";
const STATES: readonly TechState[] = ["researched", "available", "unavailable"];

const subFrameName = (state: TechState) => `GFX_subtechnology_${state}_item_bg`;

export interface GuiTechnology {
  name: string;
  label?: string;
  icon?: string;
  x: number;
  y: number;
  cost: number;
  start_year: number;
  doctrine: boolean;
  researched: boolean;
  state?: TechState;

  card?: boolean;
  card_label?: string;

  card_label_key?: string;

  design_team?: boolean;

  order?: number;

  sub_technologies?: { name: string; slot: number; state: TechState }[];

  special_project?: boolean;
  leads_to?: string[];
  xor?: string[];

  added_by_project?: boolean;
}

export interface GuiTechTree {
  tag: string;
  folder: string;
  folders: { name: string; label?: string; ledger?: string }[];
  technologies: GuiTechnology[];
  researched: number;
  research_slots: number;

  special_project_offset?: { x: number; y: number };

  base_tech_cost?: number;
}

export type TechChange = Partial<
  Pick<GuiTechnology, "label" | "card_label" | "cost" | "researched" | "state">
>;

export function withTechEdits(
  techs: readonly GuiTechnology[],
  saved: Readonly<Record<string, TechChange>>,
): GuiTechnology[] {
  return techs.map((tech) =>
    saved[tech.name] ? { ...tech, ...saved[tech.name] } : tech,
  );
}

const NO_TECH_EDITS: Readonly<Record<string, TechChange>> = {};

export function useTechTree(
  projectId: number,
  tag: string | null,
  date: string | null,
  folder: string,
) {
  return useSWR<GuiTechTree>(
    tag ? ["game-view-tech-tree", projectId, tag, date, folder] : null,
    async () => {
      const query =
        `/tech-tree?tag=${encodeURIComponent(tag as string)}` +
        (date ? `&date=${encodeURIComponent(date)}` : "") +
        (folder ? `&folder=${encodeURIComponent(folder)}` : "");
      const response = await fetch(
        withGameLanguage(`/api/go/projects/${projectId}/gui-render${query}`),
      );
      return readGuiResponse<GuiTechTree>(response);
    },
    { revalidateOnFocus: false },
  );
}

export function backgroundNames(
  folder: string,
  state: TechState,
  small: boolean,
  special = false,
): string[] {
  if (special) {
    return small
      ? [
          `GFX_technology_${folder}_special_project_small_${state}_item_bg`,
          `GFX_technology_${folder}_special_project_${state}_item_bg`,
        ]
      : [`GFX_technology_${folder}_special_project_${state}_item_bg`];
  }
  return small
    ? [
        `GFX_technology_${folder}_small_${state}_item_bg`,
        `GFX_technology_${folder}_${state}_item_bg`,
        `GFX_technology_${state}_item_bg`,
      ]
    : [
        `GFX_technology_${folder}_${state}_item_bg`,
        `GFX_technology_${state}_item_bg`,
      ];
}

export function linkFrame(from: TechState, to: TechState): number {
  if (from !== "researched") return 2;
  if (to === "researched") return 4;
  return to === "available" ? 3 : 2;
}

export interface LineSegment {
  x: number;
  y: number;
  length: number;
  vertical: boolean;
}

export function linkSegments(
  from: { x: number; y: number },
  to: { x: number; y: number },
  axis: "x" | "y",
): { runs: LineSegment[]; corners: { x: number; y: number }[] } {
  const flip = axis === "y";
  const a = flip ? { x: from.y, y: from.x } : from;
  const b = flip ? { x: to.y, y: to.x } : to;
  const runs: LineSegment[] = [];
  const corners: { x: number; y: number }[] = [];
  const centre = SLOT / 2;
  if (Math.abs(a.y - b.y) < 1) {
    runs.push({
      x: Math.min(a.x, b.x) + centre,
      y: a.y + PIPE_AT,
      length: Math.abs(b.x - a.x),
      vertical: false,
    });
  } else {
    const turn = Math.min(a.x + SLOT, b.x);
    if (turn !== a.x) {
      runs.push({
        x: Math.min(a.x, turn) + centre,
        y: a.y + PIPE_AT,
        length: Math.abs(turn - a.x),
        vertical: false,
      });
    }
    runs.push({
      x: turn + PIPE_AT,
      y: Math.min(a.y, b.y) + centre,
      length: Math.abs(b.y - a.y),
      vertical: true,
    });
    corners.push(
      { x: turn + PIPE_AT, y: a.y + PIPE_AT },
      { x: turn + PIPE_AT, y: b.y + PIPE_AT },
    );
    if (b.x > turn)
      runs.push({
        x: turn + centre,
        y: b.y + PIPE_AT,
        length: b.x - turn,
        vertical: false,
      });
  }
  if (!flip) return { runs, corners };
  return {
    runs: runs.map((run) => ({
      x: run.y,
      y: run.x,
      length: run.length,
      vertical: !run.vertical,
    })),
    corners: corners.map((corner) => ({ x: corner.y, y: corner.x })),
  };
}

export function yearAxis(page: GuiNode): "x" | "y" {
  const years: { x: number; y: number }[] = [];
  const walk = (node: GuiNode) => {
    const text = String(node.text ?? "");
    if (/^19\d{2}$/.test(text) && node.position) {
      years.push({
        x: Number(node.position.x) || 0,
        y: Number(node.position.y) || 0,
      });
    }
    for (const child of node.children ?? []) walk(child);
  };
  walk(page);
  if (years.length < 2) return "x";
  const spread = (axis: "x" | "y") =>
    Math.max(...years.map((at) => at[axis])) -
    Math.min(...years.map((at) => at[axis]));
  return spread("x") >= spread("y") ? "x" : "y";
}

export function folderNode(root: GuiNode, folder: string): GuiNode | undefined {
  if (root.name === folder) return root;
  for (const child of root.children ?? []) {
    const found = folderNode(child, folder);
    if (found) return found;
  }
  return undefined;
}

export function folderTabs(node: GuiNode): { folder: string; name: string }[] {
  const tabs: { folder: string; name: string }[] = [];
  const walk = (current: GuiNode, inside: boolean) => {
    const here = inside || current.name === "folder_tabs";
    if (
      here &&
      current.type === "buttontype" &&
      current.name?.endsWith("_tab")
    ) {
      tabs.push({
        folder: current.name.slice(0, -"_tab".length),
        name: current.name,
      });
    }
    for (const child of current.children ?? []) walk(child, here);
  };
  walk(node, false);
  return tabs;
}

export function chainRows(node: GuiNode, folder: string): string[] {
  const rows: string[] = [];
  const walk = (current: GuiNode, inside: boolean) => {
    const here = inside || current.name === folder;
    if (
      here &&
      current.type === "gridboxtype" &&
      current.name?.endsWith("_tree")
    ) {
      rows.push(current.name.slice(0, -"_tree".length));
    }
    for (const child of current.children ?? []) walk(child, here);
  };
  walk(node, false);
  return rows;
}

export const centring = (extent: number, slot = SLOT) => slot / 2 - extent / 2;

export function slotOf(box: GuiNode): { width: number; height: number } {
  const slot = box.raw?.slotsize as
    { width?: unknown; height?: unknown } | undefined;
  return {
    width: Number(slot?.width) || SLOT,
    height: Number(slot?.height) || SLOT,
  };
}

export function pageGridboxes(page: GuiNode): Map<string, GuiNode> {
  const boxes = new Map<string, GuiNode>();
  const walk = (current: GuiNode) => {
    if (
      current.type === "gridboxtype" &&
      current.name &&
      !boxes.has(current.name)
    )
      boxes.set(current.name, current);
    for (const child of current.children ?? []) walk(child);
  };
  walk(page);
  return boxes;
}

export function assignRows(
  techs: GuiTechnology[],
  rows: string[],
): Map<string, string> {
  const byName = new Map(techs.map((tech) => [tech.name, tech]));
  const row = new Map<string, string>();
  for (const root of rows) {
    const queue = [root];
    while (queue.length) {
      const name = queue.shift() as string;
      if (row.has(name) || !byName.has(name)) continue;
      row.set(name, root);
      for (const next of byName.get(name)?.leads_to ?? []) queue.push(next);
    }
  }
  return row;
}

export function techTreeWindow(
  win: GuiWindow,
  folders?: readonly string[],
): GuiWindow {
  const node = structuredClone(win.node) as GuiNode;
  for (const page of node.children ?? []) {
    if (page.raw?.margin && page.name?.endsWith("folder")) insetByMargin(page);
  }
  if (folders?.length)
    dropMissingFolderTabs(
      node,
      new Set(folders.map((name) => name.toLowerCase())),
    );
  return { ...win, node };
}

function dropMissingFolderTabs(
  node: GuiNode,
  folders: ReadonlySet<string>,
  inside = false,
): void {
  const here = inside || node.name === "folder_tabs";
  if (!node.children?.length) return;
  node.children = node.children.filter(
    (child) =>
      !(
        here &&
        child.type === "buttontype" &&
        child.name?.endsWith("_tab") &&
        !folders.has(child.name.slice(0, -"_tab".length).toLowerCase())
      ),
  );
  for (const child of node.children)
    dropMissingFolderTabs(child, folders, here);
}

export const pageContentName = (page: string) => `${page}_content`;

export function scrollTechTree(
  win: GuiWindow,
  offset: ScrollOffset,
): GuiWindow {
  if (!offset.x && !offset.y) return win;
  return {
    ...win,
    node: withContainerScroll(
      win.node,
      (node) => Boolean(node.name?.endsWith("folder_content")),
      offset,
    ),
  };
}

type Box = { x: number; y: number; w: number; h: number };

export function scrollLimits(
  view: Box,
  pieces: readonly Box[],
  offset: ScrollOffset,
): ScrollOffset {
  let right = view.x + view.w;
  let bottom = view.y + view.h;
  for (const piece of pieces) {
    if (piece.w <= 0 || piece.h <= 0) continue;
    right = Math.max(right, piece.x + piece.w + offset.x);
    bottom = Math.max(bottom, piece.y + piece.h + offset.y);
  }
  return {
    x: Math.max(0, Math.round(right - view.x - view.w)),
    y: Math.max(0, Math.round(bottom - view.y - view.h)),
  };
}

export function clampScroll(
  offset: ScrollOffset,
  limits: ScrollOffset,
): ScrollOffset {
  return {
    x: Math.round(Math.min(limits.x, Math.max(0, offset.x))),
    y: Math.round(Math.min(limits.y, Math.max(0, offset.y))),
  };
}

const LINE_SPRITES = {
  across: "GFX_techtree_line_left_states",
  along: "GFX_techtree_line_up_states",
  corner: "GFX_techline_center_all_states",
} as const;

function pipeStyle(
  sprite: GuiSprite | undefined,
  name: string,
  run: LineSegment,
  frame: number,
): CSSProperties | null {
  const url = sprite ? spriteUrlOf(sprite, name) : null;
  if (!url || !sprite) return null;
  const frames = Math.max(1, sprite.frames || 1);
  const width = run.vertical ? PIPE : run.length + PIPE;
  const height = run.vertical ? run.length + PIPE : PIPE;
  return {
    position: "absolute",
    left: run.vertical ? run.x : run.x - PIPE / 2,
    top: run.vertical ? run.y - PIPE / 2 : run.y,
    width,
    height,
    backgroundImage: `url("${url}")`,
    backgroundSize: run.vertical
      ? `${width * frames}px ${height}px`
      : `${width * frames}px ${height}px`,
    backgroundPosition: `${-(frame - 1) * width}px 0`,
    backgroundRepeat: "no-repeat",
    pointerEvents: "none",
  };
}

function TechItem({
  entry,
  tech,
  background,
  picture,
  subFrames,
  frameOffset,
  left,
  top,
}: {
  entry: GuiWindow;
  tech: GuiTechnology;
  background: { name: string; sprite: GuiSprite } | undefined;
  picture: GuiSprite | undefined;

  subFrames: Partial<Record<TechState, { name: string; sprite: GuiSprite }>>;

  frameOffset?: { x: number; y: number };
  left: number;
  top: number;
}) {
  const edit = useEditCanvas();
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;
    if (background && node.background) {
      node.background = { ...node.background, sprite: background.name };

      if (frameOffset && background.name.includes("_special_project_")) {
        const frames = Math.max(1, background.sprite.frames || 1);
        node.background.position = { x: frameOffset.x, y: frameOffset.y };
        node.background.size = {
          width: Math.round(background.sprite.width / frames),
          height: background.sprite.height,
        };
      }
    }
    const subs = new Map(
      (tech.sub_technologies ?? []).map((sub) => [
        `sub_technology_slot_${sub.slot}`,
        sub,
      ]),
    );
    node.children = (node.children ?? []).filter((child) => {
      const name = child.name ?? "";

      if (name === "bonus_icon" || name === "bonus") return false;
      if (name.startsWith("sub_technology_slot")) {
        const sub = subs.get(name);
        const frame = sub ? subFrames[sub.state] : undefined;
        if (sub && frame && child.background)
          child.background = { ...child.background, sprite: frame.name };
        return Boolean(sub);
      }
      if (name === "special_project_specialization_list") return false;
      if (name === "can_assign_design_team_icon")
        return Boolean(tech.design_team);
      return true;
    });
    for (const child of node.children) {
      if (child.name === "Icon" && picture && tech.icon)
        child.sprite = tech.icon;
    }
    assignIds(node);
    const texts: Record<string, string> = {};
    for (const hit of matchNamePath(node, "Name") as (GuiNode & {
      _id?: string;
    })[]) {
      hit.text = hit.text || "Name";
      if (hit._id) texts[hit._id] = tech.card_label ?? tech.label ?? tech.name;
    }

    const printed = namedPrinted(
      node,
      "Name",
      tech.card_label ?? tech.label ?? tech.name,
      tech.card_label ? tech.card_label_key : tech.name,
    );
    return { node, texts, printed };
  }, [entry.node, background, picture, tech, subFrames, frameOffset]);
  const sprites = useMemo(() => {
    const out = { ...entry.sprites };
    if (background) out[background.name] = background.sprite;
    if (picture && tech.icon) out[tech.icon] = picture;
    for (const frame of Object.values(subFrames))
      if (frame) out[frame.name] = frame.sprite;
    return out;
  }, [entry.sprites, background, picture, tech.icon, subFrames]);
  const size = entry.node.size as { width?: number; height?: number } | null;
  return (
    <div
      style={{ position: "absolute", left, top }}
      title={`${tech.label || tech.name} · ${tech.start_year}`}
    >
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${tech.name}`,
          { ...entry, sprites },
          built.node,
          sprites,
          built.texts,
          built.printed,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={Number(size?.width) || 72}
        baseH={Number(size?.height) || 72}
      />
    </div>
  );
}

export type ScrollUpdate = (
  update: (now: ScrollOffset) => ScrollOffset,
) => void;

export function PageScroll({
  box,
  scroll,
  limits,
  step,
  dragLeft,
  onScroll,
}: {
  box: Box;
  scroll: ScrollOffset;
  limits: ScrollOffset;
  step: number;
  dragLeft: boolean;
  onScroll: ScrollUpdate;
}) {
  const marker = useRef<HTMLDivElement | null>(null);
  const live = useRef({ box, scroll, limits, step, dragLeft, onScroll });
  useEffect(() => {
    live.current = { box, scroll, limits, step, dragLeft, onScroll };
  });
  useEffect(() => {
    const over = (e: MouseEvent) => {
      const el = marker.current;
      if (
        !el ||
        !(e.target instanceof Element) ||
        e.target.closest("[data-game-panel]")
      )
        return false;
      const r = el.getBoundingClientRect();
      if (
        e.clientX < r.left ||
        e.clientX >= r.right ||
        e.clientY < r.top ||
        e.clientY >= r.bottom
      )
        return false;
      return Boolean(
        el.closest("[data-screen-layer]")?.parentElement?.contains(e.target),
      );
    };
    const scrolls = () =>
      live.current.limits.x > 0 || live.current.limits.y > 0;
    const move = (to: (now: ScrollOffset) => ScrollOffset) =>
      live.current.onScroll((now) => clampScroll(to(now), live.current.limits));
    const notches = (delta: number) =>
      delta
        ? Math.sign(delta) * Math.max(1, Math.round(Math.abs(delta) / 100))
        : 0;
    const onWheel = (e: WheelEvent) => {
      if (!scrolls() || !over(e)) return;
      e.preventDefault();
      e.stopPropagation();
      const { limits, step } = live.current;
      let across = notches(e.deltaX);
      let along = notches(e.deltaY);
      if ((e.shiftKey || !limits.y) && !across) [across, along] = [along, 0];
      move((now) => ({ x: now.x + across * step, y: now.y + along * step }));
    };
    let drag: {
      x: number;
      y: number;
      from: ScrollOffset | null;
      moved: boolean;
    } | null = null;
    const onDown = (e: PointerEvent) => {
      if (
        !(e.button === 1 || (e.button === 0 && live.current.dragLeft)) ||
        !scrolls() ||
        !over(e)
      )
        return;

      if (e.button === 1) e.preventDefault();
      drag = { x: e.clientX, y: e.clientY, from: null, moved: false };
    };
    const onMove = (e: PointerEvent) => {
      if (!drag) return;
      const dx = e.clientX - drag.x;
      const dy = e.clientY - drag.y;

      if (!drag.moved && Math.abs(dx) <= 4 && Math.abs(dy) <= 4) return;
      drag.moved = true;

      const from = (drag.from ??= live.current.scroll);
      const el = marker.current;
      const scale =
        (el
          ? el.getBoundingClientRect().width / Math.max(1, live.current.box.w)
          : 1) || 1;
      move(() => ({ x: from.x - dx / scale, y: from.y - dy / scale }));
    };
    const swallowClick = (e: MouseEvent) => {
      e.stopPropagation();
      e.preventDefault();
    };
    const onUp = () => {
      if (drag?.moved) {
        window.addEventListener("click", swallowClick, {
          capture: true,
          once: true,
        });
        window.setTimeout(
          () =>
            window.removeEventListener("click", swallowClick, {
              capture: true,
            }),
          0,
        );
      }
      drag = null;
    };
    window.addEventListener("wheel", onWheel, {
      capture: true,
      passive: false,
    });
    window.addEventListener("pointerdown", onDown, { capture: true });
    window.addEventListener("pointermove", onMove, { capture: true });
    window.addEventListener("pointerup", onUp, { capture: true });
    window.addEventListener("pointercancel", onUp, { capture: true });
    return () => {
      window.removeEventListener("wheel", onWheel, { capture: true });
      window.removeEventListener("pointerdown", onDown, { capture: true });
      window.removeEventListener("pointermove", onMove, { capture: true });
      window.removeEventListener("pointerup", onUp, { capture: true });
      window.removeEventListener("pointercancel", onUp, { capture: true });
      window.removeEventListener("click", swallowClick, { capture: true });
    };
  }, []);
  return (
    <div
      ref={marker}
      aria-hidden
      style={{
        position: "absolute",
        left: box.x,
        top: box.y,
        width: box.w,
        height: box.h,
        pointerEvents: "none",
      }}
    />
  );
}

export function TechTreeLayer({
  win,
  projectId,
  tag,
  date,
  folder = "",
  onPickFolder,
  saved = NO_TECH_EDITS,
  onPick,
  editable = true,
  scroll = NO_SCROLL,
  onScroll,
  dragScroll = true,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  folder?: string;

  scroll?: ScrollOffset;

  onScroll?: ScrollUpdate;

  dragScroll?: boolean;
  onPickFolder?: (folder: string) => void;

  saved?: Readonly<Record<string, TechChange>>;

  onPick?: (tech: GuiTechnology, baseTechCost: number | undefined) => void;

  editable?: boolean;
}) {
  const tLayer = useTranslations("gameViewResearch.layer");
  const active = win.name === TECH_TREE;
  const fetched = useTechTree(
    projectId,
    active ? tag : null,
    date ?? null,
    folder,
  );
  const tree = useMemo(
    () => ({
      data: fetched.data
        ? {
            ...fetched.data,
            technologies: withTechEdits(
              guiArray(fetched.data.technologies),
              saved,
            ),
          }
        : undefined,
    }),
    [fetched.data, saved],
  );
  const page = tree.data?.folder ?? "";
  const wide = useTemplateWindow(
    projectId,
    active && page ? `techtree_${page}_item` : null,
  );
  const small = useTemplateWindow(
    projectId,
    active && page ? `techtree_${page}_small_item` : null,
  );
  const rows = useMemo(
    () => (active && page ? chainRows(win.node, page) : []),
    [active, win, page],
  );
  const assigned = useMemo(
    () => assignRows(tree.data?.technologies ?? [], rows),
    [tree.data, rows],
  );
  const spriteNames = useMemo(() => {
    const names = new Set<string>(Object.values(LINE_SPRITES));
    for (const tech of tree.data?.technologies ?? [])
      if (tech.icon) names.add(tech.icon);
    if (page) {
      for (const state of STATES) {
        for (const special of [false, true]) {
          for (const name of [
            ...backgroundNames(page, state, true, special),
            ...backgroundNames(page, state, false, special),
          ])
            names.add(name);
        }
        names.add(subFrameName(state));
      }
    }
    return [...names].sort();
  }, [tree.data, page]);
  const sprites = useSWR<Record<string, GuiSprite>>(
    spriteNames.length
      ? [
          "game-view-sprites",
          projectId,
          "tech-tree",
          page,
          spriteNames.join(","),
        ]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(spriteNames);
    },
    { revalidateOnFocus: false },
  );
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );

  if (!active || !tree.data || !wide.data) return null;
  const known = (name: string) => {
    const sprite = sprites.data?.[name];
    return sprite?.resolved ? sprite : undefined;
  };
  const backgroundFor = (
    state: TechState,
    isSmall: boolean,
    special: boolean,
  ) => {
    for (const name of backgroundNames(page, state, isSmall, special)) {
      const sprite = known(name);
      if (sprite) return { name, sprite };
    }
    return undefined;
  };

  const subFrames: Partial<
    Record<TechState, { name: string; sprite: GuiSprite }>
  > = {};
  for (const state of STATES) {
    const sprite = known(subFrameName(state));
    if (sprite) subFrames[state] = { name: subFrameName(state), sprite };
  }

  const frameOffset = tree.data.special_project_offset;
  const baseTechCost = tree.data.base_tech_cost;

  const folderNames = new Set(
    tree.data.folders.map((entry) => entry.name.toLowerCase()),
  );
  const tabs = folderTabs(win.node).filter(
    (tab) => !folderNames.size || folderNames.has(tab.folder.toLowerCase()),
  );
  const pageNode = page ? folderNode(win.node, page) : undefined;
  const axis = pageNode ? yearAxis(pageNode) : "x";

  const byName = new Map(
    tree.data.technologies.map((tech) => [tech.name, tech]),
  );
  const gridboxes = pageNode
    ? pageGridboxes(pageNode)
    : new Map<string, GuiNode>();

  const cellIn = (root: string | undefined, tech: GuiTechnology) => {
    const box = root ? gridboxes.get(`${root}_tree`) : undefined;
    const boxId = (box as (GuiNode & { _id?: string }) | undefined)?._id;
    const rect = boxId ? rects[boxId] : undefined;
    if (!box || !rect) return undefined;

    const slot = slotOf(box);
    const along = tech.y * (axis === "x" ? slot.width : slot.height);

    const across = tech.x * (axis === "x" ? slot.height : slot.width);
    const format = String(box.format ?? box.raw?.format ?? "").toLowerCase();
    return {
      x:
        rect.x +
        (axis === "x" ? along : across) -
        (format === "up" ? centring(rect.w, slot.width) : 0),
      y:
        rect.y +
        (axis === "x" ? across : along) -
        (format === "left" ? centring(rect.h, slot.height) : 0),
    };
  };
  const cells = new Map<string, { x: number; y: number }>();
  for (const tech of tree.data.technologies) {
    const cell = cellIn(assigned.get(tech.name), tech);
    if (cell) cells.set(tech.name, cell);
  }

  const placed = tree.data.technologies.flatMap((tech) => {
    const cell = cells.get(tech.name);
    if (!cell) return [];
    const isSmall = !tech.card && Boolean(small.data);
    const entry = (isSmall ? small.data : wide.data) as GuiWindow;
    const at = entry.node.position ?? { x: 0, y: 0 };
    const size = entry.node.size as { width?: number; height?: number } | null;
    return [
      {
        tech,
        isSmall,
        entry,
        left: cell.x + (Number(at.x) || 0),
        top: cell.y + (Number(at.y) || 0),
        w: Number(size?.width) || 72,
        h: Number(size?.height) || 72,
      },
    ];
  });

  const content = pageNode?.children?.find(
    (child) => child.name === pageContentName(page),
  );
  const contentId = (content as (GuiNode & { _id?: string }) | undefined)?._id;
  const box = contentId ? rects[contentId] : undefined;
  const shownAt = (win.node.raw?.show_position ?? win.node.position ?? {}) as {
    y?: number | string;
  };
  const screenBottom = SCREEN_H - (Number(shownAt.y) || 0);
  const view = box
    ? { ...box, h: Math.max(0, Math.min(box.y + box.h, screenBottom) - box.y) }
    : undefined;
  const pieces = contentId
    ? Object.entries(rects).flatMap(([id, rect]) =>
        id.startsWith(`${contentId}/`) && !rect.hidden ? [rect] : [],
      )
    : [];
  const limits = view
    ? scrollLimits(
        view,
        [
          ...pieces,
          ...placed.map((item) => ({
            x: item.left,
            y: item.top,
            w: item.w,
            h: item.h,
          })),
        ],
        scroll,
      )
    : NO_SCROLL;
  const wheelStep = Number(pageNode?.raw?.scroll_wheel_factor) || WHEEL_STEP;

  const clip = (children: ReactNode) =>
    box ? (
      <div
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
          {children}
        </div>
      </div>
    ) : (
      children
    );
  const stateOf = (tech: GuiTechnology | undefined): TechState =>
    tech?.state ?? (tech?.researched ? "researched" : "unavailable");

  const links = [...tree.data.technologies]
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
    .flatMap((tech) =>
      (tech.leads_to ?? []).map((next) => ({
        tech,
        next,
        frame: linkFrame(stateOf(tech), stateOf(byName.get(next))),
      })),
    )
    .sort((a, b) => a.frame - b.frame);

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {view && onScroll && (
        <PageScroll
          box={view}
          scroll={scroll}
          limits={limits}
          step={wheelStep}
          dragLeft={dragScroll}
          onScroll={onScroll}
        />
      )}

      {clip(
        links.flatMap(({ tech, next, frame }) => {
          const from = cellIn(assigned.get(next), tech);
          const to = cells.get(next);
          if (!from || !to) return [];
          const { runs, corners } = linkSegments(from, to, axis);
          return [
            ...runs.map((run, at) => {
              const name = run.vertical
                ? LINE_SPRITES.along
                : LINE_SPRITES.across;
              const style = pipeStyle(known(name), name, run, frame);
              return style ? (
                <div key={`${tech.name}->${next}:${at}`} style={style} />
              ) : null;
            }),
            ...corners.map((corner, at) => {
              const sprite = known(LINE_SPRITES.corner);
              const url = sprite
                ? spriteUrlOf(sprite, LINE_SPRITES.corner)
                : null;
              if (!sprite || !url) return null;
              const frames = Math.max(1, sprite.frames || 1);
              return (
                <div
                  key={`${tech.name}->${next}:c${at}`}
                  style={{
                    position: "absolute",
                    left: corner.x,
                    top: corner.y,
                    width: PIPE,
                    height: PIPE,
                    backgroundImage: `url("${url}")`,
                    backgroundSize: `${PIPE * frames}px ${PIPE}px`,
                    backgroundPosition: `${-(frame - 1) * PIPE}px 0`,
                    pointerEvents: "none",
                  }}
                />
              );
            }),
          ];
        }),
      )}
      {onPickFolder &&
        tabs.map((tab) => {
          const rect = rects[`name:${tab.name}`];
          if (!rect) return null;
          return (
            <button
              key={tab.name}
              type="button"
              aria-label={tab.folder}
              aria-pressed={tab.folder === page}
              onClick={() => onPickFolder(tab.folder)}
              style={{
                position: "absolute",
                left: rect.x,
                top: rect.y,
                width: rect.w || 88,
                height: rect.h || 30,
                background: "transparent",
                border: "none",
                padding: 0,
                cursor: "inherit",
                pointerEvents: "auto",
              }}
            />
          );
        })}
      {clip(
        placed.flatMap(({ tech, isSmall, entry, left, top, w, h }) => {
          const item = (
            <TechItem
              key={tech.name}
              entry={entry}
              tech={tech}
              background={
                (tech.special_project
                  ? backgroundFor(stateOf(tech), isSmall, true)
                  : undefined) ?? backgroundFor(stateOf(tech), isSmall, false)
              }
              picture={tech.icon ? known(tech.icon) : undefined}
              subFrames={subFrames}
              frameOffset={frameOffset}
              left={left}
              top={top}
            />
          );
          if (!onPick) return [item];
          return [
            item,
            <FieldHit
              key={`${tech.name}:pick`}
              x={left}
              y={top}
              w={w}
              h={h}
              label={tLayer("tech", {
                name: tech.card_label || tech.label || tech.name,
              })}
              onClick={() => onPick(tech, baseTechCost)}
              appearance={editable ? "edit" : "inspect"}
            />,
          ];
        }),
      )}
    </div>
  );
}
