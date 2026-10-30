"use client";

import { useCallback, useEffect, useState, useRef } from "react";
import { useRouter } from "@/i18n/navigation";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { InfoTooltip, TooltipProvider } from "@/components/ui/tooltip";
import { FolderOpen } from "lucide-react";
import { useProjectStore } from "@/lib/store/projectStore";
import { useSetPendingModFolder } from "@/lib/store/pendingModFolderStore";
import { PREVIEW_SECTIONS } from "@/components/workbench/import-wizard/scan";
import { performScanInBrowser } from "@/components/workbench/import-wizard/scan-client";
import { readFileByPath } from "@/lib/fs-scanner";
import {
  ModFolderPicker,
  type ModFolderPickerHandle,
} from "@/components/workbench/ModFolderPicker";
import { BrowserCompatHint } from "@/components/workbench/BrowserCompatHint";
import { API_BASE } from "@/lib/api";
import type { ProjectCreateInput } from "@/lib/api/project";
import { DlcPicker } from "@/components/workbench/DlcPicker";

interface LoadExistingModDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function waitForBrowserPaint(): Promise<void> {
  return new Promise((resolve) => {
    if (
      typeof window === "undefined" ||
      typeof window.requestAnimationFrame !== "function"
    ) {
      setTimeout(resolve, 0);
      return;
    }
    window.requestAnimationFrame(() => window.setTimeout(resolve, 0));
  });
}

interface ModPreview {
  countries: number;
  entities: number;
}

