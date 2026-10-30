"use client";

import { useMemo } from "react";
import useSWR from "swr";
import { guiArray, readGuiResponse } from "./guiResponse";
import {
  guiRenderApi,
  type GuiCountryStats,
  type GuiNode,
  type GuiPreviewIcon,
  type GuiSprite,
  type GuiWindow,
  withGameLanguage,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { gameTooltipProps } from "../gui-renderer/GameTooltip";
import { useEditCanvas, usePrintedOver } from "./editContext";
import type { PrintedText } from "./editModel";
import { countryRowPrinted, diplomacyHeaderPrinted } from "./textSources";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useCharacters } from "./CharacterLayer";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";
import { pieSlicePaths } from "./politicsModel";
import { spriteUrlOf } from "./spriteFrame";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export const DIPLOMACY = "countrydiplomacyview";

const ROW = "diplomacy_country_list_country_entry";

const ACTION = "diplomacy_action_entry";
const ACTION_W = 228;
const ACTION_H = 28;
const ACTION_SLOT = 29;

const SPIRIT_SLOT = 60;
const SPIRIT_INSET = 9;

const BIG_PIE_RADIUS = 30;
const PIE_OVERLAY = "GFX_pol_piechart_overlay";

const TARGET_HIDES = [
  "country_list",
  "player_info_box",
  "ideas_info",
  "trade_info",
  "estimated_enemy_force_info",
  "intel_ledger_container",
];

const ROW_H = 45;
const ROW_W = 500;

const IDEOLOGY_ICON = (party: string) => `GFX_ideology_${party}_group`;

const SCREEN_H = 1080;

const NOTHING_HIDDEN = new Set<string>();
const NO_PRINTED: Readonly<Record<string, PrintedText>> = {};

export interface GuiDiplomacyCountry {
  tag: string;
  name?: string;

  name_key?: string;
  ideology?: string;
  faction?: string;
  faction_icon?: string;
  faction_leader?: string;
  overlord?: string;
  flag?: { url?: string; vanilla_url?: string };
}

export interface GuiDiplomacy {
  tag: string;
  self: GuiDiplomacyCountry & {
    party?: string;
    party_label?: string;

    party_name?: string;
    elections?: boolean;
  };
  leader?: string;

  leader_token?: string;
  countries: GuiDiplomacyCountry[];

  autonomy_states?: { id: string; label?: string }[];

  actions?: GuiDiplomacyAction[];
  texts?: {
    intel_ledger?: string;
    unknown_focus?: string;
    select_focus?: string;
    no_faction?: string;
  };
}

export interface GuiDiplomacyAction {
  id: string;
  label: string;

  pp?: number;

  cp?: number;
}

