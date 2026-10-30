export type WikiType = "effect" | "trigger" | "modifier";

export interface WikiIndexEntry {
  type: WikiType;
  slug: string;
  name: string;
  scope: string[];
  category: string[];
  summary: string;
  summaryEn?: string;
  summaryZh?: string;

  usage?: number;

  hasHandsOn: boolean;
}

export function isNameTemplate(name: string): boolean {
  return name.includes("<");
}

export function orderGroupEntries<
  T extends { name: string; slug: string; usage?: number },
>(list: T[]): { templates: T[]; others: T[] } {
  const templates: T[] = [];
  const others: T[] = [];
  for (const entry of list)
    (isNameTemplate(entry.name) ? templates : others).push(entry);
  const byUsage = (a: T, b: T) =>
    (b.usage ?? 0) - (a.usage ?? 0) || a.slug.localeCompare(b.slug);
  return { templates: templates.sort(byUsage), others: others.sort(byUsage) };
}

export const WIKI_TYPE_LABELS: Record<WikiType, string> = {
  effect: "Effect",
  trigger: "Trigger",
  modifier: "Modifier",
};

export function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

export function preprocessWikiLinks(md: string): string {
  return md
    .replace(
      /\[\[(effect|trigger|modifier)-([^\]]+)\]\]/g,
      (_m, t: string, n: string) => `[${n}](/wiki/${t}/${slugify(n)})`,
    )
    .replace(/\[\[(?:scope|cat)-([^\]]+)\]\]/g, "`$1`");
}
