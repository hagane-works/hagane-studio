import { create } from "zustand";

export const CONTENT_LOCALES = [
  "en",
  "zh",
  "ru",
  "de",
  "es",
  "fr",
  "ja",
  "ko",
  "pl",
  "pt",
] as const;
export type ContentLocale = (typeof CONTENT_LOCALES)[number];

export function isContentLocale(value: unknown): value is ContentLocale {
  return (
    typeof value === "string" &&
    CONTENT_LOCALES.includes(value as ContentLocale)
  );
}

export function normalizeContentLocales(
  values: readonly unknown[] | null | undefined,
): ContentLocale[] {
  const selected = new Set<ContentLocale>(["en"]);
  for (const value of values ?? []) {
    if (isContentLocale(value)) selected.add(value);
  }
  return CONTENT_LOCALES.filter((locale) => selected.has(locale));
}

interface ContentLocaleState {
  locale: ContentLocale;
  setLocale: (locale: ContentLocale) => void;
}

export const useContentLocaleStore = create<ContentLocaleState>()((set) => ({
  locale: "en",
  setLocale: (locale) => set({ locale }),
}));
