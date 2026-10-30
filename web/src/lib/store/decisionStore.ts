import { create } from "zustand";
import type { Decision, DecisionCategory } from "@/types";
import { decisionsApi } from "@/lib/api";

let latestCategoryRequest = 0;
let latestDecisionRequest = 0;
let latestPreviewRequest = 0;
let activeProjectId: number | null = null;

interface DecisionStore {
  categories: DecisionCategory[];
  decisions: Decision[];
  selectedCatId: number | null;
  selectedDecisionId: number | null;
  scriptPreview: string;
  loading: boolean;
  error: string | null;

  fetchCategories: (projectId: number) => Promise<void>;
  selectCategory: (catId: number | null) => void;
  createCategory: (
    projectId: number,
    categoryId: string,
    countryTag?: string,
  ) => Promise<DecisionCategory>;
  updateCategory: (
    projectId: number,
    catId: number,
    data: Partial<DecisionCategory>,
  ) => Promise<void>;
  deleteCategory: (projectId: number, catId: number) => Promise<void>;

  fetchDecisions: (projectId: number, categoryId?: number) => Promise<void>;
  selectDecision: (id: number | null) => void;
  createDecision: (
    projectId: number,
    data: Partial<Decision>,
  ) => Promise<Decision>;
  updateDecision: (
    projectId: number,
    id: number,
    data: Partial<Decision>,
  ) => Promise<void>;
  deleteDecision: (projectId: number, id: number) => Promise<void>;
  fetchPreview: (projectId: number, decisionId: number) => Promise<void>;
}

