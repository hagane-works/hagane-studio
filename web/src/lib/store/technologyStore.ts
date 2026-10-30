import { create } from "zustand";
import type { Technology } from "@/types";
import { technologiesApi } from "@/lib/api";

let latestListRequest = 0;
let latestPreviewRequest = 0;
let activeProjectId: number | null = null;
import type { TechUpdateOp } from "@/lib/techEdgeSync";

type TechHistoryEntry =
  | {
      kind: "update_tech";
      techId: number;
      before: Partial<Technology>;
      after: Partial<Technology>;
    }
  | {
      kind: "update_tech_batch";

      ops: Array<{
        techId: number;
        before: Partial<Technology>;
        after: Partial<Technology>;
      }>;
    }
  | {
      kind: "create_tech";
      techIdString: string;
      techSnapshot: Technology;
    }
  | {
      kind: "delete_tech";
      techSnapshot: Technology;
    };

const UNDO_STACK_CAP = 60;

interface TechnologyStore {
  technologies: Technology[];
  selectedTechLine: string;
  selectedId: number | null;
  scriptPreview: string;
  loading: boolean;
  error: string | null;

  historyUndo: TechHistoryEntry[];
  historyRedo: TechHistoryEntry[];

  isReplaying: boolean;

  fetchTechnologies: (projectId: number, techLine?: string) => Promise<void>;
  setTechLine: (techLine: string) => void;
  selectTech: (id: number | null) => void;
  createTech: (
    projectId: number,
    data: Partial<Technology>,
  ) => Promise<Technology>;

  updateTech: (
    projectId: number,
    id: number,
    data: Partial<Technology>,
    opts?: { optimistic?: boolean },
  ) => Promise<void>;

  updateTechBatch: (projectId: number, ops: TechUpdateOp[]) => Promise<void>;
  deleteTech: (projectId: number, id: number) => Promise<void>;
  fetchPreview: (
    projectId: number,
    techLine: string,
    techId?: number,
  ) => Promise<void>;

  undo: (projectId: number) => Promise<void>;
  redo: (projectId: number) => Promise<void>;
  canUndo: () => boolean;
  canRedo: () => boolean;
}

function pushHistory(
  stack: TechHistoryEntry[],
  entry: TechHistoryEntry,
): TechHistoryEntry[] {
  const next = [...stack, entry];
  if (next.length > UNDO_STACK_CAP) next.shift();
  return next;
}

function findTech(
  technologies: Technology[],
  techId: number,
): Technology | undefined {
  return technologies.find((t) => t.id === techId);
}

function findTechByTechId(
  technologies: Technology[],
  techIdString: string,
): Technology | undefined {
  return technologies.find((t) => t.tech_id === techIdString);
}

function snapshotToCreatePayload(snap: Technology): Partial<Technology> {
  const { id: _id, project_id: _pid, source: _src, ...payload } = snap;
  return payload;
}

