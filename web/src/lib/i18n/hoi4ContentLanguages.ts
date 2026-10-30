import type { ContentLocale } from "@/lib/store/contentLocaleStore";

export const CONTENT_LANGUAGE_NATIVE_NAMES: Record<ContentLocale, string> = {
  en: "English",
  zh: "中文",
  ru: "Русский",
  de: "Deutsch",
  es: "Español",
  fr: "Français",
  ja: "日本語",
  ko: "한국어",
  pl: "Polski",
  pt: "Português",
};

const HOI4_DIRECTORY_TO_LOCALE: Readonly<Record<string, ContentLocale>> = {
  english: "en",
  simp_chinese: "zh",
  russian: "ru",
  german: "de",
  spanish: "es",
  french: "fr",
  japanese: "ja",
  korean: "ko",
  polish: "pl",
  braz_por: "pt",
};

export function contentLocaleFromLocalisationPath(
  path: string,
): ContentLocale | null {
  const normalized = path.replaceAll("\\", "/").toLowerCase();
  for (const [directory, locale] of Object.entries(HOI4_DIRECTORY_TO_LOCALE)) {
    if (
      normalized.includes(`/localisation/${directory}/`) ||
      normalized.startsWith(`localisation/${directory}/`) ||
      normalized.endsWith(`_l_${directory}.yml`)
    ) {
      return locale;
    }
  }
  return null;
}

export function detectedContentLocales(
  paths: readonly string[],
): ContentLocale[] {
  const found = new Set<ContentLocale>();
  for (const path of paths) {
    const locale = contentLocaleFromLocalisationPath(path);
    if (locale) found.add(locale);
  }
  const ordered = Object.values(HOI4_DIRECTORY_TO_LOCALE).filter(
    (locale, index, all) => all.indexOf(locale) === index,
  );
  return ordered.filter((locale) => found.has(locale));
}

export function localisationFileCountsByLocale(
  paths: readonly string[],
): Partial<Record<ContentLocale, number>> {
  const counts: Partial<Record<ContentLocale, number>> = {};
  for (const path of paths) {
    const locale = contentLocaleFromLocalisationPath(path);
    if (locale) counts[locale] = (counts[locale] ?? 0) + 1;
  }
  return counts;
}

export function shouldImportLocalisationPath(
  path: string,
  selected: readonly ContentLocale[],
): boolean {
  const locale = contentLocaleFromLocalisationPath(path);
  return locale === null || selected.includes(locale);
}

export function mergeProjectContentLanguages(
  existing: readonly string[] | null | undefined,
  selected: readonly ContentLocale[],
): ContentLocale[] {
  const merged = new Set<ContentLocale>(["en"]);
  for (const candidate of [...(existing ?? []), ...selected]) {
    if (
      Object.prototype.hasOwnProperty.call(
        CONTENT_LANGUAGE_NATIVE_NAMES,
        candidate,
      )
    ) {
      merged.add(candidate as ContentLocale);
    }
  }
  return [...merged];
}
