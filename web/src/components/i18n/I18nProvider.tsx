"use client";

import { useEffect } from "react";
import { IntlProvider } from "use-intl";
import type { AppLocale } from "@/i18n/routing";
import { LOCALE_STORAGE_KEY } from "@/i18n/routing";
import { useLocaleStore } from "@/lib/store/localeStore";

export function I18nProvider({
  children,
  locale,
  messages,
}: {
  children: React.ReactNode;
  locale: AppLocale;
  messages: Record<string, unknown>;
}) {
  useEffect(() => {
    useLocaleStore.setState({ locale });
    document.documentElement.lang = locale;
    document.documentElement.dataset.locale = locale;
    try {
      localStorage.setItem(LOCALE_STORAGE_KEY, locale);
    } catch {}
  }, [locale]);

  return (
    <IntlProvider locale={locale} messages={messages} timeZone="UTC">
      {children}
    </IntlProvider>
  );
}
