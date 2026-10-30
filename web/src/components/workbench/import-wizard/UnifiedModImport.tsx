"use client";
import { readProjectEditRevisions } from "@/components/workbench/save-diagnostics/revisionRequest";

import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslations, useLocale } from "use-intl";
import { useRouter } from "@/i18n/navigation";
import { isAppLocale, type AppLocale } from "@/i18n/routing";
import { toast } from "sonner";
import {
  Upload,
  XIcon,
  Users,
  Target,
  FlaskConical,
  Lightbulb,
  CalendarClock,
  Gavel,
  Bookmark,
  ScrollText,
  Map,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  twoStageImportApi,
  projectsApi,
  API_BASE,
  ImportHttpError,
} from "@/lib/api";
import { createUUID } from "@/lib/utils/uuid";
import type {
  ConflictPolicy,
  ScanAndParseResult,
  ScanStatsResult,
} from "@/lib/api";
import { InfoTooltip, TooltipProvider } from "@/components/ui/tooltip";

import {
  ModFolderPicker,
  type ModFolderPickerHandle,
} from "@/components/workbench/ModFolderPicker";
import { BrowserCompatHint } from "@/components/workbench/BrowserCompatHint";

import {
  imageExtensionAlternates,
  readFileByPath,
  tryGetDir,
} from "@/lib/fs-scanner";
import { useProjectCountries, useProjectStore } from "@/lib/store/projectStore";
import {
  useBackgroundImportSession,
  useCancelBackgroundImport,
  useClearBackgroundImport,
  useSetBackgroundImport,
  useUpdateBackgroundImport,
  type BackgroundImportResult,
} from "@/lib/store/backgroundImportStore";
import {
  usePendingModFolderProjectId,
  usePendingModFolderHandle,
  useClearPendingModFolder,
} from "@/lib/store/pendingModFolderStore";
import {
  saveModFolderHandle,
  loadModFolderHandle,
  clearModFolderHandle,
  ensureReadPermission,
  probeDirHandleAlive,
  type StoredModFolder,
} from "@/lib/modFolderHandleStore";
import { ImportAuditDialog } from "../ImportAuditDialog";
import {
  PreflightSummaryContent,
  type PreflightStat,
} from "../PreflightSummaryDialog";
import { WizardSteps } from "./WizardSteps";
import { deriveWizardStep } from "./wizardStep";
import { ImportProgressPanel } from "../ImportProgressPanel";

import { emptyScan, applyModeDefaults, selectCountryAuditFiles } from "./scan";
import { performScanInBrowser } from "./scan-client";
import type {
  ScanResults,
  ScanCategory,
  ScanCategoryKey,
  Phase,
  UnifiedModImportProps,
  CategoryGroup,
  SilentSeedResult,
} from "./types";
import { CountryScopeBanner } from "./CountryScopeBanner";
import { CategoryGroups } from "./CategoryGroups";
import { ImportSuccessCard } from "./ImportSuccessCard";
import { SilentSeedBanner } from "./SilentSeedBanner";
import { ExpandConfirmDialog } from "./ExpandConfirmDialog";
import { ImportInterruptDialogs } from "./ImportInterruptDialogs";
import { ImportLanguageSelector } from "./ImportLanguageSelector";
import { uploadInWindows } from "./uploadInWindows";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";
import {
  detectedContentLocales,
  localisationFileCountsByLocale,
  mergeProjectContentLanguages,
  shouldImportLocalisationPath,
} from "@/lib/i18n/hoi4ContentLanguages";

const DEFAULT_COLLAPSED_GROUPS: Record<string, boolean> = {
  characters: true,
  content: false,
  scenario: true,
};

