"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import useSWR from "swr";
import { useParams, useSearchParams } from "@/platform/navigation";
import {
  FileQuestion,
  Eye,
  Link2,
  FileCode2,
  Library,
  Lightbulb,
} from "lucide-react";
import { useIdeaStore } from "@/lib/store/ideaStore";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { useShallow } from "zustand/react/shallow";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import {
  useActiveCountry,
  useEffectiveCountryFilter,
  useProjectCountries,
  useProjectStore,
} from "@/lib/store/projectStore";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { AddCountryDialog } from "@/components/workbench/AddCountryDialog";
import { matchesCountryScope, partitionByScope } from "@/lib/utils/scope";
import {
  ScopeBadge,
  ScopeSectionHeader,
} from "@/components/workbench/ScopeBadge";
import { CountryTagChip } from "@/components/workbench/CountryTagChip";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { WorkbenchRightPane } from "@/components/workbench/WorkbenchRightPane";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import IdeaEditor from "@/components/workbench/IdeaEditor";
import { IdeaInGamePreview } from "@/components/workbench/IdeaInGamePreview";
import { ScriptEditor } from "@/shared/editors/ScriptEditor";
import { useEditorWiring } from "@/shared/editors/useEditorWiring";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { Button } from "@/components/ui/button";
import { ImportButton } from "@/components/workbench/ImportButton";
import { ideasApi, importApi, type VanillaIdeaEntry } from "@/lib/api";
import type { Idea, IdeaType } from "@/types";
import { IDEA_TYPE_OPTIONS, DESIGNER_TYPES } from "@/types";
import { useTranslations } from "use-intl";
import { FocusIconLibraryDialog } from "@/components/workbench/focus-graph/FocusIconLibraryDialog";
import { useBatchGfxApply } from "@/lib/hooks/useBatchGfxApply";
import { VanillaIdeasTab } from "@/components/workbench/VanillaIdeasTab";
import { RelationsPanel } from "@/components/workbench/RelationsPanel";

const TYPE_FILTER_GROUPS = [
  { value: "", labelKey: "typeFilterAll" },
  { value: "country", labelKey: "typeFilterSpirits" },
  { value: "hidden_ideas", labelKey: "typeFilterHidden" },
  { value: "designer", labelKey: "typeFilterDesigners" },
];

