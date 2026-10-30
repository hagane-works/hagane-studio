"use client";

import { useContext, useMemo, useState } from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiDivisions,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { BitmapText } from "../gui-renderer/BitmapText";
import { useTemplateWindow } from "./CountrySelectLayer";
import { expandButtonOf, frameStyle } from "./spriteFrame";
import { EditContext, useEditCanvas } from "./editContext";
import { guiArray } from "./guiResponse";
import { UiEditArt, artUrl } from "./UiEditArt";

export const TEMPLATE_DEPLOYMENT = "templatedeploymentwindow";

export const DEPLOYMENT_VIEW = "countrydeploymentview";

const DEPLOY_ENTRY = "deploy_entry";

const DEPLOY_ROW_PITCH = 48;

const PRIORITY_ROWS: readonly { key: string; sprite: string }[] = [
  { key: "REINFORCEMENT", sprite: "GFX_reinforcements_icon" },
  { key: "UPGRADES", sprite: "GFX_upgrades_icon" },
  { key: "SUPPLY_TRUCKS", sprite: "GFX_deploy_supply_icon" },
  { key: "OPERATIONS", sprite: "GFX_operations_deploy_icon" },
  { key: "GARRISONS", sprite: "GFX_garrison_deploy_icon" },
  { key: "RAIDS", sprite: "GFX_deploy_raids_icon" },
];

const TEMPLATE_ENTRY = "named_division_template_entry";

const FOREIGN_TEMPLATES = "foreign_templates_drop_down";

const ROW_PITCH = 79;

const RUNTIME_ONLY = new Set([
  "Background_obsolete",
  "obsolete_template_warning",
  "special_forces_icon",
  "allow_as_hq_icon",
  "can_paradrop_icon",
  "foreign_template_flag",
  "colonial_force_flag_frame",
  "exile_force_flag_frame",
  "tutorial_highlight_train",
  "template_cap",
]);

function usePriorityArt(projectId: number, enabled: boolean) {
  return useSWR<Record<string, GuiSprite>>(
    enabled ? ["game-view-sprites", projectId, "deploy-priorities"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(PRIORITY_ROWS.map((row) => row.sprite));
    },
    { revalidateOnFocus: false },
  );
}

function usePriorityLabels(projectId: number, enabled: boolean) {
  return useSWR<Record<string, string>>(
    enabled ? ["game-view-loc", projectId, "deploy-priorities"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(PRIORITY_ROWS.map((row) => row.key));
    },
    { revalidateOnFocus: false },
  );
}

