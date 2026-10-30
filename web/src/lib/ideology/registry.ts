import type { ContentLocale } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { vanillaIdeologies } from "@/lib/referenceTables";

export type IdeologyLocalisationMap = Partial<Record<ContentLocale, string>>;

export interface IdeologySubEntry {
  id: string;

  group: string;

  name?: IdeologyLocalisationMap;

  desc?: IdeologyLocalisationMap;

  random_disabled?: boolean;
}

export interface IdeologyRootDefinition {
  id: string;

  body: string;
  name?: IdeologyLocalisationMap;
  desc?: IdeologyLocalisationMap;
}

export interface IdeologyRegistry {
  groups: string[];

  root_definitions?: IdeologyRootDefinition[];
  sub_ideologies: IdeologySubEntry[];

  preset?: string;
}

export function normalizeRegistry(raw: unknown): IdeologyRegistry {
  const out: IdeologyRegistry = {
    groups: [],
    root_definitions: [],
    sub_ideologies: [],
  };
  if (!raw || typeof raw !== "object") return out;
  const r = raw as Record<string, unknown>;
  if (Array.isArray(r.groups))
    out.groups = r.groups.filter((g): g is string => typeof g === "string");
  if (Array.isArray(r.root_definitions)) {
    for (const candidate of r.root_definitions) {
      if (!candidate || typeof candidate !== "object") continue;
      const root = candidate as Partial<IdeologyRootDefinition>;
      if (typeof root.id !== "string" || !root.id.trim()) continue;
      const id = root.id.trim();
      out.root_definitions!.push({
        id,
        body: typeof root.body === "string" ? root.body.trim() : "",
        name:
          root.name && typeof root.name === "object" ? root.name : undefined,
        desc:
          root.desc && typeof root.desc === "object" ? root.desc : undefined,
      });
      if (!out.groups.includes(id)) out.groups.push(id);
    }
  }
  if (Array.isArray(r.sub_ideologies)) {
    for (const s of r.sub_ideologies) {
      if (typeof s === "string" && s) {
        out.sub_ideologies.push({ id: s, group: "" });
      } else if (
        s &&
        typeof s === "object" &&
        typeof (s as IdeologySubEntry).id === "string" &&
        (s as IdeologySubEntry).id
      ) {
        const e = s as IdeologySubEntry;
        const normalized: IdeologySubEntry = {
          id: e.id,
          group: e.group ?? "",
          name: e.name,
          random_disabled: e.random_disabled,
        };
        if (e.desc) normalized.desc = e.desc;
        out.sub_ideologies.push(normalized);
      }
    }
  }
  if (typeof r.preset === "string") out.preset = r.preset;
  return out;
}

export function mergeWithVanilla(raw: unknown): IdeologyRegistry {
  const project = normalizeRegistry(raw);
  const vanilla = vanillaIdeologies();
  const groups = [...vanilla.groups];
  for (const g of project.groups) if (!groups.includes(g)) groups.push(g);

  const byId = new Map<string, IdeologySubEntry>();
  const order: string[] = [];
  for (const s of vanilla.sub_ideologies) {
    byId.set(s.id, { ...s });
    order.push(s.id);
  }
  for (const s of project.sub_ideologies) {
    if (!byId.has(s.id)) order.push(s.id);
    byId.set(s.id, s);
  }
  const merged: IdeologyRegistry = {
    groups,
    root_definitions: project.root_definitions?.map((root) => ({ ...root })),
    sub_ideologies: order.map((id) => byId.get(id)!),
  };
  if (project.preset) merged.preset = project.preset;
  return merged;
}

export function ideologyOptionLabel(
  entry: IdeologySubEntry,
  locale: ContentLocale,
): string {
  const gloss = resolveDisplayName(
    locale,
    entry.name,
    entry.name?.en,
    entry.name?.zh,
    "",
  );
  return gloss && gloss.toLowerCase() !== entry.id.toLowerCase()
    ? `${entry.id}(${gloss})`
    : entry.id;
}

export interface IdeologyOption {
  value: string;
  label: string;
  group: string;
}

export function subIdeologyOptions(
  raw: unknown,
  locale: ContentLocale,
): IdeologyOption[] {
  return mergeWithVanilla(raw).sub_ideologies.map((s) => ({
    value: s.id,
    label: ideologyOptionLabel(s, locale),
    group: s.group,
  }));
}

const GROUP_GLOSS: Record<string, IdeologyLocalisationMap> = {
  democratic: {
    en: "Democratic",
    zh: "民主主义",
    ru: "Демократия",
    de: "Demokratisch",
    es: "Democrático",
    fr: "démocratique",
    ja: "民主主義",
    ko: "민주주의",
    pl: "Demokratyczna",
    pt: "Democrático",
  },
  communism: {
    en: "Communist",
    zh: "共产主义",
    ru: "Коммунизм",
    de: "Kommunistisch",
    es: "Comunista",
    fr: "communiste",
    ja: "共産主義",
    ko: "공산주의",
    pl: "Kom.",
    pt: "Comunista",
  },
  fascism: {
    en: "Fascist",
    zh: "法西斯主义",
    ru: "Фашизм",
    de: "Faschistisch",
    es: "Fascista",
    fr: "fasciste",
    ja: "ファシスト",
    ko: "파시즘",
    pl: "Fasz.",
    pt: "Fascista",
  },
  neutrality: {
    en: "Non-Aligned",
    zh: "中立主义",
    ru: "Нейтралитет",
    de: "Neutral",
    es: "Neutral",
    fr: "non aligné",
    ja: "非同盟/中道主義",
    ko: "비동맹주의",
    pl: "Bez powiązań",
    pt: "Não alinhado",
  },
};

export function groupIdeologyDisplayName(
  id: string,
  locale: ContentLocale,
  authored?: IdeologyLocalisationMap,
): string {
  const source = GROUP_GLOSS[id] ?? authored;
  return resolveDisplayName(locale, source, source?.en, source?.zh, id);
}

export function groupIdeologyOptions(
  raw: unknown,
  locale: ContentLocale,
): IdeologyOption[] {
  const merged = mergeWithVanilla(raw);
  const customRootNames = new Map(
    (merged.root_definitions ?? []).map((root) => [root.id, root.name]),
  );
  return merged.groups.map((g) => {
    const gl = GROUP_GLOSS[g];
    const custom = customRootNames.get(g);
    const gloss = groupIdeologyDisplayName(g, locale, gl ?? custom);
    return {
      value: g,
      label: gloss || g,
      group: g,
    };
  });
}

export function countryFlagIdeologyGroups(raw: unknown): string[] {
  return mergeWithVanilla(raw).groups;
}
