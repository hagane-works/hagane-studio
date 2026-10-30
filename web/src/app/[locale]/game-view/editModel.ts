import type {
  GuiNode,
  GuiSprite,
  GuiWindow,
  TextKind,
} from "@/lib/guiRenderApi";
import { flattenTree, isPlayerHidden } from "../gui-renderer/GuiCanvas";
import { buildNamePathMap } from "../gui-renderer/guiOverlay";
import { spriteUrlOf } from "./spriteFrame";

export type TextScope = "everywhere" | "allCountries" | "country";

export type TextSource =
  | { kind: "key"; key: string; scope: TextScope }
  | { kind: "person"; token: string };

export interface PrintedText {
  text: string;
  source?: TextSource;
}

export interface EditSelection {
  owner: string;
  win: GuiWindow;
  id: string;
  node: GuiNode;

  shown?: string;

  source?: TextSource;

  stack?: StackEntry[];
}

export interface StackEntry {
  id: string;
  node: GuiNode;

  size?: { width: number; height: number };
  shown?: string;
  source?: TextSource;
}

export type ElementKind = "button" | "text" | "icon" | "panel" | "background";

export function elementKind(node: GuiNode): ElementKind | null {
  const t = node.type.toLowerCase();
  if (t.endsWith("buttontype") || t === "checkboxtype") return "button";
  if (t.endsWith("textboxtype")) return "text";
  if (t === "icontype") return "icon";
  if (t.includes("window")) return "panel";
  if (t === "background") return "background";
  return null;
}

function isBackdrop(node: GuiNode): boolean {
  return node.type.toLowerCase() === "background";
}

export type TextLocked =
  "runtimeCountry" | "computed" | "placeholder" | "literal" | "runtime";

export type TextEdit =
  | {
      editable: true;
      via: "key";
      key: string;
      current: string;
      rewritten: boolean;
      scope: TextScope;
      runtime: boolean;
    }
  | {
      editable: true;
      via: "gui";
      target: string;
      file: string;
      originalText: string;
      originalKind: TextKind | null;
      current: string;
      rewritten: boolean;
    }
  | { editable: true; via: "person"; token: string; current: string }
  | { editable: false; reason: TextLocked; current: string };

export interface EditTarget {
  title: string;
  windowName: string;

  sprite: string | null;

  spriteSize: { width: number; height: number; frames?: number } | null;

  preview: SpritePreview | null;

  ownPicture: OwnPicture | null;

  removal: { file: string; target: string; hidden: boolean } | null;

  flag: boolean;
  text: TextEdit | null;

  stack: {
    id: string;
    kind: ElementKind | null;
    type: string;
    name: string;
    backdrop: boolean;
    picked: boolean;
  }[];
}

export interface SpritePreview {
  url: string;

  frame: number;
  frames: number;
  frameWidth: number;
  frameHeight: number;
}

export interface OwnPicture {
  file: string;
  target: string;
  gfx: string;
  width: number;
  height: number;
  frames: number;

  replaced: boolean;
}

const ENGINE_SWAPPED_WINDOWS: ReadonlySet<string> = new Set([
  "frontend_background",
  "load_screen",
]);

function numericSize(
  size: GuiNode["size"] | undefined,
): { width: number; height: number } | null {
  const width = Number(size?.width);
  const height = Number(size?.height);
  return Number.isFinite(width) &&
    Number.isFinite(height) &&
    width > 0 &&
    height > 0
    ? { width: Math.round(width), height: Math.round(height) }
    : null;
}

export function ownPictureName(file: string, target: string): string {
  const key = `${file}\0${target}`;
  let a = 0x811c9dc5;
  let b = 0x01000193;
  for (let i = 0; i < key.length; i++) {
    const c = key.charCodeAt(i);
    a = Math.imul(a ^ c, 0x01000193);
    b = Math.imul(b ^ c, 0x5bd1e995) ^ (b >>> 15);
  }
  const hash = (
    (a >>> 0).toString(16).padStart(8, "0") +
    (b >>> 0).toString(16).padStart(8, "0")
  ).slice(0, 12);
  const last = target.split("/").pop() ?? "";
  const slug = last
    .replace(/[^A-Za-z0-9_]/g, "_")
    .replace(/_+/g, "_")
    .replace(/^_|_$/g, "")
    .slice(0, 40);
  return `GFX_gv_${slug ? `${slug}_` : ""}${hash}`;
}

