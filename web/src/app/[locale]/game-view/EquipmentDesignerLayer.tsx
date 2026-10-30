"use client";

import { useMemo } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { draftQuery, type DesignDraft } from "./designEdit";
import { FieldHit } from "./FieldHit";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";

export const TANK_DESIGNER = "tank_designer_view";
export const PLANE_DESIGNER = "plane_designer_view";
export const SHIP_DESIGNER = "countryequipmentdesignerview";

export const DESIGNER_VIEWS: Readonly<Record<string, string>> = {
  tank: TANK_DESIGNER,
  plane: PLANE_DESIGNER,
  ship: SHIP_DESIGNER,
};
export const isDesignerView = (name: string | undefined) =>
  name === TANK_DESIGNER || name === PLANE_DESIGNER || name === SHIP_DESIGNER;
const SLOT_ENTRY = "equipment_designer_module_slot_entry";
const STAT_ENTRY = "equipment_designer_stat_entry";

const SLOT_W = 76;
const SLOT_H = 47;
const STAT_W = 155;
const STAT_H = 20;

const PREVIEW_W = 508;
const PREVIEW_H = 248;

const GRID_X = 5;
const EMPTY_SLOT = "GFX_EMSI_empty";
const LOCKED_SLOT = "GFX_EMSI_locked";
const SLOT_ROWS = ["fixed", "custom"] as const;

export const TANK_DESIGNER_POPUPS = [
  "equipment_select_model",
  "equipment_select_sprite",
  "historical_select_design",
  "niche_icon_selector",
  "module_selector_window",
  "designer_intel_container",
  "referenced_info",
  "equipments/expanded_window",
  "dropdown_tank_roles/expanded_window",
  "equipment_names/expanded_window",

  "refit_cost",

  "equipments/name_reset",

  "select_model",
  "toggle_model_selector_button",
  "select3d_button",
  "tank_designer_view/border_container",
];

export interface GuiDesignerSlot {
  slot: string;

  row?: "fixed" | "custom";
  index: number;
  required: boolean;
  module: string;
  label?: string;
  category?: string;
  icon?: string;

  empty_icon?: string;

  allowed?: { name: string; label: string; category: string }[];
  slot_label?: string;
}

export interface GuiDesignerStat {
  key: string;
  stat: string;
  format: string;
  value: number;
  label: string;
}

export interface GuiEquipmentDesign {
  found: boolean;

  kind?: string;
  name: string;
  type: string;
  archetype: string;
  type_label: string;
  year: number;
  role?: string;
  battalion?: string;
  texts?: { title?: string; cost?: string; save?: string };

  experience?: number;

  experience_icon?: string;

  version?: string;

  names?: string;

  dominance?: number;

  cost_decimals?: number;

  resource_order?: string[];

  icon?: string;
  slots: GuiDesignerSlot[];
  upgrades: { name: string; label: string; level: number; max?: number }[];

  variant?: boolean;

  designs?: { name: string; type: string; type_label: string }[];
  columns: {
    base: GuiDesignerStat[];
    combat: GuiDesignerStat[];
    misc: GuiDesignerStat[];
  };
  cost: number;
  resources?: Record<string, number>;
  blueprint: string;
  sprites: Record<string, GuiSprite>;
}

