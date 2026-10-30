"use client";

import {
  memo,
  useEffect,
  useId,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";
import type {
  CSSProperties,
  PointerEvent as ReactPointerEvent,
  ReactNode,
} from "react";
import useSWR from "swr";
import {
  guiRenderApi,
  type GuiBitmapFont,
  type GuiSprite,
} from "@/lib/guiRenderApi";
import { glyphRun } from "./BitmapText";
import {
  FALLBACK_MEASURE,
  TOOLTIP,
  bitmapMeasure,
  iconNamesOf,
  iconSpriteName,
  layoutTooltip,
  normalizeTooltip,
  placeTooltip,
  tooltipMarkup,
  type GameTooltipInput,
  type TooltipIconSize,
  type TooltipLayout,
} from "./tooltipText";
import {
  attachTooltipHub,
  domElement,
  tooltipHub,
  type ShownTooltip,
  type TooltipSnapshot,
  type TooltipSource,
} from "./tooltipHub";

export type { GameTooltipContent, GameTooltipInput } from "./tooltipText";

export interface TooltipStyle {
  sprite?: GuiSprite;

  font?: GuiBitmapFont;
}

const TOOLTIP_FONT_WINDOWS = ["social_friends_list_item", "nationalfocusview"];

async function loadTooltipFont(): Promise<GuiBitmapFont | undefined> {
  for (const name of TOOLTIP_FONT_WINDOWS) {
    try {
      const win = await guiRenderApi.getWindow(name);
      const bitmap = win.fonts?.[TOOLTIP.font]?.bitmap;
      if (bitmap) return bitmap;
    } catch {}
  }
  return undefined;
}

async function loadTooltipStyle(): Promise<TooltipStyle> {
  const [sprites, font] = await Promise.all([
    guiRenderApi
      .sprites([TOOLTIP.sprite])
      .catch(() => ({}) as Record<string, GuiSprite>),
    loadTooltipFont(),
  ]);
  const sprite = sprites[TOOLTIP.sprite];
  return { sprite: sprite?.resolved ? sprite : undefined, font };
}

export function useTooltipStyle(): TooltipStyle {
  const projectKey = guiRenderApi.currentProjectId();
  const { data } = useSWR(["gui-tooltip-style", projectKey], loadTooltipStyle, {
    revalidateOnFocus: false,
    shouldRetryOnError: false,
  });
  return data ?? {};
}

export function spriteUrl(name: string, sprite: GuiSprite): string {
  return sprite.vanilla_url && !sprite.overridden
    ? sprite.vanilla_url
    : guiRenderApi.spriteUrl(name, sprite.version);
}

export function useTooltipIcons(names: string[]): Record<string, GuiSprite> {
  const projectKey = guiRenderApi.currentProjectId();
  const { data } = useSWR(
    names.length ? ["gui-tooltip-icons", projectKey, names.join(",")] : null,
    () => guiRenderApi.sprites(names.map(iconSpriteName)),
    { revalidateOnFocus: false, shouldRetryOnError: false },
  );
  return useMemo(() => {
    const out: Record<string, GuiSprite> = {};
    if (!data) return out;
    for (const name of names) {
      const sprite = data[iconSpriteName(name)];
      if (sprite?.resolved) out[name] = sprite;
    }
    return out;
  }, [data, names]);
}

export function frameSize(sprite: GuiSprite): TooltipIconSize {
  const frames = Math.max(1, sprite.frames || 1);
  return {
    w: Math.max(1, Math.round(sprite.width / frames)),
    h: Math.max(1, sprite.height),
  };
}

function boxBackground(
  style: TooltipStyle,
  w: number,
  h: number,
): CSSProperties {
  const b = TOOLTIP.border;
  if (style.sprite) {
    return {
      position: "absolute",
      left: 0,
      top: 0,
      width: w,
      height: h,
      boxSizing: "border-box",
      borderStyle: "solid",
      borderWidth: b,
      borderColor: "transparent",
      borderImageSource: `url("${spriteUrl(TOOLTIP.sprite, style.sprite)}")`,
      borderImageSlice: `${b} fill`,
      borderImageRepeat: "stretch",
    };
  }

  return {
    position: "absolute",
    left: 4,
    top: 4,
    width: Math.max(0, w - 8),
    height: Math.max(0, h - 8),
    boxSizing: "border-box",
    border: "2px solid rgb(96, 96, 96)",
    background: "rgba(14, 14, 14, 0.99)",
  };
}

export function GameTooltipBox({
  layout,
  left,
  top,
  style,
  icons,
}: {
  layout: TooltipLayout;
  left: number;
  top: number;
  style: TooltipStyle;
  icons: Record<string, GuiSprite>;
}) {
  const b = TOOLTIP.border;
  const font = style.font;
  const nodes: ReactNode[] = [];
  layout.lines.forEach((line, row) => {
    const y = b + row * layout.lineHeight;
    line.items.forEach((item, at) => {
      const key = `${row}-${at}`;
      if (item.kind === "icon") {
        const sprite = icons[item.name];
        if (!sprite) return;
        const frames = Math.max(1, sprite.frames || 1);
        const frame = Math.min(frames, item.frame) - 1;
        nodes.push(
          <div
            key={`i${key}`}
            style={{
              position: "absolute",
              left: b + item.x,
              top: y + Math.round((layout.lineHeight - item.h) / 2),
              width: item.w,
              height: item.h,
              backgroundImage: `url("${spriteUrl(iconSpriteName(item.name), sprite)}")`,
              backgroundPosition: `${-frame * item.w}px 0`,
              backgroundSize: `${item.w * frames}px ${item.h}px`,
              backgroundRepeat: "no-repeat",
            }}
          />,
        );
        return;
      }
      if (font) {
        nodes.push(
          ...glyphRun(
            font,
            item.text,
            b + item.x,
            y,
            item.color ?? font.color,
            key,
            item.prev,
          ).nodes,
        );
        return;
      }
      nodes.push(
        <div
          key={`f${key}`}
          style={{
            position: "absolute",
            left: b + item.x,
            top: y,
            height: layout.lineHeight,
            lineHeight: `${layout.lineHeight}px`,
            fontSize: 14,
            fontFamily: "Georgia, 'Times New Roman', serif",
            whiteSpace: "pre",
            color: item.color ?? "rgb(255, 255, 255)",
          }}
        >
          {item.text}
        </div>,
      );
    });
  });
  return (
    <div
      style={{
        position: "absolute",
        left,
        top,
        width: layout.width,
        height: layout.height,
        pointerEvents: "none",
      }}
    >
      <div style={boxBackground(style, layout.width, layout.height)} />
      {nodes}
    </div>
  );
}

const ShownBox = memo(function ShownBox({ shown }: { shown: ShownTooltip }) {
  const style = useTooltipStyle();
  const markup = useMemo(
    () => tooltipMarkup(shown.content, shown.delayed),
    [shown.content, shown.delayed],
  );
  const iconNames = useMemo(
    () => (markup ? iconNamesOf(markup) : []),
    [markup],
  );
  const pictures = useTooltipIcons(iconNames);
  const layout = useMemo(() => {
    if (!markup) return null;
    const given = shown.content.icons ?? {};
    return layoutTooltip(
      markup,
      style.font ? bitmapMeasure(style.font) : FALLBACK_MEASURE,
      (name) => {
        const sprite = pictures[name];
        if (sprite) return frameSize(sprite);
        return given[name] ?? null;
      },
    );
  }, [markup, style.font, pictures, shown.content.icons]);
  if (!layout) return null;
  const at = placeTooltip(
    { w: layout.width, h: layout.height },
    shown.cursor,
    shown.bounds,
    shown.content.anchor,
    shown.anchorRect,
  );
  return (
    <GameTooltipBox
      layout={layout}
      left={at.x}
      top={at.y}
      style={style}
      icons={pictures}
    />
  );
});

const NONE: TooltipSnapshot = { hostId: null, shown: null };
const serverSnapshot = () => NONE;

export function GameTooltipHost({ fallback = false }: { fallback?: boolean }) {
  const id = useId();
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    attachTooltipHub();
    return tooltipHub.registerHost({
      id,
      explicit: !fallback,
      element: () => domElement(ref.current),
    });
  }, [id, fallback]);

  const snapshot = useSyncExternalStore(
    tooltipHub.subscribe,
    () => {
      const snap = tooltipHub.getSnapshot();
      return snap.hostId === id ? snap : NONE;
    },
    serverSnapshot,
  );
  return (
    <div
      ref={ref}
      aria-hidden
      style={{
        position: "fixed",
        inset: 0,
        overflow: "hidden",
        pointerEvents: "none",

        zIndex: fallback ? 2147483000 : STAGE_TOOLTIP_Z,
      }}
    >
      {snapshot.hostId === id && <StylePrefetch />}
      {snapshot.shown && <ShownBox shown={snapshot.shown} />}
    </div>
  );
}

