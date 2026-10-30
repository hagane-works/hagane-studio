import { useMemo, type ReactNode } from "react";
import type { GuiBitmapFont } from "@/lib/guiRenderApi";
import { glyphRun } from "./BitmapText";
import { frameSize, spriteUrl, useTooltipIcons } from "./GameTooltip";
import {
  bitmapMeasure,
  iconNamesOf,
  iconSpriteName,
  layoutTooltip,
} from "./tooltipText";

const NO_WRAP = 1 << 20;

export function RichText({
  font,
  markup,
  left,
  top,
  width,
  height,
  align,
  vcenter,
  maxLines,
}: {
  font: GuiBitmapFont;
  markup: string;
  left: number;
  top: number;

  width: number;

  height?: number;
  align: "left" | "center" | "right";
  vcenter: boolean;
  maxLines: number;
}) {
  const names = useMemo(() => iconNamesOf(markup), [markup]);
  const pictures = useTooltipIcons(names);
  const layout = useMemo(
    () =>
      layoutTooltip(
        markup,
        bitmapMeasure(font),
        (name) => (pictures[name] ? frameSize(pictures[name]) : null),
        width || NO_WRAP,
      ),
    [markup, font, pictures, width],
  );
  const lines = layout.lines.slice(0, Math.max(1, maxLines));
  const lineH = layout.lineHeight;
  const offsetY =
    vcenter && height ? Math.round((height - lines.length * lineH) / 2) : 0;

  const nodes: ReactNode[] = [];
  lines.forEach((line, row) => {
    const y = offsetY + row * lineH;
    const x0 =
      align === "center" && width
        ? Math.round((width - line.width) / 2)
        : align === "right" && width
          ? width - line.width
          : 0;
    line.items.forEach((item, at) => {
      const key = `${row}-${at}`;
      if (item.kind === "icon") {
        const sprite = pictures[item.name];
        if (!sprite) return;
        const frames = Math.max(1, sprite.frames || 1);
        const frame = Math.min(frames, item.frame) - 1;
        nodes.push(
          <div
            key={`i${key}`}

            data-sprite={iconSpriteName(item.name)}
            data-frame={frame + 1}
            style={{
              position: "absolute",
              left: x0 + item.x,
              top: y + Math.round((lineH - item.h) / 2),
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
      nodes.push(
        ...glyphRun(
          font,
          item.text,
          x0 + item.x,
          y,
          item.color ?? font.color,
          key,
          item.prev,
        ).nodes,
      );
    });
  });

  return (
    <div
      style={{
        position: "absolute",
        left,
        top,
        width: width || undefined,
        pointerEvents: "none",
        opacity: font.opacity,
      }}
    >
      {nodes}
    </div>
  );
}
