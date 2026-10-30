export type DraftIdentity = string;

export interface DraftSession<T> {
  identity: DraftIdentity | null;
  persistedFingerprint: string | null;
  draft: T | null;
  dirty: boolean;
  draftGeneration: number;
  saveGeneration: number;
  requestGeneration: number;
}

export interface SaveToken {
  identity: DraftIdentity;
  draftGeneration: number;
  saveGeneration: number;
}

export interface RequestToken {
  identity: DraftIdentity;
  requestGeneration: number;
}

export type SelectionResult<T> =
  | { kind: "selected"; session: DraftSession<T> }
  | { kind: "confirm-discard"; session: DraftSession<T> };

export type RefetchResult<T> =
  | { kind: "ignored"; session: DraftSession<T> }
  | { kind: "applied"; session: DraftSession<T> }
  | { kind: "external-change-replaced-draft"; session: DraftSession<T> };

export function emptyDraftSession<T>(): DraftSession<T> {
  return {
    identity: null,
    persistedFingerprint: null,
    draft: null,
    dirty: false,
    draftGeneration: 0,
    saveGeneration: 0,
    requestGeneration: 0,
  };
}

export function openDraftSession<T>(
  session: DraftSession<T>,
  identity: DraftIdentity,
  persistedFingerprint: string,
  draft: T,
): DraftSession<T> {
  return {
    ...session,
    identity,
    persistedFingerprint,
    draft,
    dirty: false,
    draftGeneration: session.draftGeneration + 1,
  };
}

export function editDraftSession<T>(
  session: DraftSession<T>,
  draft: T,
  fingerprint: string,
): DraftSession<T> {
  if (session.identity === null) return session;
  return {
    ...session,
    draft,
    dirty: fingerprint !== session.persistedFingerprint,
    draftGeneration: session.draftGeneration + 1,
  };
}

export function requestDraftSelection<T>(
  session: DraftSession<T>,
  identity: DraftIdentity,
  persistedFingerprint: string,
  draft: T,
): SelectionResult<T> {
  if (
    session.identity !== null &&
    session.identity !== identity &&
    session.dirty
  ) {
    return { kind: "confirm-discard", session };
  }
  return {
    kind: "selected",
    session: openDraftSession(session, identity, persistedFingerprint, draft),
  };
}

export function confirmDraftSelection<T>(
  session: DraftSession<T>,
  identity: DraftIdentity,
  persistedFingerprint: string,
  draft: T,
): DraftSession<T> {
  return openDraftSession(session, identity, persistedFingerprint, draft);
}

export function beginDraftSave<T>(session: DraftSession<T>): SaveToken | null {
  if (session.identity === null || session.draft === null) return null;
  session.saveGeneration += 1;
  return {
    identity: session.identity,
    draftGeneration: session.draftGeneration,
    saveGeneration: session.saveGeneration,
  };
}

export function resolveDraftSave<T>(
  session: DraftSession<T>,
  token: SaveToken,
  persistedFingerprint: string,
  savedDraft: T,
): DraftSession<T> {
  if (
    session.identity !== token.identity ||
    token.saveGeneration !== session.saveGeneration
  ) {
    return session;
  }
  if (session.draftGeneration !== token.draftGeneration) {
    return {
      ...session,
      persistedFingerprint,
      dirty: true,
    };
  }
  return {
    ...session,
    persistedFingerprint,
    draft: savedDraft,
    dirty: false,
  };
}

export function beginDraftRequest<T>(
  session: DraftSession<T>,
): RequestToken | null {
  if (session.identity === null) return null;
  session.requestGeneration += 1;
  return {
    identity: session.identity,
    requestGeneration: session.requestGeneration,
  };
}

export function resolveDraftRefetch<T>(
  session: DraftSession<T>,
  token: RequestToken,
  identity: DraftIdentity,
  persistedFingerprint: string,
  draft: T,
): RefetchResult<T> {
  if (
    session.identity !== identity ||
    token.identity !== identity ||
    token.requestGeneration !== session.requestGeneration
  ) {
    return { kind: "ignored", session };
  }
  const externalChange =
    session.dirty && persistedFingerprint !== session.persistedFingerprint;
  return {
    kind: externalChange ? "external-change-replaced-draft" : "applied",
    session: {
      ...session,
      persistedFingerprint,
      draft,
      dirty: false,
      draftGeneration: session.draftGeneration + 1,
    },
  };
}

export function stableDraftFingerprint(value: unknown): string {
  const walk = (item: unknown): unknown => {
    if (item === null || typeof item !== "object") return item;
    if (Array.isArray(item)) return item.map(walk);
    const result: Record<string, unknown> = {};
    for (const key of Object.keys(item as Record<string, unknown>).sort()) {
      result[key] = walk((item as Record<string, unknown>)[key]);
    }
    return result;
  };
  return JSON.stringify(walk(value));
}
