"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import type { AppLocale } from "@/i18n/routing";
import { cn } from "@/lib/utils";
import { Progress } from "@/components/ui/progress";
import { appApi, type GameIndexStatus } from "@/desktop/appApi";
import { GAME_INDEX_KEY } from "@/desktop/GameIndexStatus";
import {
  estimateRemainingMs,
  formatDuration,
  useImportProgressSSE,
  type ImportBucketKey,
  type ImportProgress,
  type ImportStage,
} from "@/lib/hooks/useImportProgressSSE";

interface ImportProgressPanelProps {
  projectId: number;

  sessionId: string | null;

  localPhase: "parsing" | "uploading" | "done";

  uploadProgress?: {
    loaded: number;
    total: number;
    lengthComputable: boolean;
  } | null;

  imageProgress?: { processed: number; total: number } | null;

  fallbackStatus?: string;

  compact?: boolean;

  locale?: AppLocale;
}

const BUCKETS: ImportBucketKey[] = [
  "countries",
  "characters",
  "events",
  "focus",
  "decisions",
  "ideas",
  "dynamic_mod",
  "superevents",
];

export function ImportProgressPanel({
  projectId,
  sessionId,
  localPhase,
  uploadProgress,
  imageProgress,
  fallbackStatus,
  compact = false,
  locale = "en",
}: ImportProgressPanelProps) {
  const t = useTranslations("unifiedModImport");

  const { progress, transport } = useImportProgressSSE(projectId, sessionId);

  const [startMs, setStartMs] = useState<number | null>(null);
  const lastSessionRef = useRef<string | null>(null);
  useEffect(() => {
    if (sessionId && sessionId !== lastSessionRef.current) {
      setStartMs(Date.now());
      lastSessionRef.current = sessionId;
    }
    if (!sessionId) {
      lastSessionRef.current = null;
      setStartMs(null);
    }
  }, [sessionId]);

  const recentFilesRef = useRef<string[]>([]);
  const lastUpdateMsRef = useRef<number>(0);
  const [recentFilesDisplay, setRecentFilesDisplay] = useState<string[]>([]);
  useEffect(() => {
    if (!progress?.current_file) return;
    const buf = recentFilesRef.current;
    if (buf[0] === progress.current_file) return;
    buf.unshift(progress.current_file);
    if (buf.length > 3) buf.length = 3;
    const now = Date.now();
    if (now - lastUpdateMsRef.current > 200) {
      lastUpdateMsRef.current = now;
      setRecentFilesDisplay([...buf]);
    }
  }, [progress?.current_file]);

  const [, forceTick] = useState(0);
  useEffect(() => {
    if (!sessionId || localPhase === "done") return;
    const id = setInterval(() => forceTick((x) => x + 1), 1000);
    return () => clearInterval(id);
  }, [sessionId, localPhase]);

  const uploadActive = !!uploadProgress;
  const uploadPct =
    uploadProgress && uploadProgress.total > 0
      ? Math.min(100, (uploadProgress.loaded / uploadProgress.total) * 100)
      : 0;
  const uploadDoneMb = uploadProgress
    ? uploadProgress.loaded / (1024 * 1024)
    : 0;
  const uploadTotalMb = uploadProgress
    ? uploadProgress.total / (1024 * 1024)
    : 0;

  const stage: ImportStage = uploadActive
    ? 1
    : localPhase === "uploading" || localPhase === "done"
      ? 3
      : (progress?.stage ?? 2);

  const imagePct =
    imageProgress && imageProgress.total > 0
      ? Math.min(100, (imageProgress.processed / imageProgress.total) * 100)
      : 0;
  const percent = uploadActive
    ? uploadPct
    : stage === 3
      ? imagePct
      : (progress?.percent ?? 0);

  const elapsedMs = startMs ? Date.now() - startMs : 0;
  const remainingMs = useMemo(
    () => estimateRemainingMs(percent, elapsedMs),
    [percent, elapsedMs],
  );

  const STAGE_WEIGHTS: Record<ImportStage, [number, number]> = useMemo(
    () => ({
      1: [0, 35],
      2: [35, 50],
      3: [85, 15],
    }),
    [],
  );
  const overallPercent = useMemo(() => {
    const [base, span] = STAGE_WEIGHTS[stage];
    return base + (Math.min(100, percent) / 100) * span;
  }, [STAGE_WEIGHTS, stage, percent]);

  const STAGE_NAMES: Record<ImportStage, string> = {
    1: t("stage1Read"),
    2: t("stage2Parse"),
    3: t("stage3Upload"),
  };

  return (
    <div className="space-y-4">
      <div className="text-center space-y-1">
        <div className="text-xs uppercase tracking-wider text-amber-500 font-semibold">
          {localPhase === "done" ? t("progressDone") : t("progressTitle")}
        </div>
        <div className="text-sm font-semibold">
          {localPhase === "done"
            ? t("progressDoneSummary")
            : uploadActive
              ? t("uploadingHeadline", { pct: Math.round(uploadPct) })
              : t("progressOverall", {
                  pct: Math.round(overallPercent),
                  eta: remainingMs
                    ? t("etaWithDuration", {
                        duration: formatDuration(remainingMs, locale),
                      })
                    : t("etaCalculating"),
                })}
        </div>
        {transport === "poll" && (
          <div className="text-micro text-muted-foreground italic">
            {t("transportPollFallback")}
          </div>
        )}
      </div>

      <Stepper stage={stage} stageNames={STAGE_NAMES} localPhase={localPhase} />

      <div className="rounded-md border border-border bg-muted/20 p-4 space-y-2">
        <div className="flex items-baseline justify-between">
          <div className="text-sm font-medium">
            {uploadActive ? t("uploadStageLabel") : STAGE_NAMES[stage]}
          </div>
          {uploadActive ? (
            <div className="text-xs font-mono text-amber-500">
              {uploadProgress!.lengthComputable
                ? t("uploadBytes", {
                    done: uploadDoneMb.toFixed(1),
                    total: uploadTotalMb.toFixed(1),
                  })
                : `${uploadDoneMb.toFixed(1)} MB`}{" "}
              ({uploadPct.toFixed(0)}%)
            </div>
          ) : stage === 3 && imageProgress && imageProgress.total > 0 ? (
            <div className="text-xs font-mono text-amber-500">
              {imageProgress.processed.toLocaleString()} /{" "}
              {imageProgress.total.toLocaleString()} ({imagePct.toFixed(1)}%)
            </div>
          ) : progress && progress.total > 0 ? (
            <div className="text-xs font-mono text-amber-500">
              {progress.processed.toLocaleString()} /{" "}
              {progress.total.toLocaleString()} ({percent.toFixed(1)}%)
            </div>
          ) : (
            <div className="text-xs font-mono text-muted-foreground">
              {fallbackStatus || t("progressWaitingSse")}
            </div>
          )}
        </div>
        <Progress
          value={
            uploadActive
              ? uploadPct
              : stage === 3 && imageProgress && imageProgress.total > 0
                ? imagePct
                : progress && progress.total > 0
                  ? percent
                  : null
          }
          indeterminate={
            !uploadActive &&
            (stage === 3
              ? !imageProgress || imageProgress.total === 0
              : !progress || progress.total === 0)
          }
          ariaLabel={uploadActive ? t("uploadStageLabel") : STAGE_NAMES[stage]}
        />

        {uploadActive && (
          <div className="text-xs text-muted-foreground">{t("uploadHint")}</div>
        )}

        {!uploadActive && stage === 2 && (
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <div className="flex items-center gap-1.5 min-w-0">
              <span className="inline-block w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse shrink-0" />
              {progress?.phase === "game_index" ? (
                <GameIndexWait />
              ) : (
                <span className="truncate">
                  {progress?.current_file
                    ? t("currentFile", { file: progress.current_file })
                    : t("currentFileWaiting")}
                </span>
              )}
            </div>
            {progress?.throughput_per_sec != null && (
              <div className="font-mono whitespace-nowrap ml-2">
                {t("throughput", {
                  rate: progress.throughput_per_sec.toFixed(0),
                })}
              </div>
            )}
          </div>
        )}

        {recentFilesDisplay.length > 1 && (
          <div className="pt-1 border-t border-border/40 text-micro text-muted-foreground space-y-0.5">
            {recentFilesDisplay.slice(1).map((f, i) => (
              <div key={`${f}-${i}`} className="truncate font-mono opacity-60">
                {f}
              </div>
            ))}
          </div>
        )}
      </div>

      <div
        className={cn("grid gap-2", compact ? "grid-cols-2" : "grid-cols-4")}
      >
        {BUCKETS.map((key) => {
          const count = progress?.counts?.[key] ?? null;
          return <CountCard key={key} bucketKey={key} count={count} />;
        })}
      </div>
    </div>
  );
}

