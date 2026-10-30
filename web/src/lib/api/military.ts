import { request, API_BASE } from "./_client";

export interface MIOTraitEntry {
  kind?: "initial" | "add" | "override";
  token?: string;
  name?: string;
  icon?: string;
  position?: { x: number; y: number };
  relative_position_id?: string;
  all_parents?: string[];
  any_parent?: string[];
  limit_to_equipment_type?: string[];
  equipment_bonus?: Record<string, number>;
  production_bonus?: Record<string, number>;
  organization_modifier?: Record<string, number>;
  available?: string;
  visible?: string;
  on_complete?: string;
  ai_will_do?: string;
  special_trait_background?: boolean;
  raw_extras?: Record<string, unknown>;
  [key: string]: unknown;
}

export interface MIORow {
  id: number;
  project_id: number;
  mio_id: string;
  include: string;
  icon: string;

  equipment_type: string[];
  research_categories: string[];
  allowed: string;
  available: string;
  visible: string;
  traits: MIOTraitEntry[];
  raw_extras: Record<string, unknown>;
  country_tag: string;
  scope_kind: string;
  shared_country_tags: string[] | null;
  source: string;
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;

  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  created_at: string;
  updated_at: string;
}

export const miosApi = {
  list: (projectId: number) =>
    request<{ mios: MIORow[]; total: number }>(
      `/go/projects/${projectId}/mios`,
    ),

  create: (
    projectId: number,
    data: Partial<
      Omit<MIORow, "id" | "project_id" | "created_at" | "updated_at">
    > & { mio_id: string },
  ) =>
    request<MIORow>(`/go/projects/${projectId}/mios`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (
    projectId: number,
    mioId: number,
    data: Partial<
      Omit<MIORow, "id" | "project_id" | "created_at" | "updated_at">
    >,
  ) =>
    request<MIORow>(`/go/projects/${projectId}/mios/${mioId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  remove: (projectId: number, mioId: number) =>
    request<void>(`/go/projects/${projectId}/mios/${mioId}`, {
      method: "DELETE",
    }),

  uploadIcon: async (
    projectId: number,
    mioId: number,
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
      `${API_BASE}/go/projects/${projectId}/mios/${mioId}/upload-icon`,
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
      warnings: string[];
    }>;
  },

  iconUrl: (projectId: number, mioId: number) =>
    `${API_BASE}/go/projects/${projectId}/mios/${mioId}/icon`,

  uploadTraitIcon: async (
    projectId: number,
    mioDbId: number,
    token: string,
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
      `${API_BASE}/go/projects/${projectId}/mios/${mioDbId}/traits/${encodeURIComponent(token)}/upload-icon`,
      { method: "POST", body: formData },
    );
    if (!res.ok) {
      const detail = await res.text();
      throw new Error(`Upload failed: ${detail || res.statusText}`);
    }
    return res.json() as Promise<{
      filename: string;
      icon: string;
      preview_url: string;
      warnings: string[];
    }>;
  },

  traitIconUrl: (projectId: number, mioDbId: number, token: string) =>
    `${API_BASE}/go/projects/${projectId}/mios/${mioDbId}/traits/${encodeURIComponent(token)}/icon`,
};

export const DOCTRINE_ENTITY_TYPES = [
  "folder",
  "grand_doctrine",
  "track",
  "subdoctrine",
] as const;
export type DoctrineEntityType = (typeof DOCTRINE_ENTITY_TYPES)[number];

export interface DoctrineRow {
  id: number;
  project_id: number;
  entity_type: DoctrineEntityType;
  entity_key: string;
  folder_key: string;
  track_key: string;
  name: string;
  description: string;
  icon: string;
  xp_cost: number;
  xp_type: string;

  xor?: string | null;

  allow_in_multiple_tracks?: boolean | null;
  raw_extras: Record<string, unknown>;
  source_file: string;
  source: string;
  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  created_at: string;
  updated_at: string;
}

export interface IntelAgencyUpgrade {
  token: string;
  picture?: string;
  frame?: string;
  sound?: string;
  ai_will_do?: string;
  modifiers_during_progress?: string;
  available?: string;
  visible?: string;
  levels?: string[];
  raw_extras?: Record<string, unknown>;
  [key: string]: unknown;
}

export interface IntelAgencyBranch {
  branch: string;
  upgrades: IntelAgencyUpgrade[];
}

export interface IntelligenceAgencyRow {
  id: number;
  project_id: number;
  agency_id: string;
  picture: string;
  names: string[];
  default_trigger: string;
  available: string;
  upgrade_branches: IntelAgencyBranch[];
  raw_extras: Record<string, unknown>;
  country_tag: string;
  scope_kind: string;
  shared_country_tags: string[] | null;
  source: string;
  name_en: string;
  name_zh: string;

  display_name_locs: Record<string, string>;
  created_at: string;
  updated_at: string;
}

type DoctrineWritable = Partial<
  Omit<
    DoctrineRow,
    "id" | "project_id" | "source" | "created_at" | "updated_at"
  >
>;

export const doctrinesApi = {
  list: (projectId: number, entityType?: DoctrineEntityType) =>
    request<{ doctrines: DoctrineRow[]; total: number }>(
      `/go/projects/${projectId}/doctrines${entityType ? `?entity_type=${entityType}` : ""}`,
    ),

  create: (
    projectId: number,
    data: DoctrineWritable & {
      entity_key: string;
      entity_type: DoctrineEntityType;
    },
  ) =>
    request<DoctrineRow>(`/go/projects/${projectId}/doctrines`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (projectId: number, doctrineId: number, data: DoctrineWritable) =>
    request<DoctrineRow>(`/go/projects/${projectId}/doctrines/${doctrineId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  remove: (projectId: number, doctrineId: number) =>
    request<void>(`/go/projects/${projectId}/doctrines/${doctrineId}`, {
      method: "DELETE",
    }),
};

export const intelligenceAgenciesApi = {
  list: (projectId: number) =>
    request<{ agencies: IntelligenceAgencyRow[]; total: number }>(
      `/go/projects/${projectId}/intelligence-agencies`,
    ),

  create: (
    projectId: number,
    data: Partial<
      Omit<
        IntelligenceAgencyRow,
        "id" | "project_id" | "created_at" | "updated_at"
      >
    > & { agency_id: string },
  ) =>
    request<IntelligenceAgencyRow>(
      `/go/projects/${projectId}/intelligence-agencies`,
      { method: "POST", body: JSON.stringify(data) },
    ),

  update: (
    projectId: number,
    agencyId: number,
    data: Partial<
      Omit<
        IntelligenceAgencyRow,
        "id" | "project_id" | "created_at" | "updated_at"
      >
    >,
  ) =>
    request<IntelligenceAgencyRow>(
      `/go/projects/${projectId}/intelligence-agencies/${agencyId}`,
      { method: "PUT", body: JSON.stringify(data) },
    ),

  remove: (projectId: number, agencyId: number) =>
    request<void>(
      `/go/projects/${projectId}/intelligence-agencies/${agencyId}`,
      { method: "DELETE" },
    ),

  uploadLogo: async (
    projectId: number,
    agencyId: number,
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
      `${API_BASE}/go/projects/${projectId}/intelligence-agencies/${agencyId}/upload-logo`,
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
      warnings: string[];
    }>;
  },

  logoUrl: (projectId: number, agencyId: number) =>
    `${API_BASE}/go/projects/${projectId}/intelligence-agencies/${agencyId}/logo`,
};

export const equipmentApi = {
  list: (projectId: number) =>
    request<import("@/types").Equipment[]>(
      `/go/projects/${projectId}/equipment`,
    ),
  get: (projectId: number, id: number) =>
    request<import("@/types").Equipment>(
      `/go/projects/${projectId}/equipment/${id}`,
    ),
  create: (projectId: number, data: Partial<import("@/types").Equipment>) =>
    request<import("@/types").Equipment>(
      `/go/projects/${projectId}/equipment`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    ),
  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").Equipment>,
  ) =>
    request<import("@/types").Equipment>(
      `/go/projects/${projectId}/equipment/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),
  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/equipment/${id}`, {
      method: "DELETE",
    }),
  preview: (projectId: number, id: number) =>
    request<{ script: string }>(
      `/go/projects/${projectId}/equipment/preview/${id}`,
    ),

  enabledBy: (projectId: number, equipmentIdStr: string) =>
    request<{
      equipment_id: string;
      techs: { tech_id: string; name: string; tech_line: string }[];
    }>(
      `/go/projects/${projectId}/equipment/enabled-by/${encodeURIComponent(equipmentIdStr)}`,
    ),
  importText: (
    projectId: number,
    text: string,
    sourceFile = "",
    policy = "overwrite",
  ) =>
    request<{
      created: number;
      updated: number;
      skipped: number;
      warnings: string[];
      equipment_ids: string[];
    }>(`/go/projects/${projectId}/equipment/import`, {
      method: "POST",
      body: JSON.stringify({ text, source_file: sourceFile, policy }),
    }),
};

export const subUnitsApi = {
  list: (
    projectId: number,
    filters?: { countryTag?: string; sourceFile?: string },
  ) => {
    const params = new URLSearchParams();
    if (filters?.countryTag !== undefined)
      params.set("country_tag", filters.countryTag);
    if (filters?.sourceFile !== undefined)
      params.set("source_file", filters.sourceFile);
    const query = params.toString();
    return request<import("@/types").SubUnit[]>(
      `/go/projects/${projectId}/sub-units${query ? `?${query}` : ""}`,
    );
  },
  get: (projectId: number, id: number) =>
    request<import("@/types").SubUnit>(
      `/go/projects/${projectId}/sub-units/${id}`,
    ),
  create: (projectId: number, data: Partial<import("@/types").SubUnit>) =>
    request<import("@/types").SubUnit>(`/go/projects/${projectId}/sub-units`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").SubUnit>,
  ) =>
    request<import("@/types").SubUnit>(
      `/go/projects/${projectId}/sub-units/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),
  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/sub-units/${id}`, {
      method: "DELETE",
    }),
  preview: (projectId: number, id: number) =>
    request<{ script: string }>(
      `/go/projects/${projectId}/sub-units/preview/${id}`,
    ),
  importText: (
    projectId: number,
    text: string,
    sourceFile = "",
    policy: "overwrite" | "skip" = "overwrite",
  ) =>
    request<{
      created: number;
      updated: number;
      skipped: number;
      warnings: string[];
      sub_unit_ids: string[];
    }>(`/go/projects/${projectId}/sub-units/import`, {
      method: "POST",
      body: JSON.stringify({ text, source_file: sourceFile, policy }),
    }),
};

export const divisionTemplatesApi = {
  list: (projectId: number) =>
    request<import("@/types").DivisionTemplate[]>(
      `/go/projects/${projectId}/division-templates`,
    ),
  get: (projectId: number, id: number) =>
    request<import("@/types").DivisionTemplate>(
      `/go/projects/${projectId}/division-templates/${id}`,
    ),
  create: (
    projectId: number,
    data: Partial<import("@/types").DivisionTemplate>,
  ) =>
    request<import("@/types").DivisionTemplate>(
      `/go/projects/${projectId}/division-templates`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    ),
  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").DivisionTemplate>,
  ) =>
    request<import("@/types").DivisionTemplate>(
      `/go/projects/${projectId}/division-templates/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),
  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/division-templates/${id}`, {
      method: "DELETE",
    }),
  preview: (projectId: number, id: number) =>
    request<{ script: string }>(
      `/go/projects/${projectId}/division-templates/preview/${id}`,
    ),
  importText: (
    projectId: number,
    oobFile: string,
    text: string,
    policy = "overwrite",
  ) =>
    request<{
      created: number;
      updated: number;
      skipped: number;
      warnings: string[];
      template_names: string[];
    }>(`/go/projects/${projectId}/division-templates/import`, {
      method: "POST",
      body: JSON.stringify({ oob_file: oobFile, text, policy }),
    }),
};

