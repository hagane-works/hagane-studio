"use client";

import { useTranslations } from "use-intl";
import useSWR from "swr";

import { countriesApi } from "@/lib/api";
import { useIdeologyOptions } from "@/lib/ideology/useIdeologyOptions";
import { useActiveCountry } from "@/lib/store/projectStore";

import { CountryPoliticsWorkbench } from "./CountryPoliticsWorkbench";

export interface CountryPoliticsPageProps {
  projectId: number;

  countryTag?: string;
  onChanged?: () => void | Promise<void>;
  onOpenIdeologies?: () => void;
}

export function CountryPoliticsPage({
  projectId,
  countryTag,
  onChanged,
  onOpenIdeologies,
}: CountryPoliticsPageProps) {
  const t = useTranslations("countrySettingsDialog.workbench.shared");
  const tDialog = useTranslations("countrySettingsDialog");
  const activeCountryTag = useActiveCountry();
  const effectiveCountryTag = countryTag || activeCountryTag;
  const { groupOptions, subOptions } = useIdeologyOptions();
  const { data, error } = useSWR(
    effectiveCountryTag
      ? ["country-politics-current-country", projectId, effectiveCountryTag]
      : null,
    async () => {
      const response = await countriesApi.list(projectId);
      const selected = response.countries.find(
        (item) => item.tag.toUpperCase() === effectiveCountryTag.toUpperCase(),
      );
      if (!selected)
        throw new Error(t("countryNotFound", { tag: effectiveCountryTag }));
      return selected;
    },
    { keepPreviousData: false, revalidateOnFocus: false },
  );

  if (!effectiveCountryTag) {
    return (
      <div className="ui-panel p-6 text-sm text-muted-foreground">
        {tDialog("selectCountry")}
      </div>
    );
  }
  if (error) {
    return (
      <div role="alert" className="ui-panel p-6 text-sm text-destructive">
        {error instanceof Error ? error.message : String(error)}
      </div>
    );
  }
  if (!data)
    return (
      <div className="animate-pulse-ring ui-panel p-6 text-sm">
        {t("loadingCountry")}
      </div>
    );
  return (
    <CountryPoliticsWorkbench
      key={`${projectId}-${data.id}`}
      projectId={projectId}
      country={data}
      roots={groupOptions}
      subtypes={subOptions}
      onChanged={onChanged}
      onOpenIdeologies={onOpenIdeologies}
    />
  );
}
