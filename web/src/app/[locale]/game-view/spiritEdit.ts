import type { DynamicModifier, Idea } from "@/types";

export type SpiritRow =
  { kind: "idea"; row: Idea } | { kind: "dynamic"; row: DynamicModifier };

export interface ModifierLine {
  key: string;
  value: string;
}

function modifierValue(text: string): number | string {
  const trimmed = text.trim();
  return trimmed !== "" && /^-?\d+(\.\d+)?$/.test(trimmed)
    ? Number(trimmed)
    : trimmed;
}

export function modifierMap(
  lines: readonly ModifierLine[],
): Record<string, number | string> {
  const out: Record<string, number | string> = {};
  for (const line of lines) {
    const key = line.key.trim();
    if (key) out[key] = modifierValue(line.value);
  }
  return out;
}

export function spiritName(spirit: SpiritRow, shownName: string): string {
  return spirit.row.display_name_locs?.en || spirit.row.name_en || shownName;
}