export function LoadExistingModDialog({
  open,
  onOpenChange,
}: LoadExistingModDialogProps) {
  const router = useRouter();
  const t = useTranslations("loadExistingMod");
  const tScan = useTranslations("unifiedModImport");
  const createProject = useProjectStore((s) => s.createProject);

  const setPendingModFolder = useSetPendingModFolder();

  const [detectedName, setDetectedName] = useState("");

  const [ownedDlcs, setOwnedDlcs] = useState<string[] | null>(null);
  const [pickedFolderName, setPickedFolderName] = useState<string | null>(null);
  const [parsedVersion, setParsedVersion] = useState<string | null>(null);

  const [parsedSupportedVersion, setParsedSupportedVersion] = useState<
    string | null
  >(null);

  const [parsedReplacePaths, setParsedReplacePaths] = useState<string[]>([]);
  const [preview, setPreview] = useState<ModPreview | null>(null);
  const [scanning, setScanning] = useState(false);

  const [scanStatus, setScanStatus] = useState("");

  const [picking, setPicking] = useState(false);
  const [creating, setCreating] = useState(false);
  const [note, setNote] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const dirHandleRef = useRef<FileSystemDirectoryHandle | null>(null);
  const folderWorkGenerationRef = useRef(0);
  const folderWorkAbortRef = useRef<AbortController | null>(null);
  const cancelFolderWork = useCallback(() => {
    folderWorkGenerationRef.current += 1;
    folderWorkAbortRef.current?.abort();
    folderWorkAbortRef.current = null;
  }, []);

  const pickerRef = useRef<ModFolderPickerHandle | null>(null);

  const canSubmit =
    dirHandleRef.current !== null &&
    detectedName.trim().length > 0 &&
    !scanning &&
    !creating;

  const resetState = useCallback(() => {
    cancelFolderWork();
    setDetectedName("");
    setOwnedDlcs(null);
    setPickedFolderName(null);
    setParsedVersion(null);
    setParsedSupportedVersion(null);
    setParsedReplacePaths([]);
    setPreview(null);
    setScanning(false);
    setNote(null);
    setError(null);
    dirHandleRef.current = null;
  }, [cancelFolderWork]);

  useEffect(() => cancelFolderWork, [cancelFolderWork]);

  const handlePickModFolder = () => {
    setError(null);
    setNote(null);
    setPicking(true);
    pickerRef.current?.pick();
  };

  const onFolderPicked = async (dirHandle: FileSystemDirectoryHandle) => {
    folderWorkAbortRef.current?.abort();
    const generation = folderWorkGenerationRef.current + 1;
    folderWorkGenerationRef.current = generation;
    const controller = new AbortController();
    folderWorkAbortRef.current = controller;
    const isCurrent = () =>
      folderWorkGenerationRef.current === generation &&
      !controller.signal.aborted;

    setPicking(false);

    dirHandleRef.current = dirHandle;
    setPickedFolderName(dirHandle.name);
    setScanning(true);
    setScanStatus("");
    setPreview(null);
    setParsedVersion(null);
    setParsedSupportedVersion(null);
    setParsedReplacePaths([]);

    try {
      await waitForBrowserPaint();
      if (!isCurrent()) return;

      try {
        const descriptorFile = await readFileByPath(
          dirHandle,
          "descriptor.mod",
        );
        if (!isCurrent()) return;
        const descriptorText = descriptorFile
          ? await descriptorFile.text()
          : null;
        if (!isCurrent()) return;
        if (!descriptorText) {
          setDetectedName(dirHandle.name);
          setNote(t("descriptorNotFound", { name: dirHandle.name }));
        } else {
          const res = await fetch(
            `${API_BASE}/go/projects/auto-detect-descriptor`,
            {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ descriptor_text: descriptorText }),
              signal: controller.signal,
            },
          );
          if (!isCurrent()) return;
          if (!res.ok) {
            setDetectedName(dirHandle.name);
            setNote(t("descriptorParseFailed", { status: res.status }));
          } else {
            const parsed: {
              name: string | null;
              tags: string[];
              version: string | null;
              supported_version: string | null;
              replace_paths?: string[];
            } = await res.json();
            if (!isCurrent()) return;
            setParsedVersion(parsed.version);
            setParsedSupportedVersion(parsed.supported_version);
            setParsedReplacePaths(
              Array.isArray(parsed.replace_paths) ? parsed.replace_paths : [],
            );
            if (parsed.name) {
              setDetectedName(parsed.name);
            } else {
              setDetectedName(dirHandle.name);
              setNote(t("descriptorNoName", { name: dirHandle.name }));
            }
          }
        }
      } catch (e) {
        if (!isCurrent()) return;

        setDetectedName(dirHandle.name);
        setNote(
          t("descriptorReadError", {
            msg: e instanceof Error ? e.message : String(e),
          }),
        );
      }

      if (!isCurrent()) return;

      try {
        const results = await performScanInBrowser(
          dirHandle,
          "",
          (status) => {
            if (isCurrent()) setScanStatus(status);
          },
          tScan,
          { sections: PREVIEW_SECTIONS, signal: controller.signal },
        );
        if (!isCurrent()) return;
        const entities =
          results.characters.fileCount +
          results.ideas.fileCount +
          results.focus_trees.fileCount +
          results.events.fileCount +
          results.decisions.fileCount +
          results.events.files.filter((f) => {
            const base = f.path.split("/").pop()?.toLowerCase() ?? "";
            return (
              base.endsWith("_se.txt") || base.endsWith("_super_event.txt")
            );
          }).length;
        setPreview({
          countries:
            results.country_tags.declaredTagCount ??
            results.country_tags.fileCount,
          entities,
        });
      } catch {
        if (isCurrent()) setPreview(null);
      }
    } finally {
      if (isCurrent()) setScanning(false);
      if (folderWorkAbortRef.current === controller) {
        folderWorkAbortRef.current = null;
      }
    }
  };

  const handleConfirm = async () => {
    if (!canSubmit || !dirHandleRef.current) return;
    setCreating(true);
    setError(null);
    try {
      const createInput: ProjectCreateInput = {
        name: detectedName.trim(),
        is_loaded_existing_mod: true,
        owned_dlcs: ownedDlcs,
        imported_descriptor_replace_paths: parsedReplacePaths,

        ...(parsedSupportedVersion
          ? { supported_version: parsedSupportedVersion }
          : {}),
        ...(parsedVersion ? { version: parsedVersion } : {}),
      };

      const project = await createProject(createInput);

      setPendingModFolder({
        projectId: project.id,
        dirHandle: dirHandleRef.current,
      });
      onOpenChange(false);
      resetState();

      router.push(`/project/${project.id}/characters`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setCreating(false);
    }
  };

  const handleOpenChange = (next: boolean) => {
    if (!next) resetState();
    onOpenChange(next);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange} disablePointerDismissal>
      <DialogContent>
        <TooltipProvider>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              {t("title")}
              <InfoTooltip
                title={t("structureTooltipTitle")}
                content={t.raw("structureTooltipContent")}
              />
            </DialogTitle>
            <DialogDescription>{t("description")}</DialogDescription>
          </DialogHeader>

          <div className="space-y-4 pt-2">
            <ModFolderPicker
              ref={pickerRef}
              mode="read"
              onPicked={onFolderPicked}
              onSettled={() => setPicking(false)}
              onError={(m) => setError(m)}
            />
            <BrowserCompatHint mode="folderUpload" />

            <Button
              className="w-full"
              size="lg"
              variant={pickedFolderName ? "outline" : "default"}
              onClick={handlePickModFolder}
              disabled={scanning || creating}
            >
              {scanning ? (
                <span
                  className="animate-pulse-ring mr-2 inline-block h-3 w-3 rounded-full border border-accent"
                  aria-hidden
                />
              ) : (
                <FolderOpen className="w-4 h-4 mr-2" aria-hidden />
              )}
              {scanning
                ? t("scanning")
                : pickedFolderName
                  ? t("reselect")
                  : t("selectFolder")}
            </Button>

            {picking && !scanning && (
              <div
                className="rounded-md border border-border bg-muted/40 px-3 py-2 text-xs leading-relaxed text-muted-foreground"
                role="status"
                aria-live="polite"
              >
                {t("pickingHint")}
              </div>
            )}

            {scanning && (
              <div
                className="rounded-md border border-accent/50 bg-accent/10 px-3 py-3 text-sm"
                role="status"
                aria-live="polite"
              >
                <p className="font-semibold">{t("folderAcceptedHeadline")}</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {scanStatus || t("folderAcceptedDetail")}
                </p>
              </div>
            )}

            {pickedFolderName && (
              <div className="space-y-1 rounded-md border border-border bg-muted/30 p-3">
                <p className="text-xs text-muted-foreground">
                  {t("picked")}{" "}
                  <span className="font-mono text-foreground">
                    {pickedFolderName}
                  </span>
                  {parsedVersion && (
                    <>
                      {" · "}version={" "}
                      <span className="font-mono">{parsedVersion}</span>
                    </>
                  )}
                </p>
                {preview && (
                  <p className="text-sm font-medium">
                    {t("foundPreview", {
                      name: detectedName.trim() || pickedFolderName,
                      countries: preview.countries,
                      entities: preview.entities,
                    })}
                  </p>
                )}
                {note && (
                  <p className="text-xs text-amber-700 dark:text-amber-300">
                    {note}
                  </p>
                )}
              </div>
            )}

            {pickedFolderName && (
              <div className="space-y-2">
                <Label
                  htmlFor="load-mod-name"
                  className="flex items-center gap-1"
                >
                  {t("nameLabel")}
                  <InfoTooltip
                    title={t("nameLabel")}
                    content={t.raw("nameHint")}
                  />
                </Label>
                <Input
                  id="load-mod-name"
                  placeholder="e.g. The Forgotten Republic"
                  value={detectedName}
                  onChange={(e) => setDetectedName(e.target.value)}
                />
              </div>
            )}

            {pickedFolderName && (
              <DlcPicker
                value={ownedDlcs}
                onChange={setOwnedDlcs}
                disabled={creating}
              />
            )}

            {error && (
              <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}

            {pickedFolderName && (
              <div className="rounded-md border border-amber-400/40 bg-amber-50 dark:bg-amber-950/20 px-3 py-2 text-xs text-amber-900 dark:text-amber-200">
                {t.rich("enterHint", {
                  b: (chunks) => (
                    <span className="font-semibold">{chunks}</span>
                  ),
                })}
              </div>
            )}

            <div className="flex gap-2 justify-end pt-2">
              <Button
                variant="outline"
                onClick={() => handleOpenChange(false)}
                disabled={creating}
              >
                {t("cancel")}
              </Button>
              <Button onClick={handleConfirm} disabled={!canSubmit}>
                {creating ? t("importing") : t("confirmImport")}
              </Button>
            </div>
          </div>
        </TooltipProvider>
      </DialogContent>
    </Dialog>
  );
}
