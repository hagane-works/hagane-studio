export const SPECIAL_PROJECT_COMPLEXITY_OPTIONS: {
  value: string;
  label: string;
}[] = [
  { value: "sp_complexity.small", label: "Small" },
  { value: "sp_complexity.medium", label: "Medium" },
  { value: "sp_complexity.large", label: "Large" },
  { value: "sp_complexity.insane", label: "Insane" },
];

export interface Specialization {
  id: number;
  project_id: number;
  country_tag: string;
  specialization_id: string;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;
  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  color: number[];
  blueprint_image: string;
  program_background: string;
  facility_model: string;
  sub_specializations: string[];
  raw_extras: Record<string, unknown>;
  source: string;
}

export interface SpecialProject {
  id: number;
  project_id: number;
  country_tag: string;
  project_token: string;
  specialization: string;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;
  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  icon: string;
  blueprint_image: string;
  complexity: string;
  prototype_time: string;
  project_tags: string[];
  special_project_parent: string[];
  allowed: string;
  visible: string;
  available: string;
  resource_cost: string;
  project_output: string;
  generic_prototype_rewards: string[];
  x: number;
  y: number;
  raw_extras: Record<string, unknown>;
  source: string;
}

export interface SpecialProjectValidationItem {
  kind: string;
  detail: string;
  project_token?: string;
  parent?: string;
  nodes?: string[];
}
export interface SpecialProjectValidation {
  actionable: SpecialProjectValidationItem[];
  info: SpecialProjectValidationItem[];
  total: number;
}
