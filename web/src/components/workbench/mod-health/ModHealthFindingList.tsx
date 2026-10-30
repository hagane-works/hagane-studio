"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { Link } from "@/i18n/navigation";
import {
  AlertTriangle,
  Info,
  ChevronRight,
  ChevronDown,
  FileText,
  ArrowUpRight,
} from "lucide-react";
import type { ModHealthFinding, ModHealthSeverity } from "@/lib/modHealthApi";
import { InfoTooltip, TooltipProvider } from "@/components/ui/tooltip";
import { entityHref, kindLabelKey } from "./entityHref";

const MAX_ROWS_PER_GROUP = 50;

const PERF_STATIC_CODES = new Set<string>([
  "perf.nested_scope_loop",
  "perf.nested_scope_loop_coldpath",
  "perf.hot_scope_loop",
  "perf.heavy_trigger_fanout",
  "perf.mtth_no_floor",
  "perf.expensive_available_recheck",
  "perf.oversized_focus_tree",
  "perf.oversized_event_file",
  "perf.oversized_decision_category",
  "perf.total_focus_weight",
]);

const LOG_BUCKETS = new Set<string>([
  "D_unknown",
  "D_bug5_fixed",
  "C_gap",
  "D_triage_tail",
  "mod_noise",
]);

function logBucketOf(code: string): string | null {
  if (!code.startsWith("log.")) return null;
  const bucket = code.slice("log.".length).split(".")[0];
  return LOG_BUCKETS.has(bucket) ? bucket : null;
}

interface SeverityStyle {
  band: string;

  accent: string;
  icon: typeof AlertTriangle;
}

const SEVERITY_STYLE: Record<ModHealthSeverity, SeverityStyle> = {
  error: {
    band: "border-destructive/40 bg-destructive/5",
    accent: "text-destructive",
    icon: AlertTriangle,
  },
  warning: {
    band: "border-warning/40 bg-warning/10",
    accent: "text-warning",
    icon: AlertTriangle,
  },
  info: {
    band: "border-border bg-muted/20",
    accent: "text-muted-foreground",
    icon: Info,
  },
};

