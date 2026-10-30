"use client";

import { useCallback, useEffect, useRef, useState, useMemo } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { useParams, useSearchParams } from "@/platform/navigation";
import {
  ChevronRight,
  FolderOpen,
  Search,
  FileX,
  Eye,
  BookOpen,
  Link2,
  FileCode2,
} from "lucide-react";
import { TreeValidationButton } from "@/components/workbench/TreeValidationDialog";
import { useEventStore } from "@/lib/store/eventStore";
import { useShallow } from "zustand/react/shallow";
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
  parseRunCheckEntityDeepLink,
  resolveDeepLinkScopeReveal,
} from "@/lib/runCheckDeepLink";
import {
  ScopeBadge,
  ScopeSectionHeader,
} from "@/components/workbench/ScopeBadge";
import { CountryTagChip } from "@/components/workbench/CountryTagChip";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { EventEditor } from "@/components/workbench/EventEditor";
import { ScriptEditor } from "@/shared/editors/ScriptEditor";
import { useEditorWiring } from "@/shared/editors/useEditorWiring";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { eventsApi, type VanillaEventEntry } from "@/lib/api";
import { useDeadFlagLint } from "@/shared/useDeadFlagLint";
import { DeadFlagBanner } from "@/components/workbench/DeadFlagBanner";
import type { GameEvent } from "@/types";
import { FocusIconLibraryDialog } from "@/components/workbench/focus-graph/FocusIconLibraryDialog";
import { useBatchGfxApply } from "@/lib/hooks/useBatchGfxApply";
import {
  EventInGamePreview,
  type EventPreviewTarget,
} from "@/components/workbench/EventInGamePreview";
import { EventPreviewBoundary } from "@/components/workbench/EventPreviewBoundary";
import { EventScopeBanner } from "@/components/workbench/EventScopeBanner";
import { VanillaEventsTab } from "@/components/workbench/VanillaEventsTab";
import { RelationsPanel } from "@/components/workbench/RelationsPanel";
import { WorkbenchRightPane } from "@/components/workbench/WorkbenchRightPane";

