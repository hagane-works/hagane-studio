"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Braces, Plus, Search, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { ModifierDefinitionEditor } from "@/components/workbench/modifier-definitions/ModifierDefinitionEditor";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import {
  formatModifierDefinitionScript,
  modifierDefinitionsApi,
  modifierDefinitionToDraft,
  newModifierDefinitionDraft,
  type ModifierDefinition,
  type ModifierDefinitionDraft,
} from "@/lib/api/modifierDefinitions";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import {
  beginDraftSave,
  confirmDraftSelection,
  editDraftSession,
  emptyDraftSession,
  openDraftSession,
  requestDraftSelection,
  resolveDraftSave,
  stableDraftFingerprint,
  type DraftSession,
} from "@/lib/workbench-state/draftSession";

type SaveStatus = "idle" | "success" | "error";

export default function ModifierDefinitionsPage() {
  const params = useParams<{ id: string }>();
  const projectId = Number(params.id);
  const t = useTranslations("modifierDefinitions");
  const [rows, setRows] = useState<ModifierDefinition[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [draft, setDraft] = useState<ModifierDefinitionDraft | null>(null);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");
  const [saveStatus, setSaveStatus] = useState<SaveStatus>("idle");
  const ownership =
    useRef<DraftSession<ModifierDefinitionDraft>>(emptyDraftSession());
  const selectedIdRef = useRef<number | null>(null);
  const loadGeneration = useRef(0);

  const selected = useMemo(
    () => rows.find((row) => row.id === selectedId) ?? null,
    [rows, selectedId],
  );
  const dirty = Boolean(
    selected &&
    draft &&
    JSON.stringify(draft) !==
      JSON.stringify(modifierDefinitionToDraft(selected)),
  );
  useRegisterDirty("modifier-definitions", dirty);

  const fingerprint = useCallback(
    (value: ModifierDefinitionDraft) => stableDraftFingerprint(value),
    [],
  );
  const openRow = useCallback(
    (row: ModifierDefinition) => {
      const next = modifierDefinitionToDraft(row);
      ownership.current = openDraftSession(
        ownership.current,
        String(row.id),
        fingerprint(next),
        next,
      );
      selectedIdRef.current = row.id;
      setSelectedId(row.id);
      setDraft(next);
    },
    [fingerprint],
  );
  const edit = useCallback(
    (next: ModifierDefinitionDraft) => {
      ownership.current = editDraftSession(
        ownership.current,
        next,
        fingerprint(next),
      );
      setDraft(next);
      setSaveStatus("idle");
    },
    [fingerprint],
  );

  const loadRows = useCallback(async () => {
    if (!Number.isInteger(projectId) || projectId <= 0) {
      setError(t("invalidProject"));
      setLoading(false);
      return;
    }
    const request = ++loadGeneration.current;
    setLoading(true);
    try {
      const result = await modifierDefinitionsApi.list(projectId);
      if (request !== loadGeneration.current) return;
      setRows(result.modifier_definitions);
      const selectedRow =
        result.modifier_definitions.find(
          (row) => row.id === selectedIdRef.current,
        ) ?? null;
      const first = selectedRow ?? result.modifier_definitions[0] ?? null;
      if (first && !selectedRow) openRow(first);
      if (!first) {
        selectedIdRef.current = null;
        ownership.current = emptyDraftSession();
        setSelectedId(null);
        setDraft(null);
      }
    } finally {
      setLoading(false);
    }
  }, [openRow, projectId, t]);

  const { error: listLoadError, mutate: retryList } = useSWR(
    projectId ? ["modifier-definitions/list", projectId] : null,
    loadRows,
  );

  const filteredRows = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return rows;
    return rows.filter((row) => row.modifier_id.toLowerCase().includes(needle));
  }, [query, rows]);

  const chooseRow = (row: ModifierDefinition) => {
    const next = modifierDefinitionToDraft(row);
    const requested = requestDraftSelection(
      ownership.current,
      String(row.id),
      fingerprint(next),
      next,
    );
    if (
      requested.kind === "confirm-discard" &&
      !window.confirm(t("discardConfirm"))
    )
      return;
    ownership.current =
      requested.kind === "selected"
        ? requested.session
        : confirmDraftSelection(
            ownership.current,
            String(row.id),
            fingerprint(next),
            next,
          );
    selectedIdRef.current = row.id;
    setSelectedId(row.id);
    setDraft(next);
    setSaveStatus("idle");
    setError("");
  };

  const nextNewId = () => {
    const existing = new Set(rows.map((row) => row.modifier_id));
    if (!existing.has("new_modifier")) return "new_modifier";
    let suffix = 2;
    while (existing.has(`new_modifier_${suffix}`)) suffix += 1;
    return `new_modifier_${suffix}`;
  };

  const createRow = async () => {
    if (dirty && !window.confirm(t("discardConfirm"))) return;
    setCreating(true);
    setError("");
    try {
      const created = await modifierDefinitionsApi.create(
        projectId,
        newModifierDefinitionDraft(nextNewId()),
      );
      setRows((current) => [...current, created]);
      openRow(created);
      setSaveStatus("success");
    } catch {
      setError(t("createFailed"));
    } finally {
      setCreating(false);
    }
  };

  const saveRow = async () => {
    if (!selected || !draft) return;
    const save = beginDraftSave(ownership.current);
    if (!save) return;
    setSaving(true);
    setError("");
    setSaveStatus("idle");
    try {
      const updated = await modifierDefinitionsApi.update(
        projectId,
        selected.id,
        draft,
      );
      setRows((current) =>
        current.map((row) => (row.id === updated.id ? updated : row)),
      );
      const saved = modifierDefinitionToDraft(updated);
      ownership.current = resolveDraftSave(
        ownership.current,
        save,
        fingerprint(saved),
        saved,
      );
      if (ownership.current.identity === String(updated.id))
        setDraft(ownership.current.draft);
      setSaveStatus("success");
    } catch {
      setError(t("saveFailed"));
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const deleteRow = async (row: ModifierDefinition) => {
    if (!window.confirm(t("deleteConfirm", { id: row.modifier_id }))) return;
    setError("");
    try {
      await modifierDefinitionsApi.remove(projectId, row.id);
      const remaining = rows.filter((item) => item.id !== row.id);
      setRows(remaining);
      if (selectedId === row.id) {
        const next = remaining[0] ?? null;
        if (next) openRow(next);
        else {
          selectedIdRef.current = null;
          ownership.current = emptyDraftSession();
          setSelectedId(null);
          setDraft(null);
        }
        setSaveStatus("idle");
      }
    } catch {
      setError(t("deleteFailed"));
    }
  };

  const listPanel = (
    <div className="flex min-h-full flex-col p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-base font-semibold tracking-tight">
              {t("title")}
            </h1>
            <span className="rounded-full bg-card px-2 py-0.5 text-xs text-muted-foreground">
              {rows.length}
            </span>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">{t("intro")}</p>
        </div>
        <Button size="sm" onClick={createRow} disabled={creating}>
          <Plus className="h-4 w-4" />
          {creating ? t("creating") : t("newButton")}
        </Button>
      </div>

      <div className="relative mt-4">
        <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          className="pl-9"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("searchPlaceholder")}
          aria-label={t("searchPlaceholder")}
        />
      </div>

      {error && (
        <div className="mt-3 rounded-md border border-destructive/20 bg-destructive/10 p-3 text-xs text-destructive">
          {error}
        </div>
      )}

      <div className="mt-4 space-y-1">
        {loading ? (
          <div className="animate-pulse-ring rounded-md border border-accent p-4 text-xs text-muted-foreground">
            {t("loading")}
          </div>
        ) : filteredRows.length === 0 ? (
          <p className="rounded-md border border-dashed border-border p-4 text-center text-xs text-muted-foreground">
            {query ? t("noSearchResults") : t("emptyList")}
          </p>
        ) : (
          filteredRows.map((row) => (
            <div
              key={row.id}
              className={`ui-card group flex items-center gap-1 p-1 ${
                selectedId === row.id ? "border-border bg-card" : ""
              }`}
            >
              <button
                type="button"
                className="ui-focus-ring min-w-0 flex-1 rounded-md px-3 py-2 text-left"
                onClick={() => chooseRow(row)}
              >
                <span className="block truncate font-mono text-sm">
                  {row.modifier_id}
                </span>
                <span className="mt-1 block truncate text-xs text-muted-foreground">
                  {t(`valueOptions.${row.value_type}`)} ·{" "}
                  {t("categoryCount", {
                    count: row.categories.length,
                  })}
                </span>
              </button>
              <Button
                variant="ghost"
                size="icon"
                className="opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                onClick={() => void deleteRow(row)}
                aria-label={t("deleteAria", { id: row.modifier_id })}
                title={t("deleteAria", { id: row.modifier_id })}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))
        )}
      </div>
    </div>
  );

  const emptyState = (
    <div className="flex h-full items-center justify-center p-6">
      <EditorEmptyState
        variant={rows.length === 0 ? "hero" : "compact"}
        icon={<Braces className="h-10 w-10 text-muted-foreground/60" />}
        title={rows.length === 0 ? t("emptyTitle") : t("selectTitle")}
        description={
          rows.length === 0 ? t("emptyDescription") : t("selectDescription")
        }
        actions={
          rows.length === 0 ? (
            <Button onClick={createRow} disabled={creating}>
              {t("createFirst")}
            </Button>
          ) : undefined
        }
      />
    </div>
  );

  const preview = draft ? (
    <div className="space-y-4 p-4">
      <div>
        <h2 className="text-sm font-semibold">{t("previewTitle")}</h2>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("previewDescription")}
        </p>
      </div>
      <pre className="overflow-x-auto rounded-lg border border-border bg-background p-4 font-mono text-xs leading-normal">
        {formatModifierDefinitionScript(draft, {
          source: dirty ? "created" : selected?.source,
          explicitFields: selected?.explicit_fields,
        })}
      </pre>
      <div className="rounded-lg border border-border bg-muted/30 p-4">
        <p className="text-xs font-medium">{t("usageTitle")}</p>
        <code className="mt-2 block font-mono text-xs text-muted-foreground">
          {`modifier@${draft.modifier_id}`}
        </code>
      </div>
    </div>
  ) : (
    <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
      {t("previewEmpty")}
    </div>
  );

  if (listLoadError && rows.length === 0) {
    return (
      <WorkbenchLoadError
        onRetry={() => retryList(undefined, { throwOnError: false })}
      />
    );
  }

  return (
    <WorkbenchShell
      listPanel={listPanel}
      editorPanel={
        draft && selected ? (
          <ModifierDefinitionEditor
            value={draft}
            onChange={(next) => {
              edit(next);
            }}
            onSave={() => void saveRow()}
            onDelete={() => void deleteRow(selected)}
            saving={saving}
            dirty={dirty}
            saveStatus={saveStatus}
          />
        ) : (
          emptyState
        )
      }
      previewPanel={preview}
    />
  );
}
