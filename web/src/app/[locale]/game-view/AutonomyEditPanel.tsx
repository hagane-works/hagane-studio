"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { guiRenderApi } from "@/lib/guiRenderApi";
import { autonomyStatesApi } from "@/lib/api";
import type { AutonomyState } from "@/types";
import { AutonomyStateEditor } from "@/components/workbench/AutonomyStateEditor";
import { FIELD, GamePanel, NUMBER, PanelSave, SCRIPT } from "./GamePanel";
import type { GuiAutonomyLevel } from "./SubjectsLayer";
import {
  fileUndo,
  pick,
  recordChange,
  rowUndo,
  textBefore,
  textUndo,
  undoAll,
} from "./changeLog";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

export interface AutonomyDraft {
  name?: string;
  min_freedom_level?: string;
  manpower_influence?: string;
  rule?: string;
  modifier?: string;
}

type Changes = {
  name?: string;
  values?: Record<string, number | null>;
  rule?: string;
  modifier?: string;
};

export function autonomyChanges(
  level: GuiAutonomyLevel,
  draft: AutonomyDraft,
): Changes | null {
  const out: Changes = {};
  const values: Record<string, number | null> = {};
  const number = (
    key: "min_freedom_level" | "manpower_influence",
    current: number | undefined,
    optional: boolean,
  ) => {
    const text = draft[key];
    if (text === undefined) return true;
    if (text.trim() === "") {
      if (!optional) return false;
      if (current !== undefined) values[key] = null;
      return true;
    }
    const value = Number(text);
    if (!Number.isFinite(value) || value < 0 || value > 1) return false;
    if (value !== current) values[key] = value;
    return true;
  };
  if (!number("min_freedom_level", level.min_freedom_level, false)) return null;
  if (!number("manpower_influence", level.manpower_influence, true))
    return null;
  if (Object.keys(values).length) out.values = values;
  if (draft.name !== undefined) {
    const name = draft.name.trim();
    if (!name) return null;
    if (name !== level.label) out.name = name;
  }
  if (draft.rule !== undefined && draft.rule.trim() !== level.rule)
    out.rule = draft.rule.trim();
  if (draft.modifier !== undefined && draft.modifier.trim() !== level.modifier)
    out.modifier = draft.modifier.trim();
  return out;
}

export function projectAutonomyRow(
  rows: readonly AutonomyState[] | undefined,
  id: string,
): AutonomyState | null {
  const want = id.trim().toLowerCase();
  return (
    rows?.find(
      (row) =>
        row.autonomy_id.toLowerCase() === want && row.source !== "imported",
    ) ?? null
  );
}

export function AutonomyEditPanel({
  projectId,
  level,
  onSaved,
  onClose,
}: {
  projectId: number;
  level: GuiAutonomyLevel;

  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const tw = useTranslations("gameViewWorld.autonomy");
  const tPanel = useTranslations("gameViewEdit.panel");

  const rows = useSWR(
    ["game-view-autonomy-states-db", projectId],
    async () => (await autonomyStatesApi.list(projectId)).autonomy_states,
    { revalidateOnFocus: false },
  );
  const row = projectAutonomyRow(rows.data, level.id);

  const projectReady = useWorkbenchProject(projectId);
  const workbenchHref = `/project/${projectId}/autonomy-states`;

  const saveRow = async (data: Partial<AutonomyState>) => {
    if (!row) return;
    const id = row.id;
    const keys = changedKeys(row, data);
    const before = pick(row, keys);
    const saved = await autonomyStatesApi.update(projectId, id, data);
    recordChange({
      projectId,
      title: level.label,
      kind: tw("kind", { id: saved.autonomy_id }),
      workbenchHref,
      undo: keys.length
        ? rowUndo<AutonomyState>({
            before,
            after: pick(saved, keys),
            load: () => autonomyStatesApi.get(projectId, id),
            save: (back) => autonomyStatesApi.update(projectId, id, back),
          })
        : undefined,
    });
    await rows.mutate();
    onSaved(tw("saved", { name: level.label }));
  };

  return (
    <GamePanel
      title={level.label}
      kind={tw("kind", { id: level.id })}
      detail={level.file}
      source={row ? "项目" : undefined}
      workbenchHref={row ? workbenchHref : null}

      wide={row ? "workbench" : false}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">{tw("loading")}</p>
      ) : row ? (
        projectReady ? (
          <div className="-mx-4 mt-2">
            <AutonomyStateEditor key={row.id} state={row} onSave={saveRow} />
          </div>
        ) : (
          <p className="mt-4 text-micro text-muted-foreground">
            {tPanel("loadingProject")}
          </p>
        )
      ) : (
        <LevelFields projectId={projectId} level={level} onSaved={onSaved} />
      )}
    </GamePanel>
  );
}

