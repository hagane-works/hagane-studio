import type { GuiNode } from "@/lib/guiRenderApi";
import { anchorCell } from "../gui-renderer/GuiCanvas";

export const HAS_INTERNATIONAL_MARKET = true;

export const TOPBAR_TARGETS: Readonly<Record<string, string>> = {
  player_flag: "countrypoliticsview",
  decisionview_button: "countrydecisionview",
  technology_button: "countrytechnologyview",

  diplomacy_button: HAS_INTERNATIONAL_MARKET
    ? "countryinternationalmarketview"
    : "countrydiplomacyview",
  trade_button: "countrytradeview",
  construction_button: "countryconstructionsview",
  production_button: "countryproductionlineview",
  deployment_button: "countrydeploymentview",
  logistics_button: "countrylogisticsview",
  officer_corp_button: "countryofficercorpview",
  intel_agency_button: "countryintelligenceagencyview",
  faction_button: "countryfactionview",

  army_button: "countryarmyview",
  navy_button: "countryarmyview",
  air_button: "countryarmyview",

  achievements_button: "achievements_window",
};

export const TOPBAR_SPRITE_SWAPS: Readonly<Record<string, string>> =
  HAS_INTERNATIONAL_MARKET
    ? { diplomacy_button: "GFX_topbar_international_market" }
    : {};

export const UNLIT_WHEN_OPEN: ReadonlySet<string> = new Set([
  "technology_button",
  "intel_agency_button",
]);

export function topbarButtonFor(
  panel: string | null,
  openedBy: string | null = null,
): string | null {
  if (!panel) return null;
  let button: string | null = null;
  if (openedBy && TOPBAR_TARGETS[openedBy] === panel) button = openedBy;
  else
    button =
      Object.entries(TOPBAR_TARGETS).find(
        ([, target]) => target === panel,
      )?.[0] ?? null;
  return button && !UNLIT_WHEN_OPEN.has(button) ? button : null;
}

export type TopbarControl =
  | { kind: "pause" }
  | { kind: "speed"; by: number }
  | { kind: "speedTo"; speed: number }
  | { kind: "music" }
  | { kind: "menu"; window: string };

export function topbarControlOf(
  name: string | null | undefined,
): TopbarControl | null {
  switch (name) {
    case "pause_button_date":
    case "pause_button_date_paused":
      return { kind: "pause" };
    case "button_speeddown":
      return { kind: "speed", by: -1 };
    case "button_speedup":
      return { kind: "speed", by: 1 };
    case "playlist_button":
      return { kind: "music" };
    case "menu_button":
      return { kind: "menu", window: "ingamemenu" };
    default: {
      const pip = /^button_speedstep([1-5])$/.exec(name ?? "");
      return pip ? { kind: "speedTo", speed: Number(pip[1]) } : null;
    }
  }
}

export interface ClockState {
  paused: boolean;
  speed: number;
}

export const START_CLOCK: ClockState = { paused: true, speed: 1 };

export const SPEEDS = 5;

export function nextClock(
  state: ClockState,
  control: TopbarControl,
): ClockState {
  const clamp = (v: number) => Math.min(SPEEDS, Math.max(1, v));
  if (control.kind === "pause") return { ...state, paused: !state.paused };
  if (control.kind === "speed")
    return { ...state, speed: clamp(state.speed + control.by) };
  if (control.kind === "speedTo")
    return { ...state, speed: clamp(control.speed) };
  return state;
}

export function cubicBezier(
  x1: number,
  y1: number,
  x2: number,
  y2: number,
): (t: number) => number {
  const at = (a: number, b: number, s: number) =>
    3 * (1 - s) * (1 - s) * s * a + 3 * (1 - s) * s * s * b + s * s * s;
  return (t: number) => {
    if (t <= 0) return 0;
    if (t >= 1) return 1;

    let lo = 0;
    let hi = 1;
    for (let i = 0; i < 30; i += 1) {
      const mid = (lo + hi) / 2;
      if (at(x1, x2, mid) < t) lo = mid;
      else hi = mid;
    }
    return at(y1, y2, (lo + hi) / 2);
  };
}

export const DECELERATED = cubicBezier(0, 0, 0.2, 1);
export const ACCELERATED = cubicBezier(0.4, 0, 1, 1);

