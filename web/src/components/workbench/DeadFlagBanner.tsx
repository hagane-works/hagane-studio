"use client";

import { useTranslations } from "use-intl";
import type { DeadFlagReport } from "@/lib/api";

export function DeadFlagBanner({ report }: { report: DeadFlagReport | null }) {
  const t = useTranslations("deadFlagBanner");
  if (!report || report.dead_flags.length === 0) {
    return null;
  }
  return (
    <div className="rounded-md border border-amber-300 bg-amber-50 dark:border-amber-700/60 dark:bg-amber-900/20 px-3 py-2 text-xs">
      <details>
        <summary className="cursor-pointer font-semibold text-amber-800 dark:text-amber-200">
          {t("summary", { count: report.dead_flags.length })}
          <span className="ml-2 font-normal text-amber-700/80 dark:text-amber-200/80">
            {t("summaryNote")}
          </span>
        </summary>
        <ul className="mt-2 space-y-1 pl-4">
          {report.dead_flags.map((df) => (
            <li key={df.flag_id} className="leading-relaxed">
              <code className="font-mono text-amber-900 dark:text-amber-100">
                {df.flag_id}
              </code>
              <span className="ml-2 text-muted-foreground">
                {df.set_locations.join(" · ")}
              </span>
            </li>
          ))}
        </ul>
        <p className="mt-2 text-xs text-amber-700/80 dark:text-amber-200/80">
          {t.rich("fix", {
            flag: "<flag>",
            code: (chunks) => <code className="mx-1 font-mono">{chunks}</code>,
          })}
        </p>
      </details>
    </div>
  );
}
