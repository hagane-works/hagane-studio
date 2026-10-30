"use client";

import { useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { FolderOpen, Info, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  exportApi,
  existingModName,
  isNotAModFolderError,
  type CtdIssue,
  type ExistingMod,
} from "@/lib/api";
import { diagnosticBlockersFromExportError } from "@/components/workbench/export/ExportRiskConfirmDialog";
import { FolderPicker } from "@/desktop/FolderPicker";

interface AppendToExistingModDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: number;
  projectName: string;
  replacePaths: string[];
  includeImported: boolean;
  buildIdentity: boolean;
  hasTechnologySurface: boolean;

  onExported?: (filename: string) => void;
  confirmDiagnosticRisk?: (blockers: CtdIssue[]) => Promise<boolean>;
}

export function AppendToExistingModDialog({
  open,
  onOpenChange,
  projectId,
  projectName,
  replacePaths,
  includeImported,
  buildIdentity,
  hasTechnologySurface,
  onExported,
  confirmDiagnosticRisk,
}: AppendToExistingModDialogProps) {
  const t = useTranslations("appendToExistingModDialog");
  const tExport = useTranslations("export");
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [base, setBase] = useState<ExistingMod | null>(null);
  const [folderPickerOpen, setFolderPickerOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handlePickFile = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0] ?? null;
    setBase(file);
    setError(null);
  };

  const handleConfirm = async () => {
    if (!base) return;
    setSubmitting(true);
    setError(null);
    try {
      const acknowledgeBlockers: boolean = true;
      const runAppend = (acknowledged: boolean) =>
        exportApi.appendToExisting(projectId, base, {
          include_imported: includeImported,
          build_identity: buildIdentity,
          replace_paths: replacePaths,
          base_mod_filename: existingModName(base),
          acknowledge_blockers: acknowledged,
        });
      let result;
      try {
        result = await runAppend(acknowledgeBlockers);
      } catch (appendError) {
        const finalBlockers = diagnosticBlockersFromExportError(appendError);
        if (
          !acknowledgeBlockers &&
          confirmDiagnosticRisk &&
          finalBlockers.length > 0
        ) {
          const confirmed = await confirmDiagnosticRisk(finalBlockers);
          if (!confirmed) return;
          result = await runAppend(true);
        } else {
          throw appendError;
        }
      }
      const { blob, conflictCount, filename } = result;
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename || `${projectName}_appended.zip`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      if (conflictCount > 0) {
        toast.warning(t("toastWarning", { conflictCount }));
      } else {
        toast.success(t("toastSuccess"));
      }

      onExported?.(filename || `${projectName}_appended.zip`);
      onOpenChange(false);
      setBase(null);
    } catch (appendError) {
      setError(
        isNotAModFolderError(appendError)
          ? t("folderNotAMod")
          : tExport("exportCouldNotComplete"),
      );
    } finally {
      setSubmitting(false);
    }
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
          <DialogDescription>{t("description")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {hasTechnologySurface ? (
            <section
              data-testid="technology-append-guidance"
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

          <div className="space-y-2 rounded-md border border-amber-500/40 bg-amber-500/5 p-3">
            <p className="text-xs font-semibold">{t("naiveMergeTitle")}</p>
            <p className="text-xs text-muted-foreground">
              {t("naiveMergeBody")}
            </p>
          </div>

          <div className="space-y-2">
            <p className="text-sm font-medium">{t("baseZipLabel")}</p>
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
              disabled={submitting}
              className="w-full justify-start gap-2"
            >
              <Upload className="h-4 w-4" />
              {base instanceof File ? base.name : t("pickBaseZip")}
            </Button>
            <Button
              variant="outline"
              onClick={() => setFolderPickerOpen(true)}
              disabled={submitting}
              className="w-full justify-start gap-2"
            >
              <FolderOpen className="h-4 w-4" />
              {base && !(base instanceof File)
                ? existingModName(base)
                : t("pickBaseFolder")}
            </Button>
            {base && (
              <p className="text-xs text-muted-foreground">
                {base instanceof File ? (
                  <>
                    <span className="font-mono">{base.name}</span> ·{" "}
                    {formatSize(base.size)}
                  </>
                ) : (
                  <span className="break-all font-mono">{base.folder}</span>
                )}
              </p>
            )}
            <p className="text-xs text-muted-foreground">{t("supportNote")}</p>
          </div>

          {error && (
            <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          )}

          <div className="flex gap-2 justify-end pt-2">
            <Button
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              {t("cancel")}
            </Button>
            <Button onClick={handleConfirm} disabled={!base || submitting}>
              {submitting ? t("merging") : t("confirmAppend")}
            </Button>
          </div>
        </div>
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
