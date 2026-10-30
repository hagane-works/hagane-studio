"use client";

import { useMemo } from "react";
import useSWR from "swr";
import { guiArray, guiRecord, readGuiResponse } from "./guiResponse";
import {
  guiRenderApi,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
  withGameLanguage,
} from "@/lib/guiRenderApi";
import {
  GuiCanvas,
  assignIds,
  measureWindow,
  textAlignOf,
} from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { useEditCanvas, usePrintedOver } from "./editContext";
import type { PrintedText } from "./editModel";
import { useTemplateWindow } from "./CountrySelectLayer";
import { matchNamePath } from "./gameNavigation";
import { statePrinted } from "./textSources";
import { abbreviate } from "./topbarStats";

export const STATE_VIEW = "countrystateview";

const STATE_ENTRY = "state_building_entry";
const SHARED_ENTRY = "state_shared_slot_building_entry";
const PROVINCE_ENTRY = "province_building_entry";

const RESOURCE_ENTRY = "state_resource_entry";

const STATE_SLOT = { w: 60, h: 52 };
const SHARED_SLOT = { w: 61, h: 50 };
const PROVINCE_SLOT = { w: 61, h: 84 };
const RESOURCE_SLOT = { w: 55, h: 30 };

const STRIP = "GFX_buildings_strip";
const RESOURCE_STRIP = "GFX_resources_strip";
const SCREEN_H = 1080;

const NO_TEXT: Record<string, string> = {};
const NOTHING_HIDDEN = new Set<string>();
const NO_PRINTED: Readonly<Record<string, PrintedText>> = {};

export interface GuiStateBuilding {
  building: string;
  label?: string;
  level: number;
  icon_frame?: number;
  province?: number;
  max?: number;
}

export interface GuiStateDetail {
  id: number;
  name?: string;
  category?: string;
  category_label?: string;
  manpower: number;
  victory_points: number;
  provinces: number;
  cores: string[];
  resources: Record<string, number>;
  buildings: GuiStateBuilding[];
  shared: GuiStateBuilding[];
  province_buildings: GuiStateBuilding[];
}

export interface GuiStates {
  tag: string;
  capital: number;
  states: { id: number; name?: string; manpower: number; category?: string }[];
  state?: GuiStateDetail;
  resource_frames?: Record<string, number>;
}

