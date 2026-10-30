import { useState } from "react";
import { historyFilesApi, type HistoryFileRow } from "@/lib/api/scenario";
import {
  historyRowToDraft,
  type HistoryFileDraft,
} from "@/components/workbench/HistoryFileEditor";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { stableDraftFingerprint } from "@/lib/workbench-state/draftSession";
import { pick, recordChange, rowUndo } from "./changeLog";
import { changedKeys } from "./workbenchProject";

export function asDrafted(row: HistoryFileRow): HistoryFileRow {
  return { ...row, ...historyRowToDraft(row) };
}

export function useHistoryFileDraft({
  projectId,
  row,
  title,
  kind,
  workbenchHref,
  check,
  afterSave,
}: {
  projectId: number;

  row: HistoryFileRow;

  title: string;
  kind: string;
  workbenchHref: string;

  check?: (draft: HistoryFileDraft) => string | null;

  afterSave: (saved: HistoryFileRow) => Promise<void> | void;
}) {
  const [base, setBase] = useState<HistoryFileRow>(row);
  const [draft, setDraft] = useState<HistoryFileDraft>(() =>
    historyRowToDraft(row),
  );
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const dirty =
    stableDraftFingerprint(draft) !==
    stableDraftFingerprint(historyRowToDraft(base));
  useRegisterDirty(
    `game-view-history-file:${projectId}:${row.id}:${title}`,
    dirty || saving,
  );

  const onChange = <K extends keyof HistoryFileDraft>(
    key: K,
    value: HistoryFileDraft[K],
  ) => {
    setSaved(false);
    setDraft((current) => ({ ...current, [key]: value }));
  };

  const save = async () => {
    const problem = check?.(draft) ?? null;
    if (problem) {
      setSaved(false);
      setError(problem);
      return;
    }
    const sent = draft;
    const id = base.id;
    const drafted = asDrafted(base);
    const keys = changedKeys(drafted, sent);
    const before = pick(drafted, keys);
    setSaving(true);
    setSaved(false);
    setError(null);
    try {
      const answer = await historyFilesApi.update(projectId, id, sent);
      recordChange({
        projectId,
        title,
        kind,
        workbenchHref,
        undo: keys.length
          ? rowUndo<HistoryFileRow>({
              before,
              after: pick(asDrafted(answer), keys),
              load: async () =>
                asDrafted(await historyFilesApi.get(projectId, id)),

              save: (back) => historyFilesApi.update(projectId, id, back),
            })
          : undefined,
      });
      setBase(answer);

      setDraft((current) =>
        current === sent ? historyRowToDraft(answer) : current,
      );
      setSaved(true);
      await afterSave(answer);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const adopt = (
    written: HistoryFileRow,
    change: (current: HistoryFileDraft) => HistoryFileDraft,
  ) => {
    setBase(written);
    setDraft(change);
  };

  return { base, draft, onChange, save, saving, saved, error, dirty, adopt };
}
