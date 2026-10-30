"use client";

import type { GuiSprite } from "@/lib/guiRenderApi";
import { pickable } from "./spriteFrame";

type BarSprite = GuiSprite & {
  size?: { x: number; y: number };
  texture_urls?: string[];
};

export function LoadingBar({
  sprite,
  rect,
  progress,
}: {
  sprite: GuiSprite | undefined;
  rect: { x: number; y: number; w: number; h: number };

  progress: number;
}) {
  const bar = sprite as BarSprite | undefined;
  if (!bar) return null;
  const w = bar.size?.x || rect.w || 934;
  const h = bar.size?.y || rect.h || 35;
  const fill = bar.texture_urls?.[0] || bar.vanilla_url;
  const track = bar.texture_urls?.[1];
  const filled = Math.round(w * Math.min(1, Math.max(0, progress)));
  const art = (url: string, width: number) => ({
    position: "absolute" as const,
    left: 0,
    top: 0,
    width,
    height: h,
    backgroundImage: `url("${url}")`,
    backgroundSize: `${w}px ${h}px`,
    backgroundRepeat: "no-repeat",
  });
  return (
    <div
      style={{
        position: "absolute",
        left: rect.x,
        top: rect.y,
        width: w,
        height: h,
        pointerEvents: "none",
      }}
    >
      {track && <div style={art(track, w)} />}
      {fill && filled > 0 && <div style={art(fill, filled)} />}
      <div
        {...pickable("GFX_loadingstatus_progress", 1, "progressbar")}
        style={{ position: "absolute", inset: 0 }}
      />
    </div>
  );
}
