import { request, API_BASE } from "./_client";
import {
  revisionRequest,
  revisionFetch,
} from "@/components/workbench/save-diagnostics/revisionRequest";
import {
  editRevisionSnapshot,
  withoutEditRevisions,
  type EditRevisions,
} from "@/components/workbench/save-diagnostics/editRevisions";

export interface CountryRow {
  id: number;
  project_id: number;
  tag: string;
  name: string;
  color: number[];

  color_ui?: number[];
  ideology: string;

  graphical_culture?: string;
  graphical_culture_2d?: string;
  name_en: string;
  name_zh: string;
  display_name_locs: Record<string, string>;
  adjective_en: string;
  adjective_zh: string;
  adjective_locs: Record<string, string>;
  ideology_name_locs: Record<string, Record<string, string>>;
  flag_image_path: string;

  flag_paths_by_ideology?: Record<string, string>;

  name_democratic_en?: string;
  name_democratic_zh?: string;
  name_fascism_en?: string;
  name_fascism_zh?: string;
  name_communism_en?: string;
  name_communism_zh?: string;
  name_neutrality_en?: string;
  name_neutrality_zh?: string;
}

export interface CountryDeletePreview {
  country_id: number;
  tag: string;
  name: string;
  will_be_last_country: boolean;
  cascade_counts: Record<string, number>;
  total: number;

  owned_states: number;

  vanilla_tag: boolean;
}

export interface CountryClassificationItem {
  id: number;
  tag: string;
  name: string;
  name_en: string;
  name_zh: string;

  color: number[];
  auto_added: boolean;
  is_active: boolean;
  is_authored: boolean;

  counts: Record<string, number>;
  total_content: number;
}

export interface CountryClassificationResponse {
  project_id: number;
  active_tag: string;
  authored: CountryClassificationItem[];
  reference: CountryClassificationItem[];
  authored_total: number;
  reference_total: number;
}

export const countriesApi = {
  list: (projectId: number) =>
    request<{ countries: CountryRow[]; total: number }>(
      `/go/projects/${projectId}/countries`,
    ),

  grouped: (projectId: number) =>
    request<CountryClassificationResponse>(
      `/go/projects/${projectId}/countries/grouped`,
    ),

  takeIn: (projectId: number, tag: string) =>
    request<{ country: CountryRow; created: boolean }>(
      `/go/projects/${projectId}/countries/take-in`,
      { method: "POST", body: JSON.stringify({ tag }) },
    ),

  update: (
    projectId: number,
    countryId: number,
    data: Partial<Omit<CountryRow, "id" | "project_id">>,
    revisions: EditRevisions = {},
  ) =>
    revisionRequest<CountryRow>(
      projectId,
      `/go/projects/${projectId}/countries/${countryId}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
      revisions,
    ),

  deletePreview: (projectId: number, countryId: number) =>
    request<CountryDeletePreview>(
      `/go/projects/${projectId}/countries/${countryId}/delete-preview`,
    ),

  delete: async (
    projectId: number,
    countryId: number,
    revisions: EditRevisions = {},
  ): Promise<void> => {
    const res = await revisionFetch(
      projectId,
      `${API_BASE}/go/projects/${projectId}/countries/${countryId}`,
      { method: "DELETE" },
      revisions,
    );
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.detail || `HTTP ${res.status}`);
    }
  },

  uploadFlag: async (
    projectId: number,
    countryId: number,
    file: File,

    ideology?: string,
  ): Promise<CountryRow> => {
    const formData = new FormData();
    formData.append("file", file);
    const qs = ideology ? `?ideology=${encodeURIComponent(ideology)}` : "";
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/countries/${countryId}/upload-flag${qs}`,
      { method: "POST", body: formData },
    );
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.detail || `HTTP ${res.status}`);
    }
    return res.json();
  },

  flagPreviewUrl: (projectId: number, countryId: number, ideology?: string) =>
    `${API_BASE}/go/projects/${projectId}/countries/${countryId}/flag` +
    (ideology ? `?ideology=${encodeURIComponent(ideology)}` : ""),
};