export const ALERT_TARGETS: Readonly<Record<string, string>> = {
  alert_no_research: "countrytechnologyview",
  alert_free_civilian_factories: "countryconstructionsview",
  alert_free_military_factories: "countryproductionlineview",
  alert_free_naval_dockyards: "countryproductionlineview",
  alert_production_no_template: "countryproductionlineview",
  alert_outdated_equipment: "countryproductionlineview",
  alert_lack_of_resources: "countrytradeview",
  alert_select_focus: "nationalfocusview",
  alert_decision_new: "countrydecisionview",
  alert_pick_new_idea: "countrypoliticsview",
  alert_officer_corps: "countryofficercorpview",
  alert_doctrine_unlock: "countryofficercorpview",
  alert_not_training_divisions: "countrydeploymentview",
  alert_deployment_ready: "countrydeploymentview",
  alert_unassigned_divisions: "countryarmyview",
  alert_faction_possible_invite: "countryfactionview",
  alert_faction_can_select_goal: "countryfactionview",
};

export const PANEL_TARGETS: Readonly<Record<string, string>> = {
  "countrypoliticsview/add_national_goal_button": "nationalfocusview",

  "countrypoliticsview/show_subjects_button": "manage_subjects_window",

  "countrypoliticsview/manage_occupied_button": "countryoccupationview",
  "countrypoliticsview/show_exiles_collaborations_button": "exiles_window",
  "countrypoliticsview/show_power_balance": "powerbalanceview",

  "countrypoliticsview/new_faction_button": "countryfactionview",
  "countrypoliticsview/view_factions_button": "countryfactionview",
  "countrypoliticsview/open_faction_button": "countryfactionview",

  "countryinternationalmarketview/diplo_view_button": "countrydiplomacyview",
  "countrydiplomacyview/show_national_goal_button": "nationalfocusview",

  "countryintelligenceagencyview/agency_logo":
    "name_logo_agency_selection_window",

  "countryproductionlineview/infantry_button":
    "production_equipment_window_military",
  "countryproductionlineview/tank_button":
    "production_equipment_window_military",
  "countryproductionlineview/aircraft_button":
    "production_equipment_window_military",
  "countryproductionlineview/naval_button": "production_equipment_window",

  "countryarmyview/details_open": "equipment_overview_window",
  "countryarmyview/naval_hq_open": "building_roster_window",
  "countryarmyview/destroyed_ships_open": "navallosseswindow",

  "templatedeploymentwindow/open_designer_button":
    "countrydivisiondesignerview",

  "countryofficercorpview/commander_roster_army": "armyleaderwindow",
  "countryofficercorpview/commander_roster_navy": "navyleaderwindow",
  "countryofficercorpview/commander_roster_division": "divisioncommanderwindow",
  "countryofficercorpview/button_captured_generals": "capturedarmyleaderwindow",

  "countryofficercorpview/button_army_hq_templates_toggle":
    "armyhqtemplatewindow",

  "countryofficercorpview/land_doctrine_ring": "country_doctrine_view",
  "countryofficercorpview/naval_doctrine_ring": "country_doctrine_view",
  "countryofficercorpview/air_doctrine_ring": "country_doctrine_view",
  "countryofficercorpview/special_forces_doctrine_ring":
    "country_doctrine_view",
  "armyhqtemplatewindow/open_designer_button": "countrydivisiondesignerview",
};

export function matchNamePath(root: GuiNode, namePath: string): GuiNode[] {
  const parts = namePath.split("/").filter(Boolean);
  const leaf = parts.pop();
  const out: GuiNode[] = [];
  const walk = (node: GuiNode, ancestors: string[]) => {
    if (node.name === leaf) {
      let at = 0;
      for (const name of ancestors)
        if (at < parts.length && name === parts[at]) at += 1;
      if (at === parts.length) out.push(node);
    }
    const next = node.name ? [...ancestors, node.name] : ancestors;
    for (const child of node.children ?? []) walk(child, next);
  };
  walk(root, []);
  return out;
}

