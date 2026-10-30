"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useLinkedRowSelection } from "@/lib/workbench-state/useLinkedRowSelection";
import { useRouter } from "@/i18n/navigation";
import {
  POLITICS_WORKSPACE_ROUTES,
  politicsWorkspaceHref,
} from "@/lib/workbench/politicsWorkspaceRoutes";
import { useLocale, useTranslations } from "use-intl";
import type { CosmeticTagView } from "@/lib/api/cosmeticTags";
import { useVanillaTagName } from "@/lib/vanillaCountryTags";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { useEditorDirtyStore } from "@/lib/store/editorDirtyStore";
import { useIdeologyOptions } from "@/lib/ideology/useIdeologyOptions";
import { FileX, SearchX } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  historyFilesApi,
  countriesApi,
  type CountryRow,
  type HistoryFileRow,
} from "@/lib/api";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { useDeadFlagLint } from "@/shared/useDeadFlagLint";
import { DeadFlagBanner } from "@/components/workbench/DeadFlagBanner";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import {
  HistoryFileEditor,
  blankHistoryFileDraft,
  historyRowToDraft,
  type HistoryFileDraft,
} from "@/components/workbench/HistoryFileEditor";
import {
  useActiveCountry,
  useEffectiveCountryFilter,
} from "@/lib/store/projectStore";
import { matchesCountryScope } from "@/lib/utils/scope";
import { ScopeSectionHeader } from "@/components/workbench/ScopeBadge";
import {
  beginDraftSave,
  editDraftSession,
  emptyDraftSession,
  openDraftSession,
  resolveDraftSave,
  stableDraftFingerprint,
  type DraftSession,
} from "@/lib/workbench-state/draftSession";

const blankDraft = blankHistoryFileDraft;
const rowToDraft = historyRowToDraft;
type Draft = HistoryFileDraft;

interface HistoryFilesContentProps {
  projectId: number;
}

