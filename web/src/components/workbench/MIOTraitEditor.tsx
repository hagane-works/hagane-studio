"use client";

import { useCallback, useState } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import type { MIOTraitEntry } from "@/lib/api";
import { DialogSaveStatusBar } from "./DialogSaveStatusBar";
import { MIOTraitCard } from "./MIOTraitsEditor";

export function MIOTraitEditor({
  trait,
  index,
  projectId,
  mioDbId,
  onSave,
}: {
  trait: MIOTraitEntry;

  index: number;
  projectId: number;

  mioDbId: number;

  onSave: (next: MIOTraitEntry) => Promise<void>;
}) {
  const t = useTranslations("mIOsDialog");

  const [draft, setDraft] = useState<MIOTraitEntry>(trait);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const patchAt = useCallback(
    (_index: number, patch: (prev: MIOTraitEntry) => MIOTraitEntry) =>
      setDraft((prev) => patch({ ...prev })),
    [],
  );

  const save = async () => {
    setSaving(true);
    setSaved(false);
    setError(null);
    try {
      await onSave(draft);
      setSaved(true);
      setTimeout(() => setSaved(false), 1500);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-3 p-4">
      <MIOTraitCard
        index={index}
        trait={draft}
        projectId={projectId}
        mioDbId={mioDbId}
        patchAt={patchAt}
      />

      <div className="sticky bottom-0 bg-card pt-1 pb-2">
        <div className="mb-2 empty:mb-0">
          <DialogSaveStatusBar
            saving={saving}
            saved={saved}
            error={error}
            savingLabel={t("saving")}
            savedLabel={t("saved")}
          />
        </div>
        <div className="flex items-center justify-end gap-2">
          <Button
            size="sm"
            onClick={() => void save()}
            disabled={saving || !(draft.token ?? "").trim()}
          >
            {saving ? t("saving") : t("save")}
          </Button>
        </div>
      </div>
    </div>
  );
}
