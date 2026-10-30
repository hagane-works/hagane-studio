import { request, API_BASE } from "./_client";
import type { FocusScriptPreview } from "@/components/workbench/save-diagnostics/focusPreviewSnapshot";
import {
  revisionRequest,
  revisionFetch,
} from "@/components/workbench/save-diagnostics/revisionRequest";
import {
  editRevisionSnapshot,
  withoutEditRevisions,
  type EditRevisions,
} from "@/components/workbench/save-diagnostics/editRevisions";

function focusWrite<T>(
  projectId: number,
  treeId: number,
  suffix: string,
  method: string,
  data: object | undefined,
  revision?: string,
  followPending = false,
) {
  return revisionRequest<T>(
    projectId,
    `/go/projects/${projectId}/focus-trees/${treeId}${suffix}`,
    {
      method,
      ...(data ? { body: JSON.stringify(withoutEditRevisions(data)) } : {}),
    },
    editRevisionSnapshot(`focus_tree:${treeId}`, revision),
    followPending,
  );
}

export function isAmbiguousLegacyInitialShowPosition(
  value: string | null | undefined,
): boolean {
  return /^[+-]?\d+$/.test((value ?? "").trim());
}

export function buildInitialShowPositionUpdate(
  position: import("@/types").InitialShowPosition | null,
): Pick<
  import("@/types").FocusTree,
  "initial_show_position" | "initial_show_focus"
> {
  return {
    initial_show_position: position,
    initial_show_focus: "",
  };
}

export const focusTreesApi = {
  list: (projectId: number, signal?: AbortSignal) =>
    request<import("@/types").FocusTree[]>(
      `/go/projects/${projectId}/focus-trees`,
      { signal },
    ),

  findByScriptId: (projectId: number, scriptId: string) =>
    request<import("@/types").FocusTree[]>(
      `/go/projects/${projectId}/focus-trees?tree_id=${encodeURIComponent(scriptId)}`,
    ).then((trees) => trees[0] ?? null),

  get: (projectId: number, treeId: number) =>
    request<import("@/types").FocusTree>(
      `/go/projects/${projectId}/focus-trees/${treeId}`,
    ),

  create: (projectId: number, data?: Partial<import("@/types").FocusTree>) =>
    request<import("@/types").FocusTree>(
      `/go/projects/${projectId}/focus-trees`,
      {
        method: "POST",
        body: JSON.stringify(data || {}),
      },
    ),

  update: (
    projectId: number,
    treeId: number,
    data: Partial<import("@/types").FocusTree>,
    revision = data.edit_revision,
  ) =>
    focusWrite<import("@/types").FocusTree>(
      projectId,
      treeId,
      "",
      "PUT",
      data,
      revision,
    ),

  delete: (projectId: number, treeId: number, revision?: string) =>
    focusWrite<void>(projectId, treeId, "", "DELETE", undefined, revision),

  preview: (
    projectId: number,
    treeId: number,
    nodeId?: number,
    signal?: AbortSignal,
  ) =>
    request<FocusScriptPreview>(
      `/go/projects/${projectId}/focus-trees/${treeId}/preview${nodeId ? `?node_id=${nodeId}` : ""}`,
      { signal },
    ),

  getNode: (projectId: number, treeId: number, nodeId: number) =>
    request<import("@/types").FocusNode>(
      `/go/projects/${projectId}/focus-trees/${treeId}/nodes/${nodeId}`,
    ),

  createNode: (
    projectId: number,
    treeId: number,
    data?: import("@/types").FocusNodeCreateInput,
    revision = data?.tree_edit_revision,
  ) =>
    focusWrite<import("@/types").FocusNode>(
      projectId,
      treeId,
      "/nodes",
      "POST",
      data || {},
      revision,
    ),

  updateNode: (
    projectId: number,
    treeId: number,
    nodeId: number,
    data: Partial<import("@/types").FocusNode>,
    revision = data.tree_edit_revision,
    followPending = false,
  ) =>
    focusWrite<import("@/types").FocusNode>(
      projectId,
      treeId,
      `/nodes/${nodeId}`,
      "PUT",
      data,
      revision,
      followPending,
    ),

  bulkUpdateNodePositions: (
    projectId: number,
    treeId: number,
    positions: Array<{ id: number; x: number; y: number }>,
    revision?: string,
    followPending = false,
  ) =>
    focusWrite<import("@/types").FocusNode[]>(
      projectId,
      treeId,
      "/node-positions",
      "PUT",
      { positions },
      revision,
      followPending,
    ),

  deleteNode: (
    projectId: number,
    treeId: number,
    nodeId: number,
    revision?: string,
  ) =>
    focusWrite<void>(
      projectId,
      treeId,
      `/nodes/${nodeId}`,
      "DELETE",
      undefined,
      revision,
    ),

  setMutualExclusion: (
    projectId: number,
    treeId: number,
    nodeId: number,
    otherNodeId: number,
    enabled: boolean,
    revision?: string,
  ) =>
    focusWrite<import("@/types").FocusNode[]>(
      projectId,
      treeId,
      `/nodes/${nodeId}/mutual-exclusion`,
      "PUT",
      { other_node_id: otherNodeId, enabled },
      revision,
    ),

  forkNode: (
    projectId: number,
    treeId: number,
    nodeId: number,
    newFocusId: string,
    revision?: string,
  ) =>
    focusWrite<import("@/types").FocusNode>(
      projectId,
      treeId,
      `/nodes/${nodeId}/fork`,
      "POST",
      { new_focus_id: newFocusId },
      revision,
    ),

  parseNodePreview: (
    projectId: number,
    treeId: number,
    nodeId: number,
    script: string,
  ) =>
    request<{
      entity: Record<string, unknown> | null;
      extras: Record<string, unknown>;
      warnings: string[];
      parse_errors: {
        line: number;
        msg: string;
        severity: "error" | "warning" | "info";
      }[];
    }>(
      `/go/projects/${projectId}/focus-trees/${treeId}/nodes/${nodeId}/parse-preview`,
      {
        method: "POST",
        body: JSON.stringify({ script }),
      },
    ),

  uploadNodeIcon: async (
    projectId: number,
    treeId: number,
    nodeId: number,
    file: File,
    crop?: { x: number; y: number; width: number; height: number },
    revision?: string,
    cohort: EditRevisions = {},
  ) => {
    const formData = new FormData();
    formData.append("file", file);
    if (crop) {
      formData.append("crop_x", String(crop.x));
      formData.append("crop_y", String(crop.y));
      formData.append("crop_width", String(crop.width));
      formData.append("crop_height", String(crop.height));
    }
    const res = await revisionFetch(
      projectId,
      `${API_BASE}/go/projects/${projectId}/focus-trees/${treeId}/nodes/${nodeId}/upload-icon`,
      { method: "POST", body: formData },
      { ...cohort, ...editRevisionSnapshot(`focus_tree:${treeId}`, revision) },
    );
    if (!res.ok) {
      const detail = await res.text();
      throw new Error(`Upload failed: ${detail || res.statusText}`);
    }
    return res.json() as Promise<{
      filename: string;
      icon: string;
      preview_url: string;
    }>;
  },

  nodeIconUrl: (projectId: number, gfxName: string) =>
    `${API_BASE}/go/projects/${projectId}/focus-icons/${encodeURIComponent(gfxName)}`,
};

