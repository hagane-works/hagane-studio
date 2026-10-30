"use client";

import { useCallback, useEffect, useRef, useState, useMemo } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { useParams, useSearchParams } from "@/platform/navigation";
import {
  Eye,
  FileCode2,
  FileX,
  FolderOpen,
  Link2,
  SearchX,
} from "lucide-react";
import { TreeValidationButton } from "@/components/workbench/TreeValidationDialog";
import { useDecisionStore } from "@/lib/store/decisionStore";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { useShallow } from "zustand/react/shallow";
import {
  useActiveCountry,
  useEffectiveCountryFilter,
  useProjectCountries,
  useProjectStore,
} from "@/lib/store/projectStore";
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
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { DecisionEditor } from "@/components/workbench/DecisionEditor";
import { ScriptEditor } from "@/shared/editors/ScriptEditor";
import { useEditorWiring } from "@/shared/editors/useEditorWiring";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { Decision } from "@/types";
import { decisionsApi, workbenchIconsApi } from "@/lib/api";
import {
  useVanillaCollisions,
  suggestNonCollidingId,
} from "@/lib/vanillaCollisions";
import { useDeadFlagLint } from "@/shared/useDeadFlagLint";
import { DeadFlagBanner } from "@/components/workbench/DeadFlagBanner";
import { FocusIconLibraryDialog } from "@/components/workbench/focus-graph/FocusIconLibraryDialog";
import { useBatchGfxApply } from "@/lib/hooks/useBatchGfxApply";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { DecisionCategoryPlacementEditor } from "@/components/workbench/DecisionCategoryPlacementEditor";
import { DecisionCategoryEditor } from "@/components/workbench/DecisionCategoryEditor";
import { DecisionInGamePreview } from "@/components/workbench/DecisionInGamePreview";
import { RelationsPanel } from "@/components/workbench/RelationsPanel";

