"use client";

import { useState } from "react";
import { ArrowRight, LoaderCircle, TriangleAlert, Wrench } from "lucide-react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError } from "@/lib/api/_client";
import {
  crossRefRepairsApi,
  type CrossRefRepairApplyResult,
  type CrossRefRepairPreview,
  type CrossRefRepairRequest,
} from "@/lib/api/crossRefRepair";
import type { CrossRefRepairLintIssue } from "./treeValidationMarkers";

type RepairMode = "replace" | "delete";

interface CrossRefRepairDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: number;
  issue: CrossRefRepairLintIssue;
  onApplied: (result: CrossRefRepairApplyResult) => void;
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.detail;
  return error instanceof Error ? error.message : String(error);
}

export function CrossRefRepairDialog({
  open,
  onOpenChange,
  projectId,
  issue,
  onApplied,
}: CrossRefRepairDialogProps) {
  const t = useTranslations("treeValidationDialog");
  const [mode, setMode] = useState<RepairMode>("replace");
  const [replacementId, setReplacementId] = useState(
    issue.replacement_suggestion ?? "",
  );
  const [preview, setPreview] = useState<CrossRefRepairPreview | null>(null);
  const [previewedRequest, setPreviewedRequest] =
    useState<CrossRefRepairRequest | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState<"preview" | "apply" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const booleanDeleteBlocked = (issue.boolean_ancestry?.length ?? 0) > 0;

  const clearPreview = () => {
    setPreview(null);
    setPreviewedRequest(null);
    setConfirming(false);
    setError(null);
  };

  const chooseMode = (next: RepairMode) => {
    setMode(next);
    clearPreview();
  };

  const buildRequest = (): CrossRefRepairRequest | null => {
    const replacement = replacementId.trim();
    if (mode === "replace" && !replacement) {
      setError(t("repairReplacementRequired"));
      return null;
    }
    return {
      entity_kind: issue.entity_kind,
      entity_id: issue.entity_api_id,
      field: issue.field,
      target_id: issue.target_id,
      target_start_offset: issue.source_span.start_offset,
      target_end_offset: issue.source_span.end_offset,
      replacement_id: mode === "replace" ? replacement : null,
      delete_assignment: mode === "delete",
    };
  };

  const handlePreview = async () => {
    const requestBody = buildRequest();
    if (!requestBody) return;
    setBusy("preview");
    setError(null);
    setConfirming(false);
    try {
      const next = await crossRefRepairsApi.preview(projectId, requestBody);
      setPreview(next);
      setPreviewedRequest(requestBody);
    } catch (nextError) {
      setError(t("repairPreviewFailed", { message: errorMessage(nextError) }));
    } finally {
      setBusy(null);
    }
  };

  const handleApply = async () => {
    if (!preview?.applicable || !previewedRequest) return;
    setBusy("apply");
    setError(null);
    try {
      const result = await crossRefRepairsApi.apply(
        projectId,
        {
          ...previewedRequest,
          source_sha256: preview.source_sha256,
        },
        preview.edit_revisions ?? {},
      );
      setBusy(null);
      onApplied(result);
      onOpenChange(false);
    } catch (nextError) {
      setError(t("repairApplyFailed", { message: errorMessage(nextError) }));
      setBusy(null);
    }
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (busy) return;
    onOpenChange(nextOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange} disablePointerDismissal>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Wrench className="h-4 w-4" aria-hidden />
            {t("repairTitle")}
          </DialogTitle>
          <DialogDescription>{t("repairDescription")}</DialogDescription>
        </DialogHeader>

        <div className="rounded-lg border border-border bg-muted/40 p-3 text-xs">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span className="font-mono text-foreground">{issue.target_id}</span>
            <span className="font-mono text-muted-foreground">
              {issue.entity_kind}/{issue.entity_id} · {issue.field}
            </span>
          </div>
        </div>

        <div
          className="flex flex-wrap gap-2"
          role="group"
          aria-label={t("repairMethodLabel")}
        >
          <Button
            type="button"
            size="sm"
            variant={mode === "replace" ? "secondary" : "outline"}
            aria-pressed={mode === "replace"}
            onClick={() => chooseMode("replace")}
            disabled={busy !== null}
          >
            {t("repairReplaceMode")}
          </Button>
          <Button
            type="button"
            size="sm"
            variant={mode === "delete" ? "secondary" : "outline"}
            aria-pressed={mode === "delete"}
            onClick={() => chooseMode("delete")}
            disabled={busy !== null || booleanDeleteBlocked}
          >
            {t("repairDeleteMode")}
          </Button>
        </div>

        {mode === "replace" && (
          <div className="space-y-1.5">
            <Label htmlFor="cross-ref-replacement">
              {t("repairReplacementLabel")}
            </Label>
            <Input
              id="cross-ref-replacement"
              value={replacementId}
              onChange={(event) => {
                setReplacementId(event.target.value);
                clearPreview();
              }}
              placeholder={t("repairReplacementPlaceholder")}
              spellCheck={false}
              autoComplete="off"
              className="font-mono"
              disabled={busy !== null}
            />
          </div>
        )}

        {booleanDeleteBlocked && (
          <div className="flex items-start gap-2 rounded-lg border border-warning/40 bg-warning/10 p-3 text-xs text-warning">
            <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
            <p>
              {t("repairBooleanDeleteBlocked", {
                ancestry: issue.boolean_ancestry!.join(" > "),
              })}
            </p>
          </div>
        )}

        {mode === "delete" && !booleanDeleteBlocked && (
          <div className="flex items-start gap-2 rounded-lg border border-warning/40 bg-warning/10 p-3 text-xs text-warning">
            <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
            <p>{t("repairDeleteWarning")}</p>
          </div>
        )}

        {preview && (
          <div className="space-y-3 rounded-lg border border-border p-3">
            <div>
              <p className="text-xs font-semibold text-foreground">
                {t("repairPreviewTitle")}
              </p>
              <p className="mt-1 text-micro font-mono text-muted-foreground">
                {t("repairSpanValue", {
                  startLine: preview.source_span.start_line,
                  startColumn: preview.source_span.start_column,
                  endLine: preview.source_span.end_line,
                  endColumn: preview.source_span.end_column,
                  startOffset: preview.source_span.start_offset,
                  endOffset: preview.source_span.end_offset,
                })}
              </p>
            </div>
            <div className="grid gap-2 sm:grid-cols-[1fr_auto_1fr] sm:items-center">
              <div className="min-w-0">
                <p className="mb-1 text-micro font-semibold text-muted-foreground">
                  {t("repairBeforeLabel")}
                </p>
                <pre className="max-h-32 overflow-auto whitespace-pre-wrap break-all rounded-md bg-muted p-2 text-xs text-foreground">
                  {preview.before || t("repairEmptySource")}
                </pre>
              </div>
              <ArrowRight
                className="hidden h-4 w-4 text-muted-foreground sm:block"
                aria-hidden
              />
              <div className="min-w-0">
                <p className="mb-1 text-micro font-semibold text-muted-foreground">
                  {t("repairAfterLabel")}
                </p>
                <pre className="max-h-32 overflow-auto whitespace-pre-wrap break-all rounded-md bg-muted p-2 text-xs text-foreground">
                  {preview.after || t("repairEmptySource")}
                </pre>
              </div>
            </div>
            {!preview.applicable && (
              <div className="rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning">
                {t("repairNotApplicable", { reason: preview.reason })}
              </div>
            )}
          </div>
        )}

        {confirming && preview?.applicable && (
          <div className="rounded-lg border border-destructive/40 bg-destructive/5 p-3">
            <p className="text-xs font-semibold text-destructive">
              {t("repairConfirmTitle")}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("repairConfirmDescription")}
            </p>
          </div>
        )}

        {error && (
          <div
            role="alert"
            className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive"
          >
            {error}
          </div>
        )}

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={busy !== null}
          >
            {t("repairCancel")}
          </Button>
          {!preview && (
            <Button
              type="button"
              onClick={handlePreview}
              disabled={busy !== null}
            >
              {busy === "preview" && (
                <LoaderCircle className="animate-spin" aria-hidden />
              )}
              {busy === "preview"
                ? t("repairPreviewing")
                : t("repairPreviewButton")}
            </Button>
          )}
          {preview && !confirming && (
            <>
              <Button
                type="button"
                variant="outline"
                onClick={clearPreview}
                disabled={busy !== null}
              >
                {t("repairBackButton")}
              </Button>
              <Button
                type="button"
                onClick={() => setConfirming(true)}
                disabled={!preview.applicable || busy !== null}
              >
                {t("repairContinueButton")}
              </Button>
            </>
          )}
          {preview && confirming && (
            <>
              <Button
                type="button"
                variant="outline"
                onClick={() => setConfirming(false)}
                disabled={busy !== null}
              >
                {t("repairBackButton")}
              </Button>
              <Button
                type="button"
                variant={
                  preview.action === "delete_assignment"
                    ? "destructive"
                    : "default"
                }
                onClick={handleApply}
                disabled={busy !== null}
              >
                {busy === "apply" && (
                  <LoaderCircle className="animate-spin" aria-hidden />
                )}
                {busy === "apply"
                  ? t("repairApplying")
                  : t("repairApplyButton")}
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
