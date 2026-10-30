export interface StateVictoryPoint {
  province_id: number;
  value: number;
}

export interface StateProvinceController {
  province_id: number;
  controller: string;
}

export type StateProvinceBuildings = Record<string, Record<string, number>>;

export interface StateSourceSnapshot {
  schema_version: "hagane.state-source-snapshot.v1";
  state_id: number;
  province_ids: number[];
  owner: string | null;
  controller: string | null;
  source_path: string | null;
  source_line: number | null;
  source_content_sha256: string | null;
}

export interface StateMembershipProvenance {
  status: "known" | "legacy_unknown";
  source: number[] | null;
  current: number[];
  added: number[];
  removed: number[];
  changed: number[];
  membership_changed: boolean | null;
  owner_changed: boolean | null;
  controller_changed: boolean | null;
}

export interface State {
  id: number;
  project_id: number;
  state_id: number;

  name: string;

  display_name_locs: Record<string, string>;

  display_name?: Record<string, string>;
  manpower: number;

  state_category: string;
  resources: Record<string, number>;

  provinces: number[];

  source_snapshot?: StateSourceSnapshot | null;

  membership_provenance?: StateMembershipProvenance;

  source_file?: string | null;
  owner: string;
  controller: string | null;
  add_core_of: string[];

  add_claim_by: string[];

  remove_core_of: string[];
  victory_points: StateVictoryPoint[];

  set_province_controller: StateProvinceController[];

  buildings: Record<string, number>;

  province_buildings: StateProvinceBuildings;
  local_supplies: number | null;

  buildings_max_level_factor: number | null;
  impassable: boolean | null;

  impassable_ignored_links?: string | null;

  force_link_ownership_to?: number | null;

  raw_extras: Record<string, unknown>;
  source: string;
}

export interface StateGeoShape {
  rings: number[][][];

  bbox: [number, number, number, number];

  centroid: [number, number];
}
export interface StatesGeometry {
  format: string;
  schema_version: number;

  source: string;

  game_version: string;
  generated_at?: string;

  factor: number;

  width: number;

  height: number;
  state_count: number;

  states: Record<string, StateGeoShape>;
}

export interface StatesGeometryResponse {
  geometry: StatesGeometry | null;
  source: string;
  available: boolean;
}

export interface StateSeedRequest {
  all?: boolean;
  state_ids?: number[];
}

export interface StateSeedResult {
  seeded: number;
  skipped: number;
  missing: number[];
  warnings: string[];
}

export type StateUpdate = Partial<
  Pick<
    State,
    | "manpower"
    | "state_category"
    | "display_name_locs"
    | "provinces"
    | "resources"
    | "owner"
    | "controller"
    | "add_core_of"
    | "add_claim_by"
    | "remove_core_of"
    | "victory_points"
    | "set_province_controller"
    | "buildings"
    | "province_buildings"
    | "local_supplies"
    | "buildings_max_level_factor"
    | "impassable"
    | "impassable_ignored_links"
    | "force_link_ownership_to"
  >
>;

export type StateCoreMode = "add" | "remove" | "replace_one" | "replace_all";

export interface StateBulkCores {
  mode: StateCoreMode;
  tags: string[];

  from?: string;
}

export interface StateBulkRequest {
  state_db_ids: number[];
  owner?: string;

  controller?: string;

  claims?: StateBulkCores;
  remove_cores?: StateBulkCores;
  cores?: StateBulkCores;

  dry_run?: boolean;
}

export interface StateBulkDetail {
  id: number;
  state_id: number;
  changed: boolean;

  skipped_reason?: string;
  before_owner: string;
  after_owner: string;

  before_controller?: string;
  after_controller?: string;
  before_claims?: string[];
  after_claims?: string[];
  removed_claims?: string[];
  before_remove_cores?: string[];
  after_remove_cores?: string[];
  removed_remove_cores?: string[];
  before_cores: string[];
  after_cores: string[];

  before_source?: string;

  removed_cores?: string[];
}

export interface StateBulkResult {
  dry_run: boolean;
  changed: number;
  skipped: number;
  details: StateBulkDetail[];
}

export interface StateBulkDeleteRequest {
  state_ids: number[];
}

export interface StateBulkDeleteResult {
  deleted: number;

  skipped_created: number[];
}

export interface StateRestoreEntry {
  id: number;
  owner?: string;
  add_core_of?: string[];

  controller?: string;
  add_claim_by?: string[];
  remove_core_of?: string[];

  source?: string;
}
