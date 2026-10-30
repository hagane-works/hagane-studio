import { getCountryAliases, loadCountryAliases } from "./country-aliases";

export const IMAGE_EXTS = new Set(["png", "dds", "tga"]);

export const FLAT_COMMON_TEXT_DIRS = new Set(["wargoals"]);

export function isImageFile(name: string): boolean {
  const ext = name.split(".").pop()?.toLowerCase() || "";
  return IMAGE_EXTS.has(ext);
}

export function filenameMatchesTag(filename: string, tag: string): boolean {
  const stem = filename.replace(/\.[^.]*$/, "").toUpperCase();
  const tagUpper = tag.toUpperCase();
  return stem.split(/[_.\-\s]+/).includes(tagUpper);
}

export function filenameMatchesTagOrAlias(
  filename: string,
  tag: string,
): boolean {
  if (filenameMatchesTag(filename, tag)) return true;
  const aliases = getCountryAliases(tag);
  if (aliases.length === 0) return false;
  const stem = filename.replace(/\.[^.]*$/, "").toLowerCase();
  const tokens = stem.split(/[_.\-\s]+/);
  return aliases.some((alias) => {
    const a = alias.toLowerCase();
    if (a.includes("_")) {
      const aTokens = a.split("_");
      if (aTokens.length > tokens.length) return false;
      for (let i = 0; i <= tokens.length - aTokens.length; i++) {
        if (aTokens.every((t, j) => tokens[i + j] === t)) return true;
      }
      return false;
    }
    return tokens.includes(a);
  });
}

const SHARED_FILENAME_MARKERS = [
  "_generic",
  "generic_",
  "_shared",
  "shared_",
  "_l_english",
  "_l_simp_chinese",
  "_l_french",
  "_l_german",
  "_l_russian",
  "_l_spanish",
  "_l_polish",
  "_l_japanese",
  "_l_braz_por",
  "_l_korean",
  "_l_turkish",
  "ideas_l_",
  "characters_l_",
  "events_l_",
  "decisions_l_",
  "focus_l_",
  "technology_l_",
  "operations_l_",
  "designer_l_",
  "_mtg",
  "mtg_",
  "_wtt",
  "wtt_",
  "_lar",
  "lar_",
  "_bba",
  "bba_",
  "_nsb",
  "nsb_",
  "_aat",
  "aat_",
  "_bftb",
  "bftb_",
  "_goe",
  "goe_",
  "_got",
  "got_",
  "_dod",
  "dod_",
  "_sea",
  "sea_",
  "_toa",
  "toa_",
  "_wuw",
  "wuw_",
];

function filenameLooksShared(filename: string): boolean {
  const lower = filename.toLowerCase();
  return SHARED_FILENAME_MARKERS.some((marker) => lower.includes(marker));
}

const GLOBAL_NAMESPACE_MARKERS = new Set([
  "news",
  "generic",
  "shared",
  "world",
  "global",
  "achievement",
  "achievements",
  "election",
  "elections",
  "decisions",
  "panay",
  "capitulation",
]);

function filenameIsGlobalNamespace(filename: string): boolean {
  const stem = filename.replace(/\.[^.]*$/, "").toLowerCase();
  if (GLOBAL_NAMESPACE_MARKERS.has(stem)) return true;
  return stem
    .split(/[_.-]+/)
    .some((token) => GLOBAL_NAMESPACE_MARKERS.has(token));
}

const _COUNTRY_TAG_LINE_RE =
  /^\s*([A-Za-z0-9]{2,5})\s*=\s*"([^"]+)"\s*(?:#.*)?$/;

const _DYNAMIC_FLAG_RE = /^\s*dynamic_tags\s*=\s*yes\b/i;

const _LINE_SPLIT_RE = /\r\n|[\r\n\v\f\x1c\x1d\x1e\x85\u2028\u2029]/;

export function countDeclaredCountryTags(contents: string[]): number {
  const seen = new Set<string>();
  for (let content of contents) {
    if (!content) continue;

    if (content.charCodeAt(0) === 0xfeff) content = content.slice(1);
    for (const raw of content.split(_LINE_SPLIT_RE)) {
      collectDeclaredCountryTagLine(raw, seen);
    }
  }
  return seen.size;
}