export function actionCostText(action: GuiDiplomacyAction): string {
  const parts: string[] = [];
  if (action.cp) parts.push(`£command_power ${Math.round(action.cp)}`);
  if (action.pp) parts.push(`£pol_power ${Math.round(action.pp)}`);
  return parts.join(" ");
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

const LIST_HIDES = [
  "country_info/diplomacy_bottom",
  "country_info/back_button",
  "country_info/logistics_button",
];

export function diplomacyListHidden(win: GuiWindow): string[] {
  return LIST_HIDES.flatMap((path) =>
    matchNamePath(win.node, path).map(idOf).filter(Boolean),
  );
}

export function diplomacyTargetWindow(
  win: GuiWindow,
  target: GuiDiplomacy | undefined,
): { win: GuiWindow; hidden: string[]; texts: Record<string, string> } {
  const node = structuredClone(win.node) as GuiNode;
  assignIds(node);
  const ids = (path: string) =>
    matchNamePath(node, path).map(idOf).filter(Boolean);
  const hidden = TARGET_HIDES.flatMap(ids);
  const texts: Record<string, string> = {};
  const write = (path: string, text: string | undefined) => {
    if (text) for (const id of ids(path)) texts[id] = text;
  };
  write("info_tab_button", target?.texts?.intel_ledger);

  write(
    "national_focus_label",
    target?.texts?.select_focus || target?.texts?.unknown_focus,
  );
  return { win: { ...win, node }, hidden, texts };
}

export function useDiplomacy(
  projectId: number,
  tag: string | null,
  date: string | null,
) {
  return useSWR<GuiDiplomacy>(
    tag ? ["game-view-diplomacy", projectId, tag, date] : null,
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/diplomacy?tag=${encodeURIComponent(tag as string)}`,
        ) + (date ? `&date=${encodeURIComponent(date)}` : ""),
      );
      return readGuiResponse<GuiDiplomacy>(response);
    },
    { revalidateOnFocus: false },
  );
}

function CountryRow({
  entry,
  country,
  left,
  top,
  onOpen,
}: {
  entry: GuiWindow;
  country: GuiDiplomacyCountry;
  left: number;
  top: number;
  onOpen?: () => void;
}) {
  const edit = useEditCanvas();
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;
    assignIds(node);
    const texts: Record<string, string> = {};
    for (const child of (node.children ?? []) as (GuiNode & {
      _id?: string;
    })[]) {
      if (!child._id || !child.name) continue;
      if (
        child.name === "name" ||
        child.name === "our_opinion" ||
        child.name === "their_opinion"
      ) {
        texts[child._id] = "";
      }
    }
    return { node, texts };
  }, [entry.node]);

  const printed = useMemo(
    () => countryRowPrinted(built.node, country),
    [built.node, country],
  );
  const rects = useMemo(
    () => measureWindow(built.node, entry.sprites, 1920, SCREEN_H),
    [built.node, entry.sprites],
  );
  const nameNode = entry.node.children?.find((child) => child.name === "name");
  const font = nameNode?.font ? entry.fonts[nameNode.font]?.bitmap : undefined;
  const nameRect = rects["name:name"];
  const flagRect = rects["name:diplolist_country_flag"];
  const flag = country.flag?.vanilla_url || country.flag?.url;

  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${country.tag}`,
          entry,
          built.node,
          entry.sprites,
          built.texts,
          printed,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NOTHING_HIDDEN}
        isolateId={null}
        transparent
        baseW={ROW_W}
        baseH={ROW_H}
      />
      {flag && flagRect && (
        <div
          style={{
            position: "absolute",
            left: flagRect.x,
            top: flagRect.y,
            width: flagRect.w || 41,
            height: flagRect.h || 26,
            backgroundImage: `url("${flag}")`,
            backgroundSize: "100% 100%",
            backgroundRepeat: "no-repeat",
          }}
        />
      )}
      {font && nameRect && (
        <BitmapText
          font={font}

          text={
            country.faction
              ? `${country.name || country.tag} · ${country.faction}`
              : country.name || country.tag
          }
          left={nameRect.x}
          top={nameRect.y}
          width={nameNode?.maxwidth ?? 175}
          height={font.line + 4}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
      {onOpen && (
        <div
          role="button"
          aria-label={`country ${country.tag}`}
          onClick={onOpen}
          style={{
            position: "absolute",
            left: 0,
            top: 0,
            width: ROW_W,
            height: ROW_H,
            cursor: "pointer",
          }}
        />
      )}
    </div>
  );
}

export function DiplomacySpirits({
  grid,
  spirits,
}: {
  grid: { x: number; y: number };
  spirits: readonly GuiPreviewIcon[];
}) {
  return (
    <>
      {spirits.slice(0, 6).map((spirit, at) => {
        const url = spirit.icon_sprite?.vanilla_url;
        if (!url) return null;
        const frames = Math.max(1, spirit.icon_sprite?.frames || 1);
        const width = spirit.icon_sprite?.width
          ? Math.round(spirit.icon_sprite.width / frames)
          : 64;
        return (
          <div
            key={spirit.token}
            data-spirit={spirit.token}
            title={spirit.tooltip ? undefined : spirit.label || spirit.token}
            {...(spirit.tooltip
              ? gameTooltipProps(spirit.tooltip, {
                  key: `diplomacy-spirit:${spirit.token}`,
                })
              : {})}
            style={{
              position: "absolute",
              left: grid.x + at * SPIRIT_SLOT + SPIRIT_INSET,
              top: grid.y + SPIRIT_INSET,
              width,
              height: spirit.icon_sprite?.height || 64,
              backgroundImage: `url("${url}")`,
              backgroundSize: `${width * frames}px ${spirit.icon_sprite?.height || 64}px`,
              pointerEvents: "auto",
            }}
          />
        );
      })}
    </>
  );
}

function TargetCountry({
  win,
  rects,
  projectId,
  target,
  date,
  onBack,
  onAction,
}: {
  win: GuiWindow;
  rects: Record<string, { x: number; y: number; w: number; h: number }>;
  projectId: number;
  target: string;
  date?: string | null;
  onBack: () => void;
  onAction?: (id: string) => void;
}) {
  const diplomacy = useDiplomacy(projectId, target, date ?? null);
  const stats = useSWR<GuiCountryStats>(
    ["game-view-stats", projectId, target, date ?? null],
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.countryStats(target, date || undefined);
    },
    { revalidateOnFocus: false },
  );
  const people = useCharacters(projectId, target, date ?? null);
  const action = useTemplateWindow(projectId, ACTION);
  const self = diplomacy.data?.self;
  const iconName = self?.ideology ? IDEOLOGY_ICON(self.ideology) : "";
  const icons = useSWR<Record<string, GuiSprite>>(
    iconName
      ? ["game-view-sprites", projectId, "diplomacy-icons", iconName]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites([iconName]);
    },
    { revalidateOnFocus: false },
  );

  const printed = useMemo(
    () =>
      diplomacyHeaderPrinted(win, self, {
        name: diplomacy.data?.leader,
        token: diplomacy.data?.leader_token,
      }),
    [win, self, diplomacy.data?.leader, diplomacy.data?.leader_token],
  );
  usePrintedOver(win, "diplomacy-target", printed);
  if (!self) return null;
  const slotFont = (name: string) => {
    const found = matchNode(win.node, name);
    return found?.font ? win.fonts[found.font]?.bitmap : undefined;
  };
  const line = (slot: string, text: string | undefined, width = 200) => {
    const rect = rects[`name:${slot}`];
    const font = slotFont(slot);
    if (!rect || !font || !text) return null;
    return (
      <BitmapText
        key={slot}
        font={font}
        text={text}
        left={rect.x}
        top={rect.y}
        width={width}
        height={font.line + 4}
        align="left"
        vcenter={false}
        maxLines={1}
      />
    );
  };
  const percent = (value: number | undefined) =>
    value === undefined ? "" : `${Math.round(value * 100)}%`;

  const flag =
    stats.data?.flag?.url || self.flag?.vanilla_url || self.flag?.url;
  const frameRect = rects["name:diplo_flag_frame"];
  const badge = iconName ? icons.data?.[iconName] : undefined;
  const badgeRect = rects["name:ideology_icon"];
  const leader = people.data?.leader;
  const portraitUrl = leader?.portrait_sprite
    ? spriteUrlOf(leader.portrait_sprite, leader.portrait)
    : undefined;
  const portraitRect = rects["name:leader_portrait"];
  const overlay = rects["name:pol_view_bg"];
  const pie = stats.data?.popularities ?? [];
  const slices = overlay
    ? pieSlicePaths(
        pie,
        overlay.x + overlay.w / 2,
        overlay.y + overlay.h / 2,
        BIG_PIE_RADIUS,
      )
    : [];
  const overlaySprite = win.sprites[PIE_OVERLAY];
  const overlayUrl = overlaySprite
    ? spriteUrlOf(overlaySprite, PIE_OVERLAY)
    : undefined;
  const spirits = rects["name:national_spirit_ideas_grid"];
  const actions = rects["name:actions_grid"];
  const back = rects["name:back_button"];
  const actionList = diplomacy.data?.actions ?? [];
  return (
    <>
      {line("country_name", self.name || self.tag, 260)}
      {line(
        "faction_name",
        self.faction || diplomacy.data?.texts?.no_faction,
        260,
      )}
      {line("leader_name", diplomacy.data?.leader, 260)}
      {line("stability_value", percent(stats.data?.stability), 60)}
      {line("war_support_value", percent(stats.data?.war_support), 60)}
      {line("party_name", self.party_name || self.party_label, 240)}
      {line("ideology_name", stats.data?.politics?.ideology, 240)}
      {line("elections", stats.data?.politics?.elections, 240)}
      {flag && frameRect && (
        <div
          style={{
            position: "absolute",
            left: frameRect.x + 5,
            top: frameRect.y + 4,
            width: (frameRect.w || 128) - 10,
            height: (frameRect.h || 80) - 8,
            backgroundImage: `url("${flag}")`,
            backgroundSize: "100% 100%",
          }}
        />
      )}
      {badge?.vanilla_url && badgeRect && (
        <div
          style={{
            position: "absolute",
            left: badgeRect.x,
            top: badgeRect.y,
            width: badgeRect.w || badge.width,
            height: badgeRect.h || badge.height,
            backgroundImage: `url("${badge.vanilla_url}")`,
            backgroundSize: "100% 100%",
          }}
        />
      )}
      {portraitUrl && portraitRect && (
        <div
          title={leader?.name}
          style={{
            position: "absolute",
            left: portraitRect.x,
            top: portraitRect.y,
            width: portraitRect.w,
            height: portraitRect.h,
            backgroundImage: `url("${portraitUrl}")`,
            backgroundSize: "100% 100%",
          }}
        />
      )}
      {slices.length > 0 && (
        <svg
          width={1}
          height={1}
          style={{ position: "absolute", left: 0, top: 0, overflow: "visible" }}
          aria-hidden
        >
          {slices.map((slice) => (
            <path key={slice.party} d={slice.d} fill={slice.fill} />
          ))}
        </svg>
      )}
      {slices.length > 0 && overlay && overlayUrl && (
        <div
          style={{
            position: "absolute",
            left: overlay.x,
            top: overlay.y,
            width: overlay.w,
            height: overlay.h,
            backgroundImage: `url("${overlayUrl}")`,
            backgroundSize: "100% 100%",
          }}
        />
      )}
      {spirits && (
        <DiplomacySpirits grid={spirits} spirits={people.data?.spirits ?? []} />
      )}
      {action.data &&
        actions &&
        actionList.map((entry, at) => (
          <Filled
            key={entry.id}
            entry={action.data as GuiWindow}
            left={actions.x}
            top={actions.y + at * ACTION_SLOT}
            width={ACTION_W}
            height={ACTION_H}
            texts={{ name: entry.label, cost: actionCostText(entry) }}
            hide={["accept_icon"]}
            extraSprites={{}}
          />
        ))}
      {actions &&
        onAction &&
        actionList.map((entry, at) => (
          <div
            key={`hit-${entry.id}`}
            role="button"
            aria-label={`action ${entry.id}`}
            onClick={() => onAction(entry.id)}
            style={{
              position: "absolute",
              left: actions.x,
              top: actions.y + at * ACTION_SLOT,
              width: ACTION_W,
              height: ACTION_H,
              cursor: "pointer",
              pointerEvents: "auto",
            }}
          />
        ))}
      {back && (
        <div
          role="button"
          aria-label="Open Country List"
          onClick={onBack}
          style={{
            position: "absolute",
            left: back.x,
            top: back.y,
            width: back.w || 240,
            height: back.h || 34,
            cursor: "pointer",
            pointerEvents: "auto",
          }}
        />
      )}
    </>
  );
}

