export interface DecisionCategory {
  id: number;
  project_id: number;
  category_id: string;
  icon: string;
  picture: string;
  priority: number;
  allowed: string;
  visible: string;

  on_map_area: string;
  raw_extras: Record<string, unknown>;
  country_tag: string;
  scope_kind: string;
  shared_country_tags: string[] | null;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;

  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;

  visibility: string;
  created_at: string;
  updated_at: string;
  decision_count: number;
}

export interface Decision {
  id: number;
  project_id: number;
  category_id: number;
  decision_id: string;
  icon: string;
  allowed: string;
  available: string;
  visible: string;
  cost: number;
  fire_only_once: boolean;

  cancel_if_not_visible: boolean;
  days_remove: number;
  days_re_enable: number;
  complete_effect: string;
  remove_effect: string;
  cancel_trigger: string;
  cancel_effect: string;
  modifier: string;

  activation: string;
  timeout_effect: string;
  days_mission_timeout: number;
  selectable_mission: boolean;
  is_good: boolean;

  targeted_modifier: Record<string, unknown>;
  targets: string[];
  target_array: string[];
  target_root_trigger: string;
  target_trigger: string;
  state_target: string;

  highlight_states: string[];
  on_map_mode: string;
  priority: number;
  custom_cost_trigger: string;
  custom_cost_text: string;
  war_with_on_add: string;
  war_with_on_remove: string;
  ai_will_do: string;
  raw_extras: Record<string, unknown>;
  country_tag: string;
  scope_kind: string;
  shared_country_tags: string[] | null;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;

  display_name_locs: Record<string, string>;

  description_locs: Record<string, string>;

  effect_tooltip_locs: Record<string, string>;

  source?: string;
  created_at: string;
  updated_at: string;
}
