"use client";

import { useLocale, useTranslations } from "use-intl";
import { useSearchParams } from "@/platform/navigation";
import { usePathname, useRouter } from "@/i18n/navigation";
import { type AppLocale, isAppLocale } from "@/i18n/routing";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";
import { LanguageMenu, LANGUAGE_NATIVE_NAMES } from "./LanguageMenu";

export function LanguageSwitcher({
  className = "",
  appearance = "workbench",
}: {
  className?: string;
  appearance?: "public" | "workbench";
}) {
  const current = useLocale();
  const locale: AppLocale = isAppLocale(current) ? current : "en";
  const t = useTranslations("common.language");
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  return (
    <LanguageMenu
      locale={locale}
      appearance={appearance}
      className={className}
      labels={{
        current: t("current", { language: LANGUAGE_NATIVE_NAMES[locale] }),
        heading: t(appearance === "public" ? "publicHeading" : "heading"),
        label: t("label"),
        description: t("description"),
      }}
      onLocaleChange={(nextLocale) => {
        if (!confirmDiscardIfDirty()) return false;
        const query = searchParams.toString();
        const hash = typeof window === "undefined" ? "" : window.location.hash;
        router.replace(`${pathname}${query ? `?${query}` : ""}${hash}`, {
          locale: nextLocale,
          scroll: false,
        });
      }}
    />
  );
}
