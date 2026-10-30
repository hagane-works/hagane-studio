"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import {
  Anchor,
  CheckCircle2,
  Factory,
  Plane,
  Plus,
  Shield,
  Trash2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { technologiesApi } from "@/lib/api";
import {
  techCertifiedSourceMessageKey,
  techCertifiedSourceMessageParams,
} from "@/lib/techSource";
import type { Technology } from "@/types";
import {
  TECH_SURFACE_ENTRANCE_IDS,
  buildTechSurfaceBranchCreateInput,
  buildTechSurfaceEnvironmentConfirmInput,
  canConfirmTechSurfaceEnvironment,
  isTechSurfaceTargetSelectable,
  techSurfaceBranches,
  techSurfaceTargetForBranch,
  type TechSurfaceBranch,
  type TechSurfaceCompatibilityMode,
  type TechSurfaceDLCState,
  type TechSurfaceEntranceID,
  type TechSurfaceProfileEntry,
  type TechSurfaceProfileTarget,
} from "@/lib/techSurface";

const EMPTY_DLC_STATE: TechSurfaceDLCState = {
  no_step_back: false,
  by_blood_alone: false,
  man_the_guns: false,
};

const ENTRANCE_LABEL_KEYS: Record<TechSurfaceEntranceID, string> = {
  army: "entranceArmy",
  air: "entranceAir",
  navy: "entranceNavy",
  industry_equipment: "entranceIndustrialEquipment",
};

const TARGET_LABEL_KEYS: Record<string, string> = {
  infantry: "targetInfantry",
  armour: "targetArmour",
  airframes: "targetAirframes",
  naval_main: "targetNavalMain",
  naval_support: "targetNavalSupport",
  industry: "targetIndustry",
  electronics: "targetElectronics",
};

function technologyLabel(technology: Technology | undefined, id: string) {
  return technology?.name?.trim() || technology?.tech_id || id;
}

