"use client";

import type { ReactNode } from "react";
import {
  AlertTriangle,
  Check,
  Download,
  FileSearch,
  LoaderCircle,
} from "lucide-react";
import { useTranslations } from "use-intl";

import type { CheckPhase } from "@/components/export/ExportStepper";

export function ExportStatusCard({
  phase,
  attentionCount,
  hasBlockerAttention,
  autoCount,
  healthCount,
  checkRequestFailed,
  downloading,
  buildErrorMessage,
  repairableCount = null,
  staleReason = null,
  checkedAtLabel = null,
  onDownload,
  onFix,
}: {
  phase: CheckPhase;
  attentionCount: number;

  hasBlockerAttention: boolean;
  autoCount: number;
  healthCount: number;

  checkRequestFailed: boolean;
  downloading: boolean;
  buildErrorMessage: string | null;

  repairableCount?: number | null;

  staleReason?: "settings" | "project-edited" | null;

  checkedAtLabel?: string | null;
  onDownload: () => void;
  onFix: () => void;
}) {
  const t = useTranslations("export");

  const warn = phase === "done" && hasBlockerAttention;
  const clean = phase === "done" && attentionCount === 0;

  const staleLine =
    staleReason === "project-edited"
      ? t("ui.status.staleEdited", { time: checkedAtLabel ?? "" })
      : staleReason === "settings"
        ? t("ui.status.staleSettings")
        : null;
  const bold = (chunks: ReactNode) => (
    <b className="font-semibold text-foreground">{chunks}</b>
  );

  const title =
    phase === "idle"
      ? t("ui.status.titleIdle")
      : phase === "running"
        ? t("ui.status.checking")
        : phase === "failed"
          ? t("ui.status.titleOk")
          : clean
            ? t("ui.status.titleClean")
            : warn
              ? t("ui.status.titleAttn", { count: attentionCount })
              : t("ui.status.titleOk");

  const copy: ReactNode =
    phase === "idle"
      ? t("ui.status.copyIdle")
      : phase === "running"
        ? t("ui.status.checkingSub")
        : phase === "failed"
          ? t("ui.status.checkingSub")
          : clean
            ? autoCount > 0
              ? t("ui.status.copyCleanAuto", { auto: autoCount })
              : t("ui.status.copyClean")
            : warn
              ? t("ui.status.copyAttn", { count: attentionCount })
              : autoCount > 0
                ? t.rich("ui.status.copyOkAuto", {
                    count: attentionCount,
                    auto: autoCount,
                    b: bold,
                  })
                : t.rich("ui.status.copyOk", {
                    count: attentionCount,
                    b: bold,
                  });

  const repairableLine =
    phase === "done" &&
    attentionCount > 0 &&
    repairableCount !== null &&
    repairableCount > 0
      ? t("ui.status.copyAttnRepairable", { count: repairableCount })
      : null;

  const healthLine =
    phase === "idle"
      ? null
      : checkRequestFailed || phase === "failed"
        ? t("ui.status.healthRequestFailed")
        : healthCount > 0
          ? t("ui.status.health", { count: healthCount })
          : null;

  const edge =
    phase === "idle"
      ? "border-border border-l-border"
      : warn
        ? "border-warning/50 border-l-warning"
        : "border-success/50 border-l-success";

  return (
    <section
      aria-live="polite"
      className={`grid grid-cols-1 items-center gap-4 rounded-lg border border-l-4 bg-card p-5 md:grid-cols-[minmax(0,1fr)_auto] ${edge}`}
    >
      <div className="min-w-0">
        <div className="flex items-center gap-2.5">
          <span
            aria-hidden
            className={`grid h-6 w-6 shrink-0 place-items-center rounded-full text-xs font-semibold ${
              phase === "idle"
                ? "bg-muted text-muted-foreground"
                : warn
                  ? "bg-warning/15 text-warning"
                  : "bg-success/15 text-success"
            }`}
          >
            {phase === "running" ? (
              <LoaderCircle className="h-3.5 w-3.5 animate-spin" aria-hidden />
            ) : phase === "idle" ? (
              <FileSearch className="h-3.5 w-3.5" aria-hidden />
            ) : warn ? (
              <AlertTriangle className="h-3.5 w-3.5" aria-hidden />
            ) : (
              <Check className="h-3.5 w-3.5" aria-hidden />
            )}
          </span>
          <h3 className="text-base font-semibold text-foreground">{title}</h3>
        </div>
        <p className="mt-1.5 max-w-xl pl-[34px] text-sm text-muted-foreground">
          {copy}
          {repairableLine ? (
            <span
              className="mt-1 block text-xs"
              data-testid="repairable-count-line"
            >
              {repairableLine}
            </span>
          ) : null}
          {staleLine ? (
            <span
              className="mt-1 block text-xs text-warning"
              data-testid="check-stale-notice"
            >
              {staleLine}
            </span>
          ) : null}
          {healthLine ? (
            <span
              className="mt-1 block text-xs"
              data-testid="check-incomplete-banner"
            >
              {healthLine}
            </span>
          ) : null}
          {buildErrorMessage ? (
            <span className="mt-1 block text-xs text-destructive">
              {t("ui.status.buildFailed", { message: buildErrorMessage })}
            </span>
          ) : null}
        </p>
      </div>
      <div className="flex flex-wrap justify-start gap-2 md:justify-end">
        <button
          type="button"
          onClick={onDownload}
          disabled={downloading}
          className="inline-flex min-h-10 items-center gap-2 rounded-lg border border-accent bg-accent px-4 text-sm font-semibold text-accent-foreground ui-interactive ui-focus-ring hover:bg-accent/90 disabled:opacity-70"
        >
          {downloading ? (
            <LoaderCircle className="h-4 w-4 animate-spin" aria-hidden />
          ) : (
            <Download className="h-4 w-4" aria-hidden />
          )}
          {downloading
            ? t("ui.status.downloadPreparing")
            : t("ui.status.download")}
        </button>
        {phase === "done" && attentionCount > 0 ? (
          <button
            type="button"
            onClick={onFix}
            className="inline-flex min-h-10 items-center gap-2 rounded-lg border border-border bg-card px-4 text-sm font-semibold text-foreground ui-interactive ui-focus-ring hover:bg-muted/40"
          >
            {t("ui.status.fix", { count: attentionCount })}
          </button>
        ) : null}
      </div>
    </section>
  );
}
