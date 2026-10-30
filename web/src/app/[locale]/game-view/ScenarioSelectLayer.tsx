"use client";

import { useMemo, useState } from "react";
import {
  gameLocale,
  type GuiBookmark,
  type GuiNode,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { useEditCanvas, usePrintedOver } from "./editContext";
import { useBookmarks, useTemplateWindow } from "./CountrySelectLayer";
import { scenarioPrinted } from "./textSources";

export const SCENARIO_PICKER = "gamesetup_scenario_window";

const BOOKMARK_ENTRY = "bookmark_entry";

const TILE_PITCH = 260;

const TILE_W = 232;
const TILE_H = 211;

const MIN_TILE_SCALE = 0.55;

export function scenarioRowLayout(
  count: number,
  available: number,
): { scale: number; scroll: boolean } {
  if (count <= 0 || available <= 0) return { scale: 1, scroll: false };
  const fit = available / ((count - 1) * TILE_PITCH + TILE_W);
  if (fit >= 1) return { scale: 1, scroll: false };
  if (fit >= MIN_TILE_SCALE) return { scale: fit, scroll: false };
  return { scale: MIN_TILE_SCALE, scroll: true };
}

const MONTHS = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

export function formatBookmarkDate(
  raw: string | undefined,
  locale: string = gameLocale(),
): string {
  const parts = (raw ?? "").split(".");
  const year = Number(parts[0]);
  const month = Number(parts[1]);
  const day = Number(parts[2]);
  if (!year || !month || !day || month > 12) return raw ?? "";
  if (locale !== "en") {
    try {
      return new Intl.DateTimeFormat(locale, {
        year: "numeric",
        month: "long",
        day: "numeric",
        timeZone: "UTC",
      }).format(new Date(Date.UTC(year, month - 1, day)));
    } catch {}
  }
  return `${day} ${MONTHS[month - 1]} ${year}`;
}

function findNode(root: GuiNode, name: string): GuiNode | undefined {
  if (root.name === name) return root;
  for (const child of root.children ?? []) {
    const found = findNode(child, name);
    if (found) return found;
  }
  return undefined;
}

function ScenarioTile({
  template,
  bookmark,
  left,
  scale,
  selected,
  onSelect,
}: {
  template: GuiWindow;
  bookmark: GuiBookmark;
  left: number;
  scale: number;
  selected: boolean;
  onSelect: () => void;
}) {
  const edit = useEditCanvas();
  const node = useMemo(() => {
    const clone = structuredClone(template.node) as GuiNode;
    for (const child of clone.children ?? []) {
      if (child.name === "icon" && bookmark.picture)
        child.sprite = bookmark.picture;
    }
    assignIds(clone);
    return clone;
  }, [template.node, bookmark.picture]);
  const empty = useMemo(() => new Set<string>(), []);
  const rects = useMemo(
    () => measureWindow(node, template.sprites, 1920, 1080),
    [node, template.sprites],
  );
  const dateNode = findNode(template.node, "date");
  const font = dateNode?.font
    ? template.fonts[dateNode.font]?.bitmap
    : undefined;
  const dateRect = rects["name:date"];

  const sprites = useMemo(() => {
    const picture = bookmark.picture;
    if (!picture || !bookmark.picture_sprite) return template.sprites;
    return { ...template.sprites, [picture]: bookmark.picture_sprite };
  }, [template.sprites, bookmark.picture, bookmark.picture_sprite]);

  return (
    <div
      style={{
        position: "absolute",
        left,
        top: 0,
        width: TILE_PITCH,
        height: TILE_H,
        opacity: selected ? 1 : 0.65,
        transform: scale === 1 ? undefined : `scale(${scale})`,
        transformOrigin: "0 0",
      }}
    >
      <GuiCanvas
        node={node}
        {...edit(`${template.name}:${left}`, template, node, sprites)}
        texts={template.texts}
        fonts={template.fonts}
        hidden={empty}
        isolateId={null}
        transparent
        baseW={TILE_PITCH}
        baseH={TILE_H}
      />
      {font && dateRect && (
        <BitmapText
          font={font}
          text={formatBookmarkDate(bookmark.date)}
          left={dateRect.x}
          top={dateRect.y}
          width={190}
          height={24}
          align="center"
          vcenter={false}
          maxLines={1}
        />
      )}
      <button
        type="button"
        aria-label={bookmark.name}
        title={bookmark.label || bookmark.name}
        onClick={onSelect}
        style={{
          position: "absolute",
          left: 0,
          top: 0,
          width: TILE_PITCH,
          height: TILE_H,
          background: "transparent",
          border: selected ? "2px solid rgba(244, 178, 62, 0.8)" : "none",
          cursor: "inherit",
        }}
      />
    </div>
  );
}

export function ScenarioSelectLayer({
  win,
  projectId,
  onConfirm,
  onPicked,
}: {
  win: GuiWindow;
  projectId: number;

  onConfirm: (bookmarkIndex: number, date: string) => void;

  onPicked?: (bookmarkIndex: number) => void;
}) {
  const bookmarks = useBookmarks(projectId, win.name === SCENARIO_PICKER);
  const template = useTemplateWindow(
    projectId,
    bookmarks.data ? BOOKMARK_ENTRY : null,
  );
  const rects = useMemo(
    () => measureWindow(win.node, win.sprites, 1920, 1080),
    [win],
  );
  const [picked, setPicked] = useState(0);

  const shownMark =
    bookmarks.data?.bookmarks?.[picked] ?? bookmarks.data?.bookmarks?.[0];
  const printed = useMemo(
    () => scenarioPrinted(win, shownMark),
    [win, shownMark],
  );
  usePrintedOver(win, "scenario-select", printed);

  if (win.name !== SCENARIO_PICKER || !bookmarks.data) return null;
  const marks = bookmarks.data.bookmarks ?? [];
  const selected = marks[picked] ?? marks[0];
  const gridRect = rects["name:bookmarks_grid"];
  const selectRect = rects["name:select_button"];
  const titleNode = findNode(win.node, "bookmark_title");
  const descNode = findNode(win.node, "bookmark_desc");
  const titleFont = titleNode?.font
    ? win.fonts[titleNode.font]?.bitmap
    : undefined;
  const descFont = descNode?.font
    ? win.fonts[descNode.font]?.bitmap
    : undefined;
  const titleRect = rects["name:bookmark_title"];
  const descRect = rects["name:bookmark_desc"];

  const windowW =
    typeof win.node.size?.width === "number" ? win.node.size.width : 579;
  const available = gridRect
    ? Math.max(TILE_PITCH, windowW - 2 * gridRect.x)
    : 2 * TILE_PITCH;
  const row = scenarioRowLayout(marks.length, available);

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {template.data && gridRect && (
        <div
          style={{
            position: "absolute",
            left: gridRect.x,
            top: gridRect.y,
            width: row.scroll ? available : undefined,
            height: row.scroll ? TILE_H * row.scale + 12 : undefined,

            overflowX: row.scroll ? "auto" : undefined,
            overflowY: row.scroll ? "hidden" : undefined,
            pointerEvents: "auto",
          }}
        >
          {row.scroll && (
            <div
              style={{
                width: ((marks.length - 1) * TILE_PITCH + TILE_W) * row.scale,
                height: 1,
              }}
            />
          )}
          {marks.map((bookmark, at) => (
            <ScenarioTile
              key={bookmark.name}
              template={template.data as GuiWindow}
              bookmark={bookmark}
              left={at * TILE_PITCH * row.scale}
              scale={row.scale}
              selected={at === picked}
              onSelect={() => {
                setPicked(at);
                onPicked?.(at);
              }}
            />
          ))}
        </div>
      )}

      {titleFont && titleRect && selected && (
        <BitmapText
          font={titleFont}
          text={selected.label || selected.name}
          left={titleRect.x}
          top={titleRect.y}
          width={titleNode?.maxwidth || 340}
          height={28}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
      {descFont && descRect && selected?.desc && (
        <BitmapText
          font={descFont}
          text={selected.desc}
          left={descRect.x}
          top={descRect.y}
          width={descNode?.maxwidth || 540}
          height={140}
          align="left"
          vcenter={false}
          maxLines={6}
        />
      )}

      {selectRect && (
        <button
          type="button"
          aria-label="select_scenario"
          onClick={() => onConfirm(picked, selected?.date ?? "")}
          style={{
            position: "absolute",
            left: selectRect.x,
            top: selectRect.y,
            width: selectRect.w || 221,
            height: selectRect.h || 34,
            background: "transparent",
            border: "none",
            cursor: "inherit",
            pointerEvents: "auto",
          }}
        />
      )}
    </div>
  );
}
