"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import {
  OobEntityEditor,
  type OobSaveState,
} from "@/components/workbench/oob-units/OobEntityEditor";
import {
  airWingToDraft,
  airWingUpdate,
  taskForceToDraft,
  taskForceUpdate,
  type OobEntityDraft,
} from "@/components/workbench/oob-units/oobForm";
import { oobUnitsApi } from "@/lib/api/oobUnits";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import type { AirWing, OobUnitsSnapshot, TaskForce } from "@/types/oobUnits";
import type { GuiAirWing, GuiTaskForce } from "./ArmyOverviewLayer";
import { FIELD, GamePanel, NUMBER, PanelSave } from "./GamePanel";
import { fileUndo, pick, recordChange, rowUndo } from "./changeLog";
import { changedKeys } from "./workbenchProject";

export type OOBUnit =
  | { kind: "task_force"; unit: GuiTaskForce }
  | { kind: "air_wing"; unit: GuiAirWing };

export interface OOBUnitDraft {
  name?: string;
  amount?: string;
}

type Changes = { name?: string; amount?: number };

export function oobUnitChanges(
  item: OOBUnit,
  draft: OOBUnitDraft,
): Changes | null {
  const out: Changes = {};
  if (draft.name !== undefined) {
    const name = draft.name.trim();
    if (!name || /["{}]/.test(name)) return null;
    if (name !== item.unit.name) out.name = name;
  }
  if (item.kind === "air_wing" && draft.amount !== undefined) {
    const amount = Number(draft.amount);
    if (
      draft.amount.trim() === "" ||
      !Number.isInteger(amount) ||
      amount < 1 ||
      amount > 1000
    )
      return null;
    if (amount !== item.unit.count) out.amount = amount;
  }
  return out;
}

export function oobUnitTarget(item: OOBUnit) {
  const { oob, ordinal, raw_name } = item.unit;
  return item.kind === "task_force"
    ? { kind: item.kind, oob, ordinal, raw_name }
    : {
        kind: item.kind,
        oob,
        ordinal,
        raw_name,
        state: item.unit.state,
        state_index: item.unit.state_index,
      };
}

export type OOBUnitRow =
  { kind: "task-force"; row: TaskForce } | { kind: "air-wing"; row: AirWing };

const unquote = (text: string) => text.replace(/^"+|"+$/g, "");

export const oobFileOf = (item: OOBUnit) => unquote(item.unit.oob.trim());

export function oobUnitRow(
  snapshot: OobUnitsSnapshot | undefined,
  item: OOBUnit,
  savedId: number | null = null,
): OOBUnitRow | null {
  if (!snapshot) return null;
  if (item.kind === "task_force") {
    const saved =
      savedId === null
        ? undefined
        : snapshot.task_forces.find((row) => row.id === savedId);
    if (saved) return { kind: "task-force", row: saved };
    for (const fleet of snapshot.fleets) {
      if (fleet.source === "imported") continue;
      const force = snapshot.task_forces.find(
        (row) =>
          row.fleet_id === fleet.id && unquote(row.name) === item.unit.raw_name,
      );
      if (force) return { kind: "task-force", row: force };
    }
    return null;
  }
  const saved =
    savedId === null
      ? undefined
      : snapshot.air_wings.find((row) => row.id === savedId);
  if (saved) return { kind: "air-wing", row: saved };
  const group = snapshot.air_wing_groups.find(
    (row) =>
      row.source !== "imported" &&
      row.location.trim() === String(item.unit.state),
  );
  const wing = group
    ? snapshot.air_wings.filter((row) => row.group_id === group.id)[
        item.unit.state_index
      ]
    : undefined;
  return wing ? { kind: "air-wing", row: wing } : null;
}

export function oobUnitDraft(found: OOBUnitRow): OobEntityDraft {
  return found.kind === "task-force"
    ? {
        kind: "task-force",
        rowId: found.row.id,
        value: taskForceToDraft(found.row),
      }
    : {
        kind: "air-wing",
        rowId: found.row.id,
        value: airWingToDraft(found.row),
      };
}

export function OOBUnitEditPanel({
  projectId,
  item,
  onSaved,
  onClose,
}: {
  projectId: number;
  item: OOBUnit;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewMilitary.oob");
  const oob = oobFileOf(item);

  const rows = useSWR(
    ["game-view-oob-units-db", projectId, oob],
    () => oobUnitsApi.snapshot(projectId, { oobFile: oob }),
    {
      revalidateOnFocus: false,
    },
  );

  const [savedId, setSavedId] = useState<number | null>(null);
  const found = oobUnitRow(rows.data, item, savedId);
  const wing = item.kind === "air_wing";
  const kind = t(wing ? "kindAirWing" : "kindTaskForce", {
    n: item.unit.ordinal + 1,
  });

  return (
    <GamePanel
      title={item.unit.name}
      kind={kind}
      detail={`history/units/${item.unit.oob}.txt`}
      source={found ? "项目" : undefined}
      workbenchHref={found ? `/project/${projectId}/oob-units` : null}

      wide={found ? "workbench" : false}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">{t("loading")}</p>
      ) : found && rows.data ? (
        <div className="-mx-4 mt-2">
          <ProjectUnit
            key={`${found.kind}:${found.row.id}`}
            projectId={projectId}
            name={item.unit.name}
            kind={kind}
            wing={wing}
            found={found}
            snapshot={rows.data}
            onRowSaved={(id) => {
              setSavedId(id);
              void rows.mutate();
            }}
            onSaved={onSaved}
            onClose={onClose}
          />
        </div>
      ) : (
        <UnitFileForm projectId={projectId} item={item} onSaved={onSaved} />
      )}
    </GamePanel>
  );
}

async function loadUnitRow(
  projectId: number,
  kind: OOBUnitRow["kind"],
  id: number,
): Promise<TaskForce | AirWing | undefined> {
  const snapshot = await oobUnitsApi.snapshot(projectId);
  return kind === "task-force"
    ? snapshot.task_forces.find((row) => row.id === id)
    : snapshot.air_wings.find((row) => row.id === id);
}

function ProjectUnit({
  projectId,
  name,
  kind,
  wing,
  found,
  snapshot,
  onRowSaved,
  onSaved,
  onClose,
}: {
  projectId: number;

  name: string;
  kind: string;

  wing: boolean;
  found: OOBUnitRow;
  snapshot: OobUnitsSnapshot;

  onRowSaved: (id: number) => void;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("oobUnitsEditor");
  const tMil = useTranslations("gameViewMilitary");
  const confirmDialog = useConfirm();

  const [base, setBase] = useState<OOBUnitRow>(found);
  const [draft, setDraft] = useState<OobEntityDraft>(() => oobUnitDraft(found));
  const [savedDraft, setSavedDraft] = useState<OobEntityDraft>(draft);
  const [saveState, setSaveState] = useState<OobSaveState>("idle");
  const [error, setError] = useState<string | null>(null);
  const [removeError, setRemoveError] = useState<string | null>(null);
  const isUnsaved = JSON.stringify(draft) !== JSON.stringify(savedDraft);
  useRegisterDirty("oob-units", isUnsaved);
  const workbenchHref = `/project/${projectId}/oob-units`;
  const id = base.row.id;
  const label = (row: TaskForce | AirWing) => unquote(row.name.trim()) || name;

  const change = (field: string, value: string) => {
    setDraft(
      (current) =>
        ({
          ...current,
          value: { ...current.value, [field]: value },
        }) as OobEntityDraft,
    );
    setError(null);
    setSaveState("idle");
  };

  async function write<T extends TaskForce | AirWing>(
    row: T,
    payload: Partial<T>,
    update: (data: Partial<T>) => Promise<T>,
  ): Promise<T> {
    const keys = changedKeys(row, payload);
    const before = pick(row, keys);
    const saved = await update(payload);
    recordChange({
      projectId,
      title: label(saved),
      kind,
      workbenchHref,
      undo: keys.length
        ? rowUndo<T>({
            before,
            after: pick(saved, keys),
            load: async () =>
              (await loadUnitRow(projectId, base.kind, row.id)) as
                T | undefined,
            save: (back) => update(back),
          })
        : undefined,
    });
    return saved;
  }

  const save = async () => {
    const submitted = draft;
    setSaveState("saving");
    setError(null);
    try {
      let saved: OOBUnitRow;
      if (base.kind === "task-force" && submitted.kind === "task-force") {
        const row = await write<TaskForce>(
          base.row,
          taskForceUpdate(submitted.value),
          (data) => oobUnitsApi.taskForces.update(projectId, id, data),
        );
        saved = { kind: "task-force", row };
      } else if (base.kind === "air-wing" && submitted.kind === "air-wing") {
        const payload = airWingUpdate(submitted.value);
        if (!payload.equipment_type) {
          setError(t("equipmentTypeRequired"));
          setSaveState("error");
          return;
        }
        const row = await write<AirWing>(base.row, payload, (data) =>
          oobUnitsApi.airWings.update(projectId, id, data),
        );
        saved = { kind: "air-wing", row };
      } else {
        return;
      }
      const next = oobUnitDraft(saved);
      setBase(saved);

      setDraft((current) => (current === submitted ? next : current));
      setSavedDraft(next);
      setSaveState("saved");
      onRowSaved(saved.row.id);
      onSaved(
        tMil(wing ? "oob.savedAirWing" : "oob.savedTaskForce", {
          name: label(saved.row),
        }),
      );
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : String(caught));
      setSaveState("error");
    }
  };

  const remove = async () => {
    const body =
      base.kind === "task-force"
        ? t("deleteTaskForceConfirm", {
            count: snapshot.ships.filter((ship) => ship.task_force_id === id)
              .length,
          })
        : t("deleteLeafConfirm");
    if (
      !(await confirmDialog({ body, danger: true, confirmLabel: t("delete") }))
    )
      return;
    setRemoveError(null);
    try {
      if (base.kind === "task-force")
        await oobUnitsApi.taskForces.remove(projectId, id);
      else await oobUnitsApi.airWings.remove(projectId, id);
    } catch (caught) {
      setRemoveError(caught instanceof Error ? caught.message : String(caught));
      return;
    }

    recordChange({
      projectId,
      title: tMil("log.delete", { name: label(base.row) }),
      kind,
      workbenchHref,
    });
    onSaved(
      tMil(wing ? "oob.deletedAirWing" : "oob.deletedTaskForce", {
        name: label(base.row),
      }),
    );

    onClose();
  };

  return (
    <>
      <OobEntityEditor
        projectId={projectId}
        draft={draft}
        snapshot={snapshot}
        isUnsaved={isUnsaved}
        saveState={saveState}
        error={error}
        onFieldChange={change}
        onSave={() => void save()}
        onDelete={() => void remove()}
      />
      {removeError && (
        <p role="alert" className="px-6 pb-2 text-micro text-destructive">
          {t("operationError", { error: removeError })}
        </p>
      )}
    </>
  );
}

function UnitFileForm({
  projectId,
  item,
  onSaved,
}: {
  projectId: number;
  item: OOBUnit;
  onSaved: (message: string) => void;
}) {
  const t = useTranslations("gameViewMilitary");
  const [draft, setDraft] = useState<OOBUnitDraft>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const wing = item.kind === "air_wing";
  const changes = oobUnitChanges(item, draft);
  const changed = !!changes && Object.keys(changes).length > 0;
  const save = async () => {
    if (!changes || !changed) return;
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(
        `/api/go/projects/${projectId}/gui-render/oob-unit-save`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ ...oobUnitTarget(item), ...changes }),
        },
      );
      const out = await response.json().catch(() => ({}));
      if (!response.ok)
        throw new Error(
          (out as { detail?: string }).detail || `HTTP ${response.status}`,
        );
      recordChange({
        projectId,
        title: changes.name ?? item.unit.name,
        kind: t(wing ? "oob.kindAirWing" : "oob.kindTaskForce", {
          n: item.unit.ordinal + 1,
        }),
        undo: fileUndo(projectId, out),
      });
      setDraft({});
      onSaved(
        t(wing ? "oob.savedAirWing" : "oob.savedTaskForce", {
          name: changes.name ?? item.unit.name,
        }),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className="mt-3 space-y-2">
        <label className="block text-xs">
          {t("common.name")}
          <input
            aria-label="name"
            className={FIELD}
            value={draft.name ?? item.unit.name}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
        </label>
        {item.kind === "air_wing" && (
          <label className="block text-xs">
            {t("oob.planes", { limit: item.unit.limit })}
            <input
              aria-label="amount"
              className={NUMBER}
              inputMode="numeric"
              value={draft.amount ?? String(item.unit.count)}
              onChange={(e) => setDraft({ ...draft, amount: e.target.value })}
            />
          </label>
        )}
        <p className="text-micro text-muted-foreground">
          {item.kind === "air_wing"
            ? item.unit.design
              ? t("oob.basedAtDesign", {
                  base: item.unit.base,
                  design: item.unit.design,
                })
              : t("oob.basedAt", { base: item.unit.base })
            : item.unit.fleet
              ? t("oob.shipsInFleet", {
                  fleet: item.unit.fleet,
                  size: item.unit.size,
                })
              : t("oob.ships", { size: item.unit.size })}
        </p>
      </div>
      <PanelSave
        onSave={save}
        canSave={changed}
        busy={busy}
        onRevert={changed ? () => setDraft({}) : undefined}
        problem={
          changes === null
            ? t(wing ? "oob.problemAirWing" : "oob.problemTaskForce")
            : null
        }
        note={t(wing ? "oob.noteAirWing" : "oob.noteTaskForce")}
        error={error}
      />
    </>
  );
}