function generatedTechnologyID(name: string) {
  const readable = name
    .normalize("NFKD")
    .replace(/[^A-Za-z0-9_]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .toLowerCase();
  return `MOD_${readable || `technology_${Date.now().toString(36)}`}`;
}

interface TechSurfacePanelProps {
  projectId: number;
  technologies: Technology[];
  onCreated: (technology: Technology) => Promise<void>;
  onRepaired: () => Promise<void>;
}

export function TechSurfacePanel({
  projectId,
  technologies,
  onCreated,
  onRepaired,
}: TechSurfacePanelProps) {
  const t = useTranslations("techSurface");
  const confirmDialog = useConfirm();
  const {
    data: status,
    error,
    mutate,
  } = useSWR(projectId ? ["tech/surface", projectId] : null, () =>
    technologiesApi.surfaceStatus(projectId),
  );
  const [dlcDraft, setDlcDraft] = useState<TechSurfaceDLCState | null>(null);
  const [compatibilityDraft, setCompatibilityDraft] =
    useState<TechSurfaceCompatibilityMode | null>(null);
  const [selectedEntryID, setSelectedEntryID] =
    useState<TechSurfaceEntranceID>("army");
  const [selectedTargetID, setSelectedTargetID] = useState("");
  const [compatibilityEditing, setCompatibilityEditing] = useState(false);
  const [branchOpen, setBranchOpen] = useState(false);
  const [childBranch, setChildBranch] = useState<TechSurfaceBranch | null>(
    null,
  );
  const [branchID, setBranchID] = useState("");
  const [rootTechnologyID, setRootTechnologyID] = useState("");
  const [rootName, setRootName] = useState("");
  const [rootDescription, setRootDescription] = useState("");
  const [parentTechnologyID, setParentTechnologyID] = useState("");
  const [childTechnologyID, setChildTechnologyID] = useState("");
  const [childName, setChildName] = useState("");
  const [childDescription, setChildDescription] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [confirmingEnvironment, setConfirmingEnvironment] = useState(false);
  const [repairing, setRepairing] = useState(false);
  const [deletingBranchID, setDeletingBranchID] = useState("");
  const [environmentActionError, setEnvironmentActionError] = useState("");

  const activeProfile = status?.active_profile;
  const dlcSelection = dlcDraft ?? status?.environment.dlc ?? EMPTY_DLC_STATE;
  const compatibilityMode =
    compatibilityDraft ??
    status?.environment.compatibility_mode ??
    "all_players";
  const hasSelectedDLC = Object.values(dlcSelection).some(Boolean);
  const entriesByID = useMemo(
    () =>
      new Map(activeProfile?.entries.map((entry) => [entry.id, entry]) ?? []),
    [activeProfile],
  );
  const selectedEntry = selectedEntryID
    ? entriesByID.get(selectedEntryID)
    : undefined;
  const selectedTarget = selectedEntry?.targets.find(
    (target) => target.id === selectedTargetID,
  );
  const technologiesByID = useMemo(
    () =>
      new Map(
        technologies.map((technology) => [technology.tech_id, technology]),
      ),
    [technologies],
  );
  const usedTargets = useMemo(
    () =>
      new Set(
        techSurfaceBranches(status).map(
          (branch) => `${branch.surface}:${branch.folder}`,
        ),
      ),
    [status],
  );

  const entranceLabel = (entryID: TechSurfaceEntranceID) =>
    t(ENTRANCE_LABEL_KEYS[entryID]);
  const entranceIcon = (entryID: TechSurfaceEntranceID) => {
    if (entryID === "air") return Plane;
    if (entryID === "navy") return Anchor;
    if (entryID === "industry_equipment") return Factory;
    return Shield;
  };
  const targetLabel = (target: TechSurfaceProfileTarget) => {
    const key = TARGET_LABEL_KEYS[target.id];
    return key ? t(key) : target.id;
  };

  const updateDLC = (key: keyof TechSurfaceDLCState, checked: boolean) => {
    setDlcDraft({ ...dlcSelection, [key]: checked });
  };

  const handleConfirmEnvironment = async () => {
    if (!status) return;
    setConfirmingEnvironment(true);
    setEnvironmentActionError("");
    try {
      let currentStatus = status;
      if (currentStatus.environment.source_connection.kind === "none") {
        await technologiesApi.attachCertifiedSurfaceSource(projectId);
        currentStatus = await technologiesApi.surfaceStatus(projectId);
        await mutate(currentStatus, false);
      }
      if (!canConfirmTechSurfaceEnvironment(currentStatus)) {
        throw new Error(t("sourceLayoutUnavailable"));
      }
      const nextStatus = await technologiesApi.confirmSurfaceEnvironment(
        projectId,
        buildTechSurfaceEnvironmentConfirmInput(
          compatibilityMode === "all_players" ? EMPTY_DLC_STATE : dlcSelection,
          compatibilityMode,
        ),
      );
      await mutate(nextStatus, false);
      setDlcDraft(null);
      setCompatibilityDraft(null);
      setCompatibilityEditing(false);
      setSelectedEntryID("army");
      setSelectedTargetID("");
      toast.success(t("environmentConfirmed"));
    } catch (cause) {
      const certifiedKey = techCertifiedSourceMessageKey(cause);
      const message = certifiedKey
        ? t(certifiedKey, techCertifiedSourceMessageParams(cause))
        : cause instanceof Error
          ? cause.message
          : t("environmentConfirmFailed");
      setEnvironmentActionError(message);
      toast.error(message);
    } finally {
      setConfirmingEnvironment(false);
    }
  };

  const handleCancelCompatibility = () => {
    setDlcDraft(null);
    setCompatibilityDraft(null);
    setCompatibilityEditing(false);
    setEnvironmentActionError("");
    setSelectedEntryID("army");
    setSelectedTargetID("");
  };

  const chooseTarget = (
    entry: TechSurfaceProfileEntry,
    target: TechSurfaceProfileTarget,
  ) => {
    const alreadyUsed = usedTargets.has(`${entry.id}:${target.folder_id}`);
    if (
      !isTechSurfaceTargetSelectable(target, compatibilityMode) ||
      alreadyUsed
    )
      return;
    setSelectedEntryID(entry.id);
    setSelectedTargetID(target.id);
  };

  const handleBranchOpen = () => {
    if (!selectedEntryID || !selectedTarget) return;
    setBranchID("");
    setRootTechnologyID("");
    setRootName("");
    setRootDescription("");
    setBranchOpen(true);
  };

  const handleCreateBranch = async () => {
    if (
      !selectedEntryID ||
      !selectedTarget ||
      !rootName.trim() ||
      !rootDescription.trim()
    )
      return;
    setSubmitting(true);
    try {
      const technologyID =
        rootTechnologyID.trim() || generatedTechnologyID(rootName);
      const stableBranchID =
        branchID.trim() || `${technologyID.toLowerCase()}_branch`;
      const result = await technologiesApi.createSurfaceBranch(
        projectId,
        buildTechSurfaceBranchCreateInput({
          id: stableBranchID,
          entryID: selectedEntryID,
          targetID: selectedTarget.id,
          root: {
            tech_id: technologyID,
            name: rootName.trim(),
            description_locs: { en: rootDescription.trim() },
          },
        }),
      );
      await mutate();
      await onCreated(result.root);
      setBranchOpen(false);
      setSelectedTargetID("");
      toast.success(t("branchCreated"));
    } catch (cause) {
      toast.error(cause instanceof Error ? cause.message : t("saveFailed"));
    } finally {
      setSubmitting(false);
    }
  };

  const handleChildOpen = (branch: TechSurfaceBranch) => {
    const target = techSurfaceTargetForBranch(activeProfile, branch);
    if (!target?.timeline_years[branch.technology_ids.length]) {
      toast.error(t("timelineComplete"));
      return;
    }
    setChildBranch(branch);
    setParentTechnologyID(
      branch.technology_ids.at(-1) || branch.root_technology_id,
    );
    setChildTechnologyID("");
    setChildName("");
    setChildDescription("");
  };

  const handleRepairTimeline = async () => {
    setRepairing(true);
    try {
      const result = await technologiesApi.repairSurfaceTimeline(projectId);
      await mutate();
      await onRepaired();
      toast.success(
        t("legacyRepairComplete", {
          count: result.repaired_technology_ids.length,
        }),
      );
    } catch (cause) {
      toast.error(cause instanceof Error ? cause.message : t("saveFailed"));
    } finally {
      setRepairing(false);
    }
  };

  const handleCreateChild = async () => {
    if (
      !childBranch ||
      !parentTechnologyID ||
      !childName.trim() ||
      !childDescription.trim()
    )
      return;
    setSubmitting(true);
    try {
      const technologyID =
        childTechnologyID.trim() || generatedTechnologyID(childName);
      const result = await technologiesApi.createSurfaceBranchTechnology(
        projectId,
        childBranch.id,
        {
          parent_technology_id: parentTechnologyID,
          technology: {
            tech_id: technologyID,
            name: childName.trim(),
            description_locs: { en: childDescription.trim() },
          },
        },
      );
      await mutate();
      await onCreated(result.technology);
      setChildBranch(null);
      toast.success(t("technologyCreated"));
    } catch (cause) {
      toast.error(cause instanceof Error ? cause.message : t("saveFailed"));
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteBranch = async (branch: TechSurfaceBranch) => {
    const rootLabel = technologyLabel(
      technologiesByID.get(branch.root_technology_id),
      branch.root_technology_id,
    );
    const confirmed = await confirmDialog({
      title: t("deleteBranchTitle"),
      body: t("deleteBranchBody", {
        name: rootLabel,
        count: branch.technology_ids.length,
      }),
      danger: true,
    });
    if (!confirmed) return;
    setDeletingBranchID(branch.id);
    try {
      await technologiesApi.deleteSurfaceBranch(projectId, branch.id);
      await mutate();
      await onRepaired();
      toast.success(t("branchDeleted"));
    } catch (cause) {
      toast.error(cause instanceof Error ? cause.message : t("saveFailed"));
    } finally {
      setDeletingBranchID("");
    }
  };

  if (error) {
    return (
      <p className="px-3 py-2 text-xs text-muted-foreground">
        {t("unavailable")}
      </p>
    );
  }
  if (!status) {
    return (
      <p className="px-3 py-2 text-xs text-muted-foreground">{t("loading")}</p>
    );
  }

  const environmentReady =
    status.environment.state === "ready" && Boolean(activeProfile);
  const branches = techSurfaceBranches(status);
  const showCompatibilityForm = !environmentReady || compatibilityEditing;
  const selectedRootYear = selectedTarget?.timeline_years[0];
  const childTarget = childBranch
    ? techSurfaceTargetForBranch(activeProfile, childBranch)
    : undefined;
  const childYear = childBranch
    ? childTarget?.timeline_years[childBranch.technology_ids.length]
    : undefined;

  let environmentTitle = t("environmentBlockedTitle");
  let environmentBody = t("environmentBlockedBody");
  let environmentTone = "border-destructive/40 bg-destructive/10";
  if (status.environment.state === "baseline_missing") {
    environmentTitle = t("baselineMissingTitle");
    environmentBody = t("baselineMissingBody");
  } else if (status.environment.state === "tech_profile_source_missing") {
    environmentTitle = t("sourceMissingTitle");
    environmentBody = t("sourceMissingBody");
  } else if (status.environment.state === "tech_profile_source_drift") {
    environmentTitle = t("sourceDriftTitle");
    environmentBody = t("sourceDriftBody");
  } else if (status.environment.state === "dlc_unconfirmed") {
    environmentTitle = t("environmentUnconfirmedTitle");
    environmentBody = t("environmentUnconfirmedBody");
    environmentTone = "border-border bg-muted/30";
  }

  return (
    <section className="space-y-3 border-b border-border/40 bg-muted/25 px-3 py-3">
      <div>
        <h2 className="text-sm font-medium">{t("title")}</h2>
        <p className="mt-1 text-xs text-muted-foreground">{t("description")}</p>
      </div>

      {showCompatibilityForm ? (
        <div
          data-testid="tech-surface-environment"
          className="rounded-md border border-border bg-card px-3 py-3"
        >
          <h3 className="text-sm font-medium">{t("environmentTitle")}</h3>
          <p className="mt-1 text-xs text-muted-foreground">
            {t("environmentDescription")}
          </p>
          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            {(["all_players", "requires_dlc"] as const).map((mode) => (
              <label
                key={mode}
                className={`ui-interactive cursor-pointer rounded-md border px-3 py-2 ${compatibilityMode === mode ? "border-accent bg-accent/10" : "border-border bg-background/50"}`}
              >
                <span className="flex items-center gap-2">
                  <input
                    className="ui-focus-ring h-4 w-4 accent-primary"
                    type="radio"
                    name="tech-surface-compatibility"
                    checked={compatibilityMode === mode}
                    disabled={confirmingEnvironment}
                    onChange={() => {
                      setCompatibilityDraft(mode);
                      if (mode === "all_players") setDlcDraft(EMPTY_DLC_STATE);
                      setSelectedEntryID("army");
                      setSelectedTargetID("");
                    }}
                  />
                  <span className="text-xs font-medium text-foreground">
                    {t(
                      mode === "all_players"
                        ? "compatibilityAllPlayers"
                        : "compatibilityRequiresDLC",
                    )}
                  </span>
                </span>
                <span className="mt-1 block text-micro text-muted-foreground">
                  {t(
                    mode === "all_players"
                      ? "compatibilityAllPlayersBody"
                      : "compatibilityRequiresDLCBody",
                  )}
                </span>
              </label>
            ))}
          </div>
          {compatibilityMode === "requires_dlc" && (
            <div className="mt-3">
              <p className="text-xs font-medium text-foreground">
                {t("requiredDLCsTitle")}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("requiredDLCsBody")}
              </p>
              <div className="mt-2 grid gap-2 sm:grid-cols-3">
                {(
                  [
                    ["no_step_back", "dlcNoStepBack"],
                    ["by_blood_alone", "dlcByBloodAlone"],
                    ["man_the_guns", "dlcManTheGuns"],
                  ] as const
                ).map(([key, labelKey]) => (
                  <label
                    key={key}
                    className="ui-interactive flex cursor-pointer items-center gap-2 rounded-md border border-border bg-background/50 px-2 py-2 text-xs has-disabled:cursor-not-allowed has-disabled:opacity-50"
                  >
                    <input
                      className="ui-focus-ring h-4 w-4 accent-primary"
                      type="checkbox"
                      checked={dlcSelection[key]}
                      disabled={confirmingEnvironment}
                      onChange={(event) => updateDLC(key, event.target.checked)}
                    />
                    <span>{t(labelKey)}</span>
                  </label>
                ))}
              </div>
              {!hasSelectedDLC && (
                <p className="mt-2 text-xs text-destructive">
                  {t("requiredDLCMissing")}
                </p>
              )}
            </div>
          )}
          <div className="mt-3 flex gap-2">
            {environmentReady && (
              <Button
                className="flex-1"
                size="sm"
                variant="outline"
                disabled={confirmingEnvironment}
                onClick={handleCancelCompatibility}
              >
                {t("cancel")}
              </Button>
            )}
            <Button
              className="flex-1"
              size="sm"
              disabled={
                confirmingEnvironment ||
                (compatibilityMode === "requires_dlc" && !hasSelectedDLC)
              }
              onClick={handleConfirmEnvironment}
            >
              {confirmingEnvironment
                ? t("savingCompatibility")
                : t("saveCompatibility")}
            </Button>
          </div>
          {environmentActionError && (
            <p
              role="alert"
              className="mt-2 rounded-md border border-destructive/40 bg-destructive/10 px-2 py-2 text-xs text-destructive"
            >
              {environmentActionError}
            </p>
          )}
          {!environmentReady &&
            status.environment.state !== "baseline_missing" &&
            status.environment.state !== "dlc_unconfirmed" && (
              <div
                className={`mt-3 rounded-md border px-2 py-2 ${environmentTone}`}
              >
                <p className="text-xs font-medium text-foreground">
                  {environmentTitle}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {environmentBody}
                </p>
              </div>
            )}
        </div>
      ) : (
        <div
          data-testid="tech-surface-environment-summary"
          className="flex items-center gap-2 rounded-md border border-success/40 bg-success/10 px-3 py-2"
        >
          <CheckCircle2 className="h-4 w-4 shrink-0 text-success" aria-hidden />
          <div className="min-w-0 flex-1">
            <p className="text-xs font-medium text-foreground">
              {t(
                compatibilityMode === "all_players"
                  ? "compatibilityReadyAllPlayers"
                  : "compatibilityReadyRequiresDLC",
              )}
            </p>
            <p className="mt-0.5 text-micro text-muted-foreground">
              {t("latestVersionReady")}
            </p>
          </div>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setCompatibilityEditing(true)}
          >
            {t("change")}
          </Button>
        </div>
      )}

      {environmentReady && activeProfile && (
        <div className="space-y-2">
          <div>
            <h3 className="text-sm font-medium">{t("entrancesTitle")}</h3>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("chooseTargetDescription")}
            </p>
          </div>
          <div className="grid grid-cols-4 gap-1 rounded-md border border-border bg-card p-1">
            {TECH_SURFACE_ENTRANCE_IDS.map((entryID) => {
              const Icon = entranceIcon(entryID);
              const active = selectedEntryID === entryID;
              return (
                <button
                  key={entryID}
                  type="button"
                  className={`ui-interactive ui-focus-ring flex min-w-0 flex-col items-center gap-1 rounded px-1 py-2 text-micro ${active ? "bg-accent text-accent-foreground" : "text-muted-foreground hover:bg-muted"}`}
                  aria-pressed={active}
                  onClick={() => {
                    setSelectedEntryID(entryID);
                    setSelectedTargetID("");
                  }}
                >
                  <Icon className="h-4 w-4" aria-hidden />
                  <span className="truncate">{entranceLabel(entryID)}</span>
                </button>
              );
            })}
          </div>

          <div className="space-y-2">
            {(entriesByID.get(selectedEntryID)?.targets ?? []).map((target) => {
              const entry = entriesByID.get(selectedEntryID)!;
              const certified = isTechSurfaceTargetSelectable(
                target,
                compatibilityMode,
              );
              const alreadyUsed = usedTargets.has(
                `${entry.id}:${target.folder_id}`,
              );
              const selectable = certified && !alreadyUsed;
              const checked = selectedTargetID === target.id;
              return (
                <button
                  key={target.id}
                  type="button"
                  disabled={!selectable}
                  aria-pressed={checked}
                  onClick={() => chooseTarget(entry, target)}
                  className={`ui-interactive ui-focus-ring w-full rounded-md border px-3 py-3 text-left ${checked ? "border-accent bg-accent/10" : "border-border bg-card"} ${selectable ? "cursor-pointer" : "cursor-not-allowed opacity-55"}`}
                >
                  <span className="text-xs font-medium text-foreground">
                    {targetLabel(target)}
                  </span>
                  {target.timeline_years.length > 0 && (
                    <span
                      className="mt-2 flex items-center"
                      aria-label={t("targetYears", {
                        years: target.timeline_years.join(", "),
                      })}
                    >
                      {target.timeline_years.map((year, index) => (
                        <span
                          key={year}
                          className="flex min-w-0 flex-1 items-center last:flex-none"
                        >
                          <span
                            className={`flex h-8 min-w-12 items-center justify-center rounded border px-1 text-micro font-medium ${index === 0 ? "border-accent/60 bg-background text-foreground" : "border-border bg-muted/50 text-muted-foreground"}`}
                          >
                            {year}
                          </span>
                          {index < target.timeline_years.length - 1 && (
                            <span
                              className="h-px min-w-3 flex-1 bg-border"
                              aria-hidden
                            />
                          )}
                        </span>
                      ))}
                    </span>
                  )}
                  <span className="mt-2 block text-micro text-muted-foreground">
                    {selectable
                      ? t("targetFirstYear", {
                          year: target.timeline_years[0] ?? 0,
                        })
                      : alreadyUsed
                        ? t("targetAlreadyUsed")
                        : compatibilityMode === "all_players" &&
                            !target.all_players_supported
                          ? t("targetRequiresDLCPolicy")
                          : t("targetUncertified")}
                  </span>
                </button>
              );
            })}
            {(entriesByID.get(selectedEntryID)?.targets.length ?? 0) === 0 && (
              <p className="text-xs text-muted-foreground">
                {t("targetUnavailable")}
              </p>
            )}
          </div>
          <Button
            size="sm"
            className="w-full"
            onClick={handleBranchOpen}
            disabled={!selectedTarget}
          >
            <Plus className="mr-1 h-3.5 w-3.5" />
            {selectedRootYear
              ? t("createAtYear", { year: selectedRootYear })
              : t("createTechnology")}
          </Button>
        </div>
      )}

      {branches.length > 0 && (
        <div className="space-y-2">
          {status.legacy_timeline_repair_needed && (
            <div className="rounded-md border border-warning/40 bg-warning/10 px-2 py-2">
              <p className="text-xs text-foreground">
                {t("legacyRepairRequired")}
              </p>
              <Button
                size="sm"
                variant="outline"
                className="mt-2 w-full"
                onClick={handleRepairTimeline}
                disabled={repairing}
              >
                {t("repairLegacyTimeline")}
              </Button>
            </div>
          )}
          {branches.map((branch) => {
            const target = techSurfaceTargetForBranch(activeProfile, branch);
            const canAddNext = Boolean(
              target?.timeline_years[branch.technology_ids.length],
            );
            return (
              <div
                key={branch.id}
                className="rounded-md border border-border bg-card px-2 py-2"
              >
                <p className="truncate text-xs font-medium">
                  {technologyLabel(
                    technologiesByID.get(branch.root_technology_id),
                    branch.root_technology_id,
                  )}
                </p>
                <p className="mt-0.5 text-micro text-muted-foreground">
                  {t("branchSummary", {
                    entrance: ENTRANCE_LABEL_KEYS[branch.surface]
                      ? entranceLabel(branch.surface)
                      : branch.surface,
                    target: target ? targetLabel(target) : branch.folder,
                    count: branch.technology_ids.length,
                  })}
                </p>
                <div className="mt-2 flex gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    className="flex-1"
                    onClick={() => handleChildOpen(branch)}
                    disabled={!canAddNext}
                  >
                    <Plus className="mr-1 h-3.5 w-3.5" />
                    {canAddNext && target
                      ? t("addAtYear", {
                          year: target.timeline_years[
                            branch.technology_ids.length
                          ],
                        })
                      : t("timelineComplete")}
                  </Button>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="text-destructive hover:text-destructive"
                    title={t("deleteBranch")}
                    aria-label={t("deleteBranch")}
                    disabled={deletingBranchID === branch.id}
                    onClick={() => void handleDeleteBranch(branch)}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      <Dialog open={branchOpen} onOpenChange={setBranchOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("branchDialogTitle")}</DialogTitle>
            <DialogDescription>
              {t("branchDialogDescription", {
                entrance: selectedEntryID ? entranceLabel(selectedEntryID) : "",
                target: selectedTarget ? targetLabel(selectedTarget) : "",
              })}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="surface-root-name">{t("technologyName")}</Label>
              <Input
                id="surface-root-name"
                value={rootName}
                onChange={(event) => setRootName(event.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="surface-root-description">
                {t("technologyDescription")}
              </Label>
              <Textarea
                id="surface-root-description"
                value={rootDescription}
                onChange={(event) => setRootDescription(event.target.value)}
                required
              />
            </div>
            <div className="rounded-md border border-border bg-muted/30 px-3 py-2">
              <p className="text-xs font-medium text-foreground">
                {t("researchYear", { year: selectedRootYear ?? 0 })}
              </p>
              <p className="mt-1 text-micro text-muted-foreground">
                {t("yearComesFromPosition")}
              </p>
            </div>
            <p className="text-xs text-muted-foreground">
              {t("descriptionRequired")}
            </p>
            <details className="rounded-md border border-border px-3 py-2 text-xs text-muted-foreground">
              <summary className="cursor-pointer font-medium text-foreground">
                {t("technicalSettings")}
              </summary>
              <p className="mt-1">{t("technicalSettingsBody")}</p>
              <div className="mt-2 space-y-2">
                <div className="space-y-1">
                  <Label htmlFor="surface-root-id">{t("technologyId")}</Label>
                  <Input
                    id="surface-root-id"
                    value={rootTechnologyID}
                    onChange={(event) =>
                      setRootTechnologyID(event.target.value)
                    }
                    placeholder={t("generatedAutomatically")}
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="surface-branch-id">{t("branchId")}</Label>
                  <Input
                    id="surface-branch-id"
                    value={branchID}
                    onChange={(event) => setBranchID(event.target.value)}
                    placeholder={t("generatedAutomatically")}
                  />
                </div>
              </div>
            </details>
          </div>
          <DialogFooter>
            <Button
              onClick={handleCreateBranch}
              disabled={
                submitting || !rootName.trim() || !rootDescription.trim()
              }
            >
              {submitting ? t("creatingTechnology") : t("createTechnology")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={childBranch !== null}
        onOpenChange={(open: boolean) => {
          if (!open) setChildBranch(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("technologyDialogTitle")}</DialogTitle>
            <DialogDescription>
              {t("technologyDialogDescription")}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="surface-parent-id">
                {t("precedingTechnology")}
              </Label>
              <select
                id="surface-parent-id"
                className="ui-focus-ring h-9 w-full rounded-lg border border-input bg-card px-3 text-sm"
                value={parentTechnologyID}
                onChange={(event) => setParentTechnologyID(event.target.value)}
              >
                {childBranch?.technology_ids.map((id) => (
                  <option key={id} value={id}>
                    {technologyLabel(technologiesByID.get(id), id)}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1">
              <Label htmlFor="surface-child-name">{t("technologyName")}</Label>
              <Input
                id="surface-child-name"
                value={childName}
                onChange={(event) => setChildName(event.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="surface-child-description">
                {t("technologyDescription")}
              </Label>
              <Textarea
                id="surface-child-description"
                value={childDescription}
                onChange={(event) => setChildDescription(event.target.value)}
                required
              />
            </div>
            <div className="rounded-md border border-border bg-muted/30 px-3 py-2">
              <p className="text-xs font-medium text-foreground">
                {t("researchYear", { year: childYear ?? 0 })}
              </p>
              <p className="mt-1 text-micro text-muted-foreground">
                {t("yearComesFromPosition")}
              </p>
            </div>
            <p className="text-xs text-muted-foreground">
              {t("descriptionRequired")}
            </p>
            <details className="rounded-md border border-border px-3 py-2 text-xs text-muted-foreground">
              <summary className="cursor-pointer font-medium text-foreground">
                {t("technicalSettings")}
              </summary>
              <div className="mt-2 space-y-1">
                <Label htmlFor="surface-child-id">{t("technologyId")}</Label>
                <Input
                  id="surface-child-id"
                  value={childTechnologyID}
                  onChange={(event) => setChildTechnologyID(event.target.value)}
                  placeholder={t("generatedAutomatically")}
                />
              </div>
            </details>
          </div>
          <DialogFooter>
            <Button
              onClick={handleCreateChild}
              disabled={
                submitting ||
                !parentTechnologyID ||
                !childName.trim() ||
                !childDescription.trim()
              }
            >
              {submitting ? t("creatingTechnology") : t("createTechnology")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