export function useStates(
  projectId: number,
  tag: string | null,
  date: string | null,
  id?: number | null,
) {
  return useSWR<GuiStates>(
    tag ? ["game-view-states", projectId, tag, date, id ?? 0] : null,
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/states?tag=${encodeURIComponent(tag as string)}`,
        ) +
          (date ? `&date=${encodeURIComponent(date)}` : "") +
          (id ? `&id=${id}` : ""),
      );
      return readGuiResponse<GuiStates>(response);
    },
    { revalidateOnFocus: false },
  );
}

export function textBox(
  node: GuiNode | undefined,
  fallback: { width: number; align: "left" | "center" | "right" },
) {
  return {
    width: Number(node?.maxwidth) || fallback.width,
    align: textAlignOf(node?.format, fallback.align),
  };
}

export function resourceEntryLayout(entry: GuiWindow | undefined) {
  const child = (name: string) =>
    entry?.node.children?.find((node) => node.name === name);
  const at = (node: GuiNode | undefined, x: number, y: number) => ({
    x: node ? Number(node.position?.x ?? x) || 0 : x,
    y: node ? Number(node.position?.y ?? y) || 0 : y,
  });
  const amount = child("amount");
  return {
    icon: at(child("resource"), 53, 11),
    amount: at(amount, 70, 11),
    ...textBox(amount, { width: 37, align: "center" }),
    font: amount?.font ? entry?.fonts[amount.font]?.bitmap : undefined,
  };
}

function frameStyle(
  sprite: GuiSprite,
  frame: number,
  height: number,
): React.CSSProperties {
  const frames = Math.max(1, sprite.frames || 1);
  const index = Math.max(0, Math.min(frames - 1, (frame || 1) - 1));
  const width = Math.round(
    (sprite.width / frames) * (height / Math.max(1, sprite.height)),
  );
  return {
    position: "absolute",
    width,
    height,
    backgroundImage: sprite.vanilla_url
      ? `url("${sprite.vanilla_url}")`
      : undefined,
    backgroundPosition: `-${index * width}px 0`,
    backgroundRepeat: "no-repeat",
    backgroundSize: `${width * frames}px ${height}px`,
  };
}

function BuildingEntry({
  entry,
  item,
  strip,
  left,
  top,
  width,
  height,
}: {
  entry: GuiWindow;
  item: GuiStateBuilding;
  strip: GuiSprite | undefined;
  left: number;
  top: number;
  width: number;
  height: number;
}) {
  const edit = useEditCanvas();
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    for (const child of clone.children ?? []) {
      if (
        child.name === "damage_bar" ||
        child.name === "remove" ||
        child.name === "building_status_overlay" ||
        child.name === "building_picture"
      ) {
        child.sprite = null;
      }
    }
    assignIds(clone);
    return clone;
  }, [entry.node]);
  const rects = useMemo(
    () => measureWindow(node, entry.sprites, 1920, SCREEN_H),
    [node, entry.sprites],
  );
  const levelNode = entry.node.children?.find(
    (child) => child.name === "level",
  );
  const font = levelNode?.font
    ? entry.fonts[levelNode.font]?.bitmap
    : undefined;
  const levelBox = textBox(levelNode, { width: 60, align: "center" });
  const levelRect = rects["name:level"];
  const pictureNode = entry.node.children?.find(
    (child) => child.name === "building_picture",
  );
  const at = pictureNode
    ? {
        x: Number(pictureNode.position?.x ?? 0),
        y: Number(pictureNode.position?.y ?? 0),
      }
    : { x: 0, y: 0 };

  return (
    <div
      style={{ position: "absolute", left, top }}
      title={`${item.label || item.building} ${item.level}`}
    >
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${left},${top}`,
          entry,
          node,
          entry.sprites,
          NO_TEXT,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NOTHING_HIDDEN}
        isolateId={null}
        transparent
        baseW={width}
        baseH={height}
      />
      {strip && (
        <div
          style={{
            ...frameStyle(strip, item.icon_frame ?? 1, 40),
            left: at.x,
            top: at.y,
          }}
        />
      )}
      {font && levelRect && (
        <BitmapText
          font={font}
          text={String(item.level)}
          left={levelRect.x}
          top={levelRect.y}
          width={levelBox.width}
          height={font.line + 4}
          align={levelBox.align}
          vcenter={false}
          maxLines={1}
        />
      )}
    </div>
  );
}