function LevelFields({
  projectId,
  level,
  onSaved,
}: {
  projectId: number;
  level: GuiAutonomyLevel;
  onSaved: (message: string) => void;
}) {
  const tw = useTranslations("gameViewWorld.autonomy");
  const [draft, setDraft] = useState<AutonomyDraft>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const changes = autonomyChanges(level, draft);
  const changed = !!changes && Object.keys(changes).length > 0;
  const save = async () => {
    if (!changes || !changed) return;
    setBusy(true);
    setError(null);
    try {
      const { name, ...definition } = changes;
      let answer: unknown = null;
      if (Object.keys(definition).length) {
        const response = await fetch(
          `/api/go/projects/${projectId}/gui-render/autonomy-save`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ id: level.id, ...definition }),
          },
        );
        const out = await response.json().catch(() => ({}));
        if (!response.ok)
          throw new Error(
            (out as { detail?: string }).detail || `HTTP ${response.status}`,
          );
        answer = out;
      }

      const ways = answer !== undefined ? [fileUndo(projectId, answer)] : [];
      if (name !== undefined) {
        guiRenderApi.setProjectId(projectId);
        const before = await textBefore(level.id);
        await guiRenderApi.setText(level.id, name);
        ways.push(textUndo(level.id, name, before));
      }
      recordChange({
        projectId,
        title: name ?? level.label,
        kind: tw("kind", { id: level.id }),
        undo: undoAll(...ways),
      });
      setDraft({});
      onSaved(tw("saved", { name: level.label }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };
  const numberField = (
    key: "min_freedom_level" | "manpower_influence",
    label: string,
    current: number | undefined,
    hint?: string,
  ) => (
    <label className="block text-xs">
      {label}
      <input
        aria-label={key}
        className={NUMBER}
        inputMode="decimal"
        placeholder={hint}
        value={draft[key] ?? (current === undefined ? "" : String(current))}
        onChange={(e) => setDraft({ ...draft, [key]: e.target.value })}
      />
    </label>
  );

  return (
    <>
      <div className="mt-3 space-y-2">
        <label className="block text-xs">
          {tw("name")}
          <input
            aria-label="name"
            className={FIELD}
            value={draft.name ?? level.label}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
        </label>
        {numberField(
          "min_freedom_level",
          tw("minFreedom"),
          level.min_freedom_level,
        )}
        {numberField(
          "manpower_influence",
          tw("manpower"),
          level.manpower_influence,
          tw("unset"),
        )}
        <label className="block text-xs">
          {tw("rule")}
          <textarea
            aria-label="rule"
            className={SCRIPT}
            rows={5}
            value={draft.rule ?? level.rule}
            onChange={(e) => setDraft({ ...draft, rule: e.target.value })}
          />
        </label>
        <label className="block text-xs">
          {tw("modifier")}
          <textarea
            aria-label="modifier"
            className={SCRIPT}
            rows={5}
            value={draft.modifier ?? level.modifier}
            onChange={(e) => setDraft({ ...draft, modifier: e.target.value })}
          />
        </label>
        <p className="text-micro text-muted-foreground">{tw("scriptNote")}</p>
      </div>
      <PanelSave
        onSave={save}
        canSave={changed}
        busy={busy}
        onRevert={changed ? () => setDraft({}) : undefined}
        problem={changes === null ? tw("problem") : null}
        note={tw("note")}
        error={error}
      />
    </>
  );
}