export default function EventsPage() {
  const t = useTranslations("events");
  const tRel = useTranslations("relations");
  const tShell = useTranslations("workbenchShell");
  const tVanilla = useTranslations("vanillaBrowser");
  const tCommon = useTranslations("common");
  const params = useParams();
  const searchParams = useSearchParams();
  const projectId = Number(params.id);
  const runCheckTarget = useMemo(
    () => parseRunCheckEntityDeepLink(searchParams, "namespace", "event"),
    [searchParams],
  );
  const runCheckTargetKey = runCheckTarget
    ? `${runCheckTarget.containerId}:${runCheckTarget.entityId}`
    : null;
  const handledRunCheckTargetRef = useRef<string | null>(null);
  const runCheckRequestSequenceRef = useRef(0);
  const countryTag = useActiveCountry();

  const projectCountries = useProjectCountries();
  const hasCountries = projectCountries.length > 0;
  const [addCountryOpen, setAddCountryOpen] = useState(false);

  const nsInputRef = useRef<HTMLInputElement>(null);

  const filterCountry = useEffectiveCountryFilter();
  const viewMode = useProjectStore((s) => s.viewMode);

  const { report: deadFlagReport } = useDeadFlagLint(projectId || null);

  const nsListNav = useListboxNav();
  const eventListNav = useListboxNav();

  const confirmDialog = useConfirm();

  const {
    namespaces,
    events,
    selectedNsId,
    selectedEventId,
    scriptPreview,
    loading,
    error,
    fetchNamespaces,
    selectNamespace,
    createNamespace,
    deleteNamespace,
    fetchEvents,
    selectEvent,
    createEvent,
    updateEvent,
    deleteEvent,
    fetchPreview,
  } = useEventStore(
    useShallow((s) => ({
      namespaces: s.namespaces,
      events: s.events,
      selectedNsId: s.selectedNsId,
      selectedEventId: s.selectedEventId,
      scriptPreview: s.scriptPreview,
      loading: s.loading,
      error: s.error,
      fetchNamespaces: s.fetchNamespaces,
      selectNamespace: s.selectNamespace,
      createNamespace: s.createNamespace,
      deleteNamespace: s.deleteNamespace,
      fetchEvents: s.fetchEvents,
      selectEvent: s.selectEvent,
      createEvent: s.createEvent,
      updateEvent: s.updateEvent,
      deleteEvent: s.deleteEvent,
      fetchPreview: s.fetchPreview,
    })),
  );

  const [newNsName, setNewNsName] = useState("");

  const [isCreatingEvent, setIsCreatingEvent] = useState(false);
  const [eventSearch, setEventSearch] = useState("");

  const [showOnlyOwn, setShowOnlyOwn] = useState(false);

  const userToggledScopeRef = useRef(false);

  const [smartHiddenSharedCount, setSmartHiddenSharedCount] = useState<
    number | null
  >(null);

  const [rightPaneTabId, setRightPaneTabId] = useState("preview");
  const [eventDraft, setEventDraft] = useState<GameEvent | null>(null);
  const [previewTarget, setPreviewTarget] = useState<EventPreviewTarget | null>(
    null,
  );

  const [previewOptionIndex, setPreviewOptionIndex] = useState<number | null>(
    null,
  );

  const handleSelectPreviewTarget = useCallback(
    (target: EventPreviewTarget, subIndex?: number) => {
      setPreviewTarget(target);
      setPreviewOptionIndex(
        target === "options" && typeof subIndex === "number" ? subIndex : null,
      );
    },
    [],
  );

  const [collapsedPrefixes, setCollapsedPrefixes] = useState<Set<string>>(
    new Set(),
  );
  const togglePrefix = useCallback((prefix: string) => {
    setCollapsedPrefixes((prev) => {
      const next = new Set(prev);
      if (next.has(prefix)) next.delete(prefix);
      else next.add(prefix);
      return next;
    });
  }, []);

  const filteredNamespaces = useMemo(() => {
    if (!filterCountry) return namespaces;
    return namespaces.filter((n) => matchesCountryScope(n, filterCountry));
  }, [namespaces, filterCountry]);

  const filteredEvents = useMemo(() => {
    const byCountry = filterCountry
      ? events.filter((e) => matchesCountryScope(e, filterCountry))
      : events;

    const byScope = showOnlyOwn
      ? byCountry.filter((e) => (e.scope_kind ?? "own") === "own")
      : byCountry;
    const q = eventSearch.toLowerCase().trim();
    if (!q) return byScope;
    return byScope.filter(
      (e) =>
        e.event_id.toLowerCase().includes(q) ||
        (e.title_en || "").toLowerCase().includes(q) ||
        (e.title_zh || "").toLowerCase().includes(q),
    );
  }, [events, eventSearch, filterCountry, showOnlyOwn]);

  const scopeCounts = useMemo(() => {
    let own = 0;
    let shared = 0;
    for (const ns of namespaces) {
      const kind = ns.scope_kind ?? "own";
      const c = ns.event_count ?? 0;

      if (kind === "own" && (!ns.country_tag || ns.country_tag === countryTag))
        own += c;
      else shared += c;
    }
    return { own, shared };
  }, [namespaces, countryTag]);

  const SMART_DEFAULT_SHARED_THRESHOLD = 500;
  useEffect(() => {
    if (userToggledScopeRef.current) return;
    if (viewMode !== "single") return;
    if (scopeCounts.shared <= SMART_DEFAULT_SHARED_THRESHOLD) return;
    if (showOnlyOwn) return;
    setShowOnlyOwn(true);
    setSmartHiddenSharedCount(scopeCounts.shared);
  }, [viewMode, scopeCounts.shared, showOnlyOwn]);

  useSWR(projectId ? ["events/namespaces", projectId] : null, () =>
    fetchNamespaces(projectId),
  );

  useEffect(() => {
    if (!runCheckTarget || !runCheckTargetKey) {
      handledRunCheckTargetRef.current = null;
      return;
    }
    if (handledRunCheckTargetRef.current === runCheckTargetKey) return;

    const targetNamespace = namespaces.find(
      (namespace) => namespace.id === runCheckTarget.containerId,
    );
    if (!targetNamespace) return;

    const requestSequence = ++runCheckRequestSequenceRef.current;
    void (async () => {
      try {
        const targetEvent = await eventsApi.get(
          projectId,
          runCheckTarget.entityId,
        );
        if (runCheckRequestSequenceRef.current !== requestSequence) return;
        if (targetEvent.namespace_id !== targetNamespace.id) return;

        const projectState = useProjectStore.getState();
        const currentProject = projectState.currentProject;
        const projectCountryTags = currentProject?.country_tags?.length
          ? currentProject.country_tags
          : currentProject?.country_tag
            ? [currentProject.country_tag]
            : [];
        const effectiveCountry =
          projectState.viewMode === "all"
            ? null
            : (projectState.activeCountryTag ??
              currentProject?.country_tag ??
              null);
        const reveal = resolveDeepLinkScopeReveal(
          [targetNamespace, targetEvent],
          effectiveCountry,
          projectCountryTags,
        );
        if (reveal.kind === "country") {
          projectState.setActiveCountryTag(reveal.countryTag);
        } else if (reveal.kind === "all") {
          projectState.setViewMode("all");
        }

        setShowOnlyOwn(false);
        selectNamespace(targetNamespace.id);
        await fetchEvents(projectId, targetNamespace.id);
        if (runCheckRequestSequenceRef.current !== requestSequence) return;

        selectEvent(targetEvent.id);
        handledRunCheckTargetRef.current = runCheckTargetKey;
      } catch {}
    })();

    return () => {
      if (runCheckRequestSequenceRef.current === requestSequence) {
        runCheckRequestSequenceRef.current += 1;
      }
    };
  }, [
    fetchEvents,
    namespaces,
    projectId,
    runCheckTarget,
    runCheckTargetKey,
    selectEvent,
    selectNamespace,
  ]);

  useSWR(
    projectId && selectedNsId ? ["events/list", projectId, selectedNsId] : null,
    () => fetchEvents(projectId, selectedNsId!),
  );
  useSWR(
    projectId && selectedEventId
      ? ["events/preview", projectId, selectedEventId]
      : null,
    () => fetchPreview(projectId, selectedEventId!),
  );

  const parsePreviewCall = useCallback(
    (script: string) =>
      eventsApi.parsePreview(projectId, selectedEventId!, script),
    [projectId, selectedEventId],
  );
  const wiring = useEditorWiring({
    workbench: "events",
    entityId: selectedEventId,
    scriptPreview,
    parsePreview: parsePreviewCall,
    applyEntity: (id, entityDict) =>
      updateEvent(projectId, id, entityDict as Partial<GameEvent>),
    onAfterApply: async (id) => {
      if (selectedNsId) await fetchEvents(projectId, selectedNsId);
      await fetchPreview(projectId, id);
    },
    entityNoun: "event",
    projectId,
  });

  useEffect(() => {
    if (!filterCountry) return;
    if (selectedNsId) {
      const sel = namespaces.find((n) => n.id === selectedNsId);
      if (sel && !matchesCountryScope(sel, filterCountry)) {
        selectNamespace(null);
      }
    }
    if (selectedEventId) {
      const sel = events.find((e) => e.id === selectedEventId);
      if (sel && !matchesCountryScope(sel, filterCountry)) {
        selectEvent(null);
      }
    }
  }, [
    filterCountry,
    selectedNsId,
    selectedEventId,
    namespaces,
    events,
    selectNamespace,
    selectEvent,
  ]);

  const selected = events.find((e) => e.id === selectedEventId) || null;
  const previewEvent =
    selected && eventDraft?.id === selected.id ? eventDraft : selected;
  const previewIsDraft = useMemo(() => {
    if (!selected || !eventDraft || eventDraft.id !== selected.id) return false;
    return JSON.stringify(eventDraft) !== JSON.stringify(selected);
  }, [eventDraft, selected]);
  const handlePreviewTargetConsumed = useCallback(() => {
    setPreviewTarget(null);
    setPreviewOptionIndex(null);
  }, []);

  useEffect(() => {
    setEventDraft(null);
    setPreviewTarget(null);
    setPreviewOptionIndex(null);
  }, [selectedEventId]);

  const handleCreateNamespace = async () => {
    const name = newNsName.trim();
    if (!name) return;
    await createNamespace(projectId, name, countryTag);
    setNewNsName("");
    fetchEvents(projectId);
  };

  const handleDeleteNamespace = async (nsId: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (
      await confirmDialog({ body: t("confirmDeleteNamespace"), danger: true })
    ) {
      await deleteNamespace(projectId, nsId);
    }
  };

  const handleCreateEvent = async () => {
    if (!selectedNsId) return;
    if (isCreatingEvent) return;
    const ns = namespaces.find((n) => n.id === selectedNsId);
    if (!ns) return;

    const maxNum = events.reduce(
      (max, ev) => Math.max(max, ev.event_number),
      0,
    );
    const nextNum = maxNum + 1;

    setIsCreatingEvent(true);
    try {
      await createEvent(projectId, {
        namespace_id: selectedNsId,
        event_id: `${ns.namespace}.${nextNum}`,
        event_number: nextNum,
        event_type: "country_event",
        is_triggered_only: true,
        country_tag: countryTag,
        options: [
          {
            name: `${ns.namespace}.${nextNum}.a`,
            trigger: "",
            effects: "",
            hidden_effects: "",
            ai_chance: { base: 1 },
          },
        ],
      });

      await fetchEvents(projectId, selectedNsId);
    } finally {
      setIsCreatingEvent(false);
    }
  };

  const handleUseAsTemplate = useCallback(
    async (entry: VanillaEventEntry) => {
      if (!selectedNsId) return;
      const ns = namespaces.find((n) => n.id === selectedNsId);
      if (!ns) return;
      const maxNum = events.reduce(
        (max, ev) => Math.max(max, ev.event_number),
        0,
      );
      const nextNum = maxNum + 1;
      const optCount = Math.max(1, entry.option_count || 1);
      await createEvent(projectId, {
        namespace_id: selectedNsId,
        event_id: `${ns.namespace}.${nextNum}`,
        event_number: nextNum,
        event_type: (entry.type as GameEvent["event_type"]) || "country_event",
        is_triggered_only: entry.is_triggered_only,
        fire_only_once: entry.fire_only_once,
        hidden: entry.hidden,
        major: entry.major,
        picture: entry.picture || "",
        country_tag: countryTag,
        options: Array.from({ length: optCount }, (_, i) => ({
          name: `${ns.namespace}.${nextNum}.${String.fromCharCode(97 + i)}`,
          trigger: "",
          effects: "",
          hidden_effects: "",
          ai_chance: { base: 1 },
        })),
      });
      await fetchEvents(projectId, selectedNsId);

      setRightPaneTabId("preview");
    },
    [
      projectId,
      selectedNsId,
      namespaces,
      events,
      countryTag,
      createEvent,
      fetchEvents,
    ],
  );

  const handleSave = async (data: Partial<GameEvent>) => {
    if (!selectedEventId) return;
    await updateEvent(projectId, selectedEventId, data);
  };

  const handleDeleteEvent = async (id: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (await confirmDialog({ body: t("confirmDeleteEvent"), danger: true })) {
      await deleteEvent(projectId, id);
    }
  };

  const batchGfx = useBatchGfxApply({
    applyOne: async (eventId, gfxName) => {
      await updateEvent(projectId, eventId, { picture: gfxName });
    },
    onComplete: () => {
      if (selectedNsId) fetchEvents(projectId, selectedNsId);
    },
  });

  return (
    <div className="flex h-full min-w-0 overflow-hidden">
      <div className="flex-1 min-w-0 h-full overflow-hidden">
        <WorkbenchShell
          gameViewHref={gameViewHref(
            projectId,
            selected
              ? {
                  panel: "EventWindow",
                  event: selected.event_id,
                  tag: countryTag,
                }
              : { panel: "EventWindow", tag: countryTag },
          )}
          previewWidth="w-96"
          listPanel={
            <div className="p-3">
              {(scopeCounts.own > 0 || scopeCounts.shared > 0) && (
                <div className="mb-3">
                  <EventScopeBanner
                    ownCount={scopeCounts.own}
                    sharedCount={scopeCounts.shared}
                    projectId={projectId}
                    countryTag={countryTag}
                    hiddenSharedCount={
                      showOnlyOwn ? smartHiddenSharedCount : null
                    }
                    onOpenVanillaTab={() => setRightPaneTabId("vanilla")}
                  />
                </div>
              )}

              {deadFlagReport && deadFlagReport.dead_flags.length > 0 && (
                <div className="mb-3">
                  <DeadFlagBanner report={deadFlagReport} />
                </div>
              )}

              <div className="mb-3">
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-sm font-semibold">
                    {t("namespacesHeading")}
                  </h3>
                  <div className="flex items-center gap-1">
                    <TreeValidationButton projectId={projectId} kind="event" />
                  </div>
                </div>

                <div className="flex gap-1 mb-2">
                  <Input
                    ref={nsInputRef}
                    value={newNsName}
                    onChange={(e) => setNewNsName(e.target.value)}
                    placeholder={t("newNsPlaceholder")}
                    className="text-xs h-8"
                    onKeyDown={(e) =>
                      e.key === "Enter" && handleCreateNamespace()
                    }
                  />
                  <Button
                    size="sm"
                    className="px-2"
                    onClick={handleCreateNamespace}
                    disabled={!newNsName.trim()}
                    title={
                      newNsName.trim()
                        ? t("createNamespace")
                        : t("enterNamespaceNameFirst")
                    }
                  >
                    +
                  </Button>
                </div>

                <div
                  {...nsListNav.containerProps}
                  aria-label={t("namespacesHeading")}
                  className="space-y-0.5 mb-3 outline-none"
                >
                  {(() => {
                    const { own, shared } = partitionByScope(
                      filteredNamespaces,
                      filterCountry,
                    );
                    const renderRow = (
                      ns: (typeof filteredNamespaces)[number],
                    ) => (
                      <div
                        key={ns.id}
                        {...nsListNav.optionProps(ns.id === selectedNsId)}
                        className={`flex items-center justify-between px-2 py-1.5 rounded text-xs cursor-pointer transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
                          ns.id === selectedNsId
                            ? "bg-accent/10 text-accent font-medium shadow-[inset_3px_0_0_0] shadow-accent"
                            : "hover:bg-muted"
                        }`}
                        onClick={() => selectNamespace(ns.id)}
                      >
                        <div className="truncate flex-1 flex items-center gap-1.5">
                          {viewMode === "all" ? (
                            <CountryTagChip tag={ns.country_tag} />
                          ) : (
                            <ScopeBadge
                              entity={ns}
                              activeCountry={filterCountry}
                            />
                          )}
                          <span className="font-mono truncate">
                            {ns.namespace}
                          </span>
                        </div>
                        <span className="text-micro opacity-70 mx-1">
                          {ns.event_count}
                        </span>
                        <button
                          className="text-micro opacity-50 hover:opacity-100"
                          onClick={(e) => handleDeleteNamespace(ns.id, e)}
                          aria-label={t("deleteNamespace")}
                          title={t("deleteNamespace")}
                        >
                          X
                        </button>
                      </div>
                    );

                    const renderGrouped = (list: typeof filteredNamespaces) => {
                      if (list.length <= 6) return list.map(renderRow);
                      const groups = new Map<
                        string,
                        typeof filteredNamespaces
                      >();
                      for (const ns of list) {
                        const prefix = (
                          ns.namespace.split(/[_.]/, 1)[0] || ns.namespace
                        ).toLowerCase();
                        const existing = groups.get(prefix);
                        if (existing) existing.push(ns);
                        else groups.set(prefix, [ns]);
                      }
                      const sortedGroups = Array.from(groups.entries()).sort(
                        ([a], [b]) => a.localeCompare(b),
                      );
                      return sortedGroups.map(([prefix, items]) => {
                        if (items.length === 1) return renderRow(items[0]);
                        const isCollapsed = collapsedPrefixes.has(prefix);
                        return (
                          <div key={`prefix-${prefix}`} className="space-y-0.5">
                            <button
                              type="button"
                              onClick={() => togglePrefix(prefix)}
                              className="ui-interactive flex items-center gap-1 w-full px-2 py-1 rounded text-xs text-muted-foreground hover:bg-muted/60"
                              title={`Toggle ${prefix}_ group`}
                            >
                              <ChevronRight
                                className={`h-3 w-3 transition-transform ${isCollapsed ? "" : "rotate-90"}`}
                              />
                              <span className="font-mono">{prefix}_</span>
                              <Badge
                                variant="outline"
                                className="ml-auto h-4 px-1.5 text-micro"
                              >
                                {items.length}
                              </Badge>
                            </button>
                            {!isCollapsed && (
                              <div className="pl-3 space-y-0.5 border-l border-border ml-2">
                                {items.map(renderRow)}
                              </div>
                            )}
                          </div>
                        );
                      });
                    };
                    if (viewMode === "all") {
                      return (
                        <>
                          <ScopeSectionHeader
                            variant="all"
                            count={filteredNamespaces.length}
                          />
                          {renderGrouped(filteredNamespaces)}
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
                            {renderGrouped(own)}
                          </>
                        )}
                        {shared.length > 0 && (
                          <>
                            <ScopeSectionHeader
                              variant="shared"
                              count={shared.length}
                            />
                            {renderGrouped(shared)}
                          </>
                        )}
                      </>
                    );
                  })()}
                  {filteredNamespaces.length === 0 &&
                    (namespaces.length === 0 ? (
                      <p className="p-4 text-center text-xs text-muted-foreground">
                        {t("noNamespacesYet")}
                      </p>
                    ) : (
                      <div className="p-6 flex flex-col items-center gap-2 border border-dashed rounded-lg bg-muted/20 text-center">
                        <FolderOpen className="h-6 w-6 text-muted-foreground/60" />
                        <p className="text-xs font-medium text-foreground">
                          {t("noNamespacesForCountry", {
                            country: countryTag || t("thisCountry"),
                          })}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {t("switchCountryOrCreate")}
                        </p>
                      </div>
                    ))}
                </div>
              </div>

              {selectedNsId && (
                <>
                  <div className="flex items-center justify-between mb-2 gap-2">
                    <h3 className="text-sm font-semibold shrink-0">
                      {t("eventsHeading")}
                    </h3>
                    <div className="flex items-center gap-1 ml-auto">
                      <Button
                        size="sm"
                        variant={showOnlyOwn ? "default" : "outline"}
                        className="h-7 text-xs px-2"
                        onClick={() => {
                          userToggledScopeRef.current = true;
                          setShowOnlyOwn((v) => !v);
                          setSmartHiddenSharedCount(null);
                        }}
                        title={
                          showOnlyOwn
                            ? t("showAllTooltip")
                            : t("hideSharedTooltip")
                        }
                      >
                        {showOnlyOwn ? t("onlyOwn") : t("all")}
                      </Button>
                      <Button
                        size="sm"
                        className="h-7 text-xs"
                        onClick={handleCreateEvent}
                        disabled={isCreatingEvent || !selectedNsId}
                      >
                        {isCreatingEvent ? t("creating") : t("addEvent")}
                      </Button>
                    </div>
                  </div>

                  {events.length > 0 && (
                    <div className="mb-2 flex items-center gap-1.5">
                      <Input
                        value={eventSearch}
                        onChange={(e) => setEventSearch(e.target.value)}
                        placeholder={t("searchPlaceholder")}
                        className="text-xs h-7 flex-1"
                      />
                      {eventSearch && (
                        <Badge
                          variant="outline"
                          className="shrink-0 h-5 text-micro"
                        >
                          {filteredEvents.length} / {events.length}
                        </Badge>
                      )}
                    </div>
                  )}

                  {batchGfx.selectedIds.size > 0 && (
                    <div className="mb-2 rounded border border-primary/40 bg-primary/5 px-2 py-1.5 flex items-center gap-2 text-xs">
                      <span className="font-medium">
                        {t("selectedCount", {
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

                  {loading && events.length === 0 ? (
                    <div className="p-4 space-y-1.5">
                      {[...Array(3)].map((_, i) => (
                        <div
                          key={i}
                          className="h-7 rounded bg-muted/40 animate-pulse"
                        />
                      ))}
                    </div>
                  ) : error && events.length === 0 ? (
                    <WorkbenchLoadError
                      onRetry={() => {
                        if (selectedNsId) fetchEvents(projectId, selectedNsId);
                        else fetchNamespaces(projectId);
                      }}
                    />
                  ) : events.length === 0 ? (
                    <EditorEmptyState
                      icon={
                        <FileX className="h-6 w-6 text-muted-foreground/60" />
                      }
                      title={t("noEventsInNamespace")}
                      description={t("createFirstEvent")}
                      actions={
                        <Button
                          size="sm"
                          disabled={isCreatingEvent}
                          onClick={() => {
                            handleCreateEvent();
                          }}
                        >
                          {isCreatingEvent
                            ? t("creating")
                            : t("emptySecondFirstStep")}
                        </Button>
                      }
                    />
                  ) : filteredEvents.length === 0 ? (
                    <div className="p-6 flex flex-col items-center gap-2 border border-dashed rounded-lg bg-muted/20 text-center">
                      <Search className="h-6 w-6 text-muted-foreground/60" />
                      <p className="text-xs font-medium text-foreground">
                        {t("noMatchingEvents")}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {t("tryDifferentSearch")}
                      </p>
                    </div>
                  ) : (
                    <div
                      {...eventListNav.containerProps}
                      aria-label={t("eventsHeading")}
                      className="space-y-0.5 outline-none"
                    >
                      {filteredEvents.map((ev) => {
                        const isBatchSelected = batchGfx.selectedIds.has(ev.id);
                        return (
                          <div
                            key={ev.id}
                            {...eventListNav.optionProps(
                              ev.id === selectedEventId,
                            )}
                            className={`flex items-center justify-between px-2 py-1.5 rounded text-xs cursor-pointer transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
                              ev.id === selectedEventId
                                ? "bg-accent/10 text-accent font-medium shadow-[inset_3px_0_0_0] shadow-accent"
                                : isBatchSelected
                                  ? "bg-success/10"
                                  : "hover:bg-muted"
                            }`}
                            onClick={() => selectEvent(ev.id)}
                          >
                            <input
                              type="checkbox"
                              checked={isBatchSelected}
                              onClick={(e) => {
                                e.stopPropagation();
                                batchGfx.toggle(ev.id, {
                                  range: e.shiftKey,
                                  sortedIds: filteredEvents.map((x) => x.id),
                                });
                              }}
                              onChange={() => {}}
                              title={t("batchSelectTooltip")}
                              className="accent-accent h-3 w-3 shrink-0 mr-2"
                            />
                            <div className="truncate flex-1 flex items-center gap-1.5">
                              {viewMode === "all" ? (
                                <CountryTagChip tag={ev.country_tag} />
                              ) : (
                                <ScopeBadge
                                  entity={ev}
                                  activeCountry={filterCountry}
                                />
                              )}
                              <span className="font-mono truncate">
                                {ev.event_id}
                              </span>
                              {ev.hidden && (
                                <span className="ml-1 opacity-60 shrink-0">
                                  (H)
                                </span>
                              )}
                              {ev.major && (
                                <span className="ml-1 opacity-60 shrink-0">
                                  (M)
                                </span>
                              )}
                            </div>
                            <button
                              className="text-micro opacity-50 hover:opacity-100 ml-1"
                              onClick={(e) => handleDeleteEvent(ev.id, e)}
                            >
                              X
                            </button>
                          </div>
                        );
                      })}
                    </div>
                  )}

                  {!eventSearch && (
                    <p className="text-xs text-muted-foreground mt-3">
                      {events.length} event{events.length !== 1 ? "s" : ""}
                    </p>
                  )}
                </>
              )}
            </div>
          }
          editorPanel={
            selected ? (
              <EventEditor
                event={selected}
                onSave={handleSave}
                projectId={projectId}
                countryTag={countryTag}
                onPictureUploaded={() => {
                  if (selectedNsId) fetchEvents(projectId, selectedNsId);
                  if (selectedEventId) fetchPreview(projectId, selectedEventId);
                }}
                onDraftChange={setEventDraft}
                focusTarget={previewTarget}
                focusOptionIndex={previewOptionIndex}
                onFocusTargetConsumed={handlePreviewTargetConsumed}
                onForked={async (forked) => {
                  if (selectedNsId) await fetchEvents(projectId, selectedNsId);
                  selectEvent(forked.id);
                }}
              />
            ) : error && namespaces.length === 0 ? (
              <div className="flex h-full items-center justify-center p-8">
                <WorkbenchLoadError
                  onRetry={() => fetchNamespaces(projectId)}
                />
              </div>
            ) : namespaces.length === 0 ? (
              <div className="flex h-full items-center justify-center p-8">
                <EditorEmptyState
                  variant="hero"
                  className="max-w-md"
                  icon={
                    <FolderOpen className="h-10 w-10 text-muted-foreground/60" />
                  }
                  title={t("emptyHeroTitle")}
                  description={t("emptyHeroDesc")}
                  actions={
                    <>
                      {hasCountries ? (
                        <Button
                          size="sm"
                          onClick={() => {
                            nsInputRef.current?.focus({ preventScroll: true });
                            nsInputRef.current?.scrollIntoView({
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
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center h-full gap-3 text-center p-8">
                <FolderOpen className="h-8 w-8 text-muted-foreground/40" />
                <p className="text-sm font-medium text-foreground">
                  {selectedNsId
                    ? t("noEventSelected")
                    : t("noNamespaceSelected")}
                </p>
                <p className="text-xs text-muted-foreground">
                  {selectedNsId
                    ? t("selectOrCreateEvent")
                    : t("createOrSelectNamespaceFirst")}
                </p>
              </div>
            )
          }
          previewPanel={
            <WorkbenchRightPane
              activeTabId={rightPaneTabId}
              onTabChange={setRightPaneTabId}
              tabs={[
                {
                  id: "preview",
                  label: tShell("tabLivePreview"),
                  icon: <Eye className="h-3.5 w-3.5" />,
                  content: previewEvent ? (
                    <div className="flex h-full min-h-0 flex-col overflow-y-auto p-3">
                      {previewIsDraft && (
                        <div className="mb-2 flex shrink-0 justify-end">
                          <span className="rounded border border-accent/50 bg-accent/15 px-2 py-0.5 text-micro font-mono text-accent">
                            {tShell("liveDraftBadge")}
                          </span>
                        </div>
                      )}
                      <EventPreviewBoundary
                        resetKey={`${previewEvent.id}:${previewEvent.updated_at ?? ""}`}
                      >
                        <EventInGamePreview
                          gameEvent={previewEvent}
                          projectId={projectId}
                          isDraft={previewIsDraft}
                          activeTarget={previewTarget}
                          activeOptionIndex={previewOptionIndex}
                          onSelectTarget={handleSelectPreviewTarget}
                        />
                      </EventPreviewBoundary>
                    </div>
                  ) : (
                    <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
                      {selectedNsId
                        ? events.length === 0
                          ? tShell("livePreviewCreateFirstHint")
                          : tShell("livePreviewSelectHint")
                        : t("createOrSelectNamespaceFirst")}
                    </div>
                  ),
                },
                {
                  id: "vanilla",
                  label: tVanilla("tabVanillaEvents"),
                  icon: <BookOpen className="h-3.5 w-3.5" />,
                  content: (
                    <VanillaEventsTab
                      onUseAsTemplate={handleUseAsTemplate}
                      canUseAsTemplate={!!selectedNsId}
                      useAsTemplateDisabledHint={tVanilla(
                        "useAsTemplateNamespaceHint",
                      )}
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
                      kind="event"
                      sourceId={selectedEventId}
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
                    <div className="h-full min-h-0">
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
