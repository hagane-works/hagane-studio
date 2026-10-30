export interface TechnologyFolderPlacement {
  folder: string;
  x: number;
  y: number;
}

export interface Technology {
  id: number;
  project_id: number;
  tech_id: string;
  name: string;
  tech_line: string;
  research_cost: number;
  start_year: number;
  folder: string;
  position_x: number;
  position_y: number;
  position_y_is_explicit?: boolean | null;

  folder_placements?: TechnologyFolderPlacement[] | null;
  prerequisites: string[];
  categories: string[];
  modifiers: Record<string, Record<string, number>>;
  ai_will_do: string;
  raw_extras: Record<string, unknown>;
  country_tag: string;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;

  display_name_locs: Record<string, string>;

  description_locs: Record<string, string>;
  source?: string;
  is_banner?: boolean;

  force_use_small_tech_layout?: boolean;

  enable_equipments?: string[];
  enable_subunits?: string[];
  enable_equipment_modules?: string[];
  enable_building?: string;
  enable_tactic?: string;
  xor?: string[];
  dependencies?: Record<string, number>;
  sub_technologies?: string[];
  doctrine?: boolean;
  allow?: string;
  allow_branch?: string;
  on_research_complete?: string;
  on_research_complete_limit?: string;
  path?: Array<{ leads_to_tech: string; research_cost_coeff: number }>;
  xp_research_type?: "" | "land" | "naval" | "air";
  xp_boost_cost?: number;
  xp_research_bonus?: Record<string, number>;
}
