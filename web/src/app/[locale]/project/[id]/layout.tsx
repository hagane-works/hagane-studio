"use client";

import {
  Fragment,
  useEffect,
  useCallback,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { useTranslations } from "use-intl";
import { useParams } from "@/platform/navigation";
import { Link, usePathname } from "@/i18n/navigation";
import {
  gameViewHref,
  gameViewWindowName,
  workbenchWindowName,
} from "@/lib/gameViewLinks";
import {
  useBeforeUnloadGuard,
  confirmDiscardIfDirty,
} from "@/lib/hooks/useUnsavedGuard";
import { useEditorDirtyStore } from "@/lib/store/editorDirtyStore";
import {
  BookOpen,
  Boxes,
  Check,
  ChevronDown,
  ChevronsDownUp,
  ChevronRight,
  Cog,
  Compass,
  ExternalLink,
  Gamepad2,
  Crown,
  Dices,
  Download,
  Eye,
  Factory,
  FileText,
  FlaskConical,
  FolderSync,
  Gavel,
  GraduationCap,
  Image as ImageIcon,
  Info,
  LayoutGrid,
  Lightbulb,
  ListTree,
  MapPin,
  MessageSquare,
  PanelLeft,
  PanelsTopLeft,
  PencilLine,
  Plus,
  Puzzle,
  ScrollText,
  Send,
  Settings,
  Sparkles,
  Swords,
  Target,
  User,
  X,
  type LucideIcon,
} from "lucide-react";
import {
  useProjectStore,
  useActiveCountry,
  useProjectCountries,
} from "@/lib/store/projectStore";
import { usePendingModFolderProjectId } from "@/lib/store/pendingModFolderStore";
import {
  dialogForSection,
  useOpenProjectSettings,
} from "@/lib/store/openProjectSettingsStore";
import { useCharacterStore } from "@/lib/store/characterStore";
import { useFocusTreeStore } from "@/lib/store/focusTreeStore";
import { useTechnologyStore } from "@/lib/store/technologyStore";
import { useIdeaStore } from "@/lib/store/ideaStore";
import { useEventStore } from "@/lib/store/eventStore";
import { useDecisionStore } from "@/lib/store/decisionStore";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  Popover,
  PopoverContent,
  PopoverTitle,
  PopoverTrigger,
} from "@/components/ui/popover";
import { UnifiedModImport } from "@/components/workbench/UnifiedModImport";
import { shouldMountModImport } from "@/components/workbench/import-wizard/mountGate";
import { CountrySwitcher } from "@/components/workbench/CountrySwitcher";
import { useLinkedCountry } from "@/components/workbench/useLinkedCountry";
import { ProjectTopBar } from "@/components/workbench/ProjectTopBar";
import {
  CountryFlag,
  isResolvedCountryName,
} from "@/components/workbench/CountryFlag";
import { AllCountriesToggle } from "@/components/workbench/AllCountriesToggle";
import { AddCountryDialogContext } from "@/components/workbench/AddCountryDialogContext";
import { AddCountryDialog } from "@/components/workbench/AddCountryDialog";
import { CountrySettingsDialog } from "@/components/workbench/CountrySettingsDialog";
import { PoliticsSettingsDialog } from "@/components/workbench/PoliticsSettingsDialog";
import { ProjectSettingsDialog } from "@/components/workbench/ProjectSettingsDialog";
import { EditTargetIndicator } from "@/components/workbench/EditTargetIndicator";
import { SidebarItemTip } from "./SidebarItemTip";
import { SidebarNavPending } from "./SidebarNavPending";
import { LanguageSwitcher } from "@/components/i18n/LanguageSwitcher";
import { ContentLanguageSwitcher } from "@/components/i18n/ContentLanguageSwitcher";
import { ThemeToggle } from "@/components/i18n/ThemeToggle";
import { useVanillaTagName } from "@/lib/vanillaCountryTags";
import { SaveFullModFolderButton } from "@/components/workbench/SaveFullModFolderButton";
import { FullModPreviewButton } from "@/components/workbench/FullModPreviewDialog";
type NavItem = {
  navKey: string;
  path: string;
  icon?: LucideIcon;
  emoji?: string;
};
const NAV_ITEMS: NavItem[] = [
  { navKey: "navCharacters", path: "characters", icon: User },
  { navKey: "navIdeas", path: "ideas", icon: Lightbulb },
  { navKey: "navFocusTree", path: "focus-tree", icon: Target },
  { navKey: "navEvents", path: "events", icon: MessageSquare },
  { navKey: "navDecisions", path: "decisions", icon: Gavel },
  { navKey: "navTechnology", path: "technology", icon: Cog },

  { navKey: "navStates", path: "states", icon: MapPin },
];

type ToolsGroupId = "content" | "military" | "localisation" | "misc";
const TOOLS_GROUP_IDS: readonly ToolsGroupId[] = [
  "content",
  "military",
  "localisation",
  "misc",
];

