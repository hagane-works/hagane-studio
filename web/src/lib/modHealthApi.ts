const API_BASE = "/api";

export type ModHealthSeverity = "error" | "warning" | "info";

export interface ModHealthEntityRef {
  kind: string;
  entity_id: string | null;
  name: string | null;
  country_tag: string | null;
}

export interface ModHealthFinding {
  severity: ModHealthSeverity;

  code: string;
  title: string;
  reason: string;
  suggestion: string;
  entity_ref: ModHealthEntityRef | null;

  file: string | null;
  detail: Record<string, unknown>;
}

export interface ModHealthTotals {
  error: number;
  warning: number;
  info: number;
}

interface ModHealthReportBase {
  findings: ModHealthFinding[];
  totals: ModHealthTotals;
  count: number;
}

export interface PerfReport extends ModHealthReportBase {
  scanned: {
    focus_trees: number;
    focus_nodes: number;
    events: number;
    decisions: number;
    ideas: number;
    characters: number;
    history_files: number;
  };

  export_profile: {
    duration_ms: number;
    zip_bytes: number;
    file_count: number | null;
  } | null;
}

export interface ErrorLogReport extends ModHealthReportBase {
  lines_parsed: number;
  unique_signatures: number;
  mapped: number;
  unmapped: number;

  by_bucket: Record<string, number>;
}

export type PlaytestExpectedModStatus =
  | "not_requested"
  | "activated"
  | "not_activated"
  | "identity_collision"
  | "indeterminate";

export interface PlaytestRuntimeDiagnostics {
  system: {
    active_mod_count: number | null;
    active_dlc_count: number | null;
    active_dlc_count_interpretation: "environment_only";
  };
  expected_mod: {
    status: PlaytestExpectedModStatus;
    reason: string;
    collision_proven: boolean;
    expected: {
      name: string;
      path: string;
      build_id: string | null;
    } | null;
    matched_entries: Array<Record<string, unknown>>;
    collision_entries: Array<Record<string, unknown>>;
  };
  crash: {
    stage: string;
    confidence: "high" | "medium" | "low";
    last_sentinel: {
      source: string;
      line_number: number;
      marker: string;
      stage: string;
      confidence: "high" | "medium" | "low";
      excerpt: string;
    } | null;
    last_observed_line: string | null;
    last_observed_line_is_root_cause: false;
    root_cause: null;
    caveat: string;
  };
  descriptor_inventory: Array<Record<string, unknown>>;
}

export interface PlaytestBundleReport extends ModHealthReportBase {
  files_present: {
    game_log: boolean;
    exception_txt: boolean;
    dlc_load_json: boolean;
    system_log: boolean;
    mod_inventory_json: boolean;
  };

  built_date: string | null;

  loaded_date: string | null;
  runtime_diagnostics: PlaytestRuntimeDiagnostics | null;
  runtime_diagnostics_error: string | null;
}

export interface PlaytestBundleFiles {
  gameLog?: File | null;
  exceptionTxt?: File | null;
  dlcLoadJson?: File | null;
  systemLog?: File | null;
  modInventoryJson?: File | null;
}

export interface PlaytestExpectedModIdentity {
  name?: string;
  path?: string;
  buildId?: string;
}

export interface StartDateLintReport extends ModHealthReportBase {
  start_date: string | null;

  custom_start: boolean;
}

async function throwForStatus(res: Response): Promise<never> {
  const err = await res.json().catch(() => ({ detail: res.statusText }));
  const detail =
    typeof err?.detail === "string"
      ? err.detail
      : `Request failed: ${res.status}`;
  throw new Error(detail);
}

export async function getPerfProfile(projectId: number): Promise<PerfReport> {
  const res = await fetch(`${API_BASE}/go/lint/perf/${projectId}`, {
    credentials: "include",
  });
  if (!res.ok) return throwForStatus(res);
  return (await res.json()) as PerfReport;
}

export async function analyzeErrorLog(
  projectId: number,
  file: File,
): Promise<ErrorLogReport> {
  const form = new FormData();
  form.append("file", file, file.name || "error.log");
  const res = await fetch(`${API_BASE}/go/lint/error-log/${projectId}`, {
    method: "POST",
    credentials: "include",

    body: form,
  });
  if (!res.ok) return throwForStatus(res);
  return (await res.json()) as ErrorLogReport;
}

export async function analyzePlaytestBundle(
  projectId: number,
  files: PlaytestBundleFiles,
  expectedMod: PlaytestExpectedModIdentity = {},
): Promise<PlaytestBundleReport> {
  const form = new FormData();
  if (files.gameLog)
    form.append("game_log", files.gameLog, files.gameLog.name || "game.log");
  if (files.exceptionTxt)
    form.append(
      "exception_txt",
      files.exceptionTxt,
      files.exceptionTxt.name || "exception.txt",
    );
  if (files.dlcLoadJson)
    form.append(
      "dlc_load_json",
      files.dlcLoadJson,
      files.dlcLoadJson.name || "dlc_load.json",
    );
  if (files.systemLog)
    form.append(
      "system_log",
      files.systemLog,
      files.systemLog.name || "system.log",
    );
  if (files.modInventoryJson)
    form.append(
      "mod_inventory_json",
      files.modInventoryJson,
      files.modInventoryJson.name || "mod_inventory.json",
    );
  if (expectedMod.name?.trim())
    form.append("expected_mod_name", expectedMod.name.trim());
  if (expectedMod.path?.trim())
    form.append("expected_mod_path", expectedMod.path.trim());
  if (expectedMod.buildId?.trim())
    form.append("expected_build_id", expectedMod.buildId.trim());
  const res = await fetch(`${API_BASE}/go/lint/playtest-bundle/${projectId}`, {
    method: "POST",
    credentials: "include",

    body: form,
  });
  if (!res.ok) return throwForStatus(res);
  return (await res.json()) as PlaytestBundleReport;
}

export async function getStartDateLint(
  projectId: number,
): Promise<StartDateLintReport> {
  const res = await fetch(`${API_BASE}/go/lint/start-dates/${projectId}`, {
    credentials: "include",
  });
  if (!res.ok) return throwForStatus(res);
  return (await res.json()) as StartDateLintReport;
}

export const modHealthApi = {
  getPerfProfile,
  analyzeErrorLog,
  analyzePlaytestBundle,
  getStartDateLint,
};
