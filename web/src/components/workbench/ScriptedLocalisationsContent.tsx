"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useTranslations } from "use-intl";
import { ArrowLeft, MessageSquare, SearchX } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
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
  scriptedLocalisationsApi,
  type ScriptedLocalisationRow,
} from "@/lib/api";

const NEW_BODY_TEMPLATE =
  "name = GetExample\ntext = {\n\tlocalization_key = example_key\n}";

interface DraftState {
  name: string;
  body: string;
  source_file: string;
}

function blankDraft(): DraftState {
  return {
    name: "",
    body: NEW_BODY_TEMPLATE,
    source_file: "",
  };
}

function rowToDraft(r: ScriptedLocalisationRow): DraftState {
  return {
    name: r.name,
    body: r.body || "",
    source_file: r.source_file || "",
  };
}

export function ScriptedLocalisationsContent({
  projectId,
}: {
  projectId: number;
}) {
  const t = useTranslations("scriptedLocalisations");
  const tCommon = useTranslations("common");
  const tShell = useTranslations("workbenchShell");
  const confirmDialog = useConfirm();

  const [rows, setRows] = useState<ScriptedLocalisationRow[]>([]);

  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [activeId, setActiveId] = useState<number | "new" | null>(null);
  const [draft, setDraft] = useState<DraftState>(blankDraft());
  const [error, setError] = useState<string | null>(null);
  const [listLoadFailed, setListLoadFailed] = useState(false);
  const [saving, setSaving] = useState(false);
  const ownership = useRef<DraftSession<DraftState>>(emptyDraftSession());
  const fingerprint = useCallback(
    (value: DraftState) => stableDraftFingerprint(value),
    [],
  );
  const activeRow =
    typeof activeId === "number"
      ? (rows.find((row) => row.id === activeId) ?? null)
      : null;
  const dirty = Boolean(
    activeRow && fingerprint(draft) !== fingerprint(rowToDraft(activeRow)),
  );
  useRegisterDirty("scripted-localisations", dirty);

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      const res = await scriptedLocalisationsApi.list(projectId);
      setRows(res.localisations);
      setListLoadFailed(false);
    } catch {
      setListLoadFailed(true);
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    reload();
  }, [reload]);

  const update = useCallback(
    <K extends keyof DraftState>(key: K, value: DraftState[K]) => {
      setDraft((d) => {
        const next = { ...d, [key]: value };
        ownership.current = editDraftSession(
          ownership.current,
          next,
          fingerprint(next),
        );
        return next;
      });
    },
    [fingerprint],
  );

  const selectRow = useCallback(
    async (r: ScriptedLocalisationRow) => {
      const next = rowToDraft(r);
      const requested = requestDraftSelection(
        ownership.current,
        String(r.id),
        fingerprint(next),
        next,
      );
      if (
        requested.kind === "confirm-discard" &&
        !(await confirmDialog({ body: tCommon("discardUnsavedConfirm") }))
      )
        return;
      ownership.current =
        requested.kind === "selected"
          ? requested.session
          : confirmDraftSelection(
              ownership.current,
              String(r.id),
              fingerprint(next),
              next,
            );
      setActiveId(r.id);
      setDraft(next);
      setError(null);
    },
    [confirmDialog, fingerprint, tCommon],
  );

  const startNew = useCallback(async () => {
    const next = blankDraft();
    const requested = requestDraftSelection(
      ownership.current,
      "new",
      fingerprint(next),
      next,
    );
    if (
      requested.kind === "confirm-discard" &&
      !(await confirmDialog({ body: tCommon("discardUnsavedConfirm") }))
    )
      return;
    ownership.current =
      requested.kind === "selected"
        ? requested.session
        : confirmDraftSelection(
            ownership.current,
            "new",
            fingerprint(next),
            next,
          );
    setActiveId("new");
    setDraft(next);
    setError(null);
  }, [confirmDialog, fingerprint, tCommon]);

  const handleSave = useCallback(async () => {
    setError(null);
    const name = draft.name.trim();
    if (!name) {
      setError(t("nameRequired"));
      return;
    }
    setSaving(true);
    const save = beginDraftSave(ownership.current);
    if (!save) return;
    try {
      const payload = {
        name,
        body: draft.body,
        source_file: draft.source_file.trim(),
      };
      if (activeId === "new") {
        const created = await scriptedLocalisationsApi.create(
          projectId,
          payload,
        );
        await reload();
        const saved = rowToDraft(created);
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(saved),
          saved,
        );
        if (ownership.current.identity === "new") {
          ownership.current = openDraftSession(
            ownership.current,
            String(created.id),
            fingerprint(saved),
            saved,
          );
          setActiveId(created.id);
          setDraft(saved);
        }
      } else if (typeof activeId === "number") {
        const updated = await scriptedLocalisationsApi.update(
          projectId,
          activeId,
          payload,
        );
        await reload();
        const saved = rowToDraft(updated);
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(saved),
          saved,
        );
        if (
          ownership.current.identity === String(updated.id) &&
          ownership.current.draft !== null
        ) {
          setDraft(ownership.current.draft);
        }
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }, [activeId, draft, fingerprint, projectId, reload, t]);

  const handleDelete = useCallback(async () => {
    if (typeof activeId !== "number") return;
    if (!(await confirmDialog({ body: t("deleteConfirm"), danger: true })))
      return;
    try {
      await scriptedLocalisationsApi.remove(projectId, activeId);
      setActiveId(null);
      setDraft(blankDraft());
      await reload();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [activeId, confirmDialog, projectId, reload, t]);

  const sortedRows = useMemo(
    () => [...rows].sort((a, b) => a.name.localeCompare(b.name)),
    [rows],
  );

  const filteredRows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) return sortedRows;
    return sortedRows.filter((r) => r.name.toLowerCase().includes(needle));
  }, [sortedRows, search]);

  const zeroHero = (
    <EditorEmptyState
      variant="hero"
      icon={<MessageSquare className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyHeroTitle")}
      description={t("emptyHeroDesc")}
      actions={
        <Button
          size="sm"
          onClick={() => {
            void startNew();
          }}
        >
          {tCommon("emptyCtaCreate")}
        </Button>
      }
    />
  );

  const listPanel = (
    <div>
      <WorkbenchListHeader
        title={t("title")}
        count={filteredRows.length}
        totalCount={rows.length}
        searchValue={search}
        onSearchChange={setSearch}
        searchPlaceholder={tShell("searchPlaceholderGeneric")}
        renderSearch={rows.length > 0}
        action={
          <Button size="sm" onClick={() => void startNew()}>
            {t("newButton")}
          </Button>
        }
      />
      <div className="px-3 pb-3">
        {loading ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("loading")}
          </p>
        ) : listLoadFailed && rows.length === 0 ? null : rows.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("emptyHeroTitle")}
          </p>
        ) : filteredRows.length === 0 ? (
          <div className="p-4 flex flex-col items-center gap-2 text-center border border-dashed rounded-lg bg-muted/20">
            <SearchX className="h-5 w-5 text-muted-foreground" />
            <p className="text-xs text-muted-foreground">
              {tShell("noMatchesGeneric")}
            </p>
          </div>
        ) : (
          <div className="space-y-0.5">
            {filteredRows.map((r) => (
              <button
                key={r.id}
                type="button"
                onClick={() => void selectRow(r)}

                className={`w-full text-left px-2 py-1.5 rounded text-xs truncate transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
                  activeId === r.id
                    ? "bg-accent/10 text-accent font-medium shadow-[inset_3px_0_0_0] shadow-accent"
                    : "hover:bg-muted text-muted-foreground"
                }`}
                title={r.name}
              >
                {r.name}
                {r.source === "imported" ? (
                  <span className="ml-1 text-micro opacity-60">
                    ({t("imported")})
                  </span>
                ) : null}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );

  const editorForm = (
    <div className="max-w-3xl space-y-4 p-4">
      {activeId === "new" && (
        <button
          type="button"
          onClick={() => {
            setActiveId(null);
            setDraft(blankDraft());
          }}
          className="ui-interactive inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          {tShell("backFromNew")}
        </button>
      )}
      <p className="text-xs text-muted-foreground">{t("intro")}</p>

      {error ? (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-xs text-destructive">
          {error}
        </div>
      ) : null}

      <div className="space-y-1">
        <Label>{t("nameLabel")}</Label>
        <Input
          value={draft.name}
          onChange={(e) => update("name", e.target.value)}
          placeholder={t("namePlaceholder")}
          className="text-sm font-mono"
        />
        <p className="text-micro text-muted-foreground">{t("nameHint")}</p>
      </div>

      <div className="space-y-1">
        <Label>{t("sourceFileLabel")}</Label>
        <Input
          value={draft.source_file}
          onChange={(e) => update("source_file", e.target.value)}
          placeholder={t("sourceFilePlaceholder")}
          className="text-sm font-mono"
        />
        <p className="text-micro text-muted-foreground">
          {t("sourceFileHint")}
        </p>
      </div>

      <Separator />

      <div className="space-y-1">
        <Label>{t("bodyLabel")}</Label>
        <textarea
          value={draft.body}
          onChange={(e) => update("body", e.target.value)}
          rows={16}
          spellCheck={false}
          className="w-full rounded-md border border-input bg-background px-2 py-1.5 font-mono text-xs"
          placeholder={NEW_BODY_TEMPLATE}
        />
        <p className="text-micro text-muted-foreground">{t("bodyHint")}</p>
      </div>

      <div className="flex items-center gap-2 pt-2">
        <Button onClick={handleSave} disabled={saving}>
          {saving
            ? t("savingButton")
            : activeId === "new"
              ? t("createButton")
              : t("saveButton")}
        </Button>
        {typeof activeId === "number" ? (
          <Button variant="destructive" onClick={handleDelete}>
            {t("deleteButton")}
          </Button>
        ) : null}
      </div>
    </div>
  );

  return (
    <WorkbenchShell
      listPanel={listPanel}
      editorPanel={
        activeId === null ? (
          <div className="flex h-full items-center justify-center p-6">
            {listLoadFailed && rows.length === 0 ? (
              <WorkbenchLoadError onRetry={() => void reload()} />
            ) : rows.length === 0 ? (
              zeroHero
            ) : (
              <EditorEmptyState
                className="max-w-xs"
                icon={
                  <MessageSquare className="h-6 w-6 text-muted-foreground/60" />
                }
                title={tShell("noSelectionGeneric")}
                description={t("selectHint")}
              />
            )}
          </div>
        ) : (
          editorForm
        )
      }
      previewPanel={
        <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
          {tShell("previewUnavailableGeneric")}
        </div>
      }
    />
  );
}
