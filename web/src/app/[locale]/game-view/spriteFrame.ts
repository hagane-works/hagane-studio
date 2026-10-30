import type { CSSProperties } from "react";
import { guiRenderApi, type GuiNode, type GuiSprite } from "@/lib/guiRenderApi";
import { anchor } from "../gui-renderer/GuiCanvas";

export function spriteUrlOf(
  sprite: GuiSprite,
  name?: string | null,
): string | undefined {
  if (sprite.vanilla_url) return sprite.vanilla_url;
  if (!name || sprite.missing_file || sprite.resolved === false)
    return undefined;
  return guiRenderApi.spriteUrl(name, sprite.version);
}

export const GAME_CURSOR_ROUTE = "/api/go/gui-vanilla/gfx/cursors/normal.png";

export function gameCursor(
  sprites: Readonly<Record<string, GuiSprite>>,
): string | undefined {
  for (const sprite of Object.values(sprites)) {
    const url = sprite.vanilla_url;
    const at = url ? url.indexOf("/gfx/") : -1;
    if (url && at >= 0) {
      const pointer = url.includes("?")
        ? GAME_CURSOR_ROUTE
        : `${url.slice(0, at)}/gfx/cursors/normal.png`;
      return `url("${pointer}") 0 0, auto`;
    }
  }
  return undefined;
}

export interface PickableArt {
  "data-sprite"?: string;
  "data-frame"?: number;
  "data-title"?: string;
}

export function pickable(
  name: string | null | undefined,
  frame = 1,
  title?: string,
): PickableArt {
  if (!name) return {};
  return title
    ? { "data-sprite": name, "data-frame": frame, "data-title": title }
    : { "data-sprite": name, "data-frame": frame };
}

export interface FrameOptions {
  frame?: number;

  scale?: number;

  name?: string | null;
}

export function frameStyle(
  sprite: GuiSprite,
  left: number,
  top: number,
  { frame = 1, scale = 1, name }: FrameOptions = {},
): CSSProperties {
  const frames = Math.max(1, sprite.frames || 1);
  const w = Math.max(1, Math.round((sprite.width / frames) * scale));
  const h = Math.max(1, Math.round(sprite.height * scale));
  const at = Math.min(frames, Math.max(1, Math.round(frame))) - 1;
  const url = spriteUrlOf(sprite, name);
  return {
    position: "absolute",
    left,
    top,
    width: w,
    height: h,
    backgroundImage: url ? `url("${url}")` : undefined,
    backgroundPosition: `${-at * w}px 0`,
    backgroundRepeat: "no-repeat",
    backgroundSize: `${w * frames}px ${h}px`,
    pointerEvents: "none",
  };
}

export interface ExpandButton {
  sprite: string;

  x: number;
  y: number;
  scale: number;
}

type RawEntry = { k?: unknown; v?: unknown };

function rawField(block: unknown, key: string): unknown {
  if (!block || typeof block !== "object") return undefined;
  const want = key.toLowerCase();
  const entries = (block as { __block__?: unknown }).__block__;
  if (Array.isArray(entries)) {
    const hit = (entries as RawEntry[]).find(
      (entry) => String(entry?.k ?? "").toLowerCase() === want,
    );
    return hit?.v;
  }
  const hit = Object.entries(block as Record<string, unknown>).find(
    ([k]) => k.toLowerCase() === want,
  );
  return hit?.[1];
}

export function expandButtonOf(
  box: GuiNode | undefined,
): ExpandButton | undefined {
  if (!box) return undefined;

  if ((box.children ?? []).some((child) => child.type === "expandbutton"))
    return undefined;
  const raw = box.raw?.expandbutton;
  const sprite =
    rawField(raw, "quadTextureSprite") ?? rawField(raw, "spriteType");
  if (typeof sprite !== "string" || !sprite) return undefined;
  const position = rawField(raw, "position");
  const px = Number(rawField(position, "x") ?? 0) || 0;
  const py = Number(rawField(position, "y") ?? 0) || 0;
  const orientation = rawField(raw, "orientation");
  const w = Number(box.size?.width ?? box.size?.x ?? 0) || 0;
  const h = Number(box.size?.height ?? box.size?.y ?? 0) || 0;
  const [ax, ay] = anchor(
    typeof orientation === "string" ? orientation : null,
    0,
    0,
    w,
    h,
  );
  return {
    sprite,
    x: ax + px,
    y: ay + py,
    scale: Number(rawField(raw, "scale") ?? 1) || 1,
  };
}