export const RUNTIME_HIDDEN: Readonly<Record<string, readonly string[]>> = {
  nationalfocusview: ["continuous_focus_window", "search_tooltip"],
  topbar: [
    "reopen_ingame_lobby",
    "reopen_ingame_lobby_label",

    "decisionview_amount_timeout_bg",
    "decisionview_amount_timeout_items",
    "intelview_ops_prepared_or_completed_bg",
    "intelview_ops_prepared_or_completed_items",
    "intelview_ops_available_bg",
    "intelview_ops_available_items",
    "prototype_rewards_aquired_bg",
    "prototype_rewards_aquired",
    "non_delivering_contracts",
    "delivering_contracts",
  ],
  countrypoliticsview: [
    "power_balance_button",

    "active_goal/focus_cost",
    "active_goal/pol_power_icon",
    "active_goal/progress",
    "active_goal/drop_continuous_focus_button",
    "active_goal/drop_focus_button",

    "original_in_faction",
    "in_faction",
  ],

  gamesetup_interesting_countries_window: ["info/government"],

  EventWindow: ["Title", "Description", "btn_minimize"],
  countrystateview: ["template_need_text", "state_name"],
  countrydiplomacyview: [
    "our_opinion_value",
    "their_opinion_value",

    "ideology_name",
    "elections",

    "country_name",
    "faction_name",
    "leader_name",
    "stability_value",
    "war_support_value",

    "party_name",
  ],
  countrytechnologyview: [
    "limited_research_bonus_value",
    "research_speed_value",
  ],
  countryintelligenceagencyview: [
    "ai_factory_allocation",
    "operation_ai_score",
  ],
  countryofficercorpview: ["icon_prefered_tactic_type", "badge_text"],
  armyleaderwindow: ["no_leader_button", "show_divisional_commanders_button"],
  navyleaderwindow: ["no_leader_button", "show_ship_captains_button"],
  unitleader_trait_tree_window: [
    "advisor_promotion_traits_tab",
    "highlight_land_doctrine_folder",
    "advisor_traits_window",
    "preferred_tactic_tiling_container",
    "wiki_button",
  ],
  countryproductionlineview: [
    "factories_used_resources",
    "factories_used_outdated",
    "factories_used_stockpiling",
    "factories_used_reinforcing",
    "factories_used_upgrading",
    "factories_used_refitting",
    "operational_naval_factories_repair_value",
  ],
  countrydivisiondesignerview: [
    "subunits_hq",
    "hq_overlay_images",
    "hq_title",
    "hq_badge",

    "hq_view",
    "save_and_deploy_button",

    "designerregimentswindow",
    "div_templ_custom_icon_view",
    "div_template_select_model",
    "division_equipment_view",
    "niche_icon_selector",

    "select_model",
    "toggle_model_selector_button",

    "colonial_force_flag",
    "colonial_force_flag_frame",

    "non_hq_view/equipment_warning",
    "hq_view/equipment_warning",

    "hardness_stat/stat_text",
    "org_stat/stat_text",
    "strength_stat/stat_text",
    "speed_stat/stat_text",
    "combat_width_view/combat_width_text",
    "hq_overlay_images/combat_width_view_hq/combat_width_text",
    "division_names/name",
    "non_hq_view/div_templ_symbol_button",
    "hq_view/div_templ_symbol_button",

    "non_hq_view/division_priority_icon",
    "non_hq_view/priority_button_1",
    "non_hq_view/priority_button_2",
    "non_hq_view/priority_button_3",

    "hq_view/division_priority_icon",
    "hq_view/priority_button_1",
    "hq_view/priority_button_2",
    "hq_view/priority_button_3",
    "non_hq_view/niche_icon_bg",
    "non_hq_view/niche_button",
    "non_hq_view/niche_icon_disabled",
  ],
  equipment_upgrade_designer_window: [
    "select_model_button",
    "equipment_select_model",
  ],
};

export const RUNTIME_SHOWN: Readonly<Record<string, readonly string[]>> = {
  countrydivisiondesignerview: ["non_hq_view", "template_emblem_slot"],

  program_window: ["projects"],

  countryarmyview: [
    "info",
    "army_header",
    "army_units",
    "navy_header",
    "navy_units",
    "air_header",
    "air_units",
  ],
};

export function isRuntimeEffect(name: string | null | undefined): boolean {
  return !!name && name.toLowerCase().includes("glow");
}

export const STEADY_GLOWS: Readonly<Record<string, readonly string[]>> = {
  mainmenu_dlc_ownership: ["glowup"],
};

export const RUNTIME_PRINTED_OVER: Readonly<Record<string, readonly string[]>> =
  {
    EventWindow: ["Title", "Description"],
    countrystateview: ["state_name"],
    countrydiplomacyview: ["country_name", "leader_name"],
  };

