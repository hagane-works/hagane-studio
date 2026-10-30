import {
  revisionFetch,
  publishEditRevisionWrites,
} from "@/components/workbench/save-diagnostics/revisionRequest";
import {
  editRevisionQueue,
  editRevisionHeaders,
  validEditRevisions,
  type EditRevisions,
} from "@/components/workbench/save-diagnostics/editRevisions";
import {
  editRevisionErrorCode,
  localizedEditRevisionError,
} from "@/components/workbench/save-diagnostics/editRevisionError";
import {
  request,
  xhrUploadWithProgress,
  fetchWithTimeoutRetry,
  XhrUploadOptions,
} from "./_client";

export const GO_IMPORT_DIRECT = true;
const GO_IMPORT_BASE = "/api/go/import";
const GO_PROJECT_BASE = "/api/go/projects";

export class ImportHttpError extends Error {
  readonly status: number;

  constructor(status: number, detail: string) {
    super(detail || `Import request failed: ${status}`);
    this.name = "ImportHttpError";
    this.status = status;
  }
}

export interface ImportResult {
  total: number;
  created?: number;
  updated?: number;
  skipped?: number;
  created_nodes?: number;
  tree_id?: string;
  preserved_fields: Record<string, number>;
  warnings: string[];
  conflict_policy?: ConflictPolicy;
}

export type ConflictPolicy = "overwrite" | "skip";

export interface CrossRefReport {
  broken_refs: Record<string, number>;

  sample_ids: Record<string, string[]>;
  total_broken_calls: number;

  suggested_countries: Array<{ tag: string; hits: number }>;

  active_batch: string[];

  unmodeled_refs?: Record<string, number>;

  unmodeled_sample_ids?: Record<string, string[]>;
  total_unmodeled_calls?: number;
}

export interface ScanAndParseResult {
  characters: {
    found_files: number;
    created: number;
    updated: number;
    skipped: number;
    warnings: string[];
  };
  focus_trees: {
    found_files: number;
    created_trees: number;
    updated_trees: number;
    skipped_trees: number;
    created_nodes: number;
    warnings: string[];
  };
  technologies: {
    found_files: number;
    created: number;
    updated: number;
    skipped: number;
    warnings: string[];
  };
  ideas: {
    found_files: number;
    created: number;
    updated: number;
    skipped: number;
    skipped_other_country?: number;
    character_slot_count?: number;
    warnings: string[];
  };
  events: {
    found_files: number;
    created_namespaces: number;
    reused_namespaces?: number;
    created_events: number;
    updated_events: number;
    skipped_events: number;
    warnings: string[];
  };
  decisions: {
    found_files: number;
    found_categories: number;
    created_categories: number;
    updated_categories?: number;
    created_decisions: number;
    updated_decisions: number;
    skipped_decisions: number;
    skipped_other_country?: number;
    warnings: string[];
  };
  warnings: string[];
  needed_images: string[];
  gfx_mappings_count: number;
  session_id: string;
  conflict_policy: ConflictPolicy;
  cross_ref_report?: CrossRefReport;
}

export interface DrawioImportResult {
  tree_id: string;
  tree_db_id: number;
  focuses_created: number;
  prereq_edges_created: number;
  warnings: string[];
}

export interface EntityTriCount {
  new: number;
  modified: number;
  unchanged: number;
}

export interface ScanStatsResult {
  entity_stats: {
    characters: EntityTriCount;
    focus_trees: EntityTriCount;
    technologies: EntityTriCount;
    ideas: EntityTriCount;
    events: EntityTriCount;
    decisions: EntityTriCount;
    bookmarks: EntityTriCount;
    history_files: EntityTriCount;
    states?: EntityTriCount;
  };
  found_files: Record<string, number>;
  warnings: string[];
}

export interface UploadImagesResult {
  received: number;
  portraits_matched: number;

  portraits_vanilla_baseline?: number;
  goal_icons_saved: number;

  asset_audit?: {
    idea_pictures_ext_swap: number;
    event_pictures_ext_swap: number;
    decision_icons_ext_swap: number;
    tech_icons_ext_swap: number;
  };
  warnings: string[];
}

interface PlatformJobView {
  id: string;
  status:
    | "queued"
    | "dispatching"
    | "running"
    | "committing"
    | "succeeded"
    | "failed"
    | "cancelled";
  progress_phase?: string | null;
  progress_pct: number;
  error_detail?: string | null;
}

