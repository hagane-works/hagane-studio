export function nextDraftOnEntityRefetch<T extends object, ID>(
  prevSeededId: ID,
  entityId: ID,
  currentDraft: T,
  entity: T,

  syncFieldsOnSameId: ReadonlyArray<keyof T> = [],
): { draft: T; seededId: ID } {
  if (prevSeededId !== entityId) {
    return { draft: entity, seededId: entityId };
  }

  if (syncFieldsOnSameId.length === 0) {
    return { draft: currentDraft, seededId: prevSeededId };
  }
  const merged: T = { ...currentDraft };
  for (const f of syncFieldsOnSameId) {
    merged[f] = entity[f];
  }
  return { draft: merged, seededId: prevSeededId };
}

export interface ReseedDecision<T, ID> {
  draft: T;

  seededId: ID;

  fingerprint: string;

  reseeded: boolean;

  reason: "switch" | "content" | "none";

  clobberedDirtyDraft: boolean;
}

export interface ReseedArgs<T extends object, ID> {
  prevSeededId: ID;

  prevFingerprint: string;
  entityId: ID;
  entity: T;
  currentDraft: T;

  fingerprintOf: (value: T) => string;

  draftFingerprintOf?: (value: T) => string;

  syncFieldsOnSameId?: ReadonlyArray<keyof T>;
}

export function nextDraftOnEntityChange<T extends object, ID>(
  args: ReseedArgs<T, ID>,
): ReseedDecision<T, ID> {
  const {
    prevSeededId,
    prevFingerprint,
    entityId,
    entity,
    currentDraft,
    fingerprintOf,
    draftFingerprintOf,
    syncFieldsOnSameId = [],
  } = args;

  const fingerprint = fingerprintOf(entity);

  if (prevSeededId !== entityId) {
    return {
      draft: entity,
      seededId: entityId,
      fingerprint,
      reseeded: true,
      reason: "switch",
      clobberedDirtyDraft: false,
    };
  }

  if (fingerprint !== prevFingerprint) {
    const draftFp = (draftFingerprintOf ?? fingerprintOf)(currentDraft);
    return {
      draft: entity,
      seededId: prevSeededId,
      fingerprint,
      reseeded: true,
      reason: "content",

      clobberedDirtyDraft: draftFp !== prevFingerprint,
    };
  }

  if (syncFieldsOnSameId.length === 0) {
    return {
      draft: currentDraft,
      seededId: prevSeededId,
      fingerprint,
      reseeded: false,
      reason: "none",
      clobberedDirtyDraft: false,
    };
  }
  const merged: T = { ...currentDraft };
  for (const f of syncFieldsOnSameId) {
    merged[f] = entity[f];
  }
  return {
    draft: merged,
    seededId: prevSeededId,
    fingerprint,
    reseeded: false,
    reason: "none",
    clobberedDirtyDraft: false,
  };
}

export function stableFingerprint(
  value: unknown,
  omitKeys: readonly string[] = [],
): string {
  const omit = new Set(omitKeys);
  const walk = (node: unknown, depth: number): unknown => {
    if (node === null || typeof node !== "object") return node;
    if (Array.isArray(node)) return node.map((n) => walk(n, depth + 1));
    const src = node as Record<string, unknown>;
    const out: Record<string, unknown> = {};
    for (const k of Object.keys(src).sort()) {
      if (depth === 0 && omit.has(k)) continue;
      out[k] = walk(src[k], depth + 1);
    }
    return out;
  };
  return JSON.stringify(walk(value, 0));
}
