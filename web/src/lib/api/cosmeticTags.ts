import { request } from "./_client";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";

export const COSMETIC_SUFFIXES = [
  "",
  "ADJ",
  "DEF",
  "democratic",
  "democratic_ADJ",
  "democratic_DEF",
  "fascism",
  "fascism_ADJ",
  "fascism_DEF",
  "neutrality",
  "neutrality_ADJ",
  "neutrality_DEF",
  "communism",
  "communism_ADJ",
  "communism_DEF",
] as const;
export type CosmeticSuffix = (typeof COSMETIC_SUFFIXES)[number];
export type CosmeticNames = Record<
  string,
  Partial<Record<CosmeticSuffix, string>>
>;
export type CosmeticRGB = [number, number, number];

export const COSMETIC_LOCALES: Record<ContentLocale, string> = {
  en: "l_english",
  zh: "l_simp_chinese",
  ru: "l_russian",
  de: "l_german",
  es: "l_spanish",
  fr: "l_french",
  ja: "l_japanese",
  ko: "l_korean",
  pl: "l_polish",
  pt: "l_braz_por",
};

export interface CosmeticRawOverride {
  file_id: number;
  path: string;
  locale: string;
  suffix: CosmeticSuffix;
  value: string;
}

export interface CosmeticTagView {
  id: number;
  project_id: number;
  tag: string;
  source: "created" | "imported";
  revision: number;
  name_locs: CosmeticNames;
  imported_name_locs: CosmeticNames;
  color: CosmeticRGB | null;
  color_ui: CosmeticRGB | null;
  imported_color: CosmeticRGB | null;
  imported_color_ui: CosmeticRGB | null;
  referenced_by: string[];
  raw_overrides?: CosmeticRawOverride[];
}

export interface CosmeticNameOperation {
  locale: string;
  suffix: CosmeticSuffix;
  value: string | null;
}

export interface CosmeticTagPatch {
  expected_revision: number;
  name_ops: CosmeticNameOperation[];
  color?: CosmeticRGB | null;
  color_ui?: CosmeticRGB | null;
}

export interface CosmeticDraft {
  name_locs: CosmeticNames;
  color: CosmeticRGB | null;
  color_ui: CosmeticRGB | null;
}

export interface CosmeticResolvedName {
  value: string;
  source: "authored" | "imported" | "generated";
  inherited_from: string;
}

export interface CosmeticResolution {
  names: Record<CosmeticSuffix, CosmeticResolvedName>;
  color: CosmeticRGB;
  color_ui: CosmeticRGB;
}

const base = (projectId: number) => `/go/projects/${projectId}/cosmetic-tags`;
const item = (projectId: number, tag: string) =>
  `/go/projects/${projectId}/cosmetic-tags/${encodeURIComponent(tag)}`;

export const cosmeticTagsApi = {
  list: (projectId: number, signal?: AbortSignal) =>
    request<{ cosmetic_tags: CosmeticTagView[]; total: number }>(
      base(projectId),
      { signal },
    ),
  get: (projectId: number, tag: string, signal?: AbortSignal) =>
    request<CosmeticTagView>(item(projectId, tag), { signal }),
  create: (projectId: number, tag: string) =>
    request<CosmeticTagView>(base(projectId), {
      method: "POST",
      body: JSON.stringify({ tag }),
    }),
  update: (projectId: number, tag: string, patch: CosmeticTagPatch) =>
    request<CosmeticTagView>(item(projectId, tag), {
      method: "PATCH",
      body: JSON.stringify(patch),
    }),
  resolve: (
    projectId: number,
    tag: string,
    locale: string,
    draft: CosmeticDraft,
    signal?: AbortSignal,
  ) =>
    request<CosmeticResolution>(
      `/go/projects/${projectId}/cosmetic-tags/${encodeURIComponent(tag)}/resolve`,
      {
        method: "POST",
        body: JSON.stringify({ locale, ...draft }),
        signal,
      },
    ),
};
