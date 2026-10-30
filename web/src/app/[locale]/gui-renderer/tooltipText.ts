import type { GuiBitmapFont, GuiNode, GuiText } from "@/lib/guiRenderApi";
import { textColors } from "@/lib/referenceTables";
import { glyphAdvance, isWideChar } from "./BitmapText";

export const TOOLTIP = {
  sprite: "ToolTip",

  font: "cg_16b",

  border: 18,

  maxWidth: 400,

  wrapWidth: 400 - 4 * 18,

  lineHeight: 16,

  cursorOffset: { x: 20, y: 23 },
  showDelayMs: 50,
  hideDelayMs: 50,
  delayedDelayMs: 1000,
  screenOffset: { left: 0, right: 0, top: 0, bottom: 0 },
} as const;

export interface TooltipAnchor {
  x: number;
  y: number;

  relative: boolean;
}

export interface TooltipIconSize {
  w: number;
  h: number;
}

export interface GameTooltipContent {
  text?: string | null;

  delayed?: string | null;

  icons?: Record<string, string>;

  anchor?: TooltipAnchor | null;
}

export type GameTooltipInput = string | GameTooltipContent | null | undefined;

export function normalizeTooltip(
  input: GameTooltipInput,
): GameTooltipContent | null {
  if (input == null) return null;
  if (typeof input === "string") return input ? { text: input } : null;
  return input;
}

const RUNTIME_SLOT = /\$[^$\n]*\$|\[[^\]\n]*\]/;
const COLOR_CODE = /§./g;

export function dropRuntimeLines(text: string): string {
  const kept: string[] = [];
  let carry = "";
  for (const line of text.split("\n")) {
    if (RUNTIME_SLOT.test(line)) {
      carry += (line.match(COLOR_CODE) ?? []).join("");
      continue;
    }
    kept.push(carry + line);
    carry = "";
  }
  return kept.join("\n");
}

function hasVisibleText(markup: string): boolean {
  return markup.replace(COLOR_CODE, "").trim() !== "";
}

function trimEnd(markup: string): string {
  const tail = markup.match(/(?:\s|§.)+$/u);
  if (!tail || tail.index === undefined) return markup;
  return (
    markup.slice(0, tail.index) + (tail[0].match(COLOR_CODE) ?? []).join("")
  );
}

export function tooltipMarkup(
  content: GameTooltipContent | null,
  withDelayed: boolean,
): string | null {
  if (!content) return null;
  const instant = content.text
    ? trimEnd(dropRuntimeLines(content.text.replace(/\r/g, "")))
    : "";
  const later =
    withDelayed && content.delayed
      ? trimEnd(dropRuntimeLines(content.delayed.replace(/\r/g, "")))
      : "";
  const shownInstant = hasVisibleText(instant) ? instant : "";
  const shownLater = hasVisibleText(later) ? later : "";
  if (!shownInstant && !shownLater) return null;
  if (!shownLater) return shownInstant;
  if (!shownInstant) return shownLater;
  return `${shownInstant}\n${shownLater}`;
}

export function hasTooltip(content: GameTooltipContent | null): boolean {
  return tooltipMarkup(content, true) !== null;
}

export type TooltipPiece =
  | { kind: "text"; text: string; color: string | null }
  | { kind: "icon"; name: string; frame: number; color: string | null };

const ICON_CHAR = /[A-Za-z0-9_]/;