export function UnifiedModImport({
  projectId,
  countryTag,
  onComplete,
  isLoadExistingMod = false,
  onBackgroundToggle,
  collapsed = false,
}: UnifiedModImportProps) {
  const t = useTranslations("unifiedModImport");
  const tImport = useTranslations("import");
  const tBg = useTranslations("backgroundImport");

  const tPreflight = useTranslations("preflightDialog");
  const router = useRouter();

  const localeRaw = useLocale();
  const progressLocale: AppLocale = isAppLocale(localeRaw) ? localeRaw : "en";
  const projectCountries = useProjectCountries();
  const fetchProject = useProjectStore((s) => s.fetchProject);
  const setActiveCountryTag = useProjectStore((s) => s.setActiveCountryTag);
  const projectExportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages,
  );

  const projectName = useProjectStore((s) => s.currentProject?.name);
  const isMultiCountry = projectCountries.length > 1;

  const backgroundSession = useBackgroundImportSession();
  const setBackgroundImport = useSetBackgroundImport();
  const updateBackgroundImport = useUpdateBackgroundImport();
  const cancelBackgroundImport = useCancelBackgroundImport();
  const clearBackgroundImport = useClearBackgroundImport();

  const pendingFolderProjectId = usePendingModFolderProjectId();
  const pendingFolderHandle = usePendingModFolderHandle();
  const clearPendingModFolder = useClearPendingModFolder();

  const [whatsNextDetailOpen, setWhatsNextDetailOpen] = useState(false);

  const [countryListOpen, setCountryListOpen] = useState(false);
  const [countryFilter, setCountryFilter] = useState("");
  const [open, setOpen] = useState(false);
  const [phase, setPhase] = useState<Phase>("idle");
  const [scanStatus, setScanStatus] = useState("");

  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);

  const [runInBackground, setRunInBackground] = useState(false);

  const importSessionIdRef = useRef<string | null>(null);

  const [showCloseConfirm, setShowCloseConfirm] = useState(false);

  const [showCancelConfirm, setShowCancelConfirm] = useState(false);
  const [showAuditDialog, setShowAuditDialog] = useState(false);
  const [advancedCountrySeed, setAdvancedCountrySeed] = useState(false);
  const importRevisions = useRef<
    import("@/components/workbench/save-diagnostics/editRevisions").EditRevisions
  >({});
  const [scan, setScan] = useState<ScanResults>(emptyScan);

  const [scanCountryTag, setScanCountryTag] = useState("");
  const [selectedContentLanguages, setSelectedContentLanguages] = useState<
    ContentLocale[]
  >(["en"]);
  const [result, setResult] = useState<ScanAndParseResult | null>(null);
  const [imageStatus, setImageStatus] = useState("");

  const [imageExtSwaps, setImageExtSwaps] = useState(0);

  const [uploadProgress, setUploadProgress] = useState<{
    loaded: number;
    total: number;
    lengthComputable: boolean;
  } | null>(null);

  const uploadAbortRef = useRef<AbortController | null>(null);

  const importStartedAtRef = useRef<number>(0);

  const userCancelledRef = useRef(false);

  const imageProgressRef = useRef<{ processed: number; total: number }>({
    processed: 0,
    total: 0,
  });
  const [error, setError] = useState<string | null>(null);
  const [conflictPolicy, setConflictPolicy] =
    useState<ConflictPolicy>("overwrite");

  const [reImporting, setReImporting] = useState(false);

  const [reImportTag, setReImportTag] = useState<string | null>(null);

  const [pendingAddTag, setPendingAddTag] = useState<string | null>(null);
  const [addAcknowledged, setAddAcknowledged] = useState(false);
  const dirHandleRef = useRef<FileSystemDirectoryHandle | null>(null);

  const scanGenerationRef = useRef(0);
  const scanAbortRef = useRef<AbortController | null>(null);
  const cancelActiveScan = useCallback(() => {
    scanGenerationRef.current += 1;
    scanAbortRef.current?.abort();
    scanAbortRef.current = null;
  }, []);
  const beginScan = useCallback(() => {
    scanAbortRef.current?.abort();
    const generation = scanGenerationRef.current + 1;
    scanGenerationRef.current = generation;
    const controller = new AbortController();
    scanAbortRef.current = controller;
    return {
      controller,
      isCurrent: () =>
        scanGenerationRef.current === generation && !controller.signal.aborted,
      release: () => {
        if (scanAbortRef.current === controller) scanAbortRef.current = null;
      },
    };
  }, []);

  const modFolderPickerRef = useRef<ModFolderPickerHandle | null>(null);

  const autoConsumedRef = useRef(false);

  useEffect(() => cancelActiveScan, [cancelActiveScan]);

  useEffect(() => {
    const handler = () => {
      if (!backgroundSession) return;
      if (backgroundSession.projectId !== projectId) return;
      setOpen(true);
    };
    window.addEventListener("background-import:open", handler);
    return () => window.removeEventListener("background-import:open", handler);
  }, [backgroundSession, projectId]);

  const [scanStats, setScanStats] = useState<ScanStatsResult | null>(null);
  const [scanStatsLoading, setScanStatsLoading] = useState(false);
  const [scanStatsError, setScanStatsError] = useState<string | null>(null);

  const groupCollapseStorageKey = `unified_import_group_collapsed_${projectId}`;
  const [collapsedGroups, setCollapsedGroups] = useState<
    Record<string, boolean>
  >(() => {
    if (typeof window === "undefined") return DEFAULT_COLLAPSED_GROUPS;
    try {
      return {
        ...DEFAULT_COLLAPSED_GROUPS,
        ...JSON.parse(localStorage.getItem(groupCollapseStorageKey) || "{}"),
      };
    } catch {
      return DEFAULT_COLLAPSED_GROUPS;
    }
  });

  const [scopeNoteOpen, setScopeNoteOpen] = useState(false);

  const [conflictPolicyOpen, setConflictPolicyOpen] = useState(false);

  const [picking, setPicking] = useState(false);
  const toggleGroupCollapsed = useCallback(
    (id: string) => {
      setCollapsedGroups((prev) => {
        const next = { ...prev, [id]: !prev[id] };
        try {
          localStorage.setItem(groupCollapseStorageKey, JSON.stringify(next));
        } catch {}
        return next;
      });
    },
    [groupCollapseStorageKey],
  );

  const reset = useCallback(() => {
    cancelActiveScan();
    setPhase("idle");
    setScanStatus("");
    setScan(emptyScan());
    setScanCountryTag("");
    setSelectedContentLanguages(["en"]);
    setResult(null);
    setImageStatus("");
    setImageExtSwaps(0);
    setError(null);
    setConflictPolicy("overwrite");
    setReImporting(false);
    setScanStats(null);
    setScanStatsLoading(false);
    setScanStatsError(null);
    setCurrentSessionId(null);
    setWhatsNextDetailOpen(false);

    setShowPreflight(false);
    dirHandleRef.current = null;
  }, [cancelActiveScan]);

  const [restoredFolder, setRestoredFolder] = useState<StoredModFolder | null>(
    null,
  );
  const [restoreBusy, setRestoreBusy] = useState(false);

  useEffect(() => {
    let alive = true;
    void loadModFolderHandle(projectId).then((rec) => {
      if (alive) setRestoredFolder(rec);
    });
    return () => {
      alive = false;
    };
  }, [projectId]);

  const handleOpen = () => {
    if (
      dirHandleRef.current &&
      (phase === "scanned" || phase === "parsing" || phase === "uploading")
    ) {
      setOpen(true);
      return;
    }
    reset();
    setOpen(true);
  };

  const runScan = useCallback(
    async (
      dirHandle: FileSystemDirectoryHandle,
    ): Promise<ScanResults | null> => {
      const scanRequest = beginScan();
      setPhase("scanning");
      setError(null);
      const targetTag = countryTag.trim().toUpperCase();

      try {
        const revisions = await readProjectEditRevisions(projectId);
        const results = await performScanInBrowser(
          dirHandle,
          targetTag,
          (status) => {
            if (scanRequest.isCurrent()) setScanStatus(status);
          },
          t,
          { signal: scanRequest.controller.signal },
        );
        if (!scanRequest.isCurrent()) return null;
        importRevisions.current = revisions;

        setScan(applyModeDefaults(results, isLoadExistingMod, projectId));
        setScanCountryTag(targetTag);

        setSelectedContentLanguages(["en"]);
        setPhase("scanned");
        setScanStatus("");

        setScanStatsLoading(false);
        return results;
      } catch (err) {
        if (!scanRequest.isCurrent()) return null;
        setError(
          `Scan failed: ${err instanceof Error ? err.message : "Unknown error"}`,
        );
        setPhase("idle");
        return null;
      } finally {
        scanRequest.release();
      }
    },
    [beginScan, countryTag, isLoadExistingMod, projectId, t],
  );

  const handleFolderPicked = useCallback(
    async (dirHandle: FileSystemDirectoryHandle) => {
      setPicking(false);
      dirHandleRef.current = dirHandle;

      void saveModFolderHandle(projectId, dirHandle).then(() =>
        loadModFolderHandle(projectId).then(setRestoredFolder),
      );

      setScanStatus(t("folderAccepted", { name: dirHandle.name }));
      await runScan(dirHandle);
    },
    [runScan, t, projectId],
  );

  const handleResumeStoredFolder = useCallback(async () => {
    if (!restoredFolder) return;
    setRestoreBusy(true);
    setError(null);
    try {
      const ok = await ensureReadPermission(restoredFolder.dirHandle);
      if (!ok) {
        await clearModFolderHandle(projectId);
        setRestoredFolder(null);
        setError(t("resumeFolderDenied"));
        return;
      }

      if (!(await probeDirHandleAlive(restoredFolder.dirHandle))) {
        await clearModFolderHandle(projectId);
        setRestoredFolder(null);
        setError(t("resumeFolderGone"));
        return;
      }
      reset();
      dirHandleRef.current = restoredFolder.dirHandle;
      setScanStatus(t("folderAccepted", { name: restoredFolder.name }));
      await runScan(restoredFolder.dirHandle);
    } finally {
      setRestoreBusy(false);
    }
  }, [restoredFolder, projectId, reset, runScan, t]);

  const handlePickFolder = () => {
    setError(null);
    setPicking(true);
    modFolderPickerRef.current?.pick();
  };

  const toggleCategory = (key: ScanCategoryKey) => {
    setScan((prev) => ({
      ...prev,
      [key]: { ...prev[key], enabled: !prev[key].enabled },
    }));
  };

  const totalFound =
    scan.characters.fileCount +
    scan.focus_trees.fileCount +
    scan.technologies.fileCount +
    scan.ideas.fileCount +
    scan.events.fileCount +
    scan.decisions.fileCount +
    scan.bookmarks.fileCount +
    scan.history_files.fileCount +
    scan.states.fileCount;

  const superEventCount = useMemo(
    () =>
      scan.events.files.filter((f) => {
        const base = f.path.split("/").pop()?.toLowerCase() ?? "";
        return base.endsWith("_se.txt") || base.endsWith("_super_event.txt");
      }).length,
    [scan.events.files],
  );

  const doImport = useCallback(
    async (overrideTag?: string): Promise<void> => {
      setPhase("parsing");
      setError(null);

      const effectiveTag = (overrideTag ?? scanCountryTag ?? countryTag)
        .trim()
        .toUpperCase();

      try {
        localStorage.setItem(
          `unified_import_last_selection_${projectId}`,
          JSON.stringify({
            characters: scan.characters.enabled,
            focus_trees: scan.focus_trees.enabled,
            technologies: scan.technologies.enabled,
            ideas: scan.ideas.enabled,
            events: scan.events.enabled,
            decisions: scan.decisions.enabled,
            bookmarks: scan.bookmarks.enabled,
            history_files: scan.history_files.enabled,
            states: scan.states.enabled,
          }),
        );
      } catch {}

      const sessionId = createUUID();
      setCurrentSessionId(sessionId);
      importSessionIdRef.current = sessionId;

      const pushBgUpdate = (
        stage:
          | "queued"
          | "scanning"
          | "parsing"
          | "uploading"
          | "done"
          | "error"
          | "cancelled",
        statusText: string,
        percent: number | null = null,
      ) => {
        if (!runInBackground) return;

        if (importSessionIdRef.current !== sessionId) return;
        updateBackgroundImport(sessionId, { stage, statusText, percent });
      };

      const importStartedAt = Date.now();

      importStartedAtRef.current = importStartedAt;
      userCancelledRef.current = false;
      imageProgressRef.current = { processed: 0, total: 0 };
      const buildSummary = (r: ScanAndParseResult) => {
        const parts = [
          {
            label: t("labelCharacters"),
            count: (r.characters?.created ?? 0) + (r.characters?.updated ?? 0),
          },
          {
            label: t("labelFocusTrees"),
            count:
              (r.focus_trees?.created_trees ?? 0) +
              (r.focus_trees?.updated_trees ?? 0),
          },
          {
            label: t("labelTechnologies"),
            count:
              (r.technologies?.created ?? 0) + (r.technologies?.updated ?? 0),
          },
          {
            label: t("labelIdeas"),
            count: (r.ideas?.created ?? 0) + (r.ideas?.updated ?? 0),
          },
          {
            label: t("labelEvents"),
            count:
              (r.events?.created_events ?? 0) + (r.events?.updated_events ?? 0),
          },
          {
            label: t("labelDecisions"),
            count:
              (r.decisions?.created_decisions ?? 0) +
              (r.decisions?.updated_decisions ?? 0),
          },
        ];
        const nonzero = parts.filter((p) => p.count > 0);
        const total = nonzero.reduce((n, p) => n + p.count, 0);
        const warnings = r.warnings?.length ?? 0;
        const lines = nonzero.map((p) => `${p.count} ${p.label}`);
        if (warnings > 0)
          lines.push(t("resultWarningsLine", { count: warnings }));
        return { lines, total, warnings };
      };
      const commitTerminal = (
        stage: "done" | "error" | "cancelled",
        resultPayload: BackgroundImportResult,
        finalMessage: string,
        percent: number | null,
      ) => {
        setBackgroundImport({
          sessionId,
          projectId,
          projectName: projectName || `Project #${projectId}`,
          countryTag: effectiveTag,
          stage,
          statusText: finalMessage,
          percent,
          etaSeconds: null,
          startedAt: importStartedAt,

          finishedAt: Date.now(),
          finalMessage,
          result: resultPayload,
        });
      };

      let parsedResult: ScanAndParseResult | null = null;

      try {
        setScanStatus(t("stage1Parsing"));
        pushBgUpdate("parsing", t("stage1Parsing"), null);
        const textFiles: File[] = [];
        const textPaths: string[] = [];

        const addFiles = (category: ScanCategory) => {
          for (const f of category.files) {
            textFiles.push(f.file);
            textPaths.push(f.path);
          }
        };

        console.log("[import] enabled flags before send:", {
          characters: scan.characters.enabled,
          focus_trees: scan.focus_trees.enabled,
          technologies: scan.technologies.enabled,
          ideas: scan.ideas.enabled,
          events: scan.events.enabled,
          decisions: scan.decisions.enabled,
          bookmarks: scan.bookmarks.enabled,
          history_files: scan.history_files.enabled,
          states: scan.states.enabled,
        });
        if (scan.characters.enabled) addFiles(scan.characters);
        if (scan.focus_trees.enabled) addFiles(scan.focus_trees);
        if (scan.technologies.enabled) addFiles(scan.technologies);
        if (scan.ideas.enabled) addFiles(scan.ideas);
        if (scan.events.enabled) addFiles(scan.events);
        if (scan.decisions.enabled) addFiles(scan.decisions);
        if (scan.bookmarks.enabled) addFiles(scan.bookmarks);
        if (scan.history_files.enabled) addFiles(scan.history_files);
        if (scan.states.enabled) addFiles(scan.states);
        addFiles(scan.gfx);

        addFiles(scan.gui);

        addFiles({
          ...scan.loc,
          files: scan.loc.files.filter((file) =>
            shouldImportLocalisationPath(file.path, selectedContentLanguages),
          ),
        });

        addFiles(scan.raids);
        addFiles(scan.units);
        addFiles(scan.map);
        addFiles(scan.flags);
        addFiles(scan.common_rules);

        if (textFiles.length === 0) {
          setError(t("errNoFilesSelected"));
          setPhase("scanned");
          return;
        }
        if (scan.descriptor) {
          textFiles.push(scan.descriptor.file);
          textPaths.push(scan.descriptor.path);
        }

        const totalBytes = textFiles.reduce((sum, file) => sum + file.size, 0);
        const totalMB = totalBytes / (1024 * 1024);
        setUploadProgress({
          loaded: 0,
          total: totalBytes,
          lengthComputable: totalBytes > 0,
        });
        setScanStatus(t("uploadStarting", { mb: totalMB.toFixed(0) }));
        pushBgUpdate(
          "parsing",
          t("uploadStarting", { mb: totalMB.toFixed(0) }),
          0,
        );
        if (totalMB >= 90)
          toast.warning(t("uploadLargeCaution", { mb: totalMB.toFixed(0) }));

        const controller = new AbortController();
        uploadAbortRef.current = controller;
        let lastPercent = -1;
        const parsed = await twoStageImportApi.scanAndParseBestAvailable(
          projectId,
          textFiles,
          textPaths,
          sessionId,
          {
            characters: scan.characters.enabled,
            focus: scan.focus_trees.enabled,
            tech: scan.technologies.enabled,
            ideas: scan.ideas.enabled,
            events: scan.events.enabled,
            decisions: scan.decisions.enabled,
            bookmarks: scan.bookmarks.enabled,
            history_files: scan.history_files.enabled,
            states: scan.states.enabled,
          },
          conflictPolicy,
          effectiveTag,
          {
            signal: controller.signal,
            onUploadProgress: (progress) => {
              const total = Math.max(progress.total, 1);
              const loaded = Math.min(progress.loaded, total);
              setUploadProgress({ loaded, total, lengthComputable: true });
              const percent = Math.floor((loaded / total) * 100);
              if (percent === 100) {
                setUploadProgress(null);
                setScanStatus(t("uploadDoneParsing"));
                pushBgUpdate("parsing", t("uploadDoneParsing"), null);
                return;
              }
              if (percent !== lastPercent) {
                lastPercent = percent;
                const message = t("uploadProgress", {
                  pct: percent,
                  done: (loaded / (1024 * 1024)).toFixed(0),
                  total: (total / (1024 * 1024)).toFixed(0),
                });
                setScanStatus(message);
                pushBgUpdate("parsing", message, percent);
              }
            },
          },
          importRevisions.current,
        );
        uploadAbortRef.current = null;
        setResult(parsed);
        parsedResult = parsed;
        const neededImages = parsed.needed_images || [];
        let failedImageCount = 0;
        console.log(
          "[import] Stage 1 done. needed_images:",
          neededImages.length,
          "dirHandle:",
          !!dirHandleRef.current,
        );

        if (neededImages.length > 0 && dirHandleRef.current) {
          setPhase("uploading");
          const chunkSize = 16;
          const found: Array<{ file: File; path: string }> = [];
          const missing: string[] = [];
          let foundCount = 0;
          setScanStatus(t("stage2Collecting"));
          pushBgUpdate("uploading", t("stage2Collecting"), 0);

          for (let index = 0; index < neededImages.length; index += chunkSize) {
            const chunk = neededImages.slice(index, index + chunkSize);
            const collected = await Promise.all(
              chunk.map(async (imagePath) => ({
                imagePath,
                file: await readFileByPath(dirHandleRef.current!, imagePath),
              })),
            );
            for (const item of collected) {
              if (item.file) {
                found.push({ file: item.file, path: item.imagePath });
                foundCount += 1;
              } else {
                missing.push(item.imagePath);
              }
            }
          }

          console.log(
            `[import] Base dir: found ${foundCount}, missing ${missing.length}`,
          );
          if (missing.length > 0) {
            setScanStatus(t("stage2SearchingDlc", { n: missing.length }));
            const dlcDir = await tryGetDir(dirHandleRef.current, "dlc");
            if (dlcDir) {
              const dlcSubdirs: FileSystemDirectoryHandle[] = [];
              for await (const entry of dlcDir.values()) {
                if (entry.kind === "directory") dlcSubdirs.push(entry);
              }
              for (let index = 0; index < missing.length; index += chunkSize) {
                const chunk = missing.slice(index, index + chunkSize);
                const collected = await Promise.all(
                  chunk.map(async (imagePath) => {
                    for (const dlcSubdir of dlcSubdirs) {
                      const file = await readFileByPath(dlcSubdir, imagePath);
                      if (file) return { file, path: imagePath };
                    }
                    for (const alternatePath of imageExtensionAlternates(
                      imagePath,
                    )) {
                      const rootFile = await readFileByPath(
                        dirHandleRef.current!,
                        alternatePath,
                      );
                      if (rootFile)
                        return { file: rootFile, path: alternatePath };
                      for (const dlcSubdir of dlcSubdirs) {
                        const file = await readFileByPath(
                          dlcSubdir,
                          alternatePath,
                        );
                        if (file) return { file, path: alternatePath };
                      }
                    }
                    return null;
                  }),
                );
                for (const item of collected) {
                  if (item) {
                    found.push(item);
                    foundCount += 1;
                  }
                }
              }
              console.log(
                `[import] After DLC search: total found ${foundCount}/${neededImages.length}`,
              );
            }
          }

          setImageStatus(
            t("imagesFound", { found: foundCount, total: neededImages.length }),
          );
          if (found.length > 0) {
            const batches: Array<{
              items: Array<{ file: File; path: string }>;
              bytes: number;
            }> = [];
            let batch = {
              items: [] as Array<{ file: File; path: string }>,
              bytes: 0,
            };
            for (const item of found) {
              const size = item.file.size;
              if (
                batch.items.length > 0 &&
                (batch.bytes + size > 64 * 1024 * 1024 ||
                  batch.items.length >= 150)
              ) {
                batches.push(batch);
                batch = { items: [], bytes: 0 };
              }
              batch.items.push(item);
              batch.bytes += size;
            }
            if (batch.items.length > 0) batches.push(batch);

            const totalImages = found.length;
            imageProgressRef.current = { processed: 0, total: totalImages };
            let portraitsMatched = 0;
            let goalIconsSaved = 0;
            const warnings = new Set<string>();
            let portraitsBaseline = 0;
            let extensionSwaps = 0;
            let processed = 0;
            const updateImageProgress = () => {
              const message = t("stage2Uploading", {
                processed,
                total: totalImages,
              });
              setScanStatus(message);
              pushBgUpdate(
                "uploading",
                message,
                10 + Math.round((processed / Math.max(1, totalImages)) * 90),
              );
            };

            console.log(
              `[import] Uploading ${totalImages} images in ${batches.length} byte-budgeted batch(es), 2 in flight`,
            );
            updateImageProgress();
            const uploadBatch = async (
              currentBatch: (typeof batches)[number],
            ) => {
              const uploaded = await twoStageImportApi.uploadImages(
                projectId,
                currentBatch.items.map((item) => item.file),
                currentBatch.items.map((item) => item.path),
                sessionId,
              );
              portraitsMatched += uploaded.portraits_matched;
              goalIconsSaved += uploaded.goal_icons_saved;
              portraitsBaseline += uploaded.portraits_vanilla_baseline ?? 0;
              const audit = uploaded.asset_audit;
              if (audit) {
                extensionSwaps +=
                  audit.idea_pictures_ext_swap +
                  audit.event_pictures_ext_swap +
                  audit.decision_icons_ext_swap +
                  audit.tech_icons_ext_swap;
              }
              for (const warning of uploaded.warnings || [])
                warnings.add(warning);
              processed += currentBatch.items.length;
              imageProgressRef.current.processed = processed;
              updateImageProgress();
            };
            const { failed, firstError } = await uploadInWindows(
              batches,
              2,
              uploadBatch,
              () => userCancelledRef.current,
            );
            if (failed.length > 0) {
              failedImageCount = failed.reduce(
                (sum, failedBatch) => sum + failedBatch.items.length,
                0,
              );
              console.warn(
                `[import] ${failed.length} image batch(es) failed, ${failedImageCount} images not uploaded`,
                firstError,
              );
              warnings.add(
                t("resultImagesRemaining", { remaining: failedImageCount }),
              );
            }

            const summary =
              portraitsBaseline > 0
                ? t("portraitsSummaryWithBaseline", {
                    matched: portraitsMatched,
                    baseline: portraitsBaseline,
                    modModified: portraitsMatched - portraitsBaseline,
                    icons: goalIconsSaved,
                  })
                : t("portraitsSummary", {
                    matched: portraitsMatched,
                    icons: goalIconsSaved,
                  });
            setImageStatus(summary);
            setImageExtSwaps(extensionSwaps);
            if (warnings.size > 0) {
              setResult(
                (previous) =>
                  previous && {
                    ...previous,
                    warnings: [...previous.warnings, ...warnings],
                  },
              );
            }
          } else {
            console.log("[import] No images collected, skipping upload");
            setImageStatus(t("noImagesOnDisk"));
          }
        }

        try {
          const languages = mergeProjectContentLanguages(
            projectExportLanguages,
            selectedContentLanguages,
          );
          await projectsApi.update(projectId, { export_languages: languages });
          await fetchProject(projectId);
        } catch {
          const warning = t("contentLanguagesSaveWarning");
          setResult(
            (previous) =>
              previous && {
                ...previous,
                warnings: [...previous.warnings, warning],
              },
          );
        }

        setPhase("done");
        setScanStatus("");
        const createdCount =
          (parsed.characters?.created ?? 0) +
          (parsed.focus_trees?.created_trees ?? 0) +
          (parsed.technologies?.created ?? 0) +
          (parsed.ideas?.created ?? 0) +
          (parsed.events?.created_events ?? 0) +
          (parsed.decisions?.created_decisions ?? 0);
        setShowPreflight(false);
        if (runInBackground && importSessionIdRef.current === sessionId) {
          const { lines, total } = buildSummary(parsed);
          if (failedImageCount > 0) {
            lines.push(
              t("resultImagesRemaining", { remaining: failedImageCount }),
            );
          }
          const headline =
            total > 0
              ? t("resultHeadlineSuccess", { count: total })
              : t("resultHeadlineNoNew");
          commitTerminal(
            "done",
            { status: "success", headline, lines },
            headline,
            100,
          );
          toast.success(t("toastDone", { count: createdCount }));
        }
        importSessionIdRef.current = null;
        if (effectiveTag) setActiveCountryTag(effectiveTag);
        await onComplete();
      } catch (err) {
        uploadAbortRef.current = null;
        setUploadProgress(null);
        const aborted =
          err instanceof DOMException && err.name === "AbortError";
        const timedOut =
          err instanceof DOMException && err.name === "TimeoutError";
        const message = err instanceof Error ? err.message : "Unknown error";
        const httpStatus = err instanceof ImportHttpError ? err.status : null;
        const userCancelled = userCancelledRef.current;
        let status: "cancelled" | "failed";
        let headline: string;
        let displayMessage: string;
        if (userCancelled) {
          status = "cancelled";
          headline = t("resultHeadlineCancelled");
          displayMessage = t(
            parsedResult ? "uploadCancelledPartialMsg" : "uploadCancelledMsg",
          );
        } else if (aborted || timedOut) {
          status = "failed";
          headline = t("resultHeadlineTimeout");
          displayMessage = t("uploadTimeoutMsg");
        } else if (message.includes("network error during upload")) {
          status = "failed";
          headline = t("resultHeadlineNetwork");
          displayMessage = t("uploadNetworkMsg");
        } else if (httpStatus !== null && httpStatus >= 500) {
          status = "failed";
          headline = t("resultHeadlineFailed");
          displayMessage = t("serverUnavailableMsg", { status: httpStatus });
        } else {
          status = "failed";
          headline = t("resultHeadlineFailed");
          displayMessage = message;
        }

        const safeStop = userCancelled || aborted || timedOut;
        const lines = parsedResult ? buildSummary(parsedResult).lines : [];
        if (parsedResult && lines.length > 0) {
          lines.unshift(t("resultPartialNote"));
        } else if (!parsedResult) {
          lines.push(
            t(safeStop ? "resultNothingWrittenSafe" : "resultNothingWritten"),
          );
        }
        const imageProgress = imageProgressRef.current;
        if (imageProgress.total > 0) {
          lines.push(
            t("resultImagesProgress", {
              done: imageProgress.processed,
              total: imageProgress.total,
            }),
          );
          const remaining = Math.max(
            0,
            imageProgress.total - imageProgress.processed,
          );
          if (remaining > 0)
            lines.push(t("resultImagesRemaining", { remaining }));
        }

        setError(displayMessage);
        setPhase("scanned");
        if (userCancelled) toast.info(tBg("toastCancelled"));
        else toast.error(t("toastFail", { msg: displayMessage }));
        commitTerminal(
          status === "cancelled" ? "cancelled" : "error",
          { status, headline, lines },
          displayMessage,
          null,
        );
        importSessionIdRef.current = null;
        if (userCancelled || aborted || timedOut) return;
        throw err;
      }
    },
    [
      projectId,
      countryTag,
      scanCountryTag,
      scan,
      conflictPolicy,
      onComplete,
      t,
      tBg,
      runInBackground,
      updateBackgroundImport,
      setBackgroundImport,
      projectName,
      selectedContentLanguages,
      projectExportLanguages,
      fetchProject,
      setActiveCountryTag,
    ],
  );

  const [showPreflight, setShowPreflight] = useState(false);

  const [backgroundMode, setBackgroundMode] = useState(false);
  const [silentSeedResult, setSilentSeedResult] =
    useState<SilentSeedResult | null>(null);

  const countryPreparationInFlight =
    silentSeedResult?.phase === "auditing" ||
    silentSeedResult?.phase === "creating";

  const wizardStep = deriveWizardStep({
    phase,
    showPreflight,
    reImporting,
    countryPreparationInFlight,
  });
  const detectedContentLanguages = useMemo(
    () => detectedContentLocales(scan.loc.files.map((file) => file.path)),
    [scan.loc.files],
  );
  const contentLanguageFileCounts = useMemo(
    () =>
      localisationFileCountsByLocale(scan.loc.files.map((file) => file.path)),
    [scan.loc.files],
  );

  const selectedToggleFileCount =
    (scan.characters.enabled ? scan.characters.fileCount : 0) +
    (scan.focus_trees.enabled ? scan.focus_trees.fileCount : 0) +
    (scan.technologies.enabled ? scan.technologies.fileCount : 0) +
    (scan.ideas.enabled ? scan.ideas.fileCount : 0) +
    (scan.events.enabled ? scan.events.fileCount : 0) +
    (scan.decisions.enabled ? scan.decisions.fileCount : 0) +
    (scan.bookmarks.enabled ? scan.bookmarks.fileCount : 0) +
    (scan.history_files.enabled ? scan.history_files.fileCount : 0) +
    (scan.states.enabled ? scan.states.fileCount : 0);

  const runSilentSeed = useCallback(async (): Promise<boolean> => {
    if (scan.country_tags.fileCount === 0) return true;
    setSilentSeedResult({
      phase: "auditing",
      declared: 0,
      created: 0,
      skippedDynamic: 0,
      skippedExisting: 0,
    });
    try {
      const auditFormData = new FormData();
      selectCountryAuditFiles(scan).forEach((f) => {
        auditFormData.append("files", f.file);
        auditFormData.append("paths", f.path);
      });
      const auditRes = await fetch(
        `/api/go/projects/${projectId}/import/audit-tags`,
        { method: "POST", body: auditFormData },
      );
      if (!auditRes.ok) {
        const failure = (await auditRes.json().catch(() => ({}))) as {
          detail?: string;
        };
        throw new Error(
          failure.detail || `Audit failed: HTTP ${auditRes.status}`,
        );
      }
      const audit = await auditRes.json();

      type AuditDetected = {
        tag: string;
        is_dynamic: boolean;
        already_exists: boolean;
        name_preview?: { en: string | null; zh: string | null };
        color_preview?: number[] | null;
      };
      const toCreate: AuditDetected[] = audit.detected.filter(
        (d: AuditDetected) => !d.is_dynamic && !d.already_exists,
      );

      const samples = toCreate
        .map((d) => ({
          tag: d.tag,
          name: d.name_preview?.en || d.name_preview?.zh || d.tag,
          color: d.color_preview ?? null,
        }))
        .filter((s) => s.name !== s.tag || s.color !== null)
        .slice(0, 5);
      setSilentSeedResult({
        phase: "creating",
        declared: audit.summary.declared,
        created: 0,
        skippedDynamic: audit.summary.dynamic,
        skippedExisting: audit.summary.existing,
        presetName: audit.descriptor_name ?? null,
        samples,
      });
      let created = 0;
      for (const d of toCreate) {
        const namePref = d.name_preview;
        const displayedName = namePref?.en || namePref?.zh || d.tag;
        const r = await fetch(
          `${API_BASE}/go/projects/${projectId}/countries`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              tag: d.tag,
              name: displayedName,
              name_en: namePref?.en || "",
              name_zh: namePref?.zh || "",
              color: d.color_preview || [],
              ideology: "neutrality",
            }),
          },
        );

        if (r.ok || r.status === 409) {
          created += 1;
          setSilentSeedResult((prev) => prev && { ...prev, created });
        } else {
          const body = await r.json().catch(() => ({}));
          throw new Error(
            `Failed to create ${d.tag}: ${body.detail || r.statusText}`,
          );
        }
      }
      await fetchProject(projectId);
      setSilentSeedResult(
        (prev) => prev && { ...prev, phase: "done", created },
      );
      return true;
    } catch (e) {
      setSilentSeedResult(
        (prev) =>
          prev && { ...prev, phase: "error", errorMsg: (e as Error).message },
      );
      return false;
    }
  }, [scan, projectId, fetchProject]);

  const handleImport = async () => {
    if (scan.country_tags.fileCount === 0) {
      void doImport();
      return;
    }
    if (advancedCountrySeed) {
      setShowAuditDialog(true);
      return;
    }

    const ok = await runSilentSeed();
    if (ok) void doImport();
  };

  const handlePreflightOpen = () => {
    setShowPreflight(true);
  };

  const handlePreflightContinue = () => {
    setShowPreflight(false);
    void handleImport();
  };

  const handlePreflightCancel = () => {
    setShowPreflight(false);
  };

  useEffect(() => {
    if (autoConsumedRef.current) return;
    if (!pendingFolderHandle) return;
    if (pendingFolderProjectId !== projectId) return;

    autoConsumedRef.current = true;
    const handle = pendingFolderHandle;

    clearPendingModFolder();

    reset();
    dirHandleRef.current = handle;

    void saveModFolderHandle(projectId, handle).then(() =>
      loadModFolderHandle(projectId).then(setRestoredFolder),
    );
    setOpen(true);

    void runScan(handle);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pendingFolderHandle, pendingFolderProjectId, projectId]);

  const handleBackgroundToggle = (value: boolean) => {
    setBackgroundMode(value);
    setRunInBackground(value);
    onBackgroundToggle?.(value);
  };

  const preflightStats: PreflightStat[] = useMemo(() => {
    const totalFromTri = (
      k: keyof NonNullable<typeof scanStats>["entity_stats"],
    ) => {
      const tri = scanStats?.entity_stats[k];
      if (!tri) return undefined;
      return tri.new + tri.modified + tri.unchanged;
    };
    const sublineFromTri = (
      k: keyof NonNullable<typeof scanStats>["entity_stats"],
    ) => {
      const tri = scanStats?.entity_stats[k];
      if (!tri) return undefined;
      const parts: string[] = [];
      if (tri.new > 0) parts.push(t("triNew", { n: tri.new }));
      if (tri.modified > 0) parts.push(t("triModified", { n: tri.modified }));
      if (tri.unchanged > 0)
        parts.push(t("triUnchanged", { n: tri.unchanged }));
      return parts.length > 0 ? parts.join(" · ") : undefined;
    };

    const charsCount =
      totalFromTri("characters") ??
      (scan.characters.fileCount > 0 ? scan.characters.fileCount : undefined);
    const ideasCount =
      totalFromTri("ideas") ??
      (scan.ideas.fileCount > 0 ? scan.ideas.fileCount : undefined);
    const focusCount =
      totalFromTri("focus_trees") ??
      (scan.focus_trees.fileCount > 0 ? scan.focus_trees.fileCount : undefined);
    const eventsCount =
      totalFromTri("events") ??
      (scan.events.fileCount > 0 ? scan.events.fileCount : undefined);
    const decisionsCount =
      totalFromTri("decisions") ??
      (scan.decisions.fileCount > 0 ? scan.decisions.fileCount : undefined);

    const declaredTags = scan.country_tags.declaredTagCount ?? 0;
    const countriesCount =
      declaredTags > 0
        ? declaredTags
        : projectCountries.length > 0
          ? projectCountries.length
          : undefined;

    return [
      { key: "countries", count: countriesCount },
      {
        key: "characters",
        count: charsCount,
        subline: sublineFromTri("characters"),
      },
      { key: "ideas", count: ideasCount, subline: sublineFromTri("ideas") },
      {
        key: "focus_trees",
        count: focusCount,
        subline: sublineFromTri("focus_trees"),
      },
      { key: "events", count: eventsCount, subline: sublineFromTri("events") },
      {
        key: "decisions",
        count: decisionsCount,
        subline: sublineFromTri("decisions"),
      },

      {
        key: "dynamic_modifiers",
        count: scan.common_rules.dirCounts?.dynamic_modifiers,
      },
      {
        key: "superevents",
        count: superEventCount > 0 ? superEventCount : undefined,
      },
    ];
  }, [scan, scanStats, projectCountries, superEventCount, t]);

  const preflightTotals = useMemo(() => {
    let bytes = 0;
    let files = 0;
    const accumulate = (cat: ScanCategory) => {
      for (const f of cat.files) {
        bytes += f.file.size;
        files += 1;
      }
    };
    if (scan.characters.enabled) accumulate(scan.characters);
    if (scan.focus_trees.enabled) accumulate(scan.focus_trees);
    if (scan.technologies.enabled) accumulate(scan.technologies);
    if (scan.ideas.enabled) accumulate(scan.ideas);
    if (scan.events.enabled) accumulate(scan.events);
    if (scan.decisions.enabled) accumulate(scan.decisions);
    if (scan.bookmarks.enabled) accumulate(scan.bookmarks);
    if (scan.history_files.enabled) accumulate(scan.history_files);
    if (scan.states.enabled) accumulate(scan.states);

    accumulate(scan.gfx);
    accumulate(scan.gui);
    accumulate({
      ...scan.loc,
      files: scan.loc.files.filter((file) =>
        shouldImportLocalisationPath(file.path, selectedContentLanguages),
      ),
    });
    accumulate(scan.raids);
    accumulate(scan.units);
    accumulate(scan.common_rules);
    accumulate(scan.map);
    accumulate(scan.flags);
    return { bytes, files };
  }, [scan, selectedContentLanguages]);

  const commonRulesSummary = useMemo(() => {
    const counts = scan.common_rules.dirCounts;
    if (!counts) return "";
    const labels: Record<string, string> = {
      dynamic_modifiers: t("commonRulesDirDynamicModifiers"),
      autonomous_states: t("commonRulesDirAutonomousStates"),
      intelligence_agencies: t("commonRulesDirIntelligenceAgencies"),
      scripted_effects: t("commonRulesDirScriptedEffects"),
      scripted_triggers: t("commonRulesDirScriptedTriggers"),
      scripted_localisation: t("commonRulesDirScriptedLocalisation"),
      doctrines: t("commonRulesDirDoctrines"),
      on_actions: t("commonRulesDirOnActions"),
      ideologies: t("commonRulesDirIdeologies"),
    };
    return Object.entries(counts)
      .filter(([, n]) => n > 0)
      .map(([dir, n]) => `${labels[dir] ?? dir} ${n}`)
      .join(" · ");
  }, [scan.common_rules.dirCounts, t]);

  const handleAddCountryAndReImport = async (newTag: string) => {
    if (!dirHandleRef.current) {
      setError(t("rescanHandleLost"));
      return;
    }
    if (reImporting) return;
    setPendingAddTag(newTag.toUpperCase());
    setAddAcknowledged(false);
  };

  const performAddCountryAndReImport = async () => {
    if (!pendingAddTag || !dirHandleRef.current) return;
    const upperTag = pendingAddTag;
    setPendingAddTag(null);
    setAddAcknowledged(false);

    setReImporting(true);

    setReImportTag(upperTag);
    setError(null);
    const scanRequest = beginScan();

    try {
      setScanStatus(t("addingCountry", { tag: upperTag }));
      const res = await fetch(
        `${API_BASE}/go/projects/${projectId}/countries`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            tag: upperTag,
            name: upperTag,
            ideology: "neutrality",
          }),
          signal: scanRequest.controller.signal,
        },
      );
      if (!scanRequest.isCurrent()) return;
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(
          body.detail || t("addCountryFailed", { status: res.status }),
        );
      }
      await fetchProject(projectId);
      if (!scanRequest.isCurrent()) return;

      setScanStatus(t("rescanWithTag", { tag: upperTag }));
      const revisions = await readProjectEditRevisions(projectId);
      const newScan = await performScanInBrowser(
        dirHandleRef.current,
        upperTag,
        (status) => {
          if (scanRequest.isCurrent()) setScanStatus(status);
        },
        t,
        { signal: scanRequest.controller.signal },
      );
      if (!scanRequest.isCurrent()) return;
      setScan(newScan);
      importRevisions.current = revisions;

      await doImport(upperTag);
    } catch (err) {
      if (!scanRequest.isCurrent()) return;
      setError(
        t("reimportFailed", {
          msg: err instanceof Error ? err.message : "Unknown error",
        }),
      );
    } finally {
      scanRequest.release();
      setReImporting(false);
      setReImportTag(null);
    }
  };

  const importInFlight = phase === "parsing" || phase === "uploading";

  const handleKeepInBackground = useCallback(() => {
    const sessionId = importSessionIdRef.current;
    if (!sessionId) {
      setShowCloseConfirm(false);
      setOpen(false);
      return;
    }
    setRunInBackground(true);
    setBackgroundImport({
      sessionId,
      projectId,
      projectName: projectName || `Project #${projectId}`,
      countryTag: countryTag || "",
      stage: phase === "uploading" ? "uploading" : "parsing",
      statusText: scanStatus || imageStatus || tBg("stageParsing"),
      percent: null,
      etaSeconds: null,

      startedAt: importStartedAtRef.current || Date.now(),
    });
    setShowCloseConfirm(false);
    setOpen(false);
    toast.info(tBg("toastBackgrounded"));
  }, [
    projectId,
    projectName,
    countryTag,
    phase,
    scanStatus,
    imageStatus,
    setBackgroundImport,
    tBg,
  ]);

  const handleDialogOpenChange = useCallback(
    (next: boolean) => {
      if (next) {
        setOpen(true);
        return;
      }
      if (importInFlight) {
        if (backgroundMode) {
          handleKeepInBackground();
          return;
        }

        setShowCloseConfirm(true);
        return;
      }
      if (phase === "scanning" || reImporting) {
        reset();
      }
      setOpen(false);
    },
    [
      importInFlight,
      backgroundMode,
      handleKeepInBackground,
      phase,
      reImporting,
      reset,
    ],
  );

  const handleConfirmCancel = useCallback(async () => {
    const sessionId = importSessionIdRef.current;

    userCancelledRef.current = true;
    setShowCancelConfirm(false);
    setShowCloseConfirm(false);

    if (uploadAbortRef.current) {
      uploadAbortRef.current.abort();
      uploadAbortRef.current = null;
    }
    setUploadProgress(null);
    if (sessionId) {
      if (runInBackground) {
        await cancelBackgroundImport();
      } else {
        try {
          await fetch(
            `/api/go/import/jobs/${projectId}/${encodeURIComponent(sessionId)}/cancel`,
            { method: "POST", credentials: "same-origin" },
          );
        } catch (err) {
          console.warn("[import] cancel POST failed:", err);
        }
      }
    }
    importSessionIdRef.current = null;
    setCurrentSessionId(null);
    setRunInBackground(false);
    clearBackgroundImport();

    setPhase("scanned");

    setShowPreflight(false);
    setScanStatus("");
    setError(t("toastFail", { msg: tBg("cancelledByUser") }));
    toast.info(tBg("toastCancelled"));
  }, [
    projectId,
    runInBackground,
    cancelBackgroundImport,
    clearBackgroundImport,
    t,
    tBg,
  ]);

  const tag = scanCountryTag || countryTag?.toUpperCase() || "";
  const CATEGORY_GROUPS: CategoryGroup[] = [
    {
      id: "characters",
      label: t("groupCharacters"),
      items: [
        {
          key: "characters",
          label: t("itemCharacters"),
          icon: Users,
          desc: (s) =>
            t("descCharacters", { count: s.characters.fileCount, tag }),
        },
      ],
    },
    {
      id: "content",
      label: t("groupContent"),
      items: [
        {
          key: "focus_trees",
          label: t("itemFocusTrees"),
          icon: Target,
          desc: (s) =>
            t("descFocusTrees", { count: s.focus_trees.fileCount, tag }),
        },
        {
          key: "technologies",
          label: t("itemTechnologies"),
          icon: FlaskConical,
          desc: (s) =>
            t("descTechnologies", { count: s.technologies.fileCount }),
        },
        {
          key: "ideas",
          label: t("itemIdeas"),
          icon: Lightbulb,
          desc: (s) => t("descIdeas", { count: s.ideas.fileCount, tag }),
        },
        {
          key: "events",
          label: t("itemEvents"),
          icon: CalendarClock,
          desc: (s) => t("descEvents", { count: s.events.fileCount, tag }),
        },
        {
          key: "decisions",
          label: t("itemDecisions"),
          icon: Gavel,
          desc: (s) =>
            t("descDecisions", { count: s.decisions.fileCount, tag }),
        },
      ],
    },
    {
      id: "scenario",
      label: t("groupScenario"),
      items: [
        {
          key: "bookmarks",
          label: t("itemBookmarks"),
          icon: Bookmark,
          desc: (s) => t("descBookmarks", { count: s.bookmarks.fileCount }),
        },
        {
          key: "history_files",
          label: t("itemHistoryFiles"),
          icon: ScrollText,
          desc: (s) =>
            t("descHistoryFiles", { count: s.history_files.fileCount, tag }),
        },
        {
          key: "states",
          label: t("itemStates"),
          icon: Map,
          desc: (s) => t("descStates", { count: s.states.fileCount }),
        },
      ],
    },
  ];

  return (
    <TooltipProvider>
      <>
        <Button
          size="sm"
          onClick={handleOpen}
          title={collapsed ? t("triggerLabel") : undefined}
          aria-label={collapsed ? t("triggerLabel") : undefined}

          className={`w-full h-auto gap-2.5 rounded-lg border border-transparent py-2 text-sm bg-accent text-accent-foreground font-semibold hover:bg-accent/85 ${
            collapsed ? "justify-center px-0" : "justify-start px-3"
          }`}
        >
          <Upload className="size-4 shrink-0" strokeWidth={1.8} aria-hidden />
          {!collapsed && t("triggerLabel")}
        </Button>

        <Dialog
          open={open}
          onOpenChange={handleDialogOpenChange}
          disablePointerDismissal={phase !== "idle"}
        >
          <DialogContent className="sm:max-w-2xl flex flex-col gap-0 overflow-y-hidden p-0">
            <DialogHeader className="shrink-0 border-b border-border px-4 pt-4 pb-3">
              <DialogTitle className="flex items-center gap-2">
                {t("dialogTitle")}
                <InfoTooltip
                  title={t("tooltipTitle")}
                  content={t.raw("tooltipContent")}
                />
              </DialogTitle>

              <DialogDescription
                className={phase === "idle" ? undefined : "sr-only"}
              >
                {t("dialogDescription")}
              </DialogDescription>
            </DialogHeader>

            <div className="shrink-0 border-b border-border px-4 py-2">
              <WizardSteps
                current={wizardStep}
                finished={phase === "done" && !reImporting}
              />
            </div>

            <div className="min-h-0 flex-1 space-y-3 overflow-y-auto px-4 py-3">
              <ModFolderPicker
                ref={modFolderPickerRef}
                mode="read"
                onPicked={handleFolderPicked}
                onError={(m) => setError(m)}
                onSettled={() => setPicking(false)}
              />

              {wizardStep < 4 && (
                <CountryScopeBanner
                  isMultiCountry={isMultiCountry}
                  t={t}
                  tag={tag}
                  countryListOpen={countryListOpen}
                  setCountryListOpen={setCountryListOpen}
                  projectCountries={projectCountries}
                  countryFilter={countryFilter}
                  setCountryFilter={setCountryFilter}
                />
              )}

              {wizardStep === 4 &&
                !(phase === "done" && !reImporting) &&
                tag && (
                  <p className="text-xs text-muted-foreground">
                    {t("importingAsTag", { tag: reImportTag ?? tag })}
                  </p>
                )}

              <div className="space-y-3">
                {(phase === "scanning" ||
                  countryPreparationInFlight ||
                  reImporting) && (
                  <div
                    className="rounded-md border border-accent/50 bg-accent/10 px-3 py-3 text-sm"
                    role="status"
                    aria-live="polite"
                  >
                    <p className="font-semibold">
                      {phase === "scanning"
                        ? t("folderAcceptedHeadline")
                        : countryPreparationInFlight
                          ? t("preparingCountriesHeadline")
                          : t("reimportingHeadline")}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {phase === "scanning"
                        ? scanStatus || t("scanning")
                        : countryPreparationInFlight
                          ? silentSeedResult?.phase === "auditing"
                            ? tImport("silentSeed.auditing")
                            : silentSeedResult?.presetName
                              ? tImport("silentSeed.creatingWithPreset", {
                                  preset: silentSeedResult.presetName,
                                  created: silentSeedResult.created,
                                  total:
                                    silentSeedResult.declared -
                                    silentSeedResult.skippedDynamic -
                                    silentSeedResult.skippedExisting,
                                })
                              : tImport("silentSeed.creatingNoPreset", {
                                  created: silentSeedResult?.created ?? 0,
                                  total:
                                    (silentSeedResult?.declared ?? 0) -
                                    (silentSeedResult?.skippedDynamic ?? 0) -
                                    (silentSeedResult?.skippedExisting ?? 0),
                                })
                          : scanStatus || t("parsing")}
                    </p>
                    {phase === "scanning" && (
                      <Progress className="mt-2" indeterminate />
                    )}
                  </div>
                )}

                {wizardStep === 1 && (
                  <div className="space-y-2">
                    <BrowserCompatHint mode="folderUpload" />
                    {picking && phase === "idle" && (
                      <div
                        className="rounded-md border border-border bg-muted/40 px-3 py-2 text-xs leading-relaxed text-muted-foreground"
                        role="status"
                        aria-live="polite"
                      >
                        {t("pickingHint")}
                        <Progress className="mt-2" indeterminate />
                      </div>
                    )}

                    {restoredFolder && (
                      <>
                        <Button
                          onClick={() => void handleResumeStoredFolder()}
                          disabled={
                            restoreBusy || phase === "scanning" || reImporting
                          }
                          className="w-full"
                        >
                          {restoreBusy
                            ? t("scanning")
                            : t("resumeFolder", { name: restoredFolder.name })}
                        </Button>
                        <p className="text-center text-xs text-muted-foreground">
                          {t("resumeFolderHint")}
                        </p>
                      </>
                    )}
                    <Button
                      onClick={handlePickFolder}
                      disabled={phase === "scanning" || reImporting}
                      variant={restoredFolder ? "outline" : "default"}
                      className="w-full"
                    >
                      {phase === "scanning"
                        ? t("scanning")
                        : restoredFolder
                          ? t("selectAnotherFolder")
                          : t("selectFolder")}
                    </Button>
                    {scanStatus && (
                      <p className="text-xs text-muted-foreground text-center">
                        {scanStatus}
                      </p>
                    )}
                  </div>
                )}

                {wizardStep === 2 && (
                  <div className="space-y-2">
                    {totalFound === 0 ? (
                      <p className="text-sm text-muted-foreground text-center py-4">
                        {t("noModDataFound")}
                      </p>
                    ) : (
                      <>
                        <p className="flex flex-wrap items-baseline gap-x-2 text-xs text-muted-foreground">
                          <span className="shrink-0">
                            {t("foundSummary", {
                              files: totalFound,
                              gfx: scan.gfx.fileCount,
                              loc: scan.loc.fileCount,
                            })}
                          </span>

                          {(scan.common_rules.fileCount > 0 ||
                            scan.raids.fileCount > 0 ||
                            scan.units.fileCount > 0) &&
                            (() => {
                              const alsoIncluded = `${t("alsoIncludedPrefix")} ${[
                                commonRulesSummary,
                                scan.raids.fileCount > 0
                                  ? `${t("alsoIncludedRaids")} ${scan.raids.fileCount}`
                                  : "",

                                scan.units.fileCount > 0
                                  ? `${t("alsoIncludedUnits")} ${scan.units.fileCount}`
                                  : "",
                              ]
                                .filter(Boolean)
                                .join(" · ")}`;
                              return (
                                <span
                                  className="min-w-0 flex-1 truncate"
                                  title={alsoIncluded}
                                >
                                  {alsoIncluded}
                                </span>
                              );
                            })()}
                        </p>

                        {isLoadExistingMod && (
                          <div className="rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-1.5 text-xs leading-relaxed">
                            <button
                              type="button"
                              onClick={() => setScopeNoteOpen((v) => !v)}
                              aria-expanded={scopeNoteOpen}
                              className="flex w-full items-baseline gap-1.5 text-left"
                            >
                              <span className="shrink-0 text-muted-foreground">
                                {scopeNoteOpen ? "▼" : "▶"}
                              </span>
                              <span className="shrink-0 font-medium text-amber-600 dark:text-amber-400">
                                ⚠ {t("scopeNotYetTitle")}
                              </span>
                              <span className="min-w-0 flex-1 truncate text-muted-foreground">
                                {t("scopeNotYetList")}
                              </span>
                            </button>
                            {scopeNoteOpen && (
                              <p className="mt-1 pl-4 text-muted-foreground">
                                {t("scopeExportSteer")}
                              </p>
                            )}
                          </div>
                        )}

                        {scan.loc.fileCount > 0 ? (
                          <ImportLanguageSelector
                            detected={detectedContentLanguages}
                            counts={contentLanguageFileCounts}
                            selected={selectedContentLanguages}
                            onChange={setSelectedContentLanguages}
                          />
                        ) : null}

                        <CategoryGroups
                          CATEGORY_GROUPS={CATEGORY_GROUPS}
                          scan={scan}
                          collapsedGroups={collapsedGroups}
                          toggleGroupCollapsed={toggleGroupCollapsed}
                          scanStats={scanStats}
                          scanStatsLoading={scanStatsLoading}
                          scanStatsError={scanStatsError}
                          toggleCategory={toggleCategory}
                          phase={phase}
                          superEventCount={superEventCount}
                          t={t}
                        />

                        {phase === "scanned" && (
                          <div className="rounded-md border">
                            <button
                              type="button"
                              onClick={() => setConflictPolicyOpen((v) => !v)}
                              aria-expanded={conflictPolicyOpen}
                              className="flex w-full items-center justify-between px-2.5 py-1.5 text-xs font-medium transition-colors hover:bg-muted/40"
                            >
                              <span className="flex items-center gap-2">
                                <span className="text-muted-foreground">
                                  {conflictPolicyOpen ? "▼" : "▶"}
                                </span>
                                <span>{t("conflictPolicyTitle")}</span>
                              </span>
                              <span className="text-muted-foreground">
                                {conflictPolicy === "overwrite"
                                  ? t("overwriteLabel")
                                  : t("skipLabel")}
                              </span>
                            </button>
                            {conflictPolicyOpen && (
                              <div className="space-y-2 p-2.5 pt-0">
                                <label className="flex items-start gap-2 cursor-pointer">
                                  <input
                                    type="radio"
                                    name="conflict-policy"
                                    value="overwrite"
                                    checked={conflictPolicy === "overwrite"}
                                    onChange={() =>
                                      setConflictPolicy("overwrite")
                                    }
                                    className="accent-primary mt-0.5"
                                  />
                                  <div>
                                    <p className="text-xs">
                                      <span className="font-medium">
                                        {t("overwriteLabel")}
                                      </span>
                                      <span className="text-muted-foreground">
                                        {t("overwriteDesc")}
                                      </span>
                                    </p>
                                    <p className="text-xs text-amber-600">
                                      {t("overwriteDetail")}
                                    </p>
                                  </div>
                                </label>
                                <label className="flex items-start gap-2 cursor-pointer">
                                  <input
                                    type="radio"
                                    name="conflict-policy"
                                    value="skip"
                                    checked={conflictPolicy === "skip"}
                                    onChange={() => setConflictPolicy("skip")}
                                    className="accent-primary mt-0.5"
                                  />
                                  <div>
                                    <p className="text-xs">
                                      <span className="font-medium">
                                        {t("skipLabel")}
                                      </span>
                                      <span className="text-muted-foreground">
                                        {t("skipDesc")}
                                      </span>
                                    </p>
                                    <p className="text-xs text-muted-foreground">
                                      {t("skipDetail")}
                                    </p>
                                  </div>
                                </label>
                              </div>
                            )}
                          </div>
                        )}

                        {imageStatus && (
                          <p className="text-xs text-muted-foreground text-center">
                            {imageStatus}
                          </p>
                        )}
                      </>
                    )}
                  </div>
                )}

                {wizardStep === 3 && (
                  <PreflightSummaryContent
                    stats={preflightStats}
                    totalSizeBytes={preflightTotals.bytes}
                    totalFileCount={preflightTotals.files}
                    onBackgroundToggle={handleBackgroundToggle}
                    backgroundMode={backgroundMode}
                  />
                )}

                {(phase === "parsing" || phase === "uploading") && (
                  <ImportProgressPanel
                    projectId={projectId}
                    sessionId={currentSessionId}
                    localPhase={phase}
                    uploadProgress={uploadProgress}
                    imageProgress={
                      phase === "uploading"
                        ? { ...imageProgressRef.current }
                        : null
                    }
                    fallbackStatus={scanStatus || imageStatus || undefined}
                    locale={progressLocale}
                  />
                )}

                <ImportSuccessCard
                  phase={phase}
                  reImporting={reImporting}
                  result={result}
                  t={t}
                  tag={tag}
                  countryTag={countryTag}
                  router={router}
                  projectId={projectId}
                  setOpen={setOpen}
                  imageStatus={imageStatus}
                  whatsNextDetailOpen={whatsNextDetailOpen}
                  setWhatsNextDetailOpen={setWhatsNextDetailOpen}
                  imageExtSwaps={imageExtSwaps}
                  projectCountries={projectCountries}
                  handleAddCountryAndReImport={handleAddCountryAndReImport}
                  scanStatus={scanStatus}
                />

                {error && <p className="text-xs text-red-500">{error}</p>}
              </div>

              {wizardStep === 2 && (
                <SilentSeedBanner
                  phase={phase}
                  scan={scan}
                  silentSeedResult={silentSeedResult}
                  tImport={tImport}
                  advancedCountrySeed={advancedCountrySeed}
                  setAdvancedCountrySeed={setAdvancedCountrySeed}
                />
              )}
            </div>

            {(wizardStep === 2 || wizardStep === 3 || importInFlight) && (
              <DialogFooter className="shrink-0 mx-0 mb-0 px-4 py-3">
                {wizardStep === 2 && totalFound > 0 && (
                  <Button onClick={handlePreflightOpen}>
                    {t("importSelected")}
                  </Button>
                )}
                {wizardStep === 2 && (
                  <Button variant="outline" onClick={handlePickFolder}>
                    {t("changeFolder")}
                  </Button>
                )}

                {wizardStep === 3 && (
                  <>
                    <Button
                      variant="outline"
                      onClick={handlePreflightCancel}
                      data-testid="preflight-cancel"
                    >
                      {t("backStep")}
                    </Button>
                    <div className="flex flex-col items-end gap-1">
                      <Button
                        onClick={handlePreflightContinue}
                        disabled={selectedToggleFileCount === 0}
                        data-testid="preflight-continue"
                      >
                        {tPreflight("ctaContinue")}
                      </Button>
                      {selectedToggleFileCount === 0 && (
                        <p className="text-xs text-muted-foreground">
                          {t("confirmNothingSelected")}
                        </p>
                      )}
                    </div>
                  </>
                )}
                {importInFlight && (
                  <>
                    <Button
                      variant="outline"
                      onClick={() => setShowCancelConfirm(true)}
                      className="border-destructive/40 text-destructive hover:bg-destructive/10"
                    >
                      <XIcon className="h-3.5 w-3.5 mr-1" />
                      {tBg("cancelImport")}
                    </Button>
                    <Button disabled>
                      {phase === "parsing"
                        ? t("parsing")
                        : t("uploadingImages")}
                    </Button>
                  </>
                )}
              </DialogFooter>
            )}
          </DialogContent>
        </Dialog>

        <ExpandConfirmDialog
          pendingAddTag={pendingAddTag}
          setPendingAddTag={setPendingAddTag}
          addAcknowledged={addAcknowledged}
          setAddAcknowledged={setAddAcknowledged}
          t={t}
          performAddCountryAndReImport={performAddCountryAndReImport}
        />
        <ImportAuditDialog
          projectId={projectId}
          open={showAuditDialog}
          files={selectCountryAuditFiles(scan).map((f) => f.file)}
          paths={selectCountryAuditFiles(scan).map((f) => f.path)}
          onConfirmed={() => {
            setShowAuditDialog(false);
            void doImport();
          }}
          onCancel={() => {
            setShowAuditDialog(false);

            setShowPreflight(true);
          }}
        />

        <ImportInterruptDialogs
          showCloseConfirm={showCloseConfirm}
          setShowCloseConfirm={setShowCloseConfirm}
          handleConfirmCancel={handleConfirmCancel}
          handleKeepInBackground={handleKeepInBackground}
          showCancelConfirm={showCancelConfirm}
          setShowCancelConfirm={setShowCancelConfirm}
          tBg={tBg}
        />
      </>
    </TooltipProvider>
  );
}