export interface FocusIconLibraryEntry {
  name: string;
  thumbnail_url: string;
  source: "vanilla" | "project";
  dlc: string | null;
  category: string | null;
}

export interface FocusIconLibraryResponse {
  icons: FocusIconLibraryEntry[];
  total: number;
  returned: number;
  offset: number;
  limit: number;
}

export const focusIconsApi = {
  list: (projectId: number) =>
    request<{ icons: string[] }>(`/go/projects/${projectId}/focus-icons`),

  getUrl: (projectId: number, gfxName: string) =>
    `${API_BASE}/go/projects/${projectId}/focus-icons/${encodeURIComponent(gfxName)}`,

  library: (
    projectId: number,
    params: {
      source?: "vanilla" | "project" | "all";
      q?: string;
      limit?: number;
      offset?: number;
    } = {},
  ) => {
    const qs = new URLSearchParams();
    if (params.source) qs.set("source", params.source);
    if (params.q) qs.set("q", params.q);
    if (params.limit !== undefined) qs.set("limit", String(params.limit));
    if (params.offset !== undefined) qs.set("offset", String(params.offset));
    const suffix = qs.toString() ? `?${qs.toString()}` : "";
    return request<FocusIconLibraryResponse>(
      `/go/projects/${projectId}/focus-icons/library${suffix}`,
    );
  },

  uploadNamed: async (
    projectId: number,
    gfxName: string,
    file: File,
    revisions: EditRevisions = {},
  ): Promise<{ gfx_name: string; saved: boolean }> => {
    const formData = new FormData();
    formData.append("gfx_name", gfxName);
    formData.append("file", file);
    const res = await revisionFetch(
      projectId,
      `${API_BASE}/go/projects/${projectId}/focus-icons/upload-named`,
      { method: "POST", body: formData },
      revisions,
    );
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.detail || `Upload failed: HTTP ${res.status}`);
    }
    return res.json() as Promise<{ gfx_name: string; saved: boolean }>;
  },
};

