import type { CSSProperties, ReactNode } from "react";
import type { GuiBitmapFont } from "@/lib/guiRenderApi";

type Glyph = [number, number, number, number, number, number, number];

function glyphOf(font: GuiBitmapFont, code: number): Glyph | undefined {
  return font.chars[String(code)] as Glyph | undefined;
}

export const FALLBACK_FONT_FAMILY =
  '"Noto Sans CJK SC", "Microsoft YaHei", "PingFang SC", "Yu Gothic", "Meiryo", "Malgun Gothic", "Apple SD Gothic Neo", "Segoe UI", sans-serif';

export function fallbackFontSize(font: GuiBitmapFont): number {
  return Math.max(8, Math.round((font.line || 16) * 0.76));
}

export function isWideChar(code: number): boolean {
  return (
    (code >= 0x1100 && code <= 0x115f) ||
    (code >= 0x2e80 && code <= 0xa4cf) ||
    (code >= 0xac00 && code <= 0xd7a3) ||
    (code >= 0xf900 && code <= 0xfaff) ||
    (code >= 0xfe30 && code <= 0xfe4f) ||
    (code >= 0xff00 && code <= 0xff60) ||
    (code >= 0xffe0 && code <= 0xffe6) ||
    (code >= 0x20000 && code <= 0x3fffd)
  );
}

let measureContext: CanvasRenderingContext2D | null | undefined;
const measuredAdvances = new Map<string, number>();

function measuringContext(): CanvasRenderingContext2D | null {
  if (measureContext !== undefined) return measureContext;
  measureContext = null;
  if (
    typeof document !== "undefined" &&
    typeof navigator !== "undefined" &&
    !/jsdom/i.test(navigator.userAgent)
  ) {
    try {
      measureContext = document.createElement("canvas").getContext("2d");
    } catch {
      measureContext = null;
    }
  }
  return measureContext;
}

function webFontSpace(
  font: GuiBitmapFont,
  prev: number,
  code: number,
): boolean {
  return code === 32 && prev >= 0 && prev !== 32 && !glyphOf(font, prev);
}

function fallbackAdvance(font: GuiBitmapFont, code: number): number {
  const size = fallbackFontSize(font);
  if (isWideChar(code)) return size;
  const key = `${size}:${code}`;
  let advance = measuredAdvances.get(key);
  if (advance === undefined) {
    const context = measuringContext();
    if (context) {
      context.font = `${size}px ${FALLBACK_FONT_FAMILY}`;
      advance = context.measureText(String.fromCodePoint(code)).width;
    } else {
      advance = Math.round(size * (code === 32 ? 0.3 : 0.6));
    }
    measuredAdvances.set(key, advance);
  }
  return advance;
}

const FALLBACK_OUTLINE = "-1px 0 #000, 1px 0 #000, 0 -1px #000, 0 1px #000";

function fallbackShadow(colour: string): string | undefined {
  const hex = colour.replace("#", "");
  if (hex.length !== 6) return FALLBACK_OUTLINE;
  const r = parseInt(hex.slice(0, 2), 16);
  const g = parseInt(hex.slice(2, 4), 16);
  const b = parseInt(hex.slice(4, 6), 16);
  return 0.299 * r + 0.587 * g + 0.114 * b > 110 ? FALLBACK_OUTLINE : undefined;
}

function kern(font: GuiBitmapFont, prev: number, code: number): number {
  return font.kerning?.[`${prev},${code}`] ?? 0;
}

export function glyphAdvance(
  font: GuiBitmapFont,
  prev: number,
  code: number,
): number {
  const glyph = webFontSpace(font, prev, code)
    ? undefined
    : glyphOf(font, code);
  if (!glyph) return fallbackAdvance(font, code);
  return (prev >= 0 ? kern(font, prev, code) : 0) + glyph[6];
}

export function inkWidth(font: GuiBitmapFont, line: string): number {
  let pen = 0;
  let right = 0;
  let prev = -1;
  for (const ch of line) {
    const code = ch.codePointAt(0) ?? 32;
    const glyph = webFontSpace(font, prev, code)
      ? undefined
      : glyphOf(font, code);
    if (!glyph) {
      pen += fallbackAdvance(font, code);
      right = Math.max(right, pen);
      prev = code;
      continue;
    }
    pen += prev >= 0 ? kern(font, prev, code) : 0;
    right = Math.max(right, pen + glyph[4] + glyph[2]);
    pen += glyph[6];
    prev = code;
  }
  return Math.max(right, pen);
}

export function measureLine(font: GuiBitmapFont, line: string): number {
  let width = 0;
  let prev = -1;
  for (const ch of line) {
    const code = ch.codePointAt(0) ?? 32;
    width += glyphAdvance(font, prev, code);
    prev = code;
  }
  return width;
}

