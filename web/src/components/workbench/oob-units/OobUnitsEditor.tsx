"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { OpenInGameLink } from "@/components/workbench/OpenInGameLink";
import { gameViewHref } from "@/lib/gameViewLinks";
import { useTranslations } from "use-intl";
import { FileUp, Plane, Plus, RefreshCw, ShipWheel } from "lucide-react";

import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { oobUnitsApi } from "@/lib/api/oobUnits";
import {
  normalizeCountryTag,
  pickDefaultOobFile,
  tagFromOob,
} from "@/lib/oob/oobDefaults";
import { useActiveCountry, useProjectStore } from "@/lib/store/projectStore";
import type {
  AirWing,
  AirWingGroup,
  Fleet,
  OobUnitsSnapshot,
  Ship,
  TaskForce,
} from "@/types/oobUnits";

import { OobEntityEditor } from "./OobEntityEditor";
import { OobImportPanel } from "./OobImportPanel";
import { OobScriptPreview } from "./OobScriptPreview";
import {
  airWingGroupToDraft,
  airWingGroupUpdate,
  airWingToDraft,
  airWingUpdate,
  fleetToDraft,
  fleetUpdate,
  shipToDraft,
  shipUpdate,
  taskForceToDraft,
  taskForceUpdate,
  type OobEntityDraft,
} from "./oobForm";

type Mode = "air" | "naval";
type EntityKind = OobEntityDraft["kind"];
type Selection = Pick<OobEntityDraft, "kind" | "rowId">;

const EMPTY_SNAPSHOT: OobUnitsSnapshot = {
  air_wing_groups: [],
  air_wings: [],
  fleets: [],
  task_forces: [],
  ships: [],
};

const selectionDraft = (
  snapshot: OobUnitsSnapshot,
  selection: Selection,
): OobEntityDraft | null => {
  switch (selection.kind) {
    case "air-wing-group": {
      const row = snapshot.air_wing_groups.find(
        (item) => item.id === selection.rowId,
      );
      return row
        ? {
            kind: selection.kind,
            rowId: row.id,
            value: airWingGroupToDraft(row),
          }
        : null;
    }
    case "air-wing": {
      const row = snapshot.air_wings.find(
        (item) => item.id === selection.rowId,
      );
      return row
        ? { kind: selection.kind, rowId: row.id, value: airWingToDraft(row) }
        : null;
    }
    case "fleet": {
      const row = snapshot.fleets.find((item) => item.id === selection.rowId);
      return row
        ? { kind: selection.kind, rowId: row.id, value: fleetToDraft(row) }
        : null;
    }
    case "task-force": {
      const row = snapshot.task_forces.find(
        (item) => item.id === selection.rowId,
      );
      return row
        ? { kind: selection.kind, rowId: row.id, value: taskForceToDraft(row) }
        : null;
    }
    case "ship": {
      const row = snapshot.ships.find((item) => item.id === selection.rowId);
      return row
        ? { kind: selection.kind, rowId: row.id, value: shipToDraft(row) }
        : null;
    }
  }
};

interface OobUnitsEditorProps {
  projectId: number;
}

