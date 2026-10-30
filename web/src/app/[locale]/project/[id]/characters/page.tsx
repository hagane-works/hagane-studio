"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import useSWR from "swr";
import { toast } from "sonner";
import { useTranslations } from "use-intl";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { visibleCharacterName } from "@/lib/i18n/characterDisplayName";
import {
  Award,
  ChevronRight,
  Eye,
  FileCode2,
  Link2,
  Search,
  Users,
  X,
} from "lucide-react";
import { useParams, useSearchParams } from "@/platform/navigation";
import { useCharacterStore } from "@/lib/store/characterStore";
import { useShallow } from "zustand/react/shallow";
import {
  useActiveCountry,
  useEffectiveCountryFilter,
  useProjectStore,
  useProjectCountries,
} from "@/lib/store/projectStore";
import {
  matchesCountryScope,
  partitionByScope,
  getScopeSource,
} from "@/lib/utils/scope";
import type { ScopedEntity } from "@/lib/utils/scope";
import { ScopeSectionHeader } from "@/components/workbench/ScopeBadge";
import { CountryTagChip } from "@/components/workbench/CountryTagChip";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { WorkbenchRightPane } from "@/components/workbench/WorkbenchRightPane";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import { ListPanelHint } from "@/components/workbench/ListPanelHint";
import { CharacterEditor } from "@/components/workbench/CharacterEditor";
import { CharacterInGamePreview } from "@/components/workbench/CharacterInGamePreview";
import { ScriptEditor } from "@/shared/editors/ScriptEditor";
import { MultiFilePreviewTabs } from "@/shared/editors/MultiFilePreviewTabs";
import { RelationsPanel } from "@/components/workbench/RelationsPanel";
import { useEditorWiring } from "@/shared/editors/useEditorWiring";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { ImportButton } from "@/components/workbench/ImportButton";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { AddCountryDialog } from "@/components/workbench/AddCountryDialog";
import { charactersApi, importApi } from "@/lib/api";
import {
  useVanillaCountryTags,
  vanillaTagName,
} from "@/lib/vanillaCountryTags";
import type { Character, RoleType } from "@/types";

const GROUP_BY_COUNTRY_THRESHOLD = 50;

const PER_GROUP_RENDER_CAP = 100;

function groupByCountryStorageKey(
  projectId: number | null | undefined,
): string | null {
  if (!projectId) return null;
  return `b4-charactersGroupByCountry-${projectId}`;
}

function readGroupByCountryPref(
  projectId: number | null | undefined,
): boolean | null {
  if (typeof window === "undefined") return null;
  const key = groupByCountryStorageKey(projectId);
  if (!key) return null;
  const raw = window.localStorage.getItem(key);
  if (raw === "1") return true;
  if (raw === "0") return false;
  return null;
}

function ScopeDot({
  entity,
  activeCountry,
}: {
  entity: ScopedEntity;
  activeCountry: string | null | undefined;
}) {
  const t = useTranslations("scopeBadge");
  const source = getScopeSource(entity, activeCountry);
  const sharedTags = (entity.shared_country_tags ?? []).join(", ") || "—";
  const title =
    source === "explicit"
      ? t("explicitTitle", { tags: sharedTags })
      : source === "global"
        ? t("globalTitle")
        : source === "global_except"
          ? t("globalExceptTitle", { tags: sharedTags })
          : t("ownTitle", { tag: entity.country_tag ?? "—" });
  return (
    <span
      className="inline-block h-1.5 w-1.5 shrink-0 rounded-full bg-muted-foreground/60"
      title={title}
      aria-label={title}
    />
  );
}

