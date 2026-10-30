"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
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
import { FieldHit } from "./FieldHit";
import { useTemplateWindow } from "./CountrySelectLayer";
import {
  LINE_H,
  QUEUE_MARGIN,
  QueueLines,
  type GuiProductionQueue,
} from "./ProductionQueue";
import {
  amountColor,
  nodeNamed,
  stripEntries,
  stripSlot,
} from "./resourceStrip";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export const PRODUCTION = "countryproductionlineview";

const PANEL = "production_equipment_window_military";
const ROW = "military_equipment_info_entry";
const NEW_LINE = "production_military_new_line_entry";
const ROW_RESOURCE = "production_type_resource_item";

const TOP_RESOURCE = "resource_strip_item";

const SLOT_H = 80;
const RESOURCE_H = 23;
const RESOURCE_MAX = 6;

const TOP_RESOURCE_W = 70;

const PANEL_W = 495;
const ROW_W = 470;

const LIST_MARGIN = 12;

const NO_TEXT: Record<string, string> = {};
const NOTHING_HIDDEN = new Set<string>();

const FILTERS: readonly { button: string; group: string }[] = [
  { button: "show_equipment", group: "infantry" },
  { button: "show_armor", group: "armor" },
  { button: "show_air", group: "air" },
  { button: "show_capital_ships", group: "capital_ships" },
  { button: "show_screen_ships", group: "screen_ships" },
  { button: "show_other_ships", group: "other_ships" },
];

export interface GuiEquipmentItem {
  name: string;
  label?: string;

  general_name?: string;

  design?: boolean;

  type?: string;

  obsolete?: boolean;
  archetype: string;
  category: string;
  group?: string;
  year: number;
  cost: number;
  picture?: string;

  icon?: string;

  designer?: string;
  resources?: Record<string, number>;
}

export interface GuiProduction {
  tag: string;
  equipment: GuiEquipmentItem[];

  resource_frames?: Record<string, number>;

  queue?: GuiProductionQueue;
}

export function catalogueItems(
  equipment: readonly GuiEquipmentItem[],
  groups: readonly string[],
  showOutdated: boolean,
): GuiEquipmentItem[] {
  return equipment.filter(
    (item) =>
      groups.includes(item.group ?? "") && (showOutdated || !item.obsolete),
  );
}

export function useProduction(
  projectId: number,
  tag: string | null,
  date: string | null,
) {
  return useSWR<GuiProduction>(
    tag ? ["game-view-production", projectId, tag, date] : null,
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/production?tag=${encodeURIComponent(tag as string)}`,
        ) + (date ? `&date=${encodeURIComponent(date)}` : ""),
      );
      return readGuiResponse<GuiProduction>(response);
    },
    { revalidateOnFocus: false },
  );
}

function stackedWindow(
  entry: GuiWindow,
  blank: readonly string[],
  hide: readonly string[],
  frames: Readonly<Record<string, number>> = {},
) {
  const node = structuredClone(entry.node) as GuiNode;
  const walk = (parent: GuiNode) => {
    for (const child of parent.children ?? []) {
      if (child.name && hide.includes(child.name)) child.sprite = null;
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

function ResourceChip({
  entry,
  frame,
  amount,
  left,
  top,
  width,
  height,
  color,
}: {
  entry: GuiWindow;
  frame: number;
  amount: number;
  left: number;
  top: number;
  width: number;
  height: number;

  color?: string;
}) {
  const edit = useEditCanvas();

  const iconName =
    entry.node.children?.find(
      (child) => child.type === "icontype" && child.name === "icon",
    )?.name ??
    entry.node.children?.find((child) => child.type === "icontype")?.name ??
    "icon";
  const textName = entry.node.children?.some((child) => child.name === "value")
    ? "value"
    : "amount";
  const built = useMemo(
    () =>
      stackedWindow(entry, [textName], ["energy_ratio_anim"], {
        [iconName]: frame,
      }),
    [entry, frame, iconName, textName],
  );
  const rects = useMemo(
    () => measureWindow(built.node, entry.sprites, 1920, 1080),
    [built.node, entry.sprites],
  );
  const amountNode = entry.node.children?.find(
    (child) => child.name === textName,
  );
  const font = amountNode?.font
    ? entry.fonts[amountNode.font]?.bitmap
    : undefined;
  const amountRect = rects[`name:${textName}`];

  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${left},${top}`,
          entry,
          built.node,
          entry.sprites,
          built.texts,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NOTHING_HIDDEN}
        isolateId={null}
        transparent
        baseW={width}
        baseH={height}
      />
      {font && amountRect && (
        <BitmapText
          font={font}
          text={String(amount)}
          color={color}
          left={amountRect.x}
          top={amountRect.y}
          width={amountNode?.maxwidth ?? 30}
          height={font.line + 4}
          align="center"
          vcenter={false}
          maxLines={1}
        />
      )}
    </div>
  );
}

