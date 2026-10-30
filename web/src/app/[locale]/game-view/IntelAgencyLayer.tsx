"use client";

import { useMemo } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { FieldHit } from "./FieldHit";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";
import { readGuiResponse } from "./guiResponse";

export const INTEL_VIEW = "countryintelligenceagencyview";
export const INSIGNIA_WINDOW = "name_logo_agency_selection_window";
const BRANCH_ENTRY = "upgrade_button";
const PIP_ENTRY = "agency_upgrade_researched_icon";

const PIP_SLOT = { w: 19, h: 20 };
const PIP_OFFSET = { x: -4, y: 6 };
const PIPS_AT = { x: 5, y: 50 };
const PIPS_PER_ROW = 5;
const LOGO_ENTRY = "logo_list_entry";

const BRANCH_SLOT = { w: 102, h: 90 };
const LOGO_SLOT = { w: 78, h: 83 };

const BRANCH_OFFSET = { x: -10, y: 15 };

const LOGO_MARGIN_LEFT = 17;

const BRANCH_VEIL = "agency_branches/upgrades_before_agency";
const LOGOS_PER_ROW = 4;

const LOGO_FRAME = 1;
const LOGO_CHOSEN_FRAME = 2;

export interface GuiIntelInsignia {
  file: string;
  ordinal: number;
  picture: string;
  names: string[];
  raw_names: string[];
  default: boolean;

  available_script: string;
  default_script: string;
}

export interface GuiIntelAgency {
  tag: string;
  insignia: GuiIntelInsignia[];

  chosen: number;

  name: string;
  branches: { id: string; label: string; upgrades: GuiIntelUpgrade[] }[];
  creation: { factories: number; days: number; days_text: string };

  created?: boolean;
  operative_slots?: number;
  texts: {
    create?: string;
    forming?: string;
    no_operative?: string;
    choose?: string;
    defense_level?: string;
    no_upgrade?: string;
    recruiting?: string;
    recruit_days?: string;
    upgrades_duration?: string;
  };
  sprites: Record<string, GuiSprite>;
}

export interface GuiIntelUpgrade {
  id: string;
  label?: string;
  picture?: string;

  frame_sprite?: string;
  levels?: number;
  factories?: number;

  available?: boolean;

  available_script?: string;
  levels_script?: string[];
  file?: string;
}

export interface IntelPick {
  insignia: number | null;
  name: string | null;
}

export const NO_INTEL_PICK: IntelPick = { insignia: null, name: null };