export const useTechnologyStore = create<TechnologyStore>((set, get) => ({
  technologies: [],
  selectedTechLine: "infantry",
  selectedId: null,
  scriptPreview: "",
  loading: false,
  error: null,
  historyUndo: [],
  historyRedo: [],
  isReplaying: false,

  fetchTechnologies: async (projectId, techLine) => {
    const line = techLine || get().selectedTechLine;
    const requestId = ++latestListRequest;
    const changedProject = activeProjectId !== projectId;
    activeProjectId = projectId;
    if (changedProject) {
      latestPreviewRequest += 1;
      set({
        technologies: [],
        selectedId: null,
        scriptPreview: "",
        historyUndo: [],
        historyRedo: [],
      });
    }
    set({ loading: true, error: null });
    try {
      const techs = await technologiesApi.list(projectId);
      if (
        requestId !== latestListRequest ||
        activeProjectId !== projectId ||
        get().selectedTechLine !== line
      )
        return;
      set({ technologies: techs });
      get().fetchPreview(projectId, line, get().selectedId ?? undefined);
    } catch (err) {
      if (
        requestId === latestListRequest &&
        activeProjectId === projectId &&
        get().selectedTechLine === line
      ) {
        set({ error: (err as Error).message || "Failed to load technologies" });
      }
    } finally {
      if (
        requestId === latestListRequest &&
        activeProjectId === projectId &&
        get().selectedTechLine === line
      )
        set({ loading: false });
    }
  },

  setTechLine: (techLine) => {
    latestPreviewRequest += 1;

    set({
      selectedTechLine: techLine,
      selectedId: null,
      scriptPreview: "",
      historyUndo: [],
      historyRedo: [],
    });
  },

  selectTech: (id) => {
    latestPreviewRequest += 1;
    set({ selectedId: id, scriptPreview: "" });
  },

  createTech: async (projectId, data) => {
    const tech = await technologiesApi.create(projectId, data);
    set((s) => {
      const next: Partial<TechnologyStore> = {
        technologies: [...s.technologies, tech],
        selectedId: tech.id,
      };

      if (!s.isReplaying) {
        next.historyUndo = pushHistory(s.historyUndo, {
          kind: "create_tech",
          techIdString: tech.tech_id,
          techSnapshot: tech,
        });
        next.historyRedo = [];
      }
      return next as TechnologyStore;
    });
    get().fetchPreview(
      projectId,
      get().selectedTechLine,
      get().selectedId ?? undefined,
    );
    return tech;
  },

  updateTech: async (projectId, id, data, opts) => {
    const beforeTech = findTech(get().technologies, id);
    const before: Partial<Technology> = {};
    if (beforeTech) {
      for (const key of Object.keys(data)) {
        (before as any)[key] = (beforeTech as any)[key];
      }
    }

    const changed =
      beforeTech != null &&
      Object.keys(data).some(
        (key) =>
          JSON.stringify((data as any)[key]) !==
          JSON.stringify((before as any)[key]),
      );

    if (opts?.optimistic && beforeTech) {
      set((s) => ({
        technologies: s.technologies.map((t) =>
          t.id === id
            ? ({ ...t, ...data, source: "created" } as Technology)
            : t,
        ),
      }));
    }

    let updated: Technology;
    try {
      updated = await technologiesApi.update(projectId, id, data);
    } catch (err) {
      if (opts?.optimistic && beforeTech) {
        set((s) => ({
          technologies: s.technologies.map((t) =>
            t.id === id ? beforeTech : t,
          ),
        }));
      }
      set({ error: err instanceof Error ? err.message : String(err) });
      console.error("Failed to update tech:", err);
      throw err;
    }

    set((s) => {
      const next: Partial<TechnologyStore> = {
        technologies: s.technologies.map((t) => (t.id === id ? updated : t)),
      };
      if (!s.isReplaying && changed) {
        next.historyUndo = pushHistory(s.historyUndo, {
          kind: "update_tech",
          techId: id,
          before,
          after: data,
        });
        next.historyRedo = [];
      }
      return next as TechnologyStore;
    });
    get().fetchPreview(
      projectId,
      get().selectedTechLine,
      get().selectedId ?? undefined,
    );
  },

  updateTechBatch: async (projectId, ops) => {
    const succeeded: Array<{
      techId: number;
      before: Partial<Technology>;
      after: Partial<Technology>;
    }> = [];
    for (const op of ops) {
      const beforeTech = findTech(get().technologies, op.id);
      const before: Partial<Technology> = {};
      if (beforeTech) {
        for (const key of Object.keys(op.data)) {
          (before as any)[key] = (beforeTech as any)[key];
        }
      }
      const changed =
        beforeTech != null &&
        Object.keys(op.data).some(
          (key) =>
            JSON.stringify((op.data as any)[key]) !==
            JSON.stringify((before as any)[key]),
        );
      let updated: Technology;
      try {
        updated = await technologiesApi.update(projectId, op.id, op.data);
      } catch (err) {
        set({ error: err instanceof Error ? err.message : String(err) });
        console.error("Failed to update tech (batch):", err);
        break;
      }
      set((s) => ({
        technologies: s.technologies.map((t) => (t.id === op.id ? updated : t)),
      }));
      if (changed) succeeded.push({ techId: op.id, before, after: op.data });
    }
    if (succeeded.length > 0 && !get().isReplaying) {
      set((s) => ({
        historyUndo: pushHistory(s.historyUndo, {
          kind: "update_tech_batch",
          ops: succeeded,
        }),
        historyRedo: [],
      }));
    }
    get().fetchPreview(
      projectId,
      get().selectedTechLine,
      get().selectedId ?? undefined,
    );
  },

  deleteTech: async (projectId, id) => {
    const snapshot = findTech(get().technologies, id);
    await technologiesApi.delete(projectId, id);
    set((s) => {
      const next: Partial<TechnologyStore> = {
        technologies: s.technologies.filter((t) => t.id !== id),
        selectedId: s.selectedId === id ? null : s.selectedId,
      };
      if (!s.isReplaying && snapshot) {
        next.historyUndo = pushHistory(s.historyUndo, {
          kind: "delete_tech",
          techSnapshot: snapshot,
        });
        next.historyRedo = [];
      }
      return next as TechnologyStore;
    });
    get().fetchPreview(
      projectId,
      get().selectedTechLine,
      get().selectedId ?? undefined,
    );
  },

  fetchPreview: async (projectId, techLine, techId?) => {
    const selectedId = techId ?? null;
    const requestId = ++latestPreviewRequest;
    try {
      const res = await technologiesApi.preview(projectId, techLine, techId);
      if (
        requestId === latestPreviewRequest &&
        activeProjectId === projectId &&
        get().selectedTechLine === techLine &&
        get().selectedId === selectedId
      ) {
        set({ scriptPreview: res.script });
      }
    } catch {
      if (
        requestId === latestPreviewRequest &&
        activeProjectId === projectId &&
        get().selectedTechLine === techLine &&
        get().selectedId === selectedId
      ) {
        set({ scriptPreview: "# Preview unavailable" });
      }
    }
  },

  canUndo: () => get().historyUndo.length > 0,
  canRedo: () => get().historyRedo.length > 0,

  undo: async (projectId) => {
    const state = get();
    if (state.historyUndo.length === 0) return;
    const entry = state.historyUndo[state.historyUndo.length - 1];
    set({
      historyUndo: state.historyUndo.slice(0, -1),
      isReplaying: true,
    });
    try {
      if (entry.kind === "update_tech") {
        await technologiesApi.update(projectId, entry.techId, entry.before);

        set((s) => ({
          technologies: s.technologies.map((t) =>
            t.id === entry.techId ? { ...t, ...entry.before } : t,
          ),
          historyRedo: pushHistory(s.historyRedo, entry),
        }));
      } else if (entry.kind === "update_tech_batch") {
        for (let i = entry.ops.length - 1; i >= 0; i--) {
          const op = entry.ops[i];
          await technologiesApi.update(projectId, op.techId, op.before);
          set((s) => ({
            technologies: s.technologies.map((t) =>
              t.id === op.techId ? { ...t, ...op.before } : t,
            ),
          }));
        }
        set((s) => ({ historyRedo: pushHistory(s.historyRedo, entry) }));
      } else if (entry.kind === "create_tech") {
        const current = findTechByTechId(
          get().technologies,
          entry.techIdString,
        );
        if (current) {
          await technologiesApi.delete(projectId, current.id);
          set((s) => ({
            technologies: s.technologies.filter((t) => t.id !== current.id),
            selectedId: s.selectedId === current.id ? null : s.selectedId,
            historyRedo: pushHistory(s.historyRedo, entry),
          }));
        }
      } else if (entry.kind === "delete_tech") {
        const recreated = await technologiesApi.create(
          projectId,
          snapshotToCreatePayload(entry.techSnapshot),
        );
        set((s) => ({
          technologies: [...s.technologies, recreated],

          historyRedo: pushHistory(s.historyRedo, {
            kind: "delete_tech",
            techSnapshot: recreated,
          }),
        }));
      }
      get().fetchPreview(
        projectId,
        get().selectedTechLine,
        get().selectedId ?? undefined,
      );
    } catch (err) {
      console.error("[tech undo] failed:", err);

      set((s) => ({ historyUndo: pushHistory(s.historyUndo, entry) }));
    } finally {
      set({ isReplaying: false });
    }
  },

  redo: async (projectId) => {
    const state = get();
    if (state.historyRedo.length === 0) return;
    const entry = state.historyRedo[state.historyRedo.length - 1];
    set({
      historyRedo: state.historyRedo.slice(0, -1),
      isReplaying: true,
    });
    try {
      if (entry.kind === "update_tech") {
        await technologiesApi.update(projectId, entry.techId, entry.after);
        set((s) => ({
          technologies: s.technologies.map((t) =>
            t.id === entry.techId ? { ...t, ...entry.after } : t,
          ),
          historyUndo: pushHistory(s.historyUndo, entry),
        }));
      } else if (entry.kind === "update_tech_batch") {
        for (const op of entry.ops) {
          await technologiesApi.update(projectId, op.techId, op.after);
          set((s) => ({
            technologies: s.technologies.map((t) =>
              t.id === op.techId ? { ...t, ...op.after } : t,
            ),
          }));
        }
        set((s) => ({ historyUndo: pushHistory(s.historyUndo, entry) }));
      } else if (entry.kind === "create_tech") {
        const recreated = await technologiesApi.create(
          projectId,
          snapshotToCreatePayload(entry.techSnapshot),
        );
        set((s) => ({
          technologies: [...s.technologies, recreated],
          historyUndo: pushHistory(s.historyUndo, {
            kind: "create_tech",
            techIdString: recreated.tech_id,
            techSnapshot: recreated,
          }),
        }));
      } else if (entry.kind === "delete_tech") {
        const current = findTechByTechId(
          get().technologies,
          entry.techSnapshot.tech_id,
        );
        if (current) {
          await technologiesApi.delete(projectId, current.id);
          set((s) => ({
            technologies: s.technologies.filter((t) => t.id !== current.id),
            selectedId: s.selectedId === current.id ? null : s.selectedId,
            historyUndo: pushHistory(s.historyUndo, {
              kind: "delete_tech",
              techSnapshot: current,
            }),
          }));
        }
      }
      get().fetchPreview(
        projectId,
        get().selectedTechLine,
        get().selectedId ?? undefined,
      );
    } catch (err) {
      console.error("[tech redo] failed:", err);
      set((s) => ({ historyRedo: pushHistory(s.historyRedo, entry) }));
    } finally {
      set({ isReplaying: false });
    }
  },
}));
