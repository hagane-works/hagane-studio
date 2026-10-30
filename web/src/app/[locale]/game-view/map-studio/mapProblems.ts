import type { MapProblem } from "@/lib/mapToolRequest";

export const MAP_PROBLEM_CODES = [
  "busy",
  "unreadable",
  "not_saved",
  "bad_request",
  "too_large",
  "nothing",
  "own_map",
  "no_country",
  "state_drowned",
  "state_missing",
  "sea_divisions",
  "sea_capital",
  "sea_unreached",
  "sea_no_region",
  "region_empty",
  "sea_region_state",
  "sea_region_alone",
  "land_breaks_map",
  "world_breaks_map",
  "undo_failed",
  "nothing_to_undo",
  "map_changed",
  "map_changes_moved",
  "imported_states",
  "no_province",
  "already_water",
  "water_terrain",
  "river_on_water",
  "river_faults",
  "land_off_sea",
  "land_outside_map",
  "land_covers_sea",

  "upload_missing",
  "upload_unreadable",
  "upload_jpeg",
  "sketch_unreadable",
  "world_land_size",
  "world_per_state",
  "tag_invalid",
  "world_no_countries",
  "world_too_many_countries",
  "owners_not_png",
  "owners_too_large",
  "owners_extra_country",
  "world_country_unpainted",
  "world_size",
  "world_size_range",
  "world_no_land",
  "world_few_states",
  "upload_pictures_missing",
  "upload_size_mismatch",
  "upload_definitions_empty",
  "mod_state_gap",
  "mod_state_no_land",
  "mod_state_capital",
  "mod_needs_states_regions",
  "mod_needs_definition",
  "mod_file_outside",
  "mod_too_many_files",
  "mod_file_unreadable",
  "mod_text_too_large",
  "mod_unknown_tags",
  "kind_land_water",
  "kind_invalid",
  "lake_strands_coast",
  "terrain_unknown",
  "water_continent",
  "continent_range",
  "state_land_only",
  "state_last_province",
  "state_last_land",
  "region_missing",
  "region_last_province",
  "region_file_text",
  "province_state_region",
  "province_sea_split",
] as const;

const KNOWN = new Set<string>(MAP_PROBLEM_CODES);

export class MapProblemError extends Error {
  constructor(readonly problem: MapProblem) {
    super(problem.detail);
    this.name = "MapProblemError";
  }
}

type Translate = (
  key: string,
  values?: Record<string, string | number>,
) => string;

export function mapProblemMessage(
  problem: MapProblem,
  t: Translate,
  locale: string,
): string {
  if (problem.code && KNOWN.has(problem.code))
    return t(`problem.${problem.code}`, problem.values ?? {});
  if (locale.toLowerCase().startsWith("zh") && problem.detail)
    return problem.detail;
  return t("problem.other");
}

export function problemProvinceIndexes(
  provinces: readonly { id: number }[],
  ids: readonly number[],
): number[] {
  if (!ids.length) return [];
  const wanted = new Set(ids);
  const out: number[] = [];
  for (let i = 1; i < provinces.length; i++)
    if (wanted.has(provinces[i].id)) out.push(i);
  return out;
}

export const MAP_FINDING_CODES = [
  "size_mismatch",
  "duplicate_color",
  "duplicate_id",
  "too_many_provinces",
  "black_pixel",
  "undefined_color",
  "unused_definition",
  "disconnected_province",
  "small_province",
  "province_box",
  "x_crossing",
  "land_under_water",
  "water_above_land",
  "terrain_index",

  "province_no_state",
  "province_no_region",
  "state_ids_gap",
  "state_two_regions",
  "sea_region_split",

  "state_no_land",
] as const;

const FINDINGS = new Set<string>(MAP_FINDING_CODES);

export function mapFindingMessage(
  finding: { code: string; message: string },
  t: Translate,
  locale: string,
): string {
  if (FINDINGS.has(finding.code)) return t(`finding.${finding.code}`);
  if (locale.toLowerCase().startsWith("zh") && finding.message)
    return finding.message;
  return t("finding.other");
}

export const MAP_FINDING_FIX_CODES = [
  "size_mismatch",
  "duplicate_color",
  "duplicate_id",
  "black_pixel",
  "undefined_color",
  "province_box",
  "province_no_state",
  "province_no_region",
  "state_ids_gap",
  "state_two_regions",
  "state_no_land",
  "sea_region_split",
] as const;
const FINDING_FIXES = new Set<string>(MAP_FINDING_FIX_CODES);

export function mapFindingFix(code: string, t: Translate): string | null {
  return FINDING_FIXES.has(code) ? t(`findingFix.${code}`) : null;
}

export const ONE_CLICK_FIXES = new Set<string>([
  "black_pixel",
  "undefined_color",
  "state_no_land",
]);
