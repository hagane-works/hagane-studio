"use client";

import { useContext, useMemo, useState, type CSSProperties } from "react";
import { ChevronUp } from "lucide-react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiAlert,
  type GuiAlerts,
  type GuiCountryStats,
  type GuiNode,
  type GuiSprite,
  type GuiWindow,
} from "@/lib/guiRenderApi";
import { GuiCanvas, assignIds } from "../gui-renderer/GuiCanvas";
import { gameTooltipProps } from "../gui-renderer/GameTooltip";
import type { GameTooltipContent } from "../gui-renderer/tooltipText";
import { BLEND_FRAMES_ANIMATION, BlendFramesKeyframes } from "./blendFrames";
import { useTemplateWindow } from "./CountrySelectLayer";
import { EditContext, useEditCanvas } from "./editContext";
import { ALERT_TARGETS } from "./gameNavigation";
import { guiArray } from "./guiResponse";
import { frameStyle } from "./spriteFrame";
import { applySwaps } from "./spriteSwaps";
import { fillLoc } from "./topbarTooltips";

export const ALERT_TEMPLATE = "global_alerticon_window";

type Point = { x: number; y: number };

export interface AlertPositions {
  alerticon_startposition: Point;
  alerticon_startposition_extended: Point;
  alerticon_offset: Point;
  alerticon_endposition: Point;
}

export const VANILLA_ALERT_POSITIONS: AlertPositions = {
  alerticon_startposition: { x: 662, y: 38 },
  alerticon_startposition_extended: { x: 718, y: 38 },
  alerticon_offset: { x: 48, y: 44 },
  alerticon_endposition: { x: 315, y: 0 },
};

type AlertRow = GuiAlert & { category?: string; type?: number };
type AlertsPayload = Omit<GuiAlerts, "alerts"> & {
  alerts: AlertRow[];
  positions?: Partial<AlertPositions>;
};

export function alertSlots(
  count: number,
  positions: AlertPositions,
  rowWidth: number,
  iconWidth: number,
  extended = true,
): Point[] {
  const start = extended
    ? positions.alerticon_startposition_extended
    : positions.alerticon_startposition;
  const step = positions.alerticon_offset;
  const limit = rowWidth - positions.alerticon_endposition.x;
  const out: Point[] = [];
  let column = 0;
  let row = 0;
  for (let i = 0; i < count; i += 1) {
    if (column > 0 && start.x + column * step.x + iconWidth > limit) {
      column = 0;
      row += 1;
    }
    out.push({ x: start.x + column * step.x, y: start.y + row * step.y });
    column += 1;
  }
  return out;
}

export function alertsShown(
  slots: Point[],
  count: number,
  expanded: boolean,
): { shown: number; toggle: Point | null; hidden: number } {
  const firstLine = slots
    .slice(0, count)
    .filter((slot) => slot.y === slots[0]?.y).length;
  if (count <= firstLine) return { shown: count, toggle: null, hidden: 0 };
  if (expanded)
    return { shown: count, toggle: slots[count] ?? null, hidden: 0 };
  const shown = Math.max(0, firstLine - 1);
  return { shown, toggle: slots[shown] ?? null, hidden: count - shown };
}

const ALERTS_EXPANDED_KEY = "gameview.alerts.expanded";

function readAlertsExpanded(): boolean {
  try {
    return (
      typeof window !== "undefined" &&
      window.localStorage.getItem(ALERTS_EXPANDED_KEY) === "1"
    );
  } catch {
    return false;
  }
}

function writeAlertsExpanded(expanded: boolean) {
  try {
    window.localStorage.setItem(ALERTS_EXPANDED_KEY, expanded ? "1" : "0");
  } catch {}
}

export function alertTooltip(
  alert: AlertRow,
  stats: GuiCountryStats | undefined,
): GameTooltipContent {
  const vars: Record<string, number> = {};
  if (alert.name === "alert_no_research" && stats?.research_slots !== undefined)
    vars.COUNT = stats.research_slots;
  return {
    text: alert.instant,
    delayed: alert.delayed ? fillLoc(alert.delayed, vars) : null,
  };
}

function point(value: unknown): Point | null {
  if (!value || typeof value !== "object") return null;
  const { x, y } = value as { x?: unknown; y?: unknown };
  const nx = Number(x);
  const ny = Number(y);
  return Number.isFinite(nx) && Number.isFinite(ny) ? { x: nx, y: ny } : null;
}

