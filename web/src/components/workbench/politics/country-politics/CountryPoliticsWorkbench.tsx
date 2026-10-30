"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslations } from "use-intl";

import {
  charactersApi,
  historyFilesApi,
  type CountryRow,
  type HistoryFileRow,
} from "@/lib/api";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import type { IdeologyOption } from "@/lib/ideology/registry";
import type { Character } from "@/types";

import {
  CountryPoliticsEditor,
  type CountryPoliticsSaving,
} from "./CountryPoliticsEditor";
import {
  countryLeaders,
  historyToPoliticsDraft,
  leaderSubtypeDrafts,
  politicsHistoryPayload,
  politicsReadbackMatches,
  validatePopularities,
  type CountryPoliticsDraft,
} from "./model";

export interface CountryPoliticsWorkbenchProps {
  projectId: number;
  country: CountryRow;
  roots: readonly IdeologyOption[];
  subtypes: readonly IdeologyOption[];
  onChanged?: () => void | Promise<void>;
  onOpenIdeologies?: () => void;
}

export function CountryPoliticsWorkbench({
  projectId,
  country,
  roots,
  subtypes,
  onChanged,
  onOpenIdeologies,
}: CountryPoliticsWorkbenchProps) {
  const t = useTranslations("countrySettingsDialog.workbench.politics");
  const [history, setHistory] = useState<HistoryFileRow | null>(null);
  const [characters, setCharacters] = useState<Character[]>([]);
  const [draft, setDraft] = useState<CountryPoliticsDraft>(() =>
    historyToPoliticsDraft(null, roots),
  );
  const [savedDraft, setSavedDraft] = useState<CountryPoliticsDraft>(() =>
    historyToPoliticsDraft(null, roots),
  );
  const [leaderDrafts, setLeaderDrafts] = useState<Record<number, string>>({});
  const [savedLeaderDrafts, setSavedLeaderDrafts] = useState<
    Record<number, string>
  >({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<CountryPoliticsSaving>(null);
  const [status, setStatus] = useState<"" | "saved">("");
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [histories, allCharacters] = await Promise.all([
        historyFilesApi.list(projectId),
        charactersApi.list(projectId),
      ]);
      const nextHistory =
        histories.find(
          (row) => row.country_tag.toUpperCase() === country.tag.toUpperCase(),
        ) ?? null;
      const leaders = countryLeaders(allCharacters, country.tag);
      const nextDraft = historyToPoliticsDraft(nextHistory, roots);
      const nextLeaderDrafts = leaderSubtypeDrafts(leaders);
      setHistory(nextHistory);
      setCharacters(leaders);
      setDraft(nextDraft);
      setSavedDraft(nextDraft);
      setLeaderDrafts(nextLeaderDrafts);
      setSavedLeaderDrafts(nextLeaderDrafts);
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setLoading(false);
    }
  }, [country.tag, projectId, roots]);

  useEffect(() => {
    void load();
  }, [load]);

  const isDirty = useMemo(
    () =>
      JSON.stringify(draft) !== JSON.stringify(savedDraft) ||
      JSON.stringify(leaderDrafts) !== JSON.stringify(savedLeaderDrafts),
    [draft, leaderDrafts, savedDraft, savedLeaderDrafts],
  );
  useRegisterDirty("country-politics", isDirty);

  const savePolitics = async () => {
    if (!validatePopularities(draft.popularities).valid) return;
    setSaving("politics");
    setError(null);
    setStatus("");
    const payload = politicsHistoryPayload(draft);
    try {
      const saved = history
        ? await historyFilesApi.update(projectId, history.id, payload)
        : await historyFilesApi.create(projectId, {
            country_tag: country.tag,
            ...payload,
          });
      const readback = await historyFilesApi.get(projectId, saved.id);
      if (!politicsReadbackMatches(readback, draft)) {
        throw new Error(t("readbackMismatch"));
      }
      setHistory(readback);
      const nextDraft = historyToPoliticsDraft(readback, roots);
      setDraft(nextDraft);
      setSavedDraft(nextDraft);
      setStatus("saved");
      await onChanged?.();
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setSaving(null);
    }
  };

  const saveLeader = async (leader: Character) => {
    const nextSubtype = leaderDrafts[leader.id] ?? "";
    if (!subtypes.some((option) => option.value === nextSubtype)) {
      setError(t("unknownSubtype"));
      return;
    }
    setSaving(`leader-${leader.id}`);
    setError(null);
    setStatus("");
    try {
      await charactersApi.update(projectId, leader.id, {
        ideology: nextSubtype,
      });
      const readback = await charactersApi.get(projectId, leader.id);
      if (readback.ideology !== nextSubtype)
        throw new Error(t("readbackMismatch"));
      setCharacters((current) =>
        current.map((item) => (item.id === readback.id ? readback : item)),
      );
      setLeaderDrafts((current) => ({
        ...current,
        [readback.id]: readback.ideology,
      }));
      setSavedLeaderDrafts((current) => ({
        ...current,
        [readback.id]: readback.ideology,
      }));
      setStatus("saved");
      await onChanged?.();
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setSaving(null);
    }
  };

  if (loading) {
    return (
      <div className="animate-pulse-ring ui-panel p-6 text-sm">
        {t("loading")}
      </div>
    );
  }

  return (
    <CountryPoliticsEditor
      country={country}
      roots={roots}
      subtypes={subtypes}
      savedRulingParty={history?.set_politics_ruling_party ?? ""}
      draft={draft}
      onEdit={setDraft}
      leaders={characters}
      leaderDrafts={leaderDrafts}
      onEditLeaders={setLeaderDrafts}
      saving={saving}
      onSavePolitics={() => void savePolitics()}
      onSaveLeader={(leader) => void saveLeader(leader)}
      onOpenIdeologies={onOpenIdeologies}
      status={status}
      error={error}
    />
  );
}
