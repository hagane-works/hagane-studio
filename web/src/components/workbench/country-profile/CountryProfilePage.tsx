"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";

import { firstBaseModDepKey } from "@/components/workbench/BaseModFlagPreview";
import { countriesApi } from "@/lib/api";
import { useIdeologyOptions } from "@/lib/ideology/useIdeologyOptions";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { useActiveCountry, useProjectStore } from "@/lib/store/projectStore";

import { CountryProfileWorkbench } from "./CountryProfileWorkbench";

export interface CountryProfilePageProps {
  projectId: number;

  countryTag?: string;
  onChanged?: () => void | Promise<void>;
  onOpenPolitics?: () => void;
}

export function CountryProfilePage({
  projectId,
  countryTag,
  onChanged,
  onOpenPolitics,
}: CountryProfilePageProps) {
  const t = useTranslations("countrySettingsDialog.workbench.shared");
  const tDialog = useTranslations("countrySettingsDialog");
  const project = useProjectStore((state) => state.currentProject);
  const activeCountryTag = useActiveCountry();
  const effectiveCountryTag = countryTag || activeCountryTag;
  const contentLocale = useContentLocaleStore((state) => state.locale);
  const { groupOptions } = useIdeologyOptions();
  const baseModDepKey = useMemo(
    () => firstBaseModDepKey(project?.dependencies),
    [project?.dependencies],
  );

  const { data, error } = useSWR(
    effectiveCountryTag
      ? ["country-profile-current-country", projectId, effectiveCountryTag]
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
    <CountryProfileWorkbench
      key={`${projectId}-${data.id}`}
      projectId={projectId}
      country={data}
      roots={groupOptions}
      exportLanguages={project?.export_languages ?? ["en"]}
      contentLocale={contentLocale}
      baseModDepKey={baseModDepKey}
      onCountryReadback={() => {
        void onChanged?.();
      }}
      onOpenPolitics={onOpenPolitics}
    />
  );
}