export function StateLayer({
  win,
  projectId,
  tag,
  date,
  stateId,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  stateId?: number | null;
}) {
  const active = win.name === STATE_VIEW;
  const states = useStates(
    projectId,
    active ? tag : null,
    date ?? null,
    stateId,
  );
  const stateEntry = useTemplateWindow(projectId, active ? STATE_ENTRY : null);
  const sharedEntry = useTemplateWindow(
    projectId,
    active ? SHARED_ENTRY : null,
  );
  const provinceEntry = useTemplateWindow(
    projectId,
    active ? PROVINCE_ENTRY : null,
  );
  const resourceEntry = useTemplateWindow(
    projectId,
    active ? RESOURCE_ENTRY : null,
  );
  const strip = useSWR<Record<string, GuiSprite>>(
    active ? ["game-view-sprites", projectId, "state-strip"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites([STRIP, RESOURCE_STRIP]);
    },
    { revalidateOnFocus: false },
  );
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, SCREEN_H) : {}),
    [active, win],
  );
  const nameFont = useMemo(() => {
    const found =
      Object.keys(win.fonts).find((key) => key.includes("20")) ??
      Object.keys(win.fonts)[0];
    return found ? win.fonts[found]?.bitmap : undefined;
  }, [win.fonts]);

  const detail = states.data?.state;

  const printed = useMemo(
    () => (active ? statePrinted(win, detail) : NO_PRINTED),
    [active, win, detail],
  );
  usePrintedOver(win, "state", printed);
  if (!active || !detail) return null;
  const art = strip.data?.[STRIP];
  const resourceArt = strip.data?.[RESOURCE_STRIP];
  const nameRect = rects["name:state_name"];
  const resourceSlot = resourceEntryLayout(resourceEntry.data);

  const frameValue = (
    slot: string,
    text: string,
    fallback: { width: number; align: "left" | "center" | "right" },
  ) => {
    const node = matchNamePath(win.node, slot)[0];
    const rect = rects[`name:${slot}`];
    const font = node?.font ? win.fonts[node.font]?.bitmap : undefined;
    if (!node || !rect || !font || !text) return null;
    const box = textBox(node, fallback);
    return (
      <BitmapText
        key={slot}
        font={font}
        text={text}
        left={rect.x}
        top={rect.y}
        width={box.width}
        height={font.line + 4}
        align={box.align}
        vcenter={false}
        maxLines={1}
      />
    );
  };
  const shelves: {
    rect?: { x: number; y: number };
    entry?: GuiWindow;
    slot: { w: number; h: number };
    items: GuiStateBuilding[];
    wide: number;
    tall: number;
  }[] = [
    {
      rect: rects["name:state_building_entries"],
      entry: stateEntry.data,
      slot: STATE_SLOT,
      items: guiArray(detail.buildings),
      wide: 140,
      tall: 70,
    },
    {
      rect: rects["name:state_shared_slot_building_entries"],
      entry: sharedEntry.data,
      slot: SHARED_SLOT,
      items: guiArray(detail.shared),
      wide: 70,
      tall: 70,
    },
    {
      rect: rects["name:province_building_entries"],
      entry: provinceEntry.data,
      slot: PROVINCE_SLOT,
      items: guiArray(detail.province_buildings),
      wide: 96,
      tall: 50,
    },
  ];

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {nameFont && nameRect && (
        <BitmapText
          font={nameFont}
          text={detail.name || `#${detail.id}`}
          left={nameRect.x}
          top={nameRect.y}
          width={320}
          height={nameFont.line + 4}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
      {frameValue(
        "manpower",
        detail.manpower > 0 ? abbreviate(detail.manpower) : "",
        { width: 80, align: "center" },
      )}
      {frameValue(
        "victory_points",
        detail.victory_points > 0 ? String(detail.victory_points) : "",
        { width: 70, align: "right" },
      )}
      {shelves.flatMap(({ rect, entry, slot, items, wide, tall }, shelf) =>
        !rect || !entry
          ? []
          : items.map((item, at) => (
              <BuildingEntry
                key={`${shelf}-${item.province ?? 0}-${item.building}`}
                entry={entry}
                item={item}
                strip={art}
                left={rect.x + at * slot.w}
                top={rect.y}
                width={wide}
                height={tall}
              />
            )),
      )}

      {rects["name:state_resources_entries"] &&
        Object.entries(guiRecord(detail.resources))
          .filter(([, amount]) => amount > 0)
          .sort(
            (a, b) =>
              ((states.data?.resource_frames ?? {})[a[0]] ?? 99) -
              ((states.data?.resource_frames ?? {})[b[0]] ?? 99),
          )
          .map(([resource, amount], at) => {
            const box = rects["name:state_resources_entries"];
            const frames = states.data?.resource_frames ?? {};
            return (
              <div
                key={resource}
                style={{
                  position: "absolute",
                  left: box.x + (at % 2) * RESOURCE_SLOT.w,
                  top: box.y + Math.floor(at / 2) * RESOURCE_SLOT.h,
                  width: RESOURCE_SLOT.w,
                  height: RESOURCE_SLOT.h,
                }}
                title={`${resource} ${amount}`}
              >
                {resourceArt && (
                  <div
                    style={{
                      ...frameStyle(
                        resourceArt,
                        frames[resource] ?? 1,
                        resourceArt.height || 22,
                      ),
                      left: resourceSlot.icon.x,
                      top: resourceSlot.icon.y,
                    }}
                  />
                )}
                {(resourceSlot.font ?? nameFont) && (
                  <BitmapText
                    font={(resourceSlot.font ?? nameFont)!}
                    text={String(amount)}
                    left={resourceSlot.amount.x}
                    top={resourceSlot.amount.y}
                    width={resourceSlot.width}
                    height={(resourceSlot.font ?? nameFont)!.line + 2}
                    align={resourceSlot.align}
                    vcenter={false}
                    maxLines={1}
                  />
                )}
              </div>
            );
          })}
    </div>
  );
}
