import type { CtdIssue } from "@/lib/api/export";
import type { VanillaStateIndex } from "@/lib/vanillaStateIndex";
import {
  summarizeVanillaContentLoss,
  type VanillaContentLoss,
} from "@/lib/vanillaContentLoss";

export const VANILLA_DLC_NOTICE_EXPECTED_TIER = "attn" as const;

export const VANILLA_DLC_NOTICE_CODE = "vanilla_seed_dlc_effects_absent";

export interface DlcNoticeState {
  state_id: number;

  raw_extras?: Record<string, unknown> | null;
}

export interface VanillaDlcNoticeOptions {
  loadedExistingMod?: boolean | null;

  vanillaStates?: VanillaStateIndex;
}

function hasPreservedRawScript(state: DlcNoticeState): boolean {
  const extras = state.raw_extras;
  if (!extras || typeof extras !== "object") return false;
  return Object.keys(extras).length > 0;
}

export function selectVanillaDlcAffectedStates(
  states: readonly DlcNoticeState[] | null | undefined,
  index: VanillaStateIndex,
): number[] {
  if (!Array.isArray(states) || states.length === 0) return [];
  const hits = new Set<number>();
  for (const state of states) {
    if (!state || typeof state !== "object") continue;
    const id = state.state_id;
    if (typeof id !== "number" || !Number.isFinite(id)) continue;
    if (!index.content[id]) continue;
    if (hasPreservedRawScript(state)) continue;
    hits.add(id);
  }
  return [...hits].sort((a, b) => a - b);
}

export function vanillaDlcNamesForStates(
  stateIds: readonly number[],
  index: VanillaStateIndex,
): string[] {
  const names = new Set<string>();
  for (const id of stateIds) {
    for (const name of index.dlc[id] ?? []) names.add(name);
  }
  return [...names].sort();
}

export function buildVanillaDlcNotice(
  states: readonly DlcNoticeState[] | null | undefined,
  options: VanillaDlcNoticeOptions = {},
): CtdIssue | null {
  if (options.loadedExistingMod === true || !options.vanillaStates) return null;
  const stateIds = selectVanillaDlcAffectedStates(
    states,
    options.vanillaStates,
  );
  if (stateIds.length === 0) return null;
  const dlc = vanillaDlcNamesForStates(stateIds, options.vanillaStates);

  const loss: VanillaContentLoss = summarizeVanillaContentLoss(
    states ?? [],
    options.vanillaStates,
  );
  return {
    code: VANILLA_DLC_NOTICE_CODE,
    entity_ref: null,
    severity: "info",

    message:
      `${stateIds.length} states came from the vanilla template — the content ` +
      "in vanilla's own script did not come with them",

    fix_hint: "Restore vanilla content from the States page",

    detail: {
      count: stateIds.length,
      navalBases: loss.navalBases,
      bunkers: loss.bunkers,
      dams: loss.dams,
      landmarks: loss.landmarks,
      datedBlocks: loss.datedBlocks,
      hasDlc: loss.hasDlc,

      resistance: loss.resistance,
      otherContent: loss.otherContent,
    },
    evidence: { state_ids: stateIds, dlc, landmark_keys: loss.landmarkKeys },
  };
}
