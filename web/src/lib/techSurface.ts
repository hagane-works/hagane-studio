import type { Technology } from "@/types";

export const TECH_SURFACE_ENTRANCE_IDS = [
  "army",
  "air",
  "navy",
  "industry_equipment",
] as const;

export type TechSurfaceEntranceID = (typeof TECH_SURFACE_ENTRANCE_IDS)[number];

export interface TechSurfaceBranch {
  id: string;
  surface: TechSurfaceEntranceID;
  folder: string;
  slot: string;
  root_technology_id: string;
  technology_ids: string[];
  placements?: { folder: string; slot: string }[];
}

export interface TechSurfaceManifest {
  schema_version: number;
  profile: string;
  baseline_sha256: string;
  compatibility_mode?: TechSurfaceCompatibilityMode;
  branches: TechSurfaceBranch[];
}

export interface TechSurfaceDLCState {
  no_step_back: boolean;
  by_blood_alone: boolean;
  man_the_guns: boolean;
}

export type TechSurfaceCompatibilityMode = "all_players" | "requires_dlc";

export interface TechSurfaceSourceFile {
  path: string;
  sha256: string;
}

export interface TechSurfaceEnvironmentStatus {
  state: string;
  code?: string;
  detail: string;
  source: { files: TechSurfaceSourceFile[] };
  source_connection: TechSurfaceSourceConnection;
  dlc_confirmed: boolean;
  dlc: TechSurfaceDLCState;
  compatibility_mode: TechSurfaceCompatibilityMode;
}

export type TechSurfaceSourceKind =
  "none" | "platform_certified" | "local_hoi4" | "project_mod";

export interface TechSurfaceSourceConnection {
  state: "not_connected" | "connected" | "project_source";
  kind: TechSurfaceSourceKind;
  label?: string;
  fingerprint?: string;
  reused?: boolean;
  connected_at?: string;
}

export interface TechSurfaceTargetCertification {
  state: string;
  evidence_id: string;
  detail?: string;
}

export interface TechSurfaceProfileTarget {
  id: string;
  folder_id: string;
  authoring_enabled: boolean;
  certification: TechSurfaceTargetCertification;
  timeline_years: number[];
  all_players_supported: boolean;
}

export interface TechSurfaceProfileEntry {
  id: TechSurfaceEntranceID;
  ledger: string;
  targets: TechSurfaceProfileTarget[];
}

export interface TechSurfaceProfile {
  id: string;
  entries: TechSurfaceProfileEntry[];
}

export interface TechSurfaceStatus {
  enabled: boolean;
  baseline_attached: boolean;
  legacy_timeline_repair_needed: boolean;
  manifest: TechSurfaceManifest;
  environment: TechSurfaceEnvironmentStatus;
  active_profile?: TechSurfaceProfile;
}

export function techSurfaceBranches(
  status: TechSurfaceStatus | null | undefined,
): TechSurfaceBranch[] {
  return Array.isArray(status?.manifest?.branches)
    ? status.manifest.branches
    : [];
}

export interface TechSurfaceEnvironmentConfirmInput {
  dlc: TechSurfaceDLCState;
  compatibility_mode: TechSurfaceCompatibilityMode;
}

export interface TechSurfaceBranchCreateInput {
  id: string;
  entry_id: TechSurfaceEntranceID;
  target_id: string;
  root: Partial<Technology>;
}

export interface TechSurfaceBranchResult {
  manifest: TechSurfaceManifest;
  branch: TechSurfaceBranch;
  root: Technology;
}

export interface TechSurfaceBranchTechnologyCreateInput {
  parent_technology_id: string;
  technology: Partial<Technology>;
}

export interface TechSurfaceBranchTechnologyResult {
  manifest: TechSurfaceManifest;
  branch: TechSurfaceBranch;
  technology: Technology;
}

export interface TechSurfaceRepairResult {
  manifest: TechSurfaceManifest;
  repaired_technology_ids: string[];
}

export function buildTechSurfaceEnvironmentConfirmInput(
  dlc: TechSurfaceDLCState,
  compatibilityMode: TechSurfaceCompatibilityMode,
): TechSurfaceEnvironmentConfirmInput {
  return {
    dlc: {
      no_step_back: dlc.no_step_back,
      by_blood_alone: dlc.by_blood_alone,
      man_the_guns: dlc.man_the_guns,
    },
    compatibility_mode: compatibilityMode,
  };
}

export function buildTechSurfaceBranchCreateInput(args: {
  id: string;
  entryID: TechSurfaceEntranceID;
  targetID: string;
  root: Partial<Technology>;
}): TechSurfaceBranchCreateInput {
  return {
    id: args.id,
    entry_id: args.entryID,
    target_id: args.targetID,
    root: args.root,
  };
}

export function isTechSurfaceTargetSelectable(
  target: TechSurfaceProfileTarget,
  compatibilityMode: TechSurfaceCompatibilityMode = "requires_dlc",
): boolean {
  return (
    target.authoring_enabled &&
    target.certification.state === "engine_verified_geometry" &&
    (compatibilityMode !== "all_players" || target.all_players_supported)
  );
}

export function canConfirmTechSurfaceEnvironment(
  status: TechSurfaceStatus,
): boolean {
  return (
    status.baseline_attached &&
    (status.environment.state === "dlc_unconfirmed" ||
      status.environment.state === "ready")
  );
}

export function techSurfaceTargetForBranch(
  profile: TechSurfaceProfile | undefined,
  branch: TechSurfaceBranch,
): TechSurfaceProfileTarget | undefined {
  return profile?.entries
    .find((entry) => entry.id === branch.surface)
    ?.targets.find((target) => target.folder_id === branch.folder);
}

export function isTechSurfaceEditableLayout(
  layoutMode: boolean,
  surfaceManaged: boolean,
): boolean {
  return layoutMode && !surfaceManaged;
}
