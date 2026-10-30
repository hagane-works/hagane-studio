"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { FieldHit } from "./FieldHit";
import { Filled } from "./FilledWindow";
import { useSpecialProjects, type GuiFacility } from "./FacilitiesLayer";
import { matchNamePath } from "./gameNavigation";
import { guiArray, guiRecord } from "./guiResponse";
import { ProjectPage } from "./ProjectPageLayer";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export const PROGRAM_WINDOW = "program_window";
const ENTRY = "projects_entry";
const COST = "breakthrough_cost_item";

const SLOT = { w: 500, h: 130 };
const ENTRY_SIZE = { w: 511, h: 143 };

const COST_SLOT_H = 36;
const LOCKED_ICON = "GFX_unavailable_project";

const WINDOW_ORIGIN = { x: 0, y: 0 };

const CARD_HIDDEN = [
  "project_time",
  "research_progressbar_frame_icon",
  "research_progressbar",
  "research_speed_window",
];

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function programTitle(
  facility: Pick<GuiFacility, "label" | "state_name">,
): string {
  return `${facility.label} in ${facility.state_name}`;
}

export function programWindow(
  win: GuiWindow,
  facility: GuiFacility | null,
  pageOpen = false,
): { win: GuiWindow; texts: Record<string, string> } {
  const node = structuredClone(win.node) as GuiNode;
  if (pageOpen)
    node.children = (node.children ?? []).filter(
      (child) => child.name !== "projects",
    );
  assignIds(node);
  const texts: Record<string, string> = {};
  if (facility)
    for (const hit of matchNamePath(node, "frame/title"))
      if (idOf(hit)) texts[idOf(hit)] = programTitle(facility);
  return { win: { ...win, node }, texts };
}

type Rect = { x: number; y: number; w: number; h: number };

const PARTS = [
  "available_projects_list/project_entries",
  "available_projects_list",
  "frame/close_button",
];

export function ProgramWindowLayer({
  win,
  projectId,
  tag,
  date,
  facility,
  onClose,
  editMode = false,
  onEditProject,
  page = null,
  onOpenPage,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  facility: GuiFacility | null;
  onClose: () => void;
  editMode?: boolean;

  onEditProject?: (id: string) => void;

  page?: string | null;

  onOpenPage?: (id: string | null) => void;
}) {
  const tLayer = useTranslations("gameViewResearch.layer");
  const active = win.name === PROGRAM_WINDOW;
  const data = useSpecialProjects(projectId, active ? tag : null, date);
  const entry = useTemplateWindow(projectId, active ? ENTRY : null);
  const cost = useTemplateWindow(projectId, active ? COST : null);
  const rects = useMemo(() => {
    const out: Record<string, Rect | undefined> = {};
    if (!active) return out;
    const node = structuredClone(win.node) as GuiNode;
    const measured = measureWindow(node, win.sprites, 1920, 1080);
    for (const path of PARTS)
      out[path] = measured[idOf(matchNamePath(node, path)[0])] as
        Rect | undefined;
    return out;
  }, [active, win]);
  const costTab = useMemo(() => {
    if (!entry.data) return undefined;
    const node = structuredClone(entry.data.node) as GuiNode;
    assignIds(node);
    const measured = measureWindow(
      node,
      entry.data.sprites,
      ENTRY_SIZE.w,
      ENTRY_SIZE.h,
    );
    return measured[idOf(matchNamePath(node, "breakthrough_cost_tab")[0])] as
      Rect | undefined;
  }, [entry.data]);
  if (!active || !data.data || !entry.data || !cost.data || !facility)
    return null;
  const grid = rects["available_projects_list/project_entries"];
  const box = rects["available_projects_list"];
  const close = rects["frame/close_button"];
  const all = guiArray(data.data.projects).filter(
    (project) => project.specialization === facility.specialization,
  );
  const projects = [
    ...all.filter((project) => project.available),
    ...all.filter((project) => !project.available),
  ];
  const sprites = guiRecord(data.data.sprites);
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {page && (
        <ProjectPage
          projectId={projectId}
          tag={tag}
          date={date}
          id={page}
          origin={WINDOW_ORIGIN}
          onBack={() => onOpenPage?.(null)}
        />
      )}
      {!page && grid && box && (
        <div
          style={{
            position: "absolute",
            left: grid.x,
            top: grid.y,
            width: ENTRY_SIZE.w + 4,
            height: Math.max(0, box.y + box.h - grid.y),
            overflowY: "auto",
            overflowX: "hidden",
            ...GAME_SCROLLBAR,
            pointerEvents: "auto",
          }}
        >
          <div
            style={{
              position: "relative",
              height: Math.max(0, projects.length - 1) * SLOT.h + ENTRY_SIZE.h,
            }}
          >
            {projects.map((project, at) => {
              const top = at * SLOT.h;
              return (
                <div
                  key={project.id}
                  style={{
                    position: "absolute",
                    left: 0,
                    top,
                    width: ENTRY_SIZE.w,
                    height: ENTRY_SIZE.h,
                  }}
                >
                  <Filled
                    entry={entry.data as GuiWindow}
                    left={0}
                    top={0}
                    width={ENTRY_SIZE.w}
                    height={ENTRY_SIZE.h}
                    texts={{
                      project_name: project.label,
                      project_name_default: project.generic ?? "",
                      project_type: project.specialization_name,
                    }}
                    pictures={{
                      project_icon:
                        project.available && project.icon
                          ? project.icon
                          : LOCKED_ICON,
                    }}
                    hide={
                      project.available
                        ? CARD_HIDDEN
                        : [
                            ...CARD_HIDDEN,
                            "scientist_info",
                            "breakthrough_cost_tab",
                          ]
                    }
                    extraSprites={sprites}
                  />
                  {project.available &&
                    costTab &&
                    guiArray(project.breakthrough_cost).map((item, index) => (
                      <Filled
                        key={item.specialization}
                        entry={cost.data as GuiWindow}
                        left={costTab.x}
                        top={costTab.y + index * COST_SLOT_H}
                        width={40}
                        height={20}
                        texts={{ cost_amount: String(item.amount) }}
                        pictures={{
                          breakthrough_tab_icon: "GFX_buildings_strip",
                        }}
                        frames={
                          item.icon_frame
                            ? { breakthrough_tab_icon: item.icon_frame }
                            : undefined
                        }
                        extraSprites={sprites}
                      />
                    ))}
                  {onOpenPage && !editMode && (
                    <button
                      type="button"
                      aria-label={project.label}
                      onClick={() => onOpenPage(project.id)}
                      style={{
                        position: "absolute",
                        inset: 0,
                        padding: 0,
                        border: "none",
                        background: "transparent",
                        cursor: "inherit",
                        pointerEvents: "auto",
                      }}
                    />
                  )}
                  {editMode && onEditProject && (
                    <FieldHit
                      x={0}
                      y={0}
                      w={SLOT.w}
                      h={SLOT.h}
                      label={tLayer("specialProject", { name: project.label })}
                      onClick={() => onEditProject(project.id)}
                    />
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
      {close && (
        <div
          role="button"
          aria-label="Close projects"
          onClick={onClose}
          style={{
            position: "absolute",
            left: close.x,
            top: close.y,
            width: close.w || 30,
            height: close.h || 30,
            cursor: "pointer",
            pointerEvents: "auto",
          }}
        />
      )}
    </div>
  );
}
