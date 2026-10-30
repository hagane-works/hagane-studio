"use client";

import { useMemo } from "react";
import useSWR from "swr";
import { readGuiResponse } from "./guiResponse";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { wrapLines } from "../gui-renderer/BitmapText";
import { useTemplateWindow } from "./CountrySelectLayer";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";
import { GAME_SCROLLBAR } from "./PageScrollbar";

export interface GuiOutputItem {
  name: string;
  label: string;
  icon?: string;
  stats: { stat: string; label: string; value: string }[];
  resources: { resource: string; amount: number; frame: number }[];
}

export interface GuiProjectPage {
  found: boolean;
  id: string;
  label: string;
  label_key: string;
  description: string;
  description_key: string;
  icon: string;
  outputs: string[];
  modules: GuiOutputItem[];
  equipment: GuiOutputItem[];

  discounts: {
    name: string;
    equipment: string;
    label: string;
    uses: number;
    cost_factor: number;
  }[];
  texts: {
    output_header: string;
    rewards_header: string;
    modules_header: string;
    equipment_header: string;
  };
  sprites: Record<string, GuiSprite>;
}

export function useProjectPage(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
  id: string | null,
) {
  return useSWR<GuiProjectPage>(
    tag && id
      ? ["game-view-project-page", projectId, tag, date ?? "", id]
      : null,
    async () => {
      const query = new URLSearchParams({
        tag: tag as string,
        id: id as string,
      });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/special-project?${query}`,
        ),
      );
      return readGuiResponse<GuiProjectPage>(response);
    },
    { revalidateOnFocus: false },
  );
}

const PAGE = "active_research";
const OUTPUT = "output_window";
const SECTIONS = "detailed_output_item";
const MODULE = "technology_unit_statlist_item_module";
const EQUIPMENT = "technology_unit_statlist_item_equipment";
const STAT = "technology_unit_stat_item";

export const PAGE_AT = { x: 0, y: 43 };
const PAGE_SIZE = { w: 635, h: 520 };

const OUTPUT_AT = { x: 98, y: 160 };
const OUTPUT_SIZE = { w: 550, h: 375 };

const LIST_AT = { x: 37, y: 88 };
const LIST_SIZE = { w: 480, h: 275 };
const LAYOUT = {
  rewards: 10,
  text: 52,
  afterText: 33,
  header: 30,
  grid: 95,
  gridX: 25,
  row: 30,
  column: 224,
  afterGrid: 25,
  end: 39,

  statY: 2,
};
const ITEM_SIZE = { w: 480, h: 100 };
const STAT_SIZE = { w: 224, h: 25 };

const EQUIPMENT_HIDDEN = [
  "design_team_icon",
  "equipment_icon",
  "carrier_fighter_icon",
  "3d_container",
  "leader_bonuses_header",
  "support_category_multipliers_header",
];
const MODULE_HIDDEN = [
  "leader_bonuses_header",
  "support_category_multipliers_header",
];

type Section = {
  kind: "modules" | "equipment";
  header: number;
  items: { item: GuiOutputItem; y: number; rows: number }[];
};

export function projectPageLayout(
  page: Pick<GuiProjectPage, "outputs" | "modules" | "equipment">,
  lines: number,
  lineHeight = 16,
) {
  let y = page.outputs.length
    ? LAYOUT.text + lines * lineHeight + LAYOUT.afterText
    : LAYOUT.rewards;
  const sections: Section[] = [];
  for (const kind of ["modules", "equipment"] as const) {
    const items = page[kind];
    if (!items.length) continue;
    const section: Section = { kind, header: y, items: [] };
    y += LAYOUT.header;
    for (const item of items) {
      const rows = Math.ceil(item.stats.length / 2);
      section.items.push({ item, y, rows });
      y += LAYOUT.grid + rows * LAYOUT.row + LAYOUT.afterGrid;
    }
    sections.push(section);
  }
  const height = sections.length ? y - LAYOUT.afterGrid + LAYOUT.end : y;
  return { sections, height };
}

export function resourcesText(resources: GuiOutputItem["resources"]): string {
  return resources
    .map((item) => `${item.amount}£resources_strip|${item.frame}`)
    .join(" ");
}

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function outputWindow(entry: GuiWindow): GuiWindow {
  const node = structuredClone(entry.node) as GuiNode;
  if (node.background?.sprite) {
    node.children = [
      { ...node.background, type: "icontype", children: [] },
      ...(node.children ?? []),
    ];
    node.background = null;
  }
  assignIds(node);
  return { ...entry, node };
}

export function sectionsWindow(
  entry: GuiWindow,
  page: GuiProjectPage,
  sections: Section[],
): GuiWindow {
  const node = structuredClone(entry.node) as GuiNode;
  const at: Record<string, number> = {};
  if (page.outputs.length) {
    at.rewards_header = LAYOUT.rewards;
    at.country_effect_reward_description = LAYOUT.text;
  }
  for (const section of sections)
    at[
      section.kind === "modules"
        ? "unlocked_equipment_modules"
        : "unlocked_equipments"
    ] = section.header;
  node.children = (node.children ?? [])
    .filter((child) => child.name && child.name in at)
    .map(
      (child) =>
        ({
          ...child,
          position: { ...(child.position ?? {}), y: at[child.name as string] },
        }) as GuiNode,
    );
  assignIds(node);
  return { ...entry, node };
}

export function ProjectPage({
  projectId,
  tag,
  date,
  id,
  origin,
  onBack,
}: {
  projectId: number;
  tag: string;
  date?: string | null;
  id: string;

  origin: { x: number; y: number };
  onBack: () => void;
}) {
  const data = useProjectPage(projectId, tag, date, id);
  const page = useTemplateWindow(projectId, PAGE);
  const output = useTemplateWindow(projectId, OUTPUT);
  const sections = useTemplateWindow(projectId, SECTIONS);
  const moduleItem = useTemplateWindow(projectId, MODULE);
  const equipment = useTemplateWindow(projectId, EQUIPMENT);
  const stat = useTemplateWindow(projectId, STAT);
  const back = useMemo(() => {
    if (!page.data) return undefined;
    const node = structuredClone(page.data.node) as GuiNode;
    assignIds(node);
    const measured = measureWindow(
      node,
      page.data.sprites,
      PAGE_SIZE.w,
      PAGE_SIZE.h,
    );
    return measured[
      idOf(matchNamePath(node, "start_project_window/back_button")[0])
    ] as { x: number; y: number; w: number; h: number } | undefined;
  }, [page.data]);
  const project = data.data;
  const font = sections.data?.fonts?.["hoi4_typewriter16_inverted"]?.bitmap;
  const lines = useMemo(() => {
    if (!project?.outputs.length) return 0;
    const text = project.outputs.join("\n");
    return font ? wrapLines(font, text, 464).length : project.outputs.length;
  }, [project, font]);
  const laid = useMemo(
    () =>
      project ? projectPageLayout(project, lines, font?.line || 16) : null,
    [project, lines, font],
  );
  const paper = useMemo(
    () => (output.data ? outputWindow(output.data) : null),
    [output.data],
  );
  const header = useMemo(
    () =>
      sections.data && project && laid
        ? sectionsWindow(sections.data, project, laid.sections)
        : null,
    [sections.data, project, laid],
  );
  if (
    !project?.found ||
    !page.data ||
    !paper ||
    !header ||
    !moduleItem.data ||
    !equipment.data ||
    !stat.data ||
    !laid
  )
    return null;
  const sprites = project.sprites;
  const left = origin.x + PAGE_AT.x;
  const top = origin.y + PAGE_AT.y;
  const list = {
    x: left + OUTPUT_AT.x + LIST_AT.x,
    y: top + OUTPUT_AT.y + LIST_AT.y,
  };
  return (
    <>
      <Filled
        entry={page.data}
        left={left}
        top={top}
        width={PAGE_SIZE.w}
        height={PAGE_SIZE.h}
        texts={{
          project_name: project.label,
          project_desc: project.description,
        }}
        pictures={project.icon ? { project_icon: project.icon } : {}}
        hide={[
          "research_completed_icon",
          "project_progress",
          "approx_research_time",
          "ongoing_project_info_window",
        ]}
        extraSprites={sprites}
      />
      <Filled
        entry={paper}
        left={left + OUTPUT_AT.x}
        top={top + OUTPUT_AT.y}
        width={OUTPUT_SIZE.w}
        height={OUTPUT_SIZE.h}
        texts={{ output_header: project.texts.output_header }}
        hide={[
          "multiple_selection_reward_window",
          "go_to_previous_page_button",
          "go_to_next_page_button",
        ]}
        extraSprites={sprites}
      />
      <div
        style={{
          position: "absolute",
          left: list.x,
          top: list.y,
          width: LIST_SIZE.w,
          height: LIST_SIZE.h,
          overflowY: "auto",
          overflowX: "hidden",
          ...GAME_SCROLLBAR,
          pointerEvents: "auto",
        }}
      >
        <div
          style={{
            position: "relative",
            height: laid.height,
            overflow: "hidden",
          }}
        >
          <Filled
            entry={header}
            left={0}
            top={0}
            width={LIST_SIZE.w}
            height={laid.height}
            texts={{
              rewards_header: project.texts.rewards_header,
              country_effect_reward_description: project.outputs.join("\n"),
              "unlocked_equipment_modules/header": project.texts.modules_header,
              "unlocked_equipments/header": project.texts.equipment_header,
            }}
            extraSprites={sprites}
          />
          {laid.sections.flatMap((section) =>
            section.items.map(({ item, y }) => (
              <div key={`${section.kind}/${item.name}`}>
                <Filled
                  entry={
                    (section.kind === "modules"
                      ? moduleItem.data
                      : equipment.data) as GuiWindow
                  }
                  left={0}
                  top={y}
                  width={ITEM_SIZE.w}
                  height={ITEM_SIZE.h}
                  texts={{
                    production_resources: resourcesText(item.resources),
                  }}
                  pictures={
                    item.icon
                      ? {
                          [section.kind === "modules"
                            ? "equipment_icon"
                            : "unit_icon"]: item.icon,
                        }
                      : {}
                  }
                  hide={
                    section.kind === "modules"
                      ? MODULE_HIDDEN
                      : EQUIPMENT_HIDDEN
                  }
                  extraSprites={sprites}
                />
                {item.stats.map((line, at) => (
                  <Filled
                    key={`${line.stat}/${at}`}
                    entry={stat.data as GuiWindow}
                    left={LAYOUT.gridX + (at % 2) * LAYOUT.column}
                    top={
                      y +
                      LAYOUT.grid +
                      Math.floor(at / 2) * LAYOUT.row +
                      LAYOUT.statY
                    }
                    width={STAT_SIZE.w}
                    height={STAT_SIZE.h}
                    texts={{ name: line.label, amount: line.value }}
                    hide={["diff"]}
                    extraSprites={sprites}
                  />
                ))}
              </div>
            )),
          )}
        </div>
      </div>
      {back && (
        <div
          role="button"
          aria-label="Back to projects"
          onClick={onBack}
          style={{
            position: "absolute",
            left: left + back.x,
            top: top + back.y,
            width: back.w || 160,
            height: back.h || 30,
            cursor: "pointer",
            pointerEvents: "auto",
          }}
        />
      )}
    </>
  );
}
