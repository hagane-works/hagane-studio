export interface WargoalType {
  id: number;
  project_id: number;
  wargoal_key: string;
  war_name: string;
  safe_scalars: Record<string, string>;
  allowed: string;
  available: string;
  take_states: string;
  puppet: string;
  liberate: string;
  advanced_raw_script: string;
  source_file: string;
  source: "created" | "imported" | string;
  created_at: string;
  updated_at: string;
}
export interface WargoalWrite {
  wargoal_key: string;
  war_name: string;
  safe_scalars: Record<string, string>;
  allowed: string;
  available: string;
  take_states: string;
  puppet: string;
  liberate: string;
  advanced_raw_script: string;
  source_file: string;
}

export interface WargoalListResponse {
  wargoals: WargoalType[];
  total: number;
}