export default function DecisionsPage() {
  const t = useTranslations("decisions");
  const tRel = useTranslations("relations");
  const tShell = useTranslations("workbenchShell");
  const tCommon = useTranslations("common");
  const locale = useContentLocaleStore((s) => s.locale);
  const params = useParams();
  const projectId = Number(params.id);
  const countryTag = useActiveCountry();

  const projectCountries = useProjectCountries();
  const hasCountries = projectCountries.length > 0;
  const [addCountryOpen, setAddCountryOpen] = useState(false);

  const catInputRef = useRef<HTMLInputElement>(null);

  const filterCountry = useEffectiveCountryFilter();
  const viewMode = useProjectStore((s) => s.viewMode);

  const { report: deadFlagReport } = useDeadFlagLint(projectId || null);

  const confirmDialog = useConfirm();

  const catListNav = useListboxNav();
  const decisionListNav = useListboxNav();

  const {
    categories,
    decisions,
    selectedCatId,
    selectedDecisionId,
    scriptPreview,
    loading,
    error,
    fetchCategories,
    selectCategory,
    createCategory,
    updateCategory,
    deleteCategory,
    fetchDecisions,
    selectDecision,
    createDecision,
    updateDecision,
    deleteDecision,
    fetchPreview,
  } = useDecisionStore(
    useShallow((s) => ({
      categories: s.categories,
      decisions: s.decisions,
      selectedCatId: s.selectedCatId,
      selectedDecisionId: s.selectedDecisionId,
      scriptPreview: s.scriptPreview,
      loading: s.loading,
      error: s.error,
      fetchCategories: s.fetchCategories,
      selectCategory: s.selectCategory,
      createCategory: s.createCategory,
      updateCategory: s.updateCategory,
      deleteCategory: s.deleteCategory,
      fetchDecisions: s.fetchDecisions,
      selectDecision: s.selectDecision,
      createDecision: s.createDecision,
      updateDecision: s.updateDecision,
      deleteDecision: s.deleteDecision,
      fetchPreview: s.fetchPreview,
    })),
  );

  const selectedCategory =
    categories.find((c) => c.id === selectedCatId) || null;

  const searchParams = useSearchParams();
  const didRestoreSelection = useRef(false);
  useEffect(() => {
    if (didRestoreSelection.current || categories.length === 0) return;
    const category = Number(searchParams?.get("category"));
    const decision = Number(searchParams?.get("decision"));
    if (category && categories.some((c) => c.id === category)) {
      selectCategory(category);
      if (decision) selectDecision(decision);
    }
    didRestoreSelection.current = true;
  }, [categories, searchParams, selectCategory, selectDecision]);

  const [newCatName, setNewCatName] = useState("");
  const [decisionSearch, setDecisionSearch] = useState("");

  const vanillaCollisionsData = useVanillaCollisions();
  const catCollision = useMemo(() => {
    const trimmed = newCatName.trim();
    const isColliding =
      vanillaCollisionsData.loaded &&
      !!trimmed &&
      vanillaCollisionsData.decisionCategoryIds.has(trimmed);
    const suggested = suggestNonCollidingId(trimmed, countryTag);
    return { isColliding, suggested };
  }, [vanillaCollisionsData, newCatName, countryTag]);

  const filteredCategories = useMemo(() => {
    if (!filterCountry) return categories;
    return categories.filter((c) => matchesCountryScope(c, filterCountry));
  }, [categories, filterCountry]);

  const filteredDecisions = useMemo(() => {
    const byCountry = filterCountry
      ? decisions.filter((d) => matchesCountryScope(d, filterCountry))
      : decisions;
    const q = decisionSearch.toLowerCase().trim();
    if (!q) return byCountry;
    return byCountry.filter(
      (d) =>
        d.decision_id.toLowerCase().includes(q) ||
        (d.name_en || "").toLowerCase().includes(q) ||
        (d.name_zh || "").toLowerCase().includes(q),
    );
  }, [decisions, decisionSearch, filterCountry]);

  useSWR(projectId ? ["decisions/categories", projectId] : null, () =>
    fetchCategories(projectId),
  );
  useSWR(
    projectId && selectedCatId
      ? ["decisions/list", projectId, selectedCatId]
      : null,
    () => fetchDecisions(projectId, selectedCatId!),
  );
  useSWR(
    projectId && selectedDecisionId
      ? ["decisions/preview", projectId, selectedDecisionId]
      : null,
    () => fetchPreview(projectId, selectedDecisionId!),
  );

  const parsePreviewCall = useCallback(
    (script: string) =>
      decisionsApi.parsePreview(projectId, selectedDecisionId!, script),
    [projectId, selectedDecisionId],
  );
  const wiring = useEditorWiring({
    workbench: "decisions",
    entityId: selectedDecisionId,
    scriptPreview,
    parsePreview: parsePreviewCall,
    applyEntity: (id, entityDict) =>
      updateDecision(projectId, id, entityDict as Partial<Decision>),
    onAfterApply: async (id) => {
      if (selectedCatId) await fetchDecisions(projectId, selectedCatId);
      await fetchPreview(projectId, id);
    },
    entityNoun: "decision",
    projectId,
  });

  useEffect(() => {
    if (!filterCountry) return;
    if (selectedCatId) {
      const sel = categories.find((c) => c.id === selectedCatId);
      if (sel && !matchesCountryScope(sel, filterCountry)) {
        selectCategory(null);
      }
    }
    if (selectedDecisionId) {
      const sel = decisions.find((d) => d.id === selectedDecisionId);
      if (sel && !matchesCountryScope(sel, filterCountry)) {
        selectDecision(null);
      }
    }
  }, [
    filterCountry,
    selectedCatId,
    selectedDecisionId,
    categories,
    decisions,
    selectCategory,
    selectDecision,
  ]);

  const selected = decisions.find((d) => d.id === selectedDecisionId) || null;
  const [decisionDraft, setDecisionDraft] = useState<Decision | null>(null);
  const previewDecision =
    selected && decisionDraft?.id === selected.id ? decisionDraft : selected;
  const decisionPreviewIsDraft = useMemo(
    () =>
      Boolean(selected && previewDecision) &&
      JSON.stringify(previewDecision) !== JSON.stringify(selected),
    [previewDecision, selected],
  );

  const handleCreateCategory = async () => {
    const name = newCatName.trim();
    if (!name) return;
    await createCategory(projectId, name, countryTag);
    setNewCatName("");
  };

  const handleDeleteCategory = async (catId: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (
      await confirmDialog({ body: t("confirmDeleteCategory"), danger: true })
    ) {
      await deleteCategory(projectId, catId);
    }
  };

  const handleCreateDecision = async () => {
    if (!selectedCatId) return;
    const cat = categories.find((c) => c.id === selectedCatId);
    if (!cat) return;

    const baseId = `${countryTag || "TAG"}_new_decision`;
    const existingIds = new Set(decisions.map((d) => d.decision_id));
    let candidate = baseId;
    let suffix = 2;
    while (existingIds.has(candidate)) {
      candidate = `${baseId}_${suffix}`;
      suffix += 1;
    }
    await createDecision(projectId, {
      category_id: selectedCatId,
      decision_id: candidate,
      country_tag: countryTag,
    });

    await fetchDecisions(projectId, selectedCatId);
  };

  const handleSave = async (data: Partial<Decision>) => {
    if (!selectedDecisionId) return;
    await updateDecision(projectId, selectedDecisionId, data);
  };

  const batchGfx = useBatchGfxApply({
    applyOne: async (decisionId, gfxName) => {
      await updateDecision(projectId, decisionId, { icon: gfxName });
    },
    onComplete: () => {
      if (selectedCatId) fetchDecisions(projectId, selectedCatId);
    },
  });

  const [placementOpen, setPlacementOpen] = useState(false);

  const handleDeleteDecision = async (id: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (
      await confirmDialog({ body: t("confirmDeleteDecision"), danger: true })
    ) {
      await deleteDecision(projectId, id);
    }
  };

  const zeroCategoryHero = (
    <EditorEmptyState
      variant="hero"
      icon={<FolderOpen className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyHeroTitle")}
      description={t("emptyHeroDesc")}
      actions={
        <>
          {hasCountries ? (
            <Button
              size="sm"
              onClick={() => {
                catInputRef.current?.focus({ preventScroll: true });
                catInputRef.current?.scrollIntoView({
                  behavior: "smooth",
                  block: "center",
                });
              }}
            >
              {t("emptyHeroFirstStep")}
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
        </>
      }
    />
  );

  return (
    <div className="flex h-full min-w-0 overflow-hidden">
      <div className="flex-1 min-w-0 h-full overflow-hidden">
        <WorkbenchShell
          gameViewHref={gameViewHref(projectId, {
            panel: "countrydecisionview",
            tag: countryTag,
          })}
          previewWidth="w-96"
          listPanel={
            <div className="p-3">
              {deadFlagReport && deadFlagReport.dead_flags.length > 0 && (
                <div className="mb-3">
                  <DeadFlagBanner report={deadFlagReport} />
                </div>
              )}

              <div className="mb-3">
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-sm font-semibold">
                    {t("categoriesHeading")}
                  </h3>
                  <div className="flex items-center gap-1">
                    <TreeValidationButton
                      projectId={projectId}
                      kind="decision"
                    />

                    <Button
                      size="sm"
                      variant="outline"
                      className="h-7 text-xs gap-1"
                      onClick={() => setPlacementOpen(true)}
                      disabled={categories.length === 0}
                      title={t("placementTooltip")}
                    >
                      {t("placementButton")}
                    </Button>
                  </div>
                </div>
                <div className="mb-2">
                  <div className="flex gap-1">
                    <Input
                      ref={catInputRef}
                      value={newCatName}
                      onChange={(e) => setNewCatName(e.target.value)}
                      placeholder={t("newCatPlaceholder")}
                      className={`text-xs h-8 font-mono ${
                        catCollision.isColliding
                          ? "border-destructive focus-visible:ring-destructive/40"
                          : ""
                      }`}
                      onKeyDown={(e) =>
                        e.key === "Enter" && handleCreateCategory()
                      }
                    />
                    <Button
                      size="sm"
                      className="px-2"
                      onClick={handleCreateCategory}
                      disabled={!newCatName.trim()}
                      title={
                        newCatName.trim()
                          ? t("createCategory")
                          : t("enterCategoryNameFirst")
                      }
                    >
                      +
                    </Button>
                  </div>

                  {catCollision.isColliding && (
                    <div className="mt-1 p-2 rounded border border-destructive/40 bg-destructive/5 text-xs space-y-1">
                      <p className="text-destructive font-medium">
                        ⚠ <code className="font-mono">{newCatName.trim()}</code>{" "}
                        {t("catCollisionWarn")}
                      </p>
                      <button
                        type="button"
                        onClick={() => setNewCatName(catCollision.suggested)}
                        className="text-xs underline decoration-dotted text-destructive/80 hover:text-destructive"
                      >
                        {t("renameTo")}{" "}
                        <code className="font-mono">
                          {catCollision.suggested}
                        </code>{" "}
                        {t("recommended")}
                      </button>
                    </div>
                  )}
                </div>

                <div
                  className="space-y-0.5 mb-3 outline-none"
                  {...catListNav.containerProps}
                  aria-label={t("categoriesHeading")}
                >
                  {(() => {
                    const { own, shared } = partitionByScope(
                      filteredCategories,
                      filterCountry,
                    );
                    const renderRow = (
                      cat: (typeof filteredCategories)[number],
                    ) => (
                      <div
                        key={cat.id}
                        {...catListNav.optionProps(cat.id === selectedCatId)}
                        className={`flex items-center justify-between px-2 py-1.5 rounded text-xs cursor-pointer transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
                          cat.id === selectedCatId
                            ? "bg-accent/10 text-accent font-medium shadow-[inset_3px_0_0_0] shadow-accent"
                            : "hover:bg-muted"
                        }`}
                        onClick={() => selectCategory(cat.id)}
                        title={cat.category_id}
                      >
                        <div className="truncate flex-1 flex items-center gap-1.5">
                          {viewMode === "all" ? (
                            <CountryTagChip tag={cat.country_tag} />
                          ) : (
                            <ScopeBadge
                              entity={cat}
                              activeCountry={filterCountry}
                            />
                          )}
                          <span className="truncate">
                            {resolveDisplayName(
                              locale,
                              cat.display_name_locs,
                              cat.name_en,
                              cat.name_zh,
                              cat.category_id,
                            )}
                          </span>
                        </div>
                        <span className="text-micro opacity-70 mx-1">
                          {cat.decision_count}
                        </span>
                        <button
                          className="text-micro opacity-50 hover:opacity-100"
                          onClick={(e) => handleDeleteCategory(cat.id, e)}
                        >
                          X
                        </button>
                      </div>
                    );
                    if (!filterCountry) {
                      return (
                        <>
                          <ScopeSectionHeader
                            variant="all"
                            count={filteredCategories.length}
                          />
                          {filteredCategories.map(renderRow)}
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
                  {filteredCategories.length === 0 &&
                    (categories.length === 0 ? (
                      <p className="p-4 text-center text-xs text-muted-foreground">
                        {t("noCategoriesYet")}
                      </p>
                    ) : (
                      <div className="p-6 flex flex-col items-center gap-3 text-center border border-dashed rounded-lg bg-muted/20">
                        <FolderOpen className="h-6 w-6 text-muted-foreground" />
                        <div className="space-y-1">
                          <p className="text-sm font-medium text-foreground">
                            {t("noMatchingCategories")}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {t("noCategoriesScoped", {
                              country: countryTag || t("thisCountry"),
                            })}
                          </p>
                        </div>
                      </div>
                    ))}
                </div>
              </div>

              {selectedCatId && (
                <>
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-semibold">
                      {t("decisionsHeading")}
                    </h3>
                    <Button
                      size="sm"
                      className="h-7 text-xs"
                      onClick={handleCreateDecision}
                    >
                      {t("addDecision")}
                    </Button>
                  </div>

                  {decisions.length > 5 && (
                    <div className="mb-2">
                      <Input
                        value={decisionSearch}
                        onChange={(e) => setDecisionSearch(e.target.value)}
                        placeholder={t("searchPlaceholder")}
                        className="text-xs h-7"
                      />
                    </div>
                  )}

                  {batchGfx.selectedIds.size > 0 && (
                    <div className="mb-2 rounded border border-accent/40 bg-accent/5 px-2 py-1.5 flex items-center gap-2 text-xs">
                      <span className="font-medium">
                        {t("selectedCount", {
                          count: batchGfx.selectedIds.size,
                        })}
                      </span>
                      <button
                        type="button"
                        onClick={batchGfx.openPicker}
                        disabled={batchGfx.progress !== null}
                        className="ml-auto px-2 py-0.5 rounded bg-accent text-accent-foreground hover:bg-accent/90 disabled:opacity-50"
                      >
                        {t("batchApplyGfx")}
                      </button>
                      <button
                        type="button"
                        onClick={batchGfx.clear}
                        className="text-muted-foreground hover:text-foreground ui-interactive"
                      >
                        {t("cancel")}
                      </button>
                    </div>
                  )}
                  {batchGfx.progress && (
                    <div className="mb-2 rounded border border-success/40 bg-success/5 px-2 py-1.5 text-xs">
                      Applying{" "}
                      <span className="font-mono">
                        {batchGfx.progress.gfxName}
                      </span>{" "}
                      · {batchGfx.progress.done}/{batchGfx.progress.total}
                      {batchGfx.progress.failed > 0 && (
                        <span className="text-destructive">
                          {" "}
                          · {batchGfx.progress.failed} failed
                        </span>
                      )}
                    </div>
                  )}

                  {loading && decisions.length === 0 ? (
                    <div className="p-4 space-y-1.5">
                      {[1, 2, 3].map((i) => (
                        <div
                          key={i}
                          className="h-7 rounded bg-muted/40 animate-pulse"
                        />
                      ))}
                    </div>
                  ) : error && decisions.length === 0 ? (
                    <WorkbenchLoadError
                      onRetry={() => {
                        if (selectedCatId)
                          fetchDecisions(projectId, selectedCatId);
                        else fetchCategories(projectId);
                      }}
                    />
                  ) : decisions.length === 0 ? (
                    <EditorEmptyState
                      icon={
                        <FileX className="h-6 w-6 text-muted-foreground/60" />
                      }
                      title={t("noDecisionsInCategory")}
                      actions={
                        <Button
                          size="sm"
                          onClick={() => {
                            handleCreateDecision();
                          }}
                        >
                          {t("emptySecondFirstStep")}
                        </Button>
                      }
                    />
                  ) : filteredDecisions.length === 0 ? (
                    <div className="p-4 flex flex-col items-center gap-2 text-center border border-dashed rounded-lg bg-muted/20">
                      <SearchX className="h-5 w-5 text-muted-foreground" />
                      <p className="text-xs text-muted-foreground">
                        {t("noMatchingDecisions")}
                      </p>
                    </div>
                  ) : (
                    <div
                      className="space-y-0.5 outline-none"
                      {...decisionListNav.containerProps}
                      aria-label={t("decisionsHeading")}
                    >
                      {filteredDecisions.map((d) => {
                        const isBatchSelected = batchGfx.selectedIds.has(d.id);
                        return (
                          <div
                            key={d.id}
                            {...decisionListNav.optionProps(
                              d.id === selectedDecisionId,
                            )}
                            className={`flex items-center justify-between px-2 py-1.5 rounded text-xs cursor-pointer transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
                              d.id === selectedDecisionId
                                ? "bg-accent/10 text-accent font-medium shadow-[inset_3px_0_0_0] shadow-accent"
                                : isBatchSelected
                                  ? "bg-success/10"
                                  : "hover:bg-muted"
                            }`}
                            onClick={() => selectDecision(d.id)}
                          >
                            <input
                              type="checkbox"
                              checked={isBatchSelected}
                              onClick={(e) => {
                                e.stopPropagation();
                                batchGfx.toggle(d.id, {
                                  range: e.shiftKey,
                                  sortedIds: filteredDecisions.map((x) => x.id),
                                });
                              }}
                              onChange={() => {}}
                              title={t("batchSelectTooltip")}
                              className="accent-accent h-3 w-3 shrink-0 mr-2"
                            />
                            <div className="truncate flex-1 flex items-center gap-1.5">
                              {viewMode === "all" ? (
                                <CountryTagChip tag={d.country_tag} />
                              ) : (
                                <ScopeBadge
                                  entity={d}
                                  activeCountry={filterCountry}
                                />
                              )}
                              {d.icon && (
                                <img
                                  src={workbenchIconsApi.decisionIconUrl(
                                    projectId,
                                    d.icon,
                                  )}
                                  alt=""
                                  className="w-5 h-4 object-contain shrink-0"
                                  onError={(e) => {
                                    (
                                      e.target as HTMLImageElement
                                    ).style.display = "none";
                                  }}
                                />
                              )}
                              <span className="font-mono truncate">
                                {d.decision_id}
                              </span>
                              {d.cost > 0 && (
                                <span className="ml-1 opacity-60 shrink-0">
                                  ({d.cost}PP)
                                </span>
                              )}
                            </div>
                            <button
                              className="text-micro opacity-50 hover:opacity-100 ml-1"
                              onClick={(e) => handleDeleteDecision(d.id, e)}
                            >
                              X
                            </button>
                          </div>
                        );
                      })}
                    </div>
                  )}

                  <p className="text-xs text-muted-foreground mt-3">
                    {decisionSearch
                      ? `${filteredDecisions.length}/${decisions.length} decisions`
                      : `${decisions.length} decision${decisions.length !== 1 ? "s" : ""}`}
                  </p>
                </>
              )}
            </div>
          }
          editorPanel={
            selected ? (
              <DecisionEditor
                decision={selected}
                onSave={handleSave}
                projectId={projectId}
                countryTag={countryTag}
                onDraftChange={setDecisionDraft}
                onIconUploaded={() => {
                  if (selectedCatId) fetchDecisions(projectId, selectedCatId);
                  if (selectedDecisionId)
                    fetchPreview(projectId, selectedDecisionId);
                }}
                onForked={async (forked) => {
                  if (selectedCatId)
                    await fetchDecisions(projectId, selectedCatId);
                  selectDecision(forked.id);
                }}
              />
            ) : selectedCategory ? (
              <DecisionCategoryEditor
                projectId={projectId}
                category={selectedCategory}
                onSave={(data) =>
                  updateCategory(projectId, selectedCategory.id, data)
                }
              />
            ) : (
              <div className="flex items-center justify-center h-full p-6">
                {error && categories.length === 0 ? (
                  <WorkbenchLoadError
                    onRetry={() => fetchCategories(projectId)}
                  />
                ) : categories.length === 0 ? (
                  zeroCategoryHero
                ) : (
                  <EditorEmptyState
                    className="max-w-xs"
                    icon={
                      <FolderOpen className="h-6 w-6 text-muted-foreground/60" />
                    }
                    title={t("noCategorySelectedTitle")}
                    description={t("noCategorySelectedDesc")}
                  />
                )}
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
                  content: previewDecision ? (
                    <div className="flex h-full min-h-0 flex-col p-3">
                      {decisionPreviewIsDraft && (
                        <div className="mb-2 flex shrink-0 justify-end">
                          <span className="rounded border border-accent/50 bg-accent/15 px-2 py-0.5 text-micro font-mono text-accent">
                            {tShell("liveDraftBadge")}
                          </span>
                        </div>
                      )}
                      <DecisionInGamePreview
                        decision={previewDecision}
                        category={selectedCategory}
                        projectId={projectId}
                      />
                    </div>
                  ) : (
                    <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
                      {tShell("livePreviewSelectHint")}
                    </div>
                  ),
                },
                {
                  id: "references",
                  label: tRel("tab"),
                  icon: <Link2 className="h-3.5 w-3.5" />,
                  hint: tShell("tabReferencesHint"),
                  content: (
                    <RelationsPanel
                      kind="decision"
                      sourceId={selectedDecisionId}
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
      <DecisionCategoryPlacementEditor
        open={placementOpen}
        onOpenChange={setPlacementOpen}
        projectId={projectId}
        categories={categories}
        onSaved={() => {
          fetchCategories(projectId);
        }}
      />

      <AddCountryDialog
        projectId={projectId}
        open={addCountryOpen}
        onOpenChange={setAddCountryOpen}
      />
    </div>
  );
}