type ImportSelection = {
  characters: boolean;
  focus: boolean;
  tech: boolean;
  ideas: boolean;
  events: boolean;
  decisions: boolean;
  bookmarks?: boolean;
  history_files?: boolean;
  states?: boolean;
};

type PlatformParseOptions = XhrUploadOptions & {
  onJobProgress?: (job: PlatformJobView) => void;
};

export const twoStageImportApi = {
  scanAndParseBestAvailable: async (
    projectId: number,
    files: File[],
    paths: string[],
    sessionId: string,
    options: ImportSelection,
    conflictPolicy: ConflictPolicy = "overwrite",
    countryTag?: string,
    upload?: PlatformParseOptions,
    revisions: EditRevisions = {},
  ): Promise<ScanAndParseResult> => {
    const guardedRevisions = validEditRevisions(revisions);

    const maxBatchBytes = 64 * 1024 * 1024;
    const maxBatchFiles = 1000;
    const batches: Array<{ files: File[]; paths: string[]; bytes: number }> =
      [];
    let batch = { files: [] as File[], paths: [] as string[], bytes: 0 };
    for (let index = 0; index < files.length; index++) {
      const size = files[index].size;
      if (
        batch.files.length > 0 &&
        (batch.bytes + size > maxBatchBytes ||
          batch.files.length >= maxBatchFiles)
      ) {
        batches.push(batch);
        batch = { files: [], paths: [], bytes: 0 };
      }
      batch.files.push(files[index]);
      batch.paths.push(paths[index]);
      batch.bytes += size;
    }
    if (batch.files.length > 0) batches.push(batch);

    const loaded = new Array<number>(batches.length).fill(0);
    const total = Math.max(
      files.reduce((sum, file) => sum + file.size, 0),
      1,
    );
    const emit = () =>
      upload?.onUploadProgress?.({
        loaded: Math.min(
          loaded.reduce((sum, value) => sum + value, 0),
          total,
        ),
        total,
        lengthComputable: true,
      });
    emit();
    for (let index = 0; index < batches.length; index++) {
      for (let attempt = 1; attempt <= 2; attempt++) {
        try {
          await twoStageImportApi.stageFiles(
            projectId,
            batches[index].files,
            batches[index].paths,
            sessionId,
            {
              signal: upload?.signal,
              timeoutMs: upload?.timeoutMs,
              onUploadProgress: (event) => {
                loaded[index] = Math.min(event.loaded, batches[index].bytes);
                emit();
              },
            },
          );
          break;
        } catch (error) {
          const aborted =
            upload?.signal?.aborted ||
            (error instanceof DOMException && error.name === "AbortError");
          const retryableHTTP =
            error instanceof ImportHttpError &&
            (error.status === 408 ||
              error.status === 429 ||
              error.status >= 500);
          if (
            aborted ||
            attempt === 2 ||
            (error instanceof ImportHttpError && !retryableHTTP)
          )
            throw error;
          loaded[index] = 0;
          emit();
          await new Promise((resolve) => setTimeout(resolve, 750));
        }
      }
      loaded[index] = batches[index].bytes;
      emit();
    }
    return twoStageImportApi.scanAndParseStaged(
      projectId,
      sessionId,
      options,
      conflictPolicy,
      countryTag,
      upload,
      guardedRevisions,
    );
  },

  stageFiles: async (
    projectId: number,
    files: File[],
    paths: string[],
    sessionId: string,
    upload?: XhrUploadOptions,
  ): Promise<{ staged: number; skipped: string[]; total_staged: number }> => {
    const formData = new FormData();

    paths.forEach((p) => formData.append("paths", p));
    files.forEach((f) => formData.append("files", f));
    formData.append("session_id", sessionId);
    const res = await xhrUploadWithProgress(
      `${GO_IMPORT_BASE}/staging/${projectId}/${encodeURIComponent(sessionId)}`,
      formData,
      upload,
    );
    if (!res.ok) {
      const err = (await res
        .json()
        .catch(() => ({ detail: `Stage failed: ${res.status}` }))) as {
        detail?: string;
      };
      throw new ImportHttpError(
        res.status,
        err.detail || `Stage failed: ${res.status}`,
      );
    }
    return res.json() as Promise<{
      staged: number;
      skipped: string[];
      total_staged: number;
    }>;
  },

  scanAndParseStaged: async (
    projectId: number,
    sessionId: string,
    options: {
      characters: boolean;
      focus: boolean;
      tech: boolean;
      ideas: boolean;
      events: boolean;
      decisions: boolean;
      bookmarks?: boolean;
      history_files?: boolean;

      states?: boolean;
    },
    conflictPolicy: ConflictPolicy = "overwrite",
    countryTag?: string,
    upload?: XhrUploadOptions,
    revisions: import("@/components/workbench/save-diagnostics/editRevisions").EditRevisions = {},
  ): Promise<ScanAndParseResult> =>
    editRevisionQueue.run(projectId, revisions, async (submitted) => {
      const payload = {
        project_id: projectId,
        session_id: sessionId,
        country_tag: countryTag ?? "",
        conflict_policy: conflictPolicy,
        import_characters: options.characters,
        import_focus: options.focus,
        import_tech: options.tech,
        import_ideas: options.ideas,
        import_events: options.events,
        import_decisions: options.decisions,
        import_bookmarks: options.bookmarks ?? true,
        import_history_files: options.history_files ?? true,
        import_states: options.states ?? true,
      };
      const start = await fetch(`${GO_IMPORT_BASE}/jobs`, {
        method: "POST",
        credentials: "same-origin",
        headers: {
          "Content-Type": "application/json",
          ...editRevisionHeaders(submitted),
        },
        body: JSON.stringify(payload),
        signal: upload?.signal,
      });
      if (!start.ok) {
        const body = (await start.json().catch(() => ({}))) as {
          error?: string;
        };
        if (editRevisionErrorCode(body)) throw localizedEditRevisionError(body);
        throw new ImportHttpError(
          start.status,
          body.error ?? `Go import start failed: ${start.status}`,
        );
      }
      const deadline = Date.now() + 15 * 60_000;
      const statusURL = `${GO_IMPORT_BASE}/jobs/${projectId}/${encodeURIComponent(sessionId)}`;
      while (Date.now() < deadline) {
        if (upload?.signal?.aborted) {
          await fetch(`${statusURL}/cancel`, {
            method: "POST",
            credentials: "same-origin",
          }).catch(() => undefined);
          throw new DOMException("Import cancelled", "AbortError");
        }
        await new Promise((resolve) => setTimeout(resolve, 350));
        const response = await fetch(statusURL, {
          credentials: "same-origin",
          signal: upload?.signal,
        });
        if (!response.ok) {
          const body = (await response.json().catch(() => ({}))) as {
            error?: string;
          };
          throw new ImportHttpError(
            response.status,
            body.error ?? `Go import status failed: ${response.status}`,
          );
        }
        const state = (await response.json()) as {
          status: string;
          error?: string;
          error_code?: string;
          http_status?: number;
          result?: ScanAndParseResult;
        };
        if (state.status === "complete" && state.result) {
          const committed = validEditRevisions(
            (state.result as ScanAndParseResult & { edit_revisions?: unknown })
              .edit_revisions,
          );
          publishEditRevisionWrites(projectId, committed);
          return { value: state.result, revisions: committed };
        }
        if (state.status === "failed" || state.status === "cancelled") {
          if (editRevisionErrorCode(state.error_code))
            throw localizedEditRevisionError({ error: state.error_code });
          throw new ImportHttpError(
            state.http_status ?? 502,
            state.error ?? `Go import ${state.status}`,
          );
        }
      }
      await fetch(`${statusURL}/cancel`, {
        method: "POST",
        credentials: "same-origin",
      }).catch(() => undefined);
      throw new ImportHttpError(504, "Go import timed out after 15 minutes");
    }),

  uploadImages: async (
    projectId: number,
    files: File[],
    paths: string[],
    sessionId: string,
  ): Promise<UploadImagesResult> => {
    const formData = new FormData();
    paths.forEach((p) => formData.append("paths", p));
    files.forEach((f) => formData.append("files", f));
    formData.append("session_id", sessionId);
    const res = await fetchWithTimeoutRetry(
      `${GO_IMPORT_BASE}/images/${projectId}/${encodeURIComponent(sessionId)}`,
      { method: "POST", body: formData },

      { timeoutMs: 10 * 60_000, maxRetries: 3 },
    );
    if (!res.ok) {
      const err = (await res
        .json()
        .catch(() => ({ detail: res.statusText }))) as {
        detail?: string;
        error?: string;
      };
      throw new Error(
        err.detail || err.error || `Upload failed: ${res.status}`,
      );
    }
    return res.json();
  },
};

