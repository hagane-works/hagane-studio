import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";

export type BackgroundImportStage =
  | "queued"
  | "scanning"
  | "parsing"
  | "uploading"
  | "done"
  | "error"
  | "cancelled";

const TERMINAL_STAGE_SET: ReadonlySet<BackgroundImportStage> =
  new Set<BackgroundImportStage>(["done", "error", "cancelled"]);

export interface BackgroundImportResult {
  status: "success" | "partial" | "failed" | "cancelled";

  headline: string;

  lines: string[];

  interrupted?: boolean;
}

export interface BackgroundImportSession {
  sessionId: string;
  projectId: number;
  projectName: string;
  countryTag: string;
  stage: BackgroundImportStage;

  statusText: string;

  percent: number | null;
  etaSeconds: number | null;

  startedAt: number;

  finishedAt?: number;

  finalMessage?: string;

  result?: BackgroundImportResult;
}

interface BackgroundImportStore {
  session: BackgroundImportSession | null;
  setBackgroundImport: (session: BackgroundImportSession) => void;
  updateBackgroundImport: (
    sessionId: string,
    patch: Partial<
      Omit<BackgroundImportSession, "sessionId" | "projectId" | "startedAt">
    >,
  ) => void;
  cancelBackgroundImport: () => Promise<void>;
  clearBackgroundImport: () => void;
}

const PERSIST_KEY = "hoi4-background-import-v2";

export const useBackgroundImportStore = create<BackgroundImportStore>()(
  persist(
    (set, get) => ({
      session: null,

      setBackgroundImport: (session) => {
        set({ session });
      },

      updateBackgroundImport: (sessionId, patch) => {
        const current = get().session;
        if (!current || current.sessionId !== sessionId) return;
        set({ session: { ...current, ...patch } });
      },

      cancelBackgroundImport: async () => {
        const current = get().session;
        if (!current) return;

        set({ session: null });
        try {
          await fetch(
            `/api/go/import/jobs/${current.projectId}/${encodeURIComponent(current.sessionId)}/cancel`,
            { method: "POST", credentials: "include" },
          );
        } catch (err) {
          console.warn("[backgroundImport] cancel POST failed:", err);
        }
      },

      clearBackgroundImport: () => {
        set({ session: null });
      },
    }),
    {
      name: PERSIST_KEY,
      storage: createJSONStorage(() =>
        typeof window !== "undefined" ? window.localStorage : undefined!,
      ),

      skipHydration: true,

      partialize: (s) => ({ session: s.session }),
      onRehydrateStorage: () => (state) => {
        if (!state?.session) return;
        const s = state.session;
        if (!TERMINAL_STAGE_SET.has(s.stage)) {
          state.session = {
            ...s,
            stage: "cancelled",
            percent: null,
            statusText: s.statusText,
            result: {
              status: "cancelled",

              headline: s.result?.headline ?? "",
              lines: s.result?.lines ?? [],
              interrupted: true,
            },
          };
        }
      },
    },
  ),
);

export const useBackgroundImportSession = () =>
  useBackgroundImportStore((s) => s.session);

export const useSetBackgroundImport = () =>
  useBackgroundImportStore((s) => s.setBackgroundImport);

export const useUpdateBackgroundImport = () =>
  useBackgroundImportStore((s) => s.updateBackgroundImport);

export const useCancelBackgroundImport = () =>
  useBackgroundImportStore((s) => s.cancelBackgroundImport);

export const useClearBackgroundImport = () =>
  useBackgroundImportStore((s) => s.clearBackgroundImport);

export const rehydrateBackgroundImport = () => {
  void useBackgroundImportStore.persist.rehydrate();
};