export default function IdeasPage() {
  const params = useParams();
  const projectId = Number(params.id);
  const countryTag = useActiveCountry();

  const filterCountry = useEffectiveCountryFilter();
  const viewMode = useProjectStore((s) => s.viewMode);

  const projectCountries = useProjectCountries();
  const hasCountries = projectCountries.length > 0;
  const [addCountryOpen, setAddCountryOpen] = useState(false);
  const t = useTranslations("ideas");

  const tEnums = useTranslations("enums");
  const tRel = useTranslations("relations");
  const tShell = useTranslations("workbenchShell");
  const tVanilla = useTranslations("vanillaBrowser");
  const tCommon = useTranslations("common");
  const locale = useContentLocaleStore((s) => s.locale);

  const listNav = useListboxNav();

  const confirmDialog = useConfirm();

  const {
    ideas,
    selectedId,
    filterType,
    scriptPreview,
    loading,
    error,
    fetchIdeas,
    setFilterType,
    selectIdea,
    createIdea,
    updateIdea,
    deleteIdea,
  } = useIdeaStore(
    useShallow((s) => ({
      ideas: s.ideas,
      selectedId: s.selectedId,
      filterType: s.filterType,
      scriptPreview: s.scriptPreview,
      loading: s.loading,
      error: s.error,
      fetchIdeas: s.fetchIdeas,
      setFilterType: s.setFilterType,
      selectIdea: s.selectIdea,
      createIdea: s.createIdea,
      updateIdea: s.updateIdea,
      deleteIdea: s.deleteIdea,
    })),
  );

  const fetchPreview = useIdeaStore((s) => s.fetchPreview);

  useSWR(projectId ? ["ideas/list", projectId] : null, () =>
    fetchIdeas(projectId),
  );

  useSWR(
    projectId ? ["ideas/preview", projectId, selectedId ?? null] : null,
    () =>
      selectedId
        ? fetchPreview(projectId, selectedId)
        : fetchPreview(projectId),
  );

  const parsePreviewCall = useCallback(
    (script: string) => ideasApi.parsePreview(projectId, selectedId!, script),
    [projectId, selectedId],
  );
  const wiring = useEditorWiring({
    workbench: "ideas",
    entityId: selectedId,
    scriptPreview,
    parsePreview: parsePreviewCall,
    applyEntity: (id, entityDict) =>
      updateIdea(projectId, id, entityDict as Partial<Idea>),
    onAfterApply: async (id) => {
      await fetchIdeas(projectId);
      await fetchPreview(projectId, id);
    },
    entityNoun: "idea",
    projectId,
  });

  const selected = ideas.find((i) => i.id === selectedId) || null;

  const searchParams = useSearchParams();
  const didRestoreSelection = useRef(false);
  useEffect(() => {
    if (didRestoreSelection.current || ideas.length === 0) return;
    const wanted = Number(searchParams?.get("idea"));
    if (wanted && ideas.some((i) => i.id === wanted)) selectIdea(wanted);
    didRestoreSelection.current = true;
  }, [ideas, searchParams, selectIdea]);
  const [ideaDraft, setIdeaDraft] = useState<Idea | null>(null);
  const previewIdea =
    selected && ideaDraft?.id === selected.id ? ideaDraft : selected;
  const ideaPreviewIsDraft = useMemo(
    () =>
      Boolean(selected && previewIdea) &&
      JSON.stringify(previewIdea) !== JSON.stringify(selected),
    [previewIdea, selected],
  );

  const batchGfx = useBatchGfxApply({
    applyOne: async (ideaId, gfxName) => {
      await updateIdea(projectId, ideaId, { picture: gfxName });
    },
    onComplete: () => {
      fetchIdeas(projectId);
    },
  });

  const handleCreate = async () => {
    await createIdea(projectId, {
      idea_type: (filterType as IdeaType) || "country",
      name: "New Idea",
      country_tag: countryTag,
    });
  };

  const handleUseAsTemplate = async (entry: VanillaIdeaEntry) => {
    await createIdea(projectId, {
      idea_type:
        (filterType as IdeaType) || (entry.category as IdeaType) || "country",
      name: entry.idea_id,
      country_tag: countryTag,
      picture: entry.picture || "",
    });
  };

  const handleSave = async (data: Partial<Idea>) => {
    if (!selectedId) return;
    await updateIdea(projectId, selectedId, data);
  };

  const handleDelete = async (id: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (await confirmDialog({ body: t("confirmDelete"), danger: true })) {
      await deleteIdea(projectId, id);
    }
  };

  const handleFilterChange = (filter: string) => {
    setFilterType(filter as IdeaType | "");

    if (filter === "designer" || filter === "") {
      fetchIdeas(projectId, "");
    } else {
      fetchIdeas(projectId, filter);
    }
  };

  const [searchQuery, setSearchQuery] = useState("");

  const typeFilteredIdeas =
    filterType === ("designer" as string)
      ? ideas.filter((i) => DESIGNER_TYPES.includes(i.idea_type))
      : ideas;

  const filteredIdeas = useMemo(() => {
    if (!filterCountry) return typeFilteredIdeas;
    return typeFilteredIdeas.filter((i) =>
      matchesCountryScope(i, filterCountry),
    );
  }, [typeFilteredIdeas, filterCountry]);

  useEffect(() => {
    if (!selectedId) return;
    const sel = ideas.find((i) => i.id === selectedId);
    if (!sel) return;
    if (filterCountry && !matchesCountryScope(sel, filterCountry)) {
      selectIdea(null);
    }
  }, [filterCountry, selectedId, ideas, selectIdea]);

  const searchedIdeas = useMemo(() => {
    const q = searchQuery.toLowerCase().trim();
    if (!q) return filteredIdeas;
    return filteredIdeas.filter((i) => {
      if (i.idea_id.toLowerCase().includes(q)) return true;
      if ((i.name_en || "").toLowerCase().includes(q)) return true;
      if ((i.name_zh || "").toLowerCase().includes(q)) return true;
      const locs = i.display_name_locs || {};
      return Object.values(locs).some((v) =>
        (v || "").toLowerCase().includes(q),
      );
    });
  }, [filteredIdeas, searchQuery]);

  const sortedIdeas = useMemo(
    () =>
      [...searchedIdeas].sort((a, b) => {
        if (a.idea_type !== b.idea_type)
          return a.idea_type.localeCompare(b.idea_type);
        return a.idea_id.localeCompare(b.idea_id);
      }),
    [searchedIdeas],
  );

  const getTypeLabel = (type: string) => {
    const opt = IDEA_TYPE_OPTIONS.find((o) => o.value === type);
    return opt ? tEnums(opt.label_i18n_key) : type;
  };

  return (
    <div className="flex h-full min-w-0 overflow-hidden">
      <div className="flex-1 min-w-0 h-full overflow-hidden">
        <WorkbenchShell
          gameViewHref={gameViewHref(projectId, {
            panel: "countrypoliticsview",
            tag: countryTag,
          })}
          previewWidth="w-96"
          listPanel={
            <div className="flex flex-col h-full">
              <div className="bg-muted/5">
                <WorkbenchListHeader
                  title={t("workbenchTitle")}
                  count={sortedIdeas.length}
                  totalCount={ideas.length}
                  searchValue={searchQuery}
                  onSearchChange={setSearchQuery}
                  searchPlaceholder={t("searchPlaceholder")}
                  action={
                    <>
                      <Button
                        size="sm"
                        className="text-xs font-semibold"
                        onClick={handleCreate}
                      >
                        {t("newShort")}
                      </Button>
                    </>
                  }
                  filters={
                    <div className="flex flex-wrap gap-1.5">
                      {TYPE_FILTER_GROUPS.map((opt) => {
                        const active =
                          filterType === opt.value ||
                          (opt.value === "" && !filterType);
                        return (
                          <button
                            key={opt.value}
                            className={`
                          px-2.5 py-1 rounded-md text-xs ui-interactive
                          ${
                            active
                              ? "bg-accent/10 text-accent font-semibold"
                              : "text-muted-foreground hover:text-foreground hover:bg-muted"
                          }
                        `}
                            onClick={() => handleFilterChange(opt.value)}
                          >
                            {t(opt.labelKey)}
                          </button>
                        );
                      })}
                    </div>
                  }
                />
              </div>

              {batchGfx.selectedIds.size > 0 && (
                <div className="mb-2 rounded border border-primary/40 bg-primary/5 px-2 py-1.5 flex items-center gap-2 text-xs">
                  <span className="font-medium">
                    {t("batchSelectedCount", {
                      count: batchGfx.selectedIds.size,
                    })}
                  </span>
                  <button
                    type="button"
                    onClick={batchGfx.openPicker}
                    disabled={batchGfx.progress !== null}
                    className="ml-auto px-2 py-0.5 rounded bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                  >
                    {t("batchApplyGfx")}
                  </button>
                  <button
                    type="button"
                    onClick={batchGfx.clear}
                    className="text-muted-foreground hover:text-foreground ui-interactive"
                  >
                    {t("batchCancel")}
                  </button>
                </div>
              )}
              {batchGfx.progress && (
                <div className="mb-2 rounded border border-success/40 bg-success/5 px-2 py-1.5 text-xs">
                  {t("batchApplying")}{" "}
                  <span className="font-mono">{batchGfx.progress.gfxName}</span>{" "}
                  · {batchGfx.progress.done}/{batchGfx.progress.total}
                  {batchGfx.progress.failed > 0 && (
                    <span className="text-destructive">
                      {" "}
                      · {t("batchFailed", { count: batchGfx.progress.failed })}
                    </span>
                  )}
                </div>
              )}

              <div className="flex-1 overflow-auto p-2">
                {loading && ideas.length === 0 ? (
                  <div className="p-4 space-y-1.5">
                    {[...Array(4)].map((_, i) => (
                      <div
                        key={i}
                        className="h-9 rounded-md bg-muted/40 animate-pulse"
                      />
                    ))}
                  </div>
                ) : ideas.length === 0 ? (
                  <p className="p-4 text-center text-xs text-muted-foreground">
                    {t("onboardTitle")}
                  </p>
                ) : sortedIdeas.length === 0 ? (
                  <div className="p-8 text-center border border-dashed rounded-lg bg-muted/20">
                    <p className="text-xs text-muted-foreground italic">
                      {t("noMatchFilters")}
                    </p>
                  </div>
                ) : (
                  <div
                    {...listNav.containerProps}
                    aria-label={t("workbenchTitle")}
                    className="space-y-0.5 outline-none"
                  >
                    {(() => {
                      const { own, shared } = partitionByScope(
                        sortedIdeas,
                        filterCountry,
                      );
                      const renderRow = (
                        idea: (typeof sortedIdeas)[number],
                      ) => {
                        const active = idea.id === selectedId;
                        const isBatchSelected = batchGfx.selectedIds.has(
                          idea.id,
                        );
                        return (
                          <div
                            key={idea.id}
                            {...listNav.optionProps(active)}
                            className={`
                          group flex items-center justify-between px-3 py-2 rounded-md cursor-pointer text-sm ui-interactive focus-visible:ring-1 focus-visible:ring-ring
                          ${
                            active
                              ? "bg-accent/10 text-accent shadow-[inset_3px_0_0_0] shadow-accent"
                              : isBatchSelected
                                ? "bg-success/10"
                                : "hover:bg-muted border-transparent"
                          }
                        `}
                            onClick={() => selectIdea(idea.id)}
                          >
                            <input
                              type="checkbox"
                              checked={isBatchSelected}
                              onClick={(e) => {
                                e.stopPropagation();
                                batchGfx.toggle(idea.id, {
                                  range: e.shiftKey,
                                  sortedIds: sortedIdeas.map((i) => i.id),
                                });
                              }}
                              onChange={() => {}}
                              title={t("batchSelectTooltip")}
                              className="accent-accent h-3 w-3 shrink-0 mr-2"
                            />
                            <div className="min-w-0 flex-1">
                              <p
                                className={`font-medium truncate flex items-center gap-2 ${active ? "text-accent" : "text-foreground"}`}
                              >
                                {viewMode === "all" ? (
                                  <CountryTagChip tag={idea.country_tag} />
                                ) : (
                                  <ScopeBadge
                                    entity={idea}
                                    activeCountry={filterCountry}
                                  />
                                )}
                                <span className="font-mono text-xs opacity-70">
                                  {idea.idea_id}
                                </span>
                              </p>
                              <p className="text-xs text-muted-foreground truncate mt-0.5">
                                {getTypeLabel(idea.idea_type)} ·{" "}
                                {resolveDisplayName(
                                  locale,
                                  idea.display_name_locs,
                                  idea.name_en,
                                  idea.name_zh,
                                  "Unnamed",
                                )}
                              </p>
                            </div>
                            <button
                              className="h-6 w-6 flex items-center justify-center rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 ui-interactive ml-2"
                              onClick={(e) => handleDelete(idea.id, e)}
                            >
                              <span className="text-xs">✕</span>
                            </button>
                          </div>
                        );
                      };
                      if (!filterCountry) {
                        return (
                          <>
                            <ScopeSectionHeader
                              variant="all"
                              count={sortedIdeas.length}
                            />
                            {sortedIdeas.map(renderRow)}
                          </>
                        );
                      }
                      return (
                        <>
                          {own.length > 0 && (
                            <>
                              <ScopeSectionHeader
                                variant="own"
                                count={own.length}
                              />
                              {own.map(renderRow)}
                            </>
                          )}
                          {shared.length > 0 && (
                            <>
                              <ScopeSectionHeader
                                variant="shared"
                                count={shared.length}
                              />
                              {shared.map(renderRow)}
                            </>
                          )}
                        </>
                      );
                    })()}
                  </div>
                )}
              </div>
            </div>
          }
          editorPanel={
            selected ? (
              <IdeaEditor
                idea={selected}
                projectId={projectId}
                onSave={handleSave}
                allIdeas={ideas}
                onDraftChange={setIdeaDraft}
                onPictureUploaded={() => {
                  fetchIdeas(projectId);
                  if (selectedId) fetchPreview(projectId, selectedId);
                }}
                onForked={async (forked) => {
                  await fetchIdeas(projectId);
                  selectIdea(forked.id);
                }}
              />
            ) : error && ideas.length === 0 ? (
              <div className="flex h-full items-center justify-center p-8">
                <WorkbenchLoadError onRetry={() => fetchIdeas(projectId)} />
              </div>
            ) : !loading && ideas.length === 0 ? (
              <div className="flex h-full items-center justify-center p-8">
                <EditorEmptyState
                  variant="hero"
                  className="max-w-md"
                  icon={
                    <Lightbulb className="h-10 w-10 text-muted-foreground/60" />
                  }
                  title={t("emptyHeroTitle")}
                  description={t("emptyHeroDesc")}
                  actions={
                    <>
                      {hasCountries ? (
                        <Button
                          size="sm"
                          onClick={() => {
                            handleCreate();
                          }}
                        >
                          {t("newIdea")}
                        </Button>
                      ) : (
                        <Button
                          size="sm"
                          onClick={() => {
                            setAddCountryOpen(true);
                          }}
                        >
                          {tCommon("emptyCtaAddCountry")}
                        </Button>
                      )}

                      <ImportButton
                        label={t("quickImportTxt")}
                        onImport={async (file) =>
                          importApi.ideas(projectId, file, countryTag)
                        }
                        onComplete={() => fetchIdeas(projectId)}
                      />
                    </>
                  }
                />
              </div>
            ) : (
              <div className="flex items-center justify-center h-full p-8">
                <div className="flex flex-col items-center gap-3 text-center max-w-[220px]">
                  <div className="w-12 h-12 rounded-xl bg-muted flex items-center justify-center border border-border">
                    <FileQuestion className="h-6 w-6 text-muted-foreground/60" />
                  </div>
                  <p className="text-sm font-medium text-foreground">
                    {t("noIdeaSelected")}
                  </p>
                  <p className="text-xs text-muted-foreground leading-relaxed">
                    {t("noIdeaSelectedHint")}
                  </p>
                </div>
              </div>
            )
          }
          previewPanel={
            <WorkbenchRightPane
              tabs={[
                {
                  id: "preview",
                  label: tShell("tabLivePreview"),
                  icon: <Eye className="h-3.5 w-3.5" />,
                  content: previewIdea ? (
                    <div className="flex h-full min-h-0 flex-col p-3">
                      <div className="mb-2 flex items-center justify-between gap-2">
                        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                          {t("inGamePreview")}
                        </p>
                        {ideaPreviewIsDraft && (
                          <span className="rounded border border-accent/50 bg-accent/15 px-2 py-0.5 text-micro font-mono text-accent">
                            {tShell("liveDraftBadge")}
                          </span>
                        )}
                      </div>
                      <IdeaInGamePreview
                        idea={previewIdea}
                        projectId={projectId}
                        isDraft={ideaPreviewIsDraft}
                      />
                    </div>
                  ) : (
                    <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
                      {tShell("livePreviewSelectHint")}
                    </div>
                  ),
                },
                {
                  id: "vanilla",
                  label: tVanilla("tabVanillaIdeas"),
                  icon: <Library className="h-3.5 w-3.5" />,
                  hint: tVanilla("tabHintIdeas"),
                  content: (
                    <VanillaIdeasTab
                      onUseAsTemplate={handleUseAsTemplate}
                      canUseAsTemplate={!!filterType}
                      useAsTemplateDisabledHint={tVanilla("pickCategoryFirst")}
                    />
                  ),
                },
                {
                  id: "references",
                  label: tRel("tab"),
                  icon: <Link2 className="h-3.5 w-3.5" />,
                  hint: tShell("tabReferencesHint"),
                  content: (
                    <RelationsPanel
                      kind="idea"
                      sourceId={selectedId}
                      projectId={projectId}
                    />
                  ),
                },
                {
                  id: "script",
                  label: tShell("tabScript"),
                  icon: <FileCode2 className="h-3.5 w-3.5" />,
                  hint: tShell("tabScriptHint"),
                  content: (
                    <ScriptEditor
                      value={wiring.scriptDraft}
                      onChange={wiring.setScriptDraft}
                      onBlur={wiring.triggerParse}
                      markers={wiring.markers}
                      indicator={wiring.indicator}
                      onResetToForm={wiring.handleResetToForm}
                      onApply={wiring.handleApplyEditor}
                      canApply={wiring.canApply}
                      applyLabel={t("applyLabel")}
                    />
                  ),
                },
              ]}
            />
          }
        />
      </div>
      <FocusIconLibraryDialog
        open={batchGfx.pickerOpen}
        onOpenChange={(o) =>
          o ? batchGfx.openPicker() : batchGfx.closePicker()
        }
        projectId={projectId}
        onSelect={batchGfx.applyGfx}
      />

      <AddCountryDialog
        projectId={projectId}
        open={addCountryOpen}
        onOpenChange={setAddCountryOpen}
      />
    </div>
  );
}