export const RUNTIME_TEXT: Readonly<
  Record<string, Readonly<Record<string, string>>>
> = {
  countryofficercorpview: {
    commander_roster_army: "COMMANDER_ROSTER_ARMY_BUTTON",
    commander_roster_division: "COMMANDER_ROSTER_DIVISION_BUTTON",
    commander_roster_navy: "COMMANDER_ROSTER_NAVY_BUTTON",
    commander_roster_ship: "COMMANDER_ROSTER_SHIP_BUTTON",
  },
  countrypoliticsview: {
    "active_goal/title": "POLITICS_SELECT_FOCUS",
    manage_exiled_collaborations_governments: "SHOW_COLLABORATIONS",
  },
};

export const COMPANIONS: Readonly<Record<string, readonly string[]>> = {
  countrydeploymentview: ["templatedeploymentwindow"],
};

export const OPENS_OVER: Readonly<Record<string, string>> = {
  countrydivisiondesignerview: "countrydeploymentview",

  tank_designer_view: "countryproductionlineview",

  plane_designer_view: "countryproductionlineview",

  countryequipmentdesignerview: "countryproductionlineview",

  diplomacy_action_info_window: "countrydiplomacyview",

  name_logo_agency_selection_window: "countryintelligenceagencyview",

  manage_subjects_window: "countrypoliticsview",

  manage_occupied_window: "countrypoliticsview",
  exiles_window: "countrypoliticsview",
  powerbalanceview: "countrypoliticsview",

  program_window: "countrytechnologyview",

  scientist_roster_window: "countrytechnologyview",

  intelligence_agency_upgrades_window: "countryintelligenceagencyview",

  production_equipment_window_military: "countryproductionlineview",
  production_equipment_window: "countryproductionlineview",

  equipment_overview_window: "countryarmyview",
  building_roster_window: "countryarmyview",
  navallosseswindow: "countryarmyview",

  political_ideas_window: "countrypoliticsview",

  industrial_organisation_list_window: "countrypoliticsview",
  industrial_organisation_detail_window: "industrial_organisation_list_window",

  armyleaderwindow: "countryofficercorpview",
  navyleaderwindow: "countryofficercorpview",
  divisioncommanderwindow: "countryofficercorpview",
  capturedarmyleaderwindow: "countryofficercorpview",

  armyhqtemplatewindow: "countryofficercorpview",

  unitleader_trait_tree_window: "armyleaderwindow",
};

export const NAMED_PLACES: Readonly<
  Record<string, { position: string; orientation?: string }>
> = {
  industrial_organisation_list_window: { position: "position_default" },
  armyleaderwindow: {
    position: "pos_roster_overview_orientation_top_left",
    orientation: "UPPER_LEFT",
  },
  navyleaderwindow: {
    position: "pos_roster_overview_orientation_top_left",
    orientation: "UPPER_LEFT",
  },
};

export function basePanelOf(
  panel: string | null,
  over: Readonly<Record<string, string>> = OPENS_OVER,
): string | null {
  let at = panel;
  for (let hops = 0; at && over[at] && hops < 8; hops++) at = over[at];
  return at;
}

export function opensOverWith(
  pickerOver: string | null,
  designerOver: string | null = null,
): Readonly<Record<string, string>> {
  let out: Record<string, string> = OPENS_OVER;
  if (pickerOver) out = { ...out, political_ideas_window: pickerOver };

  if (designerOver) out = { ...out, countrydivisiondesignerview: designerOver };
  return out;
}

export type NavAction =
  | { kind: "open"; window: string }
  | { kind: "close" }
  | { kind: "unsupported"; control: string }
  | { kind: "none" };

function isCloseButton(node: GuiNode): boolean {
  const shortcut = String(node.raw?.shortcut ?? "").toUpperCase();
  return (
    node.name === "close_button" ||
    node.name === "btn_close" ||
    node.name === "menu_close_button" ||
    node.name === "close" ||
    shortcut === "ESCAPE"
  );
}