export function HistoryFilesContent({ projectId }: HistoryFilesContentProps) {
  const t = useTranslations("historyFilesDialog");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const confirmDialog = useConfirm();
  const locale = useLocale();
  const vanillaName = useVanillaTagName();
  const [countries, setCountries] = useState<CountryRow[]>([]);
  useEffect(() => {
    let cancelled = false;
    countriesApi
      .list(projectId)
      .then((response) => {
        if (!cancelled) setCountries(response.countries);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [projectId]);
  const countryNames = useMemo(
    () =>
      Object.fromEntries(
        countries.map((country) => {
          const name =
            country.display_name_locs?.[locale] ||
            (locale === "zh"
              ? country.name_zh
              : locale === "en"
                ? country.name_en
                : "") ||
            (vanillaName(country.tag) ?? country.name) ||
            country.tag;
          return [country.tag, name];
        }),
      ),
    [countries, locale, vanillaName],
  );
  const open = true;

  const activeCountry = useActiveCountry();

  const filterCountry = useEffectiveCountryFilter();
  const [rows, setRows] = useState<HistoryFileRow[]>([]);
  const [activeId, setActiveId] = useState<number | "new" | null>(null);

  const [search, setSearch] = useState("");
  const [draft, setDraft] = useState<Draft>(() => blankDraft());
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [listLoadFailed, setListLoadFailed] = useState(false);
  const [savedAppearance, setSavedAppearance] = useState("");
  const [latestAppearance, setLatestAppearance] =
    useState<CosmeticTagView | null>(null);
  const [status, setStatus] = useState<"" | "saved" | "deleted">("");
  const statusClearTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const ownership = useRef<DraftSession<Draft>>(emptyDraftSession());
  const latestListRequest = useRef(0);
  useRegisterDirty(
    `history-files:${projectId}`,
    ownership.current.dirty || saving,
  );
  const fingerprint = useCallback(
    (value: Draft) => stableDraftFingerprint(value),
    [],
  );

  const { groupOptions } = useIdeologyOptions();

  const { report: deadFlagReport } = useDeadFlagLint(open ? projectId : null);

  const clearStatusLater = useCallback(() => {
    if (statusClearTimer.current) clearTimeout(statusClearTimer.current);
    statusClearTimer.current = setTimeout(() => {
      statusClearTimer.current = null;
      setStatus("");
    }, 1500);
  }, []);

  useEffect(
    () => () => {
      if (statusClearTimer.current) clearTimeout(statusClearTimer.current);
    },
    [],
  );

  const refresh = useCallback(async (): Promise<HistoryFileRow[] | null> => {
    const request = ++latestListRequest.current;
    try {
      const res = await historyFilesApi.list(projectId);
      if (request !== latestListRequest.current) return null;
      setRows(res);
      setListLoadFailed(false);
      return res;
    } catch {
      if (request === latestListRequest.current) setListLoadFailed(true);
      return null;
    }
  }, [projectId]);

  const confirmReplacement = useCallback(
    async (
      nextTag: string | undefined,
      replaceHistory: boolean,
      deletingHistory = false,
    ) => {
      const session = ownership.current;
      const currentTag = session.draft?.set_cosmetic_tag;
      const losesCosmetic =
        !!currentTag &&
        currentTag !== nextTag &&
        !!useEditorDirtyStore.getState().dirtyWorkbenches[
          `cosmetic-tag:${projectId}:${currentTag}`
        ];
      const losesAssignment =
        replaceHistory &&
        !!useEditorDirtyStore.getState().dirtyWorkbenches[
          `appearance-setup:${projectId}:${session.draft?.country_tag}`
        ];
      const losesDraft =
        (replaceHistory && session.dirty) || losesCosmetic || losesAssignment;
      if (losesDraft || deletingHistory) {
        const body = deletingHistory
          ? [
              t("confirmDelete"),
              ...(losesDraft ? [tCommon("discardUnsavedConfirm")] : []),
            ].join("\n\n")
          : tCommon("discardUnsavedConfirm");
        if (!(await confirmDialog({ body, danger: deletingHistory })))
          return null;
      }
      return ownership.current === session ? session : null;
    },
    [confirmDialog, projectId, t, tCommon],
  );

  const selectRow = useCallback(
    async (row: HistoryFileRow) => {
      const next = rowToDraft(row);
      const session = await confirmReplacement(next.set_cosmetic_tag, true);
      if (!session || ownership.current !== session) return;
      ownership.current = openDraftSession(
        session,
        String(row.id),
        fingerprint(next),
        next,
      );
      setActiveId(row.id);
      setSavedAppearance(row.set_cosmetic_tag || "");
      setDraft(next);
      setStatus("");
      setError(null);
    },
    [confirmReplacement, fingerprint],
  );

  useLinkedRowSelection("hf", rows, selectRow);

  const startNew = useCallback(async () => {
    const next = blankDraft();
    const session = await confirmReplacement(next.set_cosmetic_tag, true);
    if (!session || ownership.current !== session) return;
    ownership.current = openDraftSession(
      session,
      "new",
      fingerprint(next),
      next,
    );
    setActiveId("new");
    setSavedAppearance("");
    setDraft(next);
    setStatus("");
    setError(null);
  }, [confirmReplacement, fingerprint]);

  const filteredRows = useMemo(() => {
    const scoped = rows.filter((r) => matchesCountryScope(r, filterCountry));
    const needle = search.trim().toLowerCase();
    if (!needle) return scoped;
    return scoped.filter(
      (r) =>
        r.country_tag.toLowerCase().includes(needle) ||
        countryNames[r.country_tag]?.toLowerCase().includes(needle) ||
        (r.set_politics_ruling_party || "").toLowerCase().includes(needle),
    );
  }, [rows, filterCountry, search, countryNames]);

  useEffect(() => {
    if (open) {
      void refresh();
    } else {
      setActiveId(null);
      setError(null);
      setStatus("");
      ownership.current = emptyDraftSession();
    }
  }, [open, refresh]);

  const update = <K extends keyof Draft>(key: K, value: Draft[K]) => {
    setStatus("");
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
        const response = await historyFilesApi.create(projectId, savingDraft);
        const created = await historyFilesApi.get(projectId, response.id);
        if ((created.set_cosmetic_tag || "") !== savingDraft.set_cosmetic_tag)
          throw new Error(t("readbackFailed"));
        await refresh();
        const saved = rowToDraft(created);
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(saved),
          saved,
        );
        if (ownership.current.identity === "new") {
          const current = ownership.current.draft ?? saved;
          ownership.current = openDraftSession(
            ownership.current,
            String(created.id),
            fingerprint(saved),
            saved,
          );
          if (fingerprint(current) !== fingerprint(saved)) {
            ownership.current = editDraftSession(
              ownership.current,
              current,
              fingerprint(current),
            );
          }
          setActiveId(created.id);
          setSavedAppearance(created.set_cosmetic_tag || "");
          setDraft(current);
        }
      } else if (typeof activeId === "number") {
        await historyFilesApi.update(projectId, activeId, savingDraft);
        const updated = await historyFilesApi.get(projectId, activeId);
        if ((updated.set_cosmetic_tag || "") !== savingDraft.set_cosmetic_tag)
          throw new Error(t("readbackFailed"));
        await refresh();
        const saved = rowToDraft(updated);
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(saved),
          saved,
        );
        if (ownership.current.identity === String(updated.id)) {
          setSavedAppearance(updated.set_cosmetic_tag || "");
          setDraft(ownership.current.draft!);
        }
      }
      setStatus("saved");
      clearStatusLater();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (typeof activeId !== "number") return;
    const session = await confirmReplacement(undefined, true, true);
    if (!session || ownership.current !== session) return;
    setDeleting(true);
    setSaving(true);
    setError(null);
    try {
      await historyFilesApi.remove(projectId, activeId);
      await refresh();
      if (ownership.current === session) {
        ownership.current = emptyDraftSession();
        setActiveId(null);
        setDraft(blankDraft());
      }
      setStatus("deleted");
      clearStatusLater();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setDeleting(false);
      setSaving(false);
    }
  };

  const zeroHistoryFileHero = (
    <EditorEmptyState
      variant="hero"
      icon={<FileX className="h-10 w-10 text-muted-foreground/60" />}
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
            {t("newFile")}
          </Button>
        }
      />
      <div className="px-3 pb-3">
        <DeadFlagBanner report={deadFlagReport} />

        {!filterCountry && filteredRows.length > 0 && (
          <ScopeSectionHeader variant="all" count={filteredRows.length} />
        )}
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
              <FileX className="w-6 h-6 text-muted-foreground/60" />
              <p className="text-xs font-medium text-foreground">
                {t("emptyAll")}
              </p>
              <p className="text-xs text-muted-foreground">
                {t("emptyScoped", {
                  country: activeCountry || t("currentCountry"),
                })}
              </p>
            </div>
          )
        ) : (
          <div className="space-y-0.5">
            {filteredRows.map((r) => (
              <button
                key={r.id}
                onClick={() => void selectRow(r)}

                className={`w-full text-left px-2 py-1.5 rounded text-xs transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
                  activeId === r.id
                    ? "bg-accent/10 text-accent shadow-[inset_3px_0_0_0] shadow-accent"
                    : "hover:bg-muted"
                }`}
              >
                <div>{countryNames[r.country_tag] || r.country_tag}</div>
                <div
                  className={`text-xs truncate ${
                    activeId === r.id
                      ? "text-accent/80"
                      : "text-muted-foreground"
                  }`}
                >
                  {groupOptions.find(
                    (option) => option.value === r.set_politics_ruling_party,
                  )?.label ||
                    r.set_politics_ruling_party ||
                    t("noRulingParty")}{" "}
                  ·{" "}
                  {t("listSummary", {
                    ideas: r.add_ideas_extra?.length || 0,
                    focuses: r.complete_national_focus_ids?.length || 0,
                  })}
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );

  const editorForm = activeId !== null && (
    <HistoryFileEditor
      projectId={projectId}
      activeId={activeId}
      draft={draft}
      onChange={update}
      rows={rows}
      countries={countries}
      countryNames={countryNames}
      onSelectCountry={async (nextCountry) => {
        const session = ownership.current;
        const setupDirty =
          useEditorDirtyStore.getState().dirtyWorkbenches[
            `appearance-setup:${projectId}:${draft.country_tag}`
          ];
        if (
          setupDirty &&
          !(await confirmDialog({ body: tCommon("discardUnsavedConfirm") }))
        )
          return;
        if (ownership.current === session) update("country_tag", nextCountry);
      }}
      savedAppearance={savedAppearance}
      latestAppearance={latestAppearance}
      onAppearanceView={setLatestAppearance}
      onAppearanceChange={async (nextTag) => {
        if (!(await confirmReplacement(nextTag, false))) return false;
        update("set_cosmetic_tag", nextTag);
        return true;
      }}
      onManageIdeologies={async () => {
        if (!(await confirmReplacement(undefined, true))) return;
        router.push(
          politicsWorkspaceHref(
            projectId,
            POLITICS_WORKSPACE_ROUTES.ideologies,
          ),
        );
      }}
      onCancelNew={async () => {
        if (!(await confirmReplacement(undefined, true))) return;
        ownership.current = emptyDraftSession();
        setActiveId(null);
        setDraft(blankDraft());
      }}
      saving={saving}
      deleting={deleting}
      saved={status === "saved"}
      dirty={ownership.current.dirty}
      error={error}
      onSave={handleSave}
      onDelete={handleDelete}
    />
  );

  return (
    <div className="flex h-full min-w-0 overflow-hidden">
      <div className="flex-1 min-w-0 h-full overflow-hidden">
        <WorkbenchShell
          gameViewHref={gameViewHref(projectId, {
            panel: "countrypoliticsview",
            tag: activeCountry,
          })}
          listPanel={listPanel}
          editorPanel={
            activeId === null ? (
              <div className="flex h-full items-center justify-center p-6">
                {listLoadFailed && rows.length === 0 ? (
                  <WorkbenchLoadError onRetry={() => void refresh()} />
                ) : rows.length === 0 ? (
                  zeroHistoryFileHero
                ) : (
                  <EditorEmptyState
                    className="max-w-xs"
                    icon={
                      <FileX className="h-6 w-6 text-muted-foreground/60" />
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
  );
}
