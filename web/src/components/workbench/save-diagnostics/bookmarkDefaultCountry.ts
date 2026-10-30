import type { BookmarkSelectableCountry } from "@/lib/api/scenario";

export function includeDefaultCountry(
  countries: BookmarkSelectableCountry[],
  tag: string,
): BookmarkSelectableCountry[] {
  const normalized = tag.trim().toUpperCase();
  if (
    !/^[A-Z][A-Z0-9]{2}$/.test(normalized) ||
    countries.some((row) => row.tag.trim().toUpperCase() === normalized)
  )
    return countries;
  return [
    ...countries,
    {
      tag: normalized,
      history_en: "",
      history_zh: "",
      history_locs: {},
      minor: false,
    },
  ];
}
