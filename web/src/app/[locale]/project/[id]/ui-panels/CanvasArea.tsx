"use client";

import { useMemo, useState } from "react";
import { useLocale, useTranslations } from "use-intl";

import type { GuiNode } from "@/lib/guiRenderApi";
import type { LayoutOverride } from "@/app/[locale]/gui-renderer/GuiCanvas";
import type { UiPanel, UiPanelControl } from "@/types";
import {
  updateControlAtPath,
  useUiPanelSelection,
} from "@/components/workbench/ui-panels/selectionStore";
import {
  addPreset,
  addTarget,
  PRESET_EN_WORDS,
  PRESETS,
  type ComponentPreset,
} from "@/app/[locale]/game-view/panelMounts";
import { useMockVarsStore } from "@/components/workbench/ui-panels/mockVars";
import {
  buildNodeMaps,
  buildSavedByName,
  listControlEntries,
} from "@/components/workbench/ui-panels/canvas/nodeControlMap";
import {
  applyMockFrames,
  buildTextOverrides,
} from "@/components/workbench/ui-panels/canvas/mockPreview";
import { usePanelWindow } from "@/components/workbench/ui-panels/canvas/usePanelWindow";
import { PaletteBar } from "@/components/workbench/ui-panels/canvas/PaletteBar";
import { SelectionRibbon } from "@/components/workbench/ui-panels/canvas/SelectionRibbon";
import { PanelStage } from "@/components/workbench/ui-panels/canvas/PanelStage";

export interface CanvasAreaProps {
  projectId: number;
  panel: UiPanel;
  controls: UiPanelControl[];
  onControlsChange: (next: UiPanelControl[]) => void;
}

const ROW_HANDLE = "row:";

export function CanvasArea(props: CanvasAreaProps) {
  const { panel } = props;
  return (
    <CanvasAreaContent
      key={`${panel.id}:${panel.updated_at ?? ""}`}
      {...props}
    />
  );
}

