import type { VanillaStateIndex } from "@/lib/vanillaStateIndex";
import {
  summarizeVanillaContentLoss,
  type VanillaContentLoss,
} from "@/lib/vanillaContentLoss";

export interface RestorableState {
  id: number;
  state_id: number;
  raw_extras?: Record<string, unknown> | null;
}

export interface RestoreVanillaRequest {
  state_db_ids?: number[];
  all?: boolean;
  dry_run?: boolean;
}

export interface VanillaContentCounts {
  naval_bases: number;
  bunkers: number;
  coastal_bunkers: number;
  dams: number;
  landmarks?: string[];
  dated_blocks: number;
  has_dlc: number;
  force_link: boolean;

  resistance: number;

  other_content: number;
}

export const RESTORE_REASON = {
  alreadyHasContent: "already_has_content",

  notInVanilla: "not_in_vanilla",

  vanillaHasNoExtras: "vanilla_has_no_extras",
} as const;

export type RestoreReason =
  (typeof RESTORE_REASON)[keyof typeof RESTORE_REASON];

export interface RestoreVanillaDetail {
  id: number;
  state_id: number;
  restored: boolean;
  reason?: RestoreReason | string;
  keys?: string[];
  content: VanillaContentCounts;
}

export interface RestoreVanillaResult {
  dry_run: boolean;
  restored: number;
  skipped: number;
  missing: number;
  restored_state_ids: number[];
  skipped_state_ids: number[];
  missing_state_ids: number[];
  totals: VanillaContentCounts;
  details: RestoreVanillaDetail[];
  vanilla_files: number;
  vanilla_states: number;

  vanilla_version_known?: boolean;

  vanilla_game_version?: string;

  expected_game_version?: string;
}

export const VANILLA_SOURCE_ERROR_CODES = [
  "vanilla_root_unconfigured",
  "vanilla_states_dir_missing",
  "vanilla_states_empty",
] as const;

export type VanillaSourceErrorCode =
  (typeof VANILLA_SOURCE_ERROR_CODES)[number];

function hasPreservedRawScript(state: RestorableState): boolean {
  const extras = state.raw_extras;
  if (!extras || typeof extras !== "object") return false;
  return Object.keys(extras).length > 0;
}

export interface RestoreSelection {
  stateDbIds: number[];

  stateIds: number[];
}

export function selectRestorableStates(
  states: readonly RestorableState[] | null | undefined,
  index: VanillaStateIndex,
): RestoreSelection {
  if (!Array.isArray(states) || states.length === 0) {
    return { stateDbIds: [], stateIds: [] };
  }
  const dbIds: number[] = [];
  const seen = new Set<number>();
  for (const state of states) {
    if (!state || typeof state !== "object") continue;
    const { id, state_id: stateId } = state;
    if (typeof id !== "number" || !Number.isFinite(id)) continue;
    if (typeof stateId !== "number" || !Number.isFinite(stateId)) continue;
    if (!index.content[stateId]) continue;
    if (hasPreservedRawScript(state)) continue;
    dbIds.push(id);
    seen.add(stateId);
  }
  return {
    stateDbIds: dbIds.sort((a, b) => a - b),
    stateIds: [...seen].sort((a, b) => a - b),
  };
}

export function hasRestorableStates(selection: RestoreSelection): boolean {
  return selection.stateDbIds.length > 0;
}

export function buildRestoreRequest(
  selection: RestoreSelection,
  options: { dryRun?: boolean } = {},
): RestoreVanillaRequest {
  return {
    state_db_ids: selection.stateDbIds,
    ...(options.dryRun ? { dry_run: true } : {}),
  };
}

export function previewFromIndex(
  states: readonly RestorableState[] | null | undefined,
  index: VanillaStateIndex,
): VanillaContentLoss {
  return summarizeVanillaContentLoss(states ?? [], index);
}

export interface RestoreSummary {
  dryRun: boolean;

  restored: number;

  skipped: number;

  missing: number;
  navalBases: number;

  bunkers: number;
  dams: number;

  landmarks: number;

  landmarkKeys: string[];
  datedBlocks: number;
  hasDlc: number;
  forceLink: boolean;

  hasContent: boolean;
}