function positionsOf(payload: AlertsPayload | undefined): AlertPositions {
  const given = payload?.positions ?? {};
  const out = { ...VANILLA_ALERT_POSITIONS };
  for (const key of Object.keys(out) as (keyof AlertPositions)[])
    out[key] = point(given[key]) ?? out[key];
  return out;
}

export function alertEditNode(
  template: GuiNode,
  frame: number,
  high: boolean,
): GuiNode {
  const clone = structuredClone(template) as GuiNode;
  const otherGlow = high ? "yellow_alert_glow" : "red_alert_glow";
  clone.children = (clone.children ?? []).filter(
    (child) => child.name !== otherGlow,
  );
  for (const child of clone.children)
    if (child.name === "icon") child.frame = frame;
  assignIds(clone);
  return clone;
}

const NOTHING_HIDDEN = new Set<string>();

function AlertEditTile({
  win,
  alert,
  at,
  width,
  height,
}: {
  win: GuiWindow;
  alert: AlertRow;
  at: Point;
  width: number;
  height: number;
}) {
  const edit = useEditCanvas();
  const node = useMemo(
    () => alertEditNode(win.node, alert.frame, alert.category === "HIGH"),
    [win.node, alert.frame, alert.category],
  );
  return (
    <div style={{ position: "absolute", left: at.x, top: at.y }}>
      <GuiCanvas
        node={node}
        {...edit(
          `${ALERT_TEMPLATE}:${alert.type ?? alert.name}:${alert.frame}`,
          win,
          node,
          win.sprites,
        )}
        texts={win.texts}
        fonts={win.fonts}
        hidden={NOTHING_HIDDEN}
        isolateId={null}
        transparent
        baseW={width}
        baseH={height}
      />
    </div>
  );
}

function childNamed(root: GuiNode, name: string): GuiNode | undefined {
  return (root.children ?? []).find((child) => child.name === name);
}

function offsetOf(node: GuiNode | undefined): Point {
  return {
    x: Number(node?.position?.x) || 0,
    y: Number(node?.position?.y) || 0,
  };
}

