"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { guiRenderApi } from "@/lib/guiRenderApi";
import { specialProjectsApi } from "@/lib/api/military";
import type { SpecialProject } from "@/types";
import { SpecialProjectEditor } from "@/components/workbench/special-projects/SpecialProjectEditor";
import type { GuiSpecialProject, GuiSpecialization } from "./FacilitiesLayer";
import { FIELD, GamePanel, NUMBER, PanelSave, SCRIPT } from "./GamePanel";
import type { GuiProjectPage } from "./ProjectPageLayer";
import {
  fileUndo,
  pick,
  recordChange,
  rowUndo,
  textBefore,
  textUndo,
  undoAll,
} from "./changeLog";
import { changedKeys } from "./workbenchProject";

export const PROTOTYPE_TIMES = [
  { value: "sp_time.prototype.short", labelKey: "prototype.short" },
  { value: "sp_time.prototype.medium", labelKey: "prototype.medium" },
  { value: "sp_time.prototype.long", labelKey: "prototype.long" },
  { value: "sp_time.prototype.very_long", labelKey: "prototype.veryLong" },
] as const;
export const COMPLEXITIES = [
  { value: "sp_complexity.small", labelKey: "complexityOption.small" },
  { value: "sp_complexity.medium", labelKey: "complexityOption.medium" },
  { value: "sp_complexity.large", labelKey: "complexityOption.large" },
  { value: "sp_complexity.insane", labelKey: "complexityOption.insane" },
] as const;
type SelectOption =
  (typeof PROTOTYPE_TIMES)[number] | (typeof COMPLEXITIES)[number];

export interface SpecialProjectDraft {
  name?: string;
  cost?: Record<string, string>;
  prototype_time?: string;
  complexity?: string;
  allowed?: string;
  available?: string;
  description?: string;

  bonus?: Record<string, { uses?: string; off?: string }>;
}

type DesignBonus = { name: string; uses: number; cost_factor: number };

type Changes = {
  name?: string;
  breakthrough_cost?: Record<string, number>;
  prototype_time?: string;
  complexity?: string;
  allowed?: string;
  available?: string;
  description?: string;
  design_bonus?: DesignBonus;
};

function costOf(project: GuiSpecialProject): Record<string, number> {
  const out: Record<string, number> = {};
  for (const item of project.breakthrough_cost)
    out[item.specialization] = item.amount;
  return out;
}

export function specialProjectChanges(
  project: GuiSpecialProject,
  draft: SpecialProjectDraft,
  page?: Pick<GuiProjectPage, "description" | "discounts"> | null,
): Changes | null {
  const out: Changes = {};
  if (draft.name !== undefined) {
    const name = draft.name.trim();
    if (!name) return null;
    if (name !== project.label) out.name = name;
  }
  if (draft.cost) {
    const current = costOf(project);
    const next: Record<string, number> = { ...current };
    for (const [spec, text] of Object.entries(draft.cost)) {
      const value = text.trim() === "" ? 0 : Number(text);
      if (!Number.isInteger(value) || value < 0 || value > 100) return null;
      if (value) next[spec] = value;
      else delete next[spec];
    }
    const same =
      Object.keys(next).length === Object.keys(current).length &&
      Object.entries(next).every(([spec, value]) => current[spec] === value);
    if (!same) out.breakthrough_cost = next;
  }
  if (
    draft.prototype_time !== undefined &&
    draft.prototype_time !== project.prototype_time
  )
    out.prototype_time = draft.prototype_time;
  if (draft.complexity !== undefined && draft.complexity !== project.complexity)
    out.complexity = draft.complexity;
  if (
    draft.allowed !== undefined &&
    draft.allowed.trim() !== project.allowed_script
  )
    out.allowed = draft.allowed.trim();
  if (
    draft.available !== undefined &&
    draft.available.trim() !== project.available_script
  )
    out.available = draft.available.trim();
  if (page && draft.description !== undefined) {
    const description = draft.description.trim();
    if (!description) return null;
    if (description !== page.description) out.description = description;
  }
  for (const bonus of page?.discounts ?? []) {
    const edit = draft.bonus?.[bonus.name];
    if (!edit) continue;
    const uses = edit.uses === undefined ? bonus.uses : Number(edit.uses);
    const off =
      edit.off === undefined ? bonus.cost_factor * 100 : Number(edit.off);
    if (
      !Number.isInteger(uses) ||
      uses < 1 ||
      uses > 100 ||
      !Number.isFinite(off) ||
      off < 0 ||
      off > 100 ||
      (edit.off ?? "x").trim() === ""
    )
      return null;
    const factor = Math.round(off * 100) / 10000;
    if (uses !== bonus.uses || factor !== bonus.cost_factor)
      out.design_bonus = { name: bonus.name, uses, cost_factor: factor };
  }
  return out;
}

