"use client";

import {
  AlertTriangle,
  Check,
  FileSearch,
  Info,
  LoaderCircle,
} from "lucide-react";
import { useTranslations } from "use-intl";
import { Link } from "@/i18n/navigation";

import type { FindingGroup } from "@/lib/exportCheckPresentation";
import { groupAttentionFindings } from "@/lib/exportCheckPresentation";
import type { PresentedFinding } from "@/lib/exportCheckPresentation";
import { repairHash } from "@/lib/exportRepairQueue";
import type { CheckPhase } from "@/components/export/ExportStepper";
import {
  findingRepairTarget,
  useFindingCopy,
} from "@/components/export/FindingCopy";
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

export function AttentionCard({
  phase,
  projectId,
  attention,
  groups,
  autoCount,
  passCount,
  healthCount,
  onOpenDetails,
}: {
  phase: CheckPhase;
  projectId: number;
  attention: PresentedFinding[];

  groups?: FindingGroup[];
  autoCount: number;
  passCount: number;
  healthCount: number;
  onOpenDetails: () => void;
}) {
  const t = useTranslations("export");
  const copyOf = useFindingCopy();

  const openProjectSettings = useOpenProjectSettings((s) => s.requestOpen);
  const rows = groups ?? groupAttentionFindings(attention);

  return (
    <section
      id="export-attention"
      aria-labelledby="export-attention-title"
      className="flex flex-col rounded-lg border border-border bg-card"
    >
      <div className="flex items-start justify-between gap-3 px-4 pb-3 pt-4">
        <div className="min-w-0">
          <h3
            id="export-attention-title"
            className="text-sm font-semibold text-foreground"
          >
            {t("ui.attention.title")}
          </h3>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {t("ui.attention.subtitle")}
          </p>
        </div>
        <span
          className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-semibold ${
            rows.length > 0
              ? "bg-warning/15 text-warning"
              : "bg-success/15 text-success"
          }`}
        >
          {t("ui.attention.countChip", { count: rows.length })}
        </span>
      </div>

      <div className="flex-1 border-t border-border">
        {phase === "idle" ? (
          <div className="flex items-center gap-3 px-4 py-5">
            <span
              aria-hidden
              className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-muted text-muted-foreground"
            >
              <FileSearch className="h-3.5 w-3.5" aria-hidden />
            </span>
            <div>
              <p className="text-sm font-semibold text-foreground">
                {t("ui.attention.idleTitle")}
              </p>
              <p className="text-xs text-muted-foreground">
                {t("ui.attention.idleBody")}
              </p>
            </div>
          </div>
        ) : phase === "running" ? (
          <div className="flex items-center gap-2.5 px-4 py-5 text-sm text-muted-foreground">
            <LoaderCircle className="h-4 w-4 animate-spin" aria-hidden />
            {t("ui.stepper.step1Running")}
          </div>
        ) : rows.length === 0 ? (
          <div className="flex items-center gap-3 px-4 py-5">
            <span
              aria-hidden
              className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-success/15 text-success"
            >
              <Check className="h-3.5 w-3.5" aria-hidden />
            </span>
            <div>
              <p className="text-sm font-semibold text-foreground">
                {t("ui.attention.emptyTitle")}
              </p>
              <p className="text-xs text-muted-foreground">
                {t("ui.attention.emptyBody")}
              </p>
            </div>
          </div>
        ) : (
          <ul className="divide-y divide-border">
            {rows.map((group, index) => {
              const finding = group.representative;
              const copy = copyOf(finding);

              const single = group.count === 1;

              const target = findingRepairTarget(projectId, finding.issue);
              const href =
                single && target?.kind === "href" ? target.href : null;

              const detailsHref = single
                ? null
                : `/project/${projectId}/export/repair#${repairHash(finding.issue)}`;

              const noRoute = target === null;

              const opensSettings =
                single && target?.kind === "project-settings";
              return (
                <li
                  key={`${group.code}-${index}`}
                  data-testid="export-attention-item"
                  className="grid grid-cols-[24px_minmax(0,1fr)_auto] items-start gap-3 px-4 py-3.5"
                >
                  <span
                    aria-hidden
                    className={`grid h-6 w-6 place-items-center ${
                      group.hasBlocker
                        ? "text-warning"
                        : "text-muted-foreground"
                    }`}
                  >
                    {group.hasBlocker ? (
                      <AlertTriangle className="h-4 w-4" aria-hidden />
                    ) : (
                      <Info className="h-4 w-4" aria-hidden />
                    )}
                  </span>
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                      <h4 className="text-sm font-semibold text-foreground">
                        {copy.what}
                      </h4>
                      {single ? null : (
                        <span
                          data-testid="export-attention-group-count"
                          className="rounded-full bg-muted px-2 py-0.5 text-micro font-semibold text-muted-foreground"
                        >
                          {t("ui.attention.groupAffected", {
                            count: group.count,
                          })}
                        </span>
                      )}
                    </div>
                    {copy.impact ? (
                      <p className="mt-1 text-xs text-muted-foreground">
                        {copy.impact}
                      </p>
                    ) : null}
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {finding.meta ? (
                        <span className="rounded-full border border-border px-2 py-0.5 text-micro text-muted-foreground">
                          {t(surfaceLabelKey(finding.meta.surface))}
                        </span>
                      ) : (
                        <span className="rounded-full border border-border px-2 py-0.5 text-micro text-muted-foreground">
                          {t("ui.attention.unknownCode")}
                        </span>
                      )}
                      <span className="rounded-full border border-border px-2 py-0.5 text-micro text-muted-foreground">
                        {t(severityLabelKey(String(finding.issue.severity)))}
                      </span>
                      {copy.jump ? (
                        <span className="rounded-full border border-border px-2 py-0.5 text-micro text-muted-foreground">
                          {copy.jump}
                        </span>
                      ) : null}
                    </div>
                    {noRoute ? (
                      <div
                        data-testid="export-attention-no-route"
                        className="mt-2 rounded-md border border-dashed border-border px-2.5 py-2"
                      >
                        <p className="text-xs font-semibold text-foreground">
                          {t("ui.repair.panelNoRoute")}
                        </p>
                        <p className="mt-0.5 text-micro text-muted-foreground">
                          {t("ui.repair.panelNoRouteBody")}
                        </p>
                      </div>
                    ) : null}
                  </div>
                  {opensSettings ? (
                    <button
                      type="button"
                      data-testid="export-attention-open-settings"
                      onClick={() => openProjectSettings()}
                      className="inline-flex min-h-8 items-center rounded-md border border-border bg-card px-3 text-xs font-semibold text-foreground ui-interactive ui-focus-ring hover:bg-muted/40"
                    >
                      {copy.action ?? t("ui.repair.panelOpenSettings")}
                    </button>
                  ) : href || detailsHref ? (
                    <Link
                      href={(href ?? detailsHref) as string}
                      className="inline-flex min-h-8 items-center rounded-md border border-border bg-card px-3 text-xs font-semibold text-foreground ui-interactive ui-focus-ring hover:bg-muted/40"
                    >
                      {href
                        ? (copy.action ?? t("ui.attention.repair"))
                        : t("ui.attention.viewGroup")}
                    </Link>
                  ) : null}
                </li>
              );
            })}
          </ul>
        )}
      </div>

      <div className="border-t border-border px-4 py-2.5 text-xs text-muted-foreground">
        {healthCount > 0
          ? t("ui.attention.footerHealth", {
              auto: autoCount,
              pass: passCount,
              health: healthCount,
            })
          : t("ui.attention.footer", { auto: autoCount, pass: passCount })}
        {" — "}
        <button
          type="button"
          onClick={onOpenDetails}
          className="text-accent underline-offset-2 ui-interactive ui-focus-ring hover:underline"
        >
          {t("ui.attention.viewDetails")} ⧉
        </button>
      </div>
    </section>
  );
}
