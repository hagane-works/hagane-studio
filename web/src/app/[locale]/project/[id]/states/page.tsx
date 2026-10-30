"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import {
  ChevronRight,
  Search,
  MapPin,
  Map as MapIcon,
  List as ListIcon,
  Loader2,
} from "lucide-react";
import { useParams, useSearchParams } from "@/platform/navigation";
import { statesApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { useStateStore } from "@/lib/store/stateStore";
import { useShallow } from "zustand/react/shallow";
import { useProjectCountries } from "@/lib/store/projectStore";
import {
  useVanillaCountryTags,
  vanillaTagName,
} from "@/lib/vanillaCountryTags";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";
import { StateEditor } from "@/components/workbench/StateEditor";
import {
  StateBulkBar,
  StateBulkUndoBar,
} from "@/components/workbench/StateBulkBar";
import {
  StateClearSeededBar,
  StateRowDeleteButton,
} from "@/components/workbench/StateDeleteControls";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import {
  StatesMap,
  MapHealthLight,
  ProvinceMoveNotice,
} from "@/components/workbench/states-map";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  buildRestoreRequest,
  hasRestorableStates,
  restoreErrorMessageKey,
  selectRestorableStates,
  summarizeRestoreResult,
  type RestoreVanillaResult,
} from "@/lib/restoreVanillaContent";
import { useVanillaStateIndex } from "@/lib/vanillaStateIndex";
import type { State } from "@/types";

const PER_GROUP_RENDER_CAP = 100;
const UNATTRIBUTED = "__UNATTRIBUTED__";

