"use client";

import {
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type PointerEvent,
  type ReactNode,
} from "react";
import { ChevronDown, GripHorizontal, PencilLine } from "lucide-react";
import { useTranslations } from "use-intl";
import { clearOfMapModes } from "./gameStage";
import { OpenInWorkbench } from "./OpenInWorkbench";

const BALL = 48;
const BALL_GAP = 8;
const PANEL_TOP = 64;
const PANEL_BOTTOM = 56 + BALL + BALL_GAP;

export const EDIT_SWITCHER_HEIGHT = 64;
export const EDIT_SWITCHER_SPACE = `${EDIT_SWITCHER_HEIGHT + BALL_GAP}px`;

const COLUMN_RIGHT = `max(${clearOfMapModes(12).right}, var(--component-bar, 0px))`;

export const PANEL_PLACE = {
  open: {
    right: COLUMN_RIGHT,
    top: `calc(${PANEL_TOP}px + var(--edit-switcher, 0px))`,
    maxHeight: `calc(100vh - ${PANEL_TOP + PANEL_BOTTOM}px - var(--edit-switcher, 0px))`,
  },
  folded: { ...clearOfMapModes(12 + BALL + BALL_GAP), bottom: 56 },
  switcher: {
    right: COLUMN_RIGHT,
    top: PANEL_TOP,
    maxHeight: `calc(100vh - ${PANEL_TOP + PANEL_BOTTOM}px)`,
  },
} as const;

const PANEL_POSITION_KEY = "gameview.editPanel.position.v1";
type PanelPosition = { left: number; top: number };
let panelPosition: PanelPosition | null | undefined;
const panelPositionListeners = new Set<() => void>();

function readPanelPosition(): PanelPosition | null {
  if (panelPosition === undefined) {
    try {
      const raw = window.localStorage.getItem(PANEL_POSITION_KEY);
      const at = raw ? (JSON.parse(raw) as Partial<PanelPosition>) : null;
      panelPosition =
        at && Number.isFinite(at.left) && Number.isFinite(at.top)
          ? { left: at.left as number, top: at.top as number }
          : null;
    } catch {
      panelPosition = null;
    }
  }
  return panelPosition;
}

export function setPanelPosition(next: PanelPosition | null, keep = true) {
  panelPosition = next;
  if (keep) {
    try {
      if (next)
        window.localStorage.setItem(PANEL_POSITION_KEY, JSON.stringify(next));
      else window.localStorage.removeItem(PANEL_POSITION_KEY);
    } catch {}
  }
  panelPositionListeners.forEach((listener) => listener());
}

function subscribePanelPosition(listener: () => void) {
  panelPositionListeners.add(listener);
  return () => panelPositionListeners.delete(listener);
}

function movedPlace(at: PanelPosition, widthRem: number) {
  const top = `max(8px, min(${at.top}px, calc(100vh - 48px)))`;
  return {
    left: `max(8px, min(${at.left}px, calc(100vw - ${widthRem}rem - 8px)))`,
    top,
    maxHeight: `calc(100vh - ${top} - 8px)`,
  };
}

export function EditSwitcherDock({ children }: { children: ReactNode }) {
  return (
    <div
      data-edit-switcher=""
      style={PANEL_PLACE.switcher}
      className="pointer-events-auto fixed z-[51] w-96 overflow-y-auto"
    >
      {children}
    </div>
  );
}

export const FIELD =
  "mt-1 w-full rounded border border-input bg-background px-2 py-1 text-sm";

export const NUMBER = `${FIELD} tabular-nums`;

export const SCRIPT = `${FIELD} font-mono text-xs`;

export const BUTTON =
  "rounded border border-border px-2 py-1 text-xs hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50";

export const PRIMARY =
  "rounded bg-accent px-3 py-1 text-xs text-accent-foreground hover:bg-accent/90 disabled:cursor-not-allowed disabled:opacity-50";

export type PanelSource = "项目" | "原版" | "mod 文件";

const SOURCE_KEY = {
  项目: "project",
  原版: "base",
  "mod 文件": "modFile",
} as const;