export interface SuperEventOption {
  name_en: string;
  name_zh: string;
  display_name_locs?: Record<string, string>;
  effect: string;
  ai_chance?: number;
}

export interface SuperEventRow {
  id: number;
  project_id: number;
  number: number;
  country_tag: string;
  title_en: string;
  title_zh: string;
  desc_en: string;
  desc_zh: string;
  quote_en: string;
  quote_zh: string;
  btn_en: string;
  btn_zh: string;

  display_name_locs: Record<string, string>;

  description_locs: Record<string, string>;
  quote_locs: Record<string, string>;
  button_locs: Record<string, string>;
  picture_path: string;
  song_token: string;
  auto_fire_on_startup: boolean;
  auto_fire_after_days: number;

  options: SuperEventOption[];
  source: string;
  created_at: string;
  updated_at: string;
}

export const superEventsApi = {
  list: (projectId: number) =>
    request<{ super_events: SuperEventRow[]; total: number }>(
      `/go/projects/${projectId}/super-events`,
    ),

  getSuperEvent: (projectId: number, seId: number) =>
    request<SuperEventRow>(`/go/projects/${projectId}/super-events/${seId}`),

  create: (
    projectId: number,
    data: Partial<
      Omit<SuperEventRow, "id" | "project_id" | "created_at" | "updated_at">
    > & { number: number },
  ) =>
    request<SuperEventRow>(`/go/projects/${projectId}/super-events`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    projectId: number,
    seId: number,
    data: Partial<
      Omit<SuperEventRow, "id" | "project_id" | "created_at" | "updated_at">
    >,
  ) =>
    request<SuperEventRow>(`/go/projects/${projectId}/super-events/${seId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  remove: (projectId: number, seId: number) =>
    request<void>(`/go/projects/${projectId}/super-events/${seId}`, {
      method: "DELETE",
    }),

  uploadPicture: async (
    projectId: number,
    seId: number,
    file: File,
  ): Promise<SuperEventRow> => {
    const formData = new FormData();
    formData.append("file", file);
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/super-events/${seId}/upload-picture`,
      { method: "POST", body: formData },
    );
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.detail || `HTTP ${res.status}`);
    }
    return res.json();
  },

  picturePreviewUrl: (projectId: number, seId: number) =>
    `${API_BASE}/go/projects/${projectId}/super-events/${seId}/picture`,
};

export type BookmarkFilterChoice119 = "new_content" | "continent" | "label";

export const BOOKMARK_FILTER_CHOICES_1_19: readonly BookmarkFilterChoice119[] =
  ["new_content", "continent", "label"];

export function isBookmarkFilterChoice119(
  token: string,
): token is BookmarkFilterChoice119 {
  return (BOOKMARK_FILTER_CHOICES_1_19 as readonly string[]).includes(token);
}

export function updateBookmarkFilterChoice119(
  filters: readonly string[],
  choice: BookmarkFilterChoice119,
  enabled: boolean,
): string[] {
  const selected = new Set(filters.filter(isBookmarkFilterChoice119));
  if (enabled) selected.add(choice);
  else selected.delete(choice);
  const legacy = filters.filter((token) => !isBookmarkFilterChoice119(token));
  return [
    ...BOOKMARK_FILTER_CHOICES_1_19.filter((token) => selected.has(token)),
    ...legacy,
  ];
}

export interface BookmarkSelectableCountry {
  tag: string;
  history_en: string;
  history_zh: string;
  history_locs?: Record<string, string>;
  minor: boolean;

  ideology?: string;
  ideas?: string[];
  focuses?: string[];

  available?: string;

  label?: string[];

  raw_extras?: Record<string, unknown>;
}

export interface BookmarkRow {
  edit_revision?: string;
  diagnostics?: readonly import("@/lib/saveDiagnostics").SaveDiagnostic[];
  id: number;
  project_id: number;
  bookmark_id: string;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;

  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  date: string;
  picture: string;
  default_country: string;
  default_flag: boolean;
  selectable_countries: BookmarkSelectableCountry[];

  filters: string[];
  source: string;
  created_at: string;
  updated_at: string;
}

