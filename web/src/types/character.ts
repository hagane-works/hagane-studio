export type RoleType =
  | "general"
  | "field_marshal"
  | "admiral"
  | "scientist"
  | "advisor"
  | "country_leader"
  | "operative";

export interface TraitApplicabilityMetadata {
  types: string[];
  trait_types: string[];
}

const UNIT_LEADER_TYPES_BY_ROLE: Partial<
  Record<RoleType, ReadonlySet<string>>
> = {
  general: new Set(["land", "corps_commander", "all"]),
  field_marshal: new Set(["land", "field_marshal", "all"]),
  admiral: new Set(["navy", "all"]),
  operative: new Set(["operative", "all"]),
};

const OPERATIVE_TRAIT_TYPES = new Set(["basic_trait", "personality_trait"]);

export function isUnitLeaderTraitEligibleForRole(
  roleType: RoleType,
  metadata: TraitApplicabilityMetadata | undefined,
): boolean {
  const allowedTypes = UNIT_LEADER_TYPES_BY_ROLE[roleType];
  if (!allowedTypes || !metadata || metadata.types.length === 0) return false;
  if (!metadata.types.some((type) => allowedTypes.has(type))) return false;
  if (roleType !== "operative") return true;
  return metadata.trait_types.some((traitType) =>
    OPERATIVE_TRAIT_TYPES.has(traitType),
  );
}

export function filterUnitLeaderTraitsForRole(
  roleType: RoleType,
  traitIds: string[],
  metadataById: Record<string, TraitApplicabilityMetadata>,
): string[] {
  return traitIds.filter((traitId) =>
    isUnitLeaderTraitEligibleForRole(roleType, metadataById[traitId]),
  );
}

export interface Character {
  id: number;
  project_id: number;
  name: string;
  tag: string;

  localisation_key: string;
  description: string;
  role_type: RoleType;
  has_advisor_role: boolean;
  is_replacement: boolean;

  replaces_vanilla_id: boolean;
  skill: number;
  attack_skill: number;
  defense_skill: number;
  planning_skill: number;
  logistics_skill: number;
  maneuvering_skill: number;
  coordination_skill: number;

  military_visible: string;
  traits: string[];
  advisor_traits: string[];
  advisor_slot: string;
  advisor_idea_token: string;
  advisor_cost: number;
  advisor_available: string;
  advisor_visible: string;
  advisor_removal_cost: number;
  advisor_modifier: string;
  advisor_on_add: string;
  advisor_on_remove: string;
  gender: string;
  can_be_captured?: boolean;

  always_show_on_actions_tooltip?: boolean | null;
  scientist_data: ScientistData;

  scientist_visible: string;

  operative_skill_level: number;
  operative_traits: string[];
  operative_nationalities: string[];
  operative_available: string;
  operative_bg: string;
  ideology: string;
  expire: string;
  large_portrait: string;
  small_portrait: string;
  raw_extras: Record<string, unknown>;
  country_tag: string;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;

  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;

  is_portrait_modified: boolean;

  source?: string;

  source_file?: string;
}

export interface ScientistData {
  specializations: { spec: string; level: number }[];
  allow_multiple: boolean;
}

export type CustomTraitType = "country_leader" | "unit_leader" | "scientist";

export const CUSTOM_TRAIT_TYPE_OPTIONS: {
  value: CustomTraitType;
  label: string;
  label_i18n_key: string;
  enabled: boolean;
}[] = [
  {
    value: "country_leader",
    label: "Country Leader / 国家领导人",
    label_i18n_key: "countryLeader",
    enabled: true,
  },
  {
    value: "unit_leader",
    label: "Unit Leader / 将领 (Army & Navy)",
    label_i18n_key: "unitLeader",
    enabled: true,
  },
  {
    value: "scientist",
    label: "Scientist / 科学家 (Special Projects)",
    label_i18n_key: "scientist",
    enabled: true,
  },
];

export const UNIT_LEADER_TRAIT_TYPE_OPTIONS: {
  value: string;
  label: string;
}[] = [
  { value: "land", label: "Land (Army)" },
  { value: "navy", label: "Navy" },
  { value: "all", label: "All (both)" },
];

export const UNIT_LEADER_SKILL_KEYS = [
  "attack_skill",
  "defense_skill",
  "logistics_skill",
  "planning_skill",
  "maneuvering_skill",
  "coordination_skill",
] as const;