export function ownPictureOf(
  selection: EditSelection,
  sprite: string | null,
): OwnPicture | null {
  const { win, node, id } = selection;
  if (!sprite || !win.file || !(win.scaffold_hash || win.current_scaffold_hash))
    return null;

  const backdrop =
    node.type.toLowerCase() === "background" && id.endsWith("/bg");
  const elementId = backdrop ? id.slice(0, -3) : id;
  const target = buildNamePathMap(win.node)[elementId];

  if (!target || target.split("/").some((segment) => /#\d+$/.test(segment)))
    return null;
  if (ENGINE_SWAPPED_WINDOWS.has(target.split("/")[0])) return null;
  const meta = win.sprites[sprite];
  const drawn = selection.stack?.find((entry) => entry.id === id)?.size;
  const own = ownPictureName(win.file, target);
  const replaced = sprite === own;
  let size: { width: number; height: number } | null;
  let frames = 1;
  if (backdrop || node.background?.sprite === sprite) {
    const window = backdrop
      ? flattenTree(win.node).find((row) => row.id === elementId)?.node
      : node;

    size = (drawn ? numericSize(drawn) : null) ?? numericSize(window?.size);

    frames = Math.max(1, meta?.frames || 1);
  } else if (meta?.cornered) {
    size = numericSize(node.size) ?? (drawn ? numericSize(drawn) : null);
  } else {
    frames = Math.max(1, meta?.frames || 1);
    size =
      meta?.resolved && meta.width && meta.height
        ? { width: Math.round(meta.width / frames), height: meta.height }
        : null;
  }
  if (!size && meta?.resolved && meta.width && meta.height) {
    frames = Math.max(1, meta.frames || 1);
    size = { width: Math.round(meta.width / frames), height: meta.height };
  }
  if (!size) return null;
  return {
    file: win.file,
    target,
    gfx: own,
    width: size.width * frames,
    height: size.height,
    frames,
    replaced,
  };
}

export function removalOf(
  selection: EditSelection,
): { file: string; target: string; hidden: boolean } | null {
  const { win, node, id } = selection;
  if (
    !win.file ||
    !(win.scaffold_hash || win.current_scaffold_hash) ||
    node.type.toLowerCase() === "background"
  )
    return null;
  const target = buildNamePathMap(win.node)[id];
  if (
    !target ||
    !target.includes("/") ||
    target.split("/").some((segment) => /#\d+$/.test(segment))
  )
    return null;
  return { file: win.file, target, hidden: isPlayerHidden(node) };
}

export function withHiddenMark(
  selection: EditSelection,
  hidden: boolean,
): EditSelection {
  const raw = { ...(selection.node.raw ?? {}) };
  if (hidden) raw.gameview_hidden = "yes";
  else delete raw.gameview_hidden;
  const node = { ...selection.node, raw };
  return {
    ...selection,
    node,
    stack: selection.stack?.map((entry) =>
      entry.id === selection.id ? { ...entry, node } : entry,
    ),
  };
}

export function withOwnPicture(
  selection: EditSelection,
  gfx: string,
  shape: { width: number; height: number; frames: number },
): EditSelection {
  const draws = (node: GuiNode): GuiNode =>
    node.sprite
      ? { ...node, sprite: gfx }
      : node.background
        ? { ...node, background: { ...node.background, sprite: gfx } }
        : node;
  const node = draws(selection.node);
  return {
    ...selection,
    node,
    win: {
      ...selection.win,
      sprites: {
        ...selection.win.sprites,
        [gfx]: {
          resolved: true,
          frames: shape.frames,
          width: shape.width,
          height: shape.height,
          effect: null,
          effect_kind: null,
          cornered: false,
          bordersize: null,
        },
      },
    },
    stack: selection.stack?.map((entry) =>
      entry.id === selection.id ? { ...entry, node } : entry,
    ),
  };
}

export function spritePreviewOf(
  name: string,
  meta: GuiSprite | undefined,
  node: GuiNode,
): SpritePreview | null {
  if (!meta || !meta.width || !meta.height) return null;
  const url = spriteUrlOf(meta, name);
  if (!url) return null;
  const frames = Math.max(1, meta.frames || 1);
  const frame = Math.min(frames, Math.max(1, Number(node.frame) || 1));
  return {
    url,
    frame,
    frames,
    frameWidth: Math.round(meta.width / frames),
    frameHeight: meta.height,
  };
}

const LOC_KEY = /^[A-Za-z0-9_.-]+$/;

const OVERLAY_KEY = "GUISUB_";

const WORDS =
  /\p{L}\p{L}|[\p{Script=Han}\p{Script=Hiragana}\p{Script=Katakana}\p{Script=Hangul}]/u;

export function hasWords(text: string): boolean {
  return WORDS.test(text);
}

export function editTargetOf(
  selection: EditSelection,
  shown: string | undefined,
  locEdits: Readonly<Record<string, string>>,
  saved: Readonly<Record<string, string>>,

  guiTexts: Readonly<Record<string, string>> = NO_GUI_TEXTS,
): EditTarget {
  const { node, win } = selection;
  const sprite = node.sprite || node.background?.sprite || null;
  const meta = sprite ? win.sprites[sprite] : undefined;
  const spriteSize =
    meta?.resolved && meta.width && meta.height
      ? {
          width: Math.round(meta.width / Math.max(1, meta.frames || 1)),
          height: meta.height,
          frames: Math.max(1, meta.frames || 1),
        }
      : null;

  const label = node.text ? undefined : labelOver(selection);
  const text = textEditOf({
    win,
    id: label?.id ?? selection.id,
    raw: node.text || label?.node.text || null,
    printed: label ? label.shown : shown,
    source: label ? label.source : selection.source,
    locEdits,
    saved,
    guiTexts,
  });
  return {
    title: node.name || node.type,
    windowName: win.name,
    sprite,
    spriteSize,
    preview: sprite ? spritePreviewOf(sprite, meta, node) : null,
    ownPicture: ownPictureOf(selection, sprite),
    removal: removalOf(selection),
    flag: meta?.block_type === "maskedshieldtype",
    text,
    stack:
      (selection.stack?.length ?? 0) > 1
        ? selection.stack!.map((entry) => ({
            id: entry.id,
            kind: elementKind(entry.node),
            type: entry.node.type,
            name: entry.node.name || entry.node.type,
            backdrop: isBackdrop(entry.node),
            picked: entry.id === selection.id,
          }))
        : [],
  };
}

const NO_GUI_TEXTS: Readonly<Record<string, string>> = {};

export function guiTextKey(file: string, target: string): string {
  return `${file}\0${target}`;
}

export function guiTextOverrides(
  win: GuiWindow,
  guiTexts: Readonly<Record<string, string>>,
): Record<string, string> {
  const out: Record<string, string> = {};
  if (!win.file || !Object.keys(guiTexts).length) return out;
  for (const [id, target] of Object.entries(buildNamePathMap(win.node))) {
    const value = guiTexts[guiTextKey(win.file, target)];
    if (value !== undefined) out[id] = value;
  }
  return out;
}

function textEditOf({
  win,
  id,
  raw,
  printed,
  source,
  locEdits,
  saved,
  guiTexts,
}: {
  win: GuiWindow;
  id: string;

  raw: string | null;

  printed: string | undefined;
  source: TextSource | undefined;
  locEdits: Readonly<Record<string, string>>;
  saved: Readonly<Record<string, string>>;
  guiTexts: Readonly<Record<string, string>>;
}): TextEdit | null {
  if (!raw && printed === undefined && !source) return null;
  const info = raw ? win.texts[raw] : undefined;
  if (source?.kind === "key") {
    const current = locEdits[source.key] ?? printed ?? info?.display ?? "";
    const rewritten = source.key in locEdits || source.key in saved;
    return {
      editable: true,
      via: "key",
      key: source.key,
      current,
      rewritten,
      scope: source.scope,
      runtime: true,
    };
  }
  if (source?.kind === "person") {
    return {
      editable: true,
      via: "person",
      token: source.token,
      current: printed ?? "",
    };
  }
  if (printed !== undefined) {
    const reason =
      printed.trim() && !hasWords(printed) ? "computed" : "runtimeCountry";
    return { editable: false, current: printed, reason };
  }
  if (!raw) return null;
  const current = locEdits[raw] ?? info?.markup ?? info?.display ?? raw;
  const gui = (rewritten: boolean): TextEdit | null => {
    const target = win.file ? buildNamePathMap(win.node)[id] : undefined;
    if (!target) return null;
    const edited = guiTexts[guiTextKey(win.file, target)];
    return {
      editable: true,
      via: "gui",
      target,
      file: win.file,
      originalText: raw,
      originalKind: info?.kind ?? null,
      current: edited ?? current,
      rewritten: rewritten || edited !== undefined,
    };
  };
  if (raw.startsWith(OVERLAY_KEY)) {
    return gui(true) ?? { editable: false, current, reason: "literal" };
  }
  if (info?.kind === "runtime")
    return { editable: false, current, reason: "runtime" };
  if (!info || info.kind === "literal" || !LOC_KEY.test(raw)) {
    if (!hasWords(current))
      return { editable: false, current, reason: "placeholder" };
    return gui(false) ?? { editable: false, current, reason: "literal" };
  }
  return {
    editable: true,
    via: "key",
    key: raw,
    current,
    rewritten: raw in locEdits || raw in saved,
    scope: "everywhere",
    runtime: false,
  };
}

function labelOver(selection: EditSelection): StackEntry | undefined {
  const t = selection.node.type.toLowerCase();
  if (!(t.endsWith("buttontype") || t === "checkboxtype")) return undefined;
  for (const entry of selection.stack ?? []) {
    if (entry.id === selection.id) return undefined;
    if (
      entry.node.text &&
      entry.node.type.toLowerCase().endsWith("textboxtype")
    )
      return entry;
  }
  return undefined;
}

export function locEditOverrides(
  win: GuiWindow,
  locEdits: Readonly<Record<string, string>>,
): Record<string, string> {
  return locEditOverridesFor(win.node, locEdits);
}

export function locEditOverridesFor(
  root: GuiNode,
  locEdits: Readonly<Record<string, string>>,
): Record<string, string> {
  const out: Record<string, string> = {};
  if (!Object.keys(locEdits).length) return out;
  for (const row of flattenTree(root)) {
    const key = row.node.text;
    if (key && key in locEdits) out[row.id] = locEdits[key];
  }
  return out;
}
