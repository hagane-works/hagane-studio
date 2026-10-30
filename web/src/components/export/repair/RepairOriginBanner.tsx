"use client";

import { CornerUpLeft } from "lucide-react";
import { useTranslations } from "use-intl";

import { Link } from "@/i18n/navigation";

export function RepairOriginBanner({ projectId }: { projectId: number }) {
  const t = useTranslations("export");

  return (
    <section className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border bg-card px-4 py-3">
      <div className="flex min-w-0 items-start gap-2.5">
        <CornerUpLeft
          className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground"
          aria-hidden
        />
        <div className="min-w-0">
          <p className="text-sm font-semibold text-foreground">
            {t("ui.repair.originTitle")}
          </p>
          <p className="text-xs text-muted-foreground">
            {t("ui.repair.originSub")}
          </p>
        </div>
      </div>
      <Link
        href={`/project/${projectId}/export`}
        className="inline-flex min-h-9 shrink-0 items-center rounded-lg border border-border bg-card px-3.5 text-xs font-semibold text-foreground ui-interactive ui-focus-ring hover:bg-muted/40"
      >
        {t("ui.repair.backToExport")}
      </Link>
    </section>
  );
}
