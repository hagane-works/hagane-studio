import type { WargoalType, WargoalWrite } from "@/types/wargoal";

export const WARGOAL_SCALAR_KEYS = [
  "generate_base_cost",
  "generate_per_state_cost",
  "take_states_limit",
  "take_states_cost",
  "puppet_cost",
  "force_government_cost",
  "expire",
  "threat",
  "take_states_threat_factor",
] as const;

export const WARGOAL_BLOCK_KEYS = [
  "allowed",
  "available",
  "take_states",
  "puppet",
  "liberate",
] as const;

export interface WargoalDraft extends WargoalWrite {
  id: number | null;
}
export function freshWargoalDraft(existingIds: string[]): WargoalDraft {
  let suffix = 1;
  let key = "my_wargoal";
  while (existingIds.includes(key)) {
    suffix += 1;
    key = `my_wargoal_${suffix}`;
  }
  return {
    id: null,
    wargoal_key: key,
    war_name: "",
    safe_scalars: {},
    allowed: "",
    available: "",
    take_states: "",
    puppet: "",
    liberate: "",
    advanced_raw_script: "",
    source_file: "",
  };
}

export function toWargoalDraft(row: WargoalType): WargoalDraft {
  return {
    id: row.id,
    wargoal_key: row.wargoal_key,
    war_name: row.war_name ?? "",
    safe_scalars: { ...(row.safe_scalars ?? {}) },
    allowed: row.allowed ?? "",
    available: row.available ?? "",
    take_states: row.take_states ?? "",
    puppet: row.puppet ?? "",
    liberate: row.liberate ?? "",
    advanced_raw_script: row.advanced_raw_script ?? "",
    source_file: row.source_file ?? "",
  };
}

export function toWargoalWrite(draft: WargoalDraft): WargoalWrite {
  const safeScalars = Object.fromEntries(
    Object.entries(draft.safe_scalars)
      .map(([key, value]) => [key, value.trim()])
      .filter(([, value]) => value.length > 0),
  );
  return {
    wargoal_key: draft.wargoal_key.trim(),
    war_name: draft.war_name.trim(),
    safe_scalars: safeScalars,
    allowed: draft.allowed.trim(),
    available: draft.available.trim(),
    take_states: draft.take_states.trim(),
    puppet: draft.puppet.trim(),
    liberate: draft.liberate.trim(),
    advanced_raw_script: draft.advanced_raw_script.trim(),
    source_file: draft.source_file.trim().replaceAll("\\", "/"),
  };
}

function indent(text: string, level: number): string {
  const prefix = "\t".repeat(level);
  return text
    .split("\n")
    .map((line) => (line ? `${prefix}${line}` : line))
    .join("\n");
}

export function buildWargoalPreview(draft: WargoalDraft): string {
  const lines: string[] = [];
  if (draft.war_name.trim()) lines.push(`war_name = ${draft.war_name.trim()}`);
  for (const key of WARGOAL_BLOCK_KEYS) {
    const body = draft[key].trim();
    if (body) lines.push(`${key} = {\n${indent(body, 1)}\n}`);
  }
  for (const key of WARGOAL_SCALAR_KEYS) {
    const value = draft.safe_scalars[key]?.trim();
    if (value) lines.push(`${key} = ${value}`);
  }
  if (draft.advanced_raw_script.trim())
    lines.push(draft.advanced_raw_script.trim());
  const body = lines.length ? `\n${indent(lines.join("\n"), 2)}\n\t` : "\n\t";
  return `wargoal_types = {\n\t${draft.wargoal_key || "my_wargoal"} = {${body}}\n}\n`;
}
