"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { FieldHit } from "./FieldHit";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";
import { readGuiResponse } from "./guiResponse";
import { SpriteBar, barFill, type BarSprite } from "./SpriteBar";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export const SUBJECTS_WINDOW = "manage_subjects_window";
export const EXILES_WINDOW = "exiles_window";
const ENTRY = "subject_item";

const SLOT_H = 76;
const ENTRY_W = 494;
const ENTRY_H = 87;

const LIST_MARGIN_BOTTOM = 15;

const BAR = { w: 93, h: 16 };
const FLAG = { w: 82, h: 52 };

const LEVEL_HIT = { x: 305, y: 12, w: 189, h: 48 };

const LAYERED = [
  "autonomy_country_flag",
  "large_flag_frame",
  "autonomy_progress",
  "prev_level_glow",
  "next_level_glow",
];

export interface GuiSubjectStep {
  id: string;
  label: string;
  icon: string;
}

export interface GuiSubject {
  tag: string;
  name: string;
  level: GuiSubjectStep;
  down: GuiSubjectStep;
  up: GuiSubjectStep;
  freedom: number;
  flag?: { url?: string; vanilla_url?: string };
}

export interface GuiAutonomyLevel {
  id: string;
  label: string;
  file: string;
  min_freedom_level: number;
  manpower_influence?: number;
  rule: string;
  modifier: string;
}

export interface GuiSubjects {
  tag: string;
  subjects: GuiSubject[];
  sprites: Record<string, GuiSprite & { texture_urls?: string[] }>;

  levels: Record<string, GuiAutonomyLevel>;
}

export function subjectRows(data: GuiSubjects | undefined): GuiSubject[] {
  return Array.isArray(data?.subjects) ? data.subjects : [];
}