export function GamePanel({
  title,
  kind,
  detail,
  source,
  workbenchHref,
  wide = true,
  onClose,
  children,
}: {
  title: ReactNode;

  kind?: ReactNode;

  detail?: ReactNode;
  source?: PanelSource;

  workbenchHref?: string | null;

  wide?: boolean | "workbench";
  onClose: () => void;
  children: ReactNode;
}) {
  const t = useTranslations("gameViewEdit.panel");
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const name = `${typeof kind === "string" ? kind : ""}|${typeof title === "string" ? title : ""}`;
  const [folded, setFolded] = useState(false);
  const [shownName, setShownName] = useState(name);
  if (shownName !== name) {
    setShownName(name);
    setFolded(false);
  }
  const label =
    typeof title === "string"
      ? title
      : typeof kind === "string"
        ? kind
        : t("editPanel");

  const moved = useSyncExternalStore(
    subscribePanelPosition,
    readPanelPosition,
    () => null,
  );
  const box = useRef<HTMLDivElement>(null);
  const drag = useRef<{
    pointerId: number;
    x: number;
    y: number;
    from: PanelPosition;
  } | null>(null);
  const widthRem = wide === "workbench" ? 36 : wide ? 24 : 20;

  return (
    <>
      {folded && (
        <button
          type="button"
          aria-label={t("expand", { label })}
          title={t("expand", { label })}
          onClick={() => setFolded(false)}
          style={PANEL_PLACE.folded}
          className="ui-interactive ui-focus-ring pointer-events-auto fixed z-50 flex h-12 w-12 items-center justify-center rounded-full border border-accent bg-card text-accent shadow-lg hover:bg-muted"
        >
          <PencilLine className="h-5 w-5" aria-hidden />
        </button>
      )}
      <div
        data-game-panel=""
        hidden={folded}
        ref={box}
        style={moved ? movedPlace(moved, widthRem) : PANEL_PLACE.open}
        className={`pointer-events-auto fixed z-50 ${wide === "workbench" ? "w-[36rem]" : wide ? "w-96" : "w-80"} overflow-y-auto rounded-md border border-border bg-card px-4 pb-4 text-card-foreground shadow-lg`}
      >
        <div
          role="button"
          tabIndex={-1}
          data-panel-handle=""
          aria-label={t("move")}
          title={t("move")}
          onPointerDown={(event: PointerEvent<HTMLDivElement>) => {
            if (event.button !== 0 || !box.current) return;
            event.currentTarget.setPointerCapture?.(event.pointerId);
            const rect = box.current.getBoundingClientRect();
            drag.current = {
              pointerId: event.pointerId,
              x: event.clientX,
              y: event.clientY,
              from: { left: rect.left, top: rect.top },
            };
          }}
          onPointerMove={(event: PointerEvent<HTMLDivElement>) => {
            const start = drag.current;
            if (!start || start.pointerId !== event.pointerId || !box.current)
              return;
            const { width } = box.current.getBoundingClientRect();
            const left = Math.min(
              Math.max(8, start.from.left + event.clientX - start.x),
              Math.max(8, window.innerWidth - width - 8),
            );
            const top = Math.min(
              Math.max(8, start.from.top + event.clientY - start.y),
              Math.max(8, window.innerHeight - 48),
            );
            setPanelPosition(
              { left: Math.round(left), top: Math.round(top) },
              false,
            );
          }}
          onPointerUp={(event: PointerEvent<HTMLDivElement>) => {
            const start = drag.current;
            if (!start || start.pointerId !== event.pointerId) return;
            drag.current = null;
            setPanelPosition(readPanelPosition());
          }}
          onDoubleClick={() => setPanelPosition(null)}
          className="sticky top-0 z-10 -mx-4 mb-1 flex h-5 cursor-move touch-none select-none items-center justify-center bg-card text-muted-foreground/60 hover:bg-muted hover:text-foreground"
        >
          <GripHorizontal className="h-4 w-4" aria-hidden />
        </div>
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <div className="truncate text-sm font-medium">{title}</div>
            {(kind || source) && (
              <div className="mt-0.5 flex min-w-0 items-center gap-1.5">
                {source && (
                  <span
                    className={`shrink-0 rounded border px-1 text-micro ${source === "项目" ? "border-accent/50 text-accent" : "border-border text-muted-foreground"}`}
                    title={t(`sourceHint.${SOURCE_KEY[source]}`)}
                  >
                    {t(`source.${SOURCE_KEY[source]}`)}
                  </span>
                )}
                {kind && (
                  <span className="truncate font-mono text-micro text-muted-foreground">
                    {kind}
                  </span>
                )}
              </div>
            )}
            {detail && (
              <div className="truncate font-mono text-micro text-muted-foreground">
                {detail}
              </div>
            )}
            <OpenInWorkbench href={workbenchHref} />
          </div>
          <div className="flex shrink-0 items-center">
            <button
              type="button"
              aria-label={t("fold")}
              title={t("foldHint")}
              onClick={() => setFolded(true)}
              className="rounded px-1.5 py-1 text-muted-foreground hover:text-foreground"
            >
              <ChevronDown className="h-4 w-4" aria-hidden />
            </button>
            <button
              type="button"
              aria-label={t("close")}
              onClick={onClose}
              className="rounded px-2 text-muted-foreground hover:text-foreground"
            >
              ×
            </button>
          </div>
        </div>
        {children}
      </div>
    </>
  );
}

export function PanelSave({
  onSave,
  canSave,
  busy,
  onRevert,
  label,
  problem,
  note,
  error,
  children,
}: {
  onSave: () => void;

  canSave: boolean;
  busy: boolean;

  onRevert?: () => void;
  label?: string;

  problem?: ReactNode;

  note?: ReactNode;
  error?: string | null;

  children?: ReactNode;
}) {
  const t = useTranslations("gameViewEdit.panel");
  return (
    <div className="mt-3 space-y-2">
      {problem && <p className="text-xs text-destructive">{problem}</p>}
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          className={PRIMARY}
          disabled={!canSave || busy}
          onClick={onSave}
        >
          {busy ? t("saving") : (label ?? t("save"))}
        </button>
        {onRevert && (
          <button
            type="button"
            className={BUTTON}
            disabled={busy}
            onClick={onRevert}
          >
            {t("revert")}
          </button>
        )}
        {children}
      </div>
      {note && <p className="text-xs text-muted-foreground">{note}</p>}
      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  );
}