export const UNIT_LEADER_SKILL_FACTOR_KEYS = [
  "attack_skill_factor",
  "defense_skill_factor",
  "logistics_skill_factor",
  "planning_skill_factor",
  "maneuvering_skill_factor",
  "coordination_skill_factor",
] as const;

export const UNIT_LEADER_ADVANCED_INT_FIELDS: {
  key: string;
  label: string;
  hint: string;
}[] = [
  {
    key: "gui_row",
    label: "gui_row",
    hint: "trait tree row (0+, omit = not in tree)",
  },
  {
    key: "gui_column",
    label: "gui_column",
    hint: "trait tree column (0+, omit = auto)",
  },
  { key: "cost", label: "cost", hint: "command power cost (default 0)" },
  {
    key: "gain_xp_on_spotting",
    label: "gain_xp_on_spotting",
    hint: "XP gained when spotting (integer)",
  },
];

export const UNIT_LEADER_ADVANCED_STRING_FIELDS: {
  key: string;
  label: string;
  hint: string;
}[] = [
  {
    key: "trait_type",
    label: "trait_type",
    hint: "basic_trait / status_trait / personality_trait / assignable_trait / basic_terrain_trait / exile",
  },
  {
    key: "mutually_exclusive",
    label: "mutually_exclusive",
    hint: "key of another trait that cannot coexist",
  },
  {
    key: "enable_ability",
    label: "enable_ability",
    hint: "ability key unlocked while equipped",
  },
  {
    key: "override_effect_tooltip",
    label: "override_effect_tooltip",
    hint: "loc key (overrides default effect text)",
  },
  {
    key: "custom_effect_tooltip",
    label: "custom_effect_tooltip",
    hint: "loc key (extra effect blurb)",
  },
  {
    key: "custom_prerequisite_tooltip",
    label: "custom_prerequisite_tooltip",
    hint: "loc key",
  },
  {
    key: "custom_gain_xp_trigger_tooltip",
    label: "custom_gain_xp_tooltip",
    hint: "loc key",
  },
];

export const UNIT_LEADER_ADVANCED_BOOL_FIELDS: {
  key: string;
  label: string;
  hint: string;
}[] = [
  {
    key: "show_in_combat",
    label: "show_in_combat",
    hint: "show this trait in combat UI",
  },
];

export const UNIT_LEADER_ADVANCED_SCRIPT_FIELDS: {
  key: string;
  label: string;
  hint: string;
}[] = [
  {
    key: "allowed",
    label: "allowed",
    hint: "trigger block · scope = unit leader",
  },
  {
    key: "prerequisites",
    label: "prerequisites",
    hint: "trigger block · scope = unit leader",
  },
  {
    key: "gain_xp",
    label: "gain_xp",
    hint: "trigger block · scope = combatant",
  },
  {
    key: "gain_xp_leader",
    label: "gain_xp_leader",
    hint: "trigger block · scope = unit leader",
  },
  {
    key: "on_add",
    label: "on_add",
    hint: "effect block · fires when trait added",
  },
  {
    key: "on_remove",
    label: "on_remove",
    hint: "effect block · fires when trait removed",
  },
  {
    key: "daily_effect",
    label: "daily_effect",
    hint: "effect block · fires each day",
  },
];

export const UNIT_LEADER_ADVANCED_MODIFIER_FIELDS: {
  key: string;
  label: string;
  hint: string;
}[] = [
  {
    key: "non_shared_modifier",
    label: "non_shared_modifier",
    hint: "only applies to the unit the leader is attached to",
  },
  {
    key: "corps_commander_modifier",
    label: "corps_commander_modifier",
    hint: "applies when leader acts as corps commander",
  },
  {
    key: "field_marshal_modifier",
    label: "field_marshal_modifier",
    hint: "applies when leader acts as field marshal",
  },
];

export const SCIENTIST_TRAIT_SPECIALIZATION_OPTIONS: {
  value: string;
  label: string;
}[] = [
  { value: "", label: "(generic — no specialization)" },
  { value: "specialization_air", label: "Air (aircraft / rocketry / radar)" },
  {
    value: "specialization_land",
    label: "Land (tanks / artillery / infantry)",
  },
  { value: "specialization_naval", label: "Naval (ships / submarines)" },
  { value: "specialization_nuclear", label: "Nuclear" },
];

