"use client";
import { withoutEditRevisions } from "./save-diagnostics/editRevisions";
import { useLinkedRowSelection } from "@/lib/workbench-state/useLinkedRowSelection";

import { useSaveDiagnostics } from "./save-diagnostics/useSaveDiagnostics";
import {
  mergeSaveDiagnostics,
  readSaveDiagnostics,
} from "@/lib/saveDiagnostics";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useTranslations } from "use-intl";
import { BookMarked, ChevronDown, ChevronUp, SearchX } from "lucide-react";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { useProjectStore } from "@/lib/store/projectStore";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { Button } from "@/components/ui/button";
import { bookmarksApi, type BookmarkRow } from "@/lib/api";
import {
  ScenarioSettingsPanel,
  type ScenarioSettingsValue,
} from "@/components/project-scenario/ScenarioSettingsPanel";
import {
  ScenarioWorldRiskPreview,
  type ScenarioWorldRiskPreviewModel,
} from "@/components/project-scenario/ScenarioWorldRiskPreview";
import { BookmarkReplacePathNotice } from "@/components/project-scenario/BookmarkReplacePathNotice";
import {
  applyProjectReadback,
  saveScenarioProjectWithReadback,
  ScenarioReadbackMismatchError,
} from "@/components/project-scenario/scenarioPersistence";
import {
  earliestSavedBookmark,
  hasCustomBookmarkOpening,
  latestSavedBookmark,
  sameHoi4Day,
  scenarioNeedsExplicitEndDate,
} from "@/components/project-scenario/scenarioTimelineUi";
import { useDeadFlagLint } from "@/shared/useDeadFlagLint";
import {
  BookmarkEditor,
  blankBookmarkDraft as blankDraft,
  bookmarkRowToDraft as rowToDraft,
  type BookmarkDraft as Draft,
} from "@/components/workbench/BookmarkEditor";
import { DeadFlagBanner } from "@/components/workbench/DeadFlagBanner";
import { DialogSaveStatusBar } from "@/components/workbench/DialogSaveStatusBar";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
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

interface BookmarksContentProps {
  projectId: number;
}

