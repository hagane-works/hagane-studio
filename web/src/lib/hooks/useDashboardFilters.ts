"use client";

import { useCallback, useMemo, useRef } from "react";
import { useSearchParams } from "@/platform/navigation";
import { useRouter, usePathname } from "@/i18n/navigation";
import type { Project } from "@/types";

export const SORT_MODES = ["recent", "name", "countries", "created"] as const;
export type SortMode = (typeof SORT_MODES)[number];
export const DEFAULT_SORT: SortMode = "recent";

export const FILTER_MODES = ["all", "in_progress", "by_mod_base"] as const;
export type FilterMode = (typeof FILTER_MODES)[number];
export const DEFAULT_FILTER: FilterMode = "all";

export const MOD_BASE_BUCKETS = ["kaiserreich", "tno", "tfr", "other"] as const;
export type ModBaseBucket = (typeof MOD_BASE_BUCKETS)[number];

export const MOD_BASE_LABEL_KEYS: Record<ModBaseBucket, string> = {
  kaiserreich: "modBase.kaiserreich",
  tno: "modBase.tno",
  tfr: "modBase.tfr",
  other: "modBase.other",
};

export const RECENT_EDIT_WINDOW_MS = 24 * 60 * 60 * 1000;

const MOD_BASE_PATTERNS: Array<[ModBaseBucket, RegExp]> = [
  ["kaiserreich", /\b(kaiserreich|kr)\b|kaiserreich|凯撒|凯瑟|铁血|铁拳/i],
  ["tno", /\b(tno|the new order|last days)\b|新秩序|黑暗时代/i],
  ["tfr", /\b(tfr|the fire rises|fire ?rises)\b|烈火|fire ?rises/i],
];

export function inferModBase(project: Project): ModBaseBucket {
  const explicit = (project as unknown as { mod_base?: string }).mod_base;
  if (explicit && (MOD_BASE_BUCKETS as readonly string[]).includes(explicit)) {
    return explicit as ModBaseBucket;
  }

  const tagsLc = (project.tags ?? []).map((t) => t.toLowerCase());
  for (const [bucket, re] of MOD_BASE_PATTERNS) {
    if (tagsLc.some((t) => re.test(t))) return bucket;
  }

  const name = project.name ?? "";
  for (const [bucket, re] of MOD_BASE_PATTERNS) {
    if (re.test(name)) return bucket;
  }

  return "other";
}

function isSortMode(v: string | null): v is SortMode {
  return v != null && (SORT_MODES as readonly string[]).includes(v);
}
function isFilterMode(v: string | null): v is FilterMode {
  return v != null && (FILTER_MODES as readonly string[]).includes(v);
}

export interface DashboardFilters {
  q: string;
  sort: SortMode;
  filter: FilterMode;

  setQ: (next: string) => void;
  setSort: (next: SortMode) => void;
  setFilter: (next: FilterMode) => void;

  visibleProjects: Project[];

  grouped: Array<{ bucket: ModBaseBucket | "all"; projects: Project[] }>;

  totalVisible: number;

  isFiltered: boolean;
}

export function useDashboardFilters(projects: Project[]): DashboardFilters {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const rawQ = searchParams?.get("q") ?? "";
  const rawSort = searchParams?.get("sort");
  const rawFilter = searchParams?.get("filter");

  const q = rawQ;
  const sort: SortMode = isSortMode(rawSort) ? rawSort : DEFAULT_SORT;
  const filter: FilterMode = isFilterMode(rawFilter)
    ? rawFilter
    : DEFAULT_FILTER;

  const pathRef = useRef(pathname);
  pathRef.current = pathname;

  const updateParam = useCallback(
    (key: "q" | "sort" | "filter", value: string | null) => {
      const params = new URLSearchParams(searchParams?.toString() ?? "");
      if (value == null || value === "" || value === defaultFor(key)) {
        params.delete(key);
      } else {
        params.set(key, value);
      }
      const qs = params.toString();
      const target = `${pathRef.current ?? "/"}${qs ? `?${qs}` : ""}`;

      router.replace(target, { scroll: false });
    },
    [router, searchParams],
  );

  const setQ = useCallback(
    (next: string) => updateParam("q", next),
    [updateParam],
  );
  const setSort = useCallback(
    (next: SortMode) => updateParam("sort", next),
    [updateParam],
  );
  const setFilter = useCallback(
    (next: FilterMode) => updateParam("filter", next),
    [updateParam],
  );

  const visibleProjects = useMemo(() => {
    const needle = q.trim().toLowerCase();

    const now = Date.now();

    let list = projects;

    if (needle) {
      list = list.filter((p) => {
        if (p.name?.toLowerCase().includes(needle)) return true;
        if (p.country_tag?.toLowerCase().includes(needle)) return true;
        if (p.country_tags?.some((t) => t.toLowerCase().includes(needle)))
          return true;

        if (inferModBase(p).includes(needle)) return true;
        return false;
      });
    }

    if (filter === "in_progress") {
      list = list.filter((p) => {
        const updated = Date.parse(p.updated_at);
        if (Number.isNaN(updated)) return false;
        return now - updated <= RECENT_EDIT_WINDOW_MS;
      });
    }

    const sorted = [...list];
    switch (sort) {
      case "name":
        sorted.sort((a, b) =>
          (a.name ?? "").localeCompare(b.name ?? "", undefined, {
            sensitivity: "base",
          }),
        );
        break;
      case "countries":
        sorted.sort(
          (a, b) =>
            (b.country_tags?.length ?? 0) - (a.country_tags?.length ?? 0),
        );
        break;
      case "created":
        sorted.sort(
          (a, b) => Date.parse(b.created_at) - Date.parse(a.created_at),
        );
        break;
      case "recent":
      default:
        sorted.sort(
          (a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at),
        );
        break;
    }
    return sorted;
  }, [projects, q, sort, filter]);

  const grouped = useMemo(() => {
    if (filter !== "by_mod_base") {
      return [{ bucket: "all" as const, projects: visibleProjects }];
    }
    const byBucket = new Map<ModBaseBucket, Project[]>();
    for (const p of visibleProjects) {
      const b = inferModBase(p);
      const arr = byBucket.get(b);
      if (arr) arr.push(p);
      else byBucket.set(b, [p]);
    }
    return MOD_BASE_BUCKETS.map((bucket) => ({
      bucket,
      projects: byBucket.get(bucket) ?? [],
    })).filter((g) => g.projects.length > 0);
  }, [visibleProjects, filter]);

  const isFiltered =
    q.trim() !== "" || sort !== DEFAULT_SORT || filter !== DEFAULT_FILTER;

  return {
    q,
    sort,
    filter,
    setQ,
    setSort,
    setFilter,
    visibleProjects,
    grouped,
    totalVisible: visibleProjects.length,
    isFiltered,
  };
}

function defaultFor(key: "q" | "sort" | "filter"): string {
  if (key === "sort") return DEFAULT_SORT;
  if (key === "filter") return DEFAULT_FILTER;
  return "";
}