function FindingRow({
  finding,
  projectId,
  forceNeutral = false,
}: {
  finding: ModHealthFinding;
  projectId: number;

  forceNeutral?: boolean;
}) {
  const t = useTranslations("modHealth");
  const [open, setOpen] = useState(false);
  const style = forceNeutral
    ? SEVERITY_STYLE.info
    : (SEVERITY_STYLE[finding.severity] ?? SEVERITY_STYLE.info);
  const Icon = style.icon;

  const code = finding.code ?? "";
  const logBucket = logBucketOf(code);
  let displayTitle = finding.title;
  let displaySuggestion = finding.suggestion;
  if (PERF_STATIC_CODES.has(code)) {
    displayTitle = t(`perfCode_${code.slice("perf.".length)}_title`);
    displaySuggestion = t(`perfCode_${code.slice("perf.".length)}_suggestion`);
  } else if (logBucket) {
    const headline = t(`logBucket_${logBucket}`);
    const sepIdx = finding.title.indexOf(": ");
    const label = sepIdx >= 0 ? finding.title.slice(sepIdx + 2) : "";
    displayTitle = label ? `${headline}: ${label}` : headline;
  }

  const href = entityHref(projectId, finding.entity_ref);
  const ref = finding.entity_ref;
  const rawLine =
    typeof finding.detail?.raw_line === "string"
      ? (finding.detail.raw_line as string)
      : null;
  const loggedCount =
    typeof finding.detail?.raw_count === "number"
      ? (finding.detail.raw_count as number)
      : null;

  return (
    <div className={`rounded-lg border p-3 ${style.band}`}>
      <div className="flex items-start gap-2.5">
        <Icon
          className={`h-4 w-4 shrink-0 mt-0.5 ${style.accent}`}
          aria-hidden
        />
        <div className="min-w-0 flex-1 space-y-1.5">
          <p className={`text-sm font-medium ${style.accent}`}>
            {displayTitle}
          </p>

          <p className="text-xs text-muted-foreground">
            <span className="font-medium text-foreground/80">
              {t("findingReason")}:{" "}
            </span>
            {finding.reason}
          </p>

          {displaySuggestion ? (
            <p className="text-xs text-muted-foreground">
              <span className="font-medium text-foreground/80">
                {t("findingSuggestion")}:{" "}
              </span>
              {displaySuggestion}
            </p>
          ) : null}

          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 pt-0.5">
            {ref && href ? (
              <Link
                href={href}
                className="inline-flex items-center gap-1 text-xs text-accent hover:underline ui-interactive ui-focus-ring rounded"
              >
                <ArrowUpRight className="h-3 w-3 shrink-0" aria-hidden />
                {ref.name
                  ? t("openEntityNamed", { name: ref.name })
                  : t("openEntity", { kind: t(kindLabelKey(ref.kind)) })}
                {ref.country_tag ? (
                  <span className="text-muted-foreground">
                    · {ref.country_tag}
                  </span>
                ) : null}
              </Link>
            ) : ref ? (
              <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
                {t(kindLabelKey(ref.kind))}
                {ref.entity_id ? (
                  <code className="font-mono text-foreground/80">
                    {ref.entity_id}
                  </code>
                ) : null}
              </span>
            ) : (
              <span className="text-xs text-muted-foreground italic">
                {t("unmappedEntity")}
              </span>
            )}

            {finding.file ? (
              <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
                <FileText className="h-3 w-3 shrink-0" aria-hidden />
                <code className="font-mono">{finding.file}</code>
              </span>
            ) : null}

            {loggedCount && loggedCount > 1 ? (
              <span className="text-xs text-muted-foreground">
                {t("findingLoggedCount", { count: loggedCount })}
              </span>
            ) : null}
          </div>

          {rawLine || Object.keys(finding.detail ?? {}).length > 0 ? (
            <div className="pt-0.5">
              <button
                type="button"
                onClick={() => setOpen((v) => !v)}
                className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground ui-interactive ui-focus-ring rounded"
                aria-expanded={open}
              >
                {open ? (
                  <ChevronDown className="h-3 w-3 shrink-0" aria-hidden />
                ) : (
                  <ChevronRight className="h-3 w-3 shrink-0" aria-hidden />
                )}
                {open ? t("hideDetails") : t("showDetails")}
              </button>
              {open ? (
                <div className="mt-1.5 space-y-1.5 rounded-md border border-border bg-card p-2">
                  {rawLine ? (
                    <div>
                      <p className="text-micro uppercase tracking-widest text-muted-foreground">
                        {t("findingRawLine")}
                      </p>
                      <pre className="mt-0.5 whitespace-pre-wrap break-all font-mono text-xs text-foreground/90">
                        {rawLine}
                      </pre>
                    </div>
                  ) : null}
                  <div>
                    <p className="text-micro uppercase tracking-widest text-muted-foreground">
                      {t("findingDetails")}
                    </p>
                    <pre className="mt-0.5 whitespace-pre-wrap break-all font-mono text-xs text-muted-foreground">
                      {JSON.stringify(finding.detail, null, 2)}
                    </pre>
                  </div>
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

const GROUP_ORDER: { severity: ModHealthSeverity; titleKey: string }[] = [
  { severity: "error", titleKey: "groupErrorsTitle" },
  { severity: "warning", titleKey: "groupWarningsTitle" },
  { severity: "info", titleKey: "groupInfoTitle" },
];

type BucketTone = "actionable" | "neutral";

const BUCKET_GROUP_ORDER: {
  bucket: string;
  tone: BucketTone;
  defaultOpen: boolean;
}[] = [
  { bucket: "D_bug5_fixed", tone: "actionable", defaultOpen: true },
  { bucket: "C_gap", tone: "neutral", defaultOpen: false },
  { bucket: "D_triage_tail", tone: "neutral", defaultOpen: false },
  { bucket: "D_unknown", tone: "neutral", defaultOpen: false },
  { bucket: "mod_noise", tone: "neutral", defaultOpen: false },
];

function bucketOf(finding: ModHealthFinding): string {
  const b = finding.detail?.bucket;
  if (typeof b === "string" && LOG_BUCKETS.has(b)) return b;
  return logBucketOf(finding.code ?? "") ?? "D_unknown";
}

function BucketGroup({
  bucket,
  tone,
  defaultOpen,
  findings,
  projectId,
}: {
  bucket: string;
  tone: BucketTone;
  defaultOpen: boolean;
  findings: ModHealthFinding[];
  projectId: number;
}) {
  const t = useTranslations("modHealth");
  const [open, setOpen] = useState(defaultOpen);
  const actionable = tone === "actionable";
  const HeaderIcon = actionable ? AlertTriangle : Info;
  const band = actionable
    ? "border-destructive/40 bg-destructive/5"
    : "border-border bg-muted/30";
  const titleColor = actionable ? "text-destructive" : "text-muted-foreground";

  const shown = findings.slice(0, MAX_ROWS_PER_GROUP);
  const hidden = findings.length - shown.length;

  return (
    <div className={`rounded-lg border ${band}`}>
      <div className="flex items-center gap-1.5 px-3 py-2">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          className="flex min-w-0 flex-1 items-center gap-2 text-left ui-interactive ui-focus-ring rounded"
        >
          {open ? (
            <ChevronDown
              className="h-3.5 w-3.5 shrink-0 text-muted-foreground"
              aria-hidden
            />
          ) : (
            <ChevronRight
              className="h-3.5 w-3.5 shrink-0 text-muted-foreground"
              aria-hidden
            />
          )}
          <HeaderIcon
            className={`h-4 w-4 shrink-0 ${titleColor}`}
            aria-hidden
          />
          <span className={`truncate text-sm font-medium ${titleColor}`}>
            {t(`bucketGroup_${bucket}_title`)}
          </span>
          <span className="shrink-0 rounded-full bg-muted px-2 text-xs text-muted-foreground">
            {findings.length}
          </span>
        </button>

        <InfoTooltip
          content={t(`bucketGroup_${bucket}_hint`)}
          className="shrink-0"
        />
      </div>
      {open ? (
        <div className="space-y-2 px-3 pb-3">
          {shown.map((f, i) => (
            <FindingRow
              key={`${f.code}-${i}`}
              finding={f}
              projectId={projectId}
              forceNeutral={!actionable}
            />
          ))}
          {hidden > 0 ? (
            <p className="text-xs text-muted-foreground italic">
              {t("findingsTruncated", { count: hidden })}
            </p>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function SeverityGroup({
  severity,
  titleKey,
  findings,
  projectId,
}: {
  severity: ModHealthSeverity;
  titleKey: string;
  findings: ModHealthFinding[];
  projectId: number;
}) {
  const t = useTranslations("modHealth");

  const [open, setOpen] = useState(
    severity === "error" || findings.length <= 4,
  );

  const [prevCount, setPrevCount] = useState(findings.length);
  if (prevCount !== findings.length) {
    setPrevCount(findings.length);
    setOpen(severity === "error" || findings.length <= 4);
  }
  const shown = findings.slice(0, MAX_ROWS_PER_GROUP);
  const hidden = findings.length - shown.length;
  const accent =
    severity === "error"
      ? "text-destructive"
      : severity === "warning"
        ? "text-warning"
        : "text-muted-foreground";
  return (
    <div className="space-y-2">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className="flex w-full items-center gap-1.5 text-left ui-interactive ui-focus-ring rounded"
      >
        {open ? (
          <ChevronDown
            className="h-3.5 w-3.5 shrink-0 text-muted-foreground"
            aria-hidden
          />
        ) : (
          <ChevronRight
            className="h-3.5 w-3.5 shrink-0 text-muted-foreground"
            aria-hidden
          />
        )}
        <span
          className={`text-micro uppercase tracking-widest font-semibold ${accent}`}
        >
          {t(titleKey)}
        </span>
        <span className="shrink-0 rounded-full bg-muted px-2 text-xs text-muted-foreground">
          {findings.length}
        </span>
      </button>
      {open ? (
        <div className="space-y-2">
          {shown.map((f, i) => (
            <FindingRow
              key={`${f.code}-${i}`}
              finding={f}
              projectId={projectId}
            />
          ))}
          {hidden > 0 ? (
            <p className="text-xs text-muted-foreground italic">
              {t("findingsTruncated", { count: hidden })}
            </p>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

export function ModHealthFindingList({
  findings,
  projectId,
  groupBy = "severity",
}: {
  findings: ModHealthFinding[];
  projectId: number;

  groupBy?: "severity" | "bucket";
}) {
  if (findings.length === 0) return null;

  if (groupBy === "bucket") {
    return (
      <TooltipProvider>
        <div className="space-y-2">
          {BUCKET_GROUP_ORDER.map(({ bucket, tone, defaultOpen }) => {
            const group = findings.filter((f) => bucketOf(f) === bucket);
            if (group.length === 0) return null;
            return (
              <BucketGroup
                key={bucket}
                bucket={bucket}
                tone={tone}
                defaultOpen={defaultOpen}
                findings={group}
                projectId={projectId}
              />
            );
          })}
        </div>
      </TooltipProvider>
    );
  }

  return (
    <div className="space-y-4">
      {GROUP_ORDER.map(({ severity, titleKey }) => {
        const group = findings.filter((f) => f.severity === severity);
        if (group.length === 0) return null;

        return (
          <SeverityGroup
            key={severity}
            severity={severity}
            titleKey={titleKey}
            findings={group}
            projectId={projectId}
          />
        );
      })}
    </div>
  );
}