const EMPTY_SUMMARY: RestoreSummary = {
  dryRun: false,
  restored: 0,
  skipped: 0,
  missing: 0,
  navalBases: 0,
  bunkers: 0,
  dams: 0,
  landmarks: 0,
  landmarkKeys: [],
  datedBlocks: 0,
  hasDlc: 0,
  forceLink: false,
  hasContent: false,
};

function numberOr0(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

export function summarizeRestoreResult(
  result: RestoreVanillaResult | null | undefined,
): RestoreSummary {
  if (!result || typeof result !== "object") {
    return { ...EMPTY_SUMMARY, landmarkKeys: [] };
  }
  const totals = (result.totals ?? {}) as Partial<VanillaContentCounts>;
  const landmarks = Array.isArray(totals.landmarks) ? totals.landmarks : [];
  const summary: RestoreSummary = {
    dryRun: result.dry_run === true,
    restored: numberOr0(result.restored),
    skipped: numberOr0(result.skipped),
    missing: numberOr0(result.missing),
    navalBases: numberOr0(totals.naval_bases),
    bunkers: numberOr0(totals.bunkers) + numberOr0(totals.coastal_bunkers),
    dams: numberOr0(totals.dams),
    landmarks: landmarks.length,
    landmarkKeys: [...new Set(landmarks)].sort(),
    datedBlocks: numberOr0(totals.dated_blocks),
    hasDlc: numberOr0(totals.has_dlc),
    forceLink: totals.force_link === true,
    hasContent: false,
  };
  summary.hasContent =
    summary.navalBases > 0 ||
    summary.bunkers > 0 ||
    summary.dams > 0 ||
    summary.landmarks > 0 ||
    summary.datedBlocks > 0 ||
    summary.hasDlc > 0 ||
    summary.forceLink;
  return summary;
}

export function groupResultStateIds(
  result: RestoreVanillaResult | null | undefined,
): { restored: number[]; skipped: number[]; missing: number[] } {
  const pick = (value: unknown): number[] =>
    Array.isArray(value)
      ? [
          ...new Set(value.filter((n): n is number => typeof n === "number")),
        ].sort((a, b) => a - b)
      : [];
  return {
    restored: pick(result?.restored_state_ids),
    skipped: pick(result?.skipped_state_ids),
    missing: pick(result?.missing_state_ids),
  };
}

interface ErrorLike {
  status?: unknown;
  payload?: unknown;
}

export function vanillaSourceErrorCode(
  error: unknown,
): VanillaSourceErrorCode | null {
  const err = error as ErrorLike | null;
  if (!err || typeof err !== "object") return null;
  if (err.status !== 503) return null;
  const payload = err.payload;
  if (!payload || typeof payload !== "object") return null;
  const code = (payload as { code?: unknown }).code;
  return (VANILLA_SOURCE_ERROR_CODES as readonly string[]).includes(
    code as string,
  )
    ? (code as VanillaSourceErrorCode)
    : null;
}

export function isVanillaSourceUnavailable(error: unknown): boolean {
  return vanillaSourceErrorCode(error) !== null;
}

export const RESTORE_VANILLA_NAMESPACE = "restoreVanillaContent";

export const RESTORE_VANILLA_I18N_KEYS = [
  "button",
  "buttonBusy",

  "dialogTitle",
  "dialogIntro",
  "willRestoreHeading",
  "willRestoreLine",
  "willNotTouchHeading",
  "willNotTouchLine",
  "skippedNote",
  "missingNote",
  "confirm",
  "cancel",

  "successTitle",
  "successBody",
  "nothingToRestore",

  "errorVanillaRootUnconfigured",
  "errorVanillaStatesDirMissing",
  "errorVanillaStatesEmpty",
  "errorGeneric",
] as const;

export type RestoreVanillaI18nKey = (typeof RESTORE_VANILLA_I18N_KEYS)[number];

export const VANILLA_SOURCE_ERROR_I18N_KEY: Record<
  VanillaSourceErrorCode,
  RestoreVanillaI18nKey
> = {
  vanilla_root_unconfigured: "errorVanillaRootUnconfigured",
  vanilla_states_dir_missing: "errorVanillaStatesDirMissing",
  vanilla_states_empty: "errorVanillaStatesEmpty",
};

export function restoreErrorMessageKey(error: unknown): RestoreVanillaI18nKey {
  const code = vanillaSourceErrorCode(error);
  return code ? VANILLA_SOURCE_ERROR_I18N_KEY[code] : "errorGeneric";
}
