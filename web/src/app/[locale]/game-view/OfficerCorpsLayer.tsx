"use client";

import { useMemo, type ReactNode } from "react";
import useSWR from "swr";
import { readGuiResponse } from "./guiResponse";
import {
  guiRenderApi,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
  withGameLanguage,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { matchNamePath } from "./gameNavigation";
import { BitmapText } from "../gui-renderer/BitmapText";
import { emptySlotSprite } from "./CharacterLayer";
import { useEditCanvas } from "./editContext";
import { personPrinted } from "./textSources";
import { useTemplateWindow } from "./CountrySelectLayer";
import { fillRuntimeSlots } from "../gui-renderer/tooltipText";
import { withContainerScroll } from "../gui-renderer/containerScroll";
import { PageScroll, type ScrollUpdate } from "./TechTreeLayer";
import { PageScrollbar } from "./PageScrollbar";

export const OFFICER_CORPS = "countryofficercorpview";

export const OFFICER_GRIDS: readonly {
  slot: string;
  grid: string;
  entry: string;
}[] = [
  {
    slot: "high_command",
    grid: "high_command_advisors",
    entry: "high_command_entry",
  },
  { slot: "theorist", grid: "theorist_advisors", entry: "theorist_entry" },
  {
    slot: "army_chief",
    grid: "army_chief_advisors",
    entry: "army_chief_entry",
  },
  {
    slot: "navy_chief",
    grid: "navy_chief_advisors",
    entry: "navy_chief_entry",
  },
  { slot: "air_chief", grid: "air_chief_advisors", entry: "air_chief_entry" },
];

const ENTRY_W = 84;
const ENTRY_H = 150;

const NAME_W = 70;
const SCREEN_H = 1080;

const NOTHING_HIDDEN = new Set<string>();

export interface GuiOfficer {
  token: string;
  name: string;
  slot: string;
  portrait?: string;
  cost?: number;
  hired?: boolean;
  traits?: { token: string; label?: string }[];
}

export interface GuiOfficerCorps {
  tag: string;
  slots: Record<string, GuiOfficer[]>;

  seats?: Record<string, number>;

  doctrines?: Record<
    string,
    { token: string; label?: string; icon: string; icon_sprite?: GuiSprite }
  >;
  spirits: { token: string; label?: string; icon?: string }[];

  spirit_slots?: Record<string, SpiritSlot[]>;

  hq_templates?: {
    name: string;
    icon?: string;
    priority?: number;
    command_power?: number;
    deploy_days?: number;
  }[];
}

export interface SpiritSlot {
  group: string;
  cost?: number;
  affordable?: boolean;
}

const PLATE_DISABLED = "GFX_officer_corp_spirit_bg_disabled";

const SPIRIT_VIEWS: Record<string, string> = {
  army: "army_spirits_view",
  navy: "navy_spirits_view",
  air: "air_spirits_view",
};

const SPIRIT_SECTIONS = ["army_header", "navy_header", "air_header"];

export function spiritEntryNames(corps: GuiOfficerCorps | undefined): string[] {
  return Object.values(corps?.spirit_slots ?? {})
    .flat()
    .map((slot) => `${slot.group}_entry`)
    .sort();
}

export function useSpiritEntries(projectId: number, names: readonly string[]) {
  return useSWR<Record<string, GuiWindow>>(
    names.length
      ? ["game-view-spirit-entries", projectId, names.join(",")]
      : null,
    async () => {
      guiRenderApi.setProjectId(projectId);
      const [wins, plate] = await Promise.all([
        Promise.all(names.map((name) => guiRenderApi.getWindow(name))),
        guiRenderApi.sprites([PLATE_DISABLED]),
      ]);

      return Object.fromEntries(
        names.map((name, at) => [
          name,
          at === 0
            ? { ...wins[at], sprites: { ...wins[at].sprites, ...plate } }
            : wins[at],
        ]),
      );
    },
    { revalidateOnFocus: false },
  );
}

export function useOfficerCorps(
  projectId: number,
  tag: string | null,
  date: string | null,
) {
  return useSWR<GuiOfficerCorps>(
    tag ? ["game-view-officers", projectId, tag, date] : null,
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/officer-corps?tag=${encodeURIComponent(tag as string)}`,
        ) + (date ? `&date=${encodeURIComponent(date)}` : ""),
      );
      return readGuiResponse<GuiOfficerCorps>(response);
    },
    { revalidateOnFocus: false },
  );
}

export function officerSeats(
  corps: GuiOfficerCorps,
  slot: string,
): (GuiOfficer | null)[] {
  const holders = (corps.slots?.[slot] ?? []).filter((person) => person.hired);
  const seats = Math.max(corps.seats?.[slot] ?? 1, holders.length);
  return Array.from({ length: seats }, (_, at) => holders[at] ?? null);
}

export function officerCorpsWindow(
  win: GuiWindow,
  corps?: GuiOfficerCorps,
  armyHQ = true,
  spiritEntries?: Record<string, GuiWindow>,
): GuiWindow {
  const node = structuredClone(win.node) as GuiNode;
  const sprites = { ...win.sprites };
  const view = (node.children ?? []).find(
    (child) => child.name === "officer_core_view",
  );
  const shift = view?.children?.find(
    (child) => child.type === "positiontype" && child.name === "no_taog_offset",
  );
  if (!armyHQ && view?.position && shift?.position) {
    view.position = {
      x: (Number(view.position.x) || 0) + (Number(shift.position.x) || 0),
      y: (Number(view.position.y) || 0) + (Number(shift.position.y) || 0),
    };
  }
  const margin = (view?.raw?.margin ?? {}) as {
    top?: number;
    left?: number;
    right?: number;
  };
  const inner =
    (Number(node.size?.width) || 0) -
    (Number(margin.left) || 0) -
    (Number(margin.right) || 0);
  for (const child of view?.children ?? []) {
    if (child.type === "positiontype" || !child.position) continue;
    child.position = {
      x: (Number(child.position.x) || 0) + (Number(margin.left) || 0),
      y: (Number(child.position.y) || 0) + (Number(margin.top) || 0),
    };
    const width = child.size?.width;
    if (
      inner > 0 &&
      typeof width === "string" &&
      /^\d+(\.\d+)?%$/.test(width)
    ) {
      child.size = {
        ...child.size,
        width: Math.round((parseFloat(width) / 100) * inner),
      };
    }
  }

  if (view && corps?.spirit_slots && spiritEntries)
    addSpirits(view, corps.spirit_slots, spiritEntries, sprites);

  if (!corps?.doctrines?.special_forces) {
    const header = findNode(node, "special_forces_header");
    if (header)
      header.children = (header.children ?? []).filter(
        (child) => child.name !== "special_forces_doctrine_button",
      );
  }

  for (const folder of ["land", "naval", "air", "special_forces"]) {
    const button = findNode(node, `${folder}_doctrine_button`)?.children?.find(
      (child) => child.name === "button",
    );
    if (button) button.name = `${folder}_doctrine_ring`;
  }

  for (const [folder, doctrine] of Object.entries(corps?.doctrines ?? {})) {
    if (!doctrine.icon_sprite?.resolved) continue;
    const ring = findNode(node, `${folder}_doctrine_button`);
    const icon = ring?.children?.find((child) => child.name === "icon");
    if (!icon) continue;
    icon.sprite = doctrine.icon;
    sprites[doctrine.icon] = doctrine.icon_sprite;
  }
  return { ...win, node, sprites };
}

function addSpirits(
  view: GuiNode,
  slots: Record<string, SpiritSlot[]>,
  entries: Record<string, GuiWindow>,
  sprites: Record<string, GuiSprite>,
): void {
  const offset = view.children?.find(
    (child) =>
      child.type === "positiontype" && child.name === "spirit_row_offset",
  );
  const grow = Number(offset?.position?.y) || 0;
  let shift = 0;
  for (const child of view.children ?? []) {
    if (
      !SPIRIT_SECTIONS.includes(child.name ?? "") &&
      child.name !== "special_forces_header"
    )
      continue;
    if (child.position)
      child.position = {
        ...child.position,
        y: (Number(child.position.y) || 0) + shift,
      };
    if (SPIRIT_SECTIONS.includes(child.name ?? "")) {
      if (child.size)
        child.size = {
          ...child.size,
          height: (Number(child.size.height) || 0) + grow,
        };
      shift += grow;
    }
  }
  for (const [command, groups] of Object.entries(slots)) {
    const holder = view.children?.find(
      (child) => child.name === SPIRIT_VIEWS[command],
    );
    const grid = holder?.children?.find(
      (child) => child.type === "gridboxtype",
    );
    if (!holder || !grid) continue;
    const slot = ((grid.raw ?? {}) as { slotsize?: { width?: number } })
      .slotsize;
    const pitch = Number(slot?.width) || 188;

    const scale = Number(holder.scale) || 1;
    groups.forEach((spirit, at) => {
      const entry = entries[`${spirit.group}_entry`];
      if (!entry) return;
      const node = structuredClone(entry.node) as GuiNode;
      node.children = (node.children ?? []).filter(
        (part) =>
          part.name !== "spirit_icon" && !/_glow$/.test(part.name ?? ""),
      );
      for (const part of node.children) {
        if (spirit.affordable !== false) continue;

        if (part.name === "button") {
          part.sprite = PLATE_DISABLED;
          part.frame = null;
        }
        if (part.name === "icon_background")
          part.frame = (Number(part.frame) || 1) + 1;
      }
      node.position = {
        x:
          (Number(grid.position?.x) || 0) +
          at * pitch +
          (Number(node.position?.x) || 0),
        y: (Number(grid.position?.y) || 0) + (Number(node.position?.y) || 0),
      };
      holder.children = [...(holder.children ?? []), scaledTree(node, scale)];
      Object.assign(sprites, entry.sprites);
    });
  }
}

export function scaledTree(node: GuiNode, scale: number): GuiNode {
  if (scale === 1) return node;
  const times = (value: unknown) =>
    typeof value === "number" ? value * scale : value;
  const walk = (part: GuiNode): GuiNode =>
    ({
      ...part,
      position: part.position
        ? {
            ...part.position,
            x: times(part.position.x),
            y: times(part.position.y),
          }
        : part.position,
      size: part.size
        ? {
            ...part.size,
            width: times(part.size.width),
            height: times(part.size.height),
          }
        : part.size,
      scale: part.sprite ? (Number(part.scale) || 1) * scale : part.scale,
      children: (part.children ?? []).map(walk),
    }) as GuiNode;
  return walk(node);
}

export function officerCorpsTextOverrides(
  win: GuiWindow,
): Record<string, string> {
  const out: Record<string, string> = {};
  if (win.name !== OFFICER_CORPS) return out;
  const put = (path: string, text: string) => {
    for (const node of matchNamePath(win.node, path)) {
      const id = (node as GuiNode & { _id?: string })._id;
      if (id) out[id] = text;
    }
  };
  put("hq_ribbon/deployed_counter", "0/0");
  put("captured_generals_amount", "0");
  return out;
}

export const OFFICER_TOOLTIP_KEYS = [
  "ARMY_HQ_TEMPLATES_TOGGLE_TOOLTIP",
  "CAPTURED_GENERALS_TOOLTIP",
  "CAPTURED_GENERALS_TOOLTIP_CLICK",
  "CAPTURED_GENERALS_TOOLTIP_RIGHT_CLICK",
  "OFFICER_CORP_PREFERRED_TACTICS_SELECTION",
  "OFFICER_CORP_NO_PREFERRED_TACTIC",
];

export function useOfficerTooltipTexts(projectId: number, enabled: boolean) {
  return useSWR<Record<string, string>>(
    enabled ? ["game-view-loc", projectId, "officer-ribbon"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(OFFICER_TOOLTIP_KEYS, { markup: true });
    },
    { revalidateOnFocus: false },
  );
}

export function officerCorpsTooltipFor(
  node: { name?: string | null },
  texts: Record<string, string> | undefined,
) {
  if (!texts) return undefined;
  switch (node.name) {
    case "button_army_hq_templates_toggle":
      return texts.ARMY_HQ_TEMPLATES_TOGGLE_TOOLTIP;
    case "button_captured_generals": {
      const markup = texts.CAPTURED_GENERALS_TOOLTIP;
      if (!markup) return undefined;
      const hints =
        (texts.CAPTURED_GENERALS_TOOLTIP_CLICK ?? "") +
        (texts.CAPTURED_GENERALS_TOOLTIP_RIGHT_CLICK ?? "");
      return fillRuntimeSlots(markup, {
        CLICK_HINTS: hints,
        OWN_COUNT: 0,
        ENEMIES_COUNT: 0,
      });
    }
    case "button_prefered_tactic_selection":
      return (
        (texts.OFFICER_CORP_PREFERRED_TACTICS_SELECTION ?? "") +
        (texts.OFFICER_CORP_NO_PREFERRED_TACTIC ?? "")
      );
    default:
      return undefined;
  }
}

export function centredOn(
  at: { x: number | string; y: number | string } | undefined,
  plate: GuiSprite,
  picture: GuiSprite,
): { x: number; y: number } {
  const width = (sprite: GuiSprite) =>
    sprite.width / Math.max(1, sprite.frames || 1);
  return {
    x: (Number(at?.x) || 0) + Math.round((width(plate) - width(picture)) / 2),
    y: (Number(at?.y) || 0) + Math.round((plate.height - picture.height) / 2),
  };
}

function OfficerEntry({
  entry,
  person,
  picture,
  portrait,
  left,
  top,
  onOpen,
}: {
  entry: GuiWindow;
  person: GuiOfficer | null;

  picture: string;
  portrait: GuiSprite | undefined;
  left: number;
  top: number;

  onOpen?: () => void;
}) {
  const edit = useEditCanvas();
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;
    for (const child of node.children ?? []) {
      if (child.name === "idea_alert_glow" || child.name === "idea_traits")
        child.sprite = null;
      const plate = child.sprite ? entry.sprites[child.sprite] : undefined;
      if (child.name === "button" && portrait && plate) {
        child.position = centredOn(child.position, plate, portrait);
        child.sprite = picture;
      }
    }
    assignIds(node);
    const texts: Record<string, string> = {};
    for (const child of (node.children ?? []) as (GuiNode & {
      _id?: string;
    })[]) {
      if (child._id && child.name === "leader_text") texts[child._id] = "";
    }
    return { node, texts };
  }, [entry.node, entry.sprites, picture, portrait]);

  const printed = useMemo(
    () => personPrinted(built.node, "leader_text", person),
    [built.node, person],
  );
  const sprites = useMemo(
    () =>
      portrait ? { ...entry.sprites, [picture]: portrait } : entry.sprites,
    [entry.sprites, picture, portrait],
  );
  const rects = useMemo(
    () => measureWindow(built.node, sprites, 1920, SCREEN_H),
    [built.node, sprites],
  );
  const textNode = entry.node.children?.find(
    (child) => child.name === "leader_text",
  );
  const font = textNode?.font ? entry.fonts[textNode.font]?.bitmap : undefined;
  const plate = rects["name:button"];

  return (
    <div
      style={{
        position: "absolute",
        left,
        top,
        width: ENTRY_W,
        height: ENTRY_H,
      }}
    >
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${left},${top}`,
          entry,
          built.node,
          sprites,
          built.texts,
          printed,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NOTHING_HIDDEN}
        isolateId={null}
        transparent
        baseW={ENTRY_W}
        baseH={ENTRY_H}
        onActivate={(pressed) => {
          if (pressed.name !== "button" || !onOpen) return false;
          onOpen();
          return true;
        }}
      />

      {person && font && (
        <BitmapText
          font={font}
          text={person.name}

          left={plate ? plate.x + Math.round((plate.w - NAME_W) / 2) : 0}
          top={(plate ? plate.y + plate.h : 90) + 2}
          width={NAME_W}
          height={font.line * 2 + 4}
          align="center"
          vcenter={false}
          maxLines={2}
        />
      )}
    </div>
  );
}

