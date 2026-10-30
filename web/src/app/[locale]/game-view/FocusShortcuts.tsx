"use client";

import { useContext, useMemo } from "react";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "../gui-renderer/GuiCanvas";
import { useTemplateWindow } from "./CountrySelectLayer";
import { EditContext, useEditCanvas } from "./editContext";
import { matchNamePath } from "./gameNavigation";

export interface FocusShortcut {
  name: string;
  label: string;
  target: string;

  zoom?: number;
}

export const SHORTCUT_GRID = {
  left: 1,
  bottom: 1024,
  slot: 68,
  max: 8,
} as const;

const NO_HIDDEN = new Set<string>();

export function shortcutSlotTop(index: number): number {
  return SHORTCUT_GRID.bottom - (index + 1) * SHORTCUT_GRID.slot;
}

export function FocusShortcuts({
  projectId,
  shortcuts,
  icons,
  onJump,
}: {
  projectId: number;
  shortcuts: readonly FocusShortcut[];

  icons: ReadonlyMap<string, { name: string; sprite: GuiSprite | undefined }>;
  onJump: (shortcut: FocusShortcut) => void;
}) {
  const entry = useTemplateWindow(
    projectId,
    shortcuts.length ? "focus_tree_shortcut_item" : null,
  );
  if (!entry.data || !shortcuts.length) return null;
  const item = entry.data;
  return (
    <>
      {shortcuts.slice(0, SHORTCUT_GRID.max).map((shortcut, index) => (
        <ShortcutButton
          key={`${shortcut.name}:${shortcut.target}`}
          entry={item}
          shortcut={shortcut}
          icon={icons.get(shortcut.target)}
          left={SHORTCUT_GRID.left}
          top={shortcutSlotTop(index)}
          onJump={onJump}
        />
      ))}
    </>
  );
}

function ShortcutButton({
  entry,
  shortcut,
  icon,
  left,
  top,
  onJump,
}: {
  entry: GuiWindow;
  shortcut: FocusShortcut;
  icon: { name: string; sprite: GuiSprite | undefined } | undefined;
  left: number;
  top: number;
  onJump: (shortcut: FocusShortcut) => void;
}) {
  const edit = useEditCanvas();

  const { editMode } = useContext(EditContext);
  const built = useMemo(() => {
    const node = structuredClone(entry.node) as GuiNode;
    if (icon?.sprite) {
      for (const hit of matchNamePath(node, "focus_button"))
        hit.sprite = icon.name;
    }
    assignIds(node);
    const texts: Record<string, string> = {};
    for (const hit of matchNamePath(node, "name") as (GuiNode & {
      _id?: string;
    })[]) {
      if (hit._id) texts[hit._id] = shortcut.label || shortcut.name;
    }
    return { node, texts };
  }, [entry.node, icon, shortcut.label, shortcut.name]);
  const sprites = useMemo(
    () =>
      icon?.sprite
        ? { ...entry.sprites, [icon.name]: icon.sprite }
        : entry.sprites,
    [entry.sprites, icon],
  );
  const at = entry.node.position ?? { x: 0, y: 0 };
  const size = entry.node.size as { width?: number; height?: number } | null;
  const width = Number(size?.width) || 190;
  const height = Number(size?.height) || 72;
  return (
    <div
      role="button"
      aria-label={shortcut.label || shortcut.name}
      title={shortcut.label || shortcut.name}
      style={{
        position: "absolute",
        left: left + (Number(at.x) || 0),
        top: top + (Number(at.y) || 0),
        width,
        height,
        cursor: editMode ? undefined : "pointer",
        pointerEvents: "auto",
      }}
      onClick={() => {
        if (!editMode) onJump(shortcut);
      }}
    >
      <GuiCanvas
        node={built.node}
        {...edit(
          `${entry.name}:${shortcut.name}`,
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
        baseW={width}
        baseH={height}
      />
    </div>
  );
}
