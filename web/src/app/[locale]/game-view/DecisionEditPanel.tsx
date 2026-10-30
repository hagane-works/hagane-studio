"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { decisionsApi } from "@/lib/api/content";
import type { Decision, DecisionCategory } from "@/types";
import {
  categoryForEditor,
  categoryIconSprite,
  decisionDeletable,
  decisionDraft,
  decisionIconSprite,
  newDecisionId,
} from "./decisionEdit";
import type {
  DecisionEdits,
  DecisionPick,
  GuiDecision,
  GuiDecisionCategory,
} from "./DecisionsLayer";
import { BUTTON, GamePanel } from "./GamePanel";
import { pick as pickKeys, recordChange, rowUndo } from "./changeLog";
import { BaseTextEdit } from "./BaseTextEdit";
import { DecisionOverride } from "./DecisionOverride";
import { DecisionEditor } from "@/components/workbench/DecisionEditor";
import {
  DecisionCategoryEditor,
  type DecisionCategoryForm,
} from "@/components/workbench/DecisionCategoryEditor";
import { draftText } from "./focusEdit";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

export function findCategoryRow(
  rows: readonly DecisionCategory[] | undefined,
  id: string,
): DecisionCategory | null {
  const want = id.toLowerCase();
  return rows?.find((row) => row.category_id.toLowerCase() === want) ?? null;
}

