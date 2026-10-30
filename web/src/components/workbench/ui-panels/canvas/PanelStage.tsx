"use client";

import { useMemo, useRef, useState } from "react";
import { useTranslations } from "use-intl";

import {
  flattenTree,
  GuiCanvas,
  measureWindow,
} from "@/app/[locale]/gui-renderer/GuiCanvas";
import type { LayoutOverride } from "@/app/[locale]/gui-renderer/GuiCanvas";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";

import { ghostSize } from "./nodeControlMap";
import type { ControlEntry, HandleEntry } from "./nodeControlMap";
import { usePanelWindow } from "./usePanelWindow";

const NO_VERSIONS: Record<string, number> = {};
const NO_HIDDEN = new Set<string>();

function listsOf(node: GuiNode, sprites: GuiWindow["sprites"]) {
  const rects = measureWindow(node, sprites, 1920, 1080, undefined, {
    nativeButtonSprites: true,
  });
  return flattenTree(node)
    .filter(
      (row) =>
        row.node.type.toLowerCase() === "gridboxtype" &&
        row.node.name?.endsWith("_grid") &&
        rects[row.id],
    )
    .map((row) => {
      const raw = (row.node.raw ?? {}) as {
        slotsize?: { width?: number | string; height?: number | string };
        max_slots_horizontal?: number | string;
      };
      const box = rects[row.id];
      return {
        entry: `${(row.node.name ?? "").slice(0, -"_grid".length)}_entry`,
        box: { x: box.x, y: box.y, w: box.w, h: box.h },
        slot: {
          w: Number(raw.slotsize?.width) || box.w,
          h: Number(raw.slotsize?.height) || 30,
        },
        columns: Math.max(1, Number(raw.max_slots_horizontal) || 1),
      };
    });
}

function ListRowsPreview({
  projectId,
  updatedAt,
  entry,
  box,
  slot,
  columns,
}: {
  projectId: number;
  updatedAt: string;
  entry: string;
  box: { x: number; y: number; w: number; h: number };
  slot: { w: number; h: number };
  columns: number;
}) {
  const { win } = usePanelWindow(projectId, entry, updatedAt);
  if (!win) return null;
  const across = Math.max(1, Math.min(columns, Math.ceil(box.w / slot.w)));
  const cells = Math.min(40, across * Math.max(1, Math.ceil(box.h / slot.h)));
  return (
    <div
      data-list-rows={entry}
      aria-hidden
      className="absolute pointer-events-none [&_*]:!pointer-events-none"
      style={{
        left: box.x,
        top: box.y,
        width: box.w,
        height: box.h,
        overflow: "hidden",
      }}
    >
      {Array.from({ length: cells }, (_, cell) => (
        <div
          key={cell}
          className="absolute"
          style={{
            left: (cell % across) * slot.w,
            top: Math.floor(cell / across) * slot.h,
            opacity: cell === 0 ? 1 : 0.45,
          }}
        >
          <GuiCanvas
            node={win.node}
            sprites={win.sprites}
            texts={win.texts}
            fonts={win.fonts}
            versions={NO_VERSIONS}
            hidden={NO_HIDDEN}
            isolateId={null}
            selectedId={null}
            onSelect={() => {}}
            editMode={false}
            transparent
            nativeButtonSprites
          />
        </div>
      ))}
    </div>
  );
}

export interface PanelStageProps {
  projectId: number;
  updatedAt: string;
  win: GuiWindow | null;
  loading: boolean;
  errorMessage: string | null;

  node: GuiNode | null;
  selectedId: string | null;
  textOverrides: Record<string, string>;
  layoutOverrides: Record<string, LayoutOverride>;

  ghosts: ControlEntry[];

  handles: HandleEntry[];
  selectedPath: string | null;
  hasControls: boolean;
  onSelectNode: (id: string, node: GuiNode) => void;
  onSelectPath: (path: string) => void;
  onLayoutPreview: (id: string, partial: LayoutOverride) => void;
  onLayoutCommit: (id: string, partial: LayoutOverride) => void;
  onLayoutCancel: (id: string) => void;
}

