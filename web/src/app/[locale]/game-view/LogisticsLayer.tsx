"use client";

import { useMemo } from "react";
import useSWR from "swr";
import { readGuiResponse } from "./guiResponse";
import {
  guiRenderApi,
  type GuiCountryStats,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
  withGameLanguage,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { useEditCanvas } from "./editContext";
import { useTemplateWindow } from "./CountrySelectLayer";
import { SpriteBar, type BarSprite } from "./SpriteBar";
import {
  amountColor,
  nodeNamed,
  stripEntries,
  stripSlot,
} from "./resourceStrip";
import { pickable } from "./spriteFrame";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export const LOGISTICS = "countrylogisticsview";

const LAND_ROW = "logistics_overview_land_equipment_entry";
const NAVAL_ROW = "logistics_overview_naval_equipment_entry";
const AIR_ROW = "logistics_overview_air_equipment_entry";
const RESOURCE_ITEM = "logistics_overview_resource_item";

const ROW_H = 58;
const ROW_W = 565;
const GRID_AT = { x: 13, y: 33 };

const STRIP_W = 70;

const ICON_BOX = { w: 147, h: 69 };

const CHIP = 25;
const CHIP_ROWS = 2;
const SCREEN_H = 1080;

const LIST_MARGIN_BOTTOM = 123;

const NOTHING_HIDDEN = new Set<string>();

const ROW_FOR: Readonly<Record<string, string>> = {
  infantry: LAND_ROW,
  armor: LAND_ROW,
  air: AIR_ROW,
  capital_ships: NAVAL_ROW,
  screen_ships: NAVAL_ROW,
  other_ships: NAVAL_ROW,
};

export interface GuiMateriel {
  archetype: string;
  model: string;
  label?: string;
  group?: string;
  needs: number;
  icon?: string;
  resources?: Record<string, number>;
}

export interface GuiLogistics {
  tag: string;
  equipment: GuiMateriel[];
  resource_frames?: Record<string, number>;
  resources?: Record<string, number>;
  convoys?: number;
  nukes?: number;

  factories?: {
    military_used: number;
    military_total: number;
    naval_used: number;
    naval_total: number;
  };
  sprites?: Record<string, BarSprite>;
}

const USAGE_BAR = "GFX_logistics_fuelbar";
const USAGE_SIZE = { w: 100, h: 14 };

export function usageFraction(
  used: number | undefined,
  total: number | undefined,
): number {
  return total ? Math.min(1, Math.max(0, (used ?? 0) / total)) : 0;
}

function useLogistics(
  projectId: number,
  tag: string | null,
  date: string | null,
) {
  return useSWR<GuiLogistics>(
    tag ? ["game-view-logistics", projectId, tag, date] : null,
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/logistics?tag=${encodeURIComponent(tag as string)}`,
        ) + (date ? `&date=${encodeURIComponent(date)}` : ""),
      );
      return readGuiResponse<GuiLogistics>(response);
    },
    { revalidateOnFocus: false },
  );
}

function stacked(
  entry: GuiWindow,
  frames: Readonly<Record<string, number>>,
  drop: readonly string[],
  blank: readonly string[] = [],
) {
  const node = structuredClone(entry.node) as GuiNode;
  const walk = (parent: GuiNode) => {
    for (const child of parent.children ?? []) {
      if (child.name && drop.includes(child.name)) child.sprite = null;
      if (child.name && frames[child.name] !== undefined)
        child.frame = frames[child.name];
      walk(child);
    }
  };
  walk(node);
  assignIds(node);
  const texts: Record<string, string> = {};
  const collect = (parent: GuiNode) => {
    for (const child of (parent.children ?? []) as (GuiNode & {
      _id?: string;
    })[]) {
      if (child._id && child.name && blank.includes(child.name))
        texts[child._id] = "";
      collect(child);
    }
  };
  collect(node);
  return { node, texts };
}

function stripStyle(
  sprite: GuiSprite,
  frame: number,
  size: number,
): React.CSSProperties {
  const frames = Math.max(1, sprite.frames || 1);
  const index = Math.max(0, Math.min(frames - 1, (frame || 1) - 1));
  const width = Math.round(
    (sprite.width / frames) * (size / Math.max(1, sprite.height)),
  );
  return {
    position: "absolute",
    width,
    height: size,
    backgroundImage: sprite.vanilla_url
      ? `url("${sprite.vanilla_url}")`
      : undefined,
    backgroundPosition: `-${index * width}px 0`,
    backgroundRepeat: "no-repeat",
    backgroundSize: `${width * frames}px ${size}px`,
  };
}

function MaterielRow({
  entry,
  item,
  icon,
  strip,
  frames,
  left,
  top,
}: {
  entry: GuiWindow;
  item: GuiMateriel;
  icon: GuiSprite | undefined;
  strip: GuiSprite | undefined;
  frames: Readonly<Record<string, number>>;
  left: number;
  top: number;
}) {
  const edit = useEditCanvas();

  const built = useMemo(
    () =>
      stacked(
        entry,
        {},
        [
          "equipment_icon",
          "efficiency_progressbar",
          "status_progressbar",
          "status_progress_marker",
        ],
        ["equipment_type", "needs", "produced", "balance", "in_stock"],
      ),
    [entry],
  );
  const node = built.node;
  const rects = useMemo(
    () => measureWindow(node, entry.sprites, 1920, SCREEN_H),
    [node, entry.sprites],
  );
  const slot = (name: string) =>
    entry.node.children?.find((child) => child.name === name);
  const typeNode = slot("equipment_type");
  const needNode = slot("needs");
  const typeFont = typeNode?.font
    ? entry.fonts[typeNode.font]?.bitmap
    : undefined;
  const needFont = needNode?.font
    ? entry.fonts[needNode.font]?.bitmap
    : undefined;
  const typeRect = rects["name:equipment_type"];
  const needRect = rects["name:needs"];
  const iconNode = slot("equipment_icon");
  const iconAt = iconNode
    ? {
        x: Number(iconNode.position?.x ?? 0),
        y: Number(iconNode.position?.y ?? 0),
      }
    : undefined;
  const chipRect = rects["name:resources_grid"];
  const bill = Object.entries(item.resources ?? {})
    .filter(([, amount]) => amount > 0)
    .sort((a, b) => (frames[a[0]] ?? 99) - (frames[b[0]] ?? 99));

  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${left},${top}`,
          entry,
          node,
          entry.sprites,
          built.texts,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NOTHING_HIDDEN}
        isolateId={null}
        transparent
        baseW={ROW_W}
        baseH={ROW_H}
      />

      {icon?.vanilla_url && iconAt && (
        <div
          style={{
            position: "absolute",
            ...(() => {
              const scale = Math.min(
                1,
                ICON_BOX.w / icon.width,
                ICON_BOX.h / icon.height,
              );
              const w = Math.round(icon.width * scale);
              const h = Math.round(icon.height * scale);
              return {
                left: Math.round(iconAt.x - w / 2),
                top: Math.round(iconAt.y - h / 2),
                width: w,
                height: h,
              };
            })(),
            backgroundImage: `url("${icon.vanilla_url}")`,
            backgroundSize: "100% 100%",
            backgroundRepeat: "no-repeat",
          }}
          {...pickable(item.icon, 1, item.label || item.model)}
        />
      )}
      {typeFont && typeRect && (
        <BitmapText
          font={typeFont}
          text={item.label || item.model}
          left={typeRect.x}
          top={typeRect.y}
          width={typeNode?.maxwidth ?? 150}
          height={typeFont.line + 4}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
      {needFont && needRect && (
        <BitmapText
          font={needFont}
          text={String(item.needs)}
          left={needRect.x}
          top={needRect.y}
          width={needNode?.maxwidth ?? 50}
          height={needFont.line + 4}
          align="right"
          vcenter={false}
          maxLines={1}
        />
      )}

      {strip &&
        chipRect &&
        bill.map(([resource, amount], at) => (
          <div
            key={resource}
            style={{
              position: "absolute",
              left: chipRect.x + Math.floor(at / CHIP_ROWS) * (CHIP + 22),
              top: chipRect.y + (at % CHIP_ROWS) * CHIP,
              width: CHIP + 20,
              height: CHIP,
            }}
          >
            <div
              style={stripStyle(strip, frames[resource] ?? 1, CHIP - 4)}
              {...pickable(
                "GFX_resources_strip",
                frames[resource] ?? 1,
                resource,
              )}
            />
            {needFont && (
              <BitmapText
                font={needFont}
                text={String(Math.round(amount))}
                left={CHIP - 2}
                top={Math.round((CHIP - needFont.line) / 2)}
                width={22}
                height={needFont.line + 2}
                align="left"
                vcenter={false}
                maxLines={1}
              />
            )}
          </div>
        ))}
    </div>
  );
}

export function LogisticsLayer({
  win,
  projectId,
  tag,
  date,
  stats,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  stats?: GuiCountryStats;
}) {
  const edit = useEditCanvas();
  const active = win.name === LOGISTICS;
  const logistics = useLogistics(projectId, active ? tag : null, date ?? null);
  const land = useTemplateWindow(projectId, active ? LAND_ROW : null);
  const naval = useTemplateWindow(projectId, active ? NAVAL_ROW : null);
  const air = useTemplateWindow(projectId, active ? AIR_ROW : null);
  const chip = useTemplateWindow(projectId, active ? RESOURCE_ITEM : null);

  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, SCREEN_H) : {}),
    [active, win],
  );
  const items = useMemo(
    () => logistics.data?.equipment ?? [],
    [logistics.data],
  );
  const iconNames = useMemo(() => {
    const names = new Set<string>();
    for (const item of items) if (item.icon) names.add(item.icon);
    names.add("GFX_resources_strip");
    return [...names].sort();
  }, [items]);
  const sprites = useSWR<Record<string, GuiSprite>>(
    active && iconNames.length > 1
      ? ["game-view-sprites", projectId, "logistics-icons", iconNames.join(",")]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(iconNames);
    },
    { revalidateOnFocus: false },
  );

  const chipNodes = useMemo(() => {
    if (!chip.data) return null;
    const out: Record<
      string,
      { node: GuiNode; texts: Record<string, string> }
    > = {};
    for (const [resource, frame] of Object.entries(
      logistics.data?.resource_frames ?? {},
    )) {
      out[resource] = stacked(chip.data, { icon: frame }, [], ["value"]);
    }
    return out;
  }, [chip.data, logistics.data]);
  const chipFont = useMemo(() => {
    if (!chip.data) return undefined;
    const value = chip.data.node.children?.find(
      (child) => child.name === "value",
    );
    return value?.font ? chip.data.fonts[value.font]?.bitmap : undefined;
  }, [chip.data]);

  if (!active || !logistics.data) return null;
  const frames = logistics.data.resource_frames ?? {};
  const strip = sprites.data?.["GFX_resources_strip"];
  const materiel = rects["name:materiel"];
  const usage = logistics.data.factories;
  const usageBar = logistics.data.sprites?.[USAGE_BAR];
  const militaryBar = rects["name:military_factories_usage"];
  const navalBar = rects["name:naval_factories_usage"];
  const nukeCount = rects["name:nuke_count"];

  const nukeFontName = Object.keys(win.fonts).find((key) =>
    key.includes("16mbs"),
  );
  const nukeFont = nukeFontName ? win.fonts[nukeFontName]?.bitmap : undefined;
  const stripBox = rects["name:resources_grid"];
  const stripGrid = nodeNamed(win.node, "resources_grid");
  const resources = stripEntries(
    frames,
    stats ?? ({ tag, resources: logistics.data.resources } as GuiCountryStats),
  );

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {chipNodes &&
        chip.data &&
        stripBox &&
        resources.map(([resource, amount], at) => (
          <div
            key={resource}
            style={{
              position: "absolute",
              left: stripBox.x + stripSlot(stripGrid, at, STRIP_W).x,
              top: stripBox.y + stripSlot(stripGrid, at, STRIP_W).y,
            }}
          >
            <GuiCanvas
              node={chipNodes[resource].node}
              {...edit(
                `${(chip.data as GuiWindow).name}:${resource}`,
                chip.data as GuiWindow,
                chipNodes[resource].node,
                (chip.data as GuiWindow).sprites,
                chipNodes[resource].texts,
              )}
              texts={(chip.data as GuiWindow).texts}
              fonts={(chip.data as GuiWindow).fonts}
              hidden={NOTHING_HIDDEN}
              isolateId={null}
              transparent
              baseW={STRIP_W}
              baseH={31}
            />
            {chipFont && (
              <BitmapText
                font={chipFont}
                text={String(amount)}
                color={amountColor(amount)}
                left={35}
                top={7}
                width={32}
                height={chipFont.line + 2}
                align="left"
                vcenter={false}
                maxLines={1}
              />
            )}
          </div>
        ))}

      {materiel && land.data && naval.data && air.data && (
        <div
          style={{
            position: "absolute",
            left: materiel.x + GRID_AT.x,
            top: materiel.y + GRID_AT.y,
            width: ROW_W,

            height: Math.max(0, materiel.h - GRID_AT.y - LIST_MARGIN_BOTTOM),
            overflowY: "auto",
            overflowX: "hidden",
            ...GAME_SCROLLBAR,
            pointerEvents: "auto",
          }}
        >
          <div style={{ position: "relative", height: items.length * ROW_H }}>
            {items.map((item, at) => {
              const which = ROW_FOR[item.group ?? ""] ?? LAND_ROW;
              const entry =
                which === AIR_ROW
                  ? air.data
                  : which === NAVAL_ROW
                    ? naval.data
                    : land.data;
              return (
                <MaterielRow
                  key={`${item.group}-${item.model}`}
                  entry={entry as GuiWindow}
                  item={item}
                  icon={item.icon ? sprites.data?.[item.icon] : undefined}
                  strip={strip}
                  frames={frames}
                  left={0}
                  top={at * ROW_H}
                />
              );
            })}
          </div>
        </div>
      )}

      {usageBar && militaryBar && (
        <SpriteBar
          x={militaryBar.x}
          y={militaryBar.y}
          size={USAGE_SIZE}
          sprite={usageBar}
          fraction={usageFraction(usage?.military_used, usage?.military_total)}
        />
      )}
      {usageBar && navalBar && (
        <SpriteBar
          x={navalBar.x}
          y={navalBar.y}
          size={USAGE_SIZE}
          sprite={usageBar}
          fraction={usageFraction(usage?.naval_used, usage?.naval_total)}
        />
      )}
      {nukeFont && nukeCount && (
        <BitmapText
          font={nukeFont}
          text={String(logistics.data.nukes ?? 0)}
          left={nukeCount.x}
          top={nukeCount.y}
          width={120}
          height={nukeFont.line + 2}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
    </div>
  );
}
