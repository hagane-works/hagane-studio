"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { readGuiResponse } from "./guiResponse";
import { equipmentApi } from "@/lib/api";
import { guiRenderApi, withGameLanguage } from "@/lib/guiRenderApi";
import type { Equipment } from "@/types";
import { EquipmentEditor } from "@/app/[locale]/project/[id]/equipment/EquipmentEditor";
import { BUTTON, FIELD, GamePanel, PanelSave } from "./GamePanel";
import {
  fileUndo,
  pick,
  recordChange,
  rowUndo,
  textBefore,
  textUndo,
  undoAll,
} from "./changeLog";
import {
  STAT_ORDER,
  equipmentDraft,
  equipmentPatch,
  equipmentProblem,
  newEquipmentProblem,
  projectEquipmentRows,
  resourceKeys,
  statKeys,
  type EquipmentDraft,
  type GameEquipment,
} from "./equipmentEdit";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

const SMALL =
  "w-20 rounded border border-input bg-background px-1 py-0.5 text-right text-sm tabular-nums";

async function fetchEquipment(
  projectId: number,
  tag: string | null,
  name: string,
): Promise<GameEquipment> {
  const query = new URLSearchParams({ name });
  if (tag) query.set("tag", tag);
  const response = await fetch(
    withGameLanguage(
      `/api/go/projects/${projectId}/gui-render/equipment?${query}`,
    ),
  );
  return readGuiResponse<GameEquipment>(response);
}

async function postJSON(url: string, body: unknown) {
  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const out = await response.json().catch(() => ({}));
  if (!response.ok)
    throw new Error(
      (out as { detail?: string }).detail || `HTTP ${response.status}`,
    );
  return out;
}

function useStatLabel() {
  const t = useTranslations("gameViewMilitary.stat");
  return (key: string) => (STAT_ORDER.includes(key) ? t(key) : key);
}

async function writeText(projectId: number, key: string, text: string) {
  guiRenderApi.setProjectId(projectId);
  const before = await textBefore(key);
  await guiRenderApi.setText(key, text);
  return textUndo(key, text, before);
}

export function EquipmentEditPanel({
  projectId,
  tag,
  name,
  onSaved,
  onClose,
}: {
  projectId: number;

  tag: string | null;

  name: string;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewMilitary.equipment");
  const tPanel = useTranslations("gameViewEdit.panel");
  const game = useSWR(
    ["game-view-equipment", projectId, tag ?? "", name],
    () => fetchEquipment(projectId, tag, name),
    {
      revalidateOnFocus: false,
    },
  );
  const rows = useSWR(
    ["game-view-equipment-rows", projectId],
    () => equipmentApi.list(projectId),
    { revalidateOnFocus: false },
  );
  const { own: row, imported } = projectEquipmentRows(rows.data, name);
  const projectReady = useWorkbenchProject(projectId);
  const title = game.data?.country_label || game.data?.label || name;
  const kind = game.data?.archetype
    ? t("kindArchetype", {
        name,
        archetype: game.data.archetype_label ?? game.data.archetype,
      })
    : t("kind", { name });
  const workbenchHref = row
    ? `/project/${projectId}/equipment?equipment=${row.id}`
    : null;
  const reloadGame = () => game.mutate();
  const reloadRows = () => rows.mutate();

  return (
    <GamePanel
      title={title}
      kind={kind}
      source={
        game.isLoading || rows.isLoading
          ? undefined
          : row
            ? "项目"
            : imported
              ? "mod 文件"
              : game.data?.found
                ? "原版"
                : undefined
      }
      workbenchHref={workbenchHref}

      wide={row ? "workbench" : true}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("loadingRows")}
        </p>
      ) : row ? (
        projectReady ? (
          <ProjectEquipment
            key={row.id}
            projectId={projectId}
            tag={tag}
            name={name}
            row={row}
            rows={rows.data ?? []}
            game={game.data}
            title={title}
            kind={kind}
            workbenchHref={workbenchHref}
            reloadGame={reloadGame}
            reloadRows={reloadRows}
            onSaved={onSaved}
          />
        ) : (
          <p className="mt-4 text-micro text-muted-foreground">
            {tPanel("loadingProject")}
          </p>
        )
      ) : (
        <GameEquipmentForm
          projectId={projectId}
          tag={tag}
          name={name}
          game={game.data}
          loading={game.isLoading}
          rows={rows.data}
          kind={kind}
          reloadGame={reloadGame}
          reloadRows={reloadRows}
          onSaved={onSaved}
        />
      )}
    </GamePanel>
  );
}

