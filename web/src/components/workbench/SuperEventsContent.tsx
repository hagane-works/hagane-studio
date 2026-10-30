"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useLinkedRowSelection } from "@/lib/workbench-state/useLinkedRowSelection";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { Clapperboard, SearchX } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TooltipProvider } from "@/components/ui/tooltip";
import { superEventsApi, type SuperEventRow } from "@/lib/api";
import {
  useActiveCountry,
  useEffectiveCountryFilter,
  useProjectStore,
} from "@/lib/store/projectStore";
import { matchesCountryScope } from "@/lib/utils/scope";
import { CountryTagChip } from "@/components/workbench/CountryTagChip";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { superEventWindowName } from "@/app/[locale]/game-view/superEventModel";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import { ScopeSectionHeader } from "@/components/workbench/ScopeBadge";
import {
  SuperEventEditor,
  blankSuperEventDraft,
  superEventToDraft,
  type SuperEventDraft,
} from "@/components/workbench/SuperEventEditor";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
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

interface SuperEventsContentProps {
  projectId: number;
}

export function SuperEventsContent({ projectId }: SuperEventsContentProps) {
  const projectTag = useProjectStore(
    (s) => s.currentProject?.country_tag ?? null,
  );
  const t = useTranslations("superEventsDialog");
  const tCommon = useTranslations("common");
  const confirmDialog = useConfirm();
  const activeCountry = useActiveCountry();

  const filterCountry = useEffectiveCountryFilter();
  const viewMode = useProjectStore((s) => s.viewMode);
  const [rows, setRows] = useState<SuperEventRow[]>([]);
  const [activeId, setActiveId] = useState<number | "new" | null>(null);

  const [search, setSearch] = useState("");
  const [draft, setDraft] = useState<SuperEventDraft>(() =>
    blankSuperEventDraft(1),
  );
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [picBust, setPicBust] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [listLoadFailed, setListLoadFailed] = useState(false);
  const [status, setStatus] = useState<"" | "saved" | "deleted">("");
  const ownership = useRef<DraftSession<SuperEventDraft>>(emptyDraftSession());
  const latestListRequest = useRef(0);
  const fingerprint = useCallback(
    (value: SuperEventDraft) => stableDraftFingerprint(value),
    [],
  );

  const refresh = useCallback(async (): Promise<SuperEventRow[] | null> => {
    const request = ++latestListRequest.current;
    try {
      const res = await superEventsApi.list(projectId);
      if (request !== latestListRequest.current) return null;
      setRows(res.super_events);
      setListLoadFailed(false);
      return res.super_events;
    } catch {
      if (request === latestListRequest.current) setListLoadFailed(true);
      return null;
    }
  }, [projectId]);

  const selectRow = useCallback(
    async (row: SuperEventRow) => {
      const next = superEventToDraft(row);
      const requested = requestDraftSelection(
        ownership.current,
        String(row.id),
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
              String(row.id),
              fingerprint(next),
              next,
            );
      setActiveId(row.id);
      setDraft(next);
      setError(null);
    },
    [confirmDialog, fingerprint, tCommon],
  );

  useLinkedRowSelection("se", rows, selectRow);

  const startNew = useCallback(async () => {
    const nextNumber =
      rows.length === 0 ? 1 : Math.max(...rows.map((row) => row.number)) + 1;
    const next = blankSuperEventDraft(nextNumber);
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
  }, [confirmDialog, fingerprint, rows, tCommon]);

  const filteredRows = useMemo(() => {
    const scoped = rows.filter((r) => matchesCountryScope(r, filterCountry));
    const needle = search.trim().toLowerCase();
    if (!needle) return scoped;
    return scoped.filter(
      (r) =>
        String(r.number).padStart(4, "0").includes(needle) ||
        r.title_en.toLowerCase().includes(needle) ||
        r.title_zh.toLowerCase().includes(needle),
    );
  }, [rows, filterCountry, search]);

  useSWR(projectId ? ["super-events/list", projectId] : null, () => refresh());

  useEffect(() => {
    if (activeId !== null && ownership.current.identity === String(activeId))
      return;
    if (activeId === "new") {
      const nextNumber =
        rows.length === 0 ? 1 : Math.max(...rows.map((r) => r.number)) + 1;
      setDraft(blankSuperEventDraft(nextNumber));
    } else if (typeof activeId === "number") {
      const row = rows.find((r) => r.id === activeId);
      if (row) {
        setDraft({
          number: row.number,
          country_tag: row.country_tag,
          title_en: row.title_en,
          title_zh: row.title_zh,
          desc_en: row.desc_en,
          desc_zh: row.desc_zh,
          quote_en: row.quote_en,
          quote_zh: row.quote_zh,
          btn_en: row.btn_en,
          btn_zh: row.btn_zh,

          display_name_locs: row.display_name_locs ?? {},

          description_locs: row.description_locs ?? {},
          quote_locs: row.quote_locs ?? {},
          button_locs: row.button_locs ?? {},
          song_token: row.song_token,
          auto_fire_on_startup: row.auto_fire_on_startup,
          auto_fire_after_days: row.auto_fire_after_days,
          options: Array.isArray(row.options) ? row.options : [],
        });
      }
    }
  }, [activeId, rows]);

  const update = <K extends keyof SuperEventDraft>(
    key: K,
    value: SuperEventDraft[K],
  ) => {
    setDraft((current) => {
      const next = { ...current, [key]: value };
      ownership.current = editDraftSession(
        ownership.current,
        next,
        fingerprint(next),
      );
      return next;
    });
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    const save = beginDraftSave(ownership.current);
    if (!save) {
      setSaving(false);
      return;
    }
    const savingDraft = draft;
    try {
      if (activeId === "new") {
        const created = await superEventsApi.create(projectId, savingDraft);
        await refresh();
        const saved = superEventToDraft(created);
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
        const updated = await superEventsApi.update(
          projectId,
          activeId,
          savingDraft,
        );
        await refresh();
        const saved = superEventToDraft(updated);
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(saved),
          saved,
        );
        if (ownership.current.identity === String(updated.id))
          setDraft(ownership.current.draft!);
      }
      setStatus("saved");
      setTimeout(() => setStatus(""), 1500);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (typeof activeId !== "number") return;
    if (!(await confirmDialog({ body: t("confirmDelete"), danger: true })))
      return;
    setSaving(true);
    setError(null);
    try {
      await superEventsApi.remove(projectId, activeId);
      await refresh();
      if (ownership.current.identity === String(activeId)) {
        ownership.current = emptyDraftSession();
        setActiveId(null);
        setDraft(blankSuperEventDraft(1));
      }
      setStatus("deleted");
      setTimeout(() => setStatus(""), 1500);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const commitPicture = async (file: File) => {
    if (typeof activeId !== "number") return;
    await superEventsApi.uploadPicture(projectId, activeId, file);
    await refresh();
    setPicBust((v) => v + 1);
    setStatus("saved");
    setTimeout(() => setStatus(""), 1500);
  };

  const handleUpload = async (file: File) => {
    if (typeof activeId !== "number") return;
    setUploading(true);
    setError(null);
    try {
      await commitPicture(file);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setUploading(false);
    }
  };

  const isExisting = typeof activeId === "number";
  const activeRow = isExisting ? rows.find((r) => r.id === activeId) : null;

  const zeroSuperEventHero = (
    <EditorEmptyState
      variant="hero"
      icon={<Clapperboard className="h-10 w-10 text-muted-foreground/60" />}

      title={t("emptyHeroTitle")}
      description={t("emptyAllHint")}
      actions={
        <>
          <Button
            size="sm"
            onClick={() => {
              void startNew();
            }}
          >
            {tCommon("emptyCtaCreate")}
          </Button>
        </>
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
        searchPlaceholder={t("searchPlaceholder")}
        renderSearch={rows.length > 0}
        action={
          <Button
            size="sm"
            variant={activeId === "new" ? "default" : "outline"}
            onClick={() => void startNew()}
          >
            {t("newSE")}
          </Button>
        }
      />
      <div className="px-3 pb-3">
        {listLoadFailed && rows.length === 0 ? null : rows.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("emptyAll")}
          </p>
        ) : filteredRows.length === 0 ? (
          search.trim() ? (
            <div className="p-4 flex flex-col items-center gap-2 text-center border border-dashed rounded-lg bg-muted/20">
              <SearchX className="h-5 w-5 text-muted-foreground" />
              <p className="text-xs text-muted-foreground">{t("noMatches")}</p>
            </div>
          ) : (
            <div className="rounded-md border border-border bg-card p-4 flex flex-col items-center gap-2 text-center">
              <Clapperboard className="w-6 h-6 text-muted-foreground/60" />
              <p className="text-xs font-medium text-foreground">
                {t("emptyScoped", {
                  country: activeCountry || t("currentCountry"),
                })}
              </p>
            </div>
          )
        ) : (
          <div className="space-y-0.5">
            {!filterCountry && (
              <ScopeSectionHeader variant="all" count={filteredRows.length} />
            )}
            {filteredRows.map((r) => (
              <button
                key={r.id}
                onClick={() => void selectRow(r)}

                className={`w-full text-left flex items-center gap-1.5 px-2 py-1.5 rounded text-xs transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
                  activeId === r.id
                    ? "bg-accent/10 text-accent font-medium shadow-[inset_3px_0_0_0] shadow-accent"
                    : "hover:bg-muted"
                }`}
              >
                {viewMode === "all" && <CountryTagChip tag={r.country_tag} />}
                <span className="font-mono">
                  {String(r.number).padStart(4, "0")}
                </span>
                <span className="opacity-60">·</span>
                <span className="truncate">
                  {r.title_en || r.title_zh || t("noTitle")}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );

  const editorForm = (
    <SuperEventEditor
      projectId={projectId}
      identity={activeId ?? "new"}
      draft={draft}
      onChange={update}
      row={activeRow ?? null}
      pictureBust={picBust}
      saving={saving}
      uploading={uploading}
      saved={status === "saved"}
      error={error}
      onSave={handleSave}
      onDelete={isExisting ? handleDelete : undefined}
      onUpload={(file) => void handleUpload(file)}
      onBack={
        activeId === "new"
          ? () => {
              ownership.current = emptyDraftSession();
              setActiveId(null);
              setDraft(blankSuperEventDraft(1));
            }
          : undefined
      }
    />
  );

  return (
    <TooltipProvider>
      <div className="flex h-full min-w-0 overflow-hidden">
        <div className="flex-1 min-w-0 h-full overflow-hidden">
          <WorkbenchShell
            gameViewHref={
              activeRow
                ? gameViewHref(projectId, {
                    panel: superEventWindowName(activeRow, projectTag),
                    superEvent: activeRow.id,
                    tag: activeRow.country_tag || projectTag,
                  })
                : null
            }
            listPanel={listPanel}
            editorPanel={
              activeId === null ? (
                <div className="flex h-full items-center justify-center p-6">
                  {listLoadFailed && rows.length === 0 ? (
                    <WorkbenchLoadError onRetry={() => void refresh()} />
                  ) : rows.length === 0 ? (
                    zeroSuperEventHero
                  ) : (
                    <EditorEmptyState
                      className="max-w-xs"
                      icon={
                        <Clapperboard className="h-6 w-6 text-muted-foreground/60" />
                      }
                      title={t("noSelectionTitle")}
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
                {t("previewUnavailable")}
              </div>
            }
          />
        </div>
      </div>
    </TooltipProvider>
  );
}
