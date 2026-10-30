"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { guiArray, guiRecord, readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { insetByMargin } from "../gui-renderer/windowMargins";
import { useTemplateWindow } from "./CountrySelectLayer";
import { FieldHit } from "./FieldHit";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";
import { SpriteBar, type BarSprite } from "./SpriteBar";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export const ARMY_OVERVIEW = "countryarmyview";

export type OverviewMode = "army" | "navy" | "air";

export function overviewMode(openedBy: string | null): OverviewMode {
  if (openedBy === "navy_button") return "navy";
  if (openedBy === "air_button") return "air";
  return "army";
}

const SECTIONS: Record<OverviewMode, readonly string[]> = {
  army: ["army_header", "army_units"],
  navy: ["navy_header", "navy_units"],
  air: ["air_header", "air_units"],
};

const NAVY_BUTTONS = ["naval_hq_open", "destroyed_ships_open"];

const BANNER_FRAME: Record<OverviewMode, number> = { army: 1, navy: 2, air: 3 };
const ROW_SIZE = { w: 511, h: 35 };

const PILLS: Record<
  OverviewMode,
  {
    entry: string;
    grid: string;
    slot: { w: number; h: number };
    size: { w: number; h: number };
  }
> = {
  army: {
    entry: "army_view_subunit_entry",
    grid: "army_header/sub_units_grid",
    slot: { w: 86, h: 29 },
    size: { w: 83, h: 40 },
  },
  navy: {
    entry: "navy_view_ship_entry",
    grid: "navy_header/ships_count_grid",
    slot: { w: 86, h: 27 },
    size: { w: 86, h: 27 },
  },
  air: {
    entry: "air_view_plane_entry",
    grid: "air_header/count_grid",
    slot: { w: 86, h: 27 },
    size: { w: 60, h: 35 },
  },
};
const PILLS_PER_ROW = 6;
const ROWS: Record<OverviewMode, string> = {
  army: "armyview_army_entry",
  navy: "armyview_navy_entry",
  air: "armyview_air_entry",
};

const BAR = { w: 28, h: 6 };
const BARS_X: Record<
  "army" | "navy",
  { organisation: number; strength: number }
> = {
  army: { organisation: 85, strength: 93 },
  navy: { organisation: 125, strength: 133 },
};

const AIR_BAR_X = 94;
const BARS_BOTTOM = 25;

const ROW_HIDDEN: Record<"army" | "navy", string[]> = {
  army: [
    "group_marker",
    "battle_icon",
    "obsolete_template_warning",
    "expeditionary_flag",
    "colonial_force_flag_frame",
    "exile_force_flag_frame",
    "foreign_template_flag",
    "organization_progressbar",
    "strength_progressbar",
  ],
  navy: ["battle_icon", "organization_progressbar", "strength_progressbar"],
};

const AIR_ROW_HIDDEN = [
  "bar_filled",
  "carrier_capable_icon",
  "on_carrier_icon",
  "ace",
  "mission_icon_1",
  "mission_icon_2",
  "mission_icon_3",
  "mission_icon_4",
  "exile_force_flag_frame",
  "foreign_wing_flag",
];

export interface GuiArmyDivision {
  name: string;
  template: string;
  icon: string;
  level: number;
  strength: number;
  organisation: number;

  oob: string;
  ordinal: number;
  experience: number;
  equipment: number;

  location?: number;

  officer?: string;
}

export interface GuiArmyOverview {
  tag: string;
  divisions: GuiArmyDivision[];
  pills: { icon: string; count: number }[];

  templates: string[];
  special_forces_cap: number;
  texts: {
    title: string;
    divisions: string;
    battalions: string;
    theatres: string;
    special_cap: string;
  };
  sprites: Record<string, GuiSprite>;
}

export interface GuiTaskForce {
  name: string;
  fleet: string;
  region: string;
  size: number;
  icon: string;
  class: string;
  pride: boolean;
  strength: number;
  organisation: number;

  oob: string;
  ordinal: number;
  raw_name: string;
}

export interface GuiAirWing {
  name: string;
  design: string;
  type: string;
  role: string;
  icon: string;
  count: number;
  limit: number;
  base: string;
  state: number;

  oob: string;
  ordinal: number;
  state_index: number;
  raw_name: string;
}

export interface GuiAirOverview {
  tag: string;
  wings: GuiAirWing[];
  pills: { role: string; frame: number; count: number }[];
  texts: { title: string; wings: string; theatres: string };
  sprites: Record<string, GuiSprite>;
}

export interface GuiNavyOverview {
  tag: string;
  task_forces: GuiTaskForce[];
  pills: { class: string; icon: string; count: number }[];
  texts: { title: string; fleets: string; theatres: string };
  sprites: Record<string, GuiSprite>;
}

