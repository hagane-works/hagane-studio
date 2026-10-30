"use client";

import { useCallback, useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { Loader2, Trash2 } from "lucide-react";
import { statesApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { State } from "@/types";

export function StateRowDeleteButton({
  projectId,
  state,
  onDeleted,
}: {
  projectId: number;
  state: State;
  onDeleted: () => Promise<void> | void;
}) {
  const t = useTranslations("states");
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const edited = state.source === "created";

  const confirmDelete = useCallback(async () => {
    setBusy(true);
    try {
      await statesApi.remove(projectId, state.id);
      setOpen(false);
      toast.success(t("deleteDone", { id: state.state_id }));
      try {
        await onDeleted();
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        toast.error(t("deleteRefreshFailed", { id: state.state_id, msg }));
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setOpen(false);
      toast.error(t("deleteFailed", { msg }));
    } finally {
      setBusy(false);
    }
  }, [projectId, state.id, state.state_id, onDeleted, t]);

  return (
    <>
      <button
        type="button"
        aria-label={t("deleteButton", { id: state.state_id })}
        title={t("deleteButton", { id: state.state_id })}
        onClick={(e) => {
          e.stopPropagation();
          setOpen(true);
        }}
        className="ui-interactive shrink-0 ml-1.5 rounded p-0.5 text-muted-foreground/60 opacity-0 group-hover:opacity-100 focus-visible:opacity-100 hover:text-destructive focus-visible:ring-1 focus-visible:ring-ring"
      >
        <Trash2 className="h-3.5 w-3.5" aria-hidden />
      </button>

      <Dialog open={open} onOpenChange={(o) => !busy && setOpen(o)}>
        <DialogContent
          className="max-w-md"
          onClick={(e) => e.stopPropagation()}
        >
          <DialogHeader>
            <DialogTitle>
              {t("deleteConfirmTitle", { id: state.state_id })}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-2 text-sm">
            <p>{t("deleteConfirmBody")}</p>
            {edited && (
              <p className="text-warning">{t("deleteConfirmEdited")}</p>
            )}
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={busy}
              onClick={() => setOpen(false)}
            >
              {t("deleteCancel")}
            </Button>
            <Button
              variant="destructive"
              size="sm"
              disabled={busy}
              onClick={confirmDelete}
              className="ui-interactive"
            >
              {busy && (
                <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
              )}
              {t("deleteConfirm")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function StateClearSeededBar({
  projectId,
  states,
  onCleared,
}: {
  projectId: number;
  states: State[];
  onCleared: () => Promise<void> | void;
}) {
  const t = useTranslations("states");
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);

  const importedIds = useMemo(
    () => states.filter((s) => s.source === "imported").map((s) => s.id),
    [states],
  );

  const apply = useCallback(async () => {
    setBusy(true);
    try {
      const res = await statesApi.bulkDelete(projectId, {
        state_ids: importedIds,
      });
      setOpen(false);
      toast.success(t("clearSeededDone", { count: res.deleted }));
      if (res.skipped_created.length > 0) {
        toast.info(
          t("clearSeededSkipped", { count: res.skipped_created.length }),
        );
      }
      try {
        await onCleared();
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        toast.error(t("clearSeededRefreshFailed", { msg }));
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setOpen(false);
      toast.error(t("clearSeededFailed", { msg }));
    } finally {
      setBusy(false);
    }
  }, [projectId, importedIds, onCleared, t]);

  if (importedIds.length === 0) return null;

  return (
    <>
      <Button
        size="sm"
        variant="outline"
        disabled={busy}
        onClick={() => setOpen(true)}
        className="ui-interactive"
      >
        {busy && <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />}
        {t("clearSeededButton", { count: importedIds.length })}
      </Button>

      <Dialog open={open} onOpenChange={(o) => !busy && setOpen(o)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("clearSeededTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 text-sm">
            <p>{t("clearSeededBody", { count: importedIds.length })}</p>

            <div className="rounded-md border border-border bg-muted/40 p-2">
              <p className="text-xs text-muted-foreground">
                {t("clearSeededImportedNote")}
              </p>
            </div>
            <p className="text-xs text-muted-foreground">
              {t("clearSeededKeepEdited")}
            </p>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={busy}
              onClick={() => setOpen(false)}
            >
              {t("deleteCancel")}
            </Button>
            <Button
              variant="destructive"
              size="sm"
              disabled={busy}
              onClick={apply}
              className="ui-interactive"
            >
              {busy && (
                <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
              )}
              {t("clearSeededConfirm", { count: importedIds.length })}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
