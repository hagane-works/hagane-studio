import { create } from "zustand";
import type { Idea, IdeaType } from "@/types";
import { ideasApi } from "@/lib/api";

let latestListRequest = 0;
let latestPreviewRequest = 0;
let activeProjectId: number | null = null;

interface IdeaStore {
  ideas: Idea[];
  selectedId: number | null;
  filterType: IdeaType | "";
  scriptPreview: string;
  loading: boolean;
  error: string | null;

  fetchIdeas: (projectId: number, ideaType?: string) => Promise<void>;
  setFilterType: (type: IdeaType | "") => void;
  selectIdea: (id: number | null) => void;
  createIdea: (projectId: number, data: Partial<Idea>) => Promise<Idea>;
  updateIdea: (
    projectId: number,
    id: number,
    data: Partial<Idea>,
  ) => Promise<void>;
  deleteIdea: (projectId: number, id: number) => Promise<void>;
  fetchPreview: (projectId: number, ideaId?: number) => Promise<void>;
}

export const useIdeaStore = create<IdeaStore>((set, get) => ({
  ideas: [],
  selectedId: null,
  filterType: "",
  scriptPreview: "",
  loading: false,
  error: null,

  fetchIdeas: async (projectId, ideaType) => {
    const type = ideaType ?? get().filterType;
    const requestId = ++latestListRequest;
    const changedProject = activeProjectId !== projectId;
    activeProjectId = projectId;
    if (changedProject) {
      latestPreviewRequest += 1;
      set({ ideas: [], selectedId: null, scriptPreview: "" });
    }
    set({ loading: true, error: null });
    try {
      const ideas = await ideasApi.list(projectId, type || undefined);
      if (
        requestId !== latestListRequest ||
        activeProjectId !== projectId ||
        get().filterType !== type
      )
        return;
      set({ ideas });

      if (!get().selectedId && ideas.length > 0) {
        set({ selectedId: ideas[0].id });
      }
      get().fetchPreview(projectId);
    } catch (err) {
      if (
        requestId === latestListRequest &&
        activeProjectId === projectId &&
        get().filterType === type
      ) {
        set({ error: (err as Error).message || "Failed to load ideas" });
      }
    } finally {
      if (
        requestId === latestListRequest &&
        activeProjectId === projectId &&
        get().filterType === type
      )
        set({ loading: false });
    }
  },

  setFilterType: (type) => {
    latestListRequest += 1;
    latestPreviewRequest += 1;
    set({ filterType: type, selectedId: null });
  },

  selectIdea: (id) => {
    latestPreviewRequest += 1;
    set({ selectedId: id });
  },

  createIdea: async (projectId, data) => {
    let idea;
    try {
      idea = await ideasApi.create(projectId, data);
    } catch (err) {
      const msg = (err as Error).message || "";
      if (msg.includes("already exists") || msg.includes("race-collision")) {
        return null as unknown as Idea;
      }
      throw err;
    }
    set((s) => ({
      ideas: [...s.ideas, idea],
      selectedId: idea.id,
    }));
    get().fetchPreview(projectId);
    return idea;
  },

  updateIdea: async (projectId, id, data) => {
    const updated = await ideasApi.update(projectId, id, data);
    set((s) => ({
      ideas: s.ideas.map((i) => (i.id === id ? updated : i)),
    }));
    get().fetchPreview(projectId);
  },

  deleteIdea: async (projectId, id) => {
    await ideasApi.delete(projectId, id);
    set((s) => ({
      ideas: s.ideas.filter((i) => i.id !== id),
      selectedId: s.selectedId === id ? null : s.selectedId,
    }));
    get().fetchPreview(projectId);
  },

  fetchPreview: async (projectId, ideaId?) => {
    const selectedId = ideaId ?? get().selectedId;
    const requestId = ++latestPreviewRequest;
    try {
      const res = await ideasApi.preview(projectId, selectedId ?? undefined);
      if (
        requestId === latestPreviewRequest &&
        activeProjectId === projectId &&
        get().selectedId === selectedId
      ) {
        set({ scriptPreview: res.script });
      }
    } catch {
      if (
        requestId === latestPreviewRequest &&
        activeProjectId === projectId &&
        get().selectedId === selectedId
      ) {
        set({ scriptPreview: "# Preview unavailable" });
      }
    }
  },
}));
