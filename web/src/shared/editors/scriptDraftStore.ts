"use client";

import { create } from "zustand";

export type ParseStatus = "idle" | "parsing" | "ok" | "error" | "info";

export interface ScriptParseError {
  line: number;
  msg: string;
  severity: "error" | "warning" | "info";
}

export interface DraftEntry {
  editRevision?: string;

  draft: string;

  baselineScript: string;
  parseStatus: ParseStatus;
  parseErrors: ScriptParseError[];
  parsedEntity: Record<string, unknown> | null;
  parsedExtras: Record<string, unknown> | null;
  parsedWarnings: string[];
  updatedAt: number;
}

export type DraftKey = string;

const LRU_CAP = 20;

function evictLRU(
  drafts: Map<DraftKey, DraftEntry>,
): Map<DraftKey, DraftEntry> {
  if (drafts.size <= LRU_CAP) return drafts;
  const sorted = [...drafts.entries()].sort(
    (a, b) => a[1].updatedAt - b[1].updatedAt,
  );
  const toEvict = sorted.slice(0, drafts.size - LRU_CAP);
  for (const [k] of toEvict) drafts.delete(k);
  return drafts;
}

interface ScriptDraftState {
  drafts: Map<DraftKey, DraftEntry>;
  getDraft: (key: DraftKey) => DraftEntry | null;
  setDraft: (
    key: DraftKey,
    draft: string,
    baseline: string,
    editRevision?: string,
  ) => void;
  setParseResult: (
    key: DraftKey,
    result: {
      status: ParseStatus;
      errors: ScriptParseError[];
      entity: Record<string, unknown> | null;
      extras: Record<string, unknown> | null;
      warnings: string[];
    },
  ) => void;
  markParsing: (key: DraftKey) => void;
  discardDraft: (key: DraftKey) => void;
  clearAll: () => void;
  isStale: (key: DraftKey, currentScript: string) => boolean;
}

export const useScriptDraftStore = create<ScriptDraftState>((set, get) => ({
  drafts: new Map(),

  getDraft: (key) => get().drafts.get(key) ?? null,

  setDraft: (key, draft, baseline, editRevision) =>
    set((state) => {
      const next = new Map(state.drafts);
      const existing = next.get(key);
      next.set(key, {
        editRevision: existing ? existing.editRevision : editRevision,
        draft,

        baselineScript: existing?.baselineScript ?? baseline,
        parseStatus: existing?.parseStatus ?? "idle",
        parseErrors: existing?.parseErrors ?? [],
        parsedEntity: existing?.parsedEntity ?? null,
        parsedExtras: existing?.parsedExtras ?? null,
        parsedWarnings: existing?.parsedWarnings ?? [],
        updatedAt: Date.now(),
      });
      evictLRU(next);
      return { drafts: next };
    }),

  setParseResult: (key, result) =>
    set((state) => {
      const existing = state.drafts.get(key);

      if (!existing) return state;
      const next = new Map(state.drafts);
      next.set(key, {
        ...existing,
        parseStatus: result.status,
        parseErrors: result.errors,
        parsedEntity: result.entity,
        parsedExtras: result.extras,
        parsedWarnings: result.warnings,
        updatedAt: Date.now(),
      });
      return { drafts: next };
    }),

  markParsing: (key) =>
    set((state) => {
      const existing = state.drafts.get(key);
      if (!existing) return state;
      const next = new Map(state.drafts);
      next.set(key, { ...existing, parseStatus: "parsing" });
      return { drafts: next };
    }),

  discardDraft: (key) =>
    set((state) => {
      if (!state.drafts.has(key)) return state;
      const next = new Map(state.drafts);
      next.delete(key);
      return { drafts: next };
    }),

  clearAll: () => set({ drafts: new Map() }),

  isStale: (key, currentScript) => {
    const entry = get().drafts.get(key);
    if (!entry) return false;
    return entry.baselineScript !== currentScript;
  },
}));

export function draftKey(
  workbench: string,
  entityId: number | null | undefined,
): DraftKey | null {
  if (entityId == null) return null;
  return `${workbench}:${entityId}`;
}