export const bookmarksApi = {
  list: (projectId: number) =>
    request<BookmarkRow[]>(`/go/projects/${projectId}/bookmarks`),

  get: (projectId: number, bmId: number) =>
    request<BookmarkRow>(`/go/projects/${projectId}/bookmarks/${bmId}`),

  create: (
    projectId: number,
    data: Partial<
      Omit<BookmarkRow, "id" | "project_id" | "created_at" | "updated_at">
    > & { bookmark_id: string },
  ) =>
    request<BookmarkRow>(`/go/projects/${projectId}/bookmarks`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    projectId: number,
    bmId: number,
    data: Partial<
      Omit<BookmarkRow, "id" | "project_id" | "created_at" | "updated_at">
    >,
    revision = data.edit_revision,
  ) =>
    revisionRequest<BookmarkRow>(
      projectId,
      `/go/projects/${projectId}/bookmarks/${bmId}`,
      { method: "PUT", body: JSON.stringify(withoutEditRevisions(data)) },
      editRevisionSnapshot(`bookmark:${bmId}`, revision),
    ),

  remove: (projectId: number, bmId: number, revision?: string) =>
    revisionRequest<void>(
      projectId,
      `/go/projects/${projectId}/bookmarks/${bmId}`,
      { method: "DELETE" },
      editRevisionSnapshot(`bookmark:${bmId}`, revision),
    ),

  uploadPicture: async (
    projectId: number,
    bmId: number,
    file: File,
    crop?: { x: number; y: number; width: number; height: number },
    revision?: string,
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
      `${API_BASE}/go/projects/${projectId}/bookmarks/${bmId}/upload-picture`,
      {
        method: "POST",
        body: formData,
      },
      editRevisionSnapshot(`bookmark:${bmId}`, revision),
    );
    if (!res.ok) {
      const detail = await res.text();
      throw new Error(`Upload failed: ${detail || res.statusText}`);
    }
    return res.json() as Promise<{
      filename: string;
      picture: string;
      preview_url: string;
      warnings: string[];
    }>;
  },

  pictureUrl: (projectId: number, bmId: number) =>
    `${API_BASE}/go/projects/${projectId}/bookmarks/${bmId}/picture`,
};

export interface HistoryFileRow {
  id: number;
  project_id: number;
  country_tag: string;

  capital: number;
  set_politics_ruling_party: string;
  set_politics_last_election: string;
  set_politics_election_frequency: number;
  set_politics_elections_allowed: boolean | null;
  set_popularities_democratic: number;
  set_popularities_communism: number;
  set_popularities_fascism: number;
  set_popularities_neutrality: number;

  set_popularities: Record<string, number>;
  set_research_slots: number;

  set_convoys: number;
  set_stability: number;
  set_war_support: number;
  add_ideas_extra: string[];
  remove_ideas: string[];
  complete_national_focus_ids: string[];

  set_technology_ids: string[];

  set_cosmetic_tag: string;

  set_oob: string;
  set_air_oob: string;
  set_naval_oob: string;

  startup_flags: string[];

  give_guarantee_tags: string[];
  add_to_faction_tags: string[];
  add_to_faction: string;
  set_autonomy: { target: string; autonomous_state: string }[];
  diplomatic_relation: { country: string; relation: string; active: boolean }[];
  date_blocks: { date: string; script: string }[];
  raw_script: string;
  source: string;
  created_at: string;
  updated_at: string;
}

