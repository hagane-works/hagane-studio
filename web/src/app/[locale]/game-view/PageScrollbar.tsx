"use client";

import { useRef } from "react";

const BAR_W = 8;

export const GAME_SCROLLBAR = {
  scrollbarWidth: "thin",
  scrollbarColor: "rgba(170, 146, 98, 0.85) rgba(0, 0, 0, 0.35)",
} as const;

export function thumbPlace(
  track: number,
  view: number,
  limit: number,
  scroll: number,
): { size: number; offset: number } {
  if (limit <= 0 || track <= 0) return { size: track, offset: 0 };
  const size = Math.max(24, Math.round((track * view) / (view + limit)));
  const offset = Math.round(
    ((track - size) * Math.min(limit, Math.max(0, scroll))) / limit,
  );
  return { size, offset };
}

export function PageScrollbar({
  box,
  scroll,
  limit,
  onScroll,
}: {
  box: { x: number; y: number; w: number; h: number };
  scroll: number;
  limit: number;
  onScroll: (y: number) => void;
}) {
  const drag = useRef<{ y: number; from: number; scale: number } | null>(null);
  if (limit <= 0) return null;
  const thumb = thumbPlace(box.h, box.h, limit, scroll);
  return (
    <div
      data-page-scrollbar

      aria-hidden
      style={{
        position: "absolute",
        left: box.x + box.w - BAR_W - 2,
        top: box.y,
        width: BAR_W,
        height: box.h,
        borderRadius: BAR_W / 2,
        background: "rgba(0, 0, 0, 0.35)",
        pointerEvents: "auto",
      }}
      onPointerDown={(e) => {
        const rect = e.currentTarget.getBoundingClientRect();
        const scale = rect.height / Math.max(1, box.h);
        const at = (e.clientY - rect.top) / scale;
        if (at < thumb.offset || at > thumb.offset + thumb.size) {
          onScroll(
            Math.round(
              ((at - thumb.size / 2) / Math.max(1, box.h - thumb.size)) * limit,
            ),
          );
        }
      }}
    >
      <div
        style={{
          position: "absolute",
          left: 0,
          top: thumb.offset,
          width: BAR_W,
          height: thumb.size,
          borderRadius: BAR_W / 2,
          background: "rgba(170, 146, 98, 0.85)",
          cursor: "grab",
        }}
        onPointerDown={(e) => {
          e.stopPropagation();
          const track = e.currentTarget.parentElement?.getBoundingClientRect();
          drag.current = {
            y: e.clientY,
            from: scroll,
            scale: track ? track.height / Math.max(1, box.h) : 1,
          };
          e.currentTarget.setPointerCapture(e.pointerId);
        }}
        onPointerMove={(e) => {
          const at = drag.current;
          if (!at) return;
          const moved = (e.clientY - at.y) / at.scale;
          onScroll(at.from + (moved * limit) / Math.max(1, box.h - thumb.size));
        }}
        onPointerUp={() => {
          drag.current = null;
        }}
      />
    </div>
  );
}