export const technologiesApi = {
  list: (projectId: number, techLine?: string) =>
    request<import("@/types").Technology[]>(
      `/go/projects/${projectId}/technologies${techLine ? `?tech_line=${techLine}` : ""}`,
    ),

  create: (projectId: number, data: Partial<import("@/types").Technology>) =>
    request<import("@/types").Technology>(
      `/go/projects/${projectId}/technologies`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    ),

  surfaceStatus: (projectId: number) =>
    request<import("@/lib/techSurface").TechSurfaceStatus>(
      `/go/projects/${projectId}/technologies/surface`,
    ),

  confirmSurfaceEnvironment: (
    projectId: number,
    data: import("@/lib/techSurface").TechSurfaceEnvironmentConfirmInput,
  ) =>
    request<import("@/lib/techSurface").TechSurfaceStatus>(
      `/go/projects/${projectId}/technologies/surface/environment`,
      { method: "PUT", body: JSON.stringify(data) },
    ),

  markSurfaceSourceConnected: (projectId: number, label: string) =>
    request<import("@/lib/techSurface").TechSurfaceSourceConnection>(
      `/go/projects/${projectId}/technologies/surface/source`,
      { method: "POST", body: JSON.stringify({ label }) },
    ),

  attachCertifiedSurfaceSource: (projectId: number) =>
    request<import("@/lib/techSurface").TechSurfaceSourceConnection>(
      `/go/projects/${projectId}/technologies/surface/source/certified`,
      { method: "POST", body: JSON.stringify({}) },
    ),

  repairSurfaceTimeline: (projectId: number) =>
    request<import("@/lib/techSurface").TechSurfaceRepairResult>(
      `/go/projects/${projectId}/technologies/surface/repair`,
      { method: "POST", body: JSON.stringify({}) },
    ),

  createSurfaceBranch: (
    projectId: number,
    data: import("@/lib/techSurface").TechSurfaceBranchCreateInput,
  ) =>
    request<import("@/lib/techSurface").TechSurfaceBranchResult>(
      `/go/projects/${projectId}/technologies/surface/branches`,
      { method: "POST", body: JSON.stringify(data) },
    ),

  createSurfaceBranchTechnology: (
    projectId: number,
    branchId: string,
    data: import("@/lib/techSurface").TechSurfaceBranchTechnologyCreateInput,
  ) =>
    request<import("@/lib/techSurface").TechSurfaceBranchTechnologyResult>(
      `/go/projects/${projectId}/technologies/surface/branches/${encodeURIComponent(branchId)}/technologies`,
      { method: "POST", body: JSON.stringify(data) },
    ),

  deleteSurfaceBranch: (projectId: number, branchId: string) =>
    request<void>(
      `/go/projects/${projectId}/technologies/surface/branches/${encodeURIComponent(branchId)}`,
      { method: "DELETE" },
    ),

  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").Technology>,
  ) =>
    request<import("@/types").Technology>(
      `/go/projects/${projectId}/technologies/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),

  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/technologies/${id}`, {
      method: "DELETE",
    }),

  parsePreview: (projectId: number, id: number, script: string) =>
    request<{
      entity: Record<string, unknown> | null;
      extras: Record<string, unknown>;
      warnings: string[];
      parse_errors: {
        line: number;
        msg: string;
        severity: "error" | "warning" | "info";
      }[];
    }>(`/go/projects/${projectId}/technologies/${id}/parse-preview`, {
      method: "POST",
      body: JSON.stringify({ script }),
    }),

  preview: (projectId: number, techLine: string, techId?: number) =>
    request<{ script: string }>(
      `/go/projects/${projectId}/technologies/preview/${techLine}${techId ? `?tech_id=${techId}` : ""}`,
    ),

  uploadIcon: async (
    projectId: number,
    techRowId: number,
    file: File,
    crop?: { x: number; y: number; width: number; height: number },
  ) => {
    const formData = new FormData();
    formData.append("file", file);
    if (crop) {
      formData.append("crop_x", String(crop.x));
      formData.append("crop_y", String(crop.y));
      formData.append("crop_width", String(crop.width));
      formData.append("crop_height", String(crop.height));
    }
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/technologies/${techRowId}/upload-icon`,
      {
        method: "POST",
        body: formData,
      },
    );
    if (!res.ok) {
      const detail = await res.text();
      throw new Error(`Upload failed: ${detail || res.statusText}`);
    }
    return res.json() as Promise<{
      filename: string;
      icon: string;
      preview_url: string;
    }>;
  },

  iconUrl: (projectId: number, techIdString: string) =>
    `${API_BASE}/go/projects/${projectId}/tech-icons/${encodeURIComponent(techIdString)}`,

  listIcons: (projectId: number) =>
    request<{ icons: string[] }>(`/go/projects/${projectId}/tech-icons`),

  reachability: (projectId: number, countryTag: string) =>
    request<{ country_tag: string; tech_ids: string[] }>(
      `/go/projects/${projectId}/technologies/reachability/${encodeURIComponent(countryTag)}`,
    ),

  reachabilityPaths: (
    projectId: number,
    countryTag: string,
    techIdStr: string,
  ) =>
    request<{
      country_tag: string;
      tech_id: string;
      paths: {
        source_kind: "focus" | "event" | "decision";
        source_id: number;
        source_name: string;
        field_name: string;
      }[];
    }>(
      `/go/projects/${projectId}/technologies/reachability/${encodeURIComponent(countryTag)}/${encodeURIComponent(techIdStr)}/paths`,
    ),
};