const TOOLS_GROUP_DEFAULT_COLLAPSED: Record<ToolsGroupId, boolean> = {
  content: true,
  military: true,
  localisation: true,
  misc: true,
};
const toolsGroupStorageKey = (id: ToolsGroupId) => `sidebar-tools-group-${id}`;

const SIDEBAR_COLLAPSED_KEY = "sidebar-collapsed";
const SIDEBAR_COLLAPSED_EVENT = "hoi4-sidebar-collapsed-updated";
let sidebarCollapsedFallback = false;

function getSidebarCollapsedSnapshot(): boolean {
  try {
    return window.localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "1";
  } catch {
    return sidebarCollapsedFallback;
  }
}

function getServerSidebarCollapsedSnapshot(): boolean {
  return false;
}

function subscribeSidebarCollapsed(onStoreChange: () => void): () => void {
  const onStorage = (event: StorageEvent) => {
    if (event.key === SIDEBAR_COLLAPSED_KEY) onStoreChange();
  };
  window.addEventListener("storage", onStorage);
  window.addEventListener(SIDEBAR_COLLAPSED_EVENT, onStoreChange);
  return () => {
    window.removeEventListener("storage", onStorage);
    window.removeEventListener(SIDEBAR_COLLAPSED_EVENT, onStoreChange);
  };
}

function writeSidebarCollapsed(next: boolean): void {
  sidebarCollapsedFallback = next;
  try {
    window.localStorage.setItem(SIDEBAR_COLLAPSED_KEY, next ? "1" : "0");
  } catch {}
  window.dispatchEvent(new Event(SIDEBAR_COLLAPSED_EVENT));
}

function SectionLabel({
  label,
  collapsed,
  action,
}: {
  label: string;
  collapsed?: boolean;

  action?: React.ReactNode;
}) {
  if (collapsed) return <div className="mx-auto my-2 h-px w-6 bg-border" />;
  return (
    <div className="px-3 pt-4 pb-1 flex items-center justify-between gap-2">
      <span className="text-micro uppercase tracking-widest text-muted-foreground font-semibold select-none">
        {label}
      </span>
      {action}
    </div>
  );
}

function ToolsGroupHeader({
  label,
  collapsed,
  onToggle,
  sidebarCollapsed,
}: {
  label: string;
  collapsed: boolean;
  onToggle: () => void;

  sidebarCollapsed?: boolean;
}) {
  if (sidebarCollapsed) return null;
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-expanded={!collapsed}

      className={`w-full flex items-center justify-between px-3 py-1.5 rounded-md text-xs uppercase ui-interactive font-medium select-none border transition-colors ${
        collapsed
          ? "bg-sidebar-accent/20 border-transparent text-muted-foreground hover:bg-sidebar-accent/45 hover:text-foreground"
          : "bg-sidebar-accent/35 border-border/60 text-foreground hover:bg-sidebar-accent/55"
      }`}
    >
      <span className="truncate" title={label}>
        {label}
      </span>
      {collapsed ? (
        <ChevronRight className="w-3.5 h-3.5 shrink-0 opacity-70" aria-hidden />
      ) : (
        <ChevronDown className="w-3.5 h-3.5 shrink-0" aria-hidden />
      )}
    </button>
  );
}