function CanvasAreaContent({
  projectId,
  panel,
  controls,
  onControlsChange,
}: CanvasAreaProps) {
  const tc = useTranslations("gameViewEdit.components");
  const locale = useLocale();
  const selectedPath = useUiPanelSelection((s) => s.selectedPath);
  const setSelectedPath = useUiPanelSelection((s) => s.setSelectedPath);
  const mockEnabled = useMockVarsStore((s) => s.enabled);
  const mockVars = useMockVarsStore((s) => s.vars);

  const { win, loading, errorMessage } = usePanelWindow(
    projectId,
    panel.derived?.window_name ?? null,
    panel.updated_at,
  );

  const entries = useMemo(
    () => listControlEntries(controls, panel.derived),
    [controls, panel.derived],
  );
  const maps = useMemo(() => (win ? buildNodeMaps(win.node) : null), [win]);
  const savedByName = useMemo(
    () => buildSavedByName(panel.controls ?? []),
    [panel.controls],
  );
  const entryByDerived = useMemo(() => {
    const m = new Map<string, (typeof entries)[number]>();
    for (const e of entries) {
      if (e.derivedName && !m.has(e.derivedName)) m.set(e.derivedName, e);
    }
    return m;
  }, [entries]);

  const selectedEntry = useMemo(
    () => entries.find((e) => e.path === selectedPath) ?? null,
    [entries, selectedPath],
  );
  const selectedId =
    selectedEntry?.derivedName != null
      ? (maps?.idByName.get(selectedEntry.derivedName) ?? null)
      : null;

  const rowPaths = useMemo(() => {
    const lists = entries
      .filter((e) => e.ctrl.kind === "container" && e.ctrl.array)
      .map((e) => e.path);
    return new Set(
      entries
        .filter(
          (e) =>
            lists.some((p) => e.path.startsWith(`${p}/`)) &&
            e.derivedName != null &&
            savedByName.has(e.ctrl.name),
        )
        .map((e) => e.path),
    );
  }, [entries, savedByName]);
  const ghosts = useMemo(
    () =>
      entries.filter(
        (e) =>
          (!e.derivedName || !maps?.idByName.get(e.derivedName)) &&
          !rowPaths.has(e.path),
      ),
    [entries, maps, rowPaths],
  );

  const handles = useMemo(
    () =>
      entries
        .filter(
          (e) =>
            e.derivedName != null &&
            maps?.idByName.get(e.derivedName) != null &&
            e.ctrl.kind !== "text" &&
            e.ctrl.kind !== "button" &&
            !e.ctrl.sprite?.trim(),
        )
        .map((e) => ({
          ...e,
          id: maps!.idByName.get(e.derivedName!)!,
        }))
        .concat(
          entries
            .filter((e) => rowPaths.has(e.path))
            .map((e) => ({ ...e, id: `${ROW_HANDLE}${e.path}` })),
        ),
    [entries, maps, rowPaths],
  );

  const textOverrides = useMemo(
    () => buildTextOverrides(entries, maps, mockVars, mockEnabled),
    [entries, maps, mockVars, mockEnabled],
  );
  const displayNode = useMemo(
    () =>
      win ? applyMockFrames(win.node, entries, mockVars, mockEnabled) : null,
    [win, entries, mockVars, mockEnabled],
  );

  const [dragPreview, setDragPreview] = useState<
    Record<string, LayoutOverride>
  >({});

  const draftOverrides = useMemo(() => {
    const out: Record<string, LayoutOverride> = {};
    if (!maps) return out;
    for (const e of entries) {
      if (!e.derivedName) continue;
      const id = maps.idByName.get(e.derivedName);
      if (!id) continue;
      const saved = savedByName.get(e.ctrl.name);
      if (!saved) continue;
      const o: LayoutOverride = {};
      if (e.ctrl.x !== saved.x) o.x = e.ctrl.x;
      if (e.ctrl.y !== saved.y) o.y = e.ctrl.y;
      if (e.ctrl.w != null && e.ctrl.w !== (saved.w ?? null))
        o.width = e.ctrl.w;
      if (e.ctrl.h != null && e.ctrl.h !== (saved.h ?? null))
        o.height = e.ctrl.h;
      if (Object.keys(o).length > 0) out[id] = o;
    }
    return out;
  }, [entries, maps, savedByName]);

  const layoutOverrides = useMemo(() => {
    const m: Record<string, LayoutOverride> = { ...draftOverrides };
    for (const [id, v] of Object.entries(dragPreview))
      m[id] = { ...m[id], ...v };
    return m;
  }, [draftOverrides, dragPreview]);

  const handleLayoutPreview = (id: string, partial: LayoutOverride) =>
    setDragPreview((prev) => ({ ...prev, [id]: partial }));

  const clearPreview = (id: string) =>
    setDragPreview((prev) => {
      if (!(id in prev)) return prev;
      const next = { ...prev };
      delete next[id];
      return next;
    });

  const handleLayoutCommit = (id: string, partial: LayoutOverride) => {
    clearPreview(id);
    const name = maps?.nameById.get(id);
    const entry = id.startsWith(ROW_HANDLE)
      ? entries.find((e) => e.path === id.slice(ROW_HANDLE.length))
      : name
        ? entryByDerived.get(name)
        : undefined;

    if (!entry) return;
    const patch: Partial<UiPanelControl> = {};
    if (partial.x != null) patch.x = Math.round(partial.x);
    if (partial.y != null) patch.y = Math.round(partial.y);
    if (partial.width != null) patch.w = Math.round(partial.width);
    if (partial.height != null) patch.h = Math.round(partial.height);
    if (Object.keys(patch).length > 0) {
      onControlsChange(updateControlAtPath(controls, entry.path, patch));
    }
  };

  const handleSelectNode = (_id: string, node: GuiNode) => {
    const entry = node.name ? entryByDerived.get(node.name) : undefined;

    setSelectedPath(entry ? entry.path : null);
  };

  const handleAdd = (preset: ComponentPreset) => {
    const words = PRESETS[preset].words ?? "text";
    const own = {
      button: tc("defaultButton"),
      text: tc("defaultText"),
      title: tc("defaultTitle"),
      paragraph: tc("defaultParagraph"),
      listItem: tc("defaultListItem"),
    }[words];
    const [next, path] = addPreset(
      controls,
      selectedPath,
      preset,
      { en: PRESET_EN_WORDS[words], locale, own },
      panel.token,
    );
    onControlsChange(next);
    setSelectedPath(path);
  };
  const target = addTarget(controls, selectedPath);
  const addInto = target
    ? (entries.find((e) => e.path === target)?.ctrl.name ?? null)
    : null;

  return (
    <div className="flex flex-col min-w-0 flex-1 border-r border-border">
      <PaletteBar onAdd={handleAdd} into={addInto} />
      <SelectionRibbon
        entry={selectedEntry}

        isGhost={
          selectedEntry != null &&
          selectedId == null &&
          !rowPaths.has(selectedEntry.path)
        }
        mockEnabled={mockEnabled}
      />
      <PanelStage
        projectId={projectId}
        updatedAt={panel.updated_at}
        win={win}
        loading={loading}
        errorMessage={errorMessage}
        node={displayNode}
        selectedId={selectedId}
        textOverrides={textOverrides}
        layoutOverrides={layoutOverrides}
        ghosts={ghosts}
        handles={handles}
        selectedPath={selectedPath}
        hasControls={controls.length > 0}
        onSelectNode={handleSelectNode}
        onSelectPath={setSelectedPath}
        onLayoutPreview={handleLayoutPreview}
        onLayoutCommit={handleLayoutCommit}
        onLayoutCancel={clearPreview}
      />
    </div>
  );
}
