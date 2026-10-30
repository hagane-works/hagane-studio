"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { useParams } from "@/platform/navigation";
import { useRouter } from "@/i18n/navigation";
import { toast } from "sonner";

import { exportApi, type BaseInstall, type CtdIssue } from "@/lib/api";
import { validationIssuesFromRequestError } from "@/lib/exportValidationRun";
import { presentCheckIssues } from "@/lib/exportCheckPresentation";
import { useVanillaStateIndex } from "@/lib/vanillaStateIndex";
import { nativeWarningIssues } from "@/lib/exportNativeWarnings";
import {
  checkStaleReason,
  settleExportCheckWithinTimeout,
  useExportCheckStore,
} from "@/lib/store/exportCheckStore";
import { countGroupsWithEntryPoint, repairHash } from "@/lib/exportRepairQueue";
import {
  ExportStepper,
  type CheckPhase,
} from "@/components/export/ExportStepper";
import { ExportStatusCard } from "@/components/export/ExportStatusCard";
import { AttentionCard } from "@/components/export/AttentionCard";
import {
  ExportContentsCard,
  summaryHasContent,
} from "@/components/export/ExportContentsCard";
import { ExportEntryRow } from "@/components/export/ExportEntryRow";
import { CheckDetailsDialog } from "@/components/export/CheckDetailsDialog";
import { InGameVerifyDialog } from "@/components/export/InGameVerifyDialog";
import { InstallModPanel } from "@/desktop/InstallModPanel";
import { inProgramWindow } from "@/desktop/programWindow";
import {
  ExportSettingsDialog,
  getSelectedReplacePaths,
} from "@/components/export/ExportSettingsDialog";
import { AppendToExistingModDialog } from "@/components/workbench/AppendToExistingModDialog";
import { DiffMergeExportDialog } from "@/components/workbench/DiffMergeExportDialog";
import { useProjectStore } from "@/lib/store/projectStore";

const NO_ISSUES: CtdIssue[] = [];
import { useCharacterStore } from "@/lib/store/characterStore";
import { useFocusTreeStore } from "@/lib/store/focusTreeStore";
import { useTechnologyStore } from "@/lib/store/technologyStore";
import { useIdeaStore } from "@/lib/store/ideaStore";
import { useEventStore } from "@/lib/store/eventStore";
import { useDecisionStore } from "@/lib/store/decisionStore";
import { useStateStore } from "@/lib/store/stateStore";
import { Button } from "@/components/ui/button";
import { ProjectSettingsDialog } from "@/components/workbench/ProjectSettingsDialog";
import { SELECTABLE_DLCS } from "@/lib/dlcCatalog";

