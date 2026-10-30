import { request, API_BASE, ApiError } from "./_client";
import { useLocaleStore } from "@/lib/store/localeStore";
import type { ModHealthFinding, StartDateLintReport } from "@/lib/modHealthApi";

export interface ExportValidationIssue {
  level: string;
  message: string;
  code?: string;
  severity?: "blocker" | "warning" | "info";
  entity_ref?: CtdEntityRef | null;
  source?: Record<string, unknown> | string | null;
  evidence?: Record<string, unknown> | null;
  repair_route?: string | null;
  fix_hint?: string;
  detail?: Record<string, unknown>;
}

function parseContentDispositionFilename(cd: string, fallback: string): string {
  const star = cd.match(/filename\*=UTF-8''([^;]+)/i);
  if (star?.[1]) {
    try {
      return decodeURIComponent(star[1]);
    } catch {}
  }
  const plain = cd.match(/filename=(?:"([^"]+)"|([^;\s]+))/);
  return plain?.[1] ?? plain?.[2] ?? fallback;
}

const withExportSelection = (
  path: string,
  includeImported: boolean,
  replacePaths: string[] = [],
  acknowledgeBlockers = false,
): string => {
  const params = new URLSearchParams();
  if (includeImported) params.set("include_imported", "true");
  if (replacePaths.length > 0) {
    params.set("replace_paths", replacePaths.join(","));
  }
  if (acknowledgeBlockers) {
    params.set("acknowledge_blockers", "true");
  }
  const query = params.toString();
  return query ? `${path}?${query}` : path;
};

function issueIdentity(issue: CtdIssue): string {
  const detailTarget =
    issue.detail?.reference ??
    issue.detail?.prereq ??
    issue.detail?.mutex ??
    issue.detail?.focus_id ??
    "";
  return [
    issue.code,
    String(issue.entity_ref?.type ?? ""),
    String(issue.entity_ref?.id ?? ""),
    String(detailTarget),
  ].join("\u0000");
}

async function throwStructuredExportError(
  response: Response,
  fallback: string,
): Promise<never> {
  const text = await response.text().catch(() => "");
  let body: unknown = text;
  try {
    body = text ? JSON.parse(text) : undefined;
  } catch {}
  const payload =
    body && typeof body === "object" && "detail" in body
      ? (body as { detail: unknown }).detail
      : body;
  const detail =
    payload &&
    typeof payload === "object" &&
    typeof (payload as { message?: unknown }).message === "string"
      ? (payload as { message: string }).message
      : typeof payload === "string" && payload
        ? payload
        : fallback;
  throw new ApiError(response.status, detail, payload);
}

export function mergeRunCheckResults(
  base: CtdCheckResult,
  supplemental: CtdIssue[],
): CtdCheckResult {
  const baseBlockers = Array.isArray(base.blockers) ? base.blockers : [];
  const baseWarnings = Array.isArray(base.warnings) ? base.warnings : [];
  const baseIssues =
    Array.isArray(base.issues) && base.issues.length > 0
      ? base.issues
      : [...baseBlockers, ...baseWarnings];
  const seen = new Set(baseIssues.map(issueIdentity));
  const issues = [...baseIssues];
  for (const issue of supplemental) {
    const key = issueIdentity(issue);
    if (seen.has(key)) continue;
    seen.add(key);
    issues.push(issue);
  }
  const blockers = issues.filter((issue) => issue.severity === "blocker");
  const warnings = issues.filter((issue) => issue.severity !== "blocker");
  return {
    ok: blockers.length === 0,
    blocker_count: blockers.length,
    warning_count: warnings.length,
    blockers,
    warnings,
    issues,
  };
}

function modHealthFindingToCtdIssue(finding: ModHealthFinding): CtdIssue {
  const entity = finding.entity_ref;
  const entityId = entity?.entity_id;
  return {
    code: String(finding.code ?? ""),
    severity: finding.severity === "error" ? "blocker" : finding.severity,
    message: String(finding.reason || finding.title || ""),
    fix_hint: String(finding.suggestion ?? ""),
    entity_ref:
      entity?.kind && entityId !== null && entityId !== undefined
        ? {
            type: String(entity.kind),
            id: entityId,
            name: entity.name ?? null,
          }
        : null,
    detail: {
      ...(finding.detail ?? {}),
      title: finding.title,
      ...(finding.file ? { file: finding.file } : {}),
      ...(entity?.country_tag ? { country_tag: entity.country_tag } : {}),
    },
  };
}

