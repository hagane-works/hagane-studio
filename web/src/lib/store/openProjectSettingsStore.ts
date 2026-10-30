import { create } from "zustand";

export const SECTION_OWNER: Record<string, "project" | "country" | "politics"> =
  {
    ideology: "politics",
  };

export function dialogForSection(
  section: string | null,
): "project" | "country" | "politics" {
  return (section && SECTION_OWNER[section]) || "project";
}
export interface OpenProjectSettingsStore {
  token: number;
  section: string | null;
  requestOpen: (section?: string) => void;
}

export const useOpenProjectSettings = create<OpenProjectSettingsStore>(
  (set) => ({
    token: 0,
    section: null,
    requestOpen: (section) =>
      set((s) => ({ token: s.token + 1, section: section ?? null })),
  }),
);
