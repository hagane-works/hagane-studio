"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { assignIds, measureWindow } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { FieldHit } from "./FieldHit";
import { Filled } from "./FilledWindow";
import { matchNamePath } from "./gameNavigation";
import { useIntelAgency, type GuiIntelAgency } from "./IntelAgencyLayer";

export const INTEL_UPGRADES_WINDOW = "intelligence_agency_upgrades_window";
const BRANCH_ROW = "upgrade_branch_entry";
const UPGRADE = "upgrade_entry";
const REPETITION = "agency_upgrade_repetition_icon";

const ROW_SLOT_H = 137;
const ROW_SIZE = { w: 1040, h: 140 };

const UPGRADES_AT = { x: 5, y: 45 };
const UPGRADE_SLOT_W = 205;
const UPGRADE_SIZE = { w: 200, h: 115 };

const DOTS = { x: 37, y: 63, w: 100, icon: 11, spacing: 2 };

const HEIGHT_SHARE = 0.9;
const SCREEN_H = 1080;

const UPGRADE_HIDDEN = [
  "continuous_glow",
  "cancel_upgrade",
  "faction_icon",
  "upgrade_button",
];
const LOCKED_BUTTON = "GFX_agency_upgrade_btn_unavailable";

function idOf(node: GuiNode | undefined): string {
  return (node as (GuiNode & { _id?: string }) | undefined)?._id ?? "";
}

export function intelUpgradesWindow(
  win: GuiWindow,
  data: GuiIntelAgency | undefined,
): { win: GuiWindow; texts: Record<string, string> } {
  const node = structuredClone(win.node) as GuiNode;
  const cap = matchNamePath(node, "max_window_size")[0];
  const capH = Number(cap?.position?.y ?? 0);
  const height = Math.round(
    capH > 0
      ? Math.min(SCREEN_H * HEIGHT_SHARE, capH)
      : SCREEN_H * HEIGHT_SHARE,
  );
  node.size = { ...node.size, height };
  for (const list of matchNamePath(node, "branches")) {
    const own = Number(list.size?.height);
    const top = Number(list.position?.y ?? 0);
    if (own < 0) list.size = { ...list.size, height: height - top + own };
  }
  assignIds(node);
  const texts: Record<string, string> = {};
  if (data?.texts?.upgrades_duration) {
    for (const hit of matchNamePath(node, "upgrade_duration_title"))
      if (idOf(hit)) texts[idOf(hit)] = data.texts.upgrades_duration;
  }
  return { win: { ...win, node }, texts };
}

export function dotsLeft(levels: number): number {
  const width = levels * DOTS.icon + Math.max(0, levels - 1) * DOTS.spacing;
  return DOTS.x + (DOTS.w - width) / 2;
}

type Rect = { x: number; y: number; w: number; h: number };

export function IntelUpgradesLayer({
  win,
  projectId,
  tag,
  date,
  onClose,
  onEditUpgrade,
}: {
  win: GuiWindow;
  projectId: number;
  tag: string;
  date?: string | null;
  onClose: () => void;
  editMode?: boolean;

  onEditUpgrade?: (id: string) => void;
}) {
  const tLayer = useTranslations("gameViewResearch.layer");
  const active = win.name === INTEL_UPGRADES_WINDOW;
  const data = useIntelAgency(projectId, active ? tag : null, date);
  const row = useTemplateWindow(projectId, active ? BRANCH_ROW : null);
  const upgrade = useTemplateWindow(projectId, active ? UPGRADE : null);
  const dot = useTemplateWindow(projectId, active ? REPETITION : null);
  const at = useMemo(() => {
    const out: Record<string, Rect | undefined> = {};
    if (!active) return out;
    const node = structuredClone(win.node) as GuiNode;
    const measured = measureWindow(node, win.sprites, 1920, 1080);
    for (const path of ["branches/branches_grid", "btn_close"])
      out[path] = measured[idOf(matchNamePath(node, path)[0])] as
        Rect | undefined;
    return out;
  }, [active, win]);
  if (!active || !data.data || !row.data || !upgrade.data) return null;
  const agency = data.data;
  const branches = Array.isArray(agency.branches) ? agency.branches : [];
  const agencySprites = agency.sprites ?? {};
  const grid = at["branches/branches_grid"];
  const close = at["btn_close"];
  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      {grid &&
        branches.map((branch, index) => {
          const top = grid.y + index * ROW_SLOT_H;
          return (
            <div key={branch.id}>
              <Filled
                entry={row.data as GuiWindow}
                left={grid.x}
                top={top}
                width={ROW_SIZE.w}
                height={ROW_SIZE.h}
                texts={{ upgrade_branch_name: branch.label }}
                hide={["selection_highlight"]}
                extraSprites={agencySprites}
              />
              {(Array.isArray(branch.upgrades) ? branch.upgrades : []).map(
                (item, column) => {
                  const left = grid.x + UPGRADES_AT.x + column * UPGRADE_SLOT_W;
                  const itemTop = top + UPGRADES_AT.y;
                  return (
                    <div key={item.id}>
                      <Filled
                        entry={upgrade.data as GuiWindow}
                        left={left}
                        top={itemTop}
                        width={UPGRADE_SIZE.w}
                        height={UPGRADE_SIZE.h}
                        texts={{
                          upgrade_name: item.label ?? item.id,
                          num_factories: String(item.factories ?? 0),
                        }}
                        pictures={{
                          upgrade_icon: item.picture,
                          upgrade_icon_frame: item.frame_sprite,

                          upgrade_background:
                            item.available === false
                              ? LOCKED_BUTTON
                              : undefined,
                        }}
                        hide={UPGRADE_HIDDEN}
                        extraSprites={agencySprites}
                      />
                      {dot.data &&
                        (item.levels ?? 0) > 1 &&
                        Array.from({ length: item.levels ?? 0 }, (_, level) => (
                          <Filled
                            key={level}
                            entry={dot.data as GuiWindow}
                            left={
                              left +
                              dotsLeft(item.levels ?? 0) +
                              level * (DOTS.icon + DOTS.spacing)
                            }
                            top={itemTop + DOTS.y}
                            width={DOTS.icon}
                            height={21}
                            texts={{}}
                            extraSprites={agencySprites}
                          />
                        ))}
                      {onEditUpgrade && (
                        <FieldHit
                          x={left}
                          y={itemTop}
                          w={UPGRADE_SLOT_W}
                          h={UPGRADE_SIZE.h - 30}
                          label={tLayer("upgrade", {
                            name: item.label ?? item.id,
                          })}
                          onClick={() => onEditUpgrade(item.id)}
                        />
                      )}
                    </div>
                  );
                },
              )}
            </div>
          );
        })}
      {close && (
        <div
          role="button"
          aria-label="Close upgrades"
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
