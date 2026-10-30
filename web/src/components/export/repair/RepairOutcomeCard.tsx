"use client";

import { Check, CircleAlert, Sparkles } from "lucide-react";
import { useTranslations } from "use-intl";

import { Link } from "@/i18n/navigation";

export type RepairOutcome = "done" | "empty" | "failed";

const ICONS = {
  done: Check,
  empty: Sparkles,
  failed: CircleAlert,
} as const;

export function RepairOutcomeCard({
  outcome,
  projectId,
}: {
  outcome: RepairOutcome;
  projectId: number;
}) {
  const t = useTranslations("export");
  const Icon = ICONS[outcome];
  const warn = outcome === "failed";
  const titleKey =
    outcome === "done"
      ? "ui.repair.doneTitle"
      : outcome === "empty"
        ? "ui.repair.emptyTitle"
        : "ui.repair.loadFailedTitle";
  const bodyKey =
    outcome === "done"
      ? "ui.repair.doneBody"
      : outcome === "empty"
        ? "ui.repair.emptyBody"
        : "ui.repair.loadFailedBody";

  return (
    <section
      data-testid={`repair-outcome-${outcome}`}
      className="rounded-lg border border-border bg-card p-8 text-center"
    >
      <span
        aria-hidden
        className={`mx-auto grid h-11 w-11 place-items-center rounded-full ${
          warn ? "bg-warning/15 text-warning" : "bg-success/15 text-success"
        }`}
      >
        <Icon className="h-5 w-5" aria-hidden />
      </span>
      <h2 className="mt-3 text-base font-semibold text-foreground">
        {t(titleKey)}
      </h2>
      <p className="mx-auto mt-1.5 max-w-md text-sm text-muted-foreground">
        {t(bodyKey)}
      </p>
      <Link
        href={`/project/${projectId}/export`}
        className="mt-5 inline-flex min-h-10 items-center rounded-lg border border-accent bg-accent px-4 text-sm font-semibold text-accent-foreground ui-interactive ui-focus-ring hover:bg-accent/90"
      >
        {t("ui.repair.backToExport")}
      </Link>
    </section>
  );
}