export function OobUnitsEditor({ projectId }: OobUnitsEditorProps) {
  const t = useTranslations("oobUnitsEditor");

  const activeCountry = useActiveCountry();
  const projectStartDate = useProjectStore(
    (state) => state.currentProject?.start_date ?? null,
  );
  const confirmDialog = useConfirm();
  const snapshotSequence = useRef(0);
  const draftGeneration = useRef(0);
  const draftIdentity = useRef(0);
  const draftRef = useRef<OobEntityDraft | null>(null);
  const [mode, setMode] = useState<Mode>("air");
  const [snapshot, setSnapshot] = useState<OobUnitsSnapshot>(EMPTY_SNAPSHOT);
  const [knownFiles, setKnownFiles] = useState<string[]>([]);
  const [selectedOob, setSelectedOob] = useState("");
  const [draft, setDraft] = useState<OobEntityDraft | null>(null);
  const [savedDraft, setSavedDraft] = useState<OobEntityDraft | null>(null);
  const [loading, setLoading] = useState(true);
  const [snapshotLoadFailed, setSnapshotLoadFailed] = useState(false);
  const [operationError, setOperationError] = useState<string | null>(null);
  const [saveState, setSaveState] = useState<
    "idle" | "saving" | "saved" | "error"
  >("idle");
  const [showImport, setShowImport] = useState(false);
  const [previewToken, setPreviewToken] = useState(0);

  const replaceDraft = useCallback((next: OobEntityDraft | null) => {
    draftRef.current = next;
    setDraft(next);
  }, []);

  const replaceDraftFromUser = useCallback(
    (next: OobEntityDraft | null, saved: OobEntityDraft | null = null) => {
      draftGeneration.current += 1;
      draftIdentity.current += 1;
      replaceDraft(next);
      setSavedDraft(saved);
    },
    [replaceDraft],
  );

  const isUnsaved =
    draft !== null && JSON.stringify(draft) !== JSON.stringify(savedDraft);
  useRegisterDirty("oob-units", isUnsaved);

  const loadSnapshot = useCallback(
    async (
      oobFile: string,
      keepSelection: Selection | null = null,
      expectedDraftGeneration: number = draftGeneration.current,
    ): Promise<"applied" | "stale" | "failed"> => {
      const sequence = ++snapshotSequence.current;
      setLoading(true);
      setSnapshotLoadFailed(false);
      try {
        const allRows = await oobUnitsApi.snapshot(projectId);
        const nextRows = oobFile
          ? await oobUnitsApi.snapshot(projectId, { oobFile })
          : allRows;
        if (snapshotSequence.current !== sequence) return "stale";

        const files = new Set<string>();
        allRows.air_wing_groups.forEach(
          (row) => row.oob_file && files.add(row.oob_file),
        );
        allRows.fleets.forEach(
          (row) => row.oob_file && files.add(row.oob_file),
        );
        setKnownFiles(
          Array.from(files).sort((left, right) => left.localeCompare(right)),
        );
        setSnapshot(nextRows);

        if (draftGeneration.current === expectedDraftGeneration) {
          const nextDraft = keepSelection
            ? selectionDraft(nextRows, keepSelection)
            : null;
          draftGeneration.current += 1;
          draftIdentity.current += 1;
          replaceDraft(nextDraft);
          setSavedDraft(nextDraft);
          return "applied";
        }
        return "stale";
      } catch {
        if (snapshotSequence.current === sequence) {
          setSnapshotLoadFailed(true);
          return "failed";
        }
        return "stale";
      } finally {
        if (snapshotSequence.current === sequence) setLoading(false);
      }
    },
    [projectId, replaceDraft],
  );

  useEffect(() => {
    void loadSnapshot(selectedOob);
  }, [loadSnapshot, selectedOob]);

  const confirmDiscard = async (): Promise<boolean> =>
    !isUnsaved || (await confirmDialog({ body: t("discardDraftConfirm") }));

  const defaultOobFile = useMemo(
    () =>
      selectedOob ||
      pickDefaultOobFile({
        countryTag: activeCountry,
        startDate: projectStartDate,
        existing: knownFiles,
      }),
    [selectedOob, activeCountry, projectStartDate, knownFiles],
  );

  const changeMode = async (nextMode: Mode) => {
    if (nextMode === mode || !(await confirmDiscard())) return;
    replaceDraftFromUser(null);
    setSaveState("idle");
    setMode(nextMode);
  };

  const changeOob = async (nextOob: string) => {
    if (nextOob === selectedOob || !(await confirmDiscard())) return;

    snapshotSequence.current += 1;
    setSnapshot(EMPTY_SNAPSHOT);
    setSnapshotLoadFailed(false);
    replaceDraftFromUser(null);
    setOperationError(null);
    setSaveState("idle");
    setSelectedOob(nextOob);
  };

  const selectEntity = async (selection: Selection) => {
    if (draft?.kind === selection.kind && draft.rowId === selection.rowId) {
      return;
    }
    if (!(await confirmDiscard())) return;
    const nextDraft = selectionDraft(snapshot, selection);
    replaceDraftFromUser(nextDraft, nextDraft);
    setOperationError(null);
    setSaveState("idle");
  };

  const updateDraft = (field: string, value: string) => {
    const current = draftRef.current;
    if (!current) return;
    draftGeneration.current += 1;
    replaceDraft({
      ...current,
      value: { ...current.value, [field]: value },
    } as OobEntityDraft);
    setOperationError(null);
    setSaveState("idle");
  };

  const createTopLevel = async () => {
    if (!(await confirmDiscard())) return;
    setOperationError(null);
    const oobFile = defaultOobFile;
    const sourceFile = oobFile ? `history/units/${oobFile}.txt` : "";

    const countryTag =
      tagFromOob(oobFile) || normalizeCountryTag(activeCountry);
    if (mode === "air") {
      replaceDraftFromUser({
        kind: "air-wing-group",
        rowId: 0,
        value: {
          oob_file: oobFile,
          source_file: sourceFile,
          country_tag: countryTag,
          location: "",
          ordering: String(snapshot.air_wing_groups.length),
        },
      });
      setSaveState("idle");
      return;
    }
    const mutationGeneration = draftGeneration.current;
    try {
      if (mode === "naval") {
        const row = await oobUnitsApi.fleets.create(projectId, {
          oob_file: oobFile,
          source_file: sourceFile,
          country_tag: countryTag,
          name: "",
          naval_base: "",
          ordering: snapshot.fleets.length,
        });
        await loadSnapshot(
          selectedOob,
          { kind: "fleet", rowId: row.id },
          mutationGeneration,
        );
      }
      setPreviewToken((token) => token + 1);
    } catch (caught) {
      setOperationError(
        caught instanceof Error ? caught.message : String(caught),
      );
    }
  };

  const createAirWing = async (group: AirWingGroup, children: AirWing[]) => {
    if (!(await confirmDiscard())) return;
    setOperationError(null);
    replaceDraftFromUser({
      kind: "air-wing",
      rowId: 0,
      value: {
        group_id: String(group.id),
        equipment_type: "",
        name: "",
        owner: group.country_tag,
        amount: "1",
        ordering: String(children.length),
      },
    });
    setSaveState("idle");
  };

  const createTaskForce = async (fleet: Fleet, children: TaskForce[]) => {
    if (!(await confirmDiscard())) return;
    setOperationError(null);
    const mutationGeneration = draftGeneration.current;
    try {
      const row = await oobUnitsApi.taskForces.create(projectId, {
        fleet_id: fleet.id,
        name: "",
        location: fleet.naval_base,
        ordering: children.length,
      });
      await loadSnapshot(
        selectedOob,
        { kind: "task-force", rowId: row.id },
        mutationGeneration,
      );
      setPreviewToken((token) => token + 1);
    } catch (caught) {
      setOperationError(
        caught instanceof Error ? caught.message : String(caught),
      );
    }
  };

  const createShip = async (taskForce: TaskForce, children: Ship[]) => {
    if (!(await confirmDiscard())) return;
    setOperationError(null);
    const mutationGeneration = draftGeneration.current;
    try {
      const row = await oobUnitsApi.ships.create(projectId, {
        task_force_id: taskForce.id,
        name: "",
        definition: "",
        start_experience_factor: "",
        equipment_type: "",
        amount: "1",
        owner: "",
        version_name: "",
        ordering: children.length,
      });
      await loadSnapshot(
        selectedOob,
        { kind: "ship", rowId: row.id },
        mutationGeneration,
      );
      setPreviewToken((token) => token + 1);
    } catch (caught) {
      setOperationError(
        caught instanceof Error ? caught.message : String(caught),
      );
    }
  };

  const saveDraft = async () => {
    const submittedDraft = draftRef.current;
    if (!submittedDraft) return;
    const mutationGeneration = draftGeneration.current;
    const mutationIdentity = draftIdentity.current;
    let refreshGeneration = mutationGeneration;
    let selection: Selection = {
      kind: submittedDraft.kind,
      rowId: submittedDraft.rowId,
    };
    setSaveState("saving");
    setOperationError(null);
    try {
      switch (submittedDraft.kind) {
        case "air-wing-group": {
          const payload = airWingGroupUpdate(submittedDraft.value);
          if (!payload.location) {
            setOperationError(t("locationRequired"));
            setSaveState("error");
            return;
          }
          if (submittedDraft.rowId === 0) {
            const row = await oobUnitsApi.airWingGroups.create(
              projectId,
              payload,
            );
            selection = { kind: submittedDraft.kind, rowId: row.id };
            if (
              draftIdentity.current === mutationIdentity &&
              draftRef.current?.kind === submittedDraft.kind &&
              draftRef.current.rowId === 0
            ) {
              const persisted = {
                kind: submittedDraft.kind,
                rowId: row.id,
                value: airWingGroupToDraft(row),
              } as OobEntityDraft;
              const unchangedDuringRequest =
                draftGeneration.current === mutationGeneration;
              const nextDraft = unchangedDuringRequest
                ? persisted
                : { ...draftRef.current, rowId: row.id };
              draftGeneration.current += 1;
              refreshGeneration = unchangedDuringRequest
                ? draftGeneration.current
                : mutationGeneration;
              replaceDraft(nextDraft);
              setSavedDraft(persisted);
            }
          } else {
            await oobUnitsApi.airWingGroups.update(
              projectId,
              submittedDraft.rowId,
              payload,
            );
          }
          break;
        }
        case "air-wing": {
          const payload = airWingUpdate(submittedDraft.value);
          if (!payload.equipment_type) {
            setOperationError(t("equipmentTypeRequired"));
            setSaveState("error");
            return;
          }
          if (submittedDraft.rowId === 0) {
            const row = await oobUnitsApi.airWings.create(projectId, payload);
            selection = { kind: submittedDraft.kind, rowId: row.id };
            if (
              draftIdentity.current === mutationIdentity &&
              draftRef.current?.kind === submittedDraft.kind &&
              draftRef.current.rowId === 0
            ) {
              const persisted = {
                kind: submittedDraft.kind,
                rowId: row.id,
                value: airWingToDraft(row),
              } as OobEntityDraft;
              const unchangedDuringRequest =
                draftGeneration.current === mutationGeneration;
              const nextDraft = unchangedDuringRequest
                ? persisted
                : { ...draftRef.current, rowId: row.id };
              draftGeneration.current += 1;
              refreshGeneration = unchangedDuringRequest
                ? draftGeneration.current
                : mutationGeneration;
              replaceDraft(nextDraft);
              setSavedDraft(persisted);
            }
          } else {
            await oobUnitsApi.airWings.update(
              projectId,
              submittedDraft.rowId,
              payload,
            );
          }
          break;
        }
        case "fleet":
          await oobUnitsApi.fleets.update(
            projectId,
            submittedDraft.rowId,
            fleetUpdate(submittedDraft.value),
          );
          break;
        case "task-force":
          await oobUnitsApi.taskForces.update(
            projectId,
            submittedDraft.rowId,
            taskForceUpdate(submittedDraft.value),
          );
          break;
        case "ship":
          await oobUnitsApi.ships.update(
            projectId,
            submittedDraft.rowId,
            shipUpdate(submittedDraft.value),
          );
          break;
      }
      const refreshResult = await loadSnapshot(
        selectedOob,
        selection,
        refreshGeneration,
      );
      const unchangedAfterRefreshFailure =
        refreshResult === "failed" &&
        draftIdentity.current === mutationIdentity &&
        draftGeneration.current === refreshGeneration;
      if (unchangedAfterRefreshFailure) {
        setSavedDraft(draftRef.current);
      }
      setSaveState(
        refreshResult === "applied" || unchangedAfterRefreshFailure
          ? "saved"
          : "idle",
      );
      setPreviewToken((token) => token + 1);
    } catch (caught) {
      setOperationError(
        caught instanceof Error ? caught.message : String(caught),
      );
      setSaveState("error");
    }
  };

  const deleteDraft = async () => {
    if (!draft) return;
    let body = t("deleteLeafConfirm");
    if (draft.kind === "air-wing-group") {
      body = t("deleteAirGroupConfirm", {
        count: snapshot.air_wings.filter((row) => row.group_id === draft.rowId)
          .length,
      });
    } else if (draft.kind === "fleet") {
      const forceIds = snapshot.task_forces
        .filter((row) => row.fleet_id === draft.rowId)
        .map((row) => row.id);
      body = t("deleteFleetConfirm", {
        forces: forceIds.length,
        ships: snapshot.ships.filter((row) =>
          forceIds.includes(row.task_force_id),
        ).length,
      });
    } else if (draft.kind === "task-force") {
      body = t("deleteTaskForceConfirm", {
        count: snapshot.ships.filter((row) => row.task_force_id === draft.rowId)
          .length,
      });
    }
    if (
      !(await confirmDialog({
        body,
        danger: true,
        confirmLabel: t("delete"),
      }))
    ) {
      return;
    }

    setOperationError(null);
    try {
      switch (draft.kind) {
        case "air-wing-group":
          await oobUnitsApi.airWingGroups.remove(projectId, draft.rowId);
          break;
        case "air-wing":
          await oobUnitsApi.airWings.remove(projectId, draft.rowId);
          break;
        case "fleet":
          await oobUnitsApi.fleets.remove(projectId, draft.rowId);
          break;
        case "task-force":
          await oobUnitsApi.taskForces.remove(projectId, draft.rowId);
          break;
        case "ship":
          await oobUnitsApi.ships.remove(projectId, draft.rowId);
          break;
      }
      await loadSnapshot(selectedOob);
      setSaveState("idle");
      setPreviewToken((token) => token + 1);
    } catch (caught) {
      setOperationError(
        caught instanceof Error ? caught.message : String(caught),
      );
    }
  };

  const reload = async () => {
    if (!(await confirmDiscard())) return;
    replaceDraftFromUser(null);
    setOperationError(null);
    setSaveState("idle");
    await loadSnapshot(selectedOob);
    setPreviewToken((token) => token + 1);
  };

  const wingsByGroup = useMemo(() => {
    const result = new Map<number, AirWing[]>();
    snapshot.air_wings.forEach((wing) => {
      const children = result.get(wing.group_id) ?? [];
      children.push(wing);
      result.set(wing.group_id, children);
    });
    return result;
  }, [snapshot.air_wings]);

  const forcesByFleet = useMemo(() => {
    const result = new Map<number, TaskForce[]>();
    snapshot.task_forces.forEach((force) => {
      const children = result.get(force.fleet_id) ?? [];
      children.push(force);
      result.set(force.fleet_id, children);
    });
    return result;
  }, [snapshot.task_forces]);

  const shipsByForce = useMemo(() => {
    const result = new Map<number, Ship[]>();
    snapshot.ships.forEach((ship) => {
      const children = result.get(ship.task_force_id) ?? [];
      children.push(ship);
      result.set(ship.task_force_id, children);
    });
    return result;
  }, [snapshot.ships]);

  const isSelected = (kind: EntityKind, rowId: number) =>
    draft?.kind === kind && draft.rowId === rowId;

  const treeRowClass = (selected: boolean) =>
    `ui-interactive ui-focus-ring flex min-w-0 flex-1 items-center gap-2 rounded-md px-2 py-2 text-left text-xs ${
      selected
        ? "bg-sidebar-accent text-sidebar-primary font-semibold"
        : "text-foreground hover:bg-muted"
    }`;

  const hasSnapshotData =
    draft !== null ||
    snapshot.air_wing_groups.length > 0 ||
    snapshot.air_wings.length > 0 ||
    snapshot.fleets.length > 0 ||
    snapshot.task_forces.length > 0 ||
    snapshot.ships.length > 0;
  const showSnapshotLoadError = snapshotLoadFailed;

  if (showSnapshotLoadError && !hasSnapshotData) {
    return <WorkbenchLoadError onRetry={() => loadSnapshot(selectedOob)} />;
  }

  return (
    <div className="flex h-full min-w-0 flex-col bg-background">
      <header className="flex min-h-16 items-center gap-4 border-b border-border px-4">
        <div className="min-w-0">
          <h1 className="truncate text-base font-semibold tracking-tight">
            {t("title")}
          </h1>
          <p className="truncate text-xs text-muted-foreground">
            {t("subtitle")}
          </p>
        </div>
        <OpenInGameLink
          href={gameViewHref(projectId, {
            panel: "countryarmyview",
            tag: activeCountry,
          })}
        />

        <div
          className="ml-4 flex h-9 items-center rounded-md border border-border bg-muted p-1"
          role="group"
          aria-label={t("title")}
        >
          <button
            type="button"
            className={`ui-interactive ui-focus-ring flex h-7 items-center gap-2 rounded-md px-3 text-xs font-medium ${
              mode === "air"
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:text-foreground"
            }`}
            aria-pressed={mode === "air"}
            onClick={() => void changeMode("air")}
          >
            <Plane className="h-4 w-4" aria-hidden />
            {t("airMode")}
          </button>
          <button
            type="button"
            className={`ui-interactive ui-focus-ring flex h-7 items-center gap-2 rounded-md px-3 text-xs font-medium ${
              mode === "naval"
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:text-foreground"
            }`}
            aria-pressed={mode === "naval"}
            onClick={() => void changeMode("naval")}
          >
            <ShipWheel className="h-4 w-4" aria-hidden />
            {t("navalMode")}
          </button>
        </div>

        <div className="ml-auto flex items-center gap-2">
          <Label
            htmlFor="oob-file-filter"
            className="text-xs text-muted-foreground"
          >
            {t("oobFilter")}
          </Label>
          <select
            id="oob-file-filter"
            className="h-9 min-w-48 rounded-md border border-input bg-card px-3 font-mono text-xs ui-focus-ring"
            value={selectedOob}
            onChange={(event) => void changeOob(event.target.value)}
          >
            <option value="">{t("allFiles")}</option>
            {knownFiles.map((file) => (
              <option key={file} value={file}>
                {file}
              </option>
            ))}
          </select>
          <Button
            type="button"
            size="icon"
            variant="outline"
            className="h-9 w-9 ui-interactive"
            title={t("reload")}
            aria-label={t("reload")}
            disabled={loading}
            onClick={() => void reload()}
          >
            <RefreshCw className="h-4 w-4" aria-hidden />
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="gap-2 ui-interactive"
            onClick={() => setShowImport((visible) => !visible)}
          >
            <FileUp className="h-4 w-4" aria-hidden />
            {showImport ? t("hideImport") : t("showImport")}
          </Button>
          <Button
            type="button"
            size="sm"
            className="gap-2 ui-interactive"
            onClick={() => void createTopLevel()}
          >
            <Plus className="h-4 w-4" aria-hidden />
            {mode === "air" ? t("newAirGroup") : t("newFleet")}
          </Button>
        </div>
      </header>

      {showImport && (
        <OobImportPanel
          key={selectedOob || "all"}
          projectId={projectId}
          defaultOobFile={defaultOobFile}
          beforeImport={async () => {
            if (!(await confirmDiscard())) return false;
            replaceDraftFromUser(null);
            return true;
          }}
          onImported={async () => {
            await loadSnapshot(selectedOob);
            setPreviewToken((token) => token + 1);
          }}
        />
      )}

      {showSnapshotLoadError && (
        <div className="border-b border-border p-2">
          <WorkbenchLoadError
            variant="compact"
            onRetry={() => loadSnapshot(selectedOob)}
          />
        </div>
      )}
      {operationError && saveState !== "error" && (
        <p
          role="alert"
          className="border-b border-destructive/20 bg-destructive/10 px-4 py-2 text-xs text-destructive"
        >
          {t("operationError", { error: operationError })}
        </p>
      )}

      <div className="flex min-h-0 flex-1">
        <aside className="flex w-80 shrink-0 flex-col border-r border-border bg-sidebar">
          <div className="flex min-h-14 items-center justify-between border-b border-sidebar-border px-3">
            <p className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
              {mode === "air" ? t("airMode") : t("navalMode")}
            </p>
            <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
              {mode === "air"
                ? snapshot.air_wing_groups.length + snapshot.air_wings.length
                : snapshot.fleets.length +
                  snapshot.task_forces.length +
                  snapshot.ships.length}
            </span>
          </div>

          <div className="min-h-0 flex-1 space-y-1 overflow-y-auto p-2">
            {loading && (
              <div className="animate-pulse-ring rounded-md border border-accent p-3 text-xs text-muted-foreground">
                {t("loadingSnapshot")}
              </div>
            )}
            {!loading &&
              mode === "air" &&
              snapshot.air_wing_groups.length === 0 && (
                <EditorEmptyState
                  className="m-2"
                  icon={<Plane className="h-8 w-8 text-muted-foreground/60" />}
                  title={t("emptyAirTitle")}
                  description={t("emptyAirHint")}
                  actions={
                    <Button size="sm" onClick={() => void createTopLevel()}>
                      {t("newAirGroup")}
                    </Button>
                  }
                />
              )}
            {!loading &&
              mode === "air" &&
              snapshot.air_wing_groups.map((group) => {
                const wings = wingsByGroup.get(group.id) ?? [];
                return (
                  <div key={group.id} className="space-y-1">
                    <div className="flex items-center gap-1">
                      <button
                        type="button"
                        className={treeRowClass(
                          isSelected("air-wing-group", group.id),
                        )}
                        onClick={() =>
                          void selectEntity({
                            kind: "air-wing-group",
                            rowId: group.id,
                          })
                        }
                      >
                        <Plane className="h-4 w-4 shrink-0" aria-hidden />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate">
                            {group.location || t("unnamedAirGroup")}
                          </span>
                          <span className="block truncate font-mono text-micro font-normal text-muted-foreground">
                            {group.oob_file}
                          </span>
                        </span>
                      </button>
                      <Button
                        type="button"
                        size="icon"
                        variant="ghost"
                        className="h-8 w-8 shrink-0 ui-interactive"
                        title={t("addAirWing")}
                        aria-label={t("addAirWing")}
                        onClick={() => void createAirWing(group, wings)}
                      >
                        <Plus className="h-4 w-4" aria-hidden />
                      </Button>
                    </div>
                    {wings.map((wing) => (
                      <button
                        key={wing.id}
                        type="button"
                        className={`${treeRowClass(isSelected("air-wing", wing.id))} ml-7`}
                        onClick={() =>
                          void selectEntity({
                            kind: "air-wing",
                            rowId: wing.id,
                          })
                        }
                      >
                        <span
                          className="h-1.5 w-1.5 shrink-0 rounded-full bg-muted-foreground"
                          aria-hidden
                        />
                        <span className="min-w-0 flex-1 truncate">
                          {wing.name ||
                            wing.equipment_type ||
                            t("unnamedAirWing")}
                        </span>
                        <span className="font-mono text-micro font-normal text-muted-foreground">
                          {wing.amount}
                        </span>
                      </button>
                    ))}
                  </div>
                );
              })}

            {!loading && mode === "naval" && snapshot.fleets.length === 0 && (
              <EditorEmptyState
                className="m-2"
                icon={
                  <ShipWheel className="h-8 w-8 text-muted-foreground/60" />
                }
                title={t("emptyNavalTitle")}
                description={t("emptyNavalHint")}
                actions={
                  <Button size="sm" onClick={() => void createTopLevel()}>
                    {t("newFleet")}
                  </Button>
                }
              />
            )}
            {!loading &&
              mode === "naval" &&
              snapshot.fleets.map((fleet) => {
                const forces = forcesByFleet.get(fleet.id) ?? [];
                return (
                  <div key={fleet.id} className="space-y-1">
                    <div className="flex items-center gap-1">
                      <button
                        type="button"
                        className={treeRowClass(isSelected("fleet", fleet.id))}
                        onClick={() =>
                          void selectEntity({ kind: "fleet", rowId: fleet.id })
                        }
                      >
                        <ShipWheel className="h-4 w-4 shrink-0" aria-hidden />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate">
                            {fleet.name || t("unnamedFleet")}
                          </span>
                          <span className="block truncate font-mono text-micro font-normal text-muted-foreground">
                            {fleet.oob_file}
                          </span>
                        </span>
                      </button>
                      <Button
                        type="button"
                        size="icon"
                        variant="ghost"
                        className="h-8 w-8 shrink-0 ui-interactive"
                        title={t("addTaskForce")}
                        aria-label={t("addTaskForce")}
                        onClick={() => void createTaskForce(fleet, forces)}
                      >
                        <Plus className="h-4 w-4" aria-hidden />
                      </Button>
                    </div>
                    {forces.map((force) => {
                      const ships = shipsByForce.get(force.id) ?? [];
                      return (
                        <div key={force.id} className="space-y-1 pl-7">
                          <div className="flex items-center gap-1">
                            <button
                              type="button"
                              className={treeRowClass(
                                isSelected("task-force", force.id),
                              )}
                              onClick={() =>
                                void selectEntity({
                                  kind: "task-force",
                                  rowId: force.id,
                                })
                              }
                            >
                              <span
                                className="h-2 w-2 shrink-0 rounded-sm border border-muted-foreground"
                                aria-hidden
                              />
                              <span className="min-w-0 flex-1 truncate">
                                {force.name || t("unnamedTaskForce")}
                              </span>
                            </button>
                            <Button
                              type="button"
                              size="icon"
                              variant="ghost"
                              className="h-8 w-8 shrink-0 ui-interactive"
                              title={t("addShip")}
                              aria-label={t("addShip")}
                              onClick={() => void createShip(force, ships)}
                            >
                              <Plus className="h-4 w-4" aria-hidden />
                            </Button>
                          </div>
                          {ships.map((ship) => (
                            <button
                              key={ship.id}
                              type="button"
                              className={`${treeRowClass(isSelected("ship", ship.id))} ml-7`}
                              onClick={() =>
                                void selectEntity({
                                  kind: "ship",
                                  rowId: ship.id,
                                })
                              }
                            >
                              <span
                                className="h-1.5 w-1.5 shrink-0 rounded-full bg-muted-foreground"
                                aria-hidden
                              />
                              <span className="min-w-0 flex-1 truncate">
                                {ship.name ||
                                  ship.definition ||
                                  t("unnamedShip")}
                              </span>
                              <span className="font-mono text-micro font-normal text-muted-foreground">
                                {ship.amount}
                              </span>
                            </button>
                          ))}
                        </div>
                      );
                    })}
                  </div>
                );
              })}
          </div>
        </aside>

        <main className="min-w-0 flex-1 overflow-y-auto">
          {!draft ? (
            <div className="flex h-full items-center justify-center p-8 text-center">
              <div className="space-y-3 text-muted-foreground">
                {mode === "air" ? (
                  <Plane className="mx-auto h-8 w-8 opacity-50" aria-hidden />
                ) : (
                  <ShipWheel
                    className="mx-auto h-8 w-8 opacity-50"
                    aria-hidden
                  />
                )}
                <p className="text-sm font-medium">{t("selectEntity")}</p>
              </div>
            </div>
          ) : (
            <OobEntityEditor
              projectId={projectId}
              draft={draft}
              snapshot={snapshot}
              isUnsaved={isUnsaved}
              saveState={saveState}
              error={operationError}
              onFieldChange={updateDraft}
              onSave={() => void saveDraft()}
              onDelete={() => void deleteDraft()}
            />
          )}
        </main>

        <OobScriptPreview
          projectId={projectId}
          oobFile={selectedOob}
          refreshToken={previewToken}
        />
      </div>
    </div>
  );
}
