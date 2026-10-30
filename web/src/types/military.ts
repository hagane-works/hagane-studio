export interface Equipment {
  id: number;
  project_id: number;
  equipment_id: string;
  country_tag: string;
  name: string;
  equipment_type: string;
  year: number | null;
  picture: string;
  archetype: string;
  parent: string;
  is_archetype: boolean;
  is_buildable: boolean | null;
  soft_attack: number | null;
  hard_attack: number | null;
  ap_attack: number | null;
  air_attack: number | null;
  breakthrough: number | null;
  defense: number | null;
  armor_value: number | null;
  hardness: number | null;
  max_strength: number | null;
  build_cost_ic: number | null;
  manpower: number | null;
  reliability: number | null;
  maximum_speed: number | null;
  can_license: boolean | null;
  naval_speed: number | null;
  fire_range: number | null;
  lg_armor_piercing: number | null;
  lg_attack: number | null;
  hg_armor_piercing: number | null;
  hg_attack: number | null;
  torpedo_attack: number | null;
  anti_air_attack: number | null;
  shore_bombardment: number | null;
  evasion: number | null;
  surface_detection: number | null;
  sub_attack: number | null;
  sub_detection: number | null;
  surface_visibility: number | null;
  sub_visibility: number | null;
  naval_range: number | null;
  port_capacity_usage: number | null;
  search_and_destroy_coordination: number | null;
  convoy_raiding_coordination: number | null;
  air_defence: number | null;
  air_range: number | null;
  air_agility: number | null;
  air_ground_attack: number | null;
  air_bombing: number | null;
  air_superiority: number | null;
  naval_strike_attack: number | null;
  naval_strike_targetting: number | null;
  carrier_size: number | null;
  default_carrier_composition_weight: number | null;
  carrier_capable: boolean | null;
  resources: Record<string, unknown>;
  raw_extras: Record<string, unknown>;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;
  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  source: string;
  source_file: string;
  warnings: string[];
}

export interface SubUnit {
  id: number;
  project_id: number;
  sub_unit_id: string;
  country_tag: string;
  source_file: string;
  source: string;
  ordering: number;
  name: string;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;
  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  abbreviation: string;
  sprite: string;
  map_icon_category: string;
  priority: number | null;
  ai_priority: number | null;
  transport: string;
  group: string;
  active: boolean | null;
  cavalry: boolean | null;
  special_forces: boolean | null;
  marines: boolean | null;
  can_exfiltrate_from_coast: boolean | null;
  mountaineers: boolean | null;
  can_be_parachuted: boolean | null;
  affects_speed: boolean | null;
  unit_types: string[];
  categories: string[];
  essential: string[];
  need: Record<string, unknown>;
  stats: Record<string, unknown>;
  terrain_modifiers: Record<string, unknown>;
  raw_extras: Record<string, unknown>;
}

export interface DivisionRegiment {
  token: string;
  x: number;
  y: number;
}

export interface DivisionTemplate {
  id: number;
  project_id: number;
  oob_file: string;
  template_name: string;
  country_tag: string;
  division_names_group: string;
  regiments: DivisionRegiment[];
  support: DivisionRegiment[];
  is_locked: boolean | null;
  priority: number | null;
  ordering: number;
  oob_units_tail: string;
  raw_extras: Record<string, unknown>;
  source: string;
}

export interface DivisionDeployment {
  id: number;
  project_id: number;
  oob_file: string;
  country_tag: string;
  division_template: string;
  location: number | null;
  count: number;
  division_name: string;
  start_experience_factor: number | null;
  start_equipment_factor: number | null;
  ordering: number;
  raw_extras: Record<string, unknown>;
  source: string;

  warnings: string[];
}

export interface VanillaSubunit {
  token: string;
  abbreviation: string;
  is_support: boolean;
  group: string;
  types: string[];
  combat_width: number;
  max_strength: number;
  max_organisation: number;
  soft_attack: number;
  hard_attack: number;
  defense: number;
  breakthrough: number;
  manpower: number;
  default_morale: number;
  supply_consumption: number;
}
