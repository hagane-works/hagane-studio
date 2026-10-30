"use client";

import type { GuiSprite } from "@/lib/guiRenderApi";

export type BarSprite = GuiSprite & { texture_urls?: string[] };

export function barFill(width: number, fraction: number): number {
  return Math.round(width * Math.min(1, Math.max(0, fraction)));
}

export function SpriteBar({
  x,
  y,
  size,
  sprite,
  fraction,
}: {
  x: number;
  y: number;
  size: { w: number; h: number };
  sprite?: BarSprite;
  fraction: number;
}) {
  const fill = sprite?.texture_urls?.[0];
  const track = sprite?.texture_urls?.[1];
  const width = barFill(size.w, fraction);
  const layer = (url: string, w: number) => (
    <div
      style={{
        position: "absolute",
        left: 0,
        top: 0,
        width: w,
        height: size.h,
        backgroundImage: `url("${url}")`,
        backgroundSize: `${size.w}px ${size.h}px`,
        backgroundRepeat: "no-repeat",
      }}
    />
  );
  return (
    <div
      style={{
        position: "absolute",
        left: x,
        top: y,
        width: size.w,
        height: size.h,
        pointerEvents: "none",
      }}
    >
      {track && layer(track, size.w)}
      {fill && width > 0 && layer(fill, width)}
    </div>
  );
}
