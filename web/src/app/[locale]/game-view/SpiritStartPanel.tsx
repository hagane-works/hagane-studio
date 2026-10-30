"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { HistoryFileEditor } from "@/components/workbench/HistoryFileEditor";
import { ideasApi } from "@/lib/api/content";
import { historyFilesApi, type HistoryFileRow } from "@/lib/api/scenario";
import { FIELD, GamePanel, PRIMARY } from "./GamePanel";
import { recordChange } from "./changeLog";
import { useHistoryFileDraft } from "./historyFileEdit";
import { useHistoryRow } from "./startStateEdit";
import { newSpiritRow, spiritToken, withStartSpirit } from "./spiritStart";

export function SpiritStartPanel({
  projectId,
  tag,
  onEdit,
  onChanged,
  onClose,
}: {
  projectId: number;
  tag: string;
  date: string | null;

  onEdit: (spirit: { token: string; label?: string }) => void;

  onChanged: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewEdit");
  const row = useHistoryRow(projectId, tag, true);

  return (
    <GamePanel
      title={t("spirit.startTitle")}
      kind={t("spirit.historyKind", { tag })}
      source={row.isLoading ? undefined : row.data ? "项目" : "原版"}
      workbenchHref={
        row.data
          ? `/project/${projectId}/history-files?hf=${row.data.id}`
          : null
      }

      wide={row.data ? "workbench" : true}
      onClose={onClose}
    >
      {row.isLoading && (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("spirit.loadingHistory")}
        </p>
      )}
      {!row.isLoading && !row.data && (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("panel.notInProjectHistory")}
        </p>
      )}
      {row.data && (
        <div className="-mx-4 mt-2">
          <ProjectStartSpirits
            key={row.data.id}
            projectId={projectId}
            tag={tag}
            row={row.data}
            reload={() => row.mutate()}
            onEdit={onEdit}
            onChanged={onChanged}
          />
        </div>
      )}
    </GamePanel>
  );
}

function ProjectStartSpirits({
  projectId,
  tag,
  row,
  reload,
  onEdit,
  onChanged,
}: {
  projectId: number;
  tag: string;
  row: HistoryFileRow;
  reload: () => Promise<unknown>;
  onEdit: (spirit: { token: string; label?: string }) => void;
  onChanged: (message: string) => void;
}) {
  const t = useTranslations("gameViewEdit.spirit");
  const history = useHistoryFileDraft({
    projectId,
    row,
    title: t("startTitle"),
    kind: t("historyKind", { tag }),
    workbenchHref: `/project/${projectId}/history-files?hf=${row.id}`,
    afterSave: async () => {
      await reload().catch(() => undefined);
      onChanged(t("startSaved", { tag }));
    },
  });
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const create = async () => {
    const text = name.trim();
    if (!text || creating) return;
    setCreating(true);
    setCreateError(null);
    try {
      const created = await ideasApi.create(projectId, newSpiritRow(tag, text));
      const saved = await historyFilesApi.update(projectId, history.base.id, {
        add_ideas_extra: withStartSpirit(
          history.base.add_ideas_extra,
          created.idea_id,
        ),
      });

      recordChange({
        projectId,
        title: t("startLog", { name: text }),
        kind: t("historyKind", { tag }),
        workbenchHref: `/project/${projectId}/history-files?hf=${row.id}`,
      });
      history.adopt(saved, (current) => ({
        ...current,
        add_ideas_extra: withStartSpirit(
          current.add_ideas_extra,
          created.idea_id,
        ),
      }));
      setName("");
      await reload().catch(() => undefined);
      onChanged(t("created", { name: text, tag }));
      onEdit({ token: created.idea_id, label: text });
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : String(err));
    } finally {
      setCreating(false);
    }
  };

  return (
    <>
      <HistoryFileEditor
        projectId={projectId}
        activeId={row.id}
        section="ideas"
        draft={history.draft}
        onChange={history.onChange}
        rows={[history.base]}
        saving={history.saving}
        saved={history.saved}
        dirty={history.dirty}
        error={history.error}
        onSave={() => void history.save()}
      />

      <section className="mx-4 mb-4 space-y-2 border-t border-border pt-3">
        <label className="block">
          <span className="text-micro font-medium text-muted-foreground">
            {t("newLabel")}
          </span>
          <input
            value={name}
            disabled={creating || history.saving}
            onChange={(event) => setName(event.target.value)}
            placeholder="Rearmament Drive"
            className={FIELD}
          />
        </label>
        {name.trim() && (
          <p className="font-mono text-micro text-muted-foreground">
            {spiritToken(tag, name)}
          </p>
        )}
        <button
          type="button"
          disabled={
            creating || history.saving || !name.trim() || /["\r\n]/.test(name)
          }
          onClick={() => void create()}
          className={PRIMARY}
        >
          {creating ? t("creating") : t("createAndAdd")}
        </button>
        <p className="text-micro text-muted-foreground">{t("createNote")}</p>
        {createError && (
          <p role="alert" className="text-micro text-destructive">
            {createError}
          </p>
        )}
      </section>
    </>
  );
}