export const divisionDeploymentsApi = {
  list: (projectId: number, oobFile?: string) =>
    request<import("@/types").DivisionDeployment[]>(
      `/go/projects/${projectId}/division-deployments` +
        (oobFile ? `?oob_file=${encodeURIComponent(oobFile)}` : ""),
    ),
  get: (projectId: number, id: number) =>
    request<import("@/types").DivisionDeployment>(
      `/go/projects/${projectId}/division-deployments/${id}`,
    ),
  create: (
    projectId: number,
    data: Partial<import("@/types").DivisionDeployment>,
  ) =>
    request<import("@/types").DivisionDeployment>(
      `/go/projects/${projectId}/division-deployments`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    ),
  update: (
    projectId: number,
    id: number,
    data: Partial<import("@/types").DivisionDeployment>,
  ) =>
    request<import("@/types").DivisionDeployment>(
      `/go/projects/${projectId}/division-deployments/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),
  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/division-deployments/${id}`, {
      method: "DELETE",
    }),
};

export const specialProjectsApi = {
  listSpecializations: (projectId: number, countryTag?: string) =>
    request<import("@/types").Specialization[]>(
      `/go/projects/${projectId}/special-projects/specializations` +
        (countryTag ? `?country_tag=${encodeURIComponent(countryTag)}` : ""),
    ),
  createSpecialization: (
    projectId: number,
    data: Partial<import("@/types").Specialization> & {
      specialization_id: string;
    },
  ) =>
    request<import("@/types").Specialization>(
      `/go/projects/${projectId}/special-projects/specializations`,
      { method: "POST", body: JSON.stringify(data) },
    ),
  updateSpecialization: (
    projectId: number,
    specPk: number,
    data: Partial<import("@/types").Specialization>,
  ) =>
    request<import("@/types").Specialization>(
      `/go/projects/${projectId}/special-projects/specializations/${specPk}`,
      { method: "PUT", body: JSON.stringify(data) },
    ),
  deleteSpecialization: (projectId: number, specPk: number) =>
    request<void>(
      `/go/projects/${projectId}/special-projects/specializations/${specPk}`,
      { method: "DELETE" },
    ),

  listProjects: (
    projectId: number,
    specialization?: string,
    countryTag?: string,
  ) => {
    const params = new URLSearchParams();
    if (specialization) params.set("specialization", specialization);
    if (countryTag) params.set("country_tag", countryTag);
    const qs = params.toString();
    return request<import("@/types").SpecialProject[]>(
      `/go/projects/${projectId}/special-projects/projects` +
        (qs ? `?${qs}` : ""),
    );
  },
  createProject: (
    projectId: number,
    data: Partial<import("@/types").SpecialProject> & { project_token: string },
  ) =>
    request<import("@/types").SpecialProject>(
      `/go/projects/${projectId}/special-projects/projects`,
      { method: "POST", body: JSON.stringify(data) },
    ),
  updateProject: (
    projectId: number,
    nodePk: number,
    data: Partial<import("@/types").SpecialProject>,
  ) =>
    request<import("@/types").SpecialProject>(
      `/go/projects/${projectId}/special-projects/projects/${nodePk}`,
      { method: "PUT", body: JSON.stringify(data) },
    ),
  deleteProject: (projectId: number, nodePk: number) =>
    request<void>(
      `/go/projects/${projectId}/special-projects/projects/${nodePk}`,
      { method: "DELETE" },
    ),

  validate: (projectId: number, countryTag?: string) =>
    request<import("@/types").SpecialProjectValidation>(
      `/go/projects/${projectId}/special-projects/validate` +
        (countryTag ? `?country_tag=${encodeURIComponent(countryTag)}` : ""),
    ),
};
