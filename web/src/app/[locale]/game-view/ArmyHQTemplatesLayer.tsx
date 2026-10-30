"use client";

import { useMemo } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { gameTooltipProps } from "../gui-renderer/GameTooltip";
import { fillRuntimeSlots } from "../gui-renderer/tooltipText";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";
import { matchNamePath } from "./gameNavigation";
import { useOfficerCorps } from "./OfficerCorpsLayer";

export const ARMY_HQ_TEMPLATES = "armyhqtemplatewindow";
const ENTRY = "named_hq_template_entry";

export interface GuiHQTemplate {
  name: string;
  icon?: string;
  priority?: number;
  command_power?: number;
  deploy_days?: number;
}

const TEXTS = {
  days: "HQ_TEMPLATE_DEPLOY_DAYS_SHORT",
  daysTip: "HQ_DEPLOY_AND_WITHDRAW_DAYS",
  dayTip: "HQ_DEPLOY_AND_WITHDRAW_DAYS_SINGULAR",
  costTip: "HQ_DEPLOY_CP_COST",
} as const;

function useRowTexts(projectId: number, enabled: boolean) {
  return useSWR<Record<string, string>>(
    enabled ? ["game-view-loc", projectId, "hq-template-rows"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(Object.values(TEXTS), { markup: true });
    },
    { revalidateOnFocus: false },
  );
}

export function hqRowTexts(
  template: GuiHQTemplate,
  texts: Record<string, string>,
) {
  const days = template.deploy_days ?? 0;
  return {
    days: texts[TEXTS.days]
      ? fillRuntimeSlots(texts[TEXTS.days], { DAYS: days })
      : `${days} days`,
    daysTip: fillRuntimeSlots(
      texts[days === 1 ? TEXTS.dayTip : TEXTS.daysTip] ?? "",
      { DAYS: days },
    ),
    cost: String(Math.round(template.command_power ?? 0)),
    costTip: fillRuntimeSlots(texts[TEXTS.costTip] ?? "", {
      COST: Math.round(template.command_power ?? 0),
    }),
  };
}

function Row({
  entry,
  template,
  texts,
  sprites,
  left,
  top,
  onEdit,
}: {
  entry: GuiWindow;
  template: GuiHQTemplate;
  texts: Record<string, string>;
  sprites: Record<string, GuiSprite>;
  left: number;
  top: number;
  onEdit?: () => void;
}) {
  const line = hqRowTexts(template, texts);
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;
    for (const child of node.children ?? []) {
      if (child.name === "counter" && template.icon) {
        child.sprite = template.icon;
        child.frame = null;
      }
      if (child.name === "division_priority_icon")
        child.frame = (template.priority ?? 1) + 1;
    }
    assignIds(node);
    const overrides: Record<string, string> = {};
    const put = (name: string, text: string) => {
      for (const hit of matchNamePath(node, name) as (GuiNode & {
        _id?: string;
      })[]) {
        if (hit._id) overrides[hit._id] = text;
      }
    };
    put("name", template.name);
    put("deploy_time", line.days);
    put("cp_value", line.cost);
    return { node, overrides };
  }, [entry.node, template, line.days, line.cost]);
  const all = useMemo(
    () => ({ ...entry.sprites, ...sprites }),
    [entry.sprites, sprites],
  );
  const rects = useMemo(
    () => measureWindow(built.node, all, 1920, 1080),
    [built.node, all],
  );
  const edit = useEditCanvas();
  const size = entry.node.size as { width?: number; height?: number } | null;
  const w = Number(size?.width) || 341;
  const h = Number(size?.height) || 95;
  const box = (name: string) => rects[`name:${name}`];
  const hover = (name: string, tip: string) => {
    const rect = box(name);
    if (!rect || !tip) return null;
    return (
      <div
        key={name}
        {...gameTooltipProps(tip, {
          key: `hq-template:${template.name}:${name}`,
        })}
        style={{
          position: "absolute",
          left: rect.x,
          top: rect.y,
          width: rect.w || 24,
          height: rect.h || 24,
          pointerEvents: "auto",
        }}
      />
    );
  };
  const editRect = box("edit_button");
  return (
    <div style={{ position: "absolute", left, top, width: w, height: h }}>
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${template.name}`,
          entry,
          built.node,
          all,
          built.overrides,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={w}
        baseH={h}
      />
      {hover("deploy_time_icon", line.daysTip)}
      {hover("cp_icon", line.costTip)}
      {onEdit && editRect && (
        <button
          type="button"
          aria-label={`edit ${template.name}`}
          onClick={onEdit}
          style={{
            position: "absolute",
            left: editRect.x,
            top: editRect.y,
            width: editRect.w || 71,
            height: editRect.h || 26,
            background: "transparent",
            border: "none",
            padding: 0,
            cursor: "inherit",
            pointerEvents: "auto",
          }}
        />
      )}
    </div>
  );
}

const NO_HIDDEN: Set<string> = new Set();

export function ArmyHQTemplatesLayer({
  win,
  projectId,
  tag,
  date,
  onEdit,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;

  onEdit?: (name: string) => void;
}) {
  const active = win.name === ARMY_HQ_TEMPLATES;
  const corps = useOfficerCorps(projectId, active ? tag : null, date ?? null);
  const entry = useTemplateWindow(projectId, active ? ENTRY : null);
  const texts = useRowTexts(projectId, active);
  const templates = useMemo<GuiHQTemplate[]>(
    () => corps.data?.hq_templates ?? [],
    [corps.data],
  );
  const icons = useMemo(
    () =>
      [
        ...new Set(
          templates
            .map((template) => template.icon)
            .filter(Boolean) as string[],
        ),
      ].sort(),
    [templates],
  );
  const sprites = useSWR<Record<string, GuiSprite>>(
    icons.length
      ? ["game-view-sprites", projectId, "hq-template-icons", icons.join(",")]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(icons);
    },
    { revalidateOnFocus: false },
  );
  const rects = useMemo(
    () => (active ? measureWindow(win.node, win.sprites, 1920, 1080) : {}),
    [active, win],
  );
  if (!active || !entry.data || !texts.data) return null;
  const grid = rects["name:templates_grid"];
  if (!grid) return null;
  const pitch =
    Number((entry.data.node.size as { height?: number } | null)?.height) || 95;
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {templates.map((template, at) => (
        <Row
          key={template.name}
          entry={entry.data as GuiWindow}
          template={template}
          texts={texts.data ?? {}}
          sprites={sprites.data ?? {}}
          left={grid.x}
          top={grid.y + at * pitch}
          onEdit={onEdit ? () => onEdit(template.name) : undefined}
        />
      ))}
    </div>
  );
}
