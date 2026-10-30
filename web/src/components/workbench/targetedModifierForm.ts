export interface TargetedModifierEntry {
  tag: string;
  modifier: Record<string, number | string>;
}

export type CustomTraitTargetedModifierType =
  "country_leader" | "unit_leader" | "scientist";

export function customTraitSupportsTargetedModifier(
  traitType: CustomTraitTargetedModifierType,
): boolean {
  return traitType !== "scientist";
}

export interface TargetedModifierKeyCollision {
  targetIndex: number;
  key: string;
}

export function findTargetedModifierKeyCollision(
  entries: TargetedModifierEntry[] | null | undefined,
): TargetedModifierKeyCollision | null {
  for (const [targetIndex, entry] of (entries ?? []).entries()) {
    const seen = new Set<string>();
    for (const rawKey of Object.keys(entry?.modifier ?? {})) {
      const key = rawKey.trim();
      if (!key) continue;
      if (seen.has(key)) return { targetIndex, key };
      seen.add(key);
    }
  }
  return null;
}

export function targetedModifierKeyConflicts(
  entry: TargetedModifierEntry,
  editedIndex: number,
  proposedKey: string,
): boolean {
  const normalized = proposedKey.trim();
  if (!normalized) return false;
  return Object.keys(entry.modifier).some(
    (key, index) => index !== editedIndex && key.trim() === normalized,
  );
}

export function targetedModifierInputValue(raw: string): number | string {
  const trimmed = raw.trim();
  if (trimmed !== "" && /^[-+]?(?:\d+\.?\d*|\.\d+)$/.test(trimmed)) {
    return Number(trimmed);
  }
  return raw;
}

export function normalizeTargetedModifiers(
  entries: TargetedModifierEntry[] | null | undefined,
): TargetedModifierEntry[] {
  return (entries ?? [])
    .map((entry) => ({
      tag: String(entry?.tag ?? "")
        .trim()
        .toUpperCase(),
      modifier: Object.fromEntries(
        Object.entries(entry?.modifier ?? {})
          .map(
            ([key, value]) =>
              [
                key.trim(),
                typeof value === "string"
                  ? targetedModifierInputValue(value)
                  : value,
              ] as const,
          )
          .filter(([key]) => key.length > 0),
      ),
    }))
    .filter((entry) => entry.tag.length > 0);
}
