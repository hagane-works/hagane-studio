import type { ContentLocale } from "@/lib/store/contentLocaleStore";

export function localizedName(
  locale: ContentLocale,
  en: string | null | undefined,
  zh: string | null | undefined,
  fallback: string,
): string {
  const enT = (en ?? "").trim();
  const zhT = (zh ?? "").trim();
  return locale === "zh" ? zhT || enT || fallback : enT || fallback;
}

export function localizedDesc(
  locale: ContentLocale,
  en: string | null | undefined,
  zh: string | null | undefined,
  fallback = "",
): string {
  return localizedName(locale, en, zh, fallback);
}

function locsValue(
  locs: Record<string, string> | null | undefined,
  lang: string,
): string {
  if (!locs) return "";
  const v = locs[lang];
  return typeof v === "string" ? v.trim() : "";
}

export function resolveDisplayName(
  locale: ContentLocale,
  locs: Record<string, string> | null | undefined,
  legacyEn: string | null | undefined,
  legacyZh: string | null | undefined,
  fallback: string,
): string {
  const enLegacy = (legacyEn ?? "").trim();
  const zhLegacy = (legacyZh ?? "").trim();
  const activeLocs = locsValue(locs, locale);
  if (locale === "zh") {
    return (
      activeLocs || zhLegacy || locsValue(locs, "en") || enLegacy || fallback
    );
  }
  return activeLocs || locsValue(locs, "en") || enLegacy || fallback;
}

export function resolveDisplayDesc(
  locale: ContentLocale,
  locs: Record<string, string> | null | undefined,
  legacyEn: string | null | undefined,
  legacyZh: string | null | undefined,
  fallback = "",
): string {
  return resolveDisplayName(locale, locs, legacyEn, legacyZh, fallback);
}
