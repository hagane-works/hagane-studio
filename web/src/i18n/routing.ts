export const APP_LOCALES = [
  "en",
  "zh",
  "ru",
  "de",
  "es",
  "fr",
  "ja",
  "ko",
] as const;
export type AppLocale = (typeof APP_LOCALES)[number];

export const DEFAULT_LOCALE: AppLocale = "en";
export const LOCALE_COOKIE_NAME = "NEXT_LOCALE";

export function isAppLocale(value: unknown): value is AppLocale {
  return typeof value === "string" && APP_LOCALES.includes(value as AppLocale);
}

export function preferredLocale(): AppLocale {
  try {
    const saved = localStorage.getItem(LOCALE_STORAGE_KEY);
    if (isAppLocale(saved)) return saved;
  } catch {}
  for (const language of navigator.languages ?? [navigator.language]) {
    const base = language.toLowerCase().split("-")[0];
    if (isAppLocale(base)) return base;
  }
  return DEFAULT_LOCALE;
}

export const LOCALE_STORAGE_KEY = "haganeworks:locale";

export const WIKI_LOCALES = ["en", "zh"] as const;
export type WikiLocale = (typeof WIKI_LOCALES)[number];

export function isWikiLocale(locale: AppLocale): locale is WikiLocale {
  return locale === "en" || locale === "zh";
}
