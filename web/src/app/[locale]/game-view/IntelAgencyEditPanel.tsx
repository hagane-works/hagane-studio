"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { intelligenceAgenciesApi } from "@/lib/api";
import type { IntelligenceAgencyRow } from "@/lib/api";
import {
  IntelAgencyEditor,
  type IntelAgencySaveData,
} from "@/components/workbench/IntelAgencyEditor";
import { BUTTON, FIELD, GamePanel, PanelSave, SCRIPT } from "./GamePanel";
import type { GuiIntelInsignia } from "./IntelAgencyLayer";
import { fileUndo, pick, recordChange, rowUndo } from "./changeLog";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

export interface InsigniaDraft {
  names?: string;
  available?: string;
  default?: string;
}

export function namesText(item: GuiIntelInsignia): string {
  return item.raw_names.join("\n");
}

export function insigniaChanges(
  item: GuiIntelInsignia,
  draft: InsigniaDraft,
): { names?: string[]; available?: string; default?: string } | null {
  const out: { names?: string[]; available?: string; default?: string } = {};
  if (draft.names !== undefined && draft.names !== namesText(item)) {
    const names = draft.names
      .split("\n")
      .map((name) => name.trim())
      .filter(Boolean);
    if (!names.length || names.some((name) => /["{}#]/.test(name))) return null;
    out.names = names;
  }
  if (
    draft.available !== undefined &&
    draft.available.trim() !== item.available_script
  )
    out.available = draft.available.trim();
  if (
    draft.default !== undefined &&
    draft.default.trim() !== item.default_script
  )
    out.default = draft.default.trim();
  return out;
}

export function insigniaRow(
  rows: readonly IntelligenceAgencyRow[] | undefined,
  item: GuiIntelInsignia,
): IntelligenceAgencyRow | null {
  const want = item.raw_names.map((name) => name.trim());
  return (
    rows?.find((row) => {
      if (row.source === "imported" || row.picture !== item.picture)
        return false;
      const names = row.names?.length
        ? row.names
        : row.agency_id
          ? [row.agency_id]
          : [];
      return (
        names.length === want.length &&
        names.every((name, at) => name.trim() === want[at])
      );
    }) ?? null
  );
}

async function loadAgency(
  projectId: number,
  id: number,
): Promise<IntelligenceAgencyRow | undefined> {
  return (await intelligenceAgenciesApi.list(projectId)).agencies.find(
    (row) => row.id === id,
  );
}

export function IntelAgencyEditPanel({
  projectId,
  tag,
  item,
  typedName,
  onSaved,
  onClose,
}: {
  projectId: number;
  tag: string;
  item: GuiIntelInsignia;

  typedName: string;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewResearch.insignia");
  const tIntel = useTranslations("gameViewResearch.intel");
  const tPanel = useTranslations("gameViewEdit.panel");
  const rows = useSWR(
    ["game-view-intel-agencies-db", projectId],
    async () => (await intelligenceAgenciesApi.list(projectId)).agencies,
    { revalidateOnFocus: false },
  );

  const [rowId, setRowId] = useState<number | null>(null);
  const row =
    rows.data?.find((agency) => agency.id === rowId) ??
    insigniaRow(rows.data, item);
  if (row && row.id !== rowId) setRowId(row.id);
  const projectReady = useWorkbenchProject(projectId);
  const [newName, setNewName] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const label = item.names[0] ?? item.picture;
  const kind = t("kind", { picture: item.picture });
  const workbenchHref = row ? `/project/${projectId}/intelligence` : null;
  const post = async (
    route: string,
    body: unknown,
    message: string,
    after: (out: unknown) => void,
  ) => {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(
        `/api/go/projects/${projectId}/gui-render/${route}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      );
      const out = await response.json().catch(() => ({}));
      if (!response.ok)
        throw new Error(
          (out as { detail?: string }).detail || `HTTP ${response.status}`,
        );
      after(out);
      onSaved(message);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const saveRow = async (data: IntelAgencySaveData) => {
    if (!row) return;
    const id = row.id;
    const keys = changedKeys<IntelligenceAgencyRow>(row, data);
    const before = pick(row, keys);
    const saved = await intelligenceAgenciesApi.update(projectId, id, data);
    const title = saved.names?.[0] || saved.agency_id || label;
    recordChange({
      projectId,
      title,
      kind,
      workbenchHref,
      undo: keys.length
        ? rowUndo({
            before,
            after: pick(saved, keys),
            load: () => loadAgency(projectId, id),
            save: (back) => intelligenceAgenciesApi.update(projectId, id, back),
          })
        : undefined,
    });
    onSaved(t("saved", { name: title }));
    await rows.mutate();
  };

  const logoUploaded = () => {
    recordChange({
      projectId,
      title: t("logoLog", { name: label }),
      kind,
      workbenchHref,
    });
    onSaved(t("logoUploaded", { name: label }));
  };

  const name = newName ?? typedName;
  const create = () => {
    if (!name.trim()) return;
    void post(
      "intel-agency-create",
      { tag, picture: item.picture, name: name.trim() },
      t("created", { tag, name: name.trim() }),
      () => {
        recordChange({
          projectId,
          title: t("createdLog", { name: name.trim() }),
          kind,
        });
        setNewName(null);
      },
    );
  };

  return (
    <GamePanel
      title={label}
      kind={kind}
      detail={t("detail", { file: item.file, n: item.ordinal + 1 })}
      source={row ? "项目" : undefined}
      workbenchHref={workbenchHref}

      wide={row ? "workbench" : false}
      onClose={onClose}
    >
      {rows.isLoading ? (
        <p className="mt-4 text-micro text-muted-foreground">
          {tIntel("loading")}
        </p>
      ) : row ? (
        projectReady ? (
          <div className="-mx-4 mt-2">
            <IntelAgencyEditor
              key={row.id}
              projectId={projectId}
              agency={row}
              onSave={saveRow}
              onLogoUploaded={logoUploaded}
            />
          </div>
        ) : (
          <p className="mt-4 text-micro text-muted-foreground">
            {tPanel("loadingProject")}
          </p>
        )
      ) : (
        <InsigniaFields
          projectId={projectId}
          item={item}
          label={label}
          kind={kind}
          busy={busy}
          post={post}
        />
      )}

      <div className="mt-3 border-t border-border pt-2">
        <label className="block text-xs">
          {t("newLabel", { tag })}
          <input
            aria-label="new insignia name"
            className={FIELD}
            value={name}
            onChange={(e) => setNewName(e.target.value)}
          />
        </label>
        <button
          type="button"
          className={`${BUTTON} mt-2`}
          disabled={busy || !name.trim()}
          onClick={create}
        >
          {t("create")}
        </button>
      </div>
      {error && <p className="mt-2 text-xs text-destructive">{error}</p>}
    </GamePanel>
  );
}

function InsigniaFields({
  projectId,
  item,
  label,
  kind,
  busy,
  post,
}: {
  projectId: number;
  item: GuiIntelInsignia;
  label: string;
  kind: string;
  busy: boolean;
  post: (
    route: string,
    body: unknown,
    message: string,
    after: (out: unknown) => void,
  ) => Promise<void>;
}) {
  const t = useTranslations("gameViewResearch.insignia");
  const [draft, setDraft] = useState<InsigniaDraft>({});
  const changes = insigniaChanges(item, draft);
  const changed = !!changes && Object.keys(changes).length > 0;
  const save = () => {
    if (!changes || !changed) return;
    void post(
      "intel-agency-save",
      {
        file: item.file,
        ordinal: item.ordinal,
        picture: item.picture,
        old_names: item.raw_names,
        ...changes,
      },
      t("saved", { name: label }),
      (out) => {
        recordChange({
          projectId,
          title: label,
          kind,
          undo: fileUndo(projectId, out),
        });
        setDraft({});
      },
    );
  };
  return (
    <>
      <div className="mt-3 space-y-2">
        <label className="block text-xs">
          {t("names")}
          <textarea
            aria-label="names"
            className={FIELD}
            rows={3}
            value={draft.names ?? namesText(item)}
            onChange={(e) => setDraft({ ...draft, names: e.target.value })}
          />
        </label>
        <label className="block text-xs">
          {t("available")}
          <textarea
            aria-label="available"
            className={SCRIPT}
            rows={3}
            placeholder={t("availablePlaceholder")}
            value={draft.available ?? item.available_script}
            onChange={(e) => setDraft({ ...draft, available: e.target.value })}
          />
        </label>
        <label className="block text-xs">
          {t("default")}
          <textarea
            aria-label="default"
            className={SCRIPT}
            rows={3}
            placeholder={t("defaultPlaceholder")}
            value={draft.default ?? item.default_script}
            onChange={(e) => setDraft({ ...draft, default: e.target.value })}
          />
        </label>
        <p className="text-micro text-muted-foreground">
          {t("conditionsNote")}
        </p>
      </div>
      <PanelSave
        onSave={save}
        canSave={changed}
        busy={busy}
        onRevert={changed ? () => setDraft({}) : undefined}
        problem={changes === null ? t("problem") : null}
        note={t("note")}
      />
    </>
  );
}