export function DiplomacyLayer({
  win,
  projectId,
  tag,
  date,
  stats,
  target = null,
  onTarget,
  onAction,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  stats?: GuiCountryStats;

  target?: string | null;
  onTarget?: (tag: string | null) => void;

  onAction?: (id: string) => void;
}) {
  const active = win.name === DIPLOMACY;
  const diplomacy = useDiplomacy(projectId, active ? tag : null, date ?? null);
  const row = useTemplateWindow(projectId, active ? ROW : null);

  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, SCREEN_H) : {}),
    [active, win],
  );
  const self = diplomacy.data?.self;
  const iconName = self?.ideology ? IDEOLOGY_ICON(self.ideology) : "";
  const icons = useSWR<Record<string, GuiSprite>>(
    iconName
      ? ["game-view-sprites", projectId, "diplomacy-icons", iconName]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites([iconName]);
    },
    { revalidateOnFocus: false },
  );
  const fonts = win.fonts;
  const slotFont = (name: string) => {
    const found = matchNode(win.node, name);
    return found?.font ? fonts[found.font]?.bitmap : undefined;
  };

  const ownHeader = active && !(target && target !== tag);
  const printed = useMemo(
    () =>
      ownHeader
        ? diplomacyHeaderPrinted(win, self, {
            name: diplomacy.data?.leader,
            token: diplomacy.data?.leader_token,
          })
        : NO_PRINTED,
    [
      ownHeader,
      win,
      self,
      diplomacy.data?.leader,
      diplomacy.data?.leader_token,
    ],
  );
  usePrintedOver(win, "diplomacy", printed);

  if (!active || !diplomacy.data || !self) return null;
  if (target && target !== tag) {
    return (
      <div
        style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
      >
        <TargetCountry
          win={win}
          rects={rects}
          projectId={projectId}
          target={target}
          date={date}
          onBack={() => onTarget?.(null)}
          onAction={onAction}
        />
      </div>
    );
  }
  const grid = rects["name:countries_grid"];

  const listBox = rects["name:countries"];
  const listMargin =
    Number(
      (
        (matchNamePath(win.node, "countries")[0]?.raw?.margin ?? {}) as {
          bottom?: number | string;
        }
      ).bottom,
    ) || 0;
  const listBottom = listBox
    ? listBox.y + listBox.h - listMargin
    : SCREEN_H - 20;
  const countries = guiArray(diplomacy.data.countries);
  const badge = iconName ? icons.data?.[iconName] : undefined;
  const badgeRect = rects["name:ideology_icon"];
  const frameRect = rects["name:diplo_flag_frame"];

  const selfFlag = stats?.flag?.url || self.flag?.vanilla_url || self.flag?.url;

  const line = (slot: string, text: string, width = 200) => {
    const rect = rects[`name:${slot}`];
    const font = slotFont(slot);
    if (!rect || !font || !text) return null;
    return (
      <BitmapText
        font={font}
        text={text}
        left={rect.x}
        top={rect.y}
        width={width}
        height={font.line + 4}
        align="left"
        vcenter={false}
        maxLines={1}
      />
    );
  };
  const percent = (value: number | undefined) =>
    value === undefined ? "" : `${Math.round(value * 100)}%`;

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {line("country_name", self.name || self.tag, 260)}
      {line("faction_name", self.faction ?? "", 260)}
      {line("leader_name", diplomacy.data.leader ?? "", 260)}
      {line("stability_value", percent(stats?.stability), 60)}
      {line("war_support_value", percent(stats?.war_support), 60)}

      {selfFlag && frameRect && (
        <div
          style={{
            position: "absolute",
            left: frameRect.x + 5,
            top: frameRect.y + 4,
            width: (frameRect.w || 128) - 10,
            height: (frameRect.h || 80) - 8,
            backgroundImage: `url("${selfFlag}")`,
            backgroundSize: "100% 100%",
            backgroundRepeat: "no-repeat",
          }}
        />
      )}
      {badge?.vanilla_url && badgeRect && (
        <div
          style={{
            position: "absolute",
            left: badgeRect.x,
            top: badgeRect.y,
            width: badgeRect.w || badge.width,
            height: badgeRect.h || badge.height,
            backgroundImage: `url("${badge.vanilla_url}")`,
            backgroundSize: "100% 100%",
            backgroundRepeat: "no-repeat",
          }}
        />
      )}

      {row.data && grid && (
        <div
          style={{
            position: "absolute",
            left: grid.x,
            top: grid.y,
            width: ROW_W + 20,
            height: Math.max(0, listBottom - grid.y),
            overflowY: "auto",
            overflowX: "hidden",
            ...GAME_SCROLLBAR,
            pointerEvents: "auto",
          }}
        >
          <div
            style={{ position: "relative", height: countries.length * ROW_H }}
          >
            {countries.map((country, at) => (
              <CountryRow
                key={country.tag}
                entry={row.data as GuiWindow}
                country={country}
                left={0}
                top={at * ROW_H}
                onOpen={onTarget ? () => onTarget(country.tag) : undefined}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function matchNode(root: GuiNode, name: string): GuiNode | undefined {
  if (root.name === name) return root;
  for (const child of root.children ?? []) {
    const found = matchNode(child, name);
    if (found) return found;
  }
  return undefined;
}
