export type ModifierEntry = [string, string];

export type ModifierValueFeedback =
  | { kind: "empty" }
  | { kind: "number"; savedValue: number }
  | { kind: "text" }
  | { kind: "truncated"; savedValue: number };

export function modifierEntriesToRecord(
  entries: readonly ModifierEntry[],
): Record<string, number | string> {
  const modifier: Record<string, number | string> = {};
  for (const [rawKey, rawValue] of entries) {
    const key = rawKey.trim();
    if (!key) continue;
    const parsed = parseFloat(rawValue);
    modifier[key] = Number.isNaN(parsed) ? rawValue : parsed;
  }
  return modifier;
}

export function updateModifierEntry(
  entries: readonly ModifierEntry[],
  index: number,
  position: 0 | 1,
  value: string,
): ModifierEntry[] {
  return entries.map((entry, entryIndex) => {
    if (entryIndex !== index) return [...entry] as ModifierEntry;
    return position === 0 ? [value, entry[1]] : [entry[0], value];
  });
}

export function modifierValueFeedback(value: string): ModifierValueFeedback {
  const trimmed = value.trim();
  if (!trimmed) return { kind: "empty" };
  const parsed = parseFloat(value);
  if (Number.isNaN(parsed)) return { kind: "text" };
  return Number(trimmed) === parsed
    ? { kind: "number", savedValue: parsed }
    : { kind: "truncated", savedValue: parsed };
}

export function knownModifierOptions(
  modifiers: readonly string[] | null | undefined,
): string[] {
  if (!modifiers) return [];
  return [...new Set(modifiers.filter(Boolean))].sort((left, right) =>
    left.localeCompare(right),
  );
}

export function isUnknownModifierKey(
  value: string,
  knownModifiers: ReadonlySet<string> | null,
): boolean {
  const key = value.trim();
  return Boolean(key && knownModifiers && !knownModifiers.has(key));
}
