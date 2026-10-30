export type ScopeKind = "own" | "global" | "explicit" | "global_except";

export interface ScopedEntity {
  scope_kind?: string | null;
  country_tag?: string | null;
  shared_country_tags?: string[] | null;
}

export function matchesCountryScope(
  entity: ScopedEntity,
  activeCountry: string | null | undefined,
): boolean {
  if (!activeCountry) return true;
  const kind = entity.scope_kind ?? "own";
  switch (kind) {
    case "own":
      return !entity.country_tag || entity.country_tag === activeCountry;
    case "global":
      return true;
    case "explicit":
      return (entity.shared_country_tags ?? []).includes(activeCountry);
    case "global_except":
      return !(entity.shared_country_tags ?? []).includes(activeCountry);
    default:
      return entity.country_tag === activeCountry;
  }
}

export type ScopeSource = "own" | "explicit" | "global" | "global_except";

export function getScopeSource(
  entity: ScopedEntity,
  activeCountry: string | null | undefined,
): ScopeSource {
  if (!activeCountry) return "own";
  const kind = entity.scope_kind ?? "own";
  if (kind === "global") return "global";
  if (kind === "global_except") return "global_except";
  if (kind === "explicit") return "explicit";

  return "own";
}

export function partitionByScope<T extends ScopedEntity>(
  entities: T[],
  activeCountry: string | null | undefined,
): { own: T[]; shared: T[] } {
  const own: T[] = [];
  const shared: T[] = [];
  for (const e of entities) {
    if (getScopeSource(e, activeCountry) === "own") {
      own.push(e);
    } else {
      shared.push(e);
    }
  }
  return { own, shared };
}