function useOverview<T>(
  kind: string,
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useSWR<T>(
    tag ? [`game-view-${kind}`, projectId, tag, date ?? ""] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/${kind}?${query}`,
        ),
      );
      return readGuiResponse<T>(response);
    },
    { revalidateOnFocus: false },
  );
}

export function useArmyOverview(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useOverview<GuiArmyOverview>("army-overview", projectId, tag, date);
}

export function useNavyOverview(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useOverview<GuiNavyOverview>("navy-overview", projectId, tag, date);
}

export function useAirOverview(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useOverview<GuiAirOverview>("air-overview", projectId, tag, date);
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

function overviewNode(win: GuiWindow, mode: OverviewMode): GuiNode {
  const node = structuredClone(win.node) as GuiNode;
  const others = (Object.keys(SECTIONS) as OverviewMode[])
    .filter((other) => other !== mode)
    .flatMap((other) => SECTIONS[other]);
  const gone = new Set([...others, ...(mode === "navy" ? [] : NAVY_BUTTONS)]);
  node.children = (node.children ?? [])
    .filter((child) => !gone.has(child.name ?? ""))
    .map((child) =>
      child.name === "info" || SECTIONS[mode].includes(child.name ?? "")
        ? insetByMargin(child)
        : child,
    );
  for (const banner of matchNamePath(node, "army_overview_top"))
    banner.frame = BANNER_FRAME[mode];
  return node;
}

export function armyOverviewWindow(
  win: GuiWindow,
  mode: OverviewMode,
  army: GuiArmyOverview | undefined,
  navy?: GuiNavyOverview | undefined,
  air?: GuiAirOverview | undefined,
): { win: GuiWindow; hidden: string[]; texts: Record<string, string> } {
  const node = overviewNode(win, mode);
  assignIds(node);
  const hidden: string[] = [];
  const hide = (path: string) => {
    for (const hit of matchNamePath(node, path))
      if (idOf(hit)) hidden.push(idOf(hit));
  };

  hide(`${SECTIONS[mode][0]}/modifiers_grid`);

  if (mode !== "army")
    for (const part of [
      "battalion_count",
      "special_forces_icon",
      "special_forces_cap",
    ])
      hide(`info/${part}`);
  const texts: Record<string, string> = {};
  const write = (path: string, text: string | undefined) => {
    if (text === undefined) return;
    for (const hit of matchNamePath(node, path))
      if (idOf(hit)) texts[idOf(hit)] = text;
  };
  if (mode === "army" && army) {
    write("army_title", army.texts.title);
    write("info/unit_count", army.texts.divisions);
    write("info/battalion_count", army.texts.battalions);
    write("info/special_forces_cap", army.texts.special_cap);
    write("theatres_count", army.texts.theatres);
  }
  if (mode === "navy" && navy) {
    write("army_title", navy.texts.title);
    write("info/unit_count", navy.texts.fleets);
    write("theatres_count", navy.texts.theatres);
  }
  if (mode === "air" && air) {
    write("army_title", air.texts.title);
    write("info/unit_count", air.texts.wings);
    write("theatres_count", air.texts.theatres);
  }
  return { win: { ...win, node }, hidden, texts };
}

type Rect = { x: number; y: number; w: number; h: number };

function UprightBar({
  x,
  sprites,
  fraction,
  sprite = "GFX_navieslist_bar_str",
}: {
  x: number;
  sprites: Record<string, GuiSprite>;
  fraction: number;
  sprite?: string;
}) {
  return (
    <div
      style={{
        position: "absolute",
        left: x,
        top: BARS_BOTTOM - BAR.w,
        width: BAR.h,
        height: BAR.w,
      }}
    >
      <div
        style={{
          position: "absolute",
          left: 0,
          top: BAR.w,
          width: BAR.w,
          height: BAR.h,
          transform: "rotate(-90deg)",
          transformOrigin: "0 0",
        }}
      >
        <SpriteBar
          x={0}
          y={0}
          size={BAR}
          sprite={sprites[sprite] as BarSprite | undefined}
          fraction={fraction}
        />
      </div>
    </div>
  );
}

function UprightBars({
  x,
  sprites,
  organisation,
  strength,
}: {
  x: { organisation: number; strength: number };
  sprites: Record<string, GuiSprite>;
  organisation: number;
  strength: number;
}) {
  return (
    <>
      <UprightBar
        x={x.organisation}
        sprites={sprites}
        fraction={organisation}
        sprite="GFX_navieslist_bar_org"
      />
      <UprightBar x={x.strength} sprites={sprites} fraction={strength} />
    </>
  );
}

export function ArmyOverviewLayer({
  win,
  projectId,
  tag,
  date,
  mode,
  onEditDivision,
  onEditUnit,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  mode: OverviewMode;
  editMode?: boolean;

  onEditDivision?: (division: GuiArmyDivision) => void;

  onEditUnit?: (
    kind: "task_force" | "air_wing",
    oob: string,
    ordinal: number,
  ) => void;
}) {
  const t = useTranslations("gameViewMilitary.overview");
  const active = win.name === ARMY_OVERVIEW;
  const section = mode;
  const [pillFilter, setPillFilter] = useState<string | null>(null);
  const army = useArmyOverview(
    projectId,
    active && section === "army" ? tag : null,
    date,
  );
  const navy = useNavyOverview(
    projectId,
    active && section === "navy" ? tag : null,
    date,
  );
  const air = useAirOverview(
    projectId,
    active && section === "air" ? tag : null,
    date,
  );
  const pill = useTemplateWindow(
    projectId,
    active ? PILLS[section].entry : null,
  );
  const row = useTemplateWindow(projectId, active ? ROWS[section] : null);
  const at = useMemo(() => {
    const out: Record<string, Rect | undefined> = {};
    if (!active) return out;
    const node = overviewNode(win, mode);
    const measured = measureWindow(node, win.sprites, 1920, 1080);
    for (const path of [
      PILLS[section].grid,
      `${section}_units/item_grid`,
      `${section}_units`,
    ]) {
      out[path] = measured[idOf(matchNamePath(node, path)[0])] as
        Rect | undefined;
    }
    return out;
  }, [active, win, mode, section]);
  if (!active || !pill.data || !row.data) return null;
  const pills = at[PILLS[section].grid];
  const grid = at[`${section}_units/item_grid`];
  const box = at[`${section}_units`];
  const data =
    section === "army" ? army.data : section === "navy" ? navy.data : air.data;
  if (!data) return null;
  const sprites = guiRecord(data.sprites);
  const allDivisions = guiArray(army.data?.divisions);
  const allTaskForces = guiArray(navy.data?.task_forces);
  const allWings = guiArray(air.data?.wings);
  const pillItems: {
    key: string;
    icon?: string;
    frame?: number;
    count: number;
  }[] =
    section === "army"
      ? guiArray(army.data?.pills).map((item) => ({ ...item, key: item.icon }))
      : section === "navy"
        ? guiArray(navy.data?.pills).map((item) => ({
            ...item,
            key: item.class,
          }))
        : guiArray(air.data?.pills).map((item) => ({
            ...item,
            key: item.role,
          }));
  const effectiveFilter = pillItems.some((item) => item.key === pillFilter)
    ? pillFilter
    : null;
  const divisions = effectiveFilter
    ? allDivisions.filter((division) => division.icon === effectiveFilter)
    : allDivisions;
  const taskForces = effectiveFilter
    ? allTaskForces.filter((force) => force.class === effectiveFilter)
    : allTaskForces;
  const wings = effectiveFilter
    ? allWings.filter((wing) => wing.role === effectiveFilter)
    : allWings;
  const rowCount =
    section === "army"
      ? divisions.length
      : section === "navy"
        ? taskForces.length
        : wings.length;
  const pillTexts = (count: number): Record<string, string> =>
    section === "army"
      ? { sub_units_count: String(count) }
      : section === "navy"
        ? { ships_count: String(count) }
        : { count: String(count) };
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {pills &&
        pillItems.map((item, index) => {
          const left =
            pills.x + (index % PILLS_PER_ROW) * PILLS[section].slot.w;
          const top =
            pills.y + Math.floor(index / PILLS_PER_ROW) * PILLS[section].slot.h;
          return (
            <div key={item.key || index}>
              <Filled
                entry={pill.data as GuiWindow}
                left={left}
                top={top}
                width={PILLS[section].size.w}
                height={PILLS[section].size.h}
                texts={pillTexts(item.count)}
                pictures={
                  section === "army"
                    ? { sub_unit_icon: item.icon }
                    : section === "navy"
                      ? { ship_icon: item.icon }
                      : {}
                }

                frames={
                  section === "air" && item.frame
                    ? { icon: item.frame }
                    : undefined
                }
                hide={
                  effectiveFilter === item.key
                    ? ["filter_button"]
                    : ["highlight", "filter_button"]
                }
                extraSprites={sprites}
              />
              <button
                type="button"
                aria-label={["filter", section, item.key].join(" ")}
                aria-pressed={effectiveFilter === item.key}
                onClick={() =>
                  setPillFilter((current) =>
                    current === item.key ? null : item.key,
                  )
                }
                style={{
                  position: "absolute",
                  left,
                  top,
                  width: PILLS[section].size.w,
                  height: PILLS[section].size.h,
                  padding: 0,
                  border: 0,
                  background: "transparent",
                  pointerEvents: "auto",
                }}
              />
            </div>
          );
        })}
      {grid && box && (
        <div
          style={{
            position: "absolute",
            left: grid.x,
            top: grid.y,
            width: ROW_SIZE.w + 4,
            height: Math.max(0, box.y + box.h - grid.y - 40),
            overflowY: "auto",
            overflowX: "hidden",
            ...GAME_SCROLLBAR,
            pointerEvents: "auto",
          }}
        >
          <div style={{ position: "relative", height: rowCount * ROW_SIZE.h }}>
            {section === "army" &&
              divisions.map((division, index) => (
                <div
                  key={`${division.name}#${index}`}
                  style={{
                    position: "absolute",
                    left: 0,
                    top: index * ROW_SIZE.h,
                    width: ROW_SIZE.w,
                    height: ROW_SIZE.h,
                  }}
                >
                  <Filled
                    entry={row.data as GuiWindow}
                    left={0}
                    top={0}
                    width={ROW_SIZE.w}
                    height={ROW_SIZE.h}
                    texts={{ name: division.name, theatre: "", group: "" }}
                    pictures={division.icon ? { icon: division.icon } : {}}
                    frames={{ level: division.level + 1 }}
                    hide={ROW_HIDDEN.army}
                    extraSprites={sprites}
                  />
                  <UprightBars
                    x={BARS_X.army}
                    sprites={sprites}
                    organisation={division.organisation}
                    strength={division.strength}
                  />
                  {onEditDivision && (
                    <FieldHit
                      x={0}
                      y={0}
                      w={ROW_SIZE.w}
                      h={ROW_SIZE.h}
                      label={t("division", { name: division.name })}
                      onClick={() => onEditDivision(division)}
                    />
                  )}
                </div>
              ))}
            {section === "navy" &&
              taskForces.map((force, index) => (
                <div
                  key={`${force.name}#${index}`}
                  style={{
                    position: "absolute",
                    left: 0,
                    top: index * ROW_SIZE.h,
                    width: ROW_SIZE.w,
                    height: ROW_SIZE.h,
                  }}
                >
                  <Filled
                    entry={row.data as GuiWindow}
                    left={0}
                    top={0}
                    width={ROW_SIZE.w}
                    height={ROW_SIZE.h}
                    texts={{
                      unit_name: force.name,
                      region: force.region,
                      unit_count: String(force.size),
                    }}
                    pictures={{ icon: force.icon }}

                    frames={{ icon: force.pride ? 2 : 1 }}
                    hide={
                      force.pride
                        ? ROW_HIDDEN.navy
                        : [...ROW_HIDDEN.navy, "pride"]
                    }
                    extraSprites={sprites}
                  />
                  <UprightBars
                    x={BARS_X.navy}
                    sprites={sprites}
                    organisation={force.organisation}
                    strength={force.strength}
                  />
                  {onEditUnit && (
                    <FieldHit
                      x={0}
                      y={0}
                      w={ROW_SIZE.w}
                      h={ROW_SIZE.h}
                      label={t("taskForce", { name: force.name })}
                      onClick={() =>
                        onEditUnit("task_force", force.oob, force.ordinal)
                      }
                    />
                  )}
                </div>
              ))}
            {section === "air" &&
              wings.map((wing, index) => (
                <div
                  key={`${wing.name}#${index}`}
                  style={{
                    position: "absolute",
                    left: 0,
                    top: index * ROW_SIZE.h,
                    width: ROW_SIZE.w,
                    height: ROW_SIZE.h,
                  }}
                >
                  <Filled
                    entry={row.data as GuiWindow}
                    left={0}
                    top={0}
                    width={ROW_SIZE.w}
                    height={ROW_SIZE.h}
                    texts={{
                      unit_name: wing.name,
                      base: wing.base,
                      count: String(wing.count),
                      limit: String(wing.limit),
                      assigned_region: "",
                    }}
                    pictures={wing.icon ? { icon: wing.icon } : {}}
                    hide={AIR_ROW_HIDDEN}
                    extraSprites={sprites}
                  />
                  <UprightBar x={AIR_BAR_X} sprites={sprites} fraction={1} />
                  {onEditUnit && (
                    <FieldHit
                      x={0}
                      y={0}
                      w={ROW_SIZE.w}
                      h={ROW_SIZE.h}
                      label={t("airWing", { name: wing.name })}
                      onClick={() =>
                        onEditUnit("air_wing", wing.oob, wing.ordinal)
                      }
                    />
                  )}
                </div>
              ))}
          </div>
        </div>
      )}
    </div>
  );
}