export interface GuiOfficerStat {
  modifier: string;
  icon: string;

  text: string;

  colour: "G" | "R" | "Y";
}

export interface GuiOfficerStats {
  tag: string;
  stats: Record<string, GuiOfficerStat[]>;
  sprites: Record<string, GuiSprite>;
}

function useOfficerStats(
  projectId: number,
  tag: string | null,
  date: string | null,
) {
  return useSWR<GuiOfficerStats>(
    tag ? ["game-view-officer-stats", projectId, tag, date] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/officer-stats?${query}`,
        ),
      );
      return readGuiResponse<GuiOfficerStats>(response);
    },
    { revalidateOnFocus: false },
  );
}

export const STAT_HEADERS: Readonly<Record<string, string>> = {
  army: "army_header",
  navy: "navy_header",
  air: "air_header",
  special_forces: "special_forces_header",
};

export function statPlaces(
  count: number,
  grid:
    | {
        slotsize?: { width?: number | string; height?: number | string };
        max_slots_horizontal?: number | string;
      }
    | undefined,
): { x: number; y: number }[] {
  const w = Number(grid?.slotsize?.width) || 72;
  const h = Number(grid?.slotsize?.height) || 33;
  const across = Number(grid?.max_slots_horizontal) || 7;
  return Array.from({ length: count }, (_, at) => ({
    x: (at % across) * w,
    y: Math.floor(at / across) * h,
  }));
}

function StatEntry({
  entry,
  stat,
  sprite,
  left,
  top,
}: {
  entry: GuiWindow;
  stat: GuiOfficerStat;
  sprite: GuiSprite | undefined;
  left: number;
  top: number;
}) {
  const edit = useEditCanvas();
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;
    for (const child of node.children ?? []) {
      if (child.name === "modifier_icon" && sprite) child.sprite = stat.icon;

      if (child.name === "modifier_value")
        child.text = child.text || "modifier_value";
    }
    assignIds(node);
    const texts: Record<string, string> = {};
    for (const hit of matchNamePath(node, "modifier_value") as (GuiNode & {
      _id?: string;
    })[]) {
      if (hit._id) texts[hit._id] = `§${stat.colour}${stat.text}§!`;
    }
    return { node, texts };
  }, [entry.node, sprite, stat]);
  const sprites = useMemo(
    () => (sprite ? { ...entry.sprites, [stat.icon]: sprite } : entry.sprites),
    [entry.sprites, sprite, stat.icon],
  );
  const size = entry.node.size as { width?: number; height?: number } | null;
  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${stat.modifier}`,
          entry,
          built.node,
          sprites,
          built.texts,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NOTHING_HIDDEN}
        isolateId={null}
        transparent
        baseW={Number(size?.width) || 83}
        baseH={Number(size?.height) || 40}
      />
    </div>
  );
}

