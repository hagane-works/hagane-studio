import { revisionRequest } from "@/components/workbench/save-diagnostics/revisionRequest";
import type { EditRevisions } from "@/components/workbench/save-diagnostics/editRevisions";
import { API_BASE, request } from "./_client";
import { requestGameIndexed } from "./gameIndex";

export interface BaseModVocabulary {
  advisor_slots: string[];
  idea_types: string[];
}

export interface ProjectCreateInput {
  name: string;
  country_tag?: string;
  country_name?: string;
  is_loaded_existing_mod?: boolean;
  supported_version?: string;
  version?: string;
  template_id?: number;
  imported_descriptor_replace_paths?: string[];

  owned_dlcs?: string[] | null;
}

export interface BaseModFolder {
  name: string;

  folder: string;

  found: boolean;
}

export const projectsApi = {
  list: () =>
    request<{ projects: import("@/types").Project[]; total: number }>(
      "/go/projects",
    ),

  get: (id: number) => request<import("@/types").Project>(`/go/projects/${id}`),

  create: (data: ProjectCreateInput) =>
    request<import("@/types").Project>("/go/projects", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    id: number,
    data: Partial<{
      name: string;
      country_tag: string;
      country_name: string;

      version: string;
      picture: string;
      tags: string[];
      dependencies: string[];
      supported_version: string;
      imported_descriptor_replace_paths: string[];

      start_date: string;
      end_date: string;
      scenario_history_policy:
        "inherit_base_history" | "project_owns_starting_world";

      stage_state_ids: number[] | null;

      export_languages: string[];

      owned_dlcs: string[] | null;

      ideology_registry: import("@/lib/ideology/registry").IdeologyRegistry;
      tech_surface_manifest: import("@/lib/techSurface").TechSurfaceManifest;
    }>,
  ) =>
    request<import("@/types").Project>(`/go/projects/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  delete: (id: number) =>
    request<void>(`/go/projects/${id}`, { method: "DELETE" }),

  baseMods: (id: number) =>
    request<{ base_mods: BaseModFolder[] }>(`/app/projects/${id}/base-mods`),

  setBaseMod: (id: number, folder: string, dependency?: string) =>
    request<{ dependency: string; base_mods: BaseModFolder[] }>(
      `/app/projects/${id}/base-mods`,
      {
        method: "PUT",
        body: JSON.stringify({ dependency, folder }),
      },
    ),

  uploadCover: async (id: number, file: File) => {
    const body = new FormData();
    body.append("file", file);
    const response = await fetch(`${API_BASE}/go/projects/${id}/cover`, {
      method: "POST",
      credentials: "include",
      body,
    });
    if (!response.ok) {
      const error = await response
        .json()
        .catch(() => ({ detail: response.statusText }));
      throw new Error(error.detail || `Request failed: ${response.status}`);
    }
    const result = (await response.json()) as {
      filename: "thumbnail.png";
      preview_url?: string | null;
      size?: number;
    };
    return {
      filename: result.filename,
      previewUrl: result.preview_url ?? null,
      size: result.size,
    };
  },

  coverUrl: (id: number, revision?: string | number) =>
    `${API_BASE}/go/projects/${id}/cover${revision === undefined ? "" : `?v=${revision}`}`,

  deleteCover: (id: number) =>
    request<void>(`/go/projects/${id}/cover`, { method: "DELETE" }),

  baseModVocabulary: (id: number) =>
    request<BaseModVocabulary>(`/go/projects/${id}/base-mod-vocabulary`),

  flipAll: (
    id: number,
    body: { dry_run: boolean } = { dry_run: false },
    revisions: EditRevisions = {},
  ) =>
    revisionRequest<{
      project_id: number;
      country_tag: string;
      flipped: Record<string, number>;
      total: number;
      dry_run: boolean;
    }>(
      id,
      `/go/projects/${id}/flip-all`,
      {
        method: "POST",
        body: JSON.stringify(body),
      },
      revisions,
    ),
};

export interface BaseModTemplate {
  id: number;

  mod_key: string;
  version: string;
  display_name: string;

  status: string;
  workshop_url: string | null;

  descriptor_names: string[];

  index_dep_key: string;
  bucket_prefix: string;
}

export const basemodTemplatesApi = {
  list: () =>
    request<{ templates: BaseModTemplate[]; total: number }>(
      "/go/basemod/templates",
    ),
};

export const tagsApi = {
  search: (q: string = "") =>
    requestGameIndexed<{ tags: import("@/types").CountryTag[]; total: number }>(
      `/go/tags?q=${encodeURIComponent(q)}`,
    ),
};

export interface ProjectUsage {
  project_id: number;
  project_name: string;
  entity_counts: Record<string, number>;
  entity_total: number;
  audit_count: number;
  tokens: {
    scope: string;
    tokens_in_total: number;
    tokens_out_total: number;
    total_calls: number;
    note: string;
  };
}

export interface ProjectAuditLogRow {
  id: number;
  created_at: string | null;
  source: string;
  summary: string;
  entity_count: number;
  entity_by_kind: Record<string, number>;
  undone: boolean;
  undone_at: string | null;
  undo_strategy: string;
}

export interface ProjectAuditLogResponse {
  project_id: number;
  total: number;
  limit: number;
  offset: number;
  logs: ProjectAuditLogRow[];
}

export const projectUsageApi = {
  usage: (projectId: number) =>
    request<ProjectUsage>(`/go/projects/${projectId}/usage`),
  auditLog: (projectId: number, limit = 50, offset = 0) =>
    request<ProjectAuditLogResponse>(
      `/go/projects/${projectId}/audit-log?limit=${limit}&offset=${offset}`,
    ),
};

export interface RefEdge {
  source_kind: string;

  source_id: number;

  source_name: string;

  source_tag: string | null;

  effect_name: string;

  target_type: string;

  target_id: string;

  field_name: string;

  resolved: boolean;

  target_tag: string | null;
}

export interface EntityRelations {
  outgoing: RefEdge[];

  incoming: RefEdge[];
  outgoing_total: number;
  incoming_total: number;
}

export const relationsApi = {
  entity: (projectId: number, kind: string, sourceId: number) =>
    request<EntityRelations>(
      `/go/projects/${projectId}/relations/entity/${encodeURIComponent(kind)}/${sourceId}`,
    ),

  country: (projectId: number, tag: string) =>
    request<EntityRelations>(
      `/go/projects/${projectId}/relations/country/${encodeURIComponent(tag)}`,
    ),
};
