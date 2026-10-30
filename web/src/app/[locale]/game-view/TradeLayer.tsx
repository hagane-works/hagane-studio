"use client";

import { useCallback, useMemo, useState } from "react";
import useSWR from "swr";
import { guiArray, guiRecord, readGuiResponse } from "./guiResponse";
import {
  type GuiCountryStats,
  type GuiNode,
  type GuiWindow,
  withGameLanguage,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { RichText } from "../gui-renderer/RichText";
import { textColors } from "@/lib/referenceTables";
import { useEditCanvas } from "./editContext";
import { useTemplateWindow } from "./CountrySelectLayer";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export const TRADE = "countrytradeview";

const COLUMN = "resources_info_entry";
const CHIP = "resources_filter_entry";
const ROW = "country_trade_entry";

const COLUMN_W = 79;
const COLUMN_H = 156;
const COLUMN_AT = { x: 101, y: 0 };

const COLUMNS_PER_ROW = 7;
const RESOURCES_BOX = { w: 655, h: 210 };

const CHIP_W = 79;
const ROW_H = 45;
const ROW_W = 648;

const SCREEN_H = 1080;

const WHEEL_ROWS = 3;

const NO_TEXT: Record<string, string> = {};
const NOTHING_HIDDEN = new Set<string>();

export interface GuiTradeColumn {
  resource: string;
  frame: number;

  produced: number;
  exported: number;
  imported?: number;
  projects?: number;
  production?: number;
  surplus?: number;
}

export function tradeCells(
  column: GuiTradeColumn,
): Record<string, { text: string; color?: string }> {
  const { G: green, R: red } = textColors();
  const incoming = (amount: number) => ({
    text: String(amount),
    color: amount > 0 ? green : undefined,
  });
  const outgoing = (amount: number) => ({
    text: amount > 0 ? `-${amount}` : "0",
    color: amount > 0 ? red : undefined,
  });
  const surplus = column.surplus ?? column.produced - column.exported;
  return {
    balance: {
      text: String(surplus),
      color: surplus > 0 ? green : surplus < 0 ? red : undefined,
    },
    produced: incoming(column.produced),
    imported: incoming(column.imported ?? 0),
    exported: outgoing(column.exported),
    special_project: outgoing(column.projects ?? 0),
    production_line: outgoing(column.production ?? 0),
    needed: { text: String(surplus), color: surplus < 0 ? red : undefined },
  };
}

export interface GuiTradeSeller {
  tag: string;
  name?: string;
  flag?: { url?: string; vanilla_url?: string };
  available: Record<string, number>;
  total: number;
  overlord?: string;
}

export interface GuiTrade {
  tag: string;
  trade_law?: string;
  trade_law_label?: string;
  export_share?: number;
  resources: GuiTradeColumn[];
  resource_frames?: Record<string, number>;
  sellers: GuiTradeSeller[];

  factories?: { total_text: string; used_text: string; available_text: string };
}

function useTrade(projectId: number, tag: string | null, date: string | null) {
  return useSWR<GuiTrade>(
    tag ? ["game-view-trade", projectId, tag, date] : null,
    async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/trade?tag=${encodeURIComponent(tag as string)}`,
        ) + (date ? `&date=${encodeURIComponent(date)}` : ""),
      );
      return readGuiResponse<GuiTrade>(response);
    },
    { revalidateOnFocus: false },
  );
}

const RUNTIME_ONLY = new Set(["cancel_imports"]);

function stacked(entry: GuiWindow, frames: Readonly<Record<string, number>>) {
  const node = structuredClone(entry.node) as GuiNode;
  const walk = (parent: GuiNode) => {
    for (const child of parent.children ?? []) {
      if (child.name && RUNTIME_ONLY.has(child.name)) child.sprite = null;
      if (child.name && frames[child.name] !== undefined)
        child.frame = frames[child.name];
      walk(child);
    }
  };
  walk(node);
  assignIds(node);
  return node;
}

function ResourceColumn({
  entry,
  column,
  left,
  top,
}: {
  entry: GuiWindow;
  column: GuiTradeColumn;
  left: number;
  top: number;
}) {
  const edit = useEditCanvas();
  const node = useMemo(
    () => stacked(entry, { resources_icon: column.frame }),
    [entry, column.frame],
  );
  const rects = useMemo(
    () => measureWindow(node, entry.sprites, 1920, SCREEN_H),
    [node, entry.sprites],
  );
  const fonts = entry.fonts;
  const nodeFont = (name: string) => {
    const found = entry.node.children?.find((child) => child.name === name);
    return found?.font ? fonts[found.font]?.bitmap : undefined;
  };
  const balanceFont = nodeFont("balance");
  const rowFont = nodeFont("produced");
  const cells = tradeCells(column);
  const value = (name: string, font = rowFont) => {
    const rect = rects[`name:${name}`];
    const cell = cells[name];
    if (!rect || !font || !cell) return null;
    const box = entry.node.children?.find((child) => child.name === name);
    return (
      <BitmapText
        key={name}
        font={font}
        text={cell.text}
        color={cell.color}
        left={rect.x}
        top={rect.y}
        width={box?.maxwidth ?? 50}
        height={font.line + 4}
        align="right"
        vcenter={false}
        maxLines={1}
      />
    );
  };

  return (
    <div style={{ position: "absolute", left, top }}>
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
        baseW={COLUMN_W}
        baseH={170}
      />

      {value("balance", balanceFont)}
      {[
        "produced",
        "imported",
        "exported",
        "special_project",
        "production_line",
        "needed",
      ].map((name) => value(name))}
    </div>
  );
}

function SellerRow({
  entry,
  seller,
  amount,
  left,
  top,
  onOpen,
}: {
  entry: GuiWindow;
  seller: GuiTradeSeller;
  amount: number;
  left: number;
  top: number;
  onOpen?: () => void;
}) {
  const edit = useEditCanvas();

  const node = useMemo(() => {
    const built = stacked(entry, {});
    for (const child of built.children ?? []) {
      if (child.name === "country" || child.name === "available_export")
        child.text = "";
    }
    return built;
  }, [entry]);
  const rects = useMemo(
    () => measureWindow(node, entry.sprites, 1920, SCREEN_H),
    [node, entry.sprites],
  );
  const nameNode = entry.node.children?.find(
    (child) => child.name === "country",
  );
  const exportNode = entry.node.children?.find(
    (child) => child.name === "available_export",
  );
  const nameFont = nameNode?.font
    ? entry.fonts[nameNode.font]?.bitmap
    : undefined;
  const exportFont = exportNode?.font
    ? entry.fonts[exportNode.font]?.bitmap
    : undefined;
  const nameRect = rects["name:country"];
  const exportRect = rects["name:available_export"];
  const flagRect = rects["name:country_flag"];
  const flag = seller.flag?.vanilla_url || seller.flag?.url;

  return (
    <div style={{ position: "absolute", left, top }}>
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
      {flagRect && onOpen && (
        <button
          type="button"
          aria-label={"open trade country " + seller.tag}
          onClick={onOpen}
          style={{
            position: "absolute",
            left: flagRect.x,
            top: flagRect.y,
            width: flagRect.w || 41,
            height: flagRect.h || 26,
            padding: 0,
            border: 0,
            background: "transparent",
            pointerEvents: "auto",
          }}
        />
      )}
      {nameFont && nameRect && (
        <BitmapText
          font={nameFont}
          text={seller.name || seller.tag}
          left={nameRect.x}
          top={nameRect.y}
          width={nameNode?.maxwidth ?? 130}
          height={nameNode?.maxheight ?? 28}
          align="left"
          vcenter
          maxLines={1}
        />
      )}
      {exportFont && exportRect && (
        <BitmapText
          font={exportFont}
          text={String(amount)}
          left={exportRect.x}
          top={exportRect.y}
          width={120}
          height={exportFont.line + 4}
          align="center"
          vcenter={false}
          maxLines={1}
        />
      )}
    </div>
  );
}

export function TradeLayer({
  win,
  projectId,
  tag,
  date,
  stats,
  onOpenCountry,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  stats?: GuiCountryStats;
  onOpenCountry?: (tag: string) => void;
}) {
  const edit = useEditCanvas();
  const active = win.name === TRADE;
  const trade = useTrade(projectId, active ? tag : null, date ?? null);
  const column = useTemplateWindow(projectId, active ? COLUMN : null);
  const chip = useTemplateWindow(projectId, active ? CHIP : null);
  const row = useTemplateWindow(projectId, active ? ROW : null);

  const [picked, setPicked] = useState<string | null>(null);

  const fastWheel = useCallback((list: HTMLDivElement | null) => {
    if (!list) return;
    const onWheel = (e: WheelEvent) => {
      if (e.ctrlKey) return;
      const step =
        e.deltaMode === 1
          ? e.deltaY * ROW_H
          : Math.abs(e.deltaY) >= 50
            ? Math.sign(e.deltaY) * WHEEL_ROWS * ROW_H
            : e.deltaY;
      if (!step) return;
      e.preventDefault();
      list.scrollTop += step;
    };
    list.addEventListener("wheel", onWheel, { passive: false });
    return () => list.removeEventListener("wheel", onWheel);
  }, []);

  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, SCREEN_H) : {}),
    [active, win],
  );
  const font = useMemo(() => {
    const name =
      Object.keys(win.fonts).find((key) => key.includes("18mbs")) ??
      Object.keys(win.fonts)[0];
    return name ? win.fonts[name]?.bitmap : undefined;
  }, [win.fonts]);

  const columns = useMemo(() => guiArray(trade.data?.resources), [trade.data]);

  const chipNodes = useMemo(() => {
    if (!chip.data) return {};
    const out: Record<string, GuiNode> = {};
    for (const entry of columns)
      out[entry.resource] = stacked(chip.data, { icon: entry.frame });
    return out;
  }, [chip.data, columns]);
  const sellers = useMemo(() => {
    const rows = guiArray(trade.data?.sellers);
    if (!picked) return rows;
    return rows
      .filter((seller) => (guiRecord(seller.available)[picked] ?? 0) > 0)
      .sort(
        (a, b) =>
          (guiRecord(b.available)[picked] ?? 0) -
          (guiRecord(a.available)[picked] ?? 0),
      );
  }, [trade.data, picked]);

  if (!active || !trade.data) return null;
  const resourcesBox = rects["name:resources"];
  const filtersBox = rects["name:filters"];
  const tradeBox = rects["name:trade"];
  const totalRect = rects["name:factories_total"];
  const usedRect = rects["name:factories_used"];
  const availableRect = rects["name:factories_available"];
  const civ = stats?.civilian_factories;
  const chipsRight = (resourcesBox?.x ?? 11) + RESOURCES_BOX.w;

  const shownAt = (win.node.raw?.show_position ?? win.node.position ?? {}) as {
    y?: number | string;
  };
  const screenBottom = SCREEN_H - (Number(shownAt.y) || 0);
  const chipsScroll = filtersBox
    ? columns.length * CHIP_W > chipsRight - filtersBox.x
    : false;

  const count = (at: { x: number; y: number } | undefined, amount: number) =>
    font && at ? (
      <BitmapText
        font={font}
        text={String(amount)}
        left={at.x}
        top={at.y}
        width={45}
        height={font.line + 4}
        align="left"
        vcenter={false}
        maxLines={1}
      />
    ) : null;

  const header = (
    at: { x: number; y: number } | undefined,
    markup: string,
    width: number,
    align: "left" | "center",
  ) =>
    font && at ? (
      <RichText
        font={font}
        markup={markup}
        left={at.x}
        top={at.y}
        width={width}
        height={font.line + 4}
        align={align}
        vcenter={false}
        maxLines={1}
      />
    ) : null;

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {trade.data.factories ? (
        <>
          {header(totalRect, trade.data.factories.total_text, 45, "left")}
          {header(usedRect, trade.data.factories.used_text, 120, "center")}
          {header(
            availableRect,
            trade.data.factories.available_text,
            120,
            "center",
          )}
        </>
      ) : (
        civ !== undefined && count(totalRect, civ)
      )}

      {column.data && resourcesBox && (
        <div
          data-trade-resources
          style={{
            position: "absolute",
            left: resourcesBox.x + COLUMN_AT.x,
            top: resourcesBox.y + COLUMN_AT.y,
            width: RESOURCES_BOX.w - COLUMN_AT.x,
            height: RESOURCES_BOX.h - COLUMN_AT.y,
            overflowX: "hidden",
            overflowY: columns.length > COLUMNS_PER_ROW ? "auto" : "hidden",
            ...GAME_SCROLLBAR,
            pointerEvents: columns.length > COLUMNS_PER_ROW ? "auto" : "none",
          }}
        >
          <div
            style={{
              position: "relative",
              height: Math.ceil(columns.length / COLUMNS_PER_ROW) * COLUMN_H,
            }}
          >
            {columns.map((entry, at) => (
              <ResourceColumn
                key={entry.resource}
                entry={column.data as GuiWindow}
                column={entry}
                left={(at % COLUMNS_PER_ROW) * COLUMN_W}
                top={Math.floor(at / COLUMNS_PER_ROW) * COLUMN_H}
              />
            ))}
          </div>
        </div>
      )}

      {chip.data && filtersBox && (
        <div
          data-trade-filters
          onWheel={(e) => {
            e.currentTarget.scrollLeft += e.deltaY || e.deltaX;
          }}
          style={{
            position: "absolute",
            left: filtersBox.x,
            top: filtersBox.y,
            width: Math.max(CHIP_W, chipsRight - filtersBox.x),
            height: 50,
            overflowX: chipsScroll ? "auto" : "hidden",
            overflowY: "hidden",
            scrollbarWidth: "none",

            pointerEvents: chipsScroll ? "auto" : "none",
          }}
        >
          <div
            style={{
              position: "relative",
              width: columns.length * CHIP_W,
              height: 50,
            }}
          >
            {columns.map((entry, at) => (
              <button
                key={entry.resource}
                type="button"
                aria-label={`filter_${entry.resource}`}
                aria-pressed={picked === entry.resource}
                onClick={() =>
                  setPicked((was) =>
                    was === entry.resource ? null : entry.resource,
                  )
                }
                style={{
                  position: "absolute",
                  left: at * CHIP_W,
                  top: 0,
                  width: CHIP_W - 5,
                  height: 50,
                  padding: 0,
                  border: "none",
                  background: "transparent",
                  cursor: "inherit",
                  pointerEvents: "auto",
                  opacity: !picked || picked === entry.resource ? 1 : 0.45,
                }}
              >
                <GuiCanvas
                  node={chipNodes[entry.resource]}
                  {...edit(
                    `${(chip.data as GuiWindow).name}:${entry.resource}`,
                    chip.data as GuiWindow,
                    chipNodes[entry.resource],
                    (chip.data as GuiWindow).sprites,
                    NO_TEXT,
                  )}
                  texts={(chip.data as GuiWindow).texts}
                  fonts={(chip.data as GuiWindow).fonts}
                  hidden={NOTHING_HIDDEN}
                  isolateId={null}
                  transparent
                  baseW={CHIP_W}
                  baseH={50}
                />
              </button>
            ))}
          </div>
        </div>
      )}

      {row.data && tradeBox && (
        <div
          ref={fastWheel}
          data-trade-sellers
          style={{
            position: "absolute",
            left: tradeBox.x,
            top: tradeBox.y,
            width: ROW_W,
            height: Math.max(0, screenBottom - tradeBox.y - 13),
            overflowY: "auto",
            overflowX: "hidden",

            ...GAME_SCROLLBAR,
            pointerEvents: "auto",
          }}
        >
          <div style={{ position: "relative", height: sellers.length * ROW_H }}>
            {sellers.map((seller, at) => (
              <SellerRow
                key={seller.tag}
                entry={row.data as GuiWindow}
                seller={seller}
                amount={
                  picked
                    ? (guiRecord(seller.available)[picked] ?? 0)
                    : seller.total
                }
                left={0}
                top={at * ROW_H}
                onOpen={
                  onOpenCountry ? () => onOpenCountry(seller.tag) : undefined
                }
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
