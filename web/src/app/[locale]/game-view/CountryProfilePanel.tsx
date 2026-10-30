"use client";

import { useMemo, useRef, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { readGuiResponse } from "./guiResponse";
import { Button } from "@/components/ui/button";

import { firstBaseModDepKey } from "@/components/workbench/BaseModFlagPreview";
import {
  CountryProfileEditor,
  type CountryProfileRows,
  type CountryProfileWrite,
} from "@/components/workbench/country-profile/CountryProfileEditor";
import {
  countriesApi,
  historyFilesApi,
  type CountryRow,
  type HistoryFileRow,
} from "@/lib/api";
import { useIdeologyOptions } from "@/lib/ideology/useIdeologyOptions";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { useProjectStore } from "@/lib/store/projectStore";

import { GamePanel } from "./GamePanel";
import { pick, recordChange, rowUndo } from "./changeLog";
import type { GameCountryProfile } from "./countryProfileEdit";
import { takeInCountry } from "./countryTakeIn";
import { useWorkbenchProject } from "./workbenchProject";
import { withGameLanguage } from "@/lib/guiRenderApi";

export function useCountryProfile(
  projectId: number,
  tag: string | null,
  date: string | null | undefined,
) {
  return useSWR<GameCountryProfile>(
    tag ? ["game-view-country-profile", projectId, tag, date ?? ""] : null,
    async () => {
      const query = new URLSearchParams({ tag: tag as string });
      if (date) query.set("date", date);
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/country-profile?${query}`,
        ),
      );
      return readGuiResponse<GameCountryProfile>(response);
    },
    { revalidateOnFocus: false },
  );
}

interface ProfileUndoWrite {
  country?: Partial<CountryRow>;
  capital?: number;
}

interface PendingProfileChange {
  countryId: number;
  historyId?: number;
  countryKeys: string[];
  before: ProfileUndoWrite;
  after: ProfileUndoWrite;
  createdHistory: boolean;
}

function profileUndo(
  projectId: number,
  countryId: number,
  historyId: number | undefined,
  countryKeys: readonly string[],
  before: ProfileUndoWrite,
  after: ProfileUndoWrite,
) {
  return rowUndo<ProfileUndoWrite>({
    before,
    after,
    load: async () => {
      const [list, historyNow] = await Promise.all([
        after.country ? countriesApi.list(projectId) : null,
        after.capital !== undefined && historyId !== undefined
          ? historyFilesApi.get(projectId, historyId)
          : null,
      ]);
      const now: ProfileUndoWrite = {};
      if (after.country) {
        const country = list?.countries.find((each) => each.id === countryId);
        if (!country) return null;
        now.country = pick(country, countryKeys);
      }
      if (after.capital !== undefined) {
        if (!historyNow) return null;
        now.capital = historyNow.capital;
      }
      return now;
    },
    save: async (back) => {
      if (back.country) {
        await countriesApi.update(projectId, countryId, back.country);
      }
      if (back.capital !== undefined && historyId !== undefined) {
        await historyFilesApi.update(projectId, historyId, {
          capital: back.capital,
        });
      }
    },
  });
}

export function CountryProfilePanel({
  projectId,
  tag,
  date,
  onSaved,
  onClose,
}: {
  projectId: number;
  tag: string;
  date?: string | null;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewEdit.country");
  const game = useCountryProfile(projectId, tag, date);
  const countries = useSWR(
    ["game-view-country-rows", projectId],
    () => countriesApi.list(projectId),
    { revalidateOnFocus: false },
  );
  const projectReady = useWorkbenchProject(projectId);
  const project = useProjectStore((state) =>
    state.currentProject?.id === projectId ? state.currentProject : null,
  );
  const contentLocale = useContentLocaleStore((state) => state.locale);
  const { groupOptions } = useIdeologyOptions();
  const baseModDepKey = useMemo(
    () => firstBaseModDepKey(project?.dependencies),
    [project?.dependencies],
  );
  const effectiveProfile = useMemo(() => {
    if (!game.data) return null;
    return {
      name: game.data.name,
      adjective: game.data.adjective,
      ideologyNames: game.data.names,
      graphicalCulture: game.data.graphical_culture,
      capital: game.data.capital?.id,
      ruling: game.data.ruling,
      flag: game.data.flag,
    };
  }, [game.data]);
  const row =
    countries.data?.countries.find(
      (country) => country.tag.toUpperCase() === tag.toUpperCase(),
    ) ?? null;
  const pending = useRef<PendingProfileChange | null>(null);
  const workbenchHref = `/project/${projectId}/country-profile`;

  const [takingIn, setTakingIn] = useState(false);
  const [takeInFailed, setTakeInFailed] = useState(false);
  const startEditing = async () => {
    setTakingIn(true);
    setTakeInFailed(false);
    const inProject = await takeInCountry(projectId, tag);
    if (inProject) await countries.mutate();
    else setTakeInFailed(true);
    setTakingIn(false);
  };

  const note = (undo?: () => Promise<void>) =>
    recordChange({
      projectId,
      title: game.data?.shown || row?.name_en || row?.name || tag,
      kind: t("kind", { tag }),
      workbenchHref,
      undo,
    });

  const save = async (
    write: CountryProfileWrite,
    rows: CountryProfileRows,
  ): Promise<HistoryFileRow | null> => {
    const countryKeys = Object.keys(write.country);
    const before: ProfileUndoWrite = {};
    const after: ProfileUndoWrite = {};
    if (countryKeys.length > 0) {
      const countryPatch = pick(write.country, countryKeys);
      before.country = pick(rows.country, countryKeys);
      const countrySaved = await countriesApi.update(
        projectId,
        rows.country.id,
        countryPatch,
      );
      after.country = pick(countrySaved, countryKeys);
    }

    let historySaved = rows.history;
    let createdHistory = false;
    try {
      if (write.capital !== undefined) {
        if (rows.history) {
          before.capital = rows.history.capital;
          historySaved = await historyFilesApi.update(
            projectId,
            rows.history.id,
            { capital: write.capital },
          );
          after.capital = historySaved.capital;
        } else {
          historySaved = await historyFilesApi.create(projectId, {
            country_tag: rows.country.tag,
            capital: write.capital,
          });
          createdHistory = true;
        }
      }
    } catch (cause) {
      if (after.country) {
        note(
          profileUndo(
            projectId,
            rows.country.id,
            undefined,
            countryKeys,
            before,
            after,
          ),
        );
        void countries.mutate();
        void game.mutate();
      }
      pending.current = null;
      throw cause;
    }

    pending.current = {
      countryId: rows.country.id,
      historyId: historySaved?.id,
      countryKeys,
      before,
      after,
      createdHistory,
    };
    return historySaved;
  };

  const profileSaved = () => {
    const change = pending.current;
    pending.current = null;
    if (change) {
      const canUndo =
        !change.createdHistory &&
        (Boolean(change.after.country) || change.after.capital !== undefined);
      note(
        canUndo
          ? profileUndo(
              projectId,
              change.countryId,
              change.historyId,
              change.countryKeys,
              change.before,
              change.after,
            )
          : undefined,
      );
    }
    onSaved(t("saved", { tag }));
  };

  const refreshCountry = () => {
    void countries.mutate();
    void game.mutate();
  };

  const flagUploaded = (root: string | undefined) => {
    note();
    onSaved(
      root
        ? t("flagSavedFor", {
            tag,
            ideology:
              groupOptions.find((option) => option.value === root)?.label ??
              root,
          })
        : t("flagSaved", { tag }),
    );
  };

  const loading = countries.isLoading || game.isLoading || !projectReady;
  const loadError = countries.error ?? game.error;

  return (
    <GamePanel
      title={game.data?.shown || row?.name_en || row?.name || tag}
      kind={t("kind", { tag })}
      source={loading ? undefined : row ? "项目" : "原版"}
      workbenchHref={workbenchHref}
      wide={row ? "workbench" : true}
      onClose={onClose}
    >
      {loading ? (
        <p className="mt-4 text-micro text-muted-foreground">{t("loading")}</p>
      ) : loadError ? (
        <p role="alert" className="mt-4 text-xs text-destructive">
          {loadError instanceof Error ? loadError.message : String(loadError)}
        </p>
      ) : !row ? (
        <div className="mt-4 space-y-2">
          <p className="text-xs text-muted-foreground">{t("takeInHint")}</p>
          <Button
            size="sm"
            onClick={() => void startEditing()}
            disabled={takingIn}
          >
            {takingIn ? t("takingIn") : t("takeIn")}
          </Button>
          {takeInFailed && (
            <p role="alert" className="text-xs text-destructive">
              {t("takeInFailed")}
            </p>
          )}
        </div>
      ) : (
        <div className="-mx-4 mt-2">
          <CountryProfileEditor
            key={row.id}
            projectId={projectId}
            country={row}
            roots={groupOptions}
            exportLanguages={project?.export_languages ?? ["en"]}
            contentLocale={contentLocale}
            baseModDepKey={baseModDepKey}
            effectiveProfile={effectiveProfile}
            onSave={save}
            onCountryReadback={refreshCountry}
            onProfileSaved={profileSaved}
            onFlagUploaded={flagUploaded}
            embedded
          />
        </div>
      )}
    </GamePanel>
  );
}