function GameIndexWait() {
  const t = useTranslations("desktop.gameIndex");
  const { data } = useSWR<GameIndexStatus>(
    GAME_INDEX_KEY,
    () => appApi.gameIndex(),
    {
      refreshInterval: 1000,
    },
  );
  return (
    <span className="truncate">
      {t("importWaiting")}
      {data?.state === "building" && data.total > 0 && (
        <span className="ml-2 font-mono">
          {data.done}/{data.total}
        </span>
      )}
    </span>
  );
}

interface StepperProps {
  stage: ImportStage;
  stageNames: Record<ImportStage, string>;
  localPhase: "parsing" | "uploading" | "done";
}

function Stepper({ stage, stageNames, localPhase }: StepperProps) {
  const stage1Done = true;
  const stage2Done = localPhase === "done" || stage === 3;
  const stage3Done = localPhase === "done";

  return (
    <div className="flex items-center px-2">
      <StepDot
        index={1}
        label={stageNames[1]}
        active={stage === 1 && !stage1Done}
        done={stage1Done}
      />
      <Connector done={stage1Done} active={stage === 2 || stage === 3} />
      <StepDot
        index={2}
        label={stageNames[2]}
        active={stage === 2 && !stage2Done}
        done={stage2Done}
      />
      <Connector done={stage2Done} active={stage === 3} />
      <StepDot
        index={3}
        label={stageNames[3]}
        active={stage === 3 && !stage3Done}
        done={stage3Done}
      />
    </div>
  );
}

