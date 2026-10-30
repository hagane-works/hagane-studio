import { create } from "zustand";
import type { Character, RoleType } from "@/types";
import { charactersApi } from "@/lib/api";

interface CharacterStore {
  characters: Character[];
  selectedId: number | null;
  scriptPreview: string;
  loading: boolean;
  error: string | null;
  roleTypeFilter: RoleType | "";

  fetchCharacters: (projectId: number) => Promise<void>;
  selectCharacter: (id: number | null) => void;
  setRoleTypeFilter: (role: RoleType | "") => void;
  createCharacter: (
    projectId: number,
    data: Partial<Character>,
  ) => Promise<Character>;
  updateCharacter: (
    projectId: number,
    id: number,
    data: Partial<Character>,
  ) => Promise<void>;
  deleteCharacter: (projectId: number, id: number) => Promise<void>;
  fetchPreview: (projectId: number, id: number) => Promise<void>;
}

let latestListRequest = 0;
let latestPreviewRequest = 0;
let activeProjectId: number | null = null;

export const useCharacterStore = create<CharacterStore>((set, get) => ({
  characters: [],
  selectedId: null,
  scriptPreview: "",
  loading: false,
  error: null,
  roleTypeFilter: "",

  fetchCharacters: async (projectId) => {
    const requestId = ++latestListRequest;
    const changedProject = activeProjectId !== projectId;
    activeProjectId = projectId;
    if (changedProject) {
      latestPreviewRequest += 1;
      set({ characters: [], selectedId: null, scriptPreview: "" });
    }
    set({ loading: true, error: null });
    try {
      const chars = await charactersApi.list(projectId);
      if (requestId === latestListRequest && activeProjectId === projectId) {
        set({ characters: chars });
      }
    } catch (err) {
      if (requestId === latestListRequest && activeProjectId === projectId) {
        set({ error: (err as Error).message || "Failed to load characters" });
      }
    } finally {
      if (requestId === latestListRequest && activeProjectId === projectId) {
        set({ loading: false });
      }
    }
  },

  selectCharacter: (id) => {
    latestPreviewRequest += 1;
    set({ selectedId: id, scriptPreview: "" });
  },

  setRoleTypeFilter: (role) => set({ roleTypeFilter: role }),

  createCharacter: async (projectId, data) => {
    let char;
    try {
      char = await charactersApi.create(projectId, data);
    } catch (err) {
      const msg = (err as Error).message || "";
      if (msg.includes("already exists") || msg.includes("race-collision")) {
        return null as unknown as Character;
      }
      throw err;
    }
    set((s) => ({ characters: [...s.characters, char], selectedId: char.id }));
    return char;
  },

  updateCharacter: async (projectId, id, data) => {
    const updated = await charactersApi.update(projectId, id, data);
    set((s) => ({
      characters: s.characters.map((c) => (c.id === id ? updated : c)),
    }));

    if (get().selectedId === id) {
      get().fetchPreview(projectId, id);
    }
  },

  deleteCharacter: async (projectId, id) => {
    await charactersApi.delete(projectId, id);
    set((s) => ({
      characters: s.characters.filter((c) => c.id !== id),
      selectedId: s.selectedId === id ? null : s.selectedId,
      scriptPreview: s.selectedId === id ? "" : s.scriptPreview,
    }));
  },

  fetchPreview: async (projectId, id) => {
    const requestId = ++latestPreviewRequest;
    try {
      const res = await charactersApi.preview(projectId, id);
      if (requestId === latestPreviewRequest && get().selectedId === id) {
        set({ scriptPreview: res.script });
      }
    } catch {
      if (requestId === latestPreviewRequest && get().selectedId === id) {
        set({ scriptPreview: "# Preview unavailable" });
      }
    }
  },
}));
