"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import {
  WarGoalEditor,
  type WargoalSaveStatus,
} from "@/components/workbench/WarGoalEditor";
import { wargoalsApi } from "@/lib/api/wargoals";
import type { WargoalType } from "@/types/wargoal";
import {
  toWargoalDraft,
  toWargoalWrite,
  type WargoalDraft,
} from "../project/[id]/wargoals/wargoalForm";
import { GamePanel, NUMBER, PanelSave } from "./GamePanel";
import { fileUndo, pick, recordChange, rowUndo } from "./changeLog";
import { changedKeys } from "./workbenchProject";
import type { GuiWarGoalType } from "./WarGoalLayer";

export const WARGOAL_FIELDS: readonly {
  key: string;
  labelKey: "baseDays" | "perStateDays" | "stateLimit" | "threat";
  of: (type: GuiWarGoalType) => number;
}[] = [
  {
    key: "generate_base_cost",
    labelKey: "baseDays",
    of: (type) => type.base_cost,
  },
  {
    key: "generate_per_state_cost",
    labelKey: "perStateDays",
    of: (type) => type.per_state_cost,
  },
  {
    key: "take_states_limit",
    labelKey: "stateLimit",
    of: (type) => type.limit,
  },
  { key: "threat", labelKey: "threat", of: (type) => type.threat },
];

export function warGoalValues(
  type: GuiWarGoalType,
  draft: Record<string, string>,
): Record<string, number> | null {
  const out: Record<string, number> = {};
  for (const field of WARGOAL_FIELDS) {
    const text = draft[field.key];
    if (text === undefined) continue;
    const value = Number(text);
    if (text.trim() === "" || !Number.isFinite(value) || value < 0) return null;
    if (value !== field.of(type)) out[field.key] = value;
  }
  return out;
}

export function projectWarGoalRow(
  rows: readonly WargoalType[] | undefined,
  id: string,
): WargoalType | null {
  const want = id.toLowerCase();
  return (
    rows?.find(
      (row) =>
        row.wargoal_key.toLowerCase() === want && row.source !== "imported",
    ) ?? null
  );
}

export function WarGoalEditPanel({
  projectId,
  type,
  onSaved,
  onClose,
}: {
  projectId: number;
  type: GuiWarGoalType;

  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const tw = useTranslations("gameViewWorld.warGoal");

  const rows = useSWR(
    ["game-view-wargoals-db", projectId],
    () => wargoalsApi.list(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  const row = projectWarGoalRow(rows.data?.wargoals, type.id);
  return (
    <GamePanel
      title={type.label}
      kind={tw("kind", { id: type.id })}
      source={row ? "项目" : undefined}
      workbenchHref={row ? `/project/${projectId}/wargoals` : null}

      wide={row ? "workbench" : false}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">{tw("loading")}</p>
      ) : row ? (
        <div className="-mx-4 mt-2">
          <ProjectWarGoal
            key={row.id}
            projectId={projectId}
            type={type}
            row={row}
            onSaved={onSaved}
          />
        </div>
      ) : (
        <WarGoalNumbers projectId={projectId} type={type} onSaved={onSaved} />
      )}
    </GamePanel>
  );
}

function ProjectWarGoal({
  projectId,
  type,
  row,
  onSaved,
}: {
  projectId: number;
  type: GuiWarGoalType;
  row: WargoalType;
  onSaved: (message: string) => void;
}) {
  const t = useTranslations("wargoals");
  const tw = useTranslations("gameViewWorld.warGoal");
  const tPerson = useTranslations("gameViewEdit.person");

  const [base, setBase] = useState<WargoalType>(row);
  const [draft, setDraft] = useState<WargoalDraft>(() => toWargoalDraft(row));
  const [status, setStatus] = useState<WargoalSaveStatus>("idle");
  const [error, setError] = useState("");

  const save = async () => {
    if (!draft.wargoal_key.trim()) return;
    const id = base.id;
    const payload = toWargoalWrite(draft);
    const keys = changedKeys(base, payload);
    const before = pick(base, keys);
    setStatus("saving");
    setError("");
    let saved: WargoalType;
    try {
      saved = await wargoalsApi.update(projectId, id, payload);
    } catch {
      setStatus("error");
      setError(t("saveFailed"));
      return;
    }
    recordChange({
      projectId,
      title: type.label,
      kind: tw("kind", { id: saved.wargoal_key }),
      workbenchHref: `/project/${projectId}/wargoals`,
      undo: keys.length
        ? rowUndo<WargoalType>({
            before,
            after: pick(saved, keys),
            load: async () =>
              (await wargoalsApi.list(projectId)).wargoals.find(
                (item) => item.id === id,
              ) ?? null,

            save: (back, now) =>
              wargoalsApi.update(projectId, id, {
                ...toWargoalWrite(toWargoalDraft(now)),
                ...back,
              }),
          })
        : undefined,
    });
    setBase(saved);
    setDraft(toWargoalDraft(saved));
    setStatus("saved");
    onSaved(tw("saved", { name: type.label }));
  };

  const remove = async () => {
    if (!window.confirm(t("deleteConfirm", { id: base.wargoal_key }))) return;
    try {
      await wargoalsApi.delete(projectId, base.id);
    } catch {
      setStatus("error");
      setError(t("deleteFailed"));
      return;
    }
    recordChange({
      projectId,
      title: tPerson("deleteLog", { name: type.label }),
      kind: tw("kind", { id: base.wargoal_key }),
      workbenchHref: `/project/${projectId}/wargoals`,
    });

    onSaved(tw("deleted", { name: type.label }));
  };

  return (
    <WarGoalEditor
      draft={draft}
      savedDraft={toWargoalDraft(base)}
      onChange={setDraft}
      status={status}
      error={error}
      onSave={() => void save()}
      onDelete={() => void remove()}
    />
  );
}

function WarGoalNumbers({
  projectId,
  type,
  onSaved,
}: {
  projectId: number;
  type: GuiWarGoalType;
  onSaved: (message: string) => void;
}) {
  const tw = useTranslations("gameViewWorld.warGoal");
  const [draft, setDraft] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const values = warGoalValues(type, draft);
  const changed = !!values && Object.keys(values).length > 0;
  const save = async () => {
    if (!values || !changed) return;
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(
        `/api/go/projects/${projectId}/gui-render/wargoal-save`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name: type.id, values }),
        },
      );
      const out = await response.json().catch(() => ({}));
      if (!response.ok)
        throw new Error(
          (out as { detail?: string }).detail || `HTTP ${response.status}`,
        );
      recordChange({
        projectId,
        title: type.label,
        kind: tw("kind", { id: type.id }),
        undo: fileUndo(projectId, out),
      });
      setDraft({});
      onSaved(tw("saved", { name: type.label }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className="mt-3 space-y-2">
        {WARGOAL_FIELDS.map((field) => (
          <label key={field.key} className="block text-xs">
            {tw(`field.${field.labelKey}`)}
            <input
              aria-label={field.key}
              className={NUMBER}
              inputMode="decimal"
              value={draft[field.key] ?? String(field.of(type))}
              onChange={(e) =>
                setDraft({ ...draft, [field.key]: e.target.value })
              }
            />
          </label>
        ))}
        <p className="text-micro text-muted-foreground">{tw("formula")}</p>
      </div>
      <PanelSave
        onSave={save}
        canSave={changed}
        busy={busy}
        onRevert={changed ? () => setDraft({}) : undefined}
        problem={values === null ? tw("problem") : null}
        note={tw("note")}
        error={error}
      />
    </>
  );
}
