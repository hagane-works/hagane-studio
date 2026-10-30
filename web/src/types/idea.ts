export type IdeaType =
  | "country"
  | "hidden_ideas"
  | "tank_manufacturer"
  | "naval_manufacturer"
  | "aircraft_manufacturer"
  | "materiel_manufacturer"
  | "industrial_concern"
  | "political_advisor"
  | "army_chief"
  | "navy_chief"
  | "air_chief"
  | "high_command"
  | "theorist"
  | "law"
  | "academy_spirit"
  | "army_spirit"
  | "division_command_spirit"
  | "naval_academy_spirit"
  | "navy_spirit"
  | "naval_command_spirit"
  | "air_force_academy_spirit"
  | "air_force_spirit"
  | "air_force_command_spirit"
  | "economy"
  | "mobilization_laws"
  | "trade_laws";

export const IDEA_TYPE_OPTIONS: {
  value: IdeaType;
  label: string;
  label_i18n_key: string;
}[] = [
  {
    value: "country",
    label: "National Spirit / 国家精神",
    label_i18n_key: "nationalSpirit",
  },
  {
    value: "hidden_ideas",
    label: "Hidden Spirit / 隐藏精神",
    label_i18n_key: "hiddenSpirit",
  },
  {
    value: "tank_manufacturer",
    label: "Tank Designer / 坦克设计公司",
    label_i18n_key: "tankDesigner",
  },
  {
    value: "naval_manufacturer",
    label: "Naval Designer / 海军设计公司",
    label_i18n_key: "navalDesigner",
  },
  {
    value: "aircraft_manufacturer",
    label: "Aircraft Designer / 飞机设计公司",
    label_i18n_key: "aircraftDesigner",
  },
  {
    value: "materiel_manufacturer",
    label: "Materiel Designer / 装备设计公司",
    label_i18n_key: "materielDesigner",
  },
  {
    value: "industrial_concern",
    label: "Industrial Concern / 工业企业",
    label_i18n_key: "industrialConcern",
  },
  {
    value: "political_advisor",
    label: "Political Advisor / 政治顾问 idea",
    label_i18n_key: "politicalAdvisor_2",
  },
  {
    value: "army_chief",
    label: "Army Chief / 陆军总长 idea",
    label_i18n_key: "armyChief_2",
  },
  {
    value: "navy_chief",
    label: "Navy Chief / 海军总长 idea",
    label_i18n_key: "navyChief_2",
  },
  {
    value: "air_chief",
    label: "Air Chief / 空军总长 idea",
    label_i18n_key: "airChief_2",
  },
  {
    value: "high_command",
    label: "High Command / 总司令部 idea",
    label_i18n_key: "highCommand_2",
  },
  {
    value: "theorist",
    label: "Theorist / 理论家 idea",
    label_i18n_key: "theorist_2",
  },
  { value: "law", label: "Law / 法律 idea", label_i18n_key: "law" },

  {
    value: "academy_spirit",
    label: "Academy Spirit / 军校精神",
    label_i18n_key: "academySpirit",
  },
  {
    value: "army_spirit",
    label: "Army Spirit / 陆军精神",
    label_i18n_key: "armySpirit",
  },
  {
    value: "division_command_spirit",
    label: "Division Command Spirit / 师指挥部精神",
    label_i18n_key: "divisionCommandSpirit",
  },
  {
    value: "naval_academy_spirit",
    label: "Naval Academy Spirit / 海军军校精神",
    label_i18n_key: "navalAcademySpirit",
  },
  {
    value: "navy_spirit",
    label: "Navy Spirit / 海军精神",
    label_i18n_key: "navySpirit",
  },
  {
    value: "naval_command_spirit",
    label: "Naval Command Spirit / 海军指挥部精神",
    label_i18n_key: "navalCommandSpirit",
  },
  {
    value: "air_force_academy_spirit",
    label: "Air Force Academy Spirit / 空军军校精神",
    label_i18n_key: "airForceAcademySpirit",
  },
  {
    value: "air_force_spirit",
    label: "Air Force Spirit / 空军精神",
    label_i18n_key: "airForceSpirit",
  },
  {
    value: "air_force_command_spirit",
    label: "Air Force Command Spirit / 空军指挥部精神",
    label_i18n_key: "airForceCommandSpirit",
  },

  {
    value: "economy",
    label: "Economy Law / 经济法 idea",
    label_i18n_key: "economyLaw",
  },
  {
    value: "mobilization_laws",
    label: "Mobilization Law / 动员法 idea",
    label_i18n_key: "mobilizationLaw",
  },
  {
    value: "trade_laws",
    label: "Trade Law / 贸易法 idea",
    label_i18n_key: "tradeLaw",
  },
];

export const DESIGNER_TYPES: IdeaType[] = [
  "tank_manufacturer",
  "naval_manufacturer",
  "aircraft_manufacturer",
  "materiel_manufacturer",
  "industrial_concern",
];

export const CHARACTER_SLOT_IDEA_TYPES: IdeaType[] = [
  "political_advisor",
  "army_chief",
  "navy_chief",
  "air_chief",
  "high_command",
  "theorist",
];

export interface IdeaTargetedModifier {
  tag: string;
  modifier: Record<string, number | string>;
}

export type IdeaLedger = "" | "civilian" | "army" | "navy" | "air";

export interface Idea {
  id: number;
  project_id: number;
  idea_id: string;
  idea_type: IdeaType;
  name: string;
  picture: string;
  modifier: Record<string, number | string>;
  removal_cost: number;
  allowed: string;
  allowed_civil_war: string;
  is_designer: boolean;
  research_bonus: Record<string, number>;
  designer_traits: string[];
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

  auto_attach_on_start: boolean;

  available: string;
  visible: string;
  allowed_to_remove: string;
  cancel: string;
  do_effect: string;
  on_add: string;
  on_remove: string;

  cancel_if_invalid: boolean;

  targeted_modifier: IdeaTargetedModifier[];
  equipment_bonus: Record<string, Record<string, number> | boolean>;
  rule: Record<string, boolean | number | string>;

  level: number;
  ledger: IdeaLedger;

  cost: number;

  source?: string;
}
