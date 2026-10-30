import type { VanillaStateIndex } from "@/lib/vanillaStateIndex";

export interface ContentLossState {
  state_id: number;
  raw_extras?: Record<string, unknown> | null;
}

export interface VanillaContentLoss {
  states: number;
  navalBases: number;
  bunkers: number;
  dams: number;

  landmarks: number;

  landmarkKeys: string[];
  datedBlocks: number;
  hasDlc: number;
  forceLink: number;

  resistance: number;

  otherContent: number;
}

const EMPTY: VanillaContentLoss = {
  states: 0,
  navalBases: 0,
  bunkers: 0,
  dams: 0,
  landmarks: 0,
  landmarkKeys: [],
  datedBlocks: 0,
  hasDlc: 0,
  forceLink: 0,
  resistance: 0,
  otherContent: 0,
};

function hasPreservedRawScript(state: ContentLossState): boolean {
  const extras = state.raw_extras;
  if (!extras || typeof extras !== "object") return false;
  return Object.keys(extras).length > 0;
}

export function summarizeVanillaContentLoss(
  states: readonly ContentLossState[] | null | undefined,
  index: VanillaStateIndex,
): VanillaContentLoss {
  if (!Array.isArray(states) || states.length === 0)
    return { ...EMPTY, landmarkKeys: [] };

  const seen = new Set<number>();
  const landmarks = new Set<string>();
  const out: VanillaContentLoss = { ...EMPTY, landmarkKeys: [] };

  for (const state of states) {
    if (!state || typeof state !== "object") continue;
    const id = state.state_id;
    if (typeof id !== "number" || !Number.isFinite(id)) continue;

    if (seen.has(id)) continue;
    const content = index.content[id];
    if (!content) continue;
    if (hasPreservedRawScript(state)) continue;

    seen.add(id);
    out.navalBases += content.navalBases ?? 0;
    out.bunkers += (content.bunkers ?? 0) + (content.coastalBunkers ?? 0);
    out.dams += content.dams ?? 0;
    out.datedBlocks += content.datedBlocks ?? 0;
    out.hasDlc += content.hasDlc ?? 0;
    out.resistance += content.resistance ?? 0;
    out.otherContent += content.otherContent ?? 0;
    if (content.forceLink) out.forceLink += 1;

    out.landmarks += (content.landmarks ?? []).length;
    for (const key of content.landmarks ?? []) landmarks.add(key);
  }

  out.states = seen.size;
  out.landmarkKeys = [...landmarks].sort();
  return out;
}

export function hasVanillaContentLoss(loss: VanillaContentLoss): boolean {
  return loss.states > 0;
}
