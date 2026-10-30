import { Languages } from "lucide-react";
import { useTranslations } from "use-intl";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";
import { CONTENT_LANGUAGE_NATIVE_NAMES } from "@/lib/i18n/hoi4ContentLanguages";

interface ImportLanguageSelectorProps {
  detected: readonly ContentLocale[];
  counts: Partial<Record<ContentLocale, number>>;
  selected: readonly ContentLocale[];
  onChange: (next: ContentLocale[]) => void;
}

export function ImportLanguageSelector({
  detected,
  counts,
  selected,
  onChange,
}: ImportLanguageSelectorProps) {
  const t = useTranslations("unifiedModImport");
  const options: ContentLocale[] = [
    "en",
    ...detected.filter((locale) => locale !== "en"),
  ];

  return (
    <section
      className="rounded-md border border-border bg-muted/20 p-3"
      aria-labelledby="import-language-title"
    >
      <div className="flex items-start gap-2">
        <Languages
          className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground"
          aria-hidden
        />
        <div className="min-w-0 flex-1">
          <h3 id="import-language-title" className="text-xs font-semibold">
            {t("contentLanguagesTitle")}
          </h3>
          <p className="mt-0.5 text-xs leading-relaxed text-muted-foreground">
            {t("contentLanguagesHelp")}
          </p>
        </div>
      </div>
      <div className="mt-2 grid grid-cols-2 gap-1.5 sm:grid-cols-3">
        {options.map((locale) => {
          const isEnglish = locale === "en";
          const checked = isEnglish || selected.includes(locale);
          return (
            <label
              key={locale}
              className="flex min-w-0 items-center gap-2 rounded border border-border bg-background px-2.5 py-2 text-xs"
            >
              <input
                type="checkbox"
                checked={checked}
                disabled={isEnglish}
                onChange={(event) => {
                  const next = event.target.checked
                    ? [...selected, locale]
                    : selected.filter((candidate) => candidate !== locale);
                  onChange([
                    "en",
                    ...next.filter((candidate) => candidate !== "en"),
                  ]);
                }}
                className="accent-primary"
              />
              <span className="min-w-0 flex-1 truncate">
                {CONTENT_LANGUAGE_NATIVE_NAMES[locale]}
              </span>
              <span className="shrink-0 font-mono text-muted-foreground">
                {counts[locale] ?? 0}
              </span>
            </label>
          );
        })}
      </div>
      {!detected.includes("en") ? (
        <p className="mt-2 text-xs text-amber-600 dark:text-amber-400">
          {t("contentLanguagesEnglishMissing")}
        </p>
      ) : null}
      <p className="mt-2 text-xs text-muted-foreground">
        {t("contentLanguagesSelectionHint")}
      </p>
    </section>
  );
}