export const STAGE_TOOLTIP_Z = 1000;

function StylePrefetch() {
  useTooltipStyle();
  return null;
}

let explicitSeq = 0;

export function gameTooltipProps(
  input: GameTooltipInput,
  options: { key?: string } = {},
): {
  onPointerEnter: (e: ReactPointerEvent<Element>) => void;
  onPointerMove: (e: ReactPointerEvent<Element>) => void;
  onPointerLeave: () => void;
} {
  const content = normalizeTooltip(input);
  const key =
    options.key ??
    (content
      ? `${content.text ?? ""}\u0000${content.delayed ?? ""}`
      : `none${explicitSeq++}`);
  const show = (e: ReactPointerEvent<Element>) => {
    attachTooltipHub();
    if (!content) return;
    const r = e.currentTarget.getBoundingClientRect();
    tooltipHub.showExplicit(key, content, e.clientX, e.clientY, {
      x: r.left,
      y: r.top,
      w: r.width,
      h: r.height,
    });
  };
  return {
    onPointerEnter: show,
    onPointerMove: show,
    onPointerLeave: () => tooltipHub.hideExplicit(key),
  };
}

export function showGameTooltip(
  key: string,
  input: GameTooltipInput,
  clientX: number,
  clientY: number,
): void {
  attachTooltipHub();
  const content = normalizeTooltip(input);
  if (content) tooltipHub.showExplicit(key, content, clientX, clientY, null);
  else tooltipHub.hideExplicit(key);
}

export function hideGameTooltip(key: string): void {
  tooltipHub.hideExplicit(key);
}

export function useTooltipSource(
  enabled: boolean,
  element: () => HTMLElement | null,
  hitTest: TooltipSource["hitTest"],
): void {
  const id = useId();
  const live = useRef({ element, hitTest });
  useEffect(() => {
    live.current = { element, hitTest };
  });
  useEffect(() => {
    if (!enabled) return;
    attachTooltipHub();
    return tooltipHub.registerSource({
      id,
      element: () => domElement(live.current.element()),
      hitTest: (x, y) => live.current.hitTest(x, y),
    });
  }, [enabled, id]);
}
