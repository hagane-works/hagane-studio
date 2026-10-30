"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import useSWR from "swr";
import dynamic from "@/platform/dynamic";
import { useParams, useSearchParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import { useTechnologyStore } from "@/lib/store/technologyStore";
import { buildConnectOps, buildDisconnectOps } from "@/lib/techEdgeSync";
import { useShallow } from "zustand/react/shallow";
import { useActiveCountry } from "@/lib/store/projectStore";
import { useLocaleStore } from "@/lib/store/localeStore";
import { localizedOptionLabel } from "@/lib/i18n/optionLabel";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import {
  TechEditor,
  TECH_LINE_OPTIONS,
} from "@/components/workbench/TechEditor";
import { TechSurfacePanel } from "@/components/workbench/TechSurfacePanel";
import { ScriptEditor } from "@/shared/editors/ScriptEditor";
import { useEditorWiring } from "@/shared/editors/useEditorWiring";
import { Button } from "@/components/ui/button";
import { technologiesApi } from "@/lib/api";
import { useTechIconInventory } from "@/lib/hooks/useIconInventory";
import {
  LedgerTabs,
  type TechLineOption,
} from "@/components/workbench/tech-graph/LedgerTabs";
import { TechCommandSearch } from "@/components/workbench/tech-graph/TechCommandSearch";
import { RelationsPanel } from "@/components/workbench/RelationsPanel";
import {
  filterByDisplayTechLine,
  applyTechVisibilityMode,
  bfsTreeLayout,
  computeRowBackfill,
  parseYearRowTable,
  buildYearRowScale,
  resolveTechLineLayout,
  resolveLayoutClass,
  updateTechnologyFolderPlacement,
  type CategoryColumn,
  type TechVisibilityMode,
  type ReachabilityMap,
} from "@/lib/techGraph";
import { useCountryAliases } from "@/lib/country-aliases";
import { useTechSections } from "@/lib/visualSectionMapping";
import { Eye, EyeOff, FlaskConical, Unlock } from "lucide-react";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import type { Technology } from "@/types";
import type { TechSurfaceStatus } from "@/lib/techSurface";

function TechGraphLoadingFallback() {
  const t = useTranslations("technology");
  return (
    <div className="flex-1 flex items-center justify-center text-muted-foreground">
      {t("loadingTechGraph")}
    </div>
  );
}

const TechFlowGraph = dynamic(
  () =>
    import("@/components/workbench/tech-graph/TechFlowGraph").then(
      (m) => m.TechFlowGraph,
    ),
  {
    ssr: false,
    loading: () => <TechGraphLoadingFallback />,
  },
);

export default function TechnologyPage() {
  const params = useParams();
  const projectId = Number(params.id);
  const countryTag = useActiveCountry();

  const locale = useLocaleStore((s) => s.locale);
  const t = useTranslations("technology");
  const tRel = useTranslations("relations");
  const confirmDialog = useConfirm();

  const {
    technologies,
    selectedTechLine,
    selectedId,
    scriptPreview,
    loading,
    error,
    fetchTechnologies,
    setTechLine,
    selectTech,
    createTech,
    updateTech,
    updateTechBatch,
    deleteTech,
    fetchPreview,
  } = useTechnologyStore(
    useShallow((s) => ({
      technologies: s.technologies,
      selectedTechLine: s.selectedTechLine,
      selectedId: s.selectedId,
      scriptPreview: s.scriptPreview,
      loading: s.loading,
      error: s.error,
      fetchTechnologies: s.fetchTechnologies,
      setTechLine: s.setTechLine,
      selectTech: s.selectTech,
      createTech: s.createTech,
      updateTech: s.updateTech,
      updateTechBatch: s.updateTechBatch,
      deleteTech: s.deleteTech,
      fetchPreview: s.fetchPreview,
    })),
  );

  const [iconCacheBust, setIconCacheBust] = useState(0);

  const { icons: availableTechIcons, refresh: refreshAvailableTechIcons } =
    useTechIconInventory(projectId || null);

  const [rightPaneTab, setRightPaneTab] = useState<"preview" | "relations">(
    "preview",
  );

  const [customLines, setCustomLines] = useState<string[]>([]);
  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch(
          `/api/go/projects/${projectId}/technologies/known-folders`,
        );
        if (!res.ok) return;
        const data = await res.json();
        if (!cancelled) {
          setCustomLines(data.project_custom_lines || []);
        }
      } catch {}
    })();
    return () => {
      cancelled = true;
    };
  }, [projectId]);

  const techLineList: TechLineOption[] = useMemo(() => {
    const base: TechLineOption[] = TECH_LINE_OPTIONS.map((o) => ({
      value: o.value,
      label: localizedOptionLabel(o.label, locale),
      folder: o.folder,
    }));
    const customEntries: TechLineOption[] = customLines.map((line) => ({
      value: line,
      label: t("customLine", { line }),
      folder: `${line}_folder`,
    }));
    return [...base, ...customEntries];
  }, [customLines, locale, t]);

  useSWR(projectId ? ["tech/list", projectId] : null, () =>
    fetchTechnologies(projectId),
  );
  const { data: techSurfaceStatus } = useSWR<TechSurfaceStatus>(
    projectId ? ["tech/surface", projectId] : null,
    () => technologiesApi.surfaceStatus(projectId),
  );
  const p0SurfaceEnabled = techSurfaceStatus?.enabled === true;

  useSWR(
    projectId
      ? ["tech/preview", projectId, selectedTechLine, selectedId ?? null]
      : null,
    () =>
      selectedId
        ? fetchPreview(projectId, selectedTechLine, selectedId)
        : fetchPreview(projectId, selectedTechLine),
  );

  const parsePreviewCall = useCallback(
    (script: string) =>
      technologiesApi.parsePreview(projectId, selectedId!, script),
    [projectId, selectedId],
  );
  const wiring = useEditorWiring({
    workbench: "technology",
    entityId: selectedId,
    scriptPreview,
    parsePreview: parsePreviewCall,
    applyEntity: (id, entityDict) =>
      updateTech(projectId, id, entityDict as Partial<Technology>),
    onAfterApply: async (id) => {
      await fetchTechnologies(projectId, selectedTechLine);
      await fetchPreview(projectId, selectedTechLine, id);
    },
    entityNoun: "technology",
    projectId,
  });

  const [visibilityMode, setVisibilityMode] =
    useState<TechVisibilityMode>("reachability");
  const [reachable, setReachable] = useState<ReachabilityMap | null>(null);
  const [techDraft, setTechDraft] = useState<Technology | null>(null);

  const [selectedPlacementIndices, setSelectedPlacementIndices] = useState<
    Record<number, number>
  >({});
  const selectedPlacementIndex =
    selectedId == null ? 0 : (selectedPlacementIndices[selectedId] ?? 0);
  const handleSelectedPlacementIndexChange = useCallback(
    (index: number) => {
      if (selectedId == null) return;
      setSelectedPlacementIndices((current) => ({
        ...current,
        [selectedId]: index,
      }));
    },
    [selectedId],
  );

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!projectId || !countryTag) {
        if (!cancelled) setReachable(null);
        return;
      }
      try {
        const data = await technologiesApi.reachability(projectId, countryTag);
        if (!cancelled) setReachable(new Set(data.tech_ids));
      } catch {
        if (!cancelled) setReachable(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [projectId, countryTag]);

  const countryAliases = useCountryAliases();
  const visibleTechnologies = useMemo(
    () =>
      applyTechVisibilityMode(technologies, {
        activeCountry: countryTag,
        mode: visibilityMode,
        reachable,
        countryTags: countryAliases.tags,
      }),
    [technologies, countryTag, visibilityMode, reachable, countryAliases],
  );

  const visualTechnologies = useMemo(() => {
    if (!techDraft || techDraft.id !== selectedId) return visibleTechnologies;
    return visibleTechnologies.map((tech) =>
      tech.id === techDraft.id ? techDraft : tech,
    );
  }, [visibleTechnologies, techDraft, selectedId]);

  const techsForLine = useMemo(
    () => filterByDisplayTechLine(visualTechnologies, selectedTechLine),
    [visualTechnologies, selectedTechLine],
  );

  const selected = technologies.find((t) => t.id === selectedId) || null;

  const searchParams = useSearchParams();
  const didRestoreSelection = useRef(false);
  useEffect(() => {
    if (didRestoreSelection.current || technologies.length === 0) return;
    const wanted = Number(searchParams?.get("tech"));
    if (wanted && technologies.some((t) => t.id === wanted)) selectTech(wanted);
    didRestoreSelection.current = true;
  }, [technologies, searchParams, selectTech]);

  useEffect(() => {
    if (selected && selected.tech_line !== selectedTechLine) {
      setTechLine(selected.tech_line);

      requestAnimationFrame(() => selectTech(selected.id));
    }
  }, [selected, selectedTechLine, setTechLine, selectTech]);

  const handleSwitchLine = (line: string) => {
    setTechLine(line);
  };

  const handleSelectFromSearch = (techDbId: number, techLine: string) => {
    if (techLine !== selectedTechLine) {
      setTechLine(techLine);
      requestAnimationFrame(() => selectTech(techDbId));
    } else {
      selectTech(techDbId);
    }
  };

  const handleCreateFromCanvas = async (data?: {
    start_year?: number;
    position_x?: number;
    position_y?: number;
  }) => {
    const lineOpt = techLineList.find((o) => o.value === selectedTechLine);
    await createTech(projectId, {
      tech_line: selectedTechLine,
      folder: lineOpt?.folder || `${selectedTechLine}_folder`,
      folder_placements: [
        {
          folder: lineOpt?.folder || `${selectedTechLine}_folder`,
          x: data?.position_x ?? 0,
          y: data?.position_y ?? 0,
        },
      ],
      country_tag: "",
      start_year: data?.start_year ?? 1936,
      position_x: data?.position_x ?? 0,
      position_y: data?.position_y ?? 0,

      position_y_is_explicit: data?.position_y !== undefined,
    });
  };

  const handleSave = async (data: Partial<Technology>) => {
    if (!selectedId) return;
    await updateTech(projectId, selectedId, data);
  };

  const handleDelete = async (id: number) => {
    if (await confirmDialog({ body: t("deleteConfirm"), danger: true })) {
      await deleteTech(projectId, id);
    }
  };

  const handleUpdatePosition = useCallback(
    async (
      techDbId: number,
      positionX: number,
      cell: { position_y: number } | null,
      fallbackStartYear: number,
      needsRowBackfill = false,
      placementIndex = 0,
    ) => {
      const legacyPayload: Partial<Technology> = cell
        ? {
            position_x: positionX,
            position_y: cell.position_y,
            position_y_is_explicit: true,
          }
        : {
            position_x: positionX,
            start_year: fallbackStartYear,
            position_y_is_explicit: false,
          };
      const current = technologies.find((tech) => tech.id === techDbId);
      const payload: Partial<Technology> = current
        ? (() => {
            const placement = updateTechnologyFolderPlacement(
              current,
              placementIndex,
              {
                x: legacyPayload.position_x,
                y: legacyPayload.position_y ?? current.position_y,
              },
            );
            return {
              folder: placement.folder,
              folder_placements: placement.folder_placements,
              position_x: placement.position_x,
              position_y: placement.position_y,
              position_y_is_explicit: legacyPayload.position_y_is_explicit,
            };
          })()
        : legacyPayload;
      try {
        if (needsRowBackfill) {
          let yearRows: Array<[string, number]> | null = null;
          try {
            const res = await technologiesApi.preview(
              projectId,
              selectedTechLine,
            );
            yearRows = parseYearRowTable(res.script);
          } catch {}
          const plan = computeRowBackfill(
            techsForLine,
            buildYearRowScale(techsForLine),
            yearRows,
          ).filter((r) => r.id !== techDbId);
          if (plan.length > 0) {
            await Promise.all(
              plan.map((r) =>
                updateTech(projectId, r.id, {
                  position_y: r.position_y,
                  position_y_is_explicit: true,
                }),
              ),
            );
          }
        }
        await updateTech(projectId, techDbId, payload, { optimistic: true });
      } catch {}
    },
    [projectId, updateTech, techsForLine, selectedTechLine, technologies],
  );

  const handleAddPrerequisite = useCallback(
    async (techDbId: number, prereqTechId: string) => {
      const ops = buildConnectOps(technologies, techDbId, prereqTechId);
      if (ops.length === 0) return;
      await updateTechBatch(projectId, ops);
    },
    [projectId, technologies, updateTechBatch],
  );

  const handleRemovePrerequisite = useCallback(
    async (techDbId: number, prereqTechId: string) => {
      const ops = buildDisconnectOps(technologies, techDbId, prereqTechId);
      if (ops.length === 0) return;
      await updateTechBatch(projectId, ops);
    },
    [projectId, technologies, updateTechBatch],
  );

  const layoutClass = useMemo(
    () => resolveLayoutClass(selectedTechLine),
    [selectedTechLine],
  );

  const selectedFolder = useMemo(() => {
    const opt = techLineList.find((o) => o.value === selectedTechLine);
    return opt?.folder ?? `${selectedTechLine}_folder`;
  }, [techLineList, selectedTechLine]);

  const techSections = useTechSections();
  const layoutPlan = useMemo(
    () => bfsTreeLayout(techsForLine, selectedFolder, techSections),
    [techsForLine, selectedFolder, techSections],
  );
  const categoryColumns: CategoryColumn[] = layoutPlan.columns;

  return (
    <div className="flex h-full min-w-0 overflow-hidden">
      <div className="flex-1 min-w-0 h-full overflow-hidden">
        <WorkbenchShell
          gameViewHref={gameViewHref(projectId, {
            panel: "countrytechnologyview",
            tag: countryTag,
          })}
          listPanel={
            <div className="flex flex-col h-full min-h-0">
              <TechSurfacePanel
                projectId={projectId}
                technologies={technologies}
                onCreated={async (technology) => {
                  await fetchTechnologies(projectId);
                  setTechLine(technology.tech_line);
                  requestAnimationFrame(() => selectTech(technology.id));
                  await fetchPreview(
                    projectId,
                    technology.tech_line,
                    technology.id,
                  );
                }}
                onRepaired={async () => {
                  await fetchTechnologies(projectId);
                }}
              />

              <div className="flex-1 min-h-0">
                <TechCommandSearch
                  techs={visibleTechnologies}
                  techLines={techLineList}
                  onSelectTech={handleSelectFromSearch}
                />
              </div>
            </div>
          }
          editorPanel={
            <div className="flex flex-col h-full min-h-0">
              <div className="flex items-stretch shrink-0">
                <div className="flex-1 min-w-0">
                  <LedgerTabs
                    techLines={techLineList}
                    techs={visibleTechnologies}
                    activeTechLine={selectedTechLine}
                    onSelectTechLine={handleSwitchLine}
                  />
                </div>

                <div className="flex items-center gap-2 px-3 border-b border-l border-border bg-card shrink-0">
                  <Button
                    size="sm"
                    variant={
                      visibilityMode === "show-all" ? "default" : "outline"
                    }
                    onClick={() =>
                      setVisibilityMode((m) =>
                        m === "reachability"
                          ? "hide-hidden"
                          : m === "hide-hidden"
                            ? "show-all"
                            : "reachability",
                      )
                    }
                    title={
                      visibilityMode === "reachability"
                        ? t("reachabilityTitle", {
                            clause:
                              reachable && reachable.size > 0
                                ? t("reachabilityClauseFound", {
                                    n: reachable.size,
                                  })
                                : t("reachabilityClauseNone"),
                          })
                        : visibilityMode === "hide-hidden"
                          ? t("hideHiddenTitle")
                          : t("showAllTitle")
                    }
                  >
                    {visibilityMode === "reachability" ? (
                      <Unlock className="h-3.5 w-3.5 mr-1" />
                    ) : visibilityMode === "hide-hidden" ? (
                      <EyeOff className="h-3.5 w-3.5 mr-1" />
                    ) : (
                      <Eye className="h-3.5 w-3.5 mr-1" />
                    )}
                    {visibilityMode === "reachability"
                      ? `${t("reachabilityLabel")}${reachable && reachable.size > 0 ? ` (${reachable.size})` : ""}`
                      : visibilityMode === "hide-hidden"
                        ? t("hideHiddenLabel")
                        : t("showAllLabel")}
                  </Button>
                </div>
              </div>

              <div className="flex-1 flex min-h-0 min-w-0">
                <div className="flex-1 min-w-0 min-h-0 relative">
                  {loading && technologies.length === 0 ? (
                    <div className="h-full p-8 grid grid-cols-3 gap-6 content-start opacity-70">
                      {Array.from({ length: 6 }).map((_, i) => (
                        <div
                          key={`skel-${i}`}
                          className="rounded-md border border-border bg-muted/20 p-3 space-y-2"
                        >
                          <div className="flex items-center gap-2">
                            <div className="h-7 w-7 rounded bg-muted/60 animate-pulse" />
                            <div className="h-3 flex-1 rounded bg-muted/50 animate-pulse" />
                          </div>
                          <div className="h-2 w-2/3 rounded bg-muted/40 animate-pulse" />
                          <div className="h-2 w-1/2 rounded bg-muted/40 animate-pulse" />
                        </div>
                      ))}
                    </div>
                  ) : error && technologies.length === 0 ? (
                    <div className="flex h-full items-center justify-center p-8">
                      <WorkbenchLoadError
                        onRetry={() => fetchTechnologies(projectId)}
                      />
                    </div>
                  ) : techsForLine.length === 0 ? (
                    <div className="flex h-full items-center justify-center p-8">
                      <EditorEmptyState
                        variant="hero"
                        className="max-w-md w-full"
                        icon={
                          <FlaskConical className="h-10 w-10 text-muted-foreground/60" />
                        }
                        title={t("emptyHeroTitle")}
                        description={t("emptyHeroDesc")}
                        actions={<></>}
                      />
                    </div>
                  ) : (
                    <TechFlowGraph
                      techs={techsForLine}
                      selectedId={selectedId}
                      projectId={projectId}
                      iconCacheBust={iconCacheBust}
                      availableTechIcons={availableTechIcons}
                      layout={resolveTechLineLayout(selectedTechLine)}
                      layoutClass={layoutClass}
                      categoryColumns={categoryColumns}
                      folder={selectedFolder}
                      reachable={reachable}
                      activeCountryTag={countryTag}
                      selectedPlacementIndex={selectedPlacementIndex}
                      layoutMode={false}
                      surfaceManaged={p0SurfaceEnabled}
                      onSelectTech={selectTech}
                      onUpdatePosition={handleUpdatePosition}
                      onCreateTech={handleCreateFromCanvas}
                      onDeleteTech={handleDelete}
                      onAddPrerequisite={handleAddPrerequisite}
                      onRemovePrerequisite={handleRemovePrerequisite}
                    />
                  )}
                </div>

                {selected && (
                  <div className="w-[380px] ui-ide-panel-left overflow-y-auto shrink-0">
                    <TechEditor
                      tech={selected}
                      allTechs={technologies}
                      onSave={handleSave}
                      projectId={projectId}
                      onDraftChange={setTechDraft}
                      selectedPlacementIndex={selectedPlacementIndex}
                      layoutMode={false}
                      onSelectedPlacementIndexChange={
                        handleSelectedPlacementIndexChange
                      }
                      onIconUploaded={() => {
                        setIconCacheBust(Date.now());

                        void refreshAvailableTechIcons();
                        if (selectedId)
                          fetchPreview(projectId, selectedTechLine, selectedId);
                      }}
                      onDelete={async (techDbId) => {
                        await deleteTech(projectId, techDbId);
                      }}
                    />
                  </div>
                )}
              </div>
            </div>
          }
          previewPanel={
            <div className="flex flex-col h-full">
              <div className="shrink-0 flex items-center border-b border-border text-micro uppercase tracking-widest font-semibold">
                <button
                  type="button"
                  onClick={() => setRightPaneTab("preview")}
                  className={`px-3 py-2 ui-interactive ${
                    rightPaneTab === "preview"
                      ? "text-accent shadow-[inset_0_-2px_0_0] shadow-accent"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {t("livePreviewTab")}
                </button>
                <button
                  type="button"
                  onClick={() => setRightPaneTab("relations")}
                  className={`px-3 py-2 ui-interactive ${
                    rightPaneTab === "relations"
                      ? "text-accent shadow-[inset_0_-2px_0_0] shadow-accent"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {tRel("tab")}
                </button>
              </div>
              {rightPaneTab === "preview" ? (
                <div className="flex-1 min-h-0">
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
                </div>
              ) : (
                <div className="flex-1 min-h-0">
                  <RelationsPanel
                    kind="technology"
                    sourceId={selectedId}
                    projectId={projectId}
                    hideOutgoing
                  />
                </div>
              )}
            </div>
          }
        />
      </div>
    </div>
  );
}
