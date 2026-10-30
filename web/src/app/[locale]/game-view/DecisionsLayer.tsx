"use client";

import { useCallback, useContext, useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { readGuiResponse } from "./guiResponse";
import {
  guiRenderApi,
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
import { BitmapText, wrapLines } from "../gui-renderer/BitmapText";
import { RichText } from "../gui-renderer/RichText";
import { useTemplateWindow } from "./CountrySelectLayer";
import { EditContext, useEditCanvas } from "./editContext";
import { namedPrinted } from "./textSources";
import { FieldHit } from "./FieldHit";
import type { GuiBitmapFont, GuiNode } from "@/lib/guiRenderApi";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export const DECISIONS = "countrydecisionview";

const EVENTS = "event_header";
const HEADER = "category_header";
const DESC = "decision_category_desc";
const ROW = "decision_item";
const EVENTS_H = 74;
const HEADER_H = 58;
const ROW_H = 40;
const END_H = 20;

const WHEEL_ROWS = 3;

const RUNTIME_ONLY = new Set([
  "btn_progress_good",
  "btn_progress_bad",
  "target_flag",
  "target_flag_frame",
]);

const EVENT_HEADER_FRAMES: Record<string, number> = {
  popup_events_checkbox: 2,
  popup_news_checkbox: 2,
};
const ROW_FRAMES: Record<string, number> = { btn_select: 1 };

export interface GuiDecision {
  name: string;
  label?: string;
  icon?: string;
  cost?: number;
  days_re_enable?: number;
  fire_only_once?: boolean;
  targeted: boolean;

  shown?: boolean;

  mission?: boolean;

  origin?: string;

  label_key?: string;

  days_remove?: number;

  icon_token?: string;

  file?: string;
}

export interface GuiDecisionCategory {
  name: string;
  label?: string;

  desc?: string;
  icon?: string;
  picture?: string;
  priority: number;
  conditional?: boolean;

  shown?: boolean;

  origin?: string;
  decisions: GuiDecision[];
}

export type DecisionPick =
  | { kind: "decision"; category: string; name: string }
  | { kind: "category"; category: string };

export type DecisionEdits = Record<
  string,
  { label?: string; cost?: number; icon?: string }
>;

export const decisionEditKey = (pick: DecisionPick) =>
  pick.kind === "decision" ? `d:${pick.name}` : `c:${pick.category}`;

export function useDecisions(
  projectId: number,
  tag: string | null,
  date?: string | null,
) {
  return useSWR<{ tag: string; categories: GuiDecisionCategory[] }>(
    tag ? ["game-view-decisions", projectId, tag, date ?? ""] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/decisions?${query}`,
        ),
      );
      return readGuiResponse<{
        tag: string;
        categories: GuiDecisionCategory[];
      }>(response);
    },
    { revalidateOnFocus: false },
  );
}

export type ListEntry =
  | { kind: "events"; key: string; top: number; height: number }
  | {
      kind: "category";
      key: string;
      pick: DecisionPick;
      label: string;

      textKey: string;
      icon?: string;

      dim?: boolean;
      top: number;
      height: number;
    }
  | {
      kind: "desc";
      key: string;
      picture?: string;
      text: string;
      textKey: string;
      dim?: boolean;
      top: number;
      height: number;
    }
  | {
      kind: "decision";
      key: string;
      pick: DecisionPick;
      label: string;
      textKey: string;
      icon?: string;
      cost?: number;
      dim?: boolean;
      top: number;
      height: number;
    }
  | { kind: "end"; key: string; top: number; height: number };

export function stackRows(
  categories: GuiDecisionCategory[],
  saved: DecisionEdits = {},
  descHeight: (category: GuiDecisionCategory) => number = () => 0,
  showHidden = false,
  collapsed: ReadonlySet<string> = new Set(),
): ListEntry[] {
  const rows: ListEntry[] = [
    { kind: "events", key: "events", top: 0, height: EVENTS_H },
  ];
  let top = EVENTS_H;
  for (const category of categories) {
    const tabHidden = category.shown === false;
    if (tabHidden && !showHidden) continue;
    const own = saved[`c:${category.name}`];
    rows.push({
      kind: "category",
      key: `c:${category.name}`,
      pick: { kind: "category", category: category.name },
      label: own?.label || category.label || category.name,
      textKey: category.name,

      icon: own?.icon || category.icon,
      dim: tabHidden,
      top,
      height: HEADER_H,
    });
    top += HEADER_H;
    if (collapsed.has(category.name)) continue;
    const descH = descHeight(category);
    if (descH > 0) {
      rows.push({
        kind: "desc",
        key: `p:${category.name}`,
        picture: category.picture,
        text: category.desc ?? "",
        textKey: `${category.name}_desc`,
        dim: tabHidden,
        top,
        height: descH,
      });
      top += descH;
    }
    for (const decision of category.decisions) {
      const hidden = tabHidden || decision.shown === false;
      if (hidden && !showHidden) continue;
      const edit = saved[`d:${decision.name}`];
      rows.push({
        kind: "decision",
        key: `d:${category.name}:${decision.name}`,
        pick: {
          kind: "decision",
          category: category.name,
          name: decision.name,
        },
        label: edit?.label || decision.label || decision.name,
        textKey: decision.label_key || decision.name,
        icon: edit?.icon || decision.icon,
        cost: edit?.cost ?? decision.cost,
        dim: hidden,
        top,
        height: ROW_H,
      });
      top += ROW_H;
    }
    rows.push({ kind: "end", key: `e:${category.name}`, top, height: END_H });
    top += END_H;
  }
  return rows;
}

const plainText = (markup: string) =>
  markup.replace(/§[A-Za-z!]/g, "").replace(/£\S+\s?/g, "");

export function descBlockHeight(
  category: GuiDecisionCategory,
  picture: GuiSprite | undefined,
  font: GuiBitmapFont | undefined,
): number {
  const text = category.desc ?? "";
  if (!category.picture && !text) return 0;
  const textH =
    font && text
      ? wrapLines(font, plainText(text), category.picture ? 380 : 470).length *
        font.line
      : 0;
  if (category.picture)
    return Math.max((picture?.height || 101) + 6, textH + 15);
  return textH + 12;
}

function prepared(entry: GuiWindow, change: (child: GuiNode) => void): GuiNode {
  const clone = structuredClone(entry.node) as GuiNode;
  const walk = (node: GuiNode) => {
    for (const child of node.children ?? []) {
      change(child);
      walk(child);
    }
  };
  walk(clone);
  assignIds(clone);
  return clone;
}

function ListRow({
  entry,
  rowKey,
  label,
  textKey,
  cost,
  iconName,
  icon,
  left,
  top,
  onPick,
  collapsed,
  onToggleCategory,
}: {
  entry: GuiWindow;
  rowKey: string;
  label: string;

  textKey: string;
  cost?: number;
  iconName?: string;
  icon: GuiSprite | undefined;
  left: number;
  top: number;

  onPick?: () => void;
  collapsed?: boolean;
  onToggleCategory?: () => void;
}) {
  const t = useTranslations("gameViewContent.decision");
  const edit = useEditCanvas();
  const { editMode: uiEdit } = useContext(EditContext);
  const node = useMemo(
    () =>
      prepared(entry, (child) => {
        if (child.name && RUNTIME_ONLY.has(child.name)) child.sprite = null;
        if (child.name && child.name in ROW_FRAMES)
          child.frame = ROW_FRAMES[child.name];

        if (child.name === "icon" && icon && iconName) child.sprite = iconName;
      }),
    [entry, icon, iconName],
  );
  const sprites = useMemo(
    () =>
      icon && iconName ? { ...entry.sprites, [iconName]: icon } : entry.sprites,
    [entry.sprites, icon, iconName],
  );

  const blanks = useMemo(() => {
    const out: Record<string, string> = {};
    for (const child of (node.children ?? []) as (GuiNode & {
      _id?: string;
    })[]) {
      if (
        child._id &&
        (child.name === "cost_and_timer_text" || child.name === "name_text")
      ) {
        out[child._id] = "";
      }
    }
    return out;
  }, [node]);

  const printed = useMemo(
    () => namedPrinted(node, "name_text", label, textKey),
    [node, label, textKey],
  );
  const empty = useMemo(() => new Set<string>(), []);
  const rects = useMemo(
    () => measureWindow(node, sprites, 1920, 1080),
    [node, sprites],
  );
  const nameNode = (entry.node.children ?? []).find(
    (child) => child.name === "name_text",
  );
  const costNode = (entry.node.children ?? []).find(
    (child) => child.name === "cost_and_timer_text",
  );
  const font = nameNode?.font ? entry.fonts[nameNode.font]?.bitmap : undefined;
  const nameRect = rects["name:name_text"];
  const costRect = rects["name:cost_and_timer_text"];
  const collapseRect = rects["name:category_collapse_btn"];

  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${rowKey}`,
          { ...entry, sprites },
          node,
          sprites,
          blanks,
          printed,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={empty}
        isolateId={null}
        transparent
        baseW={502}
        baseH={cost === undefined ? HEADER_H : ROW_H}
      />
      {!uiEdit && onToggleCategory && collapseRect && (
        <button
          type="button"
          aria-label={"toggle decision category " + rowKey}
          aria-expanded={!collapsed}
          onClick={onToggleCategory}
          style={{
            position: "absolute",
            left: collapseRect.x,
            top: collapseRect.y,
            width: collapseRect.w || 24,
            height: collapseRect.h || 24,
            padding: 0,
            border: "none",
            background: "transparent",
            cursor: "inherit",
            pointerEvents: "auto",
          }}
        />
      )}
      {font && nameRect && (
        <BitmapText
          font={font}
          text={label}
          left={nameRect.x}
          top={nameRect.y}
          width={nameNode?.maxwidth || 330}
          height={nameNode?.maxheight || 32}
          align={textAlignOf(nameNode?.format)}
          vcenter={
            String(nameNode?.raw?.vertical_alignment ?? "").toLowerCase() ===
            "center"
          }
          maxLines={2}
        />
      )}
      {font && costRect && cost !== undefined && cost > 0 && (
        <RichText
          font={font}
          markup={`£pol_power ${Math.round(cost)}`}
          left={costRect.x}
          top={costRect.y}
          width={costNode?.maxwidth || 320}
          align="right"
          vcenter={false}
          maxLines={1}
        />
      )}
      {onPick && nameRect && (
        <FieldHit
          x={nameRect.x}
          y={nameRect.y}
          w={nameNode?.maxwidth || 330}
          h={nameNode?.maxheight || 32}
          label={t(cost === undefined ? "hitCategory" : "hitDecision", {
            name: label,
          })}
          onClick={onPick}
        />
      )}
      {onPick && costRect && cost !== undefined && (
        <FieldHit
          x={costRect.x + (costNode?.maxwidth || 320) - 70}
          y={costRect.y}
          w={70}
          h={costNode?.maxheight || 24}
          label={t("hitCost", { name: label })}
          onClick={onPick}
        />
      )}
    </div>
  );
}

function PlainRow({
  entry,
  rowKey,
  frames,
  left,
  top,
  height,
}: {
  entry: GuiWindow;
  rowKey: string;
  frames: Record<string, number>;
  left: number;
  top: number;
  height: number;
}) {
  const edit = useEditCanvas();
  const node = useMemo(
    () =>
      prepared(entry, (child) => {
        if (child.name && child.name in frames)
          child.frame = frames[child.name];
      }),
    [entry, frames],
  );
  const empty = useMemo(() => new Set<string>(), []);
  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(`${entry.name}:${rowKey}`, entry, node, entry.sprites)}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={empty}
        isolateId={null}
        transparent
        baseW={515}
        baseH={height}
      />
    </div>
  );
}

