import { requestGameIndexed } from "@/lib/api/gameIndex";

export type IdeaGroup =
  | "country_spirits"
  | "hidden_spirits"
  | "design_companies"
  | "old_advisors"
  | "laws"
  | "officer_corps_spirits";

type ByGroupEntry = [string, string, string];
type ByCountryEntry = [string, string, string];

interface VanillaIdeasData {
  meta: {
    total: number;
    source: string;
    groups: Record<string, number>;
  };
  by_group: Record<string, ByGroupEntry[]>;
  by_country: Record<string, ByCountryEntry[]>;
  all_ids: string[];
}

let _data: VanillaIdeasData | null = null;
let _allIdsSet: Set<string> | null = null;
let _pending: Promise<void> | null = null;

export async function loadVanillaIdeas(): Promise<void> {
  if (_data) return;
  if (_pending) return _pending;
  _pending = (async () => {
    _data = await requestGameIndexed<VanillaIdeasData>(
      "/go/vanilla/ideas/index",
    );
    _allIdsSet = new Set(_data.all_ids);
  })().finally(() => {
    _pending = null;
  });
  return _pending;
}

export function isVanillaIdeasLoaded(): boolean {
  return _data !== null;
}

export const IDEA_GROUP_LABELS: Record<IdeaGroup, string> = {
  country_spirits: "Country Spirits / 国家精神",
  hidden_spirits: "Hidden Spirits / 隐藏精神",
  design_companies: "Design Companies / 设计公司",
  old_advisors: "Advisors (old format) / 旧版顾问格式",
  laws: "Laws / 法律",
  officer_corps_spirits: "Officer Corps / 军官团精神",
};

export const IDEA_GROUP_ORDER: IdeaGroup[] = [
  "country_spirits",
  "hidden_spirits",
  "design_companies",
  "laws",
  "officer_corps_spirits",
  "old_advisors",
];

export interface VanillaIdeaRef {
  id: string;
  ideaType: string;
  group: IdeaGroup;
  country: string;

  label?: string;
}

export function isVanillaIdea(id: string): boolean {
  return _allIdsSet?.has(id) ?? false;
}

export function getVanillaIdeaCount(): number {
  return _data?.meta.total ?? 0;
}

export function getVanillaIdeasForCountry(tag: string): VanillaIdeaRef[] {
  const entries = _data?.by_country[tag];
  if (!entries) return [];
  return entries.map(([id, ideaType, group]) => ({
    id,
    ideaType,
    group: group as IdeaGroup,
    country: tag,
    label: group === "old_advisors" ? `${id} (旧版顾问格式)` : undefined,
  }));
}

export function getVanillaIdeasByGroup(group: IdeaGroup): VanillaIdeaRef[] {
  const entries = _data?.by_group[group];
  if (!entries) return [];
  return entries.map(([id, ideaType, country]) => ({
    id,
    ideaType,
    group,
    country,
    label: group === "old_advisors" ? `${id} (旧版顾问格式)` : undefined,
  }));
}

export function getVanillaIdeaCountries(): string[] {
  if (!_data) return [];
  return Object.keys(_data.by_country).sort();
}

export function searchVanillaIdeas(
  query: string,
  options?: {
    group?: IdeaGroup;
    country?: string;
  },
  limit = 50,
): VanillaIdeaRef[] {
  const q = query.toLowerCase().trim();
  if (!q || !_data) return [];

  let source: VanillaIdeaRef[];

  if (options?.country && options?.group) {
    source = getVanillaIdeasForCountry(options.country).filter(
      (i) => i.group === options.group,
    );
  } else if (options?.country) {
    source = getVanillaIdeasForCountry(options.country);
  } else if (options?.group) {
    source = getVanillaIdeasByGroup(options.group);
  } else {
    source = _data.all_ids.map((id) => {
      for (const [group, entries] of Object.entries(_data!.by_group)) {
        const found = entries.find(([eid]) => eid === id);
        if (found) {
          return {
            id,
            ideaType: found[1],
            group: group as IdeaGroup,
            country: found[2],
          };
        }
      }
      return {
        id,
        ideaType: "unknown",
        group: "country_spirits" as IdeaGroup,
        country: "unknown",
      };
    });
  }

  const scored: { ref: VanillaIdeaRef; score: number }[] = [];
  for (const ref of source) {
    const lower = ref.id.toLowerCase();
    if (lower === q) {
      scored.push({ ref, score: 0 });
    } else if (lower.startsWith(q)) {
      scored.push({ ref, score: 1 });
    } else if (lower.includes(q)) {
      scored.push({ ref, score: 2 });
    }
  }

  scored.sort((a, b) => a.score - b.score || a.ref.id.localeCompare(b.ref.id));
  return scored.slice(0, limit).map((s) => s.ref);
}

export function buildIdeaReferenceList(
  projectIdeas: { id: string; ideaType: string }[],
  projectTag: string,
  activeTab:
    | "project"
    | "country"
    | "shared"
    | "laws"
    | "design"
    | "old_advisors"
    | "other"
    | null,
): VanillaIdeaRef[] {
  const results: VanillaIdeaRef[] = [];

  if (!activeTab || activeTab === "project") {
    for (const pi of projectIdeas) {
      results.push({
        id: pi.id,
        ideaType: pi.ideaType,
        group: "country_spirits",
        country: "project",
      });
    }
    if (activeTab === "project") return results;
  }

  if (!activeTab || activeTab === "country") {
    results.push(...getVanillaIdeasForCountry(projectTag));
    if (activeTab === "country") return results;
  }

  if (!activeTab || activeTab === "shared") {
    const shared = getVanillaIdeasForCountry("shared").filter(
      (i) => i.group === "country_spirits" || i.group === "hidden_spirits",
    );
    results.push(...shared);
    if (activeTab === "shared") return results;
  }

  if (!activeTab || activeTab === "laws") {
    results.push(...getVanillaIdeasByGroup("laws"));
    if (activeTab === "laws") return results;
  }

  if (!activeTab || activeTab === "design") {
    results.push(...getVanillaIdeasByGroup("design_companies"));
    if (activeTab === "design") return results;
  }

  if (!activeTab || activeTab === "old_advisors") {
    results.push(...getVanillaIdeasByGroup("old_advisors"));
    if (activeTab === "old_advisors") return results;
  }

  if (activeTab === "other") {
    for (const country of getVanillaIdeaCountries()) {
      if (country !== projectTag && country !== "shared") {
        results.push(...getVanillaIdeasForCountry(country));
      }
    }
  }

  return results;
}