async function getStartDateLintIssues(projectId: number): Promise<CtdIssue[]> {
  const report = await request<StartDateLintReport>(
    `/go/lint/start-dates/${projectId}`,
  );
  return (report.findings ?? []).map(modHealthFindingToCtdIssue);
}

async function getGoRunCheckIssues(
  projectId: number,
  includeImported: boolean,
): Promise<CtdIssue[]> {
  const response = await fetch(
    `/api/go/run-check/projects/${projectId}${includeImported ? "?include_imported=true" : ""}`,
    { method: "GET", credentials: "include" },
  );
  if (!response.ok) return [];
  const body = (await response.json()) as { issues?: unknown };
  if (!Array.isArray(body.issues)) return [];
  return body.issues.filter((issue): issue is CtdIssue => {
    if (issue === null || typeof issue !== "object") return false;
    const candidate = issue as Partial<CtdIssue>;
    return (
      typeof candidate.code === "string" &&
      ["blocker", "warning", "info"].includes(String(candidate.severity))
    );
  });
}

export interface ExportJobStatus {
  job_id: string;
  status: "queued" | "snapshotting" | "running" | "complete" | "failed";
  files?: string[];
  error?: unknown;
  http_status?: number;
  filename?: string;
  size_bytes?: number;

  native_warnings?: string[];
}

const exportLocale = () => useLocaleStore.getState().locale;