export function DecisionEditPanel({
  projectId,
  tag,
  pick,
  category,
  onSaved,
  onCreated,
  onDeleted,
  onClose,
}: {
  projectId: number;
  tag: string;
  pick: DecisionPick;

  category: GuiDecisionCategory | undefined;

  onSaved: (pick: DecisionPick, shown: DecisionEdits[string]) => void;

  onCreated: (pick: Extract<DecisionPick, { kind: "decision" }>) => void;

  onDeleted: (pick: DecisionPick) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewContent.decision");
  const shownDecision: GuiDecision | undefined =
    pick.kind === "decision"
      ? category?.decisions.find((d) => d.name === pick.name)
      : undefined;
  const categories = useSWR(
    ["game-view-decision-categories-db", projectId],
    () => decisionsApi.listCategories(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  const categoryRow = findCategoryRow(categories.data, pick.category);
  const decisionRow = useSWR(
    pick.kind === "decision"
      ? ["game-view-decision-db", projectId, pick.name]
      : null,
    () =>
      decisionsApi.findByScriptId(projectId, (pick as { name: string }).name),
    { revalidateOnFocus: false },
  );

  const title =
    pick.kind === "decision"
      ? shownDecision?.label || pick.name
      : category?.label || pick.category;
  const scriptId = pick.kind === "decision" ? pick.name : pick.category;

  const notProject = (origin?: string) =>
    origin === "mod" ? "mod 文件" : "原版";
  const source =
    pick.kind === "decision"
      ? decisionRow.isLoading
        ? undefined
        : decisionRow.data
          ? "项目"
          : notProject(shownDecision?.origin)
      : categories.isLoading
        ? undefined
        : categoryRow
          ? "项目"
          : notProject(category?.origin);

  return (
    <GamePanel
      title={title}
      kind={t(pick.kind === "decision" ? "kind" : "categoryKind", {
        id: scriptId,
      })}
      source={source}
      workbenchHref={
        decisionRow.data
          ? `/project/${projectId}/decisions?category=${decisionRow.data.category_id}&decision=${decisionRow.data.id}`
          : pick.kind !== "decision" && categoryRow
            ? `/project/${projectId}/decisions?category=${categoryRow.id}`
            : null
      }

      wide={
        (pick.kind === "decision" ? decisionRow.data : categoryRow)
          ? "workbench"
          : true
      }
      onClose={onClose}
    >
      {pick.kind === "decision" ? (
        <DecisionFields
          key={pick.name}
          projectId={projectId}
          tag={tag}
          pick={pick}
          shown={shownDecision}
          row={decisionRow.data ?? null}
          loading={decisionRow.isLoading}
          reload={() => decisionRow.mutate()}
          onSaved={onSaved}
          onDeleted={onDeleted}
        />
      ) : (
        <CategoryFields
          key={pick.category}
          projectId={projectId}
          tag={tag}
          pick={pick}
          shown={category}
          row={categoryRow}
          loading={categories.isLoading}
          reload={() => categories.mutate()}
          onSaved={onSaved}
          onCreated={onCreated}
        />
      )}
    </GamePanel>
  );
}

function DecisionFields({
  projectId,
  tag,
  pick,
  shown,
  row,
  loading,
  reload,
  onSaved,
  onDeleted,
}: {
  projectId: number;

  tag: string;
  pick: Extract<DecisionPick, { kind: "decision" }>;
  shown: GuiDecision | undefined;
  row: Decision | null;
  loading: boolean;
  reload: () => Promise<unknown>;
  onSaved: (pick: DecisionPick, shown: DecisionEdits[string]) => void;
  onDeleted: (pick: DecisionPick) => void;
}) {
  const t = useTranslations("gameViewContent.decision");
  const tEdit = useTranslations("gameViewEdit");
  const projectReady = useWorkbenchProject(projectId);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const remove = async () => {
    if (!row) return;
    setBusy(true);
    setError(null);
    try {
      await decisionsApi.delete(projectId, row.id);
      recordChange({
        projectId,
        title: tEdit("person.deleteLog", { name: shown?.label || pick.name }),
        kind: t("kind", { id: pick.name }),
        workbenchHref: `/project/${projectId}/decisions?category=${row.category_id}&decision=${row.id}`,
      });
      onDeleted(pick);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  };

  const save = async (data: Partial<Decision>) => {
    if (!row) return;
    const id = row.id;
    const keys = changedKeys(row, data);
    const before = pickKeys(row, keys);
    const saved = await decisionsApi.update(projectId, id, data);
    const label = decisionDraft(saved, shown ?? {}, "").name;
    recordChange({
      projectId,
      title: label || shown?.label || pick.name,
      kind: t("kind", { id: pick.name }),
      workbenchHref: `/project/${projectId}/decisions?category=${row.category_id}&decision=${id}`,
      undo: keys.length
        ? rowUndo({
            before,
            after: pickKeys(saved, keys),
            load: () => decisionsApi.get(projectId, id),
            save: (back) => decisionsApi.update(projectId, id, back),
          })
        : undefined,
    });
    onSaved(pick, {
      label,
      cost: saved.cost,
      icon: saved.icon ? decisionIconSprite(saved.icon) : undefined,
    });
    await reload();
  };

  if (loading)
    return (
      <p className="mt-4 text-micro text-muted-foreground">{t("loading")}</p>
    );
  if (!row) {
    const nameKey = shown?.label_key || pick.name;
    return (
      <>
        <BaseTextEdit
          projectId={projectId}
          origin={shown?.origin}
          file={shown?.file}
          title={shown?.label || pick.name}
          fields={[
            { field: "name", key: nameKey },
            { field: "description", key: `${pick.name}_desc` },
          ]}
          logicNote="logicNoteDecision"
          onSaved={(key, value) =>
            onSaved(pick, key === nameKey ? { label: value ?? undefined } : {})
          }
        />

        {shown && (
          <DecisionOverride
            key={JSON.stringify([
              shown.origin,
              shown.cost,
              shown.icon_token,
              shown.days_remove,
              shown.days_re_enable,
              shown.fire_only_once,
            ])}
            projectId={projectId}
            pick={pick}
            shown={shown}
            onSaved={onSaved}
          />
        )}
      </>
    );
  }
  return (
    <section className="mt-2 space-y-3">
      {projectReady ? (
        <div className="-mx-4">
          <DecisionEditor
            key={row.id}
            decision={row}
            projectId={projectId}
            countryTag={tag}
            onSave={save}
            onIconUploaded={() => {
              void reload();
              onSaved(pick, {});
            }}
          />
        </div>
      ) : (
        <p className="text-micro text-muted-foreground">
          {tEdit("panel.loadingProject")}
        </p>
      )}
      {error && <p className="text-micro text-destructive">{error}</p>}

      {decisionDeletable(row) ? (
        confirmDelete ? (
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
              {tEdit("person.delete")}
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => setConfirmDelete(false)}
              className="rounded border border-border px-2 py-0.5 text-micro hover:bg-muted"
            >
              {tEdit("person.cancel")}
            </button>
          </div>
        ) : (
          <button
            type="button"
            disabled={busy}
            onClick={() => setConfirmDelete(true)}
            className="rounded border border-border px-2 py-0.5 text-micro text-destructive hover:bg-muted disabled:opacity-50"
          >
            {t("delete")}
          </button>
        )
      ) : (
        <p className="text-micro text-muted-foreground">{t("notDeletable")}</p>
      )}
    </section>
  );
}

function CategoryFields({
  projectId,
  tag,
  pick,
  shown,
  row,
  loading,
  reload,
  onSaved,
  onCreated,
}: {
  projectId: number;
  tag: string;
  pick: Extract<DecisionPick, { kind: "category" }>;
  shown: GuiDecisionCategory | undefined;
  row: DecisionCategory | null;
  loading: boolean;
  reload: () => Promise<unknown>;
  onSaved: (pick: DecisionPick, shown: DecisionEdits[string]) => void;
  onCreated: (pick: Extract<DecisionPick, { kind: "decision" }>) => void;
}) {
  const t = useTranslations("gameViewContent.decision");
  const tEdit = useTranslations("gameViewEdit");
  const projectReady = useWorkbenchProject(projectId);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const save = async (data: DecisionCategoryForm) => {
    if (!row) return;
    const id = row.id;
    const keys = changedKeys<DecisionCategory>(row, data);
    if (!keys.length) return;
    const before = pickKeys(row, keys);
    const saved = await decisionsApi.updateCategory(projectId, id, data);
    const name = draftText(
      saved.display_name_locs,
      saved.name_en || undefined,
      shown?.label ?? pick.category,
    );
    recordChange({
      projectId,
      title: name,
      kind: t("categoryKind", { id: pick.category }),
      workbenchHref: `/project/${projectId}/decisions?category=${id}`,
      undo: rowUndo({
        before,
        after: pickKeys(saved, keys),
        load: async () =>
          (await decisionsApi.listCategories(projectId)).find(
            (category) => category.id === id,
          ),
        save: (back) => decisionsApi.updateCategory(projectId, id, back),
      }),
    });
    onSaved(pick, {
      label: name,
      icon: saved.icon ? categoryIconSprite(saved.icon) : undefined,
    });
    await reload();
  };
  const addDecision = async () => {
    if (!row) return;
    setBusy(true);
    setError(null);
    try {
      const decisionId = newDecisionId(
        tag,
        (shown?.decisions ?? []).map((d) => d.name),
      );
      const created = await decisionsApi.create(projectId, {
        category_id: row.id,
        decision_id: decisionId,
        country_tag: tag,
        cost: 0,
        display_name_locs: { en: "New Decision" },
      });
      recordChange({
        projectId,
        title: t("newLog", {
          name: created.display_name_locs?.en || "New Decision",
        }),
        kind: t("kind", { id: created.decision_id }),
        workbenchHref: `/project/${projectId}/decisions?category=${row.id}&decision=${created.id}`,
      });
      onCreated({
        kind: "decision",
        category: pick.category,
        name: created.decision_id,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  if (loading)
    return (
      <p className="mt-4 text-micro text-muted-foreground">
        {t("loadingCategory")}
      </p>
    );
  if (!row) {
    return (
      <BaseTextEdit
        projectId={projectId}
        origin={shown?.origin}
        title={shown?.label || pick.category}
        fields={[
          { field: "name", key: pick.category },
          { field: "description", key: `${pick.category}_desc` },
        ]}
        onSaved={(key, value) =>
          onSaved(
            pick,
            key === pick.category ? { label: value ?? undefined } : {},
          )
        }
      />
    );
  }
  const category = categoryForEditor(row, shown?.icon);
  return (
    <>
      {projectReady ? (
        <div className="-mx-4 mt-2">
          <DecisionCategoryEditor
            key={row.id}
            projectId={projectId}
            category={category}
            onSave={save}
          />
        </div>
      ) : (
        <p className="mt-4 text-micro text-muted-foreground">
          {tEdit("panel.loadingProject")}
        </p>
      )}
      <section className="mt-2 space-y-1 border-t border-border pt-3">
        <button
          type="button"
          disabled={busy}
          onClick={() => void addDecision()}
          className={BUTTON}
        >
          {t("add")}
        </button>
        <p className="text-micro text-muted-foreground">
          {t("addNote", { tag })}
        </p>
        {error && (
          <p role="alert" className="text-micro text-destructive">
            {error}
          </p>
        )}
      </section>
    </>
  );
}
