"use client";

import { Check } from "lucide-react";
import { useTranslations } from "use-intl";

import { Progress } from "@/components/ui/progress";
import {
  useFindingCopy,
  findingRepairTarget,
} from "@/components/export/FindingCopy";
import type { RepairQueueItem } from "@/lib/exportRepairQueue";
import { groupRepairQueue, repairProgress } from "@/lib/exportRepairQueue";

export function RepairQueueList({
  projectId,
  items,
  activeIndex,
  onSelect,
}: {
  projectId: number;
  items: RepairQueueItem[];
  activeIndex: number;
  onSelect: (index: number) => void;
}) {
  const t = useTranslations("export");
  const copyOf = useFindingCopy();
  const progress = repairProgress(items);

  const indexByKey = new Map(items.map((item, index) => [item.key, index]));
  const groups = groupRepairQueue(items).map((group) => {
    const indices = group.rows.map((row) => indexByKey.get(row.key) ?? -1);
    const firstOpen = group.rows.findIndex((row) => !row.resolved);
    return {
      ...group,
      indices,

      targetIndex: indices[firstOpen >= 0 ? firstOpen : 0] ?? 0,
      active: indices.includes(activeIndex),
    };
  });

  return (
    <aside
      aria-label={t("ui.repair.queueAria")}
      className="overflow-hidden rounded-lg border border-border bg-card lg:sticky lg:top-4"
    >
      <div className="border-b border-border px-4 pb-3 pt-3.5">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-semibold text-foreground">
            {t("ui.repair.queueTitle", { count: progress.remaining })}
          </p>
          <span className="shrink-0 rounded-full border border-border px-2 py-0.5 text-micro text-muted-foreground">
            {t("ui.repair.queueProgress", {
              done: progress.done,
              total: progress.total,
            })}
          </span>
        </div>
        <Progress
          value={progress.percent}
          ariaLabel={t("ui.repair.queueAria")}
          className="mt-2.5 h-1"
        />
      </div>

      <ol className="max-h-[60vh] overflow-y-auto p-1.5">
        {groups.map((group, position) => {
          const item = group.representative;
          const copy = copyOf(item.finding);

          const hasRoute =
            findingRepairTarget(projectId, item.finding.issue) !== null;
          const subtitle = group.resolved
            ? t("ui.repair.queueResolved")
            : (copy.jump ?? (hasRoute ? null : t("ui.repair.queueNoRoute")));
          const addedByRecheck = group.rows.some(
            (row) => row.addedByRecheck && !row.resolved,
          );
          return (
            <li key={group.code}>
              <button
                type="button"
                data-testid="repair-queue-item"
                aria-current={group.active ? "true" : undefined}
                onClick={() => onSelect(group.targetIndex)}
                className={`flex w-full items-start gap-2.5 rounded-lg border px-2.5 py-2 text-left ui-interactive ui-focus-ring ${
                  group.active
                    ? "border-accent/40 bg-accent/10"
                    : "border-transparent hover:bg-muted/40"
                }`}
              >
                <span
                  aria-hidden
                  className={`mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded-full border text-micro ${
                    group.resolved
                      ? "border-success/60 bg-success/15 text-success"
                      : group.active
                        ? "border-accent text-accent"
                        : "border-border text-muted-foreground"
                  }`}
                >
                  {group.resolved ? (
                    <Check className="h-3 w-3" aria-hidden />
                  ) : (
                    position + 1
                  )}
                </span>
                <span className="min-w-0 flex-1">
                  <span
                    className={`block text-xs font-semibold ${
                      group.resolved
                        ? "text-muted-foreground line-through"
                        : "text-foreground"
                    }`}
                  >
                    {copy.what}
                  </span>
                  {subtitle ? (
                    <span className="mt-0.5 block text-micro text-muted-foreground">
                      {subtitle}
                    </span>
                  ) : null}
                  <span className="mt-1 flex flex-wrap items-center gap-1.5">
                    {group.count > 1 ? (
                      <span
                        data-testid="repair-queue-group-count"
                        className="inline-flex rounded-full border border-border px-1.5 text-micro text-muted-foreground"
                      >
                        {t("ui.attention.groupAffected", {
                          count: group.count,
                        })}
                      </span>
                    ) : null}
                    {addedByRecheck ? (
                      <span className="inline-flex rounded-full border border-warning/50 px-1.5 text-micro text-warning">
                        {t("ui.repair.queueNew")}
                      </span>
                    ) : null}
                  </span>
                </span>
              </button>
            </li>
          );
        })}
      </ol>
    </aside>
  );
}
