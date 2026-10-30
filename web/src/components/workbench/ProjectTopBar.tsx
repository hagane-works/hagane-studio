"use client";

import { useTranslations } from "use-intl";
import useSWR from "swr";
import {
  useActiveCountry,
  useProjectCountries,
} from "@/lib/store/projectStore";
import { useVanillaTagName } from "@/lib/vanillaCountryTags";
import {
  CountryFlag,
  isResolvedCountryName,
} from "@/components/workbench/CountryFlag";
import { countriesApi } from "@/lib/api";

interface ProjectTopBarProps {
  projectId: number;
}

export function ProjectTopBar({ projectId }: ProjectTopBarProps) {
  const countries = useProjectCountries();
  const activeCountry = useActiveCountry();
  const t = useTranslations("countrySwitcher");
  const vanillaName = useVanillaTagName();

  const { data: grouped } = useSWR(
    projectId ? ["countries/grouped", projectId] : null,
    () => countriesApi.grouped(projectId),
  );

  if (countries.length <= 1) return null;

  const item = [
    ...(grouped?.authored ?? []),
    ...(grouped?.reference ?? []),
  ].find((c) => c.tag === activeCountry);

  const display =
    vanillaName(activeCountry) ??
    (item?.name && isResolvedCountryName(item.name, activeCountry)
      ? item.name
      : "");

  const flag = <CountryFlag color={item?.color} />;

  const openPalette = () => {
    if (typeof window === "undefined") return;
    window.dispatchEvent(new CustomEvent("hoi4-open-country-palette"));
  };

  return (
    <div className="border-b border-border bg-card/40 px-4 py-1.5 flex items-center gap-3 shrink-0">
      <span className="text-micro uppercase tracking-widest text-muted-foreground font-semibold">
        {t("activeCountryLabel")}
      </span>
      <div
        aria-label={`${t("activeCountryLabel")}: ${activeCountry}${display ? ` ${display}` : ""}`}
        className="flex min-w-0 items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1"
      >
        {flag}
        <span className="font-mono font-semibold text-xs">
          {activeCountry || "—"}
        </span>
        {display && (
          <span className="text-xs text-muted-foreground truncate max-w-[14rem]">
            {display}
          </span>
        )}
      </div>
      <button
        type="button"
        onClick={openPalette}
        className="ui-interactive ml-auto flex items-center gap-1 text-micro text-muted-foreground hover:text-foreground"
        title={t("openPalette")}
      >
        <kbd className="rounded border border-border bg-muted px-1 py-0.5 font-mono">
          ⌘K
        </kbd>
        <span>{t("shortcutHint")}</span>
      </button>
    </div>
  );
}