export const historyFilesApi = {
  list: (projectId: number) =>
    request<HistoryFileRow[]>(`/go/projects/${projectId}/history-files`),

  get: (projectId: number, hfId: number) =>
    request<HistoryFileRow>(`/go/projects/${projectId}/history-files/${hfId}`),

  create: (
    projectId: number,
    data: Partial<
      Omit<HistoryFileRow, "id" | "project_id" | "created_at" | "updated_at">
    > & { country_tag: string },
  ) =>
    request<HistoryFileRow>(`/go/projects/${projectId}/history-files`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    projectId: number,
    hfId: number,
    data: Partial<
      Omit<HistoryFileRow, "id" | "project_id" | "created_at" | "updated_at">
    >,
  ) =>
    request<HistoryFileRow>(`/go/projects/${projectId}/history-files/${hfId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  remove: (projectId: number, hfId: number) =>
    request<void>(`/go/projects/${projectId}/history-files/${hfId}`, {
      method: "DELETE",
    }),
};

export type RawFileKind = "verbatim_override" | "empty_shadow";

export interface RawFileRow {
  id: number;
  project_id: number;
  dest_path: string;
  content: string;

  is_binary: boolean;

  generate_replace_path: boolean;

  raw_kind: RawFileKind;
  target_layer: string | null;
  target_sha256: string | null;
  note: string;
  created_at: string;
  updated_at: string;
}

export interface RawTemplateInfo {
  key: string;
  label_en: string;
  label_zh: string;
  desc_en: string;
  desc_zh: string;
  labels?: Partial<
    Record<"en" | "zh" | "ru" | "de" | "es" | "fr" | "ja" | "ko", string>
  >;
  descriptions?: Partial<
    Record<"en" | "zh" | "ru" | "de" | "es" | "fr" | "ja" | "ko", string>
  >;
  main_param: string;
  route_path: string;
}

const RAW_TEMPLATE_ROUTE_FALLBACK: Record<string, string> = {
  scripted_gui: "",
  on_actions: "on-actions",
  game_rules: "game-rules",
  ai_strategy: "ai-strategy",
  gfx_sprite: "gfx",
  frontendgamesetup_slotsize: "frontendgamesetup",
  defines: "defines",
  bop: "bop",
  music: "music",
  peace_conference: "peace-conference",
};

export const rawFilesApi = {
  list: (projectId: number) =>
    request<RawFileRow[]>(`/go/projects/${projectId}/raw-files`),

  create: (
    projectId: number,
    data: {
      dest_path: string;
      content?: string;
      generate_replace_path?: boolean;
      raw_kind?: "verbatim_override";
      note?: string;
    },
  ) =>
    request<RawFileRow>(`/go/projects/${projectId}/raw-files`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  createEmptyShadow: (
    projectId: number,
    data: {
      dest_path: string;
      target_layer: string;
      target_sha256: string;
      note?: string;
    },
  ) =>
    request<RawFileRow>(`/go/projects/${projectId}/raw-files`, {
      method: "POST",
      body: JSON.stringify({
        ...data,
        content: "",
        generate_replace_path: false,
        raw_kind: "empty_shadow",
      }),
    }),

  createBinary: async (
    projectId: number,
    data: {
      dest_path: string;
      file: File;
      generate_replace_path?: boolean;
      note?: string;
    },
  ): Promise<RawFileRow> => {
    const formData = new FormData();
    formData.append("dest_path", data.dest_path);
    formData.append("file", data.file);
    formData.append(
      "generate_replace_path",
      String(data.generate_replace_path ?? false),
    );
    formData.append("note", data.note ?? "");
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/raw-files/binary`,
      { method: "POST", body: formData },
    );
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.detail || `HTTP ${res.status}`);
    }
    return res.json();
  },

  fromTemplate: (
    projectId: number,
    data: {
      panel_key: string;
      title?: string;
      variables?: { key: string; label: string }[];
    },
  ) =>
    request<RawFileRow[]>(`/go/projects/${projectId}/raw-files/from-template`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  listTemplates: (projectId: number) =>
    request<{ templates: RawTemplateInfo[] }>(
      `/go/projects/${projectId}/raw-files/templates`,
    ),

  fromTemplateGeneric: (
    projectId: number,
    templateKey: string,
    key: string,
    template?: Partial<Pick<RawTemplateInfo, "route_path" | "main_param">>,
  ) => {
    const routePath =
      template?.route_path ??
      RAW_TEMPLATE_ROUTE_FALLBACK[templateKey] ??
      templateKey.replace(/_/g, "-");

    const mainParam = template?.main_param ?? "key";
    return request<RawFileRow[]>(
      `/go/projects/${projectId}/raw-files/from-template${routePath ? `/${routePath}` : ""}`,
      {
        method: "POST",
        body: JSON.stringify(mainParam ? { [mainParam]: key } : {}),
      },
    );
  },

  update: (
    projectId: number,
    rfId: number,
    data: Partial<
      Pick<
        RawFileRow,
        | "dest_path"
        | "content"
        | "generate_replace_path"
        | "raw_kind"
        | "target_layer"
        | "target_sha256"
        | "note"
      >
    >,
  ) =>
    request<RawFileRow>(`/go/projects/${projectId}/raw-files/${rfId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  remove: (projectId: number, rfId: number) =>
    request<void>(`/go/projects/${projectId}/raw-files/${rfId}`, {
      method: "DELETE",
    }),
};

export const autonomyStatesApi = {
  list: (projectId: number) =>
    request<{
      autonomy_states: import("@/types").AutonomyState[];
      total: number;
    }>(`/go/projects/${projectId}/autonomy-states`),
  get: (projectId: number, id: number) =>
    request<import("@/types").AutonomyState>(
      `/go/projects/${projectId}/autonomy-states/${id}`,
    ),
  create: (projectId: number, data: Partial<import("@/types").AutonomyState>) =>
    request<import("@/types").AutonomyState>(
      `/go/projects/${projectId}/autonomy-states`,
      { method: "POST", body: JSON.stringify(data) },
    ),
  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").AutonomyState>,
  ) =>
    request<import("@/types").AutonomyState>(
      `/go/projects/${projectId}/autonomy-states/${id}`,
      { method: "PUT", body: JSON.stringify(data) },
    ),
  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/autonomy-states/${id}`, {
      method: "DELETE",
    }),
};

