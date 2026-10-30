"use client";

import { Check, LoaderCircle, RefreshCw } from "lucide-react";
import { useTranslations } from "use-intl";

export type CheckPhase = "idle" | "running" | "done" | "failed";

export function ExportStepper({
  phase,
  attentionCount,
  downloaded,
  onOpenVerify,
  onRerunCheck,
}: {
  phase: CheckPhase;
  attentionCount: number;
  downloaded: boolean;
  onOpenVerify: () => void;

  onRerunCheck?: () => void;
}) {
  const t = useTranslations("export");

  const step1Sub =
    phase === "idle"
      ? t("ui.stepper.step1SubIdle")
      : phase === "running"
        ? t("ui.stepper.step1Running")
        : phase === "failed"
          ? t("ui.stepper.step1Failed")
          : attentionCount > 0
            ? t("ui.stepper.step1SubIssues", { count: attentionCount })
            : t("ui.stepper.step1SubClean");

  const buttonLabel =
    phase === "running"
      ? t("ui.repair.recheckRunning")
      : phase === "idle"
        ? t("step1ButtonIdle")
        : t("ui.repair.recheck");

  const indexBadge = (state: "done" | "current" | "idle", label: string) => (
    <span
      aria-hidden
      className={`grid h-7 w-7 shrink-0 place-items-center rounded-full border text-xs font-semibold ${
        state === "done"
          ? "border-success text-success"
          : state === "current"
            ? "border-accent bg-accent text-accent-foreground"
            : "border-border text-muted-foreground"
      }`}
    >
      {state === "done" ? <Check className="h-3.5 w-3.5" aria-hidden /> : label}
    </span>
  );

  return (
    <nav
      aria-label={t("ui.stepper.aria")}
      className="grid grid-cols-1 gap-px overflow-hidden rounded-lg border border-border bg-border sm:grid-cols-3"
    >
      <div className="flex items-center gap-2.5 bg-card px-4 py-3">
        {indexBadge(
          phase === "done" ? "done" : phase === "idle" ? "idle" : "current",
          "1",
        )}
        <span className="min-w-0 flex-1">
          <span className="block text-xs font-semibold text-foreground">
            {t("ui.stepper.step1Title")}
          </span>
          <span className="block text-micro text-muted-foreground">
            {step1Sub}
          </span>
        </span>
        {onRerunCheck ? (
          <button
            type="button"
            onClick={onRerunCheck}
            disabled={phase === "running"}
            data-testid="export-rerun-check"
            title={buttonLabel}
            className={
              phase === "idle"
                ? "inline-flex min-h-8 shrink-0 items-center gap-1.5 rounded-md border border-accent bg-accent px-2.5 text-micro font-semibold text-accent-foreground ui-interactive ui-focus-ring hover:bg-accent/90"
                : "inline-flex min-h-8 shrink-0 items-center gap-1.5 rounded-md border border-border bg-card px-2.5 text-micro font-semibold text-muted-foreground ui-interactive ui-focus-ring hover:bg-muted/40 disabled:text-muted-foreground"
            }
          >
            {phase === "running" ? (
              <LoaderCircle className="h-3.5 w-3.5 animate-spin" aria-hidden />
            ) : (
              <RefreshCw className="h-3.5 w-3.5" aria-hidden />
            )}
            {buttonLabel}
          </button>
        ) : null}
      </div>
      <div className="flex items-center gap-2.5 bg-card px-4 py-3">
        {indexBadge(downloaded ? "done" : "current", "2")}
        <span className="min-w-0">
          <span className="block text-xs font-semibold text-foreground">
            {t("ui.stepper.step2Title")}
          </span>
          <span className="block text-micro text-muted-foreground">
            {downloaded
              ? t("ui.stepper.step2SubDone")
              : t("ui.stepper.step2Sub")}
          </span>
        </span>
      </div>
      {downloaded ? (
        <button
          type="button"
          onClick={onOpenVerify}
          className="flex items-center gap-2.5 bg-card px-4 py-3 text-left ui-interactive ui-focus-ring hover:bg-muted/40"
        >
          {indexBadge("current", "3")}
          <span className="min-w-0">
            <span className="block text-xs font-semibold text-foreground">
              {t("ui.stepper.step3Title")}
            </span>
            <span className="block text-micro text-muted-foreground">
              {t("ui.stepper.step3SubReady")}
            </span>
          </span>
        </button>
      ) : (
        <div className="flex items-center gap-2.5 bg-card px-4 py-3 opacity-70">
          {indexBadge("idle", "3")}
          <span className="min-w-0">
            <span className="block text-xs font-semibold text-muted-foreground">
              {t("ui.stepper.step3Title")}
            </span>
            <span className="block text-micro text-muted-foreground">
              {t("ui.stepper.step3SubLocked")}
            </span>
          </span>
        </div>
      )}
    </nav>
  );
}
