import { useMemo } from "react";
import { create } from "zustand";
import type { Project } from "@/types";
import { projectsApi } from "@/lib/api";
import {
  type ContentLocale,
  normalizeContentLocales,
  useContentLocaleStore,
} from "@/lib/store/contentLocaleStore";

export type WorkbenchViewMode = "single" | "all";

function viewModeStorageKey(
  projectId: number | null | undefined,
): string | null {
  if (!projectId) return null;
  return `mtag-viewMode-${projectId}`;
}

function activeCountryStorageKey(
  projectId: number | null | undefined,
): string | null {
  if (!projectId) return null;
  return `hoi4:activeCountry:${projectId}`;
}

const contentLanguageMutationQueues = new Map<number, Promise<unknown>>();

function enqueueContentLanguageMutation<T>(
  projectId: number,
  mutation: () => Promise<T>,
): Promise<T> {
  const previous =
    contentLanguageMutationQueues.get(projectId) ?? Promise.resolve();
  const current = previous.catch(() => undefined).then(mutation);
  contentLanguageMutationQueues.set(projectId, current);
  void current
    .finally(() => {
      if (contentLanguageMutationQueues.get(projectId) === current) {
        contentLanguageMutationQueues.delete(projectId);
      }
    })
    .catch(() => undefined);
  return current;
}

interface ProjectStore {
  projects: Project[];
  currentProject: Project | null;
  loading: boolean;
  error: string | null;

  activeCountryTag: string | null;

  viewMode: WorkbenchViewMode;

  fetchProjects: () => Promise<void>;
  fetchProject: (id: number) => Promise<void>;

  createProject: (
    data: import("@/lib/api/project").ProjectCreateInput,
  ) => Promise<Project>;
  deleteProject: (id: number) => Promise<void>;
  setContentLanguageEnabled: (
    locale: ContentLocale,
    enabled: boolean,
  ) => Promise<Project>;
  setActiveCountryTag: (tag: string | null) => void;
  setViewMode: (mode: WorkbenchViewMode) => void;
  toggleViewMode: () => void;
}