export default function CharactersPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const projectId = Number(params.id);
  const t = useTranslations("characters");
  const tc = useTranslations("common");
  const tShell = useTranslations("workbenchShell");
  const tRel = useTranslations("relations");

  const locale = useContentLocaleStore((s) => s.locale);
  const { lookup: lookupVanillaTag } = useVanillaCountryTags();

  const listNav = useListboxNav();

  const confirmDialog = useConfirm();

  const ROLE_FILTER_TABS: { value: RoleType | ""; label: string }[] = useMemo(
    () => [
      { value: "", label: tc("all") },
      {
        value: "country_leader" as RoleType,
        label: t("filterRoleCountryLeader"),
      },
      { value: "general" as RoleType, label: t("filterRoleGeneral") },
      {
        value: "field_marshal" as RoleType,
        label: t("filterRoleFieldMarshal"),
      },
      { value: "admiral" as RoleType, label: t("filterRoleAdmiral") },
      { value: "advisor" as RoleType, label: t("filterRoleAdvisor") },
      { value: "scientist" as RoleType, label: t("filterRoleScientist") },
      { value: "operative" as RoleType, label: t("filterRoleOperative") },
    ],
    [t, tc],
  );

  const countryTag = useActiveCountry();
  const filterCountry = useEffectiveCountryFilter();
  const viewMode = useProjectStore((s) => s.viewMode);
  const projectCountries = useProjectCountries();
  const projectIdFromStore = useProjectStore((s) => s.currentProject?.id);

  const [groupByCountryPref, setGroupByCountryPref] = useState<boolean | null>(
    null,
  );

  const [lastProjectIdForPref, setLastProjectIdForPref] = useState<
    number | null | undefined
  >(undefined);
  if (lastProjectIdForPref !== projectIdFromStore) {
    setLastProjectIdForPref(projectIdFromStore);
    setGroupByCountryPref(readGroupByCountryPref(projectIdFromStore));
  }

  const smartGroupDefault =
    projectCountries.length > GROUP_BY_COUNTRY_THRESHOLD;
  const groupByCountryActive =
    viewMode === "all" && (groupByCountryPref ?? smartGroupDefault);

  const toggleGroupByCountry = useCallback(() => {
    const next = !groupByCountryActive;
    setGroupByCountryPref(next);
    if (typeof window !== "undefined") {
      const key = groupByCountryStorageKey(projectIdFromStore);
      if (key) window.localStorage.setItem(key, next ? "1" : "0");
    }
  }, [groupByCountryActive, projectIdFromStore]);

  const [searchQuery, setSearchQuery] = useState("");

  const [collapsedCountries, setCollapsedCountries] = useState<Set<string>>(
    () => new Set(),
  );
  const toggleCountryCollapse = useCallback((tag: string) => {
    setCollapsedCountries((prev) => {
      const next = new Set(prev);
      if (next.has(tag)) next.delete(tag);
      else next.add(tag);
      return next;
    });
  }, []);

  const [showAllForGroup, setShowAllForGroup] = useState<Set<string>>(
    () => new Set(),
  );
  const showAllForGroupHandler = useCallback((tag: string) => {
    setShowAllForGroup((prev) => {
      if (prev.has(tag)) return prev;
      const next = new Set(prev);
      next.add(tag);
      return next;
    });
  }, []);

  const {
    characters,
    selectedId,
    scriptPreview,
    loading,
    error,
    roleTypeFilter,
    fetchCharacters,
    selectCharacter,
    setRoleTypeFilter,
    createCharacter,
    updateCharacter,
    deleteCharacter,
    fetchPreview,
  } = useCharacterStore(
    useShallow((s) => ({
      characters: s.characters,
      selectedId: s.selectedId,
      scriptPreview: s.scriptPreview,
      loading: s.loading,
      error: s.error,
      roleTypeFilter: s.roleTypeFilter,
      fetchCharacters: s.fetchCharacters,
      selectCharacter: s.selectCharacter,
      setRoleTypeFilter: s.setRoleTypeFilter,
      createCharacter: s.createCharacter,
      updateCharacter: s.updateCharacter,
      deleteCharacter: s.deleteCharacter,
      fetchPreview: s.fetchPreview,
    })),
  );

  const searchNeedle = searchQuery.trim().toLowerCase();

  const filteredCharacters = useMemo(() => {
    let byRole = characters;
    if (roleTypeFilter === "advisor") {
      byRole = characters.filter(
        (c) => c.role_type === "advisor" || c.has_advisor_role,
      );
    } else if (roleTypeFilter) {
      byRole = characters.filter((c) => c.role_type === roleTypeFilter);
    }

    let bySearch = byRole;
    if (searchNeedle) {
      bySearch = byRole.filter((c) => {
        if ((c.name_en ?? "").toLowerCase().includes(searchNeedle)) return true;
        if ((c.name_zh ?? "").toLowerCase().includes(searchNeedle)) return true;
        if ((c.name ?? "").toLowerCase().includes(searchNeedle)) return true;
        if ((c.tag ?? "").toLowerCase().includes(searchNeedle)) return true;
        if ((c.localisation_key ?? "").toLowerCase().includes(searchNeedle)) {
          return true;
        }
        if ((c.role_type ?? "").toLowerCase().includes(searchNeedle))
          return true;
        if ((c.country_tag ?? "").toLowerCase().includes(searchNeedle))
          return true;

        const locs = c.display_name_locs || {};
        return Object.values(locs).some((v) =>
          (v ?? "").toLowerCase().includes(searchNeedle),
        );
      });
    }

    if (!filterCountry) return bySearch;
    return bySearch.filter((c) => matchesCountryScope(c, filterCountry));
  }, [characters, roleTypeFilter, filterCountry, searchNeedle]);

  const { own: ownCharacters, shared: sharedCharacters } = useMemo(
    () => partitionByScope(filteredCharacters, filterCountry),
    [filteredCharacters, filterCountry],
  );

  const UNATTRIBUTED = "__UNATTRIBUTED__";
  const groupedByCountry = useMemo(() => {
    if (!groupByCountryActive) return null;
    const groups = new Map<string, Character[]>();
    for (const c of filteredCharacters) {
      const key =
        c.country_tag && c.country_tag.length > 0
          ? c.country_tag
          : UNATTRIBUTED;
      const existing = groups.get(key);
      if (existing) existing.push(c);
      else groups.set(key, [c]);
    }

    const entries = Array.from(groups.entries()).sort(([a], [b]) => {
      if (a === UNATTRIBUTED) return 1;
      if (b === UNATTRIBUTED) return -1;
      return a.localeCompare(b);
    });
    return entries;
  }, [groupByCountryActive, filteredCharacters]);

  useEffect(() => {
    if (!selectedId) return;
    const sel = characters.find((c) => c.id === selectedId);
    if (!sel) return;

    if (filterCountry && !matchesCountryScope(sel, filterCountry)) {
      selectCharacter(null);
    }
  }, [filterCountry, selectedId, characters, selectCharacter]);

  const didRestoreSelection = useRef(false);
  useEffect(() => {
    if (didRestoreSelection.current) return;
    if (characters.length === 0) return;
    const raw = searchParams?.get("c");
    if (raw) {
      const id = Number(raw);
      if (Number.isFinite(id) && characters.some((c) => c.id === id)) {
        selectCharacter(id);
      }
    }
    didRestoreSelection.current = true;
  }, [characters, searchParams, selectCharacter]);

  useEffect(() => {
    if (!didRestoreSelection.current) return;
    if (typeof window === "undefined") return;
    const url = new URL(window.location.href);
    if (selectedId) url.searchParams.set("c", String(selectedId));
    else url.searchParams.delete("c");
    window.history.replaceState(window.history.state, "", url);
  }, [selectedId]);

  useSWR(projectId ? ["characters/list", projectId] : null, () =>
    fetchCharacters(projectId),
  );
  useSWR(
    projectId && selectedId
      ? ["characters/preview", projectId, selectedId]
      : null,
    () => fetchPreview(projectId, selectedId!),
  );

  const parsePreviewCall = useCallback(
    (script: string) =>
      charactersApi.parsePreview(projectId, selectedId!, script),
    [projectId, selectedId],
  );
  const wiring = useEditorWiring({
    workbench: "characters",
    entityId: selectedId,
    scriptPreview,
    parsePreview: parsePreviewCall,
    applyEntity: (id, entityDict) =>
      updateCharacter(projectId, id, entityDict as Partial<Character>),
    onAfterApply: async (id) => {
      await fetchCharacters(projectId);
      await fetchPreview(projectId, id);
    },
    entityNoun: "character",
    projectId,
  });

  const selected = characters.find((c) => c.id === selectedId) || null;
  const [characterDraft, setCharacterDraft] = useState<Character | null>(null);

  const [portraitVersion, setPortraitVersion] = useState(0);
  const previewCharacter =
    selected && characterDraft?.id === selected.id ? characterDraft : selected;
  const characterPreviewIsDraft = useMemo(
    () =>
      Boolean(selected && previewCharacter) &&
      JSON.stringify(previewCharacter) !== JSON.stringify(selected),
    [previewCharacter, selected],
  );

  const handleCreate = async () => {
    await createCharacter(projectId, {
      name: "",
      name_en: "",
      tag: `${countryTag || "MOD"}_new_character`,
      localisation_key: "",
      role_type: "general",
      country_tag: countryTag,
    });
  };

  const [addCountryOpen, setAddCountryOpen] = useState(false);
  const noCountries =
    Boolean(projectIdFromStore) && projectCountries.length === 0;
  const handleHeroFirstStep = () => {
    if (noCountries) {
      setAddCountryOpen(true);
      return;
    }
    void handleCreate();
  };

  const handleSave = async (data: Partial<Character>) => {
    if (selectedId) {
      await updateCharacter(projectId, selectedId, data);
    }
  };

  const handleDelete = async (id: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (await confirmDialog({ body: t("confirmDelete"), danger: true })) {
      await deleteCharacter(projectId, id);
    }
  };

  const handleFlipBack = async () => {
    if (!selected) return;
    const tag = selected.tag;
    if (
      !(await confirmDialog({
        body:
          `Restore vanilla "${tag}"?\n\n` +
          `This will delete the replacement character row (cascade removes attached traits/portrait flag).\n\n` +
          `On the next export, vanilla "${tag}" displays again — the trait_overlay event, portrait override, and history file retire+create block all stop emitting for this character.\n\n` +
          `Destructive · not undoable.`,
        danger: true,
      }))
    ) {
      return;
    }
    try {
      await deleteCharacter(projectId, selected.id);
      selectCharacter(null);

      toast.success(t("flipBackSuccess", { tag }));
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      toast.error(t("flipBackError", { msg }));
    }
  };

  return (
    <div className="flex h-full min-w-0 overflow-hidden">
      <div className="flex-1 min-w-0 h-full overflow-hidden">
        <WorkbenchShell
          gameViewHref={gameViewHref(projectId, {
            panel: "countrypoliticsview",
            tag: selected?.country_tag || countryTag,
          })}
          previewWidth="w-96"
          listPanel={
            <div className="p-3">
              <WorkbenchListHeader
                title={t("workbenchTitle")}
                count={filteredCharacters.length}
                totalCount={characters.length}
                searchValue=""
                onSearchChange={() => {}}
                renderSearch={false}
                className="p-0 mb-3"
                action={
                  <>
                    <Button
                      size="sm"
                      className="text-xs font-semibold"
                      onClick={handleCreate}
                    >
                      {t("newCharacter")}
                    </Button>
                  </>
                }
              />
              <div className="flex flex-col gap-1 mb-3">
                <ListPanelHint className="mb-3">
                  {t.rich("importSidebarHint", {
                    b: (chunks) => (
                      <span className="font-medium text-foreground">
                        {chunks}
                      </span>
                    ),
                  })}
                </ListPanelHint>
                <ImportButton
                  label={t("importTxtOnly")}
                  onImport={(file) =>
                    importApi.characters(projectId, file, countryTag)
                  }
                  onComplete={() => fetchCharacters(projectId)}
                />
              </div>

              {characters.length > 0 && (
                <div className="flex items-center gap-1 mb-2">
                  <div className="relative flex-1">
                    <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                    <Input
                      type="text"
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      placeholder={t("searchPlaceholder")}
                      className="h-7 pl-7 pr-7 text-xs focus:ring-1 focus:ring-ring"
                    />
                    {searchQuery && (
                      <button
                        type="button"
                        onClick={() => setSearchQuery("")}
                        aria-label={tc("clear")}
                        title={tc("clear")}
                        className="absolute right-1.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    )}
                  </div>
                  {viewMode === "all" && projectCountries.length > 1 && (
                    <button
                      type="button"
                      onClick={toggleGroupByCountry}
                      aria-pressed={groupByCountryActive}
                      title={
                        groupByCountryActive
                          ? t("groupedViewActive")
                          : t("flatViewActive")
                      }
                      className={
                        "shrink-0 h-7 px-2 rounded-md border text-xs font-mono ui-interactive flex items-center gap-1 " +
                        (groupByCountryActive
                          ? "border-accent/60 bg-accent/15 text-accent-foreground"
                          : "border-border text-muted-foreground hover:text-foreground hover:bg-muted/50")
                      }
                    >
                      <span aria-hidden>
                        {groupByCountryActive ? "▤" : "≡"}
                      </span>
                      <span>{t("groupByCountry")}</span>
                    </button>
                  )}
                </div>
              )}

              {characters.length > 0 && (
                <div className="flex flex-wrap gap-1.5 mb-3">
                  {ROLE_FILTER_TABS.map((tab) => {
                    let count: number;
                    if (!tab.value) {
                      count = characters.length;
                    } else if (tab.value === "advisor") {
                      count = characters.filter(
                        (c) => c.role_type === "advisor" || c.has_advisor_role,
                      ).length;
                    } else {
                      count = characters.filter(
                        (c) => c.role_type === tab.value,
                      ).length;
                    }
                    return (
                      <button
                        key={tab.value}
                        disabled={count === 0 && tab.value !== ""}
                        className={`px-2.5 py-1 rounded-md text-xs ui-interactive ${
                          roleTypeFilter === tab.value
                            ? "bg-accent/10 text-accent font-semibold"
                            : count === 0 && tab.value !== ""
                              ? "text-muted-foreground/40 cursor-not-allowed"
                              : "text-muted-foreground hover:text-foreground hover:bg-muted"
                        }`}
                        onClick={() => {
                          if (count > 0 || !tab.value)
                            setRoleTypeFilter(tab.value as RoleType | "");
                        }}
                      >
                        {tab.label}
                        {tab.value !== "" && count > 0 && (
                          <span className="ml-1 text-micro opacity-50">
                            {count}
                          </span>
                        )}
                      </button>
                    );
                  })}
                </div>
              )}
              {loading && characters.length === 0 ? (
                <div className="space-y-1.5">
                  {Array.from({ length: 8 }).map((_, i) => (
                    <div
                      key={`skel-${i}`}
                      className="flex items-center justify-between px-3 py-2 rounded-md"
                    >
                      <div className="flex items-center gap-2 flex-1 min-w-0">
                        <div className="h-3 w-8 rounded bg-muted/50 animate-pulse" />
                        <div className="h-3 w-32 rounded bg-muted/60 animate-pulse" />
                      </div>
                      <div className="h-3 w-12 rounded bg-muted/40 animate-pulse" />
                    </div>
                  ))}
                </div>
              ) : characters.length === 0 ? (
                <p className="text-xs text-muted-foreground">
                  {t("emptyNoCharacters")}
                </p>
              ) : filteredCharacters.length === 0 ? (
                <p className="text-xs text-muted-foreground">
                  {t("emptyNoType")}
                </p>
              ) : (
                <div
                  {...listNav.containerProps}
                  aria-label={t("workbenchTitle")}
                  className="space-y-1 outline-none"
                >
                  {(() => {
                    const renderRow = (
                      char: (typeof filteredCharacters)[number],
                    ) => {
                      const isDualRole =
                        char.has_advisor_role && char.role_type !== "advisor";
                      const roleLabel = isDualRole
                        ? `${char.role_type} ⚡ advisor`
                        : char.role_type;

                      const displayName = visibleCharacterName(
                        resolveDisplayName(
                          locale,
                          char.display_name_locs,
                          char.name_en,
                          char.name_zh,
                          "",
                        ),
                      );
                      const isUnnamed = displayName.length === 0;
                      return (
                        <div
                          key={char.id}
                          {...listNav.optionProps(selectedId === char.id)}
                          className={`group flex items-center justify-between px-3 py-2 rounded-md cursor-pointer text-sm transition-colors focus:outline-none focus-visible:ring-1 focus-visible:ring-ring ${
                            selectedId === char.id
                              ? "bg-primary/10 text-primary"
                              : "hover:bg-muted"
                          }`}
                          onClick={() => selectCharacter(char.id)}
                        >
                          <div className="min-w-0 flex-1">
                            <p className="font-medium truncate flex items-center gap-1.5">
                              {viewMode === "all" && !groupByCountryActive ? (
                                <CountryTagChip tag={char.country_tag} />
                              ) : !groupByCountryActive ? (
                                <ScopeDot
                                  entity={char}
                                  activeCountry={filterCountry}
                                />
                              ) : null}
                              {isUnnamed ? (
                                <span className="truncate text-muted-foreground">
                                  {t("unnamedCharacter")}
                                </span>
                              ) : (
                                <span className="truncate">{displayName}</span>
                              )}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {roleLabel} ·{" "}
                              <span className="font-mono break-all">
                                {char.tag}
                              </span>
                            </p>
                          </div>

                          <button
                            type="button"
                            aria-label={tc("delete")}
                            title={tc("delete")}
                            className="text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 focus-visible:opacity-100 shrink-0 ml-2 rounded focus:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                            onClick={(e) => handleDelete(char.id, e)}
                          >
                            <X className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      );
                    };

                    if (groupByCountryActive && groupedByCountry) {
                      if (groupedByCountry.length === 0) {
                        return (
                          <p className="text-xs text-muted-foreground">
                            {t("noMatches")}
                          </p>
                        );
                      }
                      return groupedByCountry.map(([groupTag, items], idx) => {
                        if (items.length === 0) return null;

                        const defaultExpanded = idx === 0;
                        const forceExpand = searchNeedle.length > 0;
                        const toggledByUser = collapsedCountries.has(groupTag);
                        const isExpanded = forceExpand
                          ? true
                          : toggledByUser
                            ? !defaultExpanded
                            : defaultExpanded;
                        const isUnattributed = groupTag === UNATTRIBUTED;
                        const vanilla = isUnattributed
                          ? null
                          : lookupVanillaTag(groupTag);
                        const groupLabel = isUnattributed
                          ? t("unattributed")
                          : vanilla
                            ? vanillaTagName(vanilla, locale)
                            : groupTag;
                        const totalForGroup = items.length;
                        const cap = showAllForGroup.has(groupTag)
                          ? totalForGroup
                          : PER_GROUP_RENDER_CAP;
                        const visible =
                          totalForGroup > cap ? items.slice(0, cap) : items;
                        const hiddenCount = totalForGroup - visible.length;
                        return (
                          <div
                            key={`group-${groupTag}`}
                            className="space-y-0.5"
                          >
                            <button
                              type="button"
                              onClick={() => toggleCountryCollapse(groupTag)}
                              aria-expanded={isExpanded}
                              className="ui-interactive flex items-center gap-1.5 w-full px-2 py-1 rounded text-xs text-muted-foreground hover:bg-muted/60"
                              title={
                                isUnattributed
                                  ? groupLabel
                                  : `${groupTag} · ${groupLabel}`
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
                                {isUnattributed ? "—" : groupTag}
                              </span>
                              <span className="truncate opacity-70">
                                {groupLabel}
                              </span>
                              <Badge
                                variant="outline"
                                className="ml-auto h-4 px-1.5 text-micro shrink-0"
                              >
                                {totalForGroup}
                              </Badge>
                            </button>
                            {isExpanded && (
                              <div className="pl-3 space-y-0.5 border-l border-border ml-2">
                                {visible.map(renderRow)}
                                {hiddenCount > 0 && (
                                  <button
                                    type="button"
                                    onClick={() =>
                                      showAllForGroupHandler(groupTag)
                                    }
                                    className="ui-interactive w-full px-2 py-1 text-xs text-muted-foreground hover:text-foreground hover:bg-muted/40 rounded text-left"
                                  >
                                    {t("showMore", { count: hiddenCount })}
                                  </button>
                                )}
                              </div>
                            )}
                          </div>
                        );
                      });
                    }

                    if (!filterCountry) {
                      return (
                        <>
                          <ScopeSectionHeader
                            variant="all"
                            count={filteredCharacters.length}
                          />
                          {filteredCharacters.map(renderRow)}
                        </>
                      );
                    }
                    return (
                      <>
                        {ownCharacters.length > 0 && (
                          <>
                            <ScopeSectionHeader
                              variant="own"
                              count={ownCharacters.length}
                            />
                            {ownCharacters.map(renderRow)}
                          </>
                        )}
                        {sharedCharacters.length > 0 && (
                          <>
                            <ScopeSectionHeader
                              variant="shared"
                              count={sharedCharacters.length}
                            />
                            {sharedCharacters.map(renderRow)}
                          </>
                        )}
                      </>
                    );
                  })()}
                </div>
              )}
            </div>
          }
          editorPanel={
            selected ? (
              <CharacterEditor
                projectId={projectId}
                character={selected}
                projectCharacters={characters}
                onSave={handleSave}
                onRefresh={() => {
                  setPortraitVersion((value) => value + 1);
                  void fetchCharacters(projectId);
                }}
                onDraftChange={setCharacterDraft}
                onFlipBack={handleFlipBack}
                onForked={async (forked) => {
                  await fetchCharacters(projectId);
                  selectCharacter(forked.id);
                }}
              />
            ) : error && characters.length === 0 ? (
              <div className="flex h-full items-center justify-center p-8">
                <WorkbenchLoadError
                  onRetry={() => fetchCharacters(projectId)}
                />
              </div>
            ) : !loading && characters.length === 0 ? (
              <div className="flex h-full items-center justify-center p-8">
                <EditorEmptyState
                  variant="hero"
                  className="w-full max-w-md"
                  icon={
                    <Users className="h-10 w-10 text-muted-foreground/60" />
                  }
                  title={t("emptyHeroTitle")}
                  description={t("emptyHeroDesc")}
                  actions={
                    <>
                      <Button size="sm" onClick={handleHeroFirstStep}>
                        {noCountries
                          ? tc("emptyCtaAddCountry")
                          : t("emptyHeroFirstStep")}
                      </Button>
                    </>
                  }
                />
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center h-full gap-4 text-center p-8">
                <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center">
                  <Award
                    className="h-6 w-6 text-muted-foreground"
                    aria-hidden
                  />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">
                    {t("selectOrCreate")}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {t("selectHint")}
                  </p>
                </div>
                <Button size="sm" variant="outline" onClick={handleCreate}>
                  {t("newCharacter")}
                </Button>
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
                  content: previewCharacter ? (
                    <div className="flex h-full min-h-0 flex-col p-3">
                      {characterPreviewIsDraft && (
                        <div className="mb-2 flex shrink-0 justify-end">
                          <span className="rounded border border-accent/50 bg-accent/15 px-2 py-0.5 text-micro font-mono text-accent">
                            {tShell("liveDraftBadge")}
                          </span>
                        </div>
                      )}
                      <CharacterInGamePreview
                        character={previewCharacter}
                        projectId={projectId}
                        isDraft={characterPreviewIsDraft}
                        portraitVersion={portraitVersion}
                      />
                    </div>
                  ) : (
                    <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
                      {characters.length === 0
                        ? t("previewEmptyNoCharacters")
                        : tShell("livePreviewSelectHint")}
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
                      kind="character"
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
                    <MultiFilePreviewTabs
                      projectId={projectId}
                      characterId={selectedId}
                      scriptEditor={
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
                      }
                    />
                  ),
                },
              ]}
            />
          }
        />
      </div>

      <AddCountryDialog
        projectId={projectId}
        open={addCountryOpen}
        onOpenChange={setAddCountryOpen}
      />
    </div>
  );
}