export interface CustomTrait {
  id: number;
  project_id: number;
  trait_type: CustomTraitType;
  trait_id: string;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;

  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  modifier: Record<string, number | string>;
  targeted_modifier: Array<{
    tag: string;
    modifier: Record<string, number | string>;
  }>;
  ai_will_do: string;
  random: boolean;
  extra_fields: Record<string, unknown>;
  raw_extras: Record<string, unknown>;
  country_tag: string;
}

export type AdvisorSlot =
  | "political_advisor"
  | "army_chief"
  | "navy_chief"
  | "air_chief"
  | "high_command"
  | "theorist";
export type ScientistSpecialization =
  | "specialization_air"
  | "specialization_land"
  | "specialization_naval"
  | "specialization_nuclear";

export const ROLE_TYPE_OPTIONS: {
  value: RoleType;
  label: string;
  label_i18n_key: string;
}[] = [
  {
    value: "country_leader",
    label: "国家领导人 (Country Leader)",
    label_i18n_key: "countryLeader",
  },
  {
    value: "general",
    label: "将领 (Corps Commander)",
    label_i18n_key: "corpsCommander",
  },
  {
    value: "field_marshal",
    label: "元帅 (Field Marshal)",
    label_i18n_key: "fieldMarshal",
  },
  { value: "admiral", label: "海军将领 (Admiral)", label_i18n_key: "admiral" },
  {
    value: "scientist",
    label: "科学家 (Scientist)",
    label_i18n_key: "scientist",
  },
  { value: "advisor", label: "顾问 (Advisor)", label_i18n_key: "advisor" },

  {
    value: "operative",
    label: "特工 (Operative · LaR DLC)",
    label_i18n_key: "operativeLarDlc",
  },
];

export const IDEOLOGY_OPTIONS = [
  {
    value: "neutrality",
    label: "中立 (Neutrality)",
    label_i18n_key: "neutrality",
  },
  {
    value: "democracy",
    label: "民主 (Democracy)",
    label_i18n_key: "democracy",
  },
  { value: "fascism", label: "法西斯 (Fascism)", label_i18n_key: "fascism" },
  {
    value: "communism",
    label: "共产 (Communism)",
    label_i18n_key: "communism",
  },
] as const;

export const ADVISOR_SLOT_OPTIONS: {
  value: AdvisorSlot;
  label: string;
  label_i18n_key: string;
}[] = [
  {
    value: "political_advisor",
    label: "政治顾问 (Political Advisor)",
    label_i18n_key: "politicalAdvisor",
  },
  {
    value: "army_chief",
    label: "陆军总长 (Army Chief)",
    label_i18n_key: "armyChief",
  },
  {
    value: "navy_chief",
    label: "海军总长 (Navy Chief)",
    label_i18n_key: "navyChief",
  },
  {
    value: "air_chief",
    label: "空军总长 (Air Chief)",
    label_i18n_key: "airChief",
  },
  {
    value: "high_command",
    label: "最高统帅部 (High Command)",
    label_i18n_key: "highCommand",
  },
  {
    value: "theorist",
    label: "军事理论家 (Theorist)",
    label_i18n_key: "theorist",
  },
];

export const SCIENTIST_SPECIALIZATION_OPTIONS: {
  value: ScientistSpecialization;
  label: string;
  label_i18n_key: string;
}[] = [
  { value: "specialization_air", label: "空军 (Air)", label_i18n_key: "air" },
  {
    value: "specialization_land",
    label: "陆军 (Land)",
    label_i18n_key: "land",
  },
  {
    value: "specialization_naval",
    label: "海军 (Naval)",
    label_i18n_key: "naval",
  },
  {
    value: "specialization_nuclear",
    label: "核物理 (Nuclear)",
    label_i18n_key: "nuclear",
  },
];

export const ARMY_TRAIT_OPTIONS: string[] = [
  "brilliant_strategist",
  "inflexible_strategist",
  "trait_cautious",
  "trait_reckless",
  "career_officer",
  "old_guard",
  "war_hero",
  "media_personality",
  "politically_connected",
  "harsh_leader",
  "infantry_officer",
  "cavalry_officer",
  "armor_officer",
  "panzer_leader",
  "commando",
  "trickster",
  "trait_engineer",
  "fortress_buster",
  "organizer",
  "skilled_staffer",
  "hill_fighter",
  "jungle_rat",
  "desert_fox",
  "winter_specialist",
  "naval_invader",
  "offensive_doctrine",
  "defensive_doctrine",
];

