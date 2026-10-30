import { request } from "./_client";

export type PoliticsIdeologyKind = "root" | "subtype";

export type PoliticsSourceOwnership =
  "project_managed" | "vanilla" | "imported" | "base_mod";

export interface PoliticsTarget {
  kind: PoliticsIdeologyKind;
  script_id: string;
}

export interface PoliticsIdeology {
  kind: PoliticsIdeologyKind;
  script_id: string;
  root_id?: string;
  display_name?: string;
  source: PoliticsSourceOwnership;
}

export interface PoliticsEntityReference {
  kind: string;
  id: string;
  path: string;
}

export interface PoliticsReference {
  target: PoliticsTarget;
  entity: PoliticsEntityReference;
  context?: string;
}

export interface PoliticsRootDependency {
  custom_root_id: string;
  expected_root: string;
  reference_key: string;
  disposition: "required_compatibility" | "intentional_design";
  entity: PoliticsEntityReference;
}

export type PoliticsConflictReason =
  | "duplicate_script_id"
  | "missing_root"
  | "unknown_root"
  | "malformed_definition";

export interface PoliticsIdeologyConflict {
  script_id: string;
  kind?: PoliticsIdeologyKind;
  root_id?: string;
  reason: PoliticsConflictReason;

  detail: string;

  usage_count: number;
}

export interface PoliticsReferenceSnapshot {
  project_id: string;
  ideologies: PoliticsIdeology[];
  references?: PoliticsReference[];
  dependencies?: PoliticsRootDependency[];
  conflicts?: PoliticsIdeologyConflict[];
}

export interface PoliticsOwnershipDecision {
  allowed: boolean;
  action: "edit" | "delete" | "takeover";
  target: PoliticsTarget;
  source: PoliticsSourceOwnership;
  reason?: string;
}

export interface PoliticsDeletePreview {
  token: string;
  snapshot_hash: string;
  target: PoliticsTarget;
  allowed: boolean;
  ownership: PoliticsOwnershipDecision;
  impacts: PoliticsReference[];
  automatic_rewrites: false;
}

export interface PoliticsDeleteApplyInput {
  preview_token: string;
}

export interface PoliticsDeleteAuthorization {
  authorized: boolean;
  snapshot_hash: string;
  target: PoliticsTarget;
  impacts: PoliticsReference[];
  automatic_rewrites: false;
}

export interface PoliticsDefinition {
  kind: PoliticsIdeologyKind;
  script_id: string;
  root_id?: string;
  body?: string;
  name: Record<string, string>;
  desc?: Record<string, string>;
  random_disabled?: boolean;
  source: PoliticsSourceOwnership;
}

export interface PoliticsRootCreateInput {
  script_id: string;
  body: string;
  name: Record<string, string>;
  desc?: Record<string, string>;
}

export interface PoliticsSubtypeCreateInput {
  script_id: string;
  root_id: string;
  name: Record<string, string>;
  desc?: Record<string, string>;
  random_disabled: boolean;
}

export interface PoliticsDefinitionPatch {
  name?: Record<string, string>;
  desc?: Record<string, string>;
  body?: string;
  random_disabled?: boolean;
}

export const politicsApi = {
  references: (projectId: number, signal?: AbortSignal) =>
    request<PoliticsReferenceSnapshot>(
      `/go/projects/${projectId}/politics/references`,
      { signal },
    ),

  discardConflict: (projectId: number, scriptId: string) =>
    request<PoliticsIdeologyConflict>(
      `/go/projects/${projectId}/politics/conflicts/${encodeURIComponent(scriptId)}`,
      { method: "DELETE" },
    ),

  createRoot: (projectId: number, input: PoliticsRootCreateInput) =>
    request<PoliticsDefinition>(`/go/projects/${projectId}/politics/roots`, {
      method: "POST",
      body: JSON.stringify(input),
    }),

  createSubtype: (projectId: number, input: PoliticsSubtypeCreateInput) =>
    request<PoliticsDefinition>(`/go/projects/${projectId}/politics/subtypes`, {
      method: "POST",
      body: JSON.stringify(input),
    }),

  updateDefinition: (
    projectId: number,
    target: PoliticsTarget,
    patch: PoliticsDefinitionPatch,
  ) =>
    request<PoliticsDefinition>(
      `/go/projects/${projectId}/politics/definitions/${target.kind}/${encodeURIComponent(target.script_id)}`,
      { method: "PATCH", body: JSON.stringify(patch) },
    ),

  deletePreview: (projectId: number, target: PoliticsTarget) =>
    request<PoliticsDeletePreview>(
      `/go/projects/${projectId}/politics/delete-preview`,
      { method: "POST", body: JSON.stringify(target) },
    ),

  deleteApply: (projectId: number, input: PoliticsDeleteApplyInput) =>
    request<PoliticsDeleteAuthorization>(
      `/go/projects/${projectId}/politics/delete-apply`,
      {
        method: "POST",
        body: JSON.stringify(input),
      },
    ),
};
