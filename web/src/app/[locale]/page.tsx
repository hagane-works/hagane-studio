"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import { useRouter } from "@/i18n/navigation";
import { useLocale, useTranslations } from "use-intl";
import { Upload, Plus, ArrowRight } from "lucide-react";
import { toast } from "sonner";
import { useProjectStore } from "@/lib/store/projectStore";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Select } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  InfoTooltip,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { LoadExistingModDialog } from "@/components/workbench/LoadExistingModDialog";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { LanguageSwitcher } from "@/components/i18n/LanguageSwitcher";
import {
  SiteHeader,
  siteHeaderStyles as siteHeader,
} from "@/components/site/SiteHeader";
import { ThemeToggle } from "@/components/i18n/ThemeToggle";
import { DashboardToolbar } from "@/components/dashboard/DashboardToolbar";
import {
  MOD_BASE_LABEL_KEYS,
  useDashboardFilters,
  type ModBaseBucket,
} from "@/lib/hooks/useDashboardFilters";
import {
  markRecentlyOpened,
  useRecentProjects,
  type RecentProjectView,
} from "@/lib/hooks/useRecentProjects";
import { useVanillaTagName } from "@/lib/vanillaCountryTags";
import type { Project } from "@/types";
import {
  DEFAULT_SUPPORTED_VERSION,
  HOI4_VERSION_OPTIONS,
} from "@/data/hoi4Versions";
import { DlcPicker } from "@/components/workbench/DlcPicker";
import { DesktopSettingsButton } from "@/desktop/DesktopSettings";

const ENTITY_TYPE_DISPLAY_ORDER = [
  "country",
  "character",
  "custom_trait",
  "opinion_modifier",
  "dynamic_modifier",
  "modifier_definition",
  "ai_strategy_plan",
  "ai_template",
  "wargoal",
  "idea",
  "focus_tree",
  "focus_node",
  "technology",
  "event_namespace",
  "event",
  "super_event",
  "decision_category",
  "decision",
  "bookmark",
  "mio",
  "history_file",
] as const;

const KNOWN_ENTITY_TYPES: ReadonlySet<string> = new Set(
  ENTITY_TYPE_DISPLAY_ORDER,
);

export default function HomePage() {
  return (
    <Suspense fallback={null}>
      <LoggedInDashboard />
    </Suspense>
  );
}