export default function ExportPage() {
  const t = useTranslations("export");
  const params = useParams();
  const router = useRouter();
  const projectId = Number(params.id);
  const currentProject = useProjectStore((s) => s.currentProject);
  const projectName = currentProject?.name ?? "mod";
  const isLoadedExistingMod = currentProject?.is_loaded_existing_mod ?? false;

  const hasDependencies = (currentProject?.dependencies?.length ?? 0) > 0;
  const hasTechnologySurface =
    (currentProject?.tech_surface_manifest?.branches?.length ?? 0) > 0;
  const technologyCompatibilityMode =
    currentProject?.tech_surface_manifest?.compatibility_mode ??
    "legacy_profile";

  type ExportIdentity =
    "submod" | "loaded_existing" | "standalone" | "ambiguous";
  const identity: ExportIdentity =
    hasDependencies && isLoadedExistingMod
      ? "ambiguous"
      : hasDependencies
        ? "submod"
        : isLoadedExistingMod
          ? "loaded_existing"
          : "standalone";

  const includeImportedForIdentity =
    identity === "loaded_existing" || identity === "ambiguous";

  const [sourceModeFull, setSourceModeFull] = useState<boolean | null>(null);
  const includeImported = sourceModeFull ?? includeImportedForIdentity;
  const identityDescKey =
    identity === "submod"
      ? "identitySubmodDesc"
      : identity === "loaded_existing"
        ? "identityLoadedDesc"
        : identity === "ambiguous"
          ? "identityAmbiguousDesc"
          : "identityStandaloneDesc";
  const showCoverageNote = identity === "loaded_existing";

  const checkRun = useExportCheckStore(
    (s) => s.runsByProject[projectId] ?? null,
  );
  const checkRunning = useExportCheckStore((s) =>
    Boolean(s.activeByProject[projectId]),
  );

  const [downloading, setDownloading] = useState(false);
  const [buildError, setBuildError] = useState<string | null>(null);
  const [files, setFiles] = useState<string[]>([]);
  const [builtSizeBytes, setBuiltSizeBytes] = useState<number | null>(null);

  const [nativeWarnings, setNativeWarnings] = useState<string[]>([]);

  const [exportedFilename, setExportedFilename] = useState<string | null>(null);

  const [buildIdentity, setBuildIdentity] = useState(false);
  const [replacePaths, setReplacePaths] = useState<Record<string, boolean>>({});
  const selectedReplacePaths = useMemo(
    () => getSelectedReplacePaths(replacePaths),
    [replacePaths],
  );

  const [detailsOpen, setDetailsOpen] = useState(false);
  const [verifyOpen, setVerifyOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);

  const [projectSettingsOpen, setProjectSettingsOpen] = useState(false);
  const ownedDlcs = currentProject?.owned_dlcs ?? null;
  const [appendDialogOpen, setAppendDialogOpen] = useState(false);
  const [diffMergeOpen, setDiffMergeOpen] = useState(false);

  const [baseInstalls, setBaseInstalls] = useState<BaseInstall[]>([]);

  const states = useStateStore((s) => s.states);
  useEffect(() => {
    if (projectId) void useStateStore.getState().fetchStates(projectId);
  }, [projectId]);

  const refreshAllStores = () => {
    if (!projectId) return;
    useCharacterStore.getState().fetchCharacters(projectId);
    useFocusTreeStore.getState().fetchTrees(projectId);
    useTechnologyStore.getState().fetchTechnologies(projectId);
    useIdeaStore.getState().fetchIdeas(projectId);
    useEventStore.getState().fetchNamespaces(projectId);
    useDecisionStore.getState().fetchCategories(projectId);
  };

  const replacePathsKey = selectedReplacePaths.join(",");
  const checkKey = `${projectId}|${includeImported}|${replacePathsKey}`;

  const runCheck = useCallback(() => {
    if (!projectId) return;
    const store = useExportCheckStore.getState();
    const request = store.startRun(projectId, checkKey);
    if (!request) return;
    const rps = replacePathsKey ? replacePathsKey.split(",") : [];
    const failedRun = () => ({
      projectId,
      key: checkKey,

      issues: [],
      checkFailed: true,
      summary: null,
      assetAudit: null,
      summaryFailed: true,
      ranAt: Date.now(),
    });
    void settleExportCheckWithinTimeout(
      Promise.allSettled([
        exportApi.check(projectId, includeImported, rps),
        exportApi.validate(projectId, includeImported, rps),
      ]),
    ).then((outcome) => {
      const currentStore = useExportCheckStore.getState();
      if (outcome.kind !== "completed") {
        currentStore.finishRun(request, failedRun());
        return;
      }
      const [checkResult, validateResult] = outcome.value;

      currentStore.finishRun(request, {
        projectId,
        key: checkKey,

        issues:
          checkResult.status === "fulfilled"
            ? (checkResult.value.issues ?? [])
            : [],
        checkFailed: checkResult.status !== "fulfilled",
        summary:
          validateResult.status === "fulfilled"
            ? validateResult.value.summary
            : null,
        assetAudit:
          validateResult.status === "fulfilled"
            ? (validateResult.value.asset_audit ?? null)
            : null,
        summaryFailed: validateResult.status !== "fulfilled",
        ranAt: Date.now(),
      });
    });
  }, [projectId, includeImported, replacePathsKey, checkKey]);

  const currentRun =
    checkRun && checkRun.projectId === projectId ? checkRun : null;
  const staleReason = checkStaleReason(
    currentRun,
    checkKey,
    currentProject?.updated_at,
  );

  const checkedAtLabel = currentRun
    ? new Date(currentRun.ranAt).toLocaleTimeString(undefined, {
        hour: "2-digit",
        minute: "2-digit",
      })
    : null;
  const phase: CheckPhase = checkRunning
    ? "running"
    : !currentRun
      ? "idle"
      : currentRun.checkFailed
        ? "failed"
        : "done";
  const issues = currentRun?.issues ?? NO_ISSUES;
  const checkRequestFailed = currentRun?.checkFailed ?? false;
  const summary = currentRun?.summary ?? null;
  const assetAudit = currentRun?.assetAudit ?? null;
  const summaryUnavailable = currentRun?.summaryFailed ?? false;

  useEffect(() => {
    if (!projectId || !hasDependencies) return;
    let alive = true;
    exportApi
      .baseInstalls(projectId)
      .then((res) => {
        if (alive) setBaseInstalls(res.base_installs ?? []);
      })
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, [projectId, hasDependencies]);
  const effectiveBaseInstalls = hasDependencies ? baseInstalls : [];

  const nativeWarningFindings = useMemo(
    () => nativeWarningIssues(nativeWarnings, t("ui.nativeWarning.label")),
    [nativeWarnings, t],
  );

  const vanillaStates = useVanillaStateIndex();
  const presentation = useMemo(
    () =>
      presentCheckIssues(
        nativeWarningFindings.length > 0
          ? [...issues, ...nativeWarningFindings]
          : issues,
        {
          states,
          loadedExistingMod: isLoadedExistingMod,
          vanillaStates,
        },
      ),
    [issues, nativeWarningFindings, states, isLoadedExistingMod, vanillaStates],
  );
  const hasContent = summaryHasContent(summary);

  const attentionCount = presentation.attentionGroups.length;
  const hasBlockerAttention = presentation.attention.some(
    (f) => f.issue.severity === "blocker",
  );

  const openDownloadStatusWindow = () => {
    if (inProgramWindow()) return null;
    const downloadWindow = window.open("about:blank", "_blank");
    if (downloadWindow) {
      const doc = downloadWindow.document;
      doc.title = t("checkingExport");
      doc.body.style.cssText =
        "margin:0;display:grid;place-items:center;min-height:100vh;font:14px system-ui;background:#faf8f3;color:#5f4a2d";
      const status = doc.createElement("div");
      status.textContent = t("checkingExport");
      doc.body.replaceChildren(status);
    }
    return downloadWindow;
  };

  const doDownload = async (includeImported: boolean) => {
    const rps = selectedReplacePaths;
    setDownloading(true);
    setBuildError(null);
    const downloadWindow = openDownloadStatusWindow();
    let builtNoteCount = 0;
    try {
      const built = await exportApi.buildArtifact(
        projectId,
        includeImported,
        rps,
        true,
        buildIdentity,
      );
      const jobId = built.job_id;
      if (built.files && built.files.length > 0) setFiles(built.files);
      setBuiltSizeBytes(
        typeof built.size_bytes === "number" ? built.size_bytes : null,
      );

      const notes = Array.isArray(built.native_warnings)
        ? built.native_warnings.filter((w) => typeof w === "string" && w.trim())
        : [];
      setNativeWarnings(notes);
      builtNoteCount = notes.length;
      const downloadUrl = exportApi.buildJobDownloadUrl(projectId, jobId);
      if (downloadWindow) downloadWindow.location.replace(downloadUrl);
      else window.location.assign(downloadUrl);
    } catch (error) {
      downloadWindow?.close();

      const recovered = validationIssuesFromRequestError(
        error,
        t("validationRequestFailed"),
      );
      setBuildError(recovered[0]?.message ?? t("exportCouldNotComplete"));
      console.warn("Background export failed:", error);
      toast.error(t("exportCouldNotComplete"));
      setDownloading(false);
      return;
    }
    setDownloading(false);
    toast.success(
      t("exportTriggered") +
        (rps.length ? t("replacePathSuffix", { count: rps.length }) : "") +
        (builtNoteCount > 0
          ? t("ui.nativeWarning.toastSuffix", { count: builtNoteCount })
          : ""),
    );

    setExportedFilename(`${projectName}.zip`);
  };

  const handlePrimaryDownload = () => {
    if (downloading) return;
    void doDownload(includeImported);
  };

  const handleFix = () => {
    const first = presentation.attention[0];
    const hash = first ? `#${repairHash(first.issue)}` : "";
    router.push(`/project/${projectId}/export/repair${hash}`);
  };

  const checkEntrySubtitle =
    phase === "idle"
      ? t("ui.stepper.step1SubIdle")
      : phase === "running"
        ? t("ui.stepper.step1Running")
        : phase === "failed"
          ? t("ui.stepper.step1Failed")
          : presentation.health.length > 0
            ? t("ui.entries.checkSubHealth", {
                auto: presentation.auto.length,
                pass: presentation.pass.length,
                health: presentation.health.length,
              })
            : t("ui.entries.checkSub", {
                auto: presentation.auto.length,
                pass: presentation.pass.length,
              });

  return (
    <div className="h-full overflow-y-auto">
      <div className="mx-auto max-w-4xl space-y-4 p-6">
        <div>
          <h2 className="mb-1 text-lg font-semibold">{t("pageTitle")}</h2>
          <p className="text-sm text-muted-foreground">{t("ui.pageIntro")}</p>
        </div>

        <div
          data-testid="dlc-choice-export-summary"
          className="flex items-start justify-between gap-4 rounded-lg border border-border bg-muted/30 px-4 py-3"
        >
          <div className="min-w-0">
            <p className="text-sm font-medium text-foreground">
              {ownedDlcs == null || ownedDlcs.length === SELECTABLE_DLCS.length
                ? t("dlcChoiceTitleAll")
                : ownedDlcs.length === 0
                  ? t("dlcChoiceTitleNone")
                  : t("dlcChoiceTitleSome", { list: ownedDlcs.join(", ") })}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("dlcChoiceBody")}
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setProjectSettingsOpen(true)}
          >
            {t("dlcChoiceChange")}
          </Button>
        </div>

        {hasTechnologySurface ? (
          <div
            data-testid="technology-compatibility-export-summary"
            className="rounded-lg border border-border bg-muted/30 px-4 py-3"
          >
            <p className="text-sm font-medium text-foreground">
              {t(
                technologyCompatibilityMode === "all_players"
                  ? "technologyCompatibilityAllPlayersTitle"
                  : "technologyCompatibilityRequiresDLCTitle",
              )}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {t(
                technologyCompatibilityMode === "all_players"
                  ? "technologyCompatibilityAllPlayersBody"
                  : "technologyCompatibilityRequiresDLCBody",
              )}
            </p>
          </div>
        ) : null}

        <ExportStepper
          phase={phase}
          attentionCount={attentionCount}
          downloaded={exportedFilename !== null}
          onOpenVerify={() => setVerifyOpen(true)}
          onRerunCheck={runCheck}
        />

        <ExportStatusCard
          phase={phase}
          attentionCount={attentionCount}
          hasBlockerAttention={hasBlockerAttention}
          autoCount={presentation.auto.length}
          healthCount={presentation.health.length}
          checkRequestFailed={checkRequestFailed}
          downloading={downloading}
          buildErrorMessage={buildError}
          repairableCount={countGroupsWithEntryPoint(
            presentation.attentionGroups,
          )}
          staleReason={staleReason}
          checkedAtLabel={checkedAtLabel}
          onDownload={handlePrimaryDownload}
          onFix={handleFix}
        />

        <InstallModPanel
          projectId={projectId}
          includeImported={includeImported}
          replacePaths={selectedReplacePaths}
        />

        <div className="grid grid-cols-1 items-stretch gap-4 lg:grid-cols-[minmax(0,1.55fr)_minmax(280px,0.85fr)]">
          <AttentionCard
            phase={phase}
            projectId={projectId}
            attention={presentation.attention}
            groups={presentation.attentionGroups}
            autoCount={presentation.auto.length}
            passCount={presentation.pass.length}
            healthCount={presentation.health.length}
            onOpenDetails={() => setDetailsOpen(true)}
          />
          <ExportContentsCard
            pending={
              currentRun ? null : phase === "running" ? "running" : "idle"
            }
            summary={summary}
            summaryUnavailable={summaryUnavailable}
            assetAudit={assetAudit}
            builtFileCount={files.length > 0 ? files.length : null}
            builtSizeBytes={builtSizeBytes}
            identityDescKey={identityDescKey}
            legacyMapNote={
              presentation.legacyMap
                ? t("ui.dialogCheck.legacyMapAggregate", {
                    count: presentation.legacyMap.count,
                  })
                : null
            }
          />
        </div>

        <ExportEntryRow
          checkSubtitle={checkEntrySubtitle}
          verifyUnlocked={exportedFilename !== null}
          onOpenDetails={() => setDetailsOpen(true)}
          onOpenVerify={() => setVerifyOpen(true)}
          onOpenSettings={() => setSettingsOpen(true)}
        />

        {!hasContent && !summaryUnavailable && phase === "done" ? (
          <p className="text-xs text-muted-foreground">{t("noContentHint")}</p>
        ) : null}
      </div>

      <CheckDetailsDialog
        open={detailsOpen}
        onOpenChange={setDetailsOpen}
        auto={presentation.auto}
        health={presentation.health}
        pass={presentation.pass}
        legacyMap={presentation.legacyMap}
      />

      <InGameVerifyDialog
        open={verifyOpen}
        onOpenChange={setVerifyOpen}
        projectId={projectId}
        exportedFilename={exportedFilename}
        hasDependencies={hasDependencies}
        baseInstalls={effectiveBaseInstalls}
        dependencies={currentProject?.dependencies ?? []}
      />

      {projectSettingsOpen && (
        <ProjectSettingsDialog
          projectId={projectId}
          open={projectSettingsOpen}
          onOpenChange={setProjectSettingsOpen}
        />
      )}

      <ExportSettingsDialog
        open={settingsOpen}
        onOpenChange={setSettingsOpen}
        projectId={projectId}
        projectName={projectName}
        files={files}
        buildIdentity={buildIdentity}
        onBuildIdentityChange={(value) => {
          setBuildIdentity(value);
        }}
        hasTechnologySurface={hasTechnologySurface}
        replacePaths={replacePaths}
        onToggleReplacePath={(key, checked) => {
          setReplacePaths((prev) => ({ ...prev, [key]: checked }));

          setFiles([]);
          setBuiltSizeBytes(null);
        }}
        showCoverageNote={showCoverageNote}
        showSourceModeChoice={identity === "ambiguous"}
        includeImported={includeImported}
        onIncludeImportedChange={(value) => {
          setSourceModeFull(value);

          setFiles([]);
          setBuiltSizeBytes(null);
        }}
        onOpenAppend={() => {
          setSettingsOpen(false);
          setAppendDialogOpen(true);
        }}
        onOpenDiffMerge={() => {
          setSettingsOpen(false);
          setDiffMergeOpen(true);
        }}
        onBatchFlipSuccess={refreshAllStores}
      />

      <AppendToExistingModDialog
        open={appendDialogOpen}
        onOpenChange={setAppendDialogOpen}
        projectId={projectId}
        projectName={projectName}
        replacePaths={selectedReplacePaths}
        includeImported={false}
        buildIdentity={buildIdentity}
        hasTechnologySurface={hasTechnologySurface}
        confirmDiagnosticRisk={() => Promise.resolve(true)}
        onExported={(filename) => setExportedFilename(filename)}
      />
      <DiffMergeExportDialog
        open={diffMergeOpen}
        onOpenChange={setDiffMergeOpen}
        projectId={projectId}
        projectName={projectName}
        hasTechnologySurface={hasTechnologySurface}
        confirmDiagnosticRisk={() => Promise.resolve(true)}
        onExported={(filename) => setExportedFilename(filename)}
      />
    </div>
  );
}