export const useDecisionStore = create<DecisionStore>((set, get) => ({
  categories: [],
  decisions: [],
  selectedCatId: null,
  selectedDecisionId: null,
  scriptPreview: "",
  loading: false,
  error: null,

  fetchCategories: async (projectId) => {
    const requestId = ++latestCategoryRequest;
    const changedProject = activeProjectId !== projectId;
    activeProjectId = projectId;
    if (changedProject) {
      latestDecisionRequest += 1;
      latestPreviewRequest += 1;
      set({
        categories: [],
        decisions: [],
        selectedCatId: null,
        selectedDecisionId: null,
        scriptPreview: "",
      });
    }
    set({ loading: true, error: null });
    try {
      const categories = await decisionsApi.listCategories(projectId);
      if (requestId !== latestCategoryRequest || activeProjectId !== projectId)
        return;
      set({ categories });
      if (!get().selectedCatId && categories.length > 0) {
        set({ selectedCatId: categories[0].id });
        await get().fetchDecisions(projectId, categories[0].id);
      }
    } catch (err) {
      if (
        requestId === latestCategoryRequest &&
        activeProjectId === projectId
      ) {
        set({
          error: (err as Error).message || "Failed to load decision categories",
        });
      }
    } finally {
      if (requestId === latestCategoryRequest && activeProjectId === projectId)
        set({ loading: false });
    }
  },

  selectCategory: (catId) => {
    latestDecisionRequest += 1;
    latestPreviewRequest += 1;
    set({
      selectedCatId: catId,
      selectedDecisionId: null,
      decisions: [],
      scriptPreview: "",
    });
  },

  createCategory: async (projectId, categoryId, countryTag) => {
    let cat;
    try {
      cat = await decisionsApi.createCategory(projectId, {
        category_id: categoryId,
        country_tag: countryTag,
      });
    } catch (err) {
      const msg = (err as Error).message || "";
      if (msg.includes("already exists") || msg.includes("race-collision")) {
        return null as unknown as DecisionCategory;
      }
      throw err;
    }

    set((s) => ({
      categories: [...s.categories, cat],
      selectedCatId: cat.id,
      selectedDecisionId: null,
      decisions: [],
      scriptPreview: "",
    }));
    return cat;
  },

  updateCategory: async (projectId, catId, data) => {
    const updated = await decisionsApi.updateCategory(projectId, catId, data);
    set((s) => ({
      categories: s.categories.map((c) =>
        c.id === catId ? { ...c, ...updated } : c,
      ),
    }));
  },

  deleteCategory: async (projectId, catId) => {
    await decisionsApi.deleteCategory(projectId, catId);
    set((s) => ({
      categories: s.categories.filter((c) => c.id !== catId),
      selectedCatId: s.selectedCatId === catId ? null : s.selectedCatId,
      decisions: s.selectedCatId === catId ? [] : s.decisions,
      selectedDecisionId:
        s.selectedCatId === catId ? null : s.selectedDecisionId,
    }));
  },

  fetchDecisions: async (projectId, categoryId) => {
    const catId = categoryId ?? get().selectedCatId ?? undefined;
    const selectedCategoryId = catId ?? null;
    const requestId = ++latestDecisionRequest;
    set({ loading: true, error: null });
    try {
      const decisions = await decisionsApi.list(projectId, catId);
      if (
        requestId !== latestDecisionRequest ||
        activeProjectId !== projectId ||
        get().selectedCatId !== selectedCategoryId
      )
        return;
      set({ decisions });
      if (!get().selectedDecisionId && decisions.length > 0) {
        set({ selectedDecisionId: decisions[0].id });
        get().fetchPreview(projectId, decisions[0].id);
      }
    } catch (err) {
      if (
        requestId === latestDecisionRequest &&
        activeProjectId === projectId &&
        get().selectedCatId === selectedCategoryId
      ) {
        set({ error: (err as Error).message || "Failed to load decisions" });
      }
    } finally {
      if (
        requestId === latestDecisionRequest &&
        activeProjectId === projectId &&
        get().selectedCatId === selectedCategoryId
      )
        set({ loading: false });
    }
  },

  selectDecision: (id) => {
    latestPreviewRequest += 1;
    set({ selectedDecisionId: id, scriptPreview: "" });
  },

  createDecision: async (projectId, data) => {
    let d;
    try {
      d = await decisionsApi.create(projectId, data);
    } catch (err) {
      const msg = (err as Error).message || "";
      if (msg.includes("already exists") || msg.includes("race-collision")) {
        return null as unknown as Decision;
      }
      throw err;
    }
    set((s) => ({
      decisions: [...s.decisions, d],
      selectedDecisionId: d.id,
      categories: s.categories.map((c) =>
        c.id === d.category_id
          ? { ...c, decision_count: c.decision_count + 1 }
          : c,
      ),
    }));
    get().fetchPreview(projectId, d.id);
    return d;
  },

  updateDecision: async (projectId, id, data) => {
    const updated = await decisionsApi.update(projectId, id, data);
    set((s) => ({
      decisions: s.decisions.map((d) => (d.id === id ? updated : d)),
    }));
    get().fetchPreview(projectId, id);
  },

  deleteDecision: async (projectId, id) => {
    const d = get().decisions.find((d) => d.id === id);
    await decisionsApi.delete(projectId, id);
    set((s) => ({
      decisions: s.decisions.filter((d) => d.id !== id),
      selectedDecisionId:
        s.selectedDecisionId === id ? null : s.selectedDecisionId,
    }));
    if (d) {
      set((s) => ({
        categories: s.categories.map((c) =>
          c.id === d.category_id
            ? { ...c, decision_count: Math.max(0, c.decision_count - 1) }
            : c,
        ),
      }));
    }
  },

  fetchPreview: async (projectId, decisionId) => {
    const requestId = ++latestPreviewRequest;
    try {
      const res = await decisionsApi.preview(projectId, decisionId);
      if (
        requestId === latestPreviewRequest &&
        activeProjectId === projectId &&
        get().selectedDecisionId === decisionId
      ) {
        set({ scriptPreview: res.script });
      }
    } catch {
      if (
        requestId === latestPreviewRequest &&
        activeProjectId === projectId &&
        get().selectedDecisionId === decisionId
      ) {
        set({ scriptPreview: "# Preview unavailable" });
      }
    }
  },
}));