function ProjectEquipment({
  projectId,
  tag,
  name,
  row,
  rows,
  game,
  title,
  kind,
  workbenchHref,
  reloadGame,
  reloadRows,
  onSaved,
}: {
  projectId: number;
  tag: string | null;
  name: string;
  row: Equipment;
  rows: readonly Equipment[];
  game: GameEquipment | undefined;
  title: string;
  kind: string;
  workbenchHref: string | null;
  reloadGame: () => Promise<unknown>;
  reloadRows: () => Promise<unknown>;
  onSaved: (message: string) => void;
}) {
  const t = useTranslations("gameViewMilitary.equipment");

  const save = async (update: Partial<Equipment>) => {
    const id = row.id;
    const keys = changedKeys(row, update);
    const before = pick(row, keys);
    const saved = await equipmentApi.update(projectId, id, update);
    recordChange({
      projectId,
      title,
      kind,
      workbenchHref,
      undo: keys.length
        ? rowUndo({
            before,
            after: pick(saved, keys),
            load: () => equipmentApi.get(projectId, id),
            save: (back) => equipmentApi.update(projectId, id, back),
          })
        : undefined,
    });
    await reloadRows();
    await reloadGame();
    onSaved(t("saved", { name: title }));
  };

  return (
    <>
      <div className="-mx-4 mt-2">
        <EquipmentEditor equipment={row} list={rows} onSave={save} />
      </div>
      <section className="mt-4 space-y-3 border-t border-border pt-3">
        <p className="text-micro text-muted-foreground">{t("extrasNote")}</p>
        {game?.country_key && (
          <CountryName
            projectId={projectId}
            tag={tag}
            game={game}
            countryKey={game.country_key}
            kind={kind}
            workbenchHref={workbenchHref}
            reloadGame={reloadGame}
            onSaved={onSaved}
          />
        )}
        <NewModel
          projectId={projectId}
          name={name}
          base={{ archetype: row.archetype || name, year: row.year }}
          rows={rows}
          reloadRows={reloadRows}
          onSaved={onSaved}
        />
      </section>
    </>
  );
}

