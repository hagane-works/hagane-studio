"use client";

import { useMemo } from "react";
import useSWR from "swr";
import { readGuiResponse } from "./guiResponse";
import {
  guiRenderApi,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
  withGameLanguage,
} from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { applyMargins } from "./FactionLayer";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";
import { pickable } from "./spriteFrame";

export const CONSTRUCTION = "countryconstructionsview";

const STRIP = "GFX_buildings_strip";

const CONVERT = "GFX_convert_ic_button";
const CONVERT_GROUP = "shared";

const SLOT_W = 46;
const SLOT_H = 44;
const COLUMNS = 4;

const GRIDS: readonly { section: string; grid: string; group: string }[] = [
  {
    section: "possible_state_constructions",
    grid: "non_shared_item_grid",
    group: "state",
  },
  {
    section: "possible_shared_constructions",
    grid: "shared_item_grid",
    group: "shared",
  },
  {
    section: "possible_province_constructions",
    grid: "province_item_grid",
    group: "province",
  },
  {
    section: "possible_facility_constructions",
    grid: "facility_item_grid",
    group: "facility",
  },
];

const SECTION_HEAD = 52;

const ROW = { w: 400, h: 52 };
const ROW_TEMPLATES: Record<string, string> = {
  consumer_goods: "static_consumer_goods_entry",
  traded: "static_traded_factories_entry",
  national_projects: "static_project_factories_entry",
  contracts: "static_contract_payment_entry",
};

const STEP_BUTTONS = [
  "factories_sub_5x",
  "factories_sub_10x",
  "factories_add_5x",
  "factories_add_10x",
];

export interface GuiConstruction {
  tag: string;
  civilian: number;
  used: number;
  from_trade: number;
  speed: number;
  rows: { kind: string; factories: number }[];

  energy?: { resource: string; frame: number; amount: number };
}

const ENERGY_CHIP = "resource_strip_item";
const ENERGY_SIZE = { w: 70, h: 30 };

export function energyText(amount: number): string {
  if (amount > 0) return `§G${amount}§!`;
  if (amount < 0) return `§R${amount}§!`;
  return "0";
}

export interface GuiBuilding {
  name: string;
  label?: string;
  icon?: string;
  icon_frame: number;
  base_cost: number;
  max_level: number;
  is_province: boolean;
  group?: string;
  built: number;
}

export function useBuildings(
  projectId: number,
  tag: string | null,
  date: string | null,
) {
  return useSWR<{ tag: string; buildings: GuiBuilding[] }>(
    tag ? ["game-view-buildings", projectId, tag, date] : null,
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/buildings?tag=${encodeURIComponent(tag as string)}`,
        ) + (date ? `&date=${encodeURIComponent(date)}` : ""),
      );
      return readGuiResponse<{ tag: string; buildings: GuiBuilding[] }>(
        response,
      );
    },
    { revalidateOnFocus: false },
  );
}

export function useConstruction(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useSWR<GuiConstruction>(
    tag ? ["game-view-construction", projectId, tag, date ?? ""] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/construction?${query}`,
        ),
      );
      return readGuiResponse<GuiConstruction>(response);
    },
    { revalidateOnFocus: false },
  );
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

function groupOf(building: GuiBuilding): string {
  return building.group || (building.is_province ? "province" : "state");
}

export function speedText(factor: number): string {
  const percent = Math.round(factor * 100);
  if (percent > 0) return `§G+${percent}%§!`;
  if (percent < 0) return `§R${percent}%§!`;
  return "0%";
}

export function constructionWindow(
  win: GuiWindow,
  data: GuiConstruction | undefined,
  buildings: GuiBuilding[] | undefined,
): { win: GuiWindow; hidden: string[]; texts: Record<string, string> } {
  const node = structuredClone(win.node) as GuiNode;
  applyMargins(node);
  const empty: string[] = [];
  if (buildings) {
    let top: number | null = null;
    for (const { section, group } of GRIDS) {
      const container = matchNamePath(
        node,
        `possible_constructions/${section}`,
      )[0];
      if (!container) continue;
      if (top === null) top = Number(container.position?.y) || 0;
      const count =
        buildings.filter((building) => groupOf(building) === group).length +
        (group === CONVERT_GROUP ? 1 : 0);
      if (!count) {
        empty.push(section);
        continue;
      }
      container.position = { ...container.position, y: top };
      top += SECTION_HEAD + Math.ceil(count / COLUMNS) * SLOT_H;
    }
  }
  assignIds(node);
  const hidden: string[] = [];
  const hide = (path: string) => {
    for (const hit of matchNamePath(node, path))
      if (idOf(hit)) hidden.push(idOf(hit));
  };
  for (const section of empty) hide(`possible_constructions/${section}`);
  for (const button of STEP_BUTTONS) hide(`count_container/${button}`);

  hide("countryconstructionsview/expand");
  hide("production_win_top/cic_bank_icon");
  hide("production_win_top/cic_bank_value");
  const texts: Record<string, string> = {};
  const write = (path: string, text: string) => {
    for (const hit of matchNamePath(node, path))
      if (idOf(hit)) texts[idOf(hit)] = text;
  };
  write("count_container/factories_used_txt", "0");
  if (data) {
    write(
      "production_win_top/available_factories_value",
      `§Y${data.used}/${data.civilian + data.from_trade}§!`,
    );
    write("production_win_top/traded_factories_value", String(data.from_trade));
    write("production_win_top/local_factories_value", String(data.civilian));
    write("speed_container/construction_speed_value", speedText(data.speed));
  }
  return { win: { ...win, node }, hidden, texts };
}

