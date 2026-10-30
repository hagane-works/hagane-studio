"use client";

import { useMemo } from "react";
import useSWR from "swr";
import { guiArray, guiRecord, readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";
import { TECHNOLOGY } from "./ResearchLayer";
import { SpriteBar, type BarSprite } from "./SpriteBar";

export const FACILITIES_VIEW = "facilities_view";
const BAND_ENTRY = "specialization_item";
const CARD_ENTRY = "program_item";

export type ResearchTab = "slots" | "facilities";

const TAB_OPEN = 1;
const TAB_OTHER = 2;

const BAND_H = 37;
const CARD_H = 210;
const CARD_W = 515;

const BREAKTHROUGH_BAR = { w: 60, h: 8 };

const OVERVIEW_TOP = 10;
const BAND_LIST_TOP = 33;
const BOTTOM_GAP = 8;

const CARD_HIDDEN = [
  "unread_prototype_reward",
  "facility_damage_bar",
  "supply_icon",
  "abort_dismantle_facility_button",
  "faction_icon",
  "change_scientist_button",
  "unassign_scientist_button",
  "scientist_injured",
  "scientist_removal_clock_icon",
  "scientist_name",
  "skill_levels",
  "scientist_trait_icons",
  "status_info_text",
  "basic_research",
  "ongoing_project_window",
  "dismantle_window",
];

export interface GuiSpecialization {
  id: string;
  label: string;
  name: string;
  projects: number;
  color: number[] | null;
  points: number;
  progress: number;
  icon_frame?: number;
  building?: string;
}

export interface GuiFacility {
  state: number;
  state_name: string;
  province: number;
  building: string;
  label: string;
  specialization: string;
  icon_frame: number;
}

export interface GuiSpecialProject {
  id: string;
  label: string;

  label_key: string;

  generic?: string;
  specialization: string;
  specialization_name: string;
  available: boolean;
  icon: string;
  breakthrough_cost: {
    specialization: string;
    amount: number;
    icon_frame?: number;
  }[];
  prototype_time: string;
  complexity: string;
  file: string;
  allowed_script: string;
  available_script: string;
}

export interface GuiSpecialProjects {
  tag: string;
  specializations: GuiSpecialization[];
  facilities: GuiFacility[];
  projects: GuiSpecialProject[];
  sprites: Record<string, GuiSprite>;
}

export function useSpecialProjects(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useSWR<GuiSpecialProjects>(
    tag ? ["game-view-special-projects", projectId, tag, date ?? ""] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/special-projects?${query}`,
        ),
      );
      return readGuiResponse<GuiSpecialProjects>(response);
    },
    { revalidateOnFocus: false },
  );
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function researchTabWindow(
  win: GuiWindow,
  tab: ResearchTab,
): { win: GuiWindow; hidden: string[] } {
  const node = structuredClone(win.node) as GuiNode;
  assignIds(node);
  for (const button of matchNamePath(node, "tabs/research_slots_tab"))
    button.frame = tab === "slots" ? TAB_OPEN : TAB_OTHER;
  for (const button of matchNamePath(node, "tabs/facilities_tab"))
    button.frame = tab === "facilities" ? TAB_OPEN : TAB_OTHER;
  const hidden =
    tab === "facilities"
      ? matchNamePath(node, "research_slots").map(idOf).filter(Boolean)
      : [];
  return { win: { ...win, node }, hidden };
}

export function facilitiesView(view: GuiWindow, bands: number): GuiWindow {
  const node = structuredClone(view.node) as GuiNode;
  for (const bottom of matchNamePath(
    node,
    "facilities_overview_bottom_window",
  )) {
    bottom.position = {
      x: bottom.position?.x ?? 1,
      y: OVERVIEW_TOP + BAND_LIST_TOP + bands * BAND_H + BOTTOM_GAP,
    };
  }
  assignIds(node);
  return { ...view, node };
}

export function bandWash(color: number[] | null | undefined): string {
  const [r, g, b] = color && color.length >= 3 ? color : [128, 128, 128];

  return `linear-gradient(to right, rgba(${r}, ${g}, ${b}, 0) 10%, rgba(${r}, ${g}, ${b}, 0.12) 40%, rgba(${r}, ${g}, ${b}, 0.3))`;
}

type Rect = { x: number; y: number; w: number; h: number };

function rectsOf(
  node: GuiNode,
  sprites: GuiWindow["sprites"],
  w: number,
  h: number,
  paths: readonly string[],
) {
  const measured = measureWindow(node, sprites, w, h);
  const out: Record<string, Rect | undefined> = {};
  for (const path of paths)
    out[path] = measured[idOf(matchNamePath(node, path)[0])] as
      Rect | undefined;
  return out;
}

const SCREEN_PARTS = [
  "tabs/research_slots_tab",
  "tabs/facilities_tab",
  "facilities",
];
const VIEW_PARTS = [
  "specialization_item_list",
  "program_list",
  "open_scientist_roster",
];

export function FacilitiesLayer({
  win,
  projectId,
  tag,
  date,
  tab,
  onTab,
  onOpenFacility,
  onOpenScientists,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  tab: ResearchTab;
  onTab: (tab: ResearchTab) => void;

  onOpenFacility?: (facility: GuiFacility) => void;

  onOpenScientists?: () => void;
}) {
  const active = win.name === TECHNOLOGY;
  const open = active && tab === "facilities";
  const data = useSpecialProjects(projectId, open ? tag : null, date);
  const view = useTemplateWindow(projectId, open ? FACILITIES_VIEW : null);
  const band = useTemplateWindow(projectId, open ? BAND_ENTRY : null);
  const card = useTemplateWindow(projectId, open ? CARD_ENTRY : null);
  const screen = useMemo(
    () =>
      active ? rectsOf(win.node, win.sprites, 1920, 1080, SCREEN_PARTS) : {},
    [active, win],
  );
  const bands = data.data ? guiArray(data.data.specializations).length : 4;
  const laid = useMemo(() => {
    const at = screen["facilities"];
    if (!view.data || !at) return null;
    const shown = facilitiesView(view.data, bands);
    const parts = rectsOf(
      shown.node,
      shown.sprites,
      at.w || 550,
      at.h || 900,
      VIEW_PARTS,
    );
    return { at, shown, parts };
  }, [view.data, screen, bands]);

  const paper = useMemo(() => {
    if (!card.data) return null;
    const node = structuredClone(card.data.node) as GuiNode & {
      background?: GuiNode | null;
    };
    node.background = {
      name: "Background",
      type: "icontype",
      sprite: "GFX_sp_facility_card_bg",
      frame: 1,
      position: { x: 0, y: 0 },
      children: [],
    } as unknown as GuiNode;
    return { ...card.data, node } as GuiWindow;
  }, [card.data]);
  if (!active) return null;
  const tabHit = (rect: Rect | undefined, which: ResearchTab, label: string) =>
    rect ? (
      <div
        role="button"
        aria-label={label}
        onClick={() => onTab(which)}
        style={{
          position: "absolute",
          left: rect.x,
          top: rect.y,
          width: rect.w || 250,
          height: rect.h || 32,
          cursor: "pointer",
          pointerEvents: "auto",
        }}
      />
    ) : null;
  const tabs = (
    <>
      {tabHit(screen["tabs/research_slots_tab"], "slots", "Research Slots")}
      {tabHit(
        screen["tabs/facilities_tab"],
        "facilities",
        "Experimental Facilities",
      )}
    </>
  );
  if (!open || !data.data || !laid || !band.data || !paper) {
    return (
      <div
        style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
      >
        {tabs}
      </div>
    );
  }
  const projects = data.data;
  const specializations = guiArray(projects.specializations);
  const facilities = guiArray(projects.facilities);
  const sprites = guiRecord(projects.sprites);
  const barSprite = band.data.sprites["GFX_breakthrough_progressbar"] as
    BarSprite | undefined;
  const bandBar = (() => {
    const node = structuredClone(band.data.node) as GuiNode;
    assignIds(node);
    return rectsOf(node, band.data.sprites, 600, BAND_H, ["progress"])[
      "progress"
    ];
  })();
  const list = laid.parts["specialization_item_list"];
  const cards = laid.parts["program_list"];
  const scientists = laid.parts["open_scientist_roster"];
  const originX = laid.at.x;
  const originY = laid.at.y;
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      <Filled
        entry={laid.shown}
        left={originX}
        top={originY}
        width={laid.at.w || 550}
        height={laid.at.h || 900}
        texts={{ owned_facilities_amount: String(facilities.length) }}
        extraSprites={{}}
      />
      {list &&
        specializations.map((item, at) => {
          const left = originX + list.x;
          const top = originY + list.y + at * BAND_H;
          return (
            <div key={item.id}>
              <div
                aria-label={`band ${item.id}`}
                style={{
                  position: "absolute",
                  left: left + 22,
                  top: top + 2,
                  width: list.w - 26,
                  height: BAND_H - 4,
                  background: bandWash(item.color),
                }}
              />
              <Filled
                entry={band.data as GuiWindow}
                left={left}
                top={top}
                width={600}
                height={BAND_H}
                texts={{
                  specialization: item.label,
                  available_projects_amount: String(item.projects),
                  breakthroughs_amount: String(item.points),
                }}
                pictures={{ specialization_button: "GFX_buildings_strip" }}
                frames={
                  item.icon_frame
                    ? { specialization_button: item.icon_frame }
                    : undefined
                }
                hide={["unavailable", "gaining_breakthrough_anim", "progress"]}
                extraSprites={sprites}
              />
              {bandBar && (
                <SpriteBar
                  x={left + bandBar.x}
                  y={top + bandBar.y}
                  size={BREAKTHROUGH_BAR}
                  sprite={barSprite}
                  fraction={item.progress}
                />
              )}
            </div>
          );
        })}
      {cards &&
        facilities.map((facility, at) => (
          <div
            key={`${facility.state}-${facility.province}-${facility.building}-${at}`}
          >
            <Filled
              entry={paper}
              left={originX + cards.x}
              top={originY + cards.y + at * CARD_H}
              width={CARD_W}
              height={CARD_H}
              texts={{
                state_name: facility.state_name,
                facility_name: facility.label,
                research_speed_text: "x1.00",
              }}
              pictures={{ facility_icon: "GFX_buildings_strip" }}
              frames={{ facility_icon: facility.icon_frame }}
              hide={CARD_HIDDEN}
              extraSprites={sprites}
            />
            {onOpenFacility && (
              <div
                role="button"
                aria-label={`facility ${facility.state_name} ${facility.building}`}
                onClick={() => onOpenFacility(facility)}
                style={{
                  position: "absolute",
                  left: originX + cards.x,
                  top: originY + cards.y + at * CARD_H,
                  width: CARD_W,
                  height: CARD_H - 30,
                  cursor: "pointer",
                  pointerEvents: "auto",
                }}
              />
            )}
          </div>
        ))}
      {onOpenScientists && scientists && (
        <div
          role="button"
          aria-label="Scientists"
          onClick={onOpenScientists}
          style={{
            position: "absolute",
            left: originX + scientists.x,
            top: originY + scientists.y,
            width: scientists.w || 130,
            height: scientists.h || 30,
            cursor: "pointer",
            pointerEvents: "auto",
          }}
        />
      )}
      {tabs}
    </div>
  );
}
