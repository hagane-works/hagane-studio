"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import type { Technology } from "@/types";
import { technologiesApi } from "@/lib/api/focus-tech";
import { historyFilesApi } from "@/lib/api/scenario";
import type { GuiTechnology, TechChange } from "./TechTreeLayer";
import { BUTTON, FIELD, GamePanel, PRIMARY } from "./GamePanel";
import { pick, recordChange, rowUndo } from "./changeLog";
import { BaseTextEdit } from "./BaseTextEdit";
import { useHistoryRow } from "./startStateEdit";
import {
  findTechRow,
  researchDays,
  startToggleReaches,
  techDraft,
  withResearched,
} from "./techEdit";
import {
  TECH_YEAR_ROWS,
  newTechDraft,
  newTechProblem,
  newTechRow,
  shownNewTech,
  techToken,
  withPathTo,
  withoutReferences,
  type NewTechDraft,
} from "./techCreate";
import { TechEditor } from "@/components/workbench/TechEditor";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

const loadTechRow = async (projectId: number, id: number) =>
  (await technologiesApi.list(projectId)).find((row) => row.id === id) ?? null;

export function TechEditPanel({
  projectId,
  tag,
  tech,
  baseTechCost,
  onSaved,
  onClose,
  onCreated,
  onDeleted,
}: {
  projectId: number;
  tag: string;

  tech: GuiTechnology;

  baseTechCost?: number;
  onSaved: (name: string, change: TechChange) => void;
  onClose: () => void;

  onCreated?: (tech: GuiTechnology) => void;

  onDeleted?: (name: string) => void;
}) {
  const t = useTranslations("gameViewResearch.tech");
  const tPanel = useTranslations("gameViewEdit.panel");
  const rows = useSWR(
    ["game-view-technologies-db", projectId],
    () => technologiesApi.list(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  const row = findTechRow(rows.data, tech.name, tag);
  const projectReady = useWorkbenchProject(projectId);

  const cardFollows = tech.card && tech.card_label_key === tech.name;

  const save = async (data: Partial<Technology>) => {
    if (!row) return;
    const keys = changedKeys(row, data);
    const id = row.id;
    const before = pick(row, keys);
    const saved = await technologiesApi.update(projectId, id, data);
    const name = techDraft(saved, { label: tech.label, cost: tech.cost }).name;
    recordChange({
      projectId,
      title: (cardFollows ? name : tech.card_label) || name || tech.name,
      kind: t("kind", { id: tech.name }),
      workbenchHref: `/project/${projectId}/technology?tech=${id}`,
      undo: keys.length
        ? rowUndo({
            before,
            after: pick(saved, keys),
            load: () => loadTechRow(projectId, id),
            save: (back) => technologiesApi.update(projectId, id, back),
          })
        : undefined,
    });
    onSaved(tech.name, {
      label: name,
      cost: saved.research_cost,
      ...(cardFollows ? { card_label: name } : {}),
    });
    await rows.mutate();
  };
  const days = row ? researchDays(row.research_cost, baseTechCost) : null;
  return (
    <GamePanel
      title={tech.card_label || tech.label || tech.name}
      kind={t("kind", { id: tech.name })}
      source={rows.isLoading ? undefined : row ? "项目" : "原版"}
      workbenchHref={
        row ? `/project/${projectId}/technology?tech=${row.id}` : null
      }

      wide={row ? "workbench" : true}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">{t("loading")}</p>
      ) : row ? (
        projectReady && rows.data ? (
          <>
            {tech.card && !cardFollows && (
              <p className="mt-3 text-micro text-muted-foreground">
                {t.rich("cardLabelNote", {
                  label: tech.card_label ?? "",
                  source: tech.card_label_key ?? "",
                  mono: (chunks) => <span className="font-mono">{chunks}</span>,
                })}
              </p>
            )}
            {days !== null && (
              <p className="mt-1 text-micro text-muted-foreground">
                {t("days", {
                  days: String(days),
                  base: String(baseTechCost),
                  cost: String(row.research_cost),
                })}
              </p>
            )}

            <div className="-mx-4 mt-2">
              <TechEditor
                key={row.id}
                tech={row}
                allTechs={rows.data}
                projectId={projectId}
                layoutMode={false}
                onSave={save}
                onIconUploaded={() => {
                  void rows.mutate();
                  onSaved(tech.name, {});
                }}
              />
            </div>
          </>
        ) : (
          <p className="mt-4 text-micro text-muted-foreground">
            {tPanel("loadingProject")}
          </p>
        )
      ) : (
        <BaseTextEdit
          projectId={projectId}
          title={tech.label || tech.name}
          fields={[
            { field: "name", key: tech.name },
            ...(tech.card_label_key && tech.card_label_key !== tech.name
              ? [{ field: "cardLabel" as const, key: tech.card_label_key }]
              : []),
            { field: "description" as const, key: `${tech.name}_desc` },
          ]}
          onSaved={(key, value) =>
            onSaved(
              tech.name,
              key === tech.name
                ? { label: value ?? undefined }
                : key === tech.card_label_key
                  ? { card_label: value ?? undefined }
                  : {},
            )
          }
        />
      )}
      <StartResearched
        projectId={projectId}
        tag={tag}
        tech={tech}
        onSaved={onSaved}
      />
      {row && rows.data && onCreated && (
        <NewTechAfter
          projectId={projectId}
          parent={tech}
          parentRow={row}
          rows={rows.data}
          reload={() => rows.mutate()}
          onCreated={onCreated}
        />
      )}
      {row &&
        rows.data &&
        onDeleted &&
        tech.added_by_project &&
        row.source === "created" && (
          <DeleteTech
            projectId={projectId}
            row={row}
            rows={rows.data}
            onDeleted={onDeleted}
          />
        )}
    </GamePanel>
  );
}

