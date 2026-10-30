"use client";

import { useMemo, type ReactNode } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";

export interface GuiQueueLine {
  priority: number;
  naval: boolean;
  equipment: string;
  factories: number;
  requested: number;
  name: string;
  general_name: string;
  icon?: string;
  group?: string;
  resources?: Record<string, number>;

  speed?: string;
  daily?: number;

  stock?: number;

  amount?: number;
}

export interface GuiProductionQueue {
  lines: GuiQueueLine[];
  military_used: number;
  military_total: number;
  naval_used: number;
  naval_total: number;
  categories: Record<string, number>;
  modifiers: Record<string, number>;
}

const MILITARY_LINE = "production_military_line_entry";
const NAVAL_LINE = "production_naval_line_entry";
const MILITARY_FACTORY = "production_military_factory_item";
const NAVAL_FACTORY = "production_naval_factory_item";

export const LINE_H = 111;
const LINE_W = 513;

export const QUEUE_MARGIN = { top: 36, bottom: 13 };

const FACTORY = { w: 29, h: 23, across: 5 };
const RESOURCE = { w: 40, h: 24 };

const CATEGORY_TEXTS: readonly { text: string; group: string }[] = [
  { text: "factories_used_equipment", group: "infantry" },
  { text: "factories_used_armor", group: "armor" },
  { text: "factories_used_air", group: "air" },
  { text: "factories_used_capital_ships", group: "capital_ships" },
  { text: "factories_used_screen_ships", group: "screen_ships" },
  { text: "factories_used_other_ships", group: "other_ships" },
];

export function usedText(used: number, total: number): string {
  return total > 0 && used >= total
    ? `§G${used}/${total}§!`
    : `§Y${used}/${total}§!`;
}

export function percentText(value: number, signed = true): string {
  const percent = Math.round(value * 100);
  if (!signed) return `${percent}%`;
  if (percent > 0) return `§G+${percent}%§!`;
  if (percent < 0) return `§R${percent}%§!`;
  return "+0%";
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function productionTexts(
  win: GuiWindow,
  queue: GuiProductionQueue | undefined,
): Record<string, string> {
  const texts: Record<string, string> = {};
  if (!queue) return texts;
  const write = (name: string, text: string) => {
    for (const hit of matchNamePath(win.node, name))
      if (idOf(hit)) texts[idOf(hit)] = text;
  };
  write(
    "operational_factories_value",
    usedText(queue.military_used, queue.military_total),
  );
  write(
    "operational_naval_factories_value",
    usedText(queue.naval_used, queue.naval_total),
  );
  const mods = queue.modifiers ?? {};
  write("production_output_value", percentText(mods.production_output ?? 0));
  write("docyards_output_value", percentText(mods.dockyards_output ?? 0));
  write("efficiency_cap_value", percentText(mods.efficiency_cap ?? 0.5, false));
  write("efficiency_growth_value", percentText(mods.efficiency_growth ?? 0));
  write(
    "production_efficiency_loss_value",
    percentText(mods.production_efficiency_loss ?? 0),
  );
  write(
    "factory_bombing_defence_value",
    percentText(mods.factory_bombing_defence ?? 0),
  );
  for (const { text, group } of CATEGORY_TEXTS)
    write(text, String(queue.categories?.[group] ?? 0));
  return texts;
}

export function lineTexts(
  line: GuiQueueLine,
  auto: string,
): Record<string, string> {
  const texts: Record<string, string> = {
    priority: String(line.priority),
    name: line.name,
    general_name: line.general_name,
    speed: line.speed ?? "",

    factories_used_txt:
      line.factories < line.requested
        ? `${line.factories}/${line.requested}`
        : String(line.factories),
  };
  if (line.naval) {
    texts.amount = String(line.amount ?? 1);
    texts.deploy_name = auto;
  } else if (line.stock !== undefined) {
    texts.total_need = String(line.stock);
  }
  return texts;
}

const LINE_HIDDEN = [
  "factories_sub_5x",
  "factories_sub_10x",
  "factories_add_5x",
  "factories_add_10x",
  "creator_flag",
  "outdated_equipment_overlay",
  "carrier_fighter_icon",
  "carrier_air_wing_composition",
  "equipment_designer_can_upgrade",
  "equipment_designer",
  "convert",
];

export function lineHidden(line: GuiQueueLine): string[] {
  if (line.naval)
    return [...LINE_HIDDEN, "name_list_btn", "total_need", "need_state_icon"];
  const bars = [
    "efficiency_progressbar_to_max",
    "efficiency_progressbar",
    "sub_tech",
  ];
  return line.stock === undefined
    ? [...LINE_HIDDEN, ...bars, "total_need", "need_state_icon"]
    : [...LINE_HIDDEN, ...bars];
}

function useSprites(projectId: number, names: string[], key: string) {
  return useSWR<Record<string, GuiSprite>>(
    names.length
      ? ["game-view-sprites", projectId, key, names.join(",")]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(names);
    },
    { revalidateOnFocus: false },
  );
}

