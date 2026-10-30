"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { guiArray, readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";
import { namedPrinted } from "./textSources";
import { matchNamePath } from "./gameNavigation";

export const MIO_LIST_WINDOW = "industrial_organisation_list_window";
const ITEM = "industrial_organisation_list_item";
const EQUIPMENT_ITEM = "industrial_organisation_equipment_icon_item";

const ROW_H = 100;
const WHEEL_STEP = 98;

const EQUIPMENT_W = 70;
const NO_HIDDEN = new Set<string>();

export function opensIndustrialOrgDetails(
  name: string | null | undefined,
): boolean {
  return name === "background_button" || name === "details_button";
}

export type MioCategory = "materiel" | "tank" | "plane" | "ship";

export const MIO_SLOTS: Readonly<Record<string, MioCategory>> = {
  materiel_manufacturer: "materiel",
  tank_manufacturer: "tank",
  aircraft_manufacturer: "plane",
  naval_manufacturer: "ship",
};

export const MIO_FILTER_BUTTONS: Readonly<Record<string, MioCategory>> = {
  materiel_filter_button: "materiel",
  tank_filter_button: "tank",
  plane_filter_button: "plane",
  ship_filter_button: "ship",
};

const HIDDEN_WINDOW_PARTS = [
  "sorting_options/sort_cost",
  "sorting_options/sort_capacity",
];

const HIDDEN_ROW_PARTS = new Set([
  "task_capacity_container",
  "has_policy_icon",
  "assign_cost",
  "details_button_glow",
  "ai_value",
]);

export interface GuiIndustrialOrgEquipment {
  group: string;
  label?: string;
  icon: string;
  icon_sprite?: GuiSprite;

  design: number;
  production: number;
  level: number;
}

export interface GuiIndustrialOrg {
  token: string;
  name: string;
  icon?: string;
  icon_sprite?: GuiSprite;
  initial_trait?: string;
  category: MioCategory;
  size: number;
  equipment: GuiIndustrialOrgEquipment[];
}

export function useIndustrialOrgs(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useSWR<{ tag: string; items: GuiIndustrialOrg[] }>(
    tag ? ["game-view-industrial-orgs", projectId, tag, date ?? ""] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/industrial-orgs?${query}`,
        ),
      );
      return readGuiResponse<{ tag: string; items: GuiIndustrialOrg[] }>(
        response,
      );
    },
    { revalidateOnFocus: false },
  );
}

export function filterOrgs(
  items: readonly GuiIndustrialOrg[],
  filters: ReadonlySet<MioCategory>,
): GuiIndustrialOrg[] {
  return filters.size
    ? items.filter((item) => filters.has(item.category))
    : [...items];
}

export function toggleFilter(
  filters: ReadonlySet<MioCategory>,
  category: MioCategory,
): ReadonlySet<MioCategory> {
  const next = new Set(filters);
  if (next.has(category)) next.delete(category);
  else next.add(category);
  return next;
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function mioListWindow(
  win: GuiWindow,
  filters: ReadonlySet<MioCategory>,
): { win: GuiWindow; hidden: string[] } {
  const node = structuredClone(win.node) as GuiNode;
  const walk = (at: GuiNode) => {
    for (const child of at.children ?? []) {
      const category = MIO_FILTER_BUTTONS[child.name ?? ""];
      if (category) child.frame = filters.has(category) ? 2 : 1;
      walk(child);
    }
  };
  walk(node);
  const hidden = HIDDEN_WINDOW_PARTS.flatMap((path) =>
    matchNamePath(node, path).map(idOf),
  ).filter(Boolean);
  return { win: { ...win, node }, hidden };
}

function EquipmentIcon({
  entry,
  equipment,
  left,
  top,
}: {
  entry: GuiWindow;
  equipment: GuiIndustrialOrgEquipment;
  left: number;
  top: number;
}) {
  const edit = useEditCanvas();
  const icon = equipment.icon_sprite ? equipment.icon : null;
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    for (const child of clone.children ?? []) {
      if (child.name === "icon") child.sprite = icon;
    }
    assignIds(clone);
    return clone;
  }, [entry.node, icon]);
  const sprites = useMemo(
    () =>
      icon && equipment.icon_sprite
        ? { ...entry.sprites, [icon]: equipment.icon_sprite }
        : entry.sprites,
    [entry.sprites, icon, equipment.icon_sprite],
  );
  const texts = useMemo(() => {
    const level = idOf(matchNamePath(node, "trait_level")[0]);
    return level ? { [level]: String(equipment.level) } : {};
  }, [node, equipment.level]);
  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${equipment.group}`,
          { ...entry, sprites },
          node,
          sprites,
          texts,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={EQUIPMENT_W}
        baseH={48}
      />
    </div>
  );
}

