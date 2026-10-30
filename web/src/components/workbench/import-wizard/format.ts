import type { WarnTranslator } from "./types";

export function formatTriple(
  created: number,
  updated: number,
  skipped: number,
  unit = "",
  words: { created: string; updated: string; skipped: string } = {
    created: "created",
    updated: "updated",
    skipped: "skipped",
  },
): string {
  const parts: string[] = [];
  if (created > 0) parts.push(`${created} ${words.created}`);
  if (updated > 0) parts.push(`${updated} ${words.updated}`);
  if (skipped > 0) parts.push(`${skipped} ${words.skipped}`);
  return parts.length > 0 ? parts.join(", ") + unit : `0${unit}`;
}

export function renderImportWarning(raw: string, t: WarnTranslator): string {
  if (!raw.startsWith("§§")) return raw;

  const firstSep = raw.indexOf("§§", 2);
  if (firstSep === -1) return raw;
  const code = raw.slice(2, firstSep);
  const secondSep = raw.indexOf("§§", firstSep + 2);
  if (secondSep === -1) return raw;
  const jsonStr = raw.slice(firstSep + 2, secondSep);
  const fallback = raw.slice(secondSep + 2);
  const key = `importWarning.${code}`;
  if (!t.has(key)) return fallback;
  let params: Record<string, string | number> = {};
  try {
    const parsed = JSON.parse(jsonStr) as unknown;
    if (parsed && typeof parsed === "object") {
      params = parsed as Record<string, string | number>;
    }
  } catch {
    return fallback;
  }
  try {
    return t(key, params);
  } catch {
    return fallback;
  }
}