function PriorityRow({
  entry,
  label,
  sprite,
  art,
  left,
  top,
}: {
  entry: GuiWindow;
  label: string;
  sprite: string;

  art: GuiSprite | undefined;
  left: number;
  top: number;
}) {
  const edit = useEditCanvas();
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    for (const child of clone.children ?? []) {
      if (child.name === "reinforcements_icon") child.sprite = sprite;
    }
    assignIds(clone);
    return clone;
  }, [entry.node, sprite]);
  const empty = useMemo(() => new Set<string>(), []);
  const nameId = useMemo(() => {
    const name = (node.children ?? []).find(
      (child) => child.name === "name",
    ) as (GuiNode & { _id?: string }) | undefined;
    return name?._id ?? "";
  }, [node]);
  const textOverrides = useMemo(
    () => (nameId ? { [nameId]: label } : {}),
    [nameId, label],
  );

  const sprites = useMemo(
    () => (art ? { ...entry.sprites, [sprite]: art } : entry.sprites),
    [entry.sprites, sprite, art],
  );

  return (
    <div style={{ position: "absolute", left, top }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${left},${top}`,
          entry,
          node,
          sprites,
          textOverrides,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={empty}
        isolateId={null}
        transparent
        baseW={500}
        baseH={DEPLOY_ROW_PITCH}
      />
    </div>
  );
}

export function useDivisions(
  projectId: number,
  tag: string | null,
  date: string | null,
) {
  return useSWR<GuiDivisions>(
    tag ? ["game-view-divisions", projectId, tag, date] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.divisions(tag as string, date || undefined);
    },
    { revalidateOnFocus: false },
  );
}

function findNode(root: GuiNode, name: string): GuiNode | undefined {
  if (root.name === name) return root;
  for (const child of root.children ?? []) {
    const found = findNode(child, name);
    if (found) return found;
  }
  return undefined;
}

function TemplateRow({
  template,
  entry,
  counter,
  left,
  top,
  onEdit,
}: {
  template: GuiDivisions["templates"][number];
  entry: GuiWindow;

  counter: GuiSprite | undefined;
  left: number;
  top: number;
  onEdit?: () => void;
}) {
  const edit = useEditCanvas();
  const { swaps } = useContext(EditContext);
  const node = useMemo(() => {
    const clone = structuredClone(entry.node) as GuiNode;
    for (const child of clone.children ?? []) {
      if (child.name && RUNTIME_ONLY.has(child.name)) child.sprite = null;

      if (child.name === "division_priority_icon")
        child.frame = template.priority + 1;
    }
    assignIds(clone);
    return clone;
  }, [entry.node, template.priority]);
  const empty = useMemo(() => new Set<string>(), []);
  const capId = useMemo(() => {
    const cap = (node.children ?? []).find(
      (child) => child.name === "template_cap",
    ) as (GuiNode & { _id?: string }) | undefined;
    return cap?._id ?? "";
  }, [node]);

  const textOverrides = useMemo(() => (capId ? { [capId]: "" } : {}), [capId]);
  const rects = useMemo(
    () => measureWindow(node, entry.sprites, 1920, 1080),
    [node, entry.sprites],
  );
  const counterRect = rects["name:counter"];
  const editRect = rects["name:edit_button"];
  const nameNode = findNode(entry.node, "name");
  const font = nameNode?.font ? entry.fonts[nameNode.font]?.bitmap : undefined;
  const nameRect = rects["name:name"];

  const composition = [...template.regiments, ...template.support]
    .map((unit) => `${unit.count}× ${unit.label || unit.type}`)
    .join(", ");

  const counterUrl = artUrl(template.icon, counter, swaps);
  return (
    <div
      style={{ position: "absolute", left, top }}
      title={`${template.name} — ${composition}`}
    >
      <GuiCanvas
        node={node}
        {...edit(
          `${entry.name}:${left},${top}`,
          entry,
          node,
          entry.sprites,
          textOverrides,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={empty}
        isolateId={null}
        transparent
        baseW={355}
        baseH={ROW_PITCH}
      />
      {counter && counterRect && (
        <div
          style={{
            position: "absolute",
            left: counterRect.x,
            top: counterRect.y,
            width: counterRect.w || 32,
            height: counterRect.h || 32,

            backgroundImage: counterUrl ? `url("${counterUrl}")` : undefined,
            backgroundPosition: "0 0",
            backgroundRepeat: "no-repeat",
            backgroundSize: `${(counterRect.w || 32) * Math.max(1, counter.frames || 1)}px ${
              counterRect.h || 32
            }px`,
          }}
        />
      )}

      {counterRect && (
        <UiEditArt
          owner={`${entry.name}:${left},${top}:counter`}
          gfx={template.icon}
          sprite={counter}
          title={template.name}
          x={counterRect.x}
          y={counterRect.y}
          w={counterRect.w || 32}
          h={counterRect.h || 32}
        />
      )}
      {onEdit && editRect && (
        <button
          type="button"
          aria-label={`edit ${template.name}`}
          onClick={onEdit}
          style={{
            position: "absolute",
            left: editRect.x,
            top: editRect.y,
            width: editRect.w || 44,
            height: editRect.h || 18,
            background: "transparent",
            border: "none",
            padding: 0,
            cursor: "inherit",
            pointerEvents: "auto",
          }}
        />
      )}
      {font && nameRect && (
        <BitmapText
          font={font}
          text={template.name}
          left={nameRect.x}
          top={nameRect.y}
          width={190}
          height={font.line + 4}
          align="left"
          vcenter={false}
          maxLines={1}
        />
      )}
    </div>
  );
}

export function DeploymentLayer({
  win,
  projectId,
  tag,
  date,
  onEditTemplate,
  onUnavailable,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;

  date?: string | null;

  onEditTemplate?: (name: string) => void;

  onUnavailable?: (control: string) => void;
}) {
  const isTemplates = win.name === TEMPLATE_DEPLOYMENT;
  const isPanel = win.name === DEPLOYMENT_VIEW;

  const [showDecommissioned, setShowDecommissioned] = useState(false);
  const divisions = useDivisions(
    projectId,
    isTemplates ? tag : null,
    date ?? null,
  );
  const entry = useTemplateWindow(
    projectId,
    divisions.data ? TEMPLATE_ENTRY : null,
  );
  const deployEntry = useTemplateWindow(
    projectId,
    isPanel ? DEPLOY_ENTRY : null,
  );
  const labels = usePriorityLabels(projectId, isPanel);
  const art = usePriorityArt(projectId, isPanel);
  const rects = useMemo(
    () =>
      isTemplates || isPanel
        ? measureWindow(win.node, win.sprites, 1920, 1080)
        : {},
    [isTemplates, isPanel, win],
  );

  const counterNames = useMemo(() => {
    const names = new Set<string>();
    for (const item of guiArray(divisions.data?.templates))
      if (item.icon) names.add(item.icon);
    return [...names].sort();
  }, [divisions.data]);
  const counters = useSWR<Record<string, GuiSprite>>(
    counterNames.length
      ? [
          "game-view-sprites",
          projectId,
          "template-counters",
          counterNames.join(","),
        ]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(counterNames);
    },
    { revalidateOnFocus: false },
  );

  const foreignArrow = useMemo(
    () =>
      isTemplates
        ? expandButtonOf(findNode(win.node, FOREIGN_TEMPLATES))
        : undefined,
    [isTemplates, win.node],
  );
  const foreignArt = useSWR<Record<string, GuiSprite>>(
    foreignArrow
      ? [
          "game-view-sprites",
          projectId,
          "foreign-templates",
          foreignArrow.sprite,
        ]
      : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites([foreignArrow?.sprite ?? ""]);
    },
    { revalidateOnFocus: false },
  );

  if (isPanel) {
    const listRect = rects["name:deployments_list_gridbox"];
    if (!listRect || !deployEntry.data || !labels.data) return null;
    return (
      <div
        style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
      >
        {PRIORITY_ROWS.map((row, at) => (
          <PriorityRow
            key={row.key}
            entry={deployEntry.data as GuiWindow}
            label={labels.data?.[row.key] ?? row.key}
            sprite={row.sprite}
            art={art.data?.[row.sprite]}
            left={listRect.x}
            top={listRect.y + at * DEPLOY_ROW_PITCH}
          />
        ))}
      </div>
    );
  }

  if (!isTemplates || !divisions.data || !entry.data) return null;
  const gridRect = rects["name:templates_grid"];
  if (!gridRect) return null;
  const decommissionedRect = rects["name:checkbox"];
  const decommissionedNode = findNode(win.node, "checkbox");
  const decommissionedSprite = decommissionedNode?.sprite
    ? win.sprites[decommissionedNode.sprite]
    : undefined;
  const foreignRect = rects[`name:${FOREIGN_TEMPLATES}`];
  const foreignSprite = foreignArrow
    ? foreignArt.data?.[foreignArrow.sprite]
    : undefined;

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {foreignArrow &&
        foreignRect &&
        foreignSprite &&
        !foreignSprite.missing_file && (
          <>
            <div
              style={frameStyle(
                foreignSprite,
                foreignRect.x + foreignArrow.x,
                foreignRect.y + foreignArrow.y,
                {
                  scale: foreignArrow.scale,
                  name: foreignArrow.sprite,
                },
              )}
            />
            <button
              type="button"
              aria-label={FOREIGN_TEMPLATES}
              onClick={() => onUnavailable?.(FOREIGN_TEMPLATES)}
              style={{
                position: "absolute",
                left: foreignRect.x + foreignArrow.x,
                top: foreignRect.y + foreignArrow.y,
                width: Math.max(
                  1,
                  Math.round(
                    (foreignSprite.width /
                      Math.max(1, foreignSprite.frames || 1)) *
                      foreignArrow.scale,
                  ),
                ),
                height: Math.max(
                  1,
                  Math.round(foreignSprite.height * foreignArrow.scale),
                ),
                padding: 0,
                border: "none",
                background: "transparent",
                cursor: "inherit",
                pointerEvents: "auto",
              }}
            />
          </>
        )}
      {decommissionedRect && (
        <>
          {showDecommissioned &&
            decommissionedNode?.sprite &&
            decommissionedSprite && (
              <div
                style={frameStyle(
                  decommissionedSprite,
                  decommissionedRect.x,
                  decommissionedRect.y,
                  {
                    frame: 2,
                    name: decommissionedNode.sprite,
                  },
                )}
              />
            )}
          <button
            type="button"
            aria-label="show_decommissioned_templates"
            aria-pressed={showDecommissioned}
            onClick={() => setShowDecommissioned((shown) => !shown)}
            style={{
              position: "absolute",
              left: decommissionedRect.x,
              top: decommissionedRect.y,
              width: decommissionedRect.w || 34,
              height: decommissionedRect.h || 30,
              padding: 0,
              border: "none",
              background: "transparent",
              cursor: "inherit",
              pointerEvents: "auto",
            }}
          />
        </>
      )}
      {guiArray(divisions.data.templates).map((template, at) => (
        <TemplateRow
          key={template.name}
          template={template}
          entry={entry.data as GuiWindow}
          counter={template.icon ? counters.data?.[template.icon] : undefined}
          left={gridRect.x}
          top={gridRect.y + at * ROW_PITCH}
          onEdit={
            onEditTemplate ? () => onEditTemplate(template.name) : undefined
          }
        />
      ))}
    </div>
  );
}
