"use client";
import type { ReactNode } from "react";

import { useEffect, useRef, useState } from "react";
import {
  Info,
  OctagonAlert,
  Search,
  TriangleAlert,
  Wrench,
} from "lucide-react";
import { useTranslations } from "use-intl";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { API_BASE } from "@/lib/api";
import { Button } from "@/components/ui/button";
import type {
  CrossRefRepairLintIssue,
  LintReport,
  LintTab,
} from "./treeValidationMarkers";
import { asCrossRefRepairIssue } from "./treeValidationMarkers";
import { CrossRefRepairDialog } from "./CrossRefRepairDialog";

export type {
  LintIssue,
  LintReport,
  LintSourceSpan,
  LintTab,
} from "./treeValidationMarkers";

interface TreeValidationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: number;

  initialTab?: LintTab;

  onReport?: (report: LintReport | null) => void;
}

const LEVEL_BG: Record<string, string> = {
  error: "bg-destructive/10 border-destructive/40",
  warning: "bg-warning/10 border-warning/40",
  info: "bg-muted border-border",
};

const LEVEL_ICON: Record<string, ReactNode> = {
  error: (
    <OctagonAlert
      className="h-3.5 w-3.5 shrink-0 text-destructive"
      aria-hidden
    />
  ),
  warning: (
    <TriangleAlert className="h-3.5 w-3.5 shrink-0 text-warning" aria-hidden />
  ),
  info: (
    <Info className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden />
  ),
};

const LEVEL_LABEL: Record<string, string> = {
  error: "Error",
  warning: "Warning",
  info: "Info",
};

export function TreeValidationDialog({
  open,
  onOpenChange,
  projectId,
  initialTab = "focus",
  onReport,
}: TreeValidationDialogProps) {
  const t = useTranslations("treeValidationDialog");
  const [report, setReport] = useState<LintReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<LintTab>(initialTab);
  const [validationRun, setValidationRun] = useState(0);
  const [repairIssue, setRepairIssue] =
    useState<CrossRefRepairLintIssue | null>(null);
  const onReportRef = useRef(onReport);

  useEffect(() => {
    onReportRef.current = onReport;
  }, [onReport]);

  useEffect(() => {
    if (!open) return;

    let cancelled = false;
    Promise.resolve().then(() => {
      if (cancelled) return;
      setTab(initialTab);
      setLoading(true);
      setError(null);
      setReport(null);
      onReportRef.current?.(null);
    });
    fetch(`${API_BASE}/go/lint/tree-validation/${projectId}`, {
      credentials: "include",
    })
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json() as Promise<LintReport>;
      })
      .then((j) => {
        if (cancelled) return;
        setReport(j);
        onReportRef.current?.(j);
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, projectId, initialTab, validationRun]);

  const issues = (report ? report[tab] : []) ?? [];
  const total =
    (report?.totals?.error ?? 0) +
    (report?.totals?.warning ?? 0) +
    (report?.totals?.info ?? 0);

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent className="sm:max-w-2xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Search className="h-4 w-4 shrink-0" aria-hidden />
            {t("title")}
          </DialogTitle>
          <DialogDescription>{t("description")}</DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-2 pt-2 border-b border-border">
          {(["focus", "decision", "event", "idea"] as const).map((k) => {
            const count = report?.[k]?.length ?? 0;
            const active = k === tab;
            return (
              <button
                key={k}
                type="button"
                onClick={() => setTab(k)}
                className={
                  "px-3 py-1.5 text-xs font-medium border-b-2 transition-colors " +
                  (active
                    ? "border-primary text-primary"
                    : "border-transparent text-muted-foreground hover:text-foreground")
                }
              >
                {k} ({count})
              </button>
            );
          })}
          <div className="ml-auto text-xs text-muted-foreground pr-2">
            {report &&
              `${report.totals.error ?? 0} err · ${report.totals.warning ?? 0} warn · ${
                report.totals.info ?? 0
              } info · total ${total}`}
          </div>
        </div>

        <div className="flex-1 overflow-y-auto pt-2 space-y-1.5">
          {loading && (
            <p className="text-sm text-muted-foreground">{t("runningLint")}</p>
          )}
          {error && (
            <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          )}
          {!loading && !error && issues.length === 0 && (
            <p className="text-sm text-muted-foreground italic">
              {t("clean", { tab })}
            </p>
          )}
          {issues.map((it, i) => {
            const repairable = asCrossRefRepairIssue(it);
            return (
              <div
                key={`${tab}-${i}`}
                className={`rounded-md border px-3 py-2 ${LEVEL_BG[it.level]}`}
              >
                <div className="flex items-start gap-2">
                  <span className="text-base leading-none mt-0.5">
                    {LEVEL_ICON[it.level]}
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-micro uppercase font-mono font-semibold">
                        {LEVEL_LABEL[it.level]}
                      </span>
                      <span className="text-micro font-mono text-muted-foreground">
                        {it.entity_kind}/{it.entity_id}
                      </span>
                      {it.field && (
                        <span className="text-micro font-mono px-1 py-0.5 rounded bg-muted">
                          {it.field}
                        </span>
                      )}
                    </div>
                    <p className="text-xs mt-1">{it.msg}</p>
                    {it.suggested_fix && (
                      <p className="text-xs text-muted-foreground mt-1">
                        <span className="font-semibold">{t("fixLabel")}</span>{" "}
                        {it.suggested_fix}
                      </p>
                    )}
                    {repairable && (
                      <Button
                        type="button"
                        variant="outline"
                        size="xs"
                        className="mt-2"
                        onClick={() => setRepairIssue(repairable)}
                      >
                        <Wrench aria-hidden />
                        {t("repairButton")}
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </DialogContent>
      {repairIssue && (
        <CrossRefRepairDialog
          key={`${repairIssue.entity_kind}:${repairIssue.entity_api_id}:${repairIssue.field}:${repairIssue.source_span.start_offset}`}
          open
          onOpenChange={(nextOpen) => {
            if (!nextOpen) setRepairIssue(null);
          }}
          projectId={projectId}
          issue={repairIssue}
          onApplied={() => {
            setRepairIssue(null);
            setValidationRun((run) => run + 1);
          }}
        />
      )}
    </Dialog>
  );
}

export function TreeValidationButton({
  projectId,
  kind,
  size = "sm",
  onReport,
}: {
  projectId: number;
  kind: LintTab;
  size?: "sm" | "default";
  onReport?: (report: LintReport | null) => void;
}) {
  const t = useTranslations("treeValidationDialog");
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => setOpen(true)}
        className={size === "sm" ? "h-7 gap-1 px-2" : "h-9 gap-1 px-3 text-sm"}
        title={t("buttonTitle")}
      >
        <Search className="h-3.5 w-3.5 shrink-0" aria-hidden />
        <span>{t("validate")}</span>
      </Button>
      <TreeValidationDialog
        open={open}
        onOpenChange={setOpen}
        projectId={projectId}
        initialTab={kind}
        onReport={onReport}
      />
    </>
  );
}
