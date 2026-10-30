"use client";

import { useMemo, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { FolderOpen, Info, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  exportApi,
  existingModName,
  isNotAModFolderError,
  type CtdIssue,
  type ConflictReport,
  type DiffExportPreview,
  type ExistingMod,
} from "@/lib/api";
import { diagnosticBlockersFromExportError } from "@/components/workbench/export/ExportRiskConfirmDialog";
import { FolderPicker } from "@/desktop/FolderPicker";

interface DiffMergeExportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: number;
  projectName: string;
  hasTechnologySurface: boolean;

  onExported?: (filename: string) => void;
  confirmDiagnosticRisk?: (blockers: CtdIssue[]) => Promise<boolean>;
}

type Phase = "pick_file" | "previewing" | "resolving" | "submitting" | "done";

function triggerBlobDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);

  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

const STATUS_TONE: Record<ConflictReport["status"], string> = {
  no_change: "text-muted-foreground",
  clean_merge: "text-foreground",
  auto_mergeable: "text-foreground",
  conflict: "text-destructive",
};

const ENTITY_LABEL_KEYS = new Set([
  "character",
  "event",
  "decision",
  "focus_node",
  "idea",
]);

export function DiffMergeExportDialog({
  open,
  onOpenChange,
  projectId,
  projectName,
  hasTechnologySurface,
  onExported,
  confirmDiagnosticRisk,
}: DiffMergeExportDialogProps) {
  const t = useTranslations("diffMergeExportDialog");
  const tExport = useTranslations("export");
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [base, setBase] = useState<ExistingMod | null>(null);
  const [folderPickerOpen, setFolderPickerOpen] = useState(false);
  const [phase, setPhase] = useState<Phase>("pick_file");
  const [preview, setPreview] = useState<DiffExportPreview | null>(null);
  const [resolutions, setResolutions] = useState<
    Record<string, "keep_user" | "keep_target" | "manual">
  >({});
  const [manualValues, setManualValues] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);

  const [doneSummary, setDoneSummary] = useState<{
    filename: string;
    entityCount: number;
    blob: Blob;
  } | null>(null);

  const conflictsByEntity = useMemo(() => {
    if (!preview) return [] as { entityType: string; report: ConflictReport }[];
    const out: { entityType: string; report: ConflictReport }[] = [];
    for (const [entityType, reports] of Object.entries(preview.reports)) {
      for (const r of reports) {
        if (r.status === "conflict") {
          out.push({ entityType, report: r });
        }
      }
    }
    return out;
  }, [preview]);

  const reset = () => {
    setBase(null);
    setPhase("pick_file");
    setPreview(null);
    setResolutions({});
    setManualValues({});
    setError(null);
    setDoneSummary(null);
  };

  const handleOpenChange = (next: boolean) => {
    if (!next) reset();
    onOpenChange(next);
  };

  const handlePickFile = () => fileInputRef.current?.click();

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setBase(e.target.files?.[0] ?? null);
    setError(null);
  };

  const runPreview = async () => {
    if (!base) return;
    setPhase("previewing");
    setError(null);
    try {
      const p = await exportApi.diffPreview(projectId, base);
      setPreview(p);
      if (p.totals.conflict === 0) {
        await submitResolveAndDownloadZip(p, {}, {});
      } else {
        setPhase("resolving");
      }
    } catch (previewError) {
      setError(
        isNotAModFolderError(previewError)
          ? t("folderNotAMod")
          : tExport("exportCouldNotComplete"),
      );
      setPhase("pick_file");
    }
  };

  const allConflictsResolved = useMemo(() => {
    if (!preview) return false;
    for (const [entityType, reports] of Object.entries(preview.reports)) {
      for (const r of reports) {
        if (r.status !== "conflict") continue;
        for (const c of r.conflicts) {
          const key = `${entityType}/${r.entity_id}/${c.path}`;
          if (!resolutions[key]) return false;
          if (
            resolutions[key] === "manual" &&
            manualValues[key] === undefined
          ) {
            return false;
          }
        }
      }
    }
    return true;
  }, [preview, resolutions, manualValues]);

  const parseManualValues = (
    manual: Record<string, string>,
  ): Record<string, unknown> => {
    const parsed: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(manual)) {
      try {
        parsed[k] = JSON.parse(v);
      } catch {
        parsed[k] = v;
      }
    }
    return parsed;
  };

  const submitResolveAndDownloadZip = async (
    p: DiffExportPreview,
    res: Record<string, "keep_user" | "keep_target" | "manual">,
    manual: Record<string, string>,
  ) => {
    if (!base) return;
    setPhase("submitting");
    try {
      const acknowledgeBlockers: boolean = true;
      const runDiffExport = (acknowledged: boolean) =>
        exportApi.diffExportZip(
          projectId,
          base,
          res,
          parseManualValues(manual),
          acknowledged,
        );
      let result;
      try {
        result = await runDiffExport(acknowledgeBlockers);
      } catch (diffError) {
        const finalBlockers = diagnosticBlockersFromExportError(diffError);
        if (
          !acknowledgeBlockers &&
          confirmDiagnosticRisk &&
          finalBlockers.length > 0
        ) {
          const confirmed = await confirmDiagnosticRisk(finalBlockers);
          if (!confirmed) {
            setPhase(p.totals.conflict > 0 ? "resolving" : "pick_file");
            return;
          }
          result = await runDiffExport(true);
        } else {
          throw diffError;
        }
      }
      triggerBlobDownload(result.blob, result.filename);
      setDoneSummary({
        filename: result.filename,
        entityCount: result.entityCount,
        blob: result.blob,
      });
      setPhase("done");

      onExported?.(result.filename || `${projectName}_merged.zip`);
      const conflictTotal = p.totals.total_conflict_paths;
      if (conflictTotal > 0) {
        toast.success(
          t("toastZoneConflict", {
            conflictTotal,
            entityCount: result.entityCount,
          }),
        );
      } else {
        toast.success(
          t("toastZoneNoConflict", { entityCount: result.entityCount }),
        );
      }
    } catch {
      setError(tExport("exportCouldNotComplete"));
      setPhase("resolving");
    }
  };

  const submitResolveAdvancedJson = async (
    res: Record<string, "keep_user" | "keep_target" | "manual">,
    manual: Record<string, string>,
  ) => {
    if (!base) return;
    setPhase("submitting");
    try {
      const result = await exportApi.diffResolve(
        projectId,
        base,
        res,
        parseManualValues(manual),
      );
      const jsonBlob = new Blob([JSON.stringify(result, null, 2)], {
        type: "application/json",
      });
      const jsonName =
        existingModName(base).replace(/\.zip$/i, "") + "_resolution.json";
      triggerBlobDownload(jsonBlob, jsonName);
      const entityCount = Object.keys(result.final_ops_by_entity).length;
      const opCount = Object.values(result.final_ops_by_entity).reduce(
        (sum, ops) => sum + ops.length,
        0,
      );
      toast.success(t("toastJson", { entityCount, opCount }));
      setPhase("resolving");
    } catch {
      setError(tExport("exportCouldNotComplete"));
      setPhase("resolving");
    }
  };

  const handleConfirmResolutions = async () => {
    if (!preview) return;
    await submitResolveAndDownloadZip(preview, resolutions, manualValues);
  };

  const handleAdvancedJson = async () => {
    await submitResolveAdvancedJson(resolutions, manualValues);
  };

  const handleReDownload = () => {
    if (doneSummary) {
      triggerBlobDownload(doneSummary.blob, doneSummary.filename);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange} disablePointerDismissal>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
          <DialogDescription>{t("description")}</DialogDescription>
        </DialogHeader>

        {hasTechnologySurface ? (
          <section
            data-testid="technology-diff-merge-guidance"
            className="flex items-start gap-2.5 rounded-md border border-border bg-muted/20 p-3"
          >
            <Info
              className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground"
              aria-hidden
            />
            <div className="min-w-0 space-y-1.5">
              <p className="text-xs font-semibold text-foreground">
                {t("technologyMergeTitle")}
              </p>
              <p className="text-xs text-muted-foreground">
                {t("technologyMergeBody")}
              </p>
            </div>
          </section>
        ) : null}

        {phase === "pick_file" || phase === "previewing" ? (
          <div className="space-y-4 pt-2">
            <p className="text-sm font-medium">{t("baseModZipLabel")}</p>
            <input
              ref={fileInputRef}
              type="file"
              accept=".zip,application/zip,application/x-zip-compressed"
              className="hidden"
              onChange={handleFileChange}
            />
            <Button
              variant="outline"
              onClick={handlePickFile}
              disabled={phase === "previewing"}
              className="ui-interactive w-full justify-start gap-2"
            >
              <Upload className="h-4 w-4" />
              {base instanceof File ? base.name : t("pickBaseZip")}
            </Button>
            <Button
              variant="outline"
              onClick={() => setFolderPickerOpen(true)}
              disabled={phase === "previewing"}
              className="ui-interactive w-full justify-start gap-2"
            >
              <FolderOpen className="h-4 w-4" />
              {base && !(base instanceof File)
                ? existingModName(base)
                : t("pickBaseFolder")}
            </Button>
            {base && !(base instanceof File) ? (
              <p className="break-all font-mono text-xs text-muted-foreground">
                {base.folder}
              </p>
            ) : null}
            {error ? (
              <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            ) : null}
            <div className="flex justify-end gap-2 pt-2">
              <Button
                variant="outline"
                onClick={() => handleOpenChange(false)}
                disabled={phase === "previewing"}
              >
                {t("cancel")}
              </Button>
              <Button
                onClick={runPreview}
                disabled={!base || phase === "previewing"}
                className="bg-accent text-accent-foreground hover:bg-accent/90 ui-interactive"
              >
                {phase === "previewing" ? t("analyzing") : t("analyzeDiff")}
              </Button>
            </div>
          </div>
        ) : null}

        {phase === "resolving" && preview ? (
          <div className="space-y-4 pt-2">
            <div className="ui-panel p-4">
              <p className="text-sm font-semibold tracking-tight">
                {t("conflictSummary", {
                  conflict: preview.totals.conflict,
                  paths: preview.totals.total_conflict_paths,
                })}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("mergeSummary", {
                  auto: preview.totals.auto_mergeable,
                  clean: preview.totals.clean_merge,
                  noChange: preview.totals.no_change,
                })}
              </p>
            </div>

            <div className="max-h-[420px] overflow-y-auto space-y-3 pr-1">
              {conflictsByEntity.map(({ entityType, report }) => (
                <div
                  key={`${entityType}/${report.entity_id}`}
                  className="ui-panel p-4"
                >
                  <div className="flex items-baseline justify-between">
                    <p className="text-sm font-semibold">
                      {ENTITY_LABEL_KEYS.has(entityType)
                        ? t(`entityLabel.${entityType}`)
                        : entityType}{" "}
                      ·{" "}
                      <span className="font-mono text-xs text-muted-foreground">
                        {report.entity_id}
                      </span>
                    </p>
                    <span className={`text-xs ${STATUS_TONE[report.status]}`}>
                      {t(`status.${report.status}`)} ·{" "}
                      {t("conflictPaths", { count: report.conflicts.length })}
                    </span>
                  </div>
                  <ul className="mt-3 space-y-3">
                    {report.conflicts.map((c) => {
                      const key = `${entityType}/${report.entity_id}/${c.path}`;
                      const choice = resolutions[key];
                      return (
                        <li key={key} className="ui-card p-3">
                          <p className="font-mono text-xs text-muted-foreground">
                            {c.path}
                            {c.kind === "remove_vs_change"
                              ? ` ${t("removeVsChange")}`
                              : null}
                          </p>
                          <div className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2 text-xs">
                            <div className="rounded-md border border-border bg-muted/40 p-2">
                              <p className="text-muted-foreground">
                                {t("yourChange")}
                              </p>
                              <p className="mt-1 font-mono break-all">
                                {formatValue(
                                  c.user_op.value ?? c.user_op.old,
                                  t("emptyValue"),
                                  t("emptyString"),
                                )}
                              </p>
                            </div>
                            <div className="rounded-md border border-border bg-muted/40 p-2">
                              <p className="text-muted-foreground">
                                {t("baseModCurrent")}
                              </p>
                              <p className="mt-1 font-mono break-all">
                                {formatValue(
                                  c.target_op.value ?? c.target_op.old,
                                  t("emptyValue"),
                                  t("emptyString"),
                                )}
                              </p>
                            </div>
                          </div>
                          <div className="mt-3 flex flex-wrap gap-2">
                            <ChoiceButton
                              active={choice === "keep_user"}
                              onClick={() =>
                                setResolutions((p) => ({
                                  ...p,
                                  [key]: "keep_user",
                                }))
                              }
                            >
                              {t("useMine")}
                            </ChoiceButton>
                            <ChoiceButton
                              active={choice === "keep_target"}
                              onClick={() =>
                                setResolutions((p) => ({
                                  ...p,
                                  [key]: "keep_target",
                                }))
                              }
                            >
                              {t("useBaseMod")}
                            </ChoiceButton>
                            <ChoiceButton
                              active={choice === "manual"}
                              onClick={() =>
                                setResolutions((p) => ({
                                  ...p,
                                  [key]: "manual",
                                }))
                              }
                            >
                              {t("manualWrite")}
                            </ChoiceButton>
                          </div>
                          {choice === "manual" ? (
                            <Input
                              className="mt-3 font-mono text-xs"
                              placeholder={t("manualPlaceholder")}
                              value={manualValues[key] ?? ""}
                              onChange={(e) =>
                                setManualValues((p) => ({
                                  ...p,
                                  [key]: e.target.value,
                                }))
                              }
                            />
                          ) : null}
                        </li>
                      );
                    })}
                  </ul>
                </div>
              ))}
            </div>

            {error ? (
              <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            ) : null}

            <div className="flex justify-between gap-2 pt-2">
              <Button
                variant="outline"
                onClick={handleAdvancedJson}
                disabled={!allConflictsResolved}
                className="ui-interactive"
                title={t("advancedJsonTitle")}
              >
                {t("advancedJsonBtn")}
              </Button>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={() => handleOpenChange(false)}
                >
                  {t("cancel")}
                </Button>
                <Button
                  onClick={handleConfirmResolutions}
                  disabled={!allConflictsResolved}
                  className="bg-accent text-accent-foreground hover:bg-accent/90 ui-interactive"
                >
                  {t("downloadMergedZip")}
                </Button>
              </div>
            </div>
          </div>
        ) : null}

        {phase === "submitting" ? (
          <div className="py-8 text-center text-sm text-muted-foreground">
            {t("merging")}
          </div>
        ) : null}

        {phase === "done" && doneSummary ? (
          <div className="space-y-3 pt-2">
            <div className="ui-panel p-4">
              <p className="text-sm font-semibold tracking-tight">
                {t("mergeDone")}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("filenameLabel")}{" "}
                <span className="font-mono">{doneSummary.filename}</span>
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("mergedSummary", {
                  projectName,
                  entityCount: doneSummary.entityCount,
                })}
              </p>
            </div>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={handleReDownload}
                className="ui-interactive"
              >
                {t("reDownload")}
              </Button>
              <Button
                onClick={() => handleOpenChange(false)}
                className="bg-accent text-accent-foreground hover:bg-accent/90 ui-interactive"
              >
                {t("close")}
              </Button>
            </div>
          </div>
        ) : null}
        <FolderPicker
          open={folderPickerOpen}
          onOpenChange={setFolderPickerOpen}
          title={t("folderPickerTitle")}
          description={t("folderPickerHint")}
          onPick={(folder) => {
            setFolderPickerOpen(false);
            setBase({ folder });
            setError(null);
          }}
        />
      </DialogContent>
    </Dialog>
  );
}

function ChoiceButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        active
          ? "ui-interactive h-7 rounded-md border border-accent bg-accent px-3 text-xs text-accent-foreground"
          : "ui-interactive h-7 rounded-md border border-border bg-transparent px-3 text-xs text-foreground hover:bg-muted"
      }
    >
      {children}
    </button>
  );
}

function formatValue(
  v: unknown,
  emptyLabel: string,
  emptyStringLabel: string,
): string {
  if (v === null || v === undefined) return emptyLabel;
  if (typeof v === "string") return v.length > 0 ? v : emptyStringLabel;
  try {
    return JSON.stringify(v);
  } catch {
    return String(v);
  }
}