export const useProjectStore = create<ProjectStore>((set, get) => ({
  projects: [],
  currentProject: null,
  loading: false,
  error: null,
  activeCountryTag: null,
  viewMode: "single",

  fetchProjects: async () => {
    set({ loading: true, error: null });
    try {
      const res = await projectsApi.list();
      set({ projects: res.projects, loading: false });
    } catch (e) {
      set({ error: (e as Error).message, loading: false });
    }
  },

  fetchProject: async (id: number) => {
    set({ loading: true, error: null });
    try {
      const project = await projectsApi.get(id);

      let restoredViewMode: WorkbenchViewMode = "single";

      let restoredActiveCountryTag: string | null = null;
      if (typeof window !== "undefined") {
        const vmKey = viewModeStorageKey(project.id);
        if (vmKey) {
          const raw = window.localStorage.getItem(vmKey);
          if (raw === "all" || raw === "single") restoredViewMode = raw;
        }
        const acKey = activeCountryStorageKey(project.id);
        if (acKey) {
          const raw = window.localStorage.getItem(acKey);

          if (raw && (project.country_tags ?? []).includes(raw)) {
            restoredActiveCountryTag = raw;
          }
        }
      }
      let switchedProject = false;
      set((state) => {
        switchedProject = state.currentProject?.id !== project.id;
        return {
          currentProject: project,
          loading: false,

          activeCountryTag:
            state.currentProject?.id === project.id &&
            state.activeCountryTag &&
            (project.country_tags ?? []).includes(state.activeCountryTag)
              ? state.activeCountryTag
              : (restoredActiveCountryTag ?? project.country_tag),
          viewMode:
            state.currentProject?.id === project.id
              ? state.viewMode
              : restoredViewMode,
        };
      });

      if (switchedProject) useContentLocaleStore.getState().setLocale("en");
    } catch (e) {
      set({ error: (e as Error).message, loading: false });
    }
  },

  createProject: async (data) => {
    const project = await projectsApi.create(data);
    set((state) => ({ projects: [project, ...state.projects] }));
    return project;
  },

  deleteProject: async (id: number) => {
    await projectsApi.delete(id);
    set((state) => ({
      projects: state.projects.filter((p) => p.id !== id),
    }));
  },

  setContentLanguageEnabled: async (locale, enabled) => {
    const requestedProject = get().currentProject;
    if (!requestedProject) throw new Error("No project is currently open.");
    if (locale === "en" && !enabled) {
      throw new Error("English content language cannot be disabled.");
    }
    const projectId = requestedProject.id;

    return enqueueContentLanguageMutation(projectId, async () => {
      const stateBeforeRequest = get();
      const project =
        (stateBeforeRequest.currentProject?.id === projectId
          ? stateBeforeRequest.currentProject
          : stateBeforeRequest.projects.find(
              (candidate) => candidate.id === projectId,
            )) ?? requestedProject;
      const currentLanguages = normalizeContentLocales(
        project.export_languages,
      );
      const alreadyEnabled = currentLanguages.includes(locale);
      if (alreadyEnabled === enabled) {
        if (get().currentProject?.id === projectId) {
          if (enabled) useContentLocaleStore.getState().setLocale(locale);
          if (!enabled && useContentLocaleStore.getState().locale === locale) {
            useContentLocaleStore.getState().setLocale("en");
          }
        }
        return project;
      }

      const nextLanguages = normalizeContentLocales(
        enabled
          ? [...currentLanguages, locale]
          : currentLanguages.filter((candidate) => candidate !== locale),
      );
      const response = await projectsApi.update(projectId, {
        export_languages: nextLanguages,
      });
      const responseLanguages = normalizeContentLocales(
        response.export_languages,
      );
      let updatedProject: Project = {
        ...project,
        export_languages: responseLanguages,
      };

      set((state) => {
        const latest =
          (state.currentProject?.id === projectId
            ? state.currentProject
            : state.projects.find((candidate) => candidate.id === projectId)) ??
          project;
        updatedProject = {
          ...latest,
          export_languages: responseLanguages,
        };
        return {
          currentProject:
            state.currentProject?.id === projectId
              ? updatedProject
              : state.currentProject,
          projects: state.projects.map((candidate) =>
            candidate.id === projectId ? updatedProject : candidate,
          ),
        };
      });

      if (get().currentProject?.id === projectId) {
        if (enabled) {
          useContentLocaleStore.getState().setLocale(locale);
        } else if (useContentLocaleStore.getState().locale === locale) {
          useContentLocaleStore.getState().setLocale("en");
        }
      }
      return updatedProject;
    });
  },

  setActiveCountryTag: (tag: string | null) => {
    set({ activeCountryTag: tag });
    if (typeof window !== "undefined") {
      const pid = get().currentProject?.id;
      const key = activeCountryStorageKey(pid);
      if (key) {
        if (tag !== null) {
          window.localStorage.setItem(key, tag);
        } else {
          window.localStorage.removeItem(key);
        }
      }
    }
  },

  setViewMode: (mode: WorkbenchViewMode) => {
    set({ viewMode: mode });
    if (typeof window !== "undefined") {
      const pid = get().currentProject?.id;
      const key = viewModeStorageKey(pid);
      if (key) window.localStorage.setItem(key, mode);
    }
  },

  toggleViewMode: () => {
    const next: WorkbenchViewMode =
      get().viewMode === "single" ? "all" : "single";
    get().setViewMode(next);
  },
}));

export function useActiveCountry(): string {
  const activeCountryTag = useProjectStore((s) => s.activeCountryTag);
  const currentProject = useProjectStore((s) => s.currentProject);
  return activeCountryTag ?? currentProject?.country_tag ?? "";
}

export function useEffectiveCountryFilter(): string | null {
  const activeCountryTag = useProjectStore((s) => s.activeCountryTag);
  const viewMode = useProjectStore((s) => s.viewMode);
  const currentProject = useProjectStore((s) => s.currentProject);
  if (viewMode === "all") return null;
  return activeCountryTag ?? currentProject?.country_tag ?? null;
}

const EMPTY_COUNTRIES: string[] = [];

export function useProjectCountries(): string[] {
  const currentProject = useProjectStore((s) => s.currentProject);
  const countryTags = currentProject?.country_tags;
  const countryTag = currentProject?.country_tag;
  return useMemo(() => {
    if (!currentProject) return EMPTY_COUNTRIES;
    const tags = countryTags?.length ? countryTags : [countryTag];

    return Array.from(new Set(tags.filter(Boolean))) as string[];
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [countryTags, countryTag]);
}
