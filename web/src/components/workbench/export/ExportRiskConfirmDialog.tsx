"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { CtdIssue } from "@/lib/api/export";
import { ApiError } from "@/lib/api/_client";
import {
  playerIssueCopy,
  type PlayerTranslator,
} from "@/components/workbench/mod-check/validationIssuePresentation";

interface ExportRiskConfirmDialogProps {
  open: boolean;
  blockerCount: number;
  blockers: CtdIssue[];
  onCancel: () => void;
  onConfirm: () => void;
}

export function diagnosticBlockersFromExportError(error: unknown): CtdIssue[] {
  if (!(error instanceof ApiError) || error.status !== 409) return [];
  if (!error.payload || typeof error.payload !== "object") return [];
  const candidates = (error.payload as { blockers?: unknown }).blockers;
  if (!Array.isArray(candidates)) return [];
  return candidates.flatMap((candidate) => {
    if (!candidate || typeof candidate !== "object") return [];
    const item = candidate as Record<string, unknown>;
    if (typeof item.code !== "string" || item.severity !== "blocker") return [];
    const entityRef = item.entity_ref;
    if (
      entityRef !== null &&
      (typeof entityRef !== "object" || Array.isArray(entityRef))
    ) {
      return [];
    }
    return [
      {
        ...item,
        entity_ref: entityRef as CtdIssue["entity_ref"],
        severity: "blocker" as const,

        message: "",
        fix_hint: "",
      } as CtdIssue,
    ];
  });
}

export function ExportRiskConfirmDialog({
  open,
  blockerCount,
  blockers,
  onCancel,
  onConfirm,
}: ExportRiskConfirmDialogProps) {
  const t = useTranslations("export");
  const tCheck = useTranslations("modCheck") as PlayerTranslator;
  const groupedProblems = useMemo(() => {
    const groups = new Map<string, CtdIssue[]>();
    for (const blocker of blockers) {
      const presentation = playerIssueCopy(tCheck, blocker);
      const existing = groups.get(presentation.groupKey) ?? [];
      existing.push(blocker);
      groups.set(presentation.groupKey, existing);
    }
    return [...groups.values()].map(
      (group) => playerIssueCopy(tCheck, group[0], group.length).message,
    );
  }, [blockers, tCheck]);

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onCancel();
      }}
      disablePointerDismissal
    >
      <DialogContent className="sm:max-w-lg" showCloseButton={false}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-base">
            <AlertTriangle
              className="h-4 w-4 shrink-0 text-warning"
              aria-hidden
            />
            {t("riskConfirmTitle")}
          </DialogTitle>
          <DialogDescription>
            {t("riskConfirmBody", { count: blockerCount })}
          </DialogDescription>
        </DialogHeader>

        {groupedProblems.length > 0 && (
          <ul
            className="max-h-48 list-disc space-y-1 overflow-y-auto rounded-lg border border-warning/40 bg-warning/5 px-8 py-3 text-sm text-foreground"
            data-testid="export-risk-problems"
          >
            {groupedProblems.map((problem) => (
              <li key={problem}>{problem}</li>
            ))}
          </ul>
        )}

        <div className="space-y-1 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2">
          <p className="text-sm font-medium text-foreground">
            {t("riskConfirmImpact")}
          </p>
          <p className="text-xs text-muted-foreground">
            {t("riskConfirmReminder")}
          </p>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            {t("riskConfirmCancel")}
          </Button>
          <Button variant="destructive" onClick={onConfirm}>
            {t("riskConfirmProceed")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