function DescRow({
  entry,
  rowKey,
  pictureName,
  picture,
  text,
  textKey,
  left,
  top,
  height,
}: {
  entry: GuiWindow;
  rowKey: string;
  pictureName?: string;
  picture: GuiSprite | undefined;
  text: string;

  textKey: string;
  left: number;
  top: number;
  height: number;
}) {
  const edit = useEditCanvas();
  const node = useMemo(() => {
    const clone = prepared(entry, (child) => {
      if (child.name === "picture")
        child.sprite = pictureName && picture ? pictureName : null;
    });
    clone.size = { ...(clone.size ?? {}), height };
    return clone;
  }, [entry, pictureName, picture, height]);
  const sprites = useMemo(
    () =>
      picture && pictureName
        ? { ...entry.sprites, [pictureName]: picture }
        : entry.sprites,
    [entry.sprites, picture, pictureName],
  );
  const empty = useMemo(() => new Set<string>(), []);
  const box = (entry.node.children ?? []).find(
    (child) => child.name === (pictureName ? "short_text" : "full_text"),
  );
  const font = box?.font ? entry.fonts[box.font]?.bitmap : undefined;
  const at = box?.position;

  const printed = useMemo(
    () =>
      namedPrinted(
        node,
        pictureName ? "short_text" : "full_text",
        text,
        textKey,
      ),
    [node, pictureName, text, textKey],
  );
  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${rowKey}`,
          { ...entry, sprites },
          node,
          sprites,
          undefined,
          printed,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={empty}
        isolateId={null}
        transparent
        baseW={510}
        baseH={height}
      />
      {font && text && (
        <RichText
          font={font}
          markup={text}
          left={Number(at?.x) || (pictureName ? 125 : 14)}
          top={Number(at?.y) || (pictureName ? 9 : 6)}
          width={box?.maxwidth || (pictureName ? 380 : 470)}
          align="left"
          vcenter={false}
          maxLines={12}
        />
      )}
    </div>
  );
}

async function spritesInChunks(
  projectId: number,
  names: string[],
): Promise<Record<string, GuiSprite>> {
  guiRenderApi.setProjectId(projectId);
  const chunks: string[][] = [];
  for (let i = 0; i < names.length; i += 100)
    chunks.push(names.slice(i, i + 100));
  const parts = await Promise.all(
    chunks.map((chunk) => guiRenderApi.sprites(chunk)),
  );
  return Object.assign({}, ...parts);
}

export function DecisionsLayer({
  win,
  projectId,
  tag,
  date,
  saved,
  onPick,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;

  date?: string | null;

  saved?: DecisionEdits;

  onPick?: (pick: DecisionPick) => void;
}) {
  const active = win.name === DECISIONS;
  const decisions = useDecisions(projectId, active ? tag : null, date);
  const events = useTemplateWindow(projectId, active ? EVENTS : null);
  const header = useTemplateWindow(projectId, active ? HEADER : null);
  const desc = useTemplateWindow(projectId, active ? DESC : null);
  const row = useTemplateWindow(projectId, active ? ROW : null);
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  const categories = useMemo(
    () => decisions.data?.categories ?? [],
    [decisions.data],
  );
  const iconNames = useMemo(() => {
    const names = new Set<string>();
    for (const category of categories) {
      const own = saved?.[`c:${category.name}`]?.icon;
      for (const name of [own, category.icon, category.picture])
        if (name) names.add(name);
      for (const decision of category.decisions) {
        const icon = saved?.[`d:${decision.name}`]?.icon || decision.icon;
        if (icon) names.add(icon);
      }
    }
    return [...names].sort();
  }, [categories, saved]);
  const icons = useSWR<Record<string, GuiSprite>>(
    iconNames.length
      ? ["game-view-sprites", projectId, "decision-icons", iconNames.join(",")]
      : null,
    () => spritesInChunks(projectId, iconNames),
    { revalidateOnFocus: false },
  );
  const descFont = useMemo(() => {
    const box = (desc.data?.node.children ?? []).find(
      (child) => child.name === "short_text",
    );
    return box?.font ? desc.data?.fonts[box.font]?.bitmap : undefined;
  }, [desc.data]);
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set());
  const rows = useMemo(
    () =>
      stackRows(
        categories,
        saved,
        (category) =>
          descBlockHeight(
            category,
            category.picture ? icons.data?.[category.picture] : undefined,
            descFont,
          ),

        true,
        collapsed,
      ),
    [categories, saved, icons.data, descFont, collapsed],
  );
  const [scroll, setScroll] = useState(0);

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
  const grid = rects["name:decision_grid"];
  const container = rects["name:decision_grid_container"];
  if (
    !active ||
    !decisions.data ||
    !grid ||
    !events.data ||
    !header.data ||
    !desc.data ||
    !row.data
  )
    return null;

  const shownAt = (win.node.raw?.show_position ?? win.node.position ?? {}) as {
    y?: number | string;
  };
  const screenBottom = 1080 - (Number(shownAt.y) || 0);
  const viewH = Math.max(
    0,
    Math.min(
      screenBottom,
      (container?.y ?? grid.y) + (container?.h ?? screenBottom),
    ) - grid.y,
  );
  const total = rows.length
    ? rows[rows.length - 1].top + rows[rows.length - 1].height
    : 0;
  const maxScroll = Math.max(0, total - viewH);
  const at = Math.min(scroll, maxScroll);
  const visible = rows.filter(
    (entry) => entry.top + entry.height > at && entry.top < at + viewH,
  );

  const eventsWin = events.data;
  const headerWin = header.data;
  const descWin = desc.data;
  const rowWin = row.data;
  const draw = (entry: ListEntry, top: number) => {
    switch (entry.kind) {
      case "events":
        return (
          <PlainRow
            entry={eventsWin}
            rowKey={entry.key}
            frames={EVENT_HEADER_FRAMES}
            left={0}
            top={top}
            height={entry.height}
          />
        );
      case "desc":
        return (
          <DescRow
            entry={descWin}
            rowKey={entry.key}
            pictureName={entry.picture}
            picture={entry.picture ? icons.data?.[entry.picture] : undefined}
            text={entry.text}
            textKey={entry.textKey}
            left={0}
            top={top}
            height={entry.height}
          />
        );
      case "end":
        return null;
      default:
        return (
          <ListRow
            entry={entry.kind === "category" ? headerWin : rowWin}
            rowKey={entry.key}
            label={entry.label}
            textKey={entry.textKey}
            cost={entry.kind === "category" ? undefined : (entry.cost ?? 0)}
            iconName={entry.icon}
            icon={entry.icon ? icons.data?.[entry.icon] : undefined}
            left={0}
            top={top}
            onPick={onPick ? () => onPick(entry.pick) : undefined}
            collapsed={
              entry.kind === "category" && collapsed.has(entry.pick.category)
            }
            onToggleCategory={
              entry.kind === "category"
                ? () =>
                    setCollapsed((current) => {
                      const next = new Set(current);
                      if (next.has(entry.pick.category))
                        next.delete(entry.pick.category);
                      else next.add(entry.pick.category);
                      return next;
                    })
                : undefined
            }
          />
        );
    }
  };

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      <div
        ref={fastWheel}
        data-decision-list
        style={{
          position: "absolute",
          left: grid.x,
          top: grid.y,
          width: 515,
          height: viewH,
          overflowY: "auto",
          overflowX: "hidden",

          ...GAME_SCROLLBAR,
          pointerEvents: "auto",
        }}
        onScroll={(e) => setScroll(e.currentTarget.scrollTop)}
      >
        <div style={{ position: "relative", height: total }}>
          {visible.map((entry) => (
            <div
              key={entry.key}
              style={
                "dim" in entry && entry.dim ? { opacity: 0.45 } : undefined
              }
            >
              {draw(entry, entry.top)}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
