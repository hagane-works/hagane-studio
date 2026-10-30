"use client";

import { Fragment, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { focusTreesApi } from "@/lib/api/focus-tech";
import { FocusNodeEditor } from "@/components/workbench/FocusNodeEditor";
import type { FocusNode as FocusRow } from "@/types";
import type { FocusNode } from "./FocusTreeLayer";
import { draftText } from "./focusEdit";
import { BUTTON, GamePanel } from "./GamePanel";
import { pick, recordChange, rowUndo } from "./changeLog";
import { BaseTextEdit } from "./BaseTextEdit";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

export function FocusEditPanel({
  projectId,
  treeId,
  focus,
  onSaved,
  onAddChild,
  onDelete,
  focuses,
  picking,
  onPick,
  onUnlinkPrereq,
  onUnlinkExclusive,
  onClose,
}: {
  projectId: number;

  treeId: string;
  focus: FocusNode;

  onSaved: (focusId: string, change: { name?: string; cost?: number }) => void;

  onAddChild: () => Promise<void>;

  onDelete: () => Promise<void>;

  focuses: readonly FocusNode[];

  picking: { kind: "prereq" | "mutex"; into?: number } | null;

  onPick: (kind: "prereq" | "mutex", into?: number) => void;
  onUnlinkPrereq: (group: number, id: string) => void;
  onUnlinkExclusive: (id: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewContent.focus");
  const tEdit = useTranslations("gameViewEdit");
  const tree = useSWR(
    ["game-view-focus-db", projectId, treeId],
    () => focusTreesApi.findByScriptId(projectId, treeId),
    {
      revalidateOnFocus: false,
    },
  );
  const node = tree.data?.nodes?.find((n) => n.focus_id === focus.id) ?? null;
  const projectReady = useWorkbenchProject(projectId);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const [seen, setSeen] = useState(focus.id);
  if (seen !== focus.id) {
    setSeen(focus.id);
    setError(null);
    setConfirmDelete(false);
  }
  const kind = t("kind", { tree: treeId, id: focus.id });
  const workbenchHref =
    tree.data && node
      ? `/project/${projectId}/focus-tree?tree=${tree.data.id}&focus=${node.id}`
      : null;

  const save = async (data: Partial<FocusRow>) => {
    if (!node || !tree.data) return;
    const { tree_edit_revision: revision, ...fields } = data;
    const treeDbId = tree.data.id;
    const nodeId = node.id;
    const keys = changedKeys(node, fields);
    const before = pick(node, keys);
    const saved = await focusTreesApi.updateNode(
      projectId,
      treeDbId,
      nodeId,
      fields,
      revision ?? node.tree_edit_revision,
    );
    const name = draftText(saved.display_name_locs, saved.name_en, focus.name);
    recordChange({
      projectId,
      title: name || focus.name,
      kind,
      workbenchHref,
      undo: keys.length
        ? rowUndo({
            before,
            after: pick(saved, keys),
            load: () => focusTreesApi.getNode(projectId, treeDbId, nodeId),
            save: (back, now) =>
              focusTreesApi.updateNode(
                projectId,
                treeDbId,
                nodeId,
                back,
                now.tree_edit_revision,
              ),
          })
        : undefined,
    });
    onSaved(focus.id, {
      ...(name !== focus.name ? { name } : {}),
      ...(saved.cost !== focus.cost ? { cost: saved.cost } : {}),
    });
    await tree.mutate();
  };

  const run = async (work: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await work();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const nameOf = (id: string) => focuses.find((f) => f.id === id)?.name || id;

  const addChild = async () => {
    await onAddChild();
    recordChange({
      projectId,
      title: t("newChildLog", { name: focus.name }),
      kind,
      workbenchHref,
    });
  };
  const remove = async () => {
    await onDelete();
    recordChange({
      projectId,
      title: tEdit("person.deleteLog", { name: focus.name }),
      kind,
      workbenchHref,
    });
  };

  return (
    <GamePanel
      title={focus.name}
      kind={kind}
      source={tree.isLoading ? undefined : node ? "项目" : "原版"}
      workbenchHref={workbenchHref}

      wide={node ? "workbench" : true}
      onClose={onClose}
    >
      {tree.isLoading && (
        <p className="mt-4 text-micro text-muted-foreground">{t("loading")}</p>
      )}
      {!tree.isLoading && !node && (
        <>
          {tree.data && (
            <p className="mt-4 text-micro text-muted-foreground">
              {t("notInTree")}
            </p>
          )}

          <BaseTextEdit
            projectId={projectId}
            title={focus.name}
            fields={[
              { field: "name", key: focus.id },
              { field: "description", key: `${focus.id}_desc` },
            ]}
            onSaved={(key, value) =>
              onSaved(
                focus.id,
                key === focus.id ? { name: value ?? undefined } : {},
              )
            }
          />
        </>
      )}

      {node &&
        tree.data &&
        (projectReady ? (
          <div className="-mx-4 mt-2">
            <FocusNodeEditor
              key={node.id}
              node={node}
              allNodes={tree.data.nodes ?? []}
              projectId={projectId}
              countryTag={tree.data.country_tag ?? ""}
              treeId={tree.data.id}
              compact
              onSave={save}
              onIconUploaded={() => {
                void tree.mutate();
                onSaved(focus.id, {});
              }}
            />
          </div>
        ) : (
          <p className="mt-4 text-micro text-muted-foreground">
            {tEdit("panel.loadingProject")}
          </p>
        ))}
      {error && <p className="mt-2 text-micro text-destructive">{error}</p>}

      {node && (
        <section className="mt-4 space-y-3">
          <div>
            <div className="text-micro font-medium text-muted-foreground">
              {t("prereqs")}
            </div>
            <p className="text-micro text-muted-foreground">
              {t("prereqHint")}
            </p>
            {(focus.prereq ?? []).map((group, i) => (
              <div key={i} className="mt-1 flex flex-wrap items-center gap-1">
                {group.map((id, j) => (
                  <Fragment key={id}>
                    {j > 0 && (
                      <span className="text-micro text-muted-foreground">
                        {t("or")}
                      </span>
                    )}
                    <LinkChip
                      label={nameOf(id)}
                      id={id}
                      onRemove={() => onUnlinkPrereq(i, id)}
                    />
                  </Fragment>
                ))}
                <PickButton
                  active={picking?.kind === "prereq" && picking.into === i}
                  onClick={() => onPick("prereq", i)}
                >
                  {t("addOr")}
                </PickButton>
              </div>
            ))}
            <div className="mt-1">
              <PickButton
                active={
                  picking?.kind === "prereq" && picking.into === undefined
                }
                onClick={() => onPick("prereq")}
              >
                {t("addPrereq")}
              </PickButton>
            </div>
          </div>
          <div>
            <div className="text-micro font-medium text-muted-foreground">
              {t("mutex")}
            </div>
            <div className="mt-1 flex flex-wrap items-center gap-1">
              {(focus.mutex ?? []).map((id) => (
                <LinkChip
                  key={id}
                  label={nameOf(id)}
                  id={id}
                  onRemove={() => onUnlinkExclusive(id)}
                />
              ))}
              <PickButton
                active={picking?.kind === "mutex"}
                onClick={() => onPick("mutex")}
              >
                {t("addMutex")}
              </PickButton>
            </div>
          </div>
        </section>
      )}

      {node && (
        <section className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={() => void run(addChild)}
            className={BUTTON}
          >
            {t("newChild")}
          </button>
          {confirmDelete ? (
            <>
              <button
                type="button"
                disabled={busy}
                onClick={() => void run(remove)}
                className="rounded bg-destructive px-3 py-1 text-micro text-destructive-foreground hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {t("confirmDelete")}
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => setConfirmDelete(false)}
                className="rounded border border-border px-3 py-1 text-micro hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50"
              >
                {tEdit("person.cancel")}
              </button>
            </>
          ) : (
            <button
              type="button"
              disabled={busy}
              onClick={() => setConfirmDelete(true)}
              className="rounded border border-border px-3 py-1 text-micro text-destructive hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50"
            >
              {t("delete")}
            </button>
          )}
        </section>
      )}
    </GamePanel>
  );
}

function LinkChip({
  label,
  id,
  onRemove,
}: {
  label: string;
  id: string;
  onRemove: () => void;
}) {
  const t = useTranslations("gameViewContent.focus");
  return (
    <span
      title={id}
      className="inline-flex max-w-full items-center gap-1 rounded border border-border bg-muted px-1.5 py-0.5 text-micro"
    >
      <span className="truncate">{label}</span>
      <button
        type="button"
        aria-label={t("remove", { id })}
        onClick={onRemove}
        className="text-muted-foreground hover:text-foreground"
      >
        ×
      </button>
    </span>
  );
}

function PickButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  const t = useTranslations("gameViewContent.focus");
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded border px-2 py-0.5 text-micro ${
        active
          ? "border-primary bg-primary text-primary-foreground"
          : "border-border hover:bg-muted"
      }`}
    >
      {active ? t("picking") : children}
    </button>
  );
}
