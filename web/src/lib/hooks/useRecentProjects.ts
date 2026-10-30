import { useCallback, useMemo, useSyncExternalStore } from "react";
import type { Project } from "@/types";

const STORAGE_KEY = "hoi4-recent-projects";
const MAX_ENTRIES = 10;

interface RecentEntry {
  id: number;
  last_opened_at: string;
}

interface StoredShape {
  entries: RecentEntry[];
}

const EMPTY_ENTRIES: RecentEntry[] = [];

function readEntries(): RecentEntry[] {
  if (typeof window === "undefined") return EMPTY_ENTRIES;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return EMPTY_ENTRIES;
    const parsed = JSON.parse(raw) as StoredShape;
    if (!parsed || !Array.isArray(parsed.entries)) return EMPTY_ENTRIES;

    return parsed.entries.filter(
      (e): e is RecentEntry =>
        e !== null &&
        typeof e === "object" &&
        typeof e.id === "number" &&
        typeof e.last_opened_at === "string",
    );
  } catch {
    return EMPTY_ENTRIES;
  }
}

function writeEntries(entries: RecentEntry[]): void {
  if (typeof window === "undefined") return;
  try {
    const payload: StoredShape = { entries };
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
  } catch {}
}

let cachedRaw: string | null = null;
let cachedEntries: RecentEntry[] = EMPTY_ENTRIES;

function getEntriesSnapshot(): RecentEntry[] {
  if (typeof window === "undefined") return EMPTY_ENTRIES;
  const raw = window.localStorage.getItem(STORAGE_KEY);
  if (raw === cachedRaw) return cachedEntries;
  cachedRaw = raw;
  cachedEntries = readEntries();
  return cachedEntries;
}

function getServerSnapshot(): RecentEntry[] {
  return EMPTY_ENTRIES;
}

function subscribeEntries(onStoreChange: () => void): () => void {
  if (typeof window === "undefined") return () => {};
  const onUpdate = () => onStoreChange();
  const onStorage = (e: StorageEvent) => {
    if (e.key === STORAGE_KEY) onStoreChange();
  };
  window.addEventListener("hoi4-recent-projects-updated", onUpdate);
  window.addEventListener("storage", onStorage);
  return () => {
    window.removeEventListener("hoi4-recent-projects-updated", onUpdate);
    window.removeEventListener("storage", onStorage);
  };
}

export function markRecentlyOpened(projectId: number): void {
  if (typeof window === "undefined") return;
  const now = new Date().toISOString();
  const current = readEntries();
  const filtered = current.filter((e) => e.id !== projectId);
  const next: RecentEntry[] = [
    { id: projectId, last_opened_at: now },
    ...filtered,
  ].slice(0, MAX_ENTRIES);
  writeEntries(next);

  window.dispatchEvent(new CustomEvent("hoi4-recent-projects-updated"));
}

export interface RecentProjectView {
  project: Project;
  last_opened_at: string;
}

export function useRecentProjects(
  projects: Project[],
  limit: number = 3,
): RecentProjectView[] {
  const entries = useSyncExternalStore(
    subscribeEntries,
    getEntriesSnapshot,
    getServerSnapshot,
  );

  return useMemo(() => {
    if (entries.length === 0 || projects.length === 0) return [];
    const byId = new Map(projects.map((p) => [p.id, p] as const));
    const out: RecentProjectView[] = [];
    for (const entry of entries) {
      const proj = byId.get(entry.id);
      if (!proj) continue;
      out.push({ project: proj, last_opened_at: entry.last_opened_at });
      if (out.length >= limit) break;
    }
    return out;
  }, [entries, projects, limit]);
}

export function useMarkOpenedHandler() {
  return useCallback((projectId: number, after?: () => void) => {
    markRecentlyOpened(projectId);
    if (after) after();
  }, []);
}
