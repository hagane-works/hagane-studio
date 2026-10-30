import { create } from "zustand";
import { persist } from "zustand/middleware";
import { APP_LOCALES, type AppLocale, DEFAULT_LOCALE } from "@/i18n/routing";

export type Locale = AppLocale;
export type { AppLocale } from "@/i18n/routing";

export const LOCALES: readonly AppLocale[] = APP_LOCALES;

export type DefaultContentLocale = "en";
export function defaultContentLocaleFor(
  appLocale: AppLocale,
): DefaultContentLocale {
  void appLocale;
  return "en";
}

interface LocaleStore {
  locale: AppLocale;
  setLocale: (locale: AppLocale) => void;
}

export const useLocaleStore = create<LocaleStore>()(
  persist(
    (set) => ({
      locale: DEFAULT_LOCALE,
      setLocale: (locale) => set({ locale }),
    }),
    {
      name: "hoi4-locale",

      skipHydration: true,

      onRehydrateStorage: () => (state) => {
        if (state && typeof document !== "undefined") {
          document.documentElement.lang = state.locale;
          document.documentElement.dataset.locale = state.locale;
        }
      },
    },
  ),
);