export const OFFICER_VIEW = "officer_core_view";
const OFFICER_WHEEL = 40;

export function scrollOfficerCorps(win: GuiWindow, y: number): GuiWindow {
  if (!y) return win;
  return {
    ...win,
    node: withContainerScroll(win.node, (node) => node.name === OFFICER_VIEW, {
      x: 0,
      y,
    }),
  };
}

export function officerScrollBox(
  win: GuiWindow,
  rects: Record<string, MeasuredRectLike>,
  y: number,
  screenH = SCREEN_H,
): {
  box: { x: number; y: number; w: number; h: number };
  limit: number;
} | null {
  const view = findNode(win.node, OFFICER_VIEW) as
    (GuiNode & { _id?: string }) | undefined;
  const rect = view?._id ? rects[view._id] : undefined;
  if (!view || !rect || rect.hidden) return null;
  const margin = (view.raw?.margin ?? {}) as { bottom?: number };
  const bottom =
    screenH -
    (Number(win.node.position?.y) || 0) -
    (Number(margin.bottom) || 0);
  const box = {
    x: rect.x,
    y: rect.y,
    w: rect.w,
    h: Math.max(0, Math.min(rect.y + rect.h, bottom) - rect.y),
  };
  let foot = rect.y;
  for (const child of view.children ?? []) {
    const at = (child as GuiNode & { _id?: string })._id;
    const piece = at ? rects[at] : undefined;
    if (
      child.type === "positiontype" ||
      !piece ||
      piece.hidden ||
      piece.w <= 0 ||
      piece.h <= 0
    )
      continue;
    foot = Math.max(foot, piece.y + piece.h + y);
  }
  return {
    box,
    limit: Math.max(
      0,
      Math.round(foot + (Number(margin.bottom) || 0) - (box.y + box.h)),
    ),
  };
}