function frameStyle(
  sprite: GuiSprite,
  frame: number,
  size: number,
): React.CSSProperties {
  const frames = Math.max(1, sprite.frames || 1);
  const index = Math.max(0, Math.min(frames - 1, (frame || 1) - 1));
  return {
    position: "absolute",
    left: 0,
    top: 0,
    width: size,
    height: size,
    backgroundImage: sprite.vanilla_url
      ? `url("${sprite.vanilla_url}")`
      : undefined,
    backgroundPosition: `-${index * size}px 0`,
    backgroundRepeat: "no-repeat",
    backgroundSize: `${size * frames}px ${size}px`,
  };
}

export function ConstructionLayer({
  win,
  projectId,
  tag,
  date,
  onUnavailable,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  onUnavailable?: (control: string) => void;
}) {
  const active = win.name === CONSTRUCTION;
  const buildings = useBuildings(projectId, active ? tag : null, date ?? null);
  const construction = useConstruction(projectId, active ? tag : null, date);
  const consumerRow = useTemplateWindow(
    projectId,
    active ? ROW_TEMPLATES.consumer_goods : null,
  );
  const projectRow = useTemplateWindow(
    projectId,
    active ? ROW_TEMPLATES.national_projects : null,
  );
  const tradedRow = useTemplateWindow(
    projectId,
    active ? ROW_TEMPLATES.traded : null,
  );
  const contractRow = useTemplateWindow(
    projectId,
    active ? ROW_TEMPLATES.contracts : null,
  );
  const energyChip = useTemplateWindow(projectId, active ? ENERGY_CHIP : null);
  const strip = useSWR<Record<string, GuiSprite>>(
    active ? ["game-view-sprites", projectId, "buildings-strip"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites([STRIP, CONVERT]);
    },
    { revalidateOnFocus: false },
  );
  const laid = useMemo(() => {
    if (!active) return null;
    const { win: shown } = constructionWindow(
      win,
      construction.data,
      buildings.data?.buildings,
    );
    const rects = measureWindow(shown.node, shown.sprites, 1920, 1080);
    const rect = (path: string) =>
      rects[idOf(matchNamePath(shown.node, path)[0])] as
        { x: number; y: number } | undefined;
    return {
      grids: Object.fromEntries(
        GRIDS.map(({ grid }) => [grid, rect(`possible_constructions/${grid}`)]),
      ),
      list: rect("production_lines/item_grid"),
      energy: rect("resource_strip_container/resources_grid"),
    };
  }, [active, win, construction.data, buildings.data]);

  const rows = buildings.data?.buildings;
  if (!active || !rows || !laid) return null;
  const sprite = strip.data?.[STRIP];
  const convert = strip.data?.[CONVERT];
  const templates: Record<string, GuiWindow | undefined> = {
    consumer_goods: consumerRow.data,
    national_projects: projectRow.data,
    traded: tradedRow.data,
    contracts: contractRow.data,
  };
  const list = laid.list;
  const energy = construction.data?.energy;

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {energyChip.data && energy && laid.energy && (
        <Filled
          entry={energyChip.data}
          left={laid.energy.x}
          top={laid.energy.y}
          width={ENERGY_SIZE.w}
          height={ENERGY_SIZE.h}
          texts={{ value: energyText(energy.amount) }}
          frames={{ icon: energy.frame }}
          hide={["energy_ratio_anim"]}
          extraSprites={{}}
        />
      )}
      {list &&
        (construction.data?.rows ?? []).map((row, k) => {
          const entry = templates[row.kind];
          if (!entry) return null;
          return (
            <div
              key={row.kind}
              aria-label={`construction ${row.kind}`}
              style={{
                position: "absolute",
                left: list.x,
                top: list.y + k * ROW.h,
                width: ROW.w,
                height: ROW.h,
              }}
            >
              <Filled
                entry={entry}
                left={0}
                top={0}
                width={ROW.w}
                height={ROW.h}
                texts={{ factories: String(row.factories) }}
                extraSprites={{}}
              />
            </div>
          );
        })}
      {GRIDS.flatMap(({ grid, group }) => {
        const rect = laid.grids[grid];
        if (!rect) return [];
        const mine = rows.filter((building) => groupOf(building) === group);
        const slot = (at: number): React.CSSProperties => ({
          position: "absolute",
          left: rect.x + (at % COLUMNS) * SLOT_W,
          top: rect.y + Math.floor(at / COLUMNS) * SLOT_H,
          width: SLOT_W - 2,
          height: SLOT_H - 2,
        });
        const converter =
          group === CONVERT_GROUP && convert?.vanilla_url ? (
            <button
              key={`${grid}-convert`}
              type="button"
              aria-label="convert factories"
              onClick={() => onUnavailable?.("convert factories")}
              style={{
                ...slot(mine.length),
                padding: 0,
                border: 0,
                background: "transparent",
                pointerEvents: "auto",
              }}
            >
              <div
                style={frameStyle(convert, 1, SLOT_W - 4)}
                {...pickable(CONVERT, 1, "convert factories")}
              />
            </button>
          ) : null;
        return [
          ...mine.map((building, at) => (
            <button
              key={`${grid}-${building.name}`}
              type="button"
              aria-label={`building ${building.name}`}
              title={building.label || building.name}
              style={{
                ...slot(at),
                padding: 0,
                border: 0,
                background: "transparent",
                pointerEvents: "auto",
              }}
            >
              {sprite && (
                <div
                  style={frameStyle(sprite, building.icon_frame, SLOT_W - 4)}
                  {...pickable(
                    STRIP,
                    building.icon_frame || 1,
                    building.label || building.name,
                  )}
                />
              )}
            </button>
          )),
          converter,
        ];
      })}
    </div>
  );
}