function GhostBox({
  entry,
  selected,
  hint,
  badge,
  onSelect,
}: {
  entry: ControlEntry;
  selected: boolean;
  hint: string;
  badge: string;
  onSelect: (path: string) => void;
}) {
  const size = ghostSize(entry.ctrl);
  return (
    <button
      type="button"
      title={hint}
      onClick={() => onSelect(entry.path)}
      className={`absolute pointer-events-auto text-left rounded-sm border border-dashed border-accent/70 bg-accent/5 ui-interactive ui-focus-ring ${
        selected ? "ring-2 ring-accent" : ""
      }`}
      style={{
        left: entry.absX,
        top: entry.absY,
        width: size.w,
        height: size.h,
      }}
    >
      <span
        className={`absolute text-micro font-mono whitespace-nowrap rounded px-1.5 border border-accent/60 bg-popover text-accent ${
          entry.absY < 18 ? "top-0.5 left-0.5" : "-top-4 left-0"
        }`}
      >
        {entry.ctrl.name} · {badge}
      </span>
    </button>
  );
}

export function PanelStage({
  projectId,
  updatedAt,
  win,
  loading,
  errorMessage,
  node,
  selectedId,
  textOverrides,
  layoutOverrides,
  ghosts,
  handles,
  selectedPath,
  hasControls,
  onSelectNode,
  onSelectPath,
  onLayoutCommit,
  onLayoutCancel,
}: PanelStageProps) {
  const t = useTranslations("uiPanelsCanvas");
  const lists = useMemo(
    () => (win && node ? listsOf(node, win.sprites) : []),
    [win, node],
  );

  const [drag, setDrag] = useState<{
    path: string;
    dx: number;
    dy: number;
  } | null>(null);
  const dragRef = useRef<{ path: string; dx: number; dy: number } | null>(null);
  const dragStart = useRef<{ px: number; py: number } | null>(null);
  const setDragBoth = (v: { path: string; dx: number; dy: number } | null) => {
    dragRef.current = v;
    setDrag(v);
  };

  const handleDragStart =
    (h: HandleEntry) => (e: React.PointerEvent<HTMLElement>) => {
      if (dragStart.current) return;
      e.currentTarget.setPointerCapture(e.pointerId);
      dragStart.current = { px: e.clientX, py: e.clientY };
      setDragBoth({ path: h.path, dx: 0, dy: 0 });
      onSelectPath(h.path);
    };
  const handleDragMove = (e: React.PointerEvent<HTMLElement>) => {
    const start = dragStart.current;
    const d = dragRef.current;
    if (!start || !d) return;
    setDragBoth({
      path: d.path,
      dx: e.clientX - start.px,
      dy: e.clientY - start.py,
    });
  };
  const handleDragEnd =
    (h: HandleEntry) => (e: React.PointerEvent<HTMLElement>) => {
      if (e.currentTarget.hasPointerCapture(e.pointerId)) {
        e.currentTarget.releasePointerCapture(e.pointerId);
      }
      const d = dragRef.current;
      if (
        d &&
        d.path === h.path &&
        (Math.abs(d.dx) > 2 || Math.abs(d.dy) > 2)
      ) {
        onLayoutCommit(h.id, { x: h.ctrl.x + d.dx, y: h.ctrl.y + d.dy });
      }
      setDragBoth(null);
      dragStart.current = null;
    };
  const handleDragCancel = (h: HandleEntry) => () => {
    setDragBoth(null);
    dragStart.current = null;
    onLayoutCancel(h.id);
  };

  if (loading) {
    return (
      <div className="flex-1 grid place-items-center p-6">
        <p className="text-xs text-muted-foreground">{t("stageLoading")}</p>
      </div>
    );
  }

  if (win && node) {
    return (
      <div className="flex-1 overflow-auto p-6 bg-background">
        <div className="relative inline-block border border-border rounded-sm">
          <GuiCanvas
            node={node}
            sprites={win.sprites}
            texts={win.texts}
            fonts={win.fonts}
            versions={NO_VERSIONS}
            hidden={NO_HIDDEN}
            isolateId={null}
            selectedId={selectedId}
            onSelect={onSelectNode}
            editMode
            textOverrides={textOverrides}
            layoutOverrides={layoutOverrides}

            nativeButtonSprites
          />
          {lists.map((list) => (
            <ListRowsPreview
              key={list.entry}
              projectId={projectId}
              updatedAt={updatedAt}
              {...list}
            />
          ))}

          {(ghosts.length > 0 || handles.length > 0) && (
            <div className="absolute inset-0 pointer-events-none">
              {ghosts.map((g) => (
                <GhostBox
                  key={g.path}
                  entry={g}
                  selected={selectedPath === g.path}
                  hint={t("ghostHint")}
                  badge={t("ghostBadge")}
                  onSelect={onSelectPath}
                />
              ))}
              {handles.map((h) => {
                const selected = selectedPath === h.path;
                const size = ghostSize(h.ctrl);

                const dx = drag?.path === h.path ? drag.dx : 0;
                const dy = drag?.path === h.path ? drag.dy : 0;
                return (
                  <div key={`handle-${h.path}`}>
                    <button
                      type="button"
                      onClick={() => onSelectPath(h.path)}
                      onPointerDown={handleDragStart(h)}
                      onPointerMove={handleDragMove}
                      onPointerUp={handleDragEnd(h)}
                      onPointerCancel={handleDragCancel(h)}
                      className={`absolute pointer-events-auto touch-none cursor-grab active:cursor-grabbing text-micro font-mono rounded px-1.5 border ui-interactive ui-focus-ring whitespace-nowrap ${
                        selected
                          ? "border-accent text-accent bg-accent/10"
                          : "border-border text-muted-foreground bg-popover"
                      }`}
                      style={{
                        left: h.absX + dx,

                        top: (h.absY >= 18 ? h.absY - 16 : h.absY + 2) + dy,
                      }}
                    >
                      {h.ctrl.name}
                    </button>
                    {selected && (
                      <div
                        role="presentation"
                        onPointerDown={handleDragStart(h)}
                        onPointerMove={handleDragMove}
                        onPointerUp={handleDragEnd(h)}
                        onPointerCancel={handleDragCancel(h)}
                        className="absolute pointer-events-auto touch-none cursor-grab active:cursor-grabbing rounded-sm border border-dashed border-accent/70 bg-accent/5"
                        style={{
                          left: h.absX + dx,
                          top: h.absY + dy,
                          width: size.w,
                          height: size.h,
                        }}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    );
  }

  if (errorMessage) {
    return (
      <div className="flex-1 grid place-items-center p-6">
        <div className="ui-panel p-4 max-w-sm space-y-1">
          <p className="text-xs font-medium">{t("stageError")}</p>
          <p className="text-xs text-muted-foreground font-mono break-all">
            {errorMessage}
          </p>
        </div>
      </div>
    );
  }

  const extentW = Math.max(
    360,
    ...ghosts.map((g) => g.absX + ghostSize(g.ctrl).w + 40),
  );
  const extentH = Math.max(
    240,
    ...ghosts.map((g) => g.absY + ghostSize(g.ctrl).h + 40),
  );
  return (
    <div className="flex-1 overflow-auto p-6 space-y-4">
      <div className="ui-panel p-4 max-w-md space-y-1">
        <p className="text-xs font-medium">{t("stageNotSavedTitle")}</p>
        <p className="text-xs text-muted-foreground">
          {hasControls ? t("stageNotSavedBody") : t("stageEmpty")}
        </p>
      </div>
      {ghosts.length > 0 && (
        <div
          className="relative rounded-md border border-border bg-card/40"
          style={{ width: extentW, height: extentH }}
        >
          {ghosts.map((g) => (
            <GhostBox
              key={g.path}
              entry={g}
              selected={selectedPath === g.path}
              hint={t("ghostHint")}
              badge={t("ghostBadge")}
              onSelect={onSelectPath}
            />
          ))}
        </div>
      )}
    </div>
  );
}
