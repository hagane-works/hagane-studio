"use client";

import * as React from "react";
import { useTranslations } from "use-intl";

import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";
import { useProjectStore } from "@/lib/store/projectStore";
import { cn } from "@/lib/utils";

import {
  HOI4_SUPPORTED_LANGS,
  LANG_DISPLAY,
  availableLangsToAdd,
  isLegacyFallback,
  localisationValueStatus,
  orderLanguages,
  resolveActiveValue,
} from "./displayNameLocsHelpers";

const LANG_FULL_NAMES = HOI4_SUPPORTED_LANGS;

export interface DescriptionLocsTabsProps {
  entityKey: React.Key;

  value: Record<string, string>;

  exportLanguages: string[];

  onEnableLanguage?: (locale: ContentLocale) => void | Promise<void>;

  onChange: (next: Record<string, string>) => void;

  legacyFallback?: { en?: string; zh?: string };

  inputComponent?: "input" | "textarea";
  placeholder?: string;

  className?: string;

  ariaLabel?: string;

  rows?: number;

  valueSource?: "imported" | "saved";

  statusMessages?: {
    imported: string;
    saved: string;
    legacy: string;
    missing: string;
  };
}

export const PreferredLanguage = React.createContext<string | null>(null);

function DescriptionLocsTabsForEntity({
  value,
  exportLanguages,
  onEnableLanguage,
  onChange,
  legacyFallback,
  inputComponent = "textarea",
  placeholder,
  className,
  ariaLabel,
  rows = 4,
  valueSource = "saved",
  statusMessages,
}: DescriptionLocsTabsProps) {
  const t = useTranslations("descriptionLocsTabs");
  const setContentLanguageEnabled = useProjectStore(
    (state) => state.setContentLanguageEnabled,
  );
  const valueRef = React.useRef(value);
  const onChangeRef = React.useRef(onChange);
  const mountedRef = React.useRef(false);
  valueRef.current = value;
  onChangeRef.current = onChange;

  React.useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const tabs = React.useMemo(
    () => orderLanguages(exportLanguages, Object.keys(value || {})),
    [exportLanguages, value],
  );

  const preferred = React.useContext(PreferredLanguage);
  const [activeLang, setActiveLang] = React.useState<string>(
    preferred && tabs.includes(preferred) ? preferred : (tabs[0] ?? "en"),
  );
  React.useEffect(() => {
    if (!tabs.includes(activeLang)) {
      setActiveLang(tabs[0] ?? "en");
    }
  }, [tabs, activeLang]);

  const [addOpen, setAddOpen] = React.useState(false);
  const [addPending, setAddPending] = React.useState<ContentLocale | null>(
    null,
  );
  const [addErrorLang, setAddErrorLang] = React.useState<ContentLocale | null>(
    null,
  );
  const addPendingRef = React.useRef(false);
  const tabIdBase = React.useId();
  const tabRefs = React.useRef(new Map<string, HTMLButtonElement>());
  const addable = React.useMemo(
    () => availableLangsToAdd(tabs, HOI4_SUPPORTED_LANGS),
    [tabs],
  );

  const activeValue = resolveActiveValue(value, activeLang, legacyFallback);
  const showingLegacy = isLegacyFallback(value, activeLang, legacyFallback);
  const statusFor = (lang: string) =>
    localisationValueStatus(value, lang, legacyFallback, valueSource);
  const activeStatus = statusFor(activeLang);

  const handleInputChange = (next: string) => {
    const nextDict = { ...(value || {}), [activeLang]: next };
    valueRef.current = nextDict;
    onChange(nextDict);
  };

  const handleAddOpenChange = (open: boolean) => {
    if (!open && addPendingRef.current) return;
    setAddOpen(open);
    if (!open) setAddErrorLang(null);
  };

  const handleAddLang = async (lang: ContentLocale) => {
    if (addPendingRef.current) return;
    addPendingRef.current = true;
    setAddPending(lang);
    setAddErrorLang(null);
    try {
      if (onEnableLanguage) await onEnableLanguage(lang);
      else await setContentLanguageEnabled(lang, true);

      if (!mountedRef.current) return;

      const nextDict = { ...(valueRef.current || {}), [lang]: "" };
      valueRef.current = nextDict;
      onChangeRef.current(nextDict);
      setActiveLang(lang);
      setAddOpen(false);
    } catch {
      if (!mountedRef.current) return;
      setAddErrorLang(lang);
    } finally {
      addPendingRef.current = false;
      if (mountedRef.current) setAddPending(null);
    }
  };

  const handleTabKeyDown = (
    event: React.KeyboardEvent<HTMLButtonElement>,
    lang: string,
  ) => {
    const currentIndex = tabs.indexOf(lang);
    if (currentIndex < 0 || tabs.length === 0) return;

    let nextIndex: number | null = null;
    if (event.key === "ArrowRight")
      nextIndex = (currentIndex + 1) % tabs.length;
    else if (event.key === "ArrowLeft")
      nextIndex = (currentIndex - 1 + tabs.length) % tabs.length;
    else if (event.key === "Home") nextIndex = 0;
    else if (event.key === "End") nextIndex = tabs.length - 1;
    if (nextIndex === null) return;

    event.preventDefault();
    const nextLang = tabs[nextIndex];
    setActiveLang(nextLang);
    tabRefs.current.get(nextLang)?.focus();
  };

  const InputEl = inputComponent === "textarea" ? Textarea : Input;
  const panelId = `${tabIdBase}-panel`;

  return (
    <div className={cn("space-y-1", className)} aria-label={ariaLabel}>
      <div
        role="tablist"
        aria-label={t("tablistAriaLabel")}
        className="flex flex-wrap items-center gap-1"
      >
        {tabs.map((lang) => {
          const isActive = lang === activeLang;
          const fullName = LANG_FULL_NAMES.includes(lang as never)
            ? t(`fullName.${lang}`)
            : lang.toUpperCase();
          const status = statusFor(lang);
          const statusLabel = statusMessages?.[status];
          return (
            <button
              key={lang}
              type="button"
              role="tab"
              id={`${tabIdBase}-tab-${lang}`}
              aria-controls={panelId}
              aria-selected={isActive}
              tabIndex={isActive ? 0 : -1}
              title={statusLabel ? `${fullName} · ${statusLabel}` : fullName}
              onClick={() => setActiveLang(lang)}
              onKeyDown={(event) => handleTabKeyDown(event, lang)}
              ref={(node) => {
                if (node) tabRefs.current.set(lang, node);
                else tabRefs.current.delete(lang);
              }}
              className={cn(
                "ui-interactive ui-focus-ring px-2 py-0.5 text-xs rounded-md border transition-colors",
                isActive
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-input bg-background hover:bg-muted text-muted-foreground",
              )}
            >
              <span
                aria-hidden="true"
                className={cn(
                  "mr-1 inline-block size-1.5 rounded-full align-middle",
                  status === "imported" || status === "saved"
                    ? "bg-success"
                    : status === "legacy"
                      ? "bg-muted-foreground/70"
                      : "bg-muted-foreground/40",
                )}
              />
              {LANG_DISPLAY[lang] ?? lang.toUpperCase()}
            </button>
          );
        })}
        {addable.length > 0 && (
          <Popover open={addOpen} onOpenChange={handleAddOpenChange}>
            <PopoverTrigger
              render={(props) => (
                <Button
                  {...props}
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-6 px-2 text-xs text-muted-foreground"
                >
                  + {t("addLanguage")}
                </Button>
              )}
            />
            <PopoverContent className="w-44 p-1" align="start">
              <ul className="flex flex-col" role="listbox">
                {addable.map((lang) => {
                  const fullName = LANG_FULL_NAMES.includes(lang as never)
                    ? t(`fullName.${lang}`)
                    : lang.toUpperCase();
                  return (
                    <li key={lang}>
                      <button
                        type="button"
                        onClick={() => handleAddLang(lang as ContentLocale)}
                        disabled={addPending !== null}
                        aria-busy={addPending === lang || undefined}
                        className="ui-interactive ui-focus-ring flex w-full items-center justify-between gap-2 rounded px-2 py-1 text-left text-xs hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        <span className="font-mono w-6 text-muted-foreground">
                          {LANG_DISPLAY[lang] ?? lang.toUpperCase()}
                        </span>
                        <span className="flex-1 truncate">
                          {addPending === lang
                            ? t("addingLanguage", { language: fullName })
                            : fullName}
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
              {addErrorLang && (
                <p role="alert" className="px-2 py-1 text-xs text-destructive">
                  {t("addLanguageError", {
                    language: t(`fullName.${addErrorLang}`),
                  })}
                </p>
              )}
            </PopoverContent>
          </Popover>
        )}
      </div>

      <div
        id={panelId}
        role="tabpanel"
        aria-labelledby={`${tabIdBase}-tab-${activeLang}`}
      >
        <InputEl
          value={activeValue}
          onChange={(
            e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
          ) => handleInputChange(e.target.value)}
          aria-label={ariaLabel}
          placeholder={placeholder}
          rows={inputComponent === "textarea" ? rows : undefined}
          className={
            inputComponent === "textarea" ? "min-h-16 text-xs" : undefined
          }
        />
      </div>

      {statusMessages && (
        <p
          className={cn(
            "text-micro",
            activeStatus === "missing"
              ? "text-muted-foreground"
              : activeStatus === "legacy"
                ? "text-muted-foreground"
                : "text-success",
          )}
        >
          {statusMessages[activeStatus]}
        </p>
      )}
      {!statusMessages && showingLegacy && (
        <p className="text-micro text-muted-foreground">
          {t("legacyFallbackHint")}
        </p>
      )}
    </div>
  );
}

export function DescriptionLocsTabs(props: DescriptionLocsTabsProps) {
  return <DescriptionLocsTabsForEntity key={props.entityKey} {...props} />;
}