type MeasuredRectLike = {
  x: number;
  y: number;
  w: number;
  h: number;
  hidden?: boolean;
};

function inside(root: GuiNode | undefined, name: string): boolean {
  return Boolean(root && findNode(root, name));
}

export function OfficerCorpsLayer({
  win,
  projectId,
  tag,
  date,
  onOpenSeat,
  scroll = 0,
  onScroll,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  onOpenSeat?: (slot: string) => void;

  scroll?: number;
  onScroll?: ScrollUpdate;
}) {
  const active = win.name === OFFICER_CORPS;
  const corps = useOfficerCorps(projectId, active ? tag : null, date ?? null);
  const highCommand = useTemplateWindow(
    projectId,
    active ? "high_command_entry" : null,
  );
  const theorist = useTemplateWindow(
    projectId,
    active ? "theorist_entry" : null,
  );
  const army = useTemplateWindow(projectId, active ? "army_chief_entry" : null);
  const navy = useTemplateWindow(projectId, active ? "navy_chief_entry" : null);
  const airForce = useTemplateWindow(
    projectId,
    active ? "air_chief_entry" : null,
  );
  const stats = useOfficerStats(projectId, active ? tag : null, date ?? null);
  const statEntry = useTemplateWindow(
    projectId,
    active ? "modifier_entry" : null,
  );
  const windows: Record<string, GuiWindow | undefined> = {
    high_command_entry: highCommand.data,
    theorist_entry: theorist.data,
    army_chief_entry: army.data,
    navy_chief_entry: navy.data,
    air_chief_entry: airForce.data,
  };

  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, SCREEN_H) : {}),
    [active, win],
  );
  const seats = useMemo(
    () =>
      corps.data
        ? OFFICER_GRIDS.map((grid) => ({
            ...grid,
            seats: officerSeats(corps.data as GuiOfficerCorps, grid.slot),
          }))
        : [],
    [corps.data],
  );
  const portraitNames = useMemo(() => {
    const names = new Set<string>();
    for (const grid of seats) {
      for (const person of grid.seats)
        names.add(person?.portrait ?? emptySlotSprite(grid.slot));
    }
    return [...names].sort();
  }, [seats]);
  const portraits = useSWR<Record<string, GuiSprite>>(
    portraitNames.length
      ? [
          "game-view-sprites",
          projectId,
          "officer-portraits",
          portraitNames.join(","),
        ]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(portraitNames);
    },
    { revalidateOnFocus: false },
  );

  if (!active || !corps.data) return null;
  const page = officerScrollBox(win, rects, scroll);
  const view = findNode(win.node, OFFICER_VIEW);

  const onPage = (node: ReactNode) =>
    page ? (
      <div
        style={{
          position: "absolute",
          left: page.box.x,
          top: page.box.y,
          width: page.box.w,
          height: page.box.h,
          overflow: "hidden",
          pointerEvents: "none",
        }}
      >
        <div
          style={{ position: "absolute", left: -page.box.x, top: -page.box.y }}
        >
          {node}
        </div>
      </div>
    ) : (
      node
    );
  const seatEntries = (pageOnly: boolean) =>
    seats.flatMap(({ slot, grid, entry, seats: row }) => {
      if (inside(view, grid) !== pageOnly) return [];
      const box = rects[`name:${grid}`];
      const window = windows[entry];
      if (!box || box.hidden || !window) return [];
      const gridNode = findNode(win.node, grid);
      const raw = gridNode?.raw as
        { slotsize?: { width?: number } } | undefined;
      const pitch = Number(raw?.slotsize?.width) || 70;

      const at = window.node.position ?? { x: 0, y: 0 };
      return row.map((person, index) => (
        <OfficerEntry
          key={`${slot}-${index}`}
          entry={window}
          person={person}
          picture={person?.portrait ?? emptySlotSprite(slot)}
          portrait={portraits.data?.[person?.portrait ?? emptySlotSprite(slot)]}
          left={box.x + index * pitch + (Number(at.x) || 0)}
          top={box.y + (Number(at.y) || 0)}
          onOpen={onOpenSeat ? () => onOpenSeat(slot) : undefined}
        />
      ));
    });

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {seatEntries(false)}
      {onPage(seatEntries(true))}
      {onPage(
        statEntry.data &&
          Object.entries(STAT_HEADERS).flatMap(([branch, header]) => {
            const grid = matchNamePath(
              win.node,
              `${header}/modifiers_grid`,
            )[0] as (GuiNode & { _id?: string }) | undefined;
            const box = grid?._id ? rects[grid._id] : undefined;
            const items = stats.data?.stats?.[branch] ?? [];
            if (!grid || !box || box.hidden) return [];
            const places = statPlaces(
              items.length,
              grid.raw as Parameters<typeof statPlaces>[1],
            );
            return items.map((stat, at) => (
              <StatEntry
                key={`${branch}-${stat.modifier}`}
                entry={statEntry.data as GuiWindow}
                stat={stat}
                sprite={stats.data?.sprites?.[stat.icon]}
                left={box.x + places[at].x}
                top={box.y + places[at].y}
              />
            ));
          }),
      )}
      {page && onScroll && page.limit > 0 && (
        <>
          <PageScroll
            box={page.box}
            scroll={{ x: 0, y: scroll }}
            limits={{ x: 0, y: page.limit }}
            step={OFFICER_WHEEL}
            dragLeft={false}
            onScroll={onScroll}
          />
          <PageScrollbar
            box={page.box}
            scroll={scroll}
            limit={page.limit}
            onScroll={(y) =>
              onScroll(() => ({
                x: 0,
                y: Math.round(Math.min(page.limit, Math.max(0, y))),
              }))
            }
          />
        </>
      )}
    </div>
  );
}

function findNode(root: GuiNode, name: string): GuiNode | undefined {
  if (root.name === name) return root;
  for (const child of root.children ?? []) {
    const found = findNode(child, name);
    if (found) return found;
  }
  return undefined;
}