function useGrids(entry: GuiWindow | undefined) {
  return useMemo(() => {
    if (!entry) return null;
    const node = structuredClone(entry.node) as GuiNode;
    assignIds(node);
    const rects = measureWindow(node, entry.sprites, LINE_W, LINE_H);
    const rect = (name: string) =>
      rects[idOf(matchNamePath(node, name)[0])] as
        { x: number; y: number; w: number; h: number } | undefined;
    return {
      factories: rect("factories_grid"),
      resources: rect("resources_grid"),
      equipment: rect("equipment_button") ?? rect("equipment_icon"),
    };
  }, [entry]);
}

export function QueueLines({
  projectId,
  lines,
  frames,
  resourceChip,
  onInspectEquipment,
}: {
  projectId: number;
  lines: GuiQueueLine[];
  frames: Record<string, number>;

  resourceChip: (
    resource: string,
    frame: number,
    amount: number,
    left: number,
    top: number,
  ) => ReactNode;

  onInspectEquipment?: (equipment: string) => void;
}) {
  const military = useTemplateWindow(
    projectId,
    lines.some((line) => !line.naval) ? MILITARY_LINE : null,
  );
  const naval = useTemplateWindow(
    projectId,
    lines.some((line) => line.naval) ? NAVAL_LINE : null,
  );
  const militaryFactory = useTemplateWindow(
    projectId,
    lines.length ? MILITARY_FACTORY : null,
  );
  const navalFactory = useTemplateWindow(
    projectId,
    lines.some((line) => line.naval) ? NAVAL_FACTORY : null,
  );
  const auto = useSWR<Record<string, string>>(
    lines.some((line) => line.naval)
      ? [
          "game-view-loc",
          projectId,
          "PRODUCTION_NAVAL_DEPLOYMENT_AUTO_DEPLOYMENT",
        ]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(["PRODUCTION_NAVAL_DEPLOYMENT_AUTO_DEPLOYMENT"]);
    },
    { revalidateOnFocus: false },
  );
  const iconNames = useMemo(
    () =>
      [
        ...new Set(
          lines
            .map((line) => line.icon)
            .filter((icon): icon is string => Boolean(icon)),
        ),
      ].sort(),
    [lines],
  );
  const icons = useSprites(projectId, iconNames, "queue-icons");
  const militaryGrids = useGrids(military.data);
  const navalGrids = useGrids(naval.data);
  const autoText =
    auto.data?.PRODUCTION_NAVAL_DEPLOYMENT_AUTO_DEPLOYMENT ?? "Auto";

  return (
    <>
      {lines.map((line, at) => {
        const entry = line.naval ? naval.data : military.data;
        const grids = line.naval ? navalGrids : militaryGrids;
        const factory = line.naval ? navalFactory.data : militaryFactory.data;
        if (!entry) return null;
        const icon =
          line.icon && icons.data?.[line.icon] ? line.icon : undefined;
        return (
          <div
            key={line.priority}
            aria-label={`production line ${line.priority}`}
            style={{
              position: "absolute",
              left: 0,
              top: at * LINE_H,
              width: LINE_W,
              height: LINE_H,
            }}
          >
            <Filled
              entry={entry}
              left={0}
              top={0}
              width={LINE_W}
              height={LINE_H}
              texts={lineTexts(line, autoText)}
              pictures={icon ? { equipment_icon: icon } : undefined}
              hide={lineHidden(line)}
              extraSprites={icons.data ?? {}}
            />
            {factory &&
              grids?.factories &&
              Array.from(
                { length: Math.min(line.factories, FACTORY.across * 3) },
                (_, k) => (
                  <Filled
                    key={k}
                    entry={factory}
                    left={
                      (grids.factories as { x: number }).x +
                      (k % FACTORY.across) * FACTORY.w
                    }
                    top={
                      (grids.factories as { y: number }).y +
                      Math.floor(k / FACTORY.across) * FACTORY.h
                    }
                    width={FACTORY.w}
                    height={FACTORY.h}
                    texts={{}}
                    hide={["icon_overlay"]}
                    extraSprites={{}}
                  />
                ),
              )}
            {grids?.resources &&
              Object.entries(line.resources ?? {})
                .sort((a, b) => (frames[a[0]] ?? 99) - (frames[b[0]] ?? 99))
                .slice(0, 3)
                .map(([resource, amount], k) =>
                  resourceChip(
                    resource,
                    frames[resource] ?? 1,
                    Math.round(amount),
                    (grids.resources as { x: number }).x,
                    (grids.resources as { y: number }).y + k * RESOURCE.h,
                  ),
                )}
            {grids?.equipment && onInspectEquipment && (
              <button
                type="button"
                aria-label={"inspect production equipment " + line.equipment}
                onClick={() => onInspectEquipment(line.equipment)}
                style={{
                  position: "absolute",
                  left: grids.equipment.x,
                  top: grids.equipment.y,
                  width: grids.equipment.w || 72,
                  height: grids.equipment.h || 72,
                  padding: 0,
                  border: "none",
                  background: "transparent",
                  cursor: "inherit",
                  pointerEvents: "auto",
                }}
              />
            )}
          </div>
        );
      })}
    </>
  );
}