function collectDeclaredCountryTagLine(raw: string, seen: Set<string>): void {
  const line = raw.replace(/\s+$/, "");
  if (!line || line.trimStart().startsWith("#")) return;
  if (_DYNAMIC_FLAG_RE.test(line)) return;
  const match = _COUNTRY_TAG_LINE_RE.exec(line);
  if (!match) return;
  seen.add(match[1].toUpperCase());
}

function isCountryTagLineBoundary(code: number): boolean {
  return (
    code === 0x0a ||
    code === 0x0b ||
    code === 0x0c ||
    code === 0x0d ||
    (code >= 0x1c && code <= 0x1e) ||
    code === 0x85 ||
    code === 0x2028 ||
    code === 0x2029
  );
}

async function collectDeclaredCountryTagsCooperatively(
  content: string,
  seen: Set<string>,
  checkpoint: () => Promise<void> | null,
): Promise<void> {
  if (!content) return;

  let lineStart = content.charCodeAt(0) === 0xfeff ? 1 : 0;
  let charsSinceCheckpoint = 0;
  for (let cursor = lineStart; cursor < content.length; cursor += 1) {
    const code = content.charCodeAt(cursor);
    charsSinceCheckpoint += 1;
    if (isCountryTagLineBoundary(code)) {
      collectDeclaredCountryTagLine(content.slice(lineStart, cursor), seen);

      if (code === 0x0d && content.charCodeAt(cursor + 1) === 0x0a) cursor += 1;
      lineStart = cursor + 1;
    }

    if (charsSinceCheckpoint >= 2_048 || lineStart === cursor + 1) {
      charsSinceCheckpoint = 0;
      const pendingYield = checkpoint();
      if (pendingYield) await pendingYield;
    }
  }
  collectDeclaredCountryTagLine(content.slice(lineStart), seen);
}

const COUNTRY_TAG_READ_CONCURRENCY = 8;

export async function collectDeclaredCountryTagCount(
  files: WalkFile[],
): Promise<number> {
  const sorted = [...files].sort((a, b) => a.path.localeCompare(b.path));
  const seen = new Set<string>();
  const checkpoint = createMainThreadCheckpoint();

  for (
    let start = 0;
    start < sorted.length;
    start += COUNTRY_TAG_READ_CONCURRENCY
  ) {
    const batch = sorted.slice(start, start + COUNTRY_TAG_READ_CONCURRENCY);
    const contents = await Promise.all(batch.map((item) => item.file.text()));
    for (const content of contents) {
      await collectDeclaredCountryTagsCooperatively(content, seen, checkpoint);
    }

    const pendingYield = checkpoint();
    if (pendingYield) await pendingYield;
  }
  return seen.size;
}

export async function tryGetDir(
  parent: FileSystemDirectoryHandle,
  ...segments: string[]
): Promise<FileSystemDirectoryHandle | null> {
  let current = parent;
  for (const seg of segments) {
    try {
      current = await current.getDirectoryHandle(seg);
    } catch {
      return null;
    }
  }
  return current;
}

export interface WalkFile {
  file: File;
  path: string;
}

export interface WalkOptions {
  predicate: (name: string, fullPath: string) => boolean;

  skipSubtrees?: string[];
}

const WALK_IO_CONCURRENCY = 16;

type YieldScheduler = { yield?: () => Promise<void> };

export function yieldToMainThread(): Promise<void> {
  const scheduler = (
    globalThis as typeof globalThis & { scheduler?: YieldScheduler }
  ).scheduler;
  if (typeof scheduler?.yield === "function") {
    return scheduler.yield.call(scheduler);
  }

  return new Promise((resolve) => setTimeout(resolve, 0));
}

const SCANNER_TASK_BUDGET_MS = 8;
const SCANNER_CHECKPOINTS_PER_YIELD = 256;

function scannerNow(): number {
  return typeof performance !== "undefined" &&
    typeof performance.now === "function"
    ? performance.now()
    : Date.now();
}

export function createMainThreadCheckpoint(
  budgetMs = SCANNER_TASK_BUDGET_MS,
): () => Promise<void> | null {
  let sliceStartedAt = scannerNow();
  let checkpoints = 0;

  return () => {
    checkpoints += 1;
    if (
      checkpoints < SCANNER_CHECKPOINTS_PER_YIELD &&
      scannerNow() - sliceStartedAt < budgetMs
    ) {
      return null;
    }
    checkpoints = 0;
    return yieldToMainThread().then(() => {
      sliceStartedAt = scannerNow();
    });
  };
}