function LoggedInDashboard() {
  const router = useRouter();
  const t = useTranslations("landing");
  const locale = useLocale();
  const confirmDialog = useConfirm();

  const projects = useProjectStore((s) => s.projects);
  const loading = useProjectStore((s) => s.loading);
  const fetchProjects = useProjectStore((s) => s.fetchProjects);
  const deleteProject = useProjectStore((s) => s.deleteProject);

  const [dialogOpen, setDialogOpen] = useState(false);

  const [loadModDialogOpen, setLoadModDialogOpen] = useState(false);
  const [relativeTimeReference, setRelativeTimeReference] = useState<
    number | null
  >(null);

  useEffect(() => {
    let cancelled = false;
    void fetchProjects().finally(() => {
      if (!cancelled) setRelativeTimeReference(Date.now());
    });
    return () => {
      cancelled = true;
    };
  }, [fetchProjects]);

  const handleDelete = async (id: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (
      !(await confirmDialog({ body: t("confirmDeleteProject"), danger: true }))
    )
      return;
    try {
      await deleteProject(id);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      toast.error(t("deleteFailed", { msg }));
      fetchProjects();
    }
  };

  const openProject = (id: number) => {
    markRecentlyOpened(id);
    const p = projects.find((x) => x.id === id);
    const counts = p?.entity_counts;
    const isEmptyFromScratch =
      !!p &&
      !p.is_loaded_existing_mod &&
      !!counts &&
      Object.values(counts).reduce((acc, n) => acc + (n || 0), 0) === 0;

    router.push(
      `/project/${id}/${isEmptyFromScratch ? "start" : "characters"}`,
    );
  };

  const stats = useMemo(() => {
    const projectCount = projects.length;
    const allTags = new Set<string>();
    let entityCount = 0;
    const entityBreakdownMap = new Map<string, number>();
    let latestProject: (typeof projects)[number] | null = null;
    let latestTs = -Infinity;
    let anyEntityCountsDict = false;
    for (const p of projects) {
      for (const tag of p.country_tags) {
        if (tag) allTags.add(tag);
      }

      if (p.entity_counts) {
        anyEntityCountsDict = true;
        for (const [key, count] of Object.entries(p.entity_counts)) {
          if (typeof count !== "number" || count <= 0) continue;
          entityCount += count;
          entityBreakdownMap.set(
            key,
            (entityBreakdownMap.get(key) ?? 0) + count,
          );
        }
      } else {
        entityCount +=
          (p.character_count || 0) +
          (p.focus_node_count || 0) +
          (p.technology_count || 0);
      }
      const ts = new Date(p.updated_at).getTime();
      if (Number.isFinite(ts) && ts > latestTs) {
        latestTs = ts;
        latestProject = p;
      }
    }

    const known = ENTITY_TYPE_DISPLAY_ORDER.filter((k) =>
      entityBreakdownMap.has(k),
    );
    const unknown = Array.from(entityBreakdownMap.keys())
      .filter(
        (k) => !(ENTITY_TYPE_DISPLAY_ORDER as readonly string[]).includes(k),
      )
      .sort();
    const entityBreakdown = [...known, ...unknown].map((key) => ({
      key,
      count: entityBreakdownMap.get(key) ?? 0,
    }));
    return {
      projectCount,
      countryCount: allTags.size,
      entityCount,
      entityBreakdown,

      hasEntityBreakdown: anyEntityCountsDict && entityBreakdown.length > 0,
      latestProject,
      latestTs: Number.isFinite(latestTs) ? latestTs : null,
    };
  }, [projects]);

  const formatStatRelativeTime = (ts: number | null): string => {
    if (ts == null) return t("statRecentNone");
    const diffMs = Math.max(0, (relativeTimeReference ?? ts) - ts);
    const diffMin = Math.floor(diffMs / 60000);
    if (diffMin < 1) return t("timeJustNow");
    if (diffMin < 60) return t("timeMinutesAgo", { n: diffMin });
    const diffHr = Math.floor(diffMin / 60);
    if (diffHr < 24) return t("timeHoursAgo", { n: diffHr });
    const diffDay = Math.floor(diffHr / 24);
    if (diffDay === 1) return t("timeYesterday");
    if (diffDay < 7) return t("timeDaysAgo", { n: diffDay });
    return new Date(ts).toLocaleDateString();
  };

  return (
    <TooltipProvider>
      <div className="flex flex-col min-h-screen">
        <SiteHeader
          brandHref={`/${locale}`}
          brandLabel={t("public.brandHome")}
          navLabel={t("dashboardNavigationLabel")}
          containerClassName="mx-auto w-[min(72rem,calc(100%-3rem))]"
          nav={null}
          tools={
            <>
              <Suspense
                fallback={<span className="h-11 w-20" aria-hidden="true" />}
              >
                <LanguageSwitcher
                  appearance="public"
                  className={siteHeader.languageControl}
                />
              </Suspense>
              <ThemeToggle className={siteHeader.themeControl} />
              <DesktopSettingsButton
                className={`${siteHeader.action} ui-interactive ui-focus-ring`}
              />
            </>
          }
        />

        <LoadExistingModDialog
          open={loadModDialogOpen}
          onOpenChange={setLoadModDialogOpen}
        />

        <NewProjectDialog open={dialogOpen} onOpenChange={setDialogOpen} />

        <div id="dashboard-projects" tabIndex={-1} className="outline-none" />

        {!loading && projects.length > 0 && (
          <section className="px-6 pt-8">
            <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4">
              <h1 className="text-3xl font-medium">{t("dashboardTitle")}</h1>
              <div className="flex w-full flex-wrap gap-3 sm:w-auto">
                <Button
                  className="h-11 flex-1 px-4 sm:flex-none"
                  onClick={() => setLoadModDialogOpen(true)}
                >
                  {t("ctaLoadExisting")}
                </Button>
                <Button
                  variant="outline"
                  className="h-11 flex-1 px-4 sm:flex-none"
                  onClick={() => setDialogOpen(true)}
                >
                  {t("ctaNewProject")}
                </Button>
              </div>
            </div>
          </section>
        )}

        {!loading && projects.length > 0 && (
          <section className="px-6 pt-6 pb-2">
            <div className="max-w-6xl mx-auto grid grid-cols-2 md:grid-cols-4 gap-3">
              <Card className="p-3">
                <div className="text-micro uppercase tracking-wider text-muted-foreground">
                  {t("statProjects")}
                </div>
                <div className="text-lg font-semibold mt-0.5">
                  {stats.projectCount}
                </div>
              </Card>
              <Card className="p-3">
                <div className="text-micro uppercase tracking-wider text-muted-foreground">
                  {t("statCountries")}
                </div>
                <div className="text-lg font-semibold mt-0.5">
                  {stats.countryCount}
                </div>
              </Card>

              {stats.hasEntityBreakdown ? (
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Card className="p-3 cursor-help">
                        <div className="text-micro uppercase tracking-wider text-muted-foreground">
                          {t("statEntities")}
                        </div>
                        <div className="text-lg font-semibold mt-0.5">
                          {stats.entityCount.toLocaleString()}
                        </div>
                      </Card>
                    }
                  />
                  <TooltipContent side="bottom" className="max-w-xs">
                    <div className="font-medium mb-1.5">
                      {t("statEntitiesTooltipTitle")}
                    </div>
                    <ul className="space-y-0.5 text-muted-foreground">
                      {stats.entityBreakdown.map(({ key, count }) => (
                        <li
                          key={key}
                          className="flex items-center justify-between gap-3 font-mono tabular-nums"
                        >
                          <span className="truncate">
                            {KNOWN_ENTITY_TYPES.has(key)
                              ? t(`entityType.${key}` as never)
                              : key}
                          </span>
                          <span>{count.toLocaleString()}</span>
                        </li>
                      ))}
                    </ul>
                  </TooltipContent>
                </Tooltip>
              ) : (
                <Card className="p-3">
                  <div className="text-micro uppercase tracking-wider text-muted-foreground">
                    {t("statEntities")}
                  </div>
                  <div className="text-lg font-semibold mt-0.5">
                    {stats.entityCount.toLocaleString()}
                  </div>
                </Card>
              )}
              <Card className="p-3">
                <div className="text-micro uppercase tracking-wider text-muted-foreground">
                  {t("statRecentEdit")}
                </div>
                <div className="text-sm font-semibold mt-1">
                  {formatStatRelativeTime(stats.latestTs)}
                </div>
                {stats.latestProject && (
                  <div className="text-xs text-muted-foreground truncate">
                    {stats.latestProject.name}
                  </div>
                )}
              </Card>
            </div>
          </section>
        )}

        <main className="flex-1 px-6 py-8">
          <div className="max-w-6xl mx-auto">
            {loading ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {Array.from({ length: 6 }).map((_, i) => (
                  <Card key={`skel-${i}`} className="p-5">
                    <div className="space-y-3">
                      <div className="flex items-start justify-between">
                        <div className="space-y-2 flex-1">
                          <div className="h-5 w-2/3 rounded bg-muted/60 animate-pulse" />
                          <div className="h-3 w-1/3 rounded bg-muted/40 animate-pulse" />
                        </div>
                      </div>
                      <div className="flex gap-2 flex-wrap">
                        <div className="h-5 w-20 rounded bg-muted/40 animate-pulse" />
                        <div className="h-5 w-24 rounded bg-muted/40 animate-pulse" />
                        <div className="h-5 w-16 rounded bg-muted/40 animate-pulse" />
                      </div>
                    </div>
                  </Card>
                ))}
              </div>
            ) : projects.length === 0 ? (
              <div className="py-12">
                <div className="max-w-3xl mx-auto text-center mb-10">
                  <h2 className="text-2xl font-semibold mb-3">
                    {t("emptyHero.title")}
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    {t("emptyHero.subtitle")}
                  </p>
                </div>

                <div className="max-w-3xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-4">
                  <button
                    type="button"
                    onClick={() => setLoadModDialogOpen(true)}
                    className="group flex flex-col p-6 rounded-xl border border-border bg-card text-left hover:border-primary/50 hover:bg-muted/30 ui-interactive"
                    data-testid="empty-hero-load"
                  >
                    <div className="w-12 h-12 rounded-lg bg-muted text-foreground/80 flex items-center justify-center mb-4 border border-border">
                      <Upload className="w-6 h-6" aria-hidden />
                    </div>
                    <h3 className="font-semibold text-base mb-1">
                      {t("emptyHero.load.title")}
                    </h3>
                    <p className="text-xs text-muted-foreground mb-4 flex-1">
                      {t("emptyHero.load.blurb")}
                    </p>
                    <div className="inline-flex items-center gap-1 text-xs text-foreground/80 font-medium group-hover:text-primary">
                      {t("emptyHero.load.cta")}
                      <ArrowRight className="w-3 h-3" aria-hidden />
                    </div>
                  </button>

                  <button
                    type="button"
                    onClick={() => setDialogOpen(true)}
                    className="group flex flex-col p-6 rounded-xl border border-border bg-card text-left hover:border-primary/50 hover:bg-muted/30 ui-interactive"
                    data-testid="empty-hero-new"
                  >
                    <div className="w-12 h-12 rounded-lg bg-muted text-muted-foreground flex items-center justify-center mb-4 border border-border">
                      <Plus className="w-6 h-6" aria-hidden />
                    </div>
                    <h3 className="font-semibold text-base mb-1">
                      {t("emptyHero.scratch.title")}
                    </h3>
                    <p className="text-xs text-muted-foreground mb-4 flex-1">
                      {t("emptyHero.scratch.blurb")}
                    </p>
                    <div className="inline-flex items-center gap-1 text-xs text-foreground/80 font-medium group-hover:text-primary">
                      {t("emptyHero.scratch.cta")}
                      <ArrowRight className="w-3 h-3" aria-hidden />
                    </div>
                  </button>
                </div>
              </div>
            ) : (
              <>
                <RecentOpenedSection projects={projects} onOpen={openProject} />

                <Suspense fallback={<ProjectGridFallback />}>
                  <ProjectGridWithToolbar
                    projects={projects}
                    onOpen={openProject}
                    onDelete={handleDelete}
                    onNewProject={() => setDialogOpen(true)}
                    onImportMod={() => setLoadModDialogOpen(true)}
                  />
                </Suspense>
              </>
            )}
          </div>
        </main>
      </div>
    </TooltipProvider>
  );
}

function NewProjectDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations("landing");
  const router = useRouter();
  const createProject = useProjectStore((s) => s.createProject);

  const [modName, setModName] = useState("");

  const [supportedVersion, setSupportedVersion] = useState(
    DEFAULT_SUPPORTED_VERSION,
  );

  const [ownedDlcs, setOwnedDlcs] = useState<string[] | null>(null);
  const [creating, setCreating] = useState(false);

  const handleCreate = async () => {
    if (!modName.trim() || creating) return;
    setCreating(true);
    try {
      const project = await createProject({
        name: modName.trim(),
        supported_version: supportedVersion,
        owned_dlcs: ownedDlcs,
      });
      onOpenChange(false);
      setModName("");
      setSupportedVersion(DEFAULT_SUPPORTED_VERSION);
      setOwnedDlcs(null);

      router.push(`/project/${project.id}/start`);
    } catch (err) {
      toast.error(
        t("baseModShelf.createFailed", {
          msg: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      setCreating(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("baseModShelf.dialogTitle")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-5 pt-1">
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t("modNameLabel")}
              <InfoTooltip
                title={t("modNamingTitle")}
                content={t.raw("modNamingContent")}
              />
            </Label>
            <Input
              placeholder={t("modNamePlaceholder")}
              value={modName}
              onChange={(e) => setModName(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">{t("modNameHint")}</p>
          </div>

          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t("supportedVersionLabel")}
              <InfoTooltip
                title={t("supportedVersionLabel")}
                content={t.raw("supportedVersionHint")}
              />
            </Label>
            <Select
              value={supportedVersion}
              onChange={(e) => setSupportedVersion(e.target.value)}
              className="px-2 font-mono"
            >
              {HOI4_VERSION_OPTIONS.map((v) => (
                <option key={v} value={v}>
                  {v}
                  {v === DEFAULT_SUPPORTED_VERSION
                    ? ` · ${t("supportedVersionDefault")}`
                    : ""}
                </option>
              ))}
            </Select>
          </div>

          <DlcPicker
            value={ownedDlcs}
            onChange={setOwnedDlcs}
            disabled={creating}
          />

          <Button
            className="w-full"
            onClick={handleCreate}
            disabled={!modName.trim() || creating}
          >
            {creating ? t("creating") : t("createProject")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ProjectGridFallback() {
  const t = useTranslations("landing");
  return (
    <p className="text-center text-muted-foreground py-20">
      {t("gridLoading")}
    </p>
  );
}

interface ProjectGridWithToolbarProps {
  projects: Project[];
  onOpen: (id: number) => void;
  onDelete: (id: number, e: React.MouseEvent) => void;

  onNewProject: () => void;
  onImportMod: () => void;
}

function ProjectGridWithToolbar({
  projects,
  onOpen,
  onDelete,
  onNewProject,
  onImportMod,
}: ProjectGridWithToolbarProps) {
  const t = useTranslations("landing");
  const {
    q,
    sort,
    filter,
    setQ,
    setSort,
    setFilter,
    grouped,
    totalVisible,
    isFiltered,
  } = useDashboardFilters(projects);

  return (
    <>
      <DashboardToolbar
        q={q}
        sort={sort}
        filter={filter}
        onQChange={setQ}
        onSortChange={setSort}
        onFilterChange={setFilter}
      />
      {totalVisible === 0 ? (
        <div className="text-center py-16">
          <p className="text-muted-foreground mb-3">
            {isFiltered ? t("toolbar.noMatch") : t("emptyState")}
          </p>
          {isFiltered && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setQ("");
                setFilter("all");
                setSort("recent");
              }}
            >
              {t("toolbar.clearFilters")}
            </Button>
          )}
        </div>
      ) : (
        <div className="space-y-8">
          {grouped.map((group, gi) => (
            <section key={group.bucket}>
              {group.bucket !== "all" && (
                <ModBaseGroupHeader
                  bucket={group.bucket}
                  count={group.projects.length}
                />
              )}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {group.projects.map((project) => (
                  <ProjectCard
                    key={project.id}
                    project={project}
                    onOpen={onOpen}
                    onDelete={onDelete}
                  />
                ))}

                {gi === grouped.length - 1 && !isFiltered && (
                  <>
                    <GhostActionCard
                      icon={<Plus className="h-5 w-5" aria-hidden />}
                      label={t("ctaNewProject")}
                      onClick={onNewProject}
                    />
                    <GhostActionCard
                      icon={<Upload className="h-5 w-5" aria-hidden />}
                      label={t("ctaLoadExisting")}
                      onClick={onImportMod}
                    />
                  </>
                )}
              </div>
            </section>
          ))}
        </div>
      )}
    </>
  );
}

function GhostActionCard({
  icon,
  label,
  onClick,
}: {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="ui-interactive flex h-full min-h-32 flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-muted/10 p-5 text-muted-foreground hover:border-primary/50 hover:bg-muted/30 hover:text-foreground"
    >
      <span className="text-primary">{icon}</span>
      <span className="text-sm font-medium">{label}</span>
    </button>
  );
}

function ModBaseGroupHeader({
  bucket,
  count,
}: {
  bucket: ModBaseBucket;
  count: number;
}) {
  const t = useTranslations("landing");
  const label = t(MOD_BASE_LABEL_KEYS[bucket]);
  return (
    <div className="flex items-center gap-3 mb-3">
      <span className="text-xs uppercase tracking-wide font-semibold text-primary">
        {label} <span className="text-muted-foreground">({count})</span>
      </span>
      <div className="h-px flex-1 bg-border/50" />
    </div>
  );
}

function isUnresolvedCountryName(name: string, tag: string): boolean {
  const n = name.trim().toUpperCase();
  const tg = tag.trim().toUpperCase();
  if (!n || !tg) return true;
  if (n === tg) return true;
  return n.startsWith(tg) && /^\d+$/.test(n.slice(tg.length));
}

function ProjectCard({
  project,
  onOpen,
  onDelete,
}: {
  project: Project;
  onOpen: (id: number) => void;
  onDelete: (id: number, e: React.MouseEvent) => void;
}) {
  const t = useTranslations("landing");
  const vanillaName = useVanillaTagName();

  const storedName = project.country_name.trim();
  const countryDisplayName =
    storedName && !isUnresolvedCountryName(storedName, project.country_tag)
      ? storedName
      : vanillaName(project.country_tag);
  return (
    <Card
      className="p-5 cursor-pointer hover:border-primary/50 ui-interactive group"
      onClick={() => onOpen(project.id)}
    >
      <div className="flex items-start justify-between mb-3">
        <div>
          <h3 className="font-semibold text-lg flex items-center gap-2">
            {project.name}
            {project.is_loaded_existing_mod && (
              <Badge variant="outline" className="text-micro font-normal">
                {t("loadedBadge")}
              </Badge>
            )}
          </h3>
          {project.country_tags.length === 0 ? (
            <p className="text-sm text-muted-foreground italic">
              {t("cardNoCountries")}
            </p>
          ) : project.country_tags.length === 1 ? (
            <p className="text-sm text-muted-foreground flex items-center gap-1">
              <span className="font-mono">{project.country_tag}</span>
              {countryDisplayName && <span>{" - " + countryDisplayName}</span>}
            </p>
          ) : (
            <p className="text-sm text-muted-foreground">
              <span className="font-mono">{project.country_tag}</span>
              <span className="text-xs ml-1">
                {t("cardMore", { n: project.country_tags.length - 1 })}
              </span>
            </p>
          )}
        </div>
        <button
          className="text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 ui-interactive text-sm"
          onClick={(e) => onDelete(project.id, e)}
        >
          {t("cardDelete")}
        </button>
      </div>
      <div className="flex gap-2 flex-wrap">
        <Badge variant="secondary">
          {t("cardCharacters", { n: project.character_count })}
        </Badge>
        <Badge variant="secondary">
          {t("cardFocusNodes", { n: project.focus_node_count })}
        </Badge>
        <Badge variant="secondary">
          {t("cardTechnologies", { n: project.technology_count })}
        </Badge>
      </div>
      <p className="text-xs text-muted-foreground mt-3">
        {t("cardUpdated", {
          date: new Date(project.updated_at).toLocaleDateString(),
        })}
      </p>
    </Card>
  );
}

function RecentOpenedSection({
  projects,
  onOpen,
}: {
  projects: Project[];
  onOpen: (projectId: number) => void;
}) {
  const t = useTranslations("landing");
  const recent = useRecentProjects(projects, 3);

  if (recent.length === 0) return null;

  return (
    <section className="mb-8">
      <h2 className="text-xs uppercase tracking-wider text-muted-foreground mb-3 font-semibold">
        {t("recentOpened")}
      </h2>

      <div className="space-y-3">
        {recent.map((view, i) => (
          <RecentRow
            key={view.project.id}
            view={view}
            onOpen={onOpen}
            isCurrent={i === 0}
          />
        ))}
      </div>
    </section>
  );
}

function RecentRow({
  view,
  onOpen,
  isCurrent,
}: {
  view: RecentProjectView;
  onOpen: (projectId: number) => void;

  isCurrent: boolean;
}) {
  const t = useTranslations("landing");
  const { project, last_opened_at } = view;

  const ageMs = Date.now() - new Date(last_opened_at).getTime();
  const showEditingNow = isCurrent && ageMs / (1000 * 60 * 60) < 24;

  const chips = project.country_tags.slice(0, 5);
  const extraCountries = Math.max(
    0,
    project.country_tags.length - chips.length,
  );

  const accentClass = isCurrent
    ? "border-primary/60 ring-1 ring-primary/30"
    : "hover:border-primary/50";

  return (
    <Card
      className={`ui-interactive group flex cursor-pointer flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between ${accentClass}`}
      onClick={() => onOpen(project.id)}
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          {showEditingNow && (
            <span className="flex items-center gap-1.5">
              <span className="relative flex h-1.5 w-1.5">
                <span className="absolute inline-flex h-full w-full rounded-full bg-primary opacity-75 animate-ping" />
                <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-primary" />
              </span>
              <span className="text-micro font-semibold uppercase tracking-wide text-primary">
                {t("editingNow")}
              </span>
            </span>
          )}
          <span className="text-xs text-muted-foreground">
            {showEditingNow && "· "}
            {formatRecentRelativeTime(ageMs, t)}
          </span>
        </div>

        <h3 className="mt-1.5 flex items-center gap-2 truncate text-lg font-semibold tracking-tight">
          {project.name}
          {project.is_loaded_existing_mod && (
            <Badge variant="outline" className="text-micro font-normal">
              {t("loadedBadge")}
            </Badge>
          )}
        </h3>

        <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs text-muted-foreground">
          {chips.length === 0 ? (
            <span className="italic">{t("noCountries")}</span>
          ) : (
            <span className="flex flex-wrap items-center gap-1.5">
              {chips.map((tag) => (
                <span
                  key={tag}
                  className="rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-micro"
                >
                  {tag}
                </span>
              ))}
              {extraCountries > 0 && (
                <span className="text-micro">
                  {t("moreCountries", { n: extraCountries })}
                </span>
              )}
            </span>
          )}
          <span className="text-border">|</span>
          <span>{t("entityCharacters", { n: project.character_count })}</span>
          <span>·</span>
          <span>{t("entityFocusNodes", { n: project.focus_node_count })}</span>
          <span>·</span>
          <span>
            {t("entityTechnologies", { n: project.technology_count })}
          </span>
        </div>
      </div>

      <Button
        variant={isCurrent ? "default" : "outline"}
        className="shrink-0 sm:w-auto"
        onClick={(e) => {
          e.stopPropagation();
          onOpen(project.id);
        }}
      >
        {t("continueEditing")}
      </Button>
    </Card>
  );
}

function formatRecentRelativeTime(
  ageMs: number,
  t: (key: string, values?: Record<string, number>) => string,
): string {
  const seconds = Math.floor(ageMs / 1000);
  if (seconds < 60) return t("justNow");
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return t("minAgo", { n: minutes });
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return t("hourAgo", { n: hours });
  const days = Math.floor(hours / 24);
  return t("dayAgo", { n: days });
}
