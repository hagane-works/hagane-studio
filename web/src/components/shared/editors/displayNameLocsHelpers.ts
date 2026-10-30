export const HOI4_SUPPORTED_LANGS = [
  "en",
  "zh",
  "ru",
  "fr",
  "de",
  "es",
  "ja",
  "ko",
  "pl",
  "pt",
] as const;

export type Hoi4Lang = (typeof HOI4_SUPPORTED_LANGS)[number];

export const DEFAULT_EXPORT_LANGUAGES: string[] = ["en"];

export const LANG_DISPLAY: Record<string, string> = {
  en: "EN",
  zh: "中",
  ru: "RU",
  fr: "FR",
  de: "DE",
  es: "ES",
  ja: "JA",
  ko: "KO",
  pl: "PL",
  pt: "PT",
};

export function orderLanguages(
  exportLanguages: readonly string[],
  valueKeys: readonly string[] = [],
): string[] {
  const ordered: string[] = ["en"];
  const seen = new Set(["en"]);
  for (const lang of exportLanguages) {
    if (!seen.has(lang)) {
      ordered.push(lang);
      seen.add(lang);
    }
  }
  for (const lang of valueKeys) {
    if (!seen.has(lang)) {
      ordered.push(lang);
      seen.add(lang);
    }
  }
  return ordered;
}

export function resolveActiveValue(
  value: Record<string, string> | null | undefined,
  activeLang: string,
  legacyFallback?: Record<string, string | undefined> | undefined,
): string {
  const primary = value?.[activeLang];
  if (primary != null && primary !== "") return primary;
  const legacy = legacyFallback?.[activeLang];
  if (legacy != null && legacy !== "") return legacy;
  return "";
}

export function availableLangsToAdd(
  currentTabs: readonly string[],
  supported: readonly string[] = HOI4_SUPPORTED_LANGS,
): string[] {
  const taken = new Set(currentTabs);
  return supported.filter((lang) => !taken.has(lang));
}

export function isLegacyFallback(
  value: Record<string, string> | null | undefined,
  activeLang: string,
  legacyFallback?: Record<string, string | undefined> | undefined,
): boolean {
  const primary = value?.[activeLang];
  if (primary != null && primary !== "") return false;
  const legacy = legacyFallback?.[activeLang];
  return legacy != null && legacy !== "";
}

export type LocalisationValueStatus =
  "imported" | "saved" | "legacy" | "missing";

export function localisationValueStatus(
  value: Record<string, string> | null | undefined,
  activeLang: string,
  legacyFallback?: Record<string, string | undefined>,
  valueSource: "imported" | "saved" = "saved",
): LocalisationValueStatus {
  const canonical = value?.[activeLang];
  if (typeof canonical === "string" && canonical.trim()) {
    return valueSource;
  }
  if (isLegacyFallback(value, activeLang, legacyFallback)) return "legacy";
  return "missing";
}
