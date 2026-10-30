"use client";

import { useState, useRef } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { readProjectEditRevisions } from "./save-diagnostics/revisionRequest";
import type { EditRevisions } from "./save-diagnostics/editRevisions";
import { projectsApi } from "@/lib/api";

const WORKBENCH_KEYS = [
  "characters",
  "ideas",
  "focus_trees",
  "events",
  "decisions",
  "technologies",
] as const;

type Counts = Record<string, number>;

type Phase = "idle" | "previewing" | "preview" | "flipping" | "done" | "error";

export function BatchFlipButton({
  projectId,
  onSuccess,
  variant = "outline",
}: {
  projectId: number;
  onSuccess?: () => void;
  variant?: "outline" | "destructive";
}) {
  const t = useTranslations("batchFlipButton");
  const revisions = useRef<EditRevisions>({});
  const [open, setOpen] = useState(false);
  const [phase, setPhase] = useState<Phase>("idle");
  const [counts, setCounts] = useState<Counts>({});
  const [total, setTotal] = useState(0);
  const [error, setError] = useState<string>("");
  const [acknowledged, setAcknowledged] = useState(false);

  const reset = () => {
    setPhase("idle");
    setCounts({});
    setTotal(0);
    setError("");
    setAcknowledged(false);
  };

  const handleOpen = async () => {
    reset();
    setOpen(true);
    setPhase("previewing");
    try {
      revisions.current = await readProjectEditRevisions(projectId);
      const res = await projectsApi.flipAll(
        projectId,
        { dry_run: true },
        revisions.current,
      );
      setCounts(res.flipped);
      setTotal(res.total);
      setPhase("preview");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setPhase("error");
    }
  };

  const handleConfirm = async () => {
    setPhase("flipping");
    try {
      const res = await projectsApi.flipAll(
        projectId,
        { dry_run: false },
        revisions.current,
      );
      setCounts(res.flipped);
      setTotal(res.total);
      setPhase("done");

      toast.success(t("toastDone", { total: res.total }));
      onSuccess?.();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg);
      setPhase("error");
      toast.error(t("toastFail", { msg }));
    }
  };

  const handleClose = (isOpen: boolean) => {
    setOpen(isOpen);
    if (!isOpen) reset();
  };

  return (
    <>
      <Button
        variant={variant}
        className="w-full text-sm"
        onClick={handleOpen}
        title={t("buttonTitle")}
      >
        {t("buttonLabel")}
      </Button>

      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>
              {phase === "done" ? t("titleDone") : t("titleConfirm")}
            </DialogTitle>
            <DialogDescription>
              {phase === "done" ? t("descDone", { total }) : t("descConfirm")}
            </DialogDescription>
          </DialogHeader>

          {(phase === "previewing" || phase === "flipping") && (
            <p className="text-sm text-muted-foreground">
              {phase === "previewing" ? t("previewing") : t("flipping")}
            </p>
          )}

          {phase === "preview" && (
            <div className="space-y-3">
              <div className="rounded-md border border-red-500/60 bg-red-500/10 p-3 text-sm text-red-700 dark:text-red-300">
                <p className="font-semibold">{t("warnTitle")}</p>
                <p className="mt-1 text-xs">{t("warnBody1")}</p>
                <p className="mt-1 text-xs">{t("warnBody2")}</p>
              </div>

              <div className="rounded-md border border-border bg-muted/40 p-3 text-sm">
                <div className="grid grid-cols-2 gap-y-1 font-mono text-xs">
                  {WORKBENCH_KEYS.map((key) => (
                    <div key={key} className="contents">
                      <span className="text-muted-foreground">
                        {t(`workbench.${key}`)}
                      </span>
                      <span className="text-right tabular-nums">
                        {counts[key] ?? 0}
                      </span>
                    </div>
                  ))}
                </div>
                <div className="mt-2 border-t border-border pt-2 flex justify-between text-sm font-semibold">
                  <span>{t("total")}</span>
                  <span className="tabular-nums">{total}</span>
                </div>
                {total === 0 && (
                  <p className="mt-2 text-xs text-muted-foreground">
                    {t("noFlip")}
                  </p>
                )}
                {total > 0 && (
                  <p className="mt-2 text-xs text-amber-700 dark:text-amber-400">
                    {t("irreversible")}
                  </p>
                )}
              </div>

              {total > 0 && (
                <label className="flex items-start gap-2 rounded-md border border-red-500/40 bg-red-500/5 p-3 text-xs cursor-pointer">
                  <input
                    type="checkbox"
                    checked={acknowledged}
                    onChange={(e) => setAcknowledged(e.target.checked)}
                    className="mt-0.5 h-4 w-4 accent-red-500 shrink-0"
                  />
                  <span>
                    {t.rich("ackText", {
                      b: (chunks) => <strong>{chunks}</strong>,
                    })}
                  </span>
                </label>
              )}
            </div>
          )}

          {phase === "done" && (
            <div className="rounded-md border border-border bg-muted/40 p-3 text-sm">
              <div className="grid grid-cols-2 gap-y-1 font-mono text-xs">
                {WORKBENCH_KEYS.map((key) => (
                  <div key={key} className="contents">
                    <span className="text-muted-foreground">
                      {t(`workbench.${key}`)}
                    </span>
                    <span className="text-right tabular-nums">
                      {counts[key] ?? 0}
                    </span>
                  </div>
                ))}
              </div>
              <div className="mt-2 border-t border-border pt-2 flex justify-between text-sm font-semibold">
                <span>{t("total")}</span>
                <span className="tabular-nums">{total}</span>
              </div>
              <p className="mt-2 text-xs text-muted-foreground">
                {t("openExportHint")}
              </p>
            </div>
          )}

          {phase === "error" && (
            <div className="rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/30 dark:text-red-300">
              <p className="font-semibold">{t("flipFailed")}</p>
              <p className="mt-1 text-xs font-mono break-all">{error}</p>
            </div>
          )}

          <DialogFooter>
            {phase === "preview" && (
              <>
                <Button variant="outline" onClick={() => handleClose(false)}>
                  {t("cancel")}
                </Button>
                <Button
                  onClick={handleConfirm}
                  disabled={total === 0 || !acknowledged}
                  variant="destructive"
                >
                  {total === 0
                    ? t("noFlipButton")
                    : !acknowledged
                      ? t("needAck")
                      : t("flipN", { total })}
                </Button>
              </>
            )}
            {(phase === "done" || phase === "error") && (
              <Button onClick={() => handleClose(false)}>{t("close")}</Button>
            )}
            {(phase === "previewing" || phase === "flipping") && (
              <Button disabled>{t("processing")}</Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
