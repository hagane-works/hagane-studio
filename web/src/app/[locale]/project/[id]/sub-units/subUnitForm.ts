import type { SubUnit } from "@/types";

export type TriState = "unset" | "yes" | "no";

export const SUB_UNIT_BOOLEAN_FIELDS = [
  "active",
  "cavalry",
  "special_forces",
  "marines",
  "can_exfiltrate_from_coast",
  "mountaineers",
  "can_be_parachuted",
  "affects_speed",
] as const;

export const SUB_UNIT_INTERNAL_TYPES = [
  "infantry",
  "support",
  "artillery",
  "anti_tank",
  "anti_air",
  "armor",
  "fighter",
  "cas",
  "naval_bomber",
  "interceptor",
  "suicide",
  "tactical_bomber",
  "strategic_bomber",
  "air_transport",
  "missile",
  "submarine",
  "screen_ship",
  "capital_ship",
  "carrier",
  "motorized",
  "mechanized",
  "flame",
  "amphibious",
] as const;

const INTERNAL_TYPE_SET = new Set<string>(SUB_UNIT_INTERNAL_TYPES);
const BOOLEAN_FIELD_SET = new Set<string>(SUB_UNIT_BOOLEAN_FIELDS);
const CATEGORY_TOKEN_RE = /^[A-Za-z0-9_@.$:-]+$/;

export type SubUnitBooleanKey = (typeof SUB_UNIT_BOOLEAN_FIELDS)[number];
export type SubUnitJsonKey =
  "need" | "stats" | "terrain_modifiers" | "raw_extras";
export type SubUnitDraftIssue =
  | { field: "unit_types"; kind: "booleanField" | "unsupported"; token: string }
  | { field: "categories"; kind: "invalidCategory"; token: string };

export interface SubUnitDraft {
  sub_unit_id: string;
  country_tag: string;
  source_file: string;
  ordering: string;
  abbreviation: string;
  sprite: string;
  map_icon_category: string;
  priority: string;
  ai_priority: string;
  transport: string;
  group: string;
  name_en: string;
  name_zh: string;
  display_name_locs: Record<string, string>;
  booleans: Record<SubUnitBooleanKey, TriState>;
  unit_types: string;
  categories: string;
  essential: string;
  json: Record<SubUnitJsonKey, string>;
}

const triStateFromValue = (value: boolean | null): TriState =>
  value === null || value === undefined ? "unset" : value ? "yes" : "no";

const valueFromTriState = (value: TriState): boolean | null =>
  value === "unset" ? null : value === "yes";

const numberString = (value: number | null | undefined): string =>
  value === null || value === undefined ? "" : String(value);

const nullableInteger = (value: string): number | null =>
  value.trim() === "" ? null : Math.trunc(Number(value));

const splitTokens = (value: string): string[] =>
  value
    .split(/[\s,]+/)
    .map((token) => token.trim())
    .filter(Boolean);

export function getSubUnitDraftIssue(
  draft: SubUnitDraft,
): SubUnitDraftIssue | null {
  for (const token of splitTokens(draft.unit_types)) {
    if (BOOLEAN_FIELD_SET.has(token)) {
      return { field: "unit_types", kind: "booleanField", token };
    }
    if (!INTERNAL_TYPE_SET.has(token)) {
      return { field: "unit_types", kind: "unsupported", token };
    }
  }
  for (const token of splitTokens(draft.categories)) {
    if (!CATEGORY_TOKEN_RE.test(token)) {
      return { field: "categories", kind: "invalidCategory", token };
    }
  }
  return null;
}

const stableJson = (value: Record<string, unknown>): string =>
  JSON.stringify(value ?? {}, null, 2);

function parseJsonObject(
  value: string,
  field: SubUnitJsonKey,
): Record<string, unknown> {
  if (!value.trim()) return {};
  const parsed: unknown = JSON.parse(value);
  if (parsed === null || Array.isArray(parsed) || typeof parsed !== "object") {
    throw new Error(`${field} must be a JSON object`);
  }
  return parsed as Record<string, unknown>;
}

