export interface Project {
  id: number;
  name: string;
  country_tag: string;
  country_tags: string[];
  country_name: string;
  mod_path: string;

  ideology_icon_paths: Record<string, string>;

  version: string;
  picture: string;
  tags: string[];
  dependencies: string[];
  supported_version: string;

  imported_descriptor_replace_paths?: string[];

  start_date?: string | null;

  end_date?: string | null;

  scenario_history_policy?:
    "inherit_base_history" | "project_owns_starting_world";

  stage_state_ids?: number[] | null;

  export_languages: string[];

  owned_dlcs?: string[] | null;

  is_loaded_existing_mod: boolean;

  basemod_template_id?: number | null;
  basemod_pinned_version?: string | null;

  ideology_registry?: import("@/lib/ideology/registry").IdeologyRegistry;

  tech_surface_manifest?: import("@/lib/techSurface").TechSurfaceManifest;
  created_at: string;
  updated_at: string;
  character_count: number;
  focus_node_count: number;
  technology_count: number;

  entity_counts?: Record<string, number>;
}

export interface CountryTag {
  tag: string;
  name: string;
}

export interface DynamicModifier {
  id: number;
  project_id: number;
  dm_id: string;
  country_tag: string;
  name_en: string;
  name_zh: string;

  display_name_locs: Record<string, string>;
  icon: string;
  remove_trigger: string;
  enable: string;
  modifier: Record<string, string | number>;
  raw_extras: Record<string, unknown>;
  source: string;
  created_at: string;
  updated_at: string;
}

export interface AutonomyState {
  id: number;
  project_id: number;
  autonomy_id: string;
  country_tag: string;
  name_en: string;
  name_zh: string;
  display_name_locs: Record<string, string>;
  is_puppet: boolean;

  is_default: boolean;
  manpower_influence: number | null;
  min_freedom_level: number;
  rule: string;
  ai_weights: Record<string, unknown>;
  modifier: Record<string, string | number>;
  raw_extras: Record<string, unknown>;
  source: string;
  created_at: string;
  updated_at: string;
}

export interface UiPanelControl {
  kind: "container" | "icon" | "text" | "progressbar" | "button";
  name: string;
  x: number;
  y: number;
  w?: number | null;
  h?: number | null;
  sprite?: string;
  frame?: number;
  text?: string;
  text_locs?: Record<string, string>;
  font?: string;
  format?: "left" | "center" | "right";
  visible_trigger?: string;
  frame_var?: string;
  image_expr?: string;
  tooltip?: string;
  tooltip_locs?: Record<string, string>;
  click_effect?: string;
  click_enabled_trigger?: string;
  raw_extras?: Record<string, unknown>;
  children?: UiPanelControl[];
  scroll?: boolean;
  array?: string;
  columns?: number;
  slot_w?: number;
  slot_h?: number;
  change_scope?: boolean;
}

export interface UiPanelDerivedControl {
  control_name: string;
  visible_trigger_key: string;
  text_loc_key: string;

  click_key: string;
  click_enabled_key: string;
  tooltip_loc_key: string;
}

export interface UiPanelDerived {
  token: string;
  scripted_gui_key: string;
  window_name: string;
  gui_path: string;
  txt_path: string;
  loc_basename: string;
  title_loc_key: string;
  controls: Record<string, UiPanelDerivedControl>;
}

export interface UiPanel {
  id: number;
  project_id: number;
  token: string;
  country_tag: string;
  name_en: string;
  name_zh: string;
  display_name_locs: Record<string, string>;
  context_type: string;
  parent_window_token: string;
  visible_trigger: string;
  window_props: Record<string, unknown>;
  controls: UiPanelControl[];
  raw_extras: Record<string, unknown>;
  source: string;
  created_at: string;
  updated_at: string;
  warnings?: string[];
  derived?: UiPanelDerived;
}

export interface UiPanelFiles {
  token: string;
  window_name: string;
  gui_path: string;
  gui_text: string;
  txt_path: string;
  txt_text: string;
  loc: Record<string, string>;
  loc_paths: string[];
}
