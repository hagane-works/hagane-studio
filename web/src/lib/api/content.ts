import { request, API_BASE } from "./_client";
import { normalizeGameEvent } from "./eventPayload";

export const ideasApi = {
  list: (projectId: number, ideaType?: string) => {
    const q = ideaType ? `?idea_type=${ideaType}` : "";
    return request<import("@/types").Idea[]>(
      `/go/projects/${projectId}/ideas${q}`,
    );
  },

  create: (projectId: number, data: Partial<import("@/types").Idea>) =>
    request<import("@/types").Idea>(`/go/projects/${projectId}/ideas`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").Idea>,
  ) =>
    request<import("@/types").Idea>(`/go/projects/${projectId}/ideas/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/ideas/${id}`, {
      method: "DELETE",
    }),

  fork: (projectId: number, id: number, newIdeaId: string) =>
    request<import("@/types").Idea>(
      `/go/projects/${projectId}/ideas/${id}/fork`,
      {
        method: "POST",
        body: JSON.stringify({ new_idea_id: newIdeaId }),
      },
    ),

  preview: (projectId: number, ideaId?: number) =>
    request<{ script: string }>(
      `/go/projects/${projectId}/ideas/preview${ideaId ? `?idea_id=${ideaId}` : ""}`,
    ),

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
    }>(`/go/projects/${projectId}/ideas/${id}/parse-preview`, {
      method: "POST",
      body: JSON.stringify({ script }),
    }),

  uploadPicture: async (
    projectId: number,
    ideaId: number,
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
      `${API_BASE}/go/projects/${projectId}/ideas/${ideaId}/upload-picture`,
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
      picture: string;
      preview_url: string;
    }>;
  },

  pictureUrl: (projectId: number, ideaId: number) =>
    `${API_BASE}/go/projects/${projectId}/ideas/${ideaId}/picture`,
};