function OrgRow({
  entry,
  equipmentEntry,
  org,
  top,
  onDetails,
}: {
  entry: GuiWindow;
  equipmentEntry: GuiWindow;
  org: GuiIndustrialOrg;
  top: number;
  onDetails?: (token: string) => void;
}) {
  const edit = useEditCanvas();
  const icon = org.icon && org.icon_sprite ? org.icon : null;
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    const walk = (at: GuiNode) => {
      at.children = (at.children ?? []).filter(
        (child) => !HIDDEN_ROW_PARTS.has(child.name ?? ""),
      );
      for (const child of at.children) {
        if (child.name === "icon") child.sprite = icon;

        if (child.name === "initial_trait")
          child.text = child.text || "initial_trait";
        walk(child);
      }
    };
    walk(clone);
    assignIds(clone);
    return clone;
  }, [entry.node, icon]);
  const sprites = useMemo(
    () =>
      icon && org.icon_sprite
        ? { ...entry.sprites, [icon]: org.icon_sprite }
        : entry.sprites,
    [entry.sprites, icon, org.icon_sprite],
  );
  const texts = useMemo(() => {
    const out: Record<string, string> = {};
    const set = (path: string, text: string) => {
      const id = idOf(matchNamePath(node, path)[0]);
      if (id) out[id] = text;
    };
    set("name", org.name);
    set("initial_trait", org.initial_trait ? `§Y${org.initial_trait}§!` : "");
    set("size_icon_window/size", String(org.size));
    return out;
  }, [node, org.name, org.initial_trait, org.size]);

  const printed = useMemo(
    () => namedPrinted(node, "name", org.name, org.token),
    [node, org.name, org.token],
  );
  const rects = useMemo(
    () => measureWindow(node, sprites, 1920, 1080),
    [node, sprites],
  );
  const box = rects["name:equipment_icons"];
  return (
    <div style={{ position: "absolute", left: 0, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${org.token}`,
          { ...entry, sprites },
          node,
          sprites,
          texts,
          printed,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={520}
        baseH={ROW_H}
        onActivate={(pressed) => {
          if (!opensIndustrialOrgDetails(pressed.name) || !onDetails)
            return false;
          onDetails(org.token);
          return true;
        }}
      />
      {box &&
        guiArray(org.equipment).map((equipment, index) => (
          <EquipmentIcon
            key={equipment.group}
            entry={equipmentEntry}
            equipment={equipment}
            left={box.x + index * EQUIPMENT_W}
            top={box.y}
          />
        ))}
    </div>
  );
}

export function IndustrialOrgLayer({
  win,
  projectId,
  tag,
  date,
  filters,
  onDetails,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  filters: ReadonlySet<MioCategory>;
  onDetails?: (token: string) => void;
}) {
  const active = win.name === MIO_LIST_WINDOW;
  const orgs = useIndustrialOrgs(projectId, active ? tag : null, date);
  const item = useTemplateWindow(projectId, active ? ITEM : null);
  const equipment = useTemplateWindow(
    projectId,
    active ? EQUIPMENT_ITEM : null,
  );
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const [scroll, setScroll] = useState(0);
  const shown = useMemo(
    () => filterOrgs(guiArray(orgs.data?.items), filters),
    [orgs.data, filters],
  );
  const list = rects["name:organisation_list"];
  const container = rects["name:organisations"];
  if (!active || !list || !item.data || !equipment.data) return null;

  const holder = matchNamePath(win.node, "organisations")[0];
  const marginTop =
    Number((holder?.raw?.margin as { top?: number } | undefined)?.top) || 0;
  const viewTop = list.y + marginTop;
  const viewH = Math.max(
    0,
    Math.min(
      list.y + list.h,
      container ? container.y + container.h : list.y + list.h,
    ) - viewTop,
  );
  const maxScroll = Math.max(0, shown.length * ROW_H - viewH);
  const at = Math.min(scroll, maxScroll);
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      <div
        style={{
          position: "absolute",
          left: list.x,
          top: viewTop,
          width: list.w,
          height: viewH,
          overflow: "hidden",
          pointerEvents: "auto",
        }}
        onWheel={(e) => {
          const step = e.deltaY > 0 ? WHEEL_STEP : -WHEEL_STEP;
          setScroll((now) =>
            Math.min(maxScroll, Math.max(0, Math.min(now, maxScroll) + step)),
          );
        }}
      >
        {shown.map((org, index) => (
          <OrgRow
            key={org.token}
            entry={item.data as GuiWindow}
            equipmentEntry={equipment.data as GuiWindow}
            org={org}
            top={index * ROW_H - at}
            onDetails={onDetails}
          />
        ))}
      </div>
    </div>
  );
}