export function useSubjects(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useSWR<GuiSubjects>(
    tag ? ["game-view-subjects", projectId, tag, date ?? ""] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/subjects?${query}`,
        ),
      );
      return readGuiResponse<GuiSubjects>(response);
    },
    { revalidateOnFocus: false },
  );
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

function slidListWindow(win: GuiWindow, listName: string): { win: GuiWindow } {
  const node = structuredClone(win.node) as GuiNode;
  assignIds(node);
  for (const list of matchNamePath(node, listName)) {
    const show = list.raw?.show_position as
      { x?: number | string; y?: number | string } | undefined;
    if (show) list.position = { x: show.x ?? 0, y: show.y ?? 0 };
  }
  return { win: { ...win, node } };
}

export function subjectsWindow(win: GuiWindow): { win: GuiWindow } {
  return slidListWindow(win, "subject_items_window");
}

export function exilesWindow(win: GuiWindow): { win: GuiWindow } {
  return slidListWindow(win, "exile_items_window");
}

export function freedomWidth(freedom: number): number {
  return barFill(BAR.w, freedom);
}

type Rect = { x: number; y: number; w: number; h: number };

export function SubjectsLayer({
  win,
  projectId,
  tag,
  date,
  onEditLevel,
  onOpenCountry,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  editMode?: boolean;

  onEditLevel?: (id: string) => void;

  onOpenCountry?: (tag: string) => void;
}) {
  const tw = useTranslations("gameViewWorld.autonomy");
  const active = win.name === SUBJECTS_WINDOW;
  const data = useSubjects(projectId, active ? tag : null, date);
  const entry = useTemplateWindow(projectId, active ? ENTRY : null);
  const listAt = useMemo(() => {
    if (!active) return null;
    const { win: shown } = subjectsWindow(win);
    const rects = measureWindow(shown.node, shown.sprites, 1920, 1080);
    const rect = (path: string) =>
      rects[idOf(matchNamePath(shown.node, path)[0])] as Rect | undefined;
    const grid = rect("subject_items_window/subjects_grid");
    const root = rects[idOf(shown.node)] as Rect | undefined;
    const list = rect("subject_items_window");
    if (!grid) return null;
    const bottom =
      list && list.h > 0
        ? list.y + list.h
        : root
          ? root.y + root.h
          : grid.y + 540;
    return {
      x: grid.x,
      y: grid.y,
      h: Math.max(0, bottom - grid.y - LIST_MARGIN_BOTTOM),
    };
  }, [active, win]);

  const parts = useMemo(() => {
    if (!entry.data) return null;
    const node = structuredClone(entry.data.node) as GuiNode;
    assignIds(node);
    const rects = measureWindow(node, entry.data.sprites, ENTRY_W, ENTRY_H);
    const rect = (name: string) =>
      rects[idOf(matchNamePath(node, name)[0])] as Rect | undefined;
    const frameNode = matchNamePath(node, "large_flag_frame")[0];
    const frame: GuiWindow | null = frameNode
      ? {
          ...entry.data,
          name: "large_flag_frame",
          node: {
            ...(structuredClone(frameNode) as GuiNode),
            position: { x: 0, y: 0 },
          },
        }
      : null;
    return {
      flag: rect("autonomy_country_flag"),
      frameAt: rect("large_flag_frame"),
      bar: rect("autonomy_progress"),
      frame,
    };
  }, [entry.data]);
  if (!active || !data.data || !entry.data || !listAt || !parts) return null;
  const subjects = subjectRows(data.data);
  const barSprite = entry.data.sprites["GFX_subject_progressbar"] as
    BarSprite | undefined;
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      <div
        style={{
          position: "absolute",
          left: listAt.x,
          top: listAt.y,
          width: ENTRY_W + 4,
          height: listAt.h,
          overflowY: "auto",
          overflowX: "hidden",
          ...GAME_SCROLLBAR,
          pointerEvents: "auto",
        }}
      >
        <div
          style={{
            position: "relative",
            height: (subjects.length - 1) * SLOT_H + ENTRY_H,
          }}
        >
          {subjects.map((subject, at) => {
            const top = at * SLOT_H;
            const flag = subject.flag?.vanilla_url || subject.flag?.url;
            return (
              <div
                key={subject.tag}
                style={{
                  position: "absolute",
                  left: 0,
                  top,
                  width: ENTRY_W,
                  height: ENTRY_H,
                }}
              >
                <Filled
                  entry={entry.data as GuiWindow}
                  left={0}
                  top={0}
                  width={ENTRY_W}
                  height={ENTRY_H}
                  texts={{
                    autonomy_country_name: subject.name,
                    current_autonomy_name: subject.level.label,
                  }}
                  pictures={{
                    current_autonomy_level_icon: subject.level.icon,
                    prev_autonomy_level_icon: subject.down.icon,
                    next_autonomy_level_icon: subject.up.icon,
                  }}
                  hide={LAYERED}
                  extraSprites={data.data?.sprites ?? {}}
                />
                {flag && parts.flag && (
                  <div
                    aria-label={`flag ${subject.tag}`}
                    style={{
                      position: "absolute",
                      left: parts.flag.x,
                      top: parts.flag.y,
                      width: parts.flag.w || FLAG.w,
                      height: parts.flag.h || FLAG.h,
                      backgroundImage: `url("${flag}")`,
                      backgroundSize: "100% 100%",
                      backgroundRepeat: "no-repeat",
                    }}
                  />
                )}
                {parts.flag && onOpenCountry && (
                  <button
                    type="button"
                    aria-label={"open country " + subject.tag}
                    onClick={() => onOpenCountry(subject.tag)}
                    style={{
                      position: "absolute",
                      left: parts.flag.x,
                      top: parts.flag.y,
                      width: parts.flag.w || FLAG.w,
                      height: parts.flag.h || FLAG.h,
                      padding: 0,
                      border: 0,
                      background: "transparent",
                      pointerEvents: "auto",
                    }}
                  />
                )}
                {parts.frame && parts.frameAt && (
                  <Filled
                    entry={parts.frame}
                    left={parts.frameAt.x}
                    top={parts.frameAt.y}
                    width={parts.frameAt.w || FLAG.w + 10}
                    height={parts.frameAt.h || FLAG.h + 6}
                    texts={{}}
                    extraSprites={{}}
                  />
                )}
                {parts.bar && (
                  <SpriteBar
                    x={parts.bar.x}
                    y={parts.bar.y}
                    size={BAR}
                    sprite={barSprite}
                    fraction={subject.freedom}
                  />
                )}
                {onEditLevel && (
                  <FieldHit
                    x={LEVEL_HIT.x}
                    y={LEVEL_HIT.y}
                    w={LEVEL_HIT.w}
                    h={LEVEL_HIT.h}
                    label={tw("hit", {
                      tag: subject.tag,
                      name: subject.level.label,
                    })}
                    onClick={() => onEditLevel(subject.level.id)}
                  />
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
