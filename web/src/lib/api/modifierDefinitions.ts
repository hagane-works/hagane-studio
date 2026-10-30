import { request } from "./_client";

export const MODIFIER_COLOR_TYPES = ["good", "bad", "neutral"] as const;
export const MODIFIER_VALUE_TYPES = [
  "number",
  "percentage",
  "percentage_in_hundred",
  "yes_no",
] as const;
export const MODIFIER_POSTFIXES = ["none", "days", "hours", "daily"] as const;
export const MODIFIER_CATEGORIES = [
  "none",
  "all",
  "country",
  "state",
  "unit_leader",
  "army",
  "naval",
  "air",
  "peace",
  "politics",
  "ai",
  "defensive",
  "aggressive",
  "war_production",
  "military_advancements",
  "military_equipment",
  "autonomy",
  "government_in_exile",
  "intelligence_agency",
] as const;

export type ModifierColorType = (typeof MODIFIER_COLOR_TYPES)[number];
export type ModifierValueType = (typeof MODIFIER_VALUE_TYPES)[number];
export type ModifierPostfix = (typeof MODIFIER_POSTFIXES)[number];
export type ModifierCategory = (typeof MODIFIER_CATEGORIES)[number];

export interface ModifierDefinitionDraft {
  modifier_id: string;
  color_type: ModifierColorType;
  value_type: ModifierValueType;
  precision: number;
  postfix: ModifierPostfix;
  categories: ModifierCategory[];
}

export interface ModifierDefinition extends ModifierDefinitionDraft {
  id: number;
  project_id: number;
  source_file: string;
  explicit_fields: string[];
  source: string;
  created_at: string;
  updated_at: string;
}

export function newModifierDefinitionDraft(
  modifierId = "new_modifier",
): ModifierDefinitionDraft {
  return {
    modifier_id: modifierId,
    color_type: "bad",
    value_type: "number",
    precision: 2,
    postfix: "none",
    categories: ["all"],
  };
}

export function modifierDefinitionToDraft(
  row: ModifierDefinition,
): ModifierDefinitionDraft {
  return {
    modifier_id: row.modifier_id,
    color_type: row.color_type,
    value_type: row.value_type,
    precision: row.precision,
    postfix: row.postfix,
    categories: [...row.categories],
  };
}

export function formatModifierDefinitionScript(
  definition: ModifierDefinitionDraft,
  options?: { source?: string; explicitFields?: string[] },
): string {
  const emitAll = options?.source !== "imported";
  const explicit = new Set(options?.explicitFields ?? []);
  const lines = [`${definition.modifier_id} = {`];
  if (emitAll || explicit.has("color_type")) {
    lines.push(`\tcolor_type = ${definition.color_type}`);
  }
  if (emitAll || explicit.has("value_type")) {
    lines.push(`\tvalue_type = ${definition.value_type}`);
  }
  if (emitAll || explicit.has("precision")) {
    lines.push(`\tprecision = ${definition.precision}`);
  }
  if (emitAll || explicit.has("postfix")) {
    lines.push(`\tpostfix = ${definition.postfix}`);
  }
  if (emitAll || explicit.has("category")) {
    lines.push(
      ...definition.categories.map((category) => `\tcategory = ${category}`),
    );
  }
  lines.push("}");
  return lines.join("\n");
}

export const modifierDefinitionsApi = {
  list: (projectId: number) =>
    request<{ modifier_definitions: ModifierDefinition[]; total: number }>(
      `/go/projects/${projectId}/modifier-definitions`,
    ),
  get: (projectId: number, definitionId: number) =>
    request<ModifierDefinition>(
      `/go/projects/${projectId}/modifier-definitions/${definitionId}`,
    ),
  create: (projectId: number, data: ModifierDefinitionDraft) =>
    request<ModifierDefinition>(
      `/go/projects/${projectId}/modifier-definitions`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    ),
  update: (
    projectId: number,
    definitionId: number,
    data: Partial<ModifierDefinitionDraft>,
  ) =>
    request<ModifierDefinition>(
      `/go/projects/${projectId}/modifier-definitions/${definitionId}`,
      { method: "PUT", body: JSON.stringify(data) },
    ),
  remove: (projectId: number, definitionId: number) =>
    request<void>(
      `/go/projects/${projectId}/modifier-definitions/${definitionId}`,
      {
        method: "DELETE",
      },
    ),
};
