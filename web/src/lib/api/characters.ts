import { request, API_BASE } from "./_client";

export const charactersApi = {
  list: (projectId: number) =>
    request<import("@/types").Character[]>(
      `/go/projects/${projectId}/characters`,
    ),

  get: (projectId: number, id: number) =>
    request<import("@/types").Character>(
      `/go/projects/${projectId}/characters/${id}`,
    ),

  create: (projectId: number, data: Partial<import("@/types").Character>) =>
    request<import("@/types").Character>(
      `/go/projects/${projectId}/characters`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    ),

  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").Character>,
  ) =>
    request<import("@/types").Character>(
      `/go/projects/${projectId}/characters/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),

  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/characters/${id}`, {
      method: "DELETE",
    }),

  preview: (projectId: number, id: number) =>
    request<{ script: string }>(
      `/go/projects/${projectId}/characters/${id}/preview`,
    ),

  previewFiles: (projectId: number, id: number) =>
    request<{
      files: Record<string, { path: string; content: string }>;
    }>(`/go/projects/${projectId}/characters/${id}/preview-files`),

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
    }>(`/go/projects/${projectId}/characters/${id}/parse-preview`, {
      method: "POST",
      body: JSON.stringify({ script }),
    }),

  fork: (projectId: number, id: number, newTag: string) =>
    request<import("@/types").Character>(
      `/go/projects/${projectId}/characters/${id}/fork`,
      {
        method: "POST",
        body: JSON.stringify({ new_tag: newTag }),
      },
    ),

  uploadPortrait: async (
    projectId: number,
    characterId: number,
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
      `${API_BASE}/go/projects/${projectId}/characters/${characterId}/upload-portrait`,
      {
        method: "POST",
        body: formData,
      },
    );
    if (!res.ok) throw new Error("Upload failed");
    return res.json();
  },

  resetPortrait: (projectId: number, characterId: number) =>
    request<{ ok: boolean; deleted: string[]; is_portrait_modified: false }>(
      `/go/projects/${projectId}/characters/${characterId}/portrait`,
      { method: "DELETE" },
    ),

  attachAnimatedPortrait: async (
    projectId: number,
    characterId: number,
    input: {
      gfxName: string;
      smallGfxName: string;
      frameCount: number;
      fps: number;
      largeDds: Blob;
      smallDds: Blob;
      previewPng: Blob;
    },
  ) => {
    const formData = new FormData();
    formData.append("gfx_name", input.gfxName);
    formData.append("small_gfx_name", input.smallGfxName);
    formData.append("frame_count", String(input.frameCount));
    formData.append("fps", String(input.fps));
    formData.append("large_dds", input.largeDds, "portrait_animated.dds");
    formData.append("small_dds", input.smallDds, "portrait_small.dds");
    formData.append("preview_png", input.previewPng, "preview.png");
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/characters/${characterId}/attach-animated-portrait`,
      { method: "POST", body: formData },
    );
    if (!res.ok) throw new Error("Attach failed");
    return res.json() as Promise<{
      ok: boolean;
      gfx_name: string;
      small_gfx_name: string;
      frame_count: number;
      fps: number;
    }>;
  },
};

export const customTraitsApi = {
  uploadIcon: async (
    projectId: number,
    traitDbId: number,
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
      `${API_BASE}/go/projects/${projectId}/custom-traits/${traitDbId}/upload-icon`,
      { method: "POST", body: formData },
    );
    if (!res.ok) {
      const detail = await res.text();
      throw new Error(`Upload failed: ${detail || res.statusText}`);
    }
    return res.json() as Promise<{
      filename: string;
      sprite: string;
      preview_url: string;
      warnings: string[];
    }>;
  },

  iconUrl: (projectId: number, traitDbId: number) =>
    `${API_BASE}/go/projects/${projectId}/custom-traits/${traitDbId}/icon`,
};
