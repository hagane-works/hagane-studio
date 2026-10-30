"use client";

import { useMemo } from "react";
import useSWR from "swr";
import { guiArray, guiRecord, readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { spriteUrl } from "../gui-renderer/GameTooltip";
import { useTemplateWindow } from "./CountrySelectLayer";
import { Filled } from "./FilledWindow";
import type { TextSource } from "./editModel";
import { matchNamePath } from "./gameNavigation";
import { pieSlicePaths } from "./politicsModel";
import { SpriteBar, type BarSprite } from "./SpriteBar";
import { keySource } from "./textSources";

export const FACTION_VIEW = "countryfactionview";
export const FACTION_TABS = [
  "goals",
  "rules",
  "research",
  "army",
  "intelligence",
] as const;
export type FactionTab = (typeof FACTION_TABS)[number];
const TERMS = [
  { term: "short_term", grid: "SHORT_TERM" },
  { term: "medium_term", grid: "MEDIUM_TERM" },
  { term: "long_term", grid: "LONG_TERM" },
] as const;
const GOAL_ITEM = "goal_item_container";
const MEMBER_ITEM = "faction_influence_item";
const SPYMASTER_SLOT = "intelligence_advisor_slot_spymaster";
const BRANCH_SLOT = "intelligence_advisor_slot";

const INTEL_SLOT = { w: 525, h: 142 };
const VACANT_HIDE = [
  "operative_name",
  "operative_skill",
  "take_over_button",
  "small_flag_frame_thin",
  "flag",
  "box_list",
] as const;
const VACANT_PORTRAIT = "vacant_portrait";

const CANNOT_TAKE = ["unlock_button", VACANT_PORTRAIT] as const;

const GOAL = { w: 174, h: 84 };
const MEMBER = { w: 50, h: 80, slot: 55 };

const MANIFEST_BAR = { w: 256, h: 30 };

const PIE_OVERLAY = "GFX_pol_piechart_overlay_63x63";

const PIE_RATIO = 27 / 63;

export interface GuiFactionGoal {
  id: string;
  name: string;
  icon?: string;
}

export interface GuiFactionMember {
  tag: string;
  name: string;
  share: number;
  flag?: { url?: string; vanilla_url?: string };
  color?: [number, number, number];
}

export interface GuiFactionIntel {
  sharing_bonus?: string;
  vacant?: string;
  spymaster_button?: string;
  spymaster_can_take?: boolean;
  branches?: { header: string; button: string; can_take?: boolean }[];

  portrait?: string;
}

export interface GuiFaction {
  tag: string;
  in_faction: boolean;
  name?: string;

  name_key?: string;
  manifest_key?: string;
  icon?: string;
  leader?: string;
  manifest?: string;
  manifest_progress?: number;
  initiative?: number;
  goals?: Record<string, GuiFactionGoal | null>;
  members?: GuiFactionMember[];
  rank?: number;
  rank_text?: string;
  intel?: GuiFactionIntel;
  sprites?: Record<string, GuiSprite & { texture_urls?: string[] }>;
}

export function useFaction(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useSWR<GuiFaction>(
    tag ? ["game-view-faction", projectId, tag, date ?? ""] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/faction?${query}`,
        ),
      );
      return readGuiResponse<GuiFaction>(response);
    },
    { revalidateOnFocus: false },
  );
}

export function factionTabOf(
  name: string | null | undefined,
): FactionTab | null {
  const hit = /^(goals|rules|research|army|intelligence)_tab_button$/.exec(
    name ?? "",
  );
  return hit ? (hit[1] as FactionTab) : null;
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function applyMargins(node: GuiNode): void {
  const margin = node.raw?.margin as
    { top?: number | string; left?: number | string } | undefined;
  const top = Number(margin?.top) || 0;
  const left = Number(margin?.left) || 0;
  for (const child of node.children ?? []) {
    const anchored = (child.orientation ?? "upper_left").toLowerCase();
    if ((top || left) && anchored.startsWith("upper")) {
      const y = Number(child.position?.y);
      const x = Number(child.position?.x);
      child.position = {
        x: Number.isFinite(x) && left ? x + left : child.position.x,
        y: Number.isFinite(y) && top ? y + top : child.position.y,
      };
    }
    applyMargins(child);
  }
}

export function factionWindow(
  win: GuiWindow,
  data: GuiFaction | undefined,
  tab: FactionTab,
): {
  win: GuiWindow;
  hidden: string[];
  texts: Record<string, string>;
  sources: Record<string, TextSource>;
} {
  const node = structuredClone(win.node) as GuiNode;
  applyMargins(node);
  for (const each of FACTION_TABS) {
    const frame = each === tab ? 1 : 2;
    for (const button of matchNamePath(node, `tabs/${each}_tab_button`))
      button.frame = frame;
    for (const icon of matchNamePath(node, `tabs/${each}_tab_button_icon`))
      icon.frame = frame;
  }
  if (data?.icon)
    for (const emblem of matchNamePath(node, "header_section/faction_icon"))
      emblem.sprite = data.icon;
  assignIds(node);
  const hidden: string[] = [];
  const hide = (path: string) => {
    for (const hit of matchNamePath(node, path))
      if (idOf(hit)) hidden.push(idOf(hit));
  };
  for (const each of FACTION_TABS)
    if (each !== tab) hide(`tab_content_parent_view/${each}_section`);

  hide("header_section/manifest_progress");
  const texts: Record<string, string> = {};
  const sources: Record<string, TextSource> = {};

  const write = (path: string, text: string | undefined, key?: string) => {
    if (text === undefined) return;
    const source = keySource(key, "everywhere");
    for (const hit of matchNamePath(node, path)) {
      if (!idOf(hit)) continue;
      texts[idOf(hit)] = text;
      if (source) sources[idOf(hit)] = source;
    }
  };
  if (data?.in_faction) {
    write("header_section/faction_name", data.name, data.name_key);

    write(
      "header_section/manifest_name",
      data.manifest ? `§H${data.manifest}§!` : "",
      data.manifest_key,
    );
    write("header_section/power_projection_value", data.rank_text);
    write(
      "faction_initiative/faction_initiative_value",
      String(data.initiative ?? 0),
    );
    write(
      "intelligence_sizer/intelligence_sharing_bonus",
      data.intel?.sharing_bonus,
    );

    if (tab === "intelligence") hide("intelligence_sizer/opacity_window");
  } else if (data) {
    write("header_section/faction_name", "");
    write("header_section/manifest_name", "");
    write("header_section/power_projection_value", "");
    write("faction_initiative/faction_initiative_value", "0");
  }
  const sprites = { ...win.sprites, ...guiRecord(data?.sprites) };
  return { win: { ...win, node, sprites }, hidden, texts, sources };
}

type Rect = { x: number; y: number; w: number; h: number };

export function goalCard(entry: GuiWindow): GuiWindow {
  const node = structuredClone(entry.node) as GuiNode;
  for (const label of matchNamePath(
    node,
    "assigned_goal_item_container/label",
  )) {
    const box = label.maxheight || 60;
    const y = Number(label.position?.y) || 0;
    label.position = { ...label.position, y: y - box / 2 };
    label.raw = { ...(label.raw ?? {}), vertical_alignment: "center" };
  }
  return { ...entry, node };
}

export function FactionLayer({
  win,
  projectId,
  tag,
  date,
  tab,
  onOpenCountry,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  tab: FactionTab;
  onOpenCountry?: (tag: string) => void;
}) {
  const active = win.name === FACTION_VIEW;
  const data = useFaction(projectId, active ? tag : null, date);
  const goalItem = useTemplateWindow(projectId, active ? GOAL_ITEM : null);
  const goalEntry = useMemo(
    () => (goalItem.data ? goalCard(goalItem.data) : null),
    [goalItem.data],
  );
  const memberItem = useTemplateWindow(projectId, active ? MEMBER_ITEM : null);
  const intelTab = active && tab === "intelligence";
  const spymasterItem = useTemplateWindow(
    projectId,
    intelTab ? SPYMASTER_SLOT : null,
  );
  const branchItem = useTemplateWindow(
    projectId,
    intelTab ? BRANCH_SLOT : null,
  );
  const portrait = data.data?.intel?.portrait;
  const spymasterEntry = useMemo(
    () =>
      spymasterItem.data ? nestedSlot(spymasterItem.data, portrait) : null,
    [spymasterItem.data, portrait],
  );
  const branchEntry = useMemo(
    () => (branchItem.data ? nestedSlot(branchItem.data, portrait) : null),
    [branchItem.data, portrait],
  );
  const at = useMemo(() => {
    if (!active) return null;
    const { win: shown } = factionWindow(win, data.data, tab);
    const rects = measureWindow(shown.node, shown.sprites, 1920, 1080);
    const rect = (path: string) =>
      rects[idOf(matchNamePath(shown.node, path)[0])] as Rect | undefined;
    return {
      bar: rect("header_section/manifest_progress"),
      members: rect("chart_explanation/countries_grid"),
      grids: Object.fromEntries(
        TERMS.map(({ term, grid }) => [term, rect(`goals_container/${grid}`)]),
      ) as Record<string, Rect | undefined>,
      pie: rect("influence_pie_chart/pol_view_bg"),
      intel: {
        section: rect("tab_content_parent_view/intelligence_section"),
        slots: rect("intelligence_sizer/intelligence_slots"),
        shade: rect("intelligence_sizer/opacity_window"),
      },
    };
  }, [active, win, data.data, tab]);

  const memberParts = useMemo(() => {
    if (!memberItem.data) return null;
    const node = structuredClone(memberItem.data.node) as GuiNode;
    assignIds(node);
    const rects = measureWindow(
      node,
      memberItem.data.sprites,
      MEMBER.w,
      MEMBER.h,
    );
    const rect = (name: string) =>
      rects[idOf(matchNamePath(node, name)[0])] as Rect | undefined;
    return { flag: rect("flag"), strip: rect("country_color") };
  }, [memberItem.data]);

  const shade = useMemo((): GuiWindow | null => {
    if (!intelTab) return null;
    const node = matchNamePath(
      win.node,
      "intelligence_sizer/opacity_window",
    )[0];
    if (!node) return null;
    const own = structuredClone(node) as GuiNode;
    own.position = { x: 0, y: 0 };
    return { ...win, node: own };
  }, [intelTab, win]);
  if (!active || !data.data?.in_faction || !at) return null;
  const faction = data.data;
  const factionSprites = guiRecord(faction.sprites);
  const members = guiArray(faction.members);
  const goals = guiRecord(faction.goals);
  const barSprite = (factionSprites["GFX_faction_manifest_progress_bar"] ??
    win.sprites["GFX_faction_manifest_progress_bar"]) as BarSprite | undefined;

  const overlaySprite = win.sprites[PIE_OVERLAY];
  const slices = at.pie
    ? pieSlicePaths(
        members.map((member) => ({
          party: member.tag,
          value: member.share,
          color: member.color ?? [128, 128, 128],
        })),
        at.pie.x + at.pie.w / 2,
        at.pie.y + at.pie.h / 2,
        at.pie.w * PIE_RATIO,
      )
    : [];
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {at.pie && (
        <>
          <svg
            width={1}
            height={1}
            style={{
              position: "absolute",
              left: 0,
              top: 0,
              overflow: "visible",
            }}
            aria-hidden
          >
            {slices.map((slice) => (
              <path key={slice.party} d={slice.d} fill={slice.fill} />
            ))}
          </svg>
          {overlaySprite?.resolved && (
            <div
              style={{
                position: "absolute",
                left: at.pie.x,
                top: at.pie.y,
                width: at.pie.w,
                height: at.pie.h,
                backgroundImage: `url("${spriteUrl(PIE_OVERLAY, overlaySprite)}")`,
                backgroundSize: "100% 100%",
                backgroundRepeat: "no-repeat",
              }}
            />
          )}
        </>
      )}
      {at.bar && (
        <SpriteBar
          x={at.bar.x}
          y={at.bar.y}
          size={MANIFEST_BAR}
          sprite={barSprite}
          fraction={faction.manifest_progress ?? 0}
        />
      )}
      {memberItem.data &&
        at.members &&
        members.slice(0, 5).map((member, k) => {
          const flag = member.flag?.vanilla_url || member.flag?.url;
          const left = (at.members as Rect).x + k * MEMBER.slot;
          const top = (at.members as Rect).y;
          return (
            <div
              key={member.tag}
              aria-label={`member ${member.tag}`}
              style={{
                position: "absolute",
                left,
                top,
                width: MEMBER.w,
                height: MEMBER.h,
              }}
            >
              <Filled
                entry={memberItem.data as GuiWindow}
                left={0}
                top={0}
                width={MEMBER.w}
                height={MEMBER.h}
                texts={{ ratio: `(${member.share}%)` }}
                hide={["flag", "country_color"]}
                extraSprites={{}}
              />
              {flag && memberParts?.flag && (
                <div
                  style={{
                    position: "absolute",
                    left: memberParts.flag.x,
                    top: memberParts.flag.y,
                    width: memberParts.flag.w || 36,
                    height: memberParts.flag.h || 24,
                    backgroundImage: `url("${flag}")`,
                    backgroundSize: "100% 100%",
                  }}
                />
              )}
              {memberParts?.flag && onOpenCountry && (
                <button
                  type="button"
                  aria-label={"open faction member " + member.tag}
                  onClick={() => onOpenCountry(member.tag)}
                  style={{
                    position: "absolute",
                    left: memberParts.flag.x,
                    top: memberParts.flag.y,
                    width: memberParts.flag.w || 36,
                    height: memberParts.flag.h || 24,
                    padding: 0,
                    border: 0,
                    background: "transparent",
                    pointerEvents: "auto",
                  }}
                />
              )}
              {member.color && memberParts?.strip && (
                <div
                  style={{
                    position: "absolute",
                    left: memberParts.strip.x,
                    top: memberParts.strip.y,
                    width: memberParts.strip.w || 8,
                    height: memberParts.strip.h || 8,
                    borderRadius: "50%",
                    background: `rgb(${member.color.join(",")})`,
                  }}
                />
              )}
            </div>
          );
        })}
      {tab === "goals" &&
        goalEntry &&
        TERMS.map(({ term }) => {
          const grid = at.grids[term];
          if (!grid) return null;
          const goal = goals[term] ?? null;
          return (
            <div
              key={term}
              aria-label={`goal ${term}`}
              style={{
                position: "absolute",
                left: grid.x,
                top: grid.y,
                width: GOAL.w,
                height: GOAL.h,
              }}
            >
              <Filled
                entry={goalEntry}
                left={0}
                top={0}
                width={GOAL.w}
                height={GOAL.h}
                texts={goal ? { label: goal.name } : {}}
                pictures={goal?.icon ? { group_icon: goal.icon } : {}}
                hide={
                  goal
                    ? ["empty_goal_item_container", "idea_alert_glow"]
                    : ["assigned_goal_item_container"]
                }
                extraSprites={factionSprites}
              />
            </div>
          );
        })}
      {tab === "intelligence" && at.intel.section && at.intel.slots && (
        <IntelPositions
          clip={at.intel.section}
          grid={at.intel.slots}
          shadeAt={at.intel.shade}
          intel={faction.intel ?? {}}
          spymaster={spymasterEntry}
          branch={branchEntry}
          shade={shade}
          sprites={factionSprites}
        />
      )}
    </div>
  );
}

export function nestedSlot(entry: GuiWindow, portrait?: string): GuiWindow {
  const inner = structuredClone(entry.node) as GuiNode;
  const list = inner.children.findIndex((child) => child.name === "box_list");
  if (portrait && list >= 0) {
    const card: GuiNode = {
      ...inner.children[list],
      name: VACANT_PORTRAIT,
      type: "icontype",
      sprite: portrait,
      sprite_key: "spriteType",
      size: null,
      children: [],
      raw: {},
    } as GuiNode;
    inner.children.splice(list + 1, 0, card);
  }
  const node: GuiNode = {
    ...inner,
    name: `${inner.name}_slot`,
    position: { x: 0, y: 0 },
    background: null,
    children: [inner],
    raw: {},
  };
  assignIds(node);
  return { ...entry, node };
}

function IntelPositions({
  clip,
  grid,
  shadeAt,
  intel,
  spymaster,
  branch,
  shade,
  sprites,
}: {
  clip: Rect;
  grid: Rect;
  shadeAt: Rect | undefined;
  intel: GuiFactionIntel;
  spymaster: GuiWindow | null;
  branch: GuiWindow | null;
  shade: GuiWindow | null;
  sprites: Record<string, GuiSprite>;
}) {
  const slots: {
    key: string;
    entry: GuiWindow | null;
    texts: Record<string, string>;
    canTake: boolean;
  }[] = [
    {
      key: "spymaster",
      entry: spymaster,
      texts: {
        operative_title: intel.vacant ?? "",
        unlock_button: intel.spymaster_button ?? "",
      },
      canTake: Boolean(intel.spymaster_can_take),
    },
    ...guiArray(intel.branches).map((each) => ({
      key: each.header,
      entry: branch,
      texts: {
        header_text: each.header,
        operative_title: intel.vacant ?? "",
        unlock_button: each.button,
      },
      canTake: Boolean(each.can_take),
    })),
  ];
  return (
    <div
      aria-label="intelligence positions"
      style={{
        position: "absolute",
        left: clip.x,
        top: clip.y,
        width: clip.w,
        height: clip.h,
        overflow: "hidden",
      }}
    >
      {slots.map(({ key, entry, texts, canTake }, k) =>
        entry ? (
          <div
            key={key}
            style={{
              position: "absolute",
              left: grid.x - clip.x,
              top: grid.y - clip.y + k * INTEL_SLOT.h,
              width: INTEL_SLOT.w,
              height: INTEL_SLOT.h,
            }}
          >
            <Filled
              entry={entry}
              left={0}
              top={0}
              width={INTEL_SLOT.w}
              height={INTEL_SLOT.h}
              texts={texts}
              hide={VACANT_HIDE}
              disabled={canTake ? undefined : CANNOT_TAKE}
              extraSprites={sprites}
            />
          </div>
        ) : null,
      )}
      {shade && shadeAt && (
        <Filled
          entry={shade}
          left={shadeAt.x - clip.x}
          top={shadeAt.y - clip.y}
          width={shadeAt.w}
          height={shadeAt.h}
          texts={{}}
          extraSprites={{}}
        />
      )}
    </div>
  );
}