export function parseTooltipMarkup(markup: string): TooltipPiece[][] {
  const paragraphs: TooltipPiece[][] = [[]];
  const stack: (string | null)[] = [null];
  let buffer = "";
  const color = () => stack[stack.length - 1];
  const flush = () => {
    if (!buffer) return;
    const line = paragraphs[paragraphs.length - 1];
    const last = line[line.length - 1];
    if (last && last.kind === "text" && last.color === color())
      last.text += buffer;
    else line.push({ kind: "text", text: buffer, color: color() });
    buffer = "";
  };
  const chars = [...markup];
  for (let i = 0; i < chars.length; i++) {
    const ch = chars[i];
    if (ch === "\r") continue;
    if (ch === "\n") {
      flush();
      paragraphs.push([]);
      continue;
    }
    if (ch === "§") {
      const code = chars[i + 1];
      if (code === undefined) break;
      i += 1;
      flush();
      if (code === "!") {
        if (stack.length > 1) stack.pop();
      } else {
        stack.push(textColors()[code] ?? color());
      }
      continue;
    }
    if (ch === "£") {
      let j = i + 1;
      let name = "";
      while (j < chars.length && ICON_CHAR.test(chars[j])) name += chars[j++];
      if (!name) {
        buffer += ch;
        continue;
      }
      let frame = 1;
      if (chars[j] === "|") {
        let digits = "";
        let k = j + 1;
        while (k < chars.length && /[0-9]/.test(chars[k])) digits += chars[k++];
        if (digits) {
          frame = Math.max(1, parseInt(digits, 10));
          j = k;
        }
      }

      if (chars[j] === "£") j += 1;
      else if (chars[j] === " ") j += 1;
      flush();
      paragraphs[paragraphs.length - 1].push({
        kind: "icon",
        name,
        frame,
        color: color(),
      });
      i = j - 1;
      continue;
    }
    buffer += ch;
  }
  flush();
  return paragraphs;
}

const SLOT = /\$([^$|\n]+)(?:\|([^$\n]*))?\$/g;

export function fillRuntimeSlots(
  markup: string,
  vars: Readonly<Record<string, number | string>>,
): string {
  let dropped = false;
  const filled = markup.replace(SLOT, (_match, name: string, format = "") => {
    const value = vars[name];
    if (value === undefined) {
      dropped = true;
      return "";
    }
    let text = String(value);
    if (typeof value === "number") {
      const digits = format.match(/[0-9]/)?.[0];
      const shown = format.includes("%") ? value * 100 : value;
      text =
        digits !== undefined ? shown.toFixed(Number(digits)) : String(shown);
      if (format.includes("+") && shown > 0) text = `+${text}`;
      if (format.includes("%")) text += "%";
    }

    const color = [...format].find(
      (ch) => /[A-Za-z]/.test(ch) && ch in textColors(),
    );
    return color ? `§${color}${text}§!` : text;
  });

  return dropped ? filled.replace(/ ?\((?:\s|§.)*\)/g, "") : filled;
}

export function iconNamesOf(markup: string): string[] {
  const names = new Set<string>();
  for (const paragraph of parseTooltipMarkup(markup)) {
    for (const piece of paragraph)
      if (piece.kind === "icon") names.add(piece.name);
  }
  return [...names];
}

export function iconSpriteName(name: string): string {
  return name.startsWith("GFX_") ? name : `GFX_${name}`;
}

export function shortcutMarkup(shortcut: string): {
  text: string;
  icons: Record<string, string>;
} {
  const keys = shortcut
    .split("+")
    .map((key) => key.trim())
    .filter(Boolean);
  const icons: Record<string, string> = {};
  for (const key of keys) icons[key] = key.toUpperCase();
  return { text: keys.map((key) => `£${key}£`).join("+"), icons };
}

export interface TooltipMeasure {
  line: number;
  advance(prev: number, code: number): number;
}

export function bitmapMeasure(font: GuiBitmapFont): TooltipMeasure {
  return {
    line: font.line || TOOLTIP.lineHeight,
    advance: (prev, code) => glyphAdvance(font, prev, code),
  };
}

export const FALLBACK_MEASURE: TooltipMeasure = {
  line: TOOLTIP.lineHeight,
  advance: (_prev, code) => (code === 32 ? 4 : 7),
};

export type TooltipIconResolver = (
  name: string,
) => TooltipIconSize | string | null;

export type TooltipItem =
  | {
      kind: "text";

      x: number;
      text: string;
      color: string | null;

      prev: number;
    }
  | {
      kind: "icon";
      x: number;
      name: string;
      frame: number;
      w: number;
      h: number;
    };

export interface TooltipLine {
  width: number;
  items: TooltipItem[];
}

export interface TooltipLayout {
  lines: TooltipLine[];

  width: number;
  height: number;
  lineHeight: number;
}

type Atom =
  | { kind: "char"; ch: string; code: number; color: string | null }
  | { kind: "icon"; name: string; frame: number; w: number; h: number };