export function BookmarksContent({ projectId }: BookmarksContentProps) {
  const t = useTranslations("bookmarksDialog");
  const tScenario = useTranslations("ScenarioSettings");
  const tCommon = useTranslations("common");
  const confirmDialog = useConfirm();
  const locale = useContentLocaleStore((s) => s.locale);
  const currentProject = useProjectStore((s) => s.currentProject);
  const open = true;
  const [rows, setRows] = useState<BookmarkRow[]>([]);
  const [activeId, setActiveId] = useState<number | "new" | null>(null);

  const [search, setSearch] = useState("");
  const [draft, setDraft] = useState<Draft>(() => blankDraft());
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [listLoadFailed, setListLoadFailed] = useState(false);
  const [status, setStatus] = useState<"" | "saved" | "deleted">("");
  const { diagnostics } = useSaveDiagnostics(
    `${projectId}:bookmark:${activeId}`,
    rows.find((row) => row.id === activeId && row.project_id === projectId),
  );
  const [scenarioValue, setScenarioValue] = useState<ScenarioSettingsValue>({
    scenarioHistoryPolicy: "inherit_base_history",
    endDate: "",
  });
  const [scenarioSaving, setScenarioSaving] = useState(false);
  const [scenarioStatus, setScenarioStatus] = useState<"" | "saved">("");
  const [scenarioError, setScenarioError] = useState<string | null>(null);
  const [campaignOpeningSaving, setCampaignOpeningSaving] = useState(false);
  const [campaignOpeningStatus, setCampaignOpeningStatus] = useState<
    "" | "saved"
  >("");
  const [campaignOpeningError, setCampaignOpeningError] = useState<
    string | null
  >(null);
  const [campaignSetupExpanded, setCampaignSetupExpanded] = useState(false);
  const [navigationReadyStatus, setNavigationReadyStatus] = useState(false);
  const seededScenarioProjectRef = useRef<number | null>(null);
  const ownership = useRef<DraftSession<Draft>>(emptyDraftSession());
  const activeProjectId = useRef(projectId);
  activeProjectId.current = projectId;
  const latestListRequest = useRef(0);
  const fingerprint = useCallback(
    (value: Draft) => stableDraftFingerprint(withoutEditRevisions(value)),
    [],
  );
  const syncedBmIdRef = useRef<typeof activeId>(activeId);

  const { report: deadFlagReport } = useDeadFlagLint(open ? projectId : null);

  const refresh = useCallback(async (): Promise<BookmarkRow[] | null> => {
    const requestId = ++latestListRequest.current;
    try {
      const res = await bookmarksApi.list(projectId);
      if (
        requestId !== latestListRequest.current ||
        activeProjectId.current !== projectId
      )
        return null;
      setRows((previous) =>
        res.map((row) => {
          const diagnostics = mergeSaveDiagnostics(
            readSaveDiagnostics(
              previous.find(
                (old) => old.id === row.id && old.project_id === row.project_id,
              ),
            ),
            readSaveDiagnostics(row),
          );
          return diagnostics === undefined ? row : { ...row, diagnostics };
        }),
      );
      setListLoadFailed(false);
      return res;
    } catch {
      if (requestId === latestListRequest.current) setListLoadFailed(true);
      return null;
    }
  }, [projectId]);

  useEffect(() => {
    if (
      !currentProject ||
      currentProject.id !== projectId ||
      seededScenarioProjectRef.current === projectId
    ) {
      return;
    }
    seededScenarioProjectRef.current = projectId;
    setScenarioValue({
      scenarioHistoryPolicy:
        currentProject.scenario_history_policy === "project_owns_starting_world"
          ? "project_owns_starting_world"
          : "inherit_base_history",
      endDate: currentProject.end_date || "",
    });
    setScenarioStatus("");
    setScenarioError(null);
    setCampaignOpeningStatus("");
    setCampaignOpeningError(null);
  }, [currentProject, projectId]);

  const selectRow = useCallback(
    async (row: BookmarkRow) => {
      const next = rowToDraft(row);
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
      syncedBmIdRef.current = row.id;
      setActiveId(row.id);
      setDraft(next);
      setError(null);
      setStatus("");
    },
    [confirmDialog, fingerprint, tCommon],
  );

  useLinkedRowSelection("bm", rows, selectRow);

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
    syncedBmIdRef.current = "new";
    setActiveId("new");
    setDraft(next);
    setError(null);
    setStatus("");
  }, [confirmDialog, fingerprint, tCommon]);

  useEffect(() => {
    if (open) {
      void refresh();
    } else {
      ownership.current = emptyDraftSession();
      setActiveId(null);
      setError(null);
      setStatus("");
    }
  }, [open, refresh]);

  useEffect(() => {
    if (syncedBmIdRef.current === activeId) return;
    syncedBmIdRef.current = activeId;
    if (activeId === "new") {
      setDraft(blankDraft());
    } else if (typeof activeId === "number") {
      const row = rows.find((r) => r.id === activeId);
      if (row) {
        setDraft({
          bookmark_id: row.bookmark_id,
          name_en: row.name_en,
          name_zh: row.name_zh,
          desc_en: row.desc_en,
          desc_zh: row.desc_zh,

          display_name_locs: { ...(row.display_name_locs ?? {}) },
          description_locs: { ...(row.description_locs ?? {}) },
          date: row.date,
          picture: row.picture,
          default_country: row.default_country,
          default_flag: row.default_flag,
          selectable_countries: Array.isArray(row.selectable_countries)
            ? row.selectable_countries
            : [],
          filters: Array.isArray(row.filters) ? row.filters : [],
        });
      }
    }
  }, [activeId, rows]);

  const editDraft = useCallback(
    (change: (current: Draft) => Draft) => {
      setStatus("");
      setDraft((current) => {
        const next = change(current);
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

  const cancelNew = useCallback(() => {
    ownership.current = emptyDraftSession();
    syncedBmIdRef.current = null;
    setActiveId(null);
    setDraft(blankDraft());
  }, []);

  const handleSaveScenarioSettings = async () => {
    setScenarioSaving(true);
    setScenarioStatus("");
    setScenarioError(null);
    try {
      const readback = await saveScenarioProjectWithReadback(projectId, {
        end_date: scenarioValue.endDate.trim(),
        scenario_history_policy: scenarioValue.scenarioHistoryPolicy,
      });
      applyProjectReadback(readback);
      setScenarioStatus("saved");
      window.setTimeout(() => setScenarioStatus(""), 1500);
    } catch (cause) {
      setScenarioError(
        cause instanceof ScenarioReadbackMismatchError
          ? tScenario("saveReadbackMismatch")
          : (cause as Error).message || tScenario("saveFailed"),
      );
    } finally {
      setScenarioSaving(false);
    }
  };

  const handleUseCampaignOpening = async () => {
    const savedBookmark = earliestSavedBookmark(rows);
    const openingDate = savedBookmark?.date.trim() || "";
    if (!openingDate) {
      setCampaignOpeningError(tScenario("campaignOpening.noValidBookmark"));
      return;
    }

    setCampaignOpeningSaving(true);
    setCampaignOpeningStatus("");
    setCampaignOpeningError(null);
    try {
      const readback = await saveScenarioProjectWithReadback(projectId, {
        start_date: openingDate,
      });
      applyProjectReadback(readback);
      setCampaignOpeningStatus("saved");
      window.setTimeout(() => setCampaignOpeningStatus(""), 1500);
    } catch (cause) {
      setCampaignOpeningError(
        cause instanceof ScenarioReadbackMismatchError
          ? tScenario("saveReadbackMismatch")
          : (cause as Error).message || tScenario("saveFailed"),
      );
    } finally {
      setCampaignOpeningSaving(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    setStatus("");
    const completesSecondBookmark = activeId === "new" && rows.length === 1;
    const save = beginDraftSave(ownership.current);
    if (!save) {
      setSaving(false);
      return;
    }
    let savedIdentity = save.identity;
    try {
      if (activeId === "new") {
        const created = await bookmarksApi.create(projectId, draft);
        if (activeProjectId.current !== projectId) return;
        savedIdentity = String(created.id);
        setRows((previous) => [
          ...previous.filter((row) => row.id !== created.id),
          created,
        ]);
        await refresh();
        const saved = rowToDraft(created);
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(saved),
          saved,
        );
        if (ownership.current.identity === "new") {
          const current = ownership.current.draft!;
          const hasPostSaveEdit = ownership.current.dirty;
          ownership.current = openDraftSession(
            ownership.current,
            String(created.id),
            fingerprint(saved),
            current,
          );
          if (hasPostSaveEdit) {
            ownership.current = editDraftSession(
              ownership.current,
              current,
              fingerprint(current),
            );
          }
          syncedBmIdRef.current = created.id;
          setActiveId(created.id);
          if (!hasPostSaveEdit) setDraft(saved);
        }
      } else if (typeof activeId === "number") {
        const updated = await bookmarksApi.update(projectId, activeId, draft);
        if (activeProjectId.current !== projectId) return;
        setRows((previous) =>
          previous.map((row) =>
            row.id === updated.id
              ? {
                  ...updated,
                  diagnostics: mergeSaveDiagnostics(
                    readSaveDiagnostics(row),
                    readSaveDiagnostics(updated),
                  ),
                }
              : row,
          ),
        );
        await refresh();
        const saved = rowToDraft(updated);
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(saved),
          saved,
        );
        if (
          ownership.current.identity === String(updated.id) &&
          !ownership.current.dirty
        ) {
          setDraft(saved);
        }
      }
      if (
        !ownership.current.dirty &&
        ownership.current.saveGeneration === save.saveGeneration &&
        ownership.current.identity === savedIdentity
      ) {
        setStatus("saved");
        setTimeout(() => {
          if (
            ownership.current.saveGeneration === save.saveGeneration &&
            ownership.current.identity === savedIdentity
          )
            setStatus("");
        }, 1500);
      }
      if (completesSecondBookmark) {
        setNavigationReadyStatus(true);
        window.setTimeout(() => setNavigationReadyStatus(false), 3500);
      }
    } catch (e) {
      const msg = (e as Error).message || "";
      const isCollision =
        msg.includes("already exists") && msg.includes("Pick a unique");
      if (
        !isCollision &&
        ownership.current.identity === save.identity &&
        activeProjectId.current === projectId &&
        ownership.current.saveGeneration === save.saveGeneration
      ) {
        setError(msg);
      }
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
      await bookmarksApi.remove(projectId, activeId, draft.edit_revision);
      await refresh();
      if (ownership.current.identity === String(activeId)) {
        ownership.current = emptyDraftSession();
        syncedBmIdRef.current = null;
        setActiveId(null);
        setDraft(blankDraft());
      }
      setStatus("deleted");
      setTimeout(() => setStatus(""), 1500);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const campaignStartDate = currentProject?.start_date?.trim() || "";
  const campaignOpeningBookmark = earliestSavedBookmark(rows);
  const latestCampaignBookmark = latestSavedBookmark(rows);
  const campaignOpeningIsSynced = Boolean(
    campaignOpeningBookmark &&
    sameHoi4Day(campaignOpeningBookmark.date, campaignStartDate),
  );
  const isStandaloneProject = !(currentProject?.dependencies || []).some(
    (dependency) => dependency.trim() !== "",
  );
  const scenarioWorldPreview: ScenarioWorldRiskPreviewModel =
    scenarioValue.scenarioHistoryPolicy === "inherit_base_history"
      ? { state: "inherit_base_world" }
      : { state: "scenario_world_source_unverified" };

  const filteredRows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) return rows;
    return rows.filter(
      (r) =>
        r.bookmark_id.toLowerCase().includes(needle) ||
        (r.name_en || "").toLowerCase().includes(needle) ||
        (r.name_zh || "").toLowerCase().includes(needle),
    );
  }, [rows, search]);

  const zeroBookmarkHero = (
    <EditorEmptyState
      variant="hero"
      icon={<BookMarked className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyHeroTitle")}
      description={t("emptyListHint")}
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

  const campaignSetupPanel = (
    <section className="shrink-0 border-b border-border bg-card/80">
      <div className="flex items-start gap-4 px-4 py-3">
        <div className="min-w-0 flex-1 space-y-2">
          <div>
            <h2 className="text-sm font-semibold tracking-tight">
              {tScenario("campaignOpening.workspaceTitle")}
            </h2>
            <p className="text-xs text-muted-foreground">
              {tScenario("campaignOpening.workspaceIntro")}
            </p>
          </div>
          <div className="flex flex-wrap gap-2 text-xs">
            <span className="rounded-md border border-border bg-background px-2 py-1">
              {tScenario("campaignOpening.currentLabel")}:{" "}
              <span className="font-mono tech-tabular">
                {campaignStartDate || tScenario("campaignOpening.notSelected")}
              </span>
            </span>
            <span className="rounded-md border border-border bg-background px-2 py-1">
              {scenarioValue.scenarioHistoryPolicy ===
              "project_owns_starting_world"
                ? tScenario("worldOptions.own.title")
                : tScenario("worldOptions.inherit.title")}
            </span>
            <span className="rounded-md border border-border bg-background px-2 py-1">
              {tScenario("endDate.label")}:{" "}
              <span className="font-mono tech-tabular">
                {scenarioValue.endDate || tScenario("campaignOpening.notSet")}
              </span>
            </span>
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          aria-expanded={campaignSetupExpanded}
          onClick={() => setCampaignSetupExpanded((expanded) => !expanded)}
        >
          {campaignSetupExpanded ? (
            <ChevronUp aria-hidden className="h-4 w-4" />
          ) : (
            <ChevronDown aria-hidden className="h-4 w-4" />
          )}
          {campaignSetupExpanded
            ? tScenario("campaignOpening.closeWorkspace")
            : tScenario("campaignOpening.editWorkspace")}
        </Button>
      </div>

      {campaignSetupExpanded ? (
        <div className="max-h-[48vh] overflow-y-auto border-t border-border bg-background/40 p-4">
          <div className="grid gap-3 xl:grid-cols-2">
            <section
              className="rounded-lg border border-border bg-muted/20 px-4 py-3"
              aria-labelledby="campaign-opening-title"
            >
              <div className="space-y-1">
                <h3
                  id="campaign-opening-title"
                  className="text-base font-semibold tracking-tight"
                >
                  {tScenario("campaignOpening.title")}
                </h3>
                <p className="text-xs text-muted-foreground">
                  {tScenario("campaignOpening.intro")}
                </p>
              </div>
              <div className="mt-3 space-y-1">
                <p className="text-xs font-medium">
                  {tScenario("campaignOpening.currentLabel")}
                </p>
                <p className="font-mono text-sm tech-tabular">
                  {campaignStartDate ||
                    tScenario("campaignOpening.notSelected")}
                </p>
                <p className="text-xs text-muted-foreground">
                  {campaignOpeningBookmark && campaignOpeningIsSynced
                    ? tScenario("campaignOpening.linked", {
                        bookmark: resolveDisplayName(
                          locale,
                          campaignOpeningBookmark.display_name_locs,
                          campaignOpeningBookmark.name_en,
                          campaignOpeningBookmark.name_zh,
                          campaignOpeningBookmark.bookmark_id,
                        ),
                      })
                    : campaignOpeningBookmark
                      ? tScenario("campaignOpening.unlinked", {
                          date: campaignOpeningBookmark.date,
                        })
                      : tScenario("campaignOpening.noValidBookmark")}
                </p>
                {campaignOpeningBookmark ? (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => void handleUseCampaignOpening()}
                    disabled={campaignOpeningSaving || campaignOpeningIsSynced}
                  >
                    {campaignOpeningSaving
                      ? t("saving")
                      : campaignOpeningIsSynced
                        ? tScenario("campaignOpening.synced")
                        : tScenario("campaignOpening.syncEarliest")}
                  </Button>
                ) : null}
                {campaignOpeningStatus === "saved" ? (
                  <p className="text-xs text-success" role="status">
                    {t("saved")}
                  </p>
                ) : null}
                {campaignOpeningError ? (
                  <p className="text-xs text-destructive" role="alert">
                    {campaignOpeningError}
                  </p>
                ) : null}
              </div>
            </section>

            <div className="space-y-3">
              <ScenarioSettingsPanel
                value={scenarioValue}
                onChange={setScenarioValue}
                isStandaloneProject={isStandaloneProject}
                needsEndDate={
                  isStandaloneProject &&
                  scenarioNeedsExplicitEndDate(
                    latestCampaignBookmark?.date || campaignStartDate,
                  )
                }
              />
              <ScenarioWorldRiskPreview preview={scenarioWorldPreview} />
              <DialogSaveStatusBar
                saving={scenarioSaving}
                saved={scenarioStatus === "saved"}
                error={scenarioError}
                savingLabel={t("saving")}
                savedLabel={t("saved")}
              />
              <Button
                type="button"
                onClick={() => void handleSaveScenarioSettings()}
                disabled={scenarioSaving}
                size="sm"
              >
                {scenarioSaving
                  ? t("saving")
                  : tScenario("campaignOpening.saveCampaignSettings")}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
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
            {t("newBookmark")}
          </Button>
        }
      />
      <div className="px-3 pb-3">
        <BookmarkReplacePathNotice
          bookmarkCount={hasCustomBookmarkOpening(rows) ? rows.length : 0}
          title={t("replacePathAutoTitle")}
          description={t("replacePathAutoDesc")}
          singleBookmarkWarning={t("replacePathSingleWarning")}
          singleBookmarkAction={t("replacePathSingleAction")}
          addSecondBookmarkLabel={t("replacePathAddSecond")}
          secondBookmarkDraftMessage={t("replacePathSecondDraft")}
          navigationReadyMessage={t("replacePathSecondReady")}
          isCreatingSecond={activeId === "new" && rows.length === 1}
          showNavigationReady={navigationReadyStatus && rows.length >= 2}
          onAddSecond={() => void startNew()}
        />
      </div>
      <div className="px-3 pb-3">
        <DeadFlagBanner report={deadFlagReport} />
        {listLoadFailed && rows.length === 0 ? null : rows.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("emptyList")}
          </p>
        ) : filteredRows.length === 0 ? (
          <div className="p-4 flex flex-col items-center gap-2 text-center border border-dashed rounded-lg bg-muted/20">
            <SearchX className="h-5 w-5 text-muted-foreground" />
            <p className="text-xs text-muted-foreground">{t("noMatches")}</p>
          </div>
        ) : (
          <div className="space-y-0.5">
            {filteredRows.map((r) => (
              <button
                key={r.id}
                onClick={() => void selectRow(r)}

                className={`w-full text-left px-2 py-1.5 rounded text-xs transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
                  activeId === r.id
                    ? "bg-accent/10 text-accent font-medium shadow-[inset_3px_0_0_0] shadow-accent"
                    : "hover:bg-muted"
                }`}
              >
                <div className="font-mono truncate">{r.bookmark_id}</div>
                <div
                  className={`text-xs truncate ${
                    activeId === r.id
                      ? "text-accent/80"
                      : "text-muted-foreground"
                  }`}
                >
                  {resolveDisplayName(
                    locale,
                    r.display_name_locs,
                    r.name_en,
                    r.name_zh,
                    t("noName"),
                  )}
                  {r.default_flag && t("defaultMark")}
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );

  const editorForm = activeId !== null && (
    <BookmarkEditor
      projectId={projectId}
      activeId={activeId}
      draft={draft}
      onEdit={editDraft}
      rows={rows}
      saving={saving}
      saved={status === "saved"}
      error={error}
      diagnostics={diagnostics}
      onSave={() => void handleSave()}
      onDelete={() => void handleDelete()}
      onCancelNew={cancelNew}
      onPictureUploaded={() => void refresh()}
    />
  );

  return (
    <div className="flex h-full min-w-0 overflow-hidden">
      <div className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-clip">
        {campaignSetupPanel}
        <div className="min-h-0 flex-1 overflow-clip">
          <WorkbenchShell
            gameViewHref={gameViewHref(projectId, {
              panel: "gamesetup_scenario_window",
            })}
            listPanel={listPanel}
            editorPanel={
              <div>
                {activeId === null ? (
                  <div className="flex min-h-[24rem] items-center justify-center p-6">
                    {listLoadFailed && rows.length === 0 ? (
                      <WorkbenchLoadError onRetry={() => void refresh()} />
                    ) : rows.length === 0 ? (
                      zeroBookmarkHero
                    ) : (
                      <EditorEmptyState
                        className="max-w-xs"
                        icon={
                          <BookMarked className="h-6 w-6 text-muted-foreground/60" />
                        }
                        title={t("noSelectionTitle")}
                        description={t("selectHint")}
                      />
                    )}
                  </div>
                ) : (
                  editorForm
                )}
              </div>
            }
            previewPanel={
              <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
                {t("previewUnavailable")}
              </div>
            }
          />
        </div>
      </div>
    </div>
  );
}
