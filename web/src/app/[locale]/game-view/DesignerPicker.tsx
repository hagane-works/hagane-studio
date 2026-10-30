"use client";

import { Pencil } from "lucide-react";
import { useMemo } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import {
  guiRenderApi,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "../gui-renderer/GuiCanvas";
import { gameTooltipProps, hideGameTooltip } from "../gui-renderer/GameTooltip";
import { fillRuntimeSlots } from "../gui-renderer/tooltipText";
import { useTemplateWindow } from "./CountrySelectLayer";
import { matchNamePath } from "./gameNavigation";
import { useEditCanvas } from "./editContext";
import type { DesignerUnit, PickerEntry } from "./designerEdit";

const GROUP_ENTRY = "designer_subunit_category_entry";
const UNIT_ENTRY = "designer_subunit_entry";
const REMOVE_ICON = "GFX_remove_icon";

const ENTRY_BG = "GFX_mini_tooltip";

const EDIT_HANDLE = 16;

const LINES = {
  add: "DESIGNER_ADD_DESC",
  replace: "DESIGNER_REPLACE_DESC",
  gain: "DESIGNER_REMOVE_NEGATIVE_DESC",
  removeCost: "DESIGNER_REMOVE_DESC",
  remove: "DESIGNER_REMOVE",
  selected: "SELECTED_SUBUNIT",
  group: "NEW_GROUP_ADDITIONAL_COST",

  paradrop: "BATALION_CAN_BE_PARCHUTED",
} as const;

const NO_HIDDEN: Set<string> = new Set();

export function costLine(
  markup: string | undefined,
  value: number,
  tone: "R" | "Y" | "G",
): string {
  if (!markup) return "";
  const filled = fillRuntimeSlots(markup, { VALUE: value }).replace(
    /§!/g,
    `§${tone}`,
  );
  return `§${tone}${filled}§!`;
}

function usePickerTexts(projectId: number, enabled: boolean) {
  return useSWR<Record<string, string>>(
    enabled ? ["game-view-loc", projectId, "designer-picker"] : null,
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.loc(Object.values(LINES), { markup: true });
    },
    { revalidateOnFocus: false },
  );
}

const GROUP_ICON_UNIT: Record<string, string> = {
  infantry: "infantry",
  mobile: "motorized",
  armor: "light_armor",
  combat_support: "artillery_brigade",
  mobile_combat_support: "mot_artillery_brigade",
};

function groupIcon(
  units: readonly DesignerUnit[],
  group: string,
): string | undefined {
  const named = units.find(
    (unit) => unit.type === GROUP_ICON_UNIT[group] && unit.icon,
  );
  return (named ?? units.find((unit) => unit.group === group && unit.icon))
    ?.icon;
}

