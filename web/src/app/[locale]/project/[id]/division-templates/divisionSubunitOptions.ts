import type { SubUnit, VanillaSubunit } from "@/types";

export interface DivisionSubunitOption extends VanillaSubunit {
  source: "vanilla" | "project";
}

const numericStat = (value: unknown): number =>
  typeof value === "number" && Number.isFinite(value) ? value : 0;

export const isProjectSupportSubunit = (row: SubUnit): boolean =>
  row.group?.trim().toLowerCase() === "support" ||
  (row.unit_types ?? []).includes("support");

const projectOption = (row: SubUnit): DivisionSubunitOption => {
  const stats = row.stats ?? {};
  return {
    token: row.sub_unit_id,
    abbreviation: row.abbreviation ?? "",
    is_support: isProjectSupportSubunit(row),
    group: row.group ?? "",
    types: [...(row.unit_types ?? [])],
    combat_width: numericStat(stats.combat_width),
    max_strength: numericStat(stats.max_strength),
    max_organisation: numericStat(stats.max_organisation),
    soft_attack: numericStat(stats.soft_attack),
    hard_attack: numericStat(stats.hard_attack),
    defense: numericStat(stats.defense),
    breakthrough: numericStat(stats.breakthrough),
    manpower: numericStat(stats.manpower),
    default_morale: numericStat(stats.default_morale),
    supply_consumption: numericStat(stats.supply_consumption),
    source: "project",
  };
};

export function buildDivisionSubunitCatalog(
  vanilla: Record<string, VanillaSubunit>,
  projectRows: SubUnit[],
): Record<string, DivisionSubunitOption> {
  const catalog: Record<string, DivisionSubunitOption> = {};
  for (const [token, row] of Object.entries(vanilla)) {
    catalog[token] = { ...row, source: "vanilla" };
  }
  for (const row of projectRows) {
    const token = row.sub_unit_id?.trim();
    if (token) catalog[token] = projectOption(row);
  }
  return catalog;
}

export function optionsPreservingCurrentToken(
  tokens: string[],
  currentToken: string,
): string[] {
  if (!currentToken || tokens.includes(currentToken)) return tokens;
  return [currentToken, ...tokens];
}
