"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { useEditTargets } from "@/lib/hooks/useEditTargets";

interface EditTargetIndicatorProps {
  projectId: number;
  modName: string;
  primaryTag: string;

  hideOnError?: boolean;
}

const MODE_LABEL: Record<string, { text: string; cls: string }> = {
  append: {
    text: "append",
    cls: "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-300",
  },
  shared: {
    text: "shared",
    cls: "bg-blue-100 text-blue-800 dark:bg-blue-950/30 dark:text-blue-300",
  },
  overwrite: {
    text: "overwrite",
    cls: "bg-amber-100 text-amber-900 dark:bg-amber-950/30 dark:text-amber-300",
  },
};

const EMPTY_BY_KIND: Record<string, never> = {};

export function EditTargetIndicator({
  projectId,
  modName,
  primaryTag,
  hideOnError = false,
}: EditTargetIndicatorProps) {
  const t = useTranslations("editTargetIndicator");
  const [open, setOpen] = useState(false);
  const { data, loading, error } = useEditTargets(projectId);

  if (error && hideOnError) return null;

  const totalFiles = data?.total_files ?? 0;

  const byKind = data?.by_kind ?? EMPTY_BY_KIND;
  const kindCount = Object.keys(byKind).length;

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="inline-flex items-center gap-2 rounded-md border border-amber-400/50 bg-amber-50 dark:bg-amber-950/20 px-2.5 py-1 text-xs text-amber-900 dark:text-amber-200 hover:bg-amber-100 dark:hover:bg-amber-950/40 transition-colors"
        title={t("titleAttr")}
      >
        <span className="font-semibold">★ Edit Target</span>
        <span className="font-mono">{modName || "(unnamed)"}</span>
        <span className="text-amber-700/80 dark:text-amber-300/80">·</span>
        {loading ? (
          <span className="text-amber-700/70">computing...</span>
        ) : error ? (
          <span className="text-amber-700/70">unavailable</span>
        ) : (
          <span>
            {kindCount} kinds · {totalFiles} files
          </span>
        )}
        <span className="text-micro text-amber-700/70 dark:text-amber-300/70 ml-1">
          ▼
        </span>
      </button>

      <Dialog open={open} onOpenChange={setOpen} disablePointerDismissal>
        <DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>★ Edit Targets · {modName}</DialogTitle>
            <DialogDescription>
              {t.rich("descBody", {
                primaryTag: primaryTag || "—",
                totalFiles,
                kindCount,
                tag: (chunks) => <span className="font-mono">{chunks}</span>,
              })}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 pt-2">
            {loading && (
              <p className="text-sm text-muted-foreground">
                Computing edit targets…
              </p>
            )}
            {error && (
              <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}
            {data && Object.keys(byKind).length === 0 && (
              <p className="text-sm text-muted-foreground">{t("emptyState")}</p>
            )}
            {data &&
              Object.entries(byKind)
                .sort(([, a], [, b]) => b.count - a.count)
                .map(([kind, entry]) => (
                  <div
                    key={kind}
                    className="rounded-md border border-border bg-card/30 px-3 py-2"
                  >
                    <div className="flex items-center justify-between gap-2 mb-1.5">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-sm">
                          {t.has(`kind_${kind}`)
                            ? t(`kind_${kind}`)
                            : entry.label}
                        </span>
                        <span className="text-micro text-muted-foreground font-mono">
                          {kind}
                        </span>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <span className="text-xs text-muted-foreground">
                          {entry.count} files
                        </span>
                        {Object.entries(entry.modes ?? {}).map(([mode, n]) => {
                          const ml = MODE_LABEL[mode] ?? {
                            text: mode,
                            cls: "bg-muted text-muted-foreground",
                          };
                          return (
                            <span
                              key={mode}
                              className={`text-micro px-1.5 py-0.5 rounded ${ml.cls}`}
                            >
                              {ml.text} ×{n}
                            </span>
                          );
                        })}
                      </div>
                    </div>
                    {entry.paths.length > 0 && (
                      <ul className="text-xs text-muted-foreground font-mono leading-relaxed max-h-32 overflow-y-auto space-y-0.5">
                        {entry.paths.map((p) => (
                          <li key={p}>{p}</li>
                        ))}
                        {entry.count > entry.paths.length && (
                          <li className="italic">
                            … + {entry.count - entry.paths.length} more
                          </li>
                        )}
                      </ul>
                    )}
                  </div>
                ))}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
