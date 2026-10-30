import { create } from "zustand";

export interface EditorDirtyState {
  dirtyWorkbenches: Record<string, boolean>;

  confirmMessage: string;
  setDirty: (key: string, isDirty: boolean) => void;

  hasAnyDirty: () => boolean;
  clearAll: () => void;
  setConfirmMessage: (msg: string) => void;
}

const EMPTY_DIRTY: Record<string, boolean> = Object.freeze({});

export const useEditorDirtyStore = create<EditorDirtyState>((set, get) => ({
  dirtyWorkbenches: EMPTY_DIRTY,
  confirmMessage: "Discard unsaved changes?",
  setDirty: (key, isDirty) =>
    set((s) => {
      const current = !!s.dirtyWorkbenches[key];
      if (current === isDirty) return s;
      const next = { ...s.dirtyWorkbenches };
      if (isDirty) next[key] = true;
      else delete next[key];
      return { dirtyWorkbenches: next };
    }),
  hasAnyDirty: () => {
    const map = get().dirtyWorkbenches;
    for (const k in map) {
      if (map[k]) return true;
    }
    return false;
  },
  clearAll: () =>
    set((s) =>
      s.dirtyWorkbenches === EMPTY_DIRTY
        ? s
        : { dirtyWorkbenches: EMPTY_DIRTY },
    ),
  setConfirmMessage: (msg) => set({ confirmMessage: msg }),
}));
