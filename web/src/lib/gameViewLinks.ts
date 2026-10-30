export interface GameViewTarget {
  tag?: string | null;

  panel?: string | null;

  event?: string | null;

  state?: number | null;

  tree?: string | null;

  superEvent?: number | null;

  template?: string | null;

  mapStudio?: boolean;
}

export function gameViewWindowName(projectId: number): string {
  return `hoi4-game-view-${projectId}`;
}

export function workbenchWindowName(projectId: number): string {
  return `hoi4-workbench-${projectId}`;
}

export function gameViewTargetOf(href: string): string {
  const id = Number(
    new URLSearchParams(href.split("?")[1] ?? "").get("project_id"),
  );
  return gameViewWindowName(Number.isFinite(id) ? id : 0);
}

export function workbenchTargetOf(href: string): string {
  const id = Number(/\/project\/(\d+)/.exec(href)?.[1]);
  return workbenchWindowName(Number.isFinite(id) ? id : 0);
}

export function gameViewHref(
  projectId: number,
  target: GameViewTarget = {},
): string {
  const query = new URLSearchParams({ project_id: String(projectId) });
  if (target.tag?.trim()) query.set("tag", target.tag.trim().toUpperCase());
  if (target.panel) query.set("panel", target.panel);
  if (target.event) query.set("event", target.event);
  if (target.state) query.set("state", String(target.state));
  if (target.tree) query.set("tree", target.tree);
  if (target.superEvent) query.set("super_event", String(target.superEvent));
  if (target.template) query.set("template", target.template);
  if (target.mapStudio) query.set("mapstudio", "1");
  return `/game-view?${query}`;
}

const WORKBENCH_OF_SCREEN: Readonly<Record<string, string>> = {
  countrypoliticsview: "country-politics",
  political_ideas_window: "ideas",
  nationalfocusview: "focus-tree",
  countrytechnologyview: "technology",
  countrytechtreeview: "technology",
  facilities_view: "special-projects",
  program_window: "special-projects",
  scientist_roster_window: "special-projects",
  countrydecisionview: "decisions",
  EventWindow: "events",
  countrystateview: "states",
  countrydeploymentview: "deployments",
  templatedeploymentwindow: "deployments",
  countrydivisiondesignerview: "division-templates",
  armyhqtemplatewindow: "division-templates",
  countryproductionlineview: "equipment",
  tank_designer_view: "equipment",
  plane_designer_view: "equipment",
  countryequipmentdesignerview: "equipment",
  countryofficercorpview: "characters",
  armyleaderwindow: "characters",
  navyleaderwindow: "characters",
  unitleader_trait_tree_window: "characters",
  industrial_organisation_list_window: "mio",
  industrial_organisation_detail_window: "mio",
  countryintelligenceagencyview: "intelligence",
  intelligence_agency_upgrades_window: "intelligence",
  name_logo_agency_selection_window: "intelligence",
  country_doctrine_view: "doctrine",
  countryarmyview: "oob-units",
  countrydiplomacyview: "wargoals",
  diplomacy_action_info_window: "wargoals",
  manage_subjects_window: "autonomy-states",
  gamesetup_scenario_window: "bookmarks",
  gamesetup_interesting_countries_window: "bookmarks",
};

export function workbenchHrefForScreen(
  projectId: number,
  screen: string | null | undefined,
): string {
  const name = screen ?? "";
  const slug = name.endsWith("_SE_main_container")
    ? "super-events"
    : WORKBENCH_OF_SCREEN[name];
  return slug ? `/project/${projectId}/${slug}` : `/project/${projectId}/start`;
}

export const WORKBENCH_COUNTRY_PARAM = "country";

export function withWorkbenchCountry(
  href: string,
  tag: string | null | undefined,
): string {
  const country = tag?.trim().toUpperCase();
  if (!country) return href;
  const hashAt = href.indexOf("#");
  const hash = hashAt >= 0 ? href.slice(hashAt) : "";
  const base = hashAt >= 0 ? href.slice(0, hashAt) : href;
  const queryAt = base.indexOf("?");
  const params = new URLSearchParams(
    queryAt >= 0 ? base.slice(queryAt + 1) : "",
  );
  params.set(WORKBENCH_COUNTRY_PARAM, country);
  return `${queryAt >= 0 ? base.slice(0, queryAt) : base}?${params}${hash}`;
}