function atomsOf(
  paragraph: TooltipPiece[],
  icons: TooltipIconResolver,
): Atom[] {
  const atoms: Atom[] = [];
  const pushText = (text: string, color: string | null) => {
    for (const ch of text)
      atoms.push({ kind: "char", ch, code: ch.codePointAt(0) ?? 32, color });
  };
  for (const piece of paragraph) {
    if (piece.kind === "text") {
      pushText(piece.text, piece.color);
      continue;
    }
    const how = icons(piece.name);
    if (typeof how === "string") pushText(how, piece.color);
    else if (how)
      atoms.push({
        kind: "icon",
        name: piece.name,
        frame: piece.frame,
        w: how.w,
        h: how.h,
      });
  }
  return atoms;
}

function wordsOf(atoms: Atom[]): Atom[][] {
  const words: Atom[][] = [];
  let word: Atom[] = [];
  for (const atom of atoms) {
    const wide = atom.kind === "char" && isWideChar(atom.code);
    if (wide && word.length) {
      words.push(word);
      word = [];
    }
    word.push(atom);
    if ((atom.kind === "char" && atom.ch === " ") || wide) {
      words.push(word);
      word = [];
    }
  }
  if (word.length) words.push(word);
  return words;
}

function withoutTrailingSpaces(atoms: Atom[]): Atom[] {
  let end = atoms.length;
  while (end > 0) {
    const atom = atoms[end - 1];
    if (atom.kind !== "char" || atom.ch !== " ") break;
    end -= 1;
  }
  return atoms.slice(0, end);
}

function lastCode(atoms: Atom[]): number {
  const last = atoms[atoms.length - 1];
  return last && last.kind === "char" ? last.code : -1;
}

function widthOf(atoms: Atom[], measure: TooltipMeasure, prev: number): number {
  let width = 0;
  for (const atom of atoms) {
    if (atom.kind === "icon") {
      width += atom.w;
      prev = -1;
    } else {
      width += measure.advance(prev, atom.code);
      prev = atom.code;
    }
  }
  return width;
}

function wrapParagraph(
  atoms: Atom[],
  measure: TooltipMeasure,
  limit: number,
): Atom[][] {
  const lines: Atom[][] = [];
  let line: Atom[] = [];
  let lineW = 0;
  const breakLine = () => {
    lines.push(line);
    line = [];
    lineW = 0;
  };
  for (const word of wordsOf(atoms)) {
    let w = widthOf(word, measure, lastCode(line));
    if (line.length && lineW + w > limit) {
      breakLine();
      w = widthOf(word, measure, -1);
    }
    if (
      !line.length &&
      word.length > 1 &&
      widthOf(withoutTrailingSpaces(word), measure, -1) > limit
    ) {
      for (const atom of word) {
        let aw = widthOf([atom], measure, lastCode(line));
        if (line.length && lineW + aw > limit) {
          breakLine();
          aw = widthOf([atom], measure, -1);
        }
        line.push(atom);
        lineW += aw;
      }
      continue;
    }
    line.push(...word);
    lineW += w;
  }
  lines.push(line);
  return lines;
}

function lineOf(atoms: Atom[], measure: TooltipMeasure): TooltipLine {
  const items: TooltipItem[] = [];
  let pen = 0;
  let prev = -1;
  for (const atom of atoms) {
    if (atom.kind === "icon") {
      items.push({
        kind: "icon",
        x: pen,
        name: atom.name,
        frame: atom.frame,
        w: atom.w,
        h: atom.h,
      });
      pen += atom.w;
      prev = -1;
      continue;
    }
    const last = items[items.length - 1];
    if (last && last.kind === "text" && last.color === atom.color)
      last.text += atom.ch;
    else
      items.push({
        kind: "text",
        x: pen,
        text: atom.ch,
        color: atom.color,
        prev,
      });
    pen += measure.advance(prev, atom.code);
    prev = atom.code;
  }
  return { width: pen, items };
}