export function AlertsLayer({
  projectId,
  topbar,
  stats,
  onOpen,
  onUnavailable,
}: {
  projectId: number;

  topbar: GuiNode;
  stats: GuiCountryStats | undefined;
  onOpen?: (window: string) => void;

  onUnavailable?: (control: string) => void;
}) {
  const payload = useSWR<AlertsPayload>(
    ["game-view-alerts", projectId],
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.alerts() as Promise<AlertsPayload>;
    },

    { revalidateOnFocus: false, shouldRetryOnError: false },
  );
  const template = useTemplateWindow(
    projectId,
    guiArray(payload.data?.alerts).length ? ALERT_TEMPLATE : null,
  );
  const { editMode: uiEdit, swaps } = useContext(EditContext);
  const t = useTranslations("gameViewControls");
  const [expanded, setExpanded] = useState(readAlertsExpanded);

  const layout = useMemo(() => {
    const loaded = template.data;
    const alerts = guiArray(payload.data?.alerts);
    if (!loaded || !alerts.length) return null;

    const { sprites, versions } = applySwaps(loaded.sprites, swaps);
    const win = { ...loaded, sprites };
    const iconNode = childNamed(win.node, "icon");
    const iconName = iconNode?.sprite || payload.data?.sprite || "";
    const icon: GuiSprite | undefined = win.sprites[iconName];
    if (!icon?.resolved) return null;
    const red = childNamed(win.node, "red_alert_glow");
    const yellow = childNamed(win.node, "yellow_alert_glow");
    const iconW = Math.round(icon.width / Math.max(1, icon.frames));
    const size = win.node.size ?? {};
    const templateW = Number(size.width ?? size.x) || iconW;
    const rowWidth = Number(topbar.size?.width ?? topbar.size?.x) || 1920;
    const extended = (topbar.children ?? []).some((child) =>
      (child.sprite ?? "").endsWith("_extended"),
    );

    const slots = alertSlots(
      alerts.length + 1,
      positionsOf(payload.data),
      rowWidth,
      templateW,
      extended,
    );
    return {
      win,
      versions,
      alerts,
      icon,
      iconName,
      iconNode,
      red,
      yellow,
      slots,
      templateW,
    };
  }, [template.data, payload.data, topbar, swaps]);

  if (!layout) return null;
  const {
    win,
    versions,
    alerts,
    icon,
    iconName,
    iconNode,
    red,
    yellow,
    slots,
    templateW,
  } = layout;
  const iconAt = offsetOf(iconNode);
  const iconW = Math.round(icon.width / Math.max(1, icon.frames));
  const view = alertsShown(slots, alerts.length, expanded);
  const shownAlerts = alerts.slice(0, view.shown);
  const toggleLabel = expanded
    ? t("alertsLess")
    : t("alertsMore", { count: view.hidden });
  const toggle = view.toggle && (
    <button
      type="button"
      aria-expanded={expanded}
      aria-label={toggleLabel}
      title={toggleLabel}
      onClick={() => {
        setExpanded(!expanded);
        writeAlertsExpanded(!expanded);
      }}
      className="ui-interactive flex items-center justify-center rounded-sm border border-ingame-edge bg-ingame-panel/90 text-xs font-medium text-ingame-gold shadow-lg hover:border-ingame-gold/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
      style={{
        position: "absolute",
        left: view.toggle.x + iconAt.x,
        top: view.toggle.y + iconAt.y,
        width: iconW,
        height: icon.height,
        pointerEvents: "auto",
      }}
    >
      {expanded ? <ChevronUp size={18} aria-hidden /> : `+${view.hidden}`}
    </button>
  );

  if (uiEdit) {
    const tileH = Math.max(
      icon.height + iconAt.y,
      Number(win.node.size?.height ?? win.node.size?.y) || 0,
    );
    return (
      <div
        style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
      >
        {shownAlerts.map((alert, i) => (
          <AlertEditTile
            key={`${alert.type ?? alert.name}-${alert.frame}`}
            win={win}
            alert={alert}
            at={slots[i]}
            width={Math.max(templateW, iconAt.x + iconW)}
            height={tileH}
          />
        ))}
        {toggle}
      </div>
    );
  }

  const art = (
    sprite: GuiSprite,
    left: number,
    top: number,
    frame: number,
    name: string | null | undefined,
  ): CSSProperties => {
    const style = frameStyle(sprite, left, top, { frame, name });
    const version = name ? versions[name] : undefined;
    return version && name
      ? {
          ...style,
          backgroundImage: `url("${guiRenderApi.spriteUrl(name, version)}")`,
        }
      : style;
  };

  return (
    <div
      style={{ position: "absolute", left: 0, top: 0, pointerEvents: "none" }}
    >
      <BlendFramesKeyframes />
      {shownAlerts.map((alert, i) => {
        const at = slots[i];
        const glowNode = alert.category === "HIGH" ? red : yellow;
        const glow = glowNode?.sprite
          ? win.sprites[glowNode.sprite]
          : undefined;
        const glowAt = offsetOf(glowNode);
        const target = ALERT_TARGETS[alert.name];
        return (
          <div
            key={`${alert.type ?? alert.name}-${alert.frame}`}
            style={{ position: "absolute", left: at.x, top: at.y }}
          >
            <div style={art(icon, iconAt.x, iconAt.y, alert.frame, iconName)} />
            {glow?.resolved && (
              <>
                <div
                  style={art(glow, glowAt.x, glowAt.y, 1, glowNode?.sprite)}
                />
                <div
                  style={{
                    ...art(glow, glowAt.x, glowAt.y, 2, glowNode?.sprite),
                    animation: BLEND_FRAMES_ANIMATION,
                  }}
                />
              </>
            )}
            <div
              role="button"
              aria-label={alert.name}
              onClick={
                target && onOpen
                  ? () => onOpen(target)
                  : onUnavailable
                    ? () => onUnavailable(alert.name)
                    : undefined
              }
              style={{
                position: "absolute",
                left: iconAt.x,
                top: iconAt.y,
                width: iconW,
                height: icon.height,
                pointerEvents: "auto",
              }}
              {...gameTooltipProps(alertTooltip(alert, stats), {
                key: `alert:${alert.type ?? alert.name}:${alert.frame}`,
              })}
            />
          </div>
        );
      })}
      {toggle}
    </div>
  );
}
