"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { DeploymentEditor } from "@/app/[locale]/project/[id]/deployments/DeploymentEditor";
import {
  divisionDeploymentsApi,
  divisionTemplatesApi,
} from "@/lib/api/military";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";
import { guiRenderApi, type GuiDivisions } from "@/lib/guiRenderApi";
import type { DivisionDeployment } from "@/types";
import { BUTTON, GamePanel } from "./GamePanel";
import { pick, recordChange, rowUndo } from "./changeLog";
import {
  countryDeployments,
  deploymentLabel,
  newDeploymentTemplate,
} from "./deploymentEdit";
import { changedKeys } from "./workbenchProject";

export function DeploymentEditPanel({
  projectId,
  tag,
  date,
  divisions,
  onSaved,
  onClose,
}: {
  projectId: number;
  tag: string;
  date?: string | null;

  divisions: GuiDivisions | undefined;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewMilitary");

  const label = (
    row: Pick<DivisionDeployment, "count" | "division_template">,
  ) => deploymentLabel(row, t("deployment.noTemplate"));

  const rows = useSWR(
    ["game-view-deployments-db", projectId],
    () => divisionDeploymentsApi.list(projectId),
    {
      revalidateOnFocus: false,
    },
  );

  const templates = useSWR(
    ["game-view-division-templates-db", projectId],
    () => divisionTemplatesApi.list(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  const mine = countryDeployments(rows.data, tag);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const selected = mine.find((row) => row.id === selectedId) ?? null;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const kind = t("deployment.kind", { tag });
  const workbenchHref = `/project/${projectId}/deployments`;

  const run = async (work: () => Promise<string>) => {
    setBusy(true);
    setError(null);
    try {
      const message = await work();
      await rows.mutate();
      onSaved(message);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const select = (id: number | null) => {
    if (confirmDiscardIfDirty()) setSelectedId(id);
  };

  const add = () => {
    if (!confirmDiscardIfDirty()) return;
    void run(async () => {
      guiRenderApi.setProjectId(projectId);
      const taken = await guiRenderApi.takeOOB(tag, date);

      const template = newDeploymentTemplate(
        await templates.mutate(),
        taken.oob_file,
        divisions,
      );
      const created = await divisionDeploymentsApi.create(projectId, {
        oob_file: taken.oob_file,
        country_tag: tag.toUpperCase(),
        division_template: template,
        count: 1,
      });

      recordChange({
        projectId,
        title: t("log.create", { name: label(created) }),
        kind,
        workbenchHref,
      });
      setSelectedId(created.id);
      return taken.imported
        ? t("deployment.addedImported", {
            name: label(created),
            file: taken.oob_file,
          })
        : t("deployment.added", { name: label(created) });
    });
  };

  const remove = (row: DivisionDeployment) =>
    void run(async () => {
      await divisionDeploymentsApi.delete(projectId, row.id);

      recordChange({
        projectId,
        title: t("log.delete", { name: label(row) }),
        kind,
        workbenchHref,
      });
      if (row.id === selectedId) setSelectedId(null);
      return t("deployment.deleted", { name: label(row) });
    });

  const save = async (payload: Partial<DivisionDeployment>) => {
    if (!selected) return;
    const id = selected.id;
    const keys = changedKeys(selected, payload);
    const before = pick(selected, keys);
    const saved = await divisionDeploymentsApi.update(projectId, id, payload);
    recordChange({
      projectId,
      title: label(saved),
      kind,
      workbenchHref,
      undo: keys.length
        ? rowUndo<DivisionDeployment>({
            before,
            after: pick(saved, keys),
            load: () => divisionDeploymentsApi.get(projectId, id),
            save: (back) => divisionDeploymentsApi.update(projectId, id, back),
          })
        : undefined,
    });
    await rows.mutate();
    onSaved(t("common.saved", { name: label(saved) }));
  };

  return (
    <GamePanel
      title={t("deployment.title")}
      kind={kind}
      detail={(divisions?.oob ?? []).join(", ") || "…"}
      workbenchHref={workbenchHref}

      wide={selected ? "workbench" : true}
      onClose={onClose}
    >
      <section className="mt-3">
        <div className="text-micro font-medium text-muted-foreground">
          {t("deployment.countsHeading")}
        </div>
        <ul className="mt-1 space-y-0.5">
          {(divisions?.templates ?? []).map((template) => (
            <li key={template.name} className="flex justify-between text-sm">
              <span className="truncate">{template.name}</span>
              <span className="tabular-nums text-muted-foreground">
                {t("deployment.divisions", { count: template.deployed ?? 0 })}
              </span>
            </li>
          ))}
        </ul>
      </section>

      <section className="mt-4">
        <div className="text-micro font-medium text-muted-foreground">
          {t("deployment.projectUnits")}
        </div>
        {rows.isLoading && (
          <p className="mt-1 text-micro text-muted-foreground">
            {t("common.loading")}
          </p>
        )}
        {!rows.isLoading && !mine.length && (
          <p className="mt-1 text-micro text-muted-foreground">
            {t("deployment.none")}
          </p>
        )}
        <ul className="mt-1 space-y-1">
          {mine.map((row) => {
            const open = row.id === selectedId;
            return (
              <li key={row.id} className="flex items-center gap-2 text-sm">
                <button
                  type="button"
                  aria-pressed={open}
                  title={
                    open ? t("deployment.collapse") : t("deployment.editHint")
                  }
                  disabled={busy}
                  onClick={() => select(open ? null : row.id)}
                  className={`ui-interactive min-w-0 flex-1 truncate rounded px-1.5 py-0.5 text-left outline-none hover:bg-muted focus-visible:ring-2 focus-visible:ring-accent/60 disabled:cursor-not-allowed disabled:opacity-50 ${open ? "bg-accent/15" : ""}`}
                >
                  {label(row)}
                  <span className="ml-1 font-mono text-micro text-muted-foreground">
                    @{row.location ?? "?"}
                  </span>
                </button>
                <button
                  type="button"
                  aria-label={t("deployment.deleteLabel", { name: label(row) })}
                  disabled={busy}
                  onClick={() => remove(row)}
                  className={`${BUTTON} text-destructive`}
                >
                  {t("deployment.delete")}
                </button>
              </li>
            );
          })}
        </ul>
        <button
          type="button"
          disabled={busy}
          onClick={add}
          className={`mt-2 ${BUTTON}`}
        >
          {t("deployment.add")}
        </button>
      </section>

      {selected && (
        <div className="-mx-4 mt-3 border-t border-border">
          <DeploymentEditor
            key={selected.id}
            projectId={projectId}
            deployment={selected}
            templates={templates.data ?? []}
            onSave={save}
          />
        </div>
      )}
      <p className="mt-2 text-micro text-muted-foreground">
        {t("deployment.note")}
        {!selected && mine.length > 0 && ` ${t("deployment.pickHint")}`}
      </p>
      {error && <p className="mt-2 text-micro text-destructive">{error}</p>}
    </GamePanel>
  );
}
