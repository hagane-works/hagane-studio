import type { CountryRow, HistoryFileRow } from "@/lib/api";
import { deriveGraphicalCulture2d } from "@/lib/graphicalCultures";

export interface CountryProfileDraft {
  nameEn: string;
  nameZh: string;
  displayNameLocs: Record<string, string>;
  adjectiveEn: string;
  adjectiveZh: string;
  adjectiveLocs: Record<string, string>;
  ideologyNameLocs: Record<string, Record<string, string>>;
  color: number[];
  colorUi: number[];
  graphicalCulture: string;
  capital: number;
}

export interface EffectiveCountryProfile {
  name?: string;
  adjective?: string;
  ideologyNames?: Record<string, string>;
  graphicalCulture?: string;
  capital?: number;
  ruling?: string;
  flag?: {
    url: string;
    width: number;
    height: number;
    project?: boolean;
  };
}

function inheritedText(
  raw: string | undefined,
  tag: string,
  fallback: string | undefined,
): string {
  const value = raw ?? "";
  if (!fallback) return value;
  const trimmed = value.trim();
  return !trimmed || trimmed.toUpperCase() === tag.toUpperCase()
    ? fallback
    : value;
}

function withEnglishFallback(
  values: Record<string, string> | undefined,
  tag: string,
  fallback: string | undefined,
): Record<string, string> {
  const next = { ...(values ?? {}) };
  if (
    fallback &&
    (!next.en?.trim() || next.en.trim().toUpperCase() === tag.toUpperCase())
  ) {
    next.en = fallback;
  }
  return next;
}

function ideologyNamesWithFallback(
  values: Record<string, Record<string, string>> | undefined,
  tag: string,
  fallbacks: Record<string, string> | undefined,
): Record<string, Record<string, string>> {
  const next = Object.fromEntries(
    Object.entries(values ?? {}).map(([root, locs]) => [root, { ...locs }]),
  );
  for (const [root, fallback] of Object.entries(fallbacks ?? {})) {
    next[root] = withEnglishFallback(next[root], tag, fallback);
  }
  return next;
}

export function countryToProfileDraft(
  country: CountryRow,
  history: HistoryFileRow | null,
  effective?: EffectiveCountryProfile | null,
): CountryProfileDraft {
  return {
    nameEn: inheritedText(country.name_en, country.tag, effective?.name),
    nameZh: country.name_zh ?? "",
    displayNameLocs: withEnglishFallback(
      country.display_name_locs,
      country.tag,
      effective?.name,
    ),
    adjectiveEn: (country.adjective_en ?? "").trim()
      ? country.adjective_en
      : (effective?.adjective ?? ""),
    adjectiveZh: country.adjective_zh ?? "",
    adjectiveLocs: withEnglishFallback(
      country.adjective_locs,
      "",
      effective?.adjective,
    ),
    ideologyNameLocs: ideologyNamesWithFallback(
      country.ideology_name_locs,
      country.tag,
      effective?.ideologyNames,
    ),
    color: Array.isArray(country.color) ? country.color : [],
    colorUi: Array.isArray(country.color_ui) ? country.color_ui : [],
    graphicalCulture: country.graphical_culture?.trim()
      ? country.graphical_culture
      : (effective?.graphicalCulture ?? ""),
    capital: history?.capital ?? effective?.capital ?? 0,
  };
}

function cloneRecord(
  values: Record<string, string> | undefined,
): Record<string, string> {
  return { ...(values ?? {}) };
}

function changedRecord(
  stored: Record<string, string> | undefined,
  initial: Record<string, string>,
  current: Record<string, string>,
): Record<string, string> | undefined {
  if (semanticEqual(initial, current)) return undefined;
  const next = cloneRecord(stored);
  for (const key of new Set([
    ...Object.keys(initial),
    ...Object.keys(current),
  ])) {
    if (semanticEqual(initial[key], current[key])) continue;
    if (Object.prototype.hasOwnProperty.call(current, key))
      next[key] = current[key];
    else delete next[key];
  }
  return next;
}

function changedNestedRecord(
  stored: Record<string, Record<string, string>> | undefined,
  initial: Record<string, Record<string, string>>,
  current: Record<string, Record<string, string>>,
): Record<string, Record<string, string>> | undefined {
  if (semanticEqual(initial, current)) return undefined;
  const next = Object.fromEntries(
    Object.entries(stored ?? {}).map(([root, locs]) => [root, { ...locs }]),
  );
  for (const root of new Set([
    ...Object.keys(initial),
    ...Object.keys(current),
  ])) {
    const changed = changedRecord(
      next[root],
      initial[root] ?? {},
      current[root] ?? {},
    );
    if (changed === undefined) continue;
    if (Object.prototype.hasOwnProperty.call(current, root))
      next[root] = changed;
    else delete next[root];
  }
  return next;
}

