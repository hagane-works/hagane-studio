import { create } from "zustand";

import type { CtdIssue } from "@/lib/api";
import type {
  AssetAudit,
  ExportSummary,
} from "@/components/export/ExportContentsCard";

export interface ExportCheckRun {
  projectId: number;

  key: string;
  issues: CtdIssue[];

  checkFailed: boolean;
  summary: ExportSummary | null;
  assetAudit: AssetAudit | null;
  summaryFailed: boolean;

  ranAt: number;
}

export interface ExportCheckRequest {
  projectId: number;
  key: string;
  generation: number;
}

export const EXPORT_CHECK_TIMEOUT_MS = 90_000;

export type TimedExportCheckResult<T> =
  { kind: "completed"; value: T } | { kind: "failed" } | { kind: "timed-out" };

export function settleExportCheckWithinTimeout<T>(
  work: Promise<T>,
  timeoutMs = EXPORT_CHECK_TIMEOUT_MS,
): Promise<TimedExportCheckResult<T>> {
  return new Promise((resolve) => {
    let settled = false;
    const timer = setTimeout(() => {
      settled = true;
      resolve({ kind: "timed-out" });
    }, timeoutMs);
    void work.then(
      (value) => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        resolve({ kind: "completed", value });
      },
      () => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        resolve({ kind: "failed" });
      },
    );
  });
}

interface ExportCheckState {
  runsByProject: Record<number, ExportCheckRun>;
  activeByProject: Record<number, ExportCheckRequest>;
  nextGeneration: number;
  startRun: (projectId: number, key: string) => ExportCheckRequest | null;
  finishRun: (request: ExportCheckRequest, run: ExportCheckRun) => void;
  getRun: (projectId: number) => ExportCheckRun | null;
  isRunning: (projectId: number) => boolean;
}

function keepRecentRuns(
  runsByProject: Record<number, ExportCheckRun>,
): Record<number, ExportCheckRun> {
  return Object.fromEntries(
    Object.entries(runsByProject)
      .sort(([, left], [, right]) => right.ranAt - left.ranAt)
      .slice(0, 2),
  ) as Record<number, ExportCheckRun>;
}

export const useExportCheckStore = create<ExportCheckState>((set, get) => ({
  runsByProject: {},
  activeByProject: {},
  nextGeneration: 0,
  startRun: (projectId, key) => {
    let request: ExportCheckRequest | null = null;
    set((state) => {
      if (state.activeByProject[projectId]) return state;
      const generation = state.nextGeneration + 1;
      const nextRequest: ExportCheckRequest = { projectId, key, generation };
      request = nextRequest;
      return {
        nextGeneration: generation,
        activeByProject: {
          ...state.activeByProject,
          [projectId]: nextRequest,
        },
      };
    });
    return request;
  },
  finishRun: (request, run) =>
    set((state) => {
      const active = state.activeByProject[request.projectId];
      if (
        !active ||
        active.generation !== request.generation ||
        active.key !== request.key
      ) {
        return state;
      }
      const remainingActive = { ...state.activeByProject };
      delete remainingActive[request.projectId];
      return {
        activeByProject: remainingActive,
        runsByProject: keepRecentRuns({
          ...state.runsByProject,
          [request.projectId]: run,
        }),
      };
    }),
  getRun: (projectId) => get().runsByProject[projectId] ?? null,
  isRunning: (projectId) => Boolean(get().activeByProject[projectId]),
}));

export type StaleReason = "settings" | "project-edited";

export function checkStaleReason(
  run: ExportCheckRun | null,
  currentKey: string,
  projectUpdatedAt: string | null | undefined,
): StaleReason | null {
  if (!run) return null;
  if (run.key !== currentKey) return "settings";
  if (!projectUpdatedAt) return null;
  const updated = Date.parse(projectUpdatedAt);

  if (Number.isNaN(updated)) return null;
  return updated > run.ranAt ? "project-edited" : null;
}