export function glyphRun(
  font: GuiBitmapFont,
  text: string,
  left: number,
  top: number,
  tint: string | undefined,
  keyPrefix: string,
  prev = -1,
): { nodes: ReactNode[]; pen: number; prev: number } {
  const nodes: ReactNode[] = [];
  const atlasSize = `${font.atlas_w}px ${font.atlas_h}px`;
  let pen = left;

  let run: { left: number; text: string } | null = null;
  const flush = () => {
    if (!run) return;
    const colour = tint ?? font.color ?? "#ffffff";
    nodes.push(
      <span
        key={`${keyPrefix}-f${nodes.length}`}
        style={{
          position: "absolute",
          left: run.left,
          top,
          height: font.line,
          lineHeight: `${font.line}px`,
          fontSize: fallbackFontSize(font),
          fontFamily: FALLBACK_FONT_FAMILY,
          color: colour,
          textShadow: fallbackShadow(colour),
          whiteSpace: "pre",
        }}
      >
        {run.text}
      </span>,
    );
    run = null;
  };
  for (const ch of text) {
    const code = ch.codePointAt(0) ?? 32;

    const glyph = run && code === 32 ? undefined : glyphOf(font, code);
    if (!glyph) {
      if (!run) run = { left: pen, text: "" };
      run.text += ch;
      pen += fallbackAdvance(font, code);
      prev = code;
      continue;
    }
    flush();
    if (prev >= 0) pen += kern(font, prev, code);
    prev = code;
    const [gx, gy, gw, gh, xo, yo, xa] = glyph;
    if (gw > 0 && gh > 0) {
      const box: CSSProperties = {
        position: "absolute",
        left: pen + xo,
        top: top + yo,
        width: gw,
        height: gh,
      };
      const key = `${keyPrefix}-${nodes.length}`;
      const position = `-${gx}px -${gy}px`;

      if (!tint || font.tint === "luminance") {
        nodes.push(
          <div
            key={`a${key}`}
            style={{
              ...box,
              backgroundImage: `url("${font.atlas}")`,
              backgroundPosition: position,
              backgroundSize: atlasSize,
            }}
          />,
        );
      }
      if (tint) {
        const mask = `url("${font.atlas}")`;
        nodes.push(
          <div
            key={`c${key}`}
            style={{
              ...box,
              backgroundColor: tint,
              maskImage: mask,
              WebkitMaskImage: mask,
              maskPosition: position,
              WebkitMaskPosition: position,
              maskSize: atlasSize,
              WebkitMaskSize: atlasSize,
              maskRepeat: "no-repeat",
              WebkitMaskRepeat: "no-repeat",
              maskMode: font.tint,
            }}
          />,
        );
      }
    }
    pen += xa;
  }
  flush();
  return { nodes, pen, prev };
}

export function wrapLines(
  font: GuiBitmapFont,
  text: string,
  maxWidth: number,
): string[] {
  const out: string[] = [];

  const safeText = typeof text === "string" ? text : "";
  for (const paragraph of safeText.split(/\r?\n/)) {
    if (!maxWidth) {
      out.push(paragraph);
      continue;
    }
    let current = "";
    for (const word of paragraph.split(" ")) {
      breakPieces(word).forEach((piece, at) => {
        const candidate =
          at > 0
            ? `${current}${piece}`
            : current
              ? `${current} ${piece}`
              : piece;
        if (current && measureLine(font, candidate) > maxWidth) {
          out.push(current);
          current = piece;
        } else {
          current = candidate;
        }
      });
    }
    out.push(current);
  }
  return out;
}

function breakPieces(word: string): string[] {
  const pieces: string[] = [];
  let narrow = "";
  for (const ch of word) {
    if (isWideChar(ch.codePointAt(0) ?? 32)) {
      if (narrow) pieces.push(narrow);
      narrow = "";
      pieces.push(ch);
    } else {
      narrow += ch;
    }
  }
  if (narrow || !pieces.length) pieces.push(narrow);
  return pieces;
}

export function fitLines(
  font: GuiBitmapFont,
  text: string,
  width: number,
  maxLines: number,
  ellipsis = false,
): string[] {
  const all = wrapLines(font, text, width);
  const cap = Math.max(1, maxLines);
  if (!ellipsis || !width || all.length <= cap) return all.slice(0, cap);
  let rest = all.slice(cap - 1).join(" ");
  while (rest && inkWidth(font, `${rest}...`) > width) rest = rest.slice(0, -1);
  return [...all.slice(0, cap - 1), `${rest.trimEnd()}...`];
}

export function BitmapText({
  font,
  text,
  left,
  top,
  width,
  height,
  align,
  vcenter,
  maxLines,
  color,
  ellipsis,
}: {
  font: GuiBitmapFont;
  text: string;
  left: number;
  top: number;

  width: number;

  height?: number;
  align: "left" | "center" | "right";
  vcenter: boolean;
  maxLines: number;

  color?: string;

  ellipsis?: boolean;
}) {
  const lines = fitLines(font, text, width, maxLines, ellipsis);
  const tint = color ?? font.color;
  const blockH = lines.length * font.line;
  const offsetY = vcenter && height ? Math.round((height - blockH) / 2) : 0;

  const glyphs: ReactNode[] = [];
  lines.forEach((line, row) => {
    const lineW = measureLine(font, line);
    const pen =
      align === "center" && width
        ? Math.round((width - lineW) / 2)
        : align === "right" && width
          ? width - lineW
          : 0;
    glyphs.push(
      ...glyphRun(font, line, pen, offsetY + row * font.line, tint, String(row))
        .nodes,
    );
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
      {glyphs}
    </div>
  );
}