export function findSpecialProjectRow(
  rows: readonly SpecialProject[] | undefined,
  id: string,
): SpecialProject | null {
  const want = id.toLowerCase();
  return (
    rows?.find(
      (row) =>
        row.project_token.toLowerCase() === want && row.source !== "imported",
    ) ?? null
  );
}

const loadRow = async (projectId: number, id: number) =>
  (await specialProjectsApi.listProjects(projectId)).find(
    (row) => row.id === id,
  ) ?? null;

export function SpecialProjectEditPanel({
  projectId,
  project,
  specializations,
  page = null,
  onSaved,
  onClose,
}: {
  projectId: number;
  project: GuiSpecialProject;
  specializations: GuiSpecialization[];

  page?: GuiProjectPage | null;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewResearch.specialProject");
  const rows = useSWR(
    ["game-view-special-projects-db", projectId],
    () => specialProjectsApi.listProjects(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  const row = findSpecialProjectRow(rows.data, project.id);
  return (
    <GamePanel
      title={project.label}
      kind={t("kind", { id: project.id })}
      detail={project.file}
      source={row ? "项目" : undefined}
      workbenchHref={row ? `/project/${projectId}/special-projects` : null}

      wide={row ? "workbench" : false}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">{t("loading")}</p>
      ) : row ? (
        <WorkbenchFields
          key={row.id}
          projectId={projectId}
          project={project}
          row={row}
          specializations={specializations}
          page={page}
          reload={() => rows.mutate()}
          onSaved={onSaved}
        />
      ) : (
        <GameFields
          projectId={projectId}
          project={project}
          specializations={specializations}
          page={page}
          onSaved={onSaved}
        />
      )}
    </GamePanel>
  );
}

function WorkbenchFields({
  projectId,
  project,
  row,
  specializations,
  page,
  reload,
  onSaved,
}: {
  projectId: number;
  project: GuiSpecialProject;
  row: SpecialProject;
  specializations: GuiSpecialization[];
  page: GuiProjectPage | null;
  reload: () => Promise<SpecialProject[] | undefined>;
  onSaved: (message: string) => void;
}) {
  const t = useTranslations("gameViewResearch.specialProject");
  const tPanel = useTranslations("gameViewEdit.panel");

  const specs = useSWR(
    ["game-view-special-project-specs-db", projectId],
    () => specialProjectsApi.listSpecializations(projectId),
    {
      revalidateOnFocus: false,
    },
  );

  const save = async (data: Partial<SpecialProject>) => {
    const id = row.id;
    const keys = changedKeys(row, data);
    const before = pick(row, keys);
    const saved = await specialProjectsApi.updateProject(projectId, id, data);
    recordChange({
      projectId,
      title: project.label,
      kind: t("kind", { id: project.id }),
      workbenchHref: `/project/${projectId}/special-projects`,
      undo: keys.length
        ? rowUndo({
            before,
            after: pick(saved, keys),
            load: () => loadRow(projectId, id),
            save: (back) =>
              specialProjectsApi.updateProject(projectId, id, back),
          })
        : undefined,
    });
    await reload();
    onSaved(t("saved", { name: project.label }));
  };
  if (specs.isLoading)
    return (
      <p className="mt-4 text-micro text-muted-foreground">
        {tPanel("loadingProject")}
      </p>
    );
  return (
    <>
      <div className="-mx-4 mt-2">
        <SpecialProjectEditor
          project={row}
          specs={specs.data ?? []}
          onSave={save}
        />
      </div>
      <section className="mt-4 border-t border-border pt-3">
        <h4 className="text-micro font-semibold uppercase tracking-widest text-muted-foreground">
          {t("gameShown")}
        </h4>
        <p className="mt-1 text-micro text-muted-foreground">
          {t("gameShownNote")}
        </p>
        <GameFields
          projectId={projectId}
          project={project}
          specializations={specializations}
          page={page}
          row={row}
          reload={reload}
          onSaved={onSaved}
        />
      </section>
    </>
  );
}

function GameFields({
  projectId,
  project,
  specializations,
  page,
  row = null,
  reload,
  onSaved,
}: {
  projectId: number;
  project: GuiSpecialProject;
  specializations: GuiSpecialization[];
  page: GuiProjectPage | null;

  row?: SpecialProject | null;
  reload?: () => Promise<SpecialProject[] | undefined>;
  onSaved: (message: string) => void;
}) {
  const t = useTranslations("gameViewResearch.specialProject");
  const tCommon = useTranslations("gameViewResearch.common");
  const [draft, setDraft] = useState<SpecialProjectDraft>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const changes = specialProjectChanges(project, draft, page);
  const changed = !!changes && Object.keys(changes).length > 0;
  const cost = costOf(project);
  const save = async () => {
    if (!changes || !changed) return;
    setBusy(true);
    setError(null);
    try {
      const { name, description, ...definition } = changes;

      const ways: ((() => Promise<void>) | undefined)[] = [];
      if (Object.keys(definition).length) {
        const response = await fetch(
          `/api/go/projects/${projectId}/gui-render/special-project-save`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ id: project.id, ...definition }),
          },
        );
        const out = await response.json().catch(() => ({}));
        if (!response.ok)
          throw new Error(
            (out as { detail?: string }).detail || `HTTP ${response.status}`,
          );
        if (row) {
          const id = row.id;
          const after = (await reload?.())?.find((r) => r.id === id);
          ways.push(
            after
              ? rowUndo<SpecialProject>({
                  before: pick(row, ["raw_extras"]),
                  after: pick(after, ["raw_extras"]),
                  load: () => loadRow(projectId, id),
                  save: (back) =>
                    specialProjectsApi.updateProject(projectId, id, back),
                })
              : undefined,
          );
        } else {
          ways.push(fileUndo(projectId, out));
        }
      }
      if (name !== undefined) {
        guiRenderApi.setProjectId(projectId);
        const before = await textBefore(project.label_key);
        await guiRenderApi.setText(project.label_key, name);
        ways.push(textUndo(project.label_key, name, before));
      }
      if (description !== undefined && page) {
        guiRenderApi.setProjectId(projectId);
        const before = await textBefore(page.description_key);
        await guiRenderApi.setText(page.description_key, description);
        ways.push(textUndo(page.description_key, description, before));
      }
      recordChange({
        projectId,
        title: name ?? project.label,
        kind: t("kind", { id: project.id }),
        workbenchHref: row
          ? `/project/${projectId}/special-projects`
          : undefined,
        undo: undoAll(...ways),
      });
      setDraft({});
      onSaved(t("saved", { name: name ?? project.label }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };
  const select = (
    key: "prototype_time" | "complexity",
    label: string,
    options: readonly SelectOption[],
  ) => {
    const value = draft[key] ?? project[key];
    return (
      <label className="block text-xs">
        {label}
        <select
          aria-label={key}
          className={FIELD}
          value={value}
          onChange={(e) => setDraft({ ...draft, [key]: e.target.value })}
        >
          {!options.some((option) => option.value === value) && (
            <option value={value}>{value || t("unwritten")}</option>
          )}
          {options.map((option) => (
            <option key={option.value} value={option.value}>
              {t(option.labelKey)}
            </option>
          ))}
        </select>
      </label>
    );
  };

  return (
    <>
      <div className="mt-3 space-y-2">
        <label className="block text-xs">
          {project.label_key !== project.id
            ? t("nameLocal", { key: project.label_key })
            : tCommon("name")}
          <input
            aria-label="name"
            className={FIELD}
            value={draft.name ?? project.label}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
        </label>
        {page && (
          <label className="block text-xs">
            {t("description")}
            <textarea
              aria-label="description"
              className={FIELD}
              rows={4}
              value={draft.description ?? page.description}
              onChange={(e) =>
                setDraft({ ...draft, description: e.target.value })
              }
            />
          </label>
        )}

        {!row &&
          page?.discounts.map((bonus) => (
            <div key={bonus.name} className="text-xs">
              {t("discount", { label: bonus.label })}
              <div className="mt-1 grid grid-cols-2 gap-2">
                <label className="block text-micro text-muted-foreground">
                  {t("discountUses")}
                  <input
                    aria-label={`uses ${bonus.name}`}
                    className={NUMBER}
                    inputMode="numeric"
                    value={
                      draft.bonus?.[bonus.name]?.uses ?? String(bonus.uses)
                    }
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        bonus: {
                          ...draft.bonus,
                          [bonus.name]: {
                            ...draft.bonus?.[bonus.name],
                            uses: e.target.value,
                          },
                        },
                      })
                    }
                  />
                </label>
                <label className="block text-micro text-muted-foreground">
                  {t("discountOff")}
                  <input
                    aria-label={`off ${bonus.name}`}
                    className={NUMBER}
                    inputMode="decimal"
                    value={
                      draft.bonus?.[bonus.name]?.off ??
                      String(Math.round(bonus.cost_factor * 10000) / 100)
                    }
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        bonus: {
                          ...draft.bonus,
                          [bonus.name]: {
                            ...draft.bonus?.[bonus.name],
                            off: e.target.value,
                          },
                        },
                      })
                    }
                  />
                </label>
              </div>
            </div>
          ))}
        <div className="text-xs">
          {t("cost")}
          <div className="mt-1 grid grid-cols-2 gap-2">
            {specializations.map((spec) => (
              <label
                key={spec.id}
                className="block text-micro text-muted-foreground"
              >
                {spec.label}
                <input
                  aria-label={`cost ${spec.id}`}
                  className={NUMBER}
                  inputMode="numeric"
                  value={
                    draft.cost?.[spec.id] ??
                    (cost[spec.id] ? String(cost[spec.id]) : "")
                  }
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      cost: { ...draft.cost, [spec.id]: e.target.value },
                    })
                  }
                />
              </label>
            ))}
          </div>
        </div>

        {!row && (
          <>
            {select("prototype_time", t("prototypeTime"), PROTOTYPE_TIMES)}
            {select("complexity", t("complexity"), COMPLEXITIES)}
            <label className="block text-xs">
              {t("allowed")}
              <textarea
                aria-label="allowed"
                className={SCRIPT}
                rows={3}
                placeholder={t("allowedPlaceholder")}
                value={draft.allowed ?? project.allowed_script}
                onChange={(e) =>
                  setDraft({ ...draft, allowed: e.target.value })
                }
              />
            </label>
            <label className="block text-xs">
              {t("available")}
              <textarea
                aria-label="available"
                className={SCRIPT}
                rows={4}
                placeholder={tCommon("anytime")}
                value={draft.available ?? project.available_script}
                onChange={(e) =>
                  setDraft({ ...draft, available: e.target.value })
                }
              />
            </label>
            <p className="text-micro text-muted-foreground">
              {t("conditionsNote")}
            </p>
          </>
        )}
      </div>
      <PanelSave
        onSave={save}
        canSave={changed}
        busy={busy}
        onRevert={changed ? () => setDraft({}) : undefined}
        problem={
          changes === null ? (row ? t("problemWorkbench") : t("problem")) : null
        }
        note={row ? t("noteWorkbench") : t("note")}
        error={error}
      />
    </>
  );
}
