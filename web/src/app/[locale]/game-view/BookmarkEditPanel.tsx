"use client";

import { useCallback, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import {
  BookmarkEditor,
  bookmarkRowToDraft,
  type BookmarkDraft,
} from "@/components/workbench/BookmarkEditor";
import { withoutEditRevisions } from "@/components/workbench/save-diagnostics/editRevisions";
import { useSaveDiagnostics } from "@/components/workbench/save-diagnostics/useSaveDiagnostics";
import { bookmarksApi, type BookmarkRow } from "@/lib/api";
import type { GuiBookmark } from "@/lib/guiRenderApi";
import { GamePanel } from "./GamePanel";
import { bookmarkRowFor, bookmarkTitle, dateProblem } from "./bookmarkEdit";
import { pick, recordChange, rowUndo } from "./changeLog";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

export function BookmarkEditPanel({
  projectId,
  shown,
  onSaved,
  onClose,
}: {
  projectId: number;

  shown: GuiBookmark | undefined;
  onSaved: (label: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewContent.bookmark");
  const tEdit = useTranslations("gameViewEdit");

  const rows = useSWR(
    ["game-view-bookmarks-db", projectId],
    () => bookmarksApi.list(projectId),
    { revalidateOnFocus: false },
  );
  const row = bookmarkRowFor(rows.data, shown);

  const projectReady = useWorkbenchProject(projectId);

  return (
    <GamePanel
      title={shown?.label || shown?.name || t("fallbackName")}
      kind={t("kind", { id: row?.bookmark_id ?? shown?.name ?? "…" })}
      source={
        rows.isLoading ? undefined : row ? "项目" : shown ? "原版" : undefined
      }
      workbenchHref={
        row ? `/project/${projectId}/bookmarks?bm=${row.id}` : null
      }

      wide={row ? "workbench" : true}
      onClose={onClose}
    >
      {rows.isLoading && (
        <p className="mt-4 text-micro text-muted-foreground">{t("loading")}</p>
      )}
      {!rows.isLoading && shown && !row && (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("notInProject")}
        </p>
      )}
      {row &&
        (projectReady ? (
          <div className="-mx-4 mt-2">
            <ProjectBookmark
              key={row.id}
              projectId={projectId}
              row={row}
              rows={rows.data ?? []}
              shown={shown}
              reload={() => rows.mutate()}
              onSaved={onSaved}
            />
          </div>
        ) : (
          <p className="mt-4 text-micro text-muted-foreground">
            {tEdit("panel.loadingProject")}
          </p>
        ))}
    </GamePanel>
  );
}

function asDrafted(row: BookmarkRow): BookmarkRow {
  return { ...row, ...bookmarkRowToDraft(row) };
}

function ProjectBookmark({
  projectId,
  row,
  rows,
  shown,
  reload,
  onSaved,
}: {
  projectId: number;

  row: BookmarkRow;
  rows: readonly BookmarkRow[];
  shown: GuiBookmark | undefined;

  reload: () => Promise<BookmarkRow[] | undefined>;
  onSaved: (label: string) => void;
}) {
  const t = useTranslations("gameViewContent.bookmark");
  const [draft, setDraft] = useState<BookmarkDraft>(() =>
    bookmarkRowToDraft(row),
  );
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [written, setWritten] = useState<BookmarkRow>(row);
  const { diagnostics } = useSaveDiagnostics(
    `${projectId}:bookmark:${row.id}`,
    written,
  );
  const workbenchHref = `/project/${projectId}/bookmarks?bm=${row.id}`;

  const edit = useCallback(
    (change: (current: BookmarkDraft) => BookmarkDraft) => {
      setSaved(false);
      setDraft(change);
    },
    [],
  );

  const reloadQuietly = () => reload().catch(() => undefined);

  const save = async () => {
    const problem = dateProblem(draft.date);
    if (problem) {
      setSaved(false);
      setError(t(`problem.${problem}`));
      return;
    }
    const sent = draft;
    const id = row.id;
    const base = asDrafted(row);
    const keys = changedKeys(base, withoutEditRevisions(sent));
    const before = pick(base, keys);
    setSaving(true);
    setSaved(false);
    setError(null);
    try {
      const answer = await bookmarksApi.update(projectId, id, sent);
      const title = bookmarkTitle(answer, shown, t("fallbackName"));
      recordChange({
        projectId,
        title,
        kind: t("kind", { id: answer.bookmark_id }),
        workbenchHref,
        undo: keys.length
          ? rowUndo<BookmarkRow>({
              before,
              after: pick(asDrafted(answer), keys),
              load: async () =>
                asDrafted(await bookmarksApi.get(projectId, id)),

              save: (back, now) =>
                bookmarksApi.update(projectId, id, back, now.edit_revision),
            })
          : undefined,
      });
      setWritten(answer);

      setDraft((current) =>
        current === sent
          ? bookmarkRowToDraft(answer)
          : { ...current, edit_revision: answer.edit_revision },
      );
      setSaved(true);
      await reloadQuietly();
      onSaved(title);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const pictureUploaded = async () => {
    const fresh = (await reloadQuietly())?.find((item) => item.id === row.id);
    if (fresh)
      setDraft((current) => ({
        ...current,
        edit_revision: fresh.edit_revision,
      }));
    const title = bookmarkTitle(fresh ?? row, shown, t("fallbackName"));
    recordChange({
      projectId,
      title: t("pictureLog", { name: title }),
      kind: t("kind", { id: row.bookmark_id }),
      workbenchHref,
    });
    onSaved(title);
  };

  return (
    <BookmarkEditor
      projectId={projectId}
      activeId={row.id}
      draft={draft}
      onEdit={edit}
      rows={rows}
      saving={saving}
      saved={saved}
      error={error}
      diagnostics={diagnostics}
      onSave={() => void save()}
      onPictureUploaded={() => void pictureUploaded()}
    />
  );
}
