"use client";

import { useCallback, useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { guiArray, guiRecord, readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { matchNamePath } from "./gameNavigation";
import { FieldHit } from "./FieldHit";
import { Filled } from "./FilledWindow";
import type { DoctrineChoice } from "./doctrineEdit";
import type { TextSource } from "./editModel";
import { GAME_SCROLLBAR } from "./PageScrollbar";
import { keySource } from "./textSources";

export const DOCTRINE_VIEW = "country_doctrine_view";
const TAB = "doctrine_folder_tab";
const TRACK = "doctrine_track_item";
const PICKER = "doctrine_selection_list";
const ITEM = "doctrine_list_item";
const REWARD = "reward_item";

const REWARD_W = 72;
const REWARD_H = 88;
const BAR = { x: 19, w: 32, h: 72 };
const REWARD_SIDES = {
  right: { medal: { x: 60, y: 20 }, level: { x: 125, y: 64 } },
  left: { medal: { x: -94, y: 20 }, level: { x: -98, y: 64 } },
} as const;
const REWARD_TEXT = { x: 14, y: 68 };

const PICKER_W = 560;
const PICKER_H = 576;
const ITEM_W = 250;
const ITEM_H = 75;

const ITEM_PAD_X = 13;
const ITEM_PAD_Y = 15;

const RIGHT_ANCHOR = { x: 385, y: 210 };
const LEFT_ANCHOR_X = -500;

const GRAND_PICKER = { x: 674, y: 352 };

export const TRACK_W = 424;
export const TRACK_H = 764;

const VIEW_MARGIN = 15;
const VIEW_WHEEL = 70;

export const TRACKS_AREA_W = 1920 - 2 * VIEW_MARGIN;

export function trackGrid(
  count: number,
  width: number,
  grand?: { max_track_columns?: number; max_track_rows?: number },
): { columns: number; rows: number } {
  if (count <= 0) return { columns: 0, rows: 0 };
  let columns = Math.max(1, Math.floor(width / TRACK_W));
  if (grand?.max_track_columns && grand.max_track_columns > 0)
    columns = grand.max_track_columns;
  else if (grand?.max_track_rows && grand.max_track_rows > 0)
    columns = Math.max(columns, Math.ceil(count / grand.max_track_rows));
  columns = Math.min(columns, count);
  return { columns, rows: Math.ceil(count / columns) };
}

export function trackSpot(
  index: number,
  grid: { columns: number },
  areaWidth: number,
): { x: number; y: number } {
  const left = Math.max(0, (areaWidth - grid.columns * TRACK_W) / 2);
  return {
    x: left + (index % grid.columns) * TRACK_W,
    y: Math.floor(index / grid.columns) * TRACK_H,
  };
}

const TAB_W = 256;
const TAB_H = 62;

const TABS_X = 4;

export const DOCTRINE_RINGS: Readonly<Record<string, string>> = {
  land_doctrine_ring: "land",
  naval_doctrine_ring: "naval",
  air_doctrine_ring: "air",
  special_forces_doctrine_ring: "special_forces",
};

export interface GuiDoctrineChoice {
  token: string;
  label: string;
  icon: string;
  xp_cost: number;
  xp_type?: string;
}

export interface GuiDoctrineReward {
  token: string;
  label: string;
  level: number;

  mastery: number;
  unlocked: boolean;

  progress: number;
}

export interface GuiDoctrineTrack {
  token: string;
  label: string;
  background: string;
  icon: string;
  icon_frame: string;
  mastery: number;
  sub?: { token: string; label: string; icon: string };
  choices: GuiDoctrineChoice[];
  rewards?: GuiDoctrineReward[];

  complete?: boolean;
}

export interface GuiDoctrines {
  tag: string;
  folder: string;
  folders: { token: string; label: string; tab: string; color_frame: number }[];
  color_frame: number;
  grand?: {
    token: string;
    label: string;
    icon: string;
    max_track_columns?: number;
    max_track_rows?: number;
  };
  tracks?: GuiDoctrineTrack[];
  texts?: {
    no_grand?: string;
    select_grand?: string;
    select_sub?: string;
    mastery?: string;
    grand_desc?: string;
    sub_desc?: string;
    army_cost?: string;
    navy_cost?: string;
    air_cost?: string;
  };
  grand_choices?: GuiDoctrineChoice[];
  experience?: Record<string, number>;
  sprites: Record<string, GuiSprite>;
}

export function useDoctrines(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
  folder: string | null,
) {
  return useSWR<GuiDoctrines>(
    tag
      ? ["game-view-doctrines", projectId, tag, date ?? "", folder ?? ""]
      : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      if (folder) query.set("folder", folder);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/doctrines?${query}`,
        ),
      );
      return readGuiResponse<GuiDoctrines>(response);
    },

    { revalidateOnFocus: false, keepPreviousData: true },
  );
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function doctrineWindow(
  win: GuiWindow,
  data: GuiDoctrines | undefined,
): {
  win: GuiWindow;
  hidden: string[];
  texts: Record<string, string>;
  sources: Record<string, TextSource>;
} {
  const node = structuredClone(win.node) as GuiNode;
  const tracks = guiArray(data?.tracks);
  const dataSprites = guiRecord(data?.sprites);
  const walk = (at: GuiNode) => {
    const margin =
      at.name === "doctrine_folder_view"
        ? Number((at.raw?.margin as { top?: number } | undefined)?.top) || 0
        : 0;
    for (const child of at.children ?? []) {
      if (margin && child.position)
        child.position = {
          ...child.position,
          y: (Number(child.position.y) || 0) + margin,
        };
      if (child.name === "grand_doctrine_button")
        child.frame = data?.color_frame ?? 1;
      if (
        child.name === "grand_doctrine_icon" &&
        data?.grand?.icon &&
        dataSprites[data.grand.icon]
      )
        child.sprite = data.grand.icon;
      if (child.name === "tracks_container" && tracks.length) {
        const width =
          trackGrid(tracks.length, TRACKS_AREA_W, data?.grand).columns *
          TRACK_W;
        child.size = { width, height: TRACK_H };
        child.position = { x: -width / 2, y: Number(child.position?.y) || 0 };
      }
      walk(child);
    }
  };
  walk(node);
  assignIds(node);
  const ids = (path: string) =>
    matchNamePath(node, path).map(idOf).filter(Boolean);

  const hidden = [
    ...ids("grand_doctrine/bonus_container"),
    ...(data?.grand
      ? ids("no_grand_doctrine_info")
      : [
          ...ids("grand_doctrine/grand_doctrine_icon"),
          ...ids("tracks_container"),
        ]),
  ];
  const texts: Record<string, string> = {};
  const sources: Record<string, TextSource> = {};
  const banner = data?.grand?.label ?? data?.texts?.no_grand;
  if (data && banner)
    for (const id of ids("grand_doctrine/grand_doctrine_name"))
      texts[id] = banner;

  const grand = keySource(
    data?.grand?.label ? data.grand.token : null,
    "everywhere",
  );
  if (grand)
    for (const id of ids("grand_doctrine/grand_doctrine_name"))
      sources[id] = grand;
  return {
    win: {
      ...win,
      node,
      sprites: { ...win.sprites, ...(data?.sprites ?? {}) },
    },
    hidden,
    texts,
    sources,
  };
}

export function insetWindow(
  win: GuiWindow,
  width: number,
  height: number,
): GuiWindow {
  const node = structuredClone(win.node) as GuiNode;
  const inset = (
    value: number | string | undefined,
    near: number | string | undefined,
    parent: number,
  ) => {
    const n = Number(value);
    return typeof value === "number" ||
      (typeof value === "string" && value.trim() !== "" && !value.includes("%"))
      ? n < 0
        ? Math.max(0, parent - (Number(near) || 0) + n)
        : n
      : undefined;
  };
  const walk = (at: GuiNode, w: number, h: number) => {
    for (const child of at.children ?? []) {
      let cw = w;
      let ch = h;
      const size = child.size as
        | {
            width?: number | string;
            height?: number | string;
            x?: number | string;
            y?: number | string;
          }
        | undefined;
      if (size) {
        const sw = inset(size.width ?? size.x, child.position?.x, w);
        const sh = inset(size.height ?? size.y, child.position?.y, h);
        if (sw !== undefined || sh !== undefined) {
          child.size = {
            ...size,
            ...(sw !== undefined ? { width: sw } : {}),
            ...(sh !== undefined ? { height: sh } : {}),
          } as GuiNode["size"];
        }
        if (sw !== undefined) cw = sw;
        if (sh !== undefined) ch = sh;
      }
      walk(child, cw, ch);
    }
  };
  walk(node, width, height);
  return { ...win, node };
}

export function rewardWindow(
  win: GuiWindow,
  side: keyof typeof REWARD_SIDES,
): GuiWindow {
  const node = structuredClone(win.node) as GuiNode;
  const place = REWARD_SIDES[side];
  for (const child of node.children ?? []) {
    if (child.name === "reward_container") child.position = { ...place.medal };
    if (child.name === "level") child.position = { ...place.level };
    if (child.name === "mastery_text") child.position = { ...REWARD_TEXT };
  }
  return { ...win, node };
}

export function rewardFrames(
  reward: GuiDoctrineReward,
  colorFrame: number,
  pieces = 10,
) {
  const half = Math.max(1, Math.floor(pieces / 2));
  return {
    reward_frame: reward.unlocked ? 2 * colorFrame - 1 : 2 * colorFrame,
    reward_level_icon: reward.unlocked ? reward.level : half + reward.level,
    level: reward.level,
    mastery_text_bg: reward.unlocked ? colorFrame + 1 : 1,
  };
}

export function itemPlace(index: number): { left: number; top: number } {
  return {
    left: ITEM_PAD_X + (index % 2) * ITEM_W,
    top: ITEM_PAD_Y + Math.floor(index / 2) * ITEM_H,
  };
}

export function masteryText(format: string | undefined, value: number): string {
  if (!format) return String(value);
  return format.replace(
    /\$VALUE(?:\|([A-Za-z]?)[^$]*)?\$/,
    (_, colour: string | undefined) =>
      colour ? `§${colour}${value}§!` : String(value),
  );
}

export function costText(
  doctrines: GuiDoctrines,
  choice: GuiDoctrineChoice,
): string {
  const type = choice.xp_type || "army";
  const format = doctrines.texts?.[`${type}_cost` as "army_cost"] ?? "$COST$";
  const short = choice.xp_cost > (doctrines.experience?.[type] ?? 0);
  const value = short ? `§R${choice.xp_cost}§!` : String(choice.xp_cost);
  return format.replace(/\$COST[^$]*\$/, value);
}

export function pickerPlace(
  trackLeft: number,
  trackTop: number,
): { x: number; y: number } {
  const right = trackLeft + RIGHT_ANCHOR.x;
  return {
    x: right + PICKER_W <= 1920 ? right : trackLeft + LEFT_ANCHOR_X,
    y: trackTop + RIGHT_ANCHOR.y,
  };
}

export function trackFill(
  track: GuiDoctrineTrack,
  colorFrame: number,
  masteryFormat?: string,
) {
  return {
    texts: {
      "content/track_name": track.label,
      "sub_doctrine/sub_doctrine_name": track.sub?.label ?? "",
      "mastery_text_container/mastery_text": masteryText(
        masteryFormat,
        track.mastery,
      ),
    },
    pictures: {
      track_bg_image: track.background,
      milestone_icon: track.icon,
      milestone_bg_icon: track.icon_frame,
      sub_doctrine_icon: track.sub?.icon,
    },
    frames: {
      sub_doctrine_button: colorFrame,
      milestone_icon: track.complete ? 1 : 2,
      milestone_bg_icon: track.complete ? 1 : 2,
    },
    hide: [
      "milestone/preview_container",
      "sub_doctrine/bonus_container",
      "sub_doctrine/bonus_text",
      "lock_icon",
      ...(track.sub ? [] : ["sub_doctrine/sub_doctrine_icon"]),
    ],
  };
}

export function DoctrineLayer({
  win,
  projectId,
  tag,
  date,
  folder,
  onPickFolder,
  editMode = false,
  onChoose,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  folder: string | null;
  onPickFolder: (folder: string) => void;
  editMode?: boolean;

  onChoose?: (choice: DoctrineChoice) => void;
}) {
  const t = useTranslations("gameViewMilitary.doctrine");
  const active = win.name === DOCTRINE_VIEW;
  const data = useDoctrines(projectId, active ? tag : null, date, folder);
  const tab = useTemplateWindow(projectId, active ? TAB : null);
  const trackWin = useTemplateWindow(projectId, active ? TRACK : null);
  const trackEntry = useMemo(
    () =>
      trackWin.data ? insetWindow(trackWin.data, TRACK_W, TRACK_H) : undefined,
    [trackWin.data],
  );
  const rewardWin = useTemplateWindow(projectId, active ? REWARD : null);
  const rewardEntries = useMemo(
    () =>
      rewardWin.data
        ? {
            right: rewardWindow(rewardWin.data, "right"),
            left: rewardWindow(rewardWin.data, "left"),
          }
        : undefined,
    [rewardWin.data],
  );
  const pickerWin = useTemplateWindow(projectId, active ? PICKER : null);
  const itemWin = useTemplateWindow(projectId, active ? ITEM : null);
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const trackRects = useMemo(
    () =>
      trackEntry
        ? measureWindow(trackEntry.node, trackEntry.sprites, TRACK_W, TRACK_H)
        : {},
    [trackEntry],
  );
  const pickerEntry = useMemo(
    () =>
      pickerWin.data
        ? insetWindow(pickerWin.data, PICKER_W, PICKER_H)
        : undefined,
    [pickerWin.data],
  );
  const pickerRects = useMemo(
    () =>
      pickerEntry
        ? measureWindow(
            pickerEntry.node,
            pickerEntry.sprites,
            PICKER_W,
            PICKER_H,
          )
        : {},
    [pickerEntry],
  );
  const sprites = useMemo(() => guiRecord(data.data?.sprites), [data.data]);

  const [picker, setPicker] = useState<{
    folder: string | null;
    track: string | null;
  } | null>(null);
  if (picker && picker.folder !== folder) setPicker(null);

  const [scrolledAt, setScrolled] = useState({
    folder: null as string | null,
    x: 0,
    y: 0,
  });

  const scrolled = scrolledAt.folder === folder ? scrolledAt : { x: 0, y: 0 };

  const wheel = useCallback((page: HTMLDivElement | null) => {
    if (!page) return;
    const onWheel = (e: WheelEvent) => {
      if (e.ctrlKey || page.scrollHeight <= page.clientHeight) return;
      const step =
        e.deltaMode === 1
          ? e.deltaY * VIEW_WHEEL
          : Math.abs(e.deltaY) >= 50
            ? Math.sign(e.deltaY) * VIEW_WHEEL
            : e.deltaY;
      if (!step) return;
      e.preventDefault();
      page.scrollTop += step;
    };
    page.addEventListener("wheel", onWheel, { passive: false });
    return () => page.removeEventListener("wheel", onWheel);
  }, []);
  if (!active || !data.data) return null;
  const doctrines = data.data;
  const folders = guiArray(doctrines.folders);
  const doctrineTracks = guiArray(doctrines.tracks);
  const tabs = rects["name:folders_grid"];
  const tracks = rects["name:tracks_container"];
  const banner = rects["name:grand_doctrine_button"];
  const slot = trackRects["name:sub_doctrine_button"];
  const rewardsBox = trackRects["name:rewards_container"];

  const grid = trackGrid(doctrineTracks.length, TRACKS_AREA_W, doctrines.grand);

  const screenBottom = 1080 - (Number(win.node.position?.y) || 0) - VIEW_MARGIN;
  const area = tracks
    ? {
        x: VIEW_MARGIN,
        y: tracks.y,
        w: TRACKS_AREA_W,
        h: Math.max(TRACK_H / 4, screenBottom - tracks.y),
      }
    : null;
  const spot = (index: number) => trackSpot(index, grid, TRACKS_AREA_W);
  const wide = grid.columns * TRACK_W > TRACKS_AREA_W;
  const openTrack = picker?.track
    ? doctrineTracks.findIndex((track) => track.token === picker.track)
    : -1;
  const pickerTrack = openTrack >= 0 ? doctrineTracks[openTrack] : undefined;
  const choices = picker
    ? pickerTrack
      ? guiArray(pickerTrack.choices)
      : picker.track
        ? []
        : guiArray(doctrines.grand_choices)
    : [];
  const current = pickerTrack ? pickerTrack.sub?.token : doctrines.grand?.token;
  const place = picker
    ? pickerTrack && area
      ? pickerPlace(
          area.x + spot(openTrack).x - scrolled.x,
          area.y + spot(openTrack).y - scrolled.y,
        )
      : GRAND_PICKER
    : null;
  const list = pickerRects["name:list_container"];
  const close = pickerRects["name:close_button"];
  const choose = (token: string) => {
    if (!editMode || !onChoose) return;
    onChoose(
      pickerTrack
        ? { kind: "sub", from: current, to: token }
        : { kind: "grand", from: current, to: token },
    );
    setPicker(null);
  };
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {tabs &&
        tab.data &&
        folders.map((item, index) => (
          <div
            key={item.token}
            style={{
              position: "absolute",
              left: TABS_X + index * TAB_W,
              top: tabs.y,
              width: TAB_W,
              height: TAB_H,
              pointerEvents: "auto",
            }}
          >
            <Filled
              entry={tab.data as GuiWindow}
              left={0}
              top={0}
              width={TAB_W}
              height={TAB_H}
              texts={{ folder_name: item.label }}
              pictures={{ folder_button: item.tab }}
              frames={{
                folder_button: item.token === doctrines.folder ? 1 : 2,
              }}
              extraSprites={sprites}
            />
            <div
              role="button"
              aria-label={`doctrine folder ${item.token}`}
              onClick={() => onPickFolder(item.token)}
              style={{ position: "absolute", inset: 0, cursor: "pointer" }}
            />
          </div>
        ))}
      {area && doctrineTracks.length > 0 && (
        <div
          key={folder ?? ""}
          ref={wheel}
          data-doctrine-tracks
          style={{
            position: "absolute",
            left: area.x,
            top: area.y,
            width: area.w,
            height: area.h,
            overflowY: "auto",

            overflowX: wide ? "auto" : "hidden",

            ...GAME_SCROLLBAR,
            pointerEvents: "auto",
          }}
          onScroll={(e) =>
            setScrolled({
              folder,
              x: e.currentTarget.scrollLeft,
              y: e.currentTarget.scrollTop,
            })
          }
        >
          <div
            style={{
              position: "relative",
              width: wide ? grid.columns * TRACK_W : undefined,
              height: grid.rows * TRACK_H,
              overflow: "hidden",
            }}
          >
            {trackEntry &&
              doctrineTracks.map((item, index) => {
                const fill = trackFill(
                  item,
                  doctrines.color_frame,
                  doctrines.texts?.mastery,
                );
                return (
                  <Filled
                    key={item.token}
                    entry={trackEntry}
                    left={spot(index).x}
                    top={spot(index).y}
                    width={TRACK_W}
                    height={TRACK_H}
                    texts={fill.texts}
                    pictures={fill.pictures}
                    frames={fill.frames}
                    hide={fill.hide}
                    extraSprites={sprites}
                  />
                );
              })}
            {rewardsBox &&
              rewardEntries &&
              doctrineTracks.map((item, index) =>
                guiArray(item.rewards).length ? (
                  <RewardColumn
                    key={`rewards-${item.token}`}
                    rewards={guiArray(item.rewards)}
                    left={
                      spot(index).x +
                      rewardsBox.x +
                      rewardsBox.w / 2 -
                      REWARD_W / 2
                    }
                    top={spot(index).y + rewardsBox.y}
                    entries={rewardEntries}
                    folder={doctrines.folder}
                    colorFrame={doctrines.color_frame}
                    sprites={sprites}
                  />
                ) : null,
              )}
            {slot &&
              doctrineTracks.map((item, index) => (
                <Target
                  key={`slot-${item.token}`}
                  editMode={editMode}
                  rect={{
                    x: spot(index).x + slot.x,
                    y: spot(index).y + slot.y,
                    w: slot.w,
                    h: slot.h,
                  }}
                  label={
                    editMode
                      ? t("pickSub", { name: item.label })
                      : `pick subdoctrine ${item.token}`
                  }
                  onClick={() => setPicker({ folder, track: item.token })}
                />
              ))}
          </div>
        </div>
      )}
      {banner && (
        <Target
          editMode={editMode}
          rect={banner}
          label={editMode ? t("pickGrand") : "pick grand doctrine"}
          onClick={() => setPicker({ folder, track: null })}
        />
      )}
      {place && pickerEntry && itemWin.data && (
        <div
          style={{
            position: "absolute",
            left: place.x,
            top: place.y,
            pointerEvents: "auto",
          }}
        >
          <Filled
            entry={pickerEntry}
            left={0}
            top={0}
            width={PICKER_W}
            height={PICKER_H}
            texts={{
              title_text:
                (pickerTrack
                  ? doctrines.texts?.select_sub
                  : doctrines.texts?.select_grand) ?? "",
              description_text:
                (pickerTrack
                  ? doctrines.texts?.sub_desc
                  : doctrines.texts?.grand_desc) ?? "",
            }}
            hide={["unlock_button"]}
            extraSprites={sprites}
          />
          {list && (
            <div
              style={{
                position: "absolute",
                left: list.x,
                top: list.y,
                width: list.w,
                height: list.h,
                overflowX: "hidden",
                overflowY: "auto",
                ...GAME_SCROLLBAR,
              }}
            >
              <div
                style={{
                  position: "relative",
                  height:
                    itemPlace(Math.max(0, choices.length - 1)).top +
                    ITEM_H +
                    ITEM_PAD_Y,
                }}
              />
              {choices.map((choice, index) => (
                <div
                  key={choice.token}
                  style={{
                    position: "absolute",
                    ...itemPlace(index),
                    width: ITEM_W,
                    height: ITEM_H,
                  }}
                >
                  <Filled
                    entry={itemWin.data as GuiWindow}
                    left={0}
                    top={0}
                    width={ITEM_W}
                    height={ITEM_H}
                    texts={{
                      doctrine_name: choice.label,
                      xp_cost_text: costText(doctrines, choice),
                    }}
                    pictures={{ doctrine_icon: choice.icon }}
                    hide={[
                      "extra_advisor_window",
                      "main_advisor_window",

                      ...(choice.token === current ? [] : ["selected_frame"]),
                    ]}
                    extraSprites={sprites}
                  />

                  <div
                    role="button"
                    aria-label={
                      editMode
                        ? t("startAs", { name: choice.label })
                        : choice.label
                    }
                    onClick={() => choose(choice.token)}
                    style={{
                      position: "absolute",
                      inset: 0,
                      cursor: editMode ? "pointer" : undefined,
                    }}
                  />
                </div>
              ))}
            </div>
          )}
          {close && (
            <div
              role="button"
              aria-label="close doctrine picker"
              onClick={() => setPicker(null)}
              style={{
                position: "absolute",
                left: close.x,
                top: close.y,
                width: close.w,
                height: close.h,
                cursor: "pointer",
              }}
            />
          )}
        </div>
      )}
    </div>
  );
}

function RewardColumn({
  rewards,
  left,
  top,
  entries,
  folder,
  colorFrame,
  sprites,
}: {
  rewards: GuiDoctrineReward[];
  left: number;
  top: number;
  entries: { right: GuiWindow; left: GuiWindow };
  folder: string;
  colorFrame: number;
  sprites: Record<string, GuiSprite>;
}) {
  const empty = sprites.GFX_mastery_progress_bar_bg?.vanilla_url;
  const strip = sprites[`GFX_mastery_progress_bar_animated_${folder}`];
  const pieces = sprites.GFX_mastery_reward_icons?.frames || 10;
  return (
    <>
      {rewards.map((reward, index) => {
        const side = index % 2 === 0 ? "right" : "left";
        const x = left;
        const y = top + index * REWARD_H;
        const filled = Math.round(BAR.h * reward.progress);
        return (
          <div
            key={`${reward.token}-${reward.level}`}
            style={{
              position: "absolute",
              left: x,
              top: y,
              width: REWARD_W,
              height: REWARD_H,
            }}
          >
            {empty && (
              <div
                style={{
                  position: "absolute",
                  left: BAR.x,
                  top: 0,
                  width: BAR.w,
                  height: BAR.h,
                  backgroundImage: `url("${empty}")`,
                  backgroundSize: `${BAR.w}px ${BAR.h}px`,
                }}
              />
            )}
            {strip?.vanilla_url && filled > 0 && (
              <div
                style={{
                  position: "absolute",
                  left: BAR.x,
                  top: 0,
                  width: BAR.w,
                  height: filled,

                  backgroundImage: `url("${strip.vanilla_url}")`,
                  backgroundSize: `${strip.width}px ${strip.height}px`,
                  backgroundRepeat: "no-repeat",
                }}
              />
            )}
            <Filled
              entry={entries[side]}
              left={0}
              top={0}
              width={REWARD_W}
              height={REWARD_H}
              texts={{ mastery_text: String(reward.mastery) }}
              frames={rewardFrames(reward, colorFrame, pieces)}
              hide={["progress_line", "progress_fill", "highlight_container"]}
              extraSprites={sprites}
            />
            {reward.label && (
              <div
                title={reward.label}
                style={{
                  position: "absolute",
                  left: REWARD_SIDES[side].medal.x,
                  top: REWARD_SIDES[side].medal.y,
                  width: 100,
                  height: 88,
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

function Target({
  editMode,
  rect,
  label,
  onClick,
}: {
  editMode: boolean;
  rect: { x: number; y: number; w: number; h: number };
  label: string;
  onClick: () => void;
}) {
  if (editMode)
    return (
      <FieldHit
        x={rect.x}
        y={rect.y}
        w={rect.w}
        h={rect.h}
        label={label}
        onClick={onClick}
      />
    );
  return (
    <div
      role="button"
      aria-label={label}
      onClick={onClick}
      style={{
        position: "absolute",
        left: rect.x,
        top: rect.y,
        width: rect.w,
        height: rect.h,
        pointerEvents: "auto",
      }}
    />
  );
}
