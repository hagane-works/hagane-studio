import type { WalkFile } from "@/lib/fs-scanner";
import type { LucideIcon } from "lucide-react";

export interface ScanCategory {
  files: WalkFile[];
  fileCount: number;
  enabled: boolean;

  skippedLocal?: number;

  declaredTagCount?: number;

  dirCounts?: Record<string, number>;
}

export interface ScanResults {
  characters: ScanCategory;
  focus_trees: ScanCategory;
  technologies: ScanCategory;
  ideas: ScanCategory;
  events: ScanCategory;
  decisions: ScanCategory;

  bookmarks: ScanCategory;

  history_files: ScanCategory;

  states: ScanCategory;

  country_tags: ScanCategory;
  gfx: ScanCategory;

  gui: ScanCategory;

  loc: ScanCategory;

  raids: ScanCategory;

  units: ScanCategory;

  common_rules: ScanCategory;

  map: ScanCategory;

  flags: ScanCategory;

  descriptor: { file: WalkFile["file"]; path: string } | null;
}

export type ScanCategoryKey = {
  [K in keyof ScanResults]: ScanResults[K] extends ScanCategory ? K : never;
}[keyof ScanResults];

export type Phase =
  "idle" | "scanning" | "scanned" | "parsing" | "uploading" | "done";

export type WarnTranslator = {
  (key: string, values?: Record<string, string | number>): string;
  has(key: string): boolean;
};

export type ScanStatusTranslator = (
  key: string,
  values?: Record<string, string | number>,
) => string;

export interface UnifiedModImportProps {
  projectId: number;
  countryTag: string;
  onComplete: () => void | Promise<void>;

  collapsed?: boolean;

  isLoadExistingMod?: boolean;

  onBackgroundToggle?: (value: boolean) => void;
}

export type CategoryKey =
  | "characters"
  | "focus_trees"
  | "technologies"
  | "ideas"
  | "events"
  | "decisions"
  | "bookmarks"
  | "history_files"
  | "states";
export type CategoryItem = {
  key: CategoryKey;
  label: string;
  icon: LucideIcon;
  desc: (s: ScanResults) => string;
};
export type CategoryGroup = {
  id: "characters" | "content" | "scenario";
  label: string;
  items: CategoryItem[];
};

export type SilentSeedResult = {
  phase: "idle" | "auditing" | "creating" | "done" | "error";
  declared: number;
  created: number;
  skippedDynamic: number;
  skippedExisting: number;

  presetName?: string | null;
  errorMsg?: string;

  samples?: { tag: string; name: string; color: number[] | null }[];
};
