import { request, API_BASE } from "./_client";
import { requestGameIndexed } from "./gameIndex";

export const SCRIPTED_SNIPPET_KINDS = ["effect", "trigger"] as const;
export type ScriptedSnippetKind = (typeof SCRIPTED_SNIPPET_KINDS)[number];

export interface ScriptedSnippetRow {
  id: number;
  project_id: number;
  kind: ScriptedSnippetKind;
  name: string;
  body: string;
  source_file: string;
  source: string;
  raw_extras: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

type ScriptedSnippetWritable = Partial<
  Omit<
    ScriptedSnippetRow,
    "id" | "project_id" | "source" | "created_at" | "updated_at"
  >
>;

export const scriptedSnippetsApi = {
  list: (projectId: number, kind?: ScriptedSnippetKind) =>
    request<{ snippets: ScriptedSnippetRow[]; total: number }>(
      `/go/projects/${projectId}/scripted-snippets${kind ? `?kind=${kind}` : ""}`,
    ),

  create: (
    projectId: number,
    data: ScriptedSnippetWritable & { name: string; kind: ScriptedSnippetKind },
  ) =>
    request<ScriptedSnippetRow>(`/go/projects/${projectId}/scripted-snippets`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    projectId: number,
    snippetId: number,
    data: ScriptedSnippetWritable,
  ) =>
    request<ScriptedSnippetRow>(
      `/go/projects/${projectId}/scripted-snippets/${snippetId}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),

  remove: (projectId: number, snippetId: number) =>
    request<void>(`/go/projects/${projectId}/scripted-snippets/${snippetId}`, {
      method: "DELETE",
    }),
};

export interface ScriptedLocalisationRow {
  id: number;
  project_id: number;
  name: string;
  body: string;
  source_file: string;
  source: string;
  raw_extras: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

type ScriptedLocalisationWritable = Partial<
  Omit<
    ScriptedLocalisationRow,
    "id" | "project_id" | "source" | "created_at" | "updated_at"
  >
>;

export const scriptedLocalisationsApi = {
  list: (projectId: number) =>
    request<{ localisations: ScriptedLocalisationRow[]; total: number }>(
      `/go/projects/${projectId}/scripted-localisations`,
    ),

  create: (
    projectId: number,
    data: ScriptedLocalisationWritable & { name: string },
  ) =>
    request<ScriptedLocalisationRow>(
      `/go/projects/${projectId}/scripted-localisations`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    ),

  update: (
    projectId: number,
    locId: number,
    data: ScriptedLocalisationWritable,
  ) =>
    request<ScriptedLocalisationRow>(
      `/go/projects/${projectId}/scripted-localisations/${locId}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),

  remove: (projectId: number, locId: number) =>
    request<void>(`/go/projects/${projectId}/scripted-localisations/${locId}`, {
      method: "DELETE",
    }),
};

export const IDEOLOGY_ICON_GROUPS = [
  "democratic",
  "communism",
  "fascism",
  "neutrality",
] as const;
export type IdeologyGroupKey = string;

export const ideologyIconsApi = {
  upload: async (
    projectId: number,
    group: string,
    file: File,
  ): Promise<import("@/types").Project> => {
    const formData = new FormData();
    formData.append("file", file);
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/ideology-icons/${group}/upload`,
      { method: "POST", body: formData },
    );
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.detail || `HTTP ${res.status}`);
    }
    return res.json();
  },

  previewUrl: (projectId: number, group: string) =>
    `${API_BASE}/go/projects/${projectId}/ideology-icons/${group}`,
};

export const dynamicModifiersApi = {
  list: (projectId: number) =>
    request<{
      dynamic_modifiers: import("@/types").DynamicModifier[];
      total: number;
    }>(`/go/projects/${projectId}/dynamic-modifiers`),
  get: (projectId: number, id: number) =>
    request<import("@/types").DynamicModifier>(
      `/go/projects/${projectId}/dynamic-modifiers/${id}`,
    ),
  create: (
    projectId: number,
    data: Partial<import("@/types").DynamicModifier>,
  ) =>
    request<import("@/types").DynamicModifier>(
      `/go/projects/${projectId}/dynamic-modifiers`,
      { method: "POST", body: JSON.stringify(data) },
    ),
  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").DynamicModifier>,
  ) =>
    request<import("@/types").DynamicModifier>(
      `/go/projects/${projectId}/dynamic-modifiers/${id}`,
      { method: "PUT", body: JSON.stringify(data) },
    ),
  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/dynamic-modifiers/${id}`, {
      method: "DELETE",
    }),
};

export interface CwtoolAntiPattern {
  id: string;
  regex: string;
  severity: "error" | "warning" | "info";
  message: string;
}

export interface CwtoolFieldEnum {
  whitelist?: string[];
  blacklist?: string[];
  blacklist_message?: string;

  skip_ancestors?: string[];
}

export interface CwtoolRules {
  version: number;
  anti_patterns: CwtoolAntiPattern[];
  trigger_enums: Record<string, string[]>;
  field_enums: Record<string, CwtoolFieldEnum>;
  scope_keys_must_be_block: string[];
  modifier_framework_keys: string[];

  known_effects: string[];
  known_triggers: string[];
  known_modifiers: string[];

  allowed_effects_in_modifier_block: string[];

  modifier_scalar_parent_effects: string[];

  option_scalar_parent_triggers: string[];
}

export const cwtoolApi = {
  whitelist: () => requestGameIndexed<CwtoolRules>("/go/cwtool/whitelist"),
};

export interface QiuqiModifierEntry {
  name_zh: string;
  desc_zh: string;
}

export interface QiuqiModifierBatch {
  version: number;
  count: number;
  translations: Record<string, QiuqiModifierEntry>;
}

export const qiuqiApi = {
  modifiers: () => request<QiuqiModifierBatch>("/go/qiuqi/modifiers"),
};

export const workbenchIconsApi = {
  decisionIconUrl: (projectId: number, iconName: string) =>
    `${API_BASE}/go/projects/${projectId}/decision-icons/${encodeURIComponent(iconName)}`,

  techIconUrl: (projectId: number, techId: string) =>
    `${API_BASE}/go/projects/${projectId}/tech-icons/${encodeURIComponent(techId)}`,

  techIconSize: (projectId: number, techId: string) =>
    request<{ width: number; height: number; base_game: boolean }>(
      `/go/projects/${projectId}/tech-icons/${encodeURIComponent(techId)}/size`,
    ),
};
