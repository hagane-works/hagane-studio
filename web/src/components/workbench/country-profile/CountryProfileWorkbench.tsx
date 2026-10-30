"use client";

import { OpenInGameLink } from "@/components/workbench/OpenInGameLink";
import { gameViewHref } from "@/lib/gameViewLinks";
import { countriesApi, historyFilesApi, type CountryRow } from "@/lib/api";
import type { IdeologyOption } from "@/lib/ideology/registry";

import {
  CountryProfileEditor,
  type CountryProfileRows,
  type CountryProfileWrite,
} from "./CountryProfileEditor";

export interface CountryProfileWorkbenchProps {
  projectId: number;
  country: CountryRow;
  roots: readonly IdeologyOption[];
  exportLanguages: string[];
  contentLocale: string;
  baseModDepKey: string | null;
  onCountryReadback?: (country: CountryRow) => void;
  onOpenPolitics?: () => void;
}

export function CountryProfileWorkbench({
  projectId,
  country,
  roots,
  exportLanguages,
  contentLocale,
  baseModDepKey,
  onCountryReadback,
  onOpenPolitics,
}: CountryProfileWorkbenchProps) {
  const save = async (
    { country: data, capital }: CountryProfileWrite,
    rows: CountryProfileRows,
  ) => {
    if (Object.keys(data).length > 0) {
      await countriesApi.update(projectId, rows.country.id, data);
    }
    if (capital === undefined) return rows.history;
    return rows.history
      ? historyFilesApi.update(projectId, rows.history.id, { capital })
      : historyFilesApi.create(projectId, {
          country_tag: rows.country.tag,
          capital,
        });
  };

  return (
    <CountryProfileEditor
      projectId={projectId}
      country={country}
      roots={roots}
      exportLanguages={exportLanguages}
      contentLocale={contentLocale}
      baseModDepKey={baseModDepKey}
      onSave={save}
      onCountryReadback={onCountryReadback}
      onOpenPolitics={onOpenPolitics}
      headerAction={
        <OpenInGameLink
          href={gameViewHref(projectId, {
            panel: "countrypoliticsview",
            tag: country.tag,
          })}
        />
      }
    />
  );
}