export function resolveActivation(
  owner: string,
  node: GuiNode,
  current: string | null,
  over: Readonly<Record<string, string>> = OPENS_OVER,
): NavAction {
  const name = node.name ?? "";
  const unsupported = (): NavAction => ({
    kind: "unsupported",
    control: name.trim() || node.type || "unnamed_button",
  });
  if (owner === "topbar") {
    const target = TOPBAR_TARGETS[name];
    if (!target) return unsupported();
    return target === basePanelOf(current, over)
      ? { kind: "close" }
      : { kind: "open", window: target };
  }
  const panelTarget = PANEL_TARGETS[`${owner}/${name}`];
  if (panelTarget) return { kind: "open", window: panelTarget };
  if (isCloseButton(node)) {
    const under = over[owner];
    if (under && owner === current) return { kind: "open", window: under };
    return { kind: "close" };
  }
  return unsupported();
}

export function movesBetweenScreens(
  owner: string,
  node: GuiNode,
  current: string | null,
  over: Readonly<Record<string, string>> = OPENS_OVER,
): boolean {
  const action = resolveActivation(owner, node, current, over);
  return action.kind === "open" || action.kind === "close";
}

const CODE_KEYS: Readonly<Record<string, string>> = {
  Space: "space",
  NumpadAdd: "kp_plus",
  NumpadSubtract: "kp_minus",
};

export function hotkeyOf(
  e: Pick<
    KeyboardEvent,
    "key" | "shiftKey" | "ctrlKey" | "altKey" | "metaKey"
  > & { code?: string },
): string {
  const named = e.code ? CODE_KEYS[e.code] : undefined;
  const key =
    named ?? (e.key.length === 1 ? e.key.toLowerCase() : e.key.toUpperCase());
  const parts: string[] = [];
  if (e.ctrlKey || e.metaKey) parts.push("ctrl");
  if (e.shiftKey && e.key.length === 1 && !named) parts.push("shift");
  if (e.altKey) parts.push("alt");
  parts.push(key);
  return parts.join("+");
}

export function shortcutsOf(root: GuiNode): Map<string, GuiNode> {
  const out = new Map<string, GuiNode>();
  const walk = (node: GuiNode) => {
    const raw = node.raw?.shortcut;
    if (typeof raw === "string" && raw.trim()) {
      const spelled = raw.trim().toLowerCase();

      if (!out.has(spelled)) out.set(spelled, node);
    }
    for (const child of node.children ?? []) walk(child);
  };
  walk(root);
  return out;
}

export function restPlacement(
  node: GuiNode,
  screenW = 1920,
  screenH = 1080,
): { x: number; y: number; fromX: number; fromY: number; ms: number } {
  const num = (v: unknown, d = 0) => {
    const n = Number(v);
    return Number.isFinite(n) ? n : d;
  };

  const place = NAMED_PLACES[node.name ?? ""];
  const named = (node.children ?? []).find(
    (child) =>
      child.type === "positiontype" &&
      child.name === (place?.position ?? "position_default"),
  );
  const from = named?.position ?? node.position ?? { x: 0, y: 0 };
  const show = node.raw?.show_position as
    { x?: unknown; y?: unknown } | undefined;

  const [col, row] = anchorCell(
    named && place?.orientation ? place.orientation : node.orientation,
  );
  const size = node.size ?? {};

  const cap = (node.children ?? []).find(
    (child) =>
      child.type === "positiontype" && child.name === "max_window_size",
  )?.position;
  const extent = (value: unknown, screen: number, most: unknown) => {
    const text = String(value ?? "");
    const share = text.endsWith("%") ? Number(text.slice(0, -1)) : NaN;
    const out = Number.isFinite(share) ? (screen * share) / 100 : num(value);
    const limit = num(most);
    return limit > 0 ? Math.min(out, limit) : out;
  };
  const width = extent((size as { width?: unknown }).width, screenW, cap?.x);
  const height = extent((size as { height?: unknown }).height, screenH, cap?.y);

  const [selfCol, selfRow] = anchorCell(node.origo);
  const anchorX = (screenW * col) / 2 - (width * selfCol) / 2;
  const anchorY = (screenH * row) / 2 - (height * selfRow) / 2;
  const x = anchorX + (show ? num(show.x) : num(from.x));
  const y = anchorY + (show ? num(show.y) : num(from.y));
  return {
    x,
    y,
    fromX: anchorX + num(from.x),
    fromY: anchorY + num(from.y),
    ms: show ? num(node.raw?.animation_time, 300) : 0,
  };
}
