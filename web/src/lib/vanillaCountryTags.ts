import { useCallback, useSyncExternalStore } from "react";
import { useLocale } from "use-intl";
import type { ContentLocale } from "@/lib/store/contentLocaleStore";
import { requestGameIndexed, sessionGameData } from "@/lib/api/gameIndex";

export interface VanillaCountryTag {
  tag: string;

  name_en: string;

  names: Record<string, string>;
}

export interface VanillaCountryTags {
  tags: readonly VanillaCountryTag[];

  lookup: (tag: string | null | undefined) => VanillaCountryTag | null;
}

function countryList(tags: VanillaCountryTag[]): VanillaCountryTags {
  const byTag = new Map(tags.map((c) => [c.tag, c]));
  return {
    tags,
    lookup: (tag) => (tag ? (byTag.get(tag.toUpperCase()) ?? null) : null),
  };
}

const countries = sessionGameData(
  () =>
    requestGameIndexed<{ tags: VanillaCountryTag[] }>(
      "/go/vanilla/country-tags",
    ).then((data) => countryList(data.tags)),
  countryList([]),
);

export function useVanillaCountryTags(): VanillaCountryTags {
  return useSyncExternalStore(
    countries.subscribe,
    countries.current,
    countries.current,
  );
}

export const loadVanillaCountryTags = countries.load;

export function vanillaTagName(
  country: VanillaCountryTag,
  locale: string,
): string {
  return country.names[locale] || country.name_en;
}

export function useVanillaTagName(): (
  tag: string | null | undefined,
) => string | null {
  const { lookup } = useVanillaCountryTags();
  const locale = useLocale();
  return useCallback(
    (tag: string | null | undefined) => {
      const country = lookup(tag);
      return country ? vanillaTagName(country, locale) : null;
    },
    [lookup, locale],
  );
}

export interface ResolvedTagToken {
  raw: string;

  tag: string;

  display: string;

  flag: string;

  start: number;
  end: number;
}

const TAG_TOKEN_RE = /\[([A-Z]{2,4}):NameWithFlag\]/g;

export function parseTagTokens(
  text: string,

  lookupVanilla: VanillaCountryTags["lookup"],
  projectTagLookup?: (tag: string) => { display: string; flag: string } | null,

  locale: ContentLocale = "en",
): ResolvedTagToken[] {
  const out: ResolvedTagToken[] = [];
  if (!text) return out;
  let match: RegExpExecArray | null;
  TAG_TOKEN_RE.lastIndex = 0;
  while ((match = TAG_TOKEN_RE.exec(text)) !== null) {
    const tag = match[1].toUpperCase();
    const vanilla = lookupVanilla(tag);
    let display = tag;
    let flag = "";
    if (vanilla) {
      display = vanillaTagName(vanilla, locale);
    } else if (projectTagLookup) {
      const custom = projectTagLookup(tag);
      if (custom) {
        display = custom.display || tag;
        flag = custom.flag || "";
      }
    }
    out.push({
      raw: match[0],
      tag,
      display,
      flag,
      start: match.index,
      end: match.index + match[0].length,
    });
  }
  return out;
}

export function renderTagTokensPlain(
  text: string,
  lookupVanilla: VanillaCountryTags["lookup"],
  projectTagLookup?: (tag: string) => { display: string; flag: string } | null,
  locale: ContentLocale = "en",
): string {
  const tokens = parseTagTokens(text, lookupVanilla, projectTagLookup, locale);
  if (tokens.length === 0) return text;
  let out = "";
  let cursor = 0;
  for (const tok of tokens) {
    out += text.slice(cursor, tok.start);
    out += tok.flag ? `${tok.flag} ${tok.display}` : tok.display;
    cursor = tok.end;
  }
  out += text.slice(cursor);
  return out;
}
