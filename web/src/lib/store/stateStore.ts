import { create } from "zustand";
import type { State, StateUpdate } from "@/types";
import { statesApi } from "@/lib/api";
import {
  planProvinceMove,
  type DonorEdit,
  type ProvinceMovePlan,
} from "@/lib/states/provinceIndex";

export interface ProvinceMoveRecord {
  id: number;
  at: number;
  target: {
    rowId: number;
    stateId: number;
    name: string;

    restore: number[];
  };
  donors: {
    rowId: number;
    stateId: number;
    name: string;

    restore: number[];
    taken: number[];
  }[];
  movedProvinceIds: number[];
}

export interface ProvinceMoveFailure {
  kind: "move_failed" | "undo_failed";
  provinceIds: number[];
  stateIds: number[];

  detail: string;
}

let moveSeq = 0;
let latestListRequest = 0;
let activeProjectId: number | null = null;

interface StateStore {
  states: State[];
  selectedId: number | null;
  loading: boolean;
  error: string | null;

  lastProvinceMove: ProvinceMoveRecord | null;
  provinceMoveFailure: ProvinceMoveFailure | null;
  undoingProvinceMove: boolean;

  fetchStates: (projectId: number) => Promise<void>;
  selectState: (id: number | null) => void;
  updateState: (
    projectId: number,
    id: number,
    data: StateUpdate,
  ) => Promise<State>;
  undoProvinceMove: (projectId: number) => Promise<void>;
  dismissProvinceMove: () => void;
}

export const useStateStore = create<StateStore>((set, get) => ({
  states: [],
  selectedId: null,
  loading: false,
  error: null,
  lastProvinceMove: null,
  provinceMoveFailure: null,
  undoingProvinceMove: false,

  fetchStates: async (projectId) => {
    const requestId = ++latestListRequest;
    const changedProject = activeProjectId !== projectId;
    activeProjectId = projectId;
    if (changedProject) {
      set({
        states: [],
        selectedId: null,
        lastProvinceMove: null,
        provinceMoveFailure: null,
      });
    }
    set({ loading: true, error: null });
    try {
      const states = await statesApi.list(projectId);
      if (requestId !== latestListRequest || activeProjectId !== projectId)
        return;
      set({ states });

      const sel = get().selectedId;
      if (sel != null && !states.some((s) => s.id === sel)) {
        set({ selectedId: null });
      }
    } catch (err) {
      if (requestId === latestListRequest && activeProjectId === projectId) {
        set({ error: (err as Error).message || "Failed to load states" });
      }
    } finally {
      if (requestId === latestListRequest && activeProjectId === projectId)
        set({ loading: false });
    }
  },

  selectState: (id) => set({ selectedId: id }),

  updateState: async (projectId, id, data) => {
    const replaceRow = (row: State) =>
      set((s) => ({
        states: s.states.map((st) => (st.id === row.id ? row : st)),
      }));

    const plan: ProvinceMovePlan | null = data.provinces
      ? planProvinceMove(get().states, id, data.provinces)
      : null;
    const donors: DonorEdit[] = plan?.donors ?? [];

    if (donors.length === 0) {
      const updated = await statesApi.update(projectId, id, data);
      replaceRow(updated);
      return updated;
    }

    const written: DonorEdit[] = [];
    const rollback = async () => {
      for (const d of [...written].reverse()) {
        try {
          const back = await statesApi.update(projectId, d.rowId, {
            provinces: d.prevProvinces,
          });
          replaceRow(back);
        } catch {}
      }
    };

    for (const d of donors) {
      try {
        const moved = await statesApi.update(projectId, d.rowId, {
          provinces: d.nextProvinces,
        });
        replaceRow(moved);
        written.push(d);
      } catch (err) {
        await rollback();
        set({
          provinceMoveFailure: {
            kind: "move_failed",
            provinceIds: plan?.movedProvinceIds ?? [],
            stateIds: donors.map((x) => x.stateId),
            detail: (err as Error).message || String(err),
          },
        });
        throw err;
      }
    }

    let updated: State;
    try {
      updated = await statesApi.update(projectId, id, data);
    } catch (err) {
      await rollback();
      set({
        provinceMoveFailure: {
          kind: "move_failed",
          provinceIds: plan?.movedProvinceIds ?? [],
          stateIds: donors.map((x) => x.stateId),
          detail: (err as Error).message || String(err),
        },
      });
      throw err;
    }
    replaceRow(updated);

    moveSeq += 1;
    set({
      provinceMoveFailure: null,
      lastProvinceMove: {
        id: moveSeq,
        at: Date.now(),
        target: {
          rowId: id,
          stateId: plan?.targetStateId ?? updated.state_id,
          name: plan?.targetName ?? updated.name ?? "",
          restore: plan?.targetPrevProvinces ?? [],
        },
        donors: donors.map((d) => ({
          rowId: d.rowId,
          stateId: d.stateId,
          name: d.name,
          restore: d.prevProvinces,
          taken: d.taken,
        })),
        movedProvinceIds: plan?.movedProvinceIds ?? [],
      },
    });
    return updated;
  },

  undoProvinceMove: async (projectId) => {
    const rec = get().lastProvinceMove;
    if (!rec || get().undoingProvinceMove) return;
    const replaceRow = (row: State) =>
      set((s) => ({
        states: s.states.map((st) => (st.id === row.id ? row : st)),
      }));

    set({ undoingProvinceMove: true });
    try {
      const back = await statesApi.update(projectId, rec.target.rowId, {
        provinces: rec.target.restore,
      });
      replaceRow(back);
      for (const d of rec.donors) {
        const restored = await statesApi.update(projectId, d.rowId, {
          provinces: d.restore,
        });
        replaceRow(restored);
      }
      set({ lastProvinceMove: null, provinceMoveFailure: null });
    } catch (err) {
      set({
        provinceMoveFailure: {
          kind: "undo_failed",
          provinceIds: rec.movedProvinceIds,
          stateIds: [rec.target.stateId, ...rec.donors.map((d) => d.stateId)],
          detail: (err as Error).message || String(err),
        },
      });
    } finally {
      set({ undoingProvinceMove: false });
    }
  },

  dismissProvinceMove: () =>
    set({ lastProvinceMove: null, provinceMoveFailure: null }),
}));