export async function walkFiles(
  dirHandle: FileSystemDirectoryHandle,
  prefix: string,
  opts: WalkOptions,
): Promise<WalkFile[]> {
  const shouldSkip = (p: string): boolean =>
    !!opts.skipSubtrees?.some((s) => p === s || p.startsWith(s + "/"));

  type Slot =
    | { kind: "file"; handle: FileSystemFileHandle; path: string }
    | { kind: "dir"; handle: FileSystemDirectoryHandle; path: string };

  const mainThreadCheckpoint = createMainThreadCheckpoint();

  async function walk(
    dir: FileSystemDirectoryHandle,
    p: string,
  ): Promise<WalkFile[]> {
    if (shouldSkip(p)) return [];

    const slots: Slot[] = [];
    for await (const entry of dir.values()) {
      const entryPath = p ? `${p}/${entry.name}` : entry.name;
      if (entry.kind === "file") {
        if (opts.predicate(entry.name, entryPath)) {
          slots.push({
            kind: "file",
            handle: entry as FileSystemFileHandle,
            path: entryPath,
          });
        }
      } else if (entry.kind === "directory") {
        slots.push({
          kind: "dir",
          handle: entry as FileSystemDirectoryHandle,
          path: entryPath,
        });
      }
      const pendingYield = mainThreadCheckpoint();
      if (pendingYield) await pendingYield;
    }

    const out: WalkFile[] = [];
    let i = 0;
    while (i < slots.length) {
      const slot = slots[i];
      if (slot.kind === "dir") {
        const nested = await walk(slot.handle, slot.path);

        for (const file of nested) {
          out.push(file);
          const pendingYield = mainThreadCheckpoint();
          if (pendingYield) await pendingYield;
        }
        i += 1;
        continue;
      }

      let j = i;
      while (
        j < slots.length &&
        slots[j].kind === "file" &&
        j - i < WALK_IO_CONCURRENCY
      )
        j += 1;
      const batch = slots.slice(i, j) as Extract<Slot, { kind: "file" }>[];
      const files = await Promise.all(batch.map((s) => s.handle.getFile()));
      for (let k = 0; k < batch.length; k++) {
        out.push({ file: files[k], path: batch[k].path });
      }
      const pendingYield = mainThreadCheckpoint();
      if (pendingYield) await pendingYield;
      i = j;
    }
    return out;
  }

  return walk(dirHandle, prefix);
}

export async function collectTxtFiles(
  dirHandle: FileSystemDirectoryHandle,
  prefix: string,
  opts: {
    filterTag?: string;
    keepShared?: boolean;

    keepGlobalNamespace?: boolean;
    skipSubtrees?: string[];

    stats?: { seen: number; kept: number };
  } = {},
): Promise<WalkFile[]> {
  if (opts.filterTag) await loadCountryAliases().catch(() => undefined);
  return walkFiles(dirHandle, prefix, {
    predicate: (name) => {
      if (!name.endsWith(".txt")) return false;
      if (opts.stats) opts.stats.seen += 1;
      const keep = ((): boolean => {
        if (!opts.filterTag) return true;
        if (filenameMatchesTagOrAlias(name, opts.filterTag)) return true;
        if (opts.keepShared && filenameLooksShared(name)) return true;
        if (opts.keepGlobalNamespace && filenameIsGlobalNamespace(name))
          return true;
        return false;
      })();
      if (keep && opts.stats) opts.stats.kept += 1;
      return keep;
    },
    skipSubtrees: opts.skipSubtrees,
  });
}

export async function collectGfxFiles(
  dirHandle: FileSystemDirectoryHandle,
  prefix: string,
): Promise<WalkFile[]> {
  return walkFiles(dirHandle, prefix, {
    predicate: (name) => name.endsWith(".gfx"),
  });
}

export async function collectGuiFiles(
  dirHandle: FileSystemDirectoryHandle,
  prefix: string,
): Promise<WalkFile[]> {
  return walkFiles(dirHandle, prefix, {
    predicate: (name) => name.endsWith(".gui"),
  });
}

export async function collectGfxAndGuiFiles(
  dirHandle: FileSystemDirectoryHandle,
  prefix: string,
): Promise<{ gfx: WalkFile[]; gui: WalkFile[] }> {
  const files = await walkFiles(dirHandle, prefix, {
    predicate: (name) => name.endsWith(".gfx") || name.endsWith(".gui"),
  });
  const gfx: WalkFile[] = [];
  const gui: WalkFile[] = [];
  const checkpoint = createMainThreadCheckpoint();
  for (const file of files) {
    if (file.file.name.endsWith(".gfx")) gfx.push(file);
    else gui.push(file);
    const pendingYield = checkpoint();
    if (pendingYield) await pendingYield;
  }
  return { gfx, gui };
}