export const NAVY_TRAIT_OPTIONS: string[] = [
  "navy_career_officer",
  "old_guard_navy",
  "bold",
  "gentlemanly",
  "aviation_enthusiast",
  "caustic_personality",
  "gunnery_expert",
  "blockade_runner",
  "superior_tactician",
  "spotter",
  "fleet_protector",
  "blue_water_expert",
  "cuts_corners",
];

export const ADVISOR_TRAIT_OPTIONS: string[] = [
  "silent_workhorse",
  "war_industrialist",
  "captain_of_industry",
  "financial_expert",
  "armaments_organizer",
  "smooth_talking_charmer",
  "compassionate_gentleman",
  "editor",
  "economist",
  "popular_figurehead",
  "prince_of_terror",
  "backroom_backstabber",
  "quartermaster_general",
  "head_of_intelligence",
  "anti_communist_crusader",
  "democratic_reformer",
  "fascist_demagogue",
  "communist_revolutionary",
  "army_chief_offensive_2",
  "army_chief_morale_2",
  "army_chief_planning_2",
  "army_chief_drill_2",
  "navy_chief_reform_2",
  "navy_chief_commerce_raiding_2",
  "navy_chief_naval_aviation_2",
  "air_chief_reform_3",
  "air_chief_night_operations_2",
  "air_chief_safety_2",
  "army_infantry_2",
  "army_commando_2",
  "army_logistics_2",
  "navy_carrier_2",
  "navy_naval_air_defense_2",
  "navy_amphibious_assault_2",
  "air_strategic_bombing_2",
  "air_close_air_support_2",
  "air_tactical_bombing_2",
  "air_pilot_training_3",
  "mobile_warfare_expert",
  "superior_firepower_expert",
  "naval_theorist",
  "naval_aviation_pioneer",
  "victory_through_airpower",
  "air_warfare_theorist",
];

export const SCIENTIST_GENERIC_TRAITS: string[] = [
  "scientist_trait_genius",
  "scientist_trait_brilliant_theorist",
  "scientist_trait_gifted_engineer",
  "scientist_trait_bright",
  "scientist_trait_fast_learner",
  "scientist_trait_resourceful",
];

export const SCIENTIST_SPEC_TRAITS: Record<ScientistSpecialization, string[]> =
  {
    specialization_air: [
      "scientist_trait_aircraft_specialist",
      "scientist_trait_rocketry_specialist",
      "scientist_trait_electronics_specialist",
      "scientist_trait_helicopter_expert",
      "scientist_trait_jet_expert",
      "scientist_trait_bombs_expert",
      "scientist_trait_guidance_systems_expert",
    ],
    specialization_land: [
      "scientist_trait_armored_specialist",
      "scientist_trait_artillery_specialist",
    ],
    specialization_naval: [
      "scientist_trait_submarine_specialist",
      "scientist_trait_surface_ship_specialist",
    ],
    specialization_nuclear: [
      "scientist_trait_nuclear_specialist",
      "scientist_trait_nuclear_warfare_expert",
      "scientist_trait_nuclear_power_expert",
    ],
  };

export const SCIENTIST_TRAIT_OPTIONS: string[] = [
  ...SCIENTIST_GENERIC_TRAITS,
  ...Object.values(SCIENTIST_SPEC_TRAITS).flat(),
];

export function getTraitOptionsForRole(roleType: RoleType): string[] {
  switch (roleType) {
    case "general":
    case "field_marshal":
      return ARMY_TRAIT_OPTIONS;
    case "admiral":
      return NAVY_TRAIT_OPTIONS;
    case "advisor":
      return ADVISOR_TRAIT_OPTIONS;
    case "scientist":
      return SCIENTIST_TRAIT_OPTIONS;
    default:
      return [];
  }
}

export function getVisibleVanillaTraitsForPicker(
  roleType: RoleType,
  vanillaTraitIds: string[],
  search: string,
  searchLimit = 20,
): string[] {
  const query = search.trim().toLowerCase();
  if (query) {
    return vanillaTraitIds
      .filter((traitId) => traitId.toLowerCase().includes(query))
      .slice(0, searchLimit);
  }

  const available = new Set(vanillaTraitIds);
  const curated = getTraitOptionsForRole(roleType).filter((traitId) =>
    available.has(traitId),
  );
  return curated.length > 0 ? curated : vanillaTraitIds.slice(0, searchLimit);
}
