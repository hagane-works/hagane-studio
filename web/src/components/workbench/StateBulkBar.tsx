"use client";

import { useCallback, useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { Undo2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { statesApi } from "@/lib/api";
import {
  summarizeBulk,
  summarizeCurrentValues,
  toRestoreEntries,
} from "@/lib/states/bulkOwnerCore";
import type { StateBulkResult, StateCoreMode } from "@/types";

const CORE_MODES: StateCoreMode[] = [
  "add",
  "remove",
  "replace_one",
  "replace_all",
];

interface StateBulkBarProps {
  projectId: number;

  selectedIds: number[];

  selectedGeoCount: number;
  onClear: () => void;

  onApplied: (result: StateBulkResult) => void;

  knownTags: string[];

  selectedRows: { owner?: string | null; add_core_of?: string[] | null }[];
}

export function StateBulkBar({
  projectId,
  selectedIds,
  selectedGeoCount,
  onClear,
  onApplied,
  knownTags,
  selectedRows,
}: StateBulkBarProps) {
  const t = useTranslations("states");
  const [owner, setOwner] = useState("");

  const [controller, setController] = useState("");

  const [coreMode, setCoreMode] = useState<StateCoreMode>("add");
  const [coreFrom, setCoreFrom] = useState("");
  const [coreTag, setCoreTag] = useState("");
  const [preview, setPreview] = useState<StateBulkResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const ownerTrimmed = owner.trim();
  const controllerTrimmed = controller.trim();
  const coreTagTrimmed = coreTag.trim();
  const coreFromTrimmed = coreFrom.trim();

  const coresPayload = useMemo(() => {
    if (!coreTagTrimmed) return undefined;
    if (coreMode === "replace_one" && !coreFromTrimmed) return undefined;
    return {
      mode: coreMode,
      tags: [coreTagTrimmed],
      ...(coreMode === "replace_one" ? { from: coreFromTrimmed } : {}),
    };
  }, [coreMode, coreTagTrimmed, coreFromTrimmed]);

  const canSubmit =
    selectedIds.length > 0 &&
    (ownerTrimmed !== "" || controllerTrimmed !== "" || coresPayload != null);

  const runPreview = useCallback(async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await statesApi.bulkOwnerCore(projectId, {
        state_db_ids: selectedIds,
        ...(ownerTrimmed ? { owner: ownerTrimmed } : {}),
        ...(controllerTrimmed ? { controller: controllerTrimmed } : {}),
        ...(coresPayload ? { cores: coresPayload } : {}),
        dry_run: true,
      });
      setPreview(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [projectId, selectedIds, ownerTrimmed, controllerTrimmed, coresPayload]);

  const runApply = useCallback(async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await statesApi.bulkOwnerCore(projectId, {
        state_db_ids: selectedIds,
        ...(ownerTrimmed ? { owner: ownerTrimmed } : {}),
        ...(controllerTrimmed ? { controller: controllerTrimmed } : {}),
        ...(coresPayload ? { cores: coresPayload } : {}),
      });
      setPreview(null);
      onApplied(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [
    projectId,
    selectedIds,
    ownerTrimmed,
    controllerTrimmed,
    coresPayload,
    onApplied,
  ]);

  if (selectedGeoCount === 0) return null;

  const summary = summarizeBulk(preview);
  const current = summarizeCurrentValues(selectedRows);

  const notInProject = Math.max(0, selectedGeoCount - selectedIds.length);

  return (
    <>
      <div className="border-t border-border bg-card px-4 py-3">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <span className="text-sm">
            {t("bulkSelected", { count: selectedGeoCount })}
          </span>

          {current.owners.length > 0 && (
            <span className="text-xs text-muted-foreground">
              {t("bulkCurrentOwners")}{" "}
              <span className="font-mono">
                {current.owners
                  .slice(0, 3)
                  .map((o) => `${o.tag || t("bulkNoOwner")} ×${o.count}`)
                  .join(" · ")}
              </span>
              {current.owners.length > 3 &&
                " · " +
                  t("bulkCurrentOwnersMore", {
                    count: current.owners.length - 3,
                  })}
            </span>
          )}
          {notInProject > 0 && (
            <span className="text-xs text-muted-foreground">
              {t("bulkNotInProject", { count: notInProject })}
            </span>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={onClear}
            className="ui-interactive"
          >
            <X className="h-3.5 w-3.5" aria-hidden />
            {t("bulkClear")}
          </Button>

          <div className="flex items-center gap-2">
            <label
              className="text-xs text-muted-foreground"
              htmlFor="bulk-owner"
            >
              {t("bulkOwnerLabel")}
            </label>
            <Input
              id="bulk-owner"
              value={owner}
              onChange={(e) => setOwner(e.target.value.toUpperCase())}
              placeholder={t("bulkTagPlaceholder")}
              list="bulk-known-tags"
              className="h-8 w-28 font-mono"
              spellCheck={false}
            />
          </div>

          <div className="flex items-center gap-2">
            <label
              className="text-xs text-muted-foreground"
              htmlFor="bulk-controller"
            >
              {t("bulkControllerLabel")}
            </label>
            <Input
              id="bulk-controller"
              value={controller}
              onChange={(e) => setController(e.target.value.toUpperCase())}
              placeholder={t("bulkTagPlaceholder")}
              list="bulk-known-tags"
              className="h-8 w-28 font-mono"
              spellCheck={false}
            />
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs text-muted-foreground">
              {t("bulkCoresLabel")}
            </span>
            <div className="inline-flex rounded-md border border-border overflow-hidden">
              {CORE_MODES.map((mode) => (
                <button
                  key={mode}
                  type="button"
                  aria-pressed={coreMode === mode}
                  onClick={() => setCoreMode(mode)}
                  className={`px-2.5 py-1 text-xs ui-interactive border-l border-border first:border-l-0 ${
                    coreMode === mode
                      ? "bg-accent text-accent-foreground"
                      : "bg-background text-muted-foreground hover:bg-muted"
                  }`}
                >
                  {t(`bulkCoreMode_${mode}`)}
                </button>
              ))}
            </div>
            {coreMode === "replace_one" && (
              <>
                <Input
                  value={coreFrom}
                  onChange={(e) => setCoreFrom(e.target.value.toUpperCase())}
                  placeholder={t("bulkCoreFromPlaceholder")}
                  list="bulk-current-cores"
                  className="h-8 w-28 font-mono"
                  spellCheck={false}
                  aria-label={t("bulkCoreFromPlaceholder")}
                />
                <datalist id="bulk-current-cores">
                  {current.cores.map((c) => (
                    <option key={c} value={c} />
                  ))}
                </datalist>
              </>
            )}
            <Input
              value={coreTag}
              onChange={(e) => setCoreTag(e.target.value.toUpperCase())}
              placeholder={t("bulkTagPlaceholder")}
              list="bulk-known-tags"
              className="h-8 w-28 font-mono"
              spellCheck={false}
              aria-label={t("bulkCoresLabel")}
            />

            {current.cores.length > 0 && (
              <span className="text-xs text-muted-foreground">
                {t("bulkCurrentCores")}{" "}
                <span className="font-mono">{current.cores.join(" ")}</span>
              </span>
            )}
          </div>

          <Button
            size="sm"
            disabled={!canSubmit || busy}
            onClick={runPreview}
            className="ml-auto ui-interactive"
          >
            {busy && preview == null ? t("bulkPreviewing") : t("bulkPreview")}
          </Button>
        </div>

        <datalist id="bulk-known-tags">
          {knownTags.map((tag) => (
            <option key={tag} value={tag} />
          ))}
        </datalist>

        {error && (
          <p className="mt-2 text-xs text-destructive">
            {t("bulkFailed", { msg: error })}
          </p>
        )}
      </div>

      <Dialog
        open={preview != null}
        onOpenChange={(o) => !o && setPreview(null)}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>
              {t("bulkConfirmTitle", { count: selectedIds.length })}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-3 text-sm">
            {controllerTrimmed && (
              <p>{t("bulkConfirmController", { tag: controllerTrimmed })}</p>
            )}
            {ownerTrimmed && (
              <div>
                <p>{t("bulkConfirmOwner", { tag: ownerTrimmed })}</p>
                {summary.skipped > 0 && (
                  <p className="text-xs text-muted-foreground">
                    {t("bulkConfirmSkipped", { count: summary.skipped })}
                  </p>
                )}
              </div>
            )}
            {coresPayload && (
              <p>
                {coreMode === "replace_one"
                  ? t("bulkConfirmCoresReplaceOne", {
                      from: coreFromTrimmed,
                      tag: coreTagTrimmed,
                    })
                  : t(`bulkConfirmCores_${coreMode}`, { tag: coreTagTrimmed })}
              </p>
            )}

            {summary.removedTags.length > 0 && (
              <div className="rounded-md border border-border bg-muted/40 p-2">
                <p className="text-xs">{t("bulkConfirmRemovedTitle")}</p>
                <ul className="mt-1 space-y-0.5">
                  {summary.removedTags.map(({ tag, count }) => (
                    <li key={tag} className="font-mono text-xs">
                      {t("bulkConfirmRemovedItem", { tag, count })}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {summary.changed === 0 ? (
              <p className="text-xs text-muted-foreground">
                {t("bulkConfirmNoop")}
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                {t("bulkConfirmUndoable")}
              </p>
            )}
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setPreview(null)}
            >
              {t("bulkCancel")}
            </Button>
            <Button
              size="sm"
              disabled={busy || summary.changed === 0}
              onClick={runApply}
              className="ui-interactive"
            >
              {busy
                ? t("bulkApplying")
                : t("bulkConfirmApply", { count: summary.changed })}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

interface StateBulkUndoBarProps {
  projectId: number;
  result: StateBulkResult;
  onUndone: () => void;
  onDismiss: () => void;
}

export function StateBulkUndoBar({
  projectId,
  result,
  onUndone,
  onDismiss,
}: StateBulkUndoBarProps) {
  const t = useTranslations("states");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const entries = useMemo(() => toRestoreEntries(result), [result]);

  const undo = useCallback(async () => {
    setBusy(true);
    setError(null);
    try {
      await statesApi.bulkRestore(projectId, entries);
      onUndone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [projectId, entries, onUndone]);

  if (result.changed === 0) return null;

  return (
    <div className="flex flex-wrap items-center gap-2 border-b border-border bg-muted/40 px-4 py-2 text-xs">
      <span>{t("bulkDone", { count: result.changed })}</span>
      {result.skipped > 0 && (
        <span className="text-muted-foreground">
          {t("bulkDoneSkipped", { count: result.skipped })}
        </span>
      )}
      <Button
        variant="outline"
        size="sm"
        disabled={busy}
        onClick={undo}
        className="ml-auto ui-interactive"
      >
        <Undo2 className="h-3.5 w-3.5" aria-hidden />
        {busy ? t("bulkUndoing") : t("bulkUndo")}
      </Button>
      <Button
        variant="ghost"
        size="sm"
        onClick={onDismiss}
        aria-label={t("bulkDismiss")}
      >
        <X className="h-3.5 w-3.5" aria-hidden />
      </Button>
      {error && (
        <p className="w-full text-destructive">
          {t("bulkFailed", { msg: error })}
        </p>
      )}
    </div>
  );
}
