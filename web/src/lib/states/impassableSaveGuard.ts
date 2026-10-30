export interface ImpassableOwnershipFields {
  impassable?: boolean | null;

  force_link_ownership_to?: number | string | null;
}

export function normalizeForceLinkOwnership(value: unknown): number | null {
  let n: number;
  if (typeof value === "number") {
    n = value;
  } else if (typeof value === "string") {
    const trimmed = value.trim();
    if (trimmed === "") return null;
    n = Number(trimmed);
  } else {
    return null;
  }
  if (!Number.isInteger(n)) return null;
  if (n < 1) return null;
  return n;
}

export function isImpassableMissingForceLink(
  fields: ImpassableOwnershipFields | null | undefined,
): boolean {
  if (!fields || typeof fields !== "object") return false;
  if (fields.impassable !== true) return false;
  return normalizeForceLinkOwnership(fields.force_link_ownership_to) === null;
}

export type ImpassableSaveBlockReason = "impassable_missing_force_link";

export const IMPASSABLE_MISSING_FORCE_LINK_MESSAGE_KEY =
  "impassableMissingForceLink" as const;

export const IMPASSABLE_FORCE_LINK_FIELD = "force_link_ownership_to" as const;

export interface ImpassableSaveGuardResult {
  ok: boolean;

  reason: ImpassableSaveBlockReason | null;

  messageKey: string | null;

  field: typeof IMPASSABLE_FORCE_LINK_FIELD | null;
}

const OK: ImpassableSaveGuardResult = Object.freeze({
  ok: true,
  reason: null,
  messageKey: null,
  field: null,
});

export function checkImpassableSave(
  fields: ImpassableOwnershipFields | null | undefined,
): ImpassableSaveGuardResult {
  if (!isImpassableMissingForceLink(fields)) return OK;
  return {
    ok: false,
    reason: "impassable_missing_force_link",
    messageKey: IMPASSABLE_MISSING_FORCE_LINK_MESSAGE_KEY,
    field: IMPASSABLE_FORCE_LINK_FIELD,
  };
}