export function subUnitToDraft(row: SubUnit): SubUnitDraft {
  const booleans = {} as Record<SubUnitBooleanKey, TriState>;
  for (const field of SUB_UNIT_BOOLEAN_FIELDS) {
    booleans[field] = triStateFromValue(row[field]);
  }
  return {
    sub_unit_id: row.sub_unit_id ?? "",
    country_tag: row.country_tag ?? "",
    source_file: row.source_file ?? "",
    ordering: String(row.ordering ?? 0),
    abbreviation: row.abbreviation ?? "",
    sprite: row.sprite ?? "",
    map_icon_category: row.map_icon_category ?? "",
    priority: numberString(row.priority),
    ai_priority: numberString(row.ai_priority),
    transport: row.transport ?? "",
    group: row.group ?? "",
    name_en: row.name_en ?? "",
    name_zh: row.name_zh ?? "",
    display_name_locs: { ...(row.display_name_locs ?? {}) },
    booleans,
    unit_types: (row.unit_types ?? []).join(" "),
    categories: (row.categories ?? []).join(" "),
    essential: (row.essential ?? []).join(" "),
    json: {
      need: stableJson(row.need),
      stats: stableJson(row.stats),
      terrain_modifiers: stableJson(row.terrain_modifiers),
      raw_extras: stableJson(row.raw_extras),
    },
  };
}

export function emptySubUnitDraft(): SubUnitDraft {
  return subUnitToDraft({
    id: 0,
    project_id: 0,
    sub_unit_id: "new_sub_unit",
    country_tag: "",
    source_file: "",
    source: "created",
    ordering: 0,
    name: "",
    name_en: "",
    name_zh: "",
    desc_en: "",
    desc_zh: "",
    display_name_locs: {},
    description_locs: {},
    abbreviation: "",
    sprite: "",
    map_icon_category: "",
    priority: null,
    ai_priority: null,
    transport: "",
    group: "",
    active: null,
    cavalry: null,
    special_forces: null,
    marines: null,
    can_exfiltrate_from_coast: null,
    mountaineers: null,
    can_be_parachuted: null,
    affects_speed: null,
    unit_types: [],
    categories: [],
    essential: [],
    need: {},
    stats: {},
    terrain_modifiers: {},
    raw_extras: {},
  });
}

export function subUnitDraftToPayload(draft: SubUnitDraft): Partial<SubUnit> {
  const issue = getSubUnitDraftIssue(draft);
  if (issue?.kind === "booleanField") {
    throw new Error(
      `unit_types contains boolean field '${issue.token}'; set ${issue.token}=yes instead`,
    );
  }
  if (issue?.kind === "unsupported") {
    throw new Error(`unsupported sub-unit internal type '${issue.token}'`);
  }
  if (issue) {
    throw new Error(`invalid sub-unit category token '${issue.token}'`);
  }
  const payload: Record<string, unknown> = {
    sub_unit_id: draft.sub_unit_id.trim(),
    country_tag: draft.country_tag.trim().toUpperCase(),
    source_file: draft.source_file.trim().replaceAll("\\", "/"),
    ordering:
      draft.ordering.trim() === "" ? 0 : Math.trunc(Number(draft.ordering)),
    abbreviation: draft.abbreviation,
    sprite: draft.sprite.trim(),
    map_icon_category: draft.map_icon_category.trim(),
    priority: nullableInteger(draft.priority),
    ai_priority: nullableInteger(draft.ai_priority),
    transport: draft.transport.trim(),
    group: draft.group.trim(),
    name_en: draft.name_en,
    name_zh: draft.name_zh,
    display_name_locs: { ...draft.display_name_locs },
    unit_types: splitTokens(draft.unit_types),
    categories: splitTokens(draft.categories),
    essential: splitTokens(draft.essential),
  };
  for (const field of SUB_UNIT_BOOLEAN_FIELDS) {
    payload[field] = valueFromTriState(draft.booleans[field]);
  }
  for (const field of [
    "need",
    "stats",
    "terrain_modifiers",
    "raw_extras",
  ] as const) {
    payload[field] = parseJsonObject(draft.json[field], field);
  }
  return payload as Partial<SubUnit>;
}

export function buildSubUnitUpdate(
  row: SubUnit,
  draft: SubUnitDraft,
): Partial<SubUnit> {
  const original = subUnitDraftToPayload(subUnitToDraft(row));
  const candidate = subUnitDraftToPayload(draft);
  const update: Record<string, unknown> = {};
  for (const [field, value] of Object.entries(candidate)) {
    if (
      JSON.stringify(value) !==
      JSON.stringify(original[field as keyof typeof original])
    ) {
      update[field] = value;
    }
  }
  return update as Partial<SubUnit>;
}
