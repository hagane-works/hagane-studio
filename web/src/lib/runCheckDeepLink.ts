import { matchesCountryScope, type ScopedEntity } from "@/lib/utils/scope";

export interface RunCheckEntityDeepLink {
  containerId: number;
  entityId: number;
}

interface SearchParamReader {
  get(name: string): string | null;
}

export function parseRunCheckEntityDeepLink(
  searchParams: SearchParamReader,
  containerParam: string,
  entityParam: string,
): RunCheckEntityDeepLink | null {
  const containerId = parseDatabaseId(searchParams.get(containerParam));
  const entityId = parseDatabaseId(searchParams.get(entityParam));
  if (containerId === null || entityId === null) return null;
  return { containerId, entityId };
}

function parseDatabaseId(raw: string | null): number | null {
  if (!raw || !/^[1-9]\d*$/.test(raw)) return null;
  const value = Number(raw);
  return Number.isSafeInteger(value) ? value : null;
}

export type DeepLinkScopeReveal =
  | { kind: "unchanged" }
  | { kind: "country"; countryTag: string }
  | { kind: "all" };

export function resolveDeepLinkScopeReveal(
  entityOrEntities: ScopedEntity | readonly ScopedEntity[],
  currentCountry: string | null,
  projectCountries: readonly string[],
): DeepLinkScopeReveal {
  const entities: readonly ScopedEntity[] = Array.isArray(entityOrEntities)
    ? entityOrEntities
    : [entityOrEntities as ScopedEntity];
  if (
    !currentCountry ||
    entities.every((entity) => matchesCountryScope(entity, currentCountry))
  ) {
    return { kind: "unchanged" };
  }

  const candidate = projectCountries.find((countryTag) =>
    entities.every((entity) => matchesCountryScope(entity, countryTag)),
  );
  if (candidate) {
    return { kind: "country", countryTag: candidate };
  }
  return { kind: "all" };
}
