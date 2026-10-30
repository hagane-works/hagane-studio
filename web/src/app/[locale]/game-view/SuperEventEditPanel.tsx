"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { superEventsApi, type SuperEventRow } from "@/lib/api/scenario";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import {
  SuperEventEditor,
  superEventToDraft,
  type SuperEventDraft,
} from "@/components/workbench/SuperEventEditor";
import { GamePanel } from "./GamePanel";
import { pick, recordChange, rowUndo } from "./changeLog";
import { superEventTexts } from "./superEventModel";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

function useSuperEventKind(): (row: SuperEventRow) => string {
  const tKind = useTranslations("gameViewContent.superEvent");
  return (row) =>
    tKind("kind", {
      number: String(row.number).padStart(4, "0"),
      tag: row.country_tag || tKind("projectCountry"),
    });
}

export function SuperEventEditPanel({
  projectId,
  row,
  onSaved,
  onDeleted,
  onClose,
}: {
  projectId: number;
  row: SuperEventRow;

  onSaved: (message: string, saved: SuperEventRow) => void;

  onDeleted?: (message: string) => void;
  onClose: () => void;
}) {
  const tEdit = useTranslations("gameViewEdit");
  const superEventKind = useSuperEventKind();

  const projectReady = useWorkbenchProject(projectId);
  return (
    <GamePanel
      title={superEventTexts(row).title}
      kind={superEventKind(row)}
      source="项目"
      workbenchHref={`/project/${projectId}/super-events?se=${row.id}`}

      wide="workbench"
      onClose={onClose}
    >
      {projectReady ? (
        <div className="-mx-4 mt-2">
          <ProjectSuperEvent
            key={row.id}
            projectId={projectId}
            row={row}
            onSaved={onSaved}
            onDeleted={onDeleted}
          />
        </div>
      ) : (
        <p className="mt-4 text-micro text-muted-foreground">
          {tEdit("panel.loadingProject")}
        </p>
      )}
    </GamePanel>
  );
}

function ProjectSuperEvent({
  projectId,
  row,
  onSaved,
  onDeleted,
}: {
  projectId: number;
  row: SuperEventRow;
  onSaved: (message: string, saved: SuperEventRow) => void;
  onDeleted?: (message: string) => void;
}) {
  const t = useTranslations("superEventsDialog");
  const tPanel = useTranslations("gameViewContent.superEvent");
  const tEdit = useTranslations("gameViewEdit");
  const superEventKind = useSuperEventKind();
  const confirmDialog = useConfirm();

  const [base, setBase] = useState<SuperEventRow>(row);
  const [draft, setDraft] = useState<SuperEventDraft>(() =>
    superEventToDraft(row),
  );
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pictureBust, setPictureBust] = useState(0);
  const workbenchHref = `/project/${projectId}/super-events?se=${row.id}`;

  const flashSaved = () => {
    setSaved(true);
    setTimeout(() => setSaved(false), 1500);
  };

  const save = async () => {
    const id = base.id;

    const savedDraft = superEventToDraft(base);
    const keys = changedKeys(savedDraft, draft);
    const before = pick(savedDraft, keys);
    setSaving(true);
    setError(null);
    try {
      const next = await superEventsApi.update(projectId, id, draft);
      const title = superEventTexts(next).title;
      recordChange({
        projectId,
        title,
        kind: superEventKind(next),
        workbenchHref,
        undo: keys.length
          ? rowUndo<SuperEventRow>({
              before,
              after: pick(next, keys),
              load: () => superEventsApi.getSuperEvent(projectId, id),
              save: (back) => superEventsApi.update(projectId, id, back),
            })
          : undefined,
      });
      setBase(next);
      setDraft(superEventToDraft(next));
      flashSaved();
      onSaved(tPanel("saved", { title }), next);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  const commitPicture = async (file: File) => {
    const next = await superEventsApi.uploadPicture(projectId, base.id, file);

    recordChange({
      projectId,
      title: superEventTexts(next).title,
      kind: superEventKind(next),
      workbenchHref,
    });
    setBase(next);
    setPictureBust((v) => v + 1);
    flashSaved();
    onSaved(tPanel("pictureSaved"), next);
  };

  const upload = async (file: File) => {
    setUploading(true);
    setError(null);
    try {
      await commitPicture(file);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setUploading(false);
    }
  };

  const remove = async () => {
    if (!onDeleted) return;
    if (!(await confirmDialog({ body: t("confirmDelete"), danger: true })))
      return;
    const title = superEventTexts(base).title;
    setSaving(true);
    setError(null);
    try {
      await superEventsApi.remove(projectId, base.id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setSaving(false);
      return;
    }
    recordChange({
      projectId,
      title: tEdit("person.deleteLog", { name: title }),
      kind: superEventKind(base),
      workbenchHref: `/project/${projectId}/super-events`,
    });
    setSaving(false);
    onDeleted(tPanel("deleted", { title }));
  };

  return (
    <SuperEventEditor
      projectId={projectId}
      identity={base.id}
      draft={draft}
      onChange={(key, value) =>
        setDraft((current) => ({ ...current, [key]: value }))
      }
      row={base}
      pictureBust={pictureBust}
      saving={saving}
      uploading={uploading}
      saved={saved}
      error={error}
      onSave={() => void save()}
      onDelete={onDeleted ? () => void remove() : undefined}
      onUpload={(file) => void upload(file)}
    />
  );
}
