"use client";

import type { ComponentType } from "react";
import { useCallback, useMemo, useRef, useState } from "react";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { BrainCircuit, Plus, Search, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
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
import {
  aiPlanningApi,
  aiStrategyPlanToDraft,
  aiTemplateToDraft,
  formatAiStrategyPlanScript,
  formatAiTemplateScript,
  newAiStrategyPlanDraft,
  newAiTemplateDraft,
  type AiStrategyPlan,
  type AiStrategyPlanDraft,
  type AiTemplate,
  type AiTemplateDraft,
} from "@/lib/api/aiPlanning";

import { AiStrategyPlanEditor, AiTemplateEditor } from "./AiPlanningEditors";

type SaveStatus = "idle" | "success" | "error";
interface RowMeta {
  id: number;
  source: string;
  raw_block: string;
}
interface EditorProps<Draft> {
  value: Draft;
  onChange: (value: Draft) => void;
  onSave: () => void;
  onDelete: () => void;
  saving?: boolean;
  dirty?: boolean;
  saveStatus?: SaveStatus;
}
interface PlanningConfig<Row extends RowMeta, Draft> {
  kind: "plan" | "template";
  dirtyKey: string;
  Editor: ComponentType<EditorProps<Draft>>;
  list(projectId: number): Promise<Row[]>;
  create(projectId: number, draft: Draft): Promise<Row>;
  update(projectId: number, id: number, draft: Draft): Promise<Row>;
  remove(projectId: number, id: number): Promise<void>;
  toDraft(row: Row): Draft;
  newDraft(id: string): Draft;
  rowId(row: Row): string;
  subtitle(row: Row): string;
  format(
    draft: Draft,
    options?: { source?: string; rawBlock?: string },
  ): string;
}

const PLAN_CONFIG: PlanningConfig<AiStrategyPlan, AiStrategyPlanDraft> = {
  kind: "plan",
  dirtyKey: "ai-strategy-plans",
  Editor: AiStrategyPlanEditor,
  list: async (projectId) =>
    (await aiPlanningApi.listPlans(projectId)).ai_strategy_plans,
  create: aiPlanningApi.createPlan,
  update: aiPlanningApi.updatePlan,
  remove: aiPlanningApi.deletePlan,
  toDraft: aiStrategyPlanToDraft,
  newDraft: newAiStrategyPlanDraft,
  rowId: (row) => row.plan_id,
  subtitle: (row) => row.name || row.desc,
  format: formatAiStrategyPlanScript,
};

const TEMPLATE_CONFIG: PlanningConfig<AiTemplate, AiTemplateDraft> = {
  kind: "template",
  dirtyKey: "ai-templates",
  Editor: AiTemplateEditor,
  list: async (projectId) =>
    (await aiPlanningApi.listTemplates(projectId)).ai_templates,
  create: aiPlanningApi.createTemplate,
  update: aiPlanningApi.updateTemplate,
  remove: aiPlanningApi.deleteTemplate,
  toDraft: aiTemplateToDraft,
  newDraft: newAiTemplateDraft,
  rowId: (row) => row.template_id,
  subtitle: (row) => row.role,
  format: formatAiTemplateScript,
};

function PlanningWorkbench<Row extends RowMeta, Draft>({
  config,
}: {
  config: PlanningConfig<Row, Draft>;
}) {
  const params = useParams<{ id: string }>();
  const projectId = Number(params.id);
  const t = useTranslations("aiPlanning");
  const [rows, setRows] = useState<Row[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");
  const [saveStatus, setSaveStatus] = useState<SaveStatus>("idle");
  const ownership = useRef<DraftSession<Draft>>(emptyDraftSession<Draft>());
  const selectedIdRef = useRef<number | null>(null);
  const loadGeneration = useRef(0);
  const selected = useMemo(
    () => rows.find((row) => row.id === selectedId) ?? null,
    [rows, selectedId],
  );
  const dirty = Boolean(
    selected &&
    draft &&
    JSON.stringify(draft) !== JSON.stringify(config.toDraft(selected)),
  );
  useRegisterDirty(config.dirtyKey, dirty);

  const fingerprint = useCallback(
    (value: Draft) => stableDraftFingerprint(value),
    [],
  );
  const openRow = useCallback(
    (row: Row) => {
      const next = config.toDraft(row);
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
    [config, fingerprint],
  );
  const edit = useCallback(
    (next: Draft) => {
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

  const load = useCallback(async () => {
    if (!Number.isInteger(projectId) || projectId <= 0) {
      setError(t("invalidProject"));
      setLoading(false);
      return;
    }
    const request = ++loadGeneration.current;
    setLoading(true);
    try {
      const loaded = await config.list(projectId);
      if (request !== loadGeneration.current) return;
      setRows(loaded);
      const selectedRow =
        loaded.find((row) => row.id === selectedIdRef.current) ?? null;
      const current = selectedRow ?? loaded[0] ?? null;

      if (current && selectedRow === null) openRow(current);
      if (!current) {
        selectedIdRef.current = null;
        ownership.current = emptyDraftSession<Draft>();
        setSelectedId(null);
        setDraft(null);
      }
    } finally {
      setLoading(false);
    }
  }, [config, openRow, projectId, t]);
  const { error: listLoadError, mutate: retryList } = useSWR(
    projectId ? ["ai-planning/list", config.kind, projectId] : null,
    load,
  );

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return needle
      ? rows.filter((row) => config.rowId(row).toLowerCase().includes(needle))
      : rows;
  }, [config, query, rows]);
  const nextId = () => {
    const stem = config.kind === "plan" ? "new_ai_plan" : "new_ai_template";
    const ids = new Set(rows.map(config.rowId));
    if (!ids.has(stem)) return stem;
    let suffix = 2;
    while (ids.has(`${stem}_${suffix}`)) suffix += 1;
    return `${stem}_${suffix}`;
  };
  const choose = (row: Row) => {
    const next = config.toDraft(row);
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
  const create = async () => {
    if (dirty && !window.confirm(t("discardConfirm"))) return;
    setCreating(true);
    setError("");
    try {
      const created = await config.create(projectId, config.newDraft(nextId()));
      setRows((current) => [...current, created]);
      openRow(created);
      setSaveStatus("success");
    } catch {
      setError(t("createFailed"));
    } finally {
      setCreating(false);
    }
  };
  const save = async () => {
    if (!selected || !draft) return;
    const save = beginDraftSave(ownership.current);
    if (!save) return;
    setSaving(true);
    setError("");
    setSaveStatus("idle");
    try {
      const updated = await config.update(projectId, selected.id, draft);
      setRows((current) =>
        current.map((row) => (row.id === updated.id ? updated : row)),
      );
      const saved = config.toDraft(updated);
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
  const remove = async (row: Row) => {
    if (!window.confirm(t("deleteConfirm", { id: config.rowId(row) }))) return;
    setError("");
    try {
      await config.remove(projectId, row.id);
      const remaining = rows.filter((item) => item.id !== row.id);
      setRows(remaining);
      if (selectedId === row.id) {
        const next = remaining[0] ?? null;
        if (next) openRow(next);
        else {
          selectedIdRef.current = null;
          ownership.current = emptyDraftSession<Draft>();
          setSelectedId(null);
          setDraft(null);
        }
        setSaveStatus("idle");
      }
    } catch {
      setError(t("deleteFailed"));
    }
  };

  const titleKey = config.kind === "plan" ? "plansTitle" : "templatesTitle";
  const introKey = config.kind === "plan" ? "plansIntro" : "templatesIntro";
  const listPanel = (
    <div className="flex min-h-full flex-col p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-base font-semibold">{t(titleKey)}</h1>
            <span className="rounded-full bg-card px-2 py-0.5 text-xs text-muted-foreground">
              {rows.length}
            </span>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">{t(introKey)}</p>
        </div>
        <Button size="sm" onClick={() => void create()} disabled={creating}>
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
          <p className="rounded-md border border-border p-4 text-xs text-muted-foreground">
            {t("loading")}
          </p>
        ) : filtered.length === 0 ? (
          <p className="rounded-md border border-dashed border-border p-4 text-center text-xs text-muted-foreground">
            {query ? t("noSearchResults") : t("emptyList")}
          </p>
        ) : (
          filtered.map((row) => (
            <div
              key={row.id}
              className={`ui-card group flex items-center gap-1 p-1 ${selectedId === row.id ? "border-border bg-card" : ""}`}
            >
              <button
                type="button"
                className="ui-focus-ring min-w-0 flex-1 rounded-md px-3 py-2 text-left"
                onClick={() => choose(row)}
              >
                <span className="block truncate font-mono text-sm">
                  {config.rowId(row)}
                </span>
                <span className="mt-1 block truncate text-xs text-muted-foreground">
                  {config.subtitle(row) || t("noDescription")}
                </span>
              </button>
              <Button
                variant="ghost"
                size="icon"
                className="opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                onClick={() => void remove(row)}
                aria-label={t("deleteAria", { id: config.rowId(row) })}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))
        )}
      </div>
    </div>
  );
  const empty = (
    <div className="flex h-full items-center justify-center p-6">
      <EditorEmptyState
        variant={rows.length === 0 ? "hero" : "compact"}
        icon={<BrainCircuit className="h-10 w-10 text-muted-foreground/60" />}
        title={rows.length === 0 ? t("emptyTitle") : t("selectTitle")}
        description={
          rows.length === 0 ? t("emptyDescription") : t("selectDescription")
        }
        actions={
          rows.length === 0 ? (
            <Button onClick={() => void create()}>{t("createFirst")}</Button>
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
      <pre className="overflow-x-auto whitespace-pre-wrap rounded-lg border border-border bg-background p-4 font-mono text-xs leading-normal">
        {config.format(draft, {
          source: dirty ? "created" : selected?.source,
          rawBlock: selected?.raw_block,
        })}
      </pre>
      <p className="rounded-lg border border-border bg-muted/30 p-4 text-xs text-muted-foreground">
        {t("previewMutationNote")}
      </p>
    </div>
  ) : (
    <div className="p-6 text-xs text-muted-foreground">{t("previewEmpty")}</div>
  );
  const Editor = config.Editor;
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
          <Editor
            value={draft}
            onChange={edit}
            onSave={() => void save()}
            onDelete={() => void remove(selected)}
            saving={saving}
            dirty={dirty}
            saveStatus={saveStatus}
          />
        ) : (
          empty
        )
      }
      previewPanel={preview}
    />
  );
}

export function AiStrategyPlansWorkbench() {
  return <PlanningWorkbench config={PLAN_CONFIG} />;
}
export function AiTemplatesWorkbench() {
  return <PlanningWorkbench config={TEMPLATE_CONFIG} />;
}