function EquipmentRow({
  entry,
  resourceEntry,
  item,
  icon,
  frames,
  left,
  top,
}: {
  entry: GuiWindow;
  resourceEntry: GuiWindow | undefined;
  item: GuiEquipmentItem;
  icon: GuiSprite | undefined;
  frames: Readonly<Record<string, number>>;
  left: number;
  top: number;
}) {
  const edit = useEditCanvas();
  const built = useMemo(
    () =>
      stackedWindow(
        entry,
        ["name", "general_name", "production_cost"],
        [
          "outdated_equipment_overlay",
          "prod_line_glow",
          "creator_flag",
          "role_icon",
          "equipment_designer",
          "equipment_designer_can_upgrade",
          "sub_tech",
          "carrier_fighter_icon",
          "design_invalid",
          "blueprint_icon",

          "equipment_icon",
        ],
      ),
    [entry],
  );
  const rects = useMemo(
    () => measureWindow(built.node, entry.sprites, 1920, 1080),
    [built.node, entry.sprites],
  );
  const nameNode = entry.node.children?.find((child) => child.name === "name");
  const generalNode = entry.node.children?.find(
    (child) => child.name === "general_name",
  );
  const generalFont = generalNode?.font
    ? entry.fonts[generalNode.font]?.bitmap
    : undefined;
  const generalRect = rects["name:general_name"];
  const costNode = entry.node.children?.find(
    (child) => child.name === "production_cost",
  );
  const nameFont = nameNode?.font
    ? entry.fonts[nameNode.font]?.bitmap
    : undefined;
  const costFont = costNode?.font
    ? entry.fonts[costNode.font]?.bitmap
    : undefined;
  const nameRect = rects["name:name"];
  const costRect = rects["name:production_cost"];
  const iconRect = rects["name:equipment_icon"];
  const resourceRect = rects["name:resources_grid"];
  const resources = Object.entries(item.resources ?? {})
    .filter(([, amount]) => amount > 0)
    .sort((a, b) => (frames[a[0]] ?? 99) - (frames[b[0]] ?? 99))
    .slice(0, RESOURCE_MAX);

  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${left},${top}`,
          entry,
          built.node,
          entry.sprites,
          built.texts,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NOTHING_HIDDEN}
        isolateId={null}
        transparent
        baseW={470}
        baseH={SLOT_H}
      />

      {icon?.vanilla_url && iconRect && (
        <div
          style={{
            position: "absolute",
            left: iconRect.x + Math.round((iconRect.w - icon.width) / 2),
            top: iconRect.y + Math.round((iconRect.h - icon.height) / 2),
            width: icon.width,
            height: icon.height,
            backgroundImage: `url("${icon.vanilla_url}")`,
            backgroundSize: "100% 100%",
            backgroundRepeat: "no-repeat",
          }}
        />
      )}
      {nameFont && nameRect && (
        <BitmapText
          font={nameFont}
          text={item.label || item.name}
          left={nameRect.x}
          top={nameRect.y}
          width={nameNode?.maxwidth ?? 308}
          height={nameFont.line + 4}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
      {generalFont && generalRect && item.general_name && (
        <BitmapText
          font={generalFont}
          text={item.general_name}
          left={generalRect.x}
          top={generalRect.y}
          width={generalNode?.maxwidth ?? 308}
          height={generalFont.line + 4}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
      {costFont && costRect && (
        <BitmapText
          font={costFont}
          text={String(Math.round(item.cost * 10) / 10)}
          left={costRect.x}
          top={costRect.y}
          width={costNode?.maxwidth ?? 160}
          height={costFont.line + 4}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
      {resourceEntry &&
        resourceRect &&
        resources.map(([resource, amount], at) => (
          <ResourceChip
            key={resource}
            entry={resourceEntry}
            frame={frames[resource] ?? 1}
            amount={amount}
            left={resourceRect.x}
            top={resourceRect.y + at * RESOURCE_H}
            width={35}
            height={27}
          />
        ))}
    </div>
  );
}

export function ProductionLayer({
  win,
  projectId,
  tag,
  date,
  stats,
  onEditEquipment,
  onOpenDesigner,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  stats?: GuiCountryStats;

  onEditEquipment?: (name: string) => void;

  onOpenDesigner?: (name: string, kind: string) => void;
}) {
  const t = useTranslations("gameViewMilitary.production");
  const edit = useEditCanvas();
  const active = win.name === PRODUCTION;
  const production = useProduction(
    projectId,
    active ? tag : null,
    date ?? null,
  );
  const panel = useTemplateWindow(projectId, active ? PANEL : null);
  const row = useTemplateWindow(projectId, active ? ROW : null);
  const newLine = useTemplateWindow(projectId, active ? NEW_LINE : null);
  const rowResource = useTemplateWindow(
    projectId,
    active ? ROW_RESOURCE : null,
  );
  const topResource = useTemplateWindow(
    projectId,
    active ? TOP_RESOURCE : null,
  );

  const [open, setOpen] = useState(false);
  const [groups, setGroups] = useState<readonly string[]>(() =>
    FILTERS.map((f) => f.group),
  );

  const [showOutdated, setShowOutdated] = useState(false);

  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );

  const panelBuilt = useMemo(
    () =>
      panel.data
        ? stackedWindow(panel.data, ["available_MIOs"], [], {
            checkbox: showOutdated ? 2 : 1,
          })
        : null,
    [panel.data, showOutdated],
  );
  const panelNode = panelBuilt?.node ?? null;
  const showAt = useMemo(() => {
    const raw = panel.data?.node.raw as
      { show_position?: { x: number; y: number } } | undefined;
    const fallback = panel.data?.node.position;
    return {
      x: raw?.show_position?.x ?? Number(fallback?.x ?? 0),
      y: raw?.show_position?.y ?? Number(fallback?.y ?? 0),
    };
  }, [panel.data]);
  const panelRects = useMemo(
    () =>
      panelNode && panel.data
        ? measureWindow(panelNode, panel.data.sprites, 1920, 1080)
        : {},
    [panelNode, panel.data],
  );
  const items = useMemo(
    () =>
      catalogueItems(production.data?.equipment ?? [], groups, showOutdated),
    [production.data, groups, showOutdated],
  );
  const iconNames = useMemo(() => {
    const names = new Set<string>();
    for (const item of items) if (item.icon) names.add(item.icon);
    return [...names].sort();
  }, [items]);
  const icons = useSWR<Record<string, GuiSprite>>(
    open && iconNames.length
      ? ["game-view-sprites", projectId, "equipment-icons", iconNames.join(",")]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(iconNames);
    },
    { revalidateOnFocus: false },
  );
  const frames = production.data?.resource_frames ?? {};

  const variantAt = useMemo(
    () =>
      row.data
        ? measureWindow(row.data.node, row.data.sprites, ROW_W, SLOT_H)[
            "name:variant"
          ]
        : undefined,
    [row.data],
  );

  if (!active || !production.data) return null;
  const grid = panelRects["name:equipments_grid"];
  const list = panelRects["name:equipments"];
  const queue = rects["name:item_grid"];
  const lines = rects["name:production_lines"];
  const queueLines = production.data.queue?.lines ?? [];
  const resourceGrid = rects["name:resources_grid"];
  const resourceGridNode = nodeNamed(win.node, "resources_grid");

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {topResource.data &&
        resourceGrid &&
        stripEntries(frames, stats).map(([resource, amount], at) => (
          <ResourceChip
            key={resource}
            entry={topResource.data as GuiWindow}
            frame={frames[resource] ?? 1}
            amount={amount}
            color={amountColor(amount)}
            left={
              resourceGrid.x + stripSlot(resourceGridNode, at, TOP_RESOURCE_W).x
            }
            top={
              resourceGrid.y + stripSlot(resourceGridNode, at, TOP_RESOURCE_W).y
            }
            width={TOP_RESOURCE_W}
            height={30}
          />
        ))}

      {queue && (
        <div
          aria-label="production queue"
          style={{
            position: "absolute",
            left: queue.x,
            top: queue.y + QUEUE_MARGIN.top,
            width: 520,
            height: Math.max(
              0,
              (lines ? lines.y + lines.h - QUEUE_MARGIN.bottom : 1080) -
                queue.y -
                QUEUE_MARGIN.top,
            ),
            overflowY: "auto",
            overflowX: "hidden",
            ...GAME_SCROLLBAR,
            pointerEvents: "auto",
          }}
        >
          <div
            style={{
              position: "relative",
              height: (queueLines.length + 1) * LINE_H,
            }}
          >
            <QueueLines
              projectId={projectId}
              lines={queueLines}
              frames={frames}
              onInspectEquipment={onEditEquipment}
              resourceChip={(resource, frame, amount, left, top) =>
                topResource.data ? (
                  <ResourceChip
                    key={resource}
                    entry={topResource.data}
                    frame={frame}
                    amount={amount}
                    left={left}
                    top={top}
                    width={48}
                    height={27}
                  />
                ) : null
              }
            />
            {newLine.data && (
              <button
                type="button"
                aria-label="production_military_new_line_entry"
                onClick={() => setOpen((was) => !was)}
                style={{
                  position: "absolute",
                  left: 0,
                  top: queueLines.length * LINE_H,
                  width: 500,
                  height: 108,
                  padding: 0,
                  border: "none",
                  background: "transparent",
                  cursor: "inherit",
                  pointerEvents: "auto",
                }}
              >
                <GuiCanvas
                  node={newLine.data.node}
                  {...edit(
                    newLine.data.name,
                    newLine.data,
                    newLine.data.node,
                    newLine.data.sprites,
                    NO_TEXT,
                  )}
                  texts={newLine.data.texts}
                  fonts={newLine.data.fonts}
                  hidden={NOTHING_HIDDEN}
                  isolateId={null}
                  transparent
                  baseW={500}
                  baseH={108}
                />

                {onEditEquipment && (
                  <span
                    role="button"
                    aria-label={t("openCatalogue")}
                    onClick={(e) => {
                      e.stopPropagation();
                      setOpen((was) => !was);
                    }}
                    style={{
                      position: "absolute",
                      inset: 0,
                      cursor: "pointer",
                    }}
                  />
                )}
              </button>
            )}
          </div>
        </div>
      )}

      {FILTERS.map(({ button, group }) => {
        const rect = rects[`name:${button}`];
        if (!rect) return null;
        const on = groups.includes(group);
        return (
          <button
            key={button}
            type="button"
            aria-label={button}
            aria-pressed={on}
            onClick={() =>
              setGroups((was) =>
                was.includes(group)
                  ? was.filter((other) => other !== group)
                  : [...was, group],
              )
            }
            style={{
              position: "absolute",
              left: rect.x,
              top: rect.y,
              width: rect.w || 45,
              height: rect.h || 32,
              padding: 0,
              border: "none",
              background: "transparent",
              cursor: "inherit",
              pointerEvents: "auto",

              opacity: on ? 1 : 0.45,
            }}
          />
        );
      })}

      {open && panel.data && panelNode && (
        <div style={{ position: "absolute", left: showAt.x, top: showAt.y }}>
          <GuiCanvas
            node={panelNode}
            {...edit(
              panel.data.name,
              panel.data,
              panelNode,
              panel.data.sprites,
              panelBuilt?.texts ?? NO_TEXT,
            )}
            texts={panel.data.texts}
            fonts={panel.data.fonts}
            hidden={NOTHING_HIDDEN}
            isolateId={null}
            transparent
            baseW={PANEL_W}
            baseH={1080}
          />
          {panelRects["name:checkbox"] && (
            <button
              type="button"
              aria-label="outdated_units"
              aria-pressed={showOutdated}
              onClick={() => setShowOutdated((was) => !was)}
              style={{
                position: "absolute",
                left: panelRects["name:checkbox"].x,
                top: panelRects["name:checkbox"].y,
                width: panelRects["name:checkbox"].w || 28,
                height: panelRects["name:checkbox"].h || 28,
                padding: 0,
                border: "none",
                background: "transparent",
                cursor: "inherit",
                pointerEvents: "auto",
              }}
            />
          )}
          {panelRects["name:close"] && (
            <button
              type="button"
              aria-label="close"
              onClick={() => setOpen(false)}
              style={{
                position: "absolute",
                left: panelRects["name:close"].x,
                top: panelRects["name:close"].y,
                width: panelRects["name:close"].w || 28,
                height: panelRects["name:close"].h || 28,
                padding: 0,
                border: "none",
                background: "transparent",
                cursor: "inherit",
                pointerEvents: "auto",
              }}
            />
          )}

          {row.data && grid && list && (
            <div
              style={{
                position: "absolute",
                left: grid.x,
                top: grid.y,
                width: ROW_W,

                height: Math.max(
                  0,
                  Math.min(
                    list.h - (grid.y - list.y),
                    1080 - showAt.y - grid.y,
                  ) - LIST_MARGIN,
                ),
                overflowY: "auto",
                overflowX: "hidden",
                ...GAME_SCROLLBAR,
                pointerEvents: "auto",
              }}
            >
              <div
                style={{ position: "relative", height: items.length * SLOT_H }}
              >
                {items.map((item, at) => (
                  <EquipmentRow
                    key={`${item.design ? "design" : "type"}:${item.name}`}
                    entry={row.data as GuiWindow}
                    resourceEntry={rowResource.data}
                    item={item}
                    icon={item.icon ? icons.data?.[item.icon] : undefined}
                    frames={frames}
                    left={0}
                    top={at * SLOT_H}
                  />
                ))}
                {onEditEquipment &&
                  items.map((item, at) => (
                    <FieldHit
                      key={`edit-${item.design ? "design" : "type"}:${item.name}`}
                      x={0}
                      y={at * SLOT_H}
                      w={ROW_W}
                      h={SLOT_H}
                      label={t(item.design ? "design" : "equipment", {
                        name: item.label || item.name,
                      })}
                      onClick={() =>
                        item.design && item.designer && onOpenDesigner
                          ? onOpenDesigner(item.name, item.designer)
                          : onEditEquipment(
                              item.design
                                ? (item.type ?? item.name)
                                : item.name,
                            )
                      }
                    />
                  ))}
                {onOpenDesigner &&
                  variantAt &&
                  items.map((item, at) =>
                    item.designer ? (
                      <div
                        key={`design-${item.design ? "design" : "type"}:${item.name}`}
                        role="button"
                        aria-label={`design ${item.name}`}
                        onClick={(e) => {
                          e.stopPropagation();
                          onOpenDesigner(item.name, item.designer as string);
                        }}
                        style={{
                          position: "absolute",
                          left: variantAt.x,
                          top: at * SLOT_H + variantAt.y,
                          width: variantAt.w || 34,
                          height: variantAt.h || 34,
                          cursor: "pointer",
                          pointerEvents: "auto",
                        }}
                      />
                    ) : null,
                  )}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
