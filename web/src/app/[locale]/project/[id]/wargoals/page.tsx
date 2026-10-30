"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import { FileCode2, Plus } from "lucide-react";

import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import {
  WarGoalEditor,
  type WargoalSaveStatus,
} from "@/components/workbench/WarGoalEditor";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { useActiveCountry } from "@/lib/store/projectStore";
import { Button } from "@/components/ui/button";
import { wargoalsApi } from "@/lib/api/wargoals";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";
import type { WargoalType } from "@/types/wargoal";

import {
  buildWargoalPreview,
  freshWargoalDraft,
  toWargoalDraft,
  toWargoalWrite,
  type WargoalDraft,
} from "./wargoalForm";

export default function WargoalsPage() {
  const params = useParams<{ id: string }>();
  const projectId = Number(params.id);
  const t = useTranslations("wargoals");
  const tShell = useTranslations("workbenchShell");
  const [rows, setRows] = useState<WargoalType[]>([]);
  const activeCountry = useActiveCountry();
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [draft, setDraft] = useState<WargoalDraft | null>(null);
  const [savedDraft, setSavedDraft] = useState<WargoalDraft | null>(null);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<WargoalSaveStatus>("idle");
  const [error, setError] = useState("");
  const [listLoadFailed, setListLoadFailed] = useState(false);
  const seededIdRef = useRef<number | null>(null);
  const selectedIdRef = useRef<number | null>(null);
  useEffect(() => {
    selectedIdRef.current = selectedId;
  }, [selectedId]);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const result = await wargoalsApi.list(projectId);
      setRows(result.wargoals);
      setListLoadFailed(false);
      setSelectedId((current) => {
        if (
          current != null &&
          result.wargoals.some((row) => row.id === current)
        ) {
          return current;
        }
        return result.wargoals[0]?.id ?? null;
      });
    } catch {
      setListLoadFailed(true);
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (selectedId == null) {
      seededIdRef.current = null;
      if (draft?.id != null) {
        setDraft(null);
        setSavedDraft(null);
      }
      return;
    }
    const row = rows.find((item) => item.id === selectedId);
    if (!row) return;
    if (seededIdRef.current === selectedId) return;
    const next = toWargoalDraft(row);
    seededIdRef.current = selectedId;
    setDraft(next);
    setSavedDraft(next);
    setStatus("idle");
    setError("");
  }, [selectedId, rows, draft?.id]);

  const filteredRows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) return rows;
    return rows.filter((row) =>
      [row.wargoal_key, row.war_name, row.source_file]
        .join(" ")
        .toLowerCase()
        .includes(needle),
    );
  }, [rows, search]);

  const handleNew = () => {
    const next = freshWargoalDraft(rows.map((row) => row.wargoal_key));
    setSelectedId(null);
    setDraft(next);
    setSavedDraft(null);
    setStatus("idle");
    setError("");
  };

  const handleSave = async () => {
    if (!draft || !draft.wargoal_key.trim()) return;
    setStatus("saving");
    setError("");
    try {
      const payload = toWargoalWrite(draft);
      const saved =
        draft.id == null
          ? await wargoalsApi.create(projectId, payload)
          : await wargoalsApi.update(projectId, draft.id, payload);
      setRows((current) => {
        const found = current.some((row) => row.id === saved.id);
        const next = found
          ? current.map((row) => (row.id === saved.id ? saved : row))
          : [...current, saved];
        return next.sort((a, b) => a.wargoal_key.localeCompare(b.wargoal_key));
      });
      const nextDraft = toWargoalDraft(saved);

      if (
        selectedIdRef.current === saved.id ||
        (selectedIdRef.current === null && draft.id == null)
      ) {
        seededIdRef.current = saved.id;
        setDraft(nextDraft);
        setSavedDraft(nextDraft);
        setSelectedId(saved.id);
      }
      setStatus("saved");
    } catch {
      setStatus("error");
      setError(t("saveFailed"));
    }
  };

  const handleDelete = async () => {
    if (!draft?.id) return;
    if (!window.confirm(t("deleteConfirm", { id: draft.wargoal_key }))) return;
    try {
      await wargoalsApi.delete(projectId, draft.id);
      const remaining = rows.filter((row) => row.id !== draft.id);
      setRows(remaining);
      setSelectedId(remaining[0]?.id ?? null);
      setDraft(null);
      setSavedDraft(null);
    } catch {
      setStatus("error");
      setError(t("deleteFailed"));
    }
  };

  const listPanel = (
    <div>
      <WorkbenchListHeader
        title={t("title")}
        count={filteredRows.length}
        totalCount={rows.length}
        searchValue={search}
        onSearchChange={setSearch}
        searchPlaceholder={t("searchPlaceholder")}
        action={
          <Button size="sm" onClick={handleNew}>
            <Plus aria-hidden />
            {t("newButton")}
          </Button>
        }
      />
      <div className="space-y-1 px-3 pb-3">
        {filteredRows.map((row) => (
          <button
            key={row.id}
            type="button"
            onClick={() => {
              if (row.id === selectedId || confirmDiscardIfDirty())
                setSelectedId(row.id);
            }}
            className={`ui-interactive ui-focus-ring w-full rounded-md px-3 py-2 text-left hover:bg-sidebar-accent/50 ${
              row.id === selectedId
                ? "bg-sidebar-accent text-sidebar-primary"
                : "text-foreground"
            }`}
          >
            <span className="block truncate font-mono text-xs">
              {row.wargoal_key}
            </span>
            <span className="mt-1 block truncate text-xs text-muted-foreground">
              {row.war_name || t("noWarName")}
            </span>
          </button>
        ))}
        {!loading && filteredRows.length === 0 && rows.length > 0 && (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {tShell("noMatchesGeneric")}
          </p>
        )}
      </div>
    </div>
  );

  const editorPanel = !draft ? (
    <div className="flex h-full items-center justify-center p-6">
      {!loading && listLoadFailed && rows.length === 0 ? (
        <WorkbenchLoadError onRetry={() => void load()} />
      ) : (
        <EditorEmptyState
          variant={rows.length === 0 ? "hero" : "compact"}
          icon={<FileCode2 className="h-8 w-8 text-muted-foreground/60" />}
          title={
            loading
              ? t("loading")
              : rows.length === 0
                ? t("emptyTitle")
                : t("selectTitle")
          }
          description={
            rows.length === 0 ? t("emptyDescription") : t("selectDescription")
          }
          actions={
            rows.length === 0 ? (
              <Button onClick={handleNew}>{t("newButton")}</Button>
            ) : undefined
          }
        />
      )}
    </div>
  ) : (
    <WarGoalEditor
      draft={draft}
      savedDraft={savedDraft}
      onChange={setDraft}
      status={status}
      error={error}
      onSave={handleSave}
      onDelete={handleDelete}
    />
  );

  const previewPanel = draft ? (
    <div className="p-4">
      <p className="mb-3 text-xs text-muted-foreground">
        {t("previewDescription")}
      </p>
      <pre className="overflow-x-auto whitespace-pre rounded-md border border-border bg-muted p-3 font-mono text-xs text-foreground">
        {buildWargoalPreview(draft)}
      </pre>
    </div>
  ) : (
    <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
      {tShell("previewUnavailableGeneric")}
    </div>
  );

  return (
    <div className="h-full min-w-0 overflow-hidden">
      <WorkbenchShell
        gameViewHref={gameViewHref(projectId, {
          panel: "countrydiplomacyview",
          tag: activeCountry,
        })}
        listPanel={listPanel}
        editorPanel={editorPanel}
        previewPanel={previewPanel}
        previewWidth="w-96"
      />
    </div>
  );
}