export default function ProjectLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const t = useTranslations("layout");
  const tPolitics = useTranslations("politicsPanel");

  const vanillaName = useVanillaTagName();

  const tCommon = useTranslations("common");

  const tExport = useTranslations("export");
  const setConfirmMessage = useEditorDirtyStore((s) => s.setConfirmMessage);
  const params = useParams();
  const pathname = usePathname();
  const projectId = Number(params.id);

  const currentProject = useProjectStore((s) => s.currentProject);
  const fetchProject = useProjectStore((s) => s.fetchProject);
  const activeCountry = useActiveCountry();

  useLinkedCountry(projectId, currentProject?.id === projectId);

  useEffect(() => {
    window.name = workbenchWindowName(projectId);
  }, [projectId]);
  const countries = useProjectCountries();

  const pendingFolderProjectId = usePendingModFolderProjectId();

  const [hadPendingFolder, setHadPendingFolder] = useState(false);
  const [latchProjectId, setLatchProjectId] = useState(projectId);
  if (latchProjectId !== projectId) {
    setLatchProjectId(projectId);
    setHadPendingFolder(false);
  } else if (pendingFolderProjectId === projectId && !hadPendingFolder) {
    setHadPendingFolder(true);
  }
  const [addCountryOpen, setAddCountryOpen] = useState(false);

  const [nudgeDismissed, setNudgeDismissed] = useState(false);

  const sidebarCollapsed = useSyncExternalStore(
    subscribeSidebarCollapsed,
    getSidebarCollapsedSnapshot,
    getServerSidebarCollapsedSnapshot,
  );
  const toggleSidebar = useCallback(() => {
    writeSidebarCollapsed(!getSidebarCollapsedSnapshot());
  }, []);

  const openAddCountry = useCallback(() => {
    setAddCountryOpen(true);
  }, []);
  const [countrySettingsOpen, setCountrySettingsOpen] = useState(false);
  const [politicsSettingsOpen, setPoliticsSettingsOpen] = useState(false);

  const [projectSettingsOpen, setProjectSettingsOpen] = useState(false);

  const openSettingsToken = useOpenProjectSettings((s) => s.token);
  const openSettingsSection = useOpenProjectSettings((s) => s.section);
  const [handledSettingsToken, setHandledSettingsToken] = useState(0);
  if (openSettingsToken > 0 && openSettingsToken !== handledSettingsToken) {
    setHandledSettingsToken(openSettingsToken);

    const owner = dialogForSection(openSettingsSection);
    if (owner === "country") setCountrySettingsOpen(true);
    else if (owner === "politics") setPoliticsSettingsOpen(true);
    else setProjectSettingsOpen(true);
  }

  const prevPathnameRef = useRef<string | null>(null);
  useEffect(() => {
    if (
      prevPathnameRef.current !== null &&
      prevPathnameRef.current !== pathname
    ) {
      setCountrySettingsOpen(false);
      setPoliticsSettingsOpen(false);
    }
    prevPathnameRef.current = pathname;
  }, [pathname]);

  const [toolGroupCollapsed, setToolGroupCollapsed] = useState<
    Record<ToolsGroupId, boolean>
  >(TOOLS_GROUP_DEFAULT_COLLAPSED);
  useEffect(() => {
    try {
      const stored: Partial<Record<ToolsGroupId, boolean>> = {};
      for (const id of TOOLS_GROUP_IDS) {
        const raw = window.localStorage.getItem(toolsGroupStorageKey(id));
        if (raw === "1") stored[id] = true;
        else if (raw === "0") stored[id] = false;
      }
      if (Object.keys(stored).length > 0) {
        setToolGroupCollapsed((prev) => ({ ...prev, ...stored }));
      }
    } catch {}
  }, []);

  const expandedGroupCount = TOOLS_GROUP_IDS.filter(
    (id) => !toolGroupCollapsed[id],
  ).length;
  const collapseAllToolGroups = () => {
    for (const id of TOOLS_GROUP_IDS) {
      try {
        window.localStorage.setItem(toolsGroupStorageKey(id), "1");
      } catch {}
    }
    setToolGroupCollapsed(
      Object.fromEntries(TOOLS_GROUP_IDS.map((id) => [id, true])) as Record<
        ToolsGroupId,
        boolean
      >,
    );
  };

  const toggleToolGroup = (id: ToolsGroupId) => {
    const next = !toolGroupCollapsed[id];
    try {
      window.localStorage.setItem(toolsGroupStorageKey(id), next ? "1" : "0");
    } catch {}
    setToolGroupCollapsed((prev) => ({ ...prev, [id]: next }));
  };
  useEffect(() => {
    if (projectId) fetchProject(projectId);
  }, [projectId, fetchProject]);

  useBeforeUnloadGuard();
  useEffect(() => {
    setConfirmMessage(tCommon("discardUnsavedConfirm"));
  }, [tCommon, setConfirmMessage]);

  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prev;
    };
  }, []);

  const guardNav = useCallback((e: { preventDefault: () => void }) => {
    if (!confirmDiscardIfDirty()) e.preventDefault();
  }, []);

  const currentTab = pathname.split("/").pop() || "characters";
  const countryProfileActive = currentTab === "country-profile";
  const countrySettingsTitle = activeCountry
    ? `${t("countrySettingsTitle")} (${activeCountry})`
    : t("countrySettingsTitle");
  const countrySettingsLabel = activeCountry
    ? `${t("countrySettings")} (${activeCountry})`
    : t("countrySettings");

  const startEntryQuiet = !!currentProject?.is_loaded_existing_mod;

  const nonCountryEntityTotal = currentProject?.entity_counts
    ? Object.entries(currentProject.entity_counts).reduce(
        (acc, [key, n]) => (key === "country" ? acc : acc + (n || 0)),
        0,
      )
    : (currentProject?.character_count ?? 0) +
      (currentProject?.focus_node_count ?? 0) +
      (currentProject?.technology_count ?? 0);
  const showLoadModeBanner =
    !!currentProject?.is_loaded_existing_mod &&
    countries.length === 1 &&
    nonCountryEntityTotal === 0;

  const refreshAllStores = useCallback(async () => {
    if (!projectId) return;
    const refreshVisibleWorkbench = (() => {
      switch (currentTab) {
        case "characters":
          return useCharacterStore.getState().fetchCharacters(projectId);
        case "focus-tree":
          return useFocusTreeStore.getState().fetchTrees(projectId);
        case "technology":
          return useTechnologyStore.getState().fetchTechnologies(projectId);
        case "ideas":
          return useIdeaStore.getState().fetchIdeas(projectId);
        case "events":
          return useEventStore.getState().fetchNamespaces(projectId);
        case "decisions":
          return useDecisionStore.getState().fetchCategories(projectId);
        default:
          return Promise.resolve();
      }
    })();
    await Promise.all([refreshVisibleWorkbench, fetchProject(projectId)]);

    useFocusTreeStore.getState().notifyImported();
  }, [projectId, fetchProject, currentTab]);

  return (
    <div className="flex flex-col h-screen overflow-hidden">
      <header className="shrink-0 h-14 border-b border-border bg-sidebar flex items-center gap-2 px-5">
        <Link
          href="/"
          onNavigate={guardNav}
          className="ui-interactive shrink-0 text-micro uppercase tracking-[0.2em] text-muted-foreground hover:text-foreground flex items-center gap-1.5 font-semibold"
        >
          <span>←</span> Projects
        </Link>
        <span className="shrink-0 text-muted-foreground/50">/</span>
        <h1 className="min-w-0 max-w-[320px] truncate font-semibold text-sm tracking-tight">
          {currentProject?.name || "Loading..."}
        </h1>

        <button
          type="button"
          onClick={toggleSidebar}
          aria-label={
            sidebarCollapsed ? t("expandSidebar") : t("collapseSidebar")
          }
          title={sidebarCollapsed ? t("expandSidebar") : t("collapseSidebar")}
          aria-expanded={!sidebarCollapsed}
          className="ui-interactive ml-2 shrink-0 rounded-md p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          <PanelLeft className="h-4 w-4" aria-hidden />
        </button>

        {currentProject && countries.length === 0 && !nudgeDismissed && (
          <div className="mx-auto flex min-w-0 items-center gap-2 rounded-full border border-accent/40 bg-accent/10 px-3 py-1.5">
            <Info className="h-3.5 w-3.5 shrink-0 text-accent" aria-hidden />
            <p className="min-w-0 truncate text-xs text-muted-foreground">
              {t.rich("noCountryNudge", {
                b: (c) => (
                  <span className="font-semibold text-foreground">{c}</span>
                ),
              })}
            </p>
            <button
              type="button"
              onClick={() => setNudgeDismissed(true)}
              aria-label={t("dismissNudge")}
              title={t("dismissNudge")}
              className="ui-interactive shrink-0 rounded-full p-0.5 text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        )}

        <div className="ml-auto flex shrink-0 items-center gap-2">
          <ContentLanguageSwitcher />
          <LanguageSwitcher />
          <ThemeToggle />
          <Button size="sm" onClick={openAddCountry}>
            {t("addCountry")}
          </Button>

          {currentProject && (
            <Link
              data-tour-id="game-preview"

              href={
                countries.length === 0
                  ? gameViewHref(projectId, {
                      panel: "gamesetup_scenario_window",
                    })
                  : gameViewHref(projectId, {
                      tag: activeCountry || currentProject.country_tag,
                      panel: "countrypoliticsview",
                    })
              }
              target={gameViewWindowName(projectId)}
              className={cn(
                buttonVariants({ variant: "outline", size: "sm" }),
                "no-underline",
              )}
            >
              <Gamepad2 aria-hidden />
              {t("navGameView")}
              <ExternalLink aria-hidden />
            </Link>
          )}
        </div>
      </header>

      <div className="flex flex-1 min-h-0 h-full">
        <aside
          className={`border-r border-sidebar-border flex flex-col bg-sidebar shrink-0 min-h-0 h-full overflow-hidden py-4 transition-[width] duration-150 ${
            sidebarCollapsed ? "w-14 px-2" : "w-56 px-3"
          }`}
        >
          {sidebarCollapsed ? (
            currentProject ? (
              <SidebarItemTip
                collapsed
                label={
                  countries.length > 0 ? countrySettingsLabel : t("addCountry")
                }
              >
                {countries.length > 0 ? (
                  <Link
                    href={`/project/${projectId}/country-profile`}
                    onNavigate={guardNav}
                    title={countrySettingsTitle}
                    aria-label={countrySettingsLabel}

                    className={cn(
                      "ui-interactive mb-1 flex w-full items-center justify-center rounded-lg border py-2",
                      countryProfileActive
                        ? "border-accent/40 bg-accent/15 text-accent"
                        : "border-accent/25 bg-accent/5 text-accent hover:border-accent/40 hover:bg-accent/10",
                    )}
                  >
                    <PencilLine className="h-4 w-4" aria-hidden />
                  </Link>
                ) : (
                  <button
                    type="button"
                    onClick={openAddCountry}
                    aria-label={t("addCountry")}
                    className="ui-interactive mb-1 flex w-full items-center justify-center rounded-lg border border-dashed border-accent/50 bg-accent/10 py-2 text-accent hover:bg-accent/15"
                  >
                    <Plus className="h-4 w-4" aria-hidden />
                  </button>
                )}
              </SidebarItemTip>
            ) : null
          ) : (
            <div className="pb-2">
              <div className="min-h-8 flex items-center">
                {currentProject && countries.length === 0 && (
                  <button
                    onClick={openAddCountry}

                    className="w-full flex items-center justify-center gap-2.5 rounded-lg border border-dashed border-accent/50 bg-accent/10 px-3 py-2 text-sm font-semibold text-accent hover:bg-accent/15 hover:border-accent ui-interactive group"
                  >
                    <span>{t("addCountry")}</span>
                    <span className="opacity-0 group-hover:opacity-100 italic font-mono text-micro">
                      TAG
                    </span>
                  </button>
                )}

                {currentProject &&
                  countries.length === 1 &&
                  (() => {
                    const tag = currentProject.country_tag;
                    const v = vanillaName(tag);
                    const storedName = (
                      currentProject.country_name ?? ""
                    ).trim();
                    const displayName = isResolvedCountryName(storedName, tag)
                      ? storedName
                      : (v ?? "");
                    return (
                      <div className="flex w-full items-center justify-between gap-2 rounded-lg px-3 py-2">
                        <p className="flex min-w-0 items-center gap-1.5 truncate text-xs text-accent">
                          {v && <CountryFlag />}
                          <span className="font-mono">{tag}</span>
                          {displayName && (
                            <span className="truncate">
                              {" · " + displayName}
                            </span>
                          )}
                        </p>
                        <div className="flex shrink-0 items-center gap-0.5">
                          <Link
                            href={`/project/${projectId}/country-profile`}
                            onNavigate={guardNav}
                            className={cn(
                              "ui-interactive rounded-md border p-1",
                              countryProfileActive
                                ? "border-accent/40 bg-accent/15 text-accent"
                                : "border-accent/25 bg-accent/5 text-accent hover:border-accent/40 hover:bg-accent/10",
                            )}
                            title={countrySettingsTitle}
                            aria-label={countrySettingsLabel}
                          >
                            <PencilLine className="h-3.5 w-3.5" aria-hidden />
                          </Link>
                          <button
                            type="button"
                            onClick={openAddCountry}
                            className="ui-interactive rounded-md p-1 text-muted-foreground hover:bg-accent/10 hover:text-accent"
                            title={t("addCountryTitle")}
                            aria-label={t("addCountryTitle")}
                          >
                            <Plus className="h-3.5 w-3.5" aria-hidden />
                          </button>
                        </div>
                      </div>
                    );
                  })()}

                {currentProject && countries.length > 1 && (
                  <div className="flex w-full items-center gap-1">
                    <div className="min-w-0 flex-1">
                      <CountrySwitcher projectId={projectId} />
                    </div>
                    <Link
                      href={`/project/${projectId}/country-profile`}
                      onNavigate={guardNav}
                      className={cn(
                        "ui-interactive shrink-0 rounded-md border p-1.5",
                        countryProfileActive
                          ? "border-accent/40 bg-accent/15 text-accent"
                          : "border-accent/25 bg-accent/5 text-accent hover:border-accent/40 hover:bg-accent/10",
                      )}
                      title={countrySettingsTitle}
                      aria-label={countrySettingsLabel}
                    >
                      <PencilLine className="h-3.5 w-3.5" aria-hidden />
                    </Link>
                  </div>
                )}
              </div>

              {currentProject && countries.length > 1 && (
                <div className="mt-2">
                  <AllCountriesToggle />
                </div>
              )}
            </div>
          )}

          {sidebarCollapsed && currentProject && countries.length > 1 && (
            <div className="hidden">
              <CountrySwitcher projectId={projectId} />
            </div>
          )}

          {currentProject &&
            shouldMountModImport({
              countryCount: countries.length,
              hadPendingFolder,
              isLoadExistingMod: !!currentProject.is_loaded_existing_mod,
            }) && (
              <div className="py-1">
                <UnifiedModImport
                  projectId={projectId}
                  countryTag={activeCountry || currentProject.country_tag}
                  onComplete={refreshAllStores}
                  isLoadExistingMod={currentProject.is_loaded_existing_mod}
                  collapsed={sidebarCollapsed}
                />
              </div>
            )}

          <nav className="flex-1 min-h-0 overflow-y-auto overflow-x-hidden py-2 space-y-1">
            <SidebarItemTip collapsed={sidebarCollapsed} label={t("navStart")}>
              <Link
                href={`/project/${projectId}/start`}
                onNavigate={guardNav}

                className={`w-full flex items-center gap-2.5 py-2 rounded-lg text-sm ui-interactive text-left border ${
                  sidebarCollapsed ? "justify-center px-0" : "px-3"
                } ${
                  currentTab === "start"
                    ? "bg-accent/15 border-accent/35 text-foreground"
                    : startEntryQuiet
                      ? "border-transparent text-muted-foreground hover:text-foreground hover:bg-sidebar-accent/50"
                      : "border-transparent text-accent/90 hover:text-accent hover:bg-accent/10"
                }`}
              >
                <Compass
                  className="h-4 w-4 shrink-0"
                  strokeWidth={1.8}
                  aria-hidden
                />
                {!sidebarCollapsed && (
                  <span className="truncate">{t("navStart")}</span>
                )}
                {!sidebarCollapsed && <SidebarNavPending />}
              </Link>
            </SidebarItemTip>

            <SectionLabel
              label={t("sectionCreate")}
              collapsed={sidebarCollapsed}
            />
            {NAV_ITEMS.map((item) => {
              const Icon = item.icon;
              return (
                <Fragment key={item.path}>
                  <SidebarItemTip
                    collapsed={sidebarCollapsed}
                    label={t(item.navKey)}
                  >
                    <Link
                      href={`/project/${projectId}/${item.path}`}
                      onNavigate={guardNav}
                      className={`w-full flex items-center gap-2.5 py-2 rounded-lg text-sm ui-interactive text-left border ${
                        sidebarCollapsed ? "justify-center px-0" : "px-3"
                      } ${
                        currentTab === item.path
                          ? "bg-accent/15 border-accent/35 text-foreground"
                          : "border-transparent text-muted-foreground hover:bg-sidebar-accent/50 hover:text-foreground"
                      }`}
                    >
                      {Icon ? (
                        <Icon
                          className={`w-4 h-4 shrink-0 ${
                            currentTab === item.path ? "text-accent" : ""
                          }`}
                          strokeWidth={1.7}
                          aria-hidden
                        />
                      ) : (
                        <span className="w-4 shrink-0 text-center text-xs">
                          {item.emoji}
                        </span>
                      )}
                      {!sidebarCollapsed && t(item.navKey)}
                      {!sidebarCollapsed && <SidebarNavPending />}
                    </Link>
                  </SidebarItemTip>
                  {item.path === "characters" && (
                    <SidebarItemTip
                      collapsed={sidebarCollapsed}
                      label={tPolitics("title")}
                    >
                      <Link
                        href={`/project/${projectId}/country-politics`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 py-2 rounded-lg text-sm hover:text-foreground hover:bg-sidebar-accent/50 ui-interactive text-left border ${
                          sidebarCollapsed ? "justify-center px-0" : "px-3"
                        } ${
                          currentTab === "country-politics" ||
                          currentTab === "ideologies"
                            ? "bg-accent/15 border-accent/35 text-foreground"
                            : "border-transparent text-muted-foreground"
                        }`}
                        title={tPolitics("title")}
                      >
                        <Crown
                          className={`w-4 h-4 shrink-0 ${
                            currentTab === "country-politics" ||
                            currentTab === "ideologies"
                              ? "text-accent"
                              : ""
                          }`}
                          strokeWidth={1.7}
                          aria-hidden
                        />
                        {!sidebarCollapsed && <span>{tPolitics("title")}</span>}
                        {!sidebarCollapsed && <SidebarNavPending />}
                      </Link>
                    </SidebarItemTip>
                  )}
                </Fragment>
              );
            })}

            {currentProject && (
              <SectionLabel
                label={t("sectionDeepen")}
                collapsed={sidebarCollapsed}
                action={
                  !sidebarCollapsed && expandedGroupCount > 0 ? (
                    <button
                      type="button"
                      onClick={collapseAllToolGroups}
                      title={t("collapseAllGroupsTitle")}
                      className="ui-interactive shrink-0 inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-micro text-muted-foreground hover:bg-sidebar-accent/40 hover:text-foreground"
                    >
                      <ChevronsDownUp
                        className="w-3 h-3 shrink-0"
                        aria-hidden
                      />
                      {t("collapseAllGroups")}
                    </button>
                  ) : null
                }
              />
            )}
            {currentProject && (
              <div className="px-2 py-1">
                <div className="mt-1 space-y-0.5">
                  <ToolsGroupHeader
                    label={t("toolsGroupContent")}
                    collapsed={toolGroupCollapsed.content}
                    onToggle={() => toggleToolGroup("content")}
                    sidebarCollapsed={sidebarCollapsed}
                  />
                  {!toolGroupCollapsed.content && !sidebarCollapsed && (
                    <>
                      <Link
                        href={`/project/${projectId}/super-events`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "super-events"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("superEventsTitle")}
                      >
                        <Sparkles
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("superEvents")}</span>
                        <SidebarNavPending />
                      </Link>
                      <Link
                        href={`/project/${projectId}/bookmarks`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "bookmarks"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("bookmarksTitle")}
                      >
                        <BookOpen
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("bookmarks")}</span>
                        <SidebarNavPending />
                      </Link>
                      <Link
                        href={`/project/${projectId}/history-files`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "history-files"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("historyFilesTitle")}
                      >
                        <ScrollText
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("historyFiles")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/scripted-snippets`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "scripted-snippets"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("scriptedSnippetsTitle")}
                      >
                        <Puzzle
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("scriptedSnippets")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/ui-panels`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "ui-panels"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("uiPanelsTitle")}
                      >
                        <PanelsTopLeft
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("uiPanels")}</span>
                        <span className="ml-auto text-micro font-semibold uppercase tracking-widest text-accent">
                          {t("uiPanelsNew")}
                        </span>
                        <SidebarNavPending />
                      </Link>
                    </>
                  )}
                  <ToolsGroupHeader
                    label={t("toolsGroupMilitary")}
                    collapsed={toolGroupCollapsed.military}
                    onToggle={() => toggleToolGroup("military")}
                    sidebarCollapsed={sidebarCollapsed}
                  />
                  {!toolGroupCollapsed.military && !sidebarCollapsed && (
                    <>
                      <Link
                        href={`/project/${projectId}/mio`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "mio"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("mioTitle")}
                      >
                        <Factory
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("mio")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/equipment`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "equipment"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("equipmentTitle")}
                      >
                        <Swords
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("equipment")}</span>
                        <SidebarNavPending />
                      </Link>
                      <Link
                        href={`/project/${projectId}/sub-units`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "sub-units"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("subUnitsTitle")}
                      >
                        <Boxes
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("subUnits")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/division-templates`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "division-templates"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("divisionTemplatesTitle")}
                      >
                        <LayoutGrid
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("divisionTemplates")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/deployments`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "deployments"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("deploymentsTitle")}
                      >
                        <Send
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("deployments")}</span>
                        <SidebarNavPending />
                      </Link>
                      <Link
                        href={`/project/${projectId}/oob-units`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "oob-units"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("oobUnitsTitle")}
                      >
                        <ListTree
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("oobUnits")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/special-projects`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "special-projects"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("specialProjectsTitle")}
                      >
                        <FlaskConical
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("specialProjects")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/doctrine`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "doctrine"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("doctrineTitle")}
                      >
                        <GraduationCap
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("doctrine")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/intelligence`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "intelligence"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("intelligenceAgenciesTitle")}
                      >
                        <Eye
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("intelligenceAgencies")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/autonomy-states`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "autonomy-states"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("autonomyStatesTitle")}
                      >
                        <Crown
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("autonomyStates")}</span>
                        <SidebarNavPending />
                      </Link>
                    </>
                  )}
                  <ToolsGroupHeader
                    label={t("toolsGroupLocalisation")}
                    collapsed={toolGroupCollapsed.localisation}
                    onToggle={() => toggleToolGroup("localisation")}
                    sidebarCollapsed={sidebarCollapsed}
                  />
                  {!toolGroupCollapsed.localisation && !sidebarCollapsed && (
                    <>
                      <Link
                        href={`/project/${projectId}/scripted-localisations`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "scripted-localisations"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("scriptedLocalisationsTitle")}
                      >
                        <FileText
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("scriptedLocalisations")}</span>
                        <SidebarNavPending />
                      </Link>
                    </>
                  )}
                  <ToolsGroupHeader
                    label={t("toolsGroupMisc")}
                    collapsed={toolGroupCollapsed.misc}
                    onToggle={() => toggleToolGroup("misc")}
                    sidebarCollapsed={sidebarCollapsed}
                  />
                  {!toolGroupCollapsed.misc && !sidebarCollapsed && (
                    <>
                      <Link
                        href={`/project/${projectId}/dynamic-modifiers`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "dynamic-modifiers"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("dynamicModifiersTitle")}
                      >
                        <Dices
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("dynamicModifiers")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/modifier-definitions`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "modifier-definitions"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("modifierDefinitionsTitle")}
                      >
                        <FileText
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("modifierDefinitions")}</span>
                        <SidebarNavPending />
                      </Link>
                      <Link
                        href={`/project/${projectId}/ai-strategy-plans`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "ai-strategy-plans"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("aiStrategyPlansTitle")}
                      >
                        <ListTree
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("aiStrategyPlans")}</span>
                        <SidebarNavPending />
                      </Link>
                      <Link
                        href={`/project/${projectId}/ai-templates`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "ai-templates"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("aiTemplatesTitle")}
                      >
                        <Boxes
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("aiTemplates")}</span>
                        <SidebarNavPending />
                      </Link>
                      <Link
                        href={`/project/${projectId}/wargoals`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "wargoals"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("wargoalsTitle")}
                      >
                        <Swords
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("wargoals")}</span>
                        <SidebarNavPending />
                      </Link>

                      <Link
                        href={`/project/${projectId}/image-batch`}
                        onNavigate={guardNav}
                        className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm hover:bg-sidebar-accent/50 ui-interactive text-left ${
                          currentTab === "image-batch"
                            ? "bg-sidebar-accent text-sidebar-primary font-semibold"
                            : "text-muted-foreground hover:text-foreground"
                        }`}
                        title={t("imageBatchTitle")}
                      >
                        <ImageIcon
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("imageBatch")}</span>
                        <SidebarNavPending />
                      </Link>
                      <button
                        type="button"
                        onClick={() => setProjectSettingsOpen(true)}
                        className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm text-muted-foreground hover:text-foreground hover:bg-sidebar-accent/50 ui-interactive text-left"
                        title={t("modDescriptorTitle")}
                      >
                        <Settings
                          className="w-4 h-4 shrink-0"
                          strokeWidth={1.8}
                          aria-hidden
                        />
                        <span>{t("modDescriptor")}</span>
                      </button>
                    </>
                  )}
                </div>
              </div>
            )}
          </nav>

          <div className="shrink-0 py-2 border-t border-sidebar-border space-y-1">
            <SectionLabel
              label={t("sectionFinish")}
              collapsed={sidebarCollapsed}
            />
            <SidebarItemTip
              collapsed={sidebarCollapsed}
              label={t("checkMyMod")}
            >
              <Link
                href={`/project/${projectId}/export`}
                onNavigate={guardNav}
                className={`w-full flex items-center gap-2.5 py-2 rounded-lg text-sm text-muted-foreground hover:text-foreground hover:bg-sidebar-accent/50 ui-interactive text-left ${
                  sidebarCollapsed ? "justify-center px-0" : "px-3"
                }`}
                title={t("checkMyModTitle")}
              >
                <Check
                  className="h-4 w-4 shrink-0"
                  strokeWidth={1.7}
                  aria-hidden
                />
                {!sidebarCollapsed && (
                  <span className="truncate">{t("checkMyMod")}</span>
                )}
                {!sidebarCollapsed && <SidebarNavPending />}
              </Link>
            </SidebarItemTip>

            <SidebarItemTip
              collapsed={sidebarCollapsed}
              label={tExport("sidebarButton")}
            >
              <Link
                href={`/project/${projectId}/export`}
                onNavigate={guardNav}

                className={cn(
                  buttonVariants({ variant: "default" }),
                  "w-full h-auto justify-start gap-2.5 px-3 rounded-lg py-2 text-sm font-semibold",
                  currentTab === "export" && "ring-2 ring-ring ring-offset-1",
                )}
              >
                <Download className="h-4 w-4 shrink-0" />
                {!sidebarCollapsed && tExport("sidebarButton")}
              </Link>
            </SidebarItemTip>

            <Popover>
              <PopoverTrigger
                title={t("localSyncGroup")}
                aria-label={t("localSyncGroup")}
                className={`ui-interactive w-full flex items-center gap-2.5 py-2 rounded-lg text-sm text-muted-foreground hover:text-foreground hover:bg-sidebar-accent/50 text-left ${
                  sidebarCollapsed ? "justify-center px-0" : "px-3"
                }`}
              >
                <FolderSync
                  className="h-4 w-4 shrink-0"
                  strokeWidth={1.7}
                  aria-hidden
                />
                {!sidebarCollapsed && (
                  <span className="truncate">{t("localSync")}</span>
                )}
              </PopoverTrigger>

              <PopoverContent
                side="right"
                align="end"
                className="w-80 gap-1 p-2"
              >
                <PopoverTitle className="px-3 py-1.5 text-micro font-semibold uppercase tracking-widest text-muted-foreground">
                  {t("localSyncGroup")}
                </PopoverTitle>
                <FullModPreviewButton
                  projectId={projectId}
                  compact
                  variant="ghost"
                />
                <SaveFullModFolderButton
                  projectId={projectId}
                  compact
                  variant="ghost"
                />
              </PopoverContent>
            </Popover>
          </div>
        </aside>

        <main className="flex-1 min-w-0 min-h-0 h-full flex flex-col bg-background">
          {currentProject && <ProjectTopBar projectId={projectId} />}

          {showLoadModeBanner && (
            <div className="border-b border-accent/40 bg-sidebar-accent px-4 py-2 flex items-center justify-between gap-3 shrink-0">
              <p className="text-xs text-sidebar-primary">
                {t.rich("loadModeBanner", {
                  b: (c) => <span className="font-semibold">{c}</span>,
                })}
              </p>
              <span className="text-micro text-sidebar-primary/70 shrink-0 font-mono">
                {t("setEditTargets")}
              </span>
            </div>
          )}

          {currentProject &&
            currentProject.is_loaded_existing_mod &&
            countries.length > 0 &&
            (currentProject.character_count > 0 ||
              currentProject.focus_node_count > 0 ||
              currentProject.technology_count > 0) && (
              <div className="border-b border-border bg-card/30 px-4 py-2 flex items-center gap-3 shrink-0">
                <EditTargetIndicator
                  projectId={projectId}
                  modName={currentProject.name}
                  primaryTag={currentProject.country_tag}
                />
                <span className="text-micro text-muted-foreground italic">
                  {t.rich("appendNote", {
                    tag: currentProject.country_tag,
                    code: (c) => <span className="font-mono">{c}</span>,
                  })}
                </span>
              </div>
            )}

          <div className="grid flex-1 grid-rows-1 min-h-0">
            <AddCountryDialogContext.Provider value={openAddCountry}>
              {children}
            </AddCountryDialogContext.Provider>
          </div>
        </main>
      </div>

      <AddCountryDialog
        projectId={projectId}
        open={addCountryOpen}
        onOpenChange={setAddCountryOpen}
      />

      <CountrySettingsDialog
        projectId={projectId}
        open={countrySettingsOpen}
        onOpenChange={setCountrySettingsOpen}
        onManagePolitics={() => {
          setCountrySettingsOpen(false);
          setPoliticsSettingsOpen(true);
        }}
      />

      <PoliticsSettingsDialog
        projectId={projectId}
        open={politicsSettingsOpen}
        onOpenChange={setPoliticsSettingsOpen}
        onChanged={() => fetchProject(projectId)}
      />

      <ProjectSettingsDialog
        projectId={projectId}
        open={projectSettingsOpen}
        onOpenChange={setProjectSettingsOpen}
      />
    </div>
  );
}
