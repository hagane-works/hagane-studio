"use client";

import { create } from "zustand";
import { guiRenderApi } from "@/lib/guiRenderApi";
import { withWorkbenchCountry } from "@/lib/gameViewLinks";
import { chosenCountry, noteEdited } from "./countryTakeIn";

export type ChangeState = "saved" | "undoing" | "undone" | "failed";

export interface ChangeEntry {
  id: number;
  projectId: number;

  at: number;

  title: string;

  kind?: string;
  workbenchHref?: string | null;

  undo?: () => Promise<void>;
  state: ChangeState;
  error?: string;

  refused?: { reason: UndoRefused["reason"]; fields: readonly string[] };
}

interface ChangeLog {
  entries: ChangeEntry[];
  record: (
    entry: Omit<ChangeEntry, "id" | "at" | "state" | "error" | "refused">,
  ) => void;

  undo: (id: number) => Promise<boolean>;
}

let nextId = 1;

export const useChangeLog = create<ChangeLog>((set, get) => ({
  entries: [],
  record: (entry) =>
    set((log) => ({
      entries: [
        { ...entry, id: nextId++, at: Date.now(), state: "saved" },
        ...log.entries,
      ],
    })),
  undo: async (id) => {
    const entry = get().entries.find((e) => e.id === id);
    if (!entry?.undo || entry.state === "undoing" || entry.state === "undone")
      return false;
    const mark = (
      state: ChangeState,
      error?: string,
      refused?: ChangeEntry["refused"],
    ) =>
      set((log) => ({
        entries: log.entries.map((e) =>
          e.id === id ? { ...e, state, error, refused } : e,
        ),
      }));
    mark("undoing");
    try {
      await entry.undo();
      mark("undone");
      return true;
    } catch (err) {
      mark(
        "failed",
        err instanceof Error ? err.message : String(err),
        err instanceof UndoRefused
          ? { reason: err.reason, fields: err.fields }
          : undefined,
      );
      return false;
    }
  },
}));

export function recordChange(
  entry: Omit<ChangeEntry, "id" | "at" | "state" | "error" | "refused">,
): void {
  const country = chosenCountry(entry.projectId);
  useChangeLog
    .getState()
    .record(
      entry.workbenchHref
        ? {
            ...entry,
            workbenchHref: withWorkbenchCountry(entry.workbenchHref, country),
          }
        : entry,
    );
  noteEdited(entry.projectId);
}

export function fileUndo(
  projectId: number,
  answer: unknown,
): (() => Promise<void>) | undefined {
  const files = (answer as { undo?: { files?: unknown } } | null | undefined)
    ?.undo?.files;
  if (!Array.isArray(files) || files.length === 0) return undefined;
  return async () => {
    const response = await fetch(
      `/api/go/projects/${projectId}/gui-render/undo-files`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ files }),
      },
    );
    if (!response.ok) {
      const out = (await response.json().catch(() => ({}))) as {
        detail?: string;
      };
      throw new Error(out.detail || `HTTP ${response.status}`);
    }
  };
}

export function pick<T extends object>(
  row: T,
  keys: readonly string[],
): Partial<T> {
  const out: Record<string, unknown> = {};
  for (const key of keys) out[key] = (row as Record<string, unknown>)[key];
  return out as Partial<T>;
}

export class UndoRefused extends Error {
  constructor(
    readonly reason: "gone" | "changedSince" | "notAllBack",
    readonly fields: readonly string[] = [],
  ) {
    super(fields.length ? `${reason}: ${fields.join(", ")}` : reason);
    this.name = "UndoRefused";
  }
}

export function rowUndo<T extends object>({
  before,
  after,
  load,
  save,
}: {
  before: Partial<T>;
  after: Partial<T>;
  load: () => Promise<T | null | undefined>;
  save: (patch: Partial<T>, now: T) => Promise<unknown>;
}): () => Promise<void> {
  return async () => {
    const now = await load();
    if (!now) throw new UndoRefused("gone");
    for (const [key, value] of Object.entries(after)) {
      if (
        JSON.stringify((now as Record<string, unknown>)[key]) !==
        JSON.stringify(value)
      )
        throw new UndoRefused("changedSince");
    }
    await save(before, now);

    const back = await load();
    const stuck = Object.entries(before)
      .filter(
        ([key, value]) =>
          JSON.stringify(
            (back as Record<string, unknown> | null | undefined)?.[key] ?? null,
          ) !== JSON.stringify(value ?? null),
      )
      .map(([key]) => key);
    if (stuck.length) throw new UndoRefused("notAllBack", stuck);
  };
}

export async function textBefore(key: string): Promise<string | null> {
  return (await guiRenderApi.textOverrides())[key] ?? null;
}

export function textUndo(
  key: string,
  wrote: string | null,
  before: string | null,
  shown?: {
    set?: (key: string, value: string) => void;
    cleared?: (key: string) => void;
  },
): () => Promise<void> {
  return async () => {
    if ((await textBefore(key)) !== wrote)
      throw new UndoRefused("changedSince");
    if (before === null) {
      await guiRenderApi.clearText(key);
      shown?.cleared?.(key);
    } else {
      await guiRenderApi.setText(key, before);
      shown?.set?.(key, before);
    }
  };
}

export function undoAll(
  ...ways: ((() => Promise<void>) | undefined)[]
): (() => Promise<void>) | undefined {
  if (ways.length === 0 || ways.some((way) => !way)) return undefined;
  const steps = ways as (() => Promise<void>)[];
  return async () => {
    for (const step of [...steps].reverse()) await step();
  };
}
