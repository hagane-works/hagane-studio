export type OobSource = "created" | "imported";

export type OpaqueOobAst = Array<Record<string, unknown>>;

export interface AirWingGroup {
  id: number;
  project_id: number;
  oob_file: string;
  source_file: string;
  country_tag: string;
  location: string;
  ordering: number;
  layout: OpaqueOobAst;
  source: OobSource;

  warnings: string[];
}

export interface AirWing {
  id: number;
  project_id: number;
  group_id: number;
  equipment_type: string;
  name: string;
  owner: string;
  amount: string;
  ordering: number;
  body_ast: OpaqueOobAst;
  source: OobSource;
}

export interface Fleet {
  id: number;
  project_id: number;
  oob_file: string;
  source_file: string;
  country_tag: string;
  name: string;
  naval_base: string;
  ordering: number;
  layout: OpaqueOobAst;
  source: OobSource;

  warnings: string[];
}

export interface TaskForce {
  id: number;
  project_id: number;
  fleet_id: number;
  name: string;
  location: string;
  ordering: number;
  layout: OpaqueOobAst;
  source: OobSource;

  warnings: string[];
}

export interface Ship {
  id: number;
  project_id: number;
  task_force_id: number;
  name: string;
  definition: string;
  start_experience_factor: string;
  equipment_type: string;
  amount: string;
  owner: string;
  version_name: string;
  ordering: number;
  body_ast: OpaqueOobAst;
  source: OobSource;
}

export interface OobUnitsSnapshot {
  air_wing_groups: AirWingGroup[];
  air_wings: AirWing[];
  fleets: Fleet[];
  task_forces: TaskForce[];
  ships: Ship[];
}

export type AirWingGroupCreate = Pick<
  AirWingGroup,
  "oob_file" | "source_file" | "country_tag" | "location" | "ordering"
>;

export type AirWingGroupUpdate = Partial<AirWingGroupCreate>;

export type AirWingCreate = Pick<
  AirWing,
  "group_id" | "equipment_type" | "name" | "owner" | "amount" | "ordering"
>;

export type AirWingUpdate = Partial<AirWingCreate>;

export type FleetCreate = Pick<
  Fleet,
  | "oob_file"
  | "source_file"
  | "country_tag"
  | "name"
  | "naval_base"
  | "ordering"
>;

export type FleetUpdate = Partial<FleetCreate>;

export type TaskForceCreate = Pick<
  TaskForce,
  "fleet_id" | "name" | "location" | "ordering"
>;

export type TaskForceUpdate = Partial<TaskForceCreate>;

export type ShipCreate = Pick<
  Ship,
  | "task_force_id"
  | "name"
  | "definition"
  | "start_experience_factor"
  | "equipment_type"
  | "amount"
  | "owner"
  | "version_name"
  | "ordering"
>;

export type ShipUpdate = Partial<ShipCreate>;

export interface OobUnitsImportRequest {
  oob_file: string;
  source_file: string;
  country_tag: string;
  text: string;
  policy: "overwrite" | "skip";
}

export interface OobUnitsImportResult {
  air_wing_groups: number;
  air_wings: number;
  fleets: number;
  task_forces: number;
  ships: number;
  skipped: number;
  warnings: string[];
}

export interface OobUnitsPreview {
  script: string;
}
