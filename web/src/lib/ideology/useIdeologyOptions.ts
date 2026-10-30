import { useMemo } from "react";
import useSWR from "swr";
import { useProjectStore } from "@/lib/store/projectStore";
import { useLocale } from "use-intl";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";
import { basemodIndexApi } from "@/lib/api";
import type { BaseModIdeologyRegistry } from "@/types";
import {
  subIdeologyOptions,
  groupIdeologyOptions,
  ideologyOptionLabel,
  mergeWithVanilla,
  normalizeRegistry,
  type IdeologyOption,
  type IdeologyRegistry,
} from "./registry";

export const CUSTOM_IDEOLOGY_SENTINEL = "__custom__";

const EMPTY_BASE_REGISTRY: BaseModIdeologyRegistry = {
  groups: [],
  sub_ideologies: [],
};

const EMPTY_DEPENDENCIES: string[] = [];

export interface IdeologyOptionsResult {
  subOptions: IdeologyOption[];

  groupOptions: IdeologyOption[];

  merged: IdeologyRegistry;

  knownSubIds: Set<string>;

  knownSubIdsSettled: boolean;

  baseDependency: string;
}

export function useIdeologyOptions(): IdeologyOptionsResult {
  const registry = useProjectStore((s) => s.currentProject?.ideology_registry);
  const projectId = useProjectStore((s) => s.currentProject?.id);
  const dependencies = useProjectStore(
    (s) => s.currentProject?.dependencies ?? EMPTY_DEPENDENCIES,
  );

  const locale = useLocale() as ContentLocale;
  const hasBaseDependency = dependencies.some(
    (dependency) =>
      typeof dependency === "string" && dependency.trim().length > 0,
  );

  const {
    data: base = EMPTY_BASE_REGISTRY,
    isLoading: baseLoading,
    error: baseError,
  } = useSWR<BaseModIdeologyRegistry>(
    projectId && hasBaseDependency
      ? ["basemod-ideology-registry", projectId]
      : null,
    () => basemodIndexApi.ideologyRegistry(projectId as number),
    { keepPreviousData: true, revalidateOnFocus: false },
  );

  return useMemo(() => {
    const merged = mergeWithVanilla(registry);
    const projectVanillaSubs = subIdeologyOptions(registry, locale);
    const projectVanillaGroups = groupIdeologyOptions(registry, locale);

    const knownGroupIds = new Set(projectVanillaGroups.map((o) => o.value));
    const baseGroups: IdeologyOption[] = [];
    for (const g of base.groups) {
      if (!g.value || knownGroupIds.has(g.value)) continue;
      knownGroupIds.add(g.value);

      baseGroups.push({
        value: g.value,
        label: g.label || g.value,
        group: g.value,
      });
    }

    const knownSub = new Set(projectVanillaSubs.map((o) => o.value));
    const baseSubs: IdeologyOption[] = [];
    for (const s of base.sub_ideologies) {
      if (!s.id || knownSub.has(s.id)) continue;
      knownSub.add(s.id);

      baseSubs.push({
        value: s.id,
        label: ideologyOptionLabel(
          { id: s.id, group: s.group, name: s.name },
          locale,
        ),
        group: s.group,
      });
    }

    const project = normalizeRegistry(registry);
    const projectOwnSubIds = new Set(project.sub_ideologies.map((s) => s.id));
    const projectOwnGroupIds = new Set(project.groups);
    const subOptions = baseSubs.length
      ? [
          ...projectVanillaSubs.filter((o) => projectOwnSubIds.has(o.value)),
          ...baseSubs,
          ...projectVanillaSubs.filter((o) => !projectOwnSubIds.has(o.value)),
        ]
      : projectVanillaSubs;
    const groupOptions = baseGroups.length
      ? [
          ...projectVanillaGroups.filter((o) =>
            projectOwnGroupIds.has(o.value),
          ),
          ...baseGroups,
          ...projectVanillaGroups.filter(
            (o) => !projectOwnGroupIds.has(o.value),
          ),
        ]
      : projectVanillaGroups;

    const firstDependency =
      dependencies.find((d) => typeof d === "string" && d.trim())?.trim() ?? "";

    return {
      subOptions,
      groupOptions,
      merged,
      knownSubIds: knownSub,

      knownSubIdsSettled: !baseLoading && !baseError,
      baseDependency: firstDependency,
    };
  }, [registry, base, locale, dependencies, baseLoading, baseError]);
}
