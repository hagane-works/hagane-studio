"use client";

import { useCallback, useEffect, useState } from "react";
import { useTranslations } from "use-intl";
import { XIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import {
  useBackgroundImportSession,
  useCancelBackgroundImport,
  useClearBackgroundImport,
  rehydrateBackgroundImport,
  type BackgroundImportStage,
} from "@/lib/store/backgroundImportStore";

const STAGE_LABEL_KEYS: Record<BackgroundImportStage, string> = {
  queued: "stageQueued",
  scanning: "stageScanning",
  parsing: "stageParsing",
  uploading: "stageUploading",
  done: "stageDone",
  error: "stageError",
  cancelled: "stageCancelled",
};

const TERMINAL_STAGES: readonly BackgroundImportStage[] = [
  "done",
  "error",
  "cancelled",
];

function formatElapsed(startedAt: number, now: number): string {
  const secs = Math.max(0, Math.floor((now - startedAt) / 1000));
  if (secs < 60) return `${secs}s`;
  const m = Math.floor(secs / 60);
  const s = secs % 60;
  return s === 0 ? `${m}m` : `${m}m ${s}s`;
}

function formatEta(eta: number | null): string | null {
  if (eta === null || eta <= 0) return null;
  if (eta < 60) return `~${Math.round(eta)}s`;
  const m = Math.round(eta / 60);
  return `~${m}m`;
}

export function BackgroundImportWidget() {
  const session = useBackgroundImportSession();
  const cancelBackgroundImport = useCancelBackgroundImport();
  const clearBackgroundImport = useClearBackgroundImport();
  const t = useTranslations("backgroundImport");
  const confirmDialog = useConfirm();

  useEffect(() => {
    rehydrateBackgroundImport();
  }, []);

  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!session) return;
    if (TERMINAL_STAGES.includes(session.stage)) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [session]);

  const handleOpen = useCallback(() => {
    window.dispatchEvent(new CustomEvent("background-import:open"));
  }, []);

  const handleDismissOrCancel = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      if (!session) return;
      if (TERMINAL_STAGES.includes(session.stage)) {
        clearBackgroundImport();
        return;
      }
      const ok = await confirmDialog({
        body: t("confirmCancel"),
        danger: true,
      });
      if (!ok) return;
      await cancelBackgroundImport();
    },
    [session, cancelBackgroundImport, clearBackgroundImport, t, confirmDialog],
  );

  if (!session) return null;

  const stageLabel = t(STAGE_LABEL_KEYS[session.stage]);
  const isTerminal = TERMINAL_STAGES.includes(session.stage);

  const elapsed = formatElapsed(
    session.startedAt,
    isTerminal && session.finishedAt ? session.finishedAt : now,
  );
  const eta = formatEta(session.etaSeconds);
  const percent = session.percent;

  const accentClass =
    session.stage === "error"
      ? "border-destructive/70 bg-destructive/10"
      : session.stage === "done"
        ? "border-emerald-500/70 bg-emerald-500/10"
        : session.stage === "cancelled"
          ? "border-zinc-500/70 bg-zinc-500/10"
          : "border-blue-500/70 bg-blue-500/10";

  return (
    <div
      className={`fixed bottom-24 right-4 z-40 w-72 rounded-lg border ${accentClass} shadow-lg backdrop-blur-sm`}
      role="status"
      aria-live="polite"
      data-testid="background-import-widget"
    >
      <button
        type="button"
        onClick={handleOpen}
        className="w-full text-left p-3 cursor-pointer hover:bg-background/30 transition-colors rounded-lg"

        aria-label={t("reopenAria", { project: session.projectName })}
      >
        <div className="flex items-start justify-between gap-2 mb-1.5">
          <div className="flex-1 min-w-0">
            <p className="text-xs font-semibold truncate">
              {t("titlePrefix")} · {session.projectName}
            </p>
            <p className="text-xs text-muted-foreground truncate">
              {session.countryTag ? `${session.countryTag} · ` : ""}
              {stageLabel}
            </p>
          </div>
          <span
            role="button"
            tabIndex={0}
            aria-label={isTerminal ? t("dismissAria") : t("cancelAria")}
            onClick={handleDismissOrCancel}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                void handleDismissOrCancel(e as unknown as React.MouseEvent);
              }
            }}
            className="shrink-0 inline-flex items-center justify-center h-6 w-6 rounded hover:bg-foreground/10 cursor-pointer"
          >
            <XIcon className="h-3.5 w-3.5" />
          </span>
        </div>

        <p className="text-xs text-muted-foreground line-clamp-2 min-h-[2em]">
          {session.statusText || stageLabel}
        </p>

        {isTerminal && session.result && (
          <div className="mt-2 space-y-0.5 border-t border-foreground/10 pt-2">
            {session.result.interrupted && (
              <p className="text-xs font-medium text-amber-600 dark:text-amber-400">
                {t("interruptedNote")}
              </p>
            )}
            {session.result.headline && (
              <p className="text-xs font-semibold">{session.result.headline}</p>
            )}
            {session.result.lines.map((line, i) => (
              <p
                key={`${i}-${line.slice(0, 16)}`}
                className="text-micro text-muted-foreground"
              >
                {line}
              </p>
            ))}
          </div>
        )}

        {!isTerminal && (
          <div className="mt-2 h-1 w-full rounded-full bg-foreground/10 overflow-hidden">
            {percent === null ? (
              <div className="h-full w-1/3 bg-blue-500 animate-pulse rounded-full" />
            ) : (
              <div
                className="h-full bg-blue-500 transition-[width] duration-300 rounded-full"
                style={{ width: `${Math.min(100, Math.max(0, percent))}%` }}
              />
            )}
          </div>
        )}

        <div className="mt-1.5 flex items-center justify-between text-micro text-muted-foreground">
          <span>
            {t("elapsedPrefix")} {elapsed}
          </span>
          {percent !== null && !isTerminal && (
            <span>{Math.round(percent)}%</span>
          )}
          {eta && !isTerminal && <span>{eta}</span>}
          {isTerminal && session.finalMessage && (
            <span className="truncate max-w-[10rem]">
              {session.finalMessage}
            </span>
          )}
        </div>
      </button>

      {isTerminal && (
        <div className="flex justify-end gap-1 px-3 pb-2">
          <Button
            size="sm"
            variant="ghost"
            className="h-6 px-2 text-xs"
            onClick={(e) => {
              e.stopPropagation();
              clearBackgroundImport();
            }}
          >
            {t("dismiss")}
          </Button>
        </div>
      )}
    </div>
  );
}
