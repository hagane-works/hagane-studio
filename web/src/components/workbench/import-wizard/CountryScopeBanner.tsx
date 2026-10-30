import type { Dispatch, SetStateAction } from "react";
import { useTranslations } from "use-intl";
import { ChevronDown, ChevronRight, Search } from "lucide-react";
import { Input } from "@/components/ui/input";

export interface CountryScopeBannerProps {
  isMultiCountry: boolean;
  t: ReturnType<typeof useTranslations>;
  tag: string;
  countryListOpen: boolean;
  setCountryListOpen: Dispatch<SetStateAction<boolean>>;
  projectCountries: string[];
  countryFilter: string;
  setCountryFilter: Dispatch<SetStateAction<string>>;
}

export function CountryScopeBanner({
  isMultiCountry,
  t,
  tag,
  countryListOpen,
  setCountryListOpen,
  projectCountries,
  countryFilter,
  setCountryFilter,
}: CountryScopeBannerProps) {
  return (
    <div
      className={
        isMultiCountry
          ? "rounded-md border border-amber-500/60 bg-amber-500/10 px-3 py-2 text-sm"
          : "rounded-md border border-border bg-muted/40 px-3 py-1.5 text-xs text-muted-foreground"
      }
    >
      {isMultiCountry ? (
        <>
          <div className="flex items-center gap-2">
            <span className="text-amber-500 font-semibold">⚠</span>
            <span>
              {t.rich("tagBannerImporting", {
                tag,
                mono: (chunks) => (
                  <span className="font-mono font-semibold text-amber-500">
                    {chunks}
                  </span>
                ),
              })}{" "}
              {t("tagBannerSwitch")}
            </span>
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            <button
              type="button"
              onClick={() => setCountryListOpen((v) => !v)}
              className="inline-flex items-center gap-1 hover:text-foreground"
              aria-expanded={countryListOpen}
            >
              {countryListOpen ? (
                <ChevronDown className="h-3 w-3 shrink-0" />
              ) : (
                <ChevronRight className="h-3 w-3 shrink-0" />
              )}
              <span>
                {t("tagBannerMultiCountry", { count: projectCountries.length })}
              </span>
            </button>
            {countryListOpen && (
              <div className="mt-1.5">
                {projectCountries.length > 8 && (
                  <div className="mb-1.5 flex items-center gap-1 rounded border border-border bg-background/60 px-1.5 py-0.5">
                    <Search className="h-3 w-3 shrink-0 text-muted-foreground" />
                    <Input
                      type="text"
                      value={countryFilter}
                      onChange={(e) => setCountryFilter(e.target.value)}
                      placeholder={t("tagBannerSearch")}
                      className="bg-transparent text-xs placeholder:text-muted-foreground"
                    />
                  </div>
                )}
                <div className="max-h-40 overflow-y-auto">
                  <div className="flex flex-wrap gap-1">
                    {projectCountries
                      .filter((c) =>
                        c
                          .toUpperCase()
                          .includes(countryFilter.trim().toUpperCase()),
                      )
                      .map((c) => (
                        <span
                          key={c}
                          className={
                            "rounded px-1.5 py-0.5 font-mono " +
                            (c === tag
                              ? "bg-amber-500/20 font-semibold text-amber-600 dark:text-amber-400"
                              : "bg-muted")
                          }
                        >
                          {c}
                        </span>
                      ))}
                  </div>
                </div>
              </div>
            )}
          </div>
        </>
      ) : (
        <>
          {t.rich("tagBannerActiveCountry", {
            tag,
            mono: (chunks) => (
              <span className="font-mono font-semibold text-foreground">
                {chunks}
              </span>
            ),
          })}
        </>
      )}
    </div>
  );
}