function StepDot({
  index,
  label,
  active,
  done,
}: {
  index: number;
  label: string;
  active: boolean;
  done: boolean;
}) {
  return (
    <div className="flex items-center gap-2 shrink-0">
      <div
        className={cn(
          "w-7 h-7 rounded-full border-2 flex items-center justify-center text-xs font-mono font-semibold shrink-0 transition-colors",
          done
            ? "bg-green-500/20 border-green-500 text-green-500"
            : active
              ? "bg-amber-500/20 border-amber-500 text-amber-500 animate-pulse"
              : "bg-muted/30 border-border text-muted-foreground",
        )}
      >
        {done ? (
          <svg
            className="w-3.5 h-3.5"
            fill="none"
            stroke="currentColor"
            strokeWidth="3"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path d="M20 6L9 17l-5-5" />
          </svg>
        ) : (
          index
        )}
      </div>
      <div
        className={cn(
          "text-xs font-medium",
          done
            ? "text-foreground"
            : active
              ? "text-amber-500"
              : "text-muted-foreground",
        )}
      >
        {label}
      </div>
    </div>
  );
}

function Connector({ done, active }: { done: boolean; active: boolean }) {
  return (
    <div
      className={cn(
        "flex-1 h-px mx-2 min-w-[16px]",
        done ? "bg-green-500/50" : active ? "bg-amber-500/30" : "bg-border",
      )}
    />
  );
}

function CountCard({
  bucketKey,
  count,
}: {
  bucketKey: ImportBucketKey;
  count: { done: number; total: number } | null;
}) {
  const t = useTranslations("unifiedModImport");
  const label = t(`bucket_${bucketKey}` as Parameters<typeof t>[0]);
  const hasData = count !== null && (count.done > 0 || count.total > 0);

  const flatTally = hasData && count!.total === 0;
  const complete = hasData && count!.total > 0 && count!.done >= count!.total;

  return (
    <div className="bg-muted/20 border border-border rounded p-2 text-center">
      <div className="text-micro text-muted-foreground uppercase tracking-wider">
        {label}
      </div>
      <div
        className={cn(
          "text-sm font-semibold font-mono",
          !hasData
            ? "text-muted-foreground"
            : complete
              ? "text-green-500"
              : "text-amber-500",
        )}
      >
        {!hasData ? (
          "—"
        ) : flatTally ? (
          count!.done.toLocaleString()
        ) : complete ? (
          <>
            {count!.done.toLocaleString()} <span className="text-xs">✓</span>
          </>
        ) : (
          <>
            {count!.done.toLocaleString()} / {count!.total.toLocaleString()}
          </>
        )}
      </div>
    </div>
  );
}

export type { ImportProgress };