function CountryName({
  projectId,
  tag,
  game,
  countryKey,
  kind,
  workbenchHref,
  reloadGame,
  onSaved,
}: {
  projectId: number;
  tag: string | null;
  game: GameEquipment;
  countryKey: string;
  kind: string;
  workbenchHref: string | null;
  reloadGame: () => Promise<unknown>;
  onSaved: (message: string) => void;
}) {
  const t = useTranslations("gameViewMilitary");
  const start = game.country_label ?? "";
  const [text, setText] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const value = text ?? start;
  const next = value.trim();

  const problem =
    game.country_label && !next ? t("equipment.problem.countryNameKept") : null;
  const changed = !!next && next !== start;

  const save = async () => {
    if (!changed || problem) return;
    setBusy(true);
    setError(null);
    try {
      const undo = await writeText(projectId, countryKey, next);
      recordChange({ projectId, title: next, kind, workbenchHref, undo });
      await reloadGame();
      setText(null);
      onSaved(t("common.saved", { name: next }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div>
      <label className="block">
        <span className="text-micro text-muted-foreground">
          {t(
            game.country_label
              ? "equipment.countryNameFor"
              : "equipment.countryNameForOptional",
            { tag: tag ?? "" },
          )}
        </span>
        <input
          aria-label={t("equipment.countryName")}
          value={value}
          placeholder={game.label ?? ""}
          disabled={busy}
          onChange={(e) => setText(e.target.value)}
          className={FIELD}
        />
      </label>
      <PanelSave
        onSave={() => void save()}
        canSave={changed && !problem}
        busy={busy}
        onRevert={
          text !== null && text !== start ? () => setText(null) : undefined
        }
        label={t("equipment.saveCountryName")}
        problem={problem}
        note={t("equipment.countryNameNote", { key: countryKey })}
        error={error}
      />
    </div>
  );
}

function GameEquipmentForm({
  projectId,
  tag,
  name,
  game,
  loading,
  rows,
  kind,
  reloadGame,
  reloadRows,
  onSaved,
}: {
  projectId: number;
  tag: string | null;
  name: string;
  game: GameEquipment | undefined;
  loading: boolean;
  rows: readonly Equipment[] | undefined;
  kind: string;
  reloadGame: () => Promise<unknown>;
  reloadRows: () => Promise<unknown>;
  onSaved: (message: string) => void;
}) {
  const t = useTranslations("gameViewMilitary");
  const statLabel = useStatLabel();
  const start: EquipmentDraft | null = game?.found
    ? equipmentDraft(game)
    : null;
  const [draft, setDraft] = useState<EquipmentDraft | null>(null);
  const identity = `${name}/${JSON.stringify(start)}`;
  const [seen, setSeen] = useState(identity);
  if (seen !== identity) {
    setSeen(identity);
    setDraft(null);
  }
  const current = draft ?? start;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const problem = current ? equipmentProblem(current, game) : null;
  const problemText = !problem
    ? null
    : problem.key === "notNumber"
      ? t("equipment.problem.notNumber", { field: statLabel(problem.field) })
      : t(`equipment.problem.${problem.key}`);
  const patch = current && start ? equipmentPatch(current, start) : null;
  const changed =
    !!patch &&
    (Object.keys(patch.values).length > 0 ||
      Object.keys(patch.resources).length > 0 ||
      !!patch.label ||
      !!patch.countryLabel);
  const save = async () => {
    if (!patch || !current || problem) return;
    setBusy(true);
    setError(null);
    try {
      const ways: ((() => Promise<void>) | undefined)[] = [];
      if (
        Object.keys(patch.values).length ||
        Object.keys(patch.resources).length
      ) {
        const out = await postJSON(
          `/api/go/projects/${projectId}/gui-render/equipment-save`,
          {
            name,
            values: patch.values,
            resources: patch.resources,
          },
        );
        ways.push(fileUndo(projectId, out));
      }
      if (patch.label) ways.push(await writeText(projectId, name, patch.label));
      if (patch.countryLabel && game?.country_key)
        ways.push(
          await writeText(projectId, game.country_key, patch.countryLabel),
        );
      recordChange({
        projectId,
        title:
          patch.countryLabel ||
          patch.label ||
          current.countryLabel ||
          current.label ||
          name,
        kind,
        undo: undoAll(...ways),
      });
      await reloadGame();
      onSaved(
        t("common.saved", {
          name:
            patch.countryLabel ??
            patch.label ??
            current.countryLabel ??
            current.label,
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
      {loading && (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("equipment.loading")}
        </p>
      )}
      {game && !game.found && (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("equipment.notFound")}
        </p>
      )}
      {current && game && (
        <section className="mt-4 space-y-3">
          {game.country_key && (
            <label className="block">
              <span className="text-micro text-muted-foreground">
                {t(
                  game.country_label
                    ? "equipment.countryNameFor"
                    : "equipment.countryNameForOptional",
                  { tag: tag ?? "" },
                )}
              </span>
              <input
                aria-label={t("equipment.countryName")}
                value={current.countryLabel}
                placeholder={current.label}
                disabled={busy}
                onChange={(e) =>
                  setDraft({ ...current, countryLabel: e.target.value })
                }
                className={FIELD}
              />
            </label>
          )}
          <div className="grid grid-cols-3 gap-2">
            <label className="col-span-2 block">
              <span className="text-micro text-muted-foreground">
                {t("equipment.genericName")}
              </span>
              <input
                aria-label={t("equipment.nameLabel")}
                value={current.label}
                disabled={busy}
                onChange={(e) =>
                  setDraft({ ...current, label: e.target.value })
                }
                className={FIELD}
              />
            </label>
            <label className="block">
              <span className="text-micro text-muted-foreground">
                {t("equipment.year")}
              </span>
              <input
                aria-label={t("equipment.year")}
                value={current.year}
                disabled={busy}
                onChange={(e) => setDraft({ ...current, year: e.target.value })}
                className={FIELD}
              />
            </label>
          </div>
          <div>
            <div className="flex items-center justify-between text-micro text-muted-foreground">
              <span>{t("equipment.numbers")}</span>
              <span>{t("equipment.numbersHint")}</span>
            </div>
            {statKeys(game).map((key) => (
              <label key={key} className="mt-1 flex items-center gap-2">
                <span className="min-w-0 flex-1 truncate text-sm">
                  {statLabel(key)}
                  <span className="ml-1 font-mono text-micro text-muted-foreground">
                    {key}
                  </span>
                </span>
                <input
                  aria-label={t("equipment.statValue", {
                    stat: statLabel(key),
                  })}
                  value={current.numbers[key] ?? ""}
                  placeholder={
                    game.inherited?.[key] !== undefined
                      ? String(game.inherited[key])
                      : ""
                  }
                  disabled={busy}
                  onChange={(e) =>
                    setDraft({
                      ...current,
                      numbers: { ...current.numbers, [key]: e.target.value },
                    })
                  }
                  className={SMALL}
                />
              </label>
            ))}
          </div>
          {resourceKeys(game).length > 0 && (
            <div>
              <div className="text-micro text-muted-foreground">
                {t("equipment.resources")}
              </div>
              {resourceKeys(game).map((key) => (
                <label key={key} className="mt-1 flex items-center gap-2">
                  <span className="min-w-0 flex-1 truncate font-mono text-sm">
                    {key}
                  </span>
                  <input
                    aria-label={t("equipment.resource", { key })}
                    value={current.resources[key] ?? ""}
                    placeholder={
                      game.inherited_resources?.[key] !== undefined
                        ? String(game.inherited_resources[key])
                        : ""
                    }
                    disabled={busy}
                    onChange={(e) =>
                      setDraft({
                        ...current,
                        resources: {
                          ...current.resources,
                          [key]: e.target.value,
                        },
                      })
                    }
                    className={SMALL}
                  />
                </label>
              ))}
            </div>
          )}
          <PanelSave
            onSave={() => void save()}
            canSave={changed && !problem}
            busy={busy}
            onRevert={draft ? () => setDraft(null) : undefined}
            label={t("equipment.save")}
            problem={problemText}
            note={t("equipment.fileNote", { file: game.file ?? "" })}
            error={error}
          />

          <NewModel
            projectId={projectId}
            name={name}
            base={{ archetype: game.archetype || name, year: game.year }}
            rows={rows}
            disabled={busy}
            reloadRows={reloadRows}
            onSaved={onSaved}
          />
        </section>
      )}
    </>
  );
}

function NewModel({
  projectId,
  name,
  base,
  rows,
  disabled = false,
  reloadRows,
  onSaved,
}: {
  projectId: number;
  name: string;

  base: { archetype: string; year: number | null | undefined };
  rows: readonly Equipment[] | undefined;

  disabled?: boolean;
  reloadRows: () => Promise<unknown>;
  onSaved: (message: string) => void;
}) {
  const t = useTranslations("gameViewMilitary");
  const [newModel, setNewModel] = useState({ token: "", label: "", year: "" });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const off = busy || disabled;

  const tokenProblem = newEquipmentProblem(newModel.token, [name]);
  const newProblem = tokenProblem
    ? tokenProblem.key === "tokenTaken"
      ? t("equipment.problem.tokenTaken", { token: tokenProblem.token })
      : t(`equipment.problem.${tokenProblem.key}`)
    : !newModel.label.trim()
      ? t("equipment.newModel.problemName")
      : newModel.year.trim() && !/^\d{4}$/.test(newModel.year.trim())
        ? t("equipment.newModel.problemYear")
        : null;
  const createModel = async () => {
    if (newProblem) return;
    setBusy(true);
    setError(null);
    try {
      const token = newModel.token.trim();
      const exists = await fetchEquipment(projectId, null, token);
      if (exists.found || rows?.some((item) => item.equipment_id === token))
        throw new Error(t("equipment.problem.tokenTaken", { token }));
      const label = newModel.label.trim();
      const created = await equipmentApi.create(projectId, {
        equipment_id: token,
        archetype: base.archetype,
        parent: name,
        is_archetype: false,
        year: newModel.year.trim()
          ? Number(newModel.year)
          : (base.year ?? null),
        name_en: label,
        display_name_locs: { en: label },
      });

      recordChange({
        projectId,
        title: t("log.create", { name: label }),
        kind: t("equipment.kind", { name: token }),
        workbenchHref: `/project/${projectId}/equipment?equipment=${created.id}`,
      });
      await reloadRows();
      setNewModel({ token: "", label: "", year: "" });
      onSaved(t("equipment.newModel.created", { name: label }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-1 rounded border border-dashed border-border p-2">
      <div className="text-micro text-muted-foreground">
        {t("equipment.newModel.heading")}
      </div>
      <div className="flex items-center gap-1">
        <input
          aria-label={t("equipment.newModel.token")}
          value={newModel.token}
          placeholder={t("equipment.newModel.tokenPlaceholder")}
          disabled={off}
          onChange={(e) => setNewModel({ ...newModel, token: e.target.value })}
          className="w-32 rounded border border-input bg-background px-1 py-0.5 font-mono text-sm"
        />
        <input
          aria-label={t("equipment.newModel.name")}
          value={newModel.label}
          placeholder={t("equipment.newModel.namePlaceholder")}
          disabled={off}
          onChange={(e) => setNewModel({ ...newModel, label: e.target.value })}
          className="min-w-0 flex-1 rounded border border-input bg-background px-1 py-0.5 text-sm"
        />
        <input
          aria-label={t("equipment.newModel.year")}
          value={newModel.year}
          placeholder={t("equipment.newModel.yearPlaceholder")}
          disabled={off}
          onChange={(e) => setNewModel({ ...newModel, year: e.target.value })}
          className="w-14 rounded border border-input bg-background px-1 py-0.5 text-sm"
        />
      </div>
      {(newModel.token || newModel.label) && newProblem && (
        <p className="text-micro text-destructive">{newProblem}</p>
      )}
      {error && <p className="text-micro text-destructive">{error}</p>}
      <div className="flex justify-end">
        <button
          type="button"
          className={BUTTON}
          disabled={off || !!newProblem}
          onClick={() => void createModel()}
        >
          {t("equipment.newModel.create")}
        </button>
      </div>
    </div>
  );
}
