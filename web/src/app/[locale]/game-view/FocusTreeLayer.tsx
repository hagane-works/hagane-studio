import { useEffect, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import {
  guiRenderApi,
  type GuiBitmapFont,
  type GuiSprite,
} from "@/lib/guiRenderApi";
import { BitmapText } from "../gui-renderer/BitmapText";
import { gameTooltipProps } from "../gui-renderer/GameTooltip";
import { cellAt, exclusivePairs, relativeDescendants } from "./focusEdit";
import {
  ContinuousFocusPanel,
  type ContinuousFocusPanelPayload,
} from "./ContinuousFocusPanel";
import { FocusShortcuts, type FocusShortcut } from "./FocusShortcuts";
import { drawSize, type SpriteSwaps } from "./spriteSwaps";
import { spriteUrlOf } from "./spriteFrame";

const FOCUS_ZOOM_MIN = 0.2;
const FOCUS_ZOOM_MAX = 1.0;

export interface FocusNode {
  id: string;
  name: string;
  x: number;
  y: number;
  cost: number;
  prereq: string[][] | null;
  mutex: string[] | null;
  icon: string;
  icon_sprite: GuiSprite;

  relative_to?: string;

  tooltip?: string;
}

export interface FocusTreePayload {
  id: string;
  focuses: FocusNode[];
  art: Record<"unavailable" | "available" | "completed", GuiSprite> &
    Partial<Record<"exclusive" | "exclusive_line", GuiSprite>>;
  spacing: { x: number; y: number };
  font?: { bitmap?: GuiBitmapFont };

  shortcuts?: FocusShortcut[];

  continuous?: ContinuousFocusPanelPayload;
}

const PANEL_OFFSETS_KEY = "gameview:continuous-panel-offsets";

type PanelOffsets = Record<string, { dx: number; dy: number }>;

function loadPanelOffsets(): PanelOffsets {
  try {
    const raw =
      typeof window === "undefined"
        ? null
        : window.localStorage.getItem(PANEL_OFFSETS_KEY);
    const parsed: unknown = raw ? JSON.parse(raw) : {};
    return parsed && typeof parsed === "object" ? (parsed as PanelOffsets) : {};
  } catch {
    return {};
  }
}

function savePanelOffsets(offsets: PanelOffsets): void {
  try {
    window.localStorage.setItem(PANEL_OFFSETS_KEY, JSON.stringify(offsets));
  } catch {}
}

const GRID_ORIGIN = 50;

export function treePoint(
  at: { x: number; y: number },
  min: { x: number; y: number },
  spacing: { x: number; y: number },
): { left: number; top: number } {
  return {
    left: at.x - GRID_ORIGIN - min.x * spacing.x,
    top: at.y - GRID_ORIGIN - min.y * spacing.y,
  };
}

const PLATE_W = 164;
const PLATE_H = 50;
const ICON_W = 94;
const ICON_H = 76;
const LINK_W = 3;

const LINK_BEGIN_Y = ICON_H - 6 + 24;
const LINK_BAR_Y = LINK_BEGIN_Y + 32;
const LINK_END_Y = 18;

export function focusLinkBars(
  from: { cx: number; cy: number },
  child: { cx: number; cy: number },
): { left: number; top: number; width: number; height: number }[] {
  const startY = from.cy + LINK_BEGIN_Y;
  const endY = child.cy + LINK_END_Y;
  const midY = Math.min(from.cy + LINK_BAR_Y, endY);
  const bars = [
    {
      left: from.cx - LINK_W / 2,
      top: startY,
      width: LINK_W,
      height: Math.max(0, midY - startY),
    },
  ];
  if (Math.abs(child.cx - from.cx) > 1) {
    bars.push({
      left: Math.min(from.cx, child.cx),
      top: midY - LINK_W / 2,
      width: Math.abs(child.cx - from.cx),
      height: LINK_W,
    });
  }
  bars.push({
    left: child.cx - LINK_W / 2,
    top: midY,
    width: LINK_W,
    height: Math.max(0, endY - midY),
  });
  return bars;
}

function spriteURL(sprite: GuiSprite | undefined): string | undefined {
  return sprite?.vanilla_url;
}

export function jumpView(
  frame: { width: number; height: number },
  treeW: number,
  target: { x: number; y: number },
  zoom: number,
): { zoom: number; x: number; y: number } {
  return {
    zoom,
    x: Math.round(
      frame.width / 2 - (frame.width - treeW * zoom) / 2 - target.x * zoom,
    ),
    y: Math.round(frame.height / 2 - target.y * zoom),
  };
}

export function FocusTreeLayer({
  projectId,
  tree,
  left,
  top,
  width,
  height,
  selectedId,
  onSelect,
  versions,
  onMove,
  onMoveContinuous,
  continuousHint,
  onCreateAt,
}: {
  projectId: number;
  tree: FocusTreePayload;

  left: number;
  top: number;
  width: number;
  height: number;
  selectedId?: string | null;
  onSelect?: (focus: FocusNode) => void;

  versions?: SpriteSwaps;

  onMove?: (focus: FocusNode, x: number, y: number) => void;

  onMoveContinuous?: (position: { x: number; y: number }) => void;

  continuousHint?: string;

  onCreateAt?: (x: number, y: number) => void;
}) {
  const t = useTranslations("gameViewContent.focus");
  const spacingX = tree.spacing?.x || 96;
  const spacingY = tree.spacing?.y || 130;
  const xs = tree.focuses.map((f) => f.x);
  const ys = tree.focuses.map((f) => f.y);
  const minX = Math.min(...xs, 0);
  const minY = Math.min(...ys, 0);
  const treeW = (Math.max(...xs, 0) - minX) * spacingX + PLATE_W;
  const treeH = (Math.max(...ys, 0) - minY) * spacingY + ICON_H + PLATE_H;

  const [view, setView] = useState<{
    zoom: number;
    x: number;
    y: number;
  } | null>(null);
  const zoom = view?.zoom ?? FOCUS_ZOOM_MAX;
  const panX = view?.x ?? 0;
  const panY = view?.y ?? 0;
  const drag = useRef<{
    x: number;
    y: number;
    panX: number;
    panY: number;
    moved: boolean;

    focus?: FocusNode;

    panel?: boolean;
  } | null>(null);

  const [panelShift, setPanelShift] = useState<{
    dx: number;
    dy: number;
  } | null>(null);

  const [panelOffsets, setPanelOffsets] =
    useState<PanelOffsets>(loadPanelOffsets);
  const panelKey = `${projectId}:${tree.id}`;
  const viewOffset = onMoveContinuous ? undefined : panelOffsets[panelKey];
  const setViewOffset = (offset: { dx: number; dy: number } | null) => {
    setPanelOffsets((current) => {
      const next = { ...current };
      if (offset) next[panelKey] = offset;
      else delete next[panelKey];
      savePanelOffsets(next);
      return next;
    });
  };

  const [moving, setMoving] = useState<{
    id: string;
    dx: number;
    dy: number;
    with: Set<string>;
  } | null>(null);

  const [menu, setMenu] = useState<{
    left: number;
    top: number;
    x: number;
    y: number;
  } | null>(null);
  const frameRef = useRef<HTMLDivElement>(null);
  const treeRef = useRef<HTMLDivElement>(null);

  const unitScale = () => {
    const rect = treeRef.current?.getBoundingClientRect();
    return rect?.width && treeW ? rect.width / treeW : zoom;
  };
  useEffect(() => {
    if (!menu) return;
    const close = (e: KeyboardEvent | PointerEvent) => {
      if (e instanceof KeyboardEvent && e.key !== "Escape") return;
      if (
        e instanceof PointerEvent &&
        (e.target as Element | null)?.closest?.("[data-focus-menu]")
      )
        return;
      setMenu(null);
    };
    window.addEventListener("keydown", close);
    window.addEventListener("pointerdown", close);
    return () => {
      window.removeEventListener("keydown", close);
      window.removeEventListener("pointerdown", close);
    };
  }, [menu]);

  const centreOf = (focus: FocusNode) => {
    const shift =
      moving && (moving.id === focus.id || moving.with.has(focus.id));
    return {
      cx: (focus.x + (shift ? moving.dx : 0) - minX) * spacingX + spacingX / 2,
      cy: (focus.y + (shift ? moving.dy : 0) - minY) * spacingY,
    };
  };
  const byId = new Map(tree.focuses.map((f) => [f.id, f]));

  const jump = (shortcut: FocusShortcut) => {
    const focus = byId.get(shortcut.target);
    if (!focus) return;
    const nextZoom = shortcut.zoom
      ? Math.min(FOCUS_ZOOM_MAX, Math.max(FOCUS_ZOOM_MIN, shortcut.zoom))
      : zoom;
    const { cx, cy } = centreOf(focus);
    setView(
      jumpView(
        { width, height },
        treeW,
        { x: cx, y: cy + (ICON_H + PLATE_H) / 2 },
        nextZoom,
      ),
    );
  };
  const shortcutIcons = new Map(
    (tree.shortcuts ?? []).flatMap((shortcut) => {
      const focus = byId.get(shortcut.target);
      return focus
        ? [
            [
              shortcut.target,
              { name: focus.icon, sprite: focus.icon_sprite },
            ] as const,
          ]
        : [];
    }),
  );
  const plate = tree.art?.unavailable;
  const plateURL = spriteURL(plate);
  const font = tree.font?.bitmap;

  const links: React.ReactNode[] = [];
  for (const focus of tree.focuses) {
    const child = centreOf(focus);
    for (const group of focus.prereq ?? []) {
      for (const parentId of group) {
        const parent = byId.get(parentId);
        if (!parent) continue;
        const key = `${parentId}->${focus.id}`;
        focusLinkBars(centreOf(parent), child).forEach((style, at) => {
          links.push(
            <div
              key={`${key}-${at}`}
              style={{ position: "absolute", background: "#5d7480", ...style }}
            />,
          );
        });
      }
    }
  }

  const exclusiveArt = tree.art?.exclusive;
  const exclusiveLine = tree.art?.exclusive_line;
  const exclusives: React.ReactNode[] = [];
  if (exclusiveArt?.vanilla_url) {
    const frames = Math.max(1, exclusiveArt.frames || 3);
    const iconW = Math.round((exclusiveArt.width || 96) / frames);
    const iconH = exclusiveArt.height || 32;
    const icon = (key: string, frame: number, x: number, y: number) => (
      <div
        key={key}
        style={{
          position: "absolute",
          left: Math.round(x),
          top: Math.round(y - iconH / 2),
          width: iconW,
          height: iconH,
          backgroundImage: `url("${exclusiveArt.vanilla_url}")`,
          backgroundSize: `${iconW * frames}px ${iconH}px`,
          backgroundPosition: `${-iconW * frame}px 0`,
          pointerEvents: "none",
        }}
      />
    );
    for (const [left, right] of exclusivePairs(tree.focuses)) {
      const a = centreOf(left);
      const b = centreOf(right);
      const x0 = a.cx + PLATE_W / 2;
      const x1 = b.cx - PLATE_W / 2;
      const y = a.cy + ICON_H - 6 + PLATE_H / 2;
      const key = `${left.id}<>${right.id}`;
      if (exclusiveLine?.vanilla_url && x1 > x0) {
        const lineFrames = Math.max(1, exclusiveLine.frames || 4);
        const lineH = exclusiveLine.height || 16;
        const width = x1 - x0 + 12;
        exclusives.push(
          <div
            key={`${key}-line`}
            style={{
              position: "absolute",
              left: Math.round(x0 - 6),
              top: Math.round(y - lineH / 2),
              width: Math.round(width),
              height: lineH,
              backgroundImage: `url("${exclusiveLine.vanilla_url}")`,
              backgroundSize: `${Math.round(width) * lineFrames}px ${lineH}px`,
              backgroundRepeat: "no-repeat",
              pointerEvents: "none",
            }}
          />,
        );
      }
      if (x1 - x0 >= iconW * 3) {
        exclusives.push(
          icon(`${key}-l`, 1, x0, y),
          icon(`${key}-r`, 2, x1 - iconW, y),
        );
      }
      exclusives.push(icon(`${key}-m`, 0, (x0 + x1) / 2 - iconW / 2, y));
    }
  }

  return (
    <>
      <div
        ref={frameRef}
        style={{
          position: "absolute",
          left,
          top,
          width,
          height,
          overflow: "hidden",
          cursor: moving ? "move" : "grab",
        }}
        onContextMenu={(e) => {
          if (!onCreateAt) return;
          e.preventDefault();

          if ((e.target as Element).closest?.("[data-focus-id]")) {
            setMenu(null);
            return;
          }
          const rect = treeRef.current?.getBoundingClientRect();
          const frame = frameRef.current?.getBoundingClientRect();
          if (!rect || !frame) return;
          const unit = unitScale();
          const cell = cellAt(
            (e.clientX - rect.left) / unit,
            (e.clientY - rect.top) / unit,
            { x: minX, y: minY },
            {
              x: spacingX,
              y: spacingY,
            },
          );

          if (tree.focuses.some((f) => f.x === cell.x && f.y === cell.y)) {
            setMenu(null);
            return;
          }

          const frameScale = frame.width / width || 1;
          setMenu({
            left: (e.clientX - frame.left) / frameScale,
            top: (e.clientY - frame.top) / frameScale,
            ...cell,
          });
        }}
        onWheel={(e) => {
          const next = Math.min(
            FOCUS_ZOOM_MAX,
            Math.max(FOCUS_ZOOM_MIN, zoom * (e.deltaY < 0 ? 1.15 : 1 / 1.15)),
          );
          setView({ zoom: next, x: panX, y: panY });
        }}
        onPointerDown={(e) => {
          drag.current = {
            x: e.clientX,
            y: e.clientY,
            panX,
            panY,
            moved: false,
          };
        }}
        onPointerMove={(e) => {
          const start = drag.current;
          if (!start) return;
          const dx = e.clientX - start.x;
          const dy = e.clientY - start.y;

          if (!start.moved && (Math.abs(dx) > 3 || Math.abs(dy) > 3)) {
            start.moved = true;
            (e.currentTarget as Element).setPointerCapture?.(e.pointerId);
          }
          if (!start.moved) return;
          if (start.panel) {
            const unit = unitScale();
            setPanelShift({
              dx: Math.round(dx / unit),
              dy: Math.round(dy / unit),
            });
            return;
          }
          if (start.focus) {
            const unit = unitScale();
            const cellX = Math.round(dx / unit / spacingX);
            const cellY = Math.round(dy / unit / spacingY);
            if (
              moving?.dx !== cellX ||
              moving?.dy !== cellY ||
              moving?.id !== start.focus.id
            ) {
              setMoving({
                id: start.focus.id,
                dx: cellX,
                dy: cellY,
                with:
                  moving?.id === start.focus.id
                    ? moving.with
                    : relativeDescendants(tree.focuses, start.focus.id),
              });
            }
            return;
          }
          setView({ zoom, x: start.panX + dx, y: start.panY + dy });
        }}
        onPointerUp={(e) => {
          if (drag.current?.moved)
            (e.currentTarget as Element).releasePointerCapture?.(e.pointerId);
          const dragged = drag.current?.focus;
          if (
            dragged &&
            moving?.id === dragged.id &&
            (moving.dx || moving.dy)
          ) {
            onMove?.(dragged, dragged.x + moving.dx, dragged.y + moving.dy);
          }
          setMoving(null);
          if (
            drag.current?.panel &&
            panelShift &&
            (panelShift.dx || panelShift.dy) &&
            tree.continuous
          ) {
            const at = tree.continuous.position;
            if (onMoveContinuous)
              onMoveContinuous({
                x: at.x + panelShift.dx,
                y: at.y + panelShift.dy,
              });
            else
              setViewOffset({
                dx: (viewOffset?.dx ?? 0) + panelShift.dx,
                dy: (viewOffset?.dy ?? 0) + panelShift.dy,
              });
          }
          setPanelShift(null);

          const finished = drag.current;
          window.setTimeout(() => {
            if (drag.current === finished) drag.current = null;
          }, 0);
        }}
        className="active:cursor-grabbing"
      >
        <div
          ref={treeRef}
          style={{
            position: "absolute",
            left: Math.round((width - treeW * zoom) / 2) + panX,
            top: panY,
            width: treeW,
            height: treeH,
            transform: `scale(${zoom})`,
            transformOrigin: "top left",
          }}
        >
          {links}
          {tree.focuses.map((focus) => {
            const { cx, cy } = centreOf(focus);

            const swapped = versions?.[focus.icon];
            const icon = swapped
              ? guiRenderApi.spriteUrl(focus.icon, swapped.version)
              : focus.icon_sprite
                ? spriteUrlOf(focus.icon_sprite, focus.icon)
                : undefined;
            const selected = selectedId === focus.id;
            const declaredW = focus.icon_sprite?.width
              ? Math.round(
                  focus.icon_sprite.width /
                    Math.max(1, focus.icon_sprite.frames || 1),
                )
              : ICON_W;

            const { width: iconW, height: iconH } = drawSize(
              versions ?? {},
              focus.icon,
              declaredW,
              focus.icon_sprite?.height || ICON_H,
            );
            return (
              <div
                key={focus.id}
                data-focus-id={focus.id}
                {...(focus.tooltip && !moving
                  ? gameTooltipProps(focus.tooltip, {
                      key: `focus:${focus.id}`,
                    })
                  : {})}
                style={{
                  position: "absolute",
                  left: 0,
                  top: 0,
                  opacity: moving?.id === focus.id ? 0.85 : undefined,
                }}
                onPointerDown={(e) => {
                  if (!onMove || e.button !== 0) return;

                  e.stopPropagation();
                  drag.current = {
                    x: e.clientX,
                    y: e.clientY,
                    panX,
                    panY,
                    moved: false,
                    focus,
                  };
                }}
                onClick={() => {
                  if (drag.current?.moved) return;
                  onSelect?.(focus);
                }}
              >
                <div
                  role="button"
                  aria-label={focus.id}
                  style={{
                    position: "absolute",
                    left: cx - PLATE_W / 2,
                    top: cy - 4,
                    width: PLATE_W,
                    height: ICON_H + PLATE_H + 4,
                    cursor: onMove ? "move" : "inherit",
                    borderRadius: 4,
                    outline: selected
                      ? "2px solid var(--color-accent, #f4b23e)"
                      : undefined,
                    background: selected
                      ? "rgba(244, 178, 62, 0.12)"
                      : "transparent",
                  }}
                />
                {icon && (
                  <div
                    style={{
                      position: "absolute",
                      left: cx - iconW / 2,
                      top: cy,
                      width: iconW,
                      height: iconH,
                      backgroundImage: `url("${icon}")`,
                      backgroundSize: `${iconW}px ${iconH}px`,
                      backgroundRepeat: "no-repeat",
                    }}
                  />
                )}
                {plateURL && (
                  <div
                    style={{
                      position: "absolute",
                      left: cx - PLATE_W / 2,
                      top: cy + ICON_H - 6,
                      width: PLATE_W,
                      height: PLATE_H,
                      backgroundImage: `url("${plateURL}")`,
                      backgroundSize: `${PLATE_W}px ${PLATE_H}px`,
                      backgroundRepeat: "no-repeat",
                    }}
                  />
                )}
                {font ? (
                  <BitmapText
                    font={font}
                    text={focus.name}
                    left={cx - 72}
                    top={cy + ICON_H + 2}
                    width={145}
                    height={PLATE_H - 8}
                    align="center"
                    vcenter
                    maxLines={2}
                  />
                ) : null}
              </div>
            );
          })}
          {exclusives}
          {tree.continuous && (
            <div
              data-continuous-focus-panel
              style={{ position: "absolute", left: 0, top: 0, cursor: "move" }}
              title={continuousHint}
              onPointerDown={(e) => {
                if (e.button !== 0) return;

                e.stopPropagation();
                drag.current = {
                  x: e.clientX,
                  y: e.clientY,
                  panX,
                  panY,
                  moved: false,
                  panel: true,
                };
              }}
              onDoubleClick={() => {
                if (!onMoveContinuous && viewOffset) setViewOffset(null);
              }}
            >
              <ContinuousFocusPanel
                projectId={projectId}
                panel={tree.continuous}
                {...treePoint(
                  {
                    x:
                      tree.continuous.position.x +
                      (viewOffset?.dx ?? 0) +
                      (panelShift?.dx ?? 0),
                    y:
                      tree.continuous.position.y +
                      (viewOffset?.dy ?? 0) +
                      (panelShift?.dy ?? 0),
                  },
                  { x: minX, y: minY },
                  { x: spacingX, y: spacingY },
                )}
              />
            </div>
          )}
        </div>
        {menu && onCreateAt && (
          <div
            data-focus-menu
            style={{
              position: "absolute",
              left: menu.left,
              top: menu.top,
              cursor: "default",
            }}
            onPointerDown={(e) => e.stopPropagation()}
            className="z-10 rounded-md border border-border bg-card p-1 text-card-foreground shadow-lg"
          >
            <button
              type="button"
              className="block w-full whitespace-nowrap rounded px-3 py-1 text-left text-sm hover:bg-muted"
              onClick={() => {
                onCreateAt(menu.x, menu.y);
                setMenu(null);
              }}
            >
              {t("createHere")}
            </button>
          </div>
        )}
      </div>

      <FocusShortcuts
        projectId={projectId}
        shortcuts={tree.shortcuts ?? []}
        icons={shortcutIcons}
        onJump={jump}
      />
    </>
  );
}
