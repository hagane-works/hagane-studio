"use client";

import { useState, type ReactNode } from "react";
import useSWR, { mutate as globalMutate } from "swr";
import { toast } from "sonner";
import { useTranslations } from "use-intl";
import { charactersApi } from "@/lib/api/characters";
import { CharacterEditor } from "@/components/workbench/CharacterEditor";
import type { Character } from "@/types";
import { GamePanel } from "./GamePanel";
import { personDeletable } from "./personCreate";
import { pick, recordChange, rowUndo } from "./changeLog";
import { BasePersonText } from "./BaseTextEdit";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

const PEOPLE_KEYS = new Set([
  "game-view-characters",
  "game-view-idea-choices",
  "game-view-stats",
]);

export function CharacterEditPanel({
  projectId,
  token,
  name,
  onClose,
}: {
  projectId: number;

  token: string;

  name?: string;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewEdit.person");
  const rows = useSWR(
    ["game-view-characters-db", projectId],
    () => charactersApi.list(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  const row = rows.data?.find((character) => character.tag === token) ?? null;
  const [confirmDelete, setConfirmDelete] = useState(false);
  const reloadPeople = () =>
    globalMutate(
      (key) =>
        Array.isArray(key) &&
        typeof key[0] === "string" &&
        PEOPLE_KEYS.has(key[0]),
    );
  const [deleting, setDeleting] = useState(false);
  const remove = async () => {
    if (!row) return;
    setDeleting(true);
    try {
      await charactersApi.delete(projectId, row.id);
      recordChange({
        projectId,
        title: t("deleteLog", { name: name || token }),
        kind: t("kind", { token }),
        workbenchHref: `/project/${projectId}/characters?c=${row.id}`,
      });
      await rows.mutate();
      await globalMutate(
        (key) =>
          Array.isArray(key) &&
          typeof key[0] === "string" &&
          PEOPLE_KEYS.has(key[0]),
      );
      toast.success(t("deleted", { name: row.name || row.tag }));
      onClose();
    } catch (err) {
      toast.error(
        t("deleteFailed", {
          error: err instanceof Error ? err.message : String(err),
        }),
      );
      setDeleting(false);
    }
  };
  return (
    <GamePanel
      title={name || token}
      kind={t("kind", { token })}
      source={rows.isLoading ? undefined : row ? "项目" : "原版"}
      workbenchHref={
        row ? `/project/${projectId}/characters?c=${row.id}` : null
      }

      wide={row ? "workbench" : true}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">{t("loading")}</p>
      ) : !row ? (
        <BasePersonText
          projectId={projectId}
          token={token}
          name={name}
          onSaved={() => void reloadPeople()}
        />
      ) : (
        <ProjectCharacterEditor
          projectId={projectId}
          row={row}
          kind={t("kind", { token })}
          name={name || token}
          reload={async () => {
            await rows.mutate();
            await reloadPeople();
          }}
        />
      )}
      {row && personDeletable(row) && (
        <div className="mt-4 border-t border-border pt-3">
          {confirmDelete ? (
            <div className="flex items-center gap-2">
              <span className="text-micro text-destructive">
                {t("confirmDelete")}
              </span>
              <button
                type="button"
                disabled={deleting}
                onClick={() => void remove()}
                className="rounded bg-destructive px-2 py-0.5 text-micro text-destructive-foreground disabled:opacity-50"
              >
                {t("delete")}
              </button>
              <button
                type="button"
                disabled={deleting}
                onClick={() => setConfirmDelete(false)}
                className="rounded border border-border px-2 py-0.5 text-micro hover:bg-muted disabled:opacity-50"
              >
                {t("cancel")}
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => setConfirmDelete(true)}
              className="rounded border border-border px-2 py-0.5 text-micro text-destructive hover:bg-muted"
            >
              {t("deletePerson")}
            </button>
          )}
        </div>
      )}
    </GamePanel>
  );
}

export function ProjectCharacterEditor({
  projectId,
  row,
  kind,
  name,
  reload,
  onDraftChange,
  children,
}: {
  projectId: number;
  row: Character;

  kind: string;

  name?: string;

  reload: () => Promise<unknown>;

  onDraftChange?: (draft: Character) => void;
  children?: ReactNode;
}) {
  const t = useTranslations("gameViewEdit");
  const projectReady = useWorkbenchProject(projectId);
  const workbenchHref = `/project/${projectId}/characters?c=${row.id}`;

  const save = async (data: Partial<Character>) => {
    const id = row.id;
    const keys = changedKeys(row, data);
    const before = pick(row, keys);
    const saved = await charactersApi.update(projectId, id, data);
    recordChange({
      projectId,
      title: saved.name || row.name || row.tag,
      kind,
      workbenchHref,
      undo: keys.length
        ? rowUndo({
            before,
            after: pick(saved, keys),
            load: () => charactersApi.get(projectId, id),
            save: (back) => charactersApi.update(projectId, id, back),
          })
        : undefined,
    });
    await reload();
  };

  const portraitChanged = async () => {
    recordChange({
      projectId,
      title: t("person.portraitLog", { name: name || row.name || row.tag }),
      kind,
      workbenchHref,
    });
    await reload();
  };
  if (!projectReady)
    return (
      <p className="mt-4 text-micro text-muted-foreground">
        {t("panel.loadingProject")}
      </p>
    );
  return (
    <>
      <div className="-mx-4 mt-2">
        <CharacterEditor
          key={row.id}
          projectId={projectId}
          character={row}
          onSave={save}
          onRefresh={() => void portraitChanged()}
          onDraftChange={onDraftChange}
        />
      </div>
      {children}
    </>
  );
}
