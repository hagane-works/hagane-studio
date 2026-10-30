import { useSyncExternalStore } from "react";
import { requestGameIndexed, sessionGameData } from "@/lib/api/gameIndex";

export interface CountryAliases {
  byTag: Readonly<Record<string, readonly string[]>>;

  tags: ReadonlySet<string>;
}

function aliasesOf(byTag: Record<string, string[]>): CountryAliases {
  return { byTag, tags: new Set(Object.keys(byTag)) };
}

const countryAliases = sessionGameData(
  () =>
    requestGameIndexed<{ aliases: Record<string, string[]> }>(
      "/go/vanilla/country-aliases",
    ).then((data) => aliasesOf(data.aliases ?? {})),
  aliasesOf({}),
);

export function useCountryAliases(): CountryAliases {
  return useSyncExternalStore(
    countryAliases.subscribe,
    countryAliases.current,
    countryAliases.current,
  );
}

export const loadCountryAliases = countryAliases.load;

export function getCountryAliases(tag: string): readonly string[] {
  return countryAliases.current().byTag[tag.toUpperCase()] ?? [];
}
