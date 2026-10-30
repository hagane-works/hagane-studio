import { collectTxtFiles, readFileByPath, tryGetDir } from "@/lib/fs-scanner";

export const TECHNOLOGY_SOURCE_GUI_PATH = "interface/countrytechtreeview.gui";
export const TECHNOLOGY_SOURCE_TAGS_PATH =
  "common/technology_tags/00_technology.txt";

export interface LocalTechnologySourceSelection {
  label: string;
  files: File[];
  paths: string[];
}

export class TechnologySourceSelectionError extends Error {
  readonly code: "gui_missing" | "tags_missing";

  constructor(code: "gui_missing" | "tags_missing") {
    super(code);
    this.name = "TechnologySourceSelectionError";
    this.code = code;
  }
}

export async function collectLocalTechnologySource(
  root: FileSystemDirectoryHandle,
): Promise<LocalTechnologySourceSelection> {
  const gui = await readFileByPath(root, TECHNOLOGY_SOURCE_GUI_PATH);
  if (!gui) {
    throw new TechnologySourceSelectionError("gui_missing");
  }
  const tagsDirectory = await tryGetDir(root, "common", "technology_tags");
  if (!tagsDirectory) {
    throw new TechnologySourceSelectionError("tags_missing");
  }
  const tagEntries = await collectTxtFiles(
    tagsDirectory,
    "common/technology_tags",
  );
  if (
    !tagEntries.some(
      (entry) => entry.path.toLowerCase() === TECHNOLOGY_SOURCE_TAGS_PATH,
    )
  ) {
    throw new TechnologySourceSelectionError("tags_missing");
  }
  tagEntries.sort((left, right) => left.path.localeCompare(right.path));
  return {
    label: root.name.trim() || "Local HOI4",
    files: [gui, ...tagEntries.map((entry) => entry.file)],
    paths: [
      TECHNOLOGY_SOURCE_GUI_PATH,
      ...tagEntries.map((entry) => entry.path),
    ],
  };
}

export function technologySourceSessionID(projectId: number): string {
  const suffix =
    globalThis.crypto?.randomUUID?.() ??
    `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  return `tech-source-${projectId}-${suffix}`;
}

export const TECHNOLOGY_SOURCE_IMPORT_OPTIONS = {
  characters: false,
  focus: false,
  tech: false,
  ideas: false,
  events: false,
  decisions: false,
  bookmarks: false,
  history_files: false,
  states: false,
} as const;

interface ErrorLike {
  status?: unknown;
  payload?: unknown;
}

export const TECH_CERTIFIED_SOURCE_ERROR_CODES = [
  "certified_tech_source_unconfigured",
  "certified_tech_source_file_missing",
  "certified_tech_source_version_mismatch",

  "certified_tech_source_content_mismatch",
  "certified_tech_source_unavailable",
] as const;

export type TechCertifiedSourceErrorCode =
  (typeof TECH_CERTIFIED_SOURCE_ERROR_CODES)[number];

export type TechCertifiedSourceI18nKey =
  | "sourceCertifiedUnconfigured"
  | "sourceCertifiedFileMissing"
  | "sourceCertifiedVersionMismatch"
  | "sourceCertifiedContentMismatch"
  | "sourceCertifiedUnavailable";

const TECH_CERTIFIED_SOURCE_I18N_KEY: Record<
  TechCertifiedSourceErrorCode,
  TechCertifiedSourceI18nKey
> = {
  certified_tech_source_unconfigured: "sourceCertifiedUnconfigured",
  certified_tech_source_file_missing: "sourceCertifiedFileMissing",
  certified_tech_source_version_mismatch: "sourceCertifiedVersionMismatch",
  certified_tech_source_content_mismatch: "sourceCertifiedContentMismatch",
  certified_tech_source_unavailable: "sourceCertifiedUnavailable",
};

export function techCertifiedSourceErrorCode(
  error: unknown,
): TechCertifiedSourceErrorCode | null {
  const err = error as ErrorLike | null;
  if (!err || typeof err !== "object") return null;
  if (err.status !== 503) return null;
  const payload = err.payload;
  if (!payload || typeof payload !== "object") return null;
  const code = (payload as { code?: unknown }).code;
  return (TECH_CERTIFIED_SOURCE_ERROR_CODES as readonly string[]).includes(
    code as string,
  )
    ? (code as TechCertifiedSourceErrorCode)
    : null;
}

export function techCertifiedSourceMessageKey(
  error: unknown,
): TechCertifiedSourceI18nKey | null {
  const code = techCertifiedSourceErrorCode(error);
  return code ? TECH_CERTIFIED_SOURCE_I18N_KEY[code] : null;
}

function shortSha(sum: string): string {
  const trimmed = sum.trim().toLowerCase();
  return trimmed.length > 12 ? trimmed.slice(0, 12) : trimmed;
}

export function techCertifiedSourceMessageParams(
  error: unknown,
): Record<string, string> {
  const code = techCertifiedSourceErrorCode(error);
  if (!code) return {};
  const payload = (error as ErrorLike | null)?.payload as
    Record<string, unknown> | undefined;
  const pick = (...names: string[]): string => {
    for (const name of names) {
      const value = payload?.[name];
      if (typeof value === "string" && value.trim() !== "") return value;
    }
    return "unknown";
  };
  if (code === "certified_tech_source_file_missing") {
    return { path: pick("path") };
  }
  if (code === "certified_tech_source_version_mismatch") {
    return {
      path: pick("path"),
      serverVersion: pick("server_version", "server_raw_version"),
      expectedVersion: pick("expected_game_version", "expected_raw_version"),
    };
  }
  if (code === "certified_tech_source_content_mismatch") {
    return {
      path: pick("path"),
      reportedVersion: pick("server_version", "server_raw_version"),
      serverSha: shortSha(pick("server_sha256")),
      expectedSha: shortSha(pick("expected_sha256")),
    };
  }
  return {};
}