async function runExportJob(
  projectId: number,
  includeImported: boolean,
  replacePaths: string[],
  acknowledgeBlockers: boolean,
  buildIdentity: boolean,
): Promise<ExportJobStatus & { files: string[] }> {
  const started = await request<ExportJobStatus>(`/go/export/jobs`, {
    method: "POST",
    body: JSON.stringify({
      project_id: projectId,
      include_imported: includeImported,
      replace_paths: replacePaths,
      acknowledge_blockers: acknowledgeBlockers,
      build_identity: buildIdentity,
      locale: exportLocale(),
    }),
  });
  if (started.status === "complete")
    return { ...started, files: started.files ?? [] };
  if (started.status === "failed") {
    const message =
      typeof started.error === "string"
        ? started.error
        : JSON.stringify(started.error ?? "Export build failed");
    throw new ApiError(started.http_status ?? 500, message, started.error);
  }
  const deadline = Date.now() + 20 * 60 * 1000;
  let transientFailures = 0;
  while (Date.now() < deadline) {
    let status: ExportJobStatus;
    try {
      status = await request<ExportJobStatus>(
        `/go/export/jobs/${projectId}/${started.job_id}`,
      );
      transientFailures = 0;
    } catch (error) {
      if (
        (error as { status?: unknown } | null)?.status === 503 &&
        transientFailures < 5
      ) {
        await new Promise((resolve) =>
          setTimeout(resolve, Math.min(500 * 2 ** transientFailures, 5_000)),
        );
        transientFailures += 1;
        continue;
      }
      throw error;
    }
    if (status.status === "complete")
      return { ...status, files: status.files ?? [] };
    if (status.status === "failed") {
      const message =
        typeof status.error === "string"
          ? status.error
          : JSON.stringify(status.error ?? "Export build failed");
      throw new ApiError(status.http_status ?? 500, message, status.error);
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error("Export build timed out after 20 minutes");
}

export type ExistingMod = File | { folder: string };

export function existingModName(base: ExistingMod): string {
  return base instanceof File
    ? base.name
    : (base.folder.split(/[\\/]/).filter(Boolean).pop() ?? base.folder);
}

function appendExistingMod(form: FormData, base: ExistingMod): void {
  if (base instanceof File) form.append("base_mod_file", base);
  else form.append("base_mod_folder", base.folder);
}

export function isNotAModFolderError(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    (error.payload as { error?: unknown } | null)?.error === "not_a_mod"
  );
}

export const exportApi = {
  validate: (
    projectId: number,
    includeImported = false,
    replacePaths: string[] = [],
  ) =>
    request<{
      issues: ExportValidationIssue[];

      summary: {
        characters: number;
        focus_trees: number;
        focus_nodes: number;
        technologies: number;
        ideas: number;
        event_namespaces: number;
        events: number;
        decision_categories: number;
        decisions: number;
        custom_traits: number;
        opinion_modifiers: number;
        dynamic_modifiers: number;
        super_events: number;
        bookmarks: number;
        history_files: number;

        raw_files: number;
      };

      asset_audit?: {
        idea_pictures_missing: number;
        event_pictures_missing: number;
        decision_icons_missing: number;
        tech_icons_missing: number;
        focus_icons_missing: number;
        portraits_missing: number;
        total_missing: number;
      };
    }>(
      withExportSelection(
        `/go/projects/${projectId}/export/validate`,
        includeImported,
        replacePaths,
      ),
    ),

  check: async (
    projectId: number,
    includeImported = false,
    replacePaths: string[] = [],
  ) => {
    const [base, supplemental, startDates] = await Promise.all([
      request<CtdCheckResult>(
        withExportSelection(
          `/go/projects/${projectId}/export/check`,
          includeImported,
          replacePaths,
        ),
        { method: "POST" },
      ),
      getGoRunCheckIssues(projectId, includeImported).catch(() => []),
      getStartDateLintIssues(projectId).catch(() => []),
    ]);

    return mergeRunCheckResults(base, [...supplemental, ...startDates]);
  },

  buildArtifact: (
    projectId: number,
    includeImported = false,
    replacePaths: string[] = [],
    acknowledgeBlockers = false,
    buildIdentity = false,
  ): Promise<ExportJobStatus & { files: string[] }> =>
    runExportJob(
      projectId,
      includeImported,
      replacePaths,
      acknowledgeBlockers,
      buildIdentity,
    ),

  previewStructure: (
    projectId: number,
    includeImported = false,
    replacePaths: string[] = [],
    acknowledgeBlockers = false,
    buildIdentity = false,
  ) =>
    exportApi.buildArtifact(
      projectId,
      includeImported,
      replacePaths,
      acknowledgeBlockers,
      buildIdentity,
    ),

  buildJobDownloadUrl: (projectId: number, jobId: string) =>
    `${API_BASE}/go/export/jobs/${projectId}/${encodeURIComponent(jobId)}/download`,

  install: (
    projectId: number,
    includeImported = false,
    replacePaths: string[] = [],
  ) =>
    request<{ mod_folder: string; descriptor: string; files: number }>(
      `/app/projects/${projectId}/install`,
      {
        method: "POST",
        body: JSON.stringify({
          include_imported: includeImported,
          replace_paths: replacePaths,
          locale: exportLocale(),
        }),
      },
    ),

  baseInstalls: (projectId: number) =>
    request<{ base_installs: BaseInstall[] }>(
      `/go/projects/${projectId}/export/base-installs`,
    ),

  diffPreview: async (
    projectId: number,
    base: ExistingMod,
  ): Promise<DiffExportPreview> => {
    const form = new FormData();
    appendExistingMod(form, base);
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/export/diff-preview`,
      { method: "POST", body: form },
    );
    if (!res.ok) {
      await throwStructuredExportError(
        res,
        `Diff preview failed (${res.status})`,
      );
    }
    return (await res.json()) as DiffExportPreview;
  },

  diffResolve: async (
    projectId: number,
    base: ExistingMod,
    resolutions: Record<string, "keep_user" | "keep_target" | "manual">,
    manualValues: Record<string, unknown>,
  ): Promise<DiffExportResolveResult> => {
    const form = new FormData();
    appendExistingMod(form, base);
    form.append(
      "resolutions_json",
      JSON.stringify({ resolutions, manual_values: manualValues }),
    );
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/export/diff-resolve`,
      { method: "POST", body: form },
    );
    if (!res.ok) {
      await throwStructuredExportError(
        res,
        `Diff resolve failed (${res.status})`,
      );
    }
    return (await res.json()) as DiffExportResolveResult;
  },

  diffExportZip: async (
    projectId: number,
    base: ExistingMod,
    resolutions: Record<string, "keep_user" | "keep_target" | "manual">,
    manualValues: Record<string, unknown>,
    acknowledgeBlockers = false,
  ): Promise<{ blob: Blob; entityCount: number; filename: string }> => {
    const form = new FormData();
    appendExistingMod(form, base);
    form.append(
      "resolutions_json",
      JSON.stringify({ resolutions, manual_values: manualValues }),
    );
    form.append("acknowledge_blockers", String(acknowledgeBlockers));
    form.append("locale", exportLocale());
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/export/diff-export-zip`,
      { method: "POST", body: form },
    );
    if (!res.ok) {
      await throwStructuredExportError(
        res,
        "Diff export ZIP could not be created",
      );
    }
    const blob = await res.blob();
    const entityCount =
      parseInt(res.headers.get("x-merge-entity-count") ?? "0", 10) || 0;

    const cd = res.headers.get("content-disposition") ?? "";
    const filename = parseContentDispositionFilename(
      cd,
      `merged_${existingModName(base)}`,
    );
    return { blob, entityCount, filename };
  },

  appendToExisting: async (
    projectId: number,
    base: ExistingMod,
    options: {
      include_imported?: boolean;
      build_identity?: boolean;
      replace_paths?: string[];
      base_mod_filename?: string;
      acknowledge_blockers?: boolean;
    } = {},
  ): Promise<{ blob: Blob; conflictCount: number; filename: string }> => {
    const form = new FormData();
    appendExistingMod(form, base);
    form.append("include_imported", String(options.include_imported ?? false));
    form.append("build_identity", String(options.build_identity ?? false));
    form.append("replace_paths", (options.replace_paths ?? []).join(","));
    form.append(
      "base_mod_filename",
      options.base_mod_filename ?? existingModName(base),
    );
    form.append(
      "acknowledge_blockers",
      String(options.acknowledge_blockers ?? false),
    );
    form.append("locale", exportLocale());
    const res = await fetch(
      `${API_BASE}/go/projects/${projectId}/export/append-to-existing`,
      { method: "POST", body: form },
    );
    if (!res.ok) {
      await throwStructuredExportError(
        res,
        "Append export ZIP could not be created",
      );
    }
    const conflictCount = Number(
      res.headers.get("X-Merge-Conflict-Count") ?? "0",
    );
    const disposition = res.headers.get("Content-Disposition") ?? "";
    const filename = parseContentDispositionFilename(
      disposition,
      `appended_${existingModName(base)}`,
    );
    const blob = await res.blob();
    return { blob, conflictCount, filename };
  },
};

export interface CtdEntityRef {
  type: string;
  id: string | number;
  [k: string]: unknown;
}

export interface CtdIssue {
  code: string;
  entity_ref: CtdEntityRef | null;
  severity: "blocker" | "warning" | "info";
  message: string;
  fix_hint: string;

  source?: Record<string, unknown> | string | null;
  evidence?: Record<string, unknown> | null;
  repair_route?: string | null;

  detail?: Record<string, unknown>;

  out_of_stage?: boolean;
}

export interface CtdCheckResult {
  ok: boolean;
  blocker_count: number;
  warning_count: number;
  blockers: CtdIssue[];
  warnings: CtdIssue[];
  issues: CtdIssue[];
}

export interface BaseInstall {
  dependency: string;
  matched: boolean;
  display_name: string;
  version: string;
  workshop_url: string;
  mod_key: string;
}

export type DiffOp = {
  op: "add" | "remove" | "replace";
  path: string;
  value?: unknown;
  old?: unknown;
};
export type FieldConflict = {
  path: string;
  user_op: DiffOp;
  target_op: DiffOp;
  kind: "value" | "remove_vs_change";
};
export type ConflictReport = {
  status: "no_change" | "clean_merge" | "auto_mergeable" | "conflict";
  ops: DiffOp[];
  conflicts: FieldConflict[];
  target_value: Record<string, unknown> | null;
  entity_id: string;
  entity_type: string;
};
export type DiffExportPreview = {
  reports: Record<string, ConflictReport[]>;
  totals: {
    no_change: number;
    clean_merge: number;
    auto_mergeable: number;
    conflict: number;
    total_conflict_paths: number;
  };
  warnings: string[];
};
export type DiffExportResolveResult = {
  final_ops_by_entity: Record<string, DiffOp[]>;
  warnings: string[];
};

export interface DeadFlagEntry {
  flag_id: string;
  set_locations: string[];
}

export interface DeadFlagReport {
  set_total: number;
  check_total: number;
  dead_flags: DeadFlagEntry[];
}

export const lintApi = {
  deadFlags: (projectId: number) =>
    request<DeadFlagReport>(`/go/lint/dead-flags/${projectId}`),
  startDates: (projectId: number) =>
    request<StartDateLintReport>(`/go/lint/start-dates/${projectId}`),
};
