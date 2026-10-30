"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Check, Languages, Plus } from "lucide-react";
import { useTranslations } from "use-intl";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useProjectStore } from "@/lib/store/projectStore";
import {
  CONTENT_LOCALES,
  type ContentLocale,
  normalizeContentLocales,
  useContentLocaleStore,
} from "@/lib/store/contentLocaleStore";

const NATIVE_NAMES: Record<ContentLocale, string> = {
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

export function ContentLanguageSwitcher() {
  const t = useTranslations("layout");
  const [open, setOpen] = useState(false);
  const [busyLocale, setBusyLocale] = useState<ContentLocale | null>(null);
  const [error, setError] = useState<string | null>(null);
  const projectEpochRef = useRef(0);
  const locale = useContentLocaleStore((state) => state.locale);
  const setLocale = useContentLocaleStore((state) => state.setLocale);
  const exportLanguages = useProjectStore(
    (state) => state.currentProject?.export_languages,
  );
  const projectId = useProjectStore((state) => state.currentProject?.id);
  const observedProjectIdRef = useRef(projectId);
  const setContentLanguageEnabled = useProjectStore(
    (state) => state.setContentLanguageEnabled,
  );
  const available = useMemo(() => {
    return normalizeContentLocales(exportLanguages);
  }, [exportLanguages]);
  const remaining = useMemo(
    () => CONTENT_LOCALES.filter((candidate) => !available.includes(candidate)),
    [available],
  );

  useEffect(() => {
    if (!available.includes(locale)) setLocale("en");
  }, [available, locale, setLocale]);

  useEffect(() => {
    if (observedProjectIdRef.current === projectId) return;
    observedProjectIdRef.current = projectId;

    projectEpochRef.current += 1;
    setLocale("en");
    const epoch = projectEpochRef.current;
    let cancelled = false;
    void Promise.resolve().then(() => {
      if (cancelled || projectEpochRef.current !== epoch) return;
      setOpen(false);
      setBusyLocale(null);
      setError(null);
    });
    return () => {
      cancelled = true;
    };
  }, [projectId, setLocale]);

  const handleAddLanguage = async (candidate: ContentLocale) => {
    const requestProjectId = projectId;
    const requestEpoch = projectEpochRef.current;
    setBusyLocale(candidate);
    setError(null);
    try {
      await setContentLanguageEnabled(candidate, true);
      if (
        projectEpochRef.current !== requestEpoch ||
        useProjectStore.getState().currentProject?.id !== requestProjectId
      ) {
        return;
      }
      setOpen(false);
    } catch (cause) {
      if (
        projectEpochRef.current !== requestEpoch ||
        useProjectStore.getState().currentProject?.id !== requestProjectId
      ) {
        return;
      }
      setError((cause as Error).message || t("contentLanguageUnknownError"));
    } finally {
      if (
        projectEpochRef.current === requestEpoch &&
        useProjectStore.getState().currentProject?.id === requestProjectId
      ) {
        setBusyLocale(null);
      }
    }
  };

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (!nextOpen) setError(null);
      }}
    >
      <PopoverTrigger
        aria-label={t("contentLanguageCurrent", {
          language: NATIVE_NAMES[locale],
        })}
        title={t("contentLanguageHelp")}
        className="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-md border border-border bg-background px-2.5 text-xs font-semibold text-foreground ui-interactive ui-focus-ring hover:bg-muted"
      >
        <Languages className="h-4 w-4" aria-hidden />
        <span>{t("contentLanguageShort")}</span>
        <span className="font-mono text-muted-foreground">
          {locale.toUpperCase()}
        </span>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-72 gap-0 overflow-hidden p-0">
        <div className="px-4 pb-2 pt-4">
          <p className="text-sm font-semibold">{t("contentLanguageTitle")}</p>
          <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
            {t("contentLanguageHelp")}
          </p>
        </div>
        <div className="border-t border-border p-2">
          <p className="px-2.5 pb-1 pt-1 text-xs font-medium text-muted-foreground">
            {t("contentLanguageEnabledLabel")}
          </p>
          <div role="radiogroup" aria-label={t("contentLanguageTitle")}>
            {available.map((option) => (
              <button
                key={option}
                type="button"
                role="radio"
                aria-checked={option === locale}
                onClick={() => {
                  setLocale(option);
                  setOpen(false);
                }}
                className="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-sm hover:bg-muted ui-interactive ui-focus-ring"
              >
                <span className="min-w-0 flex-1 truncate">
                  {NATIVE_NAMES[option]}
                </span>
                <span className="font-mono text-xs text-muted-foreground">
                  {option.toUpperCase()}
                </span>
                {option === locale ? (
                  <Check className="h-4 w-4 text-accent" aria-hidden />
                ) : null}
              </button>
            ))}
          </div>
        </div>
        <div className="border-t border-border p-2">
          <p className="px-2.5 pb-1 pt-1 text-xs font-medium text-muted-foreground">
            {t("contentLanguageAddLabel")}
          </p>
          {remaining.map((option) => (
            <button
              key={option}
              type="button"
              disabled={busyLocale !== null}
              aria-label={t("contentLanguageAddAction", {
                language: NATIVE_NAMES[option],
              })}
              onClick={() => void handleAddLanguage(option)}
              className="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-sm hover:bg-muted ui-interactive ui-focus-ring disabled:cursor-not-allowed disabled:text-muted-foreground"
            >
              <Plus className="h-4 w-4" aria-hidden />
              <span className="min-w-0 flex-1 truncate">
                {busyLocale === option
                  ? t("contentLanguageUpdating", {
                      language: NATIVE_NAMES[option],
                    })
                  : NATIVE_NAMES[option]}
              </span>
              <span className="font-mono text-xs text-muted-foreground">
                {option.toUpperCase()}
              </span>
            </button>
          ))}
          {remaining.length === 0 ? (
            <p className="px-2.5 py-2 text-xs text-muted-foreground">
              {t("contentLanguageAllEnabled")}
            </p>
          ) : null}
        </div>
        {error ? (
          <p
            role="alert"
            className="border-t border-border bg-destructive/10 px-4 py-3 text-xs text-destructive"
          >
            {t("contentLanguageUpdateFailed", { message: error })}
          </p>
        ) : null}
        <p className="border-t border-border px-4 py-3 text-xs text-muted-foreground">
          {t("contentLanguageAddHint")}
        </p>
      </PopoverContent>
    </Popover>
  );
}