export default function StatesPage() {
  const params = useParams();
  const projectId = Number(params.id);
  const t = useTranslations("states");
  const locale = useContentLocaleStore((s) => s.locale);
  const { lookup: lookupVanillaTag } = useVanillaCountryTags();
  const projectCountries = useProjectCountries();

  const {
    states,
    selectedId,
    loading,
    error,
    fetchStates,
    selectState,
    updateState,
  } = useStateStore(
    useShallow((s) => ({
      states: s.states,
      selectedId: s.selectedId,
      loading: s.loading,
      error: s.error,
      fetchStates: s.fetchStates,
      selectState: s.selectState,
      updateState: s.updateState,
    })),
  );

  const listNav = useListboxNav();

  const [searchQuery, setSearchQuery] = useState("");
  const [collapsedOwners, setCollapsedOwners] = useState<Set<string>>(
    () => new Set(),
  );
  const [showAllForGroup, setShowAllForGroup] = useState<Set<string>>(
    () => new Set(),
  );
  const [seeding, setSeeding] = useState(false);

  const [adoptingGeoId, setAdoptingGeoId] = useState<number | null>(null);

  const [viewMode, setViewMode] = useState<"list" | "map">("list");

  const [selectedGeoIds, setSelectedGeoIds] = useState<Set<number>>(
    () => new Set(),
  );

  const [lastBulk, setLastBulk] = useState<
    import("@/types").StateBulkResult | null
  >(null);

  const toggleGeoId = useCallback((geoId: number) => {
    setSelectedGeoIds((prev) => {
      const next = new Set(prev);
      if (next.has(geoId)) next.delete(geoId);
      else next.add(geoId);
      return next;
    });
  }, []);

  const clearBulkSelection = useCallback(() => {
    setSelectedGeoIds(() => new Set());
  }, []);

  const mapMultiSelect = useMemo(
    () => ({ selectedGeoIds, onToggleGeoId: toggleGeoId }),
    [selectedGeoIds, toggleGeoId],
  );

  const selectedDbIds = useMemo(() => {
    if (selectedGeoIds.size === 0) return [];
    return states
      .filter((s) => selectedGeoIds.has(s.state_id))
      .map((s) => s.id);
  }, [states, selectedGeoIds]);

  const selectedRows = useMemo(() => {
    if (selectedGeoIds.size === 0) return [];
    return states.filter((s) => selectedGeoIds.has(s.state_id));
  }, [states, selectedGeoIds]);

  const knownTags = useMemo(() => {
    const tags = new Set<string>();
    for (const tag of projectCountries) if (tag) tags.add(tag);

    for (const s of states) if (s.owner) tags.add(s.owner);
    return [...tags].sort();
  }, [projectCountries, states]);

  const afterBulkWrite = useCallback(
    async (result: import("@/types").StateBulkResult) => {
      setLastBulk(result);
      clearBulkSelection();
      await fetchStates(projectId);
    },
    [clearBulkSelection, fetchStates, projectId],
  );

  const guardedSelect = useCallback(
    (id: number) => {
      if (id === selectedId) return;
      if (!confirmDiscardIfDirty()) return;
      selectState(id);
    },
    [selectedId, selectState],
  );

  const selectOneAndClearBulk = useCallback(
    (id: number) => {
      clearBulkSelection();
      guardedSelect(id);
    },
    [clearBulkSelection, guardedSelect],
  );

  const selectByGeoId = useCallback(
    (hoi4StateId: number) => {
      const match = states.find((s) => s.state_id === hoi4StateId);
      if (match) guardedSelect(match.id);
    },
    [states, guardedSelect],
  );

  const adoptGeoId = useCallback(
    async (hoi4StateId: number) => {
      if (adoptingGeoId != null) return;
      if (!confirmDiscardIfDirty()) return;
      setAdoptingGeoId(hoi4StateId);
      try {
        const res = await statesApi.seedFromVanilla(projectId, {
          state_ids: [hoi4StateId],
        });
        await fetchStates(projectId);

        const fresh = useStateStore
          .getState()
          .states.find((s) => s.state_id === hoi4StateId);
        if (fresh) {
          selectState(fresh.id);
          return;
        }
        toast.error(
          res.missing.includes(hoi4StateId)
            ? t("mapAdoptNotInBase", { id: hoi4StateId })
            : t("mapAdoptNoRow", { id: hoi4StateId }),
        );
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        toast.error(t("mapAdoptFailed", { id: hoi4StateId, msg }));
      } finally {
        setAdoptingGeoId(null);
      }
    },
    [projectId, adoptingGeoId, fetchStates, selectState, t],
  );

  const handleSeedFromVanilla = useCallback(async () => {
    setSeeding(true);
    try {
      const res = await statesApi.seedFromVanilla(projectId, { all: true });
      await fetchStates(projectId);
      toast.success(
        t("seedDone", { seeded: res.seeded, skipped: res.skipped }),
      );
      if (res.warnings.length > 0) {
        toast.warning(t("seedWarnings", { count: res.warnings.length }));
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      toast.error(t("seedFailed", { msg }));
    } finally {
      setSeeding(false);
    }
  }, [projectId, fetchStates, t]);

  useSWR(projectId ? ["states/list", projectId] : null, () =>
    fetchStates(projectId),
  );

  const searchNeedle = searchQuery.trim().toLowerCase();

  const filteredStates = useMemo(() => {
    if (!searchNeedle) return states;

    const matchesAnyLanguage = (map: Record<string, string> | undefined) =>
      Object.values(map ?? {}).some((text) =>
        (text ?? "").toLowerCase().includes(searchNeedle),
      );
    return states.filter(
      (s) =>
        String(s.state_id).includes(searchNeedle) ||
        (s.name ?? "").toLowerCase().includes(searchNeedle) ||
        (s.owner ?? "").toLowerCase().includes(searchNeedle) ||
        matchesAnyLanguage(s.display_name) ||
        matchesAnyLanguage(s.display_name_locs),
    );
  }, [states, searchNeedle]);

  const grouped = useMemo(() => {
    const groups = new Map<string, State[]>();
    for (const s of filteredStates) {
      const key = s.owner && s.owner.length > 0 ? s.owner : UNATTRIBUTED;
      const arr = groups.get(key);
      if (arr) arr.push(s);
      else groups.set(key, [s]);
    }
    return Array.from(groups.entries()).sort(([a], [b]) => {
      if (a === UNATTRIBUTED) return 1;
      if (b === UNATTRIBUTED) return -1;
      return a.localeCompare(b);
    });
  }, [filteredStates]);

  const toggleOwner = useCallback((tag: string) => {
    setCollapsedOwners((prev) => {
      const next = new Set(prev);
      if (next.has(tag)) next.delete(tag);
      else next.add(tag);
      return next;
    });
  }, []);

  const showAllForGroupHandler = useCallback((tag: string) => {
    setShowAllForGroup((prev) => {
      if (prev.has(tag)) return prev;
      const next = new Set(prev);
      next.add(tag);
      return next;
    });
  }, []);

  const selected = states.find((s) => s.id === selectedId) || null;

  const searchParams = useSearchParams();
  const didRestoreSelection = useRef(false);
  useEffect(() => {
    if (didRestoreSelection.current || states.length === 0) return;
    const wanted = Number(searchParams?.get("state"));
    const row = wanted ? states.find((s) => s.state_id === wanted) : undefined;
    if (row) selectState(row.id);
    didRestoreSelection.current = true;
  }, [states, searchParams, selectState]);
  const selectedDisplay = selected
    ? resolveDisplayName(
        locale,
        selected.display_name ?? selected.display_name_locs,
        null,
        null,
        "",
      )
    : "";

  const handleSave = useCallback(
    (id: number, data: Parameters<typeof updateState>[2]) =>
      updateState(projectId, id, data),
    [projectId, updateState],
  );

  const zeroStatesHero = (
    <EditorEmptyState
      variant="hero"
      icon={<MapPin className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyHeroTitle")}
      description={t("emptyHeroDesc")}
      actions={
        <Button
          size="sm"
          onClick={() => {
            handleSeedFromVanilla();
          }}
          disabled={seeding}
          className="bg-accent text-accent-foreground hover:bg-accent/90 ui-interactive"
        >
          {seeding ? t("seeding") : t("seedFromVanilla")}
        </Button>
      }
    />
  );

  const editorPanelNode = selected ? (
    <StateEditor
      state={selected}
      countryTags={projectCountries}
      onSave={handleSave}
    />
  ) : error && states.length === 0 ? (
    <div className="flex h-full items-center justify-center p-6">
      <WorkbenchLoadError onRetry={() => fetchStates(projectId)} />
    </div>
  ) : states.length === 0 ? (
    <div className="flex items-center justify-center h-full p-6">
      {zeroStatesHero}
    </div>
  ) : (
    <div className="flex flex-col items-center justify-center h-full gap-4 text-center p-8">
      <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center">
        <MapPin className="h-6 w-6 text-muted-foreground" />
      </div>
      <div>
        <p className="text-sm font-medium text-foreground">
          {t("selectPrompt")}
        </p>
        <p className="text-xs text-muted-foreground mt-1">
          {viewMode === "map" ? t("selectHintMap") : t("selectHint")}
        </p>
      </div>
    </div>
  );

  const viewToggle = (
    <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border/40 shrink-0">
      <div className="inline-flex rounded-md border border-border overflow-hidden">
        <button
          type="button"
          onClick={() => setViewMode("list")}
          aria-pressed={viewMode === "list"}
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 text-xs ui-interactive ${
            viewMode === "list"
              ? "bg-accent text-accent-foreground"
              : "bg-background text-muted-foreground hover:bg-muted"
          }`}
        >
          <ListIcon className="h-3.5 w-3.5" aria-hidden />
          {t("viewList")}
        </button>
        <button
          type="button"
          onClick={() => setViewMode("map")}
          aria-pressed={viewMode === "map"}
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 text-xs border-l border-border ui-interactive ${
            viewMode === "map"
              ? "bg-accent text-accent-foreground"
              : "bg-background text-muted-foreground hover:bg-muted"
          }`}
        >
          <MapIcon className="h-3.5 w-3.5" aria-hidden />
          {t("viewMap")}
        </button>
      </div>

      <div className="ml-auto flex items-center gap-2">
        <StateClearSeededBar
          projectId={projectId}
          states={states}
          onCleared={() => fetchStates(projectId)}
        />
        <RestoreVanillaBar
          projectId={projectId}
          states={states}
          onRestored={() => fetchStates(projectId)}
        />
      </div>
    </div>
  );

  if (viewMode === "map" && error && states.length === 0) {
    return (
      <div className="flex flex-col h-full min-h-0">
        {viewToggle}
        <div className="flex flex-1 items-center justify-center p-6">
          <WorkbenchLoadError onRetry={() => fetchStates(projectId)} />
        </div>
      </div>
    );
  }

  if (viewMode === "map") {
    return (
      <div className="flex flex-col h-full min-h-0">
        {viewToggle}

        <ProvinceMoveNotice projectId={projectId} />
        {lastBulk && (
          <StateBulkUndoBar
            projectId={projectId}
            result={lastBulk}
            onUndone={async () => {
              setLastBulk(null);
              await fetchStates(projectId);
            }}
            onDismiss={() => setLastBulk(null)}
          />
        )}
        <div className="flex flex-1 min-h-0">
          <div className="relative flex flex-1 min-w-0 min-h-0">
            <StatesMap
              projectId={projectId}
              states={states}
              selectedId={selectedId}
              onSelectState={selectOneAndClearBulk}
              onAdoptGeoId={adoptGeoId}
              adoptingGeoId={adoptingGeoId}

              multiSelect={mapMultiSelect}
            />
            <div className="absolute z-20 top-2 right-2">
              <MapHealthLight
                projectId={projectId}
                onSelectStateGeoId={selectByGeoId}
              />
            </div>
          </div>

          <div className="w-96 border-l border-border shrink-0 min-h-0 bg-card flex flex-col">
            <ScrollArea className="flex-1 min-h-0">
              {editorPanelNode}
            </ScrollArea>
          </div>
        </div>

        {selectedGeoIds.size >= 2 && (
          <StateBulkBar
            projectId={projectId}
            selectedIds={selectedDbIds}
            selectedGeoCount={selectedGeoIds.size}
            knownTags={knownTags}
            selectedRows={selectedRows}
            onClear={clearBulkSelection}
            onApplied={afterBulkWrite}
          />
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      {viewToggle}
      <ProvinceMoveNotice projectId={projectId} />
      <div className="flex-1 min-h-0">
        <WorkbenchShell
          gameViewHref={gameViewHref(projectId, {
            panel: "countrystateview",
            state: selected?.state_id,
            tag: selected?.owner,
          })}
          listPanel={
            <div>
              <WorkbenchListHeader
                title={t("title")}
                count={filteredStates.length}
                totalCount={states.length}
                searchValue={searchQuery}
                onSearchChange={setSearchQuery}
                searchPlaceholder={t("searchPlaceholder")}
                renderSearch={states.length > 0}
              />

              <div className="px-3 pb-3">
                {loading && states.length === 0 ? (
                  <div className="space-y-1.5">
                    {Array.from({ length: 8 }).map((_, i) => (
                      <div
                        key={`skel-${i}`}
                        className="h-8 rounded-md bg-muted/50 animate-pulse"
                      />
                    ))}
                  </div>
                ) : states.length === 0 ? (
                  <p className="p-4 text-center text-xs text-muted-foreground">
                    {t("emptyNoStatesTitle")}
                  </p>
                ) : filteredStates.length === 0 ? (
                  <EditorEmptyState
                    icon={
                      <Search className="h-6 w-6 text-muted-foreground/60" />
                    }
                    title={t("noMatches")}
                  />
                ) : (
                  <div
                    {...listNav.containerProps}
                    aria-label={t("title")}
                    className="space-y-1 outline-none"
                  >
                    {grouped.map(([ownerTag, items], idx) => {
                      if (items.length === 0) return null;
                      const defaultExpanded = idx === 0;
                      const forceExpand = searchNeedle.length > 0;
                      const toggledByUser = collapsedOwners.has(ownerTag);
                      const isExpanded = forceExpand
                        ? true
                        : toggledByUser
                          ? !defaultExpanded
                          : defaultExpanded;
                      const isUnattributed = ownerTag === UNATTRIBUTED;
                      const vanilla = isUnattributed
                        ? null
                        : lookupVanillaTag(ownerTag);
                      const groupLabel = isUnattributed
                        ? t("unattributed")
                        : vanilla
                          ? vanillaTagName(vanilla, locale)
                          : ownerTag;
                      const total = items.length;
                      const cap = showAllForGroup.has(ownerTag)
                        ? total
                        : PER_GROUP_RENDER_CAP;
                      const visible = total > cap ? items.slice(0, cap) : items;
                      const hidden = total - visible.length;

                      return (
                        <div key={`g-${ownerTag}`} className="space-y-0.5">
                          <button
                            type="button"
                            onClick={() => toggleOwner(ownerTag)}
                            aria-expanded={isExpanded}
                            className="ui-interactive flex items-center gap-1.5 w-full px-2 py-1 rounded text-xs text-muted-foreground hover:bg-muted/60"
                            title={
                              isUnattributed
                                ? groupLabel
                                : `${ownerTag} · ${groupLabel}`
                            }
                          >
                            <ChevronRight
                              className={`h-3 w-3 transition-transform shrink-0 ${isExpanded ? "rotate-90" : ""}`}
                            />
                            {!isUnattributed && (
                              <span
                                aria-hidden
                                className="text-xs leading-none"
                              >
                                🏳
                              </span>
                            )}
                            <span className="font-mono font-semibold tracking-tight">
                              {isUnattributed ? "—" : ownerTag}
                            </span>
                            <span className="truncate opacity-70">
                              {groupLabel}
                            </span>
                            <Badge
                              variant="outline"
                              className="ml-auto h-4 px-1.5 text-micro shrink-0"
                            >
                              {total}
                            </Badge>
                          </button>
                          {isExpanded && (
                            <div className="pl-3 space-y-0.5 border-l border-border ml-2">
                              {visible.map((st) => {
                                const display = resolveDisplayName(
                                  locale,
                                  st.display_name ?? st.display_name_locs,
                                  null,
                                  null,
                                  "",
                                );
                                return (
                                  <div
                                    key={st.id}
                                    {...listNav.optionProps(
                                      selectedId === st.id,
                                    )}
                                    onClick={() => guardedSelect(st.id)}
                                    className={`group flex items-center justify-between px-3 py-1.5 rounded-md cursor-pointer text-sm transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
                                      selectedId === st.id
                                        ? "bg-primary/10 text-primary"
                                        : "hover:bg-muted"
                                    }`}
                                  >
                                    <span className="font-mono text-xs shrink-0 w-12 text-muted-foreground">
                                      {st.state_id}
                                    </span>
                                    <span className="truncate flex-1 min-w-0 ml-1">
                                      {display ||
                                        st.name ||
                                        `STATE_${st.state_id}`}
                                    </span>
                                    {st.membership_provenance?.status ===
                                      "known" &&
                                      st.membership_provenance
                                        .membership_changed === true && (
                                        <span
                                          className="ml-1.5 shrink-0 text-warning"
                                          aria-label={t(
                                            "membershipFlagChanged",
                                          )}
                                          title={t("membershipFlagChanged")}
                                        >
                                          ●
                                        </span>
                                      )}
                                    {display && (
                                      <span
                                        className="font-mono text-micro text-muted-foreground shrink-0 ml-1.5 max-w-[7rem] truncate"
                                        title={st.name}
                                      >
                                        {st.name}
                                      </span>
                                    )}

                                    <StateRowDeleteButton
                                      projectId={projectId}
                                      state={st}
                                      onDeleted={() => fetchStates(projectId)}
                                    />
                                  </div>
                                );
                              })}
                              {hidden > 0 && (
                                <button
                                  type="button"
                                  onClick={() =>
                                    showAllForGroupHandler(ownerTag)
                                  }
                                  className="ui-interactive w-full px-2 py-1 text-xs text-muted-foreground hover:text-foreground hover:bg-muted/40 rounded text-left"
                                >
                                  {t("showMore", { count: hidden })}
                                </button>
                              )}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          }
          editorPanel={editorPanelNode}
          previewPanel={
            selected ? (
              <div className="p-3 space-y-3 text-xs">
                <h4 className="text-micro uppercase tracking-widest font-semibold text-muted-foreground">
                  {t("summaryTitle")}
                </h4>
                <SummaryRow
                  label={t("stateId")}
                  value={String(selected.state_id)}
                />
                <SummaryRow
                  label={t("displayName")}
                  value={selectedDisplay || "—"}
                />
                <SummaryRow label={t("owner")} value={selected.owner || "—"} />
                <SummaryRow
                  label={t("controller")}
                  value={selected.controller || t("controllerSameAsOwner")}
                />
                <SummaryRow
                  label={t("manpower")}
                  value={selected.manpower.toLocaleString()}
                />
                <SummaryRow
                  label={t("provinceCount")}
                  value={String(selected.provinces.length)}
                />
                <SummaryRow
                  label={t("membershipFlag")}
                  value={
                    selected.membership_provenance?.status !== "known" ||
                    selected.membership_provenance.membership_changed == null
                      ? t("membershipFlagUnknown")
                      : selected.membership_provenance.membership_changed
                        ? t("membershipFlagChanged")
                        : t("membershipFlagUnchanged")
                  }
                />
                <SummaryRow
                  label={t("cores")}
                  value={
                    selected.add_core_of.length
                      ? selected.add_core_of.join(", ")
                      : "—"
                  }
                />
                <p className="pt-2 text-micro text-muted-foreground border-t border-border/40">
                  {t("previewPreserveNote")}
                </p>
              </div>
            ) : (
              <div className="p-3 text-xs text-muted-foreground">
                {t("previewEmpty")}
              </div>
            )
          }
        />
      </div>
    </div>
  );
}

function RestoreVanillaBar({
  projectId,
  states,
  onRestored,
}: {
  projectId: number;
  states: State[];
  onRestored: () => Promise<void> | void;
}) {
  const t = useTranslations("restoreVanillaContent");
  const [busy, setBusy] = useState(false);

  const [preview, setPreview] = useState<RestoreVanillaResult | null>(null);

  const vanillaStates = useVanillaStateIndex();
  const selection = useMemo(
    () => selectRestorableStates(states, vanillaStates),
    [states, vanillaStates],
  );

  const openPreview = useCallback(async () => {
    setBusy(true);
    try {
      const res = await statesApi.restoreVanillaContent(
        projectId,
        buildRestoreRequest(selection, { dryRun: true }),
      );

      if (summarizeRestoreResult(res).restored === 0) {
        toast.info(t("nothingToRestore"));
        return;
      }
      setPreview(res);
    } catch (e) {
      toast.error(t(restoreErrorMessageKey(e)));
    } finally {
      setBusy(false);
    }
  }, [projectId, selection, t]);

  const apply = useCallback(async () => {
    setBusy(true);
    try {
      const done = await statesApi.restoreVanillaContent(
        projectId,
        buildRestoreRequest(selection),
      );
      const s = summarizeRestoreResult(done);
      setPreview(null);
      toast.success(t("successTitle"), {
        description: t("successBody", {
          count: s.restored,
          navalBases: s.navalBases,
          landmarks: s.landmarks,
        }),
      });
      await onRestored();
    } catch (e) {
      setPreview(null);
      toast.error(t(restoreErrorMessageKey(e)));
    } finally {
      setBusy(false);
    }
  }, [projectId, selection, t, onRestored]);

  if (!hasRestorableStates(selection)) return null;

  const s = summarizeRestoreResult(preview);

  return (
    <>
      <Button
        size="sm"
        variant="outline"
        disabled={busy}
        onClick={openPreview}
        className="ml-auto ui-interactive"
      >
        {busy && <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />}
        {busy
          ? t("buttonBusy")
          : t("button", { count: selection.stateDbIds.length })}
      </Button>

      <Dialog
        open={preview != null}
        onOpenChange={(o) => !o && setPreview(null)}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("dialogTitle")}</DialogTitle>
          </DialogHeader>

          <div className="space-y-3 text-sm">
            <p>{t("dialogIntro", { count: s.restored })}</p>

            {s.hasContent && (
              <div className="rounded-md border border-border bg-muted/40 p-2">
                <p className="text-xs font-semibold">
                  {t("willRestoreHeading")}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {t("willRestoreLine", {
                    navalBases: s.navalBases,
                    bunkers: s.bunkers,
                    dams: s.dams,
                    landmarks: s.landmarks,
                    datedBlocks: s.datedBlocks,
                    hasDlc: s.hasDlc,
                  })}
                </p>
              </div>
            )}

            <div className="rounded-md border border-border p-2">
              <p className="text-xs font-semibold">
                {t("willNotTouchHeading")}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("willNotTouchLine")}
              </p>
            </div>

            {s.skipped > 0 && (
              <p className="text-xs text-muted-foreground">
                {t("skippedNote", { count: s.skipped })}
              </p>
            )}
            {s.missing > 0 && (
              <p className="text-xs text-muted-foreground">
                {t("missingNote", { count: s.missing })}
              </p>
            )}
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={busy}
              onClick={() => setPreview(null)}
            >
              {t("cancel")}
            </Button>
            <Button
              size="sm"
              disabled={busy || s.restored === 0}
              onClick={apply}
              className="ui-interactive"
            >
              {busy && (
                <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
              )}

              {t("confirm", { count: s.restored })}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline gap-2">
      <span className="text-muted-foreground w-24 shrink-0">{label}</span>
      <span className="font-mono break-all">{value}</span>
    </div>
  );
}