export const basemodIndexApi = {
  searchRefs: async (
    projectId: number,
    type: string,
    q: string,
    limit?: number,
  ): Promise<import("@/types").BaseModRefSearchResult> => {
    const qs = new URLSearchParams();
    qs.set("type", type);
    if (q) qs.set("q", q);
    if (limit !== undefined) qs.set("limit", String(limit));
    return request<import("@/types").BaseModRefSearchResult>(
      `/go/basemod/refs?project_id=${projectId}&${qs.toString()}`,
    );
  },

  ideologyRegistry: async (
    projectId: number,
  ): Promise<import("@/types").BaseModIdeologyRegistry> => {
    return request<import("@/types").BaseModIdeologyRegistry>(
      `/go/basemod/ideology-registry?project_id=${projectId}`,
    );
  },
};

export const importApi = {
  drawio: async (
    projectId: number,
    file: File,
    countryTag?: string,
    treeName?: string,
  ): Promise<DrawioImportResult> => {
    const formData = new FormData();
    formData.append("file", file);
    if (countryTag) formData.append("country_tag", countryTag);
    if (treeName) formData.append("tree_name", treeName);
    const res = await fetch(`${GO_PROJECT_BASE}/${projectId}/import/drawio`, {
      method: "POST",
      body: formData,
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({ detail: res.statusText }));
      throw new Error(err.detail || `Draw.io import failed: ${res.status}`);
    }
    return res.json();
  },

  characters: async (
    projectId: number,
    file: File,
    countryTag?: string,
    conflictPolicy?: ConflictPolicy,
  ): Promise<ImportResult> => {
    const formData = new FormData();
    formData.append("file", file);
    if (conflictPolicy) formData.append("conflict_policy", conflictPolicy);

    const q = countryTag
      ? `?country_tag=${encodeURIComponent(countryTag)}`
      : "";
    const res = await fetch(
      `${GO_PROJECT_BASE}/${projectId}/import/characters${q}`,
      {
        method: "POST",
        body: formData,
      },
    );
    if (!res.ok) throw new Error("Import failed");
    return res.json();
  },

  focusTree: async (
    projectId: number,
    file: File,
    countryTag?: string,
    conflictPolicy?: ConflictPolicy,
    revisions: import("@/components/workbench/save-diagnostics/editRevisions").EditRevisions = {},
  ): Promise<ImportResult> => {
    const formData = new FormData();
    formData.append("file", file);
    if (conflictPolicy) formData.append("conflict_policy", conflictPolicy);
    const q = countryTag
      ? `?country_tag=${encodeURIComponent(countryTag)}`
      : "";
    const res = await revisionFetch(
      projectId,
      `${GO_PROJECT_BASE}/${projectId}/import/focus-tree${q}`,
      {
        method: "POST",
        body: formData,
      },
      revisions,
    );
    if (!res.ok) throw new Error("Import failed");
    return res.json();
  },

  technologies: async (
    projectId: number,
    file: File,
    techLine: string,
    folderName: string,
    countryTag?: string,
  ): Promise<ImportResult> => {
    const formData = new FormData();
    formData.append("file", file);
    const extra = countryTag
      ? `&country_tag=${encodeURIComponent(countryTag)}`
      : "";
    const res = await fetch(
      `${GO_PROJECT_BASE}/${projectId}/import/technologies?tech_line=${encodeURIComponent(techLine)}&folder_name=${encodeURIComponent(folderName)}${extra}`,
      { method: "POST", body: formData },
    );
    if (!res.ok) throw new Error("Import failed");
    return res.json();
  },

  ideas: async (
    projectId: number,
    file: File,
    countryTag?: string,
  ): Promise<ImportResult> => {
    const formData = new FormData();
    formData.append("file", file);
    const q = countryTag
      ? `?country_tag=${encodeURIComponent(countryTag)}`
      : "";
    const res = await fetch(
      `${GO_PROJECT_BASE}/${projectId}/import/ideas${q}`,
      {
        method: "POST",
        body: formData,
      },
    );
    if (!res.ok) throw new Error("Import failed");
    return res.json();
  },
};
