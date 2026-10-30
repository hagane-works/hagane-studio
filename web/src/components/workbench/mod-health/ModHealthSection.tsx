"use client";

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type RefObject,
} from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import {
  Activity,
  AlertTriangle,
  CalendarClock,
  ScrollText,
  Upload,
  CheckCircle2,
  RefreshCw,
  Info,
  Gamepad2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  modHealthApi,
  type PerfReport,
  type ErrorLogReport,
  type PlaytestBundleReport,
  type PlaytestBundleFiles,
  type PlaytestExpectedModIdentity,
  type PlaytestRuntimeDiagnostics,
  type ModHealthFinding,
  type ModHealthTotals,
  type StartDateLintReport,
} from "@/lib/modHealthApi";
import { ModHealthFindingList } from "./ModHealthFindingList";

const MAX_LOG_BYTES = 25 * 1024 * 1024;

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(2)} MB`;
}

function TotalsBadge({ totals }: { totals: ModHealthTotals }) {
  const t = useTranslations("modHealth");
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {totals.error > 0 ? (
        <span className="rounded-full border border-destructive/40 bg-destructive/10 px-2 py-0.5 text-xs text-destructive">
          {t("severityError")} · {totals.error}
        </span>
      ) : null}
      {totals.warning > 0 ? (
        <span className="rounded-full border border-warning/40 bg-warning/10 px-2 py-0.5 text-xs text-warning">
          {t("severityWarning")} · {totals.warning}
        </span>
      ) : null}
      {totals.info > 0 ? (
        <span className="rounded-full border border-border bg-muted/30 px-2 py-0.5 text-xs text-muted-foreground">
          {t("severityInfo")} · {totals.info}
        </span>
      ) : null}
    </div>
  );
}

function WorkingPill({ label }: { label: string }) {
  return (
    <span className="animate-pulse-ring inline-flex items-center gap-1.5 rounded-md border border-accent/60 px-2.5 py-1 text-xs text-foreground">
      {label}
    </span>
  );
}

function ErrorLogFindings({
  report,
  projectId,
}: {
  report: ErrorLogReport;
  projectId: number;
}) {
  const t = useTranslations("modHealth");
  const bb = report.by_bucket ?? {};
  const bugs = bb.D_bug5_fixed ?? 0;
  const unclassified = bb.D_unknown ?? 0;
  const gap = bb.C_gap ?? 0;

  const noise = (bb.mod_noise ?? 0) + (bb.D_triage_tail ?? 0);
  const rest = Math.max(0, report.unique_signatures - bugs);
  const hasBugs = bugs > 0;

  const secondary = [
    unclassified > 0 ? `${t("countUnclassifiedLabel")} ${unclassified}` : null,
    gap > 0 ? `${t("countGapLabel")} ${gap}` : null,
    noise > 0 ? `${t("countNoiseLabel")} ${noise}` : null,
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <>
      {hasBugs ? (
        <div className="flex items-start gap-2 rounded-lg border border-warning/40 bg-warning/10 px-3 py-2.5">
          <AlertTriangle
            className="h-4 w-4 shrink-0 mt-0.5 text-warning"
            aria-hidden
          />
          <p className="text-xs text-foreground">
            {t("logVerdictIssues", { k: bugs, m: rest })}
          </p>
        </div>
      ) : (
        <div className="flex items-start gap-2 rounded-lg border border-success/40 bg-success/5 px-3 py-2.5">
          <CheckCircle2
            className="h-4 w-4 shrink-0 mt-0.5 text-success"
            aria-hidden
          />
          <p className="text-xs text-foreground">
            {t("logVerdictClean", { n: rest })}
          </p>
        </div>
      )}

      <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
        <span
          className={`text-sm font-medium ${hasBugs ? "text-destructive" : "text-foreground"}`}
        >
          {t("countBugsLabel")} · {bugs}
        </span>
        {secondary ? (
          <span className="text-xs text-muted-foreground">· {secondary}</span>
        ) : null}
      </div>

      <p className="text-xs text-muted-foreground">
        {t("logSummary", {
          unique: report.unique_signatures,
          lines: report.lines_parsed,
          mapped: report.mapped,
        })}
      </p>

      <ModHealthFindingList
        findings={report.findings}
        projectId={projectId}
        groupBy="bucket"
      />
    </>
  );
}

type TimelineTFn = ReturnType<typeof useTranslations>;

function localizeTimelineFinding(
  f: ModHealthFinding,
  t: TimelineTFn,
): ModHealthFinding {
  const code = f.code ?? "";
  if (!code.startsWith("timeline.")) return f;
  const sub = code.slice("timeline.".length);
  const d = f.detail ?? {};
  const str = (k: string): string | null =>
    typeof d[k] === "string" && (d[k] as string) !== ""
      ? (d[k] as string)
      : null;

  let values: Record<string, string> | null;
  let hasSuggestion = true;
  switch (sub) {
    case "bookmark_date_mismatch": {
      const id = str("bookmark_id");
      const bookmarkDate = str("bookmark_date");
      const startDate = str("start_date");
      values =
        id && bookmarkDate && startDate
          ? { id, bookmarkDate, startDate }
          : null;
      break;
    }
    case "no_bookmark_for_custom_start": {
      const startDate = str("start_date");
      values = startDate ? { startDate } : null;
      break;
    }
    case "defines_conflict": {
      const definesDate = str("defines_date");
      const startDate = str("start_date");
      values = definesDate && startDate ? { definesDate, startDate } : null;
      break;
    }
    case "defines_manual": {
      const definesDate = str("defines_date");
      values = definesDate ? { definesDate } : null;
      break;
    }
    case "empty_bookmarks_replace":
      values = {};
      break;
    case "vanilla_bookmarks_hidden":
      values = {};
      hasSuggestion = false;
      break;
    case "start_date_ok": {
      const startDate = str("start_date");
      values = startDate ? { startDate } : null;
      hasSuggestion = false;
      break;
    }
    default:
      return f;
  }
  if (values === null) return f;
  return {
    ...f,
    title: t(`timelineCode_${sub}_title`),
    reason: t(`timelineCode_${sub}_reason`, values),
    suggestion: hasSuggestion
      ? t(`timelineCode_${sub}_suggestion`, values)
      : f.suggestion,
  };
}

function TimelinePanel({ projectId }: { projectId: number }) {
  const t = useTranslations("modHealth");
  const [report, setReport] = useState<StartDateLintReport | null>(null);
  const [running, setRunning] = useState(false);

  const run = useCallback(async () => {
    setRunning(true);
    try {
      const r = await modHealthApi.getStartDateLint(projectId);
      setReport(r);
    } catch (err) {
      toast.error(t("timelineFailed", { err: String(err) }));
    } finally {
      setRunning(false);
    }
  }, [projectId, t]);

  useEffect(() => {
    void run();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  const localized = report
    ? report.findings.map((f) => localizeTimelineFinding(f, t))
    : [];

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-2 min-w-0">
          <CalendarClock
            className="h-4 w-4 shrink-0 mt-0.5 text-accent"
            aria-hidden
          />
          <div className="min-w-0">
            <h3 className="text-sm font-semibold text-foreground">
              {t("timelineTitle")}
            </h3>
            <p className="text-xs text-muted-foreground">
              {t("timelineSubtitle")}
            </p>
          </div>
        </div>
        <div className="shrink-0">
          {running ? (
            <WorkingPill label={t("timelineRunning")} />
          ) : (
            <Button
              onClick={run}
              variant="outline"
              size="sm"
              className="ui-interactive"
            >
              <RefreshCw className="h-3.5 w-3.5" aria-hidden />
              {t("timelineRescan")}
            </Button>
          )}
        </div>
      </div>

      {report ? (
        <>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
            <TotalsBadge totals={report.totals} />
            {report.start_date ? (
              <span className="text-xs text-muted-foreground">
                {t("timelineStartDate", { date: report.start_date })}
              </span>
            ) : null}
          </div>
          {report.count === 0 ? (
            <div className="flex items-start gap-2 rounded-lg border border-border bg-muted/20 px-3 py-2.5">
              <Info
                className="h-4 w-4 shrink-0 mt-0.5 text-muted-foreground"
                aria-hidden
              />
              <p className="text-xs text-muted-foreground">
                {t("timelineFollowsVanilla")}
              </p>
            </div>
          ) : (
            <ModHealthFindingList findings={localized} projectId={projectId} />
          )}
        </>
      ) : running ? null : (
        <Button
          onClick={run}
          variant="outline"
          size="sm"
          className="ui-interactive"
        >
          {t("timelineRescan")}
        </Button>
      )}
    </section>
  );
}

function PerfPanel({ projectId }: { projectId: number }) {
  const t = useTranslations("modHealth");
  const [report, setReport] = useState<PerfReport | null>(null);
  const [running, setRunning] = useState(false);

  const run = useCallback(async () => {
    setRunning(true);
    try {
      const r = await modHealthApi.getPerfProfile(projectId);
      setReport(r);
    } catch (err) {
      toast.error(t("perfFailed", { err: String(err) }));
    } finally {
      setRunning(false);
    }
  }, [projectId, t]);

  const ep = report?.export_profile ?? null;

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-2 min-w-0">
          <Activity
            className="h-4 w-4 shrink-0 mt-0.5 text-accent"
            aria-hidden
          />
          <div className="min-w-0">
            <h3 className="text-sm font-semibold text-foreground">
              {t("perfTitle")}
            </h3>
            <p className="text-xs text-muted-foreground">{t("perfSubtitle")}</p>
          </div>
        </div>
        <div className="shrink-0">
          {running ? (
            <WorkingPill label={t("perfRunning")} />
          ) : (
            <Button
              onClick={run}
              variant="outline"
              size="sm"
              className="ui-interactive"
            >
              <RefreshCw className="h-3.5 w-3.5" aria-hidden />
              {t("perfRescan")}
            </Button>
          )}
        </div>
      </div>

      {report ? (
        <>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
            <TotalsBadge totals={report.totals} />
            <span className="text-xs text-muted-foreground">
              {t("perfScanned", {
                focusNodes: report.scanned.focus_nodes,
                events: report.scanned.events,
                decisions: report.scanned.decisions,
                ideas: report.scanned.ideas,
                characters: report.scanned.characters,
              })}
            </span>
          </div>

          {ep ? (
            <p className="text-xs text-muted-foreground">
              {ep.file_count != null
                ? t("perfExportProfile", {
                    ms: Math.round(ep.duration_ms),
                    size: formatBytes(ep.zip_bytes),
                    files: ep.file_count,
                  })
                : t("perfExportProfileNoFiles", {
                    ms: Math.round(ep.duration_ms),
                    size: formatBytes(ep.zip_bytes),
                  })}
            </p>
          ) : null}

          {report.count === 0 ? (
            <div className="flex items-center gap-2 rounded-lg border border-success/40 bg-success/5 px-3 py-2.5">
              <CheckCircle2
                className="h-4 w-4 shrink-0 text-success"
                aria-hidden
              />
              <p className="text-xs text-foreground">{t("perfClean")}</p>
            </div>
          ) : (
            <ModHealthFindingList
              findings={report.findings}
              projectId={projectId}
            />
          )}
        </>
      ) : running ? null : (
        <Button
          onClick={run}
          variant="outline"
          size="sm"
          className="ui-interactive"
        >
          {t("perfRescan")}
        </Button>
      )}
    </section>
  );
}

function ErrorLogPanel({ projectId }: { projectId: number }) {
  const t = useTranslations("modHealth");
  const [report, setReport] = useState<ErrorLogReport | null>(null);
  const [running, setRunning] = useState(false);

  const [uploadedBytes, setUploadedBytes] = useState<number | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  const onPick = useCallback(
    async (file: File | undefined) => {
      if (!file) return;
      if (file.size > MAX_LOG_BYTES) {
        toast.error(t("logTooLarge"));
        return;
      }
      setRunning(true);
      try {
        const r = await modHealthApi.analyzeErrorLog(projectId, file);
        setReport(r);
        setUploadedBytes(file.size);
      } catch (err) {
        toast.error(t("logFailed", { err: String(err) }));
      } finally {
        setRunning(false);
      }
    },
    [projectId, t],
  );

  const looksUnparsed =
    report != null && report.lines_parsed === 0 && (uploadedBytes ?? 0) > 0;

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-2 min-w-0">
          <ScrollText
            className="h-4 w-4 shrink-0 mt-0.5 text-accent"
            aria-hidden
          />
          <div className="min-w-0">
            <h3 className="text-sm font-semibold text-foreground">
              {t("logTitle")}
            </h3>
            <p className="text-xs text-muted-foreground">{t("logSubtitle")}</p>
          </div>
        </div>
        <div className="shrink-0">
          {running ? (
            <WorkingPill label={t("logUploading")} />
          ) : (
            <Button
              onClick={() => inputRef.current?.click()}
              variant={report ? "outline" : "default"}
              size="sm"
              className={
                report
                  ? "ui-interactive"
                  : "bg-accent text-accent-foreground ui-interactive"
              }
            >
              <Upload className="h-3.5 w-3.5" aria-hidden />
              {report ? t("logReanalyze") : t("logUploadButton")}
            </Button>
          )}
        </div>
      </div>

      <input
        ref={inputRef}
        type="file"
        accept=".log,.txt,text/plain"
        className="sr-only"
        onChange={(e) => {
          void onPick(e.target.files?.[0]);

          e.target.value = "";
        }}
      />

      {report ? (
        looksUnparsed ? (
          <div className="flex items-start gap-2 rounded-lg border border-border bg-muted/20 px-3 py-2.5">
            <Info
              className="h-4 w-4 shrink-0 mt-0.5 text-muted-foreground"
              aria-hidden
            />
            <p className="text-xs text-muted-foreground">
              {t("logNotRecognized")}
            </p>
          </div>
        ) : report.count === 0 ? (
          <>
            <div className="flex items-center gap-2 rounded-lg border border-success/40 bg-success/5 px-3 py-2.5">
              <CheckCircle2
                className="h-4 w-4 shrink-0 text-success"
                aria-hidden
              />
              <p className="text-xs text-foreground">{t("logClean")}</p>
            </div>
            <p className="text-xs text-muted-foreground">
              {t("logSummary", {
                unique: report.unique_signatures,
                lines: report.lines_parsed,
                mapped: report.mapped,
              })}
            </p>
          </>
        ) : (
          <ErrorLogFindings report={report} projectId={projectId} />
        )
      ) : running ? null : (
        <div className="flex flex-col items-center gap-2 rounded-lg border border-border bg-muted/20 px-4 py-8 text-center">
          <ScrollText className="h-6 w-6 text-muted-foreground" aria-hidden />
          <p className="text-sm font-medium text-foreground">
            {t("logEmptyTitle")}
          </p>
          <p className="max-w-md text-xs text-muted-foreground">
            {t("logEmptyHelper")}
          </p>
          <Button
            onClick={() => inputRef.current?.click()}
            size="sm"
            className="mt-1 bg-accent text-accent-foreground ui-interactive"
          >
            <Upload className="h-3.5 w-3.5" aria-hidden />
            {t("logUploadButton")}
          </Button>
        </div>
      )}
    </section>
  );
}

function PlaytestRuntimeSummary({
  diagnostics,
}: {
  diagnostics: PlaytestRuntimeDiagnostics;
}) {
  const t = useTranslations("modHealth");
  const status = diagnostics.expected_mod.status;
  const showIdentity = status !== "not_requested";
  const showEnvironment =
    diagnostics.system.active_mod_count !== null ||
    diagnostics.system.active_dlc_count !== null;
  const showLifecycle = Boolean(
    diagnostics.crash.last_sentinel || diagnostics.crash.last_observed_line,
  );
  if (!showIdentity && !showEnvironment && !showLifecycle) return null;

  const identityCopy = (() => {
    switch (status) {
      case "activated":
        return t("playtestIdentityActivated");
      case "not_activated":
        return t("playtestIdentityNotActivated");
      case "identity_collision":
        return t("playtestIdentityCollision");
      default:
        return t("playtestIdentityIndeterminate");
    }
  })();
  const identityTone =
    status === "activated"
      ? "border-success/40 bg-success/5"
      : status === "not_activated" || status === "identity_collision"
        ? "border-warning/40 bg-warning/5"
        : "border-border bg-muted/20";
  const stageLabel = (() => {
    switch (diagnostics.crash.stage) {
      case "launcher_inventory":
        return t("playtestStageLauncher");
      case "engine_boot":
        return t("playtestStageEngine");
      case "database_loading":
        return t("playtestStageDatabase");
      case "map_loading":
        return t("playtestStageMap");
      case "history_loading":
        return t("playtestStageHistory");
      case "scenario_started":
        return t("playtestStageScenario");
      case "in_game":
        return t("playtestStageInGame");
      default:
        return t("playtestStageUnknown");
    }
  })();
  const confidenceLabel =
    diagnostics.crash.confidence === "high"
      ? t("playtestConfidenceHigh")
      : diagnostics.crash.confidence === "medium"
        ? t("playtestConfidenceMedium")
        : t("playtestConfidenceLow");

  return (
    <div className="space-y-2">
      {showIdentity ? (
        <div className={`rounded-lg border px-3 py-2.5 ${identityTone}`}>
          <p className="text-xs font-medium text-foreground">
            {t("playtestIdentityTitle")}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">{identityCopy}</p>
        </div>
      ) : null}
      {showEnvironment ? (
        <div className="rounded-lg border border-border bg-muted/20 px-3 py-2.5">
          <p className="text-xs font-medium text-foreground">
            {t("playtestRuntimeEnvironmentTitle")}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {t("playtestRuntimeCounts", {
              mods: diagnostics.system.active_mod_count ?? "—",
              dlcs: diagnostics.system.active_dlc_count ?? "—",
            })}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {t("playtestRuntimeDlcEnvironmentOnly")}
          </p>
        </div>
      ) : null}
      {showLifecycle ? (
        <div className="rounded-lg border border-border bg-muted/20 px-3 py-2.5">
          <p className="text-xs font-medium text-foreground">
            {t("playtestLifecycleTitle")}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {t("playtestLifecycleStage", {
              stage: stageLabel,
              confidence: confidenceLabel,
            })}
          </p>
          {diagnostics.crash.last_sentinel ? (
            <>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("playtestLastSentinel", {
                  source: diagnostics.crash.last_sentinel.source,
                  line: diagnostics.crash.last_sentinel.line_number,
                  marker: diagnostics.crash.last_sentinel.marker,
                })}
              </p>
              <p className="mt-1 break-all font-mono text-xs text-foreground">
                {diagnostics.crash.last_sentinel.excerpt}
              </p>
            </>
          ) : null}
          {diagnostics.crash.last_observed_line ? (
            <div className="mt-2 border-t border-border pt-2">
              <p className="text-xs font-medium text-foreground">
                {t("playtestChronologyEvidence")}
              </p>
              <p className="mt-1 break-all font-mono text-xs text-foreground">
                {diagnostics.crash.last_observed_line}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("playtestChronologyEvidenceHelper")}
              </p>
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function PlaytestBundlePanel({ projectId }: { projectId: number }) {
  const t = useTranslations("modHealth");
  const [report, setReport] = useState<PlaytestBundleReport | null>(null);
  const [running, setRunning] = useState(false);
  const [files, setFiles] = useState<PlaytestBundleFiles>({});
  const [expectedMod, setExpectedMod] = useState<PlaytestExpectedModIdentity>(
    {},
  );
  const gameRef = useRef<HTMLInputElement | null>(null);
  const excRef = useRef<HTMLInputElement | null>(null);
  const dlcRef = useRef<HTMLInputElement | null>(null);
  const systemRef = useRef<HTMLInputElement | null>(null);
  const inventoryRef = useRef<HTMLInputElement | null>(null);

  const hasAny = Boolean(
    files.gameLog ||
    files.exceptionTxt ||
    files.dlcLoadJson ||
    files.systemLog ||
    files.modInventoryJson,
  );
  const identityStatus = report?.runtime_diagnostics?.expected_mod.status;
  const runtimeNeedsAttention =
    identityStatus === "not_activated" ||
    identityStatus === "identity_collision";

  const pick = useCallback(
    (key: keyof PlaytestBundleFiles, file: File | undefined) => {
      if (!file) return;
      if (file.size > MAX_LOG_BYTES) {
        toast.error(t("logTooLarge"));
        return;
      }
      setFiles((prev) => ({ ...prev, [key]: file }));
    },
    [t],
  );

  const analyze = useCallback(async () => {
    if (!hasAny) return;
    setRunning(true);
    try {
      const r = await modHealthApi.analyzePlaytestBundle(
        projectId,
        files,
        expectedMod,
      );
      setReport(r);
    } catch (err) {
      toast.error(t("playtestFailed", { err: String(err) }));
    } finally {
      setRunning(false);
    }
  }, [projectId, files, expectedMod, hasAny, t]);

  const slots: {
    key: keyof PlaytestBundleFiles;
    label: string;
    ref: RefObject<HTMLInputElement | null>;
    accept: string;
  }[] = [
    {
      key: "gameLog",
      label: t("playtestSlotGameLog"),
      ref: gameRef,
      accept: ".log,.txt,text/plain",
    },
    {
      key: "exceptionTxt",
      label: t("playtestSlotException"),
      ref: excRef,
      accept: ".txt,text/plain",
    },
    {
      key: "dlcLoadJson",
      label: t("playtestSlotDlc"),
      ref: dlcRef,
      accept: ".json,application/json",
    },
    {
      key: "systemLog",
      label: t("playtestSlotSystemLog"),
      ref: systemRef,
      accept: ".log,.txt,text/plain",
    },
    {
      key: "modInventoryJson",
      label: t("playtestSlotModInventory"),
      ref: inventoryRef,
      accept: ".json,application/json",
    },
  ];

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-2 min-w-0">
          <Gamepad2
            className="h-4 w-4 shrink-0 mt-0.5 text-accent"
            aria-hidden
          />
          <div className="min-w-0">
            <h3 className="text-sm font-semibold text-foreground">
              {t("playtestTitle")}
            </h3>
            <p className="text-xs text-muted-foreground">
              {t("playtestRuntimeSubtitle")}
            </p>
          </div>
        </div>
        <div className="shrink-0">
          {running ? (
            <WorkingPill label={t("playtestAnalyzing")} />
          ) : (
            <Button
              onClick={analyze}
              disabled={!hasAny}
              variant={report ? "outline" : "default"}
              size="sm"
              className={
                report || !hasAny
                  ? "ui-interactive"
                  : "bg-accent text-accent-foreground ui-interactive"
              }
            >
              <Upload className="h-3.5 w-3.5" aria-hidden />
              {report ? t("playtestReanalyze") : t("playtestAnalyze")}
            </Button>
          )}
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        {slots.map((s) => {
          const picked = files[s.key];
          return (
            <div key={s.key} className="flex items-center gap-1.5">
              <Button
                onClick={() => s.ref.current?.click()}
                variant="outline"
                size="sm"
                className="ui-interactive"
              >
                {picked ? (
                  <CheckCircle2
                    className="h-3.5 w-3.5 text-success"
                    aria-hidden
                  />
                ) : (
                  <Upload className="h-3.5 w-3.5" aria-hidden />
                )}
                {s.label}
              </Button>
              {picked ? (
                <span className="max-w-[10rem] truncate text-xs text-muted-foreground">
                  {picked.name}
                </span>
              ) : null}
              <input
                ref={s.ref}
                type="file"
                accept={s.accept}
                className="sr-only"
                onChange={(e) => {
                  pick(s.key, e.target.files?.[0]);
                  e.target.value = "";
                }}
              />
            </div>
          );
        })}
      </div>

      <div className="rounded-lg border border-border bg-muted/20 p-3">
        <p className="text-xs font-medium text-foreground">
          {t("playtestExpectedIdentityTitle")}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("playtestExpectedIdentityHelper")}
        </p>
        <div className="mt-3 grid grid-cols-3 gap-2">
          <label className="text-xs text-foreground">
            {t("playtestExpectedName")}
            <Input
              value={expectedMod.name ?? ""}
              onChange={(event) =>
                setExpectedMod((current) => ({
                  ...current,
                  name: event.target.value,
                }))
              }
              placeholder={t("playtestExpectedNamePlaceholder")}
              className="mt-1 h-8 text-xs"
            />
          </label>
          <label className="text-xs text-foreground">
            {t("playtestExpectedPath")}
            <Input
              value={expectedMod.path ?? ""}
              onChange={(event) =>
                setExpectedMod((current) => ({
                  ...current,
                  path: event.target.value,
                }))
              }
              placeholder={t("playtestExpectedPathPlaceholder")}
              className="mt-1 h-8 text-xs"
            />
          </label>
          <label className="text-xs text-foreground">
            {t("playtestExpectedBuildId")}
            <Input
              value={expectedMod.buildId ?? ""}
              onChange={(event) =>
                setExpectedMod((current) => ({
                  ...current,
                  buildId: event.target.value,
                }))
              }
              placeholder={t("playtestExpectedBuildIdPlaceholder")}
              className="mt-1 h-8 text-xs"
            />
          </label>
        </div>
      </div>

      {report ? (
        <>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
            <TotalsBadge totals={report.totals} />
            {report.loaded_date ? (
              <span className="text-xs text-muted-foreground">
                {report.built_date
                  ? t("playtestDates", {
                      built: report.built_date,
                      loaded: report.loaded_date,
                    })
                  : t("playtestLoadedOnly", { loaded: report.loaded_date })}
              </span>
            ) : null}
          </div>
          {report.runtime_diagnostics ? (
            <PlaytestRuntimeSummary diagnostics={report.runtime_diagnostics} />
          ) : null}
          {report.count === 0 && !runtimeNeedsAttention ? (
            <div className="flex items-center gap-2 rounded-lg border border-success/40 bg-success/5 px-3 py-2.5">
              <CheckCircle2
                className="h-4 w-4 shrink-0 text-success"
                aria-hidden
              />
              <p className="text-xs text-foreground">
                {t("playtestNoIssueFound")}
              </p>
            </div>
          ) : report.count > 0 ? (
            <ModHealthFindingList
              findings={report.findings}
              projectId={projectId}
            />
          ) : null}
        </>
      ) : running ? null : (
        <div className="flex flex-col items-center gap-2 rounded-lg border border-border bg-muted/20 px-4 py-6 text-center">
          <Gamepad2 className="h-6 w-6 text-muted-foreground" aria-hidden />
          <p className="text-sm font-medium text-foreground">
            {t("playtestEmptyTitle")}
          </p>
          <p className="max-w-md text-xs text-muted-foreground">
            {t("playtestRuntimeEmptyHelper")}
          </p>
        </div>
      )}
    </section>
  );
}

export function ModHealthSection({ projectId }: { projectId: number }) {
  const t = useTranslations("modHealth");
  return (
    <section className="space-y-3">
      <div>
        <h3 className="text-sm font-semibold text-foreground">
          {t("sectionTitle")}
        </h3>
        <p className="text-xs text-muted-foreground">{t("sectionSubtitle")}</p>
      </div>
      <PerfPanel projectId={projectId} />
      <TimelinePanel projectId={projectId} />
      <ErrorLogPanel projectId={projectId} />
      <PlaytestBundlePanel projectId={projectId} />
    </section>
  );
}