export function layoutTooltip(
  markup: string,
  measure: TooltipMeasure,
  icons: TooltipIconResolver = () => null,
  wrapWidth: number = TOOLTIP.wrapWidth,
): TooltipLayout {
  const lines: TooltipLine[] = [];
  for (const paragraph of parseTooltipMarkup(markup)) {
    for (const atoms of wrapParagraph(
      atomsOf(paragraph, icons),
      measure,
      wrapWidth,
    )) {
      lines.push(lineOf(atoms, measure));
    }
  }
  return {
    lines,
    ...tooltipBoxSize(
      lines.map((line) => line.width),
      measure.line,
    ),
    lineHeight: measure.line,
  };
}

export function tooltipBoxSize(
  lineWidths: number[],
  lineHeight: number = TOOLTIP.lineHeight,
): { width: number; height: number } {
  const widest = lineWidths.reduce((a, b) => Math.max(a, b), 0);
  return {
    width: widest + 2 * TOOLTIP.border,
    height: lineWidths.length * lineHeight + 2 * TOOLTIP.border,
  };
}

export interface TooltipRect {
  x: number;
  y: number;
  w: number;
  h: number;
}

export function placeTooltip(
  size: { w: number; h: number },
  cursor: { x: number; y: number },
  screen: { w: number; h: number },
  anchor?: TooltipAnchor | null,
  element?: TooltipRect | null,
): { x: number; y: number } {
  const off = TOOLTIP.screenOffset;
  let x: number;
  let y: number;
  if (anchor) {
    x = anchor.relative && element ? element.x + anchor.x : anchor.x;
    y = anchor.relative && element ? element.y + anchor.y : anchor.y;
  } else {
    x = cursor.x + TOOLTIP.cursorOffset.x;
    y = cursor.y + TOOLTIP.cursorOffset.y;
    if (x + size.w > screen.w - off.right)
      x = cursor.x - TOOLTIP.cursorOffset.x - size.w;
    if (y + size.h > screen.h - off.bottom) y = cursor.y - size.h;
  }
  x = Math.max(off.left, Math.min(x, screen.w - off.right - size.w));
  y = Math.max(off.top, Math.min(y, screen.h - off.bottom - size.h));
  return { x: Math.round(x), y: Math.round(y) };
}

function numberOf(value: unknown): number | null {
  const n =
    typeof value === "number"
      ? value
      : typeof value === "string"
        ? parseFloat(value)
        : NaN;
  return Number.isFinite(n) ? n : null;
}

function truthy(value: unknown): boolean {
  if (typeof value === "boolean") return value;
  if (typeof value === "number") return value !== 0;
  return typeof value === "string" && /^(yes|true|1)$/i.test(value.trim());
}

export function tooltipAnchorOf(raw: unknown): TooltipAnchor | null {
  if (!raw || typeof raw !== "object") return null;
  const entries: Record<string, unknown> = {};
  const block = (raw as { __block__?: unknown }).__block__;
  if (Array.isArray(block)) {
    for (const item of block) {
      if (
        item &&
        typeof item === "object" &&
        typeof (item as { k?: unknown }).k === "string"
      ) {
        entries[(item as { k: string }).k.toLowerCase()] = (
          item as { v?: unknown }
        ).v;
      }
    }
  } else {
    for (const [key, value] of Object.entries(raw))
      entries[key.toLowerCase()] = value;
  }
  const x = numberOf(entries.x);
  const y = numberOf(entries.y);
  if (x === null || y === null) return null;
  return {
    x,
    y,
    relative: entries.relative === undefined ? true : truthy(entries.relative),
  };
}

function textOf(texts: Record<string, GuiText>, raw: unknown): string {
  const key = typeof raw === "string" ? raw.trim().replace(/^"|"$/g, "") : "";
  if (!key) return "";
  const entry = texts[key];

  if (!entry || entry.kind === "key") return "";
  return entry.display || "";
}

export function nodeTooltip(
  node: GuiNode,
  texts: Record<string, GuiText>,
): GameTooltipContent | null {
  const raw = node.raw ?? {};
  const text = textOf(texts, raw.pdx_tooltip);
  const delayed = textOf(texts, raw.pdx_tooltip_delayed);
  if (!text && !delayed) return null;
  return { text, delayed, anchor: tooltipAnchorOf(raw.pdx_tooltip_anchor) };
}
