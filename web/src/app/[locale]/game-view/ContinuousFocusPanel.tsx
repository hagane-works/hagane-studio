"use client";

import { useMemo } from "react";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { useEditCanvas } from "./editContext";
import { matchNamePath } from "./gameNavigation";

export interface ContinuousFocus {
  id: string;
  name: string;
  icon: string;
  icon_sprite?: GuiSprite;
  daily_cost?: number;
}

export interface ContinuousFocusPanelPayload {
  palette: string;

  position: { x: number; y: number };
  focuses: ContinuousFocus[];
}

const PANEL = { width: 770, height: 380 } as const;
export const CONTINUOUS_LIST = {
  left: 13 + 16,
  top: 80 + 33,
  slotW: 180,
  slotH: 120,
  across: 4,
} as const;

const ENTRIES = { left: 13, top: 80, width: 744, height: 280 } as const;

const NO_HIDDEN = new Set<string>();

export function continuousSlot(index: number): { left: number; top: number } {
  return {
    left:
      CONTINUOUS_LIST.left +
      (index % CONTINUOUS_LIST.across) * CONTINUOUS_LIST.slotW,
    top:
      CONTINUOUS_LIST.top +
      Math.floor(index / CONTINUOUS_LIST.across) * CONTINUOUS_LIST.slotH,
  };
}

export function ContinuousFocusPanel({
  projectId,
  panel,
  left,
  top,
}: {
  projectId: number;
  panel: ContinuousFocusPanelPayload;

  left: number;
  top: number;
}) {
  const view = useTemplateWindow(projectId, "nationalfocusview");
  const item = useTemplateWindow(
    projectId,
    panel.focuses.length ? "continuous_national_focus_item" : null,
  );
  const edit = useEditCanvas();
  const frame = useMemo(() => {
    if (!view.data) return null;
    const found = matchNamePath(view.data.node, "continuous_focus_window")[0];
    if (!found) return null;
    const node = structuredClone(found) as GuiNode;

    assignIds(node);
    return node;
  }, [view.data]);
  if (!view.data || !frame) return null;
  const win = view.data;
  return (
    <div
      style={{
        position: "absolute",
        left,
        top,
        width: PANEL.width,
        height: PANEL.height,
      }}
    >
      <GuiCanvas
        node={frame}
        {...edit(
          `${win.name}:continuous_focus_window`,
          win,
          frame,
          win.sprites,
          {},
        )}
        texts={win.texts}
        fonts={win.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={PANEL.width}
        baseH={PANEL.height}
      />
      {item.data && (
        <div
          style={{
            position: "absolute",
            left: ENTRIES.left,
            top: ENTRIES.top,
            width: ENTRIES.width,
            height: ENTRIES.height,
            overflow: "hidden",
          }}
        >
          {panel.focuses.map((focus, index) => {
            const slot = continuousSlot(index);
            return (
              <ContinuousFocusItem
                key={focus.id}
                entry={item.data as GuiWindow}
                focus={focus}
                left={slot.left - ENTRIES.left}
                top={slot.top - ENTRIES.top}
              />
            );
          })}
        </div>
      )}
    </div>
  );
}

function ContinuousFocusItem({
  entry,
  focus,
  left,
  top,
}: {
  entry: GuiWindow;
  focus: ContinuousFocus;
  left: number;
  top: number;
}) {
  const edit = useEditCanvas();
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;

    node.children = (node.children ?? []).filter(
      (child) => child.name !== "continuous_glow",
    );
    if (focus.icon_sprite) {
      for (const hit of matchNamePath(node, "symbol")) hit.sprite = focus.icon;
    }
    assignIds(node);
    const texts: Record<string, string> = {};
    for (const hit of matchNamePath(node, "name") as (GuiNode & {
      _id?: string;
    })[]) {
      if (hit._id) texts[hit._id] = focus.name || focus.id;
    }
    for (const hit of matchNamePath(node, "focus_cost") as (GuiNode & {
      _id?: string;
    })[]) {
      if (hit._id && focus.daily_cost != null)
        texts[hit._id] = String(focus.daily_cost);
    }
    return { node, texts };
  }, [entry.node, focus]);
  const sprites = useMemo(
    () =>
      focus.icon_sprite
        ? { ...entry.sprites, [focus.icon]: focus.icon_sprite }
        : entry.sprites,
    [entry.sprites, focus.icon, focus.icon_sprite],
  );
  const size = entry.node.size as { width?: number; height?: number } | null;
  return (
    <div
      style={{ position: "absolute", left, top }}
      title={focus.name || focus.id}
    >
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${focus.id}`,
          { ...entry, sprites },
          built.node,
          sprites,
          built.texts,
        )}
        texts={entry.texts}
        fonts={entry.fonts}
        hidden={NO_HIDDEN}
        isolateId={null}
        transparent
        baseW={Number(size?.width) || 80}
        baseH={Number(size?.height) || 124}
      />
    </div>
  );
}