export function countryProfilePatch(
  country: CountryRow,
  initial: CountryProfileDraft,
  current: CountryProfileDraft,
): Partial<CountryRow> {
  const patch: Partial<CountryRow> = {};
  if (initial.nameEn !== current.nameEn) patch.name_en = current.nameEn;
  if (initial.nameZh !== current.nameZh) patch.name_zh = current.nameZh;
  const displayNames = changedRecord(
    country.display_name_locs,
    initial.displayNameLocs,
    current.displayNameLocs,
  );
  if (displayNames !== undefined) patch.display_name_locs = displayNames;
  if (initial.adjectiveEn !== current.adjectiveEn)
    patch.adjective_en = current.adjectiveEn;
  if (initial.adjectiveZh !== current.adjectiveZh)
    patch.adjective_zh = current.adjectiveZh;
  const adjectives = changedRecord(
    country.adjective_locs,
    initial.adjectiveLocs,
    current.adjectiveLocs,
  );
  if (adjectives !== undefined) patch.adjective_locs = adjectives;
  const ideologyNames = changedNestedRecord(
    country.ideology_name_locs,
    initial.ideologyNameLocs,
    current.ideologyNameLocs,
  );
  if (ideologyNames !== undefined) patch.ideology_name_locs = ideologyNames;
  if (!semanticEqual(initial.color, current.color)) patch.color = current.color;
  if (!semanticEqual(initial.colorUi, current.colorUi))
    patch.color_ui = current.colorUi;
  if (initial.graphicalCulture !== current.graphicalCulture) {
    patch.graphical_culture = current.graphicalCulture;
    patch.graphical_culture_2d = deriveGraphicalCulture2d(
      current.graphicalCulture,
    );
  }
  return patch;
}

export function profileCountryPatchMatches(
  country: CountryRow,
  patch: Partial<CountryRow>,
): boolean {
  return Object.entries(patch).every(([key, value]) =>
    semanticEqual(country[key as keyof CountryRow], value),
  );
}

function semanticEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;

  if (Array.isArray(a) || Array.isArray(b)) {
    if (!Array.isArray(a) || !Array.isArray(b) || a.length !== b.length)
      return false;
    return a.every((value, index) => semanticEqual(value, b[index]));
  }

  if (
    a === null ||
    b === null ||
    typeof a !== "object" ||
    typeof b !== "object"
  ) {
    return false;
  }

  const aRecord = a as Record<string, unknown>;
  const bRecord = b as Record<string, unknown>;
  const aKeys = Object.keys(aRecord);
  const bKeys = Object.keys(bRecord);
  if (aKeys.length !== bKeys.length) return false;

  return aKeys.every(
    (key) =>
      Object.prototype.hasOwnProperty.call(bRecord, key) &&
      semanticEqual(aRecord[key], bRecord[key]),
  );
}

function sameArray(
  a: readonly number[] | undefined,
  b: readonly number[],
): boolean {
  return semanticEqual(Array.isArray(a) ? a : [], b);
}

export function profileCountryReadbackMatches(
  country: CountryRow,
  draft: CountryProfileDraft,
): boolean {
  return (
    country.name_en === draft.nameEn &&
    country.name_zh === draft.nameZh &&
    semanticEqual(country.display_name_locs ?? {}, draft.displayNameLocs) &&
    country.adjective_en === draft.adjectiveEn &&
    country.adjective_zh === draft.adjectiveZh &&
    semanticEqual(country.adjective_locs ?? {}, draft.adjectiveLocs) &&
    semanticEqual(country.ideology_name_locs ?? {}, draft.ideologyNameLocs) &&
    sameArray(country.color, draft.color) &&
    sameArray(country.color_ui, draft.colorUi) &&
    (country.graphical_culture ?? "") === draft.graphicalCulture
  );
}

export function profileHistoryReadbackMatches(
  history: HistoryFileRow,
  draft: CountryProfileDraft,
): boolean {
  return history.capital === draft.capital;
}

export function visibleCountryName(
  country: CountryRow,
  draft: CountryProfileDraft,
  root: string,
  contentLocale: string,
): { value: string; fallback: boolean } {
  const variant = draft.ideologyNameLocs[root]?.[contentLocale]?.trim();
  if (variant) return { value: variant, fallback: false };
  const legacyVariant = legacyIdeologyCountryName(country, root, contentLocale);
  if (legacyVariant) return { value: legacyVariant, fallback: false };
  const canonical = draft.displayNameLocs[contentLocale]?.trim();
  if (canonical) return { value: canonical, fallback: true };
  const legacy = contentLocale === "zh" ? draft.nameZh : draft.nameEn;
  return { value: legacy || country.name || country.tag, fallback: true };
}

export function legacyIdeologyCountryName(
  country: CountryRow,
  root: string,
  contentLocale: string,
): string {
  const language = contentLocale === "zh" ? "zh" : "en";
  switch (root) {
    case "democratic":
      return country[`name_democratic_${language}`]?.trim() ?? "";
    case "fascism":
      return country[`name_fascism_${language}`]?.trim() ?? "";
    case "communism":
      return country[`name_communism_${language}`]?.trim() ?? "";
    case "neutrality":
      return country[`name_neutrality_${language}`]?.trim() ?? "";
    default:
      return "";
  }
}
