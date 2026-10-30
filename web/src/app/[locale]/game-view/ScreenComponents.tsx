"use client";

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type PointerEvent,
} from "react";
import { useLocale, useTranslations } from "use-intl";
import useSWR, { mutate as globalMutate } from "swr";
import { toast } from "sonner";
import {
  AlignLeft,
  AppWindow,
  GripHorizontal,
  Heading,
  Image as ImageIcon,
  Minus,
  MousePointerClick,
  PanelsTopLeft,
  RectangleHorizontal,
  Redo2,
  ScrollText,
  Square,
  SquareDashed,
  StretchHorizontal,
  ToggleRight,
  Type,
  Undo2,
  List,
  LayoutGrid,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { uiPanelsApi } from "@/lib/api/panels";
import type { UiPanel, UiPanelControl } from "@/types";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import {
  controlNameSlug,
  getControlAtPath,
  removeControlAtPath,
  uniqueControlName,
  updateControlAtPath,
  useUiPanelSelection,
} from "@/components/workbench/ui-panels/selectionStore";
import { InspectorArea } from "@/app/[locale]/project/[id]/ui-panels/InspectorArea";
import { PreferredLanguage } from "@/components/shared/editors/DescriptionLocsTabs";
import { BUTTON, GamePanel, PRIMARY } from "./GamePanel";
import { HoverTip } from "./HoverTip";
import {
  controlPathOf,
  fitFor,
  flagOpsOf,
  isWindowPanel,
  mountOf,
  newWindowPanel,
  openFlagOf,
  openTrigger,
  opensByButton,
  closeFlagEffect,
  PANEL_MOUNTS,
  contextOfMount,
  panelNames,
  panelsOnScreen,
  pickedControlNames,
  placeComponent,
  PRESET_EN_WORDS,
  PRESET_GROUPS,
  PRESETS,
  renamedPanel,
  SCREEN_EN_NAMES,
  relocateControl,
  screenPanelToken,
  type Box,
  type ComponentPreset,
  type Guide,
  type ScrollOf,
  WINDOW_LOOK,
} from "./panelMounts";

const PRESET_ICON: Record<ComponentPreset, LucideIcon> = {
  box: Square,
  scroll_box: ScrollText,
  list: List,
  grid: LayoutGrid,
  group: SquareDashed,
  text: Type,
  title: Heading,
  paragraph: AlignLeft,
  button: MousePointerClick,
  small_button: MousePointerClick,
  wide_button: StretchHorizontal,
  image_button: ImageIcon,
  switch: ToggleRight,
  icon: ImageIcon,
  divider: Minus,
  progressbar: RectangleHorizontal,
};

export type PaletteItem = ComponentPreset | "window_panel";

const PALETTE_GROUPS: {
  group: "panel" | (typeof PRESET_GROUPS)[number]["group"];
  items: PaletteItem[];
}[] = [
  { group: "panel", items: ["window_panel"] },
  ...PRESET_GROUPS.map((entry) => ({
    group: entry.group,
    items: entry.presets as PaletteItem[],
  })),
];

const ITEM_ICON: Record<PaletteItem, LucideIcon> = {
  ...PRESET_ICON,
  window_panel: AppWindow,
};

export function itemSize(item: PaletteItem): { w: number; h: number } {
  return item === "window_panel"
    ? { w: WINDOW_LOOK.w, h: WINDOW_LOOK.h }
    : { w: PRESETS[item].look.w, h: PRESETS[item].look.h };
}

const SCREEN_OF_MOUNT: Readonly<Record<string, string>> = Object.fromEntries(
  Object.entries(PANEL_MOUNTS).map(([screen, mount]) => [mount, screen]),
);
export function screenOfMount(mount: string | null | undefined): string | null {
  return (mount && SCREEN_OF_MOUNT[mount]) || null;
}

let scrollOffsets: ReadonlyMap<string, number> = new Map();
const scrollListeners = new Set<() => void>();
const NO_SCROLLS: ReadonlyMap<string, number> = new Map();
export function componentScroll(guiName: string): number {
  return scrollOffsets.get(guiName) ?? 0;
}
export function setComponentScroll(guiName: string, y: number) {
  if (componentScroll(guiName) === y) return;
  scrollOffsets = new Map(scrollOffsets).set(guiName, y);
  scrollListeners.forEach((listener) => listener());
}
function subscribeScroll(listener: () => void) {
  scrollListeners.add(listener);
  return () => scrollListeners.delete(listener);
}

export function useComponentScrolls(): ReadonlyMap<string, number> {
  return useSyncExternalStore(
    subscribeScroll,
    () => scrollOffsets,
    () => NO_SCROLLS,
  );
}

export function panelScrollOf(token: string): ScrollOf {
  const prefix = controlNameSlug(token) || "panel";
  return (control) =>
    control.scroll
      ? componentScroll(
          `${prefix}_${controlNameSlug(control.name) || "control"}`,
        )
      : 0;
}

export interface DropGhost extends Box {
  preset: PaletteItem;

  target?: "screen" | "topbar" | number | "new";

  origin?: { x: number; y: number };
  guides: Guide[];
  into: Box | null;
}

let ghost: DropGhost | null = null;
const ghostListeners = new Set<() => void>();
export function setDropGhost(next: DropGhost | null) {
  if (ghost === next) return;
  ghost = next;
  ghostListeners.forEach((listener) => listener());
}
export function dropGhostNow(): DropGhost | null {
  return ghost;
}
function subscribeGhost(listener: () => void) {
  ghostListeners.add(listener);
  return () => ghostListeners.delete(listener);
}

export function refreshPanelWindows(projectId: number, token: string) {
  const prefix = `${controlNameSlug(token) || "panel"}_`;
  return globalMutate(
    (key) =>
      Array.isArray(key) &&
      key[0] === "game-view-window" &&
      key[1] === projectId &&
      String(key[2]).startsWith(prefix),
  );
}

export interface PanelState {
  token: string;
  base: Pick<
    UiPanel,
    | "country_tag"
    | "context_type"
    | "parent_window_token"
    | "name_en"
    | "name_zh"
  > &
    Partial<Pick<UiPanel, "display_name_locs">>;
  panel: Pick<UiPanel, "window_props" | "controls"> | null;
}

export function panelState(panel: UiPanel): PanelState {
  return {
    token: panel.token,
    base: {
      country_tag: panel.country_tag,
      context_type: panel.context_type,
      parent_window_token: panel.parent_window_token,
      name_en: panel.name_en,
      name_zh: panel.name_zh,
      display_name_locs: panel.display_name_locs,
    },
    panel: { window_props: panel.window_props, controls: panel.controls },
  };
}

export type ComponentChange =
  "place" | "move" | "resize" | "edit" | "delete" | "window" | "panel";

export type RecordChange = (
  change: ComponentChange,
  before: PanelState,
) => void;

interface HistoryEntry {
  change: ComponentChange;
  state: PanelState;
}

const HISTORY_LIMIT = 50;

async function putBack(
  projectId: number,
  state: PanelState,
): Promise<PanelState> {
  const found = ((await uiPanelsApi.list(projectId)).panels ?? []).find(
    (panel) => panel.token === state.token,
  );
  const now = found
    ? panelState(await uiPanelsApi.get(projectId, found.id))
    : { ...state, panel: null };
  if (!state.panel) {
    if (found) await uiPanelsApi.delete(projectId, found.id);
  } else if (found) {
    await uiPanelsApi.update(projectId, found.id, state.panel);
  } else {
    await uiPanelsApi.create(projectId, {
      ...state.base,
      token: state.token,
      ...state.panel,
    });
  }
  await Promise.all([
    globalMutate(["game-view-ui-panels", projectId]),
    refreshPanelWindows(projectId, state.token),
    globalMutate(
      (key) =>
        Array.isArray(key) &&
        key[0] === "game-view-ui-panel" &&
        key[1] === projectId,
    ),
  ]);
  return now;
}

export function useComponentHistory(
  projectId: number,
  onRestored?: () => void,
) {
  const t = useTranslations("gameViewEdit.components");
  const [stacks, setStacks] = useState<{
    projectId: number;
    undo: HistoryEntry[];
    redo: HistoryEntry[];
  }>({ projectId, undo: [], redo: [] });
  const [busy, setBusy] = useState(false);

  const undo = stacks.projectId === projectId ? stacks.undo : [];
  const redo = stacks.projectId === projectId ? stacks.redo : [];
  const record: RecordChange = useCallback(
    (change, before) =>
      setStacks((current) => ({
        projectId,
        undo: [
          ...(current.projectId === projectId ? current.undo : []),
          { change, state: before },
        ].slice(-HISTORY_LIMIT),
        redo: [],
      })),
    [projectId],
  );
  const step = async (from: "undo" | "redo") => {
    const list = from === "undo" ? undo : redo;
    const entry = list[list.length - 1];
    if (!entry || busy) return;
    setBusy(true);
    try {
      const was = await putBack(projectId, entry.state);
      const back = { change: entry.change, state: was };
      setStacks((current) => ({
        projectId,
        undo:
          from === "undo" ? current.undo.slice(0, -1) : [...current.undo, back],
        redo:
          from === "undo" ? [...current.redo, back] : current.redo.slice(0, -1),
      }));
      onRestored?.();
      toast.success(
        t(from === "undo" ? "undone" : "redone", {
          what: t(`change.${entry.change}`),
        }),
      );
    } catch {
      toast.error(t("undoFailed"));
    } finally {
      setBusy(false);
    }
  };
  return {
    record,
    undo: () => void step("undo"),
    redo: () => void step("redo"),
    busy,

    nextUndo: undo.at(-1)?.change ?? null,
    nextRedo: redo.at(-1)?.change ?? null,
  };
}

export type ComponentHistory = ReturnType<typeof useComponentHistory>;

export function useScreenPanels(projectId: number, screen: string | null) {
  const list = useSWR(
    ["game-view-ui-panels", projectId],
    async () => (await uiPanelsApi.list(projectId)).panels ?? [],
    {
      revalidateOnFocus: false,
    },
  );
  return { list, panels: panelsOnScreen(list.data ?? [], screen) };
}

export interface PaletteDrag {
  onDragMove: (preset: PaletteItem, clientX: number, clientY: number) => void;

  onDrop: (preset: PaletteItem, clientX: number, clientY: number) => void;

  onDragCancel: () => void;
}

const DRAG_START_PX = 4;

const PALETTE_POSITION_KEY = "gameview.componentPalette.position.v3";

export type PalettePosition = { right: number; top: number };

function gameEdge(): number {
  const scale = Math.min(window.innerWidth / 1920, window.innerHeight / 1080);
  return (window.innerWidth - 1920 * scale) / 2;
}

const SNAP_PX = 16;

function storedPalettePosition(): PalettePosition | null {
  try {
    const raw = window.localStorage.getItem(PALETTE_POSITION_KEY);
    const at = raw ? (JSON.parse(raw) as Partial<PalettePosition>) : null;
    return at && Number.isFinite(at.right) && Number.isFinite(at.top)
      ? { right: at.right as number, top: at.top as number }
      : null;
  } catch {
    return null;
  }
}

function storePalettePosition(at: PalettePosition | null) {
  try {
    if (at)
      window.localStorage.setItem(PALETTE_POSITION_KEY, JSON.stringify(at));
    else window.localStorage.removeItem(PALETTE_POSITION_KEY);
  } catch {}
}

function keepOnScreen(
  at: PalettePosition,
  width: number,
  height: number,
): PalettePosition {
  const edge = gameEdge();
  return {
    right: Math.round(
      Math.min(
        Math.max(-edge, at.right),
        Math.max(-edge, window.innerWidth - edge - width),
      ),
    ),
    top: Math.round(
      Math.min(
        Math.max(8, at.top),
        Math.max(8, window.innerHeight - height - 8),
      ),
    ),
  };
}

const GAME_TOP_RIGHT = 120;
const GAME_MAP_MODES = 290;

export function defaultPalettePosition(
  width: number,
  tallest: number,
): PalettePosition {
  const scale = Math.min(window.innerWidth / 1920, window.innerHeight / 1080);
  const stageTop = (window.innerHeight - 1080 * scale) / 2;
  const from = stageTop + GAME_TOP_RIGHT * scale;
  const to = stageTop + (1080 - GAME_MAP_MODES) * scale;
  const top = tallest > to - from ? from : (from + to - tallest) / 2;
  return keepOnScreen({ right: 0, top }, width, tallest);
}

function tallestOf(bar: HTMLElement, height: number): number {
  const items = bar.querySelector("[data-palette-items]");
  const shown = items?.children.length ?? 0;
  if (!items || !shown) return height;
  const most = Math.max(...PALETTE_GROUPS.map((entry) => entry.items.length));

  const row = (items.getBoundingClientRect().height + 4) / shown;
  return height + Math.max(0, most - shown) * row;
}

function roomOf(at: PalettePosition, width: number): number {
  return Math.abs(at.right) <= 24
    ? Math.round(gameEdge() + at.right + width + 8)
    : 0;
}

export function ComponentPalette({
  screen,
  busy,
  drag,
  history,
  panels = [],
  onPickPanel,
  onRoom,
  topBar = false,
}: {
  screen: string | null;
  busy: boolean;
  drag?: PaletteDrag;

  history?: ComponentHistory;

  panels?: UiPanel[];
  onPickPanel?: (panel: UiPanel) => void;

  onRoom?: (px: number) => void;

  topBar?: boolean;
}) {
  const t = useTranslations("gameViewEdit.components");
  const mountable = mountOf(screen) !== null;

  const windows = panels.some(
    (panel) => isWindowPanel(panel) && panel.context_type === "player_context",
  );
  const [group, setGroup] =
    useState<(typeof PALETTE_GROUPS)[number]["group"]>("panel");
  const items =
    PALETTE_GROUPS.find((entry) => entry.group === group)?.items ?? [];
  const usable = (item: PaletteItem) =>
    !busy && (item === "window_panel" || mountable || windows || topBar);

  const press = useRef<{
    preset: PaletteItem;
    pointerId: number;
    x: number;
    y: number;
    dragging: boolean;
  } | null>(null);
  const [held, setHeld] = useState<PaletteItem | null>(null);
  const [listOpen, setListOpen] = useState(false);

  const bar = useRef<HTMLDivElement>(null);
  const [shown, setShown] = useState<{
    right: number;
    top: number;
    docked: boolean;
  } | null>(null);
  const place = useRef<PalettePosition | null>(null);
  const grip = useRef<{
    pointerId: number;
    x: number;
    y: number;
    from: PalettePosition;
  } | null>(null);
  const onRoomRef = useRef(onRoom);
  useEffect(() => {
    onRoomRef.current = onRoom;
  });
  const room = useRef(-1);
  const moveTo = useCallback((next: PalettePosition, width: number) => {
    place.current = next;
    setShown({
      right: gameEdge() + next.right,
      top: next.top,
      docked: next.right === 0,
    });
    const px = roomOf(next, width);
    if (px === room.current) return;
    room.current = px;
    onRoomRef.current?.(px);
  }, []);
  const placeBar = useCallback(() => {
    const el = bar.current;
    if (!el) return;
    const { width, height } = el.getBoundingClientRect();
    const kept = storedPalettePosition();
    moveTo(
      kept
        ? keepOnScreen(kept, width, height)
        : defaultPalettePosition(width, tallestOf(el, height)),
      width,
    );
  }, [moveTo]);
  useEffect(() => {
    const frame = requestAnimationFrame(placeBar);
    window.addEventListener("resize", placeBar);
    const sized =
      typeof ResizeObserver === "undefined"
        ? null
        : new ResizeObserver(() => {
            const el = bar.current;
            if (!el || !place.current) return;
            const { width, height } = el.getBoundingClientRect();
            moveTo(keepOnScreen(place.current, width, height), width);
          });
    if (bar.current) sized?.observe(bar.current);
    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener("resize", placeBar);
      sized?.disconnect();

      if (room.current > 0) onRoomRef.current?.(0);
    };
  }, [placeBar, moveTo]);
  const cancel = () => {
    press.current = null;
    setHeld(null);
    drag?.onDragCancel();
  };

  useEffect(() => {
    if (!held) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      press.current = null;
      setHeld(null);
      drag?.onDragCancel();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [held, drag]);

  useEffect(() => {
    if (!history) return;
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.tagName === "SELECT" ||
          target.isContentEditable)
      )
        return;
      if (!(event.ctrlKey || event.metaKey) || event.altKey) return;
      const key = event.key.toLowerCase();
      if (key === "z" && !event.shiftKey) {
        event.preventDefault();
        history.undo();
      } else if ((key === "z" && event.shiftKey) || key === "y") {
        event.preventDefault();
        history.redo();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [history]);
  return (
    <div
      ref={bar}
      data-component-palette=""

      className={`pointer-events-auto fixed z-[61] flex w-max max-w-48 flex-col gap-1 border border-ingame-edge/70 bg-ingame-panel p-1.5 text-ingame-text shadow-lg ${shown?.docked ? "rounded-l-md border-r-0" : "rounded-md"}`}

      style={
        shown
          ? { right: shown.right, top: shown.top }
          : { right: 0, top: 0, opacity: 0 }
      }
    >
      <div className="flex items-center gap-1">
        <div
          role="button"
          tabIndex={-1}
          data-palette-grip=""
          aria-label={t("paletteGrip")}
          title={t("paletteGrip")}
          onPointerDown={(event: PointerEvent<HTMLDivElement>) => {
            if (event.button !== 0 || !place.current) return;
            event.currentTarget.setPointerCapture?.(event.pointerId);
            grip.current = {
              pointerId: event.pointerId,
              x: event.clientX,
              y: event.clientY,
              from: place.current,
            };
          }}
          onPointerMove={(event: PointerEvent<HTMLDivElement>) => {
            const start = grip.current;
            if (!start || start.pointerId !== event.pointerId || !bar.current)
              return;
            const { width, height } = bar.current.getBoundingClientRect();
            const next = keepOnScreen(
              {
                right: start.from.right - (event.clientX - start.x),
                top: start.from.top + event.clientY - start.y,
              },
              width,
              height,
            );

            moveTo(
              Math.abs(next.right) <= SNAP_PX ? { ...next, right: 0 } : next,
              width,
            );
          }}
          onPointerUp={(event: PointerEvent<HTMLDivElement>) => {
            const start = grip.current;
            if (!start || start.pointerId !== event.pointerId) return;
            grip.current = null;
            storePalettePosition(place.current);
          }}
          onDoubleClick={() => {
            storePalettePosition(null);
            placeBar();
          }}
          className="flex h-6 cursor-move touch-none select-none items-center rounded px-1 text-ingame-gold/70 hover:bg-ingame-button/35 hover:text-ingame-gold"
        >
          <GripHorizontal className="h-3.5 w-3.5" aria-hidden />
        </div>

        <HoverTip
          tip={
            busy
              ? t("placing")
              : mountable
                ? t("dragHint")
                : t("unmountableWindowOk")
          }
          side="left"
        >
          {(describedBy) => (
            <span
              data-palette-title=""
              aria-describedby={describedBy}
              className="cursor-help text-xs text-ingame-gold"
            >
              {t("title")}
            </span>
          )}
        </HoverTip>
        {history && (
          <div className="ml-auto flex items-center gap-0.5 pl-2">
            <button
              type="button"
              data-history="undo"
              disabled={!history.nextUndo || history.busy}
              onClick={history.undo}
              title={
                history.nextUndo
                  ? t("undoWhat", { what: t(`change.${history.nextUndo}`) })
                  : t("undoNone")
              }
              aria-label={t("undo")}
              className="rounded p-1 text-ingame-title hover:bg-ingame-button/35 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <Undo2 className="h-3.5 w-3.5" aria-hidden />
            </button>
            <button
              type="button"
              data-history="redo"
              disabled={!history.nextRedo || history.busy}
              onClick={history.redo}
              title={
                history.nextRedo
                  ? t("redoWhat", { what: t(`change.${history.nextRedo}`) })
                  : t("redoNone")
              }
              aria-label={t("redo")}
              className="rounded p-1 text-ingame-title hover:bg-ingame-button/35 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <Redo2 className="h-3.5 w-3.5" aria-hidden />
            </button>
          </div>
        )}
      </div>
      <div
        role="group"
        aria-label={t("groupsLabel")}
        className="flex flex-col gap-0.5 border-t border-ingame-edge/40 pt-1"
      >
        {PALETTE_GROUPS.map((entry) => (
          <button
            key={entry.group}
            type="button"
            aria-pressed={entry.group === group}
            onClick={() => setGroup(entry.group)}
            className={`w-full rounded px-2 py-0.5 text-left text-xs ${entry.group === group ? "bg-ingame-button/45 text-ingame-title" : "text-ingame-text/80 hover:text-ingame-title"}`}
          >
            {t(`group.${entry.group}`)}
          </button>
        ))}
      </div>

      <div
        role="group"
        data-palette-items=""
        aria-label={t(`group.${group}`)}
        className="flex flex-col gap-1 border-t border-ingame-edge/40 pt-1"
      >
        {items.map((preset) => {
          const Icon = ITEM_ICON[preset];
          return (
            <button
              key={preset}
              type="button"
              data-preset={preset}
              data-held={held === preset ? "" : undefined}
              disabled={!usable(preset)}
              onPointerDown={(event: PointerEvent<HTMLButtonElement>) => {
                if (event.button !== 0 || !usable(preset)) return;
                event.currentTarget.setPointerCapture?.(event.pointerId);
                press.current = {
                  preset,
                  pointerId: event.pointerId,
                  x: event.clientX,
                  y: event.clientY,
                  dragging: false,
                };
              }}
              onPointerMove={(event: PointerEvent<HTMLButtonElement>) => {
                const at = press.current;
                if (!at || at.pointerId !== event.pointerId) return;
                if (
                  !at.dragging &&
                  Math.hypot(event.clientX - at.x, event.clientY - at.y) <
                    DRAG_START_PX
                )
                  return;
                if (!at.dragging) {
                  at.dragging = true;
                  setHeld(preset);
                }
                drag?.onDragMove(preset, event.clientX, event.clientY);
              }}
              onPointerUp={(event: PointerEvent<HTMLButtonElement>) => {
                const at = press.current;
                if (!at || at.pointerId !== event.pointerId) return;
                press.current = null;
                setHeld(null);
                if (at.dragging)
                  drag?.onDrop(preset, event.clientX, event.clientY);
              }}
              onPointerCancel={() => {
                if (press.current) cancel();
              }}
              title={
                usable(preset) || busy ? t(`hint.${preset}`) : t("unmountable")
              }
              className={`flex w-full touch-none select-none items-center gap-1 rounded border border-ingame-edge/70 bg-ingame-band px-2 py-1 text-xs text-ingame-title hover:border-ingame-gold/60 hover:bg-ingame-button/35 disabled:cursor-not-allowed disabled:opacity-50 ${held === preset ? "cursor-grabbing border-ingame-gold/60" : "cursor-grab"}`}
            >
              <Icon className="h-3.5 w-3.5" aria-hidden />
              {t(`preset.${preset}`)}
            </button>
          );
        })}
      </div>
      {onPickPanel && (
        <div className="relative border-t border-ingame-edge/40 pt-1">
          <button
            type="button"
            data-panel-list-toggle=""
            aria-expanded={listOpen}
            onClick={() => setListOpen((open) => !open)}
            className="flex w-full items-center gap-1 rounded border border-ingame-edge/70 bg-ingame-band px-2 py-1 text-xs text-ingame-title hover:border-ingame-gold/60 hover:bg-ingame-button/35"
          >
            <PanelsTopLeft className="h-3.5 w-3.5" aria-hidden />
            {t("panelList", { count: panels.length })}
          </button>
          {listOpen && (
            <PanelList
              panels={panels}
              below={shown !== null && shown.top < 120}
              onPick={(panel) => {
                setListOpen(false);
                onPickPanel(panel);
              }}
              onClose={() => setListOpen(false)}
            />
          )}
        </div>
      )}
    </div>
  );
}

function PanelList({
  panels,
  below = false,
  onPick,
  onClose,
}: {
  panels: UiPanel[];
  below?: boolean;
  onPick: (panel: UiPanel) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewEdit.components");
  const locale = useLocale();

  const box = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    const onDown = (event: globalThis.PointerEvent) => {
      if (box.current && !box.current.contains(event.target as Node)) onClose();
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("pointerdown", onDown, true);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("pointerdown", onDown, true);
    };
  }, [onClose]);
  const where = (panel: UiPanel) => {
    if (isWindowPanel(panel))
      return panel.context_type === "player_context"
        ? t("whereWindow")
        : t("whereOther", { context: panel.context_type });
    const screen = screenOfMount(panel.parent_window_token);
    return screen
      ? t("whereScreen", { screen: t(`screen.${panel.parent_window_token}`) })
      : t("whereOther", { context: panel.parent_window_token });
  };
  return (
    <div
      ref={box}
      data-panel-list=""
      role="dialog"
      aria-label={t("panelListTitle")}

      className={`absolute right-full mr-2 w-72 max-h-80 overflow-y-auto rounded-md border border-ingame-edge/70 bg-ingame-panel p-1.5 shadow-lg ${below ? "top-0" : "bottom-0"}`}
    >
      <p className="px-1.5 pb-1 text-xs text-ingame-gold">
        {t("panelListTitle")}
      </p>
      {panels.length === 0 ? (
        <p className="px-1.5 py-2 text-xs text-ingame-text/70">
          {t("panelListEmpty")}
        </p>
      ) : (
        <ul className="space-y-0.5">
          {panels.map((panel) => (
            <li key={panel.id}>
              <button
                type="button"
                data-panel-item={panel.token}
                onClick={() => onPick(panel)}
                className="w-full rounded px-1.5 py-1 text-left hover:bg-ingame-button/35 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ingame-gold/60"
              >
                <span className="block truncate text-xs text-ingame-title">
                  {panelLabel(panel, locale)}
                </span>
                <span className="block truncate text-xs text-ingame-text/70">
                  {where(panel)}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export function panelLabel(
  panel: Pick<UiPanel, "token" | "name_en" | "name_zh" | "display_name_locs">,
  locale: string,
): string {
  return (
    panel.display_name_locs?.[locale] ||
    (locale === "zh" ? panel.name_zh : panel.name_en) ||
    panel.name_en ||
    panel.token
  );
}

export function DropGhostLayer({ left, top }: { left: number; top: number }) {
  const t = useTranslations("gameViewEdit.components");
  const shown = useSyncExternalStore(subscribeGhost, dropGhostNow, () => null);
  if (!shown) return null;
  return (
    <div
      data-drop-ghost={shown.preset}
      style={{
        position: "absolute",
        left,
        top,
        pointerEvents: "none",
        zIndex: 5,
      }}
    >
      {shown.into && (
        <div
          data-drop-into=""
          className="absolute rounded-sm bg-accent/15 outline-2 outline-offset-2 outline-dashed outline-accent"
          style={{
            left: shown.into.x,
            top: shown.into.y,
            width: shown.into.w,
            height: shown.into.h,
          }}
        />
      )}
      <SnapGuides guides={shown.guides} />
      <div
        className="absolute rounded-sm border-2 border-dashed border-accent bg-accent/20"
        style={{ left: shown.x, top: shown.y, width: shown.w, height: shown.h }}
      />
      <span
        className="absolute whitespace-nowrap rounded bg-popover px-1.5 py-0.5 text-xs tabular-nums text-popover-foreground shadow"
        style={{ left: shown.x, top: shown.y + shown.h + 4 }}
      >
        {t(`preset.${shown.preset}`)} ·{" "}
        {t("coords", {
          x: shown.x - (shown.origin?.x ?? 0),
          y: shown.y - (shown.origin?.y ?? 0),
        })}
      </span>
    </div>
  );
}

export function SnapGuides({ guides }: { guides: readonly Guide[] }) {
  return (
    <>
      {guides.map((guide) => (
        <div
          key={`${guide.axis}${guide.at}`}
          data-snap-guide={guide.axis}
          className="absolute bg-accent"
          style={
            guide.axis === "x"
              ? {
                  left: guide.at,
                  top: guide.from - 8,
                  width: 1,
                  height: guide.to - guide.from + 16,
                }
              : {
                  left: guide.from - 8,
                  top: guide.at,
                  width: guide.to - guide.from + 16,
                  height: 1,
                }
          }
        />
      ))}
    </>
  );
}

export function usePlaceComponent(
  projectId: number,
  screen: string | null,
  tag: string | null,
  record?: RecordChange,
) {
  const t = useTranslations("gameViewEdit.components");
  const locale = useLocale();
  const { list, panels } = useScreenPanels(projectId, screen);
  const [busy, setBusy] = useState(false);

  const place = async (
    preset: ComponentPreset,
    x: number,
    y: number,
    opts: {
      into?: UiPanel;
      extra?: Partial<UiPanelControl>;
      size?: { w: number; h: number };
      mount?: string;
    } = {},
  ): Promise<{ panel: UiPanel; path: string } | null> => {
    const mount = opts.mount ?? mountOf(screen);
    if (!mount && !opts.into) {
      toast.error(t("unmountable"));
      return null;
    }
    const words = PRESETS[preset].words ?? "text";
    const own = {
      button: t("defaultButton"),
      text: t("defaultText"),
      title: t("defaultTitle"),
      paragraph: t("defaultParagraph"),
      listItem: t("defaultListItem"),
    }[words];
    const text = { en: PRESET_EN_WORDS[words], locale, own };
    setBusy(true);
    try {
      const target =
        opts.into ??
        (opts.mount
          ? (list.data ?? []).find(
              (one) => one.parent_window_token === opts.mount,
            )
          : panels[0]);
      const existing = target
        ? await uiPanelsApi.get(projectId, target.id)
        : null;
      const token = existing?.token ?? screenPanelToken(tag, mount ?? "");
      let placed = placeComponent(
        existing ?? {
          token,
          window_props: { background_sprite: "" },
          controls: [],
        },
        preset,
        x,
        y,
        text,
        panelScrollOf(token),
        opts.size,
      );
      if (opts.extra)
        placed = {
          ...placed,
          controls: updateControlAtPath(
            placed.controls,
            placed.path,
            opts.extra,
          ),
        };
      const base = {
        country_tag: tag ?? "",
        context_type: contextOfMount(mount),
        parent_window_token: mount ?? "",
        ...panelNames(
          `${SCREEN_EN_NAMES[mount ?? ""] ?? screen} components`,
          locale,
          `${t(`screen.${mount}`)} · ${t("panelName")}`,
        ),
      };
      const panel = existing
        ? await uiPanelsApi.update(projectId, existing.id, {
            window_props: placed.window_props,
            controls: placed.controls,
          })
        : await uiPanelsApi.create(projectId, {
            token,
            ...base,
            window_props: placed.window_props,
            controls: placed.controls,
          });
      record?.(
        "place",
        existing ? panelState(existing) : { token, base, panel: null },
      );

      await Promise.all([
        list.mutate(),
        globalMutate(["game-view-ui-panel", projectId, panel.id], panel, {
          revalidate: false,
        }),
      ]);
      toast.success(t("placed"));
      return { panel, path: placed.path };
    } catch {
      toast.error(t("placeFailed"));
      return null;
    } finally {
      setBusy(false);
    }
  };

  const placeWindow = async (x: number, y: number): Promise<UiPanel | null> => {
    setBusy(true);
    try {
      const taken = new Set((list.data ?? []).map((panel) => panel.token));
      const prefix =
        (tag ?? "").toUpperCase().replace(/[^A-Z0-9]/g, "") || "MOD";
      let n = 1;
      while (taken.has(`${prefix}_window_${n}_ui`)) n += 1;
      const token = `${prefix}_window_${n}_ui`;
      const made = newWindowPanel(token, x, y, {
        en: "New panel",
        locale,
        own: t("defaultWindowTitle"),
      });
      const base = {
        country_tag: tag ?? "",
        context_type: "player_context",
        parent_window_token: "",
        ...panelNames(`Panel ${n}`, locale, `${t("windowPanelName")} ${n}`),
      };
      const panel = await uiPanelsApi.create(projectId, { ...made, ...base });
      record?.("place", { token, base, panel: null });
      await Promise.all([
        list.mutate(),
        globalMutate(["game-view-ui-panel", projectId, panel.id], panel, {
          revalidate: false,
        }),
      ]);
      toast.success(t("windowPlaced"));
      return panel;
    } catch {
      toast.error(t("placeFailed"));
      return null;
    } finally {
      setBusy(false);
    }
  };
  return { place, placeWindow, busy };
}

export function useEditWords(projectId: number, record?: RecordChange) {
  const t = useTranslations("gameViewEdit.components");
  const locale = useLocale();
  return async (
    envelope: { panel_id: number; controls: Record<string, { path: string }> },
    nodeName: string | null | undefined,
    words: string,
  ): Promise<UiPanel | null> => {
    const names = pickedControlNames(envelope, nodeName);
    if (!names) return null;
    try {
      const fresh = await uiPanelsApi.get(projectId, envelope.panel_id);
      const path = controlPathOf(fresh.controls ?? [], names);
      const control = path
        ? getControlAtPath(fresh.controls ?? [], path)
        : null;
      if (!path || !control) return null;
      const locs = { ...(control.text_locs ?? {}), [locale]: words };

      const patch: Partial<UiPanelControl> =
        locale === "en"
          ? { text: words, text_locs: locs }
          : {
              text_locs: locs,
              ...(control.text?.trim() ? {} : { text: words }),
            };
      const controls = updateControlAtPath(fresh.controls ?? [], path, patch);
      const panel = await uiPanelsApi.update(projectId, fresh.id, { controls });
      record?.("edit", panelState(fresh));
      return panel;
    } catch {
      toast.error(t("saveFailed"));
      return null;
    }
  };
}

export function useMoveWindow(projectId: number, record?: RecordChange) {
  const t = useTranslations("gameViewEdit.components");
  return async (
    panelId: number,
    to: { x?: number; y?: number; width?: number; height?: number },
  ): Promise<UiPanel | null> => {
    try {
      const fresh = await uiPanelsApi.get(projectId, panelId);
      const window_props = { ...fresh.window_props };
      for (const [key, value] of Object.entries(to))
        if (value !== undefined)
          window_props[key] = Math.max(
            key === "width" || key === "height" ? 40 : -10000,
            Math.round(value),
          );
      const panel = await uiPanelsApi.update(projectId, fresh.id, {
        window_props,
      });
      record?.("window", panelState(fresh));
      return panel;
    } catch {
      toast.error(t("moveFailed"));
      return null;
    }
  };
}

export function useMoveComponent(projectId: number, record?: RecordChange) {
  const t = useTranslations("gameViewEdit.components");
  return async (
    envelope: { panel_id: number; controls: Record<string, { path: string }> },
    nodeName: string | null | undefined,
    to: { x?: number; y?: number; width?: number; height?: number },
  ): Promise<{ panel: UiPanel; path: string; from: string } | null> => {
    const names = pickedControlNames(envelope, nodeName);
    if (!names) return null;
    try {
      const fresh = await uiPanelsApi.get(projectId, envelope.panel_id);
      const path = controlPathOf(fresh.controls ?? [], names);
      if (!path) return null;
      const moved = relocateControl(
        fresh,
        path,
        to,
        panelScrollOf(fresh.token),
      );
      const panel = await uiPanelsApi.update(projectId, fresh.id, {
        window_props: moved.window_props,
        controls: moved.controls,
      });
      record?.(
        to.x === undefined && to.y === undefined ? "resize" : "move",
        panelState(fresh),
      );
      return { panel, path: moved.path, from: path };
    } catch {
      toast.error(t("moveFailed"));
      return null;
    }
  };
}

type PanelFields = Pick<
  UiPanel,
  | "name_en"
  | "name_zh"
  | "display_name_locs"
  | "visible_trigger"
  | "window_props"
>;

function showModeOf(
  panel: Pick<UiPanel, "token" | "visible_trigger">,
): "always" | "button" | "custom" {
  if (opensByButton(panel)) return "button";
  const trigger = (panel.visible_trigger ?? "").trim();
  return !trigger || trigger === "always = yes" ? "always" : "custom";
}

export function openersOf(panels: readonly UiPanel[], token: string): number {
  const flag = openFlagOf(token);
  let count = 0;
  const walk = (controls: readonly UiPanelControl[]) => {
    for (const control of controls) {
      if (
        control.kind === "button" &&
        flagOpsOf(control.click_effect).some(
          (op) => op.flag === flag && op.op !== "clear",
        )
      )
        count += 1;
      if (control.children) walk(control.children);
    }
  };
  for (const panel of panels) walk(panel.controls ?? []);
  return count;
}

const FIELD =
  "h-8 w-full rounded-md border border-input bg-card px-2 text-xs text-foreground ui-focus-ring";

function PanelSettings({
  panel,
  onChange,
  openers,
  canAddOpener,
  onAddOpener,
  onDeletePanel,
  busy,
}: {
  panel: UiPanel;
  onChange: (
    fields: Partial<PanelFields>,
    controls?: (controls: UiPanelControl[]) => UiPanelControl[],
  ) => void;
  openers: number;
  canAddOpener: boolean;
  onAddOpener?: () => void;
  onDeletePanel: () => void;
  busy: boolean;
}) {
  const t = useTranslations("gameViewEdit.components");
  const locale = useLocale();
  const mode = showModeOf(panel);
  const flag = openFlagOf(panel.token);
  const windowPanel = isWindowPanel(panel);
  const props = panel.window_props ?? {};
  const setProp = (key: string, value: unknown) =>
    onChange({ window_props: { ...props, [key]: value } });
  const setMode = (next: "always" | "button" | "custom") => {
    if (next === "always") onChange({ visible_trigger: "" });
    else if (next === "custom")
      onChange({
        visible_trigger: mode === "custom" ? panel.visible_trigger : "",
      });
    else
      onChange(
        { visible_trigger: openTrigger(flag) },

        windowPanel
          ? (controls) =>
              controls.some((control) =>
                flagOpsOf(control.click_effect).some(
                  (op) => op.flag === flag && op.op === "clear",
                ),
              )
                ? controls
                : [
                    ...controls,
                    {
                      kind: "button",
                      name: uniqueControlName(controls, "close"),
                      x: -42,
                      y: 9,
                      w: 32,
                      h: 33,
                      sprite: "GFX_closebutton",
                      text: "",
                      click_effect: closeFlagEffect(flag),
                      click_enabled_trigger: "",
                      raw_extras: { orientation: "UPPER_RIGHT" },
                    },
                  ]
          : undefined,
      );
  };
  return (
    <section
      data-panel-settings=""
      className="space-y-3 rounded-md border border-border p-3"
    >
      <p className="text-xs font-medium text-foreground">
        {t("settingsTitle")} · <span className="font-mono">{panel.token}</span>
      </p>
      <label className="block space-y-1">
        <span className="text-xs text-muted-foreground">
          {t("settingsName")}
        </span>
        <input
          className={FIELD}
          value={
            panelLabel(panel, locale) === panel.token
              ? ""
              : panelLabel(panel, locale)
          }
          onChange={(e) =>
            onChange(renamedPanel(panel, locale, e.target.value))
          }
        />
      </label>
      <div className="space-y-1">
        <span className="text-xs text-muted-foreground">
          {t("settingsShow")}
        </span>
        <div
          role="radiogroup"
          aria-label={t("settingsShow")}
          className="inline-flex rounded-md border border-border bg-card p-0.5"
        >
          {(["always", "button", "custom"] as const).map((option) => (
            <button
              key={option}
              type="button"
              role="radio"
              aria-checked={mode === option}
              onClick={() => setMode(option)}
              className={`h-7 rounded px-2.5 text-xs ui-interactive ui-focus-ring ${mode === option ? "bg-accent text-accent-foreground" : "text-muted-foreground hover:bg-muted"}`}
            >
              {t(`show_${option}`)}
            </button>
          ))}
        </div>
        {mode === "button" && (
          <div className="flex flex-wrap items-center gap-2 pt-1">
            <span
              data-openers={openers}
              className={`text-xs ${openers ? "text-muted-foreground" : "text-warning"}`}
            >
              {openers ? t("openers", { count: openers }) : t("openersNone")}
            </span>
            {onAddOpener && (
              <button
                type="button"
                className={BUTTON}
                disabled={!canAddOpener || busy}
                title={canAddOpener ? undefined : t("addOpenerUnmountable")}
                onClick={onAddOpener}
              >
                {t("addOpener")}
              </button>
            )}
          </div>
        )}
        {mode === "custom" && (
          <textarea
            aria-label={t("show_custom")}
            className="mt-1 h-20 w-full rounded-md border border-input bg-card p-2 font-mono text-xs text-foreground ui-focus-ring"
            placeholder="has_country_flag = my_flag"
            value={panel.visible_trigger ?? ""}
            onChange={(e) => onChange({ visible_trigger: e.target.value })}
          />
        )}
      </div>
      {windowPanel && (
        <div className="space-y-2">
          <div className="grid grid-cols-4 gap-2">
            {(
              [
                ["x", "windowX"],
                ["y", "windowY"],
                ["width", "windowW"],
                ["height", "windowH"],
              ] as const
            ).map(([key, label]) => (
              <label key={key} className="space-y-1">
                <span className="text-xs text-muted-foreground">
                  {t(label)}
                </span>
                <input
                  type="number"
                  className={`${FIELD} font-mono`}
                  value={String(props[key] ?? "")}
                  onChange={(e) => setProp(key, Number(e.target.value) || 0)}
                />
              </label>
            ))}
          </div>
          <label className="block space-y-1">
            <span className="text-xs text-muted-foreground">
              {t("windowBackground")}
            </span>
            <input
              className={`${FIELD} font-mono`}
              value={String(props.background_sprite ?? "")}
              placeholder={WINDOW_LOOK.sprite}
              onChange={(e) => setProp("background_sprite", e.target.value)}
            />
          </label>
          <label className="flex items-center gap-2 text-xs text-foreground">
            <input
              type="checkbox"
              className="h-4 w-4 accent-accent ui-focus-ring"
              checked={props.moveable === true}
              onChange={(e) => setProp("moveable", e.target.checked)}
            />
            {t("windowMoveable")}
          </label>
        </div>
      )}
      <button
        type="button"
        className={BUTTON}
        disabled={busy}
        onClick={onDeletePanel}
      >
        {t("deletePanel")}
      </button>
    </section>
  );
}

export function PanelComponentEditor({
  projectId,
  panelId,
  path = null,
  names = null,
  onClose,
  onSaved,
  onRecord,
  panels = [],
  canAddOpener = false,
  onAddOpener,
  onDeleted,
}: {
  projectId: number;
  panelId: number;

  path?: string | null;

  names?: string | null;
  onClose: () => void;

  onSaved: (panel: UiPanel) => Promise<void> | void;

  onRecord?: RecordChange;

  panels?: UiPanel[];

  canAddOpener?: boolean;
  onAddOpener?: (panel: UiPanel) => Promise<void> | void;

  onDeleted?: (panel: UiPanel) => Promise<void> | void;
}) {
  const t = useTranslations("gameViewEdit.components");
  const locale = useLocale();
  const confirm = useConfirm();
  const panel = useSWR(
    ["game-view-ui-panel", projectId, panelId],
    () => uiPanelsApi.get(projectId, panelId),
    { revalidateOnFocus: false },
  );
  const [draft, setDraft] = useState<UiPanelControl[] | null>(null);
  const [fields, setFields] = useState<Partial<PanelFields> | null>(null);
  const [busy, setBusy] = useState(false);
  const selectedPath = useUiPanelSelection((s) => s.selectedPath);
  const setSelectedPath = useUiPanelSelection((s) => s.setSelectedPath);
  const saved = panel.data;
  const controls = draft ?? saved?.controls ?? [];
  const shown = saved ? { ...saved, ...fields } : null;
  const unsaved =
    (draft !== null &&
      JSON.stringify(draft) !== JSON.stringify(saved?.controls ?? [])) ||
    (fields !== null &&
      saved !== undefined &&
      Object.entries(fields).some(
        ([key, value]) =>
          JSON.stringify(value) !== JSON.stringify(saved[key as keyof UiPanel]),
      ));

  const picked =
    path ??
    (saved && names ? controlPathOf(saved.controls ?? [], names) : null);
  useEffect(() => {
    useUiPanelSelection.getState().setSelectedPath(picked);
  }, [picked]);

  const save = async (
    next: UiPanelControl[],
    change: ComponentChange = "edit",
  ): Promise<boolean> => {
    if (!saved || !shown) return false;
    setBusy(true);
    try {
      const fitted = fitFor(saved, shown.window_props ?? {}, next);
      const updated = await uiPanelsApi.update(projectId, saved.id, {
        ...(fields ?? {}),
        window_props: fitted.window_props,
        controls: fitted.controls,
      });
      onRecord?.(change, panelState(saved));
      await panel.mutate(updated, { revalidate: false });
      setDraft(null);
      setFields(null);
      await onSaved(updated);
      if (change === "edit") toast.success(t("saved"));
      return true;
    } catch {
      toast.error(t("saveFailed"));
      return false;
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    if (!selectedPath) return;
    if (
      !(await confirm({
        title: t("deleteTitle"),
        body: t("deleteBody"),
        danger: true,
        confirmLabel: t("delete"),
      }))
    )
      return;
    setSelectedPath(null);
    if (await save(removeControlAtPath(controls, selectedPath), "delete"))
      toast.info(t("deletedUndo"));
  };

  const removeNow = useRef<() => void>(() => {});
  useEffect(() => {
    removeNow.current = () => {
      if (!selectedPath || busy) return;
      setSelectedPath(null);
      void save(removeControlAtPath(controls, selectedPath), "delete").then(
        (done) => {
          if (done) toast.info(t("deletedUndo"));
        },
      );
    };
  });
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== "Delete" && event.key !== "Backspace") return;
      const target = event.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.tagName === "SELECT" ||
          target.isContentEditable)
      )
        return;
      event.preventDefault();
      removeNow.current();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const removePanel = async () => {
    if (!saved) return;
    if (
      !(await confirm({
        title: t("deletePanelTitle"),
        body: t("deletePanelBody"),
        danger: true,
        confirmLabel: t("deletePanel"),
      }))
    )
      return;
    setBusy(true);
    try {
      await uiPanelsApi.delete(projectId, saved.id);
      onRecord?.("panel", panelState(saved));
      toast.success(t("panelDeleted"));
      await onDeleted?.(saved);
    } catch {
      toast.error(t("saveFailed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <GamePanel
      title={t("panelTitle")}
      kind={
        saved
          ? `${t("panelKind")} · ${panelLabel(saved, locale)}`
          : t("panelKind")
      }
      source="项目"
      workbenchHref={`/project/${projectId}/ui-panels`}
      wide="workbench"
      onClose={onClose}
    >
      {!saved || !shown ? (
        <p className="mt-3 text-xs text-muted-foreground">
          {panel.error ? t("loadFailed") : t("loading")}
        </p>
      ) : (
        <div className="mt-3 space-y-3">
          <p className="text-xs text-muted-foreground">
            {isWindowPanel(saved)
              ? t("exportNoteWindow")
              : t("exportNote", { mount: saved.parent_window_token || "—" })}
          </p>
          <PanelSettings
            panel={shown}
            busy={busy}
            openers={openersOf(
              panels.map((other) =>
                other.id === saved.id ? { ...other, controls } : other,
              ),
              saved.token,
            )}
            canAddOpener={canAddOpener}
            onAddOpener={
              onAddOpener ? () => void onAddOpener(shown) : undefined
            }
            onDeletePanel={() => void removePanel()}
            onChange={(next, change) => {
              setFields((current) => ({ ...(current ?? {}), ...next }));
              if (change) setDraft(change(controls));
            }}
          />

          <PreferredLanguage.Provider value={locale}>
            <InspectorArea
              projectId={projectId}
              panel={saved}
              controls={controls}
              onControlsChange={setDraft}
            />
          </PreferredLanguage.Provider>
          <div className="flex flex-wrap items-center gap-2 border-t border-border pt-3">
            <button
              type="button"
              className={PRIMARY}
              disabled={!unsaved || busy}
              onClick={() => void save(controls)}
            >
              {busy ? t("saving") : t("save")}
            </button>
            <button
              type="button"
              className={BUTTON}
              disabled={!unsaved || busy}
              onClick={() => {
                setDraft(null);
                setFields(null);
              }}
            >
              {t("discard")}
            </button>
            <button
              type="button"
              className={BUTTON}
              disabled={!selectedPath || busy}
              onClick={() => void remove()}
            >
              {t("delete")}
            </button>
            {unsaved && (
              <span className="text-xs text-warning">{t("unsaved")}</span>
            )}
          </div>
        </div>
      )}
    </GamePanel>
  );
}