export function isModMapFile(path: string): boolean {
  const normalized = path.replace(/\\/g, "/").toLowerCase();
  if (!normalized.startsWith("map/")) return false;
  if (normalized === "map/terrain/colormap_rgb_cityemissivemask_a.dds")
    return true;
  return /\.(bmp|txt|csv|map)$/.test(normalized);
}

export async function collectMapFiles(
  dirHandle: FileSystemDirectoryHandle,
): Promise<WalkFile[]> {
  return walkFiles(dirHandle, "map", {
    predicate: (_name, path) => isModMapFile(path),
  });
}

export function isModFlagFile(path: string): boolean {
  const normalized = path.replace(/\\/g, "/").toLowerCase();
  return /^gfx\/flags\/[a-z0-9]{3}(?:_[a-z0-9_]+)?\.(tga|dds|png)$/.test(
    normalized,
  );
}

export async function collectFlagFiles(
  dirHandle: FileSystemDirectoryHandle,
): Promise<WalkFile[]> {
  return walkFiles(dirHandle, "gfx/flags", {
    predicate: (_name, path) => isModFlagFile(path),
    skipSubtrees: ["gfx/flags/medium", "gfx/flags/small"],
  });
}

export async function collectLocFiles(
  dirHandle: FileSystemDirectoryHandle,
  prefix: string,
): Promise<WalkFile[]> {
  return walkFiles(dirHandle, prefix, {
    predicate: (name) => name.toLowerCase().endsWith(".yml"),
  });
}

export async function readFileByPath(
  dirHandle: FileSystemDirectoryHandle,
  path: string,
): Promise<File | null> {
  const parts = path.replace(/\\/g, "/").split("/").filter(Boolean);
  let current = dirHandle;
  for (let i = 0; i < parts.length - 1; i++) {
    try {
      current = await current.getDirectoryHandle(parts[i]);
    } catch {
      return null;
    }
  }
  try {
    const fileHandle = await current.getFileHandle(parts[parts.length - 1]);
    return await fileHandle.getFile();
  } catch {
    return null;
  }
}

export function imageExtensionAlternates(path: string): string[] {
  const dotIdx = path.lastIndexOf(".");
  if (dotIdx <= 0) return [];
  const base = path.substring(0, dotIdx);
  const ext = path.substring(dotIdx + 1).toLowerCase();
  if (!IMAGE_EXTS.has(ext)) return [];

  return [...IMAGE_EXTS].filter((e) => e !== ext).map((e) => `${base}.${e}`);
}

export async function findImageFile(
  dirHandle: FileSystemDirectoryHandle,
  path: string,
): Promise<WalkFile | null> {
  const exact = await readFileByPath(dirHandle, path);
  if (exact) return { file: exact, path };

  for (const altPath of imageExtensionAlternates(path)) {
    const altFile = await readFileByPath(dirHandle, altPath);
    if (altFile) return { file: altFile, path: altPath };
  }

  const dlcDir = await tryGetDir(dirHandle, "dlc");
  if (!dlcDir) return null;

  for await (const entry of dlcDir.values()) {
    if (entry.kind !== "directory") continue;
    const dlcSub = entry as FileSystemDirectoryHandle;
    const dlcFile = await readFileByPath(dlcSub, path);
    if (dlcFile) return { file: dlcFile, path };
    for (const altPath of imageExtensionAlternates(path)) {
      const altFile = await readFileByPath(dlcSub, altPath);
      if (altFile) return { file: altFile, path: altPath };
    }
  }
  return null;
}

export async function findDirContaining(
  parent: FileSystemDirectoryHandle,
  tag: string,
): Promise<{ handle: FileSystemDirectoryHandle; name: string }[]> {
  const tagUpper = tag.toUpperCase();
  const matches: { handle: FileSystemDirectoryHandle; name: string }[] = [];
  for await (const entry of parent.values()) {
    if (
      entry.kind === "directory" &&
      entry.name.toUpperCase().includes(tagUpper)
    ) {
      matches.push({
        handle: entry as FileSystemDirectoryHandle,
        name: entry.name,
      });
    }
  }
  return matches;
}
