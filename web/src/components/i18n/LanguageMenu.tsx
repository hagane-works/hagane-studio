"use client";

import { useMemo, useState } from "react";
import { Check, ChevronDown, Globe2 } from "lucide-react";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { APP_LOCALES, type AppLocale } from "@/i18n/routing";

export const LANGUAGE_NATIVE_NAMES: Record<AppLocale, string> = {
  en: "English",
  zh: "中文",
  ru: "Русский",
  de: "Deutsch",
  es: "Español",
  fr: "Français",
  ja: "日本語",
  ko: "한국어",
};

const DISPLAY_LANGUAGE_TAGS: Record<AppLocale, string> = {
  en: "en",
  zh: "zh-Hans",
  ru: "ru",
  de: "de",
  es: "es",
  fr: "fr",
  ja: "ja",
  ko: "ko",
};

export function LanguageMenu({
  locale,
  onLocaleChange,
  labels,
  className = "",
  appearance = "workbench",
  disabled = false,
  positionerClassName,
}: {
  locale: AppLocale;
  onLocaleChange: (locale: AppLocale) => void | boolean;
  labels: {
    current: string;
    heading: string;
    label: string;
    description?: string;
  };
  className?: string;
  appearance?: "public" | "workbench";
  disabled?: boolean;
  positionerClassName?: string;
}) {
  const isPublic = appearance === "public";
  const [open, setOpen] = useState(false);
  const displayNames = useMemo(() => {
    try {
      return new Intl.DisplayNames([locale], { type: "language" });
    } catch {
      return null;
    }
  }, [locale]);
  const switchLocale = (nextLocale: AppLocale) => {
    if (nextLocale === locale || onLocaleChange(nextLocale) !== false)
      setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        disabled={disabled}
        aria-label={labels.current}
        className={`inline-flex h-9 shrink-0 items-center gap-2 border bg-background px-3 text-sm text-foreground ui-interactive ui-focus-ring hover:bg-muted ${isPublic ? "rounded-md border-border font-medium" : "rounded-full border-accent/70 font-semibold shadow-sm"} ${className}`}
      >
        <Globe2 className="h-4 w-4" aria-hidden />
        <span lang={DISPLAY_LANGUAGE_TAGS[locale]}>
          {isPublic ? LANGUAGE_NATIVE_NAMES[locale] : locale.toUpperCase()}
        </span>
        <ChevronDown
          className={`h-3.5 w-3.5 text-muted-foreground transition-transform ${open ? "rotate-180" : ""}`}
          aria-hidden
        />
      </PopoverTrigger>
      <PopoverContent
        align="end"
        sideOffset={10}
        positionerClassName={positionerClassName}
        className={`${isPublic ? "public-site w-64" : "w-80"} max-w-[calc(100vw-2rem)] gap-0 overflow-hidden p-0`}
      >
        <div
          className={
            isPublic
              ? "px-5 py-3 text-xs font-medium text-muted-foreground"
              : "px-5 pb-2 pt-5 text-xs font-semibold tracking-widest text-muted-foreground"
          }
        >
          {labels.heading}
        </div>
        <div
          className={`max-h-[min(28rem,70vh)] overflow-y-auto px-2 pb-2 ${isPublic ? "pt-1" : ""}`}
          role="radiogroup"
          aria-label={labels.label}
        >
          {APP_LOCALES.map((option) => {
            const active = option === locale;
            const gloss =
              displayNames?.of(DISPLAY_LANGUAGE_TAGS[option]) ??
              LANGUAGE_NATIVE_NAMES[option];
            return (
              <button
                key={option}
                type="button"
                role="radio"
                aria-checked={active}
                onClick={() => switchLocale(option)}
                className={`relative flex w-full items-center gap-3 px-3 text-left ui-interactive ui-focus-ring ${
                  isPublic
                    ? `min-h-11 rounded-md py-3 text-foreground ${active ? "bg-muted" : "hover:bg-muted/70"}`
                    : `rounded-lg border-l-2 py-2.5 ${active ? "border-accent bg-muted text-foreground" : "border-transparent text-foreground hover:bg-muted/70"}`
                }`}
              >
                <span className="min-w-0 flex-1">
                  <span
                    lang={DISPLAY_LANGUAGE_TAGS[option]}
                    className={`block text-sm ${isPublic ? "font-medium" : "font-semibold"}`}
                  >
                    {LANGUAGE_NATIVE_NAMES[option]}
                  </span>
                  {!isPublic && (
                    <span className="block truncate text-xs text-muted-foreground">
                      {gloss}
                    </span>
                  )}
                </span>
                {!isPublic && (
                  <span className="text-micro font-semibold uppercase text-muted-foreground">
                    {option}
                  </span>
                )}
                {active ? (
                  <Check className="h-4 w-4 text-accent" aria-hidden />
                ) : null}
              </button>
            );
          })}
        </div>
        {!isPublic && (
          <p className="border-t border-border px-5 py-3 text-xs leading-relaxed text-muted-foreground">
            {labels.description}
          </p>
        )}
      </PopoverContent>
    </Popover>
  );
}
