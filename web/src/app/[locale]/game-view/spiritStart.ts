import type { Idea } from "@/types";
import { slugOf } from "./personCreate";

export function spiritToken(tag: string, name: string): string {
  return `${tag.toUpperCase()}_${slugOf(name) || "new_spirit"}`;
}

export function withStartSpirit(
  extra: readonly string[],
  token: string,
): string[] {
  return extra.includes(token) ? [...extra] : [...extra, token];
}

export function withoutStartSpirit(
  extra: readonly string[],
  token: string,
): string[] {
  return extra.filter((item) => item !== token);
}

function spiritFor(idea: Idea, tag: string): boolean {
  if (idea.idea_type !== "country") return false;
  const own = (idea.country_tag ?? "").toUpperCase();
  return (
    !own ||
    own === tag.toUpperCase() ||
    (idea.shared_country_tags ?? []).some(
      (t) => t.toUpperCase() === tag.toUpperCase(),
    )
  );
}

export function addableSpirits(
  ideas: readonly Idea[],
  tag: string,
  started: readonly string[],
  query: string,
  limit = 30,
): Idea[] {
  const want = query.trim().toLowerCase();
  const taken = new Set(started);
  return ideas
    .filter((idea) => spiritFor(idea, tag) && !taken.has(idea.idea_id))
    .filter((idea) => {
      if (!want) return true;
      const name = (
        idea.display_name_locs?.en ||
        idea.name_en ||
        idea.name ||
        ""
      ).toLowerCase();
      return name.includes(want) || idea.idea_id.toLowerCase().includes(want);
    })
    .slice(0, limit);
}

export function newSpiritRow(tag: string, name: string): Partial<Idea> {
  const text = name.trim();
  return {
    idea_id: spiritToken(tag, text),
    idea_type: "country",
    country_tag: tag.toUpperCase(),
    name: text,
    name_en: text,
    display_name_locs: { en: text },
    modifier: {},
    auto_attach_on_start: false,
  };
}

export function staysOnStartup(idea: Idea | undefined, tag: string): boolean {
  if (
    !idea ||
    idea.idea_type !== "country" ||
    idea.auto_attach_on_start === false
  )
    return false;
  return (idea.country_tag ?? "").toUpperCase() === tag.toUpperCase();
}