export function useIntelAgency(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useSWR<GuiIntelAgency>(
    tag ? ["game-view-intel-agency", projectId, tag, date ?? ""] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/intel-agency?${query}`,
        ),
      );
      return readGuiResponse<GuiIntelAgency>(response);
    },
    { revalidateOnFocus: false },
  );
}

export function intelChoice(
  data: GuiIntelAgency | undefined,
  pick: IntelPick,
): { insignia: number; name: string } {
  const insignia = pick.insignia ?? data?.chosen ?? -1;
  return { insignia, name: pick.name ?? data?.name ?? "" };
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function intelAgencyWindow(
  win: GuiWindow,
  data: GuiIntelAgency | undefined,
  forming: boolean,
): { win: GuiWindow; hidden: string[]; texts: Record<string, string> } {
  if (data?.created) return createdAgencyWindow(win, data);
  const node = structuredClone(win.node) as GuiNode;
  assignIds(node);
  const hidden: string[] = [];
  const hide = (path: string) => {
    for (const hit of matchNamePath(node, path))
      if (idOf(hit)) hidden.push(idOf(hit));
  };

  hide("name_logo");
  hide("creation_progressbar/creation_progressbar_content");
  hide("agency_agents/upgrade_progressbar");
  hide("agency_agents/operatives_progressbar");

  hide(BRANCH_VEIL);
  if (!forming) hide("before_creation_panel/cancel_creation_button");
  const texts: Record<string, string> = {};
  const write = (path: string, text: string) => {
    for (const hit of matchNamePath(node, path))
      if (idOf(hit)) texts[idOf(hit)] = text;
  };
  if (data) {
    const creation = data.creation;
    if (creation) {
      write("creation_progressbar_factories", String(creation.factories));
      write("creation_progressbar_days", creation.days_text);
    }
    const texts = data.texts ?? {};
    const banner = forming ? texts.forming : texts.create;
    if (banner) write("before_creation_panel/creation_button", banner);
  }
  write("agency_agents/total_operative_text", "0/0");
  return { win: { ...win, node }, hidden, texts };
}

export function createdAgencyWindow(
  win: GuiWindow,
  data: GuiIntelAgency,
): { win: GuiWindow; hidden: string[]; texts: Record<string, string> } {
  const node = structuredClone(win.node) as GuiNode;
  const insignia = Array.isArray(data.insignia) ? data.insignia : [];
  const picture = insignia[data.chosen]?.picture;
  for (const logo of matchNamePath(node, "name_logo/agency_logo"))
    if (picture) logo.sprite = picture;
  assignIds(node);
  const hidden: string[] = [];
  const hide = (path: string) => {
    for (const hit of matchNamePath(node, path))
      if (idOf(hit)) hidden.push(idOf(hit));
  };
  hide("before_creation_panel");
  hide(BRANCH_VEIL);
  hide("agency_agents/operatives_before_agency");

  for (const part of [
    "spymaster_title",
    "spymaster_button",
    "faction_button",
    "spymaster_button_name",
    "spymaster_button_cost",
    "spymaster_flag",
  ]) {
    hide(`spymaster/${part}`);
  }

  hide("upgrade_progressbar/progressbar_content");
  hide("upgrade_progressbar/progressbar_days");
  hide("upgrade_progressbar/cancel_upgrade");
  hide("operatives_progressbar/progressbar_content");
  const texts: Record<string, string> = {};
  const write = (path: string, text: string | undefined) => {
    if (text === undefined) return;
    for (const hit of matchNamePath(node, path))
      if (idOf(hit)) texts[idOf(hit)] = text;
  };
  write("name_logo/agency_name", data.name);
  const agencyTexts = data.texts ?? {};
  write("name_logo/agency_defense_level", agencyTexts.defense_level);
  write("upgrade_progressbar/progressbar_desc", agencyTexts.no_upgrade);
  write("operatives_progressbar/progressbar_desc", agencyTexts.recruiting);
  write("operatives_progressbar/progressbar_days", agencyTexts.recruit_days);
  write("agency_agents/total_operative_text", `0/${data.operative_slots ?? 0}`);
  const agencySprites = data.sprites ?? {};
  const sprites =
    picture && agencySprites[picture]
      ? { ...win.sprites, [picture]: agencySprites[picture] }
      : win.sprites;
  return { win: { ...win, node, sprites }, hidden, texts };
}

export function insigniaWindow(
  win: GuiWindow,
  data: GuiIntelAgency | undefined,
  pick: IntelPick,
): { win: GuiWindow; texts: Record<string, string> } {
  const node = structuredClone(win.node) as GuiNode;
  assignIds(node);
  const texts: Record<string, string> = {};
  for (const hit of matchNamePath(node, "agency_name/agency_name_entry")) {
    if (idOf(hit)) texts[idOf(hit)] = intelChoice(data, pick).name;
  }
  return { win: { ...win, node }, texts };
}

type Rect = { x: number; y: number; w: number; h: number };

function useRects(win: GuiWindow, active: boolean, paths: readonly string[]) {
  return useMemo(() => {
    const out: Record<string, Rect | undefined> = {};
    if (!active) return out;
    const node = structuredClone(win.node) as GuiNode;
    const measured = measureWindow(node, win.sprites, 1920, 1080);
    for (const path of paths)
      out[path] = measured[idOf(matchNamePath(node, path)[0])];
    return out;
  }, [active, win, paths]);
}

const PANEL_PARTS = [
  "agency_branches",
  "agency_branches/upgrades_list",
  BRANCH_VEIL,
  "before_creation_panel/creation_button",
  "before_creation_panel/cancel_creation_button",
];
const WINDOW_PARTS = [
  "logo_list/item_grid",
  "agency_name/agency_name_entry",
  "create_button",
  "btn_close",
];

const hit = (
  rect: Rect | undefined,
  label: string,
  onClick: () => void,
  fallback = { w: 40, h: 40 },
) =>
  rect ? (
    <div
      role="button"
      aria-label={label}
      onClick={onClick}
      style={{
        position: "absolute",
        left: rect.x,
        top: rect.y,
        width: rect.w || fallback.w,
        height: rect.h || fallback.h,
        cursor: "pointer",
        pointerEvents: "auto",
      }}
    />
  ) : null;

export function IntelAgencyLayer({
  win,
  projectId,
  tag,
  date,
  pick,
  forming,
  onOpenInsignia,
  onPick,
  onCreate,
  onCancel,
  onClose,
  onEditInsignia,
  onOpenUpgrades,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  pick: IntelPick;
  forming: boolean;
  onOpenInsignia: () => void;
  onPick: (pick: IntelPick) => void;
  onCreate: () => void;
  onCancel: () => void;
  onClose: () => void;
  editMode?: boolean;

  onEditInsignia?: (item: GuiIntelInsignia) => void;

  onOpenUpgrades?: () => void;
}) {
  const tLayer = useTranslations("gameViewResearch.layer");
  const panel = win.name === INTEL_VIEW;
  const chooser = win.name === INSIGNIA_WINDOW;
  const data = useIntelAgency(projectId, panel || chooser ? tag : null, date);
  const branchEntry = useTemplateWindow(projectId, panel ? BRANCH_ENTRY : null);
  const pipEntry = useTemplateWindow(
    projectId,
    panel && data.data?.created ? PIP_ENTRY : null,
  );
  const logoEntry = useTemplateWindow(projectId, chooser ? LOGO_ENTRY : null);
  const panelAt = useRects(win, panel, PANEL_PARTS);
  const windowAt = useRects(win, chooser, WINDOW_PARTS);
  const logoFrames = useMemo(
    () => ({
      plain: { button: LOGO_FRAME },
      chosen: { button: LOGO_CHOSEN_FRAME },
    }),
    [],
  );

  const veil = useMemo(() => {
    if (!panel) return null;
    const found = matchNamePath(win.node, BRANCH_VEIL)[0];
    if (!found) return null;
    const node = {
      ...(structuredClone(found) as GuiNode),
      position: { x: 0, y: 0 },
    };
    return { ...win, name: "upgrades_before_agency", node } as GuiWindow;
  }, [panel, win]);
  if (!data.data) return null;
  const agency = data.data;
  if (panel) {
    const branches = Array.isArray(agency.branches) ? agency.branches : [];
    const agencySprites = agency.sprites ?? {};
    const grid = panelAt["agency_branches/upgrades_list"];
    const veilAt = panelAt[BRANCH_VEIL];
    const section = panelAt["agency_branches"];

    const veilHeight =
      veilAt && section
        ? Math.min(veilAt.h, section.y + section.h - veilAt.y)
        : 0;
    return (
      <div
        style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
      >
        {branchEntry.data &&
          grid &&
          branches.map((branch, at) => {
            const left = grid.x + BRANCH_OFFSET.x + at * BRANCH_SLOT.w;
            const top = grid.y + BRANCH_OFFSET.y;
            return (
              <div key={branch.id}>
                <Filled
                  entry={branchEntry.data as GuiWindow}
                  left={left}
                  top={top}
                  width={BRANCH_SLOT.w}
                  height={BRANCH_SLOT.h}
                  texts={{ agency_branches_title: branch.label }}
                  hide={["researched_icons"]}
                  extraSprites={{}}
                />
                {agency.created &&
                  onOpenUpgrades &&
                  hit(
                    {
                      x: left,
                      y: top,
                      w: BRANCH_SLOT.w,
                      h: BRANCH_SLOT.h - 10,
                    },
                    `branch ${branch.label}`,
                    onOpenUpgrades,
                  )}
                {agency.created &&
                  pipEntry.data &&
                  (Array.isArray(branch.upgrades) ? branch.upgrades : []).map(
                    (upgrade, index) => (
                      <Filled
                        key={upgrade.id}
                        entry={pipEntry.data as GuiWindow}
                        left={
                          left +
                          PIPS_AT.x +
                          PIP_OFFSET.x +
                          (index % PIPS_PER_ROW) * PIP_SLOT.w
                        }
                        top={
                          top +
                          PIPS_AT.y +
                          PIP_OFFSET.y +
                          Math.floor(index / PIPS_PER_ROW) * PIP_SLOT.h
                        }
                        width={21}
                        height={21}
                        texts={{}}
                        hide={["agency_upgrade_anim"]}
                        extraSprites={agencySprites}
                      />
                    ),
                  )}
              </div>
            );
          })}
        {!agency.created && veil && veilAt && veilHeight > 0 && (
          <Filled
            entry={veil}
            left={veilAt.x}
            top={veilAt.y}
            width={veilAt.w}
            height={veilHeight}
            texts={{}}
            extraSprites={{}}
          />
        )}
        {!agency.created &&
          !forming &&
          hit(
            panelAt["before_creation_panel/creation_button"],
            "Create Agency",
            onOpenInsignia,
            { w: 500, h: 80 },
          )}
        {!agency.created &&
          forming &&
          hit(
            panelAt["before_creation_panel/cancel_creation_button"],
            "Cancel agency creation",
            onCancel,
            { w: 24, h: 24 },
          )}
      </div>
    );
  }
  if (!chooser) return null;
  const grid = windowAt["logo_list/item_grid"];
  const nameBox = windowAt["agency_name/agency_name_entry"];
  const choice = intelChoice(agency, pick);
  const insignia = Array.isArray(agency.insignia) ? agency.insignia : [];
  const agencySprites = agency.sprites ?? {};
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {logoEntry.data &&
        grid &&
        insignia.map((item, at) => {
          const left =
            grid.x + LOGO_MARGIN_LEFT + (at % LOGOS_PER_ROW) * LOGO_SLOT.w;
          const top = grid.y + Math.floor(at / LOGOS_PER_ROW) * LOGO_SLOT.h;
          const label =
            (Array.isArray(item.names) ? item.names[0] : undefined) ??
            item.picture;
          return (
            <div key={`${item.file}#${item.ordinal}`}>
              <Filled
                entry={logoEntry.data as GuiWindow}
                left={left}
                top={top}
                width={LOGO_SLOT.w}
                height={LOGO_SLOT.h}
                texts={{}}
                pictures={{ button: item.picture }}
                frames={
                  at === choice.insignia ? logoFrames.chosen : logoFrames.plain
                }
                extraSprites={agencySprites}
              />
              {onEditInsignia ? (
                <FieldHit
                  x={left}
                  y={top}
                  w={LOGO_SLOT.w}
                  h={LOGO_SLOT.h}
                  label={tLayer("insignia", { name: label })}
                  onClick={() => {
                    onPick({ ...pick, insignia: at });
                    onEditInsignia(item);
                  }}
                />
              ) : (
                hit(
                  { x: left, y: top, w: LOGO_SLOT.w, h: LOGO_SLOT.h },
                  `insignia ${label}`,
                  () => onPick({ ...pick, insignia: at }),
                )
              )}
            </div>
          );
        })}
      {nameBox && (
        <input
          aria-label="Agency name"
          value={choice.name}
          maxLength={40}
          onChange={(e) => onPick({ ...pick, name: e.target.value })}
          onKeyDown={(e) => {
            if (e.key === "Enter") onCreate();
          }}
          style={{
            position: "absolute",
            left: nameBox.x,
            top: nameBox.y,
            width: nameBox.w || 258,
            height: nameBox.h || 23,
            background: "transparent",
            border: "none",
            outline: "none",
            color: "transparent",
            caretColor: "white",
            pointerEvents: "auto",
          }}
        />
      )}
      {hit(windowAt["create_button"], "Create Agency", onCreate, {
        w: 221,
        h: 34,
      })}
      {hit(windowAt["btn_close"], "Close", onClose, { w: 24, h: 24 })}
    </div>
  );
}
