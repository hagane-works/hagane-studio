"use client";

import { useTranslations } from "use-intl";
import { useProjectStore, useProjectCountries } from "@/lib/store/projectStore";

interface AllCountriesToggleProps {
  className?: string;
}

export function AllCountriesToggle({ className }: AllCountriesToggleProps) {
  const countries = useProjectCountries();
  const viewMode = useProjectStore((s) => s.viewMode);
  const toggleViewMode = useProjectStore((s) => s.toggleViewMode);
  const t = useTranslations("allCountriesToggle");

  if (countries.length <= 1) return null;

  const isAll = viewMode === "all";
  const label = isAll ? t("allCountries") : t("activeCountry");
  const next = isAll ? t("switchToActive") : t("switchToAll");

  return (
    <button
      type="button"
      onClick={toggleViewMode}
      title={next}
      aria-pressed={isAll}
      className={
        "w-full flex items-center justify-between gap-2 px-2 h-8 rounded-md " +
        "border border-border text-xs ui-interactive " +
        (isAll
          ? "bg-accent/15 border-accent/60 text-accent-foreground"
          : "bg-transparent text-muted-foreground hover:text-foreground hover:bg-muted/50") +
        (className ? ` ${className}` : "")
      }
    >
      <span className="flex items-center gap-1.5">
        <span aria-hidden className="font-mono leading-none">
          {isAll ? "⊞" : "⊟"}
        </span>
        <span className="truncate">{label}</span>
      </span>
      <span className="text-micro font-mono text-muted-foreground">
        {isAll ? countries.length : 1}/{countries.length}
      </span>
    </button>
  );
}