export function useEquipmentDesigner(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
  name: string | null,
  draft?: DesignDraft | null,
  options = false,
) {
  const { modules, upgrades } = draftQuery(draft);
  return useSWR<GuiEquipmentDesign>(
    tag && name
      ? [
          "game-view-equipment-designer",
          projectId,
          tag,
          date ?? "",
          name,
          modules,
          upgrades,
          options,
        ]
      : null,
    async () => {
      const query = new URLSearchParams({
        tag: tag as string,
        name: name as string,
      });
      if (date) query.set("date", date);

      if (modules) query.set("modules", modules);
      if (upgrades) query.set("upgrades", upgrades);
      if (options) query.set("options", "1");
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/equipment-designer?${query}`,
        ),
      );
      return readGuiResponse<GuiEquipmentDesign>(response);
    },
    { revalidateOnFocus: false, keepPreviousData: true },
  );
}

export function formatDesignerStat(value: number, format: string): string {
  const n = Number(value) || 0;
  if (format === "speed")
    return `${(Math.floor(n * 10 + 1e-9) / 10).toFixed(1)} km/h`;
  if (format === "km") return `${n.toFixed(0)} km`;

  if (format === "kn") return `${n.toFixed(1)} kn`;
  if (format === "int") return n.toFixed(0);
  if (format === "pct0") return `${(n * 100).toFixed(0)}%`;
  if (format === "percent") return `${(n * 100).toFixed(1)}%`;
  if (format === "rate") return n.toFixed(2);
  return n.toFixed(1);
}

export function designerSaveText(
  format: string | undefined,
  cost = 0,
  icon = "army_experience",
): string {
  if (!format) return "";
  return format
    .replace("$XPICON$", icon)
    .replace(/\$XPCOST[^$]*\$/, `§Y${cost}§!`);
}

export function statEntryWindow(win: GuiWindow): GuiWindow {
  const node = structuredClone(win.node) as GuiNode;
  for (const child of node.children ?? []) {
    if (child.name === "count") {
      child.position = { ...(child.position ?? { y: 0 }), x: 0 };
      (child as GuiNode & { maxwidth?: number }).maxwidth = STAT_W;
    }
  }
  return { ...win, node };
}

export function designerCostText(
  format: string | undefined,
  cost: number,
  decimals = 2,
): string {
  const value = (Number(cost) || 0).toFixed(decimals);
  if (!format) return value;
  return format.replace(/\$VALUE[^$]*\$/, value);
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function equipmentDesignerWindow(
  win: GuiWindow,
  data: GuiEquipmentDesign | undefined,

  draftName?: string,
): { win: GuiWindow; hidden: string[]; texts: Record<string, string> } {
  const node = structuredClone(win.node) as GuiNode;
  const walk = (at: GuiNode) => {
    for (const child of at.children ?? []) {
      if (child.name === "production_value" && win.name === SHIP_DESIGNER)
        delete (child as { orientation?: unknown }).orientation;
      if (
        child.name === "btn_equipment_icon" &&
        data?.icon &&
        data.sprites[data.icon]
      )
        child.sprite = data.icon;
      walk(child);
    }
  };
  walk(node);
  assignIds(node);
  const ids = (path: string) =>
    matchNamePath(node, path).map(idOf).filter(Boolean);
  const hidden = TANK_DESIGNER_POPUPS.flatMap(ids);

  hidden.push(...ids("plane_role/role_icon"));
  const texts: Record<string, string> = {};
  const write = (path: string, text: string) => {
    for (const id of ids(path)) texts[id] = text;
  };
  if (data?.found) {
    if (data.texts?.title) write("designer_title", data.texts.title);
    const renamed = !!draftName && draftName !== data.name;
    write("equipments/name", renamed ? draftName : data.name);
    write("position_version", renamed ? "" : (data.version ?? ""));
    write(
      data.kind === "plane"
        ? "plane_role/role_text"
        : "dropdown_tank_roles/role_text",
      data.role ?? "",
    );
    write("blueprint_type_name", data.type_label);
    write(
      "production_value",
      designerCostText(data.texts?.cost, data.cost, data.cost_decimals ?? 2),
    );
    if (data.names) write("equipment_names/name", data.names);
    if (data.kind === "ship") {
      write(
        "info/ship_designer_naval_dominance_value",
        String(Math.floor((data.dominance ?? 0) + 1e-9)),
      );
    }
    const resources = data.resources ?? {};
    const order = data.resource_order?.length
      ? data.resource_order
      : Object.keys(resources);
    write(
      "production_resources",
      order.map((name) => String(resources[name] ?? 0)).join(" "),
    );
    const xpIcon = data.experience_icon || "army_experience";
    write("save_button", designerSaveText(data.texts?.save, 0, xpIcon));
    write("experience", `£${xpIcon} ${data.experience ?? 0}`);
    [0, 1].forEach((at) => {
      const upgrade = data.upgrades[at];
      if (!upgrade) {
        hidden.push(...ids(`equipment_upgrade_${at}`));
        return;
      }
      write(`equipment_upgrade_${at}/name`, upgrade.label);
      write(`equipment_upgrade_${at}/level`, String(upgrade.level));
    });
  }
  return {
    win: {
      ...win,
      node,
      sprites: { ...win.sprites, ...(data?.sprites ?? {}) },
    },
    hidden,
    texts,
  };
}

export function slotPlace(
  win: GuiWindow,
  index: number,
  row: "fixed" | "custom" = "custom",
): { x: number; y: number } | null {
  const modules = matchNamePath(win.node, "equipment_modules")[0];
  const pos = matchNamePath(
    win.node,
    `equipment_modules/pos_${row}_module_slot_window_${index}`,
  )[0];
  if (!modules || !pos) return null;
  return {
    x: (Number(modules.position?.x) || 0) + (Number(pos.position?.x) || 0),
    y: (Number(modules.position?.y) || 0) + (Number(pos.position?.y) || 0),
  };
}

export function lockedSlots(
  win: GuiWindow,
  slots: GuiDesignerSlot[],
): { row: "fixed" | "custom"; index: number }[] {
  const out: { row: "fixed" | "custom"; index: number }[] = [];
  const modules = matchNamePath(win.node, "equipment_modules")[0];
  for (const row of SLOT_ROWS) {
    const used = new Set(
      slots
        .filter((slot) => (slot.row ?? "custom") === row)
        .map((slot) => slot.index),
    );
    for (const child of modules?.children ?? []) {
      const match = child.name?.match(
        new RegExp(`^pos_${row}_module_slot_window_(\\d+)$`),
      );
      if (match && !used.has(Number(match[1])))
        out.push({ row, index: Number(match[1]) });
    }
  }
  return out;
}

export function blueprintUnfitted(
  node: GuiNode,
  slots: GuiDesignerSlot[],
): string[] {
  const holder = matchNamePath(node, "module_slots")[0];
  const out: string[] = [];
  for (const slot of holder?.children ?? []) {
    const name = slot.name ?? "";
    const fitted = slots.find((entry) => entry.slot === name);
    const fittedModule = fitted?.module ?? "";
    const pictures = (slot.children ?? [])
      .map((child) => child.name ?? "")
      .filter((child) => child && !child.includes("@"));

    const own = fittedModule
      ? (pictures
          .filter((child) => fittedModule.startsWith(child))
          .sort((a, b) => b.length - a.length)[0] ??
        pictures.find((child) => child === fitted?.category))
      : undefined;
    for (const child of pictures) {
      if (child !== name && child !== own) out.push(`${name}/${child}`);
    }
  }
  return out;
}

export function blueprintHighlights(node: GuiNode): string[] {
  const out: string[] = [];
  const walk = (at: GuiNode) => {
    for (const child of at.children ?? []) {
      if (child.name?.includes("@")) out.push(child.name);
      walk(child);
    }
  };
  walk(node);
  return [...new Set(out)];
}

export function EquipmentDesignerLayer({
  win,
  projectId,
  tag,
  date,
  name,
  draft = null,
  editMode = false,
  onEditEquipment,
  onEditDesign,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  name: string | null;

  draft?: DesignDraft | null;
  editMode?: boolean;

  onEditEquipment?: (name: string) => void;

  onEditDesign?: (slot?: string) => void;
}) {
  const t = useTranslations("gameViewMilitary.designer");
  const active = isDesignerView(win.name);
  const data = useEquipmentDesigner(
    projectId,
    active ? tag : null,
    date,
    active ? name : null,
    draft,
    editMode,
  );
  const slotWin = useTemplateWindow(projectId, active ? SLOT_ENTRY : null);
  const statTemplate = useTemplateWindow(projectId, active ? STAT_ENTRY : null);
  const statWin = useMemo(
    () => (statTemplate.data ? statEntryWindow(statTemplate.data) : undefined),
    [statTemplate.data],
  );
  const blueprintWin = useTemplateWindow(
    projectId,
    active && data.data?.blueprint ? data.data.blueprint : null,
  );
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1091, 542) : {}),
    [active, win],
  );
  const highlights = useMemo(
    () =>
      blueprintWin.data
        ? [
            ...blueprintHighlights(blueprintWin.data.node),
            ...blueprintUnfitted(
              blueprintWin.data.node,
              data.data?.slots ?? [],
            ),
          ]
        : [],
    [blueprintWin.data, data.data],
  );
  const sprites = useMemo(() => data.data?.sprites ?? {}, [data.data]);
  if (!active || !data.data?.found) return null;
  const design = data.data;
  const modules = rects["name:equipment_modules"];
  const preview = rects["name:equipment_preview"];
  const columns: [keyof GuiEquipmentDesign["columns"], string][] = [
    ["base", "base_stats"],
    ["combat", "combat_stats"],
    ["misc", "misc_stats"],
  ];
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {blueprintWin.data && modules && preview && (
        <Filled
          entry={blueprintWin.data}
          left={preview.x}
          top={preview.y}
          width={PREVIEW_W}
          height={PREVIEW_H}
          texts={{}}
          hide={highlights}
          extraSprites={sprites}
        />
      )}
      {slotWin.data &&
        design.slots.map((slot) => {
          const place = slotPlace(win, slot.index, slot.row ?? "custom");
          if (!place) return null;
          return (
            <Filled
              key={slot.slot}
              entry={slotWin.data as GuiWindow}
              left={place.x}
              top={place.y}
              width={SLOT_W}
              height={SLOT_H}
              texts={{}}
              pictures={{
                btn_module_slot: slot.icon ?? slot.empty_icon ?? EMPTY_SLOT,
              }}
              hide={["icn_requirements", "icn_warning"]}
              extraSprites={sprites}
            />
          );
        })}
      {slotWin.data &&
        lockedSlots(win, design.slots).map(({ row, index }) => {
          const place = slotPlace(win, index, row);
          if (!place) return null;
          return (
            <Filled
              key={`locked-${row}-${index}`}
              entry={slotWin.data as GuiWindow}
              left={place.x}
              top={place.y}
              width={SLOT_W}
              height={SLOT_H}
              texts={{}}
              pictures={{ btn_module_slot: LOCKED_SLOT }}
              hide={["icn_requirements", "icn_warning"]}
              extraSprites={sprites}
            />
          );
        })}
      {statWin &&
        columns.map(([column, box]) => {
          const at = rects[`name:${box}`];
          if (!at) return null;
          return design.columns[column].map((row, index) => (
            <Filled
              key={`${column}-${row.stat}`}
              entry={statWin}
              left={at.x + GRID_X}
              top={at.y + index * STAT_H}
              width={STAT_W}
              height={STAT_H}
              texts={{
                label: row.label,
                count: formatDesignerStat(row.value, row.format),
              }}
              extraSprites={sprites}
            />
          ));
        })}
      {onEditDesign &&
        design.slots.map((slot) => {
          const place = slotPlace(win, slot.index, slot.row ?? "custom");
          if (!place) return null;
          return (
            <FieldHit
              key={`edit-${slot.slot}`}
              x={place.x}
              y={place.y}
              w={SLOT_W}
              h={SLOT_H}
              label={t("slot", { slot: slot.slot_label || slot.slot })}
              onClick={() => onEditDesign(slot.slot)}
            />
          );
        })}
      {onEditDesign && rects["name:equipments"] && (
        <FieldHit
          x={rects["name:equipments"].x}
          y={rects["name:equipments"].y}
          w={rects["name:equipments"].w}
          h={rects["name:equipments"].h}
          label={t("design", { name: design.name })}
          onClick={() => onEditDesign()}
        />
      )}
      {onEditEquipment && preview && (
        <FieldHit
          x={preview.x}
          y={preview.y}
          w={PREVIEW_W}
          h={PREVIEW_H}
          label={t(
            design.kind === "plane"
              ? "airframe"
              : design.kind === "ship"
                ? "hull"
                : "chassis",
            { name: design.type_label },
          )}
          onClick={() => onEditEquipment(design.type)}
        />
      )}
    </div>
  );
}