function Entry({
  entry,
  label,
  cost,
  icon,
  sprites,
  special,
  paradrop = false,
  paradropTip,
  left,
  top,
  onClick,
  onHover,
  tooltip,
  onEdit,
}: {
  entry: GuiWindow;
  label: string;
  cost: string;
  icon: string | undefined;
  sprites: Record<string, GuiSprite>;
  special: boolean;

  paradrop?: boolean;

  paradropTip?: string;
  left: number;
  top: number;
  onClick: () => void;

  onHover?: (over: boolean) => void;
  tooltip?: string;

  onEdit?: () => void;
}) {
  const t = useTranslations("gameViewMilitary.designer");
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;
    node.children = (node.children ?? []).filter((child) => {
      if (child.name === "can_paradrop_icon") return paradrop;
      if (child.name === "special_forces_icon") return special;
      if (child.name === "cost") return Boolean(cost);
      return true;
    });
    for (const child of node.children) {
      if (child.name === "subunit_icon") child.sprite = icon ?? child.sprite;

      if (child.name === "special_forces_icon") child.frame = 2;
    }
    assignIds(node);
    const texts: Record<string, string> = {};
    for (const hit of matchNamePath(node, "name") as (GuiNode & {
      _id?: string;
    })[]) {
      if (hit._id) texts[hit._id] = label;
    }
    for (const hit of matchNamePath(node, "cost") as (GuiNode & {
      _id?: string;
    })[]) {
      if (hit._id) texts[hit._id] = cost;
    }
    return { node, texts };
  }, [entry.node, label, cost, icon, special, paradrop]);
  const allSprites = useMemo(
    () => ({ ...entry.sprites, ...sprites }),
    [entry.sprites, sprites],
  );

  const badge = useMemo(() => {
    const node = (entry.node.children ?? []).find(
      (child) => child.name === "can_paradrop_icon",
    );
    const sprite = node?.sprite ? allSprites[node.sprite] : undefined;
    if (!node || !sprite) return null;
    return {
      x: Number(node.position?.x ?? 0),
      y: Number(node.position?.y ?? 0),
      w: Math.round(sprite.width / Math.max(1, sprite.frames || 1)),
      h: sprite.height,
    };
  }, [entry.node, allSprites]);
  const edit = useEditCanvas();
  const size = entry.node.size as { width?: number; height?: number } | null;
  const w = Number(size?.width) || 250;
  const h = Number(size?.height) || 66;
  return (
    <div style={{ position: "absolute", left, top, width: w, height: h }}>
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${label}`,
          entry,
          built.node,
          allSprites,
          built.texts,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={w}
        baseH={h}
      />
      <button
        type="button"
        aria-label={label}
        onClick={() => {
          hideGameTooltip(`designer-picker:${label}`);
          onClick();
        }}
        {...(() => {
          const tip = gameTooltipProps(tooltip || null, {
            key: `designer-picker:${label}`,
          });
          return {
            onPointerEnter: (e: React.PointerEvent<Element>) => {
              onHover?.(true);
              tip.onPointerEnter(e);
            },
            onPointerMove: tip.onPointerMove,
            onPointerLeave: () => {
              onHover?.(false);
              tip.onPointerLeave();
            },
          };
        })()}
        style={{
          position: "absolute",
          inset: 0,
          background: "transparent",
          border: "none",
          padding: 0,
          cursor: "inherit",
          pointerEvents: "auto",
        }}
      >
        {paradrop && badge && paradropTip && (
          <span
            {...(() => {
              const tip = gameTooltipProps(paradropTip, {
                key: `designer-picker-paradrop:${label}`,
              });
              return {
                onPointerEnter: (e: React.PointerEvent<Element>) => {
                  hideGameTooltip(`designer-picker:${label}`);
                  tip.onPointerEnter(e);
                },
                onPointerMove: (e: React.PointerEvent<Element>) => {
                  e.stopPropagation();
                  tip.onPointerMove(e);
                },
                onPointerLeave: tip.onPointerLeave,
              };
            })()}
            style={{
              position: "absolute",
              left: badge.x,
              top: badge.y,
              width: badge.w,
              height: badge.h,
            }}
          />
        )}
      </button>
      {onEdit && (
        <button
          type="button"
          aria-label={t("editBattalion", { name: label })}
          onClick={(e) => {
            e.stopPropagation();
            hideGameTooltip(`designer-picker:${label}`);
            onEdit();
          }}
          className="flex items-center justify-center rounded-sm bg-card/80 text-foreground"
          style={{
            position: "absolute",
            left: 3,
            top: 2,
            width: EDIT_HANDLE,
            height: EDIT_HANDLE,
            padding: 0,
            pointerEvents: "auto",
            outline: "1px dashed rgba(244, 178, 62, 0.7)",
          }}
        >
          <Pencil size={10} aria-hidden />
        </button>
      )}
    </div>
  );
}

export function DesignerPicker({
  projectId,
  grid,
  back,
  entries,
  units,
  titles,
  experience,
  onEntry,
  onBack,
  onHover,
  tooltipOf,
  onEditSubUnit,
}: {
  projectId: number;

  grid: {
    x: number;
    y: number;
    slot: { w: number; h: number };
    columns: number;
  };

  back: { x: number; y: number; w: number; h: number } | undefined;
  entries: PickerEntry[];
  units: readonly DesignerUnit[];
  titles: Record<string, string>;

  experience: number;
  onEntry: (entry: PickerEntry) => void;
  onBack: () => void;

  onHover?: (entry: PickerEntry | null) => void;

  tooltipOf?: (entry: PickerEntry) => string | undefined;

  onEditSubUnit?: (type: string) => void;
}) {
  const groupEntry = useTemplateWindow(projectId, GROUP_ENTRY);
  const unitEntry = useTemplateWindow(projectId, UNIT_ENTRY);
  const texts = usePickerTexts(projectId, true);
  const iconNames = useMemo(() => {
    const names = new Set<string>([REMOVE_ICON, ENTRY_BG]);
    for (const unit of units) if (unit.icon) names.add(unit.icon);
    return [...names].sort();
  }, [units]);
  const icons = useSWR<Record<string, GuiSprite>>(
    ["game-view-sprites", projectId, "designer-picker", iconNames.join(",")],
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.sprites(iconNames);
    },
    { revalidateOnFocus: false },
  );
  if (!groupEntry.data || !unitEntry.data || !texts.data) return null;
  const line = texts.data;
  const tone = (cost: number) => (cost > experience ? "R" : "Y");
  const sprites = icons.data ?? {};

  return (
    <>
      {entries.map((item, at) => {
        const left = grid.x + (at % grid.columns) * grid.slot.w;
        const top = grid.y + Math.floor(at / grid.columns) * grid.slot.h;
        if (item.kind === "remove") {
          return (
            <Entry
              key="remove"
              entry={unitEntry.data as GuiWindow}
              label={line[LINES.remove] ?? "Remove"}

              cost={
                item.cost > 0
                  ? costLine(line[LINES.removeCost], item.cost, tone(item.cost))
                  : costLine(
                      line[LINES.gain],
                      -item.cost,
                      item.cost < 0 ? "G" : "Y",
                    )
              }
              icon={REMOVE_ICON}
              sprites={sprites}
              special={false}
              left={left}
              top={top}
              onClick={() => onEntry(item)}
              onHover={(over) => onHover?.(over ? item : null)}
              tooltip={tooltipOf?.(item)}
            />
          );
        }
        if (item.kind === "group") {
          return (
            <Entry
              key={`group-${item.group}`}
              entry={groupEntry.data as GuiWindow}
              label={titles[item.group] ?? item.group}
              cost={
                item.cost === null
                  ? ""
                  : costLine(line[LINES.group], item.cost, tone(item.cost))
              }
              icon={groupIcon(units, item.group)}
              sprites={sprites}
              special={false}
              left={left}
              top={top}
              onClick={() => onEntry(item)}
              onHover={(over) => onHover?.(over ? item : null)}
              tooltip={tooltipOf?.(item)}
            />
          );
        }
        const cost =
          item.line === "selected"
            ? (line[LINES.selected] ?? "")
            : costLine(
                line[item.line === "add" ? LINES.add : LINES.replace],
                item.cost,
                tone(item.cost),
              );
        return (
          <Entry
            key={`unit-${item.unit.type}`}
            entry={unitEntry.data as GuiWindow}
            label={item.unit.label || item.unit.type}
            cost={cost}
            icon={item.unit.icon}
            sprites={sprites}
            special={Boolean(item.unit.special_force)}
            paradrop={Boolean(item.unit.paradrop)}
            paradropTip={line[LINES.paradrop]}
            left={left}
            top={top}
            onClick={() => onEntry(item)}
            onHover={(over) => onHover?.(over ? item : null)}
            tooltip={tooltipOf?.(item)}
            onEdit={
              onEditSubUnit ? () => onEditSubUnit(item.unit.type) : undefined
            }
          />
        );
      })}
      {back && (
        <button
          type="button"
          aria-label="cancel_button"
          onClick={onBack}
          style={{
            position: "absolute",
            left: back.x,
            top: back.y,
            width: back.w || 123,
            height: back.h || 34,
            background: "transparent",
            border: "none",
            padding: 0,
            cursor: "inherit",
            pointerEvents: "auto",
          }}
        />
      )}
    </>
  );
}
