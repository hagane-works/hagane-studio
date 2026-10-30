import type { CtdIssue } from "@/lib/api/export";
import {
  isImpassableMissingForceLink,
  type ImpassableOwnershipFields,
} from "@/lib/states/impassableSaveGuard";

export { isImpassableMissingForceLink };

export const IMPASSABLE_FORCE_LINK_NOTICE_CODE =
  "impassable_without_force_link";

export interface ImpassableForceLinkState extends ImpassableOwnershipFields {
  state_id: number;
}

export function selectImpassableWithoutForceLink(
  states: readonly ImpassableForceLinkState[] | null | undefined,
): number[] {
  if (!Array.isArray(states) || states.length === 0) return [];
  const hits = new Set<number>();
  for (const state of states) {
    if (!state || typeof state !== "object") continue;
    if (!isImpassableMissingForceLink(state)) continue;
    const id = state.state_id;
    if (typeof id !== "number" || !Number.isFinite(id)) continue;
    hits.add(id);
  }

  return [...hits].sort((a, b) => a - b);
}

export function buildImpassableForceLinkNotice(
  states: readonly ImpassableForceLinkState[] | null | undefined,
): CtdIssue | null {
  const stateIds = selectImpassableWithoutForceLink(states);
  if (stateIds.length === 0) return null;
  return {
    code: IMPASSABLE_FORCE_LINK_NOTICE_CODE,
    entity_ref: null,

    severity: "info",

    message:
      `${stateIds.length} impassable states do not say which neighbouring ` +
      "state owns them",
    fix_hint:
      "Open each state and pick the neighbouring state that should own it",
    detail: { count: stateIds.length },
    evidence: { state_ids: stateIds },
  };
}