function StartResearched({
  projectId,
  tag,
  tech,
  onSaved,
}: {
  projectId: number;
  tag: string;
  tech: GuiTechnology;
  onSaved: (name: string, change: TechChange) => void;
}) {
  const t = useTranslations("gameViewResearch.tech");
  const tPanel = useTranslations("gameViewEdit.panel");
  const history = useHistoryRow(projectId, tag, true);

  const [pending, setPending] = useState<boolean | null>(null);
  const [error, setError] = useState<string | null>(null);
  const row = history.data;
  const ids = row?.set_technology_ids ?? [];
  const reaches = startToggleReaches(ids, tech.name, tech.researched);

  const toggle = async (researched: boolean) => {
    if (!row) return;
    setPending(researched);
    setError(null);
    try {
      const id = row.id;
      const keys = ["set_technology_ids"];
      const before = pick(row, keys);
      const saved = await historyFilesApi.update(projectId, id, {
        set_technology_ids: withResearched(ids, tech.name, researched),
      });
      recordChange({
        projectId,
        title: tech.card_label || tech.label || tech.name,
        kind: t("kind", { id: tech.name }),

        workbenchHref: `/project/${projectId}/history-files?hf=${id}`,
        undo: rowUndo({
          before,
          after: pick(saved, keys),
          load: () => historyFilesApi.get(projectId, id),
          save: (back) => historyFilesApi.update(projectId, id, back),
        }),
      });
      await history.mutate();

      onSaved(tech.name, {
        researched,
        state: researched ? "researched" : "available",
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setPending(null);
    }
  };

  return (
    <section className="mt-4 space-y-2 border-t border-border pt-3">
      <div className="text-micro font-medium text-muted-foreground">
        {t("start")}
      </div>
      {history.isLoading ? (
        <p className="text-micro text-muted-foreground">
          {t("loadingHistory", { tag })}
        </p>
      ) : !row ? (
        <p className="text-micro text-muted-foreground">
          {tPanel("notInProjectHistory")}
        </p>
      ) : (
        <>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              className="h-3.5 w-3.5"
              checked={pending ?? tech.researched}
              disabled={pending !== null || !reaches}
              onChange={(e) => void toggle(e.target.checked)}
            />
            {t("researchedAtStart", { tag })}
          </label>
          {!reaches && (
            <p className="text-micro text-muted-foreground">
              {t("otherBlock")}
            </p>
          )}
          <p className="text-micro text-muted-foreground">
            {t("startNote", { tag })}
          </p>
        </>
      )}
      {error && <p className="text-micro text-destructive">{error}</p>}
    </section>
  );
}

function NewTechAfter({
  projectId,
  parent,
  parentRow,
  rows,
  reload,
  onCreated,
}: {
  projectId: number;
  parent: GuiTechnology;
  parentRow: Technology;
  rows: readonly Technology[];
  reload: () => Promise<unknown>;
  onCreated: (tech: GuiTechnology) => void;
}) {
  const t = useTranslations("gameViewResearch.tech");
  const [draft, setDraft] = useState<NewTechDraft>(() => newTechDraft(parent));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const taken = useMemo(() => new Set(rows.map((r) => r.tech_id)), [rows]);
  const problem = newTechProblem(draft, taken);

  const create = async () => {
    if (problem || busy) return;
    setBusy(true);
    setError(null);
    try {
      const created = await technologiesApi.create(
        projectId,
        newTechRow(parent, parentRow, draft),
      );

      recordChange({
        projectId,
        title: shownNewTech(created).label || created.tech_id,
        kind: t("kind", { id: created.tech_id }),
        workbenchHref: `/project/${projectId}/technology?tech=${created.id}`,
      });

      await technologiesApi.update(projectId, parentRow.id, {
        path: withPathTo(parentRow.path, created.tech_id),
      });
      await reload();
      onCreated(shownNewTech(created));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <details className="mt-4 rounded border border-border p-2">
      <summary className="cursor-pointer text-micro font-medium text-muted-foreground">
        {t("newAfter")}
      </summary>
      <div className="mt-2 space-y-2">
        <label className="block">
          <span className="text-micro font-medium text-muted-foreground">
            {t("newName")}
          </span>
          <input
            value={draft.name}
            disabled={busy}
            placeholder="Improved Rifle Drill"
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
            className={FIELD}
          />
        </label>
        {draft.name.trim() && (
          <p className="font-mono text-micro text-muted-foreground">
            {techToken(draft.name)}
          </p>
        )}
        <div className="flex gap-2">
          <label className="block flex-1">
            <span className="text-micro font-medium text-muted-foreground">
              {t("year")}
            </span>
            <select
              aria-label={t("yearLabel")}
              value={draft.year}
              disabled={busy}
              onChange={(e) =>
                setDraft({ ...draft, year: Number(e.target.value) })
              }
              className={FIELD}
            >
              {TECH_YEAR_ROWS.map((column) => (
                <option key={column.year} value={column.year}>
                  {column.year}
                </option>
              ))}
            </select>
          </label>
          <label className="block w-24">
            <span className="text-micro font-medium text-muted-foreground">
              {t("x")}
            </span>
            <input
              type="number"
              min={0}
              max={20}
              step={1}
              value={draft.x}
              disabled={busy}
              onChange={(e) =>
                setDraft({ ...draft, x: Number(e.target.value) })
              }
              className={FIELD}
            />
          </label>
          <label className="block w-24">
            <span className="text-micro font-medium text-muted-foreground">
              {t("cost")}
            </span>
            <input
              type="number"
              min={0}
              step={0.05}
              value={draft.cost}
              disabled={busy}
              onChange={(e) =>
                setDraft({ ...draft, cost: Number(e.target.value) })
              }
              className={FIELD}
            />
          </label>
        </div>
        {problem && draft.name.trim() && (
          <p className="text-micro text-destructive">
            {t(`problem.${problem.key}`, problem.values)}
          </p>
        )}
        <button
          type="button"
          disabled={busy || !!problem}
          onClick={() => void create()}
          className={PRIMARY}
        >
          {t("create")}
        </button>
        <p className="text-micro text-muted-foreground">{t("newNote")}</p>
        {error && <p className="text-micro text-destructive">{error}</p>}
      </div>
    </details>
  );
}

function DeleteTech({
  projectId,
  row,
  rows,
  onDeleted,
}: {
  projectId: number;
  row: Technology;
  rows: readonly Technology[];
  onDeleted: (name: string) => void;
}) {
  const t = useTranslations("gameViewResearch.tech");
  const tCommon = useTranslations("gameViewResearch.common");
  const [confirm, setConfirm] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const remove = async () => {
    setBusy(true);
    setError(null);
    try {
      for (const change of withoutReferences(rows, row.tech_id)) {
        await technologiesApi.update(projectId, change.id, change.patch);
      }
      await technologiesApi.delete(projectId, row.id);

      recordChange({
        projectId,
        title: row.display_name_locs?.en || row.name || row.tech_id,
        kind: t("kind", { id: row.tech_id }),
        workbenchHref: `/project/${projectId}/technology`,
      });
      onDeleted(row.tech_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  };

  return (
    <section className="mt-4 border-t border-border pt-3">
      {confirm ? (
        <div className="flex items-center gap-2">
          <span className="text-micro text-destructive">
            {t("confirmDelete")}
          </span>
          <button
            type="button"
            disabled={busy}
            onClick={() => void remove()}
            className="rounded bg-destructive px-2 py-0.5 text-micro text-destructive-foreground disabled:opacity-50"
          >
            {tCommon("delete")}
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={() => setConfirm(false)}
            className={BUTTON}
          >
            {tCommon("cancel")}
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => setConfirm(true)}
          className={`${BUTTON} text-destructive`}
        >
          {t("deleteTech")}
        </button>
      )}
      {error && <p className="mt-2 text-micro text-destructive">{error}</p>}
    </section>
  );
}