export const statesApi = {
  list: (projectId: number) =>
    request<import("@/types").State[]>(`/go/projects/${projectId}/states`),

  byOwner: (projectId: number, ownerTag: string) =>
    request<import("@/types").State[]>(
      `/go/projects/${projectId}/states/by-owner/${encodeURIComponent(ownerTag)}`,
    ),

  get: (projectId: number, id: number) =>
    request<import("@/types").State>(`/go/projects/${projectId}/states/${id}`),

  geometry: (projectId: number) =>
    request<import("@/types").StatesGeometryResponse>(
      `/go/projects/${projectId}/states-geometry`,
    ),

  update: (
    projectId: number,
    id: number,
    data: import("@/types").StateUpdate,
  ) =>
    request<import("@/types").State>(`/go/projects/${projectId}/states/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  remove: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/states/${id}`, {
      method: "DELETE",
    }),

  bulkDelete: (
    projectId: number,
    body: import("@/types").StateBulkDeleteRequest,
  ) =>
    request<import("@/types").StateBulkDeleteResult>(
      `/go/projects/${projectId}/states/bulk-delete`,
      { method: "POST", body: JSON.stringify(body) },
    ),

  seedFromVanilla: (
    projectId: number,
    body: import("@/types").StateSeedRequest,
  ) =>
    request<import("@/types").StateSeedResult>(
      `/go/projects/${projectId}/states/seed-from-vanilla`,
      { method: "POST", body: JSON.stringify(body) },
    ),

  bulkOwnerCore: (
    projectId: number,
    body: import("@/types").StateBulkRequest,
  ) =>
    request<import("@/types").StateBulkResult>(
      `/go/projects/${projectId}/states/bulk-owner-core`,
      { method: "POST", body: JSON.stringify(body) },
    ),

  bulkRestore: (
    projectId: number,
    entries: import("@/types").StateRestoreEntry[],
  ) =>
    request<import("@/types").StateBulkResult>(
      `/go/projects/${projectId}/states/bulk-restore`,
      { method: "POST", body: JSON.stringify({ entries }) },
    ),

  restoreVanillaContent: (
    projectId: number,
    body: import("@/lib/restoreVanillaContent").RestoreVanillaRequest,
  ) =>
    request<import("@/lib/restoreVanillaContent").RestoreVanillaResult>(
      `/go/projects/${projectId}/states/restore-vanilla-content`,
      { method: "POST", body: JSON.stringify(body) },
    ),
};
