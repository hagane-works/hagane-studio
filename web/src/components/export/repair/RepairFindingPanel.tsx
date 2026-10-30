"use client";

import {
  ExternalLink,
  Info,
  LoaderCircle,
  RefreshCw,
  Settings2,
} from "lucide-react";
import { useTranslations } from "use-intl";

import { Link } from "@/i18n/navigation";
import {
  findingRepairTarget,
  useFindingCopy,
} from "@/components/export/FindingCopy";
import type { RepairQueueItem } from "@/lib/exportRepairQueue";
import { repairRoutePath } from "@/lib/exportRepairQueue";
import { useOpenProjectSettings } from "@/lib/store/openProjectSettingsStore";

function surfaceLabelKey(surface: string): string {
  switch (surface) {
    case "playtest":
      return "ui.surface.playtest";
    case "import":
      return "ui.surface.import";
    case "ai":
      return "ui.surface.ai";
    case "workbench":
      return "ui.surface.workbench";
    default:
      return "ui.surface.export";
  }
}

function severityLabelKey(severity: string): string {
  if (severity === "blocker") return "ui.severity.blocker";
  if (severity === "info") return "ui.severity.info";
  return "ui.severity.warning";
}

export function RepairFindingPanel({
  projectId,
  item,
  index,
  total,
  rechecking,
  recheckFailed,
  onRecheck,
  onNext,
  hasNext,
}: {
  projectId: number;
  item: RepairQueueItem;
  index: number;
  total: number;
  rechecking: boolean;
  recheckFailed: boolean;
  onRecheck: () => void;
  onNext: () => void;
  hasNext: boolean;
}) {
  const t = useTranslations("export");
  const copyOf = useFindingCopy();
  const copy = copyOf(item.finding);
  const { issue, meta } = item.finding;
  const target = findingRepairTarget(projectId, issue);
  const openProjectSettings = useOpenProjectSettings((s) => s.requestOpen);
  const routePath = repairRoutePath(issue);
  const resolved = item.resolved;

  const targetTone = resolved
    ? "border-success/60 bg-success/10"
    : "border-warning/60 bg-warning/10";
  const targetLabelTone = resolved ? "text-success" : "text-warning";

  return (
    <article className="overflow-hidden rounded-lg border border-border bg-card">
      <header className="flex flex-wrap items-start justify-between gap-3 border-b border-border px-5 py-4">
        <div className="min-w-0">
          <p className="text-micro text-muted-foreground">
            {t("ui.repair.itemPosition", { index: index + 1, total })}
          </p>
          <h2 className="mt-0.5 text-base font-semibold text-foreground">
            {copy.what}
          </h2>
          <div className="mt-2 flex flex-wrap gap-1.5">
            <span className="rounded-full border border-border px-2 py-0.5 text-micro text-muted-foreground">
              {meta
                ? t(surfaceLabelKey(meta.surface))
                : t("ui.attention.unknownCode")}
            </span>

            {meta ? null : (
              <span className="inline-flex items-center gap-1 rounded-full border border-border px-2 py-0.5 text-micro text-muted-foreground">
                {t("ui.repair.codeLabel")}
                <code className="font-mono">{issue.code}</code>
              </span>
            )}
            <span className="rounded-full border border-border px-2 py-0.5 text-micro text-muted-foreground">
              {t(severityLabelKey(String(issue.severity)))}
            </span>
            {resolved ? (
              <span className="rounded-full border border-success/50 px-2 py-0.5 text-micro text-success">
                {t("ui.repair.stateResolved")}
              </span>
            ) : null}
          </div>
        </div>
        <span className="shrink-0 rounded-full border border-border px-2 py-0.5 font-mono text-micro text-muted-foreground">
          {routePath ?? "—"}
        </span>
      </header>

      <div className="flex items-start gap-2.5 border-b border-accent/30 bg-accent/5 px-5 py-3">
        <Info className="mt-0.5 h-4 w-4 shrink-0 text-accent" aria-hidden />
        <div className="min-w-0">
          <p className="text-xs font-semibold text-foreground">
            {t("ui.repair.panelImpact")}
          </p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {copy.impact ?? t("ui.repair.panelImpactUnknown")}
          </p>
        </div>
      </div>

      <div className="space-y-1 p-4">
        <div
          className={`grid grid-cols-1 gap-2 rounded-lg border p-3.5 sm:grid-cols-[170px_minmax(0,1fr)] sm:gap-4 ${targetTone}`}
        >
          <div className="min-w-0">
            <p className={`text-xs font-semibold ${targetLabelTone}`}>
              {t("ui.repair.panelWhere")}
            </p>
            <p className="mt-0.5 text-micro text-muted-foreground">
              {copy.jump ?? t("ui.repair.panelWhereUnknown")}
            </p>
          </div>
          <div className="min-w-0">
            {target?.kind === "href" ? (
              <>
                <Link
                  href={target.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  data-testid="repair-open-workbench"
                  className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-accent bg-accent px-3.5 text-xs font-semibold text-accent-foreground ui-interactive ui-focus-ring hover:bg-accent/90"
                >
                  {copy.action ?? t("ui.repair.panelOpen")}
                  <ExternalLink className="h-3.5 w-3.5" aria-hidden />
                </Link>
                <p className="mt-2 text-micro text-muted-foreground">
                  {t("ui.repair.panelOpenHint")}
                </p>
              </>
            ) : target?.kind === "project-settings" ? (
              <>
                <button
                  type="button"
                  onClick={() => openProjectSettings()}
                  data-testid="repair-open-settings"
                  className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-accent bg-accent px-3.5 text-xs font-semibold text-accent-foreground ui-interactive ui-focus-ring hover:bg-accent/90"
                >
                  {copy.action ?? t("ui.repair.panelOpenSettings")}
                  <Settings2 className="h-3.5 w-3.5" aria-hidden />
                </button>
                <p className="mt-2 text-micro text-muted-foreground">
                  {t("ui.repair.panelOpenDialogHint")}
                </p>
              </>
            ) : (
              <div data-testid="repair-no-route">
                <p className="text-xs font-semibold text-foreground">
                  {t("ui.repair.panelNoRoute")}
                </p>
                <p className="mt-1 text-micro text-muted-foreground">
                  {t("ui.repair.panelNoRouteBody")}
                </p>
              </div>
            )}
          </div>
        </div>

        {copy.verbatim ? null : (
          <div className="grid grid-cols-1 gap-2 rounded-lg border border-transparent p-3.5 sm:grid-cols-[170px_minmax(0,1fr)] sm:gap-4">
            <div className="min-w-0">
              <p className="text-xs font-semibold text-foreground">
                {t("ui.repair.panelRaw")}
              </p>
              <p className="mt-0.5 font-mono text-micro text-muted-foreground">
                {issue.code}
              </p>
            </div>
            <div className="min-w-0">
              <p className="text-xs text-muted-foreground">{issue.message}</p>
              {issue.fix_hint ? (
                <p className="mt-1 text-micro text-muted-foreground">
                  {issue.fix_hint}
                </p>
              ) : null}
            </div>
          </div>
        )}
      </div>

      <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-border px-4 py-3">
        <p
          className={`text-xs ${resolved ? "text-success" : "text-muted-foreground"}`}
        >
          {recheckFailed
            ? t("ui.repair.recheckFailed")
            : resolved
              ? t("ui.repair.panelResolved")
              : t("ui.repair.panelHint")}
        </p>
        <div className="flex flex-wrap gap-2">
          {hasNext ? (
            <button
              type="button"
              onClick={onNext}
              className="inline-flex min-h-9 items-center rounded-lg border border-border bg-card px-3.5 text-xs font-semibold text-foreground ui-interactive ui-focus-ring hover:bg-muted/40"
            >
              {t("ui.repair.next")}
            </button>
          ) : null}
          <button
            type="button"
            onClick={onRecheck}
            disabled={rechecking}
            data-testid="repair-recheck"
            className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-accent bg-accent px-3.5 text-xs font-semibold text-accent-foreground ui-interactive ui-focus-ring hover:bg-accent/90 disabled:opacity-70"
          >
            {rechecking ? (
              <LoaderCircle className="h-3.5 w-3.5 animate-spin" aria-hidden />
            ) : (
              <RefreshCw className="h-3.5 w-3.5" aria-hidden />
            )}
            {rechecking
              ? t("ui.repair.recheckRunning")
              : t("ui.repair.recheck")}
          </button>
        </div>
      </footer>
    </article>
  );
}