export const eventsApi = {
  listNamespaces: (projectId: number) =>
    request<import("@/types").EventNamespace[]>(
      `/go/projects/${projectId}/events/namespaces`,
    ),

  createNamespace: (
    projectId: number,
    data: { namespace: string; country_tag?: string },
  ) =>
    request<import("@/types").EventNamespace>(
      `/go/projects/${projectId}/events/namespaces`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    ),

  updateNamespace: (
    projectId: number,
    nsId: number,
    data: Partial<{ namespace: string; country_tag: string }>,
  ) =>
    request<import("@/types").EventNamespace>(
      `/go/projects/${projectId}/events/namespaces/${nsId}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),

  deleteNamespace: (projectId: number, nsId: number) =>
    request<void>(`/go/projects/${projectId}/events/namespaces/${nsId}`, {
      method: "DELETE",
    }),

  previewNamespace: (projectId: number, nsId: number) =>
    request<{ script: string }>(
      `/go/projects/${projectId}/events/namespaces/${nsId}/preview`,
    ),

  list: (projectId: number, namespaceId?: number) => {
    const q = namespaceId ? `?namespace_id=${namespaceId}` : "";
    return request<import("@/types").GameEvent[]>(
      `/go/projects/${projectId}/events${q}`,
    ).then((events) => events.map(normalizeGameEvent));
  },

  get: (projectId: number, id: number) =>
    request<import("@/types").GameEvent>(
      `/go/projects/${projectId}/events/${id}`,
    ).then(normalizeGameEvent),

  findByScriptId: (projectId: number, eventId: string) =>
    request<import("@/types").GameEvent[]>(
      `/go/projects/${projectId}/events?event_id=${encodeURIComponent(eventId)}`,
    ).then((events) => (events[0] ? normalizeGameEvent(events[0]) : null)),

  create: (projectId: number, data: Partial<import("@/types").GameEvent>) =>
    request<import("@/types").GameEvent>(`/go/projects/${projectId}/events`, {
      method: "POST",
      body: JSON.stringify(data),
    }).then(normalizeGameEvent),

  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").GameEvent>,
  ) =>
    request<import("@/types").GameEvent>(
      `/go/projects/${projectId}/events/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ).then(normalizeGameEvent),

  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/events/${id}`, {
      method: "DELETE",
    }),

  preview: (projectId: number, id: number) =>
    request<{ script: string }>(
      `/go/projects/${projectId}/events/${id}/preview`,
    ),

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
    }>(`/go/projects/${projectId}/events/${id}/parse-preview`, {
      method: "POST",
      body: JSON.stringify({ script }),
    }),

  fork: (projectId: number, id: number, newEventId: string) =>
    request<import("@/types").GameEvent>(
      `/go/projects/${projectId}/events/${id}/fork`,
      {
        method: "POST",
        body: JSON.stringify({ new_event_id: newEventId }),
      },
    ),

  uploadPicture: async (
    projectId: number,
    eventId: number,
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
      `${API_BASE}/go/projects/${projectId}/events/${eventId}/upload-picture`,
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
      picture: string;
      preview_url: string;
    }>;
  },

  pictureUrl: (projectId: number, eventId: number) =>
    `${API_BASE}/go/projects/${projectId}/events/${eventId}/picture`,

  lint: (projectId: number, eventId: number) =>
    request<{
      warnings: {
        code: string;
        message: string;
        severity: "warning" | "error" | "info";
      }[];
    }>(`/go/projects/${projectId}/events/${eventId}/lint`),
};

export const decisionsApi = {
  listCategories: (projectId: number) =>
    request<import("@/types").DecisionCategory[]>(
      `/go/projects/${projectId}/decisions/categories`,
    ),

  createCategory: (
    projectId: number,
    data: { category_id: string; country_tag?: string; icon?: string },
  ) =>
    request<import("@/types").DecisionCategory>(
      `/go/projects/${projectId}/decisions/categories`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    ),

  updateCategory: (
    projectId: number,
    catId: number,
    data: Partial<import("@/types").DecisionCategory>,
  ) =>
    request<import("@/types").DecisionCategory>(
      `/go/projects/${projectId}/decisions/categories/${catId}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),

  deleteCategory: (projectId: number, catId: number) =>
    request<void>(`/go/projects/${projectId}/decisions/categories/${catId}`, {
      method: "DELETE",
    }),

  previewCategory: (projectId: number, catId: number) =>
    request<{ script: string }>(
      `/go/projects/${projectId}/decisions/categories/${catId}/preview`,
    ),

  uploadCategoryIcon: async (
    projectId: number,
    catId: number,
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
      `${API_BASE}/go/projects/${projectId}/decisions/categories/${catId}/upload-icon`,
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

  categoryIconUrl: (projectId: number, catId: number) =>
    `${API_BASE}/go/projects/${projectId}/decisions/categories/${catId}/icon`,

  list: (projectId: number, categoryId?: number) => {
    const q = categoryId ? `?category_id=${categoryId}` : "";
    return request<import("@/types").Decision[]>(
      `/go/projects/${projectId}/decisions${q}`,
    );
  },

  get: (projectId: number, id: number) =>
    request<import("@/types").Decision>(
      `/go/projects/${projectId}/decisions/${id}`,
    ),

  findByScriptId: (projectId: number, decisionId: string) =>
    request<import("@/types").Decision[]>(
      `/go/projects/${projectId}/decisions?decision_id=${encodeURIComponent(decisionId)}`,
    ).then((decisions) => decisions[0] ?? null),

  create: (projectId: number, data: Partial<import("@/types").Decision>) =>
    request<import("@/types").Decision>(`/go/projects/${projectId}/decisions`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").Decision>,
  ) =>
    request<import("@/types").Decision>(
      `/go/projects/${projectId}/decisions/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),

  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/decisions/${id}`, {
      method: "DELETE",
    }),

  preview: (projectId: number, id: number) =>
    request<{ script: string }>(
      `/go/projects/${projectId}/decisions/${id}/preview`,
    ),

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
    }>(`/go/projects/${projectId}/decisions/${id}/parse-preview`, {
      method: "POST",
      body: JSON.stringify({ script }),
    }),

  fork: (projectId: number, id: number, newDecisionId: string) =>
    request<import("@/types").Decision>(
      `/go/projects/${projectId}/decisions/${id}/fork`,
      {
        method: "POST",
        body: JSON.stringify({ new_decision_id: newDecisionId }),
      },
    ),

  uploadIcon: async (
    projectId: number,
    decisionId: number,
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
      `${API_BASE}/go/projects/${projectId}/decisions/${decisionId}/upload-icon`,
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

  iconUrl: (projectId: number, decisionId: number) =>
    `${API_BASE}/go/projects/${projectId}/decisions/${decisionId}/icon`,
};
